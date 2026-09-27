package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDelayedDeathAuthorSurface(t *testing.T) {
	target := dsl.TargetPermanent("watched", dsl.IsCreature)
	body := dsl.WhenDiesThisTurn(target, dsl.GainLife(dsl.You, dsl.Fixed(3)))
	card := dsl.NewSorcery("Watch", "", dsl.Spell(body, dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(body).Role != "delayed-event" {
		t.Fatal("missing delayed event descriptor")
	}
}

func TestDelayedLeaveAuthorSurface(t *testing.T) {
	watched := dsl.TargetPermanent("watched", dsl.IsCreature)
	body := dsl.WhenLeavesThisTurn(watched, dsl.GainLife(dsl.You, dsl.Fixed(3)))
	card := dsl.NewSorcery("Watch", "", dsl.Spell(body, dsl.WithTargets(watched)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := mage.InspectDefinition(body); meta.Role != "delayed-event" || meta.Text != "when that permanent leaves the battlefield this turn" {
		t.Fatalf("unexpected descriptor: %+v", meta)
	}
}
