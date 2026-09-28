package dsl_test

import (
	"testing"

	dsl "github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDamagePlayerRecipientPublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("recipient")
	_ = dsl.WhenEvent(dsl.DealsDamage(dsl.This, dsl.BindNumber("amount")).WithRecipientPlayer(player),
		dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)))
}
