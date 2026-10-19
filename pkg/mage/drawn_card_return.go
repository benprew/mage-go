package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type drawnHandVisit struct{ player, card, visit uuid.UUID }

// MayDrawCardsThen runs the body after the player accepts the draws.
// Replaced draws do not cancel the accepted continuation.
func MayDrawCardsThen(player PlayerRef, count int, body ResolvingEffect) ResolvingEffect {
	return &mayDrawCardsThen{player, count, body}
}

type mayDrawCardsThen struct {
	player PlayerRef
	count  int
	body   ResolvingEffect
}

func (e *mayDrawCardsThen) Text() string { return fmt.Sprintf("may draw %d cards", e.count) }
func (e *mayDrawCardsThen) Properties() EffectProperties {
	p := drawAmountProperties(Fixed(e.count))
	if !nilDefinition(e.body) {
		mergeEffectProperties(&p, e.body.Properties())
	}
	return p
}
func (e *mayDrawCardsThen) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}, {"body", e.body}}}
	if e.count < 0 || nilDefinition(e.body) {
		n.problem = "draw continuation requires a nonnegative count and a body"
	}
	return n
}
func (e *mayDrawCardsThen) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.body)
}
func (e *mayDrawCardsThen) Apply(ctx *EffectContext) error {
	p := handOperationPlayer(e.player, ctx)
	if p == nil || !p.ChooseMayAbility(e.Text()) {
		return nil
	}
	for range e.count {
		ctx.Game.PlayerDrawCard(p)
	}
	return e.body.Apply(ctx)
}

// PayLifeOrReturnDrawnCards chooses current hand visits before any payment.
// Each chosen card requires a separate life payment or library return.
func PayLifeOrReturnDrawnCards(player PlayerRef, count, life int) ResolvingEffect {
	return &drawnCardReturn{player, count, life}
}

type drawnCardReturn struct {
	player      PlayerRef
	count, life int
}

func (*drawnCardReturn) Text() string {
	return "choose cards drawn this turn; pay life or put each on top of its owner's library"
}
func (*drawnCardReturn) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *drawnCardReturn) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *drawnCardReturn) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}}}
	if e.count < 0 || e.life < 0 {
		n.problem = "drawn card return requires nonnegative count and life"
	}
	return n
}
func (e *drawnCardReturn) Apply(ctx *EffectContext) error {
	p := handOperationPlayer(e.player, ctx)
	if p == nil {
		return nil
	}
	g := ctx.Game
	var candidates []Card
	visits := make(map[uuid.UUID]uuid.UUID)
	for _, c := range p.Hand() {
		visit := g.handVisit(c.ID())
		if g.trackers.Turn.drawnHandVisits[drawnHandVisit{p.PlayerID(), c.ID(), visit}] {
			candidates = append(candidates, c)
			visits[c.ID()] = visit
		}
	}
	current := func(c Card) bool {
		return c != nil && visits[c.ID()] != uuid.Nil && g.handVisit(c.ID()) == visits[c.ID()] && slices.ContainsFunc(p.Hand(), func(now Card) bool { return now.ID() == c.ID() })
	}
	var chosen []Card
	for range min(e.count, len(candidates)) {
		pick := p.ChooseCardFromHand(slices.Clone(candidates), e.Text(), g)
		index := -1
		if pick != nil {
			index = slices.IndexFunc(candidates, func(c Card) bool { return c.ID() == pick.ID() && current(c) })
		}
		if index < 0 {
			index = slices.IndexFunc(candidates, current)
		}
		if index < 0 {
			break
		}
		chosen = append(chosen, candidates[index])
		candidates = slices.Delete(candidates, index, index+1)
	}
	for _, c := range chosen {
		if !current(c) {
			continue
		}
		if tryResolutionCostPaymentFor(ctx, p.PlayerID(), []Cost{LifePayCost(e.life)}, false, c.Name()) {
			continue
		}
		if !current(c) {
			continue
		}
		proposal := &CardZoneMoveAction{source: ctx.SourceID, card: c.ID(), visit: visits[c.ID()], from: core.ZoneHand, destination: core.ZoneLibrary, owner: c.Owner()}
		if result, ok := g.effects.ApplyReplacements(proposal, g).(*CardZoneMoveAction); ok && result != nil {
			g.performCardZoneMove(result)
		}
	}
	return nil
}
