package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCombinedTimingComposition(t *testing.T) {
	step, turn := dsl.OnlyDuring(core.Upkeep), dsl.OnlyOnYourTurn()
	body := dsl.Regenerate(dsl.This)
	card := dsl.NewArtifact("Timed abilities", "",
		dsl.WithAbility(dsl.Activate(dsl.Costs(), body, step, turn)),
		dsl.WithAbility(dsl.ActivateModes(dsl.Costs(), dsl.ModeOf("one", body), dsl.ModeOf("two", body), turn, step)),
		dsl.WithAbility(dsl.Static(dsl.GrantActivatedModes(dsl.This, dsl.Costs(), dsl.ModeOf("one", body), dsl.ModeOf("two", body), step, turn))),
		dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Until(core.Indefinite, dsl.ActivatedAbilityModification(dsl.This, dsl.Costs(), body, turn, step)))),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
