package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSpellColorCopy(n Node, ctx scope) (effectFactory, error) {
	if !ctx.spellTargets || ctx.permanent || ctx.activation || ctx.delayedStep || ctx.drawReplacementBody {
		return nil, unsupported(n, "spell color copy context")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subject, err := compileSpellRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.target == nil || !subject.target.instantOrSorcery {
		return nil, problem(n.Args[0], "color copy requires a target restricted to instants or sorceries")
	}
	effects, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(effects) != 1 || effects[0].Name != "SetColor" {
		return nil, problem(n.Args[1], "spell copy requires one fixed SetColor operation")
	}
	colors, err := compileColorList(effects[0])
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.CopyInstantOrSorceryWithColors(subject.build(bindings), colors...)
	}, nil
}

func spellFilterOnlyInstantsOrSorceries(n Node) bool {
	if n.Kind != Tuple || len(n.Args) != 1 {
		return false
	}
	if n.Name == "IsCardtype" {
		return n.Args[0].Kind == Symbol && (n.Args[0].Name == "Instant" || n.Args[0].Name == "Sorcery")
	}
	if (n.Name != "And" && n.Name != "Or") || n.Args[0].Kind != List || len(n.Args[0].Args) == 0 {
		return false
	}
	for _, part := range n.Args[0].Args {
		only := spellFilterOnlyInstantsOrSorceries(part)
		if n.Name == "And" && only {
			return true
		}
		if n.Name == "Or" && !only {
			return false
		}
	}
	return n.Name == "Or"
}
