package main

import "strings"

// NameGenerator generates evocative card names from color-themed word pools.

func CreatureName(color Color, subtype string, rarity Rarity, rng *RNG) string {
	phil := PieFor(color).Philosophy

	type pattern int
	const (
		adjectiveType pattern = iota
		legendaryType
		locationCreature
		theNounType
		typeOfNoun
		singleWord
		adjectiveNounType
		verbingType
		titleType
		nounNounType
		theAdjective
	)

	legendaryWeight := 0.04
	titleWeight := 0.02
	if rarity == Rare {
		legendaryWeight = 0.20
		titleWeight = 0.12
	}

	p := WeightedPick([]WeightedOption[pattern]{
		W(adjectiveType, 0.25),
		W(legendaryType, legendaryWeight),
		W(locationCreature, 0.12),
		W(theNounType, 0.10),
		W(typeOfNoun, 0.08),
		W(singleWord, 0.05),
		W(adjectiveNounType, 0.06),
		W(verbingType, 0.08),
		W(titleType, titleWeight),
		W(nounNounType, 0.06),
		W(theAdjective, 0.04),
	}, rng)

	switch p {
	case adjectiveType:
		return rng.Pick(phil.Adjectives) + " " + subtype

	case legendaryType:
		name := rng.Pick(phil.LegendaryNames)
		switch rng.Intn(3) {
		case 0:
			return name + "'s " + subtype
		case 1:
			return name + " " + subtype
		default:
			return subtype + " of " + name
		}

	case locationCreature:
		return rng.Pick(phil.Locations) + " " + subtype

	case theNounType:
		return rng.Pick(phil.Nouns) + " " + subtype

	case typeOfNoun:
		noun := rng.Pick(phil.Nouns)
		switch rng.Intn(3) {
		case 0:
			return subtype + " of " + noun
		case 1:
			return subtype + " of the " + noun
		default:
			return subtype + " of " + rng.Pick(phil.Adjectives) + " " + noun
		}

	case singleWord:
		if rarity == Rare && rng.Bool() {
			name := rng.Pick(phil.LegendaryNames)
			titles := []string{
				"the " + rng.Pick(phil.Adjectives),
				rng.Pick(phil.Nouns) + " Keeper",
				"of the " + rng.Pick(phil.Nouns),
				subtype + " King",
				"Last of the " + subtype + "s",
			}
			return name + ", " + rng.Pick(titles)
		}
		return rng.Pick(phil.Adjectives) + " " + subtype

	case adjectiveNounType:
		return rng.Pick(phil.Adjectives) + rng.Pick(phil.Nouns) + " " + subtype

	case verbingType:
		verb := rng.Pick(phil.Verbs)
		gerund := toGerund(verb)
		return gerund + " " + subtype

	case titleType:
		name := rng.Pick(phil.LegendaryNames)
		adj := rng.Pick(phil.Adjectives)
		titles := []string{
			name + ", the " + adj,
			name + ", " + adj + " " + subtype,
			name + " the " + adj,
			name + ", " + subtype + " Lord",
			name + ", " + rng.Pick(phil.Nouns) + " of " + rng.Pick(phil.Locations),
		}
		return rng.Pick(titles)

	case nounNounType:
		return rng.Pick(phil.Nouns) + rng.Pick(phil.Nouns) + " " + subtype

	case theAdjective:
		adj := rng.Pick(phil.Adjectives)
		if rarity == Rare {
			return "The " + adj + " " + subtype
		}
		return adj + " " + subtype
	}

	return rng.Pick(phil.Adjectives) + " " + subtype
}

func SpellName(color Color, rarity Rarity, rng *RNG) string {
	phil := PieFor(color).Philosophy

	type pattern int
	const (
		adjectiveNoun pattern = iota
		nounOfNoun
		verbNoun
		legendaryPossessive
		singleNoun
		verbOfThe
		adjectiveVerbNoun
		verbTheNoun
		nounNoun
		locationNoun
		theAdjNoun
		verbAndVerb
	)

	legendaryWeight := 0.04
	if rarity == Rare {
		legendaryWeight = 0.12
	}

	p := WeightedPick([]WeightedOption[pattern]{
		W(adjectiveNoun, 0.20),
		W(nounOfNoun, 0.12),
		W(verbNoun, 0.15),
		W(legendaryPossessive, legendaryWeight),
		W(singleNoun, 0.06),
		W(verbOfThe, 0.08),
		W(adjectiveVerbNoun, 0.06),
		W(verbTheNoun, 0.10),
		W(nounNoun, 0.08),
		W(locationNoun, 0.06),
		W(theAdjNoun, 0.04),
		W(verbAndVerb, 0.05),
	}, rng)

	switch p {
	case adjectiveNoun:
		return rng.Pick(phil.Adjectives) + " " + rng.Pick(phil.Nouns)
	case nounOfNoun:
		return rng.Pick(phil.Nouns) + " of " + rng.Pick(phil.Nouns)
	case verbNoun:
		return rng.Pick(phil.Verbs) + " " + rng.Pick(phil.Nouns)
	case legendaryPossessive:
		name := rng.Pick(phil.LegendaryNames)
		if rng.Bool() {
			return name + "'s " + rng.Pick(phil.Nouns)
		}
		return name + "'s " + rng.Pick(phil.Adjectives) + " " + rng.Pick(phil.Nouns)
	case singleNoun:
		return rng.Pick(phil.Nouns)
	case verbOfThe:
		return rng.Pick(phil.Nouns) + " of the " + rng.Pick(phil.Adjectives)
	case adjectiveVerbNoun:
		return rng.Pick(phil.Adjectives) + " " + rng.Pick(phil.Nouns)
	case verbTheNoun:
		return rng.Pick(phil.Verbs) + " the " + rng.Pick(phil.Nouns)
	case nounNoun:
		return rng.Pick(phil.Nouns) + rng.Pick(phil.Nouns)
	case locationNoun:
		return rng.Pick(phil.Locations) + " " + rng.Pick(phil.Nouns)
	case theAdjNoun:
		return "The " + rng.Pick(phil.Adjectives) + " " + rng.Pick(phil.Nouns)
	case verbAndVerb:
		return rng.Pick(phil.Verbs) + " and " + rng.Pick(phil.Verbs)
	}
	return rng.Pick(phil.Adjectives) + " " + rng.Pick(phil.Nouns)
}

func EnchantmentName(color Color, rarity Rarity, rng *RNG) string {
	// Enchantments use the same patterns as spells.
	return SpellName(color, rarity, rng)
}

func toGerund(verb string) string {
	if strings.HasSuffix(verb, "e") {
		return verb[:len(verb)-1] + "ing"
	}
	return verb + "ing"
}
