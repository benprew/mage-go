package gametest

import (
	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// PlayerRef is used in test harness to refer to players.
type PlayerRef int

const (
	PlayerA PlayerRef = iota
	PlayerB
)

// ScriptedAction represents a scripted action for the test player.
type ScriptedAction struct {
	Turn     int
	Step     core.PhaseStep
	Action   string
	CardName string
	Targets  []string
}

// blockPair represents a single blocker → attacker assignment.
type blockPair struct {
	blocker  string
	attacker string
}

// TestPlayer is a scripted player for testing.
type TestPlayer struct {
	*mage.BasePlayer
	actions                   []ScriptedAction
	attackActions             map[int][]string
	blockActions              map[int][]blockPair
	bandFormations            map[int][][]string
	choosePermanent           []string
	chooseDiscard             [][]string
	chooseManaColor           []core.Color
	chooseFromLibrary         []string
	chooseBandingDistribution []map[string]int
	chooseMode                []int
	chooseNumber              []int
	combatBlockerOrder        map[string][]string
	combatDamageAssignment    map[string]map[string]int
}

func NewTestPlayer(name string) *TestPlayer {
	return &TestPlayer{
		BasePlayer:             mage.NewBasePlayer(name),
		attackActions:          make(map[int][]string),
		blockActions:           make(map[int][]blockPair),
		bandFormations:         make(map[int][][]string),
		combatBlockerOrder:     make(map[string][]string),
		combatDamageAssignment: make(map[string]map[string]int),
	}
}

// SetCombatBlockerOrder records the damage-assignment order the attacker's
// controller will choose for the named attacker (CR 510.1c).
func (tp *TestPlayer) SetCombatBlockerOrder(attackerName string, blockerNames []string) {
	tp.combatBlockerOrder[attackerName] = blockerNames
}

// SetCombatDamageAssignment records the per-blocker damage split the attacker's
// controller will assign for the named attacker (CR 510.1c).
func (tp *TestPlayer) SetCombatDamageAssignment(attackerName string, distribution map[string]int) {
	tp.combatDamageAssignment[attackerName] = distribution
}

// GetBlockerOrder implements mage.CombatDamageAssigner.
func (tp *TestPlayer) GetBlockerOrder(attacker *mage.Permanent, blockers []*mage.Permanent) []uuid.UUID {
	names, ok := tp.combatBlockerOrder[attacker.Name()]
	if !ok {
		return nil
	}
	idByName := make(map[string]uuid.UUID, len(blockers))
	for _, b := range blockers {
		idByName[b.Name()] = b.ID()
	}
	out := make([]uuid.UUID, 0, len(names))
	for _, n := range names {
		if id, ok := idByName[n]; ok {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// GetCombatDamageAssignment implements mage.CombatDamageAssigner.
func (tp *TestPlayer) GetCombatDamageAssignment(attacker *mage.Permanent, blockers []*mage.Permanent, totalPower int) map[uuid.UUID]int {
	dist, ok := tp.combatDamageAssignment[attacker.Name()]
	if !ok {
		return nil
	}
	out := make(map[uuid.UUID]int, len(dist))
	for _, b := range blockers {
		if dmg, ok := dist[b.Name()]; ok {
			out[b.ID()] = dmg
		}
	}
	return out
}

// AddBandFormation records that the given creatures should attack as a band on the given turn.
func (tp *TestPlayer) AddBandFormation(turn int, creatures []string) {
	tp.bandFormations[turn] = append(tp.bandFormations[turn], creatures)
}

// GetBandFormations implements BandFormer. Resolves scripted creature names to IDs.
func (tp *TestPlayer) GetBandFormations(turn int, g *mage.Game) [][]uuid.UUID {
	formations, ok := tp.bandFormations[turn]
	if !ok {
		return nil
	}
	var result [][]uuid.UUID
	for _, names := range formations {
		var ids []uuid.UUID
		for _, name := range names {
			perm := g.FindPermanentByName(name, tp.PlayerID())
			if perm != nil {
				ids = append(ids, perm.ID())
			}
		}
		if len(ids) >= 2 {
			result = append(result, ids)
		}
	}
	return result
}

// GetBandingDamageDistribution returns the scripted damage distribution for a banded group.
func (tp *TestPlayer) GetBandingDamageDistribution(members []*mage.Permanent) map[uuid.UUID]int {
	if len(tp.chooseBandingDistribution) == 0 {
		return nil
	}
	choice := tp.chooseBandingDistribution[0]
	tp.chooseBandingDistribution = tp.chooseBandingDistribution[1:]
	result := make(map[uuid.UUID]int)
	for _, m := range members {
		if dmg, ok := choice[m.Name()]; ok {
			result[m.ID()] = dmg
		}
	}
	return result
}

func (tp *TestPlayer) AddAction(a ScriptedAction) {
	tp.actions = append(tp.actions, a)
}

func (tp *TestPlayer) SetAttackers(turn int, creatures []string) {
	tp.attackActions[turn] = creatures
}

func (tp *TestPlayer) SetBlockers(turn int, blockers map[string]string) {
	for blocker, attacker := range blockers {
		tp.blockActions[turn] = append(tp.blockActions[turn], blockPair{blocker, attacker})
	}
}

// DeclareAttackers returns the list of creature IDs to attack with.
func (tp *TestPlayer) DeclareAttackers(g *mage.Game) []uuid.UUID {
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
func (tp *TestPlayer) DeclareBlockers(g *mage.Game) []mage.BlockAssignment {
	pairs, ok := tp.blockActions[g.Turn]
	if !ok {
		return nil
	}
	var result []mage.BlockAssignment
	for _, bp := range pairs {
		blocker := g.FindPermanentByName(bp.blocker, tp.PlayerID())
		var attacker *mage.Permanent
		for _, p := range g.Battlefield {
			if p.Name() == bp.attacker && g.Combat.IsAttacking(p.ID()) {
				attacker = p
				break
			}
		}
		if blocker != nil && attacker != nil {
			result = append(result, mage.BlockAssignment{
				BlockerID:  blocker.ID(),
				AttackerID: attacker.ID(),
			})
		}
	}
	return result
}

// ChooseTargets selects from possible targets (for auto-targeting).
func (tp *TestPlayer) ChooseTargets(possible []uuid.UUID, min, max int, g *mage.Game) []uuid.UUID {
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

// ChooseMode picks a mode from a list of options.
func (tp *TestPlayer) ChooseMode(modes []string, reason string) int {
	if len(tp.chooseMode) > 0 {
		choice := tp.chooseMode[0]
		tp.chooseMode = tp.chooseMode[1:]
		return choice
	}
	return 0
}

// ChoosePermanent picks a permanent from candidates.
func (tp *TestPlayer) ChoosePermanent(candidates []*mage.Permanent, reason string, g mage.GameReader) *mage.Permanent {
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

// ChooseCardsFromHand picks cards from hand by name.
func (tp *TestPlayer) ChooseCardsFromHand(amount int, reason string, g mage.GameReader) []mage.Card {
	hand := tp.Hand()
	if amount > len(hand) {
		amount = len(hand)
	}
	if len(tp.chooseDiscard) > 0 {
		names := tp.chooseDiscard[0]
		tp.chooseDiscard = tp.chooseDiscard[1:]
		var result []mage.Card
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
	result := make([]mage.Card, amount)
	copy(result, hand[:amount])
	return result
}

// ChooseManaColor picks a mana color.
func (tp *TestPlayer) ChooseManaColor(reason string) core.Color {
	if len(tp.chooseManaColor) > 0 {
		c := tp.chooseManaColor[0]
		tp.chooseManaColor = tp.chooseManaColor[1:]
		return c
	}
	return core.White
}

// ChooseCardFromLibrary picks a card from candidates.
func (tp *TestPlayer) ChooseCardFromLibrary(candidates []mage.Card, reason string, g mage.GameReader) mage.Card {
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

// ChooseNumber picks a number from the given range.
func (tp *TestPlayer) ChooseNumber(min, max int, reason string) int {
	if len(tp.chooseNumber) > 0 {
		n := tp.chooseNumber[0]
		tp.chooseNumber = tp.chooseNumber[1:]
		if n < min {
			return min
		}
		if n > max {
			return max
		}
		return n
	}
	return max // default: choose maximum
}
