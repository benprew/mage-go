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
func DrawFor(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &drawSubjectEffect{subject, amount}
}

type drawSubjectEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

func (e *drawSubjectEffect) Text() string                 { return "draw for " + InspectDefinition(e.subject).Text }
func (e *drawSubjectEffect) Properties() EffectProperties { return drawAmountProperties(e.amount) }
func (e *drawSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *drawSubjectEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return DrawCards(Fixed(amount)).Apply(&child)
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

type destroyEachEffect struct{ subjects PermanentSet }

// DestroyEach performs one simultaneous destruction group through the engine.
func DestroyEach(subjects PermanentSet) ResolvingEffect { return &destroyEachEffect{subjects} }
func (*destroyEachEffect) Text() string                 { return "destroy each permanent" }
func (*destroyEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Destroys: true, Mass: true}
}
func (e *destroyEachEffect) Apply(ctx *EffectContext) error {
	ctx.Game.DestroyPermanents(e.subjects.resolvePermanents(ctx))
	return nil
}
func (e *destroyEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "destroy-each", children: []definitionEdge{{"subjects", e.subjects}}}
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
func TapSubject(subject PermanentRef) ResolvingEffect { return &tapSubjectEffect{subject} }

type tapSubjectEffect struct{ subject PermanentRef }

func (*tapSubjectEffect) Text() string { return "tap permanent" }
func (*tapSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Taps: true}
}
func (e *tapSubjectEffect) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.TapPermanent(p)
	}
	return nil
}
func (e *tapSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
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
