package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type spellTargetPlan struct {
	name             string
	instantOrSorcery bool
}

type spellReferencePlan struct {
	target *spellTargetPlan
	event  *castEventPlan
}

func compileSpellRef(n Node, ctx scope) (spellReferencePlan, error) {
	if n.Kind == Symbol && ctx.castEvent != nil && n.Name == ctx.castEvent.spellName {
		return spellReferencePlan{event: ctx.castEvent}, nil
	}
	if n.Kind == Symbol && ctx.spellTarget != nil && n.Name == ctx.spellTarget.name {
		return spellReferencePlan{target: ctx.spellTarget}, nil
	}
	return spellReferencePlan{}, problem(n, "unbound or incompatible spell reference")
}

func (p spellReferencePlan) build(bindings instanceBindings) mage.StackObjectRef {
	if p.event != nil {
		return mage.FromSpell(bindings.events[p.event].spell)
	}
	return bindings.spells[p.target]
}

func compileSpellManaValue(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	spell, err := compileSpellRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.ManaValueOfSpellWithLKI(spell.build(bindings))
	}, nil
}

func compileSpellFilter(n Node) (mage.CardFilter, error) {
	return compileSpellCardFilter(n, false)
}

func compileSpellCardFilter(n Node, targeting bool) (mage.CardFilter, error) {
	if n.Kind == Symbol && n.Name == "AnySpell" {
		return mage.AnyCardFilter(), nil
	}
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	switch n.Name {
	case "CanTargetOnly":
		if !targeting {
			return mage.CardFilter{}, unsupported(n, "spell characteristic filter")
		}
		return compileTargetOnlySubtypeFilter(n)
	case "IsEnchantmentType":
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "Aura" {
			return mage.CardFilter{}, unsupported(n.Args[0], "spell enchantment subtype")
		}
		return mage.AndCardFilter(mage.HasCardTypeCardFilter(core.TypeEnchantment), mage.HasSubTypeCardFilter("Aura")), nil
	case "IsCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		return mage.HasCardTypeCardFilter(kind), nil
	case "IsColor":
		value, err := color(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		return mage.HasColorCardFilter(value), nil
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		if len(parts) == 0 {
			return mage.CardFilter{}, problem(n, "spell filter requires at least one operand")
		}
		filters := make([]mage.CardFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileSpellCardFilter(part, targeting)
			if err != nil {
				return mage.CardFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.AndCardFilter(filters...), nil
		}
		return mage.OrCardFilter(filters...), nil
	default:
		return mage.CardFilter{}, unsupported(n, "spell target filter")
	}
}

func compileSpellTarget(n Node, ctx scope) (targetFactory, string, error) {
	parts := []Node{n}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, "", err
		}
		var err error
		parts, err = list(n.Args[0])
		if err != nil {
			return nil, "", err
		}
		if len(parts) == 0 {
			return nil, "", problem(n, "spell filter requires at least one operand")
		}
	}
	controlled := false
	var manaValueX *manaValueXTargetPlan
	var filters []mage.SpellFilter
	for _, part := range parts {
		if part.Name == "ManaValueIs" {
			if manaValueX != nil {
				return nil, "", problem(part, "supply one spell mana value restriction")
			}
			plan, err := compileManaValueXRestriction(part, ctx)
			if err != nil {
				return nil, "", err
			}
			manaValueX = plan
			continue
		}
		if part.Name == "TargetsAPermanent" {
			if controlled {
				return nil, "", problem(part, "supply one permanent target restriction")
			}
			if err := compileControlledPermanentSpellRestriction(part); err != nil {
				return nil, "", err
			}
			controlled = true
			continue
		}
		filter, err := compileContextualSpellFilter(part, ctx)
		if err != nil {
			return nil, "", err
		}
		filters = append(filters, filter)
	}
	if controlled && manaValueX != nil {
		return nil, "", problem(n, "combined spell mana value and permanent target restrictions are not supported")
	}
	if len(filters) == 0 {
		filters = append(filters, mage.SpellCardMatches(mage.CardFilter{}))
	}
	return func(instanceBindings) mage.Target {
		name := ctx.targetNamespace + "Ref_TargetSpell"
		if manaValueX != nil {
			return manaValueX.spellTarget(name).WithFilter(mage.AllSpellFilters(filters...))
		}
		if controlled {
			return mage.DeclareSpellTargetTargetingControlledPermanent(name).WithFilter(mage.AllSpellFilters(filters...))
		}
		return mage.DeclareSpellTarget(name).WithFilter(mage.AllSpellFilters(filters...))
	}, "Ref_TargetSpell", nil
}

func compileControlledPermanentSpellRestriction(n Node) error {
	for _, name := range []string{"TargetsAPermanent", "ControlledByAPlayer", "SinglePlayer"} {
		if n.Name != name {
			return unsupported(n, "controlled permanent spell restriction")
		}
		if err := arity(n, 1); err != nil {
			return err
		}
		n = n.Args[0]
	}
	if n.Kind != Symbol || n.Name != "You" {
		return unsupported(n, "spell target permanent controller")
	}
	return nil
}
