package interactive

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// HumanPlayer wraps BasePlayer for interactive TUI play. It overrides
// ChooseMode so the game loop can pre-set the mode before casting.
type HumanPlayer struct {
	*mage.BasePlayer
	PendingMode int
}

// NewHumanPlayer creates a new human player for the TUI.
func NewHumanPlayer(name string) *HumanPlayer {
	return &HumanPlayer{
		BasePlayer: mage.NewBasePlayer(name),
	}
}

func (p *HumanPlayer) ChooseMode(modes []string, reason string) int {
	mode := p.PendingMode
	p.PendingMode = 0
	return mode
}

// AIPlayer is a computer-controlled player with simple decision-making.
type AIPlayer struct {
	*mage.BasePlayer
}

// NewAIPlayer creates a new AI player.
func NewAIPlayer(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
	}
}

// GetPriorityAction decides what the AI should do when it has priority.
func (ai *AIPlayer) GetPriorityAction(g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction {
	playerID := ai.PlayerID()

	if mainPhase {
		if landsPlayed < 1 {
			for _, c := range ai.Hand() {
				if c.HasType(core.TypeLand) {
					return PriorityAction{
						Type:     ActionPlayLand,
						CardID:   c.ID(),
						CardName: c.Name(),
					}
				}
			}
		}

		var bestCard mage.Card
		bestCMC := -1
		for _, card := range g.GetCastableSpells(playerID) {
			if card.HasType(core.TypeInstant) {
				continue
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

	for _, card := range ai.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}
		hasDamage := false
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok {
				for _, e := range sa.Effects() {
					if mage.IsDamageEffect(e) {
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
func (ai *AIPlayer) AIAttackers(g *mage.Game) []uuid.UUID {
	var attackers []uuid.UUID
	for _, perm := range g.Battlefield {
		if perm.Controller != ai.PlayerID() {
			continue
		}
		if !perm.HasType(core.TypeCreature) {
			continue
		}
		if perm.Tapped || (perm.SummonSick && !perm.HasKeyword(core.Haste)) {
			continue
		}
		if !g.Effects.CanAttack(perm.ID()) {
			continue
		}
		if !mage.CanAttackCheck(perm, g) {
			continue
		}
		attackers = append(attackers, perm.ID())
	}
	return attackers
}

// AIBlockers returns a blocker->attacker mapping for the AI's blocking decisions.
func (ai *AIPlayer) AIBlockers(g *mage.Game) []mage.BlockAssignment {
	var assignments []mage.BlockAssignment

	var available []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != ai.PlayerID() || !perm.HasType(core.TypeCreature) || perm.Tapped {
			continue
		}
		available = append(available, perm)
	}

	for _, group := range g.Combat.Groups {
		if group.DefenderID != ai.PlayerID() {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}

		for i, blk := range available {
			if blk == nil {
				continue
			}
			if !mage.CanBlock(blk, atk, g) {
				continue
			}
			if mage.HasLandwalkEvasion(atk, ai.PlayerID(), g) {
				continue
			}
			atkPow := atk.CurrentPower(g)
			blkPow := blk.CurrentPower(g)
			atkTough := atk.CurrentToughness(g)

			if blkPow >= atkTough || atkPow >= 3 {
				assignments = append(assignments, mage.BlockAssignment{
					BlockerID:  blk.ID(),
					AttackerID: atk.ID(),
				})
				available[i] = nil
				break
			}
		}
	}

	return assignments
}

// DeclareAttackers implements the Player interface for AI.
func (ai *AIPlayer) DeclareAttackers(g *mage.Game) []uuid.UUID {
	return ai.AIAttackers(g)
}

// DeclareBlockers implements the Player interface for AI.
func (ai *AIPlayer) DeclareBlockers(g *mage.Game) []mage.BlockAssignment {
	return ai.AIBlockers(g)
}

func (ai *AIPlayer) autoSelectTargets(g *mage.Game, card mage.Card) []uuid.UUID {
	playerID := ai.PlayerID()

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, t := range sa.Targets() {
			possible := t.Possible(playerID, card, g)
			if len(possible) == 0 {
				return nil
			}

			switch t.(type) {
			case *mage.AnyTarget:
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller != playerID && perm.HasType(core.TypeCreature) {
						return []uuid.UUID{id}
					}
				}
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}

			case *mage.CreatureTarget:
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
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller == playerID {
						return []uuid.UUID{id}
					}
				}

			case *mage.PlayerTarget, *mage.OpponentTarget:
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}

			default:
				if len(possible) > 0 {
					return []uuid.UUID{possible[0]}
				}
			}
		}
	}

	return nil
}
