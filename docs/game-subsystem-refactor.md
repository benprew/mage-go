# Game Subsystem Refactor

## Status

Proposed.

This document defines a design and implementation plan to separate the
responsibilities of mage.Game into cohesive subsystems. The refactor must
preserve rules behavior, card-facing compatibility, deterministic tests, and
AI search cloning.

## Problem

Game is the aggregate root of the rules engine. It has also become the direct
owner of many unrelated responsibilities:

- player and battlefield queries;
- zone changes and last-known information;
- casting, activation, priority, and stack resolution;
- mana production, planning, and payment;
- damage, prevention, and damage history;
- turn progression and combat bookkeeping;
- continuous effects, replacement effects, and game rules;
- pending, delayed, and state triggers;
- resolution-local values such as X, modes, targets, and event data;
- per-turn and per-duel statistics;
- deterministic random inputs; and
- interactive callbacks.

Splitting methods across game_*.go files improves navigation, but it does not
reduce the responsibilities or API of Game. Callers still depend on a large
concrete type, GameReader continues to grow, and Game.Clone must know how to
copy every field.

This creates several risks:

1. A change in one rules domain can depend on unrelated state.
2. Tests often require a complete game when they need only one subsystem.
3. New mutable state can be omitted from Game.Clone.
4. The public method set grows without a clear ownership rule.
5. A file split can hide architectural coupling without removing it.

## Goals

- Make Game a coordinator of rules domains instead of the implementation of
  every operation.
- Give each mutable field one clear owner.
- Group state with the behavior that maintains its invariants.
- Replace broad dependencies with small consumer-defined interfaces.
- Preserve one authoritative path for each rules mutation.
- Preserve event, replacement, trigger, state-based-action, and priority
  ordering.
- Make cloning an explicit responsibility of every stateful subsystem.
- Allow incremental migration without a repository-wide rewrite.
- Keep card code concise and independent of subsystem implementation details.

## Non-goals

- Do not change Magic rules behavior.
- Do not add cards or mechanics.
- Do not initially split pkg/mage into several Go packages.
- Do not require every existing Game method to disappear.
- Do not replace the effect DSL, action replacement pipeline, or total-cost
  transaction model.
- Do not expose subsystem implementations as the new card-facing API.

## Design Principles

### Game remains the aggregate root

Game owns the complete state of one game. It coordinates operations that cross
subsystem boundaries and remains responsible for construction, cloning,
top-level game flow, and operation ordering.

Game must not become a passive bag of public services.

### State and behavior have the same owner

A subsystem owns both its mutable state and the operations that maintain that
state. Other subsystems must not modify those fields directly.

For example, TriggerSystem owns pending, delayed, and armed state triggers.
TurnSystem can ask it to check or enqueue triggers, but it cannot edit its
queues.

### Dependencies are capabilities

Subsystems depend on the smallest interface needed for an operation. They do
not use a broad service-locator interface that exposes the complete game.

Interfaces should normally be declared near the consumer. This makes each
dependency explicit and prevents a replacement for GameReader from becoming
another general-purpose interface.

### Queries and commands are separate

Pure queries are distinct from commands that change state. Random choices,
payments, and event dispatch are commands even when they return a value.

This distinction matters because trigger conditions, filters, AI evaluation,
and mana planning can evaluate queries more than once.

### Cross-domain workflows have one coordinator

Some operations cannot belong to one subsystem. Casting a spell uses timing,
targets, costs, mana, zones, the stack, events, and triggers. Such workflows
remain orchestration operations on Game or on a focused coordinator owned by
Game.

A coordinator calls subsystem APIs. It does not edit subsystem state.

### Compatibility is temporary and explicit

Existing Game methods can remain as forwarding methods during migration:

    func (g *Game) TryPayMana(playerID uuid.UUID, cost string) bool {
        return g.mana.TryPay(g.manaContext(), playerID, ParseManaCost(cost))
    }

Each forwarding method must have a documented disposition:

- keep as a stable facade;
- migrate internal callers and remove; or
- replace with a card-facing DSL operation.

## Target Architecture

The exact field and type names can change during implementation. The intended
ownership model is:

    Game
      |-- players and player lookup
      |-- ZoneSystem
      |     |-- battlefield and exile coordination
      |     |-- zone transitions
      |     '-- last-known information
      |-- TurnSystem
      |     |-- turn and active player
      |     |-- phase-step schedule
      |     '-- extra turns and skipped steps
      |-- ResolutionState
      |     |-- resolving card and cast context
      |     |-- X, mode, targets, and event data
      |     '-- damage and counter distributions
      |-- TriggerSystem
      |     |-- pending triggers
      |     |-- delayed triggers
      |     '-- armed state triggers
      |-- ManaSystem
      |     |-- source discovery and planning
      |     |-- mana production
      |     '-- automatic payment support
      |-- DamageSystem
      |     |-- damage execution
      |     |-- prevention coordination
      |     '-- aggregation and history
      |-- TrackerSystem
      |     |-- per-turn trackers
      |     '-- per-duel trackers
      |-- RandomSource
      |-- Stack
      |-- Combat
      '-- EffectManager
            |-- continuous and replacement effects
            '-- GameRules

Player-owned storage remains player-owned. Players continue to own life totals,
mana pools, hands, libraries, graveyards, and ante zones. ManaSystem owns mana
rules and workflows, not each player's pool. ZoneSystem coordinates moves
without centralizing all zone storage.

## Subsystem Responsibilities

### ZoneSystem

Owns:

- battlefield storage and copy-on-write metadata;
- exile storage and visibility metadata;
- the entering permanent reference;
- last-known-information snapshots; and
- bookkeeping that must remain consistent with zone storage.

Provides battlefield and card lookup, mutable permanent access, battlefield
entry and removal, zone moves, phasing, attachment cleanup, and LKI lookup.

Zone operations must continue to coordinate replacements, continuous effects,
events, triggers, ownership, and object identity. Moving storage must not bypass
those workflows.

### TurnSystem

Owns the current turn, active player, current phase step, turn schedule, extra
turns, and skipped turn or step state.

Provides turn and phase queries, schedule mutation, step advancement, turn
advancement, and cleanup lifecycle hooks.

The top-level progression loop can remain on Game until priority, combat,
triggers, state-based actions, and mana-emptying boundaries have stable APIs.

### ResolutionState

Owns temporary context for the object currently resolving:

- X and selected mode;
- triggering event amount and source;
- resolving card, cast zone, and cast context;
- targets and color overrides;
- damage and counter distributions;
- the last sacrificed object; and
- cost-local values read during resolution.

It provides scoped installation and restoration of resolution context. Prefer
one scoped operation, or an explicit begin/end pair with deferred cleanup, over
independent field assignments. Errors and early returns must not leave stale
resolution state.

### TriggerSystem

Owns pending triggers, delayed triggers, state-trigger arming state, and future
trigger scheduling or suppression state.

Provides event inspection, pending-trigger creation, state-trigger checks,
active-player/nonactive-player ordering, stack placement, and delayed-trigger
registration and expiration.

Rules events and presentation callbacks remain distinct. A rules event can
create triggers. A presentation event reports a completed change and must not
become a second rules path.

### ManaSystem

Owns reusable mana-planning scratch state and caches whose lifetime is limited
to one planning operation.

Provides mana-source discovery, production metadata, mana ability activation,
solution planning, automatic payment support, and tapped-for-mana events.

Mana pools remain on Player. Total-cost payment remains a cross-domain
transaction because it combines mana and nonmana costs.

### DamageSystem

Owns combat-step damage aggregation, per-turn damage history, persistent damage
relationships, and execution state such as the combat-damage flag.

Provides damage to players and permanents, replacement and prevention
coordination, aggregation, history queries, and damage-related events.

Damage remains a typed replaceable action. The subsystem must preserve the
current order of replacement, prevention, execution, life loss, damage marking,
events, and state-based actions.

### TrackerSystem

Owns observations of completed game actions that do not define the primary
state transition. It contains separate TurnTrackers and DuelTrackers values so
their reset policies are visible.

It provides typed recording and query methods. Callers do not edit tracker maps
directly.

### RandomSource

Owns scripted integer results, scripted coin-flip results, and the fallback
random implementation.

Random operations are commands, not read-only queries. Clones consume
independent scripted input sequences.

### Existing components

Stack, Combat, EffectManager, GameRules, and TurnSchedule already represent
partial subsystem boundaries. Strengthen these boundaries instead of replacing
working abstractions only for consistency.

EffectManager currently combines continuous effects, replacement effects, and
derived rules. A later proposal can decide whether to split it. That decision
is not required for the initial Game refactor.

## Responsibilities That Remain on Game

Methods remain on Game when they coordinate several subsystems or form a stable
external entry point. Expected examples include:

- game construction and complete-game cloning;
- starting, stopping, and determining the outcome;
- top-level rules-engine advancement;
- casting a spell and activating an ability;
- resolving a stack object;
- priority and state-based-action loops; and
- intentional compatibility facade methods.

Simple domain operations do not remain implemented on Game. A temporary
forwarder is acceptable, but the subsystem is the source of truth.

## Query Interfaces

Replace GameReader incrementally with small capabilities:

    type PlayerReader interface {
        GetPlayer(uuid.UUID) Player
        GetOpponent(uuid.UUID) Player
        ActivePlayerObj() Player
    }

    type BattlefieldReader interface {
        FindPermanent(uuid.UUID) *Permanent
        FilterBattlefield(PermanentFilter) []*Permanent
        CountBattlefield(PermanentFilter) int
    }

    type ResolutionReader interface {
        XValue() int
        ModeValue() int
        EventAmount() int
        EventSourceID() uuid.UUID
    }

    type TurnReader interface {
        CurrentTurn() int
        CurrentStep() PhaseStep
    }

A consumer combines only the capabilities it needs:

    type TriggerContext interface {
        PlayerReader
        BattlefieldReader
        LKIView
    }

The final interfaces must follow actual call sites. Do not create speculative
interfaces with no consumer.

During migration, GameReader can embed the smaller interfaces and retain
compatibility queries. Remove obsolete members after consumers use narrow
types. Remove GameReader itself if it no longer has a coherent purpose.

## Command Interfaces and Card API

Cards must not receive unrestricted access to every subsystem. Calls such as
ctx.Game.Mana().TryPay would turn Game into a service locator and couple cards
to the engine structure.

Card effects should continue to prefer effect constructors and the DSL. When a
custom effect needs engine operations, EffectContext should eventually provide
curated query and command capabilities:

    type EffectContext struct {
        Query      EffectQuery
        Actions    EffectActions
        SourceID   uuid.UUID
        Controller uuid.UUID
        Targets    []uuid.UUID
        Vars       map[string]any
    }

    type EffectActions interface {
        DealDamage(DamageSpec) error
        MoveCard(ZoneMoveSpec) error
        AddMana(ManaSpec) error
        FireEvent(GameEvent)
    }

The interface describes rules operations, not subsystem getters. Existing
ctx.Game access can remain during migration. New card code should prefer the
DSL or a narrow operation when one exists.

## Operation Ordering

The refactor must preserve observable rules order. Moving code must not change
when these actions occur:

1. validate an attempted operation;
2. create a replaceable action;
3. apply replacement and prevention effects;
4. commit the state mutation;
5. update LKI and trackers;
6. emit rules events;
7. enqueue triggered abilities;
8. apply continuous effects;
9. check state-based actions; and
10. grant priority or continue resolution.

Not every operation uses every step, and some rules require a different local
order. Existing tests and rule-specific behavior remain authoritative. This is
a review checklist, not a replacement for the rules.

Subsystem contracts must make ordering visible. A subsystem method must not
silently run a priority round or state-based-action loop unless that behavior
is part of its explicit contract.

## Cloning and Search

Every stateful subsystem defines its own clone behavior. Game.Clone delegates
instead of copying private subsystem fields:

    func (g *Game) Clone() *Game {
        clone := &Game{
            zones:      g.zones.CloneForSearch(),
            turn:       g.turn.Clone(),
            resolution: g.resolution.Clone(),
            triggers:   g.triggers.Clone(),
            damage:     g.damage.Clone(),
            trackers:   g.trackers.Clone(),
            random:     g.random.Clone(),
        }
        clone.rebindSubsystems()
        return clone
    }

Each subsystem classifies its fields as:

- immutable and safe to share;
- copied by value;
- deep-copied;
- copy-on-write; or
- runtime-only and cleared in search clones.

Battlefield copy-on-write behavior must remain intact. If a subsystem stores a
Game back-reference, construction and cloning must rebind it. Prefer passing a
narrow context when this avoids back-reference management.

Clone tests mutate both the original and clone. They cover slices, maps, nested
maps, queues, resolution context, random inputs, and permanent copy-on-write
state.

## Package and File Strategy

Keep the first implementation in package mage. This establishes ownership
without immediately creating package cycles among effects, costs, permanents,
filters, and game operations.

Possible files are:

    pkg/mage/game.go
    pkg/mage/game_clone.go
    pkg/mage/zone_system.go
    pkg/mage/turn_system.go
    pkg/mage/resolution_state.go
    pkg/mage/trigger_system.go
    pkg/mage/mana_system.go
    pkg/mage/damage_system.go
    pkg/mage/tracker_system.go
    pkg/mage/random_source.go

File names are secondary to ownership. One subsystem can use several files,
but it must still expose one coherent API.

Move a subsystem into an internal package only after its dependencies form a
stable, acyclic boundary. Package extraction is a separate decision and must
not block this refactor.

## Lessons from Forge

Forge validates the use of an aggregate Game that composes specialized phase,
trigger, replacement, stack, zone, and mana components. This proposal adopts
that direction.

Forge also shows two failure modes to avoid:

1. A broad GameAction component can become another god object.
2. Handlers with a concrete reference to the complete game retain most of the
   original coupling.

mage-go should have domain command surfaces instead of one catch-all action
service. Subsystems should accept narrow capabilities where practical, and
each stateful subsystem should own its clone implementation.

## Implementation Plan

The work proceeds in small, behavior-preserving changes. Every phase leaves the
repository buildable and tested.

### Phase 0: Inventory and guardrails

1. Inventory all Game fields and methods.
2. Assign each item to a subsystem or aggregate coordination.
3. Record direct field access outside the intended owner.
4. Identify public APIs used by cards, the DSL, interactive clients, and AI.
5. Document event and mutation ordering for high-risk workflows.
6. Add regression tests where current ordering is not protected.

Deliverable: an ownership table and migration list. No behavior moves in this
phase.

### Phase 1: Extract ResolutionState

1. Create ResolutionState with all resolving-object fields.
2. Move setup and cleanup into scoped operations.
3. Forward existing Game resolution queries to the component.
4. Update stack resolution to install and clear context through one API.
5. Add clone and stale-state regression tests.

Exit criteria:

- no resolution-context field remains directly on Game;
- early returns cannot leak resolution state; and
- existing card-facing queries behave identically.

### Phase 2: Extract TrackerSystem

1. Create separate TurnTrackers and DuelTrackers values.
2. Move recording, queries, resets, and previous-turn snapshots.
3. Replace direct map writes with typed methods.
4. Delegate cloning to each tracker value.
5. Preserve existing query methods as temporary forwarders.

Exit criteria:

- tracker reset lifetimes are explicit;
- no external caller edits a tracker map; and
- original and cloned games have independent tracker state.

### Phase 3: Extract RandomSource

1. Move scripted coin and integer results into RandomSource.
2. Move random consumption and deterministic setup operations.
3. Remove random mutation from read-only interfaces.
4. Update selectors and effects to use a random command capability.
5. Test deterministic order and clone independence.

Exit criteria:

- read-only conditions cannot consume random state through their interface;
- all random paths use one source; and
- existing deterministic tests remain stable.

### Phase 4: Extract TriggerSystem

1. Move pending, delayed, and armed trigger state.
2. Move trigger discovery and queue operations.
3. Define narrow query and stack-command dependencies.
4. Keep Game.FireEvent as a compatibility coordinator if needed.
5. Preserve active-player/nonactive-player ordering and controller snapshots.
6. Test event triggers, state-trigger rearming, delayed triggers, zone-active
   triggers, and clone independence.

Exit criteria:

- trigger queues have one owner;
- event inspection cannot mutate unrelated state; and
- trigger ordering is unchanged.

### Phase 5: Extract ManaSystem

1. Move source discovery, production, planning, and scratch state.
2. Leave mana pools on Player.
3. Define dependencies for permanent lookup, player lookup, payment, and event
   dispatch.
4. Keep total-cost transactions as cross-domain coordinators.
5. Retain public helpers as forwarders until callers migrate.
6. Test restricted mana, conversions, hybrid costs, combinations, dynamic
   production, post-production effects, and tapped-for-mana events.

Exit criteria:

- planning does not depend on unrelated Game methods;
- payment remains transactional; and
- manual and automatic mana activation preserve their required equivalence.

### Phase 6: Extract DamageSystem

1. Move damage execution state, aggregation, and history.
2. Define dependencies on replacements, players, permanents, LKI, events,
   trackers, and callbacks.
3. Keep state-based-action checks under the explicit coordinator.
4. Preserve combat and noncombat distinctions.
5. Test prevention, redirection, reflection, life loss, lethal damage, combat
   aggregation, LKI, and cloning.

Exit criteria:

- all damage mutations use the subsystem;
- damage history has one owner; and
- replacement and event order is unchanged.

### Phase 7: Extract ZoneSystem

This is the highest-risk state extraction and occurs after the supporting
interfaces are stable.

1. Move battlefield and exile ownership into ZoneSystem.
2. Move copy-on-write permanent access and entering-permanent state.
3. Move LKI storage and lookup.
4. Route all zone changes through typed operations.
5. Keep player-owned zones on Player.
6. Preserve identity, ownership, controller initialization, attachments,
   phasing, exile visibility, and zone-change events.
7. Add broad transition and clone tests.

Exit criteria:

- direct battlefield and exile mutations exist only inside ZoneSystem;
- each modeled transition has one authoritative implementation; and
- copy-on-write remains correct in both clone directions.

### Phase 8: Consolidate TurnSystem

1. Move turn, active-player, schedule, and extra-turn state.
2. Move domain-local turn queries and mutations.
3. Keep cross-domain progression on Game until its workflow is clear.
4. Move that loop only if TurnSystem can use narrow capabilities without
   becoming a god object.
5. Test skipped steps, extra turns, cleanup repetition, mana emptying,
   priority, and combat transitions.

Exit criteria:

- turn state has one owner;
- phase ordering is unchanged; and
- cleanup resets each subsystem through explicit lifecycle hooks.

### Phase 9: Shrink compatibility surfaces

1. Update internal callers to use narrow subsystem APIs.
2. Replace card calls with DSL constructors or curated effect commands.
3. Remove obsolete Game forwarding methods.
4. Split or remove GameReader after consumers use smaller capabilities.
5. Update pkg/mage/doc.go and the main architecture document.
6. Evaluate stable internal-package boundaries.

Exit criteria:

- each remaining Game method is an aggregate operation, stable facade, or
  necessary shared query;
- no replacement god interface exists; and
- subsystem dependency directions are documented and acyclic.

## Testing Strategy

For each implementation phase:

1. Add or identify focused tests for the behavior being moved.
2. Run them before the move to establish the baseline.
3. Move one coherent responsibility.
4. Run focused package tests.
5. Run all engine tests.
6. Run the full repository test suite.
7. Run lint.

Cross-cutting checks include:

- event and trigger ordering;
- replacement-effect application order;
- state-based-action timing;
- cleanup and end-of-turn expiration;
- original-versus-clone independence;
- deterministic AI search outcomes where fixtures exist;
- DSL and card-package compatibility; and
- absence of import cycles.

No phase is complete if tests pass only after changing expected rules behavior,
unless that behavior change is separately justified as a bug fix.

## Review Checklist

- Does the subsystem own all state needed to maintain its invariants?
- Can another component mutate that state directly?
- Does the subsystem accept only the capabilities it needs?
- Did a catch-all interface or action object appear?
- Is operation ordering explicit?
- Is clone behavior defined for every field?
- Are cleanup and reset lifetimes explicit?
- Are forwarding methods marked for removal or retention?
- Did card code become more coupled to engine internals?
- Do focused, engine, repository, and lint checks pass?

## Risks and Mitigations

### Circular dependencies

Keep the first extraction inside package mage, use consumer-defined capability
interfaces, and leave cross-domain workflows on Game.

### Hidden behavior changes

Moving a method can change when events, effects, or state-based actions run.
Record ordering before each high-risk move and protect it with focused tests.

### Clone omissions

Require a clone method and mutation-based independence tests for every stateful
subsystem. Game.Clone must not inspect private subsystem fields.

### Replacement god objects

Do not create a general GameAction, GameServices, or GameContext that contains
every command and query. Review subsystem size, dependencies, and method count
during each phase.

### Excessive forwarding methods

Track every compatibility method in the Phase 0 inventory. Remove internal
forwarders when their callers migrate. Keep only intentional external facades.

### Premature package extraction

Separate ownership before packages. Package moves should follow proven
dependency boundaries instead of being used to discover them.

## Completion Criteria

The refactor is complete when:

- Game primarily constructs, coordinates, clones, and exposes intentional
  stable entry points;
- each mutable field has one documented subsystem owner;
- domain mutations do not edit another subsystem's private state;
- broad query and command interfaces are replaced with small capabilities;
- random mutation is absent from read-only interfaces;
- each stateful subsystem owns and tests its clone behavior;
- card code uses the DSL or curated operations instead of service lookup;
- event, replacement, trigger, priority, and state-based-action order remains
  unchanged;
- no catch-all action service replaces the old Game method set; and
- the full test suite and lint pass.

## Recommended First Change

Start with Phase 0 and Phase 1 only. Produce the ownership inventory, add any
missing resolution-context regression tests, and extract ResolutionState.

This provides a small end-to-end example of subsystem ownership, compatibility
forwarding, scoped cleanup, and delegated cloning before high-risk rules
behavior moves.
