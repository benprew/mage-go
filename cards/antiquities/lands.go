package antiquities

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerLands()
}

func registerLands() {
	// Mishra's Factory
	// Land
	// {T}: Add {C}.
	// {1}: Mishra's Factory becomes a 2/2 Assembly-Worker artifact creature until end of turn.
	// It's still a land.
	// {T}: Target Assembly-Worker creature gets +1/+1 until end of turn.
	Register("Mishra's Factory", func() Card {
		return NewLand("Mishra's Factory",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				FuncEffect("become 2/2 Assembly-Worker artifact creature until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(1), SelectTarget),
				TapSourceCost(),
				WithTarget(TargetCreature(HasSubType("Assembly-Worker"))),
			),
		)
	})

	// Mishra's Workshop
	// Land
	// {T}: Add {C}{C}{C}. Spend this mana only to cast artifact spells.
	Register("Mishra's Workshop", func() Card {
		return NewLand("Mishra's Workshop",
			WithActivatedAbility(
				FuncEffect("add {C}{C}{C} for artifacts only",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p != nil {
							p.ManaPool().Add(Colorless, 3)
							g.SetArtifactManaOnly(controller)
						}
						return nil
					}),
				TapSourceCost(),
			),
		)
	})

	// Strip Mine
	// Land
	// {T}: Add {C}.
	// {T}, Sacrifice Strip Mine: Destroy target land.
	Register("Strip Mine", func() Card {
		return NewLand("Strip Mine",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				FuncEffect("destroy target land",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm != nil {
							g.DestroyPermanent(perm)
						}
						return nil
					}),
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetPermanent(IsLand)),
			),
		)
	})

	// Urza's Mine
	// Land — Urza's Mine
	// {T}: Add {C}. If you control an Urza's Power-Plant and an Urza's Tower, add {C}{C} instead.
	Register("Urza's Mine", func() Card {
		return NewLand("Urza's Mine",
			WithSubTypes("Urza's", "Mine"),
			WithActivatedAbility(
				FuncEffect("add {C} or {C}{C} with Tron",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						hasPowerPlant := g.AnyBattlefield(And(ControlledBy(controller), HasSubType("Power-Plant")))
						hasTower := g.AnyBattlefield(And(ControlledBy(controller), HasSubType("Tower")))
						if hasPowerPlant && hasTower {
							p.ManaPool().Add(Colorless, 2)
						} else {
							p.ManaPool().Add(Colorless, 1)
						}
						return nil
					}),
				TapSourceCost(),
			),
		)
	})

	// Urza's Power Plant
	// Land — Urza's Power-Plant
	// {T}: Add {C}. If you control an Urza's Mine and an Urza's Tower, add {C}{C} instead.
	Register("Urza's Power Plant", func() Card {
		return NewLand("Urza's Power Plant",
			WithSubTypes("Urza's", "Power-Plant"),
			WithActivatedAbility(
				FuncEffect("add {C} or {C}{C} with Tron",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						hasMine := g.AnyBattlefield(And(ControlledBy(controller), HasSubType("Mine")))
						hasTower := g.AnyBattlefield(And(ControlledBy(controller), HasSubType("Tower")))
						if hasMine && hasTower {
							p.ManaPool().Add(Colorless, 2)
						} else {
							p.ManaPool().Add(Colorless, 1)
						}
						return nil
					}),
				TapSourceCost(),
			),
		)
	})

	// Urza's Tower
	// Land — Urza's Tower
	// {T}: Add {C}. If you control an Urza's Mine and an Urza's Power-Plant, add {C}{C}{C} instead.
	Register("Urza's Tower", func() Card {
		return NewLand("Urza's Tower",
			WithSubTypes("Urza's", "Tower"),
			WithActivatedAbility(
				FuncEffect("add {C} or {C}{C}{C} with Tron",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						hasMine := g.AnyBattlefield(And(ControlledBy(controller), HasSubType("Mine")))
						hasPowerPlant := g.AnyBattlefield(And(ControlledBy(controller), HasSubType("Power-Plant")))
						if hasMine && hasPowerPlant {
							p.ManaPool().Add(Colorless, 3)
						} else {
							p.ManaPool().Add(Colorless, 1)
						}
						return nil
					}),
				TapSourceCost(),
			),
		)
	})
}
