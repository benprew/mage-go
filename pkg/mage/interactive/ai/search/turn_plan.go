package search

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

type turnPlanKind uint64

const (
	turnPlanPriority turnPlanKind = iota + 1
	turnPlanAttackers
	turnPlanBlockers
)

const (
	turnPlanRootKey     uint64 = 0xa24baed4963ee407
	turnPlanActorKey    uint64 = 0x9fb21c651e98df25
	turnPlanPriorityKey uint64 = 0xc13fa9a902a6328f
	turnPlanAttackKey   uint64 = 0x91e10da5c79e7b1d
	turnPlanBlockKey    uint64 = 0xd192ed03a629c1b7
)

type turnPlan struct {
	turn         int
	rootPlayerID uuid.UUID
	steps        []ChainStep
}

func (s *Strategy) replayPriorityAction(p mage.Player, g *mage.Game) (interactive.PriorityAction, bool) {
	if s == nil || g == nil || p == nil || !s.turnPlanMatches(g, p.PlayerID()) {
		return interactive.PriorityAction{}, false
	}
	key := s.planDecisionKey(g, p.PlayerID(), turnPlanPriority)
	for _, step := range s.turnPlan.steps {
		if step.StateKey != key || step.PlayerID != p.PlayerID() || step.Move == nil {
			continue
		}
		if !priorityMoveStillLegal(g, p, step.Move) {
			return interactive.PriorityAction{}, false
		}
		s.LastNodes = 0
		return moveToAction(step.Move), true
	}
	return interactive.PriorityAction{}, false
}

func (s *Strategy) replayAttackers(p mage.Player, g *mage.Game) ([]uuid.UUID, bool) {
	if s == nil || g == nil || p == nil || !s.turnPlanMatches(g, p.PlayerID()) {
		return nil, false
	}
	key := s.planDecisionKey(g, p.PlayerID(), turnPlanAttackers)
	for _, step := range s.turnPlan.steps {
		if step.StateKey != key || step.PlayerID != p.PlayerID() || step.Attackers == nil {
			continue
		}
		s.LastNodes = 0
		return append([]uuid.UUID(nil), step.Attackers...), true
	}
	return nil, false
}

func (s *Strategy) replayBlockers(p mage.Player, g *mage.Game) ([]mage.BlockAssignment, bool) {
	if s == nil || g == nil || p == nil || !s.turnPlanMatches(g, p.PlayerID()) {
		return nil, false
	}
	key := s.planDecisionKey(g, p.PlayerID(), turnPlanBlockers)
	for _, step := range s.turnPlan.steps {
		if step.StateKey != key || step.PlayerID != p.PlayerID() || step.Blocks == nil {
			continue
		}
		s.LastNodes = 0
		return append([]mage.BlockAssignment(nil), step.Blocks...), true
	}
	return nil, false
}

func (s *Strategy) storeTurnPlan(g *mage.Game, rootPlayerID uuid.UUID, res Result) {
	if s == nil || g == nil || len(res.Chain) == 0 {
		if s != nil {
			s.turnPlan = nil
		}
		return
	}
	steps := make([]ChainStep, 0, len(res.Chain))
	for _, step := range res.Chain {
		if step.StateKey == 0 || step.PlayerID == (uuid.UUID{}) {
			continue
		}
		steps = append(steps, cloneChainStep(step))
	}
	if len(steps) == 0 {
		s.turnPlan = nil
		return
	}
	s.turnPlan = &turnPlan{
		turn:         g.CurrentTurn(),
		rootPlayerID: rootPlayerID,
		steps:        steps,
	}
}

func (s *Strategy) turnPlanMatches(g *mage.Game, rootPlayerID uuid.UUID) bool {
	return s.turnPlan != nil &&
		s.turnPlan.turn == g.CurrentTurn() &&
		s.turnPlan.rootPlayerID == rootPlayerID
}

func (s *Strategy) planDecisionKey(g *mage.Game, actorID uuid.UUID, kind turnPlanKind) uint64 {
	z := DefaultZobrist
	if s != nil && s.zobrist != nil {
		z = s.zobrist
	}
	return turnPlanDecisionKey(z, g, s.turnPlanRootPlayerID(actorID), actorID, kind)
}

func (s *Strategy) turnPlanRootPlayerID(fallback uuid.UUID) uuid.UUID {
	if s != nil && s.turnPlan != nil && s.turnPlan.rootPlayerID != (uuid.UUID{}) {
		return s.turnPlan.rootPlayerID
	}
	return fallback
}

func (s *searcher) planDecisionKey(g *mage.Game, actorID uuid.UUID, kind turnPlanKind) uint64 {
	z := s.zobrist
	if z == nil {
		z = DefaultZobrist
	}
	return turnPlanDecisionKey(z, g, s.rootPlayer.PlayerID(), actorID, kind)
}

func turnPlanDecisionKey(z *ZobristTables, g *mage.Game, rootPlayerID, actorID uuid.UUID, kind turnPlanKind) uint64 {
	if z == nil {
		z = DefaultZobrist
	}
	h := z.Hash(g)
	h ^= ttSupplementalKey(g)
	h ^= mixUUIDForPlan(rootPlayerID, turnPlanRootKey)
	h ^= mixUUIDForPlan(actorID, turnPlanActorKey)
	switch kind {
	case turnPlanPriority:
		h ^= turnPlanPriorityKey
	case turnPlanAttackers:
		h ^= turnPlanAttackKey
	case turnPlanBlockers:
		h ^= turnPlanBlockKey
	}
	return h
}

func mixUUIDForPlan(id uuid.UUID, seed uint64) uint64 {
	h := seed
	for _, b := range id {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

func cloneChainStep(step ChainStep) ChainStep {
	out := step
	if step.Move != nil {
		out.Move = cloneMove(step.Move)
	}
	if step.Attackers != nil {
		out.Attackers = append([]uuid.UUID{}, step.Attackers...)
	}
	if step.Blocks != nil {
		out.Blocks = append([]mage.BlockAssignment{}, step.Blocks...)
	}
	return out
}

func cloneMove(move *Move) *Move {
	if move == nil {
		return nil
	}
	out := *move
	out.Targets = append([]uuid.UUID(nil), move.Targets...)
	out.Attackers = append([]uuid.UUID(nil), move.Attackers...)
	return &out
}

func priorityMoveStillLegal(g *mage.Game, p mage.Player, move *Move) bool {
	if move == nil {
		return false
	}
	if move.Type == interactive.ActionPass {
		return true
	}
	for _, candidate := range legalPriorityMoves(g, p, true) {
		if sameMove(candidate, *move) {
			return true
		}
	}
	return false
}

func sameMove(a, b Move) bool {
	if a.Type != b.Type ||
		a.CardID != b.CardID ||
		a.PermanentID != b.PermanentID ||
		a.AbilityIndex != b.AbilityIndex ||
		a.XValue != b.XValue ||
		a.ModeIndex != b.ModeIndex {
		return false
	}
	if len(a.Targets) != len(b.Targets) {
		return false
	}
	for i := range a.Targets {
		if a.Targets[i] != b.Targets[i] {
			return false
		}
	}
	return true
}
