package ai

import (
	"time"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/interactive"
	"github.com/mage/mage/pkg/mage/interactive/eval"
)

// SearchConfig controls the search parameters.
type SearchConfig struct {
	MaxDepth  int
	MaxNodes  int
	TimeLimit time.Duration
}

// DefaultSearchConfig returns the default search configuration.
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		MaxDepth:  3,
		MaxNodes:  5000,
		TimeLimit: 500 * time.Millisecond,
	}
}

// SearchStrategy implements AIStrategy using minimax search with alpha-beta pruning.
type SearchStrategy struct {
	Config    SearchConfig
	Evaluator eval.StateEvaluator
	Fallback  *HeuristicStrategy
}

const (
	minScore = -1000000
	maxScore = 1000000
)

func (s *SearchStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	moves := GeneratePriorityMoves(g, p, landsPlayed, mainPhase)
	if len(moves) <= 1 {
		return interactive.PriorityAction{Type: interactive.ActionPass}
	}

	nodes := 0
	deadline := time.Now().Add(s.Config.TimeLimit)

	bestScore := minScore
	var bestMove *Move

	for i := range moves {
		m := &moves[i]
		if m.Type == interactive.ActionPass {
			continue
		}

		clone := cloneGameForSearch(g)
		if clone == nil {
			continue
		}
		applyMoveToClone(clone, p.PlayerID(), m, landsPlayed)

		score := s.minimax(clone, s.Config.MaxDepth-1, minScore, maxScore,
			false, p.PlayerID(), &nodes, deadline)

		if score > bestScore {
			bestScore = score
			bestMove = m
		}

		if nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
			break
		}
	}

	if bestMove == nil {
		return s.Fallback.PriorityAction(p, g, landsPlayed, mainPhase)
	}

	passScore := s.eval(g, p.PlayerID())
	if bestScore < passScore && bestMove.Type != interactive.ActionPlayLand && !bestMove.IsCreature {
		return interactive.PriorityAction{Type: interactive.ActionPass}
	}

	return moveToAction(bestMove)
}

func (s *SearchStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	sets := GenerateAttackerSets(g, p.PlayerID())
	if len(sets) <= 1 {
		if len(sets) == 1 {
			return sets[0]
		}
		return nil
	}

	nodes := 0
	deadline := time.Now().Add(s.Config.TimeLimit)

	bestScore := minScore
	var bestSet []uuid.UUID

	for _, set := range sets {
		clone := cloneGameForSearch(g)
		if clone == nil {
			continue
		}

		applyAttackersToClone(clone, p.PlayerID(), set)

		score := s.eval(clone, p.PlayerID())
		if score > bestScore {
			bestScore = score
			bestSet = set
		}

		nodes++
		if nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
			break
		}
	}

	if bestSet == nil {
		return s.Fallback.Attackers(p, g)
	}
	return bestSet
}

func (s *SearchStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	return s.Fallback.Blockers(p, g)
}

func (s *SearchStrategy) minimax(g *mage.Game, depth, alpha, beta int,
	maximizing bool, playerID uuid.UUID, nodes *int, deadline time.Time) int {

	*nodes++

	if g.IsGameOver() {
		winner := gameWinner(g, playerID)
		if winner == 1 {
			return maxScore - (s.Config.MaxDepth - depth)
		}
		if winner == -1 {
			return minScore + (s.Config.MaxDepth - depth)
		}
		return 0
	}

	if depth <= 0 || *nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
		return s.eval(g, playerID)
	}

	var movePlayerID uuid.UUID
	if maximizing {
		movePlayerID = playerID
	} else {
		opp := g.GetOpponent(playerID)
		if opp == nil {
			return s.eval(g, playerID)
		}
		movePlayerID = opp.PlayerID()
	}

	movePlayer := g.GetPlayer(movePlayerID)
	if movePlayer == nil {
		return s.eval(g, playerID)
	}

	moves := GeneratePriorityMoves(g, movePlayer, g.LandsPlayedThisTurn, g.Step.IsMainPhase())

	if maximizing {
		best := minScore
		for i := range moves {
			m := &moves[i]
			if m.Type == interactive.ActionPass {
				score := s.eval(g, playerID)
				if score > best {
					best = score
				}
				if best > alpha {
					alpha = best
				}
				if alpha >= beta {
					break
				}
				continue
			}

			clone := cloneGameForSearch(g)
			if clone == nil {
				continue
			}
			applyMoveToClone(clone, movePlayerID, m, g.LandsPlayedThisTurn)

			score := s.minimax(clone, depth-1, alpha, beta, false, playerID, nodes, deadline)
			if score > best {
				best = score
			}
			if best > alpha {
				alpha = best
			}
			if alpha >= beta {
				break
			}
		}
		return best
	}

	best := maxScore
	for i := range moves {
		m := &moves[i]
		if m.Type == interactive.ActionPass {
			score := s.eval(g, playerID)
			if score < best {
				best = score
			}
			if best < beta {
				beta = best
			}
			if alpha >= beta {
				break
			}
			continue
		}

		clone := cloneGameForSearch(g)
		if clone == nil {
			continue
		}
		applyMoveToClone(clone, movePlayerID, m, g.LandsPlayedThisTurn)

		score := s.minimax(clone, depth-1, alpha, beta, true, playerID, nodes, deadline)
		if score < best {
			best = score
		}
		if best < beta {
			beta = best
		}
		if alpha >= beta {
			break
		}
	}
	return best
}

func (s *SearchStrategy) eval(g *mage.Game, playerID uuid.UUID) int {
	return s.Evaluator(g, playerID)
}

func gameWinner(g *mage.Game, playerID uuid.UUID) int {
	for _, p := range g.Players {
		if !p.IsAlive() {
			if p.PlayerID() == playerID {
				return -1
			}
			return 1
		}
	}
	return 0
}

func moveToAction(m *Move) interactive.PriorityAction {
	return interactive.PriorityAction{
		Type:         m.Type,
		CardID:       m.CardID,
		CardName:     m.CardName,
		Targets:      m.Targets,
		PermanentID:  m.PermanentID,
		AbilityIndex: m.AbilityIndex,
	}
}

// ── Game cloning for search ─────────────────────────────────────────────────

func cloneGameForSearch(g *mage.Game) *mage.Game {
	if g == nil {
		return nil
	}

	pa := clonePlayer(g.Players[0])
	pb := clonePlayer(g.Players[1])

	clone := mage.NewGame(pa, pb)
	clone.Turn = g.Turn
	clone.Step = g.Step
	clone.ActivePlayer = g.ActivePlayer
	clone.LandsPlayedThisTurn = g.LandsPlayedThisTurn

	clone.Battlefield = make([]*mage.Permanent, len(g.Battlefield))
	for i, perm := range g.Battlefield {
		cp := clonePermanent(perm, pa, pb, g)
		clone.Battlefield[i] = cp
	}

	return clone
}

func clonePlayer(p mage.Player) *mage.BasePlayer {
	bp := mage.NewBasePlayerWithID(p.PlayerID(), p.Name())
	bp.SetLife(p.Life())

	hand := make([]mage.Card, len(p.Hand()))
	copy(hand, p.Hand())
	bp.SetHand(hand)

	lib := make([]mage.Card, len(p.Library()))
	copy(lib, p.Library())
	bp.SetLibrary(lib)

	return bp
}

func clonePermanent(perm *mage.Permanent, pa, pb *mage.BasePlayer, origGame *mage.Game) *mage.Permanent {
	cp := mage.NewPermanent(perm.Card, perm.Controller)
	cp.Card.SetID(perm.ID())
	cp.Tapped = perm.Tapped
	cp.Damage = perm.Damage
	cp.PhasedOut = perm.PhasedOut
	cp.FaceDown = perm.FaceDown
	cp.AttachedTo = perm.AttachedTo
	cp.BasePTOverride = perm.BasePTOverride
	cp.ChosenColor = perm.ChosenColor
	cp.ChosenPlayer = perm.ChosenPlayer

	for ct, n := range perm.Counters {
		cp.AddCounter(ct, n)
	}

	if !perm.HasAttr(core.AttrSummonSick) {
		cp.RevokeBaseAttr(core.AttrSummonSick)
	}

	return cp
}

// ── Simplified move application on clones ───────────────────────────────────

func applyMoveToClone(g *mage.Game, playerID uuid.UUID, m *Move, landsPlayed int) {
	switch m.Type {
	case interactive.ActionPlayLand:
		applyLandPlay(g, playerID, m)
	case interactive.ActionCastSpell:
		applySpellCast(g, playerID, m)
	case interactive.ActionActivateAbility:
		applyAbilityActivation(g, playerID, m)
	}
	g.CheckStateBasedActions()
}

func applyLandPlay(g *mage.Game, playerID uuid.UUID, m *Move) {
	p := g.GetPlayer(playerID)
	if p == nil {
		return
	}
	card, ok := p.RemoveFromHand(m.CardID)
	if !ok {
		return
	}
	perm := mage.NewPermanent(card, playerID)
	perm.GrantBaseAttr(core.AttrSummonSick)
	g.Battlefield = append(g.Battlefield, perm)
	g.LandsPlayedThisTurn++
}

func applySpellCast(g *mage.Game, playerID uuid.UUID, m *Move) {
	p := g.GetPlayer(playerID)
	if p == nil {
		return
	}

	card, ok := p.RemoveFromHand(m.CardID)
	if !ok {
		return
	}

	mc := card.ManaCost()
	if !mc.IsZero() {
		_ = g.AutoTapForCost(playerID, mc)
		_ = p.ManaPool().Pay(mc)
	}

	if card.HasType(core.TypeCreature) {
		perm := mage.NewPermanent(card, playerID)
		perm.GrantBaseAttr(core.AttrSummonSick)
		g.Battlefield = append(g.Battlefield, perm)
		return
	}

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, e := range sa.Effects() {
			props := e.Properties()

			if props.DamageValue != nil && len(m.Targets) > 0 {
				dmg := props.DamageValue.Resolve(g, card.ID(), playerID)
				targetID := m.Targets[0]
				if tp := g.GetPlayer(targetID); tp != nil {
					tp.LoseLife(dmg)
				} else if perm := g.FindPermanent(targetID); perm != nil {
					perm.Damage += dmg
				}
				return
			}

			if props.Outcome == mage.OutcomeDetriment && props.DamageValue == nil && !props.Mass {
				if len(m.Targets) > 0 {
					if perm := g.FindPermanent(m.Targets[0]); perm != nil {
						removePermanentFromBattlefield(g, perm)
					}
				}
				return
			}

			if props.DrawCount > 0 {
				for i := 0; i < props.DrawCount; i++ {
					p.DrawCard()
				}
				return
			}

			if props.Mass && props.Outcome == mage.OutcomeDetriment {
				var surviving []*mage.Permanent
				for _, perm := range g.Battlefield {
					if !perm.HasType(core.TypeCreature) {
						surviving = append(surviving, perm)
					}
				}
				g.Battlefield = surviving
				return
			}
		}
	}
}

func applyAbilityActivation(g *mage.Game, playerID uuid.UUID, m *Move) {
	perm := g.FindPermanent(m.PermanentID)
	if perm == nil {
		return
	}
	if m.AbilityIndex >= len(perm.RuntimeAbilities) {
		return
	}

	a := perm.RuntimeAbilities[m.AbilityIndex]
	aa, ok := a.(mage.ActivatedAbility)
	if !ok {
		return
	}

	for _, c := range aa.Costs() {
		_ = c.Pay(perm.ID(), playerID, g)
	}

	for _, e := range aa.Effects() {
		props := e.Properties()
		if props.DamageValue != nil && len(m.Targets) > 0 {
			dmg := props.DamageValue.Resolve(g, perm.ID(), playerID)
			targetID := m.Targets[0]
			if tp := g.GetPlayer(targetID); tp != nil {
				tp.LoseLife(dmg)
			} else if target := g.FindPermanent(targetID); target != nil {
				target.Damage += dmg
			}
		}
	}
}

func applyAttackersToClone(g *mage.Game, playerID uuid.UUID, attackers []uuid.UUID) {
	opp := g.GetOpponent(playerID)
	if opp == nil {
		return
	}

	totalDamage := 0
	for _, id := range attackers {
		perm := g.FindPermanent(id)
		if perm == nil {
			continue
		}
		totalDamage += perm.CurrentPower(g)
	}
	if totalDamage > 0 {
		opp.LoseLife(totalDamage)
	}
}

func removePermanentFromBattlefield(g *mage.Game, target *mage.Permanent) {
	for i, p := range g.Battlefield {
		if p.ID() == target.ID() {
			g.Battlefield = append(g.Battlefield[:i], g.Battlefield[i+1:]...)
			return
		}
	}
}

// NewSearchAI creates an AI player that uses minimax search.
func NewSearchAI(name string, config SearchConfig, personality Personality) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy: &SearchStrategy{
			Config:    config,
			Evaluator: eval.DefaultEvaluator,
			Fallback:  &HeuristicStrategy{Personality: personality},
		},
	}
}
