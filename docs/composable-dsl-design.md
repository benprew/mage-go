# Regular, Composable Card DSL

Status: Proposed

## Summary

The card DSL should use a small grammar of typed, composable functions. Each
function should represent one concept:

```go
When(
	Controls(HasSubType("Dwarf")),
	Sacrifice(This),
)
```

In this example:

- `Controls` is a condition.
- `HasSubType` is a permanent filter.
- `This` is a subject.
- `Sacrifice` is a resolving effect.
- `When` constructs a triggered ability.

The DSL should avoid constructors that combine several concepts in their
names, such as `SacrificeIfControls`, `DrawCardsActivePlayer`, and
`BoostOtherControlledCreatures`.

The migration should proceed in vertical slices. Each slice introduces one
complete semantic family, migrates every current caller, and then removes the
old synonyms for that family.

## Goals

- Make card definitions read consistently across abilities and effects.
- Make subjects, predicates, timing, costs, duration, and actions independently
  composable.
- Reject invalid combinations through Go types where practical.
- Give each operation one canonical card-facing name.
- Preserve exact Oracle behavior, including timing and state-trigger rules.
- Keep card-specific policy out of the game engine.
- Preserve rules text and AI metadata through composition.

## Non-goals

- Do not build a serializable universal expression tree in this refactor.
- Do not force resolving, continuous, and replacement effects through one
  executable interface.
- Do not convert costs, targets, events, or state-based actions into effects.
- Do not migrate the entire DSL in one flag-day change.

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

A small metadata interface may expose rules text and AI properties across
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

## DSL Grammar

### Subjects and selectors

Use explicit subjects instead of encoding subjects in verb names:

```go
This
ChosenTarget
EventSource
AttachedPermanent
You
ChosenPlayer
ActivePlayer
EachPlayer
ToGathered("saved")
Matching(IsCreature)
ControlledPermanents(HasSubType("Goblin"))
```

Keep different selector interfaces where their domains differ. For example, a
player selector must not be accepted by `Destroy`, and a permanent selector
must not be accepted by `Draw`.

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

### Resolving effects

Use a canonical verb whose first argument identifies the affected subject:

```go
Sacrifice(This)
Destroy(ChosenTarget)
DestroyWithoutRegeneration(ChosenTarget)
Exile(ChosenTarget)
Tap(ChosenTarget)
Untap(This)
Draw(You, Fixed(2))
Discard(ChosenPlayer, Fixed(2))
GainLife(You, Fixed(3))
LoseLife(ChosenPlayer, Fixed(2))
DealDamage(This, ChosenTarget, Fixed(3))
AddCounters(ChosenTarget, P1P1, Fixed(2))
```

The same effect must work at the top level, inside a sequence, inside a mode,
or inside a loop. Do not create `Step` variants.

Use composition for resolution control flow:

```go
Sequence(
	Snapshot(AttachedPermanent, "host"),
	Sacrifice(This),
	AddCounters(ToGathered("host"), P1P1, Fixed(1)),
)
```

Keep resolution-time branches separate from state conditions when the branch
has side effects. A coin flip should be an explicit resolution operation, not
a reusable state predicate.

### Triggered abilities

Use a small set of trigger constructors over reusable trigger patterns:

```go
When(Controls(HasSubType("Dwarf")), Sacrifice(This))
When(Enters(This), Draw(You, Fixed(1)))
When(Dies(This), GainLife(You, Fixed(3)))
At(YourUpkeep, LoseLife(You, Fixed(1)))
```

`When` accepts an event pattern or a state-trigger condition. `At` accepts a
typed turn-step pattern. Both create the same generic triggered-ability
representation.

Intervening-if clauses need a distinct typed wrapper because the engine checks
them when the ability triggers and again when it resolves. A resolution-only
`If` is not equivalent.

### Activated and static abilities

Compose activated abilities from typed costs, effects, targets, timing, and
limits:

```go
Activate(
	Costs(Mana("{U}"), TapCost(This)),
	Until(EndOfTurn, GrantKeyword(This, Flying)),
)
```

Replace permissive `...any` inputs with closed option interfaces. A valid
effect, target declaration, cost group, or timing restriction can implement
the applicable option interface; unrelated values cannot.

Represent reusable continuous modifications separately from the wrapper that
gives them a lifetime:

```go
Until(EndOfTurn,
	ModifyPT(ChosenTarget, Fixed(2), Fixed(0)),
	GrantKeyword(ChosenTarget, Trample),
)

Static(
	While(Controls(HasSubType("Goblin")),
		ModifyPT(This, Fixed(1), Fixed(1)),
	),
)
```

`Until` creates a resolving effect that installs continuous modifications.
`While` creates a conditional continuous effect. `Static` creates the static
ability that owns continuous or replacement rules.

## Representative Card Migrations

### Goblins of the Flarg

```go
// Before
WithAbility(SacrificeIfControls("Dwarf"))
```

```go
// After
WithAbility(When(
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
WithAbility(When(
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
WithAbility(When(
	BattlefieldHasNone(IsCreature),
	Sacrifice(This),
))
```

### Mana Vortex

The current implementation uses `NewStateTriggered` plus a card-specific
`FuncEffect` that finds and sacrifices its source. It becomes:

```go
WithAbility(When(
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
Sequence(
	Snapshot(AttachedPermanent, "host"),
	Sacrifice(This),
	AddCounters(ToGathered("host"), P1P1, Fixed(1)),
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
GainLife(ChosenPlayer, Fixed(3))
```

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
Destroy(ChosenTarget)
Destroy(Each(Matching(IsCreature)))
```

Target legality belongs to the target declaration. The resolving effect does
not repeat the target's card-type restriction in its name.

## First Vertical Slice: State Triggers and Sacrifice

### Add

- `StateCondition`
- `StateConditionFunc`
- `When`
- `Controls`
- `ControlsNone`
- `BattlefieldHasNone`
- `AllConditions`
- `AnyCondition`
- `NotCondition`
- the permanent-subject abstraction
- `This`
- `Sacrifice(subject)`

### Remove after migrating callers

- `SacrificeIfControls`
- `SacrificeIfControlsAbility`
- `SacrificeUnlessLand`
- `SacrificeUnlessLandAbility`
- public DSL access to `NewStateTriggered`
- `SacrificeSource`
- `SacrificeSourceStep`
- `SacrificeTarget`
- `SacrificeTargetStep`
- `SacrificeGathered`
- `ControllerHasNoPermanentMatching`
- `NoBattlefieldPermanentMatching`

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

1. Consolidate permanent action verbs: sacrifice, destroy, exile, tap, untap,
   regenerate, attach, and zone movement.
2. Consolidate player action verbs: draw, discard, gain life, lose life,
   poison, mill, and mana production.
3. Consolidate values and selectors, including source, chosen target, event
   source, attached object, players, and stored pipeline values.
4. Consolidate temporary and static continuous modifications, durations,
   conditions, and layers.
5. Consolidate event and turn-step trigger patterns. Add explicit
   intervening-if support.
6. Consolidate activated abilities, costs, targets, timing, activation limits,
   and permissions. Remove `...any` builders.
7. Consolidate replacement and prevention rules without merging them into the
   resolving-effect executor.

For every slice, maintain a migration table that maps each removed export to
its canonical replacement. Do not keep deprecated aliases after all in-tree
callers migrate.

## State-Trigger Engine Requirements

CR 603.8 requires a state trigger to use the stack. It must not trigger again
while its prior instance is pending or on the stack. If that instance resolves,
is countered, fizzles, or otherwise leaves the stack while the condition is
still true, it triggers again at the next state-trigger check.

Implement this by recording the source and ability identity on pending and
stacked state-trigger instances. Clear the outstanding identity through every
stack-leave path. Do not use false-to-true condition transitions as the rearm
rule.

The condition is checked to create the trigger. Removing the Dwarf in response
to Goblins of the Flarg does not stop the sacrifice because the Oracle text has
no intervening-if clause.

## Testing and Acceptance Criteria

- Add compile-focused tests for each public grammar type and constructor.
- Verify that invalid categories cannot be passed to unrelated constructors.
- Test every combinator independently and in nested combinations.
- Verify that composed effects preserve rules text and `EffectProperties`.
- Verify that the same resolving effect works directly, in a sequence, in a
  loop, in a mode, and in a triggered or activated ability.
- Verify that state triggers use the stack and do not duplicate while an
  instance is outstanding.
- Verify re-triggering after resolution, countering, fizzling, and other stack
  removal when the state remains true.
- Verify source-inclusive subtype matching, control changes, continuous type
  changes, and cloned-game isolation.
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
