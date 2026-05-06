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
#include "abi.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
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
	mu       sync.Mutex
	game     *mage.Game
	players  [2]*interactive.HumanPlayer
	chans    [2]playerChans
	current  pending
	done     bool
	stateBuf *apiGameState // cached snapshotState; cleared on each Step
}

// cachedSnapshotState returns the cached game-state snapshot for the
// handle, lazily populating it. The handle's mu must be held by the
// caller. The cache is invalidated after each MageStep advance.
func cachedSnapshotState(h *handle) *apiGameState {
	if h.stateBuf == nil {
		h.stateBuf = snapshotState(h.game)
	}
	return h.stateBuf
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
	PlayerA  deckSpec `json:"player_a"`
	PlayerB  deckSpec `json:"player_b"`
	NameA    string   `json:"name_a"`
	NameB    string   `json:"name_b"`
	Seed     int64    `json:"seed"`
	Shuffle  bool     `json:"shuffle"`
	HandSize int      `json:"hand_size"`
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
	// CardUUID / PermanentUUID / IDUUID mirror the string IDs above as
	// raw uuid.UUID values. Populated alongside the strings during
	// snapshot conversion so the encode hot path can avoid a round-trip
	// through uuid.Parse on every option lookup. Excluded from JSON to
	// keep the wire format unchanged.
	CardUUID      uuid.UUID `json:"-"`
	PermanentUUID uuid.UUID `json:"-"`
	IDUUID        uuid.UUID `json:"-"`
}

type apiTarget struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// IDUUID mirrors ID for the encode hot path. See apiOption above.
	IDUUID uuid.UUID `json:"-"`
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

func newEncodeResult(rowsWritten int64, code int64, msg string) C.MageEncodeResult {
	var cmsg *C.char
	if msg != "" {
		cmsg = C.CString(msg)
	}
	return C.MageEncodeResult{
		decision_rows_written: C.int64_t(rowsWritten),
		error_code:            C.int64_t(code),
		error_message:         cmsg,
	}
}

func parseEncodeConfigC(cfg *C.MageEncodeConfig) encodeConfig {
	return encodeConfig{
		maxOptions:          int64(cfg.max_options),
		maxTargetsPerOption: int64(cfg.max_targets_per_option),
		maxCachedChoices:    int64(cfg.max_cached_choices),
		zoneSlotCount:       int64(cfg.zone_slot_count),
		gameInfoDim:         int64(cfg.game_info_dim),
		optionScalarDim:     int64(cfg.option_scalar_dim),
		targetScalarDim:     int64(cfg.target_scalar_dim),
		decisionCapacity:    int64(cfg.decision_capacity),
		emitRenderPlan:      int64(cfg.emit_render_plan) != 0,
		renderPlanCapacity:  int64(cfg.render_plan_capacity),
		dedupCardBodies:     int64(cfg.dedup_card_bodies) != 0,
	}
}

func makeOutputViewsC(n int64, cfg encodeConfig, out *C.MageEncodeOutputs) (outputViews, *encodeError) {
	requiredI64 := func(ptr *C.int64_t, count int64, name string) ([]int64, *encodeError) {
		if ptr == nil {
			return nil, &encodeError{code: mageEncodeErrInvalidArgument, message: fmt.Sprintf("out.%s must be non-nil", name)}
		}
		return unsafe.Slice((*int64)(unsafe.Pointer(ptr)), count), nil
	}
	requiredF32 := func(ptr *C.float, count int64, name string) ([]float32, *encodeError) {
		if ptr == nil {
			return nil, &encodeError{code: mageEncodeErrInvalidArgument, message: fmt.Sprintf("out.%s must be non-nil", name)}
		}
		return unsafe.Slice((*float32)(unsafe.Pointer(ptr)), count), nil
	}
	requiredU8 := func(ptr *C.uint8_t, count int64, name string) ([]byte, *encodeError) {
		if ptr == nil {
			return nil, &encodeError{code: mageEncodeErrInvalidArgument, message: fmt.Sprintf("out.%s must be non-nil", name)}
		}
		return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), count), nil
	}
	requiredI32 := func(ptr *C.int32_t, count int64, name string) ([]int32, *encodeError) {
		if ptr == nil {
			return nil, &encodeError{code: mageEncodeErrInvalidArgument, message: fmt.Sprintf("out.%s must be non-nil", name)}
		}
		return unsafe.Slice((*int32)(unsafe.Pointer(ptr)), count), nil
	}

	view := outputViews{}
	var err *encodeError
	if view.traceKindID, err = requiredI64(out.trace_kind_id, n, "trace_kind_id"); err != nil {
		return view, err
	}
	if view.slotCardRows, err = requiredI64(out.slot_card_rows, n*cfg.zoneSlotCount, "slot_card_rows"); err != nil {
		return view, err
	}
	if view.slotOccupied, err = requiredF32(out.slot_occupied, n*cfg.zoneSlotCount, "slot_occupied"); err != nil {
		return view, err
	}
	if view.slotTapped, err = requiredF32(out.slot_tapped, n*cfg.zoneSlotCount, "slot_tapped"); err != nil {
		return view, err
	}
	if view.gameInfo, err = requiredF32(out.game_info, n*cfg.gameInfoDim, "game_info"); err != nil {
		return view, err
	}
	if view.pendingKindID, err = requiredI64(out.pending_kind_id, n, "pending_kind_id"); err != nil {
		return view, err
	}
	if view.numPresentOptions, err = requiredI64(out.num_present_options, n, "num_present_options"); err != nil {
		return view, err
	}
	if view.optionKindIDs, err = requiredI64(out.option_kind_ids, n*cfg.maxOptions, "option_kind_ids"); err != nil {
		return view, err
	}
	if view.optionScalars, err = requiredF32(out.option_scalars, n*cfg.maxOptions*cfg.optionScalarDim, "option_scalars"); err != nil {
		return view, err
	}
	if view.optionMask, err = requiredF32(out.option_mask, n*cfg.maxOptions, "option_mask"); err != nil {
		return view, err
	}
	if view.optionRefSlotIdx, err = requiredI64(out.option_ref_slot_idx, n*cfg.maxOptions, "option_ref_slot_idx"); err != nil {
		return view, err
	}
	if view.optionRefCardRow, err = requiredI64(out.option_ref_card_row, n*cfg.maxOptions, "option_ref_card_row"); err != nil {
		return view, err
	}
	if view.targetMask, err = requiredF32(out.target_mask, n*cfg.maxOptions*cfg.maxTargetsPerOption, "target_mask"); err != nil {
		return view, err
	}
	if view.targetTypeIDs, err = requiredI64(out.target_type_ids, n*cfg.maxOptions*cfg.maxTargetsPerOption, "target_type_ids"); err != nil {
		return view, err
	}
	if view.targetScalars, err = requiredF32(out.target_scalars, n*cfg.maxOptions*cfg.maxTargetsPerOption*cfg.targetScalarDim, "target_scalars"); err != nil {
		return view, err
	}
	if view.targetOverflow, err = requiredF32(out.target_overflow, n*cfg.maxOptions, "target_overflow"); err != nil {
		return view, err
	}
	if view.targetRefSlotIdx, err = requiredI64(out.target_ref_slot_idx, n*cfg.maxOptions*cfg.maxTargetsPerOption, "target_ref_slot_idx"); err != nil {
		return view, err
	}
	if view.targetRefIsPlayer, err = requiredU8(out.target_ref_is_player, n*cfg.maxOptions*cfg.maxTargetsPerOption, "target_ref_is_player"); err != nil {
		return view, err
	}
	if view.targetRefIsSelf, err = requiredU8(out.target_ref_is_self, n*cfg.maxOptions*cfg.maxTargetsPerOption, "target_ref_is_self"); err != nil {
		return view, err
	}
	if view.mayMask, err = requiredU8(out.may_mask, n, "may_mask"); err != nil {
		return view, err
	}
	if view.decisionStart, err = requiredI64(out.decision_start, n, "decision_start"); err != nil {
		return view, err
	}
	if view.decisionCount, err = requiredI64(out.decision_count, n, "decision_count"); err != nil {
		return view, err
	}
	if view.decisionOptionIdx, err = requiredI64(out.decision_option_idx, cfg.decisionCapacity*cfg.maxCachedChoices, "decision_option_idx"); err != nil {
		return view, err
	}
	if view.decisionTargetIdx, err = requiredI64(out.decision_target_idx, cfg.decisionCapacity*cfg.maxCachedChoices, "decision_target_idx"); err != nil {
		return view, err
	}
	if view.decisionMask, err = requiredU8(out.decision_mask, cfg.decisionCapacity*cfg.maxCachedChoices, "decision_mask"); err != nil {
		return view, err
	}
	if view.usesNoneHead, err = requiredU8(out.uses_none_head, cfg.decisionCapacity, "uses_none_head"); err != nil {
		return view, err
	}
	if cfg.emitRenderPlan {
		if view.renderPlan, err = requiredI32(out.render_plan, n*cfg.renderPlanCapacity, "render_plan"); err != nil {
			return view, err
		}
		if view.renderPlanLengths, err = requiredI64(out.render_plan_lengths, n, "render_plan_lengths"); err != nil {
			return view, err
		}
		if view.renderPlanOverflow, err = requiredI64(out.render_plan_overflow, n, "render_plan_overflow"); err != nil {
			return view, err
		}
	}
	return view, nil
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
		Turn:         g.CurrentTurn(),
		Step:         g.GetStep().String(),
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
		out = append(out, apiTarget{ID: id.String(), Label: label, IDUUID: id})
	}
	return out
}

func convertPriorityOptions(opts []interactive.ActionOption) []apiOption {
	out := make([]apiOption, 0, len(opts))
	for _, o := range opts {
		// magic-ai enumerates native rollout candidates from valid_targets alone.
		// If an option still has NeedsTarget=true but no current valid targets,
		// exposing it here creates an un-replayable targetless candidate.
		if o.NeedsTarget && len(o.ValidTargets) == 0 {
			continue
		}
		ao := apiOption{
			Label:        o.Label,
			CardName:     o.CardName,
			AbilityIndex: o.AbilityIndex,
			ManaCost:     o.ManaCost,
			ValidTargets: targetsFromActionOption(o),
		}
		if o.CardID != uuid.Nil {
			ao.CardID = o.CardID.String()
			ao.CardUUID = o.CardID
		}
		if o.PermanentID != uuid.Nil {
			ao.PermanentID = o.PermanentID.String()
			ao.PermanentUUID = o.PermanentID
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
			ao.IDUUID = o.ID
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
			ao.PermanentUUID = o.PermanentID
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
			ao.PermanentUUID = o.PermanentID
		}
		for i, id := range o.ValidTargets {
			label := ""
			if i < len(o.ValidTargetLabels) {
				label = o.ValidTargetLabels[i]
			}
			ao.ValidTargets = append(ao.ValidTargets, apiTarget{ID: id.String(), Label: label, IDUUID: id})
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

func winnerPlayerIndex(h *handle) int64 {
	if h == nil || h.game == nil {
		return -1
	}
	winner := h.current.Winner
	if winner == "" {
		winner = h.game.Winner()
	}
	if winner == "" {
		return -1
	}
	for idx := 0; idx < h.game.PlayerCount(); idx++ {
		player := h.game.PlayerAt(idx)
		if player != nil && player.Name() == winner {
			return int64(idx)
		}
	}
	return -1
}

func priorityActionFromChoiceCol(pending *apiPending, col int64, maxOptions int64, maxTargetsPerOption int64) (actionRequest, error) {
	optionCount := minInt64(int64(len(pending.Options)), maxOptions)
	candidateIdx := int64(0)
	for optIdx := int64(0); optIdx < optionCount; optIdx++ {
		option := pending.Options[optIdx]
		switch option.Kind {
		case "pass":
			if candidateIdx == col {
				return actionRequest{Kind: "pass"}, nil
			}
			candidateIdx++
		case "play_land":
			if candidateIdx == col {
				return actionRequest{Kind: "play_land", CardID: option.CardID}, nil
			}
			candidateIdx++
		case "cast_spell", "activate_ability":
			targetCount := minInt64(int64(len(option.ValidTargets)), maxTargetsPerOption)
			if targetCount == 0 {
				if candidateIdx == col {
					req := actionRequest{Kind: option.Kind}
					if option.Kind == "cast_spell" {
						req.CardID = option.CardID
					} else {
						req.PermanentID = option.PermanentID
						req.AbilityIndex = option.AbilityIndex
					}
					return req, nil
				}
				candidateIdx++
				continue
			}
			for tgtIdx := int64(0); tgtIdx < targetCount; tgtIdx++ {
				if candidateIdx == col {
					req := actionRequest{
						Kind:    option.Kind,
						Targets: []string{option.ValidTargets[tgtIdx].ID},
					}
					if option.Kind == "cast_spell" {
						req.CardID = option.CardID
					} else {
						req.PermanentID = option.PermanentID
						req.AbilityIndex = option.AbilityIndex
					}
					return req, nil
				}
				candidateIdx++
			}
		}
	}
	return actionRequest{}, fmt.Errorf("priority choice column %d out of range", col)
}

func actionFromStepChoice(pending *apiPending, selectedCols []int64, maySelected int64, maxOptions int64, maxTargetsPerOption int64) (actionRequest, error) {
	if pending == nil {
		return actionRequest{}, fmt.Errorf("no pending request")
	}
	switch pending.Kind {
	case "may":
		return actionRequest{Accepted: maySelected != 0}, nil
	case "priority":
		if len(selectedCols) != 1 {
			return actionRequest{}, fmt.Errorf("priority expects 1 selected choice column, got %d", len(selectedCols))
		}
		return priorityActionFromChoiceCol(pending, selectedCols[0], maxOptions, maxTargetsPerOption)
	case "attackers":
		optionCount := minInt64(int64(len(pending.Options)), maxOptions)
		if len(selectedCols) != int(optionCount) {
			return actionRequest{}, fmt.Errorf("attackers expects %d selected columns, got %d", optionCount, len(selectedCols))
		}
		attackers := make([]string, 0, len(selectedCols))
		for optIdx, rawCol := range selectedCols {
			if rawCol == 1 {
				if id := pending.Options[optIdx].PermanentID; id != "" {
					attackers = append(attackers, id)
				}
				continue
			}
			if rawCol != 0 {
				return actionRequest{}, fmt.Errorf("attacker row %d selected invalid column %d", optIdx, rawCol)
			}
		}
		return actionRequest{Attackers: attackers}, nil
	case "blockers":
		optionCount := minInt64(int64(len(pending.Options)), maxOptions)
		if len(selectedCols) != int(optionCount) {
			return actionRequest{}, fmt.Errorf("blockers expects %d selected columns, got %d", optionCount, len(selectedCols))
		}
		assignments := make([]blockerAssign, 0, len(selectedCols))
		for optIdx, rawCol := range selectedCols {
			if rawCol == 0 {
				continue
			}
			targetIdx := rawCol - 1
			if targetIdx < 0 || targetIdx >= minInt64(int64(len(pending.Options[optIdx].ValidTargets)), maxTargetsPerOption) {
				return actionRequest{}, fmt.Errorf("blocker row %d selected invalid column %d", optIdx, rawCol)
			}
			blockerID := pending.Options[optIdx].PermanentID
			attackerID := pending.Options[optIdx].ValidTargets[targetIdx].ID
			if blockerID != "" && attackerID != "" {
				assignments = append(assignments, blockerAssign{Blocker: blockerID, Attacker: attackerID})
			}
		}
		return actionRequest{Blockers: assignments}, nil
	case "mana_color":
		if len(selectedCols) != 1 {
			return actionRequest{}, fmt.Errorf("mana_color expects 1 selected column, got %d", len(selectedCols))
		}
		selectedIdx := selectedCols[0]
		if selectedIdx < 0 || selectedIdx >= minInt64(int64(len(pending.Options)), maxOptions) {
			return actionRequest{}, fmt.Errorf("mana_color selected invalid option %d", selectedIdx)
		}
		return actionRequest{SelectedColor: pending.Options[selectedIdx].Color}, nil
	case "permanent", "cards_from_hand", "card_from_library":
		if len(selectedCols) != 1 {
			return actionRequest{}, fmt.Errorf("%s expects 1 selected column, got %d", pending.Kind, len(selectedCols))
		}
		selectedIdx := selectedCols[0]
		if selectedIdx < 0 || selectedIdx >= minInt64(int64(len(pending.Options)), maxOptions) {
			return actionRequest{}, fmt.Errorf("%s selected invalid option %d", pending.Kind, selectedIdx)
		}
		selectedID := pending.Options[selectedIdx].ID
		if selectedID == "" {
			return actionRequest{SelectedIDs: nil}, nil
		}
		return actionRequest{SelectedIDs: []string{selectedID}}, nil
	case "mode", "number":
		if len(selectedCols) != 1 {
			return actionRequest{}, fmt.Errorf("%s expects 1 selected column, got %d", pending.Kind, len(selectedCols))
		}
		selectedIdx := selectedCols[0]
		if selectedIdx < 0 || selectedIdx >= minInt64(int64(len(pending.Options)), maxOptions) {
			return actionRequest{}, fmt.Errorf("%s selected invalid option %d", pending.Kind, selectedIdx)
		}
		return actionRequest{SelectedIndex: int(selectedIdx)}, nil
	default:
		if len(selectedCols) != 1 {
			return actionRequest{}, fmt.Errorf("%s expects 1 selected column, got %d", pending.Kind, len(selectedCols))
		}
		selectedIdx := selectedCols[0]
		if selectedIdx < 0 || selectedIdx >= minInt64(int64(len(pending.Options)), maxOptions) {
			return actionRequest{}, fmt.Errorf("%s selected invalid option %d", pending.Kind, selectedIdx)
		}
		return actionRequest{SelectedIndex: int(selectedIdx)}, nil
	}
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
	h.stateBuf = nil
	state := cachedSnapshotState(h)
	h.mu.Unlock()

	out := apiResponse{
		OK:       true,
		State:    state,
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
		State:    cachedSnapshotState(h),
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
	h.stateBuf = nil

	out := apiResponse{
		OK:       true,
		State:    cachedSnapshotState(h),
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
	sort.Strings(names)
	b, _ := json.Marshal(names)
	return C.CString(string(b))
}

//export MageRegisteredManaCosts
func MageRegisteredManaCosts() *C.char {
	defer func() { _ = recover() }()
	b, _ := json.Marshal(registeredManaCostStrings())
	return C.CString(string(b))
}

//export MageSetCardNameRows
func MageSetCardNameRows(cardNameRowsJSON *C.char) *C.char {
	defer func() { _ = recover() }()
	if cardNameRowsJSON == nil {
		return errResponse("null card mapping")
	}
	var rows map[string]int64
	if err := json.Unmarshal([]byte(C.GoString(cardNameRowsJSON)), &rows); err != nil {
		return errResponse("parse card mapping: %v", err)
	}
	setCardRowOverrides(rows)
	return toCStringResponse(apiResponse{OK: true})
}

//export MageBatchPoll
func MageBatchPoll(req *C.MageBatchRequest, out *C.MageBatchPollOutputs) (res C.MageEncodeResult) {
	defer func() {
		if r := recover(); r != nil {
			res = newEncodeResult(0, mageEncodeErrEncodeFailure, fmt.Sprintf("panic: %v", r))
		}
	}()
	if req == nil || out == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req and out must be non-nil")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	if n == 0 {
		return newEncodeResult(0, mageEncodeErrOK, "")
	}
	if req.handles == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.handles must be non-nil when n > 0")
	}
	if out.ready == nil || out.game_over == nil || out.pending_player_idx == nil || out.winner_player_idx == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "poll outputs must be non-nil")
	}

	handles := unsafe.Slice((*int64)(unsafe.Pointer(req.handles)), n)
	ready := unsafe.Slice((*int64)(unsafe.Pointer(out.ready)), n)
	gameOver := unsafe.Slice((*int64)(unsafe.Pointer(out.game_over)), n)
	pendingPlayerIdx := unsafe.Slice((*int64)(unsafe.Pointer(out.pending_player_idx)), n)
	winnerPlayerIdx := unsafe.Slice((*int64)(unsafe.Pointer(out.winner_player_idx)), n)

	for i, handleID := range handles {
		h := getHandle(handleID)
		if h == nil {
			return newEncodeResult(0, mageEncodeErrUnknownHandle, fmt.Sprintf("unknown handle %d", handleID))
		}
		h.mu.Lock()
		pending := buildPending(h.current)
		if h.done {
			gameOver[i] = 1
			ready[i] = 0
			pendingPlayerIdx[i] = -1
			winnerPlayerIdx[i] = winnerPlayerIndex(h)
		} else {
			gameOver[i] = 0
			if pending != nil {
				ready[i] = 1
				pendingPlayerIdx[i] = int64(pending.PlayerIdx)
			} else {
				ready[i] = 0
				pendingPlayerIdx[i] = -1
			}
			winnerPlayerIdx[i] = -1
		}
		h.mu.Unlock()
	}
	return newEncodeResult(0, mageEncodeErrOK, "")
}

//export MageBatchStepByChoice
func MageBatchStepByChoice(req *C.MageStepChoiceRequest) (res C.MageEncodeResult) {
	defer func() {
		if r := recover(); r != nil {
			res = newEncodeResult(0, mageEncodeErrEncodeFailure, fmt.Sprintf("panic: %v", r))
		}
	}()
	if req == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req must be non-nil")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	if req.max_options < 0 || req.max_targets_per_option < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "max_options and max_targets_per_option must be non-negative")
	}
	if n == 0 {
		return newEncodeResult(0, mageEncodeErrOK, "")
	}
	if req.handles == nil || req.decision_start == nil || req.decision_count == nil || req.may_selected == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "handles, decision_start, decision_count, and may_selected must be non-nil")
	}

	handles := unsafe.Slice((*int64)(unsafe.Pointer(req.handles)), n)
	decisionStart := unsafe.Slice((*int64)(unsafe.Pointer(req.decision_start)), n)
	decisionCount := unsafe.Slice((*int64)(unsafe.Pointer(req.decision_count)), n)
	maySelected := unsafe.Slice((*int64)(unsafe.Pointer(req.may_selected)), n)

	maxSelected := int64(0)
	for i := int64(0); i < n; i++ {
		if decisionStart[i] < 0 || decisionCount[i] < 0 {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "decision_start and decision_count must be non-negative")
		}
		if end := decisionStart[i] + decisionCount[i]; end > maxSelected {
			maxSelected = end
		}
	}
	var selectedChoiceCols []int64
	if maxSelected > 0 {
		if req.selected_choice_cols == nil {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "selected_choice_cols must be non-nil when any decision_count > 0")
		}
		selectedChoiceCols = unsafe.Slice((*int64)(unsafe.Pointer(req.selected_choice_cols)), maxSelected)
	}

	for i, handleID := range handles {
		h := getHandle(handleID)
		if h == nil {
			return newEncodeResult(0, mageEncodeErrUnknownHandle, fmt.Sprintf("unknown handle %d", handleID))
		}
		h.mu.Lock()
		if h.done {
			h.mu.Unlock()
			return newEncodeResult(0, mageEncodeErrGameOver, fmt.Sprintf("handle %d is over", handleID))
		}
		pending := buildPending(h.current)
		count := decisionCount[i]
		start := decisionStart[i]
		var cols []int64
		if count > 0 {
			if start+count > int64(len(selectedChoiceCols)) {
				h.mu.Unlock()
				return newEncodeResult(0, mageEncodeErrInvalidArgument, fmt.Sprintf("selection slice out of range for handle %d", handleID))
			}
			cols = selectedChoiceCols[start : start+count]
		}
		action, err := actionFromStepChoice(pending, cols, maySelected[i], int64(req.max_options), int64(req.max_targets_per_option))
		if err != nil {
			h.mu.Unlock()
			return newEncodeResult(0, mageEncodeErrInvalidArgument, fmt.Sprintf("handle %d: %v", handleID, err))
		}
		if err := routeAction(h, action); err != nil {
			h.mu.Unlock()
			return newEncodeResult(0, mageEncodeErrEncodeFailure, fmt.Sprintf("handle %d: %v", handleID, err))
		}
		ev := waitForNext(h)
		h.current = ev
		h.done = ev.Over
		h.stateBuf = nil
		h.mu.Unlock()
	}
	return newEncodeResult(0, mageEncodeErrOK, "")
}

//export MageEncodeBatch
func MageEncodeBatch(req *C.MageBatchRequest, cfg *C.MageEncodeConfig, out *C.MageEncodeOutputs) (res C.MageEncodeResult) {
	defer func() {
		if r := recover(); r != nil {
			res = newEncodeResult(0, mageEncodeErrEncodeFailure, fmt.Sprintf("panic: %v", r))
		}
	}()
	if req == nil || cfg == nil || out == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req, cfg, and out must be non-nil")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	cfgGo := parseEncodeConfigC(cfg)
	if err := validateEncodeConfig(cfgGo); err != nil {
		return newEncodeResult(0, err.code, err.message)
	}
	if n == 0 {
		return newEncodeResult(0, mageEncodeErrOK, "")
	}
	if req.handles == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.handles must be non-nil when n > 0")
	}
	reqGo := batchRequest{
		handles: unsafe.Slice((*int64)(unsafe.Pointer(req.handles)), n),
	}
	if req.perspective_player_idx != nil {
		reqGo.perspectives = unsafe.Slice((*int64)(unsafe.Pointer(req.perspective_player_idx)), n)
	}
	views, viewErr := makeOutputViewsC(n, cfgGo, out)
	if viewErr != nil {
		return newEncodeResult(0, viewErr.code, viewErr.message)
	}
	rowsWritten, err := encodeBatchGo(reqGo, cfgGo, views)
	if err != nil {
		return newEncodeResult(rowsWritten, err.code, err.message)
	}
	return newEncodeResult(rowsWritten, mageEncodeErrOK, "")
}

// Stores the borrowed pointers in “tokenTables“ for use by the future
// native text-encoder assembler. Returns 0 on success or a positive error
// code on a wire-format inconsistency. Calling with a nil pointer clears
// the registration.
//
//export MageRegisterTokenTables
func MageRegisterTokenTables(tables *C.MageTokenTables) C.int32_t {
	defer func() { _ = recover() }()
	if err := registerTokenTables(tables); err != nil {
		// Error path: keep prior registration intact, surface the code as
		// nonzero. We use 1 generically; the message is logged only in tests.
		return C.int32_t(1)
	}
	return C.int32_t(0)
}

// Returns a JSON summary of the currently registered token tables (sizes
// per category). Used by the Phase-3 round-trip parity test to verify the
// wire format unpacks correctly. Returns "null" if no tables registered.
//
//export MageTokenTableSummary
func MageTokenTableSummary() *C.char {
	defer func() { _ = recover() }()
	t := getTokenTables()
	if t == nil {
		return C.CString("null")
	}
	summary := map[string]any{
		"fragment_count":     len(t.structuralOffsets) - 1,
		"structural_tokens":  len(t.structuralTokens),
		"turn_min":           t.turnMin,
		"turn_max":           t.turnMax,
		"step_count":         t.stepCount,
		"turn_step_tokens":   len(t.turnStepTokens),
		"life_min":           t.lifeMin,
		"life_max":           t.lifeMax,
		"owner_count":        t.ownerCount,
		"life_owner_tokens":  len(t.lifeOwnerTokens),
		"ability_min":        t.abilityMin,
		"ability_max":        t.abilityMax,
		"ability_tokens":     len(t.abilityTokens),
		"count_min":          t.countMin,
		"count_max":          t.countMax,
		"count_tokens":       len(t.countTokens),
		"zone_count":         t.zoneCount,
		"zone_open_tokens":   len(t.zoneOpenTokens),
		"zone_close_tokens":  len(t.zoneCloseTok),
		"action_verb_count":  t.actionVerbCount,
		"action_verb_tokens": len(t.actionVerbTokens),
		"mana_color_count":   t.manaColorCount,
		"mana_tokens":        len(t.manaTokens),
		"card_ref_count":     t.cardRefCount,
		"card_row_count":     t.cardRowCount,
		"card_body_tokens":   len(t.cardBodyToks),
		"card_name_tokens":   len(t.cardNameToks),
		"card_closer":        t.cardCloser,
		"status_tapped":      t.statusTapped,
		"status_untapped":    t.statusUntapped,
		"pad_id":             t.padID,
		"option_id":          t.optionID,
		"target_open_id":     t.targetOpenID,
		"target_close_id":    t.targetCloseID,
		"tapped_id":          t.tappedID,
		"untapped_id":        t.untappedID,
		// Inline-blank singletons + digit table (Step 1 plumb-through).
		"choose_target_id":       t.chooseTargetID,
		"choose_block_id":        t.chooseBlockID,
		"choose_damage_order_id": t.chooseDamageOrderID,
		"choose_mode_id":         t.chooseModeID,
		"choose_may_id":          t.chooseMayID,
		"choose_x_digit_id":      t.chooseXDigitID,
		"choose_mana_source_id":  t.chooseManaSourceID,
		"choose_play_id":         t.choosePlayID,
		"use_ability_id":         t.useAbilityID,
		"chosen_id":              t.chosenID,
		"yes_id":                 t.yesID,
		"no_id":                  t.noID,
		"none_id":                t.noneID,
		"x_end_id":               t.xEndID,
		"mulligan_id":            t.mulliganID,
		"keep_id":                t.keepID,
		"num_count":              t.numCount,
		"num_ids":                t.numIDs,
	}
	b, err := json.Marshal(summary)
	if err != nil {
		return errResponse("marshal summary: %v", err)
	}
	return C.CString(string(b))
}

// Test/debug accessor: returns the JSON-encoded token-id list for a single
// (kind, key) pair. “kind“ is one of:
//
//	0=fragment, 1=turn_step, 2=life_owner, 3=ability, 4=count,
//	5=zone_open, 6=zone_close, 7=action_verb, 8=mana_glyph,
//	9=card_body, 10=card_name, 11=card_ref (single id list).
//
// Two key fields cover all (zero or one used).
//
//export MageTokenTableLookup
func MageTokenTableLookup(kind C.int32_t, k0 C.int32_t, k1 C.int32_t) *C.char {
	defer func() { _ = recover() }()
	t := getTokenTables()
	if t == nil {
		return C.CString("null")
	}
	var span []int32
	switch int32(kind) {
	case 0:
		span = t.fragmentSpan(int32(k0))
	case 1:
		span = t.turnStepSpan(int32(k0), int32(k1))
	case 2:
		span = t.lifeOwnerSpan(int32(k0), int32(k1))
	case 3:
		span = t.abilitySpan(int32(k0))
	case 4:
		span = t.countSpan(int32(k0))
	case 5:
		span = t.zoneOpenSpan(int32(k0), int32(k1))
	case 6:
		span = t.zoneCloseSpan(int32(k0), int32(k1))
	case 7:
		span = t.actionVerbSpan(int32(k0))
	case 8:
		span = t.manaGlyphSpan(int32(k0))
	case 9:
		span = t.cardBodySpan(int32(k0))
	case 10:
		span = t.cardNameSpan(int32(k0))
	case 11:
		idx := int32(k0)
		if idx < 0 || idx >= t.cardRefCount {
			span = nil
		} else {
			span = []int32{t.cardRefIDs[idx]}
		}
	case 12:
		// Inline-blank singletons keyed by index (see blankSingletonAt).
		// Indices 0..13 cover the 14 named singletons in abi.h order.
		if id, ok := t.blankSingletonAt(int32(k0)); ok {
			span = []int32{id}
		}
	case 13:
		// Digit token id for num_ids[k0].
		idx := int32(k0)
		if idx < 0 || idx >= t.numCount || int(idx) >= len(t.numIDs) {
			span = nil
		} else {
			span = []int32{t.numIDs[idx]}
		}
	default:
		return C.CString("null")
	}
	out := make([]int32, len(span))
	copy(out, span)
	b, err := json.Marshal(out)
	if err != nil {
		return errResponse("marshal: %v", err)
	}
	return C.CString(string(b))
}

//export MageEncodeTimingSummary
func MageEncodeTimingSummary(reset C.int32_t) *C.char {
	summary := packedEncodeTimingSnapshot(reset != 0)
	b, err := json.Marshal(summary)
	if err != nil {
		return errResponse("marshal timing summary: %v", err)
	}
	return C.CString(string(b))
}

//export MageEncodeTokensPacked
func MageEncodeTokensPacked(
	req *C.MageBatchRequest,
	cfg *C.MageEncodeConfig,
	out *C.MageEncodeOutputs,
	tokCfg *C.MageTokenAssemblerConfig,
	packedOut *C.MagePackedTokenAssemblerOutputs,
	blankCfg *C.MageBlankAssemblerConfig,
	blankOut *C.MagePackedBlankOutputs,
) (res C.MageEncodeResult) {
	defer func() {
		if r := recover(); r != nil {
			res = newEncodeResult(0, mageEncodeErrEncodeFailure, fmt.Sprintf("panic: %v", r))
		}
	}()
	if req == nil || cfg == nil || out == nil || tokCfg == nil || packedOut == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req, cfg, out, tok_cfg, packed_out must be non-nil")
	}
	if getTokenTables() == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "MageRegisterTokenTables must be called before MageEncodeTokensPacked")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	cfgGo := parseEncodeConfigC(cfg)
	cfgGo.emitTokensPacked = true
	cfgGo.tokenMaxTokens = int32(tokCfg.max_tokens)
	cfgGo.tokenMaxOptions = int32(tokCfg.max_options)
	cfgGo.tokenMaxTargets = int32(tokCfg.max_targets)
	cfgGo.tokenMaxCardRefs = int32(tokCfg.max_card_refs)
	if blankCfg != nil || blankOut != nil {
		if blankCfg == nil || blankOut == nil {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "blank_cfg and blank_out must be both nil or both non-nil")
		}
		cfgGo.blankMaxBlanks = int32(blankCfg.max_blanks)
		cfgGo.blankMaxLegal = int32(blankCfg.max_legal_per_blank)
	}
	if cfgGo.tokenMaxTokens <= 0 || cfgGo.tokenMaxOptions <= 0 ||
		cfgGo.tokenMaxTargets < 0 || cfgGo.tokenMaxCardRefs <= 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "token assembler config has non-positive dimension")
	}
	if (blankCfg != nil || blankOut != nil) && (cfgGo.blankMaxBlanks < 0 || cfgGo.blankMaxLegal < 0) {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "blank assembler config has negative dimension")
	}
	if err := validateEncodeConfig(cfgGo); err != nil {
		return newEncodeResult(0, err.code, err.message)
	}
	if n == 0 {
		return newEncodeResult(0, mageEncodeErrOK, "")
	}
	if req.handles == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.handles must be non-nil when n > 0")
	}
	reqGo := batchRequest{
		handles: unsafe.Slice((*int64)(unsafe.Pointer(req.handles)), n),
	}
	if req.perspective_player_idx != nil {
		reqGo.perspectives = unsafe.Slice((*int64)(unsafe.Pointer(req.perspective_player_idx)), n)
	}
	views, viewErr := makeOutputViewsC(n, cfgGo, out)
	if viewErr != nil {
		return newEncodeResult(0, viewErr.code, viewErr.message)
	}
	if err := attachPackedTokenViews(n, cfgGo, packedOut, &views); err != nil {
		return newEncodeResult(0, err.code, err.message)
	}
	if blankOut != nil {
		if err := attachPackedBlankViews(n, cfgGo, blankOut, &views); err != nil {
			return newEncodeResult(0, err.code, err.message)
		}
	}
	rowsWritten, err := encodeBatchGo(reqGo, cfgGo, views)
	if err != nil {
		return newEncodeResult(rowsWritten, err.code, err.message)
	}
	return newEncodeResult(rowsWritten, mageEncodeErrOK, "")
}

// attachPackedTokenViews wires the C-side packed token-assembler
// buffers into the outputViews slices. Token-shaped arrays are sized
// at the worst case “B * max_tokens“ so a single pre-allocated
// buffer can be reused across calls of varying live-token totals.
func attachPackedTokenViews(
	n int64,
	cfg encodeConfig,
	packedOut *C.MagePackedTokenAssemblerOutputs,
	views *outputViews,
) *encodeError {
	totalCap := n * int64(cfg.tokenMaxTokens)
	totalOptions := n * int64(cfg.tokenMaxOptions)
	totalTargets := n * int64(cfg.tokenMaxOptions) * int64(cfg.tokenMaxTargets)
	totalCardRefs := n * int64(cfg.tokenMaxCardRefs)

	if packedOut.token_ids == nil ||
		packedOut.cu_seqlens == nil ||
		packedOut.seq_lengths == nil ||
		packedOut.state_positions == nil ||
		packedOut.option_positions == nil ||
		packedOut.option_mask == nil ||
		packedOut.target_positions == nil ||
		packedOut.target_mask == nil ||
		packedOut.card_ref_positions == nil ||
		packedOut.token_overflow == nil {
		return &encodeError{code: mageEncodeErrInvalidArgument, message: "packed token outputs must be non-nil"}
	}

	views.packedTokenIDs = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.token_ids)), totalCap)
	views.packedCuSeqlens = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.cu_seqlens)), n+1)
	views.packedSeqLengths = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.seq_lengths)), n)
	views.packedStatePositions = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.state_positions)), n)
	views.packedOptionPos = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.option_positions)), totalOptions)
	views.packedOptionMask = unsafe.Slice((*byte)(unsafe.Pointer(packedOut.option_mask)), totalOptions)
	views.packedTargetPos = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.target_positions)), totalTargets)
	views.packedTargetMask = unsafe.Slice((*byte)(unsafe.Pointer(packedOut.target_mask)), totalTargets)
	views.packedCardRefPos = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.card_ref_positions)), totalCardRefs)
	views.packedTokenOverflow = unsafe.Slice((*int32)(unsafe.Pointer(packedOut.token_overflow)), n)
	return nil
}

func attachPackedBlankViews(
	n int64,
	cfg encodeConfig,
	blankOut *C.MagePackedBlankOutputs,
	views *outputViews,
) *encodeError {
	if cfg.blankMaxBlanks <= 0 || cfg.blankMaxLegal <= 0 {
		return nil
	}
	if int32(blankOut.k_max) != cfg.blankMaxBlanks || int32(blankOut.v_max) != cfg.blankMaxLegal {
		return &encodeError{code: mageEncodeErrInvalidArgument, message: "blank output dimensions must match blank assembler config"}
	}
	totalBlanks := n * int64(cfg.blankMaxBlanks)
	totalLegal := totalBlanks * int64(cfg.blankMaxLegal)
	if blankOut.blank_positions == nil ||
		blankOut.blank_kind == nil ||
		blankOut.blank_group == nil ||
		blankOut.blank_group_kind == nil ||
		blankOut.blank_option_index == nil ||
		blankOut.blank_legal_ids == nil ||
		blankOut.blank_legal_mask == nil ||
		blankOut.blank_overflow == nil {
		return &encodeError{code: mageEncodeErrInvalidArgument, message: "packed blank outputs must be non-nil"}
	}

	views.packedBlankPos = unsafe.Slice((*int32)(unsafe.Pointer(blankOut.blank_positions)), totalBlanks)
	views.packedBlankKind = unsafe.Slice((*int32)(unsafe.Pointer(blankOut.blank_kind)), totalBlanks)
	views.packedBlankGroup = unsafe.Slice((*int32)(unsafe.Pointer(blankOut.blank_group)), totalBlanks)
	views.packedBlankGroupKind = unsafe.Slice((*int32)(unsafe.Pointer(blankOut.blank_group_kind)), totalBlanks)
	views.packedBlankOptionIdx = unsafe.Slice((*int32)(unsafe.Pointer(blankOut.blank_option_index)), totalBlanks)
	views.packedBlankLegalIDs = unsafe.Slice((*int32)(unsafe.Pointer(blankOut.blank_legal_ids)), totalLegal)
	views.packedBlankLegalMask = unsafe.Slice((*byte)(unsafe.Pointer(blankOut.blank_legal_mask)), totalLegal)
	views.packedBlankOverflow = unsafe.Slice((*int32)(unsafe.Pointer(blankOut.blank_overflow)), n)
	return nil
}

//export MagePendingPlayer
func MagePendingPlayer(id C.int64_t) C.int64_t {
	defer func() { _ = recover() }()
	h := getHandle(int64(id))
	if h == nil {
		return -1
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return -1
	}
	if pending := buildPending(h.current); pending != nil {
		return C.int64_t(pending.PlayerIdx)
	}
	return -1
}

//export MageIsOver
func MageIsOver(id C.int64_t) C.int64_t {
	defer func() { _ = recover() }()
	h := getHandle(int64(id))
	if h == nil {
		return 1
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return 1
	}
	return 0
}

//export MageWinner
func MageWinner(id C.int64_t) *C.char {
	defer func() { _ = recover() }()
	h := getHandle(int64(id))
	if h == nil {
		return C.CString("")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current.Winner != "" {
		return C.CString(h.current.Winner)
	}
	if h.game != nil {
		return C.CString(h.game.Winner())
	}
	return C.CString("")
}

func main() {} // required for c-shared
