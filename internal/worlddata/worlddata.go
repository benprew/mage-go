// Package worlddata contains the static definitions for the MUD world —
// rooms, NPCs, and loot tables. It has no engine dependencies.
//
// To add or modify world content, edit rooms.go and npcs.go. The types
// defined here are the shared vocabulary used by the world and mud packages.
package worlddata

// DeckEntry is a card name paired with a count, used to define NPC decklists
// and player deck saves.
type DeckEntry struct {
	Name  string
	Count int
}

// LootEntry is one possible card drop from an NPC's loot table.
// Weight is a relative integer weight for random selection (higher = more
// likely). A Weight of 0 is treated as 1.
type LootEntry struct {
	Name   string
	Weight int
}

// NPCDef defines a non-player character: their deck, AI personality, loot,
// ante behaviour, dialog, and respawn time.
type NPCDef struct {
	// ID is the unique machine identifier for this NPC (used for boss locks,
	// defeat tracking, and cross-referencing from rooms).
	ID string

	// Name is the display name shown to the player.
	Name string

	// Glyph is the single-character symbol used to represent this NPC on the
	// ASCII world map (e.g. 'M' for Elder Maren, 'B' for Baron Sengir).
	Glyph rune

	// Description is shown when the player examines or approaches the NPC.
	Description string

	// Deck is the NPC's starting library.
	Deck []DeckEntry

	// Personality maps to an interactive.Personality — one of:
	// "aggro", "control", "midrange", "tempo", "burn"
	Personality string

	// AnteForced means the NPC always plays with ante regardless of the
	// player's choice.
	AnteForced bool

	// LootFixed lists card names always awarded to the player on winning.
	LootFixed []string

	// LootTable is the probabilistic reward pool. After a win, LootTableCount
	// cards are drawn from this table (weighted random without replacement).
	LootTable      []LootEntry
	LootTableCount int // default 1 if zero

	// GoldReward is the base gold awarded on winning (before any modifiers).
	GoldReward int

	// Dialog is a list of flavor lines cycled through when the player talks
	// to this NPC.
	Dialog []string

	// RespawnMinutes is how long until this NPC can be challenged again after
	// being defeated. 0 means never respawns (one-time encounter).
	RespawnMinutes int
}

// Treasure is a hidden cache of cards discoverable by searching a room.
type Treasure struct {
	// Cards is the list of card names in this cache.
	Cards []string

	// SearchDC is the difficulty of finding this treasure. 0 = always found
	// when searching; higher values require luck or items (not yet implemented
	// — for now treat any SearchDC > 0 as a 1-in-SearchDC chance per search).
	SearchDC int

	// Trap marks this cache as dangerous. When a trapped chest is opened,
	// the mud layer should trigger an encounter or penalty before awarding
	// the cards (e.g. a forced duel with a summoned creature, or gold loss).
	// If the player fails the encounter, the cards are not awarded.
	Trap bool
}

// RoomDef defines a location in the world.
type RoomDef struct {
	// ID is the unique identifier for this room, used for exit targets and
	// player location tracking.
	ID string

	// Name is the short display title.
	Name string

	// Description is the prose shown when the player looks at the room.
	Description string

	// MapX and MapY are the top-left pixel position of this room's 9×5 box on
	// the ASCII world map canvas (35×41 tiles). Used by the NetHack-style map
	// renderer in internal/mud/mapview.go.
	MapX, MapY int

	// Exits maps direction strings ("north", "south", "east", "west") to room IDs.
	Exits map[string]string

	// NPCIDs lists the IDs of NPCs present in this room at startup.
	NPCIDs []string

	// GoldLock, if > 0, requires the player to pay this many gold to enter.
	GoldLock int

	// BossLock, if non-empty, requires the named NPC to have been defeated
	// before the player can enter this room.
	BossLock string

	// Hidden is the list of treasure caches that can be found by searching.
	Hidden []Treasure

	// StartChest marks this room as containing the new-player starter deck
	// chest. Only one room should have this set.
	StartChest bool
}

// Rooms returns all defined rooms. See rooms.go.
func Rooms() []RoomDef { return allRooms }

// NPCs returns all defined NPCs. See npcs.go.
func NPCs() []NPCDef { return allNPCs }

// NPCByID returns the NPCDef with the given ID, and whether it was found.
func NPCByID(id string) (NPCDef, bool) {
	for _, n := range allNPCs {
		if n.ID == id {
			return n, true
		}
	}
	return NPCDef{}, false
}

// RoomByID returns the RoomDef with the given ID, and whether it was found.
func RoomByID(id string) (RoomDef, bool) {
	for _, r := range allRooms {
		if r.ID == id {
			return r, true
		}
	}
	return RoomDef{}, false
}

// StartRoom returns the ID of the room new players begin in.
func StartRoom() string {
	for _, r := range allRooms {
		if r.StartChest {
			return r.ID
		}
	}
	// fallback: first room
	if len(allRooms) > 0 {
		return allRooms[0].ID
	}
	return ""
}
