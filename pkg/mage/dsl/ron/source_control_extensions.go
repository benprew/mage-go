package ron

import "github.com/benprew/mage-go/pkg/mage"

func sourcePowerLEShape(n Node) error {
	if err := arity(n, 1); err != nil {
		return err
	}
	comparison := n.Args[0]
	if comparison.Name != "LessThanOrEqualTo" {
		return unsupported(comparison, "source power comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return err
	}
	power := comparison.Args[0]
	if power.Name != "PowerOfPermanent" {
		return unsupported(power, "source power limit")
	}
	if err := arity(power, 1); err != nil {
		return err
	}
	source := power.Args[0]
	if source.Kind != Symbol || source.Name != "ThisPermanent" {
		return unsupported(source, "source power reference")
	}
	return nil
}

func isSourcePowerControlDuration(n Node) bool {
	if n.Name != "Or" || len(n.Args) != 1 || n.Args[0].Kind != List {
		return false
	}
	for _, part := range n.Args[0].Args {
		if part.Name == "UntilPermanentNoLongerPassesFilter" {
			return true
		}
	}
	return false
}

func compileSourcePowerControl(n Node, ctx scope, subject permanentReferencePlan) (effectFactory, error) {
	if !ctx.permanent || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "source power control requires an ordinary permanent-source resolving body")
	}
	duration := n.Args[1]
	if err := arity(duration, 1); err != nil {
		return nil, err
	}
	parts, err := list(duration.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(duration, "supply one tapped endpoint and one power endpoint")
	}
	tapped, power := false, false
	for _, part := range parts {
		switch part.Name {
		case "ForAsLongAsPermanentRemainsTapped":
			if tapped {
				return nil, problem(part, "duplicate tapped endpoint")
			}
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "ThisPermanent" {
				return nil, unsupported(part.Args[0], "source tapped reference")
			}
			tapped = true
		case "UntilPermanentNoLongerPassesFilter":
			if power {
				return nil, problem(part, "duplicate power endpoint")
			}
			if err := arity(part, 2); err != nil {
				return nil, err
			}
			endpoint, err := compilePermanentRef(part.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			if endpoint != subject {
				return nil, problem(part.Args[0], "power endpoint must use the controlled subject")
			}
			if part.Args[1].Name != "PowerIs" {
				return nil, unsupported(part.Args[1], "control power predicate")
			}
			if err := sourcePowerLEShape(part.Args[1]); err != nil {
				return nil, err
			}
			power = true
		default:
			return nil, unsupported(part, "control duration endpoint")
		}
	}
	if !tapped || !power {
		return nil, problem(duration, "supply tapped and power endpoints")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.GainControlWhileSourceTappedAndPowerLE(subject.build(bindings))
	}, nil
}

func compilePluralSourceControl(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.permanent || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "plural source control requires an ordinary permanent-source resolving body")
	}
	duration := n.Args[1]
	if duration.Name != "UntilPermanentChangesControl" {
		return nil, unsupported(duration, "plural source control duration")
	}
	if err := arity(duration, 2); err != nil {
		return nil, err
	}
	if duration.Args[0].Kind != Symbol || duration.Args[0].Name != "You" || duration.Args[1].Kind != Symbol || duration.Args[1].Name != "ThisPermanent" {
		return nil, problem(duration, "plural duration requires You and ThisPermanent")
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.GainControlEachWhileSourceControlled(subjects(bindings))
	}, nil
}
