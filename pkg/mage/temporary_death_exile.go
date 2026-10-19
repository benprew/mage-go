package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentGraveyardAction struct {
	subject, incarnation, exileSource uuid.UUID
	destination                       core.Zone
	lockReturnedHand                  bool
}

func (a *permanentGraveyardAction) ActionSource() uuid.UUID { return uuid.Nil }
func (a *permanentGraveyardAction) PermanentID() uuid.UUID  { return a.subject }

type temporaryDeathExileEffect struct{ subject PermanentRef }

// ExileIfWouldDieThisTurn saves one permanent until cleanup.
// A returned permanent cannot use the saved rule.
func ExileIfWouldDieThisTurn(subject PermanentRef) ResolvingEffect {
	return &temporaryDeathExileEffect{subject: subject}
}
func (*temporaryDeathExileEffect) Text() string                 { return "exile if it would die this turn" }
func (*temporaryDeathExileEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *temporaryDeathExileEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "temporary-death-exile", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *temporaryDeathExileEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}
func (e *temporaryDeathExileEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok || p == nil || p.PhasedOut {
		return nil
	}
	ctx.Game.effects.AddReplacement(&temporaryDeathExileReplacement{
		replacementBase: replacementBase{sourceID: ctx.SourceID, duration: core.EndOfTurn},
		subject:         p.ID(), incarnation: p.incarnationID, reference: damageSourceFromPermanent(p),
	})
	return nil
}

type temporaryDeathExileReplacement struct {
	replacementBase
	subject, incarnation uuid.UUID
	reference            DamageSource
	historySource        DamageSource
	historyFilter        PermanentFilter
}

func (*temporaryDeathExileReplacement) survivesSourceExit() bool { return true }
func (*temporaryDeathExileReplacement) IsActive(GameReader) bool { return true }
func (r *temporaryDeathExileReplacement) Clone() ReplacementEffect {
	c := *r
	c.reference = cloneDamageSource(r.reference)
	c.historySource = cloneDamageSource(r.historySource)
	return &c
}
func (r *temporaryDeathExileReplacement) Matches(action Action, g GameReader) bool {
	var id uuid.UUID
	switch a := action.(type) {
	case *DestroyPermanentAction:
		id = a.PermanentID()
	case *SacrificePermanentAction:
		if a.Destination() != core.ZoneGraveyard {
			return false
		}
		id = a.PermanentID()
	case *permanentGraveyardAction:
		if a.destination != core.ZoneGraveyard {
			return false
		}
		id = a.subject
	default:
		return false
	}
	p := g.FindPermanent(id)
	if p == nil || p.PhasedOut {
		return false
	}
	if r.historySource.Incarnation() != uuid.Nil {
		game, ok := g.(*Game)
		return ok && r.historyFilter.Match(p, game) && game.damage.dealtDamageToPermanentThisTurn(r.historySource, capturedPermanent{p.ID(), p.incarnationID})
	}
	return p.ID() == r.subject && p.incarnationID == r.incarnation
}
func (r *temporaryDeathExileReplacement) Replace(action Action, g *Game) Action {
	if a, ok := action.(*SacrificePermanentAction); ok {
		return a.WithExile(r.sourceID)
	}
	subject, incarnation := r.subject, r.incarnation
	if r.historySource.Incarnation() != uuid.Nil {
		var id uuid.UUID
		switch a := action.(type) {
		case *DestroyPermanentAction:
			id = a.PermanentID()
		case *permanentGraveyardAction:
			id = a.subject
		}
		if p := g.FindPermanent(id); p != nil {
			subject, incarnation = p.ID(), p.incarnationID
		}
	}
	return &permanentGraveyardAction{subject: subject, incarnation: incarnation, destination: core.ZoneExile, exileSource: r.sourceID}
}

func (g *Game) performPermanentGraveyardAction(a *permanentGraveyardAction) {
	p := g.FindPermanent(a.subject)
	if p == nil || p.PhasedOut || p.incarnationID != a.incarnation {
		return
	}
	switch a.destination {
	case core.ZoneExile:
		g.exilePermanent(p, a.exileSource)
	case core.ZoneHand:
		g.bouncePermanentToHand(p, a.lockReturnedHand)
	default:
		g.putPermanentIntoGraveyard(p)
	}
}
