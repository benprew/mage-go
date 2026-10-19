package ron

import "github.com/benprew/mage-go/pkg/mage"

type hostDeathPlan struct {
	name         string
	departedAura bool
	watched      *permanentReferencePlan
}

type hostDeathBindings struct {
	controller, owner *mage.PlayerBinding
	toughness         *mage.NumberBinding
	permanent         *mage.PermanentBinding
	aura              *mage.CardBinding
}

func compileHostDeathTrigger(subject, body Node, ctx scope) (optionFactory, error) {
	ref, err := compilePermanentRef(subject, ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentHost {
		return nil, unsupported(subject, "host death reference")
	}
	ctx.hostDeath = &hostDeathPlan{name: "Trigger_ThatPermanent"}
	ctx.deathCard = &deathCardPlan{name: "Trigger_ThatGraveyardCard"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		death := hostDeathBindings{
			controller: mage.BindPlayer("ControllerOfDeadPermanent"),
			owner:      mage.BindPlayer("OwnerOfDeadPermanent"),
			toughness:  mage.BindNumber("ToughnessOfDeadPermanent"),
			permanent:  mage.BindPermanent(ctx.hostDeath.name),
			aura:       mage.BindCard("departed Aura"),
		}
		card := mage.BindCard(ctx.deathCard.name)
		actions := plan.buildWithBindings(instanceBindings{
			hostDeaths: map[*hostDeathPlan]hostDeathBindings{ctx.hostDeath: death},
			deathCards: map[*deathCardPlan]*mage.CardBinding{ctx.deathCard: card},
		})
		pattern := mage.AttachedPermanentDies().WithController(death.controller).WithOwner(death.owner).
			WithToughness(death.toughness).WithPermanent(death.permanent).WithCard(card)
		if ctx.hostDeath.departedAura {
			pattern = pattern.WithDepartedAuraCard(death.aura)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileDeadGuestDamage(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if ctx.hostDeath == nil {
		return nil, problem(n, "damage requires an attached permanent death event")
	}
	amount, err := compileInstanceValue(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	recipient, err := compileDamagePacketRecipient(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.DealDamageFrom(mage.DamageSourceOf(mage.This), recipient(bindings), amount(bindings))
	}, nil
}
