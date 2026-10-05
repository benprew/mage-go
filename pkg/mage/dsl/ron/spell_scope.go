package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

type spellTargetPlan struct{ name string }

type spellReferencePlan struct {
	target *spellTargetPlan
	event  *castEventPlan
}

func compileSpellRef(n Node, ctx scope) (spellReferencePlan, error) {
	if n.Kind == Symbol && ctx.castEvent != nil && n.Name == ctx.castEvent.spellName {
		// The engine does not save event spell bindings for delayed effects.
		if ctx.delayedStep {
			return spellReferencePlan{}, unsupported(n, "delayed spell-event reference")
		}
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
