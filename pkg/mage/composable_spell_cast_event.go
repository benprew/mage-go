package mage

import (
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type spellCastPattern struct {
	color                       Color
	filterColor                 bool
	duplicateColor              bool
	cardType                    CardType
	filterType                  bool
	duplicateType               bool
	enchantmentSubtype          string
	filterEnchantmentSubtype    bool
	duplicateEnchantmentSubtype bool
	opponentOnly                bool
	caster                      *PlayerBinding
	captureCaster               bool
	duplicateCaster             bool
	manaSpent                   *NumberBinding
	captureManaSpent            bool
	duplicateManaSpent          bool
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

// WithManaSpent saves the cast spell's paid mana total for this trigger.
func (p spellCastPattern) WithManaSpent(spent *NumberBinding) spellCastPattern {
	if p.captureManaSpent {
		p.duplicateManaSpent = true
	}
	p.manaSpent = spent
	p.captureManaSpent = true
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

// OfEnchantmentSubtype checks an enchantment spell's subtype when it is cast.
func (p spellCastPattern) OfEnchantmentSubtype(subtype string) spellCastPattern {
	if p.filterEnchantmentSubtype {
		p.duplicateEnchantmentSubtype = true
	}
	p.enchantmentSubtype = subtype
	p.filterEnchantmentSubtype = true
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
	if p.filterEnchantmentSubtype {
		t.AndConditionData(spellCastHasEnchantmentSubtype{subtype: p.enchantmentSubtype})
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
	if p.filterEnchantmentSubtype {
		n.Text += " with enchantment subtype " + p.enchantmentSubtype
		family, known := knownSubtypeFamily(p.enchantmentSubtype)
		if !known || family != SubtypeEnchantment || p.duplicateEnchantmentSubtype || (p.filterType && p.cardType != TypeEnchantment) {
			n.problem = "spell cast event: supply one compatible enchantment subtype"
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
	if p.captureManaSpent {
		n.bindingWrites = append(n.bindingWrites, p.manaSpent)
		n.children = append(n.children, definitionEdge{"mana spent", &eventAmountCapture{binding: p.manaSpent}})
		if p.manaSpent == nil {
			n.problem = "spell cast event: supply a mana-spent binding"
		}
	}
	if p.duplicateManaSpent {
		n.problem = "spell cast event: supply one mana-spent binding"
	}
	return n
}

type spellCastHasType struct{ cardType CardType }

type spellCastHasEnchantmentSubtype struct{ subtype string }

func (c spellCastHasEnchantmentSubtype) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
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
	return object != nil && !object.IsAbility && object.SourceID == event.SourceID && object.Card != nil && object.Card.HasType(TypeEnchantment) && object.Card.HasSubType(c.subtype)
}

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
	if !p.captureCaster && !p.captureManaSpent {
		return effects
	}
	captures := make([]ResolvingEffect, 0, 2)
	if p.captureCaster {
		captures = append(captures, &eventPlayerCapture{binding: p.caster, player: event.PlayerID})
	}
	if p.captureManaSpent {
		captures = append(captures, &eventAmountCapture{binding: p.manaSpent, amount: event.CastManaSpent})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
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
