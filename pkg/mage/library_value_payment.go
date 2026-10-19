package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// IfSacrificeWithManaValue saves one required sacrifice's mana value.
// Only the successful continuation can read the saved value.
func IfSacrificeWithManaValue(player PlayerRef, filter PermanentFilter, result *NumberBinding, paid ResolvingEffect) ResolvingEffect {
	return &sacrificeManaContinuation{player, filter, result, paid}
}

type sacrificeManaContinuation struct {
	player PlayerRef
	filter PermanentFilter
	result *NumberBinding
	paid   ResolvingEffect
}

func (*sacrificeManaContinuation) Text() string { return "sacrifice, then use its mana value" }
func (e *sacrificeManaContinuation) Properties() EffectProperties {
	if nilDefinition(e.paid) {
		return EffectProperties{}
	}
	return e.paid.Properties()
}
func (e *sacrificeManaContinuation) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.paid)
}
func (e *sacrificeManaContinuation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "sacrifice-mana-continuation", children: []definitionEdge{{"player", e.player}, {"result", e.result}, {"paid", Sequence(&sacrificeManaResult{e.result}, e.paid)}}}
	if e.filter.IsZero() {
		n.problem = "supply an explicit sacrifice filter"
	}
	return n
}
func (e *sacrificeManaContinuation) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return nil
	}
	cost := SacrificeMatchingIncludingSourceCostWithManaValue(e.filter, "Sacrifice a matching permanent", e.result)
	tx, err := ctx.Game.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: ctx.SourceID, Controller: id, Costs: []Cost{cost}, Targets: ctx.Targets, XValue: ctx.Game.XValue()})
	if err != nil || tx.Commit() != nil {
		return nil
	}
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	values := ctx.Game.resolution.numberBindings
	previous, had := values[e.result]
	values[e.result] = tx.paidNumbers[e.result]
	defer func() {
		if had {
			values[e.result] = previous
		} else {
			delete(values, e.result)
		}
	}()
	return e.paid.Apply(ctx)
}

// SearchLibraryWithManaDifference offers payment above a saved mana value.
// An unpaid find moves to its owner's graveyard before the library is shuffled.
func SearchLibraryWithManaDifference(player PlayerRef, filter CardFilter, allowance ResolutionValue) ResolvingEffect {
	return &libraryManaDifference{player, filter, allowance}
}

type libraryManaDifference struct {
	player    PlayerRef
	filter    CardFilter
	allowance ResolutionValue
}

func (*libraryManaDifference) Text() string {
	return "search, pay the mana value difference, then shuffle"
}
func (*libraryManaDifference) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *libraryManaDifference) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.allowance)
}
func (e *libraryManaDifference) inspectDefinition() definitionNode {
	n := definitionNode{Role: "library-mana-difference", children: []definitionEdge{{"player", e.player}, {"allowance", e.allowance}}}
	if e.filter.IsZero() {
		n.problem = "supply an explicit search filter"
	}
	return n
}
func (e *libraryManaDifference) Apply(ctx *EffectContext) error {
	p := handOperationPlayer(e.player, ctx)
	if p == nil {
		return nil
	}
	allowance := max(0, e.allowance.resolveValue(ctx))
	visits := map[uuid.UUID]uuid.UUID{}
	var candidates []Card
	for _, card := range p.Library() {
		if e.filter.Match(card) {
			visit, storage := ctx.Game.cardMovementVisit(card.ID(), core.ZoneLibrary)
			if visit != uuid.Nil && storage == p.PlayerID() {
				visits[card.ID()] = visit
				candidates = append(candidates, card)
			}
		}
	}
	defer p.ShuffleLibrary()
	card := chooseLibrarySearchCard(p, candidates, e.Text(), ctx.Game, false)
	if card == nil {
		return nil
	}
	current := func() bool {
		visit, storage := ctx.Game.cardMovementVisit(card.ID(), core.ZoneLibrary)
		return visit != uuid.Nil && visit == visits[card.ID()] && storage == p.PlayerID()
	}
	if !current() {
		return nil
	}
	difference := card.ManaCost().CMC() - allowance
	paid := difference <= 0
	if !paid {
		paid = tryResolutionCostPayment(ctx, p.PlayerID(), []Cost{&ManaCostPayment{MC: core.ManaCost{Generic: difference}}}, false)
	}
	if !current() {
		return nil
	}
	if !paid {
		ctx.Game.moveLibraryFindToGraveyard(ctx.SourceID, card, visits[card.ID()])
		return nil
	}
	perm, failed := ctx.Game.prepareBattlefieldEntryResultWithContext(card, p.PlayerID(), nil, battlefieldEntryContext{actor: p.PlayerID()}, false)
	if !current() || perm == nil && failed == nil {
		return nil
	}
	if perm == nil {
		ctx.Game.moveLibraryFindToGraveyard(ctx.SourceID, card, visits[card.ID()])
		return nil
	}
	remaining := make([]Card, 0, len(p.Library())-1)
	for _, c := range p.Library() {
		if c.ID() != card.ID() {
			remaining = append(remaining, c)
		}
	}
	p.SetLibrary(remaining)

	ctx.Game.registerBattlefieldEntry(perm, nil, uuid.Nil)
	ctx.Game.completeBattlefieldEntryFrom(perm, p.PlayerID(), p.PlayerID(), 0, core.ZoneLibrary)
	return nil
}

type sacrificeManaResult struct{ binding *NumberBinding }

func (*sacrificeManaResult) Text() string                 { return "saved sacrifice mana value" }
func (*sacrificeManaResult) Properties() EffectProperties { return EffectProperties{} }
func (*sacrificeManaResult) Apply(*EffectContext) error   { return nil }
func (e *sacrificeManaResult) inspectDefinition() definitionNode {
	return definitionNode{Role: "sacrifice-mana-result", bindingWrite: e.binding, children: []definitionEdge{{"result", e.binding}}}
}

func (g *Game) moveLibraryFindToGraveyard(source uuid.UUID, card Card, visit uuid.UUID) {
	move := &CardZoneMoveAction{source: source, card: card.ID(), visit: visit, from: core.ZoneLibrary, destination: core.ZoneGraveyard, owner: card.Owner()}
	if result, ok := g.effects.ApplyReplacements(move, g).(*CardZoneMoveAction); ok && result != nil {
		g.performCardZoneMove(result)
	}
}
