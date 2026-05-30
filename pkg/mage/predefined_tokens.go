package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Predefined tokens are tokens whose name, type, color, mana cost, and abilities
// are defined by the Comprehensive Rules rather than by the spell or ability
// that creates them (CR 111.10). The constructors in this file produce one
// token of the named kind and put it onto the battlefield under the controller
// of the resolving effect.

// newTreasureToken constructs a fresh Treasure token: a colorless artifact
// token with subtype "Treasure" and the activated ability
// "{T}, Sacrifice this artifact: Add one mana of any color." (CR 111.10c).
func newTreasureToken() *BaseCard {
	tok := NewToken("Treasure", 0, 0, []CardType{TypeArtifact}, []string{"Treasure"})
	tok.colorOverride = []Color{Colorless}
	tok.AddAbility(NewActivatedAbility(
		AddAnyMana(1, Colorless),
		Tap(),
		WithCost(SacrificeSourceCost()),
	))
	return tok
}

// newFoodToken constructs a fresh Food token: a colorless artifact token with
// subtype "Food" and the activated ability
// "{2}, {T}, Sacrifice this artifact: You gain 3 life." (CR 111.10d).
func newFoodToken() *BaseCard {
	tok := NewToken("Food", 0, 0, []CardType{TypeArtifact}, []string{"Food"})
	tok.colorOverride = []Color{Colorless}
	tok.AddAbility(NewActivatedAbility(
		GainLife(3),
		GenericCost(2),
		WithCost(Tap()),
		WithCost(SacrificeSourceCost()),
	))
	return tok
}

// createPredefinedTokenEffect creates one or more predefined tokens and puts
// them onto the battlefield under the controller's control.
type createPredefinedTokenEffect struct {
	name  string
	count int
	build func() *BaseCard
}

// CreateTreasureToken creates an effect that puts one Treasure token onto the
// battlefield under the controller's control. A Treasure token is a colorless
// Treasure artifact token with "{T}, Sacrifice this artifact: Add one mana of
// any color." (CR 111.10c)
func CreateTreasureToken() Effect {
	return &createPredefinedTokenEffect{name: "Treasure", count: 1, build: newTreasureToken}
}

// CreateTreasureTokens creates an effect that puts N Treasure tokens onto the
// battlefield under the controller's control.
func CreateTreasureTokens(count int) Effect {
	return &createPredefinedTokenEffect{name: "Treasure", count: count, build: newTreasureToken}
}

// CreateFoodToken creates an effect that puts one Food token onto the
// battlefield under the controller's control. A Food token is a colorless Food
// artifact token with "{2}, {T}, Sacrifice this artifact: You gain 3 life."
// (CR 111.10d)
func CreateFoodToken() Effect {
	return &createPredefinedTokenEffect{name: "Food", count: 1, build: newFoodToken}
}

// CreateFoodTokens creates an effect that puts N Food tokens onto the
// battlefield under the controller's control.
func CreateFoodTokens(count int) Effect {
	return &createPredefinedTokenEffect{name: "Food", count: count, build: newFoodToken}
}

func (e *createPredefinedTokenEffect) Apply(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
	count := e.count
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		tok := e.build()
		tok.SetOwner(controller)
		g.PutOnBattlefield(tok, controller)
	}
	return nil
}

func (e *createPredefinedTokenEffect) Text() string {
	if e.count <= 1 {
		return fmt.Sprintf("create a %s token", e.name)
	}
	return fmt.Sprintf("create %d %s tokens", e.count, e.name)
}

func (e *createPredefinedTokenEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
