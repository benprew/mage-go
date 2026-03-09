package interactive

import (
	"time"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// SearchConfig controls the search parameters.
type SearchConfig struct {
	MaxDepth  int           // search depth in plies (default 3)
	MaxNodes  int           // node budget (default 5000)
	TimeLimit time.Duration // per-decision time budget (default 500ms)
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
	Evaluator StateEvaluator
	Fallback  *HeuristicStrategy
}

const (
	minScore = -1000000
	maxScore = 1000000
)

// PriorityAction generates legal moves and searches to find the best one.
func (s *SearchStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction {
	moves := GeneratePriorityMoves(g, p, landsPlayed, mainPhase)
	if len(moves) <= 1 {
		// Only pass available
		return PriorityAction{Type: ActionPass}
	}

	nodes := 0
	deadline := time.Now().Add(s.Config.TimeLimit)

	bestScore := minScore
	var bestMove *Move

	for i := range moves {
		m := &moves[i]
		if m.Type == ActionPass {
			continue
		}

		// Clone and apply move
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

	// If search found nothing useful or timed out with no result, fall back
	if bestMove == nil {
		return s.Fallback.PriorityAction(p, g, landsPlayed, mainPhase)
	}

	// Check if the best move is actually better than passing.
	// Land plays and creature casts are always worth taking (develop the board
	// even if the static eval doesn't capture the full long-term value).
	passScore := s.eval(g, p.PlayerID())
	if bestScore < passScore && bestMove.Type != ActionPlayLand && !bestMove.IsCreature {
		return PriorityAction{Type: ActionPass}
	}

	return moveToAction(bestMove)
}

// Attackers evaluates attacker subsets via search to find the best one.
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

		// Simulate attack damage for this set
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

// Blockers falls back to heuristic (blocking combinatorics too complex for v1).
func (s *SearchStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	return s.Fallback.Blockers(p, g)
}

// minimax performs minimax search with alpha-beta pruning.
func (s *SearchStrategy) minimax(g *mage.Game, depth, alpha, beta int,
	maximizing bool, playerID uuid.UUID, nodes *int, deadline time.Time) int {

	*nodes++

	// Terminal conditions
	if g.IsGameOver() {
		winner := gameWinner(g, playerID)
		if winner == 1 {
			return maxScore - (s.Config.MaxDepth - depth) // prefer faster wins
		}
		if winner == -1 {
			return minScore + (s.Config.MaxDepth - depth) // prefer slower losses
		}
		return 0
	}

	if depth <= 0 || *nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
		return s.eval(g, playerID)
	}

	// Determine whose turn it is for move generation
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

	// Generate moves for the current player
	moves := GeneratePriorityMoves(g, movePlayer, g.LandsPlayedThisTurn, g.Step.IsMainPhase())

	if maximizing {
		best := minScore
		for i := range moves {
			m := &moves[i]
			if m.Type == ActionPass {
				// Evaluate passing as the current position
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

	// Minimizing
	best := maxScore
	for i := range moves {
		m := &moves[i]
		if m.Type == ActionPass {
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

// eval scores the game state from playerID's perspective.
func (s *SearchStrategy) eval(g *mage.Game, playerID uuid.UUID) int {
	return s.Evaluator(g, playerID)
}

// gameWinner returns 1 if playerID won, -1 if they lost, 0 if neither.
func gameWinner(g *mage.Game, playerID uuid.UUID) int {
	for _, p := range g.Players {
		if !p.IsAlive() {
			if p.PlayerID() == playerID {
				return -1 // we lost
			}
			return 1 // opponent lost
		}
	}
	return 0
}

// moveToAction converts a Move to a PriorityAction.
func moveToAction(m *Move) PriorityAction {
	return PriorityAction{
		Type:         m.Type,
		CardID:       m.CardID,
		CardName:     m.CardName,
		Targets:      m.Targets,
		PermanentID:  m.PermanentID,
		AbilityIndex: m.AbilityIndex,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Game cloning for search
// ──────────────────────────────────────────────────────────────────────────────

// cloneGameForSearch creates a lightweight copy of the game state suitable
// for search. It copies players (life, hand, library size), battlefield
// permanents (tapped state, P/T, damage, counters, attrs), and basic game
// fields (turn, step, lands played). It does NOT copy the stack, triggers,
// continuous effects, or combat state — search uses simplified move application.
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

	// Clone battlefield
	clone.Battlefield = make([]*mage.Permanent, len(g.Battlefield))
	for i, perm := range g.Battlefield {
		cp := clonePermanent(perm, pa, pb, g)
		clone.Battlefield[i] = cp
	}

	return clone
}

// clonePlayer creates a new BasePlayer with copied state.
func clonePlayer(p mage.Player) *mage.BasePlayer {
	bp := mage.NewBasePlayerWithID(p.PlayerID(), p.Name())
	bp.SetLife(p.Life())

	// Copy hand
	hand := make([]mage.Card, len(p.Hand()))
	copy(hand, p.Hand())
	bp.SetHand(hand)

	// Copy library (just preserve count — search doesn't draw)
	lib := make([]mage.Card, len(p.Library()))
	copy(lib, p.Library())
	bp.SetLibrary(lib)

	return bp
}

// clonePermanent copies a permanent for the search clone.
// NewPermanent already populates baseAttrs from the card, so we only need
// to sync mutable state (tapped, damage, counters) and handle summoning sickness.
func clonePermanent(perm *mage.Permanent, pa, pb *mage.BasePlayer, origGame *mage.Game) *mage.Permanent {
	cp := mage.NewPermanent(perm.Card, perm.Controller)
	// Override the card ID to match the original so FindPermanent works
	cp.Card.SetID(perm.ID())
	cp.Tapped = perm.Tapped
	cp.Damage = perm.Damage
	cp.PhasedOut = perm.PhasedOut
	cp.FaceDown = perm.FaceDown
	cp.AttachedTo = perm.AttachedTo
	cp.BasePTOverride = perm.BasePTOverride
	cp.ChosenColor = perm.ChosenColor
	cp.ChosenPlayer = perm.ChosenPlayer

	// Copy counters
	for ct, n := range perm.Counters {
		cp.AddCounter(ct, n)
	}

	// NewPermanent sets summoning sickness by default; clear it if original doesn't have it
	if !perm.HasAttr(core.AttrSummonSick) {
		cp.RevokeBaseAttr(core.AttrSummonSick)
	}

	return cp
}

// ──────────────────────────────────────────────────────────────────────────────
// Simplified move application on clones
// ──────────────────────────────────────────────────────────────────────────────

// applyMoveToClone applies a move to a cloned game state in a simplified way.
// This does NOT run the full game loop — it directly manipulates state.
func applyMoveToClone(g *mage.Game, playerID uuid.UUID, m *Move, landsPlayed int) {
	switch m.Type {
	case ActionPlayLand:
		applyLandPlay(g, playerID, m)
	case ActionCastSpell:
		applySpellCast(g, playerID, m)
	case ActionActivateAbility:
		applyAbilityActivation(g, playerID, m)
	}
	// Run state-based actions to clean up dead creatures
	g.CheckStateBasedActions()
}

// applyLandPlay puts a land from hand onto the battlefield.
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

// applySpellCast approximates casting a spell on the clone:
// 1. Remove card from hand
// 2. Tap lands to pay cost
// 3. For creatures: put on battlefield
// 4. For damage spells: deal damage to target
// 5. For removal: destroy target
func applySpellCast(g *mage.Game, playerID uuid.UUID, m *Move) {
	p := g.GetPlayer(playerID)
	if p == nil {
		return
	}

	// Find and remove card from hand
	card, ok := p.RemoveFromHand(m.CardID)
	if !ok {
		return
	}

	// Pay mana cost by tapping lands
	mc := card.ManaCost()
	if !mc.IsZero() {
		_ = g.AutoTapForCost(playerID, mc)
		_ = p.ManaPool().Pay(mc)
	}

	// Apply effects based on card type
	if card.HasType(core.TypeCreature) {
		// Put creature on battlefield
		perm := mage.NewPermanent(card, playerID)
		perm.GrantBaseAttr(core.AttrSummonSick)
		g.Battlefield = append(g.Battlefield, perm)
		return
	}

	// Non-creature spells: examine effects
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, e := range sa.Effects() {
			props := e.Properties()

			// Damage effects
			if props.DamageValue != nil && len(m.Targets) > 0 {
				dmg := props.DamageValue.Resolve(g, card.ID(), playerID)
				targetID := m.Targets[0]
				// Check if target is a player
				if tp := g.GetPlayer(targetID); tp != nil {
					tp.LoseLife(dmg)
				} else if perm := g.FindPermanent(targetID); perm != nil {
					perm.Damage += dmg
				}
				return
			}

			// Destruction effects
			if props.Outcome == mage.OutcomeDetriment && props.DamageValue == nil && !props.Mass {
				if len(m.Targets) > 0 {
					if perm := g.FindPermanent(m.Targets[0]); perm != nil {
						// Remove from battlefield (simplified destroy)
						removePermanentFromBattlefield(g, perm)
					}
				}
				return
			}

			// Draw effects
			if props.DrawCount > 0 {
				// Just add cards to hand count conceptually
				for i := 0; i < props.DrawCount; i++ {
					p.DrawCard()
				}
				return
			}

			// Mass removal
			if props.Mass && props.Outcome == mage.OutcomeDetriment {
				// Destroy all creatures (simplified Wrath)
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

	// If we couldn't determine the effect, put non-permanent spells in graveyard
	// (they're already removed from hand, just discard them)
}

// applyAbilityActivation approximates an ability activation on a clone.
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

	// Pay costs (simplified: just tap if tap cost)
	for _, c := range aa.Costs() {
		_ = c.Pay(perm.ID(), playerID, g)
	}

	// Apply effects
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

// applyAttackersToClone simulates an attack on a cloned game by dealing
// unblocked damage to the opponent.
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

// removePermanentFromBattlefield removes a permanent from the battlefield slice.
func removePermanentFromBattlefield(g *mage.Game, target *mage.Permanent) {
	for i, p := range g.Battlefield {
		if p.ID() == target.ID() {
			g.Battlefield = append(g.Battlefield[:i], g.Battlefield[i+1:]...)
			return
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Constructor
// ──────────────────────────────────────────────────────────────────────────────

// NewSearchAI creates an AI player that uses minimax search with alpha-beta pruning.
// It falls back to heuristic decisions for blocking and when search times out.
func NewSearchAI(name string, config SearchConfig, personality Personality) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy: &SearchStrategy{
			Config:    config,
			Evaluator: DefaultEvaluator,
			Fallback:  &HeuristicStrategy{Personality: personality},
		},
	}
}
