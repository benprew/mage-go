package jumpstart

import . "git.sr.ht/~cdcarter/mage-go/pkg/mage"

func init() {
	registerArtifacts()
}

func registerArtifacts() {

// Aether Spellbomb {1}
// Artifact
// {U}, Sacrifice this artifact: Return target creature to its owner's hand.
// {1}, Sacrifice this artifact: Draw a card.
// TODO: implement
	Register("Aether Spellbomb", func() Card {
		return NewArtifact("Aether Spellbomb", "{1}")
	})


// Arcane Encyclopedia {3}
// Artifact — Book
// {3}, {T}: Draw a card.
// TODO: implement
	Register("Arcane Encyclopedia", func() Card {
		return NewArtifact("Arcane Encyclopedia", "{3}")
	})


// Bubbling Cauldron {2}
// Artifact
// {1}, {T}, Sacrifice a creature: You gain 4 life.
// {1}, {T}, Sacrifice a creature named Festering Newt: Each opponent loses 4 life. You gain life equal to the life lost this way.
// TODO: implement
	Register("Bubbling Cauldron", func() Card {
		return NewArtifact("Bubbling Cauldron", "{2}")
	})


// Chromatic Sphere {1}
// Artifact
// {1}, {T}, Sacrifice this artifact: Add one mana of any color. Draw a card.
// TODO: implement
	Register("Chromatic Sphere", func() Card {
		return NewArtifact("Chromatic Sphere", "{1}")
	})


// Dreamstone Hedron {6}
// Artifact
// {T}: Add {C}{C}{C}.
// {3}, {T}, Sacrifice this artifact: Draw three cards.
// TODO: implement
	Register("Dreamstone Hedron", func() Card {
		return NewArtifact("Dreamstone Hedron", "{6}")
	})


// Guardian Idol {2}
// Artifact
// This artifact enters tapped.
// {T}: Add {C}.
// {2}: This artifact becomes a 2/2 Golem artifact creature until end of turn.
// TODO: implement
	Register("Guardian Idol", func() Card {
		return NewArtifact("Guardian Idol", "{2}")
	})


// Hedron Archive {4}
// Artifact
// {T}: Add {C}{C}.
// {2}, {T}, Sacrifice this artifact: Draw two cards.
// TODO: implement
	Register("Hedron Archive", func() Card {
		return NewArtifact("Hedron Archive", "{4}")
	})


// Herald's Horn {3}
// Artifact
// As this artifact enters, choose a creature type.
// Creature spells you cast of the chosen type cost {1} less to cast.
// At the beginning of your upkeep, look at the top card of your library. If it's a creature card of the chosen type, you may reveal it and put it into your hand.
// TODO: implement
	Register("Herald's Horn", func() Card {
		return NewArtifact("Herald's Horn", "{3}")
	})


// Mana Geode {3}
// Artifact
// When this artifact enters, scry 1.
// {T}: Add one mana of any color.
// TODO: implement
	Register("Mana Geode", func() Card {
		return NewArtifact("Mana Geode", "{3}")
	})


// Marauder's Axe {2}
// Artifact — Equipment
// Equipped creature gets +2/+0.
// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
// TODO: implement
	Register("Marauder's Axe", func() Card {
		return NewArtifact("Marauder's Axe", "{2}")
	})


// Pirate's Cutlass {3}
// Artifact — Equipment
// When this Equipment enters, attach it to target Pirate you control.
// Equipped creature gets +2/+1.
// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
// TODO: implement
	Register("Pirate's Cutlass", func() Card {
		return NewArtifact("Pirate's Cutlass", "{3}")
	})


// Prophetic Prism {2}
// Artifact
// When this artifact enters, draw a card.
// {1}, {T}: Add one mana of any color.
// TODO: implement
	Register("Prophetic Prism", func() Card {
		return NewArtifact("Prophetic Prism", "{2}")
	})


// Rogue's Gloves {2}
// Artifact — Equipment
// Whenever equipped creature deals combat damage to a player, you may draw a card.
// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
// TODO: implement
	Register("Rogue's Gloves", func() Card {
		return NewArtifact("Rogue's Gloves", "{2}")
	})


// Scroll of Avacyn {1}
// Artifact
// {1}, Sacrifice this artifact: Draw a card. If you control an Angel, you gain 5 life.
// TODO: implement
	Register("Scroll of Avacyn", func() Card {
		return NewArtifact("Scroll of Avacyn", "{1}")
	})


// Terrarion {1}
// Artifact
// This artifact enters tapped.
// {2}, {T}, Sacrifice this artifact: Add two mana in any combination of colors.
// When this artifact is put into a graveyard from the battlefield, draw a card.
// TODO: implement
	Register("Terrarion", func() Card {
		return NewArtifact("Terrarion", "{1}")
	})


// Unstable Obelisk {3}
// Artifact
// {T}: Add {C}.
// {7}, {T}, Sacrifice this artifact: Destroy target permanent.
// TODO: implement
	Register("Unstable Obelisk", func() Card {
		return NewArtifact("Unstable Obelisk", "{3}")
	})


// Warmonger's Chariot {2}
// Artifact — Equipment
// Equipped creature gets +2/+2.
// As long as equipped creature has defender, it can attack as though it didn't have defender.
// Equip {3} ({3}: Attach to target creature you control. Equip only as a sorcery.)
// TODO: implement
	Register("Warmonger's Chariot", func() Card {
		return NewArtifact("Warmonger's Chariot", "{2}")
	})

}
