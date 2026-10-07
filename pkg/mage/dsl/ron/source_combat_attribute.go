package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileSourceCombatAttribute(n, operation Node, ctx scope) (optionFactory, error) {
	if _, err := symbol(operation); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource {
		return nil, unsupported(n.Args[0], "static source combat attribute reference")
	}
	attribute := core.AttrMustAttack
	if operation.Name == "CanBlockAnAdditionalCreature" {
		attribute = core.CanBlockAdditional
	}
	return func() mage.CardOption {
		condition := mage.StateConditionFunc("static source combat attribute", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.WithAbility(mage.Static(mage.While(condition, mage.GrantKeywordModification(subject.build(instanceBindings{}), attribute))))
	}, nil
}
