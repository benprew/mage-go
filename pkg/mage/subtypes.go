package mage

import (
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// SubtypeFamily identifies one of the independent subtype sets defined by CR
// 205.3. Setting subtypes replaces only subtypes in the selected family.
type SubtypeFamily uint8

const (
	SubtypeArtifact SubtypeFamily = iota
	SubtypeCreature
	SubtypeEnchantment
	SubtypeLand
	SubtypePlaneswalker
	SubtypeSpell
	subtypeFamilyCount
)

func (f SubtypeFamily) valid() bool {
	return f < subtypeFamilyCount
}

func (p *Permanent) supportsSubtypeFamily(family SubtypeFamily) bool {
	switch family {
	case SubtypeArtifact:
		return p.HasType(TypeArtifact)
	case SubtypeCreature:
		return p.HasType(TypeCreature)
	case SubtypeEnchantment:
		return p.HasType(TypeEnchantment)
	case SubtypeLand:
		return p.HasType(TypeLand)
	case SubtypePlaneswalker:
		return p.HasType(TypePlaneswalker)
	case SubtypeSpell:
		return p.HasType(TypeInstant) || p.HasType(TypeSorcery)
	default:
		return false
	}
}

var artifactSubtypes = subtypeNameSet(
	"Attraction", "Blood", "Bobblehead", "Clue", "Contraption", "Equipment",
	"Food", "Fortification", "Gold", "Incubator", "Infinity", "Junk", "Lander",
	"Map", "Mutagen", "Powerstone", "Spacecraft", "Stone", "Treasure", "Vehicle",
)

var enchantmentSubtypes = subtypeNameSet(
	"Aura", "Background", "Cartouche", "Case", "Class", "Curse", "Role", "Room",
	"Rune", "Saga", "Shard", "Shrine",
)

var spellSubtypes = subtypeNameSet("Adventure", "Arcane", "Lesson", "Omen", "Trap")

var planeswalkerSubtypes = subtypeNameSet(
	"Ajani", "Aminatou", "Angrath", "Arlinn", "Ashiok", "Bahamut", "Bolas", "Calix",
	"Chandra", "Comet", "Dack", "Dakkon", "Daretti", "Davriel", "Dihada", "Domri",
	"Dovin", "Ellywick", "Elminster", "Elspeth", "Estrid", "Freyalise", "Garruk",
	"Gideon", "Grist", "Guff", "Huatli", "Jace", "Jared", "Jaya", "Jeska", "Kaito",
	"Karn", "Kasmina", "Kaya", "Kiora", "Koth", "Liliana", "Lolth", "Lukka", "Minsc",
	"Mordenkainen", "Nahiri", "Narset", "Niko", "Nissa", "Nixilis", "Oko", "Quintorius",
	"Ral", "Rowan", "Saheeli", "Samut", "Sarkhan", "Serra", "Sivitri", "Sorin", "Szat",
	"Tamiyo", "Tasha", "Teferi", "Teyo", "Tezzeret", "Tibalt", "Tyvar", "Ugin", "Urza",
	"Venser", "Vivien", "Vraska", "Vronos", "Will", "Windgrace", "Wrenn", "Xenagos",
	"Yanggu", "Yanling", "Zariel",
)

func subtypeNameSet(names ...string) map[string]bool {
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = true
	}
	return result
}

func subtypeFamilyForCard(card Card, subtype string) (SubtypeFamily, bool) {
	if family, ok := knownSubtypeFamily(subtype); ok {
		return family, true
	}
	switch {
	case card.HasType(TypeCreature):
		return SubtypeCreature, true
	case card.HasType(TypeArtifact):
		return SubtypeArtifact, true
	case card.HasType(TypeEnchantment):
		return SubtypeEnchantment, true
	case card.HasType(TypeLand):
		return SubtypeLand, true
	case card.HasType(TypePlaneswalker):
		return SubtypePlaneswalker, true
	case card.HasType(TypeInstant) || card.HasType(TypeSorcery):
		return SubtypeSpell, true
	default:
		return 0, false
	}
}

func knownSubtypeFamily(subtype string) (SubtypeFamily, bool) {
	switch {
	case isLandSubtype(subtype):
		return SubtypeLand, true
	case artifactSubtypes[subtype]:
		return SubtypeArtifact, true
	case enchantmentSubtypes[subtype]:
		return SubtypeEnchantment, true
	case spellSubtypes[subtype]:
		return SubtypeSpell, true
	case planeswalkerSubtypes[subtype]:
		return SubtypePlaneswalker, true
	default:
		return 0, false
	}
}

func (p *Permanent) computedSubtypes() []string {
	base := p.Card.SubTypes()
	if len(p.SubTypeOverride) > 0 {
		base = p.SubTypeOverride
	}
	base = append(append([]string(nil), base...), p.SubTypeAdditions...)

	var families [subtypeFamilyCount][]string
	var unclassified []string
	for _, subtype := range base {
		family, ok := subtypeFamilyForCard(p.Card, subtype)
		if !ok {
			unclassified = append(unclassified, subtype)
			continue
		}
		families[family] = appendUniqueSubtype(families[family], subtype)
	}

	for family := range subtypeFamilyCount {
		if p.subtypeFamilyOverrideSet[family] {
			families[family] = append([]string(nil), p.subtypeFamilyOverrides[family]...)
		}
		for _, subtype := range p.subtypeFamilyAdditions[family] {
			families[family] = appendUniqueSubtype(families[family], subtype)
		}
	}

	result := append([]string(nil), unclassified...)
	for family := range subtypeFamilyCount {
		if !p.supportsSubtypeFamily(family) {
			continue
		}
		for _, subtype := range families[family] {
			result = appendUniqueSubtype(result, subtype)
		}
	}
	return result
}

func appendUniqueSubtype(subtypes []string, subtype string) []string {
	if subtype == "" || slices.Contains(subtypes, subtype) {
		return subtypes
	}
	return append(subtypes, subtype)
}

func (p *Permanent) setSubtypeFamily(family SubtypeFamily, subtypes []string) {
	if !family.valid() {
		return
	}
	p.subtypeFamilyOverrideSet[family] = true
	p.subtypeFamilyOverrides[family] = append([]string(nil), subtypes...)
	p.subtypeFamilyAdditions[family] = nil
}

func (p *Permanent) addSubtypeFamily(family SubtypeFamily, subtypes []string) {
	if !family.valid() {
		return
	}
	for _, subtype := range subtypes {
		p.subtypeFamilyAdditions[family] = appendUniqueSubtype(p.subtypeFamilyAdditions[family], subtype)
	}
}

func (p *Permanent) hasSubtypeFamilyEffects() bool {
	for family := range subtypeFamilyCount {
		if p.subtypeFamilyOverrideSet[family] || len(p.subtypeFamilyAdditions[family]) > 0 {
			return true
		}
	}
	return false
}

func (p *Permanent) resetSubtypeFamilyEffects() {
	for family := range subtypeFamilyCount {
		p.subtypeFamilyOverrideSet[family] = false
		p.subtypeFamilyOverrides[family] = nil
		p.subtypeFamilyAdditions[family] = nil
	}
}
