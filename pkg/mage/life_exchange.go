package mage

import (
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ExchangeLifeTotals saves both totals before it applies life changes.
// A prohibited gain or loss stops the complete exchange.
func ExchangeLifeTotals(first, second PlayerRef) ResolvingEffect {
	return &lifeExchangeEffect{first: first, second: second}
}

type lifeExchangeEffect struct{ first, second PlayerRef }

func (*lifeExchangeEffect) Text() string                 { return "exchange life totals" }
func (*lifeExchangeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *lifeExchangeEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "life-total-exchange", Text: e.Text(), children: []definitionEdge{{"first player", e.first}, {"second player", e.second}}}
}
func (e *lifeExchangeEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.first) || playerRefReadsSourcePermanent(e.second)
}
func (e *lifeExchangeEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	first, ok := e.first.resolvePlayer(ctx)
	if !ok || g.GetPlayer(first) == nil {
		return nil
	}
	second, ok := e.second.resolvePlayer(ctx)
	if !ok || first == second || g.GetPlayer(second) == nil {
		return nil
	}
	high, low := g.GetPlayer(first), g.GetPlayer(second)
	if high.Life() < low.Life() {
		high, low = low, high
	}
	if high.Life() == low.Life() || !g.CanLoseLife(high.PlayerID()) || !g.CanGainLife(low.PlayerID()) {
		return nil
	}
	if low.Life() < 0 && high.Life() > math.MaxInt+low.Life() {
		return fmt.Errorf("life exchange amount exceeds integer range")
	}
	amount := high.Life() - low.Life()
	gain, loss := 0, 0
	for _, player := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
		if player.PlayerID() == high.PlayerID() {
			loss = g.prepareLifeLoss(high, amount)
		} else if player.PlayerID() == low.PlayerID() {
			gain = g.prepareLifeGain(low, amount)
		}
	}
	if gain > 0 {
		low.GainLife(gain)
	}
	if loss > 0 {
		high.LoseLife(loss)
	}
	if loss > 0 {
		g.FireEvent(core.GameEvent{Type: core.EvtLifeLost, PlayerID: high.PlayerID(), Amount: loss})
	}
	if gain > 0 {
		g.FireEvent(core.GameEvent{Type: core.EvtLifeGained, PlayerID: low.PlayerID(), Amount: gain})
	}
	return nil
}

// AddCantGainLife prohibits a player's life gain for this layer cycle.
func (r *GameRules) AddCantGainLife(player uuid.UUID) {
	if r.cantGainLife == nil {
		r.cantGainLife = make(map[uuid.UUID]bool)
	}
	r.cantGainLife[player] = true
}

// AddCantLoseLife prohibits a player's life loss for this layer cycle.
func (r *GameRules) AddCantLoseLife(player uuid.UUID) {
	if r.cantLoseLife == nil {
		r.cantLoseLife = make(map[uuid.UUID]bool)
	}
	r.cantLoseLife[player] = true
}

// CanGainLife checks current life-gain permission without replacement choices.
func (g *Game) CanGainLife(player uuid.UUID) bool {
	return g.GetPlayer(player) != nil && !g.effects.Rules.cantGainLife[player]
}

// CanLoseLife checks current life-loss permission without replacement choices.
func (g *Game) CanLoseLife(player uuid.UUID) bool {
	return g.GetPlayer(player) != nil && !g.effects.Rules.cantLoseLife[player]
}

// LifeLossAction supplies direct life loss to the shared replacement service.
type LifeLossAction struct {
	source, playerID uuid.UUID
	amount           int
}

// NewLifeLossAction creates a proposed life loss.
func NewLifeLossAction(source, player uuid.UUID, amount int) *LifeLossAction {
	return &LifeLossAction{source: source, playerID: player, amount: amount}
}

func (a *LifeLossAction) ActionSource() uuid.UUID { return a.source }
func (a *LifeLossAction) PlayerID() uuid.UUID     { return a.playerID }
func (a *LifeLossAction) Amount() int             { return a.amount }

func (g *Game) prepareLifeLoss(player Player, amount int) int {
	if amount <= 0 || !g.CanLoseLife(player.PlayerID()) {
		return 0
	}
	result := g.effects.ApplyReplacements(NewLifeLossAction(uuid.Nil, player.PlayerID(), amount), g)
	if loss, ok := result.(*LifeLossAction); ok && loss.Amount() > 0 && g.CanLoseLife(player.PlayerID()) {
		return loss.Amount()
	}
	return 0
}
