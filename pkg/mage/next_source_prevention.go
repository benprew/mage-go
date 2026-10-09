package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PreventNextDamageFromPermanent prevents the next damage from the subject to
// the player this turn. The filter is checked at installation and before damage.
// A zero filter permits all permanent sources.
func PreventNextDamageFromPermanent(subject PermanentRef, recipient PlayerRef, filter PermanentFilter) ResolvingEffect {
	return &nextPermanentSourcePrevention{subject: subject, recipient: recipient, filter: filter}
}

// PreventNextCombatDamageFromPermanentExcept leaves the specified damage amount
// in the next combat event from this creature to the player this turn.
// The amount is saved at resolution. The shield remains if it prevents no damage.
func PreventNextCombatDamageFromPermanentExcept(subject PermanentRef, recipient PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &nextPermanentSourcePrevention{subject: subject, recipient: recipient, filter: IsCreature, combatOnly: true, leave: amount}
}

type nextPermanentSourcePrevention struct {
	subject    PermanentRef
	recipient  PlayerRef
	filter     PermanentFilter
	combatOnly bool
	leave      ResolutionValue
}

func (*nextPermanentSourcePrevention) Text() string {
	return "prevent next damage from a permanent this turn"
}
func (*nextPermanentSourcePrevention) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *nextPermanentSourcePrevention) inspectDefinition() definitionNode {
	n := definitionNode{Role: "install-next-source-prevention", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"recipient", e.recipient}}}
	if e.combatOnly {
		n.children = append(n.children, definitionEdge{"amount", e.leave})
		if amount, ok := e.leave.(fixedValue); ok && amount.n < 0 {
			n.problem = "supply a nonnegative damage amount"
		}
	}
	return n
}
func (e *nextPermanentSourcePrevention) Apply(ctx *EffectContext) error {
	source, ok := e.subject.resolvePermanent(ctx)
	if !ok || source == nil || source.PhasedOut {
		return nil
	}
	player, ok := e.recipient.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil || !e.filter.Match(source, ctx.Game) {
		return nil
	}
	leave := 0
	if e.combatOnly {
		leave = max(0, e.leave.resolveValue(ctx))
	}
	ctx.Game.effects.AddReplacement(&sourcePreventionReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn},
		playerID:        player, dmgSource: source.ID(), incarnation: source.incarnationID, from: e.filter,
		combatOnly: e.combatOnly, leave: leave,
	})
	return nil
}

func (r *sourcePreventionReplacement) capturesSourceIdentity() bool {
	return r.incarnation != uuid.Nil || r.exactSource.Valid()
}

func (r *sourcePreventionReplacement) preventBatch(actions []Action, applied []map[ReplacementEffect]bool, g *Game) preventionLifeGain {
	// One simultaneous event can contain several packets from this source.
	if r.amount != nil {
		total := 0
		for i, action := range actions {
			if action != nil && !applied[i][r] && !damageReplacementBlocked(action, g) && r.Matches(action, g) {
				total += damageActionAmount(action)
			}
		}
		prevented := r.preventProposedAmount(total)
		remaining := prevented
		for i, action := range actions {
			if action == nil || applied[i][r] || damageReplacementBlocked(action, g) || !r.Matches(action, g) {
				continue
			}
			applied[i][r] = true
			amount := damageActionAmount(action)
			reduction := min(amount, remaining)
			remaining -= reduction
			actions[i] = damageActionWithAmount(action, amount-reduction)
		}
		return r.lifeGainForPrevention(prevented)
	}
	leave := r.leave
	for i, action := range actions {
		if action == nil || applied[i][r] || damageReplacementBlocked(action, g) || !r.Matches(action, g) {
			continue
		}
		applied[i][r] = true
		if r.combatOnly {
			amount := min(damageActionAmount(action), leave)
			leave -= amount
			actions[i] = damageActionWithAmount(action, amount)
			r.consumed = r.consumed || damageActionAmount(action) > amount
		} else {
			actions[i] = r.Replace(action, g)
		}
	}
	return preventionLifeGain{}
}
