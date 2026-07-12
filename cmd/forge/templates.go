package main

// Template weights by rarity. The generator normalizes and rolls against them.
var templateWeights = map[CardTemplate]map[Rarity]float64{
	// Creatures
	Vanilla:              {Common: 0.18, Uncommon: 0.04, Rare: 0.00},
	FrenchVanilla:        {Common: 0.22, Uncommon: 0.14, Rare: 0.08},
	ETBCreature:          {Common: 0.06, Uncommon: 0.14, Rare: 0.16},
	ActivatedCreature:    {Common: 0.04, Uncommon: 0.10, Rare: 0.14},
	Lord:                 {Common: 0.00, Uncommon: 0.03, Rare: 0.10},
	DrawbackCreature:     {Common: 0.02, Uncommon: 0.04, Rare: 0.06},
	WallCreature:         {Common: 0.04, Uncommon: 0.02, Rare: 0.00},
	ManaCreature:         {Common: 0.03, Uncommon: 0.02, Rare: 0.00},
	EvasionCreature:      {Common: 0.05, Uncommon: 0.08, Rare: 0.06},
	MultiKeywordCreature: {Common: 0.00, Uncommon: 0.04, Rare: 0.12},
	CantripCreature:      {Common: 0.04, Uncommon: 0.06, Rare: 0.04},
	TapCreature:          {Common: 0.03, Uncommon: 0.06, Rare: 0.06},
	// Spells
	Burn:              {Common: 0.06, Uncommon: 0.06, Rare: 0.04},
	Removal:           {Common: 0.05, Uncommon: 0.08, Rare: 0.06},
	CombatTrick:       {Common: 0.08, Uncommon: 0.04, Rare: 0.02},
	DrawSpell:         {Common: 0.03, Uncommon: 0.05, Rare: 0.04},
	Counterspell:      {Common: 0.02, Uncommon: 0.04, Rare: 0.04},
	BounceSpell:       {Common: 0.04, Uncommon: 0.03, Rare: 0.02},
	DiscardSpell:      {Common: 0.03, Uncommon: 0.03, Rare: 0.02},
	LifeGainSpell:     {Common: 0.03, Uncommon: 0.02, Rare: 0.00},
	DisenchantSpell:   {Common: 0.02, Uncommon: 0.03, Rare: 0.02},
	MassDamage:        {Common: 0.00, Uncommon: 0.02, Rare: 0.06},
	LandDestruction:   {Common: 0.00, Uncommon: 0.02, Rare: 0.03},
	DrainSpell:        {Common: 0.02, Uncommon: 0.03, Rare: 0.03},
	Wrath:             {Common: 0.00, Uncommon: 0.00, Rare: 0.08},
	ExileRemoval:      {Common: 0.02, Uncommon: 0.04, Rare: 0.04},
	PumpSwarm:         {Common: 0.02, Uncommon: 0.03, Rare: 0.02},
	KeywordGrant:      {Common: 0.04, Uncommon: 0.03, Rare: 0.02},
	DamagePlayerSpell: {Common: 0.03, Uncommon: 0.03, Rare: 0.02},
	MassBounce:        {Common: 0.00, Uncommon: 0.01, Rare: 0.04},
	Cantrip:           {Common: 0.04, Uncommon: 0.04, Rare: 0.02},
	// Enchantments
	GlobalBuffEnch: {Common: 0.00, Uncommon: 0.02, Rare: 0.06},
	AuraBuffEnch:   {Common: 0.06, Uncommon: 0.04, Rare: 0.02},
	DebuffAura:     {Common: 0.04, Uncommon: 0.03, Rare: 0.02},
	GlobalDebuff:   {Common: 0.00, Uncommon: 0.02, Rare: 0.04},
	KeywordAura:    {Common: 0.04, Uncommon: 0.03, Rare: 0.02},
}

// PickTemplate picks a random template for a color and rarity.
func PickTemplate(color Color, rarity Rarity, rng *RNG) CardTemplate {
	candidates := EligibleTemplates(color, rarity)
	if len(candidates) == 0 {
		return Vanilla
	}
	return WeightedPick(candidates, rng)
}

// EligibleTemplates returns templates the color can use at this rarity, with weights.
func EligibleTemplates(color Color, rarity Rarity) []WeightedOption[CardTemplate] {
	pie := PieFor(color)
	var out []WeightedOption[CardTemplate]

	for _, tmpl := range AllTemplates {
		rarWeights, ok := templateWeights[tmpl]
		if !ok {
			continue
		}
		baseWeight := rarWeights[rarity]
		if baseWeight <= 0 {
			continue
		}

		if et, hasEffect := tmpl.PrimaryEffectType(); hasEffect {
			access := pie.EffectAccess[et]
			if access < 0.05 {
				continue
			}
			out = append(out, W(tmpl, baseWeight*access))
		} else {
			out = append(out, W(tmpl, baseWeight))
		}
	}

	return out
}

// Creature CMC curve weights.
var creatureCurve = map[int]map[Rarity]float64{
	1: {Common: 0.15, Uncommon: 0.10, Rare: 0.08},
	2: {Common: 0.25, Uncommon: 0.18, Rare: 0.12},
	3: {Common: 0.25, Uncommon: 0.22, Rare: 0.18},
	4: {Common: 0.18, Uncommon: 0.20, Rare: 0.18},
	5: {Common: 0.12, Uncommon: 0.16, Rare: 0.18},
	6: {Common: 0.05, Uncommon: 0.10, Rare: 0.14},
	7: {Common: 0.00, Uncommon: 0.04, Rare: 0.08},
	8: {Common: 0.00, Uncommon: 0.00, Rare: 0.04},
}

// Enchantment CMC curve weights.
var enchantmentCurve = map[int]map[Rarity]float64{
	1: {Common: 0.15, Uncommon: 0.10, Rare: 0.08},
	2: {Common: 0.30, Uncommon: 0.25, Rare: 0.18},
	3: {Common: 0.30, Uncommon: 0.28, Rare: 0.25},
	4: {Common: 0.18, Uncommon: 0.22, Rare: 0.22},
	5: {Common: 0.07, Uncommon: 0.10, Rare: 0.15},
	6: {Common: 0.00, Uncommon: 0.05, Rare: 0.12},
}

// PickCMC picks a CMC from a curve at a given rarity.
func PickCMC(curve map[int]map[Rarity]float64, rarity Rarity, rng *RNG) int {
	var options []WeightedOption[int]
	for cmc, weights := range curve {
		w := weights[rarity]
		if w > 0 {
			options = append(options, W(cmc, w))
		}
	}
	if len(options) == 0 {
		return 3
	}
	return WeightedPick(options, rng)
}

// Instant probability per spell template.
var instantProbability = map[CardTemplate]float64{
	Burn:              0.70,
	Removal:           0.40,
	CombatTrick:       0.95,
	DrawSpell:         0.20,
	Counterspell:      1.00,
	BounceSpell:       0.80,
	DiscardSpell:      0.05,
	LifeGainSpell:     0.50,
	DisenchantSpell:   0.60,
	MassDamage:        0.10,
	LandDestruction:   0.00,
	DrainSpell:        0.30,
	Wrath:             0.00,
	ExileRemoval:      0.50,
	PumpSwarm:         0.30,
	KeywordGrant:      0.85,
	DamagePlayerSpell: 0.20,
	MassBounce:        0.05,
	Cantrip:           0.60,
}

const instantPremium = 0.5

// IsInstant rolls whether a spell template produces an instant.
func IsInstant(template CardTemplate, rng *RNG) bool {
	prob := instantProbability[template]
	return rng.Float64() < prob
}
