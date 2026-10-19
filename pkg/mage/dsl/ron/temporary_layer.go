package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTemporaryLayer(n Node, ctx scope) (effectFactory, error) {
	if containsDeclarationName(n, "PermanentRuleEffect") {
		subject, err := compileAttackRestrictionGrant(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.AcquireStaticAbilityUntil(subject.build(bindings), mage.CantAttack(mage.This), core.EndOfCombat)
		}, nil
	}

	if len(n.Args) == 3 && n.Args[2].Name == "UntilPermanentLeavesBattlefield" {
		return compileSourceLandType(n, ctx)
	}
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	duration, err := compileTemporaryLayerDuration(n.Args[2], ctx, subject)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	var modifications []func(instanceBindings) mage.ContinuousModification
	var adjustments []mage.PTAdjustment
	var valueAdjustments []func(instanceBindings) mage.PTValueAdjustment
	var keywordGrants []core.Attr
	var rampageGrants []int
	onlyRampage := true
	onlyKeywords := true
	onlyMixed := true
	for _, part := range parts {
		if part.Name != "AddAbility" && part.Name != "AdjustPT" && part.Name != "AdjustPTX" {
			onlyMixed = false
		}
		if part.Name != "AddAbility" {
			onlyRampage = false
			onlyKeywords = false
		}
		if part.Name == "SetLandType" && n.Args[2].Name == "ForAsLongAsPermanentHasACounterOfTypeOnIt" {
			build, err := compileBasicLandTypeModification(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "SwitchPT" {
			if _, err := symbol(part); err != nil {
				return nil, err
			}
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				return mage.SwitchPowerToughness(subject.build(bindings))
			})
			continue
		}
		if part.Name == "AddAbilityVariable" {
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "TheChosenAbility" || ctx.chosenAbility == nil {
				return nil, problem(part.Args[0], "unbound or incompatible chosen ability")
			}
			ability := *ctx.chosenAbility
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				return ability.modification(subject.build(bindings))
			})
			continue
		}
		if part.Name == "SetPower" {
			build, err := compileTemporaryPower(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "LosesAbility" {
			build, err := compileTemporaryKeywordRemoval(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "SetCardtypes" {
			builds, err := compileTemporaryArtifactCreature(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, builds...)
			continue
		}
		if part.Name == "SetPowerAndToughnessBoth" {
			build, err := compileTemporaryEqualPT(part, ctx, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "SetPT" || part.Name == "AddCardtype" || part.Name == "AddCreatureType" || part.Name == "SetCreatureType" {
			var build func(instanceBindings) mage.ContinuousModification
			var err error
			if part.Name == "SetPT" && part.Kind == Tuple && len(part.Args) == 1 && part.Args[0].Name == "ManualPT" {
				build, err = compileTemporaryManualPT(part.Args[0], ctx, subject)
			} else {
				build, err = compileTemporaryType(part, subject)
			}
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "AddAbility" {
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			abilities, err := list(part.Args[0])
			if err != nil {
				return nil, err
			}
			for _, node := range abilities {
				ability, err := compileTemporaryAbility(node)
				if err != nil {
					return nil, err
				}
				if ability.isRampage {
					rampageGrants = append(rampageGrants, ability.rampage)
				} else {
					onlyRampage = false
				}
				if ability.keyword == 0 {
					onlyMixed = false
					onlyKeywords = false
				} else {
					keywordGrants = append(keywordGrants, ability.keyword)
				}
				modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
					return ability.modification(subject.build(bindings))
				})
			}
			continue
		}
		if part.Name == "AdjustPTX" {
			build, err := compileTemporaryPTAdjustment(part, ctx, subject)
			if err != nil {
				return nil, err
			}
			valueAdjustments = append(valueAdjustments, build)
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				value := build(bindings)
				return mage.ModifyPT(subject.build(bindings), value.Power, value.Toughness)
			})
			continue
		}
		if part.Name != "AdjustPT" {
			return nil, unsupported(part, "temporary layer action")
		}
		power, toughness, err := compilePT(part)
		if err != nil {
			return nil, err
		}
		adjustments = append(adjustments, mage.PTAdjustment{Power: power, Toughness: toughness})
		valueAdjustments = append(valueAdjustments, func(_ instanceBindings) mage.PTValueAdjustment {
			return mage.PTValueAdjustment{Power: mage.Fixed(power), Toughness: mage.Fixed(toughness)}
		})
		modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
			return mage.ModifyPT(subject.build(bindings), mage.Fixed(power), mage.Fixed(toughness))
		})
	}
	if len(modifications) == 0 {
		return nil, problem(n.Args[1], "supply at least one temporary modification")
	}
	return func(bindings instanceBindings) mage.Effect {
		if !ctx.permanent && onlyRampage && n.Args[2].Name == "UntilEndOfTurn" {
			return (mage.TemporaryRampageInstruction{Subject: subject.build(bindings), Amounts: rampageGrants}).Effect()
		}
		if !ctx.permanent && onlyKeywords && n.Args[2].Name == "UntilEndOfTurn" {
			return (mage.TemporaryKeywordInstruction{Subject: subject.build(bindings), Keywords: keywordGrants}).Effect()
		}
		if !ctx.permanent && len(adjustments) == len(parts) && n.Args[2].Name == "UntilEndOfTurn" {
			return (mage.TemporaryPTInstruction{Subject: subject.build(bindings), Adjustments: adjustments}).Effect()
		}
		if !ctx.permanent && len(valueAdjustments) == len(parts) && n.Args[2].Name == "UntilEndOfTurn" {
			values := make([]mage.PTValueAdjustment, len(valueAdjustments))
			for i, build := range valueAdjustments {
				values[i] = build(bindings)
			}
			return (mage.TemporaryPTValueInstruction{Subject: subject.build(bindings), Adjustments: values}).Effect()
		}
		if !ctx.permanent && onlyMixed && len(keywordGrants) > 0 && len(valueAdjustments) > 0 && n.Args[2].Name == "UntilEndOfTurn" {
			values := make([]mage.PTValueAdjustment, len(valueAdjustments))
			for i, build := range valueAdjustments {
				values[i] = build(bindings)
			}
			return (mage.TemporaryMixedLayerInstruction{Subject: subject.build(bindings), Keywords: keywordGrants, Adjustments: values}).Effect()
		}
		values := make([]mage.ContinuousModification, len(modifications))
		for i, build := range modifications {
			values[i] = build(bindings)
		}
		return duration(bindings, values)
	}, nil
}
