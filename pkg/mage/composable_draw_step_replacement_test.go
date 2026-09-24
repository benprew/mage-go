package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticDrawStepReplacementLifetime(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Static skip", "", 1, 4, WithAbility(Static(SkipDrawStepForLife(3))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	assertSkip := func(game *Game, want bool) {
		t.Helper()
		got := game.effects.ApplyReplacements(&beginDrawStepAction{playerID: a.PlayerID()}, game) == nil
		if got != want {
			t.Fatalf("skip = %v, want %v", got, want)
		}
	}
	g.PutOnBattlefield(card, a.PlayerID())
	assertSkip(g, true)
	if len(g.triggers.Pending()) != 0 || g.stack.Size() != 0 || len(g.effects.replacements) != 0 {
		t.Fatal("static rule used entry registration")
	}
	clone := g.Clone()
	clone.AddContinuousEffect(RemoveAllAbilities(card.ID(), EndOfTurn))
	assertSkip(clone, false)
	assertSkip(g, true)
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	assertSkip(clone, true)
	clone.PhaseOut(card.ID())
	assertSkip(clone, false)
	clone.PhaseIn([]uuid.UUID{card.ID()})
	assertSkip(clone, true)
	clone.RemoveFromBattlefield(clone.FindPermanent(card.ID()))
	assertSkip(clone, false)
	clone.PutOnBattlefield(card, a.PlayerID())
	assertSkip(clone, true)
	if len(clone.turns.pendingStepLifeGains) != 4 {
		t.Fatalf("pending gains = %d", len(clone.turns.pendingStepLifeGains))
	}
	clone.RunStepWithPriority(PrecombatMain)
	if clone.GetPlayer(a.PlayerID()).Life() != 32 || a.Life() != 20 {
		t.Fatal("gain amount or clone isolation failed")
	}
}

func TestStaticDrawStepReplacementControl(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	b := &acceptingMayPlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	card := NewCreature("Static skip", "", 1, 4, WithAbility(Static(SkipDrawStepForLife(2))))
	g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.effects.ApplyReplacements(&beginDrawStepAction{playerID: a.PlayerID()}, g) == nil {
		t.Fatal("old controller skipped")
	}
	if g.effects.ApplyReplacements(&beginDrawStepAction{playerID: b.PlayerID()}, g) != nil {
		t.Fatal("new controller did not skip")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.RunStepWithPriority(PrecombatMain)
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatal("pending gain changed recipient")
	}
}

func TestStaticDrawStepReplacementDecline(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 2 {
		g.PutOnBattlefield(NewEnchantment("Static skip", "", WithAbility(Static(SkipDrawStepForLife(2)))), a.PlayerID())
	}
	action := &beginDrawStepAction{playerID: a.PlayerID()}
	if g.effects.ApplyReplacements(action, g) != action || len(g.turns.pendingStepLifeGains) != 0 {
		t.Fatal("declined replacement skipped the step")
	}
}

func TestStaticDrawStepReplacementAmountAndMultiplicity(t *testing.T) {
	for _, amount := range []int{0, 2, 5} {
		a := &acceptingMayPlayer{NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		for range 2 {
			g.PutOnBattlefield(NewEnchantment("Static skip", "", WithAbility(Static(SkipDrawStepForLife(amount)))), a.PlayerID())
		}
		action := &beginDrawStepAction{playerID: a.PlayerID()}
		if g.effects.ApplyReplacements(action, g) != nil {
			t.Fatal("accepted replacement did not skip")
		}
		if len(g.turns.pendingStepLifeGains) != 1 {
			t.Fatal("one skipped step produced multiple gains")
		}
		g.RunStepWithPriority(PrecombatMain)
		if a.Life() != 20+amount {
			t.Fatalf("life = %d, want %d", a.Life(), 20+amount)
		}
	}
}

func TestStaticDrawStepReplacementNegativeAmount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("negative amount accepted")
		}
	}()
	SkipDrawStepForLife(-1)
}
