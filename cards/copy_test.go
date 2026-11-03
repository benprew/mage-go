package cards_test

import (
	"testing"

	_ "github.com/benprew/mage-go/cards"
	_ "github.com/benprew/mage-go/cards/jumpstart"
	_ "github.com/benprew/mage-go/cards/secretsofstrixhaven"
	"github.com/benprew/mage-go/pkg/mage"
)

func TestRegisteredCardsCopyOwnsAbilityState(t *testing.T) {
	for _, name := range mage.RegisteredCardNames() {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if reason := recover(); reason != nil {
					t.Errorf("copy failed: %v", reason)
				}
			}()
			original, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			copied := original.Copy()
			cloned := mage.NewCreature("Clone", "{3}{U}", 0, 0)
			cloned.CloneFrom(original)
			for _, card := range []*mage.Card{copied, cloned} {
				if card.ID() == original.ID() || card.Name() != original.Name() {
					t.Fatal("copy lost its identity or characteristics")
				}
				abilities := original.Abilities()
				if len(card.Abilities()) != len(abilities) {
					t.Fatal("copy lost abilities")
				}
				for i, ability := range card.Abilities() {
					if card == copied && ability.AbilityID() != abilities[i].AbilityID() {
						t.Fatal("copy changed definition identity")
					}
				}
			}
		})
	}
}

func TestStoneGiantCopyOwnsTargets(t *testing.T) {
	card, err := mage.CreateCard("Stone Giant")
	if err != nil {
		t.Fatal(err)
	}
	target := card.Abilities()[0].(mage.ActivatedAbility).Targets()[0]
	copyTarget := card.NewCopy().Abilities()[0].(mage.ActivatedAbility).Targets()[0]
	cloneTarget := card.CloneInstance().Abilities()[0].(mage.ActivatedAbility).Targets()[0]
	if copyTarget != target || cloneTarget != target {
		t.Fatal("copy changed immutable target specification")
	}
}
