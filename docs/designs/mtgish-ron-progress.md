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
