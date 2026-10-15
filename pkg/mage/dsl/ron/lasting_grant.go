package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileLastingAbilityGrants(n Node, ctx scope, subject permanentReferencePlan, parts []Node) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "lasting ability grants require an ordinary resolving body")
	}
	var builds []effectFactory
	for _, part := range parts {
		if part.Name == "LosesAbility" {
			if ctx.graveyardEnchant && subject.kind == permanentSource {
				return nil, unsupported(part, "graveyard enchant removal")
			}
			build, err := compileCreatureEnchantRemoval(part, subject)
			if err != nil {
				return nil, err
			}
			builds = append(builds, func(bindings instanceBindings) mage.Effect { return mage.Until(core.Indefinite, build(bindings)) })
			continue
		}
		if part.Name != "AddAbility" {
			return nil, unsupported(part, "lasting ability modification")
		}
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		abilities, err := list(part.Args[0])
		if err != nil {
			return nil, err
		}
		if len(abilities) == 0 {
			return nil, problem(part, "supply at least one ability")
		}
		for _, ability := range abilities {
			switch ability.Name {
			case "TriggerA":
				build, err := compileLastingUpkeepGrant(ability, ctx, subject)
				if err != nil {
					return nil, err
				}
				builds = append(builds, build)
			case "If":
				if err := arity(ability, 2); err != nil {
					return nil, err
				}
				recipientScope := scope{permanent: true}
				condition, err := compileStaticCondition(ability.Args[0], recipientScope)
				if err != nil {
					return nil, err
				}
				rules, err := list(ability.Args[1])
				if err != nil {
					return nil, err
				}
				if len(rules) != 1 || rules[0].Name != "PermanentRuleEffect" {
					return nil, unsupported(ability.Args[1], "acquired conditional untap body")
				}
				ref, err := compileUntapSubject(rules[0], recipientScope)
				if err != nil {
					return nil, err
				}
				builds = append(builds, func(bindings instanceBindings) mage.Effect {
					return mage.AcquireStaticAbility(subject.build(bindings), mage.PreventUntapDuringStep(ref, condition))
				})
			default:
				return nil, unsupported(ability, "lasting ability grant")
			}
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(builds))
		for i, build := range builds {
			effects[i] = build(bindings)
		}
		return mage.Sequence(effects...)
	}, nil
}
