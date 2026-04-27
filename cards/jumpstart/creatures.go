package jumpstart

import (
	"math/rand"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var _ = uuid.Nil

func init() {
	registerCreatures()
}

func registerCreatures() {

	// ===== WHITE CREATURES =====

// Affa Guard Hound {2}{W}
// Creature — Dog
// 2/2
// Flash (You may cast this spell any time you could cast an instant.)
// When this creature enters, target creature gets +0/+3 until end of turn.
// TODO: implement
	Register("Affa Guard Hound", func() Card {
		return NewCreature("Affa Guard Hound", "{2}{W}", 2, 2,
			WithSubTypes("Dog"),
		)
	})

// Ajani's Chosen {2}{W}{W}
// Creature — Cat Soldier
// 3/3
// Whenever an enchantment you control enters, create a 2/2 white Cat creature token. If that enchantment is an Aura, you may attach it to the token.
// TODO: implement
	Register("Ajani's Chosen", func() Card {
		return NewCreature("Ajani's Chosen", "{2}{W}{W}", 3, 3,
			WithSubTypes("Cat", "Soldier"),
		)
	})

// Alabaster Mage {1}{W}
// Creature — Human Wizard
// 2/1
// {1}{W}: Target creature you control gains lifelink until end of turn. (Damage dealt by the creature also causes its controller to gain that much life.)
// TODO: implement
	Register("Alabaster Mage", func() Card {
		return NewCreature("Alabaster Mage", "{1}{W}", 2, 1,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Angel of Mercy {4}{W}
// Creature — Angel
// 3/3
// Flying
// When this creature enters, you gain 3 life.
// TODO: implement
	Register("Angel of Mercy", func() Card {
		return NewCreature("Angel of Mercy", "{4}{W}", 3, 3,
			WithSubTypes("Angel"),
		)
	})

// Angel of the Dire Hour {5}{W}{W}
// Creature — Angel
// 5/4
// Flash
// Flying
// When this creature enters, if you cast it from your hand, exile all attacking creatures.
// TODO: implement
	Register("Angel of the Dire Hour", func() Card {
		return NewCreature("Angel of the Dire Hour", "{5}{W}{W}", 5, 4,
			WithSubTypes("Angel"),
		)
	})

// Angelic Arbiter {5}{W}{W}
// Creature — Angel
// 5/6
// Flying
// Each opponent who cast a spell this turn can't attack with creatures.
// Each opponent who attacked with a creature this turn can't cast spells.
// TODO: implement
	Register("Angelic Arbiter", func() Card {
		return NewCreature("Angelic Arbiter", "{5}{W}{W}", 5, 6,
			WithSubTypes("Angel"),
		)
	})

// Angelic Page {1}{W}
// Creature — Angel Spirit
// 1/1
// Flying
// {T}: Target attacking or blocking creature gets +1/+1 until end of turn.
// TODO: implement
	Register("Angelic Page", func() Card {
		return NewCreature("Angelic Page", "{1}{W}", 1, 1,
			WithSubTypes("Angel", "Spirit"),
		)
	})

// Archon of Justice {3}{W}{W}
// Creature — Archon
// 4/4
// Flying
// When this creature dies, exile target permanent.
// TODO: implement
	Register("Archon of Justice", func() Card {
		return NewCreature("Archon of Justice", "{3}{W}{W}", 4, 4,
			WithSubTypes("Archon"),
		)
	})

// Archon of Redemption {3}{W}{W}
// Creature — Archon
// 3/4
// Flying
// Whenever this creature or another creature you control with flying enters, you may gain life equal to that creature's power.
// TODO: implement
	Register("Archon of Redemption", func() Card {
		return NewCreature("Archon of Redemption", "{3}{W}{W}", 3, 4,
			WithSubTypes("Archon"),
		)
	})

// Blessed Spirits {2}{W}
// Creature — Spirit
// 2/2
// Flying
// Whenever you cast an enchantment spell, put a +1/+1 counter on this creature.
// TODO: implement
	Register("Blessed Spirits", func() Card {
		return NewCreature("Blessed Spirits", "{2}{W}", 2, 2,
			WithSubTypes("Spirit"),
		)
	})

// Brightmare {2}{W}
// Creature — Unicorn
// 2/3
// When this creature enters, tap up to one target creature. You gain life equal to that creature's power.
// TODO: implement
	Register("Brightmare", func() Card {
		return NewCreature("Brightmare", "{2}{W}", 2, 3,
			WithSubTypes("Unicorn"),
		)
	})

// Bulwark Giant {5}{W}
// Creature — Giant Soldier
// 3/6
// When this creature enters, you gain 5 life.
// TODO: implement
	Register("Bulwark Giant", func() Card {
		return NewCreature("Bulwark Giant", "{5}{W}", 3, 6,
			WithSubTypes("Giant", "Soldier"),
		)
	})

// Cathar's Companion {2}{W}
// Creature — Dog
// 3/1
// Whenever you cast a noncreature spell, this creature gains indestructible until end of turn. (Damage and effects that say "destroy" don't destroy it.)
// TODO: implement
	Register("Cathar's Companion", func() Card {
		return NewCreature("Cathar's Companion", "{2}{W}", 3, 1,
			WithSubTypes("Dog"),
		)
	})

// Emancipation Angel {1}{W}{W}
// Creature — Angel
// 3/3
// Flying
// When this creature enters, return a permanent you control to its owner's hand.
// TODO: implement
	Register("Emancipation Angel", func() Card {
		return NewCreature("Emancipation Angel", "{1}{W}{W}", 3, 3,
			WithSubTypes("Angel"),
		)
	})

// Emiel the Blessed {2}{W}{W}
// Legendary Creature — Unicorn
// 4/4
// {3}: Exile another target creature you control, then return it to the battlefield under its owner's control.
// Whenever another creature you control enters, you may pay {G/W}. If you do, put a +1/+1 counter on it. If it's a Unicorn, put two +1/+1 counters on it instead. ({G/W} can be paid with either {G} or {W}.)
// TODO: implement
	Register("Emiel the Blessed", func() Card {
		return NewCreature("Emiel the Blessed", "{2}{W}{W}", 4, 4,
			WithSubTypes("Unicorn"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Healer's Hawk {W}
// Creature — Bird
// 1/1
// Flying
// Lifelink (Damage dealt by this creature also causes you to gain that much life.)
// TODO: implement
	Register("Healer's Hawk", func() Card {
		return NewCreature("Healer's Hawk", "{W}", 1, 1,
			WithSubTypes("Bird"),
		)
	})

// High Sentinels of Arashin {3}{W}
// Creature — Bird Soldier
// 3/4
// Flying
// This creature gets +1/+1 for each other creature you control with a +1/+1 counter on it.
// {3}{W}: Put a +1/+1 counter on target creature.
// TODO: implement
	Register("High Sentinels of Arashin", func() Card {
		return NewCreature("High Sentinels of Arashin", "{3}{W}", 3, 4,
			WithSubTypes("Bird", "Soldier"),
		)
	})

// Inspiring Captain {3}{W}
// Creature — Human Knight
// 3/3
// When this creature enters, creatures you control get +1/+1 until end of turn.
// TODO: implement
	Register("Inspiring Captain", func() Card {
		return NewCreature("Inspiring Captain", "{3}{W}", 3, 3,
			WithSubTypes("Human", "Knight"),
		)
	})

// Inspiring Unicorn {2}{W}{W}
// Creature — Unicorn
// 2/2
// Whenever this creature attacks, creatures you control get +1/+1 until end of turn.
// TODO: implement
	Register("Inspiring Unicorn", func() Card {
		return NewCreature("Inspiring Unicorn", "{2}{W}{W}", 2, 2,
			WithSubTypes("Unicorn"),
		)
	})

// Isamaru, Hound of Konda {W}
// Legendary Creature — Dog
// 2/2
// TODO: implement
	Register("Isamaru, Hound of Konda", func() Card {
		return NewCreature("Isamaru, Hound of Konda", "{W}", 2, 2,
			WithSubTypes("Dog"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Knight of the Tusk {4}{W}{W}
// Creature — Human Knight
// 3/7
// Vigilance (Attacking doesn't cause this creature to tap.)
// TODO: implement
	Register("Knight of the Tusk", func() Card {
		return NewCreature("Knight of the Tusk", "{4}{W}{W}", 3, 7,
			WithSubTypes("Human", "Knight"),
		)
	})

// Kor Spiritdancer {1}{W}
// Creature — Kor Wizard
// 0/2
// This creature gets +2/+2 for each Aura attached to it.
// Whenever you cast an Aura spell, you may draw a card.
// TODO: implement
	Register("Kor Spiritdancer", func() Card {
		return NewCreature("Kor Spiritdancer", "{1}{W}", 0, 2,
			WithSubTypes("Kor", "Wizard"),
		)
	})

// Lena, Selfless Champion {4}{W}{W}
// Legendary Creature — Human Knight
// 3/3
// When Lena enters, create a 1/1 white Soldier creature token for each nontoken creature you control.
// Sacrifice Lena: Creatures you control with power less than Lena's power gain indestructible until end of turn.
// TODO: implement
	Register("Lena, Selfless Champion", func() Card {
		return NewCreature("Lena, Selfless Champion", "{4}{W}{W}", 3, 3,
			WithSubTypes("Human", "Knight"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Lightwalker {1}{W}
// Creature — Human Warrior
// 2/1
// This creature has flying as long as it has a +1/+1 counter on it.
// TODO: implement
	Register("Lightwalker", func() Card {
		return NewCreature("Lightwalker", "{1}{W}", 2, 1,
			WithSubTypes("Human", "Warrior"),
		)
	})

// Linvala, Keeper of Silence {2}{W}{W}
// Legendary Creature — Angel
// 3/4
// Flying
// Activated abilities of creatures your opponents control can't be activated.
// TODO: implement
	Register("Linvala, Keeper of Silence", func() Card {
		return NewCreature("Linvala, Keeper of Silence", "{2}{W}{W}", 3, 4,
			WithSubTypes("Angel"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Mentor of the Meek {2}{W}
// Creature — Human Soldier
// 2/2
// Whenever another creature you control with power 2 or less enters, you may pay {1}. If you do, draw a card.
// TODO: implement
	Register("Mentor of the Meek", func() Card {
		return NewCreature("Mentor of the Meek", "{2}{W}", 2, 2,
			WithSubTypes("Human", "Soldier"),
		)
	})

// Mesa Unicorn {1}{W}
// Creature — Unicorn
// 2/2
// Lifelink (Damage dealt by this creature also causes you to gain that much life.)
// TODO: implement
	Register("Mesa Unicorn", func() Card {
		return NewCreature("Mesa Unicorn", "{1}{W}", 2, 2,
			WithSubTypes("Unicorn"),
		)
	})

// Mikaeus, the Lunarch {X}{W}
// Legendary Creature — Human Cleric
// 0/0
// Mikaeus enters with X +1/+1 counters on it.
// {T}: Put a +1/+1 counter on Mikaeus.
// {T}, Remove a +1/+1 counter from Mikaeus: Put a +1/+1 counter on each other creature you control.
// TODO: implement
	Register("Mikaeus, the Lunarch", func() Card {
		return NewCreature("Mikaeus, the Lunarch", "{X}{W}", 0, 0,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Patron of the Valiant {3}{W}{W}
// Creature — Angel
// 4/4
// Flying
// When this creature enters, put a +1/+1 counter on each creature you control with a +1/+1 counter on it.
// TODO: implement
	Register("Patron of the Valiant", func() Card {
		return NewCreature("Patron of the Valiant", "{3}{W}{W}", 4, 4,
			WithSubTypes("Angel"),
		)
	})

// Rhox Faithmender {3}{W}
// Creature — Rhino Monk
// 1/5
// Lifelink (Damage dealt by this creature also causes you to gain that much life.)
// If you would gain life, you gain twice that much life instead.
// TODO: implement
	Register("Rhox Faithmender", func() Card {
		return NewCreature("Rhox Faithmender", "{3}{W}", 1, 5,
			WithSubTypes("Rhino", "Monk"),
		)
	})

// Ronom Unicorn {1}{W}
// Creature — Unicorn
// 2/2
// Sacrifice this creature: Destroy target enchantment.
// TODO: implement
	Register("Ronom Unicorn", func() Card {
		return NewCreature("Ronom Unicorn", "{1}{W}", 2, 2,
			WithSubTypes("Unicorn"),
		)
	})

// Steel-Plume Marshal {3}{W}{W}
// Creature — Bird Soldier
// 3/3
// Flying
// Whenever this creature attacks, other attacking creatures you control with flying get +2/+2 until end of turn.
// TODO: implement
	Register("Steel-Plume Marshal", func() Card {
		return NewCreature("Steel-Plume Marshal", "{3}{W}{W}", 3, 3,
			WithSubTypes("Bird", "Soldier"),
		)
	})

// Stone Haven Pilgrim {1}{W}
// Creature — Kor Cleric
// 2/2
// Whenever this creature attacks, if you control an artifact or enchantment, this creature gets +1/+1 and gains lifelink until end of turn.
// TODO: implement
	Register("Stone Haven Pilgrim", func() Card {
		return NewCreature("Stone Haven Pilgrim", "{1}{W}", 2, 2,
			WithSubTypes("Kor", "Cleric"),
		)
	})

// Supply Runners {4}{W}
// Creature — Dog
// 2/2
// When this creature enters, put a +1/+1 counter on each other creature you control.
// TODO: implement
	Register("Supply Runners", func() Card {
		return NewCreature("Supply Runners", "{4}{W}", 2, 2,
			WithSubTypes("Dog"),
		)
	})

// Trusty Retriever {3}{W}
// Creature — Dog
// 2/3
// When this creature enters, choose one —
// • Put a +1/+1 counter on this creature.
// • Return target artifact or enchantment card from your graveyard to your hand.
// TODO: implement
	Register("Trusty Retriever", func() Card {
		return NewCreature("Trusty Retriever", "{3}{W}", 2, 3,
			WithSubTypes("Dog"),
		)
	})

// Voice of the Provinces {4}{W}{W}
// Creature — Angel
// 3/3
// Flying
// When this creature enters, create a 1/1 white Human creature token.
// TODO: implement
	Register("Voice of the Provinces", func() Card {
		return NewCreature("Voice of the Provinces", "{4}{W}{W}", 3, 3,
			WithSubTypes("Angel"),
		)
	})


	// ===== BLUE CREATURES =====

// Aegis Turtle {U}
// Creature — Turtle
// 0/5
// TODO: implement
	Register("Aegis Turtle", func() Card {
		return NewCreature("Aegis Turtle", "{U}", 0, 5,
			WithSubTypes("Turtle"),
		)
	})

// Archaeomender {2}{U}
// Creature — Human Wizard
// 2/3
// When this creature enters, return target artifact card from your graveyard to your hand.
// TODO: implement
	Register("Archaeomender", func() Card {
		return NewCreature("Archaeomender", "{2}{U}", 2, 3,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Battleground Geist {4}{U}
// Creature — Spirit
// 3/3
// Flying
// Other Spirit creatures you control get +1/+0.
// TODO: implement
	Register("Battleground Geist", func() Card {
		return NewCreature("Battleground Geist", "{4}{U}", 3, 3,
			WithSubTypes("Spirit"),
		)
	})

// Belltower Sphinx {4}{U}
// Creature — Sphinx
// 2/5
// Flying
// Whenever a source deals damage to this creature, that source's controller mills that many cards.
// TODO: implement
	Register("Belltower Sphinx", func() Card {
		return NewCreature("Belltower Sphinx", "{4}{U}", 2, 5,
			WithSubTypes("Sphinx"),
		)
	})

// Bruvac the Grandiloquent {2}{U}
// Legendary Creature — Human Advisor
// 1/4
// If an opponent would mill one or more cards, they mill twice that many cards instead. (To mill a card, a player puts the top card of their library into their graveyard.)
// TODO: implement
	Register("Bruvac the Grandiloquent", func() Card {
		return NewCreature("Bruvac the Grandiloquent", "{2}{U}", 1, 4,
			WithSubTypes("Human", "Advisor"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Cloudreader Sphinx {4}{U}
// Creature — Sphinx
// 3/4
// Flying
// When this creature enters, scry 2. (Look at the top two cards of your library, then put any number of them on the bottom and the rest on top in any order.)
// TODO: implement
	Register("Cloudreader Sphinx", func() Card {
		return NewCreature("Cloudreader Sphinx", "{4}{U}", 3, 4,
			WithSubTypes("Sphinx"),
		)
	})

// Corsair Captain {2}{U}
// Creature — Human Pirate
// 2/2
// When this creature enters, create a Treasure token. (It's an artifact with "{T}, Sacrifice this token: Add one mana of any color.")
// Other Pirates you control get +1/+1.
// TODO: implement
	Register("Corsair Captain", func() Card {
		return NewCreature("Corsair Captain", "{2}{U}", 2, 2,
			WithSubTypes("Human", "Pirate"),
		)
	})

// Crookclaw Transmuter {3}{U}
// Creature — Bird Wizard
// 3/1
// Flash
// Flying
// When this creature enters, switch target creature's power and toughness until end of turn.
// TODO: implement
	Register("Crookclaw Transmuter", func() Card {
		return NewCreature("Crookclaw Transmuter", "{3}{U}", 3, 1,
			WithSubTypes("Bird", "Wizard"),
		)
	})

// Cryptic Serpent {5}{U}{U}
// Creature — Serpent
// 6/5
// This spell costs {1} less to cast for each instant and sorcery card in your graveyard.
// TODO: implement
	Register("Cryptic Serpent", func() Card {
		return NewCreature("Cryptic Serpent", "{5}{U}{U}", 6, 5,
			WithSubTypes("Serpent"),
		)
	})

// Departed Deckhand {1}{U}
// Creature — Spirit Pirate
// 2/2
// When this creature becomes the target of a spell, sacrifice it.
// This creature can't be blocked except by Spirits.
// {3}{U}: Another target creature you control can't be blocked this turn except by Spirits.
// TODO: implement
	Register("Departed Deckhand", func() Card {
		return NewCreature("Departed Deckhand", "{1}{U}", 2, 2,
			WithSubTypes("Spirit", "Pirate"),
		)
	})

// Erratic Visionary {1}{U}
// Creature — Human Wizard
// 1/3
// {1}{U}, {T}: Draw a card, then discard a card.
// TODO: implement
	Register("Erratic Visionary", func() Card {
		return NewCreature("Erratic Visionary", "{1}{U}", 1, 3,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Exclusion Mage {2}{U}
// Creature — Human Wizard
// 2/2
// When this creature enters, return target creature an opponent controls to its owner's hand.
// TODO: implement
	Register("Exclusion Mage", func() Card {
		return NewCreature("Exclusion Mage", "{2}{U}", 2, 2,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Inniaz, the Gale Force {3}{U}{U}
// Legendary Creature — Djinn
// 4/4
// Flying
// {2}{W/U}: Attacking creatures with flying get +1/+1 until end of turn. ({W/U} can be paid with either {W} or {U}.)
// Whenever three or more creatures you control with flying attack, each player gains control of a nonland permanent of your choice controlled by the player to their right.
// TODO: implement
	Register("Inniaz, the Gale Force", func() Card {
		return NewCreature("Inniaz, the Gale Force", "{3}{U}{U}", 4, 4,
			WithSubTypes("Djinn"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Kira, Great Glass-Spinner {1}{U}{U}
// Legendary Creature — Spirit
// 2/2
// Flying
// Creatures you control have "Whenever this creature becomes the target of a spell or ability for the first time each turn, counter that spell or ability."
// TODO: implement
	Register("Kira, Great Glass-Spinner", func() Card {
		return NewCreature("Kira, Great Glass-Spinner", "{1}{U}{U}", 2, 2,
			WithSubTypes("Spirit"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Kitesail Corsair {1}{U}
// Creature — Human Pirate
// 2/1
// This creature has flying as long as it's attacking.
// TODO: implement
	Register("Kitesail Corsair", func() Card {
		return NewCreature("Kitesail Corsair", "{1}{U}", 2, 1,
			WithSubTypes("Human", "Pirate"),
		)
	})

// Murmuring Phantasm {1}{U}
// Creature — Spirit
// 0/5
// Defender
// TODO: implement
	Register("Murmuring Phantasm", func() Card {
		return NewCreature("Murmuring Phantasm", "{1}{U}", 0, 5,
			WithSubTypes("Spirit"),
		)
	})

// Mystic Archaeologist {1}{U}
// Creature — Human Wizard
// 2/1
// {3}{U}{U}: Draw two cards.
// TODO: implement
	Register("Mystic Archaeologist", func() Card {
		return NewCreature("Mystic Archaeologist", "{1}{U}", 2, 1,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Nebelgast Herald {2}{U}
// Creature — Spirit
// 2/1
// Flash
// Flying
// Whenever this creature or another Spirit you control enters, tap target creature an opponent controls.
// TODO: implement
	Register("Nebelgast Herald", func() Card {
		return NewCreature("Nebelgast Herald", "{2}{U}", 2, 1,
			WithSubTypes("Spirit"),
		)
	})

// Octoprophet {3}{U}
// Creature — Octopus
// 3/3
// When this creature enters, scry 2. (Look at the top two cards of your library, then put any number of them on the bottom and the rest on top in any order.)
// TODO: implement
	Register("Octoprophet", func() Card {
		return NewCreature("Octoprophet", "{3}{U}", 3, 3,
			WithSubTypes("Octopus"),
		)
	})

// Oneirophage {3}{U}
// Creature — Squid Illusion
// 1/2
// Flying
// Whenever you draw a card, put a +1/+1 counter on this creature.
// TODO: implement
	Register("Oneirophage", func() Card {
		return NewCreature("Oneirophage", "{3}{U}", 1, 2,
			WithSubTypes("Squid", "Illusion"),
		)
	})

// Ormos, Archive Keeper {4}{U}{U}
// Legendary Creature — Sphinx
// 5/5
// Flying
// If you would draw a card while your library has no cards in it, instead put five +1/+1 counters on Ormos.
// {1}{U}{U}, Discard three cards with different names: Draw five cards.
// TODO: implement
	Register("Ormos, Archive Keeper", func() Card {
		return NewCreature("Ormos, Archive Keeper", "{4}{U}{U}", 5, 5,
			WithSubTypes("Sphinx"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Prescient Chimera {3}{U}{U}
// Creature — Chimera
// 3/4
// Flying
// Whenever you cast an instant or sorcery spell, scry 1. (Look at the top card of your library. You may put that card on the bottom.)
// TODO: implement
	Register("Prescient Chimera", func() Card {
		return NewCreature("Prescient Chimera", "{3}{U}{U}", 3, 4,
			WithSubTypes("Chimera"),
		)
	})

// Prosperous Pirates {4}{U}
// Creature — Human Pirate
// 3/4
// When this creature enters, create two Treasure tokens. (They're artifacts with "{T}, Sacrifice this token: Add one mana of any color.")
// TODO: implement
	Register("Prosperous Pirates", func() Card {
		return NewCreature("Prosperous Pirates", "{4}{U}", 3, 4,
			WithSubTypes("Human", "Pirate"),
		)
	})

// Rattlechains {1}{U}
// Creature — Spirit
// 2/1
// Flash
// Flying
// When this creature enters, target Spirit gains hexproof until end of turn.
// You may cast Spirit spells as though they had flash.
// TODO: implement
	Register("Rattlechains", func() Card {
		return NewCreature("Rattlechains", "{1}{U}", 2, 1,
			WithSubTypes("Spirit"),
		)
	})

// Reckless Scholar {2}{U}
// Creature — Human Wizard
// 2/1
// {T}: Target player draws a card, then discards a card.
// TODO: implement
	Register("Reckless Scholar", func() Card {
		return NewCreature("Reckless Scholar", "{2}{U}", 2, 1,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Rishadan Airship {2}{U}
// Creature — Human Pirate
// 3/1
// Flying
// This creature can block only creatures with flying.
// TODO: implement
	Register("Rishadan Airship", func() Card {
		return NewCreature("Rishadan Airship", "{2}{U}", 3, 1,
			WithSubTypes("Human", "Pirate"),
		)
	})

// Sage's Row Savant {1}{U}
// Creature — Vedalken Wizard
// 2/1
// When this creature enters, scry 2.
// TODO: implement
	Register("Sage's Row Savant", func() Card {
		return NewCreature("Sage's Row Savant", "{1}{U}", 2, 1,
			WithSubTypes("Vedalken", "Wizard"),
		)
	})

// Sailor of Means {2}{U}
// Creature — Human Pirate
// 1/4
// When this creature enters, create a Treasure token. (It's an artifact with "{T}, Sacrifice this token: Add one mana of any color.")
// TODO: implement
	Register("Sailor of Means", func() Card {
		return NewCreature("Sailor of Means", "{2}{U}", 1, 4,
			WithSubTypes("Human", "Pirate"),
		)
	})

// Scholar of the Lost Trove {5}{U}{U}
// Creature — Sphinx
// 5/5
// Flying
// When this creature enters, you may cast target instant, sorcery, or artifact card from your graveyard without paying its mana cost. If an instant or sorcery spell cast this way would be put into your graveyard, exile it instead.
// TODO: implement
	Register("Scholar of the Lost Trove", func() Card {
		return NewCreature("Scholar of the Lost Trove", "{5}{U}{U}", 5, 5,
			WithSubTypes("Sphinx"),
		)
	})

// Sea Gate Oracle {2}{U}
// Creature — Human Wizard
// 1/3
// When this creature enters, look at the top two cards of your library. Put one of them into your hand and the other on the bottom of your library.
// TODO: implement
	Register("Sea Gate Oracle", func() Card {
		return NewCreature("Sea Gate Oracle", "{2}{U}", 1, 3,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Selhoff Occultist {2}{U}
// Creature — Human Rogue
// 2/3
// Whenever this creature or another creature dies, target player mills a card.
// TODO: implement
	Register("Selhoff Occultist", func() Card {
		return NewCreature("Selhoff Occultist", "{2}{U}", 2, 3,
			WithSubTypes("Human", "Rogue"),
		)
	})

// Sharding Sphinx {4}{U}{U}
// Artifact Creature — Sphinx
// 4/4
// Flying
// Whenever an artifact creature you control deals combat damage to a player, you may create a 1/1 blue Thopter artifact creature token with flying.
// TODO: implement
	Register("Sharding Sphinx", func() Card {
		return NewCreature("Sharding Sphinx", "{4}{U}{U}", 4, 4,
			WithSubTypes("Sphinx"),
			WithCardType(TypeArtifact),
		)
	})

// Sigiled Starfish {1}{U}
// Creature — Starfish
// 0/3
// {T}: Scry 1. (Look at the top card of your library. You may put that card on the bottom.)
// TODO: implement
	Register("Sigiled Starfish", func() Card {
		return NewCreature("Sigiled Starfish", "{1}{U}", 0, 3,
			WithSubTypes("Starfish"),
		)
	})

// Spectral Sailor {U}
// Creature — Spirit Pirate
// 1/1
// Flash (You may cast this spell any time you could cast an instant.)
// Flying
// {3}{U}: Draw a card.
// TODO: implement
	Register("Spectral Sailor", func() Card {
		return NewCreature("Spectral Sailor", "{U}", 1, 1,
			WithSubTypes("Spirit", "Pirate"),
		)
	})

// Storm Sculptor {3}{U}
// Creature — Merfolk Wizard
// 3/2
// This creature can't be blocked.
// When this creature enters, return a creature you control to its owner's hand.
// TODO: implement
	Register("Storm Sculptor", func() Card {
		return NewCreature("Storm Sculptor", "{3}{U}", 3, 2,
			WithSubTypes("Merfolk", "Wizard"),
		)
	})

// Talrand, Sky Summoner {2}{U}{U}
// Legendary Creature — Merfolk Wizard
// 2/2
// Whenever you cast an instant or sorcery spell, create a 2/2 blue Drake creature token with flying.
	Register("Talrand, Sky Summoner", func() Card {
		instOrSorc := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Talrand, Sky Summoner", "{2}{U}{U}", 2, 2,
			WithSubTypes("Merfolk", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithAbility(WheneverYouCastSpellTrigger(
				CreateColoredToken("Drake", 2, 2, []Color{Blue}, []CardType{TypeCreature}, []string{"Drake"}, Flying),
				false, instOrSorc,
			)),
		)
	})

// Towering-Wave Mystic {1}{U}
// Creature — Merfolk Wizard
// 2/1
// Whenever this creature deals damage, target player mills that many cards.
	Register("Towering-Wave Mystic", func() Card {
		return NewCreature("Towering-Wave Mystic", "{1}{U}", 2, 1,
			WithSubTypes("Merfolk", "Wizard"),
			WithAbility(NewTriggered(EvtDamageDealt, false,
				MillTargetPlayer(EventAmountValue()),
			).SetConditionData(EventSourceIsSelf{}).AddTarget(TargetPlayer())),
		)
	})

// Vedalken Archmage {2}{U}{U}
// Creature — Vedalken Wizard
// 0/2
// Whenever you cast an artifact spell, draw a card.
	Register("Vedalken Archmage", func() Card {
		return NewCreature("Vedalken Archmage", "{2}{U}{U}", 0, 2,
			WithSubTypes("Vedalken", "Wizard"),
			WithAbility(WheneverYouCastSpellTrigger(
				DrawCards(Fixed(1)),
				false, IsArtifactCard,
			)),
		)
	})

// Vedalken Entrancer {3}{U}
// Creature — Vedalken Wizard
// 1/4
// {U}, {T}: Target player mills two cards.
	Register("Vedalken Entrancer", func() Card {
		return NewCreature("Vedalken Entrancer", "{3}{U}", 1, 4,
			WithSubTypes("Vedalken", "Wizard"),
			WithActivatedAbility(
				MillTargetPlayer(Fixed(2)),
				ManaCostOf("{U}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetPlayer()),
			),
		)
	})

// Wall of Lost Thoughts {1}{U}
// Creature — Wall
// 0/4
// Defender (This creature can't attack.)
// When this creature enters, target player mills four cards. (They put the top four cards of their library into their graveyard.)
	Register("Wall of Lost Thoughts", func() Card {
		return NewCreature("Wall of Lost Thoughts", "{1}{U}", 0, 4,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(EntersBattlefieldTrigger(MillTargetPlayer(Fixed(4)), false).AddTarget(TargetPlayer())),
		)
	})

// Warden of Evos Isle {2}{U}
// Creature — Bird Wizard
// 2/2
// Flying
// Creature spells with flying you cast cost {1} less to cast.
// XXX: requires keyword-based spell cost reduction
	Register("Warden of Evos Isle", func() Card {
		return NewCreature("Warden of Evos Isle", "{2}{U}", 2, 2,
			WithSubTypes("Bird", "Wizard"),
			WithKeyword(Flying),
		)
	})

// Windreader Sphinx {5}{U}{U}
// Creature — Sphinx
// 3/7
// Flying
// Whenever a creature with flying attacks, you may draw a card.
	Register("Windreader Sphinx", func() Card {
		return NewCreature("Windreader Sphinx", "{5}{U}{U}", 3, 7,
			WithSubTypes("Sphinx"),
			WithKeyword(Flying),
			WithAbility(NewTriggered(EvtDeclaredAttacker, true,
				DrawCards(Fixed(1)),
			).SetConditionData(EventSourceMatchesPermanentFilter{Filter: HasKeywordFilter(Flying)})),
		)
	})

// Windstorm Drake {4}{U}
// Creature — Drake
// 3/3
// Flying
// Other creatures you control with flying get +1/+0.
	Register("Windstorm Drake", func() Card {
		return NewCreature("Windstorm Drake", "{4}{U}", 3, 3,
			WithSubTypes("Drake"),
			WithKeyword(Flying),
			WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				for _, p := range g.FilterBattlefield(And(IsCreature, ControlledBy(src.Controller), HasKeywordFilter(Flying))) {
					if p.ID() == sourceID {
						continue
					}
					p.BoostPT(1, 0)
				}
				return nil
			})),
		)
	})

// Wishful Merfolk {1}{U}
// Creature — Merfolk
// 3/2
// Defender
// {1}{U}: This creature loses defender and becomes a Human until end of turn.
// XXX: requires temporary subtype change with keyword removal until EOT
	Register("Wishful Merfolk", func() Card {
		return NewCreature("Wishful Merfolk", "{1}{U}", 3, 2,
			WithSubTypes("Merfolk"),
			WithKeyword(Defender),
		)
	})


	// ===== BLACK CREATURES =====

// A-Blood Artist {1}{B}
// Creature — Vampire
// 0/1
// Whenever Blood Artist or another creature dies, target opponent loses 1 life and you gain 1 life.
	Register("A-Blood Artist", func() Card {
		return NewCreature("A-Blood Artist", "{1}{B}", 0, 1,
			WithSubTypes("Vampire"),
			WithAbility(AnyCreatureDiesTrigger(
				FuncEffect("target opponent loses 1 life and you gain 1 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 {
							if tp := g.GetPlayer(targets[0]); tp != nil {
								tp.LoseLife(1)
							}
						}
						if cp := g.GetPlayer(controller); cp != nil {
							g.PlayerGainLife(cp, 1)
						}
						return nil
					}),
				false,
			).AddTarget(TargetOpponent())),
		)
	})

// Black Cat {1}{B}
// Creature — Zombie Cat
// 1/1
// When this creature dies, target opponent discards a card at random.
	Register("Black Cat", func() Card {
		return NewCreature("Black Cat", "{1}{B}", 1, 1,
			WithSubTypes("Zombie", "Cat"),
			WithAbility(NewTriggered(EvtCreatureDied, false, DiscardRandom(1)).
				SetConditionData(EventSourceIsSelf{}).
				AddTarget(TargetOpponent())),
		)
	})

// Blighted Bat {2}{B}
// Creature — Zombie Bat
// 2/1
// Flying
// {1}: This creature gains haste until end of turn.
	Register("Blighted Bat", func() Card {
		return NewCreature("Blighted Bat", "{2}{B}", 2, 1,
			WithSubTypes("Zombie", "Bat"),
			WithKeyword(Flying),
			WithActivatedAbility(
				GrantKeyword(Haste).Targeting(ToSource()),
				GenericCost(1),
			),
		)
	})

// Blood Artist {1}{B}
// Creature — Vampire
// 0/1
// Whenever this creature or another creature dies, target player loses 1 life and you gain 1 life.
	Register("Blood Artist", func() Card {
		return NewCreature("Blood Artist", "{1}{B}", 0, 1,
			WithSubTypes("Vampire"),
			WithAbility(AnyCreatureDiesTrigger(
				FuncEffect("target player loses 1 life and you gain 1 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 {
							if tp := g.GetPlayer(targets[0]); tp != nil {
								tp.LoseLife(1)
							}
						}
						if cp := g.GetPlayer(controller); cp != nil {
							g.PlayerGainLife(cp, 1)
						}
						return nil
					}),
				false,
			).AddTarget(TargetPlayer())),
		)
	})

// Blood Host {3}{B}{B}
// Creature — Vampire
// 3/3
// {1}{B}, Sacrifice another creature: Put a +1/+1 counter on this creature and you gain 2 life.
	Register("Blood Host", func() Card {
		return NewCreature("Blood Host", "{3}{B}{B}", 3, 3,
			WithSubTypes("Vampire"),
			WithActivatedAbility(
				CompositeEffects("put +1/+1 counter on this creature and gain 2 life",
					AddCounters(P1P1, Fixed(1)).Targeting(ToSource()),
					GainLife(2),
				),
				ManaCostOf("{1}{B}"),
				WithCost(SacrificeCreatureCost()),
			),
		)
	})

// Bloodbond Vampire {2}{B}{B}
// Creature — Vampire Shaman Ally
// 3/3
// Whenever you gain life, put a +1/+1 counter on this creature.
// XXX: requires gain-life event trigger
	Register("Bloodbond Vampire", func() Card {
		return NewCreature("Bloodbond Vampire", "{2}{B}{B}", 3, 3,
			WithSubTypes("Vampire", "Shaman", "Ally"),
		)
	})

// Bloodhunter Bat {3}{B}
// Creature — Bat
// 2/2
// Flying
// When this creature enters, target player loses 2 life and you gain 2 life.
	Register("Bloodhunter Bat", func() Card {
		return NewCreature("Bloodhunter Bat", "{3}{B}", 2, 2,
			WithSubTypes("Bat"),
			WithKeyword(Flying),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("target player loses 2 life and you gain 2 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 {
							if tp := g.GetPlayer(targets[0]); tp != nil {
								tp.LoseLife(2)
							}
						}
						if cp := g.GetPlayer(controller); cp != nil {
							g.PlayerGainLife(cp, 2)
						}
						return nil
					}),
				false,
			).AddTarget(TargetPlayer())),
		)
	})

// Bogbrew Witch {3}{B}
// Creature — Human Wizard
// 1/3
// {2}, {T}: Search your library for a card named Festering Newt or Bubbling Cauldron, put it onto the battlefield tapped, then shuffle.
// XXX: requires search-by-name-list-to-battlefield-tapped primitive
	Register("Bogbrew Witch", func() Card {
		return NewCreature("Bogbrew Witch", "{3}{B}", 1, 3,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Bone Picker {3}{B}
// Creature — Bird
// 3/2
// This spell costs {3} less to cast if a creature died this turn.
// Flying, deathtouch
// XXX: requires conditional cast-cost reduction
	Register("Bone Picker", func() Card {
		return NewCreature("Bone Picker", "{3}{B}", 3, 2,
			WithSubTypes("Bird"),
			WithKeyword(Flying),
			WithKeyword(Deathtouch),
		)
	})

// Burglar Rat {1}{B}
// Creature — Rat
// 1/1
// When this creature enters, each opponent discards a card.
	Register("Burglar Rat", func() Card {
		return NewCreature("Burglar Rat", "{1}{B}", 1, 1,
			WithSubTypes("Rat"),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("each opponent discards a card",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, p := range g.AllPlayers() {
							if p.PlayerID() == controller {
								continue
							}
							if len(p.Hand()) == 0 {
								continue
							}
							for _, c := range p.ChooseCardsFromHand(1, "discard", g) {
								p.DiscardCard(c.ID())
							}
						}
						return nil
					}),
				false,
			)),
		)
	})

// Cadaver Imp {1}{B}{B}
// Creature — Imp
// 1/1
// Flying
// When this creature enters, you may return target creature card from your graveyard to your hand.
	Register("Cadaver Imp", func() Card {
		return NewCreature("Cadaver Imp", "{1}{B}{B}", 1, 1,
			WithSubTypes("Imp"),
			WithKeyword(Flying),
			WithAbility(EntersBattlefieldTrigger(
				ReturnFromGraveyardToHandTarget(),
				true,
			).AddTarget(TargetCardInYourGraveyard(IsCreatureCard))),
		)
	})

// Cauldron Familiar {B}
// Creature — Cat
// 1/1
// When this creature enters, each opponent loses 1 life and you gain 1 life.
// Sacrifice a Food: Return this card from your graveyard to the battlefield.
// XXX: requires Food token primitive
	Register("Cauldron Familiar", func() Card {
		return NewCreature("Cauldron Familiar", "{B}", 1, 1,
			WithSubTypes("Cat"),
		)
	})

// Child of Night {1}{B}
// Creature — Vampire
// 2/1
// Lifelink
	Register("Child of Night", func() Card {
		return NewCreature("Child of Night", "{1}{B}", 2, 1,
			WithSubTypes("Vampire"),
			WithKeyword(Lifelink),
		)
	})

// Corpse Hauler {1}{B}
// Creature — Human Rogue
// 2/1
// {2}{B}, Sacrifice this creature: Return another target creature card from your graveyard to your hand.
	Register("Corpse Hauler", func() Card {
		return NewCreature("Corpse Hauler", "{1}{B}", 2, 1,
			WithSubTypes("Human", "Rogue"),
			WithActivatedAbility(
				ReturnFromGraveyardToHandTarget(),
				ManaCostOf("{2}{B}"),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCardInYourGraveyard(IsCreatureCard)),
			),
		)
	})

// Corpse Traders {3}{B}
// Creature — Human Rogue
// 3/3
// {2}{B}, Sacrifice a creature: Target opponent reveals their hand. You choose a card from it. That player discards that card. Activate only as a sorcery.
// XXX: requires reveal-hand-and-controller-chooses-discard primitive
	Register("Corpse Traders", func() Card {
		return NewCreature("Corpse Traders", "{3}{B}", 3, 3,
			WithSubTypes("Human", "Rogue"),
		)
	})

// Crow of Dark Tidings {2}{B}
// Creature — Zombie Bird
// 2/1
// Flying
// When this creature enters or dies, mill two cards. (Put the top two cards of your library into your graveyard.)
	Register("Crow of Dark Tidings", func() Card {
		millSelf2 := FuncEffect("mill two cards",
			EffectProperties{},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				p := g.GetPlayer(controller)
				if p == nil {
					return nil
				}
				lib := p.Library()
				for i := 0; i < 2 && len(lib) > 0; i++ {
					card := lib[len(lib)-1]
					lib = lib[:len(lib)-1]
					p.AddToGraveyard(card)
				}
				p.SetLibrary(lib)
				return nil
			})
		return NewCreature("Crow of Dark Tidings", "{2}{B}", 2, 1,
			WithSubTypes("Zombie", "Bird"),
			WithKeyword(Flying),
			WithAbility(EntersBattlefieldTrigger(millSelf2, false)),
			WithAbility(NewTriggered(EvtCreatureDied, false, millSelf2).
				SetConditionData(EventSourceIsSelf{})),
		)
	})

// Drainpipe Vermin {B}
// Creature — Rat
// 1/1
// When this creature dies, you may pay {B}. If you do, target player discards a card.
// XXX: requires player-controlled may-pay-mana primitive in trigger resolution
	Register("Drainpipe Vermin", func() Card {
		return NewCreature("Drainpipe Vermin", "{B}", 1, 1,
			WithSubTypes("Rat"),
		)
	})

// Drana, Liberator of Malakir {1}{B}{B}
// Legendary Creature — Vampire Ally
// 2/3
// Flying, first strike
// Whenever Drana deals combat damage to a player, put a +1/+1 counter on each attacking creature you control.
	Register("Drana, Liberator of Malakir", func() Card {
		return NewCreature("Drana, Liberator of Malakir", "{1}{B}{B}", 2, 3,
			WithSubTypes("Vampire", "Ally"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithKeyword(FirstStrike),
			WithAbility(NewTriggered(EvtDamageDealt, false,
				FuncEffect("put a +1/+1 counter on each attacking creature you control",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, p := range g.FilterBattlefield(And(IsCreature, IsAttacking, ControlledBy(controller))) {
							p.AddCounter(P1P1, 1)
						}
						return nil
					}),
			).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
				EventSourceIsSelfDamageToPlayer{},
				EventFlagIsTrue{},
			}})),
		)
	})

// Dutiful Attendant {2}{B}
// Creature — Human Warrior
// 1/2
// When this creature dies, return another target creature card from your graveyard to your hand.
	Register("Dutiful Attendant", func() Card {
		return NewCreature("Dutiful Attendant", "{2}{B}", 1, 2,
			WithSubTypes("Human", "Warrior"),
			WithAbility(NewTriggered(EvtCreatureDied, false,
				ReturnFromGraveyardToHandTarget(),
			).SetConditionData(EventSourceIsSelf{}).
				AddTarget(TargetCardInYourGraveyard(IsCreatureCard))),
		)
	})

// Entomber Exarch {2}{B}{B}
// Creature — Phyrexian Cleric
// 2/2
// When this creature enters, choose one —
// • Return target creature card from your graveyard to your hand.
// • Target opponent reveals their hand. You choose a noncreature card from it. That player discards that card.
// XXX: requires modal-ETB and reveal-hand-and-controller-chooses-discard primitives
	Register("Entomber Exarch", func() Card {
		return NewCreature("Entomber Exarch", "{2}{B}{B}", 2, 2,
			WithSubTypes("Phyrexian", "Cleric"),
		)
	})

// Eternal Taskmaster {1}{B}
// Creature — Zombie
// 2/3
// This creature enters tapped.
// Whenever this creature attacks, you may pay {2}{B}. If you do, return target creature card from your graveyard to your hand.
// XXX: requires player-controlled may-pay-mana primitive in trigger resolution
	Register("Eternal Taskmaster", func() Card {
		return NewCreature("Eternal Taskmaster", "{1}{B}", 2, 3,
			WithSubTypes("Zombie"),
			WithKeyword(EntersTapped),
		)
	})

// Falkenrath Noble {3}{B}
// Creature — Vampire Noble
// 2/2
// Flying
// Whenever this creature or another creature dies, target player loses 1 life and you gain 1 life.
	Register("Falkenrath Noble", func() Card {
		return NewCreature("Falkenrath Noble", "{3}{B}", 2, 2,
			WithSubTypes("Vampire", "Noble"),
			WithKeyword(Flying),
			WithAbility(AnyCreatureDiesTrigger(
				FuncEffect("target player loses 1 life and you gain 1 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 {
							if tp := g.GetPlayer(targets[0]); tp != nil {
								tp.LoseLife(1)
							}
						}
						if cp := g.GetPlayer(controller); cp != nil {
							g.PlayerGainLife(cp, 1)
						}
						return nil
					}),
				false,
			).AddTarget(TargetPlayer())),
		)
	})

// Fell Specter {3}{B}
// Creature — Specter
// 1/3
// Flying
// When this creature enters, target opponent discards a card.
// Whenever an opponent discards a card, that player loses 2 life.
// XXX: requires discard event for the "whenever an opponent discards" trigger
	Register("Fell Specter", func() Card {
		return NewCreature("Fell Specter", "{3}{B}", 1, 3,
			WithSubTypes("Specter"),
			WithKeyword(Flying),
		)
	})

// Festering Newt {B}
// Creature — Salamander
// 1/1
// When this creature dies, target creature an opponent controls gets -1/-1 until end of turn. That creature gets -4/-4 instead if you control a creature named Bogbrew Witch.
	Register("Festering Newt", func() Card {
		return NewCreature("Festering Newt", "{B}", 1, 1,
			WithSubTypes("Salamander"),
			WithAbility(NewTriggered(EvtCreatureDied, false,
				FuncEffect("target creature gets -1/-1 (-4/-4 with Bogbrew Witch) until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						amount := 1
						if g.AnyBattlefield(And(IsCreature, ControlledBy(controller), Named("Bogbrew Witch"))) {
							amount = 4
						}
						ce := TemporaryBoost(targets[0], -amount, -amount)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					}),
			).SetConditionData(EventSourceIsSelf{}).
				AddTarget(TargetPermanentOpponentControls(IsCreature))),
		)
	})

// Ghoulcaller Gisa {3}{B}{B}
// Legendary Creature — Human Wizard
// 3/4
// {B}, {T}, Sacrifice another creature: Create X 2/2 black Zombie creature tokens, where X is the sacrificed creature's power.
// XXX: requires sacrificed-creature-power capture in cost->effect chain
	Register("Ghoulcaller Gisa", func() Card {
		return NewCreature("Ghoulcaller Gisa", "{3}{B}{B}", 3, 4,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Ghoulcaller's Accomplice {1}{B}
// Creature — Human Rogue
// 2/2
// {3}{B}, Exile this card from your graveyard: Create a 2/2 black Zombie creature token. Activate only as a sorcery.
// XXX: requires activated-from-graveyard abilities
	Register("Ghoulcaller's Accomplice", func() Card {
		return NewCreature("Ghoulcaller's Accomplice", "{1}{B}", 2, 2,
			WithSubTypes("Human", "Rogue"),
		)
	})

// Ghoulraiser {1}{B}{B}
// Creature — Zombie
// 2/2
// When this creature enters, return a Zombie card at random from your graveyard to your hand.
	Register("Ghoulraiser", func() Card {
		return NewCreature("Ghoulraiser", "{1}{B}{B}", 2, 2,
			WithSubTypes("Zombie"),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("return a Zombie card at random from your graveyard to your hand",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						var zombies []Card
						for _, c := range p.Graveyard() {
							if c.HasType(TypeCreature) && c.HasSubType("Zombie") {
								zombies = append(zombies, c)
							}
						}
						if len(zombies) == 0 {
							return nil
						}
						idx := rand.Intn(len(zombies))
						chosen := zombies[idx]
						if card, ok := p.RemoveFromGraveyard(chosen.ID()); ok {
							p.AddToHand(card)
						}
						return nil
					}),
				false,
			)),
		)
	})

// Gifted Aetherborn {B}{B}
// Creature — Aetherborn Vampire
// 2/3
// Deathtouch, lifelink
	Register("Gifted Aetherborn", func() Card {
		return NewCreature("Gifted Aetherborn", "{B}{B}", 2, 3,
			WithSubTypes("Aetherborn", "Vampire"),
			WithKeyword(Deathtouch),
			WithKeyword(Lifelink),
		)
	})

// Gonti, Lord of Luxury {2}{B}{B}
// Legendary Creature — Aetherborn Rogue
// 2/3
// Deathtouch
// When Gonti enters, look at the top four cards of target opponent's library, exile one of them face down, then put the rest on the bottom of that library in a random order. You may cast that card for as long as it remains exiled, and mana of any type can be spent to cast that spell.
// XXX: requires cast-from-exile-with-any-color
	Register("Gonti, Lord of Luxury", func() Card {
		return NewCreature("Gonti, Lord of Luxury", "{2}{B}{B}", 2, 3,
			WithSubTypes("Aetherborn", "Rogue"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Deathtouch),
		)
	})

// Gravewaker {4}{B}{B}
// Creature — Bird Spirit
// 5/5
// Flying (This creature can't be blocked except by creatures with flying or reach.)
// {5}{B}{B}: Return target creature card from your graveyard to the battlefield tapped.
	Register("Gravewaker", func() Card {
		return NewCreature("Gravewaker", "{4}{B}{B}", 5, 5,
			WithSubTypes("Bird", "Spirit"),
			WithKeyword(Flying),
			WithActivatedAbility(
				FuncEffect("return target creature card from your graveyard to the battlefield tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						card, ok := p.RemoveFromGraveyard(targets[0])
						if !ok {
							return nil
						}
						perm := g.PutOnBattlefield(card, controller)
						if perm != nil {
							g.TapPermanent(perm)
						}
						return nil
					}),
				ManaCostOf("{5}{B}{B}"),
				WithTarget(TargetCardInYourGraveyard(IsCreatureCard)),
			),
		)
	})

// Gristle Grinner {4}{B}
// Creature — Zombie
// 3/3
// Whenever a creature dies, this creature gets +2/+2 until end of turn.
	Register("Gristle Grinner", func() Card {
		return NewCreature("Gristle Grinner", "{4}{B}", 3, 3,
			WithSubTypes("Zombie"),
			WithAbility(AnyCreatureDiesTrigger(
				Boost(Fixed(2), Fixed(2)).Targeting(ToSource()).Until(EndOfTurn),
				false,
			)),
		)
	})

// Harvester of Souls {4}{B}{B}
// Creature — Demon
// 5/5
// Deathtouch (Any amount of damage this deals to a creature is enough to destroy it.)
// Whenever another nontoken creature dies, you may draw a card.
	Register("Harvester of Souls", func() Card {
		return NewCreature("Harvester of Souls", "{4}{B}{B}", 5, 5,
			WithSubTypes("Demon"),
			WithKeyword(Deathtouch),
			WithAbility(NewTriggered(EvtCreatureDied, true,
				DrawCards(Fixed(1)),
			).SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
				if evt.SourceID == sourceID {
					return false
				}
				c := g.FindCardAnywhere(evt.SourceID)
				return c != nil && !c.IsToken()
			})),
		)
	})

// Kalastria Nightwatch {4}{B}
// Creature — Vampire Warrior Ally
// 4/5
// Whenever you gain life, this creature gains flying until end of turn.
// XXX: requires gain-life event trigger
	Register("Kalastria Nightwatch", func() Card {
		return NewCreature("Kalastria Nightwatch", "{4}{B}", 4, 5,
			WithSubTypes("Vampire", "Warrior", "Ally"),
		)
	})

// Kels, Fight Fixer {2}{B}{B}
// Legendary Creature — Azra Warlock
// 4/3
// Menace
// Whenever you sacrifice a creature, you may pay {U/B}. If you do, draw a card. ({U/B} can be paid with either {U} or {B}.)
// {1}, Sacrifice a creature: Kels gains indestructible until end of turn.
// XXX: requires sacrifice event and hybrid-mana costs
	Register("Kels, Fight Fixer", func() Card {
		return NewCreature("Kels, Fight Fixer", "{2}{B}{B}", 4, 3,
			WithSubTypes("Azra", "Warlock"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Menace),
			WithActivatedAbility(
				GrantKeyword(Indestructible).Targeting(ToSource()).Until(EndOfTurn),
				GenericCost(1),
				WithCost(SacrificeCreatureCost()),
			),
		)
	})

// Lawless Broker {2}{B}
// Creature — Aetherborn Rogue
// 3/2
// When this creature dies, put a +1/+1 counter on target creature you control.
	Register("Lawless Broker", func() Card {
		return NewCreature("Lawless Broker", "{2}{B}", 3, 2,
			WithSubTypes("Aetherborn", "Rogue"),
			WithAbility(NewTriggered(EvtCreatureDied, false,
				AddCounters(P1P1, Fixed(1)),
			).SetConditionData(EventSourceIsSelf{}).
				AddTarget(TargetControlledCreature())),
		)
	})

// Liliana's Elite {2}{B}
// Creature — Zombie
// 1/1
// This creature gets +1/+1 for each creature card in your graveyard.
	Register("Liliana's Elite", func() Card {
		return NewCreature("Liliana's Elite", "{2}{B}", 1, 1,
			WithSubTypes("Zombie"),
			WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				p := g.GetPlayer(src.Controller)
				if p == nil {
					return nil
				}
				count := 0
				for _, c := range p.Graveyard() {
					if c.HasType(TypeCreature) {
						count++
					}
				}
				src.BoostPT(count, count)
				return nil
			})),
		)
	})

// Liliana's Reaver {2}{B}{B}
// Creature — Zombie
// 4/3
// Deathtouch
// Whenever this creature deals combat damage to a player, that player discards a card and you create a tapped 2/2 black Zombie creature token.
	Register("Liliana's Reaver", func() Card {
		return NewCreature("Liliana's Reaver", "{2}{B}{B}", 4, 3,
			WithSubTypes("Zombie"),
			WithKeyword(Deathtouch),
			WithAbility(NewTriggered(EvtDamageDealt, false,
				FuncEffect("damaged player discards a card and you create a tapped 2/2 black Zombie token",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						var damaged Player
						if len(targets) > 0 {
							damaged = g.GetPlayer(targets[0])
						}
						if damaged != nil && len(damaged.Hand()) > 0 {
							for _, c := range damaged.ChooseCardsFromHand(1, "discard", g) {
								damaged.DiscardCard(c.ID())
							}
						}
						token := NewToken("Zombie", 2, 2, []CardType{TypeCreature}, []string{"Zombie"})
						token.SetOwner(controller)
						colors := []Color{Black}
						perm := g.PutOnBattlefield(token, controller)
						if perm != nil {
							perm.ColorOverride = &colors
							g.TapPermanent(perm)
						}
						return nil
					}),
			).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
				EventSourceIsSelfDamageToPlayer{},
				EventFlagIsTrue{},
			}})),
		)
	})

// Malakir Familiar {2}{B}
// Creature — Bat
// 2/1
// Flying, deathtouch
// Whenever you gain life, this creature gets +1/+1 until end of turn.
// XXX: requires gain-life event trigger
	Register("Malakir Familiar", func() Card {
		return NewCreature("Malakir Familiar", "{2}{B}", 2, 1,
			WithSubTypes("Bat"),
			WithKeyword(Flying),
			WithKeyword(Deathtouch),
		)
	})

// Mausoleum Turnkey {3}{B}
// Creature — Ogre Rogue
// 3/2
// When this creature enters, return target creature card of an opponent's choice from your graveyard to your hand.
// XXX: requires opponent-chooses-target primitive
	Register("Mausoleum Turnkey", func() Card {
		return NewCreature("Mausoleum Turnkey", "{3}{B}", 3, 2,
			WithSubTypes("Ogre", "Rogue"),
		)
	})

// Miasmic Mummy {1}{B}
// Creature — Zombie Jackal
// 2/2
// When this creature enters, each player discards a card.
	Register("Miasmic Mummy", func() Card {
		return NewCreature("Miasmic Mummy", "{1}{B}", 2, 2,
			WithSubTypes("Zombie", "Jackal"),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("each player discards a card",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, p := range g.AllPlayers() {
							if len(p.Hand()) == 0 {
								continue
							}
							for _, c := range p.ChooseCardsFromHand(1, "discard", g) {
								p.DiscardCard(c.ID())
							}
						}
						return nil
					}),
				false,
			)),
		)
	})

// Mire Triton {1}{B}
// Creature — Zombie Merfolk
// 2/1
// Deathtouch
// When this creature enters, mill two cards and you gain 2 life. (To mill a card, put the top card of your library into your graveyard.)
	Register("Mire Triton", func() Card {
		return NewCreature("Mire Triton", "{1}{B}", 2, 1,
			WithSubTypes("Zombie", "Merfolk"),
			WithKeyword(Deathtouch),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("mill two cards and gain 2 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						lib := p.Library()
						for i := 0; i < 2 && len(lib) > 0; i++ {
							card := lib[len(lib)-1]
							lib = lib[:len(lib)-1]
							p.AddToGraveyard(card)
						}
						p.SetLibrary(lib)
						g.PlayerGainLife(p, 2)
						return nil
					}),
				false,
			)),
		)
	})

// Nightshade Stinger {B}
// Creature — Faerie Rogue
// 1/1
// Flying
// This creature can't block.
	Register("Nightshade Stinger", func() Card {
		return NewCreature("Nightshade Stinger", "{B}", 1, 1,
			WithSubTypes("Faerie", "Rogue"),
			WithKeyword(Flying),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.RevokeAttr(sourceID, AttrCanBlock)
				return nil
			})),
		)
	})

// Nocturnal Feeder {2}{B}
// Creature — Vampire Rogue
// 2/1
// Flying
// When this creature dies, each opponent loses 2 life and you gain 2 life.
	Register("Nocturnal Feeder", func() Card {
		return NewCreature("Nocturnal Feeder", "{2}{B}", 2, 1,
			WithSubTypes("Vampire", "Rogue"),
			WithKeyword(Flying),
			WithAbility(NewTriggered(EvtCreatureDied, false,
				FuncEffect("each opponent loses 2 life and you gain 2 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, p := range g.AllPlayers() {
							if p.PlayerID() != controller {
								p.LoseLife(2)
							}
						}
						if cp := g.GetPlayer(controller); cp != nil {
							g.PlayerGainLife(cp, 2)
						}
						return nil
					}),
			).SetConditionData(EventSourceIsSelf{})),
		)
	})

// Nyxathid {1}{B}{B}
// Creature — Elemental
// 7/7
// As this creature enters, choose an opponent.
// This creature gets -1/-1 for each card in the chosen player's hand.
// XXX: requires as-enters-choose-player
	Register("Nyxathid", func() Card {
		return NewCreature("Nyxathid", "{1}{B}{B}", 7, 7,
			WithSubTypes("Elemental"),
		)
	})

// Ogre Slumlord {3}{B}{B}
// Creature — Ogre Rogue
// 3/3
// Whenever another nontoken creature dies, you may create a 1/1 black Rat creature token.
// Rats you control have deathtouch.
	Register("Ogre Slumlord", func() Card {
		return NewCreature("Ogre Slumlord", "{3}{B}{B}", 3, 3,
			WithSubTypes("Ogre", "Rogue"),
			WithAbility(NewTriggered(EvtCreatureDied, true,
				CreateColoredToken("Rat", 1, 1, []Color{Black}, []CardType{TypeCreature}, []string{"Rat"}),
			).SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
				if evt.SourceID == sourceID {
					return false
				}
				c := g.FindCardAnywhere(evt.SourceID)
				return c != nil && !c.IsToken()
			})),
			WithStaticAbility(GrantKeywordToControlled(Deathtouch, HasSubType("Rat"))),
		)
	})

// Oona's Blackguard {1}{B}
// Creature — Faerie Rogue
// 1/1
// Flying
// Each other Rogue creature you control enters with an additional +1/+1 counter on it.
// Whenever a creature you control with a +1/+1 counter on it deals combat damage to a player, that player discards a card.
// XXX: requires "enters with additional counter" replacement effect
	Register("Oona's Blackguard", func() Card {
		return NewCreature("Oona's Blackguard", "{1}{B}", 1, 1,
			WithSubTypes("Faerie", "Rogue"),
			WithKeyword(Flying),
		)
	})

// Phyrexian Broodlings {1}{B}{B}
// Creature — Phyrexian Minion
// 2/2
// {1}, Sacrifice a creature: Put a +1/+1 counter on this creature.
	Register("Phyrexian Broodlings", func() Card {
		return NewCreature("Phyrexian Broodlings", "{1}{B}{B}", 2, 2,
			WithSubTypes("Phyrexian", "Minion"),
			WithActivatedAbility(
				AddCounters(P1P1, Fixed(1)).Targeting(ToSource()),
				GenericCost(1),
				WithCost(SacrificeCreatureCost()),
			),
		)
	})

// Phyrexian Debaser {3}{B}
// Creature — Phyrexian Carrier
// 2/2
// Flying
// {T}, Sacrifice this creature: Target creature gets -2/-2 until end of turn.
	Register("Phyrexian Debaser", func() Card {
		return NewCreature("Phyrexian Debaser", "{3}{B}", 2, 2,
			WithSubTypes("Phyrexian", "Carrier"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(-2), Fixed(-2)).Until(EndOfTurn),
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

// Phyrexian Gargantua {4}{B}{B}
// Creature — Phyrexian Horror
// 4/4
// When this creature enters, you draw two cards and you lose 2 life.
	Register("Phyrexian Gargantua", func() Card {
		return NewCreature("Phyrexian Gargantua", "{4}{B}{B}", 4, 4,
			WithSubTypes("Phyrexian", "Horror"),
			WithAbility(EntersBattlefieldTrigger(
				CompositeEffects("draw two cards and lose 2 life",
					DrawCards(Fixed(2)),
					LoseLife(2),
				),
				false,
			)),
		)
	})

// Phyrexian Rager {2}{B}
// Creature — Phyrexian Horror
// 2/2
// When this creature enters, you draw a card and you lose 1 life.
	Register("Phyrexian Rager", func() Card {
		return NewCreature("Phyrexian Rager", "{2}{B}", 2, 2,
			WithSubTypes("Phyrexian", "Horror"),
			WithAbility(EntersBattlefieldTrigger(
				CompositeEffects("draw a card and lose 1 life",
					DrawCards(Fixed(1)),
					LoseLife(1),
				),
				false,
			)),
		)
	})

// Plagued Rusalka {B}
// Creature — Spirit
// 1/1
// {B}, Sacrifice a creature: Target creature gets -1/-1 until end of turn.
	Register("Plagued Rusalka", func() Card {
		return NewCreature("Plagued Rusalka", "{B}", 1, 1,
			WithSubTypes("Spirit"),
			WithActivatedAbility(
				Boost(Fixed(-1), Fixed(-1)).Until(EndOfTurn),
				ManaCostOf("{B}"),
				WithCost(SacrificeCreatureCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

// Ravenous Chupacabra {2}{B}{B}
// Creature — Beast Horror
// 2/2
// When this creature enters, destroy target creature an opponent controls.
// TODO: implement
	Register("Ravenous Chupacabra", func() Card {
		return NewCreature("Ravenous Chupacabra", "{2}{B}{B}", 2, 2,
			WithSubTypes("Beast", "Horror"),
		)
	})

// Sangromancer {2}{B}{B}
// Creature — Vampire Shaman
// 3/3
// Flying
// Whenever a creature an opponent controls dies, you may gain 3 life.
// Whenever an opponent discards a card, you may gain 3 life.
// TODO: implement
	Register("Sangromancer", func() Card {
		return NewCreature("Sangromancer", "{2}{B}{B}", 3, 3,
			WithSubTypes("Vampire", "Shaman"),
		)
	})

// Sanitarium Skeleton {B}
// Creature — Skeleton
// 1/2
// {2}{B}: Return this card from your graveyard to your hand.
// TODO: implement
	Register("Sanitarium Skeleton", func() Card {
		return NewCreature("Sanitarium Skeleton", "{B}", 1, 2,
			WithSubTypes("Skeleton"),
		)
	})

// Scourge of Nel Toth {5}{B}{B}
// Creature — Zombie Dragon
// 6/6
// Flying
// You may cast this creature from your graveyard by paying {B}{B} and sacrificing two creatures rather than paying its mana cost.
// TODO: implement
	Register("Scourge of Nel Toth", func() Card {
		return NewCreature("Scourge of Nel Toth", "{5}{B}{B}", 6, 6,
			WithSubTypes("Zombie", "Dragon"),
		)
	})

// Shambling Goblin {B}
// Creature — Zombie Goblin
// 1/1
// When this creature dies, target creature an opponent controls gets -1/-1 until end of turn.
// TODO: implement
	Register("Shambling Goblin", func() Card {
		return NewCreature("Shambling Goblin", "{B}", 1, 1,
			WithSubTypes("Zombie", "Goblin"),
		)
	})

// Sheoldred, Whispering One {5}{B}{B}
// Legendary Creature — Phyrexian Praetor
// 6/6
// Swampwalk (This creature can't be blocked as long as defending player controls a Swamp.)
// At the beginning of your upkeep, return target creature card from your graveyard to the battlefield.
// At the beginning of each opponent's upkeep, that player sacrifices a creature of their choice.
// TODO: implement
	Register("Sheoldred, Whispering One", func() Card {
		return NewCreature("Sheoldred, Whispering One", "{5}{B}{B}", 6, 6,
			WithSubTypes("Phyrexian", "Praetor"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Slate Street Ruffian {2}{B}
// Creature — Human Warrior
// 2/2
// Whenever this creature becomes blocked, defending player discards a card.
// TODO: implement
	Register("Slate Street Ruffian", func() Card {
		return NewCreature("Slate Street Ruffian", "{2}{B}", 2, 2,
			WithSubTypes("Human", "Warrior"),
		)
	})

// Swarm of Bloodflies {4}{B}
// Creature — Insect
// 0/0
// Flying
// This creature enters with two +1/+1 counters on it.
// Whenever another creature dies, put a +1/+1 counter on this creature.
// TODO: implement
	Register("Swarm of Bloodflies", func() Card {
		return NewCreature("Swarm of Bloodflies", "{4}{B}", 0, 0,
			WithSubTypes("Insect"),
		)
	})

// Tempting Witch {2}{B}
// Creature — Human Warlock
// 1/3
// When this creature enters, create a Food token. (It's an artifact with "{2}, {T}, Sacrifice this token: You gain 3 life.")
// {2}, {T}, Sacrifice a Food: Target player loses 3 life.
// TODO: implement
	Register("Tempting Witch", func() Card {
		return NewCreature("Tempting Witch", "{2}{B}", 1, 3,
			WithSubTypes("Human", "Warlock"),
		)
	})

// Tinybones, Trinket Thief {1}{B}
// Legendary Creature — Skeleton Rogue
// 1/2
// At the beginning of each end step, if an opponent discarded a card this turn, you draw a card and you lose 1 life.
// {4}{B}{B}: Each opponent with no cards in hand loses 10 life.
// TODO: implement
	Register("Tinybones, Trinket Thief", func() Card {
		return NewCreature("Tinybones, Trinket Thief", "{1}{B}", 1, 2,
			WithSubTypes("Skeleton", "Rogue"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Tithebearer Giant {5}{B}
// Creature — Giant Warrior
// 4/5
// When this creature enters, you draw a card and you lose 1 life.
// TODO: implement
	Register("Tithebearer Giant", func() Card {
		return NewCreature("Tithebearer Giant", "{5}{B}", 4, 5,
			WithSubTypes("Giant", "Warrior"),
		)
	})

// Vampire Neonate {B}
// Creature — Vampire
// 0/3
// {2}, {T}: Each opponent loses 1 life and you gain 1 life.
	Register("Vampire Neonate", func() Card {
		return NewCreature("Vampire Neonate", "{B}", 0, 3,
			WithSubTypes("Vampire"),
			WithActivatedAbility(
				FuncEffect("each opponent loses 1 life and you gain 1 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, pl := range g.AllPlayers() {
							if pl.PlayerID() != controller {
								pl.LoseLife(1)
							}
						}
						if you := g.GetPlayer(controller); you != nil {
							g.PlayerGainLife(you, 1)
						}
						return nil
					}),
				ManaCostOf("{2}"),
				WithCost(TapSourceCost()),
			),
		)
	})

// Wailing Ghoul {1}{B}
// Creature — Zombie
// 1/3
// When this creature enters, mill two cards. (Put the top two cards of your library into your graveyard.)
	Register("Wailing Ghoul", func() Card {
		return NewCreature("Wailing Ghoul", "{1}{B}", 1, 3,
			WithSubTypes("Zombie"),
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"mill two cards",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					lib := p.Library()
					for i := 0; i < 2 && len(lib) > 0; i++ {
						card := lib[len(lib)-1]
						lib = lib[:len(lib)-1]
						p.AddToGraveyard(card)
					}
					p.SetLibrary(lib)
					return nil
				},
			), false)),
		)
	})

// Wight of Precinct Six {1}{B}
// Creature — Zombie
// 1/1
// This creature gets +1/+1 for each creature card in your opponents' graveyards.
	Register("Wight of Precinct Six", func() Card {
		return NewCreature("Wight of Precinct Six", "{1}{B}", 1, 1,
			WithSubTypes("Zombie"),
			WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				count := 0
				for _, pl := range g.AllPlayers() {
					if pl.PlayerID() == src.Controller {
						continue
					}
					for _, c := range pl.Graveyard() {
						if IsCreatureCard.Match(c) {
							count++
						}
					}
				}
				src.BoostPT(count, count)
				return nil
			})),
		)
	})

// Witch of the Moors {3}{B}{B}
// Creature — Human Warlock
// 4/4
// Deathtouch
// At the beginning of your end step, if you gained life this turn, each opponent sacrifices a creature of their choice and you return up to one target creature card from your graveyard to your hand.
// XXX: requires gained-life-this-turn tracking
	Register("Witch of the Moors", func() Card {
		return NewCreature("Witch of the Moors", "{3}{B}{B}", 4, 4,
			WithSubTypes("Human", "Warlock"),
			WithKeyword(Deathtouch),
		)
	})


	// ===== RED CREATURES =====

// Ashmouth Hound {1}{R}
// Creature — Elemental Dog
// 2/1
// Whenever this creature blocks or becomes blocked by a creature, this creature deals 1 damage to that creature.
	Register("Ashmouth Hound", func() Card {
		return NewCreature("Ashmouth Hound", "{1}{R}", 2, 1,
			WithSubTypes("Elemental", "Dog"),
			WithAbility(NewTriggered(EvtDeclaredBlocker, false, FuncEffect(
				"deal 1 damage to that creature",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
					var otherID uuid.UUID
					for _, tid := range targets {
						if tid != sourceID {
							otherID = tid
						}
					}
					other := g.FindPermanent(otherID)
					if other != nil {
						g.DealDamageToPermanent(other, 1, sourceID)
					}
					return nil
				},
			)).SetConditionData(OrTriggerCond{Conditions: []TriggerConditionData{
				EventSourceIsSelf{},
				EventTargetIsSelf{},
			}})),
		)
	})

// Ball Lightning {R}{R}{R}
// Creature — Elemental
// 6/1
// Trample (This creature can deal excess combat damage to the player or planeswalker it's attacking.)
// Haste (This creature can attack and {T} as soon as it comes under your control.)
// At the beginning of the end step, sacrifice this creature.
	Register("Ball Lightning", func() Card {
		return NewCreature("Ball Lightning", "{R}{R}{R}", 6, 1,
			WithSubTypes("Elemental"),
			WithKeyword(Trample),
			WithKeyword(Haste),
			WithAbility(NewTriggered(EvtEndStep, false, FuncEffect(
				"sacrifice this creature",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
					if perm := g.FindPermanent(sourceID); perm != nil {
						g.Sacrifice(perm)
					}
					return nil
				},
			))),
		)
	})

// Beetleback Chief {2}{R}{R}
// Creature — Goblin Warrior
// 2/2
// When this creature enters, create two 1/1 red Goblin creature tokens.
	Register("Beetleback Chief", func() Card {
		return NewCreature("Beetleback Chief", "{2}{R}{R}", 2, 2,
			WithSubTypes("Goblin", "Warrior"),
			WithAbility(EntersBattlefieldTrigger(
				CreateTokens(2, "Goblin", 1, 1, []CardType{TypeCreature}, []string{"Goblin"}),
				false,
			)),
		)
	})

// Bloodrage Brawler {1}{R}
// Creature — Minotaur Warrior
// 4/3
// When this creature enters, discard a card.
	Register("Bloodrage Brawler", func() Card {
		return NewCreature("Bloodrage Brawler", "{1}{R}", 4, 3,
			WithSubTypes("Minotaur", "Warrior"),
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"discard a card",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil || len(p.Hand()) == 0 {
						return nil
					}
					chosen := p.ChooseCardsFromHand(1, "discard", g)
					for _, c := range chosen {
						p.RemoveFromHand(c.ID())
						p.AddToGraveyard(c)
					}
					return nil
				},
			), false)),
		)
	})

// Bloodrock Cyclops {2}{R}
// Creature — Cyclops
// 3/3
// This creature attacks each combat if able.
	Register("Bloodrock Cyclops", func() Card {
		return NewCreature("Bloodrock Cyclops", "{2}{R}", 3, 3,
			WithSubTypes("Cyclops"),
			WithKeyword(MustAttack),
		)
	})

// Bloodshot Trainee {3}{R}
// Creature — Goblin Warrior
// 2/3
// {T}: This creature deals 4 damage to target creature. Activate only if this creature's power is 4 or greater.
// TODO: implement
	Register("Bloodshot Trainee", func() Card {
		return NewCreature("Bloodshot Trainee", "{3}{R}", 2, 3,
			WithSubTypes("Goblin", "Warrior"),
		)
	})

// Boggart Brute {2}{R}
// Creature — Goblin Warrior
// 3/2
// Menace (This creature can't be blocked except by two or more creatures.)
	Register("Boggart Brute", func() Card {
		return NewCreature("Boggart Brute", "{2}{R}", 3, 2,
			WithSubTypes("Goblin", "Warrior"),
			WithKeyword(Menace),
		)
	})

// Borderland Marauder {1}{R}
// Creature — Human Warrior
// 1/2
// Whenever this creature attacks, it gets +2/+0 until end of turn.
	Register("Borderland Marauder", func() Card {
		return NewCreature("Borderland Marauder", "{1}{R}", 1, 2,
			WithSubTypes("Human", "Warrior"),
			WithStaticAbility(BoostSelf(2, 0, WhileSourceAttacking)),
		)
	})

// Borderland Minotaur {2}{R}{R}
// Creature — Minotaur Warrior
// 4/3
	Register("Borderland Minotaur", func() Card {
		return NewCreature("Borderland Minotaur", "{2}{R}{R}", 4, 3,
			WithSubTypes("Minotaur", "Warrior"),
		)
	})

// Chained Brute {1}{R}
// Creature — Devil
// 4/3
// This creature doesn't untap during your untap step.
// {1}, Sacrifice another creature: Untap this creature. Activate only during your turn.
// XXX: requires sorcery-speed activation restriction; see CR 605
	Register("Chained Brute", func() Card {
		return NewCreature("Chained Brute", "{1}{R}", 4, 3,
			WithSubTypes("Devil"),
			WithKeyword(DoesNotUntapKW),
			WithActivatedAbility(
				UntapSource(),
				ManaCostOf("{1}"),
				WithCost(SacrificeMatchingCost(NewPermanentFilter("another creature", func(p *Permanent, _ *Game) bool {
					return p.HasType(TypeCreature)
				}), "Sacrifice another creature")),
			),
		)
	})

// Charmbreaker Devils {5}{R}
// Creature — Devil
// 4/4
// At the beginning of your upkeep, return an instant or sorcery card at random from your graveyard to your hand.
// Whenever you cast an instant or sorcery spell, this creature gets +4/+0 until end of turn.
// XXX: requires random-card-from-graveyard primitive
	Register("Charmbreaker Devils", func() Card {
		return NewCreature("Charmbreaker Devils", "{5}{R}", 4, 4,
			WithSubTypes("Devil"),
			WithAbility(WheneverYouCastSpellTrigger(
				Boost(Fixed(4), Fixed(0)).Targeting(ToSource()),
				false,
				NewCardFilter("instant or sorcery", func(c Card) bool {
					return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
				}),
			)),
		)
	})

// Cinder Elemental {3}{R}
// Creature — Elemental
// 2/2
// {X}{R}, {T}, Sacrifice this creature: It deals X damage to any target.
	Register("Cinder Elemental", func() Card {
		return NewCreature("Cinder Elemental", "{3}{R}", 2, 2,
			WithSubTypes("Elemental"),
			WithActivatedAbility(
				DealDamage(XValue()),
				ManaCostOf("{X}{R}"),
				WithCost(TapSourceCost()),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

// Dragon Hatchling {1}{R}
// Creature — Dragon
// 0/1
// Flying
// {R}: This creature gets +1/+0 until end of turn.
	Register("Dragon Hatchling", func() Card {
		return NewCreature("Dragon Hatchling", "{1}{R}", 0, 1,
			WithSubTypes("Dragon"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
		)
	})

// Dragonlord's Servant {1}{R}
// Creature — Goblin Shaman
// 1/3
// Dragon spells you cast cost {1} less to cast.
// XXX: requires subtype-based spell cost reduction
	Register("Dragonlord's Servant", func() Card {
		return NewCreature("Dragonlord's Servant", "{1}{R}", 1, 3,
			WithSubTypes("Goblin", "Shaman"),
		)
	})

// Dragonspeaker Shaman {1}{R}{R}
// Creature — Human Barbarian Shaman
// 2/2
// Dragon spells you cast cost {2} less to cast.
// XXX: requires subtype-based spell cost reduction
	Register("Dragonspeaker Shaman", func() Card {
		return NewCreature("Dragonspeaker Shaman", "{1}{R}{R}", 2, 2,
			WithSubTypes("Human", "Barbarian", "Shaman"),
		)
	})

// Dualcaster Mage {1}{R}{R}
// Creature — Human Wizard
// 2/2
// Flash
// When this creature enters, copy target instant or sorcery spell. You may choose new targets for the copy.
// XXX: requires spell-copy primitive
	Register("Dualcaster Mage", func() Card {
		return NewCreature("Dualcaster Mage", "{1}{R}{R}", 2, 2,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Etali, Primal Storm {4}{R}{R}
// Legendary Creature — Elder Dinosaur
// 6/6
// Whenever Etali attacks, exile the top card of each player's library, then you may cast any number of spells from among those cards without paying their mana costs.
// XXX: requires cast-from-exile alternate-cost
	Register("Etali, Primal Storm", func() Card {
		return NewCreature("Etali, Primal Storm", "{4}{R}{R}", 6, 6,
			WithSubTypes("Elder", "Dinosaur"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Fanatical Firebrand {R}
// Creature — Goblin Pirate
// 1/1
// Haste (This creature can attack and {T} as soon as it comes under your control.)
// {T}, Sacrifice this creature: It deals 1 damage to any target.
	Register("Fanatical Firebrand", func() Card {
		return NewCreature("Fanatical Firebrand", "{R}", 1, 1,
			WithSubTypes("Goblin", "Pirate"),
			WithKeyword(Haste),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

// Flametongue Kavu {3}{R}
// Creature — Kavu
// 4/2
// When this creature enters, it deals 4 damage to target creature.
	Register("Flametongue Kavu", func() Card {
		return NewCreature("Flametongue Kavu", "{3}{R}", 4, 2,
			WithSubTypes("Kavu"),
			WithETBEffect(DealDamage(Fixed(4))),
		)
	})

// Forge Devil {R}
// Creature — Devil
// 1/1
// When this creature enters, it deals 1 damage to target creature and 1 damage to you.
	Register("Forge Devil", func() Card {
		return NewCreature("Forge Devil", "{R}", 1, 1,
			WithSubTypes("Devil"),
			WithETBEffect(CompositeEffects("deal 1 to target creature and 1 to you",
				DealDamage(Fixed(1)),
				DealDamageToPlayers(Fixed(1), SelectController()),
			)),
		)
	})

// Furnace Whelp {2}{R}{R}
// Creature — Dragon
// 2/2
// Flying
// {R}: This creature gets +1/+0 until end of turn.
	Register("Furnace Whelp", func() Card {
		return NewCreature("Furnace Whelp", "{2}{R}{R}", 2, 2,
			WithSubTypes("Dragon"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
		)
	})

// Goblin Chieftain {1}{R}{R}
// Creature — Goblin
// 2/2
// Haste (This creature can attack and {T} as soon as it comes under your control.)
// Other Goblin creatures you control get +1/+1 and have haste.
	Register("Goblin Chieftain", func() Card {
		return NewCreature("Goblin Chieftain", "{1}{R}{R}", 2, 2,
			WithSubTypes("Goblin"),
			WithKeyword(Haste),
			WithStaticAbility(
				BoostControlledCreatures(1, 1, HasSubType("Goblin")),
				GrantKeywordToAll(Haste, HasSubType("Goblin")),
			),
		)
	})

// Goblin Commando {4}{R}
// Creature — Goblin
// 2/2
// When this creature enters, it deals 2 damage to target creature.
	Register("Goblin Commando", func() Card {
		return NewCreature("Goblin Commando", "{4}{R}", 2, 2,
			WithSubTypes("Goblin"),
			WithETBEffect(DealDamage(Fixed(2))),
		)
	})

// Goblin Goon {3}{R}
// Creature — Goblin Mutant
// 6/6
// This creature can't attack unless you control more creatures than defending player.
// This creature can't block unless you control more creatures than attacking player.
// XXX: requires power-comparison restrictions (creature-count attack/block constraints)
	Register("Goblin Goon", func() Card {
		return NewCreature("Goblin Goon", "{3}{R}", 6, 6,
			WithSubTypes("Goblin", "Mutant"),
		)
	})

// Goblin Instigator {1}{R}
// Creature — Goblin Rogue
// 1/1
// When this creature enters, create a 1/1 red Goblin creature token.
	Register("Goblin Instigator", func() Card {
		return NewCreature("Goblin Instigator", "{1}{R}", 1, 1,
			WithSubTypes("Goblin", "Rogue"),
			WithAbility(EntersBattlefieldTrigger(
				CreateToken("Goblin", 1, 1, []CardType{TypeCreature}, []string{"Goblin"}),
				false,
			)),
		)
	})

// Goblin Shortcutter {1}{R}
// Creature — Goblin Scout
// 2/1
// When this creature enters, target creature can't block this turn.
	Register("Goblin Shortcutter", func() Card {
		return NewCreature("Goblin Shortcutter", "{1}{R}", 2, 1,
			WithSubTypes("Goblin", "Scout"),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("target creature can't block this turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, _, _ uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 {
							g.AddContinuousEffect(PreventBlockingUntilEndOfCombat(targets[0]))
						}
						return nil
					}),
				false,
			).AddTarget(TargetCreature())),
		)
	})

// Grim Lavamancer {R}
// Creature — Human Wizard
// 1/1
// {R}, {T}, Exile two cards from your graveyard: This creature deals 2 damage to any target.
	Register("Grim Lavamancer", func() Card {
		return NewCreature("Grim Lavamancer", "{R}", 1, 1,
			WithSubTypes("Human", "Wizard"),
			WithActivatedAbility(
				DealDamage(Fixed(2)),
				ManaCostOf("{R}"),
				WithCost(TapSourceCost()),
				WithCost(ExileFromGraveyardCost(2)),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

// Hamletback Goliath {6}{R}
// Creature — Giant Warrior
// 6/6
// Whenever another creature enters, you may put X +1/+1 counters on this creature, where X is that creature's power.
	Register("Hamletback Goliath", func() Card {
		return NewCreature("Hamletback Goliath", "{6}{R}", 6, 6,
			WithSubTypes("Giant", "Warrior"),
			WithAbility(NewTriggered(EvtEntersBattlefield, true,
				FuncEffect("put X +1/+1 counters on this creature where X is that creature's power",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						entered := g.FindPermanent(targets[0])
						src := g.FindPermanent(sourceID)
						if entered == nil || src == nil {
							return nil
						}
						x := entered.CurrentPower(g)
						if x > 0 {
							src.AddCounter(P1P1, x)
						}
						return nil
					}),
			).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
				EventSourceNotSelf{},
				EventSourceHasType{Type: TypeCreature},
			}})),
		)
	})

// Hellrider {2}{R}{R}
// Creature — Devil
// 3/3
// Haste
// Whenever a creature you control attacks, this creature deals 1 damage to the player or planeswalker it's attacking.
	Register("Hellrider", func() Card {
		return NewCreature("Hellrider", "{2}{R}{R}", 3, 3,
			WithSubTypes("Devil"),
			WithKeyword(Haste),
			WithAbility(NewTriggered(EvtDeclaredAttacker, false,
				FuncEffect("deal 1 damage to the player or planeswalker it's attacking",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						atkID := targets[0]
						for _, group := range g.CombatGroups() {
							if group.AttackerID == atkID {
								if def := g.GetPlayer(group.DefenderID); def != nil {
									g.DealDamageToPlayer(def, 1, sourceID)
								}
								return nil
							}
						}
						return nil
					}),
			).SetConditionData(EventSourceControlledByController{})),
		)
	})

// Inferno Hellion {3}{R}
// Creature — Hellion
// 7/3
// Trample (This creature can deal excess combat damage to the player or planeswalker it's attacking.)
// At the beginning of each end step, if this creature attacked or blocked this turn, its owner shuffles it into their library.
// TODO: implement
	Register("Inferno Hellion", func() Card {
		return NewCreature("Inferno Hellion", "{3}{R}", 7, 3,
			WithSubTypes("Hellion"),
		)
	})

// Kiln Fiend {1}{R}
// Creature — Elemental Beast
// 1/2
// Whenever you cast an instant or sorcery spell, this creature gets +3/+0 until end of turn.
// TODO: implement
	Register("Kiln Fiend", func() Card {
		return NewCreature("Kiln Fiend", "{1}{R}", 1, 2,
			WithSubTypes("Elemental", "Beast"),
		)
	})

// Krenko, Mob Boss {2}{R}{R}
// Legendary Creature — Goblin Warrior
// 3/3
// {T}: Create X 1/1 red Goblin creature tokens, where X is the number of Goblins you control.
// TODO: implement
	Register("Krenko, Mob Boss", func() Card {
		return NewCreature("Krenko, Mob Boss", "{2}{R}{R}", 3, 3,
			WithSubTypes("Goblin", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Lathliss, Dragon Queen {4}{R}{R}
// Legendary Creature — Dragon
// 6/6
// Flying
// Whenever another nontoken Dragon you control enters, create a 5/5 red Dragon creature token with flying.
// {1}{R}: Dragons you control get +1/+0 until end of turn.
// TODO: implement
	Register("Lathliss, Dragon Queen", func() Card {
		return NewCreature("Lathliss, Dragon Queen", "{4}{R}{R}", 6, 6,
			WithSubTypes("Dragon"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Lightning Elemental {3}{R}
// Creature — Elemental
// 4/1
// Haste (This creature can attack and {T} as soon as it comes under your control.)
// TODO: implement
	Register("Lightning Elemental", func() Card {
		return NewCreature("Lightning Elemental", "{3}{R}", 4, 1,
			WithSubTypes("Elemental"),
		)
	})

// Lightning Phoenix {2}{R}
// Creature — Phoenix
// 2/2
// Flying, haste
// This creature can't block.
// At the beginning of your end step, if an opponent was dealt 3 or more damage this turn, you may pay {R}. If you do, return this card from your graveyard to the battlefield.
// TODO: implement
	Register("Lightning Phoenix", func() Card {
		return NewCreature("Lightning Phoenix", "{2}{R}", 2, 2,
			WithSubTypes("Phoenix"),
		)
	})

// Lightning Shrieker {4}{R}
// Creature — Dragon
// 5/5
// Flying, trample, haste
// At the beginning of the end step, this creature's owner shuffles it into their library.
// TODO: implement
	Register("Lightning Shrieker", func() Card {
		return NewCreature("Lightning Shrieker", "{4}{R}", 5, 5,
			WithSubTypes("Dragon"),
		)
	})

// Lightning Visionary {1}{R}
// Creature — Minotaur Shaman
// 2/1
// Prowess (Whenever you cast a noncreature spell, this creature gets +1/+1 until end of turn.)
// TODO: implement
	Register("Lightning Visionary", func() Card {
		return NewCreature("Lightning Visionary", "{1}{R}", 2, 1,
			WithSubTypes("Minotaur", "Shaman"),
		)
	})

// Living Lightning {3}{R}
// Creature — Elemental Shaman
// 3/2
// When this creature dies, return target instant or sorcery card from your graveyard to your hand.
// TODO: implement
	Register("Living Lightning", func() Card {
		return NewCreature("Living Lightning", "{3}{R}", 3, 2,
			WithSubTypes("Elemental", "Shaman"),
		)
	})

// Minotaur Skullcleaver {2}{R}
// Creature — Minotaur Berserker
// 2/2
// Haste
// When this creature enters, it gets +2/+0 until end of turn.
// TODO: implement
	Register("Minotaur Skullcleaver", func() Card {
		return NewCreature("Minotaur Skullcleaver", "{2}{R}", 2, 2,
			WithSubTypes("Minotaur", "Berserker"),
		)
	})

// Minotaur Sureshot {2}{R}
// Creature — Minotaur Archer
// 2/3
// Reach (This creature can block creatures with flying.)
// {1}{R}: This creature gets +1/+0 until end of turn.
// TODO: implement
	Register("Minotaur Sureshot", func() Card {
		return NewCreature("Minotaur Sureshot", "{2}{R}", 2, 3,
			WithSubTypes("Minotaur", "Archer"),
		)
	})

// Molten Ravager {2}{R}
// Creature — Elemental
// 0/4
// {R}: This creature gets +1/+0 until end of turn.
// TODO: implement
	Register("Molten Ravager", func() Card {
		return NewCreature("Molten Ravager", "{2}{R}", 0, 4,
			WithSubTypes("Elemental"),
		)
	})

// Muxus, Goblin Grandee {4}{R}{R}
// Legendary Creature — Goblin Noble
// 4/4
// When Muxus enters, reveal the top six cards of your library. Put all Goblin creature cards with mana value 5 or less from among them onto the battlefield and the rest on the bottom of your library in a random order.
// Whenever Muxus attacks, it gets +1/+1 until end of turn for each other Goblin you control.
// TODO: implement
	Register("Muxus, Goblin Grandee", func() Card {
		return NewCreature("Muxus, Goblin Grandee", "{4}{R}{R}", 4, 4,
			WithSubTypes("Goblin", "Noble"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Ornery Goblin {1}{R}
// Creature — Goblin Warrior
// 2/1
// Whenever this creature blocks or becomes blocked by a creature, this creature deals 1 damage to that creature.
// TODO: implement
	Register("Ornery Goblin", func() Card {
		return NewCreature("Ornery Goblin", "{1}{R}", 2, 1,
			WithSubTypes("Goblin", "Warrior"),
		)
	})

// Pyroclastic Elemental {3}{R}{R}
// Creature — Elemental
// 5/4
// {1}{R}{R}: This creature deals 1 damage to target player.
// TODO: implement
	Register("Pyroclastic Elemental", func() Card {
		return NewCreature("Pyroclastic Elemental", "{3}{R}{R}", 5, 4,
			WithSubTypes("Elemental"),
		)
	})

// Rageblood Shaman {1}{R}{R}
// Creature — Minotaur Shaman
// 2/3
// Trample
// Other Minotaur creatures you control get +1/+1 and have trample.
// TODO: implement
	Register("Rageblood Shaman", func() Card {
		return NewCreature("Rageblood Shaman", "{1}{R}{R}", 2, 3,
			WithSubTypes("Minotaur", "Shaman"),
		)
	})

// Rapacious Dragon {4}{R}
// Creature — Dragon
// 3/3
// Flying
// When this creature enters, create two Treasure tokens. (They're artifacts with "{T}, Sacrifice this token: Add one mana of any color.")
// TODO: implement
	Register("Rapacious Dragon", func() Card {
		return NewCreature("Rapacious Dragon", "{4}{R}", 3, 3,
			WithSubTypes("Dragon"),
		)
	})

// Seismic Elemental {3}{R}{R}
// Creature — Elemental
// 4/4
// When this creature enters, creatures without flying can't block this turn.
// TODO: implement
	Register("Seismic Elemental", func() Card {
		return NewCreature("Seismic Elemental", "{3}{R}{R}", 4, 4,
			WithSubTypes("Elemental"),
		)
	})

// Sethron, Hurloon General {3}{R}{R}
// Legendary Creature — Minotaur Warrior
// 4/4
// Whenever Sethron or another nontoken Minotaur you control enters, create a 2/3 red Minotaur creature token.
// {2}{B/R}: Minotaurs you control get +1/+0 and gain menace and haste until end of turn. ({B/R} can be paid with either {B} or {R}.)
// TODO: implement
	Register("Sethron, Hurloon General", func() Card {
		return NewCreature("Sethron, Hurloon General", "{3}{R}{R}", 4, 4,
			WithSubTypes("Minotaur", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Sin Prodder {2}{R}
// Creature — Devil
// 3/2
// Menace
// At the beginning of your upkeep, reveal the top card of your library. Any opponent may have you put that card into your graveyard. If a player does, this creature deals damage to that player equal to that card's mana value. Otherwise, put that card into your hand.
// TODO: implement
	Register("Sin Prodder", func() Card {
		return NewCreature("Sin Prodder", "{2}{R}", 3, 2,
			WithSubTypes("Devil"),
		)
	})

// Spiteful Prankster {2}{R}
// Creature — Devil
// 3/2
// During your turn, this creature has first strike.
// Whenever another creature dies, this creature deals 1 damage to target player or planeswalker.
// TODO: implement
	Register("Spiteful Prankster", func() Card {
		return NewCreature("Spiteful Prankster", "{2}{R}", 3, 2,
			WithSubTypes("Devil"),
		)
	})

// Thermo-Alchemist {1}{R}
// Creature — Human Shaman
// 0/3
// Defender
// {T}: This creature deals 1 damage to each opponent.
// Whenever you cast an instant or sorcery spell, untap this creature.
// TODO: implement
	Register("Thermo-Alchemist", func() Card {
		return NewCreature("Thermo-Alchemist", "{1}{R}", 0, 3,
			WithSubTypes("Human", "Shaman"),
		)
	})

// Tibalt's Rager {1}{R}
// Creature — Devil
// 1/2
// When this creature dies, it deals 1 damage to any target.
// {1}{R}: This creature gets +2/+0 until end of turn.
// TODO: implement
	Register("Tibalt's Rager", func() Card {
		return NewCreature("Tibalt's Rager", "{1}{R}", 1, 2,
			WithSubTypes("Devil"),
		)
	})

// Torch Fiend {1}{R}
// Creature — Devil
// 2/1
// {R}, Sacrifice this creature: Destroy target artifact.
// TODO: implement
	Register("Torch Fiend", func() Card {
		return NewCreature("Torch Fiend", "{1}{R}", 2, 1,
			WithSubTypes("Devil"),
		)
	})

// Volley Veteran {3}{R}
// Creature — Goblin Warrior
// 4/2
// When this creature enters, it deals damage to target creature an opponent controls equal to the number of Goblins you control.
// TODO: implement
	Register("Volley Veteran", func() Card {
		return NewCreature("Volley Veteran", "{3}{R}", 4, 2,
			WithSubTypes("Goblin", "Warrior"),
		)
	})

// Warfire Javelineer {3}{R}
// Creature — Minotaur Warrior
// 2/3
// When this creature enters, it deals X damage to target creature an opponent controls, where X is the number of instant and sorcery cards in your graveyard.
// TODO: implement
	Register("Warfire Javelineer", func() Card {
		return NewCreature("Warfire Javelineer", "{3}{R}", 2, 3,
			WithSubTypes("Minotaur", "Warrior"),
		)
	})

// Weaver of Lightning {2}{R}
// Creature — Human Shaman
// 1/4
// Reach (This creature can block creatures with flying.)
// Whenever you cast an instant or sorcery spell, this creature deals 1 damage to target creature an opponent controls.
// TODO: implement
	Register("Weaver of Lightning", func() Card {
		return NewCreature("Weaver of Lightning", "{2}{R}", 1, 4,
			WithSubTypes("Human", "Shaman"),
		)
	})

// Young Pyromancer {1}{R}
// Creature — Human Shaman
// 2/1
// Whenever you cast an instant or sorcery spell, create a 1/1 red Elemental creature token.
// TODO: implement
	Register("Young Pyromancer", func() Card {
		return NewCreature("Young Pyromancer", "{1}{R}", 2, 1,
			WithSubTypes("Human", "Shaman"),
		)
	})

// Zurzoth, Chaos Rider {2}{R}
// Legendary Creature — Devil
// 2/3
// Whenever an opponent draws their first card each turn, if it's not their turn, you create a 1/1 red Devil creature token with "When this token dies, it deals 1 damage to any target."
// Whenever one or more Devils you control attack one or more players, you and those players each draw a card, then discard a card at random.
// TODO: implement
	Register("Zurzoth, Chaos Rider", func() Card {
		return NewCreature("Zurzoth, Chaos Rider", "{2}{R}", 2, 3,
			WithSubTypes("Devil"),
			WithSuperTypes(SuperLegendary),
		)
	})


	// ===== GREEN CREATURES =====

// Affectionate Indrik {5}{G}
// Creature — Beast
// 4/4
// When this creature enters, you may have it fight target creature you don't control. (Each deals damage equal to its power to the other.)
// TODO: implement
	Register("Affectionate Indrik", func() Card {
		return NewCreature("Affectionate Indrik", "{5}{G}", 4, 4,
			WithSubTypes("Beast"),
		)
	})

// Allosaurus Shepherd {G}
// Creature — Elf Shaman
// 1/1
// This spell can't be countered.
// Green spells you control can't be countered.
// {4}{G}{G}: Until end of turn, each Elf creature you control has base power and toughness 5/5 and becomes a Dinosaur in addition to its other creature types.
// TODO: implement
	Register("Allosaurus Shepherd", func() Card {
		return NewCreature("Allosaurus Shepherd", "{G}", 1, 1,
			WithSubTypes("Elf", "Shaman"),
		)
	})

// Ambassador Oak {3}{G}
// Creature — Treefolk Warrior
// 3/3
// When this creature enters, create a 1/1 green Elf Warrior creature token.
// TODO: implement
	Register("Ambassador Oak", func() Card {
		return NewCreature("Ambassador Oak", "{3}{G}", 3, 3,
			WithSubTypes("Treefolk", "Warrior"),
		)
	})

// Armorcraft Judge {3}{G}
// Creature — Elf Artificer
// 3/3
// When this creature enters, draw a card for each creature you control with a +1/+1 counter on it.
// TODO: implement
	Register("Armorcraft Judge", func() Card {
		return NewCreature("Armorcraft Judge", "{3}{G}", 3, 3,
			WithSubTypes("Elf", "Artificer"),
		)
	})

// Awakener Druid {2}{G}
// Creature — Human Druid
// 1/1
// When this creature enters, target Forest becomes a 4/5 green Treefolk creature for as long as this creature remains on the battlefield. It's still a land.
// TODO: implement
	Register("Awakener Druid", func() Card {
		return NewCreature("Awakener Druid", "{2}{G}", 1, 1,
			WithSubTypes("Human", "Druid"),
		)
	})

// Brindle Shoat {1}{G}
// Creature — Boar
// 1/1
// When this creature dies, create a 3/3 green Boar creature token.
// TODO: implement
	Register("Brindle Shoat", func() Card {
		return NewCreature("Brindle Shoat", "{1}{G}", 1, 1,
			WithSubTypes("Boar"),
		)
	})

// Brushstrider {1}{G}
// Creature — Beast
// 3/1
// Vigilance (Attacking doesn't cause this creature to tap.)
// TODO: implement
	Register("Brushstrider", func() Card {
		return NewCreature("Brushstrider", "{1}{G}", 3, 1,
			WithSubTypes("Beast"),
		)
	})

// Carven Caryatid {1}{G}{G}
// Creature — Spirit
// 2/5
// Defender (This creature can't attack.)
// When this creature enters, draw a card.
// TODO: implement
	Register("Carven Caryatid", func() Card {
		return NewCreature("Carven Caryatid", "{1}{G}{G}", 2, 5,
			WithSubTypes("Spirit"),
		)
	})

// Champion of Lambholt {1}{G}{G}
// Creature — Human Warrior
// 1/1
// Creatures with power less than this creature's power can't block creatures you control.
// Whenever another creature you control enters, put a +1/+1 counter on this creature.
// TODO: implement
	Register("Champion of Lambholt", func() Card {
		return NewCreature("Champion of Lambholt", "{1}{G}{G}", 1, 1,
			WithSubTypes("Human", "Warrior"),
		)
	})

// Craterhoof Behemoth {5}{G}{G}{G}
// Creature — Beast
// 5/5
// Haste
// When this creature enters, creatures you control gain trample and get +X/+X until end of turn, where X is the number of creatures you control.
	Register("Craterhoof Behemoth", func() Card {
		return NewCreature("Craterhoof Behemoth", "{5}{G}{G}{G}", 5, 5,
			WithSubTypes("Beast"),
			WithKeyword(Haste),
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"creatures you control gain trample and get +X/+X until end of turn, where X is the number of creatures you control",
				EffectProperties{Outcome: OutcomeBenefit, Mass: true},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					x := g.CountBattlefield(And(IsCreature, ControlledBy(controller)))
					for _, p := range g.FilterBattlefield(And(IsCreature, ControlledBy(controller))) {
						b := TemporaryBoost(p.ID(), x, x)
						b.SetSourceID(sourceID)
						g.AddContinuousEffect(b)
						k := TemporaryKeyword(p.ID(), Trample)
						k.SetSourceID(sourceID)
						g.AddContinuousEffect(k)
					}
					g.ApplyContinuousEffects()
					return nil
				},
			), false)),
		)
	})

// Dawntreader Elk {1}{G}
// Creature — Elk
// 2/2
// {G}, Sacrifice this creature: Search your library for a basic land card, put it onto the battlefield tapped, then shuffle.
// TODO: implement
	Register("Dawntreader Elk", func() Card {
		return NewCreature("Dawntreader Elk", "{1}{G}", 2, 2,
			WithSubTypes("Elk"),
		)
	})

// Drover of the Mighty {1}{G}
// Creature — Human Druid
// 1/1
// This creature gets +2/+2 as long as you control a Dinosaur.
// {T}: Add one mana of any color.
// TODO: implement
	Register("Drover of the Mighty", func() Card {
		return NewCreature("Drover of the Mighty", "{1}{G}", 1, 1,
			WithSubTypes("Human", "Druid"),
		)
	})

// Dwynen's Elite {1}{G}
// Creature — Elf Warrior
// 2/2
// When this creature enters, if you control another Elf, create a 1/1 green Elf Warrior creature token.
// TODO: implement
	Register("Dwynen's Elite", func() Card {
		return NewCreature("Dwynen's Elite", "{1}{G}", 2, 2,
			WithSubTypes("Elf", "Warrior"),
		)
	})

// Elvish Archdruid {1}{G}{G}
// Creature — Elf Druid
// 2/2
// Other Elf creatures you control get +1/+1.
// {T}: Add {G} for each Elf you control.
// TODO: implement
	Register("Elvish Archdruid", func() Card {
		return NewCreature("Elvish Archdruid", "{1}{G}{G}", 2, 2,
			WithSubTypes("Elf", "Druid"),
		)
	})

// Fa'adiyah Seer {1}{G}
// Creature — Human Shaman
// 1/1
// {T}: Draw a card and reveal it. If it isn't a land card, discard it.
// TODO: implement
	Register("Fa'adiyah Seer", func() Card {
		return NewCreature("Fa'adiyah Seer", "{1}{G}", 1, 1,
			WithSubTypes("Human", "Shaman"),
		)
	})

// Feral Hydra {X}{G}
// Creature — Hydra Beast
// 0/0
// This creature enters with X +1/+1 counters on it.
// {3}: Put a +1/+1 counter on this creature. Any player may activate this ability.
// TODO: implement
	Register("Feral Hydra", func() Card {
		return NewCreature("Feral Hydra", "{X}{G}", 0, 0,
			WithSubTypes("Hydra", "Beast"),
		)
	})

// Feral Prowler {1}{G}
// Creature — Cat
// 1/3
// When this creature dies, draw a card.
// TODO: implement
	Register("Feral Prowler", func() Card {
		return NewCreature("Feral Prowler", "{1}{G}", 1, 3,
			WithSubTypes("Cat"),
		)
	})

// Fertilid {2}{G}
// Creature — Elemental
// 0/0
// This creature enters with two +1/+1 counters on it.
// {1}{G}, Remove a +1/+1 counter from this creature: Target player searches their library for a basic land card, puts it onto the battlefield tapped, then shuffles.
// TODO: implement
	Register("Fertilid", func() Card {
		return NewCreature("Fertilid", "{2}{G}", 0, 0,
			WithSubTypes("Elemental"),
		)
	})

// Ghalta, Primal Hunger {10}{G}{G}
// Legendary Creature — Elder Dinosaur
// 12/12
// This spell costs {X} less to cast, where X is the total power of creatures you control.
// Trample (This creature can deal excess combat damage to the player or planeswalker it's attacking.)
// TODO: implement
	Register("Ghalta, Primal Hunger", func() Card {
		return NewCreature("Ghalta, Primal Hunger", "{10}{G}{G}", 12, 12,
			WithSubTypes("Elder", "Dinosaur"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Ghirapur Guide {2}{G}
// Creature — Elf Scout
// 3/2
// {2}{G}: Target creature you control can't be blocked by creatures with power 2 or less this turn.
// TODO: implement
	Register("Ghirapur Guide", func() Card {
		return NewCreature("Ghirapur Guide", "{2}{G}", 3, 2,
			WithSubTypes("Elf", "Scout"),
		)
	})

// Grave Bramble {1}{G}{G}
// Creature — Plant
// 3/4
// Defender, protection from Zombies
// TODO: implement
	Register("Grave Bramble", func() Card {
		return NewCreature("Grave Bramble", "{1}{G}{G}", 3, 4,
			WithSubTypes("Plant"),
		)
	})

// Initiate's Companion {1}{G}
// Creature — Cat
// 3/1
// Whenever this creature deals combat damage to a player, untap target creature or land.
// TODO: implement
	Register("Initiate's Companion", func() Card {
		return NewCreature("Initiate's Companion", "{1}{G}", 3, 1,
			WithSubTypes("Cat"),
		)
	})

// Ironshell Beetle {1}{G}
// Creature — Insect
// 1/1
// When this creature enters, put a +1/+1 counter on target creature.
// TODO: implement
	Register("Ironshell Beetle", func() Card {
		return NewCreature("Ironshell Beetle", "{1}{G}", 1, 1,
			WithSubTypes("Insect"),
		)
	})

// Keeper of Fables {3}{G}{G}
// Creature — Cat
// 4/5
// Whenever one or more non-Human creatures you control deal combat damage to a player, draw a card.
// TODO: implement
	Register("Keeper of Fables", func() Card {
		return NewCreature("Keeper of Fables", "{3}{G}{G}", 4, 5,
			WithSubTypes("Cat"),
		)
	})

// Leaf Gilder {1}{G}
// Creature — Elf Druid
// 2/1
// {T}: Add {G}.
// TODO: implement
	Register("Leaf Gilder", func() Card {
		return NewCreature("Leaf Gilder", "{1}{G}", 2, 1,
			WithSubTypes("Elf", "Druid"),
		)
	})

// Nessian Hornbeetle {1}{G}
// Creature — Insect
// 2/2
// At the beginning of combat on your turn, if you control another creature with power 4 or greater, put a +1/+1 counter on this creature.
// TODO: implement
	Register("Nessian Hornbeetle", func() Card {
		return NewCreature("Nessian Hornbeetle", "{1}{G}", 2, 2,
			WithSubTypes("Insect"),
		)
	})

// Neyith of the Dire Hunt {2}{G}{G}
// Legendary Creature — Human Warrior
// 3/3
// Whenever one or more creatures you control fight or become blocked, draw a card.
// At the beginning of combat on your turn, you may pay {2}{R/G}. If you do, double target creature's power until end of turn. That creature must be blocked this combat if able. ({R/G} can be paid with either {R} or {G}.)
// TODO: implement
	Register("Neyith of the Dire Hunt", func() Card {
		return NewCreature("Neyith of the Dire Hunt", "{2}{G}{G}", 3, 3,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Oracle of Mul Daya {3}{G}
// Creature — Elf Shaman
// 2/2
// You may play an additional land on each of your turns.
// Play with the top card of your library revealed.
// You may play lands from the top of your library.
// TODO: implement
	Register("Oracle of Mul Daya", func() Card {
		return NewCreature("Oracle of Mul Daya", "{3}{G}", 2, 2,
			WithSubTypes("Elf", "Shaman"),
		)
	})

// Orazca Frillback {2}{G}
// Creature — Dinosaur
// 4/2
// TODO: implement
	Register("Orazca Frillback", func() Card {
		return NewCreature("Orazca Frillback", "{2}{G}", 4, 2,
			WithSubTypes("Dinosaur"),
		)
	})

// Overgrown Battlement {1}{G}
// Creature — Wall
// 0/4
// Defender
// {T}: Add {G} for each creature you control with defender.
// TODO: implement
	Register("Overgrown Battlement", func() Card {
		return NewCreature("Overgrown Battlement", "{1}{G}", 0, 4,
			WithSubTypes("Wall"),
		)
	})

// Penumbra Bobcat {2}{G}
// Creature — Cat
// 2/1
// When this creature dies, create a 2/1 black Cat creature token.
// TODO: implement
	Register("Penumbra Bobcat", func() Card {
		return NewCreature("Penumbra Bobcat", "{2}{G}", 2, 1,
			WithSubTypes("Cat"),
		)
	})

// Pouncing Cheetah {2}{G}
// Creature — Cat
// 3/2
// Flash
// TODO: implement
	Register("Pouncing Cheetah", func() Card {
		return NewCreature("Pouncing Cheetah", "{2}{G}", 3, 2,
			WithSubTypes("Cat"),
		)
	})

// Primordial Sage {4}{G}{G}
// Creature — Spirit
// 4/5
// Whenever you cast a creature spell, you may draw a card.
// TODO: implement
	Register("Primordial Sage", func() Card {
		return NewCreature("Primordial Sage", "{4}{G}{G}", 4, 5,
			WithSubTypes("Spirit"),
		)
	})

// Rampaging Brontodon {5}{G}{G}
// Creature — Dinosaur
// 7/7
// Trample
// Whenever this creature attacks, it gets +1/+1 until end of turn for each land you control.
// TODO: implement
	Register("Rampaging Brontodon", func() Card {
		return NewCreature("Rampaging Brontodon", "{5}{G}{G}", 7, 7,
			WithSubTypes("Dinosaur"),
		)
	})

// Ravenous Baloth {2}{G}{G}
// Creature — Beast
// 4/4
// Sacrifice a Beast: You gain 4 life.
// TODO: implement
	Register("Ravenous Baloth", func() Card {
		return NewCreature("Ravenous Baloth", "{2}{G}{G}", 4, 4,
			WithSubTypes("Beast"),
		)
	})

// Rishkar, Peema Renegade {2}{G}
// Legendary Creature — Elf Druid
// 2/2
// When Rishkar enters, put a +1/+1 counter on each of up to two target creatures.
// Each creature you control with a counter on it has "{T}: Add {G}."
// TODO: implement
	Register("Rishkar, Peema Renegade", func() Card {
		return NewCreature("Rishkar, Peema Renegade", "{2}{G}", 2, 2,
			WithSubTypes("Elf", "Druid"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Rumbling Baloth {2}{G}{G}
// Creature — Beast
// 4/4
// TODO: implement
	Register("Rumbling Baloth", func() Card {
		return NewCreature("Rumbling Baloth", "{2}{G}{G}", 4, 4,
			WithSubTypes("Beast"),
		)
	})

// Scrounging Bandar {1}{G}
// Creature — Cat Monkey
// 0/0
// This creature enters with two +1/+1 counters on it.
// At the beginning of your upkeep, you may move any number of +1/+1 counters from this creature onto another target creature.
// TODO: implement
	Register("Scrounging Bandar", func() Card {
		return NewCreature("Scrounging Bandar", "{1}{G}", 0, 0,
			WithSubTypes("Cat", "Monkey"),
		)
	})

// Selvala, Heart of the Wilds {1}{G}{G}
// Legendary Creature — Elf Scout
// 2/3
// Whenever another creature enters, its controller may draw a card if its power is greater than each other creature's power.
// {G}, {T}: Add X mana in any combination of colors, where X is the greatest power among creatures you control.
// TODO: implement
	Register("Selvala, Heart of the Wilds", func() Card {
		return NewCreature("Selvala, Heart of the Wilds", "{1}{G}{G}", 2, 3,
			WithSubTypes("Elf", "Scout"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Silhana Wayfinder {1}{G}
// Creature — Elf Scout
// 2/1
// When this creature enters, look at the top four cards of your library. You may reveal a creature or land card from among them and put it on top of your library. Put the rest on the bottom of your library in a random order.
// TODO: implement
	Register("Silhana Wayfinder", func() Card {
		return NewCreature("Silhana Wayfinder", "{1}{G}", 2, 1,
			WithSubTypes("Elf", "Scout"),
		)
	})

// Somberwald Stag {3}{G}{G}
// Creature — Elk
// 4/3
// When this creature enters, you may have it fight target creature you don't control.
// TODO: implement
	Register("Somberwald Stag", func() Card {
		return NewCreature("Somberwald Stag", "{3}{G}{G}", 4, 3,
			WithSubTypes("Elk"),
		)
	})

// Soul of the Harvest {4}{G}{G}
// Creature — Elemental
// 6/6
// Trample
// Whenever another nontoken creature you control enters, you may draw a card.
// TODO: implement
	Register("Soul of the Harvest", func() Card {
		return NewCreature("Soul of the Harvest", "{4}{G}{G}", 6, 6,
			WithSubTypes("Elemental"),
		)
	})

// Sporemound {3}{G}{G}
// Creature — Fungus
// 3/3
// Landfall — Whenever a land you control enters, create a 1/1 green Saproling creature token.
// TODO: implement
	Register("Sporemound", func() Card {
		return NewCreature("Sporemound", "{3}{G}{G}", 3, 3,
			WithSubTypes("Fungus"),
		)
	})

// Sylvan Brushstrider {2}{G}
// Creature — Beast
// 3/2
// When this creature enters, you gain 2 life.
// TODO: implement
	Register("Sylvan Brushstrider", func() Card {
		return NewCreature("Sylvan Brushstrider", "{2}{G}", 3, 2,
			WithSubTypes("Beast"),
		)
	})

// Sylvan Ranger {1}{G}
// Creature — Elf Scout Ranger
// 1/1
// When this creature enters, you may search your library for a basic land card, reveal it, put it into your hand, then shuffle.
// TODO: implement
	Register("Sylvan Ranger", func() Card {
		return NewCreature("Sylvan Ranger", "{1}{G}", 1, 1,
			WithSubTypes("Elf", "Scout", "Ranger"),
		)
	})

// Thragtusk {4}{G}
// Creature — Beast
// 5/3
// When this creature enters, you gain 5 life.
// When this creature leaves the battlefield, create a 3/3 green Beast creature token.
// TODO: implement
	Register("Thragtusk", func() Card {
		return NewCreature("Thragtusk", "{4}{G}", 5, 3,
			WithSubTypes("Beast"),
		)
	})

// Thundering Spineback {5}{G}{G}
// Creature — Dinosaur
// 5/5
// Other Dinosaurs you control get +1/+1.
// {5}{G}: Create a 3/3 green Dinosaur creature token with trample. (It can deal excess combat damage to the player or planeswalker it's attacking.)
// TODO: implement
	Register("Thundering Spineback", func() Card {
		return NewCreature("Thundering Spineback", "{5}{G}{G}", 5, 5,
			WithSubTypes("Dinosaur"),
		)
	})

// Towering Titan {4}{G}{G}
// Creature — Giant
// 0/0
// This creature enters with X +1/+1 counters on it, where X is the total toughness of other creatures you control.
// Sacrifice a creature with defender: All creatures gain trample until end of turn.
// TODO: implement
	Register("Towering Titan", func() Card {
		return NewCreature("Towering Titan", "{4}{G}{G}", 0, 0,
			WithSubTypes("Giant"),
		)
	})

// Ulvenwald Hydra {4}{G}{G}
// Creature — Hydra
// */*
// Reach
// Ulvenwald Hydra's power and toughness are each equal to the number of lands you control.
// When this creature enters, you may search your library for a land card, put it onto the battlefield tapped, then shuffle.
// TODO: implement
	Register("Ulvenwald Hydra", func() Card {
		return NewCreature("Ulvenwald Hydra", "{4}{G}{G}", 0, 0,
			WithSubTypes("Hydra"),
		)
	})

// Wall of Blossoms {1}{G}
// Creature — Plant Wall
// 0/4
// Defender
// When this creature enters, draw a card.
// TODO: implement
	Register("Wall of Blossoms", func() Card {
		return NewCreature("Wall of Blossoms", "{1}{G}", 0, 4,
			WithSubTypes("Plant", "Wall"),
		)
	})

// Wall of Vines {G}
// Creature — Plant Wall
// 0/3
// Defender (This creature can't attack.)
// Reach (This creature can block creatures with flying.)
// TODO: implement
	Register("Wall of Vines", func() Card {
		return NewCreature("Wall of Vines", "{G}", 0, 3,
			WithSubTypes("Plant", "Wall"),
		)
	})

// Wildheart Invoker {2}{G}{G}
// Creature — Elf Shaman
// 4/3
// {8}: Target creature gets +5/+5 and gains trample until end of turn. (It can deal excess combat damage to the player or planeswalker it's attacking.)
// TODO: implement
	Register("Wildheart Invoker", func() Card {
		return NewCreature("Wildheart Invoker", "{2}{G}{G}", 4, 3,
			WithSubTypes("Elf", "Shaman"),
		)
	})

// Woodborn Behemoth {3}{G}{G}
// Creature — Elemental
// 4/4
// As long as you control eight or more lands, this creature gets +4/+4 and has trample. (It can deal excess combat damage to the player or planeswalker it's attacking.)
// TODO: implement
	Register("Woodborn Behemoth", func() Card {
		return NewCreature("Woodborn Behemoth", "{3}{G}{G}", 4, 4,
			WithSubTypes("Elemental"),
		)
	})

// Wren's Run Vanquisher {1}{G}
// Creature — Elf Warrior
// 3/3
// As an additional cost to cast this spell, reveal an Elf card from your hand or pay {3}.
// Deathtouch (Any amount of damage this deals to a creature is enough to destroy it.)
// TODO: implement
	Register("Wren's Run Vanquisher", func() Card {
		return NewCreature("Wren's Run Vanquisher", "{1}{G}", 3, 3,
			WithSubTypes("Elf", "Warrior"),
		)
	})


	// ===== MULTICOLOR CREATURES =====

// Dinrova Horror {4}{U}{B}
// Creature — Horror
// 4/4
// When this creature enters, return target permanent to its owner's hand, then that player discards a card.
// TODO: implement
	Register("Dinrova Horror", func() Card {
		return NewCreature("Dinrova Horror", "{4}{U}{B}", 4, 4,
			WithSubTypes("Horror"),
		)
	})

// Fusion Elemental {W}{U}{B}{R}{G}
// Creature — Elemental
// 8/8
// TODO: implement
	Register("Fusion Elemental", func() Card {
		return NewCreature("Fusion Elemental", "{W}{U}{B}{R}{G}", 8, 8,
			WithSubTypes("Elemental"),
		)
	})

// Ironroot Warlord {1}{G}{W}
// Creature — Treefolk Soldier
// */5
// Ironroot Warlord's power is equal to the number of creatures you control.
// {3}{G}{W}: Create a 1/1 white Soldier creature token.
// TODO: implement
	Register("Ironroot Warlord", func() Card {
		return NewCreature("Ironroot Warlord", "{1}{G}{W}", 0, 5,
			WithSubTypes("Treefolk", "Soldier"),
		)
	})

// Maelstrom Archangel {W}{U}{B}{R}{G}
// Creature — Angel
// 5/5
// Flying
// Whenever this creature deals combat damage to a player, you may cast a spell from your hand without paying its mana cost.
// TODO: implement
	Register("Maelstrom Archangel", func() Card {
		return NewCreature("Maelstrom Archangel", "{W}{U}{B}{R}{G}", 5, 5,
			WithSubTypes("Angel"),
		)
	})

// Raging Regisaur {2}{R}{G}
// Creature — Dinosaur
// 4/4
// Whenever this creature attacks, it deals 1 damage to any target.
// TODO: implement
	Register("Raging Regisaur", func() Card {
		return NewCreature("Raging Regisaur", "{2}{R}{G}", 4, 4,
			WithSubTypes("Dinosaur"),
		)
	})


	// ===== COLORLESS CREATURES =====

// Alloy Myr {3}
// Artifact Creature — Myr
// 2/2
// {T}: Add one mana of any color.
// TODO: implement
	Register("Alloy Myr", func() Card {
		return NewCreature("Alloy Myr", "{3}", 2, 2,
			WithSubTypes("Myr"),
			WithCardType(TypeArtifact),
		)
	})

// Ancestral Statue {4}
// Artifact Creature — Golem
// 3/4
// When this creature enters, return a nonland permanent you control to its owner's hand.
// TODO: implement
	Register("Ancestral Statue", func() Card {
		return NewCreature("Ancestral Statue", "{4}", 3, 4,
			WithSubTypes("Golem"),
			WithCardType(TypeArtifact),
		)
	})

// Chamber Sentry {X}
// Artifact Creature — Construct
// 0/0
// This creature enters with a +1/+1 counter on it for each color of mana spent to cast it.
// {X}, {T}, Remove X +1/+1 counters from this creature: It deals X damage to any target.
// {W}{U}{B}{R}{G}: Return this card from your graveyard to your hand.
// TODO: implement
	Register("Chamber Sentry", func() Card {
		return NewCreature("Chamber Sentry", "{X}", 0, 0,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
		)
	})

// Dragonloft Idol {4}
// Artifact Creature — Gargoyle
// 3/3
// As long as you control a Dragon, this creature gets +1/+1 and has flying and trample.
// TODO: implement
	Register("Dragonloft Idol", func() Card {
		return NewCreature("Dragonloft Idol", "{4}", 3, 3,
			WithSubTypes("Gargoyle"),
			WithCardType(TypeArtifact),
		)
	})

// Gargoyle Sentinel {3}
// Artifact Creature — Gargoyle
// 3/3
// Defender (This creature can't attack.)
// {3}: Until end of turn, this creature loses defender and gains flying.
// TODO: implement
	Register("Gargoyle Sentinel", func() Card {
		return NewCreature("Gargoyle Sentinel", "{3}", 3, 3,
			WithSubTypes("Gargoyle"),
			WithCardType(TypeArtifact),
		)
	})

// Gingerbrute {1}
// Artifact Creature — Food Golem
// 1/1
// Haste (This creature can attack and {T} as soon as it comes under your control.)
// {1}: This creature can't be blocked this turn except by creatures with haste.
// {2}, {T}, Sacrifice this creature: You gain 3 life.
// TODO: implement
	Register("Gingerbrute", func() Card {
		return NewCreature("Gingerbrute", "{1}", 1, 1,
			WithSubTypes("Food", "Golem"),
			WithCardType(TypeArtifact),
		)
	})

// Jousting Dummy {2}
// Artifact Creature — Scarecrow Knight
// 2/1
// {3}: This creature gets +1/+0 until end of turn.
// TODO: implement
	Register("Jousting Dummy", func() Card {
		return NewCreature("Jousting Dummy", "{2}", 2, 1,
			WithSubTypes("Scarecrow", "Knight"),
			WithCardType(TypeArtifact),
		)
	})

// Lightning-Core Excavator {1}
// Artifact Creature — Golem
// 0/3
// {5}, {T}, Sacrifice this creature: It deals 3 damage to any target.
// TODO: implement
	Register("Lightning-Core Excavator", func() Card {
		return NewCreature("Lightning-Core Excavator", "{1}", 0, 3,
			WithSubTypes("Golem"),
			WithCardType(TypeArtifact),
		)
	})

// Meteor Golem {7}
// Artifact Creature — Golem
// 3/3
// When this creature enters, destroy target nonland permanent an opponent controls.
// TODO: implement
	Register("Meteor Golem", func() Card {
		return NewCreature("Meteor Golem", "{7}", 3, 3,
			WithSubTypes("Golem"),
			WithCardType(TypeArtifact),
		)
	})

// Myr Sire {2}
// Artifact Creature — Phyrexian Myr
// 1/1
// When this creature dies, create a 1/1 colorless Phyrexian Myr artifact creature token.
// TODO: implement
	Register("Myr Sire", func() Card {
		return NewCreature("Myr Sire", "{2}", 1, 1,
			WithSubTypes("Phyrexian", "Myr"),
			WithCardType(TypeArtifact),
		)
	})

// Perilous Myr {2}
// Artifact Creature — Phyrexian Myr
// 1/1
// When this creature dies, it deals 2 damage to any target.
// TODO: implement
	Register("Perilous Myr", func() Card {
		return NewCreature("Perilous Myr", "{2}", 1, 1,
			WithSubTypes("Phyrexian", "Myr"),
			WithCardType(TypeArtifact),
		)
	})

// Roving Keep {7}
// Artifact Creature — Wall
// 5/7
// Defender
// {7}: This creature gets +2/+0 and gains trample until end of turn. It can attack this turn as though it didn't have defender.
// TODO: implement
	Register("Roving Keep", func() Card {
		return NewCreature("Roving Keep", "{7}", 5, 7,
			WithSubTypes("Wall"),
			WithCardType(TypeArtifact),
		)
	})

// Runed Servitor {2}
// Artifact Creature — Construct
// 2/2
// When this creature dies, each player draws a card.
// TODO: implement
	Register("Runed Servitor", func() Card {
		return NewCreature("Runed Servitor", "{2}", 2, 2,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
		)
	})

// Scarecrone {3}
// Artifact Creature — Scarecrow
// 1/2
// {1}, Sacrifice a Scarecrow: Draw a card.
// {4}, {T}: Return target artifact creature card from your graveyard to the battlefield.
// TODO: implement
	Register("Scarecrone", func() Card {
		return NewCreature("Scarecrone", "{3}", 1, 2,
			WithSubTypes("Scarecrow"),
			WithCardType(TypeArtifact),
		)
	})

// Scuttlemutt {3}
// Artifact Creature — Scarecrow
// 2/2
// {T}: Add one mana of any color.
// {T}: Target creature becomes the color or colors of your choice until end of turn.
// TODO: implement
	Register("Scuttlemutt", func() Card {
		return NewCreature("Scuttlemutt", "{3}", 2, 2,
			WithSubTypes("Scarecrow"),
			WithCardType(TypeArtifact),
		)
	})

// Signpost Scarecrow {4}
// Artifact Creature — Scarecrow
// 2/4
// Vigilance
// {2}: Add one mana of any color.
// TODO: implement
	Register("Signpost Scarecrow", func() Card {
		return NewCreature("Signpost Scarecrow", "{4}", 2, 4,
			WithSubTypes("Scarecrow"),
			WithCardType(TypeArtifact),
		)
	})

// Skittering Surveyor {3}
// Artifact Creature — Construct
// 1/2
// When this creature enters, you may search your library for a basic land card, reveal it, put it into your hand, then shuffle.
// TODO: implement
	Register("Skittering Surveyor", func() Card {
		return NewCreature("Skittering Surveyor", "{3}", 1, 2,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
		)
	})

// Suspicious Bookcase {2}
// Artifact Creature — Wall
// 0/4
// Defender
// {3}, {T}: Target creature can't be blocked this turn.
// TODO: implement
	Register("Suspicious Bookcase", func() Card {
		return NewCreature("Suspicious Bookcase", "{2}", 0, 4,
			WithSubTypes("Wall"),
			WithCardType(TypeArtifact),
		)
	})

}
