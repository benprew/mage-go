package mage

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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

func (c *ManaCostPayment) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	mc := c.reducedCost(sourceID, g)
	return g.CanAfford(controller, mc)
}

func (c *ManaCostPayment) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	mc := c.reducedCost(sourceID, g)
	return p.ManaPool().Pay(mc)
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

// tapSourceCost requires tapping the source permanent.
type tapSourceCost struct{}

// TapSourceCost creates a cost that requires tapping the source permanent ({T}).
func TapSourceCost() Cost { return &tapSourceCost{} }

func (c *tapSourceCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && !p.Tapped && p.CanTapForEffect(g)
}

func (c *tapSourceCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
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

func (c *tapSourceCost) Text() string { return "{T}" }

// removeCountersCost requires removing counters from the source.
type removeCountersCost struct {
	ct     CounterType
	amount int
}

// RemoveCountersCost creates a cost that requires removing n counters of the given type from the source.
func RemoveCountersCost(ct CounterType, n int) Cost {
	return &removeCountersCost{ct: ct, amount: n}
}

func (c *removeCountersCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && int(p.Counters[c.ct]) >= c.amount
}

func (c *removeCountersCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
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

// sacrificeSourceCost requires sacrificing the source.
type sacrificeSourceCost struct{}

// SacrificeSourceCost creates a cost that requires sacrificing the source permanent.
func SacrificeSourceCost() Cost { return &sacrificeSourceCost{} }

func (c *sacrificeSourceCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return g.FindPermanent(sourceID) != nil
}

func (c *sacrificeSourceCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return ErrSourceNotFound
	}
	g.CaptureSacrificed(p)
	g.Sacrifice(p)
	return nil
}

func (c *sacrificeSourceCost) Text() string { return "Sacrifice ~" }

// lifePayCost requires paying life.
type lifePayCost struct {
	amount int
}

// LifePayCost creates a cost that requires the controller to pay life.
func LifePayCost(amount int) Cost {
	return &lifePayCost{amount: amount}
}

func (c *lifePayCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	return p != nil && p.Life() > c.amount // must have more life than cost
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

// sacrificeMatchingCost requires sacrificing a permanent you control matching a filter.
type sacrificeMatchingCost struct {
	filter PermanentFilter
	text   string
}

// SacrificeMatchingCost creates a cost that requires sacrificing a permanent you control
// (other than the source) that matches the given filter.
func SacrificeMatchingCost(filter PermanentFilter, text string) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text}
}

// SacrificeArtifactCost creates a cost that requires sacrificing an artifact you control (other than the source).
func SacrificeArtifactCost() Cost {
	return SacrificeMatchingCost(IsArtifact, "Sacrifice an artifact")
}

// SacrificeCreatureCost creates a cost that requires sacrificing a creature you control (other than the source).
func SacrificeCreatureCost() Cost {
	return SacrificeMatchingCost(IsCreature, "Sacrifice a creature")
}

func (c *sacrificeMatchingCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.battlefield {
		if p.Controller == controller && p.ID() != sourceID && c.filter.Match(p, g) {
			return true
		}
	}
	return false
}

func (c *sacrificeMatchingCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.battlefield {
		if p.Controller == controller && p.ID() != sourceID && c.filter.Match(p, g) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no permanent to sacrifice")
	}
	player := g.GetPlayer(controller)
	chosen := player.ChoosePermanent(candidates, c.text, g)
	if chosen == nil {
		return fmt.Errorf("no permanent chosen")
	}
	g.CaptureSacrificed(chosen)
	g.Sacrifice(chosen)
	return nil
}

func (c *sacrificeMatchingCost) Text() string { return c.text }

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
	for _, p := range g.battlefield {
		if p.Controller == controller && p.ID() != sourceID && !p.Tapped && p.CanTapForEffect(g) && c.filter.Match(p, g) {
			return true
		}
	}
	return false
}

func (c *tapMatchingCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.battlefield {
		if p.Controller == controller && p.ID() != sourceID && !p.Tapped && p.CanTapForEffect(g) && c.filter.Match(p, g) {
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
	amount int
}

// DiscardCost creates a cost that requires the controller to discard n cards.
func DiscardCost(n int) Cost {
	return &discardCost{amount: n}
}

func (c *discardCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	return p != nil && len(p.Hand()) >= c.amount
}

func (c *discardCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	chosen := p.ChooseCardsFromHand(c.amount, "discard cost", g)
	for _, card := range chosen {
		g.PlayerDiscard(p, card.ID())
	}
	return nil
}

func (c *discardCost) Text() string {
	if c.amount == 1 {
		return "Discard a card"
	}
	return fmt.Sprintf("Discard %d cards", c.amount)
}

// eitherCost is a branching additional cost: the controller chooses one of
// several alternatives at pay time. Used for "as an additional cost,
// discard a card or pay {N}" and similar OR-style additional costs.
type eitherCost struct {
	options []Cost
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

func (c *eitherCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, opt := range c.options {
		if opt.CanPay(sourceID, controller, g) {
			return true
		}
	}
	return false
}

func (c *eitherCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
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
	parts := ""
	for i, opt := range c.options {
		if i > 0 {
			parts += " or "
		}
		parts += opt.Text()
	}
	return parts
}

// exileFromGraveyardCost requires exiling cards from your graveyard.
type exileFromGraveyardCost struct {
	amount int
}

// ExileFromGraveyardCost creates a cost that requires the controller to exile n cards from their graveyard.
func ExileFromGraveyardCost(n int) Cost {
	return &exileFromGraveyardCost{amount: n}
}

func (c *exileFromGraveyardCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	return p != nil && len(p.Graveyard()) >= c.amount
}

func (c *exileFromGraveyardCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	gy := p.Graveyard()
	n := c.amount
	if n > len(gy) {
		n = len(gy)
	}
	for i := 0; i < n; i++ {
		card := gy[i]
		if _, ok := p.RemoveFromGraveyard(card.ID()); ok {
			g.exile = append(g.exile, ExiledCard{Card: card, ExiledBy: sourceID})
		}
	}
	return nil
}

func (c *exileFromGraveyardCost) Text() string {
	if c.amount == 1 {
		return "Exile a card from your graveyard"
	}
	return fmt.Sprintf("Exile %d cards from your graveyard", c.amount)
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
	for _, p := range g.battlefield {
		if p.Controller == controller && p.ID() != sourceID && c.matchFilter(p, g) {
			return true
		}
	}
	return false
}

func (c *returnToHandCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.battlefield {
		if p.Controller == controller && p.ID() != sourceID && c.matchFilter(p, g) {
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
	g.RemoveFromBattlefield(chosen)
	owner := g.GetPlayer(chosen.Card.Owner())
	if owner != nil {
		owner.AddToHand(chosen.Card)
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
	x := g.currentX
	if x <= 0 {
		return nil
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	mc := ManaCost{Generic: x}
	if !p.ManaPool().CanPay(mc) {
		return fmt.Errorf("cannot pay {%d}", x)
	}
	return p.ManaPool().Pay(mc)
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
