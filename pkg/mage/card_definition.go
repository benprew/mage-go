package mage

import (
	"fmt"
	"reflect"
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TypeLine contains the printed types of a card.
type TypeLine struct {
	SuperTypes []SuperType
	CardTypes  []CardType
	SubTypes   []string
}

// CardDefinitionSpec supplies printed characteristics and compiled rules.
// It must not contain parser nodes or source-format instructions.
type CardDefinitionSpec struct {
	Name                     string
	TypeLine                 TypeLine
	ManaCost                 ManaCost
	Power, Toughness         int
	Colors                   []Color
	Abilities                []Ability
	AdditionalCosts          []Cost
	AlternateCosts           []AlternateCost
	CastTargets              []Target
	Uncounterable            bool
	AuraAIProfile            *AuraAIProfile
	FlipFace                 *CardDefinition
	OracleText               string
	BaseLoyalty              int
	PowerStar, ToughnessStar bool
}

// CardDefinition owns immutable printed characteristics and compiled rules.
// It has no card identity, owner, selected targets, or ability execution state.
type CardDefinition struct {
	cardCharacteristics

	abilities                []Ability
	additionalCosts          []Cost
	alternateCosts           []AlternateCost
	castTargets              []Target
	uncounterable            bool
	auraAIProfile            *AuraAIProfile
	flipFace                 *CardDefinition
	oracleText               string
	baseLoyalty              int
	powerStar, toughnessStar bool
}

// NewCardDefinition copies the supplied data before publication.
func NewCardDefinition(spec CardDefinitionSpec) (*CardDefinition, error) {
	if spec.Name == "" || len(spec.TypeLine.CardTypes) == 0 {
		return nil, fmt.Errorf("card definition requires a name and card types")
	}
	for i, ability := range spec.Abilities {
		if ability == nil || (reflect.ValueOf(ability).Kind() == reflect.Pointer && reflect.ValueOf(ability).IsNil()) {
			return nil, fmt.Errorf("card %q: ability %d is nil", spec.Name, i)
		}
	}

	spec.TypeLine.SuperTypes = slices.Clone(spec.TypeLine.SuperTypes)
	spec.TypeLine.CardTypes = slices.Clone(spec.TypeLine.CardTypes)
	spec.TypeLine.SubTypes = slices.Clone(spec.TypeLine.SubTypes)
	spec.ManaCost = copyPrintedManaCost(spec.ManaCost)
	spec.Colors = slices.Clone(spec.Colors)
	spec.Abilities = slices.Clone(spec.Abilities)
	d := &CardDefinition{
		name: spec.Name, manaCost: spec.ManaCost, types: spec.TypeLine.CardTypes, superTypes: spec.TypeLine.SuperTypes, subTypes: spec.TypeLine.SubTypes, power: spec.Power, toughness: spec.Toughness, colorOverride: spec.Colors,
		abilities: spec.Abilities, additionalCosts: slices.Clone(spec.AdditionalCosts), alternateCosts: cloneAlternateCosts(spec.AlternateCosts), castTargets: slices.Clone(spec.CastTargets), uncounterable: spec.Uncounterable,
		oracleText: spec.OracleText, baseLoyalty: spec.BaseLoyalty, powerStar: spec.PowerStar, toughnessStar: spec.ToughnessStar,
	}
	if spec.AuraAIProfile != nil {
		profile := *spec.AuraAIProfile
		d.auraAIProfile = &profile
	}
	if spec.FlipFace != nil {
		d.flipFace = spec.FlipFace
	}
	return d, nil
}

func copyPrintedManaCost(cost ManaCost) ManaCost {
	cost.Hybrid = slices.Clone(cost.Hybrid)
	return cost
}

func (d *CardDefinition) Name() string            { return d.name }
func (d *CardDefinition) ManaCost() ManaCost      { return copyPrintedManaCost(d.manaCost) }
func (d *CardDefinition) Types() []CardType       { return slices.Clone(d.types) }
func (d *CardDefinition) SuperTypes() []SuperType { return slices.Clone(d.superTypes) }
func (d *CardDefinition) SubTypes() []string      { return slices.Clone(d.subTypes) }
func (d *CardDefinition) Power() int              { return d.power }
func (d *CardDefinition) Toughness() int          { return d.toughness }
func (d *CardDefinition) Colors() []Color {
	if d.colorOverride != nil {
		return slices.Clone(d.colorOverride)
	}
	return d.manaCost.Colors()
}
func (d *CardDefinition) Modes() []string {
	for _, ability := range d.abilities {
		if action, ok := ability.(*ActionDefinition); ok && action.kind == ActionSpell && len(action.modes) > 0 {
			return modalLabels(action)
		}
	}
	return nil
}

// NewInstance creates a card with a new identity and the supplied owner.
func (d *CardDefinition) NewInstance(owner uuid.UUID) *Card {
	c := NewCard(d)
	c.SetOwner(owner)
	return c
}

// TypeLine returns a private copy of the complete printed type line.
func (d *CardDefinition) TypeLine() TypeLine {
	return TypeLine{SuperTypes: d.SuperTypes(), CardTypes: d.Types(), SubTypes: d.SubTypes()}
}

func (d *CardDefinition) cloneDefinition() *CardDefinition {
	cloned := *d
	cloned.cardCharacteristics = *d.clone()
	cloned.abilities = slices.Clone(d.abilities)

	cloned.additionalCosts = slices.Clone(d.additionalCosts)
	cloned.alternateCosts = cloneAlternateCosts(d.alternateCosts)
	cloned.castTargets = slices.Clone(d.castTargets)
	if d.auraAIProfile != nil {
		profile := *d.auraAIProfile
		cloned.auraAIProfile = &profile
	}
	return &cloned
}

// Abilities returns the immutable ability specifications in a private slice.
func (d *CardDefinition) Abilities() []Ability { return slices.Clone(d.abilities) }

// AdditionalCosts returns a private list of additional casting costs.
func (d *CardDefinition) AdditionalCosts() []Cost { return slices.Clone(d.additionalCosts) }

// AlternateCosts returns a private list of alternate casting costs.
func (d *CardDefinition) AlternateCosts() []AlternateCost {
	return cloneAlternateCosts(d.alternateCosts)
}

// CastTargets returns a private list of cast target specifications.
func (d *CardDefinition) CastTargets() []Target { return slices.Clone(d.castTargets) }

// NewCardFromDefinition creates an owned instance of a shared printed face.
func NewCardFromDefinition(definition *CardDefinition, owner uuid.UUID) *Card {
	return definition.NewInstance(owner)
}

func (d *CardDefinition) OracleText() string        { return d.oracleText }
func (d *CardDefinition) BaseLoyalty() int          { return d.baseLoyalty }
func (d *CardDefinition) PowerIsStar() bool         { return d.powerStar }
func (d *CardDefinition) ToughnessIsStar() bool     { return d.toughnessStar }
func (d *CardDefinition) FlipFace() *CardDefinition { return d.flipFace }
func (d *CardDefinition) IsUncounterable() bool     { return d.uncounterable }
