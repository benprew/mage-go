package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONSourceBlockTimeElementalNative(t *testing.T) {
	for _, attack := range []bool{false, true} {
		t.Run(fmt.Sprint(attack), func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			card := repositoryCard(t, "LEG", "Time Elemental")
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			other := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), b.PlayerID())
			other.RevokeBaseAttr(core.AttrSummonSick)
			if attack {
				a.SetAttackers(1, []string{"Time Elemental"})
			} else {
				g.SetActivePlayerIndex(1)
				b.SetAttackers(1, []string{"Attacker"})
				a.SetBlockers(1, map[string]string{"Time Elemental": "Attacker"})
			}
			g.RunStepWithPriority(core.DeclareAttackers)
			if !attack && len(g.DelayedTriggers()) != 0 {
				t.Fatal("block trigger ran at attack")
			}
			g.RunStepWithPriority(core.DeclareBlockers)
			if len(g.DelayedTriggers()) != 1 || a.Life() != 20 || g.FindPermanent(p.ID()) == nil {
				t.Fatal("delay timing")
			}
			if !attack && !g.GetCombat().IsBlocking(p.ID()) {
				t.Fatal("native block rejected")
			}
			g.RunStepWithPriority(core.EndCombat)
			if g.FindPermanent(p.ID()) != nil || a.Life() != 15 || b.Life() != 20 || len(a.Graveyard()) != 1 {
				t.Fatal("end of combat sacrifice and damage")
			}
			g.RunStepWithPriority(core.EndCombat)
			if a.Life() != 15 {
				t.Fatal("delay repeated")
			}
		})
	}
}

func TestRONSourceBlockTimeElementalIdentity(t *testing.T) {
	for _, change := range []string{"normal", "return-before", "return-after", "phase", "phase-cycle", "control", "removed-after", "removed-before", "inactive", "copy", "prevent", "two"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Time Elemental")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEG", "Time Elemental"), a.PlayerID())
			if change == "removed-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			if change == "inactive" {
				g.PhaseOut(p.ID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: p.ID(), TargetID: uuid.New(), PlayerID: a.PlayerID(), Flag: true})
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: p.ID(), TargetID: uuid.New(), PlayerID: a.PlayerID(), Flag: false})
			if change == "two" {
				g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: other.ID(), PlayerID: a.PlayerID(), Flag: true})
			}
			if change == "return-before" {
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			g.PutTriggersOnStack()
			want := 1
			if change == "two" {
				want = 2
			}
			if change == "removed-before" || change == "inactive" {
				want = 0
			}
			if g.StackSize() != want || len(g.DelayedTriggers()) != 0 {
				t.Fatal("wrong event multiplicity or resolution timing")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if len(g.DelayedTriggers()) != want+boolInt(change == "copy") {
				t.Fatal("delay installation")
			}
			switch change {
			case "return-after":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "removed-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "prevent":
				g.AddPreventionShield(a.PlayerID(), 3)
			}
			clone := g.Clone()
			clone.RunStepWithPriority(core.EndCombat)
			if a.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.RunStepWithPriority(core.EndCombat)
			for _, game := range []*mage.Game{g, clone} {
				present := change == "return-before" || change == "return-after" || change == "phase" || change == "control" || want == 0
				if (game.FindPermanentIncludingPhased(p.ID()) != nil) != present {
					t.Fatal("sacrifice followed wrong incarnation or controller")
				}
				life := 20 - 5*want
				if change == "prevent" {
					life += 3
				}
				if game.GetPlayer(a.PlayerID()).Life() != life {
					t.Fatalf("life %d want %d", game.GetPlayer(a.PlayerID()).Life(), life)
				}
				otherLife := 20
				if change == "copy" {
					otherLife = 15
				}
				if game.GetPlayer(b.PlayerID()).Life() != otherLife {
					t.Fatal("wrong delayed controller")
				}
				if (game.FindPermanent(other.ID()) == nil) != (change == "two") {
					t.Fatal("crossed source instances")
				}
			}
		})
	}
}

func TestRONSourceBlockTimeElementalActivation(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Time Elemental")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.CurrentPower(g) != 0 || p.CurrentToughness(g) != 2 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("casting metadata or entry trigger")
	}
	p.RevokeBaseAttr(core.AttrSummonSick)
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{target.ID()}); err == nil || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("unpaid activation")
	}
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Blue, 2)
	aura := attachRONTestAura(t, g, target, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{target.ID()}); err == nil || a.ManaPool().TotalMana() != 4 || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("enchanted target rollback")
	}
	g.RemoveFromBattlefield(aura)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("activation payment")
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()) != nil || len(b.Hand()) != 1 {
		t.Fatal("return to owner")
	}
}

func TestRONSourceBlockInvalidAndInventory(t *testing.T) {
	valid := `TriggerA(WhenACreatureBlocks(SinglePermanent(ThisPermanent)),ActionList([TapPermanent(Trigger_ThatCreature)]))`
	for _, rule := range []string{
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(HostPermanent)", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent())", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "AnyPermanent", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(Ref_TargetPermanent)", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "WhenACreatureBlocks(SinglePermanent(ThisPermanent))", "WhenACreatureBlocks", 1),
		strings.Replace(valid, "WhenACreatureBlocks(SinglePermanent(ThisPermanent))", "WhenACreatureBlocks(SinglePermanent(ThisPermanent),AnyPermanent)", 1),
		strings.Replace(valid, "Trigger_ThatCreature", "Trigger_ThatOtherCreature", 1),
		strings.Replace(valid, "Trigger_ThatCreature", "Trigger_ThatCreature()", 1),
		valid + `,Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatCreature)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:4),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	source := `Card(Name:"Block",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:4),Rules:[` + valid + `])`
	report, err := InspectFS(fstest.MapFS{"b.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal("block declaration rejected")
	}
	found := false
	for _, s := range report.Signatures {
		if s.Name == "ThisPermanent" && strings.Contains(s.Context, "WhenACreatureBlocks") {
			found = true
			if s.Role != "permanent_reference" {
				t.Fatal("event subject role")
			}
		}
	}
	if !found {
		t.Fatal("missing event subject")
	}
}

func TestRONSourceBlockLegalPairsAndEntry(t *testing.T) {
	for _, scenario := range []string{"multiple", "illegal", "enters-blocking"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 5), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5), a.PlayerID())
			g.GetCombat().AddAttacker(first.ID(), b.PlayerID())
			g.GetCombat().AddAttacker(second.ID(), b.PlayerID())
			card := repositoryCard(t, "LEG", "Time Elemental")
			if scenario == "enters-blocking" {
				g.PutOnBattlefieldBlocking(card, b.PlayerID(), first.ID())
				g.PutTriggersOnStack()
				g.ResolveStack()
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("entry blocking triggered blocks")
				}
				return
			}
			p := g.PutOnBattlefield(card, b.PlayerID())
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.CanBlockAny)), p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if scenario == "illegal" {
				g.TapPermanent(g.FindPermanent(p.ID()))
			}
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: first.ID()}, {BlockerID: p.ID(), AttackerID: second.ID()}})
			count := 1
			if scenario == "illegal" {
				count = 0
			}
			if len(g.DelayedTriggers()) != count {
				t.Fatalf("%d delays want %d", len(g.DelayedTriggers()), count)
			}
			g.RunStepWithPriority(core.EndCombat)
			if b.Life() != 20-5*count {
				t.Fatal("one blocks trigger per legal blocker")
			}
		})
	}
}

func TestRONSourceBlockFactoryAndFilterRecheck(t *testing.T) {
	nodes, err := Parse(strings.NewReader(mustRONSet(t, "LEG")), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	var factory mage.CardFactory
	for _, n := range nodes {
		if n.Fields["Name"].String == "Time Elemental" {
			factory, err = Compile(n)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if factory == nil {
		t.Fatal("missing record")
	}
	first, second := factory(), factory()
	if first.Abilities()[1] == second.Abilities()[1] || first.Abilities()[2].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[2].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shares declarations")
	}
	for _, change := range []string{"enchanted", "aura-phased", "returned", "target-phased", "source-leaves", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			card := repositoryCard(t, "LEB", "Llanowar Elves")
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			a.ManaPool().Add(core.Blue, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "enchanted", "aura-phased":
				aura := attachRONTestAura(t, g, target, b.PlayerID())
				if change == "aura-phased" {
					g.PhaseOut(aura.ID())
				}
			case "returned":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "target-phased":
				g.PhaseOut(target.ID())
			case "source-leaves":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				present := change == "enchanted" || change == "returned" || change == "target-phased"
				if (game.FindPermanentIncludingPhased(target.ID()) != nil) != present {
					t.Fatal("target legality recheck")
				}
			}
		})
	}
	for _, filter := range []string{"IsntEnchanted()", "IsntEnchanted(AnyPermanent)"} {
		nodes, parseErr := Parse(strings.NewReader(filter), "invalid-filter.ron")
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		_, err := compileFilter(nodes[0])
		if err == nil {
			t.Fatal("malformed filter accepted")
		}
	}
}
