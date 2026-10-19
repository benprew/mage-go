package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePlayerDamageHistory(n Node, ctx scope) (valueFactory, error) {
	artifact := false
	switch n.Name {
	case "DamageDealtToPlayerThisTurn":
	case "DamageDealtToPlayerThisTurnByPermanents":
		artifact = true
	default:
		return nil, unsupported(n, "player damage history value")
	}
	count := 1
	if artifact {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if artifact {
		filter := n.Args[1]
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		if filter.Name != "IsCardtype" || filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Artifact" {
			return nil, unsupported(filter, "damage history source filter")
		}
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		subject := player.build(bindings)
		if artifact {
			return mage.ArtifactDamageTakenThisTurn(subject)
		}
		return mage.DamageTakenThisTurn(subject)
	}, nil
}
