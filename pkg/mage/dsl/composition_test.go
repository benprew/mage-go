package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedResolutionComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	member := dsl.BindPermanent("member")
	effect := dsl.FlipCoin(dsl.You,
		dsl.OnWin(dsl.ForEach(dsl.ControlledPermanents(dsl.IsCreature), member,
			dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.From(member), dsl.Fixed(1), dsl.Fixed(1))),
		)),
		dsl.OnLoss(dsl.UnlessPaid(dsl.ControllerOf(target), dsl.Costs(dsl.Mana("{1}")), dsl.Destroy(target))),
	)
	card := mage.NewCreature("Composition", "", 1, 1, mage.WithAbility(dsl.Activate(
		dsl.Costs(), effect, dsl.WithTargets(target), dsl.OnlyAsSorcery(), dsl.ActivationLimit(1),
	)))
	if err := mage.ValidateCardDefinition("Composition", card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(effect)
	if metadata.Role != "coin-flip" || !metadata.Properties.Destroys || !metadata.Properties.Mass {
		t.Fatalf("composition metadata lost: %+v", metadata)
	}
}
