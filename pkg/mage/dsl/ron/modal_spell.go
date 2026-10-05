package ron

import (
	"strconv"
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileModalSpell(n Node, ctx scope) (optionFactory, error) {
	if ctx.castingCondition != nil {
		return nil, unsupported(n, "casting window on a modal spell")
	}
	if ctx.permanent {
		return nil, unsupported(n, "modal spell on a permanent")
	}
	plans, err := compileChooseOneModes(n, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.NewModalSpell(buildChooseOneModes(plans, instanceBindings{})))
	}, nil
}

func compileChooseOneModes(n Node, ctx scope) ([]actionPlan, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	bodies, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(bodies) < 2 {
		return nil, problem(n, "choose-one requires at least two modes")
	}
	plans := make([]actionPlan, len(bodies))
	for i, body := range bodies {
		// A mode cannot read targets from another mode.
		local := ctx
		local.targetNamespace = modalTargetNamespace(i)
		local.target = ""
		local.playerTarget = nil
		local.permanentTarget = nil
		local.permanentTargets = nil
		local.spellTarget = nil
		local.cardTarget = nil
		local.damageTarget = nil
		plans[i], err = compileActions(body, local)
		if err != nil {
			return nil, err
		}
	}
	return plans, nil
}

func buildChooseOneModes(plans []actionPlan, bindings instanceBindings) []mage.Mode {
	modes := make([]mage.Mode, len(plans))
	for i, plan := range plans {
		actions := plan.buildWithBindings(bindings)
		prefix := modalTargetNamespace(i)
		label := strings.NewReplacer(
			prefix+"Ref_TargetPermanent", "target permanent",
			prefix+"Ref_TargetPlayer", "target player",
			prefix+"Ref_TargetSpell", "target spell",
			prefix+"Ref_TargetGraveyardCard", "target graveyard card",
			"Ref_AnyTarget", "any target",
		).Replace(mage.Sequence(actions.effects...).Text())
		modes[i] = mage.Mode{Label: label, Targets: actions.targets, Effects: actions.effects}
	}
	return modes
}

func modalTargetNamespace(index int) string {
	return "mode" + strconv.Itoa(index+1) + "/"
}
