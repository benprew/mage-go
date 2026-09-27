package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCombatOpponentsAuthorSurface(t *testing.T) {
	var card dsl.Card = dsl.NewCreature("Combat Watcher", "{1}{G}", 2, 4,
		dsl.WithAbility(dsl.WhenEvent(dsl.BlocksOrBecomesBlockedBy(dsl.This),
			dsl.Until(dsl.EndOfTurn,
				dsl.GrantEachKeyword(dsl.BlockingOrBlockedBy(dsl.This), dsl.FirstStrike)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestBlockerGroupWithoutSubtypeAuthorSurface(t *testing.T) {
	var card dsl.Card = dsl.NewCreature("Group watcher", "{1}{G}", 2, 4,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.BlocksOrBecomesBlockedBy(dsl.This).WithoutSubtype("Wall"),
			dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
