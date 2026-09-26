package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSourceTappedState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("State source", "", 1, 1)
	g.PutOnBattlefield(card, a.PlayerID())
	p := g.FindPermanent(card.ID())
	ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
	for _, tc := range []struct{ tapped, want bool }{{false, false}, {true, true}} {
		g.MutablePermanent(card.ID()).Tapped = tc.tapped
		if got := SourceTappedCondition().checkState(g, ctx); got != tc.want {
			t.Fatalf("tapped=%v: got %v", tc.tapped, got)
		}
		if got := SourceTappedCondition().checkState(g.Clone(), ctx); got != tc.want {
			t.Fatalf("clone tapped=%v: got %v", tc.tapped, got)
		}
		if got := SourceUntappedCondition().checkState(g, ctx); got == tc.want {
			t.Fatalf("untapped=%v: got %v", !tc.tapped, got)
		}
	}
	clone := g.Clone()
	clone.MutablePermanent(card.ID()).Tapped = false
	if !SourceTappedCondition().checkState(g, ctx) || SourceTappedCondition().checkState(clone, ctx) {
		t.Fatal("clone shares tapped state")
	}
	g.MutablePermanent(card.ID()).PhasedOut = true
	if SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("phased-out source matched")
	}
	if SourceUntappedCondition().checkState(g, ctx) {
		t.Fatal("phased-out source matched untapped")
	}
	g.MutablePermanentIncludingPhased(card.ID()).PhasedOut = false
	if !SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("phased-in source lost identity")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
	if SourceUntappedCondition().checkState(g, ctx) {
		t.Fatal("absent source matched untapped")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(card.ID()).Tapped = true
	if SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("returned source matched old incarnation")
	}
	if SourceTappedCondition().checkState(g, StateContext{SourceID: uuid.Nil}) {
		t.Fatal("missing source matched")
	}
}

func TestSourceTappedStateRegistration(t *testing.T) {
	card := NewCreature("Tapped state", "", 1, 1, WithAbility(WhenState(SourceTappedCondition(), Sacrifice(This))))
	if err := ValidateCardDefinition("Tapped state", card); err != nil {
		t.Fatal(err)
	}
	spell := NewInstant("Invalid state", "", WithEffect(IfState(SourceTappedCondition(), Sacrifice(This))))
	if err := ValidateCardDefinition("Invalid state", spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceTappedStaticAndInterveningIf(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Tapped source", "", 1, 1,
		WithAbility(Static(While(SourceUntappedCondition(), ModifyPT(This, Fixed(0), Fixed(3))))),
		WithAbility(WhenEvent(Enters(This), GainLifeFor(You, Fixed(2)), WithInterveningIf(SourceUntappedCondition()))))
	if err := ValidateCardDefinition("Tapped source", card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 4 {
		t.Fatalf("untapped toughness = %d", got)
	}
	g.PutTriggersOnStack()
	g.MutablePermanent(card.ID()).Tapped = true
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 1 {
		t.Fatalf("tapped toughness = %d", got)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("intervening condition failed recheck: life = %d", a.Life())
	}
}
