# Execute the mtgish RON card collection

Status: implementation design for successive AI runs. Baseline: 2026-10-03.

## Objective and completion contract

Use `cards/mtgish/*.ron` as the only production card definitions. Implement the
RON vocabulary through `pkg/mage/dsl/ron` and reusable engine services. Remove
all Go card definitions under `cards/`, then remove unused functions from
`pkg/mage/dsl`. Keep engine behavior and useful regression tests.

The scope is the entire collection, including ante, outside-the-game choices,
and subgames. Parsing or compilation alone does not prove a card works.
Completion requires all of the following:

1. Every record parses, compiles, builds a factory, and passes engine definition
   validation. Reprints share one canonical definition by name.
2. Every rule signature and context has contract evidence. Each unique card has
   a RON-backed behavior test or explicit test mapping that covers all its rules
   and interactions. Vanilla cards can share metadata tests.
3. All applications and tests use the complete RON registry. Invalid collections
   fail startup with useful diagnostics. There is no Go factory fallback,
   skipped record, allowlist, or partial production registry.
4. No Go production card constructor, registration function, or card-specific
   executor remains under `cards/`. A small embedding/registration package and
   RON-backed tests can remain.
5. No obsolete Go-card DSL function remains merely for a deleted caller. Retain
   services needed by RON or the engine. Retire obsolete generators and fixtures
   that force the old authoring API to survive.
6. `make test`, `make build`, `go vet ./...`, and `make lint` pass.
   Native startup smoke tests prove the embedded collection loads.

The removed `implement-card` and `implement-set` skills are not prerequisites.
Do not recreate them or generate Go card stubs. This design supersedes the old
composable DSL migration plan for selecting work. Existing engine contracts
remain useful implementation references.

## Measured baseline

There are 12 files, 2,445 records, and 986 unique names. An AST walk finds 600
tuple constructor signatures and 51 distinct top-level rule names. There are
no conflicting position-free definitions for the same name in this snapshot.
Counts include reprints; they are not separate gameplay implementations.

| File stem | Records |
| --- | ---: |
| LEA | 289 |
| LEB | 291 |
| 2ED | 291 |
| 3ED | 296 |
| CED | 291 |
| CEI | 291 |
| ARN | 78 |
| ATQ | 85 |
| LEG | 309 |
| DRK | 119 |
| FEM | 102 |
| PROMOS | 3 |

`mtgish-ron-vocabulary.json` beside this document contains all syntax signatures,
counts, examples, file hashes, and the loader's first diagnostic for each rejected
record. It is an inventory, not proof of implementation. Symbols include
subtypes, counters, references, and zero-argument actions. Not every symbol
needs a Go function. Names can have multiple roles and argument shapes.

After this change's build repairs, `go run ./cmd/loadcards -json` reports 836
compiled records and 1,609 rejected records. It exits nonzero and publishes no
registry. The compiler stops at the first error within each card, so these
diagnostics hide later gaps. Refresh this baseline after each compiler slice.

### Components and restrictions

- `parser.go`: positioned `Node` AST and `Parse` support streamed multiline RON
  records, comments, symbols, lists, tuples, named fields, strings, and integers.
  Use this parser for inventories, rather than regular expressions.
- `loader.go`: `LoadFS`, `LoadInto`, atomic replacement, structural reprint
  checks, `Issue`, and `LoadError`. Keep strict loading and source diagnostics.
- `compiler.go`: metadata, basic keywords, single targets, selected fixed costs,
  filters, integer/X values, damage/destruction, tap/untap, regeneration,
  sacrifice, drawing, life, mana, and temporary boosts. Support is restricted to
  specific forms; a case label does not mean its entire family is implemented.
- `layers.go`: selected Aura and creature-global boosts and ability grants.
- `triggers.go`: selected source entry/death and controller-upkeep events.
- `vocabulary.go`: adapters that keep action lists with target declarations.
- `cmd/loadcards`: validation CLI, not application startup integration.
- `mage.Registry.ReplaceWith`: atomic installation. Preserve factory validation
  and diacritic name lookup.

The flat `scope` cannot represent multiple target groups, event bindings,
optional costs, action results, entry choices, or saved references. Action lists
allow only one target binding. Spell rules reject needed casting/deck rules.
Filters lack an explicit subject context. Static layers accept only Aura hosts
or creature-global subjects. Triggers expose no typed event environment.
X activation payments and most decision forms are missing.

Audit accepted cases too: protection exceptions must exempt only the named Aura;
player actions must use their explicit player; mana abilities must supply correct
solver metadata; target references must keep their own group identity.

Read `pkg/mage/doc.go` first, then relevant engine code, `docs/dsl-reference.md`,
and `docs/engine-rules-contracts.md`. Use the Comprehensive Rules index to read
only needed sections. Derive card requirements from RON, not old Go definitions.

## Architecture and implementation rules

### Compilation stages and ownership

Use `RON source -> positioned AST -> validated typed plan -> fresh card factory`.
Plans describe rules without game state. Lower them to existing mage actions,
targets, costs, events, layers, replacements, and player decisions.

`ron` owns spelling, argument validation, lexical scope, diagnostics, and lowering.
`mage` owns execution and subsystem state. A missing adapter belongs in `ron`;
a missing game rule belongs in the engine. Do not implement parallel zone, cost,
layer, damage, or event systems in `ron`, or call a Go factory by card name.

The compiler may import `mage` directly. It does not need the broad aliases in
`pkg/mage/dsl`. RON functions need not become exported Go functions one for one.
Keep reusable typed primitives that the compiler actually uses. Avoid reflection,
generic `any` dispatch, and card-name branches.

### Roles, references, and binding environments

Replace the flat scope incrementally with typed environments. Separate:

- Rules, resolving actions, costs, static modifications, replacement operations,
  trigger events, entry operations, and durations.
- Permanent, player, card-in-zone, spell, ability, damage-source and recipient
  references. Keep singular references distinct from collections.
- Permanent/card/spell/player/ability filters; resolution, announcement, cost,
  entry, event, and static values.
- Target, choice, event, payment, action-result, static-choice, loop-member,
  and linked-object bindings.

Add private role-specific compiler functions as needed, such as
`compilePermanentRef`, `compilePlayerRef`, `compileCardRef`, `compileRecipient`,
`compileCondition`, `compileDuration`, and `compileReplacementOperation`.
These are proposals, not existing APIs. Spelling alone does not determine a role:
`And`, `If`, `TapPermanent`, and `SacrificePermanent` occur in different roles.

Each binding has a type, identity, producer, scope, and lifetime. Numbered
`Ref_*` names refer to distinct target groups. `ActionPlayer`, `ActionPermanent`,
and `EachablePermanent` are loop-local. An event supplies only its matching
`Trigger_*` values; a replacement supplies only its matching `Would*` values.
`HostPermanent` needs an attachment contract. Source-zone references need the
corresponding `FromGraveyard`, `FromExile`, or other zone declaration.

Reject unbound, ambiguous, incompatible, and out-of-scope reads. Validate
dominance: a result cannot be used before its producer. Optional payment results
are available only on paid paths. Independent costs, branches, modes, and loop
iterations cannot overwrite one another. Never use a global last-action slot.

Build targets, abilities, costs, and bindings together per factory call, so
effects refer to that instance's declarations. Do not capture mutable target
state in the compiled factory. Preserve isolation when games, spells, tokens,
or abilities are cloned/copied. Share immutable declarations only where the
engine explicitly permits it; never share chosen state.

### Timing, identity, and exact execution

Announcement chooses modes, X, targets, allocations, and casting payments.
Resolution performs choices and actions in the declared order. Static queries
read current state without mutation or RNG. Events capture facts before a
trigger is queued. Entry operations run before entry. Replacements change a
proposed event through the shared replacement pipeline. Intervening-if checks
run at both event time and resolution.

Permanent references keep incarnation identity; leave-and-return makes a new
object. Use LKI only where required. Read controller, owner, characteristics,
and attachment at the specified time. Phasing is not a zone change. Delayed
effects capture their context rather than following a returned source.

Keep simultaneous actions in shared batches. Do not replace damage with life
loss, entry replacements with ETB triggers, or optional payments with forced
payments. Use typed results for actual damage, cards moved, paid objects,
counters removed, and successful outcomes. Costs use transactional validation
and rollback; pure affordability queries cannot make choices or consume RNG.

### Diagnostics and coverage tracking

Add deterministic inventory/report support to `cmd/loadcards`. Walk all AST
nodes, including descendants of unsupported parents. Report expression path,
source position, role, kind/arity, distinct cards, and examples. Map support by
`(role, name, shape, context)`, not function name alone. Report-only inspection
must not weaken strict production loading.

Collect independent errors where possible. Do not fabricate bindings or roles
to check a dependent expression. Sort diagnostics and maintain a stable JSON
schema with nonzero exit for rejected collections.

Track parsed, compiled, engine-validated, and behavior-verified separately.
Create `docs/designs/mtgish-ron-progress.md` in the first implementation run.
Record each capability's ID/status, supported shapes/contexts, dependencies,
engine APIs, affected unique cards, test mappings, validation evidence, source
defects, remaining forms, and next work. A partially supported family remains
partial. Every card needs evidence for all its rules and their interactions.

## Dependency-ordered implementation backlog

The companion inventory is exhaustive for syntax. The following families define
how to implement it. Each phase can require many iterations. Pick a small complete
capability, and inspect every affected card's full record before editing.

| Phase | Actual RON forms and capability | Implementation boundary and evidence |
| --- | --- | --- |
| 0 | Baseline, semantic inventory, validation, instance scopes | Keep accepted cards working; report nested gaps; reject invalid bindings; validate constructed definitions. |
| 1 | `SinglePlayer`, `SinglePermanent`, `Player`, `Permanent`, `EachPermanent`, `EachPlayer`, `ControlledByAPlayer`, `ControlledByPlayer`, `Other`, `IsNamed`, supertype and numeric filters, `Minus`, `Plus`, `Twice`, rounding | Typed references and pure queries. Test owner/controller changes, absent/phased objects, and value evaluation stage. |
| 2 | Numbered `Ref_*`, `NumberTargetPermanents`, `NumberTargetGroupPermanents`, `AnyNumberOfAnyTargets`, player/permanent unions, graveyard/ability targets, opponent choices, `Targeted_Modal`, `Modal_ChooseOne`, `TargetedDistributed` | Announcement and target services. Test independent groups, uniqueness, mode-local targets, X bounds, partial/all-illegal resolution, and retargeting copies. |
| 3 | `If`, `Unless`, `IfElse`, `MayAction`, `MayActions`, choices of objects/colors/names/numbers, `PlayerAction`, `EachPlayerAction`, `ActionForEachPermanent`, `The*ThisWay` results | Typed producers, branch scopes, and player decisions. Test decline/absence, sequencing, saved values, loop identity, and clone isolation. |
| 4 | `PayManaX`, `PayLife`, counters/discard/sacrifice costs, plural payments, `MayCost`, `PlayerMayCost`, `AnyPlayerMayCost`, `RepeatedMayCost`, `ActivatedWithModifiers`, `CastEffect`, casting restrictions and cost changes | Transactional payments and announced X. Separate optional resolution payments from activation costs. Test rollback, distinct paid objects, self-sacrifice permission, limits, timing, and paying player. |
| 5 | Draw/discard/reveal/search/mill, return/exile/sacrifice, counters, tokens, tap/untap, life, `Fight`, library and graveyard order, `CreateTokens`, `TokenWithPT` | Zone/counter/token/damage services. Test hidden information, owners, failed search, shuffle choices, ordering, actual outcomes, and batch actions. |
| 6 | `TriggerA`, `TriggerI`, phase/step, enter/leave/die, cast/activate, mana tapping, attack/block, damage/discard, state triggers, `From*`, history values | Event snapshots, APNAP, LKI, histories, and trigger arming. Test departure, controller changes, event versus resolution checks, distinct pending events, and zone-source abilities. |
| 7 | Permanent/global/sticky layer effects, rule/player effects, `CDA_Power`, `CDA_Toughness`, `AdjustPTX`, `AdjustPTXY`, control/types/colors/abilities | Shared layers, dependencies, timestamps, and rule services. Test CDA/set/modify/switch order, basic land types, ability removal/grants, membership capture, combat and player restrictions. |
| 8 | `AsPermanentEnters`, `EnterAsACopyOfAPermanent`, entry counters/choices, `CopyEffects`, copiable changes, spell copying and word-variable changes | Entry replacements and copiable values. Test exceptions, choices before entry, correct text layer, characteristics in applicable zones, and copy isolation. |
| 9 | `MultipleRecipients`, `DamageToRecipients`, multiple/distributed/divided damage, dynamic/restricted mana, actual damage results | Shared damage batches and mana solver metadata. Test source LKI, partial legality, nonpositive damage, prevention, allocation, simultaneous packets, and spend restrictions. |
| 10 | `PreventDamage`, future/temporary prevention, static/future/temporary `ReplaceWould*`, prevention amounts, redirection, optional payment operations | Proposed-event replacement pipeline, affected-player choices, priorities, and shields. Test rechecks, loops, competing replacements, actual damage, and draw/discard/entry/mana/zone changes. |
| 11 | Delayed triggers and durations, linked exile, graveyard Auras, phasing, face-down casting, combat reassignment, exchange/control/ownership, ante, outside game, subgames, repeated/coin procedures | Explicit subsystem and decision contracts before adapters. Test returned incarnations, attachments, independent links, recursion, deterministic RNG, and AI simulation. No rare card can be skipped. |
| 12 | Embedded RON registry, application cutover, Go-card removal | Migrate tests/consumers, remove definitions and Go-set generation, and prove native startup without Go imports or filesystem assumptions. |
| 13 | Unused DSL removal and final audit | Search all callers; retire exports, wrappers, dead code, and obsolete fixtures in small groups; satisfy the entire completion contract. |

Dependencies override numeric order. Entry copying needs choices and layers;
target bounds need announcement values; payments need result scopes; damage
triggers need source identity; delayed triggers need bindings and durations.
Implement prerequisites first, not incomplete dependent adapters.

Within those dependencies, prioritize shared capabilities that unlock multiple
unique cards. Reassess priority on every run from fresh inventory and loader
evidence; phase numbers and the ledger's next slice are not fixed priorities.
Compare at least three plausible candidates, or all if fewer remain. Record
distinct affected names, estimated full-card unlocks, remaining blockers,
downstream capabilities enabled, and implementation/test effort. Count reprints
once. Read complete records and nested forms to estimate unlocks; first-error
frequency alone does not show whether a card will compile after the change.
Choose the largest credible multi-card benefit for the effort. A shared
prerequisite can take priority without immediate unlocks when it enables a
concrete multi-card capability. Defer single-card capabilities while useful
multi-card work remains, unless they are required prerequisites. Record actual
gains against estimates and rank next candidates after validation.

### Concrete acceptance checkpoints

- `Healing Salve`: mode-local life/prevention targets and explicit players.
- `Swords to Plowshares` and `Crumble`: save characteristics/controller before
  departure; failure to destroy does not erase independent later instructions.
- `Fireball` and `Pyrotechnics`: target costs, multiple identities, rounding,
  announced allocation, and partly illegal targets.
- `Force of Nature`, `Conversion`, `Power Sink`: optional payment, the correct
  paying player, independent payment results, and unpaid branches.
- `Howling Mine`, `Black Vise`, `Hypnotic Specter`: event players, intervening-if
  checks, stored choices, and damage-event facts.
- `Balance`, `Wrath of God`: shared decisions and simultaneous batches.
- `Blood Moon`, `Titania's Song`, `Vesuvan Doppelganger`: type, ability, copy,
  dependency, and P/T interactions. Synthetic contracts may add harder layer
  cases, but only inventory cards count toward card coverage.
- `Animate Dead`, `Tawnos's Coffin`, `Oubliette`, `Cyclopean Tomb`,
  `All Hallow's Eve`: links, graveyard attachment, delayed effects, phasing,
  per-source history, and source-zone behavior.
- `Time Vault`, `Chains of Mephistopheles`, `Illusionary Mask`: turn/draw
  replacements and face-down casting with exact payment restrictions.
- `Shahrazad`, `Word of Command`, `Ring of Ma'rûf`, ante cards: full decision
  and lifecycle services for uncommon mechanics.

These checkpoints do not replace the full inventory or card-to-test mapping.

### Source defects and uncertain semantics

Do not invent semantics to hide a RON transcription error. In the snapshot,
both `Gaea's Liege` branches use `IsntAttacking`; `Fastbond` gates its damage on
`IsFirstLandPlayedByPlayerThisTurn`. Audit these before verifying the cards.
These are observations, not approved corrections or verified Oracle comparisons.

Record card/file/expression, expected behavior, and supporting rules text. Check
local authoritative card data; retrieve current Oracle text if needed. Implement
the reusable capability independently. Correct RON only with evidence, and
update all identical reprints. Unknown argument meaning and contradictory data
remain explicit blockers. Never invert a predicate globally to fix one card.
Preserve existing spelling, including `MayPlayAnyNumberOfLandsDuringThierTurn`,
unless a separate tested format migration changes all uses.

## Runtime cutover and code removal

Place embedded assets in a small `cards/mtgish` Go package next to the RON files;
`go:embed` cannot embed parent directories. It may depend on `ron` and `mage`.
The engine must not import `cards`, which would create test import cycles.
Engine tests use local synthetic cards; integration tests use explicit RON
registries. Expose an error-returning loader, such as `mtgish.LoadInto(registry)`.
Prefer explicit startup calls over side-effect imports that panic.

Compile once per process and publish only a fully validated collection. Do not
merge into a Go registry. Preserve name lookup, IDs, deck/catalog lookup, set
membership, art, AI metadata, and replay behavior. Filename-derived set
membership is separate from the canonical definition registry.

Audit TUI, server, pylib, gametest, scenario tools, oracle replay, and
generators. `cards/all.go` blank-imports Go sets. Gametest, AI tests, and
`dsl/state_migration_test.go` also import individual sets. `cmd/genset` emits Go
DSL definitions. Replace or retire every dependency before deleting definitions.

First compare Go-registered and RON-defined names. Go packages include sets
outside this collection, including Astral, Fourth Edition, Jumpstart, and Secrets
of Strixhaven. Record impacts on fixtures and decks. Migrate shared engine tests
to synthetic fixtures and retained card scenarios to RON. Do not silently drop
regressions, invent RON sets, or keep Go definitions to hide missing names.

Delete production Go definitions in reviewable groups after complete RON support
and startup wiring pass. Move or rewrite useful tests and registration setup.
Test-only constructors must remain fixtures, never production fallbacks.

Then inventory public names in `pkg/mage/dsl/dsl.go` and `composable.go`.
Search the whole repository, including dot imports, templates, docs, and
compile-failure fixtures. Remove unused aliases, wrappers, and implementations
after callers migrate. Retire tests of obsolete authoring APIs; preserve engine
behavior tests. If no production use remains, remove the broad `dsl` package
while keeping `dsl/ron`. Removing an alias does not authorize deleting its
still-used engine service.

Update `AGENTS.md`, `pkg/mage/doc.go`, authoring docs, package comments, and
tooling references during cutover. Replace Go set/`Register()` guidance with RON
authoring and loading. Do not restore removed card/set skills.

## AI iteration procedure

1. Read instructions, this design, engine reference, and current progress.
   Inspect status and preserve unrelated changes. Read full RON requirements.
2. Establish fresh focused tests and loader diagnostics. Inspect all nested
   forms for the selected capability. Separate adapter, engine, data, and
   environment gaps.
3. Compare candidates using the multi-card priority policy above. Select one
   dependency-ready slice with observable behavior, exact shapes, affected cards,
   reuse boundary, and acceptance tests. Do not recreate an implemented function
   because an old plan calls it missing.
4. Write failing tests first and confirm the intended failure. Add declaration
   rejection cases, needed engine contracts, and RON-backed gameplay evidence.
5. Implement the smallest complete capability. Preserve other callers until
   cutover. Write new comments in ASD-STE100 and explain why.
6. Run focused tests with `-count=1`, relevant package tests, `make test`,
   `make build`, `go vet ./...`, `make lint`, and `git diff --check`. Inspect
   automatic lint changes and rerun affected checks.
7. Refresh loader diagnostics; update the ledger and card-to-test mapping.
   Record exact shapes, gameplay evidence, counts, blockers, and next work.
   Compilation gains do not establish gameplay support.
8. Commit only this validated slice when the iteration prompt authorizes commits.
   If blocked, leave it reviewable and report the exact reason. Stop after one
   slice. Emit `MIGRATION_COMPLETE` only with all completion criteria proved;
   otherwise emit `MIGRATION_INCOMPLETE`.

## Initial next slice

The build repairs restore continuous-effect lifecycle separation, layer and
replacement ordering, factory validation, and typed adapters used by the partial
RON compiler. Start with semantic inventory/reporting and scoped reference
plans. Then implement explicit player/permanent references and player-scoped
life actions before dependent modes, choices, and costs. Unsupported cases must
remain errors until their full contracts are implemented.
