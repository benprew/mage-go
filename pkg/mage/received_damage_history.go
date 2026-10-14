package mage

// WasDealtDamageThisTurn matches present visits that received positive damage.
// Damage removal does not erase the history. Final turn cleanup clears it.
var WasDealtDamageThisTurn = NewPermanentFilter("was dealt damage this turn", func(p *Permanent, g *Game) bool {
	if p == nil || g == nil || p.PhasedOut {
		return false
	}
	present := g.FindPermanent(p.ID())
	return present != nil && present.incarnationID == p.incarnationID &&
		g.damage.turnDamageRecipients[capturedPermanent{p.ID(), p.incarnationID}]
})
