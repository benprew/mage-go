package mage

import (
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

func ManaCostOf(s string) Cost {
	return &ManaCostPayment{MC: ParseManaCost(s)}
}

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
		return fmt.Errorf("player not found")
	}
	return p.ManaPool().Pay(c.MC)
}

func (c *ManaCostPayment) Text() string {
	return c.MC.String()
}

// tapSourceCost requires tapping the source permanent.
type tapSourceCost struct{}

func TapSourceCost() Cost { return &tapSourceCost{} }

func (c *tapSourceCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && !p.Tapped
}

func (c *tapSourceCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return fmt.Errorf("source not found on battlefield")
	}
	if p.Tapped {
		return fmt.Errorf("source is already tapped")
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
		return fmt.Errorf("source not found on battlefield")
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

func SacrificeSourceCost() Cost { return &sacrificeSourceCost{} }

func (c *sacrificeSourceCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return g.FindPermanent(sourceID) != nil
}

func (c *sacrificeSourceCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return fmt.Errorf("source not found on battlefield")
	}
	g.Sacrifice(p)
	return nil
}

func (c *sacrificeSourceCost) Text() string { return "Sacrifice ~" }

// lifePayCost requires paying life.
type lifePayCost struct {
	amount int
}

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
		return fmt.Errorf("player not found")
	}
	p.LoseLife(c.amount)
	return nil
}

func (c *lifePayCost) Text() string {
	return fmt.Sprintf("Pay %d life", c.amount)
}

// sacrificeCreatureCost requires sacrificing a creature you control.
type sacrificeCreatureCost struct{}

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
	// Sacrifice the first creature controlled that isn't the source
	for _, p := range g.Battlefield {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			g.Sacrifice(p)
			return nil
		}
	}
	return fmt.Errorf("no creature to sacrifice")
}

func (c *sacrificeCreatureCost) Text() string { return "Sacrifice a creature" }
