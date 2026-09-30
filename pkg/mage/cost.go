package mage

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Cost represents a cost to pay for a spell or ability.
type Cost interface {
	CanPay(sourceID uuid.UUID, controller uuid.UUID, g *Game) bool
	Pay(sourceID uuid.UUID, controller uuid.UUID, g *Game) error
	Text() string
}

// ManaCostPayment wraps a ManaCost as a Cost.
type ManaCostPayment struct {
	MC ManaCost
}

// ManaCostOf creates a cost that requires paying the given mana cost string (e.g. "{1}{R}").
func ManaCostOf(s string) Cost {
	return &ManaCostPayment{MC: ParseManaCost(s)}
}

// GenericCost creates a cost that requires paying n generic mana.
func GenericCost(n int) Cost {
	return &ManaCostPayment{MC: ManaCost{Generic: n}}
}

type actionCostContext struct {
	Targets []uuid.UUID
	XValue  int
}

type contextualActionCost interface {
	Cost
	resolveActionCost(actionCostContext) Cost
}

type manaCostPerTarget struct {
	perTarget ManaCost
}

// ManaCostPerTarget creates a contextual mana cost multiplied by the number
// of targets chosen for the spell or ability. Target identities are acquired
// before this cost is evaluated and paid.
func ManaCostPerTarget(cost string) Cost {
	return &manaCostPerTarget{perTarget: ParseManaCost(cost)}
}

func (c *manaCostPerTarget) CanPay(uuid.UUID, uuid.UUID, *Game) bool { return true }
func (c *manaCostPerTarget) Pay(uuid.UUID, uuid.UUID, *Game) error {
	return fmt.Errorf("per-target mana cost requires action context")
}
func (c *manaCostPerTarget) Text() string { return c.perTarget.String() + " for each target" }
func (c *manaCostPerTarget) resolveActionCost(ctx actionCostContext) Cost {
	return &ManaCostPayment{MC: multiplyManaCost(c.perTarget, len(ctx.Targets))}
}

func multiplyManaCost(mc ManaCost, n int) ManaCost {
	if n <= 0 {
		return ManaCost{}
	}
	out := ManaCost{
		Generic: mc.Generic * n,
		White:   mc.White * n,
		Blue:    mc.Blue * n,
		Black:   mc.Black * n,
		Red:     mc.Red * n,
		Green:   mc.Green * n,
		HasX:    mc.HasX,
		XCount:  mc.XCount * n,
	}
	for range n {
		out.Hybrid = append(out.Hybrid, mc.Hybrid...)
	}
	return out
}

func (c *ManaCostPayment) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	mc := c.reducedCost(sourceID, g)
	// Spell casting extracts this cost into the locked total-mana transaction;
	// this standalone path is used by activated and other non-spell actions.
	return g.CanAfford(controller, mc, nil)
}

func (c *ManaCostPayment) Pay(sourceID, controller uuid.UUID, g *Game) error {
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller:               controller,
		SourceID:                 sourceID,
		Costs:                    []Cost{c},
		ApplyActivationReduction: true,
	})
	if err != nil {
		return err
	}
	return tx.Commit()
}

// reducedCost applies activation cost reductions (e.g. Power Artifact) to the mana cost.
// The reduction lowers the generic component but can't reduce total mana below 1.
func (c *ManaCostPayment) reducedCost(sourceID uuid.UUID, g *Game) ManaCost {
	reduction := g.effects.Rules.ActivationCostReductions[sourceID]
	if reduction <= 0 {
		return c.MC
	}
	mc := c.MC
	// Total colored mana in the cost
	coloredTotal := mc.White + mc.Blue + mc.Black + mc.Red + mc.Green + len(mc.Hybrid)
	total := coloredTotal + mc.Generic
	// Can't reduce below 1 total mana
	minGeneric := 0
	if total > 0 {
		minGeneric = max(0, 1-coloredTotal)
	}
	mc.Generic = max(minGeneric, mc.Generic-reduction)
	return mc
}

func (c *ManaCostPayment) Text() string {
	return c.MC.String()
}

// tap implements both Cost and Effect. As a cost it taps the source
// permanent ({T}). As an effect it taps ctx.Targets[0]. Single DSL constructor
// (Tap) — the call site (cost slot vs effect slot) selects the path.
type tap struct{}

// Tap creates a tap operation usable as a Cost (taps source) or Effect (taps
// target). The call site (cost slot vs effect slot) selects the path.
func Tap() *tap { return &tap{} }

// --- Cost interface ---

func (c *tap) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && !p.Tapped && p.CanTapForEffect(g)
}

func (c *tap) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return ErrSourceNotFound
	}
	if p.Tapped {
		return ErrSourceTapped
	}
	g.TapPermanent(p)
	return nil
}

// --- Effect / Cost shared ---

// Text returns the cost-style symbol "{T}". This is what the eval heuristic
// (AbilityQuality) and rules-text rendering expect for tap costs. When used
// as an effect, the rendered string is the same — slightly less descriptive
// than "tap target permanent" but consistent.
func (c *tap) Text() string { return "{T}" }

// --- Effect interface ---

func (c *tap) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Taps: true}
}

// removeCountersCost requires removing counters from the source.
type removeCountersCost struct {
	zeroWithoutSource bool
	ct                CounterType
	amount            int
	paid              *NumberBinding
	capture           bool
	incarnation       uuid.UUID
	locked            bool
}

// RemoveCountersCost creates a cost that requires removing n counters of the given type from the source.
func RemoveCountersCost(ct CounterType, n int) Cost {
	return &removeCountersCost{ct: ct, amount: n}
}

// RemoveCountersCostInto saves the number paid by one activation.
func RemoveCountersCostInto(ct CounterType, n int, paid *NumberBinding) Cost {
	return &removeCountersCost{ct: ct, amount: n, paid: paid, capture: true}
}

func (c *removeCountersCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", battlefieldSource: c.capture}
	if c.ct < 0 || c.ct >= NumCounters {
		n.problem = "invalid counter type"
	}
	if c.amount < 0 {
		n.problem = "counter count cannot be negative"
	}
	if c.capture {
		n.bindingWrite = c.paid
		n.children = []definitionEdge{{"paid", c.paid}}
		if c.paid == nil {
			n.problem = "supply a non-nil paid counter binding"
		}
		if c.amount < 0 {
			n.problem = "paid counter count cannot be negative"
		}
	}
	return n
}

func (c *removeCountersCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	if c.ct < 0 || c.ct >= NumCounters {
		return false
	}
	if c.zeroWithoutSource && c.amount == 0 {
		return true
	}
	p := g.FindPermanent(sourceID)
	return p != nil && (!c.locked || p.incarnationID == c.incarnation) && c.amount >= 0 && int(p.Counters[c.ct]) >= c.amount
}

func (c *removeCountersCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay counter cost")
	}
	if c.zeroWithoutSource && c.amount == 0 {
		return nil
	}
	p := g.MutablePermanent(sourceID)
	if p == nil {
		return ErrSourceNotFound
	}
	if !p.RemoveCounter(c.ct, c.amount) {
		return fmt.Errorf("not enough %s counters", c.ct)
	}
	return nil
}

func (c *removeCountersCost) Text() string {
	return fmt.Sprintf("Remove %d %s counter(s)", c.amount, c.ct)
}

// removeXCountersCost requires removing X counters of the given type from
// the source, where X is the current activation X (g.currentX). Used by
// abilities like Chamber Sentry: "{X}, {T}, Remove X +1/+1 counters from
// this creature: …".
type removeXCountersCost struct {
	ct CounterType
}

type capturedXCountersCost struct {
	ct   CounterType
	paid *NumberBinding
}

// RemoveXCountersFromSourceCostInto saves the counters paid for the announced X.
func RemoveXCountersFromSourceCostInto(ct CounterType, paid *NumberBinding) Cost {
	return &capturedXCountersCost{ct: ct, paid: paid}
}

func (c *capturedXCountersCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", Text: c.Text(), battlefieldSource: true,
		bindingWrite: c.paid, children: []definitionEdge{{"paid", c.paid}}}
	if c.paid == nil {
		n.problem = "supply a non-nil paid counter binding"
	}
	if c.ct < 0 || c.ct >= NumCounters {
		n.problem = "invalid counter type"
	}
	return n
}

func (c *capturedXCountersCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return (&removeCountersCost{ct: c.ct, amount: g.resolution.X()}).CanPay(sourceID, controller, g)
}

func (c *capturedXCountersCost) Pay(uuid.UUID, uuid.UUID, *Game) error {
	return fmt.Errorf("X counter payment requires an announced action")
}

func (c *capturedXCountersCost) Text() string {
	return fmt.Sprintf("Remove X %s counters", c.ct)
}

func (c *capturedXCountersCost) resolveActionCost(ctx actionCostContext) Cost {
	return &removeCountersCost{ct: c.ct, amount: ctx.XValue, paid: c.paid, capture: true}
}

// RemoveXCountersFromSourceCost creates a cost that removes X counters of
// the given type from the source permanent, where X is the activation's
// X value (set via ActivateAbilityWithX). When X = 0, no counters are
// removed and the cost is trivially payable.
func RemoveXCountersFromSourceCost(ct CounterType) Cost {
	return &removeXCountersCost{ct: ct}
}

func (c *removeXCountersCost) resolveActionCost(ctx actionCostContext) Cost {
	return &removeCountersCost{ct: c.ct, amount: ctx.XValue, zeroWithoutSource: true}
}

func (c *removeXCountersCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", Text: c.Text()}
	if c.ct < 0 || c.ct >= NumCounters {
		n.problem = "invalid counter type"
	}
	return n
}

func (c *removeXCountersCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	if c.ct < 0 || c.ct >= NumCounters {
		return false
	}
	x := g.resolution.X()
	if x <= 0 {
		return true
	}
	p := g.FindPermanent(sourceID)
	return p != nil && int(p.Counters[c.ct]) >= x
}

func (c *removeXCountersCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay X counter cost")
	}
	x := g.resolution.X()
	if x <= 0 {
		return nil
	}
	p := g.MutablePermanent(sourceID)
	if p == nil {
		return ErrSourceNotFound
	}
	if !p.RemoveCounter(c.ct, x) {
		return fmt.Errorf("not enough %s counters to remove %d", c.ct, x)
	}
	return nil
}

func (c *removeXCountersCost) Text() string {
	return fmt.Sprintf("Remove X %s counter(s)", c.ct)
}

// requireCountersCost is a gate cost that checks for N+ counters but does not remove them.
type requireCountersCost struct {
	ct     CounterType
	amount int
}

// RequireCountersCost creates a cost that requires the source to have at least n counters
// of the given type. The counters are NOT removed when the cost is paid.
func RequireCountersCost(ct CounterType, n int) Cost {
	return &requireCountersCost{ct: ct, amount: n}
}

func (c *requireCountersCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && int(p.Counters[c.ct]) >= c.amount
}

func (c *requireCountersCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	// Gate cost only — counters are not removed
	return nil
}

func (c *requireCountersCost) Text() string {
	return fmt.Sprintf("Requires %d+ %s counter(s)", c.amount, c.ct)
}

// lifePayCost requires paying life.
type lifePayCost struct {
	amount  int
	paid    *NumberBinding
	capture bool
}

// LifePayCost creates a cost that requires the controller to pay life.
func LifePayCost(amount int) Cost {
	return &lifePayCost{amount: amount}
}

// LifePayCostInto saves the amount paid by this cost for its resolving ability.
func LifePayCostInto(amount int, paid *NumberBinding) Cost {
	return &lifePayCost{amount: amount, paid: paid, capture: true}
}

type xLifePayCost struct{ paid *NumberBinding }

// LifePayXCostInto pays the announced X life and saves the committed amount.
func LifePayXCostInto(paid *NumberBinding) Cost { return &xLifePayCost{paid: paid} }

func (c *xLifePayCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "life-payment-cost", Text: c.Text(), bindingWrite: c.paid,
		children: []definitionEdge{{"paid", c.paid}}}
	if c.paid == nil {
		n.problem = "supply a non-nil paid life binding"
	}
	return n
}

func (c *xLifePayCost) CanPay(_, controller uuid.UUID, g *Game) bool {
	return (&lifePayCost{amount: g.resolution.X()}).CanPay(uuid.Nil, controller, g)
}
func (c *xLifePayCost) Pay(uuid.UUID, uuid.UUID, *Game) error {
	return fmt.Errorf("X life payment requires an announced action")
}
func (c *xLifePayCost) Text() string { return "Pay X life" }
func (c *xLifePayCost) resolveActionCost(ctx actionCostContext) Cost {
	return &lifePayCost{amount: ctx.XValue, paid: c.paid, capture: true}
}

func (c *lifePayCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "life-payment-cost", Text: c.Text()}
	if c.capture {
		n.bindingWrite = c.paid
		n.children = []definitionEdge{{"paid", c.paid}}
		if c.paid == nil {
			n.problem = "supply a non-nil paid life binding"
		}
	}
	if c.amount < 0 {
		n.problem = "negative life payment"
	}
	return n
}

// LifePaymentAmount reports the life required by a life-payment cost, or zero
// for costs that do not pay life.
func LifePaymentAmount(cost Cost) int {
	if lifeCost, ok := cost.(*lifePayCost); ok {
		return lifeCost.amount
	}
	return 0
}

func (c *lifePayCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	return p != nil && c.amount >= 0 && (c.amount == 0 || p.Life() >= c.amount)
}

func (c *lifePayCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	g.PlayerLoseLife(p, c.amount)
	return nil
}

func (c *lifePayCost) Text() string {
	return fmt.Sprintf("Pay %d life", c.amount)
}

// tapMatchingCost requires tapping an untapped permanent you control matching a filter.
type tapMatchingCost struct {
	filter PermanentFilter
	text   string
}

// TapMatchingCost creates a cost that requires tapping an untapped permanent you control
// (other than the source) that matches the given filter.
func TapMatchingCost(filter PermanentFilter, text string) Cost {
	return &tapMatchingCost{filter: filter, text: text}
}

// TapCreatureCost creates a cost that requires tapping an untapped creature you control
// (other than the source). Used by convoke-like abilities and tap-creature costs.
func TapCreatureCost() Cost {
	return TapMatchingCost(IsCreature, "Tap an untapped creature you control")
}

func (c *tapMatchingCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == controller && p.ID() != sourceID && !p.Tapped && p.CanTapForEffect(g) && c.filter.Match(p, g) {
			return true
		}
	}
	return false
}

func (c *tapMatchingCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == controller && p.ID() != sourceID && !p.Tapped && p.CanTapForEffect(g) && c.filter.Match(p, g) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no permanent to tap")
	}
	player := g.GetPlayer(controller)
	chosen := player.ChoosePermanent(candidates, c.text, g)
	if chosen == nil {
		return fmt.Errorf("no permanent chosen")
	}
	g.TapPermanent(chosen)
	return nil
}

func (c *tapMatchingCost) Text() string { return c.text }

// discardCost requires discarding cards from hand.
type discardCost struct {
	amount         int
	differentNames bool
	filter         CardFilter
	selected       []capturedCard
	locked         bool
	paidCard       *CardBinding
	captureCard    bool
	paidCards      *CardsBinding
	captureCards   bool
	results        []capturedCard
}

// DiscardCost creates a cost that requires the controller to discard n cards.
func DiscardCost(n int) Cost {
	return &discardCost{amount: n}
}

// DiscardCardCostInto saves one discarded card during its graveyard visit.
func DiscardCardCostInto(card *CardBinding) Cost {
	return &discardCost{amount: 1, paidCard: card, captureCard: true}
}

// DiscardCostWithCards saves the new graveyard visits for this payment.
func DiscardCostWithCards(n int, paid *CardsBinding) Cost {
	return &discardCost{amount: n, paidCards: paid, captureCards: true}
}

// DiscardMatchingCardCost requires one hand card that matches the filter.
func DiscardMatchingCardCost(filter CardFilter) Cost {
	return &discardCost{amount: 1, filter: filter}
}

// DiscardMatchingCardCostInto must not follow the card through later zone changes.
func DiscardMatchingCardCostInto(filter CardFilter, paid *CardBinding) Cost {
	return &discardCost{amount: 1, filter: filter, paidCard: paid, captureCard: true}
}

func (c *discardCost) choose(p Player, g *Game) []Card {
	if c.amount == 0 {
		return nil
	}
	if c.filter.IsZero() {
		return p.ChooseCardsFromHand(c.amount, c.Text(), g)
	}
	var candidates []Card
	for _, card := range p.Hand() {
		if c.filter.Match(card) {
			candidates = append(candidates, card)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	chosen := p.ChooseCardFromHand(candidates, c.Text(), g)
	if chosen == nil {
		return nil
	}
	return []Card{chosen}
}

func (c *discardCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "discard-cost", Text: c.Text()}
	if c.amount < 0 {
		n.problem = "supply a nonnegative discard count"
	}
	if c.captureCard {
		if c.paidCard == nil {
			n.problem = "supply a non-nil paid card binding"
		}
		if c.amount != 1 {
			n.problem = "paid card requires one discard"
		}
		n.bindingWrites = []savedBinding{c.paidCard}
		n.children = []definitionEdge{{"card", c.paidCard}}
	}
	if c.captureCards {
		if c.paidCards == nil {
			n.problem = "supply a non-nil paid cards binding"
		}
		n.bindingWrites = append(n.bindingWrites, c.paidCards)
		n.children = append(n.children, definitionEdge{"paid cards", c.paidCards})
	}
	return n
}

func (c *discardCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	if p == nil || c.amount < 0 || len(p.Hand()) < c.amount {
		return false
	}
	if !c.locked {
		if !c.filter.IsZero() {
			return slices.ContainsFunc(p.Hand(), c.filter.Match)
		}
		if c.differentNames {
			names := make(map[string]bool)
			for _, card := range p.Hand() {
				names[card.Name()] = true
			}
			return len(names) >= c.amount
		}
		return true
	}
	if len(c.selected) != c.amount {
		return false
	}
	available := make(map[uuid.UUID]Card, len(p.Hand()))
	for _, card := range p.Hand() {
		available[card.ID()] = card
	}
	visitor, ok := p.(interface{ handVisit(uuid.UUID) uuid.UUID })
	if len(c.selected) > 0 && !ok {
		return false
	}
	names := make(map[string]bool, len(c.selected))
	for _, card := range c.selected {
		current := available[card.id]
		if current == nil || card.visit == uuid.Nil || visitor.handVisit(card.id) != card.visit {
			return false
		}
		if !c.filter.Match(current) {
			return false
		}
		if c.differentNames && names[current.Name()] {
			return false
		}
		names[current.Name()] = true
		delete(available, card.id)
	}
	return true
}

func (c *discardCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	c.results = nil
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay chosen discard")
	}
	var selected []uuid.UUID
	if c.locked {
		for _, card := range c.selected {
			selected = append(selected, card.id)
		}
	} else {
		var chosen []Card
		if c.amount > 0 {
			chosen = c.choose(p, g)
		}
		if len(chosen) != c.amount {
			return fmt.Errorf("cannot pay chosen discard")
		}
		locked := *c
		locked.locked = true
		locked.selected = nil
		visitor, ok := p.(interface{ handVisit(uuid.UUID) uuid.UUID })
		if len(chosen) > 0 && !ok {
			return fmt.Errorf("cannot pay chosen discard")
		}
		for _, card := range chosen {
			if card == nil {
				return fmt.Errorf("cannot pay chosen discard")
			}
			locked.selected = append(locked.selected, capturedCard{id: card.ID(), visit: visitor.handVisit(card.ID()), zone: ZoneHand})
			selected = append(selected, card.ID())
		}
		if !locked.CanPay(sourceID, controller, g) {
			return fmt.Errorf("cannot pay chosen discard")
		}
	}
	var actions []*DiscardAction
	for _, id := range selected {
		action := newCostDiscardAction(sourceID, controller, id)
		action.requiresCharacteristics = c.differentNames || !c.filter.IsZero()
		actions = append(actions, action)
	}
	events := g.discardActionsTogetherWithEvents(actions, true)
	if len(events) != c.amount {
		return fmt.Errorf("cannot pay chosen discard")
	}
	if c.captureCards {
		for _, event := range events {
			if event.DiscardedCardVisit != uuid.Nil {
				c.results = append(c.results, capturedCard{id: event.SourceID, visit: event.DiscardedCardVisit, zone: ZoneGraveyard})
			}
		}
	}
	return nil
}

func (c *discardCost) Text() string {
	if !c.filter.IsZero() {
		return "Discard one " + c.filter.Text()
	}
	if c.amount == 1 {
		return "Discard a card"
	}
	if c.differentNames {
		return fmt.Sprintf("Discard %d cards with different names", c.amount)
	}
	return fmt.Sprintf("Discard %d cards", c.amount)
}

// DiscardCardsWithDifferentNamesCost requires distinct names for the payment.
func DiscardCardsWithDifferentNamesCost(n int) Cost {
	return &discardCost{amount: n, differentNames: true}
}

// DiscardCardsWithDifferentNamesCostWithCards saves the new graveyard visits.
func DiscardCardsWithDifferentNamesCostWithCards(n int, paid *CardsBinding) Cost {
	return &discardCost{amount: n, differentNames: true, paidCards: paid, captureCards: true}
}

// revealFromHandCost is an additional cost that requires the controller
// to reveal a card from their hand matching the given filter. The card
// stays in hand after being revealed (CR 701.16) — no zone change.
//
// Used by "as an additional cost, reveal an X card from your hand"
// (Wren's Run Vanquisher's first OR branch, etc.).
type revealFromHandCost struct {
	filter   CardFilter
	label    string
	paidCard *CardBinding
	capture  bool
	selected uuid.UUID
	visit    uuid.UUID
	locked   bool
}

// RevealFromHandCost creates a cost that requires revealing a card from
// the controller's hand that matches filter. Combine with [EitherCost]
// to encode "reveal X or pay {N}" patterns.
//
// Example: Wren's Run Vanquisher — "As an additional cost to cast this
// spell, reveal an Elf card from your hand or pay {3}":
//
//	WithAdditionalCost(EitherCost(
//	    RevealFromHandCost(HasSubTypeCardFilter("Elf"), "Reveal an Elf card"),
//	    ManaCostOf("{3}"),
//	))
func RevealFromHandCost(filter CardFilter, label string) Cost {
	return &revealFromHandCost{filter: filter, label: label}
}

// RevealFromHandCostWithCard saves the revealed card for this payment.
func RevealFromHandCostWithCard(filter CardFilter, label string, card *CardBinding) Cost {
	return &revealFromHandCost{filter: filter, label: label, paidCard: card, capture: true}
}

func (c *revealFromHandCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "reveal-cost", Text: c.Text()}
	if c.capture {
		if c.paidCard == nil {
			n.problem = "supply a non-nil paid card binding"
		}
		if c.label == "" {
			n.problem = "supply cost text"
		}
		n.bindingWrites = []savedBinding{c.paidCard}
		n.children = []definitionEdge{{"card", c.paidCard}}
	}
	return n
}

func (c *revealFromHandCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	if p == nil {
		return false
	}
	if c.locked {
		visitor, ok := p.(interface{ handVisit(uuid.UUID) uuid.UUID })
		if !ok || c.visit == uuid.Nil || visitor.handVisit(c.selected) != c.visit {
			return false
		}
	}
	for _, card := range p.Hand() {
		// Skip the card currently being cast (it is no longer in hand
		// by the time costs are paid, but defensively skip).
		if card.ID() == sourceID {
			continue
		}
		if (c.filter.IsZero() || c.filter.Match(card)) && (!c.locked || card.ID() == c.selected) {
			return true
		}
	}
	return false
}

func (c *revealFromHandCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	var matches []Card
	for _, card := range p.Hand() {
		if card.ID() == sourceID {
			continue
		}
		if c.filter.IsZero() || c.filter.Match(card) {
			matches = append(matches, card)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("no matching card to reveal")
	}
	if c.locked {
		if !c.CanPay(sourceID, controller, g) {
			return fmt.Errorf("chosen reveal card is absent")
		}
		for _, card := range matches {
			if card.ID() == c.selected {
				g.resolution.SetLastCostReveal(card)
				g.recordPublicReveal(card)
				return nil
			}
		}
	}
	// Use ChooseCardsFromHand to pick one. For TestPlayer this routes
	// through scripted choices; for BasePlayer it returns the first.
	// We delegate the picking by passing the matches as if they were a
	// standalone hand via ChooseFromCards on the player when available;
	// otherwise pick the first match.
	chosen := matches[0]
	if cp, ok := p.(interface {
		ChooseFromCards(cards []Card, reason string) Card
	}); ok {
		if c2 := cp.ChooseFromCards(matches, c.label); c2 != nil {
			chosen = c2
		}
	}
	// Reveal the card; no zone change (CR 701.16). The card stays in
	// hand. Engine clients can observe the reveal via [Game.LastCostReveal].
	g.resolution.SetLastCostReveal(chosen)
	g.recordPublicReveal(chosen)
	return nil
}

// LastCostReveal returns the most recently revealed card from a
// [RevealFromHandCost] payment, or nil if none.
func (g *Game) LastCostReveal() Card { return g.resolution.LastCostReveal() }

func (c *revealFromHandCost) Text() string {
	if c.label != "" {
		return c.label
	}
	return "Reveal a card from your hand"
}

// eitherCost is a branching additional cost: the controller chooses one of
// several alternatives at pay time. Used for "as an additional cost,
// discard a card or pay {N}" and similar OR-style additional costs.
type eitherCost struct {
	options  []Cost
	locked   bool
	selected int
}

// EitherCost creates a branching cost that lets the controller pick which
// underlying cost to pay (CR 118.1 — "or" in cost text). All options are
// tried during CanPay; payment routes to the option chosen by the player
// via ChooseMode. If only one option is payable, that one is selected
// automatically.
//
// Example: "As an additional cost, discard a card or pay {5}." →
// EitherCost(DiscardCost(1), ManaCostOf("{5}")).
func EitherCost(options ...Cost) Cost {
	return &eitherCost{options: options}
}

func (c *eitherCost) resolveActionCost(ctx actionCostContext) Cost {
	resolved := *c
	resolved.options = append([]Cost(nil), c.options...)
	for i, option := range resolved.options {
		switch nested := option.(type) {
		case *removeXCountersCost:
			resolved.options[i] = nested.resolveActionCost(ctx)
		case *eitherCost:
			resolved.options[i] = nested.resolveActionCost(ctx)
		case *optionalCost:
			resolved.options[i] = nested.resolveActionCost(ctx)
		}
	}
	return &resolved
}

func (c *eitherCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "alternative-cost", Text: c.Text()}
	if len(c.options) == 0 {
		n.problem = "supply at least one cost option"
	}
	for i, option := range c.options {
		if mill, ok := option.(*millCost); ok && mill.capture {
			n.problem = "paid mill count requires a direct cost"
		}
		if discard, ok := option.(*discardCost); ok && (discard.captureCards || discard.captureCard) {
			n.problem = "paid discard cards require a direct cost"
		}
		if exile, ok := option.(*exileFromGraveyardCost); ok && exile.paidRequired {
			n.problem = "paid graveyard exile cards require a direct cost"
		}
		n.children = append(n.children, definitionEdge{fmt.Sprintf("options[%d]", i), option})
	}
	return n
}

func (c *eitherCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	if c.locked {
		return c.selected >= 0 && c.selected < len(c.options) && c.options[c.selected].CanPay(sourceID, controller, g)
	}
	for _, opt := range c.options {
		if opt.CanPay(sourceID, controller, g) {
			return true
		}
	}
	return false
}

func (c *eitherCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if c.locked {
		if !c.CanPay(sourceID, controller, g) {
			return fmt.Errorf("chosen cost is no longer payable")
		}
		return c.options[c.selected].Pay(sourceID, controller, g)
	}
	var payable []int
	var labels []string
	for i, opt := range c.options {
		if opt.CanPay(sourceID, controller, g) {
			payable = append(payable, i)
			labels = append(labels, opt.Text())
		}
	}
	if len(payable) == 0 {
		return fmt.Errorf("no payable option")
	}
	if len(payable) == 1 {
		return c.options[payable[0]].Pay(sourceID, controller, g)
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	choice := p.ChooseMode(labels, c.Text())
	if choice < 0 || choice >= len(payable) {
		choice = 0
	}
	return c.options[payable[choice]].Pay(sourceID, controller, g)
}

func (c *eitherCost) Text() string {
	var parts strings.Builder
	for i, opt := range c.options {
		if i > 0 {
			parts.WriteString(" or ")
		}
		parts.WriteString(opt.Text())
	}
	return parts.String()
}

// exileFromGraveyardCost requires exiling cards from your graveyard.
type exileFromGraveyardCost struct {
	amount       int
	locked       bool
	selected     []capturedCard
	paid         *CardsBinding
	paidRequired bool
}

// ExileFromGraveyardCost creates a cost that requires the controller to exile n cards from their graveyard.
func ExileFromGraveyardCost(n int) Cost {
	return &exileFromGraveyardCost{amount: n}
}

// ExileFromGraveyardCostWithCards saves cards moved to exile by this payment.
func ExileFromGraveyardCostWithCards(n int, paid *CardsBinding) Cost {
	return &exileFromGraveyardCost{amount: n, paid: paid, paidRequired: true}
}

func (c *exileFromGraveyardCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "graveyard-exile-cost", Text: c.Text()}
	if c.amount < 0 {
		n.problem = "graveyard exile cost count cannot be negative"
	}
	if c.paid != nil {
		n.bindingWrite = c.paid
		n.children = []definitionEdge{{"paid cards", c.paid}}
	} else if c.paidRequired {
		n.problem = "paid cards binding must be non-nil"
	}
	return n
}

func (c *exileFromGraveyardCost) choose(controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil || c.amount < 0 || len(p.Graveyard()) < c.amount {
		return fmt.Errorf("cannot pay graveyard exile cost")
	}
	c.selected = make([]capturedCard, 0, c.amount)
	candidates := append([]Card(nil), p.Graveyard()...)
	for range c.amount {
		chosen := p.ChooseCardFromGraveyard(candidates, c.Text(), g)
		index := slices.IndexFunc(candidates, func(card Card) bool {
			return chosen != nil && card.ID() == chosen.ID()
		})
		if index < 0 {
			return fmt.Errorf("invalid graveyard exile cost choice")
		}
		c.selected = append(c.selected, capturedCard{id: chosen.ID(), visit: g.graveyardVisit(chosen.ID()), zone: ZoneGraveyard})
		candidates = slices.Delete(candidates, index, index+1)
	}
	c.locked = true
	return nil
}

func (c *exileFromGraveyardCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	if p == nil || c.amount < 0 || len(p.Graveyard()) < c.amount {
		return false
	}
	if !c.locked {
		return true
	}
	if len(c.selected) != c.amount {
		return false
	}
	for _, selected := range c.selected {
		if selected.visit == uuid.Nil || g.graveyardVisit(selected.id) != selected.visit || !slices.ContainsFunc(p.Graveyard(), func(card Card) bool { return card.ID() == selected.id }) {
			return false
		}
	}
	return true
}

func (c *exileFromGraveyardCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.locked {
		chosen := *c
		if err := chosen.choose(controller, g); err != nil {
			return err
		}
		return chosen.Pay(sourceID, controller, g)
	}
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay chosen graveyard exile cost")
	}
	ids := make([]uuid.UUID, 0, c.amount)
	for _, selected := range c.selected {
		ids = append(ids, selected.id)
	}
	removed := g.MoveCardsFromGraveyard(controller, ids, ZoneExile)
	for _, card := range removed {
		g.ExileCard(card, sourceID)
	}
	return nil
}

func (c *exileFromGraveyardCost) Text() string {
	if c.amount == 1 {
		return "Exile a card from your graveyard"
	}
	return fmt.Sprintf("Exile %d cards from your graveyard", c.amount)
}

type exileMatchingCardFromGraveyardCost struct {
	filter      CardFilter
	text        string
	paid        *NumberBinding
	capture     bool
	paidCard    *CardBinding
	captureCard bool
	locked      bool
	selected    uuid.UUID
	visit       uuid.UUID
}

// ExileMatchingCardFromGraveyardCost creates a cost that requires the controller
// to exile a card matching the given filter from their graveyard.
func ExileMatchingCardFromGraveyardCost(filter CardFilter, text string) Cost {
	return &exileMatchingCardFromGraveyardCost{filter: filter, text: text}
}

// ExileMatchingCardFromGraveyardCostInto saves the chosen card's mana value during payment.
func ExileMatchingCardFromGraveyardCostInto(filter CardFilter, text string, paid *NumberBinding) Cost {
	return &exileMatchingCardFromGraveyardCost{filter: filter, text: text, paid: paid, capture: true}
}

// ExileMatchingCardFromGraveyardCostWithCard saves the exiled card in its new exile visit.
func ExileMatchingCardFromGraveyardCostWithCard(filter CardFilter, text string, paid *CardBinding) Cost {
	return &exileMatchingCardFromGraveyardCost{filter: filter, text: text, paidCard: paid, captureCard: true}
}

func (c *exileMatchingCardFromGraveyardCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "exile-graveyard-cost", Text: c.Text()}
	if c.capture {
		n.bindingWrite = c.paid
		n.children = []definitionEdge{{"paid", c.paid}}
		if c.paid == nil {
			n.problem = "supply a non-nil paid mana value binding"
		} else if c.text == "" {
			n.problem = "supply exile cost text"
		}
	}
	if c.captureCard {
		n.bindingWrite = c.paidCard
		n.children = []definitionEdge{{"paid card", c.paidCard}}
		if c.paidCard == nil {
			n.problem = "supply a non-nil paid card binding"
		} else if c.text == "" {
			n.problem = "supply exile cost text"
		}
	}
	return n
}

func (c *exileMatchingCardFromGraveyardCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	if p == nil {
		return false
	}
	if c.locked {
		return slices.ContainsFunc(p.Graveyard(), func(card Card) bool {
			return card.ID() == c.selected && g.graveyardVisit(card.ID()) == c.visit && c.filter.Match(card)
		})
	}
	return slices.ContainsFunc(p.Graveyard(), c.filter.Match)
}

func (c *exileMatchingCardFromGraveyardCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	var candidates []Card
	for _, card := range p.Graveyard() {
		if c.filter.Match(card) {
			candidates = append(candidates, card)
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no matching card in graveyard to exile")
	}
	var chosen Card
	if c.locked {
		for _, card := range candidates {
			if card.ID() == c.selected && g.graveyardVisit(card.ID()) == c.visit {
				chosen = card
				break
			}
		}
		if chosen == nil {
			return fmt.Errorf("chosen graveyard card is absent")
		}
	} else {
		chosen = p.ChooseCardFromGraveyard(candidates, c.text, g)
		if chosen == nil {
			chosen = candidates[0]
		}
	}
	removed, ok := g.MoveFromGraveyard(controller, chosen.ID(), ZoneExile)
	if !ok || removed == nil {
		return fmt.Errorf("failed to move card %s from graveyard", chosen.Name())
	}
	g.ExileCard(removed, sourceID)
	g.SetLastExiledCard(removed)
	return nil
}

func (c *exileMatchingCardFromGraveyardCost) Text() string {
	if c.text != "" {
		return c.text
	}
	return "Exile a card from your graveyard"
}

// returnToHandCost requires returning a permanent you control to its owner's hand.
type returnToHandCost struct {
	filter *PermanentFilter
}

// ReturnToHandCost creates a cost that requires bouncing a permanent you control to hand.
// Pass nil for filter to allow any permanent.
func ReturnToHandCost(filter *PermanentFilter) Cost {
	return &returnToHandCost{filter: filter}
}

func (c *returnToHandCost) matchFilter(p *Permanent, g *Game) bool {
	if c.filter == nil {
		return true
	}
	return c.filter.Match(p, g)
}

func (c *returnToHandCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == controller && p.ID() != sourceID && c.matchFilter(p, g) {
			return true
		}
	}
	return false
}

func (c *returnToHandCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == controller && p.ID() != sourceID && c.matchFilter(p, g) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no permanent to return")
	}
	player := g.GetPlayer(controller)
	chosen := player.ChoosePermanent(candidates, "return to hand cost", g)
	if chosen == nil {
		return fmt.Errorf("no permanent chosen")
	}
	isToken := chosen.IsToken
	g.RemoveFromBattlefield(chosen)
	if !isToken {
		owner := g.GetPlayer(chosen.Card.Owner())
		if owner != nil {
			owner.AddToHand(chosen.Card)
		}
	}
	return nil
}

func (c *returnToHandCost) Text() string { return "Return a permanent you control to its owner's hand" }

// exileSourceCost requires exiling the source permanent.
type exileSourceCost struct{}

// ExileSourceCost creates a cost that requires exiling the source permanent.
func ExileSourceCost() Cost { return &exileSourceCost{} }

func (c *exileSourceCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return g.FindPermanent(sourceID) != nil
}

func (c *exileSourceCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return ErrSourceNotFound
	}
	g.ExilePermanent(p)
	return nil
}

func (c *exileSourceCost) Text() string { return "Exile ~" }

// xManaCost requires paying X generic mana, where X is set by the game/harness.
type xManaCost struct{}

// XManaCost creates a cost that requires paying X generic mana.
// The X value must be set on g.currentX before activation.
func XManaCost() Cost { return &xManaCost{} }

func (c *xManaCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return true // X can always be 0
}

func (c *xManaCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	x := g.resolution.X()
	if x <= 0 {
		return nil
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	mc := ManaCost{Generic: x}
	if !p.ManaPool().CanPay(mc, nil) {
		return fmt.Errorf("cannot pay {%d}", x)
	}
	return p.ManaPool().Pay(mc, nil)
}

func (c *xManaCost) Text() string { return "{X}" }

// discardRandomCost requires discarding n cards at random from hand.
type discardRandomCost struct {
	amount int
}

// DiscardRandomCost creates a cost that requires the controller to discard n cards at random.
func DiscardRandomCost(n int) Cost {
	return &discardRandomCost{amount: n}
}

func (c *discardRandomCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	return p != nil && len(p.Hand()) >= c.amount
}

func (c *discardRandomCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	hand := p.Hand()
	if len(hand) < c.amount {
		return fmt.Errorf("not enough cards in hand to discard")
	}
	for i := 0; i < c.amount; i++ {
		hand = p.Hand()
		if len(hand) == 0 {
			break
		}
		idx := rand.Intn(len(hand))
		card := hand[idx]
		g.PlayerDiscard(p, card.ID())
	}
	return nil
}

func (c *discardRandomCost) Text() string {
	if c.amount == 1 {
		return "Discard a card at random"
	}
	return fmt.Sprintf("Discard %d cards at random", c.amount)
}
