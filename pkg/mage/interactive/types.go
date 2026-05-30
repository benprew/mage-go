package interactive

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// AutoPlayer is implemented by AI players to provide priority decisions to the
// game loop. This interface decouples the game loop from the concrete AIPlayer
// type, which lives in the ai/ sub-package.
type AutoPlayer interface {
	GetPriorityAction(g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction
}

// PlayerChannels bundles the communication channels between the game loop and one player's TUI.
type PlayerChannels struct {
	ToPlayer    chan<- GameMsg
	FromPlayer  <-chan PriorityAction
	ChoiceReqs  chan<- ChoiceRequest
	ChoiceResps <-chan ChoiceResponse
}

//go:generate enumer -type=ActionType -trimprefix=Action -output=interactive_enumer.go

// ActionType identifies what kind of action a player is taking.
type ActionType int

const (
	ActionPass ActionType = iota
	ActionPlayLand
	ActionCastSpell
	ActionActivateAbility
	ActionSelectAttackers
	ActionSelectBlockers
	ActionUndo
	ActionAssignCombatDamage
)

// PriorityAction is a decision sent from the TUI (or AI) to the game loop.
type PriorityAction struct {
	Type         ActionType
	CardID       uuid.UUID
	CardName     string
	Targets      []uuid.UUID
	PermanentID  uuid.UUID
	AbilityIndex int
	XValue       int
	Attackers    []uuid.UUID
	Blockers     []mage.BlockAssignment
	DamageOrder  []uuid.UUID
	Damage       map[uuid.UUID]int
}

// PromptType tells the TUI what kind of input is needed.
type PromptType int

const (
	PromptNone PromptType = iota
	PromptMainPhaseAction
	PromptPriority
	PromptDeclareAttackers
	PromptDeclareBlockers
	PromptChooseTargets
	PromptAssignCombatDamage
)

func (pt PromptType) String() string {
	switch pt {
	case PromptMainPhaseAction:
		return "Main Phase"
	case PromptPriority:
		return "Priority"
	case PromptDeclareAttackers:
		return "Declare Attackers"
	case PromptDeclareBlockers:
		return "Declare Blockers"
	case PromptChooseTargets:
		return "Choose Targets"
	case PromptAssignCombatDamage:
		return "Assign Combat Damage"
	default:
		return ""
	}
}

// ActionOption describes one available action in the TUI menu.
type ActionOption struct {
	Type         ActionType
	Label        string
	CardName     string // raw card/permanent name, without cost prefix
	CardID       uuid.UUID
	PermanentID  uuid.UUID
	AbilityIndex int
	NeedsTarget  bool
	TargetType   mage.Target
	ManaCost     string

	ValidTargets      []uuid.UUID
	ValidTargetLabels []string

	NeedsX    bool
	MaxXValue int
}

// GameMsg is sent from the game goroutine to the TUI.
type GameMsg struct {
	State    *GameState
	Prompt   PromptType
	Options  []ActionOption
	Log      []string
	GameOver bool
	Winner   string
	CanUndo  bool
}

type undoSnapshot struct {
	valid          bool
	hand           []mage.Card
	manaPool       []core.Mana
	landsPlayed    int
	battlefieldLen int
	stackSize      int
	tappedState    map[uuid.UUID]bool
	logLen         int
}

// GameState is a read-only snapshot of the game for the TUI.
type GameState struct {
	Turn         int
	Step         string
	ActivePlayer string
	You          PlayerState
	Opponent     PlayerState
	StackItems   []StackItemState
}

// PlayerState is a snapshot of a player.
type PlayerState struct {
	ID             uuid.UUID
	Name           string
	Life           int
	HandCount      int
	Hand           []CardState
	Battlefield    []PermanentState
	Graveyard      []CardState
	GraveyardCount int
	ManaPool       ManaPoolState
	LibraryCount   int
	Exile          []CardState
}

// PermanentState is a snapshot of a permanent.
type PermanentState struct {
	ID          uuid.UUID
	Name        string
	Power       int
	Toughness   int
	Damage      int
	Tapped      bool
	SummonSick  bool
	FaceDown    bool
	PhasedOut   bool
	IsCreature  bool
	IsLand      bool
	IsArtifact  bool
	Attacking   bool
	Blocking    uuid.UUID
	Counters    map[string]int
	RawCounters [core.NumCounters]uint8
	Keywords    []string
	ManaCost    string
	Types       string
	SubTypes    string
	RulesText   string
	AttachedTo  uuid.UUID
}

// CardState is a snapshot of a card in a zone (hand, graveyard, exile).
//
// FaceDown is set for cards in face-down exile (CR 707, 406.3). When the
// viewer producing this snapshot is not permitted to inspect a face-down
// exiled card's identity, Name / ManaCost / Types / SubTypes / Power /
// Toughness / RulesText are zero-valued and only ID + FaceDown are
// populated, so opponents see "a face-down card" without leaking
// identity. The owning player's snapshot is also redacted unless the
// owner was added to the card's RevealedTo set.
type CardState struct {
	ID        uuid.UUID
	Name      string
	ManaCost  string
	IsLand    bool
	Types     string
	SubTypes  string
	Power     int
	Toughness int
	RulesText string
	FaceDown  bool
}

// ManaPoolState is a snapshot of a mana pool.
type ManaPoolState struct {
	White     int
	Blue      int
	Black     int
	Red       int
	Green     int
	Colorless int
}

// StackItemState is a snapshot of a stack object.
type StackItemState struct {
	ID          string
	Name        string
	Controller  string
	IsAbility   bool
	Targets     []string    // human-readable target names
	TargetIDs   []uuid.UUID // permanent/player IDs of targets, parallel to Targets
	XValue      int         // chosen X for X-cost spells, 0 otherwise
	EventAmount int         // amount from triggering event (e.g. damage dealt for triggered abilities)
}

// ChoiceType identifies what kind of interactive choice the human player must make.
type ChoiceType int

const (
	ChoicePermanent       ChoiceType = iota // pick one permanent from candidates
	ChoiceCardsFromHand                     // pick N cards from hand (multi-select)
	ChoiceManaColor                         // pick a mana color
	ChoiceCardFromLibrary                   // pick one card from library candidates
	ChoiceMay                               // yes/no for an optional ability
	ChoiceMode                              // pick one mode from a modal spell/ability
	ChoiceNumber                            // pick a number from a range
)

// ChoiceOption is one selectable item in a ChoiceRequest.
type ChoiceOption struct {
	ID    uuid.UUID // permanent or card ID (zero for color/boolean options)
	Label string
	Color core.Color // populated for ChoiceManaColor options
}

// ChoiceRequest is sent by HumanPlayer to the TUI when a player decision is needed
// during game resolution.
type ChoiceRequest struct {
	Type    ChoiceType
	Reason  string
	Amount  int // for ChoiceCardsFromHand: how many to select
	Options []ChoiceOption
}

// ChoiceResponse is sent by the TUI back to the waiting HumanPlayer.
type ChoiceResponse struct {
	SelectedIDs   []uuid.UUID // for permanent/card choices
	SelectedColor core.Color  // for ChoiceManaColor
	Accepted      bool        // for ChoiceMay: true = yes
	SelectedIndex int         // for ChoiceMode: index of chosen mode
}
