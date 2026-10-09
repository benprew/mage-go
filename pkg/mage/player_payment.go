package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PlayerManaPaymentThisTurn permits a repeatable payment at instant timing.
// The body keeps the source and references saved when the permission starts.
func PlayerManaPaymentThisTurn(player PlayerRef, mana core.ManaCost, body ResolvingEffect) ResolvingEffect {
	mana.Hybrid = slices.Clone(mana.Hybrid)
	return &playerPaymentInstallation{player: player, mana: mana, body: body}
}

type playerPaymentInstallation struct {
	player PlayerRef
	mana   core.ManaCost
	body   ResolvingEffect
}

func (*playerPaymentInstallation) Text() string                 { return "permit a mana payment this turn" }
func (*playerPaymentInstallation) Properties() EffectProperties { return EffectProperties{} }
func (e *playerPaymentInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "player-payment", Text: e.Text(), isolatedBindings: true, children: []definitionEdge{{"player", e.player}, {"body", e.body}}}
	if nilDefinition(e.player) || nilDefinition(e.body) {
		n.problem = "player payment requires a player and body"
	}
	if e.mana.HasX || e.mana.XCount != 0 || len(e.mana.Hybrid) != 0 || e.mana.Generic < 0 || e.mana.White < 0 || e.mana.Blue < 0 || e.mana.Black < 0 || e.mana.Red < 0 || e.mana.Green < 0 {
		n.problem = "player payment requires fixed nonnegative ordinary mana"
	}
	return n
}
func (e *playerPaymentInstallation) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.body)
}
func (e *playerPaymentInstallation) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	saved := *ctx
	saved.Game = nil
	saved.Targets = slices.Clone(ctx.Targets)
	saved.targetBindings = cloneDelayedTargets(ctx.targetBindings)
	saved.Vars = maps.Clone(ctx.Vars)
	saved.DamageDistribution = maps.Clone(ctx.DamageDistribution)
	saved.CounterDistribution = maps.Clone(ctx.CounterDistribution)
	if !saved.sourceCaptured {
		if p := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); p != nil {
			saved.sourceIncarnation = p.incarnationID
		}
	}
	saved.sourceCaptured = true
	frame := ctx.Game.resolution.Clone()
	frame.manaTracking = false
	frame.resolvingStackID = uuid.Nil
	frame.resolvingCastContext = ctx.Game.resolution.resolvingCastContext.clone()
	ctx.Game.playerPayments = append(ctx.Game.playerPayments, installedPlayerPayment{id: uuid.New(), player: player, mana: e.mana, body: e.body, context: saved, frame: frame})
	return nil
}

type installedPlayerPayment struct {
	id      uuid.UUID
	player  uuid.UUID
	mana    core.ManaCost
	body    ResolvingEffect
	context EffectContext
	frame   ResolutionState
}

// PlayerPaymentInfo identifies a current payment permission.
type PlayerPaymentInfo struct {
	ID          uuid.UUID
	SourceID    uuid.UUID
	Description string
}

// GetPlayerPayments lists affordable permissions without payment or choices.
// The caller must supply a time at which the player could cast an instant.
func (g *Game) GetPlayerPayments(player uuid.UUID) []PlayerPaymentInfo {
	if g.GetPlayer(player) == nil || g.resolution.resolvingController != uuid.Nil {
		return nil
	}
	var result []PlayerPaymentInfo
	for _, payment := range g.playerPayments {
		if payment.player == player && g.CanAfford(player, payment.mana, nil) {
			result = append(result, PlayerPaymentInfo{ID: payment.id, SourceID: payment.context.SourceID, Description: payment.body.Text()})
		}
	}
	return result
}

// TakePlayerPayment pays one permission and applies its body without the stack.
// The caller must supply a time at which the player could cast an instant.
func (g *Game) TakePlayerPayment(player, id uuid.UUID) error {
	if g.resolution.resolvingController != uuid.Nil {
		return fmt.Errorf("player payment requires instant timing")
	}
	for _, payment := range g.playerPayments {
		if payment.id != id {
			continue
		}
		if payment.player != player {
			return fmt.Errorf("player payment belongs to another player")
		}
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: player, SourceID: payment.context.SourceID, Costs: []Cost{&ManaCostPayment{MC: payment.mana}}})
		if err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return payment.applyBody(g)
	}
	return fmt.Errorf("player payment is absent or expired")
}

func (payment installedPlayerPayment) applyBody(g *Game) error {
	outer := g.resolution
	g.resolution = payment.frame.Clone()
	g.resolution.resolvingCastContext = payment.frame.resolvingCastContext.clone()
	defer func() { g.resolution = outer }()
	ctx := payment.context
	ctx.Game = g
	ctx.Targets = slices.Clone(ctx.Targets)
	ctx.targetBindings = cloneDelayedTargets(ctx.targetBindings)
	ctx.Vars = maps.Clone(ctx.Vars)
	ctx.DamageDistribution = maps.Clone(ctx.DamageDistribution)
	ctx.CounterDistribution = maps.Clone(ctx.CounterDistribution)
	return payment.body.Apply(&ctx)
}
