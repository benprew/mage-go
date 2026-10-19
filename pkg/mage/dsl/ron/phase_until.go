package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePhaseUntil(n Node, ctx scope) (effectFactory, error) {
	if !ctx.permanent || !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "phasing until source departure requires an ordinary resolving permanent body")
	}
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	duration := n.Args[1]
	if duration.Name != "UntilPermanentLeavesBattlefield" {
		return nil, unsupported(duration, "phasing duration")
	}
	if err := arity(duration, 1); err != nil {
		return nil, err
	}
	if duration.Args[0].Kind != Symbol || duration.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(duration.Args[0], "phasing duration source")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "TapAsPhasesIn" {
		return nil, unsupported(n.Args[2], "phase-in operation")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PhaseOutUntilSourceLeaves(subject.build(bindings), true)
	}, nil
}
