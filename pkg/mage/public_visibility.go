package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RevealAllHands keeps every player's current hand visible while the source is active.
func RevealAllHands() StaticRule { return publicVisibilityRule{hands: true} }

// RevealAllLibraryTops keeps every library's top card visible while the source is active.
func RevealAllLibraryTops() StaticRule { return publicVisibilityRule{} }

type publicVisibilityRule struct{ hands bool }

func (publicVisibilityRule) staticRule() {}
func (r publicVisibilityRule) inspectDefinition() definitionNode {
	text := "public library tops"
	if r.hands {
		text = "public hands"
	}
	return definitionNode{Role: "static-rule", Text: text, battlefieldSource: true}
}
func (r publicVisibilityRule) staticEffects() []ContinuousEffect {
	if !r.hands {
		return []ContinuousEffect{RevealTopCardsOfAllLibraries()}
	}
	return []ContinuousEffect{FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		if g.FindPermanent(sourceID) == nil {
			return nil
		}
		for _, player := range g.AllPlayers() {
			g.effects.Rules.AddRevealedHand(player.PlayerID())
		}
		return nil
	})}
}

// AddRevealedHand marks a hand as public for the current layer pass.
func (r *GameRules) AddRevealedHand(playerID uuid.UUID) {
	if r.revealedHand == nil {
		r.revealedHand = make(map[uuid.UUID]bool)
	}
	r.revealedHand[playerID] = true
}

// IsHandRevealed reports whether a player's current hand is public.
func (r *GameRules) IsHandRevealed(playerID uuid.UUID) bool { return r.revealedHand[playerID] }

// IsHandRevealed reports whether a player's current hand is public.
func (g *Game) IsHandRevealed(playerID uuid.UUID) bool {
	return g.effects.Rules.IsHandRevealed(playerID)
}
