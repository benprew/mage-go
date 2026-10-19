package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithPrintedCardDefinitionPlan retains the physical card's printed rules.
// Copy prototypes use a separate plan for their battlefield characteristics.
func WithPrintedCardDefinitionPlan(plan *CardDefinitionPlan) CardOption {
	return func(card *BaseCard) { card.printedDefinitionPlan = plan }
}

// BuildEntryCopy builds independent copy rules for a physical card's next entry.
// The caller must supply entry permission, copy choices and replacement order.
// The original card must leave its former zone before the result enters.
func BuildEntryCopy(original Card, values *CopiableValues) (Card, error) {
	definition := original
	if copied, ok := original.(permanentSpellCopyCard); ok {
		definition = copied.Card
	}
	base, ok := definition.(*BaseCard)
	if !ok || base == nil || base.printedDefinitionPlan == nil {
		return nil, fmt.Errorf("entry copy requires a physical printed definition plan")
	}
	if original.ID() == uuid.Nil {
		return nil, fmt.Errorf("entry copy requires a physical card identity")
	}
	characteristics := base.textEntryValues
	if characteristics == nil && base.copiableValues != nil {
		characteristics = &base.copiableValues.printed
	}
	if !isEntryCopyPermanent(original) || base.copiableValues == nil || characteristics == nil || !characteristics.matches(base) || base.definitionPlan != base.copiableValues.definition {
		return nil, fmt.Errorf("entry copy requires tracked permanent characteristics")
	}
	if base.departureCard == nil && base.printedDefinitionPlan != base.definitionPlan {
		return nil, fmt.Errorf("entry copy requires matching physical definition provenance")
	}
	if base.departureCard != nil {
		physical, ok := base.departureCard.(*BaseCard)
		if !ok || physical.definitionPlan != base.printedDefinitionPlan || !base.printedDefinitionPlan.copiableValues.printed.matches(physical) {
			return nil, fmt.Errorf("entry copy requires tracked physical definition provenance")
		}
	}
	physical, err := base.printedDefinitionPlan.Build()
	if err != nil {
		return nil, fmt.Errorf("entry copy physical definition: %w", err)
	}
	copied, err := values.Build()
	if err != nil {
		return nil, fmt.Errorf("entry copy values: %w", err)
	}
	if !isEntryCopyPermanent(copied) {
		return nil, fmt.Errorf("entry copy values require a permanent definition")
	}
	result := copied.(*BaseCard)
	physical.SetID(original.ID())
	physical.SetOwner(original.Owner())
	physical.(*BaseCard).isToken = original.IsToken()
	result.id = original.ID()
	result.owner = original.Owner()
	result.isToken = original.IsToken()
	result.requiresAnte = physical.(*BaseCard).requiresAnte
	result.printedDefinitionPlan = base.printedDefinitionPlan
	result.departureCard = physical
	return result, nil
}

func isEntryCopyPermanent(card Card) bool {
	return card.HasType(core.TypeCreature) || card.HasType(core.TypeArtifact) || card.HasType(core.TypeEnchantment) || card.HasType(core.TypeLand) || card.HasType(core.TypePlaneswalker)
}

func cardForBattlefieldDeparture(card Card) Card {
	if original := faceDownOriginal(card); original != nil {
		return departureCardIdentity(card, cardForBattlefieldDeparture(original))
	}
	if base, ok := card.(*BaseCard); ok && base != nil && base.departureCard != nil {
		return departureCardIdentity(card, base.departureCard)
	}
	return card
}

func departureCardIdentity(subject, physical Card) Card {
	if subject.ID() == physical.ID() && subject.Owner() == physical.Owner() {
		return physical
	}
	result := physical.Copy()
	result.SetID(subject.ID())
	result.SetOwner(subject.Owner())
	return result
}
