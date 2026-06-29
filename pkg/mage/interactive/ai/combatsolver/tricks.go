package combatsolver

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// trickPairing names one of the player's combat creatures and the enemy
// creature it is fighting (its blocker, or the attacker it blocks). A Nil
// enemyID marks an unblocked attacker, for which face-damage pump is not yet
// modeled.
type trickPairing struct {
	oursID  uuid.UUID
	enemyID uuid.UUID
}

// handPump is a pump spell held in hand that the solver can model casting on one
// of our combat creatures during the post-blockers response window (e.g. Giant
// Growth). It is a one-shot resource: once modeled on one pairing it is removed
// from the available pool so it can't be double-counted.
type handPump struct {
	power, toughness, cost int
}

// applyCombatTricks models the AI firing combat pump during the post-blockers
// response window. For each pairing where our creature is fighting an enemy, it
// applies the cheapest combination of self-pump abilities (mana-only,
// untargeted) and held pump spells that flips the matchup to "kill the enemy and
// survive", within the player's available mana. Pump is applied only when it
// achieves that flip, so the solver never models wasting a trick or mana.
//
// includeHandSpells is false when modeling the opponent's pump: per the package
// contract the solver does not read the opponent's hand, only their on-board
// activated abilities.
func applyCombatTricks(g *mage.Game, playerID uuid.UUID, pairings []trickPairing, includeHandSpells bool) {
	budget := g.HypotheticalMana(playerID)
	if budget <= 0 || len(pairings) == 0 {
		return
	}

	var handPumps []handPump
	if includeHandSpells {
		handPumps = gatherHandPumps(g, playerID)
	}

	for _, pr := range pairings {
		if pr.enemyID == uuid.Nil {
			continue
		}
		ours := g.FindPermanent(pr.oursID)
		enemy := g.FindPermanent(pr.enemyID)
		if ours == nil || enemy == nil || ours.Controller != playerID {
			continue
		}
		budget = applyPumpToFlip(g, ours, enemy, &handPumps, budget)
	}
}

// applyPumpToFlip pumps `ours` just enough to both kill `enemy` and survive the
// exchange, drawing on its self-pump ability and the shared one-shot hand-pump
// pool. It applies the cheapest feasible plan and returns the remaining mana
// budget; if no plan within budget achieves the flip (or none is needed), it
// leaves the game unchanged and returns the budget as-is.
func applyPumpToFlip(g *mage.Game, ours, enemy *mage.Permanent, handPumps *[]handPump, budget int) int {
	oursPow := ours.CurrentPower(g)
	oursTough := ours.CurrentToughness(g)
	enemyPow := enemy.CurrentPower(g)
	enemyTough := enemy.CurrentToughness(g) - enemy.Damage

	killThreshold := enemyTough
	if ours.HasKeyword(core.Deathtouch) {
		killThreshold = 1
	}
	enemyDeathtouch := enemy.HasKeyword(core.Deathtouch)

	kills := oursPow >= killThreshold
	survives := oursTough > enemyPow && (!enemyDeathtouch || enemyPow <= 0)
	if kills && survives {
		return budget
	}
	// A deathtouch enemy with any power kills our creature regardless of how much
	// toughness we add, so no pump can produce a kill-and-survive flip.
	if enemyDeathtouch && enemyPow > 0 {
		return budget
	}

	needP := max(killThreshold-oursPow, 0)
	needT := max(enemyPow-oursTough+1, 0)

	self, hasSelf := selfPumpAbility(ours)
	maxUses := 0
	if hasSelf && self.cost > 0 {
		maxUses = budget / self.cost
	}

	bestCost := -1
	bestSpell := -1
	bestAddP, bestAddT := 0, 0

	spellChoices := []int{-1}
	for i := range *handPumps {
		spellChoices = append(spellChoices, i)
	}
	for _, spellIdx := range spellChoices {
		spP, spT, spCost := 0, 0, 0
		if spellIdx >= 0 {
			sp := (*handPumps)[spellIdx]
			spP, spT, spCost = sp.power, sp.toughness, sp.cost
		}
		for uses := 0; uses <= maxUses; uses++ {
			cost := uses*self.cost + spCost
			if cost > budget {
				break
			}
			addP := uses*self.power + spP
			addT := uses*self.toughness + spT
			if addP >= needP && addT >= needT {
				if bestCost < 0 || cost < bestCost {
					bestCost = cost
					bestSpell = spellIdx
					bestAddP, bestAddT = addP, addT
				}
				break
			}
		}
	}

	if bestCost < 0 {
		return budget
	}

	eff := mage.TemporaryBoost(ours.ID(), bestAddP, bestAddT)
	eff.SetSourceID(ours.ID())
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()

	if bestSpell >= 0 {
		hp := *handPumps
		*handPumps = append(hp[:bestSpell], hp[bestSpell+1:]...)
	}
	return budget - bestCost
}

// gatherHandPumps collects affordable "+X/+Y" combat tricks from the player's
// hand as one-shot pump options. Only instants the solver classifies as
// RolePump and that target a creature are included.
func gatherHandPumps(g *mage.Game, playerID uuid.UUID) []handPump {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	var pumps []handPump
	for _, card := range p.Hand() {
		if ClassifyCombat(card) != RolePump {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost(), mage.SpellContextForCard(card)) {
			continue
		}
		power, toughness, targetsCreature := pumpSpellBoost(card)
		if !targetsCreature || (power <= 0 && toughness <= 0) {
			continue
		}
		pumps = append(pumps, handPump{power: power, toughness: toughness, cost: card.ManaCost().CMC()})
	}
	return pumps
}

// pumpSpellBoost extracts the total +power/+toughness a pump spell grants and
// whether it targets a creature (so it can be aimed at one of our combatants).
func pumpSpellBoost(card mage.Card) (power, toughness int, targetsCreature bool) {
	for _, ab := range card.Abilities() {
		def, ok := ab.(*mage.ActionDefinition)
		if !ok || def.Kind() != mage.ActionSpell {
			continue
		}
		for _, t := range def.Targets() {
			if _, ok := t.(*mage.CreatureTarget); ok {
				targetsCreature = true
			}
		}
		for _, e := range def.Effects() {
			props := e.Properties()
			if props.PowerBoost > 0 {
				power += props.PowerBoost
			}
			if props.ToughnessBoost > 0 {
				toughness += props.ToughnessBoost
			}
		}
	}
	return power, toughness, targetsCreature
}

// defenderPairings maps each of the player's blocks to the (blocker, attacker)
// combat it represents.
func defenderPairings(blocks []mage.BlockAssignment) []trickPairing {
	pairings := make([]trickPairing, 0, len(blocks))
	for _, b := range blocks {
		pairings = append(pairings, trickPairing{oursID: b.BlockerID, enemyID: b.AttackerID})
	}
	return pairings
}

// attackerPairings maps each of the player's attackers that is blocked by
// exactly one creature to the (attacker, blocker) combat it represents.
// Multi-blocked attackers are skipped: modeling pump against a gang block needs
// the full damage-assignment picture, which is left to a later pass.
func attackerPairings(g *mage.Game, playerID uuid.UUID, blocks []mage.BlockAssignment) []trickPairing {
	blockerCount := map[uuid.UUID]int{}
	for _, b := range blocks {
		blockerCount[b.AttackerID]++
	}
	var pairings []trickPairing
	seen := map[uuid.UUID]bool{}
	for _, b := range blocks {
		if blockerCount[b.AttackerID] != 1 || seen[b.AttackerID] {
			continue
		}
		atk := g.FindPermanent(b.AttackerID)
		if atk == nil || atk.Controller != playerID {
			continue
		}
		seen[b.AttackerID] = true
		pairings = append(pairings, trickPairing{oursID: b.AttackerID, enemyID: b.BlockerID})
	}
	return pairings
}
