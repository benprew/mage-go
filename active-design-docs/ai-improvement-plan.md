# AI Improvement Plan

## Current State

The AI in `pkg/mage/interactive/` is a solid greedy v1:

- **`AIStrategy` interface**: `PriorityAction`, `Attackers`, `Blockers`
- **`HeuristicStrategy`**: Personality-driven boolean flags (AttackAll, HoldInstants, TargetFace, BlockPowerThreshold)
- **`AdaptiveStrategy`**: Switches aggressive/defensive based on `DefaultEvaluator` sign
- **`SequentialStrategy`**: Tries strategies in order, uses first non-pass
- **`StateEvaluator`** (evaluator.go): Weighted life + creatures + hand + lands
- **`spellValue`** (eval.go): Spell ordering by effect properties + board state
- **No game cloning**: Existing `undoSnapshot` is too shallow (hand, mana, tapped, battlefield length only)
- **No search**: Every decision is single-ply heuristic

### Key Files

| File | Lines | Purpose |
|---|---|---|
| aiplayer.go | 602 | Strategies, Personalities, profitableToAttack, autoSelectTargets |
| eval.go | 134 | spellValue, ThreatPerMana, spellIsWorthless, evalCreature |
| evaluator.go | 247 | DefaultEvaluator, position scoring, keyword/ability bonuses |
| types.go | 220 | PriorityAction, Personality, AIStrategy interface, Choice types |

---

## Implementation Order

```
Phase 0  →  Phase 2  →  Phase 3  →  Phase 1  →  Phase 4  →  Phase 5
Quick Wins  Weights     Eval        Clone       Search      Response/Combat
```

Phase 0 gives immediate play quality. Phase 2+3 make even heuristic play stronger. Phase 1+4 add lookahead. Phase 5 polishes interaction timing.

---

## Phase 0: Quick Wins (no search, no new infrastructure)

### 0A: Lethal Detection

Before every priority/attack/block decision, check "do I have lethal?" and "does opponent have lethal?"

**Add to `eval.go`:**

```go
type LethalInfo struct {
    IHaveLethal      bool
    TheyHaveLethal   bool
    MyBoardDamage    int           // total damage I can push through
    TheirBoardDamage int
    LethalAttackers  []uuid.UUID   // minimal set that kills opponent
}

func CalculateLethal(g mage.GameReader, playerID uuid.UUID) LethalInfo
```

**Logic:**
- Sum power of creatures with evasion (flying, unblockable, fear, landwalk) vs opponent life
- For "they have lethal": sum their evasive attackers vs our life, check if we have blockers
- Account for trample: power minus best blocker toughness goes through

**Integration in `aiplayer.go`:**
- `Attackers()`: if iHaveLethal, attack with exactly the lethal set (not everything — avoid walking into a trick when you have exact lethal with evasion)
- `Blockers()`: if theyHaveLethal, block to survive (assign blockers to prevent lethal damage, even bad trades)
- `PriorityAction()`: if theyHaveLethal and we have removal, prioritize removing their biggest threat over any other play

**Tests:** Board states where lethal exists/doesn't. AI attacks for lethal. AI blocks to prevent lethal. AI prioritizes removal when facing lethal.

### 0B: Race Calculation

Compare "clock" (turns to kill) for both players.

**Add to `eval.go`:**

```go
type RaceInfo struct {
    MyClock    int // turns until I kill opponent (math.MaxInt if never)
    TheirClock int // turns until opponent kills me
    Racing     bool // true if both clocks < 5
}

func CalculateRace(g mage.GameReader, playerID uuid.UUID) RaceInfo
```

**Logic:**
- My clock = ceil(opponent life / my expected damage per attack)
- Their clock = ceil(my life / their expected damage per attack)
- Expected damage = sum of evasive power + (non-evasive power - expected blocks)

**Integration:**
- `Attackers()`: if my clock < their clock, race (attack aggressively, don't hold back). If their clock < my clock, hold back blockers.
- `Blockers()`: if racing favorably, only chump-block lethal. If behind, block aggressively.

**Tests:** Various clock comparisons, verify attack/block decisions change.

### 0C: Mana Curve Awareness

Prefer casting on-curve. Don't waste turn 5 on a 2-drop when you have a 5-drop.

**Modify `aiplayer.go` spell selection:**

```go
func manaCurveBonus(card mage.Card, availableMana int, handCMCs []int) float64
```

- Count untapped mana sources
- Penalize spells that leave significant mana unused when higher-CMC options exist in hand
- Bonus for spending all/most mana this turn
- If only low-CMC spells in hand, no penalty (cast what you have)

**Tests:** Hand with 2-drop + 5-drop, 5 mana → cast 5-drop. Hand with only 2-drops → cast 2-drop.

### 0D: Ability Cost Analysis

Score activated abilities by cost vs effect quality. Currently `abilityBonus` gives flat +1.

**Modify `evaluator.go`:**

```go
func abilityQuality(ab mage.ActivatedAbility) int {
    // Tap-to-draw: +5
    // Tap-to-deal-damage: +4
    // Tap-for-mana (any color): +3
    // Tap-for-mana (fixed): +2
    // Expensive activated (+3 CMC): +1
    // Free activated (no mana): +3
}
```

**Also in `PriorityAction`:** Consider activating abilities (currently AI only activates mana abilities). Check `g.GetActivatableAbilities(playerID)` and return `ActionActivateAbility` when valuable. Need target selection for activated abilities via `autoSelectTargets`.

**Tests:** Creature with tap-to-ping vs creature with {5}: +1/+1 → AI values pinger higher.

---

## Phase 2: Weighted Personality System

**Dependencies:** None. Independent of cloning.

### Replace booleans with continuous weights

**New type in `types.go`:**

```go
type WeightedPersonality struct {
    Name string

    // Evaluation weights (flow into StateEvaluator)
    LifeWeight      float64 // Aggro: low (life is a resource), Control: high
    BoardWeight     float64 // Midrange: high
    CardWeight      float64 // Control: high, Aggro: low
    ManaWeight      float64 // Tempo: high
    TempoWeight     float64 // importance of untapped mana

    // Decision weights (0.0 to 1.0 continuous)
    Aggression      float64 // 1.0 = attack everything, 0.0 = only profitable
    BlockThreshold  float64 // 0.0 = block everything, 1.0 = never block
    HoldInstants    float64 // 0.0 = cast immediately, 1.0 = always hold
    TargetFace      float64 // 0.0 = always target creatures, 1.0 = always go face
    CurvePreference float64 // 0.0 = cheapest first, 1.0 = most expensive first
}
```

**Preset conversions (backward compatible):**

| Preset | Life | Board | Card | Mana | Tempo | Aggression | Block | Hold | Face | Curve |
|--------|------|-------|------|------|-------|------------|-------|------|------|-------|
| Aggro | 1.0 | 3.0 | 0.5 | 0.5 | 0.5 | 1.0 | 0.7 | 0.0 | 0.3 | 0.0 |
| Control | 4.0 | 1.0 | 3.0 | 1.0 | 2.0 | 0.0 | 0.0 | 1.0 | 0.0 | 1.0 |
| Midrange | 2.0 | 3.0 | 2.0 | 1.0 | 1.0 | 0.7 | 0.3 | 0.0 | 0.0 | 1.0 |
| Tempo | 1.5 | 2.0 | 1.5 | 2.0 | 3.0 | 0.8 | 0.3 | 0.7 | 0.0 | 0.0 |
| Burn | 0.5 | 1.0 | 0.5 | 0.5 | 0.5 | 1.0 | 0.95 | 0.0 | 1.0 | 0.0 |

**Migration:**
- Old `Personality` struct stays, with a `ToWeighted() WeightedPersonality` method
- `HeuristicStrategy` stores `WeightedPersonality` internally
- Existing constructors (`NewAggroAI`, etc.) convert via `ToWeighted()`
- All decision methods use continuous weights: `if rand.Float64() < p.Aggression` or threshold comparisons

**Key behavior changes:**
- `profitableToAttack` uses `Aggression` weight: at 0.5, only attack if trade is somewhat favorable; at 1.0, always attack; at 0.0, only attack if guaranteed profitable
- `BlockPowerThreshold` becomes continuous: `int(p.BlockThreshold * maxPowerOnBoard)`
- `HoldInstants` at 0.5 means hold only if instant value > 2x sorcery-speed play value
- `CurvePreference` interpolates between cheapest-first and expensive-first sort

**Tests:** Verify converted presets produce identical behavior to existing boolean personalities on reference game states. Test intermediate weight values.

---

## Phase 3: Better Board Evaluation

**Dependencies:** Phase 2 (weighted personality for tunable parameters).

### 3A: Tempo Scoring (`evaluator.go`)

Currently `defaultEvaluate` gives +1 per untapped land. Make this configurable and weight by `TempoWeight`:

```go
score += untappedMana * int(personality.TempoWeight * 2)
```

Untapped mana represents options. A 3/3 with 3 open mana is scarier than a 3/3 tapped out.

### 3B: Role Detection (new `eval_roles.go`)

Classify permanents into roles for weighted evaluation:

```go
type PermanentRole int
const (
    RoleThreat  PermanentRole = iota // power >= 3, or evasion
    RoleUtility                       // tap abilities (draw, removal, mana)
    RoleEngine                        // incremental advantage (Howling Mine, etc.)
    RoleMana                          // lands, mana creatures, mana artifacts
    RoleDefense                       // walls, high-toughness blockers
)

func ClassifyPermanent(p *mage.Permanent) PermanentRole
```

Weight evaluation by role × personality:
- Aggro values Threats highly, doesn't care about Engines
- Control values Engines and Utility, less about raw Threats
- Everyone values Mana, weighted by ManaWeight

### 3C: Lethal-on-Board in Eval

Integrate Phase 0A `CalculateLethal` into `defaultEvaluate`:
- If I have lethal on board: return large positive score (near-win)
- If opponent has lethal on board: subtract large penalty
- This makes the search (Phase 4) naturally find lethal lines

### 3D: Board Texture

Score card advantage and quality:
- Hand count weighted by `CardWeight`
- Quality of hand: sum of `spellValue` for castable spells (don't just count cards — a hand of 7 lands is worse than a hand of 3 spells)
- Mana advantage: count lands, weighted by `ManaWeight`

### 3E: Ability Quality in Eval

Replace flat +1 per activated ability with `abilityQuality()` from Phase 0D:
- Tap-to-draw creature is worth ~5 more than a vanilla of same P/T
- Tap-to-ping creature gets +4
- Expensive pump (+1/+1 for {3}) gets +1

**Tests:** Reference board positions with known evaluations. Ensure role detection doesn't regress existing tests.

---

## Phase 1: Game Cloning Infrastructure

**Dependencies:** None (but Phase 4 requires this).

### Why Clone, Not Undo/Snapshot

1. Existing `undoSnapshot` is too shallow — only hand, mana, tapped, battlefield length
2. Undo requires perfectly symmetric save/restore for every field. Missing one = subtle bugs.
3. Minimax explores a tree, not linear sequence — need multiple branches alive simultaneously
4. Continuous effects re-apply from scratch each `Apply()` cycle, so cloning the effect list suffices

### New Files

**`pkg/mage/clone.go`** — `Game.Clone() *Game` and helpers:

```go
func (g *Game) Clone() *Game

func clonePermanent(p *Permanent) *Permanent
func cloneCombat(c *Combat) *Combat
func cloneStack(s *Stack) *Stack
func cloneEffectManager(em *EffectManager) *EffectManager
func cloneGameRules(gr *GameRules) *GameRules
func cloneDamageSystem(ds *DamageSystem) *DamageSystem
```

**`pkg/mage/clone_test.go`** — correctness and isolation tests

**`pkg/mage/search_player.go`** — `SearchPlayer` wrapping `BasePlayer` state:

```go
type SearchPlayer struct {
    *BasePlayer // embedded, same UUID, copied state
}
```

Stubs all interactive choice methods with reasonable defaults:
- `ChooseTargets` → first valid target
- `DeclareAttackers` → empty (pass)
- `DeclareBlockers` → empty (pass)
- `ChooseDiscard` → cheapest card
- `ChooseManaColor` → first available

### Cloning Strategy by State Category

**Shared (immutable during play, pointer-shared):**
- Card objects (abilities, effects set at construction)
- Effect interfaces (ContinuousEffect, ReplacementEffect are stateless — receive `*Game` as parameter)

**Deep copied:**
- Permanents: new struct, same Card ref, copy Tapped/Damage/Counters/baseAttrs/grantedAttrs/powerBonus/toughBonus/AttachedTo/Attachments/RuntimeAbilities
- Players: clone BasePlayer (same UUID!), copy hand/graveyard/library/ante slices (sharing Card refs), SnapshotPool + new pool for mana
- EffectManager: copy effects/replacements/cycleReplacements slices (sharing interface values), clone GameRules (copy all maps), clone DamageSystem (copy reflection map)
- Stack: clone []*StackObject slice
- Combat: clone Groups/Attackers/FirstStruck/Bands
- All UUID-keyed maps: two-level copy for nested maps

**Nil'd in clone:**
- `OnPriority`, `AfterPriorityAction`, `BeforeStackResolve` (function pointers — search clones never call back to UI)

**Recomputed:**
- `attrDeltas`, `blockPairRestrictions` — recomputed each `Apply()` cycle, start empty

### Risk: ReplacementEffect Mutable State

Some replacement effects hold mutable state (e.g. `preventionShieldReplacement.remaining`). These need structural cloning, not pointer-sharing.

**Options:**
1. Add `Clone() ReplacementEffect` to the interface
2. Type-switch on known types in `cloneEffectManager`

Option 1 is cleaner. Each of the 17 built-in replacement types gets a `Clone()` method (trivial — they're small structs).

### Performance Target

< 50μs per clone for typical mid-game board (10-15 permanents, 2-3 stack objects). Profile and optimize if needed.

### Tests

- Clone mid-turn game, verify all fields match
- Mutate clone, verify original unchanged (isolation)
- Clone with active effects/combat/stack, verify correctness
- Benchmark clone cost

---

## Phase 4: Minimax with Alpha-Beta Pruning

**Dependencies:** Phase 1 (game cloning), Phase 3 (evaluation function).

### New Files

**`pkg/mage/interactive/search.go`:**

```go
type SearchConfig struct {
    MaxDepth  int           // search depth in plies (default 3)
    MaxNodes  int           // node budget (default 10000)
    TimeLimit time.Duration // per-decision time budget (default 200ms)
    Evaluator StateEvaluator
}

type SearchStrategy struct {
    Config    SearchConfig
    Weights   WeightedPersonality
    Fallback  *HeuristicStrategy // for non-search decisions, timeouts
}

func (s *SearchStrategy) PriorityAction(...) PriorityAction
func (s *SearchStrategy) Attackers(...) []uuid.UUID
func (s *SearchStrategy) Blockers(...) []mage.BlockAssignment
```

**`pkg/mage/interactive/movegen.go`:**

```go
type Move struct {
    Action    PriorityAction
    Attackers []uuid.UUID
    Blockers  []mage.BlockAssignment
}

func GenerateMoves(g *mage.Game, playerID uuid.UUID) []Move
func sortMovesByHeuristic(moves []Move, g *mage.Game, playerID uuid.UUID)
```

### Minimax Core

```go
func (s *SearchStrategy) minimax(g *mage.Game, depth, alpha, beta int,
    maximizing bool, playerID uuid.UUID, nodes *int) int {

    if depth == 0 || g.IsGameOver() || *nodes >= s.Config.MaxNodes {
        return s.Config.Evaluator(g, playerID)
    }

    moves := GenerateMoves(g, currentPlayer(g))
    sortMovesByHeuristic(moves, g, playerID) // best-first for pruning

    if maximizing {
        value := math.MinInt
        for _, move := range moves {
            clone := g.Clone()
            applyMove(clone, move)
            v := s.minimax(clone, depth-1, alpha, beta, false, playerID, nodes)
            if v > value { value = v }
            if v > alpha { alpha = v }
            if beta <= alpha { break } // prune
            *nodes++
        }
        return value
    }
    // symmetric for minimizing
}
```

### Move Generation Pruning

Attack/block combinatorics explode (2^N for N creatures). Prune to tractable set:

**Attacker subsets generated:**
- Attack with all
- Attack with none
- Each individual creature solo
- All evasive creatures only
- All non-evasive creatures only
- Top-3 by `evalCreature` score

**Blocker assignments generated:**
- No blocks
- Best single blocker per attacker (greedy, like current)
- Gang-block the biggest attacker (Phase 5 enhancement)

### Applying Moves to Clones

`applyMove(g *mage.Game, move Move)` executes on cloned game:
- Spell cast → `g.CastSpell(cardID, targets)` (need internal API or public method)
- Land play → `g.PlayLand(cardID)`
- Ability activation → `g.ActivateAbility(permID, targets)`
- Pass → advance to next priority/step

**Risk:** Engine's full game loop (stack resolution, triggers, SBAs) fires during search. Triggered abilities requiring player choices hit `SearchPlayer` stubs. These stubs return defaults (first permanent, cheapest discard, etc.) — imperfect but functional.

**Mitigation:** SearchPlayer defaults are "good enough" for 2-3 ply search. Deeper search or complex triggered chains may need a "simplified resolution" mode later.

### Integration

- `SearchStrategy` implements `AIStrategy` interface
- `PriorityAction()` runs minimax, returns best move
- Falls back to `HeuristicStrategy` on timeout or degenerate positions
- `NewSearchAI(config, weights)` constructor

### Tests

- Simple position: AI finds lethal that requires removal → attack sequence (heuristic misses this)
- AI casts bolt on blocker, then attacks for lethal (2-ply)
- Performance: 2-ply < 100ms, 3-ply < 500ms for typical boards
- Correctness: search result ≥ heuristic result on reference positions

---

## Phase 5: Instant/Response Timing and Combat Math

**Dependencies:** Phase 4 (search), Phase 0 (lethal/race).

### 5A: Stack Response Evaluation

When opponent casts a spell or attacks, use search to evaluate "respond with instant X" vs "don't respond":

- Opponent targets our creature with removal → search: pump/protect worth it?
- Opponent declares attackers → search: combat trick on our blocker worth it?
- Search depth 1-2 for responses (fast — small move set)

### 5B: Hold-Back Heuristic

Replace binary `HoldInstants` with:

```go
func holdBackValue(instant mage.Card, g *mage.Game, personality WeightedPersonality) float64
```

Compare best instant's expected response value vs best sorcery-speed play value:
- If instant EV > sorcery EV × personality.HoldInstants → pass, hold mana open
- Continuous personality weight makes this tunable

### 5C: Multi-Blocker Coordination (Gang Blocks)

Enhance `Blockers()`:

1. Identify attackers no single blocker can kill
2. Find blocker combinations where total power ≥ attacker toughness
3. Evaluate trade: losing 2 small creatures worth killing their big threat?
4. Use `evalCreature` scores: gang-block if attacker score > sum of blocker scores

Add gang-block combinations to movegen for search.

### 5D: Damage Assignment Optimization

When multiple blockers on one attacker:
- Order blockers by toughness (kill most with first strike)
- Assign lethal to each blocker in order that maximizes kills

### 5E: Race-Informed Attack/Block

Integrate Phase 0B `CalculateRace` into all combat decisions:
- Racing favorably: attack aggressively, chump-block only if it saves a turn
- Racing unfavorably: hold back, block everything, trade up
- Tied: maintain board parity, trade for value

---

## Dependency Graph

```
Phase 0 (Quick Wins) ──────────────────────┐
    │                                       │
Phase 2 (Weighted Personality)              │
    │                                       │
Phase 3 (Better Eval) ◄────────────────────┘
    │
Phase 1 (Game Cloning) ────────┐
    │                          │
Phase 4 (Minimax Search) ◄────┘
    │
Phase 5 (Response + Combat)
```

Phases 0, 1, and 2 are independent of each other and can be worked in parallel.
Phase 3 benefits from Phase 2 weights.
Phase 4 requires Phase 1 (cloning) and Phase 3 (eval).
Phase 5 requires Phase 4 (search).
