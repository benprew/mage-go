package mage

import (
	"maps"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// gameCloneContext preserves object identity within one simulation branch.
// Rules copies use a separate operation to reset execution state.
type gameCloneContext struct {
	games            map[*Game]*Game
	cards            map[*Card]*Card
	abilities        map[Ability]Ability
	targets          map[Target]Target
	permanents       map[*Permanent]*Permanent
	castContexts     map[*CastContext]*CastContext
	events           map[*GameEvent]*GameEvent
	effects          map[ContinuousEffect]ContinuousEffect
	cardStorage      []Card
	permanentStorage []Permanent
}

func newGameCloneContext() *gameCloneContext {
	return &gameCloneContext{}
}

func newGameCloneContextForGame(g *Game) *gameCloneContext {
	ctx := newGameCloneContext()
	permanentCount := len(g.zones.battlefield) + len(g.zones.lki)
	if g.zones.enteringPermanent != nil {
		permanentCount++
	}
	cardCount := permanentCount + len(g.zones.exile)
	for _, player := range g.players {
		cardCount += len(player.Hand()) + len(player.Library()) + len(player.Graveyard()) + len(player.Ante())
	}
	ctx.cardStorage = make([]Card, cardCount)
	ctx.permanentStorage = make([]Permanent, permanentCount)
	ctx.cards = make(map[*Card]*Card, cardCount)
	ctx.permanents = make(map[*Permanent]*Permanent, permanentCount)
	return ctx
}

func (ctx *gameCloneContext) card(card *Card) *Card {
	if card == nil {
		return nil
	}
	if cloned, ok := ctx.cards[card]; ok {
		return cloned
	}
	var cloned *Card
	if len(ctx.cardStorage) > 0 {
		cloned = &ctx.cardStorage[0]
		ctx.cardStorage = ctx.cardStorage[1:]
	} else {
		cloned = &Card{}
	}
	*cloned = *card
	if ctx.cards == nil {
		ctx.cards = make(map[*Card]*Card)
	}
	ctx.cards[card] = cloned
	cloned.capabilityAttrs = maps.Clone(card.capabilityAttrs)
	cloned.castTargets = ctx.targetList(card.castTargets)
	cloned.additionalCosts = slices.Clone(card.additionalCosts)
	cloned.alternateCosts = cloneAlternateCosts(card.alternateCosts)
	cloned.abilities = ctx.abilityList(card.abilities)
	cloned.cardCharacteristics = *card.clone()
	cloned.characteristics = card.characteristics.clone()
	if card.auraAIProfile != nil {
		profile := *card.auraAIProfile
		cloned.auraAIProfile = &profile
	}
	return cloned
}

func (ctx *gameCloneContext) cardList(cards []*Card) []*Card {
	if cards == nil {
		return nil
	}
	cloned := make([]*Card, len(cards))
	for i, card := range cards {
		cloned[i] = ctx.card(card)
	}
	return cloned
}

func (ctx *gameCloneContext) abilityList(abilities []Ability) []Ability {
	if abilities == nil {
		return nil
	}
	cloned := make([]Ability, len(abilities))
	for i, ability := range abilities {
		cloned[i] = ctx.ability(ability)
	}
	return cloned
}

func (ctx *gameCloneContext) permanent(permanent *Permanent) *Permanent {
	if permanent == nil {
		return nil
	}
	if cloned, ok := ctx.permanents[permanent]; ok {
		return cloned
	}
	var cloned *Permanent
	if len(ctx.permanentStorage) > 0 {
		cloned = &ctx.permanentStorage[0]
		ctx.permanentStorage = ctx.permanentStorage[1:]
	} else {
		cloned = &Permanent{}
	}
	if ctx.permanents == nil {
		ctx.permanents = make(map[*Permanent]*Permanent)
	}
	ctx.permanents[permanent] = cloned
	clonePermanentInto(cloned, permanent)
	cloned.Card = ctx.card(permanent.Card)
	cloned.RuntimeAbilities = ctx.abilityList(permanent.RuntimeAbilities)
	return cloned
}

func (ctx *gameCloneContext) castContext(context *CastContext) *CastContext {
	if context == nil {
		return nil
	}
	if cloned, ok := ctx.castContexts[context]; ok {
		return cloned
	}
	cloned := cloneCastContext(context)
	if ctx.castContexts == nil {
		ctx.castContexts = make(map[*CastContext]*CastContext)
	}
	ctx.castContexts[context] = cloned
	cloned.RevealedAtCast = ctx.cardList(context.RevealedAtCast)
	return cloned
}

func (ctx *gameCloneContext) event(event *GameEvent) *GameEvent {
	if event == nil {
		return nil
	}
	if cloned, ok := ctx.events[event]; ok {
		return cloned
	}
	cloned := *event
	if ctx.events == nil {
		ctx.events = make(map[*GameEvent]*GameEvent)
	}
	ctx.events[event] = &cloned
	return &cloned
}
