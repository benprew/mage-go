package mtgish

import (
	"encoding/json"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type compileScope struct {
	kind           mage.ActionKind
	bindsAnyTarget bool
}

func compileActionDefinition(raw json.RawMessage, path string, kind mage.ActionKind, cost mage.Cost) (*mage.ActionDefinition, error) {
	target, effects, err := compileActions(raw, path, compileScope{kind: kind})
	if err != nil {
		return nil, err
	}
	options := []mage.ActionOption{mage.WithEffects(effects...)}
	if target != nil {
		options = append(options, mage.WithTarget(target))
	}
	if cost != nil {
		options = append(options, mage.WithCost(cost))
	}
	return mage.NewAction(kind, options...), nil
}

func compileActions(raw json.RawMessage, path string, scope compileScope) (mage.Target, []mage.Effect, error) {
	node, err := decodeTaggedNode(raw, "_Actions", path)
	if err != nil {
		return nil, nil, err
	}
	switch node.Tag {
	case "ActionList":
		var actions []json.RawMessage
		if err := node.decodeArgs(&actions); err != nil {
			return nil, nil, err
		}
		var effects []mage.Effect
		for i, action := range actions {
			lowered, err := compileAction(action, indexPath(node.argsPath(), i), scope)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, lowered...)
		}
		return nil, effects, nil
	case "Targeted":
		if scope.bindsAnyTarget {
			return nil, nil, node.errorAt(path, UnsupportedConstruct, "nested targeting")
		}
		args, err := node.tuple(2)
		if err != nil {
			return nil, nil, err
		}
		targetPath := indexPath(node.argsPath(), 0)
		var targets []json.RawMessage
		if err := decodeValue(args[0], targetPath, &targets); err != nil {
			return nil, nil, withNode(err, node)
		}
		if len(targets) == 0 {
			return nil, nil, node.errorAt(targetPath, InvalidInput, "expected at least one target")
		}
		if len(targets) > 1 {
			return nil, nil, node.errorAt(targetPath, UnsupportedConstruct, "only one target group is supported")
		}
		if err := requireNullaryTag(targets[0], "_Target", "AnyTarget", indexPath(targetPath, 0)); err != nil {
			return nil, nil, err
		}
		scope.bindsAnyTarget = true
		_, effects, err := compileActions(args[1], indexPath(node.argsPath(), 1), scope)
		if err != nil {
			return nil, nil, err
		}
		return mage.TargetDamageAnyTarget(), effects, nil
	default:
		return nil, nil, node.unsupported()
	}
}

func compileCost(raw json.RawMessage, path string) (mage.Cost, error) {
	node, err := decodeTaggedNode(raw, "_Cost", path)
	if err != nil {
		return nil, err
	}
	if node.Tag != "TapPermanent" {
		return nil, node.unsupported()
	}
	if err := node.requireArgs(); err != nil {
		return nil, err
	}
	if err := requireNullaryTag(node.Args, "_Permanent", "ThisPermanent", node.argsPath()); err != nil {
		return nil, err
	}
	return mage.Tap(), nil
}

func compileAction(raw json.RawMessage, path string, scope compileScope) ([]mage.Effect, error) {
	node, err := decodeTaggedNode(raw, "_Action", path)
	if err != nil {
		return nil, err
	}
	switch node.Tag {
	case "SpellDealsDamage", "PermanentDealsDamage":
		effect, err := compileDamage(node, scope)
		if err != nil {
			return nil, err
		}
		return []mage.Effect{effect}, nil
	case "AddMana":
		if err := node.requireArgs(); err != nil {
			return nil, err
		}
		productions, err := compileFixedMana(node.Args, node.argsPath(), nil)
		if err != nil {
			return nil, err
		}
		effects := make([]mage.Effect, 0, len(productions))
		for _, production := range productions {
			effects = append(effects, mage.AddMana(production.Color, production.Amount))
		}
		return effects, nil
	default:
		return nil, node.unsupported()
	}
}

func compileDamage(node taggedNode, scope compileScope) (mage.Effect, error) {
	args, err := node.tuple(3)
	if err != nil {
		return nil, err
	}
	sourceFamily, sourceTag, sourceKind := "_Spell", "ThisSpell", mage.ActionSpell
	if node.Tag == "PermanentDealsDamage" {
		sourceFamily, sourceTag, sourceKind = "_Permanent", "ThisPermanent", mage.ActionActivated
	}
	sourcePath := indexPath(node.argsPath(), 0)
	if err := requireNullaryTag(args[0], sourceFamily, sourceTag, sourcePath); err != nil {
		return nil, err
	}
	if scope.kind != sourceKind {
		return nil, diagnostic(sourcePath, InvalidInput, sourceFamily, sourceTag, "source reference does not match the enclosing ability")
	}
	amount, err := compileInteger(args[1], indexPath(node.argsPath(), 1))
	if err != nil {
		return nil, err
	}
	recipientPath := indexPath(node.argsPath(), 2)
	if err := requireNullaryTag(args[2], "_DamageRecipient", "Ref_AnyTarget", recipientPath); err != nil {
		return nil, err
	}
	if !scope.bindsAnyTarget {
		return nil, diagnostic(recipientPath, InvalidInput, "_DamageRecipient", "Ref_AnyTarget", "no AnyTarget group in this ability")
	}
	return mage.DealDamageFrom(mage.SourceObject(), mage.TargetGroup(0), amount), nil
}

func compileInteger(raw json.RawMessage, path string) (mage.ValueSource, error) {
	node, err := decodeTaggedNode(raw, "_GameNumber", path)
	if err != nil {
		return nil, err
	}
	if node.Tag != "Integer" {
		return nil, node.unsupported()
	}
	var amount int32
	if err := node.decodeArgs(&amount); err != nil {
		return nil, err
	}
	return mage.Fixed(int(amount)), nil
}

var fixedManaColors = map[string]core.Color{
	"ManaProduceW": core.White, "ManaProduceU": core.Blue, "ManaProduceB": core.Black,
	"ManaProduceR": core.Red, "ManaProduceG": core.Green, "ManaProduceC": core.Colorless,
}

func compileFixedMana(raw json.RawMessage, path string, productions []mage.ManaProduction) ([]mage.ManaProduction, error) {
	node, err := decodeTaggedNode(raw, "_ManaProduce", path)
	if err != nil {
		return nil, err
	}
	if node.Tag == "And" {
		var symbols []json.RawMessage
		if err := node.decodeArgs(&symbols); err != nil {
			return nil, err
		}
		for i, symbol := range symbols {
			productions, err = compileFixedMana(symbol, indexPath(node.argsPath(), i), productions)
			if err != nil {
				return nil, err
			}
		}
		return productions, nil
	}
	color, ok := fixedManaColors[node.Tag]
	if !ok {
		return nil, node.unsupported()
	}
	if err := node.rejectArgs(); err != nil {
		return nil, err
	}
	if last := len(productions) - 1; last >= 0 && productions[last].Color == color {
		productions[last].Amount++
		return productions, nil
	}
	return append(productions, mage.ManaProduction{Color: color, Amount: 1}), nil
}
