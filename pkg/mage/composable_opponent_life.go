package mage

import "github.com/google/uuid"

type opponentsRef struct{ player PlayerRef }

// OpponentsOf selects the other player in this two-player engine.
// An absent player produces an empty set.
func OpponentsOf(player PlayerRef) PlayerSet { return &opponentsRef{player} }

func (*opponentsRef) playerSet() {}
func (s *opponentsRef) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	id, ok := s.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var opponents []uuid.UUID
	for _, candidate := range EachPlayer.resolvePlayers(ctx) {
		if candidate != id {
			opponents = append(opponents, candidate)
		}
	}
	return opponents
}
func (s *opponentsRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set", Text: "opponents of " + InspectDefinition(s.player).Text, children: []definitionEdge{{"player", s.player}}}
}

type loseLifeEachEffect struct {
	subjects PlayerSet
	amount   ResolutionValue
}

// LoseLifeEach makes each selected player lose the specified amount of life.
// The instruction reads the amount once in the current resolution frame.
// It captures the players and amount before it changes life totals.
func LoseLifeEach(subjects PlayerSet, amount ResolutionValue) ResolvingEffect {
	return &loseLifeEachEffect{subjects, amount}
}

func (e *loseLifeEachEffect) Text() string {
	amount := ""
	if !nilDefinition(e.amount) {
		amount = e.amount.Text()
	}
	return "lose " + amount + " life for " + InspectDefinition(e.subjects).Text
}
func (*loseLifeEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *loseLifeEachEffect) Apply(ctx *EffectContext) error {
	ids := e.subjects.resolvePlayers(ctx)
	if len(ids) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	players := make([]Player, 0, len(ids))
	for _, id := range ids {
		if player := ctx.Game.GetPlayer(id); player != nil {
			players = append(players, player)
		}
	}
	ctx.Game.playersLoseLife(players, amount)
	return nil
}
func (e *loseLifeEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "lose-life-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
}
