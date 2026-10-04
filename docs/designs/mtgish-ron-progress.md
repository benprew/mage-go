# mtgish RON execution progress

## Run 2026-10-03: P0-INVENTORY

Status: complete semantic inventory/reporting slice. Scope/instance prerequisites
remain unfinished; this run does not start another slice. The working tree began
with staged skill deletions, changes to engine/registry/docs, and untracked RON
sources, compiler, loader, design, and local scripts. These were preserved.

### Contract and dependencies

`InspectFS(fs.FS, string) (*Inventory, error)` reports every parsed AST node,
including descendants of unsupported parents. CLI:
`go run ./cmd/loadcards -inventory -json`. The strict loader still determines
CLI success, so inventory cannot make a rejected collection acceptable.

Dependencies: existing `Parse`, `Compile`, `Node`, `Position`, `Issue`,
`definitionKey`, and `LoadFS`. Engine reuse is unchanged: existing factory,
action, target, damage, zone, and registry services execute accepted cards.
No new engine rules, runtime registry, Go-card fallback, or DSL deletion is part
of this slice. CR 608.2b/c/n describes the spell regression's target, resolution,
and final graveyard handling; the slice does not change these rules.

Schema, exact role table, shape/context rules, paths, and API limits:
[mtgish-ron-inventory.md](mtgish-ron-inventory.md), implemented in
`pkg/mage/dsl/ron/inventory.go`. Shapes group immediate argument kinds/names
and sorted fields. Context keeps constructor/argument paths and removes only
list indices. Known roles come from parent contracts; descendants of unknown
contracts remain explicitly unknown. All expression support is unverified.
Engine-validation and behavior-verification counts are null, not zero.

Supported reporting contexts are all symbols, literals, lists, tuples, and
structs, in every parsed card and nested expression. Role adapters cover:

| Parent role and exact shape | Argument roles |
| --- | --- |
| card `Card` named fields | `Rules`: rule; other fields: metadata |
| metadata | metadata descendants |
| rule `SpellActions/1`, `Activated/2`, `TriggerA/2` | actions; cost/actions; event/actions |
| rule `EnchantPermanent/1`, `EachPermanentLayerEffect/2`, `PermanentLayerEffect/2` | permanent filter; filter/static modification; permanent reference/static modification |
| actions `Targeted/2`, `ActionList/1` | target/actions; action |
| target `TargetPermanent/1`, `TargetPlayer/1`, `TargetSpell/1` | permanent/player/spell filter |
| cost `And/1`, `TapPermanent/1`, `SacrificePermanent/1`, `PayMana/1` | cost; permanent reference; permanent reference; metadata |
| action single-permanent tap/untap/destroy/destroy-no-regen/regenerate/sacrifice `/1` | permanent reference |
| action `PlayerAction/2`, `SpellDealsDamage/3`, `PermanentDealsDamage/3` | player reference/player action; spell reference/value/recipient; permanent reference/value/recipient |
| action or player action draw-number/gain-life/lose-life `/1` | value |
| permanent filter `And/1`, `Or/1`, `SinglePermanent/1` | permanent filter; permanent filter; permanent reference |
| event enter/die/upkeep `/1` | permanent filter; permanent filter; player filter |
| player filter `SinglePlayer/1` | player reference |
| list, any role | same element role |
| other forms | unknown descendants, fully walked |

Unknown-field validation in `compiler.go` now selects fields in sorted order.
A repeated invalid-declaration test first failed with `UnknownZ` instead of
`UnknownA`; the sorting repair makes the diagnostic deterministic.

### Fresh measurements and source findings

All 12 source files were parsed and walked in full: 2,445 records, 986 unique
card names, 72,938 AST node occurrences, and 8,528 context/shape signatures.
23,096 node occurrences have unknown roles. This larger signature count includes
literals, list envelopes, and context; it is not comparable to the snapshot's
600 tuple signatures. Reporting coverage now includes every parsed node.

Strict loader before and after: **836 compiled records, 1,609 rejected records**.
There are 243 unique compiled names and 743 rejected names in this collection.
Compilation gain: zero. Newly supported gameplay cards: zero. The reporting
slice affects all 986 unique cards without changing their RON records.

First diagnostics remain compiler-adapter/scope gaps, including:

| First diagnostic | Records |
| --- | ---: |
| unsupported rule `PermanentRuleEffect` | 104 |
| unsupported rule `AsPermanentEnters` | 90 |
| unsupported rule `ActivatedWithModifiers` | 61 |
| unsupported non-spell rule `CastEffect` | 53 |
| unsupported rule `If` | 46 |
| unsupported action `ChooseADamageSource` | 46 |
| unsupported permanent filter `ControlledByAPlayer` | 43 |
| unsupported action `MayCost` | 43 |

These counts are first diagnostics, not full semantic support measurements.
The complete nested report can be regenerated with the CLI; it is not stored as
another large snapshot. All source hashes match the vocabulary snapshot. No
source defects were corrected. The design's Gaea's Liege/Fastbond observations
remain unverified source concerns and must be audited in their behavior slices.
No engine contract or environment failure blocked inventory implementation.

### Acceptance and card-to-test evidence

Tests were written before implementation. The first API/CLI tests failed because
`InspectFS`/`run` were absent. The diagnostic stability test then failed on the
unsorted field selection. Focused tests pass after the implementation.

| Test | Evidence |
| --- | --- |
| `TestInventoryWalksRejectedDescendantsAndSeparatesRoles` | Nested gaps after the first error, exact path/position, cost/action roles, reprint distinct names, deterministic JSON, strict rejection |
| `TestInventoryKeepsCompilationSeparateFromSupport` | Conflicting reprints and no syntax-based support claim |
| `TestInventoryReportsMalformedAndEmptyFiles` | Malformed/empty files, bad pattern, missing collection |
| `TestInventoryConflictKeepsFirstReprintPosition` | Conflict points at the first of several identical reprints |
| `TestInventoryInvalidFieldsHaveStableDiagnostics` | Repeated invalid declarations produce a stable first error |
| `TestInventoryRepositoryCollection` | Full collection counts and real nested permanent references |
| `TestInventoryCLIUsesStrictExitStatus` | Accepted and rejected collections, default JSON loader compatibility, unmeasured execution totals |
| `TestInventoryLeavesRealRONGameplayAndRegistryUnchanged` | LEB Lightning Bolt remains executable after whole-collection inspection; opponent loses 3 life, spell goes to graveyard; process registry unchanged |

Existing RON gameplay tests were rerun. Their observed mappings are:

| Unique cards | Existing test |
| --- | --- |
| Forest, Tropical Island | `TestRONBasicLandTypesSupplyMana` |
| Lightning Bolt, Air Elemental | `TestRONDamageUsesPreventionAndTargetLegality` |
| Ancestral Recall, Forest | `TestRONAncestralRecallDrawsForTargetPlayer` |
| Drudge Skeletons, Death Ward | `TestRONRegenerationUsesItsDeclaredSubject` |
| Black Lotus | `TestRONActivatedManaPaysTapAndSacrificeCosts` |
| White Knight, Crusade, Holy Strength | `TestRONStaticLayersTrackAuraAndGlobalBoosts` |
| Crimson Kobolds | `TestRONColorIndicatorOverridesManaCostColors` |

These tests touch 12 unique real cards; they are partial behavior evidence, not
proof of every rule and interaction on those cards. Source-entry trigger testing
uses a synthetic RON card. Factory-isolation and invalid-binding tests remain
in `loader_test.go`. Inventory introduces no bindings, costs, replacements,
simultaneous operations, or phasing behavior; those contracts remain future work.

### Validation

Baseline focused tests, `go test ./pkg/mage/... -short -count=1`, and
`make build` passed. Baseline loader returned exit 1 for unsupported collection.

| Final command | Result |
| --- | --- |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS, including new inventory tests and retained RON gameplay |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS in the supplied working tree |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues; only new test imports were reordered |
| `make wasm` | PASS; generated outputs were restored to their prior state; no runtime startup claim |
| `git diff --check` | PASS |
| `go run ./cmd/loadcards -json` | Expected exit 1: 836/2,445 compiled, 1,609 rejected |
| `go run ./cmd/loadcards -inventory -json` | Expected exit 1; full deterministic report, same diagnostics as strict loader |

Two whole-collection inventory runs produced byte-identical JSON. File hashes
match the existing vocabulary snapshot, so no snapshot refresh is needed.
Inspection's issue array equals the fresh strict loader issue array exactly.
The report does not establish engine-validation or full gameplay totals.

Commit readiness is blocked by pre-existing untracked prerequisites. An isolated
export of `HEAD` with only the seven files touched by this slice fails
`make build` (exit 2): `Issue`, `Position`, `Actions`, and `Node` are undefined.
Their implementations are in the pre-existing untracked RON loader, parser,
and adapters. The isolated test did not change the working tree. Committing
those prerequisites and their assets/engine repairs would include work outside
this slice. No commit was made, no files were staged, and the user's staged
skill deletions were left intact. Workspace validation passed, but a standalone
slice commit is not build-ready until the prerequisite migration work is
committed. This is a baseline dependency failure, not a collection rejection.

Files changed by this slice:
`pkg/mage/dsl/ron/inventory.go`, `inventory_test.go`, `compiler.go` (sorted
unknown-field checks only); `cmd/loadcards/main.go`, `main_test.go`;
`docs/designs/mtgish-ron-inventory.md`, `mtgish-ron-progress.md`.

### Next slice and removal gates

Next: P0-SCOPE, typed reference plans with fresh per-factory bindings. Start with
existing single-target contexts; prove declaration rejection and factory/copy
isolation before adding explicit player/permanent wrappers and player-scoped
life actions. The flat `scope` and effect factory's single permanent binding are
still prerequisites to replace. Use the nested inventory to find all uses and
inspect complete records. Modes, multiple target groups, optional costs, event
bindings, and saved results are not supported by this reporting work.

Production still uses Go definitions. Complete RON loading, behavior coverage,
startup integration, native/WASM smoke tests, Go-card deletion, and whole-repo DSL
caller/removal audits remain open. The removal gates are not met. Authoring
instruction cutover, obsolete generator removal, and DSL removal are deferred to
the design's gated phases. Migration remains incomplete.

## Run 2026-10-03: P0-SCOPE-PLAYER

Status: complete single-player target/read prerequisite slice. P0-SCOPE remains
partial. This run stops here; it does not start player-scoped life actions.

### Contract, dependencies, and reuse

Exact supported signatures/contexts are recorded in
[mtgish-ron-player-scopes.md](mtgish-ron-player-scopes.md).
`TargetPlayer(AnyPlayer)/1` inside a single-element `Targeted/2` declaration
produces a typed player declaration. `PlayerAction/2` accepts symbol `You` or
symbol `Ref_TargetPlayer`; `DrawNumberCards/1` accepts the existing `Integer/1`
or announced spell `ValueX` value forms. Direct draws retain the resolving
controller. The existing spell, activation, and supported trigger action builders
share this lowering. Nested/multiple declarations remain errors.

`playerTargetPlan` gives a player declaration an immutable lexical identity.
`playerReferencePlan` reads that identity or the resolving controller.
`instanceBindings` maps the identity to a fresh engine player target for each
factory/action build. Effect factories now receive this instance environment
instead of only a permanent binding. Permanent binding behavior is retained.
Player draws use `mage.DrawFor`, `mage.ValueAtResolution`, and
`mage.DeclarePlayerTarget`; `mage.WithTargets` normalizes typed declarations.
Engine definition validation now sees the target read and can reject a missing
declaration. No chosen state, registered Go factory, or card-name branch is
stored in the compiler plan.

Dependencies: existing parser, compiler action plans, typed player targets,
resolving contexts, draw actions, definition validation, stack copy/retargeting,
game clones, and transactional casting. All required engine contracts exist;
no engine implementation changes are needed. RON owns the player spelling and
scope rejection. Relevant existing rules are CR 113.7a and 608.2b/c/n.
No permanent LKI query, phasing operation, replacement ordering, or simultaneous
operation is introduced; the retained engine services still own those rules.

### Complete-record and nested-form audit

The fresh AST inventory was inspected for all nested `TargetPlayer` signatures,
including forms under unsupported parents. Complete distinct RON records were
read for these 36 unique cards:
Alabaster Potion, Amnesia, Ancestral Recall, Braingeyser, Disrupting Scepter,
Drafna's Restoration, Drain Power, Dwarven Catapult, Fire and Brimstone,
Glasses of Urza, Gwendlyn Di Corci, Healing Salve, Hurkyl's Recall,
Hymn to Tourach, Inquisition, Jovial Evil, Juxtapose, Mana Clash, Mana Short,
Millstone, Mind Twist, Mirror Universe, Natural Selection, Nebuchadnezzar,
Orcish Spy, Petra Sphinx, Rag Man, Relic Bind, Storm Seeker, Stream of Life,
Tempest Efreet, Tormod's Crypt, Visions, Wand of Ith, Word of Command, Xira Arien.

Only Ancestral Recall and Braingeyser are currently accepted records affected
by player draw lowering. The other 34 still have compiler prerequisites such as
modes, filters, results, discard/mill, optional payment, or timing modifiers.
They are not gameplay claims. Complete accepted `DrawNumberCards` records were
also checked; there are no other accepted unique cards using that action.
The trigger-context acceptance test also found a missing adapter prerequisite:
constructors retained a nil effect before appending the actions. Engine definition
validation rejected `Draw Entry: abilities[0].effects[0]: supply a non-nil expression`.
`compileTriggerRule` now passes the complete action sequence to the constructor
and adds only its targets afterward. Complete accepted trigger records were read
for Onulet and Su-Chi; both use source-death triggers and keep their original
resolution effects. This repair is part of the same instance-lowering slice.
No source-data defect was corrected. All source hashes remain unchanged, so the
vocabulary snapshot does not need a source refresh. No environment failure
prevented workspace checks.

### TDD and card-to-test mappings

The new typed-reference test failed before implementation:
`Ancestral Recall: expected one typed player target read, got 0`.
The initial invalid-scope, copy/clone, and X/rollback tests were also written and
run before implementation. After lowering changes, focused tests pass.

| Unique card or context | Test and evidence |
| --- | --- |
| Ancestral Recall, Braingeyser | `TestRONPlayerReadsUseDeclaredTypedTargets`: typed read metadata and engine definition validation |
| Ancestral Recall | `TestRONPlayerTargetInstancesCopiesAndClones`: concurrent distinct instances, copy retargets from opponent to caster without changing original, independent clone resolution |
| Braingeyser | `TestRONBraingeyserAnnouncementAndRollback`: announced X=4, correct paying/target player, insufficient mana rollback, invalid target rollback, draw and final graveyard state |
| Ancestral Recall | Retained `TestRONAncestralRecallDrawsForTargetPlayer` |
| Synthetic declarations | `TestRONPlayerScopeRejectsInvalidReads`: undeclared, wrong type, wrong syntax, and reads from another rule |
| Synthetic factory and spell | `TestRONPlayerFactoryBindsReadsToEachDeclaration`: fresh declarations from one compiled factory, engine rejects missing declaration, sibling `You` scope remains independent |
| Synthetic activation | `TestRONPlayerTargetActivationSurvivesSourceDeparture`: tap payment, no announcement-time draw, source departure, clone isolation, stored player target on resolution |
| Synthetic trigger | `TestRONPlayerTargetTriggerUsesPlacementBinding`: valid definition, chosen player distinct from source controller, draw at resolution |
| Onulet, Su-Chi | `TestRONScopeTriggerRepairKeepsRealDeathActionsValid`: valid engine definitions, source death, life/mana actions occur at resolution |

Gameplay evidence expands from 12 to 15 touched real unique cards by adding
Braingeyser, Onulet, and Su-Chi. Ancestral Recall has stronger copy/clone evidence. This is partial
behavior evidence, not the full per-card completion audit. Compilation gain is
zero, and no global engine-validation/behavior-verification total is claimed.

### Fresh measurements and validation

Baseline focused tests and `make build` passed. Baseline loader exited 1 for
collection rejection. Final strict loader still reports **836 compiled records
of 2,445**, **1,609 rejected records**, **243 compiled unique names**, and
**743 rejected unique names**. No previously accepted record became rejected.

Two diagnostics changed for Crumble (`3ED:50`, `ATQ:23`):
`ControllerOfPermanent` now reports `unbound or incompatible player reference`
instead of `unbound or incompatible target reference`. This is a missing player
query adapter, not a source correction or newly accepted shape. The leading
remaining diagnostics are unchanged: `PermanentRuleEffect` 104 records,
`AsPermanentEnters` 90, `ActivatedWithModifiers` 61, non-spell `CastEffect` 53,
rule `If` 46, `ChooseADamageSource` 46, `ControlledByAPlayer` 43, `MayCost` 43.
These are first-error counts; the full AST inventory remains necessary.

| Command | Result |
| --- | --- |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS in supplied workspace |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues; formatter reordered new test imports |
| `git diff --check` | PASS |
| `go run ./cmd/loadcards -json` | Expected exit 1; 836 compiled, 1,609 rejected |
| `go run ./cmd/loadcards -inventory -json` | Expected exit 1; full nested baseline inspection |

No assets, loading platform, startup wiring, or WASM code changed. `make wasm`
is therefore not required for this slice, and no new WASM validation is claimed.

Commit blocker: a fresh isolated export of `HEAD` plus the four touched compiler
files fails `make build` with exit 2: `Actions` and `Node` are undefined. The
pre-existing parser/adapters, CLI, sources, and earlier compiler work are still
untracked prerequisites. A standalone commit cannot build without including
that unrelated work. The supplied workspace checks pass; collection rejection
is expected and is separate from this isolated dependency failure. No commit
was made and no files were staged. Existing staged skill deletions and all
unrelated changes remain in place.

Files changed by this slice: `pkg/mage/dsl/ron/compiler.go`,
`pkg/mage/dsl/ron/triggers.go`, new
`pkg/mage/dsl/ron/player_scope.go`, new `pkg/mage/dsl/ron/player_scope_test.go`,
new `docs/designs/mtgish-ron-player-scopes.md`, and this progress ledger.

### Next slice and removal gates

Next: P0-SCOPE-PERMANENT, immutable typed plans for permanent/source/host reads
and per-instance lowering of the remaining single-target actions. Preserve
incarnation identity and test source departure, LKI where required, phasing,
copies, and declaration validation before wrappers or multiple groups. Existing
permanent target declarations are fresh, but several actions still use legacy
first-target selectors. Spell and union target reads also remain to migrate.
Player-scoped life actions remain rejected pending their next dependency-ready
slice. Modes, numbered groups, saved results, event environments, and payment
bindings remain open.

Go-card deletion, runtime cutover, generator retirement, and DSL removal gates
remain unmet. No DSL deletion was attempted, so no unused-function removal
claim is made. Full RON gameplay and native/WASM startup completion remain open.

## Run 2026-10-03: P0-SCOPE-PERMANENT-UNARY

Status: complete unary-action prerequisite slice. P0-SCOPE remains partial.
This run stops here; temporary layer selectors and damage references remain open.

### Contract, dependencies, and reuse

Exact signatures, supported contexts, complete-record audit, and reference
lifetimes are in [mtgish-ron-permanent-scopes.md](mtgish-ron-permanent-scopes.md).
The six unary actions `DestroyPermanent/1`, `DestroyPermanentNoRegen/1`,
`TapPermanent/1`, `UntapPermanent/1`, `RegeneratePermanent/1`, and
`SacrificePermanent/1` now lower typed permanent references in the existing
spell, activation, and trigger action-list contexts. Symbols are
`Ref_TargetPermanent` in a single permanent declaration, `ThisPermanent` for a
permanent source, and `HostPermanent` for an Aura source. Source/host/target
reads have separate typed plans; permanent target declarations have immutable
lexical identities and fresh per-instance engine bindings. Invalid syntax,
types, missing declarations, cross-rule reads, and unsupported event references
remain compilation errors. Engine definition validation now sees all six reads.

Dependency: P0-SCOPE-PLAYER's per-instance effect environment, existing target
normalization, parser, action-list compiler, and engine typed permanent refs.
Reuse: `DeclarePermanentTarget`, `This`, `AttachedPermanent`, `Destroy`,
`DestroyWithoutRegeneration`, `TapSubject`, `UntapPermanent`, `Regenerate`,
`Sacrifice`, definition validation, stack target/copy services, game clones,
retained source attachments, regeneration/replacement execution, and
transactional activation costs. All needed engine contracts exist; no engine
implementation change is needed. RON owns spelling and scope checks. CR 113.7a,
400.7, 608.2b/c/h/n, and 702.26b/d/g supply identity, timing, LKI, legality,
and phasing contracts. No saved characteristic reader or simultaneous batch is
introduced, and replacement ordering remains the shared engine's contract.

### Nested forms, affected cards, and source findings

Fresh whole-collection AST inspection covers descendants after first errors.
All nested occurrences of the six names were inspected, including separate cost
roles and unknown contexts. Complete distinct RON records were read for the 92
cards with direct action occurrences. The companion document lists every card,
including the 43 accepted unique cards affected by this lowering. Other cards
still need surrounding rules, events, filters, costs, or later actions; these
records are not accepted by partial factories.

Newly accepted cards: **Regeneration** (six identical reprints), **The Brute**
(one), and **Thrull Retainer** (one). The first two use mana-paid host
regeneration. Thrull Retainer sacrifices its Aura as a cost and regenerates its
last host at resolution. Its +1/+1 static layer still ends when the Aura leaves.
The engine's retained attachment service supplies this exact behavior.

No data defect was corrected, and no RON asset changed. Fresh inventory file
hashes match all 12 vocabulary snapshot hashes, so no source snapshot refresh
is needed. The Gaea's Liege/Fastbond concerns remain outside this slice.
There was no environment failure. Missing adapters were resolved in `ron`;
no missing engine service was found.

### TDD and card-to-test evidence

The initial test file needed a type assertion before it could run. Before any
implementation change, executable tests then failed as expected: destroy,
destroy-no-regeneration, tap, and sacrifice exposed zero typed permanent reads;
all three host-regeneration cards rejected `HostPermanent`. Existing untap and
regeneration target reads already passed. The copy test was corrected to offer
one legal new target, rather than assume the base player's choice order.
After implementation, two static abilities required selecting activation index
1 in the real Aura tests. These were test setup corrections, not engine gaps.

| Card or context | Test and evidence |
| --- | --- |
| Six unary target actions | `TestRONPermanentActionsDeclareTypedReads`: fresh factory targets, exact typed read metadata, valid definitions, missing declaration rejected by engine validation |
| Invalid declarations | `TestRONPermanentScopeRejectsInvalidReads`: undeclared, wrong type/syntax, non-Aura host, instant permanent source, cross-rule and event reads |
| Regeneration, The Brute, Thrull Retainer, Air Elemental | `TestRONHostRegenerationAndSacrificeLKI`: full RON definitions; opposing host; static P/T; mana failure rollback; sacrifice paid before resolution; retained attachment; clone isolation; regeneration shield consumed by destruction |
| Regeneration, Air Elemental | `TestRONHostReferenceReadsCurrentAttachmentAndRejectsNewIncarnation`: attachment read at resolution; departed original source cannot follow returned source; returned/phased host absent |
| Ice Storm, Forest | `TestRONPermanentTargetCopiesClonesAndLegality`: copy retargeting, original target leave-and-return, independent cloned resolution, source/target zone handling |
| Icy Manipulator, Air Elemental | `TestRONPermanentActionsPhasingAndSourceDeparture`: paid mana/tap costs, effect only at resolution, departed source, phased target illegal, phase-in retains identity |
| Six unary target actions | `TestRONUnaryPermanentActionsUseSharedRules`: normal destruction consumes regeneration, no-regeneration destruction removes shielded subject, tap/untap, regeneration, controller restriction on sacrifice |
| Synthetic source and target | `TestRONSourceAndTargetReadsStayDistinct`: one action body uses source and target separately; returned source stays untouched; target still affected |
| Shatter, Fissure, Icy Manipulator, Air Elemental, Ley Druid, Forest | `TestRONRealPermanentRemovalAndUntap`: real RON ordinary/no-regeneration destruction, owner graveyards, target untap distinct from tap payment |
| Synthetic trigger and Aura | `TestRONPermanentTriggerBindingAndHostEntryTiming`: typed trigger declaration, choice at placement, effect at resolution after source departure; Aura attachment exists before host entry trigger resolves |

Existing RON gameplay tests remain passing. Touched real unique-card gameplay
coverage rises from 15 to **23** by adding Regeneration, The Brute, Thrull
Retainer, Ice Storm, Icy Manipulator, Shatter, Fissure, and Ley Druid. This is
partial behavior evidence, not the per-card full completion audit. The unary
contracts use synthetic tests for forms whose real cards still have surrounding
gaps. Compilation totals and gameplay coverage remain separate.

### Fresh measurements and validation

Baseline focused RON/CLI tests and `make build` passed. Baseline strict loader
returned expected exit 1: 836/2,445 compiled, 1,609 rejected. Final loader returns
expected exit 1: **844/2,445 compiled**, **1,601 rejected**, **246 compiled unique
names**, **740 rejected unique names**. Gain: eight records and three unique
cards. No accepted record became rejected. Strict loading still publishes no
partial collection.

Eight first diagnostics advanced without accepting those records: six Paralyze
reprints now fail at `PermanentRuleEffect`, Tangle Kelp at rule `If`, and
Venarian Gold at `PutCounters`. Leading remaining first diagnostics:
`PermanentRuleEffect` 110 records, `AsPermanentEnters` 90,
`ActivatedWithModifiers` 61, non-spell `CastEffect` 53, rule `If` 47,
`ChooseADamageSource` 46, `ControlledByAPlayer` 43, and `MayCost` 43.
They are compiler/context gaps, not collection build failures or proven source
defects. The full nested inventory remains necessary to see subsequent gaps.

| Command | Result |
| --- | --- |
| Focused unary/host tests with `-count=1` | Expected pre-implementation failures; PASS after implementation |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS; rerun after final test and lint fixes |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS in supplied workspace |
| `go vet ./...` | PASS |
| `make lint` | First run found a test variable named `copy`; renamed it. Final run PASS, zero issues. Formatter reordered new test imports only |
| `git diff --check` | PASS |
| `go run ./cmd/loadcards -json` | Expected exit 1, 844 compiled, 1,601 rejected |
| `go run ./cmd/loadcards -inventory -json` | Expected exit 1, same strict diagnostics; full nested walk |

No assets, platform loading, startup wiring, or WASM code changed; this slice
does not require `make wasm` and makes no new WASM validation claim.

The standalone commit remains blocked by untracked baseline prerequisites.
A fresh isolated `git archive HEAD` plus the five touched Go files fails
`make build` with exit 2: `Actions` and `Node` are undefined. The parser,
adapters, prior compiler work, and assets remain untracked. Including them
would commit work outside this slice. The supplied workspace checks pass;
expected collection rejection is separate from this isolated dependency
failure. No commit was made, and nothing was staged. User changes, unrelated
untracked files, and existing staged skill deletions remain intact.

Files changed: `pkg/mage/dsl/ron/compiler.go`, `player_scope.go`,
`inventory_test.go` (exact measured collection counts), new `permanent_scope.go`,
new `permanent_scope_test.go`, new
`docs/designs/mtgish-ron-permanent-scopes.md`, and this ledger. No DSL function
was removed, so no whole-repository removal audit or cleanup claim is made.

### Next slice and removal gates

Next dependency-ready prerequisite: P0-SCOPE-PERMANENT-LAYERS. Replace
`CreatePermanentLayerEffectUntil/3` source/host/target legacy selectors with typed
reference lowering for the currently supported end-of-turn fixed P/T and keyword
forms. Prove incarnation capture, cleanup, source departure, host timing, and
copy/clone isolation before extending durations or modification shapes. Damage
sources/recipients, spell and union references, numbered groups, wrappers,
player life actions, saved results, events, and payment bindings remain open.

Runtime cutover, complete RON gameplay, native/WASM startup, Go-card deletion,
obsolete generator retirement, and whole-repository DSL removal gates remain
unmet. No further slice was started in this run.

## Run 2026-10-03: P0-SCOPE-SPELL-FILTERS

Status: complete single-spell-target filter and counter slice. No second slice
was started. The initial working tree had user changes to `.gitignore` and
`docs/designs/mtgish-ron-execution.md`; these are preserved and excluded from
this slice. Prior compiler prerequisites are now tracked, so the previous
ledger's untracked-dependency commit blocker no longer applies.

The fresh three-candidate comparison, exact signatures/contexts, dependencies,
engine reuse, complete affected-name lists, card-to-test mapping, source findings
and ranked next candidates are in
[mtgish-ron-spell-scopes.md](mtgish-ron-spell-scopes.md).

Selection: single typed spell targets with type/color and Boolean filters unlock
five unique cards for a small adapter change. Layer reference lowering has zero
immediate unlocks; explicit-player life actions unlock one card. The selected
prerequisite also enables later controller/value readers and counter modes.

Implemented `TargetSpell/1` with `AnySpell`, `IsCardtype/1`, `IsColor/1` and
nonempty recursive `And/1`/`Or/1`, plus declaration-local `CounterSpell/1` in
spell, activation and supported-trigger action lists. The compiler builds fresh
`DeclareSpellTarget` bindings and lowers counter reads to `mage.Counter`.
Engine services handle legality, identity, copies/clones, cost rollback,
source departure, uncounterability and owner graveyards. No engine rule, asset,
startup wiring, Go definition, generator or DSL export changed.

TDD: pre-implementation tests failed on the untyped AnySpell declaration and
unsupported filters in all five selected real cards. Focused tests now pass.
Additional tests cover invalid declarations, nested Boolean filters, abilities
excluded from targets, query purity, clone/copy frames, changed-color legality,
removed/recast spells, departed/phased sources, payment failure and queued entry
triggers. Full mapping is in the companion document.

Fresh strict loading: **859/2,445 compiled records**, **1,586 rejected records**,
**251 compiled unique names**, **735 rejected unique names**. Gain equals the
estimate: **15 records, five unique cards** (Artifact Blast, Deathgrip, Flash
Counter, Lifeforce, Remove Soul). No accepted record regressed. Counterspell also
has a typed-read gameplay regression. Six counter cards gain direct
gameplay tests; four additional real cards serve as spell fixtures. No global
behavior-verification total or full-collection certification is claimed.
RON source hashes match the vocabulary snapshot. No source defect was corrected.

Leading remaining first diagnostics: `PermanentRuleEffect` 110 records,
`AsPermanentEnters` 90, `ActivatedWithModifiers` 61, non-spell `CastEffect` 53,
rule `If` 47, `ChooseADamageSource` 46, `ControlledByAPlayer` 43, `MayCost` 43,
and `WhenAPlayerCastsASpell` 43. Seven unique cards advance their first errors
without acceptance; the companion table lists every changed diagnostic. These
are missing compiler contexts/adapters, not build failures or proven data defects.

Inventory now reports `CounterSpell/1` operands as spell references and preserves
spell-filter roles through nested `And`/`Or`. Support remains unverified in the
inventory; role reporting does not claim gameplay support.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, `make build` | PASS |
| Baseline `go run ./cmd/loadcards -json` | Expected exit 1; 844 compiled, 1,601 rejected |
| Focused new tests with `-count=1` | Expected failures before implementation; PASS after implementation |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS; final run after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS; repeated after final test changes and lint |
| `make test` | PASS; final run after lint |
| `make build` | PASS; final run after lint |
| `go vet ./...` | PASS; final run after lint |
| `make lint` | PASS, zero issues; formatter changed new test import order only; final rerun PASS |
| `git diff --check` | PASS |
| `go run ./cmd/loadcards -json` | Expected exit 1; 859 compiled, 1,586 rejected; no partial registry |
| `go run ./cmd/loadcards -inventory -json` | Expected exit 1; same compilation counts and nested diagnostics |
| Vocabulary source hash comparison | All 12 hashes match |

Files: `compiler.go`, `player_scope.go`, `inventory.go`, `inventory_test.go`,
new `spell_scope.go` and `spell_scope_test.go` under `pkg/mage/dsl/ron`,
`docs/dsl-reference.md`, new `docs/designs/mtgish-ron-spell-scopes.md`, and this
ledger. No automatic lint edit changed unrelated files. All prerequisites are
tracked and required checks pass, so these nine files form one standalone commit.
The two initial user-modified files remain outside the commit.

Next ranked slice: typed temporary-layer reference lowering; then permanent
controller filters/references; then optional payment result scopes. No next slice
was started. Full RON gameplay, runtime loading, Go-card deletion, obsolete
Go-set generation retirement, and unused-DSL removal gates remain unmet.

## Run 2026-10-03: P0-SCOPE-PERMANENT-LAYERS

Status: complete typed fixed temporary-layer slice. No second slice was started.
Initial user changes to `.gitignore` and `mtgish-ron-execution.md` remain outside
this slice. No unrelated untracked files existed at the start.

The fresh three-candidate comparison, complete affected-name lists, remaining
card diagnostics, exact signatures/contexts, dependencies, engine reuse,
card-to-test mapping and ranked next candidates are in
[mtgish-ron-temporary-layers.md](mtgish-ron-temporary-layers.md).

Selection: shared typed temporary-layer subjects for 35 accepted unique cards,
with zero immediate full-card unlocks. Static controller filters have 15 affected
names and four estimated unlocks; optional counter payments have four affected
names and two estimated core unlocks. They need broader filter/payer/result
contexts. This smaller prerequisite enables later dynamic boosts (Howl from
Beyond, Great Defender), keyword removal (Radjan Spirit), switching
(Transmutation) and subject identity for all 35 accepted cards. The phase order
and previous ledger suggestion were not used as fixed priorities.

Implemented `CreatePermanentLayerEffectUntil/3` with symbol source/host/local
single target, fixed integer `AdjustPT/2`, basic keyword `AddAbility/1`, and
symbol `UntilEndOfTurn` in existing spell, activation and supported trigger
contexts. RON uses immutable permanentReferencePlan and per-instance target
bindings. One `mage.Until` installs each layer action's modifications together,
using `ModifyPT`, `Fixed` and `GrantKeywordModification`. No engine contract or
service was missing. Empty modifications and invalid references are rejected
at compilation. Engine definition validation now sees the typed subject reads.

Reuse includes retained original-source attachment, captured incarnations,
stack target/copy frames, clone isolation, shared layers, end-of-turn expiry
and transactional activation costs. CR 113.7a, 400.7, 608.2b/c/h, 611.2a/c,
613.1f/4c and 702.26b/d/e/f supply the contract. Replacement ordering,
characteristic LKI readers, plural operation batches and payment-result scopes
introduce no new behavior in this slice. No engine implementation, RON asset,
startup wiring, Go definition, generator or DSL export was changed.

TDD: executable tests first failed on absent typed layer subject metadata.
Later inventory-role tests failed before the inventory adapter was added.
Test setup corrections used ARN for Flying Carpet, read live objects after
transactional payment replaced state, and checked pending-trigger effect timing
instead of assuming immediate stack placement. Final tests cover declaration
validation, fresh factories, one installation for mixed modifications,
announcement/resolution timing, retargeted copies, clones, returned sources and
targets before/after installation, source departure, host LKI/reattachment,
phasing before/after resolution, rollback, sacrifice costs and expiry.

Compilation remains **859/2,445 records**, with **1,586 rejected records**,
**251 compiled unique names** and **735 rejected unique names**. Estimated
unlock gain zero equals actual gain zero. No accepted record regressed and no
first diagnostic changed. Strict loading publishes no partial registry.
Nested inventory now reports temporary modifications, typed subjects, values,
keywords and durations separately; support remains unverified.

Six real cards have additional direct temporary-layer gameplay evidence:
Giant Growth, Jump, Elven Lyre, Flying Carpet, Shivan Dragon and Blessing.
Llanowar Elves is their real target fixture. This is behavior coverage for
accepted cards, separate from compilation progress. No full-collection behavior
verification total or all-rules certification for the 35 names is claimed.

Leading remaining first gaps: PermanentRuleEffect 110 records, AsPermanentEnters
90, ActivatedWithModifiers 61, instant/sorcery CastEffect 53, rule If 47,
ChooseADamageSource 46, ControlledByAPlayer 43, MayCost 43,
WhenAPlayerCastsASpell 43. These are compiler/context gaps, not build failures.
No source defect was corrected. All 12 source hashes match the vocabulary
snapshot; no snapshot refresh is needed. Gaea's Liege/Fastbond concerns remain
open. No environment failure or pre-existing check failure blocked this run.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, `make build` | PASS |
| Baseline `go run ./cmd/loadcards -json` | Expected exit 1; 859 compiled, 1,586 rejected |
| Baseline `go run ./cmd/loadcards -inventory -json` | Expected exit 1; complete nested AST inventory |
| New focused temporary-layer tests with `-count=1` | Expected failures before adapters; PASS afterward |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS |
| `go test ./pkg/mage/... -short -count=1` | PASS; repeated after final changes and lint |
| `make test` | PASS; repeated after final changes and lint |
| `make build` | PASS; repeated after lint |
| `go vet ./...` | PASS; repeated after lint |
| `make lint` | First run found a redundant test assignment; fixed it. Final PASS, zero issues |
| `git diff --check` | PASS |
| Final `go run ./cmd/loadcards -json` | Expected exit 1; 859 compiled, 1,586 rejected |
| Final `go run ./cmd/loadcards -inventory -json` | Expected exit 1; same counts; updated nested roles |
| Vocabulary hash comparison | All 12 source hashes match |

Automatic lint changes only reordered the new test imports. No unrelated file
changed during lint. Final required checks passed after the assignment fix.
This slice changes `compiler.go`, `inventory.go`, new `temporary_layer_test.go`,
`docs/dsl-reference.md`, the new temporary-layer contract document and this ledger.
All dependencies are tracked; these six files form one standalone commit.

Next ranked slice: source-aware static controller filters (four concrete card
unlocks), then typed variable/numeric temporary modifications, then optional
fixed mana-payment result scopes. No further slice was started. Complete RON
loading/gameplay, embedded startup, native/WASM startup evidence, Go-card
deletion, obsolete Go-set generator retirement and unused-DSL removal gates
remain unmet. No DSL removal or whole-repository caller audit is claimed.

## Run 2026-10-03: P1-STATIC-CONTROLLER

Status: complete controlled static creature-layer slice. No second slice was
started. Initial user changes to `.gitignore` and `mtgish-ron-execution.md` are
preserved and excluded from the commit. No unrelated untracked files existed.

The fresh three-candidate comparison, exact shapes/contexts, dependencies,
complete affected-name lists, source findings, test mappings and ranked next
candidates are in [mtgish-ron-static-controller.md](mtgish-ron-static-controller.md).
Selection: static controller conjunctions affect 15 unique names and unlock five
for a small adapter. Variable temporary boosts affect six names and unlock two;
optional fixed counter payments affect four names and unlock two with broader
payer/decision/result prerequisites. Fresh inspection found Orcish Oriflamme as
a fifth static unlock beyond the prior ledger's four-card estimate.

Implemented `ControlledByAPlayer(SinglePlayer(You))/1` in creature-restricted
`EachPermanentLayerEffect/2`, below nonempty nested `And/1`, for existing fixed
`AdjustPT/2` and basic `AddAbility/1` modifications. RON validates and decomposes
immutable static filters; each factory builds fresh continuous effects. Engine
reuse: BoostControlledCreatures, GrantKeywordToControlled, shared filters,
printed static registration, layer ordering, phasing and clone ownership.
Other player references, controller predicates below Or and other compiler
contexts remain rejected. Inventory now assigns player selector/reference roles
below controller predicates without claiming support.

TDD identified two missing engine contracts after the initial compiler failures.
Control duration now survives target phasing, while cleanup and unmet conditions
still expire it. Explicit targeted/global ability-removal services now expose
removal membership to printed static dependency ordering; application checks
current activity after ordering. Targeted removal expiry state is deep-cloned.
These changes are prerequisites for complete source-control and keyword-grant
lifecycle behavior in the selected shape, not a second capability slice.
Rules: CR 611.2b/3a/3b, 613.1b/e/f/g, 613.4c, 613.8a/b, 702.26b/d/f.
API/rules documentation records only those established contracts. No new public
engine API, cost, LKI value, target binding, replacement or simultaneous zone
operation was added. Existing regression tests retain those services' evidence.

Five cards gain direct RON gameplay evidence: Castle, Fortified Area, Goblin War
Drums, Jacques le Vert and Orcish Oriflamme. Tests cover opposing-controller
exclusion, owner/control distinction, matching source inclusion, source and
recipient control changes, late entry, tap/untap, combat membership, color,
phasing, source departure/return, ability removal and clone/factory isolation.
Llanowar Elves and Wall of Wood supply complete real RON fixtures. Invalid
selector/scope/Boolean declarations and nested inventory roles are tested.
New engine regressions independently test control phasing, conditional expiry,
cleanup, targeted/global and printed/direct removal dependencies, restoration,
unrelated abilities, returned sources and clone latch isolation. Complete
card-to-test mapping is in the companion report; no global behavior-verification
total or full-collection gameplay certification is claimed.

Fresh strict loading: **874/2,445 compiled records**, **1,571 rejected records**,
**256 compiled unique names**, **730 rejected unique names**. Gain: **15 records,
five unique cards**, exactly this run's estimate. No accepted record regressed;
no remaining first diagnostic changed. Strict loading still exits 1 and publishes
no partial registry. Remaining first gaps include PermanentRuleEffect (110),
AsPermanentEnters (90), ActivatedWithModifiers (61), CastEffect on spells (53),
rule If (47), ChooseADamageSource (46), MayCost (43), WhenAPlayerCastsASpell (43),
Targeted_Modal (42), HostController upkeep references (35), and
ControlledByAPlayer in other contexts (28).

No source defect was corrected; all 12 source hashes match the vocabulary
snapshot. No snapshot refresh is needed. Gaea's Liege/Fastbond observations
remain open. No environment failure or pre-existing check failure blocked this
run. Initial lint found a deprecated Keyword alias in a new test; it was changed
to Attr. Automatic formatting only changed new test import order. User-modified
files and unrelated files were not changed by lint.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, `make build` | PASS |
| Baseline strict loader and AST inventory | Expected exit 1; 859 compiled, 1,586 rejected |
| Focused new tests with `-count=1` | Expected compiler, declaration, phasing, dependency and inventory failures before their fixes; PASS afterward |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | PASS after new-test alias correction; zero issues |
| `git diff --check` | PASS |
| Final `go run ./cmd/loadcards -json` | Expected exit 1; 874 compiled, 1,571 rejected |
| Final `go run ./cmd/loadcards -inventory -json` | Expected exit 1; same counts, nested controller roles |
| Vocabulary hash comparison | All 12 hashes match |

Files: `layers.go`, `inventory.go`, `inventory_test.go`, new
`static_controller_test.go` under `pkg/mage/dsl/ron`; `effect_control.go`,
`effect_manager.go`, `continuous_effects.go`, `controller_operation_test.go` and
`subtype_ability_effects_test.go` under `pkg/mage`; `docs/dsl-reference.md`,
`docs/engine-rules-contracts.md`, the new static-controller report and this ledger.
All dependencies are tracked. These 13 files form one descriptive slice commit;
the two initial user changes stay outside it.

Next ranked slice: static Other(ThisPermanent) source exclusion for the three
Kobold lords, then variable temporary P/T for Howl from Beyond/Great Defender,
then optional fixed counter payments. Complete RON gameplay/loading, embedded
startup, native startup evidence, Go-card deletion, obsolete Go-set generator
retirement and unused-DSL removal gates remain unmet. No deletion or caller audit
is claimed. This run stops after P1-STATIC-CONTROLLER.

## Run 2026-10-03: P1-STATIC-OTHER

Status: complete static source-exclusion slice. No second slice was started.
Initial user changes to `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from this commit. No unrelated untracked files existed initially.

The fresh three-candidate comparison, complete affected-name lists, exact
signatures/contexts, dependencies, engine reuse, source findings, card-to-test
mappings and ranked next candidates are in
[mtgish-ron-static-other.md](mtgish-ron-static-other.md).

Selection: static source exclusion affects six unique names and unlocks three
Kobold lords. Variable temporary P/T affects six names and unlocks two; optional
fixed counter payments affect four names and unlock two with broader scope and
payment prerequisites. Complete records exposed the remaining subtype-only,
landwalk and granted-activation gaps of Goblin King, Lord of Atlantis and Zombie
Master. Source exclusion offers the largest credible gain for this effort.

Implemented `Other(ThisPermanent)/1` in creature-restricted
`EachPermanentLayerEffect/2` filters below nonempty recursive `And/1`, with or
without the existing `ControlledByAPlayer(SinglePlayer(You))/1` conjunct. Fixed
`AdjustPT/2` and basic `AddAbility/1` modifications retain their contracts.
RON keeps source exclusion in an immutable staticCreatureFilterPlan; each factory
builds fresh effects. Inventory reports the typed permanent operand. Malformed
and wrong-role operands, source-dependent Or predicates and unsupported compiler
contexts remain rejected.

Engine reuse: BoostOtherControlledCreatures, GrantKeywordToOtherControlled,
BoostAllCreatures with WhileOnBattlefield, GrantKeywordToAll, printed static
registration, layers, source-removal dependencies, control/phasing and clone
services. No engine prerequisite was missing. CR 611.3a/b/c, 613.1b/f/g, 613.4c,
613.8a/b and 702.26b/d supply the contract. No new cost, LKI value, target,
replacement or batch operation was added. No RON asset, engine implementation,
startup wiring, Go definition, generator or DSL export was changed.

TDD: tests first failed with unsupported Other in the three real lords and the
synthetic global declaration, and unknown source operand inventory roles. The
synthetic creature fixture needed CardPT before the intended compiler failure.
Focused tests now cover all three real cards, printed first strike, both fixed
bonuses, granted keywords, source exclusion, independent products of the same
factory, owner/control distinction, other-instance accumulation, later entry,
non-Kobold/opposing exclusions, source and recipient control/phasing, source
ability removal/restoration, departure/return and clone isolation. Declaration
and nested/global tests establish the supported boundaries. Crimson Kobolds,
Llanowar Elves and Forest are real RON fixtures.

Strict loading: **877/2,445 compiled records**, **1,568 rejected records**,
**259 compiled unique names**, **727 rejected unique names**. Gain: **three
records and three unique cards**, exactly the estimate: Kobold Drill Sergeant,
Kobold Overlord and Kobold Taskmaster. These cards also gain direct RON gameplay
coverage for their printed rules. Compilation and behavior coverage remain
separate; no global gameplay total or completion certification is claimed.
No accepted record regressed. Eighteen reprints of three names advance their
first errors: Goblin King/Lord of Atlantis to subtype-only global filters;
Zombie Master to Landwalk. Strict loading still publishes no partial registry.

Leading first gaps remain PermanentRuleEffect 110 records, AsPermanentEnters 90,
ActivatedWithModifiers 61, spell CastEffect 53, rule If 47, ChooseADamageSource
46, MayCost 43, WhenAPlayerCastsASpell 43, Targeted_Modal 42, HostController
upkeep references 35, and ControlledByAPlayer in other contexts 28. These are
compiler/context gaps, not build or environment failures. No source defect was
corrected. All 12 RON hashes match the vocabulary snapshot; no refresh is needed.
Gaea's Liege/Fastbond observations remain open.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, `make build` | PASS |
| Baseline `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; 874 compiled, 1,571 rejected |
| New `go test ./pkg/mage/dsl/ron -run TestRONOtherStatic -count=1` | Expected compiler/inventory failures before implementation; PASS afterward |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | Final PASS |
| `go test ./pkg/mage/... -short -count=1` | Final PASS after count updates and lint |
| `make test` | Final PASS after count updates and lint |
| `make build` | PASS; repeated after lint |
| `go vet ./...` | PASS; repeated after lint |
| `make lint` | PASS, zero issues |
| `git diff --check` | PASS |
| Final `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; 877 compiled, 1,568 rejected |
| Vocabulary source hash comparison | All 12 hashes match |

The first broad test run failed only on two exact inventory assertions that still
expected the old compilation counts. They now assert the fresh measured counts;
registry and gameplay checks were retained. Automatic lint formatting only
reordered imports in the new test. No unrelated file was changed by lint.
No environment or pre-existing check failure blocks this slice.

Files: `layers.go`, `inventory.go`, `inventory_test.go`, new `static_other_test.go`
under `pkg/mage/dsl/ron`; `docs/dsl-reference.md`, new static-other report and this
ledger. These seven files form one descriptive slice commit. Existing user
changes remain outside it.

Next ranked slice: variable temporary P/T for Howl from Beyond/Great Defender;
then optional fixed counter payments for Force Spike/Vodalian Mage; then static
landwalk and creature-subtype subjects for Goblin King/Lord of Atlantis (Zombie
Master also requires granted activation support). Reassess fresh evidence first.
Full RON gameplay/loading, embedded startup, native startup evidence, Go-card
and obsolete-generator deletion, and unused-DSL removal gates remain unmet.
No deletion or whole-repository caller audit is claimed. This run stops here.

## Run 2026-10-03: P1-TEMPORARY-VALUES

Status: complete captured temporary P/T slice. No second slice was started.
Initial user changes to `.gitignore` and `mtgish-ron-execution.md` remain preserved
and excluded from the commit. No unrelated untracked files existed initially.

The fresh three-candidate comparison, affected-name lists, exact signatures and
contexts, dependencies, engine reuse, card-to-test mappings and ranked next
candidates are in [mtgish-ron-temporary-values.md](mtgish-ron-temporary-values.md).
Selection: temporary variable P/T affects six unique names and unlocks two;
optional fixed counter payments affect four and unlock two with broader
payer/decision/result prerequisites; static landwalk/subtype subjects affect
three and unlock two with two adapter families. Temporary values offer the
smallest implementation with concrete downstream benefit for Kry Shield/Subdue
and a numeric lowering boundary for Blood Lust/Berserk. The previous ledger
suggestion was reassessed rather than used as fixed priority.

Implemented `AdjustPTX/3` inside existing temporary layers, with PlusX, MinusX,
Zero and Integer/1 coordinates. Numeric values support declared ValueX,
Integer/1 and ManaValueOfPermanent/1 of the modified subject. ActionPermanent
is local to the numeric modification and names that subject. Unbound,
malformed, wrong-role and other-object references fail at compilation.
Static AdjustPTXY, power/toughness readers, arithmetic and broader value
contexts remain unsupported. Each factory builds values through its own typed
permanent bindings; no shared mutable binding or last-action slot was added.

Engine reuse: ModifyPT, Until, Fixed, ValueAtResolution, XValue, ManaValueOf,
MultiplyValues, transactional casting, target legality, copy/clone frames,
captured incarnations, layers and cleanup. No engine prerequisite was missing.
Rules evidence: CR 107.3a/g, 202.3/3e, 400.7, 608.2b/c/h, 611.2a/c/d, 613.4c
and 702.26b/d/f. Values are read when the layer starts, then captured. Other-object
reads need a separate LKI contract and stay rejected. No replacement, batch zone
operation, new engine API, source-data correction or RON asset change occurred.

TDD first failed with unsupported AdjustPTX for both real cards and the synthetic
mixed modification; nested inventory-role tests then failed before the adapter.
Fixture corrections fixed an incorrect copy-method name, inactive source-less
static fixture bonus and a missing synthetic Aura enchant declaration.
Final tests cover invalid declarations, fresh factory/definition bindings,
announced X and zero X, payment rollback, announcement/resolution timing,
retargeted copies, clone isolation, returned and phased targets, installed
phasing, expiry, source departure, local subject identity, signed coordinates,
mixed modifications and changed mana values before/after resolution. Synthetic
source, host and trigger contexts also execute through the engine. A returned
target after installation gets no old bonus. No tests were weakened.

Strict loading: **884/2,445 compiled records**, **1,561 rejected records**,
**261 compiled unique names**, **725 rejected unique names**. Gain: **seven
records and two unique cards**, exactly the estimate. Howl from Beyond's six
records and Great Defender's one record gain direct RON gameplay coverage for
all their rules. No accepted record regressed. Remaining records still cause
exit 1, and strict loading publishes no partial registry. Compilation and
gameplay coverage remain separate; no global behavior total is claimed.

Remaining first gaps include PermanentRuleEffect (110 records), AsPermanentEnters
(90), If (63 across contexts), ActivatedWithModifiers (61), CastEffect (54 across
contexts), ChooseADamageSource (46), MayCost and WhenAPlayerCastsASpell (43 each),
Targeted_Modal (42), HostController (35), ControlledByAPlayer (29 across contexts).
These are compiler/context gaps. Gaea's Liege/Fastbond observations remain open.
All 12 source hashes match the vocabulary snapshot; no refresh is needed.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, `make build` | PASS |
| Baseline strict loader and nested inventory | Expected exit 1; 877 compiled, 1,568 rejected |
| New focused `TestRONTemporaryValue` tests with `-count=1` | Expected unsupported-shape/inventory failures before adapters; final PASS |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues |
| `git diff --check` | PASS |
| Final strict loader and nested inventory | Expected exit 1; 884 compiled, 1,561 rejected |
| Vocabulary hash comparison | All 12 hashes match |

Two inventory assertions initially failed on old exact compilation counts;
only those counts were updated, retaining gameplay/registry assertions.
Automatic lint changes only reordered new test imports. Final required checks
were run after the last test additions and lint. No environment or pre-existing
failure blocks the slice.

Files: compiler.go, inventory.go, inventory_test.go, new temporary_value.go and
new temporary_value_test.go under pkg/mage/dsl/ron; docs/dsl-reference.md, new
temporary-values report and this ledger. These eight files form one standalone
slice commit. User-modified files remain excluded.

Next ranked slice: optional fixed counter payments (Force Spike/Vodalian Mage),
then static landwalk/subtype subjects (Goblin King/Lord of Atlantis), then
power/toughness and branch prerequisites for Blood Lust/Berserk. Reassess fresh
evidence before selection. Full RON gameplay/loading, startup cutover, native
startup evidence, Go-card/generator deletion and unused-DSL removal gates remain
unmet. No deletion or whole-repository caller audit is claimed. This run stops.

## Run 2026-10-03: P7-STATIC-LANDWALK

Status: complete static landwalk grant slice. No second slice was started.
Initial user changes to `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from the slice commit. No unrelated untracked files existed.

The fresh three-candidate comparison, complete affected-name lists, exact
signatures/contexts, dependencies, rules evidence, engine reuse, source findings
and card-to-test mappings are in
[mtgish-ron-static-landwalk.md](mtgish-ron-static-landwalk.md).
Static landwalk affects six distinct names and unlocks three for a small adapter;
optional fixed counter payments affect four and unlock two with broader scope
and payment prerequisites; numeric branches/readers affect four and unlock one
with broader value/LKI prerequisites. Fresh nested inventory found the Aura
and Hidden Path gains beyond the prior ledger's lord-only suggestion.

Implemented `AddAbility([Landwalk(IsLandType(symbol))])/1` in Aura-host
`PermanentLayerEffect/2` and existing explicit creature-restricted
`EachPermanentLayerEffect/2`, including controlled/source-excluded contexts.
Accepted symbols: Plains, Island, Swamp, Mountain, Forest and Desert. Printed
landwalk parsing is reused through a private helper. Each factory builds fresh
effects. Inventory reports static keyword operands, landwalk filters and land
subtypes without asserting support. Temporary and rule landwalk contexts remain
separate. No public DSL export or mutable binding was added.

The subtype-only global guard remains. CR 109.2/205.3m/208.3a require a contract
for matching noncreature subtype permanents before accepting Goblin King/Lord
of Atlantis. Zombie Master also needs granted activated abilities. No source
defect was corrected. All 12 RON hashes match the vocabulary snapshot; no
refresh is needed. Gaea's Liege/Fastbond observations remain open.

Engine reuse: static plans, GrantAbilityToAttached, global/controlled keyword
grants, printed-static registration, shared layer/removal ordering, attachments,
phasing, control, clones and landwalk combat queries. TDD exposed a prerequisite:
HasLandwalkEvasion counted phased defending lands. A focused engine regression
failed for six subtype attributes plus legendary landwalk, then passed after
both query branches excluded phased lands. The implement-engine skill supplied
the contract-test workflow. No new engine API, replacement, cost, event, LKI
reader or simultaneous operation was added.

Tests first confirmed unsupported grants for Burrowing/Fishliver Oil/Hidden Path
and missing inventory roles. Two initial fixture API errors were corrected
before that expected failure. Real RON gameplay tests cover complete Aura
records, target legality/return/phasing, mana state after rejected announcement,
resolution timing, attachment changes, indirect phasing, source ability removal,
source departure, defender control/land phasing and clone isolation. Hidden Path
covers current color/creature membership, both controllers, late entry, source
control, phase cycles, removal/restoration and departure/return. Synthetic tests
cover all six subtypes, mixed grants/P/T, factory isolation, accumulation and
controlled source exclusion. Engine tests add ownership/control, departure/return
and independent clone phasing evidence. Complete test mappings are in the report.

Strict loader: **892/2,445 compiled records**, **1,553 rejected records**,
**264 compiled unique names**, **722 rejected unique names**. Gain: **eight
records and three unique cards**, exactly the estimate. Burrowing (six reprints),
Fishliver Oil and Hidden Path gain full-record RON gameplay evidence. No accepted
record regressed. Six Zombie Master records advance their first diagnostic to
the second subtype-only global selector; other rejected diagnostics are unchanged.
Strict loading still exits 1 and publishes no partial registry. Compilation and
gameplay coverage are separate; no global gameplay total is claimed.

Remaining first gaps include PermanentRuleEffect (110 records), AsPermanentEnters
(90), If (63 across contexts), ActivatedWithModifiers (61), CastEffect (54 across
contexts), ChooseADamageSource (46), MayCost and WhenAPlayerCastsASpell (43 each),
Targeted_Modal (42), HostController (35), And (31 across contexts) and
ControlledByAPlayer (29 across contexts). These are compiler/context gaps.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, `make build` | PASS |
| Baseline strict loader and AST inventory | Expected exit 1; 884 compiled, 1,561 rejected |
| Focused `TestRONStaticLandwalk` with `-count=1` | Expected unsupported grants/inventory roles before adapters; final PASS |
| Focused `TestLandwalkEvasionPhasingAndController` with `-count=1` | Expected phased-land failures before fix; final PASS |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues |
| `git diff --check` | PASS after removing a trailing ledger blank line |
| Final `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; 892 compiled, 1,553 rejected |
| Vocabulary source hash comparison | All 12 hashes match |

Two exact inventory count assertions were updated to fresh measured counts;
existing gameplay/registry assertions remain. Automatic lint changed only the
new RON test's import grouping. It did not change user files or unrelated code.
No environment or pre-existing check failure blocks this slice.

Files: compiler.go, layers.go, inventory.go, inventory_test.go and new
static_landwalk_test.go under pkg/mage/dsl/ron; combat.go and new
landwalk_evasion_test.go under pkg/mage; docs/dsl-reference.md,
docs/engine-rules-contracts.md, the new static-landwalk report and this ledger.
These eleven files form one descriptive commit. No RON record, runtime wiring,
Go definition, generator or DSL export was changed.

Next ranked slice: general static subtype subjects for Goblin King/Lord of
Atlantis, with exact noncreature grant/P/T contracts first; then optional fixed
counter payments for Force Spike/Vodalian Mage; then characteristic reads,
arithmetic and branches for Blood Lust and downstream Berserk/Kry Shield/Subdue.
Reassess fresh evidence before choosing. Complete RON gameplay/loading, embedded
startup, native startup evidence, Go-card and generator deletion, and unused-DSL
removal gates remain unmet. No deletion or caller audit is claimed. This run stops.

## Run 2026-10-03: P7-STATIC-PERMANENTS

Status: complete general static permanent subject slice. No second slice was
started. Initial user changes to `.gitignore` and `mtgish-ron-execution.md` remain
preserved and excluded. No unrelated untracked files existed initially.

The fresh three-candidate comparison, exact affected-name lists, signatures,
contexts, dependencies, source observations, rules evidence and card-to-test
mapping are in [mtgish-ron-static-permanents.md](mtgish-ron-static-permanents.md).
General static subjects unlock two unique cards for a small adapter using
existing engine primitives. Optional counter payments also offer two unlocks
but need broader scope/payment work; numeric branches offer one with broader
value contracts. General subjects also supply concrete prerequisites for Zombie
Master, Energy Flux, Kormus Bell, Living Lands, Living Plane, Conversion, Blood
Moon and Guardian Beast. Complete records and nested forms were inspected;
first-error counts were not used as full-card estimates.

Implemented general `EachPermanentLayerEffect(filter, modifications)/2` for
fixed `AdjustPT/2` and existing basic keyword or supported `Landwalk/1` grants
inside `AddAbility/1`. Existing permanent filters select noncreatures when the
RON does not specify Creature. Static And/Or lists are recursively nonempty.
Other(ThisPermanent) and ControlledByAPlayer(SinglePlayer(You)) work at the root
and in conjunctions, with source-dependent disjunctions still rejected. General
modification lists must produce at least one modification. Other static
modifiers and ability grants remain unsupported. Per-factory sets and rules use
MatchingPermanents, ControlledPermanents, WithoutPermanent, ModifyEachPT,
GrantEachKeyword, While, Static and a constant pure StateConditionFunc. The
engine owns current membership, source lifetime, control, printed ability
activity, incarnation capture, layers and clones. No engine repair or new public
API was needed. Explicit-creature and Aura paths retain their engine services.

TDD first confirmed unsupported-subject failures for both real lords and both
synthetic noncreature contexts. The inventory-role test later confirmed an
unknown subtype operand before its adapter. Two cleanup fixture API errors
were corrected before that intended inventory failure. Focused tests establish
complete real Goblin King/Lord of Atlantis rules, casting costs and timing,
rejected unpaid casting, cloned stack resolution, global/controlled noncreature
keywords, later animation, explicit creature boundaries, subtype changes and
expiry, root/nested source exclusion, owner/control distinction, accumulation,
independent same-factory sources, later entry, phasing, departure/return,
source ability removal/restoration, landwalk defender identity and clone
isolation. Invalid declarations cover malformed/unbound/wrong-role filters,
controller forms, dependent disjunctions and nested empty logical lists.

Inventory reports creature_subtype operands for unary IsCreatureType and
IsNonCreatureType under permanent filters, including rejected ancestors. All
signature support remains unverified; global engine-validation and behavior
verification totals remain null. Two exact count assertions were updated.
The former root-Other fixed-P/T rejection is now a supported contract, tested
by the new root-exclusion test; its rejection case now checks unsupported
static AdjustPTX. No other gameplay/registry assertion was weakened.

Strict loading: **904/2,445 compiled records**, **1,541 rejected records**,
**266 compiled unique names**, **720 rejected unique names**. Gain: **12 records
and two unique cards**, exactly the estimate: Goblin King and Lord of Atlantis,
six identical records each. Both gain direct full-record RON gameplay coverage.
Before/after record diagnostic comparison found no accepted-record regression.
Five rejected names advance to later modifiers/grants: Zombie Master to
Activated; Energy Flux to TriggerA; Kormus Bell/Living Lands/Living Plane to
SetPT. Strict loading still exits 1 and publishes no registry. Compilation and
gameplay coverage remain separate; no global gameplay total is claimed.

Leading first gaps: PermanentRuleEffect 110 records, AsPermanentEnters 90,
If 63, ActivatedWithModifiers 61, CastEffect 54, ChooseADamageSource 46,
MayCost/WhenAPlayerCastsASpell 43 each, Targeted_Modal 42, HostController 35,
ControlledByAPlayer 29, TriggerI 27. These are compiler/context gaps. No source
defect was corrected. Gaea's Liege/Fastbond observations remain open. All 12
source hashes match the vocabulary snapshot; no refresh is needed.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, `make build` | PASS |
| Baseline strict loader and nested inventory | Expected exit 1; 892 compiled, 1,553 rejected |
| New `go test ./pkg/mage/dsl/ron -run TestRONStaticPermanent -count=1` | Expected subject and then inventory failures before adapters; final PASS |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS; repeated after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues |
| `git diff --check` | PASS |
| Final `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; 904 compiled, 1,541 rejected |
| Vocabulary source hash comparison | All 12 hashes match |

The first complete RON package run failed only on two old exact compilation
counts and the newly supported declaration's old rejection assertion. These
were updated as described above. Lint made no further source changes after
gofmt. No environment or pre-existing failure blocks this slice.

Files: layers.go, inventory.go, inventory_test.go, static_other_test.go and new
static_permanent_test.go under pkg/mage/dsl/ron; docs/dsl-reference.md,
docs/engine-rules-contracts.md, the new static-permanents report and this ledger.
These nine files form one descriptive slice commit. No RON asset, Go card,
engine implementation, generator or startup wiring changed.

Next ranked candidates: shared static land animation for Living Lands/Living
Plane/Kormus Bell (three possible unlocks); optional fixed counter payments for
Force Spike/Vodalian Mage (two); numeric readers/arithmetic/branches for Blood
Lust and downstream Berserk/Kry Shield/Subdue (one immediate unlock). Reassess
fresh evidence before selection. Defer singleton grants while useful multi-card
work remains. Complete gameplay/loading, embedded startup, native startup,
Go-card/generator deletion and unused-DSL removal gates remain unmet. No deletion
or whole-repository caller audit is claimed. This run stops after this slice.

## Run 2026-10-04: P7-STATIC-ANIMATION

Status: complete combined static animation slice. No second slice was started.
Initial user changes to `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from the slice commit. No unrelated untracked files existed.

The three-candidate comparison, complete name lists, exact supported contexts,
dependencies, engine reuse, rules evidence, source findings and test mappings
are in [mtgish-ron-static-animation.md](mtgish-ron-static-animation.md).
Static animation offers three full-card unlocks; optional fixed counter payments
offer two with broader payer/result scopes; numeric branches offer one with
broader value/branch prerequisites. Complete records and nested forms were
inspected, including identical reprints. Animation was selected for its largest
credible gain. Missing exact engine behavior was implemented in this same slice
using the implement-engine skill workflow, before compiler lowering.

Supported rule: `EachPermanentLayerEffect(filter, modifications)/2` on permanent
cards, with exactly one `SetPT(PT(signed_integer, signed_integer))/1`, exactly
one `AddCardtype(Creature)/1`, and optionally one
`SetColor(SimpleColorList(list))/1`. Colors are the five actual color symbols;
an empty list makes subjects colorless. Omission preserves colors. List order
is arbitrary; duplicates, dynamic values, other card types, standalone
modifications and mixed unsupported lists remain errors. Existing static
filters, source exclusion and controller conjunctions are reused. Aura and
temporary contexts are unchanged.

New engine signature:
`AnimateEach(PermanentSet, int, int, ...core.Color) StaticRule`, installed by
`Static`. The service adds Creature in layer 4, optionally sets color in layer
5, and sets base P/T in layer 7b. The effect manager retains selected incarnation
references for one layer cycle (CR 613.6), including after type addition or
layer-6 source ability removal, then selects current membership on the next
application. Rule/source identities keep instances distinct. Cloned effect
managers have independent cycle state. The new service tests membership-changing
type dependencies on isolated game/effect clones and uses the existing sorter.
This does not establish complete dependency support for all engine effects.

Prerequisite repairs: printed static wrappers forward P/T sublayers; added
creature types supply ordinary combat eligibility without a Defender exception.
Printed-creature base restrictions and explicit attack/block prohibitions remain
in force. Existing source lifetime, incarnation, control, phasing, copy-on-write
clones, intrinsic land mana, announcement costs, stack resolution and event
capture services are reused. RON owns spelling and compilation rejection;
reusable rules stay in the engine. No fallback, card-name branch, last-action
slot or partial production registry was added.

TDD confirmed absent AnimateEach and unsupported real SetPT. Additional engine
tests failed on static timestamp order, Defender bypass and a later land-type
dependency before their fixes. Inventory operand tests failed with unknown roles
before the adapter. Fixture API errors were corrected without changing the
contract. The first broader test run exposed the printed-creature base attack
restriction regression and two old exact inventory counts. The eligibility
change was narrowed to added creature types, those regression tests passed,
and count assertions were updated to measured totals. Gameplay assertions were
retained. No validation failure was waived.

Affected cards and evidence:

| Unique card or context | Tests |
| --- | --- |
| Living Lands, Living Plane, Kormus Bell | `TestRONStaticAnimationRealCards`: complete real records, casting costs and rollback, resolution timing, both players, current membership, matching bounds, late entry, summoning sickness, preserved land types/mana, colors, phasing, source ability removal, departure/return and cloned stack resolution |
| Synthetic scope/instances/colors | `TestRONStaticAnimationInvalid`, `TestRONStaticAnimationInstancesAndControl`, `TestRONStaticAnimationColorsAndInventory` |
| Engine layer/identity contracts | `TestStaticAnimationLayerContinuation`, `TestStaticAnimationLayerOrder`, `TestStaticAnimationTypeDependency`, `TestStaticAnimationCombatAndMana`, `TestStaticAnimationEntryAndIncarnations`, `TestStaticAnimationMetadataAndInvalidSet` |

Engine tests add noncreature selection retention, layer-6 continuation, both
7b timestamp orders, 7c boosts/counters, 7d switching, type dependencies,
Defender/prohibitions, source and subject identities, entry-time creature/power
capture, departure before trigger resolution, metadata and invalid sets.
No new general LKI reader, replacement order, optional payment, APNAP or
simultaneous zone-operation contract is claimed.

Strict loader and fresh AST inventory: **917/2,445 compiled records**,
**1,528 rejected records**, **269 compiled unique names**, **717 rejected
unique names**. Actual gain: **13 records and three unique cards**, exactly the
estimate: Living Lands and Kormus Bell each six reprints, Living Plane one.
Before/after diagnostics show no accepted-record regressions; all remaining
record diagnostics are identical. Each new card has direct complete-record
RON gameplay coverage. Compilation and gameplay remain separate; no global
behavior or engine-validation total is claimed. Strict loading still exits 1
and publishes no registry.

Remaining leading first gaps: PermanentRuleEffect 110 records, AsPermanentEnters
90, If 63, ActivatedWithModifiers 61, CastEffect 54, ChooseADamageSource 46,
MayCost/WhenAPlayerCastsASpell 43 each, Targeted_Modal 42, HostController 35,
ControlledByAPlayer 29 and TriggerI 27. These are compiler/context gaps, not
environment failures. The new inventory operand roles still report support
unverified. Local LEA/LEG Oracle data agrees with the selected records. No RON
source changed or defect was corrected; all 12 file hashes match the vocabulary
snapshot, so no source refresh is required. Gaea's Liege/Fastbond concerns remain
unresolved.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1`, `make build` | PASS |
| Baseline strict loader and nested inventory | Expected exit 1; 904 compiled, 1,541 rejected |
| New focused `TestStaticAnimation*` and `TestRONStaticAnimation*`, `-count=1` | Expected failures above before fixes; final PASS |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues |
| Post-lint focused engine/RON/CLI checks with `-count=1` | PASS |
| `git diff --check` | PASS |
| Final `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; 917 compiled, 1,528 rejected |
| Before/after diagnostics and vocabulary hashes | No accepted regression, remaining diagnostics unchanged, all source hashes match |

Automatic formatting reordered the new engine service's import groups. Review
found no semantic lint changes or changes to the two user files. No environment
or pre-existing check failure blocks this slice. Native startup, WASM and
runtime cutover were not changed, and no startup claim is made.

Files: `pkg/mage/static_animation.go`, `static_animation_test.go`, `card.go`,
`effect_manager.go`, `doc.go`; `pkg/mage/dsl/ron/static_animation.go`,
`static_animation_test.go`, `layers.go`, `inventory.go`, `inventory_test.go`;
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md`, the new static-animation
report and this ledger. Only these fourteen files are included in the slice
commit.

Next ranked candidates: optional fixed counter payments for Force Spike/Vodalian
Mage (two immediate unlocks); shared temporary animation prerequisites for
Mishra's Factory/Jade Statue/Thelonite Druid (fresh timing, cost, duration and
collection audit required); then characteristic readers/branches for Blood Lust
and downstream Berserk/Kry Shield/Subdue. Reassess fresh evidence before
selection. Defer single-card adapters while useful multi-card work remains.
Complete RON gameplay/loading, embedded runtime startup, native smoke tests,
Go-card/generator deletion and unused-DSL cleanup gates remain unmet. No deletion
or whole-repository caller audit is claimed. This run stops after this slice.

## Run 2026-10-04: P4-OPTIONAL-FIXED-MANA

Status: complete the in-progress optional payment slice. Overall migration
remains incomplete. Production still uses Go card definitions.

The working tree began with optional-payment compiler/player-scope changes,
the new adapter and gameplay tests, and user edits to `.gitignore` and the
execution design. The latter two files are preserved outside this slice commit.
The existing payment gameplay tests passed. Both failing tests were exact
inventory counts that still expected the preceding slice's 917 compiled records.
They now assert the measured 919 compiled and 1,526 rejected records; retained
registry and gameplay assertions are unchanged.

Exact shapes, boundaries, dependencies, rules and card-to-test mapping:
[mtgish-ron-optional-payment.md](mtgish-ron-optional-payment.md).
The compiler lowers each adjacent payment/unpaid pair into one `UnlessPaid`.
It reuses `ControllerOfSpell`, typed stack targets, fresh instance bindings,
cost factories, `Sequence`, counter restrictions and action payment transactions.
No engine service or RON asset changed. Independent/nested payment results
cannot overwrite one another. General payment-result variables remain deferred.

Candidate comparison, based on complete RON records and fresh diagnostics:

| Candidate | Distinct affected names | Estimated full-card unlocks | Remaining prerequisites and effort |
| --- | --- | ---: | --- |
| Optional fixed counter payments | Force Spike, Vodalian Mage; downstream Power Sink, Thrull Wizard | 2 | Existing engine services; small adapter plus scope/payment tests. Power Sink still needs X payment and player action lists; Thrull Wizard needs alternative costs. |
| Temporary animation | Mishra's Factory, Jade Statue, Thelonite Druid | Up to 3 with all prerequisites | Temporary type/subtype/base P/T composition, collection animation, timing modifiers, end-of-combat duration and matching sacrifice costs; substantially larger test scope. |
| Characteristic readers and branches | Blood Lust, Berserk, Kry Shield, Subdue | 1 immediately | Blood Lust needs toughness, subtraction and conditional branches; others retain delayed trigger, casting restriction or prevention blockers. Medium effort. |

The fixed payment pair provides two immediate unlocks with the least new
machinery and completes the work already in progress. Inspection of nested
payment forms found other payer, branch, cost and parent-context blockers;
no additional full-card unlocks are claimed.

TDD for the missing reporting contract confirmed that `PayMana` under
`PlayerMayCost` lacked its cost role. Inventory now reports payment player/cost,
continuation condition/action and spell-controller operand roles. The new test
passes. Signature support stays unverified, and global behavior/engine-validation
totals remain null.

Strict loader and inventory: **919/2,445 compiled records**, **1,526 rejected**,
**271 compiled unique names**, **715 rejected unique names**. Actual gain:
**two records and two unique cards**, matching the estimate: Force Spike and
Vodalian Mage. A loader run with a `HEAD` compiler overlay reproduced the
917-record baseline. Record-by-record comparison found no new rejection.
Three still-rejected names advance to precise later diagnostics: Power Sink
(`PayManaX`, six records), Thrull Wizard (alternative cost), and Chain Lightning
(recipient-to-player reference; its paid branch remains another blocker).
All other rejected diagnostics are unchanged. Strict loading still exits 1
and publishes no partial registry; this is the expected migration state.

Gameplay tests cover payment/refusal/inability, resolution timing, mana-source
use, unpaid casting/activation, paid tap costs, source departure/phasing,
current stack controller, illegal targets, uncounterable spells, recast identity,
copies, clones, zero costs, colored affordability and nested result independence.
Local LEG/FEM Oracle data agrees with the selected complete card records.
No source data correction or runtime cutover is part of this slice.

Validation: focused RON/CLI tests with `-count=1`, `make test`, `make build`,
`go vet ./...`, `make lint` (zero issues), and `git diff --check` pass.
The build and lint were repeated with approved cache access after sandbox
cache-write errors. Lint only reordered test imports and aligned the role map.
Post-lint focused tests pass. Loader and inventory totals agree.

Next ranked candidates: temporary animation for Mishra's Factory/Jade Statue/
Thelonite Druid; fixed upkeep payments with `MayCost` for Force of Nature and
related cards after a complete-record audit; characteristic readers/branches
for Blood Lust and the downstream cards above. Reassess estimates before
starting a new slice. Complete loading/gameplay, embedded startup, native/WASM
smoke tests, Go-card/generator deletion and unused-DSL cleanup remain open.

## Run 2026-10-04: P4-IMPLICIT-FIXED-MANA

Status: complete one fixed controller upkeep payment slice, with the
P5-CONTROLLER-DAMAGE adapter prerequisite. No second slice was started.
Initial user edits to `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from this commit. No unrelated untracked files existed.

The [upkeep payment report](mtgish-ron-upkeep-payment.md) records the three
candidate comparison, distinct names, complete-record audit, estimates,
dependencies, exact contexts, reusable engine contracts, source evidence and
card-to-test mappings. Fixed payments offered six estimated complete-card
unlocks at lower effort than temporary animation (three with several missing
contracts) or characteristic branches (one plus three blocked downstream cards).
The payment pair and the Force of Nature damage-recipient prerequisite were
selected and their contract and acceptance cases stated before edits.

Exact supported additions:

- `MayCost(PayMana(list))/1` adjacent to `Unless(CostWasPaid, list)/2` in
  action sequences, including supported trigger, activation and spell contexts.
  Fixed mana only; the unpaid body must be nonempty. Nested pairs have local
  outcomes. The payer is the stored resolving controller, independently of
  player action selectors. Existing `PlayerMayCost/2` callers keep working.
- `PermanentDealsDamage(ThisPermanent, value, Player(You))/3` in a permanent
  scope, with existing Integer/1 or bound ValueX values. Explicit Player/1 here
  accepts only the symbol You. Other sources/player forms remain errors.
- Inventory cost roles under MayCost/1 and player-reference roles under
  recipient Player/1. Reporting still does not claim signature support.

Engine reuse: `UnlessPaid`, `Costs`, `Sequence`, fresh cost/effect factories,
`DealDamageFrom`, `This`, `You`, `ValueAtResolution`; shared total-payment
transactions, mana solver, trigger queue/controller capture, incarnations,
LKI and damage/prevention/event services. No missing engine contract was found.
No engine implementation, RON source, runtime wiring, Go definition or DSL export
was changed. No fallback, card-name branch or global last-action result was
introduced. The old removal gates are still closed; no unused-caller audit or
production deletion is claimed.

TDD first failed on missing MayCost and inventory roles after fixture API
corrections. Later gameplay failures exposed incorrect test expectations for
sacrifice after control transfer, cloned BasePlayer choices, and trigger indexes
before activated abilities. Those expectations were corrected using existing
engine behavior. Clones use their default decline decision and retain separate
mana/life/stack state. Full RON tests then failed only on two old inventory totals;
these were updated from fresh measured counts without changing gameplay tests.

Affected unique cards and gameplay mapping (all tests in upkeep_payment_test.go):

| Card or context | Tests |
| --- | --- |
| Force of Nature, Phantasmal Forces, Junún Efreet, Sunken City, Palladia-Mors, Vaevictis Asmadi | `TestRONUpkeepPaymentRealCards`: ten scenarios per complete RON record; payment timing, own/opponent upkeep, pay/refuse/inability, colored mana sources and rollback, source departure/return/phasing, control changes and prevention |
| Sunken City; selected creatures' flying/trample; Vaevictis Asmadi's three pumps | `TestRONUpkeepPaymentOtherRules`: both controllers' blue creatures, boost departure, keywords, all three colored activations |
| Independent source/controller instances and game clones | `TestRONUpkeepPaymentInstancesAndClones` |
| Zero costs, fresh factories and nested independent results | `TestRONUpkeepPaymentZeroAndNested` |
| Juzám Djinn, Serendib Efreet, Elves of Deep Shadow | `TestRONUpkeepDamageAdditionalCards`: upkeep self damage, flying, immediate mana ability with tap cost and self damage |
| Force of Nature's source LKI/event incarnation and recipient | `TestRONUpkeepDamageLKI`: source departure, return under another controller, phasing and control transfer |
| Invalid shapes, costs, conditions and bindings | `TestRONUpkeepPaymentInvalid`, `TestRONUpkeepDamageInvalid` |
| Nested operand roles | `TestRONUpkeepPaymentInventory` |

Compilation increases from **919 to 939 of 2,445 records**; strict rejection
falls from **1,526 to 1,506 records**. Accepted unique names increase from
**271 to 280**, leaving **706 rejected unique names**. Actual gain is **20
records and nine names**: Force of Nature and Phantasmal Forces six reprints
each, Serendib Efreet two, and the other six names one each. The estimate of
six names counted the payment users but missed three direct damage-prerequisite
users. Their full records and gameplay were subsequently included. Compilation
and gameplay evidence remain separate; inventory engine_validated and
behavior_verified global totals remain null. The nine names have direct RON
rules tests; this is not a claim of complete collection behavior.

Before/after diagnostics show zero new rejection and 27 changed remaining
record diagnostics. Conversion now reaches SetLandType; Demonic Hordes reaches
opponent sacrifice choices; Arcades Sabboth reaches IsntAttacking; Breeding Pit
reaches end-step/token rules; Nicol Bolas reaches damage events; the Chants reach
entry events; Forethought Amulet reaches its replacement. Other changes expose
unsupported nonmana costs, paid branches and MustCost. All other rejected
record diagnostics stay the same. The leading first gaps are
PermanentRuleEffect 110, AsPermanentEnters 90, If 64, ActivatedWithModifiers 61,
CastEffect 54, ChooseADamageSource 46, WhenAPlayerCastsASpell 43,
Targeted_Modal 42, HostController 35 and ControlledByAPlayer 29 records.
These are compiler/context gaps, not environment failures.

Local Oracle data in LEA/ARN/DRK/LEG agrees with all nine unlocked records.
No source defect was corrected. All 12 RON file hashes match the vocabulary
snapshot, so its source syntax does not need refresh. Gaea's Liege/Fastbond
observations remain unresolved. The report lists inspected partially affected
cards and their other blockers; none is counted as a full unlock.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/... -short -count=1`, `make build` | PASS |
| Baseline strict loader and AST inventory | Expected exit 1; 919 compiled, 1,526 rejected |
| New `go test ./pkg/mage/dsl/ron -run TestRONUpkeepPayment -count=1` | Expected unsupported MayCost and missing-role failures; final PASS |
| Focused `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS, repeated after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS, repeated after final gameplay test change |
| `make test` | PASS, repeated after final gameplay test change |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues; repeated |
| `git diff --check` | PASS |
| Final `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; both 939 compiled, 1,506 rejected; no partial registry |
| Source hashes and rejection comparison | All hashes unchanged; no accepted-record regression |

Automatic lint changes only reordered the new test file's imports. Reviewed
changes have no semantic effect and neither user file was changed by lint.
No required check was waived and no environment/pre-existing failure blocks
this slice. The slice changes compiler.go, optional_payment.go, inventory.go,
inventory_test.go and the new upkeep_payment_test.go; dsl-reference.md,
engine-rules-contracts.md, optional-payment report, the new upkeep-payment
report and this ledger. Only these ten files are staged for one descriptive
commit. No startup or cutover validation is claimed.

Next ranked candidates: shared spell-cast events plus fixed paid continuations
for Crystal Rod, Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere and Urza's
Chalice (six credible unlocks after exact event/payer audit); temporary animation
for Mishra's Factory/Jade Statue/Thelonite Druid (three possible unlocks with
layer/timing/duration/cost dependencies); characteristic filters/branches for
Blood Lust and downstream Berserk/Kry Shield/Subdue (one immediate unlock).
Reassess fresh evidence next run. Complete loading/gameplay, embedded runtime,
native startup, Go-card/generator deletion and unused-DSL cleanup remain open.
