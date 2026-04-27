package jumpstart

import . "git.sr.ht/~cdcarter/mage-go/pkg/mage"

func init() {
	registerEnchantments()
}

func registerEnchantments() {

// Assault Formation {1}{G}
// Enchantment
// Each creature you control assigns combat damage equal to its toughness rather than its power.
// {G}: Target creature with defender can attack this turn as though it didn't have defender.
// {2}{G}: Creatures you control get +0/+1 until end of turn.
// TODO: implement
	Register("Assault Formation", func() Card {
		return NewEnchantment("Assault Formation", "{1}{G}")
	})


// Barrage of Expendables {R}
// Enchantment
// {R}, Sacrifice a creature: This enchantment deals 1 damage to any target.
// TODO: implement
	Register("Barrage of Expendables", func() Card {
		return NewEnchantment("Barrage of Expendables", "{R}")
	})


// Black Market {3}{B}{B}
// Enchantment
// Whenever a creature dies, put a charge counter on this enchantment.
// At the beginning of your first main phase, add {B} for each charge counter on this enchantment.
// TODO: implement
	Register("Black Market", func() Card {
		return NewEnchantment("Black Market", "{3}{B}{B}")
	})


// Blessed Sanctuary {3}{W}{W}
// Enchantment
// Prevent all noncombat damage that would be dealt to you and creatures you control.
// Whenever a nontoken creature you control enters, create a 2/2 white Unicorn creature token.
// TODO: implement
	Register("Blessed Sanctuary", func() Card {
		return NewEnchantment("Blessed Sanctuary", "{3}{W}{W}")
	})


// Branching Evolution {2}{G}
// Enchantment
// If one or more +1/+1 counters would be put on a creature you control, twice that many +1/+1 counters are put on that creature instead.
// TODO: implement
	Register("Branching Evolution", func() Card {
		return NewEnchantment("Branching Evolution", "{2}{G}")
	})


// Cathars' Crusade {3}{W}{W}
// Enchantment
// Whenever a creature you control enters, put a +1/+1 counter on each creature you control.
// TODO: implement
	Register("Cathars' Crusade", func() Card {
		return NewEnchantment("Cathars' Crusade", "{3}{W}{W}")
	})


// Celestial Mantle {3}{W}{W}{W}
// Enchantment — Aura
// Enchant creature
// Enchanted creature gets +3/+3.
// Whenever enchanted creature deals combat damage to a player, double its controller's life total.
// TODO: implement
	Register("Celestial Mantle", func() Card {
		return NewAura("Celestial Mantle", "{3}{W}{W}{W}")
	})


// Coastal Piracy {2}{U}{U}
// Enchantment
// Whenever a creature you control deals combat damage to an opponent, you may draw a card.
// TODO: implement
	Register("Coastal Piracy", func() Card {
		return NewEnchantment("Coastal Piracy", "{2}{U}{U}")
	})


// Cradle of Vitality {3}{W}
// Enchantment
// Whenever you gain life, you may pay {1}{W}. If you do, put a +1/+1 counter on target creature for each 1 life you gained.
// TODO: implement
	Register("Cradle of Vitality", func() Card {
		return NewEnchantment("Cradle of Vitality", "{3}{W}")
	})


// Curiosity {U}
// Enchantment — Aura
// Enchant creature
// Whenever enchanted creature deals damage to an opponent, you may draw a card.
// TODO: implement
	Register("Curiosity", func() Card {
		return NewAura("Curiosity", "{U}")
	})


// Curious Obsession {U}
// Enchantment — Aura
// Enchant creature
// Enchanted creature gets +1/+1 and has "Whenever this creature deals combat damage to a player, you may draw a card."
// At the beginning of your end step, if you didn't attack with a creature this turn, sacrifice this Aura.
// TODO: implement
	Register("Curious Obsession", func() Card {
		return NewAura("Curious Obsession", "{U}")
	})


// Death's Approach {B}
// Enchantment — Aura
// Enchant creature
// Enchanted creature gets -X/-X, where X is the number of creature cards in its controller's graveyard.
// TODO: implement
	Register("Death's Approach", func() Card {
		return NewAura("Death's Approach", "{B}")
	})


// Duelist's Heritage {2}{W}
// Enchantment
// Whenever one or more creatures attack, you may have target attacking creature gain double strike until end of turn.
// TODO: implement
	Register("Duelist's Heritage", func() Card {
		return NewEnchantment("Duelist's Heritage", "{2}{W}")
	})


// Eternal Thirst {1}{B}
// Enchantment — Aura
// Enchant creature
// Enchanted creature has lifelink and "Whenever a creature an opponent controls dies, put a +1/+1 counter on this creature." (Damage dealt by a creature with lifelink also causes its controller to gain that much life.)
// TODO: implement
	Register("Eternal Thirst", func() Card {
		return NewAura("Eternal Thirst", "{1}{B}")
	})


// Exquisite Blood {4}{B}
// Enchantment
// Whenever an opponent loses life, you gain that much life.
// TODO: implement
	Register("Exquisite Blood", func() Card {
		return NewEnchantment("Exquisite Blood", "{4}{B}")
	})


// Face of Divinity {2}{W}
// Enchantment — Aura
// Enchant creature
// Enchanted creature gets +2/+2.
// As long as another Aura is attached to enchanted creature, it has first strike and lifelink.
// TODO: implement
	Register("Face of Divinity", func() Card {
		return NewAura("Face of Divinity", "{2}{W}")
	})


// Feral Invocation {2}{G}
// Enchantment — Aura
// Flash (You may cast this spell any time you could cast an instant.)
// Enchant creature
// Enchanted creature gets +2/+2.
// TODO: implement
	Register("Feral Invocation", func() Card {
		return NewAura("Feral Invocation", "{2}{G}")
	})


// Forced Worship {1}{W}
// Enchantment — Aura
// Enchant creature
// Enchanted creature can't attack.
// {2}{W}: Return this Aura to its owner's hand.
// TODO: implement
	Register("Forced Worship", func() Card {
		return NewAura("Forced Worship", "{1}{W}")
	})


// Indomitable Will {1}{W}
// Enchantment — Aura
// Flash (You may cast this spell any time you could cast an instant.)
// Enchant creature
// Enchanted creature gets +1/+2.
// TODO: implement
	Register("Indomitable Will", func() Card {
		return NewAura("Indomitable Will", "{1}{W}")
	})


// Knightly Valor {4}{W}
// Enchantment — Aura
// Enchant creature
// When this Aura enters, create a 2/2 white Knight creature token with vigilance. (Attacking doesn't cause it to tap.)
// Enchanted creature gets +2/+2 and has vigilance.
// TODO: implement
	Register("Knightly Valor", func() Card {
		return NewAura("Knightly Valor", "{4}{W}")
	})


// Lawmage's Binding {1}{W}{U}
// Enchantment — Aura
// Flash
// Enchant creature
// Enchanted creature can't attack or block, and its activated abilities can't be activated.
// TODO: implement
	Register("Lawmage's Binding", func() Card {
		return NewAura("Lawmage's Binding", "{1}{W}{U}")
	})


// Lightning Diadem {5}{R}
// Enchantment — Aura
// Enchant creature
// When this Aura enters, it deals 2 damage to any target.
// Enchanted creature gets +2/+2.
// TODO: implement
	Register("Lightning Diadem", func() Card {
		return NewAura("Lightning Diadem", "{5}{R}")
	})


// Lurking Predators {4}{G}{G}
// Enchantment
// Whenever an opponent casts a spell, reveal the top card of your library. If it's a creature card, put it onto the battlefield. Otherwise, you may put that card on the bottom of your library.
// TODO: implement
	Register("Lurking Predators", func() Card {
		return NewEnchantment("Lurking Predators", "{4}{G}{G}")
	})


// Makeshift Munitions {1}{R}
// Enchantment
// {1}, Sacrifice an artifact or creature: This enchantment deals 1 damage to any target.
// TODO: implement
	Register("Makeshift Munitions", func() Card {
		return NewEnchantment("Makeshift Munitions", "{1}{R}")
	})


// Mark of the Vampire {3}{B}
// Enchantment — Aura
// Enchant creature
// Enchanted creature gets +2/+2 and has lifelink.
// TODO: implement
	Register("Mark of the Vampire", func() Card {
		return NewAura("Mark of the Vampire", "{3}{B}")
	})


// Narcolepsy {1}{U}
// Enchantment — Aura
// Enchant creature
// At the beginning of each upkeep, if enchanted creature is untapped, tap it.
// TODO: implement
	Register("Narcolepsy", func() Card {
		return NewAura("Narcolepsy", "{1}{U}")
	})


// New Horizons {2}{G}
// Enchantment — Aura
// Enchant land
// When this Aura enters, put a +1/+1 counter on target creature you control.
// Enchanted land has "{T}: Add two mana of any one color."
// TODO: implement
	Register("New Horizons", func() Card {
		return NewAura("New Horizons", "{2}{G}")
	})


// Pacifism {1}{W}
// Enchantment — Aura
// Enchant creature
// Enchanted creature can't attack or block.
// TODO: implement
	Register("Pacifism", func() Card {
		return NewAura("Pacifism", "{1}{W}")
	})


// Parasitic Implant {3}{B}
// Enchantment — Aura
// Enchant creature
// At the beginning of your upkeep, enchanted creature's controller sacrifices it and you create a 1/1 colorless Phyrexian Myr artifact creature token.
// TODO: implement
	Register("Parasitic Implant", func() Card {
		return NewAura("Parasitic Implant", "{3}{B}")
	})


// Path of Bravery {2}{W}
// Enchantment
// As long as your life total is greater than or equal to your starting life total, creatures you control get +1/+1.
// Whenever one or more creatures you control attack, you gain life equal to the number of attacking creatures.
// TODO: implement
	Register("Path of Bravery", func() Card {
		return NewEnchantment("Path of Bravery", "{2}{W}")
	})


// Phyrexian Reclamation {B}
// Enchantment
// {1}{B}, Pay 2 life: Return target creature card from your graveyard to your hand.
// TODO: implement
	Register("Phyrexian Reclamation", func() Card {
		return NewEnchantment("Phyrexian Reclamation", "{B}")
	})


// Presence of Gond {2}{G}
// Enchantment — Aura
// Enchant creature
// Enchanted creature has "{T}: Create a 1/1 green Elf Warrior creature token."
// TODO: implement
	Register("Presence of Gond", func() Card {
		return NewAura("Presence of Gond", "{2}{G}")
	})


// Primeval Bounty {5}{G}
// Enchantment
// Whenever you cast a creature spell, create a 3/3 green Beast creature token.
// Whenever you cast a noncreature spell, put three +1/+1 counters on target creature you control.
// Landfall — Whenever a land you control enters, you gain 3 life.
// TODO: implement
	Register("Primeval Bounty", func() Card {
		return NewEnchantment("Primeval Bounty", "{5}{G}")
	})


// Rhystic Study {2}{U}
// Enchantment
// Whenever an opponent casts a spell, you may draw a card unless that player pays {1}.
// TODO: implement
	Register("Rhystic Study", func() Card {
		return NewEnchantment("Rhystic Study", "{2}{U}")
	})


// Sarkhan's Unsealing {3}{R}
// Enchantment
// Whenever you cast a creature spell with power 4, 5, or 6, this enchantment deals 4 damage to any target.
// Whenever you cast a creature spell with power 7 or greater, this enchantment deals 4 damage to each opponent and each creature and planeswalker they control.
// TODO: implement
	Register("Sarkhan's Unsealing", func() Card {
		return NewEnchantment("Sarkhan's Unsealing", "{3}{R}")
	})


// Sky Tether {W}
// Enchantment — Aura
// Enchant creature
// Enchanted creature has defender and loses flying.
// TODO: implement
	Register("Sky Tether", func() Card {
		return NewAura("Sky Tether", "{W}")
	})


// Stab Wound {2}{B}
// Enchantment — Aura
// Enchant creature
// Enchanted creature gets -2/-2.
// At the beginning of the upkeep of enchanted creature's controller, that player loses 2 life.
// TODO: implement
	Register("Stab Wound", func() Card {
		return NewAura("Stab Wound", "{2}{B}")
	})


// Vastwood Zendikon {4}{G}
// Enchantment — Aura
// Enchant land
// Enchanted land is a 6/4 green Elemental creature. It's still a land.
// When enchanted land dies, return that card to its owner's hand.
// TODO: implement
	Register("Vastwood Zendikon", func() Card {
		return NewAura("Vastwood Zendikon", "{4}{G}")
	})


// Verdant Embrace {3}{G}{G}
// Enchantment — Aura
// Enchant creature
// Enchanted creature gets +3/+3 and has "At the beginning of each upkeep, create a 1/1 green Saproling creature token."
// TODO: implement
	Register("Verdant Embrace", func() Card {
		return NewAura("Verdant Embrace", "{3}{G}{G}")
	})


// Waterknot {1}{U}{U}
// Enchantment — Aura
// Enchant creature
// When this Aura enters, tap enchanted creature.
// Enchanted creature doesn't untap during its controller's untap step.
// TODO: implement
	Register("Waterknot", func() Card {
		return NewAura("Waterknot", "{1}{U}{U}")
	})


// Zendikar's Roil {3}{G}{G}
// Enchantment
// Landfall — Whenever a land you control enters, create a 2/2 green Elemental creature token.
// TODO: implement
	Register("Zendikar's Roil", func() Card {
		return NewEnchantment("Zendikar's Roil", "{3}{G}{G}")
	})


// Zombie Infestation {1}{B}
// Enchantment
// Discard two cards: Create a 2/2 black Zombie creature token.
// TODO: implement
	Register("Zombie Infestation", func() Card {
		return NewEnchantment("Zombie Infestation", "{1}{B}")
	})

}
