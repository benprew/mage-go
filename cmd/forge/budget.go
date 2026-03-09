package main

import (
	"math"
	"math/rand"
)

// Budget is the economics of card design. Every mana buys a certain amount
// of "stuff" — stats, keywords, effects. Calibrated to 4th Edition era.

// VanillaStats returns total stats (P+T) a vanilla creature gets for N mana.
func VanillaStats(cmc int) float64 {
	switch cmc {
	case 0:
		return 0.5
	case 1:
		return 2.0
	case 2:
		return 4.0
	case 3:
		return 5.5
	case 4:
		return 7.0
	case 5:
		return 8.5
	case 6:
		return 10.0
	case 7:
		return 11.5
	default:
		return 10.0 + float64(cmc-6)*1.5
	}
}

// KeywordCosts maps each keyword to its stat-budget cost.
var KeywordCosts = map[Keyword]float64{
	Flying:        0.75,
	FirstStrike:   0.50,
	DoubleStrike:  1.50,
	Trample:       0.50,
	Vigilance:     0.25,
	Haste:         0.50,
	Lifelink:      0.50,
	Deathtouch:    1.00,
	Reach:         0.25,
	Defender:      -1.00,
	Menace:        0.50,
	Hexproof:      0.75,
	Indestructible: 2.00,
	Flash:         0.25,
}

// KeywordCost returns how much stat budget a keyword consumes.
func KeywordCost(kw Keyword) float64 {
	if c, ok := KeywordCosts[kw]; ok {
		return c
	}
	return 0.5
}

// ColorCommitmentBonus returns extra stat budget for multi-pip costs.
func ColorCommitmentBonus(pips int, tendency ManaPipTendency) float64 {
	switch pips {
	case 2:
		return tendency.DoublePipBonus
	case 3:
		return tendency.TriplePipBonus
	default:
		return 0.0
	}
}

// RarityModifier returns the stat bonus for rarity.
func RarityModifier(r Rarity) float64 {
	switch r {
	case Common:
		return 0.0
	case Uncommon:
		return 0.25
	case Rare:
		return 0.50
	}
	return 0.0
}

// SpellBaseCost returns the base mana cost for a spell effect.
func SpellBaseCost(effect EffectType, magnitude int) float64 {
	mag := float64(magnitude)
	switch effect {
	case DirectDamage:
		return mag / 1.5
	case DamageAllCreatures:
		return 2.0 + mag*0.75
	case DestroyCreature:
		return 2.5
	case DestroyArtifact, DestroyEnchantment:
		return 1.5
	case DestroyLand:
		return 2.5
	case Bounce:
		return 1.0
	case CounterSpell:
		return 2.0
	case DrawCards:
		return mag * 1.5
	case Discard:
		return mag * 1.0
	case GainLife:
		return mag / 3.0
	case DrainLife:
		return mag / 1.25
	case PumpTarget:
		return mag / 2.0
	case PumpSelf:
		return mag / 2.5
	case TapTarget, UntapTarget:
		return 1.5
	case AddMana:
		return 0.0
	case GlobalBuff:
		return 2.0 + mag*1.0
	case AuraBuff:
		return mag * 0.75
	case ExileCreature:
		return 2.75
	case DestroyAllCreatures:
		return 3.5
	case MassReturn:
		return 3.0
	case GrantKeyword:
		return 0.5 + mag*0.25
	case DamagePlayer:
		return mag * 1.0
	case AuraDebuff:
		return mag * 0.75
	}
	return 1.0
}

// CreatureBudget calculates total stat budget for a creature.
func CreatureBudget(cmc, coloredPips int, rarity Rarity, tendency ManaPipTendency) float64 {
	base := VanillaStats(cmc)
	commit := ColorCommitmentBonus(coloredPips, tendency)
	rar := RarityModifier(rarity)
	return base + commit + rar
}

// AllocateStats converts a stat budget + shape into power/toughness.
func AllocateStats(budget float64, shape StatShape, rng *RNG) (power, toughness int) {
	totalStats := int(math.Round(budget))
	if totalStats < 0 {
		totalStats = 0
	}

	if totalStats <= 1 {
		p := totalStats
		if p < 0 {
			p = 0
		}
		return p, 1
	}

	var ratio, jitter float64
	switch shape {
	case Balanced:
		ratio, jitter = 0.50, 0.05
	case Aggressive:
		ratio, jitter = 0.65, 0.05
	case Defensive:
		ratio, jitter = 0.30, 0.07
	case Extreme:
		if rng.Bool() {
			ratio, jitter = 0.80, 0.05
		} else {
			ratio, jitter = 0.15, 0.05
		}
	}

	actualRatio := ratio + rng.Float64Range(-jitter, jitter)
	power = int(math.Round(float64(totalStats) * actualRatio))
	toughness = totalStats - power

	if toughness < 1 {
		toughness = 1
		power = totalStats - 1
		if power < 0 {
			power = 0
		}
	}
	if power < 0 {
		power = 0
	}

	return power, toughness
}

// RNG wraps math/rand.Rand with convenience methods.
type RNG struct {
	r *rand.Rand
}

func (rng *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return rng.r.Intn(n)
}

func (rng *RNG) IntRange(min, max int) int {
	if max <= min {
		return min
	}
	return min + rng.r.Intn(max-min+1)
}

func (rng *RNG) Float64() float64 {
	return rng.r.Float64()
}

func (rng *RNG) Float64Range(min, max float64) float64 {
	return min + rng.r.Float64()*(max-min)
}

func (rng *RNG) Bool() bool {
	return rng.r.Intn(2) == 0
}

func (rng *RNG) Pick(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[rng.Intn(len(s))]
}

func (rng *RNG) PickKeyword(ks []Keyword) Keyword {
	return ks[rng.Intn(len(ks))]
}

// WeightedPick picks from weighted options.
func WeightedPick[T any](options []WeightedOption[T], rng *RNG) T {
	var total float64
	for _, o := range options {
		total += o.Weight
	}
	roll := rng.Float64() * total
	for _, o := range options {
		roll -= o.Weight
		if roll < 0 {
			return o.Value
		}
	}
	return options[len(options)-1].Value
}

type WeightedOption[T any] struct {
	Value  T
	Weight float64
}

func W[T any](v T, w float64) WeightedOption[T] {
	return WeightedOption[T]{Value: v, Weight: w}
}
