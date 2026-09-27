package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceDamagedOpponentThisTurnIdentityAndOutcome(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 4)
	old := g.PutOnBattlefield(card, a.PlayerID())
	condition := SourceDamagedOpponentThisTurnCondition()
	ctx := StateContext{SourceID: old.ID(), SourceIncarnation: old.incarnationID, ControllerID: a.PlayerID()}
	if condition.checkState(g, ctx) {
		t.Fatal("matched before damage")
	}
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealDamageToPlayer(b, 1, old.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("prevented damage entered history")
	}
	g.DealDamageToPlayer(a, 1, old.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("damage to controller matched opponent")
	}
	g.DealDamageToPlayer(b, 1, old.ID())
	if !condition.checkState(g, ctx) {
		t.Fatal("opponent damage was not recorded")
	}
	clone := g.Clone()
	clone.damage.ClearEndOfTurn()
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone shared turn history")
	}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.zones.ClearLKI()
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: returned.ID(), SourceIncarnation: returned.incarnationID, ControllerID: a.PlayerID()}) {
		t.Fatal("returned source inherited old history")
	}
	g.DealDamageToPlayer(b, 1, returned.ID())
	newCtx := StateContext{SourceID: returned.ID(), SourceIncarnation: returned.incarnationID, ControllerID: a.PlayerID()}
	if !condition.checkState(g, newCtx) {
		t.Fatal("new incarnation did not record damage")
	}
	returned.PhasedOut = true
	if condition.checkState(g, newCtx) {
		t.Fatal("phased source matched")
	}
	returned.PhasedOut = false
	if !condition.checkState(g, newCtx) {
		t.Fatal("phasing erased source identity")
	}
	g.damage.ClearEndOfTurn()
	if condition.checkState(g, newCtx) {
		t.Fatal("history survived turn cleanup")
	}
}

func TestSourceDamagedOpponentThisTurnInterveningIf(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := SourceDamagedOpponentThisTurnCondition()
	card := NewCreature("Watcher", "", 1, 4, WithAbility(At(EachEndStep, GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	spell := NewInstant("Invalid", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" {
		t.Fatalf("inspection = %+v", meta)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("triggered before damage")
	}
	g.DealDamageToPlayer(b, 1, card.ID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("did not trigger after damage")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != a.Life()+1 {
		t.Fatal("clone trigger did not resolve independently")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("returned source passed resolution recheck")
	}
}

func TestSourceDamagedOpponentThisTurnBatchAndCleanupPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 4)
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	source := g.CaptureDamageSource(permanent.ID())
	condition := SourceDamagedOpponentThisTurnCondition()
	ctx := StateContext{SourceID: permanent.ID(), SourceIncarnation: permanent.incarnationID, ControllerID: a.PlayerID()}
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(a, 1, source),
		PlayerDamageAssignment(b, 1, source),
	)
	if condition.checkState(g, ctx) {
		t.Fatal("fully prevented batch recipient entered history")
	}
	g.DealSimultaneousDamage(PlayerDamageAssignment(b, 1, source))
	if !condition.checkState(g, ctx) {
		t.Fatal("batch damage did not enter turn history")
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	if !g.doCleanupActions() || !condition.checkState(g, ctx) {
		t.Fatal("cleanup priority removed history early")
	}
	g.ResolveStack()
	g.RunStepWithPriority(Cleanup)
	if condition.checkState(g, ctx) {
		t.Fatal("final cleanup kept turn history")
	}
}
