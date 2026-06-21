# Card Effect DSL Guide

This guide teaches the **data-driven effect DSL** — the recommended way to define
card effects in this engine. It replaces older closure-based patterns
(`FuncEffect(func(g *Game, ...) error { ... })`) with composable data values
that the executor interprets.

If you've read `pkg/mage/doc.go`, this is the practical companion. `doc.go` is
the broad API reference; this guide focuses specifically on **how to compose
effects**.

> **Note:** `doc.go`'s "Effects" section currently still describes the old
> `Effect.Apply(g, sourceID, controller, targets) error` interface. That method
> has been removed — `Effect` now only requires `Text()` and `Properties()`.
> Execution flows through `ApplyEffect(g, e, ...)` and the executor's type
> switch. The rest of `doc.go` is accurate.

---

## Mental model

A card effect is **inert data**. It says *what* should happen — "deal 3
damage," "exile the target," "the controller gains life equal to its power" —
without knowing *how* to mutate the game.

The engine has a single dispatch point, `ExecuteEffect(ctx, e)`, which type-
switches over the concrete effect type and performs the mutation. Card code
never sees `*Game`. Pipelines just sequence effect values.

```go
// What you write — pure data:
Pipeline("exile target creature; controller gains life equal to its power",
    EffectProperties{Outcome: OutcomeDetriment},
    SnapshotPermanent(SelectTarget, "victim"),
    ExileGathered("victim"),
    GainLifeFromVar("victim.controller", "victim.power"),
)

// What runs at resolution — the executor walks the pipeline,
// dispatches each step, and threads an EffectContext between them.
```

The `Effect` interface itself is minimal:

```go
type Effect interface {
    Text() string
    Properties() EffectProperties
}
```

`EffectData` is a type alias for `Effect`. `DataEffect()` and `UnwrapEffect()`
are identity functions kept for backward compatibility — **don't wrap with
them in new code**.

---

## The five composition primitives

These are the building blocks. Memorize what each does and you can read most
cards in the codebase.

### 1. `Pipeline(text, props, steps...)` — sequence steps with shared state

Each step runs in turn, sharing an `EffectContext` so earlier steps can hand
values to later ones.

```go
Pipeline("destroy target artifact; controller gains life equal to its CMC",
    EffectProperties{Outcome: OutcomeDetriment},
    SnapshotPermanent(SelectTarget, "target"),     // stash properties under "target"
    DestroyGatheredNoRegen("target"),              // destroy the snapshotted permanent
    GainLifeFromVar("target.controller", "target.cmc"),
)
```

Use `Pipeline` whenever a later step needs to read state captured by an earlier
step *before* the earlier step's mutation lands. Snapshots are the typical case
(see "Variables and snapshots" below).

### 2. `IfElse(text, cond, then, else)` — branch on a condition

```go
IfElse("sacrifice if no creatures",
    NoBattlefieldPermanentMatching{Filter: IsCreature},
    SacrificeSourceStep(),
    nil,                  // nil else branch is fine
)
```

`cond` is a `TriggerConditionData` value — the same predicate vocabulary used
for trigger conditions (see "Trigger conditions" below), evaluated against
game state at resolution time. The predicate receives a zero-value event, so
only state predicates make sense here; event predicates like
`EventSourceIsSelf` always see empty event fields. Common ones:

- `SourceIsTapped{}` / `SourceIsUntapped{}` — source tap state.
- `NoBattlefieldPermanentMatching{Filter: ...}` — no permanent matches.
- `SourceHasCounterCond{CounterType: ..., MinCount: 1}` — counter check.
- `FlipCoinCond{}` — coin flip. Resolution-only: each evaluation is a fresh
  flip, so never use it as a trigger or state-trigger condition.
- `NotTriggerCond{Inner: ...}`, `AndTriggerCond`, `OrTriggerCond` — combinators.

For branching on a pipeline variable gathered by an earlier step, use
`IfVarGT(text, name, value, then, else)` instead — variables live in the
pipeline's `EffectContext`, not in game state:

```go
IfVarGT("gain life if credits > 0", "credits", 0,
    GainLifeFromVar("src.controller", "credits"),
    nil,
)
```

"You may pay {cost}. If you do, ..." and "... unless you pay {cost}" clauses
are not conditions — they prompt the player and pay costs. Use
`EffectIfPaid(cost, effect)` and `UnlessTargetPays(payer, cost, prompt,
ifNotPaid)`.

### 3. `ModalEffect(text, modes...)` — choose-one effects

```go
c := NewInstant("Healing Salve", "{W}",
    NewTargetedSpell(TargetAnyTarget(), ModalEffect(
        "target player gains 3 life or prevent the next 3 damage to any target this turn",
        GainLifeTarget(Fixed(3)),
        PreventDamageToTarget(Fixed(3)),
    )),
)
c.SetModes([]string{
    "Target player gains 3 life",
    "Prevent the next 3 damage that would be dealt to any target this turn",
})
```

`SetModes` provides the menu strings shown to the choosing player. The mode
index is read from `g.ModeValue()` at resolution.

### 4. `ForEachPermanent(filter, inner, text)` — iterate

Run `inner` once per permanent matching `filter`, with the matched ID set as
`targets[0]` for that iteration.

```go
ForEachPermanent(IsCreature,
    DealDamageStep(Fixed(1)),
    "deal 1 damage to each creature",
)
```

Combat-specific iterators live in `effect_pipeline_ext.go`:
`ForEachBlockerOfSource`, `ForEachAttackerBlockedBySource`,
`ForEachCombatOpponent`, etc.

### 5. `CompositeEffects(text, effects...)` — fan out to many effects, no shared state

When steps don't need to communicate via context vars, `CompositeEffects` is
lighter than `Pipeline`:

```go
CompositeEffects("deal 1 damage to each creature and each player",
    DealDamageToAllCreatures(Fixed(1), PermanentFilter{}),
    DealDamageToPlayers(Fixed(1), SelectEachPlayer()),
)
```

Rule of thumb: **`CompositeEffects` for parallel actions, `Pipeline` for stages
that depend on each other.**

---

## Variables and snapshots

Pipelines share an `EffectContext` with a `Vars map[string]any`. The convention
is:

1. A **Snapshot** step reads properties off a permanent (or hand size, or
   counter count) and writes them under a chosen name.
2. A later step reads those vars by name.

The most common shape: snapshot before mutation, use the snapshotted values
after. This matters when the mutation removes the permanent — Swords to
Plowshares can't read `target.power` *after* exile.

```go
// Swords to Plowshares
Pipeline("exile target creature. Its controller gains life equal to its power",
    EffectProperties{Outcome: OutcomeDetriment},
    SnapshotPermanent(SelectTarget, "victim"),
    ExileGathered("victim"),
    GainLifeFromVar("victim.controller", "victim.power"),
)
```

`SnapshotPermanent` writes a fixed set of keys with the prefix you pass:
`"victim.power"`, `"victim.toughness"`, `"victim.cmc"`, `"victim.controller"`,
`"victim.name"`, `"victim.id"`. `SnapshotAttached` does the same for the
permanent the source is attached to.

Snapshot the permanent's **ID** as well — many "gathered" operations
(`DestroyGathered`, `ExileGathered`, `BounceGathered`, etc.) expect a UUID
stored under the var name and operate on it.

---

## Targets, selectors, value sources

Effect data is decoupled from "who" and "how much" via three indirections:

### Targets — `TargetSelector`

Used by chainable effects (`Boost(...).Targeting(x)`) to decide *which*
permanent the effect lands on. You'll usually pick one of:

- `ToTarget()` — `targets[0]` (the spell's first chosen target). **Default.**
- `ToSource()` — the permanent generating the effect (e.g., self-pump).
- `ToAttached()` — the permanent the source is attached to (auras/equipment).
- `ToGathered("name")` — read UUID from a context var.
- `ToMatching(filter)` / `ToAllMatching(filter)` — every matching permanent.

### Players — `PlayerSelector`

Decides *who*: `SelectController()`, `SelectActivePlayer()`,
`SelectEachPlayer()`, `SelectEachOpponent()`, `SelectAttachedController()`,
`SelectTargetPlayer()`, `VarPlayer("name")` (read UUID from context).

### Numbers — `ValueSource`

Decides *how much*. `Fixed(3)` for a constant; `XValue()` for the X paid;
`EventAmountValue()` for triggered effects that read damage/life amounts;
`CountBattlefield(who, filter)` for a permanent count; `VarInt("name")` for a
context lookup; `Add`/`Mul`/`HalfRoundUp` for arithmetic.

---

## Chainable effects

Several effects use a fluent builder. They default sensibly (target =
`targets[0]`, duration = end of turn for boosts) but you can override.

### Permanent-targeting (use `TargetSelector`)

`Boost`, `GrantKeyword`, `GrantType`, `AddCounters`, `RemoveCounters`:

```go
Boost(Fixed(2), Fixed(2))                                   // +2/+2 to targets[0] until EOT
Boost(Fixed(1), Fixed(0)).Targeting(ToSource())             // self-pump
GrantKeyword(Flying).Targeting(ToSource()).Until(Indefinite) // permanent flying
GrantType(TypeCreature).Until(EndOfTurn)                    // animate
AddCounters(Charge, Fixed(1))                               // +1 charge counter on targets[0]
AddCounters(P1P1, Fixed(1)).Targeting(ToSource()).Max(3)    // up to 3 +1/+1 counters
```

### Player-targeting (use `PlayerSelector`)

`DrawCards`, `DiscardCards`, `MillTargetPlayer`, `PoisonTargetPlayer` all
default to acting on `targets[0]` (the spell's chosen player) but accept
`.Targeting(playerSelector)` to override:

```go
DrawCards(Fixed(2))                                         // targets[0] draws 2
DrawCards(Fixed(1)).Targeting(SelectController())           // controller draws 1
DiscardCards(Fixed(3)).Targeting(SelectDefendingPlayer())   // defending player discards 3
MillTargetPlayer(Fixed(5)).Targeting(SelectActivePlayer())  // active player mills 5
PoisonTargetPlayer(1).Targeting(SelectEachOpponent())       // each opponent gets 1 poison
```

Available `PlayerSelector`s include `SelectController`, `SelectActivePlayer`,
`SelectEachPlayer`, `SelectEachOpponent`, `SelectAttachedController`,
`SelectDefendingPlayer` (for triggers on attackers), `SelectTargetPlayer`,
`SelectTargetPermanentController`, and `VarPlayer("name")` for context-bound
players.

This is how Mindstab Thrull's "defending player discards three cards" effect
is wired:

```go
WithAbility(NewTriggered(EvtBlockersDecl, /*optional=*/true,
    SacrificeSource(),
    DiscardCards(Fixed(3)).Targeting(SelectDefendingPlayer()),
).SetConditionData(SourceIsUnblockedAttacker{}))
```

> **Note:** `Targeting` on permanent-targeting effects takes a `TargetSelector`
> (constructed with `ToSource`/`ToTarget`/etc.). On player-targeting effects it
> takes a `PlayerSelector` (constructed with `SelectController()`/
> `SelectEachOpponent()`/etc.). Same method name, different selector type.

---

## Triggers and conditions

Triggered abilities use the same data-driven philosophy. The body is just an
`Effect`; the trigger is a `TriggerCondition` value.

```go
// Whenever this creature deals combat damage to a player, draw a card.
WithAbility(NewTriggered(
    core.EvtDamageToPlayer,
    /*persistent=*/false,
    DrawCards(Fixed(1)),
    AndTriggerCond{Conditions: []TriggerConditionData{
        EventSourceIsSelf{},
        EventSourceIsSelfDamageToPlayer{},
    }},
))
```

Predicates are atomic, composable, and live in `trigger_condition_data.go`:

- **Source identity:** `EventSourceIsSelf`, `EventSourceNotSelf`,
  `EventSourceControlledByController`, `EventSourceControlledByOpponent`.
- **Source state:** `SourceIsTapped`, `SourceIsUntapped`, `SourceNotSummonSick`,
  `SourceOnBattlefield`.
- **Combat:** `SourceIsBlockedAttacker`, `SourceIsUnblockedAttacker`,
  `SourceIsBlockingInCombat`, `SourceInCombat`,
  `SourceBlockedByCreatureMatching{Filter}`.
- **Type matching:** `EventSourceHasType{Type}`,
  `EventSourceMatchesPermanentFilter{Filter}`.
- **Combinators:** `AndTriggerCond`, `OrTriggerCond`, `NotTriggerCond`.

Convenience trigger constructors wrap common shapes:
`AttacksTrigger(effect, persistent)`, `BlocksTrigger`, `ETBTrigger`,
`DiesTrigger`, `BeginningOfEachEndStepTrigger`,
`DealsDamageToOpponentTrigger`, `WheneverSpellCastTrigger`. See `triggered.go`.

---

## Cookbook

Real cards from the codebase. Read these first when implementing something
similar.

### Snapshot → mutate → use snapshotted value

**Swords to Plowshares** (`cards/limited/spells.go`)

```go
NewInstant("Swords to Plowshares", "{W}",
    NewTargetedSpell(TargetCreature(), Pipeline(
        "exile target creature. Its controller gains life equal to its power",
        EffectProperties{Outcome: OutcomeDetriment},
        SnapshotPermanent(SelectTarget, "victim"),
        ExileGathered("victim"),
        GainLifeFromVar("victim.controller", "victim.power"),
    )),
)
```

### Coin-flip branch

**Bottle of Suleiman** (`cards/arabian/artifacts.go`)

```go
WithActivatedAbility(
    IfElse("flip coin: 5/5 Djinn or 5 damage",
        FlipCoinCond{},
        CreateToken("Djinn", 5, 5,
            []CardType{TypeArtifact, TypeCreature}, []string{"Djinn"}, Flying),
        DealDamageToPlayers(Fixed(5), SelectController()),
    ),
    ManaCostOf("{1}"),
    WithCost(SacrificeSourceCost()),
)
```

### Modal spell

**Healing Salve** — see "ModalEffect" above.

### Conditional sacrifice trigger

**Pestilence** (`cards/limited/spells.go`)

```go
NewEnchantment("Pestilence", "{2}{B}{B}",
    WithAbility(BeginningOfEachEndStepTrigger(
        IfElse("sacrifice if no creatures",
            NoBattlefieldPermanentMatching{Filter: IsCreature},
            SacrificeSourceStep(),
            nil,
        ), false,
    )),
    WithActivatedAbility(
        CompositeEffects("deal 1 damage to each creature and each player",
            DealDamageToAllCreatures(Fixed(1), PermanentFilter{}),
            DealDamageToPlayers(Fixed(1), SelectEachPlayer()),
        ),
        ManaCostOf("{B}"),
    ),
)
```

### Aura that snapshots its host

**Thrull Retainer** (`cards/fallen_empires/enchantments.go`)

```go
NewAura("Thrull Retainer", "{B}",
    WithStaticAbility(BoostAttached(1, 1, AttachAura)),
    WithActivatedAbility(
        Pipeline("Regenerate enchanted creature",
            EffectProperties{},
            SnapshotAttached("attached"),
            RegenerateGathered("attached"),
        ),
        SacrificeSourceCost(),
    ),
)
```

### Self-pump activation

```go
WithActivatedAbility(
    Boost(Fixed(2), Fixed(2)).Targeting(ToSource()),
    SacrificeArtifactCost(),
)
```

### Delayed trigger from a pipeline

A common combat pattern: "when ~ attacks, at end of combat, sacrifice it and
deal 5 damage."

```go
WithAbility(AttacksTrigger(
    RegisterDelayedTriggerStep(EvtEndOfCombat, "",
        SacrificeSource(),
        DealDamageToPlayers(Fixed(5), SelectController()),
    ),
    /*persistent=*/false,
))
```

---

## When to fall back to `FuncEffect`

The DSL doesn't cover everything. Fall back to a closure when:

- The card captures **per-instance state** that no shared primitive expresses
  (Stangg's twin token ID, a chosen-color filter that combines fields).
- The mechanic **requires a primitive that doesn't exist yet** — adding one is
  the right move when you'll see it on more than one card.
- A **set-specific check** (e.g., "from Arabian Nights") doesn't fit a generic
  filter.

Mark the spot with `// XXX: <why>` and consider whether the missing primitive
is worth adding. Per `CLAUDE.md`, never silently simplify the rules to fit the
DSL — wrong-but-simple is wrong.

```go
// XXX: needs ChosenColor + ChosenPlayer + nontoken multi-field filter primitive.
SetCondition(func(g *Game, ...) bool { ... })
```

The migration doc (`active-design-docs/data-driven-effects-migration.md`)
tracks which closure patterns still lack DSL equivalents.

---

## Where to look next

- **`pkg/mage/doc.go`** — broad engine API: card constructors, costs, options,
  layers, replacement effects, the `*Game` API surface.
- **`pkg/mage/effect_pipeline.go`**, **`effect_pipeline_ext.go`** — pipeline
  primitives. Read these when writing a non-trivial card; the public functions
  are short and self-explanatory.
- **`pkg/mage/trigger_condition_data.go`** — every trigger predicate.
- **Existing converted cards** as templates. Good starting points:
  Swords to Plowshares, Pestilence, Crumble, Bottle of Suleiman, Thrull
  Retainer, Healing Salve, Time Elemental.
- **`active-design-docs/data-driven-effects-migration.md`** — migration status,
  what's converted, what closure patterns still lack primitives.
