package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestLifePaymentBranchesRestrictions(t *testing.T) {
	for _, amount := range []int{0, 10} {
		a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		g.effects.Rules.AddCantLoseLife(a.PlayerID())
		effect := PayOrElse(You, Costs(LifePayCost(amount)), GainLifeFor(ActivePlayer, Fixed(3)), GainLifeFor(ActivePlayer, Fixed(7)))
		if err := ValidateCardDefinition("Life branches", NewInstant("Life branches", "", Spell(effect))); err != nil {
			t.Fatal(err)
		}
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 27
		if amount == 0 {
			want = 23
		}
		if a.Life() != want || b.Life() != 20 {
			t.Fatal("life restriction or zero payment", amount, a.Life(), b.Life())
		}
	}
}

func TestLifePaymentBranchesReplacement(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		g.AddPreventionShield(a.PlayerID(), 100)
		g.AddReplacementEffect(&testPrioritizedReplacement{
			matchFunc: func(action Action) bool { _, ok := action.(*LifeLossAction); return ok },
			replaceFunc: func(action Action, _ *Game) Action {
				if cancel {
					return nil
				}
				return action
			},
		})
		effect := PayOrElse(You, Costs(LifePayCost(10)), GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(7)))
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 13
		if cancel {
			want = 23
		}
		if a.Life() != want || b.Life() != 20 {
			t.Fatal("replacement changed payment acceptance or damage prevention stopped life payment", a.Life(), b.Life())
		}
	}
}
