package main

import (
	"math/rand"
	"sort"
)

// choosePriorityAction picks an action from the legal actions list.
// Priority: play land > cast spell (highest CMC first) > activate ability > pass.
// Propagates targets and X values for spells that need them.
func choosePriorityAction(legal []legalAction, state *cvState, rng *rand.Rand) actionMsg {
	// Play a land if possible
	for _, a := range legal {
		if a.Kind == "play_land" {
			return actionMsg{Type: "action", Kind: "play_land", CardName: a.CardName}
		}
	}

	// Collect castable spells, prefer highest CMC (biggest impact)
	type castCandidate struct {
		action  legalAction
		targets []string
	}
	var candidates []castCandidate
	for _, a := range legal {
		if a.Kind != "cast_spell" {
			continue
		}
		if len(a.Targets) > 0 {
			candidates = append(candidates, castCandidate{
				action:  a,
				targets: []string{a.Targets[0]},
			})
		} else {
			candidates = append(candidates, castCandidate{action: a})
		}
	}

	if len(candidates) > 0 {
		// Pick a random candidate among them (avoids always casting the same
		// first-alphabetical spell in deterministic order)
		pick := candidates[rng.Intn(len(candidates))]
		msg := actionMsg{
			Type:     "action",
			Kind:     "cast_spell",
			CardName: pick.action.CardName,
			Targets:  pick.targets,
		}
		if pick.action.HasX {
			x := pick.action.MaxX
			if x < 1 {
				x = 1
			}
			msg.XValue = x
		}
		return msg
	}

	// Try activating an ability
	for _, a := range legal {
		if a.Kind == "activate_ability" {
			msg := actionMsg{
				Type:          "action",
				Kind:          "activate_ability",
				PermanentName: a.PermanentName,
				AbilityIndex:  a.AbilityIndex,
			}
			if len(a.Targets) > 0 {
				msg.Targets = []string{a.Targets[0]}
			}
			return msg
		}
	}

	return actionMsg{Type: "action", Kind: "pass"}
}

// chooseAttackers picks creatures to attack with using a simple heuristic.
// Attacks with creatures whose power > 0, avoiding sending in 0-power creatures.
// When the opponent has no creatures, attacks with everything.
// When they do, avoids attacking with creatures that would die for free
// (power 0 creatures that can't trade).
func chooseAttackers(legal []legalAction, state *cvState) actionMsg {
	// Find the "attack" action(s) in legal — the legal list may contain
	// individual creature names as attackers, or a single action with a list.
	var eligible []string
	for _, a := range legal {
		if a.Kind == "attack" || a.Kind == "declare_attackers" {
			if a.CardName != "" {
				eligible = append(eligible, a.CardName)
			}
			eligible = append(eligible, a.Targets...)
		}
	}

	if len(eligible) == 0 {
		return actionMsg{Type: "action", Kind: "attack", Attackers: nil}
	}

	// Find opponent's creatures for blocking assessment
	activeIdx := 0
	if state != nil {
		activeIdx = state.ActivePlayerIdx
	}
	defIdx := 1 - activeIdx

	var defenderCreatures []cvPermanent
	if state != nil {
		for _, p := range state.Players[defIdx].Battlefield {
			if p.Power > 0 || p.Toughness > 0 {
				if !p.Tapped {
					defenderCreatures = append(defenderCreatures, p)
				}
			}
		}
	}

	// If opponent has no untapped creatures, attack with everything
	if len(defenderCreatures) == 0 {
		return actionMsg{Type: "action", Kind: "attack", Attackers: eligible}
	}

	// Find our creatures' P/T from battlefield
	var attackerPerms []cvPermanent
	if state != nil {
		for _, p := range state.Players[activeIdx].Battlefield {
			for _, name := range eligible {
				if p.Name == name {
					attackerPerms = append(attackerPerms, p)
					break
				}
			}
		}
	}

	// Sort defenders by toughness ascending to find smallest blocker
	sort.Slice(defenderCreatures, func(i, j int) bool {
		return defenderCreatures[i].Toughness < defenderCreatures[j].Toughness
	})
	smallestBlockerPower := 0
	if len(defenderCreatures) > 0 {
		smallestBlockerPower = defenderCreatures[0].Power
	}

	// Attack with creatures that either:
	// - Have power >= smallest blocker's toughness (can trade or win)
	// - Have toughness > smallest blocker's power (survive the block)
	var attackers []string
	for _, ap := range attackerPerms {
		if ap.Power <= 0 {
			continue
		}
		survives := ap.Toughness > smallestBlockerPower
		canTrade := ap.Power >= defenderCreatures[0].Toughness
		if survives || canTrade {
			attackers = append(attackers, ap.Name)
		}
	}

	// If our heuristic filtered everything, at least attack with the biggest creature
	if len(attackers) == 0 && len(attackerPerms) > 0 {
		best := attackerPerms[0]
		for _, ap := range attackerPerms[1:] {
			if ap.Power > best.Power {
				best = ap
			}
		}
		if best.Power > 0 {
			attackers = []string{best.Name}
		}
	}

	return actionMsg{Type: "action", Kind: "attack", Attackers: attackers}
}

// chooseBlockers picks blocking assignments. Tries to make profitable blocks:
// block attackers we can kill without losing our blocker, or trade up (block a
// bigger creature with a smaller one if our toughness survives or we at least trade).
func chooseBlockers(legal []legalAction, state *cvState) actionMsg {
	if state == nil {
		return actionMsg{Type: "action", Kind: "block"}
	}

	activeIdx := state.ActivePlayerIdx
	defIdx := 1 - activeIdx

	// Build maps of attacker and defender creatures from battlefield
	type creatureInfo struct {
		name      string
		power     int
		toughness int
	}

	var attackers []creatureInfo
	for _, p := range state.Players[activeIdx].Battlefield {
		if p.Tapped && (p.Power > 0 || p.Toughness > 0) {
			attackers = append(attackers, creatureInfo{p.Name, p.Power, p.Toughness})
		}
	}

	var blockers []creatureInfo
	for _, p := range state.Players[defIdx].Battlefield {
		if !p.Tapped && (p.Power > 0 || p.Toughness > 0) {
			blockers = append(blockers, creatureInfo{p.Name, p.Power, p.Toughness})
		}
	}

	if len(attackers) == 0 || len(blockers) == 0 {
		return actionMsg{Type: "action", Kind: "block"}
	}

	// Sort attackers by power descending — block the most dangerous first
	sort.Slice(attackers, func(i, j int) bool {
		return attackers[i].power > attackers[j].power
	})

	// Sort blockers by power ascending — use the smallest sufficient blocker
	sort.Slice(blockers, func(i, j int) bool {
		return blockers[i].power < blockers[j].power
	})

	used := make(map[string]bool)
	var pairs []blockerPair

	for _, att := range attackers {
		for _, blk := range blockers {
			if used[blk.name] {
				continue
			}
			// Block if we can kill the attacker
			canKill := blk.power >= att.toughness
			// And we either survive or it's a favorable trade (attacker bigger)
			survives := blk.toughness > att.power
			favorableTrade := canKill && att.power >= blk.power

			if canKill && (survives || favorableTrade) {
				pairs = append(pairs, blockerPair{Blocker: blk.name, Attacker: att.name})
				used[blk.name] = true
				break
			}
		}
	}

	return actionMsg{Type: "action", Kind: "block", Blockers: pairs}
}

// chooseChoice responds to a mid-resolution choice decision point.
// Picks the first option for mode/permanent/card choices, accepts may abilities,
// and picks the first color for mana color choices.
func chooseChoice(msg *oracleMsg) actionMsg {
	if msg.Choice == nil {
		return actionMsg{Type: "action", Kind: "choice", SelectedIndex: 0, Accepted: true}
	}

	switch msg.Choice.Kind {
	case "may":
		return actionMsg{Type: "action", Kind: "choice", Accepted: true}

	case "mode", "permanent", "card", "card_from_library", "number":
		idx := 0
		name := ""
		if len(msg.Choice.Options) > 0 {
			idx = msg.Choice.Options[0].Index
			name = msg.Choice.Options[0].Label
		}
		return actionMsg{
			Type:          "action",
			Kind:          "choice",
			SelectedIndex: idx,
			SelectedName:  name,
		}

	case "mana_color":
		name := ""
		if len(msg.Choice.Options) > 0 {
			name = msg.Choice.Options[0].Label
		}
		return actionMsg{
			Type:         "action",
			Kind:         "choice",
			SelectedName: name,
		}
	}

	return actionMsg{Type: "action", Kind: "choice", SelectedIndex: 0, Accepted: true}
}
