package main

// JSON types decoding the JSONL emitted by XMage's SelfPlayJsonRecorder
// (Mage.Tests/.../selfplay/SelfPlayJsonRecorder.java). One META line, then
// one EVENT line per recorded engine event.

type metaLine struct {
	Record      string         `json:"record"` // "META"
	GameID      string         `json:"gameId"`
	StartedAt   string         `json:"startedAt"`
	EndedAt     string         `json:"endedAt"`
	WinnerID    string         `json:"winnerId"`
	WinnerName  string         `json:"winnerName"`
	TotalEvents int            `json:"totalEvents"`
	Players     []playerInfo   `json:"players"`
	Extras      map[string]any `json:"extras"`
}

type playerInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	StartingLife int    `json:"startingLife"`
	// Deck = hand cards (first HandSizeAtStart entries) + library top-to-bottom,
	// captured at the first PRIORITY event so replayers reconstruct the
	// post-shuffle library without needing a draw-7 step.
	Deck            []string `json:"deck"`
	HandSizeAtStart int      `json:"handSizeAtStart"`
}

type eventLine struct {
	Record       string            `json:"record"` // "EVENT"
	GameID       string            `json:"gameId"`
	Seq          int               `json:"seq"`
	Type         string            `json:"type"` // PRIORITY, STACK_PUSH, LAND_PLAYED, ATTACKERS_DECLARED, BLOCKERS_DECLARED, DISCARD_TAKEN, TRIGGER_ORDER, STACK_RESOLVE, LOG, GAME_START, GAME_END
	Description  string            `json:"description"`
	Snapshot     *snapshot         `json:"snapshot"`
	Action       *actionData       `json:"action,omitempty"`
	TriggerOrder *triggerOrderData `json:"triggerOrder,omitempty"`
}

type snapshot struct {
	Turn             int              `json:"turn"`
	Phase            string           `json:"phase"`
	Step             string           `json:"step"`
	ActivePlayerID   string           `json:"activePlayerId"`
	PriorityPlayerID string           `json:"priorityPlayerId"`
	Players          []playerSnap     `json:"players"`
	Battlefield      []permanentSnap  `json:"battlefield"`
	Stack            []stackSnap      `json:"stack"`
	PlayableActions  []playableAction `json:"playableActions"`
}

type playerSnap struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Life        int      `json:"life"`
	LibrarySize int      `json:"librarySize"`
	HandSize    int      `json:"handSize"`
	Hand        []string `json:"hand"`
	Graveyard   []string `json:"graveyard"`
	Exile       []string `json:"exile"`
	ManaPool    string   `json:"manaPool"`
	HasLost     bool     `json:"hasLost"`
	HasWon      bool     `json:"hasWon"`
}

type permanentSnap struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	ControllerID      string         `json:"controllerId"`
	Tapped            bool           `json:"tapped"`
	SummoningSickness bool           `json:"summoningSickness"`
	Power             *int           `json:"power"`
	Toughness         *int           `json:"toughness"`
	Damage            int            `json:"damage"`
	AttachedTo        string         `json:"attachedTo"`
	Counters          map[string]int `json:"counters"`
}

type stackSnap struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ControllerID string `json:"controllerId"`
	Kind         string `json:"kind"`
}

type playableAction struct {
	AbilityID    string       `json:"abilityId"`
	ControllerID string       `json:"controllerId"`
	SourceID     string       `json:"sourceId"`
	SourceName   string       `json:"sourceName"`
	Kind         string       `json:"kind"` // PLAY_LAND, CAST_SPELL, ACTIVATE_MANA, ACTIVATE
	Cost         string       `json:"cost"`
	Rule         string       `json:"rule"`
	Description  string       `json:"description"`
	Targets      []targetSlot `json:"targets"`
}

type targetSlot struct {
	Name       string            `json:"name"`
	MinTargets int               `json:"minTargets"`
	MaxTargets int               `json:"maxTargets"`
	Candidates []targetCandidate `json:"candidates"`
}

type targetCandidate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// actionData carries the chosen decision attached to non-PRIORITY events
// (LAND_PLAYED, STACK_PUSH, ATTACKERS_DECLARED, BLOCKERS_DECLARED, DISCARD_TAKEN).
// Field population varies by event type — Gson serializes nulls so all fields
// arrive even when unused; null-decode them as zero values.
type actionData struct {
	Kind          string      `json:"kind"`
	PlayerID      string      `json:"playerId"`
	PlayerName    string      `json:"playerName"`
	SourceID      string      `json:"sourceId"`
	SourceName    string      `json:"sourceName"`
	XValue        *int        `json:"xValue"`
	IsCost        *bool       `json:"isCost"`
	ChosenTargets []targetRef `json:"chosenTargets"`
	Attackers     []targetRef `json:"attackers"`
	Blockers      []blockPair `json:"blockers"`
	Discarded     []targetRef `json:"discarded"`
}

type targetRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slot string `json:"slot"`
}

type blockPair struct {
	BlockerID    string `json:"blockerId"`
	BlockerName  string `json:"blockerName"`
	AttackerID   string `json:"attackerId"`
	AttackerName string `json:"attackerName"`
}

type triggerOrderData struct {
	PlayerID        string             `json:"playerId"`
	PlayerName      string             `json:"playerName"`
	Candidates      []triggerCandidate `json:"candidates"`
	ChosenAbilityID string             `json:"chosenAbilityId"`
}

type triggerCandidate struct {
	AbilityID  string `json:"abilityId"`
	SourceID   string `json:"sourceId"`
	SourceName string `json:"sourceName"`
	Rule       string `json:"rule"`
}
