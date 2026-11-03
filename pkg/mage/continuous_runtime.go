package mage

import "uuid"

type activeContinuousEffect struct {
	ContinuousEffect
	sourceID                  uuid.UUID
	controllerID              uuid.UUID
	controllerBound           bool
	grantID                   uuid.UUID
	printed                   bool
	abilityIndex, effectIndex int
	expired                   bool
	applied                   bool
	initialized               bool
	originalValues            CopiableValues
}

func (e *activeContinuousEffect) SourceID() uuid.UUID { return e.sourceID }
func (e *activeContinuousEffect) context(g *Game, latch bool) *EffectContext {
	ctx := NewEffectContext(g, e.sourceID, uuid.Nil(), nil)
	ctx.continuous = e
	ctx.latchExpiration = latch
	if e.printed {
		if permanent := g.FindPermanent(e.sourceID); permanent != nil {
			ctx.Controller = permanent.ControllerID()
		}
	} else {
		if !e.controllerBound {
			if g.resolution.resolvingSourceID == e.sourceID {
				e.controllerID = g.resolution.resolvingController
			} else if permanent := g.FindPermanent(e.sourceID); permanent != nil {
				e.controllerID = permanent.ControllerID()
			}
			e.controllerBound = e.controllerID != uuid.Nil()
		}
		ctx.Controller = e.controllerID
	}
	return ctx
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
	err := e.ContinuousEffect.Apply(e.context(ctx.Game, true))
	for _, permanent := range ctx.Game.zones.battlefield {
		for _, ability := range permanent.RuntimeAbilities {
			if granted, ok := ability.(*grantedByEffect); ok && granted.grantID == uuid.Nil() {
				granted.grantID = e.grantID
			}
		}
	}
	return err
}
