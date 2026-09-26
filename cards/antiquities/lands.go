package antiquities

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerLands()
}

func registerLands() {
	// Mishra's Factory
	// Land
	// {T}: Add {C}.
	// {1}: This land becomes a 2/2 Assembly-Worker artifact creature until end of turn. It's still a land.
	// {T}: Target Assembly-Worker creature gets +1/+1 until end of turn.
	Register("Mishra's Factory", func() Card {
		creature := TargetPermanent("creature", And(IsCreature, HasSubType("Assembly-Worker")))
		return NewLand("Mishra's Factory",
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithActivatedAbility(
				// TODO: convert to pipeline — needs "animate" + "add type/subtype until EOT" steps
				FuncEffect("become 2/2 Assembly-Worker artifact creature until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						// Animate as 2/2 artifact creature
						animEff := TemporaryAnimate(sourceID, 2, 2)
						animEff.SetSourceID(sourceID)
						g.AddContinuousEffect(animEff)
						// Add Artifact type and Assembly-Worker subtype until end of turn
						subEff := TargetEffect(LayerType, EndOfTurn, sourceID, func(g2 *Game, target *Permanent) error {
							target.Card.AddType(TypeArtifact)
							target.Card.AddSubType("Assembly-Worker")
							return nil
						})
						subEff.SetSourceID(sourceID)
						g.AddContinuousEffect(subEff)
						g.ApplyContinuousEffects()
						return nil
					}),
				GenericCost(1),
			),
			WithAbility(Activate(Costs(TapCost(This)),
				Until(EndOfTurn, ModifyPT(creature, Fixed(1), Fixed(1))), WithTargets(creature))),
		)
	})

	// Mishra's Workshop
	// Land
	// {T}: Add {C}{C}{C}. Spend this mana only to cast artifact spells.
	Register("Mishra's Workshop", func() Card {
		return NewLand("Mishra's Workshop",
			WithMultiManaAbility(ManaProduction{
				Color:       Colorless,
				Amount:      3,
				Restriction: ArtifactSpellsOnly{},
			}),
		)
	})

	// Strip Mine
	// Land
	// {T}: Add {C}.
	// {T}, Sacrifice this land: Destroy target land.
	Register("Strip Mine", func() Card {
		land := TargetPermanent("land", IsLand)
		return NewLand("Strip Mine",
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithAbility(Activate(Costs(TapCost(This), SacrificeSourceCost()),
				Destroy(land), WithTargets(land))),
		)
	})

	// Urza's Mine
	// Land — Urza's Mine
	// {T}: Add {C}. If you control an Urza's Power-Plant and an Urza's Tower, add {C}{C} instead.
	Register("Urza's Mine", func() Card {
		return NewLand("Urza's Mine",
			WithSubTypes("Urza's", "Mine"),
			WithAbility(Activate(Costs(TapCost(This)),
				AddManaAmount(Colorless, urzaLandManaAmount("Power-Plant", "Tower", 1)))),
		)
	})

	// Urza's Power Plant
	// Land — Urza's Power-Plant
	// {T}: Add {C}. If you control an Urza's Mine and an Urza's Tower, add {C}{C} instead.
	Register("Urza's Power Plant", func() Card {
		return NewLand("Urza's Power Plant",
			WithSubTypes("Urza's", "Power-Plant"),
			WithAbility(Activate(Costs(TapCost(This)),
				AddManaAmount(Colorless, urzaLandManaAmount("Mine", "Tower", 1)))),
		)
	})

	// Urza's Tower
	// Land — Urza's Tower
	// {T}: Add {C}. If you control an Urza's Mine and an Urza's Power-Plant, add {C}{C}{C} instead.
	Register("Urza's Tower", func() Card {
		return NewLand("Urza's Tower",
			WithSubTypes("Urza's", "Tower"),
			WithAbility(Activate(Costs(TapCost(This)),
				AddManaAmount(Colorless, urzaLandManaAmount("Mine", "Power-Plant", 2)))),
		)
	})
}

func urzaLandManaAmount(first, second string, extra int) ResolutionValue {
	pair := MinValue(Fixed(1), MinValue(
		CountPermanents(ControlledPermanents(And(HasSubType("Urza's"), HasSubType(first)))),
		CountPermanents(ControlledPermanents(And(HasSubType("Urza's"), HasSubType(second)))),
	))
	// The DSL has subtraction but no addition operation.
	return SubtractValues(MultiplyValues(Fixed(extra), pair), Fixed(-1))
}
