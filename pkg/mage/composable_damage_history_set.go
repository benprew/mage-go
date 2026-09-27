package mage

import "github.com/google/uuid"

type damageHistoryPlayers struct {
	source     DamageSourceRef
	candidates PlayerSet
}

// PlayersDealtDamageBy selects candidates damaged by one permanent incarnation this game.
func PlayersDealtDamageBy(source DamageSourceRef, candidates PlayerSet) PlayerSet {
	return damageHistoryPlayers{source: source, candidates: candidates}
}

func (damageHistoryPlayers) playerSet() {}

func (s damageHistoryPlayers) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		return nil
	}
	source := s.source.resolveDamageSource(ctx)
	if source.Incarnation() == uuid.Nil {
		return nil
	}
	var selected []uuid.UUID
	for _, id := range s.candidates.resolvePlayers(ctx) {
		if ctx.Game.HasDealtDamageToPlayerFromSource(source, id) {
			selected = append(selected, id)
		}
	}
	return selected
}

func (s damageHistoryPlayers) inspectDefinition() definitionNode {
	n := definitionNode{Role: "player-set", Text: "players dealt damage by source this game"}
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		n.problem = "supply a damage source and player set"
		return n
	}
	n.children = []definitionEdge{{"source", damageSourceDefinition{s.source}}, {"candidates", s.candidates}}
	return n
}

type damageHistoryPermanents struct {
	source     DamageSourceRef
	candidates PermanentSet
}

// PermanentsDealtDamageBy selects candidates damaged by one permanent incarnation this game.
func PermanentsDealtDamageBy(source DamageSourceRef, candidates PermanentSet) PermanentSet {
	return damageHistoryPermanents{source: source, candidates: candidates}
}

func (damageHistoryPermanents) permanentSet() {}

func (s damageHistoryPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		return nil
	}
	source := s.source.resolveDamageSource(ctx)
	if source.Incarnation() == uuid.Nil {
		return nil
	}
	var selected []*Permanent
	for _, permanent := range s.candidates.resolvePermanents(ctx) {
		if permanent != nil && !permanent.PhasedOut && ctx.Game.HasDealtDamageToPermanentFromSource(source, permanent) {
			selected = append(selected, permanent)
		}
	}
	return selected
}

func (s damageHistoryPermanents) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents dealt damage by source this game"}
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		n.problem = "supply a damage source and permanent set"
		return n
	}
	n.children = []definitionEdge{{"source", damageSourceDefinition{s.source}}, {"candidates", s.candidates}}
	return n
}
