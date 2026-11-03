package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAllAbilities(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Iterator Bird", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}},
		Abilities: []Ability{WardTrigger(ManaCostOf("{2}")), keywordRule(Flying), keywordRule(FirstStrike), keywordRule(Flying)},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range []*Card{
		NewCreature("Empty", "", 1, 1),
		NewCreature("Legacy Bird", "{U}", 1, 1, WithKeyword(Flying), WithWard(ManaCostOf("{2}"))),
		def.NewInstance(uuid.New()),
	} {
		t.Run(card.Name(), func(t *testing.T) {
			snapshot := card.Abilities()
			sequence := card.AllAbilities()
			for range 2 {
				count := 0
				for index, ability := range sequence {
					if index != count || ability != snapshot[index] {
						t.Fatal("iteration changed ability order or identity")
					}
					count++
				}
				if count != len(snapshot) {
					t.Fatal("iteration omitted an ability")
				}
			}
			visited := 0
			for range card.AllAbilities() {
				visited++
				break
			}
			if len(snapshot) > 0 && visited != 1 {
				t.Fatal("iteration did not stop early")
			}
			if len(snapshot) == 0 && visited != 0 {
				t.Fatal("empty card produced an ability")
			}
			if len(snapshot) > 0 {
				snapshot[0] = nil
				for _, ability := range card.AllAbilities() {
					if ability == nil {
						t.Fatal("snapshot mutation changed the card")
					}
				}
			}
		})
	}
}

func TestAbilityIterationAllocations(t *testing.T) {
	card := NewCreature("Flash Bird", "{U}", 1, 1, WithKeyword(Flying), WithKeyword(Flash))
	visited := 0
	if allocations := testing.AllocsPerRun(100, func() {
		for _, ability := range card.AllAbilities() {
			if AbilityAttribute(ability) != 0 {
				visited++
			}
		}
	}); allocations != 0 {
		t.Fatalf("ability iteration allocated %g times", allocations)
	}
	if visited == 0 {
		t.Fatal("iteration did not inspect abilities")
	}
	found := false
	if allocations := testing.AllocsPerRun(100, func() { found = cardHasKeyword(card, Flash) }); allocations != 0 {
		t.Fatalf("keyword check allocated %g times", allocations)
	}
	if !found {
		t.Fatal("keyword check did not find Flash")
	}
}
