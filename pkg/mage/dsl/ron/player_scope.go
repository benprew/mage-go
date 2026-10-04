package ron

import "github.com/benprew/mage-go/pkg/mage"

type playerTargetPlan struct {
	name string
}

type playerReferencePlan struct {
	target *playerTargetPlan
}

type instanceBindings struct {
	permanents map[*permanentTargetPlan]*mage.PermanentTargetBinding
	players    map[*playerTargetPlan]*mage.PlayerTargetBinding
	spells     map[*spellTargetPlan]*mage.SpellTargetBinding
}

func compilePlayerRef(n Node, ctx scope) (playerReferencePlan, error) {
	if n.Kind == Symbol {
		switch n.Name {
		case "You":
			return playerReferencePlan{}, nil
		case "Ref_TargetPlayer":
			if ctx.playerTarget != nil && ctx.playerTarget.name == n.Name {
				return playerReferencePlan{target: ctx.playerTarget}, nil
			}
		}
	}
	return playerReferencePlan{}, problem(n, "unbound or incompatible player reference")
}

func (p playerReferencePlan) build(bindings instanceBindings) mage.PlayerRef {
	if p.target == nil {
		return mage.You
	}
	return bindings.players[p.target]
}
