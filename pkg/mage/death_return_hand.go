package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ReturnToHandInsteadOfDyingWithPlayLock replaces the source's death with a return.
// The returned hand visit stays revealed and cannot be played until its owner's next turn.
func ReturnToHandInsteadOfDyingWithPlayLock() Ability {
	return &deathReturnHandAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}}
}

type deathReturnHandAbility struct{ BaseAbility }

func (*deathReturnHandAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-death-return", Text: "return to hand instead of dying with next-turn play lock", battlefieldSource: true}
}

type deathReturnHandReplacement struct {
	replacementBase
	incarnation, ability uuid.UUID
}

func (r *deathReturnHandReplacement) IsActive(g GameReader) bool {
	p := g.FindPermanent(r.sourceID)
	if p == nil || p.PhasedOut || p.FaceDown || p.incarnationID != r.incarnation {
		return false
	}
	for _, ability := range p.RuntimeAbilities {
		if ability.AbilityID() == r.ability {
			if _, ok := UnwrapAbility(ability).(*deathReturnHandAbility); ok {
				return true
			}
		}
	}
	return false
}

func deathMoveSubject(action Action) (uuid.UUID, core.Zone) {
	switch a := action.(type) {
	case *DestroyPermanentAction:
		return a.PermanentID(), core.ZoneGraveyard
	case *SacrificePermanentAction:
		return a.PermanentID(), a.Destination()
	case *permanentGraveyardAction:
		return a.subject, a.destination
	}
	return uuid.Nil, core.Zone(-1)
}

func (r *deathReturnHandReplacement) Matches(action Action, g GameReader) bool {
	id, destination := deathMoveSubject(action)
	if id != r.sourceID || destination != core.ZoneGraveyard || !r.IsActive(g) {
		return false
	}
	switch a := action.(type) {
	case *SacrificePermanentAction:
		return a.incarnation == r.incarnation
	case *permanentGraveyardAction:
		return a.incarnation == r.incarnation
	}
	return true
}

func (r *deathReturnHandReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	if sacrifice, ok := action.(*SacrificePermanentAction); ok {
		cloned := *sacrifice
		cloned.destination = core.ZoneHand
		cloned.lockReturnedHand = true
		return &cloned
	}
	return &permanentGraveyardAction{subject: r.sourceID, incarnation: r.incarnation, destination: core.ZoneHand, lockReturnedHand: true}
}
func (*deathReturnHandReplacement) Text() string { return "return to hand instead of dying" }
func (r *deathReturnHandReplacement) Clone() ReplacementEffect {
	cloned := *r
	return &cloned
}

func (g *Game) deathReturnHandCandidates(action Action) []ReplacementEffect {
	id, _ := deathMoveSubject(action)
	p := g.FindPermanent(id)
	if p == nil || p.PhasedOut || p.FaceDown {
		return nil
	}
	var candidates []ReplacementEffect
	for _, ability := range p.RuntimeAbilities {
		if _, ok := UnwrapAbility(ability).(*deathReturnHandAbility); ok {
			candidates = append(candidates, &deathReturnHandReplacement{
				replacementBase: replacementBase{sourceID: id}, incarnation: p.incarnationID, ability: ability.AbilityID(),
			})
		}
	}
	return candidates
}

type handVisitPlayLock struct {
	effectSource
	player, card, visit uuid.UUID
	afterTurn           int
}

func (*handVisitPlayLock) Text() string                 { return "revealed hand card cannot be played until next turn" }
func (*handVisitPlayLock) Properties() EffectProperties { return EffectProperties{} }
func (*handVisitPlayLock) GetLayer() core.Layer         { return core.LayerAbility }
func (*handVisitPlayLock) GetDuration() core.Duration   { return core.Indefinite }
func (e *handVisitPlayLock) NextTurnExpiry() (player uuid.UUID, afterTurn int) {
	return e.player, e.afterTurn
}
func (e *handVisitPlayLock) IsActive(g *Game) bool {
	return e.visit != uuid.Nil && g.handVisit(e.card) == e.visit && g.findCardInZone(e.player, e.card, core.ZoneHand) != nil
}
func (*handVisitPlayLock) ApplyContinuous(*EffectContext) error { return nil }
func (e *handVisitPlayLock) cloneEffect() ContinuousEffect      { cloned := *e; return &cloned }

func (g *Game) lockReturnedHandCard(owner uuid.UUID, card Card) {
	visit := g.handVisit(card.ID())
	if visit == uuid.Nil {
		return
	}
	g.effects.Add(&handVisitPlayLock{player: owner, card: card.ID(), visit: visit, afterTurn: g.CurrentTurn()})
	g.syncHandVisibility()
}

func (g *Game) isHandVisitPlayLocked(player, card uuid.UUID) bool {
	for _, effect := range g.effects.effects {
		if rule, ok := effect.(*handVisitPlayLock); ok && rule.player == player && rule.card == card && rule.IsActive(g) {
			return true
		}
	}
	return false
}

// IsHandCardRevealed reports current public visibility for one hand card.
// A later visit cannot use the former visit's reveal.
func (g *Game) IsHandCardRevealed(player, card uuid.UUID) bool {
	return g.findCardInZone(player, card, core.ZoneHand) != nil && (g.IsHandRevealed(player) || g.isHandVisitPlayLocked(player, card))
}
