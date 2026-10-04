# RON fixed draws and activation life costs

Capability: P5-FIXED-DRAW, with P4-FIXED-LIFE and the required fixed color-choice
mana adapter repair. Status: complete for the shapes below. This is one slice.

## Selection and dependencies

Fresh baseline: focused RON/CLI tests, engine short tests and build pass.
Strict loader exits 1: 939/2,445 records compile, 1,506 records reject;
280 unique names compile, 706 reject. Inventory walks nested forms, including
unsupported parents. First diagnostics alone were not used to estimate unlocks.
The two initial user edits (`.gitignore` and the execution design) are preserved.

Three candidates were compared after reading complete RON records:

| Candidate | Distinct names | Full-card estimate | Other blockers, downstream value and effort |
| --- | --- | ---: | --- |
| Fixed draws plus fixed activation life cost | Jayemdae Tome, Xira Arien, Book of Rass, Greed, Standing Stones | 5 | Two small adapters using existing engine services; medium gameplay matrix. Draw supports later Jalum Tome/Sindbad work; life cost supports later payment capabilities. |
| Spell-cast life triggers | Crystal Rod, Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere, Urza's Chalice, Sol'kanar the Swamp King | 7 | Event filtering and paid continuation lowering; larger event/controller test scope. Enables later Verduran Enchantress, Nether Void and Presence of the Master event work, which retain optional-action or event-reference blockers. |
| Entry copying | Clone, Copy Artifact, Vesuvan Doppelganger | 2 initially | Entry choices and copy modifications; Doppelganger retains recurring copy/granted ability dependencies. Large entry, layer and copy-isolation test scope; enables later copy effects. |

Fixed draw activations provide the largest credible gain for the estimated
effort. Spell-cast events remain the highest-ranked next candidate. Selection,
reuse boundary and acceptance cases were stated before edits.

## Exact contract

- `DrawACard` is a **symbol**, in an action list in supported spell, activation
  and trigger contexts. It lowers to `DrawFor(You, Fixed(1))` by default.
- `PlayerAction(player, DrawACard)/2` uses the existing typed player-reference
  compiler and instance environment. Supported references remain `You`, a
  declared `Ref_TargetPlayer`, and the existing spell-controller references.
  The player selector does not escape to sibling actions. `DrawACard()` and
  argument-bearing forms are rejected.
- Activation cost `PayLife(Integer(n))/1`, including nested `And(list)/1`,
  accepts a nonnegative integer literal. It builds a fresh `LifePayCost(n)`
  for each factory call. Zero and payment equal to available life are legal.
  Dynamic amounts, negative amounts, bare integer amounts and malformed shapes
  reject at compilation. Optional life payments are still rejected by the
  existing optional-payment compiler; no paid-result binding is introduced.
- Inventory classifies the operand of cost `PayLife/1` as `value`. Descendants
  of unsupported parents remain unknown under the existing reporting contract.
  No global engine-validation or behavior-verification total is inferred.
- Fixed `AddMana(AnyManaColor)/1` and
  `AddManaRepeated(AnyManaColor, Integer(n))/2` use
  `AddManaOfAnyOneColor(Fixed(amount))`. The player chooses one color for the
  full amount. The previous adapter passed `AnyColor` directly to `AddMana`,
  which stored a wildcard pool value. This was a compiler adapter defect, not
  a source-data defect or a missing engine service.

RON owns spelling, shape checks and lowering. Engine reuse includes `DrawFor`,
`Fixed`, `LifePayCost`, `AddManaOfAnyOneColor`, the existing fresh target/action
factories, mana classification/solver, total-cost transaction, stack controller,
clone state, draw replacements/events, library movement and state-based actions.
No engine implementation, Go card definition, RON asset or production registry
was changed. No fallback, name branch or game-time declaration panic is added.

Relevant rules: CR 119.4/4b and 119.6 (life payment and zero life), 121.1/4/6
(individual draw, empty library and replacement), 602.1a/2/5a (payer, atomic
activation and tap restriction), 113.7a (independent stacked ability), and
605.1a/3 (immediate activated mana ability). The slice reuses these services;
it does not establish new general LKI, simultaneous-operation or replacement
ordering contracts.

## Complete-record audit

All distinct complete records containing `DrawACard`, including nested forms,
were inspected: Aladdin's Lamp, Book of Rass, Chains of Mephistopheles, Goblin
Artisans, Greed, Howling Mine, Island Sanctuary, Jalum Tome, Jandor's Ring,
Jayemdae Tome, Jeweled Bird, Library of Alexandria, Ring of Ma'rûf, Sage of
Lat-Nam, Sindbad, Urza's Miter, Verduran Enchantress, Xira Arien.

Beyond the four draw unlocks, these retain modifiers, optional actions, event
bindings, replacement operations, discard/sacrifice costs, ante, coin choices,
results or hidden-zone dependencies. Jalum Tome now reaches `DiscardACard`;
Sindbad reaches `RevealTheCardDrawnThisWay`. These are not full unlocks.

All complete records containing fixed/dynamic `PayLife` were read: Book of
Rass, Bronze Tablet, Cleansing, Erosion, Greed, Season of the Witch, Standing
Stones, Sylvan Library, Tempest Efreet, Wand of Ith. Optional, repeated,
alternative or per-object payment contexts stay unsupported. `Channel` contains
`MayPayLifeToProduceManaAsManaAbility` and remains a different contract.

For the color-choice repair, complete records were read for Birds of Paradise,
Black Lotus, Celestial Prism, City of Brass, Fellwar Stone, Implements of
Sacrifice, Metamorphosis, Rainbow Vale and Standing Stones. City of Brass and
Rainbow Vale retain event/delayed-trigger blockers; Fellwar Stone and
Metamorphosis use different dynamic/restricted production shapes. Already
accepted fixed users are included in regression tests.

Local Oracle data in `data/LEA.json`, `data/DRK.json` and `data/LEG.json` agrees
with all five newly accepted records. No RON source defect was corrected.
Gaea's Liege/Fastbond observations remain outside this slice. All source hashes
remain unchanged; the vocabulary syntax snapshot does not need refresh.

## Tests and measured outcomes

Tests were written and run before implementation. After correcting test API
calls, expected failures reported unsupported `DrawACard`, unsupported `PayLife`,
and an unknown life amount role. The Standing Stones test then proved the
wildcard color adapter defect. Additional regressions failed on that same defect
before its repair. Fixture corrections used the actual ability index, explicit
removal of summoning sickness for established sources, and `IsAlive` after
state-based actions clear the empty-draw flag. No production rule was changed
to match a test expectation.

| Card/context | Test evidence in `fixed_draw_test.go` |
| --- | --- |
| Jayemdae Tome, Xira Arien, Book of Rass, Greed | `TestRONFixedDrawRealActivations`: timing, correct player versus owner, costs, source control/departure/return/phasing, clone isolation; Xira flying |
| Same four | `TestRONFixedDrawPaymentRollback`: mana/life/target/tap/phasing/sickness failures preserve state; `TestRONFixedDrawLifeAndManaSources`: mana solver payments, self-target Xira and life payment bypasses damage prevention |
| Greed | `TestRONFixedDrawReplacementAndEmptyLibrary`: paid cost remains paid, empty draw loses, replacement prevents that loss |
| Xira Arien | `TestRONFixedDrawXiraInstances`: separate controllers and target declarations |
| Standing Stones | `TestRONFixedLifeStandingStones`: complete cost, immediate mana classification, each of five color choices |
| Birds of Paradise, Black Lotus, Celestial Prism, Implements of Sacrifice | `TestRONFixedDrawManaPrerequisiteRegressions`: all five colors, full one-color amount, costs, immediate execution, sacrifice and Birds flying |
| Synthetic boundaries | `TestRONFixedLifeBoundariesAndCombinedRollback`: zero/equal life, fresh costs, combined life costs roll back tap and mana-source payment |
| Synthetic spell and trigger | `TestRONFixedDrawSharedContextsAndCopies`: declaration validation, fresh targets from one factory, retargeted spell copy, controller/player scope and resolution timing |
| Invalid declarations | `TestRONFixedDrawInvalidDeclarations`: malformed draws/costs, invalid typed reads, negative/dynamic amounts and retained optional-life rejection |
| Inventory | `TestRONFixedLifeInventory`: typed life amount role |

Actual loader gain equals estimate: **five unique names and ten records**.
Jayemdae Tome contributes six reprints; each other new name contributes one.
Final: **949/2,445 compiled records**, **1,496 rejected records**,
**285 compiled unique names**, **701 rejected unique names**. No accepted-record
regression; only two remaining first diagnostics change (Jalum Tome, Sindbad).
Whole-collection strict loading still exits 1 and publishes no partial registry.
Compilation and gameplay remain separate; these tests do not prove full
collection behavior.

## Next work and gates

1. Spell-cast life triggers and fixed paid continuations: seven estimated unique
   unlocks after fresh event/payer/complete-record checks.
2. Temporary animation: Mishra's Factory, Jade Statue, Thelonite Druid; up to
   three unlocks with type/base P/T, timing, duration and cost dependencies.
3. Discard action prerequisite: Jalum Tome plus a fresh full inventory of
   discard users. Jalum Tome now exposes this next blocker directly; do not
   select it from first-error frequency alone.

No runtime cutover or deletion gate is satisfied. Complete RON gameplay,
embedded production startup, native startup smoke tests, Go-card/generator
removal and whole-repository unused-DSL caller audit/removal remain open.
