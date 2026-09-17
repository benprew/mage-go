# Regular, Composable Card DSL

Status: In progress. See the [implementation inventory](composable-dsl-implementation.md)
for supported adapters, verification, and remaining acceptance work.

## Summary

The card DSL should use a small grammar of typed, composable functions. Each
function should represent one concept:

```go
WhenState(
    Controls(HasSubType("Dwarf")),
    Sacrifice(This),
)
```

In this example:

- `Controls` is a condition.
- `HasSubType` is a permanent filter.
- `This` is a subject.
- `Sacrifice` is a resolving effect.
- `WhenState` constructs a state-triggered ability.

The DSL should avoid constructors that combine several concepts in their
names, such as `SacrificeIfControls`, `DrawCardsActivePlayer`, and
`BoostOtherControlledCreatures`.

The migration should proceed in vertical slices. Each slice introduces one
complete semantic family, migrates every current caller, and then removes the
old synonyms for that family.

Ability constructors state their lifecycle, such as event-triggered or
state-triggered. Effect verbs state the action. Their types distinguish
resolving, continuous, and replacement behavior.

## Scope: DSL Refactoring, Not Engine Replacement

The DSL is a typed authoring surface over existing engine services. The
[engine audit](composable-dsl-engine-audit.md) identifies services and tests to
reuse, confirmed gaps, and behavior that still needs investigation. A missing
DSL definition does not establish missing engine functionality.

Semantic contracts below constrain adapters. They do not authorize parallel
implementations of casting, payments, replacement processing, layers, trigger
storage, zone management, or cloning. Propose engine work separately with an
existing code path, a failing regression, and the smallest required change.
Broader identity changes and general event/action infrastructure are deferred
architecture proposals, not prerequisites for the DSL.

## Goals

- Make card definitions read consistently across abilities and effects.
- Make subjects, predicates, timing, costs, duration, and actions independently
  composable.
- Reject invalid combinations through Go types where practical.
- Give each operation one canonical card-facing name.
- Preserve exact Oracle behavior, including timing and state-trigger rules.
- Keep card-specific policy out of the game engine.
- Preserve diagnostic text and structural AI metadata through composition.
- Keep exact Oracle text as authoritative card data.

## Non-goals

- Do not build a serializable universal expression tree in this refactor.
- Do not force resolving, continuous, and replacement effects through one
  executable interface.
- Do not convert costs, targets, events, or state-based actions into effects.
- Do not migrate the entire DSL in one flag-day change.

## Design Constraints

The design favors compile-time rejection over compact implementation types. A
card definition must not compile when it passes a player to `Destroy`, uses a
resolving effect as a cost, installs a resolving effect as a static rule, or
uses an event predicate as a repeatable state condition.

Go types cannot reject every invalid card definition. Errors that depend on
values, such as an empty mode list or two bindings with the same debug name,
must fail during card registration. They must not remain latent until a game
resolves the ability. Registration errors must include the card name and the
path to the invalid expression.

The card-facing syntax should remain declarative and short. Engine lifecycle
details belong in typed values returned by constructors, not in card files.
The DSL may use more internal types than it exposes as constructor names.

The following priorities resolve design conflicts, in order:

1. Preserve exact rules behavior.
2. Reject invalid combinations at compile time where Go permits it.
3. Keep common card definitions short and readable.
4. Provide explicit, typed escape hatches for uncommon behavior.

## Reference Semantics and Existing Identity

Typed references must distinguish physical cards from rules objects. The engine
already assigns `Permanent.incarnationID` in `NewPermanent`; `Permanent.ID()`
currently returns the physical card ID. Reuse the existing incarnation in the
private reference resolver. A public ID rename, repository-wide lookup rewrite,
or new object registry is not a prerequisite of this DSL refactor.

`PermanentRef` must resolve only the captured battlefield incarnation. Phasing
preserves that incarnation. A missing permanent is absent for battlefield
actions; source characteristics can still come from the existing LKI service.
`CardRef` and `StackObjectRef` must likewise respect the applicable zone and
object lifetime. Audit each existing zone selector before wrapping it. If it
cannot distinguish a leave-and-return case, record a separate engine gap and
hold back only the affected adapter.

Zone-movement bindings expose the actual destination and object produced by the
existing move operation, following CR 400.7 where permitted. They must not
follow a physical card through unrelated later moves. Add a return value or
incarnation check to an existing path when necessary; do not create a parallel
zone system merely to provide a typed binding.

`This` resolves the source in its enclosing context: a battlefield reference
for permanent actions and a damage source with LKI for damage. Registration
rejects a battlefield action in a spell-only source context. Use existing spell
and ability copy paths to preserve source semantics.

During resolution, `You` is the stored spell or ability controller. For static
abilities and trigger detection, it uses the applicable current or pre-event
controller. It is not captured permanently when the card is created.

## Current Problems

The current card-facing API mixes several construction styles:

- Subjects are encoded in effect names: `SacrificeSource`, `UntapTarget`, and
  `DrawCardsActivePlayer`.
- Execution context is encoded in names: `SacrificeSourceStep` and
  `DestroyTargetStep` duplicate effects that already work in pipelines.
- Conditions are encoded in bespoke abilities such as `SacrificeIfControls`
  and `SacrificeUnlessLand`.
- Some effects accept selectors as positional arguments, while others use a
  mutable `.Targeting(...)` builder.
- Trigger phrases are represented by many unrelated constructors.
- Some builders accept `...any` and report invalid combinations with runtime
  panics instead of compile errors.
- `TriggerConditionData` currently includes event predicates, pure game-state
  predicates, and resolution-time random choices. These values are not safe in
  all the same contexts.

The state-trigger implementations also expose rules errors. Goblins of the
Flarg and the `SacrificeUnlessLand` cards are processed as state-based actions,
so their abilities do not use the stack. Dandân, Island Fish Jasconius, and
Merchant Ship approximate the same behavior with a battlefield zone-change
trigger, which misses relevant control and characteristic changes.

The current card review found twelve implemented cards with state-triggered
sacrifice abilities, plus Seasinger, which is not implemented yet.

## Type Model

### Effects

Effect categories should be siblings classified by their execution lifecycle:

- `ResolvingEffect` mutates a resolution context once.
- `ContinuousEffect` participates in the continuous-effect and layer system.
- `ReplacementEffect` intercepts a proposed game action.

A small metadata interface may expose diagnostic text and AI properties across
categories. It must not provide a universal execution method. APIs must accept
the specific effect category that they can execute.

Keep `Effect` as a temporary alias for `ResolvingEffect` while callers migrate.
Remove the alias when the migration no longer needs it.

### Abilities

Keep `Ability` as the shared identity and ownership interface because cards
store different ability types in one collection. Execute abilities through
their narrower capability interfaces:

- `TriggeredAbility`
- `ActivatedAbility`
- static ability holders
- mana abilities

The game must not add a new concrete ability type switch for each card
behavior. Bespoke card combinations should be built from generic triggers,
conditions, costs, and effects.

### Typed expression roles

Use distinct interfaces for values that have different valid contexts:

- permanent, player, card, stack-object, and damage-recipient selectors;
- permanent and card filters;
- pure state conditions;
- event patterns and event conditions;
- activation conditions;
- resolution-time choices and branches;
- resolving, continuous, and replacement effects;
- costs and timing restrictions.

Random choices such as a coin flip must not implement `StateCondition`. The
engine can evaluate a state condition repeatedly, so it must be pure.

Use sealed marker methods to keep these roles distinct. Marker methods should
be unexported in the package that owns the interfaces so external card packages
cannot accidentally implement the wrong role. Public escape-hatch constructors
wrap functions in the correct internal implementation.

The public roles include cardinality. At minimum, distinguish:

```go
type PermanentRef interface { permanentRef() }
type PermanentSet interface { permanentSet() }
type PlayerRef interface { playerRef() }
type PlayerSet interface { playerSet() }
type DamageSourceRef interface { damageSourceRef() }
type CardRef interface { cardRef() }
type CardSet interface { cardSet() }
type StackObjectRef interface { stackObjectRef() }
```

An optional lookup is not represented by returning a zero UUID. The resolver
returns an explicit absent result and the consuming operation defines what
absence means. Common resolving effects normally do nothing when their subject
no longer exists. Required values used by the card definition validator produce
a registration error if no producer exists.

### Typed bindings

Target declarations and resolution-time snapshots create typed bindings. An
effect refers to a binding, not to a position in a flat target slice:

```go
creature := TargetPermanent("creature", IsCreature)
player := TargetPlayer("player", AnyPlayer)

Activate(
    Costs(Mana("{1}{R}")),
    Sequence(
        DealDamage(This, creature, Fixed(2)),
        GainLife(player, Fixed(2)),
    ),
    WithTargets(creature, player),
)
```

The string is a diagnostic label only. The Go value is the identity of the
binding. Duplicate labels may reduce diagnostic quality, so registration
validation rejects them within one ability.

Bindings preserve target groups, minimum and maximum counts, target legality,
mode ownership, and divided values. When some targets become illegal, effects
still receive the surviving members of each binding. The engine counters a
spell or ability for having no legal targets only when the rules require it.

Pipeline storage uses the same typed approach:

```go
host := BindPermanent("host")

Sequence(
    Snapshot(AttachedPermanent, host),
    Sacrifice(This),
    AddCounters(From(host), P1P1, Fixed(1)),
)
```

Do not expose `map[string]any` or string-based lookup as the canonical DSL.
Legacy string variables may remain inside the engine during migration, but new
card-facing constructors must use typed bindings.

Binding declarations are immutable. Values live in a separate frame for each
activation, spell, or trigger resolution. Reusing a definition must not share
chosen targets or saved values between instances. Game clones copy frames;
spell and ability copies preserve choices as required by the copy rules and
use independent mutable storage. Retargeting changes only the copied frame.

Registration performs definite-assignment analysis, not just a search for a
producer. A required producer must execute before its consumer on every path
that reaches that consumer. A value produced only in one branch, an optional
operation, or a loop that can execute zero times is optional outside that
scope. Consumers must handle absence explicitly. Loop-local bindings cannot
escape without a typed collection or reduction. Branches can merge values
only through an explicit typed result. Reject reads before writes and cyclic
definition dependencies.

### Evaluation lifecycle

Each use of an expression has a defined evaluation time. A pure condition can
be reused in different lifecycles; its enclosing constructor selects the time:

| Role | Evaluation time | Data source |
| --- | --- | --- |
| Event pattern | When the event is dispatched | Event payload and event LKI |
| State-trigger condition | After each rules-visible state change, including during resolution | Current derived game state |
| Static condition | During continuous-effect evaluation | State at the applicable layer |
| Intervening-if condition | At trigger creation and before resolution | Current derived game state |
| Target declaration | When targets are chosen | Current legal game state |
| Target binding | During resolution | Stored choices with legality rechecked |
| Resolution selector | When its consuming effect resolves | Resolution context |
| Value source | When its consuming operation requests it | Resolution context unless its type says snapshot |
| Continuous modification | Capture at installation where required; apply in its layer | Layer state and explicitly captured values |
| Activation condition | Before activation and before costs are paid | Current game state |

Collection selectors snapshot their member references once when the consuming
operation starts, unless a constructor explicitly documents repeated
evaluation. This prevents mutation during iteration from changing which
objects the same operation affects.

Outside layer evaluation, permanent filters inspect effective characteristics
after continuous effects are current. Inside layer evaluation, filters inspect
the state available at that layer and must not recursively request a complete
continuous-effect recalculation. Card filters inspect the characteristics
defined for their zone.
The two filter interfaces remain distinct even when constructors such as
`HasSubType` can return a value that implements both safely.

### Public signature shape

Prefer required positional arguments followed by a closed option interface.
Do not use heterogeneous `...any` arguments or a variadic list that can contain
both effects and options. Multiple resolving effects compose with `Sequence`.

Representative signatures are:

```go
func Sacrifice(PermanentRef) ResolvingEffect
func Destroy(PermanentRef) ResolvingEffect
func DestroyEach(PermanentSet) ResolvingEffect
func Draw(PlayerRef, ValueSource) ResolvingEffect
func DrawEach(PlayerSet, ValueSource) ResolvingEffect

func WhenEvent(EventPattern, ResolvingEffect, ...TriggerOption) TriggeredAbility
func WhenState(StateCondition, ResolvingEffect, ...TriggerOption) TriggeredAbility
func At(TurnStepPattern, ResolvingEffect, ...TriggerOption) TriggeredAbility

func Activate(CostGroup, ResolvingEffect, ...ActivationOption) ActivatedAbility
func Spell(ResolvingEffect, ...SpellOption) SpellAbility
func WithTargets(...TargetBinding) ActivationOption
func WithInterveningIf(StateCondition) TriggerOption

func ModifyPT(PermanentRef, ValueSource, ValueSource) ContinuousModification
func Until(Duration, ...ContinuousModification) ResolvingEffect
func While(StateCondition, ...ContinuousModification) ContinuousEffect
func Static(...StaticRule) Ability
```

Go does not support a variadic slice of differently instantiated generic
bindings. `TargetBinding` is therefore a sealed non-generic declaration role;
each concrete binding also implements its domain-specific reference interface.
For example, a `PermanentTargetBinding` implements both `TargetBinding` and
`PermanentRef`, but it does not implement `PlayerRef`.

Use `WithTargets` for spells and triggered abilities as well as activated
abilities, with the applicable closed option type. If Go cannot express one
shared option without weakening the API, provide `WithSpellTargets`,
`WithTriggerTargets`, and `WithActivationTargets`. Prefer a slightly longer
name to losing compile-time category checks.

## DSL Grammar

### Subjects and selectors

Use explicit subjects instead of encoding subjects in verb names:

```go
This
EventSource
AttachedPermanent
You
ActivePlayer
EachPlayer
From(saved)
MatchingPermanents(IsCreature)
ControlledPermanents(HasSubType("Goblin"))
```

Keep different selector interfaces where their domains differ. For example, a
player selector must not be accepted by `Destroy`, and a permanent selector
must not be accepted by `Draw`.

Do not use a global `ChosenTarget` or `ChosenPlayer`. Target declarations return
typed bindings as described above. This keeps effects correct when an ability
has multiple target groups.

Singular and collection selectors are different types. `Destroy` accepts a
`PermanentRef`; `DestroyEach` accepts a `PermanentSet`. The card-facing names
make iteration explicit without a generic `Each` wrapper:

```go
Destroy(creature)
DestroyEach(MatchingPermanents(IsCreature))
Draw(You, Fixed(2))
DrawEach(EachPlayer, Fixed(2))
```

This avoids a universal selector interface and prevents an operation from
silently using the first member of a collection.

### Filters and conditions

Filters describe object characteristics. Conditions describe game state:

```go
And(IsCreature, HasSubType("Dwarf"))
Or(HasColorFilter(Red), HasColorFilter(Green))
Not(IsToken)

Controls(HasSubType("Dwarf"))
ControlsNone(HasSubType("Island"))
BattlefieldHasNone(IsCreature)
AllConditions(a, b)
AnyCondition(a, b)
NotCondition(a)
```

Add `StateConditionFunc` as an explicit escape hatch for uncommon, pure state
predicates. Jihad can use it because its condition depends on its chosen
player, chosen color, token status, and current permanent colors.

`StateConditionFunc` receives only `StateReader` and typed context values. Its
contract forbids mutation, randomness, player choices, wall-clock reads, and
mutable captured state. The wrapper must be safe to share across cloned games.
Prefer a named data implementation when a predicate carries state that must be
cloned or compared.

Implement `StateReader` as a restricted query surface with read-only object
views. Do not alias the current `GameReader`: it exposes `FlipCoin`, and its
returned engine objects can expose mutation. The reader must not provide a
route to mutable game, player, card, or permanent values. Arbitrary callbacks
remain trusted code: registration can check declared dependencies and context,
but cannot prove purity or discover mutable closure captures. Require an
explicit dependency declaration and review for each callback. Tests for
repeatability and clone isolation supplement that review.

### Resolving effects

Use a canonical verb whose first argument identifies the affected subject:

```go
Sacrifice(This)
Destroy(target)
DestroyWithoutRegeneration(target)
Exile(target)
Tap(target)
Untap(This)
Draw(You, Fixed(2))
Discard(player, Fixed(2))
GainLife(You, Fixed(3))
LoseLife(player, Fixed(2))
DealDamage(This, target, Fixed(3))
AddCounters(target, P1P1, Fixed(2))
```

The same effect must work at the top level, inside a sequence, inside a mode,
or inside a loop. Do not create `Step` variants.

Use composition for resolution control flow:

```go
host := BindPermanent("host")

Sequence(
    Snapshot(AttachedPermanent, host),
    Sacrifice(This),
    AddCounters(From(host), P1P1, Fixed(1)),
)
```

Keep resolution-time branches separate from state conditions when the branch
has side effects. A coin flip should be an explicit resolution operation, not
a reusable state predicate.

### Action results and execution boundaries

Each primitive documents its actor, subject domain, evaluation time, absence
behavior, replacement handling, simultaneous behavior, emitted events, and
typed result. An impossible rules action returns a normal result and the
sequence continues as far as possible. An engine error is a separate failure;
it must not mean that a player declined or could not perform an instruction.

Results distinguish a choice being accepted, a cost being paid, an action
being performed, objects moved, and actual amounts such as damage dealt or
cards drawn. No single success boolean represents all of these. A replaced
action records the actual outcome without losing the payment or choice facts
needed by the rules. A branch for “if you do” consumes the applicable payment
or action result. “When you do” creates a reflexive trigger and is not an
inline branch. “That many” reads an explicit amount binding. Each constructor
must state which result it produces and how replacement affects that result.

`Sequence` executes children in order. A loop executes one body per member in
its documented order. Neither implies simultaneous execution.
`DestroyEach` collects its subjects and performs one simultaneous destruction
action group. All members share the appropriate pre-action observation state;
trigger detection must not observe intermediate removals as separate batches.
For “each player sacrifices a creature,” collect choices in APNAP order, then
perform the sacrifices simultaneously. These are preservation requirements for adapters, not proof that the engine
lacks batch behavior. Audit the existing action and event paths for the caller
first. A failing simultaneous-action case needs a separate engine correction;
this document does not prescribe a new batch executor or event representation.

Batch behavior is specific to the action. For example, multiple card draws
occur as individual draws under the draw rules. Do not infer timing from the
`Each` suffix alone. State-trigger checks occur after rules-visible actions,
not between implementation mutations inside one simultaneous action. No
combinator grants priority or inserts state-based actions during ordinary
resolution. Cancellation of a choice and rollback of cost payment follow
their own contracts; they do not roll back an already resolving sequence.

### Triggered abilities

Use separate constructors for event, state, and turn-step triggers. This is
slightly more explicit than an overloaded `When`, but it lets Go reject an
event pattern where a pure state condition is required:

```go
WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This))
WhenEvent(Enters(This), Draw(You, Fixed(1)))
WhenEvent(Dies(This), GainLife(You, Fixed(3)))
At(YourUpkeep, LoseLife(You, Fixed(1)))
```

`WhenEvent` accepts only `EventPattern`. `WhenState` accepts only
`StateCondition`. `At` accepts only `TurnStepPattern`. They may create the same
internal triggered-ability representation, but their public signatures stay
distinct.

Intervening-if clauses need a distinct typed wrapper because the engine checks
them when the ability triggers and again when it resolves. A resolution-only
`If` is not equivalent.

```go
WhenEvent(
    Enters(This),
    Draw(You, Fixed(1)),
    WithInterveningIf(Controls(IsArtifact)),
)
```

At detection, a pending trigger captures its controller, source, event
bindings, and required LKI. Modes and targets are selected when it is put on
the stack and then stored on that instance. Resolution does not read these
captured values from the source again.

Event patterns expose typed participants: actor, source, affected object,
recipient, and zone-change results as applicable. Use typed adapters over the current event payload. Replacing the internal
event representation is not required by this refactor. Record before/after characteristics and group identity where required.
Patterns declare whether they use current state or look-back state. A death
trigger must still be detected when its source leaves in the same event group.
“One or more” patterns consume a rules event group, not a heuristic burst of
unrelated calls.

Define delayed and reflexive trigger constructors with captured bindings,
controller, source, active zones, duration, and one-shot or repeat behavior.
Define linked-ability storage separately from ordinary resolution bindings.
Trigger limits belong to a specified ability instance and reset boundary.
Granted abilities need stable identities across derived-state recalculation;
two distinct granted instances must not share a state-trigger suppression key.

### Activated and static abilities

Compose activated abilities from typed costs, effects, targets, timing, and
limits:

```go
target := TargetPermanent("creature", IsCreature)

Activate(
    Costs(Mana("{U}"), TapCost(This)),
    Until(EndOfTurn, GrantKeyword(target, Flying)),
    WithTargets(target),
)
```

Replace permissive `...any` inputs with closed option interfaces. Required
costs and effects remain positional arguments. Options contain only the
applicable target, timing, limit, or permission declarations.

Represent reusable continuous modifications separately from installed
continuous effects and the wrapper that gives them a lifetime:

```go
Until(EndOfTurn,
    ModifyPT(target, Fixed(2), Fixed(0)),
    GrantKeyword(target, Trample),
)

Static(
    While(Controls(HasSubType("Goblin")),
        ModifyPT(This, Fixed(1), Fixed(1)),
    ),
)
```

`ModifyPT` and `GrantKeyword` return `ContinuousModification`. `Until` accepts
only continuous modifications and creates a resolving effect that installs
continuous-effect instances. `While` accepts only a `StateCondition` and
continuous modifications and creates a `ContinuousEffect`. `Static` accepts
only `StaticRule`; `ContinuousEffect` and `ReplacementEffect` implement that
sealed role. `ResolvingEffect` does not.

Each continuous modification declares its layer and dependency inputs. Its
captured permanent references retain the existing incarnation identity. The engine applies it
in the normal layer system and must not depend on constructor order except
where the comprehensive rules use timestamps or dependencies.

`Sacrifice(permanent)` instructs `You`, the stored resolving controller, to
sacrifice that permanent. It succeeds only if that player currently controls
the permanent and can sacrifice it. It does not transfer the instruction to a
new controller. If A controls a sacrifice trigger and B gains control of its
source in response, A cannot sacrifice that source. Effects such as
“target player sacrifices a creature” use a separate player-choice composition
that records the instructed player and lets that player choose a legal object.
Damage effects retain a `DamageSourceRef` and the required LKI so prevention,
lifelink, deathtouch, and source characteristics use the correct object.

### Continuous-effect capture contract

Continuous modifications belong to an installed effect instance with one
source, controller context, timestamp, and lifetime. A compound effect retains
this identity across its layers. Splitting it into layer entries must not
create unrelated timestamps or expiration behavior.

For resolving effects that modify characteristics or control, capture affected
objects when the effect begins (CR 611.2c). A temporary team boost does not
include a creature that enters later. A static team boost recalculates its
membership and includes that creature (CR 611.3a). Rules-modifying effects can
have different membership behavior; classify each part of a compound effect
independently. Do not implement every continuous effect as a frozen set.

Numeric expressions declare whether the rules require a value determined at
resolution, a saved value, or a value recalculated during application. Reuse of
`ValueSource` must not silently change that policy. Durations specify their
start and end conditions, including “for as long as” conditions that must hold
when the effect begins. Leaving, phasing, and losing an ability are distinct
events for lifetime purposes.

Dependency metadata describes reads and writes, but overlapping fields alone
do not establish a rules dependency. Full dependency handling under CR 613 requires separate engine verification.
An uncovered case belongs in the existing effect manager with a regression,
not in a DSL-specific layer executor. Use a layer-scoped reader for applicability
and values. Reuse existing layer and duration tests. Add uncovered capture, dependency,
or expiration cases only when relevant to a migrating caller.

### Rules text and AI metadata

Exact Oracle text remains card data above `Register`. The DSL does not attempt
to regenerate exact Oracle prose from composed nodes. Each node may provide a
short diagnostic description for logs, prompts, and registration errors, but
that description is not authoritative rules text.

AI metadata is structural. It remains attached to the affected subject and
operation instead of being flattened immediately into one `EffectProperties`
value. Composition follows these rules:

- `Sequence` preserves ordered child metadata.
- A branch preserves metadata for each alternative and its condition.
- A mode preserves separate metadata and targets for each choice.
- A loop retains its member selector and per-member operation.
- A wrapper such as `Until` retains the installed modifications and duration.
- An explicit AI hint annotates a node and does not replace its rules behavior.

Legacy consumers may derive an `EffectProperties` summary from this structure
during migration. The summary is allowed to lose strategic detail, but it must
not control rules execution. Tests compare both the structural metadata and
the legacy summary for migrated effects.

## Adapter Contracts for Existing Engine Services

### Spells, costs, and mana abilities

`Spell` defines the resolving ability of an instant or sorcery. The card
constructor supplies printed characteristics and mana cost. For example:

```go
target := TargetPermanent("creature", IsCreature)

Spell(
    DealDamage(This, target, Fixed(3)),
    WithSpellTargets(target),
)
```

`WithSpellTargets`, `WithTriggerTargets`, and `WithActivationTargets` are the
fallback names if a shared closed `WithTargets` option is not practical. They
are not three ways to express the same option in one context.

Define modes as effect bodies with their own target declarations. The casting
contract selects modes, X, alternative and additional costs, targets, and
divided values in the order required by CR 601. It then determines and locks
the total cost, activates mana abilities where permitted, and pays the cost.
Activated abilities use the corresponding CR 602 procedure. Optional and
repeated costs record what was paid in the instance context. Resolution reads
that record rather than a mutable game-global X or last-paid-object slot.

Integrate typed costs with the existing total-cost transaction system. Cost
results can bind sacrificed, discarded, or exiled objects and their LKI.
Affordability checks and canceled transactions must not publish bindings,
consume choices or randomness, or mutate the live game. Variable costs,
alternative payments, reductions, and spending restrictions share the same
locked payment model. An explicitly empty cost group is valid for a zero-cost
activation; validation must distinguish it from a missing required cost.

Casting permissions declare allowed zones, timing, duration, and whether they
permit or instruct casting during resolution. Spell copies preserve modes, X,
cost-related facts, target groups, and divided values as required by the rules.
Retargeting validates each group without flattening it. Copying a spell is not
casting it and must not emit a cast event.

Mana abilities have an explicit capability and eligibility validation. They
resolve without the stack when the rules require it, including their
post-production effects. The mana planner and manual activation must use the
same typed costs, choices, restrictions, and production metadata. Add
triggered mana abilities to this contract rather than treating every trigger
as a normal stacked ability.

### Choices and hidden zones

Non-target choice declarations specify the choosing player, candidates,
cardinality, optionality, visibility, and evaluation time. Their typed results
feed effects without becoming target bindings. Search, reveal, look, order,
discard, and random selection remain distinct operations. A search declares
the zone and restrictions so the engine can apply permitted failure to find;
it must not reuse mandatory public-object selection behavior.

Reveal and look operations define which players receive information and how
long that permission lasts. Prompts, logs, and AI views must use those
permissions. Ordering records a chosen permutation; random choices use the
game RNG and clone-safe state. Empty candidate sets and declined optional
choices return explicit results. These contracts need adapters for human
players, tests, and AI before replacing existing choice helpers.

### Replacement and prevention rules

A replacement pattern binds a typed proposed action, not an event that has
already occurred. Its transformation returns a revised action or a prevention
result. It cannot call the ordinary resolving executor as a substitute for
replacement processing. Reuse `EffectManager.ApplyReplacements` for applicability rechecks and
transformations. Full CR 616 player ordering is a separate engine audit item;
do not infer that it already exists or build another pipeline in the DSL.

Track which replacement instance has applied to the event so it cannot apply
repeatedly where the rules prohibit that. Prevention records the amount
prevented, remaining shield capacity, source and recipient scope, and expiry.
Registration distinguishes static rules from resolving operations that install
temporary shields. Zone-entry replacements, optional replacements, damage
prevention, and unpreventable damage need separate acceptance cases. Emit
events for actual outcomes; do not dispatch the unreplaced proposal as an
event that occurred. Preserve action/payment results needed by later clauses.

### Persistent state and set coverage

Store chosen names, colors, and types in typed state owned by the appropriate
card or ability instance. Linked exiled objects use zone-object references and
link identities; they do not use a process-global list or resolution frame.
Specify which state survives control changes, copying, phasing, and zone
changes. Granted abilities and copied characteristics must not share mutable
state between game clones or card instances.

Maintain a capability inventory for sets selected for migration. Include
counters, tokens, attachments, copied characteristics, granted abilities,
face-down cards, multiple faces, alternate casting mechanics, and abilities
that function outside the battlefield. For each family, record its typed
representation, engine support, representative cards, tests, and owning slice.
A family without an exact contract remains explicitly unsupported or uses a
reviewed typed escape hatch. This design does not imply that all such mechanics
are already supported. Before adding a new set, update the inventory and
schedule any engine gaps; do not hide those gaps in card-specific callbacks.

## Representative Card Migrations

### Goblins of the Flarg

```go
// Before
WithAbility(SacrificeIfControls("Dwarf"))
```

```go
// After
WithAbility(WhenState(
    Controls(HasSubType("Dwarf")),
    Sacrifice(This),
))
```

The source is included. If Goblins of the Flarg gains the Dwarf subtype, its
ability triggers.

### Island-dependent creatures

Vodalian Knights, Giant Shark, Pirate Ship, and Sea Serpent currently use:

```go
// Before
WithAbility(SacrificeUnlessLand("Island"))
```

They become:

```go
// After
WithAbility(WhenState(
    ControlsNone(HasSubType("Island")),
    Sacrifice(This),
))
```

Dandân, Island Fish Jasconius, and Merchant Ship currently approximate the
same rule with `EvtZoneChange`. They migrate to the same new form. This makes
control changes, land-type changes, and other non-zone state changes work.

### Drop of Honey

```go
// Before
WithAbility(NewStateTriggered(false, SacrificeSource()).
    SetConditionData(NoBattlefieldPermanentMatching{Filter: IsCreature}))
```

```go
// After
WithAbility(WhenState(
    BattlefieldHasNone(IsCreature),
    Sacrifice(This),
))
```

### Mana Vortex

The current implementation uses `NewStateTriggered` plus a card-specific
`FuncEffect` that finds and sacrifices its source. It becomes:

```go
WithAbility(WhenState(
    BattlefieldHasNone(IsLand),
    Sacrifice(This),
))
```

### Pipeline sacrifice

```go
// Before
Sequence(
    SnapshotAttached("host"),
    SacrificeSourceStep(),
    AddCounters(P1P1, Fixed(1)).Targeting(ToGathered("host")),
)
```

```go
// After
host := BindPermanent("host")

Sequence(
    Snapshot(AttachedPermanent, host),
    Sacrifice(This),
    AddCounters(From(host), P1P1, Fixed(1)),
)
```

### Drawing cards

```go
// Before
DrawCardsActivePlayer(Fixed(1))
DrawCards(Fixed(1)).Targeting(SelectController())
GainLifeTarget(Fixed(3))
```

```go
// After
Draw(ActivePlayer, Fixed(1))
Draw(You, Fixed(1))
GainLife(player, Fixed(3))
```

Here `player` is a `PlayerTargetBinding` returned by the surrounding target
declaration. It is not a global target slot.

### Destruction

```go
// Before
DestroyTarget()
DestroyTargetStep()
DestroyTargetPermanent()
DestroyAllCreatures()
```

```go
// After
Destroy(target)
DestroyEach(MatchingPermanents(IsCreature))
```

Target legality belongs to the target declaration. The resolving effect does
not repeat the target's card-type restriction in its name.

## Refactor Entry Point

Start with the DSL package, sealed roles, definition inspection, and adapter
tests for one family. Reuse engine IDs, events, pending triggers, effect
managers, payment transactions, and clone hooks. Introduce inspection and
validation incrementally; no global engine rewrite is an entry requirement.

Before migrating a family, record its existing implementations and tests using
the [engine audit](composable-dsl-engine-audit.md). Separate missing DSL syntax,
missing test coverage, and confirmed engine defects. Correct a defect in a
separate engine change when exact migration depends on it. Unaffected families
can proceed independently.

## First Vertical Slice: State Triggers and Sacrifice

This slice changes the card-facing grammar. The separate state-trigger and
sacrifice correctness fixes below are dependencies for affected callers, not
new trigger or action subsystems within the refactor.

### Add

- `StateCondition`
- `StateConditionFunc`
- `WhenState`
- `Controls`
- `ControlsNone`
- `BattlefieldHasNone`
- `AllConditions`
- `AnyCondition`
- `NotCondition`
- the permanent-subject abstraction
- `This`
- `Sacrifice(subject)`
- adapters to corrected existing state-trigger and sacrifice paths

### Remove after migrating callers

- `SacrificeIfControls`
- `SacrificeIfControlsAbility`
- `SacrificeUnlessLand`
- `SacrificeUnlessLandAbility`
- public DSL access to `NewStateTriggered`
- `SacrificeSource`
- `SacrificeSourceStep`
- `ControllerHasNoPermanentMatching`
- `NoBattlefieldPermanentMatching`

Do not remove `SacrificeTarget`, `SacrificeTargetStep`, or
`SacrificeGathered` in this slice. Remove them in the typed-binding slice after
`TargetPermanent`, `BindPermanent`, and `From` cover every caller. This keeps
the first slice small and prevents a temporary untyped replacement.

Keep sacrifice costs, including `SacrificeSourceCost` and
`SacrificeMatchingCost`. Costs are a separate DSL role.

### Migrate cards

- Goblins of the Flarg
- Vodalian Knights
- Giant Shark
- Pirate Ship
- Sea Serpent
- Dandân
- Island Fish Jasconius
- Merchant Ship
- Serendib Djinn
- Drop of Honey
- Mana Vortex
- Jihad

Use the new API when Seasinger is implemented.

## Later Vertical Slices

1. Add typed target and saved-value adapters over existing resolution contexts.
   Define scope and validation; adapt existing copy and retargeting paths.
2. Consolidate permanent and player action constructors over existing actions.
   Preserve their replacement and event behavior. Add typed result adapters
   only where a current caller needs them.
3. Adapt existing choice, search, reveal, ordering, and random helpers to typed
   bindings. Reuse human, AI, and test choice interfaces.
4. Compose existing temporary and static continuous effects through typed
   modification and duration constructors. Keep the current effect manager.
5. Consolidate existing event, turn-step, and delayed-trigger constructors.
   Audit intervening-if and other less common forms before claiming support.
6. Consolidate spell and activated-ability builders, modes, targets, and costs.
   Reuse casting, copying, payment transactions, and mana planning.
7. Expose typed replacement/prevention constructors over the existing pipeline.
8. Finish production DSL imports, generators, and author documentation after
   all required families have adapters.

For every slice, map removed exports to their canonical replacements and
existing engine executors. Retain an unmigrated helper until all callers have
an exact representation. Do not expand a slice into an engine subsystem rewrite
because an optional future mechanic might need more support.

Existing rules tests are the regression baseline. Add DSL composition tests
and missing behavioral cases, rather than duplicating every engine test.

## Separate State-Trigger Engine Fixes

The [engine audit](composable-dsl-engine-audit.md#confirmed-transient-state-trigger-gap)
confirmed a transient-state observation gap and identified rearming behavior
that must be corrected in the existing trigger system. These are separate
engine changes required before migrating affected cards to `WhenState`.

The acceptance contract is: record a trigger when its state becomes true,
including between instructions during resolution; retain it if the condition
later becomes false; suppress repeated triggering while its original instance
is outstanding; and rearm after that instance leaves the stack. State checks
must not expose intermediate mutations inside a simultaneous rules action.
Pending triggers use the existing priority and APNAP placement paths.

Implement the smallest correction to existing observation, pending-instance,
and cleanup paths. No new shared action executor is prescribed. Add focused
regressions before fixing the engine, including the empty-hand discard/draw
case and rearming after resolution/countering. Use existing incarnation data
where source-lifetime checks are needed.

The sacrifice-controller correction is also separate: the instructed player
must still control the permanent. Removing the condition that originally
caused a non-intervening-if trigger does not cancel the instruction.

## Registration Validation

Compile-time types are the primary safety mechanism. Registration validation
handles constraints that Go cannot express. It runs once for every registered
card in tests and development builds, before a game starts.

Each definition node exposes an internal inspection descriptor: role, named
children, binding declarations/reads/writes, context requirements, and semantic
metadata. The visitor uses named edges and indices to build stable expression
paths. Execution remains on the lifecycle-specific interfaces. This is an
inspectable definition graph, not a serializable universal executor.

Card factories must produce a stable, immutable definition independent of
randomness and game state. Validate one definition per registered factory;
runtime instances allocate their own IDs and state. Add a registry validation
entry point that loads all intended set packages and reports all errors before
tests or development games start. Factory construction and validation run
outside the registry lock. Convert local constructor failures into diagnostics
with the card name at that boundary. Dynamic granted abilities must originate
from validated definitions; callback-generated definitions require validation
before installation.

Opaque escape hatches declare their lifecycle, reads, writes, bindings, and AI
metadata. The visitor validates those declarations, not arbitrary Go behavior.
Each escape hatch has focused behavioral tests and an entry in the capability
inventory. Do not claim that registration can prove callback correctness.

Validation includes:

- empty sequences, mode lists, or collections whose contract requires members;
- missing required costs, while permitting explicitly empty cost groups;
- missing or non-dominating producers for typed bindings, including reads
  before writes, optional branches, and zero-iteration loops;
- duplicate diagnostic binding names within one ability;
- a target binding used outside the ability or mode that declares it;
- invalid minimum and maximum target counts;
- incompatible divided-value and target cardinality declarations;
- mutually exclusive ability options;
- `nil` expressions hidden behind an interface;
- an installed continuous effect without a valid layer or lifetime policy;
- event bindings used by an unrelated event pattern; and
- escape-hatch functions used in a context with an incompatible lifecycle.

Validation errors include the card name, ability index, expression path, and a
short corrective message. Constructors may panic immediately for local value
errors, such as a target range whose minimum exceeds its maximum. Do not defer
a card-definition error until resolution.

## Package Boundary

The interfaces and execution implementations live in `pkg/mage`. The
`pkg/mage/dsl` package is the only supported card-author import surface. It
re-exports closed interfaces and constructors but does not expose mutable engine
representations.

Cards import `pkg/mage/dsl` and `pkg/mage/core` only. Escape hatches are
constructor functions in `dsl`; cards do not implement sealed interfaces
directly. This keeps engine execution methods out of card packages and avoids
an import cycle between the grammar and the engine.

During migration, existing cards may continue to import `pkg/mage` for
unmigrated families. A slice is complete when its migrated expressions compile
through `pkg/mage/dsl`, all callers of its removed exports have migrated, and
its capability entries pass validation. A card with an unrelated legacy
ability does not force that ability into the same slice. Track its remaining
direct imports explicitly; remove them when their families migrate.

The final production card surface includes registration, card constructors,
typed state, and all required grammar exports. Card tests may still import
`pkg/mage/gametest` and engine test helpers. Update `cmd/genset`,
`pkg/mage/doc.go`, and card-author examples before requiring DSL-only production
imports repository-wide. Re-exporting mutable engine types is not completion
of the package boundary.

## Testing and Acceptance Criteria

The checklist below names behaviors to preserve, not tests that must all be
written again. Link existing passing tests first, strengthen weak assertions,
and add only uncovered cases or adapter-specific tests. Unverified engine
capabilities remain separate audit items and do not block unrelated slices.

Each primitive has a semantic contract with its rules references, actor,
subjects, timing, capture policy, absence behavior, replacements, event groups,
results, and clone behavior. Build reusable behavioral tests from those
contracts. Type tests establish valid composition, not Oracle fidelity.

For each representative or migrated card, maintain a clause-to-test table:

| Oracle clause | DSL expression path | Rules contract | Behavioral test |
| --- | --- | --- | --- |
| Trigger condition | Ability trigger | Observation time and active zone | Relevant condition becomes true |
| Restriction or qualifier | Filter, condition, or option | Actor, scope, and legality | Near-match is excluded |
| Action and dependent clause | Effect/result binding | Actual outcome and timing | Replacement or partial outcome |
| Duration | Installed effect | Capture and expiry | Later entry and expiration |

Use exact Oracle data from the set inventory and independent expected rules
outcomes. Migration parity is evidence of preserved behavior, not proof of
correctness. Record known legacy rules errors and their intentional corrections
with new regression tests instead of preserving them as expected results.

- Add positive compile tests for each public grammar type and constructor.
- Add negative fixtures under `testdata/compilefail` and check them with
  `go/types`. Each fixture contains one invalid category combination and the
  expected compiler diagnostic. Ordinary Go tests cannot contain intentionally
  invalid expressions.
- Verify at compile time that invalid categories cannot be passed to unrelated
  constructors, including player/permanent, singular/collection,
  event/state-condition, cost/effect, and modification/installed-effect pairs.
- Run registration validation over every registered card and test focused
  diagnostics for each value-dependent error class.
- Test every combinator independently and in nested combinations.
- Verify that composed effects preserve diagnostic text, structural AI
  metadata, and the derived legacy `EffectProperties` summary.
- Verify that the same resolving effect works directly, in a sequence, in a
  loop, in a mode, and in a triggered or activated ability.
- Verify that state triggers use the stack and do not duplicate while an
  instance is outstanding.
- Verify a transient state during resolution queues a trigger even when the
  condition is false before priority; verify simultaneous actions do not expose
  intermediate implementation states.
- Verify sacrifice fails when the instructed player no longer controls the
  source, and distinguish that from removing the trigger condition in response.
- Verify re-triggering after resolution, countering, fizzling, and other stack
  removal when the state remains true.
- Verify that an outstanding trigger from an old permanent does not suppress a
  state trigger from a new permanent created when the card returns.
- Verify that `This`, a chosen target, an attachment, or a stored binding does
  not resolve to a later permanent created from the same card.
- Verify reference adapters distinguish battlefield incarnations while keeping
  the same reference valid across phasing. Do not require a public ID rename.
- Verify graveyard and exile round trips invalidate old references, permitted
  zone-change results find the new object, and replacement destinations are
  respected. Test spell and ability source references after copying and exit.
- Verify event last-known information for death, leave-zone, and control-change
  patterns.
- Verify multiple target groups, partial target illegality, modal target scope,
  and divided values without positional target access.
- Verify binding scope, reads before writes, branch merging, empty loops,
  repeated activations, cloned frames, spell copies, and retargeting.
- Verify action results distinguish choice, payment, actual action, and amount;
  impossible instructions continue a sequence and do not become engine errors.
- Verify simultaneous destruction, APNAP choices followed by simultaneous
  sacrifices, grouped triggers, and death triggers whose source also leaves.
- Verify static versus temporary membership, captured versus dynamic values,
  compound timestamps, dependency loops, and conditional duration boundaries.
- Verify hidden-zone visibility and legal search failure through player, AI,
  logging, and test adapters.
- Verify replacement ordering and rechecks, shield consumption, typed outcomes,
  and cost transaction rollback without live-state or binding leaks.
- Verify casting and activation choice order, modal copies, delayed/reflexive
  triggers, linked state, and immediate mana-ability execution.
- Verify source-inclusive subtype matching, control changes, continuous type
  changes, and cloned-game isolation.
- Verify deterministic state conditions by evaluating them repeatedly before
  and after cloning without changing game state.
- Add focused tests for all migrated cards before removing their old helpers.
- Run focused engine tests, affected card-package tests, all engine tests, the
  full repository suite, and lint after each vertical slice.

## Migration Rules

- Write or update behavior tests before changing each family.
- Preserve exact Oracle text above each card registration.
- Do not change unrelated card behavior while migrating syntax.
- Do not retain two canonical spellings for the same operation.
- Keep an escape hatch only when the generic vocabulary cannot express exact
  behavior. Give each escape hatch an explicit context-specific type.
- Update `pkg/mage/doc.go` and `pkg/mage/dsl` with each completed slice.
- Remove obsolete implementations, exports, tests, and engine type switches
  only after all callers use the replacement.
- Maintain a semantic parity table for each slice. For every removed export,
  list all in-tree callers, the typed replacement, target and timing behavior,
  LKI requirements, metadata behavior, and the focused regression test.
- Do not remove an old helper merely because the new syntax can approximate its
  common case. Every caller must have an exact typed representation first.

## Rules References

Use the [official Comprehensive Rules](https://magic.wizards.com/en/rules) and
the repository's `docs/comprehensive-rules.md` when implementing each contract.
Record the rules revision in the semantic test inventory. Relevant sections
include CR 101.4 (simultaneous choices), 400.7 (zone-change identity), 601–602
(casting and activation), 603.7–603.12 (trigger lifecycles and observation),
605 (mana abilities), 608 (resolution), 611–613 (continuous effects and layers),
614–616 (replacement and prevention), and the sacrifice and search keyword
actions in CR 701. Rule numbering for keyword actions can change between
revisions; match the named rule as well as its number.
