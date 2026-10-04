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
