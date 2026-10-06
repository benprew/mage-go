package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTargetedColor(n Node, ctx scope) (optionFactory, error) {
	if ctx.permanent || ctx.castingCondition != nil || ctx.castingCost != nil {
		return nil, unsupported(n, "spell/permanent color context")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	branches, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(branches) != 2 {
		return nil, problem(n, "supply one spell branch and one permanent branch")
	}
	var colors [2]core.Color
	seen := make(map[string]bool)
	for i, branch := range branches {
		if branch.Name != "Targeted" {
			return nil, unsupported(branch, "spell/permanent color branch")
		}
		if err := arity(branch, 2); err != nil {
			return nil, err
		}
		targets, err := list(branch.Args[0])
		if err != nil {
			return nil, err
		}
		if len(targets) != 1 {
			return nil, problem(branch.Args[0], "supply one target per branch")
		}
		target := targets[0]
		if err := arity(target, 1); err != nil {
			return nil, err
		}
		var ref, action, filter string
		switch target.Name {
		case "TargetSpell":
			ref, action, filter = "Ref_TargetSpell", "CreateSpellEffect", "AnySpell"
		case "TargetPermanent":
			ref, action, filter = "Ref_TargetPermanent", "CreatePermanentLayerEffect", "AnyPermanent"
		default:
			return nil, unsupported(target, "spell/permanent color target")
		}
		if seen[target.Name] {
			return nil, problem(target, "duplicate target category")
		}
		seen[target.Name] = true
		if target.Args[0].Kind != Symbol || target.Args[0].Name != filter {
			return nil, unsupported(target.Args[0], "spell/permanent color filter")
		}
		body := branch.Args[1]
		if body.Name != "ActionList" {
			return nil, unsupported(body, "spell/permanent color body")
		}
		if err := arity(body, 1); err != nil {
			return nil, err
		}
		actions, err := list(body.Args[0])
		if err != nil {
			return nil, err
		}
		if len(actions) != 1 || actions[0].Name != action {
			return nil, problem(body, "supply one color action for this target category")
		}
		op := actions[0]
		if err := arity(op, 2); err != nil {
			return nil, err
		}
		if op.Args[0].Kind != Symbol || op.Args[0].Name != ref {
			return nil, problem(op.Args[0], "unbound or incompatible color target reference")
		}
		parts, err := list(op.Args[1])
		if err != nil {
			return nil, err
		}
		if len(parts) != 1 || parts[0].Name != "SetColor" {
			return nil, problem(op.Args[1], "supply one color modification")
		}
		values, err := compileColorList(parts[0])
		if err != nil {
			return nil, err
		}
		if len(values) != 1 {
			return nil, problem(parts[0], "supply one actual color")
		}
		colors[i] = values[0]
	}
	if colors[0] != colors[1] {
		return nil, problem(n, "target categories must use the same color")
	}
	return func() mage.CardOption {
		target := mage.DeclareSpellOrPermanentTarget("spell or permanent")
		return mage.WithAbility(mage.Spell(mage.SetObjectColor(target, colors[0]), mage.WithTargets(target)))
	}, nil
}

func compileColorList(n Node) ([]core.Color, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if value.Name != "SimpleColorList" {
		return nil, unsupported(value, "color list")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	nodes, err := list(value.Args[0])
	if err != nil {
		return nil, err
	}
	colors := make([]core.Color, len(nodes))
	for i, node := range nodes {
		colors[i], err = color(node)
		if err != nil {
			return nil, err
		}
		if colors[i] == core.Colorless {
			return nil, problem(node, "supply one actual color")
		}
	}
	return colors, nil
}
