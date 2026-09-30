package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// A returned card must not supply colors for its previous battlefield visit.
type targetSourceLKI struct {
	Card
	colors []core.Color
}

func (g *Game) resolutionTargetSource(obj *StackObject) Card {
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
		return targetSourceLKI{Card: card, colors: colors}
	}
	if card == nil || card.ID() != obj.SourceID || obj.sourceIncarnation == uuid.Nil {
		return card
	}
	if p := g.FindPermanentIncludingPhased(obj.SourceID); p != nil && p.incarnationID == obj.sourceIncarnation {
		if p.PhasedOut {
			return targetSourceLKI{Card: damageSourceCardFromPermanent(p), colors: p.Colors()}
		}
		return card
	}
	if lki := g.zones.LKIByIncarnation(obj.sourceIncarnation); lki != nil && lki.Snapshot != nil && lki.ID == obj.SourceID {
		return targetSourceLKI{Card: damageSourceCardFromPermanent(lki.Snapshot), colors: lki.Snapshot.Colors()}
	}
	return card
}
