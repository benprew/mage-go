# Counter System Refactor

## Problem

`CounterType` is an `int` enum that requires editing `core/counter.go` every time a
card uses a new named counter (Mire, Wind, Fade, Age, Verse, Mining, Lore, ...).
MTG has dozens of named counter types across sets, and only a handful modify P/T.
The enum also conflates two unrelated concerns: P/T-modifying counters (which need
structured power/toughness data) and named counters (which are just a label on a
bucket).

## Design

Replace the single `CounterType int` enum with a `CounterType` **interface** and
two concrete implementations:

```go
// CounterType is anything that can sit on a permanent as a counter.
type CounterType interface {
    String() string
    PowerBoost() int
    ToughnessBoost() int
}
```

### Named counters: `Counter(name)`

A plain string. Cards say `Counter("Wind")`, `Counter("Mire")`, `Counter("Fade")`,
etc. No registration, no enum entry — just use it.

```go
type Counter string

func (c Counter) String() string      { return string(c) }
func (c Counter) PowerBoost() int     { return 0 }
func (c Counter) ToughnessBoost() int { return 0 }
```

Pre-declare constants for common named counters so existing code doesn't churn:

```go
const (
    Loyalty Counter = "Loyalty"
    Charge  Counter = "Charge"
)
```

Cards introduce new ones inline with zero ceremony:

```go
perm.AddCounter(Counter("Wind"), 1)   // Cyclone
perm.AddCounter(Counter("Mire"), 1)   // Cyclopean Tomb
perm.AddCounter(Counter("Fade"), 1)   // Blastoderm (future)
```

### P/T counters: `PTCounter(p, t)`

A struct carrying the power and toughness deltas. Covers the full matrix of MTG
P/T counters: +1/+1, -1/-1, +1/+0, +0/+1, +2/+0, -0/-1, +2/+2, etc.

```go
type ptCounter struct {
    power, toughness int
}

func PTCounter(power, toughness int) ptCounter {
    return ptCounter{power, toughness}
}

func (c ptCounter) String() string      { return fmt.Sprintf("%+d/%+d", c.power, c.toughness) }
func (c ptCounter) PowerBoost() int     { return c.power }
func (c ptCounter) ToughnessBoost() int { return c.toughness }
```

Pre-declare constants for the common ones:

```go
var (
    P1P1 = PTCounter(1, 1)    // +1/+1
    M1M1 = PTCounter(-1, -1)  // -1/-1
    P1P0 = PTCounter(1, 0)    // +1/+0
)
```

Any exotic P/T counter works without a new constant:

```go
perm.AddCounter(PTCounter(-2, -1), 1)  // hypothetical
perm.AddCounter(PTCounter(0, 1), 1)    // +0/+1
```

## Map key problem

Interfaces can't be map keys unless the underlying types are comparable. Both
`Counter` (a `string`) and `ptCounter` (two `int`s) are comparable, so in principle
`map[CounterType]int` works at runtime — but the Go compiler won't allow an
interface-keyed map unless you use `any` or a concrete key.

**Solution: string-keyed map.**

```go
type Permanent struct {
    Counters map[string]int   // keyed by ct.String()
    // ...
}
```

The `AddCounter` / `RemoveCounter` methods accept `CounterType` and use
`ct.String()` as the map key:

```go
func (p *Permanent) AddCounter(ct CounterType, n int) {
    p.Counters[ct.String()] += n
}

func (p *Permanent) RemoveCounter(ct CounterType, n int) bool {
    key := ct.String()
    if p.Counters[key] < n {
        return false
    }
    p.Counters[key] -= n
    if p.Counters[key] == 0 {
        delete(p.Counters, key)
    }
    return true
}
```

### P/T calculation

`CurrentPower` / `CurrentToughness` can no longer call `ct.PowerBoost()` on a
string key. Instead, use a helper that parses P/T boost from the key string.
The format is deterministic (`%+d/%+d`) so this is reliable:

```go
// parsePTBoost extracts power/toughness deltas from a counter key string.
// Returns (0, 0) for non-P/T counters (named counters like "Wind").
func parsePTBoost(key string) (int, int) {
    var p, t int
    if n, _ := fmt.Sscanf(key, "%d/%d", &p, &t); n == 2 {
        return p, t
    }
    return 0, 0
}
```

Then `CurrentPower`/`CurrentToughness` iterate the string map:

```go
for key, n := range p.Counters {
    pw, _ := parsePTBoost(key)
    total += pw * n
}
```

Same change applies to `permPower`/`permToughness` in `interactive/evaluator.go`.

### Counter annihilation (rule 122.3)

+1/+1 and -1/-1 counters annihilate as a state-based action. With string keys
this is straightforward — check for both `"+1/+1"` and `"-1/-1"` in the map and
remove pairs. This is a **new feature** enabled by the refactor but can be a
follow-up.

## Migration plan

### Phase 1: New types in `core/counter.go`

Replace the entire file. New contents:

- `CounterType` interface
- `Counter` string type + `Loyalty`/`Charge` constants
- `ptCounter` struct + `PTCounter()` constructor + `P1P1`/`M1M1`/`P1P0` vars
- `parsePTBoost()` helper
- Delete the old `CounterType int` enum, `String()`, `PowerBoost()`,
  `ToughnessBoost()` switch statements

**Backwards-compatible names**: `P1P1`, `M1M1`, `P1P0`, `Loyalty`, `Charge` all
keep the same identifier names. `Wind` and `Mire` move from constants to inline
`Counter("Wind")` / `Counter("Mire")` at card sites.

### Phase 2: Update `Permanent` storage (`card.go`)

- Change `Counters map[CounterType]int` → `Counters map[string]int`
- Update `NewPermanent()` initialization
- Rewrite `AddCounter(ct CounterType, n int)` to key by `ct.String()`
- Rewrite `RemoveCounter(ct CounterType, n int)` to key by `ct.String()`
- Rewrite `CurrentPower()` and `CurrentToughness()` to use `parsePTBoost()`

### Phase 3: Update effects and costs

These all pass `CounterType` through — signatures stay the same but the underlying
type is now the interface:

- `effect_combat.go`: `addCountersEffect`, `removeCountersFromSourceEffect`,
  `markDestroyAtEOTAfterNActivationsEffect` — the `ct` field type changes from
  the old int enum to the `CounterType` interface. No API changes at call sites.
- `cost.go`: `removeCountersCost` — same: `ct CounterType` is now the interface.
  `CanPay` changes from `p.Counters[c.ct]` to `p.Counters[c.ct.String()]`.
- `ability.go`: `EntersWithXCountersAbility` — same field type change.
- `game.go`: The ETB counter placement (`perm.AddCounter(xc.CounterType, ...)`)
  works unchanged since `AddCounter` still takes `CounterType`.

### Phase 4: Update card implementations

- `cards/arabian/enchantments.go` (Cyclone): `Wind` → `Counter("Wind")`,
  `perm.Counters[Wind]` → `perm.Counters[Counter("Wind").String()]` (or just
  `perm.Counters["Wind"]` since `Counter.String()` returns the string verbatim)
- `cards/arabian/enchantments.go` (Unstable Mutation): `M1M1` — no change,
  constant kept
- `cards/limited/artifacts.go` (Cyclopean Tomb): `Mire` → `Counter("Mire")`
- `cards/limited/creatures.go`: All `P1P1`, `M1M1`, `P1P0` usages — no change,
  constants kept
- `cards/custom/wraithbloom.go`: `P1P1` — no change
- Any direct `perm.Counters[SomeType]` reads need to use string keys instead

### Phase 5: Update snapshot and AI evaluator

- `interactive/snapshot.go`: `ct.String()` call goes away — the key already is
  a string. Simplifies to `permState.Counters = perm.Counters` (or a copy).
- `interactive/evaluator.go`: `permPower` / `permToughness` iterate string-keyed
  map using `parsePTBoost()` instead of `ct.PowerBoost()` / `ct.ToughnessBoost()`.

### Phase 6: Update test harness and tests

- `gametest/harness.go`: `AddCounters` and `AssertCounterCount` accept
  `core.CounterType` (now the interface). `AssertCounterCount` reads
  `perm.Counters[ct.String()]`.
- All test files that reference `core.P1P1`, `core.M1M1`, `core.Mire`,
  `core.Wind` etc. — `P1P1`/`M1M1`/`P1P0`/`Charge`/`Loyalty` unchanged.
  `core.Mire` → `core.Counter("Mire")`, `core.Wind` → `core.Counter("Wind")`.

## Call-site examples (before → after)

```go
// BEFORE                                    // AFTER
core.P1P1                                    core.P1P1              // same
core.Wind                                    core.Counter("Wind")   // no enum entry needed
core.Mire                                    core.Counter("Mire")
perm.Counters[Wind]                          perm.Counters["Wind"]
perm.AddCounter(M1M1, 1)                     perm.AddCounter(M1M1, 1)        // same
RemoveCountersCost(P1P1, 3)                  RemoveCountersCost(P1P1, 3)     // same
EntersWithXCounters(P1P1)                    EntersWithXCounters(P1P1)       // same
AddCounters(P1P1, Fixed(1), SelectSource)    AddCounters(P1P1, Fixed(1), SelectSource) // same

// New card with exotic counters — zero core changes:
perm.AddCounter(core.Counter("Verse"), 1)
perm.AddCounter(core.PTCounter(0, 1), 2)     // +0/+1 counters
if perm.Counters["Verse"] >= 3 { ... }
```

## Risk and tradeoffs

**String key parsing for P/T**: `parsePTBoost` is slightly less type-safe than
method dispatch. Mitigated by deterministic format from `PTCounter.String()` and
the fact that only the engine's P/T calculation calls it.

**Interface in struct fields**: Effects/costs store `CounterType` (interface) instead
of a concrete type. Tiny allocation overhead for the interface wrapper; irrelevant
for a game engine.

**Direct map reads use strings**: Card code that reads counters directly
(e.g. Cyclone's `perm.Counters[Wind]`) changes to `perm.Counters["Wind"]`. Slightly
less discoverable but more honest — it's just a bag of labeled counters.

## Follow-up opportunities

- **Rule 122.3 annihilation**: SBA that removes matching +1/+1 and -1/-1 pairs.
  Trivial with string keys: `min(counters["+1/+1"], counters["-1/-1"])`.
- **Counter events**: Fire `EvtCounterAdded` / `EvtCounterRemoved` from
  `AddCounter` / `RemoveCounter` to support "whenever a counter is placed" triggers
  (e.g. Doubling Season, Hardened Scales in future sets).
- **HasCounter filter**: `HasCounter(ct CounterType) PermanentFilter` for targeting
  and queries.
