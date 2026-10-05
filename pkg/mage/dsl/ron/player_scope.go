package ron

import "github.com/benprew/mage-go/pkg/mage"

type playerTargetPlan struct {
	name string
}

type playerReferencePlan struct {
	permanent *permanentReferencePlan
	target    *playerTargetPlan
	spell     *spellReferencePlan
	event     *castEventPlan
	step      *stepEventPlan
	landMana  *landManaEventPlan
}

type castEventPlan struct{ spellName, casterName string }

type castEventBindings struct {
	spell  *mage.SpellBinding
	caster *mage.PlayerBinding
}

type instanceBindings struct {
	counters   map[*counterPaymentPlan]*mage.NumberBinding
	sacrifices map[*sacrificeValuePlan]sacrificeValueBindings
	events     map[*castEventPlan]castEventBindings
	steps      map[*stepEventPlan]*mage.PlayerBinding
	landMana   map[*landManaEventPlan]landManaEventBindings
	tapped     map[*permanentEventPlan]*mage.PermanentBinding
	permanents map[*permanentTargetPlan]*mage.PermanentTargetBinding
	players    map[*playerTargetPlan]*mage.PlayerTargetBinding
	spells     map[*spellTargetPlan]*mage.SpellTargetBinding
	cards      map[*cardTargetPlan]*mage.CardTargetBinding
	recipients map[*damageTargetPlan]*mage.DamageTargetBinding
}

func compilePlayerRef(n Node, ctx scope) (playerReferencePlan, error) {
	if n.Name == "ControllerOfPermanent" {
		if err := arity(n, 1); err != nil {
			return playerReferencePlan{}, err
		}
		permanent, err := compilePermanentRef(n.Args[0], ctx)
		return playerReferencePlan{permanent: &permanent}, err
	}
	if n.Name == "ControllerOfSpell" {
		if err := arity(n, 1); err != nil {
			return playerReferencePlan{}, err
		}
		spell, err := compileSpellRef(n.Args[0], ctx)
		return playerReferencePlan{spell: &spell}, err
	}
	if n.Kind == Symbol {
		switch n.Name {
		case "Trigger_ThatPlayer":
			if ctx.landManaEvent != nil {
				return playerReferencePlan{landMana: ctx.landManaEvent}, nil
			}
			if ctx.stepEvent != nil {
				return playerReferencePlan{step: ctx.stepEvent}, nil
			}
			if ctx.castEvent != nil && n.Name == ctx.castEvent.casterName {
				return playerReferencePlan{event: ctx.castEvent}, nil
			}
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
	if p.landMana != nil {
		return mage.FromPlayer(bindings.landMana[p.landMana].player)
	}
	if p.permanent != nil {
		return mage.ControllerOfWithLKI(p.permanent.build(bindings))
	}
	if p.step != nil {
		return mage.FromPlayer(bindings.steps[p.step])
	}
	if p.event != nil {
		return mage.FromPlayer(bindings.events[p.event].caster)
	}
	if p.spell != nil {
		return mage.ControllerOfSpell(p.spell.build(bindings))
	}
	if p.target == nil {
		return mage.You
	}
	return bindings.players[p.target]
}
