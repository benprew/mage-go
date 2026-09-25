# Composable Card DSL: Architecture & Invariant Specification

This document is the definitive architectural specification and engine invariant contract for `mage-go`'s composable Domain-Specific Language (DSL).

* **Card Authoring Guide**: See [`docs/dsl-reference.md`](../dsl-reference.md) for constructors, verbs, and syntax.
* **Migration Tracker**: See [`composable-dsl-implementation.md`](composable-dsl-implementation.md) for outstanding card migrations and active blockers.
* **Engine Gap Analysis**: See [`composable-dsl-engine-audit.md`](composable-dsl-engine-audit.md) for verified engine services and confirmed gaps.
* **Engine Rules Contracts**: See [`docs/engine-rules-contracts.md`](../engine-rules-contracts.md) for layer evaluation, LKI, and replacement pipelines.

---

## 1. Architectural Scope & Boundary

The card DSL is a **strictly typed authoring surface over existing engine services**. It is **not** an abstract syntax tree (AST) interpreter, an execution engine, or a replacement for rules subsystems.

```
┌────────────────────────────────────────────────────────┐
│ Cards (cards/*)                                        │
│ Import ONLY: pkg/mage/dsl and pkg/mage/core            │
└──────────────────────────┬─────────────────────────────┘
                           │ Typed Constructors & Sealed Roles
┌──────────────────────────▼─────────────────────────────┐
│ Public DSL Surface (pkg/mage/dsl)                      │
│ Immutable declarations, typed bindings, option groups  │
└──────────────────────────┬─────────────────────────────┘
                           │ Adapter Layer & Inspection Graph
┌──────────────────────────▼─────────────────────────────┐
│ Game Engine Core (pkg/mage)                            │
│ Game, ZoneSystem, TurnSystem, EffectManager, Layers    │
└────────────────────────────────────────────────────────┘
```

### Core Invariants

1. **One Concept, One Function**: Combinators represent a single semantic concept. Verbs do not combine conditions, subjects, or contexts (e.g., `WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This))` instead of composite helpers).
2. **Compile-Time Safety over Runtime Panics**: Go's type system rejects category errors at compile time via sealed marker interfaces.
3. **Static Registration Validation**: Value-dependent constraints (definite assignment, scoping, label uniqueness, non-empty collections) fail deterministically during card registration at boot time (`init`), never during gameplay.
4. **Engine Non-Replacement**: Missing DSL syntax does not imply missing engine functionality. DSL adapters map directly to existing engine primitives; proposals for new engine behavior require separate regressions and minimal changes.
5. **Exact Oracle Fidelity**: Card-facing DSL declarations preserve exact Magic: The Gathering Comprehensive Rules (CR) behavior. Declarations never simplify rules logic for authoring convenience.

---

## 2. Validation & Safety Matrix

Errors are caught at the earliest possible lifecycle boundary:

| Failure Class | Validation Boundary | Mechanism | Examples |
| :--- | :--- | :--- | :--- |
| **Category Error** | Compile-Time | Sealed Go marker interfaces | Passing `PlayerRef` to `Destroy`; using `ResolvingEffect` as a cost; installing `ResolvingEffect` as static rule; using `EventPattern` as repeatable `StateCondition`. |
| **Graph Topology & Value Error** | Card Registration (`TestRegisteredDefinitions`) | AST Inspector / Definite Assignment Visitor | Unbound variable read; duplicate binding label within ability; empty `Sequence` or mode list; non-dominating producer; invalid min/max target counts. |
| **Rules State Error** | Ability Resolution | Game Engine Runtime | Target illegality at resolution; impossible action continuation; target disappearance. |

---

## 3. Type Model & Sealed Roles

### 3.1 Effect Categories

Effects are strictly classified by their runtime execution lifecycle. They do **not** share a universal execution interface:

* `ResolvingEffect`: Executes once within a spell, activation, or trigger resolution context. Mutates the resolution frame and engine state.
* `ContinuousEffect`: Evaluated dynamically within the continuous-effect and layer system (Layers 1–7).
* `ReplacementEffect`: Intercepts and transforms proposed rules events before they occur (CR 614–616).

### 3.2 Sealed Roles & Cardinality

Public DSL interfaces represent closed domain roles using unexported marker methods. External card packages cannot implement these interfaces directly; they compose existing constructors or use typed escape hatches.

Cardinality is enforced in the type system:

```go
type PermanentRef interface { permanentRef() }   // Singular permanent
type PermanentSet interface { permanentSet() }   // Collection of permanents
type PlayerRef    interface { playerRef() }      // Singular player
type PlayerSet    interface { playerSet() }      // Collection of players
type CardRef      interface { cardRef() }        // Card in non-battlefield zone
type CardSet      interface { cardSet() }        // Collection of cards
type StackObjectRef interface { stackObjectRef() }
```

* `RULE (No Silent First-Member)`: A collection selector (`PermanentSet`) is never implicitly converted to a singular reference (`PermanentRef`). Operations taking sets (`DestroyEach`) are syntactically distinct from operations taking singular references (`Destroy`).

---

## 4. Object Identity & Reference Semantics

The DSL strictly respects the object identity model defined by CR 400.7:

1. **Incarnations vs. Card IDs**:
   * `Permanent.ID()` returns the underlying physical card UUID.
   * `PermanentRef` resolves **only** the specific battlefield incarnation (`Permanent.incarnationID`) captured at declaration or resolution.
2. **Zone Transitions**:
   * Moving a card between zones (battlefield, graveyard, hand, exile) terminates its object lifetime and creates a new incarnation. Old references become absent.
   * `RULE (Absent Reference Handling)`: Resolving an absent reference returns an explicit absent outcome. Resolving effects do nothing when their subject no longer exists, unless Last-Known Information (CR 608.2h) applies.
3. **Phasing**:
   * Phasing out preserves the battlefield incarnation. A phased-out permanent retains its identity and attached references when it phases back in.
4. **Contextual Subjects**:
   * `This`: Resolves the ability's source object in its current context (battlefield incarnation for permanents; damage source with LKI for spells/abilities).
   * `You`: Evaluates to the stored resolving controller for spells and abilities. For static abilities and trigger conditions, it dynamically resolves the current or pre-event controller.

---

## 5. Typed Bindings & Scoping

Target declarations and resolution snapshots create immutable typed bindings rather than mutating untyped string maps:

```go
creature := TargetPermanent("creature", IsCreature)
player   := TargetPlayer("player", AnyPlayer)

Activate(
    Costs(Mana("{1}{R}")),
    Sequence(
        DealDamage(This, creature, Fixed(2)),
        GainLife(player, Fixed(2)),
    ),
    WithTargets(creature, player),
)
```

### Binding Lifecycle Invariants

* `INVARIANT (Lexical Scope)`: A target binding is valid only within the ability or mode that declares it. Registration validation rejects references to targets declared in unrelated abilities or inactive modes.
* `INVARIANT (Instance Isolation)`: Binding declarations are immutable templates. Values live in a dedicated execution frame per spell cast, ability activation, or trigger resolution. Cloned games copy active frames without sharing mutable state. Retargeting mutates only the copied instance frame.
* `INVARIANT (Definite Assignment & Dominance)`: A binding producer (e.g., `Snapshot`, `TargetPermanent`) must strictly dominate every consumer along every control-flow path.
  * A value produced inside one branch of a conditional or an optional choice cannot be consumed outside that branch without an explicit fallback.
  * Loop-local bindings cannot escape the loop body without an explicit reduction or collection combinator.

---

## 6. Evaluation Lifecycle Contracts

Every DSL expression evaluates at a precise, rules-mandated point in time:

| Expression Role | Evaluation Point | Data Source | Invariants & Constraints |
| :--- | :--- | :--- | :--- |
| **Event Pattern** | Event dispatch | Event payload & event-time LKI | Evaluates looking back in time where required (e.g., death triggers, CR 603.10). |
| **State Trigger Condition** | After every state change (including during resolution) | Current derived game state | Must be **pure and deterministic**. No random choices, player choices, or mutations. |
| **Static Condition** | Continuous-effect evaluation | State at applicable layer | Evaluates within layer context; must not trigger recursive full layer evaluations. |
| **Intervening-If Condition** | (1) Trigger creation, and<br>(2) Trigger resolution | Current derived game state | CR 603.4: Rechecked at resolution. If false, ability is removed from stack with no effect. |
| **Target Declaration** | Announcement / Casting | Current game state | Validates legal targets, minimum/maximum counts, and divided values. |
| **Target Binding** | Resolution | Stored choice frame | Target legality is rechecked. Illegal targets are filtered out; spell resolves on remaining legal targets. |
| **Resolution Selector** | Effect execution | Active resolution frame | Resolves subjects dynamically or reads captured snapshot. |
| **Continuous Modification** | Dynamic per layer | Layer state & captured resolution values | Distinguishes CR 611.2c snapshots from CR 611.3a dynamic evaluations. |
| **Activation Condition** | Prior to activation | Current game state | Checked before cost payment and mode selection. |

### Continuous Effect Invariants (CR 611)

1. **Resolving Continuous Effects (CR 611.2c)**:
   * Effects created by resolving spells or abilities that modify characteristics or control snapshot their affected objects at resolution time (e.g., `Until(EndOfTurn, ModifyPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(1)))`).
   * Permanents entering the battlefield later in the turn are **not** affected.
2. **Static Continuous Effects (CR 611.3a)**:
   * Continuous effects generated by static abilities (e.g., `Static(While(Controls(...), ...))`) do **not** lock in a set of objects.
   * They continuously recalculate affected permanents at each layer evaluation.

---

## 7. Execution Boundaries & Action Results

### 7.1 Results vs. Events

An action result is a distinct typed outcome; it is **not** a simple boolean:

* Outcomes distinguish: (1) choice accepted, (2) cost paid, (3) action executed, and (4) quantitative outcome (e.g., actual damage dealt, actual cards drawn).
* Replaced actions return the outcome produced by the replacement effect while preserving the fact that the original instruction was attempted.
* Clauses like "if you do" depend on the cost/action outcome. Clauses like "when you do" create reflexive triggers rather than inline branches.

### 7.2 Partial & Impossible Actions (CR 609.3)

* If an effect attempts to perform an action on an illegal or absent object, or an action that is partially impossible, it performs as much as possible.
* Impossible rules actions return normal empty/partial outcomes and the sequence continues. They are **not** engine errors and do not abort execution.

### 7.3 Concurrency & Ordering

* `Sequence`: Executes children strictly in order. State-trigger checks occur after rules-visible instructions.
* `Batch Operations` (`DestroyEach`, etc.): Collect all matching candidates first, snapshot their pre-action state, and perform the action simultaneously. Intermediate states are not observable.
* `APNAP Turn Order (CR 101.4)`: For instructions requiring each player to make choices (e.g., "each player sacrifices a creature"), choices are made in Active Player, Non-Active Player order, and the resulting actions are executed simultaneously.

---

## 8. Subsystem Adapter Contracts

### 8.1 Casting & Activation (CR 601 / CR 602)

Spells and activated abilities execute within a transactional payment boundary:

1. **Announcement & Mode Selection**: Lock chosen modes, X value, alternative costs, and additional costs.
2. **Targeting**: Validate and record target bindings.
3. **Cost Calculation**: Compute total cost including reductions and increments.
4. **Payment Transaction**: Activate mana abilities and pay costs. Canceled transactions or failed payments roll back completely with zero state mutations or published event traces.
5. **Stack Placement**: Move object to the stack; choices become immutable on the stack instance.

### 8.2 Choices & Hidden Zones

* Private zone searches (library, hand) declare criteria and permit the choosing player to "fail to find" (CR 701.19b), even if matching cards are present. Public searches do not permit failure to find.
* Non-target choices specify: choosing player, candidates, optionality, visibility permissions (reveal vs. look), and clone-safe random selection.

### 8.3 Replacement & Prevention Pipelines (CR 614–616)

* A replacement effect intercepts a **proposed** action, not an event that has already occurred.
* DSL replacement combinators wrap `EffectManager.ApplyReplacements`. They do not re-implement replacement ordering or bypass CR 616 player choice.
* Prevention shields specify capacity, recipient, source scope, and duration, decrementing capacity as damage is replaced.

---

## 9. Registration Validation (`TestRegisteredDefinitions`)

Registration validation inspects 100% of registered card blueprints at startup. It enforces invariants that cannot be verified by Go's compiler:

```
[Card Registration] ──► [Inspect Definition Graph] ──► [Definite Assignment Visitor]
                                                               │
                                       ┌───────────────────────┴───────────────────────┐
                                       ▼                                               ▼
                              [Report All Errors]                              [Registration OK]
                               (Card, Path, Rule)
```

Validation rejects card blueprints containing:

1. **Topology Defects**: Empty `Sequence`, empty mode list, or empty cost groups where prohibited.
2. **Scoping Violations**: A target binding referenced outside the declaring ability or mode.
3. **Definite Assignment Failures**: A binding read before write, or produced conditionally without being dominated along all execution paths.
4. **Diagnostic Collisions**: Duplicate diagnostic labels within the same ability scope.
5. **Cardinality & Range Errors**: Target declarations with `min > max`, or incompatible divided-value declarations.
6. **Mutually Exclusive Options**: Conflicting activation or timing constraints attached to the same ability.
7. **Nil Implementations**: `nil` interfaces hidden behind role types.
8. **Invalid Durations**: Continuous effects installed without a valid duration or layer specification.
9. **Lifecycle Mismatches**: Escape-hatch functions declaring incompatible lifecycles (e.g., an effect mutating state inside a `StateConditionFunc`).

---

## 10. Package & Architectural Boundaries

To preserve strict separation of concerns, repository packages adhere to strict import boundaries:

```go
// Allowed imports for card packages (cards/*):
import (
    "github.com/benprew/mage/pkg/mage/dsl"
    "github.com/benprew/mage/pkg/mage/core"
)

// FORBIDDEN imports in card files:
import "github.com/benprew/mage/pkg/mage" // NEVER import engine internals in cards
```

1. **Card Authors (`cards/*`)**: Depend solely on `pkg/mage/dsl` and `pkg/mage/core`. Cards never reference `*mage.Game`, `*mage.Permanent`, or internal engine types directly.
2. **DSL Implementation (`pkg/mage/dsl`)**: Defines the public grammar, constructors, and sealed interfaces. Converts declarative structures into internal engine representations.
3. **Engine Core (`pkg/mage`)**: Implements game state, turn processing, and rules execution. Has no knowledge of card-specific logic or package syntax.
4. **Card Tests**: Card scenario tests import `pkg/mage/gametest` and test against behavioral outcomes using `gametest.TestGame`.

---

## 11. High-Leverage 3-Tier Testing Model

Testing balances exhaustive rules verification against rapid test execution:

```
           ▲
          / \     Tier 3: Risk-Based Card Tests (Targeted & High-Leverage)
         /   \    - Complex, multi-clause, or unusual Oracle rules (Balance, Fasting, Chains)
        /     \   - Regression reproductions & edge-case interactions
       /───────\  - Skip boilerplate for vanilla / french-vanilla / standard spells
      /         \
     /   Tier 2  \ Tier 2: Universal Static Card Validation (Instant & 100% Coverage)
    /             \ - TestRegisteredDefinitions: AST inspection across 100% of cards
   /               \ - testdata/compilefail: Compiler rejects category errors
  /─────────────────\
 /      Tier 1       \ Tier 1: Exhaustive Engine Contract Tests (Combinatorial & Heavy)
/                     \ - Primitives (Sacrifice, Destroy, Draw, Mill, Life, Layers 1-7)
/───────────────────────\ - Verified once in pkg/mage/ with full incarnation & clone tests
```

* `INVARIANT`: Verified engine primitives (`Sacrifice`, `Destroy`, `Exile`, `Draw`, etc.) are exhaustively tested in `pkg/mage`. Card authors do not duplicate primitive contract tests for standard cards.

---

## 12. Normative Rules References

| Subject | Relevant Comprehensive Rules | Architectural Invariant |
| :--- | :--- | :--- |
| **Simultaneous Choices** | CR 101.4 | APNAP order for choices; simultaneous action execution. |
| **Zone-Change Identity** | CR 400.7 | Zone departures create brand-new object incarnations. |
| **Casting Spells** | CR 601.2 | Transactional mode/target/cost lock and payment. |
| **Activated Abilities** | CR 602.2 | Activation steps mirror spell casting. |
| **Triggered Abilities** | CR 603 | Detection at event time; target selection on stack placement. |
| **Intervening-If** | CR 603.4 | Checked at trigger and rechecked at resolution. |
| **State-Triggered Abilities** | CR 603.8 | Triggers when state is true; does not re-trigger while on stack. |
| **Mana Abilities** | CR 605 | Resolves immediately without the stack. |
| **Resolving Spells/Abilities** | CR 608.2 | Target legality rechecked; partial resolution where able. |
| **Continuous Effects & Durations** | CR 611.2c / 611.3a | Temporary effects snapshot; static effects recalculate. |
| **Interaction of Continuous Effects** | CR 613 | Layer 1–7 evaluation; dependencies. |
| **Replacement Effects** | CR 614 / 616 | Intercept proposed actions; player ordering. |
| **Prevention Effects** | CR 615 | Capacity decrement; damage prevention. |
| **Keyword Actions** | CR 701 | Exact semantics for Sacrifice, Destroy, Exile, Search, Mill. |
