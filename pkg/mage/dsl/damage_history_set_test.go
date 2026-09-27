package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDamageHistorySetDeclaration(t *testing.T) {
	players := dsl.PlayersDealtDamageBy(dsl.This, dsl.OpponentsOf(dsl.You))
	planeswalkers := dsl.PermanentsDealtDamageBy(dsl.This, dsl.MatchingPermanents(dsl.IsPlaneswalker))
	card := dsl.NewCreature("History source", "", 2, 3, dsl.WithAbility(dsl.At(dsl.YourUpkeep,
		dsl.DealDamageEach(dsl.This, dsl.DamageRecipients(players, planeswalkers), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
