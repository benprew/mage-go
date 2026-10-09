package mage

import "github.com/google/uuid"

// ExtraTurnFor adds a turn after this turn for the selected player.
// The most recent extra turn starts first. An absent player gets no turn.
func ExtraTurnFor(player PlayerRef) ResolvingEffect {
	return &extraTurnForEffect{player: player}
}

type extraTurnForEffect struct{ player PlayerRef }

func (*extraTurnForEffect) Text() string { return "take an extra turn after this one" }
func (*extraTurnForEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *extraTurnForEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}}}
}
func (e *extraTurnForEffect) Apply(ctx *EffectContext) error {
	if player := handOperationPlayer(e.player, ctx); player != nil {
		ctx.Game.GrantExtraTurn(player.PlayerID())
	}
	return nil
}
func (e *extraTurnForEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

// AdvanceToNextTurn selects the next player and increases the turn number.
// Extra turns keep the next normal player, including through nested extra turns.
// The caller must finish the current turn before it calls this method.
func (g *Game) AdvanceToNextTurn() {
	if g.turns.nextNormalPlayer == uuid.Nil {
		g.turns.nextNormalPlayer = g.players[(g.turns.ActivePlayerIndex()+1)%len(g.players)].PlayerID()
	}
	next := g.turns.nextNormalPlayer
	for {
		extra, ok := g.turns.PopExtraTurn()
		if !ok {
			g.turns.nextNormalPlayer = uuid.Nil
			break
		}
		if g.GetPlayer(extra) != nil {
			next = extra
			break
		}
	}
	for i, player := range g.players {
		if player.PlayerID() == next {
			g.turns.SetActivePlayerIndex(i)
			break
		}
	}
	g.turns.IncrementTurn()
}
