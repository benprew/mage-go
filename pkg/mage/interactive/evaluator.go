package interactive

import (
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

// Evaluation weight constants — tunable parameters.
const (
	LifeWeight        = 3 // multiplier for (myLife - oppLife)
	PowerWeight       = 2 // per point of creature power
	ToughnessWeight   = 1 // per point of creature toughness
	CardWeight        = 2 // per card of hand advantage
	LandWeight        = 1 // per own land
	NonCreatureCMCDiv = 2 // non-creature perm value = CMC / NonCreatureCMCDiv
)

// StateEvaluator scores a game position from playerID's perspective.
// Higher scores are better for playerID.
type StateEvaluator func(g GameReader, playerID uuid.UUID) int

// DefaultEvaluator is the standard position evaluator. It considers life,
// creature board state, non-creature permanents, hand size, and mana development.
var DefaultEvaluator StateEvaluator = defaultEvaluate

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

	// Mana development: own lands × LandWeight; untapped own lands score +1 extra.
	ownLand := mage.And(mage.IsLand, mage.ControlledBy(playerID))
	score += g.CountBattlefield(ownLand) * LandWeight
	score += g.CountBattlefield(mage.And(ownLand, mage.IsUntapped))

	return score
}

// evalCreature scores a single creature permanent from its controller's perspective.
func evalCreature(perm *mage.Permanent) int {
	score := permPower(perm)*PowerWeight + permToughness(perm)*ToughnessWeight
	if perm.Tapped {
		score = score * 2 / 3
	}
	if perm.SummonSick && !perm.HasKeyword(core.Haste) {
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
			score += 1
		case mage.TriggeredAbility:
			score += 1
		}
	}
	return score
}
