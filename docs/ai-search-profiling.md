# AI Search Profiling & Optimization Notes

Baseline notes from profiling the minimax search AI in `cmd/gametest`
(`-mode-a search -mode-b search`). These numbers are a reference point
for future optimization work, not a static specification.

## First optimization: Permanent attr storage

### Problem

Each `*Permanent` held two `map[Attr]int` fields (`baseAttrs`,
`grantedAttrs`) tracking counts for additive/subtractive composition.
During minimax search we clone the entire game state per move tried,
and `cloneAttrMap` was allocating a fresh map for each of the two
fields on every permanent in the clone.

### Fix

Replaced both fields with `[NumAttrs]int8` fixed arrays
(`pkg/mage/core/attr.go`, `pkg/mage/card.go`). Array assignment is a
memcpy, so cloning is zero-allocation for these fields. `HasAttr`
becomes direct array indexing instead of two map probes.

### Results

`BenchmarkClone` (stable, 5s, best of 3):

| metric      | before    | after     | delta |
|-------------|-----------|-----------|-------|
| ns per op   | 58,000    | 40,000    | −31%  |
| bytes/op    | 12,552    | 10,680    | −15%  |
| allocs/op   | 130       | 91        | −30%  |

Whole-program profile (search vs search, 30-turn game):

| metric                 | before  | after   | delta |
|------------------------|---------|---------|-------|
| total heap alloc       | 405 MB  | 200 MB  | −51%  |
| `Game.Clone` alloc cum | 240 MB  | 105 MB  | −56%  |
| `cloneAttrMap` alloc   | 59 MB   | 0       | −100% |
| GC share of CPU        | ~45%    | ~18%    | large |

## Remaining hot spots

From the post-fix alloc profile. Allocation counts are per 30-turn
search-vs-search game.

| site                                          | alloc | notes                                           |
|-----------------------------------------------|-------|-------------------------------------------------|
| `clonePermanent` self                         | 52 MB | struct, counters map, abilities/attachments     |
| `GeneratePriorityMoves`                       | 26 MB | move list construction per priority window     |
| `cloneCardSlice` (hand/library/graveyard)     | 17 MB | per-player zone copies                          |
| `FilterBattlefield`                           | 11 MB | called per eval; rebuilds a filtered slice     |
| `getUntappedManaSources`                      | 10 MB | eval + targeting checks                         |
| `countRoles` / `countUntappedManaSources`     | 19 MB | eval helpers scanning the battlefield           |

CPU-side, the leaf evaluator (`WeightedEvaluator` + quiescence) still
accounts for ~60% of search time. Eval scans the battlefield on every
leaf, so any battlefield-scan cost multiplies at leaf nodes.

## Recommended further optimization

In rough priority order. None of these is started; all assume the
attr-array work as a baseline.

1. **Transposition table + Zobrist hashing.** See
   `~/.claude/plans/zesty-coalescing-fox.md`. A TT hit skips the
   clone + eval of an entire subtree, which attacks the two largest
   remaining cost centers simultaneously. Position identity fingerprints
   are cheap (~30 XORs) and persist across `PriorityAction` calls so
   different move orders that reach the same board state reuse work.

2. **`sync.Pool` for `*Permanent`.** `clonePermanent` self-allocs are
   still 52 MB per game. A pool keyed by nothing (just a reusable
   `*Permanent`) would absorb most of that. The tricky part is
   lifetime — permanents are referenced from `Game.Battlefield`,
   `Combat.Groups`, continuous effects, and stack targets. A pool
   release needs to happen only when the *entire cloned `Game`* is
   discarded, not when a single permanent leaves the battlefield.
   Simplest approach: put the pool on the search strategy and release
   all permanents from the discarded clone tree at the end of each
   root search.

3. **Replace `Counters map[CounterType]int` with a fixed array.**
   Same pattern as attrs. `CounterType` is a small finite enum; a
   `[numCounters]int8` eliminates another map-per-permanent on clone.
   Lower impact than attrs because counters are rarer, but still
   zero-allocation for the common case.

4. **Eval caching via TT.** Once Zobrist is in place, cache the eval
   score keyed by hash independently of the TT's alpha/beta bounds.
   Leaf evaluation on repeated positions becomes a hash lookup.

5. **`FilterBattlefield` allocation reuse.** Eval calls it repeatedly
   with similar filters; returning a slice scratch buffer owned by the
   evaluator would drop 11 MB/game. Lower priority — only worthwhile
   after bigger wins.

6. **Make/unmake instead of clone.** The nuclear option. Applies
   moves in place and undoes them on backtrack, avoiding clone
   entirely. Engineering cost is high in an MTG engine because every
   event, triggered ability, and state-based action would need an
   inverse. Probably not worth it unless the clone cost persists after
   the pool work.

## Reproducing the profile

```
go build -o gametest ./cmd/gametest/
./gametest -mode-a search -mode-b search -quiet -turns 30 \
    -cpuprofile /tmp/cpu.pprof -memprofile /tmp/mem.pprof
go tool pprof -top -cum /tmp/cpu.pprof
go tool pprof -alloc_space -top /tmp/mem.pprof
```

Games are non-deterministic (shuffle, mulligan), so single-run wall
times vary widely (0.3s–8s for 30 turns). Use the clone benchmark
(`go test -bench=BenchmarkClone -benchmem ./pkg/mage/`) for apples-to-
apples comparisons of specific changes.
