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
	colorsSpent                 *NumberBinding
	captureColorsSpent          bool
	duplicateColorsSpent        bool
	announcedX                  *NumberBinding
	captureAnnouncedX           bool
	duplicateAnnouncedX         bool
	filterXInManaCost           bool
	duplicateXInManaCost        bool
	spell                       *SpellBinding
	captureSpell                bool
	duplicateSpell              bool
	castZone                    Zone
	filterCastZone              bool
	duplicateCastZone           bool
	notOwnedByCaster            bool
	duplicateOwnership          bool
}

// SpellCast matches a spell cast by any player while this source is on the battlefield.
func SpellCast() spellCastPattern { return spellCastPattern{} }

// NotOwnedByCaster matches a cast spell whose card belongs to another player.
func (p spellCastPattern) NotOwnedByCaster() spellCastPattern {
	if p.notOwnedByCaster {
		p.duplicateOwnership = true
	}
	p.notOwnedByCaster = true
	return p
}

// FromZone matches the zone the spell left when it was cast.
func (p spellCastPattern) FromZone(zone Zone) spellCastPattern {
	if p.filterCastZone {
		p.duplicateCastZone = true
	}
	p.castZone, p.filterCastZone = zone, true
	return p
}

// WithSpell saves the cast spell's stack identity for this trigger.
func (p spellCastPattern) WithSpell(binding *SpellBinding) spellCastPattern {
	if p.captureSpell {
		p.duplicateSpell = true
	}
	p.spell, p.captureSpell = binding, true
	return p
}

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

// WithColorsSpent saves the number of mana colors spent to cast the spell.
func (p spellCastPattern) WithColorsSpent(spent *NumberBinding) spellCastPattern {
	if p.captureColorsSpent {
		p.duplicateColorsSpent = true
	}
	p.colorsSpent = spent
	p.captureColorsSpent = true
	return p
}

// WithAnnouncedX saves the cast spell's X value when it is cast.
func (p spellCastPattern) WithAnnouncedX(value *NumberBinding) spellCastPattern {
	if p.captureAnnouncedX {
		p.duplicateAnnouncedX = true
	}
	p.announcedX = value
	p.captureAnnouncedX = true
	return p
}

// WithXInManaCost matches a cast spell whose mana cost has an X symbol.
func (p spellCastPattern) WithXInManaCost() spellCastPattern {
	if p.filterXInManaCost {
		p.duplicateXInManaCost = true
	}
	p.filterXInManaCost = true
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
	if p.filterXInManaCost {
		t.AndConditionData(spellCastHasXInManaCost{})
	}
	if p.filterCastZone {
		t.AndConditionData(spellCastFromZone{zone: p.castZone})
	}
	if p.notOwnedByCaster {
		t.AndConditionData(spellCastNotOwnedByCaster{})
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
	if p.filterXInManaCost {
		n.Text += " with X in its mana cost"
		if p.duplicateXInManaCost {
			n.problem = "spell cast event: supply the X filter once"
		}
	}
	if p.filterCastZone {
		n.Text += " from " + strings.ToLower(p.castZone.String())
		if p.duplicateCastZone || !castOriginZone(p.castZone) {
			n.problem = "spell cast event: supply one valid cast zone"
		}
	}
	if p.notOwnedByCaster {
		n.Text += " and is not owned by its caster"
		if p.duplicateOwnership {
			n.problem = "spell cast event: supply the ownership filter once"
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
	if p.captureColorsSpent {
		n.bindingWrites = append(n.bindingWrites, p.colorsSpent)
		n.children = append(n.children, definitionEdge{"colors spent", &eventAmountCapture{binding: p.colorsSpent}})
		if p.colorsSpent == nil {
			n.problem = "spell cast event: supply a colors-spent binding"
		}
	}
	if p.duplicateColorsSpent {
		n.problem = "spell cast event: supply one colors-spent binding"
	}
	if p.captureAnnouncedX {
		n.bindingWrites = append(n.bindingWrites, p.announcedX)
		n.children = append(n.children, definitionEdge{"announced X", &eventAmountCapture{binding: p.announcedX}})
		if p.announcedX == nil {
			n.problem = "spell cast event: supply an announced-X binding"
		}
	}
	if p.duplicateAnnouncedX {
		n.problem = "spell cast event: supply one announced-X binding"
	}
	if p.captureSpell {
		if p.spell == nil || p.duplicateSpell {
			n.problem = "spell cast event: supply one spell binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.spell)
			n.children = append(n.children, definitionEdge{"spell", &eventSpellCapture{binding: p.spell}})
		}
	}
	return n
}

type spellCastHasType struct{ cardType CardType }

type spellCastFromZone struct{ zone Zone }

type spellCastNotOwnedByCaster struct{}

func (spellCastNotOwnedByCaster) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok || event.TargetID == uuid.Nil || event.PlayerID == uuid.Nil {
		return false
	}
	object := game.stack.FindByID(event.TargetID)
	return object != nil && !object.IsAbility && object.Card != nil && object.SourceID == event.SourceID && object.Controller == event.PlayerID && object.Card.Owner() != uuid.Nil && object.Card.Owner() != event.PlayerID
}

func castOriginZone(zone Zone) bool {
	switch zone {
	case ZoneLibrary, ZoneHand, ZoneBattlefield, ZoneGraveyard, ZoneExile, ZoneCommand, ZoneAnte:
		return true
	}
	return false
}

func (c spellCastFromZone) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok || event.TargetID == uuid.Nil {
		return false
	}
	object := game.stack.FindByID(event.TargetID)
	return object != nil && !object.IsAbility && object.Card != nil && object.SourceID == event.SourceID && object.CastZone == c.zone
}

type spellCastHasXInManaCost struct{}

func (spellCastHasXInManaCost) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	object := game.stack.FindByID(event.TargetID)
	return object != nil && !object.IsAbility && object.SourceID == event.SourceID && object.Card != nil && object.Card.ManaCost().HasX
}

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
	if !p.captureCaster && !p.captureManaSpent && !p.captureColorsSpent && !p.captureAnnouncedX && !p.captureSpell {
		return effects
	}
	captures := make([]ResolvingEffect, 0, 4)
	if p.captureCaster {
		captures = append(captures, &eventPlayerCapture{binding: p.caster, player: event.PlayerID})
	}
	if p.captureManaSpent {
		captures = append(captures, &eventAmountCapture{binding: p.manaSpent, amount: event.CastManaSpent})
	}
	if p.captureColorsSpent {
		captures = append(captures, &eventAmountCapture{binding: p.colorsSpent, amount: event.CastColorsSpent})
	}
	if p.captureAnnouncedX {
		captures = append(captures, &eventAmountCapture{binding: p.announcedX, amount: event.CastAnnouncedX})
	}
	if p.captureSpell {
		captures = append(captures, &eventSpellCapture{binding: p.spell, id: event.TargetID})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

type eventSpellCapture struct {
	binding *SpellBinding
	id      uuid.UUID
}

func (*eventSpellCapture) Text() string                 { return "remember cast spell" }
func (*eventSpellCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventSpellCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.spellBindings == nil {
		ctx.Game.resolution.spellBindings = make(map[*SpellBinding]uuid.UUID)
	}
	ctx.Game.resolution.spellBindings[e.binding] = e.id
	return nil
}
func (e *eventSpellCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-spell-capture", bindingWrite: e.binding, children: []definitionEdge{{"spell", e.binding}}}
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
