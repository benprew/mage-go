# mtgish RON execution progress

Completed slices, test mappings, validation evidence and historical decisions
are in [mtgish-ron-completed.md](mtgish-ron-completed.md).
Use this document for current status and next work.

## Current status

Latest recorded shared-worktree result (2026-10-11): **844 accepted / 142 rejected
names** (986 total), **2,180 compiled / 265 rejected records** (2,445 total).
The isolated HEAD-plus-slice result was **843/143 names** and **2,179/266 records**;
pre-existing Eureka work accounts for the difference. These are recorded results,
not a fresh baseline. The strict loader publishes no partial registry.

The latest completed slice is **PERMANENT-CARDINALITY-SACRIFICE**. Balance's land
and creature rounds execute in all six reprints. Its hand round still rejects at
`DiscardAllButNumberCards`, so Balance has no complete-card gameplay mapping.
This slice adds no compiled names or records.

## Ranked next candidates

Reassess these candidates with fresh loader and inventory evidence under
[the execution design](mtgish-ron-execution.md). Read complete RON records and
compare full-card gains, shared dependencies and effort before choosing a slice.
Count reprints once; a first diagnostic alone does not establish a full-card gain.

1. **Raiding Party: bounded optional permanent selection and plural results.**
   The shared bounded subset selector is available. Optional tapping, actual
   per-player tap counts, saved plural choices and `NotChosenThisWay` exclusions
   still need contracts. A count adapter alone does not complete the card.
2. **Balance: mandatory hand cardinality equalization.**
   The two permanent rounds are supported. Exact hand choices and simultaneous
   discard can complete the remaining instruction in all six reprints. Compare
   this single-name gain with current shared-capability candidates.
3. **City in a Bottle and Gauntlets of Chaos: shared prerequisites.**
   City in a Bottle needs state-member capture, then global spell-casting and
   land-play restrictions. Gauntlets needs dependent shared-type target comparison
   and exact exchange-result binding/conditions. Each prerequisite alone has
   zero estimated complete-card gains.

## Dependencies and source blockers

- Reuse `SacrificeAllButEach`, `LeastControlledPermanentCount`, the bounded
  subset selector and the existing simultaneous sacrifice service for permanent
  cardinality work. The latest slice supports required kept-count choices in
  APNAP order; it does not add optional cardinality or saved plural results.
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
