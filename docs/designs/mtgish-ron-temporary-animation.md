# Temporary base values and type additions

Slice P7-TEMPORARY-ANIMATION, 2026-10-04. Complete for the signatures below;
other temporary modifications and durations remain partial.

## Selection from fresh evidence

Baseline RON/CLI tests, engine short tests and build passed. Strict loader:
985/2,445 compiled records; 1,460 rejected; 296 accepted unique names and
690 rejected names. The full inventory walks 72,938 nodes and 8,528 signatures,
including descendants of rejected parents. Candidate records were read in full.
Reprints count once in this comparison.

| Candidate | Distinct names audited | Estimated full-card gain | Other blockers and downstream benefit | Effort |
| --- | --- | ---: | --- | --- |
| Temporary base P/T and type additions, source-excluding target | Mishra's Factory, Sorceress Queen, Jade Statue, Thelonite Druid, Halfdane, Farrel's Mantle | 2 | Statue needs combat timing/duration; Druid needs sacrifice choice and collection animation; Halfdane needs dynamic P/T and next upkeep; Mantle needs combat-event bindings and damage payment. Shared types/base layers supply animation prerequisites. | Small adapters plus one typed engine type operation; moderate identity/layer gameplay matrix |
| Activation modifiers | Disrupting Scepter, Gwendlyn Di Corci, Nebuchadnezzar, Instill Energy, Library of Alexandria, Vampire Bats | 1 for own-turn restriction; 3 if independent hand-count/limit forms are also implemented | Random discard; announced X/name/result scopes; attack permission and once-per-turn; hand-count condition; per-turn count. Timing and limit adapters enable several later activated cards. | Several condition and limit contracts for three unlocks; one own-turn adapter for one |
| Variable optional mana payment | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke still needs opponent cast scope and color-sharing spell filters. Typed spell values and variable resolution payments enable later taxes. | Moderate payment/value adapter and rollback/copy/event matrix |

Temporary layers provide two immediate unique unlocks with a shared, coherent
boundary. The selection, contract, reuse boundary and acceptance matrix were
stated before implementation. Counts of first errors did not determine selection.
The full nested inventory for SetPT/AddCardtype/AddCreatureType in temporary
contexts and Other in permanent targets was inspected. It also exposes Druid's
collection form under its unsupported cost. Complete records for all names above
were read, including identical Queen and Statue reprints.

## Exact supported contract

Inside existing `CreatePermanentLayerEffectUntil(permanent, modifications,
UntilEndOfTurn)/3`:

- `SetPT(PT(integer_literal, integer_literal))/1`: set both base values, layer 7b.
- `AddCardtype(Creature|Artifact|Land|Enchantment)/1`: add a type, layer 4.
- `AddCreatureType(symbol)/1` adds a creature subtype; `SetCreatureType(symbol)/1`
  replaces the creature subtype family, layer 4; apply the existing
  AssemblyWorker spelling conversion and engine subtype-family validation.
- These combine with existing AdjustPT, AdjustPTX and keyword modifications in
  one Until installation. They retain existing colors, abilities and other types.

Subjects use the existing typed source, attachment or single permanent target
reference plans. Supported spell, activation, trigger and payment continuation
builders use the same compiler action path. Single-target reads have fresh
per-factory bindings. Bare spell sources, escaped/cross-rule references and
wrong reference types remain rejected by existing scope checks.

`TargetPermanent(Other(ThisPermanent))/1`, or Other under conjunctions with
existing filters, is supported in permanent-source action contexts. RON lowers
source exclusion to DeclarePermanentTarget with TargetOtherThanSource. Other in
disjunctions, Other of a target/event/host, and Other in a spell source context
remain rejected. The general filter compiler is unchanged.

Dynamic ManualPT, UntilNextUpkeep, UntilEndOfCombat, collection animation and
activation modifiers remain gaps. This run did not start any of those slices.

## Engine boundary and rules

New engine signature:
`AddCardType(PermanentRef, core.CardType) ContinuousModification` supports the
four attribute-backed permanent types listed above. Invalid types produce a
definition validation error. It uses the existing layerModification,
capturedModification, Until, incarnation capture, effect manager and attribute
services. Creature addition also grants the engine's power/toughness identity.
Within one captured type effect, additions are applied and synchronized before
subtype checks, independent of modification list order. Other subtype operations
keep their order. Existing callers retain their behavior. Type addition does not replace subtypes;
the compiler selects the distinct subtype addition or replacement operation.

`ValidateSubtypeFamily(SubtypeFamily, ...string) error` exposes the existing
constructor check without a panic; constructors delegate to the same check and
retain their existing panic contract. RON rejects wrong families before building
factories. As before, names outside known noncreature families are permitted as
creature subtypes; this is not a new exhaustive subtype vocabulary.

Reuse: SetBasePowerToughness, AddSubtypeFamily, SetSubtypeFamily, typed target validation,
TargetOtherThanSource, shared mana solver and transactional costs, source and
attachment identity, stack copies/retargeting, game cloning, phase-in/out and
end-of-turn removal. No new LKI query, replacement ordering, APNAP, event batch,
zone operation, or payment-result contract is introduced. CR 205.1b, 302.6,
608.2b, 611.2a/c and 613.1d/4b/4c/6/7 govern preservation, legality, identity,
duration and layer order. Relevant API/rules-contract docs were read and updated
only for this established behavior. Local Oracle data establishes the source
correction described below.

## TDD and gameplay evidence

Initial engine tests failed because AddCardType was absent. An incorrect test
controller API was fixed before confirming that expected failure. Initial RON
tests then failed at SetPT and Other after correcting test-only mana/sickness API
names. No gameplay expectation was weakened. A retained invalid-context test
now rejects malformed Other(ThisPermanent()) because the valid form is supported.
Exact inventory compilation expectations were refreshed to measured counts.

| Card/context | Tests and acceptance evidence |
| --- | --- |
| Mishra's Factory | TestRONTemporaryAnimationFactory: all three characteristics added at resolution, existing land retained, immediate mana and solver recognition, payment, absent/returned/phased source, phase-in before resolution, control change, clone pending resolution, cleanup isolation, summoning sickness and retained mana ability |
| Mishra's Factory | TestRONTemporaryAnimationFactoryBoostAndRepeat: opponent Assembly-Worker boost, noncreature target rollback, reanimation preserves bonus after base set, insufficient mana rollback |
| Sorceress Queen | TestRONTemporaryAnimationQueen: self exclusion and tap rollback; opponent creature; base set before bonuses/counters, flying retained; illegal noncreature/absent/returned/phased target; phase-in, source departure/return/control change, clones and cleanup |
| Typed synthetic spell and declarations | TestRONTemporaryAnimationDeclarations: fresh factories and definition target reads; invalid P/T, type/subtype/syntax and source/filter scopes |
| Typed synthetic copy and inventory | TestRONTemporaryAnimationCopyAndInventory: retargeted copy/original independent reads; new fixed_pt/card_type/creature_subtype roles |
| Engine synthetic permanents | TestTemporaryCardTypeCapture, TestTemporaryCardTypeKinds, TestTemporaryCardTypeDefinitionValidation: supported kinds, retained characteristics, combined subtype/type/base/bonus layer application, identity/absence/phasing/control, clone expiry, undeclared subjects and invalid types |

Both newly accepted unique cards have RON gameplay evidence for every printed
rule. This is separate from collection-wide gameplay completion. Other supported
action contexts reuse the same builders and retained context regressions; these
tests do not claim exhaustive new gameplay for every context or interaction.

## Fresh results and remaining work

Actual gain equals estimate: **two unique names, three records** (Queen has two
reprints). Strict loader: **988/2,445 compiled, 1,457 rejected**. Unique names:
**298 accepted, 688 rejected**. No previously accepted record regressed.
Halfdane's diagnostic advances to unsupported UntilNextUpkeep; its ManualPT
still has no adapter. Inventory and strict issues match exactly. The vocabulary snapshot has refreshed source hashes, the renamed constructor
and current loader diagnostics. Known Gaea's Liege/Fastbond concerns remain.
The loader still rejects the collection as required. Global engine_validated and
behavior_verified values remain null. Unknown-role occurrences decrease from
19,421 to 19,408 from the new argument roles; total nodes/signatures are unchanged.

Removal gates are closed. Production registries, Go definitions/generators and
DSL exports remain. No whole-repository deletion audit or startup claim is made.
Next candidates: shared activation modifiers with a fresh multi-card timing/limit
comparison; collection temporary modifications (Druid after sacrifice choice,
plus existing mass boost cards); variable optional payment (Chaos directly,
Invoke after spell filters). Reassess dependencies before the next selection.

## Source-data correction

CR 205.1b specifies that becoming a creature-type artifact creature replaces
existing creature types. Local data/ATQ.json and data/LEA.json give Factory's
Assembly-Worker animation and Statue's Golem animation respectively. Their RON
records instead used AddCreatureType, which preserves earlier creature types.
The new prior-Human Factory regression failed with that spelling. All six uses
were corrected to SetCreatureType: Factory in ATQ; identical Jade Statue records
in 2ED, CED, CEI, LEA and LEB. The shared adapter calls SetSubtypeFamily; it does
not change the meaning of AddCreatureType. No other RON expression changed.
The prior-Human test passes after the correction; Factory's later base set also
keeps its independent bonus. Statue remains rejected for activation modifiers
and combat duration. Its corrected source is not a gameplay-support claim.

Both spellings have separate compiler lowering. Invalid declarations test both
wrong-family and malformed replacement operands. The snapshot keeps 600 tuple
signatures and 51 top-level rule names: six constructor occurrences are renamed.

## Validation

Baseline focused RON/CLI tests, engine short tests and make build passed. Final
focused engine/RON tests with `-count=1`, RON/CLI package tests,
`go test ./pkg/mage/... -short -count=1`, `make test`, `make build`, `go vet ./...`,
and `go test ./... -short -count=1` all pass after the final code/source changes.
`make lint` initially rejected a test variable named copy; it was renamed and
lint passes with zero issues. Automatic changes only reorder test imports and
were reviewed. `git diff --check` passes. Final strict loader and full inventory
both exit 1 for the remaining unsupported collection, with identical diagnostics.
No environment failure or required-check waiver blocks the commit.
