package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTurnStartUntappedCount(n Node, ctx scope) (valueFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "turn-start count context")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "turn-start count filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 3 {
		return nil, problem(filter, "turn-start count requires untapped lands and one player")
	}
	var untapped, land bool
	var player *playerReferencePlan
	for _, part := range parts {
		switch part.Name {
		case "IsUntapped":
			if _, err := symbol(part); err != nil {
				return nil, err
			}
			if untapped {
				return nil, problem(part, "duplicate untapped filter")
			}
			untapped = true
		case "IsCardtype":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if land || part.Args[0].Kind != Symbol || part.Args[0].Name != "Land" {
				return nil, unsupported(part, "turn-start land filter")
			}
			land = true
		case "ControlledByAPlayer":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			players := part.Args[0]
			if players.Name != "SinglePlayer" {
				return nil, unsupported(players, "turn-start player filter")
			}
			if err := arity(players, 1); err != nil {
				return nil, err
			}
			if player != nil {
				return nil, problem(part, "duplicate turn-start player")
			}
			ref, err := compilePlayerRef(players.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			player = &ref
		default:
			return nil, unsupported(part, "turn-start count filter")
		}
	}
	if !untapped || !land || player == nil {
		return nil, problem(filter, "turn-start count requires untapped lands and one player")
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.UntappedLandCountAtTurnStart(player.build(bindings))
	}, nil
}
