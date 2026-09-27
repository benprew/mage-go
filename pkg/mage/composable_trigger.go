package mage

import (
	"strconv"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// EventPattern describes an event at detection time.
type EventPattern interface {
	eventPattern()
	makeTrigger(ResolvingEffect) *GenericTriggered
}
type sourceEventPattern struct {
	dies   bool
	leaves bool
}

func (sourceEventPattern) eventPattern() {}

// Enters matches the source entering the battlefield.
func Enters(sourceRef) EventPattern { return sourceEventPattern{} }

// Dies matches the source leaving the battlefield for the graveyard.
func Dies(sourceRef) EventPattern { return sourceEventPattern{dies: true} }

// LeavesBattlefield matches the source moving from the battlefield to another zone.
func LeavesBattlefield(sourceRef) EventPattern { return sourceEventPattern{leaves: true} }
func (p sourceEventPattern) makeTrigger(e ResolvingEffect) *GenericTriggered {
	if p.dies {
		return DiesTrigger(e, false)
	}
	if p.leaves {
		return OnLeaveZone(ZoneBattlefield, ZoneAny, e, false)
	}
	return EntersBattlefieldTrigger(e, false)
}
func (p sourceEventPattern) inspectDefinition() definitionNode {
	label := "enters"
	if p.dies {
		label = "dies"
	} else if p.leaves {
		label = "source leaves battlefield"
	}
	return definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true}
}

type creatureEntersPattern struct {
	controllerOnly   bool
	otherOnly        bool
	nontokenOnly     bool
	subtype          string
	filterSubtype    bool
	keyword          Keyword
	filterKeyword    bool
	duplicateKeyword bool
	permanent        *PermanentBinding
	capturePermanent bool
	power            *NumberBinding
	capturePower     bool
	duplicatePower   bool
}

// CreatureYouControlEnters matches a creature that enters under your control.
func CreatureYouControlEnters() creatureEntersPattern {
	return creatureEntersPattern{controllerOnly: true}
}

// AnotherCreatureYouControlEnters matches a different creature that enters under your control.
func AnotherCreatureYouControlEnters() creatureEntersPattern {
	return creatureEntersPattern{controllerOnly: true, otherOnly: true}
}

// Nontoken limits the pattern to creatures that enter as nontoken permanents.
func (p creatureEntersPattern) Nontoken() creatureEntersPattern {
	p.nontokenOnly = true
	return p
}

// OfSubtype limits the pattern to a creature with this subtype as it enters.
func (p creatureEntersPattern) OfSubtype(subtype string) creatureEntersPattern {
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// HasKeyword checks the creature's effective keyword when it enters.
func (p creatureEntersPattern) HasKeyword(keyword Keyword) creatureEntersPattern {
	if p.filterKeyword {
		p.duplicateKeyword = true
	}
	p.keyword, p.filterKeyword = keyword, true
	return p
}

// WithPermanent saves the entering battlefield incarnation for the trigger.
func (p creatureEntersPattern) WithPermanent(permanent *PermanentBinding) creatureEntersPattern {
	p.permanent = permanent
	p.capturePermanent = true
	return p
}

// WithPower saves the creature's power when it enters.
func (p creatureEntersPattern) WithPower(power *NumberBinding) creatureEntersPattern {
	if p.capturePower {
		p.duplicatePower = true
	}
	p.power = power
	p.capturePower = true
	return p
}

func (creatureEntersPattern) eventPattern() {}

func (p creatureEntersPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtZoneChange, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
		EventSourceWasOfType{Type: TypeCreature},
	}})
	if p.controllerOnly {
		t.AndConditionData(EventSourceControlledByController{})
	}
	if p.otherOnly {
		t.AndConditionData(EventSourceNotSelf{})
	}
	if p.nontokenOnly {
		t.AndConditionData(eventSourceWasNontoken{})
	}
	if p.filterSubtype {
		t.AndConditionData(eventSourceHasEntrySubtype{subtype: p.subtype})
	}
	if p.filterKeyword {
		t.AndConditionData(eventSourceHasEntryKeyword{keyword: p.keyword})
	}
	return t
}

func (p creatureEntersPattern) inspectDefinition() definitionNode {
	text := "a creature enters"
	if p.otherOnly {
		text = "another creature enters"
	}
	if p.controllerOnly {
		text = strings.Replace(text, " enters", " you control enters", 1)
	}
	if p.nontokenOnly {
		text = strings.Replace(text, "creature", "nontoken creature", 1)
	}
	if p.filterSubtype {
		text = strings.Replace(text, "creature", p.subtype+" creature", 1)
	}
	if p.filterKeyword {
		text += " with " + p.keyword.String()
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true}
	if p.filterKeyword && (p.duplicateKeyword || p.keyword < Flying || p.keyword > AttrPrepared) {
		n.problem = "entry event: supply one valid keyword"
	}
	if p.filterSubtype {
		if p.subtype == "" {
			n.problem = "entry event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "entry event: subtype is not a creature subtype"
		}
	}
	if p.permanent != nil {
		n.bindingWrite = p.permanent
		n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
	} else if p.capturePermanent {
		n.problem = "entry event: supply a permanent binding"
	}
	if p.power != nil {
		n.bindingWrites = append(n.bindingWrites, p.power)
		n.children = append(n.children, definitionEdge{"power", &eventAmountCapture{binding: p.power}})
	} else if p.capturePower {
		n.problem = "entry event: supply a power binding"
	}
	if p.duplicatePower {
		n.problem = "entry event: supply one power binding"
	}
	return n
}

type eventSourceHasEntrySubtype struct{ subtype string }

type eventSourceHasEntryKeyword struct{ keyword Keyword }

func (c eventSourceHasEntryKeyword) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	permanent := reader.FindPermanent(event.SourceID)
	return permanent != nil && permanent.HasKeyword(c.keyword)
}

func (c eventSourceHasEntrySubtype) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	permanent := reader.FindPermanent(event.SourceID)
	return permanent != nil && permanent.HasSubType(c.subtype)
}

func (p creatureEntersPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.capturePermanent && !p.capturePower {
		return nil
	}
	permanent := observer.FindPermanent(event.SourceID)
	if permanent == nil {
		return nil
	}
	var captures []Effect
	if p.capturePermanent && p.permanent != nil {
		captures = append(captures, &eventPermanentCapture{binding: p.permanent, value: capturedPermanent{permanent.ID(), permanent.incarnationID}})
	}
	if p.capturePower && p.power != nil {
		if game, ok := observer.(*Game); ok {
			captures = append(captures, &eventAmountCapture{binding: p.power, amount: permanent.CurrentPower(game)})
		}
	}
	return captures
}

type creatureDiesPattern struct {
	creatureOnly      bool
	permanentType     CardType
	filterType        bool
	duplicateType     bool
	subtype           string
	filterSubtype     bool
	duplicateSubtype  bool
	controllerOnly    bool
	duplicateControl  bool
	opponentOnly      bool
	otherOnly         bool
	nontokenOnly      bool
	attachedOnly      bool
	controller        *PlayerBinding
	captureController bool
	owner             *PlayerBinding
	captureOwner      bool
	power             *NumberBinding
	capturePower      bool
	duplicatePower    bool
	toughness         *NumberBinding
	captureToughness  bool
	card              *CardBinding
	captureCard       bool
	excludedColor     Color
	filterColor       bool
	duplicateColor    bool
	notSacrificed     bool
	duplicateOutcome  bool
	maxPowerToughness int
	filterPT          bool
	duplicatePT       bool
	damagedByThisTurn bool
	duplicateDamage   bool
}

// CreatureDies matches any creature moving from the battlefield to a graveyard.
// It includes the source and uses the creature's type before it leaves.
func CreatureDies() creatureDiesPattern { return creatureDiesPattern{creatureOnly: true} }

// PermanentDies matches a permanent moving from the battlefield to a graveyard.
func PermanentDies() creatureDiesPattern { return creatureDiesPattern{} }

// YouControlled limits a death pattern to a permanent you controlled before it died.
func (p creatureDiesPattern) YouControlled() creatureDiesPattern {
	if p.controllerOnly {
		p.duplicateControl = true
	}
	p.controllerOnly = true
	return p
}

// OfType limits a permanent-death pattern to one permanent card type.
func (p creatureDiesPattern) OfType(cardType CardType) creatureDiesPattern {
	if p.filterType || p.creatureOnly {
		p.duplicateType = true
	}
	p.permanentType = cardType
	p.filterType = true
	return p
}

// OfSubtype matches the creature subtype before the creature dies.
func (p creatureDiesPattern) OfSubtype(subtype string) creatureDiesPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

func permanentCardType(cardType CardType) bool {
	switch cardType {
	case TypeCreature, TypeLand, TypeArtifact, TypeEnchantment, TypePlaneswalker:
		return true
	default:
		return false
	}
}

// CreatureYouControlDies matches a creature controlled by you before it dies.
func CreatureYouControlDies() creatureDiesPattern {
	return creatureDiesPattern{creatureOnly: true, controllerOnly: true}
}

// OpponentCreatureDies matches a creature an opponent controlled before death.
func OpponentCreatureDies() creatureDiesPattern {
	return creatureDiesPattern{creatureOnly: true, opponentOnly: true}
}

// AnotherCreatureYouControlDies matches a different creature controlled by you before it dies.
func AnotherCreatureYouControlDies() creatureDiesPattern {
	return creatureDiesPattern{creatureOnly: true, controllerOnly: true, otherOnly: true}
}

// AnotherCreatureDies matches a creature other than the source when it dies.
func AnotherCreatureDies() creatureDiesPattern {
	return creatureDiesPattern{creatureOnly: true, otherOnly: true}
}

// Nontoken limits the pattern to nontoken permanents before death.
func (p creatureDiesPattern) Nontoken() creatureDiesPattern {
	p.nontokenOnly = true
	return p
}

// DamagedByThisTurn matches a creature damaged by this source this turn.
func (p creatureDiesPattern) DamagedByThisTurn() creatureDiesPattern {
	if p.damagedByThisTurn {
		p.duplicateDamage = true
	}
	p.damagedByThisTurn = true
	return p
}

// WithoutColor limits the pattern to creatures without this color before death.
func (p creatureDiesPattern) WithoutColor(color Color) creatureDiesPattern {
	if p.filterColor {
		p.duplicateColor = true
	}
	p.excludedColor = color
	p.filterColor = true
	return p
}

// NotSacrificed excludes a death caused by a sacrifice action.
func (p creatureDiesPattern) NotSacrificed() creatureDiesPattern {
	if p.notSacrificed {
		p.duplicateOutcome = true
	}
	p.notSacrificed = true
	return p
}

// PowerOrToughnessAtMost checks the creature's power or toughness before death.
func (p creatureDiesPattern) PowerOrToughnessAtMost(maximum int) creatureDiesPattern {
	if p.filterPT {
		p.duplicatePT = true
	}
	p.maxPowerToughness = maximum
	p.filterPT = true
	return p
}

// AttachedCreatureDies matches the creature attached to this source when it dies.
func AttachedCreatureDies() creatureDiesPattern {
	return creatureDiesPattern{creatureOnly: true, attachedOnly: true}
}

// WithController saves the permanent's controller before it dies.
func (p creatureDiesPattern) WithController(controller *PlayerBinding) creatureDiesPattern {
	p.controller = controller
	p.captureController = true
	return p
}

// WithOwner saves the permanent's owner when the death event occurs.
func (p creatureDiesPattern) WithOwner(owner *PlayerBinding) creatureDiesPattern {
	p.owner = owner
	p.captureOwner = true
	return p
}

// WithPower saves the creature's power before it dies.
func (p creatureDiesPattern) WithPower(power *NumberBinding) creatureDiesPattern {
	if p.capturePower {
		p.duplicatePower = true
	}
	p.power = power
	p.capturePower = true
	return p
}

// WithToughness saves the creature's toughness before it dies.
func (p creatureDiesPattern) WithToughness(toughness *NumberBinding) creatureDiesPattern {
	p.toughness = toughness
	p.captureToughness = true
	return p
}

// WithCard saves the card's first graveyard visit after this death.
func (p creatureDiesPattern) WithCard(card *CardBinding) creatureDiesPattern {
	p.card = card
	p.captureCard = true
	return p
}

func (creatureDiesPattern) eventPattern() {}
func (p creatureDiesPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtZoneChange, false, effect).
		SetConditionData(EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard})
	if p.creatureOnly {
		t.AndConditionData(EventSourceWasOfType{Type: TypeCreature})
	}
	if p.filterType {
		t.AndConditionData(EventSourceWasOfType{Type: p.permanentType})
	}
	if p.filterSubtype {
		t.AndConditionData(EventSourceHasSubType{SubType: p.subtype})
	}
	if p.controllerOnly {
		t.AndConditionData(EventPlayerIsController{})
	}
	if p.opponentOnly {
		t.AndConditionData(EventPlayerIsOpponent{})
	}
	if p.otherOnly {
		t.AndConditionData(EventSourceNotSelf{})
	}
	if p.nontokenOnly {
		t.AndConditionData(eventSourceWasNontoken{})
	}
	if p.filterColor {
		t.AndConditionData(EventSourceWasNotColor{Color: p.excludedColor})
	}
	if p.notSacrificed {
		t.AndConditionData(deathWasNotSacrifice{})
	}
	if p.filterPT {
		t.AndConditionData(eventSourcePowerOrToughnessAtMost{maximum: p.maxPowerToughness})
	}
	if p.attachedOnly {
		t.AndConditionData(sourceAttachedToDyingCreature{})
	}
	if p.damagedByThisTurn {
		t.AndConditionData(eventCreatureDamagedBySourceThisTurn{})
	}
	return t
}

type eventCreatureDamagedBySourceThisTurn struct{}

func (eventCreatureDamagedBySourceThisTurn) CheckTriggerCond(event *GameEvent, reader GameReader, sourceID, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok || event.SourceIncarnation == uuid.Nil {
		return false
	}
	source := game.CaptureDamageSource(sourceID)
	return game.damage.dealtDamageToPermanentThisTurn(source, capturedPermanent{event.SourceID, event.SourceIncarnation})
}

type eventSourceWasNontoken struct{}

type deathWasNotSacrifice struct{}

type eventSourcePowerOrToughnessAtMost struct{ maximum int }

func (c eventSourcePowerOrToughnessAtMost) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	view := game.LookupObject(event.SourceID)
	return view != nil && (view.ViewPower() <= c.maximum || view.ViewToughness() <= c.maximum)
}

func (deathWasNotSacrifice) CheckTriggerCond(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return !event.Flag
}

func (eventSourceWasNontoken) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	view := game.LookupObject(event.SourceID)
	return view != nil && !view.ViewIsToken()
}

type sourceAttachedToDyingCreature struct{}

func (sourceAttachedToDyingCreature) CheckTriggerCond(event *GameEvent, reader GameReader, sourceID, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	source := game.FindPermanent(sourceID)
	if source == nil || source.PhasedOut || source.AttachedTo != event.SourceID {
		return false
	}
	var incarnation uuid.UUID
	if creature := game.FindPermanent(event.SourceID); creature != nil {
		incarnation = creature.incarnationID
	} else if lki := game.zones.LKI(event.SourceID); lki != nil && lki.Snapshot != nil {
		incarnation = lki.Snapshot.incarnationID
	}
	return incarnation != uuid.Nil && source.attachedIncarnation == incarnation
}
func (p creatureDiesPattern) inspectDefinition() definitionNode {
	subject := "permanent"
	if p.creatureOnly {
		subject = "creature"
	} else if p.filterType && permanentCardType(p.permanentType) {
		subject = strings.ToLower(p.permanentType.String())
	}
	text := "a " + subject + " dies"
	if p.controllerOnly {
		text = "a " + subject + " you control dies"
	}
	if p.opponentOnly {
		text = "a " + subject + " an opponent controls dies"
	}
	if p.otherOnly {
		text = "another " + subject + " dies"
		if p.controllerOnly {
			text = "another " + subject + " you control dies"
		}
	}
	if p.attachedOnly {
		text = "attached creature dies"
	}
	if p.nontokenOnly {
		text = strings.Replace(text, subject, "nontoken "+subject, 1)
	}
	if p.filterSubtype {
		text = strings.Replace(text, subject, p.subtype+" "+subject, 1)
	}
	if p.filterColor {
		text = strings.Replace(text, subject, "non"+strings.ToLower(p.excludedColor.String())+" "+subject, 1)
	}
	if p.notSacrificed {
		text += " and was not sacrificed"
	}
	if p.filterPT {
		text += " with power or toughness " + strconv.Itoa(p.maxPowerToughness) + " or less"
	}
	if p.damagedByThisTurn {
		text += " after damage by this source this turn"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true}
	if p.filterColor && (p.excludedColor < White || p.excludedColor > Green || p.duplicateColor) {
		n.problem = "death event: supply one valid color"
	}
	if p.duplicateType || (p.filterType && !permanentCardType(p.permanentType)) {
		n.problem = "death event: supply one permanent card type"
	}
	if p.filterSubtype {
		if p.duplicateSubtype {
			n.problem = "death event: supply one subtype filter"
		} else if p.subtype == "" {
			n.problem = "death event: supply a creature subtype"
		} else if !p.creatureOnly && (!p.filterType || p.permanentType != TypeCreature) {
			n.problem = "death event: subtype filter requires a creature pattern"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "death event: subtype is not a creature subtype"
		}
	}
	if p.duplicateOutcome {
		n.problem = "death event: supply one sacrifice outcome filter"
	}
	if p.duplicateControl || (p.controllerOnly && p.opponentOnly) {
		n.problem = "death event: supply one controller filter"
	}
	if p.duplicatePT || (p.filterPT && !p.creatureOnly) {
		n.problem = "death event: supply one creature power or toughness filter"
	}
	if p.duplicateDamage || (p.damagedByThisTurn && !p.creatureOnly) {
		n.problem = "death event: supply one creature damage-history filter"
	}
	if p.controller != nil {
		n.bindingWrite = p.controller
		n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
	} else if p.captureController {
		n.problem = "death event: supply a controller binding"
	}
	if p.owner != nil {
		n.bindingWrites = append(n.bindingWrites, p.owner)
		n.children = append(n.children, definitionEdge{"owner", &eventPlayerCapture{binding: p.owner}})
		if p.owner == p.controller {
			n.problem = "death event: owner and controller need different bindings"
		}
	} else if p.captureOwner {
		n.problem = "death event: supply an owner binding"
	}
	if p.power != nil {
		n.bindingWrites = append(n.bindingWrites, p.power)
		n.children = append(n.children, definitionEdge{"power", &eventAmountCapture{binding: p.power}})
	} else if p.capturePower {
		n.problem = "death event: supply a power binding"
	}
	if p.duplicatePower || (p.capturePower && !p.creatureOnly) {
		n.problem = "death event: supply one power binding on a creature pattern"
	}
	if p.toughness != nil {
		n.bindingWrites = append(n.bindingWrites, p.toughness)
		n.children = append(n.children, definitionEdge{"toughness", &eventAmountCapture{binding: p.toughness}})
	} else if p.captureToughness {
		n.problem = "death event: supply a toughness binding"
	}
	if p.card != nil {
		n.bindingWrites = append(n.bindingWrites, p.card)
		n.children = append(n.children, definitionEdge{"card", &eventCardCapture{binding: p.card}})
	} else if p.captureCard {
		n.problem = "death event: supply a card binding"
	}
	return n
}

func (p creatureDiesPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.capturePower && !p.captureToughness && !p.captureOwner {
		return nil
	}
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	view := game.LookupObject(event.SourceID)
	if view == nil {
		return nil
	}
	var captures []Effect
	if p.captureOwner && p.owner != nil {
		captures = append(captures, &eventPlayerCapture{binding: p.owner, player: view.ViewOwner()})
	}
	if p.capturePower && p.power != nil {
		captures = append(captures, &eventAmountCapture{binding: p.power, amount: view.ViewPower()})
	}
	if p.captureToughness && p.toughness != nil {
		captures = append(captures, &eventAmountCapture{binding: p.toughness, amount: view.ViewToughness()})
	}
	return captures
}

func (p creatureDiesPattern) captureDestinationEvent(event *GameEvent, current *Game) []Effect {
	if !p.captureCard || p.card == nil {
		return nil
	}
	return []Effect{&eventCardCapture{binding: p.card, value: capturedCard{id: event.SourceID, visit: current.graveyardVisit(event.SourceID), zone: ZoneGraveyard}}}
}

func (p creatureDiesPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureController {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.controller, player: event.PlayerID}, Sequence(effects...))}
}

type creatureYouSacrificePattern struct{}

// CreatureYouSacrifice matches a creature sacrificed by the source controller.
func CreatureYouSacrifice() EventPattern { return creatureYouSacrificePattern{} }

func (creatureYouSacrificePattern) eventPattern() {}

func (creatureYouSacrificePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtSacrifice, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventPlayerIsController{},
		EventSacrificedPermanentIsCreature{},
	}})
}

func (creatureYouSacrificePattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "you sacrifice a creature", battlefieldSource: true}
}

// TurnStepPattern describes a turn-step boundary.
type TurnStepPattern interface {
	turnStepPattern()
	makeStepTrigger(ResolvingEffect) *GenericTriggered
	delayedStep(uuid.UUID) *DelayedTrigger
}
type stepPattern struct {
	event          EventType
	controllerOnly bool
}

func (stepPattern) turnStepPattern() {}
func (p stepPattern) delayedStep(controller uuid.UUID) *DelayedTrigger {
	t := &DelayedTrigger{EventType: p.event}
	if p.controllerOnly {
		t.MatchPlayerID = controller
	}
	return t
}

// EachEndStep matches the beginning of each end step.
var EachEndStep = stepPattern{EvtEndStep, false}

// YourEndStep matches the beginning of the controller's end step.
var YourEndStep = stepPattern{EvtEndStep, true}

// EachEndOfCombat matches the beginning of each end-of-combat step.
var EachEndOfCombat = stepPattern{EvtEndOfCombat, false}

// YourUpkeep matches the beginning of the controller's upkeep.
var YourUpkeep = stepPattern{EvtUpkeep, true}

// EachUpkeep matches the beginning of each upkeep.
var EachUpkeep = stepPattern{EvtUpkeep, false}

// EachDrawStep matches each draw step without a source-state restriction.
var EachDrawStep = stepPattern{EvtDrawStep, false}

func (p stepPattern) makeStepTrigger(e ResolvingEffect) *GenericTriggered {
	t := NewTriggered(p.event, false, e)
	if p.controllerOnly {
		t.SetConditionData(EventPlayerIsController{})
	}
	return t
}
func (p stepPattern) inspectDefinition() definitionNode {
	label := p.event.String()
	if p.controllerOnly {
		label = "your " + label
	}
	return definitionNode{Role: "turn-step-pattern", Text: label}
}

type mainPhasePattern struct{ first bool }

// YourFirstMainPhase matches the beginning of your first main phase.
var YourFirstMainPhase = mainPhasePattern{first: true}

// YourPostcombatMainPhase matches the beginning of each of your postcombat main phases.
var YourPostcombatMainPhase = mainPhasePattern{}

func (mainPhasePattern) turnStepPattern() {}
func (p mainPhasePattern) makeStepTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtMainPhase, false, effect).SetConditionData(EventPlayerIsController{})
	if p.first {
		t.AndConditionData(EventFlagIsTrue{})
	} else {
		t.AndConditionData(EventFlagIsFalse{})
	}
	return t
}
func (p mainPhasePattern) delayedStep(controller uuid.UUID) *DelayedTrigger {
	return &DelayedTrigger{EventType: EvtMainPhase, MatchPlayerID: controller, MatchFlag: p.first, matchFlagFalse: !p.first}
}
func (p mainPhasePattern) inspectDefinition() definitionNode {
	text := "your postcombat main phase"
	if p.first {
		text = "your first main phase"
	}
	return definitionNode{Role: "turn-step-pattern", Text: text}
}

// TriggerOption is a closed option for triggered abilities.
type TriggerOption interface {
	triggerOption()
	applyTrigger(*GenericTriggered)
}

func (targetOptions) triggerOption() {}
func (o targetOptions) applyTrigger(t *GenericTriggered) {
	for _, b := range o.bindings {
		if nilDefinition(b) {
			panic("targets: supply a non-nil binding")
		}
		t.targets = append(t.targets, b.targetSpec())
	}
}

type interveningIfOption struct{ condition StateCondition }

// WithInterveningIf checks the condition at detection and again at resolution.
func WithInterveningIf(condition StateCondition) interveningIfOption {
	return interveningIfOption{condition}
}
func (interveningIfOption) triggerOption() {}
func (o interveningIfOption) applyTrigger(t *GenericTriggered) {
	if nilDefinition(o.condition) {
		panic("intervening-if: supply a condition")
	}
	if t.interveningDefinition != nil {
		panic("intervening-if: supply only one condition")
	}
	t.interveningDefinition = o.condition
	before := t.Condition
	t.Condition = func(evt *GameEvent, g GameReader, source, controller uuid.UUID) bool {
		return (before == nil || before(evt, g, source, controller)) && o.condition.checkState(g, StateContext{SourceID: source, ControllerID: controller})
	}
	t.effects = []Effect{IfState(o.condition, Sequence(t.effects...))}
}
func applyTriggerOptions(t *GenericTriggered, options []TriggerOption) {
	for _, o := range options {
		if nilDefinition(o) {
			panic("trigger: supply a non-nil option")
		}
		o.applyTrigger(t)
	}
}

// WhenEvent uses the existing event, last-known-information, and pending-trigger paths.
func WhenEvent(pattern EventPattern, effect ResolvingEffect, options ...TriggerOption) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("event: supply a pattern")
	}
	t := pattern.makeTrigger(effect)
	t.eventDefinition = pattern
	applyTriggerOptions(t, options)
	return t
}

// At creates a trigger for a turn-step boundary.
func At(pattern TurnStepPattern, effect ResolvingEffect, options ...TriggerOption) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("step: supply a pattern")
	}
	t := pattern.makeStepTrigger(effect)
	t.stepDefinition = pattern
	applyTriggerOptions(t, options)
	return t
}

// WithTargets adds typed targets to a legacy event trigger.
func (t *GenericTriggered) WithTargets(bindings ...TargetBinding) *GenericTriggered {
	WithTargets(bindings...).applyTrigger(t)
	return t
}
