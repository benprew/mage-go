package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSavedSacrificeSubtypePraetor(t *testing.T) {
	for _, state := range []string{"thrull", "other", "exile", "clone", "copy", "returned", "phase", "self"} {
		t.Run(state, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Ebon Praetor"), a.PlayerID())
			if !source.HasKeyword(core.FirstStrike) || !source.HasKeyword(core.Trample) {
				t.Fatal("intrinsic rules")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if source.CurrentPower(g) != 3 || source.CurrentToughness(g) != 3 {
				t.Fatal("upkeep counter")
			}
			g.SetStep(core.PrecombatMain)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err == nil {
				t.Fatal("outside upkeep")
			}
			g.SetStep(core.Upkeep)
			victim := source
			if state != "self" {
				subtype := "Thrull"
				if state == "other" {
					subtype = "Goblin"
				}
				victim = g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 1, mage.WithSubTypes(subtype)), a.PlayerID())
				a.choice = victim.ID()
			}
			if state == "exile" {
				g.AddExileIfWouldGoToGraveyardThisTurn(victim.ID(), source.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("cost not paid at announcement")
			}
			if state == "self" {
				g.ResolveStack()
				return
			}
			if source.CurrentPower(g) != 3 {
				t.Fatal("body ran during payment")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err == nil {
				t.Fatal("once each turn")
			}
			clone := g.Clone()
			switch state {
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "returned":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(source.ID())
				g.PutOnBattlefield(source.Card, a.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			}
			g.ResolveStack()
			current := g.FindPermanentIncludingPhased(source.ID())
			wantPower, wantToughness := 6, 5
			if state == "other" {
				wantPower = 5
			}
			if state == "copy" {
				wantPower = 7
			}
			if state == "returned" {
				wantPower = 5
			}
			if state == "phase" {
				wantPower, wantToughness = 3, 3
			}
			if current.CurrentPower(g) != wantPower || current.CurrentToughness(g) != wantToughness {
				t.Fatalf("P/T %d/%d want %d/%d", current.CurrentPower(g), current.CurrentToughness(g), wantPower, wantToughness)
			}
			if state == "clone" {
				clone.ResolveStack()
				if clone.FindPermanent(source.ID()).CurrentPower(clone) != 6 {
					t.Fatal("clone payment facts")
				}
			}
		})
	}
}

func TestRONSavedSacrificeSubtypeDjinn(t *testing.T) {
	n := staticBaseRecord(t, "ARN", "Serendib Djinn")
	if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "land dependence filter") {
		t.Fatal("expected separate state-event blocker", err)
	}
	rule := n.Fields["Rules"].Args[1]
	factory := staticBaseSubrule(t, n, rule)
	for _, subtype := range []string{"Island", "Swamp"} {
		g, a, b := ronGame()
		g.PutOnBattlefield(factory(), a.PlayerID())
		victim := g.PutOnBattlefield(mage.NewLand("Land", mage.WithSubTypes(subtype)), a.PlayerID())
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("opponent upkeep")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.FindPermanent(victim.ID()) == nil {
			t.Fatal("sacrifice before resolution")
		}
		g.ResolveStack()
		want := 20
		if subtype == "Island" {
			want = 17
		}
		if g.FindPermanent(victim.ID()) != nil || a.Life() != want {
			t.Fatal("saved land property", a.Life(), want)
		}
	}
}

func TestRONSavedSacrificeSubtypeDeclarations(t *testing.T) {
	for _, body := range []string{
		"If(APermanentWasSacrificedThisWay(IsLandType(Island)),[GainLife(Integer(1))])",
		"SacrificeAPermanent(IsCardtype(Land)),If(APermanentWasSacrificedThisWay,[GainLife(Integer(1))])",
		"SacrificeAPermanent(IsCardtype(Land)),If(APermanentWasSacrificedThisWay(IsLandType(Thrull)),[GainLife(Integer(1))])",
		"SacrificeAPermanent(IsCardtype(Land)),If(APermanentWasSacrificedThisWay(IsCreatureType(Island)),[GainLife(Integer(1))])",
		"SacrificeAPermanent(IsCardtype(Land)),If(APermanentWasSacrificedThisWay(IsTapped),[GainLife(Integer(1))])",
		"SacrificeAPermanent(IsCardtype(Land)),If(APermanentWasSacrificedThisWay(IsLandType(Island())),[GainLife(Integer(1))])",
		"If(PermanentPassesFilter(ThisPermanent,IsTapped),[SacrificeAPermanent(IsCardtype(Land))]),If(APermanentWasSacrificedThisWay(IsLandType(Island)),[GainLife(Integer(1))])",
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("invalid accepted", body)
		}
	}
}

func TestRONSavedSacrificeSubtypeIsolation(t *testing.T) {
	factory := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([SacrificeAPermanent(IsLandType(Island)),If(APermanentWasSacrificedThisWay(IsLandType(Island)),[GainLife(Integer(2))]),SacrificeAPermanent(IsLandType(Swamp)),If(APermanentWasSacrificedThisWay(IsLandType(Island)),[GainLife(Integer(8))])]))`)
	g, a, b := ronGame()
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	first := g.PutOnBattlefield(mage.NewLand("Island", mage.WithSubTypes("Island")), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewLand("Swamp", mage.WithSubTypes("Swamp")), a.PlayerID())
	c := factory()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("announcement ran producers")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy")
	}
	clone := g.Clone()
	clone.PhaseOut(first.ID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("producer or copy results mixed")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanentIncludingPhased(first.ID()) == nil {
		t.Fatal("clone mixed outcomes")
	}
}

func TestRONSavedSacrificeSubtypePaymentIsolation(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(SacrificeAPermanent(IsCardtype(Creature)),ActionList([If(APermanentWasSacrificedThisWay(IsCreatureType(Thrull)),[GainLife(Integer(2))])]))`)
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), a.PlayerID())
	thrull := g.PutOnBattlefield(mage.NewCreature("Thrull", "", 1, 1, mage.WithSubTypes("Thrull")), a.PlayerID())
	goblin := g.PutOnBattlefield(mage.NewCreature("Goblin", "", 1, 1, mage.WithSubTypes("Goblin")), a.PlayerID())
	a.choice = thrull.ID()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.choice = goblin.ID()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.PutOnBattlefield(thrull.Card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("independent payment snapshots")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone payment snapshots")
	}
}

func TestRONSavedSacrificeSubtypeInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"card.ron": {Data: []byte(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(SacrificeAPermanent(IsCardtype(Creature)),ActionList([If(APermanentWasSacrificedThisWay(IsCreatureType(Thrull)),[GainLife(Integer(1))])]))])`)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "IsCreatureType" && sig.Role == "saved_sacrifice_subtype_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("saved-property filter role")
	}
}
