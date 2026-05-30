package ai

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestInferPersonalityFromCards_Aggro(t *testing.T) {
	cards := make([]mage.Card, 0, 22)
	for range 16 {
		cards = append(cards, mage.NewCreature("One Drop", "{W}", 2, 1))
	}
	for range 6 {
		cards = append(cards, mage.NewBoostAura("Holy Strength", "{W}", 1, 2))
	}

	got := InferPersonalityFromCards(cards)
	if got.Name != AggroWeighted.Name {
		t.Fatalf("got %s, want %s", got.Name, AggroWeighted.Name)
	}
}

func TestInferPersonalityFromCards_Burn(t *testing.T) {
	cards := make([]mage.Card, 0, 18)
	for range 14 {
		cards = append(cards, mage.NewInstant("Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
		))
	}
	for range 4 {
		cards = append(cards, mage.NewCreature("Goblin", "{R}", 2, 1))
	}

	got := InferPersonalityFromCards(cards)
	if got.Name != BurnWeighted.Name {
		t.Fatalf("got %s, want %s", got.Name, BurnWeighted.Name)
	}
}

func TestInferPersonalityFromCards_Control(t *testing.T) {
	cards := make([]mage.Card, 0, 17)
	for range 8 {
		cards = append(cards, mage.NewSorcery("Destroy", "{2}{B}",
			mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
		))
	}
	for range 6 {
		cards = append(cards, mage.NewSorcery("Draw", "{3}{U}", mage.DrawCards(mage.Fixed(2))))
	}
	for range 3 {
		cards = append(cards, mage.NewCreature("Finisher", "{5}{U}", 5, 5, mage.WithKeyword(core.Flying)))
	}

	got := InferPersonalityFromCards(cards)
	if got.Name != ControlWeighted.Name {
		t.Fatalf("got %s, want %s", got.Name, ControlWeighted.Name)
	}
}
