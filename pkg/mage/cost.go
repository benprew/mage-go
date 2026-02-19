package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"fmt"

	"github.com/google/uuid"
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
	p := g.GetPlayer(controller)
	if p == nil {
		return false
	}
	return p.ManaPool().CanPay(c.MC)
}

func (c *ManaCostPayment) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	return p.ManaPool().Pay(c.MC)
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
	return p != nil && !p.Tapped
}

func (c *tapSourceCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return ErrSourceNotFound
	}
	if p.Tapped {
		return ErrSourceTapped
	}
	p.Tapped = true
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
	return p != nil && p.Counters[c.ct] >= c.amount
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
	p.LoseLife(c.amount)
	return nil
}

func (c *lifePayCost) Text() string {
	return fmt.Sprintf("Pay %d life", c.amount)
}

// sacrificeCreatureCost requires sacrificing a creature you control.
type sacrificeCreatureCost struct{}

// SacrificeCreatureCost creates a cost that requires sacrificing a creature you control (other than the source).
func SacrificeCreatureCost() Cost {
	return &sacrificeCreatureCost{}
}

func (c *sacrificeCreatureCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.Battlefield {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			return true
		}
	}
	return false
}

func (c *sacrificeCreatureCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.Battlefield {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return ErrNoCreature
	}
	player := g.GetPlayer(controller)
	chosen := player.ChoosePermanent(candidates, "sacrifice cost", g)
	if chosen == nil {
		return ErrNoCreature
	}
	g.Sacrifice(chosen)
	return nil
}

func (c *sacrificeCreatureCost) Text() string { return "Sacrifice a creature" }
