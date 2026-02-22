package worlddata

// allNPCs is the complete NPC roster for Vax.
// Edit this file freely — change decks, loot, dialog, respawn times.
// IDs must be unique and must match the NPCIDs listed in rooms.go.
var allNPCs = []NPCDef{

	// -------------------------------------------------------------------------
	// Elder Maren — Tangle's Edge. Green mage who knows every mana current on Vax.
	// No ante. Gentle intro fight. Tells you everything about the Baron.
	// -------------------------------------------------------------------------
	{
		ID:          "elder_maren",
		Name:        "Elder Maren",
		Glyph:       'M',
		Description: "A broad-shouldered woman with bark-stained hands and a deck worn soft from handling. She has been tending Tangle's Edge since before the black mana started spreading.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Plains", 8},
			{"Forest", 8},
			{"Savannah Lions", 4},
			{"Llanowar Elves", 4},
			{"Grizzly Bears", 4},
			{"White Knight", 3},
			{"Mesa Pegasus", 3},
			{"Giant Spider", 2},
			{"Serra Angel", 2},
			{"Giant Growth", 3},
			{"Healing Salve", 2},
			{"Swords to Plowshares", 2},
		},
		LootFixed:      []string{"Llanowar Elves"},
		LootTable:      []LootEntry{{"Giant Growth", 3}, {"Mesa Pegasus", 2}, {"Healing Salve", 2}, {"Swords to Plowshares", 1}},
		LootTableCount: 1,
		GoldReward:     15,
		Dialog: []string{
			"Vax has five mana currents — had five. The black one has been swallowing the others since Sengir arrived.",
			"A deck is a conversation with the mana. Make sure you understand both sides of it.",
			"Sengir isn't here to conquer. He wants to drain the plane entirely. Vampires feed on blood. He feeds on mana.",
			"The Dueling Road's safe enough. Past the Thornwood, be careful what you summon. The mana's sour there and summons don't always dissolve cleanly.",
		},
		RespawnMinutes: 20,
	},

	// -------------------------------------------------------------------------
	// Merchant Varro — The Bazaar. Deals in cards and knows their worth.
	// No ante. Red/artifact deck. Sells information with every transaction.
	// -------------------------------------------------------------------------
	{
		ID:          "merchant_varro",
		Name:        "Merchant Varro",
		Glyph:       'V',
		Description: "A compact man with appraiser's eyes who keeps his own deck behind the counter and isn't shy about demonstrating it. He knows the secondary market value of every card on Vax.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Mountain", 8},
			{"Plains", 5},
			{"Lightning Bolt", 4},
			{"Hurloon Minotaur", 4},
			{"Hill Giant", 3},
			{"Ironclaw Orcs", 3},
			{"Mons's Goblin Raiders", 4},
			{"Disenchant", 3},
			{"Shatter", 3},
			{"Gray Ogre", 2},
			{"Sol Ring", 1},
			{"Icy Manipulator", 1},
		},
		LootFixed:      []string{"Disenchant"},
		LootTable:      []LootEntry{{"Lightning Bolt", 3}, {"Shatter", 2}, {"Icy Manipulator", 1}, {"Sol Ring", 1}},
		LootTableCount: 1,
		GoldReward:     20,
		Dialog: []string{
			"I'll take any card in good condition. Paying premium for anything with flying right now.",
			"Sengir has been buying out the black market. Literally. Good black cards move south fast — bring them here before they disappear.",
			"Lost a Shivan Dragon in an ante match three years ago. Think about it every time I see a Mountain.",
			"The Warden's vault is worth the coin if you care about your deck. Most people don't know what's in there.",
		},
		RespawnMinutes: 30,
	},

	// -------------------------------------------------------------------------
	// The Wandering Duelist — The Dueling Road. Planes-touched veteran.
	// Ante FORCED. Blue/black control. She's seen this situation before.
	// -------------------------------------------------------------------------
	{
		ID:          "wandering_duelist",
		Name:        "The Wandering Duelist",
		Glyph:       'D',
		Description: "A lean woman in a travel-stained grey cloak, every movement unhurried. She has been at crossroads like this one on a dozen planes. She is never the one who blinks first.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Island", 8},
			{"Swamp", 8},
			{"Dark Ritual", 3},
			{"Counterspell", 4},
			{"Terror", 4},
			{"Hypnotic Specter", 3},
			{"Sengir Vampire", 2},
			{"Royal Assassin", 2},
			{"Unsummon", 3},
			{"Mind Twist", 2},
			{"Merfolk of the Pearl Trident", 3},
		},
		LootFixed:      []string{"Counterspell"},
		LootTable:      []LootEntry{{"Terror", 3}, {"Hypnotic Specter", 2}, {"Royal Assassin", 1}, {"Mind Twist", 1}},
		LootTableCount: 1,
		GoldReward:     35,
		Dialog: []string{
			"I've dueled on eleven planes. Vax is the only one where summons linger after the match. Don't let that surprise you mid-game.",
			"Ante isn't optional. Not with me. That's not a policy, it's a principle.",
			"Sengir corrupted a mage I knew on Ulgrotha. Took her whole deck when she fell. Don't let him take yours.",
			"Everyone who's gone south thinking they had a good enough deck — some of them did. Didn't matter.",
		},
		RespawnMinutes: 60,
	},

	// -------------------------------------------------------------------------
	// Scavenger Theron — The Dueling Road. Small-time card hustler.
	// No ante. Red/black aggro. Knows things he probably shouldn't.
	// -------------------------------------------------------------------------
	{
		ID:          "scavenger_theron",
		Name:        "Scavenger Theron",
		Glyph:       'T',
		Description: "A wiry young man with too many card-pouches on his belt and a habit of cracking his knuckles in sequence. He smells like someone who has been sleeping near a mana vent.",
		Personality: "aggro",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Mountain", 8},
			{"Swamp", 8},
			{"Lightning Bolt", 4},
			{"Mons's Goblin Raiders", 4},
			{"Ironclaw Orcs", 3},
			{"Gray Ogre", 3},
			{"Terror", 3},
			{"Black Knight", 2},
			{"Scathe Zombies", 2},
			{"Fear", 2},
			{"Drudge Skeletons", 2},
		},
		LootFixed:      nil,
		LootTable:      []LootEntry{{"Lightning Bolt", 3}, {"Terror", 2}, {"Black Knight", 2}, {"Ironclaw Orcs", 1}},
		LootTableCount: 1,
		GoldReward:     12,
		Dialog: []string{
			"Good cards come to me eventually. People lose them, they end up here.",
			"You hear that low hum when a mana zone opens? That's right before a summon hits real ground. Try not to be underneath it.",
			"The Duelist took three cards off me in ante last month. Still not over it. Don't play her unless you're ready.",
			"I don't go past the Thornwood. Personal policy. Started as a choice, now it's just a fact about me.",
		},
		RespawnMinutes: 25,
	},

	// -------------------------------------------------------------------------
	// The Warden — The Warden's Vault. Keeper of Vax's card archive.
	// No ante. Green midrange. Knows what Sengir is really after.
	// -------------------------------------------------------------------------
	{
		ID:          "the_warden",
		Name:        "The Warden",
		Glyph:       'W',
		Description: "An old mage whose age is hard to place, sitting at the center table with a spread deck and the stillness of someone who has been thinking for a long time. The card-sleeves on the walls were their life's work.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Forest", 16},
			{"Llanowar Elves", 4},
			{"Grizzly Bears", 4},
			{"Elvish Archers", 3},
			{"Giant Spider", 3},
			{"War Mammoth", 3},
			{"Craw Wurm", 2},
			{"Force of Nature", 2},
			{"Giant Growth", 2},
			{"Berserk", 2},
			{"Regrowth", 1},
		},
		LootFixed:      []string{"Regrowth"},
		LootTable:      []LootEntry{{"Giant Spider", 3}, {"War Mammoth", 2}, {"Force of Nature", 1}, {"Berserk", 1}, {"Llanowar Elves", 2}},
		LootTableCount: 1,
		GoldReward:     50,
		Dialog: []string{
			"These shelves held every card ever played on Vax. Most of them gone south with their owners.",
			"Sengir didn't come here for the land. He came for the archive. Cards encode mana patterns. Enough of them and you can drain a plane systematically.",
			"Forest mana resists the corruption. The green cards in your deck are anchors. Don't trade them away lightly.",
			"I'm not leaving until I finish cataloguing what's left. Someone has to know what was here.",
		},
		RespawnMinutes: 45,
	},

	// -------------------------------------------------------------------------
	// Sister Vael — The Ashen Chapel. A mage undone by the black mana.
	// Ante FORCED. White/black control. She's not entirely herself anymore.
	// Defeat her to break the ward on the Dread Keep.
	// -------------------------------------------------------------------------
	{
		ID:          "sister_vael",
		Name:        "Sister Vael",
		Glyph:       'S',
		Description: "A mage in grey robes, kneeling at a reversed altar with the stillness of someone who has made their peace with something they shouldn't have. Her deck sits beside her, face down.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Plains", 7},
			{"Swamp", 9},
			{"White Knight", 3},
			{"Black Knight", 3},
			{"Swords to Plowshares", 3},
			{"Terror", 3},
			{"Hypnotic Specter", 3},
			{"Sengir Vampire", 2},
			{"Serra Angel", 1},
			{"Dark Ritual", 3},
			{"Wrath of God", 1},
			{"Unholy Strength", 2},
			{"Royal Assassin", 1},
		},
		LootFixed:      []string{"Swords to Plowshares"},
		LootTable:      []LootEntry{{"Wrath of God", 1}, {"Sengir Vampire", 2}, {"Hypnotic Specter", 2}, {"Terror", 3}},
		LootTableCount: 1,
		GoldReward:     45,
		Dialog: []string{
			"The Baron's mana is cleaner than I expected. More precise. That's the part that got me.",
			"I used to duel for the craft of it. Now I duel for something else. I haven't named what.",
			"Everything south of here feeds him. Including me, a little. I've stopped trying to fix that.",
			"You'll need to beat me to go further. I know that. I'm not going to make it easy.",
		},
		RespawnMinutes: 50,
	},

	// -------------------------------------------------------------------------
	// Baron Sengir — The Dread Keep. The vampire lord of Ulgrotha, feeding on Vax.
	// Ante FORCED. Mono-black vampires. The final encounter. Never respawns.
	// -------------------------------------------------------------------------
	{
		ID:          "baron_sengir",
		Name:        "Baron Sengir",
		Glyph:       'B',
		Description: "Tall, pale, unhurried. He has the manner of someone who has never once needed to rush. His deck is already on the table. He's been shuffling it for days, waiting.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Swamp", 18},
			{"Dark Ritual", 4},
			{"Terror", 4},
			{"Hypnotic Specter", 4},
			{"Sengir Vampire", 4},
			{"Royal Assassin", 2},
			{"Nightmare", 2},
			{"Animate Dead", 3},
			{"Drudge Skeletons", 2},
			{"Demonic Tutor", 2},
			{"Black Knight", 3},
			{"Mind Twist", 2},
		},
		LootFixed:      []string{"Demonic Tutor", "Nightmare"},
		LootTable:      []LootEntry{{"Sengir Vampire", 3}, {"Animate Dead", 2}, {"Mind Twist", 2}, {"Royal Assassin", 1}},
		LootTableCount: 2,
		GoldReward:     200,
		Dialog: []string{
			"You've come a long way to add your cards to my collection.",
			"Every mage who's crossed that threshold has enriched me. Even the ones who refused to duel.",
			"Vax has excellent mana. Rich, layered, old. I intend to take all of it.",
			"Put your deck on the table. Let's see what you're made of.",
		},
		RespawnMinutes: 0, // never respawns — the campaign's final encounter
	},

	// =========================================================================
	// WESTERN GREEN BRANCH NPCs
	// =========================================================================

	// -------------------------------------------------------------------------
	// Asha — Greenwood Rise. Green/white midrange. Friendly intro mentor.
	// No ante. The first person you meet heading west.
	// -------------------------------------------------------------------------
	{
		ID:          "asha",
		Name:        "Asha",
		Glyph:       'A',
		Description: "A mage in white-trimmed green leathers who has been holding Greenwood Rise since before the corruption started spreading. She trains new arrivals, tells them about the mana currents, and watches the road east.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Plains", 7},
			{"Forest", 9},
			{"Llanowar Elves", 4},
			{"Savannah Lions", 4},
			{"Mesa Pegasus", 3},
			{"Grizzly Bears", 4},
			{"White Knight", 3},
			{"Serra Angel", 2},
			{"Giant Spider", 2},
			{"Giant Growth", 3},
			{"Swords to Plowshares", 2},
		},
		LootFixed:      []string{"Giant Growth"},
		LootTable:      []LootEntry{{"Swords to Plowshares", 3}, {"Mesa Pegasus", 2}, {"White Knight", 2}, {"Serra Angel", 1}},
		LootTableCount: 1,
		GoldReward:     18,
		Dialog: []string{
			"Greenwood Rise marks the edge of Vax's oldest forest. The mana here remembers what the plane was before Sengir came.",
			"I've been here since before the corruption reached the Thornwood. There used to be three times as many of us dueling under these canopies.",
			"White mana doesn't flow easily this far from civilization. I anchor it with the green. Works better than you'd expect.",
			"If you're heading toward the Thornwood, talk to Elder Maren first. She knows the mana currents better than anyone.",
		},
		RespawnMinutes: 20,
	},

	// -------------------------------------------------------------------------
	// Orvyn — The Mosshallow. Green/blue herbalist. Studies the confluence.
	// No ante. Knows about the corruption spreading west.
	// -------------------------------------------------------------------------
	{
		ID:          "orvyn",
		Name:        "Orvyn",
		Glyph:       'O',
		Description: "A researcher who has been cataloguing the mosshallow confluence for years. He doesn't duel for sport — he duels to test hypotheses. His deck is an experiment in controlled card advantage.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Forest", 8},
			{"Island", 8},
			{"Llanowar Elves", 4},
			{"Merfolk of the Pearl Trident", 4},
			{"Giant Spider", 3},
			{"Prodigal Sorcerer", 3},
			{"Water Elemental", 2},
			{"Counterspell", 3},
			{"Unsummon", 3},
			{"Giant Growth", 3},
		},
		LootFixed:      []string{"Counterspell"},
		LootTable:      []LootEntry{{"Giant Spider", 3}, {"Merfolk of the Pearl Trident", 2}, {"Water Elemental", 1}, {"Prodigal Sorcerer", 2}},
		LootTableCount: 1,
		GoldReward:     22,
		Dialog: []string{
			"I study how forest mana and water mana interact. The mosshallow is a confluence point — rarer than you'd think.",
			"Counterspell is the politest way to disagree with someone in a duel. I recommend mastering it early.",
			"Sengir's black mana is drying the moss beds. Another month and this confluence won't be worth studying.",
			"The Warden sent me here to document what's left. I suspect I'm also meant to slow the corruption. Bit outside my specialty.",
		},
		RespawnMinutes: 25,
	},

	// -------------------------------------------------------------------------
	// Cael — The Rootweave. Pure green druid. Doesn't travel.
	// No ante. Serious midrange deck with big threats.
	// -------------------------------------------------------------------------
	{
		ID:          "cael",
		Name:        "Cael",
		Glyph:       'C',
		Description: "A druid who has lived in the rootweave for thirty years and has no intention of leaving. They communicate with the trees. The trees communicate back. The Force of Nature in their deck is not metaphorical.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Forest", 14},
			{"Llanowar Elves", 4},
			{"Giant Spider", 3},
			{"War Mammoth", 3},
			{"Elvish Archers", 3},
			{"Craw Wurm", 3},
			{"Force of Nature", 1},
			{"Giant Growth", 4},
			{"Regeneration", 2},
			{"Berserk", 2},
			{"Regrowth", 1},
		},
		LootFixed:      []string{"Regrowth"},
		LootTable:      []LootEntry{{"Force of Nature", 1}, {"Berserk", 2}, {"Craw Wurm", 2}, {"Giant Spider", 3}},
		LootTableCount: 1,
		GoldReward:     35,
		Dialog: []string{
			"The rootweave is what it sounds like — every tree here shares roots. The forest communicates, if you know how to listen.",
			"Force of Nature is not a card I use lightly. The mana cost is a debt the land remembers.",
			"I don't travel. The rootweave is my post and my purpose. The world can come to me.",
			"Regrowth isn't about bringing things back. It's about acknowledging that the land contains what you've lost.",
		},
		RespawnMinutes: 35,
	},

	// -------------------------------------------------------------------------
	// Greth — Verdant Hollow. Green/red beast tamer. Guards the path south.
	// No ante. Aggressive midrange with Kird Apes.
	// -------------------------------------------------------------------------
	{
		ID:          "greth",
		Name:        "Greth",
		Glyph:       'G',
		Description: "A beast tamer with Kird Apes that he's raised from cubs and a red mana temper he hasn't managed to train out of himself. He runs the hollow like his personal territory, which it effectively is.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Forest", 8},
			{"Mountain", 8},
			{"Llanowar Elves", 4},
			{"Kird Ape", 4},
			{"Craw Wurm", 3},
			{"Lightning Bolt", 4},
			{"Fireball", 2},
			{"Giant Growth", 3},
			{"Mons's Goblin Raiders", 3},
			{"Giant Spider", 2},
		},
		LootFixed:      []string{"Kird Ape"},
		LootTable:      []LootEntry{{"Lightning Bolt", 3}, {"Fireball", 2}, {"Kird Ape", 2}, {"Giant Spider", 1}},
		LootTableCount: 1,
		GoldReward:     40,
		Dialog: []string{
			"My creatures know the difference between a duel and a hunt. Usually. The Kird Apes less so.",
			"The hollow is deeper than it looks. I've found things here that didn't come from any known mana current.",
			"Sengir's corruption reaches even here. I've lost three summoned creatures to the black seep. They didn't dissolve cleanly.",
			"You want to go further south? Beat me first. I don't let anyone pass who isn't ready.",
		},
		RespawnMinutes: 40,
	},

	// =========================================================================
	// CENTRAL SOUTH BRANCH NPCs
	// =========================================================================

	// -------------------------------------------------------------------------
	// Pardoned Knight — The Pilgrim's Road. White aggro. Reformed duelist.
	// No ante. Holds the southern road crossroads.
	// -------------------------------------------------------------------------
	{
		ID:          "pardoned_knight",
		Name:        "The Pardoned Knight",
		Glyph:       'K',
		Description: "A knight who rode south years ago, lost everything to the corruption, and rebuilt themselves from scratch. The pardon in the name is self-issued. They've earned it, by their own accounting.",
		Personality: "aggro",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Plains", 16},
			{"Savannah Lions", 4},
			{"White Knight", 4},
			{"Benalish Hero", 4},
			{"Mesa Pegasus", 3},
			{"Serra Angel", 2},
			{"Black Knight", 2},
			{"Swords to Plowshares", 4},
			{"Wrath of God", 1},
		},
		LootFixed:      []string{"White Knight"},
		LootTable:      []LootEntry{{"Serra Angel", 2}, {"Swords to Plowshares", 3}, {"Wrath of God", 1}, {"Savannah Lions", 2}},
		LootTableCount: 1,
		GoldReward:     30,
		Dialog: []string{
			"I rode south years ago thinking a white deck was enough. I was wrong. I've been building back ever since.",
			"The Pilgrim's Road is the only straight path south left. Everything else bends toward the black mana.",
			"Wrath of God isn't a card I wanted in my deck. It's a card the situation demanded.",
			"If you're going to the chapel, know that Sister Vael was a good mage once. Keep that in mind when you face her.",
		},
		RespawnMinutes: 30,
	},

	// -------------------------------------------------------------------------
	// Nixx — Charnel Fork. Red/black aggro scavenger. Knows the roads south.
	// No ante. Knows things she probably shouldn't.
	// -------------------------------------------------------------------------
	{
		ID:          "nixx",
		Name:        "Nixx",
		Glyph:       'N',
		Description: "A professional scavenger who works the junction where the black mana bleeds into the dying forest. She finds cards in places no one should be looking and knows the price of everything.",
		Personality: "aggro",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Mountain", 8},
			{"Swamp", 8},
			{"Lightning Bolt", 4},
			{"Mons's Goblin Raiders", 4},
			{"Terror", 4},
			{"Ironclaw Orcs", 3},
			{"Black Knight", 3},
			{"Scathe Zombies", 2},
			{"Fire Elemental", 2},
			{"Drudge Skeletons", 2},
		},
		LootFixed:      nil,
		LootTable:      []LootEntry{{"Lightning Bolt", 3}, {"Terror", 3}, {"Black Knight", 2}, {"Fire Elemental", 1}},
		LootTableCount: 1,
		GoldReward:     28,
		Dialog: []string{
			"Charnel Fork is where three kinds of bad luck meet. You're standing in all of them.",
			"I find things. Cards, mostly. People leave them when they run south and don't come back.",
			"The black mana gets stronger the deeper you go. I stay at the fork where it's just strong enough to be useful.",
			"Don't go into the Hollow without backup. Something down there likes the dark mana a little too much.",
		},
		RespawnMinutes: 28,
	},

	// -------------------------------------------------------------------------
	// Vex — The Hollow Deep. Black control. Has made arrangements with the dark.
	// Ante FORCED. Guards the passage east to Sengir's Road.
	// -------------------------------------------------------------------------
	{
		ID:          "vex",
		Name:        "Vex",
		Glyph:       'X',
		Description: "Whatever Vex was before arriving in the Hollow Deep, they've adjusted to the conditions. The black mana pools here have become home. The cards in their hand have become vocabulary.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Swamp", 16},
			{"Dark Ritual", 4},
			{"Hypnotic Specter", 4},
			{"Sengir Vampire", 3},
			{"Terror", 4},
			{"Animate Dead", 3},
			{"Drudge Skeletons", 3},
			{"Unholy Strength", 2},
			{"Royal Assassin", 1},
		},
		LootFixed:      []string{"Hypnotic Specter"},
		LootTable:      []LootEntry{{"Animate Dead", 3}, {"Terror", 2}, {"Sengir Vampire", 2}, {"Royal Assassin", 1}},
		LootTableCount: 1,
		GoldReward:     45,
		Dialog: []string{
			"The Hollow Deep is where the light stops and the black mana pools. I've made my arrangements with it.",
			"Animate Dead isn't resurrection. The things that walk out of it are wearing what they used to be.",
			"Sengir doesn't know I'm here. Or doesn't care. Either answer tells you something useful.",
			"You want passage east? Beat me. If you can beat me down here, you're ready for what's past the keep.",
		},
		RespawnMinutes: 45,
	},

	// =========================================================================
	// EASTERN COASTAL BRANCH NPCs
	// =========================================================================

	// -------------------------------------------------------------------------
	// Lyss — The Tidal Watch. Blue midrange. Coastal sentinel.
	// No ante. Watches for Sengir's influence reaching the coast.
	// -------------------------------------------------------------------------
	{
		ID:          "lyss",
		Name:        "Lyss",
		Glyph:       'L',
		Description: "The last sentinel at the Tidal Watch, keeping a post that was built for four. She hasn't left in three years. She says she'll leave when she can confirm the coast is clear. She can't confirm that.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Island", 16},
			{"Merfolk of the Pearl Trident", 4},
			{"Flying Men", 4},
			{"Phantasmal Forces", 3},
			{"Water Elemental", 3},
			{"Counterspell", 4},
			{"Unsummon", 4},
			{"Control Magic", 2},
			{"Prodigal Sorcerer", 2},
		},
		LootFixed:      []string{"Control Magic"},
		LootTable:      []LootEntry{{"Counterspell", 3}, {"Water Elemental", 2}, {"Phantasmal Forces", 2}, {"Merfolk of the Pearl Trident", 1}},
		LootTableCount: 1,
		GoldReward:     20,
		Dialog: []string{
			"The tidal watch post is old. We used to have four mages stationed here. Now it's just me and the birds.",
			"Sea mana is slower than river mana but it accumulates. The coastal duels are longer than you'd expect.",
			"I'm watching for Sengir's influence spreading to the coast. Haven't seen it yet. That just means I'm not looking far enough.",
			"Harbormouth is safe enough. Rhen keeps good order. Tell him Lyss sent you.",
		},
		RespawnMinutes: 20,
	},

	// -------------------------------------------------------------------------
	// Rhen — Harbormouth. Blue/red harbor master. Runs a competitive deck.
	// No ante. Knows the coastal trade routes and the deep water rumors.
	// -------------------------------------------------------------------------
	{
		ID:          "rhen",
		Name:        "Rhen",
		Glyph:       'R',
		Description: "The harbor master of Harbormouth, who runs fire-and-water out of the same deck because that's what the port requires. He's been offered a better posting three times. He turned it down each time.",
		Personality: "tempo",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Island", 8},
			{"Mountain", 8},
			{"Lightning Bolt", 4},
			{"Merfolk of the Pearl Trident", 4},
			{"Flying Men", 3},
			{"Prodigal Sorcerer", 3},
			{"Unsummon", 3},
			{"Counterspell", 2},
			{"Fireball", 2},
			{"Power Sink", 2},
			{"Hill Giant", 2},
		},
		LootFixed:      []string{"Prodigal Sorcerer"},
		LootTable:      []LootEntry{{"Lightning Bolt", 3}, {"Fireball", 2}, {"Counterspell", 2}, {"Merfolk of the Pearl Trident", 1}},
		LootTableCount: 1,
		GoldReward:     25,
		Dialog: []string{
			"Harbormouth handles trade from three coasts. What it can't handle is the rumor that Sengir is targeting the shipping lanes.",
			"I run fire and water out of the same deck. People say it's unstable. I say that's the point.",
			"The tide gate south of here has been acting strange. The water mana is resisting something. That's new.",
			"If you're after sea cards, the Tower is the place. Fen has things that haven't been seen since the old expeditions.",
		},
		RespawnMinutes: 25,
	},

	// -------------------------------------------------------------------------
	// Mora — The Tide Gate. Blue control. Gatekeeper. Ante FORCED.
	// Serious control deck. Rewards precision play.
	// -------------------------------------------------------------------------
	{
		ID:          "mora",
		Name:        "Mora",
		Glyph:       'E',
		Description: "The mage who maintains the tide gate and controls access to the deep coast. Mora does not run an aggressive deck. Mora runs the correct deck for the situation, which turns out to be control.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Island", 16},
			{"Counterspell", 4},
			{"Unsummon", 4},
			{"Merfolk of the Pearl Trident", 4},
			{"Control Magic", 3},
			{"Mahamoti Djinn", 2},
			{"Prodigal Sorcerer", 3},
			{"Power Sink", 3},
			{"Braingeyser", 1},
		},
		LootFixed:      []string{"Control Magic"},
		LootTable:      []LootEntry{{"Mahamoti Djinn", 2}, {"Counterspell", 3}, {"Braingeyser", 1}, {"Power Sink", 2}},
		LootTableCount: 1,
		GoldReward:     55,
		Dialog: []string{
			"The tide gate regulates the flow of sea mana along the coast. I regulate who passes through it.",
			"Ante isn't cruelty. It's precision. You should only duel with stakes if you can afford them.",
			"Braingeyser at the right moment wins games that nothing else touches. Timing is everything.",
			"Past here is the deep coast. The water mana gets strange — layered with something older. Pay attention or you'll lose permanents you didn't mean to sacrifice.",
		},
		RespawnMinutes: 50,
	},

	// -------------------------------------------------------------------------
	// Pelth — Shipwreck Shore. Blue/green salvager. Works the wrecks.
	// No ante. Knows what sank and why.
	// -------------------------------------------------------------------------
	{
		ID:          "pelth",
		Name:        "Pelth",
		Glyph:       'P',
		Description: "A salvager who has been working the shipwreck shore since the ships went down. The wreckage gives up something new every few weeks. Pelth has a deck full of things the sea returned.",
		Personality: "midrange",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Island", 8},
			{"Forest", 8},
			{"Merfolk of the Pearl Trident", 4},
			{"Llanowar Elves", 4},
			{"Giant Spider", 3},
			{"Water Elemental", 3},
			{"Counterspell", 3},
			{"Giant Growth", 3},
			{"Regrowth", 2},
			{"Craw Wurm", 2},
		},
		LootFixed:      []string{"Regrowth"},
		LootTable:      []LootEntry{{"Water Elemental", 2}, {"Counterspell", 2}, {"Giant Spider", 3}, {"Craw Wurm", 1}},
		LootTableCount: 1,
		GoldReward:     38,
		Dialog: []string{
			"The ships here didn't sink from storms. The sea mana destabilized during a major duel offshore. The water elementals didn't dissolve.",
			"I salvage what washes up. Cards, mostly. The ocean gives back what it took, eventually.",
			"Green and blue together is stable enough here. The land mana and sea mana are in equilibrium. Enjoy it while it lasts.",
			"Fen's tower is east from here. I'd say be careful, but Fen is careful enough for both of you.",
		},
		RespawnMinutes: 40,
	},

	// =========================================================================
	// FAR COAST BRANCH NPCs
	// =========================================================================

	// -------------------------------------------------------------------------
	// Yso — The Abyssal Rift. Blue/black deep mage. Studies the rift. Ante FORCED.
	// Where sea mana meets something older and darker.
	// -------------------------------------------------------------------------
	{
		ID:          "yso",
		Name:        "Yso",
		Glyph:       'Y',
		Description: "A mage who has been studying the abyssal rift long enough that the rift has started to study back. Yso is not concerned by this. Yso is taking notes.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Island", 8},
			{"Swamp", 8},
			{"Counterspell", 4},
			{"Terror", 3},
			{"Hypnotic Specter", 3},
			{"Sengir Vampire", 2},
			{"Mahamoti Djinn", 2},
			{"Animate Dead", 3},
			{"Unsummon", 3},
			{"Mind Twist", 2},
			{"Dark Ritual", 2},
		},
		LootFixed:      []string{"Mahamoti Djinn"},
		LootTable:      []LootEntry{{"Mind Twist", 2}, {"Counterspell", 2}, {"Animate Dead", 2}, {"Hypnotic Specter", 2}},
		LootTableCount: 1,
		GoldReward:     65,
		Dialog: []string{
			"The abyssal rift opens into a sub-layer of the ocean floor where the mana runs in reverse. I study it. I wouldn't recommend standing in it without preparation.",
			"Blue and black work well in the deep. Control and decay have more in common than mages admit.",
			"Mind Twist isn't about winning games. It's about understanding that your opponent's hand is part of the battlefield.",
			"Sengir's influence reaches the rift. Not through the land — through the sea floor. He's more patient than people realize.",
		},
		RespawnMinutes: 60,
	},

	// -------------------------------------------------------------------------
	// Fen — Sea Mage Tower. Blue artifact/control. Gatekeeper of the archive. Ante FORCED.
	// The most demanding duelist on the coast.
	// -------------------------------------------------------------------------
	{
		ID:          "fen",
		Name:        "Fen",
		Glyph:       'F',
		Description: "The custodian of the Sea Mage Tower's archive and the most methodical duelist on the coast. Fen has three decks for different scenarios and one deck built for beauty. Guests get the beauty deck.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Island", 13},
			{"Counterspell", 4},
			{"Control Magic", 3},
			{"Mahamoti Djinn", 3},
			{"Power Sink", 3},
			{"Sol Ring", 2},
			{"Icy Manipulator", 2},
			{"Flying Men", 4},
			{"Prodigal Sorcerer", 3},
			{"Unsummon", 2},
			{"Braingeyser", 1},
		},
		LootFixed:      []string{"Sol Ring"},
		LootTable:      []LootEntry{{"Icy Manipulator", 2}, {"Mahamoti Djinn", 2}, {"Control Magic", 2}, {"Counterspell", 2}},
		LootTableCount: 2,
		GoldReward:     80,
		Dialog: []string{
			"The tower has been cataloguing sea mana patterns for sixty years. The catalogue is more useful than the mana, at this point.",
			"Sol Ring. People treat it like a lucky charm. It's an engine. Respect it accordingly.",
			"I have three decks built for specific scenarios and one I built for beauty. Guess which one I'm going to play you with.",
			"If you want to see the drowned library, prove you deserve it first. The books there are not for casual tourists.",
		},
		RespawnMinutes: 60,
	},

	// -------------------------------------------------------------------------
	// Sea Pirate — Open Waters. Red/blue aggro. No jurisdiction, no rules.
	// No ante (but offers it anyway). Fast aggro deck.
	// -------------------------------------------------------------------------
	{
		ID:          "sea_pirate",
		Name:        "The Sea Pirate",
		Glyph:       'I',
		Description: "A pirate who has been anchored off the open coast long enough to become a local fixture. No flag, no faction, no rules. Extremely competitive deck. Extremely good cards for trade.",
		Personality: "aggro",
		AnteForced:  false,
		Deck: []DeckEntry{
			{"Mountain", 8},
			{"Island", 8},
			{"Lightning Bolt", 4},
			{"Mons's Goblin Raiders", 4},
			{"Flying Men", 4},
			{"Merfolk of the Pearl Trident", 4},
			{"Fireball", 3},
			{"Unsummon", 3},
			{"Power Sink", 2},
			{"Ironclaw Orcs", 2},
		},
		LootFixed:      nil,
		LootTable:      []LootEntry{{"Lightning Bolt", 3}, {"Fireball", 3}, {"Flying Men", 2}, {"Power Sink", 1}},
		LootTableCount: 1,
		GoldReward:     50,
		Dialog: []string{
			"Open waters. No jurisdiction, no locks, no bosses. Perfect for a game.",
			"I sail red and blue. Fire and water. The coast mages think I'm an aberration. The coast mages lose.",
			"Ante? On open water? Why not. Everything's ante out here.",
			"Don't linger. The mana here is contested. You never know what's going to surface.",
		},
		RespawnMinutes: 45,
	},

	// =========================================================================
	// DEEP SOUTH CORRUPTION NPCs
	// =========================================================================

	// -------------------------------------------------------------------------
	// Urath — Sengir's Road. Black aggro/control warrior. Keeps the road.
	// Ante FORCED. The last gatekeeper before the boneyard.
	// -------------------------------------------------------------------------
	{
		ID:          "urath",
		Name:        "Urath",
		Glyph:       'U',
		Description: "A warrior who maintains Sengir's road in exchange for cards that couldn't be obtained elsewhere. Urath serves no one ideologically. The arrangement is purely transactional.",
		Personality: "aggro",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Swamp", 14},
			{"Dark Ritual", 4},
			{"Black Knight", 4},
			{"Hypnotic Specter", 3},
			{"Terror", 4},
			{"Sengir Vampire", 3},
			{"Animate Dead", 3},
			{"Drudge Skeletons", 2},
			{"Mind Twist", 2},
			{"Unholy Strength", 1},
		},
		LootFixed:      []string{"Sengir Vampire"},
		LootTable:      []LootEntry{{"Animate Dead", 2}, {"Mind Twist", 2}, {"Hypnotic Specter", 2}, {"Black Knight", 2}},
		LootTableCount: 1,
		GoldReward:     75,
		Dialog: []string{
			"Sengir's road is paved in defeat. Every mage who came down this road left something behind.",
			"I serve no one. I hold this post because it keeps me sharp and the pay is in cards I couldn't get elsewhere.",
			"Ante is required. Not negotiable. The Baron's territory runs on different rules.",
			"Past me is the boneyard. Past the boneyard is worse. I'm the last reasonable thing you'll meet going south.",
		},
		RespawnMinutes: 60,
	},

	// -------------------------------------------------------------------------
	// Zara — The Boneyard. Black necromancer. Works the yard.
	// Ante FORCED. Guards the passage to the Maw.
	// -------------------------------------------------------------------------
	{
		ID:          "zara",
		Name:        "Zara",
		Glyph:       'Z',
		Description: "A necromancer who has been working the boneyard long enough to consider it an archive. She catalogues what ends up here. The catalogue is extensive.",
		Personality: "control",
		AnteForced:  true,
		Deck: []DeckEntry{
			{"Swamp", 16},
			{"Dark Ritual", 4},
			{"Demonic Tutor", 2},
			{"Animate Dead", 4},
			{"Hypnotic Specter", 3},
			{"Sengir Vampire", 3},
			{"Royal Assassin", 2},
			{"Nightmare", 2},
			{"Mind Twist", 2},
			{"Terror", 2},
		},
		LootFixed:      []string{"Demonic Tutor"},
		LootTable:      []LootEntry{{"Nightmare", 2}, {"Animate Dead", 3}, {"Sengir Vampire", 2}, {"Royal Assassin", 1}},
		LootTableCount: 2,
		GoldReward:     100,
		Dialog: []string{
			"The boneyard is where Sengir's failures end up. Useful raw material, if you know the recipes.",
			"Demonic Tutor is a contract, not a card. Read what you're agreeing to before you put it in your deck.",
			"Every card has a ghost. Not metaphorically. On Vax, summoned things leave impressions. I collect them.",
			"You want to reach the Maw? I've beaten better mages than you in this graveyard. Impress me.",
		},
		RespawnMinutes: 70,
	},
}
