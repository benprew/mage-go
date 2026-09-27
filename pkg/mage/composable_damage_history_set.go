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

type turnDamageHistoryPermanents struct {
	source     DamageSourceRef
	candidates PermanentSet
}

// PermanentsDealtDamageByThisTurn selects current recipients of this source's turn damage.
func PermanentsDealtDamageByThisTurn(source DamageSourceRef, candidates PermanentSet) PermanentSet {
	return turnDamageHistoryPermanents{source: source, candidates: candidates}
}

func (turnDamageHistoryPermanents) permanentSet() {}

func (s turnDamageHistoryPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		return nil
	}
	source := s.source.resolveDamageSource(ctx)
	if source.Incarnation() == uuid.Nil {
		return nil
	}
	var selected []*Permanent
	for _, permanent := range s.candidates.resolvePermanents(ctx) {
		if permanent != nil && !permanent.PhasedOut && ctx.Game.damage.dealtDamageToPermanentThisTurn(source, capturedPermanent{permanent.ID(), permanent.incarnationID}) {
			selected = append(selected, permanent)
		}
	}
	return selected
}

func (s turnDamageHistoryPermanents) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents dealt damage by source this turn"}
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		n.problem = "supply a damage source and permanent set"
		return n
	}
	n.children = []definitionEdge{{"source", damageSourceDefinition{s.source}}, {"candidates", s.candidates}}
	return n
}

type reverseTurnDamageHistoryPermanents struct {
	recipient  PermanentRef
	candidates PermanentSet
}

// PermanentsThatDealtDamageToThisTurn selects current sources of damage to one permanent.
func PermanentsThatDealtDamageToThisTurn(recipient PermanentRef, candidates PermanentSet) PermanentSet {
	return reverseTurnDamageHistoryPermanents{recipient: recipient, candidates: candidates}
}

func (reverseTurnDamageHistoryPermanents) permanentSet() {}

func (s reverseTurnDamageHistoryPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.recipient) || nilDefinition(s.candidates) {
		return nil
	}
	var recipient capturedPermanent
	if _, ok := s.recipient.(sourceRef); ok && ctx.sourceCaptured {
		recipient = capturedPermanent{ctx.SourceID, ctx.sourceIncarnation}
	} else if p, ok := s.recipient.resolvePermanent(ctx); ok {
		recipient = capturedPermanent{p.ID(), p.incarnationID}
	}
	if recipient.id == uuid.Nil || recipient.incarnation == uuid.Nil {
		return nil
	}
	var selected []*Permanent
	for _, candidate := range s.candidates.resolvePermanents(ctx) {
		if candidate == nil || candidate.PhasedOut {
			continue
		}
		if ctx.Game.damage.permanentDealtDamageToThisTurn(capturedPermanent{candidate.ID(), candidate.incarnationID}, recipient) {
			selected = append(selected, candidate)
		}
	}
	return selected
}

func (s reverseTurnDamageHistoryPermanents) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents that dealt damage to recipient this turn"}
	if nilDefinition(s.recipient) || nilDefinition(s.candidates) {
		n.problem = "supply a damage recipient and permanent set"
		return n
	}
	n.children = []definitionEdge{{"recipient", s.recipient}, {"candidates", s.candidates}}
	return n
}
