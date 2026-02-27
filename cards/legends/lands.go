package legends

import (
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
// TODO: implement
	Register("Adventurers' Guildhouse", withExpansion(func() Card {
		return NewLand("Adventurers' Guildhouse")
	}))


// Cathedral of Serra 
// Land
// White legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// TODO: implement
	Register("Cathedral of Serra", withExpansion(func() Card {
		return NewLand("Cathedral of Serra")
	}))


// Hammerheim 
// Legendary Land
// {T}: Add {R}.
// {T}: Target creature loses all landwalk abilities until end of turn.
// TODO: implement
	Register("Hammerheim", withExpansion(func() Card {
		return NewLand("Hammerheim",
			WithSuperTypes(SuperLegendary),
		)
	}))


// Karakas
// Legendary Land
// {T}: Add {W}.
// {T}: Return target legendary creature to its owner's hand.
	Register("Karakas", withExpansion(func() Card {
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
	}))


// Mountain Stronghold 
// Land
// Red legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// TODO: implement
	Register("Mountain Stronghold", withExpansion(func() Card {
		return NewLand("Mountain Stronghold")
	}))


// Pendelhaven
// Legendary Land
// {T}: Add {G}.
// {T}: Target 1/1 creature gets +1/+2 until end of turn.
	Register("Pendelhaven", withExpansion(func() Card {
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
	}))


// Seafarer's Quay 
// Land
// Blue legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// TODO: implement
	Register("Seafarer's Quay", withExpansion(func() Card {
		return NewLand("Seafarer's Quay")
	}))


// The Tabernacle at Pendrell Vale 
// Legendary Land
// All creatures have "At the beginning of your upkeep, destroy this creature unless you pay {1}."
// TODO: implement
	Register("The Tabernacle at Pendrell Vale", withExpansion(func() Card {
		return NewLand("The Tabernacle at Pendrell Vale",
			WithSuperTypes(SuperLegendary),
		)
	}))


// Tolaria 
// Legendary Land
// {T}: Add {U}.
// {T}: Target creature loses banding and all "bands with other" abilities until end of turn. Activate only during any upkeep step.
// TODO: implement
	Register("Tolaria", withExpansion(func() Card {
		return NewLand("Tolaria",
			WithSuperTypes(SuperLegendary),
		)
	}))


// Unholy Citadel 
// Land
// Black legendary creatures you control have "bands with other legendary creatures." (Any legendary creatures can attack in a band as long as at least one has "bands with other legendary creatures." Bands are blocked as a group. If at least two legendary creatures you control, one of which has "bands with other legendary creatures," are blocking or being blocked by the same creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// TODO: implement
	Register("Unholy Citadel", withExpansion(func() Card {
		return NewLand("Unholy Citadel")
	}))


// Urborg 
// Legendary Land
// {T}: Add {B}.
// {T}: Target creature loses first strike or swampwalk until end of turn.
// TODO: implement
	Register("Urborg", withExpansion(func() Card {
		return NewLand("Urborg",
			WithSuperTypes(SuperLegendary),
		)
	}))

}
