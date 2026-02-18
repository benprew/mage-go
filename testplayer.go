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
	actions       []ScriptedAction
	attackActions map[int][]string             // turn → creature names to attack with
	blockActions  map[int]map[string]string     // turn → blocker name → attacker name
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

// DeclareBlockers returns the blocker-attacker assignments.
func (tp *TestPlayer) DeclareBlockers(g *Game) []BlockAssignment {
	blockers, ok := tp.blockActions[g.Turn]
	if !ok {
		return nil
	}
	var assignments []BlockAssignment
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
			assignments = append(assignments, BlockAssignment{BlockerID: blocker.ID(), AttackerID: attacker.ID()})
		}
	}
	return assignments
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
