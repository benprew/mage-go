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

// CombatScore summarizes the outcome of a combat step.
type CombatScore struct {
	DamageToOpponent   int
	OurCreaturesLost   int
	TheirCreaturesLost int
	LifeGained         int // life gained by our lifelink attackers
	OpponentLifeGained int // life gained by opponent's lifelink blockers
	Score              int
}

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

func evaluateCombatOutcome(g *mage.Game, playerID uuid.UUID, attackers []uuid.UUID, blocks []mage.BlockAssignment) CombatScore {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return CombatScore{}
	}

	blockerMap := make(map[uuid.UUID][]uuid.UUID)
	for _, b := range blocks {
		blockerMap[b.AttackerID] = append(blockerMap[b.AttackerID], b.BlockerID)
	}

	res := simulateCombatDamage(g, attackers, blockerMap, nil)

	cs := CombatScore{
		DamageToOpponent:   res.damageToOpponent,
		LifeGained:         res.lifeGained,
		OpponentLifeGained: res.opponentLifeGained,
	}

	// Tally kills and lost value from accumulated damage (including deathtouch marks).
	var ourLostValue, theirLostValue int
	for _, atkID := range attackers {
		atk := g.FindPermanent(atkID)
		if atk == nil {
			continue
		}
		blockerIDs := blockerMap[atkID]
		if len(blockerIDs) == 0 {
			continue
		}
		if res.isDead(atkID, atk.CurrentToughness(g)) {
			cs.OurCreaturesLost++
			ourLostValue += eval.EvalCreatureInGame(atk, g)
		}
		for _, blkID := range blockerIDs {
			blk := g.FindPermanent(blkID)
			if blk == nil {
				continue
			}
			if res.isDead(blkID, blk.CurrentToughness(g)) {
				cs.TheirCreaturesLost++
				theirLostValue += eval.EvalCreatureInGame(blk, g)
			}
		}
	}

	cs.Score = cs.DamageToOpponent + theirLostValue - ourLostValue + cs.LifeGained - cs.OpponentLifeGained

	return cs
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

func findGangBlocks(atk *mage.Permanent, available []*mage.Permanent, g *mage.Game, playerID uuid.UUID, theyHaveLethal bool) []*mage.Permanent {
	atkTough := atk.CurrentToughness(g)
	atkScore := eval.EvalCreatureInGame(atk, g)

	// Try all 2-blocker combinations first.
	for i := range available {
		if available[i] == nil {
			continue
		}
		b1 := available[i]
		if !mage.CanBlock(b1, atk, g) {
			continue
		}

		for j := i + 1; j < len(available); j++ {
			if available[j] == nil {
				continue
			}
			b2 := available[j]
			if !mage.CanBlock(b2, atk, g) {
				continue
			}

			combinedPow := b1.CurrentPower(g) + b2.CurrentPower(g)
			if combinedPow < atkTough {
				continue
			}

			b1Score := eval.EvalCreatureInGame(b1, g)
			b2Score := eval.EvalCreatureInGame(b2, g)
			combinedScore := b1Score + b2Score
			// Allow gang block if attacker is worth at least 80% of blockers.
			if !theyHaveLethal && atkScore*100 < combinedScore*80 {
				continue
			}

			return []*mage.Permanent{b1, b2}
		}
	}

	// Try all 3-blocker combinations.
	for i := range available {
		if available[i] == nil {
			continue
		}
		b1 := available[i]
		if !mage.CanBlock(b1, atk, g) {
			continue
		}

		for j := i + 1; j < len(available); j++ {
			if available[j] == nil {
				continue
			}
			b2 := available[j]
			if !mage.CanBlock(b2, atk, g) {
				continue
			}

			for k := j + 1; k < len(available); k++ {
				if available[k] == nil {
					continue
				}
				b3 := available[k]
				if !mage.CanBlock(b3, atk, g) {
					continue
				}

				combinedPow := b1.CurrentPower(g) + b2.CurrentPower(g) + b3.CurrentPower(g)
				if combinedPow < atkTough {
					continue
				}

				b1Score := eval.EvalCreatureInGame(b1, g)
				b2Score := eval.EvalCreatureInGame(b2, g)
				b3Score := eval.EvalCreatureInGame(b3, g)
				combinedScore := b1Score + b2Score + b3Score
				// Allow gang block if attacker is worth at least 80% of blockers.
				if !theyHaveLethal && atkScore*100 < combinedScore*80 {
					continue
				}

				return []*mage.Permanent{b1, b2, b3}
			}
		}
	}

	_ = playerID
	return nil
}

func canSingleBlockKill(atk *mage.Permanent, available []*mage.Permanent, g *mage.Game) bool {
	atkTough := atk.CurrentToughness(g)
	for _, blk := range available {
		if blk == nil {
			continue
		}
		if !mage.CanBlock(blk, atk, g) {
			continue
		}
		if blk.CurrentPower(g) >= atkTough {
			return true
		}
	}
	return false
}

func shouldAttack(atk *mage.Permanent, g *mage.Game, opponentID uuid.UUID, aggression float64) bool {
	if aggression >= 1.0 {
		return true
	}
	if profitableToAttack(atk, g, opponentID) {
		// Even if the attack is profitable against blockers, be cautious if
		// opponent has significant untapped mana (could have removal/combat tricks).
		if aggression < 0.5 {
			oppUntapped := eval.CountAvailableMana(g, opponentID)
			atkValue := eval.EvalCreatureInGame(atk, g)
			// If opponent has plenty of mana open and our creature is valuable,
			// consider holding back. High-value creatures are worth protecting.
			if oppUntapped >= 3 && atkValue >= 10 && !eval.IsEvasive(atk, opponentID, g) {
				return false
			}
		}
		return true
	}
	if aggression >= 0.7 {
		return marginallyProfitableToAttack(atk, g, opponentID)
	}
	return false
}

func marginallyProfitableToAttack(atk *mage.Permanent, g *mage.Game, opponentID uuid.UUID) bool {
	var bestBlocker *mage.Permanent
	bestPow := -1
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != opponentID || !perm.HasType(core.TypeCreature) || perm.Tapped {
			continue
		}
		if !mage.CanBlock(perm, atk, g) {
			continue
		}
		if mage.HasLandwalkEvasion(atk, opponentID, g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow > bestPow {
			bestPow = pow
			bestBlocker = perm
		}
	}
	if bestBlocker == nil {
		return true
	}
	atkPow := atk.CurrentPower(g)
	blkTough := bestBlocker.CurrentToughness(g)
	return atkPow >= blkTough
}

func shouldBlock(atkPow int, _ *mage.Game, _ uuid.UUID, blockThreshold float64) bool {
	if blockThreshold <= 0.0 {
		return true
	}
	if blockThreshold >= 0.95 {
		return false
	}
	minPow := int(blockThreshold * 10.0)
	return atkPow >= minPow
}

// evaluateSingleBlock returns true if blocking this attacker with this blocker
// is a net-positive trade. Uses combat simulation with first strike, deathtouch,
// and lifelink awareness.
func evaluateSingleBlock(atk, blk *mage.Permanent, g *mage.Game, playerID uuid.UUID) bool {
	atkPow := atk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkPow := blk.CurrentPower(g)
	blkTough := blk.CurrentToughness(g)

	atkHasDT := atk.HasKeyword(core.Deathtouch)
	blkHasDT := blk.HasKeyword(core.Deathtouch)
	atkHasFS := atk.HasKeyword(core.FirstStrike) || atk.HasKeyword(core.DoubleStrike)
	blkHasFS := blk.HasKeyword(core.FirstStrike) || blk.HasKeyword(core.DoubleStrike)

	// Does the blocker kill the attacker?
	blockerKillsAtk := blkPow >= atkTough || (blkHasDT && blkPow > 0)
	// Does the attacker kill the blocker?
	attackerKillsBlk := atkPow >= blkTough || (atkHasDT && atkPow > 0)

	// First strike advantage: if blocker has FS and attacker doesn't,
	// and blocker kills attacker, the blocker survives.
	if blkHasFS && !atkHasFS && blockerKillsAtk {
		return true // blocker wins before attacker deals damage
	}
	// Converse: attacker has FS and blocker doesn't, attacker kills blocker
	// before blocker deals damage. Only block if we must (handled by caller).
	if atkHasFS && !blkHasFS && attackerKillsBlk && !blockerKillsAtk {
		return false // blocker dies for nothing
	}

	atkValue := eval.EvalCreatureInGame(atk, g)
	blkValue := eval.EvalCreatureInGame(blk, g)

	// If blocker kills attacker (mutual trade or blocker survives), check value trade.
	if blockerKillsAtk {
		if !attackerKillsBlk {
			return true // blocker survives, attacker dies
		}
		// Mutual trade: profitable if attacker is worth at least as much.
		return atkValue >= blkValue
	}

	// Blocker doesn't kill attacker but survives: damage soak (reduces damage to face).
	if !attackerKillsBlk {
		return atkPow >= 3 // worth chump-blocking big attackers to save life
	}

	// Blocker dies without killing attacker: only if damage prevented is significant.
	return atkPow >= 4 || atkPow*2 >= g.GetPlayer(playerID).Life()
}

func profitableToAttack(atk *mage.Permanent, g *mage.Game, opponentID uuid.UUID) bool {
	var bestBlocker *mage.Permanent
	bestPow := -1
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != opponentID || !perm.HasType(core.TypeCreature) || perm.Tapped {
			continue
		}
		if !mage.CanBlock(perm, atk, g) {
			continue
		}
		if mage.HasLandwalkEvasion(atk, opponentID, g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow > bestPow {
			bestPow = pow
			bestBlocker = perm
		}
	}

	if bestBlocker == nil {
		return true
	}

	atkPow := atk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkPow := bestBlocker.CurrentPower(g)
	blkTough := bestBlocker.CurrentToughness(g)

	atkSurvives := blkPow < atkTough
	blkDies := atkPow >= blkTough

	if atkSurvives {
		return true
	}
	if blkDies {
		return bestBlocker.Card.ManaCost().CMC() >= atk.Card.ManaCost().CMC()
	}
	return false
}

func holdBackValue(p mage.Player, g *mage.Game, w ai.WeightedPersonality) float64 {
	playerID := p.PlayerID()

	bestInstantValue := 0.0
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
				outcome := mage.SpellOutcome(sa.Effects())
				if outcome == mage.OutcomeDetriment {
					sv *= 1.5
				}
				if outcome == mage.OutcomeBenefit {
					sv *= 1.3
				}
			}
		}
		if sv > bestInstantValue {
			bestInstantValue = sv
		}
	}

	bestSorceryValue := 0.0
	for _, card := range g.GetCastableSpells(playerID) {
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

func (s *Strategy) evaluateResponseLegacy(p mage.Player, g *mage.Game) *interactive.PriorityAction {
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
				if inCombat {
					sv += 3
				}
			}
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

func raceInformedAttack(perm *mage.Permanent, g *mage.Game, opponentID uuid.UUID, race eval.RaceInfo) bool {
	if !race.Racing {
		return true
	}
	if race.MyClock < race.TheirClock {
		return true
	}
	if race.TheirClock < race.MyClock {
		return eval.IsEvasive(perm, opponentID, g)
	}
	if eval.IsEvasive(perm, opponentID, g) {
		return true
	}
	return profitableToAttack(perm, g, opponentID)
}

func raceInformedBlock(atk *mage.Permanent, blk *mage.Permanent, g *mage.Game, race eval.RaceInfo) bool {
	if !race.Racing {
		return true
	}

	atkPow := atk.CurrentPower(g)

	if race.MyClock < race.TheirClock {
		me := g.GetPlayer(blk.Controller)
		if me != nil && atkPow*4 < me.Life() {
			return false
		}
		return true
	}

	if race.TheirClock < race.MyClock {
		blkPow := blk.CurrentPower(g)
		atkTough := atk.CurrentToughness(g)
		if blkPow >= atkTough {
			return true
		}
		return true
	}

	blkPow := blk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkTough := blk.CurrentToughness(g)
	if blkPow >= atkTough {
		return true
	}
	if atkPow < blkTough {
		return true
	}
	return false
}
