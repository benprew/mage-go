package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type landTappedForManaPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	land            *PermanentBinding
	capturePlayer   bool
	captureAmount   bool
	captureLand     bool
	duplicatePlayer bool
	duplicateAmount bool
	duplicateLand   bool
	manaType        Color
	filterManaType  bool
	duplicateType   bool
}

// LandTappedForMana matches a land mana ability that used a tap cost and produced mana.
func LandTappedForMana() landTappedForManaPattern { return landTappedForManaPattern{} }

// OfManaType matches mana of one type that the land ability produced.
func (p landTappedForManaPattern) OfManaType(manaType Color) landTappedForManaPattern {
	if p.filterManaType {
		p.duplicateType = true
	}
	p.manaType, p.filterManaType = manaType, true
	return p
}

// WithPlayer saves the player who activated the land's mana ability.
func (p landTappedForManaPattern) WithPlayer(binding *PlayerBinding) landTappedForManaPattern {
	if p.capturePlayer {
		p.duplicatePlayer = true
	}
	p.player, p.capturePlayer = binding, true
	return p
}

// WithAmount saves the amount of mana produced by the land ability.
func (p landTappedForManaPattern) WithAmount(binding *NumberBinding) landTappedForManaPattern {
	if p.captureAmount {
		p.duplicateAmount = true
	}
	p.amount, p.captureAmount = binding, true
	return p
}

// WithLand saves the land incarnation that produced the mana.
func (p landTappedForManaPattern) WithLand(binding *PermanentBinding) landTappedForManaPattern {
	if p.captureLand {
		p.duplicateLand = true
	}
	p.land, p.captureLand = binding, true
	return p
}

func (landTappedForManaPattern) eventPattern() {}

func (p landTappedForManaPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtTappedForMana, false, effect).SetConditionData(tappedForManaLandSource{manaType: p.manaType, filterManaType: p.filterManaType})
}

type tappedForManaLandSource struct {
	manaType       Color
	filterManaType bool
}

func (p tappedForManaLandSource) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	if p.filterManaType && !slices.Contains(event.ManaTypes, p.manaType) {
		return false
	}
	game, ok := reader.(*Game)
	if !ok || event.SourceIncarnation == uuid.Nil {
		return false
	}
	if land := game.FindPermanent(event.SourceID); land != nil && land.incarnationID == event.SourceIncarnation {
		return land.HasType(TypeLand)
	}
	if lki := game.zones.LKIByIncarnation(event.SourceIncarnation); lki != nil {
		return lki.HasType(TypeLand)
	}
	return false
}

func (p landTappedForManaPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "a land is tapped for mana", battlefieldSource: true}
	if p.filterManaType && (p.duplicateType || p.manaType < Colorless || p.manaType >= AnyColor) {
		n.problem = "land tapped for mana: supply one concrete mana type"
	}
	if p.capturePlayer {
		if p.player == nil || p.duplicatePlayer {
			n.problem = "land tapped for mana: supply one player binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.player)
			n.children = append(n.children, definitionEdge{"player", &eventPlayerCapture{binding: p.player}})
		}
	}
	if p.captureAmount {
		if p.amount == nil || p.duplicateAmount {
			n.problem = "land tapped for mana: supply one amount binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.amount)
			n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
		}
	}
	if p.captureLand {
		if p.land == nil || p.duplicateLand {
			n.problem = "land tapped for mana: supply one land binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.land)
			n.children = append(n.children, definitionEdge{"land", &eventPermanentCapture{binding: p.land}})
		}
	}
	return n
}

func (p landTappedForManaPattern) captureDetectedEvent(event *GameEvent, _ GameReader, _ uuid.UUID) []Effect {
	if !p.captureLand || p.land == nil || event.SourceIncarnation == uuid.Nil {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.land, value: capturedPermanent{event.SourceID, event.SourceIncarnation}}}
}

func (p landTappedForManaPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.capturePlayer && !p.captureAmount {
		return effects
	}
	var captures []ResolvingEffect
	if p.capturePlayer {
		captures = append(captures, &eventPlayerCapture{binding: p.player, player: event.PlayerID})
	}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{binding: p.amount, amount: event.Amount})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
