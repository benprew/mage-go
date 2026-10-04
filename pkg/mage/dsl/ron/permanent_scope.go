package ron

import "github.com/benprew/mage-go/pkg/mage"

type permanentTargetPlan struct {
	name string
}

type permanentReferenceKind uint8

const (
	permanentSource permanentReferenceKind = iota
	permanentHost
	permanentTarget
)

type permanentReferencePlan struct {
	kind   permanentReferenceKind
	target *permanentTargetPlan
}

func compilePermanentRef(n Node, ctx scope) (permanentReferencePlan, error) {
	if n.Kind == Symbol {
		switch n.Name {
		case "ThisPermanent":
			if ctx.permanent {
				return permanentReferencePlan{kind: permanentSource}, nil
			}
		case "HostPermanent":
			if ctx.aura {
				return permanentReferencePlan{kind: permanentHost}, nil
			}
		case "Ref_TargetPermanent":
			if ctx.permanentTarget != nil && ctx.permanentTarget.name == n.Name {
				return permanentReferencePlan{kind: permanentTarget, target: ctx.permanentTarget}, nil
			}
		}
	}
	return permanentReferencePlan{}, problem(n, "unbound or incompatible permanent reference")
}

func (p permanentReferencePlan) build(bindings instanceBindings) mage.PermanentRef {
	switch p.kind {
	case permanentSource:
		return mage.This
	case permanentHost:
		return mage.AttachedPermanent
	default:
		return bindings.permanents[p.target]
	}
}
