package mage

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_card_type_branch.go ---
type cardTypeBranch struct {
	card   CardRef
	typeID CardType
	lacks  bool
	body   ResolvingEffect
}

// IfCardHasType runs the body if the present card has the given type.
func IfCardHasType(card CardRef, typeID CardType, body ResolvingEffect) ResolvingEffect {
	return &cardTypeBranch{card: card, typeID: typeID, body: body}
}

// IfCardLacksType runs the body if the present card lacks the given type.
func IfCardLacksType(card CardRef, typeID CardType, body ResolvingEffect) ResolvingEffect {
	return &cardTypeBranch{card: card, typeID: typeID, lacks: true, body: body}
}

func (e *cardTypeBranch) Text() string {
	condition := "if card has type"
	if e.lacks {
		condition = "if card lacks type"
	}
	if e.body == nil {
		return condition
	}
	return condition + ", " + e.body.Text()
}
func (e *cardTypeBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *cardTypeBranch) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if ok && card.HasType(e.typeID) != e.lacks {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *cardTypeBranch) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"card", e.card}, {"then", e.body}}}
	if e.typeID < TypeCreature || e.typeID > TypePlaneswalker {
		n.problem = "supply a valid card type"
	}
	return n
}

// --- from composable_cast_condition.go ---
// CastCondition reads facts saved when a spell was cast.
type CastCondition interface {
	castCondition()
	checkCast(*CastContext) bool
	inspectDefinition() definitionNode
}

type anyCastCondition struct{ conditions []CastCondition }

// AnyCast checks whether one or more saved cast facts are true.
func AnyCast(conditions ...CastCondition) CastCondition {
	return anyCastCondition{conditions: append([]CastCondition(nil), conditions...)}
}

func (anyCastCondition) castCondition() {}
func (c anyCastCondition) checkCast(ctx *CastContext) bool {
	for _, condition := range c.conditions {
		if condition.checkCast(ctx) {
			return true
		}
	}
	return false
}
func (c anyCastCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cast-condition", Text: "any cast condition"}
	if len(c.conditions) == 0 {
		n.problem = "cast condition requires at least one condition"
	}
	for _, condition := range c.conditions {
		n.children = append(n.children, definitionEdge{"condition", condition})
	}
	return n
}

type controlledCreatureSubtypeAtCast struct{ subtype string }

// ControlledCreatureSubtypeAtCast checks the caster's creature control at cast time.
func ControlledCreatureSubtypeAtCast(subtype string) CastCondition {
	return controlledCreatureSubtypeAtCast{subtype: subtype}
}

func (controlledCreatureSubtypeAtCast) castCondition() {}
func (c controlledCreatureSubtypeAtCast) checkCast(ctx *CastContext) bool {
	return ctx != nil && ctx.ControllerCreatureSubtypesAtCast[c.subtype]
}
func (c controlledCreatureSubtypeAtCast) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cast-condition", Text: "controller controlled a " + c.subtype + " creature as this spell was cast"}
	if c.subtype == "" {
		n.problem = "cast condition requires a creature subtype"
	} else if family, known := knownSubtypeFamily(c.subtype); known && family != SubtypeCreature {
		n.problem = "cast condition requires a creature subtype"
	}
	return n
}

type revealedSubtypeAtCast struct{ subtype string }

// RevealedSubtypeAtCast checks cards revealed while the spell was cast.
func RevealedSubtypeAtCast(subtype string) CastCondition {
	return revealedSubtypeAtCast{subtype: subtype}
}

func (revealedSubtypeAtCast) castCondition() {}
func (c revealedSubtypeAtCast) checkCast(ctx *CastContext) bool {
	return ctx.RevealedSubtypeAtCast(c.subtype)
}
func (c revealedSubtypeAtCast) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cast-condition", Text: "revealed a " + c.subtype + " card as this spell was cast"}
	if c.subtype == "" {
		n.problem = "cast condition requires a subtype"
	}
	return n
}

type castBranch struct {
	condition CastCondition
	effect    ResolvingEffect
}

// IfCast runs its effect if the spell's saved cast condition is true.
func IfCast(condition CastCondition, effect ResolvingEffect) ResolvingEffect {
	return &castBranch{condition: condition, effect: effect}
}

func (e *castBranch) Text() string                 { return fmt.Sprintf("if cast condition, %s", e.effect.Text()) }
func (e *castBranch) Properties() EffectProperties { return e.effect.Properties() }
func (e *castBranch) Apply(ctx *EffectContext) error {
	if e.condition.checkCast(ctx.Game.ResolvingCastContext()) {
		return e.effect.Apply(ctx)
	}
	return nil
}
func (e *castBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "cast-branch", children: []definitionEdge{{"condition", e.condition}, {"then", e.effect}}}
}

// --- from composable_control_since_turn.go ---
type controlledSinceTurnStartCondition struct{ subject PermanentRef }

// ControlledSinceTurnStart checks continuous control of a battlefield subject.
func ControlledSinceTurnStart(subject PermanentRef) StateCondition {
	return controlledSinceTurnStartCondition{subject: subject}
}

func (controlledSinceTurnStartCondition) stateCondition() {}

func (c controlledSinceTurnStartCondition) checkState(g GameReader, ctx StateContext) bool {
	source := g.FindPermanent(ctx.SourceID)
	if source == nil || source.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && source.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	p := source
	switch c.subject.(type) {
	case sourceRef:
	case attachedPermanentRef:
		p = g.FindPermanent(source.AttachedTo)
		if p == nil || source.attachedIncarnation == uuid.Nil || p.incarnationID != source.attachedIncarnation {
			return false
		}
	default:
		return false
	}
	return !p.PhasedOut && ctx.ControllerID != uuid.Nil && p.ControllerID() == ctx.ControllerID && p.ControlledSinceTurnStart(g)
}

func (c controlledSinceTurnStartCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", battlefieldSource: true}
	switch c.subject.(type) {
	case sourceRef:
		n.Text = "source controlled since turn start"
	case attachedPermanentRef:
		n.Text = "attached permanent controlled since turn start"
	default:
		n.problem = "state condition requires This or AttachedPermanent"
	}
	return n
}

// --- from composable_delayed.go ---
// DelayedStepPattern describes a future turn-step event.
type DelayedStepPattern interface {
	delayedStep(uuid.UUID) *DelayedTrigger
}

// DelayedStepOption is a closed option for a delayed step trigger.
type DelayedStepOption interface {
	delayedStepOption()
	applyDelayedStep(*nextStepEffect)
}

func (targetOptions) delayedStepOption() {}
func (o targetOptions) applyDelayedStep(e *nextStepEffect) {
	for _, b := range o.bindings {
		if nilDefinition(b) {
			panic("delayed targets: supply a non-nil binding")
		}
		e.targets = append(e.targets, b.targetSpec())
	}
}

func (interveningIfOption) delayedStepOption() {}
func (o interveningIfOption) applyDelayedStep(e *nextStepEffect) {
	if nilDefinition(o.condition) {
		panic("intervening-if: supply a condition")
	}
	if e.condition != nil {
		panic("intervening-if: supply only one condition")
	}
	e.condition = o.condition
}

type nextStepEffect struct {
	expiresAtCleanup bool
	modes            []Mode
	condition        StateCondition
	targets          []Target
	pattern          DelayedStepPattern
	body             ResolvingEffect
}

// AtNext creates a one-shot trigger for the next matching turn-step event.
// Saved references and the resolving controller are captured when it resolves.
func AtNext(pattern DelayedStepPattern, body ResolvingEffect, options ...DelayedStepOption) ResolvingEffect {
	e := &nextStepEffect{pattern: pattern, body: body}
	for _, option := range options {
		if nilDefinition(option) {
			panic("delayed step: supply a non-nil option")
		}
		option.applyDelayedStep(e)
	}
	return e
}

// AtNextThisTurn creates one trigger for the next matching step this turn.
// Cleanup ends an unused schedule. A pending trigger keeps its saved body.
func AtNextThisTurn(pattern DelayedStepPattern, body ResolvingEffect, options ...DelayedStepOption) ResolvingEffect {
	e := AtNext(pattern, body, options...).(*nextStepEffect)
	e.expiresAtCleanup = true
	return e
}

// ModalDelayedStepOption is a mode or condition for a delayed step trigger.
type ModalDelayedStepOption interface {
	modalDelayedStepOption()
	applyModalDelayedStep(*nextStepEffect)
}

func (ModeDefinition) modalDelayedStepOption() {}
func (m ModeDefinition) applyModalDelayedStep(e *nextStepEffect) {
	e.modes = append(e.modes, m.mode)
}
func (interveningIfOption) modalDelayedStepOption() {}
func (o interveningIfOption) applyModalDelayedStep(e *nextStepEffect) {
	o.applyDelayedStep(e)
}

// AtNextModes creates a one-shot step trigger with a mode chosen at stack placement.
func AtNextModes(pattern DelayedStepPattern, options ...ModalDelayedStepOption) ResolvingEffect {
	if nilDefinition(pattern) {
		panic("delayed step: supply a pattern")
	}
	e := &nextStepEffect{pattern: pattern}
	for _, option := range options {
		if nilDefinition(option) {
			panic("delayed modes: supply a non-nil option")
		}
		option.applyModalDelayedStep(e)
	}
	if len(e.modes) < 2 {
		panic("modes: supply at least two mode bodies")
	}
	return e
}

func (e *nextStepEffect) Text() string {
	if e.expiresAtCleanup {
		return "at the next matching turn step this turn"
	}
	return "at the next matching turn step"
}
func (e *nextStepEffect) Properties() EffectProperties {
	if len(e.modes) > 0 {
		var effects []Effect
		for _, mode := range e.modes {
			effects = append(effects, mode.Effects...)
		}
		return Sequence(effects...).Properties()
	}
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *nextStepEffect) inspectDefinition() definitionNode {
	children := declaredTargetNodes(e.targets)
	children = append(children, definitionEdge{"pattern", e.pattern})
	if e.condition != nil {
		children = append(children, definitionEdge{"condition", e.condition})
	}
	if len(e.modes) > 0 {
		children = append(children, modeEdges(e.modes)...)
	} else {
		children = append(children, definitionEdge{"body", e.body})
	}
	return definitionNode{Role: "delayed-step", Text: e.Text(), isolatedBindings: true, children: children}
}
func (e *nextStepEffect) Apply(ctx *EffectContext) error {
	dt := e.pattern.delayedStep(ctx.Controller)
	dt.expiresAtCleanup = e.expiresAtCleanup
	body := captureDelayedBody(ctx, e.body)
	dt.interveningCondition = e.condition
	if e.condition != nil && e.body != nil {
		body.body = IfState(e.condition, body.body)
	}
	dt.targets = slices.Clone(e.targets)
	dt.SourceID = ctx.SourceID
	dt.Controller = ctx.Controller
	dt.sourceCaptured = true
	dt.sourceIncarnation = body.sourceIncarnation
	if len(e.modes) > 0 {
		dt.modes = slices.Clone(e.modes)
		for i, mode := range dt.modes {
			captured := *body
			captured.body = Sequence(mode.Effects...)
			if e.condition != nil {
				captured.body = IfState(e.condition, captured.body)
			}
			dt.modes[i].Effects = []Effect{&captured}
			dt.modes[i].Targets = slices.Clone(mode.Targets)
		}
	} else {
		dt.Effects = []Effect{body}
	}
	ctx.Game.RegisterDelayedTrigger(dt)
	return nil
}

type capturedDelayedBody struct {
	body               ResolvingEffect
	cards              map[*CardBinding]capturedCard
	cardSets           map[*CardsBinding][]capturedCard
	spells             map[*SpellBinding]uuid.UUID
	spellSources       map[*SpellBinding]DamageSource
	targetSpellSources map[uuid.UUID]DamageSource
	damageSources      map[*DamageSourceBinding]DamageSource
	damageResults      map[*DamageResultsBinding][]damageOutcome
	permanents         map[*PermanentBinding]capturedPermanent
	createdTokens      map[*CreatedTokensBinding][]capturedPermanent
	players            map[*PlayerBinding]uuid.UUID
	words              map[*wordPairBinding]wordPairResult
	names              map[*CardNameBinding]string
	numbers            map[*NumberBinding]int
	manaTypes          map[*ManaTypesBinding][]Color
	targets            map[*targetDeclaration][]capturedPermanent
	sourceIncarnation  uuid.UUID
	x                  int
}

func cloneDelayedTargets(targets map[*targetDeclaration][]capturedPermanent) map[*targetDeclaration][]capturedPermanent {
	result := maps.Clone(targets)
	for binding, values := range result {
		result[binding] = slices.Clone(values)
	}
	return result
}
func (e *capturedDelayedBody) Text() string                 { return e.body.Text() }
func (e *capturedDelayedBody) Properties() EffectProperties { return e.body.Properties() }
func (e *capturedDelayedBody) Apply(ctx *EffectContext) error {
	state := &ctx.Game.resolution
	outerCards, outerCardSets := state.cardBindings, state.cardsBindings
	state.cardBindings = maps.Clone(e.cards)
	state.cardsBindings = cloneCardsBindings(e.cardSets)
	defer func() { state.cardBindings = outerCards; state.cardsBindings = outerCardSets }()
	outerResults := state.damageResultsBindings
	state.damageResultsBindings = cloneDamageResultsBindings(e.damageResults)
	defer func() { state.damageResultsBindings = outerResults }()
	outerDamageSources := state.damageSourceBindings
	state.damageSourceBindings = cloneDamageSourceBindings(e.damageSources)
	defer func() { state.damageSourceBindings = outerDamageSources }()
	outerSpells := state.spellBindings
	state.spellBindings = maps.Clone(e.spells)
	defer func() { state.spellBindings = outerSpells }()
	outerNumbers := state.numberBindings
	state.numberBindings = maps.Clone(e.numbers)
	defer func() { state.numberBindings = outerNumbers }()
	outerManaTypes := state.manaTypesBindings
	state.manaTypesBindings = cloneManaTypesBindings(e.manaTypes)
	defer func() { state.manaTypesBindings = outerManaTypes }()
	outerWords := state.wordPairBindings
	state.wordPairBindings = maps.Clone(e.words)
	defer func() { state.wordPairBindings = outerWords }()
	outerNames := state.cardNameBindings
	state.cardNameBindings = maps.Clone(e.names)
	defer func() { state.cardNameBindings = outerNames }()
	outerPlayers := state.playerBindings
	state.playerBindings = maps.Clone(e.players)
	defer func() { state.playerBindings = outerPlayers }()
	outer, x := state.permanentBindings, state.X()
	state.permanentBindings = maps.Clone(e.permanents)
	state.SetX(e.x)
	defer func() { state.permanentBindings = outer; state.SetX(x) }()
	outerCreated := state.createdTokenBindings
	state.createdTokenBindings = cloneCreatedTokenBindings(e.createdTokens)
	defer func() { state.createdTokenBindings = outerCreated }()
	child := *ctx
	child.targetBindings = cloneDelayedTargets(e.targets)
	if child.targetBindings == nil {
		child.targetBindings = make(map[*targetDeclaration][]capturedPermanent)
	}
	for binding, values := range ctx.targetBindings {
		child.targetBindings[binding] = slices.Clone(values)
	}
	child.sourceCaptured = true
	child.sourceIncarnation = e.sourceIncarnation
	return e.body.Apply(&child)
}

// --- from composable_delayed_death.go ---
type delayedPermanentEventEffect struct {
	once         bool
	untaps       bool
	subject      PermanentRef
	body         ResolvingEffect
	leaves       bool
	unblocked    bool
	deathCapture *creatureDiesPattern
}

// WhenDiesThisTurn watches one battlefield object until this turn ends.
func WhenDiesThisTurn(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedPermanentEventEffect{subject: subject, body: body}
}

// WhenDiesThisTurnWithBindings saves the watched object's facts at death.
// The body uses these bindings only within its saved resolution frame.
func WhenDiesThisTurnWithBindings(subject PermanentRef, permanent *PermanentBinding, owner, controller *PlayerBinding, body ResolvingEffect) ResolvingEffect {
	capture := PermanentDies().WithPermanent(permanent).WithOwner(owner).WithController(controller)
	return &delayedPermanentEventEffect{subject: subject, body: body, deathCapture: &capture}
}

// WhenLeavesThisTurn watches one battlefield object until this turn ends.
func WhenLeavesThisTurn(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedPermanentEventEffect{subject: subject, body: body, leaves: true}
}

// WhenLeaves watches the next departure of one exact battlefield visit.
func WhenLeaves(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedPermanentEventEffect{subject: subject, body: body, leaves: true, once: true}
}

// WhenUntaps watches the next actual untap of one exact battlefield visit.
func WhenUntaps(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedPermanentEventEffect{subject: subject, body: body, untaps: true, once: true}
}

// WhenLeavesOrUntaps consumes one trigger at the first matching event.
func WhenLeavesOrUntaps(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedPermanentEventEffect{subject: subject, body: body, leaves: true, untaps: true, once: true}
}

// WhenAttacksUnblockedThisTurn watches the original object in each combat this turn.
func WhenAttacksUnblockedThisTurn(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedPermanentEventEffect{subject: subject, body: body, unblocked: true}
}

func (e *delayedPermanentEventEffect) Text() string {
	if e.once {
		if e.leaves && e.untaps {
			return "when that permanent leaves the battlefield or becomes untapped"
		}
		if e.leaves {
			return "when that permanent leaves the battlefield"
		}
		return "when that permanent becomes untapped"
	}
	if e.unblocked {
		return "when that creature attacks unblocked this turn"
	}
	if e.leaves {
		return "when that permanent leaves the battlefield this turn"
	}
	return "when that permanent dies this turn"
}

func (e *delayedPermanentEventEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || effectReadsSourcePermanent(e.body)
}

func (e *delayedPermanentEventEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}

func (e *delayedPermanentEventEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "delayed-event", Text: e.Text(), isolatedBindings: true,
		children: []definitionEdge{{"subject", e.subject}, {"body", e.body}}}
	if nilDefinition(e.subject) || nilDefinition(e.body) {
		n.problem = "delayed permanent event: supply a subject and body"
	}
	if e.deathCapture != nil {
		capture := e.deathCapture
		if capture.permanent == nil || capture.owner == nil || capture.controller == nil || capture.owner == capture.controller {
			n.problem = "delayed death: supply a permanent binding and separate owner and controller bindings"
		}
		n.children[1].value = Sequence(&eventPermanentCapture{binding: capture.permanent}, &eventPlayerCapture{binding: capture.owner}, &eventPlayerCapture{binding: capture.controller}, e.body)
	}
	return n
}

func (e *delayedPermanentEventEffect) Apply(ctx *EffectContext) error {
	permanent, ok := e.subject.resolvePermanent(ctx)
	if !ok || permanent.PhasedOut {
		return nil
	}
	body := captureDelayedBody(ctx, e.body)
	dt := &DelayedTrigger{
		EventType:         EvtZoneChange,
		MatchEventID:      permanent.ID(),
		MatchFromZone:     ZoneBattlefield,
		SourceID:          ctx.SourceID,
		Controller:        ctx.Controller,
		Effects:           []Effect{body},
		sourceCaptured:    true,
		sourceIncarnation: body.sourceIncarnation,
		matchIncarnation:  permanent.incarnationID,
		deathCapture:      e.deathCapture,
		Persistent:        !e.once,
	}
	if e.untaps {
		if e.leaves {
			dt.alternateEventType = EvtBecameUntapped
		} else {
			dt.EventType = EvtBecameUntapped
			dt.MatchFromZone = ZoneAny
		}
	} else if e.unblocked {
		dt.EventType = EvtAttacksUnblocked
		dt.MatchFromZone = ZoneAny
	} else if !e.leaves {
		dt.MatchToZone = ZoneGraveyard
	}
	ctx.Game.RegisterDelayedTrigger(dt)
	return nil
}

// --- from composable_permanent_subtype_branch.go ---
// IfPermanentHasEnchantmentSubtype checks an enchantment subtype during resolution.
func IfPermanentHasEnchantmentSubtype(subject PermanentRef, subtype string, body ResolvingEffect) ResolvingEffect {
	return &permanentEnchantmentSubtypeBranch{subject: subject, subtype: subtype, body: body}
}

type permanentEnchantmentSubtypeBranch struct {
	subject PermanentRef
	subtype string
	body    ResolvingEffect
}

func (e *permanentEnchantmentSubtypeBranch) Text() string {
	return "if permanent has enchantment subtype " + e.subtype
}

func (e *permanentEnchantmentSubtypeBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}

func (e *permanentEnchantmentSubtypeBranch) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		if p.HasSubType(e.subtype) {
			return e.body.Apply(ctx)
		}
		return nil
	}
	var captured capturedPermanent
	switch ref := e.subject.(type) {
	case savedPermanentRef:
		captured = ctx.Game.resolution.permanentBindings[ref.binding]
	case playerChosenPermanentRef:
		captured, _ = ref.informationIdentity(ctx)
	case sourceRef:
		if ctx.sourceCaptured {
			captured = capturedPermanent{id: ctx.SourceID, incarnation: ctx.sourceIncarnation}
		}
	}
	if captured.incarnation != uuid.Nil {
		if lki := ctx.Game.zones.LKIByIncarnation(captured.incarnation); lki != nil && lki.ID == captured.id && lki.HasSubType(e.subtype) {
			return e.body.Apply(ctx)
		}
	}
	return nil
}

func (e *permanentEnchantmentSubtypeBranch) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"permanent", e.subject}, {"then", e.body}}}
	if !enchantmentSubtypes[e.subtype] {
		n.problem = "supply an enchantment subtype"
	}
	return n
}

// --- from composable_player_controls.go ---
type playerControlsCondition struct {
	player PlayerRef
	filter PermanentFilter
}

// PlayerControls checks the current control of matching battlefield permanents.
func PlayerControls(player PlayerRef, filter PermanentFilter) StateCondition {
	return playerControlsCondition{player: player, filter: filter}
}

func (playerControlsCondition) stateCondition() {}

func (c playerControlsCondition) checkState(g GameReader, ctx StateContext) bool {
	playerID := resolveStatePlayer(g, ctx, c.player)
	if playerID == uuid.Nil || g.GetPlayer(playerID) == nil {
		return false
	}
	for _, permanent := range g.FilterBattlefield(c.filter) {
		if permanent.ControllerID() == playerID {
			return true
		}
	}
	return false
}

func (c playerControlsCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "player controls matching permanent"}
	switch ref := c.player.(type) {
	case controllerRef, activePlayerRef, savedPlayerRef, combatDefendingPlayerRef:
		n.children = []definitionEdge{{"player", c.player}}
	case chosenPermanentPlayerRef:
		n.battlefieldSource = true
		n.children = []definitionEdge{{"player", c.player}}
		if _, ok := ref.subject.(sourceRef); !ok {
			n.problem = "state condition chosen player requires This"
		}
	default:
		n.problem = "state condition requires You, ActivePlayer, a saved player, the combat defending player, or the source chosen player"
	}
	return n
}

type opponentControlsCondition struct{ filter PermanentFilter }

// OpponentControls checks permanents that the controller's opponent controls.
func OpponentControls(filter PermanentFilter) StateCondition {
	return opponentControlsCondition{filter: filter}
}

func (opponentControlsCondition) stateCondition() {}

func (c opponentControlsCondition) checkState(g GameReader, ctx StateContext) bool {
	if g.GetPlayer(ctx.ControllerID) == nil {
		return false
	}
	opponent := g.GetOpponent(ctx.ControllerID)
	if opponent == nil {
		return false
	}
	return g.AnyBattlefield(And(ControlledBy(opponent.PlayerID()), c.filter))
}

func (opponentControlsCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "opponent controls matching permanent"}
}

// --- from composable_positive_branch.go ---
// IfPositive runs the body when the resolved value is greater than zero.
func IfPositive(value ResolutionValue, body ResolvingEffect) ResolvingEffect {
	return &positiveBranch{value: value, body: body}
}

type positiveBranch struct {
	value ResolutionValue
	body  ResolvingEffect
}

func (e *positiveBranch) Text() string                 { return "if positive, " + e.body.Text() }
func (e *positiveBranch) Properties() EffectProperties { return e.body.Properties() }
func (e *positiveBranch) Apply(ctx *EffectContext) error {
	if e.value.resolveValue(ctx) > 0 {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *positiveBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", Text: "if positive", children: []definitionEdge{{"value", e.value}, {"then", e.body}}}
}

// --- from composable_random.go ---
// CoinBranch selects a result of a coin flip during resolution.
type CoinBranch interface {
	coinBranch()
	coinResult() (bool, ResolvingEffect)
}

type coinBranch struct {
	win    bool
	effect ResolvingEffect
}

func (coinBranch) coinBranch()                           {}
func (b coinBranch) coinResult() (bool, ResolvingEffect) { return b.win, b.effect }
func (b coinBranch) inspectDefinition() definitionNode {
	role := "on-loss"
	if b.win {
		role = "on-win"
	}
	return definitionNode{Role: role, children: []definitionEdge{{"effect", b.effect}}}
}

// OnWin runs the effect if the instructed player wins the flip.
func OnWin(effect ResolvingEffect) CoinBranch { return coinBranch{true, effect} }

// OnLoss runs the effect if the instructed player loses the flip.
func OnLoss(effect ResolvingEffect) CoinBranch { return coinBranch{false, effect} }

type coinFlipEffect struct {
	player   PlayerRef
	branches []CoinBranch
}

// FlipCoin makes one choice through the game's random source at resolution.
// An omitted result has no effect. Each supplied result must be unique.
func FlipCoin(player PlayerRef, branches ...CoinBranch) ResolvingEffect {
	return &coinFlipEffect{player, append([]CoinBranch(nil), branches...)}
}
func (*coinFlipEffect) Text() string { return "flip a coin" }
func (e *coinFlipEffect) Properties() EffectProperties {
	var effects []Effect
	for _, branch := range e.branches {
		if !nilDefinition(branch) {
			_, effect := branch.coinResult()
			if !nilDefinition(effect) {
				effects = append(effects, effect)
			}
		}
	}
	return CompositeEffects("", effects...).Properties()
}
func (e *coinFlipEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	win := ctx.Game.FlipCoin(player)
	for _, branch := range e.branches {
		result, effect := branch.coinResult()
		if result == win {
			return effect.Apply(ctx)
		}
	}
	return nil
}
func (e *coinFlipEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "coin-flip", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if len(e.branches) == 0 {
		n.problem = "supply at least one coin branch"
	}
	seen := map[bool]bool{}
	for i, branch := range e.branches {
		if !nilDefinition(branch) {
			result, _ := branch.coinResult()
			if seen[result] {
				n.problem = "remove duplicate coin result branch"
			}
			seen[result] = true
		}
		n.children = append(n.children, definitionEdge{fmt.Sprintf("branches[%d]", i), branch})
	}
	return n
}

// --- from composable_state.go ---
// StateCondition is a pure condition that can be checked more than once.
type StateCondition interface {
	stateCondition()
	checkState(GameReader, StateContext) bool
	inspectDefinition() definitionNode
}

// StateContext identifies the source and controller for a condition check.
type StateContext struct {
	permanents        map[*PermanentBinding]capturedPermanent
	cards             map[*CardBinding]capturedCard
	players           map[*PlayerBinding]uuid.UUID
	SourceID          uuid.UUID
	ControllerID      uuid.UUID
	SourceIncarnation uuid.UUID
}

type yourTurnCondition struct{}

type controllerGainedLifeThisTurnCondition struct{}

type sourceDamagedOpponentThisTurnCondition struct{}

// SourceDamagedOpponentThisTurnCondition checks this source's turn damage to an opponent.
func SourceDamagedOpponentThisTurnCondition() StateCondition {
	return sourceDamagedOpponentThisTurnCondition{}
}

func (sourceDamagedOpponentThisTurnCondition) stateCondition() {}
func (sourceDamagedOpponentThisTurnCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	if p == nil || p.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && p.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	opponent := g.GetOpponent(ctx.ControllerID)
	return opponent != nil && g.HasDealtDamageToPlayerThisTurn(p.ID(), p.incarnationID, opponent.PlayerID())
}
func (sourceDamagedOpponentThisTurnCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "source dealt damage to an opponent this turn", battlefieldSource: true}
}

// ControllerGainedLifeThisTurn checks actual life gained in the current turn.
func ControllerGainedLifeThisTurn() StateCondition { return controllerGainedLifeThisTurnCondition{} }

func (controllerGainedLifeThisTurnCondition) stateCondition() {}
func (controllerGainedLifeThisTurnCondition) checkState(g GameReader, ctx StateContext) bool {
	return ctx.ControllerID != uuid.Nil && g.PlayerLifeGainedThisTurn(ctx.ControllerID) > 0
}
func (controllerGainedLifeThisTurnCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "controller gained life this turn"}
}

type uniqueHighestLifeCondition struct{}

type controllerLifeAtLeastStartingLifeCondition struct{}

// ControllerLifeAtLeastStartingLife compares current life with its starting total.
func ControllerLifeAtLeastStartingLife() StateCondition {
	return controllerLifeAtLeastStartingLifeCondition{}
}

func (controllerLifeAtLeastStartingLifeCondition) stateCondition() {}
func (controllerLifeAtLeastStartingLifeCondition) checkState(g GameReader, ctx StateContext) bool {
	if ctx.ControllerID == uuid.Nil {
		return false
	}
	player := g.GetPlayer(ctx.ControllerID)
	return player != nil && player.Life() >= player.StartingLife()
}
func (controllerLifeAtLeastStartingLifeCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "controller life is at least starting life"}
}

// HasUniqueHighestLife checks whether one player alone has the most life.
func HasUniqueHighestLife() StateCondition { return uniqueHighestLifeCondition{} }

func (uniqueHighestLifeCondition) stateCondition() {}
func (uniqueHighestLifeCondition) checkState(g GameReader, _ StateContext) bool {
	players := g.AllPlayers()
	if len(players) < 2 {
		return false
	}
	best, ties := players[0].Life(), 1
	for _, player := range players[1:] {
		switch {
		case player.Life() > best:
			best, ties = player.Life(), 1
		case player.Life() == best:
			ties++
		}
	}
	return ties == 1
}
func (uniqueHighestLifeCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "one player has the highest life total"}
}

type controllerHandSizeCondition struct{ size int }

// ControllerHandSizeIs checks the ability controller's hand when evaluated.
func ControllerHandSizeIs(size int) StateCondition { return controllerHandSizeCondition{size: size} }

func (controllerHandSizeCondition) stateCondition() {}
func (c controllerHandSizeCondition) checkState(g GameReader, ctx StateContext) bool {
	if ctx.ControllerID == uuid.Nil {
		return false
	}
	player := g.GetPlayer(ctx.ControllerID)
	return player != nil && len(player.Hand()) == c.size
}
func (c controllerHandSizeCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: fmt.Sprintf("controller hand size is %d", c.size)}
	if c.size < 0 {
		n.problem = "hand size must be nonnegative"
	}
	return n
}

// IsYourTurn checks the active player when the condition is evaluated.
func IsYourTurn() StateCondition { return yourTurnCondition{} }

func (yourTurnCondition) stateCondition() {}
func (yourTurnCondition) checkState(g GameReader, ctx StateContext) bool {
	active := g.ActivePlayerObj()
	return active != nil && ctx.ControllerID != uuid.Nil && active.PlayerID() == ctx.ControllerID
}
func (yourTurnCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "it is your turn"}
}

type sourceTappedCondition struct{ tapped bool }

type permanentTappedCondition struct {
	subject PermanentRef
	tapped  bool
}

type permanentAttackingCondition struct{ subject PermanentRef }
type permanentBlockingCondition struct{ subject PermanentRef }

// PermanentIsAttacking checks current combat membership for the selected permanent.
func PermanentIsAttacking(subject PermanentRef) StateCondition {
	return permanentAttackingCondition{subject: subject}
}

func (permanentAttackingCondition) stateCondition() {}
func (c permanentAttackingCondition) checkState(g GameReader, ctx StateContext) bool {
	source := g.FindPermanent(ctx.SourceID)
	if source == nil || source.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && source.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	p := source
	switch c.subject.(type) {
	case sourceRef:
	case attachedPermanentRef:
		p = g.FindPermanent(source.AttachedTo)
		if p == nil || source.attachedIncarnation == uuid.Nil || p.incarnationID != source.attachedIncarnation {
			return false
		}
	default:
		return false
	}
	return !p.PhasedOut && g.IsAttackingIncarnation(p.ID(), p.incarnationID)
}
func (c permanentAttackingCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", battlefieldSource: true}
	switch c.subject.(type) {
	case sourceRef:
		n.Text = "source is attacking"
	case attachedPermanentRef:
		n.Text = "attached permanent is attacking"
	default:
		n.problem = "state condition requires This or AttachedPermanent"
	}
	return n
}

// PermanentIsBlocking checks current combat membership for the selected permanent.
func PermanentIsBlocking(subject PermanentRef) StateCondition {
	return permanentBlockingCondition{subject: subject}
}

func (permanentBlockingCondition) stateCondition() {}
func (c permanentBlockingCondition) checkState(g GameReader, ctx StateContext) bool {
	source := g.FindPermanent(ctx.SourceID)
	if source == nil || source.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && source.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	p := source
	switch c.subject.(type) {
	case sourceRef:
	case attachedPermanentRef:
		p = g.FindPermanent(source.AttachedTo)
		if p == nil || source.attachedIncarnation == uuid.Nil || p.incarnationID != source.attachedIncarnation {
			return false
		}
	default:
		return false
	}
	return !p.PhasedOut && g.IsBlockingIncarnation(p.ID(), p.incarnationID)
}
func (c permanentBlockingCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", battlefieldSource: true}
	switch c.subject.(type) {
	case sourceRef:
		n.Text = "source is blocking"
	case attachedPermanentRef:
		n.Text = "attached permanent is blocking"
	default:
		n.problem = "state condition requires This or AttachedPermanent"
	}
	return n
}

// PermanentIsTapped checks the current source or its attached permanent.
func PermanentIsTapped(subject PermanentRef) StateCondition {
	return permanentTappedCondition{subject: subject, tapped: true}
}

// PermanentIsUntapped checks the current source or its attached permanent.
func PermanentIsUntapped(subject PermanentRef) StateCondition {
	return permanentTappedCondition{subject: subject}
}

func (permanentTappedCondition) stateCondition() {}
func (c permanentTappedCondition) checkState(g GameReader, ctx StateContext) bool {
	source := g.FindPermanent(ctx.SourceID)
	if source == nil || source.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && source.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	p := source
	switch c.subject.(type) {
	case sourceRef:
	case attachedPermanentRef:
		p = g.FindPermanent(source.AttachedTo)
		if p == nil || source.attachedIncarnation == uuid.Nil || p.incarnationID != source.attachedIncarnation {
			return false
		}
	default:
		return false
	}
	return !p.PhasedOut && p.Tapped == c.tapped
}
func (c permanentTappedCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", battlefieldSource: true}
	switch c.subject.(type) {
	case sourceRef:
		n.Text = "source is tapped"
	case attachedPermanentRef:
		n.Text = "attached permanent is tapped"
	default:
		n.problem = "state condition requires This or AttachedPermanent"
	}
	if !c.tapped {
		n.Text = strings.Replace(n.Text, "is tapped", "is untapped", 1)
	}
	return n
}

// SourceTappedCondition checks the current battlefield source.
func SourceTappedCondition() StateCondition { return sourceTappedCondition{tapped: true} }

// SourceUntappedCondition checks the current battlefield source.
func SourceUntappedCondition() StateCondition { return sourceTappedCondition{tapped: false} }

func (sourceTappedCondition) stateCondition() {}
func (c sourceTappedCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	return p != nil && !p.PhasedOut && (ctx.SourceIncarnation == uuid.Nil || p.incarnationID == ctx.SourceIncarnation) && p.Tapped == c.tapped
}
func (c sourceTappedCondition) inspectDefinition() definitionNode {
	text := "source is untapped"
	if c.tapped {
		text = "source is tapped"
	}
	return definitionNode{Role: "state-condition", Text: text, battlefieldSource: true}
}

type sourceBlockingCondition struct{}

// SourceBlockingCondition checks whether the current source is blocking.
func SourceBlockingCondition() StateCondition { return sourceBlockingCondition{} }

func (sourceBlockingCondition) stateCondition() {}
func (sourceBlockingCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	return p != nil && !p.PhasedOut &&
		(ctx.SourceIncarnation == uuid.Nil || p.incarnationID == ctx.SourceIncarnation) &&
		g.IsBlockingIncarnation(ctx.SourceID, p.incarnationID)
}
func (sourceBlockingCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "source is blocking", battlefieldSource: true}
}

type sourceAttackedThisTurnCondition struct{}

// SourceAttackedThisTurnCondition checks the current source's attack history.
func SourceAttackedThisTurnCondition() StateCondition { return sourceAttackedThisTurnCondition{} }

func (sourceAttackedThisTurnCondition) stateCondition() {}
func (sourceAttackedThisTurnCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	if p == nil || p.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && p.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	return g.HasAttackedThisTurnIncarnation(ctx.SourceID, p.incarnationID)
}
func (sourceAttackedThisTurnCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "source attacked this turn", battlefieldSource: true}
}

type sourceAttackedOrBlockedThisTurnCondition struct{}

// SourceAttackedOrBlockedThisTurnCondition checks this incarnation's turn history.
func SourceAttackedOrBlockedThisTurnCondition() StateCondition {
	return sourceAttackedOrBlockedThisTurnCondition{}
}

func (sourceAttackedOrBlockedThisTurnCondition) stateCondition() {}
func (sourceAttackedOrBlockedThisTurnCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	if p == nil || p.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && p.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	return g.HasAttackedOrBlockedThisTurnIncarnation(ctx.SourceID, p.incarnationID)
}
func (sourceAttackedOrBlockedThisTurnCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "source attacked or blocked this turn", battlefieldSource: true}
}

type sourceAttackedOrBlockedThisCombatCondition struct{}

// SourceAttackedOrBlockedThisCombatCondition checks if the source attacked or blocked in this combat.
func SourceAttackedOrBlockedThisCombatCondition() StateCondition {
	return sourceAttackedOrBlockedThisCombatCondition{}
}

func (sourceAttackedOrBlockedThisCombatCondition) stateCondition() {}
func (sourceAttackedOrBlockedThisCombatCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	if p == nil || p.PhasedOut || (ctx.SourceIncarnation != uuid.Nil && p.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	return g.HasAttackedOrBlockedThisCombatIncarnation(ctx.SourceID, p.incarnationID)
}
func (sourceAttackedOrBlockedThisCombatCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "source attacked or blocked this combat", battlefieldSource: true}
}

// StateDependency declares the data that a callback reads.
type StateDependency uint8

const (
	StateBattlefield StateDependency = iota + 1
	StateSourceChoices
)

// StatePermanent is a snapshot. It contains no mutable engine objects.
type StatePermanent struct {
	id, controller, chosenPlayer uuid.UUID
	chosenColor                  Color
	colors                       []Color
	token                        bool
}

func (p StatePermanent) ID() uuid.UUID           { return p.id }
func (p StatePermanent) ControllerID() uuid.UUID { return p.controller }
func (p StatePermanent) ChosenPlayer() uuid.UUID { return p.chosenPlayer }
func (p StatePermanent) ChosenColor() Color      { return p.chosenColor }
func (p StatePermanent) HasColor(c Color) bool   { return slices.Contains(p.colors, c) }
func (p StatePermanent) IsToken() bool           { return p.token }

// StateReader supplies immutable snapshots to pure condition callbacks.
type StateReader interface {
	Permanent(uuid.UUID) (StatePermanent, bool)
	Battlefield() []StatePermanent
	stateReader()
}
type stateSnapshot struct{ permanents []StatePermanent }

func (stateSnapshot) stateReader() {}
func (s stateSnapshot) Permanent(id uuid.UUID) (StatePermanent, bool) {
	for _, p := range s.permanents {
		if p.id == id {
			return p, true
		}
	}
	return StatePermanent{}, false
}
func (s stateSnapshot) Battlefield() []StatePermanent { return slices.Clone(s.permanents) }
func snapshotState(g GameReader) StateReader {
	s := stateSnapshot{}
	for _, p := range g.FilterBattlefield(AnyPermanent) {
		s.permanents = append(s.permanents, StatePermanent{id: p.ID(), controller: p.ControllerID(), chosenPlayer: p.ChosenPlayer, chosenColor: p.ChosenColor, colors: slices.Clone(p.Colors()), token: p.IsToken})
	}
	return s
}

type statePredicate struct {
	label        string
	filter       PermanentFilter
	kind         string
	children     []StateCondition
	dependencies []StateDependency
	fn           func(StateReader, StateContext) bool
}

func (*statePredicate) stateCondition() {}
func (c *statePredicate) checkState(g GameReader, ctx StateContext) bool {
	switch c.kind {
	case "controls", "none":
		found := false
		for _, p := range g.FilterBattlefield(c.filter) {
			if p.ControllerID() == ctx.ControllerID {
				found = true
				break
			}
		}
		return found == (c.kind == "controls")
	case "battlefield-none":
		return len(g.FilterBattlefield(c.filter)) == 0
	case "all":
		for _, child := range c.children {
			if !child.checkState(g, ctx) {
				return false
			}
		}
		return true
	case "any":
		for _, child := range c.children {
			if child.checkState(g, ctx) {
				return true
			}
		}
		return false
	case "not":
		return !c.children[0].checkState(g, ctx)
	default:
		return c.fn(snapshotState(g), ctx)
	}
}

// Controls includes the source if it matches the filter.
func Controls(f PermanentFilter) StateCondition {
	return &statePredicate{kind: "controls", filter: f, label: "you control " + f.Text()}
}

// ControlsNone checks all permanents of the current controller.
func ControlsNone(f PermanentFilter) StateCondition {
	return &statePredicate{kind: "none", filter: f, label: "you control no " + f.Text()}
}

// BattlefieldHasNone checks all players' permanents.
func BattlefieldHasNone(f PermanentFilter) StateCondition {
	return &statePredicate{kind: "battlefield-none", filter: f, label: "no " + f.Text() + " on the battlefield"}
}

// AllConditions requires each condition to be true.
func AllConditions(cs ...StateCondition) StateCondition {
	return &statePredicate{kind: "all", children: slices.Clone(cs)}
}

// AnyCondition requires at least one condition to be true.
func AnyCondition(cs ...StateCondition) StateCondition {
	return &statePredicate{kind: "any", children: slices.Clone(cs)}
}

// NotCondition reverses a condition.
func NotCondition(c StateCondition) StateCondition {
	return &statePredicate{kind: "not", children: []StateCondition{c}}
}

// StateConditionFunc wraps a pure callback. Declare all data that it reads.
// The callback must not use randomness, choices, mutation, or mutable captures.
func StateConditionFunc(label string, dependencies []StateDependency, fn func(StateReader, StateContext) bool) StateCondition {
	return &statePredicate{kind: "callback", label: label, dependencies: slices.Clone(dependencies), fn: fn}
}

// WhenState uses the existing pending-trigger and stack lifecycle.
func WhenState(condition StateCondition, effect ResolvingEffect, options ...TriggerOption) TriggeredAbility {
	t := NewStateTriggered(false, effect)
	t.stateDefinition = condition
	t.dslState = true
	t.Condition = func(_ *GameEvent, g GameReader, source, controller uuid.UUID) bool {
		return condition.checkState(g, StateContext{SourceID: source, ControllerID: controller})
	}
	applyTriggerOptions(t, options)
	return t
}

// IfState checks a pure condition when the effect resolves.
func IfState(condition StateCondition, effect ResolvingEffect) ResolvingEffect {
	return &stateBranch{condition: condition, effect: effect}
}

type stateBranch struct {
	condition StateCondition
	effect    ResolvingEffect
}

func (e *stateBranch) Text() string {
	return "if " + e.condition.inspectDefinition().Text + ", " + e.effect.Text()
}
func (e *stateBranch) Properties() EffectProperties { return e.effect.Properties() }
func (e *stateBranch) Apply(ctx *EffectContext) error {
	if e.condition.checkState(ctx.Game, StateContext{SourceID: ctx.SourceID, ControllerID: ctx.Controller, SourceIncarnation: ctx.sourceIncarnation, players: ctx.Game.resolution.playerBindings, cards: ctx.Game.resolution.cardBindings, permanents: ctx.Game.resolution.permanentBindings}) {
		return e.effect.Apply(ctx)
	}
	return nil
}

// Sequence executes instructions in order through the existing effect executor.
func Sequence(effects ...ResolvingEffect) ResolvingEffect {
	return &sequenceEffect{effects: slices.Clone(effects)}
}

type sequenceEffect struct{ effects []ResolvingEffect }

func (e *sequenceEffect) Text() string {
	texts := make([]string, len(e.effects))
	for i, c := range e.effects {
		texts[i] = c.Text()
	}
	return strings.Join(texts, "; ")
}
func (e *sequenceEffect) Properties() EffectProperties {
	return CompositeEffects("", e.effects...).Properties()
}
func (e *sequenceEffect) Apply(ctx *EffectContext) error {
	return CompositeEffects("", e.effects...).Apply(ctx)
}

func captureDelayedBody(ctx *EffectContext, body ResolvingEffect) *capturedDelayedBody {
	captured := &capturedDelayedBody{
		body:               body,
		permanents:         maps.Clone(ctx.Game.resolution.permanentBindings),
		spells:             maps.Clone(ctx.Game.resolution.spellBindings),
		spellSources:       ctx.Game.captureDelayedSpellSources(),
		targetSpellSources: captureDelayedTargetSpellSources(ctx),
		cards:              maps.Clone(ctx.Game.resolution.cardBindings),
		cardSets:           cloneCardsBindings(ctx.Game.resolution.cardsBindings),
		damageSources:      cloneDamageSourceBindings(ctx.Game.resolution.damageSourceBindings),
		damageResults:      cloneDamageResultsBindings(ctx.Game.resolution.damageResultsBindings),
		createdTokens:      cloneCreatedTokenBindings(ctx.Game.resolution.createdTokenBindings),
		players:            maps.Clone(ctx.Game.resolution.playerBindings),
		words:              maps.Clone(ctx.Game.resolution.wordPairBindings),
		names:              maps.Clone(ctx.Game.resolution.cardNameBindings),
		numbers:            maps.Clone(ctx.Game.resolution.numberBindings),
		manaTypes:          cloneManaTypesBindings(ctx.Game.resolution.manaTypesBindings),
		targets:            cloneDelayedTargets(ctx.targetBindings),
		sourceIncarnation:  ctx.sourceIncarnation,
		x:                  ctx.Game.resolution.X(),
	}
	if !ctx.sourceCaptured {
		if source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); source != nil {
			captured.sourceIncarnation = source.incarnationID
		}
	}
	return captured
}
