package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TypeLine contains the printed types of a card.
type TypeLine struct {
	SuperTypes []SuperType
	CardTypes  []CardType
	SubTypes   []string
}

// RuleDefinition creates private execution state for one printed rule.
// The function must return a new ability and all of its mutable data on each call.
// It must not capture mutable state. Effects, costs, and callback data must be
// immutable. Put execution state on the game or source object.
// Source parsers must validate and compile their input before they create a rule.
type RuleDefinition struct{ instantiate func() Ability }

// NewRuleDefinition defines a rule with a function that creates its execution state.
func NewRuleDefinition(instantiate func() Ability) RuleDefinition {
	return RuleDefinition{instantiate: instantiate}
}

// CardDefinitionSpec supplies printed characteristics and compiled rules.
// It must not contain parser nodes or source-format instructions.
type CardDefinitionSpec struct {
	Name             string
	TypeLine         TypeLine
	ManaCost         ManaCost
	Power, Toughness int
	Colors           []Color
	Keywords         map[Attr]int
	Modes            []string
	Rules            []RuleDefinition
}

// CardDefinition owns immutable printed characteristics and compiled rules.
// It has no card identity, owner, selected targets, or ability execution state.
type CardDefinition struct{ printed CardDefinitionSpec }

// NewCardDefinition copies the supplied data before publication.
func NewCardDefinition(spec CardDefinitionSpec) (*CardDefinition, error) {
	if spec.Name == "" || len(spec.TypeLine.CardTypes) == 0 {
		return nil, fmt.Errorf("card definition requires a name and card types")
	}
	for i, rule := range spec.Rules {
		if err := validateRuleDefinition(rule); err != nil {
			return nil, fmt.Errorf("card %q: rule %d: %w", spec.Name, i, err)
		}
	}
	spec.TypeLine.SuperTypes = slices.Clone(spec.TypeLine.SuperTypes)
	spec.TypeLine.CardTypes = slices.Clone(spec.TypeLine.CardTypes)
	spec.TypeLine.SubTypes = slices.Clone(spec.TypeLine.SubTypes)
	spec.ManaCost = copyPrintedManaCost(spec.ManaCost)
	spec.Colors = slices.Clone(spec.Colors)
	spec.Keywords = maps.Clone(spec.Keywords)
	spec.Modes = slices.Clone(spec.Modes)
	spec.Rules = slices.Clone(spec.Rules)
	return &CardDefinition{printed: spec}, nil
}

func copyPrintedManaCost(cost ManaCost) ManaCost {
	cost.Hybrid = slices.Clone(cost.Hybrid)
	return cost
}

func (d *CardDefinition) Name() string            { return d.printed.Name }
func (d *CardDefinition) ManaCost() ManaCost      { return copyPrintedManaCost(d.printed.ManaCost) }
func (d *CardDefinition) Types() []CardType       { return slices.Clone(d.printed.TypeLine.CardTypes) }
func (d *CardDefinition) SuperTypes() []SuperType { return slices.Clone(d.printed.TypeLine.SuperTypes) }
func (d *CardDefinition) SubTypes() []string      { return slices.Clone(d.printed.TypeLine.SubTypes) }
func (d *CardDefinition) Power() int              { return d.printed.Power }
func (d *CardDefinition) Toughness() int          { return d.printed.Toughness }
func (d *CardDefinition) Colors() []Color         { return slices.Clone(d.printed.Colors) }
func (d *CardDefinition) Modes() []string         { return slices.Clone(d.printed.Modes) }

// NewInstance creates a card with a new identity and private ability state.
func (d *CardDefinition) NewInstance(owner uuid.UUID) *Card {
	c := NewCard(d)
	c.SetOwner(owner)
	return c
}

func validateRuleDefinition(rule RuleDefinition) (err error) {
	defer func() {
		if reason := recover(); reason != nil {
			err = fmt.Errorf("invalid rule: %v", reason)
		}
	}()
	if rule.instantiate == nil {
		return fmt.Errorf("rule has no implementation")
	}
	first, second := rule.instantiate(), rule.instantiate()
	if first == nil || second == nil {
		return fmt.Errorf("rule returned a nil ability")
	}
	if first == second {
		return fmt.Errorf("rule reuses mutable ability state")
	}
	cloneInstanceAbility(first)
	return nil
}

// TypeLine returns a private copy of the complete printed type line.
func (d *CardDefinition) TypeLine() TypeLine {
	return TypeLine{SuperTypes: d.SuperTypes(), CardTypes: d.Types(), SubTypes: d.SubTypes()}
}
