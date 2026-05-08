package mage

import (
	"sync/atomic"
	"time"
)

type EngineTimingSnapshot struct {
	Enabled             bool    `json:"enabled"`
	PriorityRounds      int64   `json:"engine_priority_rounds"`
	PriorityIterations  int64   `json:"engine_priority_iterations"`
	SBAMs               float64 `json:"engine_sba_ms"`
	TriggersMs          float64 `json:"engine_triggers_ms"`
	OnPriorityMs        float64 `json:"engine_on_priority_ms"`
	ExecuteActionMs     float64 `json:"engine_execute_action_ms"`
	AfterActionMs       float64 `json:"engine_after_action_ms"`
	ResolveMs           float64 `json:"engine_resolve_ms"`
	ApplyEffectsMs      float64 `json:"engine_apply_effects_ms"`
	SBAPerIterationUs   float64 `json:"engine_sba_per_iteration_us"`
	TriggersPerIterUs   float64 `json:"engine_triggers_per_iteration_us"`
	OnPriorityPerIterUs float64 `json:"engine_on_priority_per_iteration_us"`
	ExecutePerActionUs  float64 `json:"engine_execute_per_action_us"`
}

var (
	engineTimingEnabled            int64
	engineTimingPriorityRounds     int64
	engineTimingPriorityIterations int64
	engineTimingSBANs              int64
	engineTimingTriggersNs         int64
	engineTimingOnPriorityNs       int64
	engineTimingExecuteNs          int64
	engineTimingAfterNs            int64
	engineTimingResolveNs          int64
	engineTimingApplyEffectsNs     int64
	engineTimingExecutedActions    int64
)

func EngineTimingEnabled() bool {
	return atomic.LoadInt64(&engineTimingEnabled) != 0
}

func EnableEngineTiming(reset bool) {
	atomic.StoreInt64(&engineTimingEnabled, 1)
	if reset {
		atomic.StoreInt64(&engineTimingPriorityRounds, 0)
		atomic.StoreInt64(&engineTimingPriorityIterations, 0)
		atomic.StoreInt64(&engineTimingSBANs, 0)
		atomic.StoreInt64(&engineTimingTriggersNs, 0)
		atomic.StoreInt64(&engineTimingOnPriorityNs, 0)
		atomic.StoreInt64(&engineTimingExecuteNs, 0)
		atomic.StoreInt64(&engineTimingAfterNs, 0)
		atomic.StoreInt64(&engineTimingResolveNs, 0)
		atomic.StoreInt64(&engineTimingApplyEffectsNs, 0)
		atomic.StoreInt64(&engineTimingExecutedActions, 0)
	}
}

func addEngineTiming(target *int64, elapsed time.Duration) {
	atomic.AddInt64(target, int64(elapsed))
}

func addEnginePriorityRound() {
	atomic.AddInt64(&engineTimingPriorityRounds, 1)
}

func addEnginePriorityIteration() {
	atomic.AddInt64(&engineTimingPriorityIterations, 1)
}

func addEngineExecutedAction() {
	atomic.AddInt64(&engineTimingExecutedActions, 1)
}

func EngineTimingSnapshotAndReset(reset bool) EngineTimingSnapshot {
	load := atomic.LoadInt64
	rounds := load(&engineTimingPriorityRounds)
	iterations := load(&engineTimingPriorityIterations)
	executed := load(&engineTimingExecutedActions)
	sba := load(&engineTimingSBANs)
	triggers := load(&engineTimingTriggersNs)
	onPriority := load(&engineTimingOnPriorityNs)
	execute := load(&engineTimingExecuteNs)
	after := load(&engineTimingAfterNs)
	resolve := load(&engineTimingResolveNs)
	applyEffects := load(&engineTimingApplyEffectsNs)
	if reset {
		EnableEngineTiming(true)
	}

	toMs := func(ns int64) float64 { return float64(ns) / 1e6 }
	perIterUs := func(ns int64) float64 {
		if iterations == 0 {
			return 0
		}
		return float64(ns) / 1000.0 / float64(iterations)
	}
	perActionUs := func(ns int64) float64 {
		if executed == 0 {
			return 0
		}
		return float64(ns) / 1000.0 / float64(executed)
	}
	return EngineTimingSnapshot{
		Enabled:             EngineTimingEnabled(),
		PriorityRounds:      rounds,
		PriorityIterations:  iterations,
		SBAMs:               toMs(sba),
		TriggersMs:          toMs(triggers),
		OnPriorityMs:        toMs(onPriority),
		ExecuteActionMs:     toMs(execute),
		AfterActionMs:       toMs(after),
		ResolveMs:           toMs(resolve),
		ApplyEffectsMs:      toMs(applyEffects),
		SBAPerIterationUs:   perIterUs(sba),
		TriggersPerIterUs:   perIterUs(triggers),
		OnPriorityPerIterUs: perIterUs(onPriority),
		ExecutePerActionUs:  perActionUs(execute),
	}
}
