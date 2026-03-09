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
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// Weights holds evaluation weights used by WeightedEvaluator and NewWeightedEvaluator.
type Weights struct {
	Life  float64 // importance of life difference
	Board float64 // importance of creature board
	Card  float64 // importance of hand advantage
	Mana  float64 // importance of mana development
	Tempo float64 // importance of untapped mana
}

// GameReader is the subset of *mage.Game methods used by StateEvaluator.
// *mage.Game satisfies this interface automatically.
type GameReader interface {
	GetPlayer(uuid.UUID) mage.Player
	GetOpponent(uuid.UUID) mage.Player
	FilterBattlefield(mage.PermanentFilter) []*mage.Permanent
	CountBattlefield(mage.PermanentFilter) int
}

// Compile-time check that *mage.Game satisfies GameReader.
var _ GameReader = (*mage.Game)(nil)

// Evaluation weight constants — default parameters used by DefaultEvaluator.
const (
	LifeWeight        = 3 // multiplier for (myLife - oppLife)
	PowerWeight       = 2 // per point of creature power
	ToughnessWeight   = 1 // per point of creature toughness
	CardWeight        = 2 // per card of hand advantage
	LandWeight        = 1   // per own land
	NonCreatureCMCDiv = 2   // non-creature perm value = CMC / NonCreatureCMCDiv
	LethalBonus       = 100 // bonus/penalty for having/facing lethal on board
)

// StateEvaluator scores a game position from playerID's perspective.
// Higher scores are better for playerID.
type StateEvaluator func(g GameReader, playerID uuid.UUID) int

// DefaultEvaluator is the standard position evaluator.
var DefaultEvaluator StateEvaluator = defaultEvaluate

// WeightedEvaluator returns a StateEvaluator that uses the given
// Weights instead of the default constants.
func WeightedEvaluator(w Weights) StateEvaluator {
	return func(g GameReader, playerID uuid.UUID) int {
		return weightedEvaluate(g, playerID, w)
	}
}

func defaultEvaluate(g GameReader, playerID uuid.UUID) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := (me.Life() - opp.Life()) * LifeWeight

	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		v := EvalCreature(perm)
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
			score -= v
		}
	}

	nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
	for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
		v := perm.Card.ManaCost().CMC() / NonCreatureCMCDiv
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
			score -= v
		}
	}

	score += (len(me.Hand()) - len(opp.Hand())) * CardWeight

	ownLand := mage.And(mage.IsLand, mage.ControlledBy(playerID))
	score += g.CountBattlefield(ownLand) * LandWeight

	untappedMana := countUntappedManaSources(g, playerID)
	score += untappedMana * 2

	lethal := calculateLethalOnBoard(g, playerID)
	score += lethal

	return score
}

// NewWeightedEvaluator creates a StateEvaluator that uses Weights
// with role-based permanent classification, hand quality scoring,
// board diversity, tempo, and lethal detection.
func NewWeightedEvaluator(w Weights) StateEvaluator {
	return func(g GameReader, playerID uuid.UUID) int {
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
			v := float64(EvalCreature(perm))
			role := ClassifyPermanent(perm)
			roleWeight := roleWeightForPersonality(role, w)
			if perm.Controller == playerID {
				score += v * roleWeight
			} else if perm.Controller == oppID {
				score -= v * roleWeight
			}
		}

		nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
		for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
			v := float64(perm.Card.ManaCost().CMC()) / float64(NonCreatureCMCDiv)
			role := ClassifyPermanent(perm)
			roleWeight := roleWeightForPersonality(role, w)
			if perm.Controller == playerID {
				score += v * roleWeight
			} else if perm.Controller == oppID {
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

func handQuality(p mage.Player, g GameReader) int {
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
				score += 1
			} else {
				score -= 1 // flood penalty
			}
		} else {
			cmc := card.ManaCost().CMC()
			if cmc <= availMana {
				score += 2 // castable now
			} else if cmc <= availMana+2 {
				score += 1 // castable soon
			}
			// else: dead card, no value
		}
	}
	return score
}

func countUntappedManaSources(g GameReader, playerID uuid.UUID) int {
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

func calculateLethalOnBoard(g GameReader, playerID uuid.UUID) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := 0

	myDamage := 0
	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		if perm.Controller == playerID && canPotentiallyAttack(perm) {
			myDamage += permPower(perm)
		}
	}
	if myDamage >= opp.Life() && myDamage > 0 {
		score += LethalBonus
	}

	theirDamage := 0
	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		if perm.Controller == oppID && canPotentiallyAttack(perm) {
			theirDamage += permPower(perm)
		}
	}
	if theirDamage >= me.Life() && theirDamage > 0 {
		score -= LethalBonus
	}

	return score
}

func canPotentiallyAttack(perm *mage.Permanent) bool {
	if perm.Tapped {
		return false
	}
	if perm.HasAttr(core.AttrSummonSick) && !perm.HasAttr(core.Haste) {
		return false
	}
	if perm.HasKeyword(core.Defender) {
		return false
	}
	return true
}

func weightedEvaluate(g GameReader, playerID uuid.UUID, w Weights) int {
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
		v := float64(EvalCreature(perm)) * boardScale
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
			score -= v
		}
	}

	nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
	for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
		v := float64(perm.Card.ManaCost().CMC()) / 2.0 * boardScale
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
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
func EvalCreature(perm *mage.Permanent) int {
	score := permPower(perm)*PowerWeight + permToughness(perm)*ToughnessWeight
	if perm.Tapped {
		score = score * 2 / 3
	}
	if perm.HasAttr(core.AttrSummonSick) && !perm.HasAttr(core.Haste) {
		score = score / 2
	}
	score += keywordBonus(perm)
	score += abilityBonus(perm)
	return score
}

func permPower(p *mage.Permanent) int {
	pw := p.Card.Power()
	if p.BasePTOverride != nil {
		pw = p.BasePTOverride[0]
	}
	for ct, n := range p.Counters {
		pw += ct.PowerBoost() * n
	}
	return pw
}

func permToughness(p *mage.Permanent) int {
	tg := p.Card.Toughness()
	if p.BasePTOverride != nil {
		tg = p.BasePTOverride[1]
	}
	for ct, n := range p.Counters {
		tg += ct.ToughnessBoost() * n
	}
	return tg
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
	for _, kw := range []core.Keyword{
		core.Islandwalk, core.Swampwalk, core.Forestwalk,
		core.Mountainwalk, core.Plainswalk,
	} {
		if perm.HasKeyword(kw) {
			score += 2
		}
	}
	if perm.HasKeyword(core.CantBeBlockedByWalls) {
		score += 1
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
		score += 1
	}
	if perm.HasKeyword(core.CanBlockAdditional) {
		score += 1
	}
	if perm.HasKeyword(core.CanBlockAny) {
		score += 1
	}
	if perm.HasKeyword(core.MustBeBlocked) {
		score += 1
	}
	if perm.HasKeyword(core.Banding) {
		score += 1
	}
	if perm.HasKeyword(core.Defender) {
		score -= 2
	}
	if perm.HasKeyword(core.DoesNotUntapKW) {
		score -= 2
	}
	if perm.HasKeyword(core.MustAttack) {
		score -= 1
	}

	return score
}

func abilityBonus(perm *mage.Permanent) int {
	score := 0
	for _, a := range perm.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		switch ab := inner.(type) {
		case *mage.ManaAbility:
			if ab.AnyColor {
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
	for _, e := range ab.Effects() {
		props := e.Properties()

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
