package worlddata

// StarterArchetype is one of the starting deck options offered to a new player.
type StarterArchetype struct {
	Name        string
	Description string
	Deck        []DeckEntry
}

// StarterArchetypes are the deck choices available from the starter chest.
// The player picks one; those cards become their initial collection.
var StarterArchetypes = []StarterArchetype{
	{
		Name:        "White/Green Aggro",
		Description: "Fast creatures and pump spells. Get on the board early and apply pressure.",
		Deck: []DeckEntry{
			{"Plains", 9},
			{"Forest", 7},
			{"Savannah Lions", 4},
			{"Llanowar Elves", 4},
			{"White Knight", 4},
			{"Grizzly Bears", 4},
			{"Mesa Pegasus", 3},
			{"Serra Angel", 2},
			{"Giant Growth", 3},
			{"Healing Salve", 2},
			{"Swords to Plowshares", 2},
		},
	},
	{
		Name:        "Red/Black Aggro",
		Description: "Burn spells and fast threats. Deal damage directly and clear the path.",
		Deck: []DeckEntry{
			{"Mountain", 8},
			{"Swamp", 8},
			{"Lightning Bolt", 4},
			{"Mons's Goblin Raiders", 4},
			{"Black Knight", 4},
			{"Ironclaw Orcs", 4},
			{"Hypnotic Specter", 3},
			{"Terror", 3},
			{"Hill Giant", 2},
			{"Dark Ritual", 3},
		},
	},
	{
		Name:        "Blue/White Control",
		Description: "Counter threats and answer everything. Win with flyers after stabilizing.",
		Deck: []DeckEntry{
			{"Island", 9},
			{"Plains", 7},
			{"Counterspell", 4},
			{"Disenchant", 3},
			{"Swords to Plowshares", 3},
			{"Merfolk of the Pearl Trident", 4},
			{"White Knight", 3},
			{"Serra Angel", 2},
			{"Air Elemental", 2},
			{"Unsummon", 3},
			{"Healing Salve", 2},
		},
	},
	{
		Name:        "Green Ramp",
		Description: "Accelerate your mana and deploy large threats before your opponent is ready.",
		Deck: []DeckEntry{
			{"Forest", 16},
			{"Llanowar Elves", 4},
			{"Grizzly Bears", 4},
			{"Giant Spider", 3},
			{"Elvish Archers", 3},
			{"War Mammoth", 3},
			{"Craw Wurm", 2},
			{"Giant Growth", 3},
			{"Regrowth", 2},
			{"Berserk", 1},
		},
	},
}
