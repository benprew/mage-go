package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CardRef identifies one card in a nonbattlefield zone.
type CardRef interface {
	cardRef()
	resolveCard(*EffectContext) (Card, bool)
}

// CardTargetBinding declares one graveyard card target.
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
	if config.manaValueOptions > 0 {
		spec = &manaValueBoundGraveyardTarget{inner: spec, maximum: config.manaValue, optionCount: config.manaValueOptions}
	}
	return &CardTargetBinding{&targetDeclaration{name: name, spec: spec}}
}

func (*CardTargetBinding) cardRef() {}
func (b *CardTargetBinding) resolveCard(ctx *EffectContext) (Card, bool) {
	value, ok := firstTargetValue(ctx.targetBindings[b.targetDeclaration])
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
	return (&returnFromGraveyardToHandTargetEffect{}).Properties()
}
func (e *returnCardToHandEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
}
func (e *returnCardToHandEffect) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	owner := ctx.Game.GetPlayer(card.Owner())
	if owner == nil {
		return nil
	}
	moved, _, ok := ctx.Game.MoveFromAnyGraveyard(card.ID(), core.ZoneHand)
	if ok {
		owner.AddToHand(moved)
	}
	return nil
}

type returnCardToBattlefieldEffect struct {
	card       CardRef
	controller PlayerRef
}

// ReturnCardToBattlefield puts the referenced graveyard card onto the battlefield.
func ReturnCardToBattlefield(card CardRef, controller PlayerRef) ResolvingEffect {
	return &returnCardToBattlefieldEffect{card: card, controller: controller}
}
func (*returnCardToBattlefieldEffect) Text() string {
	return "return card from graveyard to the battlefield"
}
func (*returnCardToBattlefieldEffect) Properties() EffectProperties {
	return (&returnFromAnyGraveyardEffect{}).Properties()
}
func (e *returnCardToBattlefieldEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}, {"controller", e.controller}}}
}
func (e *returnCardToBattlefieldEffect) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	controller, ok := e.controller.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	returnGraveyardCardToBattlefield(ctx.Game, card.ID(), controller)
	return nil
}
