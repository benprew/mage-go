package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileDepartureDelay(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "delayed departure requires an ordinary resolving body")
	}
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	duration, err := symbol(n.Args[2])
	if err != nil {
		return nil, err
	}
	if duration == "UntilEndOfCombat" && n.Args[0].Name == "Or" {
		return compileCombatPairDelay(n, ctx)
	}
	if duration != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "delayed departure duration")
	}
	event := n.Args[0]
	if event.Name == "WhenAPlayerTapsAPermanentForMana" {
		return compileSubtypeManaBonusUntil(n)
	}
	if event.Name == "WhenAPermanentDealsDamageToAPermanent" {
		return compileDelayedDamageCapture(n, ctx)
	}
	if event.Name == "WhenACreatureAttacksAndIsntBlocked" {
		return compileUnblockedDelay(n, ctx)
	}
	if event.Name != "WhenAPermanentDies" && event.Name != "WhenAPermanentLeavesTheBattlefield" {
		return nil, unsupported(event, "delayed departure event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "delayed departure subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "delayed departure body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed departure requires at least one action")
	}
	ctx.delayedStep = true
	ctx.drawReplacementBody = false
	ctx.delayedDeath = nil
	if event.Name == "WhenAPermanentDies" {
		ctx.delayedDeath = &hostDeathPlan{name: "Trigger_ThatPermanent", watched: &ref}
		ctx.deathCard = &deathCardPlan{name: "Trigger_ThatGraveyardCard"}
	}
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		watched := ref.build(bindings)
		if ctx.delayedDeath != nil {
			bindings.hostDeaths = maps.Clone(bindings.hostDeaths)
			if bindings.hostDeaths == nil {
				bindings.hostDeaths = make(map[*hostDeathPlan]hostDeathBindings)
			}
			bindings.deathCards = maps.Clone(bindings.deathCards)
			if bindings.deathCards == nil {
				bindings.deathCards = make(map[*deathCardPlan]*mage.CardBinding)
			}
			label := "DelayedDeath@" + n.Position.String()
			bindings.deathCards[ctx.deathCard] = mage.BindCard(label + ".Card")
			bindings.hostDeaths[ctx.delayedDeath] = hostDeathBindings{
				permanent:  mage.BindPermanent(label + ".Permanent"),
				owner:      mage.BindPlayer(label + ".Owner"),
				controller: mage.BindPlayer(label + ".Controller"),
			}
		}
		values := make([]mage.Effect, len(effects))
		for i, build := range effects {
			values[i] = build(bindings)
		}
		if event.Name == "WhenAPermanentDies" {
			death := bindings.hostDeaths[ctx.delayedDeath]
			return mage.WhenDiesThisTurnWithCard(watched, death.permanent, death.owner, death.controller, bindings.deathCards[ctx.deathCard], mage.Sequence(values...))
		}
		return mage.WhenLeavesThisTurn(watched, mage.Sequence(values...))
	}, nil
}
