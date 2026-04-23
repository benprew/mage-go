package legends

import (
	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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
				GrantKeywordToControlled(Banding, And(HasColorFilter(Green), IsLegendary)),
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
				GrantKeywordToControlled(Banding, And(HasColorFilter(White), IsLegendary)),
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
			// TODO: convert to pipeline — needs RevokeAttrUntilEndOfTurn step (iterate LandwalkAttrs)
			WithActivatedAbility(
				FuncEffect(
					"target creature loses all landwalk abilities until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						for kw := range LandwalkAttrs() {
							eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
								g.RevokeAttr(target.ID(), kw)
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
				GrantKeywordToControlled(Banding, And(HasColorFilter(Red), IsLegendary)),
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
				Boost(Fixed(1), Fixed(2)),
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
				GrantKeywordToControlled(Banding, And(HasColorFilter(Blue), IsLegendary)),
			),
		)
	})


// The Tabernacle at Pendrell Vale
// Legendary Land
// All creatures have "At the beginning of your upkeep, destroy this creature unless you pay {1}."
	Register("The Tabernacle at Pendrell Vale", func() Card {
		return NewLand("The Tabernacle at Pendrell Vale",
			WithSuperTypes(SuperLegendary),
			WithStaticAbility(
				GrantTriggeredAbilityToAll(
					EvtUpkeep, false,
					EventPlayerIsController{},
					IsCreature,
					Pipeline("destroy this creature unless you pay {1}",
						EffectProperties{Outcome: OutcomeDetriment},
						SnapshotPermanent(SelectSource, "self"),
						IfElse("pay {1} or be destroyed",
							&NotCond{Inner: &TryPayManaCond{Cost: "{1}"}},
							DestroyGathered("self"),
							nil),
					),
				),
			),
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
				Pipeline("target creature loses banding until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					RevokeKeywordFromTargetUntilEOT(Banding),
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
				GrantKeywordToControlled(Banding, And(HasColorFilter(Black), IsLegendary)),
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
			// TODO: convert to pipeline — needs ModalEffect + RevokeAttrUntilEndOfTurn step
			WithActivatedAbility(
				FuncEffect(
					"target creature loses first strike or swampwalk until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						// Oracle says "first strike or swampwalk" — controller chooses which to remove
						p := g.GetPlayer(controller)
						mode := p.ChooseMode([]string{"First strike", "Swampwalk"}, "Urborg")
						if mode == 0 {
							// Remove first strike
							eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
								g.RevokeAttr(target.ID(), FirstStrike)
								return nil
							})
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
						} else {
							// Remove swampwalk
							eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
								g.RevokeAttr(target.ID(), Swampwalk)
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
