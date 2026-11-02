package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

var benchmarkAbilityList []Ability
var benchmarkKeyword bool
var benchmarkPermanent *Permanent

func BenchmarkCardAbilitiesSnapshot(b *testing.B) {
	card := NewCreature("Keyword Creature", "{U}", 1, 1, WithKeyword(Flying), WithKeyword(FirstStrike))
	b.ReportAllocs()
	for b.Loop() {
		benchmarkAbilityList = card.Abilities()
	}
}

func BenchmarkCardHasKeyword(b *testing.B) {
	card := NewCreature("Keyword Creature", "{U}", 1, 1, WithKeyword(Flying), WithKeyword(Flash))
	b.ReportAllocs()
	for b.Loop() {
		benchmarkKeyword = cardHasKeyword(card, Flash)
	}
}

func BenchmarkNewPermanentWithKeywords(b *testing.B) {
	card := NewCreature("Keyword Creature", "{U}", 1, 1, WithKeyword(Flying), WithKeyword(FirstStrike))
	controller := uuid.New()
	b.ReportAllocs()
	for b.Loop() {
		benchmarkPermanent = NewPermanent(card, controller)
	}
}

func BenchmarkCardAllAbilities(b *testing.B) {
	card := NewCreature("Keyword Creature", "{U}", 1, 1, WithKeyword(Flying), WithKeyword(FirstStrike))
	b.ReportAllocs()
	for b.Loop() {
		found := false
		for _, ability := range card.AllAbilities() {
			found = found || AbilityAttribute(ability) == FirstStrike
		}
		benchmarkKeyword = found
	}
}
