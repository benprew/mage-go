package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONUnlimitedLandPlayRealRules(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Fastbond")
			factory := staticBaseSubrule(t, cloneRepositoryNode(record), record.Fields["Rules"].Args[0])
			g, a, b := ronGame()
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			for i := range 4 {
				land := mage.NewLand("Land")
				a.AddToHand(land)
				if len(g.GetPlayableLands(a.PlayerID())) != 1 || g.GetLandsPlayedThisTurn() != i {
					t.Fatal("wrong play query or history")
				}
				if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
			}
			if a.Life() != 20 {
				t.Fatal("permission clause changed life")
			}
			g.SetActivePlayerIndex(1)
			for i := range 2 {
				land := mage.NewLand("Opponent land")
				b.AddToHand(land)
				g.SetLandsPlayedThisTurn(i)
				if err := g.PlayLand(b.PlayerID(), land.ID()); (err == nil) != (i == 0) {
					t.Fatal("permission escaped controller", err)
				}
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "WhenAPlayerPlaysALand") {
				t.Fatalf("wrong full-record boundary: %v", err)
			}
		})
	}
}

func TestRONUnlimitedLandPlayInvalid(t *testing.T) {
	for _, rule := range []string{
		`PlayerEffect(You(),[MayPlayAnyNumberOfLandsDuringThierTurn])`,
		`PlayerEffect(Opponent,[MayPlayAnyNumberOfLandsDuringThierTurn])`,
		`PlayerEffect(ActionPlayer,[MayPlayAnyNumberOfLandsDuringThierTurn])`,
		`PlayerEffect(You,[MayPlayAnyNumberOfLandsDuringThierTurn()])`,
		`PlayerEffect(You,[MayPlayAnyNumberOfLandsDuringThierTurn(Integer(1))])`,
		`PlayerEffect(You,[MayPlayAnyNumberOfLandsDuringThierTurn,Unknown])`,
		`EachPlayerEffect(AnyPlayer,[MayPlayAnyNumberOfLandsDuringThierTurn])`,
		`Activated(PayMana([]),ActionList([MayPlayAnyNumberOfLandsDuringThierTurn]))`,
		`SpellActions(ActionList([PlayerEffect(You,[MayPlayAnyNumberOfLandsDuringThierTurn])]))`,
	} {
		t.Run(rule, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid permission accepted")
			}
		})
	}
}

func TestRONUnlimitedLandPlayAtomicLoadAndInventory(t *testing.T) {
	source := `Card(Name:"Permission",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[PlayerEffect(You,[MayPlayAnyNumberOfLandsDuringThierTurn])])`
	files := fstest.MapFS{"good.ron": {Data: []byte(source)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v, %+v", err, report)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "MayPlayAnyNumberOfLandsDuringThierTurn" && sig.Role == "player_rule_operation" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing permission inventory role")
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "Name:\"Permission\"", "Name:\"Bad\"", 1))}
	files["bad.ron"].Data = []byte(strings.Replace(string(files["bad.ron"].Data), "You,", "Opponent,", 1))
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}
