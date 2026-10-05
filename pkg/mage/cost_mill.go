package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

type millCost struct {
	amount       int
	lockedAmount int
	locked       bool
	paid         *NumberBinding
	capture      bool
}

// MillCost pays by milling the specified number of cards.
func MillCost(amount int) Cost { return &millCost{amount: amount} }

// MillCostInto saves the adjusted count only after payment commits.
func MillCostInto(amount int, paid *NumberBinding) Cost {
	return &millCost{amount: amount, paid: paid, capture: true}
}

func (c *millCost) Text() string { return fmt.Sprintf("Mill %d cards", c.amount) }

func (c *millCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", Text: c.Text()}
	if c.amount <= 0 {
		n.problem = "mill cost requires a positive count"
	}
	if c.capture {
		if c.paid == nil {
			n.problem = "paid mill count binding must be non-nil"
		} else {
			n.bindingWrite = c.paid
			n.children = append(n.children, definitionEdge{"paid count", c.paid})
		}
	}
	return n
}

func (c *millCost) payableAmount(controller uuid.UUID, g *Game) (int, bool) {
	if c.amount <= 0 || g.GetPlayer(controller) == nil {
		return 0, false
	}
	amount := max(0, g.ApplyMillModifiers(controller, c.amount))
	if c.locked && amount != c.lockedAmount {
		return 0, false
	}
	return amount, len(g.GetPlayer(controller).Library()) >= amount
}

func (c *millCost) CanPay(_, controller uuid.UUID, g *Game) bool {
	_, ok := c.payableAmount(controller, g)
	return ok
}

func (c *millCost) Pay(_, controller uuid.UUID, g *Game) error {
	amount, ok := c.payableAmount(controller, g)
	if !ok {
		return fmt.Errorf("cannot pay mill cost")
	}
	g.millCards(controller, amount)
	return nil
}

func (g *Game) millCards(playerID uuid.UUID, amount int) {
	p := g.GetPlayer(playerID)
	if p == nil || amount <= 0 {
		return
	}
	library := slices.Clone(p.Library())
	count := min(amount, len(library))
	remaining := slices.Clone(library[:count])
	ordered := make([]Card, 0, count)
	for len(remaining) > 1 {
		snapshots := make([]Card, len(remaining))
		for i, card := range remaining {
			snapshots[i] = card.Copy()
			snapshots[i].SetID(card.ID())
		}
		chosen := p.ChooseCardFromLibrary(snapshots, "choose the next card for your graveyard", g)
		index := max(0, slices.IndexFunc(remaining, func(card Card) bool { return chosen != nil && card.ID() == chosen.ID() }))
		ordered = append(ordered, remaining[index])
		remaining = slices.Delete(remaining, index, index+1)
	}
	ordered = append(ordered, remaining...)
	// All choices precede the moves so the player sees one complete batch.
	p.SetLibrary(library[count:])
	for _, card := range ordered {
		p.AddToGraveyard(card)
	}
}
