package mage

import (
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// cardCharacteristics contains values without card identity or execution state.
type cardCharacteristics struct {
	name          string
	manaCost      ManaCost
	types         []CardType
	superTypes    []SuperType
	subTypes      []string
	power         int
	toughness     int
	colorOverride []Color
}

func (c *cardCharacteristics) clone() *cardCharacteristics {
	if c == nil {
		return nil
	}
	cloned := *c
	cloned.manaCost = copyPrintedManaCost(c.manaCost)
	cloned.types = slices.Clone(c.types)
	cloned.superTypes = slices.Clone(c.superTypes)
	cloned.subTypes = slices.Clone(c.subTypes)
	cloned.colorOverride = slices.Clone(c.colorOverride)
	return &cloned
}
