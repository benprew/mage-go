package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type tapAttachedPermanentCost struct {
	locked            bool
	sourceIncarnation uuid.UUID
	host              capturedPermanent
}

// TapAttachedPermanentCost taps the permanent attached to the ability source.
func TapAttachedPermanentCost() Cost { return &tapAttachedPermanentCost{} }

func (c *tapAttachedPermanentCost) Text() string { return "Tap attached permanent" }

func (c *tapAttachedPermanentCost) inspectDefinition() definitionNode {
	return definitionNode{Role: "cost", Text: c.Text(), battlefieldSource: true}
}

func (c *tapAttachedPermanentCost) capture(sourceID uuid.UUID, g *Game) bool {
	source := g.FindPermanent(sourceID)
	if source == nil || source.PhasedOut || source.AttachedTo == uuid.Nil || source.attachedIncarnation == uuid.Nil {
		return false
	}
	c.sourceIncarnation = source.incarnationID
	c.host = capturedPermanent{id: source.AttachedTo, incarnation: source.attachedIncarnation}
	c.locked = true
	return true
}

func (c *tapAttachedPermanentCost) CanPay(sourceID, _ uuid.UUID, g *Game) bool {
	source := g.FindPermanent(sourceID)
	if source == nil || source.PhasedOut || source.AttachedTo == uuid.Nil || source.attachedIncarnation == uuid.Nil {
		return false
	}
	if c.locked && (source.incarnationID != c.sourceIncarnation || source.AttachedTo != c.host.id || source.attachedIncarnation != c.host.incarnation) {
		return false
	}
	host := g.FindPermanent(source.AttachedTo)
	return host != nil && !host.PhasedOut && !host.Tapped && host.incarnationID == source.attachedIncarnation
}

func (c *tapAttachedPermanentCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot tap attached permanent")
	}
	source := g.FindPermanent(sourceID)
	g.TapPermanent(g.FindPermanent(source.AttachedTo))
	return nil
}
