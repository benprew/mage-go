package ron

import "github.com/benprew/mage-go/pkg/mage"

type sourceCombatPlayerPlan struct{ name string }

func compileSourceCombatTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "source combat subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind == permanentHost {
		return compileHostCombatTrigger(event, body, ctx)
	}
	if ref.kind != permanentSource {
		return nil, unsupported(subject.Args[0], "source combat reference")
	}
	ctx.sourceAttack = event.Name == "WhenACreatureAttacks" || event.Name == "WhenACreatureAttacksAndIsntBlocked"
	ctx.sourceBlock = event.Name == "WhenACreatureBlocks" || event.Name == "WhenACreatureBecomesBlocked"
	if ctx.sourceAttack {
		ctx.defendingPlayer = &sourceCombatPlayerPlan{name: "Trigger_DefendingPlayer"}
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		bindings := instanceBindings{}
		var defender *mage.PlayerBinding
		if ctx.defendingPlayer != nil {
			defender = mage.BindPlayer(ctx.defendingPlayer.name)
			bindings.defenders = map[*sourceCombatPlayerPlan]*mage.PlayerBinding{ctx.defendingPlayer: defender}
		}
		actions := plan.buildWithBindings(bindings)
		var pattern mage.EventPattern = mage.Attacks(mage.This)
		switch event.Name {
		case "WhenACreatureBlocks":
			pattern = mage.Blocks(mage.This)
		case "WhenACreatureBecomesBlocked":
			pattern = mage.BecomesBlocked(mage.This)
		case "WhenACreatureAttacksAndIsntBlocked":
			pattern = mage.AttacksUnblocked(mage.This).WithDefendingPlayer(defender)
		case "WhenACreatureAttacks":
			pattern = mage.Attacks(mage.This).WithDefendingPlayer(defender)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
