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

type nextPermanentSourcePrevention struct {
	subject   PermanentRef
	recipient PlayerRef
	filter    PermanentFilter
}

func (*nextPermanentSourcePrevention) Text() string {
	return "prevent next damage from a permanent this turn"
}
func (*nextPermanentSourcePrevention) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *nextPermanentSourcePrevention) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-next-source-prevention", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"recipient", e.recipient}}}
}
func (e *nextPermanentSourcePrevention) Apply(ctx *EffectContext) error {
	source, ok := e.subject.resolvePermanent(ctx)
	if !ok || source == nil || source.PhasedOut {
		return nil
	}
	player, ok := e.recipient.resolvePlayer(ctx)
	if !ok || !e.filter.Match(source, ctx.Game) {
		return nil
	}
	ctx.Game.effects.AddReplacement(&sourcePreventionReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn},
		playerID:        player, dmgSource: source.ID(), incarnation: source.incarnationID, from: e.filter,
	})
	return nil
}

func (r *sourcePreventionReplacement) capturesSourceIdentity() bool {
	return r.incarnation != uuid.Nil || r.exactSource.Valid()
}

func (r *sourcePreventionReplacement) preventBatch(actions []Action, applied []map[ReplacementEffect]bool, g *Game) {
	// One simultaneous event can contain several packets from this source.
	for i, action := range actions {
		if action == nil || applied[i][r] || damageReplacementBlocked(action, g) || !r.Matches(action, g) {
			continue
		}
		applied[i][r] = true
		actions[i] = r.Replace(action, g)
	}
}
