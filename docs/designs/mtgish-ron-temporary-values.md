# RON captured temporary P/T values

Capability: P1-TEMPORARY-VALUES. Complete for the signatures and contexts below.

## Selection and fresh evidence

The working tree initially had user changes to `.gitignore` and
`mtgish-ron-execution.md`; both are preserved and excluded from this slice.
No unrelated untracked files existed. Baseline compiler tests and `make build`
passed. Strict loading and nested AST inventory exited 1 for collection rejection:
877/2,445 compiled records, 1,568 rejected records, 259 compiled unique names,
727 rejected unique names. The inventory inspects descendants of rejected rules.
The vocabulary snapshot was read as syntax evidence; all 12 source hashes match.

Complete RON records and nested forms were inspected for these candidates.
Reprints count once. The variable candidate is restricted to temporary layers;
Aspect of Wolf's static AdjustPTXY is a separate context and capability.

| Candidate | Distinct affected names | Full-card unlock estimate | Other blockers | Downstream benefit | Effort |
| --- | --- | --- | --- | --- | --- |
| Captured temporary P/T, selected | Berserk, Blood Lust, Great Defender, Howl from Beyond, Kry Shield, Subdue | 2: Great Defender, Howl from Beyond | Berserk needs casting timing, power reads and delayed triggers; Blood Lust needs branches, toughness reads and subtraction; Kry Shield needs prevention and controlled target filters; Subdue needs prevention | Captured numbers and action-local subject identity for four later cards; typed layer-value lowering | Small adapters, medium identity/capture/gameplay tests |
| Optional fixed counter payments | Force Spike, Power Sink, Thrull Wizard, Vodalian Mage | 2: Force Spike, Vodalian Mage | Power Sink needs X payments, player actions, mana loss and collections; Thrull Wizard needs payment alternatives | Payer references, optional decisions and separate payment results | Medium/large payer/scope/payment implementation and tests |
| Static landwalk and subtype subjects | Goblin King, Lord of Atlantis, Zombie Master | 2: Goblin King, Lord of Atlantis | Subtype-only subjects and parameterized landwalk need adapters; Zombie Master also needs granted activated abilities | Shared lord effects and landwalk predicates | Medium adapters and static/combat lifecycle tests |

Temporary values supply two concrete unlocks with the smallest adapter and reuse
existing typed reference, value and layer services. The ledger suggestion was
reassessed against fresh evidence. No first-error frequency was treated as an
unlock estimate. No second slice was started.

## Supported contract and reuse boundary

Inside existing `CreatePermanentLayerEffectUntil(subject, modifications,
UntilEndOfTurn)/3`, `AdjustPTX(power, toughness, value)/3` accepts:

- Coordinate symbols `PlusX`, `MinusX`, `Zero`, or `Integer(signed literal)/1`.
- Value symbol `ValueX` with an existing announced-X declaration,
  `Integer(signed literal)/1`, or `ManaValueOfPermanent(reference)/1`.
- A mana-value reference equal to the layer's typed subject, or the local symbol
  `ActionPermanent`, which names that subject only in this value expression.

The layer subject retains the existing source, Aura host or local single-target
contracts. Existing spell, activation and supported triggered action contexts
can use the numeric modification. Static modifications, AdjustPTXY, power and
toughness readers, arithmetic expressions, other-object characteristic reads,
X activation payment declarations, collections and extended durations remain
unsupported. A value is validated even when both coordinates are constant.

Wrong arity, tuple coordinate symbols, undeclared X, wrong-role references,
undeclared targets, malformed references and a mana-value read of a different
permanent fail compilation with positioned diagnostics. ActionPermanent is not
inserted into the enclosing scope or shared with another action. Other-object
reads are deferred because source-departure/LKI semantics need a separate
contract. No zero-valued substitute is used to accept such declarations.

RON owns spelling, role validation and immutable build plans. The subject plan
is resolved through per-factory instanceBindings. Each factory creates fresh
targets; a modification's numeric reader uses the same binding as its subject.
The engine owns capture, execution, copy frames, clones, legality and expiry.
Reuse: `ModifyPT`, `Until`, `Fixed`, `ValueAtResolution`, `XValue`, `ManaValueOf`
and `MultiplyValues`. MinusX multiplies by -1; literal coordinates remain
independent of the supplied value. Mixed fixed/numeric/keyword modifications
retain one Until installation. No missing engine prerequisite was found.

Rules evidence: CR 107.3a/g, 202.3/3e, 400.7, 608.2b/c/h, 611.2a/c/d,
613.4c and 702.26b/d/f. X is announced and paid when casting. Mana value is read
when the effect starts; both values are then captured, rather than read again
during continuous-effect application. Copies keep announced X and read their
own retargeted subject. Returned objects have a different incarnation. Phased
or illegal targets cannot receive a new bonus; an installed bonus survives
phasing and ends at cleanup. This slice has no replacement or simultaneous zone
operation. It adds no engine LKI service and does not claim new LKI behavior.

## Tests and card coverage

TDD: executable gameplay tests first failed on unsupported AdjustPTX for both
real cards and the synthetic mixed modification. Inventory tests later failed
on absent coordinate/value/reference roles before those roles were added.
Fixture corrections replaced an incorrect card-copy method name and an inactive
source-less static test bonus; the final fixture uses one real resolving layer
installation. No production behavior was changed to accommodate those fixtures.

| Test | Mapping and acceptance cases |
| --- | --- |
| TestRONTemporaryValueCards/Howl_from_Beyond | Full real LEB record: announced X=4, X payment, rollback, announcement versus resolution, opponent target, retargeted copy with X, returned/phased/phase-in target, cloned stack and cleanup isolation, installed phase cycle and expiry |
| TestRONTemporaryValueCards/Great_Defender | Full real LEG record: target gets toughness equal to its mana value; copy reads Shivan Dragon's six rather than Llanowar Elves' one; same legality, identity, cloning, phasing and expiry cases |
| TestRONTemporaryValueZeroXAndAnnouncementLegality | Howl from Beyond accepts X=0; rejects Forest as a noncreature without consuming mana or changing the stack |
| TestRONTemporaryValueCaptureAndLocalScope | Fresh synthetic factories and definition validation; local ActionPermanent, Integer coordinate, MinusX, mixed fixed/numeric/keyword modifications, source departure, mana value changed before resolution and again after installation, captured amount, returned target excludes old bonus |
| TestRONTemporaryValueRejectsInvalidDeclarations | Wrong arity/kind/role, unbound X, malformed and unrelated permanent references, unsupported reader |
| TestRONTemporaryValueSourceHostAndTriggerContexts | Synthetic source, Aura host and source-entry trigger contexts; signed literal coordinate, fixed numeric value, local mana-value reader and state-based action for zero toughness |
| TestRONTemporaryValueInventory | Complete nested temporary-modification, coordinate, numeric value and permanent-reference roles; inspection does not imply support |
| Existing inventory collection/registry tests | Fresh exact counts; inspection still preserves accepted real Lightning Bolt behavior and the registry |

Real RON fixtures: Llanowar Elves, Shivan Dragon and Forest. Two unique cards
have direct gameplay tests for their complete records. Compilation progress and
behavior coverage are separate; no collection-wide gameplay total is claimed.
Existing temporary-layer tests retain host attachment/LKI, ability target
bindings, transactional costs and lifecycle evidence. No new replacement-order
or simultaneous-operation contract is introduced.

## Results and next work

Strict loader and inventory: 884 compiled records, 1,561 rejected records,
261 compiled unique names, 725 rejected unique names. Gain: seven records and
two unique names, exactly the estimate. Howl from Beyond has six reprints;
Great Defender has one record. No accepted record regressed. Unsupported records
still cause exit 1 and no partial registry is installed.

No RON source changed; no vocabulary snapshot refresh is needed. No source defect
was corrected. Gaea's Liege/Fastbond observations remain open. Remaining errors
are compiler/context gaps, not native build failures.

Ranked next candidates, subject to fresh comparison:

1. Optional fixed counter payments: Force Spike and Vodalian Mage; typed payer,
   decision and paid/unpaid scope prerequisites.
2. Static landwalk plus subtype subjects: Goblin King and Lord of Atlantis;
   Zombie Master remains blocked on granted activated abilities.
3. Numeric power/toughness and branch prerequisites: Blood Lust/Berserk;
   prevention and controlled target filters can then use this slice for
   Kry Shield/Subdue. This is broader than the completed mana-value shape.

Required command results are in the progress ledger. Runtime loading, complete
RON gameplay, native startup integration, Go-card and generator deletion, and
unused-DSL removal gates remain unmet. No deletion or caller audit is claimed.
