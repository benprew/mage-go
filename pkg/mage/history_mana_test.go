package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func historyManaSource(g *Game, a Player) *Permanent {
	history := BindActivationHistory("conversion")
	ability := NewAction(ActionActivated, WithCost(ManaPayment("{1}")), WithEffect(Sequence(AddMana(Black, 1), IfActivationsAtLeast(history, 4, AtNext(EachEndStep, Sacrifice(This))))), WithActivationHistory(history))
	return g.PutOnBattlefield(NewCreature("History conversion", "", 1, 3, WithAbility(ability)), a.PlayerID())
}

func TestHistoryManaPlanning(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := historyManaSource(g, a)
	a.ManaPool().Add(Red, 5)
	plan, err := g.mana.PlanManaForCost(g, a.PlayerID(), ManaCost{Black: 5}, AutoTapHint{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 0 || a.ManaPool().Count(Red) != 5 {
		t.Fatal("planning changed state")
	}
	clone := g.Clone()
	if err := g.mana.ApplyManaSolution(g, a.PlayerID(), plan); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Black) != 5 || g.StackSize() != 0 || len(g.DelayedTriggers()) != 2 {
		t.Fatal("planned activation lost its synchronous body")
	}
	if len(clone.DelayedTriggers()) != 0 || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Red) != 5 {
		t.Fatal("clone shared execution")
	}
	g.RunStepWithPriority(EndStep)
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("delayed sacrifice missing")
	}
	if clone.FindPermanent(p.ID()) == nil {
		t.Fatal("clone shared sacrifice")
	}
}

func TestHistoryManaMixedExecution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := historyManaSource(g, a)
	for range 3 {
		a.ManaPool().Clear()
		a.ManaPool().Add(Red, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(Red, 1)
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Black: 1}); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 1 || a.ManaPool().Count(Black) != 1 {
		t.Fatal("planned activation did not read explicit history")
	}
	a.ManaPool().Clear()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation accepted")
	}
	if len(g.DelayedTriggers()) != 1 {
		t.Fatal("failed payment created delay")
	}
	a.ManaPool().Add(Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 2 {
		t.Fatal("explicit activation did not read planned history")
	}
}

func TestHistoryManaExcludedBodies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, kind := range []string{"immediate", "choice", "conditional-mana", "wrong-binding"} {
		history := BindActivationHistory("conversion")
		var body = AtNext(EachEndStep, Sacrifice(This))
		var mana = AddMana(Black, 1)
		read := history
		switch kind {
		case "immediate":
			body = Sacrifice(This)
		case "choice":
			mana = AddManaOfAnyOneColor(Fixed(1))
		case "conditional-mana":
			body = AddMana(Black, 1)
		case "wrong-binding":
			read = BindActivationHistory("other")
		}
		ability := NewAction(ActionActivated, WithCost(ManaPayment("{1}")), WithActivationHistory(history), WithEffect(Sequence(mana, IfActivationsAtLeast(read, 4, body))))
		p := g.PutOnBattlefield(NewArtifact("Excluded", "", WithAbility(ability)), a.PlayerID())
		if _, ok := manaSourceAbilityForPlanning(p.RuntimeAbilities[0], p.ID(), 0, g); ok {
			t.Fatal("unsupported body planned", kind)
		}
	}
}

func TestHistoryManaActivationEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := historyManaSource(g, a)
	activations, taps := 0, 0
	watch := NewTriggered(EvtAbilityActivated, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == p.ID() {
			activations++
			if evt.Flag || len(g.DelayedTriggers()) != max(0, activations-4) || a.ManaPool().Count(Black) != activations-1 {
				t.Fatal("activation event timing or tap flag")
			}
		}
		return false
	})
	tapWatch := NewTriggered(EvtTappedForMana, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == p.ID() {
			taps++
		}
		return false
	})
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(watch), WithAbility(tapWatch)), a.PlayerID())
	a.ManaPool().Add(Red, 5)
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Black: 5}); err != nil {
		t.Fatal(err)
	}
	if activations != 5 || taps != 0 || len(g.DelayedTriggers()) != 2 {
		t.Fatal("lost or duplicate events")
	}
}
