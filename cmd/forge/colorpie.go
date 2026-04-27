package main

// ColorPie encodes the mechanical and philosophical identity of each color.
// All weights are 0.0 (never) to 1.0 (signature/primary).
// Calibrated to 4th Edition era design.
type ColorPie struct {
	Color             Color
	KeywordAffinities map[Keyword]float64
	StatShape         StatShapeWeights
	EffectAccess      map[EffectType]float64
	CreatureSubtypes  []SubtypePool
	ManaPipTendency   ManaPipTendency
	Philosophy        ColorPhilosophy
}

// StatShapeWeights controls how a color distributes power vs toughness.
type StatShapeWeights struct {
	Balanced   float64
	Aggressive float64
	Defensive  float64
	Extreme    float64
}

// SubtypePool is a creature subtype with CMC range affinity.
type SubtypePool struct {
	Name         string
	MinCMC       int
	MaxCMC       int
	Weight       float64
	TribalWorthy bool
}

// ManaPipTendency controls colored vs generic mana distribution.
type ManaPipTendency struct {
	CommonSinglePip   float64
	UncommonSinglePip float64
	RareSinglePip     float64
	DoublePipBonus    float64
	TriplePipBonus    float64
}

// ColorPhilosophy holds flavor word pools for name generation.
type ColorPhilosophy struct {
	Adjectives     []string
	Nouns          []string
	Verbs          []string
	Locations      []string
	LegendaryNames []string
}

var defaultPipTendency = ManaPipTendency{
	CommonSinglePip:   0.7,
	UncommonSinglePip: 0.5,
	RareSinglePip:     0.3,
	DoublePipBonus:    0.25,
	TriplePipBonus:    0.5,
}

// PieFor returns the color pie for a given color.
func PieFor(c Color) ColorPie {
	switch c {
	case White:
		return whitePie
	case Blue:
		return bluePie
	case Black:
		return blackPie
	case Red:
		return redPie
	case Green:
		return greenPie
	case Colorless:
		return colorlessPie
	}
	return whitePie
}

// PickStatShape picks a stat shape weighted by the color's preferences.
func (sw StatShapeWeights) Pick(roll float64) StatShape {
	total := sw.Balanced + sw.Aggressive + sw.Defensive + sw.Extreme
	n := roll * total
	acc := sw.Balanced
	if n < acc {
		return Balanced
	}
	acc += sw.Aggressive
	if n < acc {
		return Aggressive
	}
	acc += sw.Defensive
	if n < acc {
		return Defensive
	}
	return Extreme
}

// ──────────────────────────────────────────────
// WHITE
// ──────────────────────────────────────────────

var whitePie = ColorPie{
	Color: White,
	KeywordAffinities: map[Keyword]float64{
		Flying: 0.55, FirstStrike: 0.65, Vigilance: 0.70, Lifelink: 0.40,
		Defender: 0.35, Reach: 0.05, Trample: 0.00, Haste: 0.00,
		Deathtouch: 0.00, Menace: 0.00, DoubleStrike: 0.10, Hexproof: 0.05,
		Indestructible: 0.05, Flash: 0.10,
	},
	StatShape: StatShapeWeights{Balanced: 0.40, Aggressive: 0.20, Defensive: 0.30, Extreme: 0.10},
	EffectAccess: map[EffectType]float64{
		DirectDamage: 0.00, DamageAllCreatures: 0.05, DestroyCreature: 0.40,
		DestroyArtifact: 0.20, DestroyEnchantment: 0.60, DestroyLand: 0.00,
		Bounce: 0.00, CounterSpell: 0.00, DrawCards: 0.00, Discard: 0.00,
		GainLife: 0.80, DrainLife: 0.00, PumpSelf: 0.15, PumpTarget: 0.35,
		TapTarget: 0.25, UntapTarget: 0.10, AddMana: 0.00,
		GlobalBuff: 0.70, AuraBuff: 0.50, ExileCreature: 0.80,
		DestroyAllCreatures: 0.70, MassReturn: 0.00, GrantKeyword: 0.40,
		DamagePlayer: 0.00, AuraDebuff: 0.30,
	},
	CreatureSubtypes: []SubtypePool{
		{"Soldier", 1, 3, 1.5, true}, {"Knight", 2, 4, 1.2, true},
		{"Cleric", 1, 3, 1.0, true}, {"Bird", 1, 3, 0.8, false},
		{"Cat", 1, 3, 0.5, false}, {"Griffin", 3, 5, 0.7, false},
		{"Angel", 4, 7, 0.8, false}, {"Archon", 5, 7, 0.3, false},
	},
	ManaPipTendency: defaultPipTendency,
	Philosophy: ColorPhilosophy{
		Adjectives: []string{
			"Holy", "Divine", "Stalwart", "Radiant", "Blessed", "Noble",
			"Sacred", "Righteous", "Vigilant", "Resolute", "Shining",
			"Serene", "Pious", "Devoted", "Sworn", "Hallowed",
			"Anointed", "Austere", "Benevolent", "Celestial", "Dauntless",
			"Exalted", "Faithful", "Gallant", "Glorious", "Immaculate",
			"Luminous", "Merciful", "Penitent", "Regal", "Solemn",
			"Steadfast", "Unbowed", "Valiant", "Venerable", "Zealous",
			"Ivory", "Gilded", "Pearlescent", "Sunlit", "Oathsworn",
			"Tireless", "Unyielding", "Wardbound", "Tempered", "Consecrated",
		},
		Nouns: []string{
			"Light", "Shield", "Blessing", "Dawn", "Faith", "Order",
			"Grace", "Valor", "Honor", "Justice", "Virtue", "Covenant",
			"Decree", "Warden", "Sentinel", "Banner",
			"Oath", "Prayer", "Halo", "Hymn", "Vigil", "Paragon",
			"Crusade", "Phalanx", "Aegis", "Absolution", "Mandate",
			"Armistice", "Bulwark", "Citadel", "Edict", "Garrison",
			"Herald", "Judicator", "Lance", "Martyr", "Pinnacle",
			"Proclamation", "Quorum", "Reprieve", "Sigil", "Tribunal",
			"Bastion", "Canticle", "Dawnbringer", "Empyrean", "Templar",
		},
		Verbs: []string{
			"Protect", "Purify", "Banish", "Heal", "Consecrate", "Smite",
			"Exalt", "Absolve", "Rally", "Fortify",
			"Anoint", "Bless", "Cleanse", "Deliver", "Embolden",
			"Guard", "Inspire", "Judge", "Muster", "Ordain",
			"Pacify", "Redeem", "Sanctify", "Unite", "Vanquish",
			"Bolster", "Denounce", "Garrison", "Liberate", "Shepherd",
		},
		Locations: []string{
			"Cathedral", "Citadel", "Monastery", "Chapel", "Sanctum",
			"Ivory Tower", "High Mesa", "Golden Meadow", "Shining City",
			"Basilica", "Barracks", "Reliquary", "Dawnspire", "Oathhold",
			"Temple", "Courtyard", "Bell Tower", "White Keep", "Hallowed Ground",
			"Sunlit Terrace", "Marble Hall", "Garrison", "Watchtower",
			"Pilgrim's Road", "Silver Gate", "Beacon Hill", "Pearlreach",
			"Alabaster Spire", "Chancery", "Judgment Hall", "Starfield",
		},
		LegendaryNames: []string{
			"Serra", "Avacyn", "Elspeth", "Orim", "Gerrard", "Akroma",
			"Thalia", "Mikaeus", "Heliod", "Brimaz", "Jazal",
			"Darien", "Kemba", "Lyra", "Nahiri", "Rune-Tail",
			"Isamaru", "Brigid", "Evra", "Linvala", "Reya",
			"Sigarda", "Gideon", "Elesh", "Linden", "Gisela",
			"Katilda", "Adeline", "Teshar", "Danitha", "Shalai",
		},
	},
}

// ──────────────────────────────────────────────
// BLUE
// ──────────────────────────────────────────────

var bluePie = ColorPie{
	Color: Blue,
	KeywordAffinities: map[Keyword]float64{
		Flying: 0.80, FirstStrike: 0.00, Vigilance: 0.00, Lifelink: 0.00,
		Defender: 0.30, Reach: 0.00, Trample: 0.00, Haste: 0.00,
		Deathtouch: 0.00, Menace: 0.00, DoubleStrike: 0.00, Hexproof: 0.30,
		Indestructible: 0.00, Flash: 0.45,
	},
	StatShape: StatShapeWeights{Balanced: 0.20, Aggressive: 0.10, Defensive: 0.55, Extreme: 0.15},
	EffectAccess: map[EffectType]float64{
		DirectDamage: 0.00, DamageAllCreatures: 0.00, DestroyCreature: 0.00,
		DestroyArtifact: 0.05, DestroyEnchantment: 0.00, DestroyLand: 0.00,
		Bounce: 0.90, CounterSpell: 1.00, DrawCards: 0.95, Discard: 0.00,
		GainLife: 0.00, DrainLife: 0.00, PumpSelf: 0.05, PumpTarget: 0.05,
		TapTarget: 0.65, UntapTarget: 0.30, AddMana: 0.00,
		GlobalBuff: 0.05, AuraBuff: 0.15, ExileCreature: 0.05,
		DestroyAllCreatures: 0.00, MassReturn: 0.80, GrantKeyword: 0.35,
		DamagePlayer: 0.00, AuraDebuff: 0.15,
	},
	CreatureSubtypes: []SubtypePool{
		{"Wizard", 1, 4, 1.3, true}, {"Merfolk", 1, 3, 1.2, true},
		{"Drake", 2, 4, 1.0, false}, {"Illusion", 1, 4, 0.7, false},
		{"Serpent", 4, 7, 0.6, false}, {"Sphinx", 4, 7, 0.5, false},
		{"Crab", 1, 2, 0.3, false}, {"Djinn", 3, 6, 0.5, false},
	},
	ManaPipTendency: ManaPipTendency{
		CommonSinglePip: 0.65, UncommonSinglePip: 0.45, RareSinglePip: 0.25,
		DoublePipBonus: 0.30, TriplePipBonus: 0.60,
	},
	Philosophy: ColorPhilosophy{
		Adjectives: []string{
			"Cunning", "Arcane", "Mystic", "Phantom", "Ethereal", "Subtle",
			"Veiled", "Thoughtful", "Elusive", "Sagacious", "Chromatic",
			"Azure", "Cerulean", "Shifting", "Calculating",
			"Astral", "Brilliant", "Cryptic", "Deepwater", "Ephemeral",
			"Fathomless", "Glacial", "Hypnotic", "Illusory", "Labyrinthine",
			"Lucid", "Mercurial", "Nebulous", "Omniscient", "Prismatic",
			"Quicksilver", "Reflective", "Shimmering", "Tidal", "Unfathomable",
			"Whispering", "Bottomless", "Conjured", "Drowning", "Frozen",
			"Invisible", "Mirrored", "Pelagic", "Sapphire", "Spectral",
		},
		Nouns: []string{
			"Mind", "Thought", "Vision", "Riddle", "Enigma", "Tide",
			"Current", "Knowledge", "Omen", "Oracle", "Insight", "Scroll",
			"Glyph", "Rune", "Scepter", "Mirror",
			"Aether", "Conduit", "Deluge", "Eddy", "Fathom",
			"Hex", "Intellect", "Labyrinth", "Maelstrom", "Nullification",
			"Paradox", "Quandary", "Reverie", "Stratagem", "Theorem",
			"Undertow", "Vortex", "Wellspring", "Zephyr", "Cascade",
			"Chronometer", "Dreamscape", "Fog", "Mirage", "Ripple",
			"Sigil", "Tempest", "Whirlpool", "Archive", "Prism",
		},
		Verbs: []string{
			"Counter", "Ponder", "Unravel", "Deceive", "Foresee", "Dispel",
			"Transmute", "Conjure", "Manipulate", "Recall",
			"Analyze", "Beguile", "Cipher", "Distort", "Erase",
			"Freeze", "Glimpse", "Hypothesize", "Imprison", "Nullify",
			"Outwit", "Perplex", "Refract", "Silence", "Twist",
			"Unmake", "Warp", "Bewilder", "Confound", "Dissolve",
		},
		Locations: []string{
			"Ivory Tower", "Academy", "Library", "Observatory", "Reef",
			"Coral Atoll", "Tidal Flats", "Sapphire Shore", "Cloud Spire",
			"Sanctum", "Lighthouse", "Archive", "Wizard's Study", "Aquifer",
			"Crystal Grotto", "Flooded Vault", "Moonlit Cove", "Nexus",
			"Pearl Lagoon", "Reflecting Pool", "Scriptorium", "Spellhall",
			"Tidal Basin", "Undersea Ruin", "Whirlpool", "Aether Spire",
			"Deepwater Keep", "Fog Bank", "Glacial Shelf", "Riptide Lab",
		},
		LegendaryNames: []string{
			"Teferi", "Jace", "Arcanis", "Talrand", "Azami", "Barrin",
			"Urza", "Ertai", "Venser", "Meloku", "Kira",
			"Naru", "Gadwick", "Tamiyo", "Thassa", "Emry",
			"Muzzio", "Braids", "Ixidor", "Keiga", "Lorthos",
			"Memnarch", "Narset", "Palinchron", "Sakashima", "Thada",
			"Vendilion", "Baral", "Callaphe", "Jin-Gitaxias", "Jalira",
		},
	},
}

// ──────────────────────────────────────────────
// BLACK
// ──────────────────────────────────────────────

var blackPie = ColorPie{
	Color: Black,
	KeywordAffinities: map[Keyword]float64{
		Flying: 0.40, FirstStrike: 0.10, Vigilance: 0.00, Lifelink: 0.25,
		Defender: 0.10, Reach: 0.00, Trample: 0.05, Haste: 0.10,
		Deathtouch: 0.70, Menace: 0.50, DoubleStrike: 0.00, Hexproof: 0.00,
		Indestructible: 0.05, Flash: 0.10,
	},
	StatShape: StatShapeWeights{Balanced: 0.30, Aggressive: 0.40, Defensive: 0.15, Extreme: 0.15},
	EffectAccess: map[EffectType]float64{
		DirectDamage: 0.05, DamageAllCreatures: 0.10, DestroyCreature: 0.95,
		DestroyArtifact: 0.00, DestroyEnchantment: 0.00, DestroyLand: 0.15,
		Bounce: 0.00, CounterSpell: 0.00, DrawCards: 0.40, Discard: 0.90,
		GainLife: 0.10, DrainLife: 0.80, PumpSelf: 0.50, PumpTarget: 0.15,
		TapTarget: 0.05, UntapTarget: 0.00, AddMana: 0.15,
		GlobalBuff: 0.15, AuraBuff: 0.40, ExileCreature: 0.10,
		DestroyAllCreatures: 0.60, MassReturn: 0.00, GrantKeyword: 0.10,
		DamagePlayer: 0.10, AuraDebuff: 0.70,
	},
	CreatureSubtypes: []SubtypePool{
		{"Zombie", 1, 4, 1.5, true}, {"Skeleton", 1, 3, 0.8, false},
		{"Vampire", 2, 5, 1.0, true}, {"Rat", 1, 2, 0.7, false},
		{"Shade", 2, 4, 0.6, false}, {"Specter", 3, 5, 0.5, false},
		{"Demon", 4, 7, 0.6, false}, {"Horror", 3, 6, 0.5, false},
	},
	ManaPipTendency: ManaPipTendency{
		CommonSinglePip: 0.60, UncommonSinglePip: 0.45, RareSinglePip: 0.30,
		DoublePipBonus: 0.30, TriplePipBonus: 0.55,
	},
	Philosophy: ColorPhilosophy{
		Adjectives: []string{
			"Dark", "Dread", "Foul", "Wretched", "Cursed", "Grim", "Fell",
			"Blighted", "Vile", "Unholy", "Malignant", "Putrid", "Ravenous",
			"Insidious", "Accursed",
			"Abyssal", "Baleful", "Cadaverous", "Desolate", "Eldritch",
			"Festering", "Ghastly", "Hollow", "Infernal", "Jagged",
			"Leprous", "Morbid", "Necrotic", "Ominous", "Pestilent",
			"Ruinous", "Sepulchral", "Tainted", "Undying", "Voracious",
			"Withered", "Ashen", "Bloodstained", "Charnel", "Decrepit",
			"Forsaken", "Grievous", "Hungering", "Loathsome", "Nightborn",
		},
		Nouns: []string{
			"Shadow", "Bone", "Grave", "Blight", "Torment", "Pestilence",
			"Doom", "Dusk", "Decay", "Agony", "Malice", "Betrayal",
			"Rot", "Pact", "Rite", "Dirge",
			"Ambition", "Blood", "Coffin", "Despair", "Effigy",
			"Filth", "Gallows", "Hemlock", "Ichor", "Knell",
			"Languish", "Miasma", "Necrosis", "Oblivion", "Plague",
			"Requiem", "Scourge", "Thrall", "Undeath", "Vendetta",
			"Wraith", "Abyss", "Carrion", "Damnation", "Entropy",
			"Famine", "Gibbet", "Husk", "Lament", "Mortis",
		},
		Verbs: []string{
			"Drain", "Corrupt", "Reanimate", "Sacrifice", "Wither", "Devour",
			"Defile", "Enslave", "Consume", "Extort",
			"Afflict", "Behead", "Condemn", "Desecrate", "Entomb",
			"Fester", "Gnaw", "Haunt", "Impale", "Leech",
			"Murder", "Necromance", "Oppress", "Plague", "Reap",
			"Shackle", "Torment", "Unearth", "Victimize", "Bleed",
		},
		Locations: []string{
			"Crypt", "Swamp", "Catacomb", "Mausoleum", "Bog", "Dungeon",
			"Charnel Pit", "Shadowmoor", "Fen", "Ossuary",
			"Gallows", "Graveyard", "Necropolis", "Oubliette", "Plague Pit",
			"Boneyard", "Black Marsh", "Cairn", "Dreadhold", "Lichyard",
			"Mire", "Pit", "Sewer", "Tomb", "Undercity",
			"Vault of Whispers", "Weeping Hollow", "Bloodwell", "Ghoul's Den",
			"Rotting Garden", "Skull Keep", "Tar Pit",
		},
		LegendaryNames: []string{
			"Lim-Dul", "Sengir", "Ihsan", "Toshiro", "Shirei", "Volrath",
			"Geth", "Liliana", "Erebos", "Kalitas", "Sheoldred",
			"Drana", "Endrek", "Gonti", "Horobi", "Ink-Eyes",
			"Josu", "Kokusho", "Marrow-Gnawer", "Nashi", "Rankle",
			"Sidisi", "Tourach", "Vito", "Whisper", "Yahenni",
			"Acererak", "Balthor", "Chainer", "Phage", "Seizan",
		},
	},
}

// ──────────────────────────────────────────────
// RED
// ──────────────────────────────────────────────

var redPie = ColorPie{
	Color: Red,
	KeywordAffinities: map[Keyword]float64{
		Flying: 0.15, FirstStrike: 0.60, Vigilance: 0.00, Lifelink: 0.00,
		Defender: 0.05, Reach: 0.00, Trample: 0.15, Haste: 0.80,
		Deathtouch: 0.00, Menace: 0.40, DoubleStrike: 0.15, Hexproof: 0.00,
		Indestructible: 0.00, Flash: 0.05,
	},
	StatShape: StatShapeWeights{Balanced: 0.20, Aggressive: 0.55, Defensive: 0.05, Extreme: 0.20},
	EffectAccess: map[EffectType]float64{
		DirectDamage: 1.00, DamageAllCreatures: 0.60, DestroyCreature: 0.10,
		DestroyArtifact: 0.60, DestroyEnchantment: 0.00, DestroyLand: 0.65,
		Bounce: 0.00, CounterSpell: 0.00, DrawCards: 0.05, Discard: 0.10,
		GainLife: 0.00, DrainLife: 0.00, PumpSelf: 0.70, PumpTarget: 0.30,
		TapTarget: 0.05, UntapTarget: 0.00, AddMana: 0.15,
		GlobalBuff: 0.10, AuraBuff: 0.20, ExileCreature: 0.00,
		DestroyAllCreatures: 0.15, MassReturn: 0.00, GrantKeyword: 0.35,
		DamagePlayer: 0.80, AuraDebuff: 0.00,
	},
	CreatureSubtypes: []SubtypePool{
		{"Goblin", 1, 3, 1.8, true}, {"Orc", 2, 4, 0.7, false},
		{"Elemental", 2, 5, 0.8, false}, {"Barbarian", 1, 4, 0.6, false},
		{"Dwarf", 1, 3, 0.5, false}, {"Phoenix", 3, 5, 0.3, false},
		{"Dragon", 4, 7, 0.7, false}, {"Viashino", 2, 4, 0.4, false},
	},
	ManaPipTendency: ManaPipTendency{
		CommonSinglePip: 0.75, UncommonSinglePip: 0.55, RareSinglePip: 0.35,
		DoublePipBonus: 0.20, TriplePipBonus: 0.45,
	},
	Philosophy: ColorPhilosophy{
		Adjectives: []string{
			"Raging", "Volcanic", "Fierce", "Wild", "Burning", "Thunder",
			"Blazing", "Reckless", "Furious", "Seething", "Scorching",
			"Frenzied", "Savage", "Molten", "Chaotic",
			"Berserking", "Bloodshot", "Calamitous", "Destructive", "Explosive",
			"Fiery", "Guttering", "Hellbent", "Impetuous", "Jagged",
			"Kindled", "Livid", "Maddening", "Needling", "Onrushing",
			"Pyretic", "Rabid", "Smoldering", "Thunderous", "Unstoppable",
			"Wrathful", "Ashen", "Blistering", "Crackling", "Defiant",
			"Fervent", "Glowing", "Heated", "Incandescent", "Relentless",
		},
		Nouns: []string{
			"Flame", "Thunder", "Lightning", "Fury", "Blaze", "Ash",
			"Magma", "Inferno", "Rage", "Ember", "Spark", "Eruption",
			"Hammer", "Bolt", "Shrapnel", "Avalanche",
			"Anvil", "Barrage", "Cinder", "Detonation", "Earthquake",
			"Firestorm", "Geyser", "Havoc", "Impact", "Juggernaut",
			"Keg", "Landslide", "Mortar", "Napalm", "Ordnance",
			"Pyre", "Quarrel", "Rampage", "Salvo", "Tempest",
			"Upheaval", "Volley", "Warhorn", "Brand", "Crucible",
			"Flare", "Gauntlet", "Hellfire", "Incendiary", "Slag",
		},
		Verbs: []string{
			"Burn", "Ignite", "Shatter", "Raze", "Erupt", "Scorch",
			"Incinerate", "Demolish", "Unleash", "Detonate",
			"Annihilate", "Bombard", "Char", "Dismantle", "Engulf",
			"Fling", "Gouge", "Hurl", "Immolate", "Kindle",
			"Lacerate", "Melt", "Obliterate", "Pummel", "Ravage",
			"Sear", "Topple", "Upend", "Wreck", "Combust",
		},
		Locations: []string{
			"Volcano", "Forge", "Crater", "Furnace", "Caldera",
			"Cinder Marsh", "Lava Field", "Mountain Pass", "Obsidian Peak",
			"Arena", "Badlands", "Clifftop", "Dragon's Lair", "Ember Gorge",
			"Fissure", "Geothermal Vent", "Hellmouth", "Iron Mine", "Kiln",
			"Lava Tube", "Magma Chamber", "Powder Keg", "Ridgeline",
			"Scorched Earth", "Tinderbox", "War Camp", "Ashmouth",
			"Battleground", "Cauldron", "Firepit", "Smoldering Ruins",
		},
		LegendaryNames: []string{
			"Jaya", "Koth", "Purphoros", "Balthor", "Lovisa", "Urabrask",
			"Daretti", "Chandra", "Krenko", "Neheb", "Feldon",
			"Godo", "Heartless", "Ilharg", "Kazuul", "Lathliss",
			"Magda", "Norin", "Pashalik", "Rograkh", "Squee",
			"Toggo", "Valduk", "Wort", "Zurzoth",
			"Angrath", "Borborygmos", "Drakuseth", "Etali", "Torbran",
		},
	},
}

// ──────────────────────────────────────────────
// GREEN
// ──────────────────────────────────────────────

var greenPie = ColorPie{
	Color: Green,
	KeywordAffinities: map[Keyword]float64{
		Flying: 0.00, FirstStrike: 0.00, Vigilance: 0.25, Lifelink: 0.10,
		Defender: 0.15, Reach: 0.70, Trample: 0.85, Haste: 0.10,
		Deathtouch: 0.25, Menace: 0.00, DoubleStrike: 0.00, Hexproof: 0.20,
		Indestructible: 0.05, Flash: 0.10,
	},
	StatShape: StatShapeWeights{Balanced: 0.45, Aggressive: 0.25, Defensive: 0.20, Extreme: 0.10},
	EffectAccess: map[EffectType]float64{
		DirectDamage: 0.00, DamageAllCreatures: 0.00, DestroyCreature: 0.05,
		DestroyArtifact: 0.40, DestroyEnchantment: 0.50, DestroyLand: 0.00,
		Bounce: 0.00, CounterSpell: 0.00, DrawCards: 0.10, Discard: 0.00,
		GainLife: 0.40, DrainLife: 0.00, PumpSelf: 0.30, PumpTarget: 0.85,
		TapTarget: 0.00, UntapTarget: 0.10, AddMana: 0.90,
		GlobalBuff: 0.20, AuraBuff: 0.30, ExileCreature: 0.00,
		DestroyAllCreatures: 0.00, MassReturn: 0.00, GrantKeyword: 0.30,
		DamagePlayer: 0.00, AuraDebuff: 0.00,
	},
	CreatureSubtypes: []SubtypePool{
		{"Elf", 1, 3, 1.6, true}, {"Bear", 2, 3, 0.8, false},
		{"Spider", 2, 4, 0.7, false}, {"Beast", 3, 6, 1.0, false},
		{"Treefolk", 3, 6, 0.6, false}, {"Wurm", 5, 8, 0.8, false},
		{"Elemental", 3, 6, 0.5, false}, {"Snake", 1, 3, 0.4, false},
	},
	ManaPipTendency: defaultPipTendency,
	Philosophy: ColorPhilosophy{
		Adjectives: []string{
			"Ancient", "Towering", "Primal", "Feral", "Verdant", "Mighty",
			"Wild", "Untamed", "Colossal", "Primeval", "Lush", "Enormous",
			"Ferocious", "Earthen", "Overgrown",
			"Bestial", "Boundless", "Carnivorous", "Dense", "Enduring",
			"Flourishing", "Gargantuan", "Hulking", "Immense", "Jungled",
			"Keen-Fanged", "Lumbering", "Mammoth", "Nesting", "Ornery",
			"Predatory", "Rampaging", "Savage", "Thundering", "Uprooted",
			"Venomous", "Weathered", "Ageless", "Bristling", "Creeping",
			"Deeproot", "Evergreen", "Gnarled", "Hibernating", "Ironhide",
		},
		Nouns: []string{
			"Claw", "Fang", "Root", "Vine", "Thorn", "Growth",
			"Canopy", "Briar", "Horn", "Maw", "Bark", "Undergrowth",
			"Harvest", "Bloom", "Stampede", "Prey",
			"Antler", "Burrow", "Chrysalis", "Den", "Emergence",
			"Frond", "Glade", "Hive", "Instinct", "Jaw",
			"Kin", "Lichen", "Moss", "Nest", "Outcrop",
			"Pack", "Quill", "Rampart", "Sap", "Talon",
			"Underbrush", "Verdure", "Wilds", "Apex", "Bough",
			"Copse", "Druid", "Foliage", "Gorge", "Heartwood",
		},
		Verbs: []string{
			"Grow", "Trample", "Devour", "Flourish", "Overwhelm", "Uproot",
			"Cultivate", "Overrun", "Entangle", "Enrage",
			"Ambush", "Bolster", "Consume", "Dominate", "Evolve",
			"Forage", "Germinate", "Hibernate", "Invoke", "Lure",
			"Maul", "Nurture", "Outgrow", "Prowl", "Regenerate",
			"Sprout", "Thrash", "Unleash", "Verdure", "Wrestle",
		},
		Locations: []string{
			"Forest", "Jungle", "Grove", "Thicket", "Glen", "Wildwood",
			"Thornwood", "Emerald Valley", "Primeval Stand",
			"Canopy", "Clearing", "Deepwood", "Elder Grove", "Fernmoor",
			"Grassland", "Hollow", "Ironwood", "Kessig Trail", "Llanowar",
			"Marshwood", "Nantuko Colony", "Oakenhold", "Pinewood",
			"Rootweb", "Savannah", "Tanglewood", "Undergrowth", "Vineyard",
			"Wolfrun", "Briarwood", "Cragwood", "Duskwood", "Elfhame",
		},
		LegendaryNames: []string{
			"Titania", "Kaysa", "Kamahl", "Yavimaya", "Garruk", "Nylea",
			"Ezuri", "Selvala", "Freyalise", "Multani", "Yisan",
			"Ayula", "Baru", "Challenger", "Dosan", "Eladamri",
			"Fyndhorn", "Ghalta", "Iwamori", "Jasmine", "Kodama",
			"Lhurgoyf", "Marwyn", "Nissa", "Oviya", "Polukranos",
			"Rishkar", "Surrak", "Thrun", "Vigor", "Xenagos",
		},
	},
}

// ──────────────────────────────────────────────
// COLORLESS
// ──────────────────────────────────────────────

var colorlessPie = ColorPie{
	Color: Colorless,
	KeywordAffinities: map[Keyword]float64{
		Flying: 0.20, FirstStrike: 0.10, Vigilance: 0.15, Lifelink: 0.05,
		Defender: 0.30, Reach: 0.10, Trample: 0.20, Haste: 0.05,
		Deathtouch: 0.05, Menace: 0.10, DoubleStrike: 0.05, Hexproof: 0.05,
		Indestructible: 0.10, Flash: 0.05,
	},
	StatShape: StatShapeWeights{Balanced: 0.50, Aggressive: 0.15, Defensive: 0.25, Extreme: 0.10},
	EffectAccess: map[EffectType]float64{
		DirectDamage: 0.10, DamageAllCreatures: 0.05, DestroyCreature: 0.10,
		DestroyArtifact: 0.10, DestroyEnchantment: 0.05, DestroyLand: 0.05,
		Bounce: 0.10, CounterSpell: 0.00, DrawCards: 0.15, Discard: 0.05,
		GainLife: 0.15, DrainLife: 0.05, PumpSelf: 0.20, PumpTarget: 0.15,
		TapTarget: 0.30, UntapTarget: 0.15, AddMana: 0.50,
		GlobalBuff: 0.05, AuraBuff: 0.00, ExileCreature: 0.05,
		DestroyAllCreatures: 0.05, MassReturn: 0.05, GrantKeyword: 0.10,
		DamagePlayer: 0.05, AuraDebuff: 0.00,
	},
	CreatureSubtypes: []SubtypePool{
		{"Golem", 3, 6, 1.0, false}, {"Construct", 2, 5, 0.8, false},
		{"Wall", 1, 4, 0.6, false}, {"Gargoyle", 3, 5, 0.5, false},
		{"Juggernaut", 4, 6, 0.3, false},
	},
	ManaPipTendency: ManaPipTendency{
		CommonSinglePip: 1.0, UncommonSinglePip: 1.0, RareSinglePip: 1.0,
		DoublePipBonus: 0.0, TriplePipBonus: 0.0,
	},
	Philosophy: ColorPhilosophy{
		Adjectives: []string{
			"Iron", "Obsidian", "Crystal", "Brass", "Rusted", "Gilded",
			"Mechanical", "Clockwork", "Stone", "Steel", "Ornate",
			"Adamantine", "Bronze", "Cobalt", "Darksteel", "Etched",
			"Filigree", "Galvanized", "Hardened", "Inert", "Jeweled",
			"Latticed", "Mithral", "Nickel", "Oxidized", "Plated",
			"Reinforced", "Silvered", "Tempered", "Unwound", "Vitreous",
			"Welded", "Ancient", "Battered", "Chromed", "Dormant",
		},
		Nouns: []string{
			"Machine", "Gear", "Anvil", "Colossus", "Monolith", "Obelisk",
			"Edifice", "Engine", "Vessel", "Shard", "Relic", "Totem",
			"Apparatus", "Bellows", "Cog", "Dynamo", "Effigy",
			"Furnace", "Girder", "Hydraulic", "Ingot", "Juncture",
			"Keystone", "Lattice", "Mechanism", "Node", "Orb",
			"Piston", "Reactor", "Sentinel", "Turbine", "Urn",
			"Vise", "Windmill", "Axis", "Bastion", "Crucible",
		},
		Verbs: []string{
			"Forge", "Assemble", "Animate", "Construct", "Activate", "Grind",
			"Anneal", "Bolt", "Calibrate", "Disassemble", "Excavate",
			"Fuse", "Galvanize", "Hammer", "Ignite", "Link",
			"Meld", "Overload", "Polish", "Reinforce", "Smelt",
			"Tighten", "Weld", "Alloy", "Bore", "Crank",
		},
		Locations: []string{
			"Foundry", "Scrapyard", "Mineshaft", "Workshop", "Vault",
			"Ruins", "Monument",
			"Armory", "Blast Furnace", "Clocktower", "Depot", "Engine Room",
			"Factory", "Gallery", "Ironworks", "Junkyard", "Kiln",
			"Labyrinth", "Manufactory", "Quarry", "Refinery", "Smithy",
			"Tinker's Den", "Warehouse", "Assembly Hall", "Bunker",
			"Darksteel Citadel", "Edifice", "Grinding Station", "Phyrexia",
		},
		LegendaryNames: []string{
			"Karn", "Mishra", "Arcum", "Feldon", "Memnarch", "Bosh",
			"Arcbound", "Darksteel", "Epochrasite", "Golos", "Hangarback",
			"Ichorclaw", "Jhoira", "Kozilek", "Liberator", "Masticore",
			"Nettlecyst", "Pentavus", "Rishadan", "Scuttlemutt", "Triskelion",
			"Ulamog", "Voltaic", "Wurmcoil", "Brudiclad",
		},
	},
}
