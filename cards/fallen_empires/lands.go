package fallen_empires

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerLands()
}

// sacLandFactory creates a sacrifice land that enters tapped, taps for one mana
// of the given color, and can be sacrificed (with tap) to add two mana of that color.
func sacLandFactory(name string, color Color) func() Card {
	return withExpansion(func() Card {
		return NewLand(name,
			WithKeyword(EntersTapped),
			WithManaAbility(color),
			WithActivatedAbility(
				AddMana(color, 2),
				Tap(),
				WithCost(SacrificeSourceCost()),
			),
		)
	})
}

// storageLandFactory creates a storage land that enters tapped, may choose not to untap,
// gains storage counters each upkeep while tapped, and taps to remove all storage counters
// for mana of the given color.
func storageLandFactory(name string, color Color) func() Card {
	return withExpansion(func() Card {
		return NewLand(name,
			WithKeyword(EntersTapped),
			WithKeyword(AttrMayNotUntap),
			// At the beginning of your upkeep, if this land is tapped, put a storage counter on it.
			// TODO: convert to pipeline — needs conditional "if source is tapped" check
			WithAbility(
				BeginningOfUpkeepTrigger(
					FuncEffect("add storage counter if tapped",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.MutablePermanent(sourceID)
							if perm == nil || !perm.Tapped {
								return nil
							}
							perm.AddCounter(Storage, 1)
							return nil
						}),
					false, // not optional
				),
			),
			// {T}, Remove any number of storage counters: Add {color} for each counter removed.
			// TODO: convert to pipeline — needs RemoveAllCounters + AddManaFromVar steps
			WithActivatedAbility(
				FuncEffect("add mana for each storage counter removed",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.MutablePermanent(sourceID)
						if perm == nil {
							return nil
						}
						count := int(perm.Counters[Storage])
						if count > 0 {
							perm.RemoveCounter(Storage, count)
							p := g.GetPlayer(controller)
							if p != nil {
								p.ManaPool().Add(color, count)
							}
						}
						return nil
					}),
				Tap(),
			),
		)
	})
}

func registerLands() {

	// Bottomless Vault
	// Land
	// This land enters tapped.
	// You may choose not to untap this land during your untap step.
	// At the beginning of your upkeep, if this land is tapped, put a storage counter on it.
	// {T}, Remove any number of storage counters from this land: Add {B} for each storage counter removed this way.
	Register("Bottomless Vault", storageLandFactory("Bottomless Vault", Black))

	// Dwarven Hold
	// Land
	// This land enters tapped.
	// You may choose not to untap this land during your untap step.
	// At the beginning of your upkeep, if this land is tapped, put a storage counter on it.
	// {T}, Remove any number of storage counters from this land: Add {R} for each storage counter removed this way.
	Register("Dwarven Hold", storageLandFactory("Dwarven Hold", Red))

	// Dwarven Ruins
	// Land
	// This land enters tapped.
	// {T}: Add {R}.
	// {T}, Sacrifice this land: Add {R}{R}.
	Register("Dwarven Ruins", sacLandFactory("Dwarven Ruins", Red))

	// Ebon Stronghold
	// Land
	// This land enters tapped.
	// {T}: Add {B}.
	// {T}, Sacrifice this land: Add {B}{B}.
	Register("Ebon Stronghold", sacLandFactory("Ebon Stronghold", Black))

	// Havenwood Battleground
	// Land
	// This land enters tapped.
	// {T}: Add {G}.
	// {T}, Sacrifice this land: Add {G}{G}.
	Register("Havenwood Battleground", sacLandFactory("Havenwood Battleground", Green))

	// Hollow Trees
	// Land
	// This land enters tapped.
	// You may choose not to untap this land during your untap step.
	// At the beginning of your upkeep, if this land is tapped, put a storage counter on it.
	// {T}, Remove any number of storage counters from this land: Add {G} for each storage counter removed this way.
	Register("Hollow Trees", storageLandFactory("Hollow Trees", Green))

	// Icatian Store
	// Land
	// This land enters tapped.
	// You may choose not to untap this land during your untap step.
	// At the beginning of your upkeep, if this land is tapped, put a storage counter on it.
	// {T}, Remove any number of storage counters from this land: Add {W} for each storage counter removed this way.
	Register("Icatian Store", storageLandFactory("Icatian Store", White))

	// Rainbow Vale
	// Land
	// {T}: Add one mana of any color. An opponent gains control of this land at the beginning of the next end step.
	// TODO: implement
	Register("Rainbow Vale", withExpansion(func() Card {
		return NewLand("Rainbow Vale")
	}))

	// Ruins of Trokair
	// Land
	// This land enters tapped.
	// {T}: Add {W}.
	// {T}, Sacrifice this land: Add {W}{W}.
	Register("Ruins of Trokair", sacLandFactory("Ruins of Trokair", White))

	// Sand Silos
	// Land
	// This land enters tapped.
	// You may choose not to untap this land during your untap step.
	// At the beginning of your upkeep, if this land is tapped, put a storage counter on it.
	// {T}, Remove any number of storage counters from this land: Add {U} for each storage counter removed this way.
	Register("Sand Silos", storageLandFactory("Sand Silos", Blue))

	// Svyelunite Temple
	// Land
	// This land enters tapped.
	// {T}: Add {U}.
	// {T}, Sacrifice this land: Add {U}{U}.
	Register("Svyelunite Temple", sacLandFactory("Svyelunite Temple", Blue))
}
