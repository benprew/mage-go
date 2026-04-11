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

## Second optimization round: Counters, slab alloc, CoW abilities

Three independent changes targeting the `clonePermanent` self-cost that
dominated after the attr-array work.

### Counters map → fixed array

`Permanent.Counters` changed from `map[CounterType]int` to
`[NumCounters]int` (`pkg/mage/core/counter.go`, `pkg/mage/card.go`).
`NumCounters` is the new terminal enum member, same pattern as
`NumAttrs`. `clonePermanent` no longer allocates a map per permanent;
the array lives inline in the struct so cloning it is a memcpy.

- `delete(p.Counters, ct)` rewritten as `p.Counters[ct] = 0`
- `for ct, n := range p.Counters` rewritten as an indexed loop over
  `[0, NumCounters)` that skips zeros
- Snapshot/eval hot paths updated for the new access pattern

`BenchmarkClone`: 91 → 67 allocs/op (−26%), 39 μs → 30 μs (−23%).

### Slab allocation for cloned permanents

`Game.Clone` now allocates the entire battlefield's permanents as a
single `make([]Permanent, N)` slab and hands out pointers into it
(`clone.go:40`). The prior loop allocated N individual `&Permanent{}`
structs. All Permanent allocations are now attributed to `Game.Clone`
flat in alloc profiles.

`BenchmarkClone`: 67 → 55 allocs/op (−18%).

### RuntimeAbilities copy-on-write

`clonePermanentInto` shares `src.RuntimeAbilities` with the clone via
the three-index slice form `src.RuntimeAbilities[:n:n]`
(`clone.go:230`). All current mutators of that slice are either
`= nil`, `= freshSlice`, or `append(...)` — no index writes — so
`cap == len` sharing is safe. `append` on either side reallocates
because `cap == len`; the old backing array stays intact for the
other side.

Profile impact (cum CPU share of the 10-game `search` vs `search`
run, different wall-clocks so ratios are the comparison):

| site                                      | pre-slab | +slab  | +slab+CoW |
|-------------------------------------------|----------|--------|-----------|
| `Game.Clone` cum                          | 15.34%   | 14.07% | **12.91%**|
| `clonePermanent(Into)` cum                | 6.21%    | 4.30%  | **3.45%** |
| `runtime.mallocgc` cum                    | 21.05%   | 19.52% | **17.94%**|
| `RuntimeAbilities` make+copy line cum     | 2.43s    | 6.22s  | **1.18s** |

The RuntimeAbilities line dropped 81% in absolute time. `Game.Clone`
cum share dropped from 15.34% → 12.91% across the three changes.

## Remaining hot spots (as of slab+CoW round)

Biggest lines inside `clonePermanentInto` now:

| line                                        | cum  |
|---------------------------------------------|------|
| `*dst = Permanent{...}` (~400 B memcpy)     | 6.46s|
| `Counters` array copy (within struct lit)   | 2.26s|
| `grantedAttrs` array copy (within struct)   | 2.36s|
| `Attachments` make+copy                     | 0.99s|
| `RuntimeAbilities[:n:n]`                    | 1.18s|

The Permanent value copy itself is now the dominant line — it's
inflated by the two inline fixed arrays (`Counters`, attr arrays).
Can't shrink without retyping those fields to narrower ints.

Biggest allocations outside clonePermanentInto:

| site                                          | cum    |
|-----------------------------------------------|--------|
| `cloneBasePlayer.library = cloneCardSlice(...)` | 3.78s  |
| `cloneBasePlayer.graveyard = cloneCardSlice(...)` | 1.55s  |
| `cloneBasePlayer.hand = cloneCardSlice(...)`    | 1.31s  |
| `FilterBattlefield` self                        | 3.30s (14.49s cum) |
| `countUntappedManaSources` cum                  | 12.19s |
| `GeneratePriorityMoves` cum                     | 10.71s |

CPU-side, the leaf evaluator (`WeightedEvaluator` + quiescence) still
accounts for ~60% of search time. Eval scans the battlefield on every
leaf, so any battlefield-scan cost multiplies at leaf nodes.

## TODO — next optimization targets

In rough priority order.

### 1. Library CoW in `cloneBasePlayer` — biggest individual remaining target

`cloneBasePlayer.library` copy is 3.78s cum in the profile. The
library's mutators are:

- `AddToLibrary`: `append(...)` — safe with `cap == len`
- `DrawCard`: `library = library[1:]` — header only, safe
- `ShuffleLibrary`: in-place swap — **unsafe**, would corrupt shared
  memory

Change: share via `library[:n:n]` in `cloneBasePlayer`; make
`ShuffleLibrary` copy before shuffling. Shuffles are rare in search
(only on specific card effects), so the copy cost is negligible.

Expected win: eliminate most of the 3.78s library cum, partial
relief for hand/graveyard via similar treatment (see below).

File: `pkg/mage/clone.go:179`, `pkg/mage/player.go:216`.

### 2. Hand/graveyard/ante CoW — trickier, do after library

Same `[:n:n]` idea, but `RemoveFromHand` / `RemoveFromGraveyard` /
`RemoveFromAnte` all use `append(slice[:i], slice[i+1:]...)` which
writes the shift-down in place. That corrupts shared backing arrays.

Options:

- Rewrite `RemoveFromX` to allocate a new slice (`make([]Card, len-1)`
  + two copies). Costs one alloc per removal but removals are much
  rarer than clones.
- Or track a `shared` bit per zone and copy before mutation.

Prefer the rewrite — simpler, no bookkeeping. Measure after library
CoW to see if this is still worth it.

File: `pkg/mage/player.go:152,178,206`.

### 3. Shrink `Permanent` struct width

The inline `Counters` array is `[NumCounters]int` (~264 B on 64-bit).
Narrowing to `int16` cuts it to 66 B and shrinks the memcpy line. The
tradeoff is a type conversion at every `PowerBoost() * n` site, but
that's just a cast.

Similarly, `Attachments []uuid.UUID` allocates whenever non-empty.
Most permanents have 0 attachments — but we already skip cloning
empty. When non-empty they're 1-2 entries; an inline `[2]uuid.UUID`
with a length byte could eliminate the allocation entirely.

Lower priority than the library CoW, but small changes.

### 4. `FilterBattlefield` scratch buffer

3.30s self CPU / 14.49s cum. Called from eval hot paths; builds a new
slice every call. Evaluator-owned reusable scratch buffer drops the
allocations. Not zero-cost because of the per-call size reset, but
cheap.

File: `pkg/mage/game.go` (search for `FilterBattlefield`).

### 5. `countUntappedManaSources` allocation

12.19s cum. Probably building transient slices per call. Look for
obvious scratch-buffer opportunities before more exotic changes.

File: `pkg/mage/interactive/eval/eval.go`.

### 6. `sync.Pool[*Permanent]`

Previously suggested; the slab change captured the easy win. Pool
would further reduce struct-memcpy cost if it let us skip the memclr
on reuse (since pool objects retain their old fields that we
overwrite). Probably not worth the lifetime-management complexity
until the other items are done.

### 7. Transposition table + Zobrist hashing

Still the highest-leverage structural change — a TT hit skips the
clone + eval of an entire subtree. Referenced from the first
optimization round. Work to do the fingerprinting is significant; do
it after the cheap wins above.

### 8. Eval caching via TT

Once Zobrist is in place (item 7), cache eval scores keyed by hash
independently of the TT's alpha/beta bounds. Leaf evaluation on
repeated positions becomes a hash lookup.

### 9. Make/unmake instead of clone

The nuclear option. Applies moves in place and undoes them on
backtrack, avoiding clone entirely. Engineering cost is high in an MTG
engine because every event, triggered ability, and state-based action
would need an inverse. Probably not worth it unless the clone cost
persists after all of the above.

## Reproducing the profile

```
go build -o gametest ./cmd/gametest/
./gametest -mode-a search -mode-b search -quiet -games 10 -seed 1 \
    -cpuprofile /tmp/cpu.pprof -memprofile /tmp/mem.pprof
go tool pprof -top -cum /tmp/cpu.pprof
go tool pprof -alloc_space -top /tmp/mem.pprof
go tool pprof -list clonePermanentInto /tmp/cpu.pprof
```

`-games N` runs N games back-to-back so profile samples aggregate.
`-seed` is only used for deck selection; mulligans and in-game card
draws still use the global RNG, so the same seed does not reproduce
identical wall-clock numbers. Use the clone microbenchmark
(`go test -bench=BenchmarkClone -benchmem ./pkg/mage/`) for
apples-to-apples comparisons of specific changes — but note that the
bench permanents are vanilla creatures, so it won't exercise code
paths like `RuntimeAbilities` cloning.
