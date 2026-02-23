package interactive

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
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
	for _, perm := range g.Battlefield {
		tapped[perm.ID()] = perm.Tapped
	}
	return undoSnapshot{
		valid:          true,
		hand:           hand,
		manaPool:       pool,
		landsPlayed:    g.LandsPlayedThisTurn,
		battlefieldLen: len(g.Battlefield),
		stackSize:      g.Stack.Size(),
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
	g.LandsPlayedThisTurn = snap.landsPlayed
	if len(g.Battlefield) > snap.battlefieldLen {
		g.Battlefield = g.Battlefield[:snap.battlefieldLen]
	}
	for g.Stack.Size() > snap.stackSize {
		g.Stack.Pop()
	}
	for _, perm := range g.Battlefield {
		if was, ok := snap.tappedState[perm.ID()]; ok {
			perm.Tapped = was
		}
	}
}

// findPlayerIndex returns the index of the player with the given ID in g.Players.
func findPlayerIndex(g *mage.Game, id uuid.UUID) int {
	for i, p := range g.Players {
		if p.PlayerID() == id {
			return i
		}
	}
	return 0
}

func describeAction(action PriorityAction) string {
	switch action.Type {
	case ActionPlayLand:
		return fmt.Sprintf("plays %s", action.CardName)
	case ActionCastSpell:
		return fmt.Sprintf("casts %s", action.CardName)
	case ActionActivateAbility:
		return fmt.Sprintf("activates ability on %s", action.CardName)
	default:
		return "passes"
	}
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

func executeAction(g *mage.Game, playerID uuid.UUID, action PriorityAction, addLog func(string)) {
	var err error
	switch action.Type {
	case ActionPlayLand:
		err = g.PlayLand(playerID, action.CardID)
		if err == nil {
			p := g.GetPlayer(playerID)
			name := action.CardName
			if name == "" && p != nil {
				name = "a land"
			}
			addLog(fmt.Sprintf("%s plays %s", p.Name(), name))
		}
	case ActionCastSpell:
		err = g.CastSpellByID(playerID, action.CardID, action.Targets, action.XValue)
		if err == nil {
			p := g.GetPlayer(playerID)
			// Use the card name from the stack object — action.CardName may be a
			// display label like "Cast Giant Growth {G}" from the TUI option.
			name := action.CardName
			obj := g.Stack.Peek()
			if obj != nil && obj.Card != nil {
				name = obj.Card.Name()
			}
			addLog(fmt.Sprintf("%s casts %s%s", p.Name(), name, targetSuffix(g, action.Targets)))
		}
	case ActionActivateAbility:
		err = g.ActivateAbilityByIndex(playerID, action.PermanentID, action.AbilityIndex, action.Targets)
		if err == nil {
			addLog(fmt.Sprintf("Activated ability: %s%s", action.CardName, targetSuffix(g, action.Targets)))
		}
	}
	if err != nil {
		addLog(fmt.Sprintf("Error: %v", err))
	}
}

func getEligibleAttackers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.Battlefield {
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

func performAttack(g *mage.Game, attackerIDs []uuid.UUID, addLog func(string)) {
	defender := g.NonActivePlayerObj()
	active := g.ActivePlayerObj()
	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		if !atk.HasKeyword(core.Vigilance) {
			atk.Tapped = true
		}
		g.Combat.AddAttacker(id, defender.PlayerID())
		addLog(fmt.Sprintf("%s attacks with %s %d/%d",
			active.Name(), atk.Name(), atk.CurrentPower(g), atk.CurrentToughness(g)))
		g.FireEvent(core.GameEvent{
			Type:     core.EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
		})
	}
}

func getEligibleBlockers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.CanDeclareAsBlocker(g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func blockerOptions(g *mage.Game, eligible []*mage.Permanent) []ActionOption {
	var options []ActionOption
	for _, perm := range eligible {
		options = append(options, ActionOption{
			Type:        ActionSelectBlockers,
			Label:       fmt.Sprintf("%s %d/%d", perm.Name(), perm.CurrentPower(g), perm.CurrentToughness(g)),
			PermanentID: perm.ID(),
		})
	}
	options = append(options, ActionOption{
		Type:  ActionPass,
		Label: "Done (confirm blocks)",
	})
	return options
}

func performBlock(g *mage.Game, blockers []mage.BlockAssignment, addLog func(string)) {
	nonActive := g.NonActivePlayerObj()
	for _, ba := range blockers {
		blocker := g.FindPermanent(ba.BlockerID)
		attacker := g.FindPermanent(ba.AttackerID)
		if blocker == nil || attacker == nil {
			continue
		}
		if !mage.CanBlock(blocker, attacker, g) {
			continue
		}
		if mage.HasLandwalkEvasion(attacker, nonActive.PlayerID(), g) {
			continue
		}
		g.Combat.AddBlocker(ba.BlockerID, ba.AttackerID)
		addLog(fmt.Sprintf("%s blocks %s with %s",
			nonActive.Name(), attacker.Name(), blocker.Name()))
		g.FireEvent(core.GameEvent{
			Type:     core.EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: ba.AttackerID,
			PlayerID: nonActive.PlayerID(),
		})
	}
}

func reportCombatResults(g *mage.Game, humanIdx int, addLog func(string)) {
	for _, p := range g.Players {
		addLog(fmt.Sprintf("%s: %d life", p.Name(), p.Life()))
	}
}

// logCombatPreview logs a summary of each combat group before damage is dealt.
func logCombatPreview(g *mage.Game, addLog func(string)) {
	for _, grp := range g.Combat.Groups {
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
