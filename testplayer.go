package mage

import "github.com/google/uuid"

// ScriptedAction represents a scripted action for the test player.
type ScriptedAction struct {
	Turn     int
	Step     PhaseStep
	Action   string   // "cast", "activate", "attack", "block"
	CardName string
	Targets  []string // target names (resolved to UUIDs at execution)
}

// TestPlayer is a scripted player for testing.
type TestPlayer struct {
	*BasePlayer
	actions            []ScriptedAction
	attackActions      map[int][]string         // turn → creature names to attack with
	blockActions       map[int]map[string]string // turn → blocker name → attacker name
	choosePermanent    []string                 // queue of permanent names
	chooseDiscard      [][]string               // queue of card name lists
	chooseManaColor    []Color                  // queue of colors
	chooseFromLibrary  []string                 // queue of card names
}

func NewTestPlayer(name string) *TestPlayer {
	return &TestPlayer{
		BasePlayer:   NewBasePlayer(name),
		attackActions: make(map[int][]string),
		blockActions:  make(map[int]map[string]string),
	}
}

func (tp *TestPlayer) AddAction(a ScriptedAction) {
	tp.actions = append(tp.actions, a)
}

func (tp *TestPlayer) SetAttackers(turn int, creatures []string) {
	tp.attackActions[turn] = creatures
}

func (tp *TestPlayer) SetBlockers(turn int, blockers map[string]string) {
	tp.blockActions[turn] = blockers
}

// DeclareAttackers returns the list of creature IDs to attack with.
func (tp *TestPlayer) DeclareAttackers(g *Game) []uuid.UUID {
	creatures, ok := tp.attackActions[g.Turn]
	if !ok {
		return nil
	}
	var ids []uuid.UUID
	for _, name := range creatures {
		p := g.FindPermanentByName(name, tp.PlayerID())
		if p != nil {
			ids = append(ids, p.ID())
		}
	}
	return ids
}

// DeclareBlockers returns blocker-attacker assignments for the current turn.
func (tp *TestPlayer) DeclareBlockers(g *Game) []BlockAssignment {
	blockers, ok := tp.blockActions[g.Turn]
	if !ok {
		return nil
	}
	var result []BlockAssignment
	for blockerName, attackerName := range blockers {
		blocker := g.FindPermanentByName(blockerName, tp.PlayerID())
		// Attacker could be controlled by any player
		var attacker *Permanent
		for _, p := range g.Battlefield {
			if p.Name() == attackerName && g.Combat.IsAttacking(p.ID()) {
				attacker = p
				break
			}
		}
		if blocker != nil && attacker != nil {
			result = append(result, BlockAssignment{
				BlockerID:  blocker.ID(),
				AttackerID: attacker.ID(),
			})
		}
	}
	return result
}

// ChooseTargets selects from possible targets (for auto-targeting).
func (tp *TestPlayer) ChooseTargets(possible []uuid.UUID, min, max int, g *Game) []uuid.UUID {
	if len(possible) >= min {
		n := min
		if n > len(possible) {
			n = len(possible)
		}
		return possible[:n]
	}
	return nil
}

// ChooseMayAbility always accepts optional abilities.
func (tp *TestPlayer) ChooseMayAbility(description string) bool {
	return true
}

// ChoosePermanent picks a permanent from candidates. If the queue has a scripted
// name, find the matching candidate; otherwise fall back to first candidate.
func (tp *TestPlayer) ChoosePermanent(candidates []*Permanent, reason string, g *Game) *Permanent {
	if len(candidates) == 0 {
		return nil
	}
	if len(tp.choosePermanent) > 0 {
		name := tp.choosePermanent[0]
		tp.choosePermanent = tp.choosePermanent[1:]
		for _, c := range candidates {
			if c.Name() == name {
				return c
			}
		}
	}
	return candidates[0]
}

// ChooseCardsFromHand picks cards from hand by name. If the queue has scripted
// names, find matching cards; otherwise fall back to first N cards.
func (tp *TestPlayer) ChooseCardsFromHand(amount int, reason string, g *Game) []Card {
	hand := tp.Hand()
	if amount > len(hand) {
		amount = len(hand)
	}
	if len(tp.chooseDiscard) > 0 {
		names := tp.chooseDiscard[0]
		tp.chooseDiscard = tp.chooseDiscard[1:]
		var result []Card
		for _, name := range names {
			for _, c := range hand {
				if c.Name() == name {
					result = append(result, c)
					break
				}
			}
		}
		if len(result) > amount {
			result = result[:amount]
		}
		return result
	}
	result := make([]Card, amount)
	copy(result, hand[:amount])
	return result
}

// ChooseManaColor picks a mana color. If the queue has a scripted color, use it;
// otherwise fall back to White.
func (tp *TestPlayer) ChooseManaColor(reason string) Color {
	if len(tp.chooseManaColor) > 0 {
		c := tp.chooseManaColor[0]
		tp.chooseManaColor = tp.chooseManaColor[1:]
		return c
	}
	return White
}

// ChooseCardFromLibrary picks a card from candidates. If the queue has a scripted
// name, find the matching candidate; otherwise fall back to first candidate.
func (tp *TestPlayer) ChooseCardFromLibrary(candidates []Card, reason string, g *Game) Card {
	if len(candidates) == 0 {
		return nil
	}
	if len(tp.chooseFromLibrary) > 0 {
		name := tp.chooseFromLibrary[0]
		tp.chooseFromLibrary = tp.chooseFromLibrary[1:]
		for _, c := range candidates {
			if c.Name() == name {
				return c
			}
		}
	}
	return candidates[0]
}
