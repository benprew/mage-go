package heuristic

import (
	"maps"
	"slices"
	"sort"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/combatsolver"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// combatDamageResult holds the per-creature damage and life totals produced by
// simulating a combat's damage step(s).
type combatDamageResult struct {
	dmgTaken           map[uuid.UUID]int
	deathtouchMarked   map[uuid.UUID]bool
	damageToOpponent   int
	lifeGained         int // life gained by attacking lifelink creatures
	opponentLifeGained int // life gained by blocking lifelink creatures
}

// isDead returns true if a creature has been killed by accumulated damage
// (including deathtouch marking).
func (r combatDamageResult) isDead(id uuid.UUID, toughness int) bool {
	if r.deathtouchMarked[id] && r.dmgTaken[id] > 0 {
		return true
	}
	return r.dmgTaken[id] >= toughness
}

// simulateCombatDamage resolves the combat damage step(s) for the given
// attackers and block assignments without mutating game state, returning the
// damage marked on each creature and the life totals exchanged. initialDamage
// seeds damage already marked on creatures (e.g. from a burn spell earlier in
// the turn); pass nil for a fresh prediction.
func simulateCombatDamage(g *mage.Game, attackers []uuid.UUID, blockerMap map[uuid.UUID][]uuid.UUID, initialDamage map[uuid.UUID]int) combatDamageResult {
	res := combatDamageResult{
		dmgTaken:         make(map[uuid.UUID]int),
		deathtouchMarked: make(map[uuid.UUID]bool),
	}
	maps.Copy(res.dmgTaken, initialDamage)

	// Check if any creature in this combat has first strike or double strike.
	// If so, we need to split into two damage sub-steps.
	hasFirstStrikeStep := false
	for _, atkID := range attackers {
		atk := g.FindPermanent(atkID)
		if atk == nil {
			continue
		}
		if atk.HasKeyword(core.FirstStrike) || atk.HasKeyword(core.DoubleStrike) {
			hasFirstStrikeStep = true
			break
		}
		for _, blkID := range blockerMap[atkID] {
			blk := g.FindPermanent(blkID)
			if blk != nil && (blk.HasKeyword(core.FirstStrike) || blk.HasKeyword(core.DoubleStrike)) {
				hasFirstStrikeStep = true
				break
			}
		}
		if hasFirstStrikeStep {
			break
		}
	}

	dmgTaken := res.dmgTaken
	deathtouchMarked := res.deathtouchMarked
	isDead := res.isDead

	// resolveStep applies one damage sub-step. Damage within a step is simultaneous:
	// creatures that die this step still deal their damage. Only creatures killed
	// in a *previous* step are excluded.
	resolveStep := func(isFirstStrikeStep bool) {
		// Collect new damage in a separate map, then merge after the step.
		stepDmg := make(map[uuid.UUID]int)
		stepDT := make(map[uuid.UUID]bool)

		for _, atkID := range attackers {
			atk := g.FindPermanent(atkID)
			if atk == nil {
				continue
			}
			atkPow := atk.CurrentPower(g)
			atkTough := atk.CurrentToughness(g)
			atkHasFS := atk.HasKeyword(core.FirstStrike)
			atkHasDS := atk.HasKeyword(core.DoubleStrike)
			atkHasDT := atk.HasKeyword(core.Deathtouch)
			atkHasLL := atk.HasKeyword(core.Lifelink)

			blockerIDs, isBlocked := blockerMap[atkID]
			if !isBlocked || len(blockerIDs) == 0 {
				// Unblocked attacker.
				var atkDealsThisStep bool
				if isFirstStrikeStep {
					atkDealsThisStep = atkHasFS || atkHasDS
				} else {
					atkDealsThisStep = !atkHasFS || atkHasDS
				}
				// Dead attackers (from a previous step) don't deal damage.
				if isDead(atkID, atkTough) {
					continue
				}
				if atkDealsThisStep && atkPow > 0 {
					res.damageToOpponent += atkPow
					if atkHasLL {
						res.lifeGained += atkPow
					}
				}
				continue
			}

			// Determine if attacker deals damage this step.
			var atkDealsThisStep bool
			if isFirstStrikeStep {
				atkDealsThisStep = atkHasFS || atkHasDS
			} else {
				// Normal step: creatures without first strike deal damage,
				// plus double strikers deal damage again.
				atkDealsThisStep = !atkHasFS || atkHasDS
			}

			// Dead attackers (from a previous step) don't deal damage.
			if isDead(atkID, atkTough) {
				atkDealsThisStep = false
			}

			// Attacker assigns damage to blockers.
			if atkDealsThisStep {
				remainingAtkDmg := atkPow
				atkDmgDealt := atkPow // attacker always deals its full power
				for _, blkID := range blockerIDs {
					blk := g.FindPermanent(blkID)
					if blk == nil {
						continue
					}
					blkTough := blk.CurrentToughness(g)

					// Skip blockers already dead from a previous step.
					if isDead(blkID, blkTough) {
						continue
					}

					neededToKill := blkTough - dmgTaken[blkID]
					if atkHasDT && neededToKill > 0 {
						neededToKill = 1
					}
					if neededToKill <= 0 {
						continue
					}

					assigned := min(neededToKill, remainingAtkDmg)
					stepDmg[blkID] += assigned
					if atkHasDT && assigned > 0 {
						stepDT[blkID] = true
					}
					remainingAtkDmg -= assigned
					if remainingAtkDmg <= 0 {
						break
					}
				}

				// Trample: excess damage goes to opponent.
				if remainingAtkDmg > 0 && atk.HasKeyword(core.Trample) {
					res.damageToOpponent += remainingAtkDmg
				}

				// Lifelink: attacker gains life for all damage dealt.
				if atkHasLL && atkDmgDealt > 0 {
					res.lifeGained += atkDmgDealt
				}
			}

			// Blockers deal damage to attacker this step.
			for _, blkID := range blockerIDs {
				blk := g.FindPermanent(blkID)
				if blk == nil {
					continue
				}
				blkPow := blk.CurrentPower(g)
				blkTough := blk.CurrentToughness(g)
				blkHasFS := blk.HasKeyword(core.FirstStrike)
				blkHasDS := blk.HasKeyword(core.DoubleStrike)
				blkHasDT := blk.HasKeyword(core.Deathtouch)
				blkHasLL := blk.HasKeyword(core.Lifelink)

				// Skip blockers dead from a previous step (not this step — simultaneous).
				if isDead(blkID, blkTough) {
					continue
				}

				var blkDealsThisStep bool
				if isFirstStrikeStep {
					blkDealsThisStep = blkHasFS || blkHasDS
				} else {
					blkDealsThisStep = !blkHasFS || blkHasDS
				}

				if blkDealsThisStep && blkPow > 0 {
					stepDmg[atkID] += blkPow
					if blkHasDT {
						stepDT[atkID] = true
					}
					if blkHasLL {
						res.opponentLifeGained += blkPow
					}
				}
			}
		}

		// Merge step damage into accumulated totals.
		for id, dmg := range stepDmg {
			dmgTaken[id] += dmg
		}
		for id := range stepDT {
			deathtouchMarked[id] = true
		}
	}

	if hasFirstStrikeStep {
		resolveStep(true)  // first strike sub-step
		resolveStep(false) // normal damage sub-step
	} else {
		resolveStep(false) // single simultaneous step
	}

	return res
}

// considerRegeneration looks for one of our creatures that the upcoming combat
// damage will destroy and, if we control an affordable regeneration ability that
// can save it, returns an action to activate that ability. Regeneration shields
// must be installed before combat damage is dealt — state-based actions destroy
// a creature with lethal damage before any player receives priority — so this is
// evaluated during the declare-blockers step, once blocks are known but before
// damage. Damage already marked on a creature (e.g. from a burn spell earlier in
// the turn) is folded into the lethality prediction.
func (s *Strategy) considerRegeneration(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	combat := g.GetCombat()
	if combat == nil {
		return nil
	}

	// Act only with an empty stack: blocks and any combat tricks have settled, so
	// the prediction is final, and we won't stack a second regeneration on a
	// creature whose shield ability is still resolving.
	if len(g.StackObjects()) > 0 {
		return nil
	}

	attackers := make([]uuid.UUID, 0, len(combat.Groups))
	blockerMap := make(map[uuid.UUID][]uuid.UUID, len(combat.Groups))
	initialDamage := make(map[uuid.UUID]int, len(combat.Groups)*2)
	for _, grp := range combat.Groups {
		atk := g.FindPermanent(grp.AttackerID)
		if atk == nil {
			continue
		}
		attackers = append(attackers, grp.AttackerID)
		blockerMap[grp.AttackerID] = grp.BlockerIDs
		initialDamage[grp.AttackerID] = atk.Damage
		for _, bid := range grp.BlockerIDs {
			if blk := g.FindPermanent(bid); blk != nil {
				initialDamage[bid] = blk.Damage
			}
		}
	}
	if len(attackers) == 0 {
		return nil
	}

	res := simulateCombatDamage(g, attackers, blockerMap, initialDamage)

	// Collect our creatures combat will destroy, skipping any that already carry
	// a regeneration shield so we never spend a redundant activation.
	var doomed []*mage.Permanent
	seen := make(map[uuid.UUID]bool)
	consider := func(id uuid.UUID) {
		if seen[id] {
			return
		}
		seen[id] = true
		perm := g.FindPermanent(id)
		if perm == nil || perm.Controller != playerID || !perm.HasType(core.TypeCreature) {
			return
		}
		if g.HasRegenerationShield(id) {
			return
		}
		if res.isDead(id, perm.CurrentToughness(g)) {
			doomed = append(doomed, perm)
		}
	}
	for _, atkID := range attackers {
		consider(atkID)
		for _, bid := range blockerMap[atkID] {
			consider(bid)
		}
	}
	if len(doomed) == 0 {
		return nil
	}
	// Save the most valuable creature first.
	sort.SliceStable(doomed, func(i, j int) bool {
		return eval.EvalCreatureInGame(doomed[i], g) > eval.EvalCreatureInGame(doomed[j], g)
	})

	abilities := g.GetActivatableAbilities(playerID)
	for _, target := range doomed {
		if action := s.regenerationAbilityAction(playerID, g, abilities, target); action != nil {
			return action
		}
		if action := s.regenerationSpellAction(p, g, target); action != nil {
			return action
		}
		if action := s.pumpToughnessAction(playerID, g, abilities, target, res, initialDamage[target.ID()]); action != nil {
			return action
		}
	}

	return nil
}

// pumpToughnessAction looks for an affordable toughness-boost activation that
// pushes target's toughness out of range of the combat damage res predicts for
// it, mirroring the regeneration fallback in considerRegeneration for
// creatures with no regeneration ability. initialDamage is the damage already
// marked on target when res was simulated, so it can be subtracted back out to
// isolate the damage this combat step alone will deal — pumping toughness only
// helps against that, never against damage already marked. A creature marked
// by deathtouch this step dies regardless of toughness, so pumping can't help.
func (s *Strategy) pumpToughnessAction(playerID uuid.UUID, g *mage.Game, abilities []mage.ActivatableInfo, target *mage.Permanent, res combatDamageResult, initialDamage int) *interactive.PriorityAction {
	if res.deathtouchMarked[target.ID()] {
		return nil
	}
	dmg := res.dmgTaken[target.ID()] - initialDamage
	if dmg <= 0 {
		return nil
	}
	effToughness := target.CurrentToughness(g) - target.Damage
	return s.pumpAbilityAction(playerID, g, abilities, target, dmg, effToughness)
}

// considerPumpForCombatKill looks for one of our attackers or blockers that
// doesn't currently deal enough damage to kill the creature it's fighting and,
// if we control an affordable power-boost ability, pumps it enough to secure
// the kill. Only single-attacker/single-blocker pairings are considered — gang
// blocks split an attacker's damage across multiple blockers, so pumping one
// combatant's power doesn't reliably determine what the opposing creature
// takes. Like considerRegeneration, this must act before combat damage is
// dealt, so it only runs during the declare-blockers step with an empty stack.
func (s *Strategy) considerPumpForCombatKill(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	combat := g.GetCombat()
	if combat == nil {
		return nil
	}
	if len(g.StackObjects()) > 0 {
		return nil
	}

	attackers := make([]uuid.UUID, 0, len(combat.Groups))
	blockerMap := make(map[uuid.UUID][]uuid.UUID, len(combat.Groups))
	initialDamage := make(map[uuid.UUID]int, len(combat.Groups)*2)
	for _, grp := range combat.Groups {
		atk := g.FindPermanent(grp.AttackerID)
		if atk == nil {
			continue
		}
		attackers = append(attackers, grp.AttackerID)
		blockerMap[grp.AttackerID] = grp.BlockerIDs
		initialDamage[grp.AttackerID] = atk.Damage
		for _, bid := range grp.BlockerIDs {
			if blk := g.FindPermanent(bid); blk != nil {
				initialDamage[bid] = blk.Damage
			}
		}
	}
	if len(attackers) == 0 {
		return nil
	}

	res := simulateCombatDamage(g, attackers, blockerMap, initialDamage)

	// Pairs of (our creature, the single creature it's fighting) where our
	// creature is not already killing its opponent.
	type pair struct{ mine, theirs *mage.Permanent }
	var candidates []pair
	for _, atkID := range attackers {
		blockerIDs := blockerMap[atkID]
		if len(blockerIDs) != 1 {
			continue
		}
		atk := g.FindPermanent(atkID)
		blk := g.FindPermanent(blockerIDs[0])
		if atk == nil || blk == nil {
			continue
		}
		if atk.Controller == playerID && atk.HasType(core.TypeCreature) && blk.HasType(core.TypeCreature) &&
			!res.isDead(blk.ID(), blk.CurrentToughness(g)) {
			candidates = append(candidates, pair{mine: atk, theirs: blk})
		}
		if blk.Controller == playerID && blk.HasType(core.TypeCreature) && atk.HasType(core.TypeCreature) &&
			!res.isDead(atk.ID(), atk.CurrentToughness(g)) {
			candidates = append(candidates, pair{mine: blk, theirs: atk})
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return eval.EvalCreatureInGame(candidates[i].theirs, g) > eval.EvalCreatureInGame(candidates[j].theirs, g)
	})

	abilities := g.GetActivatableAbilities(playerID)
	for _, c := range candidates {
		deficit := c.theirs.CurrentToughness(g) - res.dmgTaken[c.theirs.ID()]
		if deficit <= 0 {
			continue
		}
		if action := s.pumpPowerAction(playerID, g, abilities, c.mine, deficit); action != nil {
			return action
		}
	}
	return nil
}

// pumpPowerAction returns the first affordable power-boost activation on
// source that raises its power by at least deficit, or nil if none can fully
// cover it from currently available mana.
func (s *Strategy) pumpPowerAction(playerID uuid.UUID, g *mage.Game, abilities []mage.ActivatableInfo, source *mage.Permanent, deficit int) *interactive.PriorityAction {
	for i := range abilities {
		info := &abilities[i]
		perm := g.FindPermanent(info.PermanentID)
		if perm == nil || perm.ID() != source.ID() {
			continue
		}
		if info.AbilityIndex < 0 || info.AbilityIndex >= len(perm.RuntimeAbilities) {
			continue
		}
		ab, ok := mage.UnwrapAbility(perm.RuntimeAbilities[info.AbilityIndex]).(mage.ActivatedAbility)
		if !ok || len(ab.Targets()) != 0 {
			continue
		}
		boost := abilityPowerBoost(ab)
		if boost <= 0 {
			continue
		}
		need := (deficit + boost - 1) / boost
		if affordablePumpCount(ab, g, playerID, need) < need {
			continue
		}
		return &interactive.PriorityAction{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
		}
	}
	return nil
}

// abilityPowerBoost sums the power granted by a single activation of ab.
func abilityPowerBoost(ab mage.ActivatedAbility) int {
	boost := 0
	for _, e := range ab.Effects() {
		boost += e.Properties().PowerBoost
	}
	return boost
}

// considerRegenerationAgainstRemoval looks for an opponent's spell or ability on
// the stack that would destroy one of our creatures and, if we can, installs a
// regeneration shield in response so the creature survives. Regeneration
// replaces destruction (CR 701.15), so this answers "destroy target creature"
// removal and lethal burn — but not exile, bounce, -X/-X, sacrifice, or
// "can't be regenerated" destroys, which set no Destroys property and are
// therefore skipped.
func (s *Strategy) considerRegenerationAgainstRemoval(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()

	var doomed []*mage.Permanent
	seen := make(map[uuid.UUID]bool)
	addDoomed := func(perm *mage.Permanent) {
		if perm == nil || seen[perm.ID()] {
			return
		}
		seen[perm.ID()] = true
		if perm.Controller != playerID || !perm.HasType(core.TypeCreature) {
			return
		}
		if g.HasRegenerationShield(perm.ID()) {
			return
		}
		doomed = append(doomed, perm)
	}

	for _, obj := range g.StackObjects() {
		if obj.Controller == playerID {
			continue
		}
		for _, e := range obj.Effects {
			props := e.Properties()
			switch {
			case props.Destroys && props.Mass:
				for _, perm := range g.AllBattlefield() {
					if perm.Controller == playerID && perm.HasType(core.TypeCreature) {
						addDoomed(perm)
					}
				}
			case props.Destroys:
				for _, tid := range obj.Targets {
					addDoomed(g.FindPermanent(tid))
				}
			case props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment:
				dmg := props.DamageValue.Resolve(g, obj.SourceID, obj.Controller, obj.Targets)
				for _, tid := range obj.Targets {
					perm := g.FindPermanent(tid)
					if perm == nil || perm.Controller != playerID || !perm.HasType(core.TypeCreature) {
						continue
					}
					if dmg >= perm.CurrentToughness(g)-perm.Damage {
						addDoomed(perm)
					}
				}
			}
		}
	}
	if len(doomed) == 0 {
		return nil
	}

	// Save the most valuable creature first.
	sort.SliceStable(doomed, func(i, j int) bool {
		return eval.EvalCreatureInGame(doomed[i], g) > eval.EvalCreatureInGame(doomed[j], g)
	})

	abilities := g.GetActivatableAbilities(playerID)
	for _, target := range doomed {
		if action := s.regenerationAbilityAction(playerID, g, abilities, target); action != nil {
			return action
		}
		if action := s.regenerationSpellAction(p, g, target); action != nil {
			return action
		}
	}
	return nil
}

func (s *Strategy) regenerationAbilityAction(playerID uuid.UUID, g *mage.Game, abilities []mage.ActivatableInfo, target *mage.Permanent) *interactive.PriorityAction {
	for i := range abilities {
		info := &abilities[i]
		source := g.FindPermanent(info.PermanentID)
		if source == nil {
			continue
		}
		if info.AbilityIndex < 0 || info.AbilityIndex >= len(source.RuntimeAbilities) {
			continue
		}
		ab, ok := mage.UnwrapAbility(source.RuntimeAbilities[info.AbilityIndex]).(mage.ActivatedAbility)
		if !ok || !regeneratesCreature(ab) {
			continue
		}

		if len(ab.Targets()) == 0 {
			if info.PermanentID != target.ID() && !regeneratesAttachedTarget(ab, source, target) {
				continue
			}
			return &interactive.PriorityAction{
				Type:         interactive.ActionActivateAbility,
				PermanentID:  info.PermanentID,
				AbilityIndex: info.AbilityIndex,
			}
		}

		tgt := ab.Targets()[0]
		if !slices.Contains(tgt.Possible(playerID, source.Card, g), target.ID()) {
			continue
		}
		return &interactive.PriorityAction{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
			Targets:      []uuid.UUID{target.ID()},
		}
	}
	return nil
}

func (s *Strategy) regenerationSpellAction(p mage.Player, g *mage.Game, target *mage.Permanent) *interactive.PriorityAction {
	playerID := p.PlayerID()
	for _, card := range g.GetCastableSpells(playerID) {
		for _, ability := range card.Abilities() {
			sa, ok := ability.(*mage.SpellAbility)
			if !ok || sa.Kind() != mage.ActionSpell || !regeneratesEffects(sa.Effects()) {
				continue
			}
			targets := sa.Targets()
			if len(targets) == 0 {
				continue
			}
			if !slices.Contains(targets[0].Possible(playerID, card, g), target.ID()) {
				continue
			}
			return &interactive.PriorityAction{
				Type:     interactive.ActionCastSpell,
				CardID:   card.ID(),
				CardName: card.Name(),
				Targets:  []uuid.UUID{target.ID()},
				XValue:   bestXValue(g, playerID, card, []uuid.UUID{target.ID()}),
			}
		}
	}
	return nil
}

// considerPumpAgainstRemoval saves one of our creatures from lethal burn on the
// stack by activating a toughness-boost ability enough times to push it out of
// range. Unlike regeneration (a single shield replaces destruction), pumping
// pays mana per point of toughness, so the AI commits only when it can afford to
// fully clear the incoming damage — pumping part-way wastes mana without saving
// the creature. One activation is returned per call; the engine re-grants
// priority, so the AI stacks the remaining activations across later passes.
func (s *Strategy) considerPumpAgainstRemoval(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()

	// Map each of our creatures to the largest single burn it faces on the stack.
	burn := make(map[uuid.UUID]int)
	for _, obj := range g.StackObjects() {
		if obj.Controller == playerID {
			continue
		}
		for _, e := range obj.Effects {
			props := e.Properties()
			if props.DamageValue == nil || props.Outcome != mage.OutcomeDetriment {
				continue
			}
			dmg := props.DamageValue.Resolve(g, obj.SourceID, obj.Controller, obj.Targets)
			for _, tid := range obj.Targets {
				perm := g.FindPermanent(tid)
				if perm == nil || perm.Controller != playerID || !perm.HasType(core.TypeCreature) {
					continue
				}
				if dmg > burn[tid] {
					burn[tid] = dmg
				}
			}
		}
	}
	if len(burn) == 0 {
		return nil
	}

	var doomed []*mage.Permanent
	for id := range burn {
		if perm := g.FindPermanent(id); perm != nil {
			doomed = append(doomed, perm)
		}
	}
	// Save the most valuable threatened creature first.
	sort.SliceStable(doomed, func(i, j int) bool {
		return eval.EvalCreatureInGame(doomed[i], g) > eval.EvalCreatureInGame(doomed[j], g)
	})

	abilities := g.GetActivatableAbilities(playerID)
	for _, target := range doomed {
		dmg := burn[target.ID()]
		// Effective toughness left before the burn resolves; a creature survives
		// only once this exceeds the incoming damage.
		effToughness := target.CurrentToughness(g) - target.Damage
		if effToughness > dmg {
			continue
		}
		if action := s.pumpAbilityAction(playerID, g, abilities, target, dmg, effToughness); action != nil {
			return action
		}
	}
	return nil
}

// pumpAbilityAction returns the first affordable toughness-boost activation that
// can lift target out of lethal-burn range, or nil if no such ability can fully
// save it from its current mana.
func (s *Strategy) pumpAbilityAction(playerID uuid.UUID, g *mage.Game, abilities []mage.ActivatableInfo, target *mage.Permanent, dmg, effToughness int) *interactive.PriorityAction {
	for i := range abilities {
		info := &abilities[i]
		source := g.FindPermanent(info.PermanentID)
		if source == nil {
			continue
		}
		if info.AbilityIndex < 0 || info.AbilityIndex >= len(source.RuntimeAbilities) {
			continue
		}
		ab, ok := mage.UnwrapAbility(source.RuntimeAbilities[info.AbilityIndex]).(mage.ActivatedAbility)
		if !ok {
			continue
		}
		targets, ok := pumpAbilityTargets(ab, source, target, playerID, g)
		if !ok {
			continue
		}
		boost := abilityToughnessBoost(ab)
		if boost <= 0 {
			continue
		}
		need := pumpsToSurvive(dmg, effToughness, boost)
		if need <= 0 {
			continue
		}
		if affordablePumpCount(ab, g, playerID, need) < need {
			continue
		}
		return &interactive.PriorityAction{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
			Targets:      targets,
		}
	}
	return nil
}

// pumpAbilityTargets reports whether ab can boost target and the targets to pass
// when activating it. Source-boosting abilities (e.g. "{B}: this creature gets
// +1/+1") take no targets and only help their own permanent; targeted boosts
// must be able to legally choose the doomed creature.
func pumpAbilityTargets(ab mage.ActivatedAbility, source, target *mage.Permanent, playerID uuid.UUID, g *mage.Game) ([]uuid.UUID, bool) {
	if len(ab.Targets()) == 0 {
		return nil, source.ID() == target.ID()
	}
	tgt := ab.Targets()[0]
	if !slices.Contains(tgt.Possible(playerID, source.Card, g), target.ID()) {
		return nil, false
	}
	return []uuid.UUID{target.ID()}, true
}

// abilityToughnessBoost sums the toughness granted by a single activation of ab.
func abilityToughnessBoost(ab mage.ActivatedAbility) int {
	boost := 0
	for _, e := range ab.Effects() {
		boost += e.Properties().ToughnessBoost
	}
	return boost
}

// pumpsToSurvive returns how many activations granting boost toughness each are
// needed for a creature with effToughness toughness left to survive dmg damage,
// or 0 if it is already safe.
func pumpsToSurvive(dmg, effToughness, boost int) int {
	if boost <= 0 {
		return 0
	}
	deficit := dmg - effToughness + 1
	if deficit <= 0 {
		return 0
	}
	return (deficit + boost - 1) / boost
}

// affordablePumpCount returns how many times (capped at max) the player can pay
// for ab from currently available mana. Abilities carrying a non-mana cost (a
// tap or sacrifice) or an {X} cost can be used at most once, since those costs
// cannot be repaid repeatedly this priority pass.
func affordablePumpCount(ab mage.ActivatedAbility, g *mage.Game, playerID uuid.UUID, max int) int {
	per := core.ManaCost{}
	repeatable := true
	for _, c := range ab.Costs() {
		mp, ok := c.(*mage.ManaCostPayment)
		if !ok || mp.MC.HasX {
			repeatable = false
			continue
		}
		per = addManaCost(per, mp.MC)
	}
	// The ability came from GetActivatableAbilities, so one activation is known
	// to be affordable already.
	if !repeatable {
		return 1
	}
	combined := core.ManaCost{}
	count := 0
	for count < max {
		combined = addManaCost(combined, per)
		if !g.CanAfford(playerID, combined, nil) {
			break
		}
		count++
	}
	return count
}

// addManaCost returns the sum of two mana costs (ignoring {X}, handled by callers).
func addManaCost(a, b core.ManaCost) core.ManaCost {
	a.Generic += b.Generic
	a.White += b.White
	a.Blue += b.Blue
	a.Black += b.Black
	a.Red += b.Red
	a.Green += b.Green
	a.Hybrid = append(append([]core.HybridSymbol{}, a.Hybrid...), b.Hybrid...)
	return a
}

// regeneratesCreature reports whether activating ab sets a regeneration shield.
func regeneratesCreature(ab mage.ActivatedAbility) bool {
	return regeneratesEffects(ab.Effects())
}

func regeneratesEffects(effects []mage.Effect) bool {
	return slices.ContainsFunc(effects, mage.IsRegenerationEffect)
}

func regeneratesAttachedTarget(ab mage.ActivatedAbility, source *mage.Permanent, target *mage.Permanent) bool {
	if source.AttachedTo != target.ID() {
		return false
	}
	return effectsRegenerateAttached(ab.Effects(), nil)
}

func effectsRegenerateAttached(effects []mage.Effect, attachedVars map[string]bool) bool {
	for _, effect := range effects {
		switch e := effect.(type) {
		case *mage.SnapshotAttachedData:
			if attachedVars == nil {
				attachedVars = make(map[string]bool)
			}
			attachedVars[e.StoreAs] = true
		case *mage.RegenerateGatheredData:
			if attachedVars[e.VarName] {
				return true
			}
		case *mage.PipelineData:
			if effectsRegenerateAttached(e.Steps, maps.Clone(attachedVars)) {
				return true
			}
		}
	}
	return false
}

// bestHoldableInstant returns the highest-value castable instant in hand worth
// considering for holding, along with its hold-adjusted value (removal and
// beneficial tricks are weighted up). Returns nil/0 when no usable instant is
// affordable.
func bestHoldableInstant(p mage.Player, g *mage.Game) (mage.Card, float64) {
	playerID := p.PlayerID()

	var bestCard mage.Card
	bestValue := 0.0
	for _, card := range p.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost(), mage.SpellContextForCard(card)) {
			continue
		}
		hasUsableEffect := false
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
				if mage.SpellOutcome(sa.Effects()) != mage.OutcomeUnknown {
					hasUsableEffect = true
					break
				}
			}
		}
		if !hasUsableEffect {
			continue
		}
		sv := float64(eval.SpellValue(card, p, g))
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
				switch mage.SpellOutcome(sa.Effects()) {
				case mage.OutcomeDetriment:
					sv *= 1.5
				case mage.OutcomeBenefit:
					sv *= 1.3
				}
			}
		}
		if sv > bestValue {
			bestValue = sv
			bestCard = card
		}
	}
	return bestCard, bestValue
}

func holdBackValue(p mage.Player, g *mage.Game, w ai.WeightedPersonality) float64 {
	_, bestInstantValue := bestHoldableInstant(p, g)

	bestSorceryValue := 0.0
	for _, card := range g.GetCastableSpells(p.PlayerID()) {
		if card.HasType(core.TypeInstant) {
			continue
		}
		sv := float64(eval.SpellValue(card, p, g))
		if sv > bestSorceryValue {
			bestSorceryValue = sv
		}
	}

	if w.HoldInstants <= 0 {
		return 0
	}

	threshold := bestSorceryValue / w.HoldInstants
	if bestInstantValue >= threshold && bestInstantValue > 0 {
		return bestInstantValue - threshold
	}
	return 0
}

// instantToHold returns the instant the AI should keep mana up for this main
// phase, or nil when no instant is worth holding. Callers reserve the returned
// card's mana when making sorcery-speed plays so the AI develops its board
// instead of passing the whole main phase to hold a single trick.
func instantToHold(p mage.Player, g *mage.Game, w ai.WeightedPersonality) mage.Card {
	if holdBackValue(p, g, w) <= 0 {
		return nil
	}
	card, _ := bestHoldableInstant(p, g)
	return card
}

func (s *Strategy) evaluateResponse(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return nil
	}

	stackHasThreat := false
	if len(g.StackObjects()) > 0 {
		for _, obj := range g.StackObjects() {
			if obj.Controller != playerID {
				stackHasThreat = true
				break
			}
		}
	}

	// Detect combat phase — instants are more valuable during combat.
	inCombat := g.GetStep() == core.DeclareAttackers || g.GetStep() == core.DeclareBlockers ||
		g.GetStep() == core.CombatDamage || g.GetStep() == core.FirstStrikeDamage

	var bestAction *interactive.PriorityAction
	bestValue := 0

	for _, card := range p.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost(), mage.SpellContextForCard(card)) {
			continue
		}
		if !aiHintAllowsTiming(cardAIHint(card), g, false) {
			continue
		}
		// Pump tricks are most informed once blockers are declared (we know
		// which attackers are blocked, so we can pump for lethal or to save a
		// creature). Hold them through the declare-attackers window unless a
		// threat is already on the stack that we need to respond to.
		if g.GetStep() == core.DeclareAttackers && !stackHasThreat &&
			combatsolver.ClassifyCombat(card) == combatsolver.RolePump {
			continue
		}

		hasUsableEffect := false
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
				if mage.SpellOutcome(sa.Effects()) != mage.OutcomeUnknown {
					hasUsableEffect = true
					break
				}
			}
		}
		if !hasUsableEffect {
			continue
		}

		sv := eval.SpellValue(card, p, g)

		if stackHasThreat {
			sv += 3
		}

		for _, a := range card.Abilities() {
			sa, ok := a.(*mage.SpellAbility)
			if !ok || sa.Kind() != mage.ActionSpell {
				continue
			}
			outcome := mage.SpellOutcome(sa.Effects())
			if outcome == mage.OutcomeDetriment {
				sv += 2
				// Removal during combat is especially valuable.
				if inCombat {
					sv += 3
				}
			}
			// Combat tricks (buff spells) are best during combat.
			if outcome == mage.OutcomeBenefit && inCombat {
				sv += 4
			}
		}

		if sv > bestValue {
			targets := s.autoSelectTargets(p, g, card)
			if len(targets) > 0 {
				bestValue = sv
				bestAction = &interactive.PriorityAction{
					Type:     interactive.ActionCastSpell,
					CardID:   card.ID(),
					CardName: card.Name(),
					Targets:  targets,
					XValue:   bestXValue(g, playerID, card, targets),
				}
			}
		}
	}

	if bestAction != nil && bestValue >= 3 {
		return bestAction
	}

	return nil
}
