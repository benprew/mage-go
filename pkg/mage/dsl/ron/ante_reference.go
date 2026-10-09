package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAnteTarget(n Node, ctx scope) (targetFactory, string, error) {
	filter := n.Args[0]
	if filter.Kind != Symbol || filter.Name != "AnyCard" {
		return nil, "", unsupported(filter, "ante card target filter")
	}
	return func(instanceBindings) mage.Target {
		return mage.DeclareAnteCardTarget(ctx.targetNamespace + "Ref_TargetAnteCard")
	}, "Ref_TargetAnteCard", nil
}
