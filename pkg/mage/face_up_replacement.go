package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// TurnResolvedPermanentFaceUpBeforeAction replaces assignment, damage and tapping.
// The exact spell's permanent turns face up before the original event continues.
func TurnResolvedPermanentFaceUpBeforeAction(spell DamageSourceRef) ResolvingEffect {
	return &faceUpBeforeActionEffect{spell: spell}
}

type faceUpBeforeActionEffect struct{ spell DamageSourceRef }

func (*faceUpBeforeActionEffect) Text() string {
	return "turn the resolved permanent face up before its action"
}
func (*faceUpBeforeActionEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *faceUpBeforeActionEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"spell", damageSourceDefinition{e.spell}}}}
}
func (e *faceUpBeforeActionEffect) Apply(ctx *EffectContext) error {
	spell := e.spell.resolveDamageSource(ctx)
	if !spell.hasExactIdentity() || spell.StackID() == uuid.Nil {
		return nil
	}
	ctx.Game.effects.AddReplacement(&faceUpActionReplacement{replacementBase: replacementBase{sourceID: ctx.SourceID, duration: core.Indefinite}, spellID: spell.StackID(), cardID: spell.ID()})
	return nil
}

type faceUpActionReplacement struct {
	replacementBase
	spellID, cardID uuid.UUID
}

func (*faceUpActionReplacement) survivesSourceExit() bool { return true }
func (r *faceUpActionReplacement) IsActive(g GameReader) bool {
	if p := g.(*Game).FindPermanentIncludingPhased(r.cardID); p != nil && p.resolvedSpellID == r.spellID {
		return p.FaceDown && !p.turnedFaceUp
	}
	return g.(*Game).stack.FindByID(r.spellID) != nil
}
func (r *faceUpActionReplacement) subject(g GameReader) *Permanent {
	p := g.FindPermanent(r.cardID)
	if p == nil || p.PhasedOut || !p.FaceDown || p.turnedFaceUp || p.resolvedSpellID != r.spellID {
		return nil
	}
	return p
}
func (r *faceUpActionReplacement) Matches(action Action, g GameReader) bool {
	p := r.subject(g)
	if p == nil {
		return false
	}
	switch a := action.(type) {
	case *tapPermanentAction:
		return a.subject == (capturedPermanent{p.ID(), p.incarnationID}) && !p.Tapped
	case *assignDamageAction:
		return a.subject == (capturedPermanent{p.ID(), p.incarnationID})
	case *DamageToPlayerAction:
		return a.Amount() > 0 && a.DamageSource().ID() == p.ID() && a.DamageSource().Incarnation() == p.incarnationID
	case *DamageToCreatureAction:
		return a.Amount() > 0 && (a.PermanentID() == p.ID() && (a.recipientIncarnation == uuid.Nil || a.recipientIncarnation == p.incarnationID) || a.DamageSource().ID() == p.ID() && a.DamageSource().Incarnation() == p.incarnationID)
	}
	return false
}
func (r *faceUpActionReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	p := r.subject(g)
	g.turnFaceUp(p)
	return refreshTurnedDamageSource(action, g, p.ID(), p.incarnationID)
}
func refreshTurnedDamageSource(action Action, g *Game, id, incarnation uuid.UUID) Action {
	switch a := action.(type) {
	case *DamageToPlayerAction:
		if a.DamageSource().ID() == id && a.DamageSource().Incarnation() == incarnation {
			cp := *a
			cp.sourceState = g.CaptureDamageSource(id)
			return &cp
		}
	case *DamageToCreatureAction:
		if a.DamageSource().ID() == id && a.DamageSource().Incarnation() == incarnation {
			cp := *a
			cp.sourceState = g.CaptureDamageSource(id)
			return &cp
		}
	}
	return action
}
func (r *faceUpActionReplacement) matchesDamageBatch(actions []Action, _ []map[ReplacementEffect]bool, index int, g *Game) bool {
	return r.Matches(actions[index], g)
}
func (r *faceUpActionReplacement) replaceDamageBatch(actions []Action, _ []map[ReplacementEffect]bool, index int, g *Game) {
	p := r.subject(g)
	if p == nil || !r.Matches(actions[index], g) {
		return
	}
	id, incarnation := p.ID(), p.incarnationID
	g.turnFaceUp(p)
	for i, action := range actions {
		actions[i] = refreshTurnedDamageSource(action, g, id, incarnation)
	}
}
func (r *faceUpActionReplacement) Clone() ReplacementEffect { cp := *r; return &cp }
func (*faceUpActionReplacement) Text() string               { return "turn face up, then continue the event" }

type tapPermanentAction struct{ subject capturedPermanent }

func (a *tapPermanentAction) ActionSource() uuid.UUID { return a.subject.id }
func (a *tapPermanentAction) PermanentID() uuid.UUID  { return a.subject.id }

type assignDamageAction struct{ subject capturedPermanent }

func (a *assignDamageAction) ActionSource() uuid.UUID { return a.subject.id }
func (a *assignDamageAction) PermanentID() uuid.UUID  { return a.subject.id }

func (g *Game) preparePermanentTap(p *Permanent) *Permanent {
	if p == nil {
		return nil
	}
	live := g.FindPermanent(p.ID())
	if live == nil || live.PhasedOut || live.Tapped || live.incarnationID != p.incarnationID {
		return nil
	}
	action := &tapPermanentAction{capturedPermanent{p.ID(), p.incarnationID}}
	if g.effects.ApplyReplacements(action, g) == nil {
		return nil
	}
	live = g.MutablePermanent(p.ID())
	if live == nil || live.PhasedOut || live.Tapped || live.incarnationID != p.incarnationID {
		return nil
	}
	return live
}
