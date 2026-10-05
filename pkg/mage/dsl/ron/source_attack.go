package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSourceAttackTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "source attack subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(subject.Args[0], "source attack reference")
	}
	ctx.sourceAttack = true
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		var pattern mage.EventPattern = mage.Attacks(mage.This)
		if event.Name == "WhenACreatureAttacksAndIsntBlocked" {
			pattern = mage.AttacksUnblocked(mage.This)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
