package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAfterCombatHistory(t *testing.T) {
	g := newPriorityTestGame()
	condition := AfterFirstCombat()
	check := func(game *Game, want bool) {
		t.Helper()
		if got := condition.checkState(game, StateContext{}); got != want || game.CombatHasEnded() != want {
			t.Fatalf("after combat at %s = %v, want %v", game.GetStep(), got, want)
		}
	}
	for _, step := range []PhaseStep{PrecombatMain, BeginCombat, DeclareAttackers, DeclareBlockers, FirstStrikeDamage, CombatDamage, EndCombat} {
		g.SetStep(step)
		check(g, false)
	}
	clone := g.Clone()
	calls := 0
	g.onPriority = func(game *Game, _ int, _ bool) PriorityAction {
		calls++
		check(game, false)
		return PriorityAction{Type: PriorityPass}
	}
	g.RunStepWithPriority(EndCombat)
	g.onPriority = autoPass()
	if calls < 2 {
		t.Fatal("end-of-combat priority was not checked")
	}
	check(g, true)
	check(clone, false)
	for _, step := range []PhaseStep{BeginCombat, DeclareAttackers, EndCombat, PostcombatMain, EndStep, Cleanup, Cleanup} {
		g.SetStep(step)
		check(g, true)
	}
	g.SetTurn(g.CurrentTurn() + 1)
	g.SetStep(PrecombatMain)
	check(g, false)
	g.SetStep(EndCombat)
	g.SetStep(PostcombatMain)
	check(g, true)
	g.turns.IncrementTurn()
	g.SetStep(PostcombatMain)
	check(g, false)
	if err := ValidateCardDefinition("Timing", NewInstant("Timing", "", WithAbility(NewAction(ActionSpell, WithCastingCondition(condition), WithEffect(GainLifeFor(You, Fixed(1))))))); err != nil {
		t.Fatal(err)
	}
}

func TestAfterCombatSkippedAndEmpty(t *testing.T) {
	for _, beforeBuild := range []bool{true, false} {
		g := newPriorityTestGame()
		if beforeBuild {
			g.SkipNextCombatPhase()
		}
		g.turns.BuildNextTurn()
		if !beforeBuild {
			g.SkipNextCombatPhase()
		}
		for {
			step, ok := g.turns.PopNextStep()
			if !ok {
				t.Fatal("missing main phase")
			}
			if step < PrecombatMain {
				continue
			}
			g.RunStepWithPriority(step)
			if g.CombatHasEnded() {
				t.Fatal("skipped phase completed a combat")
			}
			if step == PostcombatMain {
				break
			}
		}
		g.RunStepWithPriority(BeginCombat)
		g.RunStepWithPriority(DeclareAttackers)
		if g.CombatHasEnded() {
			t.Fatal("empty attackers step completed combat")
		}
		g.SetStep(EndCombat)
		if g.CombatHasEnded() {
			t.Fatal("end-of-combat priority is after combat")
		}
		g.RunStepWithPriority(EndCombat)
		if !g.CombatHasEnded() {
			t.Fatal("empty extra combat did not complete")
		}
	}
	g := newPriorityTestGame()
	g.SetStep(BeginCombat)
	g.SetStep(DeclareAttackers)
	g.SetStep(PostcombatMain)
	if !g.CombatHasEnded() {
		t.Fatal("skipped end step lost actual phase departure")
	}
}
