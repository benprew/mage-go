package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// SetConfig configures set generation.
type SetConfig struct {
	CardCount          int
	IncludeArtifacts   bool
	ArtifactPercentage float64
	SetName            string
	SetCode            string
}

// GeneratedSet is a complete set of generated cards.
type GeneratedSet struct {
	Config     SetConfig
	Cards      []GeneratedCard
	Statistics SetStatistics
}

// GeneratedCard is a card with its generation metadata.
type GeneratedCard struct {
	Card            Card
	Color           Color
	Rarity          Rarity
	Template        CardTemplate
	CollectorNumber int
}

// SetStatistics about the generated set.
type SetStatistics struct {
	TotalCards         int
	ByColor            map[Color]int
	ByRarity           map[Rarity]int
	ByType             map[string]int
	AverageCMCByColor  map[Color]float64
	CreaturePercentage float64
	RemovalCount       int
}

// GenerateSet generates a complete card set.
func GenerateSet(config SetConfig, seed int64) GeneratedSet {
	rng := &RNG{r: rand.New(rand.NewSource(seed))}
	return generateSet(config, rng)
}

func generateSet(config SetConfig, rng *RNG) GeneratedSet {
	skeleton := buildSkeleton(config, rng)
	cards := make([]GeneratedCard, 0, len(skeleton))
	collectorNumber := 1

	for _, slot := range skeleton {
		var card Card
		if slot.template >= 0 {
			card = Generate(slot.color, slot.rarity, slot.template, rng)
		} else {
			card = GenerateAuto(slot.color, slot.rarity, rng)
		}

		tmpl := slot.template
		if tmpl < 0 {
			tmpl = inferTemplate(card)
		}

		cards = append(cards, GeneratedCard{
			Card:            card,
			Color:           slot.color,
			Rarity:          slot.rarity,
			Template:        tmpl,
			CollectorNumber: collectorNumber,
		})
		collectorNumber++
	}

	// Validation passes
	cards = ensureRemovalDensity(cards, rng)
	cards = ensureCreatureCurve(cards, rng)
	cards = deduplicateNames(cards, rng)

	// Renumber
	for i := range cards {
		cards[i].CollectorNumber = i + 1
	}

	stats := computeStatistics(cards)

	return GeneratedSet{
		Config:     config,
		Cards:      cards,
		Statistics: stats,
	}
}

type skeletonSlot struct {
	color    Color
	rarity   Rarity
	template CardTemplate // -1 = auto
}

func buildSkeleton(config SetConfig, rng *RNG) []skeletonSlot {
	var slots []skeletonSlot

	artifactCount := 0
	if config.IncludeArtifacts {
		artifactCount = int(float64(config.CardCount) * config.ArtifactPercentage)
	}
	coloredCount := config.CardCount - artifactCount
	perColor := coloredCount / 5

	for _, color := range AllColors {
		commons := int(float64(perColor) * 0.55)
		uncommons := int(float64(perColor) * 0.30)
		rares := perColor - commons - uncommons

		for i := 0; i < commons; i++ {
			slots = append(slots, skeletonSlot{color: color, rarity: Common, template: -1})
		}
		for i := 0; i < uncommons; i++ {
			slots = append(slots, skeletonSlot{color: color, rarity: Uncommon, template: -1})
		}
		for i := 0; i < rares; i++ {
			slots = append(slots, skeletonSlot{color: color, rarity: Rare, template: -1})
		}
	}

	artifactRarities := []Rarity{Common, Common, Uncommon, Uncommon, Rare}
	for i := 0; i < artifactCount; i++ {
		r := artifactRarities[rng.Intn(len(artifactRarities))]
		slots = append(slots, skeletonSlot{color: Colorless, rarity: r, template: -1})
	}

	return slots
}

// Validation: ensure removal density.
func ensureRemovalDensity(cards []GeneratedCard, rng *RNG) []GeneratedCard {
	result := make([]GeneratedCard, len(cards))
	copy(result, cards)

	removalTemplates := map[CardTemplate]bool{Burn: true, Removal: true, MassDamage: true}

	for _, color := range AllColors {
		pie := PieFor(color)
		hasRemoval := pie.EffectAccess[DestroyCreature] > 0.3 || pie.EffectAccess[DirectDamage] > 0.3
		if !hasRemoval {
			continue
		}

		var colorIndices []int
		removalCount := 0
		var replaceableIndices []int

		for i, gc := range result {
			if gc.Color != color {
				continue
			}
			colorIndices = append(colorIndices, i)
			if removalTemplates[gc.Template] {
				removalCount++
			}
			if gc.Template.IsSpell() && !removalTemplates[gc.Template] {
				replaceableIndices = append(replaceableIndices, i)
			}
		}

		if removalCount < 2 {
			needed := 2 - removalCount
			for j := 0; j < needed && j < len(replaceableIndices); j++ {
				idx := replaceableIndices[j]
				old := result[idx]
				tmpl := Removal
				if pie.EffectAccess[DirectDamage] > 0.5 {
					tmpl = Burn
				}
				newCard := Generate(color, old.Rarity, tmpl, rng)
				result[idx] = GeneratedCard{
					Card:            newCard,
					Color:           color,
					Rarity:          old.Rarity,
					Template:        tmpl,
					CollectorNumber: old.CollectorNumber,
				}
			}
		}
	}

	return result
}

// Validation: ensure creature curve.
func ensureCreatureCurve(cards []GeneratedCard, rng *RNG) []GeneratedCard {
	result := make([]GeneratedCard, len(cards))
	copy(result, cards)

	for _, color := range AllColors {
		// Check for at least one 2-drop creature at common
		hasTwoDrop := false
		replaceIdx := -1

		for i, gc := range result {
			if gc.Color != color || gc.Rarity != Common {
				continue
			}
			if gc.Template.IsCreature() && gc.Card.CMC() == 2 {
				hasTwoDrop = true
				break
			}
			if !gc.Template.IsCreature() && replaceIdx < 0 {
				replaceIdx = i
			}
		}

		if !hasTwoDrop && replaceIdx >= 0 {
			old := result[replaceIdx]
			var candidate Card
			for attempts := 0; attempts < 10; attempts++ {
				candidate = Generate(color, Common, Vanilla, rng)
				if candidate.CMC() == 2 {
					break
				}
			}
			result[replaceIdx] = GeneratedCard{
				Card:            candidate,
				Color:           color,
				Rarity:          Common,
				Template:        Vanilla,
				CollectorNumber: old.CollectorNumber,
			}
		}
	}

	return result
}

// Validation: deduplicate names.
func deduplicateNames(cards []GeneratedCard, rng *RNG) []GeneratedCard {
	result := make([]GeneratedCard, len(cards))
	copy(result, cards)
	seen := make(map[string]bool)

	for i := range result {
		name := result[i].Card.Name
		attempts := 0
		for seen[name] && attempts < 20 {
			gc := result[i]
			var newCard Card
			if gc.Template >= 0 {
				newCard = Generate(gc.Color, gc.Rarity, gc.Template, rng)
			} else {
				newCard = GenerateAuto(gc.Color, gc.Rarity, rng)
			}
			result[i] = GeneratedCard{
				Card:            newCard,
				Color:           gc.Color,
				Rarity:          gc.Rarity,
				Template:        gc.Template,
				CollectorNumber: gc.CollectorNumber,
			}
			name = newCard.Name
			attempts++
		}
		seen[name] = true
	}

	return result
}

func computeStatistics(cards []GeneratedCard) SetStatistics {
	byColor := make(map[Color]int)
	byRarity := make(map[Rarity]int)
	byType := make(map[string]int)
	cmcSums := make(map[Color]float64)
	cmcCounts := make(map[Color]int)
	creatureCount := 0
	removalCount := 0

	removalTemplates := map[CardTemplate]bool{Burn: true, Removal: true, MassDamage: true}

	for _, gc := range cards {
		byColor[gc.Color]++
		byRarity[gc.Rarity]++
		byType[gc.Card.CardType.String()]++
		cmcSums[gc.Color] += float64(gc.Card.CMC())
		cmcCounts[gc.Color]++
		if gc.Card.IsCreature() {
			creatureCount++
		}
		if removalTemplates[gc.Template] {
			removalCount++
		}
	}

	avgCMC := make(map[Color]float64)
	for c, sum := range cmcSums {
		if cnt := cmcCounts[c]; cnt > 0 {
			avgCMC[c] = sum / float64(cnt)
		}
	}

	creaturePct := 0.0
	if len(cards) > 0 {
		creaturePct = float64(creatureCount) / float64(len(cards))
	}

	return SetStatistics{
		TotalCards:         len(cards),
		ByColor:            byColor,
		ByRarity:           byRarity,
		ByType:             byType,
		AverageCMCByColor:  avgCMC,
		CreaturePercentage: creaturePct,
		RemovalCount:       removalCount,
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func inferTemplate(card Card) CardTemplate {
	if card.IsCreature() {
		if len(card.Keywords) > 0 {
			return FrenchVanilla
		}
		if len(card.Abilities) > 0 {
			return ETBCreature
		}
		return Vanilla
	}
	if card.IsInstant() || card.IsSorcery() {
		return Burn
	}
	if card.IsEnchantment() {
		return GlobalBuffEnch
	}
	return Vanilla
}

// ── Pretty Printing ──

func PrintSummary(set GeneratedSet) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  %s [%s]\n", set.Config.SetName, set.Config.SetCode)
	fmt.Printf("  %d cards + 5 basic lands\n", set.Statistics.TotalCards)
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	fmt.Println("── Distribution ──")
	for _, color := range append(AllColors, Colorless) {
		count := set.Statistics.ByColor[color]
		avg := set.Statistics.AverageCMCByColor[color]
		avgStr := "—"
		if avg > 0 {
			avgStr = fmt.Sprintf("%.1f", avg)
		}
		fmt.Printf("  [%s] %d cards, avg CMC %s\n", color.Symbol(), count, avgStr)
	}
	fmt.Println()

	fmt.Println("── Rarity ──")
	for _, r := range AllRarities {
		fmt.Printf("  %s: %d\n", r, set.Statistics.ByRarity[r])
	}
	fmt.Println()

	fmt.Println("── Types ──")
	types := make([]string, 0, len(set.Statistics.ByType))
	for t := range set.Statistics.ByType {
		types = append(types, t)
	}
	sort.Strings(types)
	for _, t := range types {
		fmt.Printf("  %s: %d\n", capitalize(t), set.Statistics.ByType[t])
	}
	fmt.Printf("  Creature %%: %.0f%%\n", set.Statistics.CreaturePercentage*100)
	fmt.Printf("  Removal spells: %d\n", set.Statistics.RemovalCount)
	fmt.Println()
}

func PrintCardList(set GeneratedSet) {
	colorOrder := append(AllColors, Colorless)

	byColor := make(map[Color][]GeneratedCard)
	for _, gc := range set.Cards {
		byColor[gc.Color] = append(byColor[gc.Color], gc)
	}

	for _, color := range colorOrder {
		colorCards := byColor[color]
		if len(colorCards) == 0 {
			continue
		}

		// Sort by CMC
		sort.Slice(colorCards, func(i, j int) bool {
			return colorCards[i].Card.CMC() < colorCards[j].Card.CMC()
		})

		fmt.Printf("── %s (%d cards) ──\n", color.Label(), len(colorCards))
		for _, gc := range colorCards {
			c := gc.Card
			num := fmt.Sprintf("#%03d", gc.CollectorNumber)
			rar := gc.Rarity.Char()

			if c.IsCreature() {
				kwStr := ""
				if len(c.Keywords) > 0 {
					var kws []string
					for _, kw := range c.Keywords {
						kws = append(kws, kw.String())
					}
					kwStr = " [" + strings.Join(kws, ", ") + "]"
				}
				abilStr := ""
				if len(c.Abilities) > 0 {
					abilStr = " {" + strings.Join(c.Abilities, "; ") + "}"
				}
				fmt.Printf("  %s [%s] %s %s — %s %d/%d%s%s\n",
					num, rar, c.ManaCost, c.Name,
					strings.Join(c.Subtypes, " "),
					c.Power, c.Toughness, kwStr, abilStr)
			} else if c.IsInstant() || c.IsSorcery() {
				typeStr := "Sorcery"
				if c.IsInstant() {
					typeStr = "Instant"
				}
				effectStr := ""
				if len(c.Abilities) > 0 {
					effectStr = c.Abilities[0]
				}
				fmt.Printf("  %s [%s] %s %s — %s: %s\n",
					num, rar, c.ManaCost, c.Name, typeStr, effectStr)
			} else if c.IsEnchantment() {
				subStr := "Enchantment"
				if len(c.Subtypes) > 0 {
					subStr = "Enchantment — " + strings.Join(c.Subtypes, " ")
				}
				descStr := ""
				if len(c.Abilities) > 0 {
					descStr = c.Abilities[0]
				}
				fmt.Printf("  %s [%s] %s %s — %s: %s\n",
					num, rar, c.ManaCost, c.Name, subStr, descStr)
			} else {
				fmt.Printf("  %s [%s] %s %s\n", num, rar, c.ManaCost, c.Name)
			}
		}
		fmt.Println()
	}
}
