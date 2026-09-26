package mage

import (
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type spellCastPattern struct {
	color           Color
	filterColor     bool
	duplicateColor  bool
	cardType        CardType
	filterType      bool
	duplicateType   bool
	opponentOnly    bool
	caster          *PlayerBinding
	captureCaster   bool
	duplicateCaster bool
}

// SpellCast matches a spell cast by any player while this source is on the battlefield.
func SpellCast() spellCastPattern { return spellCastPattern{} }

// ByOpponent checks the caster against the source controller when the spell is cast.
func (p spellCastPattern) ByOpponent() spellCastPattern {
	p.opponentOnly = true
	return p
}

// WithCaster saves the player who cast the spell for this trigger instance.
func (p spellCastPattern) WithCaster(caster *PlayerBinding) spellCastPattern {
	if p.captureCaster {
		p.duplicateCaster = true
	}
	p.caster = caster
	p.captureCaster = true
	return p
}

// OfColor checks the spell's effective color when it is cast.
func (p spellCastPattern) OfColor(color Color) spellCastPattern {
	if p.filterColor {
		p.duplicateColor = true
	}
	p.color = color
	p.filterColor = true
	return p
}

// OfType checks the spell's type when it is cast.
func (p spellCastPattern) OfType(cardType CardType) spellCastPattern {
	if p.filterType {
		p.duplicateType = true
	}
	p.cardType = cardType
	p.filterType = true
	return p
}

func (spellCastPattern) eventPattern() {}

func (p spellCastPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtSpellCast, false, effect)
	if p.opponentOnly {
		t.SetConditionData(EventPlayerIsNotController{})
	}
	if p.filterColor {
		t.AndConditionData(spellCastHasEffectiveColor{color: p.color})
	}
	if p.filterType {
		t.AndConditionData(spellCastHasType{cardType: p.cardType})
	}
	return t
}

func (p spellCastPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "a spell is cast", battlefieldSource: true}
	if p.filterColor {
		n.Text = "a " + strings.ToLower(p.color.String()) + " spell is cast"
		if p.duplicateColor || p.color < White || p.color > Green {
			n.problem = "spell cast event: supply one valid color"
		}
	}
	if p.opponentOnly {
		if p.filterColor {
			n.Text = "an opponent casts a " + strings.ToLower(p.color.String()) + " spell"
		} else {
			n.Text = "an opponent casts a spell"
		}
	}
	if p.filterType {
		n.Text += " with type " + strings.ToLower(p.cardType.String())
		if p.duplicateType || p.cardType < TypeCreature || p.cardType > TypePlaneswalker || p.cardType == TypeLand {
			n.problem = "spell cast event: supply one valid spell type"
		}
	}
	if p.caster != nil {
		n.bindingWrite = p.caster
		n.children = append(n.children, definitionEdge{"caster", &eventPlayerCapture{binding: p.caster}})
	} else if p.captureCaster {
		n.problem = "spell cast event: supply a caster binding"
	}
	if p.duplicateCaster {
		n.problem = "spell cast event: supply one caster binding"
	}
	return n
}

type spellCastHasType struct{ cardType CardType }

func (c spellCastHasType) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	stackID := event.TargetID
	if stackID == uuid.Nil {
		object := game.FindStackObject(event.SourceID)
		if object == nil {
			return false
		}
		stackID = object.ID
	}
	object := game.stack.FindByID(stackID)
	return object != nil && !object.IsAbility && object.SourceID == event.SourceID && object.Card != nil && object.Card.HasType(c.cardType)
}

func (p spellCastPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureCaster {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.caster, player: event.PlayerID}, Sequence(effects...))}
}

type spellCastHasEffectiveColor struct{ color Color }

func (c spellCastHasEffectiveColor) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	stackID := event.TargetID
	if stackID == uuid.Nil {
		object := game.FindStackObject(event.SourceID)
		if object == nil {
			return false
		}
		stackID = object.ID
	}
	object := game.stack.FindByID(stackID)
	return object != nil && !object.IsAbility && object.SourceID == event.SourceID && slices.Contains(game.EffectiveColors(stackID), c.color)
}
