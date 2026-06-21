package interactive

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func captureForUndo(g *mage.Game, playerID uuid.UUID, logLen int) undoSnapshot {
	p := g.GetPlayer(playerID)
	if p == nil {
		return undoSnapshot{}
	}
	hand := make([]mage.Card, len(p.Hand()))
	copy(hand, p.Hand())
	pool := p.ManaPool().SnapshotPool()
	tapped := make(map[uuid.UUID]bool)
	for _, perm := range g.AllBattlefield() {
		tapped[perm.ID()] = perm.Tapped
	}
	return undoSnapshot{
		valid:          true,
		hand:           hand,
		manaPool:       pool,
		landsPlayed:    g.GetLandsPlayedThisTurn(),
		battlefieldLen: len(g.AllBattlefield()),
		stackSize:      g.StackSize(),
		tappedState:    tapped,
		logLen:         logLen,
	}
}

func restoreFromUndo(g *mage.Game, playerID uuid.UUID, snap undoSnapshot) {
	p := g.GetPlayer(playerID)
	if p == nil || !snap.valid {
		return
	}
	p.SetHand(snap.hand)
	p.ManaPool().RestorePool(snap.manaPool)
	g.SetLandsPlayedThisTurn(snap.landsPlayed)
	if len(g.AllBattlefield()) > snap.battlefieldLen {
		g.TruncateBattlefield(snap.battlefieldLen)
	}
	for g.StackSize() > snap.stackSize {
		g.GetStack().Pop()
	}
	for _, perm := range g.AllBattlefield() {
		if was, ok := snap.tappedState[perm.ID()]; ok {
			perm.Tapped = was
		}
	}
}

// findPlayerIndex returns the index of the player with the given ID in g.Players.
func findPlayerIndex(g *mage.Game, id uuid.UUID) int {
	for i, p := range g.AllPlayers() {
		if p.PlayerID() == id {
			return i
		}
	}
	return 0
}

func shouldAutoPassPriority(options []ActionOption) bool {
	return len(options) == 1 && options[0].Type == ActionPass
}

// resolveTargetName returns a human-readable name for a target UUID (permanent or player).
func resolveTargetName(g *mage.Game, id uuid.UUID) string {
	if perm := g.FindPermanent(id); perm != nil {
		return perm.Name()
	}
	if player := g.GetPlayer(id); player != nil {
		return player.Name()
	}
	return "unknown"
}

// targetSuffix builds " targeting X, Y" for a list of target IDs, or "" if none.
func targetSuffix(g *mage.Game, targets []uuid.UUID) string {
	if len(targets) == 0 {
		return ""
	}
	names := make([]string, len(targets))
	for i, id := range targets {
		names[i] = resolveTargetName(g, id)
	}
	return " targeting " + strings.Join(names, ", ")
}

func getEligibleAttackers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != playerID || !perm.CanDeclareAsAttacker(g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func attackerOptions(eligible []*mage.Permanent) []ActionOption {
	var options []ActionOption
	for _, perm := range eligible {
		options = append(options, ActionOption{
			Type:        ActionSelectAttackers,
			Label:       fmt.Sprintf("%s %d/%d", perm.Name(), perm.Card.Power(), perm.Card.Toughness()),
			PermanentID: perm.ID(),
		})
	}
	return options
}

func getEligibleBlockers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != playerID || !perm.CanDeclareAsBlocker(g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func blockerOptions(g *mage.Game, defenderID uuid.UUID, eligible []*mage.Permanent) []ActionOption {
	var options []ActionOption
	for _, perm := range eligible {
		opt := ActionOption{
			Type:        ActionSelectBlockers,
			Label:       fmt.Sprintf("%s %d/%d", perm.Name(), perm.CurrentPower(g), perm.CurrentToughness(g)),
			PermanentID: perm.ID(),
		}
		for _, group := range g.CombatGroups() {
			attacker := g.FindPermanent(group.AttackerID)
			if attacker == nil {
				continue
			}
			if mage.CanBlock(perm, attacker, g) && !mage.HasLandwalkEvasion(attacker, defenderID, g) {
				opt.ValidTargets = append(opt.ValidTargets, group.AttackerID)
			}
		}
		options = append(options, opt)
	}
	options = append(options, ActionOption{
		Type:  ActionPass,
		Label: "Done (confirm blocks)",
	})
	return options
}

func combatDamageOptions(g *mage.Game, attacker *mage.Permanent, blockers []*mage.Permanent, totalPower int) []ActionOption {
	opt := ActionOption{
		Type:        ActionAssignCombatDamage,
		Label:       fmt.Sprintf("%s assigns %d damage", attacker.Name(), totalPower),
		CardName:    attacker.Name(),
		PermanentID: attacker.ID(),
		MaxXValue:   totalPower,
	}
	for _, blocker := range blockers {
		opt.ValidTargets = append(opt.ValidTargets, blocker.ID())
		opt.ValidTargetLabels = append(opt.ValidTargetLabels,
			fmt.Sprintf("%s %d/%d", blocker.Name(), blocker.CurrentPower(g), blocker.CurrentToughness(g)))
	}
	return []ActionOption{opt}
}

func reportCombatResults(g *mage.Game, addLog func(string)) {
	for _, p := range g.AllPlayers() {
		addLog(fmt.Sprintf("%s: %d life", p.Name(), p.Life()))
	}
}

// logCombatPreview logs a summary of each combat group before damage is dealt.
func logCombatPreview(g *mage.Game, addLog func(string)) {
	for _, grp := range g.CombatGroups() {
		atk := g.FindPermanent(grp.AttackerID)
		if atk == nil {
			continue
		}
		defender := g.GetPlayer(grp.DefenderID)
		defName := "player"
		if defender != nil {
			defName = defender.Name()
		}
		if len(grp.BlockerIDs) == 0 {
			addLog(fmt.Sprintf("  %s (%d/%d) → %s unblocked",
				atk.Name(), atk.Card.Power(), atk.Card.Toughness(), defName))
		} else {
			parts := make([]string, 0, len(grp.BlockerIDs))
			for _, bid := range grp.BlockerIDs {
				blk := g.FindPermanent(bid)
				if blk != nil {
					parts = append(parts, fmt.Sprintf("%s (%d/%d)", blk.Name(), blk.Card.Power(), blk.Card.Toughness()))
				}
			}
			addLog(fmt.Sprintf("  %s (%d/%d) blocked by %s",
				atk.Name(), atk.Card.Power(), atk.Card.Toughness(), strings.Join(parts, ", ")))
		}
	}
}
