package combatsolver

import (
	"slices"

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

// abilityPump is a mana-only activated ability that pumps a target creature
// (e.g. "{G}: target creature gets +2/+2"). Unlike a hand spell it is a
// repeatable resource — it can fire as many times as mana allows — so it is
// bounded only by the shared mana budget, never consumed.
type abilityPump struct {
	power, toughness, cost int
	source                 *mage.Permanent
	target                 mage.Target
}

// pumpUnit is one repeatable +power/+toughness-for-cost increment applicable to
// a specific creature (its own self-pump ability, or a targeted pump ability
// that can legally target it).
type pumpUnit struct {
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
	abilityPumps := gatherAbilityPumps(g, playerID)

	for _, pr := range pairings {
		if pr.enemyID == uuid.Nil {
			continue
		}
		ours := g.FindPermanent(pr.oursID)
		enemy := g.FindPermanent(pr.enemyID)
		if ours == nil || enemy == nil || ours.ControllerID() != playerID {
			continue
		}
		budget = applyPumpToFlip(g, playerID, ours, enemy, &handPumps, abilityPumps, budget)
	}
}

// applyPumpToFlip pumps `ours` just enough to both kill `enemy` and survive the
// exchange. It draws on the repeatable pump engines applicable to `ours` (its
// own self-pump ability and any targeted pump ability that can legally target
// it) plus the shared one-shot hand-pump pool. It applies the cheapest feasible
// plan and returns the remaining mana budget; if no plan within budget achieves
// the flip (or none is needed), it leaves the game unchanged.
func applyPumpToFlip(g *mage.Game, playerID uuid.UUID, ours, enemy *mage.Permanent, handPumps *[]handPump, abilityPumps []abilityPump, budget int) int {
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

	units := applicablePumpUnits(g, playerID, ours, abilityPumps)

	bestCost := -1
	bestSpell := -1
	bestAddP, bestAddT := 0, 0

	// Try each repeatable engine on its own (combined with at most one one-shot
	// hand spell). A single engine plus a spell covers the common cases; mixing
	// two different repeatable engines on one creature is left unmodeled.
	unitChoices := []int{-1}
	for i := range units {
		unitChoices = append(unitChoices, i)
	}
	spellChoices := []int{-1}
	for i := range *handPumps {
		spellChoices = append(spellChoices, i)
	}

	for _, unitIdx := range unitChoices {
		var unit pumpUnit
		maxUses := 0
		if unitIdx >= 0 {
			unit = units[unitIdx]
			if unit.cost > 0 {
				maxUses = budget / unit.cost
			}
		}
		for _, spellIdx := range spellChoices {
			spP, spT, spCost := 0, 0, 0
			if spellIdx >= 0 {
				sp := (*handPumps)[spellIdx]
				spP, spT, spCost = sp.power, sp.toughness, sp.cost
			}
			for uses := 0; uses <= maxUses; uses++ {
				cost := uses*unit.cost + spCost
				if cost > budget {
					break
				}
				addP := uses*unit.power + spP
				addT := uses*unit.toughness + spT
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

// applicablePumpUnits returns the repeatable pump increments that can be applied
// to `ours`: its own self-pump ability (if any) and every targeted pump ability
// the player controls that can legally target it.
func applicablePumpUnits(g *mage.Game, playerID uuid.UUID, ours *mage.Permanent, abilityPumps []abilityPump) []pumpUnit {
	var units []pumpUnit
	if self, ok := selfPumpAbility(ours); ok {
		units = append(units, pumpUnit(self))
	}
	for _, ap := range abilityPumps {
		if slices.Contains(ap.target.Possible(playerID, ap.source.Card, g), ours.ID()) {
			units = append(units, pumpUnit{power: ap.power, toughness: ap.toughness, cost: ap.cost})
		}
	}
	return units
}

// gatherAbilityPumps collects the player's mana-only activated abilities that
// pump a single target creature (e.g. "{G}: target creature gets +2/+2").
// Abilities with non-mana cost components (tap, sacrifice) are excluded — they
// are one-shot and would need tap/timing modeling, left to a later pass.
func gatherAbilityPumps(g *mage.Game, playerID uuid.UUID) []abilityPump {
	var pumps []abilityPump
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != playerID {
			continue
		}
		for _, raw := range perm.RuntimeAbilities {
			ab, ok := mage.UnwrapAbility(raw).(mage.ActivatedAbility)
			if !ok {
				continue
			}
			targets := ab.Targets()
			if len(targets) != 1 {
				continue
			}
			if _, ok := targets[0].(*mage.CreatureTarget); !ok {
				continue
			}
			cost := manaOnlyCost(ab)
			if cost <= 0 {
				continue
			}
			power, toughness := pumpBoost(ab.Effects())
			if power <= 0 && toughness <= 0 {
				continue
			}
			pumps = append(pumps, abilityPump{
				power: power, toughness: toughness, cost: cost,
				source: perm, target: targets[0],
			})
		}
	}
	return pumps
}

// manaOnlyCost returns the total mana value of an ability's cost, or 0 if the
// ability has any non-mana cost component (tap, sacrifice, discard, etc.).
func manaOnlyCost(ab mage.ActivatedAbility) int {
	total := 0
	for _, c := range ab.Costs() {
		mc, ok := c.(*mage.ManaCostPayment)
		if !ok {
			return 0
		}
		total += mc.MC.CMC()
	}
	return total
}

// pumpBoost sums the beneficial +power/+toughness an effect list grants.
func pumpBoost(effects []mage.Effect) (power, toughness int) {
	for _, e := range effects {
		props := e.Properties()
		if props.Outcome != mage.OutcomeBenefit {
			continue
		}
		power += max(props.PowerBoost, 0)
		toughness += max(props.ToughnessBoost, 0)
	}
	return power, toughness
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
		if atk == nil || atk.ControllerID() != playerID {
			continue
		}
		seen[b.AttackerID] = true
		pairings = append(pairings, trickPairing{oursID: b.AttackerID, enemyID: b.BlockerID})
	}
	return pairings
}
