package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PlayerRef selects one player during resolution.
type PlayerRef interface {
	DamageRecipientRef
	playerRef()
	resolvePlayer(*EffectContext) (uuid.UUID, bool)
}
type controllerRef struct{}
type activePlayerRef struct{}

// You is the stored resolving controller.
var You = controllerRef{}

// ActivePlayer selects the active player when the effect resolves.
var ActivePlayer = activePlayerRef{}

func (controllerRef) playerRef()   {}
func (activePlayerRef) playerRef() {}
func (controllerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	return ctx.Controller, ctx.Game.GetPlayer(ctx.Controller) != nil
}
func (activePlayerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	p := ctx.Game.ActivePlayerObj()
	if p == nil {
		return uuid.Nil, false
	}
	return p.PlayerID(), true
}
func (controllerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "you"}
}
func (activePlayerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "active player"}
}

type subjectAction struct {
	permanent PermanentRef
	player    PlayerRef
	effect    Effect
	verb      string
	amount    ValueSource
}

func (e *subjectAction) Text() string {
	s := any(e.permanent)
	if e.player != nil {
		s = e.player
	}
	return e.verb + " " + InspectDefinition(s).Text
}
func (e *subjectAction) Properties() EffectProperties { return e.effect.Properties() }
func (e *subjectAction) Apply(ctx *EffectContext) error {
	var id uuid.UUID
	if e.permanent != nil {
		p, ok := e.permanent.resolvePermanent(ctx)
		if !ok {
			return nil
		}
		id = p.ID()
	} else {
		p, ok := e.player.resolvePlayer(ctx)
		if !ok {
			return nil
		}
		id = p
	}
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return e.effect.Apply(&child)
}
func (e *subjectAction) inspectDefinition() definitionNode {
	s := any(e.permanent)
	if e.player != nil {
		s = e.player
	}
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", s}}}
	if e.player != nil || e.verb == "add counters to" {
		n.children = append(n.children, definitionEdge{"amount", e.amount})
	}
	return n
}

// Destroy destroys the selected battlefield incarnation through the normal action.
func Destroy(subject PermanentRef) ResolvingEffect {
	return &subjectAction{permanent: subject, effect: DestroyTarget(), verb: "destroy"}
}

// DestroyWithoutRegeneration destroys the subject and prevents regeneration.
func DestroyWithoutRegeneration(subject PermanentRef) ResolvingEffect {
	return &subjectAction{permanent: subject, effect: DestroyTargetNoRegen(), verb: "destroy without regeneration"}
}

// DestroyWithResult destroys one permanent and saves its action result.
func DestroyWithResult(subject PermanentRef, option DestroyResultOption) ResolvingEffect {
	return newDestroySubjectEffect(subject, false, []DestroyResultOption{option})
}

// DestroyWithoutRegenerationWithResult destroys one permanent without regeneration and saves its result.
func DestroyWithoutRegenerationWithResult(subject PermanentRef, option DestroyResultOption) ResolvingEffect {
	return newDestroySubjectEffect(subject, true, []DestroyResultOption{option})
}

type destroySubjectEffect struct {
	subject          PermanentRef
	cannotRegenerate bool
	result           *NumberBinding
	optionCount      int
	invalidOption    bool
}

// DestroyResultOption selects a result of one destruction instruction.
type DestroyResultOption interface {
	destroyResultOption()
	applyDestroyResult(*destroySubjectEffect)
}

type destroyedThisWayOption struct{ binding *NumberBinding }

// DestroyedThisWayInto saves one when the instruction destroys its subject.
func DestroyedThisWayInto(binding *NumberBinding) DestroyResultOption {
	return destroyedThisWayOption{binding}
}
func (destroyedThisWayOption) destroyResultOption() {}
func (o destroyedThisWayOption) applyDestroyResult(e *destroySubjectEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}
func newDestroySubjectEffect(subject PermanentRef, cannotRegenerate bool, options []DestroyResultOption) ResolvingEffect {
	e := &destroySubjectEffect{subject: subject, cannotRegenerate: cannotRegenerate}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyDestroyResult(e)
	}
	return e
}
func (e *destroySubjectEffect) Text() string { return "destroy permanent" }
func (*destroySubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Destroys: true}
}
func (e *destroySubjectEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		result := ctx.Game.destroyPermanents([]*Permanent{p}, e.cannotRegenerate)
		if e.result != nil {
			ctx.Game.resolution.numberBindings[e.result] = result.destroyed
		}
	}
	return nil
}
func (e *destroySubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.invalidOption {
		n.problem = "supply a non-nil destruction result binding"
	} else if e.optionCount > 1 {
		n.problem = "supply the destruction result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}

// Exile exiles the selected battlefield incarnation.
func Exile(subject PermanentRef) ResolvingEffect {
	return &subjectAction{permanent: subject, effect: &exileTargetEffect{}, verb: "exile"}
}

// ReturnToHand returns the selected permanent to its owner's hand.
func ReturnToHand(subject PermanentRef) ResolvingEffect {
	return &subjectAction{permanent: subject, effect: ReturnToHandTarget(), verb: "return to owner's hand"}
}

// UntapPermanent untaps the selected battlefield incarnation.
func UntapPermanent(subject PermanentRef) ResolvingEffect {
	return &untapPermanentEffect{subject}
}

// DrawFor draws individual cards for the selected player. An absent player has no effect.
func DrawFor(subject PlayerRef, amount ResolutionValue, options ...DrawResultOption) ResolvingEffect {
	e := &drawSubjectEffect{subject: subject, amount: amount}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyDrawResult(e)
	}
	return e
}

type drawSubjectEffect struct {
	subject       PlayerRef
	amount        ResolutionValue
	result        *NumberBinding
	cardResult    *CardBinding
	optionCount   int
	cardCount     int
	invalidOption bool
}

// DrawResultOption selects one result for a draw instruction.
type DrawResultOption interface {
	drawResultOption()
	applyDrawResult(*drawSubjectEffect)
}

type drawAmountOption struct{ binding *NumberBinding }

type drawCardOption struct{ binding *CardBinding }

// DrawCardInto saves the card from a single completed draw.
func DrawCardInto(binding *CardBinding) DrawResultOption { return drawCardOption{binding} }
func (drawCardOption) drawResultOption()                 {}
func (o drawCardOption) applyDrawResult(e *drawSubjectEffect) {
	e.cardResult = o.binding
	e.cardCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DrawAmountInto saves the number of cards drawn by one instruction.
func DrawAmountInto(binding *NumberBinding) DrawResultOption { return drawAmountOption{binding} }
func (drawAmountOption) drawResultOption()                   {}
func (o drawAmountOption) applyDrawResult(e *drawSubjectEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

func (e *drawSubjectEffect) Text() string                 { return "draw for " + InspectDefinition(e.subject).Text }
func (e *drawSubjectEffect) Properties() EffectProperties { return drawAmountProperties(e.amount) }
func (e *drawSubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
	if e.invalidOption {
		n.problem = "supply a non-nil draw result binding"
	} else if e.optionCount > 1 || e.cardCount > 1 {
		n.problem = "supply the draw result binding only once"
	} else if e.result != nil && e.cardResult != nil {
		n.problem = "supply one kind of draw result binding"
	} else if e.cardResult != nil {
		if fixed, ok := e.amount.(fixedValue); !ok || fixed.n != 1 {
			n.problem = "draw one card when saving its card result"
		}
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	if e.cardResult != nil {
		n.bindingWrite = e.cardResult
		n.children = append(n.children, definitionEdge{"card result", e.cardResult})
	}
	return n
}
func (e *drawSubjectEffect) Apply(ctx *EffectContext) error {
	if e.cardResult != nil {
		if ctx.Game.resolution.cardBindings == nil {
			ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
		}
		delete(ctx.Game.resolution.cardBindings, e.cardResult)
	}
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	p := ctx.Game.GetPlayer(id)
	if p == nil {
		return nil
	}
	for range amount {
		if card, drawn := ctx.Game.PlayerDrawCard(p); drawn {
			if e.result != nil {
				ctx.Game.resolution.numberBindings[e.result]++
			}
			if e.cardResult != nil {
				if visitor, ok := p.(interface{ handVisit(uuid.UUID) uuid.UUID }); ok {
					ctx.Game.resolution.cardBindings[e.cardResult] = capturedCard{id: card.ID(), visit: visitor.handVisit(card.ID()), zone: core.ZoneHand}
				}
			}
		}
	}
	return nil
}
func drawAmountProperties(amount ResolutionValue) EffectProperties {
	p := EffectProperties{Outcome: OutcomeBenefit}
	if value, ok := amount.(fixedValue); ok {
		p.DrawCount = value.n
	}
	return p
}

// Discard lets the selected player discard through the existing choice and replacement paths.
func Discard(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &discardSubjectEffect{subject, amount}
}

// PlayerSet selects a collection of players when an operation starts.
type PlayerSet interface {
	playerSet()
	resolvePlayers(*EffectContext) []uuid.UUID
}
type allPlayersRef struct{}

// EachPlayer includes all players in turn order.
var EachPlayer = allPlayersRef{}

func (allPlayersRef) playerSet() {}
func (allPlayersRef) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	var ids []uuid.UUID
	active := ctx.Game.ActivePlayerObj()
	if active != nil {
		ids = append(ids, active.PlayerID())
	}
	for _, p := range ctx.Game.players {
		if active == nil || p.PlayerID() != active.PlayerID() {
			ids = append(ids, p.PlayerID())
		}
	}
	return ids
}
func (allPlayersRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set", Text: "each player"}
}

type drawEachEffect struct {
	subjects PlayerSet
	amount   ResolutionValue
}

// DrawEach draws cards for each selected player, using individual draw actions.
// It reads the amount once before the first draw.
func DrawEach(subjects PlayerSet, amount ResolutionValue) ResolvingEffect {
	return &drawEachEffect{subjects, amount}
}
func (e *drawEachEffect) Text() string { return "draw for each player" }
func (e *drawEachEffect) Properties() EffectProperties {
	p := drawAmountProperties(e.amount)
	p.Mass = true
	return p
}
func (e *drawEachEffect) Apply(ctx *EffectContext) error {
	ids := e.subjects.resolvePlayers(ctx)
	if len(ids) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	for _, id := range ids {
		child := *ctx
		child.Targets = []uuid.UUID{id}
		if err := DrawCards(Fixed(amount)).Apply(&child); err != nil {
			return err
		}
	}
	return nil
}
func (e *drawEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "draw-each", children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
}

// PermanentSet selects battlefield incarnations at the start of an operation.
type PermanentSet interface {
	permanentSet()
	resolvePermanents(*EffectContext) []*Permanent
}
type matchingPermanentSet struct {
	filter     PermanentFilter
	controlled bool
}

// MatchingPermanents selects all matching permanents, including the source.
func MatchingPermanents(filter PermanentFilter) PermanentSet {
	return matchingPermanentSet{filter: filter}
}

// ControlledPermanents selects matching permanents controlled by You.
func ControlledPermanents(filter PermanentFilter) PermanentSet {
	return matchingPermanentSet{filter: filter, controlled: true}
}
func (matchingPermanentSet) permanentSet() {}
func (s matchingPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(s.filter) {
		if !p.PhasedOut && (!s.controlled || p.ControllerID() == ctx.Controller) {
			result = append(result, p)
		}
	}
	return result
}
func (s matchingPermanentSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: s.filter.Text()}
}

type destroyEachEffect struct {
	subjects         PermanentSet
	cannotRegenerate bool
	creatureDeaths   *NumberBinding
	optionCount      int
	invalidOption    bool
}

// DestroyEachResultOption selects a result of one destruction group.
type DestroyEachResultOption interface {
	destroyEachResultOption()
	applyDestroyEachResult(*destroyEachEffect)
}

type creaturesDiedThisWayOption struct{ binding *NumberBinding }

// CreaturesDiedThisWayInto saves the number of creatures moved to a graveyard by this instruction.
func CreaturesDiedThisWayInto(binding *NumberBinding) DestroyEachResultOption {
	return creaturesDiedThisWayOption{binding}
}
func (creaturesDiedThisWayOption) destroyEachResultOption() {}
func (o creaturesDiedThisWayOption) applyDestroyEachResult(e *destroyEachEffect) {
	e.creatureDeaths = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DestroyEach performs one simultaneous destruction group through the engine.
func DestroyEach(subjects PermanentSet, options ...DestroyEachResultOption) ResolvingEffect {
	return newDestroyEachEffect(subjects, false, options)
}

// DestroyEachWithoutRegeneration destroys the set without regeneration.
func DestroyEachWithoutRegeneration(subjects PermanentSet, options ...DestroyEachResultOption) ResolvingEffect {
	return newDestroyEachEffect(subjects, true, options)
}
func newDestroyEachEffect(subjects PermanentSet, cannotRegenerate bool, options []DestroyEachResultOption) ResolvingEffect {
	e := &destroyEachEffect{subjects: subjects, cannotRegenerate: cannotRegenerate}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyDestroyEachResult(e)
	}
	return e
}
func (e *destroyEachEffect) Text() string {
	if e.cannotRegenerate {
		return "destroy each permanent; they can't be regenerated"
	}
	return "destroy each permanent"
}
func (*destroyEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Destroys: true, Mass: true}
}
func (e *destroyEachEffect) Apply(ctx *EffectContext) error {
	if e.creatureDeaths != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.creatureDeaths] = 0
	}
	deaths := ctx.Game.destroyPermanents(e.subjects.resolvePermanents(ctx), e.cannotRegenerate)
	if e.creatureDeaths != nil {
		ctx.Game.resolution.numberBindings[e.creatureDeaths] = deaths.creatureDeaths
	}
	return nil
}
func (e *destroyEachEffect) inspectDefinition() definitionNode {
	role := "destroy-each"
	if e.cannotRegenerate {
		role = "destroy-each-without-regeneration"
	}
	n := definitionNode{Role: role, Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
	if e.invalidOption {
		n.problem = "supply a non-nil destruction result binding"
	} else if e.optionCount > 1 {
		n.problem = "supply the destruction result binding only once"
	}
	if e.creatureDeaths != nil {
		n.bindingWrite = e.creatureDeaths
		n.children = append(n.children, definitionEdge{"creature deaths", e.creatureDeaths})
	}
	return n
}

type returnEachToHandEffect struct{ subjects PermanentSet }

// ReturnEachToHand returns all selected permanents to their owners' hands at the same time.
func ReturnEachToHand(subjects PermanentSet) ResolvingEffect {
	return &returnEachToHandEffect{subjects: subjects}
}
func (*returnEachToHandEffect) Text() string { return "return each permanent to its owner's hand" }
func (*returnEachToHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, IsBounce: true, Mass: true}
}
func (e *returnEachToHandEffect) Apply(ctx *EffectContext) error {
	ctx.Game.BouncePermanentsToHand(e.subjects.resolvePermanents(ctx))
	return nil
}
func (e *returnEachToHandEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "return-each-to-hand", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}

// GainLifeFor gains life for the selected player through the existing life action.
func GainLifeFor(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &lifeAction{subject: subject, amount: amount, gain: true}
}

// LoseLifeFor loses life for the selected player through the existing life action.
func LoseLifeFor(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &lifeAction{subject: subject, amount: amount}
}

// AddCountersTo adds counters through the existing replacement action.
func AddCountersTo(subject PermanentRef, counter core.CounterType, amount ResolutionValue) ResolvingEffect {
	return &counterSubjectEffect{subject, counter, amount}
}

// TapSubject taps one battlefield incarnation through the normal tap action.
func TapSubject(subject PermanentRef, options ...TapResultOption) ResolvingEffect {
	e := &tapSubjectEffect{subject: subject}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyTapResult(e)
	}
	return e
}

type tapSubjectEffect struct {
	subject       PermanentRef
	result        *NumberBinding
	optionCount   int
	invalidOption bool
}

// TapResultOption selects one result for a tap instruction.
type TapResultOption interface {
	tapResultOption()
	applyTapResult(*tapSubjectEffect)
}

type tapAmountOption struct{ binding *NumberBinding }

// TapAmountInto saves one after a tap transition, or zero otherwise.
func TapAmountInto(binding *NumberBinding) TapResultOption { return tapAmountOption{binding} }
func (tapAmountOption) tapResultOption()                   {}
func (o tapAmountOption) applyTapResult(e *tapSubjectEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

func (*tapSubjectEffect) Text() string { return "tap permanent" }
func (*tapSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Taps: true}
}
func (e *tapSubjectEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	if p, ok := e.subject.resolvePermanent(ctx); ok && !p.Tapped {
		ctx.Game.TapPermanent(p)
		if e.result != nil {
			ctx.Game.resolution.numberBindings[e.result] = 1
		}
	}
	return nil
}
func (e *tapSubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.invalidOption {
		n.problem = "supply a non-nil tap result binding"
	} else if e.optionCount > 1 {
		n.problem = "supply the tap result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}

// TapCost pays by tapping the source. It uses the existing payment transaction.
func TapCost(sourceRef) Cost { return Tap() }

type lifeAction struct {
	subject PlayerRef
	amount  ResolutionValue
	gain    bool
}

func (e *lifeAction) Text() string {
	verb := "lose"
	if e.gain {
		verb = "gain"
	}
	amount := ""
	if !nilDefinition(e.amount) {
		amount = e.amount.Text()
	}
	return verb + " " + amount + " life for " + InspectDefinition(e.subject).Text
}

func (e *lifeAction) Properties() EffectProperties {
	p := EffectProperties{Outcome: OutcomeDetriment}
	if e.gain {
		p.Outcome = OutcomeBenefit
		if value, ok := e.amount.(fixedValue); ok {
			p.LifeGain = max(0, value.n)
		}
	}
	return p
}

func (e *lifeAction) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	// The recipient must not replace targets used to calculate the amount.
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{id}
	if e.gain {
		return GainLifeTarget(Fixed(amount)).Apply(&child)
	}
	return TargetPlayerLoseLife(Fixed(amount)).Apply(&child)
}

func (e *lifeAction) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}

type untapPermanentEffect struct{ subject PermanentRef }

func (e *untapPermanentEffect) Text() string { return "untap " + InspectDefinition(e.subject).Text }
func (*untapPermanentEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *untapPermanentEffect) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.UntapPermanent(p)
	}
	return nil
}
func (e *untapPermanentEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}

// RemovePermanentFromCombat removes the selected incarnation from combat.
func RemovePermanentFromCombat(subject PermanentRef) ResolvingEffect {
	return &removePermanentFromCombatEffect{subject: subject}
}

type removePermanentFromCombatEffect struct{ subject PermanentRef }

func (e *removePermanentFromCombatEffect) Text() string {
	return "remove from combat " + InspectDefinition(e.subject).Text
}
func (e *removePermanentFromCombatEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *removePermanentFromCombatEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *removePermanentFromCombatEffect) Apply(ctx *EffectContext) error {
	if permanent, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.RemoveFromCombat(permanent.ID())
	}
	return nil
}
