package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type chosenHandCardPlan struct{ name string }

func compileChosenHandCardRef(n Node, ctx scope) (func(instanceBindings) mage.CardRef, error) {
	if n.Kind != Symbol || n.Name != "TheCardInHandChosenThisWay" || ctx.chosenHandCard == nil {
		return nil, problem(n, "chosen hand card requires a successful choice branch")
	}
	plan := ctx.chosenHandCard
	return func(bindings instanceBindings) mage.CardRef { return mage.FromCard(bindings.chosenHandCards[plan]) }, nil
}

func compileOptionalHandChoice(payment, continuation Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(payment, "hand choice context")
	}
	count := 1
	if payment.Name == "PlayerMayCost" {
		count = 2
	}
	if err := arity(payment, count); err != nil {
		return nil, err
	}
	payer := Node{Kind: Symbol, Name: "You"}
	if count == 2 {
		payer = payment.Args[0]
	}
	player, err := compilePlayerRef(payer, ctx)
	if err != nil {
		return nil, err
	}
	choice := payment.Args[count-1]
	if err := arity(choice, 1); err != nil {
		return nil, err
	}
	var receipt bool
	var filter mage.CardFilter
	if choice.Args[0].Name == "AnyCard" {
		if choice.Args[0].Kind != Symbol {
			return nil, problem(choice.Args[0], "AnyCard requires a symbol")
		}
	} else {
		filter, receipt, err = compileActivationManaHandFilter(choice.Args[0], ctx)
		if err != nil {
			return nil, err
		}
	}
	count = 2
	if continuation.Name == "IfElse" {
		count = 3
	} else if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, unsupported(continuation, "hand choice continuation")
	}
	if err := arity(continuation, count); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid" {
		return nil, unsupported(condition, "hand choice payment condition")
	}
	firstPaid := (continuation.Name != "Unless") == (condition.Name == "CostWasPaid")
	plan := &chosenHandCardPlan{name: "chosen hand card " + choice.Position.String()}
	branches := make([][]effectFactory, 2)
	for i, branch := range continuation.Args[1:] {
		parts, err := list(branch)
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(branch, "hand choice branch requires at least one action")
		}
		branchCtx := ctx
		if (i == 0) == firstPaid {
			branchCtx.chosenHandCard = plan
		} else {
			branchCtx.chosenHandCard = nil
		}
		branches[i], err = compileActionSequence(parts, branchCtx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.chosenHandCards = maps.Clone(bindings.chosenHandCards)
		if bindings.chosenHandCards == nil {
			bindings.chosenHandCards = make(map[*chosenHandCardPlan]*mage.CardBinding)
		}
		result := mage.BindCard(plan.name)
		bindings.chosenHandCards[plan] = result
		bodies := make([]mage.ResolvingEffect, 2)
		for i, branch := range branches {
			if len(branch) == 0 {
				continue
			}
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			bodies[i] = mage.Sequence(effects...)
		}
		if !firstPaid {
			bodies[0], bodies[1] = bodies[1], bodies[0]
		}
		if receipt {
			return mage.ChooseHandCardWithActivationManaOrElse(player.build(bindings), filter, bindings.activationMana, result, bodies[0], bodies[1])
		}
		return mage.ChooseHandCardOrElse(player.build(bindings), filter, result, bodies[0], bodies[1])
	}, nil
}
