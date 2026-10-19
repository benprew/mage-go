package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageSourceFactory func(instanceBindings) mage.DamageSourceRef

func compileSelfDamageSource(n Node, ctx scope, permanent bool) (damageSourceFactory, error) {
	if ctx.discardCauseEvent != nil {
		return nil, unsupported(n, "source damage in a discard event")
	}
	if permanent {
		ref, err := compilePermanentRef(n, ctx)
		if err != nil || ref.kind != permanentSource {
			return nil, unsupported(n, "damage source")
		}
		return func(bindings instanceBindings) mage.DamageSourceRef {
			return mage.DamageSourceOf(ref.build(bindings))
		}, nil
	}
	if n.Kind != Symbol || n.Name != "ThisSpell" || ctx.permanent || ctx.sourceZone != nil {
		return nil, unsupported(n, "damage source")
	}
	return func(instanceBindings) mage.DamageSourceRef { return mage.This }, nil
}
