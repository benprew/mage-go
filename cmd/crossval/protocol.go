package main

// JSON protocol types shared between Go driver and Java CrossValOracle.
// All identification is by card name (no UUIDs cross the wire).

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

type actionMsg struct {
	Type     string   `json:"type"` // "action"
	Kind     string   `json:"kind"` // pass, play_land, cast_spell, activate_ability, attack, block, choice
	CardName string   `json:"card_name,omitempty"`
	Targets  []string `json:"targets,omitempty"`
	XValue   int      `json:"x_value,omitempty"`
	// For activate_ability: which permanent and which ability index
	PermanentName string `json:"permanent_name,omitempty"`
	AbilityIndex  int    `json:"ability_index,omitempty"`
	// For attackers: list of creature names to attack with
	Attackers []string `json:"attackers,omitempty"`
	// For blockers: blocker->attacker pairs
	Blockers []blockerPair `json:"blockers,omitempty"`
	// For choices
	SelectedIndex int    `json:"selected_index,omitempty"`
	SelectedName  string `json:"selected_name,omitempty"`
	Accepted      bool   `json:"accepted,omitempty"`
}

type blockerPair struct {
	Blocker  string `json:"blocker"`
	Attacker string `json:"attacker"`
}

// --- Java -> Go ---

type oracleMsg struct {
	Type string `json:"type"` // "decision_point", "game_over", "error", "ready"

	// decision_point fields
	State  *cvState      `json:"state,omitempty"`
	Legal  []legalAction `json:"legal,omitempty"`
	Kind   string        `json:"kind,omitempty"` // priority, attackers, blockers, choice
	Choice *choiceInfo   `json:"choice,omitempty"`

	// game_over fields
	WinnerIdx int    `json:"winner_idx,omitempty"`
	Winner    string `json:"winner,omitempty"`

	// error fields
	Message string `json:"message,omitempty"`

	// ready fields (card intersection response)
	Cards []string `json:"cards,omitempty"`
}

// cvState is the canonical game state for comparison.
type cvState struct {
	Turn            int             `json:"turn"`
	Step            string          `json:"step"`
	ActivePlayerIdx int             `json:"active_player_idx"`
	Players         [2]cvPlayer     `json:"players"`
	Stack           []string        `json:"stack"`
}

type cvPlayer struct {
	Name        string        `json:"name"`
	Life        int           `json:"life"`
	Hand        []string      `json:"hand"`        // sorted card names
	Battlefield []cvPermanent `json:"battlefield"`  // sorted by name
	Graveyard   []string      `json:"graveyard"`    // sorted card names
	LibrarySize int           `json:"library_size"`
}

type cvPermanent struct {
	Name         string `json:"name"`
	Tapped       bool   `json:"tapped"`
	Power        int    `json:"power"`
	Toughness    int    `json:"toughness"`
	SummonSick   bool   `json:"summoning_sick"`
}

type legalAction struct {
	Kind          string   `json:"kind"` // pass, play_land, cast_spell, activate_ability
	CardName      string   `json:"card_name,omitempty"`
	PermanentName string   `json:"permanent_name,omitempty"`
	AbilityIndex  int      `json:"ability_index,omitempty"`
	Targets       []string `json:"targets,omitempty"` // valid target names
	HasX          bool     `json:"has_x,omitempty"`
	MaxX          int      `json:"max_x,omitempty"` // max payable X value
}

type choiceOption struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

type choiceInfo struct {
	Kind    string         `json:"kind"` // permanent, may, mode, mana_color, number
	Reason  string         `json:"reason,omitempty"`
	Options []choiceOption `json:"options,omitempty"`
}
