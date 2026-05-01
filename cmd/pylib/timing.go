package main

import (
	"sync/atomic"
	"time"
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

var (
	packedTimingCalls      int64
	packedTimingTotalNs    int64
	packedTimingGameNs     int64
	packedTimingRenderNs   int64
	packedTimingAssemblyNs int64
	packedTimingMetadataNs int64
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
