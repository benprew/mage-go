package mage

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_ability_activation_event.go ---
type abilityActivatedPattern struct {
	ability            *AbilityBinding
	captureAbility     bool
	duplicateAbility   bool
	attachedOnly       bool
	duplicateAttached  bool
	withTapCost        bool
	nonMana            bool
	duplicateNonMana   bool
	activator          *PlayerBinding
	captureActivator   bool
	duplicateActivator bool
	source             *DamageSourceBinding
	captureSource      bool
	duplicateSource    bool
	artifactOnly       bool
	opponentOnly       bool
	duplicateArtifact  bool
	duplicateOpponent  bool
	withoutTapCost     bool
	duplicateTapFilter bool
}

// ActivatedAbilityEvent matches a completed activated ability announcement.
func ActivatedAbilityEvent() abilityActivatedPattern { return abilityActivatedPattern{} }

// FromArtifact limits the event to an ability of an artifact at activation.
func (p abilityActivatedPattern) FromArtifact() abilityActivatedPattern {
	if p.artifactOnly {
		p.duplicateArtifact = true
	}
	p.artifactOnly = true
	return p
}

// ByOpponent limits the event to an ability activated by an opponent.
func (p abilityActivatedPattern) ByOpponent() abilityActivatedPattern {
	if p.opponentOnly {
		p.duplicateOpponent = true
	}
	p.opponentOnly = true
	return p
}

// WithoutTapCost limits the event to an activation with no tap symbol in its cost.
func (p abilityActivatedPattern) WithoutTapCost() abilityActivatedPattern {
	if p.withoutTapCost {
		p.duplicateTapFilter = true
	}
	p.withoutTapCost = true
	return p
}

// WithActivator saves the player who activated the ability.
func (p abilityActivatedPattern) WithActivator(binding *PlayerBinding) abilityActivatedPattern {
	if p.captureActivator {
		p.duplicateActivator = true
	}
	p.activator, p.captureActivator = binding, true
	return p
}

// WithSource saves the source as it exists when the ability is activated.
func (p abilityActivatedPattern) WithSource(binding *DamageSourceBinding) abilityActivatedPattern {
	if p.captureSource {
		p.duplicateSource = true
	}
	p.source, p.captureSource = binding, true
	return p
}

func (abilityActivatedPattern) eventPattern() {}

func (p abilityActivatedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtAbilityActivated, false, effect)
	if p.attachedOnly {
		t.AndConditionData(activationFromAttachedPermanent{})
	}
	if p.withTapCost {
		t.AndConditionData(activationHasTapCost{})
	}
	if p.nonMana {
		t.AndConditionData(activationUsesStack{})
	}
	if p.artifactOnly {
		t.AndConditionData(activatedSourceIsArtifact{})
	}
	if p.opponentOnly {
		t.AndConditionData(EventPlayerIsOpponent{})
	}
	if p.withoutTapCost {
		t.AndConditionData(activationWithoutTapCost{})
	}
	return t
}

type activatedSourceIsArtifact struct{}

func (activatedSourceIsArtifact) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok || event.SourceIncarnation == uuid.Nil {
		return false
	}
	if permanent := game.FindPermanentIncludingPhased(event.SourceID); permanent != nil && permanent.incarnationID == event.SourceIncarnation {
		return permanent.HasType(TypeArtifact)
	}
	lki := game.zones.LKIByIncarnation(event.SourceIncarnation)
	return lki != nil && lki.ID == event.SourceID && lki.HasType(TypeArtifact)
}

func (p abilityActivatedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "an ability is activated", battlefieldSource: true}
	if p.duplicateArtifact || p.duplicateOpponent || p.duplicateTapFilter || p.duplicateAttached || p.duplicateNonMana || (p.withTapCost && p.withoutTapCost) {
		n.problem = "ability activation event: supply each filter once"
	}
	if p.captureAbility {
		if p.ability == nil || p.duplicateAbility {
			n.problem = "ability activation event: supply one ability binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.ability)
			n.children = append(n.children, definitionEdge{"ability", &eventAbilityCapture{binding: p.ability}})
		}
	}
	if p.captureActivator {
		if p.activator == nil || p.duplicateActivator {
			n.problem = "ability activation event: supply one activator binding"
		} else {
			n.bindingWrite = p.activator
			n.children = append(n.children, definitionEdge{"activator", &eventPlayerCapture{binding: p.activator}})
		}
	}
	if p.captureSource {
		if p.source == nil || p.duplicateSource {
			n.problem = "ability activation event: supply one source binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.source)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.source}})
		}
	}
	return n
}

func (p abilityActivatedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.captureAbility && p.ability != nil {
		captures = append(captures, &eventAbilityCapture{binding: p.ability, id: event.ActivatedAbilityID})
	}
	if !p.captureSource || p.source == nil || event.SourceIncarnation == uuid.Nil {
		return captures
	}
	game, ok := observer.(*Game)
	if !ok {
		return captures
	}
	var source DamageSource
	if permanent := game.FindPermanentIncludingPhased(event.SourceID); permanent != nil && permanent.incarnationID == event.SourceIncarnation {
		source = damageSourceFromPermanent(permanent)
	} else if lki := game.zones.LKIByIncarnation(event.SourceIncarnation); lki != nil && lki.ID == event.SourceID {
		source = damageSourceFromLKI(lki)
	}
	return append(captures, &eventDamageSourceCapture{binding: p.source, source: source})
}

type eventDamageSourceCapture struct {
	binding *DamageSourceBinding
	source  DamageSource
}

func (*eventDamageSourceCapture) Text() string                 { return "remember the event source" }
func (*eventDamageSourceCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventDamageSourceCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.damageSourceBindings == nil {
		ctx.Game.resolution.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource)
	}
	ctx.Game.resolution.damageSourceBindings[e.binding] = cloneDamageSource(e.source)
	return nil
}
func (e *eventDamageSourceCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-damage-source-capture", bindingWrite: e.binding, children: []definitionEdge{{"source", e.binding}}}
}

func (p abilityActivatedPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureActivator {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.activator, player: event.PlayerID}, Sequence(effects...))}
}

// --- from composable_artifact_activation_event.go ---
type artifactActivationPattern struct {
	artifact            *PermanentBinding
	controller          *PlayerBinding
	captureArtifact     bool
	captureController   bool
	duplicateArtifact   bool
	duplicateController bool
	attachedOnly        bool
}

// ArtifactAbilityActivatedWithoutTapCost matches an artifact ability with no tap symbol in its cost.
func ArtifactAbilityActivatedWithoutTapCost() artifactActivationPattern {
	return artifactActivationPattern{}
}

// AttachedToSource limits the event to the artifact enchanted by the source.
func (p artifactActivationPattern) AttachedToSource() artifactActivationPattern {
	p.attachedOnly = true
	return p
}

// WithArtifact saves the activated artifact's battlefield incarnation.
func (p artifactActivationPattern) WithArtifact(binding *PermanentBinding) artifactActivationPattern {
	if p.captureArtifact {
		p.duplicateArtifact = true
	}
	p.artifact, p.captureArtifact = binding, true
	return p
}

// WithController saves the artifact controller when the ability is activated.
func (p artifactActivationPattern) WithController(binding *PlayerBinding) artifactActivationPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

func (artifactActivationPattern) eventPattern() {}

func (p artifactActivationPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtAbilityActivated, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		activationWithoutTapCost{}, activatedSourceIsArtifact{},
	}})
	if p.attachedOnly {
		t.AndConditionData(AttachedToIsEventSourceNoTapCost{})
	}
	return t
}

type activationWithoutTapCost struct{}

func (activationWithoutTapCost) CheckTriggerCond(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return !event.Flag
}

func (p artifactActivationPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "artifact ability activated without tap cost", battlefieldSource: true}
	if p.attachedOnly {
		n.Text = "attached artifact ability activated without tap cost"
	}
	if p.captureArtifact {
		if p.artifact == nil || p.duplicateArtifact {
			n.problem = "artifact activation event: supply one artifact binding"
		} else {
			n.children = append(n.children, definitionEdge{"artifact", &eventPermanentCapture{binding: p.artifact}})
			n.bindingWrites = append(n.bindingWrites, p.artifact)
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "artifact activation event: supply one controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
			n.bindingWrites = append(n.bindingWrites, p.controller)
		}
	}
	return n
}

func (p artifactActivationPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	game, ok := observer.(*Game)
	if !ok || event.SourceIncarnation == uuid.Nil {
		return nil
	}
	var controller uuid.UUID
	if artifact := game.FindPermanentIncludingPhased(event.SourceID); artifact != nil && artifact.incarnationID == event.SourceIncarnation {
		controller = artifact.ControllerID()
	} else if lki := game.zones.LKIByIncarnation(event.SourceIncarnation); lki != nil && lki.ID == event.SourceID {
		controller = lki.ViewController()
	} else {
		return nil
	}
	var captures []Effect
	if p.captureArtifact && p.artifact != nil {
		captures = append(captures, &eventPermanentCapture{binding: p.artifact, value: capturedPermanent{event.SourceID, event.SourceIncarnation}})
	}
	if p.captureController && p.controller != nil {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: controller})
	}
	return captures
}

// --- from composable_artifact_tap_event.go ---
type artifactTappedPattern struct {
	artifact          *PermanentBinding
	controller        *PlayerBinding
	captureArtifact   bool
	captureController bool
	duplicateArtifact bool
	duplicatePlayer   bool
	opponentOnly      bool
	duplicateOpponent bool
}

// ArtifactBecomesTapped matches an artifact that changes from untapped to tapped.
func ArtifactBecomesTapped() artifactTappedPattern { return artifactTappedPattern{} }

// ByOpponent limits the event to an artifact an opponent controls when it taps.
func (p artifactTappedPattern) ByOpponent() artifactTappedPattern {
	if p.opponentOnly {
		p.duplicateOpponent = true
	}
	p.opponentOnly = true
	return p
}

// WithArtifact saves the tapped artifact's battlefield incarnation.
func (p artifactTappedPattern) WithArtifact(binding *PermanentBinding) artifactTappedPattern {
	if p.captureArtifact {
		p.duplicateArtifact = true
	}
	p.artifact, p.captureArtifact = binding, true
	return p
}

// WithController saves the tapped artifact's controller at the tap event.
func (p artifactTappedPattern) WithController(binding *PlayerBinding) artifactTappedPattern {
	if p.captureController {
		p.duplicatePlayer = true
	}
	p.controller, p.captureController = binding, true
	return p
}

func (artifactTappedPattern) eventPattern() {}

func (p artifactTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtTapped, false, effect).SetConditionData(EventSourceHasType{Type: TypeArtifact})
	if p.opponentOnly {
		t.AndConditionData(EventSourceControlledByOpponent{})
	}
	return t
}

func (p artifactTappedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "artifact becomes tapped", battlefieldSource: true}
	if p.opponentOnly {
		n.Text = "opponent artifact becomes tapped"
	}
	if p.duplicateOpponent {
		n.problem = "artifact tap event: supply each filter once"
	}
	if p.captureArtifact {
		if p.artifact == nil || p.duplicateArtifact {
			n.problem = "artifact tap event: supply one artifact binding"
		} else {
			n.children = append(n.children, definitionEdge{"artifact", &eventPermanentCapture{binding: p.artifact}})
			n.bindingWrites = append(n.bindingWrites, p.artifact)
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicatePlayer {
			n.problem = "artifact tap event: supply one controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
			n.bindingWrites = append(n.bindingWrites, p.controller)
		}
	}
	return n
}

func (p artifactTappedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.captureArtifact && p.artifact != nil {
		if artifact := observer.FindPermanent(event.SourceID); artifact != nil && !artifact.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.artifact, value: capturedPermanent{artifact.ID(), artifact.incarnationID}})
		}
	}
	if p.captureController && p.controller != nil {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: event.PlayerID})
	}
	return captures
}

// --- from composable_attached_step.go ---
type attachedControllerUpkeepPattern struct {
	host   *PermanentBinding
	player *PlayerBinding
}

// AttachedControllerUpkeep saves the attachment and its controller when upkeep begins.
// The body reads these subjects with From and FromPlayer without targets.
func AttachedControllerUpkeep(host *PermanentBinding, player *PlayerBinding) EventPattern {
	return attachedControllerUpkeepPattern{host, player}
}
func (attachedControllerUpkeepPattern) eventPattern() {}
func (attachedControllerUpkeepPattern) makeTrigger(body ResolvingEffect) *GenericTriggered {
	return BeginningOfAttachedControllerUpkeepTrigger(body, false)
}
func (p attachedControllerUpkeepPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "attached permanent controller upkeep", battlefieldSource: true,
		children: []definitionEdge{{"host", &eventPermanentCapture{binding: p.host}}, {"player", &eventPlayerCapture{binding: p.player}}}}
}
func (p attachedControllerUpkeepPattern) captureDetectedEvent(event *GameEvent, g GameReader, source uuid.UUID) []Effect {
	var host capturedPermanent
	if aura := g.FindPermanent(source); aura != nil {
		if perm := g.FindPermanent(aura.AttachedTo); perm != nil {
			host = capturedPermanent{perm.ID(), perm.incarnationID}
		}
	}
	return []Effect{&eventPermanentCapture{p.host, host}, &eventPlayerCapture{p.player, event.PlayerID}}
}

type eventPermanentCapture struct {
	binding *PermanentBinding
	value   capturedPermanent
}

func (*eventPermanentCapture) Text() string                 { return "remember the event permanent" }
func (*eventPermanentCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventPermanentCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	ctx.Game.resolution.permanentBindings[e.binding] = e.value
	return nil
}
func (e *eventPermanentCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-permanent-capture", bindingWrite: e.binding, children: []definitionEdge{{"permanent", e.binding}}}
}

func captureDetectedTriggerEvent(ability TriggeredAbility, event *GameEvent, observer GameReader, current *Game, source uuid.UUID) []Effect {
	if trigger, ok := ability.(*GenericTriggered); ok {
		var captures []Effect
		if pattern, ok := trigger.eventDefinition.(interface {
			captureDetectedEvent(*GameEvent, GameReader, uuid.UUID) []Effect
		}); ok {
			captures = pattern.captureDetectedEvent(event, observer, source)
		}
		if pattern, ok := trigger.eventDefinition.(interface {
			captureDestinationEvent(*GameEvent, *Game) []Effect
		}); ok {
			captures = append(captures, pattern.captureDestinationEvent(event, current)...)
		}
		return captures
	}
	return nil
}

// --- from composable_attached_tap.go ---
type attachedBecomesTappedPattern struct {
	host *PermanentBinding
}

// AttachedBecomesTapped saves the attached permanent when it becomes tapped.
// Later attachment changes do not change the saved permanent.
func AttachedBecomesTapped(host *PermanentBinding) EventPattern {
	return attachedBecomesTappedPattern{host}
}
func (attachedBecomesTappedPattern) eventPattern() {}
func (attachedBecomesTappedPattern) makeTrigger(body ResolvingEffect) *GenericTriggered {
	return WhenAttachedBecomesTappedTrigger(body, false)
}
func (p attachedBecomesTappedPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "attached permanent becomes tapped", battlefieldSource: true,
		children: []definitionEdge{{"host", &eventPermanentCapture{binding: p.host}}}}
}
func (p attachedBecomesTappedPattern) captureDetectedEvent(event *GameEvent, g GameReader, _ uuid.UUID) []Effect {
	var host capturedPermanent
	if perm := g.FindPermanent(event.SourceID); perm != nil {
		host = capturedPermanent{perm.ID(), perm.incarnationID}
	}
	return []Effect{&eventPermanentCapture{p.host, host}}
}

// --- from composable_controller_damage_event.go ---
type controllerDamageEventPattern struct {
	amount          *NumberBinding
	source          *DamageSourceBinding
	captureSource   bool
	duplicateSource bool
}

// ControllerIsDealtDamage saves damage dealt to the source controller.
func ControllerIsDealtDamage(amount *NumberBinding) controllerDamageEventPattern {
	return controllerDamageEventPattern{amount: amount}
}

// WithSource saves the damage source when damage occurs.
func (p controllerDamageEventPattern) WithSource(source *DamageSourceBinding) controllerDamageEventPattern {
	if p.captureSource {
		p.duplicateSource = true
	}
	p.source, p.captureSource = source, true
	return p
}

func (controllerDamageEventPattern) eventPattern() {}
func (controllerDamageEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventTargetIsController{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p controllerDamageEventPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "you are dealt damage", battlefieldSource: true,
		children: []definitionEdge{{"amount", &eventAmountCapture{binding: p.amount}}}}
	if p.captureSource {
		if p.source == nil || p.duplicateSource {
			n.problem = "controller damage event requires one source binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.source)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.source}})
		}
	}
	return n
}
func (p controllerDamageEventPattern) captureDetectedEvent(_ *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureSource || p.source == nil {
		return nil
	}
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	return []Effect{&eventDamageSourceCapture{binding: p.source, source: cloneDamageSource(game.damageEventSource)}}
}
func (p controllerDamageEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventAmountCapture{binding: p.amount, amount: event.Amount}, Sequence(effects...))}
}

// --- from composable_discard_event.go ---
type playerDiscardEventPattern struct {
	player       *PlayerBinding
	card         *CardBinding
	cardCaptures int
	opponentOnly bool
}

// PlayerDiscardsCard saves the player who discards one card.
func PlayerDiscardsCard(player *PlayerBinding) playerDiscardEventPattern {
	return playerDiscardEventPattern{player: player}
}

// OpponentDiscardsCard saves the player who discards one card.
func OpponentDiscardsCard(player *PlayerBinding) playerDiscardEventPattern {
	return playerDiscardEventPattern{player: player, opponentOnly: true}
}

// WithCard saves the discarded card only while its graveyard visit lasts.
func (p playerDiscardEventPattern) WithCard(card *CardBinding) playerDiscardEventPattern {
	p.card = card
	p.cardCaptures++
	return p
}

func (playerDiscardEventPattern) eventPattern() {}
func (p playerDiscardEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	trigger := NewTriggered(EvtDiscard, false, effect)
	if p.opponentOnly {
		trigger.SetConditionData(EventPlayerIsOpponent{})
	}
	return trigger
}
func (p playerDiscardEventPattern) inspectDefinition() definitionNode {
	label := "a player discards a card"
	if p.opponentOnly {
		label = "an opponent discards a card"
	}
	n := definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.cardCaptures > 1 {
		n.problem = "save the discarded card only once"
	} else if p.cardCaptures == 1 && p.card == nil {
		n.problem = "supply a non-nil discarded card binding"
	} else if p.card != nil {
		n.bindingWrites = []savedBinding{p.card}
		n.children = append(n.children, definitionEdge{"card", p.card})
	}
	return n
}
func (p playerDiscardEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	captures := []ResolvingEffect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.cardCaptures == 1 {
		captures = append(captures, &eventCardCapture{binding: p.card, value: capturedCard{id: event.SourceID, visit: event.DiscardedCardVisit, zone: ZoneGraveyard}})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

// --- from composable_draw_event.go ---
type playerDrawEventPattern struct {
	player       *PlayerBinding
	card         *CardBinding
	cardCapture  bool
	cardCaptures int
	anyPlayer    bool
	opponentOnly bool
}

// PlayerDrawsCard saves the player who completes a card draw.
func PlayerDrawsCard(player *PlayerBinding) playerDrawEventPattern {
	return playerDrawEventPattern{player: player, anyPlayer: true}
}

// ControllerDrawsCard matches a card drawn by the source controller.
// It saves the player at detection without declaring a target.
func ControllerDrawsCard(player *PlayerBinding) EventPattern {
	return playerDrawEventPattern{player: player}
}

// ControllerDrawsCardInto saves the drawn card for its current hand visit.
func ControllerDrawsCardInto(player *PlayerBinding, card *CardBinding) EventPattern {
	return playerDrawEventPattern{player: player, card: card, cardCapture: true}
}

// OpponentDrawsCard saves the opponent who drew a card when the draw occurs.
func OpponentDrawsCard(player *PlayerBinding) playerDrawEventPattern {
	return playerDrawEventPattern{player: player, opponentOnly: true}
}

// WithCard saves the drawn card for its current hand visit.
func (p playerDrawEventPattern) WithCard(card *CardBinding) playerDrawEventPattern {
	p.card = card
	p.cardCapture = true
	p.cardCaptures++
	return p
}

func (playerDrawEventPattern) eventPattern() {}
func (p playerDrawEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.anyPlayer {
		return NewTriggered(EvtCardDrawn, false, effect)
	}
	condition := TriggerConditionData(EventPlayerIsController{})
	if p.opponentOnly {
		condition = EventPlayerIsOpponent{}
	}
	return NewTriggered(EvtCardDrawn, false, effect).SetConditionData(condition)
}
func (p playerDrawEventPattern) inspectDefinition() definitionNode {
	text := "you draw a card"
	if p.anyPlayer {
		text = "a player draws a card"
	} else if p.opponentOnly {
		text = "an opponent draws a card"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.cardCaptures > 1 {
		n.problem = "save the drawn card only once"
	} else if p.card != nil {
		n.bindingWrites = []savedBinding{p.card}
		n.children = append(n.children, definitionEdge{"card", p.card})
	} else if p.cardCapture {
		n.problem = "supply a non-nil drawn card binding"
	}
	return n
}
func (p playerDrawEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	captures := []ResolvingEffect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.cardCapture {
		captures = append(captures, &eventCardCapture{binding: p.card, value: capturedCard{id: event.DrawnCardID, visit: event.DrawnCardVisit, zone: ZoneHand}})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

// --- from composable_enchantment_entry.go ---
type controlledEnchantmentEntryPattern struct {
	subtype          string
	filterSubtype    bool
	duplicateSubtype bool
	permanent        *PermanentBinding
	capturePermanent bool
	duplicateCapture bool
}

// EnchantmentYouControlEnters matches an enchantment that enters under your control.
func EnchantmentYouControlEnters() controlledEnchantmentEntryPattern {
	return controlledEnchantmentEntryPattern{}
}

// OfSubtype checks the entering enchantment's effective subtype.
func (p controlledEnchantmentEntryPattern) OfSubtype(subtype string) controlledEnchantmentEntryPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.subtype, p.filterSubtype = subtype, true
	return p
}

// WithPermanent saves the entering enchantment's battlefield incarnation.
func (p controlledEnchantmentEntryPattern) WithPermanent(permanent *PermanentBinding) controlledEnchantmentEntryPattern {
	if p.capturePermanent {
		p.duplicateCapture = true
	}
	p.permanent, p.capturePermanent = permanent, true
	return p
}

func (controlledEnchantmentEntryPattern) eventPattern() {}

func (p controlledEnchantmentEntryPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtZoneChange, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
		EventSourceHasType{Type: TypeEnchantment},
		EventSourceControlledByController{},
	}})
	if p.filterSubtype {
		t.AndConditionData(eventSourceHasEntrySubtype{subtype: p.subtype})
	}
	return t
}

func (p controlledEnchantmentEntryPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "an enchantment you control enters", battlefieldSource: true}
	if p.filterSubtype {
		if family, known := knownSubtypeFamily(p.subtype); !known || family != SubtypeEnchantment || p.duplicateSubtype {
			n.problem = "enchantment entry event: supply one enchantment subtype"
		} else {
			n.Text = "an enchantment you control enters with subtype " + p.subtype
		}
	}
	if p.capturePermanent {
		if p.permanent == nil || p.duplicateCapture {
			n.problem = "enchantment entry event: supply one permanent binding"
		} else {
			n.bindingWrite = p.permanent
			n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
		}
	}
	return n
}

func (p controlledEnchantmentEntryPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.capturePermanent || p.permanent == nil {
		return nil
	}
	permanent := observer.FindPermanent(event.SourceID)
	if permanent == nil {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.permanent, value: capturedPermanent{permanent.ID(), permanent.incarnationID}}}
}

// --- from composable_event_player.go ---
type damageEventPattern struct {
	sourcePermanent           *PermanentBinding
	captureSourcePermanent    bool
	duplicateSourcePermanent  bool
	source                    PermanentRef
	player                    *PlayerBinding
	opponentOnly              bool
	controllerOnly            bool
	sourceController          *PlayerBinding
	captureSourceController   bool
	duplicateSourceController bool
	damageSource              *DamageSourceBinding
	captureDamageSource       bool
	duplicateDamageSource     bool
	amount                    *NumberBinding
	captureAmount             bool
	anyRecipient              bool
	combatOnly                bool
	permanent                 *PermanentBinding
	capturePermanent          bool
	duplicatePermanent        bool
	recipientPlayer           *PlayerBinding
	captureRecipientPlayer    bool
	duplicateRecipientPlayer  bool
}

// DealsDamage matches damage from the source to any recipient.
// It saves the actual damage amount without declaring a target.
func DealsDamage(source PermanentRef, amount *NumberBinding) damageEventPattern {
	return damageEventPattern{source: source, amount: amount, captureAmount: true, anyRecipient: true}
}

// WithPermanent saves the permanent that receives damage.
func (p damageEventPattern) WithPermanent(binding *PermanentBinding) damageEventPattern {
	if p.capturePermanent {
		p.duplicatePermanent = true
	}
	p.permanent, p.capturePermanent = binding, true
	return p
}

// WithRecipientPlayer saves the player damaged by an any-recipient event.
func (p damageEventPattern) WithRecipientPlayer(binding *PlayerBinding) damageEventPattern {
	if p.captureRecipientPlayer {
		p.duplicateRecipientPlayer = true
	}
	p.recipientPlayer, p.captureRecipientPlayer = binding, true
	return p
}

// DealsDamageToPlayer matches damage from the source to either player.
// It saves the damaged player without making that player a target.
func DealsDamageToPlayer(source PermanentRef, player *PlayerBinding) damageEventPattern {
	return damageEventPattern{source: source, player: player}
}

// ToController matches damage to the trigger controller when damage occurs.
func (p damageEventPattern) ToController() damageEventPattern {
	p.controllerOnly = true
	return p
}

// WithSourceController saves the damage source controller when damage occurs.
func (p damageEventPattern) WithSourceController(binding *PlayerBinding) damageEventPattern {
	if p.captureSourceController {
		p.duplicateSourceController = true
	}
	p.sourceController, p.captureSourceController = binding, true
	return p
}

// WithSource saves the damage source when damage occurs.
func (p damageEventPattern) WithSource(binding *DamageSourceBinding) damageEventPattern {
	if p.captureDamageSource {
		p.duplicateDamageSource = true
	}
	p.damageSource, p.captureDamageSource = binding, true
	return p
}

// WithSourcePermanent saves the exact permanent that deals damage.
// Later attachment changes cannot change this reference.
func (p damageEventPattern) WithSourcePermanent(binding *PermanentBinding) damageEventPattern {
	if p.captureSourcePermanent {
		p.duplicateSourcePermanent = true
	}
	p.sourcePermanent, p.captureSourcePermanent = binding, true
	return p
}

// DealsCombatDamageToPlayer matches combat damage from the source to a player.
func DealsCombatDamageToPlayer(_ sourceRef, player *PlayerBinding) damageEventPattern {
	return damageEventPattern{player: player, combatOnly: true}
}

// DealsDamageToOpponent matches damage to an opponent of the source controller.
// It checks the controller when damage occurs and saves the damaged player.
func DealsDamageToOpponent(_ sourceRef, player *PlayerBinding) damageEventPattern {
	return damageEventPattern{player: player, opponentOnly: true}
}

// WithAmount saves the damage event amount for the trigger body.
// It preserves the player capture and does not read later source power.
func (p damageEventPattern) WithAmount(amount *NumberBinding) EventPattern {
	p.amount, p.captureAmount = amount, true
	return p
}

func (damageEventPattern) eventPattern() {}
func (p damageEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.combatOnly {
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventSourceIsSelf{}, EventTargetIsPlayer{}, EventIsCombatDamage{},
		}})
	}
	if p.anyRecipient {
		if _, ok := p.source.(attachedPermanentRef); ok {
			return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsAttachedTo{})
		}
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsSelf{})
	}
	if p.opponentOnly {
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsSelfDamageToOpponent{})
	}
	sourceCondition := TriggerConditionData(EventSourceIsSelf{})
	if _, ok := p.source.(attachedPermanentRef); ok {
		sourceCondition = EventSourceIsAttachedTo{}
	}
	conditions := []TriggerConditionData{sourceCondition, EventTargetIsPlayer{}}
	if p.controllerOnly {
		conditions = append(conditions, EventTargetIsController{})
	}
	return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: conditions})
}
func (p damageEventPattern) inspectDefinition() definitionNode {
	text := "source deals damage to a player"
	if p.opponentOnly {
		text = "source deals damage to an opponent"
	}
	if p.combatOnly {
		text = "source deals combat damage to a player"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.anyRecipient {
		n.Text, n.bindingWrite, n.children = "source deals damage", nil, nil
		switch p.source.(type) {
		case sourceRef:
		case attachedPermanentRef:
			n.Text = "attached permanent deals damage"
		default:
			n.problem = "damage event source requires This or AttachedPermanent"
		}
	}
	if !p.anyRecipient && !p.combatOnly && !p.opponentOnly {
		switch p.source.(type) {
		case sourceRef:
			n.Text = "source deals damage to a player"
		case attachedPermanentRef:
			n.Text = "attached permanent deals damage to a player"
		default:
			n.problem = "damage event source requires This or AttachedPermanent"
		}
	}
	if p.controllerOnly {
		if p.anyRecipient || p.combatOnly || p.opponentOnly {
			n.problem = "controller damage filter requires DealsDamageToPlayer"
		} else {
			n.Text += " to trigger controller"
		}
	}
	if p.captureSourceController {
		if p.anyRecipient || p.combatOnly || p.opponentOnly || p.sourceController == nil || p.duplicateSourceController {
			n.problem = "damage event requires one source controller binding on DealsDamageToPlayer"
		} else if p.sourceController == p.player {
			n.problem = "damage event has duplicate player and source controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"source controller", &eventPlayerCapture{binding: p.sourceController}})
		}
	}
	if p.captureDamageSource {
		if p.combatOnly || p.opponentOnly || p.damageSource == nil || p.duplicateDamageSource {
			n.problem = "damage event requires one source binding on DealsDamage or DealsDamageToPlayer"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.damageSource)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.damageSource}})
		}
	}
	if p.captureAmount {
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
	}
	if p.capturePermanent {
		if !p.anyRecipient || p.permanent == nil || p.duplicatePermanent {
			n.problem = "damage event requires one permanent binding on DealsDamage"
		} else {
			n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
		}
	}
	if p.captureSourcePermanent {
		if p.combatOnly || p.opponentOnly || p.sourcePermanent == nil || p.duplicateSourcePermanent || p.capturePermanent && p.sourcePermanent == p.permanent {
			n.problem = "damage event requires one distinct source permanent binding on DealsDamage or DealsDamageToPlayer"
		} else {
			n.children = append(n.children, definitionEdge{"source permanent", &eventPermanentCapture{binding: p.sourcePermanent}})
		}
	}
	if p.captureRecipientPlayer {
		if !p.anyRecipient || p.recipientPlayer == nil || p.duplicateRecipientPlayer {
			n.problem = "damage event requires one player recipient binding on DealsDamage"
		} else {
			n.children = append(n.children, definitionEdge{"player recipient", &eventPlayerCapture{binding: p.recipientPlayer}})
		}
	}
	return n
}
func (p damageEventPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	var captures []Effect
	if p.captureSourcePermanent && p.sourcePermanent != nil {
		captures = append(captures, &eventPermanentCapture{binding: p.sourcePermanent, value: capturedPermanent{event.SourceID, event.SourceIncarnation}})
	}
	if p.captureDamageSource && p.damageSource != nil {
		captures = append(captures, &eventDamageSourceCapture{binding: p.damageSource, source: cloneDamageSource(game.damageEventSource)})
	}
	if p.capturePermanent && p.permanent != nil && event.TargetIncarnation != uuid.Nil {
		captures = append(captures, &eventPermanentCapture{binding: p.permanent, value: capturedPermanent{event.TargetID, event.TargetIncarnation}})
	}
	return captures
}
func (p damageEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	var captures []ResolvingEffect
	if !p.anyRecipient {
		captures = append(captures, &eventPlayerCapture{p.player, event.TargetID})
	}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	if p.captureSourceController {
		captures = append(captures, &eventPlayerCapture{binding: p.sourceController, player: event.SourceControllerID})
	}
	if p.captureRecipientPlayer {
		player := uuid.Nil
		if event.TargetIncarnation == uuid.Nil {
			player = event.TargetID
		}
		captures = append(captures, &eventPlayerCapture{binding: p.recipientPlayer, player: player})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

type eventPlayerCapture struct {
	binding *PlayerBinding
	player  uuid.UUID
}

func (*eventPlayerCapture) Text() string                 { return "remember the event player" }
func (*eventPlayerCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventPlayerCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.playerBindings == nil {
		ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
	}
	ctx.Game.resolution.playerBindings[e.binding] = e.player
	return nil
}
func (e *eventPlayerCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-player-capture", bindingWrite: e.binding, children: []definitionEdge{{"player", e.binding}}}
}

func captureTriggerEventEffects(pt *pendingTrigger, obj *StackObject) {
	if pt.event != nil {
		obj.Effects = captureDeathCause(obj.Effects, pt.event)
	}
	if pt.delayed != nil && pt.delayed.eventPlayerBinding != nil && pt.event != nil {
		for i, effect := range obj.Effects {
			captured, ok := effect.(*capturedDelayedBody)
			if !ok {
				continue
			}
			body := *captured
			body.players = maps.Clone(captured.players)
			if body.players == nil {
				body.players = make(map[*PlayerBinding]uuid.UUID)
			}
			body.players[pt.delayed.eventPlayerBinding] = pt.event.PlayerID
			obj.Effects[i] = &body
		}
	}
	if len(pt.eventCaptures) != 0 {
		if pt.delayed != nil && (pt.delayed.deathCapture != nil || pt.delayed.combatPair != nil || pt.delayed.damageReceived != nil) {
			for i, effect := range obj.Effects {
				captured, ok := effect.(*capturedDelayedBody)
				if !ok {
					continue
				}
				body := *captured
				body.permanents = maps.Clone(captured.permanents)
				if body.permanents == nil {
					body.permanents = make(map[*PermanentBinding]capturedPermanent)
				}
				body.players = maps.Clone(captured.players)
				if body.players == nil {
					body.players = make(map[*PlayerBinding]uuid.UUID)
				}
				body.numbers = maps.Clone(captured.numbers)
				if body.numbers == nil {
					body.numbers = make(map[*NumberBinding]int)
				}
				for _, capture := range pt.eventCaptures {
					switch fact := capture.(type) {
					case *eventAmountCapture:
						body.numbers[fact.binding] = fact.amount
					case *eventPermanentCapture:
						body.permanents[fact.binding] = fact.value
					case *eventPlayerCapture:
						body.players[fact.binding] = fact.player
					}
				}
				obj.Effects[i] = &body
			}
		} else {
			obj.Effects = []Effect{Sequence(Sequence(pt.eventCaptures...), Sequence(obj.Effects...))}
		}
	}
	if pt.event == nil {
		return
	}
	trigger, ok := pt.ability.(*GenericTriggered)
	if !ok {
		return
	}
	if pattern, ok := trigger.eventDefinition.(interface {
		captureEventEffects(*GameEvent, []Effect) []Effect
	}); ok {
		obj.Effects = pattern.captureEventEffects(pt.event, obj.Effects)
	}
}

// --- from composable_graveyard_leave_event.go ---
type graveyardLeaveBatchPattern struct {
	count          *NumberBinding
	captureCount   bool
	duplicateCount bool
}

// OneOrMoreCardsLeaveYourGraveyard matches one graveyard leave event group.
func OneOrMoreCardsLeaveYourGraveyard() graveyardLeaveBatchPattern {
	return graveyardLeaveBatchPattern{}
}

// WithCount saves the number of cards that left in the event group.
func (p graveyardLeaveBatchPattern) WithCount(count *NumberBinding) graveyardLeaveBatchPattern {
	if p.captureCount {
		p.duplicateCount = true
	}
	p.count = count
	p.captureCount = true
	return p
}

func (graveyardLeaveBatchPattern) eventPattern() {}

func (graveyardLeaveBatchPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtCardsLeftGraveyard, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventPlayerIsController{}, EventAmountGreaterThan{N: 0},
	}})
}

func (p graveyardLeaveBatchPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "one or more cards leave your graveyard", battlefieldSource: true}
	if p.captureCount {
		if p.count == nil || p.duplicateCount {
			n.problem = "graveyard leave event: supply one count binding"
		} else {
			n.bindingWrite = p.count
			n.children = append(n.children, definitionEdge{"count", &eventAmountCapture{binding: p.count}})
		}
	}
	return n
}

func (p graveyardLeaveBatchPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureCount {
		return effects
	}
	return []Effect{Sequence(&eventAmountCapture{binding: p.count, amount: event.Amount}, Sequence(effects...))}
}

// --- from composable_land_entry_event.go ---
type landEntersPattern struct {
	permanent           *PermanentBinding
	capturePermanent    bool
	duplicatePermanent  bool
	subtype             string
	filterSubtype       bool
	duplicateSubtype    bool
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
	actor               *PlayerBinding
	captureActor        bool
	duplicateActor      bool
}

// LandEnters matches each land that enters the battlefield.
func LandEnters() landEntersPattern { return landEntersPattern{} }

// WithPermanent saves the entering land's incarnation before trigger placement.
func (p landEntersPattern) WithPermanent(binding *PermanentBinding) landEntersPattern {
	if p.capturePermanent {
		p.duplicatePermanent = true
	}
	p.permanent, p.capturePermanent = binding, true
	return p
}

// OfSubtype limits the event to an entering land with this land subtype.
func (p landEntersPattern) OfSubtype(subtype string) landEntersPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.subtype, p.filterSubtype = subtype, true
	return p
}

// WithController saves the entering land's controller at the entry event.
func (p landEntersPattern) WithController(binding *PlayerBinding) landEntersPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

// WithActor saves the player who caused the land to enter.
func (p landEntersPattern) WithActor(binding *PlayerBinding) landEntersPattern {
	if p.captureActor {
		p.duplicateActor = true
	}
	p.actor, p.captureActor = binding, true
	return p
}

func (landEntersPattern) eventPattern() {}

func (p landEntersPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	trigger := NewTriggered(EvtZoneChange, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
		EventSourceHasType{Type: TypeLand},
	}})
	if p.filterSubtype && isLandSubtype(p.subtype) {
		trigger.AndConditionData(EventSourceHasSubType{SubType: p.subtype})
	}
	if p.captureActor {
		trigger.AndConditionData(entryActorKnown{})
	}
	return trigger
}

type entryActorKnown struct{}

func (entryActorKnown) CheckTriggerCond(event *GameEvent, observer GameReader, _, _ uuid.UUID) bool {
	return observer.GetPlayer(event.ActorID) != nil
}

func (p landEntersPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "a land enters", battlefieldSource: true}
	if p.capturePermanent {
		if p.permanent == nil || p.duplicatePermanent {
			n.problem = "land entry event: supply one permanent binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.permanent)
			n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
		}
	}
	if p.filterSubtype {
		n.Text = "a " + p.subtype + " enters"
		if p.duplicateSubtype || !isLandSubtype(p.subtype) {
			n.problem = "land entry event: supply one valid land subtype"
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "land entry event: supply one controller binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.controller)
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
		}
	}
	if p.captureActor {
		if p.actor == nil || p.duplicateActor {
			n.problem = "land entry event: supply one actor binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.actor)
			n.children = append(n.children, definitionEdge{"actor", &eventPlayerCapture{binding: p.actor}})
		}
	}
	if p.captureActor && p.captureController && p.actor == p.controller && p.actor != nil {
		n.problem = "land entry event: actor and controller need different bindings"
	}
	return n
}

func (p landEntersPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.capturePermanent && !p.captureController && !p.captureActor {
		return nil
	}
	var captures []Effect
	if p.capturePermanent && p.permanent != nil {
		if land := observer.FindPermanent(event.SourceID); land != nil {
			captures = append(captures, &eventPermanentCapture{binding: p.permanent, value: capturedPermanent{land.ID(), land.incarnationID}})
		}
	}
	if p.captureController {
		land := observer.FindPermanent(event.SourceID)
		if land != nil {
			captures = append(captures, &eventPlayerCapture{binding: p.controller, player: land.ControllerID()})
		}
	}
	if p.captureActor {
		captures = append(captures, &eventPlayerCapture{binding: p.actor, player: event.ActorID})
	}
	return captures
}

// --- from composable_land_mana_event.go ---
type landTappedForManaPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	land            *PermanentBinding
	types           *ManaTypesBinding
	capturePlayer   bool
	captureAmount   bool
	captureLand     bool
	captureTypes    bool
	duplicatePlayer bool
	duplicateAmount bool
	duplicateLand   bool
	duplicateTypes  bool
	manaType        Color
	filterManaType  bool
	duplicateType   bool
	attachedOnly    bool
	duplicateAttach bool
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

// OfAttachedLand matches the land attached to the trigger source at event time.
func (p landTappedForManaPattern) OfAttachedLand() landTappedForManaPattern {
	if p.attachedOnly {
		p.duplicateAttach = true
	}
	p.attachedOnly = true
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

// WithManaTypes saves the types produced by the triggering land ability.
func (p landTappedForManaPattern) WithManaTypes(binding *ManaTypesBinding) landTappedForManaPattern {
	if p.captureTypes {
		p.duplicateTypes = true
	}
	p.types, p.captureTypes = binding, true
	return p
}

func (landTappedForManaPattern) eventPattern() {}

func (p landTappedForManaPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtTappedForMana, false, effect).SetConditionData(tappedForManaLandSource{manaType: p.manaType, filterManaType: p.filterManaType, attachedOnly: p.attachedOnly})
}

type tappedForManaLandSource struct {
	manaType       Color
	filterManaType bool
	attachedOnly   bool
}

func (p tappedForManaLandSource) CheckTriggerCond(event *GameEvent, reader GameReader, sourceID, _ uuid.UUID) bool {
	if p.filterManaType && !slices.Contains(event.ManaTypes, p.manaType) {
		return false
	}
	game, ok := reader.(*Game)
	if !ok || event.SourceIncarnation == uuid.Nil {
		return false
	}
	if p.attachedOnly {
		source := game.FindPermanent(sourceID)
		if source == nil || source.AttachedTo != event.SourceID || source.attachedIncarnation != event.SourceIncarnation {
			return false
		}
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
	if p.attachedOnly {
		n.Text = "attached land is tapped for mana"
	}
	if p.filterManaType && (p.duplicateType || p.manaType < Colorless || p.manaType >= AnyColor) {
		n.problem = "land tapped for mana: supply one concrete mana type"
	}
	if p.duplicateAttach {
		n.problem = "land tapped for mana: supply one attached-land filter"
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
	if p.captureTypes {
		if p.types == nil || p.duplicateTypes {
			n.problem = "land tapped for mana: supply one mana types binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.types)
			n.children = append(n.children, definitionEdge{"mana types", &eventManaTypesCapture{binding: p.types}})
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
	if !p.capturePlayer && !p.captureAmount && !p.captureTypes {
		return effects
	}
	var captures []ResolvingEffect
	if p.capturePlayer {
		captures = append(captures, &eventPlayerCapture{binding: p.player, player: event.PlayerID})
	}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{binding: p.amount, amount: event.Amount})
	}
	if p.captureTypes {
		captures = append(captures, &eventManaTypesCapture{binding: p.types, types: slices.Clone(event.ManaTypes)})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

// --- from composable_land_tap_event.go ---
type opponentLandTappedPattern struct {
	subtype          string
	filterSubtype    bool
	duplicateSubtype bool
	land             *PermanentBinding
	captureLand      bool
	duplicateLand    bool
}

// OpponentLandBecomesTapped matches a land an opponent controls when it becomes tapped.
func OpponentLandBecomesTapped() opponentLandTappedPattern { return opponentLandTappedPattern{} }

// OfSubtype limits the event to a land with the given subtype.
func (p opponentLandTappedPattern) OfSubtype(subtype string) opponentLandTappedPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithLand saves the tapped land's battlefield incarnation for the trigger.
func (p opponentLandTappedPattern) WithLand(binding *PermanentBinding) opponentLandTappedPattern {
	if p.captureLand {
		p.duplicateLand = true
	}
	p.land, p.captureLand = binding, true
	return p
}

func (opponentLandTappedPattern) eventPattern() {}

func (p opponentLandTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtTapped, false, effect).SetConditionData(EventSourceHasType{Type: TypeLand})
	t.AndConditionData(EventSourceControlledByOpponent{})
	if p.filterSubtype && p.subtype != "" {
		t.AndConditionData(EventSourceHasSubType{SubType: p.subtype})
	}
	return t
}

func (p opponentLandTappedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "opponent land becomes tapped", battlefieldSource: true}
	if p.filterSubtype {
		n.Text = "opponent " + p.subtype + " becomes tapped"
		if p.duplicateSubtype || !isLandSubtype(p.subtype) {
			n.problem = "land tap event: supply a valid land subtype"
		}
	}
	if p.captureLand {
		if p.land == nil || p.duplicateLand {
			n.problem = "land tap event: supply one land binding"
		} else {
			n.children = append(n.children, definitionEdge{"land", &eventPermanentCapture{binding: p.land}})
			n.bindingWrites = append(n.bindingWrites, p.land)
		}
	}
	return n
}

func (p opponentLandTappedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureLand || p.land == nil {
		return nil
	}
	land := observer.FindPermanent(event.SourceID)
	if land == nil || land.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.land, value: capturedPermanent{land.ID(), land.incarnationID}}}
}

// --- from composable_life_event.go ---
type lifeLossEventPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	captureAmount   bool
	duplicateAmount bool
	opponentOnly    bool
}

// OpponentLosesLife matches an opponent's life loss when it occurs.
// It saves the player without declaring a target.
func OpponentLosesLife(player *PlayerBinding) lifeLossEventPattern {
	return lifeLossEventPattern{player: player, opponentOnly: true}
}

// PlayerLosesLife saves any player who loses a positive amount of life.
func PlayerLosesLife(player *PlayerBinding) lifeLossEventPattern {
	return lifeLossEventPattern{player: player}
}

// WithAmount saves the life loss amount for the trigger body.
func (p lifeLossEventPattern) WithAmount(amount *NumberBinding) lifeLossEventPattern {
	if p.captureAmount {
		p.duplicateAmount = true
	}
	p.amount, p.captureAmount = amount, true
	return p
}

func (lifeLossEventPattern) eventPattern() {}
func (p lifeLossEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.opponentOnly {
		return wheneverOpponentLosesLifeTrigger(effect, false).AndConditionData(EventAmountGreaterThan{N: 0})
	}
	return NewTriggered(EvtLifeLost, false, effect).SetConditionData(EventAmountGreaterThan{N: 0})
}
func (p lifeLossEventPattern) inspectDefinition() definitionNode {
	text := "a player loses life"
	if p.opponentOnly {
		text = "an opponent loses life"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.captureAmount {
		n.bindingWrites = append(n.bindingWrites, p.amount)
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
		if p.duplicateAmount {
			n.problem = "life loss event: supply one amount binding"
		}
	}
	return n
}
func (p lifeLossEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	captures := []Effect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

// --- from composable_life_gain_event.go ---
type lifeGainEventPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	captureAmount   bool
	duplicateAmount bool
	controllerOnly  bool
}

// ControllerGainsLife matches positive life gain by the source controller.
// It saves the player at detection without declaring a target.
func ControllerGainsLife(player *PlayerBinding) lifeGainEventPattern {
	return lifeGainEventPattern{player: player, controllerOnly: true}
}

// PlayerGainsLife saves any player who gains a positive amount of life.
func PlayerGainsLife(player *PlayerBinding) lifeGainEventPattern {
	return lifeGainEventPattern{player: player}
}

// WithAmount saves the amount of life gained when the event occurs.
func (p lifeGainEventPattern) WithAmount(amount *NumberBinding) lifeGainEventPattern {
	if p.captureAmount {
		p.duplicateAmount = true
	}
	p.amount, p.captureAmount = amount, true
	return p
}

func (lifeGainEventPattern) eventPattern() {}
func (p lifeGainEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	trigger := NewTriggered(EvtLifeGained, false, effect).SetConditionData(EventAmountGreaterThan{N: 0})
	if p.controllerOnly {
		trigger.AndConditionData(EventPlayerIsController{})
	}
	return trigger
}
func (p lifeGainEventPattern) inspectDefinition() definitionNode {
	text := "a player gains life"
	if p.controllerOnly {
		text = "you gain life"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.captureAmount {
		n.bindingWrites = append(n.bindingWrites, p.amount)
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
		if p.duplicateAmount {
			n.problem = "life gain event: supply one amount binding"
		}
	}
	return n
}
func (p lifeGainEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	captures := []Effect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	return []Effect{Sequence(append(captures, Sequence(effects...))...)}
}

// --- from composable_permanent_tap_event.go ---
type permanentTappedPattern struct {
	cardType            CardType
	filterType          bool
	duplicateType       bool
	opponentOnly        bool
	duplicateOpponent   bool
	permanent           *PermanentBinding
	capturePermanent    bool
	duplicatePermanent  bool
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
}

// PermanentBecomesTapped matches a permanent that changes to tapped.
func PermanentBecomesTapped() permanentTappedPattern { return permanentTappedPattern{} }

// OfType limits the event to a permanent with this card type.
func (p permanentTappedPattern) OfType(cardType CardType) permanentTappedPattern {
	if p.filterType {
		p.duplicateType = true
	}
	p.cardType, p.filterType = cardType, true
	return p
}

// ByOpponent limits the event to a permanent an opponent controls.
func (p permanentTappedPattern) ByOpponent() permanentTappedPattern {
	if p.opponentOnly {
		p.duplicateOpponent = true
	}
	p.opponentOnly = true
	return p
}

// WithPermanent saves the tapped permanent's battlefield incarnation.
func (p permanentTappedPattern) WithPermanent(binding *PermanentBinding) permanentTappedPattern {
	if p.capturePermanent {
		p.duplicatePermanent = true
	}
	p.permanent, p.capturePermanent = binding, true
	return p
}

// WithController saves the tapped permanent's controller at the event.
func (p permanentTappedPattern) WithController(binding *PlayerBinding) permanentTappedPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

func (permanentTappedPattern) eventPattern() {}

func (p permanentTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtTapped, false, effect)
	if p.filterType {
		t.SetConditionData(EventSourceHasType{Type: p.cardType})
	}
	if p.opponentOnly {
		t.AndConditionData(EventSourceControlledByOpponent{})
	}
	return t
}

func (p permanentTappedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "permanent becomes tapped", battlefieldSource: true}
	if p.duplicateType || p.duplicateOpponent || !validTappedPermanentType(p.cardType, p.filterType) {
		n.problem = "permanent tap event: supply each valid filter once"
	}
	if p.capturePermanent {
		if p.permanent == nil || p.duplicatePermanent {
			n.problem = "permanent tap event: supply one permanent binding"
		} else {
			n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
			n.bindingWrites = append(n.bindingWrites, p.permanent)
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "permanent tap event: supply one controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
			n.bindingWrites = append(n.bindingWrites, p.controller)
		}
	}
	return n
}

func validTappedPermanentType(cardType CardType, filtered bool) bool {
	if !filtered {
		return true
	}
	switch cardType {
	case TypeCreature, TypeLand, TypeArtifact, TypeEnchantment, TypePlaneswalker:
		return true
	default:
		return false
	}
}

func (p permanentTappedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.capturePermanent && p.permanent != nil {
		if permanent := observer.FindPermanent(event.SourceID); permanent != nil && !permanent.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.permanent, value: capturedPermanent{permanent.ID(), permanent.incarnationID}})
		}
	}
	if p.captureController && p.controller != nil {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: event.PlayerID})
	}
	return captures
}

// --- from composable_source_tap_event.go ---
type sourceTappedPattern struct{}

// BecomesTapped matches a battlefield source when its status changes to tapped.
func BecomesTapped(sourceRef) EventPattern { return sourceTappedPattern{} }

func (sourceTappedPattern) eventPattern() {}

func (sourceTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtTapped, false, effect).SetConditionData(EventSourceIsSelf{})
}

func (sourceTappedPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "source becomes tapped", battlefieldSource: true}
}

// --- from composable_source_targeted.go ---
type sourceTargetedBySpellPattern struct{}
type sourceTargetedPattern struct {
	opponentOnly        bool
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
}

// BecomesTargeted matches a spell or ability that targets the battlefield source.
func BecomesTargeted(sourceRef) sourceTargetedPattern { return sourceTargetedPattern{} }

// ByOpponent checks the targeting object's controller when it targets the source.
func (p sourceTargetedPattern) ByOpponent() sourceTargetedPattern {
	p.opponentOnly = true
	return p
}

// WithController saves the targeting object's controller for this trigger.
func (p sourceTargetedPattern) WithController(binding *PlayerBinding) sourceTargetedPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

// BecomesTargetOfSpell matches a spell that targets the battlefield source.
func BecomesTargetOfSpell(sourceRef) EventPattern { return sourceTargetedBySpellPattern{} }

func (sourceTargetedBySpellPattern) eventPattern() {}
func (sourceTargetedPattern) eventPattern()        {}

func (p sourceTargetedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtBecomesTarget, false, effect).SetConditionData(EventTargetIsSelf{})
	if p.opponentOnly {
		t.AndConditionData(EventPlayerIsNotController{})
	}
	return t
}

func (p sourceTargetedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source becomes the target of a spell or ability", battlefieldSource: true}
	if p.opponentOnly {
		n.Text += " an opponent controls"
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "targeting event: supply one controller binding"
		} else {
			n.bindingWrite = p.controller
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
		}
	}
	return n
}

func (p sourceTargetedPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureController {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.controller, player: event.PlayerID}, Sequence(effects...))}
}

func (sourceTargetedBySpellPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtBecomesTarget, false, effect).SetConditionData(AndTriggerCond{
		Conditions: []TriggerConditionData{EventTargetIsSelf{}, EventFlagIsFalse{}},
	})
}

func (sourceTargetedBySpellPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "source becomes the target of a spell", battlefieldSource: true}
}

// --- from composable_spell_cast_event.go ---
type spellCastPattern struct {
	sourceSpell                 bool
	contextualFilter            *SpellFilter
	filterCount                 int
	color                       Color
	filterColor                 bool
	duplicateColor              bool
	cardType                    CardType
	filterType                  bool
	duplicateType               bool
	instantOrSorcery            bool
	targetsCreature             bool
	duplicateTargetsCreature    bool
	enchantmentSubtype          string
	filterEnchantmentSubtype    bool
	duplicateEnchantmentSubtype bool
	opponentOnly                bool
	controllerOnly              bool
	duplicateCasterFilter       bool
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

// SourceSpellCast matches the source spell when it becomes cast on the stack.
func SourceSpellCast() spellCastPattern { return spellCastPattern{sourceSpell: true} }

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
	if p.opponentOnly || p.controllerOnly {
		p.duplicateCasterFilter = true
	}
	p.opponentOnly = true
	return p
}

// ByController matches a spell cast by the trigger source's controller.
func (p spellCastPattern) ByController() spellCastPattern {
	if p.opponentOnly || p.controllerOnly {
		p.duplicateCasterFilter = true
	}
	p.controllerOnly = true
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
	if p.filterType || p.instantOrSorcery {
		p.duplicateType = true
	}
	p.cardType = cardType
	p.filterType = true
	return p
}

// OfInstantOrSorcery matches a spell with either spell type.
func (p spellCastPattern) OfInstantOrSorcery() spellCastPattern {
	if p.filterType || p.instantOrSorcery {
		p.duplicateType = true
	}
	p.instantOrSorcery = true
	return p
}

// TargetsCreature checks the current creature status of a chosen spell target.
func (p spellCastPattern) TargetsCreature() spellCastPattern {
	if p.targetsCreature {
		p.duplicateTargetsCreature = true
	}
	p.targetsCreature = true
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
	if p.sourceSpell {
		t.sourceZone = ZoneStack
	}
	if p.opponentOnly {
		t.SetConditionData(EventPlayerIsNotController{})
	}
	if p.controllerOnly {
		t.SetConditionData(EventPlayerIsController{})
	}
	if p.sourceSpell {
		t.AndConditionData(EventSourceIsSelf{})
	}
	if p.filterColor {
		t.AndConditionData(spellCastHasEffectiveColor{color: p.color})
	}
	if p.filterType {
		t.AndConditionData(spellCastHasType{cardType: p.cardType})
	}
	if p.instantOrSorcery {
		t.AndConditionData(spellCastIsInstantOrSorcery{})
	}
	if p.targetsCreature {
		t.AndConditionData(spellCastTargetsCreature{})
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
	if p.filterCount > 0 {
		t.AndConditionData(spellCastMatchesFilter{filter: *p.contextualFilter})
	}
	return t
}

func (p spellCastPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "a spell is cast", battlefieldSource: !p.sourceSpell}
	if p.filterCount > 0 {
		n.children = append(n.children, definitionEdge{"filter", p.contextualFilter})
	}
	if p.filterCount > 1 {
		n.problem = "supply one contextual spell filter"
	}
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
	if p.controllerOnly {
		n.Text = "this source's controller casts a spell"
	}
	if p.duplicateCasterFilter {
		n.problem = "spell cast event: supply one caster filter"
	}
	if p.filterType {
		n.Text += " with type " + strings.ToLower(p.cardType.String())
		if p.duplicateType || p.cardType < TypeCreature || p.cardType > TypePlaneswalker || p.cardType == TypeLand {
			n.problem = "spell cast event: supply one valid spell type"
		}
	}
	if p.instantOrSorcery {
		n.Text += " with instant or sorcery type"
		if p.duplicateType {
			n.problem = "spell cast event: supply one spell type filter"
		}
	}
	if p.targetsCreature {
		n.Text += " that targets a creature"
		if p.duplicateTargetsCreature {
			n.problem = "spell cast event: supply the creature-target filter once"
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

type spellCastIsInstantOrSorcery struct{}
type spellCastTargetsCreature struct{}

func (spellCastIsInstantOrSorcery) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok || event.TargetID == uuid.Nil {
		return false
	}
	object := game.stack.FindByID(event.TargetID)
	return object != nil && !object.IsAbility && object.Card != nil && object.SourceID == event.SourceID &&
		(object.Card.HasType(TypeInstant) || object.Card.HasType(TypeSorcery))
}

func (spellCastTargetsCreature) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok || event.TargetID == uuid.Nil {
		return false
	}
	object := game.stack.FindByID(event.TargetID)
	if object == nil || object.IsAbility || object.Card == nil || object.SourceID != event.SourceID {
		return false
	}
	for _, id := range object.Targets {
		if object.TargetZones[id] != ZoneBattlefield {
			continue
		}
		permanent := game.FindPermanent(id)
		if permanent != nil && !permanent.PhasedOut && permanent.incarnationID == object.targetIncarnations[id] && permanent.HasType(TypeCreature) {
			return true
		}
	}
	return false
}

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
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

type eventSpellCapture struct {
	binding *SpellBinding
	id      uuid.UUID
	source  DamageSource
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

// --- from composable_step_player.go ---
// TurnStepEventPattern supports ordinary and delayed step-event capture.
type TurnStepEventPattern interface {
	EventPattern
	DelayedStepPattern
}

type turnStepEventPattern struct {
	step   TurnStepPattern
	player *PlayerBinding
}

// TurnStepBegins matches a step pattern and saves the player whose step begins.
// Use FromPlayer in the event body to read that player without a target.
func TurnStepBegins(step TurnStepPattern, player *PlayerBinding) TurnStepEventPattern {
	if nilDefinition(step) {
		panic("step event: supply a pattern")
	}
	return turnStepEventPattern{step, player}
}

func (turnStepEventPattern) eventPattern() {}
func (p turnStepEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return p.step.makeStepTrigger(effect)
}
func (p turnStepEventPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "beginning of " + InspectDefinition(p.step).Text,
		bindingWrite: p.player, children: []definitionEdge{{"step", p.step}, {"player", p.player}}}
}
func (p turnStepEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.player, event.PlayerID}, Sequence(effects...))}
}

func (p turnStepEventPattern) delayedStep(controller uuid.UUID) *DelayedTrigger {
	trigger := p.step.delayedStep(controller)
	trigger.eventPlayerBinding = p.player
	return trigger
}

// --- from composable_trigger.go ---
// EventPattern describes an event at detection time.
type EventPattern interface {
	eventPattern()
	makeTrigger(ResolvingEffect) *GenericTriggered
}
type sourceEventPattern struct {
	dies                 bool
	leaves               bool
	colorsSpent          *NumberBinding
	captureColorsSpent   bool
	duplicateColorsSpent bool
}

func (sourceEventPattern) eventPattern() {}

// Enters matches the source entering the battlefield.
func Enters(sourceRef) sourceEventPattern { return sourceEventPattern{} }

// Dies matches the source leaving the battlefield for the graveyard.
func Dies(sourceRef) sourceEventPattern { return sourceEventPattern{dies: true} }

// LeavesBattlefield matches the source moving from the battlefield to another zone.
func LeavesBattlefield(sourceRef) sourceEventPattern { return sourceEventPattern{leaves: true} }

// WithColorsSpent saves the former spell payment when this permanent enters.
func (p sourceEventPattern) WithColorsSpent(binding *NumberBinding) sourceEventPattern {
	if p.captureColorsSpent {
		p.duplicateColorsSpent = true
	}
	p.colorsSpent = binding
	p.captureColorsSpent = true
	return p
}
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
	n := definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true}
	if p.captureColorsSpent {
		n.bindingWrites = append(n.bindingWrites, p.colorsSpent)
		n.children = append(n.children, definitionEdge{"colors spent", &eventAmountCapture{binding: p.colorsSpent}})
		if p.dies || p.leaves {
			n.problem = "colors spent capture requires an entry event"
		} else if p.colorsSpent == nil {
			n.problem = "entry event: supply a colors-spent binding"
		} else if p.duplicateColorsSpent {
			n.problem = "entry event: supply one colors-spent binding"
		}
	}
	return n
}

func (p sourceEventPattern) captureDetectedEvent(event *GameEvent, _ GameReader, _ uuid.UUID) []Effect {
	if !p.captureColorsSpent || p.colorsSpent == nil {
		return nil
	}
	return []Effect{&eventAmountCapture{binding: p.colorsSpent, amount: event.CastColorsSpent}}
}

type creatureEntersPattern struct {
	controllerOnly      bool
	otherOnly           bool
	nontokenOnly        bool
	subtype             string
	filterSubtype       bool
	keyword             Keyword
	filterKeyword       bool
	duplicateKeyword    bool
	permanent           *PermanentBinding
	capturePermanent    bool
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
	power               *NumberBinding
	capturePower        bool
	duplicatePower      bool
}

// CreatureYouControlEnters matches a creature that enters under your control.
func CreatureYouControlEnters() creatureEntersPattern {
	return creatureEntersPattern{controllerOnly: true}
}

// AnotherCreatureYouControlEnters matches a different creature that enters under your control.
func AnotherCreatureYouControlEnters() creatureEntersPattern {
	return creatureEntersPattern{controllerOnly: true, otherOnly: true}
}

// AnotherCreatureEnters matches a creature other than the source under any controller.
func AnotherCreatureEnters() creatureEntersPattern {
	return creatureEntersPattern{otherOnly: true}
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

// WithController saves the entering creature's controller at entry.
func (p creatureEntersPattern) WithController(controller *PlayerBinding) creatureEntersPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = controller, true
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
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "entry event: supply one controller binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.controller)
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
		}
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
	if !p.capturePermanent && !p.capturePower && !p.captureController {
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
	if p.captureController && p.controller != nil {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: permanent.ControllerID()})
	}
	if p.capturePower && p.power != nil {
		if game, ok := observer.(*Game); ok {
			captures = append(captures, &eventAmountCapture{binding: p.power, amount: permanent.CurrentPower(game)})
		}
	}
	return captures
}

type creatureDiesPattern struct {
	creatureOnly       bool
	permanentType      CardType
	filterType         bool
	duplicateType      bool
	subtype            string
	filterSubtype      bool
	duplicateSubtype   bool
	controllerOnly     bool
	duplicateControl   bool
	opponentOnly       bool
	otherOnly          bool
	nontokenOnly       bool
	attachedOnly       bool
	controller         *PlayerBinding
	captureController  bool
	owner              *PlayerBinding
	captureOwner       bool
	power              *NumberBinding
	capturePower       bool
	duplicatePower     bool
	toughness          *NumberBinding
	captureToughness   bool
	card               *CardBinding
	captureCard        bool
	permanent          *PermanentBinding
	capturePermanent   bool
	duplicatePermanent bool
	excludedColor      Color
	filterColor        bool
	duplicateColor     bool
	notSacrificed      bool
	duplicateOutcome   bool
	maxPowerToughness  int
	filterPT           bool
	duplicatePT        bool
	damagedByThisTurn  bool
	duplicateDamage    bool
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

// AttachedPermanentDies matches the attached permanent when it moves to a graveyard.
func AttachedPermanentDies() creatureDiesPattern {
	return creatureDiesPattern{attachedOnly: true}
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

// WithPermanent saves the departing battlefield incarnation for its damage source.
func (p creatureDiesPattern) WithPermanent(permanent *PermanentBinding) creatureDiesPattern {
	if p.capturePermanent {
		p.duplicatePermanent = true
	}
	p.permanent = permanent
	p.capturePermanent = true
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
		text = "attached " + subject + " dies"
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
	if p.permanent != nil {
		n.bindingWrites = append(n.bindingWrites, p.permanent)
		n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
	} else if p.capturePermanent {
		n.problem = "death event: supply a permanent binding"
	}
	if p.duplicatePermanent {
		n.problem = "death event: supply one permanent binding"
	}
	return n
}

func (p creatureDiesPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.capturePower && !p.captureToughness && !p.captureOwner && !p.capturePermanent {
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
	if p.capturePermanent && p.permanent != nil && event.SourceIncarnation != uuid.Nil {
		captures = append(captures, &eventPermanentCapture{binding: p.permanent, value: capturedPermanent{event.SourceID, event.SourceIncarnation}})
	}
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

type creatureYouSacrificePattern struct {
	permanent *PermanentBinding
	captures  int
}

// CreatureYouSacrifice matches a creature sacrificed by the source controller.
func CreatureYouSacrifice() creatureYouSacrificePattern { return creatureYouSacrificePattern{} }

// WithPermanent saves the sacrificed battlefield incarnation for this trigger.
func (p creatureYouSacrificePattern) WithPermanent(binding *PermanentBinding) creatureYouSacrificePattern {
	p.permanent = binding
	p.captures++
	return p
}

func (creatureYouSacrificePattern) eventPattern() {}

func (creatureYouSacrificePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtSacrifice, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventPlayerIsController{},
		EventSacrificedPermanentIsCreature{},
	}})
}

func (p creatureYouSacrificePattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "you sacrifice a creature", battlefieldSource: true}
	if p.captures > 1 {
		n.problem = "save the sacrificed permanent only once"
	} else if p.captures == 1 && p.permanent == nil {
		n.problem = "supply a non-nil sacrificed permanent binding"
	} else if p.permanent != nil {
		n.bindingWrite = p.permanent
		n.children = []definitionEdge{{"permanent", &eventPermanentCapture{binding: p.permanent}}}
	}
	return n
}

func (p creatureYouSacrificePattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if p.permanent == nil {
		return effects
	}
	return []Effect{Sequence(&eventPermanentCapture{binding: p.permanent, value: capturedPermanent{event.SourceID, event.SourceIncarnation}}, Sequence(effects...))}
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

// EachBeginCombat matches the start of each combat, including extra combats.
var EachBeginCombat = stepPattern{EvtBeginCombat, false}

// YourBeginCombat matches each combat start on the controller's turn.
var YourBeginCombat = stepPattern{EvtBeginCombat, true}

// YourUpkeep matches the beginning of the controller's upkeep.
var YourUpkeep = stepPattern{EvtUpkeep, true}

// YourMainPhase matches the start of each main phase on the controller's turn.
var YourMainPhase = stepPattern{EvtMainPhase, true}

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
		if before != nil && !before(evt, g, source, controller) {
			return false
		}
		players := capturedTriggerTargetPlayers(&pendingTrigger{ability: t, event: evt, sourceID: source, controller: controller})
		state := StateContext{SourceID: source, ControllerID: controller, players: players}
		if game, ok := g.(*Game); ok {
			state.cards = game.sourceZoneCardFacts(t, source)
		}
		return o.condition.checkState(g, state)
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
