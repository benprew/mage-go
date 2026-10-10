package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CopiableValues keeps characteristics and a builder for independent rules.
// It contains no source object, owner, or saved game choices.
type CopiableValues struct {
	definition *CardDefinitionPlan
	printed    copiableCharacteristics
	faceDown   bool
}

type copiableCharacteristics struct {
	name          string
	manaCost      core.ManaCost
	hasManaCost   bool
	types         []core.CardType
	superTypes    []core.SuperType
	subTypes      []string
	power         int
	toughness     int
	colorOverride []core.Color
	attrSeeds     map[core.Attr]int
	modes         []string
}

func printedCopiableCharacteristics(card *BaseCard) copiableCharacteristics {
	cost := card.manaCost
	cost.Hybrid = slices.Clone(cost.Hybrid)
	return copiableCharacteristics{
		name: card.name, manaCost: cost, hasManaCost: card.hasManaCost,
		types: slices.Clone(card.types), superTypes: slices.Clone(card.superTypes),
		subTypes: slices.Clone(card.subTypes), power: card.power, toughness: card.toughness,
		colorOverride: slices.Clone(card.colorOverride), attrSeeds: maps.Clone(card.attrSeeds),
		modes: slices.Clone(card.modes),
	}
}

func (v copiableCharacteristics) matches(card *BaseCard) bool {
	cost, other := v.manaCost, card.manaCost
	return v.name == card.name && v.power == card.power && v.toughness == card.toughness &&
		v.hasManaCost == card.hasManaCost &&
		slices.Equal(v.types, card.types) && slices.Equal(v.superTypes, card.superTypes) &&
		slices.Equal(v.subTypes, card.subTypes) && (v.colorOverride == nil) == (card.colorOverride == nil) && slices.Equal(v.colorOverride, card.colorOverride) &&
		maps.Equal(v.attrSeeds, card.attrSeeds) && slices.Equal(v.modes, card.modes) &&
		cost.Generic == other.Generic && cost.White == other.White && cost.Blue == other.Blue &&
		cost.Black == other.Black && cost.Red == other.Red && cost.Green == other.Green &&
		cost.HasX == other.HasX && cost.XCount == other.XCount && slices.Equal(cost.Hybrid, other.Hybrid)
}

// CopiableValues returns the plan's immutable printed characteristics.
func (p *CardDefinitionPlan) CopiableValues() *CopiableValues {
	if p == nil {
		return nil
	}
	return p.copiableValues
}

// HasManaCost separates an absent cost from a printed zero cost.
func (v *CopiableValues) HasManaCost() bool { return v != nil && v.printed.hasManaCost }

// WithCopiableValues retains known copy provenance on a constructed prototype.
// Capture checks the characteristics before it uses this provenance.
func WithCopiableValues(values *CopiableValues) CardOption {
	return func(card *BaseCard) { card.copiableValues = values }
}

// Build creates an independent face-up prototype from the saved values.
// The caller must supply ownership and the new object's lifecycle.
func (v *CopiableValues) Build() (Card, error) {
	if v == nil {
		return nil, fmt.Errorf("copiable values require a snapshot")
	}
	if v.definition != nil {
		card, err := v.definition.Build()
		if err != nil {
			return nil, err
		}
		card.(*BaseCard).isToken = false
		return card, nil
	}
	if !v.faceDown {
		return nil, fmt.Errorf("copiable values require validated provenance")
	}
	card := NewCreature("", "", 2, 2)
	card.copiableValues = v
	return card, nil
}

// CaptureCopiableValues reads one present incarnation without executing rules.
// Unknown copy layers cannot supply accurate provenance and must return an error.
func (g *Game) CaptureCopiableValues(source *Permanent) (*CopiableValues, error) {
	if g == nil || source == nil || nilDefinition(source.Card) {
		return nil, fmt.Errorf("copiable values require a present permanent")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID {
		return nil, fmt.Errorf("copiable values require the original present incarnation")
	}
	if current.FaceDown {
		return &CopiableValues{faceDown: true, printed: copiableCharacteristics{types: []core.CardType{core.TypeCreature}, power: 2, toughness: 2}}, nil
	}
	for _, effect := range g.effects.effects {
		if effect.GetLayer() == core.LayerCopy && effect.IsActive(g) {
			return nil, fmt.Errorf("copiable values cannot read an untracked copy layer")
		}
	}
	definition := current.Card
	if copied, ok := definition.(permanentSpellCopyCard); ok {
		definition = copied.Card
	}
	card, ok := definition.(*BaseCard)
	if !ok || card == nil || card.copiableValues == nil {
		return nil, fmt.Errorf("copiable values require retained definition provenance")
	}
	characteristics := &card.copiableValues.printed
	if card.textEntryValues != nil {
		characteristics = card.textEntryValues
	}
	if !characteristics.matches(card) {
		return nil, fmt.Errorf("copiable values cannot read untracked characteristic changes")
	}
	if card.definitionPlan != card.copiableValues.definition {
		return nil, fmt.Errorf("copiable values require matching definition provenance")
	}
	return card.copiableValues, nil
}
