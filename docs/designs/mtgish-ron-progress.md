# mtgish RON execution progress

Completed slices, test mappings, validation evidence and historical decisions
are in [mtgish-ron-completed.md](mtgish-ron-completed.md).
Use this document for current status and next work.

## Current status

Latest measured result (2026-10-14): **880 accepted / 106 rejected names**
(986 total), **2,249 compiled / 196 rejected records** (2,445 total).
The latest slice is **LIVE-STATIC-COLLECTION-QUERIES**. Typed live sets supply
static control conditions and permanent-source prevention recipients. Queries
use the exact source visit and current controller; waiting references use the
same membership and condition checks.
Camel now has complete-record native casting, band formation, combat,
prevention, spell-copy, phasing, return and clone evidence. Bronze Horse advances
to its spell-source prevention blocker. Guardian Beast protection still rejects.
Estimated and actual gains are **one unique name / one record**. No RON source
changed. Accepted records remain accepted; loader baseline and hashes are fresh.

The loader still rejects the collection atomically. Runtime cutover, Go-card
removal and DSL cleanup remain gated by complete support and native startup.

## Ranked next candidates

Reassess with fresh loader and complete-record evidence. Count reprints once.

1. **Shared live static collection queries.** Compare Camel's prevention
   recipients, Bronze Horse's source exclusion and Guardian Beast's current
   controller membership before choosing a complete supported query context.
2. **Conditional artifact protection rules — Guardian Beast.** Live membership
   and new-Aura/control acquisition restrictions need exact shared contracts.
3. **Entry payment/result producers.** Compare Nameless Race, Frankenstein's
   Monster and Wood Elemental before implementing a typed shared prerequisite.

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
- Gauntlets of Chaos's Aura-only source defect at `cards/mtgish/LEG.lines.ron:95`
  is resolved by CONTROL-EXCHANGE-RESULT. The complete record now compiles and
  has RON gameplay evidence; attached Equipment is retained.
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


## 2026-10-13 — CONTROL-EXCHANGE-RESULT

### Candidate comparison and selection

The working tree was clean. Native environment: Go 1.27.2 on Darwin/arm64.
Fresh baseline `make build` and `go test ./pkg/mage/dsl/ron -count=1` passed.
The baseline strict loader exited 1 for unsupported collection records, not a
build failure: 855 accepted / 131 rejected names; 2,197 compiled / 248 rejected
records, out of 986 names / 2,445 records. Complete RON records and nested forms
were inspected for all compared names; reprints were counted once.

| Candidate from the previous ranking | Distinct affected names | Estimated full-card gains | Remaining blockers and downstream value | Full implementation/test effort |
| --- | --- | --- | --- | --- |
| Finite self-recipient prevention | Rock Hydra; Rasputin Dreamweaver | 0 names / 0 records | Local LEA/LEG Oracle data says damage **to** the source, while all seven RON records say `ByPermanent`. The exact corrected `ToRecipient(Integer(1), Permanent(ThisPermanent))` form already has AST lowering through `compileFinitePrevention`, typed `PreventNextDamage` and shared shields. Hydra still needs static per-point counter-removal prevention; Rasputin needs turn-start history and a permanent counter cap. Two-name benefit from data correction and gameplay evidence, but no missing compiler/engine capability for this candidate. | Small data/coverage task; it is not a new execution slice. Static Hydra prevention is a separate small/medium engine slice. |
| Mandatory announced-X entry exile payment | Frankenstein's Monster | 0 names / 0 records | Integer counts and entry X are ready. A typed mandatory graveyard-exile entry transaction would enable one DRK record's later counter-choice distribution and failed-entry continuation. Both later forms still block the full card. | Medium; transaction, replacement, identity and entry-order tests are needed. |
| Saved exchange outcomes and Aura-only source correction | Gauntlets of Chaos | 1 name / 1 record | Exact dependent targets, shared atomic exchange, contextual attachment sets and destruction are ready. The missing result consumer and documented Aura-only defect are its only remaining blockers. A typed historical result supports ordinary resolving sequences and delayed continuations for future exchange consumers. | Small/medium; completes in this run. |

Candidate #1 was evaluated first. Its Oracle-correct shape is already implemented;
adding a second adapter or assigning recipient meaning to `ByPermanent` would
be unnecessary or incorrect. No new shared capability was available in that
candidate. It remains a source/coverage task and was not combined with this run.
Candidate #3 has the largest credible complete-card gain among these three for
its effort. This run implements only the missing exchange-result capability and
its dependent, Oracle-supported Gauntlets filter correction. The selection,
reuse boundary and acceptance cases were stated before implementation.

### Exact contract and reuse boundary

Engine APIs:

```go
func BindControlExchangeResult(name string) *ControlExchangeResultBinding
func ExchangePermanentControlInto(first, second PermanentRef, result *ControlExchangeResultBinding) ResolvingEffect
func IfControlExchanged(result *ControlExchangeResultBinding, body ResolvingEffect) ResolvingEffect
```

The producer saves success or failure on every call. It calls the existing
`Game.ExchangeControl` service; atomic layer 2 application, current controller
snapshots, control prohibitions, source-independent duration and continuous
control history keep their existing contracts. Both exact visits must be
present and unphased. Same subjects/controllers, prohibited control changes,
departure or return save failure. Failure clears prior success. LKI cannot
supply an absent action subject. The result is historical and does not reread
later controllers. The result-free `ExchangePermanentControl` API is preserved.

The public typed result wraps a private number binding in the existing shared
resolution frame. It does not add another frame map or global last-action slot.
Definition inspection checks producer dominance and nil declarations. Producer
references and conditional bodies retain explicit source-read traversal.
Copies, clones, delays and factory instances use the existing saved frame and
exact-reference services. No new decision, payment or replacement service is
introduced; existing cost transactions and simultaneous destruction are reused.

RON spelling and lexical scope stay in `ron`. The supported consumer is exactly
`If(PermanentsChangedControlThisWay, [actions...])`, with a symbol condition and
a nonempty body, after the latest local two-reference `ExchangeControl` in an
ordinary resolving sequence. Spell, activation and delayed contexts have direct
gameplay evidence; ordinary resolving compiler paths also retain trigger and
player/choice scopes. Branch producers cannot escape. Delayed bodies can capture
an earlier outer result while a later producer gets a separate binding. Entry,
replacement, static, cost and other conditional forms remain unsupported.

### Source evidence and failing-first tests

`data/LEG.json` Oracle text requires destruction of Auras attached to the
exchanged permanents. The one complete `cards/mtgish/LEG.lines.ron:95` record now
intersects `IsEnchantmentType(Aura)` with the original two attachment sets.
A fresh collection search found no identical reprint for this name. No card-name
engine branch, Go fallback or partial production registry was added. Hydra and
Rasputin's `ByPermanent` source defects remain unchanged and explicit.

Rules evidence: local CR 701.12a-b (all-or-nothing exchange and simultaneous
control), 400.7 (new visits), and 702.26 (phasing). Existing attachment,
destruction, cost, target, control-layer and delayed-frame contracts are reused.

The initial native test failed to compile on the three missing APIs. The RON
test, after an attachment fixture method correction, failed at the unsupported
`PermanentsChangedControlThisWay` condition. After result lowering, correct
attachment setup made the Aura-only assertion fail because the old source also
destroyed Equipment. Only then was the RON collection corrected. A stale
pre-clone permanent pointer and one synthetic RON parenthesis were corrected;
assertions were retained. Four exact inventory/count assertions then exposed
the expected one-record gain and were updated to 2,198 / 247. No gameplay test
was weakened.

- **Gauntlets of Chaos → `TestRONControlExchangeResultGauntlets`**: full RON
  record; mana payment and sacrifice, unpaid rollback, actions at resolution,
  source departure through its cost, exchange success and control prohibition,
  phasing and phase cycle, return identity, shroud, copied ability, destruction
  of attached Auras and retention of Equipment, and independent clone outcomes.
- **Gauntlets of Chaos → `TestRONTargetDependencyGauntletsPrerequisite`**:
  its former prefix projection is removed. The full RON record retains all
  earlier invalid-pair/payment rollback, dependent-type, retargeted-copy,
  source-return, control-change, phasing, identity and clone assertions.
- `TestRONControlExchangeResultInvalid`: unbound, premature, tuple-shaped,
  empty-body and branch-local escaping result reads reject at compilation.
- `TestRONControlExchangeResultContexts`: the latest failure replaces a previous
  success, the latest success replaces failure, inner branch producers keep
  their scope, and delayed historical reads survive another exchange, source
  departure and independent clone resolution.
- `TestRONControlExchangeResultCopiesAndInstances`: two copied spell resolutions
  retain separate controllers and outcomes; a blocked clone and a later factory
  instance cannot inherit earlier success.
- `TestControlExchangeResult`: saved success, repeated failure clearing, equal
  controllers, prohibited changes, phasing, phase cycle, return and identical
  subjects. `TestControlExchangeResultDeclaration` checks valid sequencing,
  unbound reads, nil results, nil subjects and nil bodies.
- Prior native `TestReferencedControlExchange*` and
  `TestExchangeControlIsAtomicAndUsesSnapshotControllers` remain in verification
  for exact visits, waiting source queries, durations and simultaneous layers.
  Earlier compiler prerequisite checks now expect complete Gauntlets acceptance.

### Verification and measured outcome

Required final checks, after the exact count updates:

- `go test ./pkg/mage/dsl/ron -run TestRONControlExchangeResult -count=1` passed.
- Focused prior exchange, target dependency, type comparison and singular-target
  tests passed. Focused new/prior native exchange tests passed.
- Focused collection-definition, inventory and numeric-consumer count checks
  passed with 2,198 accepted / 247 rejected records.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's `go test ./... -short -count=1` passed.
- Initial `make lint` reported one `prealloc` issue. The producer/continuation
  effect list now has capacity for both. All preceding required checks and the
  skill's repository short tests were rerun after this change and passed.
- Final `make lint` passed with **0 issues**. Automatic formatter/fix output was
  reviewed; no additional code changes resulted.
- `git diff --check` passed before the final loader check and again after the
  ledger and vocabulary updates.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected: **856 accepted /
  130 rejected unique names**, **2,198 compiled / 247 rejected records**, out of
  986 names / 2,445 records. The only removed diagnostic is Gauntlets' unsupported
  `PermanentsChangedControlThisWay`. All other diagnostics match the fresh
  baseline exactly. This is collection rejection, not a build/test failure.

The vocabulary snapshot was refreshed from the actual parsed source. The LEG
file hash changed. `And`, `Aura` and `IsEnchantmentType` gained one occurrence
each; three example groups received updated Gauntlets positions. No new
spelling/signature was added. The loader snapshot uses the final diagnostics.
Its previously stored loader data predated the fresh baseline, so this refresh
also records prior completed slices. The runtime baseline-to-final comparison
above still has exactly one changed record: Gauntlets of Chaos.

Actual gains match the estimate: **one complete name / one record**. Complete
RON gameplay coverage now includes Gauntlets; compilation and gameplay are
reported separately. No environmental or pre-existing failure blocks this slice.
Strict full-collection publication, runtime cutover, Go-card removal and unused
DSL cleanup gates remain open. No Go card definitions, public DSL functions or
Go-set generators were removed.

Changed files: `pkg/mage/control_exchange_refs.go`, new
`control_exchange_result.go` and `control_exchange_result_test.go`, and `doc.go`;
RON `compiler.go`, `optional_payment.go`, `player_scope.go`, new
`control_exchange_result.go` and `control_exchange_result_test.go`, existing
`control_exchange_test.go`, `target_dependency_test.go`,
`permanent_type_comparison_test.go`, `singular_target_groups_test.go`,
`definition_plan_test.go`, `inventory_test.go` and `numeric_consumers_test.go`;
`cards/mtgish/LEG.lines.ron`; `docs/dsl-reference.md`,
`docs/engine-rules-contracts.md`, vocabulary snapshot and this ledger.

### Ranked next candidates

1. **Static per-point counter-removal prevention — Rock Hydra.** Entry X and
   counter services are ready. Add an exact per-one-damage static replacement
   contract and correct finite self-recipient source data across all identical
   reprints with the local Oracle evidence. Potential one name / six records;
   small/medium effort after checking complete nested upkeep/counter forms.
   Rasputin's identical finite source defect must be corrected with matching
   source evidence, but its history and cap remain separate engine work.
2. **Mandatory entry exile payment using announced X — Frankenstein's Monster.**
   Integer storage and entry X are ready. Add the typed mandatory entry-payment
   contract before combination-counter distribution and failed-entry continuation.
   Zero immediate unlocks; one DRK record's concrete prerequisite. Medium effort.
3. **Restricted casting X payment — Drain Life.** Black-only announced-X payment
   must use the shared casting solver and rollback. Zero immediate unlocks across
   six records; clamped actual damage/life gain remains. Medium effort. Compare
   fresh shared multi-card candidates before choosing any single-card adapter.

This run stops after saved exchange outcomes.

## 2026-10-13 — COUNTER-POINT-PREVENTION

### Fresh baseline and candidate selection

The initial working tree was clean. `make build` and fresh
`go test ./pkg/mage/dsl/ron -count=1` passed. Initial strict
`go run ./cmd/loadcards -json` exited 1 with 2,198 compiled / 247 rejected
records, 856 accepted / 130 rejected names, from 2,445 records / 986 names.
This was expected collection rejection, not an environment or build failure.
Instructions, engine API, execution design, top status and latest ledger entry
were read before selection. Targeted vocabulary queries and complete RON records
supplied requirements; no Go card definition supplied requirements.

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Static per-point counter-removal prevention, ranked #1 | Rock Hydra; Rasputin Dreamweaver for matching source correction | 1 name / 6 records | Hydra entry X, counter placement and upkeep timing are ready. Its finite self shield needs an Oracle-supported recipient correction. Rasputin needs turn-start history and an absolute counter cap. The reusable engine rule already exists but must preserve counter removal under unpreventable damage. | Small/medium: exact adapter, two pipeline entry points, full-card and replacement tests |
| Mandatory entry exile payment using announced X | Frankenstein's Monster | 0 names / 0 records | Shared entry X and counter storage are ready. Entry payment results, combination-counter distribution and failed-entry continuation still block the complete DRK record. Establishing the typed entry transaction enables those downstream operations. | Medium: entry choices, payment rollback and result binding |
| Restricted casting X payment | Drain Life | 0 names / 0 records | Six complete reprints also need clamped actual damage/life gain. A black-only X solver contract would enable the casting prerequisite while preserving the generic and black fixed cost symbols. | Medium: mana planning, announced X, rollback and copies |

Ranked #1 is dependency-ready and gives the only credible immediate gain among
these candidates. It reuses shared engine behavior and repairs a concrete rules
contract instead of adding a card-specific executor. Complete nested Hydra rules
confirm that the finite source correction leaves no other compiler blockers.
The slice ends after this capability; later candidates are not implemented.

### Exact shapes, engine boundary and source evidence

The supported static permanent rule is:

```text
PreventDamage(EachDamageWouldBeDealtToRecipient(Permanent(ThisPermanent)),
 [If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(counter)),
  [RemoveCounters([ACounterOfTypeFromPermanent(counter,ThisPermanent)]),
   PreventThatDamage])])
```

All source references must be valid singular source references. The checked and
removed counter types must be equal and valid permanent counters. The condition,
removal quantity, operation order and static context are exact; other declarations
remain compiler errors. `compileCounterPointPrevention(Node, scope)` owns syntax
and validation and lowers to a fresh
`WithStaticAbility(PreventDamageToSourceByRemovingCounters(kind))` per factory.
Inventory roles now distinguish event, recipient, condition and counter removal.
No general action executor runs inside this static replacement.

The retained native signature is
`PreventDamageToSourceByRemovingCounters(CounterType) ContinuousEffect`.
Existing source incarnation and rule IDs control active registration. Shared
replacement candidates, affected-controller choices and APNAP batch ordering
remain in the engine. Positive damage removes and prevents the minimum of damage
and matching counters. Residual packets retain source and combat information.
No LKI read can make an absent or returned recipient actionable. Phasing and
ability removal stop the rule; phase-in or restoration resumes the present rule.
Clones keep independent counters, damage and replacement registrations.

A missing engine contract was found: both replacement pipeline entry points
skipped counter removal for unpreventable damage. CR 615.12 and 615.12a require
additional effects to occur once per event even when prevention cannot occur.
The private `appliesToUnpreventableDamage() bool` contract lets this counter rule
reach the shared candidate chooser. It removes counters and returns the unchanged
packet when prevention is prohibited. Finite shields retain capacity. Other rules
retain the existing combined prevention/redirection prohibition. No new RON
unpreventable-damage declaration or exported engine signature is introduced.

Local `data/LEA.json` Rock Hydra Oracle text says its red activation prevents
damage dealt **to** the creature. `data/LEG.json` says the same for Rasputin's
counter-paid shield. All six Hydra records (LEA, LEB, 2ED, 3ED, CED, CEI) and the
single LEG Rasputin record incorrectly used
`NextAmountOfDamageThatWouldBeDealtThisTurnByPermanent(Integer(1),ThisPermanent)`.
They now use the already supported
`NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Permanent(ThisPermanent))`.
A fresh source search found exactly those seven occurrences. No predicate was
inverted, no unrelated record was corrected, and no new spelling was invented.
Rasputin's full record still rejects at `StartedThisTurnUntapped`; its cap remains
an additional blocker. Relevant local CR sections: 615.1, 615.5–7, 615.12–12a,
616.1–2, with retained identity/phasing contracts from 400.7 and 702.26.

### Failing-first evidence and card-to-test mapping

Initial test fixture errors (nonexistent action executor and attribute names)
were corrected before recording semantic failures. Native
`TestCounterPointPreventionUnpreventable` then failed with one unpreventable
damage point retaining all three counters. RON positive tests failed at the
unsupported `EachDamageWouldBeDealtToRecipient` event. After lowering, complete
Hydra records failed at their finite `ByPermanent` source defect. The source
correction uses the Oracle evidence above. Activation fixture indices were fixed
to include entry and static rules. Assertions were retained. Fresh collection
checks exposed 2,204 / 241 in four inventory/count tests; those exact expectations
were updated. The prior Hydra entry projection is removed and now compiles the
complete records.

- **Rock Hydra → `TestRONCounterPointPreventionHydra`**: all six complete
  reprints; unpaid cast, X mana payment, entry counters and P/T, two-point
  removal, upkeep-only activation and failed-payment rollback, counter addition,
  finite shield after ability removal, and outgoing damage left unaffected.
- **Rock Hydra → `TestRONCounterPointPreventionHydraShieldLifetime`**: copied
  activations, control change, phasing before versus after resolution, phase-in,
  source return before/after resolution, cleanup expiry and independent clones.
- **Rock Hydra → `TestRONCounterPointPreventionBatchAndCombat`**: native combat
  assignment fixes power before counter removal. Generic RON rules also exercise
  simultaneous depletion, independent subjects, departed/returned source LKI,
  player damage and clone isolation.
- **Rock Hydra → `TestRONCounterPointPreventionHydraZeroToughness`**: all damage
  can be prevented while the last counter is removed. Later state actions move
  the zero-toughness Hydra to its owner's graveyard despite regeneration.
- **Rock Hydra → `TestRONEntryAnnouncedXHydraRecord`**: the former prefix test
  now uses full records and retains its entry-X checks across all six reprints.
- **Rasputin Dreamweaver → `TestRONCounterPointPreventionRasputinSourceCorrection`**:
  explicitly projected entry/mana/shield prerequisites, separate counter costs,
  immediate colorless mana and the corrected incoming shield. The complete
  record must still reject at turn-start history. No full gameplay claim is made.
- `TestRONCounterPointPrevention` covers empty counters, unpreventable damage,
  removal, phasing, phase cycle, control change, return and clone isolation.
- `TestRONCounterPointPreventionInvalid` rejects unbound/wrong source references,
  tuple symbols, unknown/mismatched counters, wrong conditions, incomplete bodies,
  wrong recipients and resolving declaration contexts.
- `TestRONCounterPointPreventionOrdering` proves the affected controller chooses
  counter prevention versus redirection and that recipient legality is rechecked.
- `TestRONCounterPointPreventionInventory` checks semantic roles and atomic loader
  rejection. Native `TestCounterPointPreventionUnpreventable` checks both pipeline
  paths, positive/zero/negative amounts, depletion, cloned counters and preservation
  of a finite shield. Prior `TestWaitingCounterPreventionReferences`/`Boundaries`
  retain exact-visit, rule-removal/restoration and waiting-query evidence.

### Verification and measured outcome

Required checks ran in fast-fail order after implementation and count updates:

- `go test ./pkg/mage/dsl/ron -run TestRONCounterPointPrevention -count=1` passed.
  Focused `TestRONEntryAnnouncedXHydraRecord` and native
  `TestCounterPointPrevention` / `TestWaitingCounterPrevention` passed.
  Prior native static prevention and allocation regressions also passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's `go test ./... -short -count=1` passed.
- Initial `make lint` found one `wastedassign` test initializer. The variable now
  uses a declaration without an explicit zero initializer. The affected native
  tests and `go vet ./...` were rerun and passed. No behavior or assertion changed.
- Final `make lint` passed with **0 issues**. Automatic formatter/fix changes
  were reviewed; no other code changes resulted.
- `git diff --check` passed before final collection validation and after this
  ledger update.
- Final `go run ./cmd/loadcards -json` exited **1**, as expected: **857 accepted /
  129 rejected unique names**, **2,204 compiled / 241 rejected records**, from
  986 names / 2,445 records. This is strict collection rejection, not a failed
  build or test. Final diagnostics exactly match the post-implementation snapshot.

The estimated and actual gains match: **one complete unique name / six records**.
Only Rock Hydra's six diagnostics disappeared. Rasputin's diagnostic moved from
unsupported finite `ByPermanent` prevention to
`LEG.lines.ron:211:736: unsupported intervening-if filter (StartedThisTurnUntapped)`.
All other diagnostics exactly match the fresh baseline. Complete RON gameplay
coverage gains Rock Hydra; Rasputin gains only tested entry/mana/shield
prerequisites and remains rejected. Compilation progress remains separate from
full-card gameplay evidence.

The vocabulary snapshot was refreshed from the parsed native inventory. Seven
file hashes changed. The unused source `ByPermanent` event signature disappeared,
so constructor signatures changed from 605 to 604. The recipient event and
`Permanent` gained seven occurrences each; six example groups moved with the
corrected source positions. No new spelling was added. Loader data uses final
2,204 / 241 diagnostics. Collection publication, native runtime cutover, Go-card
removal and obsolete DSL cleanup remain gated. No Go card definitions, generators
or public DSL functions were removed. No environmental failure blocks this slice.

Changed files: new `pkg/mage/dsl/ron/counter_point_prevention.go` and
`counter_point_prevention_test.go`, `static_prevention.go`, `inventory.go`,
`entry_announced_x_test.go`, `definition_plan_test.go`, `inventory_test.go` and
`numeric_consumers_test.go`; new native `pkg/mage/counter_point_prevention_test.go`,
`effect_manager.go`, `prevention_allocation.go`, `replacement.go` and `doc.go`;
seven source files `cards/mtgish/{LEA,LEB,2ED,3ED,CED,CEI,LEG}.lines.ron`;
API/rules-contract docs, vocabulary snapshot and this ledger.

### Ranked next candidates

1. **Mandatory entry exile payment using announced X — Frankenstein's Monster.**
   Entry X and integer counter storage are ready. Establish the typed mandatory
   entry transaction, exact graveyard choices and rollback before result-driven
   combination-counter distribution and failed-entry continuation. Estimated zero
   immediate unlocks; one complete DRK record's prerequisite. Medium effort.
2. **Restricted casting X payment — Drain Life.** Preserve fixed casting symbols
   while restricting announced X to black mana through the shared solver and
   transaction. Six reprints remain blocked by clamped actual damage/life gain.
   Estimated zero immediate unlocks; medium effort. Recompare shared multi-card
   candidates before selecting this adapter.
3. **Turn-start untapped history — Rasputin Dreamweaver.** Its corrected finite
   shield now works through existing services. Capture turn-start untapped facts
   by exact visit for intervening-if detection and resolution; its absolute seven
   dream-counter cap remains a separate contract. Estimated zero immediate
   unlocks; small/medium effort. Keep each contract a separate complete slice.

This run stops after static per-point counter-removal prevention.

## 2026-10-13 — SINGLE-GRAVEYARD-EXILE-COST

Status: complete for fixed matching plural activation payments from one graveyard.
This run implements one capability and stops. Complete collection execution,
runtime cutover, Go-card removal and obsolete DSL cleanup remain incomplete.

### Fresh baseline and candidate comparison

The working tree was clean; no user changes or unrelated untracked files were
present. Fresh `make build` and `go test ./pkg/mage/dsl/ron -count=1` passed.
Fresh `go run ./cmd/loadcards -json` exited 1 with 2,204 compiled / 241 rejected
records and 857 accepted / 129 rejected names. This was strict collection
rejection, not a build or test failure. Complete RON records and nested forms
were inspected for each candidate, including all six Drain Life reprints.
Targeted vocabulary entries confirm the three-argument single-graveyard form,
two-argument graveyard-count form and one-argument casting-color restriction.

| Candidate | Distinct affected names | Estimated complete-name gain | Other blockers and downstream use | Effort |
| --- | --- | --- | --- | --- |
| Ranked #1: mandatory announced-X entry exile payment | Frankenstein's Monster (one DRK record) | 0 | Entry X and integer counters exist, but matching plural selection, typed entry transaction/results and failure continuation are absent. Combination-counter distribution remains. The selection and batch service is an implementation prerequisite. | Medium/high for the full entry transaction and its contracts. |
| Restricted casting X payment | Drain Life (six reprints, one name) | 0 | The shared mana solver must preserve fixed symbols and restrict only X. Clamped actual-damage life gain remains. Enables the restricted X casting adapter. | Medium; no immediate complete record. |
| Matching plural exile from one eligible graveyard as activation payment | Night Soil (one FEM record); Frankenstein's Monster as a downstream selection/batch consumer | 1 | Night Soil's mana and complete Saproling token body are already supported. Monster still needs its distinct entry scope, dynamic amount, mandatory payment/result and counter branches. | Medium; reuse the existing plural cost transaction, add matching/foreign selection and complete-batch events. |

The first ranked candidate needs the reusable matching plural selector before
its entry adapter can be complete. This slice supplies that prerequisite in a
complete activation context and unlocks Night Soil immediately. It has a larger
credible gain for this run than either zero-unlock adapter. No first-error count
was treated as a complete-card estimate. No second slice was started.

### Contract, lowering and reuse boundary

RON accepts exactly
`Exile([NumberCardsInASingleGraveyard(Integer(n), cardFilter, AnyPlayer)], [])`
as an activation cost. `n` is a fixed nonnegative integer. Intrinsic card-type
and color predicates and nonempty `And`/`Or` groups use the existing card-filter
compiler. There is one exile object, no operations and at most one graveyard
exile payment in the activation. Existing `And` cost composition remains valid.
RON owns arity, spelling, role/context checks and diagnostics. Inventory gives
the object `exile_payment_object`, count `announcement_value`, predicate
`card_filter` and selector `player_collection` roles.

New native signature:
`ExileMatchingCardsFromSingleGraveyardCost(amount int, filter CardFilter, players PlayerSet) Cost`.
It extends `exileFromGraveyardCost`; current own-graveyard constructors retain
their behavior. The internal `choose` now receives source and controller IDs,
and both shared transaction call sites supply those IDs. The player set uses
that payment context. Definition validation rejects negative counts, zero
filters and nil player sets. No mutable selection enters a compiled factory.

The payer chooses distinct matching cards. Only graveyards with enough matching
cards are offered. The first choice fixes the graveyard; later choices cannot
cross graveyards. Offered visits and holder identities are saved before choices.
Zero counts make no card choices. Pure affordability queries do not choose,
move cards or use randomness. Other players do not make the payer's choices.

Existing `costPaymentTransaction` locks the copied cost and validates all costs
on its isolated game. Live commitment rechecks exact visits and matching types
before resources are spent. Returned cards cannot pay, overlapping costs cannot
reuse cards, and invalid selections do not spend mana or life. `EitherCost`
retains its locked option and the same exact selections. The new native batch
uses existing player removal, exile storage and events, putting all selected
cards in face-up exile before any per-card zone event. The graveyard holder
supplies event `PlayerID`; ownership and paying source attribution are retained.
One `EvtCardsLeftGraveyard` reports the complete count. There is no new parallel
zone or replacement system and no global last-action/result slot.

Plural payment cannot supply singular `TheCardExiledThisWay` mana-value reads.
The activation factory allocates that number binding only for the existing
singular producer. Dynamic counts, other RON player selectors, entry transactions,
resolving collections and optional resolution exile payments remain errors.
Native player-set and alternative-cost reuse does not expand those RON contexts.

### Requirements and source evidence

Night Soil's complete `cards/mtgish/FEM.lines.ron:64` record matches retained
`data/FEM.json` Oracle text: `{1}, Exile two creature cards from a single
graveyard: Create a 1/1 green Saproling creature token.` Exile is before the
colon, so it happens during activation payment, not token resolution. The payer
may choose either graveyard; no target is declared. Relevant local rules:
118.3, 118.10, 602.1a, 602.2b, 400.7, 406.3, 701.13a and 603.10.

No source-data defect or environmental failure was found in this slice. No RON
source changed, so file hashes and constructor spellings in the vocabulary
snapshot remain unchanged; no source snapshot refresh was required. The latest
loader measurements are recorded here. Frankenstein's Monster remains rejected
at its existing multiple-entry-operation diagnostic; no gameplay gain is claimed
for its complete record. Drain Life's six diagnostics remain unchanged.

### Failing-first evidence and card-to-test mapping

Test fixture API errors were corrected before semantic evidence was recorded.
Native `TestSingleGraveyardExileCost*` then failed to compile only because the
new service signature did not exist. RON positive gameplay tests failed at
`NumberCardsInASingleGraveyard`: expected one positional argument. The adapter
and engine implementation then passed those tests. An initial test assumed the
engine rejected unused target arguments; existing activation callers accept
those arguments. That fixture case was removed without changing the engine or
weakening any payment, identity, rollback or gameplay assertion.

- **Night Soil → `TestRONSingleGraveyardExileCostNightSoil`**: complete unchanged
  FEM record, native casting, own/foreign graveyards, retained noncreature cards,
  cost at announcement, token at resolution, source departure/control/phasing,
  phase cycles and return, countered activation, changed-controller copies,
  exact green 1/1 Saproling tokens, payment not repeated and cloned resolution.
- **Night Soil → `TestRONSingleGraveyardExileCostRollback`**: insufficient cards
  split across graveyards, wrong card type, unpaid mana and phased source; live
  mana, zones and stack remain unchanged after rejection.
- **Night Soil → `TestRONSingleGraveyardExileCostIndependentActivations`**:
  separate RON factory instances and two pending activations each, eight distinct
  paid cards and independent saved token controllers.
- `TestRONSingleGraveyardExileCostInvalid`: malformed and negative counts,
  dynamic X, tuple player symbols, unsupported player selectors, empty and
  incompatible predicates, operations, duplicate costs, wrong execution contexts
  and attempts to read a singular result from the plural producer.
- `TestRONSingleGraveyardExileCostInventory`: semantic roles, accepted complete
  record, rejected mixed collection and unchanged pre-existing registry.
- Native `TestSingleGraveyardExileCostSelection`: payer choices across eligible
  graveyards, distinct same-graveyard subjects, exact payment timing, ownership,
  pure queries and clone isolation.
- Native `TestSingleGraveyardExileCostRollback`: split resources, invalid,
  duplicate and cross-graveyard choices, absent/returned/changed-type visits,
  overlapping cost claims and life rollback before live commitment.
- Native `TestSingleGraveyardExileCostBatch`: all selected cards in exile before
  zone observers, two zone events, one full-count graveyard batch and holder facts.
- Native `TestSingleGraveyardExileCostDefinitionAndZero`, `ChoiceVisit` and
  `PlayerSetAndAlternative`: invalid declarations, zero/absent payer, return during
  offered choice, opponent-only native selection and locked `EitherCost` reuse.
- Retained `TestGraveyardExileCost*`, `TestExileDamageSourceStackBindings` and
  `TestRONExilePayment*` pass for current plural/singular own-graveyard callers.

### Verification and measured outcome

Required verification ran in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONSingleGraveyardExileCost -count=1`
  passed. Native `go test ./pkg/mage -run
  'TestSingleGraveyardExileCost|TestGraveyardExileCost|TestExileDamageSourceStackBindings'
  -count=1` and the retained focused RON exile payment tests passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's `go test ./... -short -count=1` passed.
- `make lint` passed with **0 issues**. Automatic import grouping changes in
  the new files were reviewed. No behavior or assertions changed. The focused
  RON and native tests were rerun after formatting and passed.
- `git diff --check` passed before final loader validation and after this ledger
  update.
- Final `go run ./cmd/loadcards -json` exited **1**, as required while gaps
  remain: **2,205 compiled / 240 rejected records**, **858 accepted / 128 rejected
  names**. Comparison with the baseline confirms that only Night Soil's issue
  disappeared and no other diagnostic changed. This is collection rejection,
  not failed native validation.

Estimated and actual gains match: **one unique complete name / one record**.
Complete RON gameplay coverage gains Night Soil. Compilation acceptance now
measures **858 accepted / 128 rejected names** (986 total) and **2,205 compiled /
240 rejected records** (2,445 total). Only Night Soil's diagnostic disappeared;
all other baseline diagnostics exactly match. The strict loader still exits 1
and publishes no partial collection. Compilation and gameplay evidence are
reported separately. Full collection gameplay, runtime loading and application
cutover remain open; no Go definitions, generators or DSL exports were removed.

Changed files: new engine `pkg/mage/single_graveyard_exile_cost.go` and its tests;
`cost.go`, `payment_transaction.go`, `doc.go`; new RON
`single_graveyard_exile_cost.go` and its tests; `exile_payment.go`,
`activation_timing.go`, `inventory.go`; collection count tests in
`definition_plan_test.go`, `inventory_test.go`, `numeric_consumers_test.go`;
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger.

### Ranked next candidates

1. **Shared entry color choice and typed chosen-color predicates — Psychic
   Allergy and Jihad.** Both complete records require an entry color and later
   color reads. Jihad also combines its opponent choice with the color choice
   and uses a conditional layer/state trigger. Establish the shared entry choice
   binding before dependent predicates. Estimated zero immediate unlocks for a
   choice-only prerequisite, with two concrete downstream names. Medium effort;
   assess the smallest complete shape and avoid partial combined-entry support.
2. **Mandatory entry exile payment using announced X — Frankenstein's Monster.**
   Matching selection, exact visits and complete-batch exile now have a native
   service. Entry scope, required transaction/result and failed entry remain,
   followed by combination-counter distribution. Estimated zero immediate
   unlocks for the transaction slice; medium effort.
3. **Restricted casting X payment — Drain Life.** Preserve fixed casting symbols
   while restricting only X through the shared solver. Six reprints still need
   clamped actual-damage life gain. Estimated zero immediate unlocks; medium
   effort. Recompare other shared capabilities before selecting.

This run stops after matching plural single-graveyard activation exile payment.

## 2026-10-13 — ENTRY-COLOR-CHOICE

Status: implementation complete; final verification results are recorded below.
This run implements one dependency-ready prerequisite and stops.

### Candidate comparison and selection

The working tree was clean before edits. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -short -count=1` passed. Fresh strict loading exited
1 with **2,205 compiled / 240 rejected records**, **858 accepted / 128 rejected
names** (2,445 records / 986 names). A reused temporary output initially showed
older counts; a new task-specific loader output confirmed the current baseline.
No build, test or environment failure was present.

Complete unchanged RON records were inspected, including all nested forms:

| Candidate | Distinct affected names | Estimated complete unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Entry color choice | Psychic Allergy, Jihad (2 names / 2 records) | 0 names / 0 records | Supplies a saved source color before dependent predicates. Psychic Allergy needs its `Opponent` upkeep selector and `IsColor(TheChosenColor)` in its upkeep count. Jihad needs combined color/opponent entry, chosen-color conditions, conditional global P/T and a chosen-player state event. | Small to medium: one entry service, lowering/validation and lifecycle tests. |
| Mandatory entry exile payment with X | Frankenstein's Monster (1 name / 1 record) | 0 names / 0 records | Existing graveyard selection, exact visits and batch exile help, but entry payer/required transaction, payment-result scope, failed entry and chosen combination-counter distribution remain. Enables the full entry-payment consumer. | Medium to high: transaction integration, rollback and result-dependent entry. |
| Restricted casting X | Drain Life (1 name / 6 records) | 0 names / 0 records | Requires X-only mana restrictions while preserving fixed symbols. Clamped actual damage life gain remains after casting support. Enables a shared restricted-X solver contract. | Medium: solver metadata, casting payments, copying and rollback tests. |

Ledger candidate #1 is dependency-ready. It has two concrete downstream unique
cards and a smaller complete contract than the payment candidates. The chosen
slice is entry color choice alone; it does not begin chosen-color predicates or
combined entry lowering. No full-card unlock was estimated. Reprints were
counted once. `ChooseAColor/1`, `TheChosenColor/0` and their examples were read
from the vocabulary snapshot using targeted searches. Alchor's Tomb uses the
existing resolving choice adapter, not the selected entry context; its full
record was inspected and is unaffected.

### Established shape, ownership and dependencies

RON accepts exactly
`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor)])` on a permanent.
`ChooseAColor` is a tuple with one bare `AnyColor` argument. There is one entry
declaration and one operation. The declaration validator rejects nonpermanent
sources, wrong arity, wrong source kinds, unsupported filters, duplicates,
additional entry declarations and combined operations. Unbound/incompatible
chosen-color consumers stay rejected. Source positions remain in diagnostics.
The semantic inventory classifies the entry argument as `color_choice_filter`.

The new reusable engine API is
`ChooseColorAsEnters() AsEntersBattlefieldAbility`. It uses the same pre-entry
service as `ChooseOpponentAsEnters` and `ChooseBasicLandTypeAsEnters`, not the
legacy post-registration ETB color helper. The entry controller makes one
`ChooseManaColor` choice before battlefield registration and entry events.
Only the five Magic colors are legal; invalid answers or absent choosers use
white. `Permanent.ChosenColor` stores a typed `core.Color` per exact visit.
The factory makes fresh ability declarations; no mutable choice is captured
by the compiler and no global result slot is added.

Shared copy replacements run first; acquired entry rules then choose once.
Copies do not inherit the donor's saved color. Shared entry batches prepare all
choices before registration and events. Control changes and phasing keep the
choice. Return starts a new visit and chooses again. Cloned games keep separate
fields and pending entries choose independently. The operation makes no payment,
adds no stack object and does not run at cast announcement or failed payment.
Definition inspection marks the service as a battlefield-source entry ability.

Dependencies reused: `AsEntersBattlefieldAbility`, shared battlefield entry,
copy replacement order/provenance, definition plans and validation, permanent
visit identity, controller lookup, phasing, native casting and game cloning.
RON owns syntax, shape restrictions, diagnostics and inventory roles. The engine
owns player decisions and permanent state. No engine reader/predicate or LKI
consumer is added in this slice; those need a separate typed information
contract. Existing Go callers and resolving color-choice adapters are retained.

### Source and rules evidence

`cards/mtgish/DRK.lines.ron:83` contains Psychic Allergy's single color entry
operation. `cards/mtgish/ARN.lines.ron:44` contains Jihad's combined color/opponent
entry operation. Retained `data/DRK.json` and `data/ARN.json` confirm the exact
Oracle choice text. CR 105.1 and 105.4 require one of the five colors; CR 614.1c,
614.12 and 614.12a require the choice before entry. CR 400.7 and 702.26 establish
new-visit and phasing behavior. No source defect was found and no RON source
changed. Vocabulary source hashes/spellings need no refresh.

### Failing-first evidence and card-to-test mappings

Test fixture API errors were corrected before recording semantic failures.
`TestChooseColorAsEnters*` then failed to compile solely because
`ChooseColorAsEnters` was absent. `TestRONColorEntryGameplay` and
`TestRONColorEntryInventoryAndRealRequirements` failed because `ChooseAColor`
was an unsupported entry operation. The new service and adapter passed these
cases. Later fixture corrections used the existing owner/hand and printed-copy
provenance contracts; no engine behavior or assertion was weakened.

- **Psychic Allergy → `TestRONColorEntryInventoryAndRealRequirements`**:
  inspects the complete unchanged record, executes its exact entry declaration
  through the engine and keeps the complete record rejected at its unsupported
  `Opponent` upkeep selector. The chosen-color predicate also remains. This is entry coverage, not complete card gameplay.
- **Jihad → `TestRONColorEntryInventoryAndRealRequirements`**: inspects the
  complete unchanged record and proves that combined entry remains rejected.
  No Jihad gameplay gain is claimed.
- `TestRONColorEntryGameplay`: strict fixture loading, native failed/paid casting,
  announcement versus resolution timing, independently resolved pending clones,
  phasing and fresh return choices.
- `TestRONColorEntryInvalid`: malformed source/operation/filter, fixed and
  colorless filters, duplicates, extra/combined entry declarations, nonpermanent
  context and unsupported/incompatible reads.
- `TestRONColorEntryInventoryAndRealRequirements`: entry semantic roles,
  accepted synthetic shape, complete-record boundaries and atomic rejection
  without replacing an existing registry.
- `TestChooseColorAsEntersTimingAndIdentity`: all five legal colors, colorless,
  wildcard, negative and out-of-range answers; controller versus owner,
  pre-registration timing, no stack object, phasing, control, clone isolation
  and a fresh returned incarnation.
- `TestChooseColorAsEntersCopyAndMissingPlayer`: independent copied definitions,
  fresh choice rather than donor color, missing chooser and nonpermanent
  definition rejection.
- `TestChooseColorAsEntersCopyReplacementAndBatch`: copy replacement before
  acquired entry choice, one choice per copy, independent source state and all
  batch choices/entries visible before each entry event.

### Verification and measured outcome

Required verification completed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONColorEntry -count=1` passed.
  Native `go test ./pkg/mage -run TestChooseColorAsEnters -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's `go test ./... -short -count=1` passed.
- `make lint` passed with **0 issues**. Automatic formatting was reviewed;
  no behavior or assertions changed. Focused RON/native tests passed again.
- `git diff --check` passed before final loading and after the ledger update.
- Final `go run ./cmd/loadcards -json` exited **1**: **2,205 compiled / 240
  rejected records**, **858 accepted / 128 rejected names**. This is required
  collection rejection while unsupported records remain, not a native failure.

Baseline/final comparison has no added or removed rejected card/file keys.
Only two messages changed: Psychic Allergy moved from unsupported entry choice
at DRK:83:195 to unsupported `Opponent` upkeep filter at DRK:83:261; Jihad's
combined entry rejection now names the color-choice shape at ARN:44:142 instead
of its opponent-choice shape. All other diagnostics are identical. The newly
exposed Psychic Allergy event adapter is a separate blocker before its dynamic
color predicate; it does not change this slice's zero-unlock estimate.

Estimated and actual full-card gains match: **zero unique names / zero records**.
Compilation coverage is unchanged. Gameplay contract coverage gains the entry
choice service and Psychic Allergy's exact entry declaration, with no complete
card gameplay gain claimed. The strict loader still publishes no partial
registry. No RON sources, Go card definitions, generators or DSL exports changed.
Runtime cutover and the complete migration criteria remain open.

Changed files: new `pkg/mage/color_entry_choice.go` and its native tests;
`pkg/mage/doc.go`; new `pkg/mage/dsl/ron/color_entry.go` and its tests;
`compiler.go`, `entry_tapped.go`, `inventory.go`; `docs/dsl-reference.md`,
`docs/engine-rules-contracts.md` and this ledger.

### Ranked next candidates

1. **Typed chosen-color predicates — Psychic Allergy and Jihad.** The single
   color entry prerequisite is now available. Establish exact source-color
   information reads for contextual counts/conditions, including departure,
   phasing and pending-trigger facts. Estimate zero immediate unlocks: Psychic Allergy also needs the
   opponent-upkeep selector exposed by final loader diagnostics. Jihad still
   needs combined entry, conditional layer and state-event support. Both names
   gain a concrete shared predicate prerequisite.
   Medium effort; inspect full records again before selecting.
2. **Combined color/opponent entry choice — Jihad prerequisite.** Establish a
   complete joint entry declaration and independent stored choices through
   copy order and new visits. Zero immediate full-card unlocks until its color
   condition/layer/state consumers work. Small to medium effort; prefer the
   shared predicate work first.
3. **Mandatory entry exile payment with X — Frankenstein's Monster.** Reuse
   matching graveyard selection and batch exile; add mandatory entry transaction,
   result scope and failed entry. Zero immediate unlocks before combination
   counters. Medium to high effort. Recompare other shared capabilities before
   selecting this single-card prerequisite.

This run stops after entry color choice. Runtime cutover, complete RON gameplay,
Go-card deletion and unused DSL removal remain gated.


## 2026-10-13 — RESOLVING-CHOSEN-COLOR-COLLECTIONS

### Candidate comparison and selection

Started from the latest entry's ranked candidates. The working tree was clean.
Fresh `make build` and `go test ./pkg/mage/dsl/ron ./pkg/mage -short -count=1`
passed. The baseline strict loader exited 1 with 2,205 compiled and 240 rejected
records, 858 accepted and 128 rejected unique names. This was collection
rejection, not a native build/test failure.

Read the complete RON records for Psychic Allergy (DRK:83), Jihad (ARN:44),
and Frankenstein's Monster (DRK:43), including all nested costs and conditions.
Targeted vocabulary reads found four TheChosenColor uses: Jihad twice, Psychic
Allergy once, and Alchor's Tomb's existing resolving color-setting continuation.
Alchor's Tomb and Dream Coat retain separate local resolving choice contracts;
Sleight of Mind's color-word references are a separate type.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Typed chosen-color collections/information | Psychic Allergy; Jihad | 0 | Psychic Allergy needs Opponent upkeep. Jihad needs combined entry, conditional static and state-event consumers. Exact information supports both cards' nontoken color predicates. | Medium; bounded resolving adapter, current/LKI reader and identity tests |
| Combined color/opponent entry | Jihad | 0 | Conditional color layer and no-matching-permanents state event remain. Enables both stored entry choices before static/state consumers. | Small to medium; choice order, copy provenance, visit/clone isolation |
| Mandatory entry exile with X | Frankenstein's Monster | 0 | Mandatory entry transaction/failure routing, plural payment count and combination counters remain. Enables payment-conditioned entry counters. | Medium to high; exact selections, rollback, batch events and failed entry |

Candidate 1 was dependency-ready: source entry color choice and exact information
snapshots already existed. Its shared two-card downstream benefit was concrete;
the alternatives were single-card prerequisites with more independent blockers.
Selected one complete resolving collection context, without claiming static,
target, entry or replacement support. The contract and acceptance cases were
stated before edits. No card definitions were used as requirements.

### Established contract and reuse boundary

Engine API:
`PermanentsOfChosenColor(reference PermanentRef, subjects PermanentSet) PermanentSet`.
It uses `readPermanentInformation` and frozen `PermanentLKI.Snapshot.ChosenColor`.
The exact visit supplies current, departure or phase information. Unknown visits,
unset colors, wildcard and invalid values select nothing. Return cannot replace
the saved visit. Current layered subject colors supply membership, including
multicolor subjects. Absent/phased subjects cannot match; duplicate visits are
removed. Information reads cannot restore a target's action legality.

RON shape: `IsColor(TheChosenColor)` in resolving permanent collections,
including conjunctive/disjunctive groups and battlefield counts. Requires a
permanent with the validated single `AsPermanentEnters(ThisPermanent,
[ChooseAColor(AnyColor)])` declaration. `scope.colorEntryChoice` records this
validated source producer; `scope.resolving` limits reads to action compilation.
Tuple/plural reads, missing producers, numeric uses, target predicates, static
layers/counts, entry and replacement contexts remain rejected. Inventory now
classifies the symbol as `color_choice_reference` under a color predicate.

Reused exact source/target/saved/host references, zone/phase snapshots, cloning,
shared layered colors, collection composition, trigger source capture, pending
copy frames, cost payment and shared tap/damage services. Added delayed-source
inspection of both typed collection operands; waiting queries execute no reads.
RON owns spelling, declaration/context validation and lowering. The engine owns
information and membership. No global last-action state, card-name branches,
Go fallbacks, partial registry, new card source, or DSL deletion was introduced.

### Rules and source evidence

Local `data/DRK.json` and `data/ARN.json` confirm the chosen-color count/condition
and nontoken restrictions in the complete RON records. CR 105.1–105.4 establishes
legal colors and current multicolor membership; CR 608.2h establishes current or
last-known source information; CR 400.7 establishes new objects after a zone
change; CR 702.26b/d establishes absent phased subjects and retained identity.
No source defect was found or corrected. No RON source changed, so source hashes
and the vocabulary snapshot need no refresh. Build/environment gaps were absent.
The remaining gaps are compiler/engine adapters with separate contracts.

### Failing-first evidence and card-to-test mapping

`TestChosenColorInformation*` first failed to compile because the public service
was missing. RON fixture API mistakes were corrected, then
`TestRONChosenColorFilter*` failed solely because the new validated scope producer
was missing. After implementation, an activation test used the entry ability's
index; correcting it to the activation index preserved the payment/timing
assertions. No engine behavior or assertion was weakened.

- **Psychic Allergy → `TestRONChosenColorFilterRealRequirements`**: reads the
  complete unchanged record; executes its exact entry operation and damage body
  through a supported synthetic AnyPlayer upkeep selector. Queues the event,
  returns the source with a different color, then deals two damage from the old
  visit's red nontoken count. Different subject controllers and multicolor
  membership are checked. The actual Opponent selector stays rejected; no full
  card gameplay claim is made.
- **Jihad → `TestRONChosenColorFilterRealRequirements`**: complete unchanged
  record remains rejected at combined entry. Resolving shared information is a
  prerequisite for its future static/state consumers, not Jihad gameplay proof.
- `TestRONChosenColorFilterGameplay`: strict RON loading, failed/paid activation,
  resolution timing, source departure/return/phasing, pending copies and clones,
  current controllers, late subjects and token exclusion.
- `TestRONChosenColorFilterInvalid`: unbound, malformed/plural and numeric reads,
  target predicates, static color sets and static color counts.
- `TestRONChosenColorFilterInventoryAndAtomicLoad`: semantic reference role and
  atomic rejection of mixed supported/unsupported collections.
- `TestChosenColorInformationIdentity`: exact current/LKI and phase reads, new
  visits, phase cycles, unknown/invalid colors, current layered membership,
  multicolor/colorless subjects, phased subjects and pure queries in clones.
- `TestChosenColorInformationSavedReferences`: target, saved and attachment-host
  information across a returned visit.
- `TestChosenColorInformationDeclarationsAndWaiting`: invalid definitions,
  unbound saved references, pure delayed inspection and separate clone choices.

### Verification and measured outcome

Required checks completed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONChosenColorFilter -count=1` passed.
  Native `go test ./pkg/mage -run TestChosenColorInformation -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- The engine skill's `go test ./... -short -count=1` passed.
- `make lint` passed with **0 issues**. Automatic import grouping in the new
  tests was reviewed; no behavior/assertions changed. Both focused suites passed
  again after formatting.
- `git diff --check` passed before final loading and after ledger completion.
- Final `go run ./cmd/loadcards -json` exited **1**, with **2,205 compiled / 240
  rejected records**, **858 accepted / 128 rejected names**, out of 2,445 records
  and 986 names. The final JSON is exactly equal to the fresh baseline JSON,
  including all diagnostics. This is required collection rejection while gaps
  remain, not a build or test failure.

Estimated and actual full-card gains match: **zero names / zero records**.
Gameplay contract coverage gains resolving source chosen-color collections and
Psychic Allergy's exact damage body. No complete card gameplay gain is claimed.
Compilation and gameplay progress remain separate. Accepted records keep passing
the full suite. Collection loading publishes no partial registry.

Changed files: new `pkg/mage/chosen_color_information.go` and native tests;
`pkg/mage/waiting_delayed_source.go`, `pkg/mage/doc.go`; new
`pkg/mage/dsl/ron/chosen_color_filter_test.go`; RON `compiler.go`,
`contextual_collection.go`, `color_entry.go`, `inventory.go`;
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger.
No RON assets, Go card definitions, generators or broad DSL exports changed.
Removal gates remain unmet, so no DSL deletion/caller audit was needed.

### Ranked next candidates

1. **Combined color/opponent entry — Jihad prerequisite.** Reuse both entry
   services and the exact information read. Define one complete joint declaration
   with choice order, copies, returns and independent storage. Zero immediate
   unlocks before conditional static color and state-event consumers. Small to
   medium effort. Recompare other shared candidates before selecting this
   single-card prerequisite.
2. **Opponent upkeep selector — Psychic Allergy.** Its resolving chosen-color
   count now works. Existing MayCost and fixed two-Island sacrifice adapters
   cover the other upkeep body. Estimate one full-card unlock, subject to full
   gameplay tests, source/controller event timing and independent pending events.
   Small effort; defer while useful multi-card work remains.
3. **Mandatory entry exile with announced X — Frankenstein's Monster.** Matching
   graveyard selection and batch exile exist; mandatory entry transaction,
   failure/result routing and combination counters remain. Zero immediate
   unlocks. Medium to high effort; recompare shared work first.

This run stops after resolving chosen-color collections. Complete collection
execution, runtime cutover, Go-card deletion and unused DSL cleanup remain gated.


## 2026-10-13 — COMBINED-COLOR-OPPONENT-ENTRY

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed before implementation. Fresh strict
loading exited 1 with 2,205 compiled / 240 rejected records and 858 accepted /
128 rejected names. This is collection rejection, not a build/test failure.
Complete RON records were inspected for Jihad, Psychic Allergy and Frankenstein's
Monster, including every nested entry operation and later consumer. Targeted
vocabulary entries confirmed the entry, color/player choice and chosen-reference
shapes. Complete records with these shared choice adapters were also inspected:
Black Vise, The Rack, Cursed Rack, Alchor's Tomb, Rainbow Vale, Backdraft and
Rohgahh of Kher Keep. Reprints count once.

| Candidate | Distinct directly affected names | Estimated full-card gains | Other blockers and downstream capabilities | Estimated implementation/test effort |
| --- | --- | --- | --- | --- |
| Combined color/opponent entry, ranked #1 | Jihad | 0 names / 0 records | Must precede Jihad's conditional static chosen-color membership and captured chosen-player state detection. Reusable required sequencing also supplies ordered child execution and entry rejection for future mandatory entry procedures such as Frankenstein's Monster | Small/medium: one sequence service, strict joint lowering and lifecycle/copy/batch contracts |
| Opponent upkeep selector, ranked #2 | Psychic Allergy | 1 name / 1 record | Complete record's entry, resolving nontoken color count and optional two-Island sacrifice already have primitives. Needs controller-relative event detection, independent event capture and upkeep text compatibility | Small/medium: event adapter/service and full-card event/payment tests |
| Mandatory announced-X entry exile, ranked #3 | Frankenstein's Monster | 0 names / 0 records for payment prerequisite alone | Needs mandatory entry transaction, exact selected graveyard visits, payment-result/failure routing and combination-counter placement. Supplies future result-bearing entry payments | Medium/high: several distinct payment/result/counter contracts; exceeds the bounded joint-choice slice |

Selected ranked #1 under the dependency-ready unblocking-prerequisite rule.
Its exact joint entry is required before Jihad's dependent consumers. The
reusable engine sequence is also usable by later multi-operation entry contracts;
this run does not implement those costs or results. No dependency-ready
multi-name full-card unlock was established among the compared candidates.
Estimated and actual full-card gains match at zero. No second slice was started.

### Signatures, supported contexts and reuse boundary

New engine API:
`RequiredEntryOperations(operations ...AsEntersBattlefieldAbility) AsEntersBattlefieldAbility`.
One required ability runs its children in order before battlefield registration.
No optional offer is made. A false child stops later children and rejects entry
through the existing failed-entry destination. The input slice is copied;
empty bodies, nil/typed-nil children, invalid nested declarations and
nonpermanent sources fail definition validation.

New RON shape:
`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAPlayer(Opponent)])`.
Only this order and these filters are accepted, in one source entry declaration
on a permanent. Wrong arities, malformed symbols, reversed order, duplicate
choices, extra entry operations, additional entry declarations and optional
joint choices reject at compilation. Existing standalone choices keep working.

RON owns declaration validation, scope producers and lowering. Every factory
creates fresh child abilities. The engine reuses `ChooseColorAsEnters`,
`ChooseOpponentAsEnters`, the shared entry/copy pipeline, permanent choice fields,
exact information snapshots, existing source reference and chosen-player upkeep
services, casting/payment, stack copying and cloning. No additional player,
replacement, zone, cost or result system was introduced. The pair stores color
and player in separate per-visit fields; pending reads cannot follow a returned
source. No new global last-action state, reflection, Go fallback, card-name
branch, partial registry, runtime declaration panic or skipped card rule exists.

The new consumer test exposed an invalid opt-in to the older whole-object upkeep
text-revision path: that path supports a standalone opponent choice. Its RON
shape detector now preserves that exact boundary. The joint pair can use
ordinary chosen-player upkeep and resolving chosen-color collections without
opting into unsupported whole-object text revision. Static/state chosen-color
consumers, joint text revision, entry payment/results and other entry pairs
retain separate contracts and remain rejected.

### Rules, source evidence and gap classification

Local `data/ARN.json` confirms Jihad's color-and-opponent choice and nontoken
condition. CR 105.1/105.4 limits color choices; CR 109.5 identifies the entering
controller; CR 614.1c/614.12a requires pre-entry choices; CR 616.1c establishes
copy replacement priority; CR 707.2 excludes donor choices from copied values;
CR 400.7 and 702.26b–d distinguish new visits from phase cycles.

The missing ordered sequence was an engine composition contract; the joint
shape and declaration validation were missing compiler adapters. Jihad still
needs its chosen-player static predicate and chosen-color static/state consumers.
Psychic Allergy's remaining diagnostic is the opponent-upkeep selector. No
source-data defect was found or corrected. No RON source changed, so no source
hash or vocabulary snapshot refresh was required. No environmental/build
failure blocked the slice.

### Failing-first evidence and card-to-test mappings

- `TestRequiredEntryOperations*` first failed compilation with
  `undefined: RequiredEntryOperations`.
- `TestRONCombinedEntryChoiceGameplay` and `Jihad` first failed on
  `color entry requires one source choice`.
- `TestRONCombinedEntryChoiceConsumers` then failed definition validation on
  `upkeep object text contains an unsupported sibling rule`; restricting the
  old text opt-in to its established standalone shape fixes that failure.
- A native timing assertion initially used `FindPermanent`, which intentionally
  exposes the proposed entering permanent. It was corrected to inspect
  `AllBattlefield` for registration, retaining the order and timing contract.
- Former rejection tests for the exact newly supported pair were changed to
  acceptance and unsupported extra-operation coverage. Complete Jihad still
  must fail at its remaining static predicate. No unsupported full card or
  skipped instruction is accepted.

**Jihad → `TestRONCombinedEntryChoiceJihad`** reads the complete unchanged ARN
record, asserts its remaining static diagnostic, validates both source producers
and executes the exact unchanged entry subrule with the record's metadata.
This is entry gameplay coverage, not complete Jihad gameplay coverage.

`TestRONCombinedEntryChoiceGameplay` uses strict RON loading and native unpaid/
paid casting, no announcement choice, permanent spell copies, independent pending
game clones, different entry controllers/owners, factory instances, control
changes, phase cycles and return choices. `Invalid` rejects every unsupported
joint declaration/context. `Jihad` also proves atomic mixed-collection rejection.
`Consumers` proves chosen-player event timing and an exact old color read after
source departure and return; the new source's blue choice cannot replace red.

Native `TestRequiredEntryOperationsOrderAndFailure` covers required sequencing,
input-slice isolation, pre-registration timing, no optional offer, early stop
and the existing failed-entry graveyard destination. `Declarations` covers nil,
typed-nil, empty/nested and invalid source definitions. `ChoiceOrder` checks
color-before-opponent, owner/controller differences, clone field isolation,
phasing and new-visit return. `CopyReplacementAndBatch` checks acquired joint
choices after copy replacement and all choices/registrations before batch events.
Existing color, opponent-entry and chosen-color consumer tests remain passing.

### Verification and measured outcome

Required checks completed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONCombinedEntryChoice -count=1` passed.
  `go test ./pkg/mage -run TestRequiredEntryOperations -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
  The engine skill's `go test ./... -short -count=1` also passed.
- `make lint` passed with 0 issues. Automatic formatting was reviewed; no
  behavior or assertion changes were introduced.
- `git diff --check` passed, including the completed ledger update.
- Final `go run ./cmd/loadcards -json` exited 1 with **2,205 compiled / 240
  rejected records** and **858 accepted / 128 rejected names**, out of 2,445
  records and 986 names. Only Jihad's diagnostic changed from its entry
  declaration to `unsupported static condition player (TheChosenPlayer)`.
  Accepted-record coverage and all other diagnostics are unchanged. The loader
  continues to reject the collection atomically.

Actual gains: **zero full names / zero records**, matching the estimate.
Gameplay contract coverage adds Jihad's exact joint entry, ordered required
engine entry operations and existing typed consumers of both saved choices.
Compilation counts do not measure that gameplay gain. Complete Jihad gameplay,
collection startup, runtime cutover, Go-card deletion and DSL cleanup remain
unfinished. No removal gate was reached; no DSL function was deleted.

Changed files: new `pkg/mage/required_entry.go`, `required_entry_test.go`;
new RON `combined_entry_choice.go`, `combined_entry_choice_test.go`;
RON `color_entry.go`, `color_entry_test.go`, `opponent_entry.go`, `entry_tapped.go`,
`text_upkeep_object.go`; `pkg/mage/doc.go`, `docs/dsl-reference.md`,
`docs/engine-rules-contracts.md` and this ledger. No card source, generator,
production registry or broad DSL export changed.

### Ranked next candidates

1. **Opponent upkeep selector — Psychic Allergy.** One estimated full name /
   one record; the complete entry, damage count and Island-sacrifice primitives
   exist. Establish controller-relative event detection, independent event
   captures and compatible text boundaries. Small/medium effort; recompare
   shared multi-card candidates first.
2. **Live chosen-player/chosen-color static membership — Jihad.** The joint entry
   prerequisite is complete. A reusable current source-choice membership service
   can supply its conditional static bonus; state detection still needs a separate
   adapter. Zero immediate unlocks for the static prerequisite. Medium effort.
3. **Mandatory entry exile with announced X — Frankenstein's Monster.** Ordered
   required child execution now exists. Entry transaction, exact graveyard
   choices, result/failure routing and combination counters remain. Zero immediate
   gains from the payment prerequisite; medium/high effort.

This run stops after combined color/opponent entry. Complete migration remains
gated by the remaining collection, gameplay, runtime and removal work.

## 2026-10-13 — OPPONENT-UPKEEP-EVENT

### Baseline and candidate selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The fresh strict loader exited 1
with 2,205 compiled / 240 rejected records and 858 accepted / 128 rejected names.
This is collection rejection, not a build or test failure. An intermediate read
of the output file occurred before the loader finished; the completed output
matches the previous ledger. No environment failure was found.

Read AGENTS.md, the engine reference and execution design, current ledger status
and latest entry, targeted upkeep/payment vocabulary entries, selected compiler
and engine services, and complete unchanged Jihad, Psychic Allergy and
Frankenstein's Monster records. Reprints were counted once. Compared the latest
ranked candidates:

| Candidate | Distinct affected names | Estimated full names / records | Other blockers and downstream benefit | Effort |
| --- | --- | --- | --- | --- |
| Opponent upkeep selector | Psychic Allergy | 1 / 1 | No remaining rule blocker: source color entry, live resolving color count, typed event player, damage, optional two-Island sacrifice and destruction exist. Adds a reusable controller-relative event capture service. | Small/medium |
| Live chosen-player/color static membership | Jihad | 0 / 0 for static prerequisite | Joint entry exists. Live chosen-color static membership and chosen-player predicate must compose; the no-matching-permanent state trigger needs its own adapter. Enables its conditional global bonus and later state consumer. | Medium |
| Mandatory entry exile with announced X | Frankenstein's Monster | 0 / 0 for payment prerequisite | Required sequencing exists, but entry payment preparation, exact graveyard choices, result/failure routing and combination counters remain. Reuses shared graveyard payments and enables entry result consumers. | Medium/high |

Selected the first ranked candidate because it is ready, completes a card in one
bounded slice, and needs no speculative entry or static subsystem. Neither of
the compared choice/payment prerequisites supplies a ready multi-card gain.
The other candidates retain their full nested blockers; first-error counts were
not treated as unlock estimates. The single-name result is explicit.

### Contract, signatures and reuse boundary

Engine API: `OpponentUpkeep(player *PlayerBinding) EventPattern`.
RON shape: `TriggerA(AtTheBeginningOfAPlayersUpkeep(Opponent), body)` in ordinary
permanent trigger rules. Bare `Opponent` selects relative to the current source
controller at detection. Each factory and declaration gets a fresh
`stepEventPlan` and `PlayerBinding`. `Trigger_ThatPlayer` is local to the body;
`You` remains that trigger's saved controller. Resolution reads current subjects,
not a set captured at detection. Independent queued upkeeps keep separate facts.

The engine composes existing `EachUpkeep`, `EventPlayerIsOpponent` and
`turnStepEventPattern` player capture, inspection and definition validation.
Existing trigger placement, APNAP, source incarnation/LKI, stack copies, delayed
capture, resolution frames and clone services own execution. The compiler owns
selector spelling, scope and fresh lowering. No parallel event, payment, zone,
damage or choice system was added. No card-name branch, Go fallback, partial
registry, global result slot, skipped instruction or runtime declaration panic
was introduced. Existing callers remain unchanged.

Nil/unnamed bindings and reads from another ability are rejected. RON rejects
`Opponent()`, `SinglePlayer(Opponent)`, extra selector arguments, unbound player
and permanent reads, opponent end/draw steps and opponent upkeep `TriggerI`.
Granted-upkeep validation keeps its existing controller-only boundary. Opponent
upkeep does not opt into the older printed-upkeep text services. Ordinary
controller, any-player and chosen-player text paths remain tested. Psychic
Allergy's controller-upkeep declaration continues to use its existing text path;
text inventory tests now include that newly compiled declaration.

### Rules evidence and source classification

Local `data/DRK.json` confirms Psychic Allergy's color choice, each opponent's
upkeep, nontoken condition, chosen-color count and two-Island upkeep alternative.
CR 109.5 and 603.3a establish the controller context; CR 503.1a and 603.2b
establish upkeep detection and stack timing. CR 113.7a preserves pending abilities
after source changes. CR 400.7 separates returned source visits. Existing color
choice, sacrifice/payment and damage contracts supply their rules unchanged.

The missing pieces were a typed engine event composition and a compiler
selector adapter. No source-data defect was found or corrected. No RON source
changed, so no vocabulary source/hash refresh was needed. Runtime cutover and
both removal gates remain closed. No DSL function or Go card was removed.

### Failing-first evidence and card-to-test mapping

- Native `TestOpponentUpkeep*` first failed compilation with
  `undefined: OpponentUpkeep`.
- `TestRONOpponentUpkeepGameplay` and `PsychicAllergy` first failed with
  `unsupported upkeep player filter (Opponent)`.
- New fixtures initially used two wrong API names for token/keyword setup;
  these were corrected to existing engine APIs before behavioral verification.
- Earlier color tests now require full Psychic Allergy compilation while Jihad
  remains rejected. Its older damage requirement test now uses the exact
  opponent selector instead of substituting AnyPlayer.
- Definition-plan and semantic inventory counts increased by one. Upkeep text
  inventories increased from 60 names / 122 records to 61 / 123 because the
  newly accepted card also has an existing controller-upkeep declaration.
  Explicit Psychic Allergy membership checks accompany those updated totals.

**Psychic Allergy → `TestRONOpponentUpkeepPsychicAllergy` and
`TestRONOpponentUpkeepPsychicAllergyPayment`.** Both execute the complete
unchanged DRK record. The first covers casting before entry choice, saved red
choice, opponent detection, nontoken/color/controller membership, current counts
at resolution, prevention, control changes, source departure/return with a new
blue choice, phasing, ability removal, stack copies and game clones. The second
covers optional full two-Island payment, decline, insufficient/wrong/phased
subjects without partial spending, owner graveyards, pending control changes,
departure, source return, pending phasing, regeneration and indestructibility.
The upkeep branch destroys rather than sacrifices the source.

`TestRONOpponentUpkeepGameplay` queues opposite players' upkeeps across a control
change, then returns the source before resolving both in independent game clones.
`Invalid` checks malformed selectors, unsupported contexts and lexical scope.
`CollectionAndTextBoundary` strictly loads the complete real record, checks
atomic rejection of a mixed invalid collection, tests separate factory instances,
and verifies rejection by both whole-object and trigger text-revision services.

Native `TestOpponentUpkeepCapture` checks detection versus resolution, independent
queued players/controllers, source return, saved delayed players and clone frames.
`SourceLifetime` covers phase suspension/restoration, ability removal before and
after detection, pending phasing and copied event-player capture. `Declarations`
checks nil/unnamed and cross-ability binding rejection. Existing shared cost,
batch sacrifice, replacement and damage tests remain the mechanics contracts.

### Verification and measured result

Required checks passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONOpponentUpkeep -count=1` and native
  `go test ./pkg/mage -run TestOpponentUpkeep -count=1` passed.
  The broader color-entry, chosen-color, combined-entry and upkeep-text focused
  run also passed after the measured inventory assertions were updated.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
  The engine skill's `go test ./... -short -count=1` also passed.
- `make lint` passed with 0 issues. Automatic formatting was reviewed; no
  behavior or assertion changes were introduced. Both new focused groups were
  rerun afterward and passed.
- `git diff --check` passed, including the final ledger update.
- Final `go run ./cmd/loadcards -json` exited 1 with **2,206 compiled / 239
  rejected records** and **859 accepted / 127 rejected names**, out of 2,445
  records and 986 names. Comparing every final diagnostic with the fresh baseline
  removes only Psychic Allergy's `unsupported upkeep player filter (Opponent)`.
  No diagnostic was added or changed; accepted records remain accepted.
  The loader continues to reject the collection atomically.

Actual gain is **one full name / one record**, matching the estimate. Compilation
coverage and gameplay evidence are separate: the new complete-record tests prove
Psychic Allergy's three rules, while the collection counts establish only
compilation and definition-validation coverage. Full collection gameplay,
embedded startup, application cutover, Go-card deletion and DSL cleanup remain
unfinished. No removal gate was reached.

Changed files: new `pkg/mage/opponent_upkeep.go`, `opponent_upkeep_test.go` and
`pkg/mage/dsl/ron/opponent_upkeep_test.go`; RON `end_step.go`,
`color_entry_test.go`, `chosen_color_filter_test.go`, `definition_plan_test.go`,
`inventory_test.go`, `numeric_consumers_test.go`, `text_upkeep_layer_test.go` and
`text_upkeep_trigger_test.go`; `pkg/mage/doc.go`, `docs/dsl-reference.md`,
`docs/engine-rules-contracts.md` and this ledger. No card source, generator,
production registry or broad DSL export changed.

### Ranked next candidates

1. **Live chosen-player/chosen-color static membership — Jihad.** The joint
   entry prerequisite is complete. Its conditional global P/T bonus needs live
   choice predicates; its no-matching-permanent state event remains a separate
   adapter. Zero immediate full-card gain from static membership alone; medium
   effort. Recompare shared multi-card candidates before selecting.
2. **Mandatory entry exile payment/result — Frankenstein's Monster.** Required
   sequencing and announced entry X exist. Exact graveyard payment, entry-local
   outcomes/failure and combination counter choice remain. Zero immediate gains
   from the payment prerequisite alone; medium/high effort.
3. **Persistent number choices — Shapeshifter.** Complete ATQ record inspected:
   entry choice 0–7, optional controller-upkeep rechoice, and live power/chosen
   number with toughness 7 minus that number. Per-visit storage must join entry,
   resolving update and static reads. One potential name / record only after
   all three contexts exist; medium/high effort. This run did not implement it.

This run stops after opponent upkeep. Complete migration remains gated by the
remaining collection, gameplay, runtime and removal work.

## 2026-10-13 — LIVE-ENTRY-CHOICE-STATIC-MEMBERSHIP

### Baseline, candidate comparison and selection

The working tree was clean. Fresh `go test ./pkg/mage/dsl/ron -count=1`
and `make build` passed. Fresh `go run ./cmd/loadcards -json` exited 1 with
2,206 compiled / 239 rejected records and 859 accepted / 127 rejected names.
This was collection rejection, not a build or test failure.

The latest ranked candidates were compared from their complete records:

| Candidate | Distinct affected names | Estimated full-card gains | Other blockers and downstream capability | Effort |
| --- | --- | --- | --- | --- |
| Live chosen-player/color static membership | Jihad (ARN:44) | 0 names / 0 records | Enables exact global +2/+1 bonus; chosen-player no-permanent state event remains | Medium |
| Mandatory entry exile payment/results | Frankenstein's Monster (DRK:43) | 0 from payment alone | Ordered entry and announced X exist; exact graveyard payment, entry-local outcomes, failure routing and chosen combination counters remain | Medium/high |
| Persistent number choices | Shapeshifter (ATQ:61) | 1 name / 1 record after all contexts | Entry 0–7 choice, optional upkeep rechoice and live power / 7-minus-choice toughness need one per-visit lifetime | Medium/high |

Candidate #1 was selected as the ranked dependency-ready prerequisite. Joint
entry choices already exist, and live membership is a small complete boundary
that unblocks Jihad's static rule and supplies a condition reusable by later
state detection. Neither other ranked candidate offers an immediate multi-card
unlock; both require more new entry state before their complete cards work.
Reprints are counted once. No frequency-based unlock estimate was used.

The pre-edit contract was a pure live query: keep the source's entry choices,
read current subject controller/color/token status, exclude phased objects,
reject stale source visits, and leave state-event adaptation separate. RON owns
syntax, declaration validation and choice scope. The engine owns membership
and identity; existing entry and layer services keep their contracts.

### Exact supported shapes, dependencies and reuse

In static `If`, `Unless` and `IfElse`, support:

- `PlayerPassesFilter(You | TheChosenPlayer, ControlsA(filter))`.
- The same player forms with `ControlsNo(filter)`.
- Existing static fixed filters, plus `IsColor(TheChosenColor)` alone or in
  nonempty, nested `And` conjunctions with supported fixed filters.

`TheChosenPlayer` requires the source's validated opponent entry declaration.
`TheChosenColor` requires its validated five-color entry declaration. A resolving
choice, tuple read or loop/event player cannot supply these static choices.
Fixed `Or` filters keep their support. Chosen-color disjunctions, target filters,
replacement contexts and the chosen-player state-event adapter remain rejected.
The existing conditional body/layer grammar is unchanged.

Engine signatures:

- `PlayerControls(player PlayerRef, filter PermanentFilter) StateCondition`
  now also accepts `ChosenPlayerOf(This)`. Existing controller, active-player
  and saved-player paths use the same shared reader with unchanged semantics.
- `PlayerControlsChosenColor(player PlayerRef, filter PermanentFilter)
  StateCondition` accepts `You`, `ActivePlayer` or `ChosenPlayerOf(This)`.

The new condition composes `PlayerControls` with `HasColorFilter`; it does not
create another battlefield or layer service. `resolveStatePermanent` checks
present/unphased source identity and `StateContext.SourceIncarnation`.
`FilterBattlefield`, `While`, `ModifyEachPT`, entry choice services and existing
layer scheduling provide current control/color checks and global P/T changes.
No global result slot, mutable compiler capture or card-name branch was added.

An invalid/missing color or player cannot satisfy positive membership. A departed
or phased source supplies no choice; old contexts cannot read a returned source.
Static queries never use departure LKI. Control changes keep the chosen player;
source phase-in restores the same choices. Current subject changes update
membership. Return makes new entry choices, and clones/instances keep separate
fields. Ability removal disables the static rule through the existing service.
Payment, replacement ordering, simultaneous operations and spell-copy entry
contracts were not changed.

### Rules evidence and source classification

Local `data/ARN.json` confirms Jihad's color/opponent entry choice, all white
creatures' +2/+1, nontoken membership and separate sacrifice state trigger.
CR 611.3a–b requires live static membership; CR 613.1b/e–g and 613.4c put control
and color before the P/T bonus and honor prior ability removal. CR 400.7 separates
returned visits. CR 702.26b–d excludes phased objects without changing identity.
CR 603.8 establishes the distinct state-trigger lifecycle that remains a gap.

This slice fills missing compiler adapters and an engine condition contract.
No source defect or environmental failure was found. No RON source changed;
the vocabulary source/hash snapshot needs no refresh. Runtime/removal gates
remain closed. No Go card, generator or broad DSL function was removed.

### Failing-first evidence and card-to-test mapping

- `TestRONChosenStaticJihadRequirement` first failed at
  `unsupported static condition player (TheChosenPlayer)`.
- Native `TestChosenStaticCondition*` failed compilation with
  `undefined: PlayerControlsChosenColor`. Initial fixture filter/phasing names
  were corrected to existing `Not(IsToken)` and the phase ID-list API before
  confirming this intended failure.
- The existing combined-entry Jihad requirement now expects its next actual
  rejection, `unsupported no-land trigger player reference (TheChosenPlayer)`.
  Its entry coverage remains intact.

**Jihad → `TestRONChosenStaticJihadRequirement`.** The complete unchanged ARN
record is inspected and must remain rejected at the state-event gap. A test-only
projection of its exact entry and static rules, with unchanged metadata, executes
actual casting and entry before testing the global bonus. It covers both players'
white creatures, nonwhite exclusion, nontoken and artifact membership, live
control/color changes, late subjects, source/subject phasing, source control,
ability removal/restoration, entry return, separate factories and game clones.
This is entry/static subrule evidence, not complete Jihad gameplay coverage.

`TestRONChosenStaticPlayerAndColorContexts` checks fixed chosen-player filters,
You with nested chosen-color conjunctions, fixed disjunctions and ControlsNo
updates after departure. `TestRONChosenStaticInvalid` checks missing producers,
malformed reads, unsupported choice disjunctions, action-local players, empty
conjunctions and atomic rejection that preserves an existing registry.

Native `TestChosenStaticConditionIdentity` checks exact/stale source visits,
invalid player/color, multicolor subjects, nontoken membership, controller changes,
phasing, departure/return and clone isolation. `Declarations` validates supported
players and rejects nil, attachment, saved-object and unbound player references.
Existing entry-copy, cost and layer tests retain their separate contracts.

### Verification and measured result

Required checks passed in fast-fail order:

- `go test ./pkg/mage/dsl/ron -run TestRONChosenStatic -count=1` passed.
  Native `go test ./pkg/mage -run TestChosenStaticCondition -count=1` passed.
  Broader chosen-color, combined-entry, conditional-static and counter tests
  passed after updating the existing Jihad diagnostic assertion.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed.
- `make lint` passed with 0 issues. Automatic changes were import grouping
  only. The new engine/RON groups and related static/entry groups were rerun
  afterward and passed; no behavior or assertion changes came from lint.
- `git diff --check` passed, including the final ledger update.
- Final `go run ./cmd/loadcards -json` exited 1 with **2,206 compiled / 239
  rejected records**, **859 accepted / 127 rejected names**, out of 2,445
  records and 986 names. Comparing all diagnostics with the fresh baseline
  shows no added or removed rejection and just one changed diagnostic: Jihad
  advances from `unsupported static condition player (TheChosenPlayer)` at
  ARN:44:249 to `unsupported no-land trigger player reference (TheChosenPlayer)`
  at ARN:44:470. Accepted records remain accepted. Loading remains atomic.

Actual full-card gain is **zero names / zero records**, matching the prerequisite
estimate. Gameplay evidence increases for Jihad's entry/static rules only.
The separate state event still blocks complete Jihad execution. Full collection
gameplay, embedded runtime startup, application cutover, Go-card deletion and
DSL cleanup remain unfinished. No removal gate was reached.

Changed files: new `pkg/mage/chosen_static_condition.go`,
`chosen_static_condition_test.go`, `pkg/mage/dsl/ron/chosen_static.go` and
`chosen_static_test.go`; updated `pkg/mage/composable_conditions.go`,
`pkg/mage/dsl/ron/conditional_static.go`, `combined_entry_choice_test.go`,
`pkg/mage/doc.go`, `docs/dsl-reference.md`, `docs/engine-rules-contracts.md`
and this ledger. No card source or production registry changed.

### Ranked next candidates

1. **Chosen-player no-permanent state event — Jihad.** This slice completes
   the live membership prerequisite. Entry choices and both static reads can
   now support exact state detection; pending-trigger suppression, pending
   source identity/controller and source phasing/removal must be tested. One
   potential complete name / record; medium effort. Recompare multi-card work.
2. **Mandatory entry exile payment/results — Frankenstein's Monster.**
   Ordered entry and announced X are ready; exact graveyard transaction,
   entry-local outcomes, failure routing and combination counter choice remain.
   Zero immediate full-card gains from payment alone; medium/high effort.
3. **Persistent number choices — Shapeshifter.** Entry 0–7 choice, optional
   upkeep update and live power / 7-minus-choice toughness still need one
   per-visit lifetime. One potential complete name / record after all contexts;
   medium/high effort.

This run stops after live entry-choice static membership. Complete migration
remains gated by collection, gameplay, runtime and removal work.


## 2026-10-13 — SOURCE-CHOICE-PLAYER-ABSENCE-STATE-EVENT

### Selection and fresh baseline

The initial working tree was clean. `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed before edits. Fresh
`go run ./cmd/loadcards -json` exited 1 with **2,206 compiled / 239 rejected
records**, **859 accepted / 127 rejected names**. This was strict collection
rejection, not a build/test/environment failure.

The latest ranked candidates were evaluated first. Complete unchanged records
were read for Jihad, Frankenstein's Monster and Shapeshifter. Targeted vocabulary
reads checked `WhenAPlayerControlsNoPermanents`, `TheChosenColor`, `ChooseAColor`,
`ChooseANumberBetween`, `TheChosenNumber` and mandatory payment constructors.
All ten distinct records using the selected absence event were also read.
Reprints count once in this comparison.

| Candidate | Distinct affected names | Estimated complete gains | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Source-choice player absence state event | Jihad; shared query also serves Pirate Ship, Sea Serpent, Dandân, Island Fish Jasconius, Merchant Ship, Serendib Djinn, Giant Shark, Seasinger, Vodalian Knights | 1 name / 1 record; seven others compile; Giant Shark and Seasinger remain rejected | Entry choices and static membership are ready. Adds safe typed negative membership and reuses state-trigger lifetime across ten names. No other Jihad blocker is found in the complete nested record. Giant Shark still rejects its damage-history combat filter; Seasinger still rejects its controller target filter. | Low/medium; adapters, guarded engine queries and lifecycle tests |
| Mandatory entry exile payment/results | Frankenstein's Monster | 0 from payment alone | ValueX entry facts are ready; exact graveyard payment, entry-local paid-card result/failure and combination-counter choice are not. A complete card requires those separate consumers. General entry transactions can support later entry-cost work, but this exact exile shape has one unique consumer. | Medium/high |
| Persistent number choices | Shapeshifter | 1 only after all three contexts | Neither the entry producer, optional upkeep writer nor power/7-minus-number layer reader exists. Both producer occurrences and both number reads belong to this one card. Requires a new shared per-visit number contract. | Medium/high |

Candidate #1 wins on a complete record gain for the smallest credible effort.
It is dependency-ready and also shares its engine query with seven accepted
absence-event cards and two already supported subrules of rejected cards. The compared payment and number families do not expose
an additional ready multi-card unlock: nested consumers are counted rather than
assuming family names imply support. No first-error frequency was treated as a
full-card estimate. No engine layer, entry or trigger service was recreated.

Before edits, the acceptance contract required validated entry producers,
current control/color membership, nontoken exclusion, exact source identity,
pending and active suppression, rearming, source/subject phasing, source control
changes, departure/return, ability removal, copies, independent instances,
clone isolation and complete Jihad gameplay. Unsupported declarations must
reject before runtime; collection rejection must remain atomic.

### Established signatures, scopes and engine boundary

Ordinary battlefield `TriggerA` now lowers:

- `WhenAPlayerControlsNoPermanents(SinglePlayer(TheChosenPlayer), filter)`.
  The player requires a validated source `ChooseAPlayer(Opponent)` entry rule.
- Fixed filters use the existing static-condition grammar, including nonempty
  fixed conjunctions/disjunctions. `IsColor(TheChosenColor)` is accepted alone
  or in nonempty nested conjunctions. It requires the source color entry rule.
- Missing producers, tuple choice reads, chosen-color disjunctions, empty groups,
  action-local players and unbound `Trigger_ThatPlayer` body reads reject.
  This state event does not introduce an event-player binding.
- Existing `SinglePlayer(You)` basic-land/card-type forms retain their restrictions.
  They now lower through the same fixed absence query.

New engine declarations:

- `PlayerControlsNone(player PlayerRef, filter PermanentFilter) StateCondition`.
  Existing controller, active-player, bound-player and source-chosen-player
  contexts use shared player resolution. Unknown or missing players return false.
- `PlayerControlsNoneOfChosenColor(player PlayerRef, filter PermanentFilter)
  StateCondition`. Supports `You`, `ActivePlayer` and `ChosenPlayerOf(This)`.
  Requires a live exact source visit and a valid five-color choice. It composes
  the fixed absence reader with a color filter; absent/phased/returned sources
  cannot supply choices and there is no state-query LKI fallback.

RON owns spelling, choice-producer checks and filter lowering. The engine owns
negative membership. Shared `PlayerControls`, `resolveStatePlayer`,
`resolveStatePermanent`, `FilterBattlefield`, `WhenState`, state keys, pending
trigger snapshots, resolution frames, sacrifice and layers provide execution.
Definition validation reuses existing typed player/source condition contracts.
Declarations contain no mutable choice or game state. Factories still build
fresh trigger/action identities. No global result slot, card-name branch,
Go factory fallback, game-time declaration panic or partial registry was added.

Pending/stacked triggers keep the detected controller/source visit and suppress
new detection. Countering or resolving allows new detection. Membership returning
does not cancel a queued sacrifice. Ability removal and phasing stop detection;
phase-in restores choices. Pending phase cycles cannot repeat a trigger. Source
control changes preserve entry choices. The old controller cannot sacrifice a
source acquired by another player. Return keeps the old pending visit separate.
Copies, separate factories and clones keep distinct choices and pending bodies.

### Rules, data and failing-first evidence

Local `data/ARN.json` confirms Jihad's color/opponent entry choices, white global
+2/+1 condition, nontoken member restriction and separate absence sacrifice.
CR 603.8 supplies pending/active suppression and rearming; CR 603.3a and 113.7a
supply detected controller and source independence. CR 400.7 separates returns,
CR 702.26b excludes phased objects, CR 701.21a restricts sacrifice to the current
controller, and current control/color layers supply live membership.

Native `TestPlayerAbsenceCondition*` first failed compilation with undefined
`PlayerControlsNone` and `PlayerControlsNoneOfChosenColor`. The initial RON
fixture used `StackObject.ControllerID`; this was corrected to the existing
`Controller` field before confirming the intended failure. All complete Jihad
RON cases then failed at `unsupported no-land trigger player reference
(TheChosenPlayer)` before implementation. Invalid cases already rejected.

One lifecycle assertion initially drained the entire stack after source control
changed while the absence condition stayed true. The engine correctly queued a
new trigger for the new controller after the original left the stack. The fixture
now restores membership before control changes to isolate the old-controller
sacrifice restriction; no engine rule or production assertion was weakened.

This slice fills compiler adapters and a reusable guarded negative-query
contract. No source defect or environment failure was found. No RON source
changed, so the vocabulary source/hash snapshot requires no refresh. Runtime
and removal gates remain closed. No Go cards, generators or broad DSL functions
were removed; no unused-function removal was attempted.

### Card-to-test mapping

**Jihad → `TestRONChosenStateAbsenceJihad`, `Lifecycle`, `SpellCopy`.** These
execute its complete unchanged record, including actual casting, entry timing,
color/opponent choices, both players' white-creature P/T bonus, artifact and
nontoken membership, current color/control, member phasing, source phasing and
ability removal/restoration, pending/active suppression, counter rearming,
pending control/return identity, restored membership before resolution,
owner graveyard destination, copied spell choices, copied triggered abilities,
independent factories and game clones. `TestRONChosenStaticJihadRequirement`
retains detailed entry/static subrule coverage, including nonwhite exclusion,
late subjects and source return; `TestRONCombinedEntryChoiceJihad` retains
its entry-only test. Earlier color/entry tests now require complete compilation.

**Seven previously accepted absence-event cards →
`TestRONChosenStateAbsenceExistingCards`.** Pirate Ship, Sea Serpent, Dandân,
Island Fish Jasconius, Merchant Ship, Serendib Djinn and Vodalian Knights execute
their complete records for presence, member phasing, pending suppression and
unconditional queued sacrifice after phase-in. Existing `TestRONLandDependence*`,
`TestRONSourceAttack*` and `TestRONNoLandType*` retain their broader rule mappings.

**Giant Shark and Seasinger → the same `ExistingCards` test, subrule coverage.**
Their complete records are required to keep rejecting respectively at
`expected 1 positional arguments (WasDealtDamageThisTurn)` and
`unsupported permanent target controller (ControlsA)`. Test-only projections
retain unchanged metadata and each exact absence trigger. They run that state
rule through the engine; this is not complete-card gameplay evidence and no
production registry skips their other rules. The fresh loader keeps both full
records rejected at identical diagnostics.

`TestRONChosenStateAbsenceFilters` checks fixed color, fixed disjunction and
nested chosen-color conjunctions through explicit RON registries.
`Invalid` checks declaration errors and atomic rejection preserving an existing
registry. Native `Identity` checks missing/invalid choices, multicolor/token
subjects, exact/stale visits, source/member phasing and clone isolation;
`Declarations` checks typed-reference boundaries.

### Validation and measured result

- Focused RON and native tests passed with `-count=1`; related chosen-entry,
  static, land-dependence, no-land, source-attack and combat-pair groups passed.
- `make build` and `go vet ./...` passed.
- The first engine short run found six obsolete acceptance/count assertions:
  two still required Jihad rejection and four required 2,206/239. They were
  updated to require compilation and exact measured 2,207/238 counts. The
  rebuilt-definition test also checks Jihad's fresh accepted factory.
- After those updates, focused/count tests, `make build`, `go vet ./...`,
  `go test ./pkg/mage/... -short -count=1` and `make test` all passed.
- Final review added direct absence-event tests for the other nine syntax users.
  The first run exposed existing Giant Shark and Seasinger full-record gaps;
  both are now asserted as rejection boundaries with test-only exact subrule
  projections. Seven complete records retain direct regression evidence.
- `make lint` passed with 0 issues. Its only automatic changes were import
  grouping in the new engine/test files. After review, native condition tests
  and the whole RON package passed; after the explicit existing-card regression
  addition, the required sequence and final native/RON slice tests passed again.
  No behavior or assertion changes came from lint.
- `git diff --check` passed, including the final ledger/documentation update.
- Final `go run ./cmd/loadcards -json` exited 1 with **2,207 compiled / 238
  rejected records**, **860 accepted / 126 rejected names**, out of 2,445 records
  and 986 names. Diagnostic comparison with the fresh baseline removes only
  Jihad at ARN:44:470 (`unsupported no-land trigger player reference
  (TheChosenPlayer)`). There are no added or changed remaining diagnostics.
  Giant Shark and Seasinger keep their exact earlier rejection diagnostics.
  All previously accepted records remain accepted. Loading remains atomic.

Actual full-card gain is **one name / one record**, matching the estimate.
Jihad advances from entry/static subrule evidence to complete-card gameplay.
The other seven accepted absence-event names remain accepted. Giant Shark and
Seasinger remain rejected at unrelated adapters, with new absence subrule
evidence only.
Complete collection gameplay, embedded loading, startup cutover, Go-card
removal and DSL cleanup remain unfinished.

Changed files: new `pkg/mage/player_absence_condition.go`, its native tests and
`pkg/mage/dsl/ron/chosen_state_absence_test.go`; updated `land_dependence.go`,
`triggers.go`, prior chosen-color/static/entry assertions, collection inventory,
numeric-consumer and definition-plan count assertions; `pkg/mage/doc.go`,
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger.

### Ranked next candidates

1. **Mandatory entry exile payment/results — Frankenstein's Monster.**
   Source entry X and required operation order are ready. Add exact graveyard
   transaction and entry-local paid-card outcomes/failure routing before the
   combination-counter consumer. Zero immediate gains from payment alone;
   medium/high effort. Recompare shared entry-cost work.
2. **Persistent number choices — Shapeshifter.** One number visit must serve
   entry, optional upkeep updates and live power/7-minus-choice toughness.
   One potential full name / record after all three contexts; medium/high effort.
3. **Nonbattlefield source lifetimes.** Compare complete Nether Shadow,
   All Hallow's Eve and Psychic Purge records to scope a shared trigger/source
   visit prerequisite. Their graveyard/exile/hand rules have separate nested
   blockers; no full gains are assumed until that comparison. High effort.

This run stops after source-choice player absence state events.


## 2026-10-13 — SOURCE-ZONE-UPKEEP-LIFETIMES

### Environment and selection

The initial working tree was clean. Fresh `make build` and the whole RON
package (`go test ./pkg/mage/dsl/ron -count=1`) passed. The baseline loader
exited 1 with 2,207 compiled / 238 rejected records and 860 accepted / 126
rejected names. This was strict collection rejection, not a build/test failure.
No user changes or unrelated untracked files needed preservation.

Current ledger context came from the top status and latest entry.
Complete requirements came from Frankenstein's Monster (DRK), Shapeshifter
(ATQ), all six Nether Shadow reprints and All Hallow's Eve/Psychic Purge (LEG).
Targeted vocabulary entries confirmed two ChooseANumberBetween occurrences,
six FromGraveyard occurrences and one each of FromExile/FromAnyZone. Reprints
count as one name. No Go card definition supplied requirements.

| Candidate | Distinct affected cards | Estimated immediate full-card gains | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Ranked 1: mandatory entry exile payment/results | Frankenstein's Monster | 0 names / 0 records | MustCost uses entry X and a controller-graveyard creature selection. Exact cost visits, local payment outcomes, failed-entry routing and combination-counter choices remain. Shared resolving/activation transactions help, but this inventory entry context has one downstream card. | Medium/high; entry transaction, rollback, payment result and replacement-order tests |
| Ranked 2: persistent number choice | Shapeshifter | 0 from storage alone; eventual 1 name / 1 record | Entry 0–7 choice, optional upkeep revision and layer 7b power/7-minus-choice toughness all need one exact visit. Existing entry choices and layers help; all three adapters remain. | Medium/high; storage, two decision contexts, live layers and copy/return tests |
| Ranked 3: nonbattlefield source lifetimes | Nether Shadow; All Hallow's Eve; Psychic Purge | 0 names / 0 records from lifetime infrastructure | Graveyard and public exile upkeep events share ownership, per-event source visits and pending frames. Nether Shadow still needs ordered creature cards above it, intervening-if and optional exact-source battlefield return. All Hallow's Eve needs self-spell exile with counters, visit-local exile counter storage/conditions/removal and simultaneous all-graveyard return. Psychic Purge needs discard cause/controller and hand/source-zone visibility; it is outside this slice's supported zones. | High for all zones; medium for the selected graveyard/exile ordinary upkeep context |

Ranked 1 has one eventual card and is not a shared multi-card prerequisite in
this inventory. It was deferred while useful shared source-zone work remained.
The selected context establishes an exact-visit/event-frame prerequisite for two
unique cards with seven records. Psychic Purge was inspected and remains a
separate hand/event-cause problem; this slice does not claim to enable its rule.
No complete unlock was estimated. This run stops after this single capability.

### Contract, dependencies and reuse boundary

Supported RON signature/context (`actions` is a list):

- `FromGraveyard(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList(actions)))`
  binds singular `ThisGraveyardCard` to the source's detected graveyard visit.
- `FromExile(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList(actions)))`
  binds singular `ThisExiledCard` to the source's detected face-up exile visit.
- `Trigger_ThatPlayer` reads the saved upkeep player. `You` reads the queued
  trigger controller. Copies may have different identities for these players.
- `PutGraveyardCardIntoHand(ThisGraveyardCard)` reuses `ReturnCardToHand`.
  `PutExiledCardIntoOwnersGraveyard(ThisExiledCard)` reuses the shared exile
  transport through the typed singular-set adapter. Existing delayed action
  lists retain source/player facts through their saved resolution frames.

RON owns all spelling, arity, lexical scope and fresh factory bindings. A new
zone scope excludes spell X, entry choices, Aura hosts and battlefield sources.
Other step/player selectors, nested wrappers, source spell damage, targets,
modes and `TriggerI` remain errors. The one required SpellActions rule remains
mandatory on instant/sorcery definitions; a zone rule does not replace it.

Native signatures:

- `WithSourceCardInZone(zone core.Zone, source *CardBinding) TriggerOption`
  selects one graveyard or public exile zone on an ordinary event without
  targets/modes/intervening-if. Validation rejects invalid/nil bindings, other
  zones, duplicate options, added zones, state/intervening-if events, targets,
  modes and battlefield source reads.
- `ExiledCardFrom(card CardRef) ExiledCardSet` adapts a current singular card
  reference to the shared exile movement service.

`Game.detectCardZoneTriggers` makes a separate source/controller context for
each detected event. Existing graveyard callers use that helper unchanged;
face-up exile joins the same scan. The pending queue, APNAP ordering, event
capture effects, typed card visits, saved resolution frames, stack copying,
cloning and movement services remain shared. Zone triggers do not borrow stale
battlefield incarnation/LKI. A departed source cannot supply an action read,
but independent queued effects still execute. Return starts another visit.
Hidden exile never supplies triggers, even if one player can see the card.

CR 113.6b supplies zone-only ability operation; CR 400.3 supplies ownership of
graveyards; CR 603.2f supplies public visibility; CR 603.3a/b supplies controller
capture and APNAP; CR 113.7a supplies queued independence; CR 400.7 separates
returns. CR 603.4 was inspected for the explicitly deferred intervening-if
contract. No new payment, replacement, simultaneous movement, damage, static
layer or battlefield phasing subsystem was introduced.

### Failing-first evidence and acceptance cases

Native `TestSourceZoneUpkeep*` initially failed compilation at undefined
`WithSourceCardInZone` and `ExiledCardFrom`. The first RON fixture mistakenly
used nonexistent Registry.Has/Count methods; it was corrected to
CardRegistered before the intended failure was confirmed. All four RON
positive cases then failed at unsupported FromGraveyard/FromExile rules.
Invalid declarations already rejected.

After the initial implementation, a target-source review found that targeted
zone triggers would need another source-information contract. A new invalid
Targeted case failed with `invalid declaration accepted`; the ordinary
ActionList boundary and native target/mode rejection then made it pass.
No accepted production record relied on that new context.

A new native detection test initially supplied the wrong PhaseIn argument type
and omitted uuid's import. Both fixture errors were corrected. Its declaration
purity assertion initially expected nil source/controller after a prior
battlefield visit. Battlefield setup had already populated those fields. The
assertion now saves them immediately before zone detection and requires exact
preservation; the immutable-detection contract remains tested.

Native Visit covers present, departed, returned and copied source reads, pending
clones, owner destinations, saved upkeep players, copied controllers and action
timing. Declarations covers zone/binding/scope errors and forbidden contexts.
Detection covers hand/battlefield/phased/hidden-exile exclusion, public entry,
separate copied definitions, unchanged detection declarations, distinct events,
owner controllers and APNAP order. RON Gameplay and Invalid cover explicit
registry execution, scope rejection, source return identity, player/controller
separation and atomic loader rejection. DelayedFrames covers two factory
instances and both owners through a later end-step event. Inventory keeps rule,
event and source-card roles while retaining unverified gameplay status.

### Card-to-test mapping

**Nether Shadow → TestRONSourceZoneUpkeepRecordPrerequisites, all six sets.**
Each complete unchanged record must still reject. A test-only projection keeps
its source wrapper, unchanged upkeep event and metadata, substitutes ordinary
TriggerA for TriggerI and uses a synthetic bound-player life probe. This proves
source-zone arming/ownership prerequisites, not its condition or return body.

**All Hallow's Eve → the same RecordPrerequisites test, LEG.** The complete
unchanged record must still reject. The test-only projection keeps its source
wrapper/upkeep event/metadata and executes the exact singular
PutExiledCardIntoOwnersGraveyard(ThisExiledCard) instruction from its original
conditional body. Its unsupported spell body and intervening-if are replaced
only in that test projection. This proves exact exile-source binding and the
shared move prerequisite, not the complete card. Production loading never
projects, skips or publishes an incomplete record.

**Psychic Purge → comparison only.** Its full FromAnyZone discard-cause record
was inspected, but there is no new gameplay claim or supported hand-source
adapter. Frankenstein's Monster and Shapeshifter remain compared alternatives.

### Validation and measured result

- Fresh baseline build and whole RON package passed; loader exited 1 as expected.
- `go test ./pkg/mage/dsl/ron -run TestRONSourceZoneUpkeep -count=1` and native
  `go test ./pkg/mage -run TestSourceZoneUpkeep -count=1` passed.
- `make build` and `go vet ./...` passed.
- `go test ./pkg/mage/... -short -count=1` and `make test` passed. The whole RON
  package also passed independently with `-count=1`. Accepted count assertions
  did not need any change. Final test-only declaration/inventory additions
  passed the focused native and RON groups again.
- The first `make lint` reported redundant type annotations in two new native
  tests and an overwritten incoming compiler scope parameter. Its automatic
  fixes removed the annotations; review confirmed no behavior change. The
  compiler helper now takes only Node and creates its own scope. Focused native
  and RON tests, build, vet, engine short tests and make test all passed again.
- Final `make lint` passed with 0 issues. No unrelated files changed.
  After review, final focused native/RON tests passed again.
- `git diff --check` passed, including the completed ledger and documentation.
- Final `go run ./cmd/loadcards -json` exited 1 with **2,207 compiled / 238
  rejected records**, **860 accepted / 126 rejected names**, out of 2,445 records
  and 986 names. No rejection was added or removed. The only diagnostic changes
  are the six Nether Shadow records: unsupported rule (FromGraveyard) at column
  149 advances to unsupported source-zone trigger (TriggerI) at column 163.
  All Hallow's Eve remains rejected at its unchanged unsupported exile subject
  (Spell) at LEG:12:190. All previously accepted records remain accepted;
  production collection loading remains atomic and rejects the collection.

Estimated full-card gains were zero. Actual gains are zero names / zero records.
Gameplay evidence grows by a shared prerequisite for two names; no complete-card
coverage is claimed. No source defect or environment failure was found. No RON
source changed, so no source/hash vocabulary snapshot refresh is needed. Runtime
loading, test cutover, Go-card removal and DSL cleanup remain gated. No Go cards,
generators or broad DSL functions were removed; no unused-function removal was
attempted. API and rules-contract documentation describe only this capability.

Changed files: new pkg/mage/source_zone_trigger.go and source_zone_upkeep_test.go;
new pkg/mage/dsl/ron/source_zone_upkeep.go and source_zone_upkeep_test.go; updated
engine game.go, triggered.go, definition.go and doc.go; RON compiler.go,
player_scope.go, graveyard_return.go, grouped_exile.go, self_damage_source.go and
inventory.go; docs/dsl-reference.md, docs/engine-rules-contracts.md and this ledger.

### Ranked next candidates

1. **Source-zone intervening-if facts and predicates.** Reuse the exact source
   visits for Nether Shadow's ordered creature-count condition and All Hallow's
   Eve's exile-counter predicate. A shared condition-query frame is a concrete
   prerequisite for two names; optional return and counter actions remain.
   Zero gains from the frame alone; medium effort. Compare full nested forms
   again before committing to a larger slice.
2. **Mandatory entry exile payment/results — Frankenstein's Monster.** Entry X
   and ordered operations are ready. Add exact transaction/results/failure
   routing before combination-counter choice. Zero gains from payment alone;
   medium/high effort for one eventual name / record.
3. **Persistent number choices — Shapeshifter.** Entry, optional upkeep revision
   and live layer values share one per-visit number. One potential name / record
   after all three contexts; medium/high effort.

This run stops after ordinary source-zone upkeep lifetimes.


## 2026-10-13 — SOURCE-ZONE-INTERVENING-QUERIES

### Candidate comparison and selection

The working tree was clean at the start. Fresh `make build` and whole RON package
tests passed. Baseline `go run ./cmd/loadcards -json` exited 1 with 2,207 compiled
and 238 rejected records: collection rejection, not a build or test failure.
The baseline JSON was saved outside the repository for diagnostic comparison.

All three latest ranked candidates were evaluated from complete records and
nested forms, counting reprints once:

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Source-zone intervening-if query frame and ordered graveyard predicate | Nether Shadow; All Hallow's Eve | 0 from this prerequisite | Nether Shadow needs MayAction exact-source battlefield return. All Hallow's Eve needs spell exile with counters, exile-counter reads/removal and simultaneous graveyard return. One pure exact-visit condition frame enables both source-zone conditions. | Medium |
| Mandatory entry exile payment/results | Frankenstein's Monster | 0 from payment alone | MustCost transaction, entry result/failure routing, NumberOfCardsOfTypeExiledThisWay and combination-counter choice all need contracts. Entry X and ordered entry operations are ready. Enables one complete entry procedure. | Medium/high |
| Persistent number choice | Shapeshifter | 1 only after all contexts | ChooseANumberBetween at entry, optional upkeep replacement of that number, and live SetPower/SetToughness reads share per-visit storage. All three need implementation and lifecycle tests. | Medium/high |

Candidate 1 is dependency-ready through the previous exact source-zone visit
slice. It wins as the shared condition-query prerequisite for two concrete
names. This run completes one supported context: source-zone owner-upkeep
intervening conditions with pure current card facts and ordered graveyard reads.
It does not bundle optional returns or exile counter storage into the slice.
The exile card-type predicate is a synthetic test of the shared frame; it is
not an inventory-card unlock or a substitute for the counter predicate.

Full Nether Shadow records were read in LEA, LEB, 2ED, 3ED, CED and CEI.
All Hallow's Eve (LEG), Frankenstein's Monster (DRK) and Shapeshifter (ATQ)
were read in full. Targeted vocabulary entries show six ordered graveyard
condition occurrences and two ExiledCardPassesFilter occurrences on All Hallow's
Eve, including its nested resolving condition. The latter counter shapes stay
unsupported. No requirements came from Go card definitions.

### Exact supported shapes, dependencies and engine boundary

RON now accepts either TriggerA(event, ActionList) or
TriggerI(event, condition, ActionList) within one FromGraveyard or FromExile
wrapper. The only event remains
AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)). Supported conditions:

- FromGraveyard: CardIsInPlayersGraveyardWithANumberOfCardsAboveIt(
  ThisGraveyardCard, You, GreaterThanOrEqualTo(Integer(N)), IsCardtype(T)).
  N is a fixed nonnegative integer and T is one supported valid card type.
- FromExile: ExiledCardPassesFilter(ThisExiledCard, IsCardtype(T)).
  This shape exercises typed current card queries. Counter predicates reject.

Wrong-zone, unbound or tuple-spelled source references, negative/dynamic amounts,
other comparisons, other filters, other upkeep players, targeted bodies and
battlefield source reads reject at declaration time. Source-card and upkeep
player bindings are fresh per factory and local to that wrapper. Instant/sorcery
zone rules still require a supported complete SpellActions body.

New engine APIs:

- GraveyardCardHasAtLeastAbove(card CardRef, player PlayerRef, minimum int,
  filter CardFilter) StateCondition.
- CardMatchesFilter(card CardRef, filter CardFilter) StateCondition.

Both require a saved FromCard reference; targets do not have a detection query
frame and fail native definition validation. The ordered condition requires
present exact membership even for minimum zero. Only matching cards after the
source in the graveyard slice count. Below-source cards and the source itself
never count. Current card facts, not LKI, supply both queries.

StateContext now carries typed card facts. WithInterveningIf obtains source
facts in a separate detection frame; IfState supplies captured resolution facts.
Card queries use a shallow read context with its own binding maps, leaving the
active resolution frame and source/controller declarations unchanged. Either
trigger-option order works. The existing FromCard visit service checks identity;
source departure and return make the old condition false. Shared pending,
stack-copy and game-clone services retain captured facts. You is the effect
controller; copies under another controller check that controller's graveyard.
Event-player bindings retain their separate captured identity.

No parallel zone/order/event service was added. CR 603.4 establishes both checks;
CR 404.1/2/3 establishes graveyard top order and existing arrival ordering;
CR 400.7 rejects returned visits; CR 603.2f keeps private exile from arming events.
Native source-zone definition validation now permits intervening conditions,
while retaining state/target/mode and battlefield-source rejection. The older
invalid-context test now uses an invalid battlefield condition instead of the
newly valid pure player condition. No payment, replacement-order, phasing or
simultaneous-movement behavior changes. Existing tests retain those services.

### Failing-first tests and card mappings

Native TestSourceZoneConditionTiming and Exile first failed compilation at the
two new undefined APIs. A fixture used lowercase clone instead of Clone; this
was corrected before confirming the intended undefined-API failure. RON positive
fixtures and all six record projections then failed at unsupported TriggerI.
A later native target-condition rejection test failed with "target condition
without a detection frame accepted" before saved-reference validation was added.
The declaration purity fixture initially used nonexistent SourceID/ControllerID
methods; it was corrected to existing native fields before the check passed.

TestSourceZoneConditionTiming covers false detection, false resolution, source
departure/return with otherwise sufficient cards above it, unrelated live-frame
sentinels, immutable declarations, copies under both controllers and pending
clones. OrderedQuery covers mixed card types and zero-minimum exact membership.
Declarations covers targets, missing/unwritten bindings and negative amounts.
Exile covers either option order and present/returned visits; HiddenExile covers
face-down cards visible to their owner.

TestRONSourceZoneConditionGameplay covers real RON registry execution, below-only
failure, success, removal before resolution, source return, owner-only upkeep,
two independent factories and exile current-card facts. Invalid covers spelling,
scope, filters, values, contexts and atomic LoadInto rejection. Inventory retains
explicit trigger-condition/card-reference/comparison/card-filter roles and
unverified gameplay status.

**Nether Shadow → TestRONSourceZoneConditionRecords (all six reprints).**
These test-only projections retain metadata, Haste, source wrapper, event and
complete unchanged condition. Only the unsupported optional-return body becomes
a life probe. Each production record is separately required to keep rejecting.
This establishes condition gameplay and no complete-card coverage.

**All Hallow's Eve → shared native/RON exile-frame contract only.** The complete
record remains rejected at spell exile. Its inspected counter predicates and
operations remain unsupported, with explicit RON rejection evidence. Earlier
source-zone wrapper/action prerequisite tests remain. The synthetic card-type
predicate does not claim its counter condition works.

Frankenstein's Monster and Shapeshifter are comparison candidates only.

### Verification and measured result

- Fresh baseline build and whole RON package tests passed; loader exited 1.
- go test ./pkg/mage/dsl/ron -run TestRONSourceZoneCondition -count=1 passed.
- go test ./pkg/mage -run 'TestSourceZone(Condition|Upkeep)' -count=1 passed.
- make build and go vet ./... passed.
- go test ./pkg/mage/... -short -count=1 passed.
- make test passed. Whole RON tests also passed independently after implementation.
- First make lint found one prealloc issue in a new native test. The slice
  allocation was corrected; focused native tests passed again. Final make lint
  passed with 0 issues. Formatting/fix output was reviewed; no unrelated file
  changed. Focused RON tests also passed again. git diff --check passed.
- Final go run ./cmd/loadcards -json exited 1 as expected: **2,207 compiled /
  238 rejected records**, **860 accepted / 126 rejected names**, out of 2,445
  records and 986 names. The baseline/final issue sets are identical. No accepted
  record became rejected. Six Nether Shadow diagnostics advance from column 163
  unsupported source-zone TriggerI to column 376 unsupported optional action
  PutGraveyardCardOntoBattlefield. All Hallow's Eve remains rejected at unchanged
  LEG:12:190 unsupported exile subject Spell. Production loading remains atomic.

Estimated full-card gains were zero; actual gains are zero names / zero records.
Prerequisite gameplay evidence grows for Nether Shadow's complete unchanged
condition and for the shared source-zone query context. Complete-card coverage
is unchanged. Previously accepted records remain accepted. No source-data defect
or environment failure was found. No RON source changed, so vocabulary source
hashes do not need refresh. Semantic inventory roles were updated in code.

Runtime cutover, Go-card deletion and unused DSL removal remain gated. No Go
card, generator or broad DSL function was removed, and no unused-function
removal was attempted. API/rules documentation describes only this slice.

Changed files: new pkg/mage/source_zone_condition.go and
source_zone_condition_test.go; new pkg/mage/dsl/ron/source_zone_condition.go and
source_zone_condition_test.go; updated composable_conditions.go,
composable_events.go, definition.go, doc.go, source_zone_upkeep_test.go;
RON source_zone_upkeep.go and inventory.go; docs/dsl-reference.md,
docs/engine-rules-contracts.md and this progress ledger.

### Ranked next candidates

1. **Restricted mana legality adapters — Drain Life and Mishra's Workshop.**
   Compare SpendOnlyColorManaOnX with fixed AddManaWithModifiers and
   CanOnlySpendOnSpells(IsCardtype(Artifact)). Reuse shared payment and mana
   solver metadata; inspect whether a shared legality prerequisite is missing
   before choosing one context. Workshop has a potential one-name/one-record
   unlock; Drain Life also needs clamped actual-damage life gain. Medium/high
   effort. Sunglasses of Urza's as-though spending is another distinct dependent
   contract. Their complete records were inspected; no implementation starts.
2. **Optional exact source-card battlefield return — Nether Shadow.** Its
   event and condition are ready. Add a resolution-time optional choice and
   exact source visit return through shared entry services; one potential
   name/six-record unlock. Medium effort. Preserve source identity, entry rules,
   decline and copied-controller semantics.
3. **Exile-card counters — All Hallow's Eve.** The source-zone query frame is
   ready; add per-exile-visit counter storage before predicates/removal and spell
   exile options. Zero immediate gains from storage; medium/high effort for one
   eventual card. Simultaneous all-graveyard return remains another dependency.

This run stops after source-zone intervening query support.


## 2026-10-13 — RESTRICTED-FIXED-MANA

### Candidate comparison and selection

The tree was clean. Fresh `make build` and whole RON package tests passed.
The baseline loader exited 1 for collection rejection, with 2,207 compiled /
238 rejected records and 860 accepted / 126 rejected unique names.
The latest ranked candidates were evaluated from complete unchanged RON records,
nested forms and targeted vocabulary entries. Reprints count once.

| Candidate | Distinct affected names and estimated full-card gains | Other blockers and downstream value | Effort |
| --- | --- | --- | --- |
| Ranked #1: restricted mana legality, selecting fixed production | Mishra's Workshop and Metamorphosis share spending restriction lowering. Workshop: one name/one record gain; Metamorphosis: already accepted. Drain Life and Sunglasses of Urza are distinct dependent contexts. | Pool and solver restrictions already exist. Fixed resolving batch production is missing. Drain Life's black-only X payment and clamped actual-damage life gain remain; Sunglasses needs as-though spending. Shared fixed production and Artifact/Creature restrictions serve both production forms without changing those contracts. | Low/medium for fixed context; medium/high for X payment. |
| Ranked #2: optional exact graveyard source return | Nether Shadow: one name/six records. | Event and condition are ready; optional choice and exact source-card entry adapter are missing. Reuses zone-visit and entry services. | Medium. |
| Ranked #3: exile-card counter storage | All Hallow's Eve: zero immediate gains, one eventual name/record. | Counter storage is a prerequisite; spell exile options, predicates/removal and simultaneous all-graveyard return also remain. | Medium/high. |

Ranked #1 is dependency-ready. The fixed context wins on credible gain per effort
and shares legality lowering with an already supported second unique card.
This run implements one production capability, not black-only X payment or
as-though color spending. The inspected records are ATQ:46 Workshop, ARN:53
Metamorphosis, all six Drain Life/Sunglasses reprints, the complete LEA Nether Shadow
record and LEG:12 All Hallow's Eve. No Go card supplied requirements.
Local data/ATQ.json and data/ARN.json confirm both spending rules. CR 106.6
keeps restrictions separate from mana type; CR 605.1a/3b requires immediate
untargeted mana ability resolution. No source-data defect was found.

### Contract, dependencies and reuse boundary

Native signature: `mage.AddRestrictedMana(productions []mage.ManaProduction,
restriction core.ManaRestriction) mage.ResolvingEffect`. It copies a nonempty
batch with concrete colors and positive fixed amounts, applies one restriction
to every unit and rejects missing restrictions or any-combination production.
The shared `addEffectMana` pipeline receives the complete batch in one event.
Caller and metadata slices cannot change the saved declaration. Manual and
planned production retain identical spending rules. Replacements can change
colors without losing restrictions. Declaration state has no per-game writes.

RON signature: `AddManaWithModifiers(fixedProduction,
CanOnlySpendOnSpells(IsCardtype(Artifact|Creature)))/2` in resolving bodies.
Fixed production accepts the six concrete mana symbols and recursively nonempty
`And([...])` groups. Any-color production, empty nested groups, other filters,
wrong arities and entry/layer contexts reject before factory publication.
`compileManaSpendingRestriction(Node)` serves fixed production and existing
`AddManaRepeatedWithModifiers(AnyManaColor,value,restriction)/3`; the latter
now also accepts Artifact and retains its one-color choice/value contracts.
`compileFixedRestrictedProduction(Node)` validates only this fixed context.
RON owns spelling and contexts; the engine owns production events and legality.

Existing ability classification, sequential solver metadata, mana pool,
spell payment contexts, type replacements, production history, transactions,
stack copies and clones are reused. Costs stay atomic; restricted spell mana
cannot pay artifact ability costs. Phasing stops source activation. Produced
mana retains its rule after source departure. Copies use their own controllers.
No source read or result binding is introduced: LKI and returned-visit reads are
not needed for this production. Existing source identity/payment rechecks remain.
No parallel payment/replacement service or runtime card-name branch is added.

### Failing-first tests and card mappings

Native tests first failed at undefined AddRestrictedMana after correcting test
API mistakes. RON tests then failed at the expected unsupported
AddManaWithModifiers action in every Workshop scenario. A later invalid nested
empty-group test failed because it was accepted; recursive validation fixed it.
The batch review replaced an interim per-production effect approach with one
native event. A registry fixture needed its owner set before source destruction.

- **Mishra's Workshop → TestRONRestrictedFixedManaWorkshop and Registry.**
  Complete unchanged real record: immediate manual production; automatic artifact
  and artifact-creature casting; nonartifact and activation-cost rejection;
  insufficient payment, phasing, rollback, two registry factories and source
  departure. Native Paths and Batch verify restrictions through replacements,
  one simultaneous production event, query purity and declaration isolation.
- **Metamorphosis → existing TestRONSacrificeManaMetamorphosis**, retained through
  shared restriction lowering. SpellCopies adds synthetic coverage for both
  restrictions, nested mixed colors, repeated one-color production, different
  copy controllers and game clones. No additional full-card gain is counted.
- Native Declarations and RON Invalid cover invalid signatures/contexts and
  atomic collection rejection with a retained registry. Registry verifies
  mana-production, spending-restriction and spell-filter inventory roles.

### Verification and measured result

Focused RON/native tests passed with -count=1. Build and vet passed. The first
engine-short run failed four stale exact collection-count assertions; all other
engine packages passed. Those assertions now require exactly 2,208/237, with an
explicit Workshop nonrejection assertion. Their focused rerun passed.
Fresh `make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1`
and `make test` all passed after that correction.
`make lint` passed with 0 issues. Automatic formatting was reviewed; no unrelated
file changed. Focused native/RON checks passed again after formatting.
`git diff --check` passed. No environment failure remains.
Final loader: **2,208 compiled / 237 rejected records**, **861 accepted /
125 rejected names**, out of 2,445 records and 986 names. It exits 1 as expected
and publishes no collection. Only Workshop's ATQ rejection disappeared; no
accepted record became rejected and all 237 remaining diagnostics are unchanged.
Estimated gains were one name/one record; actual gains match. Complete-card
RON gameplay coverage adds Workshop. No RON source changed; vocabulary source
hashes do not need refresh. API/rules documentation covers only this capability.

Changed files: new pkg/mage/restricted_fixed_mana.go and native test; new
pkg/mage/dsl/ron/restricted_fixed_mana_test.go; engine composable_costs.go,
costed_mana.go and doc.go; RON compiler.go, restricted_mana.go and inventory.go;
exact-count tests definition_plan_test.go, inventory_test.go and
numeric_consumers_test.go; docs/dsl-reference.md, engine-rules-contracts.md and
this ledger. Existing callers remain. No Go definitions, generators or broad
DSL functions were removed; the cutover/removal gates remain unmet.

### Ranked next candidates

1. **Optional exact source-card battlefield return — Nether Shadow.** Event and
   condition are ready; add the optional resolution choice and shared exact-visit
   entry operation. Potential one-name/six-record unlock; medium effort.
2. **Restricted X payment — Drain Life.** Fixed spending restrictions do not
   constrain X. Inspect announced cost decomposition and solver payment first.
   Zero immediate unlocks until clamped damage gain also works; medium/high effort.
3. **Exile-card counters — All Hallow's Eve.** Exact query frames are ready;
   storage precedes counter predicates/removal, spell exile and batch returns.
   Zero immediate gains; medium/high effort. As-though spending for Sunglasses
   remains a separate comparison candidate.

This run stops after restricted fixed mana production.

## 2026-10-13 — OPTIONAL-GRAVEYARD-SOURCE-RETURN

### Candidate comparison and selection

The tree was clean. Fresh `make build` and `go test ./pkg/mage/dsl/ron -count=1`
passed. The baseline loader exited 1 for collection rejection with 2,208 compiled /
237 rejected records and 861 accepted / 125 rejected names. This was collection rejection. The latest ranked candidates were compared from
complete records, nested forms and targeted vocabulary. Reprints count once.

| Candidate | Distinct affected names; estimated full-card gains | Other blockers; downstream value | Effort |
| --- | --- | --- | --- |
| Ranked #1: optional exact graveyard source return | Nether Shadow; one name/six records | Event, exact source binding, condition and shared entry are ready. Only the optional native operation and compiler adapter remain. Completes source-zone return; typed native optional entry can also serve future saved card results. | Low/medium |
| Ranked #2: restricted X payment | Drain Life; zero immediate gains, one eventual name/six records | Announced cost decomposition and solver restrictions need a contract. Clamped actual-damage gain is a separate remaining blocker. Enables shared X legality, including interaction with restricted mana and as-though spending. | Medium/high |
| Ranked #3: exile-card counter storage | All Hallow's Eve; zero immediate gains, one eventual name/record | Storage precedes predicates/removal. Spell-exile modifiers and simultaneous all-graveyard return also remain. Enables exact-visit counters in query and resolving frames. | Medium/high |

Candidate #1 is dependency-ready and has the largest credible complete-card gain
per effort among these candidates. No compared alternative unlocks multiple unique cards in one complete slice.
The six Nether Shadow and Drain Life records and the complete LEG All Hallow's
Eve record were inspected. No Go card supplied requirements. Local LEA/LEG
Oracle data agrees with the selected source. CR 603.4/5 requires both intervening
checks and a resolution-time optional choice; CR 400.7 prevents stale returns;
CR 404.1 preserves graveyard order; CR 603.6d and 614.1c/d put entry replacements
before entry events. No source-data defect was found.

### Contract, dependencies and reuse boundary

Native signature: `mage.MayReturnNonAuraCardToBattlefield(card mage.CardRef,
player mage.PlayerRef) mage.ResolvingEffect` for non-Aura permanents. The selected player chooses normal
entry at resolution and is the entry controller/actor. Ownership stays unchanged.
No card, no player, a nonpermanent, an Aura or a card outside a graveyard
supplies no offer.
The operation saves the exact graveyard visit before the offer and checks it
after acceptance. Departure or return cannot supply entry. Decline does not stop
independent instructions or future offers. Native definition validation retains
the existing typed card/player child contracts. No declaration holds chosen state.

RON signature/context: `MayAction(PutGraveyardCardOntoBattlefield(
ThisGraveyardCard, [EntersNormally]))` in ordinary `ActionList` bodies of supported non-Aura
`FromGraveyard` upkeep `TriggerA`/`TriggerI` and their saved delayed bodies.
`compileOptionalSourceReturn(Node, scope) (effectFactory, error)` validates the
source plan and exactly one bare entry symbol. The selected player is `You` in
this context. Fresh source bindings are built for each factory. Aura or exile sources,
other references, modifiers, malformed arities, unbound references, entry and
replacement contexts and scoped player-action optional returns still reject.
The ordinary targeted return adapter and all existing callers remain unchanged.

RON owns spelling, scope and declaration errors. The engine adds an immutable
optional flag to the existing typed return operation. It reuses
`returnGraveyardCardToBattlefieldWithEntry`, entry preparation/replacements,
post-preparation visit validation, graveyard movement, registration and events.
Event capture, intervening conditions, APNAP, stack copies, delayed frame capture
and clones keep their existing contracts. No global result slot or parallel zone
service is added. Native controller source reads use the existing waiting-query
contract without resolving operands or making offers. Phasing preserves saved
waiting identity; returned sources cannot replace it. This singular return has
no payment, rollback or simultaneous batch contract. Entry replacement ordering
is supplied by the existing shared entry service.

### Failing-first tests and card mappings

Native tests first failed at undefined `MayReturnNonAuraCardToBattlefield`, after test
API spelling corrections. RON tests independently failed at the expected
unsupported optional `PutGraveyardCardOntoBattlefield` in the complete records.
Direct-frame and cloned-player fixtures were corrected to retain exact visit
methods and explicit decisions; no production clone change was needed. A final
failing Aura test exposed missing attachment choices in the reused entry path.
The native API now explicitly filters Auras, and the compiler rejects Aura sources
using retained source-plan declaration metadata. Existing event/condition probes
now assert full Nether Shadow compilation. Exact collection assertions require
2,214 compiled / 231 rejected records.

- **Nether Shadow → TestRONOptionalGraveyardReturnNetherShadow.** All six complete
  unchanged records: haste permits immediate attack, own-upkeep only, three
  matching creatures above the source, noncreatures/below-source exclusions,
  false detection/resolution, source departure/return, choice-time return,
  acceptance/decline, entry replacement, two factories, copies with same/other
  controller, pending and stacked clones and source phasing. One name/six records
  receive complete-card coverage, including metadata.
- **TestRONOptionalGraveyardReturnFrames.** Synthetic source-zone bodies: separate
  queued events, decline followed by later acceptance, independent continuation,
  saved delayed bodies and returned delayed visits. Inventory retains action,
  card-reference and entry-operation roles without claiming behavior coverage.
- **TestRONOptionalGraveyardReturnInvalid.** Wrong arities, entry symbol shape,
  empty/multiple/unsupported entry options, wrong/unbound source zones, unsupported
  contexts and atomic rejected loading with a retained registry.
- **TestOptionalGraveyardReturn, Declarations and Waiting.** Native acceptance,
  decline, departed/returned visits, departures/returns during choice, different
  owner/controller, nonpermanents, exile, missing players, entry counters and
  tapped-entry replacement ordering. Definition rejection and waiting-query
  source identity, phasing, departure, return exclusion, opaque player purity
  and clone lifetime are tested.

### Verification and measured result

Focused RON/native tests passed with -count=1. After the Aura boundary correction,
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1` and
`make test` passed. `make lint` passed with 0 issues. Formatting was reviewed;
no unrelated file changed. Native/RON focused tests passed again after lint.
`git diff --check` passed. No environment failure or validation blocker remains.
Final `go run ./cmd/loadcards -json` exits 1 for atomic collection rejection:
**2,214 compiled / 231 rejected records**, **862 accepted / 124 rejected names**,
out of 2,445 records and 986 names. A baseline/final diagnostic comparison removes
only the six Nether Shadow issues; no accepted record regresses and all remaining
diagnostics are unchanged. Estimated one-name/six-record gains match actual gains.
Complete-card RON gameplay coverage adds Nether Shadow. Compilation counts remain
separate from behavior evidence; complete collection gameplay and runtime cutover
are still unmet.

Changed files: engine composable_actions.go/doc.go and new native return test;
RON compiler.go, tap_choice.go, source_zone_upkeep.go, new optional return adapter
and test, source-zone upkeep/condition tests, definition_plan_test.go,
inventory_test.go and numeric_consumers_test.go;
docs/dsl-reference.md, engine-rules-contracts.md and this ledger. No source RON
changed, so vocabulary file hashes need no refresh. No Go definitions, generators
or broad DSL functions were removed; all removal gates remain unmet.

### Ranked next candidates

1. **As-though colored mana spending — Sunglasses of Urza.** Complete inspected
   records have one rule and suggest one name/six records. Establish shared mana
   solver/payment legality and restriction preservation before lowering.
2. **Restricted X payment — Drain Life.** Fixed production restrictions do not
   constrain X. Zero immediate gains until clamped damage life gain works;
   medium/high effort for shared announced-cost decomposition and payment.
3. **Exile-card counters — All Hallow's Eve.** Exact query frames are ready;
   storage precedes predicates/removal, spell exile and batch returns. Zero
   immediate gains; medium/high effort. Compare broader shared candidates afresh.

This run stops after optional exact graveyard source return.


## 2026-10-13 — CONTROLLER-COLORED-MANA-SPENDING

### Candidate comparison and selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Baseline `go run ./cmd/loadcards -json`
exited 1 for collection rejection: 2,214 compiled / 231 rejected records,
862 accepted / 124 rejected names. This was not a build or test failure.

| Candidate | Distinct affected names; estimated full-card gains | Other blockers and downstream value | Effort |
| --- | --- | --- | --- |
| Ranked #1: controller colored mana spending | Sunglasses of Urza; one name/six records | Its complete records contain one static operation. The pool planner already substitutes colors, but the old static conversion is global. A controller rule and common permission graph enable shared spending legality and future restricted-X interaction. | Medium; compiler, native payment integration, lifecycle and gameplay tests |
| Ranked #2: restricted X payment | Drain Life; zero immediate gains, one eventual name/six records | `SpendOnlyColorManaOnX(Black)` needs announced cost decomposition; `TheClampedAmountOfDamageDealtThisWay` still blocks life gain. The prerequisite serves announced X legality and spending restrictions. | Medium/high |
| Ranked #3: exact exile-card counters | All Hallow's Eve; zero immediate gains, one eventual name/record | Exact storage must precede `HasACounterOfType`, `HasNoCountersOfType` and removal. `ExilesWithNumberCounters` and simultaneous each-graveyard return also block the full record. Enables source-zone counter reads and changes. | Medium/high |

Candidate #1 was dependency-ready and had the largest credible complete-card gain
for this run among the ranked candidates. Complete six-reprint Sunglasses and
Drain Life records and the complete LEG All Hallow's Eve record were inspected,
including nested forms. Reprints count once. The two alternatives do not offer
an immediate multi-card unlock. No Go card definition supplied requirements.
Local LEA Oracle text confirms white-to-red controller-only spending. No source
defect was found. CR 106.6, 609.4a/b and 613.11 establish restriction, combined
permission, actual color and post-layer rule contracts.

### Signatures, supported context and reuse boundary

Native: `mage.ControllerManaSpending(from, to core.Color) mage.ContinuousEffect`.
Both arguments must be one of the five colors; native definition validation
rejects invalid colors. The source controller gets the permission while that
source's exact static declaration remains active. RON accepts exactly
`PlayerEffect(You, [MaySpendColoredManaAsThoughItWereAnotherColor(color, color)])`
on a permanent. It rejects other players, multiple operations, malformed arities,
colorless/any-color arguments, tuple-shaped colors and unsupported contexts.
`compileManaSpendingRule(Node) (optionFactory, error)` owns syntax and validation;
`isManaSpendingRule(Node) bool` routes this declaration. Every factory supplies a
fresh static rule. Inventory children have the `color` role.

The old global `ManaConversion` and public `ManaSolverInputs.Conversions` callers
remain intact. The engine adds a per-pool value array of spending permissions.
The existing planner and solver use its transitive graph, so competing destinations,
chains and cycles do not overwrite each other. Actual pool units retain their
colors and restrictions. Pool payment, solver bounds and production demand,
automatic mana activation payment validation and final payment share this graph.
This permission does not produce mana or emit a mana replacement event.

Active static rules are read after object layers and final control enforcement.
Phasing refreshes the rule view without a full layer pass; this was a real
failing gameplay case, not a fixture-only issue. Ability removal and departure
stop permission. Phase-in, restoration and return read current declarations and
control. Clones copy permission arrays by value. Copies retain independent static
rules after donor departure. Queries do not choose or spend. Costs retain the
existing shared transaction and rollback mechanics. No new target, LKI action,
replacement ordering or simultaneous-operation subsystem is needed for this rule.
Restricted X and temporary/other-player spending remain separate contracts.

### Failing-first evidence and card-to-test mappings

Native tests first failed at undefined `ControllerManaSpending`. RON tests then
failed independently at the expected unsupported operation in all six complete
Sunglasses records. Initial implementation tests exposed stale permission after
phasing; post-layer rule reads and phase refresh fixed that failure. Test API
spellings were corrected (`core.ManaCost` for hybrid and X), without weakening
assertions. Declaration and strict-loading tests retain unsupported contexts.

- **Sunglasses of Urza → TestRONManaSpendingSunglasses.** All six complete unchanged
  LEA/LEB/2ED/3ED/CED/CEI records: metadata, paid cast and entry, controller scope,
  control changes, phasing, ability removal/restoration, leave-and-return and clones.
- **TestRONManaSpendingAutomatic.** Real Sunglasses and Plains automatic payment,
  white actual spent color for a red cost, spell resolution and failed payment
  without mana or hand changes.
- **TestRONManaSpendingCopies.** Real Sunglasses and Copy Artifact; independent
  controllers, donor departure and copied static ability removal. This is
  interaction evidence; it does not add another newly completed name.
- **TestRONManaSpendingInvalid / StrictLoad.** Bad arities, players, colors,
  operation lists and contexts; instant/sorcery static rejection; atomic rejected
  loading preserves a supplied registry. Valid isolated loading succeeds.
- **TestColoredManaSpending / ActivationAndHybrid / Declarations.** Native
  affordability and payment purity, actual colors, competing/chained permissions,
  eligible/ineligible restricted mana, clones, automatic activation payment,
  hybrid allocation, announced X bounds, independent players and invalid colors.

### Validation and measured result

Focused RON and native tests passed with `-count=1`. `make build`, `go vet ./...`,
`go test ./pkg/mage/... -short -count=1` and `make test` passed. `make lint`
passed with 0 issues. Formatting and automatic lint changes were reviewed; no
unrelated file changed. Focused native and RON tests passed again after formatting.
`git diff --check` passed. No environment or validation blocker remains.

Fresh loader evidence: **2,220 compiled / 225 rejected records**, **863 accepted /
123 rejected names**, out of 2,445 records / 986 names. Exit 1 remains expected
because collection publication is atomic. Baseline/final diagnostic comparison
removes only six Sunglasses issues and adds none; all other diagnostics are
unchanged. Estimated one-name/six-record gains match actual gains. Complete-card
RON gameplay coverage adds Sunglasses of Urza, separately from compilation counts.

Changed files: new native and RON mana_spending.go/test files; mana.go,
mana_solver.go, mana_activation_payment.go, clone.go, game.go, effect_manager.go,
doc.go; RON compiler.go/inventory.go and three collection-count test files;
dsl-reference.md, engine-rules-contracts.md and this ledger. No source RON changed,
so vocabulary source hashes need no refresh. No production Go card, generator or
broad DSL function was removed: runtime, test migration and removal gates remain
unmet. No unrelated user work was present or changed.

### Ranked next candidates

1. **Restricted X payment — Drain Life.** Shared announced-cost decomposition is
   the next payment prerequisite. Zero immediate full-card gains; clamped damage
   gain still blocks one eventual name/six records. Reassess broader multi-card
   prerequisites before selection.
2. **Exact exile-card counters — All Hallow's Eve.** Shared visit storage and
   query/mutation contract precede spell-exile modifiers and simultaneous returns.
   Zero immediate full-card gains, one eventual name/record.
3. **Clamped actual-damage gain — Drain Life.** Saved damage outcomes are ready;
   save the recipient's pre-damage life/toughness cap and compose the gain with
   those outcomes. Zero immediate gains until restricted X is complete.

This run stops after controller colored mana spending.

## 2026-10-13 — RESTRICTED-X-PAYMENT

### Candidate comparison and selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Baseline
`go run ./cmd/loadcards -json` exited 1 for collection rejection:
2,220 compiled / 225 rejected records, 863 accepted / 123 rejected names.
This was not a build or test failure.

| Candidate | Distinct affected names; estimated full-card gains | Other blockers and downstream value | Effort |
| --- | --- | --- | --- |
| Latest ranked #1: restricted X payment | Drain Life; zero immediate names/records, one eventual name/six records | Clamped actual-damage gain remains. Shared announced-cost decomposition, allocation and solver demand unblock its final gain slice and support restricted additional X costs. | Medium; engine, compiler, payment/production/modifier and rejection tests |
| Latest ranked #2: exact exile-card counters | All Hallow's Eve; zero immediate names/records, one eventual name/record | Exact visit storage must precede predicates/removal. Spell-exile entry counters and simultaneous all-graveyard returns also remain. | Medium/high; storage, identity, replacement and source-zone integration |
| Latest ranked #3: clamped actual-damage gain | Drain Life; zero immediate names/records before restricted X, one eventual name/six records | Saved damage outcomes are ready, but its complete record first needs restricted payment. Recipient pre-damage life, toughness and planeswalker loyalty caps need capture. | Medium; typed producer facts, replacement and damage-result tests |

The first candidate was dependency-ready and an unblocking prerequisite, so it
was selected. It completes one payment contract without combining the later
life-gain capability. There is no immediate multi-card gain among these ranked
candidates; this run claims no such gain. All six complete Drain Life records and
the complete All Hallow's Eve record, including their nested forms, were inspected.
Reprints count once. Go card definitions did not supply requirements.
Local LEA Oracle text agrees with the black-only X restriction. No source defect
was found. CR 107.3a/b, 601.2f/g/h, 106.6 and 609.4a/b supply announced X,
alternative-cost limits, total-cost locking, complete payment, retained mana
restrictions and as-though spending. The reusable engine work used the
`implement-engine` skill; no removed card/set skill or generator was invoked.

### Signatures, supported context and reuse boundary

Native: `mage.WithXSpendingColor(core.Color) mage.ActionOption` on a spell action
with a printed X mana cost. Definition validation rejects duplicate rules,
invalid colors, non-spell actions, missing printed X and the unsupported ordinary
spell text-continuation marker. `SpellContextForCard` builds a fresh value context
with `XSpendingRestricted`, `XSpendingColor` and `RestrictedGeneric`.

RON: exactly `CastEffect(SpendOnlyColorManaOnX(color))` on an ordinary instant or
sorcery with printed X. One of the five color symbols is required. Duplicate
CastEffect rules, malformed arities, tuple colors, colorless/any-color arguments,
permanents, modal bodies and resolving declarations reject. The ordinary spell
factory applies a fresh option; it does not install a text-continuation marker.
`compileXSpendingColor(Node, bool) (*core.Color, error)` owns spelling and validation.
Inventory assigns the argument the `color` role. No card-name branch or production
fallback was added. The complete Drain Life records remain rejected.

The engine retains the generic total cost and records which generic units came
from announced X in the selected mana cost and additional costs. Increases remain
unrestricted; reductions remove restricted X first. Announced X stays unchanged.
Channel's existing life shortcut can cover only unrestricted generic payment.
An alternative mana cost without X requires X zero. Negative X rejects before
payment. The cast-from-zone any-color permission and ordinary as-though graphs
can supply the restricted color while preserving actual spent colors.

`paymentCostForContext` supplies a color-demand allocation view to the existing
pool planner, solver bounds and production caps. This is not a change to the
mana cost. Existing per-unit spending restrictions, mana replacements, exact
mana-ability activation payments and shared cost transactions remain in use.
Activation costs do not inherit the spell restriction. Failed planning/payment
keeps live resources, zones, events and stack unchanged. Copies retain announced
X without paying again. Clones keep separate pools and stack state. Phasing can
suspend a supplying permission; queries do not produce mana or modify contexts.

`MaxXValue` counts every X symbol in its supplied cost using a copied trial
context. It retains its existing supplied-cost query boundary: it does not add
spell taxes, reductions or additional costs. A resolved-cost query supplies the
remaining amount in `RestrictedGeneric`. No battlefield source reference, LKI,
trigger timing, new replacement ordering or simultaneous-operation service is
needed for this spell declaration. Existing callers and broad DSL exports remain
intact. Only the established contract is documented in doc.go, dsl-reference.md
and engine-rules-contracts.md.

### Failing-first evidence and card-to-test mappings

Native tests first failed at undefined `WithXSpendingColor` and
`SpellPaymentContext.RestrictedGeneric`. RON tests separately failed at the
unsupported casting restriction in all six complete Drain Life records and in
five-color declaration fixtures. Existing test API names were corrected without
weakening assertions. Later failing-first checks exposed the cast-from-zone
any-color interaction, unsupported text-continuation admission and nonzero X
with an alternative cost without X. All were corrected in the shared contracts.

- **Drain Life → TestRONRestrictedXGameplay.** All complete unchanged
  LEA/LEB/2ED/3ED/CED/CEI records reject at the separate clamped gain diagnostic.
  Test-only AST fixtures omit that unsupported gain instruction, retain the
  unchanged metadata, casting restriction, target and damage instruction, and
  execute successful and rejected payments plus X damage. These are payment
  prerequisite tests, not complete-card gameplay coverage.
- **TestRONRestrictedXColors / Invalid / StrictLoadAndInventory.** Five supported
  colors, invalid forms/contexts, missing X, duplicate declarations, atomic
  rejection preserving a supplied registry, isolated successful loading and
  inventory roles. Factories are rebuilt separately for gameplay runs.
- **TestRestrictedXPayment / Planning.** Hand and alternate-zone payments,
  insufficient color despite sufficient total mana, exact allocation, eligible
  and ineligible restricted mana, invalid decomposition, query purity, X bounds,
  spell copies and game clones.
- **TestRestrictedXAutomaticAndPermissions / AnyColorProduction /
  AlternateAnyColor / ReplacementsAndConversion.** Automatic land production,
  permission phasing, actual spent colors, variable-color demand, zone-casting
  permission, mana-type replacements and repeated converters preserving final
  color demand. AutomaticRollback checks untouched sources after rejection.
- **TestRestrictedXModifiers / MultipleSymbolsAndCosts / HybridAllocation /
  AlternateWithoutX / Declarations.** X zero, taxes then reductions, retained
  announced X, Channel, nonmana-cost rollback, repeated X symbols, additional X,
  hybrid allocation, zero/illegal alternative-cost X and definition rejection.

### Validation and measured result

The final focused commands passed with fresh counts:
`go test ./pkg/mage/dsl/ron -run TestRONRestrictedX -count=1` and
`go test ./pkg/mage -run TestRestrictedX -count=1`.
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1` and
`make test` passed after the final semantic change. `make lint` passed with
0 issues. Automatic formatting only changed import grouping in the new tests;
no unrelated file changed. Both focused commands passed again after formatting.
`git diff --check` passed. No environment or validation blocker remains.

Final `go run ./cmd/loadcards -json` exits 1 for expected atomic collection
rejection: **2,220 compiled / 225 rejected records**, **863 accepted / 123 rejected
names**, out of 2,445 records / 986 names. The baseline/final diagnostic comparison
has zero added or removed rejected positions. Exactly six Drain Life messages
change from `unsupported casting restriction (SpendOnlyColorManaOnX)` at column
163 to `unsupported value (TheClampedAmountOfDamageDealtThisWay)` at column 302.
All other diagnostics are identical; every previously accepted record stays
accepted. Estimated zero-name/zero-record gains match actual gains. Full-card
RON gameplay coverage is unchanged; the payment prerequisite adds native and
RON fragment evidence separately from collection compilation.

Changed files: new native and RON restricted_x.go/test files; activated.go,
core/mana_restriction.go, definition.go, mana.go, mana_solver.go, mana_system.go,
spell_payment.go, text_spell_continuation.go and doc.go; RON compiler.go and
inventory.go; dsl-reference.md, engine-rules-contracts.md and this ledger.
No RON source changed, so vocabulary source hashes need no refresh. No Go card
definition, generator or broad DSL function was removed: complete collection
loading, gameplay migration and startup removal gates remain unmet. No unrelated
user work was present or changed. This slice is ready for its descriptive commit.

### Ranked next candidates

1. **Clamped actual-damage gain — Drain Life.** One immediate name/six records
   estimated now that restricted X works. Reuse saved actual-damage results and
   add recipient pre-damage life, creature toughness and planeswalker loyalty
   caps. Keep prevention, redirection, departure and instance frames exact.
2. **Exact last-drawn hand-card discard cost — Jandor's Ring.** One name/two
   records estimated. Complete ARN and 3ED records use one exact-card discard,
   fixed mana/tap costs and DrawACard. The last-draw hand visit and transactional
   discard adapter need assessment; a returned card cannot supply that visit.
3. **Exact exile-card counters — All Hallow's Eve.** Zero immediate gains;
   shared visit storage precedes counter queries/removal. Spell-exile counters
   and simultaneous each-graveyard return still block one eventual name/record.

This run stops after restricted X payment.

## 2026-10-13 — CLAMPED-DAMAGE-GAIN (complete)

### Baseline, candidates and selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Baseline strict loader exit 1 was
collection rejection, not a build failure: 2,220 compiled / 225 rejected
records and 863 accepted / 123 rejected names, out of 2,445 records / 986 names.
The top status and latest ledger entry supplied the candidates. Complete RON
records, nested forms and targeted vocabulary evidence were inspected.

| Candidate | Distinct affected cards | Estimated full-card gain | Other blockers and downstream benefit | Effort |
| --- | --- | --- | --- | --- |
| Clamped actual-damage gain | Drain Life | 1 name / 6 records | Restricted X, singular any-target damage and life gain are ready. Saved recipient caps complete the actual-damage result contract. | Small engine/result adapter; medium interaction testing. |
| Exact last-drawn discard cost | Jandor's Ring | 1 name / 2 records | Needs turn-local last-draw hand-visit history and exact transactional discard. Fixed mana, tap and draw forms are ready. Reusable exact-card payment prerequisite. | Medium history/payment work and rollback testing. |
| Exact exile-card counters | All Hallow's Eve | 0 immediate names / records | Visit storage precedes counter predicates/removal. Spell exile with counters and simultaneous each-graveyard returns also remain. Reusable source-zone counter prerequisite. | Medium storage/lifecycle work; later slices still needed. |

The first ranked candidate was dependency-ready and had the largest immediate
record gain of these candidates for the least engine work. It was selected as
one complete singular result capability. Reprints count once. This selection
does not claim multi-card gameplay gain from first-error frequency.

### Established contract and boundaries

Engine: `ClampedDamageAmount(results *DamageResultsBinding) ResolutionValue`
reads one positive saved singular outcome. Missing or nonsingular results return
zero. Nil and unbound declarations reject during definition validation. The
shared damage executor saves final recipient identity and pre-damage life or
loyalty after replacement/prevention. The value limits actual damage, not damage
requested, life lost or loyalty removed. Damage itself is not clamped.

Creature toughness uses current exact-visit information at the value read,
including departure LKI or phase information. Existing marked damage does not
reduce the cap. Return cannot replace the former visit. A creature planeswalker
uses both limits; negative limits yield zero. Local Oracle evidence in
`data/LEA.json` explicitly requires pre-damage life and loyalty, but does not
require pre-damage toughness. CR 120.4b–d and 608.2h–i establish timing and reads.
There was no RON source defect or source correction.

RON: symbol `TheClampedAmountOfDamageDealtThisWay` after the latest local
singular ordinary `SpellDealsDamage` or `PermanentDealsDamage`. Existing typed
player/permanent recipients and singular any-target bindings are reused.
Collections, malformed tuple forms, unbound reads, branch escapes, entry and
draw-replacement bodies reject at compilation. Plans retain singular/plural
information. Each factory creates its own existing damage-result declarations.
No global last-action slot, card-name branch or Go-card fallback was added.

Reuse: `DamageOutcomesInto`, `DealDamageFrom`, existing damage replacements and
prevention, exact permanent information, result cloning, lexical continuation
lowering, definition validation and strict atomic loading. A failing delayed
read exposed missing outcome capture in `capturedDelayedBody`; step and
permanent-event delayed bodies now clone/save/restore the damage-results map.
This supplies the same capability through delays, without a new damage service.
No plural clamping or simultaneous-operation adapter was introduced; existing
batch damage and replacement ordering remain unchanged. Costs keep their shared
restricted-X and transactional contracts.

### Failing-first evidence and card-to-test mappings

Native tests failed first at undefined `ClampedDamageAmount` after correcting
test counter names. RON gameplay tests failed at the unsupported clamped symbol
in all six complete records. The delayed RON assertion then failed because its
saved outcome was absent; shared frame capture fixed that failure. Later short
suite failures were four fixed inventory counts, now updated to the measured
2,226/219 and strengthened with an explicit Drain Life acceptance assertion.
No gameplay assertion was weakened.

- **Drain Life (LEA, LEB, 2ED, 3ED, CED, CEI) →
  TestRONClampedDamageDrainLife and TestRONRestrictedXGameplay.** Complete
  unchanged records execute payment, X, any target, damage and gain. Cases cover
  player life, creature toughness, planeswalker loyalty, prevention, X zero,
  returned illegal targets, phasing, timing, copied spells and game clones.
  The previous payment-only AST fixture now uses the entire record and checks
  gain as well as successful/failed payments and rollback.
- **TestClampedDamageAmount / Definition.** Actual final recipient after
  redirection, replacement increases, marked damage, pre-damage life/loyalty,
  negative caps, creature planeswalkers, current toughness, exact departure,
  return, phase snapshots, zero-output clearing, clone isolation and declaration
  errors. These are shared engine contract tests.
- **TestClampedDamageDelayed / TestRONClampedDamageScopes.** Step and departure
  delays, waiting-query purity, cloned delayed frames, latest local producers,
  branch-local producers and separate factory instances.
- **TestRONClampedDamageInvalid / AtomicLoad.** Invalid symbols/arity,
  collection and context rejection, absent/out-of-scope producers, complete
  real-record isolated loading, atomic failed-load registry preservation and
  the existing value inventory role.

### Validation and measured result

Final focused checks passed:
`go test ./pkg/mage/dsl/ron -run TestRONClampedDamage -count=1` and
`go test ./pkg/mage -run TestClampedDamage -count=1`.
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1` and
`make test` passed after the inventory assertions were updated.
`make lint` passed with 0 issues after removal of one redundant test variable
type. Automatic changes only regrouped imports in the new tests. Both focused
commands and the existing restricted-X/damage-outcome tests passed again after
formatting. `git diff --check` passed. No environment or validation blocker remains.

Final strict loader exit 1 remains expected atomic collection rejection:
**2,226 compiled / 219 rejected records**, **864 accepted / 122 rejected names**.
Exactly six Drain Life diagnostics disappear; there are no added rejections or
changed remaining diagnostics. Every previously accepted record stays accepted.
Estimated one-name/six-record gains match actual gains. Complete-card gameplay
coverage gains Drain Life (one unique name), backed by all six complete records.

Changed files: native clamped_damage.go/test, damage_system.go,
composable_conditions.go and doc.go; RON clamped_damage_test.go, damage_outcome.go,
sacrifice_value.go, damage_outcome_test.go, restricted_x_test.go,
definition_plan_test.go, inventory_test.go and numeric_consumers_test.go;
dsl-reference.md, engine-rules-contracts.md and this ledger. No RON source changed,
so vocabulary source hashes need no refresh. No Go definitions, generator or DSL
exports were removed; complete collection loading and runtime removal gates remain.

### Ranked next candidates

1. **Exact last-drawn discard cost — Jandor's Ring.** One name/two records;
   exact turn-local hand visits and transactional discard precede its adapter.
2. **Color-modified spell copying — Fork.** One name/six records estimated;
   complete records require a copiable red exception and optional retargeting.
   Assess the shared spell-copy and copiable-value contracts before selecting.
3. **Exact exile-card counters — All Hallow's Eve.** Zero immediate gains;
   counter storage precedes queries/removal, spell exile and batch return.

This run stops after clamped damage gain.

## 2026-10-13 — LAST-DRAWN-DISCARD-COST

### Baseline and candidate selection

The working tree was clean. Fresh `go test ./pkg/mage/dsl/ron -count=1` and
`make build` passed. The strict loader exited 1 for collection rejection:
2,226 compiled / 219 rejected records; 864 accepted / 122 rejected names,
out of 2,445 records / 986 names. This was not a compilation or environment
failure. Top status, the latest ledger entry, complete nested RON records and
targeted vocabulary keys supplied the comparison.

| Candidate | Distinct affected cards | Estimated full-card gain | Other blockers and downstream benefit | Effort |
| --- | --- | --- | --- | --- |
| Exact last-drawn discard cost | Jandor's Ring | 1 name / 2 records | Mana, source tap and drawing are ready. Per-player ID history exists but lacks the exact hand visit. Extends shared exact discard payments without another transaction service. | Small history/cost adapter; medium lifecycle and rollback tests. |
| Color-modified spell copying | Fork | 1 name / 6 records | Existing stack copy and optional retargeting are available. The explicit red copiable exception needs a new contract and copy/provenance tests. Enables reusable modified spell copies. | Medium copy mechanics and broader interaction tests. |
| Exact exile-card counters | All Hallow's Eve | 0 immediate names / records | Counter storage precedes predicates/removal. Spell exile with counters and simultaneous each-graveyard return still block the complete record. Enables the source-exile counter chain. | Medium lifecycle prerequisite; multiple later adapters. |

Candidate 1 was selected after comparing all three. The immediate unique-card
gain ties Fork, while exact discard reuses more existing execution and has a
smaller implementation boundary. Fork's larger reprint count is not a larger
unique-card gain. Exile counters have no immediate unlock and need later slices.
No multi-card unlock is claimed for these candidates. This run establishes one
complete exact identity payment shape and stops.

### Established contract and reuse boundary

Engine: `DiscardLastDrawnCardCost() Cost`. The payer's last successful draw this
turn must remain in the exact original hand visit. `BasePlayer.DrawCard` saves
both ID and visit. The existing `LastDrawnCardID()` API stays unchanged; private
`lastDrawnHandCard() capturedCard` supplies history to the cost and wrapper clone
extraction. Direct player clones also preserve the token. History clears with
the ID at final cleanup; the existing pending-cleanup timing is unchanged.

A later draw replaces history. When that later draw leaves hand, an older draw
cannot pay. Non-draw hand additions, departure and return cannot provide the
visit. Replaced or failed draws keep history; replacement bodies that actually
draw update it. Each player's history remains separate, including opponent-turn
draws. Queries and exact selection make no choices and do not consume RNG.

RON: `DiscardCard(TheLastCardDrawnThisTurn)` in activation cost declarations,
alone or inside `And([...])`. Outer arity is one; the argument must be exactly
the symbol. Each factory builds fresh costs. Other references, tuple arguments,
wrong arity, resolving actions, optional resolution payments and entry operations
reject. The compiler owns syntax checks; the engine owns draw history and payment.

The existing `discardCost` locks the selected visit on transaction preparation
and rechecks that the saved latest draw still matches before commit. Shared
full-payment simulation detects overlaps and failed discard replacements before
live mana/tap payment. Existing discard replacement ordering, batch events,
source identity, stack copying and cloning are reused. Hidden replacement
destinations may pay this identity cost; no characteristics are required.
Effect-only replacement rules do not apply to cost-caused discard.

Local Oracle evidence: `data/ARN.json` says “Discard the last card you drew this
turn.” CR 121.1–2 and 121.5–7 establish draw timing and replacements; CR 400.7
requires new visits after zone changes; CR 118.11, 601.2h and 701.9 establish
replacement payment and complete-cost rules. No source defect was found. No
RON record or vocabulary source hash changed. No global action-result slot,
Go-card fallback, card-name executor or partial production registry was added.

### Failing-first evidence and card-to-test mappings

After correcting test API names, native tests failed only at undefined
`DiscardLastDrawnCardCost`. Complete RON gameplay tests failed at unsupported
`DiscardCard` for both records before implementation. A fixture initially used
a different set for Library of Leng, then exposed that its full RON declaration
is still unsupported. It now uses a synthetic established effect-discard rule
for the interaction, without substituting a Go card definition. Native tests
supply general discard replacements through existing test fixtures.

- **Jandor's Ring (ARN, 3ED) → TestRONLastDrawnCostRing.** Both entire unchanged
  records cast for six mana, pay two mana/tap/exact discard at activation, and
  draw at resolution. Source departure, control change, phasing, return, countered
  abilities, copies under another controller and independent clones retain timing
  and controller behavior. The effect-only discard fixture cannot replace costs.
- **TestRONLastDrawnCostRollback / Instances.** Missing history, non-draw hand
  additions, returned visits, departed latest draws, insufficient mana, tapped or
  phased sources, wrong controllers and turn reset reject without live spending.
  Separate factories/players pay their own draw; a new resolution draw can pay
  a later activation.
- **TestLastDrawnCostIdentity / Transaction.** Latest card rather than an older
  hand card, other-player draws, failed empty draws, cleanup reset, returned or
  departed visits, post-preparation newer draws, duplicate costs and total-cost
  failure. Queries are pure and failures preserve mana, tap state and zones.
- **TestLastDrawnCostClone / Interleaved.** Direct and wrapped player extraction,
  separate clone history/zones, independent interleaved payer transactions, absent
  payer rejection, valid engine definitions and no card-choice callbacks.
- **TestLastDrawnCostDrawReplacement / DiscardReplacement.** Replaced versus
  replacement-generated draws; hidden library destination payment, discard event
  facts, prevented discards, replacement rechecks and full-cost rollback.
- **TestRONLastDrawnCostInvalid / AtomicLoad.** Malformed/unbound card references,
  unsupported contexts, atomic registry preservation on failure and successful
  loading of the two complete equivalent reprints.

Four fixed inventory assertions now use the measured 2,228/217 counts, with
an explicit Jandor's Ring acceptance assertion. No gameplay assertion was removed
or weakened. Compilation progress and gameplay coverage are separate: the
complete-card gameplay mapping gains one unique card, Jandor's Ring.

### Validation and measured result

Passed:
`go test ./pkg/mage/dsl/ron -run TestRONLastDrawnCost -count=1`,
`go test ./pkg/mage -run TestLastDrawnCost -count=1`, `make build`,
`go vet ./...`, `go test ./pkg/mage/... -short -count=1`, and `make test`.
The first lint run identified an unnecessary constructor wrapper and an explicit
inferred test variable type. Both were corrected. Automatic formatting only
regrouped imports in the new native tests. Focused tests, build and vet passed
again after those changes. Final `make lint` passed with **0 issues** and
`git diff --check` passed. No environment or validation blocker remains.

Final `go run ./cmd/loadcards -json` exited 1 for expected atomic collection
rejection: **2,228 compiled / 217 rejected records**, **865 accepted / 121 rejected
names**. Both Jandor's Ring diagnostics disappear. There are no added rejections
or changed remaining diagnostics; every previously accepted record stays accepted.
Estimated one-name/two-record gains match actual gains. Complete-card gameplay
coverage gains one unique name, with both complete RON reprints tested.

Changed files: `pkg/mage/cost.go`, `player.go`, `clone.go`, `doc.go` and new
`last_drawn_cost_test.go`; RON `compiler.go`, new `last_drawn_cost_test.go`,
`definition_plan_test.go`, `inventory_test.go`, `numeric_consumers_test.go`;
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger.
RON sources and vocabulary hashes are unchanged. Current callers and Go card
files remain. No DSL export was removed; collection loading, startup cutover,
Go-card deletion and caller-audited DSL removal gates remain unmet.

### Ranked next candidates

1. **Color-modified spell copying — Fork.** One name/six records estimated;
   establish the copiable red exception through shared stack copying, with
   optional retargeting, source identity, copy-of-copy and clone tests.
2. **Exact exile-card counters — All Hallow's Eve.** Zero immediate gains;
   establish visit storage before counter predicates/removal, spell exile with
   counters and simultaneous each-graveyard creature returns.
3. **Counter-bound duration — Cyclopean Tomb.** Zero immediate gains estimated;
   source/subject counter history, `EachablePermanent` duration binding and
   delayed counter cleanup still need contracts before complete-card execution.

This run stops after exact last-drawn discard cost.

## 2026-10-13 — COLOR-MODIFIED-SPELL-COPY

### Candidate comparison and selection

The working tree was clean. Fresh `make build` and the complete RON package tests
passed. Fresh loader collection rejection exited 1 with **2,228 compiled / 217
rejected records**, **865 accepted / 121 rejected names**. This was collection
rejection, not a build or test failure. All compared cards' complete RON records
and nested forms were inspected; reprints count once.

| Candidate | Distinct affected names | Estimated full-card gains | Remaining blockers / downstream benefit | Implementation and test effort |
| --- | --- | --- | --- | --- |
| Color-modified spell copying | Fork (six records) | One name / six records | No other Fork blocker. Shared copiable color exceptions use existing stack copying and retargeting. | Small lowering and copy-service extension; broad identity, color, target, copy and clone tests. |
| Exact exile-card counters | All Hallow's Eve (one record) | Zero | Needs spell exile with counters, predicates/removal, and simultaneous each-graveyard creature returns after storage. | Medium visit-storage prerequisite, then several dependent slices. |
| Counter-bound duration | Cyclopean Tomb (five records) | Zero | Needs each-subject duration binding, source placement/removal history and delayed cleanup. | Medium duration contract plus independent history work. |

Ledger candidate #1 was ready but supplied one unique-card gain, not multi-card
value. The other two ranked candidates had no immediate complete-card gain and
larger remaining dependencies. A fresh diagnostic grouping found no shared exact
first-error signature across multiple unique names. That grouping was not used
as an unlock estimate. Complete Fork records establish the bounded positive gain.
Fork therefore wins this comparison. No multi-card gain is claimed.

Before editing, the contract was fixed-color copying with optional retargeting,
source identity, copy-of-copy and clone isolation. The implementation explicitly
limits the RON context to resolving spell bodies with an instant-or-sorcery target
domain. Permanent-spell and other copy exceptions are separate capabilities.

### Established contract, dependencies and reuse

Engine API:
`CopyInstantOrSorceryWithColors(StackObjectRef, ...core.Color) ResolvingEffect`.
The five Magic colors are legal; an empty list makes colorless copies. Native
references can be target or saved spell bindings. A departed spell cannot supply
action LKI, and a permanent spell does not qualify. Definition checks reject nil,
unbound and invalid-color operands. Colors are saved independently of caller
slices and normalized through the existing retained-color modification.

The copied card receives its copiable color exception before shared target
choices. `CopyRetainColors`, `CopiableValues.WithModifications` and checked text
builders retain the exception and omit color-defining abilities. An immutable
modified provenance plan follows the copy. Ordinary stack color changes remain
separate. Later copies inherit the exception but exclude those ordinary changes.

Private `copyStackObjectWithCard` is extracted from the existing copy service.
Both public copy methods keep their signatures and behavior. The new action uses
that service for fresh card/stack identities, controller and ownership, X, mode,
selected targets, distributions, casting facts and saved payment frames. Copies
are not cast and do not pay again. Existing target services retain cancelled or
illegal new choices, recheck protection using the copy's colors, and preserve old
illegal target visits. Departure/return and phasing cannot restore old targets.
No replacement or payment ordering service is duplicated.

RON signature:
`CopySpellAndMayChooseNewTargetsWithEffects(Ref_TargetSpell,
[SetColor(SimpleColorList([color, ...]))])`.
The reference must be a local singular target whose domain requires instants or
sorceries. Direct type predicates qualify. A conjunction needs a qualifying
conjunct; every disjunction branch must qualify. One fixed `SetColor` is supported.
Empty fixed color lists are valid. Event/union references, activation, delayed,
replacement and entry contexts, variable colors, `Colorless` symbols, incorrect
arity and additional modifications reject before factory publication. Every
factory builds fresh target bindings. The semantic inventory supplies the fixed
color's existing color-word role so pending spell text can revise the exception.

Local Oracle evidence in `data/LEA.json` matches the unchanged Fork RON records.
CR 707.2, 707.9b–d and 707.10–10c supply the copy contract, including the explicit
Fork example. CR 400.7, 608.2b, 612.1–3 and 702.16 supply identity, resolution,
text revision and protection requirements. No source defect was found. RON
sources and vocabulary source hashes are unchanged. No Go fallback, card-name
branch, global action-result slot or partial production registry was added.

### Failing-first evidence and card-to-test mapping

After correcting test-only API names, native tests failed at the undefined copy
constructor and RON gameplay/atomic-load tests failed at the unsupported copy
action in all six Fork records. A separate text-revision regression failed at
“text word has no validated semantic role (Red)” before its inventory adapter.
An initial modal fixture let the default copy chooser select another player;
the final fixture explicitly cancels retargeting to test retained modal targets.
No assertion was weakened to accept that fixture error.

- **Fork (LEA, LEB, 2ED, 3ED, CED, CEI) → TestRONSpellCopyColorsFork.** All six
  complete records pay RR, select an instant, and copy it only at resolution.
  Ancestral Recall checks kept/new targets, departed targets, a new cast of the
  same physical card, copied Forks, independent clones, ordinary recoloring,
  copy-of-copy and countered Forks. Unpaid announcement leaves state unchanged.
- **Fork → TestRONSpellCopyColorsHurricane.** A sorcery copy retains X=3 and
  distinct source/controller facts. Its red damage is prevented on a protected
  flying creature; the green original damages it. Both player damage batches
  execute. Copies produce no cast history and enter no graveyard.
- **Fork → TestRONSpellCopyColorsRetarget.** Psionic Blast proves that red
  protection excludes a new target before the choice. Legal retargeting succeeds;
  cancellation and illegal answers retain the old target, which is then illegal
  for the red copy. The copy's independent self-damage uses its controller.
- **Fork → TestRONSpellCopyColorsModal / Fireball.** Healing Salve retains its
  chosen mode and target group, without another mode choice. Fireball retains X,
  target count, division and rounding without repaying its casting surcharge.
- **Fork → TestRONSpellCopyColorsText.** The pending color word can change from
  red to white; a separate clone keeps red. A later copy inherits white.
- **TestRONSpellCopyColorsShapes / Invalid / Contexts / AtomicLoad.** Fixed
  colorless and multicolor lists, qualifying target domains, fresh factories,
  malformed syntax, unbound references, unsupported contexts and atomic loading
  of the six equivalent complete reprints.
- **TestSpellCopyColorsIdentity / Retarget.** Fresh ownership/source identity,
  modes, X, allocation transfer to retargeted subjects, independent ordinary color
  changes, copy-of-copy, clones, protection, cancelled choices, phasing and exact
  returned-target visits.
- **TestSpellCopyColorsDefinition / Exceptions.** Invalid/unbound declarations,
  absent spell references, colorless/multicolor exceptions, color-definition
  omission, retained immutable snapshot provenance and independent controllers.

Four fixed inventory assertions use the measured **2,234 / 211** record counts;
an explicit Fork acceptance assertion is added. Compilation gains and gameplay
coverage remain separate: complete-card gameplay coverage gains one unique name,
with all six reprints tested. No DSL export was removed and no Go card definition
or generator was changed; their removal gates are still unmet.

### Validation and measured result

Passed, in fast-fail order:
`go test ./pkg/mage/dsl/ron -run TestRONSpellCopyColors -count=1`,
`go test ./pkg/mage -run TestSpellCopyColors -count=1`, `make build`,
`go vet ./...`, `go test ./pkg/mage/... -short -count=1`, `make test`,
and the skill-required `go test ./... -short -count=1`.
`make lint` passed with **0 issues**. Automatic formatting changed only import
grouping in the new tests. Both focused packages, build and vet passed again
after formatting. `git diff --check` passed. No validation or environment blocker
remains.

Final `go run ./cmd/loadcards -json` exited 1 for expected atomic collection
rejection: **2,234 compiled / 211 rejected records**, **866 accepted / 120 rejected
names**. An exact comparison against the fresh baseline removes only the six
Fork diagnostics; it adds no rejection and changes no remaining diagnostic.
All previously accepted records stay accepted. Estimated and actual gains match:
one unique name and six records. Complete-card gameplay coverage gains Fork,
with every unchanged reprint tested.

Changed files: engine `spell_copy.go`, new `spell_copy_colors.go`, new
`spell_copy_colors_test.go` and `doc.go`; RON `compiler.go`, `spell_scope.go`,
`inventory.go`, new `spell_copy.go`, new `spell_copy_colors_test.go`,
`definition_plan_test.go`, `inventory_test.go` and `numeric_consumers_test.go`;
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md` and this ledger.
RON data and vocabulary source hashes are unchanged. Production callers,
Go card definitions and generators remain until their cutover gates pass.

### Ranked next candidates

1. **Dynamic mana-production options — Fellwar Stone.** Assess the complete
   `AnyManaColorAPermanentCouldProduce` shape, “could produce” choices,
   replacement effects and pure mana-solver metadata. Estimated one name/one
   record if the complete source-production query can be established in one run.
2. **Exact exile-card counters — All Hallow's Eve.** Zero immediate gains;
   exact-visit storage before counter predicates/removal, spell exile with
   counters and simultaneous each-graveyard creature entry.
3. **Counter-bound duration — Cyclopean Tomb.** Zero immediate gains estimated;
   each-subject binding, source counter history and delayed removal still need
   contracts before complete-card execution.

This run stops after fixed-color instant and sorcery copying.

## 2026-10-13 — DYNAMIC-MANA-COLOR-OPTIONS (complete)

### Baseline, comparison and selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Baseline loader exit 1 was expected
atomic collection rejection: **2,234 compiled / 211 rejected records** and
**866 accepted / 120 rejected names**, out of 2,445 records and 986 names.
There was no build, test or environment failure. Complete RON records and nested
forms were inspected; reprints count once in the following comparison.

| Candidate | Distinct affected names | Estimated complete-card unlocks | Other blockers and downstream benefit | Full implementation/test effort |
| --- | --- | --- | --- | --- |
| Dynamic mana color options | Fellwar Stone | 1 name / 1 record | No other card rule blocked. Shared finite-option metadata and replacement-aware recursive queries also serve native opponent-land rules, storage-land queries and future mana-color declarations. | Moderate; one run, with solver, choice, recursion and replacement tests. |
| Exact exile-card counters | All Hallow's Eve | 0 | Storage precedes counter predicates/removal; spell exile with counters and simultaneous all-graveyard entry still block the complete card. Enables those dependent source-zone actions. | Moderate for storage; larger for the complete card. |
| Counter-bound duration | Cyclopean Tomb | 0 | Each-subject binding, source counter placement/removal history and delayed removal remain. Enables counter-conditioned layers and the Tomb's linked cleanup. | Moderate for duration; larger for the complete card. |

Ranked candidate #1 was dependency-ready and gained a complete card. Existing
mana abilities, type replacements, solver choice enumeration and payment
transactions supplied its dependencies. Neither alternative could unlock a
complete card in this run without several additional capabilities. The source
query is a reusable engine prerequisite; RON owns only spelling and validation.
Before editing, acceptance cases were fixed: empty production, one legal color
choice, costs/tapped status ignored in queried lands, current layers/control,
phasing, recursion, replacement order, clones/copies and failed payment rollback.

### Established contract and reuse boundary

Engine API:
`NewManaAbilityOfColorsOpponentPermanentsCouldProduce(filter PermanentFilter)
*ManaAbility`. It creates one tap-symbol ability adding one available color.
`Game.ManaTypesAbilityCouldProduce(Ability, uuid.UUID) []Color` retains its
signature. Present, unphased matching subjects supply current production; no
land LKI or copied spending restriction/rider supplies output. The source's live
controller defines its opponent. Empty options still allow tap activation and
produce nothing without a stack object. Return is a new current subject.

Query-local maps use source incarnation and ability ID. A least fixed point
starts with no colors, includes all shared mana replacement orders, and permits
colors to become defined through recursive dependencies. Unanchored cycles stay
empty. No global result slot or game mutation is used. Direct saved counter-cost
amounts use a hypothetical positive payment without testing affordability or
reading the active resolution's number bindings. Fixed, conditional and dynamic
production use shared engine declarations. Unknown legacy bodies supply no
invented output. Queries do not choose, pay, use RNG or execute riders.

Finite color options use comparable value metadata. Existing solver enumeration,
preservation, preferred-color matching, manual production and planned-source
rechecks use the same options. Selection becomes concrete before the shared
replacement event. Actual choices/order and payment rollback remain in existing
services. Copies and clones query their own source/controller and game state.
No engine state, zone, replacement or payment service is duplicated.

RON signature/context:
`Activated(TapPermanent(ThisPermanent), ActionList([AddMana(
AnyManaColorAPermanentCouldProduce(And([characteristicFilter,
ControlledByAPlayer(Opponent)])))]))`. Exactly one characteristic filter and one
opponent clause are required, in either order. The existing pure mana-condition
characteristic adapter supplies nested conjunctions/disjunctions. Targets, other
players/costs/tap subjects, modifiers, repetitions, multiple actions, spells,
entry and replacement bodies reject before publication. Factories create fresh
abilities and immutable query plans. Inventory descendants now have permanent
filter roles. Existing production callers and raw metadata APIs remain.

Local `data/DRK.json` Oracle text matches the unchanged Fellwar Stone record.
CR 106.7 and 605.2–3 establish queries and immediate mana activation; existing
mana replacement contracts supply CR 106.6/106.12b and 616 ordering.
[Wizards' Commander Masters release notes](https://magic.wizards.com/en/news/feature/commander-masters-release-notes)
confirm one mana, tapped/cost checks, all replacement orders and restriction/rider
exclusion. No source defect was found; RON and vocabulary hashes are unchanged.
Rainbow Vale still rejects at its source reference in a delayed control body;
City of Shadows still rejects at its exile cost. These are separate compiler
gaps. Their complete records were inspected; native fixtures cover relevant
dynamic output/riders without claiming complete-card coverage for either.

### Failing-first evidence and card-to-test mapping

Initial native tests failed at the missing constructor; complete Fellwar Stone
RON tests failed at its unsupported production tuple. After that implementation,
a direct counter-payment regression failed with no color. A recursive replacement
regression failed with only green instead of green and white, proving that a
path guard was insufficient. The fixed point passed it. An absent-ability
regression exposed a nil-query panic, then passed after the query guard.
Test-only phase/event API mistakes were corrected; a clone assertion now reads
the live permanent because the engine uses copy-on-write. Assertions were not
weakened. Two unsupported source-card fixtures were replaced by explicit native
contracts; their collection diagnostics remain unchanged.

- **Fellwar Stone (DRK) → TestRONDynamicManaColorsFellwarStone.** Complete casting
  pays two generic mana, resolves to the battlefield and activates immediately.
  Empty output, blue/green automatic payments, unavailable colors, two-color
  overpayment, source/land phasing, ability removal, clone isolation and failed
  casting payment preserve the required resources and identities.
- **Fellwar Stone → ChoiceAndRegistry / CopyArtifact.** Strict complete-record
  loading, fresh factory identities, pure repeated queries, offered legal colors,
  invalid-answer fallback, one mana from multicolor output and a complete RON
  Copy Artifact using its own controller's opponent. Inventory roles are tested.
- **Fellwar Stone → OpponentRecords.** Complete City of Brass, Dwarven Ruins,
  Bottomless Vault, Sand Silos, Dwarven Hold, Hollow Trees, Icatian Store,
  Mishra's Workshop and Urza's Tower records supply replacement-aware queries;
  source costs/restrictions do not pay or carry over. These are interactions,
  not nine new complete-card coverage claims.
- **TestRONDynamicManaColorsInvalid.** Malformed syntax, arity, unknown types,
  empty/duplicate filters, unbound players, illegal costs and contexts reject
  atomically while retaining the prior registry.
- **TestManaColorsCouldProduceOptions / Recursion / RecursiveReplacements /
  PaymentAndReplacements / CounterPaymentResult / Lifecycle / StalePlan /
  DynamicAndRiders / AbsentAbility.** Exact solver options, undefined/anchored
  cycles, recursive replacement closure, all competing orders, unpayable queried
  costs, isolated counter results, live layers/control, phased/departed/returned
  subjects, clones, stale plans, dynamic amounts, riders and safe absent reads.

Compilation and gameplay remain separate. Estimated and actual gains match:
one unique name and one record. Complete-card gameplay gains Fellwar Stone.
Four collection assertions now use the measured **2,235 / 210** counts.
No Go definition, generator or DSL export was removed; removal gates remain unmet.

### Validation, files and next work

Passed in fast-fail order:
`go test ./pkg/mage/dsl/ron -run TestRONDynamicManaColors -count=1`, native
`go test ./pkg/mage -run TestManaColorsCouldProduce -count=1`, `make build`,
`go vet ./...`, `go test ./pkg/mage/... -short -count=1`, `make test`, and
skill-required `go test ./... -short -count=1`. Existing mana replacement tests
also passed with the native slice tests. Initial lint found one new test variable
shadowing Go's built-in `copy`; it was renamed. Focused RON tests, build and vet
passed again. Final `make lint` passed with **0 issues**; automatic fixes added
no unrelated changes. `git diff --check` passed. There is no remaining validation
or environment blocker.

Final `go run ./cmd/loadcards -json` exited 1 for expected atomic collection
rejection: **2,235 compiled / 210 rejected records**, **867 accepted / 119 rejected
names**. Exact baseline comparison removes only Fellwar Stone's diagnostic and
adds or changes no remaining diagnostic. All previously accepted records stay
accepted. Estimated and actual gains are one unique name/one record; the separate
complete-card gameplay gain is Fellwar Stone.

Changed engine files: new `mana_colors_could_produce.go` and its native tests;
`mana_ability.go`, `mana_type_replacement.go`, `mana_system.go`, `mana_solver.go`,
`costed_mana.go`, `game.go` and `doc.go`. Changed RON files: new
`dynamic_mana_colors.go` and its tests; `activation_timing.go`, `inventory.go`,
`definition_plan_test.go`, `inventory_test.go` and `numeric_consumers_test.go`.
API/rules docs and this ledger are updated. RON data and vocabulary hashes,
production Go definitions, generators and DSL exports remain unchanged.

### Ranked next candidates

1. **Exact exile-card counters — All Hallow's Eve.** Zero immediate gains;
   establish exact exile visits and counter storage before predicates/removal,
   spell exile with counters and simultaneous all-graveyard entry.
2. **Counter-bound duration — Cyclopean Tomb.** Zero immediate gains; complete
   each-subject visits, source counter history and delayed removal dependencies.
3. **Player-chosen trigger targets — The Abyss.** Its complete record requires
   the two-argument `TargetPermanentOfPlayersChoice(filter, Trigger_ThatPlayer)`
   shape and an event-player chooser. Assess typed binding/timing; estimated one
   unique name/one record, with existing ordinary upkeep and destroy services.

This run stops after dynamic mana color options. Runtime cutover, complete
collection gameplay, Go-card deletion and DSL cleanup remain gated.


## 2026-10-13 — EXACT-EXILE-COUNTER-READS

### Candidate comparison and selection

Fresh starting evidence: clean working tree; `make build` passed;
`go test ./pkg/mage/dsl/ron -count=1` passed; `go run ./cmd/loadcards -json`
exited 1 for expected collection rejection, with 2,235 compiled / 210 rejected
records and 867 accepted / 119 rejected unique names. This was collection
rejection, not an environment, compiler-build or test failure. The baseline
JSON was retained outside the repository for final diagnostic comparison.

The latest ledger's three candidates were checked against complete records and
nested forms. Reprints count once. No first-error frequency was used as an
unlock estimate.

| Candidate | Distinct affected names | Estimated full-card gains | Other blockers and downstream capability | Estimated effort |
| --- | --- | --- | --- | --- |
| Exact exile-card counters and reads | All Hallow's Eve (LEG, one record) | 0 names / 0 records | Existing exact exile visits are ready. Storage and predicates enable the source countdown; spell exile with counters, removal and simultaneous all-graveyard entry remain. | Small storage/query contract, two predicate adapters, about 30 lifecycle/declaration cases; fits one run. |
| Counter-bound duration | Cyclopean Tomb (LEA, LEB, 2ED, CED, CEI; five records) | 0 names / 0 records | Subject-bound counter lifetime precedes source-specific counter placement/removal history and persistent upkeep removal. A duration adapter alone cannot complete the card. | Moderate; exact subject lifetimes and layer transitions plus longer history dependencies. |
| Event-player target chooser | The Abyss (LEG, one record) | 1 name / 1 record | Ordinary bound upkeep, creature/artifact filters and destroy-without-regeneration are ready. Explicit event-player target choice needs announcement and copy contracts. | Moderate; compiler target shape plus chooser frames, legality, target placement and retarget-copy tests. |

Selected the dependency-ready first candidate as an unblocking prerequisite,
following this run's explicit first-candidate rule. It has zero immediate gains,
not an invented multi-card benefit. The exact-visit identity service already
exists and is reused. The completed slice is storage and predicate reads, not
all exile-counter operations. The other compared candidates affect one unique
card each; Cyclopean Tomb has a longer dependency chain. The Abyss retains its
credible one-card estimate for a subsequent run.

Acceptance contract stated before edits: counters belong to an exact exile
visit, both positive and zero predicates require a present face-up source,
queries preserve outer frames, detection and resolution read current counts,
return cannot replace a saved visit, and clones/instances remain independent.
Invalid declarations reject before registry publication. No RON spelling or
source-specific policy belongs in engine storage.

### Implemented shape, engine reuse and boundaries

Engine storage: `ExiledCard.Counters [core.NumCounters]int`. The exile zone owns
this array beside `Visit`; existing exile constructors initialize it to zero.
`ZoneSystem.Clone` copies it by value. New exile visits, including re-exile,
start without counters. Counts above 255 remain exact.

Native signatures:

- `ExiledCardHasCounter(card CardRef, counter core.CounterType) StateCondition`
- `ExiledCardHasNoCounters(card CardRef, counter core.CounterType) StateCondition`

Both require saved `FromCard` references and valid counter kinds. They read only
present face-up exile visits. A wrong-zone, missing, unwritten, departed,
returned or hidden source cannot supply either positive or zero information.
Counter reads do not use departure LKI. The existing `CardBinding`, captured
card visits, `cardConditionContext`, source-zone event capture,
`WithInterveningIf`, `IfState`, stack-copy and clone services own all identity,
query frames and timing. No parallel zone, trigger or result system was added.

RON signatures/context:

```text
FromExile(TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),
  ExiledCardPassesFilter(ThisExiledCard, HasACounterOfType(counter)),
  ActionList([...])));
If(ExiledCardPassesFilter(ThisExiledCard, HasNoCountersOfType(counter)),
  [actions]);
```

Either counter predicate works in both of these source-zone contexts. The
resolving branch also reuses the existing `IsCardtype(type)` predicate.
Counter kinds reuse the checked counter parser, including its supported P/T
counter syntax. Inventory distinguishes card counter kinds and retains roles
for both event conditions and nested resolving conditions. Every factory makes
fresh card/event bindings. Counter numeric comparisons, conjunctions, other
card references, alternative branches, placement/removal and spell exile with
counters remain explicit gaps. Storage alone establishes no counter-driven
characteristic modifications or replacement instructions.

CR 122.1/122.2 and 400.7/400.8 establish counter storage, loss on departure and
new exile visits. CR 603.4 supplies event/resolution guard timing. Exile cards
do not phase; battlefield phasing, payment rollback, replacement ordering and
simultaneous counter operations do not apply to pure predicates. Existing
movement/replacement services remain unchanged. No Go card definitions,
generators or DSL exports were removed; their removal gates remain unmet.

Complete All Hallow's Eve, all five Cyclopean Tomb records and The Abyss were
read. Local `data/LEG.json` Oracle text matches the unchanged All Hallow's Eve
record. No source defect was found or corrected. RON source and vocabulary
snapshot hashes remain unchanged.

### Failing-first evidence and card-to-test mapping

The first native test run failed to compile at the missing `ExiledCard.Counters`,
`ExiledCardHasCounter` and `ExiledCardHasNoCounters` declarations. After engine
implementation, the first RON run failed with unsupported source-zone card
filters for both predicates, including the real-record projection. After the
adapters, two test assumptions were corrected: a copied RON body retains its
saved upkeep player, and a sorcery projection must keep a valid SpellActions
rule. Assertions now check those engine contracts explicitly; no test was
weakened. The former rejection fixture for the now-supported predicate uses
an unknown counter kind instead, preserving its invalid-declaration purpose.

- **All Hallow's Eve → TestRONExileCounterConditionAllHallowsEve.** Its unchanged
  full record must still reject at the independent spell-exile gap. A test-only
  projection keeps the original metadata, owner-upkeep event and both actual
  counter predicates; unsupported spell exile/removal/mass return use life
  probes. The intervening condition is checked with 0, 1, 2 and 300 counters.
  A separate resolving probe uses the original event as TriggerA to test zero,
  positive, departed and returned reads without the intervening guard blocking
  its body. Pending clones read their own counters. This is prerequisite
  gameplay evidence, not complete-card gameplay or a production partial registry.
- **TestRONExileCounterConditionGameplay.** Both predicates, strict loading,
  correct upkeep owner, two fresh instances, counts above 255, unrelated counter
  types, detection/resolution changes, departure, return, hidden sources,
  stack copies and pending clone isolation.
- **TestRONExileCounterConditionInvalid / Inventory.** Arity, unknown/malformed
  counters, unbound/wrong references, unsupported conjunctions/comparisons,
  wrong zones, ordinary spell misuse, empty bodies and atomic rejection; nested
  card/filter/counter roles remain unverified in inventory reporting.
- **TestExileCounterConditionVisits / Timing / Invalid / MissingFacts.** Exact
  visits, fresh returns and re-exile, wide counts, clone arrays, separate counter
  types, source-owner upkeep, current guards, pure query frames, pending clones,
  copied controllers, two instances, hidden/unwritten/wrong-zone reads, valid
  native resolving declarations and definition-time errors.

Estimated and actual full-card gains both equal zero names / zero records.
Complete-card gameplay coverage remains unchanged. All Hallow's Eve's two
counter predicates now have RON-backed behavior evidence; its remaining rules
retain separate gaps.

### Validation, changed files and next work

Passed in fast-fail order:

1. `go test ./pkg/mage/dsl/ron -run TestRONExileCounterCondition -count=1`;
   focused existing `TestRONSourceZoneCondition` regressions and native
   `go test ./pkg/mage -run TestExileCounterCondition -count=1` also passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1` and `make test` passed. The
   skill-required `go test ./... -short -count=1` also passed.
4. `make lint` passed with **0 issues**. Automatic formatter/fix results were
   reviewed; no unrelated or semantic changes were introduced.
   `git diff --check` passed.
5. Final `go run ./cmd/loadcards -json` exited 1 for expected atomic collection
   rejection: **2,235 compiled / 210 rejected records**, **867 accepted / 119
   rejected names**. Exact baseline/final JSON equality was asserted. All
   accepted records and all remaining diagnostics are unchanged, including
   `LEG.lines.ron:12:190: unsupported exile subject (Spell)` for All Hallow's Eve.

No required validation check failed and no environment blocker remains.
Expected collection rejection does not block this validated prerequisite slice.
Estimated and actual compilation gains match at zero names / zero records;
complete-card gameplay gains remain zero. One descriptive slice commit is made
after these checks; no next slice starts in this run.

Changed engine files: new `exile_counter_condition.go` and native tests;
`game.go`, `zone_system.go` and `doc.go`. Changed compiler files: new
`exile_counter_condition.go` and RON tests; `compiler.go`, `inventory.go`,
`source_zone_condition.go` and its rejection test. API/rules contracts and this
ledger document the established behavior. No card source, vocabulary snapshot,
production definition, generator or obsolete DSL caller was changed.

### Ranked next candidates

1. **Exact exile-source counter removal — All Hallow's Eve.** Storage and both
   predicates are ready. Lower `RemoveCounters([ACounterOfTypeFromCardInExile
   (ScreamCounter,ThisExiledCard)])` through an exact saved-visit engine operation;
   cover current counters, absent/returned subjects and removal events. Estimated
   zero full-card gains until spell exile and mass entry are implemented.
2. **Spell exile with initial counters — All Hallow's Eve.** Support its singular
   ThisSpell exile and fixed counter modifier before source-zone triggers can
   observe it. Establish stack identity, copies, departure and initialization
   timing. Estimated zero gains while mass graveyard return remains unsupported.
3. **Event-player target chooser — The Abyss.** Existing bound upkeep and destroy
   mechanics are ready. Implement its explicit target chooser with current
   legality and copy-retarget contracts; estimated one name / one record.

This run stops after exact exile-counter reads. Complete collection gameplay,
runtime loading, Go-card deletion, DSL cleanup and native cutover remain gated.

## 2026-10-13 — EXACT-EXILE-SOURCE-COUNTER-REMOVAL

### Candidate comparison and selection

The initial worktree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The baseline loader exited 1 for
atomic collection rejection, not a build or test failure: **2,235 compiled /
210 rejected records**, **867 accepted / 119 rejected names**, from 2,445
records and 986 names. Complete All Hallow's Eve, Recall and The Abyss records
were read, as were all five Cyclopean Tomb records for the distinct permanent
counter-history boundary. Targeted vocabulary entries show one occurrence each
of `ACounterOfTypeFromCardInExile`, `ExilesWithNumberCounters` and the two-argument
`TargetPermanentOfPlayersChoice`. Reprints count once.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream benefit | Implementation/test effort |
| --- | --- | --- | --- | --- |
| Exact exile-source counter removal, ranked #1 | All Hallow's Eve | 0 names / 0 records | Storage, source visits and both predicates are ready. Removes the resolving removal gap so the counter countdown and zero branch can execute; spell exile and mass graveyard entry still block the full card | Small: one typed resolving engine operation, one adapter and visit/timing/isolation/declaration tests |
| Spell exile with initial counters, ranked #2 | All Hallow's Eve; Recall shares `Spell(ThisSpell)` exile without a modifier | 0 names / 0 records | Needs exact stack departure and copy lifecycle, plus counter initialization before exile-source observations. All Hallow's Eve still needs mass entry; Recall still needs X discard and saved discard-count return. The shared spell departure contract serves two unique names | Medium: shared stack/zone movement, copied spells, result identity and initialization timing |
| Event-player target chooser, ranked #3 | The Abyss | 1 name / 1 record | Bound upkeep player and destruction exist. Needs a declared event-player target chooser and tests of announcement legality, current control, protection, no legal targets and copy retargeting; enables later explicit event choosers | Medium: target declaration/choice ownership integration and gameplay contracts |

Ranked candidate #1 is dependency-ready and an unblocking prerequisite, so it
was selected under this run's selection rule. It is a bounded complete resolving
operation. None of these candidates supplies an immediate multi-name full-card
unlock; the shared spell departure contract remains the concrete two-name next
opportunity. No first-error count was used as an unlock estimate. The estimated
zero-name / zero-record gain matches the actual gain; The Abyss's possible gain
is deferred, not claimed. No second slice was started.

### Signatures, contexts, dependencies and engine boundary

Supported RON shape:

```text
RemoveCounters([ACounterOfTypeFromCardInExile(counter, ThisExiledCard)])
```

It accepts exactly one instruction in resolving ActionList bodies under an
ordinary owner-upkeep `FromExile(TriggerA(...))` or `FromExile(TriggerI(...))`,
including nested resolving `If` bodies. It reuses the checked counter parser,
`compileCardRef`, typed source-zone plans and fresh per-factory card bindings.
The semantic inventory now identifies its counter as `card_counter_type` and
its subject as `card_reference`. Inventory support stays unverified; inventory
classification does not claim gameplay evidence. Other references, zones,
amounts, lists, costs, entry and replacement contexts remain rejected.

New reusable API:
`RemoveCounterFromExiledCard(card CardRef, counter core.CounterType) ResolvingEffect`.
The engine owns the exact saved-visit mutation. It reuses `FromCard`, the shared
card resolver, exile visit storage, source-zone trigger frames, copied stack
frames and `ZoneSystem.Clone`; no parallel zone or event system was added.
Validation rejects non-saved references, nil/unwritten bindings and invalid
counter kinds before gameplay. One available counter is removed at resolution;
zero stays zero, counts above 255 stay integer counts, and other kinds stay
unchanged. Current counts, not detection-time counts, supply the removal.
Missing, wrong-zone, departed, returned or face-down subjects perform no removal,
even when the controller can inspect the hidden card. Later instructions run.
Subsequent zero predicates see the updated count. Copies retain the saved visit;
clones and factory instances keep separate arrays and bindings.

CR 122.1/122.2, 400.7/400.8, 603.4 and 608.2c establish counter state, new visits,
intervening-if timing and instruction order. Exile cards do not phase. This
singular resolving instruction makes no choices and pays no costs; cost rollback,
plural simultaneous removal and placement replacement ordering do not apply.
Existing permanent removal has no dedicated counter-removal event. This slice
adds no such trigger contract; future event syntax needs a separate service.
Zone departure clears counters without reporting a removal (CR 122.2).

Local `data/LEG.json` Oracle text matches All Hallow's Eve's unchanged record.
No source defect or environment failure was found. RON files and vocabulary
snapshot hashes are unchanged. No Go definitions, generator, DSL export or
caller was removed; runtime, definition-removal and DSL-removal gates remain
unmet.

### Failing-first evidence and card-to-test mappings

The initial native run failed to compile at the missing
`RemoveCounterFromExiledCard` signature. After implementing the engine operation,
the native tests passed. The first RON run then failed with
`unsupported counter removal instruction (ACounterOfTypeFromCardInExile)` in
synthetic declarations and the real-record projection. After lowering, the copy
probe's life expectation was corrected: unscoped GainLife follows the copied
ability's controller. Counter behavior did not require a change; the test now
asserts both controllers and the shared saved-visit removal.

- **All Hallow's Eve → TestRONExileCounterRemovalAllHallowsEve.** The unchanged
  full record still rejects at its independent spell-exile gap. The test-only
  projection retains metadata, original upkeep event, intervening counter
  predicate, actual removal, original zero-counter branch and owner-graveyard
  movement. Only unsupported spell exile and mass entry use life probes. Two
  upkeeps remove two counters separately; the second observes zero and moves the
  source to its owner's graveyard. Later upkeeps cannot arm the departed source.
  This is prerequisite behavior evidence, not a partial production registry or
  complete-card gameplay coverage.
- **TestRONExileCounterRemovalGameplay / Nested.** Strict RON loading; owner
  upkeep; zero, one and 300 counters; current count changes before resolution;
  unrelated types; departure/return; hidden sources at detection and after
  queuing, including private visibility; copied bodies and copied controllers;
  pending clones; two factory instances; nested removal; resolving branch order;
  normal graveyard movement and later independent instructions.
- **TestRONExileCounterRemovalInvalid / Inventory.** Empty/plural lists, arity,
  unknown and malformed counter kinds, wrong and unbound subjects, wrong zone,
  spell/cost misuse, atomic invalid collection rejection and precise inventory
  roles without claiming behavior coverage.
- **TestExileCounterRemovalVisits / Definition.** Native exact visits,
  wrong-zone and unwritten reads, current wide counts, separate kinds, hidden
  subjects, fresh returns, independent clone execution and definition-time
  rejection for nil, unwritten, incompatible and invalid declarations.

Compilation coverage and complete-card gameplay coverage do not increase.
The resolving exile-source removal contract and its All Hallow's Eve countdown
prerequisite gain verified behavior coverage.

### Validation, changed files and next work

Passed in fast-fail order:

1. `go test ./pkg/mage/dsl/ron -run TestRONExileCounterRemoval -count=1`;
   native `go test ./pkg/mage -run TestExileCounterRemoval -count=1` and existing
   exile-counter, source-zone, single-removal and counter-state RON regressions
   also passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1`, `make test` and the skill-required
   `go test ./... -short -count=1` passed.
4. The first `make lint` run reported and automatically fixed one redundant
   test type declaration. The final declaration was reviewed and the native
   focused tests rerun successfully. The second `make lint` passed with
   **0 issues**; no unrelated automatic changes occurred. `git diff --check`
   passed after the final ledger update.
5. Final `go run ./cmd/loadcards -json` exited 1 for expected atomic collection
   rejection: **2,235 compiled / 210 rejected records**, **867 accepted / 119
   rejected names**. Baseline and final JSON equality was asserted, including
   every accepted count and remaining diagnostic. All Hallow's Eve still reports
   `LEG.lines.ron:12:190: unsupported exile subject (Spell)`.

No required check remains failed and no environmental blocker remains. Expected strict
collection rejection does not block this validated slice. The completed slice
is committed after the required checks; no next slice starts in this run.

Changed engine files: new `exile_counter_removal.go` and native tests; `doc.go`.
Changed compiler files: new `exile_counter_removal.go` and RON tests;
`counter_state.go` and `inventory.go`. API/rules-contract documentation and this
ledger record the established behavior. No RON source or vocabulary changed.

### Ranked next candidates

1. **Spell exile with initial counters — All Hallow's Eve; shared departure
   contract for Recall.** Resolve one exact `Spell(ThisSpell)` through shared
   stack/zone services, initialize fixed counters before exile-source observations
   and cover copied spells. Estimated zero immediate full-card gains: All
   Hallow's Eve retains mass entry and Recall retains discard/result-return gaps.
2. **Simultaneous mass graveyard entry — All Hallow's Eve.** Lower
   `PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),
   AnyPlayer,[EntersNormally])` through one shared entry batch, preserving owners,
   entry replacements and simultaneous observations. Estimated one name / one
   record only after spell exile is ready; otherwise zero.
3. **Event-player target chooser — The Abyss.** Lower the explicit saved upkeep
   player as target chooser with current legality and copy-retarget contracts.
   Estimated one name / one record.

This run stops after exact exile-source counter removal. Complete collection
gameplay, runtime loading, Go-card deletion, DSL cleanup and native cutover
remain gated.

## 2026-10-13 — SPELL-EXILE-WITH-INITIAL-COUNTERS

### Selection, baseline and dependency readiness

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Baseline
`go run ./cmd/loadcards -json` exited 1 for expected atomic collection rejection:
**2,235 compiled / 210 rejected records**, **867 accepted / 119 rejected names**,
from 2,445 records / 986 names. This was collection rejection, not a build or
test failure.

Complete records and their nested forms were inspected for all three current
ranked candidates. Reprints count once:

| Candidate | Distinct affected names | Estimated immediate full-card unlocks | Other blockers and downstream value | Full implementation/test effort |
| --- | --- | --- | --- | --- |
| Spell exile with initial counters | All Hallow's Eve; Recall | 0 names / 0 records | All Hallow's Eve retains mass graveyard entry; Recall retains variable discard and discard-count return. Establishes shared exact spell departure and makes mass entry the last All Hallow's Eve gap. | Moderate: typed resolving reference, departure lifecycle, initial counters, copied spell cleanup, cloning, compiler adapter and tests. |
| Simultaneous mass graveyard entry | All Hallow's Eve | 0 before spell exile; 1 name / 1 record after it | Spell exile is required first. Enables the final resolving branch through shared entry replacements and simultaneous observations. | Higher: entry preparation and owner choices across both graveyards, APNAP and batch observations. |
| Event-player target chooser | The Abyss | 1 name / 1 record | Requires the explicit two-argument target shape, saved upkeep-player chooser and copy-retarget contracts. Its nested nonartifact creature and event-player controller filters need preserved legality. | Moderate: announcement chooser, saved event context, target queries and copies. |

Candidate #1 was selected as the dependency-ready shared prerequisite for two
unique cards. Counter state, source-zone upkeep, intervening predicates and
singular removal already exist. Exact self-spell departure and copied exile
lifecycle were missing engine contracts; AST spelling and scope were missing
adapters. Neither source-data defects nor environment failures selected the
work. The contract and acceptance cases were stated before editing.

### Established signatures, contexts and reuse boundary

New reusable engine APIs:

- `ThisSpell StackObjectRef` reads the current resolving spell's exact stack ID.
- `ExileSpell(spell StackObjectRef) ResolvingEffect` moves one present spell.
- `ExileSpellWithCounters(spell StackObjectRef, counter core.CounterType,
  count int) ResolvingEffect` sets a fixed nonnegative count on the new exile
  visit before zone observations.

RON accepts `Exile([Spell(ThisSpell)], [])` and exactly one
`ExilesWithNumberCounters(Integer(n), counter)` modifier, with `n >= 0`, in a
resolving instant or sorcery body. Counter kinds reuse the checked permanent
counter parser, including supported literal P/T pairs. Wrong references,
unbound reads, plural subjects, malformed arity, unknown counters, dynamic or
negative amounts, face-down modifiers, costs, zone-source rules, delayed and
replacement bodies remain rejected before gameplay. Inventory roles distinguish
spell subjects/references and exile counter modifiers without claiming behavior
coverage.

The engine reuses `StackObjectRef`, resolving-reference frames, exact saved
spell readers, stack departure information, zone exile constructors, turn
tracking, zone-change events, state-based actions and existing clone services.
RON contains no zone or counter mutation. No parallel event system or global
last-action slot was added. The new departure flag stops repeated instructions,
ordinary cleanup and permanent entry from moving the departed spell again.
Later independent instructions continue. Returning the card or casting it again
cannot supply the old stack visit. The card keeps its owner; the event keeps its
former controller. Initial counters are integers, including counts above 255.

Copies use their own card and exile visit. They remain observable through the
remaining instructions, then cease at the next state-based check without a
second zone-change event. This slice does not add transport of that copy from
exile to another zone before the check. Stack clones retain the departure flag,
including clones made during resolution; zone clones keep separate counter
arrays. Phasing does not apply to spells or exile cards. This singular resolving
operation makes no choices or payment, so cost rollback and plural simultaneous
selection are outside this context. Permanent entry counter replacement rules
do not apply to exile cards; no general exile-card counter placement adapter was
added.

CR 112.1/112.1a, 400.7, 608.2b/608.2c/608.2m/608.2n, 701.13a, 704.5e and 122.2
support the established spell, identity, timing, exile, copy and counter rules.
Local `data/LEG.json` Oracle text matches both unchanged affected RON records.
No source defect was found. RON source and vocabulary snapshot hashes are
unchanged. No Go definitions, obsolete generator or broad DSL export was
removed; their removal gates remain unmet.

### Failing-first evidence and card-to-test mappings

Native tests first failed to compile at the missing engine APIs. After engine
implementation they passed. RON tests then failed at `unsupported exile subject
(Spell)`, including strict loading and both real-record projections. Lowering
removed that gap. Final review added a failing assertion for a clone made after
spell exile during resolution; the explicit stack clone initializer omitted the
new departure flag. Copying the flag repaired the test. Test harness corrections
supplied mana for a second cast and used an instant for two simultaneous source
instances; no engine behavior or assertion was weakened.

- **All Hallow's Eve → `TestRONSpellExileAllHallowsEve`.** Retains metadata,
  original cast cost, original spell exile with two counters, original owner
  upkeep and intervening-if, counter removal, zero-counter branch and owner
  graveyard move. Only mass creature entry uses a life probe. Casting initializes
  two counters; opponent upkeep does nothing; two owner upkeeps count down and
  move the source to its graveyard. Departure prevents later arming. The full
  unchanged record rejects only at mass graveyard entry. This is prerequisite
  evidence, not complete-card gameplay or a partial production registry.
- **Recall → `TestRONSpellExileRecall`.** Retains metadata, double-X cost and
  original unmodified self-exile instruction. Omits unsupported variable discard
  and result-count return in the test projection. A paid X=2 cast spends five
  mana and resolves to exile without counters or duplicate graveyard cleanup.
  The full record retains its independent variable-discard diagnostic.
- **`TestRONSpellExileGameplay`.** Strict RON registry loading, zero/two/300
  counters, announcement versus resolution, repeated exile, other controllers,
  separate factories, copied spells and pending clones.
- **`TestRONSpellExileInvalid`, `FizzleAndAtomicRejection`, `Inventory`.** Invalid
  declarations and contexts, all-illegal-target fizzle, delayed-body rejection,
  atomic invalid collection loading and precise inventory roles.
- **`TestSpellExileResolution`, `Definition`.** Native later instructions,
  counter initialization, zero/wide counts, copied controllers and copy cleanup,
  owner destinations, pending clones, return before repeat exile, existing
  exile-on-leave behavior, nil/unwritten/invalid declarations and absent self
  context.
- **`TestSpellExileSavedVisitAndObservation`.** Existing saved spell references,
  counter state before zone-event detection, former controller, exact departure,
  queued observation body and rejection of a later cast of the same card.
- **`TestSpellExileResolvingClone`.** Clone of an active departed spell frame
  cannot restart exile or reset counters; counter mutations remain separate.

Actual full-card gains match the estimate: **zero names / zero records**.
Compilation coverage and complete-card gameplay coverage do not increase.
The cast-to-exile countdown path and Recall's self-exile prerequisite gain
verified behavior coverage.

### Validation and files

Passed in fast-fail order after the final clone repair:

1. `go test ./pkg/mage/dsl/ron -run TestRONSpellExile -count=1` and native
   `go test ./pkg/mage -run TestSpellExile -count=1`. Existing counter-condition
   and counter-removal probes also passed. Their full-record rejection assertions
   now require the independent mass-entry diagnostic. The first engine short
   run exposed one stale old spell-exile diagnostic assertion; it was corrected
   to that specific remaining error before repeating the suites.
2. `make build` and `go vet ./...` passed again after the clone repair.
3. `go test ./pkg/mage/... -short -count=1`, `make test`, and skill-required
   `go test ./... -short -count=1` passed again after the clone repair.
4. Initial lint reported two style issues: combine equal parameter types and
   remove an overwritten test variable initializer. They were fixed manually.
   Final `make lint` passed with **0 issues**. Automatic formatting changes were
   reviewed; no unrelated changes occurred. Final `git diff --check` passed.
5. Final `go run ./cmd/loadcards -json` exited 1 for expected atomic rejection:
   **2,235 compiled / 210 rejected records**, **867 accepted / 119 rejected
   names**. All rejected records except All Hallow's Eve retain their baseline
   diagnostics. All Hallow's Eve now reports
   `LEG.lines.ron:12:634: unsupported action (PutEachCardFromEachPlayersGraveyardOntoTheBattlefield)`.
   Accepted counts remain unchanged. This collection failure is required while
   unsupported records remain and does not block a validated slice commit.

Changed engine files: new `spell_exile.go` and native tests; `stack.go`,
`stack_resolver.go`, `state_based_actions.go`, `game.go`, `clone.go`,
`composable_actions.go` and `doc.go`. Changed compiler files: new
`spell_exile.go` and RON tests; `permanent_exile.go`, `inventory.go`, and the two
previous exile-counter test files' gap assertions. API/rules-contract docs and
this ledger record only established behavior. No required failure or environment
blocker remains. Only this completed slice is committed; no next slice starts.

### Ranked next candidates

1. **Simultaneous mass graveyard entry — All Hallow's Eve.** Spell exile is now
   ready. Lower the complete creature/AnyPlayer/EntersNormally shape through a
   shared entry batch with owners, replacement order and simultaneous
   observations. Estimated one name / one record; verify complete gameplay.
2. **Event-player target chooser — The Abyss.** The explicit saved upkeep player
   must choose its target using current legality and copy-retarget contracts.
   Estimated one name / one record.
3. **Variable discard and saved discard-count return — Recall.** Inspect the
   discard producer and result-return consumer together; spell exile is ready.
   The discard prerequisite alone gives zero unlocks, while completing its
   dependent return can give one name / one record. Reassess shared value against
   fresh complete-record inventory before selecting it.

Complete collection gameplay, runtime loading, Go-card deletion, obsolete
Go-set generation removal and unused DSL cleanup remain gated.


## 2026-10-13 — SIMULTANEOUS-GRAVEYARD-CREATURE-ENTRY

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and uncached RON package tests passed.
Baseline loader exit 1 was expected atomic collection rejection, not a build or
test failure: 2,235 compiled / 210 rejected records and 867 accepted / 119 rejected
names. The complete LEG records for All Hallow's Eve, The Abyss and Recall were
read, including every nested condition, target restriction and result consumer.
Targeted vocabulary reads confirmed one mass-entry occurrence, one Recall
saved-discard return, and the broader fixed/variable discard family. Reprints
count once.

| Candidate | Distinct affected names | Estimated full-card gains | Other blockers and downstream capability | Effort |
| --- | --- | --- | --- | --- |
| Ranked #1: simultaneous graveyard creature entry | All Hallow's Eve | 1 name / 1 record | Spell exile, initial counters, owner upkeep, source conditions and removal are ready. Shared entry preparation exists; saved graveyard batch selection and deferred observations are missing. Establishes an owner-entry batch boundary for later collection adapters. | Small/medium: one native effect, one lowering adapter, batch and identity contracts |
| Ranked #2: event-player target chooser | The Abyss | 1 name / 1 record | Upkeep event binding exists; its saved player must choose a nonartifact creature they control. Needs chooser-aware target declaration, announcement legality and copy retargeting integration. | Medium: target contract across announcement and copies |
| Ranked #3: variable discard and saved count return | Recall | 0 for discard alone; 1 name / 1 record for the full procedure | Self-exile is ready. Variable discard must produce an exact local count for the separate graveyard-return consumer. Fixed discard already serves Bazaar of Baghdad and Mindstab Thrull; those are not new unlocks. | Large relative to this run: two dependent producer/consumer contracts |

Ranked #1 is dependency-ready. Comparison found no immediate multi-name unlock
among these candidates. It gives the same full-card gain as The Abyss with the
smallest credible shared-service effort. Recall requires two distinct contracts.
Only the mass-entry slice was implemented. Estimated and actual gains match:
**one unique name / one record**. Complete-card gameplay gains one mapping.
Compilation and gameplay coverage remain separate measurements.

### Exact supported shape and engine contract

RON accepts
`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature), AnyPlayer, [EntersNormally])`
in resolving spell, activated, triggered and delayed bodies, including `If` and
source-zone upkeep bodies. Tuple arity, symbol kinds and the single entry option
are exact. Other filters, player sets, options, costs, static rules, entry
operations, direct nonresolving scopes and draw-replacement bodies reject during
compilation. `compileGraveyardBatchEntry(Node, scope)` keeps spelling and scope
validation in `ron`. Inventory operands have graveyard-filter, player-reference
and entry-operation roles.

The reusable API is
`mage.PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl() mage.ResolvingEffect`.
It captures typed current graveyard visits once before choices. Each owner makes
entry choices in APNAP order and can order their own cards through the existing
sanitized card-order service. New cards and returned visits cannot join the
saved selection. Preparation uses independent copied definitions with physical
IDs, zero announced entry X and zero cast colors. The enclosing X is restored.

The engine reuses `prepareBattlefieldEntryResultWithContext`, copy and counter
replacements, tapped entry, required entry operations, land-entry prohibitions,
registration and entry completion. The existing exact Aura-host choice helper
was renamed `chooseEntryAuraHost` and is shared with linked exile entry; no old
caller was removed. Saved visits, owners, hosts and prohibitions are checked
again after choices. Blocked or failed entries remain in their graveyards with
their current visits. All successful removals and registrations finish before
observations. Continuous layers apply before notifications. Each owner receives
one `EvtCardsLeftGraveyard` with the actual count; entry zone events use graveyard
origin and the resolving controller as actor. Entrants observe other entrants.

No mutable declaration, card-name dispatch, Go-card fallback, partial production
registry, global last-action slot or game-time declaration panic was added.
The effect has no source read or announcement target and adds no payment form.
Clones use separate zones and pending execution. Graveyard cards do not phase;
queued activations still resolve after source departure. Aura host phasing and
incarnation checks reuse the established attachment service. Cost rollback and
payment replacement order remain with existing entry/payment services; this
slice adds no cost declaration. Broader filters and explicit entry options
remain separate adapters.

CR 101.4/101.4c, 107.3g, 400.7, 603.6a, 614.12 and 616.1 support the established
choice, identity, entry and observation behavior. Local `data/LEG.json` Oracle
text matches the complete unchanged All Hallow's Eve record. No source defect
was found. RON sources and vocabulary snapshot hashes are unchanged. Runtime
cutover, Go-card deletion, obsolete generation and unused DSL cleanup gates
remain unmet; no such removal was attempted.

### Failing-first evidence and card-to-test mappings

Native tests first failed to compile at the missing engine API. The complete
All Hallow's Eve spell test then failed at the precise unsupported mass-entry
action. Both passed after the native service and lowering were added. Test
fixture corrections used battlefield membership instead of `FindPermanent`
(which also exposes a preparing entrant), and an explicit player that accepts
optional copy choices. Assertions were retained.

A later failing native assertion proved graveyard-departure observers initially
missed entrants' static P/T layers. Applying the shared layers after registration
and before notifications repaired the ordering. A failing scope assertion proved
the initial adapter accepted nonresolving/replacement/entry scopes; the explicit
scope guard repaired that boundary. The first engine short run found stale
collection counts and the former All Hallow's Eve rejection assertion. Those
were updated to the new exact counts and complete-record acceptance.

- **All Hallow's Eve → `TestRONGraveyardBatchEntryAllHallowsEve`.** Strict loading
  of the complete original record; paid casting, exile with two counters,
  opponent-upkeep exclusion, two owner-upkeep countdowns, source graveyard move
  before batch observations, both owners' creature returns, fresh entry counters,
  pending upkeep clone isolation and no new trigger after departure.
- **All Hallow's Eve → `TestRONSpellExileAllHallowsEve`.** The former life-probe
  projection now uses the complete record and covers the empty-graveyard path.
  Existing `TestRONExileCounterConditionAllHallowsEve`,
  `TestRONExileCounterRemovalAllHallowsEve` and
  `TestRONSourceZoneUpkeepRecordPrerequisites` retain their focused probes and
  require complete-record acceptance.
- **`TestRONGraveyardBatchEntryGameplay`.** Strict synthetic RON loading,
  announcement/resolution timing, spell copies, activated source departure,
  delayed bodies, owner control and pending clone isolation.
- **`TestRONGraveyardBatchEntryInvalid`, `InvalidScope`.** Invalid arities,
  symbol/tuple substitutions, filter/player/entry variants, cost/static/entry
  contexts and direct nonresolving/replacement scopes reject.
- **`TestGraveyardBatchEntry`.** Native definition validation, owner control,
  tapped and counter replacements, noncreature exclusion, exact batch
  observations, repeat execution and zone clone isolation.
- **`TestGraveyardBatchEntryChoicesAndVisits`, `APNAPAndFinalRecheck`.** Choices
  precede movement; both active-player orders, visits returned before or after
  preparation, late additions, entry-copy land prohibition, failed entry,
  retained graveyard visits, zero entry X and restored enclosing X.
- **`TestGraveyardBatchEntryEntrantObservations`.** All three entrants observe
  each other; one owner departure notification reports three cards; static
  entrant layers are present at that notification.

### Validation and files

Final checks passed after the layer-order repair and scope guard:

1. `go test ./pkg/mage/dsl/ron -run TestRONGraveyardBatchEntry -count=1` and
   `go test ./pkg/mage -run TestGraveyardBatchEntry -count=1`. The earlier source
   exile/counter All Hallow's Eve probes and linked-exile-entry regression tests
   also passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1`, `make test`, and skill-required
   `go test ./... -short -count=1` passed.
4. Final `make lint` passed with **0 issues**. Automatic import formatting in
   the new files was reviewed; no unrelated change occurred. Final
   `git diff --check` passed.
5. Final `go run ./cmd/loadcards -json` exited 1 for required atomic collection
   rejection: **2,236 compiled / 209 rejected records**, **868 accepted / 118
   rejected names**. All Hallow's Eve's prior
   `LEG.lines.ron:12:634: unsupported action (PutEachCardFromEachPlayersGraveyardOntoTheBattlefield)`
   diagnostic disappears. A keyed comparison with the fresh baseline confirms
   every remaining rejected record retains its exact diagnostic. No accepted
   record regressed. Expected collection rejection does not block this slice's
   commit. No build, test, lint or environment blocker remains.

Changed native files: new `graveyard_batch_entry.go` and native tests,
`linked_exile_entry.go` helper reuse and `doc.go`. Changed compiler files: new
mass-entry adapter and RON tests, compiler dispatch and inventory roles.
Earlier spell/source-counter probes now require complete All Hallow's Eve
acceptance; definition-plan, inventory and numeric-consumer count assertions
use the fresh collection totals. API/rules-contract docs and this ledger cover
only established behavior. No RON source or vocabulary file changed.

### Ranked next candidates

1. **Event-player target chooser — The Abyss.** Save the explicit upkeep-player
   chooser through announcement and copy retargeting. Preserve its current
   nonartifact-creature/control legality and destroy-without-regeneration action.
   Estimated one name / one record; reassess broader shared chooser uses first.
2. **Variable discard and saved discard-count return — Recall.** Its self-exile
   prerequisite is ready. Discard alone gives zero new names; the dependent
   graveyard-return consumer can finish one name / one record. Do not count
   already accepted fixed-discard cards as new unlocks. Reassess multi-card
   prerequisites before choosing this two-contract procedure.

Complete collection gameplay, runtime loading, Go-card deletion, obsolete
Go-set generation removal and unused DSL cleanup remain gated. This run stops
after one completed slice.

## 2026-10-13 — EVENT-PLAYER-TARGET-CHOOSER

### Candidate comparison and selection

The worktree was clean. Fresh `go test ./pkg/mage/dsl/ron -count=1` passed;
fresh loader evidence was 2,236 compiled / 209 rejected records, 868 accepted /
118 rejected names. The loader exited 1 for atomic collection rejection, not a
build failure. Complete The Abyss, Recall and Transmutation records were read
from LEG; the complete Farrel's Mantle record was also read for downstream
comparison. Targeted vocabulary queries confirmed one occurrence each of
`TargetPermanentOfPlayersChoice`, `NumCardsDiscardedThisWay` and `SwitchPT`.
No Go card definition was used as a requirement. Reprints count once.

| Candidate | Distinct affected cards | Estimated full-card unlocks | Other blockers and downstream use | Effort |
| --- | --- | --- | --- | --- |
| Event-player target chooser (ranked #1) | The Abyss | 1 name / 1 record | Upkeep binding, characteristics and no-regeneration destruction are ready. Add shared saved-player choice and control filter; reusable for other bound step-target rules | Small: typed option, placement query, lowering and identity/choice tests |
| Variable discard and discard-count return (ranked #2) | Recall | 0 for discard alone; 1 name / 1 record after both contracts | Self-exile is ready. Needs actual discard count and dependent graveyard return, with saved visits and separate result scope. Enables later saved-discard consumers | Medium/large: two distinct operation contracts and result lifetime |
| Temporary layer switch | Transmutation | 1 name / 1 record | Existing target and duration lower. Needs switch-layer lowering and native contract evidence; reusable P/T switch layer | Small: layer adapter and order/lifetime tests; no multi-card occurrence in this collection |

Ranked #1 is dependency-ready and gives the same credible immediate gain as
Transmutation, while adding an explicit shared chooser contract. The records
show no dependency-ready multi-card gain among these bounded candidates.
Recall's two contracts give a smaller gain per effort. Selection, boundary and
acceptance cases were stated before edits. Only this slice was implemented.
Estimated and actual gains match: one unique name and one record.

### Exact signatures, contexts and engine reuse

- Native `TargetChosenBy(player PlayerRef) PermanentTargetOption` accepts one
  saved event player. Definition validation rejects nil, unbound or incompatible
  refs, duplicate explicit choosers and simultaneous explicit/opponent choosers.
  The declaration holds no game-time selected state.
- RON `TargetPermanentOfPlayersChoice(filter, Trigger_ThatPlayer)` accepts one
  singular or numbered permanent target group in a bound step-trigger body.
  `ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))` in its target filter
  resolves the same step binding. The existing direct player argument parsing
  also applies to this filter. Characteristic filters and nested And compose.
  Other event chooser contexts, other chooser refs, tuple player substitutions,
  wrong arities and unbound/non-step reads reject.
- Factory-local `instanceBindings.steps` builds the chooser and control reads.
  The adapter remains in ron; reusable execution remains in mage.
- `chooseTriggerTargets` uses `capturedTriggerTargetPlayers` and
  `targetPlayerQuery` to resolve the chooser. It does not replace the live
  resolution frame or run the ability body. Missing players cannot choose.
  Target candidate selection and validation still use the detected ability
  controller and source, including protection, hexproof, shroud and phasing.
- Shared target identity, resolution legality, regeneration/destruction,
  event capture, clone and stack-copy services are reused. Copies keep captured
  event players and original target identities. CR 707.10c gives their controller
  the new-target choice; the event player still restricts controlled subjects.
  Existing spell-text declaration comparison now includes the chooser count.

CR 115.1d, 603.3d, 400.7, 608.2b, 702.26 and 707.10c support target placement,
missing legal choices, exact visits, resolution legality, phasing and copy
choices. Local `data/LEG.json` Oracle text matches the complete unchanged
The Abyss record. No source defect was found. RON sources and vocabulary hashes
are unchanged; no vocabulary refresh was required.

No payment, simultaneous operation, replacement or LKI subsystem was added.
Destruction without regeneration uses the existing service and still respects
indestructibility. Trigger source LKI uses the existing target-source snapshot.
No mutable declarations, global last-action slot, card-name dispatch, fallback,
partial production registry or game-time declaration panic was added. Current
Go callers and definitions remain. Runtime cutover and all removal gates remain
unmet, so no DSL function or Go definition was deleted.

### Failing-first evidence and card-to-test mapping

Native tests first failed to compile at missing `TargetChosenBy`. The completed
RON gameplay test failed at The Abyss's exact missing two-argument target
adapter. Initial RON fixture construction used unavailable artifact/protection
helpers; those were corrected before the expected adapter failure was captured.
A failing native invalid-declaration assertion exposed nil chooser acceptance;
explicit declaration counts repaired nil and conflicting chooser checks.
The copy fixture initially assumed `ResolveStack` resolves one object; it was
corrected to assert both objects after the shared full-stack operation, retaining
the original destruction and clone assertions.

- **The Abyss → `TestRONEventTargetChooserAbyss`.** Complete original record;
  failed and paid casting, both upkeep players, choice timing, nonartifact and
  current control filters, protection, controller-relative hexproof, invalid
  chooser response, no legal target, source control/departure/return/phasing,
  phased-source detection exclusion, target control/return/shroud/phasing and
  phase cycle, indestructibility, regeneration exclusion, copy retargeting and
  clone isolation.
- **The Abyss → `TestRONEventTargetChooserBindingsAndStrictLoad`.** Exact source
  record loaded through strict registry APIs; two pending event players keep
  separate choices, clones keep separate zones, and invalid chooser collections
  fail atomically without replacing the registry.
- **The Abyss → `TestRONTextUpkeepTriggerInventory` and
  `TestRONTextUpkeepLayerInventory`.** Existing complete-record
  definition rebuild/text-revision inventory now includes The Abyss, increasing
  this declaration inventory from 61/123 to 62 names/124 records.
- **`TestRONEventTargetChooserInvalid`.** Malformed, incompatible and unbound
  chooser declarations reject in both spell and bound upkeep bodies.
- **`TestRONEventTargetChooserNumberedInstances`.** End-step event, two numbered
  groups, separate factory instances, four explicit choices and cloned execution.
- **`TestTargetEventChooser`, `Invalid`, `Conflict`.** Native saved-player
  placement, preserved enclosing query binding, copy-controller retargeting,
  original/copy target independence and declaration validation.

### Validation and files

The first and second engine short checks exposed stale upkeep trigger/layer
inventory expected counts after this newly accepted record. Exact collection and declaration counts
were updated; no assertions or scenarios were removed.

Final checks passed after the exact inventory count and lint-name corrections:

1. `go test ./pkg/mage/dsl/ron -run TestRONEventTargetChooser -count=1`
   passed; native `go test ./pkg/mage -run TestTargetEventChooser -count=1`
   passed. The existing target-player-reference tests and both upkeep inventory
   tests also passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1` and `make test` passed.
4. Final `make lint` passed with **0 issues**, and `git diff --check` passed.
   Lint initially found a test variable named `copy`; renaming it to `copied`
   repaired the issue. Automatic import formatting in the new test files was
   reviewed. Focused tests, build, vet, engine short tests and the full suite
   passed again after this correction. No unrelated changes occurred.
5. Final `go run ./cmd/loadcards -json` exited 1 for required atomic collection
   rejection: **2,237 compiled / 208 rejected records**, **869 accepted / 117
   rejected names**. A keyed comparison with the fresh baseline proves that only
   `LEG.lines.ron:261:211: expected 1 positional arguments (TargetPermanentOfPlayersChoice)`
   disappears. All remaining rejected records retain identical diagnostics.
   No accepted record regressed. Expected collection rejection does not block
   this slice's commit. No environment, build, test or lint blocker remains.

Native files: new `target_event_chooser.go` and native tests; `target.go`,
`composable_target.go`, `game.go`, `text_spell_continuation.go` and `doc.go`.
Compiler files: new `event_target_chooser.go` and RON gameplay/rejection tests;
`singular_target_groups.go` and `player_relative_target.go`. Exact totals changed
in definition-plan, inventory, numeric-consumer and upkeep trigger/layer
inventory tests. API/rules-contract docs and this ledger record only the behavior
established here. No RON source, vocabulary, Go card definition or broad DSL
export changed.

### Ranked next candidates

1. **Temporary P/T switch — Transmutation.** One name / one record; a small
   existing-layer adapter and set/modify/switch order, lifetime and clone tests.
   Reassess fresh multi-card prerequisites before selecting single-card work.
2. **Variable discard result prerequisite — Recall.** Zero names from the
   prerequisite alone; one name / one record after the dependent saved-count
   graveyard return. Establish actual discard count and exact result lifetime;
   self-exile is already ready.
3. **Host unblocked attack event — Farrel's Mantle.** Zero immediate names from
   this event prerequisite; the complete record also needs event-subject aliases,
   target exclusion and optional damage-payment contracts. A shared host event
   adapter can enable those later rules; inspect broader host-event records first.

Complete collection gameplay, runtime loading, Go-card deletion, obsolete
Go-set generation removal and unused DSL cleanup remain gated. This run stops
after one completed slice.


## 2026-10-13 — TEMPORARY-PT-SWITCH

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -short -count=1` passed. Baseline loading exited 1
for atomic collection rejection: 2,237 compiled / 208 rejected records,
869 accepted / 117 rejected names. This was not a build or test failure.
The architecture, current status and latest ledger entry were read first;
targeted vocabulary signatures and full records supplied the requirements.
Complete records read: Transmutation and Recall in LEG, Farrel's Mantle in FEM,
and Imprison in LEG for the shared host-combat boundary. Goblin Flotilla and
Infinite Authority were also inspected when ranking subsequent work.

| Candidate | Distinct affected names | Estimated full-card gains | Other blockers and downstream benefit | Estimated effort |
| --- | --- | --- | --- | --- |
| Temporary P/T switch | Transmutation | 1 name / 1 record | No other syntax blocker; existing layer/duration service needs unary definition-validation repair. Resolving singular switch contexts share the adapter | Small adapter and focused identity/layer tests |
| Variable discard result | Recall | 0 from prerequisite; 1 after saved-count return | Needs actual chosen discard count, lexical producer lifetime and independent exact graveyard choice/return; self-exile already works | Medium for result contract; larger for complete procedure |
| Host combat event capture | Farrel's Mantle, Imprison | 0 from event prerequisite | Farrel's Mantle also needs event-subject aliases and exclusion of that subject from targets. Imprison needs activation-filter and grouped attack/block support. Existing damage-action payment can be reused later | Medium event/capture tests plus separate dependent adapters |

Ranked candidate #1 is dependency-ready. It does not unlock multiple names, so
Recall and the shared host-combat prerequisite were compared before selection.
One complete card with a small shared layer adapter gives the largest credible
immediate gain for this effort. The compared shared event prerequisite gives
zero immediate gains and leaves several independent contracts for both names.
No first-error count was used as a full-card unlock estimate. Reprints count
once; Transmutation has one record. All nested Transmutation forms were ready
except the selected operation and the native unary declaration check.

### Supported contract and reuse boundary

- `compileTemporaryLayer(Node, scope) (effectFactory, error)` accepts symbol
  `SwitchPT` inside a nonempty singular
  `CreatePermanentLayerEffectUntil(permanentRef, modifications, duration)` list.
  It composes with other supported modifications. Existing resolving scopes,
  references and duration shapes remain in force. Spells, activations, ordinary
  triggers and delayed bodies share this adapter. No new duration is added.
- RON owns symbol/arity rejection, lexical subject validation and per-factory
  construction. `SwitchPT()` and argument-bearing tuples reject, as do unbound
  player/event/source reads and unsupported list children. Static, plural,
  entry and cost/payment switch forms remain unsupported.
- `SwitchPowerToughness(PermanentRef) ContinuousModification` is reused.
  `layerModification.inspectDefinition` no longer adds power/toughness children
  for a switch. Its subject still validates; numeric P/T operations still
  reject missing operands. No signature or current caller is removed.
- Existing `Until`, exact captured modifications and layer 7d mechanics supply
  resolution-time membership, current P/T, set/modify/counter/switch order,
  repeated-pass stability, double-switch cancellation and expiry. The switch
  does not snapshot numeric values. Installed effects keep their visit through
  phasing and control changes; departure/return cannot transfer them.
- Target announcement/payment, resolution legality, source and attachment
  information, delayed capture, spell copying, state-based actions and game
  clones use shared engine services. No replacement, simultaneous-operation
  or transactional payment subsystem was added. Actual cleanup is tested
  natively; RON gameplay tests also check the shared end-of-turn expiry service.

CR 613.4b-d/613.5 establishes layer order and current switched values;
CR 400.7 establishes returned-object identity; CR 514.2 establishes expiry.
Existing target/phasing and state-based-action contracts remain in force.
Local `data/LEG.json` Oracle text is exactly “Switch target creature's power
and toughness until end of turn.” It matches the unchanged complete RON record.
No source defect was found. No RON source or vocabulary hash changed, so no
vocabulary refresh was needed. No Go-card fallback, name dispatch, mutable
factory target state, global result slot, partial production registry or runtime
declaration panic was added. Removal and runtime cutover gates remain unmet.
No DSL function or Go definition was removed.

### Failing-first evidence and card-to-test mapping

`TestPTSwitchDefinition` first failed because the valid switch exposed nil
numeric children (`modifications[0].power` and `.toughness`). The native layer
execution test already passed, confirming exact mechanics existed.
`TestRONPTSwitchTransmutation` and `LayersCopiesAndClones` then failed at
`LEG.ron:273:273: unsupported temporary layer action (SwitchPT)`.
Initial fixture errors used a nonexistent Shroud attribute/helper; these were
corrected before capturing the expected compiler failure. After implementation,
two fixture mistakes were corrected: changing card data alone retained the
creature capability, and `CopySpellOnStack` needs the spell source ID. The final
fixtures use the exact creature flag and source ID; no assertions were removed.

- **Transmutation → `TestRONPTSwitchTransmutation`.** Complete original record,
  failed and paid casting, illegal noncreature announcement without spending,
  no effect at announcement, return/phasing/shroud/type-change legality,
  control changes, phase cycle, normal spell departure and duration expiry.
- **Transmutation → `TestRONPTSwitchLayersCopiesAndClones`.** Complete record,
  independent copy retargeting, pending-clone isolation, current base values,
  bonuses and counters applied after switching, a second switch cancellation,
  cleanup and independent clone lifetime.
- **Transmutation → `TestRONPTSwitchStrictLoad`.** Exact record through strict
  registry APIs; two factory instances keep distinct IDs/targets and independently
  resolve against separate subjects. Invalid mixed collections leave the prior
  complete registry intact.
- **Transmutation → `TestRONPTSwitchSignedPowerAndStateActions`.** Negative,
  zero and positive power; nonpositive switched toughness reaches the normal
  state-based graveyard action.
- **Shared contexts → `TestRONPTSwitchSourceAndHost`.** Activated source and
  Aura-host references, departed Aura attachment information, installed phase
  cycle, returned subject exclusion and clone independence.
- **Shared contexts → `TestRONPTSwitchTriggerAndDelay`.** Ordinary entry body,
  saved next-end-step body, independent clone execution, absent and phased
  subjects and returned-object exclusion.
- **Rejection → `TestRONPTSwitchInvalid`.** Wrong shape/arity, unbound or
  incompatible subject, malformed duration, unsupported list child, static,
  collection and standalone forms.
- **Native → `TestPTSwitchDefinition`, `TestPTSwitchLayers`.** Valid unary and
  invalid subject declarations, continued numeric-operand checks, CR 613.4d
  examples, layer order, repeated passes, counters, double switch and actual
  cleanup actions.

Compilation gains and complete-card gameplay gains are separately one name /
one record. Both match the estimates. Synthetic context tests add contract
coverage without counting additional collection cards.

### Validation and changed files

All required checks passed in fast-fail order:

1. `go test ./pkg/mage/dsl/ron -run TestRONPTSwitch -count=1` and
   `go test ./pkg/mage -run TestPTSwitch -count=1` passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1` and `make test` passed.
4. `make lint` passed with **0 issues**; `git diff --check` passed.
   Automatic tools were reviewed: no unrelated edits or semantic rewrites
   occurred. Both focused packages passed again after lint.
5. `go run ./cmd/loadcards -json` exited **1**, as required while unsupported
   records remain: **2,238 compiled / 207 rejected records**, **870 accepted /
   116 rejected names**. Keyed baseline comparison removed only
   `LEG.lines.ron:273:273: unsupported temporary layer action (SwitchPT)`.
   There are no new or changed remaining diagnostics and no accepted-record
   regression. No environment, build, test or lint blocker remains.

Engine files: `composable_continuous.go`, new `pt_switch_test.go`, and `doc.go`.
Compiler files: `temporary_layer.go`, new `pt_switch_test.go`, and exact count
updates in `definition_plan_test.go`, `inventory_test.go` and
`numeric_consumers_test.go`. API/rules docs: `docs/dsl-reference.md`,
`docs/engine-rules-contracts.md`; status and evidence: this ledger.
No RON source, broad DSL export, runtime startup or Go card definition changed.

### Ranked next candidates

1. **Variable discard result — Recall.** Zero immediate names from this
   prerequisite; one after the separate saved-count graveyard return. Reassess
   shared multi-card work before starting a single-card procedure.
2. **Host combat event capture — Farrel's Mantle and Imprison.** Zero immediate
   names from the event prerequisite; concrete downstream benefit for two names.
   Keep capture/aliases separate from target exclusion and grouped event/filter
   adapters unless one cohesive supported context can complete in a run.
3. **End-of-combat bounded paired trigger — Goblin Flotilla.** Inspect the
   existing exact paired-combat service and duration boundary; estimated one
   full-card unlock after all nested forms validate.

Complete collection gameplay, runtime loading, Go-card deletion, obsolete
Go-set generation removal and unused DSL cleanup remain gated. This run stops
after this completed slice.


## 2026-10-13 — HOST-COMBAT-EVENT-CAPTURE

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The baseline loader exited 1
with 2,238 compiled / 207 rejected records and 870 accepted / 116 rejected names.
This was collection rejection, not a build or test failure.
Complete records were read for Recall (LEG:213), Farrel's Mantle (FEM:28),
Imprison (LEG:129), and Goblin Flotilla (FEM:34). Each occurs once.
Targeted vocabulary queries checked the discard result, return action, combat
host events, temporary trigger creation and combat duration; the full JSON
snapshot was not dumped. Reprints count once in the following comparison.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream benefit | Estimated effort |
| --- | --- | --- | --- | --- |
| Variable discard result | Recall | 0 now; 1 after saved-count return | Variable discard and a lexical actual-count producer precede graveyard choice/return. Self-exile is ready. This prerequisite serves one collection name. | Medium: amount/result adapter, exact discard outcomes, replacements and branch/copy lifetimes; return remains separate. |
| Host combat event capture | Farrel's Mantle; Imprison | 0 now; 2 after their other adapters | Farrel's Mantle needs event-relative target exclusion. Imprison needs the host/tap/nonmana activation filter. Shared exact host capture supplies both event aliases and Imprison's attack/block union. | Small to medium: four typed patterns, one compiler adapter, event scopes and native timing/identity tests. |
| Combat-bounded paired delayed trigger | Goblin Flotilla | 1 estimated after the complete adapter | Begin-combat payment, pair filters/capture and first-strike layers exist. Temporary registration must preserve creator controller, exact source visit, repeated pair events, combat-phase expiry and independent turn-long grants. Existing pair events and layer durations alone do not establish that delayed-trigger contract. | Medium to high: bounded event subscription lifecycle plus lowering and native extra-combat/expiry tests. |

Candidate 1 was evaluated first. Its zero-gain prerequisite only serves Recall.
The shared two-card host prerequisite wins over that single-card procedure and
the larger bounded-subscription work. Selection, reuse boundary and acceptance
cases were stated before editing. This run implements only host combat capture.

### Established signatures, contexts and reuse

- Engine signatures are `AttachedAttacks(*PermanentBinding) EventPattern`,
  `AttachedBlocks(*PermanentBinding) EventPattern`,
  `AttachedAttacksUnblocked(*PermanentBinding) EventPattern`, and
  `AttachedAttacksOrBlocks(*PermanentBinding) EventPattern`.
  Each requires a valid named binding. The union is one ability, not two grants.
- RON accepts battlefield `TriggerA` with `WhenACreatureAttacks`,
  `WhenACreatureBlocks`, or `WhenACreatureAttacksAndIsntBlocked`, each with
  `SinglePermanent(HostPermanent)` on an Aura. `Or` accepts exactly one attack
  and one block event over that same host, in either order. Other event names,
  subjects, unions and unbound reads reject during declaration compilation.
- `scope.hostCombat` identifies one lexical producer. Both
  `Trigger_ThatCreature` and `Trigger_ThatPermanent` lower to its saved host.
  Each factory creates its own `PermanentBinding` in the existing instance
  binding map. Inner delayed unblocked/death event producers retain priority
  over the corresponding outer aliases. Future-step bodies retain outer facts.
- RON owns shape, arity, context checks and aliases. The engine owns event
  membership and capture. `GenericTriggered.CheckEventType` consults an optional
  private typed-pattern matcher; existing patterns keep their single-type checks.
  No broad event-list API or current caller was removed.
- Reused services: native attacker/blocker/unblocked events, the singular-block
  flag, `EventSourceIsAttachedTo`, `eventPermanentCapture`, `capturedPermanent`,
  `From`, pending-trigger controller snapshots, resolution maps, optional mana
  payment, combat removal, temporary layers, future-step capture, stack copies
  and game clones. No parallel scheduler, zone service or payment system exists.
- Detection requires a present unphased creature at the attached event visit;
  explicit stale incarnations reject. Resolution cannot follow a returned host.
  Reattachment cannot change pending subjects. Phasing uses existing exact
  reference behavior; phase cycles retain the visit. Source departure and control
  changes do not erase pending bodies or change their original controller.
  Missing subjects do not stop independent later actions. No new LKI property,
  replacement-order or simultaneous-zone contract was needed.

CR 508.1, 509.3a/g and 603.2/603.3a establish declaration timing, singular block
cardinality and pending control. CR 400.7 and existing phasing/reference services
supply visit identity. Complete source records were retained unchanged. No data
defect was corrected. No RON source or vocabulary hash changed, so no vocabulary
refresh was required. Complete card loading and removal gates remain unmet.
No Go card definition, generator, production registry or broad DSL function changed.

### Failing-first evidence and card-to-test mapping

`TestRONHostCombatIdentity` and the native tests first failed on
`unsupported source combat reference (HostPermanent)`; the union also rejected.
`TestHostCombatDefinition` first failed to compile because all four new API
signatures were absent. Initial test fixtures used BasePlayer decision helpers
and an incorrect keyword/helper name; these were fixed before the recorded
expected failures. The valid declaration tests then exposed a missing binding
producer declaration, which was fixed in the engine pattern.
An added unsupported host-becomes-blocked rejection test failed because the
adapter's default could select the unblocked pattern; explicit event validation
fixed it. Later fixture corrections used explicit trigger resolution after
`ExecuteAttackers`, current game pointers after cloning, and the actual remaining
loader diagnostics. No assertion was removed to bypass unsupported behavior.

- **Farrel's Mantle and Imprison → `TestRONHostCombatNative`.** Complete records
  are read; their original event nodes run in native attacker/blocker steps with
  a synthetic observable body. Blocked/unblocked cases establish timing. This
  is event-clause coverage, not complete-card gameplay evidence.
- **Imprison → `TestRONHostCombatImprisonClause`.** Its unchanged entire combat
  trigger clause, including optional `{1}` payment, paid tap/removal and unpaid
  Aura destruction. Native attack and block cases cover paid, declined and
  unaffordable paths; declining retains mana. The separate activation clause
  is deliberately absent from this test fixture and remains a loader error.
- **Shared capture → `TestRONHostCombatIdentity`.** All three singular events;
  distinct host/other subjects, reattachment, source departure/control changes,
  returned hosts, phasing and phase cycles, inactive sources, ability removal,
  independent pending events, stack copying and clone isolation. Life assertions
  establish saved trigger controller and independent later-action execution.
- **Union → `TestRONHostCombatUnionBlocksOnce`.** One enchanted blocker blocks
  two attackers through native declaration; exactly one life gain results.
- **Frames/factories → `TestRONHostCombatDelayedAndFactories`.** Two factory
  instances capture different hosts, arm separate next-end-step bodies, retain
  visits after reattachment and execute independently in a clone. An absent
  host does not redirect its delayed action.
- **Strict boundaries → `TestRONHostCombatInvalid` and
  `TestRONHostCombatCompleteRecordsStayRejected`.** Unsupported event, malformed
  reference, incompatible subject/player, mismatched and duplicate unions,
  escaped event binding, plus complete unchanged records' remaining blockers.
- **Native declaration/identity → `TestHostCombatDefinition` and
  `TestHostCombatEventIdentity`.** Valid producer dominance and invalid binding
  declarations; both union event kinds, unrelated-event exclusion, stale visits,
  phased subjects and suppression of extra per-attacker block events.

Estimated and actual full-card gains are both **0 names / 0 records**.
Complete-card gameplay gains are also zero. Two distinct cards gain prerequisite
and event-clause evidence; Imprison gains its full combat-clause evidence.
These partial mappings do not satisfy the full-card completion gate.

### Validation and loader evidence

All required checks passed:

1. `go test ./pkg/mage/dsl/ron -run TestRONHostCombat -count=1` and
   `go test ./pkg/mage -run TestHostCombat -count=1` passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1` and `make test` passed.
4. `make lint` passed with **0 issues**; `git diff --check` passed.
   Automatic output and the resulting diff were reviewed. No unrelated
   edits or semantic automatic rewrites occurred.
   Both focused packages passed again after lint.
5. Final `go run ./cmd/loadcards -json` exited **1**, retaining atomic rejection:
   **2,238 compiled / 207 rejected records**, **870 accepted / 116 rejected names**.
   Keyed baseline comparison found no added or removed rejected records and
   exactly one diagnostic change: Farrel's Mantle moves from
   `FEM.lines.ron:28:254: unsupported source combat reference (HostPermanent)` to
   `FEM.lines.ron:28:308: unsupported target exclusion (Trigger_ThatCreature)`.
   Every other diagnostic is unchanged. No accepted record regressed.
   No environment, build, test or lint blocker remains.


Changed files: engine `composable_host_combat.go`, `triggered.go`,
`host_combat_test.go` and `doc.go`; compiler `host_combat.go`, `source_attack.go`,
`combat_pair.go`, `compiler.go`, `permanent_scope.go` and `host_combat_test.go`;
API/rules documentation `docs/dsl-reference.md` and
`docs/engine-rules-contracts.md`; status/evidence in this ledger.

### Ranked next candidates

1. **Event-relative target exclusion — Farrel's Mantle.** Inspect binding-aware
   announcement/retarget legality and the remaining complete payment body.
   Estimated one full-card unlock after this separate adapter; reassess fresh
   shared multi-card candidates before choosing single-card work.
2. **Host activation filter — Imprison.** Inspect the existing activation event
   capture/counter service with host identity, tap-self cost and nonmana filters.
   Estimated one full-card unlock; search other activation filters for shared value.
3. **Combat-bounded paired delayed trigger — Goblin Flotilla.** Estimated one
   full-card unlock after creator control, exact visit, arming and phase-expiry
   contracts are established. Do not replace the delayed trigger with a live
   source grant without proving equivalent control/lifetime semantics.

Recall's variable discard result remains deferred. Complete collection gameplay,
runtime loading, Go-card deletion, obsolete Go-set generation removal and unused
DSL cleanup remain gated. This run stops after this one completed prerequisite.


## 2026-10-14 — EVENT-TARGET-EXCLUSION

### Environment, candidates and selection

The starting working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Fresh
`go run ./cmd/loadcards -json` exited 1 with **2,238 compiled / 207 rejected
records**, **870 accepted / 116 rejected unique names**. Collection rejection
was expected; it was not a build, test or environment failure.

The latest ranked candidates were evaluated from complete RON records and
nested forms. Reprints count once. Targeted vocabulary reads covered `Other/1`,
`TargetPermanent/1`, `WhenAPlayerActivatesAnAbility/2`, `HasTapSelfInCost`,
`NonManaAbility` and `UntilEndOfCombat`. No old Go definition supplied requirements.

| Candidate | Distinct affected names | Estimated full-card gain | Other blockers and downstream value | Full implementation/test effort |
| --- | --- | --- | --- | --- |
| Saved event target exclusion | Farrel's Mantle | 1 name / 1 record | Its complete damage payment, current controller/power reads and temporary combat-assignment rule already lower. The missing shared query contract carries saved permanent facts into placement, mode availability, copy choices and resolution. It also serves existing attached-tap and combat-opponent producers. | Moderate; extend typed target restrictions and the existing event query, then test identity and complete Aura gameplay. |
| Host activation filtering | Imprison; related accepted shapes on Artifact Possession, Haunting Wind and Powerleech | 0 for filtering alone; 1 after ability capture/countering | The three accepted cards already use the non-tap artifact adapter. Imprison adds host identity, a tap-symbol cost, a nonmana restriction and `Trigger_ThatActivatedAbility`. The current ability adapter only reads target bindings; event ability capture is another required contract. Its separate combat clause is already supported. | Higher; event filtering and typed stack ability capture, with payment/rollback and countering tests. |
| Combat-bounded paired delay | Goblin Flotilla | 1 name / 1 record | Islandwalk, beginning-of-combat trigger, optional red payment and first-strike layer already lower. The paired delayed event needs creator control, exact visit, arming and `UntilEndOfCombat` expiry. Related end-of-combat layer/prevention cards already compile; Raging River and Johan have independent blockers. | Higher; delayed lifetime and paired event integration with native phase-boundary tests. |

Candidate 1 was selected as the dependency-ready final prerequisite for Farrel's
Mantle and a shared event-target query capability. It supplies the same credible
immediate gain as the delayed candidate with less new lifecycle machinery.
No compared candidate supplied a larger immediate multi-card unlock.
The prior estimate for Imprison requires both activation filtering and ability
capture; the first diagnostic alone did not show the complete dependency chain.

### Supported contract and reuse boundary

- Engine API:
  `TargetOtherThanSavedPermanent(binding *PermanentBinding) PermanentTargetOption`.
  Use it with typed permanent target declarations. Definition inspection checks
  the saved reference, including nil bindings and producer dominance. Missing,
  zero or incomplete runtime visits give no candidates. A valid saved visit
  excludes only that incarnation, regardless of current control or attachment.
  A returned card's new visit does not inherit the exclusion.
- RON accepts `Other(eventRef)` in ordinary battlefield trigger singular
  permanent target filters, including nested `And` and numbered target groups.
  The reference must lower to `permanentEvent`; the factory builds its option
  from that instance's `PermanentBinding`. Host combat aliases, attached-tap
  captures and combat-opponent aliases use the existing reference compiler.
  Existing source aliases still use `TargetOtherThanSource()` unchanged.
  Direct hosts, prior targets, resolving choices, death-card references,
  unbound/wrong-type/tuple aliases, disjunctions and delayed target scopes reject.
- RON owns spelling, arity and context. The engine owns exact visit comparison
  and event query facts. `capturedTriggerTargetFacts` extends the existing
  snapshot collector; `targetEventQuery` extends the existing player query with
  permanent bindings. Placement, modal availability, copy retargeting and
  resolution use these facts without executing an action, making a payment or
  replacing the live resolution frame.
- Stack objects reuse `capturedPermanents`; existing stack-copy and clone map
  services keep facts separate. Pending events keep detection-time subjects.
  Reattachment, source departure/control changes and phase cycles cannot replace
  a subject. Target departure/return, phasing and protections retain normal
  resolution legality. Missing legal targets remove the trigger before payment.
- Reused services: typed event captures, exact references, target groups,
  APNAP placement/order, target protections, retained source information,
  `ControllerOfWithLKI`, `PowerOfWithLKI`, `DamageSourceOf`, `DamagePayment`,
  `PayActionOrElse`, damage prevention/replacements, temporary
  `AssignsNoCombatDamage`, cleanup, stack copying and game cloning.
  No parallel event, zone, payment or replacement service was added.

CR 400.7 supplies return identity; CR 603.3a/d supplies controller and target
placement; CR 608.2b supplies target rechecks; CR 608.2h supplies departed host
information. CR 702.26b/d makes a phased host unavailable for the damage action,
while a phase cycle keeps its visit. The existing damage-payment service allows
a departed host's saved source information and rejects a currently phased host.
The target exclusion itself reads identity and needs no characteristic LKI.
No simultaneous-operation or replacement-order contract changed.

No source-data defect was found or corrected in this slice. Gaea's Liege and
Fastbond remain separate known source investigations. All RON records and their
hashes stay unchanged; no vocabulary snapshot refresh is required.
No Go definition, startup registry, generator or broad DSL function was removed.
Runtime cutover and removal gates remain unmet.

### Failing-first evidence and card-to-test mapping

The new engine tests first failed to compile because
`TargetOtherThanSavedPermanent` and `targetEventQuery` were absent. After a test
fixture correction to the existing `TemporaryBoost` API, the RON tests failed on
`unsupported target exclusion (Trigger_ThatCreature)` for the complete unchanged
Farrel's Mantle record. Implementation then supplied the adapter and query path.

Later fixture corrections followed observed native contracts: default trigger
ordering reverses pending triggers; `ExecuteBlockers` resolves triggers at once;
clones have separate decision players, so explicit cloned decision wrappers keep
accept/decline choices equal. An added creature fixture supplied required CardPT.
A synthetic Aura loader fixture needed an enchant rule and was replaced with
the complete raw repository record. A phased-host expectation was corrected with CR 702.26b/d and existing damage
source contract evidence: that host cannot pay a damage action. No production
rule was simplified and no failed assertion was removed to hide a capability gap.
Collection assertions now require 2,239/206; the former rejection test retains
Imprison, while the new complete-record tests require Farrel's Mantle acceptance.

- **Farrel's Mantle → `TestRONEventTargetExclusionMantle`.** Its complete unchanged
  record casts and attaches as an Aura, then runs its original targeted trigger,
  optional damage payment and combat-assignment rule. Paid/declined/prevented
  paths, live power, correct payer after host control changes, reattachment,
  source departure, host departure/return/LKI, host phasing and phase cycles,
  target phasing/return/shroud, copied triggers, targetless attacks and clone
  isolation have explicit outcomes. Target choice occurs before the optional
  resolution offer; paying suppresses combat damage even when damage is prevented.
- **Farrel's Mantle → `TestRONEventTargetExclusionNativeCombat`.** Native attacker,
  blocker and combat-damage steps cover paid, declined and blocked attacks.
  Cleanup ends suppression; a subsequent attack can assign damage again.
- **Farrel's Mantle → `TestRONEventTargetExclusionStrictLoad`.** The complete raw
  repository record loads into a test registry. An incompatible alias rejects
  atomically and preserves the prior registry. Compilation and engine definition
  validation both pass without a Go factory fallback.
- **Shared compiler → `TestRONEventTargetExclusionBindings`.** Attack, block,
  unblocked and attached-tap producers; two factories and two numbered target
  groups per pending trigger keep separate exclusions and execute real actions.
- **Shared compiler → `TestRONEventTargetExclusionCombatOpponent`.** Both combat
  opponent aliases exclude the opponent rather than the source or chosen target.
- **Boundaries → `TestRONEventTargetExclusionInvalid`.** Wrong roles, unbound and
  escaped aliases, tuple spelling, malformed arity, unsupported direct host and
  disjunction declarations reject before factory execution.
- **Engine → `TestTargetSavedExclusionDefinition`, `Identity`,
  `CopyAndResolution`, `ModeAvailability`.** Producer dominance and nil bindings;
  independent pending captures despite conflicting live frames; exact visits
  through phasing/return; missing facts; valid and invalid copy retarget choices;
  target protections/return/phasing; source and host departure; cloned outcomes;
  resolution uses stack facts rather than a conflicting live binding; a mode
  with only its excluded subject is unavailable.

Estimated and actual gains are **1 unique name / 1 record**. Complete-card
RON gameplay coverage gains **Farrel's Mantle**. Shared synthetic contract tests
are separate evidence and do not add inventory card coverage. All previously
accepted records remain accepted.

### Validation and loader evidence

Final required checks passed:

1. `go test ./pkg/mage/dsl/ron -run TestRONEventTargetExclusion -count=1`
   and `go test ./pkg/mage -run TestTargetSavedExclusion -count=1` passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1` and `make test` passed.
4. `make lint` passed with **0 issues**; `git diff --check` passed.
   The first lint run requested names for the snapshot collector's two map
   results. That style fix was applied, and all preceding checks were rerun
   successfully before the final lint run. Automatic formatter changes were
   limited to import grouping in new tests; the final diff was reviewed.
   Both focused suites passed again after lint. No unrelated edit remains.
5. Final `go run ./cmd/loadcards -json` exited **1**, preserving atomic rejection:
   **2,239 compiled / 206 rejected records**, **871 accepted / 115 rejected names**.
   Baseline comparison by card/file/line found exactly one removed rejection:
   `Farrel's Mantle`, `FEM.lines.ron:28`. There were no added rejected records
   and no changed remaining diagnostics. No accepted record regressed.
   The complete collection still cannot publish a production RON registry.

There is no remaining environment, build, test, vet or lint blocker for this slice.

Changed files: compiler `player_relative_target.go`; engine
`target_saved_exclusion.go`, `target.go`, `composable_target.go`,
`target_player_reference.go`, `game.go`, `spell_copy.go`, `stack_resolver.go`;
new tests `event_target_exclusion_test.go`, `target_saved_exclusion_test.go`;
updated collection/rejection assertions `definition_plan_test.go`,
`inventory_test.go`, `numeric_consumers_test.go`, `host_combat_test.go`;
API/rules documentation `doc.go`, `docs/dsl-reference.md`,
`docs/engine-rules-contracts.md`; status/evidence in this ledger.

### Ranked next candidates

1. **Host activation filter and exact ability capture — Imprison.** The completed
   slice removes Farrel's Mantle from the backlog. Imprison's full activation
   clause needs host/tap/nonmana detection and a typed saved stack ability for
   countering. Filtering alone has zero unlocks; the cohesive complete contract
   estimates one. Preserve the three accepted artifact activation cards.
2. **Combat-bounded paired delayed trigger — Goblin Flotilla.** Estimated one
   full-card unlock. Establish creator control, exact source/opponent visits,
   arming and actual end-of-combat phase expiry; do not substitute a live grant.
3. **Variable discard payment/result — Recall.** Estimated one unlock after
   announced X, distinct hand visits, rollback and saved result cardinality are
   established. Reassess shared multi-card prerequisites before single-card work.

Complete collection gameplay, runtime loading, Go-card deletion, obsolete Go-set
generation removal and unused DSL cleanup remain gated. This run stops after
this one complete capability slice.

## 2026-10-14 — HOST-TAP-NONMANA-ACTIVATION

### Baseline, candidate comparison and selection

The working tree was clean before this slice. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Fresh strict loading exited 1 with
**2,239 compiled / 206 rejected records** and **871 accepted / 115 rejected
names**. That exit is collection rejection, not a build or environment failure.
No user changes or unrelated untracked files were removed.

The latest entry's first candidate was evaluated first. Complete unchanged
records were read for Imprison (LEG:129), Goblin Flotilla (FEM:34), and Recall
(LEG:213), including all nested clauses. Targeted vocabulary queries covered
WhenAPlayerActivatesAnAbility, HasTapSelfInCost, CounterAbility, CreateTriggerUntil,
DiscardNumberCards, NumCardsDiscardedThisWay and the saved-count graveyard return.
Reprints are counted once. Complete records were also read for the three accepted
artifact activation callers: Artifact Possession, Haunting Wind and Powerleech.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and dependencies | Downstream value and effort |
| --- | --- | --- | --- | --- |
| Host tap-cost nonmana activation with exact ability capture | Imprison; preserve Artifact Possession, Haunting Wind and Powerleech | 1 name / 1 record | Aura casting, ordinary host combat union, optional mana payment, counter service, tap and combat removal already exist. Filter lowering alone would leave the ability alias unsupported. | A reusable event-bound AbilityRef serves counter/controller consumers without target declarations. Small to medium: one event identity field, typed binding, three event options, scoped adapter and identity/payment tests. |
| Combat-bounded paired delayed trigger | Goblin Flotilla | 1 name / 1 record after the complete contract | Islandwalk, begin-combat payment, pair filters and first-strike layers exist. Creator control, exact visits, arming and actual UntilEndOfCombat expiry remain required. | Shared bounded event subscription can extend temporary triggers. Medium to high: native phase boundaries, extra combat, multiple pairs, source departure and turn-long grant isolation. |
| Variable discard effect and saved-count graveyard return | Recall | 1 name / 1 record only after both producer and consumer | X announcement and self-exile are ready. DiscardNumberCards(ValueX), actual discard cardinality, and ReturnNumberCardsFromPlayersGraveyardToHand need complete adapters. The record declares a resolving discard effect, not a discard payment. | Shared variable discard/result service extends the two already accepted fixed-discard cards. Medium to high: distinct hand visits, replacements, partial discard, saved count, ordered graveyard choices and copy isolation. |

Select Imprison's cohesive activation contract: it has the smallest remaining
end-to-end work among the compared one-name gains and closes the first ranked
candidate's shared exact-ability prerequisite. No multi-card unlock is claimed.
The other two complete gains require larger independent contracts. The selected
slice stops after this one capability.

Before editing, the contract and acceptance cases were stated: exact completed
host/tap/nonmana activation capture; paid/unpaid branches; independent pending
activations; departure, return, control, phasing, copies and clones; native
attack/block branches; invalid declarations and strict atomic loading.

### Supported signatures, boundaries and engine reuse

RON accepts one battlefield Aura `TriggerA` with:

`WhenAPlayerActivatesAnAbility(AnyPlayer,
And([AbilityOfPermanent(HostPermanent), HasTapSelfInCost, NonManaAbility]))`.

The three operands can appear in any order, exactly once. HostPermanent needs
the existing Aura scope. `Trigger_ThatActivatedAbility` binds one exact activated
stack object and supplies CounterAbility or ControllerOfAbility in the same
resolving body, including optional payment branches. Other player/source/filter
forms, disjunctions, tuple symbols, absent/duplicate operands, incompatible
aliases, cross-rule reads and delayed RON ability reads reject at compilation.
The event does not manufacture player or permanent event aliases.

New engine API:

- `BindAbility(name string) *AbilityBinding`.
- `FromAbility(binding *AbilityBinding) AbilityRef`.
- `ActivatedAbilityEvent().FromAttachedPermanent().WithTapCost().NonMana()`.
- `ActivatedAbilityEvent().WithAbility(binding *AbilityBinding)`.
- `core.GameEvent.ActivatedAbilityID uuid.UUID` for completed stack activations.

`ron` owns spelling, operand validation and the lexical ability event plan.
The existing target-ability plan remains intact; a typed reference plan selects
its target or event producer. Every factory call builds a fresh event binding.
`mage` owns exact activation identity, matching and typed reads. The new binding
uses a private key in existing stack-identity resolution frames. Existing
snapshot, delay, cloning, APNAP, counter, cost transaction and combat services
remain the only execution paths. No parallel event/payment/stack service,
card-name branch, Go-card fallback, live ability grant, skipped rule or partial
production registry was introduced.

Mana activations emit no stack identity. A targeted mana-producing ability is
nonmana and emits one. Tap filtering reads the existing written tap-symbol
cost flag. Failed payments and illegal tap activations emit no completed event.
The host filter checks the Aura's exact current attachment and event incarnation.
Capture occurs at detection, before trigger placement. A copied activation
creates no new activation event. A copied trigger retains the saved ability,
while its controller pays its own optional cost. Pending source control changes
do not change the original trigger's payer. Countering never refunds the tap cost.

CR 602.2a–b supplies stack identity and payment timing; CR 603.3a/b supplies
controller capture and APNAP placement; CR 605.1a/3b/5a supplies mana classification;
CR 701.6a–b supplies countering without refunds; CR 400.7 and 702.26b/d supply
return identity and phasing. Source LKI is not required to counter a still-present
ability; source departure cannot erase the saved stack object. Missing or departed
abilities supply no subject, and later instructions continue. Clones and delayed
engine bodies keep their own frames. No replacement-order or simultaneous-operation
contract changed.

### Failing-first evidence and card-to-test mapping

New engine tests first failed to compile because BindAbility, FromAbility and
the event filter/capture methods did not exist. Existing-API test fixture calls
were corrected (CounterStackObject, UntapPermanent, ManaPayment, AddManaAmount,
UUIDs and Aura construction). The complete unchanged Imprison test then failed
with `LEG.ron:129:217: supply one artifact source and one non-tap cost filter (And)`.
Implementation supplied the engine contract and scoped lowering. Further fixture
corrections set the main phase before Aura casting and used explicit opponent IDs.
No production rule was simplified or failed assertion removed to hide a gap.

- **Imprison → `TestRONHostActivationImprison`.** Complete unchanged LEG record
  validates, casts and attaches as an Aura. Native tap activation offers payment
  only at resolution. Paid, declined and insufficient-mana branches; two pending
  activations; reattachment; source departure/control; host return/phasing and
  phase cycles; missing ability; trigger copy with another payer; indestructible
  unpaid Aura; clone payment/resource isolation all have explicit outcomes.
- **Imprison → `TestRONHostActivationCombat`.** Former extracted combat-clause
  fixtures now compile the complete record. Native attack and block declaration
  cover paid, declined and short payments, tap and combat removal, source
  destruction and resulting combat damage. Removing the sole blocker unblocks
  its attacker. Shared `TestCombatExitSoleBlockerHistory`, `MultipleGroupsAndCloneHistory`
  and `DepartureKeepsHistory` retain the harder blocker-history contract.
- **Imprison → `TestRONHostActivationStrictLoad`.** The complete raw repository
  line loads into a test registry. A spell alias in the ability role rejects
  atomically and preserves the existing registry; no Go definition is used.
- **Shared compiler → `TestRONHostActivationBindings`.** Reordered conjunction,
  two fresh factory instances, opposing observers, multiple pending captures and
  typed ControllerOfAbility reads use the matching stack object.
- **Boundaries → `TestRONHostActivationInvalid`.** Unsupported players/sources,
  malformed arity/symbols, missing/duplicate operands, disjunctions, non-Aura
  host scopes, incompatible roles, unbound/cross-event reads and delayed aliases
  reject before factory execution.
- **Engine → `TestActivationAbilityCaptureIdentity`, `FiltersAndRollback`,
  `Definition`, `MissingAndExactEvent`, `DelayedFrame`.** Independent activations,
  exact source/host visits, controller capture, ability versus trigger copies,
  absent ability and continued actions, phasing, clone isolation, true mana
  versus targeted production, summoning sickness, failed mana payment, ability
  removal, declaration dominance and empty bindings, spell-role exclusion,
  stale incarnation events, and saved delayed frames are covered.
- **Preserved callers → `TestRONArtifactEvent*` and `TestRONAbilityTarget*`.**
  Artifact Possession, Haunting Wind, Powerleech, Rust and Ayesha Tanaka retain
  their existing event, target, gameplay and rejection evidence. The former
  complete-record rejection assertion now retains Goblin Flotilla.

Estimated and actual gains are **1 unique name / 1 record**. Complete-card
RON gameplay coverage gains **Imprison**. Shared synthetic contracts do not count
as additional inventory coverage. All previously accepted records remain accepted.

### Validation and loader evidence

All required checks passed in the requested fast-fail order:

1. `go test ./pkg/mage/dsl/ron -run TestRONHostActivation -count=1`
   and `go test ./pkg/mage -run TestActivationAbilityCapture -count=1` passed.
   The focused preservation check for TestRONArtifactEvent, TestRONAbilityTarget
   and the retained Goblin Flotilla rejection also passed.
2. `make build` and `go vet ./...` passed.
3. `go test ./pkg/mage/... -short -count=1` and `make test` passed.
   The implement-engine skill's `go test ./... -short -count=1` also passed.
4. `make lint` passed with **0 issues**; `git diff --check` passed.
   Automatic formatting and final diffs were reviewed; no unrelated changes
   were introduced. Both focused suites passed again after lint.
5. Final `go run ./cmd/loadcards -json` exited **1** with atomic collection
   rejection: **2,240 compiled / 205 rejected records**, **872 accepted / 114
   rejected names**. A baseline comparison keyed by card/file/line removed
   exactly Imprison at LEG.lines.ron:129. No rejection was added and no remaining
   diagnostic changed. No accepted record regressed. Complete collection
   publication still requires the remaining unsupported records.

There is no environment, build, test, vet or lint blocker for this slice.

No source-data defect was found or corrected. Local `data/LEG.json` Oracle text
agrees with the complete Imprison RON record. Known Gaea's Liege and Fastbond
investigations remain separate. RON files and hashes are unchanged; no vocabulary
snapshot refresh is needed. No Go definition, startup registry, generator or
broad DSL function was removed; their cutover/removal gates remain unmet.

Changed files: compiler `host_activation.go`, `ability_target.go`,
`artifact_event.go`, `compiler.go`, `player_scope.go`; engine
`activation_ability_capture.go`, `composable_events.go`, `game.go`,
`core/event.go`; new tests `activation_ability_capture_test.go`,
`host_activation_test.go`; complete-record combat and collection assertions in
`host_combat_test.go`, `definition_plan_test.go`, `inventory_test.go`,
`numeric_consumers_test.go`; API/rules documentation `doc.go`,
`docs/dsl-reference.md`, `docs/engine-rules-contracts.md`; this progress ledger.

### Ranked next candidates

1. **Combat-bounded paired delayed trigger — Goblin Flotilla.** Estimated one
   full-card unlock. Complete creator control, exact source/opponent visits,
   arming and actual combat-phase expiry using the shared temporary trigger
   service. Independent first-strike grants last until turn end.
2. **Variable discard effect/result and saved-count return — Recall.** Estimated
   one full-card unlock. Use announced X and actual discard outcomes, including
   replacements and partial discard; save cardinality before graveyard choices.
   The RON instruction is an effect, not a mandatory payment.
3. **Granted targeted activation — Equinox.** Reassess the complete record and
   rules contract before estimating gains. Its first diagnostic is the targeted
   body in an untargeted grant; predicting destruction of a land is a separate
   dependency. Recheck shared multi-card grant prerequisites before committing
   to this single-name capability.

Complete collection gameplay, runtime loading, Go-card deletion, obsolete Go-set
generation removal and unused DSL cleanup remain gated. This run stops after
this one complete capability slice.

## 2026-10-14 — COMBAT-BOUNDED-PAIRED-DELAY

### Environment, candidates and selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed before editing. Baseline
`go run ./cmd/loadcards -json` exited 1 with collection rejection:
2,240 compiled / 205 rejected records, 872 accepted / 114 rejected names.
This was an unsupported collection, with no build or test failure.

The latest entry's ranked candidates were evaluated from complete records:

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Combat-bounded pair subscription | Goblin Flotilla | 1 name / 1 FEM record | Pair matching, opponent capture, optional mana and turn keyword grants already exist. Missing creator-controlled repeating subscription and actual combat-phase expiry. Shared saved-frame helper and expiry service support further delayed-event adapters. | Small engine lifecycle and compiler slice, with identity/timing/native combat tests. |
| Variable discard result and saved-count return | Recall | 1 name / 1 LEG record if both operations are complete | Must use announced X, actual discard outcomes after replacements, and cardinality saved before resolving graveyard choices. This is a discard effect, not a mandatory cost. ThisSpell exile already exists. Enables later actual-discard result consumers. | Larger effect/result/choice contract and replacement tests. |
| Targeted granted activation | Equinox | 0 for grant prerequisite alone; 1 after a separate prediction capability | Complete Aura record grants a tap activation targeting any spell, then checks WouldDestroyAPermanent for a land controlled by the activation controller. Targeted grant and destruction prediction are distinct gaps. Enables reusable grants but needs a new consumer audit for multi-card value. | Grant machinery plus a substantially larger prediction contract; unsuitable as one small complete card slice. |

No compared candidate offered a credible immediate multi-card unlock. Select
Goblin Flotilla: candidate #1 is dependency-ready, has one independently
verified remaining blocker, and supplies the largest credible gain for the
estimated effort. No Go card definition supplied requirements. Complete RON
records and targeted vocabulary keys CreateTriggerUntil and the pair events
were inspected. Local data/FEM.json Oracle text agrees with the unchanged RON.
No source-data defect, environment failure or missing payment/layer contract
was found.

### Contract, exact context and reuse boundary

Engine signature:
`WhenCombatPairUntilEndOfCombat(subject PermanentRef, filter PermanentFilter,
opponent *PermanentBinding, body ResolvingEffect) ResolvingEffect`.
It saves the watched incarnation and creator controller at resolution, before
future events. Each matching pair captures its opponent at detection, before
APNAP placement. The body receives a separate saved resolution frame with that
capture. Source control changes and ability removal do not end the subscription.
Source departure and phasing prevent new matches; a phase cycle preserves the
visit. Departure/return cannot transfer a watched or opposing identity.
Pending bodies survive source departure and subscription expiry. A missing or
phased opponent prevents its action but does not stop subsequent instructions.

`TriggerSystem.ClearEndOfCombat()` removes only combat-bounded subscriptions
when native EndCombat priority has finished, before effect expiry and combat
reset. Cleanup also removes unused subscriptions. First-strike grants created
by the body remain active until their own turn-end expiry. Copies and clones
keep independent frames, resources and effect lifetimes.

RON signature: `CreateTriggerUntil(Or([WhenACreatureBlocksACreature(
SinglePermanent(ThisPermanent), filter), WhenACreatureBecomesBlockedByACreature(
SinglePermanent(ThisPermanent), filter)]), ActionList([...]), UntilEndOfCombat)`.
It accepts the two directions in either order, with identical operands and the
existing combat-pair creature-filter grammar, in resolving permanent-source
contexts. Trigger_ThatOtherCreature and the existing Trigger_ThatOtherPermanent
alias read the delayed body's opponent. Bindings are fresh per factory.
Host subjects, single-direction events, other durations, malformed operands,
empty bodies, noncreature filters and incompatible/unbound aliases reject.
The engine additionally supports typed watched targets, proved with a synthetic
spell; this does not broaden the RON adapter.

RON owns spelling, role checks and lexical bindings. The engine reuses
filteredCombatPairPattern matching/capture, DelayedTrigger persistence, stack
placement, exact reference services and capturedDelayedBody. The duplicated
step/departure snapshot code was extracted to captureDelayedBody without a
behavior change. A shared compileCombatPairEvent plan retains ordinary printed
pair triggers and their text-layer path. Optional payment, mana rollback,
keyword layers and native first-strike damage remain existing services.
No parallel event system, global result, card-name branch or fallback was added.

CR 603.7a–e supplies arming, duration, exact visits and creator control;
CR 603.3b supplies APNAP placement; CR 511.2 supplies combat-phase expiry;
CR 400.7 and 702.26 supply return identity and phasing. LKI cannot restore a
subject for an action. No new replacement ordering or simultaneous-operation
contract was needed; native combat uses the existing damage batches.

### Failing-first evidence and card-to-test mapping

Engine tests first failed to compile for the missing
WhenCombatPairUntilEndOfCombat signature. Test fixture calls were corrected to
GrantKeywordModification and explicit expected life totals. New complete-record
RON tests then failed at FEM.lines.ron:34:631 with
`unsupported delayed departure duration (UntilEndOfCombat)`.
Implementation supplied the engine contract and scoped adapter. Further fixture
corrections used ManaPool.Count, the loader's pattern argument and the native
FuncEffect signature. Clone assertions reacquire the game's current permanents
instead of reading stale pointers. No rule or assertion was removed to hide a gap.

- **Goblin Flotilla → TestRONCombatPairDelayFlotilla.** Complete unchanged record
  validates. Paid red mana, an affordable decline with mana retained, and an
  unaffordable payment; resolution timing, source control/departure, trigger
  copies, repeated matches and separate combat/turn lifetimes are checked.
- **Goblin Flotilla → TestRONCombatPairDelayNative.** Native beginning of combat,
  attack and block declarations, both pair directions, opponent first strike,
  first-strike lethal damage before normal damage, and combat expiry.
- **Goblin Flotilla → TestRONCombatPairDelayInstancesAndStrictLoad.** Complete raw
  record loads with no fallback. Invalid aliases reject atomically while keeping
  the registry. Opposing factory instances and pending opponents stay separate.
  Clones expire independently. An additional combat chooses payment again.
- **Goblin Flotilla → TestRONTextLandwalkRecords.** Complete-record islandwalk
  and text revision evidence now includes Flotilla: 13 names / 23 records.
- **Compiler → TestRONCombatPairDelayInvalid.** Invalid durations, host scope,
  duplicate directions, incompatible aliases and empty bodies reject before use.
- **Engine → TestCombatPairDelayIdentity, DurationAndEvents, Definition,
  EndPriority, IndependentCaptures, CleanupAndMissing.** Saved creator control,
  source/opponent return and phasing, phase cycles, ability removal, departure
  after detection, opponent control, copies/clones, stale event incarnations,
  multiple events/installations/opponents, continued actions, declaration
  binding checks, spell-created target watch, end-combat priority, pending expiry,
  cleanup and absent source all have explicit outcomes.
- **Preservation → existing compiler suite.** Ordinary pair triggers, grouped
  combat and host attack/block unions, delayed death/unblocked/step bodies, and
  accepted collection records retain their tests. Prior Flotilla rejection
  assertions now require acceptance; collection counts are updated.

Estimated and actual gains are **1 unique name / 1 record**. Complete-card RON
combat/payment coverage gains **Goblin Flotilla**. Synthetic engine evidence
adds no inventory coverage. Compilation progress remains separate from gameplay
coverage; the complete collection still has unsupported behavior.

### Validation and loader evidence

Required checks ran in fast-fail order. Focused TestRONCombatPairDelay and
TestCombatPairDelay suites passed with -count=1. Preservation checks for the
complete Flotilla record, combat-start consumers and landwalk also passed.
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1` and
`make test` passed. The implement-engine skill's
`go test ./... -short -count=1` passed. `make lint` passed with **0 issues**;
`git diff --check` passed. Automatic formatting and all final diffs were reviewed;
no unrelated change was introduced. The complete RON compiler suite and focused
engine pair/death/unblocked/bounded-step preservation checks passed again after
lint. There is no build, test, vet, lint or environment blocker for this slice.

Final `go run ./cmd/loadcards -json` exited **1**, as required while unsupported
records remain: **2,241 compiled / 204 rejected records**, **873 accepted / 113
rejected names**. Comparison by card/file/line removes exactly Goblin Flotilla
at FEM.lines.ron:34, adds no rejection and changes no remaining diagnostic.
All previously accepted records remain accepted. Collection publication is
still atomic and unavailable until remaining records are supported.

Changed files: engine combat_pair_delay.go and tests, combat_pair_matching.go,
composable_conditions.go, composable_events.go, trigger_system.go, game.go,
turn.go; compiler combat_pair_delay.go and tests, combat_pair.go,
departure_delay.go; updated begin_combat_test.go, host_combat_test.go,
text_landwalk_test.go, inventory_test.go, definition_plan_test.go,
numeric_consumers_test.go; doc.go, docs/dsl-reference.md,
docs/engine-rules-contracts.md and this ledger.

No RON source changed, so the vocabulary snapshot and source hashes stay valid.
No Go definition, registry cutover, generator or broad DSL function was removed;
the design's cutover/removal gates remain unmet. Stop after this one slice.

### Ranked next candidates

1. **Variable discard result and saved-count return — Recall.** One estimated
   full-card unlock. Reassess shared discard-result consumers, then cover actual
   outcomes, replacements, partial discard, X and resolving graveyard choices.
2. **Delayed damage-event capture — Glyph of Life.** One estimated full-card
   unlock. Its full record needs a saved recipient and filtered creature damage
   source with actual amount in the created delayed body. Reuse the captured
   frame/event fact path; inspect prevention, replacements and saved controller.
3. **Granted targeted activation prerequisite — Equinox.** Zero immediate
   unlocks while destruction prediction is missing. Audit other grant consumers
   and select shared multi-card value before a single-name prediction service.

Complete RON gameplay, runtime loading, Go-card deletion, obsolete generation
removal and unused DSL cleanup remain incomplete.

## 2026-10-14 — DISCARD-COUNT-GRAVEYARD-HAND

### Candidate comparison and selection

The working tree was clean. Fresh `make build` and the complete RON compiler
suite passed. The baseline loader exited 1 with 2,241 compiled / 204 rejected
records and 873 accepted / 113 rejected names. This was atomic collection
rejection, not a build or test failure. No unrelated change was present.

The latest ranked candidates were inspected first. Candidate 1 had one immediate
unlock, so all three were compared before selection. Complete records were read
for Recall, Glyph of Life, Equinox, Bazaar of Baghdad, Mindstab Thrull, Mind Bomb,
Life Matrix and Zombie Master. Reprints count once.

| Candidate | Distinct affected names and full-card estimate | Remaining blockers | Downstream value | Implementation/test effort |
| --- | --- | --- | --- | --- |
| Variable discard result and saved-count hand return | Recall: 1 name / 1 record. Bazaar of Baghdad and Mindstab Thrull already use supported fixed discards; Mind Bomb already has per-player counts. They gain no new full-card unlock. | None in Recall: X payment and resolving spell exile already work. Count capture and required plural graveyard choice need engine contracts. | Shared required discard-batch outcomes and exact plural graveyard return; existing fixed/card-result/per-player callers remain compatible. | Medium: one count option, one resolving return service, scoped result lowering and identity/replacement/copy/clone tests. |
| Delayed damage-event capture | Glyph of Life: at most 1 name / 1 record after a source correction; 0 correct full-card unlocks from an adapter alone. | RON filters the damage source as a Wall creature. Local Oracle requires an attacking creature. Needs exact watched recipient, actual damage capture, duration and controller contracts. | Shared damage-event frames in created delayed bodies; concrete current consumer is Glyph of Life. | High relative to this slice: delayed damage subscription, actual damage/prevention/replacement timing and source correction. |
| Granted targeted activation prerequisite | Equinox: 0 immediate unlocks. Life Matrix and Zombie Master are the other unique activated-grant consumers and already compile with untargeted grants. | Equinox still needs WouldDestroyAPermanent prediction after Targeted lowering. | Target declaration isolation in acquired abilities; no additional current unsupported targeted-grant record was found. | Medium for the prerequisite, plus a separate high-effort destruction prediction contract and interactions. |

Recall wins on a credible complete-card gain with existing dependencies. The
inspected shared grant work has no immediate additional multi-card gain, and
Glyph of Life cannot be implemented faithfully without its source audit. This
run completes only the cohesive discard-outcome-to-hand-return slice.

### Contract, signatures, contexts and reuse

Engine additions:

- `DiscardCountInto(binding *NumberBinding) DiscardCardResultOption` on
  `Discard(player PlayerRef, amount ResolutionValue, options ...DiscardCardResultOption)`.
- `ReturnChosenGraveyardCardsToHand(graveyard PlayerRef, amount ResolutionValue,
  filter CardFilter) ResolvingEffect`.

The count option requires one nonnil binding and rejects mixed count/card-result
options. A required choice reads the amount once, caps it to the hand, selects
distinct original visits and uses the shared mandatory fallback for invalid or
incomplete selections. `snapshotHandGroup`, `chooseHandDiscards` and
`discardActionsTogetherWithEvents` supply identity, ordering and replacements.
Completed library destinations count; cancellation and returned visits do not.
Nonpositive amounts and missing players clear the count. Existing fixed-discard,
card-result and per-player result callers retain their paths.

The return service saves matching current graveyard visits at resolution. The
effect controller selects distinct cards before movement. Short graveyards
supply available cards; invalid choices use the first remaining candidate.
New cards cannot join the selection. Departure and return cannot supply a saved
visit. The existing graveyard movement service gains an internal destination
callback: all selected cards enter their owners' hands before zone observations,
then one EvtCardsLeftGraveyard reports the actual batch count. Existing public
MoveCardsFromGraveyard callers retain their prior event timing.

RON accepts `DiscardNumberCards(Integer(n))` for nonnegative n and
`DiscardNumberCards(ValueX)` with a bound X. A direct DiscardACard or
DiscardNumberCards instruction can produce the symbol NumCardsDiscardedThisWay
in its remaining lexical action list. A later direct producer supplies a new
binding. Branch-local producers cannot escape; existing delayed bodies can
capture the result. `ReturnNumberCardsFromPlayersGraveyardToHand(
NumCardsDiscardedThisWay, AnyCard, player)` accepts existing singular player
references. Other return amounts/filters and other producer shapes need adapters.
Unbound values, malformed declarations, incompatible players and out-of-scope
reads reject before factory use. Factories allocate fresh typed result bindings.
Shared number frames retain isolation through resolutions, copies, clones and
delayed captures. No global last-action slot or game-time declaration panic exists.

RON owns syntax, roles, source diagnostics and lexical plans. The engine owns
choices, exact visits, actual results, replacement batches and movement. X mana
locking, spell-copy controller/identity and resolving spell exile are reused.
CR 701.9a–c, 608.2c–d, 609.3 and 400.7 supply the rules contracts. This shape
has no new battlefield LKI/phasing subject, target-announcement choice, cost
rollback or replacement-order subsystem. Existing payment and replacement
services retain those responsibilities.

### Failing-first evidence and card-to-test mapping

Engine tests first failed compilation for missing DiscardCountInto and
ReturnChosenGraveyardCardsToHand. Fixture target/filter API calls were corrected,
then the failure was confirmed to contain only the new missing signatures.
Complete-record RON tests failed at LEG.lines.ron:213:183 with
`unsupported fixed discard amount (ValueX)`; the synthetic count collection
also rejected. Implementation then supplied the contracts and scoped adapter.
Later fixture corrections used ActivePlayer for a direct engine application
without target placement, and resolved each sorcery before casting the next
factory instance. Native copies provide pending interleaving evidence. No rule
or assertion was removed to hide an unsupported behavior.

- **Recall → TestRONDiscardCountReturnRecall.** Complete unchanged LEG record;
  native XXU payment, zero X, short hand, resolving choices, changed copy
  controller, clone isolation and final exile. Mana is fully consumed as expected.
- **Recall → TestRONDiscardCountReturnInstancesAndDelayed.** Distinct factories
  and player hands/counts; a synthetic activation captures an actual discard
  result in a delayed body with separate cloned-game execution.
- **Recall → TestRONDiscardCountReturnCompleteLoad.** Complete raw LEG record
  loads into an explicit registry and creates the card with no fallback.
- **Recall → TestRONSpellExileRecall.** Preservation test now requires complete
  record acceptance before retaining the prior isolated exile assertion.
- **Compiler → TestRONDiscardCountReturnInvalid and StrictLoad.** Unbound and
  malformed count reads, incompatible filters/players, negative fixed amounts,
  missing arguments, branch escape, independent producers and atomic registry
  rejection have explicit assertions. The retained fixed-discard invalid suite
  rejects unbound ValueX in its original activation context.
- **Engine → TestDiscardCountReturnOutcomes.** Actual counts for ordinary,
  library-replaced, cancelled and returned-visit discards; zero/short operations
  and discard events observing the entire batch.
- **Engine → TestDiscardCountReturnChoicesAndVisits.** Exact graveyard visits,
  return during choice, newly added cards, distinct required choices, controller
  selection, owner destinations, short/zero operations and full destination
  visibility in the single graveyard departure observation.
- **Engine → TestDiscardCountReturnDefinition, SavedFramesAndReset and
  MandatorySelection.** Invalid definitions, producer dominance, source departure,
  delayed capture, clones, zero/missing-player reset, duplicate/foreign/nil
  selections and the mandatory distinct-choice fallback.
- **Preservation → full compiler/engine/repository suites.** Bazaar of Baghdad,
  Mindstab Thrull, Mind Bomb and prior accepted records retain their paths and
  tests. Inventory and definition-plan count assertions are updated.

Estimated and actual gains are **1 unique name / 1 record**. Complete-card native
RON gameplay coverage gains **Recall**. Synthetic service tests add no inventory
coverage. Compilation progress is separate from gameplay coverage.

### Source defects and removal gates

Recall's RON matches local data/LEG.json Oracle text: discard X, return one card
per actual discard, then exile the spell. No source correction was needed.
Glyph of Life at LEG.lines.ron:103 uses
WhenAPermanentDealsDamageToAPermanent(And([IsCreatureType(Wall),
IsCardtype(Creature)]), SinglePermanent(Ref_TargetPermanent)). Its local Oracle
text in data/LEG.json requires damage **by an attacking creature** to the chosen
Wall. The current source predicate is a transcription defect. A later delayed
capture slice must correct it with Oracle/rules evidence and validate the whole
record; this run leaves it explicitly blocked and unchanged.

No RON source changed, so vocabulary source hashes and snapshot remain valid.
Inventory lowering now assigns value/card-filter/player-reference roles to the
new return form. No Go definitions, broad DSL functions, generators or runtime
registries were removed: complete loading, gameplay mapping and native startup
still gate their removal. No fallback or partial production registry was added.

### Validation and loader evidence

Required checks ran in fast-fail order. `go test ./pkg/mage/dsl/ron -run
TestRONDiscardCountReturn -count=1` and engine `TestDiscardCountReturn` passed.
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1` and
`make test` passed. The implement-engine skill's
`go test ./... -short -count=1` also passed.

The first `make lint` reported one prealloc issue in the new compiler helper.
The effect slice now allocates capacity for the producer and body. Focused RON
checks, build, vet, engine short tests and make test all passed again after that
fix. Final `make lint` passed with **0 issues**. Automatic formatting and all
final diffs were reviewed; no unrelated file changed. `git diff --check` passed.
Focused discard-count, fixed-discard and Recall spell-exile preservation tests
passed again after lint. No check was weakened and no environment blocker remains.

Final `go run ./cmd/loadcards -json` exited **1**, as required while unsupported
records remain: **2,242 compiled / 203 rejected records**, **874 accepted / 112
rejected names**. A keyed comparison against the fresh baseline removes exactly
Recall at LEG.lines.ron:213, adds no rejection and changes no remaining diagnostic.
All previously accepted records remain accepted. Collection publication remains
atomic and unavailable until the remaining records are supported.

Changed files: engine composable_actions.go, chosen_graveyard_hand.go, game.go
and discard_count_return_test.go; compiler discard_count_return.go and tests,
compiler.go, optional_payment.go, player_scope.go, sacrifice_value.go and
inventory.go; preservation expectations in definition_plan_test.go,
inventory_test.go, numeric_consumers_test.go and spell_exile_test.go;
pkg/mage/doc.go, docs/dsl-reference.md, docs/engine-rules-contracts.md and this
ledger. The top status and candidate list are refreshed; historical entries
remain unchanged. No RON source changed.

### Ranked next candidates

1. **Delayed damage-event capture — Glyph of Life.** Reuse captured frames and
   actual damage events. Correct the source predicate to attacking creatures
   with the local Oracle evidence above. Estimate one full-card unlock; cover
   exact recipient visits, saved controller, prevention/replacements and expiry.
2. **Granted targeted activation prerequisite — Equinox.** Zero immediate
   unlocks until destruction prediction exists. Other activated-grant consumers
   are already supported; reassess shared value before selecting prediction.
3. **Player next-main delayed event and saved spell value — Mana Drain.**
   Its complete LEG record has CounterSpell, the player's next-main event and
   AddManaRepeated with ManaValueOfSpell. The value currently has a resolution
   payment adapter, not a general resolving-value adapter. Event lowering alone
   gives zero immediate unlocks; reassess a cohesive delayed-value slice before
   claiming a full-card gain. Keep saved departure facts separate from targets.

Complete RON gameplay, runtime loading, Go-card deletion, obsolete generation
removal and unused DSL cleanup remain incomplete. Stop after this slice.

## 2026-10-14 — DELAYED-DAMAGE-CAPTURE

### Fresh baseline and candidate selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The fresh loader exited 1 for
collection rejection, with **2,242 compiled / 203 rejected records**, and
**874 accepted / 112 rejected names**. This was not a build or test failure.
Complete records at LEG lines 77, 103 and 169 and their nested forms were read;
source requirements came from RON and local Oracle data, not Go definitions.
Targeted vocabulary entries confirmed one occurrence of the damage event,
Trigger_ThatMuch and player next-main event. Reprints count once.

| Candidate | Distinct affected cards | Estimated full-card gain | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Delayed damage-event capture | Glyph of Life | 1 name / 1 record | Missing exact-recipient delayed subscription and amount capture; source predicate defect. Adds reusable permanent-damage subscription and captured-number integration, including nested delays and waiting source queries. No further blocker in the full record. | Medium: one engine mechanism, one adapter, identity/event/filter tests and source correction. |
| Granted targeted activation prerequisite | Equinox | 0 names / 0 records | Targeted spell grant needs an adapter; WouldDestroyAPermanent also needs destruction prediction. Other activated-grant consumers are already supported. Enables Equinox's later prediction work, with no demonstrated additional inventory unlock. | Medium for grant alone; high for complete prediction and legality contract. |
| Player next-main delay and saved spell value | Mana Drain | 0 for event alone; potentially 1 name / 1 record for complete slice | CounterSpell already lowers. Next-main event and general resolving ManaValueOfSpell adapter remain missing. Saved mana value must survive countering or target departure; existing payment-value support is not general resolution support. Enables delayed colorless mana, but neither individual adapter unlocks the full card. | Medium to high: player timing plus saved spell facts and mana-value adapter. |

Candidate #1 is dependency-ready after one bounded engine prerequisite. The
three compared candidates each affect only one unique inventory card; no
multi-card gain is claimed. Equinox's adapter alone has zero immediate gain,
and Mana Drain needs two distinct missing contracts. Glyph gives the largest
credible complete gain for this bounded run. The reusable service is independent
of card names. Larger shared remaining families require separate subsystem
work; this run does not add incomplete adapters for them.

Before editing, the stated contract covered exact watched visits, actual damage
amounts, saved creator control, repeated events until cleanup, prevention,
redirection, simultaneous packets, LKI, phasing, copies and clones. The engine
owns those mechanics; RON owns syntax, scopes, validation and fresh bindings.
The implement-engine skill supplied the engine-contract workflow and additional
whole-repository short test requirement. User-authorized RON source correction
and wiring remain part of this vertical slice.

### Exact supported signatures, contexts and dependencies

Engine API:
`WhenReceivesDamageThisTurn(PermanentRef, PermanentFilter, *NumberBinding,
ResolvingEffect) ResolvingEffect`. The subject must be present and unphased at
installation. Its exact incarnation is saved. A matching permanent damage source
and positive actual event amount produce one pending body per event until
cleanup. The creator's source and controller remain saved. Current recipient
control or characteristics cannot retarget the subscription. Departure/return
cannot supply it; a phase cycle preserves it. Queued bodies survive expiry and
departure. Each stack body receives its own saved amount inside the captured
frame. Copies and clones keep independent frames; copy controllers retain the
ordinary shared stack-copy behavior.

RON shape:
`CreateTriggerUntil(WhenAPermanentDealsDamageToAPermanent(filter,
SinglePermanent(ref)), ActionList(nonempty_actions), UntilEndOfTurn)` in ordinary
resolving bodies. Supported singular permanent refs and the existing permanent
damage-source filter adapter are reused. The inventory record uses
`And([IsAttacking, IsCardtype(Creature)])`. `Trigger_ThatMuch` is a symbol bound
only to this body's damage amount, including nested saved bodies. Unbound,
malformed, escaped and unrelated event aliases reject. Other subjects,
durations and event contexts remain unsupported.

Engine reuse: `captureDelayedBody`, exact permanent refs, `damageFilterSource`,
`DamageSourceIsAttacking`, native damage/replacement batches, pending trigger
placement, `eventAmountCapture`, saved-number maps, stack copies, game clones,
cleanup and waiting source categories. No parallel damage system, custom payment
path, global last-action state, fallback or partial production registry was
introduced. Existing cost rollback and replacement ordering remain owned by
those shared services. Inventory roles now identify the delayed damage event,
source filter and exact recipient ref. Waiting queries preserve the watched
visit and nested installation source reads without running actions.

Rules evidence: CR 120.4 and 120.8 (actual damage after prevention/replacement;
zero damage), 400.7 (new objects after zone movement), 603.7a–e (installation,
stated duration, identities and creator control), and 702.26b,d (phasing).

### Failing-first evidence and card-to-test mapping

Engine tests first failed compilation for the new missing signature. Fixture
API mistakes were corrected, then the failure was confirmed to contain only
`WhenReceivesDamageThisTurn` missing. Complete-record and synthetic RON tests
failed for the unsupported WhenAPermanentDealsDamageToAPermanent event. After
lowering and engine implementation, real Glyph tests still failed because the
old RON source predicate excluded ordinary attackers. Correction fixed those
assertions without changing the required behavior. Final review added a waiting
source test; its `live: watched visit missing` assertion failed before the query
integration was added.

- **Glyph of Life → TestRONDelayedDamageCaptureGlyph.** Complete corrected LEG
  record, native white payment and target legality; ordinary attacking creatures,
  attacking Walls and idle Walls; partial prevention, spell-copy controller,
  clone isolation, illegal target, returned visit, phasing, phase cycle and
  cleanup expiry. Gameplay goes through native casting and damage services.
- **Glyph of Life → TestRONDelayedDamageCaptureLoad.** Complete raw record loads
  into an explicit registry. Adding an invalid record rejects atomically and
  preserves the prior registry.
- **Compiler → TestRONDelayedDamageCaptureInvalid.** Wrong durations, recipient
  shapes/arity, empty bodies, malformed and unbound aliases, incompatible
  references, malformed filters and scope escape reject before gameplay.
- **Frames → TestRONDelayedDamageCaptureInstancesAndNested.** Separate factory
  instances and controllers, multiple pending event amounts, nested future
  triggers, source/recipient departure and independent clone execution.
  Synthetic cards add no inventory coverage.
- **Engine → TestDelayedDamageCaptureFrames and Identity.** Creation timing,
  repeated actual amounts, partial/full prevention, creator control, source LKI,
  nonmatching sources, exact recipient visits, phasing, returned visits,
  pending cleanup, departed subjects, stack copies and cloned resolution.
- **Engine → TestDelayedDamageCaptureDefinition.** Nil or unbound subjects,
  missing/unnamed amount bindings, absent bodies, producer dominance and
  absent-subject installation.
- **Engine → TestDelayedDamageCaptureSimultaneousAndRedirection.** Independent
  simultaneous packet amounts, unrelated recipients, clone frames, and actual
  damage to the final redirected recipient after prevention.
- **Engine → TestDelayedDamageCaptureWaitingSources.** Exact watched visit
  through phasing, departure and return, expiry and clone isolation; nested
  installation source reads remain visible without executing the body.
- **Preservation → compiler, engine and whole-repository suites.** Existing
  accepted records and their paths remain supported. Only measured inventory
  and definition-plan count assertions change.

Estimated and actual gains are **1 unique name / 1 record**. Complete-card RON
native gameplay coverage gains **Glyph of Life**. Compilation and gameplay
coverage remain separate; synthetic tests do not increase inventory coverage.

### Source correction, documentation and removal gates

`cards/mtgish/LEG.lines.ron:103` previously selected damaging sources with
`And([IsCreatureType(Wall), IsCardtype(Creature)])`. Local `data/LEG.json`
Oracle text requires damage by an attacking creature. The source filter is now
`And([IsAttacking, IsCardtype(Creature)])`; the chosen target still requires a
Wall creature. There is one record and no identical reprint to correct.
Idle-Wall and ordinary-attacker tests distinguish the corrected semantics.
No global predicate was inverted.

The vocabulary snapshot was refreshed from parser AST walks: source hashes,
changed occurrences and positions, and current loader diagnostics. API and
rules-contract docs now describe only this supported behavior. Go production
card definitions, legacy DSL APIs and generation remain untouched because the
complete-loading, gameplay and runtime cutover gates are still unmet. No cleanup
caller audit or removal is claimed for this slice.

### Validation and loader evidence

Final checks ran in fast-fail order after the waiting-source integration and
nested-frame tests: `go test ./pkg/mage/dsl/ron -run
TestRONDelayedDamageCapture -count=1`, engine `TestDelayedDamageCapture`,
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1` and
`make test` all passed. The skill's additional
`go test ./... -short -count=1` passed. Final `make lint` passed with **0 issues**.
`git diff --check` passed. Formatting changes were limited to the new files'
import layout; all diffs were reviewed. Focused RON and engine tests passed again
after lint. No unrelated file changed.

The first lint attempt reported a test fixture type error while the waiting
source test was being added: PhaseIn needs a UUID slice. That fixture was fixed;
the expected waiting-source assertion failure was then confirmed before adding
the integration. All required checks were rerun on the final implementation.
The initial copy fixture also assumed a different stack-peek ordering; its
expected copied amount now follows the actual selected stack object. The
separate pending amounts, changed-controller copy and clone assertions remain.
No gameplay requirement or test was removed or weakened. No environment or
pre-existing build/test blocker remains.

Final `go run ./cmd/loadcards -json` exited **1**, as required while unsupported
records remain: **2,243 compiled / 202 rejected records**, **875 accepted / 111
rejected names**. A keyed comparison removes exactly Glyph of Life at LEG line
103, adds no rejection and changes no remaining diagnostic. The final report
matches the refreshed vocabulary snapshot. All previously accepted records
remain accepted. Collection publication remains atomic and unavailable until
the remaining records are supported.

Changed files: new engine delayed_damage_capture.go and tests; game.go,
trigger_system.go, composable_events.go and waiting_delayed_source.go; new RON
delayed_damage_capture.go and tests; departure_delay.go, sacrifice_value.go,
inventory.go, definition_plan_test.go, inventory_test.go and
numeric_consumers_test.go; Glyph's LEG.lines.ron record; pkg/mage/doc.go,
docs/dsl-reference.md, docs/engine-rules-contracts.md, vocabulary snapshot and
this ledger. The current summary and ranked candidates are updated. Earlier
entries remain unchanged.

### Ranked next candidates

1. **Player next-main event and saved spell mana value — Mana Drain.** Estimate
   one complete-card unlock only after both contracts are established. Save
   departure facts separately from the target; test counter failure and exact
   player main-phase timing. An event-only adapter has zero full-card gain.
2. **Granted targeted activation prerequisite — Equinox.** Zero immediate
   unlocks until destruction prediction. Reassess whether the grant alone has
   sufficient shared value before selecting it.
3. **Shared conditional static-rule bodies — Camel and Guardian Beast.** The
   complete ARN records show static conditional prevention versus noncreature
   artifact protection. Camel also needs band membership filtering; Guardian
   Beast needs enchant/control protection with current untapped status. Compare
   supported engine services and nested blockers before estimating full gains.

Complete RON gameplay, runtime loading, Go-card deletion, obsolete generation
removal and unused DSL cleanup remain incomplete. Stop after this slice.


## 2026-10-14 — NEXT-MAIN-PHASE-MANA

### Fresh baseline and selection

The working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. Fresh
`go run ./cmd/loadcards -json` exited 1 for collection rejection:
2,243 compiled / 202 rejected records; 875 accepted / 111 rejected names.
This was not a build, test or environment failure. The complete execution design,
current status, latest ledger entry, relevant API/rules contracts, engine event,
stack value and delayed-capture code, and complete candidate RON records were
read. CR 202.3e, 505.1–2, 603.7 and 608.2b–h establish the relevant contracts.
Local data/LEG.json Oracle text agrees with Mana Drain's full RON record.

Candidate comparison, counting each name once:

| Candidate | Distinct affected names | Estimated complete-card gains | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Ranked #1: next-main-phase delayed mana | Mana Drain | 1 name / 1 record | Existing stack-ID mana-value LKI, delayed captures, countering and fixed-color mana production cover prerequisites. Adds a shared either-main-phase pattern and typed spell-value production. Event-only work would gain zero. | Small engine pattern, two compiler adapters, shared value lowering and timing/identity/legality tests. |
| Ranked #2: granted targeted activation | Equinox | 0 from grant alone | Full nested body needs WouldDestroyAPermanent prediction for controller-owned lands. Targeted grants can serve later ability grants; no additional complete collection card identified for this exact form. | Medium grant/target lifecycle work; large prediction prerequisite for a complete card. |
| Ranked #3: conditional static-rule bodies | Camel; Guardian Beast | 0 from shared dispatch alone; at most 2 after distinct prerequisites | Camel needs Desert-source damage prevention, attacking status and band membership. Guardian Beast needs untapped conditional noncreature artifact membership, Aura attachment protection without detaching existing Auras, control protection and indestructibility. Existing conditional layers and attributes cover portions, not both complete bodies. | Large: mixed rule/replacement dispatch plus band and protection contracts; unsuitable as one small complete slice here. |

Ranked #1 was dependency-ready and selected. Its complete gain is credible for
small effort; the apparent two-card candidate does not offer two immediate
unlocks. No card-name branches, fallback factories or new scheduling subsystem
are needed. The engine implement-engine skill supplied the rules-contract and
engine TDD workflow; RON lowering remained in ron. No card skills were invoked.

### Supported signatures, dependencies and reuse boundary

- Engine: `YourMainPhase` implements existing `TurnStepPattern` and
  `DelayedStepPattern` via `stepPattern{EvtMainPhase, true}`.
  `AtNext(YourMainPhase, body)` consumes the next actual own main-phase event,
  regardless of its precombat flag. Skipped phases do not emit events; extra
  phases can satisfy it. Cleanup preserves this one-shot schedule. Creation is
  during resolution, detection queues for priority, and later events do not
  repeat it. The existing engine owns source/controller capture, pending
  placement, phasing, copies, clones and schedule lifetime.
- RON: `CreateFutureTrigger(AtTheBeginningOfPlayersNextMainPhase(You),
  ActionList(nonempty-actions))` in existing resolving action contexts.
  The exact bare You argument is required. Other players, targeted delayed
  bodies and empty lists remain rejected. Scope and spelling stay in ron.
- RON: `AddManaRepeated(single-fixed-color-production,
  ManaValueOfSpell(bound-singular-spell-reference))` in ordinary resolving
  bodies, including delayed bodies. Existing typed targets and spell-event
  bindings supply the reference. Invalid or unbound references, choice of
  color, multiple productions, entry and draw-replacement contexts reject.
- `compileSpellManaValue(Node, scope) (valueFactory, error)` shares exact
  reference validation and `ManaValueOfSpellWithLKI` lowering with existing
  resolution PayManaX. In the collection, In the Eye of Chaos and Invoke
  Prejudice use that existing payment form; their complete records and tests
  remain supported. Reflecting Mirror's complete record was inspected but its
  announcement XIs form remains unsupported and unchanged.
- The engine stack service retains mana value separately from live membership,
  keyed by the exact stack ID. Announced X counts. Recasting the same source
  card cannot supply the saved visit. Live reads use current values; departed
  reads use departure facts. No global last-action state was added. Delayed
  target frames retain identity without making new targets at placement.
- Counter failure on a legal uncounterable spell still creates the delayed
  instruction. An illegal sole target prevents Mana Drain from resolving.
  Shared counter, mana replacement and payment transactions retain their
  contracts. No new simultaneous-operation or replacement-order policy is
  required by this slice.

Before editing, acceptance cases were stated: creation timing, either actual
own main phase, counter failure, all-illegal targets, X, return/recast, copies,
clones, cleanup persistence and invalid declarations. No Go definitions, RON
records or production registry paths were changed. Removal gates remain unmet.

### Failing-first evidence and card-to-test mapping

Engine tests first failed compilation for missing YourMainPhase. An initial
fixture used the wrong phase enum type; after correction, the confirmed failure
was only the missing new signature. The engine pattern then passed its tests.
Complete Mana Drain gameplay/load tests failed on unsupported
AtTheBeginningOfPlayersNextMainPhase. After that adapter, the same tests failed
on unsupported ManaValueOfSpell in mana amounts. The shared value adapter
resolved that second failure. Initial fixture API errors and an extra RON
parenthesis were fixed; no behavior assertion or requirement was weakened.

- **Mana Drain → TestRONNextMainPhaseManaDrain.** Complete LEG RON record,
  paid native casting and failed-payment state preservation; two-X target cost,
  zero value, both main event flags, wrong-player events, delayed creation,
  counter success, uncounterable target, all-illegal target, recast with a
  different X, original spell copy with a different controller, delayed trigger
  copy with a different controller, countered delayed trigger, colorless mana,
  cleanup survival and independent game clone resolution.
- **Mana Drain → TestRONNextMainPhaseSeparateSpellValues.** Two complete RON
  instances retain different target spell IDs and values, create independent
  schedules at one boundary and resolve separately in both original and clone.
- **Mana Drain → TestRONNextMainPhaseLoad.** Complete raw record loads into an
  explicit registry. Fresh factories keep separate target declarations. Adding
  an invalid record rejects atomically and preserves the previous registry.
- **Compiler → TestRONNextMainPhaseInvalid, InvalidBoundAmounts, Inventory.**
  Wrong arity/kind/player, empty bodies, unbound and incompatible spell reads,
  malformed bound references, unsupported production shapes and descendant
  semantic roles. The latter two names share the TestRONNextMainPhase prefix.
- **Frames → TestRONNextMainPhaseSpellBindings.** Immediate and delayed colored
  mana from departed spell-event bindings; fresh factories, distinct
  controllers and captured event frames. Synthetic cards add no inventory gain.
- **Engine → TestNextMainPhaseTiming, PendingAndController, SkippedAndExtra,
  Definitions.** Native precombat/postcombat boundaries, prior events,
  opponent turns, cleanup, one-shot consumption, cloning, detection/placement
  timing, phased source, trigger-copy control, actual skipped and extra phases,
  absent bodies and unbound reads. All share the TestNextMainPhase prefix.
- **Preservation → TestSpellManaValue*, TestRONResolutionMana*,
  TestRONWaitingDelayedMana*, compiler, engine and whole-repository suites.**
  Existing departure LKI, mana payments and delayed mana paths keep working.

Estimated and actual gains match: **1 unique name / 1 record**. Complete-card
RON gameplay coverage gains **Mana Drain**. Compile acceptance and gameplay
coverage remain separate. Synthetic timing/binding cases do not count as
additional inventory cards.

### Documentation, loader and validation

API and rules-contract documentation describe only the behavior established by
this slice. The semantic inventory marks You under the new event as a player
reference. Collection and definition-plan count assertions advance by one.
No RON source defect was found or corrected for this slice. Vocabulary syntax
and source hashes are unchanged; only its loader_baseline is refreshed to
remove Mana Drain and record 2,244 compiled records.

Final review found a required CR 609.7a interaction: captured spell targets,
unlike existing spell-event bindings, were absent from waiting source queries.
A new complete Mana Drain assertion failed with an empty source list. The new
engine TestNextMainPhaseSavedSpellTargets also failed on missing target facts.
Delayed frames now retain target spell source facts by exact stack ID, separately
from target membership. Existing WaitingDelayedSpellSources, filtered source
choice and StackBoundSpellSources expose the facts through waiting and pending
lifetimes. Tests cover live/current and departure color/control, recasts,
deduplication, defensive query results, clone isolation and source choice without
running the mana body. This is a necessary integration within the selected slice.

Changed files: engine composable_events.go, composable_conditions.go,
waiting_delayed_spell.go and new next_main_phase_test.go; RON delayed_step.go,
mana_production.go, spell_scope.go, optional_payment.go, inventory.go and new
next_main_phase_test.go; collection-count assertions in inventory_test.go,
definition_plan_test.go and numeric_consumers_test.go; pkg/mage/doc.go,
docs/dsl-reference.md, docs/engine-rules-contracts.md, the vocabulary snapshot
and this ledger. Earlier entries remain unchanged.

Final verification ran in fast-fail order on the completed waiting-source
integration: `go test ./pkg/mage/dsl/ron -run TestRONNextMainPhase -count=1`,
engine `go test ./pkg/mage -run TestNextMainPhase -count=1`, `make build`,
`go vet ./...`, `go test ./pkg/mage/... -short -count=1` and `make test` all
passed. The skill's additional `go test ./... -short -count=1` passed.
`make lint` passed with **0 issues**. Automatic formatting/fix output was reviewed;
there were no further code changes. Focused RON and engine tests passed again
after lint. `git diff --check` passed. No environment or pre-existing validation
blocker remains. All required checks were repeated after the final integration;
earlier passing runs are not used to claim validation of later code.

Final `go run ./cmd/loadcards -json` exited **1**, as required while unsupported
records remain: **2,244 compiled / 201 rejected records**, **876 accepted / 110
rejected unique names**, out of 2,445 records / 986 names. Exact diagnostic
comparison against the fresh baseline removes only Mana Drain at LEG line 169,
adds no issue and changes no remaining diagnostic. The final report equals the
vocabulary snapshot's loader baseline. Accepted records remain accepted. Strict
collection publication remains atomic and unavailable until all remaining
records are supported.

### Ranked next candidates

1. **Shared conditional static-rule bodies — Camel and Guardian Beast.**
   Two concrete downstream names; zero immediate full unlock from dispatch
   alone. Reassess mixed rule/replacement bodies, band membership and current
   untapped enchant/control protection. Do not count shared first errors as
   complete-card gains.
2. **Granted targeted activation prerequisite — Equinox.** Zero full unlocks
   until destruction prediction. Prefer this only with concrete additional
   grant users or a dependency-ready complete prediction contract.
3. **Entry choice state — Primal Clay and Shapeshifter.** Complete ATQ records
   show finite copiable variants versus saved numeric entry/upkeep choices.
   Their shared choice infrastructure may help both, but nested rules differ.
   Verify Primal Clay's RON Wall variant against Oracle before implementation;
   no source correction or full-card unlock estimate is claimed here.

Complete RON gameplay, native runtime loading, Go-card deletion, obsolete
Go-set generation removal and unused DSL cleanup remain incomplete. No DSL
function was removed; no cleanup caller audit is claimed. Stop after this slice.


## 2026-10-14 — CONDITIONAL-STATIC-BODY

### Candidate comparison and selection

The worktree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed before editing. The baseline loader
exited 1 for collection rejection: 2,244 compiled / 201 rejected records,
876 accepted / 110 rejected unique names. This was not a build or test failure.
Complete Camel, Guardian Beast, Equinox, Primal Clay (ATQ and 3ED) and
Shapeshifter records were inspected, including nested forms. Targeted vocabulary
entries established If, BandedWith, both artifact protections and both entry
choice forms. Reprints count once.

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Conditional static-body composition | Camel; Guardian Beast | 0 names / 0 records | Camel needs collection prevention and band membership. Guardian Beast needs enchant/control protection and noncreature artifact membership. Shared composition removes the body boundary for both. | Small: compiler composition, one live prevention engine contract and focused lifecycle tests. |
| Granted targeted activation | Equinox | 0 | Complete record also needs WouldDestroyAPermanent prediction, including replacement effects and controlled-land selection. No additional complete-record grant user was established in this comparison. | Medium to large: grant identity/target scope, then a separate prediction contract. |
| Entry choice state | Primal Clay; Shapeshifter | 0 for one common prerequisite; individual completion estimates deferred | Clay needs finite copiable P/T/type/keyword variants; its Wall P/T needs an Oracle audit. Shapeshifter needs an entry number, optional upkeep update, chosen-number storage and numeric layers. These are different downstream contracts. | Large for both; at least two bounded slices and a source audit. |

Candidate #1 was dependency-ready and has concrete two-card downstream value.
It was selected as a prerequisite, without claiming complete-card unlocks.
Selection, reuse boundary and acceptance cases were stated before editing.
No source defect was corrected. The Primal Clay concern remains unverified and
was not used to invent implementation requirements. No Go definitions or RON
source records changed. No obsolete DSL function was removed; no cleanup caller
audit is claimed. Runtime and deletion gates remain unmet.

### Supported shapes, dependencies and engine contract

Top-level permanent static `If(condition, rules)`, `Unless(condition, rules)`
and `IfElse(condition, trueRules, falseRules)` require nonempty branch lists.
They compose the already supported conditional PermanentRuleEffect attack,
untap and spell-target forms, EachPlayerEffect untap limits,
ReplaceWouldDealDamage redirection, and conditional layer declarations.
Each family keeps its exact prior subjects and operation shape. Layer parts
are validated together and retain the existing same-layer restriction and
shared type-layer membership service. Bare keywords, resolving actions,
triggers, targets, payments, unsupported collection rule operations and
unsupported protection forms still reject. Both branches compile before a
factory is available. No declaration is silently dropped.

The added prevention shape is exactly
`PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(filter,
Permanent(ThisPermanent)), [PreventThatDamage])` in a conditional static body.
Combat-pair source filters and other prevention events remain unsupported here.
No target, event or lexical result binding is introduced. All factory calls
build separate abilities and registration state.

Engine signature:
`PreventDamageFromMatchingPermanentsIf(subject PermanentRef,
from PermanentFilter, condition StateCondition) StaticRule`.
The condition and subject must pass definition validation. Zero source filters
retain the existing any-permanent meaning. The reusable engine service also
supports AttachedPermanent; the new RON prevention shape is source-only.
The shared static replacement cycle installs the declaration even when the
condition is false. Each match and waiting-source query checks the current
source controller and exact incarnation. Earlier replacements can change the
condition before rechecks. False conditions supply no waiting subject.
Source departure or phasing stops prevention; phase-in restores it; a returned
source cannot restore an old cycle. Existing source filter LKI, unpreventable
damage, simultaneous packet scope, attachment identity and clone services are
reused. No new payment, replacement ordering or batch system was added.
Rules read: CR 604, 611.3a–c, 615 and the established 616 replacement contract.
API and rules documentation describe only this supported boundary.

### Failing-first evidence and card-to-test mappings

Before implementation, engine tests failed compilation only for the missing
PreventDamageFromMatchingPermanentsIf signature. RON If, Unless and IfElse mixed
body tests failed on unsupported conditional PermanentRuleEffect bodies.
After implementation, the same behavior requirements pass. A test fixture's
Unless clone expectation was corrected before the first passing run; no engine
requirement or assertion was weakened.

- Engine: **TestConditionalStaticBodyPrevention** covers true/false conditions,
  phasing, phase return, ability removal, clones, source return, unpreventable
  damage and earlier replacements that tap or untap before prevention rechecks.
- Engine: **TestConditionalStaticBodyDefinitions** rejects missing subjects,
  missing conditions and unbound condition reads.
- Engine: **TestConditionalStaticBodyWaitingAndIdentity** checks live condition
  changes without a new cycle, Aura reattachment, phase cycles and saved
  replacement identity after source return.
- Engine: **TestConditionalStaticBodyControllerAndBatch** checks current source
  control, matching permanent departure facts, independent simultaneous player
  and permanent packets, and restarted conditions under the new controller.
- RON: **TestRONConditionalStaticBodyGameplay** uses complete synthetic RON
  definitions and native paid casting. One body combines CantAttack, prevention
  and fixed P/T, tests If/Unless/IfElse directions, tap changes, attack legality
  and clone isolation.
- RON: **TestRONConditionalStaticBodyInvalid** rejects empty bodies, inactive
  branches with resolving actions, mixed layer families, out-of-scope targets
  and bare keywords.
- RON: **TestRONConditionalStaticBodyHostUntap** moves the exact formerly
  unsupported host untap/P/T shape into positive gameplay coverage. Source
  counters control both effects; native untap steps, phasing, clones and source
  departure are checked. The old invalid-body table now retains a mixed body
  with an unsupported modification instead.
- RON: **TestRONConditionalStaticBodyLoadAndInstances** proves strict loading,
  two independent factory instances, simultaneous recipient separation and
  atomic rejection that preserves an existing explicit registry.
- **Kird Ape and Sedge Troll → TestRONConditionalStaticController**;
  **Giant Tortoise → TestRONConditionalStaticUntapped**;
  **Argothian Pixies, Desert Nomads and Uncle Istvan → TestRONStaticPreventionCards
  and TestRONStaticPreventionReplacementOrder**. The new
  TestRONConditionalStaticBodyRepositoryRegression runs these complete-record
  gameplay regressions under the focused slice prefix.
- **Camel and Guardian Beast** receive shared prerequisite evidence only.
  Their complete records still reject; no full-card gameplay claim is made.
  Complete Icatian Skirmishers was read for the next-candidate dependency map:
  it shares BandedWith with Camel but uses attack-event capture and a temporary
  first-strike grant, so band formation timing must be investigated separately.

Estimated and actual gains match: **0 unique names / 0 records**. No complete
inventory-card gameplay coverage is added. Supported synthetic mixed-body
contexts gain gameplay evidence; complete existing inventory coverage is
preserved. Compilation acceptance and gameplay coverage remain separate.

### Validation and loader evidence

Focused RON and engine tests passed with `-count=1`. The first engine short
run exposed TestRONHostStaticUntapInvalid's obsolete blanket mixed-body
rejection. Its previously rejected exact shape now has the positive native
untap/P/T gameplay test above, and the negative table still checks an invalid
mixed body. No behavior test was weakened. After that test migration,
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1`,
`make test` and the skill's additional `go test ./... -short -count=1` passed.

`make lint` passed with **0 issues**. Automatic formatting and fix output was
reviewed; no additional code change remained. Focused RON and engine tests
passed again after lint. `git diff --check` passed. No environment or remaining
validation blocker exists.

Final `go run ./cmd/loadcards -json` exited **1** for expected atomic collection
rejection: **2,244 compiled / 201 rejected records**, **876 accepted / 110
rejected unique names**, out of 2,445 records / 986 names. Exact baseline
comparison changes only Camel's diagnostic, from unsupported conditional
static body (PreventDamage), ARN 12:186, to unsupported conditional static
prevention event (DamageWouldBeDealtByAPermanentToARecipient), ARN 12:200.
No record becomes accepted or rejected. All other diagnostics remain identical.
The vocabulary loader baseline equals the final report; syntax entries and
source hashes are unchanged. Complete collection publication remains gated.

Changed files: engine static_combat_prevention.go and new
conditional_static_body_test.go; compiler conditional_static.go,
static_prevention.go and new conditional_static_body_test.go; the existing
host_static_untap_test.go rejection fixture; pkg/mage/doc.go,
docs/dsl-reference.md, docs/engine-rules-contracts.md, the vocabulary loader
baseline and this ledger. No source syntax or hashes changed.

### Ranked next candidates

1. **Band-aware collection references — Camel and Icatian Skirmishers.**
   Two consumers share BandedWith. Camel also needs the
   DamageWouldBeDealtByAPermanentToARecipient collection prevention adapter.
   Skirmishers needs attack-event band timing verified before an unlock estimate.
2. **Conditional artifact protections — Guardian Beast.** Shared branch
   composition is ready. Current noncreature artifact/controller selection,
   no new Aura enchantment without removing existing Auras, and control-change
   prohibition remain concrete engine/adaptation contracts.
3. **Entry choices — Primal Clay and Shapeshifter.** Finite copiable variants
   and saved numeric entry/upkeep choices remain distinct. Audit Clay's Wall
   form before correcting all reprints or claiming full-card gains.

Complete RON gameplay, native runtime loading, Go-card deletion, obsolete
Go-set generation removal and unused DSL cleanup remain incomplete.
Stop after this slice.

## 2026-10-14 — RESOLVING-BANDED-COLLECTION

### Baseline and candidate comparison

The working tree was clean. Fresh `make build` and the complete RON package
passed before edits. Baseline `go run ./cmd/loadcards -json` exited 1 for
collection rejection, not a build failure: 2,244 compiled / 201 rejected records,
876 accepted / 110 rejected names, 2,445 records / 986 names total.

The latest ranked candidate #1 was dependency-ready. Complete Camel (ARN 12)
and Icatian Skirmishers (FEM 56) records were read, including the conditional
prevention union and attack-trigger temporary keyword grant. Targeted vocabulary
inspection finds exactly two BandedWith occurrences, one for each unique name.
Other compared records were Guardian Beast (ARN 35), Primal Clay (ATQ 55 and
3ED 197) and Shapeshifter (ATQ 61). Reprints are counted once below.

| Candidate | Distinct affected names | Estimated complete-card gains | Other blockers and downstream value | Estimated implementation/test effort |
| --- | --- | --- | --- | --- |
| Band-aware resolving collection | Camel; Icatian Skirmishers | 1 name / 1 record: Skirmishers | Camel still needs live static collection-recipient prevention. Shared typed membership also supplies contextual counts, choices and collection actions. Existing attack resolution occurs after band formation. | Small: one engine set, one compiler adapter, identity and native combat suites. |
| Conditional artifact protections | Guardian Beast | 1 name / 1 record if all contracts are supplied | Static branch composition is ready. Needs live noncreature artifact/controller membership, control prohibition, and new Aura attachment prohibition that keeps existing Auras. | Medium: multiple reusable protection contracts and attachment/control lifecycle tests. |
| Entry choice state | Primal Clay; Shapeshifter | 0 for a shared state prerequisite; up to 2 names / 3 records after separate consumers | Clay needs finite copiable variants; Shapeshifter needs saved bounded number updates and dynamic P/T. Clay's Wall source defect must be resolved. Shared choice lifecycle is concrete downstream work, but these consumers do not form one small complete slice. | Larger: entry/copy state plus separate numeric upkeep and layer contracts. |

Selected #1 for its small, complete shared contract and immediate full-card gain.
The chosen reuse boundary and acceptance cases were stated before edits.
No Go card definitions supplied requirements. Local data/FEM.json and ARN.json
Oracle text matches the selected records. CR 508.1–2, 603.3, 608.2h,
611.2c, 702.22c–g and 702.26b establish timing and identity contracts.

### Exact contract, dependencies and engine reuse

New engine API: `BandedWith(subject PermanentRef) PermanentSet`.
It selects present current band members in battlefield order. The source and
all members must match their exact combat incarnations. Unbanded attackers
supply no members; one-member bands include their subject. Missing contexts,
nil subjects and absent visits supply no members at execution. Definition
validation rejects nil and unbound subjects. Combat membership has no LKI
fallback. A returned visit cannot replace an old subject or combat participant.
Ability loss keeps the band. Combat removal, departure and phasing remove band
membership; phase-in cannot restore it. Clones read their own combat state.

RON shape: `BandedWith(permanentRef)` in ordinary resolving contextual
collections, including nested nonempty And/Or lists and existing source, host,
target, choice and event reference scopes. Entry and draw replacement contexts
reject this adapter. Static prevention and payment declarations remain separate.
The compiler uses compilePermanentRef and builds operands per factory instance.
There is no card-name branch, mutable declaration state or global action result.

The engine reuses FormAttackingBand, IsAttackingIncarnation, band removal,
PermanentRef, definition traversal, contextual set composition and saved frames.
Waiting delayed traversal now retains explicit source operands in BandedWith;
it does not read combat or execute actions. Existing AtNext, Until,
GrantEachKeyword, target legality, spell/trigger copies, phasing and clone
services keep their existing behavior. Temporary grants save recipients at
resolution. Later combat removal, source departure, control changes and a phase
cycle do not change those saved recipients; zone return cannot inherit a grant.
No payment, replacement ordering or simultaneous damage mechanism was added.

The native BandFormer path forms bands after individual attack events but before
pending attack bodies resolve. Skirmishers needs its event identity at detection
and its collection at resolution, so no attack event or band timing change is
required. The explicit ExecuteAttackers path can likewise form a band before
placing and resolving pending triggers.

### Failing-first evidence and card-to-test mappings

After correcting test setup syntax and EffectContext field names, engine tests
failed compilation only for the missing BandedWith signature. After correcting
the test's ability identity accessor, RON gameplay and strict-load tests failed
on the expected unsupported BandedWith expression in the complete Skirmishers
record. The same behavior assertions pass after implementation.

- **TestBandedSetIdentity**: band inclusion and unrelated attackers, ability loss,
  source/member return, phasing and phase cycles, combat removal, stale source
  and member combat incarnations, and clone isolation.
- **TestBandedSetDefinitions**: nil/unbound declarations, missing runtime context
  and explicit source traversal.
- **TestBandedSetWaiting**: nonattacker, unbanded attacker and one-member bands;
  repeated pure delayed queries; exact departed source retained after return;
  independent delayed expiry and no action on returned objects.
- **Icatian Skirmishers → TestRONBandedCollectionNative**: complete repository
  RON, rejected unpaid cast, paid native cast, BandFormer declaration, unrelated
  attacker exclusion, shared first-strike and normal combat damage, retained
  grants after combat removal, cleanup expiry and printed keyword retention.
- **Icatian Skirmishers → TestRONBandedCollectionPending**: no grant at detection;
  pending body survives ability removal; source/member departure and return,
  phasing, combat removal, copied triggers and independent game clones.
- **Icatian Skirmishers → TestRONBandedCollectionCapturedLifetime**: grants survive
  source departure, subject phase cycles and control changes; returned subjects
  cannot inherit them.
- **TestRONBandedCollectionTargetsAndInstances**: two factory instances and two
  independent bands with typed targets, duplicated union operands, untap actions,
  returned-target rejection and later shroud at resolution.
- **TestRONBandedCollectionInvalid**: missing, malformed, wrong-type, unbound
  target/event and unavailable host operands.
- **TestRONBandedCollectionLoad**: complete repository record strict loading,
  fresh ability instances, rejected malformed collection and registry preservation.
- **Camel**: shared engine membership prerequisite only. Its complete record
  still rejects at static collection-recipient prevention; no full-card gameplay
  claim is made.

Estimated and actual gains match: **1 unique name / 1 record**. Complete-card
RON gameplay coverage grows by Icatian Skirmishers. Synthetic contextual
collection coverage and delayed source-query coverage also grow. Compilation
acceptance remains distinct from gameplay evidence.

### Source defects and preservation

No RON source changed. Local data/ATQ.json confirms Primal Clay's Wall option is
1/6 with defender, while both complete RON records use SetCopiablePT(PT(3, 3)).
This pre-existing defect is documented for a later entry-variant slice; this run
does not correct unrelated source data. Shapeshifter's two choice sites and
numeric P/T consumer require their own complete supported contracts.

Go definitions, startup wiring and DSL exports remain intact because the
collection/runtime removal gates are not met. No DSL removal was attempted.

### Validation and final loader evidence

Focused `go test ./pkg/mage/dsl/ron -run TestRONBandedCollection -count=1` and
`go test ./pkg/mage -run TestBandedSet -count=1` passed. `make build` and
`go vet ./...` passed. The first engine short run found four obsolete collection
count assertions in definition_plan_test.go, inventory_test.go and
numeric_consumers_test.go. These now require the measured 2,245/200 result;
no gameplay assertion was removed or weakened. After the baseline updates,
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1`,
`make test` and the skill's extra `go test ./... -short -count=1` all passed.

The first `make lint` reported one preallocation issue in the new two-band test.
Both fixed-size test slices now have capacity two. Focused RON tests passed
again. The second complete `make lint` passed with **0 issues**. Automatic
formatting changes were reviewed: import grouping in the new test files, with
no unrelated changes. Focused RON and engine tests also passed after formatting.
`git diff --check` passed. There is no environment or remaining check blocker.

Final `go run ./cmd/loadcards -json` exited **1**, as expected for atomic
collection rejection: **2,245 compiled / 200 rejected records**, **877 accepted /
109 rejected unique names**, out of 2,445 records / 986 names. Exact baseline
comparison removes only Icatian Skirmishers' former unsupported BandedWith
message (FEM 56:325). Every remaining diagnostic, including Camel's unsupported
conditional static prevention event (ARN 12:200), is unchanged. No accepted
record regresses. The vocabulary loader baseline equals the final report;
syntax entries and source hashes remain unchanged.

Changed files: new engine banded_set.go and banded_set_test.go; engine
waiting_delayed_source.go; compiler contextual_collection.go and new
banded_collection_test.go; the three collection-count test files; pkg/mage/doc.go,
docs/dsl-reference.md, docs/engine-rules-contracts.md, the vocabulary loader
baseline and this ledger. No production card, startup or DSL export changed.

### Ranked next candidates

1. **Entry choice state — Primal Clay and Shapeshifter.** Compare a shared
   per-entry choice prerequisite with finite copiable variants and saved numeric
   entry/upkeep consumers. Audit the complete Clay Wall option and correct both
   reprints with Oracle evidence before any full-card claim.
2. **Conditional artifact protections — Guardian Beast.** Conditional body
   composition is ready. Current-controller noncreature artifact membership,
   control prohibition and new Aura attachment prohibition that keeps existing
   Auras remain separate reusable contracts.
3. **Static collection-recipient prevention — Camel.** The engine now has
   exact live band membership. Static lowering still needs its own collection
   context and replacement recheck contract. Compare broader multi-card
   prevention work before choosing this single-card completion.

Complete RON gameplay, runtime loading, Go-card deletion, obsolete Go-set
generation removal and unused DSL cleanup remain incomplete. Stop after this
slice.

## 2026-10-14 — COPIABLE-ENTRY-VARIANTS

### Selection and environment evidence

The working tree was clean at the start. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -short -count=1` passed. Fresh strict loading exited
1 for collection rejection, with 2,245 compiled / 200 rejected records and
877 accepted / 109 rejected names. There was no build or environment blocker.

Complete RON records were inspected for Primal Clay (ATQ and 3ED), Shapeshifter,
Guardian Beast and Camel, including their nested operations. Targeted vocabulary
queries confirmed the finite choice constructor has two occurrences, fixed
copiable P/T has six, creature-type addition has two and keyword addition has
four. Reprints count once in selection.

| Candidate | Distinct affected names | Estimated full-card unlocks | Remaining blockers and downstream value | Effort |
| --- | --- | --- | --- | --- |
| Entry choice family, finite variants first | Primal Clay, Shapeshifter | 1 name / 2 records for finite variants | Shapeshifter's entry and optional upkeep number choices need saved visit state and independent SetPower/SetToughness consumers. Immutable copiable P/T, subtype and keyword operations provide reusable entry/copy infrastructure. | Medium, one finite choice context |
| Conditional artifact protections | Guardian Beast | 1 name / 1 record | Conditional bodies and indestructible layers exist. Live controller membership, control prohibition and new Aura attachment prohibition that retains existing Auras need distinct contracts. | Large, several rule services |
| Static collection recipients | Camel | 1 name / 1 record | Exact band membership exists. Static collection scope and replacement-time recipient rechecks remain. Resolving collection lowering cannot be used as a static query. | Medium, separate static/replacement context |

The first ranked family splits into finite copiable variants and saved numeric
choices; these do not have one common choice-state contract. No compared
candidate offered more than one immediate unique-card unlock. The finite variant
slice was selected for its bounded scope and shared typed copy modifications,
which also serve later copies of the supported card. This does not claim that
finite variants implement Shapeshifter's numeric producer or payment results.
Other inspected entry consumers, Nameless Race, Wood Elemental and
Frankenstein's Monster, still require distinct payment and saved-result contracts;
they are not counted as unlocks. Complete Dance of Many and Stangg records also
retain linked-token gaps and receive no coverage claim from these modifications.

Before edits, the acceptance contract was a required entry-controller choice,
made before registration, with copiable selected characteristics; independent
factory definitions; fresh choices on later entry; retained values through
phasing; correct physical departure; and exact declaration rejection. Tests
include later copies and competing entry replacements. The engine owns these
mechanics; RON owns spelling, shape validation and factory lowering.

### Supported signatures, context and engine reuse

RON supports exactly one operation in
`AsPermanentEnters(ThisPermanent, [EntersWithLayerEffectOfChoice(options)])`.
`options` is a nonempty list of nonempty modification lists. Each list can use:

- `SetCopiablePT(PT(integer, integer))`.
- `AddCopiableCardtype(permanentType)`.
- `AddCopiableCreatureType(subtypeSymbol)`.
- `AddCopiableAbility([fixedKeywords])`, with a nonempty supported keyword list.

The adapter validates every option, including unchosen options. Host, resolving,
static-layer, empty, malformed, dynamic and unknown declarations reject.
It creates fresh abilities per factory and has no card-name dispatch.

Engine API: `ChooseCopiableVariantAsEnters(variants ...[]CopyModification) Ability`,
`CopySetPowerToughness(power, toughness int) CopyModification`,
`CopyAddSubTypes(subtypes ...string) CopyModification`, and
`CopyAddKeywords(keywords ...core.Attr) CopyModification`.
Existing `CopyAddCardTypes` supplies type addition. Fixed P/T changes base
values and keeps existing characteristic definitions; it does not implement
copy exceptions that remove those abilities.

The engine reuses `battlefieldEntryAction`, the shared replacement service,
`CopiableValues.WithModifications`, fresh definition plans,
`BuildEntryCopyWithText`, physical printed provenance and `replaceEntryDefinition`.
Stable declaration-origin/ordinal keys prevent a variant rule from applying
again after rebuilding its definition in the same entry event. Copy replacement
priority precedes variant choices; same-priority replacements keep the existing
affected-player choice. Option labels show characteristics. Preview builds make
no game choices. Invalid answers or a missing player use the first option.

Selected values become copiable under CR 707.2. Copies acquire the entry rule and
make a new choice under CR 707.5–6; copied additions remain. Counters and ordinary
layer effects do not enter the snapshot. Source control changes and phase cycles
do not repeat entry. Return starts with printed physical rules and a new visit.
Permanent spell copies retain token identity, owner and controller. There is no
new cost, damage, LKI query, APNAP policy or simultaneous-operation mechanism.
Existing batch entry makes all choices before registering any entrant.

### Failing-first evidence and card-to-test mappings

The first tests had two setup errors (HasSubtype versus HasSubType, and a player
wrapper passed to a BasePlayer helper). After correction, engine compilation
failed on only the missing variant and modification APIs. Complete-record RON
tests failed on unsupported EntersWithLayerEffectOfChoice. The same behavior
assertions pass after implementation and the evidenced Wall correction.

A later failing-first copied-spell test exposed the permanent spell-copy wrapper
path: its token did not enter. The engine now unwraps that definition when it
reads copiable provenance, as the existing entry-copy service does. That test
passes without a card-name branch or token fallback.

- **TestEntryVariantTimingIdentity**: choice before registration; controller
  rather than owner; exactly one choice; subtype/keyword retention; captured
  copiable values; exclusion of counters and ordinary layers; phase cycle,
  printed departure, fresh return and clone isolation.
- **TestEntryVariantInvalid**: empty options, nil modifications, invalid keyword
  and subtype lists, and nonpermanent source declarations.
- **TestEntryVariantPriorityAndCopy**: copy replacement first, acquired choice
  rule, no repeated replacement and retained copied flying on a new 3/3 choice.
- **TestEntryVariantInstancesAndBatch**: independent choices before batch
  registration, invalid-answer fallback, control retention and defensive option
  snapshots. A reused test declaration was corrected to build fresh abilities;
  definition validation was kept intact.
- **Primal Clay (ATQ, 3ED) → TestRONEntryVariantReal**: every complete variant,
  failed unpaid cast, paid native cast, resolution choice, exact metadata,
  flying block legality, defender attack legality, phasing, return and clones.
- **Primal Clay (ATQ, 3ED) → TestRONEntryVariantLoad**: strict reprint loading,
  independent factory abilities, malformed/context rejection and atomic registry
  preservation after a rejected collection.
- **Primal Clay → TestRONEntryVariantSpellCopy**: copied permanent spell enters
  as an owned token, chooses independently of the original, keeps flying, uses
  the shared token departure lifecycle and remains isolated in a game clone.
- **Shapeshifter, Guardian Beast, Camel**: no new complete-card coverage claim;
  their unsupported forms remain diagnostic errors.

### Source correction and preservation

Both complete Primal Clay records formerly used `SetCopiablePT(PT(3, 3))` in the
Wall option. Local authoritative `data/ATQ.json` Oracle text specifies a 1/6 Wall
with defender in addition to its other types. Both identical RON reprints now
use `SetCopiablePT(PT(1, 6))`. Existing Shapeshifter subtype and card types remain.
CR 614.1c establishes entry replacement timing; CR 614.5 prevents repeat
application; CR 707.2 and 707.5–6 establish copiable values and later entry choices.
No predicate inversion or card-specific engine behavior was introduced.

The vocabulary snapshot has fresh SHA-256 hashes for both changed files and a
fresh loader baseline. Constructor signatures, occurrence counts and source
positions are unchanged. Existing Go definitions, startup wiring and DSL exports
remain because the collection/removal gates are not met. No cleanup caller audit
or DSL deletion was warranted in this slice.

### Validation and loader evidence

Focused `go test ./pkg/mage/dsl/ron -run TestRONEntryVariant -count=1` and
`go test ./pkg/mage -run TestEntryVariant -count=1` passed. `make build`,
`go vet ./...`, `go test ./pkg/mage/... -short -count=1`, `make test` and the
skill's additional `go test ./... -short -count=1` passed after the spell-copy
fix. The four existing collection-count assertions now require the freshly
measured 2,247/198 result; gameplay assertions were not weakened.

The first `make lint` found one preallocation issue and three uses of the
deprecated core.Keyword alias in new code. The label slice now has capacity,
and the new signatures use core.Attr. Focused tests, build, vet, engine short,
full and repository short tests were repeated and passed after these changes.
The final `make lint` passed with **0 issues**. Automatic formatting changes
were reviewed: import grouping in new files, with no unrelated changes.
Post-format focused engine/RON tests and `git diff --check` passed again.

The final required `go run ./cmd/loadcards -json` exited **1** for expected
atomic collection rejection: **2,247 compiled / 198 rejected records**,
**878 accepted / 108 rejected unique names**, out of 2,445 records / 986 names.
Exact baseline comparison removes only Primal Clay's former diagnostics at
3ED 197:188 and ATQ 55:188. No diagnostic is added or changed for another card.
Every previously accepted record remains accepted. The vocabulary loader
baseline equals the final report, and all source hashes match the current files.
No environment or verification blocker remains.

Estimated and actual gains match: **one unique name / two records**. Complete
RON gameplay coverage grows by Primal Clay, including both identical reprints.
Compilation counts and gameplay coverage are separate measurements; the remaining
878 accepted names are not all claimed to have complete gameplay coverage.

Changed files: new engine entry_variant.go, copy_variant_modification.go and
entry_variant_test.go; shared entry_copy.go; new compiler entry_variant.go and
entry_variant_test.go plus entry_tapped.go; the three collection-count test files;
ATQ.lines.ron and 3ED.lines.ron; pkg/mage/doc.go, docs/dsl-reference.md,
docs/engine-rules-contracts.md, the vocabulary snapshot and this ledger.
No Go production card definition, startup path or DSL export changed.

### Ranked next candidates

1. **Saved numeric choice — Shapeshifter.** Inspect the complete entry/upkeep
   choice and its SetPower/TheChosenNumber and SetToughness/Minus consumers.
   Source-visit numeric state and optional update need their own contract;
   copiable finite variants must not substitute for that state.
2. **Conditional artifact protections — Guardian Beast.** Static branches are
   ready. Compare live noncreature artifact membership plus control and new-Aura
   prohibitions with broader rule-effect prerequisites before selection.
3. **Static collection-recipient prevention — Camel.** Exact band membership
   is ready. Static collection context and replacement-time membership rechecks
   remain. Compare broader multi-card prevention work before choosing a
   single-card completion.

Complete RON gameplay, runtime loading, Go-card deletion, obsolete Go-set
generation removal and unused DSL cleanup remain incomplete. This run stops
after the finite copiable entry-variant slice.

## 2026-10-14 — SAVED-NUMERIC-SOURCE-CHOICE

### Environment, comparison and selection

The initial working tree was clean. Fresh `make build` and
`go test ./pkg/mage/dsl/ron -count=1` passed. The initial loader exited 1 for
expected atomic collection rejection: 2,247 compiled / 198 rejected records,
878 accepted / 108 rejected names, out of 2,445 records / 986 names.

The latest ranked candidates were evaluated from their complete RON records and
nested forms, not first-error frequency. Counts below are distinct names.

| Candidate | Affected names / estimated full unlocks | Other blockers and downstream value | Estimated effort |
| --- | --- | --- | --- |
| Saved numeric choice | Shapeshifter / 1 name, 1 record | Missing exact-visit numeric state, entry/update adapters and separate static P/T consumers. Existing entry, upkeep, information and layer services are ready. Supplies reusable persistent numeric reads and optional updates. | Medium: one state contract, adapters, lifecycle and gameplay tests. |
| Conditional artifact protections | Guardian Beast / 1 name, 1 record | Live noncreature artifact membership, current controller, control prohibition and new-Aura-only prohibition. Existing attachments must remain. Enables broader conditional permanent restrictions after those contracts exist. | High: several rule services and attachment/control interaction tests. |
| Static collection-recipient prevention | Camel / 1 name, 1 record | BandedWith exists, but static collection context and replacement-time membership rechecks are missing. Supplies a prerequisite for broader collection damage rules; Blood of the Martyr still needs optional redirection. | Medium/high: live collection query, static prevention adapter and replacement/combat tests. |

Other inspected complete records included Bronze Horse, Gaea's Liege, Heroism,
Tidal Flats, Blood of the Martyr, Kry Shield, Silhouette, Nameless Race,
Frankenstein's Monster and Wood Elemental. Their distinct spell-target, combat,
loop, entry-payment and result producers prevent treating them as additional
unlocks from any of these three narrow slices. Gaea's Liege retains the known
contradictory attack predicates; it was not changed or counted as an unlock.

All three ranked candidates have one credible immediate full-card gain. The
saved numeric choice was selected because its surrounding engine dependencies
are ready and it has the smallest complete rules boundary. The finite copiable
variant implementation cannot store or update this independent number. No
additional slice, production fallback, Go-card removal or DSL cleanup was added.

### Supported signatures, contexts and engine boundary

- `ChooseNumberAsEnters(minimum, maximum int) AsEntersBattlefieldAbility`:
  nonnegative ordered integer bounds; entry controller chooses before
  registration. Missing players and invalid answers use the minimum.
- `MayChooseNumber(subject PermanentRef, minimum, maximum int) ResolvingEffect`:
  resolving controller chooses; decline keeps state. Saves the subject visit and
  rechecks after the offer and the numeric choice. Absent, returned or phased
  subjects supply no update. Successful updates refresh existing layers.
- `ChosenNumberOf(subject PermanentRef) ResolutionValue`: pure exact-visit
  current/phase/departure information. Missing choices supply zero. An integer
  plus a presence flag belongs to each Permanent; clones/LKI copy both by value.
- RON: one `AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(min,max)])`;
  `MayAction(ChooseANumberBetween(min,max))` in resolving battlefield source
  bodies; bound symbol `TheChosenNumber` in static values, including `Minus`;
  source `PermanentLayerEffect` with separate `SetPower(value)` and
  `SetToughness(value)`, at most one of each, under the number entry declaration.
  Unsupported mixed lists, duplicate declarations, malformed bounds, nonpermanent
  entry, payments, zone-source bodies and unbound/granted reads reject early.

RON owns spelling, source scope and validation. mage owns the saved state,
decision, visit rechecks and information read. Existing source references, entry
copy ordering, entry batches, upkeep/APNAP, permanent information/LKI, clone,
spell-copy/token and layer 7b/7c/7d services are reused. No new cost transaction,
replacement-order policy, zone subsystem or global action-result slot is added.
Source control and phase cycles retain the number. Return and copied entry make
new choices. Relevant local rules: CR 107.1a/107.2, 400.7, 607.2d, 613.4b–d,
614.12a, 702.26 and 707.5–6. Local ATQ Oracle text matches the unchanged record.

### Failing-first evidence and card-to-test mapping

Initial RON gameplay failed on unsupported ChooseANumberBetween. Engine tests
failed on the missing choice/read/update APIs and fields. Test setup signatures
for phase-in and free mana cost were corrected during development. Initial
post-implementation checks also corrected two test assumptions: FindPermanent
can see a provisional entrant, and a 7/0 must die at the state-based check.
No engine rule or gameplay assertion was weakened to retain it.

- `TestNumericChoiceTimingAndIdentity`: bounds, pre-registration choice,
  controller versus owner, phase cycles, clone isolation, departure LKI,
  new return choices and refusal to update the departed visit.
- `TestNumericChoiceUpdate`: decline, detection-controller retention after
  control change, layer refresh, phasing and departure during numeric choice.
- `TestNumericChoiceInvalid`: invalid bounds, nonpermanent entry, nil subjects
  and invalid update definitions.
- `TestNumericChoiceCopyAndBatch`: copy replacement before acquired entry
  choice, independent donor/copy values, all batch choices before observations
  and missing-player fallback.
- `TestNumericChoiceLayerOrderAndNoChoice`: bonuses, counters and switches,
  live update, pure missing-choice reads and phase information.
- **Shapeshifter (ATQ) → `TestRONNumericChoiceReal`**: all eight entry numbers,
  failed/paid native casting, entry versus announcement timing, opponent upkeep,
  decline, resolution updates, pending game clones, phasing and 7/0 death.
- **Shapeshifter → `TestRONNumericChoiceSpellCopyAndReturn`**: permanent spell
  copy chooses independently as a token, departure/return cannot redirect a
  pending update, clone isolation and native chosen-power combat damage.
- **Shapeshifter → `TestRONNumericChoiceLoadAndInstances`**: strict complete
  record loading, factory separation, atomic rejection and native upkeep priority.
- `TestRONNumericChoiceInvalid` / `TestRONNumericChoiceContexts`: malformed,
  duplicate, unbound, wrong-role, nonpermanent and granted-source rejection.

### Validation, loader delta and preservation

Focused `go test ./pkg/mage/dsl/ron -run TestRONNumericChoice -count=1` and
`go test ./pkg/mage -run TestNumericChoice -count=1` passed. `make build`,
`go vet ./...`, `go test ./pkg/mage/... -short -count=1`, `make test` and the
engine skill's `go test ./... -short -count=1` all passed on the final code.

The first broader engine run found an existing rejection assertion for unrelated
fixed static toughness and a printed-upkeep inventory count. Separate P/T
lowering was scoped to the selected source entry-number context, preserving that
rejection. Both printed-upkeep inventories now assert 63 names / 125 records and
explicitly include Shapeshifter. Four collection-count assertions use 2,248/197.
The first lint run found gocritic's unnamedResult on the bounds helper. Named
results fix it. Focused tests, build, vet, engine short, full and repository short
checks were repeated after this change. Final `make lint` passed with **0 issues**.
Automatic changes were import grouping in new files; they were reviewed.
Post-format focused RON/engine tests and `git diff --check` passed again.

The final required loader exited **1**, as expected for unsupported collection
records: **2,248 compiled / 197 rejected records**, **879 accepted / 107 rejected
unique names**. Exact baseline comparison removes only Shapeshifter's former
ATQ 61:189 diagnostic. No other diagnostic is added or changed. Every previously
accepted record remains accepted. The vocabulary loader baseline equals the
final report; every recorded source hash matches. No environment or validation
blocker remains.

Estimated and actual gains match: **one unique name / one record**. Native
complete-record gameplay coverage grows by Shapeshifter. Compilation coverage
and gameplay evidence remain separate; 879 accepted names are not all claimed
as complete gameplay coverage. No RON source changed and no new source defect
was corrected. The vocabulary's loader baseline is refreshed; source hashes and
syntax occurrence counts remain unchanged.

Changed files: new mage numeric_choice.go / numeric_choice_test.go; card.go and
clone.go; new ron numeric_choice.go / numeric_choice_test.go; compiler.go,
entry_tapped.go, tap_choice.go, layers.go, static_value.go and inventory.go;
collection-count and printed-upkeep inventory tests; doc.go, dsl-reference.md,
engine-rules-contracts.md, vocabulary loader baseline and this progress ledger.
Existing Go card definitions, startup paths and broad DSL exports remain gated.

### Ranked next candidates

1. **Shared live static collection queries.** Compare Camel's prevention
   recipients, Bronze Horse's Other(ThisPermanent) condition and Guardian Beast's
   current-controller artifact membership. Distinguish a reusable prerequisite
   from their separate spell-source, Aura and control-rule blockers.
2. **Conditional artifact protection rules — Guardian Beast.** Keep existing
   Auras; prohibit new Aura attachment and control acquisition only while the
   source is untapped. Reuse exact shared rules after live membership is ready.
3. **Entry payment/result producers.** Compare Nameless Race, Frankenstein's
   Monster and Wood Elemental. Their life, exile and sacrifice decisions need
   separate typed results and replacement/batch contracts before adapters.

Complete collection gameplay, runtime cutover, Go-card deletion, obsolete
Go-set generation and unused DSL cleanup remain incomplete. Stop after this
saved numeric source-choice slice.

## 2026-10-14 — LIVE-STATIC-COLLECTION-QUERIES

### Baseline, candidates and selection

The working tree was clean. Fresh `make build` and RON package tests passed.
The initial loader exited 1 for collection rejection: 2,248 compiled / 197
rejected records, 879 accepted / 107 rejected names. This was not a build failure.
Read the execution design, engine API, ledger top/latest entry, targeted vocabulary
keys, selected compiler/engine code and complete records for every candidate.

| Candidate | Distinct affected names | Estimated full-card gain | Other blockers and downstream benefit | Effort |
| --- | --- | --- | --- | --- |
| Shared live static queries | Camel, Bronze Horse, Guardian Beast | 1 name / 1 record | Camel needs live band recipients. Bronze Horse needs source exclusion, then spell-source prevention. Guardian Beast already has artifact layer membership; new Aura and control restrictions remain. Typed live sets support future rule consumers. | Moderate: adapters, two reusable consumers and lifecycle tests |
| Conditional artifact protection | Guardian Beast | 1 name / 1 record | Its layer membership is ready, but both Aura attachment and control acquisition need exact restrictions. Shared protection rules follow. | Higher: two rule contracts and attachment/control tests |
| Entry payment/result producers | Nameless Race, Frankenstein's Monster, Wood Elemental | 0 from one common prerequisite; 3 eventual names | Life bound and saved P/T; X graveyard exile, failure replacement and counter combinations; Forest sacrifice and two CDA reads are separate contracts. A common entry-result framework enables all three. | High: multiple decisions, replacements, results and layers |

Selected ranked candidate #1 as one shared live query capability with complete
static control-condition and prevention consumers. Its immediate gain is Camel;
Bronze Horse's next blocker is now exposed. Guardian Beast's existing selection
was reused as evidence of readiness, not counted as a new unlock. No card-name
branches or separate spell-source/Aura/control/payment slice was added.

Local ARN Oracle text matches Camel's unchanged full RON record. CR 400.7,
615.1/615.6/615.12, 616.1f/616.2, 702.22e–g and 702.26 establish source visits,
live prevention, rechecks, band removal and phasing. No source defect was fixed.

### Signatures, supported contexts and reuse boundary

- `PlayerControlsAny(player PlayerRef, subjects PermanentSet) StateCondition`:
  present exact source, typed player and live membership; no choices or writes.
- `PreventDamageFromMatchingPermanentsToSetIf(subjects PermanentSet,
  from PermanentFilter, condition StateCondition) StaticRule`: nonnil set and
  condition; live matching recipients, permanent sources and all damage. A zero
  source filter permits any permanent. Source visit is saved per registration.
- RON battlefield static collection shapes: `SinglePermanent(ThisPermanent)`,
  `Other(ThisPermanent)`, `BandedWith(ThisPermanent)`,
  `ControlledByAPlayer(SinglePlayer(You))`, existing static-condition filters,
  and nonempty nested `And`/`Or` lists. Other source references reject.
- Static `PlayerPassesFilter(player, ControlsA(set))` / `ControlsNo(set)` consume
  contextual sets with existing You or declared chosen-player scope. Existing
  characteristic-only and chosen-color condition adapters retain their behavior.
- `PreventDamage(DamageWouldBeDealtByAPermanentToARecipient(sourceFilter,
  APermanent(set)), [PreventThatDamage])` accepts these live sets, directly or
  in supported static If/Unless/IfElse branches. Only this operation is supported.
  Resolving, entry, payment, target and event bindings cannot enter this static
  collection scope. Spell-source prevention remains a compile error.

RON owns syntax, scope and early validation. mage owns current control, source
identity, condition checks and prevention. Reuse includes typed permanent sets,
union/intersection/difference, singleton, live control, attacking bands, source
LKI/phase facts, static cycle registration, replacement rechecks, waiting source
queries, simultaneous damage, unpreventable damage, cloning and native casting.
Existing singular prevention callers keep their contracts. No members are saved
at declaration time. Empty sets can acquire members later. Current queries and
waiting queries use the same condition and set; returned sources cannot supply
old registrations. Layer consumers keep their existing layer-cycle semantics.

### Failing-first evidence and card-to-test mapping

Initial RON tests failed on the collection-recipient event and Other condition.
Initial engine tests failed to compile because both new public APIs were absent.
The first implementation pass exposed a test timing assumption: direct PhaseOut
needs an explicit layer cycle before inspecting derived keywords. Spell-copy
lethal damage likewise needs the state-based check. Test setup used the existing
Card field and CurrentPower/CurrentToughness APIs after compilation corrections.
No rules or gameplay assertions were weakened.

- `TestLiveStaticCollectionPrevention`: live control, recipient return, source
  phase/departure, ability loss, cloned state, source LKI, replacement
  rechecks, unpreventable damage and simultaneous permanent/player packets.
- `TestLiveStaticCollectionCondition`: source exclusion, new members, phase,
  clone isolation and refusal of a returned source visit.
- `TestLiveStaticCollectionInvalid`: nil sets/conditions, targets and unbound
  saved sets fail definition validation.
- `TestLiveStaticCollectionWaitingIdentity`: live waiting recipients, phase,
  condition changes, cloned registrations and stale source return.
- **Camel (ARN) → `TestRONLiveStaticCollectionCamel`**: full record, failed/paid
  casting, printed banding, native band formation, attacking-source prevention,
  simultaneous packets, native combat, clone and removal from the band.
- **Camel → `TestRONLiveStaticCollectionCamelChanges` / `CamelSpellCopy`**:
  ability loss, source/member return and phase cycles, departed/returned damage
  source information, non-Desert sources and independent copied spell instances.
- **Camel → `TestRONLiveStaticCollectionLoadAndScope` / `Invalid` / `Inventory`**:
  complete-record strict loading, separate factories, atomic rejection, invalid
  shapes/contexts and event/recipient/band/reference inventory roles.
- `TestRONLiveStaticCollectionCondition`, `NestedControl`, `Unconditional`:
  synthetic contract evidence for Bronze Horse's source-exclusion prerequisite,
  live control, nested set composition, ControlsNo and direct prevention.
  Bronze Horse still fails on its actual unsupported spell-source event.

The engine short suite found one obsolete rejection expectation for a now-valid
ControlsA current-controller collection. Moved that declaration into positive
P/T gameplay coverage in NestedControl; other invalid declarations remain.
Four collection-count assertions now expect 2,249/196. Compilation acceptance
and complete-card gameplay evidence remain separate.

### Validation, preservation and next work

Focused `go test ./pkg/mage/dsl/ron -run TestRONLiveStaticCollection -count=1`
and `go test ./pkg/mage -run TestLiveStaticCollection -count=1` passed.
`make build`, `go vet ./...`, `go test ./pkg/mage/... -short -count=1`,
`make test` and the engine skill's `go test ./... -short -count=1` passed.
Lint's automatic changes were import grouping and a slices.ContainsFunc rewrite
of the new recursive syntax search. Reviewed both and repeated affected checks.
The first lint run reported prealloc on the waiting recipient result. Allocating
capacity from the live subjects fixes it; checks were rerun on the final code.
Final `make lint` passed with **0 issues**; `git diff --check` passed. All required
checks and repository short tests passed on the final implementation. No
validation or environment blocker remains.

The final required loader exited **1**, as expected: **2,249 compiled / 196
rejected records**, **880 accepted / 106 rejected names**. Exact baseline
comparison removes only Camel's ARN 12:200 rejection. Bronze Horse's diagnostic
advances from LEG 38:199 Other condition to LEG 38:262 spell-source prevention.
No other diagnostic changes and no new rejection appears. Every previously
accepted record stays accepted. Vocabulary loader baseline equals the final
report; all source hashes match, and syntax counts are unchanged.

Estimated and actual gains match: **one unique name / one record**. Complete RON
native gameplay coverage gains Camel; synthetic source-exclusion coverage is a
prerequisite for Bronze Horse, not a complete-card gameplay claim. Guardian Beast
retains its protection blocker. No source record, production Go definition,
startup path, generator or broad DSL export changed. Removal gates remain shut.

Changed files: new mage live_static_collection.go / live_static_collection_test.go;
static_combat_prevention.go and waiting_prevention.go; new ron
live_static_collection.go / live_static_collection_test.go; chosen_static.go,
static_prevention.go and inventory.go; conditional_static_test.go and three
collection-count test files; doc.go, dsl-reference.md, engine-rules-contracts.md,
vocabulary loader baseline and this progress ledger.

### Ranked next candidates

1. **Saved voluntary-sacrifice result contracts.** Reassess Wood Elemental and
   Sword of the Ages together. Shared visit selection, completed sacrifice batches
   and retained characteristics may enable both; entry and activation contexts
   need separate adapters. Do not infer two unlocks from the common syntax alone.
2. **Entry payment/result prerequisites.** Compare Nameless Race and
   Frankenstein's Monster with Wood Elemental. Choose one bounded shared contract,
   then account for life bounds, X/exile failure replacement, counters and CDAs.
3. **Remaining conditional static protection.** Compare Guardian Beast's Aura
   attachment/control restrictions with Bronze Horse's spell-target source
   prevention now that its condition compiles. Both are single-card follow-ups;
   prefer credible shared entry/result work while it remains dependency-ready.

Complete collection gameplay, runtime cutover, Go-card deletion, obsolete
Go-set generation and unused DSL cleanup remain incomplete. Stop after this
live static collection query slice.
