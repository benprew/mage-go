package ron

import (
	"strconv"
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileModalSpell(n Node, ctx scope) (optionFactory, error) {
	if ctx.permanent {
		return nil, unsupported(n, "modal spell on a permanent")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	bodies, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(bodies) < 2 {
		return nil, problem(n, "modal spell requires at least two modes")
	}
	plans := make([]actionPlan, len(bodies))
	for i, body := range bodies {
		// Each mode must declare its own targets before it can read them.
		local := ctx
		local.targetNamespace = modalTargetNamespace(i)
		local.target = ""
		local.playerTarget = nil
		local.permanentTarget = nil
		local.spellTarget = nil
		plans[i], err = compileActions(body, local)
		if err != nil {
			return nil, err
		}
	}
	return func() mage.CardOption {
		modes := make([]mage.Mode, len(plans))
		for i, plan := range plans {
			actions := plan.build()
			prefix := modalTargetNamespace(i)
			label := strings.NewReplacer(
				prefix+"Ref_TargetPermanent", "target permanent",
				prefix+"Ref_TargetPlayer", "target player",
				prefix+"Ref_TargetSpell", "target spell",
				"Ref_AnyTarget", "any target",
			).Replace(mage.Sequence(actions.effects...).Text())
			modes[i] = mage.Mode{Label: label, Targets: actions.targets, Effects: actions.effects}
		}
		return mage.WithAbility(mage.NewModalSpell(modes))
	}, nil
}

func modalTargetNamespace(index int) string {
	return "mode" + strconv.Itoa(index+1) + "/"
}
