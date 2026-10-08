package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// TokenDefinitionFactory builds a token declaration without saved game state.
type TokenDefinitionFactory func() TokenDefinition

// NewTokenDefinitionFactory checks fresh token rules and retains copy provenance.
// Each created token needs its own abilities and bindings.
func NewTokenDefinitionFactory(build TokenDefinitionFactory) (TokenDefinitionFactory, error) {
	plan, err := newCardDefinitionPlan(func() (Card, error) {
		d, err := buildTokenDeclaration(build)
		if err != nil {
			return nil, err
		}
		return tokenDeclarationCard(d), nil
	})
	if err != nil {
		return nil, err
	}
	return func() TokenDefinition {
		d, err := buildTokenDeclaration(build)
		if err != nil {
			return &tokenDefinition{problem: err.Error()}
		}
		copyDefinition := *d
		copyDefinition.plan = plan
		return &copyDefinition
	}, nil
}

func buildTokenDeclaration(build TokenDefinitionFactory) (d *tokenDefinition, err error) {
	defer func() {
		if problem := recover(); problem != nil {
			d, err = nil, fmt.Errorf("token definition builder: %v", problem)
		}
	}()
	if build == nil {
		return nil, fmt.Errorf("token definition requires a builder")
	}
	d, ok := build().(*tokenDefinition)
	if !ok || d == nil || d.plan != nil {
		return nil, fmt.Errorf("token definition requires an unplanned declaration")
	}
	check := NewInstant("Token declaration", "", CreateTokenCopies(You, ValueAtResolution(Fixed(1)), d))
	if err := ValidateCardDefinition(check.Name(), check); err != nil {
		return nil, err
	}
	return d, nil
}

func tokenDeclarationCard(d *tokenDefinition) *BaseCard {
	card := NewToken(d.name, d.power, d.toughness, d.types, d.subtypes, d.keywords...)
	card.colorOverride = slices.Clone(d.colors)
	for _, ability := range d.abilities {
		card.AddAbility(ability)
	}
	return card
}

func createDefinitionTokenBatch(g *Game, player uuid.UUID, count int, d *tokenDefinition) ([]*Permanent, error) {
	if d.problem != "" {
		return nil, fmt.Errorf("token definition: %s", d.problem)
	}
	if d.plan == nil {
		return createTokenBatchWithResult(g, player, count, d.name, d.power, d.toughness, d.types, d.subtypes, d.colors, d.keywords, d.abilities), nil
	}
	cards := make([]Card, 0, count)
	for range count {
		card, err := d.plan.Build()
		if err != nil {
			return nil, fmt.Errorf("token definition: %w", err)
		}
		card.SetOwner(player)
		cards = append(cards, card)
	}
	return g.putTokensOnBattlefieldTogether(cards, player), nil
}
