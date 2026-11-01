// Package mtgish compiles mtgish JSON rules into engine card definitions.
package mtgish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// Compile validates all card rules before it returns a shared definition.
// Unsupported rules and fields cause an error with the card name and source path.
func Compile(card mage.MtgishCard) (*mage.CardDefinition, error) {
	if err := validateFields(card); err != nil {
		return nil, err
	}
	spec, err := card.ToCardDefinitionSpec()
	if err != nil {
		return nil, fmt.Errorf("mtgish card %q: %w", card.Name, err)
	}
	for i, raw := range card.Rules {
		ability, err := compileAbility(raw)
		if err != nil {
			return nil, fmt.Errorf("mtgish card %q: Rules[%d]: %w", card.Name, i, err)
		}
		spec.Rules = append(spec.Rules, ability)
	}
	if slices.Contains(spec.TypeLine.CardTypes, core.TypeLand) {
		seen := make(map[core.Color]bool)
		for _, subtype := range spec.TypeLine.SubTypes {
			color, ok := basicLandColors[subtype]
			if ok && !seen[color] {
				spec.Rules = append(spec.Rules, mage.NewRuleDefinition(func() mage.Ability { return mage.NewManaAbility(color) }))
				seen[color] = true
			}
		}
	}
	return mage.NewCardDefinition(spec)
}

func compileAbility(raw json.RawMessage) (mage.RuleDefinition, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return mage.RuleDefinition{}, fmt.Errorf("expected a rule object: %w", err)
	}
	var tag string
	if err := json.Unmarshal(fields["_Rule"], &tag); err != nil || tag == "" {
		return mage.RuleDefinition{}, fmt.Errorf("rule requires a string _Rule tag")
	}
	keyword, ok := keywords[tag]
	if !ok {
		return mage.RuleDefinition{}, fmt.Errorf("unsupported rule %q", tag)
	}
	if len(fields) != 1 {
		return mage.RuleDefinition{}, fmt.Errorf("rule %q does not accept arguments or other fields", tag)
	}
	return mage.NewRuleDefinition(func() mage.Ability { return mage.NewAttributeAbility(keyword) }), nil
}

var keywords = map[string]core.Attr{
	"Flying": core.Flying, "FirstStrike": core.FirstStrike, "DoubleStrike": core.DoubleStrike,
	"Deathtouch": core.Deathtouch, "Defender": core.Defender, "Flash": core.Flash,
	"Haste": core.Haste, "Hexproof": core.Hexproof, "Indestructible": core.Indestructible,
	"Lifelink": core.Lifelink, "Menace": core.Menace, "Reach": core.Reach,
	"Shroud": core.Shroud, "Trample": core.Trample, "Vigilance": core.Vigilance,
	"Banding": core.Banding, "Fear": core.Fear,
}

func validateFields(card mage.MtgishCard) error {
	if len(card.Raw) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(card.Raw))
	decoder.DisallowUnknownFields()
	var source mage.MtgishCard
	if err := decoder.Decode(&source); err != nil {
		return fmt.Errorf("mtgish card %q: invalid source record: %w", card.Name, err)
	}
	if source.Kind == "" {
		return fmt.Errorf("mtgish card %q: source record requires _OracleCard", card.Name)
	}
	return nil
}

var basicLandColors = map[string]core.Color{
	"Plains": core.White, "Island": core.Blue, "Swamp": core.Black, "Mountain": core.Red, "Forest": core.Green,
}
