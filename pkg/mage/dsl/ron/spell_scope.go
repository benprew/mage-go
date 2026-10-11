package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type spellTargetPlan struct{ name string }

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

func compileSpellFilter(n Node) (mage.CardFilter, error) {
	if n.Kind == Symbol && n.Name == "AnySpell" {
		return mage.NewCardFilter("any spell", func(mage.Card) bool { return true }), nil
	}
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	switch n.Name {
	case "IsEnchantmentType":
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "Aura" {
			return mage.CardFilter{}, unsupported(n.Args[0], "spell enchantment subtype")
		}
		return mage.NewCardFilter("Aura spell", func(card mage.Card) bool {
			return card.HasType(core.TypeEnchantment) && card.HasSubType("Aura")
		}), nil
	case "IsCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		return mage.NewCardFilter(kind.String()+" spell", func(card mage.Card) bool { return card.HasType(kind) }), nil
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
			filters[i], err = compileSpellFilter(part)
			if err != nil {
				return mage.CardFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.AndCardFilter(filters...), nil
		}
		return mage.NewCardFilter("matching spell", func(card mage.Card) bool {
			return slices.ContainsFunc(filters, func(filter mage.CardFilter) bool { return filter.Match(card) })
		}), nil
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
	manaValueX := false
	var filters []mage.SpellFilter
	for _, part := range parts {
		if part.Name == "ManaValueIs" {
			if manaValueX {
				return nil, "", problem(part, "supply one spell mana value restriction")
			}
			if err := compileManaValueXRestriction(part, ctx); err != nil {
				return nil, "", err
			}
			manaValueX = true
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
		filter, err := compileContextualSpellFilter(part)
		if err != nil {
			return nil, "", err
		}
		filters = append(filters, filter)
	}
	if controlled && manaValueX {
		return nil, "", problem(n, "combined spell mana value and permanent target restrictions are not supported")
	}
	if len(filters) == 0 {
		filters = append(filters, mage.SpellCardMatches(mage.CardFilter{}))
	}
	return func(instanceBindings) mage.Target {
		name := ctx.targetNamespace + "Ref_TargetSpell"
		if manaValueX {
			return mage.DeclareSpellTargetManaValueX(name).WithFilter(mage.AllSpellFilters(filters...))
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
