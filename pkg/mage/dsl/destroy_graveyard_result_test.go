package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPermanentsPutIntoGraveyardThisWayPublicDeclaration(t *testing.T) {
	count := dsl.BindNumber("graveyard count")
	targets := dsl.TargetPermanentsExactlyX("mountains", dsl.And(dsl.IsLand, dsl.HasSubType("Mountain")))
	card := dsl.NewSorcery("Synthetic eruption", "{X}{U}{U}{U}", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.DestroyEach(targets, dsl.PermanentsPutIntoGraveyardThisWayInto(count)),
		dsl.DealDamageEach(dsl.This, dsl.DamageRecipients(dsl.EachPlayer, dsl.MatchingPermanents(dsl.IsCreature)), dsl.FromNumber(count)),
	), dsl.WithTargets(targets))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
