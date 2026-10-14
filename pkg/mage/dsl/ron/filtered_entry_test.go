package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const filteredEntryRule = `ReplaceWouldEnter(APermanentWouldEnterTheBattlefield(And([Or([IsCardtype(Artifact),IsCardtype(Creature),IsCardtype(Land)]),ControlledByAPlayer(Opponent)])),[EntersTapped])`

func TestRONFilteredEntryKismet(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Kismet")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil {
		t.Fatal("unpaid cast")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed cast changed zones")
	}
	a.ManaPool().Add(core.Colorless, 3)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for _, tc := range []struct {
		card   mage.Card
		tapped bool
	}{
		{mage.NewCreature("Creature", "", 2, 2), true},
		{mage.NewArtifact("Artifact", ""), true},
		{mage.NewLand("Land"), true},
		{mage.NewEnchantment("Other", ""), false},
	} {
		if p := g.PutOnBattlefield(tc.card, b.PlayerID()); p.Tapped != tc.tapped {
			t.Fatal(p.Name(), p.Tapped)
		}
	}
	if g.PutOnBattlefield(mage.NewLand("Own"), a.PlayerID()).Tapped {
		t.Fatal("own land")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if g.PutOnBattlefield(mage.NewLand("Absent"), b.PlayerID()).Tapped || !clone.PutOnBattlefield(mage.NewLand("Independent"), b.PlayerID()).Tapped {
		t.Fatal("source or clone lifecycle")
	}
}

func TestRONFilteredEntryInvalid(t *testing.T) {
	for _, bad := range []string{
		strings.Replace(filteredEntryRule, "[EntersTapped]", "[]", 1),
		strings.Replace(filteredEntryRule, "[EntersTapped]", "[EntersTapped()]", 1),
		strings.Replace(filteredEntryRule, "[EntersTapped]", "[EntersTapped,EntersTapped]", 1),
		strings.Replace(filteredEntryRule, "Opponent", "Opponent()", 1),
		strings.Replace(filteredEntryRule, "Opponent", "Trigger_ThatPlayer", 1),
		strings.Replace(filteredEntryRule, "APermanentWouldEnterTheBattlefield", "SinglePermanent", 1),
		strings.Replace(filteredEntryRule, "IsCardtype(Artifact)", "SinglePermanent(ThisPermanent)", 1),
		strings.Replace(filteredEntryRule, "Or([IsCardtype(Artifact),IsCardtype(Creature),IsCardtype(Land)])", "Or([])", 1),
		`ReplaceWouldEnter(APermanentWouldEnterTheBattlefield(IsCardtype(Creature)),[EntersTapped])`,
		`SpellActions(ActionList([` + filteredEntryRule + `]))`,
	} {
		n, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+bad+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(n[0]); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}

func TestRONFilteredEntryInventoryAndAtomicLoad(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Kismet", "kismet.ron")
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	record := fstest.MapFS{"kismet.ron": {Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}}
	report, err := InspectFS(record, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"APermanentWouldEnterTheBattlefield": "static_entry_event", "ControlledByAPlayer": "entry_permanent_filter", "Opponent": "player_filter", "EntersTapped": "static_entry_operation", "Artifact": "card_type"} {
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
	if err := LoadInto(registry, record, "*.ron"); err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Kismet")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Kismet")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("factory shared identities")
	}
	invalid := fstest.MapFS{"bad.ron": {Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + strings.Replace(filteredEntryRule, "Opponent", "Trigger_ThatPlayer", 1) + `])`)}}
	if err := LoadInto(registry, invalid, "*.ron"); err == nil {
		t.Fatal("invalid collection loaded")
	}
	if _, err := registry.CreateCard("Kismet"); err != nil {
		t.Fatal("failed load removed previous registry", err)
	}
}

func TestRONFilteredEntryCopiedKismet(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Kismet")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 3)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("spell copy")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("original missing")
	}
	var copiedPermanent *mage.Permanent
	for _, p := range g.AllBattlefield() {
		if p.Name() == "Kismet" && p.ID() != card.ID() {
			copiedPermanent = p
		}
	}
	if copiedPermanent == nil || !copiedPermanent.IsToken || copiedPermanent.ControllerID() != b.PlayerID() {
		t.Fatal("permanent copy")
	}
	if !g.PutOnBattlefield(mage.NewLand("A land"), a.PlayerID()).Tapped || !g.PutOnBattlefield(mage.NewLand("B land"), b.PlayerID()).Tapped {
		t.Fatal("copies shared controller")
	}
	g.RemoveFromBattlefield(copiedPermanent)
	if g.PutOnBattlefield(mage.NewLand("A alone"), a.PlayerID()).Tapped || !g.PutOnBattlefield(mage.NewLand("B alone"), b.PlayerID()).Tapped {
		t.Fatal("copied rule lifecycle")
	}
}
