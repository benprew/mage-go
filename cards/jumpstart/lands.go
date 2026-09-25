package jumpstart

import (
	"github.com/google/uuid"

	mage "github.com/benprew/mage-go/pkg/mage"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerLands()
}

// thrivingLand creates a Thriving cycle land. Each enters tapped, has an
// "as it enters, choose a color other than {onColor}" replacement, and a
// {T} ability that adds either {onColor} or one mana of the chosen color
// (controller's choice on activation).
//
// The on-color half is registered as a real mana ability so casting cost
// auto-tap can find it. The chosen-color half is a separate activated
// ability whose FuncEffect reads Permanent.ChosenColor at resolution time.
// Splitting the Oracle text's "X or chosen" into two abilities preserves
// the player's choice (each tap picks one of the two productions) while
// matching the engine's mana-ability detection conventions.
func thrivingLand(name string, onColor Color) Card {
	reason := "color other than " + onColor.String()
	return NewLand(name,
		WithKeyword(EntersTapped),
		WithAbility(ETBChooseColorOtherThan(reason, onColor)),
		WithManaAbility(onColor),
		WithActivatedAbility(
			FuncEffect(
				"add one mana of the chosen color",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					c := perm.ChosenColor
					if c == Colorless || c == AnyColor {
						return nil
					}
					p.ManaPool().Add(c, 1)
					return nil
				}),
			TapCost(This),
		),
	)
}

func registerLands() {

	// Buried Ruin
	// Land
	// {T}: Add {C}.
	// {2}, {T}, Sacrifice this land: Return target artifact card from your graveyard to your hand.
	Register("Buried Ruin", func() Card {
		return NewLand("Buried Ruin",
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithActivatedAbility(
				ReturnFromGraveyardToHandTarget(),
				GenericCost(2),
				WithCost(TapCost(This)),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCardInYourGraveyard(IsArtifactCard)),
			),
		)
	})

	// Mirrodin's Core
	// Land
	// {T}: Add {C}.
	// {T}: Put a charge counter on this land.
	// {T}, Remove a charge counter from this land: Add one mana of any color.
	Register("Mirrodin's Core", func() Card {
		return NewLand("Mirrodin's Core",
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithActivatedAbility(
				mage.AddCounters(Charge, Fixed(1)).Targeting(ToSource()),
				TapCost(This),
			),
			WithActivatedAbility(
				AddAnyMana(1, Colorless),
				TapCost(This),
				WithCost(RemoveCountersCost(Charge, 1)),
			),
		)
	})

	// Phyrexian Tower
	// Legendary Land
	// {T}: Add {C}.
	// {T}, Sacrifice a creature: Add {B}{B}.
	Register("Phyrexian Tower", func() Card {
		return NewLand("Phyrexian Tower",
			WithSuperTypes(SuperLegendary),
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithActivatedAbility(
				AddMana(Black, 2),
				TapCost(This),
				WithCost(SacrificeCreatureCost()),
			),
		)
	})

	// Riptide Laboratory
	// Land
	// {T}: Add {C}.
	// {1}{U}, {T}: Return target Wizard you control to its owner's hand.
	Register("Riptide Laboratory", func() Card {
		wizard := TargetPermanent("wizard", HasSubType("Wizard"), TargetYouControl())
		return NewLand("Riptide Laboratory",
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithAbility(Activate(
				Costs(Mana("{1}{U}"), TapCost(This)),
				ReturnToHand(wizard), WithTargets(wizard),
			)),
		)
	})

	// Rupture Spire
	// Land
	// This land enters tapped.
	// When this land enters, sacrifice it unless you pay {1}.
	// {T}: Add one mana of any color.
	Register("Rupture Spire", func() Card {
		return NewLand("Rupture Spire",
			WithKeyword(EntersTapped),
			WithAbility(
				EntersBattlefieldTrigger(
					UnlessTargetPays(SelectController(), ManaCostOf("{1}"),
						"pay {1} or sacrifice Rupture Spire",
						Sacrifice(This)),
					false,
				),
			),
			WithAnyColorMana(),
		)
	})

	// Terramorphic Expanse
	// Land
	// {T}, Sacrifice this land: Search your library for a basic land card, put it onto the battlefield tapped, then shuffle.
	Register("Terramorphic Expanse", func() Card {
		return NewLand("Terramorphic Expanse",
			WithActivatedAbility(
				FuncEffect(
					"search your library for a basic land card, put it onto the battlefield tapped, then shuffle",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						lib := p.Library()
						var candidates []Card
						for _, c := range lib {
							if isBasicLandCard(c) {
								candidates = append(candidates, c)
							}
						}
						if len(candidates) == 0 {
							p.ShuffleLibrary()
							return nil
						}
						chosen := p.ChooseCardFromLibrary(candidates, "search for basic land", g)
						if chosen == nil {
							p.ShuffleLibrary()
							return nil
						}
						newLib := make([]Card, 0, len(lib)-1)
						for _, c := range lib {
							if c.ID() != chosen.ID() {
								newLib = append(newLib, c)
							}
						}
						p.SetLibrary(newLib)
						p.ShuffleLibrary()
						perm := g.PutOnBattlefield(chosen, controller)
						if perm != nil {
							perm.Tapped = true
						}
						return nil
					}),
				TapCost(This),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Thriving Bluff
	// Land
	// This land enters tapped. As it enters, choose a color other than red.
	// {T}: Add {R} or one mana of the chosen color.
	Register("Thriving Bluff", func() Card { return thrivingLand("Thriving Bluff", Red) })

	// Thriving Grove
	// Land
	// This land enters tapped. As it enters, choose a color other than green.
	// {T}: Add {G} or one mana of the chosen color.
	Register("Thriving Grove", func() Card { return thrivingLand("Thriving Grove", Green) })

	// Thriving Heath
	// Land
	// This land enters tapped. As it enters, choose a color other than white.
	// {T}: Add {W} or one mana of the chosen color.
	Register("Thriving Heath", func() Card { return thrivingLand("Thriving Heath", White) })

	// Thriving Isle
	// Land
	// This land enters tapped. As it enters, choose a color other than blue.
	// {T}: Add {U} or one mana of the chosen color.
	Register("Thriving Isle", func() Card { return thrivingLand("Thriving Isle", Blue) })

	// Thriving Moor
	// Land
	// This land enters tapped. As it enters, choose a color other than black.
	// {T}: Add {B} or one mana of the chosen color.
	Register("Thriving Moor", func() Card { return thrivingLand("Thriving Moor", Black) })
}

// isBasicLandCard returns true if a card is a basic land (by name).
func isBasicLandCard(c Card) bool {
	switch c.Name() {
	case "Plains", "Island", "Swamp", "Mountain", "Forest":
		return true
	}
	return false
}
