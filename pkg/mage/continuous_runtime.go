package mage

import (
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type activeContinuousEffect struct {
	ContinuousEffect
	sourceID                  uuid.UUID
	controllerID              uuid.UUID
	grantID                   uuid.UUID
	printed                   bool // registered via AddPrinted
	abilityIndex, effectIndex int
	expired                   bool
	applied                   bool
	initialized               bool
	originalValues            CopiableValues
}

func (e *activeContinuousEffect) SourceID() uuid.UUID { return e.sourceID }
func (e *activeContinuousEffect) context(g *Game, latch bool) *EffectContext {
	controller := e.controllerID
	if e.printed {
		if permanent := g.FindPermanent(e.sourceID); permanent != nil {
			controller = permanent.ControllerID()
		}
	}
	return &EffectContext{
		Game:            g,
		SourceID:        e.sourceID,
		Controller:      controller,
		continuous:      e,
		latchExpiration: latch,
	}
}
func (e *activeContinuousEffect) active(g *Game, latch bool) bool {
	if e.printed {
		source := g.FindPermanent(e.sourceID)
		if source == nil || source.printedAbilitiesSuppressed {
			return false
		}
		present := false
		for _, ability := range source.RuntimeAbilities {
			if static, ok := UnwrapAbility(ability).(*StaticAbilityHolder); ok {
				for _, definition := range static.Effects {
					if definition == e.ContinuousEffect {
						present = true
					}
				}
			}
		}
		if !present {
			return false
		}
	}
	return e.Active(e.context(g, latch))
}
func (e *activeContinuousEffect) IsActive(g *Game) bool { return e.active(g, true) }
func (e *activeContinuousEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	prevGrantID := g.effects.activeGrantID
	g.effects.activeGrantID = e.grantID
	err := e.ContinuousEffect.Apply(e.context(g, true))
	g.effects.activeGrantID = prevGrantID
	if e.GetLayer() == LayerAbility {
		for _, permanent := range g.zones.battlefield {
			for _, ability := range permanent.RuntimeAbilities {
				if granted, ok := ability.(*grantedByEffect); ok && granted.grantID == uuid.Nil() {
					granted.grantID = e.grantID
				}
			}
		}
	}
	return err
}
