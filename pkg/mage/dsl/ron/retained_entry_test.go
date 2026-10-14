package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const retainedEntryRule = `ReplaceWouldPutAPermanentOnTheBattlefield(APlayerWouldPutAPermanentOnTheBattlefield(And([Opponent,ControlsNum(GreaterThanOrEqualTo(TheNumberOfPermanentsOnTheBattlefield(And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))]))),IsCardtype(Land))]),IsCardtype(Land)),[WouldPutPermanentOnBattlefield_PutPermanentOnBattlefield,PlayerAction(WouldPutAPermanentOnBattlefield_ThatPlayer,SacrificeAPermanent(IsCardtype(Land)))])`

func TestRONRetainedEntryLandEquilibrium(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Land Equilibrium")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast rollback")
	}
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	entrant := g.PutOnBattlefield(mage.NewLand("Entrant"), b.PlayerID())
	if g.FindPermanent(entrant.ID()) != nil || len(b.Graveyard()) != 1 {
		t.Fatal("zero land threshold")
	}
	g.PutOnBattlefield(mage.NewLand("Own"), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewLand("First"), b.PlayerID())
	if g.FindPermanent(first.ID()) == nil {
		t.Fatal("count included proposed land")
	}
	second := g.PutOnBattlefield(mage.NewLand("Second"), b.PlayerID())
	if len(b.Graveyard()) != 2 || len(g.FilterBattlefield(mage.And(mage.IsLand, mage.ControlledBy(b.PlayerID())))) != 1 {
		t.Fatal("equal threshold", second.ID())
	}
	if p := g.PutOnBattlefield(mage.NewArtifact("Other", ""), b.PlayerID()); g.FindPermanent(p.ID()) == nil {
		t.Fatal("nonland")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(mage.NewLand("Absent"), b.PlayerID())
	clone.PutOnBattlefield(mage.NewLand("Independent"), b.PlayerID())
	if len(g.FilterBattlefield(mage.And(mage.IsLand, mage.ControlledBy(b.PlayerID())))) != 2 || len(clone.FilterBattlefield(mage.And(mage.IsLand, mage.ControlledBy(b.PlayerID())))) != 1 {
		t.Fatal("clone or source lifecycle")
	}
}

func TestRONRetainedEntryInvalid(t *testing.T) {
	for _, bad := range []string{
		strings.Replace(retainedEntryRule, "Opponent", "Trigger_ThatPlayer", 1),
		strings.Replace(retainedEntryRule, "SinglePlayer(You)", "SinglePlayer(Ref_TargetPlayer)", 1),
		strings.Replace(retainedEntryRule, "GreaterThanOrEqualTo", "LessThan", 1),
		strings.Replace(retainedEntryRule, "WouldPutAPermanentOnBattlefield_ThatPlayer", "You", 1),
		strings.Replace(retainedEntryRule, "WouldPutPermanentOnBattlefield_PutPermanentOnBattlefield,", "", 1),
		strings.Replace(retainedEntryRule, "SacrificeAPermanent(IsCardtype(Land))", "SacrificePermanent(ThisPermanent)", 1),
		strings.Replace(retainedEntryRule, "And([Opponent,", "Or([Opponent,", 1),
		`SpellActions(ActionList([` + retainedEntryRule + `]))`,
	} {
		records, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+bad+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(records[0]); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}

func TestRONRetainedEntryCopiedSpell(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Land Equilibrium")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy")
	}
	g.ResolveStack()
	if len(g.FilterBattlefield(mage.IsEnchantment)) != 2 {
		t.Fatal("independent permanents")
	}
	for _, player := range []mage.Player{a, b} {
		p := g.PutOnBattlefield(mage.NewLand("Land"), player.PlayerID())
		if g.FindPermanent(p.ID()) != nil || len(player.Graveyard()) != 1 {
			t.Fatal("copy retained original controller")
		}
	}
}

func TestRONRetainedEntryFactoryAndAtomicLoad(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Land Equilibrium", "equilibrium.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("shared factory identity")
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	fs := fstest.MapFS{"good.ron": {Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{
		"APlayerWouldPutAPermanentOnTheBattlefield": "retained_entry_event",
		"ControlsNum": "retained_entry_player_filter",
		"WouldPutAPermanentOnBattlefield_ThatPlayer": "retained_entry_actor",
		"SacrificeAPermanent":                        "retained_entry_sacrifice",
	} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing role", name, role)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fs, "*.ron"); err != nil {
		t.Fatal(err)
	}
	fs["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + strings.Replace(retainedEntryRule, "Opponent", "Trigger_ThatPlayer", 1) + `])`)}
	if err := LoadInto(registry, fs, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Land Equilibrium"); err != nil {
		t.Fatal("failed load erased registry", err)
	}
}
