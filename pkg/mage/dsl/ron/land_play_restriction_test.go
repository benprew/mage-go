package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONLandPlayRestrictionRealRules(t *testing.T) {
	for _, name := range []string{"Worms of the Earth", "City in a Bottle"} {
		set := "DRK"
		if name == "City in a Bottle" {
			set = "ARN"
		}
		record := staticBaseRecord(t, set, name)
		rules := record.Fields["Rules"].Args
		rule := rules[0]
		if name == "City in a Bottle" {
			rule = rules[1]
			rule.Args[1].Args = rule.Args[1].Args[1:]
		}
		factory := staticBaseSubrule(t, record, rule)
		for _, active := range []int{0, 1} {
			t.Run(name+string(rune('A'+active)), func(t *testing.T) {
				g, a, _ := ronGame()
				g.SetActivePlayerIndex(active)
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				player := g.ActivePlayerObj()
				land := mage.NewLand("Desert")
				player.AddToHand(land)
				if err := g.PlayLand(player.PlayerID(), land.ID()); err == nil {
					t.Fatal("restricted reprint played")
				}
				if len(player.Hand()) != 1 || g.GetLandsPlayedThisTurn() != 0 || len(g.GetPlayableLands(player.PlayerID())) != 0 {
					t.Fatal("failed play or query changed state")
				}
				mountain := repositoryCard(t, "ARN", "Mountain")
				player.AddToHand(mountain)
				err := g.PlayLand(player.PlayerID(), mountain.ID())
				if (err != nil) != (name == "Worms of the Earth") {
					t.Fatalf("Mountain error: %v", err)
				}
				spell := mage.NewInstant("Aladdin's Lamp", "")
				player.AddToHand(spell)
				if err := g.CastSpellByID(player.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal("land restriction blocked spell", err)
				}
				g.ResolveStack()
				if g.PutOnBattlefield(repositoryCard(t, "ARN", "Desert"), player.PlayerID()) == nil {
					t.Fatal("land restriction blocked entry")
				}
				g.GrantExtraLandPlay(player.PlayerID(), 1)
				g.PhaseOut(source.ID())
				if err := g.PlayLand(player.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
			})
		}
		if _, err := Compile(staticBaseRecord(t, set, name)); err == nil {
			t.Fatal("incomplete card accepted")
		}
	}
}

func TestRONLandPlayRestrictionInvalid(t *testing.T) {
	for _, rule := range []string{
		`EachPlayerEffect(AnyPlayer(),[CantPlayLands])`,
		`EachPlayerEffect(You,[CantPlayLands])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLands()])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLands(Integer(1))])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLandsOfType])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLandsOfType()])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLandsOfType(SharesANameOriginallyPrintedInArabianNights())])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLandsOfType(SinglePermanent(ThisPermanent))])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLandsOfType(IsCardtype(Land))])`,
		`EachPlayerEffect(AnyPlayer,[CantPlayLands,Unknown])`,
		`PlayerEffect(You,[CantPlayLands])`,
		`Activated(PayMana([]),ActionList([CantPlayLands]))`,
	} {
		t.Run(rule, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid rule accepted")
			}
		})
	}
}

func TestRONLandPlayRestrictionInventoryAndAtomicLoad(t *testing.T) {
	source := `Card(Name:"Restriction",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[EachPlayerEffect(AnyPlayer,[CantPlayLandsOfType(SharesANameOriginallyPrintedInArabianNights)])])`
	nodes, err := Parse(strings.NewReader(source), "valid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "SharesANameOriginallyPrintedInArabianNights" && sig.Role == "land_play_card_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing card-filter role")
	}
	registry := mage.NewRegistry()
	bad := strings.Replace(source, "SharesANameOriginallyPrintedInArabianNights", "ThisPermanent", 1)
	if err := LoadInto(registry, fstest.MapFS{"good.ron": {Data: []byte(source)}, "bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 0 {
		t.Fatal("partial registry published")
	}
}

func TestRONLandPlayRestrictionCatalogAndInstances(t *testing.T) {
	rule := `EachPlayerEffect(AnyPlayer,[CantPlayLandsOfType(SharesANameOriginallyPrintedInAntiquities)])`
	nodes, err := Parse(strings.NewReader(`Card(Name:"Restriction",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "catalog.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	first, second := factory(), factory()
	if first.ID() == second.ID() {
		t.Fatal("factory reused a card instance")
	}
	source := g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	land := mage.NewLand("Mishra's Factory")
	a.AddToHand(land)
	if err := g.PlayLand(a.PlayerID(), land.ID()); err == nil {
		t.Fatal("ATQ-name land played")
	}
	g.RemoveFromBattlefield(source)
	if !g.IsLandPlayProhibited(land) {
		t.Fatal("one source removed another source's rule")
	}
	if g.IsLandPlayProhibited(mage.NewLand("Desert")) {
		t.Fatal("catalogs shared state")
	}
	other, player, _ := ronGame()
	player.AddToHand(mage.NewLand("Mishra's Factory"))
	if err := other.PlayLand(player.PlayerID(), player.Hand()[0].ID()); err != nil {
		t.Fatal("factory leaked state into another game", err)
	}
}
