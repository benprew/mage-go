# mage-go: 4+1 Architectural View Model

> A two-player Magic: The Gathering rules engine in Go, inspired by XMage.
> Reference: [4+1 Architectural View Model](https://en.wikipedia.org/wiki/4%2B1_architectural_view_model)
>
> Current through mage-go v0.5.1 (July 2026).

---

## 1. Logical View

The logical view describes the engine's principal abstractions and how card definitions use them.

### Core Domain Model

```text
Game (aggregate root rules engine and coordinator)
  |-- Players [2]
  |     |-- Hand, Library, Graveyard, Ante
  |     |-- ManaPool
  |     '-- Life, poison, loss and per-turn state
  |
  |-- ZoneSystem
  |     |-- Battlefield []*Permanent (with copy-on-write search semantics)
  |     |-- Exile []ExiledCard (including face-down visibility metadata)
  |     |-- Entering permanent tracker (ETB)
  |     '-- Last-known-information (LKI) snapshots
  |
  |-- TurnSystem
  |     |-- Turn counter, step/phase, active player index
  |     |-- TurnSchedule (remaining, inserted, and skipped steps/turns)
  |     |-- Extra turns queue
  |     '-- Skip next untap counters
  |
  |-- ResolutionState
  |     |-- Transient resolving card, cast zone, and CastContext
  |     |-- X value, chosen mode, triggering event amount & source ID
  |     |-- Resolving targets, damage/counter distributions
  |     '-- Last cost reveal & last sacrificed permanent
  |
  |-- TriggerSystem
  |     |-- Pending triggers
  |     |-- Delayed triggers
  |     '-- Armed state triggers (CR 603.8)
  |
  |-- ManaSystem
  |     |-- Mana source discovery and solver planning
  |     '-- Auto-tap payment scratch buffers
  |
  |-- DamageSystem
  |     |-- Damage execution and reflection
  |     |-- Combat damage step aggregation and source breakdown
  |     '-- Damage history
  |
  |-- TrackerSystem
  |     |-- TurnTrackers (per-turn observations)
  |     '-- DuelTrackers (per-duel observations)
  |
  |-- RandomSource (deterministic random numbers & coin flips)
  |-- Stack (StackObjects for spells and abilities)
  |-- Combat (CombatGroups, attacker/blocker pairings, and bands)
  '-- EffectManager (continuous effects, replacement effects, and GameRules)
```

### Key Types and Subsystems

| Type | Responsibility |
|------|----------------|
| `Game` | Aggregate root rules engine coordinator; owns top-level loops, cloning, and subsystem orchestration |
| `ZoneSystem` | Battlefield copy-on-write management, exile storage, LKI snapshots, and zone transitions |
| `TurnSystem` | Turn/step progression, active player tracking, extra turns, scheduling, and untap skips |
| `ResolutionState` | Transient resolution scratch state (X, mode, cast snapshot, targets, distributions) |
| `TriggerSystem` | Pending, delayed, and armed state-triggered abilities |
| `ManaSystem` | Source discovery, mana ability solver planning, and auto-tap execution |
| `DamageSystem` | Damage execution, combat aggregation, prevention, and history tracking |
| `TrackerSystem` | Turn-scoped (`TurnTrackers`) and duel-scoped (`DuelTrackers`) game action observations |
| `RandomSource` | Deterministic random number and coin flip queue |
| `Ability` | Common identity/controller contract for spell, activated, mana, triggered and static abilities |
| `ActionDefinition` | Shared definition for stack-using spells and activated abilities: effects, costs, targets, timing, limits and AI hints |
| `Action` | A pending mutation that can be transformed or prevented by replacement effects |
| `Card` | Card identity and printed characteristics, abilities, modes, costs and ownership |
| `ContinuousEffect` | Layered effect that is reapplied while active for a declared duration |
| `Cost` | Payment feasibility and execution, including mana, tap, sacrifice, discard and optional/alternate costs |
| `EffectContext` | Resolving `Game`, source, controller, targets and scratch variables passed to an effect |
| `Effect` | One-shot resolving behavior: `Apply(*EffectContext) error`, plus text and AI-visible properties |
| `GameReader` | Composed query interface (`PlayerReader`, `BattlefieldReader`, `ResolutionReader`, `TurnReader`, `CombatReader`, `TrackerReader`, `StackReader`) used by values, selectors, filters and trigger conditions |
| `PermanentLKI` / `LKIView` | Frozen battlefield state for dies, leaves and other last-known-information queries |
| `Permanent` | Mutable battlefield form of a card, including computed characteristics and control state |
| `Player` | Owns player zones and resources and supplies choices; implemented by base, human, AI, search and test players |
| `Registry` | Thread-safe mapping from card name to factory |
| `ReplacementEffect` | Matches and replaces an `Action`; returning `nil` prevents or fully consumes it |
| `StackObject` | Cast/activation-time snapshot of a spell or ability waiting to resolve |
| `Target` / filters | Legal-target discovery and validation using composable permanent and card predicates |
| `TurnSchedule` | Mutable sequence supporting extra/skipped steps, phases and turns |

### Card-Facing Effect Model

Effects no longer resolve through a `GameMutator` interface. The current canonical path is:

```text
StackObject
  -> Effect.Apply(*EffectContext)
       |-- ctx.Game        concrete *Game
       |-- ctx.SourceID
       |-- ctx.Controller
       |-- ctx.Targets
       '-- ctx.Vars        per-resolution scratch data
```

`GameReader` remains the restricted query surface for code that should be observational, such as `ValueSource`, `PlayerSelector`, filters and trigger predicates. `FlipCoin` is the documented exception and must not be called from repeatedly evaluated conditions. Card implementations normally use the re-exporting `pkg/mage/dsl` package; its explicit escape hatches expose concrete engine types only where the DSL is not yet sufficient.

Every `Effect` also reports `EffectProperties` and optional `AIHint` metadata. The heuristic and search AIs use this data to classify outcomes, timing, targets, removal, burn, pump, protection and other strategic roles without switching on each concrete card.

### Ability and Action Taxonomy

| Kind | Representation | Runtime behavior |
|------|----------------|------------------|
| Activated | `ActionDefinition` (`SimpleActivatedAbility` is a compatibility alias) | Pays costs, chooses targets and uses the stack |
| Mana | `ManaAbility` | Resolves immediately without the stack |
| Spell | `SpellAbility` over `ActionDefinition` | Cast from a permitted zone and resolves from the stack |
| Static | `StaticAbilityHolder` and specialized ability types | Registers continuous effects, replacements or rule modifiers |
| Triggered | `GenericTriggered` | Watches a typed event or state condition, snapshots its controller and creates a stack object |

Spells and non-mana activated abilities deliberately share `ActionDefinition`. It holds `ActionKind`, effects, targets, costs, timing rules, activation limits/permissions, conditions and AI hints. This keeps casting and activation enumeration aligned for the UI, AI and foreign-function interface.

### Stack and Casting Snapshots

`StackObject` stores more than a card and targets. It preserves information that must not be recomputed at resolution:

- chosen X and modal mode;
- per-mode and multi-target selections;
- the zone each target occupied when chosen;
- divided-damage allocation;
- whether the object is a spell copy;
- cast zone and exile-on-leave-stack behavior;
- `CastContext` for facts captured while casting; and
- triggering event amount and source.

This supports modal spells, alternate costs, flashback-like casting, spell copies, zone-sensitive triggers, intervening target movement and effects that refer to cast-time information.

### Replacement Action Pipeline

The engine models replaceable mutations as typed `Action` values. Current action families include:

- damage to a player or creature;
- destruction;
- drawing a card;
- gaining life; and
- adding counters, including counters placed as a permanent enters.

The flow is synchronous and inline:

```text
Game mutation request
  -> construct Action
  -> EffectManager.ApplyReplacements
       -> apply each matching non-prevention replacement at most once
       -> apply matching prevention effects
       -> repeat while a transformed action has another match
  -> execute the surviving action, or stop if it is nil
```

Replacement effects may transform the amount or action type, perform replacement-specific work, or consume the action. Regeneration, damage prevention/redirection, draw replacement, counter modification and life-gain replacement use this mechanism. Not every named action is represented by an `Action` yet; for example, sacrifice has a first-class `Game.DoSacrifice` path with zone-change, sacrifice-event and LKI handling.

### Events, Triggers and Last-Known Information

`GameEvent` carries an event type plus source, target, player, amount, flag and zone endpoints. `EvtZoneChange` is the general event for battlefield, hand, library, graveyard, exile and stack movement; specialized events remain for game actions and aggregate combat facts.

`GenericTriggered` supports:

- event predicates built from composable `TriggerConditionData`;
- triggered targets selected when the trigger is created;
- optional and modal triggers;
- abilities active in non-battlefield zones; and
- state triggers that fire once per false-to-true transition.

When a permanent leaves the battlefield, `PermanentLKI` captures its controller, types, subtypes, P/T, counters, abilities and attachment state. `LookupObject` returns one `LKIView` abstraction over either a live permanent or its snapshot. Current LKI capture is centered on battlefield departures and is cleared during cleanup; durable and non-battlefield LKI remain future extensions.

### Continuous Effects, Layers and Control

The layer enum follows the familiar MTG ordering:

| Layer | Name | Current role |
|-------|------|--------------|
| 1 | Copy | Copy effects and copied characteristics |
| 2 | Control | Timestamped and conditional controller computation |
| 3 | Text | Represented by `LayerText`; not currently traversed by `EffectManager.Apply` |
| 4 | Type | Card type and subtype changes |
| 5 | Color | Color-setting effects |
| 6 | Ability | Keyword and runtime-ability grants/removals |
| 7 | P/T | Base P/T setting and modifications |

Each application cycle clears derived state, removes expired effects, applies copy and control, then type, color, ability and P/T effects. It also rebuilds cycle-scoped replacement effects, combat restrictions and dynamic game rules.

Control is a computed Layer 2 characteristic. A permanent retains a base controller and receives a computed controller after control effects are reconciled to stability. Conditional control effects latch expiration, timestamp order decides later effects, attachment-based control follows the aura's controller, and a real control change updates ability context and summoning-sickness timing.

### Attribute System

Attributes cover keywords and battlefield capabilities in fixed-size arrays on `Permanent`:

- `baseAttrs` contains intrinsic and persistent counts;
- `grantedAttrs` is cleared and recomputed each continuous-effect cycle; and
- `HasAttr(a)` tests whether the combined value is positive.

Base grants are additive. Continuous ability grants and removals are last-writer-wins in timestamp order: a removal offsets all intrinsic instances, while a later grant restores the attribute. This matches effects such as “loses all landwalk” interacting with multiple lords more closely than simple integer subtraction.

### Searchable Game State

`Game.Clone` creates branches for AI search. Immutable `Card` and `Effect` values are shared; mutable players, stack, combat, schedules, triggers, trackers and rules are copied. Battlefield permanents use copy-on-write sharing and become private through `MutablePermanent` before mutation. Interactive callbacks are removed and cloned players become `SearchPlayer` instances with non-interactive choices.

### Design Patterns

| Pattern | Where |
|---------|-------|
| Factory | Card registry and set registration |
| Strategy | Human/test/search player choices and `AIStrategy` implementations |
| Observer | Typed events and triggered abilities |
| Chain of responsibility | Replacement-effect pipeline |
| State machine | Turn schedule, steps, priority and stack resolution |
| Proxy/view | `Permanent` over `Card`, `LKIView`, `GameReader` |
| Copy-on-write | Battlefield state in AI search clones |
| Builder/DSL | Card constructors, action options and composable effect pipelines |
| Predicate | Filters, selectors and trigger/activation conditions |

---

## 2. Process View

The process view describes runtime sequencing, game flow and concurrency.

### Turn Structure

```text
Untap -> Upkeep -> Draw -> PrecombatMain -> BeginCombat ->
DeclareAttackers -> DeclareBlockers -> FirstStrikeDamage ->
CombatDamage -> EndCombat -> PostcombatMain -> EndStep -> Cleanup
```

`TurnSchedule` builds this canonical sequence for each turn and lets effects insert or skip steps, skip combat phases and skip turns. `Game` separately queues extra turns.

A step generally performs the following work:

1. set the current step and apply continuous effects;
2. perform the step's turn-based actions and fire events;
3. put pending triggers on the stack;
4. run a priority round when that step grants priority;
5. check state-based actions and reapply effects before returning; and
6. empty mana pools at the step/phase boundary.

The untap step does not grant priority. A normal cleanup step does not either, but cleanup repeats with a priority round when a state-based action or trigger occurs during cleanup.

### Priority and Stack Resolution

There are two engine modes:

```text
No PriorityHandler (unit tests and direct engine use)
  CheckStateBasedActions -> ResolveStack atomically

PriorityHandler installed (interactive, AI, replay and FFI)
  loop:
    CheckStateBasedActions
    CheckStateTriggers
    PutTriggersOnStack
    ask active player, then non-active player
    if someone acts: execute and restart
    if both pass and stack is empty: end the step
    if both pass and stack is non-empty: resolve one object and restart
```

The engine's priority action vocabulary is pass, cast spell, activate ability and play land. The interactive package has a richer transport action type that also carries attacker, blocker and modal-choice interactions.

### Event-to-Trigger Pipeline

```text
FireEvent(GameEvent)
  -> inspect eligible live and zone-active triggered abilities
  -> evaluate read-only trigger conditions
  -> snapshot source/controller/event context in pendingTriggers
  -> choose trigger targets
  -> PutTriggersOnStack
  -> resolve Effect values with EffectContext
  -> state-based actions and newly generated triggers
```

State triggers are evaluated beside state-based actions and are armed until their condition becomes false. Delayed triggers are registered separately and remain until their event and condition match or their duration expires.

### State-Based Actions

`CheckStateBasedActions` loops until no further change is made. It currently handles:

- lethal damage and zero-or-less toughness;
- zero-loyalty planeswalkers;
- +1/+1 and -1/-1 counter annihilation;
- illegal or missing aura hosts and invalid equipment attachment;
- static sacrifice-unless-land requirements;
- legend and world rules;
- poison loss and drawing from an empty library; and
- follow-on triggers generated by those transitions.

The planeswalker implementation is intentionally partial: entry loyalty and zero-loyalty state-based actions exist, but loyalty activation rules and attacking planeswalkers are not complete.

### Mana Payment

`SolveMana` is a pure solver over floating mana, candidate sources, restrictions, conversions and preservation scores. It supports colored, generic and hybrid requirements, multi-mana sources and mana bonuses. Callers apply the returned tap plan; `CanSolveMana` uses the same logic for affordability checks. This keeps UI legality checks, AI planning and actual auto-tapping consistent.

### Concurrency Model

| Component | Model |
|-----------|-------|
| Core `Game` | Single-threaded mutation; a game loop owns its game state |
| Registry | `sync.RWMutex` protects the global card factory map |
| Human input | Buffered game/action and choice request/response channel pairs |
| AI strategy | Called synchronously by the owning game loop |
| Search | Synchronous cloned-state exploration with copy-on-write permanents |
| TUI/WASM | Game loop and transport pumps run in goroutines |
| SSH server | Concurrent sessions and games; lobby state protected by a mutex |
| C/Python FFI | Process-global handle table protected by mutexes; each game has a decision-loop goroutine |
| Batch/text rollout FFI | Coordinates multiple handles and inference batches across goroutines |

The core engine is not safe for concurrent mutation. Concurrency exists at the adapter boundary and between independent games, not inside one game's rules transitions.

### Randomness and Reproducibility

Shuffling, random discard/selection, coin flips and some tools use `math/rand`. Tests can script coin-flip results, while scenario and FFI entry points accept seeds for reproducible runs. Reproducibility therefore requires controlling both player choices and random seeding; `Game.Clone` does not own or clone an independent RNG stream.

---

## 3. Development View

The development view describes package organization, build tooling and the card workflow.

### Package Layout

```text
mage-go/
|-- pkg/mage/                       # Core rules engine
|   |-- core/                       # Enums and value types
|   |-- dsl/                        # Card-author-facing re-export surface
|   |-- gametest/                   # Scenario test harness
|   '-- interactive/                # UI-neutral human/AI adapter layer
|       |-- ai/                     # AI interface, player and personalities
|       |   |-- heuristic/          # Local personality-driven strategy
|       |   |-- search/             # Full-turn minimax, TT and Zobrist hashing
|       |   '-- combatsolver/       # Joint combat/trick solver
|       '-- eval/                   # Board, role, targeting and lethal evaluation
|
|-- pkg/catalog/                    # Indexed Scryfall metadata catalog
|
|-- cards/                          # Set packages and generated catalog data
|   |-- limited/                    # Limited Edition Alpha
|   |-- arabian/                    # Arabian Nights
|   |-- antiquities/                # Antiquities
|   |-- legends/                    # Legends
|   |-- fallen_empires/             # Fallen Empires
|   |-- fourthedition/              # Fourth Edition
|   |-- jumpstart/                  # Jumpstart implementation package
|   |-- secretsofstrixhaven/        # Secrets of Strixhaven implementation package
|   '-- custom/                     # Custom/test cards
|
|-- internal/tui/                   # Bubble Tea models, views and deck helpers
|-- internal/scenario/              # Rogue-deck parsing and scenario result types
|
|-- cmd/
|   |-- tui/                        # Local terminal game
|   |-- server/                     # SSH lobby and game server
|   |-- wasm/                       # Browser/WASM adapter
|   |-- pylib/                      # C shared library and Python package
|   |-- gametest/                   # Interactive/profilable AI-vs-AI runner
|   |-- scenariotest/               # Batch AI-vs-AI JSONL runner
|   |-- scenarioanalyze/            # Scenario anomaly/statistics analysis
|   |-- oracle-replay/              # XMage JSONL replay and state comparison
|   |-- fetchset/, fetchcatalog/    # Scryfall acquisition tools
|   |-- genset/                     # Set/card stub generator
|   |-- cardart/                    # Pixel-art renderer
|   '-- forge/                      # Procedural card generator
|
|-- data/                           # Set, catalog and tokenizer data
|-- rogue_dck/                      # Shandalar rogue deck lists
|-- web/                            # Browser UI assets
|-- docs/ and active-design-docs/   # Stable references and evolving designs
|-- .agents/skills/                 # Card, set and engine workflows
|-- pyproject.toml / setup.py       # Python binding build/package metadata
'-- Makefile
```

`cards/all.go` is the side-effect import used by applications to register the production aggregate. A package directory's presence does not necessarily mean it is imported by that aggregate; currently Jumpstart and Secrets of Strixhaven are kept outside `cards/all.go`.

### Dependencies

- The rules engine's direct third-party runtime dependency is `github.com/google/uuid`.
- Terminal and SSH applications use Bubble Tea, Lip Gloss, Wish and Charmbracelet SSH.
- Enum generation uses `github.com/dmarkham/enumer` through Go's tool dependency mechanism.
- The optional Python package uses `cffi` and `orjson` and builds the Go C shared library during packaging.

### Card Registration and DSL

Set packages register factories by exact card name during package initialization. Modern card code imports `pkg/mage/dsl` for constructors, effects, targets, costs, selectors, values and core re-exports. Factories create new card identities for each deck copy rather than sharing runtime cards.

Generated set packages normally contain:

- `creatures.go`, `spells.go`, `enchantments.go`, `artifacts.go` and `lands.go`;
- `register.go` or `set.go` for set registration and embedded catalog metadata;
- `test.go` for cross-set side-effect imports; and
- focused `*_test.go` files for card behavior and interactions.

### Build and Verification

| Target | Current command |
|--------|-----------------|
| Test everything | `make test` -> `go test ./...` |
| Build packages | `make build` -> `go build ./...` |
| Format/lint/fix | `make lint` -> `modernize -fix`, `golangci-lint fmt`, `golangci-lint run --fix` |
| Build WASM | `make wasm` -> compile `cmd/wasm` and copy `wasm_exec.js` |
| Build card art tool | `make cardart` |

Engine and card changes follow test-driven development. Card behavior is tested through `gametest.TestGame`; lower-level engine tests live beside `pkg/mage` code. The suite also contains comprehensive-rules-oriented tests grouped by rule chapter, AI/search tests, replay tests, FFI encoder tests and catalog/tool tests.

### Set Implementation Pipeline

```text
Scryfall
  |-- cmd/fetchset ------> data/<SET>.json
  |-- cmd/fetchcatalog --> data/catalog/<SET>.json
  '-- cmd/genset --------> cards/<package>/*.go + embedded catalog JSON
                              -> card-by-card TDD
                              -> implement-set / implement-card workflow
                              -> validate-set audit
                              -> go test ./... and make lint
```

`pkg/mage/doc.go` documents important engine contracts, while `pkg/mage/dsl/dsl.go` is the practical inventory of the card-author-facing surface. Comprehensive-rule details are indexed in `docs/comprehensive-rules-index.md`.

---

## 4. Physical View

The physical view maps packages to executable and integration boundaries.

### 4.1 Native TUI

```text
Terminal
  <-> Bubble Tea model (internal/tui)
  <-> HumanPlayer channels (pkg/mage/interactive)
  <-> game-loop goroutine
  <-> Game + heuristic/search AI
```

`cmd/tui` is a local human-vs-AI application. Deck construction, AI personality inference and AI mode selection happen in the native process; no network service or persistence is required.

### 4.2 SSH Server

```text
SSH clients
  <-> Wish SSH server :${PORT:-2222}
       |-- Bubble Tea lobby
       |-- mutex-protected game slots
       |-- human-vs-human multiplayer loop
       '-- human-vs-AI single-player loop
```

The server creates `.ssh/server_ed25519` through Wish's host-key handling, starts a Bubble Tea program per SSH session and forwards terminal resize events. Each player receives an independent channel set connected to the shared game loop. There is no database or durable match state.

### 4.3 Browser/WASM

```text
Static HTTP server
  -> index.html + game.js + wasm_exec.js + mage.wasm

Browser tab
  |-- JavaScript UI
  |-- syscall/js bridge
  |-- interactive game-loop goroutine
  '-- Game + selected AI, entirely client-side
```

The Go module exports four JavaScript globals:

- `mageGetCardList()`;
- `mageStartGame(deckJSON, onGameMsg, onChoiceReq, aiDeckJSON?, personality?, mode?)`;
- `mageSendAction(actionJSON)`; and
- `mageSendChoice(choiceJSON)`.

The web server only delivers static files; rules execution and AI run in the browser.

### 4.4 C Shared Library and Python Binding

```text
Python / C caller
  <-> cgo ABI (libmage.so, libmage.dylib or DLL)
       |-- process-global opaque game handles
       |-- JSON state/legal/step API
       |-- batched game polling and stepping
       |-- native token and decision-spec encoders
       |-- grammar masks
       '-- text-rollout inference scheduler
            <-> per-game decision-loop goroutines
```

`cmd/pylib` can be built with `go build -buildmode=c-shared`. `setup.py` builds and packages that library for the `mage-go` Python package. The ABI provides simple handle-based game control as well as high-throughput batch and tokenization paths intended for model training and inference.

### 4.5 Offline and Validation Tools

| Binary | Purpose |
|--------|---------|
| `gametest` | Run and profile one or more AI-vs-AI games with selectable strategies and decks |
| `scenariotest` | Run seeded batches against Shandalar rogue decks and emit JSONL results |
| `scenarioanalyze` | Group panics, timeouts, anomalous games and win statistics from scenario JSONL |
| `oracle-replay` | Replay XMage recorder JSONL/GZIP and compare legal actions and priority snapshots |
| `fetchset` / `fetchcatalog` | Download implementation and metadata inputs from Scryfall |
| `genset` | Generate set source, registration, tests and embedded catalog scaffolding |
| `cardart` | Render pixel-art card assets |
| `forge` | Generate procedural card sets |

### Deployment Comparison

| Dimension | TUI | SSH | Browser/WASM | C/Python |
|-----------|-----|-----|--------------|----------|
| Boundary | Local terminal | SSH/PTTY | `syscall/js` callbacks | C ABI and Python wrapper |
| Players | Human + AI | Human + human or human + AI | Human + AI | Caller-controlled players/batches |
| Game ownership | One local process | One goroutine per game | One browser tab | Opaque process-global handle |
| State transport | Go channels | Go channels through session TUIs | JSON callbacks | JSON or packed native buffers |
| Persistence | None | None | None | Caller-owned |

---

## +1. Scenarios View

The scenarios connect the other four views through representative workflows.

### Scenario 1: Resolve a Targeted Spell

1. A priority handler selects a spell, X value and legal targets.
2. Casting pays costs and creates a `StackObject` with target-zone and cast-time snapshots.
3. `EvtSpellCast` and `EvtBecomesTarget` events queue applicable triggers.
4. Both players pass; the top stack object resolves.
5. Each `Effect` receives an `EffectContext` and requests mutations on `*Game`.
6. Replaceable mutations pass through `EffectManager.ApplyReplacements`.
7. The engine fires resulting events, checks state-based actions and starts the next priority round.

**Views exercised:** Logical (actions, effects, stack), Process (priority and replacement flow)

### Scenario 2: Search-AI Combat Decision

1. The AI receives priority or a combat declaration callback.
2. The heuristic strategy gathers `EffectProperties`, targets and available mana, or the search strategy generates legal moves.
3. Search clones the game, sharing immutable cards/effects and battlefield permanents copy-on-write.
4. The combat solver evaluates attacker subsets, defender blocks, pump/removal tricks and first/double-strike windows.
5. Evaluation and transposition-table results select an action or cache a turn plan.
6. The chosen action is executed on the live, single-owner game loop.

**Views exercised:** Logical (clones and AI metadata), Process (synchronous search), Development (AI subpackages)

### Scenario 3: Play Through a UI Adapter

1. TUI, SSH or browser code constructs players and decks and starts an interactive game loop.
2. `SnapshotGameState` produces a viewer-oriented serializable state.
3. `GetAvailableActions` builds legal action options.
4. A human response returns through channels or a JavaScript callback bridge.
5. Modal choices use the separate choice request/response path.
6. The adapter renders updates until the engine reports a winner or disconnection.

**Views exercised:** Process (channels), Physical (native, SSH or WASM deployment)

### Scenario 4: Implement a Card or Set

1. Fetch Oracle and catalog data from Scryfall.
2. Generate set scaffolding with full Oracle comments.
3. Read `pkg/mage/doc.go`, the DSL and relevant comprehensive-rule sections.
4. Write a failing `gametest.TestGame` scenario.
5. Compose an existing action/effect/target/cost pipeline, or add exact engine support with engine tests.
6. Run focused tests, `go test ./...` and `make lint`.
7. Validate registration, Oracle fidelity, unsupported markers and regression coverage.

**Views exercised:** Logical (DSL and rules primitives), Development (generation and TDD)

### Scenario 5: Cross-Engine Replay

1. XMage records a game as JSONL, optionally GZIP-compressed.
2. `oracle-replay` reconstructs decks and scripted choices in mage-go.
3. At each recorded priority point it compares snapshots and available actions.
4. The tool reports the first state divergence, with configurable loose action matching and tracing.

**Views exercised:** Process (deterministic driving), Physical (offline validation tool)

### Scenario 6: Batched Model Rollout

1. Python creates game handles through the C ABI with seeded configuration.
2. Native game loops stop at pending decisions.
3. Batch polling gathers state and legal choices across ready games.
4. Native encoders produce packed state/decision tokens and grammar masks.
5. A model returns choices, which are decoded and stepped back into the relevant games.
6. Completed games release their handles and buffers.

**Views exercised:** Logical (serializable decisions), Process (batched goroutines), Physical (C/Python boundary)

---

## Cross-Cutting Concerns

### Oracle Fidelity

Card factories retain full Oracle text comments, and engine primitives are expected to model every restriction and edge case rather than approximate it. The general zone-change event, LKI views, cast snapshots, target-zone validation and typed replacement actions exist to preserve facts across the exact points where Magic rules need them.

### Extensibility

The card DSL composes constructors, action definitions, effects, targets, filters, values, conditions and costs. Engine changes are still required when a rule cannot be expressed exactly. `pkg/mage/dsl` makes the intended card-facing boundary visible, while `EffectContext` escape hatches make remaining coupling explicit.

### Testability and Validation

The same rules paths serve scenario tests, interactive play, AI search, foreign callers and replay validation. Seeded batch scenarios and XMage replay add system-level checks beyond unit tests. Search cloning and scripted test players allow complex branches without UI I/O.

### Performance

Hot paths avoid allocations through fixed-size attribute/counter arrays, reusable effect and mana scratch storage, copy-on-write battlefield clones, cached render plans, packed token output, transposition tables and Zobrist hashing. Timing and profiling hooks exist in the AI runner, engine loop and C ABI.

### Portability

The rules engine is shared by native terminal programs, an SSH service, WebAssembly and a C shared library. UI-specific serialization and concurrency stay in `pkg/mage/interactive` or executable adapters, while core rule transitions remain synchronous Go calls.

### Known Architectural Boundaries

- The engine is intentionally two-player.
- A single `Game` must have one mutation owner.
- Randomness is process-level `math/rand`, not game-owned cloneable state.
- Text-layer traversal and full planeswalker rules are incomplete.
- LKI is strongest for battlefield departures; general event-scoped LKI is not complete.
- The card DSL still exposes concrete engine escape hatches while its package boundary evolves.
