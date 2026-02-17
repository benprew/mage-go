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

// TapSourceCost requires tapping the source permanent.
type TapSourceCostImpl struct{}

func TapSourceCost() Cost { return &TapSourceCostImpl{} }

func (c *TapSourceCostImpl) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && !p.Tapped
}

func (c *TapSourceCostImpl) Pay(sourceID, controller uuid.UUID, g *Game) error {
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

func (c *TapSourceCostImpl) Text() string { return "{T}" }

// RemoveCountersCostImpl requires removing counters from the source.
type RemoveCountersCostImpl struct {
	CT     CounterType
	Amount int
}

func RemoveCountersCost(ct CounterType, n int) Cost {
	return &RemoveCountersCostImpl{CT: ct, Amount: n}
}

func (c *RemoveCountersCostImpl) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && p.Counters[c.CT] >= c.Amount
}

func (c *RemoveCountersCostImpl) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return fmt.Errorf("source not found on battlefield")
	}
	if !p.RemoveCounter(c.CT, c.Amount) {
		return fmt.Errorf("not enough %s counters", c.CT)
	}
	return nil
}

func (c *RemoveCountersCostImpl) Text() string {
	return fmt.Sprintf("Remove %d %s counter(s)", c.Amount, c.CT)
}

// SacrificeSourceCostImpl requires sacrificing the source.
type SacrificeSourceCostImpl struct{}

func SacrificeSourceCost() Cost { return &SacrificeSourceCostImpl{} }

func (c *SacrificeSourceCostImpl) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return g.FindPermanent(sourceID) != nil
}

func (c *SacrificeSourceCostImpl) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return fmt.Errorf("source not found on battlefield")
	}
	g.Sacrifice(p)
	return nil
}

func (c *SacrificeSourceCostImpl) Text() string { return "Sacrifice ~" }

// LifePayCostImpl requires paying life.
type LifePayCostImpl struct {
	Amount int
}

func LifePayCost(amount int) Cost {
	return &LifePayCostImpl{Amount: amount}
}

func (c *LifePayCostImpl) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	return p != nil && p.Life() > c.Amount // must have more life than cost
}

func (c *LifePayCostImpl) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return fmt.Errorf("player not found")
	}
	p.LoseLife(c.Amount)
	return nil
}

func (c *LifePayCostImpl) Text() string {
	return fmt.Sprintf("Pay %d life", c.Amount)
}

// SacrificeCreatureCostImpl requires sacrificing a creature you control.
type SacrificeCreatureCostImpl struct{}

func SacrificeCreatureCost() Cost {
	return &SacrificeCreatureCostImpl{}
}

func (c *SacrificeCreatureCostImpl) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.Battlefield {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			return true
		}
	}
	return false
}

func (c *SacrificeCreatureCostImpl) Pay(sourceID, controller uuid.UUID, g *Game) error {
	// Sacrifice the first creature controlled that isn't the source
	for _, p := range g.Battlefield {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			g.Sacrifice(p)
			return nil
		}
	}
	return fmt.Errorf("no creature to sacrifice")
}

func (c *SacrificeCreatureCostImpl) Text() string { return "Sacrifice a creature" }
