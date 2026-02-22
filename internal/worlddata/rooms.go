package worlddata

// allRooms is the complete room graph for the world.
// Edit this file freely — add rooms, change descriptions, rewire exits.
// NPCIDs must match an ID defined in npcs.go.
//
// The plane is Vax — eight rooms, three acts.
// Act I: Arrival and orientation at Tangle's Edge.
// Act II: The Dueling Road and its branches; build your deck.
// Act III: The Thornwood corruption, the Ashen Chapel, and Baron Sengir's Dread Keep.
var allRooms = []RoomDef{

	// =========================================================================
	// ACT I — ARRIVAL
	// =========================================================================

	{
		ID:   "threshold",
		Name: "The Waystone",
		Description: `The Waystone still hums with residual planar energy — whatever brought you here
hasn't fully released its grip. You are on the plane of Vax.
The air carries distinct mana signatures: green from the forests to the south,
and something else underneath, heavy and dark, pushing in from further south still.
A battered chest rests against the base of the stone. Travelers leave them here
for those who arrive without a deck.`,
		MapX:       13,
		MapY:       0,
		Exits:      map[string]string{"south": "tangled_edge", "west": "greenwood_rise", "east": "tidal_watch"},
		StartChest: true,
	},

	{
		ID:   "tangled_edge",
		Name: "Tangle's Edge",
		Description: `A settlement built into the roots of three enormous ironwood trees, their bark worn
smooth where generations of mages have leaned between duels. The mana here is thick
enough to taste — forest-green, a little electric.
Elder Maren tends the lanterns. She'll tell you about Vax. She'll warn you about
Baron Sengir whether you ask or not.`,
		MapX:   13,
		MapY:   9,
		Exits:  map[string]string{"north": "threshold", "east": "bazaar", "south": "dueling_road", "west": "mosshallow"},
		NPCIDs: []string{"elder_maren"},
	},

	{
		ID:   "bazaar",
		Name: "The Bazaar",
		Description: `A covered trading post strung with artifact lanterns that Merchant Varro keeps lit
regardless of the hour. He deals in cards, mana components, and information —
in roughly that order of reliability.
A collapsed display case near the back hasn't been cleared. The dust has settled
in interesting shapes around something underneath.`,
		MapX:   26,
		MapY:   9,
		Exits:  map[string]string{"west": "tangled_edge", "east": "harbormouth"},
		NPCIDs: []string{"merchant_varro"},
		Hidden: []Treasure{
			{
				// A Sol Ring in the rubble. Very hard to spot under the wreckage.
				Cards:    []string{"Sol Ring"},
				SearchDC: 4,
			},
		},
	},

	// =========================================================================
	// ACT II — THE DUELING ROAD
	// =========================================================================

	{
		ID:   "dueling_road",
		Name: "The Dueling Road",
		Description: `The road widens into a flat clearing — the paving stones are scorched in places
from where summons have hit the ground and refused to dissolve cleanly.
Two duelists keep their distance: a road-worn woman in grey watching everything,
and a scavenger working his knuckles like he's counting something.
To the west, a vault door with a coin-lock mechanism. Someone trusted gold more
than a ward.`,
		MapX: 13,
		MapY: 18,
		Exits: map[string]string{
			"north": "tangled_edge",
			"east":  "thornwood",
			"west":  "wardens_vault", // gold-locked — see GoldLock on the vault room
			"south": "pilgrim_road",
		},
		NPCIDs: []string{"wandering_duelist", "scavenger_theron"},
	},

	{
		ID:   "wardens_vault",
		Name: "The Warden's Vault",
		Description: `The vault is older than the settlement above. The walls are slotted with thousands
of card-sleeves — most empty, a few still sealed. Whatever this archive once held,
most of it has gone south.
The Warden sits at the center table with a deck spread before them, thinking.
The air smells of old cardstock and forest mana.`,
		MapX:     0,
		MapY:     18,
		Exits:    map[string]string{"east": "dueling_road", "north": "mosshallow", "south": "rootweave"},
		NPCIDs:   []string{"the_warden"},
		GoldLock: 75,
		Hidden: []Treasure{
			{
				// A Regrowth filed in the wrong sleeve — easy to notice if you look.
				Cards:    []string{"Regrowth"},
				SearchDC: 2,
			},
			{
				// Two Llanowar Elves tucked inside a sealed tome. Harder.
				Cards:    []string{"Llanowar Elves", "Llanowar Elves"},
				SearchDC: 3,
			},
		},
	},

	// =========================================================================
	// ACT III — THE CORRUPTION
	// =========================================================================

	{
		ID:   "thornwood",
		Name: "The Thornwood",
		Description: `The forest here is wrong. Green mana still moves through it — you can feel the pull —
but black mana underlays everything, rising like a tide held just below the surface.
The path south is clear. Someone has been using it recently.
A cairn of stones marks something buried at the path's edge. The top stone is
carved with a crude glyph. You don't recognize the school.`,
		MapX: 26,
		MapY: 18,
		Exits: map[string]string{
			"west":  "dueling_road",
			"south": "ashen_chapel",
			"east":  "tide_gate",
		},
		Hidden: []Treasure{
			{
				// Berserk buried under the cairn — but disturbing the cairn wakes something.
				Cards:    []string{"Berserk"},
				SearchDC: 3,
				Trap:     true,
			},
		},
	},

	{
		ID:   "ashen_chapel",
		Name: "The Ashen Chapel",
		Description: `The chapel was built to honor Vax's mana currents. Someone has been undoing that.
The altar's mana inlays have been reversed — they drain now instead of give. You can
feel it pulling at your reserves just standing here.
Sister Vael kneels at the altar. Her deck is on the stone beside her, face down.
On the offering shelf: a carved wooden box that radiates cold across the room.`,
		MapX:   26,
		MapY:   27,
		Exits:  map[string]string{"north": "thornwood", "south": "dread_keep", "west": "pilgrim_road"},
		NPCIDs: []string{"sister_vael"},
		Hidden: []Treasure{
			{
				// The offering box. Cold. Wrong. Powerful.
				Cards:    []string{"Demonic Tutor", "Animate Dead"},
				SearchDC: 2,
				Trap:     true,
			},
		},
	},

	{
		ID:   "dread_keep",
		Name: "The Dread Keep",
		Description: `Baron Sengir has had time to make this his.
Black mana runs through the stonework like veins — the air is thick with it, almost
visible. Every card in your deck feels heavier in this room.
He waits at the far end of the hall. He has been expecting someone for a while now.
He doesn't look impatient. He looks hungry.`,
		MapX:     26,
		MapY:     36,
		Exits:    map[string]string{"north": "ashen_chapel", "south": "sengir_road", "west": "charnel_fork"},
		NPCIDs:   []string{"baron_sengir"},
		BossLock: "sister_vael",
	},

	// =========================================================================
	// WESTERN GREEN BRANCH  (MapX = 0)
	// Five rooms in the ancient forest west of the main road.
	// =========================================================================

	{
		ID:   "greenwood_rise",
		Name: "Greenwood Rise",
		Description: `A clearing at the crest of the oldest ridge in Vax's forest, where the green mana
is cleanest and the canopy thickest. Travelers arriving from the east find the air
here noticeably easier to breathe.
A mage in white-trimmed green leathers stands at the ridge's edge, watching both
directions at once.`,
		MapX:   0,
		MapY:   0,
		Exits:  map[string]string{"east": "threshold", "south": "mosshallow"},
		NPCIDs: []string{"asha"},
	},

	{
		ID:   "mosshallow",
		Name: "The Mosshallow",
		Description: `A shallow depression in the forest floor where green and blue mana pool together,
feeding a permanent layer of luminescent moss. Studying it could take a lifetime.
The mage crouched at the edge of the pool doesn't look like they've slept recently.`,
		MapX:   0,
		MapY:   9,
		Exits:  map[string]string{"north": "greenwood_rise", "east": "tangled_edge", "south": "wardens_vault"},
		NPCIDs: []string{"orvyn"},
		Hidden: []Treasure{
			{
				// Giant Growths pressed flat under the moss — obvious to a botanist.
				Cards:    []string{"Giant Growth", "Giant Growth"},
				SearchDC: 2,
			},
		},
	},

	{
		ID:   "rootweave",
		Name: "The Rootweave",
		Description: `The forest floor here is a lattice of exposed roots so thick you have to watch
every step. The trees share root systems. They share mana. They share memory,
according to the druid who has lived here for the last thirty years.`,
		MapX:   0,
		MapY:   27,
		Exits:  map[string]string{"north": "wardens_vault", "south": "verdant_hollow"},
		NPCIDs: []string{"cael"},
		Hidden: []Treasure{
			{
				// Regrowth tucked into a hollow root. Hard to spot among the tangle.
				Cards:    []string{"Regrowth"},
				SearchDC: 3,
			},
		},
	},

	{
		ID:   "verdant_hollow",
		Name: "Verdant Hollow",
		Description: `A natural amphitheater in the forest where beast mana concentrates. Summoned
creatures from duels fought here have a habit of not dissolving properly.
The beast tamer with the Kird Apes doesn't seem to mind the company.`,
		MapX:   0,
		MapY:   36,
		Exits:  map[string]string{"north": "rootweave", "east": "charnel_fork", "south": "ancient_heart"},
		NPCIDs: []string{"greth"},
	},

	{
		ID:   "ancient_heart",
		Name: "The Ancient Heart",
		Description: `Deeper than any trail goes, where the oldest tree in the forest has been growing
since before mages came to Vax. The green mana here has a texture — almost solid.
Something valuable is buried at the base. The tree doesn't seem to mind you looking.`,
		MapX:     0,
		MapY:     45,
		Exits:    map[string]string{"north": "verdant_hollow"},
		BossLock: "greth",
		Hidden: []Treasure{
			{
				// Berserk coiled in the root system like it was planted intentionally.
				Cards:    []string{"Berserk"},
				SearchDC: 4,
			},
			{
				// Force of Nature. Buried deep. Unstable mana resonance marks the spot.
				Cards:    []string{"Force of Nature"},
				SearchDC: 5,
				Trap:     true,
			},
		},
	},

	// =========================================================================
	// CENTRAL SOUTH BRANCH  (MapX = 13)
	// Three rooms continuing the main road past the Dueling Road.
	// =========================================================================

	{
		ID:   "pilgrim_road",
		Name: "The Pilgrim's Road",
		Description: `The road narrows south of the Dueling Road. Whatever traffic once moved through here
has thinned to the occasional desperate mage heading toward the Thornwood corruption.
A knight in road-worn armor has been holding position at the crossroads.`,
		MapX:   13,
		MapY:   27,
		Exits:  map[string]string{"north": "dueling_road", "east": "ashen_chapel", "south": "charnel_fork"},
		NPCIDs: []string{"pardoned_knight"},
	},

	{
		ID:   "charnel_fork",
		Name: "Charnel Fork",
		Description: `Three paths converge at a scorched junction where the black mana from the south
meets the dying green mana from the west. The paving stones are cracked where
something heavy and wrong passed through recently.
A wiry scavenger picks through the debris with professional focus.`,
		MapX:   13,
		MapY:   36,
		Exits:  map[string]string{"north": "pilgrim_road", "east": "dread_keep", "west": "verdant_hollow", "south": "hollow_deep"},
		NPCIDs: []string{"nixx"},
	},

	{
		ID:   "hollow_deep",
		Name: "The Hollow Deep",
		Description: `Below the fork, the path descends into a ravine where black mana pools in the low
places and the air smells of old graves. Light doesn't reach the bottom cleanly.
Something that was a mage, and still is — mostly — waits at the bottom.`,
		MapX:   13,
		MapY:   45,
		Exits:  map[string]string{"north": "charnel_fork", "east": "sengir_road"},
		NPCIDs: []string{"vex"},
	},

	// =========================================================================
	// EASTERN COASTAL BRANCH  (MapX = 39)
	// Four rooms along the coast, east of the main arc.
	// =========================================================================

	{
		ID:   "tidal_watch",
		Name: "The Tidal Watch",
		Description: `A coastal watchtower that has been manned continuously for two hundred years.
The sea mana here is clean — the blue current flows north and the forest green
holds it in place. Whatever's happening to the south hasn't reached here yet.
The sentinel in the tower has good sight lines in every direction.`,
		MapX:   39,
		MapY:   0,
		Exits:  map[string]string{"west": "threshold", "south": "harbormouth"},
		NPCIDs: []string{"lyss"},
		Hidden: []Treasure{
			{
				// Unsummon scrolls stored in the duty locker. Standard coastal issue.
				Cards:    []string{"Unsummon", "Unsummon"},
				SearchDC: 2,
			},
		},
	},

	{
		ID:   "harbormouth",
		Name: "Harbormouth",
		Description: `The busiest port still operating on Vax. Three mana currents converge in the harbor:
sea blue, mountain red from the inland trade routes, and the forest green just
barely reaching the coast.
The harbor master runs a tight operation. He also runs a competitive deck.`,
		MapX:   39,
		MapY:   9,
		Exits:  map[string]string{"north": "tidal_watch", "west": "bazaar", "south": "tide_gate"},
		NPCIDs: []string{"rhen"},
		Hidden: []Treasure{
			{
				// Icy Manipulator in an unregistered cargo crate. Someone's loss.
				Cards:    []string{"Icy Manipulator"},
				SearchDC: 4,
			},
		},
	},

	{
		ID:   "tide_gate",
		Name: "The Tide Gate",
		Description: `A massive sea-lock that regulates mana flow along the coast. The mechanism is old
and precise — and so is the mage who maintains it.
The gate blocks passage south until you can demonstrate you belong there.`,
		MapX:     39,
		MapY:     18,
		Exits:    map[string]string{"north": "harbormouth", "west": "thornwood", "east": "abyssal_rift", "south": "shipwreck_shore"},
		NPCIDs:   []string{"mora"},
		GoldLock: 50,
	},

	{
		ID:   "shipwreck_shore",
		Name: "Shipwreck Shore",
		Description: `The beach where three famous ships came to rest when the sea mana destabilized
during a duel offshore. The wreckage is permanent — the mana residue is too
thick for the wood to rot. A salvager works the debris methodically.`,
		MapX:   39,
		MapY:   27,
		Exits:  map[string]string{"north": "tide_gate", "east": "sea_mage_tower"},
		NPCIDs: []string{"pelth"},
	},

	// =========================================================================
	// FAR COAST BRANCH  (MapX = 52)
	// Four rooms on the deep coast east of the shipping lanes.
	// =========================================================================

	{
		ID:   "abyssal_rift",
		Name: "The Abyssal Rift",
		Description: `A crack in the sea floor that extends down farther than any sounding has reached.
Sea mana and something older flows up through it. The mage studying the rift
has been here long enough to stop looking disturbed.`,
		MapX:     52,
		MapY:     18,
		Exits:    map[string]string{"west": "tide_gate", "south": "sea_mage_tower"},
		NPCIDs:   []string{"yso"},
		BossLock: "mora",
	},

	{
		ID:   "sea_mage_tower",
		Name: "Sea Mage Tower",
		Description: `A tower built directly over a sea mana confluence sixty years ago. The original
builders are gone. The collection they left behind is not.
Fen catalogues what remains. Fen also decides who gets access.`,
		MapX:   52,
		MapY:   27,
		Exits:  map[string]string{"north": "abyssal_rift", "west": "shipwreck_shore", "south": "open_waters"},
		NPCIDs: []string{"fen"},
		Hidden: []Treasure{
			{
				// A Sol Ring misfiled under 'S' in the artifact catalogue.
				Cards:    []string{"Sol Ring"},
				SearchDC: 5,
			},
		},
	},

	{
		ID:   "open_waters",
		Name: "Open Waters",
		Description: `Beyond the tower's sheltered harbor, the sea stretches without obstruction.
Mana here is fluid and contested — it belongs to whoever can hold it.
A ship flying no colors is anchored just offshore. A plank runs to the beach.`,
		MapX:   52,
		MapY:   36,
		Exits:  map[string]string{"north": "sea_mage_tower", "south": "drowned_library"},
		NPCIDs: []string{"sea_pirate"},
	},

	{
		ID:   "drowned_library",
		Name: "The Drowned Library",
		Description: `A structure that was built below the waterline deliberately — the sea mana acts as
a preservative. The books and cards here have survived the centuries. A few of
the stranger ones may have been improved by immersion.`,
		MapX:     52,
		MapY:     45,
		Exits:    map[string]string{"north": "open_waters"},
		BossLock: "fen",
		Hidden: []Treasure{
			{
				// Braingeyser in a waterproof case. Well-labeled.
				Cards:    []string{"Braingeyser"},
				SearchDC: 3,
			},
			{
				// Two Control Magics shelved under an incorrect subject heading.
				Cards:    []string{"Control Magic", "Control Magic"},
				SearchDC: 4,
			},
		},
	},

	// =========================================================================
	// DEEP SOUTH CORRUPTION  (MapX = 26)
	// Four rooms continuing south from the Dread Keep into Sengir's domain.
	// =========================================================================

	{
		ID:   "sengir_road",
		Name: "Sengir's Road",
		Description: `The road south of the keep has been resurfaced in black stone that pulses faintly.
The mana here is entirely Sengir's — dense, purposeful, hungry.
A warrior in black armor maintains the road with mechanical efficiency.`,
		MapX:   26,
		MapY:   45,
		Exits:  map[string]string{"north": "dread_keep", "west": "hollow_deep", "south": "bone_yard"},
		NPCIDs: []string{"urath"},
		Hidden: []Treasure{
			{
				// Dark Rituals cached under a loose road stone. Standard issue here.
				Cards:    []string{"Dark Ritual", "Dark Ritual"},
				SearchDC: 2,
			},
		},
	},

	{
		ID:   "bone_yard",
		Name: "The Boneyard",
		Description: `The place where Sengir sends what he's finished with. The terrain is rough with
jutting shapes that are better not examined closely.
A necromancer works the yard with the focused calm of someone who finds the material
conditions ideal.`,
		MapX:     26,
		MapY:     54,
		Exits:    map[string]string{"north": "sengir_road", "south": "maw_of_vax"},
		NPCIDs:   []string{"zara"},
		BossLock: "urath",
	},

	{
		ID:   "maw_of_vax",
		Name: "The Maw of Vax",
		Description: `A fissure in the plane's surface where the black mana is so concentrated that
the air is visibly dark. This is where Sengir draws power — the source point
for the corruption spreading north.
Everything of value that has been lost to the black mana has drained here eventually.`,
		MapX:     26,
		MapY:     63,
		Exits:    map[string]string{"north": "bone_yard", "south": "sunken_citadel"},
		BossLock: "zara",
		Hidden: []Treasure{
			{
				// Dark Rituals and a Mind Twist drawn into the mana sink. Tainted but functional.
				Cards:    []string{"Dark Ritual", "Dark Ritual", "Mind Twist"},
				SearchDC: 3,
				Trap:     true,
			},
			{
				// Demonic Tutor. It chose to end up here. That should tell you something.
				Cards:    []string{"Demonic Tutor"},
				SearchDC: 5,
				Trap:     true,
			},
		},
	},

	{
		ID:   "sunken_citadel",
		Name: "The Sunken Citadel",
		Description: `Below the Maw, deeper than anything should be on a terrestrial plane, sits a
structure that predates Sengir's arrival. He didn't build it. He moved in.
The cards here are from collections that were drained completely — the last
remnants of mages who came south and didn't come back.`,
		MapX:  26,
		MapY:  72,
		Exits: map[string]string{"north": "maw_of_vax"},
		Hidden: []Treasure{
			{
				// Three powerful black cards left behind by defeated mages.
				Cards:    []string{"Nightmare", "Sengir Vampire", "Sengir Vampire"},
				SearchDC: 2,
				Trap:     true,
			},
		},
	},
}
