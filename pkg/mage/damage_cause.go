package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type damageCause struct {
	stackID uuid.UUID
	targets []capturedPermanent
}

func (c damageCause) clone() damageCause {
	c.targets = slices.Clone(c.targets)
	return c
}

func damageCauseFromStack(obj *StackObject) damageCause {
	if obj == nil || obj.ID == uuid.Nil || (!obj.IsAbility && obj.Card == nil) {
		return damageCause{}
	}
	c := damageCause{stackID: obj.ID}
	for _, id := range obj.Targets {
		if incarnation := obj.targetIncarnations[id]; id != uuid.Nil && incarnation != uuid.Nil {
			c.targets = append(c.targets, capturedPermanent{id, incarnation})
		}
	}
	return c
}

func (g *Game) currentDamageCause() damageCause {
	if g.damage.resolvingCombatDamage {
		return damageCause{}
	}
	return g.resolution.damageCause.clone()
}

func damageActionCause(action Action) damageCause {
	switch a := action.(type) {
	case *DamageToCreatureAction:
		return a.cause
	case *DamageToPlayerAction:
		return a.cause
	}
	return damageCause{}
}

func keepDamageCause(before, after Action) Action {
	cause := damageActionCause(before)
	switch a := after.(type) {
	case *DamageToCreatureAction:
		a.cause = cause
		a.origin = damageOrigin(before)
	case *DamageToPlayerAction:
		a.cause = cause
		a.origin = damageOrigin(before)
	}
	return after
}

// PreventDamageCausedByTargetingThisTurn saves one subject until cleanup.
// The causing spell or ability must target that exact battlefield visit.
func PreventDamageCausedByTargetingThisTurn(subject PermanentRef) ResolvingEffect {
	return &targetingCausePrevention{subject: subject}
}

type targetingCausePrevention struct{ subject PermanentRef }

func (*targetingCausePrevention) Text() string {
	return "prevent damage caused by a targeting spell or ability this turn"
}
func (*targetingCausePrevention) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *targetingCausePrevention) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-cause-prevention", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *targetingCausePrevention) Apply(ctx *EffectContext) error {
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	saved := capturedPermanent{subject.ID(), subject.incarnationID}
	ctx.Game.effects.AddReplacement(&damagePreventionRuleReplacement{
		replacementBase:       replacementBase{duration: core.EndOfTurn},
		recipientPermanent:    saved,
		causeTargetsRecipient: true,
		noncombatOnly:         true,
		to: NewPermanentFilter("saved cause prevention subject", func(p *Permanent, _ *Game) bool {
			return p.ID() == saved.id && p.incarnationID == saved.incarnation
		}),
	})
	return nil
}

func damageOrigin(action Action) int {
	switch a := action.(type) {
	case *DamageToPlayerAction:
		return a.origin
	case *DamageToCreatureAction:
		return a.origin
	}
	return -1
}
func setDamageOrigin(action Action, origin int) {
	switch a := action.(type) {
	case *DamageToPlayerAction:
		a.origin = origin
	case *DamageToCreatureAction:
		a.origin = origin
	}
}
