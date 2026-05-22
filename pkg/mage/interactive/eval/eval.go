// Package eval provides position evaluation and scoring for the MTG engine.
// It is a leaf dependency — it imports only pkg/mage and pkg/mage/core.
//
// The core type is [StateEvaluator], a function that scores a game position
// from a player's perspective. [DefaultEvaluator] uses hardcoded weight
// constants; [WeightedEvaluator] and [NewWeightedEvaluator] accept a [Weights]
// struct for personality-driven evaluation.
//
// Supporting functions score individual creatures ([EvalCreature]), abilities
// ([AbilityQuality]), spells ([SpellValue]), and board positions
// ([CalculateLethal], [CalculateRace]).
package eval

import (
	"math"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// Weights holds evaluation weights used by WeightedEvaluator and NewWeightedEvaluator.
type Weights struct {
	Life  float64 // importance of life difference
	Board float64 // importance of creature board
	Card  float64 // importance of hand advantage
	Mana  float64 // importance of mana development
	Tempo float64 // importance of untapped mana
}

// Evaluation weight constants — default parameters used by DefaultEvaluator.
const (
	LifeWeight        = 3   // multiplier for (myLife - oppLife)
	PowerWeight       = 2   // per point of creature power
	ToughnessWeight   = 1   // per point of creature toughness
	CardWeight        = 2   // per card of hand advantage
	LandWeight        = 1   // per own land
	NonCreatureCMCDiv = 2   // non-creature perm value = CMC / NonCreatureCMCDiv
	LethalBonus       = 100 // bonus/penalty for having/facing lethal on board
)

// StateEvaluator scores a game position from playerID's perspective.
// Higher scores are better for playerID.
type StateEvaluator func(g *mage.Game, playerID uuid.UUID) int

// DefaultEvaluator is the standard position evaluator.
var DefaultEvaluator StateEvaluator = defaultEvaluate

// WeightedEvaluator returns a StateEvaluator that uses the given
// Weights instead of the default constants.
func WeightedEvaluator(w Weights) StateEvaluator {
	return func(g *mage.Game, playerID uuid.UUID) int {
		return weightedEvaluate(g, playerID, w)
	}
}

func defaultEvaluate(g *mage.Game, playerID uuid.UUID) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := (me.Life() - opp.Life()) * LifeWeight

	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		v := EvalCreatureInGame(perm, g)
		switch perm.Controller {
		case playerID:
			score += v
		case oppID:
			score -= v
		}
	}

	nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
	for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
		v := evalNonCreaturePermanent(perm)
		switch perm.Controller {
		case playerID:
			score += v
		case oppID:
			score -= v
		}
	}

	score += (handQuality(me, g) - handQuality(opp, g)) * CardWeight

	ownLand := mage.And(mage.IsLand, mage.ControlledBy(playerID))
	score += g.CountBattlefield(ownLand) * LandWeight

	untappedMana := countUntappedManaSources(g, playerID)
	score += untappedMana * 2

	lethal := calculateLethalOnBoard(g, playerID)
	score += lethal

	// Race clock integration: when both sides are on a short clock, reward
	// positions where our clock is shorter than theirs.
	race := CalculateRace(g, playerID)
	if race.Racing {
		clockAdv := race.TheirClock - race.MyClock
		score += clockAdv * 6
	}

	return score
}

// NewWeightedEvaluator creates a StateEvaluator that uses Weights
// with role-based permanent classification, hand quality scoring,
// board diversity, tempo, and lethal detection.
func NewWeightedEvaluator(w Weights) StateEvaluator {
	return func(g *mage.Game, playerID uuid.UUID) int {
		me := g.GetPlayer(playerID)
		opp := g.GetOpponent(playerID)
		if me == nil || opp == nil {
			return 0
		}
		oppID := opp.PlayerID()

		score := 0.0

		score += float64(me.Life()-opp.Life()) * w.Life

		allPerms := g.FilterBattlefield(mage.IsCreature)
		for _, perm := range allPerms {
			v := float64(EvalCreatureInGame(perm, g))
			role := ClassifyPermanent(perm)
			roleWeight := roleWeightForPersonality(role, w)
			switch perm.Controller {
			case playerID:
				score += v * roleWeight
			case oppID:
				score -= v * roleWeight
			}
		}

		nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
		for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
			v := float64(evalNonCreaturePermanent(perm))
			role := ClassifyPermanent(perm)
			roleWeight := roleWeightForPersonality(role, w)
			switch perm.Controller {
			case playerID:
				score += v * roleWeight
			case oppID:
				score -= v * roleWeight
			}
		}

		myHandQuality := handQuality(me, g)
		oppHandQuality := handQuality(opp, g)
		score += float64(myHandQuality-oppHandQuality) * w.Card

		ownLandCount := g.CountBattlefield(mage.And(mage.IsLand, mage.ControlledBy(playerID)))
		oppLandCount := g.CountBattlefield(mage.And(mage.IsLand, mage.ControlledBy(oppID)))
		score += float64(ownLandCount-oppLandCount) * w.Mana

		myUntapped := countUntappedManaSources(g, playerID)
		oppUntapped := countUntappedManaSources(g, oppID)
		score += float64(myUntapped-oppUntapped) * w.Tempo

		myRoles := countRoles(g.FilterBattlefield(mage.ControlledBy(playerID)), func(_ *mage.Permanent) bool { return true })
		oppRoles := countRoles(g.FilterBattlefield(mage.ControlledBy(oppID)), func(_ *mage.Permanent) bool { return true })
		score += float64(boardDiversity(myRoles)-boardDiversity(oppRoles)) * w.Board * 0.5

		lethal := calculateLethalOnBoard(g, playerID)
		score += float64(lethal)

		// Opponent threat anticipation: penalize positions where opponent has
		// untapped mana and we have vulnerable creatures. This models the risk
		// of walking into removal or combat tricks.
		oppUntapped2 := countUntappedManaSources(g, oppID)
		if oppUntapped2 >= 2 {
			// Penalty scales with opponent's available mana (more mana = more threat).
			// Reduced if we have many creatures (one removal doesn't wreck us).
			myCreatureCount := g.CountBattlefield(mage.And(mage.IsCreature, mage.ControlledBy(playerID)))
			if myCreatureCount <= 2 && myCreatureCount > 0 {
				score -= float64(oppUntapped2) * 0.5
			}
		}

		return int(math.Round(score))
	}
}

// NewPersonalityEvaluator creates a StateEvaluator that uses Weights plus
// the Aggression decision weight. Aggression biases the evaluator toward
// board positions where our creatures can attack profitably: untapped creatures
// and higher power get a bonus, while tapped-out boards get a small penalty.
// This makes aggro personalities prefer attacking positions and control
// personalities prefer defensive positions, even within the same search.
func NewPersonalityEvaluator(w Weights, aggression float64) StateEvaluator {
	base := NewWeightedEvaluator(w)
	if aggression < 0.01 && aggression > -0.01 {
		// No aggression bias — just use the base evaluator.
		return base
	}
	return func(g *mage.Game, playerID uuid.UUID) int {
		score := base(g, playerID)

		me := g.GetPlayer(playerID)
		opp := g.GetOpponent(playerID)
		if me == nil || opp == nil {
			return score
		}
		oppID := opp.PlayerID()

		// Aggression bonus: reward having untapped attackers with power.
		// Penalty: opponent having untapped creatures means we're not pressuring.
		aggrBonus := 0.0
		for _, perm := range g.FilterBattlefield(mage.IsCreature) {
			switch perm.Controller {
			case playerID:
				power := float64(perm.CurrentPower(g))
				if !perm.Tapped {
					aggrBonus += power * 0.5 // untapped = ready to attack
				} else {
					aggrBonus += power * 0.2 // tapped = already attacked or used
				}
			case oppID:
				power := float64(perm.CurrentPower(g))
				if !perm.Tapped {
					aggrBonus -= power * 0.3 // opponent blocker threat
				}
			}
		}

		// Scale by aggression: 1.0 = full aggro bonus, 0.0 = none
		// Negative effective aggression (control) slightly inverts — values
		// defensive positions instead.
		score += int(math.Round(aggrBonus * aggression))

		return score
	}
}

func roleWeightForPersonality(role PermanentRole, w Weights) float64 {
	switch role {
	case RoleThreat:
		return w.Board / 3.0
	case RoleUtility:
		return w.Card / 2.0
	case RoleEngine:
		return w.Card / 2.0
	case RoleMana:
		return w.Mana
	case RoleDefense:
		return w.Board / 3.0
	default:
		return 1.0
	}
}

func handQuality(p mage.Player, g *mage.Game) int {
	hand := p.Hand()
	if len(hand) == 0 {
		return 0
	}

	availMana := countUntappedManaSources(g, p.PlayerID())

	score := 0
	landCount := 0
	for _, card := range hand {
		if card.HasType(core.TypeLand) {
			landCount++
			if landCount <= 5 {
				score++
			} else {
				score-- // flood penalty
			}
		} else {
			cmc := card.ManaCost().CMC()
			if cmc <= availMana {
				score += 2 // castable now
			} else if cmc <= availMana+2 {
				score++ // castable soon
			}
			// else: dead card, no value
		}
	}
	return score
}

func countUntappedManaSources(g *mage.Game, playerID uuid.UUID) int {
	count := 0
	own := mage.And(mage.ControlledBy(playerID), mage.IsUntapped)
	for _, perm := range g.FilterBattlefield(own) {
		if perm.HasType(core.TypeLand) {
			count++
			continue
		}
		if hasManaAbility(perm) {
			count++
		}
	}
	return count
}

func calculateLethalOnBoard(g *mage.Game, playerID uuid.UUID) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := 0
	myDmg := estimateExpectedDamage(g, playerID, oppID)
	if myDmg >= opp.Life() && myDmg > 0 {
		score += LethalBonus
	}
	theirDmg := estimateExpectedDamage(g, oppID, playerID)
	if theirDmg >= me.Life() && theirDmg > 0 {
		score -= LethalBonus
	}
	return score
}

func weightedEvaluate(g *mage.Game, playerID uuid.UUID, w Weights) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := 0.0

	score += float64(me.Life()-opp.Life()) * w.Life

	boardScale := w.Board / 2.0
	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		v := float64(EvalCreatureInGame(perm, g)) * boardScale
		switch perm.Controller {
		case playerID:
			score += v
		case oppID:
			score -= v
		}
	}

	nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
	for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
		v := float64(evalNonCreaturePermanent(perm)) * boardScale
		switch perm.Controller {
		case playerID:
			score += v
		case oppID:
			score -= v
		}
	}

	score += float64(len(me.Hand())-len(opp.Hand())) * w.Card

	ownLand := mage.And(mage.IsLand, mage.ControlledBy(playerID))
	score += float64(g.CountBattlefield(ownLand)) * w.Mana

	score += float64(g.CountBattlefield(mage.And(ownLand, mage.IsUntapped))) * w.Tempo

	return int(score)
}

// EvalCreature scores a single creature permanent from its controller's perspective.
// It uses base P/T + counters but does NOT include continuous effects (Giant Growth,
// Crusade, equipment, etc.). Use EvalCreatureInGame for accurate evaluation when
// a *mage.Game is available.
func EvalCreature(perm *mage.Permanent) int {
	score := permPower(perm)*PowerWeight + permToughness(perm)*ToughnessWeight
	if perm.Tapped {
		score = score * 2 / 3
	}
	if perm.HasAttr(core.AttrSummonSick) && !perm.HasAttr(core.Haste) {
		score /= 2
	}
	score += keywordBonus(perm)
	score += abilityBonus(perm)
	return score
}

// EvalCreatureInGame scores a single creature permanent using CurrentPower and
// CurrentToughness, which include continuous effects (pumps, anthems, equipment).
// Prefer this over EvalCreature whenever a *mage.Game is available.
func EvalCreatureInGame(perm *mage.Permanent, g *mage.Game) int {
	return evalCreatureInGame(perm, g, true)
}

// EvalCreatureInGameNoTapPenalty is like EvalCreatureInGame but skips the
// tap-state penalty. Use it when evaluating mid-turn positions where the
// creature's tapped state will resolve at the next untap step (e.g., the
// active player's tapped attackers in a combat-decision leaf eval).
func EvalCreatureInGameNoTapPenalty(perm *mage.Permanent, g *mage.Game) int {
	return evalCreatureInGame(perm, g, false)
}

func evalCreatureInGame(perm *mage.Permanent, g *mage.Game, applyTapPenalty bool) int {
	score := perm.CurrentPower(g)*PowerWeight + perm.CurrentToughness(g)*ToughnessWeight
	if applyTapPenalty && perm.Tapped {
		score = score * 2 / 3
	}
	if perm.HasAttr(core.AttrSummonSick) && !perm.HasAttr(core.Haste) {
		score /= 2
	}
	score += keywordBonusInGame(perm, g)
	score += abilityBonus(perm)
	// SBAs destroy lethally-damaged creatures before eval runs, so only
	// sub-lethal damage reaches here. A mild penalty reflects vulnerability
	// (one more bolt kills it) — combat damage clears at cleanup, so this
	// is a signal, not a permanent loss.
	if perm.Damage > 0 {
		tough := perm.CurrentToughness(g)
		if tough > 0 {
			score -= score * perm.Damage / (tough * 3)
		}
	}
	return score
}

func permPower(p *mage.Permanent) int {
	pw := p.Card.Power()
	if p.BasePTOverride != nil {
		pw = p.BasePTOverride[0]
	}
	for ct := range core.NumCounters {
		if n := p.Counters[ct]; n != 0 {
			pw += ct.PowerBoost() * int(n)
		}
	}
	return pw
}

func permToughness(p *mage.Permanent) int {
	tg := p.Card.Toughness()
	if p.BasePTOverride != nil {
		tg = p.BasePTOverride[1]
	}
	for ct := range core.NumCounters {
		if n := p.Counters[ct]; n != 0 {
			tg += ct.ToughnessBoost() * int(n)
		}
	}
	return tg
}

func keywordBonusInGame(perm *mage.Permanent, g *mage.Game) int {
	power := max(perm.CurrentPower(g), 0)
	toughness := max(perm.CurrentToughness(g), 0)
	combatSize := max(power+toughness, 1)
	score := 0

	if perm.HasKeyword(core.UnblockableKW) {
		score += 3 + power
	}
	if perm.HasKeyword(core.Flying) {
		score += 2 + power
	}
	if perm.HasKeyword(core.Fear) {
		score += 1 + power
	}
	if perm.HasKeyword(core.Menace) {
		score += max(2, power/2+1)
	}
	if perm.HasKeyword(core.Trample) {
		score += max(1, power-1)
	}
	for _, kw := range []core.Attr{
		core.Islandwalk, core.Swampwalk, core.Forestwalk,
		core.Mountainwalk, core.Plainswalk,
	} {
		if perm.HasKeyword(kw) {
			score += max(2, power)
		}
	}
	if perm.HasKeyword(core.CantBeBlockedByWalls) {
		score++
	}
	if perm.HasKeyword(core.CantBeBlockedExceptByWalls) {
		score -= max(2, power)
	}
	if perm.HasKeyword(core.DoubleStrike) {
		score += max(4, power*2)
	}
	if perm.HasKeyword(core.FirstStrike) {
		score += max(2, power)
	}
	if perm.HasKeyword(core.Haste) {
		score += max(1, power/2+1)
	}
	if perm.HasKeyword(core.Indestructible) {
		score += max(4, combatSize/2)
	}
	if perm.HasKeyword(core.Hexproof) {
		score += max(2, combatSize/3)
	}
	if perm.HasKeyword(core.Shroud) {
		score += max(2, combatSize/4)
	}
	if perm.HasKeyword(core.BasiliskTouch) {
		score += max(2, toughness/2+1)
	}
	if perm.HasKeyword(core.Deathtouch) {
		score += max(3, toughness/2+1)
	}
	if perm.HasKeyword(core.Lifelink) {
		score += max(2, power)
	}
	if perm.HasKeyword(core.Vigilance) {
		score += max(1, power/2+toughness/3)
	}
	if perm.HasKeyword(core.Reach) {
		score += max(1, toughness/3)
	}
	if perm.HasKeyword(core.CanBlockAdditional) {
		score += max(1, toughness/3)
	}
	if perm.HasKeyword(core.CanBlockAny) {
		score += max(1, toughness/3)
	}
	if perm.HasKeyword(core.MustBeBlocked) {
		score += max(1, power/2)
	}
	if perm.HasKeyword(core.Banding) {
		score += max(1, combatSize/4)
	}
	if perm.HasKeyword(core.Defender) {
		score -= max(2, power+1)
	}
	if perm.HasKeyword(core.DoesNotUntapKW) {
		score -= max(2, combatSize/3)
	}
	if perm.HasKeyword(core.MustAttack) {
		score -= max(1, toughness/3)
	}

	return score
}

func keywordBonus(perm *mage.Permanent) int {
	score := 0

	if perm.HasKeyword(core.UnblockableKW) {
		score += 5
	}
	if perm.HasKeyword(core.Flying) {
		score += 4
	}
	if perm.HasKeyword(core.Fear) {
		score += 3
	}
	if perm.HasKeyword(core.Menace) {
		score += 2
	}
	if perm.HasKeyword(core.Trample) {
		score += 2
	}
	for _, kw := range []core.Attr{
		core.Islandwalk, core.Swampwalk, core.Forestwalk,
		core.Mountainwalk, core.Plainswalk,
	} {
		if perm.HasKeyword(kw) {
			score += 2
		}
	}
	if perm.HasKeyword(core.CantBeBlockedByWalls) {
		score++
	}
	if perm.HasKeyword(core.CantBeBlockedExceptByWalls) {
		score -= 2
	}
	if perm.HasKeyword(core.DoubleStrike) {
		score += 4
	}
	if perm.HasKeyword(core.FirstStrike) {
		score += 2
	}
	if perm.HasKeyword(core.Haste) {
		score += 2
	}
	if perm.HasKeyword(core.Indestructible) {
		score += 5
	}
	if perm.HasKeyword(core.Hexproof) {
		score += 3
	}
	if perm.HasKeyword(core.Shroud) {
		score += 2
	}
	if perm.HasKeyword(core.BasiliskTouch) {
		score += 2
	}
	if perm.HasKeyword(core.Deathtouch) {
		score += 3
	}
	if perm.HasKeyword(core.Lifelink) {
		score += 2
	}
	if perm.HasKeyword(core.Vigilance) {
		score += 2
	}
	if perm.HasKeyword(core.Reach) {
		score++
	}
	if perm.HasKeyword(core.CanBlockAdditional) {
		score++
	}
	if perm.HasKeyword(core.CanBlockAny) {
		score++
	}
	if perm.HasKeyword(core.MustBeBlocked) {
		score++
	}
	if perm.HasKeyword(core.Banding) {
		score++
	}
	if perm.HasKeyword(core.Defender) {
		score -= 2
	}
	if perm.HasKeyword(core.DoesNotUntapKW) {
		score -= 2
	}
	if perm.HasKeyword(core.MustAttack) {
		score--
	}

	return score
}

func abilityBonus(perm *mage.Permanent) int {
	score := 0
	for _, a := range perm.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		switch ab := inner.(type) {
		case *mage.ManaAbility:
			if ab.HasAnyColor() {
				score += 3
			} else {
				score += 2
			}
		case mage.ActivatedAbility:
			score += AbilityQuality(ab)
		case mage.TriggeredAbility:
			score += triggeredAbilityQuality(ab)
		}
	}
	return score
}

// AbilityQuality scores an activated ability by its effect type and cost efficiency.
// isEvasionKeyword reports whether a granted keyword makes a creature harder
// to block (often unblockable in practice on the relevant board state).
func isEvasionKeyword(kw core.Attr) bool {
	switch kw {
	case core.Flying, core.Fear, core.UnblockableKW:
		return true
	}
	return false
}

func AbilityQuality(ab mage.ActivatedAbility) int {
	hasTapCost := false
	manaCostTotal := 0
	for _, c := range ab.Costs() {
		if c.Text() == "{T}" {
			hasTapCost = true
		}
		if mcp, ok := c.(*mage.ManaCostPayment); ok {
			manaCostTotal += mcp.MC.CMC()
		}
	}

	bestScore := 0
	if ad, ok := ab.(*mage.ActionDefinition); ok {
		for _, hint := range ad.AIHints() {
			bestScore = max(bestScore, abilityHintQuality(hint))
		}
	}
	for _, e := range ab.Effects() {
		props := e.Properties()
		bestScore = max(bestScore, abilityHintQuality(mage.AIHint{
			Roles:     props.AIRoles,
			ValueBias: props.ValueBias,
		}))

		if props.DrawCount > 0 {
			s := 5
			if manaCostTotal > 2 {
				s = 3
			}
			if s > bestScore {
				bestScore = s
			}
		}

		if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
			s := 4
			if !hasTapCost && manaCostTotal >= 3 {
				s = 2
			}
			if s > bestScore {
				bestScore = s
			}
		}

		if props.Outcome == mage.OutcomeDetriment && props.DamageValue == nil && props.DrawCount == 0 {
			s := 3
			if manaCostTotal >= 3 {
				s = 2
			}
			if s > bestScore {
				bestScore = s
			}
		}

		if props.Outcome == mage.OutcomeBenefit {
			s := 2
			if manaCostTotal >= 4 {
				s = 1
			}
			// Evasion-granting abilities are tactical: they can convert a
			// blocked attacker into lethal damage. Score above the activation
			// threshold so the heuristic and minimax search both consider them.
			if isEvasionKeyword(props.GrantedKeyword) {
				s = 4
				if manaCostTotal >= 4 {
					s = 3
				}
			}
			if s > bestScore {
				bestScore = s
			}
		}
	}

	if bestScore == 0 {
		if hasTapCost && manaCostTotal == 0 {
			return 2
		}
		if manaCostTotal <= 2 {
			return 1
		}
		return 1
	}

	return bestScore
}

func abilityHintQuality(hint mage.AIHint) int {
	score := hint.ValueBias
	for _, role := range hint.Roles {
		switch role {
		case mage.AIRoleCardDraw, mage.AIRoleEngine:
			score = max(score, 5)
		case mage.AIRoleRemoval, mage.AIRoleBurn:
			score = max(score, 4)
		case mage.AIRolePump, mage.AIRoleProtection, mage.AIRoleCombatTrick:
			score = max(score, 3)
		case mage.AIRoleManaSink:
			score = max(score, 2)
		case mage.AIRoleFinisher:
			score = max(score, 6)
		}
	}
	return score
}

// evalNonCreaturePermanent scores a non-creature, non-land permanent (enchantment,
// artifact, etc.) by examining its abilities rather than just using CMC/2.
// Returns a value that is at least CMC/NonCreatureCMCDiv (the old baseline).
func evalNonCreaturePermanent(perm *mage.Permanent) int {
	baseValue := perm.Card.ManaCost().CMC() / NonCreatureCMCDiv

	bonus := 0
	for _, a := range perm.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		switch ab := inner.(type) {
		case *mage.ManaAbility:
			// Mana-producing artifacts get a small bonus.
			s := 2
			if ab.HasAnyColor() {
				s = 3
			}
			if s > bonus {
				bonus = s
			}
		case mage.ActivatedAbility:
			q := AbilityQuality(ab)
			if q > bonus {
				bonus = q
			}
		case mage.TriggeredAbility:
			q := triggeredAbilityQuality(ab)
			if q > bonus {
				bonus = q
			}
		}
	}

	// Permanents with abilities we couldn't classify still get a small bonus
	// for having any runtime abilities (e.g., continuous effects, static abilities).
	if len(perm.RuntimeAbilities) > 0 && bonus == 0 {
		bonus = 1
	}

	value := max(baseValue+bonus, baseValue)
	return value
}

func triggeredAbilityQuality(ab mage.TriggeredAbility) int {
	bestScore := 0
	for _, e := range ab.Effects() {
		props := e.Properties()
		if props.DrawCount > 0 {
			s := 4
			if s > bestScore {
				bestScore = s
			}
		}
		if props.DamageValue != nil {
			s := 3
			if s > bestScore {
				bestScore = s
			}
		}
		if props.Outcome == mage.OutcomeBenefit {
			s := 2
			if s > bestScore {
				bestScore = s
			}
		}
		if props.Outcome == mage.OutcomeDetriment {
			s := 2
			if s > bestScore {
				bestScore = s
			}
		}
	}
	if bestScore == 0 {
		return 1
	}
	return bestScore
}
