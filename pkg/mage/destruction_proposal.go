package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// InspectDestructionProposal reports whether one proposed destruction continues.
// The second result is false when replacement behavior is unknown. Layers must
// be current. The visit must already have passed subject and target selection.
// This query does not run callbacks, choose replacements or consume shields.
// It does not predict later zone replacements or sequential instructions.
func (g *Game) InspectDestructionProposal(visit PermanentVisit, cannotRegenerate bool) (proceeds, known bool) {
	if g == nil || visit.ID == uuid.Nil || visit.Incarnation == uuid.Nil {
		return false, false
	}
	p := g.FindPermanent(visit.ID)
	if p == nil || p.PhasedOut || p.incarnationID != visit.Incarnation || p.HasKeyword(core.Indestructible) {
		return false, true
	}
	canceled, regeneration := false, false
	if !p.FaceDown {
		for _, ability := range p.RuntimeAbilities {
			switch UnwrapAbility(ability).(type) {
			case *staticRegenerationAbility:
				regeneration = true
			case *deathReturnHandAbility:
				return false, false
			}
		}
	}
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if nilDefinition(replacement) {
				return false, false
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
					canceled = true
				}
			case *staticRegenerationReplacement:
				if r.sourceID == visit.ID && r.key.incarnation == visit.Incarnation && !p.FaceDown {
					for _, ability := range p.RuntimeAbilities {
						if _, ok := UnwrapAbility(ability).(*staticRegenerationAbility); ok && ability.AbilityID() == r.key.ability {
							regeneration = true
						}
					}
				}
			default:
				return false, false
			}
		}
	}
	if regeneration && !cannotRegenerate && !p.HasKeyword(core.CantRegenerate) {
		for _, effect := range g.effects.effects {
			if _, ok := effect.(*liveDamageRegenerationRule); ok {
				return false, false
			}
		}
		canceled = true
	}
	return !canceled, true
}
