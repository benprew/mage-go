package ron

import "github.com/benprew/mage-go/pkg/mage"

type discardCauseEventPlan struct{ name string }

func compileDiscardCauseEvent(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	rule := n.Args[0]
	if rule.Name != "TriggerA" {
		return nil, unsupported(rule, "source discard trigger")
	}
	if err := arity(rule, 2); err != nil {
		return nil, err
	}
	event := rule.Args[0]
	if event.Name != "WhenASpellOrAbilityCausesAPlayerToDiscardACard" {
		return nil, unsupported(event, "source discard event")
	}
	if err := arity(event, 3); err != nil {
		return nil, err
	}
	for i, expected := range []struct{ name, value string }{{"ControlledByAPlayer", "Opponent"}, {"SinglePlayer", "You"}, {"SingleCardInHand", "ThisCardInHand"}} {
		operand := event.Args[i]
		if operand.Name != expected.name {
			return nil, unsupported(operand, "source discard operand")
		}
		if err := arity(operand, 1); err != nil {
			return nil, err
		}
		if operand.Args[0].Kind != Symbol || operand.Args[0].Name != expected.value {
			return nil, unsupported(operand.Args[0], "source discard reference")
		}
	}
	if rule.Args[1].Name != "ActionList" {
		return nil, unsupported(rule.Args[1], "source discard action list")
	}
	plan := &discardCauseEventPlan{name: "Trigger_ControllerOfThatSpellOrAbility"}
	body, err := compileActions(rule.Args[1], scope{discardCauseEvent: plan})
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		cause := mage.BindPlayer(plan.name)
		bindings := instanceBindings{discardCauses: map[*discardCauseEventPlan]*mage.PlayerBinding{plan: cause}}
		actions := body.buildWithBindings(bindings)
		return mage.WithAbility(mage.WhenEvent(mage.ThisCardDiscardedByOpponentEffect(cause), mage.Sequence(actions.effects...)))
	}, nil
}
