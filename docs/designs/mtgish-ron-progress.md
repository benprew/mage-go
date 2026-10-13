# mtgish RON execution progress

Completed slices, test mappings, validation evidence and historical decisions
are in [mtgish-ron-completed.md](mtgish-ron-completed.md).
Use this document for current status and next work.

## Current status

Latest measured result (2026-10-12): **845 accepted / 141 rejected names**
(986 total), **2,186 compiled / 259 rejected records** (2,445 total).
Fresh baseline was 844/142 names and 2,180/265 records. The strict loader still
rejects the collection and publishes no partial registry.

The latest slice is **HAND-CARDINALITY-DISCARD**. Complete Balance gameplay now
executes land sacrifice, hand equalization and creature sacrifice in all six
reprints. This adds one compiled unique name, six compiled records and one
complete-card gameplay mapping. Runtime cutover and removal gates remain open.

## Ranked next candidates

Reassess these candidates with fresh loader and inventory evidence under
[the execution design](mtgish-ron-execution.md). Read complete RON records and
compare full-card gains, shared dependencies and effort before choosing a slice.
Count reprints once; a first diagnostic alone does not establish a full-card gain.

1. **Raiding Party: optional tapping with per-player actual results.**
   Reuse the bounded subset selector and typed player-number frames. Save only
   actual tap transitions; later plural kept choices and exclusions remain separate.
   This prerequisite has zero immediate full-card gains, then enables one name.
2. **Shared dependent permanent target comparison.**
   Gauntlets of Chaos needs shared-type comparison against the first target,
   followed by exact exchange-result conditions. Preserve its Aura-filter source
   defect. Compare current multi-card target/filter opportunities before selecting.
3. **City in a Bottle: state-member capture and global player restrictions.**
   State capture alone has zero immediate full-card gains. Spell and land-play
   restriction services can also support Worms of the Earth's land-play rule;
   assess those shared services before attempting the complete card.

## Dependencies and source blockers

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
