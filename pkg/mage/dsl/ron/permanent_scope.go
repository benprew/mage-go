package ron

import "github.com/benprew/mage-go/pkg/mage"

type permanentTargetPlan struct {
	name string
}

type permanentReferenceKind uint8

const (
	permanentSource permanentReferenceKind = iota
	permanentHost
	permanentTargetHost
	permanentTarget
	permanentEvent
	permanentDeath
	permanentChoice
	permanentDamageTarget
	permanentPlayerChoice
)

type permanentReferencePlan struct {
	kind         permanentReferenceKind
	target       *permanentTargetPlan
	event        *permanentEventPlan
	death        *hostDeathPlan
	choice       *permanentChoicePlan
	damage       *damageTargetPlan
	keyed        *permanentChoiceHistoryPlan
	choicePlayer playerReferencePlan
}

func compilePermanentRef(n Node, ctx scope) (permanentReferencePlan, error) {
	if n.Name == "ThePermanentChosenByPlayerThisWay" {
		if err := arity(n, 1); err != nil {
			return permanentReferencePlan{}, err
		}
		player, err := compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return permanentReferencePlan{}, err
		}
		if ctx.permanentChoices == nil {
			return permanentReferencePlan{}, problem(n, "player choice requires a preceding producer")
		}
		return permanentReferencePlan{kind: permanentPlayerChoice, keyed: ctx.permanentChoices, choicePlayer: player}, nil
	}
	if n.Kind == Symbol {
		if ctx.unblockedCreature != nil && n.Name == ctx.unblockedCreature.name {
			return permanentReferencePlan{kind: permanentEvent, event: ctx.unblockedCreature}, nil
		}
		if target := ctx.permanentGroups[n.Name]; target != nil {
			return permanentReferencePlan{kind: permanentTarget, target: target}, nil
		}
		if ctx.combatOpponent != nil && (n.Name == ctx.combatOpponent.name || n.Name == "Trigger_ThatOtherPermanent") {
			return permanentReferencePlan{kind: permanentEvent, event: ctx.combatOpponent}, nil
		}
		switch n.Name {
		case "AnyTargetAsAPermanent":
			if ctx.damageTarget != nil && ctx.damageTarget.name == "Ref_AnyTarget" {
				return permanentReferencePlan{kind: permanentDamageTarget, damage: ctx.damageTarget}, nil
			}
		case "TheChosenPermanent":
			if ctx.chosenPermanent != nil {
				return permanentReferencePlan{kind: permanentChoice, choice: ctx.chosenPermanent}, nil
			}
		case "Trigger_ThatCreatureOrPlaneswalker":
			if ctx.receivedSource {
				return permanentReferencePlan{kind: permanentSource}, nil
			}
		case "Trigger_ThatCreature":
			if ctx.sourceAttack || ctx.sourceBlock {
				return permanentReferencePlan{kind: permanentSource}, nil
			}
		case "ThatEnteringPermanent":
			if ctx.enteringSource {
				return permanentReferencePlan{kind: permanentSource}, nil
			}
		case "ThisPermanent":
			if ctx.permanent {
				return permanentReferencePlan{kind: permanentSource}, nil
			}
		case "HostPermanent":
			if ctx.aura {
				return permanentReferencePlan{kind: permanentHost}, nil
			}
			if ctx.targetHost != nil && !ctx.enteringSource && !ctx.drawReplacementBody {
				return permanentReferencePlan{kind: permanentTargetHost, target: ctx.targetHost}, nil
			}
		case "Trigger_ThatPermanent":
			if ctx.delayedDeath != nil {
				return permanentReferencePlan{kind: permanentDeath, death: ctx.delayedDeath}, nil
			}
			if ctx.hostDeath != nil {
				return permanentReferencePlan{kind: permanentDeath, death: ctx.hostDeath}, nil
			}
			if ctx.counterStateSource || ctx.sourceTapped || ctx.sourceLeaves {
				return permanentReferencePlan{kind: permanentSource}, nil
			}
			if ctx.permanentEvent != nil {
				ctx.permanentEvent.referenced = true
				return permanentReferencePlan{kind: permanentEvent, event: ctx.permanentEvent}, nil
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
	case permanentDamageTarget:
		return mage.PermanentFromDamageTarget(bindings.recipients[p.damage])
	case permanentSource:
		return mage.This
	case permanentHost:
		return mage.AttachedPermanent
	case permanentTargetHost:
		return mage.HostOfPermanent(bindings.permanents[p.target])
	case permanentEvent:
		return mage.From(bindings.tapped[p.event])
	case permanentDeath:
		return mage.From(bindings.hostDeaths[p.death].permanent)
	case permanentChoice:
		return mage.PermanentChosenIn(bindings.chosenPermanents[p.choice])
	case permanentPlayerChoice:
		choices := make([]*mage.PermanentChoiceBinding, 0, len(p.keyed.choices))
		for _, choice := range p.keyed.choices {
			choices = append(choices, bindings.chosenPermanents[choice])
		}
		return mage.PermanentChosenByPlayer(p.choicePlayer.build(bindings), choices...)
	default:
		return bindings.permanents[p.target]
	}
}
