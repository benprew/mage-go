package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func isResolutionActionPayment(n Node) bool {
	if n.Name != "MayCost" && n.Name != "PlayerMayCost" {
		return false
	}
	if len(n.Args) == 0 {
		return false
	}
	switch n.Args[len(n.Args)-1].Name {
	case "GainLife", "HavePermanentDealDamage", "DestroyPermanent", "CreatePermanentLayerEffectUntil":
		return true
	}
	return false
}

func compileResolutionActionPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	count := 1
	if payment.Name == "PlayerMayCost" {
		count = 2
	}
	if err := arity(payment, count); err != nil {
		return nil, err
	}
	payer := playerReferencePlan{}
	if count == 2 {
		var err error
		payer, err = compilePlayerRef(payment.Args[0], ctx)
		if err != nil {
			return nil, err
		}
	}
	cost := payment.Args[count-1]
	buildPayment, err := compileResolutionPaymentAction(cost, ctx)
	if err != nil {
		return nil, err
	}
	branchCount := 2
	if continuation.Name == "IfElse" {
		branchCount = 3
	} else if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, problem(continuation, "action payment requires an adjacent result continuation")
	}
	if err := arity(continuation, branchCount); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid" {
		return nil, unsupported(condition, "action payment condition")
	}
	branches := make([][]effectFactory, branchCount-1)
	for i := range branches {
		parts, err := list(continuation.Args[i+1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(continuation.Args[i+1], "action payment branch requires at least one action")
		}
		branches[i], err = compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		built := make([]mage.ResolvingEffect, 2)
		for i, branch := range branches {
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			built[i] = mage.Sequence(effects...)
		}
		if (continuation.Name == "Unless") == (condition.Name == "CostWasPaid") {
			built[0], built[1] = built[1], built[0]
		}
		return mage.PayActionOrElse(payer.build(bindings), buildPayment(bindings), built[0], built[1])
	}, nil
}

func compileResolutionPaymentAction(cost Node, ctx scope) (func(instanceBindings) mage.ResolutionActionPayment, error) {
	var buildPayment func(instanceBindings) mage.ResolutionActionPayment
	switch cost.Name {
	case "CreatePermanentLayerEffectUntil":
		subject, err := compileAttackRestrictionGrant(cost, ctx)
		if err != nil {
			return nil, err
		}
		buildPayment = func(bindings instanceBindings) mage.ResolutionActionPayment {
			return mage.StaticAbilityPayment(subject.build(bindings), mage.CantAttack(mage.This), core.EndOfCombat)
		}
	case "GainLife":
		if err := arity(cost, 1); err != nil {
			return nil, err
		}
		amount, err := compileInstanceValue(cost.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		buildPayment = func(bindings instanceBindings) mage.ResolutionActionPayment {
			return mage.GainLifePayment(ctx.player.build(bindings), amount(bindings))
		}
	case "HavePermanentDealDamage":
		if err := arity(cost, 3); err != nil {
			return nil, err
		}
		source, err := compilePermanentRef(cost.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		amount, err := compileInstanceValue(cost.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		recipient, err := compileDamagePacketRecipient(cost.Args[2], ctx)
		if err != nil {
			return nil, err
		}
		buildPayment = func(bindings instanceBindings) mage.ResolutionActionPayment {
			return mage.DamagePayment(mage.DamageSourceOf(source.build(bindings)), recipient(bindings), amount(bindings))
		}
	case "DestroyPermanent":
		if err := arity(cost, 1); err != nil {
			return nil, err
		}
		subject, err := compilePermanentRef(cost.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		buildPayment = func(bindings instanceBindings) mage.ResolutionActionPayment {
			return mage.DestroyPayment(subject.build(bindings))
		}
	default:
		return nil, unsupported(cost, "resolution action payment")
	}
	return buildPayment, nil
}
