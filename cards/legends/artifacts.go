package legends

import . "github.com/mage/mage/pkg/mage"

func init() {
	registerArtifacts()
}

func registerArtifacts() {

// Al-abara's Carpet {5}
// Artifact
// {5}, {T}: Prevent all damage that would be dealt to you this turn by attacking creatures without flying.
// TODO: implement
	Register("Al-abara's Carpet", withExpansion(func() Card {
		return NewArtifact("Al-abara's Carpet", "{5}")
	}))


// Alchor's Tomb {4}
// Artifact
// {2}, {T}: Target permanent you control becomes the color of your choice. (This effect lasts indefinitely.)
// TODO: implement
	Register("Alchor's Tomb", withExpansion(func() Card {
		return NewArtifact("Alchor's Tomb", "{4}")
	}))


// Arena of the Ancients {3}
// Artifact
// Legendary creatures don't untap during their controllers' untap steps.
// When this artifact enters, tap all legendary creatures.
// TODO: implement
	Register("Arena of the Ancients", withExpansion(func() Card {
		return NewArtifact("Arena of the Ancients", "{3}")
	}))


// Black Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {B}, then add an additional {B} for each charge counter removed this way.
// TODO: implement
	Register("Black Mana Battery", withExpansion(func() Card {
		return NewArtifact("Black Mana Battery", "{4}")
	}))


// Blue Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {U}, then add an additional {U} for each charge counter removed this way.
// TODO: implement
	Register("Blue Mana Battery", withExpansion(func() Card {
		return NewArtifact("Blue Mana Battery", "{4}")
	}))


// Forethought Amulet {5}
// Artifact
// At the beginning of your upkeep, sacrifice this artifact unless you pay {3}.
// If an instant or sorcery source would deal 3 or more damage to you, it deals 2 damage to you instead.
// TODO: implement
	Register("Forethought Amulet", withExpansion(func() Card {
		return NewArtifact("Forethought Amulet", "{5}")
	}))


// Gauntlets of Chaos {5}
// Artifact
// {5}, Sacrifice this artifact: Exchange control of target artifact, creature, or land you control and target permanent an opponent controls that shares one of those types with it. If those permanents are exchanged this way, destroy all Auras attached to them.
// TODO: implement
	Register("Gauntlets of Chaos", withExpansion(func() Card {
		return NewArtifact("Gauntlets of Chaos", "{5}")
	}))


// Green Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {G}, then add an additional {G} for each charge counter removed this way.
// TODO: implement
	Register("Green Mana Battery", withExpansion(func() Card {
		return NewArtifact("Green Mana Battery", "{4}")
	}))


// Horn of Deafening {4}
// Artifact
// {2}, {T}: Prevent all combat damage that would be dealt by target creature this turn.
// TODO: implement
	Register("Horn of Deafening", withExpansion(func() Card {
		return NewArtifact("Horn of Deafening", "{4}")
	}))


// Knowledge Vault {4}
// Artifact
// {2}, {T}: Exile the top card of your library face down.
// {0}: Sacrifice this artifact. If you do, discard your hand, then put all cards exiled with this artifact into their owner's hand.
// When this artifact leaves the battlefield, put all cards exiled with it into their owner's graveyard.
// TODO: implement
	Register("Knowledge Vault", withExpansion(func() Card {
		return NewArtifact("Knowledge Vault", "{4}")
	}))


// Kry Shield {2}
// Artifact
// {2}, {T}: Prevent all damage that would be dealt this turn by target creature you control. That creature gets +0/+X until end of turn, where X is its mana value.
// TODO: implement
	Register("Kry Shield", withExpansion(func() Card {
		return NewArtifact("Kry Shield", "{2}")
	}))


// Life Chisel {4}
// Artifact
// Sacrifice a creature: You gain life equal to the sacrificed creature's toughness. Activate only during your upkeep.
// TODO: implement
	Register("Life Chisel", withExpansion(func() Card {
		return NewArtifact("Life Chisel", "{4}")
	}))


// Life Matrix {4}
// Artifact
// {4}, {T}: Put a matrix counter on target creature and that creature gains "Remove a matrix counter from this creature: Regenerate this creature." Activate only during your upkeep.
// TODO: implement
	Register("Life Matrix", withExpansion(func() Card {
		return NewArtifact("Life Matrix", "{4}")
	}))


// Mana Matrix {6}
// Artifact
// Instant and enchantment spells you cast cost {2} less to cast.
// TODO: implement
	Register("Mana Matrix", withExpansion(func() Card {
		return NewArtifact("Mana Matrix", "{6}")
	}))


// Mirror Universe {6}
// Artifact
// {T}, Sacrifice this artifact: Exchange life totals with target opponent. Activate only during your upkeep.
// TODO: implement
	Register("Mirror Universe", withExpansion(func() Card {
		return NewArtifact("Mirror Universe", "{6}")
	}))


// North Star {4}
// Artifact
// {4}, {T}: For one spell this turn, you may spend mana as though it were mana of any type to pay that spell's mana cost. (Additional costs are still paid normally.)
// TODO: implement
	Register("North Star", withExpansion(func() Card {
		return NewArtifact("North Star", "{4}")
	}))


// Nova Pentacle {4}
// Artifact
// {3}, {T}: The next time a source of your choice would deal damage to you this turn, that damage is dealt to target creature of an opponent's choice instead.
// TODO: implement
	Register("Nova Pentacle", withExpansion(func() Card {
		return NewArtifact("Nova Pentacle", "{4}")
	}))


// Planar Gate {6}
// Artifact
// Creature spells you cast cost {2} less to cast.
// TODO: implement
	Register("Planar Gate", withExpansion(func() Card {
		return NewArtifact("Planar Gate", "{6}")
	}))


// Red Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {R}, then add an additional {R} for each charge counter removed this way.
// TODO: implement
	Register("Red Mana Battery", withExpansion(func() Card {
		return NewArtifact("Red Mana Battery", "{4}")
	}))


// Relic Barrier {2}
// Artifact
// {T}: Tap target artifact.
// TODO: implement
	Register("Relic Barrier", withExpansion(func() Card {
		return NewArtifact("Relic Barrier", "{2}")
	}))


// Ring of Immortals {5}
// Artifact
// {3}, {T}: Counter target instant or Aura spell that targets a permanent you control.
// TODO: implement
	Register("Ring of Immortals", withExpansion(func() Card {
		return NewArtifact("Ring of Immortals", "{5}")
	}))


// Serpent Generator {6}
// Artifact
// {4}, {T}: Create a 1/1 colorless Snake artifact creature token. It has "Whenever this creature deals damage to a player, that player gets a poison counter." (A player with ten or more poison counters loses the game.)
// TODO: implement
	Register("Serpent Generator", withExpansion(func() Card {
		return NewArtifact("Serpent Generator", "{6}")
	}))


// Sword of the Ages {6}
// Artifact
// This artifact enters tapped.
// {T}, Sacrifice this artifact and any number of creatures you control: This artifact deals X damage to any target, where X is the total power of the creatures sacrificed this way, then exile this artifact and those creature cards.
// TODO: implement
	Register("Sword of the Ages", withExpansion(func() Card {
		return NewArtifact("Sword of the Ages", "{6}")
	}))


// Triassic Egg {4}
// Artifact
// {3}, {T}: Put a hatchling counter on this artifact.
// Sacrifice this artifact: Choose one. Activate only if there are two or more hatchling counters on this artifact.
// • You may put a creature card from your hand onto the battlefield.
// • Return target creature card from your graveyard to the battlefield.
// TODO: implement
	Register("Triassic Egg", withExpansion(func() Card {
		return NewArtifact("Triassic Egg", "{4}")
	}))


// Voodoo Doll {6}
// Artifact
// At the beginning of your upkeep, put a pin counter on this artifact.
// At the beginning of your end step, if this artifact is untapped, destroy this artifact and it deals damage to you equal to the number of pin counters on it.
// {X}{X}, {T}: This artifact deals damage equal to the number of pin counters on it to any target. X is the number of pin counters on this artifact.
// TODO: implement
	Register("Voodoo Doll", withExpansion(func() Card {
		return NewArtifact("Voodoo Doll", "{6}")
	}))


// White Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {W}, then add an additional {W} for each charge counter removed this way.
// TODO: implement
	Register("White Mana Battery", withExpansion(func() Card {
		return NewArtifact("White Mana Battery", "{4}")
	}))

}
