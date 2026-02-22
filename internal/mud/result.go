// Package mud contains the bubbletea model for navigating the MUD world.
package mud

import "github.com/mage/mage/internal/worlddata"

// Result is the value returned by the world navigation program when it quits.
// The SSH handler reads this to decide what to do next.
type Result struct {
	// Action is one of "fight_npc", "deckbuilder", "quit".
	Action string

	// RoomID is the room the player was in when the program quit.
	RoomID string

	// NPC is set when Action == "fight_npc".
	NPC *worlddata.NPCDef

	// AnteEnabled is true if ante cards should be drawn before the game starts.
	AnteEnabled bool
}

// LootResult is passed back into the mud model after a combat session completes
// so the post-combat loot screen can be shown to the player.
type LootResult struct {
	Won        bool
	Cards      []string // card names earned (includes ante card if applicable)
	AnteCard   string   // the name of the player's ante card (blank if no ante)
	GoldEarned int
}
