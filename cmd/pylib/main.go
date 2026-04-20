// Package main exposes the mage engine as a C shared library for use from
// Python (via cffi) or any other language with a C FFI.
//
// Build:
//
//	go build -buildmode=c-shared -o libmage.dylib ./cmd/pylib    # macOS
//	go build -buildmode=c-shared -o libmage.so    ./cmd/pylib    # linux
//
// Every exported call returns a C string of JSON (owned by Go, released via
// MageFreeString) or an int64 handle. Handles are opaque integers keyed into
// a process-global table. Each game runs its decision loop in a goroutine;
// MageStep drives it one choice at a time.
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"

	_ "git.sr.ht/~cdcarter/mage-go/cards" // register all sets
)

// ---------------------------------------------------------------------------
// Handle table
// ---------------------------------------------------------------------------

type pending struct {
	PlayerIdx int
	Msg       *interactive.GameMsg       // priority / attackers / blockers
	Choice    *interactive.ChoiceRequest // in-resolution decision
	Over      bool
	Winner    string
}

type playerChans struct {
	toTUI       chan interactive.GameMsg
	fromTUI     chan interactive.PriorityAction
	choiceReqs  chan interactive.ChoiceRequest
	choiceResps chan interactive.ChoiceResponse
}

type handle struct {
	mu      sync.Mutex
	game    *mage.Game
	players [2]*interactive.HumanPlayer
	chans   [2]playerChans
	current pending
	done    bool
}

var (
	handlesMu sync.Mutex
	handles   = map[int64]*handle{}
	nextID    int64
)

func putHandle(h *handle) int64 {
	id := atomic.AddInt64(&nextID, 1)
	handlesMu.Lock()
	handles[id] = h
	handlesMu.Unlock()
	return id
}

func getHandle(id int64) *handle {
	handlesMu.Lock()
	defer handlesMu.Unlock()
	return handles[id]
}

func dropHandle(id int64) {
	handlesMu.Lock()
	delete(handles, id)
	handlesMu.Unlock()
}

// ---------------------------------------------------------------------------
// JSON envelopes
// ---------------------------------------------------------------------------

type deckSpec struct {
	Name  string     `json:"name"`
	Cards []deckCard `json:"cards"`
}

type deckCard struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type newGameRequest struct {
	PlayerA   deckSpec `json:"player_a"`
	PlayerB   deckSpec `json:"player_b"`
	NameA     string   `json:"name_a"`
	NameB     string   `json:"name_b"`
	Seed      int64    `json:"seed"`
	Shuffle   bool     `json:"shuffle"`
	HandSize  int      `json:"hand_size"`
}

type apiResponse struct {
	OK       bool          `json:"ok"`
	Error    string        `json:"error,omitempty"`
	State    *apiGameState `json:"state,omitempty"`
	Pending  *apiPending   `json:"pending,omitempty"`
	GameOver bool          `json:"game_over"`
	Winner   string        `json:"winner,omitempty"`
}

type apiGameState struct {
	Turn         int                          `json:"turn"`
	Step         string                       `json:"step"`
	ActivePlayer string                       `json:"active_player"`
	Players      [2]interactive.PlayerState   `json:"players"`
	Stack        []interactive.StackItemState `json:"stack"`
}

// apiPending is a unified "what does the engine want next" envelope.
type apiPending struct {
	Kind      string      `json:"kind"`
	PlayerIdx int         `json:"player_idx"`
	Reason    string      `json:"reason,omitempty"`
	Options   []apiOption `json:"options,omitempty"`
	Amount    int         `json:"amount,omitempty"`
}

// apiOption covers both priority ActionOption and ChoiceOption shapes.
type apiOption struct {
	Kind         string      `json:"kind"`
	Label        string      `json:"label"`
	CardID       string      `json:"card_id,omitempty"`
	CardName     string      `json:"card_name,omitempty"`
	PermanentID  string      `json:"permanent_id,omitempty"`
	AbilityIndex int         `json:"ability_index,omitempty"`
	ManaCost     string      `json:"mana_cost,omitempty"`
	ValidTargets []apiTarget `json:"valid_targets,omitempty"`
	ID           string      `json:"id,omitempty"`
	Color        string      `json:"color,omitempty"`
}

type apiTarget struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// actionRequest is what Python sends back to MageStep.
type actionRequest struct {
	Kind         string          `json:"kind"`
	CardID       string          `json:"card_id,omitempty"`
	PermanentID  string          `json:"permanent_id,omitempty"`
	AbilityIndex int             `json:"ability_index,omitempty"`
	Targets      []string        `json:"targets,omitempty"`
	X            int             `json:"x,omitempty"`
	Attackers    []string        `json:"attackers,omitempty"`
	Blockers     []blockerAssign `json:"blockers,omitempty"`
	// choice payloads
	SelectedIDs   []string `json:"selected_ids,omitempty"`
	SelectedIndex int      `json:"selected_index,omitempty"`
	SelectedColor string   `json:"selected_color,omitempty"`
	Accepted      bool     `json:"accepted,omitempty"`
}

type blockerAssign struct {
	Blocker  string `json:"blocker"`
	Attacker string `json:"attacker"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toCStringResponse(r apiResponse) *C.char {
	b, err := json.Marshal(r)
	if err != nil {
		return C.CString(`{"ok":false,"error":"marshal failed"}`)
	}
	return C.CString(string(b))
}

func errResponse(format string, args ...any) *C.char {
	return toCStringResponse(apiResponse{OK: false, Error: fmt.Sprintf(format, args...)})
}

func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}

func parseUUIDs(ss []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		id, err := parseUUID(s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func buildLibrary(spec deckSpec, ownerID uuid.UUID, rng *rand.Rand, shuffle bool) ([]mage.Card, error) {
	var deck []mage.Card
	for _, entry := range spec.Cards {
		for i := 0; i < entry.Count; i++ {
			card, err := mage.CreateCard(entry.Name)
			if err != nil {
				return nil, fmt.Errorf("unknown card %q: %w", entry.Name, err)
			}
			card.SetOwner(ownerID)
			deck = append(deck, card)
		}
	}
	if shuffle && rng != nil {
		rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	}
	return deck, nil
}

func snapshotState(g *mage.Game) *apiGameState {
	s0 := interactive.SnapshotGameState(g, 0)
	s1 := interactive.SnapshotGameState(g, 1)
	return &apiGameState{
		Turn:         g.Turn,
		Step:         g.Step.String(),
		ActivePlayer: g.ActivePlayerObj().Name(),
		Players:      [2]interactive.PlayerState{s0.You, s1.You},
		Stack:        s0.StackItems,
	}
}

func targetsFromActionOption(opt interactive.ActionOption) []apiTarget {
	if !opt.NeedsTarget {
		return nil
	}
	out := make([]apiTarget, 0, len(opt.ValidTargets))
	for i, id := range opt.ValidTargets {
		label := ""
		if i < len(opt.ValidTargetLabels) {
			label = opt.ValidTargetLabels[i]
		}
		out = append(out, apiTarget{ID: id.String(), Label: label})
	}
	return out
}

func convertPriorityOptions(opts []interactive.ActionOption) []apiOption {
	out := make([]apiOption, 0, len(opts))
	for _, o := range opts {
		ao := apiOption{
			Label:        o.Label,
			CardName:     o.CardName,
			AbilityIndex: o.AbilityIndex,
			ManaCost:     o.ManaCost,
			ValidTargets: targetsFromActionOption(o),
		}
		if o.CardID != uuid.Nil {
			ao.CardID = o.CardID.String()
		}
		if o.PermanentID != uuid.Nil {
			ao.PermanentID = o.PermanentID.String()
		}
		switch o.Type {
		case interactive.ActionPass:
			ao.Kind = "pass"
		case interactive.ActionPlayLand:
			ao.Kind = "play_land"
		case interactive.ActionCastSpell:
			ao.Kind = "cast_spell"
		case interactive.ActionActivateAbility:
			ao.Kind = "activate_ability"
		default:
			continue
		}
		out = append(out, ao)
	}
	return out
}

func choicePendingKind(t interactive.ChoiceType) string {
	switch t {
	case interactive.ChoicePermanent:
		return "permanent"
	case interactive.ChoiceCardsFromHand:
		return "cards_from_hand"
	case interactive.ChoiceManaColor:
		return "mana_color"
	case interactive.ChoiceCardFromLibrary:
		return "card_from_library"
	case interactive.ChoiceMay:
		return "may"
	case interactive.ChoiceMode:
		return "mode"
	case interactive.ChoiceNumber:
		return "number"
	}
	return "unknown"
}

func convertChoiceOptions(choiceType interactive.ChoiceType, opts []interactive.ChoiceOption) []apiOption {
	out := make([]apiOption, 0, len(opts))
	for _, o := range opts {
		ao := apiOption{Kind: "choice", Label: o.Label}
		if o.ID != uuid.Nil {
			ao.ID = o.ID.String()
		}
		if choiceType == interactive.ChoiceManaColor {
			ao.Color = o.Color.String()
		}
		out = append(out, ao)
	}
	return out
}

func buildAttackerPending(msg *interactive.GameMsg, playerIdx int) *apiPending {
	out := make([]apiOption, 0, len(msg.Options))
	for _, o := range msg.Options {
		ao := apiOption{Kind: "attacker", Label: o.Label}
		if o.PermanentID != uuid.Nil {
			ao.PermanentID = o.PermanentID.String()
		}
		out = append(out, ao)
	}
	return &apiPending{Kind: "attackers", PlayerIdx: playerIdx, Options: out}
}

func buildBlockerPending(msg *interactive.GameMsg, playerIdx int) *apiPending {
	out := make([]apiOption, 0, len(msg.Options))
	for _, o := range msg.Options {
		ao := apiOption{Kind: "blocker", Label: o.Label}
		if o.PermanentID != uuid.Nil {
			ao.PermanentID = o.PermanentID.String()
		}
		for i, id := range o.ValidTargets {
			label := ""
			if i < len(o.ValidTargetLabels) {
				label = o.ValidTargetLabels[i]
			}
			ao.ValidTargets = append(ao.ValidTargets, apiTarget{ID: id.String(), Label: label})
		}
		out = append(out, ao)
	}
	return &apiPending{Kind: "blockers", PlayerIdx: playerIdx, Options: out}
}

func buildPending(ev pending) *apiPending {
	if ev.Msg != nil {
		msg := ev.Msg
		switch msg.Prompt {
		case interactive.PromptMainPhaseAction, interactive.PromptPriority:
			return &apiPending{
				Kind:      "priority",
				PlayerIdx: ev.PlayerIdx,
				Options:   convertPriorityOptions(msg.Options),
			}
		case interactive.PromptDeclareAttackers:
			return buildAttackerPending(msg, ev.PlayerIdx)
		case interactive.PromptDeclareBlockers:
			return buildBlockerPending(msg, ev.PlayerIdx)
		}
		return nil
	}
	if ev.Choice != nil {
		c := ev.Choice
		return &apiPending{
			Kind:      choicePendingKind(c.Type),
			PlayerIdx: ev.PlayerIdx,
			Reason:    c.Reason,
			Amount:    c.Amount,
			Options:   convertChoiceOptions(c.Type, c.Options),
		}
	}
	return nil
}

// waitForNext pulls the next event from any of the four channels. Returns
// Over=true when the game goroutine has signaled game over or all channels
// are closed.
func waitForNext(h *handle) pending {
	t0, t1 := h.chans[0].toTUI, h.chans[1].toTUI
	c0, c1 := h.chans[0].choiceReqs, h.chans[1].choiceReqs

	for {
		if t0 == nil && t1 == nil && c0 == nil && c1 == nil {
			winner := ""
			if h.game != nil {
				winner = h.game.Winner()
			}
			return pending{Over: true, Winner: winner}
		}
		select {
		case msg, ok := <-t0:
			if !ok {
				t0 = nil
				continue
			}
			if msg.GameOver {
				return pending{Over: true, Winner: msg.Winner}
			}
			if msg.Prompt == interactive.PromptNone {
				continue
			}
			return pending{PlayerIdx: 0, Msg: &msg}
		case msg, ok := <-t1:
			if !ok {
				t1 = nil
				continue
			}
			if msg.GameOver {
				return pending{Over: true, Winner: msg.Winner}
			}
			if msg.Prompt == interactive.PromptNone {
				continue
			}
			return pending{PlayerIdx: 1, Msg: &msg}
		case req, ok := <-c0:
			if !ok {
				c0 = nil
				continue
			}
			return pending{PlayerIdx: 0, Choice: &req}
		case req, ok := <-c1:
			if !ok {
				c1 = nil
				continue
			}
			return pending{PlayerIdx: 1, Choice: &req}
		}
	}
}

// routeAction converts a Python action into a send on the correct channel
// for the current pending request. Assumes h.mu is held.
func routeAction(h *handle, req actionRequest) error {
	ev := h.current
	pc := h.chans[ev.PlayerIdx]

	if ev.Msg != nil {
		msg := ev.Msg
		switch msg.Prompt {
		case interactive.PromptMainPhaseAction, interactive.PromptPriority:
			if err := validatePriorityAction(req, msg.Options); err != nil {
				return err
			}
			pa, err := buildPriorityAction(req)
			if err != nil {
				return err
			}
			pc.fromTUI <- pa
			return nil
		case interactive.PromptDeclareAttackers:
			ids, err := parseUUIDs(req.Attackers)
			if err != nil {
				return fmt.Errorf("parse attackers: %w", err)
			}
			pc.fromTUI <- interactive.PriorityAction{
				Type:      interactive.ActionSelectAttackers,
				Attackers: ids,
			}
			return nil
		case interactive.PromptDeclareBlockers:
			assigns := make([]mage.BlockAssignment, 0, len(req.Blockers))
			for _, b := range req.Blockers {
				bid, err := parseUUID(b.Blocker)
				if err != nil {
					return fmt.Errorf("parse blocker id: %w", err)
				}
				aid, err := parseUUID(b.Attacker)
				if err != nil {
					return fmt.Errorf("parse attacker id: %w", err)
				}
				assigns = append(assigns, mage.BlockAssignment{BlockerID: bid, AttackerID: aid})
			}
			pc.fromTUI <- interactive.PriorityAction{
				Type:     interactive.ActionSelectBlockers,
				Blockers: assigns,
			}
			return nil
		}
		return fmt.Errorf("unsupported prompt: %v", msg.Prompt)
	}

	if ev.Choice != nil {
		resp := interactive.ChoiceResponse{
			SelectedIndex: req.SelectedIndex,
			Accepted:      req.Accepted,
		}
		ids, err := parseUUIDs(req.SelectedIDs)
		if err != nil {
			return fmt.Errorf("parse selected ids: %w", err)
		}
		resp.SelectedIDs = ids
		if req.SelectedColor != "" {
			resp.SelectedColor = parseColor(req.SelectedColor)
		}
		pc.choiceResps <- resp
		return nil
	}

	return fmt.Errorf("no pending request to respond to")
}

func parseColor(s string) core.Color {
	switch s {
	case "white", "W":
		return core.White
	case "blue", "U":
		return core.Blue
	case "black", "B":
		return core.Black
	case "red", "R":
		return core.Red
	case "green", "G":
		return core.Green
	case "colorless", "C":
		return core.Colorless
	}
	return core.Colorless
}

// validatePriorityAction rejects actions that don't match any currently-
// legal option. The engine itself silently no-ops on illegal priority
// actions; the shim converts that into a clear error for Python callers.
func validatePriorityAction(req actionRequest, options []interactive.ActionOption) error {
	switch req.Kind {
	case "pass":
		for _, o := range options {
			if o.Type == interactive.ActionPass {
				return nil
			}
		}
		return fmt.Errorf("illegal action: pass not available")

	case "play_land":
		cardID, err := parseUUID(req.CardID)
		if err != nil {
			return fmt.Errorf("parse card_id: %w", err)
		}
		for _, o := range options {
			if o.Type == interactive.ActionPlayLand && o.CardID == cardID {
				return nil
			}
		}
		return fmt.Errorf("illegal action: play_land %s not available (already played a land this turn, wrong phase, or card not in hand)", req.CardID)

	case "cast_spell":
		cardID, err := parseUUID(req.CardID)
		if err != nil {
			return fmt.Errorf("parse card_id: %w", err)
		}
		for _, o := range options {
			if o.Type != interactive.ActionCastSpell || o.CardID != cardID {
				continue
			}
			return validateTargets(req.Targets, o)
		}
		return fmt.Errorf("illegal action: cast_spell %s not available", req.CardID)

	case "activate_ability":
		permID, err := parseUUID(req.PermanentID)
		if err != nil {
			return fmt.Errorf("parse permanent_id: %w", err)
		}
		for _, o := range options {
			if o.Type != interactive.ActionActivateAbility {
				continue
			}
			if o.PermanentID != permID || o.AbilityIndex != req.AbilityIndex {
				continue
			}
			return validateTargets(req.Targets, o)
		}
		return fmt.Errorf("illegal action: activate_ability perm=%s idx=%d not available", req.PermanentID, req.AbilityIndex)
	}
	return fmt.Errorf("unknown action kind: %s", req.Kind)
}

func validateTargets(targets []string, opt interactive.ActionOption) error {
	if !opt.NeedsTarget {
		if len(targets) > 0 {
			return fmt.Errorf("action takes no targets but %d were given", len(targets))
		}
		return nil
	}
	if len(targets) == 0 {
		return fmt.Errorf("action requires a target (options: %v)", opt.ValidTargetLabels)
	}
	valid := make(map[uuid.UUID]bool, len(opt.ValidTargets))
	for _, id := range opt.ValidTargets {
		valid[id] = true
	}
	for _, t := range targets {
		id, err := parseUUID(t)
		if err != nil {
			return fmt.Errorf("parse target: %w", err)
		}
		if !valid[id] {
			return fmt.Errorf("illegal target %s (valid: %v)", t, opt.ValidTargetLabels)
		}
	}
	return nil
}

func buildPriorityAction(req actionRequest) (interactive.PriorityAction, error) {
	targets, err := parseUUIDs(req.Targets)
	if err != nil {
		return interactive.PriorityAction{}, fmt.Errorf("parse targets: %w", err)
	}
	switch req.Kind {
	case "pass":
		return interactive.PriorityAction{Type: interactive.ActionPass}, nil
	case "play_land":
		id, err := parseUUID(req.CardID)
		if err != nil {
			return interactive.PriorityAction{}, err
		}
		return interactive.PriorityAction{Type: interactive.ActionPlayLand, CardID: id}, nil
	case "cast_spell":
		id, err := parseUUID(req.CardID)
		if err != nil {
			return interactive.PriorityAction{}, err
		}
		return interactive.PriorityAction{
			Type:    interactive.ActionCastSpell,
			CardID:  id,
			Targets: targets,
			XValue:  req.X,
		}, nil
	case "activate_ability":
		id, err := parseUUID(req.PermanentID)
		if err != nil {
			return interactive.PriorityAction{}, err
		}
		return interactive.PriorityAction{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  id,
			AbilityIndex: req.AbilityIndex,
			Targets:      targets,
			XValue:       req.X,
		}, nil
	}
	return interactive.PriorityAction{}, fmt.Errorf("unknown action kind: %s", req.Kind)
}

func newPlayerChans() playerChans {
	return playerChans{
		toTUI:       make(chan interactive.GameMsg, 1),
		fromTUI:     make(chan interactive.PriorityAction, 1),
		choiceReqs:  make(chan interactive.ChoiceRequest, 1),
		choiceResps: make(chan interactive.ChoiceResponse, 1),
	}
}

// safeClose closes c if it isn't already closed.
func safeClose[T any](c chan T) {
	defer func() { _ = recover() }()
	close(c)
}

// ---------------------------------------------------------------------------
// Exports
// ---------------------------------------------------------------------------

//export MageNewGame
func MageNewGame(cfgJSON *C.char) (id C.int64_t, resp *C.char) {
	defer func() {
		if r := recover(); r != nil {
			id = -1
			resp = errResponse("panic: %v", r)
		}
	}()

	var req newGameRequest
	if err := json.Unmarshal([]byte(C.GoString(cfgJSON)), &req); err != nil {
		return -1, errResponse("parse config: %v", err)
	}
	if req.HandSize == 0 {
		req.HandSize = 7
	}

	nameA := req.NameA
	if nameA == "" {
		nameA = "Player A"
	}
	nameB := req.NameB
	if nameB == "" {
		nameB = "Player B"
	}
	if nameA == nameB {
		nameA, nameB = nameA+" (A)", nameB+" (B)"
	}

	chansA := newPlayerChans()
	chansB := newPlayerChans()
	pA := interactive.NewHumanPlayerWithChannels(nameA, chansA.toTUI, chansA.fromTUI, chansA.choiceReqs, chansA.choiceResps)
	pB := interactive.NewHumanPlayerWithChannels(nameB, chansB.toTUI, chansB.fromTUI, chansB.choiceReqs, chansB.choiceResps)

	var rng *rand.Rand
	if req.Seed != 0 {
		rng = rand.New(rand.NewSource(req.Seed))
	} else {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	libA, err := buildLibrary(req.PlayerA, pA.PlayerID(), rng, req.Shuffle)
	if err != nil {
		return -1, errResponse("deck A: %v", err)
	}
	libB, err := buildLibrary(req.PlayerB, pB.PlayerID(), rng, req.Shuffle)
	if err != nil {
		return -1, errResponse("deck B: %v", err)
	}
	for _, c := range libA {
		pA.AddToLibrary(c)
	}
	for _, c := range libB {
		pB.AddToLibrary(c)
	}

	g := mage.NewGame(pA, pB)
	for i := 0; i < req.HandSize; i++ {
		pA.DrawCard()
		pB.DrawCard()
	}

	h := &handle{
		game:    g,
		players: [2]*interactive.HumanPlayer{pA, pB},
		chans:   [2]playerChans{chansA, chansB},
	}
	hid := putHandle(h)

	channels := [2]interactive.PlayerChannels{
		{
			ToPlayer:    chansA.toTUI,
			FromPlayer:  chansA.fromTUI,
			ChoiceReqs:  chansA.choiceReqs,
			ChoiceResps: chansA.choiceResps,
		},
		{
			ToPlayer:    chansB.toTUI,
			FromPlayer:  chansB.fromTUI,
			ChoiceReqs:  chansB.choiceReqs,
			ChoiceResps: chansB.choiceResps,
		},
	}

	go func() {
		defer func() { _ = recover() }()
		interactive.RunMultiplayerGameLoop(g, channels)
	}()

	ev := waitForNext(h)
	h.mu.Lock()
	h.current = ev
	h.done = ev.Over
	h.mu.Unlock()

	out := apiResponse{
		OK:       true,
		State:    snapshotState(g),
		Pending:  buildPending(ev),
		GameOver: ev.Over,
		Winner:   ev.Winner,
	}
	return C.int64_t(hid), toCStringResponse(out)
}

//export MageState
func MageState(id C.int64_t) *C.char {
	defer func() { _ = recover() }()
	h := getHandle(int64(id))
	if h == nil {
		return errResponse("unknown handle %d", int64(id))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	out := apiResponse{
		OK:       true,
		State:    snapshotState(h.game),
		Pending:  buildPending(h.current),
		GameOver: h.done,
		Winner:   h.current.Winner,
	}
	return toCStringResponse(out)
}

//export MageLegal
func MageLegal(id C.int64_t) *C.char {
	defer func() { _ = recover() }()
	h := getHandle(int64(id))
	if h == nil {
		return errResponse("unknown handle %d", int64(id))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	p := buildPending(h.current)
	b, _ := json.Marshal(p)
	return C.CString(string(b))
}

//export MageStep
func MageStep(id C.int64_t, actionJSON *C.char) (resp *C.char) {
	defer func() {
		if r := recover(); r != nil {
			resp = errResponse("panic: %v", r)
		}
	}()
	h := getHandle(int64(id))
	if h == nil {
		return errResponse("unknown handle %d", int64(id))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return errResponse("game is over")
	}

	var req actionRequest
	if err := json.Unmarshal([]byte(C.GoString(actionJSON)), &req); err != nil {
		return errResponse("parse action: %v", err)
	}

	if err := routeAction(h, req); err != nil {
		return errResponse("%v", err)
	}

	ev := waitForNext(h)
	h.current = ev
	h.done = ev.Over

	out := apiResponse{
		OK:       true,
		State:    snapshotState(h.game),
		Pending:  buildPending(ev),
		GameOver: ev.Over,
		Winner:   ev.Winner,
	}
	return toCStringResponse(out)
}

//export MageFree
func MageFree(id C.int64_t) {
	defer func() { _ = recover() }()
	h := getHandle(int64(id))
	if h == nil {
		return
	}
	dropHandle(int64(id))
	// Unblock the engine goroutine by closing its input channels. It will
	// see fromTUI close → disconnect → exit, or return through the
	// choice-response path.
	for i := 0; i < 2; i++ {
		safeClose(h.chans[i].fromTUI)
		safeClose(h.chans[i].choiceResps)
	}
}

//export MageFreeString
func MageFreeString(s *C.char) {
	if s != nil {
		C.free(unsafe.Pointer(s))
	}
}

//export MageRegisteredCards
func MageRegisteredCards() *C.char {
	defer func() { _ = recover() }()
	names := mage.RegisteredCardNames()
	b, _ := json.Marshal(names)
	return C.CString(string(b))
}

func main() {} // required for c-shared
