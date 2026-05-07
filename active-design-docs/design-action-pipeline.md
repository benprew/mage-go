# Design: Action Pipeline & Message-Passing Engine

## Implementation Status

| Phase | Status | Notes |
|---|---|---|
| Phase 1: Action type system | DONE | 5 action types: `DamageToPlayerAction`, `DamageToCreatureAction`, `DestroyPermanentAction`, `LifeGainAction`, `DrawCardAction` in `action.go`, `action_damage.go`, `action_destroy.go`, `action_life.go`, `action_draw.go` |
| Phase 2: Pipeline core + replacement interface | DONE | `ReplacementEffect` interface in `action.go`, `ApplyReplacements` loop in `effect_manager.go`, two-list registration (persistent + cycle) |
| Phase 3: GameMutator bridge | DONE | `DealDamageToPlayer`, `DealDamageToPermanent`, `DestroyPermanent`, `PlayerGainLife`, `doDrawNormalDraw` all create actions and run through pipeline. `executeAction` dispatcher routes by type. |
| Phase 4: Migrate replacement effects | DONE | All 17 ad-hoc mechanisms migrated to `ReplacementEffect` implementations in `replacement.go`. DamageSystem gutted to ~140 lines (reflection only). |
| Phase 5: Migrate effects to EffectContext | NOT YET | Effects still use `Apply(GameMutator, ...)` signature. No goroutine-per-effect or channel-based communication yet. |
| Phase 6: Continuous effects in pipeline | NOT YET | Continuous effects still write directly to Permanent fields. No `CharacteristicAction` or `Emit` method yet. |

### Deviations from Original Design

- **Replace takes GameMutator, not GameReader.** The design doc specifies `Replace(Action, GameReader) Action`, but the implementation uses `Replace(Action, GameMutator) Action`. Some replacements (regeneration, Lich life gain, draw replacement) need to perform mutations during replacement (tap permanent, draw cards, choose from library).
- **No ActionResult.** Effects do not receive results from mutations. The pipeline is synchronous and inline — mutations still happen imperatively via `GameMutator` methods that internally create actions and run the pipeline.
- **No EffectContext / goroutines.** The pipeline runs synchronously in the calling goroutine. No channels or actor model yet.
- **No GameLog / structured logging.** Actions are not logged. This is deferred to a future phase.
- **Simplified action types.** Only 5 action types (damage-to-player, damage-to-creature, destroy, life-gain, draw) instead of the 12 category interfaces in the design. Zone changes, counters, tap state, characteristics, mana, tokens, combat, and game-rule mutations do not flow through the pipeline yet.
- **Minimum life fallback.** Ali from Cairo's continuous effect sets a flag on GameRules directly (`g.Effects.Rules.SetMinimumLife`), and `executeDamageToPlayer` checks it as a fallback in addition to the replacement pipeline. This avoids requiring all continuous effects to use the proxy method.

## Motivation

The engine currently has ~158 distinct mutation operations spread across
`GameMutator` methods, direct `*Permanent` field writes, and `Player` method
calls. Effects mutate game state imperatively and return nothing. This creates
three problems:

1. **Replacement effects are ad-hoc.** Each "if X would happen, do Y instead"
   pattern is a separate flag/shield system hard-coded into `game.go` methods
   (Lich flag, regeneration shields, damage prevention shields, forcefield
   shields, color prevention, damage redirects, minimum life). Adding a new
   replacement effect means touching engine internals. The MTG rules define a
   generic replacement system; we should have one.

2. **No mutation log.** We can't produce structured game traces for LLM
   training, replay, or debugging. Effects are fire-and-forget.

3. **Player choices are synchronous callbacks.** Effects call `Player.ChooseX()`
   inline during `Apply()`. This couples effect resolution to the player
   interface and makes it impossible to serialize/replay choice points.

The redesign introduces an **action pipeline**: effects produce typed actions,
the engine intercepts/applies/logs them. Effects run as goroutines communicating
with the engine via channels (actor model).

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Effect execution | Goroutine per effect, channels to engine | Natural Go concurrency; effects block on choices without complex continuation state |
| Action dispatch | Interface per category (~12 interfaces) | Type-safe matching for replacement effects; no giant discriminated union |
| Ack model | Synchronous by default, async opt-in | Effects see results of replacements (e.g., actual damage dealt); consistent state reads |
| State reads | Live `GameReader` | Per MTG rules, SBAs don't run mid-resolution and triggers queue; sync ack guarantees prior actions are applied |
| Continuous effects | Same pipeline | Characteristic mutations flow through replacement effects and logging |
| Replacement effects | `Matches(Action) bool` + `Replace(Action, GameReader) Action` | Generic system replaces all ad-hoc shields/flags |
| Logging | Low-level actions nested under resolution contexts | Both granularities for LLM training |
| Migration | Incremental — `GameMutator` becomes bridge | Existing effects keep working; convert one-by-one |

---

## Action Type System

### Base Interface

```go
// Action is the unit of game state mutation. Every mutation flows through
// the engine's action pipeline as a concrete Action value.
type Action interface {
    // ActionSource returns the ID of the permanent/spell that produced this action.
    ActionSource() uuid.UUID
}
```

### Category Interfaces (~12)

Each category is a small interface embedding `Action`. Replacement effects
match on these interfaces via type assertion. Concrete structs implement one
category interface each.

```go
// --- Damage ---

type DamageAction interface {
    Action
    TargetID() uuid.UUID      // permanent or player
    Amount() int
    IsCombatDamage() bool
    IsToPlayer() bool
}

// Concrete: dealDamageToPlayer, dealDamageToPermanent, dealDamageToAll


// --- Life ---

type LifeAction interface {
    Action
    PlayerID() uuid.UUID
    Amount() int              // positive = gain, negative = loss
}

// Concrete: gainLife, loseLife


// --- Zone Change ---

type ZoneChangeAction interface {
    Action
    ObjectID() uuid.UUID      // card or permanent being moved
    From() Zone
    To() Zone
    OwnerID() uuid.UUID
}

// Concrete: drawCard, discard, sacrifice, destroy, exile, bounce,
//           returnFromGraveyard, putOnBattlefield, moveToLibrary


// --- Destroy ---
// Separate from ZoneChange because replacement effects specifically
// care about "would be destroyed" (regeneration, indestructible).

type DestroyAction interface {
    Action
    PermanentID() uuid.UUID
}

// Concrete: destroyPermanent
// Regeneration replaces this. Indestructible prevents this.
// If not replaced, engine converts to ZoneChangeAction (battlefield → graveyard).


// --- Counter Change ---

type CounterAction interface {
    Action
    PermanentID() uuid.UUID
    CounterType() CounterType
    Delta() int               // positive = add, negative = remove
}

// Concrete: addCounters, removeCounters


// --- Tap State ---

type TapAction interface {
    Action
    PermanentID() uuid.UUID
    Tap() bool                // true = tap, false = untap
}

// Concrete: tap, untap


// --- Characteristic ---
// Used by continuous effects and one-shot pumps.

type CharacteristicAction interface {
    Action
    PermanentID() uuid.UUID
    Layer() Layer
}

// Concrete: boostPT, setBasePT, grantKeyword, revokeKeyword,
//           overrideColor, overrideSubtypes, changeController


// --- Stack ---

type StackAction interface {
    Action
    TargetStackID() uuid.UUID // stack object being affected
}

// Concrete: counterSpell, copySpell


// --- Mana ---

type ManaAction interface {
    Action
    PlayerID() uuid.UUID
    Color() Color
    Amount() int
}

// Concrete: addMana


// --- Token ---

type TokenAction interface {
    Action
    Controller() uuid.UUID
    Name() string
    Power() int
    Toughness() int
    Types() []CardType
    SubTypes() []string
    Keywords() []Keyword
}

// Concrete: createToken


// --- Combat ---

type CombatAction interface {
    Action
    PermanentID() uuid.UUID
}

// Concrete: removeFromCombat, makeUnblockable


// --- Game Rule ---
// Catch-all for game-rule mutations (extra turns, delayed triggers,
// special flags). These are less common and less likely to need
// replacement, but still flow through logging.

type GameRuleAction interface {
    Action
    Description() string
}

// Concrete: grantExtraTurn, registerDelayedTrigger, skipNextDraw,
//           setMinimumLife, setLichActive, etc.
```

### ActionResult

Every synchronous action returns a result to the effect goroutine:

```go
type ActionResult struct {
    // Applied is false if the action was fully prevented/replaced with nil.
    Applied bool

    // ActualAmount reflects the value after replacement effects.
    // For DamageAction: actual damage dealt (after prevention).
    // For LifeAction: actual life gained/lost.
    // For CounterAction: actual counters added/removed.
    // Zero if not applicable.
    ActualAmount int

    // Replaced is true if any replacement effect modified or prevented
    // this action.
    Replaced bool
}
```

---

## EffectContext (Effect ↔ Engine Communication)

Effects run in their own goroutine. Instead of calling `GameMutator` methods,
they call methods on `EffectContext`, which sends actions through a channel and
blocks until the engine applies them.

```go
type EffectContext struct {
    source     uuid.UUID
    controller uuid.UUID
    targets    []uuid.UUID

    // Channel to engine — unbuffered; each send blocks until engine processes.
    pipe       chan<- actionEnvelope

    // Read-only view of current game state.
    // Safe because sync ack ensures state reflects all prior actions.
    reader     GameReader
}

type actionEnvelope struct {
    action Action
    ack    chan<- ActionResult  // nil for fire-and-forget
}
```

### Synchronous helpers (default)

```go
func (ctx *EffectContext) DealDamage(targetID uuid.UUID, amount int) ActionResult {
    return ctx.send(&dealDamageToTarget{
        source:   ctx.source,
        targetID: targetID,
        amount:   amount,
    })
}

func (ctx *EffectContext) DestroyPermanent(permID uuid.UUID) ActionResult {
    return ctx.send(&destroyPermanent{
        source: ctx.source,
        permID: permID,
    })
}

func (ctx *EffectContext) GainLife(playerID uuid.UUID, amount int) ActionResult {
    return ctx.send(&gainLife{
        source:   ctx.source,
        playerID: playerID,
        amount:   amount,
    })
}

// ... etc for each action category

func (ctx *EffectContext) send(a Action) ActionResult {
    ack := make(chan ActionResult, 1)
    ctx.pipe <- actionEnvelope{action: a, ack: ack}
    return <-ack
}
```

### Async helpers (fire-and-forget opt-in)

For actions where the effect doesn't need the result (e.g., tapping a permanent
as a side effect):

```go
func (ctx *EffectContext) TapAsync(permID uuid.UUID) {
    ctx.pipe <- actionEnvelope{
        action: &tap{source: ctx.source, permID: permID, tapState: true},
        ack:    nil, // no ack channel
    }
}
```

### Player choice requests

Player choices are a special action category. The effect goroutine blocks until
the player responds:

```go
func (ctx *EffectContext) ChooseCardsFromHand(
    playerID uuid.UUID, amount int, reason string,
) []Card {
    result := ctx.send(&chooseCardsRequest{
        source:   ctx.source,
        playerID: playerID,
        amount:   amount,
        reason:   reason,
    })
    return result.ChosenCards
}

func (ctx *EffectContext) ChooseMode(modes []string, reason string) int {
    result := ctx.send(&chooseModeRequest{
        source: ctx.source,
        modes:  modes,
        reason: reason,
    })
    return result.ChosenIndex
}

func (ctx *EffectContext) ChoosePermanent(
    candidates []uuid.UUID, reason string,
) uuid.UUID {
    result := ctx.send(&choosePermanentRequest{
        source:     ctx.source,
        candidates: candidates,
        reason:     reason,
    })
    return result.ChosenID
}

func (ctx *EffectContext) ChooseManaColor(reason string) Color {
    result := ctx.send(&chooseColorRequest{
        source: ctx.source,
        reason: reason,
    })
    return result.ChosenColor
}
```

The engine routes these to the appropriate `Player` implementation (which may be
a `TestPlayer`, `AIPlayer`, or `HumanPlayer` with channel-based I/O).

The `ActionResult` type is extended for choice responses:

```go
type ActionResult struct {
    Applied      bool
    ActualAmount int
    Replaced     bool

    // Choice results (populated for choice actions only)
    ChosenCards  []Card
    ChosenIndex  int
    ChosenID     uuid.UUID
    ChosenColor  Color
}
```

---

## Engine Action Loop

The engine runs the action processing loop while an effect goroutine is active:

```go
func (g *Game) processEffect(ctx *EffectContext, fn func(*EffectContext)) {
    done := make(chan struct{})

    // Effect runs in its own goroutine
    go func() {
        defer close(done)
        fn(ctx)
    }()

    // Engine loop: receive actions until effect goroutine exits
    for {
        select {
        case envelope, ok := <-ctx.pipe:
            if !ok {
                return // effect closed the channel (shouldn't happen; use done)
            }
            result := g.processAction(envelope.action)
            if envelope.ack != nil {
                envelope.ack <- result
            }
        case <-done:
            // Effect goroutine finished. Drain any remaining actions.
            g.drainPipe(ctx.pipe)
            return
        }
    }
}
```

### processAction: the core pipeline

```go
func (g *Game) processAction(action Action) ActionResult {
    original := action

    // 1. Run replacement effects
    action, replaced := g.applyReplacements(action)
    if action == nil {
        // Fully prevented (e.g., damage prevention replaced to nil)
        g.log.RecordPrevented(g.currentResolution, original)
        return ActionResult{Applied: false, Replaced: true}
    }

    // 2. Apply the mutation to game state
    actualAmount := g.applyMutation(action)

    // 3. Log
    g.log.RecordAction(g.currentResolution, action, actualAmount, replaced)

    // 4. Fire events for triggered abilities (queued, not resolved)
    g.queueTriggeredAbilities(action)

    // 5. Return result to effect goroutine
    return ActionResult{
        Applied:      true,
        ActualAmount: actualAmount,
        Replaced:     replaced,
    }
}
```

### applyMutation: type-switch dispatch

```go
func (g *Game) applyMutation(action Action) int {
    switch a := action.(type) {
    case DamageAction:
        return g.applyDamage(a)
    case LifeAction:
        return g.applyLifeChange(a)
    case DestroyAction:
        return g.applyDestroy(a)
    case ZoneChangeAction:
        return g.applyZoneChange(a)
    case CounterAction:
        return g.applyCounterChange(a)
    case TapAction:
        return g.applyTap(a)
    case CharacteristicAction:
        return g.applyCharacteristic(a)
    case StackAction:
        return g.applyStackAction(a)
    case ManaAction:
        return g.applyMana(a)
    case TokenAction:
        return g.applyToken(a)
    case CombatAction:
        return g.applyCombat(a)
    case GameRuleAction:
        return g.applyGameRule(a)
    default:
        panic(fmt.Sprintf("unknown action type: %T", action))
    }
}
```

Each `applyX` method performs the actual state mutation that the current
`GameMutator` methods do today. The logic moves from `GameMutator` into these
methods.

---

## Replacement Effect System

### Interface

```go
type ReplacementEffect interface {
    // Matches returns true if this replacement applies to the given action.
    // Typically a type assertion on the action's category interface.
    Matches(action Action, g GameReader) bool

    // Replace returns a modified action, a different action, or nil to fully
    // prevent the action. The engine calls this when Matches returns true.
    //
    // A replacement may return the same action type with modified values
    // (e.g., reduce damage amount), a different action type (e.g., Lich
    // replaces life gain with card draw), or nil (e.g., prevent damage).
    Replace(action Action, g GameReader) Action

    // SourceID identifies the permanent/effect that created this replacement.
    SourceID() uuid.UUID

    // IsActive returns whether this replacement is still in effect.
    IsActive(g GameReader) bool
}
```

### Engine applies replacements in order

Per MTG rules (614.5), the affected player or controller of the affected object
chooses the order when multiple replacements apply:

```go
func (g *Game) applyReplacements(action Action) (Action, bool) {
    replaced := false
    for {
        // Find all matching active replacements
        var matching []ReplacementEffect
        for _, r := range g.replacements {
            if r.IsActive(g) && r.Matches(action, g) {
                matching = append(matching, r)
            }
        }
        if len(matching) == 0 {
            break
        }

        // If multiple apply, affected player chooses order (MTG 614.5).
        // For now: apply in registration order. TODO: player choice.
        r := matching[0]
        action = r.Replace(action, g)
        replaced = true

        if action == nil {
            return nil, true // fully prevented
        }
        // Loop: check if more replacements apply to the modified action.
        // MTG rule: each replacement applies at most once per event.
    }
    return action, replaced
}
```

### Migrating existing ad-hoc replacements

Every current shield/flag system becomes a `ReplacementEffect`:

| Current mechanism | Replacement effect |
|---|---|
| `regenerationShields` map | `RegenerationReplacement` matches `DestroyAction`, replaces with tap + remove damage + remove from combat |
| `preventionShields` map | `DamagePreventionReplacement` matches `DamageAction`, reduces `Amount()` or returns nil |
| `preventCombatDamage` flag | `FogReplacement` matches `DamageAction` where `IsCombatDamage()`, returns nil |
| `forcefieldShields` map | `ForcefieldReplacement` matches `DamageAction` where unblocked combat, caps amount to 1 |
| `colorPrevention` map | `ColorPreventionReplacement` matches `DamageAction` from source of given color, returns nil |
| `reverseDamageShields` map | `ReverseDamageReplacement` matches `DamageAction`, replaces with `LifeAction` (gain) |
| `Lich` active flag | `LichLifeGainReplacement` matches `LifeAction` (gain), replaces with draw-cards action |
| | `LichDamageReplacement` matches `DamageAction` (to Lich player), replaces with sacrifice actions |
| `minimumLife` flag | `MinimumLifeReplacement` matches `DamageAction` (to player), caps so life stays >= 1 |
| `bodyguard` / `playerDamageRedirect` maps | `DamageRedirectReplacement` matches `DamageAction` (to player), retargets to creature |
| `creatureDamageRedirect` map | `CreatureDamageRedirectReplacement` matches `DamageAction` (to creature), retargets to player |
| `damagePreventionRules` list | `FilteredPreventionReplacement` matches `DamageAction` with source/target filter, returns nil |

### Replacement that changes action type (Lich example)

```go
type LichLifeGainReplacement struct {
    playerID uuid.UUID
    sourceID uuid.UUID
}

func (r *LichLifeGainReplacement) Matches(a Action, g GameReader) bool {
    if la, ok := a.(LifeAction); ok {
        return la.PlayerID() == r.playerID && la.Amount() > 0
    }
    return false
}

func (r *LichLifeGainReplacement) Replace(a Action, g GameReader) Action {
    la := a.(LifeAction)
    // Replace "gain N life" with "draw N cards"
    return &drawCards{
        source:   la.ActionSource(),
        playerID: la.PlayerID(),
        amount:   la.Amount(),
    }
}
```

The returned `drawCards` action flows through the pipeline normally — it can
itself be matched by other replacements, gets logged, fires triggers.

### Replacement that modifies same action (damage prevention)

```go
type DamagePreventionShield struct {
    targetID  uuid.UUID
    remaining int
    sourceID  uuid.UUID
}

func (r *DamagePreventionShield) Matches(a Action, g GameReader) bool {
    if da, ok := a.(DamageAction); ok {
        return da.TargetID() == r.targetID && r.remaining > 0
    }
    return false
}

func (r *DamagePreventionShield) Replace(a Action, g GameReader) Action {
    da := a.(DamageAction)
    prevented := min(da.Amount(), r.remaining)
    r.remaining -= prevented
    newAmount := da.Amount() - prevented
    if newAmount <= 0 {
        return nil // fully prevented
    }
    // Return modified damage action with reduced amount
    return da.WithAmount(newAmount)
}
```

This requires damage action structs to have a `WithAmount` method (or the
replacement constructs a new struct). A simple approach:

```go
type dealDamageToPlayer struct {
    source   uuid.UUID
    targetID uuid.UUID
    amount   int
    combat   bool
}

func (d *dealDamageToPlayer) WithAmount(n int) DamageAction {
    cpy := *d
    cpy.amount = n
    return &cpy
}
```

---

## Logging System

### Structure

```go
type GameLog struct {
    Resolutions []ResolutionLog
}

type ResolutionLog struct {
    // High-level context
    Turn       int
    Step       PhaseStep
    Source     string    // card name
    SourceID   uuid.UUID
    Controller uuid.UUID
    Targets    []uuid.UUID

    // Low-level action trace
    Actions []ActionRecord
}

type ActionRecord struct {
    Action       Action
    ActualAmount int
    Prevented    bool      // true if fully prevented by replacement
    Replaced     bool      // true if any replacement modified it
    ReplacedFrom Action    // original action before replacement (if replaced)
}
```

### Usage

The engine sets `g.currentResolution` before processing a stack object:

```go
func (g *Game) ResolveStackObject(obj *StackObject) {
    g.currentResolution = &ResolutionLog{
        Turn:       g.TurnNumber,
        Step:       g.CurrentStep,
        Source:     obj.Card.Name(),
        SourceID:   obj.SourceID,
        Controller: obj.Controller,
        Targets:    obj.Targets,
    }
    defer func() {
        g.log.Resolutions = append(g.log.Resolutions, *g.currentResolution)
        g.currentResolution = nil
    }()

    // ... run effect goroutine via processEffect()
}
```

Each `processAction` call appends to `g.currentResolution.Actions`.

### Serialization for LLM training

Both levels are available:

```json
{
  "turn": 3,
  "step": "PrecombatMain",
  "source": "Lightning Bolt",
  "controller": "player-a",
  "targets": ["grizzly-bears-uuid"],
  "actions": [
    {
      "type": "DealDamage",
      "target": "grizzly-bears-uuid",
      "amount": 3,
      "actual_amount": 3,
      "replaced": false
    }
  ],
  "summary": "Lightning Bolt dealt 3 damage to Grizzly Bears."
}
```

High-level summaries can be generated from the action trace after resolution
(template-based or programmatic).

---

## Continuous Effects in the Pipeline

Continuous effects currently run in `ApplyContinuousEffects()`, which resets all
bonus state and re-applies every active effect in layer order. They write
directly to `Permanent` fields (`powerBonus`, `toughBonus`, `grantedAttrs`,
etc.).

Under the action pipeline, continuous effects emit `CharacteristicAction` values
through the same pipeline:

```go
type ContinuousEffect interface {
    // Emit sends characteristic actions for this effect.
    // Called during ApplyContinuousEffects() in layer order.
    Emit(ctx *EffectContext, g GameReader)

    GetLayer() Layer
    GetDuration() Duration
    IsActive(g GameReader) bool
    SourceID() uuid.UUID
}
```

Example — Glorious Anthem (+1/+1 to your creatures):

```go
func (e *boostControlled) Emit(ctx *EffectContext, g GameReader) {
    for _, perm := range g.Battlefield() {
        if perm.Controller == e.controller && perm.IsCreature() {
            ctx.BoostPT(perm.ID(), e.power, e.toughness)
        }
    }
}
```

These `CharacteristicAction` values flow through the pipeline:
- **Replacement effects** can intercept them (e.g., "effects that would modify
  this creature's power do nothing instead")
- **Logging** captures exactly what each continuous effect contributed
- **Apply** still writes to the same `Permanent` fields after reset

The layer system controls ordering. The engine calls `Emit` for each continuous
effect in layer order, processing the actions synchronously (continuous effects
don't need goroutines — they emit a known set of actions based on current state).

For continuous effects, the `EffectContext` uses **fire-and-forget** by default
(no ack needed since continuous effects are recomputed from scratch each time).

---

## MTG Rules Compliance

### Trigger queueing during resolution

Per MTG rules (603.3), triggered abilities that trigger during spell resolution
wait until the spell finishes resolving, then go on the stack.

The engine implements this naturally:
1. `processAction` calls `g.queueTriggeredAbilities(action)` which checks
   triggers but doesn't resolve them
2. After the effect goroutine finishes, `ResolveStackObject` puts queued
   triggers on the stack
3. Priority passes, and triggers resolve normally

### State-based actions

SBAs are not checked during a single spell's resolution. The engine checks SBAs
after `ResolveStackObject` completes (same as today).

### Replacement effect ordering (614.5)

When multiple replacements apply to the same event, the affected player or
controller of the affected object chooses which to apply first. This is a player
choice that can be routed through the existing `Player.ChooseMode()` mechanism.

### Self-replacement effects (614.6)

Self-replacement effects (those from the same source as the event) are applied
first. The `applyReplacements` loop can check `r.SourceID() == action.ActionSource()`
to prioritize these.

---

## Migration Plan

### Phase 1: Action type system

**Scope:** New files only. No changes to existing code.

- `pkg/mage/action.go` — `Action` interface, `ActionResult`
- `pkg/mage/action_damage.go` — `DamageAction` interface + concrete structs
- `pkg/mage/action_life.go` — `LifeAction` + concretes
- `pkg/mage/action_zone.go` — `ZoneChangeAction`, `DestroyAction` + concretes
- `pkg/mage/action_counter.go` — `CounterAction` + concretes
- `pkg/mage/action_tap.go` — `TapAction` + concretes
- `pkg/mage/action_characteristic.go` — `CharacteristicAction` + concretes
- `pkg/mage/action_stack.go` — `StackAction` + concretes
- `pkg/mage/action_misc.go` — `ManaAction`, `TokenAction`, `CombatAction`, `GameRuleAction` + concretes

**Deliverable:** Compiles, no behavioral change. Types exist but aren't used yet.

### Phase 2: Pipeline core + replacement interface

- `pkg/mage/pipeline.go` — `processAction`, `applyReplacements`,
  `applyMutation` (type-switch dispatcher), `ReplacementEffect` interface
- `pkg/mage/effect_context.go` — `EffectContext`, `actionEnvelope`, sync/async
  send helpers, player choice methods
- `pkg/mage/game_log.go` — `GameLog`, `ResolutionLog`, `ActionRecord`

**Deliverable:** Pipeline exists and can process actions. Not wired into game
loop yet.

### Phase 3: GameMutator bridge

Convert each `GameMutator` method to internally create an action and send it
through the pipeline. The method signatures don't change — existing effects
keep calling `g.DealDamageToPlayer(...)` etc. — but internally the mutation
flows through `processAction`.

This means:
- All existing effects automatically get logging
- All existing effects automatically go through replacement effects
- No effect code changes required

**Key change:** `Game` holds an internal `actionPipeline` that `GameMutator`
methods delegate to. When called outside an effect goroutine (e.g., from the
game loop itself for SBAs, combat damage), the pipeline runs synchronously in
the calling goroutine.

**Deliverable:** All tests pass. Game log captures every mutation. Ad-hoc
replacement logic in `game.go` still works (will be migrated in Phase 4).

### Phase 4: Migrate replacement effects

Convert each ad-hoc shield/flag system to a `ReplacementEffect` implementation:

1. Regeneration → `RegenerationReplacement`
2. Damage prevention shields → `DamagePreventionReplacement`
3. Fog (prevent combat damage) → `FogReplacement`
4. Forcefield → `ForcefieldReplacement`
5. Color prevention → `ColorPreventionReplacement`
6. Reverse Damage → `ReverseDamageReplacement`
7. Lich (life gain → draw) → `LichLifeGainReplacement`
8. Lich (damage → sacrifice) → `LichDamageReplacement`
9. Ali from Cairo (minimum life) → `MinimumLifeReplacement`
10. Damage redirects → `DamageRedirectReplacement`
11. Damage prevention rules → `FilteredPreventionReplacement`

Remove the `damageModifiers` struct from `EffectManager`. Tests pass against
same scenarios.

**Deliverable:** Generic replacement system handles all existing "if would"
patterns. New replacement effects can be added by cards without touching engine
internals.

### Phase 5: Migrate effects to EffectContext (incremental)

Convert effects one-by-one from `Apply(GameMutator, ...)` to goroutine-based
`Apply(EffectContext)`. Start with the simplest:

1. **Damage effects** — `DealDamage`, `DealDamageToAllCreatures`, etc.
2. **Life effects** — `GainLife`, `LoseLife`
3. **Removal effects** — `DestroyTarget`, `ExileTarget`
4. **Card effects** — `DrawCards`, `DiscardCards`
5. **Counter/tap effects** — `AddCounters`, `Tap`
6. **Complex effects** — `BalanceEffect`, `ChaosOrbEffect`, `PowerSinkEffect`
7. **FuncEffect closures** — these are the bulk; convert inline closures in
   card definitions

`FuncEffect` signature changes from:
```go
FuncEffect(text string, props EffectProperties,
    fn func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error,
) Effect
```

to:
```go
FuncEffect(text string, props EffectProperties,
    fn func(ctx *EffectContext) error,
) Effect
```

The bridge from Phase 3 means both old-style and new-style effects can coexist
during migration.

### Phase 6: Continuous effects

Migrate continuous effects to emit `CharacteristicAction` values through the
pipeline. The `ContinuousEffect` interface gains an `Emit` method.
`ApplyContinuousEffects` calls `Emit` in layer order, processing characteristic
actions through the pipeline (fire-and-forget, no ack needed).

---

## Open Questions

1. **Serialization format for game logs.** JSON is simple. Protobuf is more
   compact for large datasets. Decision can be deferred — internal types first,
   serialization format later.

2. **Replacement effect player choice UI.** When multiple replacements apply,
   the affected player must choose order. This needs a new choice type routed
   through `Player`. Low priority — most games have at most one replacement
   applying.

3. **Performance.** Goroutine-per-effect + channels adds overhead vs. direct
   method calls. For a game engine (not a high-frequency system), this should be
   negligible. Profile after Phase 3.

4. **Copy effects.** Doppelganger/Clone create copies of permanents. The copy
   layer (Layer 1) is complex. How copy actions interact with the pipeline needs
   careful design — deferred to Phase 6.

5. **Cleanup of damageModifiers.** Phase 4 removes the ad-hoc shield system.
   Some shields are consumed (one-shot) and some persist (continuous). The
   `ReplacementEffect.IsActive` method handles expiry, but consumed-on-use
   shields need mutable state in the replacement — this is fine since
   replacements are owned objects, not shared.
