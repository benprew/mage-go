package legends

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerLands()
}

func registerLands() {

// Adventurers' Guildhouse 
// Land
// Green legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// XXX: "bands with other legendary creatures" approximated as Banding
	Register("Adventurers' Guildhouse", func() Card {
		return NewLand("Adventurers' Guildhouse",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, perm := range g.FilterBattlefield(And(IsCreature, HasColorFilter(Green), ControlledBy(src.Controller))) {
						if perm.Card.HasSuperType(SuperLegendary) {
							g.Effects.GrantAttr(perm.ID(), Banding)
						}
					}
					return nil
				}),
			),
		)
	})


// Cathedral of Serra 
// Land
// White legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// XXX: "bands with other legendary creatures" approximated as Banding
	Register("Cathedral of Serra", func() Card {
		return NewLand("Cathedral of Serra",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, perm := range g.FilterBattlefield(And(IsCreature, HasColorFilter(White), ControlledBy(src.Controller))) {
						if perm.Card.HasSuperType(SuperLegendary) {
							g.Effects.GrantAttr(perm.ID(), Banding)
						}
					}
					return nil
				}),
			),
		)
	})


// Hammerheim
// Legendary Land
// {T}: Add {R}.
// {T}: Target creature loses all landwalk abilities until end of turn.
	Register("Hammerheim", func() Card {
		return NewLand("Hammerheim",
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Red),
			WithActivatedAbility(
				FuncEffect(
					"target creature loses all landwalk abilities until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						for kw := range LandwalkAttrs() {
							eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
								g.Effects.RevokeAttr(target.ID(), kw)
								return nil
							})
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
						}
						return nil
					},
				),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})


// Karakas
// Legendary Land
// {T}: Add {W}.
// {T}: Return target legendary creature to its owner's hand.
	Register("Karakas", func() Card {
		return NewLand("Karakas",
			WithSuperTypes(SuperLegendary),
			WithManaAbility(White),
			WithActivatedAbility(
				ReturnToHandTarget(),
				TapSourceCost(),
				WithTarget(TargetCreature(NewPermanentFilter("legendary", func(p *Permanent, _ *Game) bool {
					return p.Card.HasSuperType(SuperLegendary)
				}))),
			),
		)
	})


// Mountain Stronghold 
// Land
// Red legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// XXX: "bands with other legendary creatures" approximated as Banding
	Register("Mountain Stronghold", func() Card {
		return NewLand("Mountain Stronghold",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, perm := range g.FilterBattlefield(And(IsCreature, HasColorFilter(Red), ControlledBy(src.Controller))) {
						if perm.Card.HasSuperType(SuperLegendary) {
							g.Effects.GrantAttr(perm.ID(), Banding)
						}
					}
					return nil
				}),
			),
		)
	})


// Pendelhaven
// Legendary Land
// {T}: Add {G}.
// {T}: Target 1/1 creature gets +1/+2 until end of turn.
	Register("Pendelhaven", func() Card {
		return NewLand("Pendelhaven",
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Green),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(2), SelectTarget),
				TapSourceCost(),
				WithTarget(TargetCreature(NewPermanentFilter("1/1", func(p *Permanent, g *Game) bool {
					return p.CurrentPower(g) == 1 && p.CurrentToughness(g) == 1
				}))),
			),
		)
	})


// Seafarer's Quay 
// Land
// Blue legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// XXX: "bands with other legendary creatures" approximated as Banding
	Register("Seafarer's Quay", func() Card {
		return NewLand("Seafarer's Quay",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, perm := range g.FilterBattlefield(And(IsCreature, HasColorFilter(Blue), ControlledBy(src.Controller))) {
						if perm.Card.HasSuperType(SuperLegendary) {
							g.Effects.GrantAttr(perm.ID(), Banding)
						}
					}
					return nil
				}),
			),
		)
	})


// The Tabernacle at Pendrell Vale
// Legendary Land
// All creatures have "At the beginning of your upkeep, destroy this creature unless you pay {1}."
	Register("The Tabernacle at Pendrell Vale", func() Card {
		return NewLand("The Tabernacle at Pendrell Vale",
			WithSuperTypes(SuperLegendary),
			WithAbility(NewTriggered(EvtUpkeep, false, FuncEffect(
				"destroy each creature unless its controller pays {1}",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					activePlayer := g.ActivePlayerObj()
					if activePlayer == nil {
						return nil
					}
					activeID := activePlayer.PlayerID()
					// Collect creatures first to avoid modification during iteration
					var creatures []*Permanent
					for _, p := range g.FilterBattlefield(NewPermanentFilter("creature", func(p *Permanent, _ *Game) bool {
						return p.HasType(TypeCreature) && p.Controller == activeID
					})) {
						creatures = append(creatures, p)
					}
					for _, p := range creatures {
						if !g.TryPayCostFromLands(activeID, "{1}") {
							g.DestroyPermanent(p)
						}
					}
					return nil
				},
			))),
		)
	})


// Tolaria
// Legendary Land
// {T}: Add {U}.
// {T}: Target creature loses banding and all "bands with other" abilities until end of turn. Activate only during any upkeep step.
	Register("Tolaria", func() Card {
		return NewLand("Tolaria",
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Blue),
			WithActivatedAbility(
				FuncEffect(
					"target creature loses banding until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
							g.Effects.RevokeAttr(target.ID(), Banding)
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						return nil
					},
				),
				TapSourceCost(),
				WithTarget(TargetCreature()),
				WithUpkeepOnly(),
			),
		)
	})


// Unholy Citadel 
// Land
// Black legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// XXX: "bands with other legendary creatures" approximated as Banding
	Register("Unholy Citadel", func() Card {
		return NewLand("Unholy Citadel",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, perm := range g.FilterBattlefield(And(IsCreature, HasColorFilter(Black), ControlledBy(src.Controller))) {
						if perm.Card.HasSuperType(SuperLegendary) {
							g.Effects.GrantAttr(perm.ID(), Banding)
						}
					}
					return nil
				}),
			),
		)
	})


// Urborg
// Legendary Land
// {T}: Add {B}.
// {T}: Target creature loses first strike or swampwalk until end of turn.
	Register("Urborg", func() Card {
		return NewLand("Urborg",
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Black),
			WithActivatedAbility(
				FuncEffect(
					"target creature loses first strike or swampwalk until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						// Oracle says "first strike or swampwalk" — controller chooses which to remove
						p := g.GetPlayer(controller)
						mode := p.ChooseMode([]string{"First strike", "Swampwalk"}, "Urborg")
						if mode == 0 {
							// Remove first strike
							eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
								g.Effects.RevokeAttr(target.ID(), FirstStrike)
								return nil
							})
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
						} else {
							// Remove swampwalk
							eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
								g.Effects.RevokeAttr(target.ID(), Swampwalk)
								return nil
							})
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
						}
						return nil
					},
				),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})

}
