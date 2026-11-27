// Package mtgish compiles mtgish JSON rules into engine card definitions.
package mtgish

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// Compile validates all card rules before it returns a shared definition.
// Unsupported rules and fields cause an error with the card name and source path.
func Compile(card MtgishCard) (*mage.CardDefinition, error) {
	spec, err := card.ToCardDefinitionSpec()
	if err != nil {
		return nil, err
	}
	for i, raw := range card.Rules {
		ability, err := compileAbility(raw, indexPath("/Rules", i))
		if err != nil {
			return nil, withCard(err, card.Name)
		}
		spec.Abilities = append(spec.Abilities, ability)
	}
	if slices.Contains(spec.TypeLine.CardTypes, core.TypeLand) {
		seen := make(map[core.Color]bool)
		for _, subtype := range spec.TypeLine.SubTypes {
			color, ok := basicLandColors[subtype]
			if ok && !seen[color] {
				spec.Abilities = append(spec.Abilities, mage.NewManaAbility(color))
				seen[color] = true
			}
		}
	}
	definition, err := mage.NewCardDefinition(spec)
	if err != nil {
		return nil, withCard(err, card.Name)
	}
	return definition, nil
}

type taggedNode struct {
	Family string
	Tag    string
	Args   json.RawMessage
}

func decodeTaggedNode(raw json.RawMessage, family, path string) (taggedNode, error) {
	fields, err := objectFields(raw, path)
	if err != nil {
		return taggedNode{}, err
	}
	node := taggedNode{Family: family, Args: fields["args"]}
	_ = json.Unmarshal(fields[family], &node.Tag)
	if err := checkFields(fields, path, family, "args"); err != nil {
		return taggedNode{}, withNode(err, node)
	}
	if err := decodeValue(fields[family], pointerPath(path, family), &node.Tag); err != nil {
		return taggedNode{}, withNode(err, node)
	}
	if node.Tag == "" {
		return taggedNode{}, diagnostic(pointerPath(path, family), InvalidInput, family, "", "expected a nonempty tag")
	}
	return node, nil
}

func withNode(err error, node taggedNode) error {
	if source, ok := errors.AsType[*CompileError](err); ok {
		result := *source
		result.Family, result.Tag = node.Family, node.Tag
		return &result
	}
	return err
}

func compileAbility(raw json.RawMessage, path string) (mage.Ability, error) {
	if err := validateJSONDepth(raw, path, 2); err != nil {
		return nil, err
	}
	node, err := decodeTaggedNode(raw, "_Rule", path)
	if err != nil {
		return nil, err
	}
	keyword, ok := keywords[node.Tag]
	if !ok {
		return nil, diagnostic(path, UnsupportedConstruct, node.Family, node.Tag, fmt.Sprintf("unsupported rule %q", node.Tag))
	}
	if len(node.Args) != 0 {
		return nil, diagnostic(pointerPath(path, "args"), InvalidInput, node.Family, node.Tag, "keyword does not accept args")
	}
	return mage.NewAttributeAbility(keyword), nil
}

var keywords = map[string]core.Attr{
	"Flying": core.Flying, "FirstStrike": core.FirstStrike, "DoubleStrike": core.DoubleStrike,
	"Deathtouch": core.Deathtouch, "Defender": core.Defender, "Flash": core.Flash,
	"Haste": core.Haste, "Hexproof": core.Hexproof, "Indestructible": core.Indestructible,
	"Lifelink": core.Lifelink, "Menace": core.Menace, "Reach": core.Reach,
	"Shroud": core.Shroud, "Trample": core.Trample, "Vigilance": core.Vigilance,
	"Banding": core.Banding, "Fear": core.Fear,
}

var basicLandColors = map[string]core.Color{
	"Plains": core.White, "Island": core.Blue, "Swamp": core.Black, "Mountain": core.Red, "Forest": core.Green,
}
