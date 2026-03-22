# mage-go: 4+1 Architectural View Model

> A 2-player Magic: The Gathering rules engine in Go, inspired by XMage.
> Reference: [4+1 Architectural View Model](https://en.wikipedia.org/wiki/4%2B1_architectural_view_model)

---

## 1. Logical View

The logical view describes the key abstractions, their responsibilities, and relationships.

### Core Domain Model

```
Game (Central Orchestrator)
  |-- Players [2]
  |     |-- Hand, Library, Graveyard, Ante (zones)
  |     |-- ManaPool
  |     '-- Life, PoisonCounters
  |
  |-- Battlefield []*Permanent
  |     |-- Card (immutable definition)
  |     |-- Attrs (baseAttrs + grantedAttrs, additive integer counts)
  |     |-- Counters, Damage, Tapped, PhasedOut, FaceDown
  |     |-- Attachments / AttachedTo
  |     '-- RuntimeAbilities
  |
  |-- Stack (LIFO)
  |     '-- StackObject (Card, Effects, Targets, XValue, ModeChoice)
  |
  |-- Combat
  |     |-- CombatGroup (attacker -> blockers)
  |     '-- Bands (banding groups)
  |
  |-- EffectManager
  |     |-- ContinuousEffect[] (applied by layer each cycle)
  |     |-- ReplacementEffect[] (intercept Actions in pipeline)
  |     '-- GameRules (damage shields, mana restrictions, cost reductions)
  |
  '-- Exile []ExiledCard
```

### Key Types

| Type | Responsibility |
|------|---------------|
| **Game** | Central state machine: turns, phases, events, rule enforcement |
| **Player** (interface) | Identity, zones, mana, life, decisions (strategy-abstracted) |
| **Card** (interface) | Immutable card definition: name, cost, types, abilities, P/T |
| **Permanent** | Mutable on-battlefield manifestation of a Card |
| **Ability** | Base interface for all ability types (spell, triggered, activated, mana, static) |
| **Effect** | One-shot game action: `Apply(g GameMutator, sourceID, controller, targets)` |
| **ContinuousEffect** | Persistent effect applied each cycle in layer order |
| **ReplacementEffect** | Intercepts and transforms Actions (damage prevention, regeneration) |
| **Target** | Targeting requirement with validation: `Possible()`, `Choose()`, `Chosen()` |
| **PermanentFilter** | Composable predicate for permanent matching |
| **Cost** | Activation/casting cost: `CanPay()`, `Pay()` |
| **Stack / StackObject** | LIFO queue of pending spells and abilities |
| **Combat / CombatGroup** | Attacker-blocker pairs, banding, damage assignment |
| **EffectManager** | Central registry for continuous effects, replacement effects, and dynamic rules |
| **Registry** | Thread-safe card factory repository (name -> factory function) |
| **GameReader** | Read-only game state interface for effects and queries |
| **GameMutator** | Mutation interface (extends GameReader) hiding engine internals |

### Interface Segregation: GameReader vs GameMutator

Effects interact with the game exclusively through two interfaces:

- **GameReader** exposes queries: `FindPermanent`, `FilterBattlefield`, `GetPlayer`, `XValue`, `CombatGroups`, etc.
- **GameMutator** extends GameReader with mutations: `DestroyPermanent`, `DealDamageToPlayer`, `PutOnBattlefield`, `AddContinuousEffect`, `AddPreventionShield`, etc.

This boundary prevents effects from bypassing rules checks or directly touching `*Game` internals.

### Ability Taxonomy

| Ability Type | Key Struct | Trigger |
|--------------|-----------|---------|
| **Spell** | `SpellAbility` | Cast from hand, resolves on stack |
| **Triggered** | `GenericTriggered` | Event-driven (ETB, dies, attacks, upkeep, etc.) |
| **Activated** | `SimpleActivatedAbility` | Player-activated with costs and restrictions |
| **Mana** | `ManaAbility` | Tap for mana (does not use the stack) |
| **Static** | `StaticAbilityHolder` | Always-on continuous effects |

### Layer System (MTG CR 613)

Continuous effects are applied in strict layer order each cycle:

| Layer | Name | Example |
|-------|------|---------|
| 1 | Copy | Clone, Copy Artifact |
| 2 | Control | Control Magic, Old Man of the Sea |
| 3 | Text | Sleight of Mind |
| 4 | Type | Lace effects, land type changes |
| 5 | Color | Color-changing effects |
| 6 | Ability | Grant/revoke keywords and abilities |
| 7 | P/T | Power/toughness boosts |

### Attribute System

Keywords and capabilities are stored as additive integer counts on Permanents:

- `baseAttrs` -- intrinsic attributes that persist until explicitly revoked
- `grantedAttrs` -- temporary attributes reset and recomputed each effect cycle
- `HasAttr(a)` returns `baseAttrs[a] + grantedAttrs[a] > 0`

This enables proper stacking: "grant Flying" + "remove Flying" + "grant Flying" = net 1 = has Flying.

### Design Patterns

| Pattern | Where |
|---------|-------|
| **Factory** | Registry: deferred card instantiation via factory functions |
| **Strategy** | Player interface: pluggable decision-making (AI, human, test) |
| **Observer** | Triggered abilities: subscribe to GameEvent types |
| **Chain of Responsibility** | Replacement effects: pipeline intercepting Actions |
| **State Machine** | Game: explicit turn/phase/step progression |
| **Proxy** | Permanent wraps Card, delegating queries but adding mutable state |
| **Facade** | GameMutator: simplified API over EffectManager complexity |
| **Predicate** | Filters: composable predicates for card/permanent selection |

---

## 2. Process View

The process view describes runtime behavior, concurrency, game flow, and event processing.

### Turn Structure

```
Untap -> Upkeep -> Draw -> Main1 -> BeginCombat -> DeclareAttackers ->
DeclareBlockers -> FirstStrikeDamage -> CombatDamage -> EndCombat ->
Main2 -> EndStep -> Cleanup
```

Each step:
1. Reapply continuous effects: `g.Effects.Apply(g)`
2. Execute step-specific logic (untap permanents, draw card, declare combat, etc.)
3. Check state-based actions (loop until quiescent)
4. Resolve stack / run priority loop

### Priority System

Two operating modes:

**Non-Interactive (test/AI-only):**
```
CheckStateBasedActions() -> ResolveStack()  // drain atomically
```

**Interactive (human play):**
```
loop {
    CheckStateBasedActions()
    PutTriggersOnStack()
    for each player (starting active):
        action = OnPriority(g, playerIdx)
        if action != Pass:
            execute action
            restart loop
    if all pass: break
}
```

### Event -> Trigger -> Stack Pipeline

```
FireEvent(GameEvent)
    |
    v
For each permanent/ability: if eventType matches && condition(evt) holds
    |
    v
Queue in pendingTriggers[]
    |
    v
PutTriggersOnStack() -- build StackObjects, push to Stack
    |
    v
ResolveStackObject: apply effects via GameMutator
    |
    v
State-based actions -> more triggers -> priority
```

### Replacement Effect Pipeline

When the engine emits an Action (damage, destruction, draw):
```
Action -> for each active ReplacementEffect:
            if Matches(action): action = Replace(action)
       -> apply final Action (or nil = prevented)
```

Built-in replacements: regeneration (destroy -> tap + remove damage), prevention shields (absorb N damage), fog (prevent all combat damage), forcefield (reduce unblocked damage to 1).

### State-Based Actions (SBAs)

Called after every step and every stack resolution; loops until quiescent:

1. Lethal damage: destroy creatures with damage >= toughness
2. Zero toughness: remove creatures with toughness <= 0
3. Counter annihilation: cancel equal +1/+1 and -1/-1 counters
4. Aura validity: destroy auras with illegal/missing targets
5. Equipment validity: unattach from non-creatures
6. Planeswalker uniqueness, legend rule, etc.

### Concurrency Model

| Component | Model |
|-----------|-------|
| **Game Engine** (`pkg/mage`) | Single-threaded; all mutations sequential and deterministic |
| **Registry** | `sync.RWMutex` protects card factory map |
| **Interactive Loop** | Dedicated goroutine; blocks on channel I/O |
| **Human Player I/O** | Buffered channels: `toTUI`, `fromTUI`, `choiceReqs`, `choiceResps` |
| **AI Strategy** | Called synchronously from game loop goroutine |
| **SSH Server** | One goroutine per session; each runs its own game loop |

The game engine is NOT concurrent internally. All mutations happen in the game loop goroutine. Channels provide async I/O decoupling only.

### Channel-Based Player Communication

```go
type HumanPlayer struct {
    *mage.BasePlayer
    toTUI       chan GameMsg          // game -> UI
    fromTUI     chan PriorityAction   // UI -> game
    choiceReqs  chan ChoiceRequest    // game -> UI (modal)
    choiceResps chan ChoiceResponse   // UI -> game (modal)
}
```

The game loop blocks on `<-fromTUI` waiting for human decisions. The TUI reads `GameMsg`, renders, collects input, and sends `PriorityAction` back. This decouples game logic from UI completely.

---

## 3. Development View

The development view describes code organization, packages, build system, and development workflow.

### Package Layout

```
mage-go/
|-- pkg/mage/                  # Core engine (105 .go files)
|   |-- core/                  # Enums, value types (19 files, code-generated)
|   |-- gametest/              # Test harness DSL (8 files)
|   '-- interactive/           # Human/AI player layer (11 files)
|       |-- ai/                # AI strategies
|       '-- eval/              # Board evaluation
|
|-- cards/                     # Card implementations by set
|   |-- limited/               # Alpha/Beta/Unlimited
|   |-- arabian/               # Arabian Nights
|   |-- antiquities/           # Antiquities
|   |-- legends/               # Legends
|   |-- fallen_empires/        # Fallen Empires
|   '-- custom/                # Custom/test cards
|
|-- cmd/                       # Binaries
|   |-- tui/                   # Terminal UI
|   |-- server/                # SSH multiplayer server
|   |-- wasm/                  # WebAssembly build
|   |-- fetchset/              # Fetch card data from Scryfall
|   |-- genset/                # Generate card stubs from JSON
|   |-- fetchcatalog/          # Bulk fetch multiple sets
|   |-- gametest/              # Debug: two AIs playing
|   |-- cardart/               # Pixel art generator
|   '-- forge/                 # Procedural card set generator
|
|-- internal/tui/              # Bubbletea TUI components
|-- web/                       # Browser UI (HTML/JS/WASM)
|-- data/                      # Scryfall JSON per set
'-- docs/                      # Comprehensive rules, tutorials
```

### Dependencies

- **Only external runtime dep:** `github.com/google/uuid`
- **UI:** Charmbracelet ecosystem (bubbletea, lipgloss, wish, ssh)
- **Codegen:** `github.com/dmarkham/enumer` for enum String/Parse methods

### Card Registration Pattern

```go
// cards/limited/creatures.go
func init() {
    registerCreatures()
}

func registerCreatures() {
    Register("Serra Angel", func() Card {
        return NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
            WithSubTypes("Angel"),
            WithKeyword(Flying),
            WithKeyword(Vigilance),
        )
    })
}
```

Each set package has a consistent structure:
- `creatures.go`, `spells.go`, `enchantments.go`, `artifacts.go`, `lands.go` -- card definitions
- `register.go` -- exports registration functions
- `test.go` -- blank imports to trigger `init()` execution

### Test Organization

- **Card tests** live in card packages: `cards/limited/correctness_test.go`, `cards/arabian/creatures_test.go`
- **Engine tests** in `pkg/mage/*_test.go` and `pkg/mage/gametest/*_test.go`
- All tests use `gametest.TestGame` DSL for scenario-based testing

### Build System (Makefile)

| Target | Command |
|--------|---------|
| `test` | `go test ./... --count=10` |
| `build` | `go build ./...` |
| `vet` | `go vet ./...` |
| `lint` | `golangci-lint run ./... && staticcheck ./...` |
| `wasm` | `GOOS=js GOARCH=wasm go build -o web/mage.wasm ./cmd/wasm/` |
| `clean` | Remove generated binaries and coverage files |

### Code Generation

All enums in `pkg/mage/core/` use `//go:generate enumer` directives, producing `_enumer.go` files with `String()`, `Parse()`, and other methods automatically.

### Set Implementation Pipeline

```
Scryfall API -> cmd/fetchset -> data/*.json -> cmd/genset -> cards/{set}/*.go -> TDD -> tests pass
```

---

## 4. Physical View

The physical view describes deployment topology and how the system maps to infrastructure.

### Deployment Targets

#### 4.1 Standalone TUI (Native Binary)

```
+--------------------+
| User Terminal      |
|--------------------|
| TUI Binary         |
|  - Game Engine     |
|  - Bubbletea UI    |
|  - AI Opponent     |
+--------------------+
```

- **Build:** `go build ./cmd/tui/`
- **Network:** None (single-machine, local play)
- **Players:** Human vs. AI

#### 4.2 SSH Multiplayer Server

```
+-----------------+     SSH (port 2222)    +------------------------+
| SSH Client 1    |----------------------->| SSH Server             |
| (Terminal)      |<-----------------------|  - Lobby (matchmaking) |
+-----------------+                        |  - Game Engine         |
                                           |  - Per-session TUI     |
+-----------------+     SSH (port 2222)    |  - AI Engine           |
| SSH Client 2    |----------------------->|  - Host keys in .ssh/  |
| (Terminal)      |<-----------------------|                        |
+-----------------+                        +------------------------+
```

- **Build:** `go build ./cmd/server/`
- **Port:** Configurable via `PORT` env var (default 2222)
- **Protocol:** SSH + PTY
- **Concurrency:** One goroutine per session; lobby mutex protects game slots
- **Players:** Human vs. Human or Human vs. AI

#### 4.3 Browser-Based (WebAssembly)

```
+-------------------------------+     HTTP/HTTPS    +-------------------+
| Web Browser                   |<----------------->| Static HTTP Server|
|  - index.html                 |                   |  - mage.wasm      |
|  - game.js (33KB UI bridge)  |                   |  - game.js        |
|  - wasm_exec.js (Go runtime) |                   |  - index.html     |
|  - mage.wasm (Game Engine)   |                   +-------------------+
|  - AI (runs in WASM)         |
+-------------------------------+
```

- **Build:** `make wasm`
- **Serving:** Any static HTTP server (e.g., `python3 -m http.server`)
- **API Bridge:** Go exposes global JS functions via `syscall/js`:
  - `mageGetCardList()` -- all registered cards
  - `mageStartGame(deckJSON, callbacks...)` -- initiate game
  - `mageSendAction(actionJSON)` -- player actions
  - `mageSendChoice(choiceJSON)` -- modal choices
- **Players:** Human (browser) vs. AI (in WASM)
- **Note:** Game engine runs entirely client-side; server only serves static files

### Data Flow Comparison

| Dimension | TUI | SSH Server | Browser/WASM |
|-----------|-----|-----------|--------------|
| **Network** | None | SSH (2222) | HTTP |
| **Players** | 1 Human + 1 AI | 1-2 Humans + optional AI | 1 Human + 1 AI |
| **UI** | Bubbletea | Bubbletea + Wish | HTML/CSS/JS |
| **Game Loop** | Goroutine | Goroutine per game | Goroutine (WASM) |
| **State Sync** | Channels | Channels | Channels -> JS callbacks |
| **Persistence** | None | None | None |
| **Scale** | Single user | Multiple concurrent sessions | Single session per tab |

### Support Tooling

| Binary | Purpose | I/O |
|--------|---------|-----|
| `fetchset` | Fetch card data from Scryfall API | HTTP -> JSON file |
| `fetchcatalog` | Bulk fetch 37+ sets | HTTP -> data/catalog/*.json |
| `genset` | Generate Go stubs from JSON | JSON -> Go source files |
| `gametest` | Two AIs playing (debug) | stdout game log |
| `cardart` | Pixel art card images | PNG/SVG/ANSI output |
| `forge` | Procedural card generation | stdout JSON |

### Configuration

- **`PORT`** env var: SSH server port
- **`FETCHSET_SKIP_TLS`** env var: skip TLS for Scryfall (sandbox workaround)
- **Host keys:** `.ssh/server_ed25519` (auto-generated)
- **No Docker, no external databases, no persistence layer**

---

## +1. Scenarios View

The scenarios view describes key use cases that exercise and validate the other four views.

### Scenario 1: Playing a Game (Terminal)

**Actor:** Human player

1. Launch `./tui`, select deck archetype and AI personality
2. Game creates `mage.Game` with HumanPlayer + AI player
3. `RunGameLoop` starts in goroutine, communicates via channels
4. Each priority window: game sends `GameMsg` -> TUI renders -> human selects action -> `PriorityAction` sent back
5. AI takes turns via `aiPlayer.GetPriorityAction()` (heuristic, minimax, or adaptive)
6. Game resolves spells, combat, triggers, SBAs per turn structure
7. Winner announced when a player reaches 0 life or other loss condition

**Views exercised:** Logical (all types), Process (turn loop, priority, channels), Physical (TUI binary)

### Scenario 2: Playing via SSH

**Actors:** Two remote human players

1. Server listens on port 2222; two players SSH in
2. Lobby matches players into a game session
3. Both get Bubbletea TUI instances backed by shared `mage.Game`
4. Active player's TUI shows interactive options; opponent sees read-only view
5. Actions broadcast to both TUI instances via channel forwarding
6. Disconnection: closed channel detected, remaining player can continue or concede

**Views exercised:** Process (goroutine per session, channel sync), Physical (SSH server topology)

### Scenario 3: Implementing a New Card (TDD)

**Actor:** Developer

1. Find Oracle text on Scryfall
2. Read `pkg/mage/doc.go` for available effects, triggers, and constructors
3. Write failing test using `gametest.TestGame` DSL:
   ```go
   g := gametest.NewTestGame(t)
   g.AddCard(ZoneBattlefield, PlayerA, "Lightning Bolt")
   g.CastSpell(1, PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
   g.StopAt(1, EndStep)
   g.Execute()
   g.AssertLife(PlayerB, 17)
   ```
4. Implement card registration with appropriate constructors and effects
5. Run test: `go test ./cards/limited -run TestLightningBolt`
6. Iterate until passing; run full suite: `go test ./cards/...`

**Views exercised:** Logical (Card, Effect, Ability types), Development (test DSL, package structure)

### Scenario 4: Implementing a New Set

**Actor:** Developer

1. **Fetch:** `go run ./cmd/fetchset -o data/DRK.json DRK`
2. **Scaffold:** `go run ./cmd/genset "The Dark" data/DRK.json cards/thedark/`
   - Generates `creatures.go`, `spells.go`, etc. with Oracle text comments and `// TODO: implement` stubs
   - Generates `test.go` with blank imports for init() registration
3. **Implement:** TDD loop per card (or use `/implement-set` skill for automation)
4. **Validate:** `/validate-set thedark DRK` audits completeness, test coverage, Oracle fidelity

**Views exercised:** Development (genset pipeline, package layout), Logical (Registry, Card factories)

### Scenario 5: Testing Card Interactions

**Actor:** Developer

1. Set up board state with multiple cards from different sets
2. Queue actions across multiple turns:
   ```go
   g.AddCard(ZoneBattlefield, PlayerA, "Sengir Vampire")
   g.AddCard(ZoneBattlefield, PlayerB, "White Knight")
   g.Attack(1, PlayerA, "Sengir Vampire")
   g.Block(1, PlayerB, "White Knight", "Sengir Vampire")
   g.StopAt(2, Upkeep)
   g.Execute()
   g.AssertGraveyardCount(PlayerB, "White Knight", 1)
   g.AssertCounterCount(PlayerA, "Sengir Vampire", Plus1Plus1, 1)
   ```
3. Assertions verify interactions: combat damage, triggers, SBAs, continuous effects

**Views exercised:** Process (combat, triggers, SBAs), Logical (abilities, effects, layers)

### Scenario 6: Adding an Engine Feature

**Actor:** Developer

1. Card implementation reveals missing engine support (e.g., new event type, new replacement effect)
2. Design minimal addition: new `EventType`, new `GameMutator` method, or new `ContinuousEffect` variant
3. Write engine tests in `pkg/mage/gametest/mechanics_test.go`
4. Implement in core engine files
5. Verify engine tests pass
6. Update `pkg/mage/doc.go` with new API
7. Use feature in card implementations
8. Run full suite: `go test ./...`

**Views exercised:** All views (core engine touches everything)

### Scenario Summary

| Scenario | Key Actors | Entry Point | Primary Views |
|----------|-----------|-------------|---------------|
| Play (TUI) | Human, AI | `cmd/tui/` | Process, Physical |
| Play (SSH) | 2 Humans | `cmd/server/` | Process, Physical |
| Implement Card | Developer | card package | Logical, Development |
| Implement Set | Developer | `cmd/fetchset`, `cmd/genset` | Development |
| Test Interactions | Developer | test files | Process, Logical |
| Engine Feature | Developer | `pkg/mage/` | All |

---

## Cross-Cutting Concerns

### Determinism
The engine is fully deterministic given the same player decisions. No randomness exists in game logic (shuffling is external). This enables reliable testing and potential replay support.

### Extensibility
New cards require zero engine changes when existing effects cover the mechanics. The composable Effect/Ability/Cost/Target/Filter system handles the vast majority of cards. When the engine must be extended, the GameMutator interface boundary ensures changes are explicit and auditable.

### Testability
The `gametest.TestGame` DSL enables scenario-based tests that read like game narratives. The non-interactive mode (no `OnPriority` handler) allows atomic test execution without channel overhead. Over 800 tests cover card interactions, combat, triggers, and edge cases.

### Portability
A single Go codebase compiles to native binary (TUI/server), WebAssembly (browser), with no code duplication. The interactive layer abstracts over terminal I/O (Bubbletea channels) and browser I/O (JS callbacks), keeping the engine UI-agnostic.
