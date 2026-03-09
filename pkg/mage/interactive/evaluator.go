package interactive

import (
	"math"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

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

// DefaultEvaluator is the standard position evaluator. It considers life,
// creature board state, non-creature permanents, hand size, mana development,
// untapped mana sources (tempo), lethal-on-board, and board diversity.
// It uses the default hardcoded weight constants.
var DefaultEvaluator StateEvaluator = defaultEvaluate

// WeightedEvaluator returns a StateEvaluator that uses the given
// WeightedPersonality's evaluation weights instead of the default constants.
func WeightedEvaluator(w WeightedPersonality) StateEvaluator {
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

	// Creature board: own creatures add, opponent's subtract.
	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		v := evalCreature(perm)
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
			score -= v
		}
	}

	// Non-creature, non-land permanents (enchantments, artifacts, etc.):
	// value = CMC / NonCreatureCMCDiv.
	nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
	for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
		v := perm.Card.ManaCost().CMC() / NonCreatureCMCDiv
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
			score -= v
		}
	}

	// Card advantage.
	score += (len(me.Hand()) - len(opp.Hand())) * CardWeight

	// Mana development: own lands × LandWeight.
	ownLand := mage.And(mage.IsLand, mage.ControlledBy(playerID))
	score += g.CountBattlefield(ownLand) * LandWeight

	// Tempo — count untapped mana sources (lands + mana creatures + mana artifacts).
	untappedMana := countUntappedManaSources(g, playerID)
	score += untappedMana * 2

	// Lethal-on-board detection.
	lethal := calculateLethalOnBoard(g, playerID)
	score += lethal

	return score
}

// NewWeightedEvaluator creates a StateEvaluator that uses a WeightedPersonality
// to scale each evaluation dimension with role-based permanent classification,
// hand quality scoring, board diversity, tempo, and lethal detection.
func NewWeightedEvaluator(wp WeightedPersonality) StateEvaluator {
	return func(g GameReader, playerID uuid.UUID) int {
		me := g.GetPlayer(playerID)
		opp := g.GetOpponent(playerID)
		if me == nil || opp == nil {
			return 0
		}
		oppID := opp.PlayerID()

		score := 0.0

		// Life difference, scaled by LifeWeight.
		score += float64(me.Life()-opp.Life()) * wp.LifeWeight

		// Board: creatures evaluated and weighted by role.
		allPerms := g.FilterBattlefield(mage.IsCreature)
		for _, perm := range allPerms {
			v := float64(evalCreature(perm))
			role := ClassifyPermanent(perm)
			roleWeight := roleWeightForPersonality(role, wp)
			if perm.Controller == playerID {
				score += v * roleWeight
			} else if perm.Controller == oppID {
				score -= v * roleWeight
			}
		}

		// Non-creature, non-land permanents weighted by role.
		nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
		for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
			v := float64(perm.Card.ManaCost().CMC()) / float64(NonCreatureCMCDiv)
			role := ClassifyPermanent(perm)
			roleWeight := roleWeightForPersonality(role, wp)
			if perm.Controller == playerID {
				score += v * roleWeight
			} else if perm.Controller == oppID {
				score -= v * roleWeight
			}
		}

		// Hand quality, not just count.
		myHandQuality := handQuality(me)
		oppHandQuality := handQuality(opp)
		score += float64(myHandQuality-oppHandQuality) * wp.CardWeight

		// Mana development.
		ownLandCount := g.CountBattlefield(mage.And(mage.IsLand, mage.ControlledBy(playerID)))
		oppLandCount := g.CountBattlefield(mage.And(mage.IsLand, mage.ControlledBy(oppID)))
		score += float64(ownLandCount-oppLandCount) * wp.ManaWeight

		// Tempo — untapped mana sources weighted by TempoWeight.
		myUntapped := countUntappedManaSources(g, playerID)
		oppUntapped := countUntappedManaSources(g, oppID)
		score += float64(myUntapped-oppUntapped) * wp.TempoWeight

		// Board diversity bonus.
		myRoles := countRoles(g.FilterBattlefield(mage.ControlledBy(playerID)), func(_ *mage.Permanent) bool { return true })
		oppRoles := countRoles(g.FilterBattlefield(mage.ControlledBy(oppID)), func(_ *mage.Permanent) bool { return true })
		score += float64(boardDiversity(myRoles)-boardDiversity(oppRoles)) * wp.BoardWeight * 0.5

		// Lethal-on-board detection.
		lethal := calculateLethalOnBoard(g, playerID)
		score += float64(lethal)

		return int(math.Round(score))
	}
}

// roleWeightForPersonality returns a multiplier for a permanent role given a personality.
func roleWeightForPersonality(role PermanentRole, wp WeightedPersonality) float64 {
	switch role {
	case RoleThreat:
		return wp.BoardWeight / 3.0 // normalized so default midrange ~1.0
	case RoleUtility:
		return wp.CardWeight / 2.0
	case RoleEngine:
		return wp.CardWeight / 2.0
	case RoleMana:
		return wp.ManaWeight
	case RoleDefense:
		return wp.BoardWeight / 3.0
	default:
		return 1.0
	}
}

// handQuality scores a player's hand based on the quality of cards, not just count.
// A hand of spells is worth more than a hand of only lands.
func handQuality(p mage.Player) int {
	hand := p.Hand()
	if len(hand) == 0 {
		return 0
	}
	score := 0
	for _, card := range hand {
		if card.HasType(core.TypeLand) {
			score += 1 // lands are worth less than spells
		} else {
			// Non-land cards get a base value of 2 (like CardWeight).
			cmc := card.ManaCost().CMC()
			if cmc > 0 {
				score += 2
			} else {
				score += 1
			}
		}
	}
	return score
}

// countUntappedManaSources counts all untapped mana sources controlled by playerID:
// untapped lands, untapped creatures with mana abilities, and untapped artifacts with mana abilities.
func countUntappedManaSources(g GameReader, playerID uuid.UUID) int {
	count := 0
	own := mage.And(mage.ControlledBy(playerID), mage.IsUntapped)
	for _, perm := range g.FilterBattlefield(own) {
		if perm.HasType(core.TypeLand) {
			count++
			continue
		}
		// Check for mana abilities on non-land permanents.
		if hasManaAbility(perm) {
			count++
		}
	}
	return count
}

// calculateLethalOnBoard returns a score adjustment based on lethal detection.
// +LethalBonus if we have lethal on board, -LethalBonus if opponent does.
func calculateLethalOnBoard(g GameReader, playerID uuid.UUID) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := 0

	// Check if we have lethal: sum power of our creatures that can attack.
	myDamage := 0
	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		if perm.Controller == playerID && canPotentiallyAttack(perm) {
			myDamage += permPower(perm)
		}
	}
	if myDamage >= opp.Life() && myDamage > 0 {
		score += LethalBonus
	}

	// Check if opponent has lethal.
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

// canPotentiallyAttack returns true if a creature could potentially attack
// (not tapped, not summoning sick unless has haste, not defender).
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

// weightedEvaluate scores a game position using personality-specific weights.
// The score components are:
//   - Life difference × LifeWeight
//   - Creature board score × BoardWeight (scaled from default PowerWeight/ToughnessWeight)
//   - Non-creature permanents × BoardWeight / 2
//   - Card advantage × CardWeight
//   - Mana development (lands) × ManaWeight
//   - Untapped land bonus × TempoWeight
func weightedEvaluate(g GameReader, playerID uuid.UUID, w WeightedPersonality) int {
	me := g.GetPlayer(playerID)
	opp := g.GetOpponent(playerID)
	if me == nil || opp == nil {
		return 0
	}
	oppID := opp.PlayerID()

	score := 0.0

	// Life component.
	score += float64(me.Life()-opp.Life()) * w.LifeWeight

	// Creature board: own creatures add, opponent's subtract.
	// Scale creature values by BoardWeight relative to the default (2.0).
	boardScale := w.BoardWeight / 2.0
	for _, perm := range g.FilterBattlefield(mage.IsCreature) {
		v := float64(evalCreature(perm)) * boardScale
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
			score -= v
		}
	}

	// Non-creature, non-land permanents.
	nonCreatureNonLand := mage.And(mage.Not(mage.IsCreature), mage.Not(mage.IsLand))
	for _, perm := range g.FilterBattlefield(nonCreatureNonLand) {
		v := float64(perm.Card.ManaCost().CMC()) / 2.0 * boardScale
		if perm.Controller == playerID {
			score += v
		} else if perm.Controller == oppID {
			score -= v
		}
	}

	// Card advantage.
	score += float64(len(me.Hand())-len(opp.Hand())) * w.CardWeight

	// Mana development.
	ownLand := mage.And(mage.IsLand, mage.ControlledBy(playerID))
	score += float64(g.CountBattlefield(ownLand)) * w.ManaWeight

	// Untapped land bonus (tempo).
	score += float64(g.CountBattlefield(mage.And(ownLand, mage.IsUntapped))) * w.TempoWeight

	return int(score)
}

// evalCreature scores a single creature permanent from its controller's perspective.
func evalCreature(perm *mage.Permanent) int {
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

// permPower returns the creature's power from base stats and counters.
// Misses continuous-effect bonuses (e.g. Crusade) — acceptable for v1.
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

// permToughness returns the creature's toughness from base stats and counters.
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

// keywordBonus returns a score contribution for all keyword abilities on perm.
func keywordBonus(perm *mage.Permanent) int {
	score := 0

	// Unblockable.
	if perm.HasKeyword(core.UnblockableKW) {
		score += 5
	}

	// Evasion.
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

	// Combat.
	if perm.HasKeyword(core.DoubleStrike) {
		score += 4
	}
	if perm.HasKeyword(core.FirstStrike) {
		score += 2
	}
	if perm.HasKeyword(core.Haste) {
		score += 2
	}

	// Durability.
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

	// Damage.
	if perm.HasKeyword(core.Deathtouch) {
		score += 3
	}
	if perm.HasKeyword(core.Lifelink) {
		score += 2
	}

	// Versatility.
	if perm.HasKeyword(core.Vigilance) {
		score += 2
	}

	// Defensive / utility.
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

	// Drawbacks.
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

// abilityBonus scores non-keyword runtime abilities on a creature.
// Uses abilityQuality for activated abilities and triggeredAbilityQuality
// for triggered abilities, giving deeper scoring based on effect type and
// cost efficiency.
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
			score += abilityQuality(ab)
		case mage.TriggeredAbility:
			score += triggeredAbilityQuality(ab)
		}
	}
	return score
}

// abilityQuality scores an activated ability by its effect type and cost efficiency.
// More impactful abilities (tap-to-draw, tap-to-deal-damage) score higher than
// expensive pump abilities.
func abilityQuality(ab mage.ActivatedAbility) int {
	// Analyze costs: check if it requires tap, what mana cost, etc.
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

	// Analyze effects for quality.
	bestScore := 0
	for _, e := range ab.Effects() {
		props := e.Properties()

		// Tap-to-draw: very valuable.
		if props.DrawCount > 0 {
			s := 5
			if manaCostTotal > 2 {
				s = 3 // expensive draw is less good
			}
			if s > bestScore {
				bestScore = s
			}
		}

		// Tap-to-deal-damage: very valuable.
		if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
			s := 4
			if !hasTapCost && manaCostTotal >= 3 {
				s = 2 // expensive damage is less good
			}
			if s > bestScore {
				bestScore = s
			}
		}

		// Destruction / removal effects.
		if props.Outcome == mage.OutcomeDetriment && props.DamageValue == nil && props.DrawCount == 0 {
			s := 3
			if manaCostTotal >= 3 {
				s = 2
			}
			if s > bestScore {
				bestScore = s
			}
		}

		// Beneficial effects (buff, prevent, untap).
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

	// If no recognizable effect, give a small bonus based on cost.
	if bestScore == 0 {
		if hasTapCost && manaCostTotal == 0 {
			return 2 // free tap ability is decent
		}
		if manaCostTotal <= 2 {
			return 1 // cheap ability
		}
		return 1 // expensive but still an ability
	}

	return bestScore
}

// triggeredAbilityQuality scores a triggered ability based on its effects.
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
		return 1 // any triggered ability has some value
	}
	return bestScore
}
