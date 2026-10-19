package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type castSpellChoicePlan struct{ name string }

func compileCastSpellChoice(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "cast history requires an ordinary resolving body")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "cast spell filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "supply one card type and one caster")
	}
	var cardType core.CardType
	var caster *playerReferencePlan
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		switch part.Name {
		case "IsCardtype":
			if cardType != 0 {
				return nil, problem(part, "duplicate card type")
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "Sorcery" {
				return nil, unsupported(part, "cast spell type")
			}
			cardType = core.TypeSorcery
		case "CastByPlayer":
			if caster != nil {
				return nil, problem(part, "duplicate caster")
			}
			plan, err := compilePlayerRef(part.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			caster = &plan
		default:
			return nil, unsupported(part, "cast spell filter")
		}
	}
	if cardType == 0 || caster == nil {
		return nil, problem(filter, "supply one card type and one caster")
	}
	plan := &castSpellChoicePlan{name: "spell cast " + n.Position.String()}
	ctx.chosenCastSpell = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		binding := mage.BindDamageSource(plan.name)
		bindings.chosenCastSpells = maps.Clone(bindings.chosenCastSpells)
		if bindings.chosenCastSpells == nil {
			bindings.chosenCastSpells = make(map[*castSpellChoicePlan]*mage.DamageSourceBinding)
		}
		bindings.chosenCastSpells[plan] = binding
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ChooseSpellCastThisTurn(ctx.player.build(bindings), caster.build(bindings), mage.DamageSourceHasType(cardType), binding))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
func compileCastSpellDamageValue(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	ref := n.Args[0]
	if ref.Kind != Symbol || ref.Name != "TheSpellCastChosenThisWay" || ctx.chosenCastSpell == nil || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(ref, "unbound or incompatible cast spell result")
	}
	plan := ctx.chosenCastSpell
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.DamageDealtThisTurnBySource(mage.FromDamageSource(bindings.chosenCastSpells[plan]))
	}, nil
}
