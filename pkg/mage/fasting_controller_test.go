package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestFastingReplacementCurrentController(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	b := &acceptingMayPlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewEnchantment("Draw step replacement", "", WithAbility(Static(SkipDrawStepForLife(2)))), a.PlayerID())
	assertScope := func(game *Game, controller, opponent uuid.UUID) {
		t.Helper()
		if got := game.effects.ApplyReplacements(&beginDrawStepAction{playerID: opponent}, game); got == nil {
			t.Fatal("replacement applied to the opponent")
		}
		if got := game.effects.ApplyReplacements(&beginDrawStepAction{playerID: controller}, game); got != nil {
			t.Fatal("replacement did not apply to the current controller")
		}
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	assertScope(clone, b.PlayerID(), a.PlayerID())
	assertScope(g, a.PlayerID(), b.PlayerID())
	changedClone := clone.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	assertScope(clone, a.PlayerID(), b.PlayerID())
	assertScope(changedClone, b.PlayerID(), a.PlayerID())
	clone.RunStepWithPriority(PrecombatMain)
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("pending gains did not retain the player who skipped each step")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone gains changed the original game")
	}
}

func TestFastingReplacementSourceLifetime(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	card := NewEnchantment("Draw step replacement", "")
	g.effects.AddReplacement(&drawStepLifeReplacement{
		replacementBase: replacementBase{sourceID: card.ID(), duration: WhileOnBattlefield},
		amount:          2,
	})
	source := g.PutOnBattlefield(card, a.PlayerID())
	action := &beginDrawStepAction{playerID: a.PlayerID()}
	if g.effects.ApplyReplacements(action, g) == nil {
		t.Fatal("absent-source registration applied to a later permanent")
	}
	g.effects.AddReplacement(&drawStepLifeReplacement{
		replacementBase: replacementBase{sourceID: source.ID(), duration: WhileOnBattlefield},
		incarnation:     source.incarnationID, amount: 2,
	})
	clone := g.Clone()
	g.PhaseOut(source.ID())
	if g.effects.ApplyReplacements(action, g) == nil {
		t.Fatal("phased source replaced a step")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	if g.effects.ApplyReplacements(action, g) != nil {
		t.Fatal("phasing in lost the replacement")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	if g.effects.ApplyReplacements(action, g) == nil {
		t.Fatal("departed registration applied to a returned source")
	}
	if clone.effects.ApplyReplacements(action, clone) != nil {
		t.Fatal("source departure changed clone registration")
	}
	g.effects.AddReplacement(&drawStepLifeReplacement{
		replacementBase: replacementBase{sourceID: card.ID(), duration: WhileOnBattlefield},
		incarnation:     g.FindPermanent(card.ID()).incarnationID, amount: 2,
	})
	if g.effects.ApplyReplacements(action, g) != nil {
		t.Fatal("new registration did not apply to the returned source")
	}
}
