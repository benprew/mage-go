package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type millCost struct {
	amount       int
	lockedAmount int
	locked       bool
}

// MillCost pays by milling the specified number of cards.
func MillCost(amount int) Cost { return &millCost{amount: amount} }

func (c *millCost) Text() string { return fmt.Sprintf("Mill %d cards", c.amount) }

func (c *millCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", Text: c.Text()}
	if c.amount <= 0 {
		n.problem = "mill cost requires a positive count"
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
	lib := p.Library()
	for i := 0; i < amount && len(lib) > 0; i++ {
		card := lib[0]
		lib = lib[1:]
		p.AddToGraveyard(card)
	}
	p.SetLibrary(lib)
}
