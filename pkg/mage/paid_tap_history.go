package mage

import "github.com/google/uuid"

type paidTapPair struct{ source, payer capturedPermanent }

func (g *Game) recordAbilityTapPayment(source, payer capturedPermanent) {
	if source.id == uuid.Nil || source.incarnation == uuid.Nil {
		return
	}
	if g.trackers.Turn.paidTapHistory == nil {
		g.trackers.Turn.paidTapHistory = make(map[paidTapPair]bool)
	}
	g.trackers.Turn.paidTapHistory[paidTapPair{source, payer}] = true
}

func (tx *costPaymentTransaction) recordAbilityTapPayments(source capturedPermanent) {
	for _, payer := range tx.paidTaps {
		tx.game.recordAbilityTapPayment(source, payer)
	}
}

func tapCostVisits(cost Cost, sourceID uuid.UUID, g *Game) []capturedPermanent {
	var candidates []*Permanent
	switch c := cost.(type) {
	case *tap:
		candidates = append(candidates, g.FindPermanent(sourceID))
	case *tapAttachedPermanentCost:
		if source := g.FindPermanent(sourceID); source != nil {
			candidates = append(candidates, g.FindPermanent(source.AttachedTo))
		}
	case *tapNOtherMatchingCost:
		for _, visit := range c.selected {
			candidates = append(candidates, g.FindPermanent(visit.id))
		}
	case *exactPermanentTapCost:
		candidates = append(candidates, g.FindPermanent(c.subject.id))
	case *tapMatchingCost:
		candidates = g.FilterBattlefield(c.filter)
	}
	var visits []capturedPermanent
	for _, p := range candidates {
		if p != nil && !p.Tapped && !p.PhasedOut {
			visits = append(visits, capturedPermanent{p.ID(), p.incarnationID})
		}
	}
	return visits
}

// TappedForAbilitiesThisTurn selects present payers for one source visit.
// A source can leave without loss of its current-turn payment history.
func TappedForAbilitiesThisTurn(source PermanentRef) PermanentSet { return paidTapSet{source} }

type paidTapSet struct{ source PermanentRef }

func (paidTapSet) permanentSet() {}
func (s paidTapSet) inspectDefinition() definitionNode {
	return permanentInformationNode(s.source, "permanent-set", "permanents tapped to pay for this source's abilities this turn")
}
func (s paidTapSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if ctx == nil || ctx.Game == nil || nilDefinition(s.source) {
		return nil
	}
	ref, ok := s.source.(permanentInformationReference)
	if !ok {
		return nil
	}
	source, ok := ref.informationIdentity(ctx)
	if !ok {
		return nil
	}
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(AnyPermanent) {
		if !p.PhasedOut && ctx.Game.trackers.Turn.paidTapHistory[paidTapPair{source, capturedPermanent{p.ID(), p.incarnationID}}] {
			result = append(result, p)
		}
	}
	return result
}
