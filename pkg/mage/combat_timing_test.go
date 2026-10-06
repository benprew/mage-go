package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatTimingHistory(t *testing.T) {
	g := newPriorityTestGame()
	check := func(game *Game, attackers, blockers, damage, during bool) {
		t.Helper()
		for _, tc := range []struct {
			condition StateCondition
			want      bool
		}{
			{BeforeCombatStep(DeclareAttackers, false), attackers},
			{BeforeCombatStep(DeclareBlockers, false), blockers},
			{BeforeCombatStep(CombatDamage, false), damage},
			{BeforeCombatStep(DeclareBlockers, true), during},
		} {
			if got := tc.condition.checkState(game, StateContext{}); got != tc.want {
				t.Fatalf("%s at %s = %v, want %v", tc.condition.inspectDefinition().Text, game.GetStep(), got, tc.want)
			}
		}
	}
	g.SetStep(PrecombatMain)
	check(g, true, true, true, false)
	g.SetStep(BeginCombat)
	check(g, true, true, true, true)
	g.SetStep(DeclareAttackers)
	check(g, false, true, true, true)
	clone := g.Clone()
	g.SetStep(DeclareBlockers)
	check(g, false, false, true, false)
	check(clone, false, true, true, true)
	g.SetStep(FirstStrikeDamage)
	check(g, false, false, false, false)
	g.SetStep(CombatDamage)
	g.SetStep(EndCombat)
	check(g, false, false, false, false)
	g.RunStepWithPriority(EndCombat)
	check(g, false, false, false, false)
	g.SetStep(BeginCombat)
	check(g, false, false, false, true)
	g.SetStep(DeclareAttackers)
	check(g, false, false, false, true)
	g.SetStep(DeclareBlockers)
	check(g, false, false, false, false)
	g.SetTurn(g.CurrentTurn() + 1)
	g.SetStep(Untap)
	check(g, true, true, true, false)
	g.turns.IncrementTurn()
	g.SetStep(PrecombatMain)
	check(g, true, true, true, false)
}

func TestCombatTimingSkippedPoints(t *testing.T) {
	for _, steps := range [][]PhaseStep{
		{PrecombatMain, BeginCombat, DeclareAttackers, EndCombat},
		{PrecombatMain, PostcombatMain},
		{PrecombatMain, BeginCombat, EndCombat},
	} {
		g := newPriorityTestGame()
		for _, step := range steps {
			g.SetStep(step)
		}
		for _, point := range []PhaseStep{DeclareAttackers, DeclareBlockers, CombatDamage} {
			if BeforeCombatStep(point, false).checkState(g, StateContext{}) {
				t.Fatalf("skipped point %s remains open after %v", point, steps)
			}
		}
		g.SetStep(BeginCombat)
		if BeforeCombatStep(CombatDamage, false).checkState(g, StateContext{}) || !BeforeCombatStep(CombatDamage, true).checkState(g, StateContext{}) {
			t.Fatal("extra combat restored first-combat window or lost current window")
		}
	}
	for _, point := range []PhaseStep{Untap, BeginCombat, FirstStrikeDamage, EndCombat, Cleanup} {
		card := NewInstant("Invalid point", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(BeforeCombatStep(point, false)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid point accepted")
		}
	}
}

func TestCombatTimingNativeSkippedSteps(t *testing.T) {
	g := newPriorityTestGame()
	g.RunStepWithPriority(PrecombatMain)
	g.RunStepWithPriority(BeginCombat)
	g.SetStep(DeclareAttackers)
	if !g.CombatWindowOpen(CombatDamage, false) {
		t.Fatal("window closed before attackers step ended")
	}
	clone := g.Clone()
	g.RunStepWithPriority(DeclareAttackers)
	if g.CombatWindowOpen(DeclareBlockers, false) || g.CombatWindowOpen(CombatDamage, false) || !clone.CombatWindowOpen(CombatDamage, false) {
		t.Fatal("empty combat boundary or clone isolation")
	}
	g.RunStepWithPriority(DeclareBlockers)
	g.RunStepWithPriority(FirstStrikeDamage)
	g.RunStepWithPriority(CombatDamage)
	if g.GetStep() != DeclareAttackers {
		t.Fatal("skipped step became current")
	}
	g.RunStepWithPriority(EndCombat)
	g.SetTurn(g.CurrentTurn() + 1)
	g.SetStep(PrecombatMain)
	g.SkipNextCombatPhase()
	g.turns.BuildNextTurn()
	for {
		step, ok := g.turns.PopNextStep()
		if !ok {
			t.Fatal("missing postcombat main")
		}
		if step < PrecombatMain {
			continue
		}
		g.RunStepWithPriority(step)
		if step == PrecombatMain && g.CombatWindowOpen(CombatDamage, false) {
			t.Fatal("skip left window open after main ended")
		}
		if step == PostcombatMain {
			break
		}
	}
	if g.CombatWindowOpen(CombatDamage, false) {
		t.Fatal("skipped phase restored window")
	}
}

func TestCombatTimingSkippedFirstThenExtraCombat(t *testing.T) {
	for _, beforeBuild := range []bool{true, false} {
		g := newPriorityTestGame()
		if beforeBuild {
			g.SkipNextCombatPhase()
		}
		g.turns.BuildNextTurn()
		if !beforeBuild {
			g.SkipNextCombatPhase()
		}
		g.turns.Schedule().Remaining = append(g.turns.Schedule().Remaining[:len(g.turns.Schedule().Remaining)-3], BeginCombat, DeclareAttackers, DeclareBlockers, CombatDamage, EndCombat, PostcombatMain, EndStep, Cleanup)
		for {
			step, ok := g.turns.PopNextStep()
			if !ok {
				t.Fatal("no extra combat")
			}
			if step < PrecombatMain {
				continue
			}
			if step == BeginCombat {
				g.SetStep(step)
				if g.CombatWindowOpen(CombatDamage, false) || !g.CombatWindowOpen(CombatDamage, true) {
					t.Fatal("skipped first phase became actual extra phase")
				}
				break
			}
			g.RunStepWithPriority(step)
		}
	}
}

func TestCombatTimingNativeDamageSteps(t *testing.T) {
	for _, kind := range []string{"none", "printed", "granted"} {
		g := newPriorityTestGame()
		options := []CardOption{}
		if kind == "printed" {
			options = append(options, WithKeyword(FirstStrike))
		}
		if kind == "granted" {
			options = append(options, WithAbility(Static(While(Controls(IsCreature), GrantKeywordModification(This, FirstStrike)))))
		}
		attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 10, options...), g.players[0].PlayerID())
		g.GetCombat().AddAttacker(attacker.ID(), g.players[1].PlayerID())
		g.SetStep(DeclareBlockers)
		g.RunStepWithPriority(FirstStrikeDamage)
		if g.CombatWindowOpen(CombatDamage, false) == (kind != "none") {
			t.Fatal("first strike step boundary")
		}
		g.RunStepWithPriority(CombatDamage)
		if g.CombatWindowOpen(CombatDamage, false) {
			t.Fatal("normal damage step did not close window")
		}
	}
}

func TestCombatTimingSkippedBeginningStep(t *testing.T) {
	g := newPriorityTestGame()
	g.SkipNextOccurrenceOfStep(BeginCombat)
	g.turns.BuildNextTurn()
	g.RunStepWithPriority(PrecombatMain)
	if !g.CombatWindowOpen(CombatDamage, false) {
		t.Fatal("one step skip removed the phase")
	}
	g.SetStep(DeclareAttackers)
	if !g.CombatWindowOpen(DeclareBlockers, false) {
		t.Fatal("remaining combat window lost")
	}
	g.SetStep(EndCombat)
	g.SetStep(PostcombatMain)
	g.SetStep(DeclareAttackers)
	if g.CombatWindowOpen(DeclareBlockers, false) || !g.CombatWindowOpen(DeclareBlockers, true) {
		t.Fatal("extra combat with no beginning step lost its current window")
	}
}
