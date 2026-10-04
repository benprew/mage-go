package ron

import "github.com/benprew/mage-go/pkg/mage"

type playerTargetPlan struct {
	name string
}

type playerReferencePlan struct {
	target *playerTargetPlan
	spell  *spellReferencePlan
}

type instanceBindings struct {
	permanents map[*permanentTargetPlan]*mage.PermanentTargetBinding
	players    map[*playerTargetPlan]*mage.PlayerTargetBinding
	spells     map[*spellTargetPlan]*mage.SpellTargetBinding
}

func compilePlayerRef(n Node, ctx scope) (playerReferencePlan, error) {
	if n.Name == "ControllerOfSpell" {
		if err := arity(n, 1); err != nil {
			return playerReferencePlan{}, err
		}
		spell, err := compileSpellRef(n.Args[0], ctx)
		return playerReferencePlan{spell: &spell}, err
	}
	if n.Kind == Symbol {
		switch n.Name {
		case "You":
			return playerReferencePlan{}, nil
		case "ControllerOfTargetSpell":
			if ctx.spellTarget != nil {
				return playerReferencePlan{spell: &spellReferencePlan{target: ctx.spellTarget}}, nil
			}
		case "Ref_TargetPlayer":
			if ctx.playerTarget != nil && ctx.playerTarget.name == n.Name {
				return playerReferencePlan{target: ctx.playerTarget}, nil
			}
		}
	}
	return playerReferencePlan{}, problem(n, "unbound or incompatible player reference")
}

func (p playerReferencePlan) build(bindings instanceBindings) mage.PlayerRef {
	if p.spell != nil {
		return mage.ControllerOfSpell(p.spell.build(bindings))
	}
	if p.target == nil {
		return mage.You
	}
	return bindings.players[p.target]
}
