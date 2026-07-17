package main

/*
#include <stdlib.h>
#include "abi.h"
*/
import "C" //nolint:gocritic // C and unsafe are distinct despite cgo's synthetic package metadata.

import (
	"fmt"
	"sync"
	"time"
	"unsafe" //nolint:gocritic // Required for C buffer views; this is not a duplicate C import.
)

type textRolloutConfig struct {
	maxStepsPerGame     int64
	maxOptions          int64
	maxTargetsPerOption int64
	maxCachedChoices    int64
	zoneSlotCount       int64
	gameInfoDim         int64
	optionScalarDim     int64
	targetScalarDim     int64
	renderPlanCapacity  int64
	dedupCardBodies     bool
	maxTokens           int32
	maxCardRefs         int32
}

type textPendingRequest struct {
	requestID  int64
	handleID   int64
	slotID     int64
	episodeID  int64
	stepIndex  int64
	playerIdx  int64
	responseCh chan textChoice
}

type textChoice struct {
	decisionCount int64
	selectedCols  []int64
	maySelected   int64
}

type textTerminalEvent struct {
	slotID    int64
	episodeID int64
	winnerIdx int64
	isTimeout int64
	lifeP0    int64
	lifeP1    int64
}

type textRolloutScheduler struct {
	cfg       textRolloutConfig
	stopCh    chan struct{}
	readyCh   chan textPendingRequest
	termCh    chan textTerminalEvent
	wg        sync.WaitGroup
	mu        sync.Mutex
	nextReqID int64
	pending   map[int64]textPendingRequest
	stopped   bool
}

var textRollout struct {
	mu        sync.Mutex
	scheduler *textRolloutScheduler
}

func newTextReadyResult(rows, terminals, decisions, code int64, msg string) C.MageTextReadyBatchResult {
	var cmsg *C.char
	if msg != "" {
		cmsg = C.CString(msg)
	}
	return C.MageTextReadyBatchResult{
		rows_written:            C.int64_t(rows),
		terminal_events_written: C.int64_t(terminals),
		decision_rows_written:   C.int64_t(decisions),
		error_code:              C.int64_t(code),
		error_message:           cmsg,
	}
}

func textRolloutLifeTotals(h *handle) (int64, int64) {
	if h == nil || h.game == nil || h.game.PlayerCount() < 2 {
		return 0, 0
	}
	p0 := h.game.PlayerAt(0)
	p1 := h.game.PlayerAt(1)
	l0, l1 := int64(0), int64(0)
	if p0 != nil {
		l0 = int64(p0.Life())
	}
	if p1 != nil {
		l1 = int64(p1.Life())
	}
	return l0, l1
}

func (s *textRolloutScheduler) reserveRequest(req textPendingRequest) (textPendingRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return req, false
	}
	s.nextReqID++
	req.requestID = s.nextReqID
	s.pending[req.requestID] = req
	return req, true
}

func (s *textRolloutScheduler) removePending(requestID int64) {
	s.mu.Lock()
	delete(s.pending, requestID)
	s.mu.Unlock()
}

func (s *textRolloutScheduler) emitTerminal(term textTerminalEvent) {
	select {
	case s.termCh <- term:
	case <-s.stopCh:
	}
}

func (s *textRolloutScheduler) emitAbort(slotID, episodeID int64) {
	s.emitTerminal(textTerminalEvent{
		slotID: slotID, episodeID: episodeID, winnerIdx: -2,
		isTimeout: 1, lifeP0: 0, lifeP1: 0,
	})
}

func (s *textRolloutScheduler) addWorker(handleID, slotID, episodeID int64) bool {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return false
	}
	s.wg.Add(1)
	s.mu.Unlock()
	go s.worker(handleID, slotID, episodeID)
	return true
}

func (s *textRolloutScheduler) worker(handleID, slotID, episodeID int64) {
	defer s.wg.Done()
	stepIndex := int64(0)
	for {
		select {
		case <-s.stopCh:
			return
		default:
		}
		h := getHandle(handleID)
		if h == nil {
			s.emitAbort(slotID, episodeID)
			return
		}
		h.mu.Lock()
		if h.done {
			l0, l1 := textRolloutLifeTotals(h)
			term := textTerminalEvent{
				slotID: slotID, episodeID: episodeID, winnerIdx: winnerPlayerIndex(h),
				isTimeout: 0, lifeP0: l0, lifeP1: l1,
			}
			h.mu.Unlock()
			s.emitTerminal(term)
			return
		}
		if s.cfg.maxStepsPerGame > 0 && stepIndex >= s.cfg.maxStepsPerGame {
			l0, l1 := textRolloutLifeTotals(h)
			h.mu.Unlock()
			s.emitTerminal(textTerminalEvent{slotID: slotID, episodeID: episodeID, winnerIdx: -1, isTimeout: 1, lifeP0: l0, lifeP1: l1})
			return
		}
		pending := buildPending(h.current)
		if pending == nil {
			h.mu.Unlock()
			s.emitAbort(slotID, episodeID)
			return
		}
		playerIdx := int64(pending.PlayerIdx)
		h.mu.Unlock()

		req, ok := s.reserveRequest(textPendingRequest{
			handleID: handleID, slotID: slotID, episodeID: episodeID,
			stepIndex: stepIndex, playerIdx: playerIdx, responseCh: make(chan textChoice, 1),
		})
		if !ok {
			return
		}
		select {
		case s.readyCh <- req:
		case <-s.stopCh:
			s.removePending(req.requestID)
			return
		}

		var choice textChoice
		select {
		case choice = <-req.responseCh:
		case <-s.stopCh:
			s.removePending(req.requestID)
			return
		}
		s.removePending(req.requestID)

		h = getHandle(handleID)
		if h == nil {
			s.emitAbort(slotID, episodeID)
			return
		}
		h.mu.Lock()
		pending = buildPending(h.current)
		if pending == nil {
			h.mu.Unlock()
			s.emitAbort(slotID, episodeID)
			return
		}
		action, err := actionFromStepChoice(
			pending,
			choice.selectedCols,
			choice.maySelected,
			s.cfg.maxOptions,
			s.cfg.maxTargetsPerOption,
		)
		if err != nil {
			h.mu.Unlock()
			s.emitAbort(slotID, episodeID)
			return
		}
		routeErr := routeAction(h, action)
		if routeErr != nil {
			h.mu.Unlock()
			s.emitAbort(slotID, episodeID)
			return
		}
		ev := waitForNext(h)
		h.current = ev
		h.done = ev.Over
		h.stateBuf = nil
		h.mu.Unlock()
		stepIndex++
	}
}

func (s *textRolloutScheduler) stop(wait bool) {
	s.mu.Lock()
	if !s.stopped {
		s.stopped = true
		close(s.stopCh)
	}
	s.mu.Unlock()
	if wait {
		s.wg.Wait()
	}
}

func validateTextRolloutHandles(handles []int64) (int64, bool) {
	for _, handleID := range handles {
		if getHandle(handleID) == nil {
			return handleID, false
		}
	}
	return 0, true
}

func textRolloutSchedulerCurrent() *textRolloutScheduler {
	textRollout.mu.Lock()
	defer textRollout.mu.Unlock()
	return textRollout.scheduler
}

func makeTextEncodeConfig(cfg textRolloutConfig, rows int64) encodeConfig {
	return encodeConfig{
		maxOptions:          cfg.maxOptions,
		maxTargetsPerOption: cfg.maxTargetsPerOption,
		maxCachedChoices:    cfg.maxCachedChoices,
		zoneSlotCount:       cfg.zoneSlotCount,
		gameInfoDim:         cfg.gameInfoDim,
		optionScalarDim:     cfg.optionScalarDim,
		targetScalarDim:     cfg.targetScalarDim,
		decisionCapacity:    rows * cfg.maxOptions,
		emitTokensPacked:    true,
		renderPlanCapacity:  cfg.renderPlanCapacity,
		dedupCardBodies:     cfg.dedupCardBodies,
		tokenMaxTokens:      cfg.maxTokens,
		tokenMaxOptions:     int32(cfg.maxOptions),
		tokenMaxTargets:     int32(cfg.maxTargetsPerOption),
		tokenMaxCardRefs:    cfg.maxCardRefs,
	}
}

//export MageStartTextRollout
func MageStartTextRollout(req *C.MageTextRolloutStartRequest) (res C.MageEncodeResult) {
	if req == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req must be non-nil")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	if n > 0 && (req.handles == nil || req.slot_ids == nil || req.episode_ids == nil) {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "handles, slot_ids, and episode_ids must be non-nil")
	}
	cfg := textRolloutConfig{
		maxStepsPerGame:     int64(req.max_steps_per_game),
		maxOptions:          int64(req.max_options),
		maxTargetsPerOption: int64(req.max_targets_per_option),
		maxCachedChoices:    int64(req.max_cached_choices),
		zoneSlotCount:       int64(req.zone_slot_count),
		gameInfoDim:         int64(req.game_info_dim),
		optionScalarDim:     int64(req.option_scalar_dim),
		targetScalarDim:     int64(req.target_scalar_dim),
		renderPlanCapacity:  int64(req.render_plan_capacity),
		dedupCardBodies:     int64(req.dedup_card_bodies) != 0,
		maxTokens:           int32(req.max_tokens),
		maxCardRefs:         int32(req.max_card_refs),
	}
	if err := validateEncodeConfig(makeTextEncodeConfig(cfg, textMaxInt64(1, n))); err != nil {
		return newEncodeResult(0, err.code, err.message)
	}
	if getTokenTables() == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "MageRegisterTokenTables must be called before MageStartTextRollout")
	}
	readyCap := int(req.ready_queue_capacity)
	if readyCap <= 0 {
		readyCap = int(textMaxInt64(1, n))
	}
	termCap := int(req.terminal_queue_capacity)
	if termCap <= 0 {
		termCap = int(textMaxInt64(1, n))
	}
	handles := unsafe.Slice((*int64)(unsafe.Pointer(req.handles)), n)
	slots := unsafe.Slice((*int64)(unsafe.Pointer(req.slot_ids)), n)
	episodes := unsafe.Slice((*int64)(unsafe.Pointer(req.episode_ids)), n)
	if handleID, ok := validateTextRolloutHandles(handles); !ok {
		return newEncodeResult(0, mageEncodeErrUnknownHandle, fmt.Sprintf("unknown handle %d", handleID))
	}
	s := &textRolloutScheduler{
		cfg: cfg, stopCh: make(chan struct{}), readyCh: make(chan textPendingRequest, readyCap),
		termCh:  make(chan textTerminalEvent, termCap),
		pending: make(map[int64]textPendingRequest),
	}

	textRollout.mu.Lock()
	if textRollout.scheduler != nil {
		textRollout.scheduler.stop(true)
	}
	textRollout.scheduler = s
	textRollout.mu.Unlock()

	for i := int64(0); i < n; i++ {
		if !s.addWorker(handles[i], slots[i], episodes[i]) {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "text rollout scheduler is stopping")
		}
	}
	return newEncodeResult(0, mageEncodeErrOK, "")
}

//export MageAddTextRolloutGames
func MageAddTextRolloutGames(req *C.MageTextRolloutStartRequest) (res C.MageEncodeResult) {
	if req == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req must be non-nil")
	}
	s := textRolloutSchedulerCurrent()
	if s == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "text rollout scheduler is not running")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	if n > 0 && (req.handles == nil || req.slot_ids == nil || req.episode_ids == nil) {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "handles, slot_ids, and episode_ids must be non-nil")
	}
	handles := unsafe.Slice((*int64)(unsafe.Pointer(req.handles)), n)
	slots := unsafe.Slice((*int64)(unsafe.Pointer(req.slot_ids)), n)
	episodes := unsafe.Slice((*int64)(unsafe.Pointer(req.episode_ids)), n)
	if handleID, ok := validateTextRolloutHandles(handles); !ok {
		return newEncodeResult(0, mageEncodeErrUnknownHandle, fmt.Sprintf("unknown handle %d", handleID))
	}
	for i := int64(0); i < n; i++ {
		if !s.addWorker(handles[i], slots[i], episodes[i]) {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "text rollout scheduler is stopping")
		}
	}
	return newEncodeResult(0, mageEncodeErrOK, "")
}

//export MageNextTextInferenceBatch
func MageNextTextInferenceBatch(maxRows C.int64_t, timeoutMS C.int64_t, out *C.MageTextReadyBatchOutputs) (res C.MageTextReadyBatchResult) {
	defer func() {
		if r := recover(); r != nil {
			res = newTextReadyResult(0, 0, 0, mageEncodeErrEncodeFailure, fmt.Sprintf("panic: %v", r))
		}
	}()
	s := textRolloutSchedulerCurrent()
	if s == nil {
		return newTextReadyResult(0, 0, 0, mageEncodeErrInvalidArgument, "text rollout scheduler is not running")
	}
	if maxRows < 0 {
		return newTextReadyResult(0, 0, 0, mageEncodeErrInvalidArgument, "max_rows must be non-negative")
	}
	if out == nil {
		return newTextReadyResult(0, 0, 0, mageEncodeErrInvalidArgument, "out must be non-nil")
	}
	limit := int(maxRows)
	if limit == 0 {
		return newTextReadyResult(0, 0, 0, mageEncodeErrOK, "")
	}

	ready := make([]textPendingRequest, 0, limit)
	terminals := make([]textTerminalEvent, 0, limit)
	deadline := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	defer deadline.Stop()
	collectOne := func() bool {
		select {
		case ev := <-s.termCh:
			terminals = append(terminals, ev)
			return true
		case req := <-s.readyCh:
			ready = append(ready, req)
			return true
		case <-deadline.C:
			return false
		case <-s.stopCh:
			return false
		}
	}
	if !collectOne() {
		return newTextReadyResult(0, 0, 0, mageEncodeErrOK, "")
	}
	for len(ready) < limit && len(terminals) < limit {
		select {
		case ev := <-s.termCh:
			terminals = append(terminals, ev)
		case req := <-s.readyCh:
			ready = append(ready, req)
		default:
			goto drained
		}
	}
drained:
	for i, ev := range terminals {
		if out.terminal_slot_ids == nil || out.terminal_episode_ids == nil || out.terminal_winner_idx == nil ||
			out.terminal_is_timeout == nil || out.terminal_life_p0 == nil || out.terminal_life_p1 == nil {
			return newTextReadyResult(0, 0, 0, mageEncodeErrInvalidArgument, "terminal output buffers must be non-nil")
		}
		unsafe.Slice((*int64)(unsafe.Pointer(out.terminal_slot_ids)), int(maxRows))[i] = ev.slotID
		unsafe.Slice((*int64)(unsafe.Pointer(out.terminal_episode_ids)), int(maxRows))[i] = ev.episodeID
		unsafe.Slice((*int64)(unsafe.Pointer(out.terminal_winner_idx)), int(maxRows))[i] = ev.winnerIdx
		unsafe.Slice((*int64)(unsafe.Pointer(out.terminal_is_timeout)), int(maxRows))[i] = ev.isTimeout
		unsafe.Slice((*int64)(unsafe.Pointer(out.terminal_life_p0)), int(maxRows))[i] = ev.lifeP0
		unsafe.Slice((*int64)(unsafe.Pointer(out.terminal_life_p1)), int(maxRows))[i] = ev.lifeP1
	}
	rows := int64(len(ready))
	if rows == 0 {
		return newTextReadyResult(0, int64(len(terminals)), 0, mageEncodeErrOK, "")
	}
	if out.request_ids == nil || out.slot_ids == nil || out.episode_ids == nil || out.step_indices == nil || out.perspective_player_idx == nil {
		return newTextReadyResult(0, 0, 0, mageEncodeErrInvalidArgument, "ready metadata output buffers must be non-nil")
	}
	requestIDs := unsafe.Slice((*int64)(unsafe.Pointer(out.request_ids)), rows)
	slotIDs := unsafe.Slice((*int64)(unsafe.Pointer(out.slot_ids)), rows)
	episodeIDs := unsafe.Slice((*int64)(unsafe.Pointer(out.episode_ids)), rows)
	stepIndices := unsafe.Slice((*int64)(unsafe.Pointer(out.step_indices)), rows)
	perspectives := unsafe.Slice((*int64)(unsafe.Pointer(out.perspective_player_idx)), rows)
	handles := make([]int64, rows)
	for i, req := range ready {
		requestIDs[i] = req.requestID
		slotIDs[i] = req.slotID
		episodeIDs[i] = req.episodeID
		stepIndices[i] = req.stepIndex
		perspectives[i] = req.playerIdx
		handles[i] = req.handleID
	}
	cfg := makeTextEncodeConfig(s.cfg, rows)
	views, viewErr := makeOutputViewsC(rows, cfg, &out.encode)
	if viewErr != nil {
		return newTextReadyResult(0, 0, 0, viewErr.code, viewErr.message)
	}
	if err := attachPackedTokenViews(rows, cfg, &out.packed_tokens, &views); err != nil {
		return newTextReadyResult(0, 0, 0, err.code, err.message)
	}
	decisions, err := encodeBatchGo(batchRequest{handles: handles, perspectives: perspectives}, cfg, views)
	if err != nil {
		return newTextReadyResult(rows, int64(len(terminals)), decisions, err.code, err.message)
	}
	return newTextReadyResult(rows, int64(len(terminals)), decisions, mageEncodeErrOK, "")
}

//export MageSubmitTextChoices
func MageSubmitTextChoices(req *C.MageTextChoiceSubmitRequest) (res C.MageEncodeResult) {
	defer func() {
		if r := recover(); r != nil {
			res = newEncodeResult(0, mageEncodeErrEncodeFailure, fmt.Sprintf("panic: %v", r))
		}
	}()
	s := textRolloutSchedulerCurrent()
	if s == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "text rollout scheduler is not running")
	}
	if req == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req must be non-nil")
	}
	n := int64(req.n)
	if n < 0 {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "req.n must be non-negative")
	}
	if n == 0 {
		return newEncodeResult(0, mageEncodeErrOK, "")
	}
	if req.request_ids == nil || req.decision_count == nil || req.may_selected == nil {
		return newEncodeResult(0, mageEncodeErrInvalidArgument, "request_ids, decision_count, and may_selected must be non-nil")
	}
	requestIDs := unsafe.Slice((*int64)(unsafe.Pointer(req.request_ids)), n)
	counts := unsafe.Slice((*int64)(unsafe.Pointer(req.decision_count)), n)
	mays := unsafe.Slice((*int64)(unsafe.Pointer(req.may_selected)), n)
	totalCols := int64(0)
	for _, c := range counts {
		if c < 0 {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "decision_count must be non-negative")
		}
		totalCols += c
	}
	var selected []int64
	if totalCols > 0 {
		if req.selected_choice_cols == nil {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "selected_choice_cols must be non-nil when decisions are present")
		}
		selected = unsafe.Slice((*int64)(unsafe.Pointer(req.selected_choice_cols)), totalCols)
	}
	cursor := int64(0)
	for i, requestID := range requestIDs {
		s.mu.Lock()
		pending, ok := s.pending[requestID]
		s.mu.Unlock()
		if !ok {
			return newEncodeResult(0, mageEncodeErrInvalidArgument, fmt.Sprintf("request_id %d is not pending", requestID))
		}
		count := counts[i]
		cols := append([]int64(nil), selected[cursor:cursor+count]...)
		cursor += count
		select {
		case pending.responseCh <- textChoice{decisionCount: count, selectedCols: cols, maySelected: mays[i]}:
		case <-s.stopCh:
			return newEncodeResult(0, mageEncodeErrInvalidArgument, "text rollout scheduler is stopping")
		default:
			return newEncodeResult(0, mageEncodeErrInvalidArgument, fmt.Sprintf("request_id %d was already submitted", requestID))
		}
	}
	return newEncodeResult(0, mageEncodeErrOK, "")
}

//export MageStopTextRollout
func MageStopTextRollout(waitForActive C.int32_t) C.int32_t {
	textRollout.mu.Lock()
	s := textRollout.scheduler
	textRollout.scheduler = nil
	textRollout.mu.Unlock()
	if s != nil {
		s.stop(waitForActive != 0)
	}
	return 0
}

func textMaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
