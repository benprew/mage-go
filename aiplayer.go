package mage

import "github.com/google/uuid"

// AIPlayer is a computer-controlled player with simple decision-making.
type AIPlayer struct {
	*BasePlayer
}

// NewAIPlayer creates a new AI player.
func NewAIPlayer(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: NewBasePlayer(name),
	}
}

// GetPriorityAction decides what the AI should do when it has priority.
func (ai *AIPlayer) GetPriorityAction(g *Game, landsPlayed int, mainPhase bool) PriorityAction {
	playerID := ai.PlayerID()

	if mainPhase {
		// Play a land if we haven't yet
		if landsPlayed < 1 {
			for _, c := range ai.Hand() {
				if c.HasType(TypeLand) {
					return PriorityAction{
						Type:     ActionPlayLand,
						CardID:   c.ID(),
						CardName: c.Name(),
					}
				}
			}
		}

		// Try to cast the most expensive affordable creature or sorcery
		var bestCard Card
		bestCMC := -1
		for _, card := range g.GetCastableSpells(playerID) {
			if card.HasType(TypeInstant) {
				continue // save instants for later
			}
			cmc := card.ManaCost().CMC()
			if cmc > bestCMC {
				bestCMC = cmc
				bestCard = card
			}
		}
		if bestCard != nil {
			targets := ai.autoSelectTargets(g, bestCard)
			return PriorityAction{
				Type:     ActionCastSpell,
				CardID:   bestCard.ID(),
				CardName: bestCard.Name(),
				Targets:  targets,
			}
		}
	}

	// Check if we should cast an instant (damage spell at opponent's creature or face)
	for _, card := range ai.Hand() {
		if !card.HasType(TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}
		// Only cast damage-dealing instants proactively
		hasDamage := false
		for _, a := range card.Abilities() {
			if sa, ok := a.(*SpellAbility); ok {
				for _, e := range sa.Effects() {
					if _, isDmg := e.(*dealDamageEffect); isDmg {
						hasDamage = true
					}
				}
			}
		}
		if hasDamage {
			targets := ai.autoSelectTargets(g, card)
			if len(targets) > 0 {
				return PriorityAction{
					Type:     ActionCastSpell,
					CardID:   card.ID(),
					CardName: card.Name(),
					Targets:  targets,
				}
			}
		}
	}

	return PriorityAction{Type: ActionPass}
}

// AIAttackers returns the IDs of creatures the AI wants to attack with.
func (ai *AIPlayer) AIAttackers(g *Game) []uuid.UUID {
	var attackers []uuid.UUID
	for _, perm := range g.Battlefield {
		if perm.Controller != ai.PlayerID() {
			continue
		}
		if !perm.HasType(TypeCreature) {
			continue
		}
		if perm.Tapped || (perm.SummonSick && !perm.HasAbility(Haste)) {
			continue
		}
		if !g.Effects.CanAttack(perm.ID()) {
			continue
		}
		if !CanAttackCheck(perm, g) {
			continue
		}
		attackers = append(attackers, perm.ID())
	}
	return attackers
}

// AIBlockers returns a blocker->attacker mapping for the AI's blocking decisions.
func (ai *AIPlayer) AIBlockers(g *Game) map[uuid.UUID]uuid.UUID {
	blockers := make(map[uuid.UUID]uuid.UUID)

	// Get available blockers
	var available []*Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != ai.PlayerID() || !perm.HasType(TypeCreature) || perm.Tapped {
			continue
		}
		available = append(available, perm)
	}

	// For each attacker, try to assign a blocker
	for _, group := range g.Combat.Groups {
		if group.DefenderID != ai.PlayerID() {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}

		// Find the first available blocker that can legally block
		for i, blk := range available {
			if blk == nil {
				continue
			}
			if !CanBlock(blk, atk, g) {
				continue
			}
			if HasLandwalkEvasion(atk, ai.PlayerID(), g) {
				continue
			}
			// Only block if our blocker can survive or trade
			atkPow := atk.CurrentPower(g)
			blkPow := blk.CurrentPower(g)
			atkTough := atk.CurrentToughness(g)

			// Block if we can kill the attacker or if the attacker would deal significant damage
			if blkPow >= atkTough || atkPow >= 3 {
				blockers[blk.ID()] = atk.ID()
				available[i] = nil // used
				break
			}
		}
	}

	return blockers
}

// DeclareAttackers implements the Player interface for AI.
func (ai *AIPlayer) DeclareAttackers(g *Game) []uuid.UUID {
	return ai.AIAttackers(g)
}

// DeclareBlockers implements the Player interface for AI.
func (ai *AIPlayer) DeclareBlockers(g *Game) map[uuid.UUID]uuid.UUID {
	return ai.AIBlockers(g)
}

// autoSelectTargets picks targets automatically for AI spells.
func (ai *AIPlayer) autoSelectTargets(g *Game, card Card) []uuid.UUID {
	playerID := ai.PlayerID()

	for _, a := range card.Abilities() {
		sa, ok := a.(*SpellAbility)
		if !ok {
			continue
		}
		for _, t := range sa.Targets() {
			possible := t.Possible(playerID, card, g)
			if len(possible) == 0 {
				return nil
			}

			switch t.(type) {
			case *AnyTargetImpl:
				// For damage spells: prefer opponent's creature, else opponent's face
				// First look for an opponent creature we can kill
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller != playerID && perm.HasType(TypeCreature) {
						return []uuid.UUID{id}
					}
				}
				// Otherwise target opponent
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}

			case *CreatureTarget:
				// Target opponent's best creature (highest power)
				var bestID uuid.UUID
				bestPow := -1
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller != playerID {
						pow := perm.CurrentPower(g)
						if pow > bestPow {
							bestPow = pow
							bestID = id
						}
					}
				}
				if bestID != uuid.Nil {
					return []uuid.UUID{bestID}
				}
				// If targeting own creature (buff spell), pick our strongest
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller == playerID {
						return []uuid.UUID{id}
					}
				}

			case *PlayerTarget, *OpponentTarget:
				// Target opponent
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}

			default:
				// Generic: pick first possible
				if len(possible) > 0 {
					return []uuid.UUID{possible[0]}
				}
			}
		}
	}

	return nil
}
