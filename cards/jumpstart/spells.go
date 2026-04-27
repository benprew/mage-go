package jumpstart

import . "git.sr.ht/~cdcarter/mage-go/pkg/mage"

func init() {
	registerSpells()
}

func registerSpells() {

// Act of Treason {2}{R}
// Sorcery
// Gain control of target creature until end of turn. Untap that creature. It gains haste until end of turn. (It can attack and {T} this turn.)
// TODO: implement
	Register("Act of Treason", func() Card {
		return NewSorcery("Act of Treason", "{2}{R}",
			NewSpellAbility(),
		)
	})


// Aegis of the Heavens {1}{W}
// Instant
// Target creature gets +1/+7 until end of turn.
// TODO: implement
	Register("Aegis of the Heavens", func() Card {
		return NewInstant("Aegis of the Heavens", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Aerial Assault {2}{W}
// Sorcery
// Destroy target tapped creature. You gain 1 life for each creature you control with flying.
// TODO: implement
	Register("Aerial Assault", func() Card {
		return NewSorcery("Aerial Assault", "{2}{W}",
			NewSpellAbility(),
		)
	})


// Aggressive Urge {1}{G}
// Instant
// Target creature gets +1/+1 until end of turn.
// Draw a card.
// TODO: implement
	Register("Aggressive Urge", func() Card {
		return NewInstant("Aggressive Urge", "{1}{G}",
			NewSpellAbility(),
		)
	})


// Agonizing Syphon {3}{B}
// Sorcery
// Agonizing Syphon deals 3 damage to any target and you gain 3 life.
// TODO: implement
	Register("Agonizing Syphon", func() Card {
		return NewSorcery("Agonizing Syphon", "{3}{B}",
			NewSpellAbility(),
		)
	})


// Angelic Edict {4}{W}
// Sorcery
// Exile target creature or enchantment.
// TODO: implement
	Register("Angelic Edict", func() Card {
		return NewSorcery("Angelic Edict", "{4}{W}",
			NewSpellAbility(),
		)
	})


// Arbor Armament {G}
// Instant
// Put a +1/+1 counter on target creature. That creature gains reach until end of turn. (It can block creatures with flying.)
// TODO: implement
	Register("Arbor Armament", func() Card {
		return NewInstant("Arbor Armament", "{G}",
			NewSpellAbility(),
		)
	})


// Assassin's Strike {4}{B}{B}
// Sorcery
// Destroy target creature. Its controller discards a card.
// TODO: implement
	Register("Assassin's Strike", func() Card {
		return NewSorcery("Assassin's Strike", "{4}{B}{B}",
			NewSpellAbility(),
		)
	})


// Auger Spree {1}{B}{R}
// Instant
// Target creature gets +4/-4 until end of turn.
// TODO: implement
	Register("Auger Spree", func() Card {
		return NewInstant("Auger Spree", "{1}{B}{R}",
			NewSpellAbility(),
		)
	})


// Bake into a Pie {2}{B}{B}
// Instant
// Destroy target creature. Create a Food token. (It's an artifact with "{2}, {T}, Sacrifice this token: You gain 3 life.")
// TODO: implement
	Register("Bake into a Pie", func() Card {
		return NewInstant("Bake into a Pie", "{2}{B}{B}",
			NewSpellAbility(),
		)
	})


// Barter in Blood {2}{B}{B}
// Sorcery
// Each player sacrifices two creatures of their choice.
// TODO: implement
	Register("Barter in Blood", func() Card {
		return NewSorcery("Barter in Blood", "{2}{B}{B}",
			NewSpellAbility(),
		)
	})


// Bathe in Dragonfire {2}{R}
// Sorcery
// Bathe in Dragonfire deals 4 damage to target creature.
// TODO: implement
	Register("Bathe in Dragonfire", func() Card {
		return NewSorcery("Bathe in Dragonfire", "{2}{R}",
			NewSpellAbility(),
		)
	})


// Battlefield Promotion {1}{W}
// Instant
// Put a +1/+1 counter on target creature. That creature gains first strike until end of turn. You gain 2 life. (A creature with first strike deals combat damage before creatures without first strike.)
// TODO: implement
	Register("Battlefield Promotion", func() Card {
		return NewInstant("Battlefield Promotion", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Befuddle {2}{U}
// Instant
// Target creature gets -4/-0 until end of turn.
// Draw a card.
// TODO: implement
	Register("Befuddle", func() Card {
		return NewInstant("Befuddle", "{2}{U}",
			NewSpellAbility(),
		)
	})


// Blindblast {2}{R}
// Instant
// Blindblast deals 1 damage to target creature. That creature can't block this turn.
// Draw a card.
// TODO: implement
	Register("Blindblast", func() Card {
		return NewInstant("Blindblast", "{2}{R}",
			NewSpellAbility(),
		)
	})


// Blood Divination {3}{B}
// Sorcery
// As an additional cost to cast this spell, sacrifice a creature.
// Draw three cards.
// TODO: implement
	Register("Blood Divination", func() Card {
		return NewSorcery("Blood Divination", "{3}{B}",
			NewSpellAbility(),
		)
	})


// Bone Splinters {B}
// Sorcery
// As an additional cost to cast this spell, sacrifice a creature.
// Destroy target creature.
// TODO: implement
	Register("Bone Splinters", func() Card {
		return NewSorcery("Bone Splinters", "{B}",
			NewSpellAbility(),
		)
	})


// Cemetery Recruitment {1}{B}
// Sorcery
// Return target creature card from your graveyard to your hand. If it's a Zombie card, draw a card.
// TODO: implement
	Register("Cemetery Recruitment", func() Card {
		return NewSorcery("Cemetery Recruitment", "{1}{B}",
			NewSpellAbility(),
		)
	})


// Chart a Course {1}{U}
// Sorcery
// Draw two cards. Then discard a card unless you attacked this turn.
// TODO: implement
	Register("Chart a Course", func() Card {
		return NewSorcery("Chart a Course", "{1}{U}",
			NewSpellAbility(),
		)
	})


// Cloudshift {W}
// Instant
// Exile target creature you control, then return that card to the battlefield under your control.
// TODO: implement
	Register("Cloudshift", func() Card {
		return NewInstant("Cloudshift", "{W}",
			NewSpellAbility(),
		)
	})


// Collateral Damage {R}
// Instant
// As an additional cost to cast this spell, sacrifice a creature.
// Collateral Damage deals 3 damage to any target.
// TODO: implement
	Register("Collateral Damage", func() Card {
		return NewInstant("Collateral Damage", "{R}",
			NewSpellAbility(),
		)
	})


// Commune with Dinosaurs {G}
// Sorcery
// Look at the top five cards of your library. You may reveal a Dinosaur or land card from among them and put it into your hand. Put the rest on the bottom of your library in any order.
// TODO: implement
	Register("Commune with Dinosaurs", func() Card {
		return NewSorcery("Commune with Dinosaurs", "{G}",
			NewSpellAbility(),
		)
	})


// Crushing Canopy {2}{G}
// Instant
// Choose one —
// • Destroy target creature with flying.
// • Destroy target enchantment.
// TODO: implement
	Register("Crushing Canopy", func() Card {
		return NewInstant("Crushing Canopy", "{2}{G}",
			NewSpellAbility(),
		)
	})


// Dance with Devils {3}{R}
// Instant
// Create two 1/1 red Devil creature tokens. They have "When this token dies, it deals 1 damage to any target."
// TODO: implement
	Register("Dance with Devils", func() Card {
		return NewInstant("Dance with Devils", "{3}{R}",
			NewSpellAbility(),
		)
	})


// Dauntless Onslaught {2}{W}
// Instant
// Up to two target creatures each get +2/+2 until end of turn.
// TODO: implement
	Register("Dauntless Onslaught", func() Card {
		return NewInstant("Dauntless Onslaught", "{2}{W}",
			NewSpellAbility(),
		)
	})


// Divine Arrow {1}{W}
// Instant
// Divine Arrow deals 4 damage to target attacking or blocking creature.
// TODO: implement
	Register("Divine Arrow", func() Card {
		return NewInstant("Divine Arrow", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Doublecast {R}{R}
// Sorcery
// When you next cast an instant or sorcery spell this turn, copy that spell. You may choose new targets for the copy.
// TODO: implement
	Register("Doublecast", func() Card {
		return NewSorcery("Doublecast", "{R}{R}",
			NewSpellAbility(),
		)
	})


// Douse in Gloom {2}{B}
// Instant
// Douse in Gloom deals 2 damage to target creature and you gain 2 life.
// TODO: implement
	Register("Douse in Gloom", func() Card {
		return NewInstant("Douse in Gloom", "{2}{B}",
			NewSpellAbility(),
		)
	})


// Draconic Roar {1}{R}
// Instant
// As an additional cost to cast this spell, you may reveal a Dragon card from your hand.
// Draconic Roar deals 3 damage to target creature. If you revealed a Dragon card or controlled a Dragon as you cast this spell, Draconic Roar deals 3 damage to that creature's controller.
// TODO: implement
	Register("Draconic Roar", func() Card {
		return NewInstant("Draconic Roar", "{1}{R}",
			NewSpellAbility(),
		)
	})


// Dragon Fodder {1}{R}
// Sorcery
// Create two 1/1 red Goblin creature tokens.
// TODO: implement
	Register("Dragon Fodder", func() Card {
		return NewSorcery("Dragon Fodder", "{1}{R}",
			NewSpellAbility(),
		)
	})


// Elemental Uprising {1}{G}
// Instant
// Target land you control becomes a 4/4 Elemental creature with haste until end of turn. It's still a land. It must be blocked this turn if able.
// TODO: implement
	Register("Elemental Uprising", func() Card {
		return NewInstant("Elemental Uprising", "{1}{G}",
			NewSpellAbility(),
		)
	})


// Enlarge {3}{G}{G}
// Sorcery
// Target creature gets +7/+7 and gains trample until end of turn. It must be blocked this turn if able. (A creature with trample can deal excess combat damage to the player or planeswalker it's attacking.)
// TODO: implement
	Register("Enlarge", func() Card {
		return NewSorcery("Enlarge", "{3}{G}{G}",
			NewSpellAbility(),
		)
	})


// Essence Flux {U}
// Instant
// Exile target creature you control, then return that card to the battlefield under its owner's control. If it's a Spirit, put a +1/+1 counter on it.
// TODO: implement
	Register("Essence Flux", func() Card {
		return NewInstant("Essence Flux", "{U}",
			NewSpellAbility(),
		)
	})


// Exclude {2}{U}
// Instant
// Counter target creature spell.
// Draw a card.
// TODO: implement
	Register("Exclude", func() Card {
		return NewInstant("Exclude", "{2}{U}",
			NewSpellAbility(),
		)
	})


// Exhume {1}{B}
// Sorcery
// Each player puts a creature card from their graveyard onto the battlefield.
// TODO: implement
	Register("Exhume", func() Card {
		return NewSorcery("Exhume", "{1}{B}",
			NewSpellAbility(),
		)
	})


// Explore {1}{G}
// Sorcery
// You may play an additional land this turn.
// Draw a card.
// TODO: implement
	Register("Explore", func() Card {
		return NewSorcery("Explore", "{1}{G}",
			NewSpellAbility(),
		)
	})


// Flame Lash {3}{R}
// Instant
// Flame Lash deals 4 damage to any target.
// TODO: implement
	Register("Flame Lash", func() Card {
		return NewInstant("Flame Lash", "{3}{R}",
			NewSpellAbility(),
		)
	})


// Flames of the Firebrand {2}{R}
// Sorcery
// Flames of the Firebrand deals 3 damage divided as you choose among one, two, or three targets.
// TODO: implement
	Register("Flames of the Firebrand", func() Card {
		return NewSorcery("Flames of the Firebrand", "{2}{R}",
			NewSpellAbility(),
		)
	})


// Flames of the Raze-Boar {5}{R}
// Instant
// Flames of the Raze-Boar deals 4 damage to target creature an opponent controls. Then Flames of the Raze-Boar deals 2 damage to each other creature that player controls if you control a creature with power 4 or greater.
// TODO: implement
	Register("Flames of the Raze-Boar", func() Card {
		return NewInstant("Flames of the Raze-Boar", "{5}{R}",
			NewSpellAbility(),
		)
	})


// Fling {1}{R}
// Instant
// As an additional cost to cast this spell, sacrifice a creature.
// Fling deals damage equal to the sacrificed creature's power to any target.
// TODO: implement
	Register("Fling", func() Card {
		return NewInstant("Fling", "{1}{R}",
			NewSpellAbility(),
		)
	})


// Flurry of Horns {4}{R}
// Sorcery
// Create two 2/3 red Minotaur creature tokens with haste.
// TODO: implement
	Register("Flurry of Horns", func() Card {
		return NewSorcery("Flurry of Horns", "{4}{R}",
			NewSpellAbility(),
		)
	})


// Fortify {2}{W}
// Instant
// Choose one —
// • Creatures you control get +2/+0 until end of turn.
// • Creatures you control get +0/+2 until end of turn.
// TODO: implement
	Register("Fortify", func() Card {
		return NewInstant("Fortify", "{2}{W}",
			NewSpellAbility(),
		)
	})


// Funeral Rites {2}{B}
// Sorcery
// You draw two cards, lose 2 life, then mill two cards.
// TODO: implement
	Register("Funeral Rites", func() Card {
		return NewSorcery("Funeral Rites", "{2}{B}",
			NewSpellAbility(),
		)
	})


// Gird for Battle {W}
// Sorcery
// Put a +1/+1 counter on each of up to two target creatures.
// TODO: implement
	Register("Gird for Battle", func() Card {
		return NewSorcery("Gird for Battle", "{W}",
			NewSpellAbility(),
		)
	})


// Goblin Lore {1}{R}
// Sorcery
// Draw four cards, then discard three cards at random.
// TODO: implement
	Register("Goblin Lore", func() Card {
		return NewSorcery("Goblin Lore", "{1}{R}",
			NewSpellAbility(),
		)
	})


// Goblin Rally {3}{R}{R}
// Sorcery
// Create four 1/1 red Goblin creature tokens.
// TODO: implement
	Register("Goblin Rally", func() Card {
		return NewSorcery("Goblin Rally", "{3}{R}{R}",
			NewSpellAbility(),
		)
	})


// Heartfire {1}{R}
// Instant
// As an additional cost to cast this spell, sacrifice a creature or planeswalker.
// Heartfire deals 4 damage to any target.
// TODO: implement
	Register("Heartfire", func() Card {
		return NewInstant("Heartfire", "{1}{R}",
			NewSpellAbility(),
		)
	})


// Homing Lightning {2}{R}{R}
// Instant
// Homing Lightning deals 4 damage to target creature and each other creature with the same name as that creature.
// TODO: implement
	Register("Homing Lightning", func() Card {
		return NewInstant("Homing Lightning", "{2}{R}{R}",
			NewSpellAbility(),
		)
	})


// Hungry Flames {2}{R}
// Instant
// Hungry Flames deals 3 damage to target creature and 2 damage to target player or planeswalker.
// TODO: implement
	Register("Hungry Flames", func() Card {
		return NewInstant("Hungry Flames", "{2}{R}",
			NewSpellAbility(),
		)
	})


// Hunter's Insight {2}{G}
// Instant
// Choose target creature you control. Whenever that creature deals combat damage to a player or planeswalker this turn, draw that many cards.
// TODO: implement
	Register("Hunter's Insight", func() Card {
		return NewInstant("Hunter's Insight", "{2}{G}",
			NewSpellAbility(),
		)
	})


// Immolating Gyre {4}{R}{R}
// Sorcery
// Immolating Gyre deals X damage to each creature and planeswalker you don't control, where X is the number of instant and sorcery cards in your graveyard.
// TODO: implement
	Register("Immolating Gyre", func() Card {
		return NewSorcery("Immolating Gyre", "{4}{R}{R}",
			NewSpellAbility(),
		)
	})


// Innocent Blood {B}
// Sorcery
// Each player sacrifices a creature of their choice.
// TODO: implement
	Register("Innocent Blood", func() Card {
		return NewSorcery("Innocent Blood", "{B}",
			NewSpellAbility(),
		)
	})


// Inspired Charge {2}{W}{W}
// Instant
// Creatures you control get +2/+1 until end of turn.
// TODO: implement
	Register("Inspired Charge", func() Card {
		return NewInstant("Inspired Charge", "{2}{W}{W}",
			NewSpellAbility(),
		)
	})


// Inspiring Call {2}{G}
// Instant
// Draw a card for each creature you control with a +1/+1 counter on it. Those creatures gain indestructible until end of turn. (Damage and effects that say "destroy" don't destroy them.)
// TODO: implement
	Register("Inspiring Call", func() Card {
		return NewInstant("Inspiring Call", "{2}{G}",
			NewSpellAbility(),
		)
	})


// Irresistible Prey {G}
// Sorcery
// Target creature must be blocked this turn if able.
// Draw a card.
// TODO: implement
	Register("Irresistible Prey", func() Card {
		return NewSorcery("Irresistible Prey", "{G}",
			NewSpellAbility(),
		)
	})


// Languish {2}{B}{B}
// Sorcery
// All creatures get -4/-4 until end of turn.
// TODO: implement
	Register("Languish", func() Card {
		return NewSorcery("Languish", "{2}{B}{B}",
			NewSpellAbility(),
		)
	})


// Last Gasp {1}{B}
// Instant
// Target creature gets -3/-3 until end of turn.
// TODO: implement
	Register("Last Gasp", func() Card {
		return NewInstant("Last Gasp", "{1}{B}",
			NewSpellAbility(),
		)
	})


// Launch Party {3}{B}
// Instant
// As an additional cost to cast this spell, sacrifice a creature.
// Destroy target creature. Its controller loses 2 life.
// TODO: implement
	Register("Launch Party", func() Card {
		return NewInstant("Launch Party", "{3}{B}",
			NewSpellAbility(),
		)
	})


// Leave in the Dust {3}{U}
// Instant
// Return target nonland permanent to its owner's hand.
// Draw a card.
// TODO: implement
	Register("Leave in the Dust", func() Card {
		return NewInstant("Leave in the Dust", "{3}{U}",
			NewSpellAbility(),
		)
	})


// Lifecrafter's Gift {3}{G}
// Instant
// Put a +1/+1 counter on target creature, then put a +1/+1 counter on each creature you control with a +1/+1 counter on it.
// TODO: implement
	Register("Lifecrafter's Gift", func() Card {
		return NewInstant("Lifecrafter's Gift", "{3}{G}",
			NewSpellAbility(),
		)
	})


// Lightning Axe {R}
// Instant
// As an additional cost to cast this spell, discard a card or pay {5}.
// Lightning Axe deals 5 damage to target creature.
// TODO: implement
	Register("Lightning Axe", func() Card {
		return NewInstant("Lightning Axe", "{R}",
			NewSpellAbility(),
		)
	})


// Long Road Home {1}{W}
// Instant
// Exile target creature. At the beginning of the next end step, return that card to the battlefield under its owner's control with a +1/+1 counter on it.
// TODO: implement
	Register("Long Road Home", func() Card {
		return NewInstant("Long Road Home", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Macabre Waltz {1}{B}
// Sorcery
// Return up to two target creature cards from your graveyard to your hand, then discard a card.
// TODO: implement
	Register("Macabre Waltz", func() Card {
		return NewSorcery("Macabre Waltz", "{1}{B}",
			NewSpellAbility(),
		)
	})


// Magma Jet {1}{R}
// Instant
// Magma Jet deals 2 damage to any target. Scry 2.
// TODO: implement
	Register("Magma Jet", func() Card {
		return NewInstant("Magma Jet", "{1}{R}",
			NewSpellAbility(),
		)
	})


// Magmaquake {X}{R}{R}
// Instant
// Magmaquake deals X damage to each creature without flying and each planeswalker.
// TODO: implement
	Register("Magmaquake", func() Card {
		return NewInstant("Magmaquake", "{X}{R}{R}",
			NewSpellAbility(),
		)
	})


// Moment of Heroism {1}{W}
// Instant
// Target creature gets +2/+2 and gains lifelink until end of turn. (Damage dealt by the creature also causes its controller to gain that much life.)
// TODO: implement
	Register("Moment of Heroism", func() Card {
		return NewInstant("Moment of Heroism", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Momentous Fall {2}{G}{G}
// Instant
// As an additional cost to cast this spell, sacrifice a creature.
// You draw cards equal to the sacrificed creature's power, then you gain life equal to its toughness.
// TODO: implement
	Register("Momentous Fall", func() Card {
		return NewInstant("Momentous Fall", "{2}{G}{G}",
			NewSpellAbility(),
		)
	})


// Mugging {R}
// Sorcery
// Mugging deals 2 damage to target creature. That creature can't block this turn.
// TODO: implement
	Register("Mugging", func() Card {
		return NewSorcery("Mugging", "{R}",
			NewSpellAbility(),
		)
	})


// Nature's Way {1}{G}
// Sorcery
// Target creature you control gains vigilance and trample until end of turn. It deals damage equal to its power to target creature you don't control.
// TODO: implement
	Register("Nature's Way", func() Card {
		return NewSorcery("Nature's Way", "{1}{G}",
			NewSpellAbility(),
		)
	})


// Outnumber {R}
// Instant
// Outnumber deals damage to target creature equal to the number of creatures you control.
// TODO: implement
	Register("Outnumber", func() Card {
		return NewInstant("Outnumber", "{R}",
			NewSpellAbility(),
		)
	})


// Path to Exile {W}
// Instant
// Exile target creature. Its controller may search their library for a basic land card, put that card onto the battlefield tapped, then shuffle.
// TODO: implement
	Register("Path to Exile", func() Card {
		return NewInstant("Path to Exile", "{W}",
			NewSpellAbility(),
		)
	})


// Peel from Reality {1}{U}
// Instant
// Return target creature you control and target creature you don't control to their owners' hands.
// TODO: implement
	Register("Peel from Reality", func() Card {
		return NewInstant("Peel from Reality", "{1}{U}",
			NewSpellAbility(),
		)
	})


// Pillar of Flame {R}
// Sorcery
// Pillar of Flame deals 2 damage to any target. If a creature dealt damage this way would die this turn, exile it instead.
// TODO: implement
	Register("Pillar of Flame", func() Card {
		return NewSorcery("Pillar of Flame", "{R}",
			NewSpellAbility(),
		)
	})


// Raise the Alarm {1}{W}
// Instant
// Create two 1/1 white Soldier creature tokens.
// TODO: implement
	Register("Raise the Alarm", func() Card {
		return NewInstant("Raise the Alarm", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Read the Runes {X}{U}
// Instant
// Draw X cards. For each card drawn this way, discard a card unless you sacrifice a permanent.
// TODO: implement
	Register("Read the Runes", func() Card {
		return NewInstant("Read the Runes", "{X}{U}",
			NewSpellAbility(),
		)
	})


// Reanimate {B}
// Sorcery
// Put target creature card from a graveyard onto the battlefield under your control. You lose life equal to that card's mana value.
// TODO: implement
	Register("Reanimate", func() Card {
		return NewSorcery("Reanimate", "{B}",
			NewSpellAbility(),
		)
	})


// Release the Dogs {3}{W}
// Sorcery
// Create four 1/1 white Dog creature tokens.
// TODO: implement
	Register("Release the Dogs", func() Card {
		return NewSorcery("Release the Dogs", "{3}{W}",
			NewSpellAbility(),
		)
	})


// Riddle of Lightning {3}{R}{R}
// Instant
// Choose any target. Scry 3, then reveal the top card of your library. Riddle of Lightning deals damage equal to that card's mana value to that permanent or player.
// TODO: implement
	Register("Riddle of Lightning", func() Card {
		return NewInstant("Riddle of Lightning", "{3}{R}{R}",
			NewSpellAbility(),
		)
	})


// Rise of the Dark Realms {7}{B}{B}
// Sorcery
// Put all creature cards from all graveyards onto the battlefield under your control.
// TODO: implement
	Register("Rise of the Dark Realms", func() Card {
		return NewSorcery("Rise of the Dark Realms", "{7}{B}{B}",
			NewSpellAbility(),
		)
	})


// Sarkhan's Rage {4}{R}
// Instant
// Sarkhan's Rage deals 5 damage to any target. If you control no Dragons, Sarkhan's Rage deals 2 damage to you.
// TODO: implement
	Register("Sarkhan's Rage", func() Card {
		return NewInstant("Sarkhan's Rage", "{4}{R}",
			NewSpellAbility(),
		)
	})


// Savage Stomp {2}{G}
// Sorcery
// This spell costs {2} less to cast if it targets a Dinosaur you control.
// Put a +1/+1 counter on target creature you control. Then that creature fights target creature you don't control. (Each deals damage equal to its power to the other.)
// TODO: implement
	Register("Savage Stomp", func() Card {
		return NewSorcery("Savage Stomp", "{2}{G}",
			NewSpellAbility(),
		)
	})


// Settle the Score {2}{B}{B}
// Sorcery
// Exile target creature. Put two loyalty counters on a planeswalker you control.
// TODO: implement
	Register("Settle the Score", func() Card {
		return NewSorcery("Settle the Score", "{2}{B}{B}",
			NewSpellAbility(),
		)
	})


// Soul Salvage {2}{B}
// Sorcery
// Return up to two target creature cards from your graveyard to your hand.
// TODO: implement
	Register("Soul Salvage", func() Card {
		return NewSorcery("Soul Salvage", "{2}{B}",
			NewSpellAbility(),
		)
	})


// Spitting Earth {1}{R}
// Sorcery
// Spitting Earth deals damage to target creature equal to the number of Mountains you control.
// TODO: implement
	Register("Spitting Earth", func() Card {
		return NewSorcery("Spitting Earth", "{1}{R}",
			NewSpellAbility(),
		)
	})


// Sweep Away {2}{U}
// Instant
// Return target creature to its owner's hand. If that creature is attacking, you may put it on top of its owner's library instead.
// TODO: implement
	Register("Sweep Away", func() Card {
		return NewInstant("Sweep Away", "{2}{U}",
			NewSpellAbility(),
		)
	})


// Take Heart {W}
// Instant
// Target creature gets +2/+2 until end of turn. You gain 1 life for each attacking creature you control.
// TODO: implement
	Register("Take Heart", func() Card {
		return NewInstant("Take Heart", "{W}",
			NewSpellAbility(),
		)
	})


// Talrand's Invocation {2}{U}{U}
// Sorcery
// Create two 2/2 blue Drake creature tokens with flying.
// TODO: implement
	Register("Talrand's Invocation", func() Card {
		return NewSorcery("Talrand's Invocation", "{2}{U}{U}",
			NewSpellAbility(),
		)
	})


// Tandem Tactics {1}{W}
// Instant
// Up to two target creatures each get +1/+2 until end of turn. You gain 2 life.
// TODO: implement
	Register("Tandem Tactics", func() Card {
		return NewInstant("Tandem Tactics", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Thirst for Knowledge {2}{U}
// Instant
// Draw three cards. Then discard two cards unless you discard an artifact card.
// TODO: implement
	Register("Thirst for Knowledge", func() Card {
		return NewInstant("Thirst for Knowledge", "{2}{U}",
			NewSpellAbility(),
		)
	})


// Thought Collapse {1}{U}{U}
// Instant
// Counter target spell. Its controller mills three cards.
// TODO: implement
	Register("Thought Collapse", func() Card {
		return NewInstant("Thought Collapse", "{1}{U}{U}",
			NewSpellAbility(),
		)
	})


// Thought Scour {U}
// Instant
// Target player mills two cards.
// Draw a card.
// TODO: implement
	Register("Thought Scour", func() Card {
		return NewInstant("Thought Scour", "{U}",
			NewSpellAbility(),
		)
	})


// Time to Feed {2}{G}
// Sorcery
// Choose target creature an opponent controls. When that creature dies this turn, you gain 3 life. Target creature you control fights that creature. (Each deals damage equal to its power to the other.)
// TODO: implement
	Register("Time to Feed", func() Card {
		return NewSorcery("Time to Feed", "{2}{G}",
			NewSpellAbility(),
		)
	})


// Valorous Stance {1}{W}
// Instant
// Choose one —
// • Target creature gains indestructible until end of turn. (Damage and effects that say "destroy" don't destroy it.)
// • Destroy target creature with toughness 4 or greater.
// TODO: implement
	Register("Valorous Stance", func() Card {
		return NewInstant("Valorous Stance", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Volcanic Fallout {1}{R}{R}
// Instant
// This spell can't be countered.
// Volcanic Fallout deals 2 damage to each creature and each player.
// TODO: implement
	Register("Volcanic Fallout", func() Card {
		return NewInstant("Volcanic Fallout", "{1}{R}{R}",
			NewSpellAbility(),
		)
	})


// Voyage's End {1}{U}
// Instant
// Return target creature to its owner's hand. Scry 1. (Look at the top card of your library. You may put that card on the bottom.)
// TODO: implement
	Register("Voyage's End", func() Card {
		return NewInstant("Voyage's End", "{1}{U}",
			NewSpellAbility(),
		)
	})


// Whelming Wave {2}{U}{U}
// Sorcery
// Return all creatures to their owners' hands except for Krakens, Leviathans, Octopuses, and Serpents.
// TODO: implement
	Register("Whelming Wave", func() Card {
		return NewSorcery("Whelming Wave", "{2}{U}{U}",
			NewSpellAbility(),
		)
	})


// Wildsize {2}{G}
// Instant
// Target creature gets +2/+2 and gains trample until end of turn.
// Draw a card.
// TODO: implement
	Register("Wildsize", func() Card {
		return NewInstant("Wildsize", "{2}{G}",
			NewSpellAbility(),
		)
	})


// Winged Words {2}{U}
// Sorcery
// This spell costs {1} less to cast if you control a creature with flying.
// Draw two cards.
// TODO: implement
	Register("Winged Words", func() Card {
		return NewSorcery("Winged Words", "{2}{U}",
			NewSpellAbility(),
		)
	})


// Wizard's Retort {1}{U}{U}
// Instant
// This spell costs {1} less to cast if you control a Wizard.
// Counter target spell.
// TODO: implement
	Register("Wizard's Retort", func() Card {
		return NewInstant("Wizard's Retort", "{1}{U}{U}",
			NewSpellAbility(),
		)
	})

}
