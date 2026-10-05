package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileStaticAttackPermission(n, operation Node, ctx scope) (optionFactory, error) {
	if operation.Kind != Symbol {
		return nil, unsupported(operation, "static attack permission operation")
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(n.Args[0], "static attack permission reference")
	}
	return func() mage.CardOption {
		ref := subject.build(instanceBindings{})
		modification := mage.GrantKeywordModification(ref, core.AttrCanAttack)
		if operation.Name == "CanAttackAsThoughItHadHaste" {
			modification = mage.AttackAsThoughHaste(ref)
		}
		condition := mage.StateConditionFunc("static attack permission", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.WithAbility(mage.Static(mage.While(condition, modification)))
	}, nil
}
