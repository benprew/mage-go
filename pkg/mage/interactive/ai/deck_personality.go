package ai

import (
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// DeckCard is the minimal deck-list shape needed for deck-based AI profiling.
type DeckCard struct {
	Name  string
	Count int
}

// InferPersonalityFromDeck chooses an AI personality from a deck list.
func InferPersonalityFromDeck(entries []DeckCard) WeightedPersonality {
	var cards []mage.Card
	for _, entry := range entries {
		for range entry.Count {
			card, err := mage.CreateCard(entry.Name)
			if err != nil {
				continue
			}
			cards = append(cards, card)
		}
	}
	return InferPersonalityFromCards(cards)
}

// InferPersonalityFromCards chooses an AI personality from card mix, curve, and
// effect metadata.
func InferPersonalityFromCards(cards []mage.Card) WeightedPersonality {
	stats := analyzeDeck(cards)
	if stats.total == 0 {
		return MidrangeWeighted
	}

	nonLand := max(stats.total-stats.lands, 1)
	creatureRatio := float64(stats.creatures) / float64(nonLand)
	cheapCreatureRatio := float64(stats.cheapCreatures) / float64(nonLand)
	evasiveRatio := float64(stats.evasiveCreatures) / float64(max(stats.creatures, 1))
	burnRatio := float64(stats.burn) / float64(nonLand)
	removalRatio := float64(stats.removal) / float64(nonLand)
	drawRatio := float64(stats.drawEngines+stats.cardDraw) / float64(nonLand)
	pumpRatio := float64(stats.pump+stats.auras) / float64(nonLand)
	bounceRatio := float64(stats.bounce) / float64(nonLand)
	manaRatio := float64(stats.manaAccel) / float64(nonLand)

	avgCMC := 0.0
	if stats.nonLandCMCCount > 0 {
		avgCMC = float64(stats.nonLandCMC) / float64(stats.nonLandCMCCount)
	}

	aggro := creatureRatio*2.0 + cheapCreatureRatio*3.0 + pumpRatio*1.3 + burnRatio*1.2
	if avgCMC <= 2.4 {
		aggro += 1.0
	}

	burn := burnRatio*5.0 + cheapCreatureRatio*1.2 + pumpRatio*0.5

	tempo := cheapCreatureRatio*1.7 + evasiveRatio*1.7 + bounceRatio*4.0 + stats.counterScore()*2.0 + manaRatio*0.8
	if avgCMC <= 3.0 {
		tempo += 0.7
	}

	control := removalRatio*2.4 + drawRatio*2.3 + stats.counterScore()*2.5
	if creatureRatio < 0.35 {
		control += 1.0
	}
	if avgCMC >= 3.4 {
		control += 0.8
	}

	midrange := creatureRatio*1.5 + removalRatio*1.4 + manaRatio*1.0
	if avgCMC >= 2.6 && avgCMC <= 4.0 {
		midrange += 1.0
	}

	bestName := "midrange"
	bestScore := midrange
	for _, candidate := range []struct {
		name  string
		score float64
	}{
		{"aggro", aggro},
		{"burn", burn},
		{"tempo", tempo},
		{"control", control},
	} {
		if candidate.score > bestScore && deckHasProfileSignal(candidate.name, stats, avgCMC, creatureRatio, cheapCreatureRatio, burnRatio, bounceRatio, removalRatio, drawRatio) {
			bestScore = candidate.score
			bestName = candidate.name
		}
	}

	switch bestName {
	case "aggro":
		return AggroWeighted
	case "burn":
		return BurnWeighted
	case "tempo":
		return TempoWeighted
	case "control":
		return ControlWeighted
	default:
		return MidrangeWeighted
	}
}

func deckHasProfileSignal(name string, stats deckStats, avgCMC, creatureRatio, cheapCreatureRatio, burnRatio, bounceRatio, removalRatio, drawRatio float64) bool {
	switch name {
	case "aggro":
		return creatureRatio >= 0.50 && cheapCreatureRatio >= 0.42 && avgCMC <= 3.0
	case "burn":
		return burnRatio >= 0.22 && stats.burn >= 6
	case "tempo":
		return cheapCreatureRatio >= 0.30 && (bounceRatio >= 0.08 || stats.counterScore() >= 0.08 || float64(stats.evasiveCreatures) >= 6)
	case "control":
		return creatureRatio <= 0.45 && removalRatio+drawRatio+stats.counterScore() >= 0.32
	default:
		return true
	}
}

type deckStats struct {
	total            int
	lands            int
	creatures        int
	cheapCreatures   int
	evasiveCreatures int
	nonLandCMC       int
	nonLandCMCCount  int
	burn             int
	removal          int
	bounce           int
	pump             int
	auras            int
	cardDraw         int
	drawEngines      int
	counterspells    int
	manaAccel        int
}

func (s deckStats) counterScore() float64 {
	return float64(s.counterspells) / float64(max(s.total-s.lands, 1))
}

func analyzeDeck(cards []mage.Card) deckStats {
	var stats deckStats
	for _, card := range cards {
		if card == nil {
			continue
		}
		stats.total++
		if card.HasType(core.TypeLand) {
			stats.lands++
			continue
		}

		cmc := card.ManaCost().CMC()
		stats.nonLandCMC += cmc
		stats.nonLandCMCCount++

		if card.HasType(core.TypeCreature) {
			stats.creatures++
			if cmc <= 2 {
				stats.cheapCreatures++
			}
			if hasDeckEvasion(card) {
				stats.evasiveCreatures++
			}
		}
		if card.HasSubType("Aura") {
			stats.auras++
		}
		if hasManaAcceleration(card) {
			stats.manaAccel++
		}
		classifyDeckCardEffects(card, &stats)
	}
	return stats
}

func classifyDeckCardEffects(card mage.Card, stats *deckStats) {
	for _, ability := range card.Abilities() {
		switch ab := mage.UnwrapAbility(ability).(type) {
		case *mage.SpellAbility:
			classifyEffects(ab.Effects(), card, stats)
		case mage.ActivatedAbility:
			classifyEffects(ab.Effects(), card, stats)
		case mage.TriggeredAbility:
			classifyEffects(ab.Effects(), card, stats)
		}
	}
}

func classifyEffects(effects []mage.Effect, card mage.Card, stats *deckStats) {
	for _, effect := range effects {
		props := effect.Properties()
		if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
			stats.burn++
			continue
		}
		if props.IsBounce {
			stats.bounce++
			continue
		}
		if props.Outcome == mage.OutcomeDetriment {
			if strings.Contains(strings.ToLower(effect.Text()), "counter") {
				stats.counterspells++
			} else {
				stats.removal++
			}
			continue
		}
		if props.DrawCount > 0 {
			if card.HasType(core.TypeArtifact) || card.HasType(core.TypeEnchantment) {
				stats.drawEngines++
			} else {
				stats.cardDraw++
			}
		}
		if props.PowerBoost != 0 || props.ToughnessBoost != 0 || props.GrantedKeyword != 0 {
			stats.pump++
		}
	}
}

func hasDeckEvasion(card mage.Card) bool {
	attrs := card.AttrSeeds()
	for _, attr := range []core.Attr{
		core.Flying, core.Fear, core.Menace, core.UnblockableKW,
		core.Islandwalk, core.Swampwalk, core.Forestwalk,
		core.Mountainwalk, core.Plainswalk, core.Trample,
	} {
		if attrs[attr] > 0 {
			return true
		}
	}
	return false
}

func hasManaAcceleration(card mage.Card) bool {
	if card.HasType(core.TypeLand) {
		return false
	}
	for _, ability := range card.Abilities() {
		if _, ok := mage.UnwrapAbility(ability).(*mage.ManaAbility); ok {
			return true
		}
	}
	return false
}
