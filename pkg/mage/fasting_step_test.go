package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestFastingSkipsWholeStep(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.turns.SetTurn(3)
	g.turns.SetStep(Upkeep)
	card := NewEnchantment("Fasting replacement", "", WithAbility(Static(SkipDrawStepForLife(2))))
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Draw watcher", "", WithAbility(At(EachDrawStep, DrawFor(You, Fixed(1))))), a.PlayerID())
	for range 6 {
		a.AddToLibrary(NewLand("Card"))
	}
	if err := ApplyEffect(g, AtNext(EachDrawStep, GainLifeFor(You, Fixed(5))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	priorityCalls := 0
	g.SetOnPriority(func(*Game, int, bool) PriorityAction {
		priorityCalls++
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Draw)
	if priorityCalls != 0 {
		t.Fatal("skipped draw step granted priority")
	}
	if len(a.Hand()) != 0 || a.Life() != 20 || g.GetStep() != Upkeep || g.StackSize() != 0 {
		t.Fatalf("skipped step: hand=%d life=%d step=%v stack=%d", len(a.Hand()), a.Life(), g.GetStep(), g.StackSize())
	}
	clone := g.Clone()
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.RunStepWithPriority(PrecombatMain)
	if a.Life() != 22 {
		t.Fatalf("next step life = %d, want 22", a.Life())
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("gain changed clone")
	}
	clone.RunStepWithPriority(EndStep)
	if clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone lost deferred gain")
	}
	g.RunStepWithPriority(Draw)
	if len(a.Hand()) != 2 || a.Life() != 27 {
		t.Fatalf("next actual draw: hand=%d life=%d", len(a.Hand()), a.Life())
	}
	g.RunStepWithPriority(EndStep)
	if a.Life() != 27 {
		t.Fatal("gain repeated")
	}
}

func TestFastingDrawStepEligibility(t *testing.T) {
	for _, scenario := range []string{"declined", "first-turn", "other-player", "phased", "departed", "normal-draw-action"} {
		t.Run(scenario, func(t *testing.T) {
			var a Player = &acceptingMayPlayer{NewBasePlayer("A")}
			if scenario == "declined" {
				a = NewBasePlayer("A")
			}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.turns.SetTurn(3)
			card := NewEnchantment("Fasting replacement", "", WithAbility(Static(SkipDrawStepForLife(2))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			for range 3 {
				a.AddToLibrary(NewLand("Card"))
				b.AddToLibrary(NewLand("Card"))
			}
			switch scenario {
			case "first-turn":
				g.turns.SetTurn(1)
			case "other-player":
				g.turns.SetActivePlayerIndex(1)
			case "phased":
				g.PhaseOut(source.ID())
			case "departed":
				g.DestroyPermanent(source)
			}
			if scenario == "normal-draw-action" {
				g.doDrawNormalDraw()
			} else {
				g.RunStepWithPriority(Draw)
			}
			g.RunStepWithPriority(PrecombatMain)
			if a.Life() != 20 {
				t.Fatalf("ineligible gain: %d", a.Life())
			}
			want := 1
			if scenario == "first-turn" {
				want = 0
			}
			if got := len(g.ActivePlayerObj().Hand()); got != want {
				t.Fatalf("hand = %d, want %d", got, want)
			}
		})
	}
}

func TestFastingGainWaitsThroughSkippedSteps(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	g.turns.SetTurn(3)
	for range 2 {
		g.PutOnBattlefield(NewEnchantment("Fasting replacement", "", WithAbility(Static(SkipDrawStepForLife(2)))), a.PlayerID())
	}
	g.RunStepWithPriority(Draw)
	if a.Life() != 20 {
		t.Fatal("gain occurred in skipped step")
	}
	g.turns.Schedule().Remaining = []PhaseStep{PrecombatMain, BeginCombat}
	g.SkipNextOccurrenceOfStep(PrecombatMain)
	next, ok := g.turns.PopNextStep()
	if !ok || next != BeginCombat {
		t.Fatal("main phase was not skipped")
	}
	g.PutOnBattlefield(NewEnchantment("Combat observer", "", WithAbility(NewTriggered(EvtBeginCombat, false,
		FuncEffect("check life", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
			if game.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Error("gain did not precede combat trigger")
			}
			return nil
		})))), a.PlayerID())
	g.RunStepWithPriority(next)
	if a.Life() != 22 {
		t.Fatalf("multiple replacements gained %d life, want 2", a.Life()-20)
	}
}
