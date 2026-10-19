package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOtherSourceDamageHistory(n Node, ctx scope) (valueFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "damage history requires an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	filter := n.Args[1]
	if filter.Name != "And" {
		return nil, unsupported(filter, "other source damage filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "supply one Other subject and one named source filter")
	}
	var name string
	var other bool
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		switch part.Name {
		case "Other":
			if other {
				return nil, problem(part, "duplicate Other subject")
			}
			excluded, err := compilePermanentRef(part.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			if excluded != subject {
				return nil, problem(part, "Other must refer to the damage history subject")
			}
			other = true
		case "IsNamed":
			if name != "" {
				return nil, problem(part, "duplicate source name")
			}
			named := part.Args[0]
			if named.Name != "NamedCard" {
				return nil, unsupported(named, "damage history source name")
			}
			if err := arity(named, 1); err != nil {
				return nil, err
			}
			if named.Args[0].Kind != String || named.Args[0].String == "" {
				return nil, problem(named, "supply a nonempty card name")
			}
			name = named.Args[0].String
		default:
			return nil, unsupported(part, "other source damage filter")
		}
	}
	if !other || name == "" {
		return nil, problem(filter, "supply one Other subject and one named source filter")
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.DamageReceivedThisTurnFromOtherSources(subject.build(bindings), mage.DamageSourceHasName(name))
	}, nil
}

func compileDeadPermanentDamage(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if ctx.deathCard == nil || ctx.permanentEvent == nil || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "damage requires a source permanent death event")
	}
	source, err := compilePermanentRef(Node{Kind: Symbol, Name: "Trigger_ThatPermanent", Position: n.Position}, ctx)
	if err != nil {
		return nil, err
	}
	amount, err := compileInstanceValue(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	recipient, err := compileDamagePacketRecipient(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.DealDamageFrom(mage.DamageSourceOf(source.build(bindings)), recipient(bindings), amount(bindings))
	}, nil
}
