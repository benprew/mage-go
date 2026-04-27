package main

// JSON protocol types shared between Go driver and Java CrossValOracle.
// All identification is by card name (no UUIDs cross the wire).
//
// Flow: XMage drives the game (AI makes all decisions). After each action,
// XMage sends the action + resulting state. Go mirrors the action on its own
// Game instance, compares states, and sends an ack.

// --- Go -> Java ---

type setupMsg struct {
	Type     string    `json:"type"` // "setup"
	PlayerA  playerDef `json:"player_a"`
	PlayerB  playerDef `json:"player_b"`
	HandSize int       `json:"hand_size"`
	MaxTurns int       `json:"max_turns"`
}

type playerDef struct {
	Name    string   `json:"name"`
	Library []string `json:"library"` // ordered card names, top of library first
}

type ackMsg struct {
	Type string `json:"type"` // "ack"
}

// --- Java -> Go ---

// oracleMsg is the envelope for all messages from the Java oracle.
// The Type field determines which fields are populated.
type oracleMsg struct {
	Type string `json:"type"` // "step_begin", "action_taken", "attackers_declared", "blockers_declared", "game_over", "error", "ready"

	// action_taken / step_begin fields
	Turn      int         `json:"turn,omitempty"`
	Step      string      `json:"step,omitempty"`
	PlayerIdx int         `json:"player_idx,omitempty"`
	Action    *actionInfo `json:"action,omitempty"`
	State     *cvState    `json:"state,omitempty"`

	// step_begin fields
	ActivePlayerIdx int `json:"active_player_idx,omitempty"`

	// attackers_declared fields
	Attackers []string `json:"attackers,omitempty"`

	// blockers_declared fields
	Blockers []blockerPair `json:"blockers,omitempty"`

	// game_over fields
	Winner string `json:"winner,omitempty"`

	// error fields
	Message string `json:"message,omitempty"`

	// ready fields (card intersection response)
	Cards []string `json:"cards,omitempty"`
}

// actionInfo describes what the XMage AI decided to do at a priority point.
type actionInfo struct {
	Kind          string   `json:"kind"` // pass, play_land, cast_spell, activate_ability
	CardName      string   `json:"card_name,omitempty"`
	Targets       []string `json:"targets,omitempty"`
	PermanentName string   `json:"permanent_name,omitempty"`
	AbilityIndex  int      `json:"ability_index,omitempty"`
}

type blockerPair struct {
	Blocker  string `json:"blocker"`
	Attacker string `json:"attacker"`
}

// cvState is the canonical game state for comparison.
type cvState struct {
	Turn            int         `json:"turn"`
	Step            string      `json:"step"`
	ActivePlayerIdx int         `json:"active_player_idx"`
	Players         [2]cvPlayer `json:"players"`
	Stack           []string    `json:"stack"`
}

type cvPlayer struct {
	Name        string        `json:"name"`
	Life        int           `json:"life"`
	Hand        []string      `json:"hand"`        // sorted card names
	Battlefield []cvPermanent `json:"battlefield"` // sorted by name
	Graveyard   []string      `json:"graveyard"`   // sorted card names
	LibrarySize int           `json:"library_size"`
}

type cvPermanent struct {
	Name       string `json:"name"`
	Tapped     bool   `json:"tapped"`
	Power      int    `json:"power"`
	Toughness  int    `json:"toughness"`
	SummonSick bool   `json:"summoning_sick"`
}
