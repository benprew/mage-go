package search

import (
	"sort"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// incomingDamagePenalty estimates the pressure that defender will face on
// attacker's next turn. The search horizon ends at the active player's
// Cleanup, before opponent's untap and swing — without this, the eval treats
// "all my creatures tapped" the same as "all my creatures untapped".
//
// Model: assume attacker untaps and swings with every creature that can
// attack. For each attacker (largest first), pair it with the largest unused
// defender creature that can legally block it per mage.CanBlock — covering
// flying/reach, fear, protection, walls, etc. Landwalk evasion is handled
// separately via mage.HasLandwalkEvasion. A blocked attacker deals 0 (no
// trample in our card pool); unblocked power runs through lifePressure.
//
// The greedy is suboptimal vs. true bipartite max-matching, but cheap and
// good enough for the eval's resolution.
func incomingDamagePenalty(g *mage.Game, defender, attacker mage.Player) float64 {
	attackers := gatherAttackers(g, attacker.PlayerID())
	blockers := gatherBlockers(g, defender.PlayerID())
	used := make([]bool, len(blockers))

	totalDamage := 0
	for _, a := range attackers {
		if mage.HasLandwalkEvasion(a, defender.PlayerID(), g) {
			totalDamage += a.CurrentPower(g)
			continue
		}
		matched := false
		for i, b := range blockers {
			if used[i] {
				continue
			}
			if mage.CanBlock(b, a, g) {
				used[i] = true
				matched = true
				break
			}
		}
		if !matched {
			totalDamage += a.CurrentPower(g)
		}
	}

	if totalDamage <= 0 {
		return 0
	}

	postLife := defender.Life() - totalDamage
	delta := lifePressure(postLife) - lifePressure(defender.Life())
	return wPressure * delta
}

// gatherAttackers returns attackerID's creatures that can legally attack next
// turn, sorted by power descending so the greedy pairs biggest threats first.
func gatherAttackers(g *mage.Game, attackerID uuid.UUID) []*mage.Permanent {
	var attackers []*mage.Permanent
	for _, p := range g.AllBattlefield() {
		if p.Controller != attackerID || !p.HasType(core.TypeCreature) {
			continue
		}
		if !canAttackNextTurn(p) {
			continue
		}
		attackers = append(attackers, p)
	}
	sortByPowerDesc(g, attackers)
	return attackers
}

// gatherBlockers returns defenderID's creatures, sorted by power descending.
// We don't filter by Tapped: defender's own untap happens before they're
// attacked, so non-DoesNotUntap creatures will be available.
func gatherBlockers(g *mage.Game, defenderID uuid.UUID) []*mage.Permanent {
	var blockers []*mage.Permanent
	for _, p := range g.AllBattlefield() {
		if p.Controller != defenderID || !p.HasType(core.TypeCreature) {
			continue
		}
		blockers = append(blockers, p)
	}
	sortByPowerDesc(g, blockers)
	return blockers
}

// canAttackNextTurn approximates whether perm will be a legal attacker on its
// controller's next turn. Summoning sickness clears at untap, so a sick
// creature this turn can attack next turn (unless it lacks AttrCanAttack
// permanently, e.g. Defender). We're conservative on "can it attack at all".
func canAttackNextTurn(p *mage.Permanent) bool {
	if p.HasAttr(core.Defender) {
		return false
	}
	if !p.HasAttr(core.AttrCanAttack) {
		return false
	}
	return true
}

func sortByPowerDesc(g *mage.Game, perms []*mage.Permanent) {
	sort.Slice(perms, func(i, j int) bool {
		return perms[i].CurrentPower(g) > perms[j].CurrentPower(g)
	})
}
