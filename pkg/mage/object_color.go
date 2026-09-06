package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func cardColors(card Card) []Color {
	if card == nil {
		return nil
	}
	if base, ok := card.(*BaseCard); ok && len(base.colorOverride) > 0 {
		return append([]Color(nil), base.colorOverride...)
	}
	return card.ManaCost().Colors()
}

// EffectiveColors returns an object's current colors. objectID may identify a
// permanent, a stack object, or the source card of a spell on the stack.
func (g *Game) EffectiveColors(objectID uuid.UUID) []Color {
	if permanent := g.FindPermanent(objectID); permanent != nil {
		return append([]Color(nil), permanent.Colors()...)
	}
	object := g.stack.FindByID(objectID)
	if object == nil {
		object = g.stack.FindBySourceID(objectID)
	}
	if object != nil {
		if object.ColorOverride != nil {
			return append([]Color(nil), (*object.ColorOverride)...)
		}
		return cardColors(object.Card)
	}
	if objectID == g.resolution.ColorSourceID() && g.resolution.ColorOverride() != nil {
		return append([]Color(nil), (*g.resolution.ColorOverride())...)
	}
	return cardColors(g.FindCardAnywhere(objectID))
}

// ChangeObjectColor assigns one color to a permanent or spell. A spell keeps
// the change if it resolves as a permanent; the effect ends when that object
// next changes zones.
func (g *Game) ChangeObjectColor(objectID uuid.UUID, color Color) bool {
	if permanent := g.FindPermanent(objectID); permanent != nil {
		g.changePermanentColors(permanent, []Color{color}, uuid.Nil)
		return true
	}
	object := g.stack.FindByID(objectID)
	if object == nil {
		object = g.stack.FindBySourceID(objectID)
	}
	if object == nil || object.Card == nil || object.IsAbility {
		return false
	}
	colors := []Color{color}
	object.ColorOverride = &colors
	return true
}

func (g *Game) changePermanentColors(permanent *Permanent, colors []Color, sourceID uuid.UUID) {
	g.AddContinuousEffect(permanentColorEffect(permanent, colors, sourceID))
}

type objectColorEffect struct {
	targetID      uuid.UUID
	incarnationID uuid.UUID
	colors        []Color
	effectSource
}

func (e *objectColorEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *objectColorEffect) Text() string                 { return "" }
func (e *objectColorEffect) GetLayer() Layer              { return LayerColor }
func (e *objectColorEffect) GetDuration() Duration        { return Indefinite }
func (e *objectColorEffect) IsActive(g *Game) bool {
	target := g.FindPermanentIncludingPhased(e.targetID)
	return target != nil && target.incarnationID == e.incarnationID
}
func (e *objectColorEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	target := g.MutablePermanentIncludingPhased(e.targetID)
	if target == nil || target.incarnationID != e.incarnationID {
		return nil
	}
	override := append([]Color(nil), e.colors...)
	target.ColorOverride = &override
	return nil
}

func permanentColorEffect(permanent *Permanent, colors []Color, sourceID uuid.UUID) ContinuousEffect {
	effect := &objectColorEffect{
		targetID:      permanent.ID(),
		incarnationID: permanent.incarnationID,
		colors:        append([]Color(nil), colors...),
	}
	effect.SetSourceID(sourceID)
	return effect
}
