package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Generate produces a single card with given parameters.
func Generate(color Color, rarity Rarity, template CardTemplate, rng *RNG) Card {
	if template.IsCreature() {
		return generateCreature(color, rarity, template, rng)
	} else if template.IsSpell() {
		return generateSpell(color, rarity, template, rng)
	}
	return generateEnchantment(color, rarity, template, rng)
}

// GenerateAuto picks a template then generates.
func GenerateAuto(color Color, rarity Rarity, rng *RNG) Card {
	tmpl := PickTemplate(color, rarity, rng)
	return Generate(color, rarity, tmpl, rng)
}

func generateCreature(color Color, rarity Rarity, template CardTemplate, rng *RNG) Card {
	pie := PieFor(color)

	// 1. Pick CMC
	cmc := PickCMC(creatureCurve, rarity, rng)

	// 2. Determine colored pips
	pips := pickColoredPips(cmc, rarity, pie.ManaPipTendency, rng)
	manaCost := buildManaCost(cmc, pips, color)

	// 3. Calculate budget
	budget := CreatureBudget(cmc, pips, rarity, pie.ManaPipTendency)

	// 4. Pick subtype
	subtype := pickSubtype(pie.CreatureSubtypes, cmc, rng)

	// 5. Build based on template
	keywords := NewKeywordSet()
	var abilities []string
	subtypes := []string{subtype}

	switch template {
	case Vanilla:
		// Just stats

	case FrenchVanilla:
		numKW := 1
		if rarity == Rare {
			numKW = rng.IntRange(1, 2)
		}
		for i := 0; i < numKW; i++ {
			if kw, ok := pickKeyword(pie, keywords, false, rng); ok {
				keywords.Add(kw)
				budget -= KeywordCost(kw)
			}
		}

	case ETBCreature:
		etbBudget := math.Min(budget*0.4, rng.Float64Range(1.0, 2.0))
		budget -= etbBudget
		abilities = append(abilities, generateETBAbility(pie, etbBudget, rng))

	case ActivatedCreature:
		abilities = append(abilities, generateActivatedAbility(pie, color, rng))
		budget -= rng.Float64Range(0.5, 1.0)

	case Lord:
		tribal := pickTribalSubtype(pie.CreatureSubtypes, rng)
		if tribal == "" {
			tribal = subtype
		}
		subtypes = appendUnique(subtypes, tribal)
		abilities = append(abilities, fmt.Sprintf("Other %s creatures you control get +1/+1", tribal))
		budget -= 1.5

	case DrawbackCreature:
		budget += 1.5
		if rng.Bool() {
			keywords.Add(Defender)
			budget -= KeywordCost(Defender) // negative cost = adds budget
		}

	case WallCreature:
		keywords.Add(Defender)
		budget -= KeywordCost(Defender)
		if kw, ok := pickKeyword(pie, keywords, true, rng); ok {
			keywords.Add(kw)
			budget -= KeywordCost(kw)
		}

	case ManaCreature:
		abilities = append(abilities, fmt.Sprintf("{T}: Add {%s}", color.Symbol()))
		budget -= 1.5

	case EvasionCreature:
		evasive := filterKeywords([]Keyword{Flying, Menace, Haste}, pie, 0.1)
		if len(evasive) > 0 {
			kw := evasive[rng.Intn(len(evasive))]
			keywords.Add(kw)
			budget -= KeywordCost(kw)
		}
		if rarity != Common && rng.Bool() {
			if kw, ok := pickKeyword(pie, keywords, false, rng); ok {
				keywords.Add(kw)
				budget -= KeywordCost(kw)
			}
		}

	case MultiKeywordCreature:
		numKW := 2
		if rarity == Rare {
			numKW = rng.IntRange(2, 3)
		}
		for i := 0; i < numKW; i++ {
			if kw, ok := pickKeyword(pie, keywords, false, rng); ok {
				keywords.Add(kw)
				budget -= KeywordCost(kw)
			}
		}

	case CantripCreature:
		budget -= 1.5
		abilities = append(abilities, "ETB: Draw a card")

	case TapCreature:
		abilities = append(abilities, generateTapCreatureAbility(pie, color, rarity, rng))
		budget -= rng.Float64Range(0.75, 1.25)
	}

	// 6. Allocate power/toughness
	shape := pie.StatShape.Pick(rng.Float64())
	if keywords.Has(Defender) {
		shape = Defensive
	}
	power, toughness := AllocateStats(budget, shape, rng)

	// 7. Name
	name := CreatureName(color, subtype, rarity, rng)

	return Card{
		Name:      name,
		ManaCost:  manaCost,
		CardType:  CreatureType,
		Subtypes:  subtypes,
		Power:     power,
		Toughness: toughness,
		Keywords:  keywords.Slice(),
		Abilities: abilities,
	}
}

func generateSpell(color Color, rarity Rarity, template CardTemplate, rng *RNG) Card {
	pie := PieFor(color)
	isInst := IsInstant(template, rng)

	effect, baseCMC := generateSpellEffect(template, rarity, color, rng)

	finalCMC := baseCMC
	if isInst {
		finalCMC += instantPremium
	}

	cmc := max(int(math.Round(finalCMC)), 1)

	pips := pickColoredPips(cmc, rarity, pie.ManaPipTendency, rng)
	manaCost := buildManaCost(cmc, pips, color)
	name := SpellName(color, template, rarity, rng)

	ct := SorceryType
	if isInst {
		ct = InstantType
	}

	return Card{
		Name:      name,
		ManaCost:  manaCost,
		CardType:  ct,
		Abilities: []string{effect},
	}
}

func generateEnchantment(color Color, rarity Rarity, template CardTemplate, rng *RNG) Card {
	pie := PieFor(color)
	cmc := PickCMC(enchantmentCurve, rarity, rng)
	pips := pickColoredPips(cmc, rarity, pie.ManaPipTendency, rng)
	manaCost := buildManaCost(cmc, pips, color)
	name := EnchantmentName(color, template, rarity, rng)

	var subtypes []string
	var abilities []string

	switch template {
	case GlobalBuffEnch:
		bonus := max(1, (cmc-1)/2+1)
		grantKW := rng.Bool() && cmc >= 3
		if grantKW {
			kw, _ := pickKeyword(pie, NewKeywordSet(), false, rng)
			if bonus > 1 {
				abilities = append(abilities, fmt.Sprintf("Creatures you control get +%d/+%d and have %s", bonus-1, bonus-1, kw))
			} else {
				abilities = append(abilities, fmt.Sprintf("Creatures you control have %s", kw))
			}
		} else {
			abilities = append(abilities, fmt.Sprintf("Creatures you control get +%d/+%d", bonus, bonus))
		}

	case AuraBuffEnch:
		subtypes = []string{"Aura"}
		budget := max(1, cmc)
		grantKW := rng.Bool() && cmc >= 2
		var kwStr string
		if grantKW {
			if kw, ok := pickKeyword(pie, NewKeywordSet(), false, rng); ok {
				kwStr = kw.String()
				budget = max(0, budget-1)
			}
		}
		shape := pie.StatShape.Pick(rng.Float64())
		var pRatio float64
		switch shape {
		case Balanced:
			pRatio = 0.5
		case Aggressive:
			pRatio = 0.7
		case Defensive:
			pRatio = 0.3
		case Extreme:
			if rng.Bool() {
				pRatio = 0.9
			} else {
				pRatio = 0.1
			}
		}
		pBonus := max(int(math.Round(float64(budget)*pRatio)), 0)
		tBonus := max(budget-pBonus, 0)
		desc := fmt.Sprintf("Enchanted creature gets +%d/+%d", pBonus, tBonus)
		if kwStr != "" {
			desc += " and has " + kwStr
		}
		abilities = append(abilities, desc)

	case DebuffAura:
		subtypes = []string{"Aura"}
		magnitude := max(1, cmc)
		pDebuff := magnitude/2 + 1
		tDebuff := magnitude - pDebuff + 1
		abilities = append(abilities, fmt.Sprintf("Enchanted creature gets -%d/-%d", pDebuff, tDebuff))

	case GlobalDebuff:
		magnitude := max(1, (cmc-1)/2+1)
		abilities = append(abilities, fmt.Sprintf("Creatures opponents control get -%d/-%d", magnitude, magnitude))

	case KeywordAura:
		subtypes = []string{"Aura"}
		kw, _ := pickKeyword(pie, NewKeywordSet(), false, rng)
		abilities = append(abilities, fmt.Sprintf("Enchanted creature has %s", kw))

	default:
		abilities = append(abilities, "Creatures you control get +1/+1")
	}

	return Card{
		Name:      name,
		ManaCost:  manaCost,
		CardType:  EnchantmentType,
		Subtypes:  subtypes,
		Abilities: abilities,
	}
}

// ── Helpers ──

func pickColoredPips(cmc int, rarity Rarity, tendency ManaPipTendency, rng *RNG) int {
	var singlePipProb float64
	switch rarity {
	case Common:
		singlePipProb = tendency.CommonSinglePip
	case Uncommon:
		singlePipProb = tendency.UncommonSinglePip
	case Rare:
		singlePipProb = tendency.RareSinglePip
	}

	if cmc <= 1 {
		return 1
	}

	roll := rng.Float64()
	if roll < singlePipProb {
		return 1
	} else if roll < singlePipProb+(1.0-singlePipProb)*0.7 {
		return 2
	}
	return min(3, cmc)
}

func buildManaCost(cmc, pips int, color Color) ManaCost {
	generic := max(cmc-pips, 0)
	actualPips := min(pips, cmc)

	mc := ManaCost{Generic: generic}
	switch color {
	case White:
		mc.White = actualPips
	case Blue:
		mc.Blue = actualPips
	case Black:
		mc.Black = actualPips
	case Red:
		mc.Red = actualPips
	case Green:
		mc.Green = actualPips
	case Colorless:
		mc.Generic = cmc
	}
	return mc
}

func pickSubtype(pools []SubtypePool, cmc int, rng *RNG) string {
	var eligible []WeightedOption[string]
	for _, p := range pools {
		if cmc >= p.MinCMC && cmc <= p.MaxCMC {
			eligible = append(eligible, W(p.Name, p.Weight))
		}
	}
	if len(eligible) == 0 {
		// Fallback
		for _, p := range pools {
			eligible = append(eligible, W(p.Name, p.Weight))
		}
	}
	if len(eligible) == 0 {
		return "Creature"
	}
	return WeightedPick(eligible, rng)
}

func pickTribalSubtype(pools []SubtypePool, rng *RNG) string {
	var tribal []WeightedOption[string]
	for _, p := range pools {
		if p.TribalWorthy {
			tribal = append(tribal, W(p.Name, p.Weight))
		}
	}
	if len(tribal) == 0 {
		return ""
	}
	return WeightedPick(tribal, rng)
}

func pickKeyword(pie ColorPie, excluding KeywordSet, preferDefensive bool, rng *RNG) (Keyword, bool) {
	defensiveKWs := KeywordSet{Reach: true, Flying: true, Hexproof: true, Indestructible: true}

	var candidates []WeightedOption[Keyword]
	for kw, affinity := range pie.KeywordAffinities {
		if affinity <= 0.05 || excluding.Has(kw) {
			continue
		}
		w := affinity
		if preferDefensive && defensiveKWs.Has(kw) {
			w *= 2.0
		}
		candidates = append(candidates, W(kw, w))
	}

	if len(candidates) == 0 {
		return 0, false
	}

	// Sort for deterministic ordering (map iteration is random)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Value < candidates[j].Value
	})

	return WeightedPick(candidates, rng), true
}

func filterKeywords(keywords []Keyword, pie ColorPie, minAffinity float64) []Keyword {
	var out []Keyword
	for _, kw := range keywords {
		if pie.KeywordAffinities[kw] > minAffinity {
			out = append(out, kw)
		}
	}
	return out
}

func generateETBAbility(pie ColorPie, budget float64, rng *RNG) string {
	var options []WeightedOption[string]
	tryAdd := func(name string, et EffectType) {
		if w := pie.EffectAccess[et]; w > 0.05 {
			options = append(options, W(name, w))
		}
	}
	tryAdd("damage", DirectDamage)
	tryAdd("life", GainLife)
	tryAdd("destroy_artifact", DestroyArtifact)
	tryAdd("bounce", Bounce)
	tryAdd("draw", DrawCards)
	tryAdd("discard", Discard)
	tryAdd("drain", DrainLife)

	if len(options) == 0 {
		return "ETB: Gain 1 life"
	}

	magnitude := max(int(math.Round(budget)), 1)

	picked := WeightedPick(options, rng)
	switch picked {
	case "damage":
		return fmt.Sprintf("ETB: Deal %d damage to any target", magnitude)
	case "life":
		return fmt.Sprintf("ETB: Gain %d life", magnitude*2)
	case "bounce":
		return "ETB: Return target creature to its owner's hand"
	case "draw":
		count := max(magnitude/2, 1)
		return fmt.Sprintf("ETB: Draw %d card(s)", count)
	case "discard":
		count := max(magnitude/2, 1)
		return fmt.Sprintf("ETB: Target player discards %d card(s)", count)
	case "drain":
		return fmt.Sprintf("ETB: Each opponent loses %d life and you gain %d life", magnitude, magnitude)
	case "destroy_artifact":
		return "ETB: Destroy target artifact"
	}
	return fmt.Sprintf("ETB: Gain %d life", magnitude)
}

func generateActivatedAbility(pie ColorPie, color Color, rng *RNG) string {
	pumpAccess := pie.EffectAccess[PumpSelf]
	if pumpAccess > 0.3 && rng.Bool() {
		if color == Black || color == Green {
			return fmt.Sprintf("{%s}: +1/+1 until end of turn", color.Symbol())
		}
		return fmt.Sprintf("{%s}: +1/+0 until end of turn", color.Symbol())
	}

	tapAccess := pie.EffectAccess[TapTarget]
	if tapAccess > 0.3 {
		return "{T}: Tap target creature"
	}

	manaAccess := pie.EffectAccess[AddMana]
	if manaAccess > 0.3 {
		return fmt.Sprintf("{T}: Add {%s}", color.Symbol())
	}

	return "{2}: +1/+1 until end of turn"
}

func generateTapCreatureAbility(pie ColorPie, color Color, rarity Rarity, rng *RNG) string {
	pingAccess := pie.EffectAccess[DirectDamage]
	tapAccess := pie.EffectAccess[TapTarget]
	drawAccess := pie.EffectAccess[DrawCards]

	var options []WeightedOption[string]
	if pingAccess > 0.1 {
		options = append(options, W("ping", pingAccess))
	}
	if tapAccess > 0.1 {
		options = append(options, W("tap", tapAccess))
	}
	if rarity == Rare && drawAccess > 0.1 {
		options = append(options, W("draw", drawAccess*0.5))
	}

	choice := "ping"
	if len(options) > 0 {
		choice = WeightedPick(options, rng)
	}

	switch choice {
	case "ping":
		return "{T}: Deal 1 damage to any target"
	case "tap":
		return "{T}: Tap target creature"
	case "draw":
		return "{T}: Draw a card"
	}
	return "{T}: Deal 1 damage to any target"
}

func generateSpellEffect(template CardTemplate, rarity Rarity, color Color, rng *RNG) (string, float64) {
	switch template {
	case Burn:
		mag := pickMagnitude(2, rarity, rng)
		cost := SpellBaseCost(DirectDamage, mag)
		return fmt.Sprintf("Deal %d damage to any target", mag), cost

	case Removal:
		cost := SpellBaseCost(DestroyCreature, 1)
		return "Destroy target creature", cost

	case CombatTrick:
		mag := pickMagnitude(2, rarity, rng)
		cost := SpellBaseCost(PumpTarget, mag)
		var p, t int
		switch color {
		case Red:
			p, t = mag, 0
		case White:
			p, t = max(0, mag-1), mag
		default:
			p, t = mag, mag
		}
		return fmt.Sprintf("Target creature gets +%d/+%d until end of turn", p, t), cost

	case DrawSpell:
		count := rng.IntRange(1, 2)
		if rarity == Rare {
			count = rng.IntRange(2, 3)
		}
		cost := SpellBaseCost(DrawCards, count)
		return fmt.Sprintf("Draw %d cards", count), cost

	case Counterspell:
		cost := SpellBaseCost(CounterSpell, 1)
		return "Counter target spell", cost

	case BounceSpell:
		cost := SpellBaseCost(Bounce, 1)
		return "Return target creature to its owner's hand", cost

	case DiscardSpell:
		count := rng.IntRange(1, 2)
		if rarity == Rare {
			count = 2
		}
		cost := SpellBaseCost(Discard, count)
		return fmt.Sprintf("Target player discards %d card(s)", count), cost

	case LifeGainSpell:
		amount := pickMagnitude(3, rarity, rng) * 2
		cost := SpellBaseCost(GainLife, amount)
		return fmt.Sprintf("You gain %d life", amount), cost

	case DisenchantSpell:
		cost := SpellBaseCost(DestroyEnchantment, 1)
		return "Destroy target artifact or enchantment", cost

	case MassDamage:
		mag := pickMagnitude(2, rarity, rng)
		cost := SpellBaseCost(DamageAllCreatures, mag)
		return fmt.Sprintf("Deal %d damage to each creature", mag), cost

	case LandDestruction:
		cost := SpellBaseCost(DestroyLand, 1)
		return "Destroy target land", cost

	case DrainSpell:
		mag := pickMagnitude(2, rarity, rng)
		cost := SpellBaseCost(DrainLife, mag)
		return fmt.Sprintf("Deal %d damage to any target. You gain %d life", mag, mag), cost

	case Wrath:
		cost := SpellBaseCost(DestroyAllCreatures, 1)
		return "Destroy all creatures", cost

	case ExileRemoval:
		cost := SpellBaseCost(ExileCreature, 1)
		return "Exile target creature", cost

	case PumpSwarm:
		mag := pickMagnitude(2, rarity, rng)
		cost := SpellBaseCost(GlobalBuff, mag)
		var p, t int
		switch color {
		case Red:
			p, t = mag, 0
		case White:
			p, t = 1, mag
		default:
			p, t = mag, mag
		}
		return fmt.Sprintf("Creatures you control get +%d/+%d until end of turn", p, t), cost

	case KeywordGrant:
		pie := PieFor(color)
		var kwOptions []WeightedOption[Keyword]
		for kw, aff := range pie.KeywordAffinities {
			if aff > 0.1 {
				kwOptions = append(kwOptions, W(kw, aff))
			}
		}
		kw := Flying
		if len(kwOptions) > 0 {
			sort.Slice(kwOptions, func(i, j int) bool {
				return kwOptions[i].Value < kwOptions[j].Value
			})
			kw = WeightedPick(kwOptions, rng)
		}
		cost := SpellBaseCost(GrantKeyword, 1)
		return fmt.Sprintf("Target creature gains %s until end of turn", kw), cost

	case DamagePlayerSpell:
		mag := pickMagnitude(4, rarity, rng)
		cost := SpellBaseCost(DamagePlayer, mag)
		return fmt.Sprintf("Deal %d damage to target player", mag), cost

	case MassBounce:
		cost := SpellBaseCost(MassReturn, 1)
		return "Return all creatures to their owners' hands", cost

	case Cantrip:
		pie := PieFor(color)
		smallEffect, smallCost := generateCantripEffect(pie, rng)
		drawCost := SpellBaseCost(DrawCards, 1)
		return smallEffect + ". Draw a card", smallCost + drawCost
	}

	return "Gain 1 life", 1.0
}

func generateCantripEffect(pie ColorPie, rng *RNG) (string, float64) {
	var options []WeightedOption[string]
	if w := pie.EffectAccess[DirectDamage]; w > 0.1 {
		options = append(options, W("damage", w))
	}
	if w := pie.EffectAccess[GainLife]; w > 0.1 {
		options = append(options, W("life", w))
	}
	if w := pie.EffectAccess[PumpTarget]; w > 0.1 {
		options = append(options, W("pump", w))
	}
	if w := pie.EffectAccess[Bounce]; w > 0.1 {
		options = append(options, W("bounce", w))
	}
	options = append(options, W("tiny", 0.3))

	choice := WeightedPick(options, rng)
	switch choice {
	case "damage":
		return "Deal 1 damage to any target", 0.5
	case "life":
		return "Gain 2 life", 0.3
	case "pump":
		return "Target creature gets +1/+1 until end of turn", 0.3
	case "bounce":
		return "Return target creature to its owner's hand", 0.8
	}
	return "Gain 1 life", 0.2
}

func pickMagnitude(base int, rarity Rarity, rng *RNG) int {
	switch rarity {
	case Common:
		return base + rng.IntRange(-1, 0)
	case Uncommon:
		return base + rng.IntRange(0, 1)
	case Rare:
		return base + rng.IntRange(1, 2)
	}
	return base
}

func appendUnique(ss []string, s string) []string {
	for _, existing := range ss {
		if strings.EqualFold(existing, s) {
			return ss
		}
	}
	return append(ss, s)
}
