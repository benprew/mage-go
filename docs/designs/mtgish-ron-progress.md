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

## Run 2026-10-04: P5-FIXED-DRAW

Status: complete fixed draw activation slice, including fixed activation life
payment and the required color-choice mana adapter repair. Stop after this slice.
The initial user edits to `.gitignore` and `mtgish-ron-execution.md` remain intact
and are excluded from the commit. No unrelated untracked files were present.

The [fixed draw report](mtgish-ron-fixed-draw.md) records candidate comparison,
complete-record and nested-form audits, exact signatures/contexts, dependencies,
engine reuse, affected distinct names, rules evidence and card-to-test mappings.
Candidates were fixed draws/life costs (five estimated unlocks, small adapters
and a payment/binding matrix), spell-cast life triggers (seven estimated unlocks,
more event and paid-continuation work), and entry copying (two initial unlocks,
large entry/copy/layer scope; Doppelganger remains blocked). Fixed draws offered
the highest credible gain per effort. Selection and acceptance were stated
before editing; first-error frequency was not used as an unlock estimate.

Supported additions: symbol `DrawACard` in supported spell/activation/trigger
action contexts and `PlayerAction(player, DrawACard)` with existing typed player
references. Lowering uses `DrawFor(player, Fixed(1))`. Activation cost
`PayLife(Integer(n))` requires a nonnegative fixed integer, including nested
cost conjunctions, and builds fresh `LifePayCost(n)` objects. Dynamic/optional
life payments remain rejected. Inventory gives PayLife's operand the value role.
The existing fixed `AnyManaColor` adapter needed repair to complete Standing
Stones: it now uses `AddManaOfAnyOneColor` instead of storing AnyColor in the
pool. The full amount uses one chosen color. Regressions cover all accepted
users of this shape: Birds of Paradise, Black Lotus, Celestial Prism and
Implements of Sacrifice, plus the newly accepted Standing Stones.

All reusable rules already exist. Engine payment transactions, target instance
bindings, resolving controller capture, draw/replacement/empty-library services,
clone isolation, mana classification/solver and color decisions are reused.
No engine implementation, RON source, Go card definition, runtime startup or DSL
export was changed. No new general LKI, simultaneous-operation or replacement
ordering contract is claimed. The removal gates remain closed; no unused-caller
or definition-removal audit is claimed for this slice.

Tests first failed with missing DrawACard/PayLife adapters and the life inventory
role. The Standing Stones test and added accepted-card regressions then failed
on wildcard production before that adapter repair. Fixture corrections used
existing APIs, activation index zero, cleared summoning sickness for established
sources, and IsAlive after state-based actions clear the empty-draw flag.
No test was weakened to accommodate an implementation gap.

Gameplay tests in `fixed_draw_test.go` map all five new unique cards:
Jayemdae Tome, Xira Arien, Book of Rass and Greed to
`TestRONFixedDrawRealActivations`, `TestRONFixedDrawPaymentRollback` and
`TestRONFixedDrawLifeAndManaSources`; Xira additionally to
`TestRONFixedDrawXiraInstances`; Greed additionally to
`TestRONFixedDrawReplacementAndEmptyLibrary`; Standing Stones to
`TestRONFixedLifeStandingStones`. The report maps the four accepted mana-card
regressions and synthetic declaration, zero/equal/combined life payment, fresh
factory, player scope, trigger and spell-copy cases. Tests cover all rules in
the five unlocked records, but are not a full collection gameplay claim.

Estimated and actual compilation gain agree: **five names, ten records**.
Strict loader increases from **939 to 949/2,445 accepted records**; rejection
falls from **1,506 to 1,496**. Unique accepted names increase **280 to 285**;
**701 unique names remain rejected**. Jayemdae Tome contributes six reprints;
the other four new names contribute one each. No formerly accepted record
regresses. Two still-rejected diagnostics advance: Jalum Tome to DiscardACard
and Sindbad to RevealTheCardDrawnThisWay. All other rejected diagnostics remain
unchanged. Strict loader and full AST inventory diagnostics are identical.
Global engine_validated and behavior_verified measurements remain null;
compilation counts are separate from behavior evidence.

Local Oracle data agrees with all five new records. No source defect was
corrected and every RON hash matches the vocabulary snapshot, so no syntax
snapshot refresh is needed. The color-choice problem was a compiler adapter
defect. Gaea's Liege/Fastbond source concerns remain unresolved outside this
slice. No environment or pre-existing failure blocks completion.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1`, engine short tests, `make build` | PASS |
| Baseline strict loader and full inventory | Expected exit 1; 939 accepted, 1,506 rejected |
| New `go test ./pkg/mage/dsl/ron -run TestRONFixed -count=1` | Expected adapter/role failures; final PASS |
| Fixed mana prerequisite regressions before repair | Expected failure: AnyColor pool value instead of chosen color |
| Final focused RON/CLI tests with `-count=1` | PASS; repeated after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | First run rejected new test variable `copy` for builtin shadowing; renamed to `copied`; final PASS, zero issues |
| `git diff --check` | PASS |
| Final `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; both 949 accepted and 1,496 rejected; diagnostics identical |
| Source hashes and before/after rejection comparison | All hashes unchanged; no accepted-record regression |

Automatic lint changes only reordered imports in the new test file. Reviewed
changes have no semantic effect. Focused tests were repeated after the lint fix;
no required check was waived. Files in this slice: compiler.go, inventory.go,
inventory_test.go, new fixed_draw_test.go, dsl-reference.md,
engine-rules-contracts.md, new mtgish-ron-fixed-draw.md and this ledger. One
commit will include only these eight files; the two user files remain unstaged.

Next ranked candidates: spell-cast life triggers/fixed paid continuations
(Crystal Rod, Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere, Urza's Chalice,
Sol'kanar the Swamp King; seven estimated unlocks); temporary animation
(Mishra's Factory/Jade Statue/Thelonite Druid, up to three with dependencies);
shared discard action (Jalum Tome now exposes this blocker, but its multi-card
value requires a fresh nested and complete-record audit). No next slice was
started. Complete loading/gameplay, embedded production startup and native
smoke tests, Go-card/generator deletion and unused-DSL cleanup remain open.

## Run 2026-10-04: P6-SPELL-CAST-LIFE

Status: completed one shared spell-cast life-trigger slice. The initial user
changes to `.gitignore` and `mtgish-ron-execution.md` remain intact and are
excluded from the commit. No unrelated untracked files were present.

The [spell-cast life report](mtgish-ron-spell-cast-life.md) records the candidate
comparison, full-record/nested-form audit, signatures and contexts, rules
references, engine reuse, affected names, diagnostics and card-to-test mappings.
Candidates: spell-cast life triggers (seven complete unlocks, small adapters
with an event/payment matrix), temporary animation (up to three unlocks with
layer/timing/cost prerequisites), and fixed selected discard (two complete
unlocks, with later result/event/timing dependencies). The selected slice gives
the largest credible multi-card gain for its adapter effort. The initial
commentary identified Jalum Tome; the completed discard audit also found Bazaar
of Baghdad, without changing the selection. Reprints count once. Selection,
reuse boundary and acceptance cases were stated before implementation.

Supported event: `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, spell-filter),
actions)`, with symbol AnyPlayer and existing AnySpell, IsColor, IsCardtype and
nonempty And/Or spell filters. Other caster scopes, self-spell and event
references remain rejected. Fixed `MayCost(PayMana(...))` and typed
`PlayerMayCost(player, PayMana(...))` can now have an adjacent nonempty
`If(CostWasPaid, [actions])` continuation. Each pair lowers to a fresh IfPaid
with separate costs/body. Nested and successive pairs keep separate results.
The existing Unless form remains supported. X and nonmana optional payments,
other conditions and escaped payment results remain errors. The paid form
applies in supported spell/activation/source entry/death/controller-upkeep
contexts. Inventory reports typed event operands and If condition/action roles.

All rules execution is reused: WheneverSpellCastTrigger, existing pure card
filters, pending trigger placement/APNAP and captured controller, IfPaid and
Costs, transactional mana payment, life/draw actions, source incarnation,
phasing and clone state. No engine service or RON source changed. No runtime
startup, Go card definition or DSL export changed. Removal gates remain closed;
this slice does not claim a fresh unused-DSL caller audit or cutover validation.

Gameplay evidence for Crystal Rod, Iron Star, Ivory Cup, Throne of Bone,
Wooden Sphere, Urza's Chalice and Sol'kanar the Swamp King is
`TestRONSpellCastLifeRealCards`. It covers both casters, matching/nonmatching
spells, timing, refusal/inability, mana sources, failed casts, copies,
countered spells, departure/return/control/phasing and cloned resolution.
Sol'kanar's swampwalk is also checked in TestRONSpellCastLifeFilterCombinations.
IndependentInstances covers multicolor casting and four independently paid
triggers across controllers. PaymentContexts covers zero costs, refusal,
nested/successive bodies, fresh factories, explicit typed payer/target and
unavailable colored payment without partial source use. SharedContexts covers
paid activations, entry and upkeep. Invalid and Inventory tests cover
rejected shapes/bindings and nested operand roles. The report gives the full
mapping. Compilation progress and gameplay evidence remain separate.

New tests first failed on unsupported WhenAPlayerCastsASpell and missing roles.
Fixtures were corrected for the engine's priority placement, active caster,
legal supported spells and explicit clone decisions. Clones normally use
SearchPlayer, whose payment choice can differ from the test player's choice;
the clone tests install an explicit choice wrapper around cloned player state.
The old upkeep invalid case now rejects an orphan second If after a valid paid
pair, with positive upkeep evidence for the newly supported first pair.
Exact inventory expectations were refreshed. No gameplay expectation was
weakened to hide a missing implementation. The existing targeted explicit life
action compiler gap remains out of scope; typed payer coverage uses DrawACard.

Estimated and actual gain agree: **seven names, 32 records**. Five colored
artifacts contribute six reprints each; Urza's Chalice and Sol'kanar one each.
Accepted records increase **949 to 981/2,445**; rejection falls **1,496 to
1,464**. Accepted unique names increase **285 to 292/986**, leaving **694
rejected unique names**. No formerly accepted record regresses. Twelve
remaining diagnostics change, exposing caster scopes, event references and
Rohgahh's permanent-list producer. Loader and full AST inventory diagnostics
match. Global engine_validated and behavior_verified counts remain null.

Local Oracle data in LEA/ATQ/LEG agrees with all seven selected full records.
No source defect was corrected; all 12 RON hashes match the vocabulary snapshot.
Gaea's Liege/Fastbond observations remain unresolved. No environment or
pre-existing failure blocks this slice.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/dsl/ron -count=1`, engine short tests, `make build` | PASS |
| Baseline strict loader / full inventory | Expected exit 1; 949 accepted, 1,496 rejected |
| New `go test ./pkg/mage/dsl/ron -run TestRONSpellCastLife -count=1` | Expected adapter/role failures before implementation; final PASS |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS, repeated after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | PASS, zero issues |
| `git diff --check` | PASS, repeated after lint |
| `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1; 981 accepted, 1,464 rejected; identical diagnostics |
| RON hashes and before/after diagnostics | All unchanged; zero accepted-record regression |

Next ranked candidates: typed spell-cast event references (Presence of the
Master and Nether Void, plus In the Eye of Chaos after variable payment),
fixed selected discard (Jalum Tome and Bazaar of Baghdad), then temporary
animation (Mishra's Factory/Jade Statue/Thelonite Druid). Reassess full records
and dependencies next run. No next slice was started. Complete collection
loading/gameplay, embedded production startup, native smoke tests, Go-card and
generator deletion, and unused-DSL cleanup remain open.

Automatic lint changes only reordered imports in the new test file. Reviewed
changes have no semantic effect. Focused tests and diff checks passed again
after lint; the final strict loader still rejects the remaining collection.
Only the slice files are staged for one descriptive commit. No required check
is waived and no environment failure blocks the commit.

## Run 2026-10-04: P5-FIXED-DISCARD

Status: completed one fixed selected discard slice and its simultaneous-move
engine prerequisite. Initial user edits to `.gitignore` and the execution design
remain intact and excluded from this slice. No unrelated untracked files were
present. This run stops after this slice.

The [fixed discard report](mtgish-ron-fixed-discard.md) records the complete-record
and nested-form audit, candidate comparison, signatures/contexts, dependencies,
engine reuse, rules references and card-to-test mapping. Candidates were fixed
selected discard (two complete unlocks, small adapters plus shared batch repair),
typed spell-cast references (two immediate unlocks; event spell/player bindings,
variable payment and additional filters for later cards), and temporary animation
(up to three, with type/layer/timing/duration/cost prerequisites). Fixed discard
has the smallest complete boundary for two credible full-card unlocks. Selection,
reuse boundary and acceptance cases were stated before editing. Reprints count
once; first-error frequency was not used as an unlock estimate.

Supported forms: symbol `DiscardACard`, `DiscardNumberCards(Integer(n))` with a
nonnegative integer, and either action inside `PlayerAction(player, action)`.
Existing spell/activation/trigger/paid-continuation action contexts apply.
Existing typed player refs and per-instance target bindings supply the recipient;
bare actions use the captured resolving controller. Dynamic amounts, costs,
saved results, malformed declarations and undeclared event players remain errors.
Inventory labels the amount operand as `value` in both action roles.

Lowering uses `mage.Discard(ref, mage.Fixed(n))`. The no-result typed action
already delegates to legacy DiscardCards. That executor now gathers selected
cards into `DiscardAction` proposals, then uses `discardActionsTogether` to
prepare replacements, move the cards and send events after all moves. This
repairs a CR 608.2f gap without a new RON zone or event system. It also avoids
choices for nonpositive amounts. No public engine signature changed. Result
option paths, random discards and discard costs are outside this new engine
contract. Transactional activation costs, draw services, stack controller capture,
phasing, clone isolation and replacement rechecks are reused.

Gameplay mapping: Jalum Tome and Bazaar of Baghdad ->
`TestRONFixedDiscardRealCards` and `TestRONFixedDiscardActivationRollback`.
These cover their complete records: metadata validation, costs, draw-before-choice,
newly drawn card identity, short hands, source departure/return/control/phasing
and cloned resolution. Bazaar additionally maps to
`TestRONFixedDiscardReplacementAndEvents`. Shared declarations/player targets,
fresh factories, retargeted spell copies, entry and upkeep map to Contexts;
zero/empty/continuation to ZeroEmptyAndContinuation; invalid shapes and roles
to Invalid and Inventory. Engine contracts map to
`TestSelectedDiscardBatchEvents` and
`TestSelectedDiscardBatchReplacementPreparation`. The report gives exact names
and boundaries. No global gameplay completion claim is made.

Tests first failed on missing RON adapters/roles and on engine events observing
partial hands/destinations. Test fixture core event names were corrected to
actual APIs. Existing exact inventory expectations were refreshed to measured
counts, then all tests passed. The first lint run reported unparam on the existing
private withDestination helper after a new test added another constant argument
caller. The synthetic replacement now copies its own proposal and sets its test
destination; no production API or lint rule was changed. Automatic formatting
only reordered test imports. No assertion was weakened for an implementation gap.

Estimated and actual gain agree: **two unique cards, two records**. Strict loader
increases **981 to 983/2,445**; rejection falls **1,464 to 1,462**. Accepted names
increase **292 to 294/986**, leaving **692 rejected names**. No previously accepted
record regresses. Only Recall's remaining diagnostic changes, to unsupported
fixed discard amount ValueX; its results/return actions remain blockers.
Strict loader and full inventory issues match. Global engine_validated and
behavior_verified measurements remain null. Two newly compiled cards now have
RON gameplay evidence for all printed rules.

Local Oracle data agrees with both selected records. No source defect was
corrected; all 12 source hashes still match the vocabulary snapshot, so no syntax
snapshot refresh is needed. Gaea's Liege/Fastbond observations remain unresolved.
The adapter and engine gaps were distinct; no environmental blocker occurred.

| Command | Result |
| --- | --- |
| Baseline RON tests, engine short tests, `make build` | PASS |
| Baseline strict loader and inventory | Expected exit 1; 981 accepted, 1,464 rejected |
| New engine/RON focused tests with `-count=1` | Expected semantic failures before implementation; final PASS |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS after refreshing exact inventory counts; repeated after lint |
| `go test ./pkg/mage/... -short -count=1` | First run found old inventory counts; rerun PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | First run found the new test caller's unparam warning; final result recorded below |
| `git diff --check` | PASS; repeated before commit |
| Final strict loader and inventory | Expected exit 1; 983 accepted, 1,462 rejected; issues identical |
| Source hashes and diagnostics comparison | All hashes unchanged; zero accepted regression |

Next ranked candidates: typed spell-cast references (Presence of the Master and
Nether Void, with In the Eye of Chaos/Invoke Prejudice downstream); temporary
animation (Mishra's Factory, Jade Statue, Thelonite Druid); shared activation
modifiers/timing, including Disrupting Scepter. Reassess full records and fresh
inventory before selecting. No next slice was started. Complete collection
loading/gameplay, embedded production startup, native smoke tests, Go-card and
generator deletion, and unused-DSL cleanup remain open. Removal gates remain
closed; no caller-removal audit is claimed.

Final lint passed with zero issues. Formatting changes were reviewed, and focused
engine/RON/CLI checks and diff checks passed again. Only the ten slice files are
included in one descriptive commit; the two initial user files remain unstaged.
No required check was waived.

## Run 2026-10-04: P6-SPELL-CAST-REFERENCES

Status: complete typed spell/caster event-reference slice for any-player cast
triggers. This run stops after this slice. The initial user edits to `.gitignore`
and `mtgish-ron-execution.md` are preserved and excluded from the commit.
No unrelated untracked files were present.

The [cast-reference report](mtgish-ron-spell-cast-references.md) records the
complete-record/nested audit, candidate comparison, exact supported contexts,
engine reuse, rules evidence and card-to-test mappings. Three candidates were
compared before editing: cast references (Presence of the Master/Nether Void,
two complete unlocks); temporary animation (Mishra's Factory directly, Jade
Statue/Thelonite Druid after timing/cost/duration prerequisites); own-turn
activation restrictions (Disrupting Scepter directly, five other audited names
with random/results/limits/permission prerequisites). Cast references provide
the largest credible immediate multi-card gain for the adapter effort. The report
lists distinct names, blockers, downstream uses and estimated effort. Selection,
contract, reuse boundary and acceptance cases were stated before editing.
Reprints count once. First-error frequency was not used as an unlock estimate.

Supported declaration: `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, filter), actions)`
with existing AnySpell/IsCardtype/IsColor/nonempty And/Or spell filters and
ActionList or a supported single-group Targeted declaration. Symbol
`Trigger_ThatSpell` reads the exact cast stack object for CounterSpell and
ControllerOfSpell. Symbol `Trigger_ThatPlayer` reads the captured caster for
existing draw/discard actions and fixed optional PlayerMayCost continuations.
Malformed, incompatible, out-of-scope and cross-rule reads remain errors.
Explicit event-player life actions, including those through ControllerOfSpell,
remain rejected because that old life adapter still uses You. Bare life actions
remain supported. Variable payments and other event families are separate gaps.

`castEventPlan` is an immutable lexical identity. `actionPlan.buildWithEvents`
uses a typed event map alongside fresh target maps. Each trigger factory creates
fresh BindSpell/BindPlayer declarations; spell/player plans read that event
identity. Target and event groups coexist. Lowering uses WhenEvent/SpellCast,
WithSpell/WithCaster and the same SpellCastMatchesCardFilters service as the old
adapter. FromSpell/FromPlayer, pending capture/placement and APNAP, Counter,
stack copy/clone, transactional optional payment, draw/discard and source lifecycle
all reuse engine services. No engine service or RON source changed. Relevant
rules: CR 603.2/603.3/603.5, 608.2c/d/g/k and 701.6a. No new permanent LKI,
replacement-ordering or simultaneous-operation contract is introduced.

Full records were read for all candidate names and all seven cast-event shapes,
including nested Mana Vortex under FromStack. Seven already accepted life cards
retain their gameplay matrix. Presence of the Master and Nether Void each have
one record and now compile with full-record RON gameplay evidence. Local
`data/LEG.json` agrees with both. In the Eye of Chaos still needs variable optional
payment; Invoke Prejudice needs filters/payment; Mana Vortex needs source-zone,
self-spell, sacrifice and additional event support. Citanul Druid and Verduran
Enchantress remain blocked by caster scopes and counters/optional drawing.
No source defect was corrected; all 12 RON hashes match the vocabulary snapshot.
The Gaea's Liege/Fastbond observations remain unresolved.

New tests first failed on the missing spell/player references after fixing an
incorrect test helper name. The second independent cast fixture was changed to
an instant because an artifact cannot be cast while triggers are on the stack.
The characteristic-change fixture uses the actual CloneFrom API. The old invalid
symbol CounterSpell case now tests the malformed tuple; positive cases verify
the newly supported symbol. Exact inventory counts were refreshed. No gameplay
assertion was weakened to conceal missing support.

| Card/context | Evidence |
| --- | --- |
| Presence of the Master, Nether Void | TestRONSpellCastReferenceRealCards: both casters, filters/timing, no event targets, refusal/inability/mana sources, failed casts, exact counter/owner graveyard, absent/uncounterable/copy/recast objects, type/controller changes, departure/return/control/phasing, copied trigger payer and independent clone outcomes |
| Synthetic factories/events | TestRONSpellCastReferenceIndependentEvents: two instances and two pending events retain separate caster and spell reads |
| Synthetic event plus target | TestRONSpellCastReferenceDeclarationAndTarget: engine producer validation and distinct target/event spell reads with caster draw |
| Synthetic controller change | TestRONSpellCastReferenceCasterAndCurrentController: event caster draws; changed current spell controller discards |
| Invalid scopes | TestRONSpellCastReferenceInvalid: absent/wrong/escaped refs and unsupported event-life contexts |
| Seven retained cast-life cards | TestRONSpellCastLife* regressions pass after typed event lowering |

Estimated and actual gain agree: **two unique names, two records**. Compiled
records increase **983 -> 985/2,445**; rejection falls **1,462 -> 1,460**.
Accepted names increase **294 -> 296/986**, leaving **690 rejected names**.
No accepted-record regression. Only In the Eye of Chaos's remaining diagnostic
changes, from an unbound caster to unsupported optional PayManaX. Full inventory
and strict-loader issues match. Largest remaining first diagnostics are
PermanentRuleEffect (110), AsPermanentEnters (90), ActivatedWithModifiers (61),
non-spell CastEffect (53), rule If (47), ChooseADamageSource (46). These counts
are not semantic unlock estimates. Compilation and gameplay remain separate;
global engine_validated and behavior_verified measurements remain null.

| Command | Result |
| --- | --- |
| Baseline RON/CLI tests, engine short tests, make build | PASS |
| Baseline strict loader/full inventory | Expected exit 1: 983 accepted, 1,462 rejected |
| New focused `go test ./pkg/mage/dsl/ron -run TestRONSpellCastReference -count=1` | Expected missing-reference failures first; final PASS |
| `go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | First complete run found old inventory counts; final PASS, repeated after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS, repeated after final code changes and lint |
| `make test` | PASS, repeated after final code changes and lint |
| `make build` | PASS, repeated after lint |
| `go vet ./...` | PASS, repeated after lint |
| `make lint` | PASS, zero issues |
| `git diff --check` | PASS, repeated before commit |
| Final `go run ./cmd/loadcards -json` and `-inventory -json` | Expected exit 1: 985 accepted, 1,460 rejected; issues identical |
| Source hashes and diagnostic comparison | All hashes unchanged; zero accepted regressions |

Automatic lint changes only reordered new-test imports and were reviewed.
No environment or pre-existing validation failure blocks this slice. Only the
slice's eleven files are committed: compiler.go, player_scope.go, spell_scope.go,
triggers.go, new spell_cast_reference_test.go, spell_cast_life_test.go,
inventory_test.go, dsl-reference.md, engine-rules-contracts.md, new cast-reference
report and this ledger. No engine implementation, runtime registry, Go definition,
RON asset or DSL export changed. Removal gates are closed; no unused-DSL caller
audit or startup cutover is claimed.

Next ranked candidates: temporary animation (Factory directly; Queen shares
set-P/T after its Other target-filter adapter; Statue/Druid after their other
prerequisites); shared activation modifiers with a fresh once-per-turn/timing
multi-card audit; variable optional mana payment (Chaos directly, Invoke after
filters). Reassess full records and inventory next run. Complete collection
loading/gameplay, production startup/native smoke tests, Go-card/generator deletion
and unused-DSL cleanup remain open. No second slice was started.

## Run 2026-10-04: P7-TEMPORARY-ANIMATION

Status: complete fixed temporary base/type/subtype and source-excluding target
slice. This run stops here. Initial user edits to `.gitignore` and the execution
design are preserved and excluded from the commit; there were no unrelated
untracked files. Removal gates remain closed.

The [temporary animation report](mtgish-ron-temporary-animation.md) records the
fresh candidate comparison, complete-record/nested inventory audit, exact
signatures and contexts, engine boundary, rules and card-to-test evidence.
Candidates: temporary layers (Factory and Queen, two immediate unique unlocks),
activation modifiers (Scepter directly; Library/Bats with additional independent
conditions/limits; other audited names have random/results/permission blockers),
and variable optional payment (Chaos directly; Invoke after filters/caster scope).
Temporary layers provide the largest credible immediate gain for a coherent
shared adapter/engine boundary. Selection and acceptance cases were stated before
editing. Reprints count once; first-error frequency was not an unlock estimate.

Supported signatures in existing end-of-turn CreatePermanentLayerEffectUntil:
SetPT(PT(integer, integer)), AddCardtype(Creature|Artifact|Land|Enchantment),
AddCreatureType(symbol), and SetCreatureType(symbol). Existing typed source,
host and local permanent target references supply subjects; modifications share
one Until installation. Permanent-source TargetPermanent filters accept
Other(ThisPermanent), alone or under And, lowered to TargetOtherThanSource.
Disjunction/other-reference exclusions remain errors. Dynamic ManualPT, next
upkeep/combat durations, collection animation and activation modifiers remain gaps.
Spell/activation/trigger/continuation builders reuse the same action path; fresh
instance target maps retain typed reads and identity.

New reusable engine API: AddCardType(PermanentRef, core.CardType)
ContinuousModification, using existing attribute-backed types and Until capture.
Creature addition supplies power/toughness identity. Combined type additions are
synchronized before subtype checks in the same captured effect. Existing
SetBasePowerToughness, ModifyPT, AddSubtypeFamily and SetSubtypeFamily supply
layer operations. ValidateSubtypeFamily(SubtypeFamily, ...string) error exposes
the existing validation; legacy constructors call it and keep their panic
contract. RON rejects invalid declarations before factories. No new replacement,
event batch, APNAP, general LKI or payment-result contract is introduced.
Transactional costs, target legality, source incarnation, phasing, mana solver,
stack copy/retarget, clone and cleanup services are reused.

Source defect separated from adapter/engine gaps: CR 205.1b and local
`data/ATQ.json`/`data/LEA.json` show that Factory/Statue replace earlier creature
types when becoming named artifact creatures. Their AddCreatureType records
incorrectly retained prior creature types. A prior-Human Factory regression
failed after the initial adapter implementation. All six identical uses were
corrected to explicit SetCreatureType (Factory ATQ; Statue 2ED/CED/CEI/LEA/LEB).
Addition and replacement retain distinct lowering; no global predicate change or
card-name branch was added. Statue still rejects its unsupported timing/duration.
The vocabulary snapshot now has refreshed hashes, renamed syntax and current
loader diagnostics. Known Gaea's Liege/Fastbond concerns remain unresolved.

Tests first failed on absent AddCardType and unsupported SetPT/Other. Test-only
controller, mana and sickness API names were corrected before confirming intended
failures. An old invalid-context test now rejects malformed Other(ThisPermanent())
and positive gameplay tests establish the new valid form. Inventory count
expectations were refreshed to measured results. No gameplay assertion was weakened.

| Card/context | Mapping and evidence |
| --- | --- |
| Mishra's Factory | TestRONTemporaryAnimationFactory; TestRONTemporaryAnimationFactoryBoostAndRepeat: complete record, mana solver/immediate mana, animation/payment, source return/absence/phasing/control, clone and cleanup, retained ability/summoning sickness, opponent Assembly-Worker boost, invalid target/unpaid rollback, prior creature subtype replacement, reanimation/base-before-bonus |
| Sorceress Queen | TestRONTemporaryAnimationQueen: complete record, self exclusion/tap rollback, opponent creature, base-before-bonuses/counters, flying retained, absent/noncreature/returned/phased target, source departure/return/control, cloned resolution/cleanup |
| Synthetic declarations/copies/inventory | TestRONTemporaryAnimationDeclarations; TestRONTemporaryAnimationCopyAndInventory: fresh typed bindings, declaration rejection, distinct retargeted copy/original, new operand roles |
| Engine fixtures | TestTemporaryCardTypeCapture; TestTemporaryCardTypeKinds; TestTemporaryCardTypeDefinitionValidation: four supported types, combined layer group, retained characteristics, identity/absence/phasing/control, clone expiry, invalid types/undeclared reads |

Estimated and actual gain agree: **two unique names, three records** (Queen has
two reprints). Strict loader **985 -> 988/2,445 compiled**; rejection
**1,460 -> 1,457**. Names **296 -> 298 accepted**, **690 -> 688 rejected**.
No accepted record regression. Halfdane now reports UntilNextUpkeep; ManualPT
is also unsupported. Fresh full inventory and strict-loader issues match exactly.
Inventory has 72,938 nodes and 8,528 signatures; unknown-role occurrences decrease
19,421 -> 19,408 through fixed_pt/card_type/creature_subtype operand reporting.
Global engine_validated and behavior_verified remain null. Compilation and
complete collection gameplay coverage remain separate; two additional unique
cards have gameplay evidence for all printed rules in this slice.

| Command | Result |
| --- | --- |
| Baseline RON/CLI tests, engine short tests and make build | PASS |
| Baseline strict loader/full inventory | Expected exit 1: 985 compiled, 1,460 rejected |
| New focused engine/RON tests with -count=1 | Intended failures before implementation; final PASS |
| Prior-Human Factory source-defect regression | Expected failure with additive source; PASS with explicit replacement |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS after refreshing old context/count tests; repeated after lint/source correction |
| go test ./pkg/mage/... -short -count=1 | PASS; repeated after final source/tests and lint |
| make test | PASS; repeated after final source/tests and lint |
| make build | PASS; repeated after final source/tests and lint |
| go vet ./... | PASS; repeated after final source/tests and lint |
| go test ./... -short -count=1 | PASS after final code and source correction |
| make lint | First run found test variable copy shadowing a builtin; corrected; final PASS, zero issues |
| git diff --check | PASS; repeated before commit |
| Final strict loader and full inventory | Expected exit 1: 988 compiled, 1,457 rejected; issues identical |
| Refreshed vocabulary hashes and diagnostics | Match final source and loader; only six source constructor renames |

Automatic lint changes only reordered new test imports and were reviewed.
No environment or pre-existing failure blocks the commit. Only slice files are
staged: engine continuous/type/subtype changes and tests; RON adapters/inventory
and tests; six corrected assets; API/rules docs; refreshed vocabulary; the slice
report and ledger. Production registry/startup, Go definitions/generators and DSL
exports retain current callers. No deletion audit or runtime cutover is claimed.

Ranked next candidates: (1) shared activation modifiers with a fresh timing/limit
multi-card comparison; (2) collection temporary modifications, including Druid
after sacrifice choice and other mass boost cards; (3) variable optional payment
(Chaos directly, Invoke after filters). Reassess fresh complete records next run.
No second slice was started. Complete loading/gameplay, production startup/native
smoke tests, Go-card/generator deletion and unused-DSL cleanup remain open.

## Run 2026-10-04: P7-TEMPORARY-COLLECTION

Status: complete fixed temporary collection P/T slice. This run stops here.
The initial user edits to `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from the commit. No unrelated untracked files were present.

The [collection report](mtgish-ron-temporary-collection.md) contains the fresh
candidate comparison, exact supported signatures/contexts, complete-record and
nested audit, dependencies, engine reuse, rules evidence and card-to-test mappings.
Before editing, three candidates were compared: collection P/T (eight complete
unlocks); own-turn/upkeep activation timing (Scepter/Priest, two complete unlocks,
with other conditions/limits separate); variable optional mana payment (Chaos,
one complete unlock, Invoke still needs filters). The collection adapter has the
largest credible gain per effort. Selection, contract, reuse boundary and
acceptance cases were stated before editing. Reprints count once.

All nine collection context/shape signatures and the complete records of all
20 distinct collection names were read. Newly supported names are Army of Allah,
Piety, Bone Flute, Holy Light, Marsh Gas, Morale, Hell Swarm and Shield Wall.
Other affected names and their independent blockers appear in the report.
Subsequent complete timing-record inspection confirms the next candidate's
entry/counter/result/cost/event dependencies; no second slice was implemented.

Exact shape: CreateEachPermanentLayerEffectUntil(filter, nonempty list of
AdjustPT(integer_literal, integer_literal), UntilEndOfTurn). Existing spell,
activation, trigger and supported payment-continuation action builders share it.
Generic permanent filters and ControlledByAPlayer(SinglePlayer(You)) alone or
under conjunction reuse compileStaticPermanentFilter's immutable set plan.
Source exclusion and controller disjunction remain rejected in this context.
Other collection modifications, target collections, dynamic values, animation
and durations remain gaps. The compiler stores only filter/integer plans and
creates fresh modifications/Until per build. No new result/target binding is
introduced; surrounding bindings retain their existing isolation.

Engine reuse: MatchingPermanents/ControlledPermanents, ModifyEachPT, Fixed,
Until(EndOfTurn), captured incarnations, layer 7c, resolving controller and source
capture, source-independent duration, phasing, stack copies, cloned games,
transactional casting/activation and state-based actions. No engine change,
replacement ordering, LKI query or simultaneous zone-operation service is added.
Relevant local rules: CR 611.2a/c and 613.4b/c, with existing identity/phasing
contracts CR 400.7 and 702.26b/d. API/rules docs describe only this established
adapter and retained capture behavior.

Real gameplay/invalid tests were written first. After test-only API corrections,
all eight real records failed on unsupported CreateEachPermanentLayerEffectUntil.
After lowering, focused tests pass. The optional continuation fixture first
used BasePlayer, which declines; the existing accepting payment fixture now
supplies its required decision. No gameplay assertion was weakened. Automatic
lint changes only reordered imports and were reviewed.

| Card/context | Test and evidence |
| --- | --- |
| All eight newly supported names | TestRONTemporaryCollectionRealCards: complete records, engine validation, payment, resolution timing, each filter, before/after-resolution entries, phased exclusion/cycles, controller/combat membership retention, incarnation return, source departure, clones/expiry and graveyard |
| Army of Allah, Morale | TestRONTemporaryCollectionOpponentCombat: opponent's attacker receives/retains bonus on their turn |
| Shield Wall | TestRONTemporaryCollectionCopyController: copy's controller, no targets, phased exclusion, original/copy graveyard lifecycle |
| Bone Flute | TestRONTemporaryCollectionBoneFluteRollback: insufficient mana rollback; successful tap/two-mana cost; tapped source rejection |
| Holy Light | TestRONTemporaryCollectionHolyLightDeaths: both players' nonwhite creatures die by state-based actions; white creature survives unchanged |
| Synthetic shared contexts | TestRONTemporaryCollectionSharedContexts: fresh factories, engine validation, activation/entry-trigger/paid continuation, combined adjustments, resolving controller after source control change, inventory roles |
| Invalid declarations | TestRONTemporaryCollectionInvalid: malformed/empty modifications, wrong value forms, unsupported modification/duration, unbound collection/player, empty filter, disjunction/source exclusion |

Estimated and actual gain agree: **eight unique names, eight records**.
Strict loader **988 -> 996/2,445 compiled**, **1,457 -> 1,449 rejected**.
Names **298 -> 306 accepted**, **688 -> 680 rejected**. No accepted-record
regression. Inventory and strict-loader issues match. Nodes/signatures remain
72,938/8,528; unknown roles decrease 19,408 -> 19,278. Largest diagnostics:
PermanentRuleEffect (110), AsPermanentEnters (90), ActivatedWithModifiers (61),
non-spell CastEffect (53), If rule (47), ChooseADamageSource (46), Targeted_Modal
(42), HostController upkeep reference (35). First errors are not unlock counts.
Global engine_validated and behavior_verified remain null. Eight additional
unique cards have real RON gameplay evidence for all rules in their records;
compilation and complete collection gameplay remain separate.

No source-data defect was corrected. All 12 asset hashes still match the
vocabulary snapshot, so no source snapshot refresh is required. Known
Gaea's Liege/Fastbond concerns remain unresolved. No environment or pre-existing
failure blocks this slice.

| Command | Result |
| --- | --- |
| Baseline RON/CLI tests, engine short tests, make build | PASS |
| Baseline strict/full-inventory loaders | Expected exit 1: 988 compiled, 1,457 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONTemporaryCollection -count=1 | Intended missing-adapter failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS after count refresh; repeated after lint |
| go test ./pkg/mage/... -short -count=1 | PASS; repeated after lint |
| make test | PASS; repeated after lint |
| make build | PASS; repeated after lint |
| go vet ./... | PASS; repeated after lint |
| make lint | PASS, zero issues; reviewed import formatting |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and -inventory -json | Expected exit 1: 996 compiled, 1,449 rejected; issues identical |
| Source hash and accepted-record comparisons | All hashes unchanged; zero accepted regressions |

Only nine slice files are committed: compiler.go, inventory.go, inventory_test.go,
new temporary_collection.go and temporary_collection_test.go; dsl-reference.md,
engine-rules-contracts.md, new collection report and this ledger. Production
registry, Go definitions/generators and DSL exports retain their callers.
Removal gates remain closed; no DSL caller deletion audit or runtime cutover is
claimed.

Ranked next candidates: (1) own-turn/upkeep activation timing, two direct unlocks;
(2) temporary collection keyword/color changes with fresh target/combat-filter
prerequisite analysis; (3) variable optional mana payment (Chaos directly, Invoke
after filters). Reassess fresh inventory and full records next run. Complete
loading/gameplay, production startup/native validation, Go-card/generator removal
and unused-DSL cleanup remain open. This run begins no second slice.

## Run 2026-10-04: P4-ACTIVATION-TIMING

Status: complete own-turn/upkeep activation timing slice. This run stops here.
Initial user edits to `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from the commit. There were no unrelated untracked files.

The [activation timing report](mtgish-ron-activation-timing.md) contains the
candidate comparison, complete-record and nested audit, exact signatures,
engine reuse, rules evidence, card-to-test mappings and remaining dependencies.
Before editing, three candidates were compared: own-turn/upkeep timing (Scepter
and Priest, two complete unlocks); collection keyword grants (Slug, Skirmishers,
Flats and Part Water, zero immediate unlocks due to event/loop/filter/target-set
prerequisites); variable optional mana payment (Chaos directly, one unlock;
Invoke needs cast/filter support). Timing supplies the largest credible direct
multi-card gain for the effort. Selection, scope, reuse boundary and acceptance
cases were stated before editing. Reprints count once.

All distinct complete ActivatedWithModifiers records and nested forms were read.
Exact supported permanent rule is ActivatedWithModifiers(cost, actions,
ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn|IsTheirUpkeep))). The outer
arities are 3/1/2; player and filter operands are symbols. Existing supported
cost/action shapes are reused. compileActivatedRule now builds both this rule
and bare Activated with fresh costs, actions, targets and abilities per factory.
compileActivationTiming stores immutable ActionOptions. IsTheirTurn uses
WithYourTurnOnly; upkeep adds a pure WithActivationCondition step check. You
is the current activating controller. Both checks run before payment through
the engine; pending resolution does not repeat them. Other modifiers, limits,
combinations, players, conditions, spell contexts, malformed shapes, invalid
costs and unbound references remain declaration errors. Inventory supplies
operand roles without making support claims.

Engine reuse: NewAction/WithCost, timing/condition gates, transactional
activation and rollback, typed player/permanent targets, source/target capture,
incarnations, phasing, copies/retargeting, clones, mana solver, layers, discard
decisions and cleanup. No engine implementation, public DSL API, replacement
order, simultaneous operation or LKI query contract is added. Local rules read:
CR 117.1b/d, 602.1a/b, 602.2a/b, 602.5/5a and continuous-effect duration,
with existing identity/phasing/capture documentation retained.

Real gameplay and invalid declaration tests were written first. After a
test-only DrawStep -> Draw API correction, the real records failed on absent
ActivatedWithModifiers support. Final focused tests pass. A later test-only
pool getter was corrected to TotalMana. No gameplay expectation was weakened.
Automatic lint changes were reviewed and only reordered new-file imports.

| Card/context | Evidence |
| --- | --- |
| Disrupting Scepter | TestRONActivationTimingLegality; TestRONActivationTimingScepter: complete record/definition validation, current controller versus owner, own-turn combat/draw versus opponent turn, costs/tap/rollback, self/opponent discard decision and timing, source departure/return/control/phasing, pending clones |
| Svyelunite Priest | TestRONActivationTimingLegality; TestRONActivationTimingPriest; TestRONActivationTimingInstancesAndCopy: complete record/validation, own versus opponent upkeep and wrong step, sickness/controller/payment/tap rejection, self/opponent target, shroud with retained characteristics, source departure/return/control/phasing, target legality/return/phasing/control/noncreature changes, fresh instances, retargeted copy/original, clones and cleanup |
| Invalid/shared declarations | TestRONActivationTimingInvalid; TestRONActivationTimingRepeatAndInventory: arity/symbol/player/filter/modifier/cost/context/binding rejection, repeat activations without an implicit limit, resolution after timing changes, exact inventory roles |
| Mana context | TestRONActivationTimingManaPlanning: both timing shapes, pure affordability queries, correct solver restrictions, immediate mana resolution |

Estimate and actual gain agree: **two unique names, seven records** (six Scepter
reprints; one Priest). Strict loader **996 -> 1,003/2,445 compiled**;
**1,449 -> 1,442 rejected**. Names **306 -> 308 accepted**, **680 -> 678
rejected**. No accepted record regressed. Full inventory and strict-loader
issues match exactly. Nodes/signatures remain 72,938/8,528; unknown-role
occurrences fall 19,278 -> 17,943 through operand reporting, including unsupported
modifier contexts. Global engine_validated/behavior_verified remain null.
Two additional unique names have complete-record gameplay evidence; compilation
and full collection gameplay remain separate measurements.

Remaining largest first diagnostics: PermanentRuleEffect (110), AsPermanentEnters
(90), non-spell CastEffect (53), If rule (47), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep reference (35). Unsupported activation
modifiers now report their specific shape/filter errors. No source-data defect
was corrected. All 12 asset hashes match the vocabulary snapshot; no RON change
requires a snapshot refresh. Gaea's Liege/Fastbond concerns remain unresolved.
No environment or pre-existing failure blocked the slice.

| Command | Result |
| --- | --- |
| Baseline RON/CLI tests, engine short tests, make build | PASS |
| Baseline strict loader and full inventory | Expected exit 1: 996 compiled, 1,449 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONActivationTiming -count=1 | Intended missing-adapter failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS; repeated after lint |
| go test ./pkg/mage/... -short -count=1 | PASS; repeated after lint |
| make test | PASS; repeated after lint |
| make build | PASS; repeated after lint |
| go vet ./... | PASS; repeated after lint |
| make lint | PASS, zero issues; import formatting reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and -inventory -json | Expected exit 1: 1,003 compiled, 1,442 rejected; identical issues |
| Source hash and accepted-record comparisons | All source hashes unchanged; zero accepted regressions |

Only nine slice files are staged for the descriptive commit: compiler.go,
inventory.go, inventory_test.go, new activation_timing.go and
activation_timing_test.go; dsl-reference.md, engine-rules-contracts.md,
new activation timing report and this ledger. Production registry/startup,
Go definitions/generators and DSL exports retain current callers. Removal gates
remain closed; no deletion audit or runtime cutover is claimed.

Ranked next candidates: (1) activation limits, Fire Drake/Vampire Bats directly
and prerequisites for Instill Energy/Ebon Praetor/Gate to Phyrexia;
(2) general activation hand-count/state conditions, Library directly, other
conditions needing fresh complete-record comparison; (3) variable optional mana
payment, Chaos directly and Invoke after cast/filter support. Reassess inventory
and multi-card priority next run. Complete loading/gameplay, startup/native
validation, Go-card/generator removal and unused-DSL cleanup remain open.
No second slice was started.

## Run 2026-10-04: P4-ACTIVATION-LIMITS

Status: completed fixed activation limits, modifier conjunctions and their
required engine isolation repair. This run stops after this slice. User edits
to `.gitignore` and `mtgish-ron-execution.md` are preserved and excluded.

The [activation limits report](mtgish-ron-activation-limits.md) records the
complete candidate comparison, supported signatures/contexts, full-record
nested audit, reuse boundary, rules evidence and card-to-test mapping.
Before editing, three candidates were compared:

| Candidate | Distinct names | Estimated complete unlocks | Other blockers | Downstream capability | Effort |
| --- | --- | ---: | --- | --- | --- |
| Limits/conjunctions | Fire Drake, Vampire Bats, Dream Coat, Instill Energy, Ebon Praetor, Gaea's Touch, Gate to Phyrexia | 2 | Choices/colors, host attack permission, counters/results, sorcery timing/hand placement, matching sacrifice cost | Independent limited abilities and timing composition for seven names | Small adapter, medium contract tests; engine isolation repair was discovered |
| Hand-count condition | Library of Alexandria | 1 | None for seven-card equality | Pure player-state conditions | Small adapter, medium tests |
| Variable optional mana | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke's color-sharing/ExceptFor spell filter | Captured spell mana-value payments for two cast triggers | Medium adapter, substantial payment/choice tests |

Limits won on the largest credible direct multi-card gain. Complete records,
not first-error frequency, supplied the estimates; identical reprints count
once. Selection, contract, reuse boundary and acceptance cases were stated
before editing. No RON source defect or environment failure was found.

Exact support: permanent-source ActivatedWithModifiers/3 with the symbol
ActivateOnlyOnceEachTurn, or ActivateNoMoreThanNumberTimesEachTurn/1 containing
Integer/1 of a positive integer literal. Nonempty And/1 lists compose these
limits and supported You/IsTheirTurn or IsTheirUpkeep timing, including nested
conjunctions. Multiple numbered maxima remain errors to prevent option overwrite.
Once-only and one numbered maximum may coexist. Unsupported contexts, malformed
shapes, unknown modifiers, dynamic/nonpositive maxima and unbound cost/action
references remain errors. Plans contain immutable options and construct fresh
costs, actions, targets and abilities per factory. No new result binding exists.
Inventory assigns activation_modifier, activation_limit_value and integer_literal
roles without claiming support for unsupported descendants.

The first real gameplay tests failed as intended on missing limit/And adapters.
After lowering, they exposed clone count sharing and returned-object count
retention. TestActivationLimitIsolation then failed independently on synthetic
engine cards, proving runtime action mutation also changed card declarations.
ActionDefinition.markActivatedInGame now obtains the mutable source, copies
runtime abilities and records successful payment on the matching action ID.
Both stack and mana activation use the service. Public APIs, granted-ability
trackers and current callers remain. Absent sources need no lasting runtime
count. Unlimited actions avoid the copy. This is a rules prerequisite, not a
card-name workaround or parallel RON execution service.

Engine reuse: WithOncePerTurn/WithMaxActivationsPerTurn, CanActivate,
transactional costs, shared mana planning, typed targets, incarnation-bound
This/Until/ModifyPT, control, phasing, stack copies, clones and final cleanup.
CR 602.1b/602.2/602.5b/c, 400.7 and 702.26 supply the relevant contracts.
No replacement-order, simultaneous-operation or LKI query service is introduced.
Local DRK/LEG Oracle data confirms the once/twice instructions. Source data was
not changed; all 12 hashes match the vocabulary snapshot, so no refresh is needed.
Known Gaea's Liege/Fastbond defects remain uncorrected.

| Card/context | Test mapping |
| --- | --- |
| Fire Drake, Vampire Bats, all rules | TestRONActivationLimitRealCards: full-record validation, metadata/flying, payment/limits, delayed source bonuses, control/phase/departure/return identity, pending clones, cleanup and opponent-turn reuse |
| Vampire Bats | TestRONActivationLimitInstancesCopiesAndClone: independent factories/permanents, cloned counts/effects and copy before a further permitted activation |
| Fire Drake | TestRONActivationLimitPhasedReset: phased rejection, phased cleanup reset and clone isolation |
| Timing/typed targets/costs | TestRONActivationLimitConjunctionAndRollback: both timing filters, invalid target/turn rejection, mana/tap rollback, untap does not reset limit, timing change before resolution |
| Mana/inventory | TestRONActivationLimitManaAndInventory: pure repeat queries, immediate mana, solver limit exclusion, clone/reset contracts, nested roles |
| Nested/combined declarations | TestRONActivationLimitNestedAndContexts: nested timing/limits, combined once-only and numeric maximum, instant/sorcery rejection |
| Invalid declarations | TestRONActivationLimitInvalid and retained timing tests: arity/kind/value/context/reference/modifier errors, duplicate maxima across nested conjunctions |
| Reusable engine | TestActivationLimitIsolation: maximum one/two, clone, return, control, phase, immutable declaration and independent resolution |

The unsupported targeted life fixture was corrected to an existing typed tap
action. Previously invalid once/conjunction declarations now have positive
behavior tests; remaining rejection expectations were retained. Automatic lint
changes only reordered imports and were reviewed.

Estimated and actual gain agree: **two unique names, two records**. Strict
loader **1,003 -> 1,005/2,445 compiled**, **1,442 -> 1,440 rejected**.
Accepted names **308 -> 310**, rejected names **678 -> 676**. No accepted record
regressed. Strict/full-inventory diagnostics match exactly. Nodes/signatures
remain 72,938/8,528; unknown-role occurrences fall 17,943 -> 17,852.
Global engine_validated/behavior_verified remain null; two additional cards
have complete-record RON gameplay evidence. Compilation and collection-wide
behavior coverage remain separate.

Largest remaining first diagnostics: PermanentRuleEffect (110), AsPermanentEnters
(90), non-spell CastEffect (53), If rule (47), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep reference (35), PutCounters (29).
Gate to Phyrexia now reaches SacrificeAPermanent; the other dependent names
remain blocked as recorded in the report. None is counted as fully supported.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron -count=1; make build | PASS |
| Baseline strict/full-inventory loader | Expected exit 1: 1,003 compiled, 1,442 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONActivationLimit -count=1 | Expected missing-adapter failures first; final PASS |
| go test ./pkg/mage -run '^TestActivationLimitIsolation$' -count=1 | Expected engine isolation failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS |
| go test ./pkg/mage/... -short -count=1 | PASS, repeated after lint |
| make test | PASS, repeated after lint |
| make build | PASS, repeated after lint |
| go vet ./... | PASS, repeated after lint |
| make lint | PASS, zero issues; import formatting reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and -inventory -json | Expected exit 1: 1,005 compiled, 1,440 rejected; identical issues |
| Accepted-record comparison/source hashes | Zero regressions; all hashes unchanged |

This slice changes activated.go, new activation_limit_isolation_test.go,
activation_timing.go, new activation_limit_test.go, activation_timing_test.go,
inventory.go/inventory_test.go, dsl-reference.md, engine-rules-contracts.md,
new activation limits report and this ledger. Production registry/startup,
Go-card definitions/generators and DSL exports keep their callers. Removal
gates remain closed; no unused-DSL deletion or runtime cutover is claimed.

Ranked next candidates to reassess: (1) shared fixed counter actions and their
cost/result prerequisites, enabling the five Mana Batteries, Dwarven Weaponsmith,
Ebon Praetor and related entry-counter cards; (2) matching sacrifice activation
costs, now a direct dependency for Gate to Phyrexia and other multi-card cost
work; (3) variable optional mana payment (In the Eye of Chaos directly, Invoke
Prejudice after filters). Library's single-card hand-count condition stays
behind useful multi-card work. These are suggestions requiring fresh full-record
unlock estimates next run. Complete RON gameplay/loading, application cutover,
native startup, Go-card/generator removal and unused-DSL cleanup remain open.

## Run 2026-10-04: P4-SACRIFICE-MATCHING

Status: completed matching-permanent activation sacrifice costs. This run stops
here. Initial user edits in `.gitignore` and `mtgish-ron-execution.md` remain
unchanged and excluded from the commit; no unrelated untracked files were present.

The [sacrifice costs report](mtgish-ron-sacrifice-costs.md) records the complete
candidate comparison, supported signatures/contexts, nested/full-record audit,
engine/API/rules reuse and card-to-test mappings. Before editing, three candidates
were compared: matching activation sacrifices (nine complete unique unlocks);
fixed counter placement (three estimated direct names, with variable counter
removal/results still blocking five Mana Batteries); variable optional mana
payment (In the Eye of Chaos directly; Invoke Prejudice still needs filters).
Matching costs won on credible direct gain for implementation/test effort.
Dependent names and exact blockers are listed in the report; first-error counts
were not used as full-card estimates. Reprints count once.

Exact support: permanent-source Activated/2 and ActivatedWithModifiers/3 costs
with SacrificeAPermanent/1 containing a supported compileFilter permanent filter.
Nested filter And/1 and Or/1 require nonempty lists at every level. Existing
cost conjunctions and supported timing/limit modifiers compose unchanged.
compileSacrificeCostFilter(Node) (mage.PermanentFilter, error) validates immutable
filter plans; compileCosts creates a fresh matching sacrifice cost per factory.
The engine's SacrificeMatchingIncludingSourceCost permits the matching source,
uses the activating controller's candidates and one locked choice, and moves
the paid permanent to its owner's graveyard. Source-relative filters, resolving
sacrifice actions, optional/must payments, additional casting costs and saved
paid-characteristic results remain declaration errors. Inventory now supplies
the permanent_filter operand role even beneath unsupported parent contexts.
No game-time declaration panic, name branch, fallback or parallel cost system
was added.

Engine reuse: existing total-cost transaction/choice locking, typed references,
source/target incarnation capture, LKI damage, phasing, copies, clones, timing,
limits, layers/cleanup, life/draw/regeneration/destruction/damage services and
explicit immediate mana activation. No reusable engine API or implementation
changed. The tap-source automatic planner continues to exclude sacrifice costs;
explicit Altar activation produces two usable colorless mana. Tests document
pure queries without choosing or paying a sacrifice. Local CR 602.1a/602.2/2b
and 701.21a establish payment, rollback and owner-graveyard semantics; existing
400.7/702.26 and source-instance contracts establish identity/phasing behavior.
No new replacement-order or simultaneous-operation contract is claimed.

| Card/context | Test mapping |
| --- | --- |
| Atog, Fallen Angel, Dark Heart of the Wood, Goblin Chirurgeon, Horror of Horrors, Gate to Phyrexia, Orcish Mechanics, Sage of Lat-Nam, Ashnod's Altar; all rules | TestRONSacrificeCostRealCards: complete RON/definition validation, source/candidate controller versus owner, one paid choice, graveyard ownership, regeneration cannot prevent sacrifice, tap cost, payment versus resolution, source departure/return/control/phasing, pending clones, bonus/cleanup, life/draw players, targets, shields/destruction/damage and immediate mana |
| Fallen Angel, Goblin Chirurgeon | TestRONSacrificeCostSourceAndTargets: source can pay, self-target sacrificed away, returned incarnation, target departure/return/phasing/type legality |
| Fallen Angel | TestRONSacrificeCostInstancesAndCopy: independent factories, original source capture, ability copy pays no new cost, clone resolution/cleanup isolation |
| Gate to Phyrexia | TestRONSacrificeCostGateTimingAndLimits: turn/step/target rejections pay nothing, own-upkeep activation and once-per-turn, no timing recheck at resolution |
| Orcish Mechanics | TestRONSacrificeCostDamageReplacement: departed source LKI and shared prevention pipeline |
| Ashnod's Altar | TestRONSacrificeCostManualManaAndQueries: explicit immediate mana, planner exclusion/purity, usable production, no reused paid creature and clone isolation |
| Shared costs/filters/inventory | TestRONSacrificeCostRollbackAndChoice; TestRONSacrificeCostCompositionAndInventory: mana/tap rollback, insufficient/double sacrifices, phased/opposing/wrong-type candidates, invalid choice, locked single choice, nested type/color/subtype filters and operand reporting |
| Invalid declarations | TestRONSacrificeCostInvalid: kind/arity/reference, nested empty filters, source-relative filter, unsupported action/optional-payment/result contexts |

The first real gameplay tests failed as intended on missing SacrificeAPermanent
support. Fixture fixes selected ATQ Ornithopter, used phased-inclusive existence
checks, accounted for source cards bounced to hand, and set owners before
removal. A new test's automatic-sacrifice-planning assumption was corrected after
reading the documented tap-source engine/API boundary; the final test proves
that boundary and explicit usable mana. No rule expectation was bypassed.

Estimated and actual gains agree: **nine unique names, ten records** (two Atog
reprints). Loader **1,005 -> 1,015/2,445 compiled**, **1,440 -> 1,430 rejected**.
Accepted unique names **310 -> 319**, rejected **676 -> 667**. Zero previously
accepted records regressed. Strict/full-inventory issues match exactly.
Nodes/signatures remain 72,938/8,528; unknown-role occurrences fall
17,852 -> 17,801. Global engine_validated/behavior_verified remain null.
Nine additional unique cards have full-record RON gameplay evidence; collection
compilation and collection-wide behavior coverage remain separate measurements.

Largest remaining first diagnostics: PermanentRuleEffect (110), AsPermanentEnters
(90), non-spell CastEffect (53), If rule (47), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep reference (35), PutCounters (31).
PutCounters increased because Dwarven Weaponsmith now reaches its next blocker.
No source-data defect changed. All 12 source hashes match the vocabulary snapshot;
no RON edit requires a refresh. Gaea's Liege/Fastbond concerns remain open.
No environment or pre-existing failure blocked this slice.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1; engine short tests; make build | PASS |
| Baseline strict and full-inventory loader | Expected exit 1: 1,005 compiled, 1,440 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONSacrificeCost -count=1 | Missing-adapter failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS; final formatting review recorded below |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and -inventory -json | Expected exit 1: 1,015 compiled, 1,430 rejected; identical issues |
| Accepted-record comparison/source hashes | Zero regressions; all source hashes unchanged |

Slice files: compiler.go, new sacrifice_cost.go and sacrifice_cost_test.go,
inventory.go/inventory_test.go, dsl-reference.md, engine-rules-contracts.md,
new sacrifice costs report and this ledger. Production startup, Go-card
constructors/generators and DSL exports retain their current callers; removal
gates remain closed. No unused-DSL removal audit or runtime cutover is claimed.

Ranked next candidates requiring fresh comparison: (1) fixed counter placement,
now also directly benefiting Dwarven Weaponsmith, plus Armor Thrull/Fungal Bloom/
Citanul Druid and the five Batteries after removal/results; (2) typed sacrifice
characteristic results for Diamond Valley/Life Chisel, and Priest of Yawgmoth
after dynamic mana; (3) variable optional mana payment, Chaos directly and Invoke
after spell filters. Full loading/gameplay, startup/native validation, Go-card
and generator removal and unused-DSL cleanup remain open. No second slice began.

Final lint review: only new-test import grouping changed; zero lint issues.
The RON/CLI package tests were repeated after lint and pass. Fresh final strict
and semantic-inventory loaders retain the recorded counts and identical issues;
all source hashes were checked against the vocabulary snapshot. Final diff
whitespace checks pass. All required native Go checks passed before commit.

## Run 2026-10-04: P5-COUNTER-PLACEMENT

Status: completed single-permanent resolving counter placement. This run stops
here. Initial user edits in `.gitignore` and `mtgish-ron-execution.md` are
preserved and excluded. No unrelated untracked files were present.

The [counter placement report](mtgish-ron-counter-placement.md) gives the complete
candidate comparison, exact signatures/contexts, 63-name full-record/nested audit,
engine/rules reuse, card mappings and remaining blockers. Before edits, compared:

| Candidate | Distinct names | Estimated complete unlocks | Other blockers | Downstream gain | Effort |
| --- | --- | ---: | --- | --- | --- |
| Single-permanent counters | Armor Thrull, Fungal Bloom, Dwarven Weaponsmith; 60 other names listed in the report | 3 | Citanul's Opponent event; Battery removal/results; other values/entry/choices/layers | Five Batteries and spore-counter prerequisites; source/host/target action reuse | Small adapter, medium tests; existing unusual counter kinds |
| Sacrifice characteristics | Diamond Valley, Life Chisel, Priest of Yawgmoth; Homarid Spawning Bed downstream | 2 | Priest dynamic mana; Bed tokens/results | Typed independent payment characteristics | Medium adapter, substantial LKI/rollback/copy tests |
| Variable optional mana | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke's Opponent/color-sharing spell filters | Captured spell mana-value payment for two triggers | Medium adapter, substantial payment/choice tests |

Counters won on credible direct multi-card gain for the effort. Reprints count
once. The prior Citanul estimate was corrected after reading its complete record:
placement alone does not supply the Opponent cast filter. Selection, contract,
reuse boundary and acceptance cases were stated before edits. No missing engine
contract, source defect or environment failure blocked this slice.

Exact support: resolving PutCounters/1 with exactly one list instruction,
ACounterOfTypeOnPermanent/2 or NumberCountersOfTypeOnPermanent/3. Counter kinds
are 25 explicit named permanent symbols or ten existing literal PTCounter pairs
listed in the report. Amounts use Integer/1 or already-bound ValueX. Typed
ThisPermanent, Aura HostPermanent and local Ref_TargetPermanent use existing
scope checks. Actions compose through existing supported spells, activations,
triggers and optional-payment continuations. Empty/multiple lists, unknown kinds,
player instructions, other values/refs, costs/static/entry contexts and removal
remain errors. No new bindings or permissive registry path exist.

Private adapters: compileCounterPlacement(Node, scope) (effectFactory, error),
compilePermanentCounterType(Node) (core.CounterType, error). Fresh factories
lower to mage.AddCountersTo with typed instance subjects and ValueAtResolution.
Reusable rules stay in the engine. Inventory supplies counter_instruction,
permanent_counter_type, value/reference and integer_literal roles; syntax still
has unverified support. No DSL export or engine implementation changed.

Engine reuse: source-incarnation and attachment LKI, targets, cost transactions,
copy/clone state, counter P/T layers, cleanup and AddCountersAction replacement
ordering/rechecks. CR 122.1/1a/2, 400.7, 602.2 and 616.1/1e/1f support these
contracts. Departed/phased/returned subjects receive no pending placement;
phase cycles retain incarnation. Counters survive cleanup. No simultaneous
multi-object, entry or removal contract is introduced.

| Card/context | Test mapping |
| --- | --- |
| Armor Thrull, Fungal Bloom, Dwarven Weaponsmith (ATQ and 3ED), all rules | TestRONCounterPlacementRealCards: validation, payment/timing, delayed placement, opposing targets, source/target identity, phasing/control/type legality, clones, P/T and cleanup |
| Armor Thrull, Dwarven Weaponsmith and scope boundaries | TestRONCounterPlacementPaymentAndContexts: unpayable tap/sacrifice, missing artifact rollback, returned self-target, cost/static/cross-rule/source rejection |
| Source, host, upkeep trigger | TestRONCounterPlacementSourceAndHost: amounts, identity, phase cycle, controller, resolution attachment and departed Aura LKI/returned objects, pending clone |
| Copies/replacements | TestRONCounterPlacementCopiesAndReplacement; TestRONCounterPlacementReplacementOrder: independent factories/targets, copied controller, doublers/rechecks, clone mutation isolation, affected-player noncommutative ordering, prevention permits next action |
| Values and P/T | TestRONCounterPlacementAmountsAndRollback; TestRONCounterPlacementSignedPT: nonpositive/fixed/X, rollback, copied X and unusual signed coordinates |
| Declarations/inventory | TestRONCounterPlacementInvalid; TestRONCounterPlacementInventory: malformed shapes, lists, kinds/coordinates, values, refs/contexts and nested roles |

The first gameplay tests failed on missing PutCounters support. After lowering,
a noncreature Fungus test expectation was corrected: its creature subtype no
longer qualifies. Fixture compile errors were corrected to core event names and
actual counter count types. Returned sacrificed fixtures remove the original
card from its graveyard before entry. Inventory assertions now use fresh counts;
no gameplay or invalid-declaration requirement was bypassed.

Estimated and actual gain agree: **three unique names, four records**. Loader
**1,015 -> 1,019/2,445 compiled**, **1,430 -> 1,426 rejected**. Accepted names
**319 -> 322**, rejected **667 -> 664**. Record-level accepted regression audit:
zero. Strict/full-inventory diagnostics match exactly. Nodes/signatures stay
72,938/8,528; unknown-role occurrences fall 17,801 -> 17,378. Global
engine_validated/behavior_verified stay null; three additional complete cards
have RON gameplay evidence. Compilation and global behavior coverage are separate.

Largest remaining first diagnostics: PermanentRuleEffect (110), AsPermanentEnters
(90), instant/sorcery CastEffect (53), If rule (48), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep (35), TriggerI (28), ControlledByAPlayer
filter (28), TargetGraveyardCard (26). The five Batteries now reach removal costs.
Citanul still reaches Opponent cast filtering. No dependent card is counted as
complete. No RON source changed; all 12 hashes match the vocabulary snapshot.
Soul Exchange's +2/+2 counter requires a missing engine counter kind, not a source
correction. Gaea's Liege/Fastbond concerns remain open.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1; make build | PASS |
| Baseline strict and semantic inventory loaders | Expected exit 1: 1,015 compiled, 1,430 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONCounterPlacement -count=1 | Expected missing-adapter failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS, repeated after lint and fixture refinement |
| go test ./pkg/mage/... -short -count=1 | PASS, repeated after lint |
| make test | PASS, repeated after lint |
| make build | PASS, repeated after lint |
| go vet ./... | PASS, repeated after lint |
| make lint | PASS, zero issues; formatting and automatic changes reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and -inventory -json | Expected exit 1: 1,019 compiled, 1,426 rejected; identical issues |
| Accepted-record audit/source hashes | Zero regressions; all 12 hashes unchanged |

Slice files: compiler.go, new counter_placement.go/counter_placement_test.go,
inventory.go/inventory_test.go, dsl-reference.md, engine-rules-contracts.md,
new counter placement report and this ledger. Automatic lint passes introduced
no semantic changes. Current startup, Go-card constructors/generators and DSL
exports retain callers. Removal gates remain closed; no deletion, unused-DSL
audit or runtime cutover is claimed. No required check was blocked.

Ranked next candidates requiring fresh comparison: (1) fixed counter-removal
activation costs, directly Feral Thallid/Thorn Thallid and spore-token prerequisites;
(2) typed sacrifice toughness, directly Diamond Valley/Life Chisel with payment
results reused for Priest/Bed; (3) any-number counter removal and typed payment
amount/dynamic mana for five Batteries. Complete loading/gameplay, runtime/native
validation, Go-card/generator removal and unused-DSL cleanup remain open.
No second slice began.

## Run 2026-10-04: P4-COUNTER-COST

Status: completed fixed source-counter activation costs. One slice only. Initial
user edits in `.gitignore` and `mtgish-ron-execution.md` are preserved and excluded
from this slice. The starting tree had no unrelated untracked files.

The [counter costs report](mtgish-ron-counter-costs.md) records the full candidate
comparison, complete-record/nested audit, exact support and card/test mappings.
The pre-edit comparison was:

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers/downstream capabilities | Effort |
| --- | --- | ---: | --- | --- |
| Fixed source-counter costs | Feral Thallid, Thorn Thallid, Elvish Farmer, Thallid, Thallid Devourer, Spore Flower, Icatian Javelineers, Triskelion, Delif's Cube, Osai Vultures, Scavenging Ghoul, Rasputin Dreamweaver, Life Matrix | 2 | Tokens, prevention, entry, events/history and grants block the other names; direct prerequisite for three token-producing cards | Small adapter, medium tests |
| Sacrifice characteristics | Diamond Valley, Life Chisel, Priest of Yawgmoth, Homarid Spawning Bed | 2 | Priest needs repeated dynamic mana; Bed needs tokens; typed payment results serve all four | Medium adapter, substantial LKI/payment/copy tests |
| Variable optional mana | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke also needs Opponent/color-sharing spell filters; event mana-value payment serves both triggers | Medium adapter, substantial payment/event tests |

Fixed removal won on the smallest credible effort for two complete cards and
concrete multi-card downstream benefit. Reprints count once. First-error counts
were not treated as unlock estimates. Contract, reuse boundary and acceptance
cases were stated before editing. No missing engine prerequisite was found.

Exact support: `RemoveCounters/1` in permanent-source `Activated/2` and
`ActivatedWithModifiers/3`, including existing cost conjunctions and supported
modifiers. Its single-instruction list accepts `ACounterOfTypeFromPermanent/2`
or `NumberCountersOfTypeFromPermanent/3`; amount is `Integer/1` with a literal
nonnegative number, counter kind uses `compilePermanentCounterType`, and source
must be symbol `ThisPermanent`. Private
`compileCounterRemovalCost(Node) (costFactory, error)` returns a factory that
creates a fresh `mage.RemoveCountersCost` per card instance. Invalid declarations
are rejected before factory execution. Other references, variable/any/all amounts,
removal actions, entry/static contexts and optional counter payments remain gaps.
No result binding, global state, name branch, fallback or partial registry exists.
Inventory now supplies removal-instruction and typed operand roles under known
cost contexts, without claiming execution for unsupported parent contexts.

Engine reuse: total-cost transactions, locked source incarnation, permission,
timing/limits, clones, typed targets, ability copies, LKI damage, regeneration,
phasing and shared damage prevention. No engine API/service changed. CR 602.1a,
602.2/2b, 122.1/1a/2, 400.7 and existing identity/phasing contracts apply. No new
simultaneous-operation or counter-removal replacement contract is established.

| Card/context | Test mapping |
| --- | --- |
| Feral Thallid and Thorn Thallid, complete records | TestRONCounterCostRealCards: definition validation, own/opponent upkeep, pending counter placement, three-counter payment, insufficient payment, source departure/return/control/phasing/phase cycle, clones, regeneration/destruction or damage |
| Thorn Thallid | TestRONCounterCostTargetsCopiesAndInstances: permission and target rejection without costs, phased source, independent sources, cost-free copy, departed source LKI, target return/phasing/type legality, prevention, clone isolation |
| Shared transactions | TestRONCounterCostRollbackAndComposition: mana/tap order, duplicate costs, rollback, exact payment and later resolution |
| Shared factories/clones | TestRONCounterCostFactoryIsolationAndManaRollback: same compiled factory creates independent sources, later mana failure restores counter/tap payment, clones pay and resolve independently |
| Forms/timing | TestRONCounterCostFormsAndTiming: one/zero/fixed P/T counters, validation, upkeep and once-per-turn, no resolution timing recheck |
| Declarations/reporting | TestRONCounterCostInvalid; TestRONCounterCostInventory: shape/arity/value/type/reference/context rejection and typed nested roles |

TDD: after correcting test fixture compile errors for counter types and PhaseIn,
new gameplay/form/payment tests failed on `unsupported cost (RemoveCounters)`;
inventory failed on missing removal roles. After implementation, real-card tests
were corrected to use activation index 1 (index 0 is the upkeep trigger), explicit
event/stack placement for unresolved upkeep, and a valid artifact timing fixture.
No gameplay expectation was weakened. Broad checks then found two stale inventory
count assertions; they now assert fresh measured counts and retain all registry
and gameplay checks.

Estimated and actual gain agree: **two unique names, two records**. Loader
**1,019 -> 1,021/2,445 compiled**, **1,426 -> 1,424 rejected**. Accepted names
**322 -> 324**, rejected **664 -> 662**. Zero previously accepted records regressed.
Strict and semantic-inventory diagnostics match exactly. Nodes/signatures stay
72,938/8,528; unknown-role occurrences fall **17,378 -> 17,251**. Global
engine_validated/behavior_verified remain null. Two complete real cards gain RON
behavior evidence; global gameplay completion is not inferred from compilation.

Largest remaining first diagnostics: PermanentRuleEffect (110), AsPermanentEnters
(90), instant/sorcery CastEffect (53), If rule (48), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep (35), TriggerI (28), ControlledByAPlayer
filter (28), TargetGraveyardCard (26). Source hashes match the vocabulary snapshot;
no source changed, and no vocabulary source refresh is needed. No source defect
was corrected. Gaea's Liege/Fastbond concerns remain open. No environment or
pre-existing failure blocked the work.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1; make build | PASS |
| Baseline strict and inventory loaders | Expected exit 1: 1,019 compiled, 1,426 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONCounterCost -count=1 | Expected missing-adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS |
| go test ./pkg/mage/... -short -count=1 | Initially stale inventory assertions; final PASS after measured-count update |
| make test | Initially stale inventory assertions; final PASS after update |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS, zero issues; only new-test import grouping changed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and -inventory -json | Expected exit 1: 1,021 compiled, 1,424 rejected; identical issues |
| Accepted-record audit/source hashes | Zero regressions; all 12 source hashes unchanged |

Slice files: compiler.go; new counter_cost.go/counter_cost_test.go;
inventory.go/inventory_test.go; dsl-reference.md; engine-rules-contracts.md;
new counter costs report and this ledger. All required native checks pass.
Current startup, Go constructors/generators and DSL exports retain their callers.
Removal gates remain closed. No deletion, unused-DSL audit or runtime cutover is
claimed. No second slice began.

Ranked next candidates to reassess: (1) basic token creation, now directly
Thallid/Thallid Devourer/Elvish Farmer; (2) typed sacrifice toughness, directly
Diamond Valley/Life Chisel with results reused for Priest/Bed; (3) any-number
counter payment/results and dynamic mana for five Mana Batteries. Complete RON
loading/gameplay, runtime/native startup, Go-card/generator deletion and unused-DSL
cleanup remain unfinished.

## Run 2026-10-04: P5-TOKEN-CREATION

Status: completed one basic creature-token creation slice. Initial user changes
in `.gitignore` and `mtgish-ron-execution.md` are preserved and excluded from the
commit. No other unrelated untracked files were present.

The [token creation report](mtgish-ron-token-creation.md) records the complete
candidate comparison, exact signatures, nested/complete-record audit, dependencies,
engine reuse and card-to-test mappings. Fresh evidence compared basic creature
tokens (18 affected names, six estimated unlocks), sacrifice characteristics
(four names, two unlocks), and variable optional mana (two names, one unlock).
Token creation won on existing engine support and the largest credible gain for
the effort. Names and remaining blockers are listed in the report. The contract,
reuse boundary and acceptance cases were stated before implementation.

Exact supported resolving contexts: `CreateTokens/2` with a one-instruction list
and an empty modifier list; `TokenWithPT/6`, `NamedTokenWithPT/7`, and
`NumberTokens/2` wrapping either. Counts accept existing fixed integer or in-scope
announced spell X. Tokens have literal signed P/T, distinct colors/permanent
card types/creature subtypes/simple keywords and no supertypes. Unnamed tokens
use subtype names plus ` Token` (local CR 111.4); named tokens keep their explicit
name. Cost/static contexts, token copying/abilities/modifiers/supertypes/results,
and multiple instructions remain errors. Private adapters are
`compileTokenCreation(Node, scope) (effectFactory, error)` and
`compileCreatureToken(Node) (mage.TokenDefinition, error)`.

Reusable engine behavior uses immutable `DefineToken` declarations and
`CreateTokenCopies(You, ValueAtResolution(count), token)`. Per-factory actions
have no mutable chosen state in the shared declaration. Payments, target legality,
resolving controllers, source/event LKI, copy/X behavior, phasing, token lifecycle,
clone isolation and simultaneous token entry stay in their existing services.
No missing engine contract or engine implementation change was needed. No new
replacement-order operation is introduced. Relevant CR: 111.2–8, 400.7, 602.2,
608.2b. API/rules docs now record only these adapter contracts.

TDD: real-card, batch and copy tests first failed on `unsupported action
(CreateTokens)`; semantic inventory failed on missing token roles. Test fixture
API errors were corrected before the expected adapter failure was confirmed.
Lowering initially supplied nil optional token options, which engine validation
rejects; it now omits empty options. Boris's tap-cost fixture advances through
untap. Subtype rejection uses a known land subtype because the existing validator
permits custom creature subtype names. No test requirement was weakened.

New complete real-card gameplay mappings:

| Names | Tests |
| --- | --- |
| Thallid, Thallid Devourer, Elvish Farmer | TestRONTokenCreationRealCards: complete upkeep/counter-cost/token/sacrifice/life/bonus rules and cleanup |
| The Hive, Boris Devilboon | TestRONTokenCreationRealCards: definition validation, mana/tap costs, named token colors/types/keywords and resolution timing |
| Icatian Town | TestRONTokenCreationTownBatch: definition validation, casting/payment, four simultaneous entries, characteristics and graveyard |
| Shared contracts and Thallid | TestRONTokenCreationCopiesAndSource, TestRONTokenCreationCountsAndRollback, TestRONTokenCreationTriggerAndLegality: copied controllers/X, departure/return/phasing/control, clone isolation, atomic failed payments, trigger controller, all-illegal targets and token lifecycle |
| Invalid declarations/inventory | TestRONTokenCreationInvalid, TestRONTokenCreationDeclarationBoundaries, TestRONTokenCreationInventory: typed shapes, context/scope boundaries and reporting roles |

Estimated and actual gain agree: **six unique names, eleven records**. Loader
**1,021 -> 1,032/2,445 compiled**, **1,424 -> 1,413 rejected**. Accepted names
**324 -> 330**, rejected **662 -> 656**. The Hive accounts for six reprints;
the other newly accepted names have one record each. Zero accepted-record
regressions; strict and inventory issue arrays match. Nodes/signatures remain
72,938/8,528; unknown-role occurrences fall **17,251 -> 16,921**. Global
engine_validated/behavior_verified remain null. All six new unique cards have
RON gameplay and explicit definition-validation evidence. Compilation progress
is separate from collection-wide gameplay completion.

Largest remaining first diagnostics: PermanentRuleEffect (110), AsPermanentEnters
(90), instant/sorcery CastEffect (53), If rule (48), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep (35), TriggerI (28), ControlledByAPlayer
filter (28), TargetGraveyardCard (26). Token-dependent records still report their
remaining blockers; none is silently accepted. No source-data defect was corrected.
All 12 source hashes match the vocabulary snapshot, so no refresh is needed.
Gaea's Liege/Fastbond concerns remain open. No environment failure blocked work.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1; make build | PASS |
| Baseline strict and inventory loaders | Expected exit 1: 1,021 compiled, 1,424 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONTokenCreation -count=1 | Expected adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS; repeated after lint and explicit validation assertions |
| go test ./pkg/mage/... -short -count=1 | PASS; repeated after final test assertions |
| make test | PASS; repeated after final test assertions |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS, zero issues; automatic changes reviewed: new-test import grouping only |
| git diff --check | PASS |
| Final strict and inventory loaders | Expected exit 1: 1,032 compiled, 1,413 rejected; identical diagnostics |
| Accepted-record and source-hash audits | Zero regressions; all 12 hashes unchanged |

Slice files: compiler.go; new token_creation.go/token_creation_test.go;
inventory.go/inventory_test.go; dsl-reference.md; engine-rules-contracts.md;
new token creation report and this ledger. Current production startup, Go-card
constructors/generators and DSL exports retain their callers. Removal gates remain
closed; no deletion, unused-DSL removal audit or runtime cutover is claimed.

Ranked next candidates requiring fresh comparison: (1) typed sacrifice
characteristic results for Diamond Valley/Life Chisel, with mana-value results
also enabling Homarid Spawning Bed now that tokens work; (2) any-number counter
payments/results and dynamic mana for the five Mana Batteries; (3) delayed/step
trigger prerequisites for Breeding Pit, Rukh Egg and Hazezon Tamar. The next run
must reassess these against fresh inventory. Full RON loading/gameplay,
runtime/native startup, Go-card/generator deletion and DSL cleanup remain open.
No second slice began.

## Run 2026-10-04: P4-SACRIFICE-VALUES

Status: completed one saved-sacrifice-characteristic slice. User changes in
`.gitignore` and `mtgish-ron-execution.md` are preserved and excluded from the
commit. No unrelated untracked files were present.

The [sacrifice values report](mtgish-ron-sacrifice-values.md) records complete
candidate names, nested/complete-record inspection, exact signatures, contexts,
engine reuse and card-to-test mappings. Fresh evidence compared sacrifice values
(four names, three estimated unlocks), any-number counter payments plus dynamic
mana (five Batteries, five potential unlocks, larger effort), and variable
optional mana (two names, one unlock). Sacrifice values won on three complete
cards using existing typed engine services. The selection contract, reuse boundary
and acceptance cases were stated before edits. No missing engine prerequisite,
source defect or environmental blocker was found.

Exact support: permanent-source Activated/2 and ActivatedWithModifiers/3 with
one unambiguous SacrificeAPermanent/1 cost. ToughnessOfTheSacrificedCreature is a
symbol and requires the exact IsCardtype(Creature) cost filter;
ManaValueOfTheSacrificedPermanent is a symbol and accepts existing matching
sacrifice filters. Consumers are GainLife/1, LoseLife/1, DrawNumberCards/1 and
NumberTokens/2 counts in existing basic CreateTokens/2 forms. The declaration
cannot combine characteristic kinds, multiple matching sacrifices or a source
sacrifice with a matching sacrifice. Unbound, incompatible, tuple-shaped,
ambiguous and cross-activation references fail compilation. Dynamic mana,
damage values, filtered toughness costs, optional sacrifice payments, spell
additional costs and static reads remain gaps.

Private signatures: compileSacrificeValuePlan(Node, Node)
(*sacrificeValuePlan, error); compileBoundCosts(Node, *sacrificeValuePlan)
([]boundCostFactory, bool, error); compileInstanceValue(Node, scope)
(valueFactory, error). A boundCostFactory is func(instanceBindings) mage.Cost;
a valueFactory is func(instanceBindings) mage.ResolutionValue. Per-factory costs
and effects use the same private typed NumberBinding. Immutable plans carry no
game state. There is no global last-action read, name branch, fallback, declaration
panic or partial production registry.

Engine reuse: SacrificeCreatureIncludingSourceCostWithStats,
SacrificeMatchingIncludingSourceCostWithManaValue, BindNumber, FromNumber,
transactional costs, current characteristics, owner graveyards, target incarnations,
ability copies, clones, life/draw actions and CreateTokenCopies. No engine API or
implementation changed. CR 602.1a/1b, 602.2/2b, 400.7 and 608.2h apply with the
existing phasing/copy/token contracts. No new replacement operation or simultaneous
service is established. API/rules documentation records only this adapter behavior.

| Card or contract | Test mapping |
| --- | --- |
| Diamond Valley, Life Chisel, Homarid Spawning Bed | TestRONSacrificeValueRealCards: complete-record gameplay and definition validation, counters/current toughness, exact mana/tap costs, payment timing, owners, returned victim, source departure/return/control/phasing, pending copies, clones, life or Camarid count/characteristics |
| Diamond Valley copies | TestRONSacrificeValueCopyController: copied controller receives saved value without paying again |
| Life Chisel timing | TestRONSacrificeValueRealRestrictions: step, turn, permission and phased-source rejection; real-card tests change timing after payment |
| Payment/identity | TestRONSacrificeValuePendingAndRollback; TestRONSacrificeValueFactoriesAndSelfPayment: distinct pending amounts, same-factory sources, self-payment, nonpositive values, mana failure and invalid victim choices |
| Consumer contexts | TestRONSacrificeValueConsumers: both saved characteristic kinds in each supported consumer |
| Target legality | TestRONSacrificeValueTargetLegality: invalid announcement before payment; departed/returned/phased target at resolution |
| Invalid declarations | TestRONSacrificeValueInvalid: missing producer, incompatible cost, ambiguous sacrifice, mixed characteristic kinds, tuple form, spell and cross-rule scope |

TDD: after correcting fixture API errors, real-card and pending-payment tests
failed on unsupported saved values. Implementation passed focused tests. Broad
checks found two stale inventory counts and the old rejection of newly supported
creature toughness; that fixture now rejects an incompatible artifact cost.
No gameplay requirement was weakened. Final consumer/target/copy assertions
passed, followed by lint and all required checks on the final code.

Estimated and actual gain agree: **three unique names, three records**.
Loader **1,032 -> 1,035/2,445 compiled**, **1,413 -> 1,410 rejected**.
Accepted names **330 -> 333**, rejected **656 -> 653**. Zero accepted-record
regressions; strict and semantic loader diagnostics match. Nodes/signatures remain
72,938/8,528; unknown-role occurrences remain 16,921 because these operands already
had value roles. Global engine_validated/behavior_verified remain null. Three
complete cards gain explicit RON gameplay and definition-validation evidence;
compilation and collection-wide gameplay completion remain separate.

Largest remaining diagnostics: PermanentRuleEffect (110), AsPermanentEnters (90),
instant/sorcery CastEffect (53), If rule (48), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep (35), TriggerI (28),
ControlledByAPlayer filter (28), TargetGraveyardCard (26). Priest of Yawgmoth still
rejects its dynamic mana consumer. All 12 RON hashes match the vocabulary snapshot;
no source refresh is needed. No source defect was corrected. Gaea's Liege and
Fastbond concerns remain open.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron -count=1; go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline strict and inventory loaders | Expected exit 1: 1,032 compiled, 1,413 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONSacrificeValue -count=1 | Expected unsupported-value failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | Initial stale support/count fixtures; final PASS after updates, repeated after lint |
| go test ./pkg/mage/... -short -count=1 | PASS, repeated after final tests/lint |
| make test | PASS, repeated after final tests/lint |
| make build | PASS, repeated on final code |
| go vet ./... | PASS, repeated on final code |
| make lint | PASS, zero issues; automatic changes reviewed, no semantic changes |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and semantic loader | Expected exit 1: 1,035 compiled, 1,410 rejected; identical diagnostics |
| Accepted-record and source-hash audits | Zero regressions; all 12 hashes unchanged |

Slice files: new sacrifice_value.go/sacrifice_value_test.go;
activation_timing.go, compiler.go, player_scope.go, token_creation.go;
sacrifice_cost_test.go and inventory_test.go; dsl-reference.md,
engine-rules-contracts.md; the new report and this ledger. Current production
startup, Go-card definitions/generators and DSL exports retain callers. Removal
gates remain closed; no deletion, DSL removal audit or runtime cutover is claimed.

Ranked next candidates to reassess: (1) dynamic repeated mana with typed payment
values, directly Priest of Yawgmoth and downstream five Batteries; (2) any-number
counter payments/results for those Batteries; (3) shared end-step/delayed-trigger
prerequisites for Breeding Pit, Rukh Egg and Hazezon Tamar. The first candidate is
a shared prerequisite despite its one direct unlock. Complete loading/gameplay,
runtime/native startup, Go-card/generator deletion and DSL cleanup remain open.
No second slice began.

## Run 2026-10-04: P4-COUNTER-PAYMENT

Status: completed one source-counter-payment and repeated-mana adapter slice.
User changes to `.gitignore` and `mtgish-ron-execution.md` are preserved and
excluded from the commit. No unrelated untracked files were present.

The [counter payment report](mtgish-ron-counter-payment.md) contains the candidate
comparison, all affected distinct names, complete-record/nested-form blockers,
exact signatures and contexts, engine reuse and card-to-test mappings. Fresh
comparison: counter payments and saved-count mana (eleven affected names, five
estimated unlocks), saved-sacrifice mana (Priest of Yawgmoth, Sacrifice and the
related Metamorphosis modifier form; one unlock), and step/delayed triggers
(Breeding Pit, Rukh Egg, Hazezon Tamar; one end-step unlock or two with delay).
The selected five-card gain has moderate adapter/test effort and reuses existing
engine contracts. Selection, reuse boundary and acceptance cases preceded edits.
The prior ledger suggestion was reassessed; no missing engine prerequisite was
found. No source-data or environment defect was corrected.

Exact supported shape: permanent-source `Activated/2` or
`ActivatedWithModifiers/3`, one
`RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(counter, ThisPermanent)])`
cost, alone or in existing `And([costs...])`. The saved result is symbol
`TheNumberOfCountersRemovedThisWay`. Repeated mana accepts concrete colors,
`AnyManaColor` and existing `And` production lists. Existing instance-value
consumers also accept the result for life, draw and basic token counts. Unused
results are valid. Multiple/mixed removals, wrong subject/type, tuple results,
unbound and cross-activation reads are compilation errors. Optional resolution
counter payments, spell costs, X costs and general dynamic mana remain gaps.

Private interfaces: `compileCounterPaymentPlan(Node) (*counterPaymentPlan,error)`;
`compileActivationCosts(Node,*sacrificeValuePlan,*counterPaymentPlan)
([]boundCostFactory,bool,error)`; `compilePaidCounterMana(Node,scope)
([]effectFactory,error)`; `actionPlan.buildWithBindings(instanceBindings) Actions`.
`compileInstanceValue` retains its signature and validates counter-result scope.
Plans have no game state. Each factory builds a private typed number binding shared
by its payment and effects. Existing build callers retain their behavior.

Engine reuse: `RemoveAnyCountersCost`, `BindNumber`, `FromNumber`, total-cost
transactions, `AddManaAmount`, `AddManaOfAnyOneColor`, mana classification,
target/incarnation validation, copy/clone state and shared token entry. Zero is a
valid payment. Untargeted mana resolves immediately; targeted production uses
the stack. Solver queries do not choose/pay and exclude choice-cost abilities.
No engine API or rule changed. CR 602.2/2b, 605.1a/2/3b/5a, 400.7 apply. Existing
replacement and simultaneous-entry services remain in use. RON owns syntax and
scope; no fallback, card-name branch, global last-action slot, game-time declaration
panic or partial production registry was added.

| Card or contract | Test mapping |
| --- | --- |
| Black, Blue, Green, Red and White Mana Battery | `TestRONCounterPaymentBatteries`: each complete RON record, definition validation, both abilities, charging payment/timing, zero/partial/all counter removal, correct color/controller, immediate mana and clone/solver purity |
| Rollback and activation legality | `TestRONCounterPaymentInvalidChoicesAndRollback`: negative/excess choice, unpaid combined mana, unauthorized/phased/tapped source |
| Saved amounts, LKI/identity, copies and clones | `TestRONCounterPaymentSavedInstances`: distinct pending payments, source leave/return/control/phasing, copied controller, clone isolation |
| Target legality and mana classification | `TestRONCounterPaymentTargets`: announcement before payment, stacked target ability, return/phasing invalidation |
| Fresh factories and result consumers | `TestRONCounterPaymentFactoriesAndConsumers`: independent factory instances, any-color/multiple-color mana, life/draw/token counts |
| Zero/unconsumed result and other counter type | `TestRONCounterPaymentZeroAndUnused` |
| Invalid declarations and semantic roles | `TestRONCounterPaymentInvalidDeclarations`, `TestRONCounterPaymentInventory` |

TDD: fixture API errors were corrected, then new gameplay tests failed on the
expected unsupported counter instruction. The inventory test failed on missing
operand roles before the role adapter was added. A phased-source fixture now
phases in before normal battlefield lookup. The old fixed-cost rejection fixture
now rejects a host subject; repository inventory fixtures use fresh counts.
No gameplay expectation was weakened. Automatic lint changes were reviewed:
import grouping and a switch rewrite only. Final checks followed lint and the
last zero/unconsumed-result assertions.

Estimated and actual gain agree: **five unique names and five records**.
Compiled **1,035 -> 1,040/2,445**; rejected **1,410 -> 1,405**. Accepted names
**333 -> 338**, rejected **653 -> 648**. Zero previously accepted-record
regressions; strict/inspection issue arrays match. AST nodes/signatures remain
**72,938/8,528**, unknown-role occurrences **16,921 -> 16,599**. Global
engine_validated/behavior_verified remain null. Five complete cards gain explicit
RON gameplay/definition evidence; compilation is separate from collection-wide
coverage. All twelve source hashes match the vocabulary snapshot, so no source
refresh is needed. Gaea's Liege/Fastbond source concerns remain open.

Largest remaining constructor diagnostics include PermanentRuleEffect (110),
AsPermanentEnters (90), If (65 across diagnostic contexts), CastEffect (54 across
contexts), ChooseADamageSource (46), Targeted_Modal (42), HostController (35),
ControlledByAPlayer (29), TriggerI (28), TargetGraveyardCard (26). The report
preserves the storage-land and Tetravus blockers rather than claiming full support
for their payment contexts.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron -count=1; go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline strict and semantic loaders | Expected exit 1: 1,035 compiled, 1,410 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONCounterPayment -count=1 | Expected adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS, repeated after final assertions/lint |
| go test ./pkg/mage/... -short -count=1 | PASS, repeated after final assertions/lint |
| make test | PASS, repeated after final assertions/lint |
| make build | PASS, repeated on final code |
| go vet ./... | PASS, repeated on final code |
| make lint | PASS, zero issues; automatic changes reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json and semantic loader | Expected exit 1: 1,040 compiled, 1,405 rejected; identical diagnostics |
| Accepted-record and source-hash audits | Zero regressions; all twelve hashes unchanged |

Changed files: new counter_payment.go/counter_payment_test.go; activation_timing.go,
compiler.go, player_scope.go, sacrifice_value.go; inventory.go/inventory_test.go and
counter_cost_test.go; dsl-reference.md, engine-rules-contracts.md, this ledger and
the new report. Production startup, Go card definitions/generators and DSL exports
retain their callers. Removal gates remain closed; no runtime cutover or DSL
removal is claimed.

Ranked next candidates to reassess with fresh evidence: (1) entry and optional
untap/intervening-if prerequisites for Bottomless Vault, Dwarven Hold, Hollow Trees,
Icatian Store and Sand Silos, whose counter/mana activation is now supported;
(2) step/delayed-trigger services for Breeding Pit, Rukh Egg and Hazezon Tamar;
(3) saved-sacrifice repeated mana as a shared consumer prerequisite for Priest
of Yawgmoth, Sacrifice and Metamorphosis. Complete collection gameplay/loading,
runtime/native startup, Go-card/generator deletion and DSL cleanup remain open.
No second slice began.

## Run 2026-10-04: P6-END-STEP

Status: completed one ordinary end-step trigger slice. User changes in
`.gitignore` and `mtgish-ron-execution.md` are preserved and excluded from the
commit. No unrelated untracked files were present at baseline.

The [end-step report](mtgish-ron-end-step.md) records complete affected names,
nested/complete-record inspection, candidate comparison, exact interfaces,
engine reuse, remaining blockers and card-to-test mappings. The selection
contract and acceptance cases preceded edits. Comparison:

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers / effort |
| --- | --- | ---: | --- |
| Ordinary end-step event, selected | Ball Lightning, Breeding Pit, Erg Raiders, Khabál Ghoul, Osai Vultures, Pestilence, Scavenging Ghoul, Season of the Witch, Voodoo Doll, Whirling Dervish | 2 | Other eight need histories/TriggerI/damage/filters; small event adapter and typed-scope effort |
| Storage-land lifecycle | Bottomless Vault, Dwarven Hold, Hollow Trees, Icatian Store, Sand Silos | 5 with all prerequisites | Entry tapped, optional untap and tapped-source TriggerI; three contexts, moderate to large effort |
| Saved-sacrifice mana | Priest of Yawgmoth, Sacrifice, Metamorphosis | 1 | Spells need casting-payment bindings; Metamorphosis also Plus/restricted mana; small consumer, broader spell work |

Two complete cards with one small adapter won on credible gain per effort.
The selected event/player scope is a shared dependency for later end-step
history and intervening-if adapters. Storage lifecycle enables reusable entry,
untap permissions and conditions. Paid-value mana enables the two blocked
spells. Delayed next-end-step was inspected as an extension and deferred.
No engine prerequisite or environment failure was found.

Exact support: permanent-source TriggerA/2 containing
AtTheBeginningOfAPlayersEndStep/1 with symbol AnyPlayer or SinglePlayer/1 of
symbol You, and existing ActionList/1 or single-group Targeted/2 action bodies.
Symbol Trigger_ThatPlayer is a typed step-player reference in this body; it
remains separate from You. Existing player consumers include life/draw/discard
and optional fixed mana payment. Other player filters, TriggerI and future
triggers remain errors. Cross-rule, incompatible, tuple-shaped and unbound
reads fail compilation. No spell/cost/static/event-spell scope is invented.

Private signature: compileEndStepTrigger(event, body Node, ctx scope)
(optionFactory,error). Immutable nonzero-sized stepEventPlan carries scope
identity. Each factory builds a fresh PlayerBinding in instanceBindings.steps;
playerReferencePlan.step uses FromPlayer. Existing caller signatures remain.
Targets and effects are built in that factory. No card-name branch, fallback,
global last-action slot, game-time declaration panic or partial registry exists.

Engine reuse: EachEndStep, YourEndStep, TurnStepBegins, WhenEvent, BindPlayer,
FromPlayer, Sequence, pending-trigger/APNAP placement, typed target legality,
source incarnations, stack copies, clone state, sacrifice, token entry and player
actions. No engine API or rule changed. Local CR 513.1–2, 603.1–5 and 701.21a
were read; existing CR 400.7 and 608.2b identity/target contracts apply. No new
LKI query, replacement ordering or simultaneous-operation contract is claimed.
API/rules documentation describes only this ordinary-event adapter.

| Card / contract | Test mapping |
| --- | --- |
| Ball Lightning | TestRONEndStepBallLightning: complete record, definition validation, either boundary, source identity/control/phasing, countered trigger, owner graveyard and clones |
| Breeding Pit | TestRONEndStepBreedingPit: complete record, definition validation, upkeep pay/decline/short and end-step interaction, current/pending controller, departure/return/phasing/copy and exact Thrull characteristics |
| Both full cards / normal turn | TestRONEndStepNativeTurnAndCombat: failed cast rollback, real casting/mana, haste attack legality, trample damage, normal step dispatch/resolution, late entry and cleanup distinction |
| Controller / event instances | TestRONEndStepCurrentController; TestRONEndStepEventBindings: current control, phasing, independent factories/pending events, copied controller, event-player capture and clones |
| Consumers / targets | TestRONEndStepExplicitPlayerConsumers; TestRONEndStepTargetsAndPhaseMembership: explicit event-player life/draw/discard and target placement/resolution/return/phasing |
| Invalid scope / inventory | TestRONEndStepInvalidDeclarations; TestRONEndStepInventory; fresh repository inventory fixtures |

TDD: new tests first failed on the expected unsupported trigger and missing
inventory operand roles. Fixture assumptions were corrected for normal step
priority resolution, synthetic owner IDs and CR 701.21a's control restriction.
The explicit opponent event-player test then exposed the missing step-life
consumer adapter, which now uses the typed step player. No test was weakened.
Lint made only import-format changes; automatic edits were reviewed. All
required checks passed on the final code after the last assertions and lint.

Estimated and actual gain agree: **two unique names and two records**.
Compiled **1,040 -> 1,042/2,445**; rejected **1,405 -> 1,403**.
Accepted names **338 -> 340**, rejected names **648 -> 646**. Zero accepted-record
regressions; strict and inspection issue arrays match. AST occurrences/signatures
remain **72,938/8,528**; unknown-role occurrences **16,599 -> 16,588**.
Collection-wide engine_validated/behavior_verified remain null. Two complete
cards gain explicit RON gameplay/definition evidence; collection compilation
and gameplay completion remain separate. All twelve source hashes match the
vocabulary snapshot, so no source refresh is needed. No source defect was
corrected; Gaea's Liege/Fastbond concerns remain open.

Leading diagnostics remain PermanentRuleEffect 110, AsPermanentEnters 90,
If 65 across contexts, CastEffect 54 across contexts, ChooseADamageSource 46,
Targeted_Modal 42, HostController 35, ControlledByAPlayer 29, TriggerI 28 and
TargetGraveyardCard 26. Exact per-card remaining blockers are in the report.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline strict and semantic loaders | Expected exit 1: 1,040 compiled, 1,405 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONEndStep -count=1 | Expected unsupported adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS, repeated after final tests/lint |
| go test ./pkg/mage/... -short -count=1 | PASS, repeated after final tests/lint |
| make test | PASS, repeated after final tests/lint |
| make build | PASS on final code |
| go vet ./... | PASS on final code |
| make lint | PASS, zero issues; automatic formatting reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; semantic inspection | Expected exit 1: 1,042 compiled, 1,403 rejected; identical diagnostics |
| Accepted-record / snapshot source-hash audits | PASS: zero regressions, all twelve hashes unchanged |

Slice files: new end_step.go/end_step_test.go and this run's report; compiler.go,
player_scope.go, triggers.go, inventory.go/inventory_test.go; dsl-reference.md,
engine-rules-contracts.md and this ledger. Go-card definitions, generators,
production registry wiring and existing DSL exports retain callers. The removal
gates remain closed. No DSL function was removed; no deletion audit or runtime
cutover is claimed.

Ranked next candidates to reassess: (1) storage-land lifecycle prerequisites,
including comparison of smaller entry-only forms; (2) shared death-history values
for Khabál Ghoul/Scavenging Ghoul, with TriggerI downstream for Osai Vultures;
(3) next-end-step delayed capture for Rukh Egg and nested future-trigger users.
Complete loading/gameplay, runtime/native startup, Go-card/generator deletion
and DSL cleanup remain open. No second slice began.

## Run 2026-10-04: P8-ENTRY-TAPPED

Status: completed one tapped self-entry adapter slice. User changes in
`.gitignore` and `mtgish-ron-execution.md` are preserved and excluded from this
slice's commit. No unrelated untracked files were present at baseline.

The [tapped-entry report](mtgish-ron-entry-tapped.md) contains complete-record
inspection, exact shapes, dependencies, engine reuse, remaining per-card gaps
and card-to-test mappings. Fresh short tests/build passed; strict and semantic
loaders rejected unsupported records as expected. Selection and acceptance
cases preceded edits. Candidate comparison:

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers / downstream benefit / effort |
| --- | --- | ---: | --- |
| Tapped self entry, selected | Bottomless Vault, Bronze Tablet, Dwarven Hold, Dwarven Ruins, Ebon Stronghold, Havenwood Battleground, Hollow Trees, Icatian Store, Leviathan, Nevinyrral's Disk, Ruins of Trokair, Sand Silos, Svyelunite Temple, Sword of the Ages, Time Vault | 6 | Other nine need untap, TriggerI, plural payment/results, ante/ownership or turn services. Entry is a prerequisite for five storage lands. Small adapter; moderate six-card gameplay coverage. |
| Creature death history | Khabál Ghoul, Scavenging Ghoul | 2 | Exact records have no other blockers. Enables later end-step/intervening-if history consumers. Small adapter; moderate LKI/token/turn-reset tests. |
| Saved-sacrifice repeated mana | Priest of Yawgmoth, Sacrifice, Metamorphosis | 1 | Spells need casting-payment bindings; Metamorphosis also Plus/restricted mana. Shared paid-value consumer. Small activation adapter; moderate saved-result/rollback/solver tests, larger casting work. |

Six credible full-card unlocks with one small compiler adapter won on gain per
estimated effort. The parent AST signature groups entry lists by kind; nested
EntersTapped signatures were inspected to identify 15 unique affected names.
All their complete RON records and both competing candidates' complete records
were read. No engine prerequisite, environment failure or source-data defect
was found for this context. No RON source changed.

Exact support: permanent-source `AsPermanentEnters/2`, symbol `ThisPermanent`,
list containing exactly symbol `EntersTapped`. Private signature:
`compileTappedEntry(n Node) (optionFactory,error)`. Each factory lowers to
`mage.WithKeyword(core.EntersTapped)`. There is no mutable binding or chosen
state in this declaration. Inventory operands use permanent_reference and
entry_operation roles, without claiming gameplay support. Invalid arity, lists,
subjects, tuple symbols, mixed operations, spell context and general action or
keyword use are errors. Other entry choices/copy/counter contexts remain gaps.

Engine reuse: prepareBattlefieldEntry, attribute seeds, card/permanent/game
cloning, normal untap, phase-in/out, source incarnations, land play, mana solver
metadata/immediate abilities, atomic cost transactions, owner-directed sacrifice,
shared destruction batches, regeneration/indestructible checks, leave-event LKI
and stack resolution. RON owns declaration spelling and diagnostics. No engine
API/rule changed; no fallback, card-name branch, global last-action slot,
game-time declaration panic or partial registry was added. CR 400.6–7,
614.1c–d/4/12, 702.26b–d, 701.21a and 602.2/5 were read with relevant API/rules
contracts. No new competing-entry-replacement ordering contract is claimed.

| Card or contract | Test mapping |
| --- | --- |
| Dwarven Ruins, Ebon Stronghold, Havenwood Battleground, Ruins of Trokair, Svyelunite Temple | TestRONEntryTappedLands: each full FEM record, definition validation, native land play, both abilities, exact immediate mana, illegal activations, solver and clone state |
| All five lands / owner versus controller | TestRONEntryTappedNativeUntapAndOwner: native untap steps, normal untap, sacrifice ignores regeneration and goes to owner; controller receives mana |
| Nevinyrral's Disk | TestRONEntryTappedDisk: full LEB record, definition validation, actual casting and entry, casting/activation rollback, departure, three-type destruction union, regeneration once, indestructible, land/phasing exclusions, complete pre-change battlefield at death detection and clone resolution |
| All six complete cards | TestRONEntryTappedLifecycle: direct entry, separate controllers, return, phasing, independent factory instances and cloned state |
| Entry timing / transition | TestRONEntryTappedEventTiming: synthetic RON entry/ETB with typed engine observer fixtures, tapped state before ETB, no entry tap event, later tap event |
| Card cloning | TestRONEntryTappedCardCopy |
| Declaration rejection / inventory | TestRONEntryTappedInvalid, TestRONEntryTappedInventory, updated collection inventory fixtures |

TDD first corrected a fixture API typo, then confirmed expected unsupported
AsPermanentEnters and missing operand-role failures. Native land-play fixtures
now use the hand owner's card; the separate control test covers another owner.
The simultaneous observer uses the engine's pre-change leave-event state and
retains its original death filter when adding a batch condition. No gameplay
expectation was weakened. Lint found a test variable named `copy`, which was
renamed; automatic import formatting was reviewed. Final validation followed
all test changes and the successful final lint run.

Estimated and actual gains agree: **six unique names, eleven records** (six Disk
reprints and five lands). Compiled **1,042 -> 1,053/2,445**; rejected
**1,403 -> 1,392**. Accepted names **340 -> 346**, rejected names **646 -> 640**.
Zero accepted-record regressions. Strict and inspection issue arrays match.
AST occurrences/signatures remain **72,938/8,528**; unknown-role occurrences
**16,588 -> 16,310**. Collection-wide engine_validated/behavior_verified remain
null. Six complete cards gain explicit gameplay/definition evidence; compilation
and collection-wide gameplay remain separate. All twelve source hashes match
the vocabulary snapshot, so no syntax refresh is needed. Gaea's Liege/Fastbond
source concerns remain open.

Leading diagnostics: PermanentRuleEffect 121, If 65, CastEffect 54,
ChooseADamageSource 46, Targeted_Modal 42, HostController 35,
ControlledByAPlayer 29, TriggerI 28, TargetGraveyardCard 26. Storage-land and
other affected-card blockers are listed in the report; changed first errors do
not count as full-card unlocks.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline go run ./cmd/loadcards -json; semantic inspection | Expected exit 1: 1,042 compiled, 1,403 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONEntryTapped -count=1 | Expected adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS, repeated on final code after lint |
| go test ./pkg/mage/... -short -count=1 | PASS, repeated on final code after lint |
| make test | PASS, repeated on final code after lint |
| make build | PASS, repeated on final code after lint |
| go vet ./... | PASS, repeated on final code after lint |
| make lint | Initial builtin-variable failure corrected; final PASS, zero issues |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; semantic inspection | Expected exit 1: 1,053 compiled, 1,392 rejected; identical diagnostics |
| Accepted-record / snapshot source-hash audits | PASS: zero regressions, all twelve hashes unchanged |

Slice files: entry_tapped.go/entry_tapped_test.go, compiler.go,
inventory.go/inventory_test.go, dsl-reference.md, engine-rules-contracts.md,
this ledger and the new report. Production startup, Go-card definitions,
generators and existing DSL callers remain unchanged. Cutover/removal gates
remain closed. No DSL export was removed; no deletion caller audit is claimed.

Ranked next candidates to reassess: (1) creature-death history for Khabál Ghoul
and Scavenging Ghoul; (2) optional untap and tapped-source TriggerI for the five
storage lands, now with tapped entry and counter/mana payments supported;
(3) saved-sacrifice repeated mana, with casting-payment scope downstream.
Complete collection loading/gameplay, runtime/native startup, Go-card/generator
deletion and DSL cleanup remain open. No second slice began.

## Run 2026-10-04: P6-DEATH-HISTORY-COUNTERS

Status: completed one dependency-ready compiler slice. User changes in
`.gitignore` and `mtgish-ron-execution.md` are preserved and excluded from the
slice commit. Fresh short tests/build passed. Baseline strict and semantic
loading rejected 1,392 records, with 1,053 compiled. The
[death-history report](mtgish-ron-death-history.md) records contracts and tests.

| Candidate | Distinct affected names | Estimated complete unlocks | Other blockers / downstream capability / effort |
| --- | --- | ---: | --- |
| Creature death history, selected | Khabál Ghoul; Scavenging Ghoul | 2 | No other blockers. Typed engine history is ready; supports later history conditions. Small adapter, moderate gameplay coverage. |
| Storage-land lifecycle | Bottomless Vault; Dwarven Hold; Hollow Trees; Icatian Store; Sand Silos | 5 | Optional untap plus tapped-source TriggerI still missing. Entry and counter/mana payments work. Medium implementation, large lifecycle/decision tests. |
| Saved-sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 | Priest needs repeated-mana saved-value consumption; spells need casting-payment bindings, Metamorphosis also Plus/restricted mana. Small activation adapter and moderate rollback/solver tests, larger casting work. |

Complete records and nested forms for all candidate cards, all reprints, and
Osai Vultures were inspected. Two full unlocks with a single exact adapter won
on credible benefit for effort. Selection, contract, reuse boundary and
acceptance cases were stated before edits. No missing engine prerequisite,
environment failure or source defect was found in this context.

Exact support: `NumCreaturesOrPlaneswalkersThatDiedThisTurn/1` with exactly
`IsCardtype(Creature)`, used as the amount of
`PutCounters([NumberCountersOfTypeOnPermanent(amount,counter,subject)])` in
supported spell, activation and trigger action lists. Private signature:
`compileCounterAmount(Node, scope) (mage.ResolutionValue, error)`.
The filter operand receives permanent_filter inventory context; syntax support
remains unverified. Other types, mixed/controller filters and malformed shapes
are declaration errors. TriggerI history conditions remain unsupported.

Engine reuse: CreaturesDiedThisTurn, turn tracker, AddCountersTo, typed source/
target incarnations, static animation LKI, shared destruction batches,
regeneration, source phasing, pending triggers/copies, game cloning, cleanup
history reset, native casting and transactional activation costs. Integer/X
amounts retain ValueAtResolution. The new value has no mutable chosen state
or result binding; each factory builds separate effects. RON owns syntax and
shape validation. No engine rule changed, no global last-action slot, fallback,
card-name branch, declaration panic or partial registry was introduced.

Card-to-test mappings: Khabál Ghoul -> TestRONDeathHistoryCards,
TestRONDeathHistoryEvents, TestRONDeathHistoryCastingAndInstances;
Scavenging Ghoul -> TestRONDeathHistoryCards (including activation cost and
regeneration) and TestRONDeathHistoryCastingAndInstances (all six reprints).
TestRONDeathHistoryActionContexts covers synthetic spell and activation usage;
TestRONDeathHistoryInvalid and TestRONDeathHistoryInventory cover invalid
shapes and syntax roles. The report details timing, source identity, LKI,
phasing, clones, copies, simultaneous deaths and cost failure coverage.

TDD confirmed the unsupported value/filter-role failures before implementation.
Fixtures were corrected for explicit owner IDs, activation index 1, and phased
legality through the batch destruction API. No test was weakened. Lint made
import formatting changes only; changes were reviewed. Final tests ran after
all test changes and the final successful lint run.

Estimated and actual unlocks agree: **two unique names, seven records**.
Compiled **1,053 -> 1,060/2,445**; rejected **1,392 -> 1,385**. Accepted unique
names **346 -> 348**, rejected names **640 -> 638**. Zero accepted-record
regressions; strict and semantic issue arrays agree. AST occurrences/signatures
remain **72,938/8,528**; unknown-role occurrences **16,310 -> 16,303**.
Collection-wide engine_validated/behavior_verified remain null; two full cards
have new explicit RON gameplay/definition evidence. No RON source changed;
all twelve hashes match the vocabulary snapshot. Gaea's Liege/Fastbond source
concerns remain open. No source correction or vocabulary refresh was needed.

Leading first diagnostics: PermanentRuleEffect 121, If 65, CastEffect 54,
ChooseADamageSource 46, Targeted_Modal 42, HostController 35,
ControlledByAPlayer 29, TriggerI 28, TargetGraveyardCard 26.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline strict and semantic loading | Expected exit 1: 1,053 compiled, 1,392 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONDeathHistory -count=1 | Expected unsupported-value/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS on final code after lint |
| go test ./pkg/mage/... -short -count=1 | PASS on final code after lint |
| make test | PASS on final code after lint |
| make build | PASS on final code after lint |
| go vet ./... | PASS on final code after lint |
| make lint | PASS, zero issues; automatic formatting reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory | Expected exit 1: 1,060 compiled, 1,385 rejected; identical diagnostics |
| Accepted-record regression and source-hash audits | PASS: zero regressions, twelve hashes unchanged |

Changed files: counter_placement.go; new death_history_test.go; inventory.go;
inventory_test.go; dsl-reference.md; engine-rules-contracts.md; this ledger;
new mtgish-ron-death-history.md. No Go card, generator or existing DSL export
was removed. Complete-loading/gameplay and runtime cutover gates remain closed;
no removal caller audit or native startup cutover is claimed.

Ranked next candidates to reassess: (1) storage-land optional untap and
 tapped-source TriggerI prerequisites; (2) saved-sacrifice repeated mana with
 downstream casting-payment bindings; (3) death-history intervening-if for
 Osai Vultures, deferred while useful multi-card work remains unless required
 as a shared prerequisite. No second slice began.

## Run 2026-10-04: P6-STORAGE-LAND-LIFECYCLE

Status: complete one storage-land lifecycle slice. The initial working tree had
user changes in `.gitignore` and `mtgish-ron-execution.md`. They remain unchanged
and are excluded from the commit. No unrelated untracked files were present.
Fresh focused tests, short engine tests and build passed. Strict loading returned
exit 1 for unsupported collection records, not a build or environment failure.

### Candidate comparison and selection

Fresh strict loading: 1,060 compiled / 2,445 records, 1,385 rejected records,
348 accepted unique names and 638 rejected unique names. Full nested inspection:
72,938 occurrences, 8,528 context/shape signatures, 16,303 unknown-role
occurrences. The vocabulary JSON is a syntax snapshot, not support evidence.

| Candidate | Distinct affected names | Estimated full-card unlocks | Blockers, downstream value and effort |
| --- | --- | ---: | --- |
| Storage-land lifecycle, selected | Bottomless Vault; Dwarven Hold; Hollow Trees; Icatian Store; Sand Silos | 5 | Two small adapters: optional self untap and tapped self upkeep intervening-if. All entry/counter/mana prerequisites exist. Moderate lifecycle and payment tests; opens optional untap for eleven other unique cards. |
| Saved-sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 | Priest needs bound repeated mana; Sacrifice also needs casting-payment scope; Metamorphosis additionally needs Plus and restricted mana. Small activation adapter, moderate payment/solver tests, larger casting work. |
| Death-history intervening-if | Osai Vultures | 1 | Existing history and counter/boost services are ready; condition lowering remains absent. Small adapter, moderate event/resolution/history tests. Enables shared history-condition use, but this collection has only one immediate full unlock for that shape. |

Five credible complete unlocks for two small adapters won on benefit per effort.
Selection, reuse boundary and acceptance cases were stated before editing.
Complete records and nested forms were read for all three candidate groups,
including all Sacrifice reprints. No first-error count was used as an unlock
estimate. Optional untap occurs in sixteen unique cards. In addition to the five
lands, complete records were inspected for Ashnod's Battle Gear, Old Man of the
Sea, Phyrexian Gremlins, Preacher, Rubinia Soulsinger, Seasinger, Spirit Shield,
Tawnos's Coffin, Tawnos's Weaponry, Willow Satyr and Zelyon Sword. Their remaining
blockers include player-controlled filters, power/supertype filters, opponent
choices, control durations, tapped-maintained modifications, action-result
bindings, linked exile and future triggers. They remain rejected; no partial
production registry is published. Mana Vault's complete record was also read:
its tapped condition uses a draw-step event, which remains unsupported.

### Supported contract and reuse boundary

Private signatures:

- `compileOptionalUntap(Node) (optionFactory, error)` supports only rule
  `PermanentRuleEffect/2` with symbol `ThisPermanent` and a list containing
  exactly symbol `MayChooseNotToUntapDuringUntap`.
- `compileTappedUpkeep(Node, scope) (optionFactory, error)` supports only rule
  `TriggerI/3`, event `AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))`,
  condition `PermanentPassesFilter(ThisPermanent, IsTapped)`, and an action
  declaration accepted by the shared action compiler. Each constructor has its
  exact tuple arity; source/player/filter operands are symbols.

Both rules require a permanent card context. Spell, cost and action contexts,
wrong arity, tuple-shaped symbols, other subjects, filters, events and mixed
optional untap operations remain declaration errors. General PermanentRuleEffect
and TriggerI families remain partial.

Optional untap lowers to `Static(While(condition,
GrantKeywordModification(This, AttrMayNotUntap)))` with an unconditional typed
battlefield condition. It grants the attribute through the existing ability
layer so ability removal disables it. The trigger lowers to
`At(YourUpkeep, Sequence(actions...), WithInterveningIf(PermanentIsTapped(This)))`.
Its action targets are added from the same freshly built action plan. No mutable
choice is captured in a compiled factory. Per-instance target/cost bindings and
source incarnations remain owned by existing engine services.

Engine reuse: native land play, tapped entry, untap-step player decisions and
restrictions, layer/ability lifecycle, YourUpkeep, typed state conditions,
intervening-if detection and resolution checks, source incarnations, event and
pending-trigger services, stack copies/countering, phasing, control changes,
cloning, counter addition, typed chosen-counter payment, immediate mana ability
resolution and transactional rollback. RON owns spelling, shape validation,
inventory roles and lowering. No engine rule/API changed; no card-name branch,
Go-card fallback, global last-action slot, declaration panic or parallel rules
service was added. Relevant contracts read: CR 400.7, 502.1/3/4, 503.1a,
603.3/4, 605.1a/3b and 702.26b–d, with API and engine rules-contract docs.
No new LKI-characteristic query, replacement ordering or simultaneous batch
operation was introduced; existing engine behavior remains the reuse boundary.

Inventory now recognizes PermanentRuleEffect operands as permanent_reference /
permanent_rule_operation, TriggerI operands as event / trigger_condition /
actions, and PermanentPassesFilter condition operands as permanent_reference /
permanent_filter. Roles do not assert syntax support or gameplay verification.

### Tests and evidence

Tests were written first. The initial gameplay run failed at unsupported
PermanentRuleEffect for each complete land record. The inventory test separately
failed because TriggerI's event operand had unknown role. Test fixtures were
corrected to use the main phase for land play and ability index 2 (the full
ability list includes the static and triggered abilities). No rule assertion was
weakened. A native upkeep test was added and passed. Lint reordered imports;
the automatic changes were reviewed.

| Unique cards or contract | Test mapping and evidence |
| --- | --- |
| Bottomless Vault, Dwarven Hold, Hollow Trees, Icatian Store, Sand Silos | TestRONStorageLands: complete FEM records, definition validation, native land play/tapped entry, native untap decline/accept and upkeep, repeated accumulation, opponent-upkeep exclusion, untapped condition, correct immediate color/amount, zero payment, tapped/excess-payment rejection and rollback, pure solver query and clone isolation |
| Sand Silos | TestRONStorageInterveningIf: false condition at resolution, untap/retap recheck, departure/return incarnation exclusion, phasing before detection/after placement, control change, copies with another controller, countered triggers and clone resolution |
| Sand Silos | TestRONStorageUntapInstances: separate factories/controllers/choices, current-controller choice after control change, phase-in before untap, SkipNextUntap precedence, ability removal, counters retained and clone isolation |
| Invalid declarations | TestRONStorageInvalid: malformed arity/operands, unbound target source, wrong filter/event/player, mixed operations, wrong role/context |
| Semantic inventory | TestRONStorageInventory plus updated collection-count fixtures: nested roles and shared action contexts |

These five complete cards gain explicit gameplay/definition evidence. This is
separate from collection-wide behavior totals. Other optional-untap cards remain
unsupported and are not claimed as full-card gameplay gains.

### Results and validation

Estimated and actual unlocks agree: **five unique names, five records**.
Compiled **1,060 -> 1,065/2,445**; rejected **1,385 -> 1,380**. Accepted names
**348 -> 353**, rejected names **638 -> 633**. Zero accepted-record regressions.
Strict and inspection issue arrays match. Occurrences/signatures remain
**72,938/8,528**; unknown-role occurrences **16,303 -> 15,289**.
Collection-wide engine_validated and behavior_verified remain null.

Leading fresh diagnostics include optional untap operation gaps (67), non-spell
CastEffect (53), rule If (48), ChooseADamageSource (46), Targeted_Modal (42),
optional untap HostPermanent (39), upkeep HostController (35), ControlledByAPlayer
(29), TargetGraveyardCard (26) and WhenAPermanentBecomesTapped (25). Changes to
first diagnostics are not full unlocks. There were no source defects or
environment failures in this slice. All twelve source hashes still match the
vocabulary snapshot; no RON source changed, so no source snapshot refresh was
needed. Existing Gaea's Liege/Fastbond source concerns remain open.

| Command | Result |
| --- | --- |
| Baseline focused tests; go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,060 compiled, 1,385 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONStorage -count=1 | Expected adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS, zero issues; import formatting reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,065 compiled, 1,380 rejected; matching diagnostics |
| Accepted-record regression and source-hash audits | PASS: zero regressions, twelve unchanged hashes |

Slice files: new storage_land.go/storage_land_test.go; compiler.go;
inventory.go/inventory_test.go; dsl-reference.md; engine-rules-contracts.md;
this ledger. Final checks ran after final test and lint changes. Production
startup, Go-card definitions, generators and current DSL callers are preserved.
No DSL export was removed; cutover/removal gates are still closed, and no caller
removal audit or native startup cutover is claimed.

Ranked next candidates to reassess from fresh complete-record inventory:
(1) shared untap prohibitions and tapped-source durations for artifact/control
families, comparing immediate unlocks with prerequisite effort;
(2) saved-sacrifice repeated mana plus casting-payment scope for Priest of
Yawgmoth, Sacrifice and Metamorphosis;
(3) shared trigger-condition lowering, including death history for Osai Vultures.
Complete loading/gameplay, startup cutover, Go-card/generator deletion and DSL
cleanup remain open. No second slice began.

## Run 2026-10-04: P5-NEXT-UNTAP

Status: complete one resolving next-untap action slice. The initial working tree
had user changes in `.gitignore` and `mtgish-ron-execution.md`; both are preserved
and excluded from the slice commit. No unrelated untracked files were present.
Baseline focused tests, short engine tests and build passed. Strict and semantic
loading returned exit 1 for unsupported collection records, not a build failure.

### Candidate comparison and selection

Fresh baseline: 1,065 compiled / 2,445 records, 1,380 rejected records;
353 accepted unique names, 633 rejected unique names. The complete nested
inventory has 72,938 occurrences, 8,528 context/shape signatures and 15,289
unknown-role occurrences. This report and full records, rather than first-error
frequency, selected the work. The vocabulary snapshot is not support evidence.

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers, downstream value and effort |
| --- | --- | ---: | --- |
| Next-untap suppression, selected | Barl's Cage; Elvish Hunter; Homarid Warrior; Deep Spawn | 3 | Deep Spawn still needs optional mill payment. Existing typed SkipNextUntap is ready. Small shared adapter, moderate legality/lifecycle tests. Establishes the singular execution prerequisite for Spore Cloud's collection restriction. |
| Saved-sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 | Priest has ready bound payment values; Sacrifice needs casting-payment scope; Metamorphosis also needs Plus and restricted mana. Small activation adapter, moderate payment/solver tests, larger casting prerequisites. |
| Death-history trigger condition | Osai Vultures | 1 | Ready history and counter services; condition adapter remains missing. Small adapter, moderate event/resolution/history tests. One immediate full unlock for this exact condition. |

The selected shared action gives three credible full unlocks with no missing
engine contract. It wins over one immediate mana unlock with further payment
prerequisites and the single-card history condition. Selection, contract, reuse
boundary and acceptance cases were stated before editing.

Complete RON records were read for all candidate names, all six Sacrifice
reprints, and Spore Cloud. Nested AST signatures for the selected action are
`tuple(symbol:Ref_TargetPermanent)` in Activated/Targeted/ActionList and
`tuple(symbol:ThisPermanent)` in Activated/ActionList. These contain exactly the
four selected affected names. Spore Cloud's distinct collection spelling has
`tuple(tuple:Or)` in SpellActions/ActionList and remains unsupported; it also
needs combat tap and prevention. No Go card definition supplied requirements.

### Contract, dependencies and reuse boundary

Exact shape/context contract: [mtgish-ron-next-untap.md](mtgish-ron-next-untap.md).
`compileEffect(Node, scope) ([]effectFactory, error)` accepts action tuple
`PermanentDoesntUntapDuringControllersNextUntap/1`. The operand must be an
in-scope symbol accepted by `compilePermanentRef(Node, scope)`:
`ThisPermanent`, Aura `HostPermanent`, or declared `Ref_TargetPermanent`.
The shared action compiler supports spells, activations, modifier-bearing
activations and supported trigger actions. It rejects wrong arity, tuple
references, wrong roles, unbound/wrong-type/cross-rule reads. General static
untap prohibitions and collection actions remain unsupported.

Lowering reuses `mage.SkipNextUntap(subject.build(instanceBindings))` in the
existing unary permanent action builder. Fresh typed target declarations give
each factory/action its own reads. Immutable plans store no selected state.
RON owns spelling, scope, diagnostics and inventory operand roles; mage owns
resolution, legality, source incarnations, Aura attachment/departure LKI,
turn restriction storage/expiration, control, phasing, copies and clones.
Payment still uses the existing transactional activation path. No engine API
or rule changed; no fallback, card-name branch, global action slot, declaration
panic, partial registry or parallel subsystem was added.

The restriction starts at resolution. It neither taps the subject nor prevents
other untap effects. The next untap step of the current controller consumes it,
even when the subject is untapped. Repeated applications expire together.
Control changes do not capture the old controller. Phasing after installation
preserves the restriction; absent/phased subjects cannot receive it. Departure
clears it, and return cannot redirect a pending typed reference. Read contracts:
engine doc.go, composable permanent-reference/SkipNextUntap services, turn
execution/TurnSystem, definition validation, DSL/API and rules-contract docs;
CR 400.7, 502.1/3/4, 608.2b and 702.26b/d. No new LKI characteristic query,
replacement ordering or simultaneous-operation contract is introduced.

### TDD and card-to-test evidence

Tests were written first. The initial run failed at the missing action adapter
for each real record and separately at the missing inventory operand role.
After implementation, shared-context tests found that the synthetic host fixture
omitted its Aura subtype. The fixture was corrected to declare an Aura; the
compiler rejection and scope contract were retained. No assertion was weakened.
Lint found a local variable named `copy`; it was renamed `copied`. Automatic
import formatting was reviewed, and affected tests were rerun.

| Card or contract | Test mapping and evidence |
| --- | --- |
| Barl's Cage, Elvish Hunter | TestRONSkipUntapRealCards: full records and definition validation; exact mana/tap costs, unpaid rollback, resolution timing, no implied tap, inactive-player exclusion and one-step expiration |
| Homarid Warrior | TestRONSkipUntapHomaridWarrior: full record and definition validation; U payment, ordered shroud/restriction/tap actions, cleanup expiration, self return, control changes, phase-in and clone isolation |
| Barl's Cage | TestRONSkipUntapTargetLifecycle: source departure, target departure/return before resolution, phase-out before/after resolution, phase cycle, current-controller change, untapped-step consumption, other untap actions and repeated applications |
| Barl's Cage | TestRONSkipUntapCopiesAndInstances: concurrent separate factory instances, copy retargeting, independent pending targets, cloned resolution and expiration of repeated restrictions |
| Elvish Hunter | TestRONSkipUntapIllegalAnnouncementRollback: land/phased/shrouded/missing targets, wrong activating player, summoning sickness; retained mana/tap/stack state on failure |
| Synthetic spell, entry trigger, Aura and departed Aura | TestRONSkipUntapSharedContexts: shared action lowering, target placement, attachment departure LKI across Aura return and next-step expiration |
| Typed declarations | Extended TestRONPermanentActionsDeclareTypedReads: fresh declarations, typed-read metadata and engine rejection after declaration removal |
| Invalid declarations | TestRONSkipUntapInvalid: wrong arity/context/role, tuple symbols, undeclared host/target, incompatible player, cross-rule reference |
| Inventory | TestRONSkipUntapInventory and updated collection fixtures: permanent_reference operand role, exact collection counts and retained Lightning Bolt regression |

Three unique collection cards gain explicit complete-record gameplay evidence.
Synthetic contexts do not increase that count. Deep Spawn and Spore Cloud remain
rejected and are not full-card gameplay claims. Collection-wide behavior and
engine-validation totals remain unmeasured, separate from compilation counts.

### Measurements, validation and remaining gaps

Estimated and actual gains agree: **three unique names, three records**.
Compiled **1,065 -> 1,068/2,445**; rejected **1,380 -> 1,377**.
Accepted names **353 -> 356**; rejected names **633 -> 630**.
Fresh rejected-name comparison shows zero accepted-card regressions. Strict and
semantic issue arrays match. Occurrence/signature counts remain 72,938/8,528;
unknown-role occurrences fall **15,289 -> 15,285**. Only the operand role changed;
role assignment does not prove execution support.

Leading diagnostics remain optional-untap operation gaps (67 records), non-spell
CastEffect (53), rule If (48), ChooseADamageSource (46), Targeted_Modal (42),
optional-untap HostPermanent (39), upkeep HostController (35), ControlledByAPlayer
(29), TargetGraveyardCard (26) and WhenAPermanentBecomesTapped (25).
These are first diagnostics, not unlock estimates. No data defect was corrected,
no engine prerequisite was missing and no environment failure blocked checks.
All twelve source hashes match the vocabulary snapshot, so no RON-source
snapshot refresh is needed. Existing Gaea's Liege/Fastbond concerns remain open.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1; short engine tests; make build | PASS |
| Baseline go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,065 compiled, 1,380 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONSkipUntap -count=1 | Expected adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS after lint correction |
| go test ./pkg/mage/... -short -count=1 | PASS after lint correction |
| make test | PASS after lint correction |
| make build | PASS |
| go vet ./... | PASS |
| make lint | Initial local-name error corrected; final PASS, zero issues |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,068 compiled, 1,377 rejected; matching diagnostics |
| Accepted-record comparison and source-hash audit | PASS: zero regressions; twelve unchanged hashes |

Slice files: compiler.go, inventory.go, inventory_test.go,
permanent_scope_test.go, new skip_untap_test.go; dsl-reference.md,
engine-rules-contracts.md, this ledger and new mtgish-ron-next-untap.md.
No existing DSL function, Go definition, generator or startup caller was removed.
The cutover/removal gates remain closed; no deletion caller audit or native
startup cutover is claimed. Unrelated initial edits remain outside the commit.

Ranked next candidates to reassess with fresh full-record evidence:
(1) shared static untap prohibitions: Colossus of Sardia is close to complete;
Basalt Monolith and Brass Man should be compared for multi-card gains. Island
Fish Jasconius has combat/state-trigger gaps, Mana Vault also needs draw-step
TriggerI, and Time Vault needs turn replacement, so their frequency cannot be
counted as immediate unlocks;
(2) saved-sacrifice repeated mana with concrete casting-payment dependencies for
Priest of Yawgmoth, Sacrifice and Metamorphosis;
(3) shared trigger-condition lowering, including death history for Osai Vultures.
Complete collection gameplay, startup integration, Go-card/generator deletion
and DSL cleanup remain open. No second slice began.

## Run 2026-10-04: P7-SELF-STATIC-UNTAP

Status: complete self-only static untap prohibition adapter. This run stops after
this slice. Initial user changes in `.gitignore` and the migration design are
preserved and excluded from staging. No unrelated untracked files were present.

### Fresh evidence and candidate selection

Baseline focused tests, short engine tests and build passed. Strict and semantic
loading returned exit 1 for collection rejection: 1,068 compiled / 2,445 records,
1,377 rejected; 356 accepted names and 630 rejected names. Semantic inventory:
72,938 occurrences, 8,528 signatures, 15,285 unknown-role occurrences.
The vocabulary snapshot was read as syntax evidence, not proof of support.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers, downstream value and effort |
| --- | --- | ---: | --- |
| Self static untap prohibition, selected | Basalt Monolith; Brass Man; Colossus of Sardia; Island Fish Jasconius; Leviathan; Mana Vault; Time Vault | 3 | Island Fish needs attack restrictions and state triggers; Leviathan needs plural optional sacrifice and attack costs; Mana Vault needs draw-step TriggerI; Time Vault needs turn replacement/extra turns. Shared source-bound prerequisite for these cards. Small adapter, moderate lifecycle/payment tests. |
| Saved-sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 | Casting-payment scope is missing for Sacrifice; Metamorphosis also needs Plus and restricted mana. Shared paid-value lowering enables three cards after those prerequisites. Small activation adapter, moderate payment/solver tests; larger casting work. |
| Death-history trigger condition | Osai Vultures | 1 | Existing end-step/history/counter services are ready; missing condition adapter. Enables general intervening-if history lowering. Small adapter, moderate detection/resolution/history tests. |

The selected shape has three credible immediate unlocks for a small adapter
using an exact existing engine contract. It wins over the two single-card
immediate gains; no missing engine prerequisite requires a separate slice.
Selection, reuse boundary and acceptance cases were stated before implementation.
The initial commentary listed six self-rule cards; the full nested audit added
Leviathan, which has further blockers and does not change the unlock estimate.

All complete candidate records were read, including identical reprints. The
nested operation audit also read the complete records for these twelve names:
Arena of the Ancients, Cocoon, Glyph of Delusion, Goblin Rock Sled,
Magnetic Mountain, Meekstone, Merseine, Paralyze, Phyrexian Gremlins,
Tangle Kelp, Thelon's Curse, Venarian Gold. Together with the seven self-rule
names, these are all nineteen unique names containing the operation. Aura-host,
EachPermanentRuleEffect, conditional If, granted AddAbility, and temporary
CreatePermanentRuleEffectUntil contexts remain unsupported. Counts of nested
spellings are not full-card unlocks. No Go card definition supplied requirements.

### Exact contract, dependencies and reuse

`compileRule(Node, scope) (optionFactory, error)` delegates permanent rule tuple
`PermanentRuleEffect/2` to
`compileSelfUntapRule(Node) (optionFactory, error)` (renamed from
`compileOptionalUntap`). Accepted operands are symbol `ThisPermanent` and a
one-element list containing symbol `DoesntUntapDuringControllersUntap` or the
retained symbol `MayChooseNotToUntapDuringUntap`. This is a top-level permanent
rule context only. It rejects tuple spellings, bad arity/lists, mixed operations,
unbound targets, host/player reads, costs/actions, instant/sorcery rules, and
conditional wrappers. Diagnostics now describe self untap sources/operations
because the compiler is no longer limited to optional untap.

The immutable plan captures only `core.Attr`; each factory call creates its own
`mage.Static(mage.While(condition,
mage.GrantKeywordModification(mage.This, attribute)))` ability. The condition
uses the existing pure StateConditionFunc with StateBattlefield dependency.
RON owns spelling, shape validation and lowering. The engine owns static source
lifetime, ability layers, current control, phase-in, untap selection, transactional
payments, copies, incarnation identity and clones. Existing source references
are typed `mage.This`, with no mutable choices shared by compiler factories.
Inventory already assigns permanent_rule_operation to this list; no role adapter
change is needed and inventory coverage totals are unchanged.

Contracts read: engine doc.go, composable continuous modifications/Static/While,
turn execution, current API and rules-contract documentation, CR 400.7,
502.1/3/4, 604.2, 613.1f and 702.26b/d. No engine rule was added. No parallel
rules service, Go fallback, card-name branch, global last-action slot, runtime
declaration panic, or partial production registry was introduced. No new LKI
query, replacement ordering or simultaneous batch operation is part of this
shape. Existing payment and source identity services retain their contracts.

### TDD and card-to-test mapping

Meaningful tests were written first. The first test build found incorrect test
getter names; these were corrected to CurrentPower/CurrentToughness before the
red run. The red run then failed on the missing adapter for all three complete
real records, with unsupported optional untap operations. Invalid declarations
already remained rejected. Focused tests passed after implementation. A native
upkeep regression was added and passed; no assertion was weakened.

| Unique card or contract | Test and evidence |
| --- | --- |
| Basalt Monolith | TestRONStaticUntapBasaltMonolith: full LEB record/definition validation, immediate three colorless mana, ordinary untap prohibition, inactive-player exclusion, repeated untap steps, stack-based paid untap timing, insufficient payment rollback, instance identity, clone resolution, other untap effects |
| Brass Man | TestRONStaticUntapBrassMan: full ARN record/definition validation, controller upkeep only, payment at resolution, accept/decline/unaffordable branches, unchanged static rule after paid untap, controller change, departed/returned incarnation, phased source, copied trigger with another paying controller |
| Brass Man | TestRONStaticUntapNativeUpkeep: native untap/upkeep steps, payment offer, mana spent and untap resolution |
| Colossus of Sardia | TestRONStaticUntapColossus: full ATQ record/definition validation, trample and 9/9, ordinary untap prohibition, own-upkeep activation timing, wrong step/turn/controller, insufficient payment and phased-source rollback, stack timing, old source on return, copies and controller change |
| Basalt Monolith lifecycle | TestRONStaticUntapLifecycle: current controller, phase-in before untap, ability removal and cleanup restoration, return reinstalls restriction, another factory instance and clone stay independent |
| Invalid declarations | TestRONStaticUntapInvalid: malformed arity/operand/list, tuple references/operations, unbound target/host, player operand, mixed operations, wrong cost/action/spell/conditional contexts |
| Accepted collection | Updated TestInventoryRepositoryCollection and TestInventoryLeavesRealRONGameplayAndRegistryUnchanged: exact new counts, retained real Lightning Bolt execution and unchanged registry |

Three unique collection cards gain complete-record engine execution evidence
for the slice and their other declared abilities. This does not prove every
possible combat/layer interaction. Compilation totals and gameplay evidence
remain separate; collection-wide engine_validated/behavior_verified are null.
The four other self-rule cards and twelve nested-context cards remain rejected.

### Measurements, checks and remaining work

Estimated and actual unlocks agree: **three unique names, nine records**
(Basalt Monolith six, Brass Man two, Colossus of Sardia one).
Compiled **1,068 -> 1,077/2,445**; rejected **1,377 -> 1,368**.
Accepted names **356 -> 359**; rejected names **630 -> 627**.
Fresh name comparison finds zero accepted-record regressions. Strict and semantic
issues match. Occurrences/signatures/unknown roles remain 72,938/8,528/15,285.
No source defect was corrected. All twelve source hashes match the vocabulary
snapshot, so a source refresh is not required. Existing Gaea's Liege/Fastbond
source concerns remain open. All gaps in this slice were compiler adapters;
no engine contract or environment failure blocked implementation.

Leading final first diagnostics: rule If (53), non-spell CastEffect (53),
ChooseADamageSource (46), Targeted_Modal (42), self untap HostPermanent (39),
upkeep HostController (35), other self untap operations (34),
ControlledByAPlayer (29), TargetGraveyardCard (26),
WhenAPermanentBecomesTapped (25), CreateFuturePreventDamage (24),
Modal_ChooseOne (23). These are first-error counts, not full unlock estimates.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1; go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,068 compiled, 1,377 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONStaticUntap -count=1 | Expected missing-adapter failures before implementation |
| go test ./pkg/mage/dsl/ron -run 'TestRONStaticUntap\|TestRONStorage' -count=1 | PASS after implementation |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS after final code/lint changes |
| go test ./pkg/mage/... -short -count=1 | PASS after final code/lint changes |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS, zero issues; automatic formatting reviewed, no unrelated changes |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,077 compiled, 1,368 rejected; matching issues |
| Fresh accepted-record and source-hash audits | PASS: zero accepted-record regressions; twelve unchanged source hashes |

Slice files: compiler.go, storage_land.go, inventory_test.go, new
static_untap_test.go; dsl-reference.md, engine-rules-contracts.md, this ledger.
Required checks pass; this slice is ready for its descriptive commit. User
changes in the design and .gitignore remain outside the commit. No production
Go definitions, startup callers, generators or DSL exports were removed.
The cutover and removal gates remain closed. No deletion audit or runtime
cutover is claimed; the private helper rename has no other repository callers.

Ranked next candidates to reassess from fresh full records:
1. Shared collection static untap rules for Meekstone, Magnetic Mountain,
   Thelon's Curse and Arena of the Ancients. Meekstone is the closest unlock;
   others need player-event choices/payments or legendary collection tapping.
   Compare shared filter prerequisites and engine collection layer behavior
   against modal targeting before selecting an exact shape.
2. Shared paid-sacrifice mana for Priest of Yawgmoth, Sacrifice and Metamorphosis,
   including casting-payment scope and restricted mana as explicit dependencies.
3. Shared trigger-condition lowering, including Osai Vultures' death history and
   Mana Vault's draw-step tapped condition. Reassess event-context reuse before
   choosing a single-card adapter.

Complete collection gameplay, runtime loading, Go-card/generator removal and
DSL cleanup remain unfinished. No second slice began.

## Run 2026-10-04: P7-GLOBAL-LANDWALK-BLOCKING

Status: complete global landwalk blocking override slice. This run stops here.
Initial user changes in `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from staging. No unrelated untracked files were present.

### Fresh evidence and candidate comparison

Read AGENTS.md, engine doc.go, the current migration design, vocabulary snapshot,
ledger, selected compiler/engine code, and complete candidate RON records.
Requirements came from RON; no Go card definition supplied requirements.
Baseline focused RON tests, short engine tests and make build passed. Strict
and semantic loaders returned expected exit 1: 1,077 compiled and 1,368 rejected
of 2,445 records; 359 accepted and 627 rejected unique names. There are 986
unique names, 72,938 expression occurrences and 8,528 signatures. Unknown-role
occurrences were 15,285. First diagnostics were used to find candidates;
full nested records supplied the unlock estimates.

| Candidate | Distinct affected names | Estimated complete-card unlocks | Other blockers, downstream value and effort |
| --- | --- | ---: | --- |
| Global landwalk blocking, selected | Crevasse; Deadfall; Gosta Dirk; Great Wall; Lord Magnus; Quagmire; Undertow; Ur-Drago | 8 | No other blockers in these full records. Reuses exact global engine rule and existing landwalk validation. Enables global rule operation lowering and semantic operand roles. Small adapter, moderate lifecycle/legality/native combat tests. |
| Collection static untap prohibition | Meekstone; Magnetic Mountain; Thelon's Curse; Arena of the Ancients | 1 | Meekstone needs numeric PowerIs/GreaterThanOrEqualTo lowering as well as the collection adapter. Magnetic Mountain and Thelon's Curse need AnyPlayer upkeep, player-local choices, per-object payments and chosen collections. Arena needs IsSupertype for both static and tap collections. Enables shared untap rules for four cards; moderate adapter/filter work and lifecycle tests, larger payment follow-up. |
| Saved-sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 | Priest needs bound mana amount/solver lowering. Sacrifice needs additional casting-payment scope; Metamorphosis also needs Plus and restricted mana. Shared paid-value work enables three cards after explicit prerequisites. Small activation adapter, moderate payment/solver tests; larger casting work. |

All listed complete records were read, including Meekstone and Sacrifice's
identical reprints in the six base sets. The AST semantic inventory confirms
nine CanBeBlockedAsThoughItDidntHave occurrences across exactly eight names;
Lord Magnus has two rules. No hidden action/trigger/cost blocker exists in the
selected records. The selected adapter gives the largest credible immediate
multi-card gain for this effort. It does not depend on an unimplemented engine
contract. Selection, reuse boundary and acceptance cases were stated before
editing. No second slice was started.

### Exact supported contract and ownership

Capability P7-GLOBAL-LANDWALK-BLOCKING is complete for this exact shape/context:

- Top-level permanent rule tuple `EachPermanentRuleEffect/2`.
- Filter tuple `And/1` containing a list of exactly `IsCardtype(Creature)` and
  `HasAbility(Landwalk(IsLandType(subtype)))`, in either order.
- Operation list containing exactly tuple `CanBeBlockedAsThoughItDidntHave/1`
  with `Landwalk(IsLandType(subtype))`; the subtype must match the filter.
- Plains, Island, Swamp, Mountain, Forest and Desert operands are supported.

Private signatures: `compileLandwalkBlockingRule(Node) (optionFactory, error)`
and `validateGlobalLandwalkFilter(Node, core.Attr) error`. `compileRule` delegates
this family to the new adapter. RON owns spelling, arity/kind validation,
filter-operation compatibility, diagnostics and inventory roles. It captures
only an immutable core.Attr and constructs a fresh effect per factory call.
There are no chosen bindings, targets, costs or mutable declaration instances
in this shape. Existing `WithStaticAbility(NullifyLandwalkEffect(keyword))`
provides the exact global rules behavior, source lifetime, clone isolation and
ability-removal handling. Blocking checks use HasLandwalkEvasion with CanBlock;
the landwalk keyword stays on the creature and other evasion remains active.

The guard rejects narrower filters because the reused engine rule is global.
It does not silently discard controller/color/source restrictions. Other
EachPermanentRuleEffect operations, arbitrary landwalk filters, temporary or
self-only contexts remain unsupported. No global last-action slot, card-name
branch, Go fallback, runtime declaration panic or partial production registry
was added. No engine service was changed. No new LKI, cost rollback,
replacement-order or simultaneous-operation contract applies to this static
shape. CR 509.1b, 702.14a-e, 604.2, 400.7 and 702.26b/d were consulted.

Semantic roles now include EachPermanentRuleEffect's permanent_filter and
permanent_rule_operation operands, the override's keyword operand, and
HasAbility/DoesntHaveAbility keyword operands in permanent filters. Roles
continue through Landwalk/IsLandType to land_subtype. This is syntax reporting,
not a claim of general filter execution support.

### TDD and card-to-test evidence

Tests were written first. An initial test build found the incorrect test helper
name WithSubtype; it was corrected to WithSubTypes before the red run. The red
run then failed for all eight real records and six synthetic subtypes with
unsupported EachPermanentRuleEffect, plus missing semantic roles. Invalid
forms stayed rejected. The adapter and role lowering made the focused tests
pass. Later tests added native combat and another landwalk evasion check.
No assertion was weakened.

| Cards or contract | Evidence |
| --- | --- |
| Crevasse, Deadfall, Gosta Dirk, Great Wall, Lord Magnus, Quagmire, Undertow, Ur-Drago | TestRONLandwalkRuleCards: each complete LEG record compiles and passes ValidateCardDefinition; all five basic landwalks checked against both defenders; only declared subtypes lose evasion; printed keywords retained; Gosta Dirk/Lord Magnus/Ur-Drago P/T and first strike; Lord Magnus's two overrides; source control, phase cycle, ability removal/restoration, departure/return and clone isolation |
| Deadfall and existing Hidden Path grant | TestRONLandwalkRuleNativeCombat: native attacker/blocker declaration and combat damage with/without Deadfall; real RON forestwalk grant remains on attacker; a legal block survives Deadfall departure before damage |
| Six subtype shapes, instances and legality | TestRONLandwalkRuleInstancesAndLegality: Plains/Island/Swamp/Mountain/Forest/Desert; two factory instances; one departure leaves the other active; last departure restores evasion; late-entry flying remains restrictive; another landwalk still prevents blocking |
| Invalid declarations and contexts | TestRONLandwalkRuleInvalid: bad arity, missing/list shapes, mismatched subtype, invalid nested operand, wrong ability, empty/mixed operations, extra color/controller/source filters, disjunction, self and spell/action contexts; both instant and enchantment declaration contexts |
| Inventory and accepted collection regression | TestRONLandwalkRuleInventory checks nested semantic roles and supported compile count; updated TestInventoryRepositoryCollection and TestInventoryLeavesRealRONGameplayAndRegistryUnchanged retain exact counts and real Lightning Bolt execution |

Eight unique collection cards gain explicit complete-record behavior evidence.
Synthetic cards are contract coverage only. Full collection gameplay totals and
engine-validation totals remain unmeasured/null, separate from compilation.

### Measurements, validation and remaining work

Estimated versus actual gain: **eight unique names and eight records**, exactly
as predicted. Compiled **1,077 -> 1,085/2,445**; rejected **1,368 -> 1,360**.
Accepted unique names **359 -> 367**; rejected names **627 -> 619**. Fresh
rejected-name comparison finds exactly the eight selected names newly accepted
and zero regressions. Strict and semantic issue arrays match. Occurrences and
signatures remain 72,938/8,528; unknown-role occurrences fall **15,285 -> 15,073**.
No RON source changed; all twelve source SHA256 values match the vocabulary
snapshot, so no source snapshot refresh is needed. No source defect was
corrected. Existing Gaea's Liege/Fastbond concerns remain open. This slice's
missing behavior was a compiler adapter, not an engine or environment failure.

Leading final first diagnostics: rule If (53 records), non-spell CastEffect (53),
ChooseADamageSource (46), Targeted_Modal (42), self untap HostPermanent (39),
upkeep HostController (35), other self untap operations (34),
ControlledByAPlayer (29), TargetGraveyardCard (26),
WhenAPermanentBecomesTapped (25), CreateFuturePreventDamage (24),
Modal_ChooseOne (23). These remain first-error counts, not unlock estimates.

| Command | Result |
| --- | --- |
| Baseline focused RON tests, short engine tests, make build | PASS |
| Baseline strict and semantic loader | Expected exit 1: 1,077 compiled, 1,368 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONLandwalkRule -count=1 | Expected missing-adapter/role failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS including native combat and final legality cases |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS, zero issues; automatic formatting reviewed |
| git diff --check | PASS |
| Final strict and semantic loader | Expected exit 1: 1,085 compiled, 1,360 rejected; matching issues |
| Accepted-record and source-hash audits | PASS: zero regressions, twelve unchanged source hashes |

Changed files: compiler.go, inventory.go, inventory_test.go, new
landwalk_rule.go and landwalk_rule_test.go; dsl-reference.md,
engine-rules-contracts.md and this ledger. User changes remain outside the
commit. No Go definitions, generators, startup callers or DSL exports were
removed: complete loading, gameplay/test migration and startup gates are not
met. No deletion caller audit or native registry startup cutover is claimed.

Ranked next candidates to reassess:
1. Modal_ChooseOne with mode-local typed targets: Blue Elemental Blast and Red
   Elemental Blast have supported counter/destroy actions; Healing Salve also
   needs future prevention. Compare full records and engine modal services.
2. Collection static untap/filter prerequisites for Meekstone and Arena of the
   Ancients; compare two-card gain after numeric and supertype filter work.
   Magnetic Mountain and Thelon's Curse still need player choice/payment scopes.
3. Bound sacrifice mana for Priest of Yawgmoth, Sacrifice and Metamorphosis,
   with casting-payment and restricted-mana dependencies kept explicit.

Complete collection gameplay, runtime loading, Go-card/generator removal and
DSL cleanup remain unfinished. This run stops after this slice.

## Run 2026-10-04: P2-CHOOSE-ONE-SPELL-MODES

Status: complete choose-one spell adapter slice. This run stops here.
Initial user changes in `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from staging. No unrelated untracked files were present.

### Fresh evidence and selection

Read AGENTS.md, pkg/mage/doc.go, the migration design, vocabulary snapshot,
ledger, selected compiler and engine services, and complete candidate RON
records. No Go card definitions supplied requirements. Baseline focused RON
tests, short engine tests and make build passed. Strict and semantic loaders
returned expected exit 1: 1,085 compiled and 1,360 rejected records out of
2,445; 367 accepted and 619 rejected unique names out of 986. AST inventory
reported 72,938 occurrences, 8,528 signatures and 15,073 unknown-role
occurrences. Nested forms and complete records supplied unlock estimates.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers, downstream capabilities and estimated effort |
| --- | --- | ---: | --- |
| Choose-one spell modes, selected | Blue Elemental Blast; Red Elemental Blast; Active Volcano; Flash Flood; Healing Salve; Alabaster Potion | 2 | Both Blasts have supported typed spell/permanent targets and counter/destruction actions. Volcano/Flood need PutPermanentIntoItsOwnersHand. Salve/Potion need targeted player life and future prevention. Enables independent spell mode scopes, announcement and copies for six names. Small adapter; moderate declaration, legality, copy and payment tests. |
| Collection static untap rules | Meekstone; Arena of the Ancients; Magnetic Mountain; Thelon's Curse | 2 after prerequisites | Meekstone needs PowerIs/GreaterThanOrEqualTo; Arena needs IsSupertype and collection tapping. Mountain/Curse also need AnyPlayer upkeep, player-local choices, per-object payment and chosen collections. Enables shared four-card untap rules. Moderate adapter/filter and batch-action work; larger payment follow-up. |
| Bound sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 after prerequisites | Priest needs bound amount and mana solver metadata. Sacrifice needs additional casting-payment scope; Metamorphosis also needs Plus and restricted mana. Enables shared paid-value mana for three names. Small activation adapter with moderate payment/solver tests; larger casting follow-up. |

The selected shape has the lowest dependency cost for a credible two-card
unlock. The ledger's suggested priority was reassessed. All twelve Blast
reprints and the four other spell-mode records were read. The complete nested
records for Pyramids, Balm of Restoration, Relic Bind and Triassic Egg were
also read: these use activation/trigger contexts outside this slice and have
additional prevention, replacement, host-event, choice or modifier blockers.
All collection-untap and sacrifice-mana candidate records were read, including
identical reprints. Modal_ChooseOne's first-error count was not treated as a
full-card unlock estimate. Selection, reuse boundary, exact contract and
acceptance cases were stated before editing.

### Exact contract, signatures and reuse boundary

Capability P2-CHOOSE-ONE-SPELL-MODES is complete for
`SpellActions(Modal_ChooseOne([mode, mode, ...]))` on instants and sorceries.
The constructor has one positional list argument and at least two modes.
Each mode uses the existing supported ActionList/1 or Targeted/2 action-plan
shape, with at most one target declaration. Every mode body must compile.
Nested modal bodies, permanent SpellActions, activation/trigger modes and
Targeted_Modal remain errors. Unsupported actions do not become accepted
because their parent is modal.

Private adapter signature:
`compileModalSpell(Node, scope) (optionFactory, error)`.
`scope.targetNamespace string` supplies distinct internal declaration names
(`mode1/Ref_TargetPermanent`, etc.). RON spelling and lexical reads remain
unchanged. `compileTarget(Node, scope) (targetFactory, string, error)` uses
that namespace for permanent, player and spell declarations; its existing
callers have an empty namespace and retain their names. Each mode clears
inherited target plans, compiles independently, and builds fresh declarations
and effects inside each factory call. Plans capture no chosen state.

`actionPlan.build` and typed target bindings supply the existing group mapping.
`mage.NewModalSpell` receives mode targets/effects and labels from effect
sequence text with readable target names. Existing engine services choose mode/targets at announcement,
pay costs transactionally, keep mode/targets on the stack, recheck legality,
resolve only the chosen body, and copy/retarget without selecting a new mode.
`CounterSpellRef` and `Destroy` continue to use stack/permanent identity and
shared zone/regeneration services. No engine execution service was added.
There is no Go fallback, card-name branch, mutable shared target state,
last-action slot, partial production registry or game-time declaration panic.
The minimum mode count is checked before factory construction.

Semantic reporting propagates `actions` through Modal_ChooseOne's mode list,
including descendants in unsupported contexts. This is syntax-role evidence,
not general activation/trigger support. No new LKI producer, simultaneous
operation or replacement-order contract is introduced. Existing regeneration
is explicitly exercised. CR 601.2b-c, 608.2b, 700.2a/c/f/g, 400.7 and
702.26b were read with the selected engine/API contracts.

### TDD and card-to-test mappings

Initial tests were written before implementation. The first red run found a
fixture casting a creature on the wrong player's main phase; the fixture was
corrected. The confirmed red run then failed all 24 real Blast set/mode cases
and the synthetic instance test with unsupported Modal_ChooseOne. Invalid
forms stayed rejected. The first implementation exposed duplicate engine
binding names across two modes with the same RON spelling. Private mode
namespaces fixed that compiler lowering issue without changing engine rules.
A label test first failed when compiler binding names appeared in the mode
menu. Labels now use readable target names.
A later synthetic targeted-life test exposed an existing adapter gap; the test
was changed to supported explicit-player drawing to test mode scopes. Targeted
life remains an explicit blocker, not a claimed behavior. No production action
or existing test was weakened to accept it.

| Card or contract | Tests |
| --- | --- |
| Blue Elemental Blast; Red Elemental Blast | TestRONModalBlastCards executes both modes from LEA, LEB, 2ED, 3ED, CED and CEI; complete records and engine definition validation, mode-local targets at announcement, counter/destruction, owner graveyards, clone isolation and no new resolution mode choice |
| Blue Elemental Blast legality and payment | TestRONModalAnnouncementRollback: wrong-mode target, wrong color, invalid choice and unpaid cast retain hand/mana/stack state |
| Blue Elemental Blast resolution | TestRONModalResolutionLegality: spell/permanent color changes, spell recast, permanent leave/return, phased target restoration, regeneration shield and uncounterable spell |
| Blue Elemental Blast copies | TestRONModalCopyTargets: engine copy with new permanent target retains mode, does not reprompt mode, leaves original target intact, moves only appropriate victims, copy leaves no card in graveyard, clone remains isolated |
| Mode/factory identity | TestRONModalInstances: same reference spelling in two modes gives distinct declarations; second factory has fresh declarations; both definitions validate; labels hide compiler binding names |
| Untargeted and player-target mode contracts | TestRONModalUntargetedAndPlayerModes: only chosen targets are required; player draw affects the explicit opponent; untargeted life gain affects caster |
| Invalid declarations | TestRONModalInvalid: wrong kind/arity/list, fewer than two modes, cross-mode and incompatible references, unknown body, nested modes, permanent SpellActions and activation context |
| Inventory and accepted collection | TestRONModalInventory verifies nested roles; updated repository inventory counts and existing Lightning Bolt/registry regression tests retain accepted gameplay and strict loading behavior |

Two unique collection cards gain complete-record RON behavior evidence.
Synthetic cards establish contracts only. Compilation is tracked separately
from full-collection gameplay and engine-validation totals, which remain
unmeasured/null in the inventory.

### Final measurements and validation

Estimated versus actual: **two unique names and twelve records**, exactly as
predicted. Compiled **1,085 -> 1,097/2,445**; rejected **1,360 -> 1,348**.
Accepted unique names **367 -> 369**; rejected names **619 -> 617**.
Fresh record/file and unique-name rejection comparisons show exactly the
Blast unlocks and zero accepted-record regressions. Strict/semantic issue
arrays match. Occurrences/signatures remain 72,938/8,528; unknown-role
occurrences fall **15,073 -> 14,661**. All twelve source SHA256 values match
the initial inventory and vocabulary snapshot. RON source did not change,
so no vocabulary source snapshot refresh was needed.

Missing behavior was compiler lowering, not missing engine execution or an
environment failure. No source defect was corrected. Existing Gaea's Liege,
Fastbond and other unresolved source questions remain outside this slice.
Healing Salve/Alabaster Potion now expose targeted life as their first error;
Active Volcano/Flash Flood expose return-to-hand. These are explicit remaining
gaps. Leading first diagnostics remain rule If (53), non-spell CastEffect (53),
ChooseADamageSource (46), Targeted_Modal (42), self untap HostPermanent (39),
upkeep HostController (35), other self untap operations (34),
ControlledByAPlayer (29), TargetGraveyardCard (26),
WhenAPermanentBecomesTapped (25) and CreateFuturePreventDamage (24).

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron -count=1 | PASS |
| Baseline go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline strict and semantic loader | Expected exit 1: 1,085 compiled, 1,360 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONModal -count=1 | Confirmed missing-adapter failures first; final PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS; repeated after final label and lint changes |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | Initial test-fixture ineffassign; removed unused assignment, reviewed automatic formatting; rerun PASS with zero issues |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,097 compiled, 1,348 rejected; matching issues |
| Fresh accepted-record and source-hash audits | PASS: zero regressions; twelve unchanged source hashes |

Changed files: compiler.go, inventory.go, inventory_test.go, new modal_spell.go
and modal_spell_test.go; dsl-reference.md, engine-rules-contracts.md and this
ledger. The slice is ready for one descriptive commit. User changes remain
outside staging. No Go definitions, generators, startup callers or DSL exports
were removed; complete loading, gameplay/test migration and startup gates are
not met. No deletion caller audit or full registry runtime cutover is claimed.

Ranked next candidates to reassess from fresh evidence:
1. Shared return-to-owner-hand action: Active Volcano and Flash Flood now have
   supported modes and targets. Inspect all other complete records using this
   action to estimate the larger shared gain and owner/incarnation contracts.
2. Targeted player life plus future prevention for Healing Salve and Alabaster
   Potion, with Balm of Restoration's activation-mode context kept separate.
   Compare life adapters against a shared prevention prerequisite across the
   collection before choosing scope.
3. Collection untap/filter prerequisites for Meekstone and Arena of the
   Ancients; Magnetic Mountain and Thelon's Curse still need payment choices.

Complete collection gameplay, runtime loading, Go-card/generator removal and
DSL cleanup remain unfinished. No second slice began.

## Run 2026-10-04: P5-RETURN-OWNER-HAND

Status: complete singular permanent return action slice. This run stops here.
Initial user changes in `.gitignore` and `mtgish-ron-execution.md` are preserved
and excluded from staging. There were no unrelated untracked files.

### Fresh evidence and selection

Read AGENTS.md, pkg/mage/doc.go, migration design, vocabulary snapshot, ledger,
selected compiler/engine/API contracts, CR index and relevant sections, and all
complete candidate and affected RON records. No Go card definitions supplied
requirements. Baseline focused RON/CLI tests, short engine tests and make build
passed. Baseline strict/semantic loaders returned expected exit 1: 1,097 compiled
and 1,348 rejected records; 369 accepted and 617 rejected unique names.
Nested inventory: 72,938 occurrences, 8,528 signatures, 14,661 unknown-role
occurrences. Reprints are counted once in the comparison.

| Candidate | Distinct affected names | Estimated full-card unlocks | Blockers, downstream benefit and effort |
| --- | --- | ---: | --- |
| Singular return to owner's hand, selected | Unsummon; Boomerang; Active Volcano; Flash Flood; Obelisk of Undoing; Karakas; Time Elemental; Witch Hunter; Rakalite | 4 | First four have fully supported enclosing actions, modes and filters. Obelisk needs owner/controller filters; Karakas needs supertype filter; Time Elemental needs combat/delayed triggers and unenchantment filter; Witch Hunter needs player/permanent target union and opponent filter; Rakalite needs prevention and delayed triggers. Shared action enables all nine names when these dependencies arrive. Small adapter; moderate identity, zone, copy and declaration tests. |
| Player-target life and future prevention | Healing Salve; Alabaster Potion | 2 after both adapters | Mode infrastructure exists, but both life and future prevention must work before either card unlocks. Prevention also benefits Rakalite. Larger adapter and damage shield/replacement test scope than return. Life alone unlocks neither of these two cards. |
| Collection static untap restrictions | Meekstone; Arena of the Ancients; Magnetic Mountain; Thelon's Curse | 2 after prerequisites | Numeric/supertype filters and collection tapping needed for the first two. Mountain/Curse need upkeep player, per-object payment choices and chosen collections. Shared untap restrictions enable four names. Moderate filter/action work and larger payment follow-up. |

The return action gives the largest credible immediate gain for the least
implementation effort. Fresh nested inspection found 14 occurrences across four
contexts: seven direct spell, two modal spell, four activation and one delayed
trigger under an unsupported parent. All nine unique complete records, all six
Unsummon reprints, and all candidate records/reprints were inspected. First-error
counts alone did not select the slice. Contract, reuse boundary, selection and
acceptance cases were stated before edits.

### Contract and reuse boundary

Exact signature: action `PutPermanentIntoItsOwnersHand/1`, one positional
permanent-reference argument. Accepted references: symbol `Ref_TargetPermanent`
within its single typed target group, `ThisPermanent` for a permanent source,
and `HostPermanent` for an Aura source. Existing `compilePermanentRef(Node,
scope) (permanentReferencePlan, error)` rejects wrong types, shapes and scopes.
`compileEffect(Node, scope) ([]effectFactory, error)` builds
`mage.ReturnToHand(subject.build(instanceBindings))`. Per-factory declarations
and action bindings stay fresh; immutable plans capture no chosen game state.

Supported enclosing contexts: existing ActionList in instant/sorcery spell
rules, choose-one spell modes, activations and supported triggers. Unsupported
parents such as CreateFutureTrigger remain rejected. The action does not make
those enclosing contexts supported. No source data, public API or engine
execution service changed. This is a missing compiler adapter, not a missing
engine contract, data defect or environment failure.

Reuse: typed permanent references, instanceBindings, definition validation,
transactional casting/activation, stack target checks, modal copies,
ReturnToHand, BouncePermanentToHand, battlefield incarnation identity,
attachment LKI and Game.Clone. Owner is independent of controller; return is
independent of regeneration. An absent/phased object cannot move; phase-in
preserves identity, while departure/return creates a new object. Host reads use
current attachment or the captured departed source's last attachment, and
cannot follow a returned source. Existing rules: CR 400.7, 608.2b, 702.26b,
700.2a/c/f/g. No new simultaneous-operation or replacement-order contract is
introduced. Existing engine token/leave-trigger tests remain green.

No Go fallback, card-name branch, global last-action slot, shared mutable
bindings, game-time declaration panic or partial production registry was added.
No Go definitions, generators, startup wiring or DSL exports were removed;
complete loading, gameplay migration and startup removal gates remain unmet.
A DSL deletion caller audit is not claimed because no deletion was attempted.

### TDD and card-to-test mappings

Tests were written first. Initial test builds found incorrect test API names
for duration/type effects; these fixtures were corrected to core.Indefinite and
TemporaryKeyword with core.Shroud. The confirmed red run failed all eleven real
set/mode cases and identity/copy/rollback/reporting cases with unsupported
PutPermanentIntoItsOwnersHand; invalid declarations remained rejected.
The adapter and role lowering made those tests pass. Additional source-context
fixtures initially omitted the Aura subtype; declaring Aura fixed the fixture,
without changing production scope rules. No assertions were weakened.

| Card or context | Tests and evidence |
| --- | --- |
| Unsummon | TestRONReturnHandCards: all six source reprints, creature target, return to owner with different controller, cast cost, graveyard, clone isolation |
| Boomerang | TestRONReturnHandCards: complete LEG record, noncreature land target, owner hand and spell graveyard, clone isolation |
| Active Volcano; Flash Flood | TestRONReturnHandCards: both modes, blue/red destruction and Island/Mountain return, mode chosen once, correct owner zones, clone isolation |
| Unsummon identity/legality | TestRONReturnHandIdentityAndLegality: control change, departure/return, absence, phase-out and phase cycle, later shroud, regeneration independence |
| Active Volcano copy | TestRONReturnHandCopyAndInstances: retained return mode, new target without a mode choice, original unaffected, copy leaves no graveyard card, clone isolation |
| Declaration identity | TestRONPermanentActionsDeclareTypedReads extended for return: fresh factory declarations, typed read metadata and rejection by engine validation if target declaration is removed; TestRONReturnHandCopyAndInstances also validates separate instances |
| Shared owner behavior | TestRONUnaryPermanentActionsUseSharedRules extended with explicit return assertions for either player's permanent |
| Cast rollback | TestRONReturnHandRollback: unpaid cast, wrong creature type, wrong land subtype and phased target preserve hand/mana/stack/graveyard |
| Source/attachment contracts | TestRONReturnHandSourceAndHostContexts: activation tap cost and entry trigger self return; Aura sacrifice payment before return, current attachment, departed source LKI, returned host rejection and returned source isolation |
| Invalid declarations | TestRONReturnHandInvalid: symbol/zero/two arguments, list/tuple/wrong-type reference, spell source/host/event reference, undeclared/cross-mode/wrong-target reads |
| Inventory/regressions | TestRONReturnHandInventory: permanent-reference role; updated full-collection counts and retained real Lightning Bolt/registry regression |

Four unique collection cards gain complete-record behavior evidence. Synthetic
cards test contracts only. Compilation totals remain distinct from unmeasured
full-collection gameplay and engine-validation totals (null in the inventory).

### Measurements and validation

Estimated versus actual: **four unique names and nine records**, as predicted.
Compiled **1,097 -> 1,106/2,445**; rejected **1,348 -> 1,339**. Accepted unique
names **369 -> 373**; rejected unique names **617 -> 613**. Fresh rejection
comparisons find exactly Unsummon, Boomerang, Active Volcano and Flash Flood
newly accepted, and zero regressions. Strict/semantic issue arrays match.
AST occurrences/signatures remain 72,938/8,528; unknown-role occurrences fall
14,661 -> 14,648. All twelve source file hashes are unchanged and match the
vocabulary snapshot. RON source did not change, so no vocabulary refresh was
needed. No source defect was corrected; earlier Gaea's Liege/Fastbond concerns
remain open.

Leading first diagnostics: rule If (53 records), non-spell CastEffect (53),
ChooseADamageSource (46), Targeted_Modal (42), self untap HostPermanent (39),
upkeep HostController (35), other self untap operations (34),
ControlledByAPlayer (29), TargetGraveyardCard (26),
WhenAPermanentBecomesTapped (25), CreateFuturePreventDamage (24).
These are first-error counts, not full-card unlock estimates.

| Command | Result |
| --- | --- |
| Baseline focused RON/CLI tests, short engine tests, make build | PASS |
| Baseline strict/semantic loader | Expected exit 1: 1,097 compiled, 1,348 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONReturnHand -count=1 | Confirmed missing-adapter failures first; final PASS |
| Focused return and typed-reference tests with -count=1 | PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS; repeated after lint and final test assertions |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | Initial revive error for test variable copy; renamed to copied, reviewed automatic import formatting; final PASS |
| git diff --check | PASS |
| Final strict and semantic loader | Expected exit 1: 1,106 compiled, 1,339 rejected; matching issues |
| Accepted-record/source audits | PASS: zero regressions and twelve unchanged hashes |

Changed files: compiler.go, inventory.go, inventory_test.go,
permanent_scope_test.go, new return_hand_test.go, dsl-reference.md,
engine-rules-contracts.md and this ledger. Only these slice files are staged
for one descriptive commit. User changes remain outside staging.

Ranked next candidates to reassess:
1. Shared targeted-player life action: inspect all complete records, especially
   Stream of Life and Relic Bind; mode life in Healing Salve/Alabaster Potion
   still requires prevention before full-card acceptance.
2. Shared future prevention for Healing Salve, Alabaster Potion and Rakalite;
   compare damage-recipient scopes and delayed-trigger dependencies.
3. Collection untap/filter prerequisites for Meekstone/Arena of the Ancients;
   compare IsSupertype's additional Karakas gain now that return is supported.

Complete RON gameplay, runtime loading, Go-card/generator deletion, DSL cleanup
and native startup validation remain unfinished. No second slice began.

## Run 2026-10-04: P0-TYPED-DAMAGE-RECIPIENT

Status: complete single mixed damage-recipient binding prerequisite. This run
stops after this slice. Initial changes in `.gitignore` and the execution design
are preserved and excluded from staging. No unrelated untracked files existed.

### Fresh evidence and candidate comparison

Read AGENTS.md, pkg/mage/doc.go, execution design, vocabulary snapshot, ledger,
selected compiler/engine code and API/rules contracts. Used the positioned AST
inventory to find names and nested contexts, then read complete candidate and
affected RON records, including identical reprints. No Go card definitions were
used as requirements. Baseline RON/CLI tests, short engine tests and make build
passed. Strict/semantic loaders returned expected exit 1: 1,106 compiled,
1,339 rejected; 373 accepted and 613 rejected unique names.

| Candidate | Distinct affected names | Estimated complete-card unlocks | Other blockers, downstream gain, effort |
| --- | --- | ---: | --- |
| Explicit-player life | Stream of Life; Healing Salve; Alabaster Potion; Relic Bind; Spiritual Sanctuary; Swords to Plowshares; Crumble; Personal Incarnation; Nafs Asp; Psychic Purge; Shahrazad | 1 | Stream is ready. Salve/Potion need prevention; Relic needs opponent filter, tapped-host event, trigger modes and player/planeswalker union; Sanctuary needs intervening-if/player filter; Swords/Crumble need zone/value snapshots; others need event/zone/subgame/delayed services. Small adapter, moderate scope and gameplay tests. Shared life remains a useful multi-card prerequisite. |
| Next amount of recipient damage prevention | Samite Healer; Conservator; Amulet of Kroog; Argivian Blacksmith; Combat Medic; Oasis; Kei Takahashi; Healing Salve; Alabaster Potion; Balm of Restoration; Rakalite; Guardian Angel | 7 after prerequisites | First seven enclosing records are otherwise ready (17 records). Salve/Potion need player life; Balm needs activation modes; Rakalite needs delayed triggers; Guardian needs granted player payment actions. Typed recipients were missing in ron; engine shared shields lack simultaneous source allocation choice. Adapter would be small; complete engine allocation/replacement tests are substantial. |
| Collection untap restrictions | Meekstone; Arena of the Ancients; Magnetic Mountain; Thelon's Curse | 2 after prerequisites | Meekstone needs numeric power filters; Arena needs supertype filter and collection tap; Mountain/Curse need per-object payment/choice scopes. Enables four names, moderate adapter/filter work plus lifecycle tests and larger payment follow-up. |
| Typed single AnyTarget prerequisite, selected final scope | 32 declaration users below; 18 direct single-source damage users | 0 | Needed for recipient prevention and future mixed/multiple damage. Existing raw target plus implicit damage execution supplied no typed recipient read. Small compiler change with substantial declaration, identity, LKI, copy, mode and replacement tests. No engine allocation behavior is added. |

Selection was reassessed during engine-contract inspection. The initial life
candidate had one immediate unlock. The expanded nested inventory showed seven
prevention unlocks, so prevention was preferred. A trial adapter reached those
17 records, but the simultaneous audit found a missing CR 615.7 engine contract.
That trial and all its tests were removed; it is not part of this slice and none
of its temporary compilation gains are counted. The final scope is the complete
per-instance typed recipient prerequisite, which enables the larger shared
prevention capability without accepting incomplete prevention rules. Contract,
reuse boundary and acceptance cases for each scope change were stated before
implementing that scope. No second completed capability was retained.

### Engine gap and exact supported contract

Missing compiler prerequisite: a typed single mixed recipient declaration/read.
Missing engine prerequisite for future prevention: `DamageSystem` prepares each
simultaneous request in slice order, applying the shared shield immediately.
There is no allocation choice among simultaneous damage sources. CR 615.7 gives
that choice to the affected player or permanent's controller. A temporary,
standalone Go probe used two sources dealing 2 each to one player, one with
lifelink, and a shield of 2. The recipient implemented both ChooseMode and
ChooseDamageDistribution. `go run /tmp/ron-prevention-probe.go` reported
`choice calls=0; source controller life=20; recipient life=18`. Choosing one
prevented damage from each source would instead give the source controller one
life. `dealSimultaneousDamageOutcomes` and `preventionShieldReplacement` confirm
the packet-order consumption. This is engine behavior, not a source defect or
an environment failure. Future prevention remains rejected in production.

Exact supported declaration: symbol `AnyTarget` as the one target in
`Targeted([AnyTarget], ActionList(...))`. Exact read: symbol `Ref_AnyTarget` as
argument 3 of `SpellDealsDamage(ThisSpell, value, Ref_AnyTarget)` or
`PermanentDealsDamage(ThisPermanent, value, Ref_AnyTarget)`. Existing fixed/X
values and spell/activation/trigger action-list contexts remain supported.
Existing spell choose-one modes give each declaration a private namespace.
Multiple groups, numbered recipients, wrapping this mixed reference as a
permanent/player, and unsupported enclosing actions remain rejected.

`compileDamageTargetRef(Node, scope) (*damageTargetPlan, error)` checks the
lexical typed plan. `scope.damageTarget`, `actionPlan.damageTarget` and
`instanceBindings.recipients map[*damageTargetPlan]*mage.DamageTargetBinding`
carry immutable plan identity and fresh per-factory declarations. `compileTarget`
builds `mage.DeclareDamageTarget(namespace+"Ref_AnyTarget")`; action-plan building
includes the declaration with existing typed target groups. The damage body
uses `mage.DealDamageFrom(mage.This, boundRecipient,
mage.ValueAtResolution(amount))`. Mode scopes clear inherited mixed plans.

Reuse: definition validation, transactional casting/activation, target binding
and retargeting, current legality, permanent incarnation, source LKI,
DamageSystem, replacement/prevention ordering, modal and game cloning. No
engine service or public Go API changed. Rules read: CR 400.7, 601.2b-c, 608.2b,
702.26b-d, 615.3-7 and 616.1. No new simultaneous operation is lowered and no
simultaneous shield-allocation support is claimed. Comments added: none.

### Affected unique cards and gameplay mapping

The AST inventory finds 32 names with a target-role AnyTarget declaration:
Aeolipile; Alabaster Potion; Aladdin's Ring; Amulet of Kroog; Balm of Restoration;
Banshee; Brothers of Fire; Chain Lightning; Combat Medic; Cuombajj Witches;
Disintegrate; Drain Life; Goblin Grenade; Guardian Angel; Healing Salve;
Icatian Javelineers; Lightning Bolt; Orcish Artillery; Orcish Mechanics;
Pirate Ship; Prodigal Sorcerer; Psionic Blast; Psionic Entity; Psychic Purge;
Rakalite; Rocket Launcher; Rod of Ruin; Samite Healer; Sword of the Ages;
Thorn Thallid; Triskelion; Voodoo Doll. All complete records were read. Nested
unsupported parents are counted as syntax users, not accepted behavior.

Direct single-source damage users are Aeolipile, Aladdin's Ring, Chain
Lightning, Disintegrate, Drain Life, Goblin Grenade, Icatian Javelineers,
Lightning Bolt, Orcish Mechanics, Pirate Ship, Prodigal Sorcerer, Psychic Purge,
Rocket Launcher, Rod of Ruin, Sword of the Ages, Thorn Thallid, Triskelion and
Voodoo Doll. Seven already compile: Aeolipile, Aladdin's Ring, Lightning Bolt,
Orcish Mechanics, Prodigal Sorcerer, Rod of Ruin and Thorn Thallid. All others
retain their prior blockers and first diagnostics.

| Card/contract | Evidence |
| --- | --- |
| Lightning Bolt | TestRONDamageScopeRealCards: all six reprints, complete records and definition validation, player/creature/planeswalker targets, payment, damage and clone isolation |
| Prodigal Sorcerer; Rod of Ruin; Aladdin's Ring; Aeolipile | TestRONDamageScopeRealCards: complete LEA/ARN/FEM records, mana/tap/sacrifice costs, three recipient categories, source LKI after self-sacrifice, cloned resolution |
| Orcish Mechanics | Retained TestRONSacrificeCost* and TestRONSacrificeCostSourceAndTargets / TestRONSacrificeCostInstancesAndCopy: real RON sacrifice selection, timing, rollback, source LKI and recipient damage; all pass with typed lowering |
| Thorn Thallid | Retained TestRONCounterCost* real RON upkeep/counter removal, invalid payment, instance/copy and target damage scenarios; all pass with typed lowering |
| Typed read and fresh factory | TestRONDamageScopeDeclarations: distinct target declarations, exactly one typed read, definition validation and rejection after removing its declaration |
| Identity/legality | TestRONDamageScopeIdentity: current control, leave/return, phased target, phase cycle, shroud at resolution |
| Mode/copy/instance scopes | TestRONDamageScopeModesCopies: two modes with same RON spelling, fresh instances, retargeted copies retain mode, one announcement choice |
| X/payment | TestRONDamageScopeAnnouncedX: unpaid X rollback, announced X payment and copied X with new recipient |
| Replacement ordering | TestRONDamageScopeReplacementOrder: selected player chooses half-damage versus fixed shield ordering; two choices yield different actual damage |
| Trigger timing/LKI | TestRONDamageScopeTrigger: entry trigger target at placement, damage only at resolution, phased source retains its LKI |
| Self-sacrifice/LKI/rollback | TestRONDamageScopeRollbackAndSourceLKI: unpaid/illegal activation preserves state; source sacrificed at payment and returned under another controller; live and clone retain original source and target |
| Invalid declarations | TestRONDamageScopeInvalid: undeclared reference, incompatible player target, tuple read, cross-mode read, AnyTarget tuple, future prevention remains rejected |
| Accepted registry/inventory | Retained full RON/CLI tests, strict rejection, atomic registry and inventory tests |

TDD: draft life/prevention tests exposed missing adapters; these drafts were
removed with their trial implementations. The final typed-declaration test was
run against restored baseline production files and failed as expected with
`typed recipient reads=0 want=1`. Restoring the final typed implementation made
all new gameplay and declaration tests pass. Test fixtures initially confused
turn count with clearing summoning sickness and PhaseIn's slice argument; those
were corrected without weakening engine rules. The final shape's tests cover
five previously accepted names directly and retain broader RON coverage for two
more. No newly accepted card or full-collection gameplay total is claimed.

### Final measurements and validation

Final prerequisite estimate versus actual: **zero new unique cards and zero
records**, as expected. The seven/17 prevention estimate was conditional on
prerequisites; it remains a downstream estimate, not delivered compilation.
Compiled stays **1,106/2,445**; rejected stays **1,339**. Accepted/rejected unique
names stay **373/613**. Strict issue arrays are byte-for-byte equal as decoded
JSON before/after, and strict/semantic final issues match. Zero regressions and
zero new gaps. Inventory stays 72,938 occurrences, 8,528 signatures and 14,648
unknown-role occurrences. Source hashes match all twelve baseline files and the
vocabulary snapshot. RON source did not change, so no snapshot refresh was needed.
Engine-validated and behavior-verified collection totals remain unmeasured/null.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS |
| Baseline go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline strict and semantic loader | Expected exit 1: 1,106 compiled / 1,339 rejected |
| go test ./pkg/mage/dsl/ron -run TestRONDamageScopeDeclarations -count=1 against baseline | Expected FAIL: missing typed recipient read |
| go test ./pkg/mage/dsl/ron -run TestRONDamageScope -count=1 | PASS |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS, zero issues; formatting changes reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory -json | Expected exit 1: 1,106 compiled / 1,339 rejected; identical diagnostics |
| Source hashes and accepted-record comparison | PASS, unchanged hashes and zero regressions |

Leading first diagnostics remain rule If (53 records), non-spell CastEffect
(53), ChooseADamageSource (46), Targeted_Modal (42), self untap HostPermanent
(39), upkeep HostController (35), other self untap operations (34),
ControlledByAPlayer (29), TargetGraveyardCard (26),
WhenAPermanentBecomesTapped (25) and CreateFuturePreventDamage (24).
These counts are first errors and are not unlock estimates.

Changed files: compiler.go, player_scope.go, modal_spell.go, new damage_scope.go
and damage_scope_test.go; dsl-reference.md, engine-rules-contracts.md and this
ledger. No source-data correction occurred. Earlier Gaea's Liege/Fastbond
questions remain open. No Go definitions, generators, startup code or DSL
exports were removed; removal gates are not met. No deletion caller audit is
claimed. No fallback, card-name branch, shared mutable binding, global action
slot, runtime declaration panic or partial production registry was added.

Ranked next candidates:
1. Shared simultaneous shield allocation in the engine (CR 615.7), then reassess
   exact recipient prevention lowering for the seven ready cards. Test split
   prevention across sources, lifelink/deathtouch, competing replacements,
   redirection, APNAP, source/recipient identity, phasing and clone isolation.
2. Explicit-player life: Stream of Life plus the Salve/Potion prerequisite;
   preserve current event player versus current spell controller contracts.
3. Collection untap/numeric/supertype prerequisites for Meekstone and Arena of
   the Ancients; Karakas also benefits from supertype support.

Complete RON gameplay, runtime loading, Go-card/generator removal, DSL cleanup
and native startup validation remain unfinished. This run stops here.

## Run 2026-10-04: P7-TARGETED-TEMPORARY-UNBLOCKABILITY

Status: complete one supported targeted rule shape, with its fixed-power filter
prerequisite. This run stops here. Initial user changes to `.gitignore` and
`docs/designs/mtgish-ron-execution.md` are preserved and excluded from staging.
No unrelated untracked files existed. No delegation or removed skills were used.

### Fresh evidence and candidate selection

Read AGENTS.md, pkg/mage/doc.go, the execution design, vocabulary snapshot and
ledger. Read selected compiler, typed permanent references, inventory, engine
filters/continuous modifications/combat tests and relevant API/rules contracts.
Used the fresh positioned AST inventory for nested forms and distinct names,
then read complete RON records, including all six Dwarven Warriors reprints.
No Go card definitions were used as requirements.
Baseline `go test ./pkg/mage/dsl/ron -count=1` and `make build` passed. Strict and
semantic loaders exited 1 as expected: 1,106 compiled, 1,339 rejected records;
373 accepted and 613 rejected unique names. This was collection rejection, not
an environment/build failure.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers, downstream gain, effort |
| --- | --- | ---: | --- |
| Fixed-power targeting plus temporary unblockability, selected | Dwarven Warriors; Tawnos's Wand; Meekstone; Amrou Kithkin; Brassclaw Orcs; Ironclaw Orcs; Orcish Veteran; Orgg; Lesser Werewolf; Old Man of the Sea; Drop of Honey. Same CantBeBlocked operation also occurs in Teleport and Sewers of Estark. | 2 / 7 records | Warriors/Wand are otherwise ready. Meekstone needs collection untap rules; Kithkin/Orcs/Veteran/Orgg need static combat rules; Werewolf needs combat filters, conditions and counters; Old Man needs dynamic power/control durations; Drop needs minimum values, choice destruction and state triggers. Teleport needs casting restrictions; Sewers needs conditions and prevention. Two small compiler adapters with moderate target/lifecycle/gameplay tests. |
| Supertype permanent filter | Karakas; Livonya Silone; Arena of the Ancients; Adventurers' Guildhouse; Cathedral of Serra; Mountain Stronghold; Seafarer's Quay; Unholy Citadel; Willow Satyr; Gaea's Touch; Goblin Caves; Goblin Shrine; Land Tax; Untamed Wilds | 1 | Karakas is otherwise ready. Livonya needs separate landwalk lowering; Arena needs collection tap/untap; five lands need controlled-player filters and bands-with-other grants; remaining cards need control durations, choices, conditions, search or intervening-if. Small filter adapter with moderate legality tests; useful downstream multi-card prerequisite. |
| Explicit-player life | Stream of Life; Healing Salve; Alabaster Potion; Relic Bind; Spiritual Sanctuary; Swords to Plowshares; Crumble; Personal Incarnation; Nafs Asp; Psychic Purge; Shahrazad | 1 | Stream is ready. Salve/Potion need prevention; Relic needs tapped-host event, modes, opponent filter and union targets; Sanctuary needs intervening-if; Swords/Crumble need snapshots/zone actions; others need event, delayed, source-zone or subgame services. Small adapter with moderate player/X/copy tests. |
| Shared simultaneous prevention allocation prerequisite | Samite Healer; Conservator; Amulet of Kroog; Argivian Blacksmith; Combat Medic; Oasis; Kei Takahashi; Healing Salve; Alabaster Potion; Balm of Restoration; Rakalite; Guardian Angel | 0 immediately; 7 after adapter | Prior ledger gives seven otherwise-ready records' unique names (17 records). Salve/Potion also need life; Balm activation modes; Rakalite delayed triggers; Guardian player payment actions. CR 615.7 source allocation needs a reusable engine contract and substantial damage/replacement/choice tests. Higher total benefit, materially larger effort than the selected pair. |

Selection: the selected slice gives the largest credible immediate unique-card
gain among the small ready adapters, and also establishes fixed numeric filters
for several later combat/untap cards. It reuses an already tested engine blocking
restriction. The larger prevention prerequisite remains valuable; it was not
silently bypassed. First-error counts were not treated as unlock estimates.
Contract, reuse boundary and acceptance cases were stated before production edits.

### Exact shapes, contexts and reuse boundary

- `compileFixedPowerFilter(Node) (mage.PermanentFilter, error)` accepts
  `PowerIs(LessThanOrEqualTo(Integer(n)))`,
  `PowerIs(GreaterThanOrEqualTo(Integer(n)))` and `PowerIs(EqualTo(Integer(n)))`.
  Signed integer constants are allowed. Lowering uses `HasPowerLTE`,
  `HasPowerGTE`, `HasPowerEqual`; each query reads current layered power.
  These are permanent filters, including nested And/Or in supported contexts.
  Dynamic values, other comparisons, malformed tuples and bare integers reject.
- `compileTemporaryBlocking(Node, scope) (effectFactory, error)` accepts only
  `CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked],
  UntilEndOfTurn)` with a declared permanent target in resolving action lists.
  Supported spell, choose-one mode, activation and trigger contexts share the
  adapter. Source/host/result references, other operations, mixed operation
  lists and other durations remain rejected. It uses `compilePermanentRef`,
  per-instance bindings and `Until(EndOfTurn, CantBeBlocked(typedTarget))`.
- Inventory now assigns the rule subject/operation/duration roles and nested
  power comparison/value roles. Role reporting does not claim gameplay support.

RON owns syntax, declaration validation and scope; mage retains all reusable
rules. Existing definition validation, target selection/rechecks, transactional
costs, source LKI, incarnation capture, phasing, combat rules, cleanup, ability
copying and clone services are reused. No engine behavior or public API changed.
Rules read: CR 400.7, 601.2c, 608.2b/h, 611.2c, 514.2 and 702.26b-f.
No new replacement, choice or simultaneous-operation behavior is introduced.
No comments were added to code. API and rules-contract docs describe only the
supported shapes and tested reuse.

### Card-to-test mapping and TDD

| Card/contract | Evidence |
| --- | --- |
| Dwarven Warriors, LEA/LEB/2ED/3ED/CED/CEI | TestRONTemporaryBlockingRealCards: complete records, definition validation, tap payment, either player's power-two creature, stack timing, clone isolation and cleanup |
| Tawnos's Wand, ATQ | Same real-card test plus mana payment; all gameplay contract tests below use its complete RON record |
| Target rechecks and captured lifetime | TestRONTemporaryBlockingLegality: power before/after resolution, target departure/return before/after resolution, phased target, phase cycle, departed source, control change and ability removal |
| Payment/legality rollback | TestRONTemporaryBlockingRollback: insufficient mana, power-three target, noncreature, phased target, shroud and Dwarven Warriors summoning sickness; tap, pool and stack remain unchanged |
| Typed identity/copies | TestRONTemporaryBlockingCopiesAndBindings: fresh factory declarations, definition rejection without target declaration, independently retargeted ability copy and original |
| Phasing/source/expiry | TestRONTemporaryBlockingPhaseAndSourceLifetime: source departure after resolution, target phase cycle, later blockers and cleanup while phased |
| Enclosing contexts | TestRONTemporaryBlockingSpellAndTriggerContexts: synthetic spell, choose-one mode and entry trigger, target placement and resolution timing |
| Numeric predicate | TestRONFixedPowerFilters: all three comparisons, signed bounds and changing current power from layers |
| Invalid declarations/roles | TestRONTemporaryBlockingInvalid and TestRONTemporaryBlockingInventory; retained modal scope, registry rejection and inventory tests |

The initial test fixture used a nonexistent summoning-sickness member. It was
corrected to the engine's existing RevokeBaseAttr service before confirming the
expected semantic failure. Against baseline, real-card and fixed-filter tests
failed with `unsupported permanent filter (PowerIs)`. After implementation all
acceptance tests passed. Broad checks initially exposed two old hardcoded loader
counts, updated to the measured 1,113/1,332 without relaxing assertions. The new
inventory test initially counted only Warriors' six reprints in a shared signature;
it was corrected to seven, including Wand. No gameplay assertions were weakened.

### Final measurements and validation

Actual gain equals estimate: **2 unique cards and 7 records**. Compiled is
**1,113/2,445**, rejected is **1,332**; accepted/rejected unique names are
**375/611**. Zero accepted-record regressions. Only the seven Warriors/Wand
records leave the issue list. Six remaining records receive different diagnostics:
Hurr Jackal (CantBeRegenerated), Old Man of the Sea (dynamic PowerOfPermanent),
Phyrexian Gremlins (ThePermanentTappedThisWay), Tower of Coireall
(CantBeBlockedByDefenders), Whippoorwill (mixed unsupported rule list), and Wall
of Wonder (source rule target). All remain rejected; these are not unlocks.

Inventory retains 72,938 occurrences and 8,528 signatures. Unknown-role
occurrences decrease from 14,648 to 14,466. Strict and semantic issues agree.
All twelve source file hashes are unchanged; no vocabulary refresh was needed.
The newly accepted pair has complete real-RON behavior mappings above. Collection
engine-validated and behavior-verified totals remain unmeasured/null; compilation
progress is not full-collection gameplay coverage.

| Command | Final result |
| --- | --- |
| Baseline go test ./pkg/mage/dsl/ron -count=1; make build | PASS |
| Baseline strict and semantic loader | Expected exit 1, 1,106 compiled / 1,339 rejected |
| New real-card/fixed-filter tests against baseline | Expected semantic FAIL, missing PowerIs adapter |
| go test ./pkg/mage/dsl/ron ./cmd/loadcards -count=1 | PASS |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS; automatic changes reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory -json | Expected exit 1, 1,113 compiled / 1,332 rejected |
| Source hashes and accepted-record comparison | PASS; unchanged sources, zero regressions |

No source defect was found in this slice. Prior Gaea's Liege/Fastbond questions
remain open. No environment blocker occurred. The first-error leaders remain
rule If (53), non-spell CastEffect (53), ChooseADamageSource (46), Targeted_Modal
(42), host self-untap (39), upkeep HostController (35), other self-untap operations
(34), ControlledByAPlayer (29), TargetGraveyardCard (26), tapped events (25), and
CreateFuturePreventDamage (24). These are record counts, not card unlock estimates.

Changed files: compiler.go, temporary_blocking.go, temporary_blocking_test.go,
inventory.go, inventory_test.go, dsl-reference.md, engine-rules-contracts.md and
this ledger. No Go definitions, generators, startup paths or DSL exports were
removed; removal gates are not met. No deletion caller audit is claimed. No
fallback, card-name branch, global action slot, shared mutable target, declaration
panic or partial production registry was added.

Ranked next candidates:
1. Collection untap restriction: numeric filters now remove Meekstone's filter
   prerequisite. Combine with supertype/collection tap prerequisites to reassess
   Arena of the Ancients; investigate Mountain/Curse payment scopes separately.
2. Shared CR 615.7 simultaneous shield allocation, then exact recipient prevention
   adapter for the seven otherwise-ready cards. Do not bypass the engine gap.
3. Supertype filters for Karakas and downstream Arena/banding/search services;
   explicit-player life for Stream and the Salve/Potion prerequisite is also ready.

Complete RON gameplay, runtime loading, Go-card/generator removal, DSL cleanup
and native startup validation remain unfinished. No second slice began.

## Run 2026-10-04: P7-COLLECTION-UNTAP-DECISION

Status: one complete collection untap restriction shape, with the selectors and
collection tap action needed for Arena of the Ancients. Initial user changes to
`.gitignore` and `docs/designs/mtgish-ron-execution.md` are preserved and excluded
from staging. No unrelated untracked files existed at baseline. No delegation,
removed skills, generated Go cards or Go-card requirements were used.

### Fresh inventory and candidate comparison

Read AGENTS.md, pkg/mage/doc.go, the execution design, vocabulary snapshot and
progress ledger. Read selected compiler, inventory, trigger, filter, effect,
turn, layer, clone and simultaneous tap services and relevant API/rules contract
documentation. Read complete RON records and nested forms for the candidates,
including all Meekstone reprints and the downstream collection tap records.
Local Oracle data in data/LEA.json, LEG.json and DRK.json agrees with the four
completed cards. Rules read: CR 502.1-4 and 613.1-6; existing source identity,
phasing and target contracts are reused. Applied the implement-engine skill
only to the reusable engine prerequisite; RON adapters remain in ron.

Fresh baseline engine short tests and make build passed. Strict loading and
semantic inventory exited 1 as expected: 1,113 compiled and 1,332 rejected
records, with 375 accepted and 611 rejected unique names. This is collection
rejection, not a build or environment failure.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers, downstream capabilities, effort |
| --- | --- | ---: | --- |
| Collection untap decision plus Arena selectors, selected | Meekstone; Arena of the Ancients; Magnetic Mountain; Thelon's Curse. Shared legendary selector also reaches Karakas, Livonya Silone, Adventurers' Guildhouse, Cathedral of Serra, Mountain Stronghold, Seafarer's Quay, Unholy Citadel, Willow Satyr. Collection tap reaches Riptide, Mana Short, Power Sink, Feint, Spore Cloud, Word of Binding, Winter Blast, Rohgahh of Kher Keep, Arena. | 3 / 8 records: Meekstone, Arena of the Ancients, Karakas | Mountain/Curse need upkeep event players, payment choices and chosen collections. The five lands need BandsWithOthers; Livonya needs legendary landwalk; Willow needs control duration. Tap consumers need controlled-player collections, plural targets, combat relations, prevention or action results. Small adapters plus a moderate engine timing contract and lifecycle/gameplay tests. Riptide was not included in the initial estimate; it was found ready during fresh post-change loading. |
| Standalone permanent supertype adapter | Karakas; Livonya Silone; Arena of the Ancients; Adventurers' Guildhouse; Cathedral of Serra; Mountain Stronghold; Seafarer's Quay; Unholy Citadel; Willow Satyr; Gaea's Touch; Goblin Caves; Goblin Shrine; Land Tax; Untamed Wilds | 1 | Karakas ready; Arena also needs collection untap/tap; Livonya needs landwalk lowering; lands need banding grants; others need control, entry choice, conditions, intervening-if or search. Small adapter, moderate legality tests. Fold the Legendary permanent shape into the selected dependent slice; Basic and other roles remain rejected. |
| Explicit-player life | Stream of Life; Healing Salve; Alabaster Potion; Relic Bind; Spiritual Sanctuary; Swords to Plowshares; Crumble; Personal Incarnation; Nafs Asp; Psychic Purge; Shahrazad | 1 | Stream ready. Salve/Potion need shared prevention allocation; Relic needs tapped events, opponent filters, trigger modes and target unions; Sanctuary needs intervening-if; Swords/Crumble need snapshots and zone actions; others need events, replacements, delayed/source-zone or subgame services. Small adapter, moderate player/X/copy tests. |
| Shared simultaneous prevention allocation | Samite Healer; Conservator; Amulet of Kroog; Argivian Blacksmith; Combat Medic; Oasis; Kei Takahashi; Healing Salve; Alabaster Potion; Balm of Restoration; Rakalite; Guardian Angel | 0 immediately; 7 after lowering | The prior ledger identifies seven otherwise-ready unique cards / 17 records. Others need life, activation modes, delayed triggers or player payment actions. Shared CR 615.7 engine allocation enables these cards, but requires substantially larger damage/choice/replacement contract work. |

Selection rationale: complete two related collection rules and their required
selectors for more immediate unique-card gain than standalone filters or life.
The engine prerequisite also enables the same exact untap rule on Mountain and
Curse, without pretending their payment paths work. The larger prevention
prerequisite remains valuable. First-error frequency was not an unlock estimate.
The contract, engine/RON boundary and acceptance cases were stated before edits.

### Exact signatures, supported contexts and dependencies

- `PreventUntapDuringStepForMatching(PermanentFilter) ContinuousEffect` is a
  reusable engine static rule. It stores an immutable filter declaration, not
  selected game objects. `matchingUntapRestrictions() map[uuid.UUID]bool` creates
  a local snapshot after phasing and layer evaluation, before the untap loop.
  Active printed-source checks use the existing effect wrapper; final layered
  characteristics use the existing filters. Other untap effects remain legal.
  Recipient ability removal does not erase this restriction. Source departure,
  phasing and ability removal disable it. Source return builds a new instance.
  The existing `PreventUntapForMatching` API and callers remain unchanged.
- `compileCollectionPermanentRule(Node) (optionFactory, error)` accepts
  `EachPermanentRuleEffect(filter, [DoesntUntapDuringControllersUntap])` in
  permanent Rules, with one bare operation and a supported source-independent
  permanent filter. The established landwalk blocking shape delegates to its
  existing adapter. Mixed/empty operation lists, operation tuples, bound/result
  selectors and source-dependent controller collections remain rejected.
- `compileFilter` accepts `IsSupertype(Legendary)` for permanent filters,
  including nested And/Or, using `mage.IsLegendary`. Basic/Snow/World and
  card/spell filter roles remain unsupported. No new supertype engine semantics
  are invented.
- `compileEffect` accepts `TapEachPermanent(filter)` in resolving spell, mode,
  activation and trigger action lists, using
  `mage.TapEach(mage.MatchingPermanents(filter))`. Filters are the supported
  source-independent permanent forms. Target collections, choice results,
  loop bindings and player-dependent selections are not accepted. Selection
  occurs at resolution and uses existing incarnation, phasing and simultaneous
  tap/event services.
- Source-only `WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent))`
  now lowers to `mage.WhenEvent(mage.Enters(mage.This), body)`. The typed event
  path keeps event context separate from target declarations. It preserves
  source departure, phasing, return and copies without legacy auto-target fizzle.
- The layer reset now includes `ptSwitched` in its reset condition. The new
  final-power untap test exposed inconsistent repeated switch evaluation.
- Inventory assigns the TapEachPermanent child its permanent_filter role.
  Reporting still marks signature support unverified; no support claim is
  inferred from a signature name.

RON owns syntax and validation; mage owns rule execution. Factory calls build
fresh abilities/effects. Costs, targets, source LKI, incarnation checks, cloning,
continuous layers and tap batches reuse shared engine services. The local untap
snapshot cannot leak into another game or overwrite another binding. No new
payment, replacement, optional-choice or global last-action contract is added.
No Go fallback, card-name dispatch, partial registry or runtime declaration
panic was added.

### TDD and card-to-test mappings

| Card / contract | Tests |
| --- | --- |
| Meekstone: all LEA/LEB/2ED/3ED/CED/CEI records | TestRONCollectionUntapMeekstone: definition validation, either player's creatures, final boosted power, other untap effect, source departure |
| Arena of the Ancients: LEG | TestRONCollectionUntapArena: full record, definition validation, entry trigger timing, current resolution membership, legendary creature selection, nonlegendary/land exclusion, phased recipients, source departure/return/phasing/control, copies and clone isolation |
| Karakas: LEG | TestRONCollectionUntapKarakas: full record, both abilities, immediate white mana and metadata, tap cost timing, either player's legend, owner hand destination, nonlegendary/noncreature/phased target rollback, leave/return legality, copied ability and clones |
| Riptide: DRK | TestRONCollectionTapRiptide: full record, cast/resolution timing, both players' blue creatures, land/nonblue/phased exclusion, simultaneous tap visibility before events, clone isolation |
| Shared final-characteristic rule | TestMatchingUntapDecision: boost/reduction/switch, source abilities, recipient abilities, source phase/departure/return, recipient phase-in, current control, inactive player exclusion, clone source removal, other untap effects |
| Decision timing | TestMatchingUntapDecisionSnapshot: source untaps first but cannot change the current restricted set; the next step refreshes membership |
| Declarations and syntax | TestRONCollectionUntapInvalid; TestRONCollectionUntapDeclarationsAndInventory: invalid operation shapes, mixed/empty lists, unbound and wrong-role selectors, unsupported supertype forms, fresh factories, engine validation, nested semantic roles |
| Retained interactions | Existing RON source-entry, landwalk, static untap, typed target and registry tests; TestTapEachTransitionsTogether and engine layer/clone tests |

Confirmed baseline semantic failures with the existing matching untap service:
boosted/reduced power, source ability removal and recipient ability removal.
The first synthetic fixture omitted an owner for its destroyed artifact; the
fixture was corrected before recording the full semantic failure evidence.
RON tests failed at unsupported untap operations and IsSupertype. Riptide was
also confirmed to fail with the TapEachPermanent adapter absent. No gameplay
assertions were weakened. The late-entry Arena fixture used a different legendary
creature after the original duplicate-name fixture correctly invoked the legend
rule. Numeric collection inventory assertions were updated to the fresh counts.

### Measurements and validation

Actual gain: **4 unique cards / 9 records**, versus the estimate of **3 / 8**.
Riptide accounts for the additional gain through the required collection tap
adapter; its complete RON gameplay mapping was added before counting it.
Final compiled/rejected records: **1,122 / 1,323** of 2,445. Accepted/rejected
unique names: **379 / 607** of 986. Zero accepted-record regressions. Only the
nine records for the four completed cards leave the issue list.

Seventeen records receive later diagnostics: Mana Short (six controlled-player
filter errors); Magnetic Mountain (two) and Thelon's Curse (one) now reach
unsupported upkeep AnyPlayer; Spore Cloud reaches prevention; Feint reaches
IsBlockingAttacker; five legendary lands reach BandsWithOthers; Willow Satyr
reaches control duration. These are still rejected, not full-card unlocks.
Strict and semantic issues agree. Inventory retains **72,938 occurrences /
8,528 signatures**; unknown-role occurrences fall from **14,466 to 14,405**.
All twelve source hashes are unchanged, so no vocabulary source refresh is needed.
Engine-validated and behavior-verified collection totals remain null/unmeasured;
this run supplies full real-RON mappings for four additional unique cards.
Compilation and collection gameplay completion remain separate.

| Command | Result |
| --- | --- |
| Baseline go test ./pkg/mage/... -short -count=1; make build | PASS |
| Baseline strict and semantic loaders | Expected exit 1, 1,113 compiled / 1,332 rejected |
| New engine/real-RON tests against missing contracts | Expected semantic FAIL, described above |
| Focused go test ./pkg/mage ./pkg/mage/dsl/ron ./cmd/loadcards -run 'TestMatchingUntapDecision\|TestRONCollection\|TestInventory' -count=1 | PASS |
| go test ./pkg/mage/... -short -count=1 | PASS |
| make test | PASS |
| make build | PASS |
| go vet ./... | PASS |
| make lint | PASS, zero issues; automatic import formatting reviewed |
| git diff --check | PASS |
| Final go run ./cmd/loadcards -json; -inventory -json | Expected exit 1, 1,122 compiled / 1,323 rejected; strict and semantic diagnostics agree |
| Accepted-record and source-hash comparison | PASS, zero regressions and unchanged sources |
| Post-lint focused tests with -count=1 | PASS |
| Post-lint go test ./... -short -count=1 | PASS |
| Post-lint strict loader | Expected exit 1; identical 1,122 / 1,323 report |

No source defect or environment blocker occurred in this slice. Prior
Gaea's Liege/Fastbond questions remain open. Leading first diagnostics remain
rule If (53 records), non-spell CastEffect (53), ChooseADamageSource (46),
Targeted_Modal (42), host self-untap (39), upkeep HostController (35),
ControlledByAPlayer (35), other self-untap operations (34), graveyard targets
(26), tapped events (25), and CreateFuturePreventDamage (24). These are record
first-error counts, not unlock estimates.

Changed files: new engine untap_matching.go/test; turn_execution.go,
effect_manager.go and doc.go; new ron collection_untap.go/test; compiler.go,
triggers.go, inventory.go/test; dsl-reference.md, engine-rules-contracts.md and
this ledger. No Go definitions, generators, startup wiring or DSL exports were
removed; the removal gates are not met. No deletion caller audit is claimed.

Ranked next candidates:
1. Shared CR 615.7 simultaneous shield allocation, then exact prevention lowering
   for the seven otherwise-ready unique cards. Preserve replacement ordering,
   source allocation, actual damage, lifelink/deathtouch and clone contracts.
2. Banding-with-other legendary creatures: five lands now pass their complete
   static selection but stop at BandsWithOthers. Audit existing combat services
   before claiming an adapter-only task; the whole five-card benefit is concrete.
3. Explicit-player life: Stream of Life is ready; Salve/Potion still require
   prevention. Controlled-player collection selectors may instead win after a
   fresh comparison for Mana Short and downstream payment/combat collections.

Complete collection gameplay, runtime loading, Go-card and generator deletion,
DSL cleanup and native startup validation remain unfinished. This run stops
after this slice; no second slice began.

## Run 2026-10-04: RANDOM-DISCARD-BATCH

Status: complete random-discard action and simultaneous engine batch slice.
This run stops here. Runtime cutover and removal gates remain open.
The initial tree had user changes in `.gitignore` and
`docs/designs/mtgish-ron-execution.md`; both are preserved and excluded from the
slice commit. No unrelated untracked files were present.

### Fresh evidence and candidate comparison

Baseline `go test ./pkg/mage/... -short -count=1` and `make build` passed.
Strict and semantic loading each exited 1 for collection rejection:
**1,122 compiled / 1,323 rejected records**, 379 accepted / 607 rejected unique
names. All 2,445 records and 986 names were inspected through the AST inventory.
The report still has 72,938 occurrences and 8,528 context/shape signatures.
First diagnostics were used to find candidates, not to estimate full unlocks.
Complete RON records and nested forms were read for the names below.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers, downstream capabilities and effort |
| --- | --- | ---: | --- |
| Random-discard action plus simultaneous batch, selected | Hymn to Tourach; Mind Twist; Ring of Renewal; Gwendlyn Di Corci; Hypnotic Specter. Related cost/filtered consumers: Coral Helm; Draconian Cylix; Rag Man. | 3 / 8 records: Hymn, Twist, Ring | Gwendlyn was conservatively excluded for its activation modifier; existing turn-only lowering actually makes it ready. Specter needs damage-event player bindings; Helm/Cylix need random cost transactions; Rag Man needs opponent targeting and filtered reveal/discard. Shared action is a prerequisite for damage-event discard. Small compiler adapter and moderate engine batch/interaction tests; existing typed action and discard batch executor substantially reduce effort. |
| Amount prevention shields | Samite Healer; Conservator; Amulet of Kroog; Argivian Blacksmith; Combat Medic; Oasis; Kei Takahashi; Healing Salve; Alabaster Potion; Balm of Restoration; Rakalite; Guardian Angel | 0 for allocation prerequisite alone; 7 after lowering | Salve/Potion need explicit-player life; Balm needs activation modes; Rakalite needs a delayed return; Guardian needs player payment rules. Shared simultaneous CR 615.7 allocation enables the seven otherwise-ready cards, but needs a larger damage, replacement and player-choice contract with source allocation and actual damage tests. |
| Bands with other legendary creatures | Adventurers' Guildhouse; Cathedral of Serra; Mountain Stronghold; Seafarer's Quay; Unholy Citadel. Related forms: Master of the Hunt; Shelkin Brownie; Tolaria. | 5 lands after engine and adapter support | Existing combat handles ordinary Banding only. Filtered band formation and incoming damage assignment need an exact reusable engine contract; granting ordinary Banding would be wrong. Hunt needs named token support and a different band group; Brownie/Tolaria need ability removal and timing. Moderate-to-large combat and layer implementation/test effort. |

Selection: random discard gives the largest credible immediate gain for the
smaller effort. The engine already has typed player/amount actions, deterministic
random queues, clone isolation and a shared discard batch service. The missing
contract is selection before replacement/movement, followed by complete-batch
visibility. Prevention and filtered banding remain valuable multi-card work.
The behavior contract, reuse boundary, acceptance cases and three-candidate
comparison were stated before editing. The implement-engine skill was applied
only to the reusable engine prerequisite; no card definitions were changed.

Local Oracle records in data/*.json were checked for all eight random-discard
consumers. They agree with the source forms. Rules read: CR 101.3, 608.2b/c/f,
616.1/f and 701.9a/b. No source defect was found or corrected. The previous
Gaea's Liege/Fastbond concerns remain open.

### Exact signatures, contexts, dependencies and reuse

- Existing `mage.DiscardAtRandom(PlayerRef, ResolutionValue) ResolvingEffect`
  resolves the player and amount once, then selects distinct cards from a local
  copy of the current hand using `Game.RandIntn`. Selection consumes one value
  per card; empty hands and nonpositive amounts consume none. Short hands limit
  the number selected. A prevented discard does not cause another card to be
  selected or the same card to be selected again.
- The action now calls existing `discardActionsTogether(actions, false)`.
  Effect-caused `DiscardAction` replacements are prepared before any moves;
  moves finish before discard events. Prevented cards do not stop other cards
  or later instructions. Shared replacement ordering/rechecks, destination
  handling, graveyard visits, trackers and event dispatch remain in the engine.
  The public signature, metadata, definition validation and existing callers
  are preserved. Legacy random helpers and random discard costs are unchanged.
- `compileRandomDiscard(Node, scope) ([]effectFactory, error)` accepts bare
  `DiscardACardAtRandom`, `DiscardNumberCardsAtRandom(Integer(n))/1` for n >= 0,
  and `DiscardNumberCardsAtRandom(ValueX)/1` in an announced-X scope. Other
  amounts, singular tuples, malformed declarations and unbound X are rejected.
- Supported contexts are resolving action lists in spells, selected modes,
  activations and already-supported triggers. `PlayerAction(player, action)/2`
  uses `compilePlayerRef`; direct actions keep the resolving controller even
  when the action list declares a player target. Typed target, spell-controller
  and available step/cast event reads use the existing instance environment.
  No new binding kind or mutable compiler state is added.
- The factory lowers to `mage.DiscardAtRandom(ctx.player.build(bindings), amount)`.
  Each factory/action build keeps its own target/event bindings. X uses the
  existing announced resolution value. RON owns spelling, shape rejection and
  lexical scope; mage owns random selection, replacements and movement.
- Inventory gives the plural action's child the value role in action and player
  action contexts, including nested forms under unsupported parents. Reporting
  does not infer verified support from constructor names.

No LKI characteristic read or permanent target is introduced. Pending source
abilities retain their resolving controller/target after departure, return,
control change or phasing through the existing stack/source contracts. Phasing
is not interpreted as a zone change. Random choices remain resolution work;
cost affordability and failed payments do not consume random values. No global
result slot, runtime declaration panic, name branch, fallback or partial
production registry was introduced.

### TDD, real-card gameplay and acceptance evidence

Before implementation, `TestRandomDiscardBatchEventsAndReplacements` failed
because events saw partial hands and hand-dependent replacements saw a prior
card already moved. With the prevention fixture corrected to match effect
rather than cost discard, it also proved a prevented first card was selected
again instead of selecting a distinct second card. The corrected fixture was
rerun and failed before implementation. RON tests failed on unsupported random
player/direct actions; the inventory test reported an unknown amount role.

Initial gameplay fixtures needed a main step for sorceries and a player that
accepts the optional library replacement. The two-pending-spell test uses a
synthetic instant beside real Mind Twist because a second sorcery cannot be cast
with a nonempty stack. These fixture repairs preserve legality and all behavior
assertions. Gwendlyn's full-record coverage was added after fresh loading found
its existing modifier was supported; it needed no further implementation.

| Unique card / contract | Test mapping |
| --- | --- |
| Hymn to Tourach (FEM), Mind Twist (LEA/LEB/2ED/3ED/CED/CEI) | `TestRONRandomDiscardSpells`: full records, definition validation, fixed and announced-X amounts including zero, empty/short hands, distinct scripted selection, no hand-choice prompts, announcement versus resolution, graveyard and clone/RNG isolation |
| Mind Twist | `TestRONRandomDiscardCopyAndInstances`: separate real factory declarations, pending instant/real spell with different X and players, retargeted copy retains X, independent original/clone hands and targets |
| Ring of Renewal (FEM) | `TestRONRandomDiscardRingLifecycle`: full record, tap/mana timing, random discard before draw, empty-hand continuation without RNG, captured controller after source departure/return/control/phasing, clones |
| Gwendlyn Di Corci (LEG) | `TestRONRandomDiscardGwendlyn`: full record and legend/P/T metadata, self/opponent target, your-turn restriction in combat, summoning sickness/tap/phasing rejection, new controller, pending source departure/return/control, changed turn at resolution, clones |
| Mind Twist, Ring of Renewal | `TestRONRandomDiscardRollback`: insufficient mana, invalid target, tapped/phased activation; hand/mana/stack/RNG unchanged on failure |
| Hymn to Tourach | `TestRONRandomDiscardReplacement`: real target and shared optional library replacement; both destinations visible to every discard event, selected order and RNG preserved |
| Supported spell/activation/source-entry/step-trigger contexts | `TestRONRandomDiscardContexts`: explicit typed player versus direct controller; event player binding; engine definition validation |
| Selected modes | `TestRONRandomDiscardModes`: target only for active mode, announcement-time choice, explicit target or controller, inactive body does not execute |
| Declarations/reporting | `TestRONRandomDiscardInvalid`, `TestRONRandomDiscardInventory`: wrong shapes/values/roles, undeclared player/event/X/payment reads, semantic amount role |
| Shared batch and replacement rechecks | `TestRandomDiscardBatchEventsAndReplacements`: ordinary/library/prevented/rechecked actions, hand-dependent replacements before movement, complete event visibility, no retry of prevented cards, continuation |
| Retained shared contracts | `TestTypedRandomDiscard*`: dynamic values read once, absent subject, negative/zero amounts, choices bypassed, replacements, short hand, validation, copies and independent clone random queues; existing selected-discard and payment suites |

### Fresh measurements and validation

Actual gain: **4 unique cards / 9 records**, versus **3 / 8 estimated**.
The extra card is Gwendlyn Di Corci, whose existing `IsTheirTurn` modifier was
more complete than the conservative estimate. No second capability was added.
Final **1,131 compiled / 1,314 rejected records**; **383 accepted / 603 rejected
unique names**. Exactly the nine records for the four mapped cards leave the
issue list. No previously accepted record is rejected, and every remaining
first diagnostic is unchanged. Hypnotic Specter, cost consumers and filtered
Rag Man remain rejected; their shared downstream value is not a gameplay claim.

Strict and semantic issue arrays agree. AST totals remain 72,938 occurrences /
8,528 signatures; unknown-role occurrences decrease from 14,405 to 14,398.
All twelve source hashes still match the vocabulary snapshot, so no source
snapshot refresh is needed. Engine-validated and behavior-verified collection
totals remain null/unmeasured. Tests supply full-record mappings for four more
unique cards; compilation totals are not collection gameplay completion.

| Command | Result |
| --- | --- |
| Baseline `go test ./pkg/mage/... -short -count=1`; `make build` | PASS |
| Baseline `go run ./cmd/loadcards -json`; `-inventory -json` | Expected exit 1: 1,122 compiled / 1,323 rejected |
| New engine/RON tests before implementation | Expected semantic failures recorded above |
| `go test ./pkg/mage ./pkg/mage/dsl/ron ./cmd/loadcards -count=1` | PASS |
| Focused `TestRandomDiscard*`, `TestTypedRandomDiscard*`, `TestRONRandomDiscard*`, `TestInventory*` with `-count=1` | PASS |
| `go test ./pkg/mage/... -short -count=1` | PASS |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `go test ./... -short -count=1` | PASS |
| `make lint` | PASS, zero issues; automatic import ordering in the new test files reviewed |
| Post-lint engine short tests and focused tests with `-count=1` | PASS |
| Post-review copy target-isolation test with `-count=1` | PASS |
| `git diff --check` | PASS |
| Final strict and semantic loaders | Expected exit 1: 1,131 compiled / 1,314 rejected; diagnostics agree and are stable |
| Accepted-record and vocabulary hash audit | PASS, no regressions and no source changes |

No environmental or baseline build/test failure blocked this slice. Collection
rejection remains expected and is not a validation failure. The new engine
behavior uses the existing batch service and preserves current API callers.
No DSL function was removed; no deletion caller audit is claimed. No Go card,
generator, authoring-instruction cutover or runtime wiring was changed because
the design's removal gates remain unmet.

Slice files: `pkg/mage/composable_actions.go`, new
`pkg/mage/random_discard_batch_test.go`; `pkg/mage/dsl/ron/compiler.go`,
new `random_discard.go` and `random_discard_test.go`, `inventory.go` and
`inventory_test.go`; `docs/dsl-reference.md`, `docs/engine-rules-contracts.md`,
and this ledger. Only these files are included in the slice commit.

Ranked next candidates to reassess with fresh evidence:
1. Shared CR 615.7 simultaneous shield allocation, then exact prevention lowering
   for seven otherwise-ready cards; preserve replacement choices and actual
   damage. This remains the largest concrete multi-card prerequisite.
2. Filtered bands-with-other engine contract and the five legendary lands;
   ordinary Banding cannot substitute for it.
3. Damage-event player/source snapshots and typed trigger reads; enable random
   discard for Hypnotic Specter and establish the shared event prerequisite for
   Nicol Bolas, Marsh Viper, Pit Scorpion and Nafs Asp. Inspect each complete
   record for its remaining discard/counter/delayed-result adapters first.

Complete RON gameplay, whole-collection validation, application startup loading,
Go-card/generator removal, obsolete DSL cleanup and native startup smoke tests
remain unfinished. This completed slice does not begin another capability.
