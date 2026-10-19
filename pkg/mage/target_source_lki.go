package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// A returned card must not supply characteristics for its previous battlefield visit.
type targetSourceLKI struct {
	Card
	abilityDeclaration     *targetingAbilityDeclaration
	spellDeclarations      []Target
	spellDeclarationsKnown bool
	incarnation            uuid.UUID
	colors                 []core.Color
	power                  int
	powerKnown             bool
	spell                  bool
}

func (g *Game) resolutionTargetSource(obj *StackObject) Card {
	result := g.targetSourceInformation(obj)
	if result == nil {
		return nil
	}
	if !obj.IsAbility {
		return spellDeclarationSource(obj, result)
	}
	view, ok := obj.TargetSource.(targetSourceLKI)
	if !ok || view.abilityDeclaration == nil {
		return result
	}
	retained, ok := result.(targetSourceLKI)
	if !ok {
		retained = targetSourceLKI{Card: result, colors: cardColors(result)}
	}
	retained.abilityDeclaration = view.abilityDeclaration
	return retained
}

func (g *Game) targetSourceInformation(obj *StackObject) Card {
	card := obj.Card
	if obj.TargetSource != nil {
		card = obj.TargetSource
	}
	// A resolving spell or new copy can be absent from the stack lookup.
	if !obj.IsAbility && obj.Card != nil && card != nil && card.ID() == obj.SourceID {
		colors := cardColors(card)
		if obj.ColorOverride != nil {
			colors = append([]core.Color(nil), (*obj.ColorOverride)...)
		}
		return targetSourceLKI{Card: card, colors: colors, spell: true}
	}
	if card == nil || card.ID() != obj.SourceID {
		return card
	}
	if obj.sourceIncarnation == uuid.Nil {
		if obj.IsAbility {
			return targetSourceLKI{Card: card, colors: cardColors(card)}
		}
		return card
	}
	if p := g.FindPermanentIncludingPhased(obj.SourceID); p != nil && p.incarnationID == obj.sourceIncarnation {
		if p.PhasedOut {
			power := p.CurrentPower(g)
			if lki := g.zones.phaseInformation[obj.sourceIncarnation]; lki != nil {
				power = lki.Power
			}
			return targetSourceLKI{Card: damageSourceCardFromPermanent(p), colors: p.Colors(), power: power, powerKnown: true, incarnation: obj.sourceIncarnation}
		}
		return targetSourceLKI{Card: damageSourceCardFromPermanent(p), colors: p.Colors(), incarnation: obj.sourceIncarnation}
	}
	if lki := g.zones.LKIByIncarnation(obj.sourceIncarnation); lki != nil && lki.Snapshot != nil && lki.ID == obj.SourceID {
		return targetSourceLKI{Card: damageSourceCardFromPermanent(lki.Snapshot), colors: lki.Snapshot.Colors(), power: lki.Power, powerKnown: true, incarnation: obj.sourceIncarnation}
	}
	return targetSourceLKI{Card: card, colors: cardColors(card), incarnation: obj.sourceIncarnation}
}

// SpellTargetSource identifies a spell during target queries before stack entry.
// Ability sources keep their existing information view.
func SpellTargetSource(card Card) Card {
	if card == nil {
		return nil
	}
	view, ok := card.(targetSourceLKI)
	if !ok {
		view = targetSourceLKI{Card: card, colors: cardColors(card)}
	}
	view.spell = true
	view.abilityDeclaration = nil
	if !view.spellDeclarationsKnown {
		if _, modal := getModalSpellAbility(card); !modal {
			view.spellDeclarations = slices.Clone(card.CastTargets())
			view.spellDeclarationsKnown = true
		}
	}
	return view
}

func (g *Game) triggerTargetSource(pt *pendingTrigger) Card {
	card := g.FindCardAnywhere(pt.sourceID)
	if card == nil && pt.sourceIncarnation != uuid.Nil {
		if lki := g.zones.LKIByIncarnation(pt.sourceIncarnation); lki != nil && lki.Snapshot != nil {
			card = lki.Snapshot.Card
		}
	}
	return g.resolutionTargetSource(&StackObject{TargetSource: card, SourceID: pt.sourceID, sourceIncarnation: pt.sourceIncarnation, IsAbility: true})
}

// Source exclusion must not apply to a later battlefield visit of the same card.
func targetSourceIsPermanent(source Card, permanent *Permanent) bool {
	if source == nil || permanent == nil || source.ID() != permanent.ID() {
		return false
	}
	if view, ok := source.(targetSourceLKI); ok {
		if view.spell {
			return false
		}
		if view.incarnation != uuid.Nil {
			return view.incarnation == permanent.incarnationID
		}
	}
	return true
}
