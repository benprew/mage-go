//go:build js && wasm

// Command wasm builds the mage engine as a WebAssembly module, exposing
// game-loop functions to JavaScript via syscall/js.
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"syscall/js"
	"time"

	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"

	// Register all card sets.
	_ "git.sr.ht/~cdcarter/mage-go/cards"
)

func main() {
	js.Global().Set("mageGetCardList", js.FuncOf(getCardList))
	js.Global().Set("mageStartGame", js.FuncOf(startGame))
	js.Global().Set("mageSendAction", js.FuncOf(sendAction))
	js.Global().Set("mageSendChoice", js.FuncOf(sendChoice))

	// Keep the Go runtime alive.
	select {}
}

// activeGame holds the state for the currently running game.
var activeGame struct {
	fromTUI     chan interactive.PriorityAction
	choiceResps chan interactive.ChoiceResponse
}

// getCardList returns a JSON array of all registered card names.
func getCardList(this js.Value, args []js.Value) any {
	names := mage.RegisteredCardNames()
	sort.Strings(names)
	b, _ := json.Marshal(names)
	return string(b)
}

// wasmPersonalities maps lowercase personality names to WeightedPersonality values.
var wasmPersonalities = map[string]ai.WeightedPersonality{
	"aggro":    ai.AggroWeighted,
	"control":  ai.ControlWeighted,
	"midrange": ai.MidrangeWeighted,
	"tempo":    ai.TempoWeighted,
	"burn":     ai.BurnWeighted,
}

// createWasmAI builds an AI player from personality and mode strings.
func createWasmAI(name, personality, mode string) *ai.AIPlayer {
	wp := ai.MidrangeWeighted
	if w, ok := wasmPersonalities[personality]; ok {
		wp = w
	}
	switch mode {
	case "search":
		return ai.NewSearchAI(name, ai.DefaultSearchConfig(), wp)
	case "adaptive":
		return ai.NewAdaptiveAI(name)
	default:
		return ai.NewWeightedAI(name, wp)
	}
}

// startGame creates a new game with a human player vs AI and starts the game loop.
// Arguments: deckJSON (string), onGameMsg (JS callback), onChoiceReq (JS callback),
//
//	optionally: aiDeckJSON (string), aiPersonality (string), aiMode (string)
func startGame(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return "error: need deckJSON, onGameMsg, onChoiceReq"
	}

	deckJSON := args[0].String()
	onGameMsg := args[1]
	onChoiceReq := args[2]

	// Optional: AI deck, personality, and mode.
	var aiDeckJSON string
	var aiPersonality string
	var aiMode string
	if len(args) > 3 && args[3].Type() == js.TypeString {
		aiDeckJSON = args[3].String()
	}
	if len(args) > 4 && args[4].Type() == js.TypeString {
		aiPersonality = args[4].String()
	}
	if len(args) > 5 && args[5].Type() == js.TypeString {
		aiMode = args[5].String()
	}

	// Parse deck list: array of card names (one entry per copy).
	var deckNames []string
	if err := json.Unmarshal([]byte(deckJSON), &deckNames); err != nil {
		return fmt.Sprintf("error: invalid deck JSON: %v", err)
	}

	if len(deckNames) < 7 {
		return "error: deck must have at least 7 cards"
	}

	// Create players.
	toTUI := make(chan interactive.GameMsg, 1)
	fromTUI := make(chan interactive.PriorityAction, 1)
	choiceReqs := make(chan interactive.ChoiceRequest, 1)
	choiceResps := make(chan interactive.ChoiceResponse, 1)

	human := interactive.NewHumanPlayerWithChannels("You", toTUI, fromTUI, choiceReqs, choiceResps)

	// Create AI with selected personality and mode.
	aiPlayer := createWasmAI("AI", aiPersonality, aiMode)

	// Build human deck.
	humanDeck := buildDeck(deckNames, human.PlayerID())
	if humanDeck == nil {
		return "error: deck contains unknown cards"
	}

	// Build AI deck from provided list or default RB Burn.
	var aiDeckNames []string
	if aiDeckJSON != "" {
		if err := json.Unmarshal([]byte(aiDeckJSON), &aiDeckNames); err != nil {
			return fmt.Sprintf("error: invalid AI deck JSON: %v", err)
		}
	}
	if len(aiDeckNames) < 7 {
		aiDeckEntries := []struct {
			Name  string
			Count int
		}{
			{"Mountain", 8},
			{"Swamp", 8},
			{"Lightning Bolt", 4},
			{"Terror", 4},
			{"Black Knight", 4},
			{"Ironclaw Orcs", 4},
			{"Hill Giant", 4},
			{"Hypnotic Specter", 4},
		}
		aiDeckNames = nil
		for _, e := range aiDeckEntries {
			for i := 0; i < e.Count; i++ {
				aiDeckNames = append(aiDeckNames, e.Name)
			}
		}
	}
	aiDeck := buildDeck(aiDeckNames, aiPlayer.PlayerID())

	// Load libraries.
	for _, c := range humanDeck {
		human.AddToLibrary(c)
	}
	for _, c := range aiDeck {
		aiPlayer.AddToLibrary(c)
	}

	// Create game.
	g := mage.NewGame(human, aiPlayer)

	// Draw opening hands.
	for i := 0; i < 7; i++ {
		human.DrawCard()
	}
	for i := 0; i < 7; i++ {
		aiPlayer.DrawCard()
	}
	ai.MulliganAI(aiPlayer)

	// Store channels for sendAction/sendChoice.
	activeGame.fromTUI = fromTUI
	activeGame.choiceResps = choiceResps

	// Pump GameMsg to JS callback.
	go func() {
		for msg := range toTUI {
			b, _ := json.Marshal(gameMsgToJSON(msg))
			onGameMsg.Invoke(string(b))
		}
	}()

	// Pump ChoiceRequest to JS callback.
	go func() {
		for req := range choiceReqs {
			b, _ := json.Marshal(choiceReqToJSON(req))
			onChoiceReq.Invoke(string(b))
		}
	}()

	// Start game loop.
	go interactive.RunGameLoop(g, 0, 0)

	return nil
}

// sendAction sends a PriorityAction from JS to the game loop.
// Argument: actionJSON (string)
func sendAction(this js.Value, args []js.Value) any {
	if len(args) < 1 || activeGame.fromTUI == nil {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(args[0].String()), &raw); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	action := interactive.PriorityAction{}

	if v, ok := raw["type"]; ok {
		var t int
		json.Unmarshal(v, &t)
		action.Type = interactive.ActionType(t)
	}

	if v, ok := raw["cardID"]; ok {
		var s string
		json.Unmarshal(v, &s)
		action.CardID, _ = uuid.Parse(s)
	}

	if v, ok := raw["cardName"]; ok {
		json.Unmarshal(v, &action.CardName)
	}

	if v, ok := raw["permanentID"]; ok {
		var s string
		json.Unmarshal(v, &s)
		action.PermanentID, _ = uuid.Parse(s)
	}

	if v, ok := raw["abilityIndex"]; ok {
		json.Unmarshal(v, &action.AbilityIndex)
	}

	if v, ok := raw["xValue"]; ok {
		json.Unmarshal(v, &action.XValue)
	}

	if v, ok := raw["targets"]; ok {
		var strs []string
		json.Unmarshal(v, &strs)
		for _, s := range strs {
			id, _ := uuid.Parse(s)
			action.Targets = append(action.Targets, id)
		}
	}

	if v, ok := raw["attackers"]; ok {
		var strs []string
		json.Unmarshal(v, &strs)
		for _, s := range strs {
			id, _ := uuid.Parse(s)
			action.Attackers = append(action.Attackers, id)
		}
	}

	if v, ok := raw["blockers"]; ok {
		var blockers []struct {
			BlockerID  string `json:"blockerID"`
			AttackerID string `json:"attackerID"`
		}
		json.Unmarshal(v, &blockers)
		for _, b := range blockers {
			bid, _ := uuid.Parse(b.BlockerID)
			aid, _ := uuid.Parse(b.AttackerID)
			action.Blockers = append(action.Blockers, mage.BlockAssignment{
				BlockerID:  bid,
				AttackerID: aid,
			})
		}
	}

	activeGame.fromTUI <- action
	return nil
}

// sendChoice sends a ChoiceResponse from JS to the game loop.
// Argument: choiceJSON (string)
func sendChoice(this js.Value, args []js.Value) any {
	if len(args) < 1 || activeGame.choiceResps == nil {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(args[0].String()), &raw); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	resp := interactive.ChoiceResponse{}

	if v, ok := raw["selectedIDs"]; ok {
		var strs []string
		json.Unmarshal(v, &strs)
		for _, s := range strs {
			id, _ := uuid.Parse(s)
			resp.SelectedIDs = append(resp.SelectedIDs, id)
		}
	}

	if v, ok := raw["selectedColor"]; ok {
		var c int
		json.Unmarshal(v, &c)
		resp.SelectedColor = core.Color(c)
	}

	if v, ok := raw["accepted"]; ok {
		json.Unmarshal(v, &resp.Accepted)
	}

	if v, ok := raw["selectedIndex"]; ok {
		json.Unmarshal(v, &resp.SelectedIndex)
	}

	activeGame.choiceResps <- resp
	return nil
}

// buildDeck creates a shuffled deck of cards from a list of card names.
func buildDeck(names []string, ownerID uuid.UUID) []mage.Card {
	var deck []mage.Card
	for _, name := range names {
		card, err := mage.CreateCard(name)
		if err != nil {
			fmt.Printf("warning: unknown card %q, skipping\n", name)
			continue
		}
		card.SetOwner(ownerID)
		deck = append(deck, card)
	}
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

// JSON serialization helpers — these convert the interactive types to plain
// maps so they JSON-serialize cleanly with string UUIDs and readable field names.

type gameMsgJSON struct {
	State    *interactive.GameState `json:"state"`
	Prompt   int                   `json:"prompt"`
	Options  []actionOptionJSON    `json:"options"`
	Log      []string              `json:"log"`
	GameOver bool                  `json:"gameOver"`
	Winner   string                `json:"winner"`
	CanUndo  bool                  `json:"canUndo"`
}

type actionOptionJSON struct {
	Type         int    `json:"type"`
	Label        string `json:"label"`
	CardName     string `json:"cardName"`
	CardID       string `json:"cardID"`
	PermanentID  string `json:"permanentID"`
	AbilityIndex int    `json:"abilityIndex"`
	NeedsTarget  bool   `json:"needsTarget"`
	ManaCost     string `json:"manaCost"`
	NeedsX       bool   `json:"needsX,omitempty"`
	MaxXValue    int    `json:"maxXValue,omitempty"`
}

func gameMsgToJSON(msg interactive.GameMsg) gameMsgJSON {
	opts := make([]actionOptionJSON, len(msg.Options))
	for i, o := range msg.Options {
		opts[i] = actionOptionJSON{
			Type:         int(o.Type),
			Label:        o.Label,
			CardName:     o.CardName,
			CardID:       o.CardID.String(),
			PermanentID:  o.PermanentID.String(),
			AbilityIndex: o.AbilityIndex,
			NeedsTarget:  o.NeedsTarget,
			ManaCost:     o.ManaCost,
			NeedsX:       o.NeedsX,
			MaxXValue:    o.MaxXValue,
		}
	}
	return gameMsgJSON{
		State:    msg.State,
		Prompt:   int(msg.Prompt),
		Options:  opts,
		Log:      msg.Log,
		GameOver: msg.GameOver,
		Winner:   msg.Winner,
		CanUndo:  msg.CanUndo,
	}
}

type choiceReqJSON struct {
	Type    int                `json:"type"`
	Reason  string             `json:"reason"`
	Amount  int                `json:"amount"`
	Options []choiceOptionJSON `json:"options"`
}

type choiceOptionJSON struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Color int    `json:"color"`
}

func choiceReqToJSON(req interactive.ChoiceRequest) choiceReqJSON {
	opts := make([]choiceOptionJSON, len(req.Options))
	for i, o := range req.Options {
		opts[i] = choiceOptionJSON{
			ID:    o.ID.String(),
			Label: o.Label,
			Color: int(o.Color),
		}
	}
	return choiceReqJSON{
		Type:    int(req.Type),
		Reason:  req.Reason,
		Amount:  req.Amount,
		Options: opts,
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
