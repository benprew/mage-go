package mage

import (
	"fmt"
	"slices"
)

// OfferCostForEachPermanent saves exact visits for which no player pays.
// All offers precede later actions on the saved set.
func OfferCostForEachPermanent(subjects PermanentSet, players PlayerSet, costs CostGroup, unpaid *PermanentSetBinding) ResolvingEffect {
	return &permanentPayment{subjects, players, costs, unpaid}
}

type permanentPayment struct {
	subjects PermanentSet
	players  PlayerSet
	costs    CostGroup
	unpaid   *PermanentSetBinding
}

func (*permanentPayment) Text() string                 { return "offer a cost for each permanent" }
func (*permanentPayment) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *permanentPayment) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingWrite: e.unpaid, children: []definitionEdge{{"subjects", e.subjects}, {"players", e.players}, {"costs", e.costs}, {"unpaid", e.unpaid}}}
}
func (e *permanentPayment) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects) || playerSetReadsSourcePermanent(e.players)
}
func (e *permanentPayment) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.createdTokenBindings == nil {
		ctx.Game.resolution.createdTokenBindings = make(map[*PermanentSetBinding][]capturedPermanent)
	}
	delete(ctx.Game.resolution.createdTokenBindings, e.unpaid)
	eligible := e.players.resolvePlayers(ctx)
	groups := controlledPermanentGroups(ctx, EachPlayer, e.subjects)
	chooser := ctx.Game.GetPlayer(ctx.Controller)
	visits := []capturedPermanent{}
	for _, group := range groups {
		members := slices.Clone(group.members)
		for len(members) > 0 {
			index := 0
			if len(members) > 1 && chooser != nil {
				chosen := chooser.ChoosePermanent(slices.Clone(members), "Choose the next permanent for payment offers", ctx.Game)
				index = max(0, slices.IndexFunc(members, func(p *Permanent) bool {
					return chosen != nil && p.ID() == chosen.ID() && p.incarnationID == chosen.incarnationID
				}))
			}
			p := members[index]
			visits = append(visits, capturedPermanent{p.ID(), p.incarnationID})
			members = slices.Delete(members, index, index+1)
		}
	}
	unpaid := []capturedPermanent{}
	for _, visit := range visits {
		p := ctx.Game.FindPermanent(visit.id)
		if p == nil || p.PhasedOut || p.incarnationID != visit.incarnation {
			continue
		}
		if !offerEligibleResolutionCost(ctx, eligible, e.costs.costs, nil, fmt.Sprintf("%s (%s)", p.Name(), p.ID())) {
			unpaid = append(unpaid, visit)
		}
	}
	ctx.Game.resolution.createdTokenBindings[e.unpaid] = unpaid
	return nil
}
