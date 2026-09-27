package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnyDamageSourcePublicDeclaration(t *testing.T) {
	dealt := dsl.BindNumber("dealt")
	source := dsl.BindDamageSource("source")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.DealsDamage(dsl.This, dealt).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.You, dsl.FromNumber(dealt)))))
	if card == nil {
		t.Fatal("missing card")
	}
}
