package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// DestructionDestinations is the complete set of possible moves for one proposal.
// A destination describes the proposed move, including a token's move before
// it ceases to exist. It does not promise that a card remains in that zone.
type DestructionDestinations uint8

const (
	// DestructionNoMove includes canceled destruction and unavailable subjects.
	DestructionNoMove DestructionDestinations = 1 << iota
	// DestructionToGraveyard moves the subject to its owner's graveyard.
	DestructionToGraveyard
	// DestructionToExile replaces the graveyard destination with exile.
	DestructionToExile
	// DestructionToHand replaces the graveyard destination with its owner's hand.
	DestructionToHand
)

// InspectDestructionDestinations reads all possible destinations for one visit.
// The caller must first select the subject and check target legality. Layers
// must be current. Unsupported replacements return zero, false. No callback,
// choice, shield use or game change occurs. This is not sequential analysis.
func (g *Game) InspectDestructionDestinations(visit PermanentVisit, cannotRegenerate bool) (DestructionDestinations, bool) {
	if g == nil || visit.ID == uuid.Nil || visit.Incarnation == uuid.Nil {
		return 0, false
	}
	p := g.FindPermanent(visit.ID)
	if p == nil || p.PhasedOut || p.incarnationID != visit.Incarnation || p.HasKeyword(core.Indestructible) {
		return DestructionNoMove, true
	}
	var destinations DestructionDestinations
	regeneration := false
	if !p.FaceDown {
		for _, ability := range p.RuntimeAbilities {
			switch UnwrapAbility(ability).(type) {
			case *staticRegenerationAbility:
				regeneration = true
			case *deathReturnHandAbility:
				destinations |= DestructionToHand
			}
		}
	}
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if nilDefinition(replacement) {
				return 0, false
			}
			domains, known := InspectReplacementActionDomains(replacement)
			if known && domains&ReplacementDestructionActions == 0 {
				continue
			}
			switch r := replacement.(type) {
			case *regenerationReplacement:
				if r.permanentID == visit.ID && r.incarnation == visit.Incarnation && r.shields > 0 {
					regeneration = true
				}
			case *nextDestructionReplacement:
				if r.subject == visit.ID && r.incarnation == visit.Incarnation && !r.consumed {
					destinations |= DestructionNoMove
				}
			case *staticRegenerationReplacement, *deathReturnHandReplacement:
				// Current declarations above supply all active local candidates.
			case *temporaryDeathExileReplacement:
				if r.historySource.Incarnation() != uuid.Nil {
					matches, known := g.inspectDeathHistory(r.historySource, r.historyFilter, false, p)
					if !known {
						return 0, false
					}
					if matches {
						destinations |= DestructionToExile
					}
				} else if r.subject == visit.ID && r.incarnation == visit.Incarnation {
					destinations |= DestructionToExile
				}
			default:
				return 0, false
			}
		}
	}
	if regeneration && !cannotRegenerate && !p.HasKeyword(core.CantRegenerate) {
		allowed, known := g.inspectDeathHistoryRegeneration(p)
		if !known {
			return 0, false
		}
		if allowed {
			destinations |= DestructionNoMove
		}
	}
	// Each supported candidate ends the destruction or changes its destination.
	// No supported candidate can match after a hand or exile replacement.
	if destinations == 0 {
		destinations = DestructionToGraveyard
	}
	return destinations, true
}
