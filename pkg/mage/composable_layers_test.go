package mage

import (
	"slices"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableTypeAndColorCapture(t *testing.T) {
	g, player, _, land, printed := basicLandTypeTestGame(t)
	colors := []Color{Blue, Green}
	subtypes := []string{"Forest"}
	effect := Until(EndOfTurn,
		SetSubtypeFamily(This, SubtypeLand, subtypes...),
		SetColors(This, colors...),
		GrantKeywordModification(This, Flying),
	)
	colors[0] = Red
	subtypes[0] = "Island"
	if err := ApplyEffect(g, effect, land.ID(), player.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	land = g.FindPermanent(land.ID())
	if !land.HasSubType("Forest") || land.HasSubType("Mountain") || hasRuntimeAbility(land, printed) || countManaAbilityColor(land, Green) != 1 {
		t.Fatal("subtype replacement did not use the basic land rules")
	}
	if !slices.Equal(g.EffectiveColors(land.ID()), []Color{Blue, Green}) || !land.HasAttr(Flying) {
		t.Fatal("compound effect lost its color or later-layer keyword")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(land.ID()).HasSubType("Mountain") || !g.FindPermanent(land.ID()).HasSubType("Forest") {
		t.Fatal("expiration leaked between clones")
	}
	g.RemoveFromBattlefield(land)
	g.PutOnBattlefield(land.Card, player.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(land.ID()).HasSubType("Forest") || slices.Contains(g.EffectiveColors(land.ID()), Blue) {
		t.Fatal("captured modification followed a new incarnation")
	}
}

func TestComposableAddedSubtypePreservesAbilities(t *testing.T) {
	g, player, _, land, printed := basicLandTypeTestGame(t)
	if err := ApplyEffect(g, Until(EndOfTurn, AddSubtypeFamily(This, SubtypeLand, "Forest")), land.ID(), player.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	land = g.FindPermanent(land.ID())
	if !land.HasSubType("Mountain") || !land.HasSubType("Forest") || !hasRuntimeAbility(land, printed) || countManaAbilityColor(land, Red) != 1 || countManaAbilityColor(land, Green) != 1 {
		t.Fatal("adding a land subtype did not preserve old types and abilities")
	}
}

func TestComposableStaticColorUsesLayerMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(Static(While(Controls(IsCreature),
		SetEachColors(ControlledPermanents(IsCreature), Blue),
		GrantEachKeyword(ControlledPermanents(HasColorFilter(Blue)), Flying),
	))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	later := NewCreature("Later", "{R}", 2, 2)
	later.SetOwner(a.PlayerID())
	g.PutOnBattlefield(later, a.PlayerID())
	g.ApplyContinuousEffects()
	if !slices.Equal(g.EffectiveColors(later.ID()), []Color{Blue}) || !g.FindPermanent(later.ID()).HasAttr(Flying) {
		t.Fatal("static set omitted a later member")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ApplyContinuousEffects()
	if !slices.Equal(g.EffectiveColors(later.ID()), []Color{Red}) || g.FindPermanent(later.ID()).HasAttr(Flying) {
		t.Fatal("static effect remained after source exit")
	}
}

func TestComposableLayerValidation(t *testing.T) {
	for name, makeModification := range map[string]func() ContinuousModification{
		"invalid color":      func() ContinuousModification { return SetColors(This, AnyColor) },
		"colorless sentinel": func() ContinuousModification { return SetColors(This, Colorless) },
		"invalid family":     func() ContinuousModification { return SetSubtypeFamily(This, SubtypeFamily(255), "Human") },
		"wrong family":       func() ContinuousModification { return AddSubtypeFamily(This, SubtypeCreature, "Forest") },
		"nil subject":        func() ContinuousModification { return SetColors(nil, Blue) },
		"nil set":            func() ContinuousModification { return SetEachColors(nil, Blue) },
	} {
		t.Run(name, func(t *testing.T) {
			err := validateFactory(name, func() Card {
				return NewCreature(name, "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, makeModification()))))
			})
			if err == nil {
				t.Fatal("invalid modification accepted")
			}
		})
	}
}

func TestComposableColorlessAndSubtypeApplicability(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "{R}", 2, 2, WithSubTypes("Goblin"))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	effect := Until(EndOfTurn, SetColors(This), SetSubtypeFamily(This, SubtypeLand, "Forest"))
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	p := g.FindPermanent(card.ID())
	if len(g.EffectiveColors(card.ID())) != 0 || !p.HasSubType("Goblin") || p.HasSubType("Forest") {
		t.Fatal("colorless or subtype applicability contract failed")
	}
	phased := g.PhaseOut(card.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if len(g.EffectiveColors(card.ID())) != 0 {
		t.Fatal("phasing invalidated the captured reference")
	}
}
