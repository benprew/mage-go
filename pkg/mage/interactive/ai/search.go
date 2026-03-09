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
		MaxDepth:  4,
		MaxNodes:  8000,
		TimeLimit: 750 * time.Millisecond,
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

// maxMoveChain limits sequential moves per player within a single search ply
// to prevent combinatorial explosion when exploring multi-spell turns.
const maxMoveChain = 4

func (s *SearchStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	moves := GeneratePriorityMoves(g, p, landsPlayed, mainPhase)
	if len(moves) <= 1 {
		return interactive.PriorityAction{Type: interactive.ActionPass}
	}

	deadline := time.Now().Add(s.Config.TimeLimit)

	// Iterative deepening: search at increasing depths, keeping the best
	// result found so far. This ensures we always have a result even if
	// deeper searches time out, and shallower searches improve move ordering.
	var bestMove *Move
	bestScore := minScore

	for depth := 1; depth <= s.Config.MaxDepth; depth++ {
		nodes := 0
		depthBestScore := minScore
		var depthBestMove *Move

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

			score := s.minimax(clone, depth-1, minScore, maxScore,
				false, p.PlayerID(), &nodes, deadline, 1)

			if score > depthBestScore {
				depthBestScore = score
				depthBestMove = m
			}

			if nodes >= s.Config.MaxNodes || time.Now().After(deadline) {
				break
			}
		}

		// Update overall best if this depth completed or found something better.
		if depthBestMove != nil && depthBestScore > bestScore {
			bestScore = depthBestScore
			bestMove = depthBestMove
		}

		if time.Now().After(deadline) {
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
	sets := generateBlockerSets(g, p.PlayerID(), s.Fallback)
	if len(sets) <= 1 {
		if len(sets) == 1 {
			return sets[0]
		}
		return s.Fallback.Blockers(p, g)
	}

	nodes := 0
	deadline := time.Now().Add(s.Config.TimeLimit)

	bestScore := minScore
	var bestSet []mage.BlockAssignment

	for _, set := range sets {
		clone := cloneGameForSearch(g)
		if clone == nil {
			continue
		}
		applyBlockersToClone(clone, set)
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
		return s.Fallback.Blockers(p, g)
	}
	return bestSet
}

// generateBlockerSets produces a set of candidate blocking assignments for search evaluation.
func generateBlockerSets(g *mage.Game, playerID uuid.UUID, fallback *HeuristicStrategy) [][]mage.BlockAssignment {
	var attackers []*mage.Permanent
	for _, group := range g.Combat.Groups {
		if group.DefenderID != playerID {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk != nil {
			attackers = append(attackers, atk)
		}
	}
	if len(attackers) == 0 {
		return nil
	}

	var blockers []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller == playerID && perm.CanDeclareAsBlocker(g) {
			blockers = append(blockers, perm)
		}
	}
	if len(blockers) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	var sets [][]mage.BlockAssignment

	// Option 1: No blocks.
	sets = append(sets, nil)

	// Option 2: Heuristic result.
	bp := mage.NewBasePlayerWithID(playerID, "search")
	heuristicBlocks := fallback.Blockers(bp, g)
	if len(heuristicBlocks) > 0 {
		sets = append(sets, heuristicBlocks)
	}

	// Option 3: Best single block per attacker.
	for _, atk := range attackers {
		for _, blk := range blockers {
			if !mage.CanBlock(blk, atk, g) {
				continue
			}
			if mage.HasLandwalkEvasion(atk, playerID, g) {
				continue
			}
			sets = append(sets, []mage.BlockAssignment{{
				BlockerID:  blk.ID(),
				AttackerID: atk.ID(),
			}})
		}
	}

	// Option 4: Gang block the biggest attacker + single block rest.
	if len(attackers) > 0 {
		// Find biggest attacker by power.
		biggest := attackers[0]
		for _, atk := range attackers[1:] {
			if atk.CurrentPower(g) > biggest.CurrentPower(g) {
				biggest = atk
			}
		}

		// Try assigning 2 blockers to the biggest.
		var gangSet []mage.BlockAssignment
		usedBlockers := make(map[uuid.UUID]bool)
		gangCount := 0
		for _, blk := range blockers {
			if gangCount >= 2 {
				break
			}
			if !mage.CanBlock(blk, biggest, g) {
				continue
			}
			if mage.HasLandwalkEvasion(biggest, playerID, g) {
				continue
			}
			gangSet = append(gangSet, mage.BlockAssignment{
				BlockerID:  blk.ID(),
				AttackerID: biggest.ID(),
			})
			usedBlockers[blk.ID()] = true
			gangCount++
		}
		// Single block remaining attackers with leftover blockers.
		if gangCount == 2 {
			for _, atk := range attackers {
				if atk.ID() == biggest.ID() {
					continue
				}
				for _, blk := range blockers {
					if usedBlockers[blk.ID()] {
						continue
					}
					if !mage.CanBlock(blk, atk, g) {
						continue
					}
					if mage.HasLandwalkEvasion(atk, playerID, g) {
						continue
					}
					gangSet = append(gangSet, mage.BlockAssignment{
						BlockerID:  blk.ID(),
						AttackerID: atk.ID(),
					})
					usedBlockers[blk.ID()] = true
					break
				}
			}
			sets = append(sets, gangSet)
		}
	}

	return sets
}

// applyBlockersToClone simulates the effect of a blocking assignment on a cloned game.
// It resolves combat damage and removes dead creatures.
func applyBlockersToClone(g *mage.Game, blocks []mage.BlockAssignment) {
	// Build blocker map: attacker -> list of blockers
	blockerMap := make(map[uuid.UUID][]uuid.UUID)
	blockedAttackers := make(map[uuid.UUID]bool)
	for _, b := range blocks {
		blockerMap[b.AttackerID] = append(blockerMap[b.AttackerID], b.BlockerID)
		blockedAttackers[b.AttackerID] = true
	}

	// Resolve damage: unblocked attackers deal damage to the defender.
	for _, group := range g.Combat.Groups {
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}
		if blockedAttackers[group.AttackerID] {
			// Blocked: apply combat damage between creatures.
			atkPow := atk.CurrentPower(g)
			remaining := atkPow
			for _, blkID := range blockerMap[group.AttackerID] {
				blk := g.FindPermanent(blkID)
				if blk == nil {
					continue
				}
				blkTough := blk.CurrentToughness(g)
				assigned := blkTough
				if assigned > remaining {
					assigned = remaining
				}
				blk.Damage += assigned
				remaining -= assigned
				// Blocker deals damage to attacker.
				atk.Damage += blk.CurrentPower(g)
			}
			// Trample excess.
			if remaining > 0 && atk.HasKeyword(core.Trample) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					defender.LoseLife(remaining)
				}
			}
		} else {
			// Unblocked: damage to defender.
			defender := g.GetPlayer(group.DefenderID)
			if defender != nil {
				defender.LoseLife(atk.CurrentPower(g))
			}
		}
	}

	// Remove dead creatures.
	var surviving []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.HasType(core.TypeCreature) && perm.Damage >= perm.CurrentToughness(g) {
			continue // dead
		}
		surviving = append(surviving, perm)
	}
	g.Battlefield = surviving
}

func (s *SearchStrategy) minimax(g *mage.Game, depth, alpha, beta int,
	maximizing bool, playerID uuid.UUID, nodes *int, deadline time.Time, chainCount int) int {

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
				// Pass ends the priority window — switch to opponent.
				score := s.minimax(g, depth-1, alpha, beta, false, playerID, nodes, deadline, 0)
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

			var score int
			if chainCount < maxMoveChain {
				// Multi-spell: generate more moves for the same player.
				score = s.minimax(clone, depth-1, alpha, beta, true, playerID, nodes, deadline, chainCount+1)
			} else {
				// Chain limit reached — switch to opponent.
				score = s.minimax(clone, depth-1, alpha, beta, false, playerID, nodes, deadline, 0)
			}
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
			// Pass ends the priority window — switch to maximizing player.
			score := s.minimax(g, depth-1, alpha, beta, true, playerID, nodes, deadline, 0)
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

		var score int
		if chainCount < maxMoveChain {
			// Multi-spell: generate more moves for the same player.
			score = s.minimax(clone, depth-1, alpha, beta, false, playerID, nodes, deadline, chainCount+1)
		} else {
			// Chain limit reached — switch to maximizing player.
			score = s.minimax(clone, depth-1, alpha, beta, true, playerID, nodes, deadline, 0)
		}
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
		XValue:       m.XValue,
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

	// Clone combat state so blocker evaluation can see attackers.
	for _, group := range g.Combat.Groups {
		clone.Combat.AddAttacker(group.AttackerID, group.DefenderID)
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

	// For X spells, set CurrentX on the clone so XValue resolution works.
	if m.XValue > 0 {
		g.CurrentX = m.XValue
	}

	if card.HasType(core.TypeCreature) {
		perm := mage.NewPermanent(card, playerID)
		perm.GrantBaseAttr(core.AttrSummonSick)
		g.Battlefield = append(g.Battlefield, perm)
		return
	}

	// Process ALL effects in order (no early returns).
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
				// For X spells, use the move's XValue if resolution returned 0
				// (CurrentX is set above, but as a fallback).
				if m.XValue > 0 && dmg == 0 {
					dmg = m.XValue
				}
				targetID := m.Targets[0]
				if tp := g.GetPlayer(targetID); tp != nil {
					tp.LoseLife(dmg)
				} else if perm := g.FindPermanent(targetID); perm != nil {
					perm.Damage += dmg
				}
				continue
			}

			// Bounce effects
			if props.IsBounce && len(m.Targets) > 0 {
				if perm := g.FindPermanent(m.Targets[0]); perm != nil {
					removePermanentFromBattlefield(g, perm)
					// Add card back to owner's hand
					owner := perm.Card.Owner()
					if owner == uuid.Nil {
						owner = perm.Controller
					}
					if ownerPlayer := g.GetPlayer(owner); ownerPlayer != nil {
						ownerPlayer.AddToHand(perm.Card)
					}
				}
				continue
			}

			// Single-target detriment (destroy, exile, etc.)
			if props.Outcome == mage.OutcomeDetriment && props.DamageValue == nil && !props.Mass && !props.IsBounce {
				if len(m.Targets) > 0 {
					if perm := g.FindPermanent(m.Targets[0]); perm != nil {
						removePermanentFromBattlefield(g, perm)
					}
				}
				continue
			}

			// Draw effects
			if props.DrawCount > 0 {
				for i := 0; i < props.DrawCount; i++ {
					p.DrawCard()
				}
				continue
			}

			// Life gain effects
			if props.LifeGain > 0 {
				p.GainLife(props.LifeGain)
				continue
			}

			// Buff effects (BoostUntilEndOfTurn)
			if (props.PowerBoost != 0 || props.ToughnessBoost != 0) && len(m.Targets) > 0 {
				if perm := g.FindPermanent(m.Targets[0]); perm != nil {
					perm.BoostPT(props.PowerBoost, props.ToughnessBoost)
				}
				continue
			}

			// Token creation effects
			if props.TokenPower > 0 || props.TokenToughness > 0 {
				token := mage.NewCreature("Token", "{0}", props.TokenPower, props.TokenToughness)
				token.SetOwner(playerID)
				perm := mage.NewPermanent(token, playerID)
				perm.GrantBaseAttr(core.AttrSummonSick)
				g.Battlefield = append(g.Battlefield, perm)
				continue
			}

			// Mass detriment (wrath effects)
			if props.Mass && props.Outcome == mage.OutcomeDetriment {
				var surviving []*mage.Permanent
				for _, perm := range g.Battlefield {
					if !perm.HasType(core.TypeCreature) {
						surviving = append(surviving, perm)
					}
				}
				g.Battlefield = surviving
				continue
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
