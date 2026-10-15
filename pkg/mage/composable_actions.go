package mage

import (
	"fmt"
	"maps"
	"math"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_actions.go ---
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

// Destroy destroys the selected battlefield incarnation through the normal action.
func Destroy(subject PermanentRef) ResolvingEffect {
	return newDestroySubjectEffect(subject, false, nil)
}

// DestroyWithoutRegeneration destroys the subject and prevents regeneration.
func DestroyWithoutRegeneration(subject PermanentRef) ResolvingEffect {
	return newDestroySubjectEffect(subject, true, nil)
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
func (e *destroySubjectEffect) Text() string {
	verb := "destroy"
	if e.cannotRegenerate {
		verb = "destroy without regeneration"
	}
	if e.subject != nil {
		return verb + " " + InspectDefinition(e.subject).Text
	}
	return verb + " permanent"
}
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

// ExileResultOption saves the outcome of one battlefield exile instruction.
type ExileResultOption interface {
	exileResultOption()
	applyExileResult(*exileSubjectEffect)
}

type exiledThisWayOption struct{ binding *NumberBinding }

// ExiledThisWayInto saves 1 when the subject moves from the battlefield.
func ExiledThisWayInto(binding *NumberBinding) ExileResultOption {
	return exiledThisWayOption{binding: binding}
}
func (exiledThisWayOption) exileResultOption() {}
func (o exiledThisWayOption) applyExileResult(e *exileSubjectEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type exileSubjectEffect struct {
	subject       PermanentRef
	result        *NumberBinding
	optionCount   int
	invalidOption bool
	withResult    bool
}

// Exile exiles the selected battlefield incarnation.
func Exile(subject PermanentRef) ResolvingEffect {
	return &exileSubjectEffect{subject: subject}
}

// ExileWithResult exiles one permanent and saves the completed move.
func ExileWithResult(subject PermanentRef, options ...ExileResultOption) ResolvingEffect {
	e := &exileSubjectEffect{subject: subject, withResult: true}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyExileResult(e)
	}
	return e
}

func (e *exileSubjectEffect) Text() string {
	if e.withResult {
		return "exile permanent with result"
	}
	if e.subject != nil {
		return "exile " + InspectDefinition(e.subject).Text
	}
	return "exile permanent"
}

func (*exileSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, TargetPurposeOverride: AITargetExile}
}

func (e *exileSubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.withResult {
		if e.invalidOption || e.result == nil {
			n.problem = "supply a non-nil exile result binding"
		} else if e.optionCount != 1 {
			n.problem = "supply the exile result binding only once"
		}
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}

func (e *exileSubjectEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	if e.subject != nil {
		if p, ok := e.subject.resolvePermanent(ctx); ok {
			ctx.Game.ExilePermanent(p)
			if e.result != nil {
				ctx.Game.resolution.numberBindings[e.result] = 1
			}
		}
	}
	return nil
}

type exileEachEffect struct{ subjects PermanentSet }

// ExileEach moves the selected permanents to exile in one instruction.
func ExileEach(subjects PermanentSet) ResolvingEffect {
	return &exileEachEffect{subjects: subjects}
}
func (*exileEachEffect) Text() string { return "exile each permanent" }
func (*exileEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *exileEachEffect) Apply(ctx *EffectContext) error {
	ctx.Game.ExilePermanents(e.subjects.resolvePermanents(ctx))
	return nil
}
func (e *exileEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "exile-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}

type returnToHandSubjectEffect struct {
	subject PermanentRef
}

// ReturnToHand returns the selected permanent to its owner's hand.
func ReturnToHand(subject PermanentRef) ResolvingEffect {
	return &returnToHandSubjectEffect{subject: subject}
}

func (e *returnToHandSubjectEffect) Text() string {
	if e.subject != nil {
		return "return to owner's hand " + InspectDefinition(e.subject).Text
	}
	return "return to owner's hand permanent"
}

func (*returnToHandSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, IsBounce: true}
}

func (e *returnToHandSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}

func (e *returnToHandSubjectEffect) Apply(ctx *EffectContext) error {
	if e.subject != nil {
		if p, ok := e.subject.resolvePermanent(ctx); ok {
			ctx.Game.BouncePermanentToHand(p)
		}
	}
	return nil
}

// UntapPermanent untaps the selected battlefield incarnation.
func UntapPermanent(subject PermanentRef) ResolvingEffect {
	return &untapPermanentEffect{subject: subject}
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
	cardsResult   *CardsBinding
	optionCount   int
	cardCount     int
	setCount      int
	invalidOption bool
}

// DrawResultOption selects one result for a draw instruction.
type DrawResultOption interface {
	drawResultOption()
	applyDrawResult(*drawSubjectEffect)
}

type drawAmountOption struct{ binding *NumberBinding }

type drawCardOption struct{ binding *CardBinding }
type drawCardsOption struct{ binding *CardsBinding }

// DrawCardsInto saves the cards from one completed draw instruction.
func DrawCardsInto(binding *CardsBinding) DrawResultOption { return drawCardsOption{binding} }
func (drawCardsOption) drawResultOption()                  {}
func (o drawCardsOption) applyDrawResult(e *drawSubjectEffect) {
	e.cardsResult = o.binding
	e.setCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

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
	} else if e.optionCount > 1 || e.cardCount > 1 || e.setCount > 1 {
		n.problem = "supply the draw result binding only once"
	} else if (e.result != nil && (e.cardResult != nil || e.cardsResult != nil)) || (e.cardResult != nil && e.cardsResult != nil) {
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
	if e.cardsResult != nil {
		n.bindingWrite = e.cardsResult
		n.children = append(n.children, definitionEdge{"cards result", e.cardsResult})
	}
	return n
}
func (e *drawSubjectEffect) Apply(ctx *EffectContext) error {
	if e.cardsResult != nil {
		if ctx.Game.resolution.cardsBindings == nil {
			ctx.Game.resolution.cardsBindings = make(map[*CardsBinding][]capturedCard)
		}
		delete(ctx.Game.resolution.cardsBindings, e.cardsResult)
	}
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
					ctx.Game.resolution.cardBindings[e.cardResult] = capturedCard{id: card.ID(), visit: visitor.handVisit(card.ID()), zone: ZoneHand}
				}
			}
			if e.cardsResult != nil {
				if visitor, ok := p.(interface{ handVisit(uuid.UUID) uuid.UUID }); ok {
					ctx.Game.resolution.cardsBindings[e.cardsResult] = append(ctx.Game.resolution.cardsBindings[e.cardsResult], capturedCard{id: card.ID(), visit: visitor.handVisit(card.ID()), zone: ZoneHand})
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
func Discard(subject PlayerRef, amount ResolutionValue, options ...DiscardCardResultOption) ResolvingEffect {
	e := &discardSubjectEffect{subject: subject, amount: amount}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyDiscardCardResult(e)
	}
	return e
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
	deathVisits      *PermanentSetBinding
	graveyardMoves   *NumberBinding
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

type creatureDeathVisitsOption struct{ binding *PermanentSetBinding }

// CreatureDeathsInto saves exact creature visits moved to a graveyard by this instruction.
func CreatureDeathsInto(binding *PermanentSetBinding) DestroyEachResultOption {
	return creatureDeathVisitsOption{binding}
}
func (creatureDeathVisitsOption) destroyEachResultOption() {}
func (o creatureDeathVisitsOption) applyDestroyEachResult(e *destroyEachEffect) {
	e.deathVisits = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type permanentsPutIntoGraveyardThisWayOption struct{ binding *NumberBinding }

// PermanentsPutIntoGraveyardThisWayInto saves the number of selected permanents moved to a graveyard.
func PermanentsPutIntoGraveyardThisWayInto(binding *NumberBinding) DestroyEachResultOption {
	return permanentsPutIntoGraveyardThisWayOption{binding}
}
func (permanentsPutIntoGraveyardThisWayOption) destroyEachResultOption() {}
func (o permanentsPutIntoGraveyardThisWayOption) applyDestroyEachResult(e *destroyEachEffect) {
	e.graveyardMoves = o.binding
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
	if e.creatureDeaths != nil || e.graveyardMoves != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		if e.creatureDeaths != nil {
			ctx.Game.resolution.numberBindings[e.creatureDeaths] = 0
		}
		if e.graveyardMoves != nil {
			ctx.Game.resolution.numberBindings[e.graveyardMoves] = 0
		}
	}
	if e.deathVisits != nil {
		if ctx.Game.resolution.createdTokenBindings == nil {
			ctx.Game.resolution.createdTokenBindings = make(map[*PermanentSetBinding][]capturedPermanent)
		}
		ctx.Game.resolution.createdTokenBindings[e.deathVisits] = nil
	}
	deaths := ctx.Game.destroyPermanents(e.subjects.resolvePermanents(ctx), e.cannotRegenerate)
	if e.deathVisits != nil {
		ctx.Game.resolution.createdTokenBindings[e.deathVisits] = deaths.visits
	}
	if e.creatureDeaths != nil {
		ctx.Game.resolution.numberBindings[e.creatureDeaths] = deaths.creatureDeaths
	}
	if e.graveyardMoves != nil {
		ctx.Game.resolution.numberBindings[e.graveyardMoves] = deaths.destroyed
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
	if e.deathVisits != nil {
		n.bindingWrite = e.deathVisits
		n.children = append(n.children, definitionEdge{"creature visits", e.deathVisits})
	}
	if e.creatureDeaths != nil {
		n.bindingWrite = e.creatureDeaths
		n.children = append(n.children, definitionEdge{"creature deaths", e.creatureDeaths})
	}
	if e.graveyardMoves != nil {
		n.bindingWrite = e.graveyardMoves
		n.children = append(n.children, definitionEdge{"graveyard moves", e.graveyardMoves})
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

type tapEachEffect struct{ subjects PermanentSet }

// TapEach taps the selected permanents together.
func TapEach(subjects PermanentSet) ResolvingEffect {
	return &tapEachEffect{subjects: subjects}
}
func (*tapEachEffect) Text() string { return "tap each permanent" }
func (*tapEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Taps: true, Mass: true}
}
func (e *tapEachEffect) Apply(ctx *EffectContext) error {
	selected := e.subjects.resolvePermanents(ctx)
	untapped := make([]*Permanent, 0, len(selected))
	seen := make(map[uuid.UUID]bool, len(selected))
	for _, p := range selected {
		if p == nil || seen[p.ID()] {
			continue
		}
		live := ctx.Game.FindPermanent(p.ID())
		if live == nil || live.incarnationID != p.incarnationID || live.Tapped {
			continue
		}
		seen[p.ID()] = true
		untapped = append(untapped, live)
	}
	ctx.Game.tapPermanentsTogether(untapped)
	return nil
}
func (e *tapEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "tap-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}

type untapEachEffect struct{ subjects PermanentSet }

// UntapEach untaps the selected permanents together.
func UntapEach(subjects PermanentSet) ResolvingEffect {
	return &untapEachEffect{subjects: subjects}
}
func (*untapEachEffect) Text() string { return "untap each permanent" }
func (*untapEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *untapEachEffect) Apply(ctx *EffectContext) error {
	selected := e.subjects.resolvePermanents(ctx)
	tapped := make([]*Permanent, 0, len(selected))
	seen := make(map[uuid.UUID]bool, len(selected))
	for _, p := range selected {
		if p == nil || seen[p.ID()] {
			continue
		}
		live := ctx.Game.FindPermanent(p.ID())
		if live == nil || live.incarnationID != p.incarnationID || !live.Tapped {
			continue
		}
		seen[p.ID()] = true
		tapped = append(tapped, live)
	}
	ctx.Game.untapPermanentsTogether(tapped)
	return nil
}
func (e *untapEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "untap-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}

// GainLifeFor gains life for the selected player through the existing life action.
func GainLifeFor(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &lifeAction{subject: subject, amount: amount, gain: true}
}

// GainLifeForWithResult saves the life gained by one player after replacement.
func GainLifeForWithResult(subject PlayerRef, amount ResolutionValue, options ...LifeGainResultOption) ResolvingEffect {
	e := &lifeAction{subject: subject, amount: amount, gain: true, gainResult: true}
	if len(options) == 0 {
		e.lifeGainResultState.invalid = true
	}
	for _, option := range options {
		if nilDefinition(option) {
			e.lifeGainResultState.invalid = true
			continue
		}
		option.applyLifeGainResult(&e.lifeGainResultState)
	}
	return e
}

// LoseLifeFor loses life for the selected player through the existing life action.
func LoseLifeFor(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &lifeAction{subject: subject, amount: amount}
}

// LoseLifeForWithResult saves the life lost by one player in this instruction.
func LoseLifeForWithResult(subject PlayerRef, amount ResolutionValue, options ...LifeLossResultOption) ResolvingEffect {
	e := &lifeAction{subject: subject, amount: amount}
	if len(options) == 0 {
		e.lifeLossResultState.invalid = true
	}
	for _, option := range options {
		if nilDefinition(option) {
			e.lifeLossResultState.invalid = true
			continue
		}
		option.applyLifeLossResult(&e.lifeLossResultState)
	}
	return e
}

// AddCountersTo adds counters through the existing replacement action.
func AddCountersTo(subject PermanentRef, counter CounterType, amount ResolutionValue) ResolvingEffect {
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
	subject    PlayerRef
	amount     ResolutionValue
	gain       bool
	gainResult bool
	lifeGainResultState
	lifeLossResultState
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
	result := e.lifeLossResultState.result
	if e.gain {
		result = e.lifeGainResultState.result
	}
	if result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[result] = 0
	}
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
		if e.gainResult {
			if player := ctx.Game.GetPlayer(id); player != nil {
				ctx.Game.resolution.numberBindings[result] = ctx.Game.playersGainLife([]Player{player}, amount)
			}
			return nil
		}
		return GainLifeTarget(Fixed(amount)).Apply(&child)
	}
	if e.lifeLossResultState.result == nil {
		return TargetPlayerLoseLife(Fixed(amount)).Apply(&child)
	}
	player := ctx.Game.GetPlayer(id)
	if player != nil {
		ctx.Game.resolution.numberBindings[e.lifeLossResultState.result] = ctx.Game.playersLoseLife([]Player{player}, amount)
	}
	return nil
}

func (e *lifeAction) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
	if e.gainResult {
		if e.lifeGainResultState.invalid {
			n.problem = "supply a non-nil life gain result binding option"
		} else if e.lifeGainResultState.count > 1 {
			n.problem = "supply the life gain result binding only once"
		}
		if e.lifeGainResultState.result != nil {
			n.bindingWrite = e.lifeGainResultState.result
			n.children = append(n.children, definitionEdge{"result", e.lifeGainResultState.result})
		}
		return n
	}
	if e.lifeLossResultState.invalid {
		n.problem = "supply a non-nil life loss result binding"
	} else if e.lifeLossResultState.count > 1 {
		n.problem = "supply the life loss result binding only once"
	}
	if e.lifeLossResultState.result != nil {
		n.bindingWrite = e.lifeLossResultState.result
		n.children = append(n.children, definitionEdge{"result", e.lifeLossResultState.result})
	}
	return n
}

type untapPermanentEffect struct {
	subject        PermanentRef
	result         *NumberBinding
	optionCount    int
	invalidOption  bool
	requiresResult bool
}

// UntapResultOption selects an output for one untap instruction.
type UntapResultOption interface {
	untapResultOption()
	applyUntapResult(*untapPermanentEffect)
}

type untappedThisWayOption struct{ binding *NumberBinding }

// UntappedThisWayInto saves 1 only when the permanent becomes untapped.
func UntappedThisWayInto(binding *NumberBinding) UntapResultOption {
	return untappedThisWayOption{binding}
}
func (untappedThisWayOption) untapResultOption() {}
func (o untappedThisWayOption) applyUntapResult(e *untapPermanentEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// UntapPermanentWithResult untaps one permanent and saves the outcome.
func UntapPermanentWithResult(subject PermanentRef, options ...UntapResultOption) ResolvingEffect {
	e := &untapPermanentEffect{subject: subject, requiresResult: true}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyUntapResult(e)
	}
	return e
}

func (e *untapPermanentEffect) Text() string { return "untap " + InspectDefinition(e.subject).Text }
func (*untapPermanentEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *untapPermanentEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		if ctx.Game.UntapPermanent(p) && e.result != nil {
			ctx.Game.resolution.numberBindings[e.result] = 1
		}
	}
	return nil
}
func (e *untapPermanentEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.invalidOption || (e.requiresResult && e.result == nil) {
		n.problem = "supply a non-nil untap result binding"
	} else if e.optionCount > 1 {
		n.problem = "supply the untap result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}

// RemovePermanentFromCombat removes the selected incarnation from combat.
func RemovePermanentFromCombat(subject PermanentRef) ResolvingEffect {
	return &removePermanentFromCombatEffect{subject: subject}
}

// RemovePermanentFromCombatAndUnblock makes attackers blocked only by this object unblocked.
func RemovePermanentFromCombatAndUnblock(subject PermanentRef) ResolvingEffect {
	return &removePermanentFromCombatEffect{subject: subject, unblock: true}
}

type removePermanentFromCombatEffect struct {
	subject PermanentRef
	unblock bool
}

func (e *removePermanentFromCombatEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}

func (e *removePermanentFromCombatEffect) Text() string {
	if e.unblock {
		return "remove from combat and unblock " + InspectDefinition(e.subject).Text
	}
	return "remove from combat " + InspectDefinition(e.subject).Text
}
func (e *removePermanentFromCombatEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *removePermanentFromCombatEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *removePermanentFromCombatEffect) Apply(ctx *EffectContext) error {
	if permanent, ok := e.subject.resolvePermanent(ctx); ok {
		if e.unblock {
			ctx.Game.combat.removeIncarnationAndUnblock(permanent.ID(), permanent.incarnationID)
		} else {
			ctx.Game.RemoveFromCombat(permanent.ID())
		}
	}
	return nil
}

// --- from composable_counter.go ---
type counterSubjectEffect struct {
	subject PermanentRef
	counter CounterType
	amount  ResolutionValue
}

type counterEachEffect struct {
	subjects PermanentSet
	counter  CounterType
	amount   ResolutionValue
}

// AddCountersEach places counters on the selected set in one instruction.
func AddCountersEach(subjects PermanentSet, counter CounterType, amount ResolutionValue) ResolvingEffect {
	return &counterEachEffect{subjects: subjects, counter: counter, amount: amount}
}

func (*counterEachEffect) Text() string { return "add counters to each permanent" }
func (e *counterEachEffect) Properties() EffectProperties {
	properties := EffectProperties{Mass: true}
	if amount, ok := e.amount.(fixedValue); ok {
		properties.Outcome = AddCounters(e.counter, amount).Properties().Outcome
	}
	return properties
}
func (e *counterEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "counter-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
	if e.counter < 0 || e.counter >= NumCounters {
		n.problem = "supply a valid counter type"
	}
	return n
}
func (e *counterEachEffect) Apply(ctx *EffectContext) error {
	selected := e.subjects.resolvePermanents(ctx)
	if len(selected) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	source, _ := (sourceRef{}).informationIdentity(ctx)
	ctx.Game.addCountersTogetherFrom(selected, e.counter, amount, ctx.SourceID, source)
	return nil
}

func (e *counterSubjectEffect) Text() string {
	return "add counters to " + InspectDefinition(e.subject).Text
}
func (e *counterSubjectEffect) Properties() EffectProperties {
	if amount, ok := e.amount.(fixedValue); ok {
		return AddCounters(e.counter, amount).Properties()
	}
	return EffectProperties{}
}
func (e *counterSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *counterSubjectEffect) Apply(ctx *EffectContext) error {
	permanent, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{permanent.ID()}
	return AddCounters(e.counter, Fixed(amount)).Apply(&child)
}

// --- from composable_counter_removal.go ---
type removeCountersSubjectEffect struct {
	subject        PermanentRef
	counter        CounterType
	amount         ResolutionValue
	result         *NumberBinding
	requiresResult bool
}

// RemoveCountersFrom removes up to the specified number of counters from a subject.
func RemoveCountersFrom(subject PermanentRef, counter CounterType, amount ResolutionValue) ResolvingEffect {
	return &removeCountersSubjectEffect{subject: subject, counter: counter, amount: amount}
}

// RemoveCountersFromWithResult saves only counters removed by this instruction.
func RemoveCountersFromWithResult(subject PermanentRef, counter CounterType, amount ResolutionValue, result *NumberBinding) ResolvingEffect {
	return &removeCountersSubjectEffect{subject: subject, counter: counter, amount: amount, result: result, requiresResult: true}
}
func (e *removeCountersSubjectEffect) Text() string {
	return "remove " + e.counter.String() + " counters from " + InspectDefinition(e.subject).Text
}
func (*removeCountersSubjectEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *removeCountersSubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
	if e.counter < 0 || e.counter >= NumCounters {
		n.problem = "supply a valid counter type"
	} else if e.requiresResult && e.result == nil {
		n.problem = "supply a non-nil counter removal result binding"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *removeCountersSubjectEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	amount := min(e.amount.resolveValue(ctx), p.Counters[e.counter])
	if amount <= 0 {
		return nil
	}
	before := p.Counters[e.counter]
	child := *ctx
	child.Targets = []uuid.UUID{p.ID()}
	if err := RemoveCounters(e.counter, amount).Targeting(ToTarget()).Apply(&child); err != nil {
		return err
	}
	if e.result != nil {
		current := ctx.Game.FindPermanent(p.ID())
		if current != nil && current.incarnationID == p.incarnationID {
			ctx.Game.resolution.numberBindings[e.result] = before - current.Counters[e.counter]
		}
	}
	return nil
}

// --- from composable_counter_reset.go ---
type sourceCounterCondition struct {
	counter CounterType
	minimum int
}

// SourceHasCounters checks for at least minimum counters on the source.
func SourceHasCounters(counter CounterType, minimum int) StateCondition {
	return sourceCounterCondition{counter, minimum}
}
func (sourceCounterCondition) stateCondition() {}
func (c sourceCounterCondition) checkState(g GameReader, ctx StateContext) bool {
	p := g.FindPermanent(ctx.SourceID)
	return p != nil && !p.PhasedOut && p.Counters[c.counter] >= c.minimum
}
func (c sourceCounterCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: fmt.Sprintf("source has at least %d %s counters", c.minimum, c.counter), battlefieldSource: true}
	if c.minimum < 0 {
		n.problem = "supply a nonnegative counter threshold"
	}
	return n
}

type removeAllCountersEffect struct {
	subject PermanentRef
	counter CounterType
}

// RemoveAllCounters removes all counters of one kind from the subject.
func RemoveAllCounters(subject PermanentRef, counter CounterType) ResolvingEffect {
	return &removeAllCountersEffect{subject, counter}
}
func (e *removeAllCountersEffect) Text() string {
	return "remove all " + e.counter.String() + " counters"
}
func (*removeAllCountersEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *removeAllCountersEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *removeAllCountersEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{p.ID()}
	return RemoveCounters(e.counter, p.Counters[e.counter]).Targeting(ToTarget()).Apply(&child)
}

// --- from composable_damage_each.go ---
// DamageRecipientSet selects players and permanents for one damage instruction.
type DamageRecipientSet interface {
	damageRecipientSet()
	resolveDamageRecipients(*EffectContext, int, DamageSource) []DamageAssignment
}

// DamageTargetsBinding declares a bounded group of damage recipients.
type DamageTargetsBinding struct{ *targetDeclaration }

// DeclareDamageTargets declares player, creature, or planeswalker targets.
func DeclareDamageTargets(name string, minimum, maximum int) *DamageTargetsBinding {
	spec := &DamageAnyTarget{BaseTarget: BaseTarget{min: minimum, max: maximum}}
	return &DamageTargetsBinding{&targetDeclaration{name: name, spec: spec}}
}

// The slice length is the only upper bound for an unrestricted target group.
func DeclareAnyNumberDamageTargets(name string) *DamageTargetsBinding {
	return DeclareDamageTargets(name, 0, math.MaxInt)
}

func DeclareXDamageTargets(name string) *DamageTargetsBinding {
	return declareXBoundDamageTargets(name, false)
}

func DeclareUpToXDamageTargets(name string) *DamageTargetsBinding {
	return declareXBoundDamageTargets(name, true)
}

func declareXBoundDamageTargets(name string, upTo bool) *DamageTargetsBinding {
	spec := &countBoundTarget{inner: &DamageAnyTarget{BaseTarget: BaseTarget{min: 1, max: 1}}, usesX: !upTo, upToX: upTo}
	return &DamageTargetsBinding{&targetDeclaration{name: name, spec: spec}}
}

func (*DamageTargetsBinding) damageRecipientSet() {}
func (b *DamageTargetsBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "damage-recipient-set")
}
func (b *DamageTargetsBinding) resolveDamageRecipients(ctx *EffectContext, amount int, source DamageSource) []DamageAssignment {
	values := ctx.targetBindings[b.targetDeclaration]
	assignments := make([]DamageAssignment, 0, len(values))
	for _, value := range values {
		if player := ctx.Game.GetPlayer(value.id); player != nil {
			assignments = append(assignments, PlayerDamageAssignment(player, amount, source))
			continue
		}
		permanent := ctx.Game.FindPermanent(value.id)
		if permanent != nil && !permanent.PhasedOut && value.incarnation != uuid.Nil && permanent.incarnationID == value.incarnation {
			assignments = append(assignments, PermanentDamageAssignment(permanent, amount, source))
		}
	}
	return assignments
}

type mixedDamageRecipients struct {
	players    PlayerSet
	permanents PermanentSet
}

// DamageRecipients combines player and permanent sets for one simultaneous action.
func DamageRecipients(players PlayerSet, permanents PermanentSet) DamageRecipientSet {
	return mixedDamageRecipients{players: players, permanents: permanents}
}

func (mixedDamageRecipients) damageRecipientSet() {}
func (s mixedDamageRecipients) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-recipient-set", children: []definitionEdge{{"players", s.players}, {"permanents", s.permanents}}}
	if nilDefinition(s.players) {
		n.problem = "supply a player set"
	} else if nilDefinition(s.permanents) {
		n.problem = "supply a permanent set"
	}
	return n
}
func (s mixedDamageRecipients) resolveDamageRecipients(ctx *EffectContext, amount int, source DamageSource) []DamageAssignment {
	players := s.players.resolvePlayers(ctx)
	permanents := s.permanents.resolvePermanents(ctx)
	assignments := make([]DamageAssignment, 0, len(players)+len(permanents))
	for _, id := range players {
		if player := ctx.Game.GetPlayer(id); player != nil {
			assignments = append(assignments, PlayerDamageAssignment(player, amount, source))
		}
	}
	for _, permanent := range permanents {
		if permanent != nil && !permanent.PhasedOut {
			assignments = append(assignments, PermanentDamageAssignment(permanent, amount, source))
		}
	}
	return assignments
}

type damageEachEffect struct {
	source        DamageSourceRef
	recipients    DamageRecipientSet
	amount        ResolutionValue
	evenDivision  bool
	total         *NumberBinding
	results       *DamageResultsBinding
	optionCount   int
	resultsCount  int
	invalidOption bool
}

// DamageEachResultOption selects an output for a simultaneous damage instruction.
type DamageEachResultOption interface {
	damageEachResultOption()
	applyDamageEachResult(*damageEachEffect)
}

type damageTotalOption struct{ binding *NumberBinding }

// DamageTotalInto saves the total damage dealt by one simultaneous instruction.
func DamageTotalInto(binding *NumberBinding) DamageEachResultOption {
	return damageTotalOption{binding: binding}
}

func (damageTotalOption) damageEachResultOption() {}
func (o damageTotalOption) applyDamageEachResult(e *damageEachEffect) {
	e.total = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DealDamageEach deals one amount from one source to each selected recipient.
func DealDamageEach(source DamageSourceRef, recipients DamageRecipientSet, amount ResolutionValue, options ...DamageEachResultOption) ResolvingEffect {
	return newDamageEachEffect(source, recipients, amount, options)
}

// DealDamageEvenly divides the total at resolution, not at announcement.
// Only surviving recipients in this target group receive a share.
func DealDamageEvenly(source DamageSourceRef, recipients *DamageTargetsBinding, total ResolutionValue, options ...DamageEachResultOption) ResolvingEffect {
	return DealDamageEvenlyToRecipients(source, recipients, total, options...)
}

// DealDamageEvenlyToRecipients selects recipients once before damage replacements.
// Each selected recipient gets an equal share, rounded down.
func DealDamageEvenlyToRecipients(source DamageSourceRef, recipients DamageRecipientSet, total ResolutionValue, options ...DamageEachResultOption) ResolvingEffect {
	e := newDamageEachEffect(source, recipients, total, options)
	e.evenDivision = true
	return e
}

func newDamageEachEffect(source DamageSourceRef, recipients DamageRecipientSet, amount ResolutionValue, options []DamageEachResultOption) *damageEachEffect {
	e := &damageEachEffect{source: source, recipients: recipients, amount: amount}
	for _, option := range options {
		if option == nil {
			e.invalidOption = true
			continue
		}
		option.applyDamageEachResult(e)
	}
	return e
}

func (e *damageEachEffect) Text() string {
	if e.evenDivision {
		return "deal damage divided evenly, rounded down"
	}
	return "deal damage to each recipient"
}
func (*damageEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *damageEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{
		{"source", damageSourceDefinition{e.source}}, {"recipients", e.recipients}, {"amount", e.amount},
	}}
	if e.invalidOption {
		n.problem = "supply a damage result binding"
	} else if e.optionCount > 1 || e.resultsCount > 1 {
		n.problem = "supply each damage result binding only once"
	}
	if e.total != nil {
		n.bindingWrites = append(n.bindingWrites, e.total)
		n.children = append(n.children, definitionEdge{"total", e.total})
	}
	if e.results != nil {
		n.bindingWrites = append(n.bindingWrites, e.results)
		n.children = append(n.children, definitionEdge{"results", e.results})
	}
	return n
}
func (e *damageEachEffect) Apply(ctx *EffectContext) error {
	if e.total != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.total] = 0
	}
	if e.results != nil {
		if ctx.Game.resolution.damageResultsBindings == nil {
			ctx.Game.resolution.damageResultsBindings = make(map[*DamageResultsBinding][]damageOutcome)
		}
		delete(ctx.Game.resolution.damageResultsBindings, e.results)
	}
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	assignments := e.recipients.resolveDamageRecipients(ctx, amount, source)
	if e.evenDivision {
		if len(assignments) == 0 {
			return nil
		}
		share := amount / len(assignments)
		if share == 0 {
			return nil
		}
		for i := range assignments {
			assignments[i].request.amount = share
		}
	}
	outcomes := ctx.Game.dealSimultaneousDamageOutcomes(assignments...)
	total := 0
	for _, outcome := range outcomes {
		total += outcome.amount
	}
	if e.total != nil {
		ctx.Game.resolution.numberBindings[e.total] = total
	}
	if e.results != nil {
		ctx.Game.resolution.damageResultsBindings[e.results] = cloneDamageOutcomes(outcomes)
	}
	return nil
}

// --- from composable_damage_from.go ---
// DamageSourceRef identifies the object that deals damage.
type DamageSourceRef interface {
	damageSourceRef()
	resolveDamageSource(*EffectContext) DamageSource
}

func (sourceRef) damageSourceRef() {}
func (sourceRef) resolveDamageSource(ctx *EffectContext) DamageSource {
	if ctx.sourceCaptured && ctx.sourceIncarnation != uuid.Nil {
		if p := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); p != nil && p.incarnationID == ctx.sourceIncarnation {
			return damageSourceFromPermanent(p)
		}
		if lki := ctx.Game.zones.LKIByIncarnation(ctx.sourceIncarnation); lki != nil && lki.ID == ctx.SourceID {
			return damageSourceFromLKI(lki)
		}
		return DamageSource{}
	}
	return ctx.Game.CaptureDamageSource(ctx.SourceID)
}

type permanentDamageSource struct{ subject PermanentRef }

// DamageSourceOf uses one referenced permanent as a damage source.
func DamageSourceOf(subject PermanentRef) DamageSourceRef {
	return permanentDamageSource{subject: subject}
}

func (permanentDamageSource) damageSourceRef() {}

func (r permanentDamageSource) resolveDamageSource(ctx *EffectContext) DamageSource {
	if nilDefinition(r.subject) {
		return DamageSource{}
	}
	if _, ok := r.subject.(sourceRef); ok {
		return This.resolveDamageSource(ctx)
	}
	var value capturedPermanent
	switch subject := r.subject.(type) {
	case *PermanentTargetBinding:
		value, _ = firstTargetValue(ctx.targetBindings[subject.targetDeclaration])
	case savedPermanentRef:
		value = ctx.Game.resolution.permanentBindings[subject.binding]
	case playerChosenPermanentRef:
		value, _ = subject.informationIdentity(ctx)
	case damageResultPermanent:
		value = ctx.Game.resolution.damageResultBindings[subject.binding].permanent
	default:
		if p, ok := r.subject.resolvePermanent(ctx); ok {
			return damageSourceFromPermanent(p)
		}
		return DamageSource{}
	}
	if value.id == uuid.Nil || value.incarnation == uuid.Nil {
		return DamageSource{}
	}
	if p := ctx.Game.FindPermanent(value.id); p != nil && !p.PhasedOut && p.incarnationID == value.incarnation {
		return damageSourceFromPermanent(p)
	}
	if lki := ctx.Game.zones.LKIByIncarnation(value.incarnation); lki != nil && lki.ID == value.id {
		return damageSourceFromLKI(lki)
	}
	return DamageSource{}
}

func (r permanentDamageSource) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-source-ref", Text: "permanent", children: []definitionEdge{{"permanent", r.subject}}}
	if nilDefinition(r.subject) {
		n.problem = "supply a permanent damage source"
	}
	return n
}

type damageFromEffect struct {
	prevention           ResolutionValue
	withPrevention       bool
	source               DamageSourceRef
	recipient            DamageRecipientRef
	amount               ResolutionValue
	result               *NumberBinding
	playerResult         *PlayerBinding
	permanentResult      *PermanentBinding
	sourceResult         *DamageSourceBinding
	outcomes             *DamageResultsBinding
	outcomesOptionCount  int
	amountOptionCount    int
	playerOptionCount    int
	permanentOptionCount int
	sourceOptionCount    int
	invalidOption        bool
}

// DamageResultOption selects one output for a damage instruction.
type DamageResultOption interface {
	damageResultOption()
	applyDamageResult(*damageFromEffect)
}

type damageAmountOption struct{ binding *NumberBinding }

type damagePlayerOption struct{ binding *PlayerBinding }

type damagePermanentOption struct{ binding *PermanentBinding }
type damageSourceOption struct{ binding *DamageSourceBinding }

// DamageSourceInto saves the source when this instruction deals damage.
func DamageSourceInto(binding *DamageSourceBinding) DamageResultOption {
	return damageSourceOption{binding: binding}
}

func (damageSourceOption) damageResultOption() {}
func (o damageSourceOption) applyDamageResult(e *damageFromEffect) {
	e.sourceResult = o.binding
	e.sourceOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DamagePermanentInto saves the permanent that receives damage after replacements.
func DamagePermanentInto(binding *PermanentBinding) DamageResultOption {
	return damagePermanentOption{binding: binding}
}

func (damagePermanentOption) damageResultOption() {}
func (o damagePermanentOption) applyDamageResult(e *damageFromEffect) {
	e.permanentResult = o.binding
	e.permanentOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DamagePlayerInto saves the player who receives damage after replacements.
func DamagePlayerInto(binding *PlayerBinding) DamageResultOption {
	return damagePlayerOption{binding: binding}
}

func (damagePlayerOption) damageResultOption() {}
func (o damagePlayerOption) applyDamageResult(e *damageFromEffect) {
	e.playerResult = o.binding
	e.playerOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DamageAmountInto saves the damage dealt after replacement and prevention.
func DamageAmountInto(binding *NumberBinding) DamageResultOption {
	return damageAmountOption{binding: binding}
}

func (damageAmountOption) damageResultOption() {}
func (o damageAmountOption) applyDamageResult(e *damageFromEffect) {
	e.result = o.binding
	e.amountOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type damageSourceDefinition struct{ source DamageSourceRef }

func (d damageSourceDefinition) inspectDefinition() definitionNode {
	switch source := d.source.(type) {
	case savedDamageSourceRef:
		return source.inspectDefinition()
	case damageResultSource:
		return source.inspectDefinition()
	case permanentDamageSource:
		return source.inspectDefinition()
	case spellDamageSource:
		return source.inspectDefinition()
	}
	n := definitionNode{Role: "damage-source-ref", Text: "this"}
	if nilDefinition(d.source) {
		n.problem = "supply a damage source"
	}
	return n
}

// DealDamageFrom deals damage from the source to one player or permanent.
func DealDamageFrom(source DamageSourceRef, recipient DamageRecipientRef, amount ResolutionValue, options ...DamageResultOption) ResolvingEffect {
	e := &damageFromEffect{source: source, recipient: recipient, amount: amount}
	for _, option := range options {
		if option == nil {
			e.invalidOption = true
			continue
		}
		option.applyDamageResult(e)
	}
	return e
}

func (*damageFromEffect) Text() string { return "deal damage" }
func (*damageFromEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *damageFromEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"recipient", e.recipient}, {"amount", e.amount}}}
	if e.withPrevention {
		n.children = append(n.children, definitionEdge{"prevention", e.prevention})
	}
	if e.invalidOption {
		n.problem = "supply a damage result binding"
	} else if e.amountOptionCount > 1 || e.playerOptionCount > 1 || e.permanentOptionCount > 1 || e.sourceOptionCount > 1 || e.outcomesOptionCount > 1 {
		n.problem = "supply each damage result binding only once"
	}
	if e.result != nil {
		n.bindingWrites = append(n.bindingWrites, e.result)
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	if e.playerResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.playerResult)
		n.children = append(n.children, definitionEdge{"player result", e.playerResult})
	}
	if e.permanentResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.permanentResult)
		n.children = append(n.children, definitionEdge{"permanent result", e.permanentResult})
	}
	if e.sourceResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.sourceResult)
		n.children = append(n.children, definitionEdge{"source result", e.sourceResult})
	}
	if e.outcomes != nil {
		n.bindingWrites = append(n.bindingWrites, e.outcomes)
		n.children = append(n.children, definitionEdge{"outcomes", e.outcomes})
	}
	return n
}
func (e *damageFromEffect) Apply(ctx *EffectContext) error {
	outcome := damageOutcome{}
	if e.outcomes != nil {
		if ctx.Game.resolution.damageResultsBindings == nil {
			ctx.Game.resolution.damageResultsBindings = make(map[*DamageResultsBinding][]damageOutcome)
		}
		delete(ctx.Game.resolution.damageResultsBindings, e.outcomes)
	}
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	if e.playerResult != nil {
		delete(ctx.Game.resolution.playerBindings, e.playerResult)
	}
	if e.permanentResult != nil {
		delete(ctx.Game.resolution.permanentBindings, e.permanentResult)
	}
	if e.sourceResult != nil {
		delete(ctx.Game.resolution.damageSourceBindings, e.sourceResult)
	}
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() {
		return nil
	}
	id, ok := e.recipient.resolveRecipient(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	var local []ReplacementEffect
	if e.withPrevention {
		local = append(local, &instructionDamagePrevention{replacementBase: replacementBase{sourceID: source.ID()}, amount: max(0, e.prevention.resolveValue(ctx))})
	}
	if player := ctx.Game.GetPlayer(id); player != nil {
		outcome = ctx.Game.damage.dealDamageToPlayerFromSourceOutcome(ctx.Game, player, amount, source, local...)
	} else if permanent := ctx.Game.FindPermanent(id); permanent != nil {
		outcome = ctx.Game.damage.dealDamageToPermanentFromSourceOutcome(ctx.Game, permanent, amount, source, local...)
	}
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = outcome.amount
	}
	if e.playerResult != nil && outcome.playerID != uuid.Nil {
		if ctx.Game.resolution.playerBindings == nil {
			ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
		}
		ctx.Game.resolution.playerBindings[e.playerResult] = outcome.playerID
	}
	if e.permanentResult != nil && outcome.permanent.id != uuid.Nil {
		if ctx.Game.resolution.permanentBindings == nil {
			ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
		}
		ctx.Game.resolution.permanentBindings[e.permanentResult] = outcome.permanent
	}
	if e.sourceResult != nil && outcome.amount > 0 && outcome.source.Valid() {
		if ctx.Game.resolution.damageSourceBindings == nil {
			ctx.Game.resolution.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource)
		}
		ctx.Game.resolution.damageSourceBindings[e.sourceResult] = cloneDamageSource(outcome.source)
	}
	if e.outcomes != nil && outcome.amount > 0 {
		ctx.Game.resolution.damageResultsBindings[e.outcomes] = []damageOutcome{cloneDamageOutcome(outcome)}
	}
	return nil
}

// --- from composable_damage_results.go ---
// DamageResultsBinding names the final actions of one damage batch.
type DamageResultsBinding struct{ name string }

// DamageResultBinding names one final action during result iteration.
type DamageResultBinding struct{ name string }

func BindDamageResults(name string) *DamageResultsBinding { return &DamageResultsBinding{name} }
func BindDamageResult(name string) *DamageResultBinding   { return &DamageResultBinding{name} }

func (b *DamageResultsBinding) bindingName() string { return b.name }
func (b *DamageResultBinding) bindingName() string  { return b.name }
func (b *DamageResultsBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-results-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (b *DamageResultBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-result-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type damageResultsOption struct{ binding *DamageResultsBinding }

// DamageResultsInto saves final positive damage actions for later instructions.
func DamageResultsInto(binding *DamageResultsBinding) DamageEachResultOption {
	return damageResultsOption{binding}
}
func (damageResultsOption) damageEachResultOption() {}
func (o damageResultsOption) applyDamageEachResult(e *damageEachEffect) {
	e.results = o.binding
	e.resultsCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

func cloneDamageOutcome(outcome damageOutcome) damageOutcome {
	outcome.source = cloneDamageSource(outcome.source)
	return outcome
}
func cloneDamageOutcomes(outcomes []damageOutcome) []damageOutcome {
	if len(outcomes) == 0 {
		return nil
	}
	cloned := make([]damageOutcome, len(outcomes))
	for i, outcome := range outcomes {
		cloned[i] = cloneDamageOutcome(outcome)
	}
	return cloned
}
func cloneDamageResultsBindings(bindings map[*DamageResultsBinding][]damageOutcome) map[*DamageResultsBinding][]damageOutcome {
	if bindings == nil {
		return nil
	}
	cloned := make(map[*DamageResultsBinding][]damageOutcome, len(bindings))
	for binding, outcomes := range bindings {
		cloned[binding] = cloneDamageOutcomes(outcomes)
	}
	return cloned
}
func cloneDamageSourceBindings(bindings map[*DamageSourceBinding]DamageSource) map[*DamageSourceBinding]DamageSource {
	if bindings == nil {
		return nil
	}
	cloned := make(map[*DamageSourceBinding]DamageSource, len(bindings))
	for binding, source := range bindings {
		cloned[binding] = cloneDamageSource(source)
	}
	return cloned
}

type damageResultLoop struct {
	results *DamageResultsBinding
	member  *DamageResultBinding
	body    ResolvingEffect
}

// ForEachDamageResult runs its body after the complete damage batch.
func ForEachDamageResult(results *DamageResultsBinding, member *DamageResultBinding, body ResolvingEffect) ResolvingEffect {
	return &damageResultLoop{results, member, body}
}
func (*damageResultLoop) Text() string { return "for each damage result" }
func (e *damageResultLoop) Properties() EffectProperties {
	var props EffectProperties
	if !nilDefinition(e.body) {
		props = e.body.Properties()
	}
	props.Mass = true
	return props
}
func (e *damageResultLoop) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-result-loop", isolatedBindings: true, children: []definitionEdge{
		{"results", damageResultsRead{e.results}}, {"member", damageResultMember{e.member}}, {"body", e.body},
	}}
	if e.member == nil {
		n.problem = "supply a damage result member binding"
	}
	return n
}
func (e *damageResultLoop) Apply(ctx *EffectContext) error {
	outcomes := cloneDamageOutcomes(ctx.Game.resolution.damageResultsBindings[e.results])
	state := &ctx.Game.resolution
	outerWords := state.wordPairBindings
	outerNames := state.cardNameBindings
	outerNumbers, outerPlayers := state.numberBindings, state.playerBindings
	outerPermanents, outerSources := state.permanentBindings, state.damageSourceBindings
	outerResults, outerItems := state.damageResultsBindings, state.damageResultBindings
	defer func() {
		state.wordPairBindings = outerWords
		state.cardNameBindings = outerNames
		state.numberBindings, state.playerBindings = outerNumbers, outerPlayers
		state.permanentBindings, state.damageSourceBindings = outerPermanents, outerSources
		state.damageResultsBindings, state.damageResultBindings = outerResults, outerItems
	}()
	for _, outcome := range outcomes {
		state.wordPairBindings = maps.Clone(outerWords)
		state.cardNameBindings = maps.Clone(outerNames)
		state.numberBindings, state.playerBindings = maps.Clone(outerNumbers), maps.Clone(outerPlayers)
		state.permanentBindings, state.damageSourceBindings = maps.Clone(outerPermanents), cloneDamageSourceBindings(outerSources)
		state.damageResultsBindings = cloneDamageResultsBindings(outerResults)
		state.damageResultBindings = maps.Clone(outerItems)
		if state.damageResultBindings == nil {
			state.damageResultBindings = make(map[*DamageResultBinding]damageOutcome)
		}
		state.damageResultBindings[e.member] = outcome
		if err := e.body.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

type damageResultsRead struct{ binding *DamageResultsBinding }

func (r damageResultsRead) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-results-ref", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultMember struct{ binding *DamageResultBinding }

func (m damageResultMember) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-result-member", bindingWrite: m.binding, children: []definitionEdge{{"binding", m.binding}}}
}

type damageResultAmount struct{ binding *DamageResultBinding }

func DamageResultAmount(binding *DamageResultBinding) ResolutionValue {
	return damageResultAmount{binding}
}
func (r damageResultAmount) Text() string { return "damage result amount" }
func (r damageResultAmount) resolveValue(ctx *EffectContext) int {
	return ctx.Game.resolution.damageResultBindings[r.binding].amount
}
func (r damageResultAmount) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: r.Text(), bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultPlayer struct{ binding *DamageResultBinding }

func DamageResultPlayer(binding *DamageResultBinding) PlayerRef { return damageResultPlayer{binding} }
func (damageResultPlayer) playerRef()                           {}
func (damageResultPlayer) damageRecipient()                     {}
func (r damageResultPlayer) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	id := ctx.Game.resolution.damageResultBindings[r.binding].playerID
	return id, id != uuid.Nil && ctx.Game.GetPlayer(id) != nil
}
func (r damageResultPlayer) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r damageResultPlayer) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "damage result player", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultPermanent struct{ binding *DamageResultBinding }

func DamageResultPermanent(binding *DamageResultBinding) PermanentRef {
	return damageResultPermanent{binding}
}
func (damageResultPermanent) permanentRef()    {}
func (damageResultPermanent) damageRecipient() {}
func (r damageResultPermanent) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (r damageResultPermanent) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	value := ctx.Game.resolution.damageResultBindings[r.binding].permanent
	permanent := ctx.Game.FindPermanent(value.id)
	if permanent == nil || permanent.PhasedOut || value.incarnation == uuid.Nil || permanent.incarnationID != value.incarnation {
		return nil, false
	}
	return permanent, true
}
func (r damageResultPermanent) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-ref", Text: "damage result permanent", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultSource struct{ binding *DamageResultBinding }

func DamageResultSource(binding *DamageResultBinding) DamageSourceRef {
	return damageResultSource{binding}
}
func (damageResultSource) damageSourceRef() {}
func (r damageResultSource) resolveDamageSource(ctx *EffectContext) DamageSource {
	return ctx.Game.resolution.damageResultBindings[r.binding].source
}
func (r damageResultSource) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-source-ref", Text: "damage result source", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

// --- from composable_discard.go ---
type discardSubjectEffect struct {
	countResult      *NumberBinding
	countOptionCount int
	subject          PlayerRef
	amount           ResolutionValue
	cardResult       *CardBinding
	cardsResult      *CardsBinding
	optionCount      int
	setOptionCount   int
	invalidOption    bool
}

// DiscardCardResultOption selects a saved result from one discard instruction.
type DiscardCardResultOption interface {
	discardCardResultOption()
	applyDiscardCardResult(*discardSubjectEffect)
}

type discardedCountOption struct{ binding *NumberBinding }

// DiscardCountInto saves the number of completed discards after replacements.
func DiscardCountInto(binding *NumberBinding) DiscardCardResultOption {
	return discardedCountOption{binding}
}
func (discardedCountOption) discardCardResultOption() {}
func (o discardedCountOption) applyDiscardCardResult(e *discardSubjectEffect) {
	e.countResult = o.binding
	e.countOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type discardedCardOption struct{ binding *CardBinding }

type discardedCardsOption struct{ binding *CardsBinding }

// DiscardCardsInto saves graveyard visits from one discard instruction.
func DiscardCardsInto(binding *CardsBinding) DiscardCardResultOption {
	return discardedCardsOption{binding: binding}
}
func (discardedCardsOption) discardCardResultOption() {}
func (o discardedCardsOption) applyDiscardCardResult(e *discardSubjectEffect) {
	e.cardsResult = o.binding
	e.setOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DiscardCardInto saves a card that the instruction discards into a graveyard.
func DiscardCardInto(binding *CardBinding) DiscardCardResultOption {
	return discardedCardOption{binding: binding}
}
func (discardedCardOption) discardCardResultOption() {}
func (o discardedCardOption) applyDiscardCardResult(e *discardSubjectEffect) {
	e.cardResult = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

func (e *discardSubjectEffect) Text() string {
	return "discard for " + InspectDefinition(e.subject).Text
}
func (*discardSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardSubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
	if e.invalidOption {
		n.problem = "supply a non-nil discard result binding"
	} else if e.countOptionCount > 1 {
		n.problem = "supply the discard count result only once"
	} else if e.countResult != nil && (e.cardResult != nil || e.cardsResult != nil) {
		n.problem = "use a separate instruction for card and count results"
	} else if e.optionCount > 1 {
		n.problem = "supply the discard card result only once"
	} else if e.setOptionCount > 1 {
		n.problem = "supply the discard cards result only once"
	} else if e.cardResult != nil {
		if fixed, ok := e.amount.(fixedValue); !ok || fixed.n != 1 {
			n.problem = "discard one card when saving its card result"
		}
	}
	if e.countResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.countResult)
		n.children = append(n.children, definitionEdge{"count result", e.countResult})
	}
	if e.cardResult != nil {
		n.bindingWrite = e.cardResult
		n.children = append(n.children, definitionEdge{"card result", e.cardResult})
	}
	if e.cardsResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.cardsResult)
		n.children = append(n.children, definitionEdge{"cards result", e.cardsResult})
	}
	return n
}
func (e *discardSubjectEffect) Apply(ctx *EffectContext) error {
	if e.countResult != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.countResult] = 0
		id, ok := e.subject.resolvePlayer(ctx)
		if !ok {
			return nil
		}
		player := ctx.Game.GetPlayer(id)
		if player == nil {
			return nil
		}
		amount := max(0, e.amount.resolveValue(ctx))
		actions := chooseHandDiscards(ctx, snapshotHandGroup(ctx, player), amount)
		ctx.Game.resolution.numberBindings[e.countResult] = len(ctx.Game.discardActionsTogetherWithEvents(actions, false))
		return nil
	}
	if e.cardResult != nil {
		delete(ctx.Game.resolution.cardBindings, e.cardResult)
	}
	if e.cardsResult != nil {
		delete(ctx.Game.resolution.cardsBindings, e.cardsResult)
	}
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	if e.cardResult != nil || e.cardsResult != nil {
		player := ctx.Game.GetPlayer(id)
		if player == nil {
			return nil
		}
		chosen := player.ChooseCardsFromHand(amount, "discard", ctx.Game)
		for _, card := range chosen {
			if discarded, ok := ctx.Game.PlayerDiscardByEffect(player, card.ID(), ctx.SourceID); ok {
				if visit := ctx.Game.graveyardVisit(discarded.ID()); visit != uuid.Nil {
					captured := capturedCard{id: discarded.ID(), visit: visit, zone: ZoneGraveyard}
					if e.cardResult != nil {
						if ctx.Game.resolution.cardBindings == nil {
							ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
						}
						ctx.Game.resolution.cardBindings[e.cardResult] = captured
					}
					if e.cardsResult != nil {
						if ctx.Game.resolution.cardsBindings == nil {
							ctx.Game.resolution.cardsBindings = make(map[*CardsBinding][]capturedCard)
						}
						ctx.Game.resolution.cardsBindings[e.cardsResult] = append(ctx.Game.resolution.cardsBindings[e.cardsResult], captured)
					}
				}
			}
		}
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return DiscardCards(Fixed(amount)).Apply(&child)
}

// --- from composable_discard_card.go ---
type discardCardEffect struct{ card CardRef }

// DiscardCard discards a referenced card if it remains in its owner's hand.
func DiscardCard(card CardRef) ResolvingEffect { return &discardCardEffect{card: card} }

func (*discardCardEffect) Text() string { return "discard the referenced card" }
func (*discardCardEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardCardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
}
func (e *discardCardEffect) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	owner := ctx.Game.GetPlayer(card.Owner())
	if owner == nil {
		return nil
	}
	for _, held := range owner.Hand() {
		if held.ID() == card.ID() {
			ctx.Game.PlayerDiscardByEffect(owner, card.ID(), ctx.SourceID)
			return nil
		}
	}
	return nil
}

// --- from composable_discard_each.go ---
type discardEachEffect struct {
	subjects    PlayerSet
	result      *NumberBinding
	resultCount int
	invalid     bool
}

type DiscardEachResultOption interface {
	discardEachResultOption()
	applyDiscardEachResult(*discardEachEffect)
}

type discardTotalOption struct{ binding *NumberBinding }

// DiscardTotalInto saves the number of completed discards in one batch.
func DiscardTotalInto(binding *NumberBinding) DiscardEachResultOption {
	return discardTotalOption{binding}
}
func (discardTotalOption) discardEachResultOption() {}
func (o discardTotalOption) applyDiscardEachResult(e *discardEachEffect) {
	e.result = o.binding
	e.resultCount++
	if o.binding == nil {
		e.invalid = true
	}
}

// DiscardEach makes the selected players discard their hands together.
func DiscardEach(subjects PlayerSet, options ...DiscardEachResultOption) ResolvingEffect {
	e := &discardEachEffect{subjects: subjects}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyDiscardEachResult(e)
	}
	return e
}

func (e *discardEachEffect) Text() string { return "each selected player discards their hand" }
func (*discardEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *discardEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
	if e.invalid {
		n.problem = "supply a non-nil discard total binding"
	} else if e.resultCount > 1 {
		n.problem = "supply the discard total binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *discardEachEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	count := ctx.Game.discardHandsTogether(e.subjects.resolvePlayers(ctx), ctx.SourceID)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = count
	}
	return nil
}

// --- from composable_discard_hand.go ---
type discardEntireHandEffect struct {
	subject PlayerRef
	result  *NumberBinding
	count   int
	invalid bool
}

// DiscardEntireHand makes one player discard the hand they have at resolution.
func DiscardEntireHand(subject PlayerRef, options ...DiscardResultOption) ResolvingEffect {
	e := &discardEntireHandEffect{subject: subject}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyDiscardResult(e)
	}
	return e
}

// DiscardResultOption selects a result from one hand-discard instruction.
type DiscardResultOption interface {
	discardResultOption()
	applyDiscardResult(*discardEntireHandEffect)
}

type discardAmountOption struct{ binding *NumberBinding }

// DiscardAmountInto saves the number of cards discarded by this instruction.
func DiscardAmountInto(binding *NumberBinding) DiscardResultOption {
	return discardAmountOption{binding: binding}
}
func (discardAmountOption) discardResultOption() {}
func (o discardAmountOption) applyDiscardResult(e *discardEntireHandEffect) {
	e.result = o.binding
	e.count++
	if o.binding == nil {
		e.invalid = true
	}
}

func (e *discardEntireHandEffect) Text() string {
	return "discard entire hand for " + InspectDefinition(e.subject).Text
}
func (*discardEntireHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardEntireHandEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.invalid {
		n.problem = "supply a non-nil discard result binding"
	} else if e.count > 1 {
		n.problem = "supply the discard result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *discardEntireHandEffect) Apply(ctx *EffectContext) error {
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
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	count := discardPlayerHand(ctx.Game, player, ctx.SourceID)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = count
	}
	return nil
}

// --- from composable_draw_step_replacement.go ---
type drawStepLifeRule struct{ amount int }

// SkipDrawStepForLife lets the controller skip their draw step, then gain life.
// The gain occurs first in the next step that occurs. Use this rule with Static.
func SkipDrawStepForLife(amount int) StaticRule {
	if amount < 0 {
		panic("skip draw step: supply a nonnegative life amount")
	}
	return &drawStepLifeRule{amount: amount}
}

func (*drawStepLifeRule) staticRule() {}
func (r *drawStepLifeRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&drawStepLifeContinuous{amount: r.amount, ruleID: uuid.New()}}
}

type drawStepLifeContinuous struct {
	effectSource
	amount int
	ruleID uuid.UUID
}

func (e *drawStepLifeContinuous) Text() string {
	return fmt.Sprintf("you may skip your draw step; if you do, gain %d life", e.amount)
}
func (*drawStepLifeContinuous) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (*drawStepLifeContinuous) GetLayer() Layer       { return LayerAbility }
func (*drawStepLifeContinuous) GetDuration() Duration { return WhileOnBattlefield }
func (e *drawStepLifeContinuous) IsActive(g *Game) bool {
	source := g.FindPermanent(e.sourceID)
	return source != nil && !source.PhasedOut
}
func (e *drawStepLifeContinuous) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-draw-step-replacement", Text: e.Text()}
}
func (e *drawStepLifeContinuous) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.sourceID)
	if source == nil {
		return nil
	}
	ctx.Game.effects.AddCycleReplacement(&drawStepLifeReplacement{
		replacementBase: replacementBase{sourceID: e.sourceID, duration: WhileOnBattlefield},
		incarnation:     source.incarnationID, amount: e.amount, ruleID: e.ruleID,
	})
	return nil
}

func (r *drawStepLifeRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-draw-step-replacement", Text: fmt.Sprintf("you may skip your draw step; if you do, gain %d life", r.amount)}
}

func (e *drawStepLifeContinuous) cloneEffect() ContinuousEffect {
	clone := *e
	return &clone
}

// --- from composable_entry_counters.go ---
// EntryCounterValue reads a value while one permanent enters the battlefield.
type EntryCounterValue interface {
	entryCounterValue()
	entryCounterAmount(battlefieldEntryContext) int
	inspectDefinition() definitionNode
}

type entryCastColorsValue struct{}

// ColorsSpentToCastEnteringPermanent reads the entering spell's own payment.
func ColorsSpentToCastEnteringPermanent() EntryCounterValue { return entryCastColorsValue{} }

func (entryCastColorsValue) entryCounterValue() {}
func (entryCastColorsValue) entryCounterAmount(ctx battlefieldEntryContext) int {
	return ctx.castColorsSpent
}
func (entryCastColorsValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-counter-value", Text: "colors spent to cast entering permanent"}
}

type entryAnnouncedXValue struct{}

// AnnouncedXForEnteringPermanent reads X from the spell that becomes this permanent.
// Entry without a resolving permanent spell supplies zero.
func AnnouncedXForEnteringPermanent() EntryCounterValue { return entryAnnouncedXValue{} }

func (entryAnnouncedXValue) entryCounterValue() {}
func (entryAnnouncedXValue) entryCounterAmount(ctx battlefieldEntryContext) int {
	return ctx.announcedX
}
func (entryAnnouncedXValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-counter-value", Text: "announced X for entering permanent"}
}

type entryCountersAbility struct {
	BaseAbility
	counter CounterType
	amount  EntryCounterValue
}

// EntersWithCounters adds counters before the permanent enters the battlefield.
func EntersWithCounters(counter CounterType, amount EntryCounterValue) Ability {
	return &entryCountersAbility{
		BaseAbility: BaseAbility{id: uuid.New(), abilityType: AbilityStatic},
		counter:     counter,
		amount:      amount,
	}
}

func (a *entryCountersAbility) inspectDefinition() definitionNode {
	n := definitionNode{Role: "entry-replacement", Text: "enters with counters", children: []definitionEdge{{"amount", a.amount}}}
	if a.counter < 0 || a.counter >= NumCounters {
		n.problem = "invalid counter type"
	} else if nilDefinition(a.amount) {
		n.problem = "supply an entry amount"
	}
	return n
}

// --- from composable_fight.go ---
// Fight makes two referenced creatures deal damage equal to their power to each other.
func Fight(first, second PermanentRef) ResolvingEffect {
	return &fightReferencedEffect{first: first, second: second}
}

type fightReferencedEffect struct {
	first  PermanentRef
	second PermanentRef
}

func (*fightReferencedEffect) Text() string { return "creatures fight each other" }
func (*fightReferencedEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *fightReferencedEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"first", e.first}, {"second", e.second}}}
}
func (e *fightReferencedEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.first) || permanentRefReadsSourcePermanent(e.second)
}
func (e *fightReferencedEffect) Apply(ctx *EffectContext) error {
	first, firstOK := e.first.resolvePermanent(ctx)
	second, secondOK := e.second.resolvePermanent(ctx)
	if !firstOK || !secondOK || !first.HasType(TypeCreature) || !second.HasType(TypeCreature) {
		return nil
	}
	fightPermanents(ctx, first, second)
	return nil
}

// --- from composable_gain_life_each.go ---
type gainLifeEachEffect struct {
	subjects PlayerSet
	amount   ResolutionValue
	lifeGainResultState
}

type lifeGainResultState struct {
	result  *NumberBinding
	count   int
	invalid bool
}

// LifeGainResultOption selects the life gained by one instruction.
type LifeGainResultOption interface {
	lifeGainResultOption()
	applyLifeGainResult(*lifeGainResultState)
}

type lifeGainedOption struct{ binding *NumberBinding }

// LifeGainedInto saves the life gained by the selected players.
func LifeGainedInto(binding *NumberBinding) LifeGainResultOption { return lifeGainedOption{binding} }
func (lifeGainedOption) lifeGainResultOption()                   {}
func (o lifeGainedOption) applyLifeGainResult(e *lifeGainResultState) {
	e.result = o.binding
	e.count++
	if o.binding == nil {
		e.invalid = true
	}
}

// GainLifeEach gains life for each selected player in one instruction.
func GainLifeEach(subjects PlayerSet, amount ResolutionValue, options ...LifeGainResultOption) ResolvingEffect {
	e := &gainLifeEachEffect{subjects: subjects, amount: amount}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyLifeGainResult(&e.lifeGainResultState)
	}
	return e
}

func (e *gainLifeEachEffect) Text() string { return "gain life for each player" }
func (*gainLifeEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *gainLifeEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
	if e.invalid {
		n.problem = "supply a non-nil life gain result binding"
	} else if e.count > 1 {
		n.problem = "supply the life gain result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *gainLifeEachEffect) Apply(ctx *EffectContext) error {
	ids := e.subjects.resolvePlayers(ctx)
	amount := e.amount.resolveValue(ctx)
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	players := make([]Player, 0, len(ids))
	for _, id := range ids {
		if player := ctx.Game.GetPlayer(id); player != nil {
			players = append(players, player)
		}
	}
	gained := ctx.Game.playersGainLife(players, amount)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = gained
	}
	return nil
}

// --- from composable_graveyard.go ---
// CardRef identifies one card in a nonbattlefield zone.
type CardRef interface {
	cardRef()
	resolveCard(*EffectContext) (Card, bool)
}

// CardTargetBinding declares one card target in a supported zone.
type CardTargetBinding struct{ *targetDeclaration }

// GraveyardTargetOption restricts a graveyard card target.
type GraveyardTargetOption interface {
	graveyardTargetOption()
	applyGraveyardTarget(*graveyardTargetConfig)
}

type graveyardTargetConfig struct {
	any              bool
	filters          []CardFilter
	manaValue        TargetNumber
	manaValueOptions int
	upToOneOptions   int
}

type graveyardUpToOneOption struct{}

// GraveyardUpToOne permits zero or one graveyard card target.
func GraveyardUpToOne() GraveyardTargetOption         { return graveyardUpToOneOption{} }
func (graveyardUpToOneOption) graveyardTargetOption() {}
func (graveyardUpToOneOption) applyGraveyardTarget(config *graveyardTargetConfig) {
	config.upToOneOptions++
}

// TargetNumber reads a number while targets are chosen or checked.
type TargetNumber interface {
	targetNumber()
	targetValue(*Game, uuid.UUID) int
}

type fixedTargetNumber struct{ value int }

// FixedTargetNumber supplies one target restriction number.
func FixedTargetNumber(value int) TargetNumber                   { return fixedTargetNumber{value} }
func (fixedTargetNumber) targetNumber()                          {}
func (n fixedTargetNumber) targetValue(_ *Game, _ uuid.UUID) int { return n.value }
func (n fixedTargetNumber) inspectDefinition() definitionNode {
	node := definitionNode{Role: "target-number", Text: "fixed target number"}
	if n.value < 0 {
		node.problem = "mana value limit must be nonnegative"
	}
	return node
}

type controllerLifeGainedTargetNumber struct{}

// ControllerLifeGainedThisTurn reads actual life gain for the ability controller.
func ControllerLifeGainedThisTurn() TargetNumber       { return controllerLifeGainedTargetNumber{} }
func (controllerLifeGainedTargetNumber) targetNumber() {}
func (controllerLifeGainedTargetNumber) targetValue(g *Game, controller uuid.UUID) int {
	return g.PlayerLifeGainedThisTurn(controller)
}
func (controllerLifeGainedTargetNumber) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-number", Text: "controller life gained this turn"}
}

type graveyardManaValueOption struct{ maximum TargetNumber }

// GraveyardCardManaValueAtMost restricts a graveyard target by its current mana value.
func GraveyardCardManaValueAtMost(maximum TargetNumber) GraveyardTargetOption {
	return graveyardManaValueOption{maximum}
}
func (graveyardManaValueOption) graveyardTargetOption() {}
func (o graveyardManaValueOption) applyGraveyardTarget(config *graveyardTargetConfig) {
	config.manaValue = o.maximum
	config.manaValueOptions++
}

type manaValueBoundGraveyardTarget struct {
	inner       Target
	maximum     TargetNumber
	optionCount int
}

func (t *manaValueBoundGraveyardTarget) Possible(controller uuid.UUID, source Card, g *Game) []uuid.UUID {
	if nilDefinition(t.maximum) || g.GetPlayer(controller) == nil {
		return nil
	}
	limit := t.maximum.targetValue(g, controller)
	var result []uuid.UUID
	for _, id := range t.inner.Possible(controller, source, g) {
		card := g.FindCardAnywhere(id)
		if card != nil && card.ManaCost().CMC() <= limit {
			result = append(result, id)
		}
	}
	return result
}
func (t *manaValueBoundGraveyardTarget) Choose(c uuid.UUID, card Card, g *Game, chosen []uuid.UUID) error {
	return t.inner.Choose(c, card, g, chosen)
}
func (t *manaValueBoundGraveyardTarget) Chosen() []uuid.UUID { return t.inner.Chosen() }
func (t *manaValueBoundGraveyardTarget) IsChosen() bool      { return t.inner.IsChosen() }
func (t *manaValueBoundGraveyardTarget) Min() int            { return t.inner.Min() }
func (t *manaValueBoundGraveyardTarget) Max() int            { return t.inner.Max() }
func (t *manaValueBoundGraveyardTarget) Reset()              { t.inner.Reset() }
func (t *manaValueBoundGraveyardTarget) inspectDefinition() definitionNode {
	node := definitionNode{Role: "target-restriction", Text: "graveyard card mana value", children: []definitionEdge{{"mana value", t.maximum}}}
	if nilDefinition(t.maximum) || t.optionCount != 1 {
		node.problem = "supply one mana value limit"
	}
	return node
}

type anyGraveyardOption struct{}

// InAnyGraveyard permits a card in any player's graveyard.
func InAnyGraveyard() GraveyardTargetOption       { return anyGraveyardOption{} }
func (anyGraveyardOption) graveyardTargetOption() {}
func (anyGraveyardOption) applyGraveyardTarget(config *graveyardTargetConfig) {
	config.any = true
}

type graveyardCardFilterOption struct{ filter CardFilter }

// GraveyardCardFilter restricts the card types allowed by the target.
func GraveyardCardFilter(filter CardFilter) GraveyardTargetOption {
	return graveyardCardFilterOption{filter}
}
func (graveyardCardFilterOption) graveyardTargetOption() {}
func (o graveyardCardFilterOption) applyGraveyardTarget(config *graveyardTargetConfig) {
	config.filters = append(config.filters, o.filter)
}

// DeclareGraveyardCardTarget declares one target in your graveyard by default.
func DeclareGraveyardCardTarget(name string, options ...GraveyardTargetOption) *CardTargetBinding {
	config := graveyardTargetConfig{}
	for _, option := range options {
		if nilDefinition(option) {
			panic("targets: supply a non-nil graveyard target option")
		}
		option.applyGraveyardTarget(&config)
	}
	for _, filter := range config.filters {
		if nilDefinition(filter) {
			panic("targets: supply a non-nil graveyard card filter")
		}
	}
	var spec Target
	if config.any {
		spec = TargetCardInAnyGraveyard(slices.Clone(config.filters)...)
	} else {
		spec = TargetCardInYourGraveyard(slices.Clone(config.filters)...)
	}
	if config.upToOneOptions > 0 {
		spec.(*GraveyardCardTarget).min = 0
	}
	if config.manaValueOptions > 0 {
		spec = &manaValueBoundGraveyardTarget{inner: spec, maximum: config.manaValue, optionCount: config.manaValueOptions}
	}
	return &CardTargetBinding{&targetDeclaration{name: name, spec: spec, optionalGraveyardCount: config.upToOneOptions}}
}

func (*CardTargetBinding) cardRef() {}
func (b *CardTargetBinding) resolveCard(ctx *EffectContext) (Card, bool) {
	value, ok := firstTargetValue(ctx.targetBindings[b.targetDeclaration])
	if _, ante := b.spec.(*anteCardTarget); ante {
		if !ok {
			return nil, false
		}
		return ctx.Game.resolveAnteCard(value.id, value.incarnation)
	}
	if !ok || value.incarnation == uuid.Nil || ctx.Game.graveyardVisit(value.id) != value.incarnation {
		return nil, false
	}
	for _, player := range ctx.Game.players {
		for _, card := range player.Graveyard() {
			if card.ID() == value.id {
				return card, true
			}
		}
	}
	return nil, false
}
func (b *CardTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "card-ref")
}

type returnCardToHandEffect struct{ card CardRef }

// ReturnCardToHand puts the referenced graveyard card into its owner's hand.
func ReturnCardToHand(card CardRef) ResolvingEffect { return &returnCardToHandEffect{card} }
func (*returnCardToHandEffect) Text() string        { return "return card to its owner's hand" }
func (*returnCardToHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *returnCardToHandEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
}
func (e *returnCardToHandEffect) Apply(ctx *EffectContext) error {
	returnGraveyardCardToHand(ctx, e.card)
	return nil
}

type returnCardToBattlefieldEffect struct {
	card       CardRef
	controller PlayerRef
	result     *PermanentBinding
	saveResult bool
	optional   bool
	entry      battlefieldEntryOptions
}

// ReturnCardToBattlefield puts the referenced graveyard card onto the battlefield.
func ReturnCardToBattlefield(card CardRef, controller PlayerRef, options ...BattlefieldEntryOption) ResolvingEffect {
	return newReturnCardToBattlefieldEffect(card, controller, options...)
}

func newReturnCardToBattlefieldEffect(card CardRef, controller PlayerRef, options ...BattlefieldEntryOption) *returnCardToBattlefieldEffect {
	effect := &returnCardToBattlefieldEffect{card: card, controller: controller, entry: newBattlefieldEntryOptions(options)}
	return effect
}

// ReturnCardToBattlefieldInto saves the exact entrant for later instructions.
// Failed entry clears the result so an earlier entrant cannot supply it.
func ReturnCardToBattlefieldInto(card CardRef, controller PlayerRef, result *PermanentBinding, options ...BattlefieldEntryOption) ResolvingEffect {
	effect := newReturnCardToBattlefieldEffect(card, controller, options...)
	effect.result = result
	effect.saveResult = true
	return effect
}

// MayReturnNonAuraCardToBattlefield offers entry for one non-Aura graveyard permanent.
// The selected player chooses at resolution and controls the card.
// The saved visit must remain current after the choice.
func MayReturnNonAuraCardToBattlefield(card CardRef, player PlayerRef) ResolvingEffect {
	return &returnCardToBattlefieldEffect{card: card, controller: player, optional: true}
}

func (e *returnCardToBattlefieldEffect) Text() string {
	if e.optional {
		return "may return card from graveyard to the battlefield"
	}
	return "return card from graveyard to the battlefield"
}
func (*returnCardToBattlefieldEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *returnCardToBattlefieldEffect) refersToSourcePermanent() bool {
	return enchantedCardReadsSource(e.card) || playerRefReadsSourcePermanent(e.controller) || permanentRefReadsSourcePermanent(e.entry.host) || permanentRefReadsSourcePermanent(e.entry.historySource)
}
func (e *returnCardToBattlefieldEffect) inspectDefinition() definitionNode {
	node := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}, {"controller", e.controller}}}
	if e.saveResult {
		node.bindingWrite = e.result
		node.children = append(node.children, definitionEdge{"result", e.result})
	}
	e.entry.inspect(&node)
	return node
}
func (e *returnCardToBattlefieldEffect) Apply(ctx *EffectContext) error {
	entrySource := e.entry.captureHistorySource(ctx)
	if e.saveResult {
		delete(ctx.Game.resolution.permanentBindings, e.result)
	}
	card, ok := e.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	controller, ok := e.controller.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	if e.optional {
		visit := ctx.Game.graveyardVisit(card.ID())
		player := ctx.Game.GetPlayer(controller)
		if visit == uuid.Nil || !IsPermanentCard.Match(card) || card.HasSubType("Aura") || player == nil || !player.ChooseMayAbility(e.Text()) {
			return nil
		}
		if ctx.Game.graveyardVisit(card.ID()) != visit {
			return nil
		}
	}
	attachment := e.entry.captureAttachment(ctx)
	permanent := returnGraveyardCardToBattlefieldWithHistory(ctx.Game, card.ID(), controller, controller, e.entry.captureCounters(ctx), attachment, entrySource)
	if e.saveResult && permanent != nil {
		if ctx.Game.resolution.permanentBindings == nil {
			ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
		}
		ctx.Game.resolution.permanentBindings[e.result] = capturedPermanent{permanent.ID(), permanent.incarnationID}
	}
	return nil
}

// --- from composable_library_choice.go ---
type libraryChoiceEffect struct {
	amount  ResolutionValue
	surveil bool
}

// ScryAtResolution reads the amount in the current frame, then scries for You.
func ScryAtResolution(amount ResolutionValue) ResolvingEffect {
	return &libraryChoiceEffect{amount: amount}
}

// SurveilAtResolution reads the amount in the current frame, then surveils for You.
func SurveilAtResolution(amount ResolutionValue) ResolvingEffect {
	return &libraryChoiceEffect{amount: amount, surveil: true}
}

func (e *libraryChoiceEffect) verb() string {
	if e.surveil {
		return "surveil"
	}
	return "scry"
}
func (e *libraryChoiceEffect) Text() string {
	if amount, ok := e.amount.(fixedValue); ok {
		return e.verb() + " " + amount.Text()
	}
	return e.verb()
}
func (*libraryChoiceEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *libraryChoiceEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: e.verb(), Text: e.Text(), children: []definitionEdge{{"amount", e.amount}}}
}
func (e *libraryChoiceEffect) refersToSourcePermanent() bool {
	return resolutionValueReadsSourcePermanent(e.amount)
}
func (e *libraryChoiceEffect) Apply(ctx *EffectContext) error {
	amount := Fixed(e.amount.resolveValue(ctx))
	if e.surveil {
		return Surveil(amount).Apply(ctx)
	}
	return Scry(amount).Apply(ctx)
}

// --- from composable_loop.go ---
type permanentLoop struct {
	turnOrder bool
	subjects  PermanentSet
	member    *PermanentBinding
	body      ResolvingEffect
}

// ForEach saves the member list once and runs the body in battlefield order.
// Each iteration has its own saved values. The actions are not simultaneous.
func ForEach(subjects PermanentSet, member *PermanentBinding, body ResolvingEffect) ResolvingEffect {
	return &permanentLoop{subjects: subjects, member: member, body: body}
}

// ForEachInTurnOrder saves visits in turn order before it runs the body.
// The effect controller chooses the order of visits within each control group.
func ForEachInTurnOrder(subjects PermanentSet, member *PermanentBinding, body ResolvingEffect) ResolvingEffect {
	return &permanentLoop{subjects: subjects, member: member, body: body, turnOrder: true}
}

func (*permanentLoop) Text() string { return "for each permanent" }
func (e *permanentLoop) Properties() EffectProperties {
	var props EffectProperties
	if !nilDefinition(e.body) {
		props = e.body.Properties()
	}
	props.Mass = true
	return props
}
func (e *permanentLoop) Apply(ctx *EffectContext) error {
	var members []capturedPermanent
	if e.turnOrder {
		groups := controlledPermanentGroups(ctx, EachPlayer, e.subjects)
		chooser := ctx.Game.GetPlayer(ctx.Controller)
		for _, group := range groups {
			remaining := slices.Clone(group.members)
			for len(remaining) > 0 {
				index := 0
				if len(remaining) > 1 && chooser != nil {
					chosen := chooser.ChoosePermanent(slices.Clone(remaining), "Choose the next permanent for iteration", ctx.Game)
					index = max(0, slices.IndexFunc(remaining, func(p *Permanent) bool {
						return chosen != nil && p.ID() == chosen.ID() && p.incarnationID == chosen.incarnationID
					}))
				}
				p := remaining[index]
				members = append(members, capturedPermanent{p.ID(), p.incarnationID})
				remaining = slices.Delete(remaining, index, index+1)
			}
		}
	} else {
		for _, p := range e.subjects.resolvePermanents(ctx) {
			members = append(members, capturedPermanent{p.ID(), p.incarnationID})
		}
	}
	outerWords := ctx.Game.resolution.wordPairBindings
	defer func() { ctx.Game.resolution.wordPairBindings = outerWords }()
	outerNames := ctx.Game.resolution.cardNameBindings
	defer func() { ctx.Game.resolution.cardNameBindings = outerNames }()
	outerCards := ctx.Game.resolution.cardBindings
	defer func() { ctx.Game.resolution.cardBindings = outerCards }()
	outerNumbers := ctx.Game.resolution.numberBindings
	defer func() { ctx.Game.resolution.numberBindings = outerNumbers }()
	outerPlayers := ctx.Game.resolution.playerBindings
	defer func() { ctx.Game.resolution.playerBindings = outerPlayers }()
	outer := ctx.Game.resolution.permanentBindings
	defer func() { ctx.Game.resolution.permanentBindings = outer }()
	outerCreated := ctx.Game.resolution.createdTokenBindings
	defer func() { ctx.Game.resolution.createdTokenBindings = outerCreated }()
	outerCardSets := ctx.Game.resolution.cardsBindings
	outerSpells := ctx.Game.resolution.spellBindings
	outerManaTypes := ctx.Game.resolution.manaTypesBindings
	outerDamageSources := ctx.Game.resolution.damageSourceBindings
	outerDamageResults := ctx.Game.resolution.damageResultsBindings
	outerDamageResult := ctx.Game.resolution.damageResultBindings
	defer func() {
		ctx.Game.resolution.cardsBindings = outerCardSets
		ctx.Game.resolution.spellBindings = outerSpells
		ctx.Game.resolution.manaTypesBindings = outerManaTypes
		ctx.Game.resolution.damageSourceBindings = outerDamageSources
		ctx.Game.resolution.damageResultsBindings = outerDamageResults
		ctx.Game.resolution.damageResultBindings = outerDamageResult
	}()
	for _, member := range members {
		ctx.Game.resolution.cardsBindings = cloneCardsBindings(outerCardSets)
		ctx.Game.resolution.spellBindings = maps.Clone(outerSpells)
		ctx.Game.resolution.manaTypesBindings = cloneManaTypesBindings(outerManaTypes)
		ctx.Game.resolution.damageSourceBindings = cloneDamageSourceBindings(outerDamageSources)
		ctx.Game.resolution.damageResultsBindings = cloneDamageResultsBindings(outerDamageResults)
		ctx.Game.resolution.damageResultBindings = maps.Clone(outerDamageResult)
		for binding, result := range ctx.Game.resolution.damageResultBindings {
			ctx.Game.resolution.damageResultBindings[binding] = cloneDamageOutcome(result)
		}

		ctx.Game.resolution.cardBindings = maps.Clone(outerCards)
		ctx.Game.resolution.createdTokenBindings = cloneCreatedTokenBindings(outerCreated)
		ctx.Game.resolution.wordPairBindings = maps.Clone(outerWords)
		ctx.Game.resolution.cardNameBindings = maps.Clone(outerNames)
		ctx.Game.resolution.numberBindings = maps.Clone(outerNumbers)
		ctx.Game.resolution.playerBindings = maps.Clone(outerPlayers)
		frame := maps.Clone(outer)
		if frame == nil {
			frame = make(map[*PermanentBinding]capturedPermanent)
		}
		frame[e.member] = member
		ctx.Game.resolution.permanentBindings = frame
		if err := CompositeEffects("", e.body).Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}
func (e *permanentLoop) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-loop", isolatedBindings: true, children: []definitionEdge{
		{"subjects", e.subjects}, {"member", loopMember{e.member}}, {"body", e.body},
	}}
}

type loopMember struct{ binding *PermanentBinding }

func (m loopMember) inspectDefinition() definitionNode {
	return definitionNode{Role: "loop-member", bindingWrite: m.binding, children: []definitionEdge{{"binding", m.binding}}}
}

// --- from composable_card_loop.go ---
type savedCardLoop struct {
	cards  *CardsBinding
	member *CardBinding
	body   ResolvingEffect
}

// ForEachSavedCard runs the body once for each card saved by one instruction.
// Each member keeps its original zone visit, even if an earlier body moves it.
func ForEachSavedCard(cards *CardsBinding, member *CardBinding, body ResolvingEffect) ResolvingEffect {
	return &savedCardLoop{cards: cards, member: member, body: body}
}

func (*savedCardLoop) Text() string { return "for each saved card" }
func (e *savedCardLoop) Properties() EffectProperties {
	var props EffectProperties
	if !nilDefinition(e.body) {
		props = e.body.Properties()
	}
	props.Mass = true
	return props
}
func (e *savedCardLoop) inspectDefinition() definitionNode {
	n := definitionNode{Role: "saved-card-loop", isolatedBindings: true, children: []definitionEdge{
		{"cards", savedCardsSet{binding: e.cards}}, {"member", savedCardLoopMember{e.member}}, {"body", e.body},
	}}
	if e.cards == nil || e.member == nil || nilDefinition(e.body) {
		n.problem = "supply non-nil saved cards, member, and body"
	}
	return n
}
func (e *savedCardLoop) Apply(ctx *EffectContext) error {
	members := append([]capturedCard(nil), ctx.Game.resolution.cardsBindings[e.cards]...)
	state := &ctx.Game.resolution
	outer := *state
	defer func() {
		state.permanentBindings = outer.permanentBindings
		state.createdTokenBindings = outer.createdTokenBindings
		state.cardBindings = outer.cardBindings
		state.cardsBindings = outer.cardsBindings
		state.spellBindings = outer.spellBindings
		state.playerBindings = outer.playerBindings
		state.wordPairBindings = outer.wordPairBindings
		state.cardNameBindings = outer.cardNameBindings
		state.numberBindings = outer.numberBindings
		state.manaTypesBindings = outer.manaTypesBindings
		state.damageSourceBindings = outer.damageSourceBindings
		state.damageResultsBindings = outer.damageResultsBindings
		state.damageResultBindings = outer.damageResultBindings
	}()
	for _, captured := range members {
		state.permanentBindings = maps.Clone(outer.permanentBindings)
		state.createdTokenBindings = cloneCreatedTokenBindings(outer.createdTokenBindings)
		state.cardBindings = maps.Clone(outer.cardBindings)
		state.cardsBindings = cloneCardsBindings(outer.cardsBindings)
		state.spellBindings = maps.Clone(outer.spellBindings)
		state.playerBindings = maps.Clone(outer.playerBindings)
		state.wordPairBindings = maps.Clone(outer.wordPairBindings)
		state.cardNameBindings = maps.Clone(outer.cardNameBindings)
		state.numberBindings = maps.Clone(outer.numberBindings)
		state.manaTypesBindings = cloneManaTypesBindings(outer.manaTypesBindings)
		state.damageSourceBindings = cloneDamageSourceBindings(outer.damageSourceBindings)
		state.damageResultsBindings = cloneDamageResultsBindings(outer.damageResultsBindings)
		state.damageResultBindings = maps.Clone(outer.damageResultBindings)
		if state.cardBindings == nil {
			state.cardBindings = make(map[*CardBinding]capturedCard)
		}
		state.cardBindings[e.member] = captured
		if err := e.body.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

type savedCardLoopMember struct{ binding *CardBinding }

func (m savedCardLoopMember) inspectDefinition() definitionNode {
	return definitionNode{Role: "saved-card-loop-member", bindingWrite: m.binding, children: []definitionEdge{{"binding", m.binding}}}
}

// --- from composable_mill.go ---
type millSubjectEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

// Mill puts cards from the selected player's library into their graveyard.
// An absent player has no effect. The existing mill modifiers apply.
func Mill(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &millSubjectEffect{subject, amount}
}

func (e *millSubjectEffect) Text() string { return "mill for " + InspectDefinition(e.subject).Text }
func (*millSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *millSubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
	if amount, ok := e.amount.(fixedValue); ok && amount.n < 0 {
		n.problem = "supply a nonnegative mill count"
	}
	return n
}
func (e *millSubjectEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return MillTargetPlayer(Fixed(amount)).Apply(&child)
}

// --- from composable_opponent_life.go ---
type opponentsRef struct{ player PlayerRef }

// OpponentsOf selects the other player in this two-player engine.
// An absent player produces an empty set.
func OpponentsOf(player PlayerRef) PlayerSet { return &opponentsRef{player} }

func (*opponentsRef) playerSet() {}
func (s *opponentsRef) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	id, ok := s.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var opponents []uuid.UUID
	for _, candidate := range EachPlayer.resolvePlayers(ctx) {
		if candidate != id {
			opponents = append(opponents, candidate)
		}
	}
	return opponents
}
func (s *opponentsRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set", Text: "opponents of " + InspectDefinition(s.player).Text, children: []definitionEdge{{"player", s.player}}}
}

type loseLifeEachEffect struct {
	subjects PlayerSet
	amount   ResolutionValue
	lifeLossResultState
}

type lifeLossResultState struct {
	result  *NumberBinding
	count   int
	invalid bool
}

// LifeLossResultOption selects the total life lost from one set instruction.
type LifeLossResultOption interface {
	lifeLossResultOption()
	applyLifeLossResult(*lifeLossResultState)
}

type lifeLostOption struct{ binding *NumberBinding }

// LifeLostInto saves the total life lost by the selected players.
func LifeLostInto(binding *NumberBinding) LifeLossResultOption { return lifeLostOption{binding} }
func (lifeLostOption) lifeLossResultOption()                   {}
func (o lifeLostOption) applyLifeLossResult(e *lifeLossResultState) {
	e.result = o.binding
	e.count++
	if o.binding == nil {
		e.invalid = true
	}
}

// LoseLifeEach makes each selected player lose the specified amount of life.
// The instruction reads the amount once in the current resolution frame.
// It captures the players and amount before it changes life totals.
func LoseLifeEach(subjects PlayerSet, amount ResolutionValue, options ...LifeLossResultOption) ResolvingEffect {
	e := &loseLifeEachEffect{subjects: subjects, amount: amount}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyLifeLossResult(&e.lifeLossResultState)
	}
	return e
}

func (e *loseLifeEachEffect) Text() string {
	amount := ""
	if !nilDefinition(e.amount) {
		amount = e.amount.Text()
	}
	return "lose " + amount + " life for " + InspectDefinition(e.subjects).Text
}
func (*loseLifeEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *loseLifeEachEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	ids := e.subjects.resolvePlayers(ctx)
	if len(ids) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	players := make([]Player, 0, len(ids))
	for _, id := range ids {
		if player := ctx.Game.GetPlayer(id); player != nil {
			players = append(players, player)
		}
	}
	lost := ctx.Game.playersLoseLife(players, amount)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = lost
	}
	return nil
}
func (e *loseLifeEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "lose-life-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
	if e.invalid {
		n.problem = "supply a non-nil life loss result binding"
	} else if e.count > 1 {
		n.problem = "supply the life loss result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}

// --- from composable_prevention.go ---
// DamageRecipientRef identifies a player or permanent that can receive damage.
type DamageRecipientRef interface {
	damageRecipient()
	resolveRecipient(*EffectContext) (uuid.UUID, bool)
}

func (sourceRef) damageRecipient() {}
func (r sourceRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (savedPermanentRef) damageRecipient() {}
func (r savedPermanentRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (attachedPermanentRef) damageRecipient() {}
func (r attachedPermanentRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (*PermanentTargetBinding) damageRecipient() {}
func (b *PermanentTargetBinding) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(b, ctx)
}
func permanentRecipient(r PermanentRef, ctx *EffectContext) (uuid.UUID, bool) {
	p, ok := r.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return p.ID(), true
}
func (controllerRef) damageRecipient() {}
func (r controllerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (activePlayerRef) damageRecipient() {}
func (r activePlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (*PlayerTargetBinding) damageRecipient() {}
func (b *PlayerTargetBinding) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return b.resolvePlayer(ctx)
}

// DamageTargetBinding declares one target player, creature, or planeswalker.
type DamageTargetBinding struct{ *targetDeclaration }

// DeclareDamageTarget declares any damage target. DSL exports it as TargetDamageRecipient.
func DeclareDamageTarget(name string) *DamageTargetBinding {
	return &DamageTargetBinding{&targetDeclaration{name: name, spec: TargetDamageAnyTarget()}}
}

// DeclarePlayerOrPlaneswalkerTarget excludes creatures from this damage target.
func DeclarePlayerOrPlaneswalkerTarget(name string) *DamageTargetBinding {
	return &DamageTargetBinding{&targetDeclaration{name: name, spec: TargetPlayerOrPlaneswalker()}}
}

// DeclareOpponentOrPlaneswalkerTarget permits any planeswalker but excludes the controller as a player target.
func DeclareOpponentOrPlaneswalkerTarget(name string) *DamageTargetBinding {
	return &DamageTargetBinding{&targetDeclaration{name: name, spec: &PlayerOrPlaneswalkerTarget{
		BaseTarget: BaseTarget{min: 1, max: 1}, opponentOnly: true,
	}}}
}
func (*DamageTargetBinding) damageRecipient() {}
func (b *DamageTargetBinding) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	values := ctx.targetBindings[b.targetDeclaration]
	if len(values) == 0 {
		return uuid.Nil, false
	}
	value, _ := firstTargetValue(values)
	if ctx.Game.GetPlayer(value.id) != nil {
		return value.id, true
	}
	p := ctx.Game.FindPermanent(value.id)
	return value.id, p != nil && !p.PhasedOut && p.incarnationID == value.incarnation
}
func (b *DamageTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "damage-recipient-ref")
}

type preventionInstallation struct {
	recipient DamageRecipientRef
	amount    ResolutionValue
}

// PreventNextDamage installs an end-of-turn shield in the existing replacement pipeline.
func PreventNextDamage(recipient DamageRecipientRef, amount ResolutionValue) ResolvingEffect {
	return &preventionInstallation{recipient, amount}
}
func (e *preventionInstallation) Text() string { return "prevent the next damage this turn" }
func (e *preventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *preventionInstallation) Apply(ctx *EffectContext) error {
	id, ok := e.recipient.resolveRecipient(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	ctx.Game.AddPreventionShield(id, amount)
	return nil
}
func (e *preventionInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-prevention", Text: e.Text(), children: []definitionEdge{{"recipient", e.recipient}, {"amount", e.amount}}}
}

// PreventCombatDamageToAndBy prevents combat damage to and by the subject this turn.
func PreventCombatDamageToAndBy(subject PermanentRef) ResolvingEffect {
	return &combatPreventionInstallation{subject}
}

type combatPreventionInstallation struct{ subject PermanentRef }

func (*combatPreventionInstallation) Text() string {
	return "prevent combat damage to and by permanent this turn"
}
func (*combatPreventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *combatPreventionInstallation) Apply(ctx *EffectContext) error {
	if permanent, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.PreventCombatDamageToAndBy(permanent.ID())
	}
	return nil
}
func (e *combatPreventionInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-combat-prevention", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}

type combatSourcePreventionInstallation struct {
	duration Duration
	subject  PermanentRef
	subjects PermanentSet
}

// PreventCombatDamageBy prevents combat damage from one permanent this combat.
func PreventCombatDamageBy(subject PermanentRef) ResolvingEffect {
	return &combatSourcePreventionInstallation{subject: subject, duration: EndOfCombat}
}

// PreventCombatDamageByEach prevents combat damage from a selected set this combat.
func PreventCombatDamageByEach(subjects PermanentSet) ResolvingEffect {
	return &combatSourcePreventionInstallation{subjects: subjects, duration: EndOfCombat}
}

// PreventCombatDamageByThisTurn prevents combat damage from one permanent this turn.
func PreventCombatDamageByThisTurn(subject PermanentRef) ResolvingEffect {
	return &combatSourcePreventionInstallation{subject: subject, duration: EndOfTurn}
}

// PreventCombatDamageByEachThisTurn prevents combat damage from the set until cleanup.
// Selection occurs once, so later combat changes cannot add sources.
func PreventCombatDamageByEachThisTurn(subjects PermanentSet) ResolvingEffect {
	return &combatSourcePreventionInstallation{subjects: subjects, duration: EndOfTurn}
}

func (e *combatSourcePreventionInstallation) Text() string {
	if e.duration == EndOfTurn {
		return "prevent combat damage by selected source this turn"
	}
	return "prevent combat damage by selected sources this combat"
}
func (*combatSourcePreventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *combatSourcePreventionInstallation) Apply(ctx *EffectContext) error {
	var selected []*Permanent
	if e.subject != nil {
		if p, ok := e.subject.resolvePermanent(ctx); ok {
			selected = []*Permanent{p}
		}
	} else if e.subjects != nil {
		selected = e.subjects.resolvePermanents(ctx)
	}
	seen := make(map[uuid.UUID]bool)
	for _, p := range selected {
		if p == nil || p.PhasedOut || seen[p.incarnationID] {
			continue
		}
		current := ctx.Game.FindPermanent(p.ID())
		if current == nil || current.incarnationID != p.incarnationID {
			continue
		}
		seen[p.incarnationID] = true
		ctx.Game.effects.AddReplacement(&combatSourcePreventionReplacement{
			replacementBase: replacementBase{duration: e.duration},
			preventedID:     p.ID(), incarnation: p.incarnationID,
		})
	}
	return nil
}
func (e *combatSourcePreventionInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "install-combat-source-prevention", Text: e.Text()}
	if e.subject != nil && e.subjects == nil {
		n.children = []definitionEdge{{"subject", e.subject}}
	} else if e.subjects != nil && e.subject == nil {
		n.children = []definitionEdge{{"subjects", e.subjects}}
	} else {
		n.problem = "combat source prevention requires one subject or set"
	}
	return n
}

// --- from composable_random_discard.go ---
type randomDiscardSubjectEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

// DiscardAtRandom makes the selected player discard cards at random.
// Select before movement so each event sees the complete discard.
func DiscardAtRandom(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &randomDiscardSubjectEffect{subject, amount}
}

func (e *randomDiscardSubjectEffect) Text() string {
	return "discard at random for " + InspectDefinition(e.subject).Text
}
func (*randomDiscardSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *randomDiscardSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *randomDiscardSubjectEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	var actions []*DiscardAction
	for _, card := range ctx.Game.selectRandomCards(player.Hand(), amount) {
		actions = append(actions, NewDiscardAction(ctx.SourceID, id, card.ID()))
	}
	ctx.Game.discardActionsTogether(actions)
	return nil
}

// --- from composable_receives_damage.go ---
type receivesDamagePattern struct {
	amount            *NumberBinding
	controller        *PlayerBinding
	captureController bool
	source            *DamageSourceBinding
	captureSource     bool
	duplicateSource   bool
}

// ReceivesDamage saves positive damage dealt to the source permanent.
func ReceivesDamage(_ sourceRef, amount *NumberBinding) receivesDamagePattern {
	return receivesDamagePattern{amount: amount}
}

// WithSourceController saves the damage source controller at event time.
func (p receivesDamagePattern) WithSourceController(controller *PlayerBinding) receivesDamagePattern {
	p.controller, p.captureController = controller, true
	return p
}

// WithSource saves the damage source when damage occurs.
func (p receivesDamagePattern) WithSource(source *DamageSourceBinding) receivesDamagePattern {
	if p.captureSource {
		p.duplicateSource = true
	}
	p.source, p.captureSource = source, true
	return p
}

func (receivesDamagePattern) eventPattern() {}
func (receivesDamagePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventTargetIsSelf{}).
		AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p receivesDamagePattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source receives damage", battlefieldSource: true,
		bindingWrite: p.amount, children: []definitionEdge{{"amount", &eventAmountCapture{binding: p.amount}}}}
	if p.captureController {
		n.bindingWrites = append(n.bindingWrites, p.controller)
		n.children = append(n.children, definitionEdge{"source controller", &eventPlayerCapture{binding: p.controller}})
	}
	if p.captureSource {
		if p.source == nil || p.duplicateSource {
			n.problem = "damage event requires one source binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.source)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.source}})
		}
	}
	return n
}
func (p receivesDamagePattern) captureDetectedEvent(_ *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureSource || p.source == nil {
		return nil
	}
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	return []Effect{&eventDamageSourceCapture{binding: p.source, source: cloneDamageSource(game.damageEventSource)}}
}
func (p receivesDamagePattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	captures := []ResolvingEffect{&eventAmountCapture{binding: p.amount, amount: event.Amount}}
	if p.captureController {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: event.SourceControllerID})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

// --- from composable_regeneration.go ---
type regenerateSubjectEffect struct {
	subject PermanentRef
}

// Regenerate protects the subject from its next destruction this turn.
func Regenerate(subject PermanentRef) ResolvingEffect {
	return &regenerateSubjectEffect{subject: subject}
}

func (e *regenerateSubjectEffect) Text() string {
	if e.subject != nil {
		return "regenerate " + InspectDefinition(e.subject).Text
	}
	return "regenerate permanent"
}

func (*regenerateSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Regenerates: true}
}

func (e *regenerateSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}

func (e *regenerateSubjectEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}

func (e *regenerateSubjectEffect) Apply(ctx *EffectContext) error {
	if e.subject != nil {
		if p, ok := e.subject.resolvePermanent(ctx); ok {
			ctx.Game.AddRegenerationShield(p.ID())
		}
	}
	return nil
}

// --- from composable_regeneration_ai.go ---
// RegeneratesAttachment reports a direct regeneration instruction for the source's
// attachment. It follows ordered captures and snapshots without executing them.
// Unsupported instructions stop inspection. This query does not check costs,
// targets, attachment presence, or whether regeneration can prevent destruction.
func RegeneratesAttachment(ability ActivatedAbility) bool {
	attached := make(map[*PermanentBinding]bool)
	isAttached := func(subject PermanentRef) bool {
		switch ref := subject.(type) {
		case attachedPermanentRef:
			return true
		case savedPermanentRef:
			return attached[ref.binding]
		default:
			return false
		}
	}
	if action, ok := ability.(*ActionDefinition); ok {
		for _, capture := range action.dslCaptures {
			if capture, ok := capture.(activationPermanentCapture); ok {
				attached[capture.binding] = isAttached(capture.subject)
			}
		}
	}
	var inspect func(Effect) (found, supported bool)
	inspect = func(effect Effect) (bool, bool) {
		switch effect := effect.(type) {
		case *regenerateSubjectEffect:
			return isAttached(effect.subject), true
		case *permanentSnapshot:
			attached[effect.binding] = isAttached(effect.subject)
			return false, true
		case *sequenceEffect:
			for _, child := range effect.effects {
				found, supported := inspect(child)
				if found || !supported {
					return found, supported
				}
			}
			return false, true
		default:
			return false, false
		}
	}
	for _, effect := range ability.Effects() {
		found, supported := inspect(effect)
		if found || !supported {
			return found
		}
	}
	return false
}

// --- from composable_reveal_card.go ---
// PublicCardReveal records one card shown to all players.
type PublicCardReveal struct {
	CardID uuid.UUID
	Owner  uuid.UUID
	Name   string
	Types  []CardType
	Card   Card
}

func clonePublicReveals(in []PublicCardReveal) []PublicCardReveal {
	if len(in) == 0 {
		return nil
	}
	out := make([]PublicCardReveal, len(in))
	for i, reveal := range in {
		out[i] = reveal
		out[i].Types = slices.Clone(reveal.Types)
		if reveal.Card != nil {
			out[i].Card = reveal.Card.Copy()
			out[i].Card.SetID(reveal.CardID)
		}
	}
	return out
}

// PublicReveals returns public reveal history in reveal order.
func (g *Game) PublicReveals() []PublicCardReveal {
	return clonePublicReveals(g.publicReveals)
}

func (g *Game) recordPublicReveal(card Card) {
	if card == nil {
		return
	}
	copyCard := card.Copy()
	copyCard.SetID(card.ID())
	g.publicReveals = append(g.publicReveals, PublicCardReveal{
		CardID: card.ID(), Owner: card.Owner(), Name: card.Name(), Types: slices.Clone(card.Types()), Card: copyCard,
	})
}

type revealCardEffect struct{ card CardRef }

// RevealCard shows the referenced card to all players if it is still present.
func RevealCard(card CardRef) ResolvingEffect { return &revealCardEffect{card: card} }

func (*revealCardEffect) Text() string                 { return "reveal the referenced card" }
func (*revealCardEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *revealCardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
}
func (e *revealCardEffect) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if ok {
		ctx.Game.recordPublicReveal(card)
	}
	return nil
}

// --- from composable_source_counters.go ---
type sourceCounterValue struct{ counter CounterType }

// SourceCounters reads counters on the source incarnation at resolution.
// A departed source supplies its last count. An unknown source supplies zero.
func SourceCounters(counter CounterType) ResolutionValue {
	if counter < 0 || counter >= NumCounters {
		panic("SourceCounters requires a valid counter kind")
	}
	return sourceCounterValue{counter: counter}
}

func (v sourceCounterValue) Text() string {
	return v.counter.String() + " counters on source, with last-known information"
}

func (v sourceCounterValue) resolveValue(ctx *EffectContext) int {
	p := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID)
	if p != nil && (!ctx.sourceCaptured || p.incarnationID == ctx.sourceIncarnation) {
		return p.Counters[v.counter]
	}
	if ctx.sourceCaptured {
		return ctx.Game.zones.departedCounters[ctx.sourceIncarnation][v.counter]
	}
	return 0
}

func (v sourceCounterValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "source-counter-value", Text: v.Text(), battlefieldSource: true}
}

// --- from composable_tap_choice.go ---
// MayTapOrUntap lets the resolving controller change the subject's tapped state.
// The controller chooses at resolution. An absent subject has no effect.
func MayTapOrUntap(subject PermanentRef) ResolvingEffect {
	return &tapChoiceEffect{subject}
}

type tapChoiceEffect struct{ subject PermanentRef }

func (e *tapChoiceEffect) Text() string {
	return "you may tap or untap " + InspectDefinition(e.subject).Text
}
func (*tapChoiceEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *tapChoiceEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player == nil {
		return nil
	}
	tapped := p.Tapped
	verb := "tap "
	if tapped {
		verb = "untap "
	}
	if !player.ChooseMayAbility(verb + p.Name()) {
		return nil
	}
	if tapped {
		ctx.Game.UntapPermanent(p)
	} else {
		ctx.Game.TapPermanent(p)
	}
	return nil
}
func (e *tapChoiceEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "may-tap-or-untap", Text: e.Text(), children: []definitionEdge{
		{"player", You}, {"tap", TapSubject(e.subject)}, {"untap", UntapPermanent(e.subject)},
	}}
}

// --- from composable_attach_choice.go ---
// MayAttach offers a legal attachment move to the resolving controller.
func MayAttach(attachment, host PermanentRef) ResolvingEffect {
	return &mayAttachEffect{attachment: attachment, host: host}
}

type mayAttachEffect struct {
	attachment PermanentRef
	host       PermanentRef
}

func (e *mayAttachEffect) Text() string               { return "you may attach permanent to permanent" }
func (*mayAttachEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *mayAttachEffect) Apply(ctx *EffectContext) error {
	attachment, ok := e.attachment.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	host, ok := e.host.resolvePermanent(ctx)
	if !ok || !ctx.Game.canAttach(attachment, host) {
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player != nil && player.ChooseMayAbility("attach "+attachment.Name()+" to "+host.Name()) {
		ctx.Game.Attach(attachment.ID(), host.ID())
	}
	return nil
}
func (e *mayAttachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"attachment", e.attachment}, {"host", e.host}}}
}

// --- from composable_card_set_choice.go ---
type upToOneCardChoice struct {
	player   PlayerRef
	subjects CardSet
	filter   CardFilter
	binding  *CardBinding
}

// ChooseUpToOneCard saves an optional graveyard card choice at resolution.
func ChooseUpToOneCard(player PlayerRef, subjects CardSet, filter CardFilter, binding *CardBinding) ResolvingEffect {
	return &upToOneCardChoice{player: player, subjects: subjects, filter: filter, binding: binding}
}

func (*upToOneCardChoice) Text() string                 { return "choose up to one card" }
func (*upToOneCardChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *upToOneCardChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingWrite: e.binding,
		children: []definitionEdge{{"player", e.player}, {"cards", e.subjects}, {"binding", e.binding}}}
}
func (e *upToOneCardChoice) Apply(ctx *EffectContext) error {
	delete(ctx.Game.resolution.cardBindings, e.binding)
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	var candidates []Card
	visits := make(map[uuid.UUID]uuid.UUID)
	for _, card := range e.subjects.resolveCards(ctx) {
		if e.filter.Match(card) {
			candidates = append(candidates, card)
			visits[card.ID()] = ctx.Game.graveyardVisit(card.ID())
		}
	}
	if len(candidates) == 0 || !player.ChooseMayAbility("choose "+e.binding.name+" from graveyard") {
		return nil
	}
	chosen := player.ChooseCardFromGraveyard(candidates, "choose "+e.binding.name, ctx.Game)
	if chosen == nil {
		return nil
	}
	for _, card := range candidates {
		if card.ID() != chosen.ID() {
			continue
		}
		visit := visits[card.ID()]
		if visit == uuid.Nil || ctx.Game.graveyardVisit(card.ID()) != visit {
			return nil
		}
		if ctx.Game.resolution.cardBindings == nil {
			ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
		}
		ctx.Game.resolution.cardBindings[e.binding] = capturedCard{id: card.ID(), visit: visit, zone: ZoneGraveyard}
		return nil
	}
	return nil
}

// --- from composable_token.go ---
// TokenDefinition is an immutable token declaration.
type TokenDefinition interface {
	tokenDefinition()
	definitionInspector
}

// TokenOption sets one token characteristic.
type TokenOption interface {
	tokenOption()
	applyToken(*tokenDefinition)
}

type tokenDefinition struct {
	plan             *CardDefinitionPlan
	name             string
	types            []CardType
	subtypes         []string
	colors           []Color
	keywords         []Attr
	abilities        []Ability
	power, toughness int
	hasPT            bool
	problem          string
}

func (*tokenDefinition) tokenDefinition() {}

type tokenTypes []CardType

func (tokenTypes) tokenOption()                    {}
func (o tokenTypes) applyToken(d *tokenDefinition) { d.types = slices.Clone(o) }

// TokenTypes sets the token's card types.
func TokenTypes(types ...CardType) TokenOption { return tokenTypes(slices.Clone(types)) }

type tokenSubtypes []string

func (tokenSubtypes) tokenOption()                    {}
func (o tokenSubtypes) applyToken(d *tokenDefinition) { d.subtypes = slices.Clone(o) }

// TokenSubtypes sets the token's subtypes.
func TokenSubtypes(subtypes ...string) TokenOption { return tokenSubtypes(slices.Clone(subtypes)) }

type tokenColors []Color

func (tokenColors) tokenOption()                    {}
func (o tokenColors) applyToken(d *tokenDefinition) { d.colors = slices.Clone(o) }

// TokenColors sets the token's colors. An empty list makes it colorless.
func TokenColors(colors ...Color) TokenOption { return tokenColors(slices.Clone(colors)) }

type tokenKeywords []Attr

func (tokenKeywords) tokenOption()                    {}
func (o tokenKeywords) applyToken(d *tokenDefinition) { d.keywords = slices.Clone(o) }

// TokenKeywords sets the token's keywords.
func TokenKeywords(keywords ...Attr) TokenOption {
	return tokenKeywords(slices.Clone(keywords))
}

type tokenPT struct{ power, toughness int }

func (tokenPT) tokenOption() {}
func (o tokenPT) applyToken(d *tokenDefinition) {
	d.power, d.toughness, d.hasPT = o.power, o.toughness, true
}

// TokenPT sets the token's printed power and toughness.
func TokenPT(power, toughness int) TokenOption { return tokenPT{power, toughness} }

type tokenAbilities []Ability

func (tokenAbilities) tokenOption() {}
func (o tokenAbilities) applyToken(d *tokenDefinition) {
	for _, ability := range o {
		if nilDefinition(ability) {
			d.abilities = append(d.abilities, nil)
		} else {
			d.abilities = append(d.abilities, cloneTokenAbility(ability))
		}
	}
}

// TokenAbilities adds abilities to each created token.
func TokenAbilities(abilities ...Ability) TokenOption { return tokenAbilities(slices.Clone(abilities)) }

// DefineToken captures the token's characteristics and abilities.
func DefineToken(name string, options ...TokenOption) TokenDefinition {
	d := &tokenDefinition{name: name}
	seen := make(map[string]bool)
	for _, option := range options {
		if nilDefinition(option) {
			d.problem = "supply a non-nil token option"
			continue
		}
		key := fmt.Sprintf("%T", option)
		if seen[key] && key != "mage.tokenAbilities" {
			d.problem = "set each token characteristic once"
		}
		seen[key] = true
		option.applyToken(d)
	}
	return d
}

func (d *tokenDefinition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "token-definition", Text: d.name, problem: d.problem}
	if d.name == "" {
		n.problem = "supply a token name"
	}
	if len(d.types) == 0 {
		n.problem = "supply token types"
	}
	if slices.Contains(d.types, TypeCreature) != d.hasPT {
		n.problem = "set power and toughness exactly for creature tokens"
	}
	for _, color := range d.colors {
		if color <= Colorless || color >= AnyColor {
			n.problem = "supply valid token colors"
		}
	}
	for _, keyword := range d.keywords {
		if !IsKeywordAttr(keyword) {
			n.problem = "supply valid token keywords"
		}
	}
	for i, ability := range d.abilities {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("abilities[%d]", i), tokenAbilityDefinition{ability}})
	}
	return n
}

type tokenAbilityDefinition struct{ ability Ability }

func (d tokenAbilityDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "token-ability", bindingScope: true, children: []definitionEdge{{"definition", d.ability}}}
}

type createTokenCopiesEffect struct {
	player PlayerRef
	count  ResolutionValue
	token  TokenDefinition
}

type createOneTokenEffect struct {
	player PlayerRef
	token  TokenDefinition
	result *PermanentBinding
}

type createTokenCopiesIntoEffect struct {
	player PlayerRef
	count  ResolutionValue
	token  TokenDefinition
	result *CreatedTokensBinding
}

// CreateTokenCopiesInto saves the accepted incarnations from one token batch.
func CreateTokenCopiesInto(player PlayerRef, count ResolutionValue, token TokenDefinition, result *CreatedTokensBinding) ResolvingEffect {
	return &createTokenCopiesIntoEffect{player: player, count: count, token: token, result: result}
}

func (*createTokenCopiesIntoEffect) Text() string { return "create token copies into result" }
func (e *createTokenCopiesIntoEffect) Properties() EffectProperties {
	return (&createTokenCopiesEffect{token: e.token}).Properties()
}
func (e *createTokenCopiesIntoEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingWrite: e.result, children: []definitionEdge{{"player", e.player}, {"count", e.count}, {"token", e.token}, {"result", e.result}}}
}
func (e *createTokenCopiesIntoEffect) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.createdTokenBindings == nil {
		ctx.Game.resolution.createdTokenBindings = make(map[*CreatedTokensBinding][]capturedPermanent)
	}
	delete(ctx.Game.resolution.createdTokenBindings, e.result)
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	count := e.count.resolveValue(ctx)
	if count <= 0 {
		return nil
	}
	d := e.token.(*tokenDefinition)
	created, err := createDefinitionTokenBatch(ctx.Game, player, count, d)
	if err != nil {
		return err
	}
	for _, p := range created {
		ctx.Game.resolution.createdTokenBindings[e.result] = append(ctx.Game.resolution.createdTokenBindings[e.result], capturedPermanent{p.ID(), p.incarnationID})
	}
	return nil
}

// CreateOneToken saves the created token's battlefield incarnation.
func CreateOneToken(player PlayerRef, token TokenDefinition, result *PermanentBinding) ResolvingEffect {
	return &createOneTokenEffect{player: player, token: token, result: result}
}

func (*createOneTokenEffect) Text() string { return "create one token" }
func (e *createOneTokenEffect) Properties() EffectProperties {
	return (&createTokenCopiesEffect{token: e.token}).Properties()
}
func (e *createOneTokenEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingWrite: e.result, children: []definitionEdge{{"player", e.player}, {"token", e.token}, {"result", e.result}}}
}
func (e *createOneTokenEffect) Apply(ctx *EffectContext) error {
	delete(ctx.Game.resolution.permanentBindings, e.result)
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	d := e.token.(*tokenDefinition)
	created, err := createDefinitionTokenBatch(ctx.Game, player, 1, d)
	if err != nil {
		return err
	}
	if len(created) == 0 {
		return nil
	}
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	ctx.Game.resolution.permanentBindings[e.result] = capturedPermanent{created[0].ID(), created[0].incarnationID}
	return nil
}

// CreateTokenCopies creates the evaluated number of tokens for one player.
func CreateTokenCopies(player PlayerRef, count ResolutionValue, token TokenDefinition) ResolvingEffect {
	return &createTokenCopiesEffect{player, count, token}
}
func (e *createTokenCopiesEffect) Text() string { return "create token copies" }
func (e *createTokenCopiesEffect) Properties() EffectProperties {
	p := EffectProperties{Outcome: OutcomeBenefit}
	if d, ok := e.token.(*tokenDefinition); ok {
		p.TokenPower, p.TokenToughness = d.power, d.toughness
	}
	return p
}
func (e *createTokenCopiesEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"count", e.count}, {"token", e.token}}}
}
func (e *createTokenCopiesEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	count := e.count.resolveValue(ctx)
	if count <= 0 {
		return nil
	}
	d := e.token.(*tokenDefinition)
	_, err := createDefinitionTokenBatch(ctx.Game, player, count, d)
	return err
}

// --- from composable_stack.go ---
// StackObjectRef selects one stack object during resolution.
type StackObjectRef interface {
	stackObjectRef()
	resolveStackObject(*EffectContext) (*StackObject, bool)
}

// SpellBinding declares a spell seen by an event.
type SpellBinding struct{ name string }

// BindSpell declares a spell binding for a cast event.
func BindSpell(name string) *SpellBinding { return &SpellBinding{name: name} }

type savedSpellRef struct{ binding *SpellBinding }

// FromSpell reads the same stack object seen by the cast event.
func FromSpell(binding *SpellBinding) StackObjectRef { return savedSpellRef{binding: binding} }

func (savedSpellRef) stackObjectRef() {}
func (r savedSpellRef) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	id := ctx.Game.resolution.spellBindings[r.binding]
	obj := ctx.Game.stack.FindByID(id)
	return obj, obj != nil && !obj.IsAbility && obj.Card != nil
}
func (b *SpellBinding) bindingName() string { return b.name }
func (b *SpellBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "spell-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedSpellRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "stack-object-ref", Text: "cast spell", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

// SpellTargetBinding declares one spell target on the stack.
type SpellTargetBinding struct{ *targetDeclaration }

// DeclareSpellTarget declares a spell target with optional card filters.
func DeclareSpellTarget(name string, filters ...CardFilter) *SpellTargetBinding {
	spec := &typedSpellTarget{BaseTarget: BaseTarget{min: 1, max: 1}, filters: slices.Clone(filters)}
	return &SpellTargetBinding{&targetDeclaration{name: name, spec: spec}}
}

// DeclareSpellTargetTargetingControlledPermanent requires a current permanent
// target controlled by the player who chooses this spell target.
func DeclareSpellTargetTargetingControlledPermanent(name string, filters ...CardFilter) *SpellTargetBinding {
	spec := &typedSpellTarget{BaseTarget: BaseTarget{min: 1, max: 1}, filters: slices.Clone(filters), targetsControlledPermanent: true}
	return &SpellTargetBinding{&targetDeclaration{name: name, spec: spec}}
}

func (*SpellTargetBinding) stackObjectRef() {}
func (b *SpellTargetBinding) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	value, ok := firstTargetValue(ctx.targetBindings[b.targetDeclaration])
	if !ok {
		return nil, false
	}
	obj := ctx.Game.stack.FindByID(value.id)
	if _, union := b.spec.(*typedSpellOrPermanentTarget); union {
		return obj, value.incarnation == uuid.Nil && obj != nil && !obj.IsAbility && obj.Card != nil
	}
	if obj == nil {
		obj = ctx.Game.stack.FindBySourceID(value.id)
	}
	return obj, obj != nil && !obj.IsAbility && obj.Card != nil
}
func (b *SpellTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "stack-object-ref")
}

type spellControllerRef struct {
	spell   StackObjectRef
	withLKI bool
}

type spellManaValue struct {
	spell   StackObjectRef
	withLKI bool
}

// ManaValueOfSpell reads the referenced spell's mana value on the stack.
func ManaValueOfSpell(spell StackObjectRef) ResolutionValue { return spellManaValue{spell: spell} }

// ManaValueOfSpellWithLKI retains departure information because the effect can
// need the value after the spell leaves the stack.
func ManaValueOfSpellWithLKI(spell StackObjectRef) ResolutionValue {
	return spellManaValue{spell: spell, withLKI: true}
}
func (v spellManaValue) Text() string { return "mana value of spell" }
func (v spellManaValue) resolveValue(ctx *EffectContext) int {
	spell, ok := v.spell.resolveStackObject(ctx)
	if !ok {
		if !v.withLKI {
			return 0
		}
		id := spellReferenceID(v.spell, ctx)
		return ctx.Game.stack.spellManaValues[id]
	}
	return stackSpellManaValue(spell)
}
func (v spellManaValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"spell", v.spell}}}
}

func spellReferenceID(spell StackObjectRef, ctx *EffectContext) uuid.UUID {
	switch ref := spell.(type) {
	case thisSpellRef:
		return ctx.Game.resolution.resolvingStackID
	case savedSpellRef:
		return ctx.Game.resolution.spellBindings[ref.binding]
	case *SpellTargetBinding:
		if value, found := firstTargetValue(ctx.targetBindings[ref.targetDeclaration]); found {
			return value.id
		}
	}
	return uuid.Nil
}

// ControllerOfSpellWithLKI reads the controller saved when the spell left the stack.
func ControllerOfSpellWithLKI(spell StackObjectRef) PlayerRef {
	return spellControllerRef{spell: spell, withLKI: true}
}

// ControllerOfSpell reads the current controller of the referenced spell.
func ControllerOfSpell(spell StackObjectRef) PlayerRef { return spellControllerRef{spell: spell} }
func (spellControllerRef) playerRef()                  {}
func (spellControllerRef) damageRecipient()            {}
func (r spellControllerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	spell, ok := r.spell.resolveStackObject(ctx)
	if !ok {
		if r.withLKI {
			id := ctx.Game.stack.spellControllers[spellReferenceID(r.spell, ctx)]
			return id, ctx.Game.GetPlayer(id) != nil
		}
		return uuid.Nil, false
	}
	return spell.Controller, ctx.Game.GetPlayer(spell.Controller) != nil
}
func (r spellControllerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r spellControllerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "spell controller", children: []definitionEdge{{"spell", r.spell}}}
}

type typedSpellTarget struct {
	contextualFilter SpellFilter
	filterCount      int
	manaValueEqualX  bool
	BaseTarget
	filters                    []CardFilter
	targetsControlledPermanent bool
}

func (t *typedSpellTarget) Possible(controller uuid.UUID, source Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, obj := range g.stack.Objects() {
		if obj.IsAbility || obj.Card == nil {
			continue
		}
		if t.filterCount > 0 && !t.contextualFilter.matches(g, controller, source, obj) {
			continue
		}
		if t.manaValueEqualX && stackSpellManaValue(obj) != g.resolution.X() {
			continue
		}
		if t.targetsControlledPermanent && !obj.targetsPermanentControlledBy(g, controller) {
			continue
		}
		matches := true
		for _, filter := range t.filters {
			if !filter.Match(obj.Card) {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, obj.ID)
		}
	}
	return result
}

func (obj *StackObject) targetsPermanentControlledBy(g *Game, controller uuid.UUID) bool {
	for _, id := range obj.Targets {
		if obj.TargetZones[id] != ZoneBattlefield {
			continue
		}
		p := g.FindPermanent(id)
		// CR 115.9b needs current information, even if the original target is illegal.
		if p != nil && !p.PhasedOut && p.incarnationID == obj.targetIncarnations[id] && p.ControllerID() == controller {
			return true
		}
	}
	return false
}

func (t *typedSpellTarget) Choose(uuid.UUID, Card, *Game, []uuid.UUID) error { return nil }

type counterReferencedEffect struct {
	spell   StackObjectRef
	ability bool
}

// Counter counters the referenced spell if it is still on the stack.
func Counter(spell StackObjectRef) ResolvingEffect { return &counterReferencedEffect{spell: spell} }
func (e *counterReferencedEffect) Text() string {
	if e.ability {
		return "counter target activated ability"
	}
	return "counter target spell"
}
func (*counterReferencedEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *counterReferencedEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"spell", e.spell}}}
}
func (e *counterReferencedEffect) Apply(ctx *EffectContext) error {
	obj, ok := e.spell.resolveStackObject(ctx)
	if ok {
		ctx.Game.CounterStackObject(obj.ID)
	}
	return nil
}
