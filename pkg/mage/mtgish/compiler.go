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
	hasSpellActions := false
	for i, raw := range card.Rules {
		ability, err := compileRule(raw, indexPath("/Rules", i), hasSpellActions)
		if err != nil {
			return nil, withCard(err, card.Name)
		}
		spec.Abilities = append(spec.Abilities, ability)
		hasSpellActions = hasSpellActions || ability.Type() == core.AbilitySpell
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
	path   string
}

func decodeTaggedNode(raw json.RawMessage, family, path string) (taggedNode, error) {
	fields, err := objectFields(raw, path)
	if err != nil {
		return taggedNode{}, err
	}
	node := taggedNode{Family: family, Args: fields["args"], path: path}
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

func (n taggedNode) argsPath() string {
	return pointerPath(n.path, "args")
}

func (n taggedNode) errorAt(path string, code DiagnosticCode, message string) error {
	return diagnostic(path, code, n.Family, n.Tag, message)
}

func (n taggedNode) unsupported() error {
	return n.errorAt(n.path, UnsupportedConstruct, fmt.Sprintf("unsupported %s tag %q", n.Family, n.Tag))
}

func (n taggedNode) requireArgs() error {
	if len(n.Args) == 0 || isNull(n.Args) {
		return n.errorAt(n.argsPath(), InvalidInput, "required value is missing or null")
	}
	return nil
}

func (n taggedNode) rejectArgs() error {
	if len(n.Args) != 0 {
		return n.errorAt(n.argsPath(), InvalidInput, "tag does not accept args")
	}
	return nil
}

func (n taggedNode) decodeArgs(value any) error {
	if err := decodeValue(n.Args, n.argsPath(), value); err != nil {
		return withNode(err, n)
	}
	return nil
}

func (n taggedNode) tuple(size int) ([]json.RawMessage, error) {
	var args []json.RawMessage
	if err := n.decodeArgs(&args); err != nil {
		return nil, err
	}
	if len(args) != size {
		return nil, n.errorAt(n.argsPath(), InvalidInput, fmt.Sprintf("expected %d arguments, got %d", size, len(args)))
	}
	return args, nil
}

func requireNullaryTag(raw json.RawMessage, family, tag, path string) error {
	node, err := decodeTaggedNode(raw, family, path)
	if err != nil {
		return err
	}
	if node.Tag != tag {
		return node.unsupported()
	}
	return node.rejectArgs()
}

func compileRule(raw json.RawMessage, path string, hasSpellActions bool) (mage.Ability, error) {
	if err := validateJSONDepth(raw, path, 2); err != nil {
		return nil, err
	}
	node, err := decodeTaggedNode(raw, "_Rule", path)
	if err != nil {
		return nil, err
	}
	if keyword, ok := keywords[node.Tag]; ok {
		if len(node.Args) != 0 {
			return nil, node.errorAt(node.argsPath(), InvalidInput, "keyword does not accept args")
		}
		return mage.NewAttributeAbility(keyword), nil
	}

	switch node.Tag {
	case "SpellActions":
		if hasSpellActions {
			return nil, node.errorAt(path, UnsupportedConstruct, "multiple SpellActions sections")
		}
		if err := node.requireArgs(); err != nil {
			return nil, err
		}
		return compileActionDefinition(node.Args, node.argsPath(), mage.ActionSpell, nil)
	case "Activated":
		args, err := node.tuple(2)
		if err != nil {
			return nil, err
		}
		cost, err := compileCost(args[0], indexPath(node.argsPath(), 0))
		if err != nil {
			return nil, err
		}
		return compileActionDefinition(args[1], indexPath(node.argsPath(), 1), mage.ActionActivated, cost)
	default:
		return nil, node.errorAt(path, UnsupportedConstruct, fmt.Sprintf("unsupported rule %q", node.Tag))
	}
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
