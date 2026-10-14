package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type copyPowerToughness struct{ power, toughness int }

// CopySetPowerToughness sets the copiable base power and toughness.
func CopySetPowerToughness(power, toughness int) CopyModification {
	return copyPowerToughness{power: power, toughness: toughness}
}
func (copyPowerToughness) validateCopyModification() error { return nil }
func (m copyPowerToughness) applyCopyModification(card *BaseCard) {
	card.SetBasePT(m.power, m.toughness)
}

type copySubtypeAddition struct{ subtypes []string }

// CopyAddSubTypes keeps existing subtypes and adds copiable subtypes.
func CopyAddSubTypes(subtypes ...string) CopyModification {
	return copySubtypeAddition{subtypes: slices.Clone(subtypes)}
}
func (m copySubtypeAddition) validateCopyModification() error {
	if len(m.subtypes) == 0 {
		return fmt.Errorf("copy subtype addition requires at least one subtype")
	}
	for _, subtype := range m.subtypes {
		if strings.TrimSpace(subtype) == "" {
			return fmt.Errorf("copy subtype addition requires nonempty subtypes")
		}
	}
	return nil
}
func (m copySubtypeAddition) applyCopyModification(card *BaseCard) {
	for _, subtype := range m.subtypes {
		if !slices.Contains(card.subTypes, subtype) {
			card.AddSubType(subtype)
		}
	}
}

type copyKeywordAddition struct{ keywords []core.Attr }

// CopyAddKeywords keeps existing abilities and adds copiable keywords.
func CopyAddKeywords(keywords ...core.Attr) CopyModification {
	return copyKeywordAddition{keywords: slices.Clone(keywords)}
}
func (m copyKeywordAddition) validateCopyModification() error {
	if len(m.keywords) == 0 {
		return fmt.Errorf("copy keyword addition requires at least one keyword")
	}
	for _, keyword := range m.keywords {
		switch keyword {
		case core.Flying, core.FirstStrike, core.Trample, core.Vigilance, core.Haste, core.Reach, core.Banding, core.Defender, core.Fear, core.Deathtouch, core.Lifelink, core.Shroud, core.Hexproof, core.Menace, core.Indestructible:
		default:
			return fmt.Errorf("unsupported copiable keyword")
		}
	}
	return nil
}
func (m copyKeywordAddition) applyCopyModification(card *BaseCard) {
	for _, keyword := range m.keywords {
		WithKeyword(keyword)(card)
	}
}
