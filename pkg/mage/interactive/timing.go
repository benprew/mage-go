package interactive

import (
	"sync/atomic"
	"time"
)

type LoopTimingSnapshot struct {
	Enabled               bool    `json:"enabled"`
	StepCalls             int64   `json:"loop_step_calls"`
	StepTotalMs           float64 `json:"loop_step_total_ms"`
	StepPerCallMs         float64 `json:"loop_step_per_call_ms"`
	GetActionCalls        int64   `json:"loop_get_action_calls"`
	GetAvailableMs        float64 `json:"loop_get_available_ms"`
	SendPromptMs          float64 `json:"loop_send_prompt_ms"`
	ReadActionMs          float64 `json:"loop_read_action_ms"`
	AutoPasses            int64   `json:"loop_auto_passes"`
	PromptNoneSends       int64   `json:"loop_prompt_none_sends"`
	PromptDecisionSends   int64   `json:"loop_prompt_decision_sends"`
	GetAvailablePerCallUs float64 `json:"loop_get_available_per_call_us"`
	SendPromptPerCallUs   float64 `json:"loop_send_prompt_per_call_us"`
	ReadActionPerCallUs   float64 `json:"loop_read_action_per_call_us"`
}

var (
	loopTimingEnabled        atomic.Int64
	loopTimingStepCalls      int64
	loopTimingStepNs         int64
	loopTimingGetCalls       int64
	loopTimingGetNs          int64
	loopTimingSendNs         int64
	loopTimingReadNs         int64
	loopTimingAutoPasses     int64
	loopTimingPromptNone     int64
	loopTimingPromptDecision int64
)

func LoopTimingEnabled() bool {
	return loopTimingEnabled.Load() != 0
}

func EnableLoopTiming(reset bool) {
	loopTimingEnabled.Store(1)
	if reset {
		atomic.StoreInt64(&loopTimingStepCalls, 0)
		atomic.StoreInt64(&loopTimingStepNs, 0)
		atomic.StoreInt64(&loopTimingGetCalls, 0)
		atomic.StoreInt64(&loopTimingGetNs, 0)
		atomic.StoreInt64(&loopTimingSendNs, 0)
		atomic.StoreInt64(&loopTimingReadNs, 0)
		atomic.StoreInt64(&loopTimingAutoPasses, 0)
		atomic.StoreInt64(&loopTimingPromptNone, 0)
		atomic.StoreInt64(&loopTimingPromptDecision, 0)
	}
}

func AddLoopStepTiming(elapsed time.Duration) {
	atomic.AddInt64(&loopTimingStepCalls, 1)
	atomic.AddInt64(&loopTimingStepNs, int64(elapsed))
}

func AddLoopGetActionTiming(getAvailable, sendPrompt, readAction time.Duration, autoPass bool, prompt PromptType) {
	atomic.AddInt64(&loopTimingGetCalls, 1)
	atomic.AddInt64(&loopTimingGetNs, int64(getAvailable))
	atomic.AddInt64(&loopTimingSendNs, int64(sendPrompt))
	atomic.AddInt64(&loopTimingReadNs, int64(readAction))
	if autoPass {
		atomic.AddInt64(&loopTimingAutoPasses, 1)
		return
	}
	if prompt == PromptNone {
		atomic.AddInt64(&loopTimingPromptNone, 1)
	} else if prompt != 0 {
		atomic.AddInt64(&loopTimingPromptDecision, 1)
	}
}

func LoopTimingSnapshotAndReset(reset bool) LoopTimingSnapshot {
	load := atomic.LoadInt64
	stepCalls := load(&loopTimingStepCalls)
	getCalls := load(&loopTimingGetCalls)
	stepNs := load(&loopTimingStepNs)
	getNs := load(&loopTimingGetNs)
	sendNs := load(&loopTimingSendNs)
	readNs := load(&loopTimingReadNs)
	autoPasses := load(&loopTimingAutoPasses)
	promptNone := load(&loopTimingPromptNone)
	promptDecision := load(&loopTimingPromptDecision)
	if reset {
		EnableLoopTiming(true)
	}

	toMs := func(ns int64) float64 { return float64(ns) / 1e6 }
	perMs := func(ns, calls int64) float64 {
		if calls == 0 {
			return 0
		}
		return toMs(ns) / float64(calls)
	}
	perUs := func(ns, calls int64) float64 {
		if calls == 0 {
			return 0
		}
		return float64(ns) / 1000.0 / float64(calls)
	}
	return LoopTimingSnapshot{
		Enabled:               LoopTimingEnabled(),
		StepCalls:             stepCalls,
		StepTotalMs:           toMs(stepNs),
		StepPerCallMs:         perMs(stepNs, stepCalls),
		GetActionCalls:        getCalls,
		GetAvailableMs:        toMs(getNs),
		SendPromptMs:          toMs(sendNs),
		ReadActionMs:          toMs(readNs),
		AutoPasses:            autoPasses,
		PromptNoneSends:       promptNone,
		PromptDecisionSends:   promptDecision,
		GetAvailablePerCallUs: perUs(getNs, getCalls),
		SendPromptPerCallUs:   perUs(sendNs, getCalls),
		ReadActionPerCallUs:   perUs(readNs, getCalls),
	}
}
