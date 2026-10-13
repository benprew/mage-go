# mtgish RON execution progress

Completed slices, test mappings, validation evidence and historical decisions
are in [mtgish-ron-completed.md](mtgish-ron-completed.md).
Use this document for current status and next work.

## Current status

Latest measured result (2026-10-13): **848 accepted / 138 rejected names**
(986 total), **2,189 compiled / 256 rejected records** (2,445 total).
The latest slice adds source-presence control duration and unlocks Scarwood Bandits.
The strict loader still rejects the collection and publishes no partial registry.

The latest slice is **SOURCE-PRESENCE-CONTROL-DURATION**. Captured layers end
permanently on source departure or phasing. Scarwood Bandits now has complete
RON-backed gameplay coverage. Other duration roles keep separate contracts.
Runtime cutover and removal gates remain open.

## Ranked next candidates

Reassess these candidates with fresh loader and complete-record evidence under
[the execution design](mtgish-ron-execution.md). Count reprints once.

1. **Eligible-player alternatives and payer bindings.**
   Fixed mana offers are complete. Worms of the Earth and Cleansing need typed
   payer/subject scope; entry prevention and repeated result orchestration remain
   separate blockers. Select a small shared prerequisite from these contracts.
2. **Dependent permanent target comparison: Gauntlets of Chaos.**
   Zero immediate gains. Exchange-result handling and the documented Aura-only
   source correction remain required.
3. **Source-leave layer duration: Gaea's Liege.**
   Keep phasing distinct from Scarwood's source-presence control duration. Its
   defending-player filter, characteristic rules and known source defect remain.

## Dependencies and source blockers

- Reuse `TapAnyNumberEach`, `PlayerNumbersBinding` and `PlayerNumber` for optional
  tap results. The RON producer requires one conjunctive ActionPlayer controller
  clause. Saved plural choices now use `ChooseUpToPermanentsEach` and `PermanentSetBinding`.
  `EachablePlayer` is local to the repetition count; other uses remain unsupported.
- Reuse `SacrificeAllButEach`, `LeastControlledPermanentCount`, the bounded
  subset selector and the existing simultaneous sacrifice service for permanent
  cardinality work. The latest slice supports required kept-count choices in
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
