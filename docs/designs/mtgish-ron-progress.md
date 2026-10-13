# mtgish RON execution progress

Completed slices, test mappings, validation evidence and historical decisions
are in [mtgish-ron-completed.md](mtgish-ron-completed.md).
Use this document for current status and next work.

## Current status

Latest measured result (2026-10-13): **855 accepted / 131 rejected names**
(986 total), **2,197 compiled / 248 rejected records** (2,445 total).
The latest slice supplies typed announced X for entry counters. Rock Hydra's
six entry rules have casting, resolution and counter gameplay evidence.
Complete-card coverage is unchanged; Hydra now rejects static prevention.
The strict loader rejects the collection and publishes no partial registry.

The latest slice is **ENTRY-ANNOUNCED-X**. Permanent spell entry reads its own
stack object's X; direct entry supplies zero. Copies retain X; acquired entry
copy rules, clones, phasing, return and replacement order have contract tests.
Estimated and actual gains are zero names / zero records. This completes the
shared entry-X prerequisite for Rock Hydra and Frankenstein's Monster.
Entry payments, prevention and counter distribution remain. Runtime cutover,
Go-card removal and unused DSL cleanup gates remain open.

## Ranked next candidates

Reassess these candidates with fresh loader and complete-record evidence under
[the execution design](mtgish-ron-execution.md). Count reprints once.

1. **Finite self-recipient prevention.** Rock Hydra and Rasputin Dreamweaver
   share the shape. Audit the `ByPermanent` spelling against local Oracle text
   before lowering it. Zero immediate gain: Hydra also needs static
   counter-removal prevention; Rasputin needs turn-start history and a counter cap.
2. **Mandatory entry exile payment with announced X.** Frankenstein's Monster
   now has integer storage and entry X. Payment/result/combination-counter and
   failed-entry contracts remain; zero immediate gain for this prerequisite.
3. **Exchange outcomes and Aura-only correction.** Gauntlets of Chaos has exact
   targets and exchange. Estimated one name / one record after a local successful
   result and the Oracle-supported source correction. Drain Life's restricted
   casting X and clamped damage remain separate later candidates.

## Dependencies and source blockers

- Reuse `TapAnyNumberEach`, `PlayerNumbersBinding` and `PlayerNumber` for optional
  tap results. The RON producer requires one conjunctive ActionPlayer controller
  clause. Saved plural choices now use `ChooseUpToPermanentsEach` and `PermanentSetBinding`.
  `EachablePlayer` is local to the repetition count; other uses remain unsupported.
- Reuse `SacrificeAllButEach`, `LeastControlledPermanentCount`, the bounded
  subset selector and the existing simultaneous sacrifice service for permanent
  cardinality work. The cardinality adapter supports required kept-count choices in
  APNAP order; it does not add optional cardinality or saved plural results.
- Reuse `DiscardAllButEach`, `LeastHandSize`, exact hand selection and the shared
  discard batch for required hand cardinality. Optional discard counts/results
  retain their existing service and use the same selection helper.
- Reuse player-keyed permanent choices, APNAP choice groups and atomic control
  exchange where applicable. Juxtapose's APNAP source correction is complete.
- Gauntlets of Chaos has an open source defect at `cards/mtgish/LEG.lines.ron:95`:
  its attachment collection lacks Oracle's Aura-only restriction. Existing
  attachment collections can include Equipment. Preserve this defect explicitly
  when assessing its target and exchange-result prerequisites.
- Consult the completed archive for earlier source defects and capability limits
  when selecting other cards; historical next-step rankings are not current
  priorities.

## Remaining completion gates

Complete collection gameplay, runtime loading and application cutover remain
unfinished. Go-card removal and unused DSL removal must wait for their completion
gates in the execution design.

After each slice, move its completed evidence to the archive and update this
handoff with the measured status, remaining blockers and ranked next candidates.


## 2026-10-12 — HAND-CARDINALITY-DISCARD

### Candidate comparison and selection

The worktree was clean. Complete records were inspected for Raiding Party,
Balance in LEA/LEB/2ED/3ED/CED/CEI, City in a Bottle and Gauntlets of Chaos.
Targeted vocabulary entries confirmed six hand-cardinality occurrences, all
Balance reprints, and the separate optional-tap/count/choice/exclusion forms.
The complete Mind Bomb record in DRK was also read for shared-helper reuse.
Reprints count once in the following estimates.

| Candidate | Distinct affected names | Estimated full-card gains | Other blockers and downstream benefit | Estimated effort |
| --- | --- | --- | --- | --- |
| Bounded optional tap and saved plural selection | Raiding Party | 0 from tap/count prerequisite; 1 after later contracts | Needs actual per-player transitions, bounded plural results and `NotChosenThisWay`; shared selector exists, result adapters do not | Medium/large: several independently testable contracts; complete procedure exceeds this slice |
| Required hand cardinality | Balance | 1 name, 6 records | Land and creature instructions already lower; required choices and one simultaneous discard finish the card. Exact subset selection also serves existing optional discard and future result-bearing hand procedures | Small: one typed value, one engine operation, one lowering adapter and identity/batch tests |
| State-member / dependent-target prerequisites | City in a Bottle; Gauntlets of Chaos | 0 from either prerequisite alone | City needs captured state members plus global spell/land restrictions; Gauntlets needs first-target shared types and an exchange-result condition, plus the recorded Aura-only data defect. Separate services enable both names and broader restrictions/targets | Large: separate capture, player-rule and target/result contracts; not one complete slice |

Candidate #1 has no immediate multi-card unlock and still needs several engine
contracts. Among these candidates, there is no dependency-ready multi-name
full-card gain within one cohesive slice. Required hand cardinality gives the
largest credible immediate gain for the bounded effort. Estimated one-name /
six-record gain matches the actual gain exactly. No additional slice was started.

### Signatures, contexts and reuse boundary

- Engine: `LeastHandSize(players PlayerSet) ResolutionValue` and
  `DiscardAllButEach(players PlayerSet, keep ResolutionValue) ResolvingEffect`.
  Pure minimum reads include empty hands; absent/duplicate players are excluded.
  Counts below zero keep zero; counts above hand size cause no discard.
- RON: `EachPlayerAction(AnyPlayer, DiscardAllButNumberCards(count))`, where count
  is nonnegative `Integer(n)` or
  `TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer)`. This value spelling is
  local to the kept-count context. Resolving spell, activated, triggered and
  delayed action lists use the normal effect plan. Costs/static contexts,
  singular player actions, other RON player sets and other values still reject.
- Read the count once, snapshot all selected hands before choices, choose required
  discard cards in APNAP order without publishing hidden names, validate distinct
  selections and complete malformed responses from saved hand order. No saved
  plural result or global last-action state is introduced.
- `snapshotHandGroup` / `chooseHandDiscards` now also serve `DiscardUpToEach`.
  Its optional number choices and per-player actual-result bindings stay intact.
  Saved hand visits precede card choice in both services.
- Reuse `NewDiscardAction`, `discardActionsTogether`, replacement preparation,
  graveyard ordering, exact hand-visit checks and batch event dispatch. RON keeps
  spelling/arity/context checks; engine keeps reusable choices and movement.
  Existing delay/stack/clone services retain their execution frames. Typed source
  reads in the new count and operation support pure waiting queries.
- CR 101.4/101.4a, 400.7, 608.2c–f/608.2h and 701.9a–c supply ordering, hidden
  choice, identity, information timing and discard contracts. No source defect
  was found or corrected in Balance. RON source files are unchanged, so no
  vocabulary snapshot refresh is needed. The Gauntlets source defect remains.

### Failing-first tests and card mapping

Before implementation, engine tests failed to compile at the missing typed APIs.
The real-record RON test then failed in all six reprints at the expected
`unsupported all-player action (DiscardAllButNumberCards)` diagnostic.

- **Balance (LEA/LEB/2ED/3ED/CED/CEI)** →
  `TestRONHandCardinalityBalance`: actual casting and complete land/hand/creature
  instructions; discard events observe completed land sacrifice and complete
  hand movement before the creature round. Previous
  `TestRONPermanentCardinalityBalanceInstructions` retains payment failure,
  clone isolation and individual-round evidence; its obsolete full-card
  rejection assertion now requires successful compilation.
- `TestRONHandCardinalityCopiesClonesAndDelay`: stack copies, independent cloned
  hands, empty minimum and a delayed operation after source departure/return.
- `TestRONHandCardinalityInvalid`, `InventoryAndAtomicLoad`: wrong shapes,
  arities, negative counts, player collections, roles and unbound values; typed
  inventory roles and rejection without a partial registry.
- `TestHandCardinalityChoicesAndBatch`, `BoundsLeastAndDeclarations`,
  `ReplacementsAndVisits`, `ReadsOnceAndWaiting`, `ChoiceVisitsAndSnapshot`:
  required distinct choices, both APNAP orders, hidden prompts, one batch,
  empty/duplicate/absent subjects, malformed fallback, cancellation, library
  destinations, returned visits, saved pre-choice candidates, once-only value
  reads and pure waiting queries through phasing.
- **Mind Bomb (DRK)** → existing `TestRONPlayerDiscardResultsMindBomb` and
  `TimingRollbackAndPrevention` retain complete optional-discard/damage gameplay
  after selection-helper reuse. This is preserved coverage, not a new unlock.
- Existing `TestPlayerDiscardResults*` and `TestSelectedDiscardBatch*` preserve
  optional per-player outcomes, replacement preparation/rechecks, graveyard
  ordering, exact visits and frame/clone behavior after helper reuse.

Compilation coverage and gameplay coverage remain separate: this entry adds
one complete-card gameplay mapping, not gameplay certification for all 845
accepted names. No Go definitions or broad DSL functions were removed; their
cutover/removal gates are not satisfied.

### Changed files

- Engine: `pkg/mage/hand_cardinality.go`, `hand_cardinality_test.go`,
  `player_numbers.go`, `waiting_delayed_source.go`, `doc.go`.
- RON: `pkg/mage/dsl/ron/hand_cardinality.go`, `hand_cardinality_test.go`,
  `each_player.go`, `inventory.go`, `permanent_cardinality_test.go`,
  `definition_plan_test.go`, `inventory_test.go`, `numeric_consumers_test.go`.
- Documentation: this ledger, `docs/engine-rules-contracts.md` and
  `docs/dsl-reference.md`.

### Verification and measured collection result

Fresh baseline:

- `make build`: pass.
- `go test ./pkg/mage/dsl/ron -count=1`: pass before edits.
- `go run ./cmd/loadcards -json`: expected exit 1; 2,180 compiled / 265 rejected
  records, 844 accepted / 142 rejected unique names. This was collection
  rejection, not a build or environment failure.

Final validation after the preallocation correction:

- `go test ./pkg/mage/dsl/ron -run 'TestRON(HandCardinality|PlayerDiscardResults)' -count=1`: pass.
- `go test ./pkg/mage -run 'Test(HandCardinality|PlayerDiscardResults|SelectedDiscardBatch)' -count=1`: pass.
- `make build`: pass.
- `go vet ./...`: pass.
- `go test ./pkg/mage/... -short -count=1`: pass.
- `make test`: pass.
- `make lint`: pass, 0 issues. The first run found one preallocation issue in
  the new discard list; it was corrected and affected tests, build, vet, engine
  short tests and the full suite were rerun before the final lint run. Automatic
  formatting changes were reviewed; no unrelated files changed.
- `git diff --check`: pass.
- `go run ./cmd/loadcards -json`: expected exit 1; **2,186 compiled / 259 rejected
  records**, **845 accepted / 141 rejected names**. All six Balance diagnostics
  disappeared. No new diagnostic appeared; every other baseline diagnostic is
  unchanged. Strict loading still publishes no partial registry.

Four collection-count assertions initially reported the new 2,186/259 totals;
these were updated to the measured exact counts, without removing checks.
Remaining examples: Raiding Party still rejects at `TapAnyNumberOfPermanents`
(FEM:71:381), City in a Bottle at `WhenAnyNumberOfPermanentsAreOnTheBattlefield`
(ARN:13:146), and Gauntlets of Chaos at the two-argument shared-cardtype predicate
(LEG:95:416). There is no environment or pre-existing validation blocker.

### Ranked next candidates

1. Raiding Party optional tap selection with actual per-player counts: zero
   immediate full-card gains; required before bounded plural choices/exclusions.
2. Shared dependent permanent target comparison for Gauntlets of Chaos: zero
   immediate gains; shared-type target legality precedes exchange-result tests.
3. Global player spell/land restrictions and City in a Bottle state capture:
   zero gains from one prerequisite; compare reusable land-play restrictions
   with Worms of the Earth before selecting the next bounded slice.

## 2026-10-12 — OPTIONAL-PLAYER-TAP-RESULTS

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and the complete RON package test
passed. The loader exited 1 because unsupported collection records remain, not
because of build, test or environment failure. Complete RON records were read
for Raiding Party (FEM), Gauntlets of Chaos (LEG), City in a Bottle (ARN) and
Worms of the Earth (DRK). Targeted vocabulary entries confirmed their nested
producer, result, target and restriction forms. Reprints count once.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers / downstream capabilities | Implementation and test effort |
| --- | --- | --- | --- | --- |
| Optional tapping with per-player actual counts | Raiding Party | 0 now; 1 after plural kept choices and exclusions | Shared bounded subset selection and number frames exist. The missing producer is ready; saved plural choices, its EachablePlayer read and NotChosenThisWay still need contracts. Enables a complete bounded plural continuation without global action state. | Small/medium: one engine operation, lexical producer/read adapter, batch/identity/frame tests |
| Dependent permanent target shared-type comparison | Gauntlets of Chaos | 0 now; 1 after exchange-result and source repair | First target declarations and atomic exchange exist. Second-target legality needs a shared-type comparator; PermanentsChangedControlThisWay remains missing. Attachment collection still lacks Oracle's Aura-only filter. | Medium: dependent announcement/resolution target contracts, retargeted copies, source defect evidence |
| Global land-play restrictions | City in a Bottle; Worms of the Earth | 0 from land restrictions alone | City also needs captured state members and filtered spell restrictions. Worms also needs entry prohibition and alternative damage/sacrifice payment. Shared land restrictions benefit both names. | Medium/large: land-filter rules plus real play permissions, static lifecycle and clone tests |

The first ranked candidate is an unblocking, dependency-ready prerequisite and
fits one run. It reuses shared services and introduces one cohesive resolving
operation with its typed result. The other candidates also have zero immediate
unlocks and require separate contracts to finish their cards. Estimated zero
new complete names/records matches the actual result. This slice adds real RON
instruction evidence, not a complete-card mapping or runtime cutover.

### Signatures, supported contexts and reuse boundary

- Engine: `TapAnyNumberEach(players PlayerSet, subjects PermanentSet,
  result *PlayerNumbersBinding) ResolvingEffect`. Restrict subjects to the
  chooser's present untapped controlled permanents. Snapshot the entire set and
  both players' groups before choices. Duplicate/absent players, duplicate
  subjects, old visits and phasing do not contribute.
- Choices use APNAP order. Offer counts from zero to candidate count; clamp
  malformed counts. Shared `choosePermanentSubset` supplies distinct saved
  candidates and legal fallback for malformed responses. Previous public choices,
  including empty choices, appear in later players' prompts. Summoning sickness
  does not restrict this resolving effect.
- Recheck presence, exact incarnation, untapped state and control before one
  `tapPermanentsTogether` call. Save only actual transitions, before tap events.
  All tap states are visible before any event. Every invocation clears both
  result members; later state changes do not revise those historical counts.
- RON: `EachPlayerAction(AnyPlayer, TapAnyNumberOfPermanents(filter))` in
  resolving spell, activated, triggered or delayed action lists. Require exactly
  one conjunctive `ControlledByAPlayer(SinglePlayer(ActionPlayer))`, standalone
  or in nested And lists. Other clauses use existing contextual collections.
  RON removes the chooser-controller spelling; the reusable engine operation
  enforces control. Disjunctive controller clauses and other players reject.
- `NumberOfPermanentsTappedThisWayByPlayer(playerRef)` reads the most recent
  preceding lexical producer through `PlayerNumber`. Ordinary typed references
  and bound PlayerActions `ActionPlayer` use existing scopes. Branch producers
  stay local; reads before a producer, ambiguous contexts and unbound
  `EachablePlayer` reject. The latter belongs to the deferred plural procedure.
- Fresh factory declarations use `BindPlayerNumbers`; shared resolution frames,
  copies, clones and AtNext captures retain separate numbers. Explicit typed
  player/subject source reads support pure waiting queries. RON retains all
  spelling/context checks; engine owns choices, exact visits and batching.
- No cost or tap replacement pipeline was added: this is an effect, not a
  tap-symbol payment. Existing activation costs still validate/roll back through
  their shared transaction. The batch service and its existing cost callers are
  unchanged. No Go-card fallback, partial registry, game-time declaration panic,
  global last-action slot or card-name branch was introduced.
- CR 101.4/101.4b, 400.7, 608.2d–f/608.2i and 701.26a establish public choice
  order, exact visits, resolution timing, simultaneous operations and historical
  tap facts. Local `data/FEM.json` supplies Raiding Party's Oracle requirements.
  No source defect was found or repaired here. The Gauntlets defect remains.
  RON source records are unchanged; no vocabulary source refresh is needed.

### Failing-first evidence and card-to-test mapping

Before implementation, engine tests failed to compile only because
`TapAnyNumberEach` was missing. Once test-fixture compilation errors were fixed,
RON tests failed at `unsupported all-player action (TapAnyNumberOfPermanents)`
for the actual record instruction and synthetic saved-result bodies.

- **Raiding Party (FEM)** → `TestRONPlayerTapResultsRaidingPartyInstruction`:
  exact first activated instruction, actual Orc sacrifice, failed payment,
  choices only at resolution, each player's chosen count, white untapped
  creature filtering, phased exclusion and source departure. The full record
  continues to reject at `ChooseUptoNumberPermanentsForEach`. Existing
  `TestRONAbilityTargetRestrictionRaidingParty` keeps static rule coverage;
  only its expected remaining diagnostic changed.
- `TestRONPlayerTapResultsScopesCopiesClonesAndDelay`: stack copies, independent
  resolving producers, simulation choices, saved delayed counts and resets.
- `TestRONPlayerTapResultsActivationSources`: saved controller/result across
  source control changes, phasing and return; invalid cost/static contexts.
- `TestRONPlayerTapResultsInvalid`: declaration arities, malformed collections,
  missing/duplicate/disjunctive ActionPlayer clauses, wrong player references,
  unbound reads, use before producer and branch escape.
- `TestRONPlayerTapResultsInventoryAndAtomicLoad`: typed result player roles,
  accepted synthetic declaration and rejection without partial publication.
- `TestPlayerTapResultsChoicesAndBatch`: both APNAP orders, optional distinct
  choices, public prior choices, one batch and saved numbers before events.
- `TestPlayerTapResultsVisitsAndActualTransitions`: pre-tap, phase, departure /
  return and loss of chooser control after selection cannot supply tap results.
- `TestPlayerTapResultsBoundsResetAndDeclarations`: negative/excessive counts,
  malformed subset answers, duplicate/absent players, duplicate/phased/tapped /
  old subjects, empty results and definition nil/dominance validation.
- `TestPlayerTapResultsSnapshotAndWaiting`: pre-choice membership capture,
  immutable historical results, pure source queries, source phase/departure /
  return, and independent cloned delayed lifetime.

The test fixture initially assumed cloned players would choose zero; existing
BasePlayer/SearchPlayer contracts choose the maximum outside discard contexts.
The test now asserts the exact independent maximum choice and delayed saved
count. Another fixture lacked a source owner before destruction; it now assigns
one. Neither correction changes production contracts or weakens assertions.

### Changed files

- Engine: `pkg/mage/player_tap_results.go`, `player_tap_results_test.go`, `doc.go`.
- RON: `pkg/mage/dsl/ron/player_tap_results.go`, `player_tap_results_test.go`,
  `compiler.go`, `player_scope.go`, `optional_payment.go`, `sacrifice_value.go`,
  `inventory.go`, `ability_target_restriction_test.go`.
- Documentation: this ledger, `docs/engine-rules-contracts.md`,
  `docs/dsl-reference.md` (including the earlier remaining-diagnostic statement).

No Go card definition or DSL function was removed. Complete loading, gameplay,
startup, test migration and removal gates remain unsatisfied.

### Verification and measured collection result

Fresh baseline:

- `make build`: pass.
- `go test ./pkg/mage/dsl/ron -count=1`: pass.
- `go run ./cmd/loadcards -json`: expected exit 1; 2,186 compiled / 259 rejected
  records; 845 accepted / 141 rejected names. This is collection rejection.

Final checks, rerun after the lint corrections and bound ActionPlayer test:

- `go test ./pkg/mage/dsl/ron -run TestRONPlayerTapResults -count=1`: pass.
- `go test ./pkg/mage -run TestPlayerTapResults -count=1`: pass.
- Additional focused `TestRON(PlayerTapResults|AbilityTargetRestrictionRaidingParty|PlayerDiscardResults)` and
  `Test(PlayerTapResults|PermanentCardinality|PlayerDiscardResults|TapNMatchingIncludingSource)` runs: pass.
- `make build`: pass.
- `go vet ./...`: pass.
- `go test ./pkg/mage/... -short -count=1`: pass.
- `make test`: pass.
- `make lint`: pass, 0 issues. First run reported two test-slice preallocation
  issues. Capacities were fixed, a positive bound ActionPlayer read was added,
  and focused tests, build, vet, engine short tests and the full suite were
  rerun before the successful lint run. Automatic changes only grouped imports
  in the new tests; these were reviewed. No unrelated file changed.
- `git diff --check`: pass.
- `go run ./cmd/loadcards -json`: expected exit 1; **2,186 compiled / 259 rejected
  records**, **845 accepted / 141 rejected names** (2,445 records / 986 names).
  Comparing the complete baseline/final issue arrays shows only one diagnostic
  replacement: Raiding Party changes from `TapAnyNumberOfPermanents`
  (FEM:71:381) to `ChooseUptoNumberPermanentsForEach` (FEM:71:549). All other
  diagnostics and previously accepted records are unchanged. Strict loading
  still publishes no partial registry.

There is no environment or pre-existing validation blocker. No collection-count
assertion needed changing. Zero immediate full-card gains match the estimate;
Raiding Party gains instruction coverage and a dependency for its continuation.
Compilation counts and complete gameplay mappings are unchanged. The commit
contains only this completed slice, tests and documentation.

### Ranked next candidates

1. Bounded plural kept choices and exclusions for Raiding Party. Reuse optional
   tap counts and the subset selector; establish a saved plural result,
   `ChooseUptoNumberPermanentsForEach`, its local EachablePlayer count read and
   `NotChosenThisWay`. Estimate one complete-card gain after these contracts.
2. Shared dependent permanent target comparison for Gauntlets of Chaos. Zero
   immediate gains; shared-type target legality precedes exchange-result tests.
   Its open Aura-only source defect still needs an evidence-based repair.
3. Shared land-play restrictions for City in a Bottle and Worms of the Earth.
   Zero gains from that prerequisite alone; compare two-card shared value with
   City state-member capture and filtered spell restrictions before selection.


## 2026-10-12 — SAVED-PLURAL-PERMANENT-CHOICES

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The baseline strict loader exited 1
with 2,186 compiled / 259 rejected records and 845 accepted / 141 rejected names.
This was collection rejection, not an environment or compilation failure.
Complete RON records were read for Raiding Party (FEM), Gauntlets of Chaos (LEG),
City in a Bottle (ARN), and Worms of the Earth (DRK). Targeted vocabulary entries
confirmed one plural-choice constructor and one exclusion symbol, both Raiding
Party. None of these four names has a collection reprint to count separately.

| Candidate | Distinct affected names | Estimated full-card gains | Other blockers / downstream value | Estimated implementation and test effort |
| --- | --- | --- | --- | --- |
| Saved bounded plural choices and exclusions | Raiding Party | 1 name / 1 record | Tap counts and subset choices already exist. General saved sets and exact set difference serve later plural choice procedures | Small/medium: one producer, scoped adapter, shared saved-set API and frame/identity tests |
| Dependent shared-cardtype target comparison | Gauntlets of Chaos | 0 immediately; 1 after later work | Needs exchange-result condition and the recorded Aura-only source repair. Enables dependent target legality | Medium/large: announcement and resolution comparison, saved first target, exchange result and source audit |
| Shared land-play restrictions | City in a Bottle; Worms of the Earth | 0 immediately; 2 after separate prerequisites | City needs captured state members and filtered casting restrictions. Worms needs entry prevention and alternative sacrifice/damage payments. Enables shared player restrictions | Large for full cards; medium for the isolated restriction and legality tests |

Candidate #1 is dependency-ready and completes an already established procedure.
There is no dependency-ready multi-name full-card unlock among these three
candidates. Its immediate gain and reusable plural-set service give the largest
credible gain for the bounded effort. The two-name land prerequisite remains
useful next work; it would not complete either card in this slice.

### Established contract, signatures and reuse boundary

Supported RON resolving shape:

`EachPlayerAction(AnyPlayer, ChooseUptoNumberPermanentsForEach(Integer(n), filter,
NumberOfPermanentsTappedThisWayByPlayer(EachablePlayer)))`, n >= 0, after a local
tap producer. `EachablePlayer` is accepted only in this repetition count.
`NotChosenThisWay` is a symbol in later contextual permanent collections.
Branch-local results do not escape; fresh producers shadow earlier plural sets;
delayed bodies retain the saved result. Other player sets, count forms, costs,
static predicates and target filters still reject. Inventory arguments have
value / permanent-filter / value roles and a typed local player reference.

Engine APIs:

- `BindPermanentSet(name string) *PermanentSetBinding`.
- `FromPermanentSet(binding *PermanentSetBinding) PermanentSet`.
- `WithoutPermanents(subjects, excluded PermanentSet) PermanentSet`.
- `ChooseUpToPermanentsEach(players PlayerSet, subjects PermanentSet,
  limit ResolutionValue, repetitions *PlayerNumbersBinding,
  result *PermanentSetBinding) ResolvingEffect`.

The engine reads counts, limit and candidates before choices. Each player makes
one separate zero-to-limit selection per saved repetition, in APNAP order.
Subjects may belong to either player. Within one choice, members are distinct;
later choices and players may overlap. Later players see earlier public choices.
Malformed counts are clamped and malformed subjects use the existing distinct
subset fallback. The result is the union of exact saved visits, cleared on every
invocation including empty choices. Live exclusions read present visits; phasing
keeps identity but stops actions, and a return cannot inherit an earlier choice.

`CreatedTokensBinding` is now a compatibility alias for `PermanentSetBinding`.
Existing token callers remain intact. The existing exact-visit result storage,
deep clone, resolution/loop frames, delayed capture and waiting/stack source
queries serve both producers with separate named bindings. Definition validation
checks nil operands, binding names and producer dominance. RON owns spelling,
argument checks and lexical plans; the engine owns choices and visit identity.

No new cost transaction or replacement pipeline was added. Choices are effects
and make no zone changes. Raiding Party continues to use existing Orc sacrifice
payment, optional taps and actual counts, current Plains filters and simultaneous
`DestroyEach` with its destruction/regeneration/replacement behavior. No Go-card
fallback, card-name branch, partial registry, global result slot or game-time
declaration panic was introduced. No Go card definition or DSL API was removed;
the design's cutover gates remain open.

CR 101.4/101.4b/101.4c, 400.7, 608.2d–f and the existing tap contract establish
choice order, exact visits and resolution timing. Local `data/FEM.json` confirms
any player's Plains and one up-to-two selection per actual tapped creature.
No RON source defect was found or repaired. The Gauntlets Aura-only defect remains.
RON source records did not change, so no vocabulary source refresh was needed.

### Failing-first evidence and card-to-test mapping

Engine tests first failed to compile because `BindPermanentSet`,
`FromPermanentSet`, `WithoutPermanents` and `ChooseUpToPermanentsEach` did not exist.
After correcting the RON fixture's Parse arguments, its complete-record test
failed at `unsupported all-player action (ChooseUptoNumberPermanentsForEach)`.
These failures preceded implementation.

- **Raiding Party** → `TestRONPermanentSetChoiceRaidingParty`: complete FEM
  record; failed Orc payment, no choices during activation, actual tap count,
  repeated bounds including zero, opponent Plains, source departure, phased
  exclusion and independent non-Plains survival. The existing
  `TestRONPlayerTapResultsRaidingPartyInstruction` retains white/untapped/type
  filtering, failed payment and both players' actual-count evidence.
- **Raiding Party static rule** → `TestRONAbilityTargetRestrictionRaidingParty`
  now tests the complete factory and keeps white spell/ability source coverage.
- `TestRONPermanentSetChoiceCopiesClonesDelayAndScope`: copied stack producers,
  independent clone choices and separate delayed saved sets.
- `TestRONPermanentSetChoiceInvalid`: malformed arity, negative/variable bounds,
  wrong players, unbound/early reads and incompatible exclusion shapes.
- `TestRONPermanentSetChoiceBranchDominance`: valid local branch producer/read;
  escape rejects specifically with an unbound plural-choice diagnostic.
- `TestRONPermanentSetChoiceInventoryAndAtomicLoad`: exact count-player role,
  accepted declaration and rejection without publishing a partial registry.
- `TestPermanentSetChoiceRepeatedAPNAPAndOverlap`: both active-player orders,
  separate repeated choices, overlap and opponent-controlled subjects.
- `TestPermanentSetChoiceVisitsResetCloneAndDelay`: set difference, reset,
  phase cycle, returned visits, clone independence and saved delay.
- `TestPermanentSetChoiceInvalidDefinitions`: nil operands and unbound reads.
- `TestPermanentSetChoiceBoundsDistinctVisitsAndWaiting`: clamped counts,
  distinct selections across repetitions, duplicate/absent players, duplicate /
  phased / old / nil subjects, explicit source reads and pure waiting queries.
- `TestPermanentSetChoiceMalformedSubsetFallback`: invalid answers receive two
  distinct legal subjects.

Fixture corrections changed Parse argument order, PhaseIn's slice argument and
the complete card's activation index (its static rule adds earlier abilities).
The inventory test found a missing player-action role adapter; it was corrected.
These changes did not weaken gameplay assertions or alter payment semantics.
Collection-count tests now assert the measured 2,187 / 258 record counts.

### Changed files

- New engine `permanent_set_choice.go` and `permanent_set_choice_test.go`;
  compatibility/saved-set declarations in `composable_target.go`, explicit set
  difference source reads in `waiting_delayed_source.go`, and `doc.go`.
- New RON `permanent_set_choice.go` and `permanent_set_choice_test.go`; compiler
  scopes, instance bindings, action sequences, contextual collections, inventory
  roles, existing static/tap tests and three measured-count regression files.
- This ledger, `docs/engine-rules-contracts.md` and `docs/dsl-reference.md`.

### Verification and measured collection result

Required checks passed:

- `go test ./pkg/mage/dsl/ron -run TestRONPermanentSetChoice -count=1`.
- `go test ./pkg/mage -run TestPermanentSetChoice -count=1`.
- Additional focused `TestRON(PermanentSetChoice|PlayerTapResults|AbilityTargetRestrictionRaidingParty)`
  and `Test(PermanentSetChoice|PlayerTapResults)` runs passed.
- `make build` and `go vet ./...`.
- `go test ./pkg/mage/... -short -count=1`.
- `make test`.
- `make lint`: pass, 0 issues. Import formatting was reviewed; only this slice's
  files changed. No lint behavior correction was needed. Focused tests, build,
  vet, engine short tests and the full suite passed again after formatting and
  the added branch-dominance check.
- `git diff --check`.
- Final `go run ./cmd/loadcards -json`: expected exit 1; **2,187 compiled / 258
  rejected records**, **846 accepted / 140 rejected names** (2,445 / 986 totals).
  Full baseline/final diagnostic comparison removes only Raiding Party's
  `ChooseUptoNumberPermanentsForEach` issue at FEM:71:549, adds no issue and
  changes no remaining diagnostic. Previously accepted records remain accepted.
  Strict loading still publishes no partial registry.

No environment or pre-existing validation blocker remains. The completed slice
is ready for the authorized `feat(ron): implement saved plural permanent choices`
commit. No other capability slice was started.

The estimated one-name / one-record gain matches the measured result exactly.
Raiding Party now has complete RON behavior mappings for both rules. Compilation
and gameplay coverage remain separate; no other card gained a gameplay claim.
Runtime loading, native startup, Go-card deletion and DSL cleanup remain open.

### Ranked next candidates

1. Shared land-play restrictions for City in a Bottle and Worms of the Earth.
   Two downstream unique names, zero immediate full-card gains. Build on current
   player legality services; keep City's captured state/casting restrictions and
   Worms' entry/payment contracts separate.
2. Shared dependent permanent target comparison for Gauntlets of Chaos. Zero
   immediate gains; shared-type target legality precedes exchange-result tests
   and the evidence-based Aura-only source correction.
3. Captured state-member events for City in a Bottle. Zero immediate gains until
   its casting and land restrictions exist. Exact member snapshots must retain
   simultaneous-event, source-visit and pending-trigger facts.

## 2026-10-12 — STATIC-LAND-PLAY-RESTRICTIONS

### Candidate comparison and selection

The worktree was clean. Read AGENTS.md, the engine reference, execution design,
top status block and latest ledger entry. Inspected all complete records for
City in a Bottle (ARN), Worms of the Earth (DRK) and Gauntlets of Chaos (LEG),
including nested declarations. Targeted vocabulary entries have one occurrence
of each land restriction. Names are counted once; these records have no
collection reprints. Local data/ARN.json and data/DRK.json confirm the land rules.

| Candidate | Distinct affected names | Estimated complete-card unlocks | Other blockers | Downstream value | Effort |
| --- | --- | --- | --- | --- | --- |
| Shared land play restrictions | City in a Bottle; Worms of the Earth | 0 | City: captured state members, filtered casting and mixed player-rule list. Worms: land entry prevention, optional alternative sacrifice/damage payment and its result. | Both cards share one land legality contract and lifecycle; fixed-name predicates can serve City's casting rule. | Small engine query and adapters; lifecycle, rollback, declaration and exact RON subrule tests; one run. |
| Dependent permanent target comparison | Gauntlets of Chaos | 0 | Exchange-result condition; Aura-only source correction. | Shared-card-type legality for separate target declarations, before exchange outcomes. | Medium target/declaration work; announcement, partial legality, identity, phasing and copies; larger than selected work. |
| Captured state-member events | City in a Bottle | 0 | Filtered casting, land restriction and mixed player-rule list. | Typed plural state-trigger snapshots and controller sacrifices. | Medium event/binding work; simultaneous members, pending facts, rearming, returned visits and copies. |

Candidate #1 is dependency-ready and supplies the concrete two-card prerequisite
listed in the previous entry. It needs no missing target/event binding service.
The other candidates currently serve one unique card each. Selected one complete
static restriction shape rather than adding their dependent mechanics. Estimated
zero complete-card unlocks matches the final measured gain of zero. Two exact
land subrules gained gameplay evidence; neither card gained a full gameplay claim.

Fresh baseline: `make build` and `go test ./pkg/mage/dsl/ron -count=1` passed.
`go run ./cmd/loadcards -json` exited 1 as expected: 2,187 compiled / 258 rejected
records, 846 accepted / 140 rejected unique names. This is collection rejection,
not a build/test failure. Baseline diagnostics and output were saved in /tmp.

### Supported contract and engine boundary

Permanent top-level Rules accept exactly one operation in
`EachPlayerEffect(AnyPlayer, [CantPlayLands])` or
`EachPlayerEffect(AnyPlayer, [CantPlayLandsOfType(predicate)])`.
The predicate is the symbol SharesANameOriginallyPrintedInArabianNights or
SharesANameOriginallyPrintedInAntiquities. Tuple forms, wrong arity, other
player scopes, arbitrary filters, permanent references, resolving/cost contexts
and mixed operation lists remain declaration errors. Inventory classifies the
filtered operand as `land_play_card_filter`.

RON owns spelling, shape validation, catalog selection and lowering. The engine
owns `PreventLandPlays(filter CardFilter) StaticRule` and
`Game.IsLandPlayProhibited(card Card) bool`. An explicit all-card filter supplies
the unconditional rule; a zero filter fails definition validation. Each factory
builds a fresh card, static declaration and effect. No player/result binding is
needed for this all-player static query. Fixed-name predicates use the existing
immutable embedded CR 206.3 ARN/ATQ catalogs; physical set membership is irrelevant.

The query reads active typed effects through the existing printed-ability,
incarnation and ability-removal service. Source departure, phasing, face-down
status and ability removal stop the rule. Tapping and control changes do not.
Phase cycles preserve visits; return installs a new visit. Clones have independent
effect/source lifetimes. No cached per-cycle filter membership or global result
slot is added. The first cached-filter implementation failed the phasing tests
because PhaseOut need not run a layer cycle; direct active-effect queries resolved
that failure without changing the shared phasing service.

PlayLand checks the candidate before removing it from hand or a permitted library
top. Failure preserves hand/library order and land-play history. GetPlayableLands
uses the same restriction for its existing hand enumeration. Extra/unlimited
plays do not override prohibition. Direct entry and spell casting retain their
independent services. Legacy global/expansion restrictions remain compatible.
CR 101.2, 305.1–4, 206.3, 400.7, 613.11 and 702.26b–d establish these contracts.
No new zone move, cost transaction, LKI action, choice order or replacement
pipeline is introduced. Simultaneous ordering is not part of this pure legality
query. No Go definitions, DSL exports or generators were removed; cutover gates
remain open. No card-name branches or production partial registry were added.

### Failing-first evidence and card-to-test mapping

Before implementation, engine tests failed to compile because PreventLandPlays
and IsLandPlayProhibited did not exist. After fixing test-only Parse arguments
and the inventory call, RON tests failed with unsupported all-player operation
errors for CantPlayLands and CantPlayLandsOfType. Those failures preceded code.

- **City in a Bottle** → `TestRONLandPlayRestrictionRealRules`: its exact nested
  land operation in an explicitly isolated subrule; both active players, Desert
  name on a synthetic printing, unrestricted ARN Mountain, direct entry and
  casting separation, failed-play state and immediate source phasing. Complete
  record still rejects its unsupported state-member trigger.
- **Worms of the Earth** → the same real-rule test: exact unconditional land
  subrule, both players, all land names, direct entry/casting separation and
  source phasing. Complete record still rejects unsupported sibling behavior.
- `TestRONLandPlayRestrictionCatalogAndInstances`: ATQ catalog, separate factory
  instances, overlapping source lifetime, catalog separation and another game.
- `TestRONLandPlayRestrictionInvalid`: malformed symbols/arity, player scope,
  wrong reference/filter and context, and mixed operations.
- `TestRONLandPlayRestrictionInventoryAndAtomicLoad`: accepted declaration,
  semantic card-filter role, and invalid collection publishes no registry.
- `TestLandPlayRestrictionLifecycle`: both active players, tapped/control,
  face-down/phased/cycle, removal/expiry, departure/return and clone independence;
  pure queries, extra-play permission, blocked/allowed plays and direct entry.
- `TestLandPlayRestrictionLibraryAndOverlap`: permitted library top, exact
  library order/history on failure, overlapping source removal and cloned rule.
- `TestLandPlayRestrictionHandOrderAndPermissions`: exact hand order, nonland
  rejection, unlimited permission, nil/nonland queries, no stack/history change.
- `TestLandPlayRestrictionInvalid`: empty engine filter fails validation.

Fixture fixes supplied Parse's reader/file arguments, used InspectFS, set a
synthetic source owner, preserved the actual hand count after a source bounce,
and supplied an extra land play before the City's post-phasing play. No gameplay
assertion was removed to accommodate implementation behavior.

### Changed files

New engine `pkg/mage/land_play_restriction.go` and its test; land announcement in
`land.go` and playable-land enumeration in `game.go`. New RON
`land_play_restriction.go` and its test; all-player dispatch in
`public_visibility.go` and semantic operand role in `inventory.go`.
API/contracts updated in `pkg/mage/doc.go`, `docs/engine-rules-contracts.md` and
`docs/dsl-reference.md`; this ledger updates the status and appends evidence.
No source defect was corrected. The Gauntlets Aura-only defect remains open.
RON source did not change, so the vocabulary snapshot did not need a refresh.

### Verification and measured collection result

Required checks passed in the requested fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONLandPlayRestriction -count=1`.
- `go test ./pkg/mage -run TestLandPlayRestriction -count=1`.
- `make build` and `go vet ./...`.
- `go test ./pkg/mage/... -short -count=1`.
- `make test`; also `go test ./... -short -count=1` for the engine skill.
- `make lint`: 0 issues. Reviewed automatic formatting and final source; no
  behavior or additional file changes occurred. No affected rerun was required.
- `git diff --check`.
- Final `go run ./cmd/loadcards -json`: expected exit 1, **2,187 compiled / 258
  rejected records**, **846 accepted / 140 rejected names** (2,445 / 986 totals).
  Baseline/final comparison adds or removes no rejected record key. Only Worms
  of the Earth's diagnostic changes: DRK:118:204 CantPlayLands becomes
  DRK:118:221 CardsCantEnterTheBattlefield. All other diagnostics are unchanged;
  previously accepted records remain accepted. The loader still rejects the
  collection and publishes no partial registry.

No environment or pre-existing validation blocker remains. Compilation coverage
is unchanged, as estimated; gameplay evidence now covers the two isolated land
subrules and the shared engine contract. Full-card gameplay coverage remains
unchanged. This completed slice is ready for the authorized
`feat(ron): implement static land play restrictions` commit.

### Ranked next candidates

1. Filtered casting restrictions with compound all-player rule lists for City in
   a Bottle. Reuse this land rule and immutable name catalogs; captured state
   members remain separate. Zero immediate full-card gains.
2. Dependent permanent target comparison for Gauntlets of Chaos. Shared-type
   target legality precedes exchange-result handling and the Aura-only source
   correction. Zero immediate full-card gains.
3. Land entry prevention for Worms of the Earth. Reuse the entry service and
   keep optional alternative payments/results separate. Zero immediate gains.

Reassess shared multi-card opportunities from fresh inventory before selecting
another prerequisite. No second slice was started in this run. Complete RON
gameplay, runtime loading, native startup, Go-card deletion and DSL cleanup
remain unfinished.

## 2026-10-12 — FILTERED-CASTING-AND-PLAYER-RULE-LISTS

### Fresh environment and candidate comparison

The initial working tree was clean. Fresh `go test ./pkg/mage/dsl/ron -count=1`
and `make build` passed. The baseline loader exited 1 because the collection
still has unsupported records: 846 accepted / 140 rejected unique names,
2,187 compiled / 258 rejected records, from 986 names and 2,445 records.
Collection rejection was not a compilation or environment failure.

Evaluated the latest ranked candidates first. Read the complete City in a
Bottle (ARN:13), Gauntlets of Chaos (LEG:95), and Worms of the Earth (DRK:118)
records, including nested actions, filters, costs and branches. Targeted
vocabulary signatures agree with their source shapes. Each occurs once; no
reprints add unique value.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers | Downstream behavior | Estimated effort |
| --- | --- | --- | --- | --- | --- |
| Filtered casting and compound all-player rules | City in a Bottle | 0 | Captured state-member trigger and plural event subjects | Completes City's player-rule declaration; reusable card-filter cast legality and composition of supported player rules | One run, about 250–350 test lines; ready |
| Dependent permanent target comparison | Gauntlets of Chaos | 0 | Exchange-result predicate; recorded Aura-only attachment source defect | Two-target shared-type legality, then conditional attachment destruction | About 400–600 test lines; more identity and target coupling |
| Land entry prevention | Worms of the Earth | 0 | Alternative optional sacrifice/damage payment and result scope | Global entry restriction and exact land-play/entry separation | About 250–400 test lines; entry proposal and destination contracts |

Selected candidate #1 as the dependency-ready unblocking prerequisite requested
by the selection policy. It reuses exact active-effect and casting services;
the other two retain more transaction or identity dependencies. This slice does
not claim a multi-card unlock. A collection scan found City's declaration is
the only compound AnyPlayer rule list in current source. The adapter is shared
across supported player-rule operations and uses no card-name dispatch.
Contract and acceptance cases were stated before editing. Only this slice was
implemented.

### Supported signatures, engine contract and reuse boundary

RON supports `EachPlayerEffect(AnyPlayer, [CantCastSpells(predicate), ...])`
in a permanent's top-level Rules. The unary tuple accepts the bare symbols
SharesANameOriginallyPrintedInArabianNights and
SharesANameOriginallyPrintedInAntiquities. The existing immutable CR 206.3
catalogs match the candidate card's name, independent of physical printing.
The shared `compileOriginalPrintingCardFilter` retains the existing land-filter
behavior and supplies context-specific diagnostics. Inventory classifies the
casting operand as `casting_card_filter`.

All-player rule lists are nonempty and compose already supported casting,
land-play, public visibility, untap-skip, untap-limit and cost-increase operations.
Each operand validates before factory publication. Declaration order and repeated
operations are retained. Empty lists, unsupported mixed operands, wrong arity,
tuple spellings for symbols, wrong scopes, action/cost contexts, arbitrary
filters, permanent references and battlefield rules on instants remain errors.
Each factory makes fresh cards and ability declarations. There is no payment,
player-result or permanent-result binding in this pure static query.

The engine provides `PreventSpellCasts(filter CardFilter) StaticRule` and
`Game.IsSpellCastProhibited(playerID uuid.UUID, card Card) bool`. A zero filter
fails definition validation. Nil and land cards return false. Existing player
and legacy expansion restrictions remain included. The shared checkSpellTiming
precheck covers hand casting, free and alternate zone casting, cast copies and
GetCastableSpells before modes, targets, zone removal or payment. Casting
permission cannot override prohibition. Rejection keeps resources, zone order,
life and the stack intact. Stack copying, pending spell resolution, direct
battlefield entry and land plays retain separate services.

Printed active effects preserve source visits and ability-removal behavior.
Departure, phasing, face-down status and removal stop the rule; tapping and
control changes do not. Immediate phasing needs no layer pass. Return starts a
new visit; overlaps and cloned games keep separate lifetimes. The embedded
catalog is immutable; no mutable filter result or last-action slot is shared.
CR 101.2, 206.3, 400.7, 601.2e, 613.11 and 702.26b–d support the contracts.
No new LKI action, event binding, replacement ordering or simultaneous batch
is introduced. These are legality queries, not resolving effects.

### Failing-first evidence and card-to-test mapping

Before implementation, focused engine and RON tests failed to compile because
PreventSpellCasts and IsSpellCastProhibited did not exist. Implementation followed
that expected failure. Eight focused tests now cover the supported contracts:

- **City in a Bottle** → `TestRONSpellCastRestrictionCity`: exact complete
  player-rule subrule under both active players; matching names on other
  printings; both spell and land prohibition; unrestricted Mountain name;
  failed cast/query state; direct entry; immediate source phasing. A fresh
  complete record still rejects its captured state-member trigger. This is
  subrule gameplay evidence, not complete-card gameplay coverage.
- `TestRONSpellCastRestrictionCompositionAndInventory`: ATQ name catalog,
  catalog separation, compound land/hand/top-library visibility rules,
  semantic operand role and atomic collection rejection.
- `TestRONSpellCastRestrictionListOrderAndInstances`: operation permutations,
  repeated operations, fresh factories, overlapping source lifetimes and
  battlefield-only definition validation.
- `TestRONSpellCastRestrictionInvalid`: invalid shapes, arity, scope, contexts,
  unsupported filters and mixed lists, with no published registry.
- `TestSpellCastRestrictionLifecycle`: both active players, source tap/control,
  face-down/phased/cycle, ability removal/expiry, departure/return, clone
  independence, pure queries and direct entry.
- `TestSpellCastRestrictionPathsAndRollback`: ordinary, free, alternate,
  graveyard and cast-copy paths; unchanged hand/graveyard order, mana, stack
  and life on rejection.
- `TestSpellCastRestrictionStackCopyAndOverlap`: overlapping sources, clone
  isolation, allowed stack copying, pending resolution and later casting.
- `TestSpellCastRestrictionInvalid`: zero engine filter fails validation.

The existing original-printing-name rejection fixture no longer rejects valid
casting operands. Mixed public-visibility, duplicate untap-skip and duplicate
land-rule rejection fixtures now use unsupported operands instead. New positive
composition/order tests establish the supported behavior. No gameplay assertion
was removed to accommodate wrong engine behavior. Test-only fixes used the
existing IsTopCardRevealed API and re-read the original complete record because
staticBaseSubrule changes its shared Rules map; neither changed production rules.

### Changed files and source defects

New engine `pkg/mage/spell_cast_restriction.go` and its test; the shared casting
precheck in `spell_cast_stack.go`. New RON `spell_cast_restriction.go` and its
test; composition/dispatch in `public_visibility.go`, shared catalog adapter in
`land_play_restriction.go`, and semantic role in `inventory.go`. Four existing
RON declaration test files update obsolete rejection cases. API/rules contracts
are documented in `pkg/mage/doc.go`, `docs/engine-rules-contracts.md` and
`docs/dsl-reference.md`. This ledger updates the summary and appends evidence.

No source defect was corrected. The Gauntlets Aura-only defect remains open.
RON source did not change, so no vocabulary snapshot refresh was required.
No Go definitions, DSL exports, generators or runtime startup were removed or
changed. Their cutover/removal gates remain open. No partial registry, fallback,
card-name executor or game-time declaration panic was added.

### Verification and measured collection result

All required checks passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONSpellCastRestriction -count=1`.
- `go test ./pkg/mage -run TestSpellCastRestriction -count=1`.
- `make build` and `go vet ./...`.
- `go test ./pkg/mage/... -short -count=1`.
- `make test`; also `go test ./... -short -count=1` for the engine skill.
- `make lint`: 0 issues. Reviewed final tracked/untracked source and automatic
  formatting; no behavior changes or unrelated file changes were introduced.
  No affected rerun was required.
- `git diff --check`.
- `go run ./cmd/loadcards -json`: expected exit 1, 2,187 compiled / 258 rejected
  records and 846 accepted / 140 rejected names (2,445 / 986 totals).

Final rejection keys and diagnostic messages exactly equal the fresh baseline.
City still reports ARN:13:146 unsupported trigger
WhenAnyNumberOfPermanentsAreOnTheBattlefield. Previously accepted records remain
accepted. The loader rejects the collection and publishes no partial registry.

Estimated full-card unlocks were zero; actual gains are zero. Gameplay evidence
now covers City's complete compound player-rule subrule and reusable engine
casting contracts. Complete-card gameplay coverage is unchanged. There is no
remaining environment or pre-existing validation blocker. This slice is ready
for the authorized `feat(ron): implement filtered casting restrictions` commit.

### Ranked next candidates

1. Captured state-member trigger sets for City in a Bottle. Its casting and
   land-play declarations now work; exact detected plural visits and pending
   trigger suppression must precede ControllersSacrificeEachPermanent. Estimated
   one complete-card unlock if no further contract gap appears.
2. Shared dependent permanent target comparison for Gauntlets of Chaos. Reuse
   shared target legality and atomic exchange; exchange-result handling and the
   recorded Aura-only source correction remain separate. Zero immediate gains.
3. Land entry prevention for Worms of the Earth. Preserve global entry/play
   separation; alternative optional sacrifice/damage payment results remain.
   Zero immediate gains.

Reassess useful shared multi-card work with fresh inventory before selection.
No second slice was started. Complete RON gameplay, runtime loading, native
startup, Go-card deletion and unused DSL cleanup remain unfinished.

## 2026-10-12 — CAPTURED-STATE-PERMANENT-SETS

### Candidate comparison and selection

The worktree was clean. Read the complete RON records for City in a Bottle
(ARN:13), Gauntlets of Chaos (LEG:95) and Worms of the Earth (DRK:118), including
all nested actions, targets and payments. Targeted vocabulary searches confirm
one occurrence of the captured state event; the other two signatures also each
occur on one unique name. Reprints count once.

| Candidate | Distinct affected names | Estimated complete-card gain | Other blockers and downstream benefit | Implementation/test effort |
| --- | --- | --- | --- | --- |
| Captured state-member trigger sets, ranked #1 | City in a Bottle | 1 name / 1 record | Casting, land-play restrictions, source exclusion sets and simultaneous controller sacrifice already exist. Typed detection sets also serve existing collection actions and delayed bodies without a new zone subsystem. | Medium: one engine capture contract, one RON event adapter, definition dominance and lifecycle tests. Dependency-ready. |
| Dependent permanent target card-type comparison | Gauntlets of Chaos | 0 | Comparison would unblock shared target legality, but exchange-result handling and the documented Aura-only source correction remain. | Medium for comparison alone; finishing the card requires a separate result contract and source audit. |
| Global land entry prevention | Worms of the Earth | 0 | Existing entry service can enforce the prohibition, separate from land plays. Eligible-player alternative sacrifice/damage payments and their scoped result remain. | Small/medium for the entry adapter and lifecycle tests; payment procedure exceeds this slice. |

Candidate #1 is dependency-ready and finishes a card with existing downstream
services. The compared alternatives have no immediate gain. There is no
multi-name complete-card unlock among these three bounded candidates. Selected
one cohesive captured state-set capability, including its existing consumer.
Estimated one-name / one-record gain equals the measured gain. No second slice
was started.

### Contract, signatures, contexts and reuse

Engine signature:
`WhenPermanentsPresent(subjects PermanentSet, result *PermanentSetBinding,
body ResolvingEffect) TriggeredAbility`.

It builds on `WhenState`, `PermanentSetBinding`, `FromPermanentSet`, the existing
pending-trigger capture transfer and resolution frames. The engine selects exact
present, unphased visits once when the state is detected and deduplicates them.
The state condition is a definition-time binding producer for its body. Null
sets/results, wrong source zones and unproduced reads fail validation.

RON signature:
`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(subjects), actions)`
in ordinary battlefield rules. Subjects accept context-free permanent filters,
nonempty nested `And`/`Or` and `Other(ThisPermanent)`. The source operand must
be a symbol. Player/target/choice/event-dependent subjects remain rejected.
`Trigger_ThosePermanents` is a plural symbol local to this body and its captured
delayed continuation; it is not a singular reference or a sibling-rule result.
Each factory makes fresh bindings and ability declarations. Inventory marks the
event operand as a permanent filter and the exclusion operand as a permanent
reference. RON owns spelling, arity, subject restrictions and lexical scope;
engine owns detection snapshots, identity and execution.

CR 603.8 suppression uses the existing source-incarnation/ability key: one
pending or stacked trigger prevents another detection. Resolution/countering
permits fresh detection. Later entries cannot join an old set; later type changes
cannot remove detected membership. Current controllers sacrifice saved visits
through `ControllersSacrificeEach`; owner destinations and simultaneous
replacement preparation remain shared. Indestructible and regeneration cannot
stop sacrifice. Returned members cannot inherit old visits. Phasing prevents
action while preserving identity; a phase cycle permits action again.

Source departure, phasing and ability removal stop new detection. Pending bodies
retain their detected source/controller. Return of the same physical source
starts a new incarnation and can trigger while the old body remains pending.
Stack copies, cloned games and delays use the shared independent frames. Waiting
queries do not execute a delayed body. Immutable pending capture slices are
copied into each resolution frame. No global last-action slot is introduced.
CR 113.7a, 400.7, 603.3b/8, 701.21 and 702.26 support these contracts. Local
`data/ARN.json` supplies the exact City Oracle text. No new information read,
payment, damage, targeting or replacement-order subsystem was added; LKI cannot
make an absent member actionable.

### Failing-first evidence and card-to-test mapping

Before implementation:

- `go test ./pkg/mage/dsl/ron -run TestRONStatePermanentSet -count=1`
  failed on the complete City record with unsupported
  `WhenAnyNumberOfPermanentsAreOnTheBattlefield`.
- `go test ./pkg/mage -run TestStatePermanentSet -count=1` failed to compile
  because `WhenPermanentsPresent` did not exist. Implementation followed this
  expected failure. A test-only draw signature error was corrected before
  confirming the missing-API failure.

Complete-card mapping:

- **City in a Bottle** → `TestRONStatePermanentSetCity`: unchanged complete ARN
  record, both active players, captured multi-controller subjects, owner
  destinations, source/token/basic-name exclusion, late entry, repeat detection,
  spell prohibition and land-play prohibition.
- `TestRONStatePermanentSetPending`: same complete record with returned visits,
  phasing/cycles, control changes, source departure/phasing, copies, countering
  and clone independence.
- `TestRONStatePermanentSetInstancesAndInventory`: fresh factories, separate
  source exclusions/suppression, semantic reference roles and both APNAP orders.
  Two Cities can cause a new pending trigger after the first body resolves;
  the test checks the resulting order-dependent departures.
- `TestRONStatePermanentSetInvalid`: malformed arities, empty groups, unsupported
  subject references, wrong event/result shapes, sibling leakage, singular reads,
  wrong source zone and atomic collection rejection.
- `TestRONStatePermanentSetDelay`: saved members through a delayed continuation,
  source departure, later entry, clone isolation and pure waiting queries.

Native contract mapping:

- `TestStatePermanentSetCapture`: detection before placement, pending suppression,
  late entry, exact visits, phasing/cycles, current control, source departure,
  copies, counter/rearm and clone independence.
- `TestStatePermanentSetInvalid`: nil subject/result and instant-source validation.
- `TestStatePermanentSetLifecycleAndReplacement`: empty states, source phasing,
  ability removal, same-card source return, membership after type changes,
  canceled/exiled sacrifices, regeneration/indestructible and final batch state.

Existing City enclosing-rejection assertions now require successful complete
compilation. Worms still must reject. Exact accepted-collection assertions move
from 2,187/258 to 2,188/257. No gameplay assertion was deleted or weakened. One
initial two-City expectation omitted a valid new detection under the second
APNAP order; the final test explicitly checks both outcomes.

### Changed files and source defects

New `pkg/mage/state_permanent_set.go` and native tests; detection integration in
`game.go`, producer validation in `definition.go`, and corrected state-lifecycle
comment in `triggered.go`. New RON adapter and tests in
`pkg/mage/dsl/ron/state_permanent_set*.go`; scope/binding, collection-read, trigger
and inventory integration in `compiler.go`, `player_scope.go`,
`contextual_collection.go`, `triggers.go` and `inventory.go`. Existing sacrifice,
name-filter, casting/land restriction and collection-count tests update their
obsolete support expectations. Established API/contracts are documented in
`pkg/mage/doc.go`, `docs/engine-rules-contracts.md` and `docs/dsl-reference.md`.
This ledger updates the summary and appends the completed evidence.

No RON source defect was corrected. Gauntlets' Aura-only defect remains open.
RON source did not change, so the vocabulary snapshot does not require refresh.
No Go definitions, DSL exports, generators or application startup paths were
removed. Their cutover/removal gates remain open. No fallback, partial production
registry, card-name executor or game-time declaration check was introduced.

### Verification and measured collection result

Fresh clean-tree baseline: focused RON package tests and `make build` passed.
The loader exited 1 with 2,187 compiled / 258 rejected records and 846 accepted /
140 rejected names (2,445 records / 986 names total). An interim read of an old
result file showed 2,180 before the baseline process finished; its final output
matches the prior ledger. There is no baseline regression or environment failure.

Required checks passed in fast-fail order:

- Focused RON `TestRONStatePermanentSet` and native `TestStatePermanentSet`,
  both with `-count=1`.
- `make build` and `go vet ./...`.
- `go test ./pkg/mage/... -short -count=1` and `make test`.
- `go test ./... -short -count=1` for the engine skill.
- `make lint`: final 0 issues. The initial run reported one `wastedassign`
  on `matched := false`; changed it to `var matched bool`, then reran focused
  tests, build, vet, engine short tests and the full suite. Automatic changes
  only sorted imports in new files; reviewed those changes. The same-card source
  return test also passed afresh. Final lint passed after these changes.
- `git diff --check`.
- `go run ./cmd/loadcards -json`: expected exit 1, **2,188 compiled / 257 rejected
  records**, **847 accepted / 139 rejected names** (2,445 / 986 totals).

The only removed rejection is City in a Bottle at ARN:13. Every other rejection
key and diagnostic is unchanged from the fresh baseline. Previously accepted
records remain accepted. The strict loader still rejects the collection and
publishes no partial registry. Compilation coverage gains one name/record;
complete-card gameplay coverage also gains City through the mappings above.
No remaining validation or environment blocker prevents this slice's commit.

### Ranked next candidates

1. Eligible-player optional resolution-payment scope for Scarwood Bandits.
   Inspect `AnyPlayerMayCost(Opponent, PayMana(...))` and its local unpaid branch
   against the existing transaction and control-duration contracts. Expected
   one full-card gain if those remaining forms are supported. Shared eligibility
   and result scope are also prerequisites for Worms' alternative payments and
   Cleansing's per-permanent offers; those cards retain separate blockers.
2. Dependent permanent target card-type comparison for Gauntlets of Chaos.
   Existing target legality and atomic exchange are reusable. Exchange-result
   handling and the documented Aura-only source repair remain separate; zero
   immediate gains from comparison alone.
3. Global land entry prevention for Worms of the Earth. Reuse entry services,
   preserve entry/play separation; eligible-player alternative sacrifice/damage
   payments still block the complete card. Zero immediate gains.

Reassess candidates with fresh complete records and inventory before selecting.
Complete RON gameplay, runtime loading, native startup, Go-card deletion and
unused DSL cleanup remain unfinished. This run completed exactly one slice.

## 2026-10-12 — ELIGIBLE-PLAYER-FIXED-MANA-PAYMENT

### Candidate comparison and selection

The worktree was clean. Fresh RON package tests and build passed. Strict loader
baseline: 2,188 compiled / 257 rejected records, 847 accepted / 139 rejected
names, with 2,445 records / 986 names total. Its exit 1 was collection rejection,
not a build/test failure. Complete unchanged records were read for Scarwood
Bandits, Worms of the Earth, Cleansing and Gauntlets of Chaos. Targeted vocabulary
inspection found two `AnyPlayerMayCost` occurrences, one for each of Scarwood
and Worms. Reprints count once.

| Candidate | Distinct affected names | Estimated full-card gains | Remaining blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Eligible-player optional payment | Scarwood Bandits; Worms of the Earth; Cleansing as a composed consumer | 0 for this prerequisite; 1 for Scarwood after a separate duration slice | Shared player eligibility, complete payment and local branch service supports all three names. Scarwood needs its duration; Worms needs land entry prevention plus alternative sacrifice/damage and payer binding; Cleansing needs per-permanent offers/results | Small/medium: reuse the existing transaction, add player-set orchestration, lowering and contract tests |
| Dependent target card-type comparison | Gauntlets of Chaos | 0 | Exchange-result guard and the recorded Aura-only source defect remain. Shared target legality can serve future dependent target groups | Medium: announcement/resolution predicates and copy identity tests |
| Global land entry prevention | Worms of the Earth | 0 | Alternative eligible-player payments remain. Shared entry prevention would keep play/entry separation and replacement ordering | Medium: entry preflight, source lifecycle and batch tests |

Candidate #1 is dependency-ready as a shared prerequisite with concrete
three-card downstream use. The ledger's suggested one-card gain was corrected
after inspecting the nested duration, not inferred from first-error frequency.
`compileSourceControl` does not accept `UntilPermanentLeavesBattlefield`, and
the existing `UntilSourceLeaves` engine service deliberately survives phasing.
`data/DRK.json` gives Scarwood's duration as "for as long as this creature remains
on the battlefield". CR 611.2b and 702.26f require a distinct presence duration.
Adding that contract here would be a second slice. This run completes only
eligible-player fixed-mana payment. Estimated zero full-card gains matches the
actual result. No complete-card gameplay gain is claimed.

### Signatures, contexts and reuse boundary

- Engine: `PayAnyOrElse(players PlayerSet, costs CostGroup, paid, unpaid
  ResolvingEffect) ResolvingEffect`. At least one body must be supplied. An
  explicit valid cost group and player set are required. Current player IDs
  are read once, normalized to turn order and offered at most once. Absent
  and duplicate IDs cannot supply extra offers. Empty eligibility runs unpaid.
- RON: `AnyPlayerMayCost(Opponent|AnyPlayer, PayMana([...]))` followed immediately
  by `If(condition, [actions])`, `Unless(condition, [actions])` or
  `IfElse(condition, [actions], [actions])`. Condition is symbolic `CostWasPaid`
  or `CostWasntPaid`; each required body is nonempty. Ordinary spell, activated,
  triggered and delayed sequences use the same lowering. X, other cost forms,
  other player sets, entry and draw replacement contexts remain rejected.
- The shape exports no payer binding. `ActionPlayer` cannot inherit an unrelated
  enclosing player-action scope inside the pair. Other saved references keep
  their existing contexts. Nested payments cannot overwrite the outer result.
- `ron` owns syntax, adjacency, polarity, context and inventory roles. Each
  factory builds fresh costs and branch effects against its instance bindings.
  The engine owns eligibility and payment. Both bodies retain the original
  resolving controller, source, targets and saved visits.
- Private `tryResolutionCostPayment` is extracted from `unlessPaidEffect.Apply`.
  Existing optional and mandatory callers use the same helper. Affordability
  remains pure; accepted payments use `prepareActionPaymentTransaction` and
  commit atomically. No independent solver, global payment slot, replacement
  order or simultaneous movement service is introduced.
- Native service supports existing resolution cost groups, including life,
  with their existing validation. Payer-bound action alternatives and repeated
  per-permanent result orchestration remain separate adapters/contracts.
- Typed player-set and both body source declarations participate in the shared
  waiting query without resolving eligibility, paying or choosing. Saved source
  identity, LKI, target legality, phasing, copy and clone frames are reused.

### Failing-first evidence and card-to-test mappings

Before implementation, `go test ./pkg/mage -run TestEligiblePlayerPayment -count=1`
failed to compile because `PayAnyOrElse` did not exist. The focused RON command
failed gameplay tests with unsupported `AnyPlayerMayCost`, including the actual
Scarwood payment node. Implementation followed these expected failures.

Native coverage:

- `TestEligiblePlayerPaymentOrderAndResult`: both active-player orders, first
  and second acceptance, stop after payment, decline, insufficient resources,
  combined-cost rollback, opponent-only eligibility, empty set and zero costs.
- `TestEligiblePlayerPaymentRecheckAndNested`: an accepted offer becomes
  unpayable before transaction preparation, later player pays, and an inner
  failed payment cannot change the outer result.
- `TestEligiblePlayerPaymentDelayedIdentity`: live, returned, phased and
  control-changed source information, exact waiting references and query purity,
  delayed execution, clone isolation and simulation-player decline policy.
- `TestEligiblePlayerPaymentInvalid`: nil declarations/costs/bodies and saved
  results leaking between branches or to a sibling instruction.

RON coverage:

- Scarwood Bandits → `TestRONEligiblePlayerPaymentRealRecords`: read and reject
  its complete unchanged record, extract its exact payment AST node, lower it
  with a synthetic observable continuation and execute through the engine.
  This proves the payment's opponent eligibility and decline behavior, not the
  unsupported control duration or complete card. Worms' unchanged complete
  record also remains rejected. Cleansing remains a downstream mapping only.
- `TestRONEligiblePlayerPaymentGameplay`: eligible sets, success/decline/lack
  of resources, resolution-only timing and local nested outcomes.
- `TestRONEligiblePlayerPaymentPolarityAndContexts`: both symbolic conditions,
  If/Unless/IfElse, accepted/declined offers, ordinary spells, upkeep triggers
  and delayed continuations after source departure.
- `TestRONEligiblePlayerPaymentTargetsCopiesAndFactories`: fresh factories,
  independent target declarations, departure/return/control/phasing of sources,
  returned/phased/phase-cycled/shrouded targets, stack copies, game clones and
  automatic payment from actual RON Forests. Simulation clones decline optional
  payment, so their unpaid result is checked separately from a paying original.
- `TestRONEligiblePlayerPaymentInvalid`: malformed arities, symbols and costs,
  missing/empty/detached continuations, unbound payer reads, unsupported execution
  contexts and inherited payer-scope rejection.
- `TestRONEligiblePlayerPaymentInventoryAndAtomicLoading`: player-collection
  and resolution-cost roles, supported isolated registry and strict rejection
  without partial publication when another record is invalid.

### Changed files and source defects

New native `eligible_player_payment.go` and tests; shared attempt helper in
`composable_costs.go`; new RON adapter and tests with sequence integration in
`optional_payment.go` and roles in `inventory.go`. The existing landwalk test's
Scarwood blocker advances to `UntilPermanentLeavesBattlefield`; it still
requires complete-record rejection. API/contracts are documented in `doc.go`,
`docs/dsl-reference.md` and `docs/engine-rules-contracts.md`. This ledger updates
the current summary and appends this entry.

No RON source defect was corrected. Gauntlets' Aura-only defect remains open.
Scarwood's duration is an engine/adapter prerequisite, not a data defect.
RON sources did not change, so the vocabulary snapshot needs no refresh.
No Go card definitions, DSL exports, generators or runtime wiring were removed;
their gates remain open. No fallback, partial production registry, card-name
branch or game-time declaration panic was added.

### Verification and measured collection result

Focused RON and native tests, RON package tests, `make build`, `go vet ./...`,
`go test ./pkg/mage/... -short -count=1`, `make test`, and the skill's
`go test ./... -short -count=1` passed. `make lint` passed with 0 issues.
Automatic changes only reordered imports in the two new test files; reviewed
them and reran both focused test groups and `make build`, which passed.
`git diff --check` passed.

Final `go run ./cmd/loadcards -json` exited 1 for expected collection rejection:
**2,188 compiled / 257 rejected records**, **847 accepted / 139 rejected names**
(2,445 records / 986 names total). There are zero added or removed rejection
keys. The only changed diagnostic is Scarwood Bandits at DRK:91, advancing from
unsupported `AnyPlayerMayCost` at column 351 to unsupported source control
duration `UntilPermanentLeavesBattlefield` at column 480. Every other rejection
diagnostic is unchanged, and all previously accepted records remain accepted.
The loader still publishes no partial registry. Compilation coverage and
complete-card gameplay coverage both gain zero names/records; the fixed-mana
payment capability now has native and RON gameplay contracts. There is no
environment or validation blocker to committing this slice.

### Ranked next candidates

1. Eligible-player alternative payments and typed payer binding. Worms' two
   sacrifice-or-damage alternatives require payer identity and local success;
   Cleansing can reuse eligibility with fixed life and repeated per-subject
   results. Entry prevention and repeated result orchestration remain separate.
2. Source-presence control duration for Scarwood Bandits: expected one full-card
   gain after the payment prerequisite. Test phase-out and phase cycles before
   and after resolution, departure/return, source control and ability removal.
   Evaluate shared duration use in Gaea's Liege's separate layer context.
3. Dependent target card-type comparison for Gauntlets of Chaos: zero immediate
   gains; exchange-result guard and Aura-only source repair remain separate.

Reassess with complete records and fresh evidence. Complete collection gameplay,
runtime loading, application cutover, Go-card deletion and unused DSL cleanup
remain unfinished. This run completed exactly one payment capability slice.

### Completion validation — 2026-10-13

Resumed the existing fixed-mana payment slice and reviewed its engine service,
RON lowering, tests, complete affected records and documentation. No second
capability was started. Fresh focused native and RON tests, `make build`,
`go vet ./...`, `go test ./pkg/mage/... -short -count=1`, `make test`, and
`go test ./... -short -count=1` all passed. `make lint` passed with 0 issues;
it made no further changes. `git diff --check` passed.

The initial sandboxed build reported a denied Go module-cache write. The
approved build outside the sandbox and vet both passed. This was an environment
restriction, not an engine or compiler failure.

Fresh strict-loader validation exited 1 for expected collection rejection:
2,188 compiled / 257 rejected records and 847 accepted / 139 rejected names.
A temporary Go overlay of the HEAD compiler sequence and inventory files
provided a fresh pre-slice loader comparison without changing the worktree.
Both reports have identical counts and rejection keys. Only Scarwood Bandits'
diagnostic changed, from `AnyPlayerMayCost` to `UntilPermanentLeavesBattlefield`.
The estimated and actual full-card gain remains zero. The ranked next candidates
above remain applicable; eligible-player alternatives and payer bindings are
next. All required checks passed before committing this slice.

## SOURCE-PRESENCE-CONTROL-DURATION — 2026-10-13

### Baseline, candidates and selection

The working tree was clean. Fresh `make build` and focused existing eligible-player
payment/source-control tests passed. Baseline `go run ./cmd/loadcards -json`
exited 1 for collection rejection, not build failure: 2,188 compiled / 257
rejected records, 847 accepted / 139 rejected names. Complete records and nested
forms were read for Worms of the Earth, Cleansing, Scarwood Bandits, Gauntlets
of Chaos, all six Gaea's Liege reprints, Oubliette and Quarum Trench Gnomes.
The targeted vocabulary entries were inspected without loading the full snapshot.

| Compared candidate | Distinct affected names | Estimated complete-card gains | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Eligible-player alternatives and payer bindings | Worms of the Earth; Cleansing | 0 | Worms needs land-entry prevention as well as alternative sacrifice/action payments with local payer identity. Cleansing needs repeated per-land offers and typed unpaid-subject results. Shared eligibility is ready; the combined alternative/payer contract is not ready as one small adapter. Enables both payment consumers. | Large: choice, transaction/action-payment composition, binding scope and repeated outcomes need separate contracts. |
| Source-presence control duration | Scarwood Bandits | 1 | Fixed mana payment, typed artifact targets and forestwalk are ready. Supplies a reusable captured lifetime for control and characteristic modifications. Gaea's Liege and Oubliette use different lifetimes; Quarum already has its subject-bound mana replacement adapter. | Medium: capture/start checks, permanent expiration on phasing, control reconciliation, identity/copy/clone tests. |
| Dependent permanent target type comparison | Gauntlets of Chaos | 0 | Prior target identity must constrain announcement and resolution legality. Exchange-result guard and the documented Aura-only source defect remain separate. Enables the dependent target before those later prerequisites. | Medium: dependent target binding and legality; later guard/data work still required. |

Candidate #1 was evaluated first. Its combined alternative and payer shape needs
separate contracts before it is dependency-ready. Candidate #2 has the largest
credible complete-card gain for this run's effort. The initial comparison of
all `UntilPermanentLeavesBattlefield` spellings was narrowed after Oracle and
role checks: only Scarwood is a new consumer of this control lifetime. Matching
spelling does not make the layer, phasing and replacement lifetimes equivalent.
No multi-card unlock is claimed for those separate roles.

The selected contract and acceptance cases were stated before implementation:
read the exact source at resolution, capture subjects/values/controller, reject
invalid declarations, prevent absent/phased/returned sources from starting the
duration, permanently end it on source departure/phasing, preserve it through
source control changes/ability removal, retain target identity through phasing,
and keep stack copies, delayed frames and cloned games separate.

### Exact signatures, contexts and reuse boundary

New engine API:

`UntilSourcePresent(_ sourceRef, modifications ...ContinuousModification) ResolvingEffect`

The source marker is `mage.This`. A declaration needs a permanent source and
at least one valid modification. This duration accepts control modifications;
other duration validation stays unchanged. It uses `untilEffect` and
`composedLayerEffect` for capture, layers and clone state. The exact source
incarnation must be present and phased in at resolution. A completed phase
cycle before resolution can start the duration. Later source phasing latches
expiration for all its installed layers in `Game.PhaseOut`, including indirect
phasing groups, before an immediate phase-in can occur. Source departure uses
the existing removal service. Untapping, control changes, ability removal and
cleanup do not end it. `UntilSourceLeaves` still survives source phasing.

RON supports exactly:

`GainControlOfPermanentUntil(subject, UntilPermanentLeavesBattlefield(ThisPermanent))`

in ordinary resolving permanent bodies, including activated actions, triggers,
local payment branches and delayed bodies. `subject` uses the existing typed
singular permanent references. The adapter calls `UntilSourcePresent(This,
ChangeController(subject, You))`; RON owns spelling, arity, endpoint and context
checks, while the engine owns duration state and rules. Other endpoint references,
malformed symbols/tuples, spell sources, static, entry, cost, draw replacement
and layer-duration contexts gain no support. No state or choices are captured
by the compiler factory. Waiting delayed-source queries inherit the existing
explicit source-read declaration from the source-bound duration.

Rules evidence: CR 611.2b–d, 702.26d–f and 400.7. `data/DRK.json` states
Scarwood's "for as long as this creature remains on the battlefield." The
shared captured subject/value services, layer 2 timestamp reconciliation,
control-change prohibition, target resolution legality, cost transactions,
opponent offers, delayed frames and clone services are reused. This slice
adds no damage, replacement-order, APNAP or simultaneous movement service.

### Failing-first evidence and card-to-test mapping

Before implementation, `go test ./pkg/mage -run TestUntilSourcePresent -count=1`
failed to compile because the new API did not exist. The new focused RON tests
failed on unsupported `UntilPermanentLeavesBattlefield` in the actual Scarwood
record. Implementation followed those expected failures. The initial broader
layer adapter was removed after the role/Oracle check; the final boundary test
requires that layer shape to remain rejected.

Native contracts:

- `TestUntilSourcePresentLifetime`: control/P/T/keyword layers, captured members,
  source direct/indirect phasing/cycles/departure/return, control/untap/ability removal/cleanup,
  subject phasing/return, expiration while the subject is phased, and clone isolation.
- `TestUntilSourcePresentStartAndValidation`: live, absent, returned and phased
  source starts, pre-resolution phase cycle, permanent-source validation,
  empty/nil modifications and valid source-presence control declarations.
- `TestUntilSourcePresentControlOrder`: later layer wins; expiration reveals an
  earlier layer or base control; control-change prohibition remains enforced.

RON contracts and full-card mapping:

- Scarwood Bandits → `TestRONSourcePresenceDurationScarwood`: the complete
  unchanged DRK record supplies forestwalk, activation costs/rollback, artifact
  target legality, opponent paid/declined/unaffordable branches, announcement
  versus resolution timing, source/target departures and returns, phase cycles
  before/after resolution, source control/ability changes, shroud, independent
  instances, stack copies and pending game clones.
- Scarwood Bandits → `TestRONSourcePresenceDurationCastingAndBoundary`:
  complete casting cost, 2/2 and forestwalk, fresh factory target declarations,
  invalid entry/replacement/spell-source contexts and layer-role rejection.
- `TestRONSourcePresenceDurationContexts`: upkeep and delayed control bodies
  retain their resolving controller after source control changes; repeated
  waiting-source reads preserve exact identity and do not execute the body.
- `TestRONSourcePresenceDurationInvalidAndLoading`: invalid arities/endpoints,
  complete Scarwood acceptance, continued Gaea/Oubliette rejection and atomic
  loading with a valid isolated registry and an invalid additional record.

Existing payment-node tests remain in force. Their old enclosing Scarwood
rejection assertion is removed because the complete record now passes.
`TestRONTextLandwalkRecords` gains Scarwood (12 names / 22 records / 1 blocked).
Collection plan/inventory/numeric assertions advance by exactly one record;
no rejection checks or engine behavior assertions were weakened.

### Source defects, changed files and gates

No RON source changed. No vocabulary refresh is needed. Gaea's Liege retains
its known duplicate `IsntAttacking` branch defect; local `data/LEA.json` confirms
that its second branch must describe attacking. Its activated layer says
"until this creature leaves the battlefield" and needs a distinct adapter that
preserves phasing. Oubliette's local `data/ARN.json` says "phases out until this
enchantment leaves" and needs its separate phasing lifecycle. Quarum's existing
subject-bound replacement lifetime is already supported and is unchanged.
Gauntlets' Aura-only defect remains open. No source-data correction was made.

Changed engine files: `composable_continuous.go`, `effect_manager.go`, `game.go`,
`doc.go`, and new `source_presence_duration_test.go`. Changed RON files:
`source_control.go`, new `source_presence_duration_test.go`, payment/landwalk
regressions and collection plan/inventory/numeric count assertions. Established
contracts are documented in `docs/engine-rules-contracts.md` and
`docs/dsl-reference.md`; this ledger updates the current status and appends the
entry. No Go cards, DSL functions, generators or runtime wiring were removed.
Their cutover/removal gates remain open. No fallback, partial registry,
card-name branch or game-time declaration panic was added.


### Verification and measured collection result

Final checks passed in the required order:

- `go test ./pkg/mage/dsl/ron -run TestRONSourcePresenceDuration -count=1`;
  focused native `go test ./pkg/mage -run TestUntilSourcePresent -count=1`;
  focused payment/landwalk and collection regression checks also passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1`, `make test` and the engine skill's
  `go test ./... -short -count=1` passed. These checks were repeated after the
  final indirect-phasing test was added and passed again.
- `make lint` passed twice with 0 issues. The only automatic change was import
  order in the new native test file; it was reviewed, and focused tests,
  build/vet and repository suites were rerun. The final lint made no further
  changes. `git diff --check` passed.
- `go run ./cmd/loadcards -json` exited 1 for expected collection rejection:
  **2,189 compiled / 256 rejected records**, **848 accepted / 138 rejected
  names** (2,445 records / 986 names total).

The first engine-short run found four old collection-count assertions expecting
2,188 / 257. Those exact expectations were updated to 2,189 / 256; all behavior
and rejection checks remain. There was no environmental or pre-existing build
failure. No independent required check remains blocked.

The fresh loader comparison removes exactly one rejection key: Scarwood Bandits
at DRK:91, previously unsupported `UntilPermanentLeavesBattlefield` at column
480. No new rejection key appears; every remaining rejection diagnostic is
unchanged. All previously accepted records remain accepted. Strict collection
loading still publishes no partial registry.

Estimated and actual gains match: **one unique card and one record**. Scarwood
also gains complete-card RON gameplay evidence, separately from compilation
coverage. The shared engine lifetime has additional synthetic control/P/T/ability
contracts; these are not counted as inventory card unlocks.

### Ranked next candidates

1. Eligible-player alternative payments and typed payer scope for Worms of the
   Earth and Cleansing. Split out a small shared prerequisite; Worms retains
   land-entry prevention and Cleansing retains per-subject result orchestration.
2. Dependent target card-type comparison for Gauntlets of Chaos. Zero immediate
   gains; exchange-result guard and Aura-only source correction remain separate.
3. Source-leave layer duration for Gaea's Liege, preserving phasing. Defending-
   player characteristics and the known duplicate predicate defect also remain.

Reassess with complete records and fresh evidence. Complete collection gameplay,
runtime loading, application cutover, Go-card deletion and unused DSL cleanup
remain unfinished. This run completes exactly one duration capability slice.


## ELIGIBLE-PLAYER-PAYER-SCOPE — 2026-10-13

### Baseline, candidates and selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Fresh strict loader execution
exited 1 for collection rejection: 2,189 compiled / 256 rejected records,
848 accepted / 138 rejected names. This was not a build or environment failure.
Only the progress status and latest entry were read. Complete records and nested
forms were inspected for Worms of the Earth, Cleansing, Scarwood Bandits,
Gauntlets of Chaos and all six Gaea's Liege reprints. Targeted vocabulary keys
included AnyPlayerMayCost, ActionPlayer, ForEachPermanentAnyPlayerMayCost,
CostWasntPaidForIt, Or, the dependent target comparison and source-leave duration.
Local DRK/LEG/LEA Oracle data confirmed the relevant payment and duration wording.

| Compared candidate | Distinct affected names | Estimated full-card unlocks | Remaining blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Eligible-player payment prerequisite: typed payer scope | Worms of the Earth; Cleansing; existing Scarwood Bandits regression | 0 | Worms needs sacrifice-or-damage action costs plus land-entry prevention. Cleansing needs repeated per-land life offers and saved unpaid subjects. A candidate binding supplies Worms' ActionPlayer during its future action-payment adapter; the shared offer service remains available to Cleansing's repeated offers. Fixed mana and paid-path scope can be completed independently. | Medium: binding lifetime, definition path validation, lexical shadowing, transactions, delays/copies/clones. |
| Dependent permanent target type comparison | Gauntlets of Chaos | 0 | Prior target type constrains announcement and resolution. Exchange-result guard and the Aura-only source correction remain. Enables a dependent target consumer. | Medium: target context and legality adapters, current types and saved visits. |
| Source-leave layer duration | Gaea's Liege | 0 | Defending-player characteristics and the duplicate IsntAttacking defect remain. Enables its activated Forest layer while retaining phasing. | Medium: existing source-leave duration reuse, layer adapter and lifetime regressions. |

Candidate #1 is dependency-ready when limited to payer scope. This run selects
that complete prerequisite instead of combining alternative costs, repeated
subjects and entry prevention into one oversized slice. Immediate gains were
estimated at zero. The multi-card downstream path is explicit: candidate-bound
action payments for Worms and shared eligible-player offers for Cleansing.
The other compared candidates also have zero immediate gains and independent
remaining blockers. No second capability was started.

### Supported shapes, signatures and reuse boundary

New native API:

`PayAnyOrElseWithPayer(players PlayerSet, costs CostGroup, payer *PlayerBinding,
paid, unpaid ResolvingEffect) ResolvingEffect`.

Each eligible candidate is bound before affordability, decision and payment.
A failed payment removes that candidate before the next attempt. The paid body
can read the successful payer through FromPlayer; the unpaid body has no payer.
Exit removes the local value and restores a previous value if present. Definition
validation requires a named, nonnil, separate binding and permits its reads only
in the costs and paid body. Reads in eligibility, unpaid bodies or later actions
remain invalid. Branch-produced results retain their existing isolation.

RON retains `AnyPlayerMayCost(Opponent|AnyPlayer, PayMana([...]))` in ordinary
spell, activation, trigger and delayed sequences. An adjacent If/Unless/IfElse
still requires symbolic CostWasPaid/CostWasntPaid and nonempty action lists.
ActionPlayer now reads the successful payer only in If(CostWasPaid),
Unless(CostWasntPaid), or the corresponding paid IfElse branch. It shadows an
enclosing action player in that branch; the unpaid branch has no ActionPlayer.
The enclosing scope returns after the pair. Implicit player actions keep their
prior player context and You keeps the effect controller. An explicit
PlayerAction(ActionPlayer, ...) selects the payer. Nested pairs have separate
plans; each factory builds a fresh PlayerBinding and cloned declaration map.

The engine owns payer lifetime, definition path validation and execution.
RON owns branch polarity, spelling, lexical scope and fresh lowering. Reused
services: PayAnyOrElse eligibility/turn order, tryResolutionCostPayment,
shared full-cost preparation/commit and rollback, PlayerBinding/FromPlayer,
resolution frames, AtNext captured maps, target legality, saved source visits,
waiting-source enumeration, stack copies and game cloning. The old PayAnyOrElse
signature remains unchanged. No parallel payment system, global last-payer slot,
card-name branch, production fallback or partial registry is introduced.

Alternatives, action-as-payment operations, nonmana/X RON offers, repeated
per-subject results, entry and replacement bodies remain separate unsupported
shapes. The native candidate binding is ready for their later adapters; this
slice does not claim those adapters or full-card gameplay for Worms/Cleansing.

### Failing-first evidence and gameplay mapping

Before implementation, `go test ./pkg/mage -run TestEligiblePayerScope -count=1`
failed to compile solely for the missing PayAnyOrElseWithPayer API after a test
field-name correction. The focused RON tests failed with unbound ActionPlayer
in paid branches. These expected failures preceded implementation.

Native contracts:

- TestEligiblePayerScopeOffersAndRollback: each offer reads its candidate,
  complete/declined/unaffordable payments, post-choice resource recheck,
  continued offers and local value removal.
- TestEligiblePayerScopeCostBinding: typed payer reads during cost checks and
  staged payment, failed partial life payment rollback, later successful payer,
  and rejection of the same cost without a producer.
- TestEligiblePayerScopeNestedAndDelayed: separate nested payers, original
  controller, delayed capture after scope exit and source departure, waiting
  query purity, and cloned delayed-frame isolation.
- TestEligiblePayerScopeInvalid: nil/unnamed/reused binding, unpaid reads and
  escaped reads. Existing TestEligiblePlayerPayment* keeps old API order,
  affordability, empty-set, zero-cost, lifecycle and validation contracts.

RON contracts and card-to-test mapping:

- TestRONEligiblePayerScopeGameplay: paid/unpaid identities, nested failed offer,
  source departure/return/phasing/control, stack copies and pending clones.
- TestRONEligiblePayerScopePolarityAndContexts: both paid polarities and IfElse
  layouts in spells, upkeep triggers and delays, including delay creation on a
  paid path, source departure and cloned saved payer.
- TestRONEligiblePayerScopeOrderAndLoading: active-player choice on each
  resolution, separate factory instances, valid isolated loading and atomic
  rejection when an invalid unpaid payer record is added.
- TestRONEligiblePayerScopeInvalid: unpaid and escaped lexical reads remain
  compile errors. Existing rejection tests now accept a paid ActionPlayer read;
  this is the exact declaration shape established by this slice.
- Scarwood Bandits → existing TestRONEligiblePlayerPaymentRealRecords and
  TestRONSourcePresenceDurationScarwood/CastingAndBoundary/Contexts/InvalidAndLoading.
  Its unchanged complete DRK record continues to pass full gameplay regressions.
- Worms of the Earth → TestRONEligiblePlayerPaymentRealRecords still rejects
  its enclosing record. Its candidate-bound damage action is a downstream
  consumer, not newly verified gameplay. Cleansing remains rejected at its
  repeated-offer producer. Neither is counted as behavior-complete.

### Source defects, changed files and gates

No RON source changed; no vocabulary refresh is required. The existing
Gauntlets of Chaos Aura-only omission and Gaea's Liege duplicate IsntAttacking
branch remain open, consistent with local Oracle data. No new source defect
was found. No Go cards, DSL APIs, generators or runtime wiring were removed;
the collection/cutover/removal gates remain open.

Changed engine files: eligible_player_payment.go, definition.go, doc.go and new
eligible_payer_scope_test.go. Changed compiler/tests: eligible_player_payment.go,
eligible_player_payment_test.go and new eligible_payer_scope_test.go under
pkg/mage/dsl/ron. Updated docs: engine-rules-contracts.md, dsl-reference.md and
this progress ledger.

### Verification and measured collection result

All required checks passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONEligiblePayerScope -count=1` passed.
  Focused native TestEligiblePayerScope and existing eligible-player/Scarwood
  regressions also passed with -count=1.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
  The engine skill's `go test ./... -short -count=1` also passed.
- `make lint` passed with 0 issues. Automatic-fix output and final changes were
  reviewed; no further source changes or unrelated file changes were made.
  `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited 1 for expected strict collection
  rejection: **2,189 compiled / 256 rejected records**, **848 accepted / 138
  rejected names** (2,445 records / 986 names). The final JSON report is exactly
  equal to the fresh baseline report, including all rejection diagnostics.
  No accepted record regressed and no partial registry was published.

Estimated and actual gains match: zero newly accepted unique cards or records.
The new native/RON payer contracts add gameplay coverage for a shared primitive;
Scarwood's existing complete-card gameplay remains covered. Worms and Cleansing
are still unsupported complete records. Collection compilation and complete-card
behavior counts were not increased for synthetic tests. No environment failure,
pre-existing build/test failure or independent required check remains blocked.

### Ranked next candidates

1. Repeated eligible-player life offers and saved unpaid permanent sets for
   Cleansing. The shared offer service, native life cost and exact set bindings
   are ready; complete the producer/consumer and batch destruction together.
   Estimated full-card gain: one unique card / one record.
2. Eligible-player alternative/action payment adapters for Worms of the Earth.
   Candidate payer scope is ready; distinguish accepted damage instructions from
   damage actually dealt. Land-entry prevention remains a separate blocker.
   Estimated immediate gain: zero.
3. Dependent target type comparison for Gauntlets of Chaos. Exchange-result
   guard and Aura-only correction remain; estimated immediate gain: zero.

Complete collection gameplay, runtime loading, Go-card deletion and unused DSL
cleanup remain unfinished. This run implements exactly one payer-scope slice.

## 2026-10-13 — PER-PERMANENT-LIFE-PAYMENT

### Fresh baseline and candidate comparison

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Fresh strict loader execution
exited 1 for collection rejection: 2,189 compiled / 256 rejected records,
848 accepted / 138 rejected names. This was an unsupported collection,
not a build or environment failure.

Complete source records inspected: Cleansing at DRK.lines.ron:19,
Worms of the Earth at DRK.lines.ron:118, and Gauntlets of Chaos at
LEG.lines.ron:95. Targeted vocabulary entries and nested actions were checked.
Names count reprints once. Local DRK/LEG Oracle data confirmed the requirements.

| Candidate | Affected distinct names | Estimated full-card gain | Other blockers | Downstream capability | Effort |
| --- | --- | --- | --- | --- | --- |
| Per-permanent life offers and unpaid sets | Cleansing | 1 name / 1 record | None | Repeated eligible-player offers and exact unpaid collections for later consumers | Medium: engine producer, lexical lowering, lifecycle and batch tests |
| Alternative/action eligible-player payments | Worms of the Earth | 0 | Land-entry prohibition; sacrifice-or-damage acceptance adapters | General eligible-player alternatives using saved payer scope | Larger: action acceptance versus outcome, alternatives and rollback |
| Dependent target type comparison | Gauntlets of Chaos | 0 | Exchange-result guard; Aura-only RON defect | Cross-target type comparisons for target legality | Medium: dependent target reads, legality and copy contracts |

Candidate 1 was dependency-ready and completes the latest ledger's first-ranked
slice. It provides an actual full-card unlock rather than stopping at another
intermediate adapter. The other two candidates retain independent blockers.
The reusable service has no card-name policy. This run implements one capability
and stops; no runtime cutover or removal gate is satisfied.

### Exact supported signatures and ownership

Native API:

`OfferCostForEachPermanent(subjects PermanentSet, players PlayerSet,
costs CostGroup, unpaid *PermanentSetBinding) ResolvingEffect`

The producer snapshots distinct present, unphased visits and player eligibility
at resolution. Subject controller groups use turn order; the effect controller
chooses relative order within each group (CR 608.2f). Each offer identifies its
subject. For each visit the shared eligible-player loop offers the full cost in
active-player order, stopping at the first complete payment. Failure, decline
and insufficient resources continue to later players. Separate subjects require
separate payments. Zero costs require acceptance. The result saves unpaid visits
and replaces older values even when empty. All offers precede later actions.

RON signatures:

- `ForEachPermanentAnyPlayerMayCost(collection, AnyPlayer, PayLife(Integer(n)))`
  in ordinary resolving action sequences, with fixed nonnegative `n`.
- Symbolic `CostWasntPaidForIt` as a plural contextual collection read from the
  latest preceding producer. Nested branches shadow it locally and cannot export
  their producer. Following outer reads keep the outer producer.
- Ordinary spells, permanent activations, triggers and delayed bodies share the
  lowering. Delay creation can save an already produced set. Entry and draw
  replacement contexts, other player sets, other payment forms, dynamic amounts,
  malformed arity and unbound results remain rejected.

The engine owns visit capture, order decisions, payments and result frames.
RON owns spelling, declaration diagnostics, lexical plans and fresh bindings.
Dependencies reused: controlledPermanentGroups, EachPlayer turn order,
PayAnyOrElse's extracted offer loop, tryResolutionCostPayment's shared transaction
and affordability checks, LifePayCost, PermanentSetBinding/FromPermanentSet,
resolution capture, stack copies, game cloning, AtNext and DestroyEach.
The subject-aware prompt uses the same payment transaction. Old payment APIs and
prompts remain compatible. There is no global result slot, Go-card fallback,
partial registry, replacement shortcut or parallel destruction implementation.

### Failing-first evidence and gameplay mapping

Before implementation, `go test ./pkg/mage -run TestPermanentPayment -count=1`
failed to compile for missing OfferCostForEachPermanent. The focused RON suite
failed at the unsupported real Cleansing producer and isolated loading case.
The inventory-role test separately failed before its adapter was added.
These expected failures preceded their production changes.

Native evidence:

- TestPermanentPaymentOffersOrderAndRollback: both active players, effect-controller
  subject ordering, payments for another controller's land, duplicate collections,
  stop-at-success, separate charges, full-cost rollback and empty-result reset.
- TestPermanentPaymentIdentityDelayAndClone: initial phasing exclusion, saved visit
  phasing and phase cycle, delayed capture, departure/return and cloned-frame isolation.
- TestPermanentPaymentSnapshotRecheckAndSimultaneous: offer-time resource change,
  failed transaction followed by a later payer, subjects that leave and return,
  later entry exclusion, one destruction batch and simultaneous departure LKI.
- TestPermanentPaymentInvalid: missing subjects/players/costs/result, nil costs,
  empty binding name, unproduced set and valid sequential producer/consumer.
- Existing TestEligiblePlayerPayment* and TestEligiblePayerScope* retain coverage
  for the shared loop, payer bindings, affordability, local branches and delays.

RON evidence and full-card mapping:

- **Cleansing → TestRONPermanentPaymentCleansing** uses its complete unchanged DRK
  record. It covers failed casting costs, resolution-only offers, no payment,
  controller/opponent payments, per-land life affordability, phased lands,
  regeneration, indestructibility, unaffected nonlands, stack copies and clones.
- TestRONPermanentPaymentContextsAndScope covers upkeep triggers, activation,
  live delayed collection versus delayed saved results, source departure,
  waiting-query purity, cloned pending delays and nested producer isolation.
- TestRONPermanentPaymentInvalidAndLoading checks invalid player/payment forms,
  negative/dynamic life, malformed result symbol, escaped branch result,
  unbound result and atomic collection rejection.
- TestRONPermanentPaymentInventory checks resolution-cost semantic classification.
- Repository inventory/definition/numeric tests keep exact new collection totals
  and explicitly reject a regression that makes Cleansing unsupported again.

The first required engine short run exposed four stale collection-count
assertions expecting 2,189/256. They were updated to the measured 2,190/255;
no semantic assertion was removed or weakened. A late native test used an
incorrect event enum name; it was corrected to the engine's EvtZoneChange.
An intermediate commentary mistakenly read stale temporary lint output; fresh
slice-specific logs are the validation evidence below.

### Source defects, files and gates

No RON source changed, so no vocabulary-source snapshot refresh is required.
No new source defect was found. Gauntlets' Aura-only omission and Gaea's Liege's
known duplicate IsntAttacking branch remain open. No Go card, DSL export,
generator or startup wiring was removed. Whole-repository caller removal audits
remain for the design's cutover gates.

Engine files: new permanent_payment.go and permanent_payment_test.go;
shared eligible_player_payment.go and composable_costs.go; doc.go.
Compiler files: new permanent_payment.go and permanent_payment_test.go;
compiler.go, player_scope.go, optional_payment.go, contextual_collection.go,
inventory.go. Collection totals: definition_plan_test.go, inventory_test.go and
numeric_consumers_test.go. Docs: engine-rules-contracts.md, dsl-reference.md and
this ledger.

### Verification and measured collection result

All required checks passed after the collection expectation and allocation fixes:

- `go test ./pkg/mage/dsl/ron -run TestRONPermanentPayment -count=1` passed.
  Native TestPermanentPayment and existing eligible-player/payer regressions
  passed with `-count=1`.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
  The engine skill's additional `go test ./... -short -count=1` passed.
- The first fresh `make lint` found two preallocation issues in the new compiler
  and test. Explicit capacities fixed them. Focused tests, build, vet, engine
  short tests and `make test` passed again after those changes. Final `make lint`
  passed with **0 issues**. Automatic formatting/fixes were reviewed; the working
  tree contains only this slice's files. `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited **1**, as required while gaps remain:
  **2,190 compiled / 255 rejected records**, **849 accepted / 137 rejected names**
  (2,445 records / 986 names). The final issues array exactly equals the fresh
  baseline issues array with Cleansing removed. No other diagnostic changed,
  no accepted record regressed and no partial production registry was published.

There is no outstanding environment, build, test or lint blocker. Validation
logs used slice-specific names after the intermediate stale-log correction.

Estimated and actual compilation gains match: **one newly accepted unique card,
one record (Cleansing)**. Its complete RON gameplay mapping is added above.
Synthetic producer/consumer tests establish capability contracts; they are not
additional full-card coverage. Remaining compilation gaps are 137 distinct names
and 255 records. Complete collection gameplay and native runtime cutover remain
unfinished, as do Go-card deletion and obsolete DSL cleanup.

### Ranked next candidates

1. Eligible-player alternative/action payments for Worms of the Earth. Shared
   offers and candidate payer scope are ready; distinguish accepted damage from
   damage actually dealt. Land-entry prevention remains an independent blocker.
   Estimated immediate full-card gain: zero; prerequisite for one unique card.
2. Dependent target type comparison for Gauntlets of Chaos. Exchange-result guard
   and Aura-only correction remain. Estimated immediate full-card gain: zero.
3. Source-leave layer duration for Gaea's Liege. Defending-player membership,
   characteristic rules and its known source defect remain. Estimate zero until
   the full nested shape is audited in the next run.

Reassess shared multi-card candidates against fresh complete-record evidence
before selecting these remaining prerequisites. Do not infer unlocks from first
errors alone. This run ends after the per-permanent payment slice.

## 2026-10-13 — ELIGIBLE-PLAYER-ALTERNATIVE-PAYMENT

### Baseline and candidate selection

The initial working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The completed baseline loader
exited 1 with 2,190 compiled / 255 rejected records and 849 accepted /
137 rejected unique names. Collection rejection is expected and is separate
from build or test failure.

The latest entry's first candidate was dependency-ready. Complete records and
nested forms were inspected for all three ranked candidates; reprints count once:

| Candidate | Distinct affected cards | Estimated full-card unlocks | Other blockers | Downstream benefit and effort |
| --- | --- | --- | --- | --- |
| Eligible-player alternative/action payments | Worms of the Earth; Scarwood Bandits retains its existing fixed-mana form | 0 | Worms still needs CardsCantEnterTheBattlefield | Completes the sacrifice-or-damage upkeep prerequisite. Existing action preparation, transactions and payer frames make this a moderate adapter/engine/test slice. |
| Dependent target type comparison | Gauntlets of Chaos | 0 | PermanentsChangedControlThisWay and its Aura-only source correction | Enables the two-target exchange declaration; requires dependent announcement and resolution legality plus exchange-result work. Moderate to high effort. |
| Source-leave layer duration | Gaea's Liege | 0 | Defending-player forest membership, characteristic rules and duplicate IsntAttacking source defect | Enables saved land-type effects; requires broader layer/source-lifetime tests before the remaining static rules. Moderate to high effort. |

Selected the first ranked prerequisite. This completes one cohesive resolution
payment shape rather than combining it with independent entry prevention. The
service reuses action payments already used by Farrel's Zealot, Delif's Cone and
Floral Spuzzem. Their complete RON records, plus Farrel's Mantle's blocked record,
were read before the shared adapter was extracted. Their existing mappings
remain unchanged. No new complete card is claimed from the extracted rule.

The pre-edit contract was: offer affordable complete alternatives in turn order;
stop at the first payment; keep ActionPlayer local; distinguish accepted damage
from actual damage; preserve rollback, exact visits, source LKI, phasing, copies,
clones, delays and declaration rejection.

### Supported shape, engine boundary and dependencies

RON `AnyPlayerMayCost(AnyPlayer | Opponent, payment)` now accepts:

- Fixed `PayMana` with no X.
- `SacrificeAPermanent(filter)` and positive fixed
  `SacrificeNumberPermanents(Integer(n), filter)` through existing cost filters.
- `HavePermanentDealDamage(permanentRef, resolutionValue, recipient)` through
  typed permanent, value and singular damage-recipient adapters.
- Nested `Or([payment, payment, ...])` lists with at least two alternatives.
  Every descendant must compile; unsupported options still reject the declaration.

The adjacent If/Unless/IfElse uses CostWasPaid or CostWasntPaid and nonempty
branches. ActionPlayer is the candidate payer within damage options and the
successful payer within the paid branch. It cannot escape the pair or occur in
an unpaid branch. The forms apply in ordinary resolving sequences, including
permanent activations, triggers and delays. Entry, replacement and announcement
contexts retain their separate contracts. Variable sacrifice counts, other
payment actions, and combined action/cost instructions remain unsupported.

Engine API:

- `ResolutionPaymentOption` declares one complete option.
- `CostPaymentOption(CostGroup) ResolutionPaymentOption` uses shared costs.
- `ActionPaymentOption(ResolutionActionPayment) ResolutionPaymentOption` uses
  the shared action preparation and damage/life/destruction execution contract.
- `PayAnyAlternativeOrElseWithPayer(PlayerSet, []ResolutionPaymentOption,
  *PlayerBinding, ResolvingEffect, ResolvingEffect) ResolvingEffect` offers options
  and resolves one branch with the original effect controller.

Each offer prepares action values, recipient visits and source information before
choices. Only available options are listed; one option needs no mode choice.
Invalid selection or failed rechecks allow the next eligible player to pay.
Ordinary costs use tryResolutionCostPayment and the existing transaction;
failed payments do not commit partial resources. Damage uses the established
ResolutionActionPayment contract: prevention does not undo accepted payment.
Matching sacrifice uses the shared simultaneous zone batch and departure LKI.

The existing eligiblePlayerPayment effect owns payer restoration and definition
scope. A shared offerEligibleResolutionPayment loop now serves ordinary,
per-permanent and alternative payments. Existing binding validation keeps the
payer assigned only in option and paid-body edges. Effect properties include
action options. Source-read detection includes payment actions, so delayed source
queries capture exact source information without offers or mutation. Prepared
values stay local to Apply; factories create fresh bindings and option declarations.
Old payment APIs remain available without signature changes.

RON owns syntax, diagnostics and lexical plans. Mage owns ordering, choices,
transactions, action preparation, source information and result frames. No Go-card
fallback, card-name branch, last-action slot or partial production registry was
added. No independent cost, damage or zone implementation was introduced.

### Failing-first evidence and gameplay mappings

Before production changes, native TestEligibleAlternativePayment failed to compile
for the missing option type, constructors and eligible-alternative API. After test
setup API corrections, the expected missing-signature failure was confirmed.
TestRONEligibleAlternativePaymentWorms then failed with the real DRK record's
unsupported eligible-player payment cost (Or). Both failures preceded production
implementation.

Native tests:

- TestEligibleAlternativePayment: damage and sacrifice selection, prevention,
  unaffordable option exclusion, decline, invalid selection, transaction rollback,
  next-payer success and local payer restoration.
- TestEligibleAlternativePaymentOrderAndRecheck: both active players, exact target
  visits, offer-time phasing and return, phase cycles, zero and negative damage.
- TestEligibleAlternativePaymentInvalid and Declarations: nil options, absent cost
  groups, nil action inputs, nil player sets/payers, absent branches, unpaid reads
  and escaped payer reads fail engine definition validation.
- TestEligibleAlternativePaymentSacrificeBatch: both departures share retained
  battlefield information, both lands leave together, and exactly two committed
  triggers resolve. The transaction's speculative affordability/validation games
  can also evaluate conditions; assertions use the event reader and committed
  effects, not side effects on an external counter or the original game during
  speculative payment.

RON card-to-test and synthetic mapping:

- **Worms of the Earth → TestRONEligibleAlternativePaymentWorms** reads the complete
  unchanged DRK record, executes its isolated unchanged upkeep rule, and covers
  damage acceptance, full prevention, sacrifice and decline. The complete record
  still rejects CardsCantEnterTheBattlefield. This proves its payment rule, not
  full-card gameplay.
- **Scarwood Bandits → TestRONSourcePresenceDurationScarwood** retains complete
  RON gameplay for the unchanged fixed-mana payment and source-present duration.
- **Delif's Cone, Farrel's Zealot, Floral Spuzzem → TestRONActionPaymentRealCards
  and TestRONActionPaymentNativeCombat** retain their existing complete mappings
  through the extracted shared action-payment adapter. Farrel's Mantle remains
  blocked by its recorded host-event and source-data gaps.
- TestRONEligibleAlternativePaymentFrames: source departure/return/phasing/control,
  source LKI and lifelink, stack copies, cloned pending resolution, delayed source
  capture, waiting-query purity and successful-payer branch reads.
- TestRONEligibleAlternativePaymentInvalid: nested alternatives, malformed/empty
  lists, unsupported life options, invalid sacrifice counts, unbound references,
  malformed payer reads, escaped/unpaid reads and atomic collection rejection.
- Existing TestRONEligiblePlayerPayment*, TestRONEligiblePayerScope*,
  TestRONActionPayment*, TestEligiblePlayerPayment*, TestEligiblePayerScope*,
  TestResolutionActionPayment* and TestPermanentPayment* retain shared-service
  and existing card regressions, including targets, nested payments and delays.

### Files, source defects and gates

New engine files: eligible_alternative_payment.go and its test.
Shared engine file: eligible_player_payment.go. Compiler files:
eligible_player_payment.go, action_payment.go and new
eligible_alternative_payment_test.go. Documentation: pkg/mage/doc.go,
docs/engine-rules-contracts.md, docs/dsl-reference.md and this ledger.
The inventory already classifies nested resolution-cost Or, sacrifice and damage
arguments correctly; no inventory adapter or collection-count change is needed.

Local data/DRK.json Oracle text confirms Worms of the Earth's sacrifice-or-damage
choice and subsequent self-destruction. CR 118.3/11/12 and the existing action
payment contract establish affordability and acceptance despite prevention.

No RON source changed, so the source vocabulary snapshot does not need refresh.
No new source defect was found. Gauntlets' Aura omission, Gaea's Liege's duplicate
IsntAttacking branch and other recorded source defects remain open. No Go card,
DSL export, generator or startup wiring was removed; the removal gates are open.

### Verification and measured collection result

All required checks passed after the test setup and lint corrections:

- `go test ./pkg/mage/dsl/ron -run TestRONEligibleAlternativePayment -count=1`
  passed. Expanded focused runs for the existing eligible-player, payer-scope
  and action-payment regressions passed. Native alternative-payment, eligible
  payer/player, action-payment and permanent-payment regressions passed with
  `-count=1`.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
  The engine skill's additional `go test ./... -short -count=1` also passed.
- The first engine-short run caught the sacrifice test's original-game assertion
  during a speculative transaction. The test now checks simultaneous departure
  LKI on the event reader and exactly two committed trigger effects, as described
  above; it passes without an engine change or a weaker gameplay assertion.
- The first `make lint` found one wasted boolean initialization. Replacing the
  initialization with a declaration fixed it. Focused tests, build, vet, engine
  short tests and `make test` passed again after the fix. The final `make lint`
  passed with **0 issues**. Automatic changes were reviewed: import formatting
  and maps.Copy in the isolated-record test preserve its independent field map.
  `git diff --check` passed. Only this slice's ten files changed.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected while gaps remain:
  **2,190 compiled / 255 rejected records**, **849 accepted / 137 rejected names**
  (2,445 records / 986 names). The entire final JSON report exactly equals the
  fresh baseline report. Worms still reports DRK.lines.ron:118:221 unsupported
  rule CardsCantEnterTheBattlefield. No diagnostic changed, no accepted record
  regressed, and no partial production registry was published.

There is no outstanding environment, build, test or lint blocker.

Estimated and actual gains are **zero newly accepted unique cards / zero records**.
Payment-rule gameplay evidence increased; full-card gameplay coverage did not.
Worms is now blocked solely by the independent entry prohibition at its enclosing
record. Collection compilation gaps remain 137 names / 255 records. Complete
collection gameplay, native runtime cutover, Go-card deletion and obsolete DSL
cleanup remain unfinished.

### Ranked next candidates

1. CardsCantEnterTheBattlefield land-entry prohibition. Worms of the Earth's
   payment prerequisite is complete; inspect replacement ordering and every entry
   path. Estimated immediate full-card gain: one unique card / one record.
2. Dependent target type comparison for Gauntlets of Chaos. Exchange-result guard
   and Aura-only source correction remain. Estimated immediate gain: zero.
3. Source-leave layer duration for Gaea's Liege. Defending-player membership,
   characteristic rules and its known source defect remain. Estimate zero until
   the complete nested shape and source evidence are resolved.

Reassess shared multi-card work with complete records before selecting a next
slice. This run stops after eligible-player alternative resolution payments.

## 2026-10-13 — STATIC-LAND-ENTRY-PROHIBITION

### Baseline, candidates and selection

The starting working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Fresh
`go run ./cmd/loadcards -json` exited 1 for collection rejection, with
2,190 compiled / 255 rejected records and 849 accepted / 137 rejected names.
This was not a build or test failure. No unrelated work was changed.

The latest entry's ranked candidates were assessed from complete unchanged RON
records and nested forms, with reprints counted once:

| Candidate | Distinct affected names | Estimated full-card gain | Other blockers | Downstream value | Effort |
| --- | --- | --- | --- | --- | --- |
| Static land-entry prohibition | Worms of the Earth | 1 name / 1 record | None after the prior payment slice | Shared legality query, copy rechecks and zone-retention entry paths; interaction coverage for Gaea's Touch and Untamed Wilds | Moderate; one adapter, one static service, entry-path repairs and contract tests; one run |
| Dependent target type comparison | Gauntlets of Chaos | 0 | Exchange-result guard and recorded Aura-only source correction | Cross-target type legality for an exchange | High; prerequisite slice followed by separate result and data work |
| Source-leave layer duration | Gaea's Liege | 0 across 6 records | Defending-player membership, characteristic behavior and the duplicate IsntAttacking source defect | Source-bound type effects and characteristic values | High; several independent contracts and source evidence |

Candidate 1 was the only bounded, dependency-ready full-card unlock among these
candidates. The two alternatives provide no immediate shared multi-card gain.
The broad phases are dependency guidance; first-error counts do not establish
full-card unlocks. This run completes the final independent Worms rule and does
not start another capability.

Before implementation, the selected contract was an ordinary permanent rule
with exact syntax `CardsCantEnterTheBattlefield(IsCardtype(Land))`. Acceptance
cases included invalid declarations, full Worms gameplay, source lifetimes,
zone retention, copy rechecks and token batches. RON owns spelling, shape checks
and fresh factories. The engine owns entry legality, replacements and movement.
No reference or result scope is added by this static rule.

### Exact signatures, dependencies and reuse

- `PreventLandEntry() StaticRule` creates a reusable battlefield source rule.
- `Game.IsLandEntryProhibited(card Card) bool` is a pure query. Nil and nonland
  cards return false. Legacy global entry flags remain supported.
- RON accepts exactly one `IsCardtype(Land)` argument on an ordinary permanent
  rule. Other card types, compound filters, wrong arities, unbound references,
  nonpermanent rules and resolving action contexts reject before gameplay.
- Inventory descendants now use `entry_prohibition_card_filter` and `card_type`.
  The strict loader and atomic registry publication behavior remain unchanged.

The typed effect reads the existing printed declaration wrapper, source visit
and active-effect service instead of writing a cached global flag. Phasing,
face-down status, departure and ability removal stop it; tapping and control
changes do not. Return, overlapping sources and clones keep separate lifetimes.
No new mutable declaration, result slot or resolution callback is shared.

Existing entry preparation, copied definitions, replacement priorities, zone
storage, source attribution, token batches and event services remain in use.
Initial land entry is prohibited before entry replacements. A copy that becomes
a land stops the replacement loop before later entry operations. This is a
prohibition, not a new replacement effect. Prohibited token members produce no
entry event; permitted members still enter in the shared batch.

Three pre-existing entry-path defects were exposed by failing tests: legacy hand
entry consumed a prohibited card, source-graveyard return removed a prohibited
card, and delayed exile return removed a prohibited card. Those paths now retain
their cards. Hand and graveyard entry prepare before consuming a saved visit and
recheck the visit after choices; prohibited copies retain that visit. Delayed
exile return prepares before removal. Library search retains its existing
restoration and shuffle behavior. Linked exile batches share the active query.
Land play and playable-land queries also reject prohibited entry before hand or
library removal and before turn-history changes.

The local obsolete `graveyardCardCanEnter` helper had no remaining callers after
the shared entry-path changes and was removed. No broad DSL function was removed.
The existing Go definitions, legacy engine APIs, generators and startup wiring
remain in place because their removal gates are not satisfied.

### Failing-first evidence and gameplay mappings

`TestLandEntryRestriction` first failed to compile for the missing
`PreventLandEntry` signature. A test setup typo in the existing token-batch method
name was corrected, and the missing-signature failure was confirmed again before
production implementation. The initial service then exposed assertion failures
for immediate phasing and the three zone-retention paths listed above.

Before RON production changes, `TestRONLandEntryRestrictionWorms` and
`AtomicLoad` failed with unsupported CardsCantEnterTheBattlefield. Copy tests
then failed for a later replacement firing after land copying and for losing the
hand visit. The land-play test failed because a prohibited land play succeeded.
Each corresponding repair followed its observed assertion failure.

Native mappings:

- `TestLandEntryRestrictionLifecycle`: normal, tapped, control-changed, phased,
  phase-cycle, ability removal/restoration, departed/returned, cloned, overlapping
  and face-down sources; both players, land creatures and permitted nonlands;
  definition validation rejects a static entry rule on an instant.
- `TestLandEntryRestrictionZoneRetention`: hand, optional hand, graveyard,
  source-graveyard, library and delayed exile paths retain prohibited cards.
- `TestLandEntryRestrictionBatch`: mixed permitted/prohibited members and land
  tokens; only permitted members produce battlefield entry events.
- `TestLandEntryRestrictionCopyRecheck`: direct, hand, graveyard, library and
  delayed exile copies that become lands; no later replacement fires, no
  prohibited copy is placed in the graveyard, and saved visits remain intact.
- `TestLandEntryRestrictionPlay`: failed land play preserves hand identity and
  turn history; playable-land queries exclude prohibited entry.

RON mappings:

- **Worms of the Earth → TestRONLandEntryRestrictionWorms**: complete unchanged
  DRK record, cast through the engine; both upkeep players; land play and direct
  entry prohibition for both players; sacrifice, damage acceptance, full damage
  prevention, decline, self-destruction and restriction expiration. The complete
  record is newly accepted. Existing isolated payment/frame tests still cover
  source LKI, copies, clones, delayed frames and transaction behavior.
- **Gaea's Touch → TestRONLandEntryRestrictionRealEntryPaths/hand**: complete
  unchanged DRK record cannot consume a Forest from hand under Worms.
  Existing TestRONOptionalHandEntryGaeasTouch retains its full-card mapping.
- **Untamed Wilds → TestRONLandEntryRestrictionRealEntryPaths/library**: complete
  unchanged LEG spell retains a searched basic land in the library under Worms.
  Existing library-search scenarios remain valid.
- `RealEntryPaths/clone`: complete Worms source rules remain independent between
  a live game and its clone after the clone loses the source.
- `TestRONLandEntryRestrictionInvalid` and `AtomicLoad`: static shape/context
  rejection and no partial publication when any record fails.

### Source evidence, files and validation

`data/DRK.json` confirms Worms' land-play, land-entry and sacrifice-or-damage
upkeep text. CR 101.2 and 614.17 establish prohibition before the event and
entry-characteristic rechecks; CR 614.12 supplies entry-choice timing.
No source defect was corrected or introduced. All RON records remain unchanged,
so the source vocabulary snapshot needs no refresh. Previously recorded source
issues for Gauntlets, Gaea's Liege and other cards remain open.

Engine changes: new land_entry_restriction.go/test; entry integration in game.go,
effect_manager.go, effect_cards.go, optional_hand_entry.go, effect_removal.go,
linked_exile_entry.go and land_play_restriction.go. Compiler changes: new
land_entry_restriction.go/test, compiler.go and inventory.go. Existing payment,
land-play, definition-plan, numeric, inventory and upkeep inventory tests update
exact collection expectations and keep explicit Worms coverage. Documentation:
pkg/mage/doc.go, engine-rules-contracts.md, dsl-reference.md and this ledger.

Required checks:

- Focused RON land-entry tests and expanded payment, land-play, definition-plan,
  numeric and inventory tests pass with `-count=1`.
- Native land-entry and existing optional-hand, copy, library-search, linked-exile,
  permanent-spell-copy and land-play regressions pass with `-count=1`.
- `make build` and `go vet ./...` pass.
- `go test ./pkg/mage/... -short -count=1` and `make test` pass.
- The engine skill's additional `go test ./... -short -count=1` passes.
- The first engine-short run found seven obsolete expected counts or rejection
  assertions after Worms became accepted. Exact record counts changed by one;
  upkeep inventories now explicitly require Worms and count 59 names / 120
  records. The formerly unsupported fixture uses an unsupported Creature filter
  instead of the newly supported Land filter. No assertion was weakened.
  Focused tests and all build, vet, engine-short and full-suite checks then passed.
- Initial `make lint` rejected a test helper variable named copy for shadowing
  Go's built-in. It was renamed clone. The final focused RON/native tests,
  build, vet, engine short tests and `make test` passed again after that change.
  Final `make lint` passed with **0 issues**. Automatic changes were reviewed;
  only import formatting changed, with no unrelated changes. `git diff --check`
  passed. There is no outstanding environment, build, test or lint blocker.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected for the remaining
  unsupported collection: **2,191 compiled / 254 rejected records**,
  **850 accepted / 136 rejected names**, 2,445 records / 986 names total.
  Comparing every diagnostic with the fresh baseline removes exactly
  Worms of the Earth's DRK.lines.ron:118:221 CardsCantEnterTheBattlefield issue.
  No other diagnostic changed and no new diagnostic appeared. No accepted record
  regressed; no partial production registry was published.

Estimated and actual compilation gains match: **one unique card / one record**.
Full-card RON gameplay coverage gains **Worms of the Earth**. Existing Gaea's
Touch and Untamed Wilds mappings gain prohibition interaction evidence, without
new compilation gains. Compilation and gameplay coverage remain separate.
Collection gaps remain 136 names / 254 records. Complete collection gameplay,
embedded native loading, application cutover, Go-card deletion and obsolete DSL
cleanup remain unfinished.

### Ranked next candidates

1. Dependent permanent target type comparison for Gauntlets of Chaos. Its
   exchange-result guard and documented Aura-only source correction remain;
   estimated immediate full-card gain is zero. Reassess useful shared multi-card
   work before choosing this single-card prerequisite.
2. Source-leave layer duration for Gaea's Liege. Defending-player membership,
   characteristic behavior and the known duplicate predicate remain; estimated
   immediate gain is zero until the complete nested requirements are resolved.
3. Graveyard-source trigger arming for Nether Shadow. Its six reprints share one
   definition; the full nested shape also needs a graveyard-order intervening-if,
   exact zone visits and optional entry. Establish dependencies and an unlock
   estimate before implementation; compare shared event work first.

This run stops after static land-entry prohibition.

## 2026-10-13 — SAVED-DAMAGE-OUTCOMES

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and `go test ./pkg/mage/dsl/ron
-count=1` passed. Fresh strict loading exited 1 with 2,191 compiled / 254
rejected records and 850 accepted / 136 rejected unique names. Collection
rejection was not a build or test failure.

The latest ledger candidate #1 was assessed first. Complete records were read
for Gauntlets of Chaos (LEG), all six Gaea's Liege reprints, Nether Shadow's six
reprints, Syphon Soul (LEG), Mishra's War Machine (ATQ/3ED), and Drain Life's six
reprints. Nested declarations were checked; reprints count once. Targeted
vocabulary queries covered the target comparison, source-leave duration, `Other`,
actual damage total, clamped damage total and source/player result condition.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers / downstream benefit | Estimated implementation and test effort |
| --- | --- | --- | --- | --- |
| Dependent target type comparison, latest candidate #1 | Gauntlets of Chaos | 0 | Exchange-result condition and documented Aura-only source defect remain. Shared comparison would enable dependent target legality | Medium: target dependency, announcement and resolution rechecks, copies and identity |
| Source-leave layer duration | Gaea's Liege | 0 | Defending-player membership, characteristic behavior and duplicated `IsntAttacking` source defect remain. Existing source duration services reduce engine work | Small adapter; large additional full-card/rules/data work |
| Saved instruction damage outcomes | Syphon Soul; Mishra's War Machine; downstream Drain Life | 2 names / 3 records | Syphon needs player exclusion and saved actual total. Machine needs a saved source/player condition. Drain Life still needs restricted X payment and a distinct clamped total. Shared actual results enable all three | Medium, bounded: shared singular capture, typed result branch and player batch adapter, lexical compiler producer, prevention/identity/frame tests |

Nether Shadow's full nested declaration was also checked. Graveyard arming,
order-based intervening-if and optional exact-visit entry are several contracts;
it is not a smaller ready shared slice. The selected shared result contract gives
the largest credible gain here. Gauntlets remains deferred without concealing its
source defect. No production registry or card-name branch was added.

### Established signatures and boundaries

Engine additions:

- `DamageOutcomesInto(*DamageResultsBinding) DamageResultOption` saves the shared
  singular executor's positive actual outcome and clears old output before every
  attempt. Existing `DamageResultsInto` and all current callers keep their APIs.
- `IfDamageDealtToPlayer(results *DamageResultsBinding, source DamageSourceRef,
  player PlayerRef, body ResolvingEffect) ResolvingEffect` runs the body once for
  a matching positive outcome. `DamageSource.SameObject` requires exact visits.
- `PlayerDamageRecipients(PlayerSet) DamageRecipientSet` supplies distinct present
  players to the existing simultaneous damage service. Its typed operands retain
  explicit waiting source reads.

RON lowers direct `SpellDealsDamage(ThisSpell, amount, recipient)` and
`PermanentDealsDamage(permanentRef, amount, recipient)` with result consumers in
ordinary resolving action sequences. Singular `Player`, `Permanent` and union
bindings use `DealDamageFrom`; `EachPermanent` and `EachPlayer(AnyPlayer |
Opponent | Other(playerRef))` use the shared batch with one amount read. The
symbol `TheAmountOfDamageDealtThisWay` reads the latest local producer.
`If(PermanentDealtDamageToPlayerThisWay(permanentRef, playerRef), [actions])`
checks its saved actual source and recipient. Spell/permanent references use
existing typed compilation and LKI services. Result declarations are fresh per
factory; each continuation has its own plan and cloned binding map.

Branch producers cannot export results. Nested direct instructions replace the
local producer. Delayed bodies retain saved totals and outcomes in existing
frames. Unsupported instruction families cannot masquerade as a producer.
Clamped totals, per-recipient amount bindings, entry operations and replacement
operations remain separate unsupported shapes. Invalid declarations fail during
compilation or definition validation, not at game time.

Reuse: shared damage/replacement executors, immutable `damageOutcome` storage,
number/result frames, delayed captures, stack copies, game clones, source LKI,
exact incarnation matching, optional discard transaction and upkeep triggers.
There is no parallel damage system or global last-action state. No card source
changed. Runtime cutover and removal gates remain unsatisfied, so Go definitions
and DSL exports are retained; no unused function was removed without caller audit.

### Failing-first evidence and card-to-test mappings

Native tests first failed compilation for missing `DamageOutcomesInto` and
`IfDamageDealtToPlayer`, after fixing test-only context/player spelling. The
player batch test then failed for missing `PlayerDamageRecipients` before that
adapter was added. The RON tests failed with unsupported `Other`, unsupported
actual total and Machine's erroneous activation-history dispatch before compiler
changes. New lowering then exposed duplicate binding names and the requirement
for a typed player-only recipient set; both were fixed against failing tests.

- **Syphon Soul → TestRONDamageOutcomeSyphonSoul**: complete unchanged LEG spell,
  failed payment leaves hand intact, paid cast, resolution timing, other-player
  damage and exact actual life gain, partial/full prevention, copied spell with
  another controller, independent game clone outcomes.
- **Mishra's War Machine → TestRONDamageOutcomeWarMachine**: complete unchanged
  ATQ and 3ED records, banding and 5/5 metadata, controller upkeep only, optional
  discard paid/unpaid, partial/full prevention, conditional tap, controller
  change after detection, source departure/return/phasing, copied trigger and
  separate cloned resolution. Custom paid test players clone to BasePlayer;
  the clone independently declines and takes damage, which is asserted.
- `TestRONDamageOutcomeScopes`: latest direct instruction replaces older results,
  zero damage clears totals, nested branch producer does not overwrite its
  enclosing producer, exact union targets, delayed saved total after spell departure.
- `TestRONDamageOutcomeInvalid` and `ContextBoundaries`: unbound/escaped results, malformed totals and
  predicates, unsupported clamping/player reads and unsupported producer boundaries.
- `TestRONDamageOutcomeInventoryAndAtomicLoad`: typed source/player operand roles
  and atomic rejection of a collection with one invalid record.
- Native `TestDamageOutcomeCondition`: positive/partial/full prevention,
  redirection to a permanent, zero/stale outputs, wrong source/player, departure,
  return and phasing, historical identity and game clone isolation.
- Native `Definition`, `Batch`, `Waiting` and `WaitingPlayerSet`: missing and nil
  expressions, duplicate outputs, shared batch actual totals, explicit waiting
  source reads without execution or mutation.

Engine documentation records CR 120.4b–d's distinction between actual damage and
its life/counter results. Local `data/LEG.json` and `data/ATQ.json` confirm the
complete card instructions. No source defect was corrected or introduced.
Gauntlets' missing Aura-only filter and Gaea's Liege's duplicate predicate stay
open. RON sources are unchanged; no vocabulary snapshot refresh is required.

### Validation and measured gain

Required checks passed after the repairs below:

- `go test ./pkg/mage/dsl/ron -run TestRONDamageOutcome -count=1` and native
  `go test ./pkg/mage -run TestDamageOutcomeCondition -count=1` pass.
- Expanded inventory, definition-plan, discard-boundary and upkeep tests pass.
- `make build` and `go vet ./...` pass.
- `go test ./pkg/mage/... -short -count=1` and `make test` pass.
- The engine skill's additional `go test ./... -short -count=1` passes.
- Initial RON package tests reported eight obsolete count/boundary expectations
  after the three records became accepted. Exact counts are updated, Machine is
  explicitly required in upkeep inventories, and the formerly unsupported
  discard-boundary assertion now requires successful compilation. Keyword
  inventories now require all ten matching names; no unsupported error is
  tolerated. Gameplay assertions were retained.
- A new unsupported-producer test failed because packet damage between a
  captured instruction and a result read left the former result available.
  That unsupported producer family is now rejected explicitly.
- `TestRONDamageOutcomeContextBoundaries` failed for inherited result reads in
  entry/replacement contexts. Both total and condition consumers now reject
  those contexts explicitly. The focused and complete verification chain was
  rerun after this repair.
- First `make lint` reported preallocation and redundant-type style issues.
  The native test type was automatically simplified; the producer effects slice
  now reserves its complete capacity. Automatic formatting/type changes were
  reviewed. Only slice changes remain. The final lint pass reports **0 issues**.
- `git diff --check` passes after removing an added blank line at the ledger EOF.
  No environment, build, test or lint blocker remains.

Final `go run ./cmd/loadcards -json` exited **1**, as required while collection
support is incomplete: **2,194 compiled / 251 rejected records** and **852
accepted / 134 rejected names**, out of 2,445 records / 986 names. Comparing all
diagnostics with the fresh baseline removes exactly Syphon Soul's
LEG.lines.ron:255:218 `Other` error and Machine's ATQ.lines.ron:45:348 and
3ED.lines.ron:170:348 activation-history errors. No other diagnostic changed;
there are no new diagnostics and no accepted-record regression. No partial
production registry was published. Post-lint focused RON/native tests and
`git diff --check` pass again; reviewed automatic changes were formatting and
the redundant test type already corrected before the complete verification rerun.

Estimated and actual gains match: **two unique names / three records**. Full-card
RON gameplay mappings gain **Syphon Soul** and **Mishra's War Machine**.
Compilation and gameplay evidence remain separate; this does not verify all
852 accepted names. Remaining gaps are 134 names / 251 records, full collection
gameplay, embedded native loading, startup cutover, Go-card deletion and obsolete
DSL cleanup. The next run must reassess the candidates below; this run implements
no second slice.

Changed engine files: `damage_outcome_condition.go`/test,
`composable_actions.go`, `waiting_delayed_source.go` and `doc.go`.
Changed compiler files: `damage_outcome.go`/test, `compiler.go`,
`optional_payment.go`, `player_scope.go`, `sacrifice_value.go` and `inventory.go`.
Exact collection expectations and former unsupported-card assertions changed in
`chosen_discard_cost_test.go`, `definition_plan_test.go`, `inventory_test.go`,
`numeric_consumers_test.go` and the upkeep keyword/layer/trigger inventory tests.
API/rules documentation and this progress ledger are updated.

### Ranked next candidates

1. Shared announcement mana restrictions and clamped outcomes, starting from
   Drain Life. Its saved actual-damage prerequisite now exists, but restricted X
   payment and the Oracle clamp remain separate. Compare the complete nested
   requirements of other restricted-payment cards before selecting the next
   bounded engine prerequisite. Estimated immediate gain from a prerequisite
   alone is zero; completing both remaining contracts could unlock one name /
   six records.
2. Dependent permanent target type comparison for Gauntlets of Chaos. Immediate
   gain remains zero until exchange-result handling and its recorded Aura-only
   source correction are complete; compare shared target work first.
3. Graveyard-source trigger arming for Nether Shadow. Six reprints share one
   definition. Order-based intervening-if, exact visits and optional entry still
   need distinct contracts. Compare shared zone-source events before selection.

Gaea's Liege remains deferred with its defending-player and source-data blockers.
This run stops after saved damage outcomes.

## 2026-10-13 — TEMPORARY-SINGULAR-CONTROL

### Baseline and candidate selection

The working tree was clean. Fresh `make build` and the complete RON package
suite passed. Baseline `go run ./cmd/loadcards -json` exited 1 with 2,194
compiled / 251 rejected records and 852 accepted / 134 rejected names. This
was collection rejection, not a build or test failure. All complete records for
the compared candidates were read, including nested bodies and endpoints.
Targeted vocabulary reads covered casting restrictions, source-control duration
and dependent card-type comparison. Reprints count once.

| Candidate | Unique affected cards | Estimated complete unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Ranked #1 restricted announcement X payment | Drain Life (six records) | Zero for payment alone; one after a separate clamp contract | Saved actual damage now exists, but restricted X payment and the Oracle life/toughness clamp are independent. Sunglasses of Urza, North Star and Mishra's Workshop have different conversion, one-use cast and production restrictions; this X-only adapter would not unlock them. | Medium engine/planner integration and payment tests, then a separate clamp slice |
| Ranked #2 dependent permanent type comparison | Gauntlets of Chaos; related SharesACardtypeWithPermanent operand in Enchantment Alteration | Zero | Gauntlets also needs exchange-result binding and its recorded Aura-only source repair. Alteration needs attachment action and resolving host selection. Shared comparison work can enable both downstream cards. | Medium typed target/reference and legality tests, plus later independent actions |
| Temporary singular control duration | Preacher (DRK); Disharmony (LEG) | Two names / two records | All other nested forms already compile. Preacher's opponent target choice, optional untap and tap payment exist. Disharmony's casting window, attacking target, untap and combat removal exist. This closes both direct duration forms without changing controller-bound endpoints. | Small adapter and checked engine extension; broad lifecycle and card tests |

Ranked #3 Nether Shadow was also inspected: six identical graveyard-source
records still need trigger arming, graveyard-order intervening-if and exact-visit
optional return. It is a larger single-card pipeline than the selected slice.
The selected multi-card adapter has the largest credible immediate gain for this
run. No unrelated mana, comparison, graveyard or plural-control slice was started.

### Contract, implementation and reuse boundary

`GainControlOfPermanentUntil(permanentRef, UntilEndOfTurn)` supports ordinary
resolving spell, permanent activation, trigger and delayed contexts. It lowers
to `mage.Until(core.EndOfTurn, mage.ChangeController(ref, mage.You))`.
`GainControlOfPermanentUntil(permanentRef,
ForAsLongAsPermanentRemainsTapped(ThisPermanent))` requires a permanent source
and lowers to `mage.UntilSourceUntaps(mage.This,
mage.ChangeController(ref, mage.You))`. Source spelling must be a symbol.
Both forms reject entry and draw-replacement bodies; malformed endpoints and
unbound or incompatible references remain declaration errors.

The engine now accepts control modifications in checked `UntilSourceUntaps`
definitions. Existing capture, layer 2 timestamps, incarnation checks, source
transition hooks, simultaneous untap, cleanup, frames, copying and cloning run
these effects. No parallel control system, global result, fallback or card-name
branch was added. The saved effect controller receives control even after source
control changes. End-of-turn control ends at cleanup. Tapped-source control
requires the present unphased tapped original source at resolution. Actual
untapping, departure or source phasing permanently ends it. Stun replacements,
source control changes and ability removal do not end it. Target phasing keeps
identity; a returned target cannot inherit control.

RON owns spelling/context validation and fresh per-factory typed declarations.
Engine APIs keep their signatures and existing callers. The earlier invalid
native tapped-control fixture now tests the still-unsupported `UntilSourceLeaves`
control shape; new positive tests explicitly validate tapped control. Previously
unsupported RON assertions now use malformed endpoint fixtures or require full
Preacher compilation. No gameplay assertion was removed.

### Failing-first evidence and card-to-test mapping

Before implementation, `TestTemporaryControlDuration` failed definition validation
with the exact missing tapped-control contract. `TestRONTemporaryControlPreacher`
failed on unsupported tapped duration. `TestRONTemporaryControlDisharmony` failed
on its erroneous permanent-source requirement. These expected failures were
confirmed before production edits. Additional test-only compile errors in the
new start/waiting tests were repaired using the existing typed binding/source API.

- **Preacher → TestRONTemporaryControlPreacher**: full unchanged record;
  opponent chooses a creature they control despite supplied controller targets;
  summoning-sickness rollback, resolution timing, untap/phase/return before
  resolution, source control changes, target return/phasing/shroud, source
  expiration, ability removal, copies and pending clone isolation.
- **Preacher → TestRONTemporaryControlPreacherCastingAndUntap**: printed mana
  payment and rollback, 1/1 entry metadata, summoning sickness, printed optional
  untap, decline/accept, stun replacement, and the untap offer after target departure.
- **Disharmony → TestRONTemporaryControlDisharmony**: full unchanged record;
  combat window and unpaid-cast rollback, attacking target recheck, resolution
  untap/removal/control, shroud/phasing/return, copies, clones and native cleanup.
- **Disharmony → TestRONTemporaryControlContextsAndCopies**: copy retargeting
  keeps independent subjects; installed clone lifetime is independent; entry and
  replacement contexts reject the new forms.
- `TestRONTemporaryControlDelayed`: both endpoint forms save the controller and
  exact source through future triggers; repeated waiting queries make no changes;
  saved control resolves and expires independently of an intervening control effect.
- `TestRONTemporaryControlInvalid`: malformed symbol/arity/source operands,
  spell-source tapped durations and unsupported endpoints remain errors.
- Native `TestTemporaryControlDuration`: checked declaration, shared layer 2
  execution, untap-retap/source phase/return, target phase/return, control changes,
  ability removal, stun, cleanup and installed game clone isolation.
- Native `TestTemporaryControlDurationStart`: untapped, absent, phased and
  returned sources install nothing; retapping before resolution permits a start.
- Native `TestTemporaryControlDurationSimultaneousUntap`: both control effects
  end and both sources untap before either untap notification.

Local authoritative `data/DRK.json` and `data/LEG.json` confirm both full records.
CR 611.2a–c, 514.2 and 702.26f support the lifetime and phasing contracts. No source
defect was changed. Gauntlets' Aura-only defect and Gaea's Liege's duplicate
predicate remain open. RON sources are unchanged; no vocabulary refresh is needed.
API and rules documentation now records only these established control contexts.

### Validation and measured gain

Required verification passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONTemporaryControl -count=1` passed.
- Native `go test ./pkg/mage -run 'TestTemporaryControlDuration|TestUntilSourceUntaps' -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's additional `go test ./... -short -count=1` passed.
- Complete `go test ./pkg/mage/dsl/ron -count=1` passed after updating exact
  collection counts and formerly unsupported duration expectations. Its initial
  run exposed six obsolete count/boundary assertions; all now require the new
  behavior or a still-invalid declaration, while retaining gameplay checks.
- `make lint` passed with **0 issues**. Automatic edits added import-group
  spacing in the two new test files. They were reviewed; no unrelated files or
  semantic changes were introduced. Post-format focused RON/native tests passed.
- `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited **1**, as required while unsupported
  records remain. Fresh full-diagnostic comparison confirmed the counts below.

No environment, build, test or lint blocker remains.

Final collection gains are exactly **two names / two records**, matching the
estimate: **854 accepted / 132 rejected names**, **2,196 compiled / 249 rejected
records**, out of 986 names / 2,445 records. Comparing every loader diagnostic
removes only Preacher's DRK:82 tapped-duration error and Disharmony's LEG:63
permanent-source error. No diagnostics are added or changed. Compilation evidence
and gameplay evidence are separate: this run adds complete gameplay mappings for
these two cards, not every accepted card. No partial production registry is published.

Changed production files: `pkg/mage/dsl/ron/source_control.go` and
`pkg/mage/composable_continuous.go`. New native/RON temporary-control tests and
existing collection/boundary expectations are updated. `pkg/mage/doc.go`,
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger record
the contract. No Go card definition or DSL function is removed: cutover/removal
gates are still open and no caller-removal audit is warranted for this adapter.
Remaining gaps include 132 names / 249 records, full collection gameplay,
embedded native loading, startup cutover, Go-card deletion and obsolete DSL cleanup.

### Ranked next candidates

1. Shared dependent permanent type comparison: Gauntlets of Chaos and Enchantment
   Alteration. Zero immediate unlocks, but a concrete shared prerequisite for two
   names. Compare target and resolving reference contracts; keep exchange results,
   attachment actions and Gauntlets' Aura-only defect explicit.
2. Restricted announcement mana payment and clamped damage outcomes: Drain Life.
   Zero immediate gain from one prerequisite; one name / six records after both
   contracts. Compare complete restricted-payment records before selection.
3. Graveyard-source trigger arming: Nether Shadow. Six records, one name;
   order-based intervening-if, exact visits and optional entry still need contracts.

This run stops after temporary singular control.

## 2026-10-13 — RESOLVING-PERMANENT-TYPE-COMPARISON

### Candidate comparison and selection

The worktree was clean before edits. AGENTS.md, the engine reference, execution
design, current status and latest entry were read. Fresh complete RON package
tests supplied passing compilation/test evidence before production edits. The
baseline strict loader exited 1 with 2,196 compiled / 249 rejected records and
854 accepted / 132 rejected names. Collection rejection was not a build failure.

Complete records were inspected for Gauntlets of Chaos and Enchantment Alteration
in LEG and Drain Life and Nether Shadow in all six reprint sets. Targeted
vocabulary inspection found one occurrence of each card-type comparison form.
Nested forms and remaining blockers determined the estimates; reprints count once.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream benefit | Implementation/test effort |
| --- | --- | --- | --- | --- |
| Shared permanent card-type comparison | Gauntlets of Chaos; Enchantment Alteration | 0 names / 0 records from the resolving prerequisite | Gauntlets needs dependent target queries, exchange results and the Aura-only source repair. Alteration needs attachment movement and a target-derived host. A typed current/LKI comparison supplies a shared information and selection contract for both downstream capabilities. | Medium for the resolving context; high for announcement context and every target acquisition path. |
| Restricted announcement X mana | Drain Life | 0 from payment alone; 1 name / 6 records after clamped outcomes | Saved damage outcomes exist. Restricted X payment and pre-damage life/loyalty/toughness clamping remain. The payment contract could serve later restricted casting forms, but this collection form affects one name. | Medium for payment; medium/high for complete damage clamping and replacements. |
| Graveyard-source trigger arming | Nether Shadow | 0 from arming alone; 1 name / 6 records after all prerequisites | Needs exact graveyard visits, ordered creature cards above the source, intervening-if checks and optional entry. Zone-source event infrastructure can be reused downstream, but this candidate proves only one immediate name. | High for event lifecycle, order checks and native entry tests. |

Candidate #1 wins as the ranked multi-card prerequisite. Investigation found that
`targetDeclaration` stores no choices and announcement `Possible` queries do not
carry earlier groups' choices. Completing dependent targets would also require
feasibility queries, player/opponent/random selection, trigger placement, payment
rechecks and copy retargeting to share that frame. These are explicit missing
engine contracts, not adapters that can safely capture mutable target state.

This run therefore selects one complete supported context: resolving permanent
collections, including delayed bodies. The reuse boundary is exact permanent
information plus typed set selection in mage; spelling, arity and local bindings
stay in ron. Acceptance cases were stated before edits: current and LKI types,
identity across return/phasing, declaration rejection, payment rollback,
target legality, copied/cloned/delayed frames and pure queries. Announcement
predicates are deferred as a separate prerequisite, not partly accepted.

### Exact signatures, contexts and engine reuse

New engine API:

```go
PermanentsSharingCardTypes(reference PermanentRef, subjects PermanentSet,
    types ...core.CardType) PermanentSet
```

No supplied types permits all engine card types. A supplied list restricts the
intersection and is copied at construction. Definition checks reject invalid
or duplicate types, missing operands, incompatible information references and
undeclared target/choice reads. The reference uses the existing
`readPermanentInformation` current/phase/departure service. Subjects must remain
present, unphased, exact visits. Input order is retained and duplicates are removed.
Layered and face-down types are read through `Permanent.HasType` and exact LKI.
An absent reference with no retained information selects nothing. Control is
independent of type comparison; other set operands can restrict it.

RON resolving collections accept exactly:

```text
SharesACardtypeWithPermanent(permanentRef)
SharesACardtypeWithPermanentFromAmongCardtypes(permanentRef, [cardTypes...])
```

The restricted list must be nonempty and distinct. It uses existing RON type
symbols: Artifact, Creature, Enchantment, Instant, Land and Sorcery. Unrestricted
comparison can match all engine types. Typed singular source, Aura host, target,
event and prior-choice references retain their normal scope and declaration
checks. These forms compose with resolving And/Or collections and lower to the
new service over `MatchingPermanents(AnyPermanent)`. Entry and draw-replacement
bodies reject them. Target filters and static declarations retain their errors.

The shared information service, resolution/target frames, current attachments,
permanent choices, set intersections/unions, action batches, payment transaction,
target legality, copying and game cloning supply execution. Waiting delayed
source traversal now sees both reference and subject-set operands without
reading types or running filters. No mutable declarations, global results,
card-name branches, fallback registry, replacement system or zone system were
introduced. Existing engine and Go-card callers keep their signatures.

### Failing-first evidence and gameplay mappings

Before the engine implementation, `TestPermanentTypeComparison*` failed to
compile solely because `PermanentsSharingCardTypes` did not exist. Initial
test-only helper mistakes were corrected before confirming that expected
failure. Before the RON adapter, focused gameplay failed on unsupported
`SharesACardtypeWithPermanent` and the restricted form's old one-argument
filter rejection. Both expected failures were observed before their respective
production implementations. Test fixture mistakes about type attributes and an
Aura activation index were corrected; no gameplay requirement was removed.

- Native `TestPermanentTypeComparison`: unrestricted/restricted intersection,
  no types, empty intersections, layer 4 changes, departure LKI, returned source,
  phase/phase cycle, missing identity, phased members, control, clones and RNG purity.
- Native `TestPermanentTypeComparisonDeclarations`: nil operands, invalid and
  duplicate types, undeclared targets/choices, valid typed target and independent
  constructor list storage.
- Native `TestPermanentTypeComparisonSavedAndWaiting`: source, target, saved
  and attachment references; departure/return/phase/face-down information,
  clone execution and pure waiting queries for either explicit source operand.
- `TestRONPermanentTypeComparisonResolution`: both complete synthetic RON
  forms; real mana payment and rollback, resolution timing, current types and
  control, late entry, source LKI/return/phasing, member phasing, copies and clones.
- `TestRONPermanentTypeComparisonTargetsAndHost`: typed target and Aura-host
  bindings through real RON activation and shared tapping.
- `TestRONPermanentTypeComparisonTargetLegality`: illegal target information
  cannot select subjects; partial legality still runs independent life gain;
  all-illegal spells do not resolve. Covers shroud, return and phasing.
- `TestRONPermanentTypeComparisonDelayedChoice`: preceding lexical choice,
  future end-step execution, source departure, returned/phased choice LKI,
  copied activations, cloned delayed frames and waiting-query purity.
- **Gauntlets of Chaos → TestRONPermanentTypeComparisonRepositoryExpressions**:
  executes its exact nested restricted comparison with its numbered target
  binding in an explicit fixture. Complete unchanged record still rejects.
- **Enchantment Alteration → TestRONPermanentTypeComparisonRepositoryExpressions**:
  executes its exact nested unrestricted host comparison in an Aura fixture.
  This proves the information prerequisite, not a target-derived spell host or
  attachment movement. Complete unchanged record still rejects.
- `TestRONPermanentTypeComparisonInvalid`: malformed arity/kinds/type lists,
  duplicate types, unbound and incompatible references and continued complete-card
  rejection. Repository-expression tests also reject entry/replacement contexts.

These nine top-level tests add contract/gameplay evidence for this resolving
capability. The two repository-expression mappings are partial prerequisites;
they do not establish full-card gameplay coverage or increase accepted names.

### Rules, source defects and removal boundaries

Local authoritative data/LEG.json confirms the exact Oracle requirements.
CR 400.7 and 608.2h support exact-visit/current/LKI reads; 608.2b controls illegal
target information; 702.26b and existing information contracts keep phased
subjects separate from information references. CR 601.2c supports the deferred
announcement context's need for prior target choices.

Gauntlets' existing attachment collection still lacks Oracle's Aura restriction;
it could include Equipment. The full record's exchange-result condition remains
unsupported. Alteration's HostPermanent is not currently bound in its instant
spell context; a target-derived host service must define that read before its
attachment instruction can lower. These remain explicit source/engine boundaries.
No RON source was changed, so no vocabulary refresh is needed. Gaea's Liege's
duplicate attack predicate and other recorded source defects remain open.

API/reference documentation records only this resolving contract. No Go card
definition, generator or DSL function is removed: the full-collection, startup,
gameplay and caller-removal gates remain open. There is no new partial production
registry. No environment failure was observed.

### Validation and measured gain

Required verification passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONPermanentTypeComparison -count=1` passed.
- `go test ./pkg/mage -run TestPermanentTypeComparison -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's additional `go test ./... -short -count=1` passed.
- `make lint` passed with **0 issues**. Formatter/modernizer output and the
  final worktree were reviewed; no unrelated changes or semantic edits appeared.
  Post-lint focused native and RON tests passed again.
- `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited **1**, as required while remaining
  records reject. JSON comparison with the fresh baseline is exactly equal:
  every count, position and diagnostic is unchanged.

Estimated and actual full-card gains are **zero names / zero records**. The
selected resolving prerequisite adds gameplay/contract evidence but does not
wire the unsupported parents in the two inventory records. Final counts remain
**854 accepted / 132 rejected names**, **2,196 compiled / 249 rejected records**,
out of 986 names / 2,445 records. Gauntlets still reports LEG:95:416, expected one
positional filter argument for the restricted dependent target comparison.
Alteration still reports LEG:75:286, unsupported AttachPermanentToAPermanent.
There are no new diagnostics, collection regressions or validation blockers.

Changed files: `pkg/mage/permanent_type_comparison.go` and its native test;
`pkg/mage/dsl/ron/permanent_type_comparison.go` and its RON test;
`pkg/mage/dsl/ron/contextual_collection.go`, `pkg/mage/waiting_delayed_source.go`,
`pkg/mage/doc.go`, `docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and
this ledger. Remaining gaps are the 132 names / 249 records, complete collection
gameplay, embedded loading, application cutover, Go-card deletion and DSL cleanup.

### Ranked next candidates

1. **Dependent target context and target-derived host references**: Gauntlets of
   Chaos and Enchantment Alteration. The resolving comparison service is ready.
   Select a bounded complete announcement/reference prerequisite with all
   acquisition, legality and copy paths. Full cards still need exchange results,
   Aura-only correction or attachment movement; zero immediate estimated unlocks.
2. **Restricted X announcement mana payment**: Drain Life. Zero gain until its
   clamped result consumer also works; eventual one name / six records. Compare
   other complete restricted-payment records before selecting this prerequisite.
3. **Graveyard-source trigger arming**: Nether Shadow. Exact source visits,
   graveyard order, intervening-if and optional entry remain; eventual one name /
   six records. Keep event arming separate from incomplete source-zone execution.

This run stops after resolving permanent card-type comparison.


## 2026-10-13 — TARGET-DERIVED-HOST-REFERENCE

### Fresh evidence and candidate selection

The starting tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The baseline strict loader exited
1 with 854 accepted / 132 rejected names and 2,196 compiled / 249 rejected
records. Collection rejection was not a build or test failure.

The latest ranking was evaluated first. Complete records were read for
Gauntlets of Chaos, Enchantment Alteration, Drain Life and Nether Shadow;
reprints count once. Targeted vocabulary signatures confirm one unrestricted
host comparison, one restricted dependent-target comparison, six restricted-X
records and six graveyard-source records.

| Candidate | Distinct affected names | Estimated full-card gain | Other blockers and downstream work | Effort |
| --- | --- | --- | --- | --- |
| Dependent targets / target-derived host (rank 1) | Gauntlets of Chaos; Enchantment Alteration | 0 names / 0 records | Gauntlets needs announcement frames, exchange outcomes and Aura-only data correction. Alteration needs attachment movement. Resolving comparisons are ready. | Medium to large for full announcement acquisition; small for the bounded exact-host reference prerequisite. |
| Restricted X announcement mana (rank 2) | Drain Life | 0 now; eventual 1 name / 6 records | Color-restricted X payment and clamped actual-damage consumer remain. Existing damage outcomes are ready. No other complete source record uses this restriction. | Medium; mana planning, announcement legality and rollback evidence. |
| Graveyard-source arming (rank 3) | Nether Shadow | 0 now; eventual 1 name / 6 records | Exact graveyard visits, ordered creature count, intervening-if, optional entry and source-zone lifecycle remain. | Large for complete execution; medium for arming alone. |

Select the bounded target-derived host prerequisite from rank 1. This is
required before Alteration's attachment adapter can use its exact comparison.
The other half of rank 1, sequential dependent-target announcement, is a
separate capability with broader acquisition and copy paths. This run does not
claim to complete either card or that announcement capability. The single-card
prerequisite is selected because rank 1 explicitly calls for a bounded complete
reference prerequisite and the two alternatives also have zero immediate gain
with larger contracts. No useful multi-card full-card gain was established for
these three candidates.

Before editing, the contract was resolving HostPermanent in a spell with one
explicit singular Aura target, with exact binding, current attachment reads,
legality, LKI, phasing, copies, cloning and declaration rejection. RON owns
spelling and scope; the engine owns exact attachment and host information.

### Exact API, contexts and reuse boundary

New reusable API: `HostOfPermanent(attachment PermanentRef) PermanentRef`.
It accepts information-capable typed references and retains no game state.
Definition validation follows its attachment operand and rejects nil,
undeclared target and unbound saved references. Existing `AttachedPermanent`
shares its information identity implementation; its action behavior is retained.

RON accepts the symbol `HostPermanent` in resolving
`SpellActions(Targeted([TargetPermanent(filter)], body))` on a nonpermanent
spell when `filter` requires `IsEnchantmentType(Aura)` directly or through
nested conjunctions. Each fresh factory builds the target and host consumer
with the same instance binding. Existing Aura source scope retains precedence.
Disjunctive Aura alternatives, ambiguous numbered groups, untyped/plural targets,
activated and triggered declarations, entry and replacement contexts do not
infer a target host. Tuple spelling of HostPermanent remains invalid.

A present attachment reads its current host. Detachment clears the read.
Departure saves the attachment's last host. Returned attachments and hosts
cannot replace saved visits. Information consumers can read exact host LKI;
action and damage-recipient consumers require a present, unphased host.
Shared legal target bindings prevent reads from illegal targets. An all-illegal
spell does not execute independent instructions. Earlier resolving departure
and delayed bodies can retain a previously legal target's attachment facts.

Reuse: `permanentInformationReference`, exact incarnation reads,
`departedAttachments`, phase/departure information, legal target frames,
per-instance bindings, delayed capture, copy retargeting, cloning and shared
cost transactions. Waiting comparison traversal follows the attachment operand
without evaluating filters or changing state. No attachment action, zone system,
replacement pipeline, global action slot or partial registry was introduced.

### Failing-first evidence and test mappings

Before production changes, `TestPermanentHost*` failed to compile because
HostOfPermanent did not exist. An initial fixture used a nonexistent detach
method; that was corrected before confirming the constructor-only failure.
The fixture attachment was corrected to Equipment so shared attachment legality
applies. Before the RON adapter, every `TestRONTargetHostGameplay` scenario
failed with an unbound HostPermanent. No acceptance condition was removed.

Native current-Equipment host-departure expectations were corrected after
observing the shared detachment behavior: a present detached attachment has no
host. Separate departure-before-host-departure cases establish retained host
LKI, including both returned visits. A repository AST helper typo was corrected
without changing the required comparison.

- `TestPermanentHost`: current/moved/detached attachments, attachment departure
  and return, host departure and return, combined departure/return LKI, phasing,
  phase cycles, unknown identities, clone reads and RNG purity.
- `TestPermanentHostDeclarations`: nil, undeclared and unbound operands reject;
  a declared singular target validates.
- `TestPermanentHostReferencesAndWaiting`: target/source/saved operands and
  pure waiting comparison traversal without filter execution.
- `TestRONTargetHostGameplay`: complete synthetic RON spells, real casting and
  mana payment, invalid/unpaid rollback, resolution timing, current attachment,
  shroud, returned targets, target/host phasing, phase cycles, copies, retargeting
  and cloned pending spells.
- `TestRONTargetHostInstances`: two fresh factory instances keep their own
  target hosts through separate resolutions.
- `TestRONTargetHostDepartureAndDelay`: legal target departure during resolution,
  delayed exact attachment/host LKI, returned visits and cloned saved frames.
- **Enchantment Alteration → TestRONTargetHostRepositoryComparison**: its exact
  unchanged nested comparison runs through a typed target host in an explicit
  fixture. Complete record rejection on AttachPermanentToAPermanent is retained.
  This is partial prerequisite evidence, not full-card coverage.
- `TestRONTargetHostInvalid`: absent/ambiguous/non-Aura hosts, disjunctions,
  symbol/tuple mismatch, activated/triggered and entry/replacement boundaries.
- Existing `TestPermanentInformation*` regressions preserve source attachment
  information after implementation reuse.

### Rules, data and removal boundaries

Local authoritative `data/LEG.json` requires Alteration to attach a target Aura
attached to a creature or land to another permanent of that type. CR 303.4b/e,
400.7, 608.2b/h and the existing phasing contract support the exact-host read.
The future attachment adapter must independently enforce another permanent,
matching types and attachment legality. This slice makes no source correction.
Gauntlets' missing Aura-only destruction restriction and its exchange-result
blocker remain. No RON source changed, so no vocabulary refresh is needed.

No Go cards, generator or DSL functions are removed. Complete collection
execution, gameplay, native startup, cutover and caller-audit gates remain open.
API and rules-contract docs describe only the new reference behavior.

### Validation and measured gain

Required checks passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONTargetHost -count=1` passed.
- `go test ./pkg/mage -run 'TestPermanentHost|TestPermanentInformation' -count=1`
  passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's additional `go test ./... -short -count=1` passed.
- Initial `make lint` rejected a test-local variable named copy because it
  shadowed a builtin. It was renamed to copied; affected focused native and
  RON tests passed again. Final `make lint` passed with **0 issues**.
  Automatic formatting was reviewed; only new-file formatting changed and
  there were no unrelated edits or semantic modernizer changes.
- `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected. Its complete
  JSON equals the fresh baseline, including every issue, position and count.

Estimated and actual full-card gain: **zero names / zero records**. Eight new
top-level native/RON tests establish reference and gameplay contracts, including
one partial inventory-card mapping. Full-card gameplay coverage does not
increase. Final counts remain **854 accepted / 132 rejected names**,
**2,196 compiled / 249 rejected records** out of 986 names / 2,445 records.
Alteration still reports LEG:75:286, unsupported AttachPermanentToAPermanent;
Gauntlets still reports LEG:95:416, expected one positional filter argument.
No accepted record regressed. No environment or pre-existing check failure
blocked this slice.

Changed files: `pkg/mage/permanent_host.go` and its native test;
`pkg/mage/dsl/ron/target_host.go` and its RON test; `compiler.go`,
`permanent_scope.go`, `composable_information.go`, `waiting_delayed_source.go`,
`pkg/mage/doc.go`, `docs/dsl-reference.md`, `docs/engine-rules-contracts.md`
and this ledger. No RON, Go-card, generator or legacy DSL removal is included.

### Ranked next candidates

1. **Shared attachment movement — Enchantment Alteration and Kudzu.** The
   vocabulary and all seven complete records show the same two-argument
   AttachPermanentToAPermanent action. Alteration now has its host comparison
   prerequisite. Estimate one immediate full-card name / one record, contingent
   on exact required choice, another-permanent and attachment legality evidence.
   Kudzu still needs PlayerMayAction and its saved former-host controller. This
   shared action can enable two unique cards; medium implementation/test effort.
2. **Dependent target announcement — Gauntlets of Chaos.** Lower its restricted
   comparison against the preceding numbered target with complete acquisition,
   query, resolution legality and copy contracts. Zero immediate gain until
   exchange outcomes and the Aura-only source correction finish; medium to large
   effort. Keep it separate from this completed resolving-host reference.
3. **Restricted X announcement mana — Drain Life.** Zero immediate gain until
   clamped actual-damage consumption also works; eventual one name / six records.
   Medium effort. Nether Shadow remains a larger source-zone capability with
   additional graveyard-order, intervening-if and optional-entry contracts.

This run stops after target-derived host references.


## 2026-10-13 — REQUIRED-ATTACHMENT-MOVEMENT

### Candidate comparison and selection

The working tree was clean. Fresh `make build` and the complete RON package
passed before editing. The baseline loader exited 1 for unsupported collection
records: 854 accepted / 132 rejected names, 2,196 compiled / 249 rejected records.
Collection rejection was separate from build/test failure.

The latest entry's candidate #1 was dependency-ready after the exact-host slice.
All seven complete attachment records were read, along with Gauntlets of Chaos
and all six Drain Life records. Targeted vocabulary inspection confirmed the
unchanged two-argument action. Reprints count once in this comparison.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream benefit | Effort |
| --- | --- | --- | --- | --- |
| Required resolving attachment movement | Enchantment Alteration; Kudzu | 1 name / 1 record | Alteration's nested Aura target, host reference and shared-type collection already lower. Kudzu still needs PlayerMayAction and exact former-host controller/timing evidence. Shared movement serves both cards, seven records, Aura/Equipment/Fortification actions and delayed/scoped consumers. | Medium: one reusable engine action, AST adapter, native and RON gameplay tests |
| Dependent target announcement | Gauntlets of Chaos | 0 now; eventual 1 name / 1 record | Its complete second target depends on Ref_TargetPermanent1 during acquisition and legality. Exchange outcomes and the missing Aura-only attachment restriction remain. Enables constrained later-target declarations beyond resolving comparisons. | Medium/large: target acquisition/query/copy integration and legality tests |
| Restricted X announcement payment | Drain Life | 0 now; eventual 1 name / 6 records | Complete CastEffect restricts X to black mana; nested GainLife still needs clamped actual damage. Enables the restricted-X solver contract for subsequent damage consumers. | Medium: announcement solver metadata, transactional payments and result tests |

Selection: candidate #1 offers the largest credible immediate gain and a concrete
second-card dependency. No engine or source prerequisite blocks its resolving
shape. The source records remain unchanged. No old composable migration plan,
Go card requirements, removed skills or Go stubs were used.

### Established contract and reuse boundary

Before editing, the declared acceptance contract was a required, non-targeted
host choice at resolution with another-permanent and legal attachment checks.
The tests cover malformed declarations, current and returned visits, phasing,
protection, enchant restrictions, shroud/hexproof, copies, clones, real payment
and resolution timing. Scoped/delayed consumers and timestamps complete that
same action contract.

Engine signature:
`AttachPermanentToChoice(player PlayerRef, attachment PermanentRef, hosts PermanentSet) ResolvingEffect`.
It saves the attachment visit, uses the shared `ChoosePermanent` service with a
typed legal-host set, rechecks both visits, membership and legality, then calls
`Game.Attach`. A private per-instance host binding uses existing resolution maps.
No legal host makes no move; nil/invalid answers use required-choice fallback.
Both objects must be present and unphased. Return cannot substitute either visit;
a complete phase cycle preserves identity. The Aura's controller determines its
enchant limits; the explicit player determines who chooses. Movement keeps
ownership/control, removes the former host link, adds the new link, applies
static effects, updates attachment timestamps and uses the shared attach event.

RON signature:
`AttachPermanentToAPermanent(attachmentRef, hostCollection)` with exactly two
positional arguments. The first uses `compilePermanentRef`; the second uses
`compileContextualCollection`, including nested shared-card-type comparisons.
Ordinary resolving sequences, direct `PlayerActions` and delayed action bodies
are supported. Entry/draw-replacement bodies, incompatible/unbound operands,
malformed lists and escaped `TheChosenPermanent` results reject during compile.
Waiting queries retain explicit typed source reads without choices or filters.

Dependencies/reuse: exact source/target/choice visits, `HostOfPermanent`,
`PermanentsSharingCardTypes`, shared collection composition, legal target frames,
`ChoosePermanent`, enchant controller checks, protection, `Game.canAttach`,
`Game.Attach`, attachment retimestamps, delayed capture and copied/cloned frames.
RON owns spelling, shape and scope. Mage owns decisions, identity, legality,
links and layers. No parallel zone/layer/replacement service, game-time declaration
panic, global action result, card-name branch or partial registry was introduced.

### Failing-first evidence and card-to-test mappings

The native tests first failed to compile for the missing engine signature.
Fixture-only references to core enums were corrected, and the expected
constructor-only failure was confirmed before engine implementation. Every real
Alteration gameplay scenario then failed with unsupported AttachPermanentToAPermanent
before AST lowering. The direct scoped-player test separately failed with
unsupported scoped player action before adding its dispatch entry.

Initial gameplay fixtures needed a non-creature-only Aura for land movement.
Native creature-Aura tests use the actual type layer, rather than modifying a
printed card after entry. A delayed chooser fixture was corrected to give the
Aura to its intended chooser. No acceptance behavior was removed.

- **Enchantment Alteration → TestRONAttachmentMoveAlteration**: complete unchanged
  LEG record, real casting/payment, invalid and unpaid rollback, resolution-time
  choice, creature/land/creature-land matching, no destination, current Aura
  enchant-control limits, protection, shroud/hexproof hosts, illegal shrouded
  target, returned target, source/host phasing, phase cycles, changed attachment,
  copied spells, independently retargeted host comparisons and cloned games.
- **Kudzu → TestRONAttachmentMoveKudzuPrerequisite**: exact unchanged nested
  movement action in LEA/LEB/2ED/3ED/CED/CEI. Each complete record still rejects
  PlayerMayAction. This is partial prerequisite evidence, not full-card coverage.
- `TestAttachmentMove`: required choice, nil/invalid fallback, legal candidates,
  control/ownership and both links, protection/enchant limits, creature Aura,
  old/returned/phased source and host visits, phase cycles, and visit/legality
  changes during the decision.
- `TestAttachmentMoveDeclarations`: nil player/reference/set and unbound
  references reject; valid source declaration passes engine validation.
- `TestAttachmentMoveKindsAndTimestamp`: Equipment, Fortification, ordinary
  nonattachment, Aura static subject changes and shared timestamp order.
- `TestAttachmentMoveWaiting`: explicit source retention, no filter/choice/state
  execution and RNG purity during waiting queries.
- `TestRONAttachmentMoveScopesAndDelay`: two fresh factory instances, scoped
  saved chooser, delayed exact targets, returned/phased visits, phase cycles
  and independent cloned resolution frames.
- `TestRONAttachmentMovePlayerScope`: direct explicit chooser differs from the
  spell controller and performs the required choice at resolution.
- `TestRONAttachmentMoveInvalid`: arity/kind, incompatible and unbound operands,
  empty filter conjunction, entry/replacement boundary and private result escape.
- Existing `TestRONTargetHostRepositoryComparison` now asserts complete record
  compilation and retains its expression gameplay check. Type-comparison invalid
  tests retain Gauntlets' blocker; Alteration now has complete acceptance tests.
  Four inventory/definition count expectations were advanced by exactly one record.

### Rules, source and removal boundaries

Local authoritative `data/LEG.json` states: “Attach target Aura attached to a
creature or land to another permanent of that type.” CR 303.4e/j and 701.3a–c
support controller separation, illegal-attachment no-op and timestamps. Existing
exact visit, target legality and phasing services retain their contracts.

No source defect was found in the selected action and no RON source changed.
Vocabulary source hashes and syntax remain unchanged, so no snapshot refresh is
needed. Gauntlets' Aura-only collection defect stays explicit. Kudzu's optional
former-host player and destruction/movement lifecycle remain separate blockers.
No environment failure blocked implementation. No Go card, generator or legacy
DSL function was removed: full collection, cutover and removal gates remain open.
API and rules-contract docs were updated only for established movement behavior
and obsolete Alteration blocker claims.

### Verification and measured result

Required checks passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONAttachmentMove -count=1` passed.
- Native `go test ./pkg/mage -run 'TestAttachmentMove|TestMayAttach|TestPermanentHost' -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's additional `go test ./... -short -count=1` passed.
- Initial `make lint` found one prealloc issue in a two-instance test fixture.
  Its slice now reserves capacity two. Native focused tests and RON focused
  tests, including target-host and type-comparison regressions, passed again.
  Final `make lint` passed with **0 issues**. Automatic formatting changed only
  import grouping in new test files; no unrelated edits or semantic modernizer
  changes occurred. The complete slice diff was reviewed.
- `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected for the remaining
  unsupported collection. Its complete JSON equals the post-implementation
  measurement. Comparing with the fresh baseline removes only Alteration's
  LEG:75:286 attachment-action diagnostic; no issue was added or changed.

Estimated and actual gains match: **one full-card name / one record**. Final
counts are **855 accepted / 131 rejected names**, **2,197 compiled / 248 rejected
records** out of 986 names / 2,445 records. Full-card gameplay coverage increases
by one unique card, Enchantment Alteration. Kudzu's six movement-expression tests
establish a prerequisite only. Every previously accepted record remains accepted;
the accepted-definition rebuild audit and full suite pass. The strict loader
continues to reject the collection without publishing a partial registry.

Representative remaining diagnostics: Kudzu at 2ED:137:306 rejects PlayerMayAction;
Gauntlets at LEG:95:416 requires one positional filter argument; Drain Life at
2ED:68:163 rejects SpendOnlyColorManaOnX. No environmental or pre-existing check
failure remains. Runtime loading, complete gameplay, Go-card deletion and unused
DSL removal are still incomplete.

Changed files: new `pkg/mage/attachment_move.go` and native tests; new
`pkg/mage/dsl/ron/attachment_move.go` and RON tests; RON `compiler.go` and
`player_actions.go`; definition, inventory and numeric-consumer count fixtures;
prior type-comparison and target-host tests; `pkg/mage/doc.go`,
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger.
No RON source, Go card, generator or legacy DSL removal is included.

### Ranked next candidates

1. **Dependent target announcement context — Gauntlets of Chaos.** Earlier target
   bindings must be available to later target acquisition, legality, copies and
   pure queries. Zero immediate unlocks until exchange outcomes and the Aura-only
   source correction finish; medium/large effort. Reassess shared target value.
2. **Restricted X announcement payment — Drain Life.** Zero immediate unlocks
   until clamped damage-result consumption also works; eventual one name / six
   records. Medium effort. Source-zone Nether Shadow remains a larger procedure.
3. **Explicit optional player action and former-host controller — Kudzu.** The
   exact movement expression is ready. One name / six records is contingent on
   optional-choice ownership, retained former-host facts and destruction-to-move
   timing. Medium effort; defer single-card completion while shared work remains.

This run stops after required attachment movement.


## 2026-10-13 — DEPENDENT-PERMANENT-TARGET-COMPARISON

### Selection and fresh baseline

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The baseline loader exited 1 for
collection rejection: 2,197 compiled / 248 rejected records, 855 accepted /
131 rejected names. This was not a build or test failure.

The latest ranked candidates were evaluated from complete canonical RON records,
including nested bodies. Reprints count once. None supplies immediate multi-card
completion; the first-ranked candidate is a dependency-ready unblocking prerequisite.

| Candidate | Distinct affected names; estimated full-card gain | Other blockers and downstream value | Full implementation/test effort |
| --- | --- | --- | --- |
| Dependent target context | Gauntlets of Chaos; 0 names / 0 records | Exchange-result consumer and Aura-only source defect remain. Enables its exact dependent card-type target and later exchange-result continuation. Query frames are shared by spells, activations, triggers, modes and copies; this is not counted as multi-card gameplay coverage. | Medium/large: declaration order, pure complete-selection queries, acquisition paths, resolution visits and copy retargeting. |
| Restricted X casting payment | Drain Life; 0 immediate, eventual 1 name / 6 records | Clamped actual-damage consumer remains. Enables exact black-only X announcement before damage/life execution. | Medium: total-cost and mana planner integration, rollback, copies and X results. |
| Optional explicit player action | Kudzu; estimated 1 name / 6 records if fully completed | Must retain former-host controller and destruction-to-movement timing. Required movement is ready; optional ownership and host facts are not. Enables its optional resolving reattachment. | Medium: explicit decision player, retained host identity, destruction and Aura lifecycle tests. |

Selected the first-ranked unblocking prerequisite. Reuse boundaries were declared
before editing: RON owns spelling, arity and earlier-target scope; mage owns query
frames, card-type comparison, acquisition and legality. No full-card gain was
promised. The implementation stops at this one target capability.

### Supported signatures, dependencies and engine reuse

- `SharesACardtypeWithPermanent(Ref_TargetPermanentN)` and
  `SharesACardtypeWithPermanentFromAmongCardtypes(Ref_TargetPermanentN, [types])`
  in later singular permanent target filters and supported `And` conjunctions.
  The reference must name an earlier declaration in the same target list.
  The restricted list is nonempty, valid and distinct. Self/forward/unbound,
  source/host/choice/player/plural references and malformed shapes reject.
- `TargetSharesCardTypeWith(reference *PermanentTargetBinding, types
  ...core.CardType) PermanentTargetOption` copies its type list. Engine validation
  requires earlier singular permanent declarations throughout a dependent list.
  Omitted engine types permit every card type.
- `Game.PossibleTargetGroup(controller uuid.UUID, card Card, specs []Target,
  group int, chosen []uuid.UUID, x int) []uuid.UUID` keeps exact prior choices in
  a separate query frame. Invalid prefixes return no candidates. Every candidate
  permits a complete remaining selection. Queries make no decisions or payments,
  consume no randomness and leave live X/bindings unchanged.
- Explicit announcements, prompted/forced/opponent choices, spell/activation
  availability, trigger placement, modal legality and copied target choices use
  shared services. Resolution reads saved visits and current types before effects.
  Phasing/return cannot supply a present reference; a phase cycle keeps identity.
  Target relations can read a present earlier slot that became shrouded, while
  that illegal slot remains unavailable to resolving effects. No relation LKI
  restores an absent target. Existing source LKI and protection remain separate.
- Both resolving collections and target predicates use `sharesPermanentCardType`.
  Collections retain their prior LKI contract. Factory builds remain fresh and
  ordered; frames, stack copies and clones keep independent captured choices.
  Existing costs, APNAP, replacement, control-exchange and effect services remain.
  Optional/plural group boundaries retain their prior validation.
- Copy retargeting now captures explicitly reselected battlefield visits, including
  a returned card with the same physical ID, before querying later groups. An
  unchanged illegal target can remain under CR 707.10c; cancellation keeps old
  captures. The original stack object cannot inherit a copy's new visit.

### Failing-first evidence and card-to-test mappings

The native test first failed to compile for the missing target option. Two
fixture API names were corrected, then the constructor-only failure was confirmed.
RON gameplay separately failed at the restricted comparison's unsupported arity.
`TestTargetDependencyCopyReturnedChoice` later failed on stale copy incarnation
capture before the battlefield rechoice fix. No assertion was weakened.

- **Gauntlets of Chaos → TestRONTargetDependencyGauntletsPrerequisite**: unchanged
  complete LEG record still rejects its exchange-result consumer. A separate
  test-only AST prefix retains its exact targets, `ExchangeControl`, {5} payment
  and source sacrifice. Tests cover invalid/unpaid rollback, live type additions,
  illegal/returned/phased targets, phase cycles, source return, control changes,
  copied pairs, legal copied candidates and cloned control state. This is partial
  prerequisite evidence, not a complete-card mapping or production fallback.
- `TestRONTargetDependencyGameplay`, `Invalid`, `Scopes` and
  `InstancesAndUnrestricted`: spell/activation gameplay, declaration errors,
  trigger/modal/opponent acquisition, both spellings and independent factories.
- `TestTargetDependency`, `Declarations`, `Query`, `CopyAndPrompt`,
  `CopyReturnedChoice` and `Triggers`: exact visits, type loss, partial legality,
  source departure, phase cycles, copies/clones, declaration order/type errors,
  copied type lists, complete-selection pruning, pure queries and forced choices.
- Existing `TestRONSingularGroupsStrictLoadAndConsumers` advances only Gauntlets'
  expected blocker. Existing target-group, type-comparison and accepted-definition
  tests remain. No compilation count fixture needs a changed expectation.

Broader tests caught two optional-group boundary regressions during integration;
restoring per-group validation preserved duplicate choices across separate groups
and rejected excess choices within one group. The engine short suite also caught
an inspection panic on an existing nil-target validation fixture. Nil checks now
precede unwrapping. The original regression assertions pass unchanged.

### Rules, source, validation and measured result

Local `data/LEG.json` supplies Gauntlets' complete Oracle text. CR 601.2c,
602.2b, 603.3c-d, 608.2b, 400.7, 702.26b and 707.10c supply announcement,
placement, legality, identity, phasing and copy contracts. The known RON defect
remains: Gauntlets' attachment collection omits the Aura-only restriction.
No RON source changed; vocabulary spelling and source hashes are unchanged.
No Go card, generator or legacy DSL function was removed. Runtime cutover and
removal gates remain open. API/rules docs describe only the established service.

Required checks passed in fast-fail order after the integration fixes:

- `go test ./pkg/mage/dsl/ron -run TestRONTargetDependency -count=1` passed.
- Native `go test ./pkg/mage -run TestTargetDependency -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's `go test ./... -short -count=1` also passed.
- `make lint` passed with **0 issues**. Automatic changes only regrouped imports
  in the new RON test file. The tracked diff and final new files were reviewed;
  no unrelated change or semantic modernizer edit occurred. Native/RON focused
  tests and target-group/type-comparison/nil-declaration regressions passed again.
- `git diff --check` passed. Final `go run ./cmd/loadcards -json` exited **1**,
  as expected. No rejected record was added or removed. Only Gauntlets' diagnostic
  changed: LEG:95:416 rejected the comparison; LEG:95:594 now rejects
  `PermanentsChangedControlThisWay`. Every other diagnostic is unchanged.
  Drain Life still rejects `SpendOnlyColorManaOnX`; Kudzu rejects `PlayerMayAction`.

Actual gains match the zero-name / zero-record estimate. Complete-card gameplay
coverage is unchanged; Gauntlets gains target prerequisite evidence only. Final
counts remain 855 accepted / 131 rejected names and 2,197 compiled / 248 rejected
records out of 986 names / 2,445 records. The strict loader rejects the collection
and publishes no partial registry. No environmental or pre-existing failure
blocks this slice.

Changed files: new mage and RON `target_dependency.go` and test files; mage
`activated.go`, `composable_target.go`, `game.go`, `permanent_type_comparison.go`,
`resolution_state.go`, `spell_cast_stack.go`, `spell_copy.go`, `stack_resolver.go`,
`target.go`; RON `player_relative_target.go`, `singular_target_groups_test.go`;
`pkg/mage/doc.go`, `docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and
this ledger. No RON record changed.

### Ranked next candidates

1. **Announced X in entry operations — Rock Hydra, Frankenstein's Monster.**
   Reassess the shared entry-X prerequisite using these complete records. Zero
   immediate estimated unlocks: Hydra still needs finite source prevention;
   Monster still needs entry exile-payment/results and counter distribution.
   Concrete downstream value is two names; medium prerequisite effort.
2. **Exchange outcomes and Aura-only source correction — Gauntlets of Chaos.**
   Its target and exchange prefix is ready. Save a local successful exchange
   result, gate later destruction and prove the source correction from Oracle.
   Estimated one name / one record; small/medium, single-card completion.
3. **Restricted X casting payment — Drain Life.** Its clamped actual-damage
   consumer remains a separate dependency. Zero immediate gain, eventual one
   name / six records; medium effort. Kudzu's optional former-host action remains
   another single-card completion candidate.

This run stops after dependent singular permanent target comparisons.

## 2026-10-13 — INTEGER-COUNTER-CAPACITY

### Selection, dependencies and contract

The working tree was clean. Fresh `make build` and uncached RON package tests
passed. The baseline loader exited 1 for collection rejection: 2,197 compiled /
248 rejected records, 855 accepted / 131 rejected unique names. It did not fail
because of a build or environment problem.

The latest ranked candidates were compared from complete RON records:

| Candidate | Distinct names | Estimated full-card unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Announced X in entry operations | Rock Hydra; Frankenstein's Monster | 0 names / 0 records | Hydra needs finite prevention. Monster needs entry exile payment, payment results and counter distribution. Shared entry X enables both names, but native counter storage truncates at 256. | Medium, with a required storage prerequisite |
| Exchange outcomes | Gauntlets of Chaos | 1 name / 1 record after source correction | Local successful-exchange result and Aura-only attachment source correction; dependent targets and exchange prefix are ready. Enables full exchange-result consumers. | Small/medium |
| Restricted X casting payment | Drain Life | 0 names / 0 records | Clamped actual-damage consumer remains; enables one name / six records downstream. | Medium |

Selected **integer counter capacity**, the missing shared prerequisite for the
first candidate. CR 107.1 and 122 impose no byte-size counter bound. A complete
entry-X adapter cannot use wrapping byte storage for legal announced amounts or
replacement-expanded counts. This prerequisite enables Rock Hydra and
Frankenstein's Monster and repairs counter behavior for current accepted RON
cards. It has concrete multi-card value despite zero immediate compilation gains.
It is one storage/placement capability, not entry-X lowering or a second slice.

Supported RON signature remains
`AsPermanentEnters(ThisPermanent, [EntersWithNumberCounters(Integer(n), counter)])`
on permanent cards, with one operation, a validated permanent counter type and
a nonnegative literal in the engine `int` range. Amounts above 255 now lower to
`WithAbility(EntersWithNCounters(counter, n))`. Wrong arity, negative/noninteger
amounts, incompatible counters, other sources and resolving-action contexts
remain errors. `ValueX` and entry payment/result forms remain errors.

`Permanent.Counters [NumCounters]int`, exact-visit
`ZoneSystem.departedCounters map[uuid.UUID][core.NumCounters]int` and
`PermanentState.RawCounters [core.NumCounters]int` replace byte arrays.
`EntersWithNCounters`, `AddCounter`, `RemoveCounter`, counter replacement,
placement-batch and payment service signatures remain. Counter arithmetic and
loyalty damage avoid byte casts. The fixed arrays retain clone value semantics.
Existing typed byte callers receive mechanical integer updates, including the
retained Antiquities counter snapshot and two counter arithmetic statements.
The wider type also makes 243 conversions unnecessary across 97 caller files;
lint diagnostics identify each conversion, and an AST-directed edit removes only
those wrappers. Retained Go sets, tests, engine services, gametest and interactive
consumers keep their existing behavior. No compiler spelling or rule is changed
by those caller edits.
No Go card behavior was used as a requirement or rewritten. No public function
was removed; a fresh repository search found all counter-array/byte consumers.
The separate bounded-counter declaration context still limits its literal to
0–255. General numeric arithmetic remains bounded by machine `int`.

RON owns declaration checks. Mage owns storage, arithmetic, exact departure
facts and replacement execution. Shared placement, affected-controller choices,
replacement rechecks, counter payment rollback, entry observations, cloning,
phasing and return services are reused. No new binding or last-action slot exists.
No global X is read by the new adapter. Runtime loading and removal gates remain
open; no registry fallback or partial publication was added.

### Affected records and gameplay evidence

Complete fixed-entry consumer records were inspected for **Clockwork Beast,
Clockwork Avian, Tetravus, Triskelion, Icatian Moneychanger, Merseine, Divine
Intervention and Rasputin Dreamweaver**: eight unique names / thirteen records.
Rock Hydra's six complete reprints and Frankenstein's Monster's complete DRK
record were inspected for the downstream prerequisite; neither gains acceptance.
The complete Gauntlets and Drain Life records supplied comparison evidence.

Failing-first native tests exposed addition truncation at the old boundary.
RON tests independently rejected 256, 300 and 65,536 fixed entry declarations;
complete Triskelion gameplay failed with replacement-expanded entry counters.
An omitted test import was corrected before confirming the RON failures. The
native clone fixture later needed to reacquire the mutable live permanent after
copy-on-write counter addition; its count assertions were kept. No regression
assertion was weakened. The former 256 invalid declaration moves to positive
RON gameplay because its storage restriction is the capability removed here.

Card-to-test mappings:

- **Triskelion → TestRONCounterCapacityTriskelion**: unchanged complete ATQ
  record, seven counter doublers, 384 counters and 385 power at entry, rejected
  target announcement without spending a counter, paid removal to 383 and one
  damage to the opponent. Existing complete-card tests retain all prior behavior.
- **Triskelion → TestRONCounterCapacityReplacementOrder**: add 300 then double
  yields 606; double then add yields 306. The entering controller chooses, while
  the donor controller makes no replacement decision.
- `TestRONCounterCapacity`: synthetic RON spell casting and resolution with
  255, 256, 300 and 65,536 counters; no early entry, independent factory instances
  and exact clone counts after live removal.
- `TestCounterCapacity`: native large-count entry, P/T reads, addition, failed
  removal without mutation, successful removal, phase cycles, clone isolation,
  exact departure counter snapshots and fresh counts on a returned visit.
- `TestRONEntryCountersDeclarations` retains malformed-shape, negative amount,
  incompatible counter, source and context rejection. Existing counter payment,
  batch placement, source-LKI, layer, loyalty, gameplay and clone tests keep their
  original assertions with integer-typed fixtures.
- The other seven fixed-entry names keep their existing test mappings. This run
  adds no complete-card coverage claim for them or the two entry-X consumers.

Rules evidence: local CR 107.1, 122.1–122.6, 400.7, 614.1c, 614.12,
616.1 and 702.26 supply integer counts, counter placement, entry replacement,
identity and phasing contracts. No Oracle transcription was changed. Gauntlets'
known Aura-only source defect remains deferred. No RON source changed, so the
vocabulary snapshot and source hashes remain unchanged.

### Validation and measured outcome

Required verification passed after the counter-type and lint integration fixes:

- `go test ./pkg/mage/dsl/ron -run TestRONCounterCapacity -count=1` passed.
- `go test ./pkg/mage -run TestCounterCapacity -count=1` passed. The final
  uncached native/RON package run also includes these tests.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed, including
  retained Go-set callers. Both were repeated after the broad conversion cleanup.
- The engine skill's `go test ./... -short -count=1` passed, also repeated after
  the broad cleanup.
- Initial `make lint` runs exposed conversions made redundant by integer
  counter storage. Default diagnostic limits and per-line collapsing hid later
  occurrences. Expanded diagnostics and AST-directed edits removed 243 redundant
  conversions across 97 files. The remaining 14 conversions were confined to
  eight native/RON test files. Final
  `go test ./pkg/mage ./pkg/mage/dsl/ron -short -count=1` passed after those edits.
- Final `make lint` passed with **0 issues**. Automatic formatter changes and
  the conversion edits were reviewed. No unrelated semantic edit occurred.
- `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected. Its complete
  JSON report is identical to the fresh baseline: **855 accepted / 131 rejected
  names**, **2,197 compiled / 248 rejected records**, out of 986 names / 2,445
  records. No unsupported record was accepted or skipped. Rock Hydra still
  rejects `ValueX` at entry; Monster still rejects the entry sequence; Gauntlets
  still rejects `PermanentsChangedControlThisWay`; Drain Life still rejects
  `SpendOnlyColorManaOnX`.

Actual gains match the estimate: zero names / zero records. Complete-card
coverage is unchanged; existing Triskelion coverage now includes large-count
replacement and payment evidence. The shared storage prerequisite is complete.
Entry X, runtime cutover, Go-card deletion and obsolete DSL cleanup remain open.
No environmental or pre-existing failure blocks this slice.

Changed implementation files: mage `card.go`, `zone_system.go`,
`damage_system.go`, interactive `types.go`, `snapshot.go` and `eval/eval.go`;
RON `entry_tapped.go`; new native and
RON `counter_capacity_test.go`. Other code changes only adapt counter types and remove newly redundant
conversions in existing tests, engine services, gametest, interactive consumers
and retained Go sets. API/rules docs and this ledger
record the established contract. No legacy DSL function, Go definition or
set-generation tool was removed.

### Ranked next candidates

1. **Announced X in entry operations — Rock Hydra, Frankenstein's Monster.**
   Counter capacity is now ready. Carry the entering spell's own X through a
   typed entry context; direct entry must use zero rather than ambient resolution
   X. Lower Hydra's entry counter operation. Monster can reuse the context for
   later exile payment. Zero immediate unlocks; concrete two-name downstream
   value; medium effort. Other complete-record blockers above remain.
2. **Exchange outcomes and Aura-only source correction — Gauntlets of Chaos.**
   Targets and exchange are ready. Save a local successful result and gate later
   destruction; correct the attachment collection only with Oracle evidence.
   Estimated one name / one record; small/medium effort.
3. **Restricted X casting payment — Drain Life.** Clamped actual-damage results
   remain separate. Zero immediate unlocks; eventual one name / six records;
   medium effort. Kudzu's optional former-host action remains another single-card
   completion candidate.

This run stops after integer counter capacity.


## ENTRY-ANNOUNCED-X — 2026-10-13

Status: implemented; validation results are recorded below. This run completes
one entry-counter capability and its typed entry-context prerequisite.

### Environment, candidates and selection

The initial working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -short -count=1` passed. The baseline loader exited 1
for unsupported collection records, not a build or test failure: 855 accepted /
131 rejected unique names; 2,197 compiled / 248 rejected records; 986 names and
2,445 records total. No partial registry was published.

Complete Rock Hydra records in LEA, LEB, 2ED, 3ED, CED and CEI, Frankenstein's
Monster in DRK, Gauntlets of Chaos in LEG and all six Drain Life reprints were
read, including nested forms. The vocabulary's `ValueX` and
`EntersWithNumberCounters` signatures were inspected separately. Reprints count
once in this comparison:

| Candidate | Distinct affected names | Estimated full-card gain | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Announced X at entry | Rock Hydra; Frankenstein's Monster | 0 names / 0 records | Hydra needs static counter-removal prevention and finite self-recipient prevention; Monster needs entry exile payment, payment results, combination-counter choice and failed-entry handling. Exact entry facts are a shared prerequisite for both names. | Medium; one typed entry context, counter adapter and identity/replacement tests |
| Successful exchange outcomes | Gauntlets of Chaos | 1 name / 1 record | Needs local successful-exchange results and Oracle-supported Aura-only attachment correction. Existing targets and exchange mechanics are ready. | Small/medium; local result and source audit |
| Restricted casting X payment | Drain Life | 0 names / 0 records | Black-only X payment enables one name / six records later; clamped actual-damage results remain separate. | Medium; mana solver/payment policy and rollback evidence |

Candidate 1 wins because the storage prerequisite is complete and it supplies
concrete shared behavior for two names. The engine already had entry-X counters,
but they read ambient `ResolutionState.X`. That is an engine contract gap, not
just a missing compiler case. Direct entry must not inherit another action's X.
No Go card definition supplied requirements or was changed.

### Supported contract and reuse boundary

RON now lowers the exact shape
`AsPermanentEnters(ThisPermanent, [EntersWithNumberCounters(ValueX, counter)])`.
`ValueX` must be a symbol; the list contains one operation; `counter` uses the
existing permanent-counter declaration checks. Fixed `Integer(n)` forms remain
supported. Wrong arity, tuple-form X, other sources, event-value references,
incompatible counter types and resolving spell/activation use remain errors.

The factory builds a fresh
`EntersWithCounters(counter, AnnouncedXForEnteringPermanent())` ability.
`AnnouncedXForEnteringPermanent() EntryCounterValue` reads a private typed
`battlefieldEntryContext`, with separate integer fields `announcedX` and
`castColorsSpent`. The proposed entry action keeps these values through entry
copy replacement. `putPermanentSpellOnBattlefield` supplies the exact stack
object's X. Direct entry wrappers supply zero. Existing public entry and counter
APIs retain their signatures; the private entry-value method now accepts the
context rather than a color-count integer. The existing `EntersWithXCounters`
ability also uses this context and cannot read ambient resolution X.

Mage owns entry facts and counter execution; RON owns syntax and context
validation. Shared casting transactions, permanent spell copies, entry-copy
rules, counter replacements, affected-controller order, event dispatch,
incarnation LKI, phasing and cloning are reused. Zero-X entry still permits
external entry-counter additions through the existing synthesized proposal.
Doubling alone cannot create counters. A new battlefield visit receives zero
entry X unless a new resolving spell supplies it. No global result slot,
card-name execution branch or production registry fallback was added.

### Failing-first evidence and test mappings

The initial native test failed to compile because the new typed constructor was
absent. RON X=0/3/300 and Hydra entry tests failed at the existing unsupported
`ValueX` diagnostic. A later zero-X replacement assertion exposed suppression
of external additions; it failed with zero instead of two counters before the
positive-count guard was restored. Test fixture method/type names were corrected
without changing assertions. The former `ValueX` invalid-declaration case becomes
a positive gameplay case; malformed `ValueX()` remains rejected.

- **Rock Hydra → `TestRONEntryAnnouncedXHydraRecord`**: all six complete records
  are parsed. A test-only projection keeps original metadata, mana cost and P/T,
  and executes the exact entry rule with X=4 through casting and resolution.
  Every unmodified full record must still reject its unsupported remaining rule.
  This is entry-rule gameplay evidence, not complete-card coverage.
- **Frankenstein's Monster → prerequisite only**: its complete entry sequence
  remains rejected. No Monster gameplay coverage is claimed. The entry context
  now carries the spell's X for a future entry payment adapter.
- `TestRONEntryAnnouncedX`: synthetic RON X=0/3/300, no entry before resolution,
  independent spell-copy entries under another controller, clone isolation,
  current P/T, fresh factory instances and direct-entry zero.
- `TestRONEntryAnnouncedXDeclarations`: malformed X, unbound event values,
  incompatible counter/source declarations and resolving-context rejection.
- `TestRONEntryAnnouncedXReplacementOrder`: X=3 yields eight versus seven
  counters for the two orders; X=0 plus one then double yields two. Only the
  entering controller chooses. The donor controller does not choose.
- `TestRONEntryAnnouncedXObserver`: fresh ability identities and counter
  visibility at the single actual entry event, after spell resolution begins.
- `TestEntryAnnouncedXDirect` and `SpellAndReturn`: both typed and retained
  X-counter APIs; ambient-X exclusion, unpaid X rollback without resource
  mutation, copies, clones, phasing, exact departure snapshots and a fresh
  returned visit with zero X.
- `TestEntryAnnouncedXContext`: separate announced-X and cast-color facts,
  unchanged ambient X and no battlefield installation during preparation.
- `TestEntryAnnouncedXAcquiredCopyRules`: entry copying uses the entering spell's
  X=4, not the donor's 17 counters; the donor and choice count remain unchanged.
- `TestEntryAnnouncedXDefinition`: valid typed declarations and invalid counter
  definition checks. Existing entry payment-color and compile-failure fixtures
  remain in the full-suite validation.

Rules evidence: local CR 107.3a/g/m, 122.6, 400.7/d, 614.1c, 614.12,
616.1 and 702.26d. Local `data/LEA.json`, `data/DRK.json` and `data/LEG.json`
supply Oracle comparison evidence for the candidates. No RON source changed;
vocabulary counts and source hashes remain unchanged. Gauntlets' known Aura
filter defect remains deferred. The finite-prevention spelling
`NextAmountOfDamageThatWouldBeDealtThisTurnByPermanent` appears in both Hydra and
Rasputin, while local Oracle text specifies damage to the source. The next slice
must settle that source expression before lowering it; this run does not assign
new semantics to it or correct any reprint.

### Verification and measured outcome

Final required checks passed after the private entry-wrapper cleanup:

- `go test ./pkg/mage/dsl/ron -run TestRONEntryAnnouncedX -count=1` passed.
- `go test ./pkg/mage -run TestEntryAnnouncedX -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's `go test ./... -short -count=1` passed.
- Initial `make lint` found one `unparam` issue: the old private direct-entry
  result wrapper always received nil colors. A fresh whole-repository caller
  audit found four direct preparation paths: graveyard, temporary exile, linked
  exile and hand entry. These callers now pass an explicit zero-valued typed
  entry context. The redundant private wrapper was removed; public APIs remain.
  Every preceding check was rerun after this change and passed.
- Final `make lint` passed with **0 issues**. Formatter and automatic fixes were
  reviewed; no extra code edits resulted.
- `git diff --check` passed.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected: **855 accepted /
  131 rejected unique names**, **2,197 compiled / 248 rejected records**, out of
  986 names / 2,445 records. Exactly six diagnostics changed, all Rock Hydra
  reprints. Each advanced from unsupported entry `ValueX` to
  `unsupported static prevention event (EachDamageWouldBeDealtToRecipient)`.
  Monster still rejects the multi-operation entry sequence; Gauntlets still
  rejects `PermanentsChangedControlThisWay`; Drain Life still rejects
  `SpendOnlyColorManaOnX`. All other diagnostics are identical to baseline.

No environmental or pre-existing failure blocks this slice.

Actual full-card gains match the estimate: zero names / zero records. New entry
rule gameplay evidence covers Rock Hydra's six reprints. Complete-card gameplay
coverage is unchanged. Frankenstein's Monster's multi-operation entry still
rejects. Strict collection publication, runtime cutover, Go-card deletion and
unused DSL removal gates remain open. No public DSL function or Go-set generator
was removed.

Changed files: `pkg/mage/composable_actions.go`, `entry_copy.go`, `game.go`,
`selected_source_prevention.go`, `effect_cards.go`, `effect_removal.go`,
`linked_exile_entry.go`, `optional_hand_entry.go`, `doc.go` and new
`entry_announced_x_test.go`;
RON `entry_tapped.go`, `entry_counters_test.go` and new `entry_announced_x_test.go`;
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger.

### Ranked next candidates

1. **Finite self-recipient prevention — Rock Hydra; Rasputin Dreamweaver.**
   Entry X is ready for Hydra. Shared prevention and counter-payment services
   exist. Audit the `ByPermanent` source spelling against the local Oracle
   self-recipient requirement, including every identical reprint. Estimated
   zero immediate names / records: Hydra also needs its static counter-removal
   prevention adapter; Rasputin also needs
   `StartedThisTurnUntapped` and the permanent seven-counter cap. Small/medium
   adapter effort after the source contract is settled; two-name downstream value.
2. **Mandatory entry exile payment using announced X — Frankenstein's Monster.**
   Integer counts and exact entry X are ready. Add one typed entry-payment
   contract before combination-counter distribution and failed-entry continuation.
   Zero immediate unlocks; concrete prerequisite for the complete DRK record.
   Medium effort; do not combine the whole entry sequence into one oversized run.
3. **Exchange outcomes and Aura-only correction — Gauntlets of Chaos.**
   Exact targets and exchange are ready. Estimated one name / one record;
   small/medium effort. Drain Life's restricted X casting payment and clamped
   damage remain separate later candidates.

This run stops after announced X at entry.
