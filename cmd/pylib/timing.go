package main

import (
	"sync/atomic"
	"time"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

type encodeTimingSnapshot struct {
	Calls             int64   `json:"calls"`
	TotalMs           float64 `json:"total_ms"`
	GameEncodeMs      float64 `json:"game_encode_ms"`
	RenderPlanMs      float64 `json:"render_plan_ms"`
	TokenAssemblyMs   float64 `json:"token_assembly_ms"`
	TokenMetadataMs   float64 `json:"token_metadata_ms"`
	TotalPerCallMs    float64 `json:"total_per_call_ms"`
	GamePerCallMs     float64 `json:"game_encode_per_call_ms"`
	RenderPerCallMs   float64 `json:"render_plan_per_call_ms"`
	AssemblyPerCallMs float64 `json:"token_assembly_per_call_ms"`
	MetadataPerCallMs float64 `json:"token_metadata_per_call_ms"`
}

type nativeLoopTimingSnapshot struct {
	EncodeCalls               int64   `json:"encode_calls"`
	EncodeRows                int64   `json:"encode_rows"`
	EncodeTotalMs             float64 `json:"encode_total_ms"`
	EncodeViewMs              float64 `json:"encode_view_ms"`
	EncodeClearMs             float64 `json:"encode_clear_ms"`
	EncodeStateActionMs       float64 `json:"encode_state_action_ms"`
	EncodeDecisionMs          float64 `json:"encode_decision_ms"`
	EncodeOtherMs             float64 `json:"encode_other_ms"`
	EncodeRowsPerCall         float64 `json:"encode_rows_per_call"`
	EncodeTotalPerCallMs      float64 `json:"encode_total_per_call_ms"`
	EncodeTotalPerRowMs       float64 `json:"encode_total_per_row_ms"`
	EncodeStateActionPerRowMs float64 `json:"encode_state_action_per_row_ms"`
	EncodeDecisionPerRowMs    float64 `json:"encode_decision_per_row_ms"`

	StepCalls             int64                          `json:"step_calls"`
	StepRows              int64                          `json:"step_rows"`
	StepTotalMs           float64                        `json:"step_total_ms"`
	StepPrepareMs         float64                        `json:"step_prepare_ms"`
	StepPendingActionMs   float64                        `json:"step_pending_action_ms"`
	StepRouteMs           float64                        `json:"step_route_ms"`
	StepWaitMs            float64                        `json:"step_wait_ms"`
	StepOtherMs           float64                        `json:"step_other_ms"`
	StepRowsPerCall       float64                        `json:"step_rows_per_call"`
	StepTotalPerCallMs    float64                        `json:"step_total_per_call_ms"`
	StepTotalPerRowMs     float64                        `json:"step_total_per_row_ms"`
	StepPendingPerRowMs   float64                        `json:"step_pending_action_per_row_ms"`
	StepRoutePerRowMs     float64                        `json:"step_route_per_row_ms"`
	StepWaitPerRowMs      float64                        `json:"step_wait_per_row_ms"`
	NativeTimingIsEnabled bool                           `json:"enabled"`
	Loop                  interactive.LoopTimingSnapshot `json:"loop"`
	Engine                mage.EngineTimingSnapshot      `json:"engine"`
}

var (
	packedTimingCalls      int64
	packedTimingTotalNs    int64
	packedTimingGameNs     int64
	packedTimingRenderNs   int64
	packedTimingAssemblyNs int64
	packedTimingMetadataNs int64

	nativeTimingEnabledFlag atomic.Int64
	nativeEncodeCalls       int64
	nativeEncodeRows        int64
	nativeEncodeTotalNs     int64
	nativeEncodeViewNs      int64
	nativeEncodeClearNs     int64
	nativeEncodeStateNs     int64
	nativeEncodeDecisionNs  int64
	nativeStepCalls         int64
	nativeStepRows          int64
	nativeStepTotalNs       int64
	nativeStepPrepareNs     int64
	nativeStepPendingNs     int64
	nativeStepRouteNs       int64
	nativeStepWaitNs        int64
)

func addPackedEncodeTiming(total, game, render, assembly, metadata time.Duration) {
	atomic.AddInt64(&packedTimingCalls, 1)
	atomic.AddInt64(&packedTimingTotalNs, int64(total))
	atomic.AddInt64(&packedTimingGameNs, int64(game))
	atomic.AddInt64(&packedTimingRenderNs, int64(render))
	atomic.AddInt64(&packedTimingAssemblyNs, int64(assembly))
	atomic.AddInt64(&packedTimingMetadataNs, int64(metadata))
}

func packedEncodeTimingSnapshot(reset bool) encodeTimingSnapshot {
	load := atomic.LoadInt64
	calls := load(&packedTimingCalls)
	total := load(&packedTimingTotalNs)
	game := load(&packedTimingGameNs)
	render := load(&packedTimingRenderNs)
	assembly := load(&packedTimingAssemblyNs)
	metadata := load(&packedTimingMetadataNs)
	if reset {
		atomic.StoreInt64(&packedTimingCalls, 0)
		atomic.StoreInt64(&packedTimingTotalNs, 0)
		atomic.StoreInt64(&packedTimingGameNs, 0)
		atomic.StoreInt64(&packedTimingRenderNs, 0)
		atomic.StoreInt64(&packedTimingAssemblyNs, 0)
		atomic.StoreInt64(&packedTimingMetadataNs, 0)
	}

	toMs := func(ns int64) float64 { return float64(ns) / 1e6 }
	perCall := func(ns int64) float64 {
		if calls == 0 {
			return 0
		}
		return toMs(ns) / float64(calls)
	}
	return encodeTimingSnapshot{
		Calls:             calls,
		TotalMs:           toMs(total),
		GameEncodeMs:      toMs(game),
		RenderPlanMs:      toMs(render),
		TokenAssemblyMs:   toMs(assembly),
		TokenMetadataMs:   toMs(metadata),
		TotalPerCallMs:    perCall(total),
		GamePerCallMs:     perCall(game),
		RenderPerCallMs:   perCall(render),
		AssemblyPerCallMs: perCall(assembly),
		MetadataPerCallMs: perCall(metadata),
	}
}

func nativeLoopTimingEnabled() bool {
	return nativeTimingEnabledFlag.Load() != 0
}

func addNativeEncodeTiming(rows int64, total, view, clear, stateAction, decision time.Duration) {
	atomic.AddInt64(&nativeEncodeCalls, 1)
	atomic.AddInt64(&nativeEncodeRows, rows)
	atomic.AddInt64(&nativeEncodeTotalNs, int64(total))
	atomic.AddInt64(&nativeEncodeViewNs, int64(view))
	atomic.AddInt64(&nativeEncodeClearNs, int64(clear))
	atomic.AddInt64(&nativeEncodeStateNs, int64(stateAction))
	atomic.AddInt64(&nativeEncodeDecisionNs, int64(decision))
}

func addNativeStepTiming(rows int64, total, prepare, pendingAction, route, wait time.Duration) {
	atomic.AddInt64(&nativeStepCalls, 1)
	atomic.AddInt64(&nativeStepRows, rows)
	atomic.AddInt64(&nativeStepTotalNs, int64(total))
	atomic.AddInt64(&nativeStepPrepareNs, int64(prepare))
	atomic.AddInt64(&nativeStepPendingNs, int64(pendingAction))
	atomic.AddInt64(&nativeStepRouteNs, int64(route))
	atomic.AddInt64(&nativeStepWaitNs, int64(wait))
}

func nativeLoopTimingTakeSnapshot(reset bool) nativeLoopTimingSnapshot {
	load := atomic.LoadInt64
	encodeCalls := load(&nativeEncodeCalls)
	encodeRows := load(&nativeEncodeRows)
	encodeTotal := load(&nativeEncodeTotalNs)
	encodeView := load(&nativeEncodeViewNs)
	encodeClear := load(&nativeEncodeClearNs)
	encodeState := load(&nativeEncodeStateNs)
	encodeDecision := load(&nativeEncodeDecisionNs)
	stepCalls := load(&nativeStepCalls)
	stepRows := load(&nativeStepRows)
	stepTotal := load(&nativeStepTotalNs)
	stepPrepare := load(&nativeStepPrepareNs)
	stepPending := load(&nativeStepPendingNs)
	stepRoute := load(&nativeStepRouteNs)
	stepWait := load(&nativeStepWaitNs)
	if reset {
		nativeTimingEnabledFlag.Store(1)
		interactive.EnableLoopTiming(true)
		mage.EnableEngineTiming(true)
		atomic.StoreInt64(&nativeEncodeCalls, 0)
		atomic.StoreInt64(&nativeEncodeRows, 0)
		atomic.StoreInt64(&nativeEncodeTotalNs, 0)
		atomic.StoreInt64(&nativeEncodeViewNs, 0)
		atomic.StoreInt64(&nativeEncodeClearNs, 0)
		atomic.StoreInt64(&nativeEncodeStateNs, 0)
		atomic.StoreInt64(&nativeEncodeDecisionNs, 0)
		atomic.StoreInt64(&nativeStepCalls, 0)
		atomic.StoreInt64(&nativeStepRows, 0)
		atomic.StoreInt64(&nativeStepTotalNs, 0)
		atomic.StoreInt64(&nativeStepPrepareNs, 0)
		atomic.StoreInt64(&nativeStepPendingNs, 0)
		atomic.StoreInt64(&nativeStepRouteNs, 0)
		atomic.StoreInt64(&nativeStepWaitNs, 0)
	}

	toMs := func(ns int64) float64 { return float64(ns) / 1e6 }
	perCall := func(ns, calls int64) float64 {
		if calls == 0 {
			return 0
		}
		return toMs(ns) / float64(calls)
	}
	perRow := func(ns, rows int64) float64 {
		if rows == 0 {
			return 0
		}
		return toMs(ns) / float64(rows)
	}
	perRowsCall := func(rows, calls int64) float64 {
		if calls == 0 {
			return 0
		}
		return float64(rows) / float64(calls)
	}
	encodeKnown := encodeView + encodeClear + encodeState + encodeDecision
	stepKnown := stepPrepare + stepPending + stepRoute + stepWait
	return nativeLoopTimingSnapshot{
		EncodeCalls:               encodeCalls,
		EncodeRows:                encodeRows,
		EncodeTotalMs:             toMs(encodeTotal),
		EncodeViewMs:              toMs(encodeView),
		EncodeClearMs:             toMs(encodeClear),
		EncodeStateActionMs:       toMs(encodeState),
		EncodeDecisionMs:          toMs(encodeDecision),
		EncodeOtherMs:             toMs(encodeTotal - encodeKnown),
		EncodeRowsPerCall:         perRowsCall(encodeRows, encodeCalls),
		EncodeTotalPerCallMs:      perCall(encodeTotal, encodeCalls),
		EncodeTotalPerRowMs:       perRow(encodeTotal, encodeRows),
		EncodeStateActionPerRowMs: perRow(encodeState, encodeRows),
		EncodeDecisionPerRowMs:    perRow(encodeDecision, encodeRows),
		StepCalls:                 stepCalls,
		StepRows:                  stepRows,
		StepTotalMs:               toMs(stepTotal),
		StepPrepareMs:             toMs(stepPrepare),
		StepPendingActionMs:       toMs(stepPending),
		StepRouteMs:               toMs(stepRoute),
		StepWaitMs:                toMs(stepWait),
		StepOtherMs:               toMs(stepTotal - stepKnown),
		StepRowsPerCall:           perRowsCall(stepRows, stepCalls),
		StepTotalPerCallMs:        perCall(stepTotal, stepCalls),
		StepTotalPerRowMs:         perRow(stepTotal, stepRows),
		StepPendingPerRowMs:       perRow(stepPending, stepRows),
		StepRoutePerRowMs:         perRow(stepRoute, stepRows),
		StepWaitPerRowMs:          perRow(stepWait, stepRows),
		NativeTimingIsEnabled:     nativeLoopTimingEnabled(),
		Loop:                      interactive.LoopTimingSnapshotAndReset(false),
		Engine:                    mage.EngineTimingSnapshotAndReset(false),
	}
}
