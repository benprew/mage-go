# Composable DSL Outstanding Implementation Work

## Current Slice: Typed Fixed-Total Chosen Damage

Delivery step 3 T adds the first complete typed chosen-damage adapter after the
recorded announcement and simultaneous execution prerequisites. Fresh API,
caller, test, and tracker searches confirmed the preferred damage and event
slices and all recorded cast/activation routes already exist. No production
card changed.

Implemented API:
`DealAllocatedDamage(DamageSourceRef, *DamageTargetsBinding, int) ResolvingEffect`,
also exported by `dsl`. The positive integer total preserves announcement timing
without treating a resolution value as an announcement value. The declaration
is the direct body of a nonmodal spell or activation with one mixed damage-target
group. The shared division detector supplies its fixed total to existing
announcement and payment services. The adapter resolves the source once, reads
only legal declared recipient incarnations and their stored shares, and submits
one batch to `Game.DealSimultaneousDamage`. It does not add a damage executor,
mutable allocation declaration, payment transaction, or global result slot.
Inspection exposes source, recipients, total, and damage AI metadata. Existing
scope validation and a direct-body admission check reject invalid declarations
at registration. Unsupported contexts are rejected instead of silently skipping
announcement. Sealed source and recipient roles remain intact.

`TestAllocatedDamageAnnouncementAndBatch` first failed because the API was absent.
`TestAllocatedDamageDeclarations/impossible_minimum` then failed because a required
minimum larger than the total was accepted. A zero-value recipient declaration
then exposed two registration panics, including the shared target-group bounds
inspection. Nil declaration guards now reject it without a panic. These
regressions pass. Acceptance covers spell and activation announcement before payment, impossible selected counts,
illegal/duplicate/absent/phased targets, failed payment rollback, optional empty
groups, departed and returned source LKI, source controller identity, lifelink,
deathtouch, lost illegal shares, protection, finite prevention, redirection and
replacement preparation before execution, complete-batch event/callback
observations, absent source, response/copy/clone map isolation, retargeted copies,
and ordinary/alternate/prepared/Paradigm cast routes. Compile-failure fixtures
reject player sources, arbitrary recipient sets, and resolution-value totals.
A public DSL declaration checks inspection and registration; a synthetic
`gametest` card proves chosen shares and state-based death. No production card is
used as an implementation fixture. The reference example compiles in the DSL
test.

Complete Flames of the Firebrand (`data/JMP.json`), Pyrotechnics, and Fireball
(`data/4ED.json`) text was rechecked against the inventories and JSON records.
Flames of the Firebrand and Pyrotechnics can use this fixed-total declaration;
their two production `DealDividedDamage` constructions in Jumpstart and Legends
remain unchanged and need card wiring. Fireball's dynamic equal division and
per-target payment are separate work. No author export was removed and no gap
record was closed: an adapter does not complete an unmigrated card. Unrelated
tracked and untracked files are preserved.

Excluded: dynamic announcement values, nested/modal/triggered allocations,
multiple allocation instructions or recipient groups, allocation result
bindings, Fireball's equal division, prepared mana-payment fixes, and production
card migration. Next dependency-ready work is the allocation ownership and
announcement contract for nested or modal bodies, before extending this adapter.
No second slice started.

Validation passes: focused contracts and synthetic integration with `-count=1`;
`go test ./pkg/mage/... -short -count=1`; DSL registration and closed-role checks
with `-count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`;
full `make test`; `make lint` with zero issues; and `git diff --check`.
Historical missing-package and C-metadata failures did not recur. No validation
blocker remains. The post-lint diff was reviewed.

## Completed Slice: Simultaneous Legacy Divided Damage

Delivery step 3 T completes the execution prerequisite for chosen damage.
Fresh API, caller, tracker, and test searches confirmed that damage identity,
singular damage, source attack/block, filtered death, division normalization,
and all recorded announcement routes already exist. The legacy executor still
applied each share sequentially. This slice fixes that shared contract before a
typed allocation adapter is exposed. No production card changed.

`mage.DealDividedDamage(total mage.ValueSource) mage.Effect` retains its signature
and metadata. Its executor captures the resolving damage source once, builds
player/permanent assignments from the surviving positive announced shares, and
uses `Game.DealSimultaneousDamage`. Replacement and prevention prepare all shares
before execution. Damage and life-gain events and damage callbacks see the
complete batch. The existing stack resolver handles target legality and
incarnation checks; illegal shares are lost, not redistributed. Departure LKI
supplies source characteristics and controller. The legacy unknown-source ID
fallback remains. This slice adds no role, binding, declaration state,
inspection node, registration rule, payment transaction, or damage service.

`TestDividedDamageBatchObservers` first failed for both synthetic spell and
activation resolution because event observers and callbacks saw partial damage.
`TestDividedDamageBatchPreparesBeforeExecution` first failed because later
replacement processing saw an executed share. These regressions now pass.
Acceptance also covers redirection followed by prevention, finite shields,
sacrificed/returned source LKI, lifelink and deathtouch, damage-event source
identity, copy/clone isolation, absent/returned/phased/protected/prevented
recipients, empty and nonpositive shares, and legacy unknown-source IDs.
Existing normalization and all announcement-route tests cover payment rollback,
map isolation, modal selection, and division admission. The synthetic gametest
`TestDividedDamage` passes. The reference example builds independently.

Complete Flames of the Firebrand (`data/JMP.json`), Pyrotechnics, and Fireball
(`data/4ED.json`) text was checked against the inventories and gap records.
Fireball's equal division at resolution is a separate rule. A fresh caller
search finds two production `DealDividedDamage` constructions:
`cards/jumpstart/spells.go` (Flames of the Firebrand) and
`cards/legends/spells.go` (Pyrotechnics). Both remain unchanged and still need
typed allocation/card wiring. No export was removed. No inventory or JSON gap
record is closed: an engine execution fix does not complete an unmigrated card.
Unrelated tracked and untracked work is preserved.

Excluded: typed allocations, independent groups, multiple division instructions,
Fireball's rule, general replacement-pipeline extensions, prepared mana payment,
and production migration. Next dependency-ready work is a typed chosen-damage
allocation for one recorded damage-recipient target group, after a fresh audit
of its announcement, inspection, and copying contracts. No second slice started.

Validation passes: focused `TestDividedDamageBatch*`, normalization, activation,
hand, alternate-zone, copy-cast, mixed-batch, and source-identity contracts with
`-count=1`; synthetic gametest `TestDividedDamage` with `-count=1`;
`go test ./pkg/mage/... -short -count=1`; DSL registration and closed-role checks
with `-count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`;
full `make test`; `make lint` with zero issues; and `git diff --check`.
Historical missing-package and C-metadata failures did not recur. No validation
blocker remains. The diff was reviewed after lint.

## Completed Slice: Card-Copy Division Admission

Delivery step 3 T completes the remaining card-copy announcement prerequisite
for chosen damage. Fresh API, tracker, caller, and test searches confirmed the
preferred damage identity, singular damage, source attack/block, filtered death,
normalization, activation, hand, and alternate-zone slices already exist.
Prepared and Paradigm copies still accepted illegal targets and impossible
positive-share divisions. This slice fixes that shared admission contract;
it does not add a typed adapter or change production cards.

`CastPreparedSpellCopy(playerID, permID uuid.UUID, spellFactory func() Card) error`
and `CastParadigmCopyFromExile(playerID uuid.UUID, name string) (*StackObject, error)`
retain their signatures. The unprepared-allocation branch of shared stack
construction validates nonmodal targets and uses `prepareDamageDivision` before
publishing a cast. Existing modal selection validates its own targets. Nonmodal
copy target prompts use stored X. The stack clones the allocation map. Ordinary
and alternate casts keep their existing prepayment prepared-allocation path.
No public role, binding, inspection rule, declaration state, replacement service,
executor, or AI metadata is added. The unused private post-admission chooser was
removed; all production author exports remain.

`TestCopyCastDamageDivisionAdmission` first failed for all eight invalid shapes
on both routes: negative/zero/insufficient totals, absent/phased/duplicate/wrong
category targets, and a missing required target. Acceptance also covers empty
optional target lists, selected and inactive modes, prepared X target bounds,
retained prepared state and exile originals, zero failure event traces, allocation
before publication, response/copy/clone map isolation, and announced shares at
resolution. Existing division tests cover normalization, overflow, resolution
legality, and lost shares. LKI, prevention, and replacement remain in the shared
damage service. This change adds no new damage execution or simultaneous-action
contract.

Complete Flames of the Firebrand (`data/JMP.json`), Pyrotechnics, and Fireball
(`data/4ED.json`) texts were checked against the relevant inventories and gap
records. Fireball requires equal division at resolution, a separate contract.
Two production `DealDividedDamage` constructions remain in Jumpstart and Legends.
Prepared factory and direct copy-cast callers remain in Secrets of Strixhaven;
Paradigm keeps its existing engine call path. All are unchanged. Typed divided
allocation declarations and card wiring remain open; no JSON record changed or
card was certified complete. Unrelated untracked work is preserved.

Excluded: prepared mana payment, casting-permission audits, typed grouped
allocations, multiple division instructions, simultaneous legacy divided damage,
Fireball's rule, and production migration. Next dependency-ready work is the
simultaneous legacy divided-damage execution prerequisite after a fresh audit,
before exposing a complete typed adapter. No second slice was started.

Validation passes: focused `TestCopyCastDamageDivision*` and the existing
activation/hand/alternate/normalization division and prepared/Paradigm tests
with `-count=1`; `go test ./pkg/mage/... -short -count=1`; DSL registration and
closed-role checks with `-count=1`; `go test ./... -short -count=1`;
`go build ./...`; `go vet ./...`; full `make test`; `make lint` with zero issues;
and `git diff --check`. The standalone reference example builds. Lint made no
additional changes, and the post-lint diff was reviewed. Historical missing-package
and C-metadata failures did not recur. No validation blocker remains.

## Completed Slice: Alternate-Zone Division Before Payment

Delivery step 3 T completes the shared cast-from-zone chosen-damage announcement
prerequisite. Fresh API, caller, test, and tracker searches confirmed that the
preferred damage identity, singular damage, attack/block, filtered death,
normalization, activation announcement, and ordinary hand-cast slices exist.
The shared alternate casting path still allocated after payment and omitted
nonmodal target validation. This slice corrects that engine contract; it does
not expose an incomplete typed adapter.

`CastCardFromZoneWithoutPaying`, `CastCardFromExileWithoutPaying`,
`CastCardFromZoneWithAlternateCost`, `CastCardWithAlternateCost`, and
`CastExiledCardWithPermission` retain their signatures. Their common internal
path uses existing random/opponent target acquisition, action target validation,
modal selection, `prepareDamageDivision`, spell payment transactions, and
prepared stack allocation storage. It validates and locks the selected effects'
division before cost preparation, mana abilities, or payment. The allocation
stays local until successful payment and is copied into the stack object.
No role, declaration state, binding, inspection rule, executor, replacement
service, or AI metadata is added. All production cards and exports are retained.

`TestAlternateDamageDivision*` first failed because life/mana payment and zone
removal preceded the allocation prompt, impossible divisions and illegal targets
were admitted, failed payment skipped announcement, and modal division occurred
after payment. Acceptance covers hand, graveyard, library, and exile; free,
supplied, registered, flashback, and exile-permission routes; foreign ownership;
life and automatic mana payment; positive/zero/negative totals; optional empty
target lists; absent/phased/duplicate/wrong-category targets; combined
unaffordable costs; selected versus inactive mode effects; response-map,
copy, and clone isolation; resolution of the announced amount; retained exile
permissions on failure; and flashback resolution destination. Existing division
contracts cover normalization, overflow, resolution target legality, and lost
shares. Damage LKI, replacement ordering, and execution remain in their existing
services. The reference contains signatures and a compiling synthetic example.

Complete Flames of the Firebrand (`data/JMP.json`), Pyrotechnics, and Fireball
(`data/4ED.json`) texts were checked against the inventories and Jumpstart gap
record. Chosen division differs from Fireball's equal division at resolution.
Two production `DealDividedDamage` constructions remain in Jumpstart and Legends,
unchanged. No gap record or card is completed; typed divided-target declarations
and card wiring remain. No JSON record changed. Unrelated untracked files are
preserved.

Excluded: prepared/Paradigm copy casting, typed allocations, independent grouped
allocations, multiple division instructions, simultaneous legacy divided damage,
Fireball's equal-division rule, general casting-permission audits, and production
migration. Next dependency-ready work is the prepared/Paradigm division
announcement prerequisite after a fresh audit, before exposing a typed adapter.
No second slice was started.

Validation passes: focused `TestAlternateDamageDivision*`,
`TestSpellDamageDivision*`, `TestActivationDamageDivision*`, and
`TestDamageDivision*` with `-count=1`; `go test ./pkg/mage/... -short -count=1`;
DSL registration/closed-role checks with `-count=1`;
`go test ./... -short -count=1`; `go build ./...`; `go vet ./...`;
full `make test`; `make lint` with zero issues; and `git diff --check`.
The standalone reference example builds. Its first build found a wrong package
qualifier for `ParseManaCost`; the reference now uses `core.ParseManaCost`.
Lint adjusted only the new test's import spacing; focused tests passed again.
The post-lint diff was reviewed. Historical missing-package and C-metadata
failures did not recur. No validation blocker remains.

## Completed Slice: Ordinary Hand-Cast Division Before Payment

Delivery step 3 T completes the ordinary hand-casting portion of the next
chosen-damage announcement prerequisite. Fresh API, caller, tracker, and test
searches confirmed that damage identity, singular damage, source attack/block,
filtered death, positive normalization, and activation division already exist.
The ordinary spell path still chose allocations after payment and admitted
impossible divisions. This is an engine contract correction, not a new adapter.

`Game.CastSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID,
xValues ...int) error`, `Game.CastSpellByID(playerID, cardID uuid.UUID,
targets []uuid.UUID, xValue int) error`, and the ordinary grouped hand-cast
entry retain their signatures. Shared internal `prepareDamageDivision` reuses
the existing amount reader, normalizer, and player choice service. It runs after
target validation and selected-mode acquisition, before cost calculation,
mana abilities, or payment. A prepared flag distinguishes an empty allocation
from an unprepared allocation. The stack service copies the saved map after
successful payment without another prompt or amount evaluation. Existing
activation behavior and all legacy exports remain compatible. No new role,
declaration state, scope rule, executor, result binding, or AI metadata is added.

`TestSpellDamageDivision*` first failed for post-payment life-based totals,
impossible divisions spending costs and sending cast events, failed payment
skipping announcement, and modal payment timing. Acceptance covers positive,
zero, and negative totals; optional empty target lists; life and mana payment;
automatic mana abilities; combined unaffordable costs; selected versus inactive
mode effects; absent, phased, duplicate, and wrong-category targets; and response,
copy, and clone map isolation. Resolution preserves the announced total when
payment changes its inputs. Existing `TestDamageDivision*` covers normalization,
overflow, illegal target shares, phasing at resolution, and copy/clone isolation.
The slice changes announcement only; LKI, replacement ordering, simultaneous
execution, and damage events remain in their existing services.

Complete Flames of the Firebrand (`data/JMP.json`), Pyrotechnics, and Fireball
(`data/4ED.json`) texts were checked. The Jumpstart inventory and gap record
require chosen allocations at announcement. Fireball instead needs equal
division at resolution. Synthetic spells supply all new integration cases.
Fresh searches found two production `DealDividedDamage` constructions, in
Jumpstart and Legends, both unchanged. No card or gap record is completed;
typed divided targets and production wiring remain. No JSON record changed,
so no schema validation is required. Unrelated untracked work is preserved.

Excluded: alternate-zone casting, prepared/Paradigm copy casting, typed
allocation declarations, multiple independent division instructions,
simultaneous legacy divided damage, Fireball's equal-division rule, and card
migration. Next dependency-ready work is alternate-zone division admission
and prepayment choice before exposing a typed divided-damage adapter.
No second slice was started.

Validation passes: focused `TestSpellDamageDivision*`,
`TestActivationDamageDivision*`, and `TestDamageDivision*` with `-count=1`;
`go test ./pkg/mage/... -short -count=1`; the DSL registration/closed-role
checks with `-count=1`; `go test ./... -short -count=1`; `go build ./...`;
`go vet ./...`; full `make test`; `make lint` with zero issues; and
`git diff --check`. Lint first reported a test variable that shadowed `copy`;
it was renamed and focused tests and lint passed again. The post-lint diff
was reviewed. Lint only adjusted the new test's import spacing.
Historical missing-package and C-metadata failures did not recur.
No validation blocker remains.

## Completed Slice: Activation Division Before Payment

Delivery step 3 T completes the activation announcement prerequisite for
legacy chosen damage. Fresh API, caller, test, and tracker searches confirmed
that source/recipient identity, singular damage, source attack/block events,
filtered death, and positive allocation normalization already exist. The next
missing contract was the post-payment activation allocation prompt.

`Game.ActivateAbilityByIndex(playerID, permanentID uuid.UUID, abilityIndex int,
targets []uuid.UUID) error` keeps its signature. Internal
`prepareActivationDamageDivision` uses the selected effects, existing total
reader, normalizer, and player choice service after target validation and before
cost preparation. Impossible nonempty divisions fail before choices, payment,
or events. Negative calculated totals become zero. A declaration that permits
zero targets can choose none without an allocation prompt (CR 107.1b–c).
Payment cannot change the saved total or prompt again. The map belongs to the
stack instance and uses existing copy/clone isolation. Failed payment publishes
no stack allocation. No public adapter, role, executor, declaration state,
scope rule, or AI metadata was added.

`TestActivationDamageDivision*` first failed for post-payment life-based totals,
source sacrifice before choice, impossible divisions spending costs and sending
events, payment failure skipping announcement, and modal payment timing.
Acceptance covers mana/life/tap payment, source sacrifice, dynamic totals,
positive/zero/negative totals with no targets, insufficient shares, illegal,
absent, phased, and duplicate targets, combined unaffordable costs, selected
versus inactive modes, response-map isolation, stack copies, and game clones.
Existing `TestDamageDivision*` also passes and covers normalization, overflow,
phasing at resolution, and losing illegal target shares. This slice changes
announcement only; LKI, damage replacement ordering, and damage execution use
the existing services.

Complete Flames of the Firebrand, Pyrotechnics, and Fireball texts were checked
in `data/JMP.json` and `data/4ED.json`. The Jumpstart inventory and gap record
require chosen announcement allocations for Flames of the Firebrand. The plan
names Pyrotechnics and Fireball as contrasting chosen/equal division cases;
the Fourth Edition inventory and gap record do not list them separately.
Their spell requirements remain open: this slice uses synthetic activations.
Fresh production searches found two `DealDividedDamage` constructions, in
Jumpstart and Legends, both unchanged. No production card, inventory, or JSON
gap record is completed. All legacy exports remain. The reference includes a
compiling synthetic activation example from the contract tests.

Excluded: spell casting and alternate casting admission, typed divided-target
amounts, multiple independent division instructions, simultaneous legacy
divided damage, Fireball's equal-division rule, and production wiring.
Next dependency-ready work is spell prepayment division choice and admission,
before a typed divided-damage adapter. No second slice was started. Unrelated
tracked and untracked files remain unchanged.

Validation passes: focused `TestActivationDamageDivision*` and
`TestDamageDivision*` with `-count=1`, `go test ./pkg/mage/... -short -count=1`,
the DSL registration/closed-role run with `-count=1`,
`go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, full
`make test`, `make lint` with zero issues, and `git diff --check`.
The post-lint diff was reviewed; lint adjusted test import spacing only.
Historical missing-package and C-metadata failures did not recur.
No validation blocker remains.

## Completed Slice: Positive Chosen-Damage Allocation Normalization

Delivery step 3 T completes one shared prerequisite for divided-target
announcement. Fresh API, test, caller, and tracker searches confirmed that the
preferred source/recipient identity, explicit singular damage, source attack and
block events, and filtered other-creature death slices already exist. The typed
divided-damage adapter is absent. The legacy shared normalizer accepted missing
or zero shares, used an illegal all-to-first fallback, and could accept an
overflowed sum. No adapter was exposed over that incomplete contract.

`sanitizeDamageDistribution(map[uuid.UUID]int, []uuid.UUID, int)
map[uuid.UUID]int` now preserves a valid allocation in a new map, requires every
chosen target to receive at least one, and checks the total by subtraction.
Malformed responses receive one per target plus the remainder on the first
announced target. Duplicate or nil target IDs, an empty list, or a total smaller
than the target count have no representable allocation and return nil.
`BasePlayer.ChooseDamageDistribution` uses the same normalization boundary.
All public signatures remain unchanged. This does not certify an impossible
announcement as legal; admission and prepayment choice locking remain open.

`TestDamageDivision*` first failed for missing/zero shares, malformed fallback,
integer overflow, and spell/activation allocations. Acceptance covers valid
shares, nil responses, missing/unknown targets, negative/zero/excess/short shares,
maximum integer totals, invalid target shapes, exhaustive one-to-three-target
allocations, unchanged input maps, default choices, synthetic spells and
activations, phasing, loss of illegal target shares, and independent stack-copy
and game-clone allocations. Normalization reads IDs and amounts only; it neither
resolves objects nor changes payment, LKI, replacements, events, or damage
execution. No sealed role, declaration state, scope rule, or AI metadata was
added. Existing registration and role checks remain required.

Complete Flames of the Firebrand and Pyrotechnics texts in `data/JMP.json` and
`data/4ED.json` were checked against the inventories and gap requirements.
Both require positive chosen allocations. Fireball's complete text in
`data/4ED.json` requires a different equal-division contract. Fresh production
searches found two `DealDividedDamage` constructions, in Jumpstart and Legends;
both remain unchanged. No production card or gap record is completed. The
reference includes the exact compiling synthetic engine declaration from the
integration test; no typed divided-damage author export is claimed.

Excluded: typed divided target groups and amounts, prepayment announcement
choices, transactional rejection of impossible target/total shapes, alternate
casting admission, simultaneous legacy divided damage, and card migration.
Next dependency-ready work is the T prepayment division choice and admission
contract, before a typed divided-damage adapter. No second slice was started.
Unrelated tracked and untracked work is preserved.

Validation passes: focused contracts with `-count=1`,
`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role
run with `-count=1`, `go test ./... -short -count=1`, `go build ./...`,
`go vet ./...`, full `make test`, `make lint` with zero issues, and `git diff --check`.
The post-lint diff was reviewed. Lint made no additional source changes, and
no validation blocker remains.

## Completed Slice: Typed Acceptance for X-Counter Alternative Trees

Delivery step 3 P extends the existing
`OptionalCostInto(inner Cost, prompt string, result *NumberBinding) Cost` to
`EitherCost(options ...Cost) Cost` trees with uncaptured
`RemoveXCountersFromSourceCost(counter CounterType) Cost` leaves. Fresh API,
caller, tracker, and test searches confirmed that the preferred damage identity,
singular damage, source attack/block, filtered death, legacy optional X tree,
and typed fixed-counter tree slices already exist. No function was recreated.
The extension reuses inspection, X conversion, choice locking, source-incarnation
checks, combined counter preflight, and instance-local payment frames.

The new `TestOptionalXCounterAlternativeResult*` contracts first failed for
registration rejection, missing acceptance outputs, commit-time choice, stale
source acceptance, and nested branch selection. Acceptance covers announced
versus ambient X, zero, decline, unaffordability, nested and sole payable branch
selection, source departure/return, initial absence, phasing and phase-in,
combined shortages before mana/life/counter spending or events, immutable
declarations, interleaved transactions, independent game clones, and stack copies.
Synthetic public DSL declarations cover activations, spell additional costs,
registration rejection, scope, and dominance. Existing role compile checks apply;
no new role or AI metadata is introduced. Payments are nontargeted and need
current identity for positive removal, not LKI. Replacement ordering and
simultaneous actions are unchanged.

Chamber Sentry's complete text in `data/JMP.json` was checked against the
Jumpstart inventory and gap record. It supplies the announced X-counter cost
requirement; the optional alternative wrapper is synthetic. No card is completed
by this adapter. Its entry, damage, graveyard ability, and all wiring still need
separate migration and audits. Fresh production searches found four `EitherCost`
constructions, one legacy `OptionalCost`, and no `OptionalCostInto`. All remain
unchanged. No inventory or JSON record changed; unrelated untracked work remains.

Excluded: captured leaves, mixed fixed/X or other families, nested optional gates,
general payment-order choice, mutations during commit, and production wiring.
Next dependency-ready work is T divided-target announcement after a fresh audit.
No second slice was started.

Validation passes: focused contracts with `-count=1`,
`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role
run with `-count=1`, `go test ./... -short -count=1`, `go build ./...`,
`go vet ./...`, full `make test`, `make lint` with zero issues, and
`git diff --check`. The public reference declaration compiles and validates in
the synthetic activation test. The post-lint diff was reviewed; lint adjusted
import spacing only. Historical workspace blockers did not recur.

## Completed Slice: Typed Acceptance for Fixed-Counter Alternative Trees

Delivery step 3 P extends `OptionalCostInto(inner Cost, prompt string,
result *NumberBinding) Cost` to an `EitherCost(options ...Cost) Cost` tree whose
leaves are uncaptured `RemoveCountersCost(counter CounterType, count int) Cost`
costs. Fresh code, caller, test, and tracker checks confirmed that damage
identity, explicit `DealDamageFrom`, source attack/block events, filtered death,
and the legacy optional alternative prerequisite already exist. No public
signature, sealed role, counter executor, target, AI metadata, or declaration
state was added. The extension reuses the existing inspection graph, choice
locking, source-incarnation lock, combined-total preflight, and instance frames.

`TestOptionalCounterAlternativeResult*` first failed for registration rejection,
missing acceptance outputs, commit-time choices, stale source acceptance, and
branch selection. A later regression failed for a selected enclosing alternative
that bypassed wrapper locking; that branch now uses the same path. Acceptance
covers nested selection, fixed zero, decline and unaffordability, source
departure/return, absence, phasing and phase-in, combined
shortage before resource spending or payment events, immutable declarations,
interleaved payments, stack copies, and independent game clones. Synthetic DSL
cards test accepted and unaffordable activations together, spell additional costs
with an absent source, and registration scope/dominance. Invalid declarations
retain rejection for empty trees, invalid counter types, negative counts,
captured/X/mixed leaves, nil results, empty prompts, and resolution-time capture.
These payments are nontargeted and require current identity, not LKI. Replacement
ordering and simultaneous actions are unchanged.

Osai Vultures' full text in `data/LEG.json` was checked against the Legends
inventory and gap record. It supplies the fixed-counter cost requirement; the
optional alternative wrapper is synthetic. Fresh production searches found four
`EitherCost` constructions, one legacy `OptionalCost`, and no `OptionalCostInto`.
All remain unchanged. No card or gap record is completed: card upkeep and
remaining wiring need separate migration and audits. No inventory or JSON record
changed. Unrelated untracked work is preserved. Previously noted Cauldron of
Essence and Saproling-sacrifice tracker corrections are already present.

Excluded: captured/X/mixed leaves, nested optional gates, general payment-order
choice, mutations during commit, and production wiring. Next dependency-ready
work is typed acceptance around uncaptured X-counter alternatives or T divided
target announcement. No second slice was started.

Validation passes: focused contracts with `-count=1`,
`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role
run with `-count=1`, `go test ./... -short -count=1`, `go build ./...`,
`go vet ./...`, full `make test`, `make lint` with zero issues, and
`git diff --check`. The post-lint diff was reviewed. Lint only adjusted import
spacing in the new engine test; no semantic changes or validation blockers
remain. Historical workspace failures did not recur.

## Completed Slice: Optional X-Counter Alternative Tree

Delivery step 3 P now locks the legacy engine `OptionalCost(inner Cost, prompt
string) Cost` around an `EitherCost(options ...Cost) Cost` tree of uncaptured
`RemoveXCountersFromSourceCost(counter CounterType) Cost` leaves. Fresh API,
caller, test, and tracker checks confirmed that damage identity, singular typed
damage, source attack/block events, filtered death, and fixed-counter optional
alternatives already exist. This completes a wrapper-payment prerequisite,
without adding a public signature, role, executor, target, result binding,
replacement service, or AI metadata. All production callers remain unchanged.

`optionalCost.resolveActionCost` now resolves an alternative tree before
optional affordability. Direct total-payment validation then reuses
`lockPaymentCosts`, isolated validation, source-incarnation locks, and combined
counter preflight. Announced X is used for activation and spell announcement;
direct engine preparation uses current X. Choices and X remain fixed at commit.

`TestOptionalEitherXCounterPayment*` first failed for lost acceptance, ambient X,
stale source acceptance, partial combined removal, and clone interference.
Acceptance covers decline, unaffordability, sole payable branch selection,
two nested branch choices without repetition, zero, absence, source departure
and return, phasing and phasing back in, combined shortage without resource
spending or published events/outputs, unchanged declarations and caller maps,
interleaved transactions, announced versus ambient X, and independent clones.
`TestOptionalEitherXCounterEngineDeclaration` validates the reference example,
pays at activation, copies without repaying, and resolves independently in a
clone. Registration still rejects invalid counter types, empty alternatives,
and `OptionalCostInto` around alternatives. These payments are nontargeted and
need current source identity, not LKI. Replacement ordering and simultaneous
actions are unchanged.

Chamber Sentry's complete text in `data/JMP.json` was checked against the
Jumpstart inventory and gap record. It supplies the announced X-counter
requirement; optional alternative wrappers are synthetic. Fresh searches found
four production `EitherCost` constructions, one legacy `OptionalCost`
construction, one `RemoveXCountersFromSourceCost` construction, and no
`OptionalCostInto` construction. No production caller uses this wrapper shape.
No card, inventory, or JSON record changed or closed. Entry, damage, and
graveyard card wiring require separate migration and audits. Unrelated
untracked work is preserved.

Excluded: captured leaves, mixed leaf families, nested optional gates, typed
acceptance around alternatives, general payment-order choice, mutations during
commit, and production card wiring. Next dependency-ready work is a separate P
captured-wrapper audit or T divided-target announcement prerequisite. No second
slice was started.

Validation passes: focused contracts with `-count=1`,
`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role
run with `-count=1`, `go test ./... -short -count=1`, `go build ./...`,
`go vet ./...`, full `make test`, `make lint` with zero issues, and
`git diff --check`. The post-lint diff was reviewed. Lint only added import
spacing in the new test file; no semantic source changes or validation blockers
remain. Historical workspace failures did not recur.

## Completed Slice: Optional Fixed-Counter Alternative Tree

Delivery step 3 P completed a wrapper-payment prerequisite: direct engine
total-payment preparation now locks a legacy `OptionalCost` around an
`EitherCost` tree of uncaptured fixed-counter leaves. Preferred damage identity,
explicit singular damage, attack/block, and filtered death functions already
exist. This slice reuses `lockPaymentCosts`, isolated validation, fixed-counter
incarnation locks, and combined-total preflight. Acceptance and selected branches
occur once. No new role, executor, public signature, result binding, target,
replacement service, or AI metadata is added. `OptionalCost` remains an engine
compatibility API; it is not exported by `dsl`. Production callers are preserved.

`TestOptionalEitherFixedCounterPayment*` first failed for repeated optional
choices, returned/absent/phased/short source acceptance, and partial removal on
combined shortage. Acceptance covers decline, affordability, automatic selection
of a sole payable branch, two nested branch choices without repetition, fixed
zero, phase-out/in, source identity, unchanged templates and caller maps,
interleaved payments, clone isolation, and failure without resource spending or
published outputs. `TestOptionalEitherFixedCounterEngineDeclaration` validates
the compiling reference example, pays on activation, copies the stack object
without repaying, and resolves in independent games. Declaration tests retain
rejection for negative counts, invalid counter types, empty alternatives, and
`OptionalCostInto` around alternatives. These costs are nontargeted and need
current source identity, not LKI. Replacement and simultaneous-action behavior
is unchanged.

Osai Vultures' complete text in `data/LEG.json` matches the Legends inventory
and gap record: its activated cost removes two carrion counters and its effect
grants +1/+1 until end of turn. This supplies the fixed-payment requirement;
optional alternatives are synthetic. Card upkeep and remaining wiring require
separate audits. Fresh searches found 18 production `RemoveCountersCost`
constructions, four `EitherCost` constructions, one legacy `OptionalCost`
construction, and no `OptionalCostInto` construction. No card, inventory, or
JSON record changed or closed. Unrelated untracked work is preserved.

Excluded: captured/X/mixed leaves, typed acceptance capture around alternatives,
general payment-order choice, mutations during commit, and card wiring. Next
dependency-ready work is a separate P captured-wrapper audit or T divided-target
announcement prerequisite. No second slice was started.

Validation passes: focused tests with `-count=1`,
`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role
run with `-count=1`, `go test ./... -short -count=1`, `go build ./...`,
`go vet ./...`, full `make test`, `make lint` with zero issues, and
`git diff --check`. The post-lint diff was reviewed; no automatic source
changes or validation blockers remain. Historical workspace failures did not recur.

## Current Slice: Optional X-Counter Branch in an Alternative Payment

Delivery step 3 P now recursively resolves optional uncaptured X-counter leaves
inside `EitherCost(options ...Cost) Cost` before affordability. Fresh API, tests,
caller, and tracker checks confirmed that damage identity, explicit singular
damage, source attack/block, filtered death, and optional fixed alternatives
already exist. The change reuses `optionalCost.resolveActionCost`, announcement
locking, fixed-counter source-incarnation locks, combined-total preflight, and
instance-local outcome frames. No public signature, sealed role, executor,
target, replacement service, or AI metadata was added. All production callers
remain unchanged.

`TestEitherOptionalXCounterPayment*` first failed for repeated choices, stale
source acceptance, partial combined removal, lost outcomes, and ambient X in
place of announced X. Acceptance covers zero and positive X, decline,
unaffordability, announced versus ambient X, source departure/return, absence,
phasing, phase-out/in, combined shortage without resource spending or published
outputs, two nested branch choices without repetition, unchanged templates and
caller maps, interleaved transactions, and independent game clones.
`TestEitherOptionalXCounterPublicDeclaration` compiles and validates the reference
example, pays at activation, copies the stack object, and resolves independently
in a clone. `TestEitherOptionalXCounterResultDoesNotDominate` retains registration
rejection for a result read outside the alternative branch. These costs are
nontargeted and require current identity for positive removal, not LKI.
Replacement ordering and simultaneous actions are unchanged.

Chamber Sentry's complete text in `data/JMP.json` was checked against the Jumpstart
inventory and gap record. It requires announced X source-counter payment;
optional alternative wrappers are synthetic. Its entry, damage, and graveyard
wiring still require separate audits and migration. Fresh production searches
found four `EitherCost` constructions, one `RemoveXCountersFromSourceCost`
construction, one legacy `OptionalCost` construction, and no `OptionalCostInto`
construction. No production card, inventory, or JSON gap record changed or closed.
Unrelated untracked work is preserved.

Excluded: captured inner costs, optional-around-alternative wrappers, public
result reads outside an alternative branch, general payment-order selection,
mutations during commit, and production wiring. Next dependency-ready work is a
separate T divided-announcement prerequisite or P captured-wrapper audit.
No second slice was started.

Validation passes: focused contracts with `-count=1`, engine short tests with
`-count=1`, DSL registration and closed-role checks with `-count=1`, repository
short tests with `-count=1`, `go build ./...`, `go vet ./...`, full `make test`,
`make lint` with zero issues, and `git diff --check`. The post-lint diff was
reviewed; no automatic code changes or validation blockers remain. Historical
workspace failures did not recur.

## Current Slice: Optional Fixed-Counter Branch in an Alternative Payment

Delivery step 3 P now routes a selected `EitherCost(options ...Cost) Cost` branch containing an optional direct uncaptured `RemoveCountersCost(CounterType, int) Cost` through existing optional preparation, source-incarnation locking, and combined-total preflight. Nested alternative ancestors use the same path. `OptionalCost` and `OptionalCostInto` keep their signatures. This is an engine contract correction; announcement already flattens these choices. No public API, role, executor, declaration state, target, replacement service, or AI metadata was added. Fresh code and tracker checks confirmed that the preferred damage identity, explicit singular damage, source attack/block, filtered death, and nested fixed/X alternative slices already exist.

`TestEitherOptionalFixedCounterPayment*` first failed for repeated choices, returned/absent/phased/short source acceptance, partial combined removal, and lost isolated outputs. Acceptance covers decline, affordability, fixed zero, positive removal, phase-out/in, branch selection without commit repetition, unchanged templates and caller maps, interleaved payments, clone isolation, and failure without mana/life/counter spending or published outputs. `TestEitherOptionalFixedCounterPublicDeclaration` compiles, validates, activates, copies the stack object, and resolves independently in a clone. `TestEitherOptionalFixedCounterResultDoesNotDominate` preserves registration rejection for a result read outside an alternative branch. These costs are nontargeted and require current identity, not LKI. Replacement ordering and simultaneous actions are unchanged.

Complete Osai Vultures and Black Mana Battery texts in `data/LEG.json` were checked against the Legends inventory and gap record. They require fixed and chosen source-counter payments; their Oracle text does not require this synthetic alternative optional wrapper. Their remaining trigger/mana/card wiring still needs separate audit and migration. Fresh searches found 18 production fixed-counter constructions, four alternative-cost constructions, one legacy optional-cost construction, and no `OptionalCostInto` construction. No production card, inventory, or JSON gap record changed or closed. All legacy exports and callers remain. Unrelated untracked files are preserved.

Excluded: X inner costs, captured inner costs, optional-around-alternative wrappers, public result reads outside an alternative branch, general payment-order selection, mutations during commit, and production wiring. Next dependency-ready work is a separate P optional-X branch audit or T divided-announcement prerequisite. No second slice was started.

Validation passes: focused contracts with `-count=1`, engine short tests with `-count=1`, DSL registration and closed-role checks with `-count=1`, repository short tests with `-count=1`, build, vet, full `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed; lint only grouped imports in the new public test. Post-lint validation initially failed because the configured `GOCACHE=/private/tmp/go-build` had missing standard-library cache files; a second run confirmed the same cache failure. An isolated task cache restored engine, registration/closed-role, repository short, build, vet, and full `make test` checks without source changes or weaker tests. No validation blocker remains.

## Current Slice: Nested Uncaptured X-Counter Alternative Payment

Delivery step 3 P now recursively fixes uncaptured `RemoveXCountersFromSourceCost(CounterType) Cost` leaves inside nested `EitherCost(options ...Cost) Cost` declarations before branch affordability. Fresh API, caller, and tracker checks confirmed the preferred damage identity, singular damage, attack/block, filtered death, and nested fixed-counter prerequisites already exist. No new public API, role, frame, executor, target, replacement service, or AI metadata is introduced. Selected leaves reuse existing source-incarnation locks and combined-total preflight; all production callers remain unchanged.

`TestNestedEitherXCounterPaymentAnnouncement` first failed because an announced zero was rejected with ambient X=5. Acceptance tests cover announced versus ambient X, insufficient affordability, zero, positive removal, departure/return/phasing, phase-out/in, combined shortage without mana/life/counter spending or outputs, two branch choices without repetition, shared immutable declarations, interleaved transactions, and independent cloned games. `TestNestedEitherXCounterPaymentPublicDeclaration` compiles and validates the reference example and copies the paid activation and resolves both instances independently in a clone. Existing registration, scope, dominance, and sealed-role checks remain applicable. These nontargeted costs require current identity for positive removal, not LKI; replacement and simultaneous-action contracts are unchanged.

Chamber Sentry's complete Oracle text in `data/JMP.json` was checked against the Jumpstart inventory and gap record. Its announced X payment supplies the amount requirement; nested alternatives are synthetic. Four production `EitherCost` calls and one `RemoveXCountersFromSourceCost` call remain, with no nested X alternative. No production card or gap record is completed. Card entry, damage, and graveyard wiring still require separate audits and migration. No inventory or JSON record changed. Unrelated untracked work is preserved.

Excluded: nested optional wrappers, captured leaves, general payment-order selection, mutations during commit, and production wiring. Next dependency-ready work is a separate T divided-announcement prerequisite or P nested optional-wrapper audit. No second slice was started.

Validation passes: focused contracts with `-count=1`, engine short tests with `-count=1`, DSL registration and closed-role checks with `-count=1`, repository short tests with `-count=1`, build, vet, full `make test`, `make lint` with zero issues, and `git diff --check`. Required checks passed again after adding stack-copy integration. The post-lint diff was reviewed; no automatic changes or validation blockers remain.

## Current Slice: Nested Fixed-Counter Alternative Payment

Fresh API, tests, and tracker checks confirmed the preferred damage identity, singular damage, attack/block, and filtered death slices already exist. Delivery step 3 P has a remaining nested-wrapper prerequisite. Selected nested `EitherCost(options ...Cost) Cost` branches now enter the existing leaf preparation path. Uncaptured `RemoveCountersCost(CounterType, int) Cost` leaves use source-incarnation locking and combined-total commit preflight. No new public API, role, binding, executor, target, replacement, or AI metadata is introduced. Production cards and all legacy callers remain unchanged.

`TestNestedEitherFixedCounterPaymentPreflight` first failed for returned-source acceptance, resource spending on absence/phasing/shortage, and partial counter removal on combined shortage. Isolation tests cover shared immutable declarations, independent pending payments, and cloned games. Selection tests cover two branch choices, no repeated choice at commit, fixed zero, and phase-out followed by phase-in. The public synthetic declaration compiles, validates, pays on activation, and resolves independently in a clone. Existing scope, dominance, sealed-role, and registration checks remain applicable. These costs require current source identity rather than LKI and change no simultaneous action or replacement ordering.

Complete Osai Vultures and Black Mana Battery texts in `data/LEG.json` were checked against the Legends inventory and gap record. Their fixed and chosen-counter requirements motivate the payment boundary; neither requires nested alternatives. Four production `EitherCost` calls remain, with no nested alternative. No card or gap record is completed by this synthetic engine contract. Excluded: nested optional wrappers, captured leaves, nested X, general payment-order selection, and mutations during commit. Next dependency-ready work is a separate P nested-X audit or T divided-announcement prerequisite. No second slice was started.

Validation passes: focused contracts with `-count=1`, engine short tests, DSL registration and closed-role checks, repository short tests, build, vet, full `make test`, `make lint` with zero issues, and `git diff --check`. Full tests and lint passed again after the added selection/zero contract. The post-lint diff was reviewed; no automatic code changes or validation blockers remain. Unrelated untracked work is preserved.

## Current Slice: Direct Optional Fixed-Counter Payment Locking

Delivery step 3's P wrapper prerequisite now locks acceptance and source identity for a direct uncaptured fixed-counter optional cost in engine total-payment preparation. Fresh searches confirmed that damage identity, singular damage, source attack/block, filtered death, and direct optional X locking already exist. `OptionalCost(inner Cost, prompt string) Cost`, `OptionalCostInto(inner Cost, prompt string, paid *NumberBinding) Cost`, and `RemoveCountersCost(CounterType, int) Cost` retain their signatures. The implementation reuses `lockPaymentCosts`, fixed-counter incarnation locks, isolated validation, and combined-total commit preflight. No executor, role, declaration state, target, AI metadata, or public resolution-capture context is added.

`TestDirectOptionalFixedCounterPayment*` first failed for repeated choices, returned/absent/phased/short source acceptance, partial removal on combined shortage, and missing isolated outputs. Acceptance covers affordability, decline, fixed zero with a present source, absent/phased zero, phase-out/in identity, returned zero, positive removal, unchanged declaration templates, interleaved transactions, game clones, legacy flag/map isolation, and failure without resource spending or published outputs. `TestOptionalFixedCounterPublicDeclaration` validates and executes the compiling reference example through an activation, stack copy, and independent game clone. Existing counter enum, captured-cost lifecycle, scope, dominance, registration, and sealed-role validation remains applicable. These costs are nontargeted and require current source identity rather than LKI. Replacement ordering and simultaneous actions are unchanged.

Complete Osai Vultures and Black Mana Battery Oracle texts in `data/LEG.json` were checked against the Legends inventory and JSON gap record. They require fixed and chosen source-counter payments, respectively; neither requires this synthetic optional wrapper. Their remaining trigger/mana/card wiring requires separate audits. A fresh search found 18 production `RemoveCountersCost` calls, one production `OptionalCost` construction, and no production `OptionalCostInto` call or optional fixed-counter construction. All exports and callers remain. No production card, inventory, or JSON gap record changed or closed. Unrelated untracked files are preserved.

Excluded: nested choices, captured inner costs, general payment-order selection, mutations during commit, and production wiring. Next dependency-ready work is a separate T divided-announcement prerequisite or P nested-wrapper audit. No second slice was started.

Validation passes: focused contracts with `-count=1`, engine short tests with `-count=1`, DSL registration/closed-role and public synthetic declaration checks with `-count=1`, repository short tests with `-count=1`, build, vet, full `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed; no automatic code changes or validation blockers remain. Historical package/C-metadata failures did not recur. Unrelated untracked work is preserved.

## Current Slice: Direct Total-Payment Optional X Counter Locking

Delivery step 3's remaining P wrapper prerequisite fixes current X and acceptance during direct total-payment preparation for an optional uncaptured X counter cost. Fresh searches confirmed that damage identity, singular damage, source attack/block, filtered death, direct counter payments, and announced optional X already exist. `OptionalCost(inner Cost, prompt string) Cost`, `OptionalCostInto(inner Cost, prompt string, paid *NumberBinding) Cost`, and `RemoveXCountersFromSourceCost(CounterType) Cost` retain their signatures. This slice reuses `lockPaymentCosts`, fixed-counter incarnation locks, combined-total preflight, and existing result markers. It adds no executor, role, declaration state, AI metadata, or public resolution-capture context.

`TestDirectOptionalXCounterPayment*` first failed for changed ambient X, repeated choice, returned/absent/phased source acceptance, partial resources on combined shortage, and clone isolation. Acceptance covers decline, affordability, accepted zero with absent source, positive removal, legacy flags, immutable declarations, interleaved transactions, isolated cloned payments, and failure without resource spending or outputs. Existing public synthetic declaration, copy, registration, and sealed-role tests continue to apply. These payments are nontargeted and require current source identity for positive removal, not LKI; replacements and simultaneous actions are unchanged.

Chamber Sentry's complete `data/JMP.json` text was compared with the Jumpstart inventory and gap record. It requires an announced X counter payment, not optional payment; optional integration is synthetic. The inventory's entry-mana-color and graveyard-return requirements remain open. One production uncaptured-X caller remains in `cards/jumpstart/creatures.go`, and one legacy optional-cost construction remains in `cards/jumpstart/spells.go`; no production `OptionalCostInto` caller exists. No production card or gap record changed or closed. All legacy signatures and callers remain. The compiling public example remains in `docs/dsl-reference.md`, exercised by `TestOptionalXCounterPublicDeclaration`.

Excluded: nested choices, captured inner counter costs, general payment-order choice, mutations during commit, and production wiring. Next dependency-ready work is a separate T divided-announcement prerequisite or P nested-wrapper audit. No second slice was started.

Validation passes: focused contracts with `-count=1`, engine short tests with `-count=1`, DSL registration/closed-role and public synthetic declaration checks with `-count=1`, repository short tests with `-count=1`, build, vet, full `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed; automatic fixes use `maps.Copy` and import grouping, and a redundant test variable type was removed. Focused isolation tests and required engine/build/vet/full-test checks pass after those fixes. No validation blocker remains. Unrelated untracked work is preserved.

## Current Slice: Optional Announced X Counter Payment

Delivery step 3's P wrapper-payment prerequisite now resolves a direct uncaptured X counter cost inside an announcement-time optional wrapper before affordability. Fresh API, tracker, and tests confirmed that damage identity, singular damage, attack/block, and filtered death already exist. `OptionalCostInto(inner Cost, prompt string, paid *NumberBinding) Cost` and `RemoveXCountersFromSourceCost(CounterType) Cost` retain their signatures. The wrapper copy delegates X conversion to the existing contextual counter adapter; accepted costs reuse fixed-counter locks and total preflight. No new executor, role, binding format, AI metadata, or card policy is added.

`TestOptionalXCounterPaymentAnnouncement` first failed for accepted zero with different ambient X. Preflight regressions first failed for returned, absent, phased, short, and combined-short sources. Acceptance covers decline, affordability before offering, announced versus ambient X, immutable declarations, no repeated choice, resource/output rollback, interleaved payments, clone isolation, and copied synthetic activation results. The public reference example compiles and validates in `TestOptionalXCounterPublicDeclaration`. These payments are nontargeted and require current source identity for positive removal, not LKI. Replacement ordering and simultaneous actions are unchanged.

Chamber Sentry's complete `data/JMP.json` text was checked against the Jumpstart inventory and gap record. Its X counter cost supplies the amount requirement; it does not require an optional cost. The optional integration is synthetic. One production uncaptured-X caller remains in `cards/jumpstart/creatures.go`; no production `OptionalCostInto` caller was found. All callers and exports remain. No production card or gap record changed or closed. Card wiring and the card's other abilities remain separate work.

Excluded: direct total-payment optional wrappers, nested choices, captured counter costs, resolution-time optional payments, and mutations during commit. Next dependency-ready work is a separate T divided-announcement prerequisite or P wrapper audit. No second slice was started.

Validation: focused contracts with `-count=1`, engine short tests, registration/closed-role checks, repository short tests, build, vet, full `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed; lint only grouped imports in the new test file. Historical package and C-metadata blockers did not recur. Unrelated untracked work is preserved.

## Current Slice: Selected Uncaptured X Counter Alternative Payment

Delivery step 3's P wrapper-payment prerequisite extends the existing selected fixed-counter contract to direct uncaptured X options. Fresh API and caller searches confirmed that damage identity, singular damage, attack/block events, filtered death, and direct fixed/X payments already exist. `EitherCost(options ...Cost) Cost` and `RemoveXCountersFromSourceCost(CounterType) Cost` retain their public signatures. Before branch affordability checks, action preparation converts direct uncaptured X options using announced X; total-cost preparation uses current X. The selected option then reuses fixed-counter incarnation locking, isolated validation, and combined-total commit preflight. Option slices and fixed costs are copied, so declarations stay immutable.

The new preflight cases first failed for returned-source acceptance and mana/life spending on absent, phased, short, and combined-short payments. `TestEitherXCounterPaymentAmount` first failed because commit used changed ambient X. Passing contracts now cover announced versus ambient X, direct preparation, combined shortage, zero and positive removal, reused declarations, interleaved source transactions, and clone isolation. `TestEitherXCounterPaymentPublicDeclaration` validates and executes the compiling reference example through a synthetic activation and independent clone resolution. Existing declaration validation and sealed roles remain unchanged. No target, LKI, replacement, event, simultaneous-action, output-frame, or AI metadata contract is added.

Chamber Sentry's complete text in `data/JMP.json` was compared with the Jumpstart inventory and gap record. Its announced X counter cost motivates the amount requirement; it does not require this synthetic alternative. One production `RemoveXCountersFromSourceCost` caller and four `EitherCost` callers remain, with no X-counter alternative. All exports and callers are preserved. Chamber Sentry and the surrounding inventory requirements still need separate card wiring and audits. No production card or gap record changed. Unrelated untracked files are preserved.

Excluded: optional and nested alternatives, captured-X and chosen-count options, new output bindings, general payment-order choice, and mutations during commit. The next dependency-ready slice is a separate optional X payment audit or T divided-announcement prerequisite. No second slice was started.

Validation: focused tests with `-count=1`, engine short tests, registration/closed-role checks, repository short tests, build, and vet pass. Full `make test`, `make lint` with zero issues, and `git diff --check` also pass. The post-lint diff was reviewed. No validation blocker remains.

## Current Slice: Selected Fixed-Counter Alternative Payment

Delivery step 3's P wrapper-payment prerequisite now gives a selected direct fixed-counter `EitherCost` branch the existing direct counter-payment lock and commit preflight. Fresh API and tracker searches confirmed that damage identity, singular damage, source attack/block, filtered death, and direct fixed/X counter locks already exist. No public function was recreated. `EitherCost(options ...Cost) Cost` and `RemoveCountersCost(CounterType, int) Cost` keep their signatures and DSL exports. Preparation records the selected fixed branch as a direct transaction component, then uses the existing copied incarnation lock, isolated validation payment, combined-total preflight, and counter executor. No new role, frame, AI metadata, target, replacement service, or mutable declaration was added.

New `TestCounterPaymentCommitPreflight` cases first reproduced returned-source acceptance and mana/life loss on failed selected-branch payments. Passing cases cover source return, absence, phasing, shortage, and combined fixed-counter shortage before spending resources. `TestEitherCounterPaymentLockIsolationAndZero` covers zero and positive counts, interleaved transactions from one declaration, and clone isolation. `TestEitherCounterPaymentCombinedCommit` covers successful combined fixed/chosen payments and distinct counter types. `TestEitherFixedCounterPaymentSelection` proves one branch choice and no fallback or repeat at commit. `TestEitherFixedCounterPaymentPublicDeclaration` compiles the reference example, validates registration, pays through a synthetic activation, and resolves independently in a clone. Original direct-payment tests remain. These nontargeted payments require present source identity rather than LKI; no simultaneous action or replacement ordering changes.

The complete Osai Vultures and Black Mana Battery texts in `data/LEG.json` were checked against the Legends inventory and gap record. Their fixed/chosen source-counter requirements motivate the shared payment boundary; neither card requires this synthetic alternative, and neither is migrated or certified by this correction. A fresh search found 18 production `RemoveCountersCost` calls and four `EitherCost` calls, with no fixed-counter alternative among them. All callers and exports remain. No production card, inventory, or gap record changed or closed. Unrelated untracked work is preserved.

Excluded: optional wrappers, nested alternatives, X/chosen-count branches, new captured outputs, general payment-order selection, mutations during commit, and card wiring. The next dependency-ready slice is a separate P X/optional wrapper-payment audit or T divided-announcement prerequisite. No second slice was started.

Validation: focused contracts with `-count=1`, engine short tests with `-count=1`, DSL registration/closed-role and public synthetic checks with `-count=1`, repository short tests with `-count=1`, build, vet, full `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed. Historical package/C-metadata blockers did not recur. No validation blocker remains.

## Current Slice: Direct Uncaptured X Counter-Payment Locking

Delivery step 3's P prerequisite now locks uncaptured X before direct payment. Fresh API, caller, tracker, and test searches confirmed the preferred damage identity, singular damage, attack/block, and filtered-death capabilities exist. `RemoveXCountersFromSourceCost(CounterType) Cost` and its DSL export remain unchanged. The existing contextual action-cost adapter fixes announced X; direct total-payment preparation fixes current X. Positive X reuses the fixed-counter executor, incarnation lock, and combined-total preflight. Zero retains the uncaptured constructor's no-action behavior even with an absent source. No new public role, capture frame, executor, or AI metadata is introduced.

Failing `TestCounterPaymentCommitPreflight` cases first demonstrated returned-source acceptance and mana/life loss on failed X payments. Extended cases cover return, absence, phasing, shortage, and combined X/fixed shortage. `TestUncapturedXCounterPaymentAmountAndIsolation` covers announced versus ambient X, direct preparation, interleaved amounts sharing one declaration, and clone isolation. `TestUncapturedXCounterPaymentZero` covers present, absent, phased, and returned sources. The public synthetic declaration validates and exercises activation payment, stack copying, and isolated cloned resolution. Existing captured-X and declaration contracts remain in the focused run. These nontargeted costs need present identity for positive removal, not departure LKI; replacement ordering and simultaneous actions are unchanged.

The complete Chamber Sentry text in `data/JMP.json` was compared with the Jumpstart inventory and gap record. Its X counter-removal cost supplies the acceptance requirement. A fresh caller search found one production uncaptured-X call in `cards/jumpstart/creatures.go`; it is preserved. This correction does not migrate the card or its legacy entry/graveyard ability wiring. The inventory's missing entry-adapter claim is stale: `EntersWithCounters` and `ColorsSpentToCastEnteringPermanent` already exist, as the prior tracker records. No production card, inventory, or gap record changed. The prior direct fixed-cost slice's uncaptured-X exclusion is superseded only for direct preparation.

Excluded: optional/alternative wrapper locking, general payment-order choice, mutations during commit, new payment outputs, and card wiring. Next dependency-ready work is a separate P wrapper-payment audit or T divided-announcement prerequisite. No second slice was started.

Validation: focused contracts with `-count=1`, `go test ./pkg/mage/... -short -count=1`, registration/closed-role checks with `-count=1`, `go test ./... -short -count=1`, build, vet, full `make test`, and `make lint` with zero issues pass. Lint simplified one equivalent boolean condition; the post-lint diff was reviewed, and engine tests, boundary checks, build, vet, and full `make test` passed again. `git diff --check` passes. No validation blocker remains. Unrelated untracked work is preserved.

## Current Slice: Direct Source-Counter Payment Commit Safety

Delivery step 3's P payment prerequisite now locks direct uncaptured fixed counter-removal payments and checks direct fixed/captured/chosen counter removals before commit spends resources. A fresh API, tracker, caller, and test audit confirmed that the preferred damage identity, singular damage, source attack/block, and filtered death capabilities already exist. No adapter or public signature was recreated. `RemoveCountersCost(CounterType, int) Cost`, `RemoveCountersCostInto(CounterType, int, *NumberBinding) Cost`, and `RemoveAnyCountersCost(CounterType, *NumberBinding) Cost` retain their exports and supported contexts.

`TestCounterPaymentCommitPreflight` first reproduced acceptance of a returned source by the uncaptured fixed cost and mana/life loss on failed chosen-count commits. A combined fixed/chosen payment also reproduced partial payment after a shortage where each individual removal remained affordable. Preparation now copies and locks direct fixed costs regardless of output capture. Commit uses existing source checks and totals direct fixed/chosen removal amounts by counter type before mana abilities, mana payment, life payment, or counter removal. Failure publishes no paid-number result and repeats no choice. The reusable boundary is the existing total-payment transaction, counter executor, incarnation lookup, and captured output frame; no new role, mutable declaration, AI metadata, target, or replacement executor was added.

Acceptance evidence: `TestCounterPaymentCommitPreflight` covers fixed, captured, chosen, and combined payments with source return, absence, phasing, and shortage; `TestCounterPaymentLockIsolationAndZero` covers zero, independent pending transactions sharing one declaration, and game clone isolation; `TestCounterPaymentCombinedCommit` covers success, separate counter types, a single choice, and the paid result. `TestCounterPaymentLockPublicDeclaration` compiles the reference declaration, validates it, pays at activation, and checks isolated cloned-stack resolution. Existing fixed/chosen/X capture and declaration tests remain in the focused run. These nontargeted payments use present identity rather than LKI; they add no simultaneous action or replacement contract.

The complete Osai Vultures and Black Mana Battery texts in `data/LEG.json` were checked against the Legends inventory and JSON gap record. The former requires a two-carrion-counter cost; the latter requires a chosen charge-counter cost. This engine correction completes no card migration. A fresh caller search found 18 uncaptured fixed-cost production calls in nine files and no chosen-count production caller. All production callers and legacy exports remain. No card, inventory, or gap record changed or closed. The prior counter-declaration tracker entry remains valid.

Excluded: uncaptured X incarnation locking, optional/alternative wrapper locking, general payment-order selection, mutations during commit, new captures/adapters, and production wiring. The next dependency-ready work is a separate P uncaptured-X/wrapper payment contract audit or T divided-announcement prerequisite. No second slice was started.

Validation: focused contracts and the public synthetic declaration pass with `-count=1`; `go test ./pkg/mage/... -short -count=1`, the registration/closed-role run with `-count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, full `make test`, `make lint` with zero failures, and `git diff --check` pass. The post-lint diff was reviewed; lint made no further changes. Historical package/C-metadata blockers did not recur. No validation blocker remains. Unrelated tracked and untracked work is preserved.

## Current Slice: Counter-Payment Declaration Safety

Delivery step 3's P counter-payment prerequisite now rejects invalid declarations before gameplay. A fresh API, caller, and tracker audit confirmed that step 2 damage identity, singular damage, source attack/block, and filtered other-creature death already exist. Fixed, announced-X, and chosen-count counter-removal constructors also exist; this slice repairs their declaration safety rather than recreating them.

The public signatures remain `RemoveCountersCost(CounterType, int) Cost`, `RemoveCountersCostInto(CounterType, int, *NumberBinding) Cost`, `RemoveXCountersFromSourceCost(CounterType) Cost`, `RemoveXCountersFromSourceCostInto(CounterType, *NumberBinding) Cost`, and `RemoveAnyCountersCost(CounterType, *NumberBinding) Cost`, including their `dsl` exports. Registration rejects types outside `[0, core.NumCounters)` and negative fixed counts. Uncaptured fixed and X costs now enter the existing inspection graph. Nested optional, alternative, spell additional, and resolution-payment declarations receive the same checks. Existing capture scope and context restrictions remain. Direct checks fail before array access; chosen-count preparation fails before requesting a choice. No executor, frame, role, AI metadata, or declaration state was added.

`TestCounterPaymentDeclarationBounds` first failed because all five constructors accepted invalid types and uncaptured fixed costs accepted negative counts. `TestCounterPaymentInvalidDirectCalls` first reproduced an array-index panic. The new synthetic contracts cover every supported enum value, both enum boundaries, negative counts, direct calls at zero and positive X, absent sources, nested inspection, and invalid total-payment rollback without mana, life, counter, or choice changes. The existing `TestPaidCounterChoice*`, `TestFixedCounterPayment*`, and `TestPaidXCounters*` contracts pass and cover incarnation return, phasing, insufficient counters, zero, interleaved instances, copied stack objects, clone isolation, and captured output. The public DSL reference example compiles and validates in `TestCounterPaymentBoundsPublicDeclaration`. This slice changes no targets, LKI, simultaneous actions, or replacement ordering; these nontargeted payments reuse the existing transaction.

Black Mana Battery's complete text in `data/LEG.json` was checked against the Legends inventory and JSON gap record. It requires charge-counter payment and additional mana per counter removed; this safety correction completes no card migration. A fresh search found 19 production calls to the uncaptured fixed/X constructors in nine files and no chosen-count production caller. All signatures remain. No production card, inventory, or gap record changed or closed. Excluded: new counter kinds, general payment-order selection, new adapters, counter-presence gate costs, and production wiring. Next dependency-ready work is a separate T per-target/divided-announcement prerequisite or P payment output after a fresh audit. No second slice was started.

Validation: focused new and existing counter-payment contracts with `-count=1`, engine short tests with `-count=1`, public registration and closed-role checks with `-count=1`, repository short tests with `-count=1`, build, vet, and full `make test` pass. `make lint` passes with zero issues and `git diff --check` passes. The post-lint diff was reviewed; lint made no further changes. No validation blocker remains. Unrelated untracked work is preserved.

## Current Slice: Filtered Singular Discard Payment Output

Delivery step 3's P payment output adds `DiscardMatchingCardCostInto(filter CardFilter, paid *CardBinding) Cost`, exported through `dsl`. Fresh API, test, and tracker searches confirmed that damage identity, singular damage, source attack/block, filtered other-creature death, locked filtered discard, and existing singular/set paid-card frames already exist. This constructor composes those services without a new executor, frame format, role, AI metadata, or mutable declaration state.

A direct activated cost or spell additional cost saves the chosen card's movement-time graveyard visit. Preparation offers matching candidates once. Commit rechecks the filter and hand visit before spending resources. Prevention and an unrevealed hidden destination reject characteristic-constrained payment. Graveyard departure and return cannot restore the capture. Stack copies retain it; cloned games isolate resolution. Registration guards now reject singular as well as set discard captures inside optional, alternative, and resolution-time payments. No production caller uses the singular capture constructor, so the guard correction preserves production declarations and all legacy signatures.

Tests first failed because the constructor was absent. Additional failing tests exposed the missing singular registration guards. `TestPaidFilteredDiscard*` covers filtered selection, absent/returned hand visits, filter rechecks, invalid choices, hidden destinations, prevention and replacement rechecks, mana/life rollback, reused declarations and interleaved transactions, activation/spell capture, graveyard visit identity, stack copies, clone isolation, nil/unnamed/duplicate/missing producers, zero filters, and unsupported contexts. The public DSL example compiles and is inspected and validated. Two compile-failure fixtures reject a permanent filter and a card-set binding. These nontargeted hand choices have no battlefield LKI, phasing, or target-legality contract; simultaneous set actions are unchanged.

Thirst for Knowledge's full text in `data/JMP.json` and its Jumpstart inventory/gap record were checked. Its filtered resolution-time payment is supplied by the existing noncapturing constructor; this direct payment output does not complete its wiring. A fresh search found no production `DiscardMatchingCardCostInto` or `DiscardCardCostInto` callers and one card-local `discardCardOfTypeCost` construction. No card, inventory, or gap record was changed or closed. Excluded: optional/alternative or resolution captures, characteristic snapshots, multiple filtered cards, random payments, and production wiring. Next dependency-ready work is a separately audited T per-target/divided announcement prerequisite or P payment output. No second slice was started.

Validation passes: focused contracts with `-count=1`; `go test ./pkg/mage/... -short -count=1`; public registration and closed-role checks with `-count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`; full `make test`; `make lint` with zero issues; and `git diff --check`. All required checks were repeated after the final acceptance tests and compile-failure fixtures. The post-lint diff was reviewed; lint added spacing in the compile-failure fixtures only. No validation blocker remains. Unrelated untracked work is preserved.


## Current Slice: Locked Filtered Singular Discard Payment

Delivery step 3's P payment extension now supplies `DiscardMatchingCardCost(filter CardFilter) Cost`, exported through `dsl`. Fresh API, test, and tracker searches confirmed that damage identity, singular damage, source attack/block, filtered other-creature death, and distinct-name paid-card output already exist. This slice extends `discardCost` with a filter and reuses `Player.ChooseCardFromHand`, the locked hand-visit transaction, isolated total-cost validation, commit preflight, and the shared discard replacement/movement/event service. No second executor, mutable binding declaration, result frame, target role, or AI metadata was added.

The payer chooses once from matching hand cards. Preparation and commit reject missing, returned, nonmatching, outside-hand, and nil choices without substitution; stale choices fail before mana or life payment. Prevention and an unrevealed library destination reject this characteristic-constrained payment under CR 701.9c. Zero `CardFilter` matches all hand cards. Supported contexts are direct activated costs, spell additional costs, announcement-time optional/alternative wrappers, and resolution-time payment groups. Paid-card outputs, discarded-characteristic snapshots, multiple filtered discards, random discard, revealed-library destinations, general cost-order selection, and production card migration are excluded. Hand choices have no battlefield LKI, phasing, or target-legality contract.

Tests first failed solely because the constructor was absent. `TestFilteredDiscardPayment*` covers filtered candidates, stale hand visits, filter rechecks, invalid and absent choices, prevention, hidden destinations, replacement rechecks, mana/life rollback, independent pending payments using one declaration, activation and spell costs, optional/alternative announcement wrappers, stack copies, clone isolation, zero filters, and the draw-then-pay/decline/unavailable branches. `TestFilteredDiscardPublicDeclaration` compiles and validates a synthetic complete spell sequence and inspects the cost. A compile-failure fixture rejects a permanent filter. Existing distinct-name, locked-discard, and paid-discard contracts preserve shared simultaneous-action and output guarantees.

Thirst for Knowledge's complete text in `data/JMP.json` was checked against the Jumpstart inventory and gap record: “Draw three cards. Then discard two cards unless you discard an artifact card.” This constructor supplies its filtered payment requirement. A fresh caller search found no production caller of the new constructor and one production `discardCardOfTypeCost` construction in `cards/jumpstart/spells.go`; that local cost and its legacy spell sequence still need separate migration and card validation. No production card, inventory, or JSON record changed or closed. Cauldron of Essence's typed reanimation and Elvish Farmer/Thallid Devourer's Saproling costs were rechecked and remain as already recorded.

The next dependency-ready work is a separate P filtered paid-card output or T per-target/divided announcement prerequisite after a fresh requirements audit. No second slice was started.

Validation passes: focused contracts and public declaration/registration/closed-role checks with `-count=1`; `go test ./pkg/mage/... -short -count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`; full `make test`; `make lint` with zero issues; and `git diff --check`. The post-lint diff was reviewed. Lint replaced the filtered eligibility loop with `slices.ContainsFunc` and formatted test spacing. Focused contracts, engine checks, DSL registration/closed-role checks, build, vet, and full `make test` passed after those changes. No validation blocker remains. Unrelated untracked files remain untouched.

## Current Slice: Distinct-Name Discard Payment Output

Delivery step 3's P payment output extends the completed locked, simultaneous distinct-name discard prerequisite. A fresh API and test search confirmed that damage-source identity, explicit singular damage, source attack/block events, and filtered other-creature death already exist. This slice adds only `DiscardCardsWithDifferentNamesCostWithCards(n int, paid *CardsBinding) Cost`, exported through `dsl`. The name follows the existing distinct-name cost and paid-set constructor names rather than introducing a general payment-capture role.

The constructor reuses `discardCost`, locked hand-visit preparation and commit preflight, simultaneous discard replacements/movement/events, and instance-local paid-card-set frames. Direct activation costs and spell additional costs save the batch's resulting graveyard visits. `FromCards` and `CountCards` use those visits; departure and return cannot restore an old reference. Zero is valid. Failed payment publishes no capture. The existing distinct-name characteristic constraint rejects an unrevealed hidden-zone replacement under CR 701.9c. No engine executor, mutable declaration state, target, AI metadata, or frame format was added.

Tests first failed to compile because the constructor was absent. `TestPaidDistinctDiscardCards*` now covers activation and spell payment, independent pending transactions, declaration reuse, stack copies with a different resolving controller, cloned-game isolation, graveyard departure/return, same-name refusal, stale hand visits before life payment, prevention, hidden-zone replacement rejection, zero, inspection, and a compiling public declaration. Registration rejects nil/unnamed bindings, negative counts, duplicate producers, missing producers, optional/alternative wrappers, and resolution-time captures. Two compile-failure fixtures reject singular-card and permanent bindings. The existing `TestDistinctDiscardPayment*` and `TestPaidDiscardCards*` contracts also pass and cover simultaneous event state, replacement rechecks, and invalid selections. Battlefield LKI, phasing, and target legality do not apply to these nontargeted hand choices.

The full Ormos, Archive Keeper text in `data/JMP.json` and its Jumpstart inventory/gap requirements were rechecked. Its distinct-name payment requirement informs the constraint, but paid-card output does not complete its empty-library draw replacement, flying, or production migration. A fresh caller search found one production `DiscardCardsWithDifferentNamesCost` caller in `cards/jumpstart/creatures.go` and no production caller of the new constructor. All old signatures remain. No card, inventory, or JSON gap record changed or closed. The existing tracker already records Cauldron of Essence's typed reanimation and the Saproling-sacrifice abilities for Elvish Farmer and Thallid Devourer; the source search confirmed those entries remain current.

Excluded: filtered/random discard choices, discarded-characteristic snapshots, optional/alternative captures, resolution-time captures, general cost-order selection, and production card wiring. The next dependency-ready slice is a separately audited P filtered discard-payment contract for Thirst for Knowledge, or a T per-target/divided-announcement prerequisite. No second slice was started.

Validation passes: focused new and existing discard contracts with `-count=1`; `go test ./pkg/mage/... -short -count=1`; public registration, closed-role, and declaration checks with `-count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`; full `make test`; `make lint` with zero issues; and `git diff --check`. The post-lint diff was reviewed. Lint added spacing to the two compile-failure fixtures only; focused contracts and registration/closed-role checks passed again. Unrelated untracked files remain untouched.

## Current Slice: Locked Simultaneous Distinct-Name Discard Payment

Delivery step 3's P payment extension requires an exact distinct-name discard payment before adding captures. A fresh API, test, and caller audit confirmed that step 2's damage identity, singular damage, source attack/block, and filtered death primitives already exist. `DiscardCardsWithDifferentNamesCost` existed, but its separate executor selected again at commit, substituted invalid selections, moved cards sequentially, and treated cost discards as effect-caused. Failing synthetic contracts reproduced stale hand-visit acceptance, invalid-choice acceptance, partial event state, ignored cost prevention, hidden-zone characteristic payment, and missing negative-count validation.

The existing `DiscardCardsWithDifferentNamesCost(n int) Cost` export now declares a distinct-name constraint on the shared discard cost. It reuses hand-visit locks, isolated total-cost validation, commit preflight, simultaneous discard movement, replacement rechecks, and post-movement events. No second payment or discard executor was added. Invalid choices fail without substitutes. A characteristic-constrained cost rejects the current unrevealed library replacement before movement under CR 701.9c; unfiltered discard redirection remains compatible. Negative counts fail registration and zero is valid. Announcement-time optional and alternative costs retain the same locked choices. Direct activation, spell additional costs, and resolution-time payment groups use the existing payment service. No new author role, binding frame, target, or AI metadata was added.

Acceptance covers one-time selection, absent and returned hand visits, invalid/duplicate/same-name/nil/outside/short/oversized choices, zero, registration, simultaneous event state, cost-only prevention, hidden destination rejection, replacement rechecks, independent pending transactions, declaration reuse, game clones, stack copies, spell costs, and optional/alternative announcement wrappers. `TestDistinctDiscardPublicDeclaration` compiles and validates the reference example. Phasing, battlefield LKI, and target legality do not apply to this nontargeted hand payment. Random discard, new result captures, revealed-library discard actions, general cost-order choice, and production migration are excluded.

The full Ormos, Archive Keeper text in `data/JMP.json` was checked against the Jumpstart inventory and gap record. Its three-card distinct-name payment is this contract's requirement; flying, the empty-library draw replacement, and production declaration migration remain separate. A fresh search found one production `DiscardCardsWithDifferentNamesCost` caller, in `cards/jumpstart/creatures.go`; the signature and export remain. No production card, inventory, or JSON record changed or closed. Next dependency-ready work is a separate P filtered discard-payment or paid distinct-name card output after a fresh audit; T per-target payment and divided-announcement prerequisites remain separate candidates. No second slice was started.

Validation passes: focused distinct-name, existing locked discard, paid-card, batch, and replacement contracts with `-count=1`; the public DSL declaration; `go test ./pkg/mage/... -short -count=1`; registration and closed-role tests with `-count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`; full `make test`; `make lint` with zero issues; and `git diff --check`. The post-lint diff was reviewed; lint made no further changes. Final engine, public registration/role, and vet checks passed after the test additions. Historical missing-package and C-metadata failures did not recur. All unrelated untracked files remain untouched.

## Current Slice: Multi-Card Discard Payment Output

Delivery step 3's P payment output now follows the completed locked hand-visit and simultaneous discard-payment prerequisites. A fresh API and caller search confirmed that preferred step 2 damage, source attack/block, and filtered other-creature death primitives already exist; this run does not recreate them.

`DiscardCostWithCards(n int, paid *CardsBinding) Cost` saves the resulting graveyard visits for a direct activated cost or spell additional cost. `FromCards(paid)` supplies the existing sealed `CardSet`, and `CountCards` counts only visits still present at resolution. Zero cards is valid. Redirected discards pay an unfiltered cost but supply no graveyard reference. The adapter reuses locked hand choices, isolated total-cost validation, the simultaneous discard/replacement service, instance-local payment outputs, and existing stack-copy/game-clone frames. The shared batch now returns its recorded events internally so payment results retain movement-time visits rather than reading a later zone visit. No new action or replacement executor was added.

The contract tests first failed because the constructor was absent. Synthetic tests cover activation and spell payment, two-card and zero-card groups, mixed and hidden-zone replacements, prevented payment, stale hand visits before life payment, graveyard departure and return, interleaved transactions, reused declarations, stack copies, and cloned-game isolation. Inspection and public DSL example tests pass. Registration rejects nil/unnamed bindings, negative counts, duplicate producers, missing producers, optional/alternative wrappers, and resolution-time payment captures. Compile-failure fixtures reject singular card and permanent bindings. Optional activation costs now enter inspection even without an optional-result binding, so nested unsupported captures cannot escape registration validation. Existing batch tests cover simultaneous event state and replacement rechecks. Battlefield LKI, phasing, and target legality do not apply to these nontargeted hand choices; no target, mana planner, or AI metadata contract changed.

The complete Zombie Infestation and Thirst for Knowledge texts in `data/JMP.json` were checked against the Jumpstart inventory and JSON gap record. Zombie Infestation requires two chosen discards but still needs separate production wiring. Thirst for Knowledge still needs the filtered, resolution-time artifact-card alternative; this adapter does not meet that requirement. No card, inventory, or gap record changed or closed. A fresh search found nine production `DiscardCost` occurrences and no production `DiscardCardCostInto` or `DiscardCostWithCards` callers. All existing exports remain compatible. Filtered/random/different-name choices, optional or alternative payment captures, discarded-characteristic snapshots, general cost-order choice, and production migration are excluded. Next dependency-ready work is a separately audited P filtered discard-payment contract or a T per-target/divided-announcement prerequisite. No second slice was started.

Validation passes: focused paid-discard, existing singular paid-card, discard-batch, and replacement contracts with `-count=1`; the public DSL example; `go test ./pkg/mage/... -short -count=1`; `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles|TestPaidDiscardCardsPublicDeclaration' -count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`; full `make test`; `make lint` with zero issues; and `git diff --check`. Lint only converted a test loop to Go range syntax. The post-lint diff was reviewed and focused tests were rerun. Historical missing-package and C-metadata failures did not recur. Unrelated tracked and untracked work was preserved.

## Current Slice: Simultaneous Discard-Cost Payment

Delivery step 3's P multi-card discard output requires a complete discard-payment contract first. The preferred step 2 damage, source combat, and filtered death primitives already exist. A fresh regression showed that `DiscardCost(2)` sent its first discard event while the second chosen card remained in hand. A second regression showed a replacement reading that intermediate hand. This slice corrects that engine prerequisite; it adds no payment output adapter.

`DiscardCost(n int) Cost` now prepares every chosen cost-discard replacement before movement and uses the same batch movement and event executor as effect hand discard. All accepted cards move before the first event. If a replacement prevents one chosen discard, the entire discard group fails before movement. The existing hand-visit lock, isolated total-cost validation, singular paid-card capture, registration inspection, and stack frames remain in use. The direct payment path also rejects duplicate, missing, or invalid choices before movement. Activation, spell additional-cost, and existing resolution-payment callers keep their signatures. No production cards or legacy exports changed. This slice does not add random or different-name discard batching, filtered choices, card-set outputs, or general payment-order selection.

Acceptance tests first failed for partial event state and intermediate replacement state. Synthetic activation tests now prove complete graveyard and redirected-library state at each event, correct event visit identity, history, and resolution. Additional contracts cover replacement preparation and rechecks, prevention of one discard without partial movement, stale locked visits before life payment, and clone isolation. Existing discard payment tests cover zero, duplicate/short/outside/nil choices, independent transactions, declaration reuse, stack copying, and singular result rollback. Phasing, battlefield LKI, and target legality do not apply to this nontargeted hand-payment group. A public DSL declaration validates the compiling reference example.

The full Zombie Infestation and Thirst for Knowledge texts in `data/JMP.json` were checked against the Jumpstart inventory and gap record. The former requires a two-card payment; the latter still requires filtered resolution-time choice. This contract correction completes neither card migration. The inventories and JSON records remain unchanged and open. A fresh search found nine production `DiscardCost` occurrences; they remain compatible. Next dependency-ready work is P multi-card discard payment output or a separately audited filtered discard-payment contract. No second slice was started.

Validation passes: focused batch, payment-lock, paid-card, and effect-batch tests with `-count=1`; the public declaration test with `-count=1`; `go test ./pkg/mage/... -short -count=1`; `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`; `go test ./... -short -count=1`; `go build ./...`; `go vet ./...`; full `make test`; `make lint` with zero issues; and `git diff --check`. Lint formatted the test files without further behavior changes. The post-lint diff was reviewed. Historical missing-package and C-metadata failures did not recur. Unrelated tracked and untracked work was preserved.

## Current Slice: Locked Discard Hand-Visit Identity

Delivery step 3's P payment extension requires exact locked object identity before broader discard outputs or filters. A fresh audit found that the existing prepared `DiscardCost` locked physical card IDs only and accepted a card that left the hand and returned before commit. The regression failed for singular, two-card, and paid-card discard costs. The transaction now captures the existing hand-visit identity for every chosen card, and its shared preflight rejects stale visits before mana, life, or other costs are spent. This is an engine prerequisite correction, not a new typed adapter.

Public APIs remain `DiscardCost(n int) Cost` and `DiscardCardCostInto(card *CardBinding) Cost`. Activation, spell additional-cost, and existing resolution-payment callers retain their signatures. The paid-card constructor still publishes only the resulting graveyard visit after successful payment. No sealed role, declaration scope, AI metadata, replacement order, or stack-frame format changes. The implementation reuses hand visits, clone handling, isolated total-cost validation, shared commit checks, and the discard replacement action. Phasing and target legality do not apply to a nontargeted hand choice. Simultaneous movement, filtered choices, and new payment outputs are excluded.

Synthetic contracts cover returned visits, all-or-nothing multi-card preflight, mana/life/event/result preservation on failure, zero and unchanged visits, duplicate/short/outside/nil choices, two independent prepared transactions, declaration reuse, clone isolation, and a two-card activation with a stack copy. Existing paid-card tests cover spell and activation capture, stack copying, graveyard departure, replacement to a hidden zone, and prevented-payment rollback. The complete Zombie Infestation and Thirst for Knowledge text was checked in `data/JMP.json`: the first needs an exact two-card cost; the second still needs a filtered resolution-time choice. The Jumpstart inventory and gap record remain unchanged and open. No production card changed. A fresh caller search found nine production `DiscardCost` occurrences and no `DiscardCardCostInto` callers; both exports remain. All production card migration and full card validation remain separate.

Validation passes: focused discard payment and paid-card contracts with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration, closed-role and public paid-discard declaration tests with `-count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed; lint made no further changes. Historical missing-package and C-metadata validation failures did not recur. Unrelated untracked files remain untouched. Next dependency-ready work is a separate P multi-card discard output or filtered discard-payment contract after a fresh audit; T per-target payment and divided-announcement prerequisites remain separate candidates.

## Current Slice: Singular Counter-Removal Result

`RemoveCountersFromWithResult(PermanentRef, CounterType, ResolutionValue, *NumberBinding) ResolvingEffect` saves the actual number removed by a singular resolving instruction. It reuses the existing typed subject resolution, shared counter-removal executor, inspection graph, definite-assignment validator, and per-instance number frame. The required result parameter avoids an unnecessary option family for one output. Spell, activation, and triggered bodies are supported; costs, quantity choices, simultaneous collections, counter movement, and production wiring are excluded. `RemoveCountersFrom` and all legacy exports remain compatible.

The focused contract first failed because the constructor was absent. Tests cover partial and nonpositive amounts, counter-kind isolation, stale result clearing, source absence and return, phasing and phase return, target legality with another legal target, stack copies, game clones, activated and triggered resolution, invalid counter types, nil declarations, unnamed bindings, producer dominance, and a compiling public declaration. Compile-failure fixtures reject player and set subjects. No transaction or replacement service changed; this resolving action cannot publish a payment capture.

The Antiquities inventory and complete Tetravus text in `data/ATQ.json` were checked. Actual counter-removal quantity is one requirement for its first upkeep body, but optional quantity selection, linked tokens, and all production wiring remain open. The available adapter does not complete Tetravus or close its gap. No inventory or JSON record changed. A fresh caller search found no production `RemoveCountersFromWithResult` caller. Existing production counter helpers remain. The next dependency-ready work in delivery step 3 is a separate P payment output or T per-target payment/divided-announcement prerequisite after a fresh audit.

Validation passes: focused engine and public DSL tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. Lint only formatted the new tests. The post-lint diff was reviewed; production cards and unrelated untracked work are unchanged.

## Current Slice: Announced Permanent Target Count

`AnnouncedTargetCount(*PermanentTargetsBinding) ResolutionValue` reads one group's chosen target count from the resolving stack object's recorded target choices. It works in spell, activation, and targeted trigger bodies. The count includes targets that later become illegal, phase out, or leave and return. Stack copies keep the announced count when retargeted. The value reuses the existing grouped target frame and registration scope validator; no new target or payment service is needed. Go requires a permanent target group, and registration rejects a nil or undeclared group. An object with all targets illegal still fails to resolve under CR 608.2b.

Focused tests first failed because the constructor was absent, then exposed that resolution filters illegal targets from effect bindings. The implementation reads the original recorded stack choices instead. Synthetic tests cover a phased target, a returned target, a stack copy in a cloned game, distinct target groups, an absent group, registration rejection, and a compiling public DSL declaration. The plan's T-family count proposal is now available as a resolution value; per-target payment, divided values, random targets, and production card wiring remain separate. No production card or legacy export changed. The existing inventories and JSON gap records remain open because this count alone completes none of their card requirements. The next dependency-ready T slice is an announcement-time per-target cost or divided-value declaration, after an audit of its payment or distribution prerequisite.

Validation passes: focused engine and DSL tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Source-Only Combat Prevention

`PreventCombatDamageBy(PermanentRef)` and `PreventCombatDamageByEach(PermanentSet)` install prevention for combat damage from the selected battlefield incarnations during the current combat. They use the shared damage replacement pipeline for player and permanent recipients. The singular instruction handles a blocker; the set instruction handles `BlockedAttackersOf(blocker)`. A returned card has a new source identity, phasing retains identity, and the replacement expires after the end-of-combat step. The source's later controller or departure does not change a captured damage action. These instructions do not prevent noncombat damage or combat damage to the selected sources from others. They are available in resolving spell, activation, and trigger bodies. The existing `PreventCombatDamageToAndBy` and all production callers remain unchanged.

Sewers of Estark's full Oracle text was checked in `data/PHPR.json`. Its typed combat branches and blocked-attacker set were already available; the new prevention instructions supply its remaining engine dependency. The test-only declaration compiles and validates. Focused tests first failed because the constructors were absent, then covered source and recipient scope, noncombat damage, combat expiry, incarnation return, phasing, control change, departed-source LKI, target return, clone isolation, and registration rejection. The `promo-combat-prevention` gap remains open for production wiring and full card validation; `cards/promo/spells.go` still uses `FuncEffect`. A fresh search found no production callers of either new export. Focused tests, short engine and repository suites, registration and closed-role checks, build, vet, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed. Next dependency-ready work is a P result output or T target-announcement contract after a fresh audit.

## Current Slice: Directional Blocked-Attacker Set

`BlockedAttackersOf(PermanentRef) PermanentSet` selects the attacking creatures that one referenced creature is currently blocking. It is available in resolving spell, activation, and trigger bodies. It reads the combat service's current incarnation links when consumed, so absent, phased, removed, or returned objects do not contribute. Multiple attackers blocked by one creature appear once each. The existing `BlockersOf` and `BlockingOrBlockedBy` sets remain distinct; no production card or legacy export changed.

The test-only contract first failed because the constructor was absent. Synthetic tests cover multiple and unrelated combat links, removal, phasing, leave and return, clone isolation, inspection, registration scope, and a public compiling declaration. The complete Sewers of Estark text in `data/PHPR.json` and the Promo gap record were checked. Its combat-state branches and blocked-attacker set are now available, but source-only combat prevention for this combat and production wiring remain open. The `promo-combat-prevention` gap therefore stays open. A fresh caller search found no production `BlockedAttackersOf` caller; Sewers still uses `FuncEffect`. The next dependency-ready work is source-only combat prevention with an exact combat duration.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Resolution-Time Combat Membership Branches

`IfPermanentAttacking(PermanentRef, ResolvingEffect)` and `IfPermanentBlocking(PermanentRef, ResolvingEffect)` check current, incarnation-keyed combat membership when a resolving spell, activation, or trigger reaches the branch. They accept a declared permanent target or a dominating saved permanent reference, as well as `This` and `AttachedPermanent`. The body runs only for a present, unphased permanent still in the relevant combat role. The branches reuse the combat service and existing subject resolution, inspection, and registration scope checks. Static `PermanentIsAttacking` and `PermanentIsBlocking` retain their narrower state-condition contract. No production card or legacy export changed.

Focused tests first failed because the constructors were absent. Synthetic tests cover attacking and blocking, target scope and legality, phasing, return, clone isolation, registration failures, and a compiling public declaration. Sewers of Estark's complete text in `data/PHPR.json`, the Promo inventory, and `doc/designs/promo-combat-prevention.json` requires independent attacking and blocking checks. These branches supply only that first requirement. The blocked-attacker collection and combat-only source prevention remain open; its production wiring and gap status are unchanged. A fresh caller search found no production use of either new constructor; Sewers still uses its `FuncEffect` combat checks. Focused tests, short engine and repository suites, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed. The next dependency-ready work is the blocked-attacker collection or a separate P output after a fresh audit.

## Current Slice: Creature-Entry Controller Capture

`AnotherCreatureEnters().WithController(*PlayerBinding)` and the existing controlled-creature entry patterns now save the entering creature's event-time controller in the pending trigger frame. The optional method is available in battlefield `WhenEvent` and `WhenEventModes` declarations. It uses the shared zone-change event, entry characteristic check, and player-binding service. The saved player survives a later control change, departure and return, source departure, stack copying, and game cloning. Source phasing prevents detection. Registration rejects missing, unnamed, repeated, or undominated bindings. No production card or legacy export changed.

The focused contract first failed because `WithController` was absent. Synthetic tests cover a different owner and controller, source departure, entrant return, phasing, stack copying, cloning, and invalid declarations. The public DSL declaration compiles and validates. The complete Lathliss, Dragon Queen text in `data/JMP.json` and the Jumpstart inventory were checked: her existing subtype, nontoken, and controller filter requirements are separate from this capture, and production wiring remains open. No JSON gap record closes. A fresh card search found no production `.WithController` creature-entry pattern caller. The next dependency-ready work is the T recorded target-group boundary after a fresh audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: General Permanent Tap Event

`PermanentBecomesTapped().OfType(type).ByOpponent().WithPermanent(binding).WithController(binding)` is a typed E-family pattern for battlefield `WhenEvent` and `WhenEventModes` declarations. Its filters and captures are optional and immutable. It reuses the completed `EvtTapped` transition, event-time type and controller checks, and the pending-trigger frame. A batch creates one trigger per matching tap after every member has tapped. A saved permanent keeps its battlefield incarnation and becomes absent after departure and return or during phasing. The saved controller survives later changes, source departure, stack copying, and game cloning. Registration rejects invalid or repeated filters and missing, repeated, unnamed, or undominated captures. The public DSL export is `PermanentBecomesTapped`; `BecomesTapped(This)`, `ArtifactBecomesTapped`, and `OpponentLandBecomesTapped` remain available.

Synthetic contract tests first failed because the constructor was absent. They cover transition versus repeated tap, type and opponent filtering, simultaneous taps, phasing, leave and return, source departure, clone and stack-copy isolation, registration, and a compiling public declaration. The Antiquities inventory and complete Artifact Possession, Haunting Wind, and Powerleech text in `data/ATQ.json` were checked. Existing narrower patterns already express parts of those cards; their other clauses and production wiring remain open. No production card or JSON gap record changed. A fresh production caller search found no `PermanentBecomesTapped` caller. The next dependency-ready work is another E event participant/filter or a P action output after a fresh audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Singular Untap Action Result

`UntapPermanentWithResult(PermanentRef, UntappedThisWayInto(*NumberBinding))` saves 1 only when the referenced battlefield incarnation becomes untapped. It reuses `Game.UntapPermanent`, including the stun-counter replacement and untap event. A redundant, absent, phased-out, returned, or stun-replaced attempt saves 0. The named result lives in the resolving spell, activation, or trigger frame. Registration rejects missing, nil, unnamed, repeated, or undominated bindings; Go rejects a player or set subject. `UntapPermanent` and all production callers remain unchanged. This is a P action-result output, not an untap-cost result or a collection result.

Focused contracts first failed because the typed constructor was absent. Synthetic tests cover transitions, stun replacement, target legality after departure and return, phasing, stack-copy and game-clone isolation, declaration validation, and a public DSL example. The checked inventories do not identify a card whose remaining requirements are completed by this result alone. No JSON gap record or production card was changed. A fresh search found no production `UntapPermanentWithResult` caller. The next dependency-ready work is another P output or a T announcement contract after a fresh audit. The graveyard-return result is deferred because the existing move path lacks the general CR 400.6 zone-change replacement contract.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Singular Battlefield Exile Result

`ExileWithResult(PermanentRef, ExiledThisWayInto(*NumberBinding))` records 1 when one current battlefield incarnation moves through the existing `Game.ExilePermanent` operation and 0 when the reference is absent, phased out, or stale after leave and return. The sealed result option and named number binding work in resolving spell, activation, and trigger bodies. Registration rejects a missing, nil, duplicate, or undominated result. The result lives in the resolution frame; `Exile(PermanentRef)` and all legacy author exports remain unchanged. This is a P action output. It does not link an exiled card, offer a payment, or add a batch result.

The focused contract first failed because the typed constructors were absent. Synthetic tests cover completion, absent and phased objects, returned incarnation, target legality, stack-copy and clone isolation, declaration rejection, and a public DSL declaration. Tawnos's Coffin's complete Oracle text was checked in `data/ATQ.json`. Its inventory still requires linked cards, counter and Aura snapshots, and return timing; this output alone does not complete it. Its production wiring and gap record remain open. A fresh caller search found no production `ExileWithResult` caller; existing `Exile` and legacy exile callers remain. The next dependency-ready work is a separate P payment or action output, or the T divided-target announcement contract after an audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Paid Mill Count

`MillCostInto(int, *NumberBinding) Cost` saves the modifier-adjusted count of cards milled to pay one direct activated cost or spell additional cost. It reuses `MillCost` preparation and the locked total-cost transaction. The count enters the paid spell or activation frame only after commit, so stack copies and game clones retain independent values. Zero is a valid adjusted count. Registration rejects nil or unnamed bindings, duplicate producers, undominated reads, `EitherCost`, and resolution-time payment capture. `MillCost` and all production callers remain unchanged.

The contract test first failed because the constructor was absent, then because declaration validation did not propagate the paid binding. Synthetic tests cover modified and zero counts, two pending activations, spell copying, game cloning, payment rollback, and invalid declarations. A public DSL declaration compiles. This is a P payment-count output. It does not capture milled card identities or repair the general mill zone-change and replacement path. Deep Spawn's complete text in `data/FEM.json` requires an optional mill-two payment, already available through `MillCost`; its production card and Fallen Empires gap record remain open. A fresh caller search found no production `MillCostInto` caller. The next dependency-ready work is another P output or the separate T divided-target announcement contract after a fresh audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Singular Life-Gain Result

`GainLifeForWithResult(PlayerRef, ResolutionValue, ...LifeGainResultOption)` accepts `LifeGainedInto(*NumberBinding)` and saves one player's actual gain after life-gain modifiers and replacement. It uses the existing gain and event service. An absent recipient, nonpositive amount, or fully prevented gain saves zero. The binding belongs to the current resolving spell, activation, or trigger frame; clones retain separate values. Registration requires one named result option and a dominating producer. `GainLifeFor`, `dsl.GainLife`, and production callers remain unchanged. This is a P action-result adapter; it does not capture life paid as a cost or implement another life-gain replacement contract.

The contract first failed to compile because the constructor was absent. Focused tests cover the final replacement amount, zero and negative amounts, absent recipients, clone isolation, and registration errors. A public DSL declaration compiles. Existing `GainLifeEach` tests cover the shared batch service; this slice reuses that service for one recipient. A fresh caller search found no production `GainLifeForWithResult` caller and one `GainLifeFor` call in a production card test. No inventory or JSON gap record claims that this result alone completes a blocked card, so none was closed. Production card wiring remains separate. The next dependency-ready work is another P payment or action output, or a T announcement contract after a fresh audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Resolution-Time Optional Payment Availability

`IfPaid` and `UnlessPaid` now check that each cost component can be paid before asking the player whether to pay. A player with no controlled permanent cannot choose the sacrifice branch. The accepted branch still uses the existing locked total-cost transaction. In a `ForEachSavedCard` body, each iteration checks the resulting game state anew, so sacrificing the only permanent on the first iteration makes the second iteration discard. The public signatures and sealed roles are unchanged. This is a payment-choice engine contract, not a new author adapter; no production card or legacy export changed.

Read the Runes has the complete text "Draw X cards. For each card drawn this way, discard a card unless you sacrifice a permanent" in `data/JMP.json`. Its per-card choice can be composed from `DrawCardsInto`, `ForEachSavedCard`, `UnlessPaid`, `SacrificeMatchingIncludingSourceCost(AnyPermanent, ...)`, and `Discard`. The synthetic example uses a fixed draw count; production wiring still needs its announced X and a full card review. Its Jumpstart inventory and JSON gap record remain open. A fresh caller search found the production Read the Runes definition in `cards/jumpstart/spells.go`; no production card was modified. The next dependency-ready work is a separate P result output or T divided-target announcement prerequisite after a fresh audit.

The focused contract first failed because an impossible sacrifice was offered. Tests cover no legal permanent, a permanent controlled by the opponent, a phased-out permanent, successful payment followed by an impossible second payment, declining each offer, zero drawn cards, and registration in spell, activation, and trigger contexts. A public DSL declaration compiles. The existing payment transaction tests cover cost atomicity; this slice changes the availability gate only.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Blocked Source Defending-Player Capture

`BecomesBlocked(This).WithDefendingPlayer(*PlayerBinding)` saves the defending player at the first blocked-state transition. The shared `EvtCreatureBecomesBlocked` now carries that player in `TargetID`; when the defender is a permanent, the event records its controller at that time. The pending trigger owns the player binding, so source departure, return, stack copying, and game cloning do not change it. The pattern is valid in battlefield `WhenEvent` and `WhenEventModes`; it does not bind a blocker or introduce a target. Registration rejects nil, unnamed, duplicate, and undominated bindings. The change reuses combat groups, the existing event dispatcher, and event-player capture. No production card or legacy export changed.

Slate Street Ruffian's complete text was checked against `data/JMP.json` and its Jumpstart gap record. This adapter supplies its defending-player participant, but card wiring and full card review remain open; the JSON gap record is not closed. A fresh search found no production `BecomesBlocked` caller and existing legacy combat-trigger callers remain. The prior tracker already reflects the typed Cauldron of Essence reanimation and the Saproling sacrifice abilities for Elvish Farmer and Thallid Devourer. The next dependency-ready work is a separate P payment/action output or the T divided-target announcement contract after a fresh audit.

The new contract first failed to compile because `BecomesBlocked` had no defending-player option. Synthetic tests cover declared blockers, a creature entering blocking, a planeswalker defender, source departure and return, game-clone isolation, and declaration rejection. The public `dsl` declaration compiles. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Saved-Card Result Iteration

`ForEachSavedCard(*CardsBinding, *CardBinding, ResolvingEffect) ResolvingEffect` runs a sequential body once per card captured by one result, in saved order. The member retains its original zone visit. The loop copies binding maps per iteration and restores the outer frame even when the body returns an error. Registration checks producer dominance, a named member, and loop-local scope. It works in spell, activation, and trigger bodies. The first contract test failed because the constructor was absent; focused engine and public DSL tests now cover draw order, card departure and return, clone and stack-copy isolation, scope, and frame restoration. No production card or legacy export changed.

Read the Runes in `data/JMP.json` says, "Draw X cards. For each card drawn this way, discard a card unless you sacrifice a permanent." The loop supplies only the per-card iteration. Its per-iteration optional sacrifice choice and conditional discard still need a separate contract, and its production card and Jumpstart gap record remain blocked. A fresh production search found no `ForEachSavedCard` caller. The next dependency-ready P slice is the per-card sacrifice-or-discard choice after an exact payment and action-outcome audit. Focused tests, the short engine and repository suites, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Multi-Card Draw Hand-Visit Result

`DrawCardsInto(*CardsBinding) DrawResultOption` saves the cards actually drawn by one typed `Draw` instruction. `FromCards` reads only their captured hand visits; a card that leaves and returns is absent. The declaration works in spell, activation, and triggered resolving bodies and shares the existing individual draw and replacement path. Registration rejects nil, unnamed, duplicate, mixed-kind, and undominated bindings. The set belongs to the current resolution frame and copies with that frame. No production card or legacy export changed.

The contract test first failed because `DrawCardsInto` was absent. Synthetic tests cover partial, empty, and replaced draws, draw order, hand departure and return, absent player targets, stack copies, game clones, and registration errors. A public DSL declaration compiles. Read the Runes in `data/JMP.json` needs the cards drawn this way and a separate per-card discard-or-sacrifice choice, so its card and gap record remain blocked. The next dependency-ready P slice is that per-card choice contract or another result output after a fresh audit. A fresh caller search found no production `DrawCardsInto` caller; production `DrawCards`/`DrawFor` callers and legacy exports remain. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Paid Graveyard Exile Card Set

`ExileFromGraveyardCostWithCards(int, *CardsBinding) Cost` saves the exact cards that a direct multi-card graveyard exile payment moved into exile. `FromCards` reads only their captured exile visits; a card that leaves and returns is absent. The binding is published only after payment commits and is stored in each spell or ability stack frame, including copies and game clones. The constructor works in activated `Costs` and spell additional costs. Registration rejects nil, unnamed, duplicate, and undominated bindings, negative counts, resolution-time payment, and the captured form inside `EitherCost`. The original cost remains available, including inside `EitherCost`. No production card or legacy export changed.

This slice reuses the locked graveyard choice, payment transaction, group movement, exile visit, and card-set services. Its contract test first failed because the constructor and card-set stack capture were absent. Synthetic engine tests cover activation and spell payment, selected-card identity, stack copying, clone isolation, leave and return, stale-choice rollback before life payment, and registration errors. A public DSL declaration compiles. Grim Lavamancer's complete text was checked in `data/JMP.json`; its activation needs two-card exile and typed damage, but its production wiring remains legacy. Soaring Stoneglider's complete text was checked in `data/SOS.json`; its alternative additional cost already uses the uncaptured constructor. Neither card or gap record is complete from this adapter. A fresh caller search found no production `ExileFromGraveyardCostWithCards` caller and three production `ExileFromGraveyardCost` calls. The next dependency-ready slice is a separate P paid-object/result output or the T divided-target announcement contract after a fresh requirements audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.


## Current Slice: Locked Graveyard Exile Cost Choice

`ExileFromGraveyardCost(int) Cost` now lets the payer select distinct cards during payment preparation and locks each graveyard visit. The existing transaction validates those choices on its isolated game and rechecks them before live payment. A returned card, an invalid choice, or insufficient cards fails without exiling cards or paying later life costs. An `EitherCost` branch is selected once and retains its chosen exile cards. The cost is usable in activated `Costs` groups and spell additional costs; it does not yet produce a typed paid-card set or resolve general zone-change replacement ordering. No production card or legacy export changed.

Grim Lavamancer's complete cost and Soaring Stoneglider's alternative additional cost were checked in `data/JMP.json` and `data/SOS.json`. Synthetic tests first failed because the cost did not ask for card choices, accepted a returned visit, and accepted an invalid choice. They now cover distinct selection, batch movement, stale visit rejection, rollback, clone isolation, the `EitherCost` branch, and registration rejection of a negative count. Production callers remain in `cards/jumpstart/creatures.go` and `cards/secretsofstrixhaven/creatures.go`; typed card wiring and full card audits are separate. No JSON gap record closes. The next dependency-ready P work is a paid-card set binding for multi-card graveyard exile, after checking its consumer requirements, or the T divided-target contract.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The standalone reference example compiles. The post-lint diff was reviewed.

## Current Slice: Paid Matching-Tap Permanent

`TapNOtherMatchingCostWithPermanent(PermanentFilter, string, *PermanentBinding) Cost` chooses and taps one other untapped matching permanent controlled by the payer. It uses the existing locked matching-tap payment transaction and publishes the selected battlefield incarnation only after commit. `From(paid)` works while that incarnation remains present; `DamageSourceOf(From(paid))` can use its departure LKI. It is valid in an activated `Costs` group or as a spell additional cost. Registration rejects nil, unnamed, duplicate, undominated, and resolution-time payment bindings. `TapNOtherMatchingCost` and all production callers remain unchanged.

Vodalian War Machine's complete text in `data/FEM.json` requires tracking Merfolk tapped this turn to pay for its abilities. This slice supplies the single paid-object identity only. Turn-long, source-local payment history and the death action remain open, so its production card and Fallen Empires inventory stay unchanged. No JSON gap record closes. A fresh caller search found no production `TapNOtherMatchingCostWithPermanent`, `TapNOtherMatchingCost`, or `TapMatchingCost` call; these exports remain for separate wiring. The contract test first failed for the missing constructor, then for the missing definite-assignment integration. Synthetic tests cover activated and spell payment, source LKI after departure and return, stack copy, game clone, phased or returned choices, life-cost rollback, and registration errors. The public DSL test compiles. The next dependency-ready work in delivery order is another P payment output or the T divided-target announcement contract after a fresh audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Defending-Player Control Attack Restriction

`CanAttackOnlyIfDefendingPlayerControls(PermanentFilter) StaticRule` installs the existing battlefield attack restriction through the sealed static-rule role. It checks the nonactive defending player's current matching permanents at continuous-effect evaluation in this two-player engine. Current control qualifies independently of ownership; phasing and departure remove a match. It affects attacker declaration and does not remove an existing attacker. `Static` accepts the rule; resolving effects do not. No new combat state or production card changed. The legacy `PreventFromAttackingIfDefendingPlayerControls` author export remains for eight production calls.

The contract first failed because the typed constructor was absent. Synthetic tests cover controller and owner separation, phasing, departure and return, clone isolation, inspection, registration, and a compiling public DSL declaration. Dandân's complete text in `data/ARN.json` and Pirate Ship and Sea Serpent in `data/4ED.json` require this restriction, but their production definitions and inventory rows remain open for card wiring and full card validation. No JSON gap record closes from this adapter alone. The next dependency-ready work is a separate P payment output after a fresh contract audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Pure Player-Control State Condition

`PlayerControls(PlayerRef, PermanentFilter) StateCondition` checks current effective control of matching battlefield permanents. The pure state role supports `You` and `ActivePlayer` in `ActivationIf`, `IfState`, `WhenState`, `WithInterveningIf`, and `While`. Registration rejects nil, targeted, and saved player references because this state context has no resolution frame. It reuses the battlefield filter and control services; no new control state or card-specific attack rule was added. A zero-value filter matches all permanents. Production cards and legacy exports remain unchanged.

The contract tests first failed because `PlayerControls` was absent. Focused tests cover owner versus controller, phasing, leave and return, clone isolation, resolution-time active player, static declaration, invalid references, and a compiling public DSL declaration. Relevant rules are CR 403.3–4, 604.1, and 611.3a. Dandân's complete Oracle text in `data/ARN.json` also requires a defending-player attack restriction; this pure condition does not add that restriction, so its production card and Arabian inventory remain open. A fresh caller search found no production `PlayerControls` caller and eight production `PreventFromAttackingIfDefendingPlayerControls` callers. The next dependency-ready slice is the typed defending-player attack restriction after an engine contract audit, or a separate P payment output.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed. No JSON gap record closes from this adapter alone.

## Current Slice: Directional Blocker Set

`BlockersOf(PermanentRef) PermanentSet` selects the current creature incarnations blocking a specific attacker. It works in resolving spell, activation, and trigger bodies through existing permanent-set consumers. It reads the combat service when consumed, requires the referenced incarnation to be attacking, and excludes removed, phased-out, departed, and returned blockers. A blocked attacker with no remaining blockers produces an empty set. The selector is directional; the existing `BlockingOrBlockedBy` remains for either combat role. Registration rejects nil and undeclared references, and the sealed role rejects a player in place of a permanent. No production card or legacy export changed.

Tidal Flats in `data/FEM.json` requires blockers controlled by the enchantment controller and a separate payment by each attacking creature's controller. This selector supplies only the attacker-to-blocker relationship, so its inventory and JSON gap record remain blocked and its production definition is unchanged. Glyph of Delusion requires this-turn block history and other effects, so it is also not completed by this current-combat set. A fresh search found no production `BlockersOf` or `ForEachBlockerOfTarget` caller; existing legacy and typed combat selectors remain. The next dependency-ready slice is a P payment output or the Tidal Flats controller-relative set filter after a fresh requirement audit.

The contract test first failed because `BlockersOf` was absent. Focused tests cover relation direction, unrelated combat groups, phasing, leave and return, attacker identity, clone isolation, declaration rejection, inspection, and public DSL compilation. The engine reuses the existing combat group and permanent-set services; it adds no combat state or result storage. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Continuous-Control State Condition

`ControlledSinceTurnStart(PermanentRef) StateCondition` accepts `This` or `AttachedPermanent` in battlefield activation, resolving, and static conditions. It checks that the ability controller currently controls the selected permanent and that the engine's existing control-gain turn predates the current turn. It rejects absent, phased-out, returned, and stale source incarnations. Registration rejects nil and unsupported references. The existing control service and legacy `WithControlledSinceTurnStart` author export remain unchanged. No production card changed.

Rocket Launcher's complete text was checked in `data/ATQ.json` and the Antiquities gap record. Its production ability still uses the legacy activation option; damage and delayed destruction wiring remain separate card work. The contract test first failed because the typed constructor was absent. Synthetic tests cover turn boundary, control change, source return, phasing, clone isolation, failed activation rollback, registration, inspection, and a public DSL declaration. The Antiquities gap record stays blocked because the card is unmigrated. A fresh caller search found one production `WithControlledSinceTurnStart` caller and no production typed caller. The next dependency-ready work is a separate S history condition or P payment output after a fresh requirements audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Player-Controlled Permanent Set

`PermanentsControlledBy(PlayerRef, PermanentFilter) PermanentSet` selects current, unphased battlefield permanents under the resolved player's effective control. It works in resolving spell, activation, and trigger bodies through existing permanent-set consumers. It uses the shared battlefield filter and typed player reference. Registration rejects a nil or undeclared player. It does not add a static condition, mutate control, or change production cards or legacy exports.

The contract first failed because the constructor was absent. Synthetic engine and public DSL tests cover owner/controller separation, a continuous control change, phasing, departure and return, a declared player target, copied stack frames, clone isolation, and registration errors. The S-family inventory proposes this selector, but no JSON gap record is closed by its availability alone. All card wiring remains. A fresh caller search found no production `PermanentsControlledBy` caller; the older `ControlledPermanents` callers remain. The next dependency-ready work is a separate S selector or P payment output after a fresh audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Any-Controller Other-Creature Entry

`AnotherCreatureEnters()` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It matches a different creature entering under either controller and reuses the shared entry event, source exclusion, trigger frame, and existing entry filters and captures. `.WithPower` saves effective power at entry. The pattern does not implement Hamletback Goliath's optional effect or change any production card. Its full text was checked in `data/JMP.json`; the Jumpstart inventory and gap record remain open. A fresh caller search found Hamletback Goliath still using its legacy entry trigger. Existing legacy exports remain.

The focused contract first failed because the constructor was absent. Synthetic tests cover both controllers, owner versus controller, source phasing, self and noncreature exclusion, event-time power, clone isolation, invalid capture registration, and public DSL compilation. The next dependency-ready work is a separate P payment output or E event participant after a fresh contract audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Combat-Damage Group Source Subtype Filter

`OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype(string)` is a sealed battlefield `WhenEvent` or `WhenEventModes` filter. It matches once per controller, recipient player, and combat damage step when at least one contributing creature lacks the named subtype and deals positive damage. The damage service now saves each contributing source's incarnation and characteristics when damage is dealt. Detection uses those snapshots, not a later battlefield lookup. The existing per-source amount map and unfiltered event remain available to legacy callers. The filter rejects an empty, noncreature, or repeated subtype at registration. It adds no damage executor, target, or card-specific policy.

The contract tests first failed because `WithoutSubtype` was absent. Synthetic tests cover mixed Human and non-Human sources, prevention of the only qualifying source, phasing, source departure and return, clone isolation, public DSL compilation, and registration rejection. The existing combat group tests cover real combat steps, recipient grouping, full prevention, stack copying, and source lifetime. Keeper of Fables in `data/JMP.json` requires this condition and a draw; its production declaration still uses the legacy trigger and remains unwired. Killian's Confidence in `data/SOS.json` still needs a graveyard-active event source, hybrid payment, and source-card return. The Jumpstart and Secrets of Strixhaven inventories and JSON gap records remain open. A fresh search found two production `WheneverOneOrMoreCreaturesYouControlDealCombatDamageToPlayerTrigger` callers and no typed pattern caller. The next dependency-ready work is a separate graveyard-active event source or P payment output after a fresh contract audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The first engine short run caught one extra clone allocation; lazy snapshot allocation resolved it, and the short run passed on recheck. The post-lint diff was reviewed.

## Current Slice: Battlefield Combat-Damage Batch Event

`OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(*PlayerBinding)` is a sealed battlefield `WhenEvent` or `WhenEventModes` pattern. It observes the shared post-damage `EvtCombatDamageDealt` group once per damaging controller, recipient player, and combat damage step. Its optional player binding captures the recipient in the pending trigger frame. The adapter does not add a damage executor, read a later source controller, or change legacy callers. It cannot be used by a graveyard-active ability and does not filter the damaging creatures by subtype.

The contract first failed because the constructor was absent. Synthetic tests cover a real two-creature combat step, partial and full prevention, opposing controllers, source phasing and departure/return, stack copying, clone isolation, registration errors, and the public DSL declaration. Killian's Confidence in `data/SOS.json` requires this event but also a graveyard-active source, hybrid payment, and source-card return; Keeper of Fables in `data/JMP.json` also needs a non-Human source filter. Neither card is wired, so their inventory and JSON gap records remain open. A fresh search found two production callers of `WheneverOneOrMoreCreaturesYouControlDealCombatDamageToPlayerTrigger`, in Secrets of Strixhaven and Jumpstart; that export remains. No production caller uses the typed pattern. The next dependency-ready slice is the event-time source-characteristic filter for this batch, subject to an LKI audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The earlier missing-`cards/custom` and C-metadata failures do not recur in this workspace. The post-lint diff was reviewed.

## Current Slice: Sacrifice Event Permanent Capture

`CreatureYouSacrifice().WithPermanent(*PermanentBinding)` saves the exact creature incarnation from a completed sacrifice event in each pending battlefield `WhenEvent` or `WhenEventModes` trigger. `From(binding)` is absent after departure; `DamageSourceOf(From(binding))` can use departure LKI after the card returns. The event carries its predeparture incarnation, and the pattern reuses the existing sacrifice event, trigger frame, and permanent capture effect. Registration rejects nil, unnamed, repeated, and undominated captures. Existing `CreatureYouSacrifice()` declarations remain valid. No production card or legacy export changed.

The contract tests first failed because `WithPermanent` was absent. Synthetic tests cover source phasing and self-sacrifice, old incarnation after return, trigger copying, clone isolation, and registration. A public DSL declaration compiles. Kels, Fight Fixer's complete text in `data/JMP.json` requires the sacrifice event, already available before this slice; its card wiring and hybrid payment audit remain open. No JSON gap record closes. A fresh caller search found no production `CreatureYouSacrifice` use; legacy sacrifice-trigger callers remain. The next dependency-ready work is another E event participant or P payment output after a fresh requirements audit.

Validation: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Graveyard Leave Batch Event

`OneOrMoreCardsLeaveYourGraveyard().WithCount(*NumberBinding)` is a sealed battlefield event pattern for `WhenEvent` and `WhenEventModes`. The optional count is the positive `EvtCardsLeftGraveyard` amount for one event group. The pattern compares the event's graveyard player with the source controller at detection, then saves the count in that pending trigger's frame. It reuses the existing graveyard movement and trigger services. It adds no new movement executor, target, or legacy export. Production cards remain unchanged.

The contract test first failed because the constructor was absent. Synthetic engine and public DSL tests cover a two-card batch, missing cards, separate single-card moves, another player's graveyard, source phasing, departure and return under another controller, stack copying, game cloning, inspection, and registration rejection of nil, unnamed, repeated, undominated, or wrong-zone declarations. The complete Garrison Excavator and Spirit Mascot text was checked in `data/SOS.json`. Their production declarations and the broad Secrets of Strixhaven inventory and gap record remain open. Six production callers still use `WheneverOneOrMoreCardsLeaveYourGraveyardTrigger`, so that export remains. The next dependency-ready work is a separate E combat-damage batch pattern or P payment output after a fresh requirements audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Any-Player Draw Event Card Capture

`PlayerDrawsCard(player).WithCard(card)` and `OpponentDrawsCard(player).WithCard(card)` save a completed draw's card and exact hand visit in a pending battlefield trigger. `FromCard` is absent after that visit ends, including when the same card returns to hand. The patterns reuse the existing draw event, hand-visit service, card binding, and trigger frame. A prevented or failed draw emits no draw event. The constructor return types are concrete sealed patterns so the method is available; existing `EventPattern` callers remain valid. The methods work in battlefield `WhenEvent` and `WhenEventModes` bodies. Production cards and legacy exports remain unchanged.

The contract first failed at compilation because `.WithCard` was unavailable on the returned interface. Synthetic engine and public DSL tests cover both player patterns, separate draw events, failed and replaced draws, source phasing and departure, card leave and return, stack copying, game cloning, inspection, and registration rejection. `ControllerDrawsCardInto` remains a compatible earlier form. No gap record closes: the Legends Sylvan Library requirement also needs draw-step extra draws, a turn-local card collection, and per-card choices; Underworld Dreams already had its player-only draw condition and typed damage. The complete texts were checked in `data/LEG.json`. A fresh search found no production `PlayerDrawsCard` or `OpponentDrawsCard` caller; other draw helpers remain in production. The next dependency-ready work is another bounded E event participant or a P payment output after a fresh caller audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Discard Event Card Capture

`PlayerDiscardsCard(player).WithCard(card)` and `OpponentDiscardsCard(player).WithCard(card)` save the exact graveyard visit of each completed discarded card. The sealed event pattern is valid in battlefield `WhenEvent` and `WhenEventModes` declarations. The discard event now carries that visit after direct, effect, cost, and batch discards. It reuses `CardBinding`, the pending-trigger frame, and the shared discard service. A discard redirected to a library still triggers but supplies an absent `FromCard`; a later zone visit does not revive the binding. Existing calls without `.WithCard` and all legacy exports remain available. No production card changed.

The first contract test failed at compilation because `.WithCard` did not exist. Synthetic tests cover each player pattern, distinct batch events, replacement to a library, departure and return, phasing, stack copying, game cloning, public DSL compilation, inspection, and registration rejection. The complete Psychic Purge and Fell Specter text was checked in `data/LEG.json` and the Jumpstart inventory. Fell Specter's opponent-discard pattern already existed. Psychic Purge also needs a discard-cause player and an ability active from the discarded card's zone; this card capture alone does not complete it. Their production wiring and gap records remain open. A fresh caller search found no production `.WithCard` discard pattern use. The next dependency-ready work is another bounded E participant or a P payment output; T target-announcement work still depends on the shared `{2/W}` mana-cost contract.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Any-Recipient Damage Event Player Capture

`DealsDamage(This, amount).WithRecipientPlayer(player)` and the `AttachedPermanent` form save the final player recipient of a positive damage event. `FromPlayer` is absent when damage lands on a permanent. The adapter uses the shared damage event and pending trigger frame; it adds no damage executor or legacy export. The method works in battlefield `WhenEvent` and `WhenEventModes` declarations. It preserves source identity, event timing, and the existing `DealsDamageToPlayer` pattern for callers that require only player events.

The public declaration first failed because the method was absent. Focused synthetic tests cover player and permanent recipients, prevention, player-to-permanent redirection, simultaneous damage, source departure and return, phasing, clone isolation, registration rejection, and the public DSL call. The complete Pit Scorpion text in `data/LEG.json` and Marsh Viper text in `data/4ED.json` use player-only damage triggers, already supported by `DealsDamageToPlayer`; those production cards and all other callers remain unchanged. A fresh search found no production `WithRecipientPlayer` caller and two production `DealsDamage` callers; legacy exports remain. No JSON gap record closes from this additional event participant. The next dependency-ready work is another E event participant or the T announcement-X target-number prerequisite. The latter also depends on correcting `{2/W}` mana value parsing in the shared mana-cost service.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Cast Spell Creature Target Filter

`SpellCast().ByController().OfInstantOrSorcery().TargetsCreature()` is a sealed battlefield event pattern for `WhenEvent` and `WhenEventModes`. The three filters are independent methods on the existing cast pattern. `TargetsCreature` reads the exact stack object named by the cast event and checks a chosen target's current battlefield incarnation and creature type under CR 115.9b. A departed, returned, phased, noncreature, or unchosen target does not qualify. A trigger already detected remains pending after the target leaves. `ByController` reuses the event caster and trigger source controller; `OfInstantOrSorcery` checks either spell type. The adapter adds no cast, target, or trigger executor and preserves legacy exports and production cards. The plan's proposed event pattern was split into composable filters to fit the existing typed event architecture.

The full Repartee text for Conciliator's Duelist, Forum Necroscribe, and related Secrets of Strixhaven cards was checked in `data/SOS.json`. The synthetic public declaration supplies their shared trigger condition only. Their distinct effect bodies, target announcement requirements, and production wiring remain; the broad Secrets of Strixhaven gap record and inventory stay open. A fresh search found production calls to `WheneverYouCastInstantOrSorceryTargetingCreatureTrigger` in that set and no production `TargetsCreature` caller, so the legacy export remains. The next dependency-ready work is a separate T target-announcement group or P payment output after a fresh audit.

Focused contract tests first failed because `TargetsCreature` did not exist. Engine and public DSL tests cover instant and sorcery types, mixed and empty targets, the exact stack object, target departure and return, phasing, source phasing, opponent casting, trigger copying, game cloning, real casting, failed payment rollback, inspection, and registration rejection of repeated or conflicting filters. Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Graveyard Return Tapped Entry

`ReturnCardToBattlefield(CardRef, PlayerRef, ...BattlefieldEntryOption)` now accepts `EnterTapped()`. The option sets tapped status during the shared battlefield-entry preparation, before entry triggers inspect the new permanent. It does not emit a tap-transition event or change the card's printed characteristics. The existing two-argument call remains valid. The option works in spell, activation, and triggered resolving bodies. Registration rejects a nil or repeated option. This is a typed adapter over the existing entry path, not a general zone-change replacement service.

The complete Mind Roots text in `data/SOS.json` was checked. A synthetic spell now composes the discarded-card result, optional land choice, and tapped entry under the spell controller. Focused tests first failed because `EnterTapped` and the option signature were absent. They cover opponent ownership, entry-time tapped observation, no tap event, stale graveyard visits, a copied stack object, clone isolation, and registration. The public DSL declaration compiles. Mind Roots remains unwired in `cards/secretsofstrixhaven/spells.go`; its inventory and JSON gap record remain open until its production declaration is migrated and tested. A fresh search found five production `ReturnCardToBattlefield` callers, all using the two-argument form, and one internal legacy return helper caller. No production card or legacy export changed. The next dependency-ready work is a separate P payment output or T target-announcement contract after an API and caller audit.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Previous Slice: Optional Choice from a Captured Graveyard Set

`ChooseUpToOneCard(PlayerRef, CardSet, CardFilter, *CardBinding)` lets the named player choose zero or one current card from the filtered members of a saved graveyard result. `IsLandCard` supplies the land-card filter. The choice uses the existing optional and graveyard choice services and saves a zone-visit `CardRef` in the current resolution frame. Empty, declined, absent, invalid, or departed choices leave the binding absent. It works in spell, activation, and trigger resolving bodies; it is not a target. Declaration inspection checks nil roles, the set producer, binding scope, and binding name. Production cards and legacy exports remain unchanged.

The complete Mind Roots text in `data/SOS.json` was checked: the player discards two cards, then the controller may put up to one land discarded this way onto the battlefield tapped under their control. Synthetic tests first failed because the constructor and land-card filter were absent. They now cover filtering to the captured set, an opponent-owned card, declining, invalid selection, departure and return, game clone isolation, stack copy resolution, inspection, and registration. The public DSL declaration compiles. At that time, Mind Roots still needed tapped entry; the current slice provides it. Separate production wiring remains, so its Secrets of Strixhaven inventory and gap record remain open. A fresh search found no production `ChooseUpToOneCard` caller. The next dependency-ready slice at that time was a tapped-entry option for `ReturnCardToBattlefield`.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The post-lint diff was reviewed.

## Current Slice: Multi-Card Discard Graveyard Result

`BindCards`, `DiscardCardsInto`, `FromCards`, and `CountCards` save and read all current graveyard visits produced by one typed `Discard` instruction. The affected player makes the existing hand choice. Each selected card passes through the shared effect-discard replacement action. The result excludes prevented and redirected discards, loses a member after that card changes zones, and is local to the resolving instance. `CardSet` is a sealed set role; it cannot stand in for one `CardRef`. The option works in spell, activation, and trigger bodies. The two-argument `Discard` and single-card option remain compatible; no production card or legacy export changed.

The contract first failed at compilation because these constructors and roles were absent. Focused synthetic tests cover a short hand, empty result, target absence, replacement to library, full prevention, departure and return, stack copying, clone isolation, inspection through registration, and invalid, repeated, or undominated bindings. A public DSL declaration compiles and validates. The complete Mind Roots text in `data/SOS.json` requires two discards followed by a choice of up to one land from exactly those cards and tapped entry under the spell controller. This slice provides the result set only; bounded choice, tapped entry, and production wiring in `cards/secretsofstrixhaven/spells.go` remain. Its inventory and gap record stay open. A fresh search found no production `DiscardCardsInto` caller; two legacy `DiscardCards` author calls remain in Jumpstart and Fallen Empires, plus direct discard callbacks such as Mind Roots. The next dependency-ready slice is a bounded, non-targeted choice from a captured `CardSet`, with zone-visit and chooser rules.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` were reviewed.

## Current Slice: Singular Discarded Card Result

`Discard(PlayerRef, Fixed(1), DiscardCardInto(*CardBinding))` saves the exact card from one completed discard into a graveyard. It uses the affected player's existing choice and the shared effect-discard replacement service. `FromCard` checks the saved graveyard visit, so departure and return make it absent. Spell, activation, and trigger bodies can use the binding. The existing two-argument `Discard` call and all legacy exports remain available. No production card changed.

The focused contract first failed at compilation because the option was absent. Synthetic engine and public DSL tests cover identity, absent players, empty hands, replacement to the library, departure and return, registration validation, stack copying, and game clone isolation. Mind Roots's full text in `data/SOS.json` needs a result set for two discarded cards, a choice of up to one land from that set, and tapped entry under the spell controller. This singular result does not meet its gap record, so its production wiring and status remain open. A fresh search found no production caller of `DiscardCardInto`; legacy discard callers remain. Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed. The next dependency-ready P/Z slice is a multi-card discard result set with zone-visit identity and a bounded choice from that set.

## Current Slice: Paid Source Sacrifice Permanent Identity

`SacrificeMatchingIncludingSourceCostWithPermanent(PermanentFilter, string, *PermanentBinding) Cost` saves the exact chosen battlefield incarnation during a one-permanent sacrifice payment, including the ability source when it matches. It reuses the existing locked payment transaction, incarnation checks, departure LKI, and instance-local permanent binding. It works in activated costs and spell additional costs. `From(binding)` is absent after sacrifice; `DamageSourceOf(From(binding))` can use the old incarnation. The source-exclusive constructor and all legacy exports remain. No production card changed.

The contract first failed at compilation because the constructor was absent. Synthetic engine and public DSL tests cover paying with the source as the only candidate, its departure and return under another controller, saved lifelink, stack copying, game clone isolation, phasing, stale return rollback, and registration rejection of absent, invalid, and resolution-time producers. The complete Diamond Valley and Life Chisel text was checked in `data/ARN.json` and `data/LEG.json`; both still need separate card wiring, including their own timing and effect requirements. No gap record closes. A fresh search found no production caller of either permanent-binding sacrifice constructor; existing source-inclusive sacrifice callers remain. Focused tests, short engine and repository suites, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The next dependency-ready slice is a separate P payment output or T target-announcement contract after a fresh audit.

## Current Slice: Paid Sacrifice Permanent Identity

`SacrificeMatchingCostWithPermanent(PermanentFilter, string, *PermanentBinding) Cost` saves the exact battlefield incarnation chosen for one sacrifice cost. It reuses locked sacrifice choice, payment validation, stack frames, and durable departure LKI. `From(binding)` is absent after payment; `DamageSourceOf(From(binding))` can use the old incarnation as a damage source. Spell and activation bodies can read the binding, including stack copies and game clones. The constructor excludes the ability source, like `SacrificeMatchingCost`; the source-inclusive form is documented above, while multiple-permanent payment references remain separate work. It does not change production cards or legacy exports.

Momentous Fall's full text in `data/JMP.json` calls for payment-time characteristics of its sacrificed creature. The existing `SacrificeCreatureCostWithStats` already covers its power and toughness; this slice adds identity for abilities that need a former payment object. No card gap record closes from this adapter alone. A fresh search found no production caller of the new constructor; legacy sacrifice callers remain. The next dependency-ready work is a separate P payment output or T target-announcement slice after a fresh requirement audit.

Focused synthetic tests first failed because the constructor and transaction capture were absent. They cover normal and alternative-cost spell casting, activation payment, departure LKI, return as a new incarnation, phasing, stale-choice rollback, stack copies, game clone isolation, registration validation, and the public DSL declaration. The focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Singular Sacrifice Subtype Result

`SacrificeSubtypeInto(string, *NumberBinding) SacrificeResultOption` adds an
action result to the existing singular `Sacrifice(PermanentRef, ...SacrificeResultOption)`
instruction. It saves 1 only when the referenced permanent is actually
sacrificed and had the named subtype immediately before it left the battlefield;
otherwise it saves 0. The action uses the existing permanent reference,
sacrifice executor, and instance-local number frame. It does not read the
last-sacrificed slot, perform a cost payment, or create a second sacrifice path.
The option works in spell, activated, and triggered resolving bodies and cannot
share one instruction with another sacrifice result option. Registration rejects
an empty subtype, nil binding, duplicate result option, and a read without a
dominating producer. No production card or legacy export changed.

Serendib Djinn's complete Oracle text in `data/ARN.json` requires a chosen land
and damage only if an Island was sacrificed this way. A synthetic declaration
combines `ChoosePermanent(You, ControlledPermanents(IsLand), land)`, this result,
`IfPositive`, and `DealDamageFrom`. The production card still uses its legacy
callback and remains open for a separate card migration and audit. Its Arabian
inventory and JSON gap record remain open. A fresh caller search found no
production use of `SacrificeSubtypeInto`; production `Sacrifice` and legacy
sacrifice callers remain. The next dependency-ready work is another bounded P
payment output or T target-announcement slice after a fresh API and caller audit.

The focused test first failed because the option was absent. Engine and public
DSL tests cover Island and non-Island choices, an opponent's Island, absent and
phased objects, return as a new incarnation, another controller, stale result
clearing, pending activations, stack copying, game clone isolation, inspection,
and registration. Focused tests, the short engine suite, DSL registration and
closed-role tests, short repository suite, build, vet, `make test`, `make lint`,
and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Attached Aura Collection

`AttachedAuras(PermanentRef) PermanentSet` selects live, unphased Auras attached to one current battlefield incarnation. `CountPermanents` can read the set in a resolving action or a dynamic static Layer 7c value. It uses the existing attachment links and permanent-reference contract. Registration rejects a missing or undeclared host binding. It adds no attachment action, production card, or legacy export.

Rabid Wombat's complete text was checked against `data/LEG.json` and the Legends inventory and gap record. The selector supplies its dynamic Aura count, including Auras controlled by another player; its production declaration remains unchanged and the gap record remains open for card wiring and full validation. A fresh search found no production `AttachedAuras` caller. The next dependency-ready work is a bounded P payment output or T target-announcement slice after a fresh API and caller audit.

The focused contract first failed because the selector was absent. Synthetic tests cover a static bonus, reattachment, Equipment exclusion, separate Aura control, phasing, host return, clone isolation, registration, and the public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Previous Slice: Rounded Resolution Division

`DivideRoundDown(ResolutionValue, ResolutionValue)` and `DivideRoundUp(ResolutionValue, ResolutionValue)` supply explicit rounding for typed resolution arithmetic. They reuse the existing arithmetic value, resolution frame, definition inspector, and action consumers; no combat, damage, or payment service changed. A dynamic zero divisor yields zero under CR 107.2, and registration rejects a fixed zero divisor. Signed intermediate values use mathematical floor or ceiling before the consuming action applies its own result rules. Both constructors work in spell, activation, trigger, and supported continuous-modification value contexts. No production card or legacy export changed.

Banshee and Eternal Flame were checked against their complete text in `data/DRK.json`. The half-X amount can now be declared, but Banshee still needs its activation cost, target, and two damage instructions wired; Eternal Flame still needs its Mountain count, target, and card wiring. Fireball requires a separate target-count payment and equal damage allocation audit. Their inventory and gap statuses remain open. A fresh search found no production caller of the new constructors. The next dependency-ready work is a bounded target-count/payment or event-participant slice after a fresh API audit.

The focused contract first failed at compilation because the constructors were absent. Tests cover odd, even, zero, and signed values; fixed and dynamic zero divisors; saved binding use; clone isolation; and registration rejection of nil, unbound, and fixed-zero operands. The public DSL example is compiled and validated in a synthetic test.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed. Untracked inventories and unrelated workspace files remain untouched.

## Current Slice: Event-Time Produced Mana Type Choice

`BindManaTypes(name)`, `LandTappedForMana().WithManaTypes(binding)`, and `AddManaOfProducedTypeToPlayer(player, binding, amount)` capture the land ability's produced mana types per trigger and add one chosen type to the selected player's pool. The event, trigger frame, and shared mana action supply timing and execution. The typed action is available in targetless battlefield `WhenEvent` and `WhenEventModes` bodies and resolves immediately as a triggered mana ability. Empty type sets and absent players add no mana; a choice outside the produced set uses its first type. No production card or legacy export changed.

Mana Flare's complete text was checked in `data/4ED.json` and its Limited gap record. This slice supplies its event condition, player, type choice, and mana action, but the production definition still uses `NewManaFlareAbility`; its gap remains blocked pending card migration and full card review. Wild Growth also remains on its legacy helper. The focused contract first failed because the binding and action were absent. Tests cover a single type, multiple types, another recipient, an invalid choice, immediate timing, phasing, clone isolation, and registration rejection. Existing mana event capture and activated mana tests remain. The next dependency-ready work is another bounded P payment result or T division declaration after a fresh requirement audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh caller search found the two legacy helpers in `cards/limited/enchantments.go` and no production use of the new action. The post-lint diff was reviewed.

## Current Slice: Tapped-for-Mana Triggered Mana Timing

Targetless `WhenEvent(LandTappedForMana(), body)` triggers with a direct or sequenced mana action now resolve immediately after the producing tap-cost mana ability, using the existing captured event frame and mana action. Ordinary triggers from before or during that resolution stay pending. Targeted and non-mana triggers keep ordinary timing. No public constructor, production card, or legacy export changed. The supported public declaration is `WhenEvent(LandTappedForMana().WithPlayer(player), AddManaToPlayer(FromPlayer(player), core.Green, Fixed(1)))` on a battlefield permanent. This slice does not cover conditional mana bodies, mana-added or activated-mana event triggers, event-time mana-type choice, or card wiring.

Contract tests first failed because the mana trigger remained pending. Focused tests cover immediate mana, a different recipient and controller, earlier and newly created ordinary triggers, target exclusion, source phasing, game clone isolation, registration, and the existing recipient binding test under the corrected timing. Focused tests, the short engine suite, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. Wild Growth and Mana Flare still have production callers on their legacy helpers in `cards/limited/enchantments.go`; Mana Flare still needs an event-time mana-type choice. Their gap records remain blocked. The next dependency-ready slice is a typed event-time mana-type choice and its bound result, after a fresh API and caller audit.

## Current Slice: Typed Mana Player Recipient

`AddManaToPlayer(PlayerRef, core.Color, ResolutionValue) ResolvingEffect` selects a player in the original resolution frame, evaluates the amount once, and uses the shared mana action to add one concrete type to that player's pool. An absent recipient or nonpositive amount adds nothing. Targetless activated abilities using this action are classified as mana abilities and resolve immediately. The automatic payment planner does not advertise this action's output because the recipient need not be its controller. Registration checks missing operands, invalid mana types, and unbound player bindings. The sealed player and number roles reject category errors at compile time. Existing `AddManaAmount` and legacy exports remain unchanged.

Synthetic tests first failed because the constructor was absent. They cover event-player capture after the land leaves and returns, different trigger controller and recipient, game clone isolation, absence, registration, public DSL use, and activated mana timing. Wild Growth's complete text in `data/4ED.json` requires mana for the enchanted land's controller, but its triggered mana ability must also resolve immediately under CR 605.1b and 605.4a. Mana Flare needs an event-time mana-type choice. Neither card or gap record is complete; no production card was changed. A fresh caller search found no production `AddManaToPlayer` use. The next dependency-ready slice is triggered mana classification and immediate resolution, followed by event-time mana-type choice.

Focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed. No gap record changes because the required triggered timing and production card wiring remain open.

## Current Slice: Attached Land Tapped-for-Mana Filter

`LandTappedForMana().OfAttachedLand()` matches a tapped-for-mana event only when its producing land incarnation is attached to the battlefield trigger source at event time. It composes with the existing mana-type filter and land, player, and amount captures in `WhenEvent` and `WhenEventModes`. It reuses the mana event and attachment incarnation; it adds no new event or mana executor. Registration rejects a repeated filter, and inspection describes the attached-land pattern. Synthetic tests cover an attached land, another land, a returned incarnation, phasing, attachment changes after detection, trigger copying, game clone isolation, registration, and a public DSL declaration. Focused tests first failed because the method was absent, then passed after implementation. No production card or legacy export changed.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` pass.

Wild Growth's complete text in `data/4ED.json` still needs a typed action that adds mana to the enchanted land's controller with triggered mana timing. Mana Flare still needs a choice from the event-time set of mana types and an action that adds the chosen mana to the activating player. `cards/limited/enchantments.go` still uses `NewAttachedManaBonusAbility` and `NewManaFlareAbility`; neither gap record is closed. The next dependency-ready work is the triggered mana-production contract and typed player recipient, after a focused engine audit.

## Current Slice: Tapped-for-Mana Type Event and Filter

`EvtTappedForMana.ManaTypes` now records each distinct concrete mana type produced by a tap-cost mana ability. Ordinary taps, direct mana activation, typed immediate activation, and planned payment pass the chosen types through the existing mana service. `LandTappedForMana().OfManaType(core.Color)` is a sealed event filter for battlefield `WhenEvent` and `WhenEventModes`; it composes with the existing land, player, and amount captures. Registration rejects `AnyColor`, invalid types, and duplicate filters. The event copies its type list. Existing production cards and author exports are unchanged.

Contract tests first failed because the event field and filter method were absent. Focused engine and public DSL tests cover mixed production, a planned color choice, source incarnation, controller, amount, matching and nonmatching filters, and registration rejection. The focused run, short engine and repository runs, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh production search found `NewManaFlareAbility(IsLand)` in `cards/limited/enchantments.go` and no `LandTappedForMana` or direct `FireTappedForMana` caller. Mana Flare's complete text in `data/4ED.json` requires a resolution-time choice among the types the land produced; Wild Growth needs attached-land matching and a triggered mana action. Those contracts and production card wiring remain open, so the Limited and Fourth Edition gap records stay blocked. The next dependency-ready slice is a typed event-time mana-type set capture or triggered mana-production action, after a fresh audit.

## Current Slice: Tapped-for-Mana Land Incarnation

`LandTappedForMana().WithLand(*PermanentBinding)` saves the exact producing land in each pending battlefield `WhenEvent` or `WhenEventModes` trigger. `From(land)` is absent after departure and return or while phased out. The mana event now records `SourceIncarnation` before production and post-production effects. Its land filter uses that incarnation or departure LKI, rather than a later object with the same card ID. This reuses the existing mana event, trigger capture frame, permanent binding, and copy/clone paths. No production card or legacy author export changed. The engine call `ManaSystem.FireTappedForMana` now takes the captured source incarnation.

The first card requirement is Wild Growth's enchanted-land mana trigger in `data/4ED.json`; it still needs an attached-land filter and triggered mana timing. Mana Flare also needs the types produced by the land. Both need separate card wiring. A fresh search found `NewManaFlareAbility(IsLand)` in `cards/limited/enchantments.go` and no production `LandTappedForMana().WithLand` caller. No gap record closes. The next dependency-ready slice is mana-type event capture or another E participant, after a fresh audit.

Contract tests first failed because `WithLand` was absent. Focused tests cover the event incarnation, trigger capture, source return, phasing, trigger copying, game clone isolation, combined player/amount bindings, registration rejection, and a public DSL declaration. The focused run, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` pass.

## Current Slice: Land Tapped for Mana Amount Capture

`LandTappedForMana().WithAmount(*NumberBinding)` saves the positive `EvtTappedForMana.Amount` in each pending battlefield trigger. It composes with the existing `WithPlayer` capture in `WhenEvent` and `WhenEventModes`. The adapter reuses the mana event, number binding frame, inspection, and stack/game copy paths. Zero production sends no event. This slice does not capture mana types or add a triggered mana-production action; Gauntlet of Might, Mana Flare, and Wild Growth still need those contracts and card wiring. No production card, legacy export, or gap record changed.

The contract test first failed because `WithAmount` was absent. Focused tests cover the event value, stack copy, clone isolation, source departure, invalid or undominated bindings, and a public DSL declaration composed with `WithPlayer`. Existing tapped-for-mana tests cover nonland and zero production, source phasing, and source return. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` pass. The next dependency-ready work is mana-type event capture with triggered mana timing, or a separate E event participant after a fresh audit.

## Current Slice: Spell-Cast Ownership Filter

`SpellCast().NotOwnedByCaster()` matches a completed cast when the exact cast stack object's card owner differs from the event caster. It composes with existing cast filters in battlefield `WhenEvent` and `WhenEventModes`. The existing cast event, stack identity, trigger lifecycle, and registration inspection supply the behavior; no new engine contract, production card, or legacy export changed. The complete Nita, Forum Conciliator text in `data/SOS.json` needs this filter for its first ability, but graveyard exile, cast permission, replacement exile, and card wiring remain open. No gap record closes. The next dependency-ready work is a separate P payment/result or E event-participant slice after a fresh requirements audit.

Focused tests first failed because the method did not exist. Synthetic tests cover owner versus caster, exact stack identity, real foreign exile casting, failed casting, source phasing, trigger copying, clone isolation, inspection, registration rejection, and a public DSL declaration. The existing `SpellCast` export remains compatible with production callers.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` pass. A fresh production search found no `NotOwnedByCaster` caller; existing cast-event callers remain unchanged.

## Current Slice: Spell-Cast Origin Zone Filter

`SpellCast().FromZone(core.Zone) EventPattern` matches the origin of the exact cast stack object during `EvtSpellCast`. It reuses `StackObject.CastZone`, the existing cast event and battlefield trigger lifecycle. It composes with caster and other cast filters. Registration rejects duplicate filters, `ZoneStack`, `ZoneAny`, and invalid zones. Normal hand casts and explicit exile casts are covered; failed casts produce no trigger. No production card or legacy export changed. The complete Quandrix, the Proof requirement in the Secrets of Strixhaven inventory needs a hand-cast filter for its cascade grant, but that static grant and card wiring remain open. No gap record closes from this adapter alone. The next dependency-ready work is a separate E ownership filter or P payment/result slice after a fresh requirements audit.

Contract tests first failed because `FromZone` was absent. Focused tests cover exact stack identity even with another cast of the same card, normal hand and exile casts, failed payment, source phasing, clone isolation, trigger copying, inspection, registration validation, and a public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` pass. A fresh production search found no `SpellCast().FromZone` caller; the typed adapter remains available for a separate card migration.

## Current Slice: Any-Recipient Damage Event Source Capture

`DealsDamage(This, amount).WithSource(source)` and `DealsDamage(AttachedPermanent, amount).WithSource(source)` bind the final positive damage event's source in a pending battlefield trigger for player or permanent recipients. `FromDamageSource` uses the captured incarnation and last-known characteristics after departure or return. The adapter reuses the shared damage snapshot and event capture frame. It changes no damage executor, production card, or legacy export. `DealsDamageToPlayer` remains the recipient-filtered pattern. This source capture does not complete Spirit Link, whose complete text in `data/LEG.json` already maps to `DealsDamage(AttachedPermanent, amount)` but still needs card wiring. The two production `DealsDamage` callers in Arabian Nights and Jumpstart and the legacy damage-trigger callers in Legends and Jumpstart remain unchanged. No gap record closes from this added participant. The next dependency-ready work is another E event participant or a P result after a fresh requirements audit.

Contract tests first failed because registration rejected the generic source binding. Focused tests cover player and permanent recipients, prevention, simultaneous damage, source departure and return, phasing, stack copy, clone isolation, inspection, and declaration rejection. A public DSL declaration compiles. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` pass.

## Current Slice: Damage Event Permanent Recipient Capture

`DealsDamage(This, amount).WithPermanent(binding)` saves the final permanent recipient's battlefield incarnation when positive damage occurs. `From(binding)` is absent if that permanent leaves and returns before the pending trigger resolves. The damage service sets `EvtDamageDealt.TargetIncarnation` after replacements and prevention; the adapter reuses the event and trigger capture frame. It supports battlefield `WhenEvent` and `WhenEventModes`, including `AttachedPermanent` as the source. Player damage leaves the permanent binding absent. No damage executor, production card, or legacy export changed.

Contract tests first failed because the method was absent. Focused tests cover recipient return, prevention, redirection, source and recipient phasing, simultaneous damage, stack copy, clone isolation, inspection, and registration rejection. A public DSL declaration compiles. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. A fresh search found two production `DealsDamage` callers in Arabian Nights and Jumpstart; both remain unchanged for separate audit and wiring. No gap record closes from this event participant alone. The next dependency-ready work is another E event participant or a remaining P payment result, after a fresh caller audit.

## Current Slice: Controller Damage Event Source Capture

`ControllerIsDealtDamage(amount).WithSource(source)` binds the final positive damage event's source to a pending battlefield trigger. `FromDamageSource(source)` uses its captured incarnation, controller, and characteristics after departure or return. The adapter uses the shared damage snapshot and event capture frame; it adds no damage executor. The return type is now the concrete sealed pattern so authors can select `WithSource`; existing declarations remain valid. No production card or legacy export changed.

Contract tests first failed because `WithSource` was absent. Synthetic tests cover full prevention, source departure and return, phasing, lifelink, stack copying, game cloning, inspection, and registration rejection. A public DSL declaration compiles. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass. A fresh production search found no `ControllerIsDealtDamage` or `DealsDamageToPlayer` callers; one `WheneverEnchantedPermanentDealsDamageToPlayerTrigger` and one `DealsDamageToOpponentTrigger` caller remain on legacy exports. No gap record closes because cards remain unwired. The next dependency-ready work is an event participant or simultaneous damage slice after a fresh API and caller audit.

## Current Slice: Player Damage Event Source Capture

`DealsDamageToPlayer(This, player).WithSource(*DamageSourceBinding)` saves the final positive damage event's source in a pending battlefield trigger. `AttachedPermanent` is also accepted by the existing player-damage pattern. `FromDamageSource` uses the captured incarnation, controller, and characteristics after departure or return. The adapter reuses the damage service snapshot and event capture frame. It adds no damage executor or card policy. No production card, legacy author export, or gap record changed. Existing card wiring remains separate.

Contract tests first failed because `WithSource` was absent. They cover prevention, source departure and return, phasing, source lifelink, stack copying, game cloning, inspection, registration rejection, and a public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass. A fresh production search found no `DealsDamageToPlayer` callers; the one `WheneverEnchantedPermanentDealsDamageToPlayerTrigger` caller and one `DealsDamageToOpponentTrigger` caller remain on their legacy exports. This slice does not add a controller-damage event source capture, a generic `DealsDamage` source capture, batch damage, or permanent recipient identity. The next dependency-ready work is controller-damage source capture after a fresh caller audit.

## Current Slice: Received Damage Source Capture

`ReceivesDamage(This, amount).WithSource(*DamageSourceBinding)` saves the final damage action's source when a battlefield source receives positive damage. `FromDamageSource` can use its captured incarnation, controller, and characteristics in that pending trigger, including after a permanent leaves and returns, phases out, or a spell leaves the stack. The shared damage dispatcher carries a source snapshot with each damage event, including events held during simultaneous damage. This reuses the existing damage pipeline, pending-trigger capture, resolution frame, and source cloning. It adds no damage executor, recipient binding, or card-specific rule. `WithSourceController` and all legacy author exports remain available.

The complete Belltower Sphinx text in `data/JMP.json` needs the actual damage amount and source controller, already supplied by `ReceivesDamage(...).WithSourceController(...)`; this source binding is additional reusable capability. Its production card is unchanged and its Jumpstart gap record remains open for card wiring and full review. No gap record is closed. A fresh search found no production `ReceivesDamage` or `DealDamageFrom` callers. The next dependency-ready work is a separate damage-source capture for controller or player damage events, after checking the full caller requirements; do not infer that this pattern covers those events.

Contract tests first failed because `WithSource` did not exist. Focused tests now cover prevention, redirection, a phased recipient, a phased source, departure and return, a spell after stack departure, two simultaneous sources, stack copying, clone isolation, inspection, and registration rejection of nil, unnamed, repeated, or unproduced bindings. A public DSL declaration compiles. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` pass. `make lint` passes with zero issues after one unused-parameter fix. The post-lint diff and `git diff --check` pass.

## Current Slice: Damage Event Source Incarnation

The shared damage service now records the captured `DamageSource.Incarnation()` in `EvtDamageDealt.SourceIncarnation` for positive damage to a player or permanent. This is the source incarnation after the existing replacement and prevention pipeline, and the event keeps it if the physical card leaves and returns. A spell source has no battlefield incarnation. Full prevention and zero damage emit no damage event. This prerequisite changes no public DSL signature, card, or legacy author export. The existing `ReceivesDamage(This, amount).WithSourceController(controller)` signature and its supported battlefield trigger contexts remain as documented in `docs/dsl-reference.md`.

Belltower Sphinx's complete text in `data/JMP.json` requires the amount and damage source controller; the typed event adapter already supplies those values, but production card wiring remains open. The incarnation field supports source bindings that need exact source identity. No gap record is closed by this prerequisite. The received-damage source binding is implemented in the current slice above.

Contract tests first failed because both damage event paths emitted a nil source incarnation. The tests cover departure and return, player and permanent recipients, full prevention, simultaneous assignments, and a spell source. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff was reviewed and `git diff --check` passes.

## Current Slice: Exact Multi-Sacrifice Total Power

`SacrificeNMatchingCostWithTotalPower(count, filter, text, total)` records the sum of current power for the exact, locked set of other permanents sacrificed as a direct spell additional cost or activated ability cost. It reuses the prepared-payment transaction, incarnation checks, sacrifice service, and instance-local number frame. The power snapshot is taken before any chosen permanent is sacrificed. A phased or returned choice fails without a result. Inspection and registration reject missing, unnamed, duplicate, or out-of-scope results and invalid count or text. The public DSL keeps `SacrificeNMatchingCost` and all production callers unchanged.

The full Sword of the Ages text in `data/LEG.json` requires an any-number choice, source sacrifice, linked exile, and damage; this exact-count output does not complete that card or its gap record. Fixed-count sacrifice callers in Fourth Edition and Fallen Empires still use the legacy constructor and need individual audits before migration. No production card or JSON gap record changed. The next dependency-ready slice is an any-number payment collection contract or T recorded target groups after a fresh requirements audit.

Focused contract tests first failed because the constructor was absent. They now cover derived power, locked choices, phasing and changed incarnation, failed-payment result absence, independent pending activations, stack copying, game cloning, registration, and a public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Paid Graveyard Exile Card

`ExileMatchingCardFromGraveyardCostWithCard(CardFilter, string, *CardBinding) Cost` saves the card exiled from the payer's graveyard by a direct spell additional cost or activated ability cost. The prepared-payment transaction locks its graveyard visit; the committed payment captures its new exile visit in the spell or ability frame. `FromCard` is absent after that card leaves exile, even if it returns. Exile now records a visit ID for face-up and face-down entries, and game cloning retains it. This reuses the existing cost choice, exile move, and frame services; it adds no new executor, replacement, or card policy. The legacy cost and its one production caller, Necropolis, remain unchanged. No production card or JSON gap record changed.

Necropolis's complete text in `data/DRK.json` requires the exiled card's mana value for counters. The existing `ExileMatchingCardFromGraveyardCostInto` supplies that number, while this slice supplies card identity for other paid-card rules. Necropolis still needs separate card wiring and a complete audit. This slice excludes choosing an opponent's graveyard, multiple exiled cards, and card references across other zones. The next dependency-ready work is a P collection payment output or the T target-group announcement boundary, after a fresh requirements audit.

Focused tests first failed because the typed constructor was absent. Synthetic tests cover activation and spell costs, locked graveyard visits, exile departure and return, failed-payment rollback, stack copying, game cloning, inspection, registration, and a public DSL declaration. A compile-failure fixture rejects a number binding in the card-binding role.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff and `git diff --check` pass. A fresh production search found one legacy graveyard-exile cost caller and no new adapter caller.

## Current Slice: Reverse Turn Damage Source Set

`PermanentsThatDealtDamageToThisTurn(PermanentRef, PermanentSet) PermanentSet` selects current, unphased permanents that dealt positive damage to one recipient incarnation this turn. `This` uses the resolving trigger's captured source incarnation, including after death and return. Other recipient references must resolve to a battlefield permanent. The adapter reads the existing source-and-recipient turn record after the shared damage pipeline; it adds no damage executor or historical card-ID lookup. It works in resolving spell, activation, and trigger bodies and exposes a sealed `PermanentSet` with inspection and registration checks. No production card or legacy export changed.

The complete Brine Hag text in `data/LEG.json` needs creatures that damaged it this turn, including its death-trigger recipient identity. This selector supplies that collection. A set form of indefinite base-power-and-toughness setting is still absent from the typed continuous API, so the card and its Legends gap record remain open. Focused tests first failed because the selector was missing. They now cover prevention, source and recipient return, phasing, recipient LKI cleanup, final cleanup, game clone isolation, death-trigger resolution after return, inspection, registration, and a public DSL declaration. The next dependency-ready slice is a set base-power-and-toughness modification or a P payment output after a fresh contract audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff was reviewed and `git diff --check` passes.

## Current Slice: Turn Damage Recipient Set

`PermanentsDealtDamageByThisTurn(DamageSourceRef, PermanentSet) PermanentSet` selects current, unphased permanent candidates dealt positive damage by one permanent source incarnation this turn. It uses the existing source-and-recipient incarnation record from the shared damage pipeline. Prevention and redirection determine the final record; final cleanup clears it. The selector works in resolving spell, activation, and trigger bodies with permanent-set consumers. Nil arguments and out-of-scope source bindings fail registration; the sealed source and set roles reject category errors at compile time. No production card or legacy export changed.

The D/S inventory includes damage-history collections. The full Brine Hag requirement in `doc/designs/composable-dsl-migration-legends-gaps.json` needs the reverse collection of creatures that damaged the source, so this selector does not complete it. Giant Shark also needs combat opponent capture and an Island-dependent restriction. Their gap records remain open. Synthetic tests cover prevention, source and recipient return, phasing, final cleanup, game clone isolation, inspection, registration, and a public DSL declaration. The focused test first failed because the selector was absent. The next dependency-ready work is a separate reverse source-history collection or a P payment output after a fresh audit. Production card wiring remains open.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production caller of this selector. The post-lint diff was reviewed.

## Current Slice: Controller Drawn-Card Event Capture

`ControllerDrawsCardInto(*PlayerBinding, *CardBinding) EventPattern` saves the source controller and the exact card from each completed draw. `EvtCardDrawn` now carries that card's ID and current hand-visit ID. `FromCard` is absent after the card leaves and returns. The pattern uses `WhenEvent` or `WhenEventModes` on a battlefield source. It has no target and gives no drawn-card binding for opponent or any-player patterns, which preserves hidden-hand access. The existing draw replacement, event, pending-trigger, resolution-frame, and clone services remain responsible for execution. No production card or legacy export changed.

Focused tests first failed because the constructor was absent. Synthetic tests cover distinct draws, empty-library attempts, source phasing, hand departure and return, pending-trigger copying, game cloning, and registration rejection for nil or unproduced bindings. A public DSL declaration compiles. Existing draw-event tests cover replaced draws and source lifecycle. Card wiring remains open; the next dependency-ready work in delivery step 3 is a separate P payment output or T target-announcement contract after a fresh audit. No gap record closes from this adapter alone.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `ControllerDrawsCardInto` caller and two `ControllerDrawsCard` callers in Jumpstart and The Dark; both remain unchanged. The post-lint diff was reviewed.

## Current Slice: Cast-Event Spell Binding

`BindSpell(name)`, `SpellCast().WithSpell(binding)`, and `FromSpell(binding)` save the cast spell's stack object ID for a battlefield `WhenEvent` or `WhenEventModes` body. The sealed `StackObjectRef` works with existing `Counter`, `ManaValueOfSpell`, and `ControllerOfSpell`. It uses the cast event's stack ID and the instance-local resolution frame. An absent spell is not found by card ID after another cast. Trigger copies keep the binding; game clones copy its frame. This event participant is not a target. Registration rejects nil, unnamed, repeated, and out-of-scope bindings. No production card or legacy export changed.

In the Eye of Chaos, Invoke Prejudice, Nether Void, and Presence of the Master need this stack participant according to their complete text in `doc/designs/composable-dsl-migration-legends-gaps.json`; their payment, filtering, and production wiring still need individual audits. Their gap records remain open. A fresh search found no production `WithSpell` calls. The next dependency-ready work is a P payment output or another E event participant after a fresh requirements audit.

Focused tests first failed because the binding API was absent. Synthetic tests cover exact stack identity, same-card recast, absence, trigger copying, source phasing, game clone isolation, inspection, scope validation, and a public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass with zero failures. The post-lint diff and `git diff --check` pass.

## Current Slice: Opponent Artifact Tap Filter

`ArtifactBecomesTapped().ByOpponent()` matches a tap transition only when an opponent controls the artifact at the event. It composes with `WithArtifact` and `WithController` in battlefield `WhenEvent` and `WhenEventModes` declarations. It reuses `EvtTapped`, the event source controller condition, and the pending-trigger capture frame. A repeated filter fails registration. No production card or legacy export changed.

Powerleech's complete text in `data/ATQ.json` requires this tap pattern and the separate opponent artifact activation without `{T}` pattern. Both adapters are now available, but its production declaration and full card validation remain open. Artifact Possession and Haunting Wind have separate attachment and damage requirements. Their Antiquities gap record stays open. A fresh caller search found no production `ArtifactBecomesTapped` use and one `OpponentActivatedArtifactNoTapCost` use. The next dependency-ready work is a P payment output or another E event participant after a fresh audit.

Focused tests first failed because `ByOpponent` was absent. Synthetic tests cover owner/controller distinction, repeated taps, source phasing, tapped artifact return, event-time controller capture, trigger copy, clone isolation, inspection, registration rejection of repeated filters, and a public DSL declaration. Existing artifact tap tests cover type selection, artifact phasing, source departure, and invalid binding scope. Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass with zero failures. The post-lint diff and `git diff --check` pass.

## Current Slice: Activated Ability Without Tap Cost Filter

`ActivatedAbilityEvent().WithoutTapCost()` filters completed activations by the existing activation-cost flag. It composes with `FromArtifact`, `ByOpponent`, `WithActivator`, and `WithSource` in battlefield `WhenEvent` and `WhenEventModes` declarations. It reuses the event and pending-trigger services, and leaves the separate `ArtifactAbilityActivatedWithoutTapCost` pattern and legacy callers intact. No production card changed.

Powerleech's complete text in `data/ATQ.json` requires an opponent artifact activation without `{T}` and an opponent artifact tap event. The new filter supplies the first pattern, but card wiring remains open. Artifact Possession and Haunting Wind still need their separate attachment, tapped-artifact, controller, and damage requirements. Their JSON gap records remain open. A fresh caller search found one production `OpponentActivatedArtifactNoTapCost` use and no production `ActivatedAbilityEvent` use. The next dependency-ready work is a P payment output or another E event filter after a fresh audit.

Focused tests first failed because `WithoutTapCost` was absent. Synthetic tests cover artifact and opponent selection, `{T}` exclusion, ordinary and mana activations, failed payment, source sacrifice and return, saved lifelink, phasing, trigger copy, clone isolation, registration rejection of repeated filters, and a public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass. The post-lint diff and `git diff --check` pass.

## Current Slice: Opponent Artifact Activation Filter

`ActivatedAbilityEvent().FromArtifact().ByOpponent()` adds two independent filters to the completed activated-ability event pattern. `FromArtifact` checks the event's source incarnation against the live permanent or its departure LKI. `ByOpponent` compares the recorded activator with the watcher controller at detection. Both filters compose with `WithActivator` and `WithSource` in battlefield `WhenEvent` declarations. They include activations with a tap cost. The reusable boundary is the existing activation event, trigger conditions, and pending-trigger binding frame; no payment executor or legacy export changed.

Powerleech's complete text in `data/ATQ.json` requires both an opponent artifact tap event and an opponent artifact activation without `{T}` in the cost. The existing `ArtifactAbilityActivatedWithoutTapCost` pattern provides the tap-cost exclusion; this new general filter does not. Artifact Possession and Haunting Wind have separate attached-artifact and damage requirements. All three production cards remain unchanged, and their JSON gap records remain open. A fresh caller search found one production `OpponentActivatedArtifactNoTapCost` use and no production `ActivatedAbilityEvent` use. Next dependency-ready work is a separate P payment output or E event filter after a fresh requirements audit.

The focused test first failed because the filter methods were absent. Synthetic tests cover opponent and artifact selection, a tap-cost activation, activation-cost sacrifice and return under another controller, source lifelink, trigger copying, clone isolation, registration rejection of repeated filters, and a public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass. The post-lint diff was reviewed, and `git diff --check` passes.

## Current Slice: Activated Ability Event Source Capture

`ActivatedAbilityEvent().WithSource(*DamageSourceBinding)` saves the activated permanent's event-time incarnation and characteristics in each pending trigger. `FromDamageSource` can use it for later damage even when activation sacrificed the permanent and the card returned under a different controller. The capture reuses `EvtAbilityActivated`, incarnation-indexed LKI, the shared damage-source snapshot, and the resolution frame. It composes with `WithActivator` in battlefield `WhenEvent` and `WhenEventModes` declarations. No artifact, attached-source, tap-cost, or opponent filter was added. No production card or legacy export changed.

The complete Artifact Possession, Haunting Wind, and Powerleech requirements in `data/ATQ.json` still need their separate filters and typed card wiring. Their gap records remain open. Focused tests first failed because `WithSource` did not exist, then passed with activation-cost sacrifice, leave and return, changed controller, saved lifelink, source phasing, trigger copy, game clone, inspection, registration validation, and a public DSL declaration. Other activation tests cover mana and planned mana event publication, failed payment, and phased watchers. Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, and `make test` pass. The next dependency-ready slice is an E-family activation-source filter or a P payment output after a fresh requirements audit.

## Current Slice: Activated Ability Event Source Identity Prerequisite

`EvtAbilityActivated.SourceIncarnation` now records the ability source's battlefield incarnation for ordinary, direct mana, and planned mana activations. A successful activation that sacrifices its source still emits the old identity; a returned card cannot supply it. The planned mana path now emits the same completed-activation event after successful production. Failed activations emit no event. This is an engine prerequisite for an E-family typed ability-source binding; it does not expose that binding or change the public DSL signature. `ActivatedAbilityEvent().WithActivator(*PlayerBinding)` remains available on battlefield `WhenEvent` and `WhenEventModes` declarations, with its existing activator capture and no source reference.

The complete Artifact Possession, Haunting Wind, and Powerleech requirements in `data/ATQ.json` need artifact, tap-cost, attached-artifact, or opponent filters in addition to event identity. Existing `ArtifactAbilityActivatedWithoutTapCost` supplies some of those requirements. All three production cards still declare `EvtAbilityActivated` directly in `cards/antiquities/enchantments.go`; their gap records remain open. The focused synthetic test first failed for the missing incarnation in three activation paths, then for the missing planned-mana event. It now covers all four paths and a source sacrificed and returned before inspection. Existing activation tests cover failed payment, source phasing, pending-trigger copying, game cloning, registration, and inspection. No card or legacy export changed. The next dependency-ready slice is a typed activated-ability source binding that reads this identity and captures event-time characteristics as required.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The post-lint diff was reviewed and `git diff --check` passes.

## Current Slice: Activated Ability Event Player Capture

`ActivatedAbilityEvent().WithActivator(*PlayerBinding)` is a battlefield `WhenEvent` and `WhenEventModes` pattern. The binding is optional. It matches the existing completed `EvtAbilityActivated` event for normal and mana abilities and saves `GameEvent.PlayerID` in the pending trigger's instance-local frame. A failed activation produces no trigger. This slice adds no ability-source binding, artifact filter, mana-production detail, or payment rule. The proposed `AbilityActivated` constructor name would collide with the existing core ability-type constant, so the implemented name is `ActivatedAbilityEvent`. No production card or legacy export changed.

The E-family inventory requests an activated-ability event. Complete Antiquities text in `data/ATQ.json` shows that Artifact Possession, Haunting Wind, and Powerleech also need source, artifact, tap-cost, and controller conditions; the existing narrower `ArtifactAbilityActivatedWithoutTapCost` pattern supplies some of these. This generic player capture alone does not complete those cards or close their gap records. A fresh production search found three direct `EvtAbilityActivated` declarations in `cards/antiquities/enchantments.go`; no production caller uses the new pattern. The next dependency-ready work is a separate E ability-source event binding or a P payment output after a fresh requirements audit.

The focused contract first failed because the constructor was absent. Synthetic tests cover two activators, ordinary and mana abilities, failed payment, source departure and return, phasing, stack copy, game clone, inspection, registration, modal use, and the public DSL declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Source Becomes Targeted by Spell or Ability

`BecomesTargeted(This).ByOpponent().WithController(*PlayerBinding)` is a battlefield `WhenEvent` and `WhenEventModes` pattern. Both options are optional. It matches target announcement for a spell or activated ability, compares the event player with the source controller at detection, and saves that player in the pending trigger's instance-local frame. It reuses `EvtBecomesTarget` and the existing trigger capture service. The older spell-only `BecomesTargetOfSpell` remains. A fresh search found no production caller of either typed pattern; five production `EvtBecomesTarget` declarations remain in Jumpstart and Secrets of Strixhaven, including Tenured Concocter. No production card changed.

Tenured Concocter's complete text in `data/SOS.json` requires the opponent-controlled spell-or-ability event. Its optional draw, Infusion clause, and production wiring remain separate. No gap record closes from adapter availability alone. Synthetic tests first failed because `BecomesTargeted` was absent. They cover spell and ability event kinds, actual activated targeting, opponent filtering, independent event captures, source departure and return, phasing and absence, stack copying, game cloning, inspection, registration, and a public DSL declaration. The next dependency-ready work is a separate P payment output or E event participant after a fresh audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Cast-Event Announced-X Capture

`SpellCast().WithAnnouncedX(*NumberBinding)` saves the cast spell's X value when `EvtSpellCast` occurs. It is available in battlefield `WhenEvent` and `WhenEventModes` bodies and composes with the existing cast filters. `GameEvent.CastAnnouncedX` copies the value from the identified stack object before event dispatch. The pending trigger stores it in its instance-local number frame. The adapter uses the existing cast event and binding services; it adds no casting, payment, or resolution action. Registration rejects nil, unnamed, repeated, and out-of-scope bindings. No production card or legacy export changed.

The plan's E-family cast-event requirement includes X capture. The complete Matterbending Mage text in `data/SOS.json` needs only the existing `WithXInManaCost` filter, so this capture does not close that gap. Other cast-event consumers can use the value, but their remaining requirements and card wiring need separate audits. The focused test first failed because `WithAnnouncedX` was absent. Synthetic tests cover zero and positive X, spell departure, phasing, source leave and return, stack copying, game cloning, inspection, registration, and the public DSL declaration. The next dependency-ready work is a separate P payment output or E event participant after a fresh audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Any-Player Life-Loss Event

`PlayerLosesLife(*PlayerBinding).WithAmount(*NumberBinding)` matches each positive completed life-loss event for either player. The amount option is optional. It saves the player and event amount in each pending trigger's instance-local frame and works in battlefield `WhenEvent` and `WhenEventModes` bodies. The existing `OpponentLosesLife` keeps its event-time opponent filter and now shares the same pattern implementation. The adapter reuses `EvtLifeLost` and the existing trigger lifecycle. It adds no life-loss executor, payment rule, replacement, or target. No production card or legacy export changed.

The E plan calls for general life-event participants. Synthetic tests first failed because `PlayerLosesLife` was undefined. Focused tests cover both players, direct loss and damage, zero and fully prevented damage, simultaneous loss, source phasing and departure, stack copies, game clones, inspection, registration, and a public DSL declaration. Existing opponent-loss tests cover event-time controller changes and delayed and modal bodies. A fresh search found one production `OpponentLosesLife` caller, two test-only callers, and no `PlayerLosesLife` caller; production card wiring remains open. No JSON gap record is closed by this adapter. The next dependency-ready work is a separate E participant or P payment output after a fresh audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Any-Player Life-Gain Event

`PlayerGainsLife(*PlayerBinding).WithAmount(*NumberBinding)` matches each positive completed life gain by either player and saves the player and post-replacement amount in its pending trigger. It works in battlefield `WhenEvent` and `WhenEventModes` bodies. The amount option is optional. It reuses `EvtLifeGained`, the existing replacement path, and instance-local event captures. The existing `ControllerGainsLife` shares that implementation and keeps its controller filter. No new life executor, payment timing, target, or replacement rule is added. No production card or legacy export changed.

The E plan requires life-event participants. Synthetic tests first failed because `PlayerGainsLife` was undefined. Focused tests cover both players, post-replacement amount, zero gain, source phasing and departure, stack copies, game clones, inspection, registration, and a public DSL declaration. Existing controller-gain tests cover replacement consumption and controller changes. A fresh production search found six `ControllerGainsLife` callers and no `PlayerGainsLife` caller; all card wiring remains open. No JSON gap record is closed by this adapter. The next dependency-ready work is a separate E participant or P payment output after a fresh audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Controller Life-Gain Amount Capture

`ControllerGainsLife(*PlayerBinding).WithAmount(*NumberBinding)` saves the actual positive life-gain event amount with the gaining player in each pending trigger. It works in battlefield `WhenEvent` and `WhenEventModes` declarations. The adapter uses `EvtLifeGained`, its post-replacement amount, and the existing instance-local event capture frame; it adds no life-gain executor or result store. A zero or consumed gain has no event. Registration rejects nil, unnamed, repeated, duplicate, and out-of-scope amount bindings. The concrete return type of `ControllerGainsLife` enables the optional method while preserving existing `EventPattern` callers. No production card changed.

Cradle of Vitality in `data/JMP.json` needs the captured amount. Its optional payment, payment-dependent target timing, and production wiring remain separate, so its Jumpstart gap record stays open. Six production `ControllerGainsLife` callers remain and none uses the new amount option. Existing life-gain and trigger exports remain for unmigrated callers. Focused tests first failed because `WithAmount` was unavailable. Synthetic tests cover post-replacement amount, opponent exclusion, independent gains and stack copies, game clones, registration, inspection, and a public DSL declaration. Existing controller life-gain tests cover control changes, source departure and return, phasing, zero and consumed gains, delayed and modal bodies, and copy lifecycle. The next dependency-ready work is P optional-payment output and reflexive target timing, or another E event participant after a fresh audit.

Validation: focused engine and DSL tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Any-Player Draw Event

`PlayerDrawsCard(*PlayerBinding) EventPattern` matches each completed card draw by either player and saves the drawer in that trigger's instance-local frame. It works in battlefield `WhenEvent` and `WhenEventModes` declarations. It reuses `EvtCardDrawn`, the existing draw replacement path, and pending-trigger capture. A replaced draw or an empty-library attempt creates no matching event. Registration rejects nil, unnamed, unbound, and out-of-scope bindings. No payment, draw-result binding, or target declaration is added.

The E inventory requests a generalized player-draw pattern. The complete Phyrexian Tyranny text in `data/PLS.json` confirms the any-player event requirement, but its optional mana payment and production wiring remain separate. Synthetic tests first failed because the constructor was absent. They now cover both drawers, repeated and failed draws, replacement, phasing, event-time control, source leave and return, stack copying, game cloning, inspection, registration, and a public DSL declaration. A fresh production search found no `PlayerDrawsCard` caller; existing controller/opponent patterns and direct draw-event callers remain. No JSON card gap record closes from adapter availability alone. The next dependency-ready slice is another E event participant or P payment output after a fresh audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Any-Player Discard Event

`PlayerDiscardsCard(*PlayerBinding) EventPattern` matches each completed card discard by either player and saves the discarding player in the pending trigger. It works in battlefield `WhenEvent` and `WhenEventModes` declarations. The existing `OpponentDiscardsCard` keeps its event-time controller filter. Both use the shared `EvtDiscard` dispatch and instance-local event-player capture; no new discard or replacement service was needed. Registration rejects nil, unnamed, unbound, and out-of-scope bindings. There is no target declaration, cost capture, or discard-result output in this slice.

The E inventory requests a generalized player-discard pattern. Synthetic tests first failed because the constructor was absent. They now cover both players, replacement to library, failed discard, source phasing, source departure and return, stack copy, game clone, inspection, registration, and a public DSL declaration. The existing opponent-discard tests cover per-card triggering and event-time controller filtering. A fresh production search found no `OpponentDiscardsCard` or `PlayerDiscardsCard` caller and only test-only direct `EvtDiscard` watchers; no production card or legacy author export changed. No JSON gap record closes through this pattern alone. The next dependency-ready slice is a separate E event participant or P payment output after a fresh requirements audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Cast Spell Has X in Mana Cost

`SpellCast().WithXInManaCost()` filters a battlefield `WhenEvent` or
`WhenEventModes` trigger by the cast spell's mana cost. It reads the identified
spell on the stack when `EvtSpellCast` occurs, so an announced X of zero still
matches. It reuses the cast event and pending-trigger lifecycle. It creates no
new binding, damage action, or cast-payment rule. Registration rejects a
duplicate filter or a source outside the battlefield. Existing callers and
legacy exports remain unchanged.

Matterbending Mage's complete text was checked in `data/SOS.json` and the
Secrets of Strixhaven gap inventory. The filter supplies its missing cast
predicate. Its entry choice, source-specific unblockable duration, complete
production migration, and card audit remain open. The production definition
still uses `WheneverYouCastSpellTrigger` with a card filter, so the gap record
stays blocked. Synthetic tests cover X=0, multiple X symbols, ordinary mana
costs, phasing, source leave and return, queued-trigger clone isolation,
inspection, duplicate validation, and a public DSL declaration. The focused
test first failed because `WithXInManaCost` was absent. The next
dependency-ready slice is a separate P payment output or T announcement
contract after a fresh audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short
-count=1`, DSL registration and closed-role checks, `go test ./... -short
-count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`
(zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Announced-X Counter Payment Capture

`RemoveXCountersFromSourceCostInto(core.CounterType, *NumberBinding) Cost`
fixes the announced X during activation cost preparation and uses the existing
locked fixed-counter transaction to remove that many counters. A successful
commit saves the paid count in the activation's instance-local number frame.
Zero is valid for a present source; a returned, phased-out, or insufficient
source cannot pay. The constructor is available in `Activate` and
`ActivateModes`, but not spell additional costs or resolution-time optional
payments. Registration checks binding presence, names, duplicate producers,
scope, and battlefield-source context. The older
`RemoveXCountersFromSourceCost` and all production callers remain unchanged.

Chamber Sentry's complete text was checked in `data/JMP.json` and the Jumpstart
gap record. Its X-counter cost and damage clause motivate this capture; typed
entry replacement, graveyard return, and production wiring remain separate.
The focused test first failed because the constructor was absent. Synthetic
tests cover zero and positive X, interleaved activations, stack copies, game
clones, mutable-X changes after preparation, source return, phasing, failed
payment, and declaration validation. A public DSL declaration compiles. No
gap record closes through adapter availability alone. The next
dependency-ready work is a separate P payment output or T announcement
contract after a fresh audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL
registration and closed-role checks, `go test ./... -short -count=1`,
`go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and
`git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Singular Life-Loss Result

`LoseLifeForWithResult(PlayerRef, ResolutionValue, ...LifeLossResultOption)` accepts `LifeLostInto(*NumberBinding)` and saves one selected player's actual life loss in the current resolution frame. It reuses `playersLoseLife`, including event dispatch after the life change. Missing or illegal recipients and nonpositive amounts save zero. Inspection and registration check binding presence, names, duplicates, and dominance. The separate `LoseLifeFor` and `dsl.LoseLife` two-argument signatures remain compatible with production callers and first-class function uses. No production card or legacy export changed.

This is a P action-result adapter. Bubbling Cauldron's complete text in `doc/designs/composable-dsl-migration-jumpstart-gap.json` calls for the total across each opponent; existing `LoseLifeEach(..., LifeLostInto(...))` supplies that set result, so this singular adapter does not close its card gap or change its production wiring. The focused contract test first failed at compilation because the singular constructor did not accept a result. Synthetic tests cover positive, zero, and negative amounts, absent recipient, clone isolation, scope and duplicate validation, and public spell, activation, and trigger declarations. Target illegality and event ordering use the existing player-reference and life service contracts. The next dependency-ready work is another P paid-object output or E event participant after a fresh audit.

## Current Slice: Unqualified Attack-Group Event

`OneOrMoreCreaturesAttack().WithCount(*NumberBinding)` matches one completed positive attacker declaration by either player. `WithCount` is optional and saves the declared count in the pending trigger. It uses the existing `EvtAttackersDeclared` event and trigger capture path; no combat engine contract changed. The sealed pattern works in battlefield `WhenEvent` and `WhenEventModes` declarations. Registration rejects invalid count bindings and non-battlefield sources. The existing controller-filtered pattern and legacy callers remain unchanged. No production card changed.

Duelist's Heritage's full text was checked in `cards/jumpstart/JMP.json` and the Jumpstart gap record. The new event pattern supplies its unqualified attack trigger. Its targeted optional double-strike effect and production card wiring remain separate work, so the gap record stays open. The focused test first failed because the constructor was absent. Synthetic tests cover either attack controller, once-per-group behavior, count capture, empty declarations, entering attacking, phasing, departure and return, game clones, inspection, and registration. A fresh search found legacy `WheneverOneOrMoreCreaturesAttackTrigger` production callers in Jumpstart; those callers remain. The next dependency-ready work is another E event participant or P payment output after a fresh audit.

Validation: focused `TestAnyAttackBatch` and existing attack-batch tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Simultaneous Life-Gain Total

`GainLifeEach(PlayerSet, ResolutionValue, ...LifeGainResultOption)` now accepts `LifeGainedInto(*NumberBinding)`. It saves the sum of actual positive gains after modifiers and replacements in the current resolution frame. The shared batch service still prepares each gain before changing life totals and sends gain events after all totals change. Empty or absent sets and nonpositive amounts write zero. The option works in spell, activation, and trigger bodies; a singular player cannot satisfy the set role. Inspection and registration reject nil, unnamed, duplicate, and undominated bindings. Existing two-argument callers and legacy exports remain valid. No production card changed.

The P action-result inventory requires actual outcomes distinct from attempted amounts. The focused test first failed because `LifeGainedInto` was absent. Synthetic engine and DSL tests cover result consumption in one resolution, replacement outcomes, zero and absent sets, clone isolation, and registration. Existing batch tests cover simultaneous final-total event observation. A fresh search found no production `GainLifeEach` or `LifeGainedInto` caller; existing life-gain author exports remain for unmigrated cards. No JSON gap record is closed by this result alone; complete card wiring remains separate. The next dependency-ready work is another P payment or action output after a fresh audit.

Validation: focused `TestGainLifeEach` tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Unique-Highest-Life Player Reference

`PlayerWithUniqueHighestLife() PlayerRef` selects the sole highest-life player from the current game when a resolving instruction reads it. A tie or missing player is absent, so a singular player action does nothing. The sealed reference works in spell, activation, and trigger bodies wherever `PlayerRef` is accepted, including player damage recipients. It is not a state condition or a captured trigger participant. It reuses current player life totals and the existing singular player-action path; no life engine contract or result frame was needed. No production card or legacy export changed.

Ghazbán Ogre's full Oracle text was checked in `data/ARN.json`. `HasUniqueHighestLife()` already supplies its intervening-if condition; this slice supplies its resolution-time player selector. Indefinite control change and production card wiring remain open. The untracked Arabian inventory still calls this selector absent and needs updating during card migration. The focused test first failed because the constructor was absent, then passed for ties, instruction order, resolution-time selection, clone isolation, inspection, registration, and the public DSL role. A fresh production search found no caller of the new selector. No gap record closes through this adapter alone. Next dependency-ready work is a separate P payment output, after its payment transaction audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Controller Life Versus Starting Life

`ControllerLifeAtLeastStartingLife() StateCondition` reads the ability controller's current and starting life totals at each state check. It works in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, and `IfState`. A missing controller returns false. The sealed condition role, inspection, and existing state-condition validation apply; it creates no binding and changes no payment or life service. No production card or legacy export changed.

Path of Bravery's full text was checked in `cards/jumpstart/JMP.json`. The predicate supplies its static life threshold, including equality and nondefault starting totals. Its separate one-or-more-attacking-creatures event and production card wiring remain open, so the gap record stays blocked. Focused tests first failed for the missing constructor, then passed for either controller, missing controller, threshold changes, clone isolation, static reevaluation, registration, inspection, and the public DSL export. A fresh production search found no caller of the new predicate. Next dependency-ready work is a separate E attack-batch event or P payment output after a fresh contract audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff was reviewed.

## Current Slice: Departing Permanent Event Binding

`CreatureDies().WithPermanent(*PermanentBinding)` and the other creature/permanent death patterns save the departing battlefield incarnation in each pending trigger. `From(binding)` is absent for battlefield actions after death; `DamageSourceOf(From(binding))` can use that incarnation's departure LKI after the card returns. The binding is a sealed permanent reference, not a target or a graveyard card. It works in battlefield `WhenEvent` and `WhenEventModes` bodies. Registration rejects nil, unnamed, repeated, and unbound captures. The pattern reuses the existing pre-departure zone-change event, event capture effect, resolution frame, and damage-source LKI service; no new event or damage engine contract was needed. No production card or legacy export changed.

The E inventory requires an event permanent participant distinct from its destination card. Focused tests first failed because `WithPermanent` was absent. Synthetic tests cover the departed object's former controller and lifelink, return under another controller, simultaneous source and subject death, phased-out watcher suppression, stack-copy and game-clone isolation, inspection, registration, and a public DSL declaration. Existing death-trigger tests cover event filtering and replacement outcomes. A fresh search found no production `WithPermanent` caller and two `DiesCreatureTrigger` callers; other `NewTriggered(EvtZoneChange, ...)` callers need individual audits before migration. No JSON gap record closes through this adapter alone. The next dependency-ready work is a separate P payment output or E event participant after a fresh requirements audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. Review of the post-lint diff and `git diff --check` pass.

## Current Slice: Player Life-Total Resolution Value

`LifeTotalOf(PlayerRef) ResolutionValue` reads the referenced player's current life total when a resolving instruction uses it. It works in spell, activation, and trigger bodies through the existing player reference and value inspection contracts. An absent reference supplies zero; a declared player target uses the existing resolution-time legality check. Registration rejects nil, undeclared, and undominated references. The typed value does not supply a pure state comparison, life-total exchange, or target-announcement restriction. No production card or legacy author export changed.

The family S inventory names life-total values and comparisons. Ghazbán Ogre's complete text in `data/ARN.json` confirms that it still needs a pure comparison and indefinite control change; this resolution value does not close that gap. Synthetic tests initially failed because the constructor was absent, then covered instruction order, player selection, absence, target declaration and scope, game-clone isolation, inspection, and public DSL authoring. The existing life modifier and target services supply the behavior; no new life or replacement engine contract was needed. No gap record is closed. Next dependency-ready work is a separate S state comparison or P payment output, after its requirement audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Source Damage to Opponent This Turn

`SourceDamagedOpponentThisTurnCondition() StateCondition` uses a new turn-scoped player-damage record in the shared damage executor. Each positive final player-damage action records its source battlefield incarnation and actual player recipient. The condition checks the current permanent incarnation and the ability controller's opponent when evaluated. It works in permanent `ActivationIf`, `IfState`, `WhenState`, `WithInterveningIf`, and `While` contexts. An absent, phased-out, or returned source does not match. Final cleanup clears the record, and game clones own independent copies. Existing game-long and legacy card-ID queries remain available; no production card or legacy export changed.

Whirling Dervish's complete text in `data/LEG.json` requires an intervening-if check at each end step. Synthetic tests first failed because the condition was absent. They cover prevention, damage to the controller, simultaneous damage, source return, LKI cleanup, phasing, clone isolation, cleanup priority, final turn cleanup, trigger detection and resolution recheck, declaration validation, and a public DSL declaration. A fresh `cards/` search found one `DealsDamageToOpponentTrigger` production caller, Whirling Dervish. It still needs separate card wiring. The Legends gap record stays open. The next dependency-ready work is a distinct P payment output or E participant capture after a fresh contract audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Simultaneous Hand-Discard Total

`DiscardEach(PlayerSet, ...DiscardEachResultOption)` now accepts `DiscardTotalInto(*NumberBinding)`. The sealed option saves the count of completed card moves in the existing simultaneous hand-discard batch. Replacements run before moves; redirected discards count and prevented discards do not. Empty hands and absent sets write zero to the current resolution frame. Inspection and registration reject nil, unnamed, duplicate, and undominated result bindings. Existing `DiscardEach` calls and all legacy exports remain valid. The action works in spell, activation, and trigger bodies.

The complete Wheel of Fortune text in `data/LEA.json` remains an integration requirement; its production wiring is separate. The focused tests first failed because the option was absent. Synthetic tests cover completed totals, redirected and prevented discards, empty and absent selections, clone isolation, declaration validation, and the public DSL export. Existing batch tests cover simultaneous event observation and replacement preparation. A fresh search found no production `DiscardEach` or `DiscardTotalInto` caller; two `DiscardHand` calls and one `DiscardEntireHand` call remain. No production card or gap record changed. The next dependency-ready work is a distinct batch result or payment output after a fresh requirements audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Previous Slice: Simultaneous Player-Set Hand Discard

`DiscardEach(PlayerSet) ResolvingEffect` selects players and snapshots their hands at resolution. The batch uses the existing effect-discard replacement pipeline, moves accepted cards, then sends discard events after all moves. It is available in spell, activation, and trigger bodies. A singular player does not satisfy the sealed set role; registration checks nil and out-of-scope sets. The adapter does not produce a discard result and leaves `DiscardEntireHand` and all legacy exports intact.

Wheel of Fortune's complete text in `data/LEA.json` is “Each player discards their hand, then draws seven cards.” A synthetic declaration combines `DiscardEach(EachPlayer)` with `DrawEach(EachPlayer, Fixed(7))`. Focused tests first failed because the constructor was absent. They cover simultaneous discard-event observation, replacement redirection and prevention, absent sets, selected opponents, game-clone isolation, declaration validation, and the public DSL export. Wheel of Fortune remains production card wiring; Contract from Below also requires a separate ante action. A fresh caller search found three production uses of `DiscardHand` or `DiscardEntireHand` and no `DiscardEach` uses. The next dependency-ready B work is a player-set discard result or another audited batch action.

Validation: focused engine and DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Simultaneous Player-Set Life Gain

`GainLifeEach(PlayerSet, ResolutionValue) ResolvingEffect` uses the existing life modifier and replacement path for each selected player. It captures the set and amount once, prepares all gain outcomes before changing any life total, and sends positive gain events after all life totals change. The sealed player-set role excludes singular player references at compile time. Registration checks the set, amount, and binding scope. Synthetic tests cover final-total event observation, actual replacement outcomes, zero and absent selections, clone isolation, declaration errors, and the public DSL export. The focused test first failed because the constructor was absent. No production card or legacy export changed.

The plan's collection life-gain operation is available in spell, activation, and trigger bodies. Card wiring remains open; the next dependency-ready B slice is player-set discard after its choice and batch contract audit. This slice does not add a life-gain result binding or change singular `GainLife` callers.

Validation: focused engine and DSL tests, `go test ./pkg/mage/... -short -count=1`, registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass.

## Current Slice: Simultaneous Permanent-Set Counter Placement

`AddCountersEach(PermanentSet, core.CounterType, ResolutionValue) ResolvingEffect` reads one set and amount at resolution. `Game.AddCountersTogether` prepares each live, distinct permanent's counter action through the existing replacement pipeline before placing any counters. It uses game-local mutable permanents for clone isolation. Registration inspects the set and amount and rejects missing inputs, undeclared target groups, and invalid counter types. The sealed set role rejects a singular permanent at compile time. It works in spell, activation, and trigger bodies. It does not provide entry counters, a placement result, or card-specific restrictions.

Germination Practicum's complete text in `data/SOS.json` requires two +1/+1 counters on each creature its controller controls; its Paradigm clause remains separate. Synthetic tests cover counter doubling, replacement observation before placement, phasing, stale and duplicate incarnations, target-group identity, clone isolation, inspection, registration, and the public DSL export. The focused test first failed because `AddCountersEach` was absent. Production Germination Practicum and other collection-counter cards remain unwired; no production card or legacy export changed. The Secrets of Strixhaven gap record stays open. The next dependency-ready B slice is collection life gain or player-set discard after a fresh shared-contract audit.

Validation: focused engine and DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Simultaneous Permanent-Set Untap

`UntapEach(PermanentSet) ResolvingEffect` uses the shared singular untap rule for stun counters and a new batch service that completes successful untaps before sending events. It selects one set at resolution, removes duplicates and stale or phased-out incarnations, and ignores already untapped permanents. It works in spell, activation, and trigger bodies; inspection and registration reject a nil or undeclared set, while the sealed role rejects a singular permanent at compile time. No production card or legacy export changed.

Reset's full text in `data/LEG.json` requires untapping all lands its controller controls; Molten Note's full text in `data/SOS.json` requires untapping all creatures its controller controls. Synthetic tests cover simultaneous event observation, stun replacement, absent and phased members, stale and duplicate identities, clone isolation, target-group incarnation, inspection, registration, and the public DSL export. A fresh caller search found Reset still using `ForEach(..., Untap(...))` and Molten Note still using its existing resolution code. Reset also needs a casting-time restriction; Molten Note needs its other spell clauses. Both remain production card wiring. Existing `Untap` and other untap callers remain available; no legacy export was removed. The broad Legends and Secrets of Strixhaven gap records remain open. The next dependency-ready B slice is `AddCountersEach`, after a fresh shared replacement and batch audit.

Validation: focused tests and closed-role checks, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Simultaneous Permanent-Set Tap

`TapEach(PermanentSet) ResolvingEffect` is a typed collection tap for spell, activation, and trigger resolution. It reads one permanent set at resolution, keeps only current untapped battlefield incarnations, and uses `Game.tapPermanentsTogether` so all members become tapped before any tap event fires. Each actual transition emits one tap event. Registration inspects the set and rejects nil or undeclared references; a singular `PermanentRef` cannot be passed to the collection constructor. The effect reports mass tap metadata. It does not supply untapping, target announcement, or a tap-count result.

Riptide's complete text was checked in `data/DRK.json`: "Tap all blue creatures." A synthetic declaration uses `TapEach(MatchingPermanents(HasColorFilter(Blue)))`. Focused tests first failed because the constructor was missing, then covered simultaneous event observation, already tapped and phased-out members, stale and duplicate incarnations, clone isolation, public DSL registration, inspection, and a compile-time singular/set role error. Production Riptide remains a stub, and no production card uses `TapEach`; the untracked Dark inventory still needs a card-wiring update in the separate migration. No legacy export was removed. The next dependency-ready B slice is `UntapEach` or `AddCountersEach`, after checking the corresponding shared batch contract and card requirements.

Validation: focused `TestTapEach` and closed-role tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Cast-Color Entry Counter Replacement

`ColorsSpentToCastEnteringPermanent() EntryCounterValue` and `EntersWithCounters(core.CounterType, EntryCounterValue) Ability` supply a sealed, inspectable self-entry counter replacement. The entry path passes the entering permanent spell's own distinct colored mana count to preparation; direct entry passes zero. The adapter uses the existing counter replacement service before the entry event. It keeps `EntersWithComputedCounters` and other production callers unchanged. Registration rejects a nil amount and invalid counter types.

Rancorous Archaic's complete text in `data/SOS.json` requires one +1/+1 counter per color spent to cast it. Synthetic tests cover failed payment, paid two-color entry, direct return, clone isolation, counter doubling, zero-base additional counters, inspection, registration, and the public DSL export. The initial focused test failed because both constructors were absent. Production Rancorous Archaic still uses its legacy callback, and Wildgrowth Archaic has additional trigger and hybrid-mana requirements. Their gap records remain blocked. Next dependency-ready work is a separate entry replacement participant or a P/T extension after a fresh audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. No production card or JSON gap record changed.

## Current Slice: Self-Entry Former Cast Colors Capture

`Enters(This).WithColorsSpent(*NumberBinding)` saves the distinct colored mana colors paid to cast that permanent as a spell. The resolving permanent-spell path puts the count from its own cast context on the entry event; direct entry carries zero, even if another spell causes it. The existing event-capture frame saves the number for a battlefield `WhenEvent` or `WhenEventModes` body. Registration rejects nil, unnamed, repeated, non-entry, and out-of-scope bindings. No production card or legacy export changed.

Sundering Archaic and Transcendent Archaic need this former-cast value for their entry triggers according to their complete text in `data/SOS.json`. A fresh search found both still using `ResolvingCastContext` callbacks in `cards/secretsofstrixhaven/creatures.go`; neither was migrated, and their other clauses remain open. Their gap records stay blocked. Rancorous, Magmablood, and Wildgrowth Archaic instead need entry replacement work for their "enters with" clauses. Synthetic tests cover real paid mana, failed payment rollback, uncast entry, departure and return, phasing, trigger copies, game clones, inspection, and public DSL registration. The first focused test failed because the capture method was absent. The next dependency-ready slice is a typed former-cast colors value for an entry replacement, after a separate replacement contract audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Resolving Spell Distinct-Colors Value

`ColorsSpentToCastThisSpell() ResolutionValue` reads the number of distinct colored mana colors paid to cast the current spell. It is valid in a typed `Spell` body, including a copied spell. It uses the existing cast payment record, `CastContext.DistinctColorsSpent`, resolution state, and stack copy/clone services. Registration rejects activation and trigger use. An absent cast context returns zero in an internal read; a failed payment does not put the spell on the stack. No production card or legacy export changed.

Arcane Omens and Archaic's Agony require this amount according to their complete text in `data/SOS.json`. This slice supplies the amount only; Archaic's Agony still needs excess-damage results and linked play permission. Creature entry Converge abilities need a separate former-spell cast-payment capture. Production wiring and the Secrets of Strixhaven gap records remain open. A fresh caller search found existing legacy `ResolvingCastContext` uses in that set and no production use of this typed value. Synthetic tests cover actual and failed mana payment, colorless and zero colors, spell-copy and game-clone isolation, absent context, inspection, public DSL registration, and rejection outside spell bodies. The first focused run failed because the constructor was absent. The next dependency-ready work is a typed former-cast payment capture for creature entry or another independently audited E event participant.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The diff was reviewed after lint. The final diff check is part of commit review.

## Current Slice: Spell-Cast Distinct-Colors Event Capture

`SpellCast().WithColorsSpent(*NumberBinding)` saves the number of distinct colored mana colors spent to cast the exact event spell. It works in battlefield `WhenEvent` and `WhenEventModes` abilities with existing cast filters. `EvtSpellCast.CastColorsSpent` takes the count from that spell's cast context before dispatch; the pending trigger saves it in its instance-local number frame. The shared `DistinctColorsSpent` helper now excludes colorless mana. No production card or legacy export changed.

The complete Magmablood Archaic text was checked in `data/SOS.json`. Its instant-or-sorcery cast bonus requires this event value, but also needs its own entry Converge behavior and production wiring; its gap remains open. Arcane Omens and Archaic's Agony need the distinct-color value for their own spell cast, which this battlefield event capture does not supply. Their gaps remain open. The first focused test failed because `WithColorsSpent` was absent. Synthetic tests cover two colors plus colorless mana, five colors, zero payment, spell and source departure, source phasing, clone isolation, inspection, declaration rejection, and a compile-time role error. The next dependency-ready work is a separate self-cast Converge value or another E event participant after a fresh requirements audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Spell-Cast Mana-Spent Event Capture

`SpellCast().WithManaSpent(*NumberBinding)` saves the total mana paid for the exact cast stack object at event time. The sealed battlefield event pattern works in `WhenEvent` and `WhenEventModes`; the trigger body reads the instance-local result with `FromNumber`. The cast payment record, event dispatch, and existing number frame supply the behavior. A cast without mana payment saves zero. No production card or legacy export changed.

Aberrant Manawurm and Ambitious Augmenter's complete text in `data/SOS.json` and the Secrets of Strixhaven inventory require the triggering spell's mana-spent total even after that spell leaves the stack. This slice supplies that capture, but not power/toughness comparison, token copy, or card wiring. Their gap records remain open. A fresh `cards/` search found 14 occurrences of `WheneverYouCastSpellWithManaSpent`, `OpusEffect`, and `Increment`; those legacy helpers remain. The focused test first failed because `WithManaSpent` was absent. Synthetic tests cover mixed-color and zero payment, spell departure, trigger clone isolation, source phasing, inspection, and invalid declarations. The next dependency-ready slice is an E cast-event comparison or distinct P result adapter after a fresh requirements audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Cast-Condition Union

`AnyCast(...CastCondition) CastCondition` combines saved cast facts under the existing `IfCast` spell-body branch. It evaluates to true once when any child is true, including when several children are true. It copies the child list and uses the sealed cast-condition role and existing cast snapshot. Registration rejects empty, nil, and invalid child conditions, as well as use outside a spell. No production card or legacy export changed.

Draconic Roar's full text was checked in `data/JMP.json` and `cards/jumpstart/JMP.json`. A fresh API audit found that `SnapshotPlayer(ControllerOf(target), binding)` already saves the target creature's controller before the first damage instruction. The public DSL test registers that sequence with `AnyCast`; the engine test covers neither, either, and both cast facts, plus stack-copy and game-clone isolation. The first focused test failed because `AnyCast` was undefined. Production Draconic Roar still uses its legacy callback in `cards/jumpstart/spells.go`; its Jumpstart gap and card wiring remain open. The next dependency-ready work is a separate P, T, or E slice after a fresh caller and contract audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues after renaming a shadowed test variable), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Revealed Card Subtype at Cast

`RevealedSubtypeAtCast(string) CastCondition` exposes a card subtype revealed while paying a spell's additional cost. It reads the existing cast snapshot through `IfCast`, and registration rejects empty subtypes and use outside a spell. The cast pipeline now copies the revealed card when it records the snapshot; stack copies and game clones also copy it. A later change to the original card or one copy cannot change another cast fact. The reusable boundary is the existing payment transaction, cast context, and sealed cast-condition role. No production card or legacy export changed.

Draconic Roar's full text was rechecked in `data/JMP.json`. Its optional Dragon reveal and controlled-Dragon-at-cast conditions now have typed predicates, but the bonus damage still needs a target creature controller captured at the correct time and production wiring. Its Jumpstart gap stays open. Synthetic tests cover paid and declined reveal, no matching card, snapshot isolation, stack copies, game clones, inspection, registration validation, and the public DSL export. The first focused test failed because the constructor was absent. The next dependency-ready slice is target creature controller capture for the bonus damage, after a fresh contract audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Controlled Creature Subtype at Cast

`ControlledCreatureSubtypeAtCast(string) CastCondition` and `IfCast(CastCondition, ResolvingEffect)` expose the existing cast snapshot to typed spell bodies. The snapshot now distinguishes controlled creatures from other permanents with a subtype and is copied for stack copies and game clones. Registration rejects empty or known noncreature subtypes and nonspell use. Synthetic tests cover cast-time presence and absence, later departure and arrival, phasing, inspection, copies, clone isolation, and declaration validation. Draconic Roar's complete text was checked in `data/JMP.json`; its optional reveal and revealed-at-cast condition are now available, while target-controller capture, the bonus damage sequence, and production wiring remain open. Its Jumpstart gap stays blocked. No production card or legacy export changed. The next dependency-ready work is target creature controller capture, after a fresh contract audit.

The first focused test failed because the constructors were absent. Focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Enchantment Spell-Cast Subtype Filter

`SpellCast().OfEnchantmentSubtype(string)` filters a battlefield `WhenEvent` or `WhenEventModes` trigger by the cast spell's enchantment subtype. It reads the exact stack object named by `EvtSpellCast.TargetID`; an unrelated copy with the same card ID does not match. It composes with the existing type, color, opponent, and caster filters. Registration rejects empty, unknown, nonenchantment, repeated, and type-incompatible declarations. The sealed event-pattern role and existing pending-trigger lifecycle are unchanged. No production card or legacy export changed.

Kor Spiritdancer's complete text was checked in `data/JMP.json` and the Jumpstart inventory. This slice supplies its Aura-cast event filter; its optional draw, static attached-Aura bonus, and production wiring remain open, so the gap record stays blocked. Synthetic engine tests cover exact cast-object identity, non-Aura enchantments, source phasing and absence, game-clone isolation, inspection, and registration rejection. A public DSL test registers a composed declaration. The first focused engine test failed because `OfEnchantmentSubtype` was absent. The next dependency-ready work is a separate cast-time controlled-creature snapshot for Draconic Roar or another independently audited E event participant.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. Kor Spiritdancer still uses the production `WheneverSpellCastTrigger`; the legacy export remains for it and other callers.

## Current Slice: Optional Revealed Hand-Card Payment

`OptionalRevealFromHandCost(CardFilter, text, prompt string, *NumberBinding, *CardBinding) Cost` combines an optional payment outcome with the existing chosen hand-card reveal capture. It works as a spell additional cost or activated ability cost. The choice occurs during payment preparation. A paid cost records 1 and the selected hand visit in that spell or ability frame; decline or no matching card records 0 and no card. The shared payment transaction validates the locked card again before live payment. A stale hand visit or failed total cost publishes no reveal or result. Stack copies and game clones keep separate values. Registration rejects nil or unnamed bindings, missing text, duplicate producers, and resolution-time use. `OptionalCostInto` still rejects an arbitrary captured inner cost.

Draconic Roar's complete text was rechecked in `data/JMP.json` and its Jumpstart gap record. This slice supplies the optional revealed-Dragon payment and its cast-time outcome. It does not supply the separate controlled-a-Dragon-at-cast condition or target creature controller binding. Production Draconic Roar and all other card wiring remain unchanged, so no gap record was closed. The first focused test failed because the constructor was absent. Synthetic tests cover acceptance, decline, no matching card, public reveal, stale hand visit, failed total payment, pending activations, stack copy, clone isolation, spell additional cost, public DSL registration, and invalid declarations. The next dependency-ready work is the independent cast-time controlled-creature snapshot or an E event-participant slice after a fresh requirements audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Optional Payment Outcome Binding

`OptionalCostInto(inner Cost, prompt string, paid *NumberBinding) Cost` saves 1 when the payer accepts and pays a direct optional cost and 0 when the payer declines or cannot pay. The existing cost-choice lock and total-cost transaction supply the timing; the new outcome marker writes into the spell or activation payment frame, so pending instances, stack copies, and game clones keep separate values. It does not use the legacy source-wide optional-cost tracker. Registration rejects nil or unnamed bindings, empty prompts, nested choices, other captured costs, duplicate producers, and resolution-time payment use. No production card or legacy export changed.

This prerequisite supports a future optional reveal-card branch, but it does not yet bind the revealed card, a chosen alternative cost, or cast-time controlled-creature state. Draconic Roar's complete text was checked in `data/JMP.json`; its production wiring and Jumpstart inventory stay open. Synthetic engine and public DSL tests cover acceptance, decline, inability to pay, clone and stack-copy isolation, and declaration validation. The next dependency-ready slice is optional or alternative payment of a chosen hand card with branch-local capture, or an independent E event participant.

Validation: focused tests first failed because the constructor was absent. The focused tests, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Revealed Hand-Card Payment Binding

`RevealFromHandCostWithCard(CardFilter, string, *CardBinding) Cost` saves one card chosen to pay a direct reveal-from-hand cost. The payer chooses once during total-cost preparation; isolated validation and live commit check the same card and hand visit. A successful payment stores the visit in the spell or activation frame and adds one public reveal snapshot; failed validation adds none. `FromCard` is absent after the card leaves and returns to hand. Stack copies and game clones keep their own frames. Registration rejects nil or unnamed bindings, empty cost text, duplicate producers, and reads without a producer. The first focused test failed because the constructor did not exist. Synthetic engine and public DSL tests cover pending activations, stack copies, clones, stale visits, failed total payment, public reveal records, and declaration validation. No production card was changed.

This adapter uses the existing hand-visit, payment, and `CardBinding` services. It supports direct spell additional costs and activated ability costs. It does not expose a result from `EitherCost`, optional reveal costs, or resolution-time payment. The complete text of Wren's Run Vanquisher and Draconic Roar was checked in `data/JMP.json`; both need a separate choice or optional-cost contract before this binding can represent their full costs. Their production `RevealFromHandCost` calls and all other legacy callers remain. No gap record was closed. The next dependency-ready work is a P optional or alternative payment choice with branch-local result capture, or an independent E event-participant slice after a fresh requirements audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Optional Singular Graveyard Card Target

`GraveyardUpToOne() GraveyardTargetOption` changes a `TargetGraveyardCard` declaration from exactly one to zero or one target. It uses the shared target announcement, copy, legality, and graveyard-visit services. It supports typed spells, activations, and triggers, including a trigger with no legal graveyard card. A controller may choose zero even when a card is legal. A chosen card is rechecked at resolution and cannot leave and return to keep its target identity. Registration rejects a repeated option. The declaration remains a singular `CardRef`; no card set or new choice service was added.

Moseo, Vein's New Dean's complete text in `data/SOS.json` and its Secrets of Strixhaven inventory and JSON gap record require “up to one target creature card” with a mana-value limit based on life gained this turn. This slice combines with the existing limit and history adapters to supply that target requirement. Its production card has not been migrated, and its other abilities need a separate complete audit, so the gap record stays open. The first focused test failed because `GraveyardUpToOne` was undefined. Synthetic tests cover zero legal cards, voluntary zero choice, stack copies, clone isolation, selected-target legality after leave and return, composition with a mana-value limit, public DSL registration, and duplicate-option rejection. Existing production callers and legacy exports remain unchanged. The next dependency-ready work is a distinct P payment/result or E event-participant slice after a fresh requirements audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Dynamic Graveyard Card Mana-Value Target Limit

`GraveyardCardManaValueAtMost(TargetNumber) GraveyardTargetOption` limits a typed graveyard-card target by the card's current mana value. The sealed `TargetNumber` role has `ControllerLifeGainedThisTurn()` and `FixedTargetNumber(int)` constructors. The life-gain form reads actual turn history for the ability controller at target announcement and resolution. The existing target service performs both checks, and the existing graveyard-visit binding keeps leave-and-return illegal. This slice adds no target-count option, new life-gain executor, production card declaration, or legacy export change.

Moseo, Vein's New Dean's complete text in `data/SOS.json` and its Secrets of Strixhaven inventory and JSON gap record require a creature card in its controller's graveyard with mana value no greater than life gained this turn. This adapter supplies that eligibility restriction. Moseo still needs "up to one" target cardinality and card wiring, so its gap stays open. The first focused test failed because the target-number role and graveyard option did not exist. Synthetic tests cover zero and positive gain, controller identity, own and any graveyard, fixed and dynamic limits, clone isolation, trigger target announcement, resolution recheck, graveyard leave-and-return, and registration rejection of nil, negative, or repeated limits. The public DSL test registers a synthetic trigger declaration. Existing graveyard target callers and legacy author exports remain unchanged. The next dependency-ready T slice is typed optional singular graveyard-card targeting with correct trigger and copy behavior; a separate P payment/result slice also remains open.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Resolution-Time Life-Gain History Amount

`LifeGainedThisTurn(PlayerRef) ResolutionValue` reads the existing turn-scoped total of actual positive life gain for a selected player when a resolving spell, activation, or trigger uses it. It uses the same event tracker as `ControllerGainedLifeThisTurn()` and the normal `PlayerRef` scope rules. Replacement can change or consume a gain before the tracker records it. Zero gain and life loss do not add to the total. An absent player supplies zero; source departure does not erase history. Stack copies read their own controller and game clones keep separate history. Registration rejects nil, undeclared, and undominated player references. The value is not a state condition or an announcement-time target bound. No production card or legacy export changed.

Moseo, Vein's New Dean's full text in `data/SOS.json` and the Secrets of Strixhaven inventory and JSON gap record require the amount of life gained this turn. This slice supplies that numeric history during resolution. Moseo also needs the amount to restrict a graveyard-card target when the trigger is put on the stack, plus optional target and return wiring, so its gap remains open. Other production callers still use `LifeGainedThisTurnFor` or direct tracker reads. Focused tests first failed for the missing constructor. They cover replacement, consumed and zero gain, accumulation, player and absent-source identity, target-reference validation, stack copies, clone isolation, turn reset, inspection, public DSL registration, and the closed resolution-value role. The next dependency-ready work is an announcement-time numeric history restriction for target eligibility, or a separate P payment/result slice after a fresh audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Controller Life-Gain History Condition

`ControllerGainedLifeThisTurn() StateCondition` exposes the existing positive, actual life-gain turn tracker to pure typed declarations. It is valid in `ActivationIf`, `IfState`, `WhenState`, `WithInterveningIf`, and `While`. It reads the condition's controller; a missing controller fails, and source departure does not erase history. Replacement that consumes the gain and zero gain do not satisfy it. Game clones keep separate tracker state, and final cleanup clears it. This is an adapter over `TrackerReader`, not a new life-gain executor or numeric value. No production card or legacy export changed.

Secrets of Strixhaven's Infusion inventory and `data/SOS.json` provide the acceptance requirements; Efflorescence and Moseo, Vein's New Dean show the boolean condition, while Moseo also needs the numeric gain amount and card choice/return wiring. CR 119.3 and 119.9 distinguish actual positive gain from zero gain. Focused tests first failed for the missing constructor. They cover consumed and zero gain, controller and absent-source identity, clone isolation, turn reset, resolution and static contexts, public DSL registration, and inspection. Production Infusion callers remain on their current helpers and gap records remain open. The numeric resolution value is now supplied by the slice above; announcement-time eligibility and card wiring remain open.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Creature Death After Damage by Source This Turn

`CreatureDies().DamagedByThisTurn()` is a typed battlefield event pattern for a creature that received positive damage from this source's battlefield incarnation this turn and then dies. It works in `WhenEvent` and `WhenEventModes`; the trigger body is a resolving effect. The filter reads the existing turn-scoped `DamageSystem` history and the zone-change event's new departing `SourceIncarnation` field. `FireEvent` fills that field from departure LKI before a simultaneous leave event is queued. This field is absent for unrelated events. The pattern and damage record are immutable across trigger copies and game clones. Registration rejects the filter on a noncreature death pattern and rejects repeated use. This slice adds no result binding, card-specific rule, or production card wiring.

Sengir Vampire's complete Oracle text in `data/LEA.json` and Axelrod Gunnarson's in `data/LEG.json` were rechecked against their inventories and CR 400.7, 603.6c, and 603.10a. The tests first failed because the typed filter did not exist. Focused tests now cover dealt and prevented damage, no damage, source and victim leave-and-return, phasing, simultaneous source and victim death, absence of event incarnation, clone isolation, public DSL authoring, and invalid declarations. `go test ./pkg/mage/... -short -count=1` passes. Production Sengir Vampire and Axelrod Gunnarson remain unmigrated; Axelrod also needs a complete typed damage recipient audit. Their gap records stay open. Legacy `EventSourceDamagedBySource` and card-ID damage queries remain for existing callers. The next dependency-ready slice is a distinct event capture or payment/result extension after a fresh requirement audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint.

## Current Slice: Turn-Scoped Damage Incarnation History

The damage system now records positive damage to a permanent this turn by both the captured source incarnation and recipient battlefield incarnation. It writes after replacement and prevention in the shared damage action, so prevented damage, absent recipients, and phased-out recipients add no record. Source departure and return, recipient departure and return, LKI cleanup, and phasing preserve the correct identities. Game clones copy the record independently. Final cleanup clears it after all cleanup priority rounds. The existing card-ID turn query and all production callers remain unchanged.

Sengir Vampire's complete Oracle text was checked in `data/LEA.json`, and Axelrod Gunnarson's in `data/LEG.json`, against their inventories and CR 400.7, 603.6c, and 603.10a. The card-ID turn query cannot distinguish a returned source or victim, so this engine prerequisite comes before a typed “creature dealt damage by this source this turn dies” event filter. This slice does not expose that filter or migrate either card. The internal `DamageSystem.dealtDamageToPermanentThisTurn` query takes a captured source and recipient incarnation. Focused tests initially failed because it was absent, then covered both incarnations, departure LKI cleanup, prevention, phasing, cloning, and final cleanup. The next dependency-ready slice is the typed death-event filter using this record and event-time recipient identity; it needs an event-time identity audit before implementation.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The first engine run found one excess clone allocation; conditional allocation of the new history map corrected it.

## Current Slice: Simultaneous Destruction Graveyard Result

`DestroyEach(PermanentSet, ...DestroyEachResultOption)` and `DestroyEachWithoutRegeneration` now accept `PermanentsPutIntoGraveyardThisWayInto(*NumberBinding)`. The option saves the number of selected battlefield incarnations that the shared destruction batch moves to a graveyard. It uses the batch's existing `destroyed` result and the current resolution frame; no separate executor or global history slot was added. It writes zero for an empty set. Regenerated, indestructible, phased-out, absent, returned, and replacement-exiled objects do not count. Tokens that enter a graveyard count. Existing calls and all legacy exports remain valid; inspection and registration reject nil, unnamed, duplicate, and undominated bindings.

Volcanic Eruption's complete Oracle text was checked in `data/4ED.json` and the Fourth Edition inventory against CR 400.7, 609.3, and 701.8. The old inventory says X-dependent target groups and typed mass damage are absent; a fresh API search found `TargetPermanentsExactlyX` and `DealDamageEach` already implemented. A synthetic spell combines those operations with this result. Production Volcanic Eruption remains unmigrated, so its JSON gap stays open; no card or gap record changed. The first focused test failed because the constructor was absent. Tests cover simultaneous land destruction, regeneration, indestructibility, phasing, empty-set reset, replacement to exile, target filter and X cardinality, clone isolation, a public declaration, and registration validation. A fresh caller search found no production use of the new option. The next dependency-ready work is a separate P action result or T target extension after a fresh requirements audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Singular Destruction Success Result

`DestroyWithResult(PermanentRef, DestroyResultOption)` and `DestroyWithoutRegenerationWithResult(PermanentRef, DestroyResultOption)` accept `DestroyedThisWayInto(*NumberBinding)`. The sealed option saves one only when that instruction destroys its exact battlefield incarnation, and zero when the subject is absent, phased out, indestructible, regenerated, or replaced. It uses the shared destruction batch and instance-local number frame. The existing `Destroy` and `DestroyWithoutRegeneration` signatures remain unchanged, including their function-value types; no production card or legacy export changed. Inspection and registration reject missing, unnamed, duplicate, and undominated bindings. Spell, activation, and trigger resolving bodies are supported; static and cost roles are excluded by the sealed effect type.

Cosmic Horror's full text was checked in `data/LEG.json` and its Legends gap record against CR 400.7, 609.3, and 701.8. The synthetic declaration uses the saved success value to condition damage from the source after it leaves. Production Cosmic Horror remains unwired, so its gap stays blocked. The initial focused test failed because the constructor was absent. Tests cover successful destruction, regeneration, indestructibility, phasing, absence, leave and return, no-regeneration destruction, clone isolation, target legality, damage from the departed source, and declaration errors. A fresh production search found no caller of the new functions; 62 production references to the existing singular destruction constructors remain compatible. The next dependency-ready P slice is another distinct action outcome, after checking its complete card requirement.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Simultaneous Destruction Creature-Death Result

`DestroyEach(PermanentSet, ...DestroyEachResultOption)` and `DestroyEachWithoutRegeneration(PermanentSet, ...DestroyEachResultOption)` now accept `CreaturesDiedThisWayInto(*NumberBinding)`. The sealed option saves the count of selected creatures moved to a graveyard by that one simultaneous destruction instruction in the resolving instance's number frame. The shared destruction batch supplies the result and retains pre-action creature status; no separate destroy executor or global death counter is used. An empty set writes zero. Indestructible, regenerated, absent, phased-out, and noncreature objects do not count. Existing one-argument calls and all legacy author exports remain valid. Inspection and registration reject nil, unnamed, duplicate, and undominated result bindings.

Hellfire's complete Oracle text was checked in `data/LEG.json` and its Legends gap record against CR 701.8, 400.7, and the batch contract. This slice supplies its "number of creatures that died this way" result, but the production card retains legacy wiring and still needs the full typed damage expression and a complete card migration audit. Volcanic Eruption requires a distinct count of targeted Mountains put into graveyards, so this creature-only result does not resolve its gap. No card or JSON gap record changed. The first contract test failed because the option and extended signature were absent. Synthetic tests cover two simultaneous creature deaths, an artifact, regeneration, indestructibility, phasing, a stale result cleared by an empty group, no-regeneration destruction, clone isolation, a cast spell that uses the result as damage, public spell/activation/trigger declarations, and registration errors. A fresh production search found no `CreaturesDiedThisWayInto` caller; existing `DestroyEach` callers remain compatible. The next dependency-ready P slice is a singular destruction-success result for effects such as Cosmic Horror, with a separate audit of its exact "if destroyed this way" condition.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Resolution-Time Player Hand Size

`HandSizeOf(PlayerRef) ResolutionValue` reads the selected player's current number of cards in hand in a spell, activation, or trigger resolving body. It uses the existing sealed player reference, target frame, and player hand service. An absent player supplies zero. The declaration is inspectable; registration rejects nil, undeclared, and undominated references. The value makes no card choice and reveals no hidden information. No production card or legacy export changed.

Borrowed Knowledge's complete text in `data/SOS.json`, its Secrets of Strixhaven inventory and gap record, and CR 400.1, 400.2, and 609.3 supply the acceptance case. A synthetic first mode discards the controller's hand, then draws based on the target opponent's hand at resolution. The second mode can use the existing `DiscardAmountInto` result. Production wiring remains open, so the JSON gap stays blocked. Tests first failed because `HandSizeOf` was undefined. Focused tests cover controller and active-player identity, target selection, changes before evaluation, absent references, clone isolation, inspection, registration, and the public DSL declaration. The next dependency-ready work is a separate P action result or T target extension after a fresh requirements audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `HandSizeOf` caller and two production `DiscardHand` callers; the legacy export remains.

## Current Slice: Entire-Hand Discard Result

`DiscardEntireHand(PlayerRef, ...DiscardResultOption)` now accepts `DiscardAmountInto(*NumberBinding)`. It saves the number of completed discards from one hand-discard instruction in the current resolution frame. The shared hand-discard executor takes the player's hand snapshot and calls the existing effect-discard action for each card. Prevention or an impossible discard does not count; a replacement that sends the card to the library still counts. An absent player or empty hand writes zero. The existing one-argument signature and legacy `DiscardHand` callers remain valid. Registration rejects nil, unnamed, repeated, and undominated result bindings. The sealed option is not a resolving effect.

Borrowed Knowledge's complete text was checked in `data/SOS.json`, the Secrets of Strixhaven inventory and JSON gap record, and CR 609.3 and 701.9. The synthetic second mode discards the controller's hand, then draws the number of cards actually discarded. Its other mode needs a target opponent's hand-size value, and production card wiring remains open; the JSON gap stays blocked. Focused tests first failed because the result option was absent. They now cover completed, replaced, prevented, empty, and absent discards, clone frame isolation, stack copies with different hand counts, declaration validation, and a public DSL declaration. A fresh search found no production `DiscardAmountInto` caller; `DiscardEntireHand` has one production caller and legacy `DiscardHand` has two. The next dependency-ready work is a separate P action result or T target extension after a fresh audit.

Validation: focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The first lint run found one test variable name issue; it was corrected before the passing run.

## Current Slice: Simultaneous Life-Loss Result

`LoseLifeEach(PlayerSet, ResolutionValue, ...LifeLossResultOption)` now accepts `LifeLostInto(*NumberBinding)`. The option saves the total life lost by the selected players from that one instruction in its resolution frame. The shared life service measures each total change before it dispatches `EvtLifeLost`; all selected life totals still change before those events. An empty set or nonpositive amount writes zero. Registration rejects nil, unnamed, repeated, and undominated bindings; the sealed option cannot be used as an effect. The existing two-argument call and production callers remain valid.

Bubbling Cauldron's complete second ability was checked in `data/JMP.json`, its Jumpstart inventory and gap record, and CR 119.3 and 119.6. A synthetic declaration combines the set instruction and `GainLife(You, FromNumber(lost))`. Focused tests first failed because `LifeLostInto` was absent. Engine and public DSL tests cover positive, zero, negative, and below-zero life loss, an empty set, two simultaneous recipients, clone frame isolation, declaration validation, and the compile-time role. This adapter supplies the recorded life-loss result requirement. Bubbling Cauldron still uses a legacy callback and cost for its second ability; production wiring is separate, so its gap record stays open. A fresh search found no production `LifeLostInto` caller. Other legacy life and damage callers remain.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The next dependency-ready work is a separate P action-result or T target-group extension after its own requirements audit.

## Current Slice: Optional Mill Payment

`MillCost(count int) Cost` is a reusable payment component for activated abilities, spell additional costs, and resolution-time `IfPaid`/`UnlessPaid`. It reuses the total-cost transaction, mill modifiers, and shared library-to-graveyard movement. Preparation locks the replacement-adjusted amount; commit rechecks that amount and the available library before mana or life changes. A payment fails when the library cannot supply the full modified amount. A replacement that reduces the amount to zero permits payment. Registration rejects nonpositive counts. No production card or legacy export changed.

Deep Spawn's complete text was checked in `data/FEM.json`, its Fallen Empires inventory and gap record, and CR 118.3 and 701.17b. The synthetic upkeep declaration uses `UnlessPaid(You, Costs(MillCost(2)), Sacrifice(This))`. Its other ability and production wiring remain separate, so the JSON gap stays open. Focused tests first failed because `MillCost` was absent. Engine and public DSL tests cover zero through three library cards, optional payment, replacement increase and reduction, changed replacement, preparation and commit rollback, clone isolation, and registration. The next dependency-ready work is another isolated P payment/result or a T target extension after a fresh requirements audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A first lint run found three style issues in the new helper; they were corrected before the passing run.

## Current Slice: Source-Inclusive Sacrifice Mana Value

`SacrificeMatchingIncludingSourceCostWithManaValue(filter, text, binding) Cost` lets an activated ability or spell additional cost sacrifice a matching permanent, including its source, and save that permanent's mana value in an instance-local number binding. It reuses the locked sacrifice choice and payment transaction already used by `SacrificeMatchingCostWithManaValue`; no new payment executor is added. The source is a candidate only when it satisfies the filter. An absent, phased-out, returned, or otherwise ineligible choice fails payment without a result. Registration rejects missing text or binding, empty binding names, and out-of-scope reads. No production card or legacy export changed.

Homarid Spawning Bed's complete text was checked in `data/FEM.json` and its Fallen Empires inventory and JSON gap record. Its token action already has a typed path, but full card migration remains separate; the gap record stays open. Focused tests first failed because the constructor was absent. Synthetic engine and public DSL tests cover source payment, a nonmatching other creature, mana value after source departure, clone isolation, stale-incarnation rollback, registration, and compile-time binding type rejection. Existing payment tests cover independent pending activations and spell additional costs for the shared result path. The next dependency-ready work is another P payment/action result or a T target extension after a fresh requirements audit.

Validation: focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Attached-Permanent Tap Payment

`TapAttachedPermanentCost() Cost` pays an activated ability cost by tapping the permanent attached to its battlefield source. The payment transaction locks the source and host incarnations and their attachment during preparation, then checks them before commit. It reuses the shared tap event and total-cost transaction. An absent, phased-out, detached, returned, or already tapped host cannot pay. Phasing in preserves a valid host. This cost does not impose the `{T}` symbol's summoning-sickness restriction or a host-controller restriction. The separate `PermanentIsUntapped(AttachedPermanent)` state condition expresses an explicit activation condition. The API is valid in `Costs` for an activated ability; registration rejects it as a spell additional cost. No production card or legacy export changed.

Tourach's Gate's complete text was checked in `data/FEM.json` and its Fallen Empires inventory and gap record. Its production wiring remains open, including its attacking-creature effect and other abilities; the gap record stays blocked. Focused tests first failed because the constructor was absent. Engine and public DSL tests cover payment, already tapped and unattached hosts, attachment changes, phasing, source and host leave-and-return, life-cost rollback, clone isolation, and registration. A fresh production caller search found no use of the new cost. The next dependency-ready work is another isolated P payment or action result after a requirements audit; T divided-value announcement also remains open.

Validation: focused engine and DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Exact Matching-Permanent Tap Payment

`TapNOtherMatchingCost(count, filter, text)` pays by tapping exactly that many distinct untapped matching permanents controlled by the payer, excluding the ability source. It is a `Cost` for `Costs`, spell additional costs, and `IfPaid`/`UnlessPaid`. It uses the existing isolated payment transaction and tap events. Preparation makes each choice once and records battlefield incarnations; commit rechecks control, eligibility, phasing, and identity before changing live state. All selected permanents become tapped before their tap events are sent. Other creatures can pay despite summoning sickness; the source's own `{T}` still uses `TapCost(This)`. Registration rejects nonpositive counts and empty text. The cost has no output binding and does not change legacy `TapMatchingCost` callers.

Hand of Justice's complete Oracle text in `data/FEM.json`, the Fallen Empires inventory and gap record, and CR 601.2h and 602.2 supply the acceptance case. Synthetic engine tests first failed because the constructor was absent. Focused tests cover three distinct white creatures, source and opponent exclusion, summoning sickness, event visibility, one-time choices, changed color, tap state, phasing, leave-and-return, clone isolation, insufficient resources, failed-payment rollback, and registration. A public DSL test compiles a Hand of Justice-shaped declaration. Production card wiring and its gap record remain open; no production card changed. Existing `TapMatchingCost` production callers remain. The next dependency-ready work is a separate P payment output or T target extension after a fresh audit.

Validation: `go test ./pkg/mage -run TestTapNOtherMatchingCost -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The first repository-wide short run hit a full Go build cache during linking; `go clean -cache` restored disk space, and the rerun passed.

## Current Slice: Turn-Limited Delayed Battlefield Departure

`WhenLeavesThisTurn(PermanentRef, ResolvingEffect)` watches one battlefield incarnation for departure to any zone before cleanup. It works in spell, activation, and trigger bodies. The adapter reuses the delayed death subscription, pre-departure zone event and LKI, captured source/controller/bindings, pending-trigger placement, and clone paths. Phasing does not match. A returned card has a new incarnation. The sealed subject role rejects players and sets at compile time; registration rejects an absent body or subject and an undominated binding. No legacy export or production card changed.

War Barge's complete repository Oracle text (`data/DRK.json`), The Dark inventory, and CR 400.7, 514.2, 603.7, and 603.10a supply the acceptance case. Focused tests first failed because the constructor was absent. They now cover graveyard, hand, and exile destinations, phasing, source departure, saved creature identity, leave-and-return, simultaneous source and watched departure, absent subject, cleanup expiry, clone isolation, inspection, and registration. The new public author declaration compiles. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. One parallel build initially ran out of disk space while linking `cmd/pylib`; the standalone retry passed.

War Barge retains its production wiring and full card migration remains open. The Dark inventory is an untracked input, so this slice does not modify it or close a set-wide gap record. Production still uses `RegisterDelayedTriggerStep` in Antiquities and Legends; the legacy export remains. General delayed event patterns, reflexive triggers, and participant captures remain separate work. The next dependency-ready work in delivery order is a P payment result or T target-group extension after a fresh requirements audit.

## Current Slice: Turn-Limited Delayed Death Trigger

`WhenDiesThisTurn(PermanentRef, ResolvingEffect)` installs a delayed event subscription for one current battlefield incarnation. It works in spell, activation, and trigger resolution. The watched permanent need not remain a creature after installation. A battlefield-to-graveyard event queues the body under the creating source and controller; the subscription expires at cleanup. The adapter reuses `DelayedTrigger`, the pre-departure zone batch and LKI, pending-trigger placement, and the existing captured delayed body. It adds an incarnation check to delayed event matching. A player or permanent set is a compile-time category error; registration rejects absent declarations and a read without a dominating producer. The body has its own binding scope. No legacy export or production card changed.

The full repository text of Sandals of Abdallah (`data/ARN.json`) and Time to Feed (`data/JMP.json`), their inventories, CR 603.7, 603.10a, 700.4, and cleanup rule 514.2 supply the acceptance cases. Focused tests first failed because the constructor was absent. They now cover a target and a saved permanent, source departure, simultaneous source and watched death, replacement to exile, phasing, return before and after death, cleanup expiry, clone isolation, inspection, and registration. `go test ./pkg/mage -run TestWhenDiesThisTurn -count=1`, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

A fresh search found no production `WhenDiesThisTurn` caller. Sandals and Time to Feed retain their existing production wiring and remain open for separate migrations; neither JSON gap record is resolved by an adapter alone. Legacy delayed-step callers remain, including `RegisterDelayedTriggerStep` users in Antiquities and Legends. A general turn-limited delayed leave-event subscription for a captured incarnation is the next dependency-ready event slice, needed by War Barge; general delayed event patterns, reflexive triggers, and card migration remain separate work.

## Current Slice: Creature-Death Subtype Filter

`CreatureDies().OfSubtype(string)` and the other creature-death patterns now filter the dying creature by one creature subtype in battlefield `WhenEvent` and `WhenEventModes` abilities. The adapter uses the existing pre-departure zone-change observer and `EventSourceHasSubType`; it adds no death executor or delayed subscription. Registration rejects empty, noncreature, repeated, and noncreature-pattern subtype declarations. The public declaration compiles without a new export because the existing constructor returns the sealed pattern with its method.

The plan's filtered-death family and CR 603.10a and 700.4 supply the contract. Focused tests first failed because `OfSubtype` was absent; they now cover effective subtype, other-creature filtering, a simultaneous source-and-subject death, a nonmatching subtype, phased-out source, pending-trigger copy through a game clone, card return, cleared LKI, and registration validation. `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production caller of the new method; existing death-pattern callers and legacy exports remain. No production card or JSON gap changed. The earlier tracker suggestion that Venom needs a new delayed end-of-combat constructor was stale: `AtNext(EachEndOfCombat, Destroy(From(binding)))` already captures a permanent incarnation; Venom still needs a separate card migration. The next dependency-ready work is a turn-limited delayed death-event subscription for captured permanents, required by Sandals of Abdallah and Time to Feed.

## Current Slice: Combined Per-Pair Combat Event

`BlocksOrBecomesBlockedByCreature(This)` is a sealed battlefield event pattern for `WhenEvent` and `WhenEventModes`. It matches each new legal opposing-creature pair in either combat direction. `.OfSubtype(string)` and `.WithoutSubtype(string)` inspect the opponent when the pair event occurs. `.WithOpponent(*PermanentBinding)` saves its battlefield incarnation in the pending trigger; it does not announce a target. `EvtCombatPair` reports both perspectives of a legal declared or effect-added pair and only the attacker perspective when a creature enters blocking. Existing blocker and blocked-by events and legacy triggers remain intact. Registration rejects empty, noncreature, repeated, and conflicting filters; invalid or repeated captures; and a body read without a producer. The pattern has no resolving-action or static-rule lifecycle.

The complete Venom text in `data/4ED.json`, the Fourth Edition inventory and JSON gap, and CR 509.3b, 509.3d, 509.3f, and 509.4 supply the acceptance case. The engine and public DSL tests cover multiple pairs in both directions, a later pair for the same blocker, non-Wall filtering, entry blocking, event-time characteristics, source departure and return, opponent departure and return, phasing, clone isolation, inspection, and registration validation. The focused test first failed because the constructor was absent. `go test ./pkg/mage -run TestCombatPairPattern -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. No production card changed. A fresh caller search found no production use of the typed combined pattern and one `BlocksOrBecomesBlockedTrigger` caller in The Dark; legacy exports remain. Venom still needs a delayed end-of-combat destruction contract and a separate card migration, so its gap record stays open. That delayed contract is the next dependency-ready slice.

## Current Slice: Per-Pair Source Blocks Creature

`BlocksCreature(This)` is a sealed battlefield event pattern for `WhenEvent` and `WhenEventModes`. It matches each legal attacker/blocker pair in which the source blocks an attacking creature. `.OfSubtype(string)` and `.WithoutSubtype(string)` inspect that attacker when the pair event occurs. `.WithOpponent(*PermanentBinding)` saves its battlefield incarnation in the pending trigger; it does not announce a target. The adapter uses the existing `EvtDeclaredBlocker`, legal-combat declaration, pending-trigger capture, and resolution-frame clone services. It has no new combat executor. Registration rejects invalid or repeated subtype filters, invalid bindings, and a body read without a producer. The pattern has no resolving-action or static-rule lifecycle.

The complete Venom text in `data/4ED.json` and CR 509.3b, 509.3d, 509.3f, and 509.4 require per-pair matching in both blocking directions. This slice covers only the source-blocks direction. `BecomesBlockedBy(This)` covers the attacker direction separately. A combined per-pair pattern and delayed destruction remain for Venom; the Fourth Edition gap stays open and production wiring is unchanged. The legacy `BlocksTrigger` and `BlocksOrBecomesBlockedTrigger` exports remain because production callers exist. The next dependency-ready slice is the combined per-pair event pattern with one opponent capture across both directions.

The focused tests first failed because `BlocksCreature` was absent. They now cover multiple pairs including a later pair with `Flag=false`, non-Wall filtering, opponent incarnation after return, phasing, clone isolation, inspection through registration, and invalid declarations. `go test ./pkg/mage -run TestBlocksCreature -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The documentation example compiles with `go build`.

## Current Slice: Blocker-Group Subtype Exclusion

`BlocksOrBecomesBlockedBy(This).WithoutSubtype(string)` is a sealed event-pattern filter for battlefield `WhenEvent` and `WhenEventModes` declarations. It matches one new blocker group if at least one opposing creature lacks the specified creature subtype at event detection. The existing `EvtBlockerGroup`, pending trigger, source identity, and clone paths remain the reuse boundary. The filter has no result or participant binding. Registration rejects an empty or noncreature subtype and repeated or conflicting filters. No production card or legacy export changed.

The complete Venom text in `data/4ED.json` was checked against CR 509.3b, 509.3d, and 509.3f. Venom requires a separate trigger for each opposing non-Wall creature, including when the enchanted creature blocks. A one-or-more group trigger cannot express that rule, even with a group member set. The prior tracker entry suggesting group capture as Venom's next step was stale. This group filter supports only one-or-more event clauses. Venom needs a per-pair source-blocks or source-becomes-blocked event and an opposing-creature incarnation capture, followed by delayed destruction and card wiring. Its Fourth Edition gap stays open; no gap record changed. A fresh production search found no `BlocksOrBecomesBlockedBy` caller and one `BlocksOrBecomesBlockedTrigger` caller in The Dark; legacy combat exports remain available.

The focused tests first failed because `WithoutSubtype` was absent. Synthetic and public DSL tests cover mixed groups, a Wall-only group, event-time subtype, the blocking role, phasing, clone isolation, inspection, and declaration rejection. `go test ./pkg/mage -run TestBlocksOrBecomesBlockedByWithoutSubtype -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The next dependency-ready E slice is the per-pair blocking event contract and opponent capture for Venom, after its two event directions are audited.

## Current Slice: Blocked-By Subtype Exclusion

`BecomesBlockedBy(This).WithoutSubtype(string)` is a sealed event-pattern filter for a battlefield source in `WhenEvent` and `WhenEventModes`. It checks the blocking creature's subtype when the per-pair combat event occurs. It composes with `WithBlocker` and rejects empty, wrong-family, repeated, or conflicting subtype filters at registration. The existing blocker event, pending-trigger frame, and captured permanent reference remain the reuse boundary. No production card or legacy export changed.

Venom's complete text in `data/4ED.json` requires a non-Wall creature filter on a blocks-or-becomes-blocked event. This slice supplies the per-blocker exclusion only. A later CR 509.3b/509.3d audit established that Venom needs a per-pair event in both blocking directions; the group event cannot express it. Delayed destruction and card wiring also remain. No JSON gap record is resolved. Focused tests first failed because `WithoutSubtype` was absent. They now cover legal mixed blocking, event-time subtype capture, stack copies, clone isolation, inspection, and invalid declarations; existing blocker-binding tests cover absence, phasing, leave and return, and source identity.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh production search found no `BecomesBlockedBy` caller; existing combat event and legacy exports remain available during card migration.

## Current Slice: Optional Typed Attachment Move

`MayAttach(PermanentRef, PermanentRef)` is a resolving action for spells, activations, and triggers. It resolves two current battlefield incarnations, offers the resolving controller a choice only for a legal move, and sends an accepted move through `Game.Attach`. The shared legality check, timestamp order, continuous effects, and attach event remain in one engine path. An absent, phased-out, returned, illegal, or same-host pair causes no choice or move. Registration checks both sealed permanent references and binding dominance. No production card or legacy export changed.

Ajani's Chosen's complete text was checked in `data/JMP.json`. A synthetic enchantment-entry trigger combines the existing event capture, token result, resolution-time Aura branch, and this action. Its production declaration and full card audit remain for a separate migration. The Jumpstart inventory and JSON gap record stay open. A fresh caller search finds 35 direct `Game.Attach` uses in production cards and no production `AttachToTarget` call; existing exports remain. The focused tests first failed because `MayAttach` did not exist. They now cover accept and decline, controller choice, legality, incarnation and phasing, inspection, registration, and clone isolation. The next dependency-ready work is a separate P result or B simultaneous collection action after a fresh inventory audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Attachment Move Timestamp

`Game.Attach(sourceID, targetID)` now moves an attachment's printed static effects to the current end of timestamp order after a successful move. It keeps their relative order. This uses the existing effect manager and attachment legality checks; the public engine signature does not change. Failed and same-host attempts leave the order unchanged. Game clones have independent effect order. The focused synthetic test first failed because a reattached Aura still applied before an earlier Aura. It covers opposing ability grants and removals, same-host and illegal moves, phasing, and clone isolation. No typed author action or production card changed.

Ajani's Chosen's text in `data/JMP.json` requires an optional Aura attachment after token creation. The timestamp prerequisite now supports the shared move, but the typed optional action and card wiring remain open. The Jumpstart inventory and gap record remain open. The next dependency-ready slice is that typed optional attachment action, after an audit of its choice and binding rules.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues after one local style fix), and `git diff --check` pass. A fresh search found 35 `Attach` call sites in production card files; existing engine callers remain. The public signature and all legacy exports stay in place.

## Current Slice: Attachment Move Legality

`Game.Attach(sourceID, targetID)` now checks attachment source kind, Aura enchant restrictions, protection, and attachment prohibitions before a move. It rejects phased-out or absent participants and same-host moves without changing attachment state. Successful moves record the host incarnation and continue through the shared effect and event paths. This is an engine prerequisite; it adds no typed author action or production card wiring. Synthetic tests first failed on wrong-type Aura hosts, protection, and ordinary creature attachment. They cover failed-move state, same-host behavior, phasing, departure and return, and clone isolation.

Ajani's Chosen's full Oracle text was checked in `data/JMP.json`. Its optional Aura-to-created-token clause still needs attachment timestamps, a typed optional attachment action, and a separate card migration. The Jumpstart inventory and gap record remain open. The next dependency-ready slice is attachment timestamp ordering under CR 701.3c and 613.7e. Existing `Game.Attach` callers remain available during migration.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. Three synthetic tests previously attached default creature Auras to artifacts; their declarations now specify enchant artifact. No card definition or gap record changed.

## Current Slice: Created-Token Result Set

`BindCreatedTokens`, `CreateTokenCopiesInto`, and `FromCreatedTokens` provide a typed collection of the accepted battlefield incarnations from one token-entry batch. The action is available in spell, activation, and trigger bodies, and the saved set can be read by later resolving and delayed-step instructions. It reuses `createTokenBatchWithResult` and the existing permanent-set consumers. The result excludes rejected entries, departed or returned tokens, and phased-out tokens. A nonpositive count clears the binding. Declaration inspection and dominance checks reject invalid or unproduced bindings. The frame, loops, delayed steps, and game clones copy result slices independently. No production card or legacy export changed.

The complete Oracle text of Snarl Song and Wild Hypothesis was checked against `data/SOS.json` and the Secrets of Strixhaven inventory and JSON gap record. Snarl Song needs the new two-token identity set, but also needs simultaneous counter placement and its converge value before its production declaration is complete. Wild Hypothesis creates one token, so the existing `CreateOneToken` covers its token identity; its counter and surveil clauses still need a complete card migration. The inventory and gap record stay open. A fresh search found 39 production `CreateTokenCopies` calls and no `CreateTokenCopiesInto` call; the legacy export remains for those callers. The next dependency-ready work is the attachment legality/timestamp prerequisite for an optional Aura attachment action, or a simultaneous counter-placement action for a saved permanent set.

Focused tests first failed because the new constructors and frame field were absent. Synthetic and public DSL tests cover batch identity, 0/0 tokens followed by counters before state-based actions, controller, old same-name objects, zero and rejected entries, leave and return, phasing, delayed capture, clone isolation, inspection, and registration dominance. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` passed before the final review.

## Current Slice: Singular Created-Token Result

`CreateOneToken(PlayerRef, TokenDefinition, *PermanentBinding)` creates one token through the shared token entry batch and saves the accepted battlefield incarnation in the resolving frame. `From(binding)` can feed later singular instructions. The result is absent if no token enters, if it phases out, or if it leaves; a same-card return cannot satisfy the old incarnation. Registration checks the player, token definition, named binding, and producer dominance. The typed result is available in spells, activations, and triggers. No production card or legacy export changed.

Ajani's Chosen's complete text was checked in `data/JMP.json` and its Jumpstart gap record. This slice supplies the created-token identity needed for its optional attachment clause, but the attachment choice and production wiring remain open. The Secrets of Strixhaven Fractal cards require a multi-token result set, which this singular binding does not supply. Both gap records remain open. The first focused test failed because the constructor was absent. Synthetic and public DSL tests cover the created incarnation, controller, clone isolation, absence, phasing, subsequent action, inspection, and registration. The next dependency-ready slice is an attachment action or a multi-token result collection after a fresh audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role check, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `CreateOneToken` caller; existing `CreateTokenCopies` and legacy token constructors remain for unmigrated cards.

## Current Slice: Resolution-Time Enchantment Subtype Branch

`IfPermanentHasEnchantmentSubtype(PermanentRef, string, ResolvingEffect)` checks a current or saved battlefield incarnation when a spell, activation, or trigger body resolves. For a saved permanent that departed, it reads the `ZoneSystem` LKI snapshot keyed by incarnation; a returned card does not replace that object. A phased-out object has no current subject. The branch uses the existing subtype and resolution services. It adds no token-result or attachment action. Registration checks the sealed permanent role, a known enchantment subtype, body presence, and binding dominance. No production card or legacy export changed.

Ajani's Chosen's complete text was checked in `data/JMP.json`. This branch can express its resolution-time Aura condition, after token creation. Its optional attachment to the created token, token-result binding, full card audit, and production wiring remain open. The Jumpstart inventory and JSON gap stay open. A fresh search found no production caller of `EnchantmentYouControlEnters` or the new branch; Ajani's Chosen still uses its legacy declaration. The next dependency-ready work is the token-result binding prerequisite for optional attachment, or another independent E/P slice after a fresh audit.

The first focused test failed because the constructor was absent. Synthetic and public DSL tests cover matching and nonmatching subtypes, departure LKI with effective characteristics, leave and return, phasing, illegal targets, trigger stack copies, game clones, inspection, and invalid declarations. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Controlled Enchantment Entry Subtype

`EnchantmentYouControlEnters().OfSubtype(string)` is a typed entry-time enchantment subtype filter for battlefield `WhenEvent` and `WhenEventModes` declarations. It reuses the effective entry-subtype condition and existing controlled-enchantment trigger. It composes with `WithPermanent` without changing the captured incarnation. Registration rejects an empty, unknown, wrong-family, or repeated subtype. No production card or legacy export changed.

Ajani's Chosen's complete Oracle text was checked in `data/JMP.json`. Its "If that enchantment is an Aura" clause is a condition during resolution, after token creation, so this entry filter does **not** express that clause. Its optional token attachment and production wiring remain open. The Jumpstart inventory and JSON gap record remain open. The next dependency-ready work is a typed resolution-time subtype condition on a bound permanent, together with the required token-result and attachment contracts, or an independent E/P slice after a fresh audit.

Focused tests first failed because `OfSubtype` was absent. Synthetic and public DSL tests cover effective subtype at entry, controller and owner distinction, source phasing, saved incarnation across departure and return, stack copying, game cloning, inspection, and registration rejection. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `EnchantmentYouControlEnters` caller; existing entry and legacy exports remain for unmigrated cards.

## Current Slice: Controlled Enchantment Entry

`EnchantmentYouControlEnters().WithPermanent(*PermanentBinding)` is a sealed battlefield event pattern for `WhenEvent` and `WhenEventModes`. It matches an enchantment entering under the trigger controller's control, including the source itself. The optional binding saves the entering battlefield incarnation in the pending trigger frame. The adapter reuses the zone-entry event, effective type and controller checks, and existing permanent capture. It adds no entry replacement or card-specific rule. A saved object is absent after leaving and returning or while phased out. Registration rejects nil, unnamed, repeated, and unbound captures; the typed method excludes player and card bindings.

Ajani's Chosen's complete text was checked against `data/JMP.json` and the Jumpstart inventory and gap record. The event and entering-object binding support its first clause, but the Aura test and optional attachment to the created token remain separate capabilities. Its production declaration and gap record stay open. No production card or legacy export changed. Focused synthetic tests first failed because the constructor was absent. They cover owner/controller distinction, nonenchantments, opponent control, source phasing, self-entry, source and entrant departure and return, stack copying, game cloning, inspection, and registration validation. A public DSL test compiles and validates the example. The next dependency-ready work is an entry-time Aura characteristic filter or a separate E/P participant or result prerequisite after a fresh audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `EnchantmentYouControlEnters` caller; Ajani's Chosen retains its existing card code. No gap status changes because the remaining Aura clause and card wiring are open.

## Current Slice: Self-Inclusive Controlled-Creature Entry

`CreatureYouControlEnters()` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It includes the source's own entry and shares the existing controlled-creature entry detector, effective-characteristic filters, entry power and incarnation captures, and queued-trigger frame with `AnotherCreatureYouControlEnters()`. It adds no event subsystem or card-specific rule. No production card or legacy export changed. Archon of Redemption's complete text was checked in `data/JMP.json`: this pattern can express its self-entry event, while its other-creature-with-flying branch, optional life-gain choice, full declaration audit, and card wiring remain separate. The Jumpstart inventory and gap record stay open.

The first focused test failed because the constructor was absent. Synthetic and public DSL tests cover self-entry, other controlled entry, opposing control, keyword filtering, entry power capture, phasing, queued source departure and return, clone isolation, inspection, and invalid capture declarations. Existing entry-pattern tests cover stack copies and entrant incarnation. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production caller of `CreatureYouControlEnters`; existing entry and legacy exports remain for unmigrated cards. The next dependency-ready work is a separate E event participant or P payment output after a fresh audit.

## Current Slice: Creature-Entry Keyword Filter

`AnotherCreatureYouControlEnters().HasKeyword(core.Keyword)` matches another creature entering under the trigger controller's control when it has the keyword after entry effects and continuous effects apply. It uses the existing battlefield entry event, effective permanent attributes, and pending-trigger path. The method is valid in battlefield `WhenEvent` and `WhenEventModes` declarations and composes with the pattern's nontoken, subtype, and capture options. Registration rejects a nonkeyword attribute and a repeated keyword filter. No production card or legacy export changed.

Archon of Redemption's complete Oracle text was checked against `data/JMP.json`. This filter and the existing entry power capture cover its other-creature-with-flying event requirement. Its self-entry branch, optional life gain, full declaration audit, and production wiring remain open; the Jumpstart inventory and gap record stay open. The next dependency-ready work is a separate E event participant or P payment output after a fresh audit.

Focused tests first failed because `HasKeyword` was absent. Synthetic tests cover opposing control, missing and static granted or removed keywords, source phasing, source and entering-creature departure and return, game clone isolation, inspection, and registration rejection. A public DSL test compiles the reference declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production caller of `AnotherCreatureYouControlEnters`; Archon of Redemption still uses its legacy declaration, so legacy exports remain.

## Current Slice: Creature-Entry Power Capture

`AnotherCreatureYouControlEnters().WithPower(*NumberBinding)` saves the entering creature's effective power when the entry event is detected. `FromNumber` reads the instance-local value in the trigger body, even after the creature changes, leaves, or returns. It uses the existing event detector, continuous-power query, and trigger frame. The pattern is valid in battlefield `WhenEvent` and `WhenEventModes` declarations. Registration rejects nil, unnamed, duplicate, and unproduced bindings. No production card or legacy export changed.

Archon of Redemption's complete Oracle text was checked in `data/JMP.json`. This slice supplies its event-time power value for another controlled creature, but the card also needs a self-entry branch, flying filter, optional life gain, and production wiring. The Jumpstart inventory and gap record stay open. Focused tests first failed because `WithPower` was absent. Synthetic and public DSL tests cover control filtering, phasing, changed power, departure and return, stack copying, clone isolation, inspection, and declaration validation. Focused tests, the short engine and repository suites, DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `WithPower` caller on the entry pattern; existing event and legacy exports remain for unmigrated cards. The next dependency-ready slice is another E event-participant or P payment output after a fresh audit.

## Current Slice: Creature-Death Power Capture

`CreatureDies().WithPower(*NumberBinding)` and the other creature-death patterns save the dying creature's pre-departure power in each pending trigger. The adapter reuses the existing departure view, event detection, and instance-local number binding. It is valid in battlefield `WhenEvent` and `WhenEventModes` declarations. Registration rejects a nil or unnamed binding, a repeated power capture, a general `PermanentDies()` pattern, and reads without a producer. No engine event storage, production card, or legacy export changed.

Death Watch's complete Oracle text was checked in `data/VIS.json` as an acceptance example for power and toughness at death. The public `WithPower` adapter does not complete that card: its Aura, event controller, life-loss, and production wiring need a separate complete migration. Existing set inventories and JSON gaps remain unchanged because none is fully resolved by this capture alone. The next dependency-ready work is a separate E event participant or P payment output after a fresh audit.

The contract tests first failed because `WithPower` was absent. Synthetic tests cover continuous-effect power, departure and return after LKI cleanup, source death in a simultaneous batch, phasing, stack-copy and game-clone isolation, inspection through registration, and invalid declarations. A public DSL test compiles and validates the declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. A fresh production search found no `WithPower` caller; existing death-event and legacy exports remain for unmigrated cards.

## Current Slice: Death-Event Power or Toughness Filter

`CreatureYouControlDies().PowerOrToughnessAtMost(1)` and the other creature-death patterns filter the departing creature's power **or** toughness before death. The typed adapter uses the existing pre-departure object view, zone-change event, and pending-trigger path. It accepts any integer limit and rejects a repeated filter or use on a general `PermanentDies()` pattern at registration. No engine event storage, production card, or legacy export changed.

Arnyn, Deathbloom Botanist's full Oracle text was checked in `data/SOS.json`. This filter supplies its missing 1-or-less event condition; its target-opponent effect can use existing typed instructions. The production definition and Secrets of Strixhaven inventory and gap record remain open until a separate card migration. Tests first failed because the method was absent. Synthetic tests cover either qualifying characteristic, a nonmatching creature, continuous-effect changes, controller filtering, phasing, simultaneous source and subject deaths, graveyard replacement, departure and return, stack copies, clone isolation, inspection, and declaration validation. A public DSL test compiles the trigger and target shape. The next dependency-ready work is a separate E event participant or P payment output after a fresh audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. Arnyn still uses the legacy `DiesCreatureTrigger`; that export remains for this and other production callers.

## Current Slice: Nonsacrifice Death Event Filter

`PermanentDies().YouControlled().NotSacrificed()` filters a completed battlefield-to-graveyard event by the departing permanent's former controller and sacrifice outcome. Both filters also compose with creature-death patterns. The adapter uses the existing zone-change event, its sacrifice flag, departure LKI, and pending-trigger service. It works in battlefield `WhenEvent` and `WhenEventModes` declarations. Registration rejects repeated filters. The pattern does not introduce a target or a result binding. No engine event contract, production card, or legacy export changed.

Urza's Miter's complete Oracle text was checked in `data/ATQ.json`. The new filters, `OfType(TypeArtifact)`, and existing `WithController` express its trigger condition and event-time player. The production declaration in `cards/antiquities/artifacts.go` still uses `NewTriggered` and legacy conditions; it already has an `IfPaid` body. The Antiquities inventory and gap record remain open until card migration verifies the complete declaration. The next dependency-ready work is a separate E event participant or P payment output after a fresh audit.

Contract tests first failed because the new filters were absent. Synthetic tests cover controller versus owner, sacrifice versus destruction, exile replacement, source absence, phasing, departure and return, simultaneous source death, stack copy and game clone isolation, inspection, and registration rejection. A public DSL test compiles and validates the composed declaration. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. No production caller uses either new filter yet; existing exports remain available.

## Current Slice: Damage-History Recipient Sets

`PlayersDealtDamageBy(DamageSourceRef, PlayerSet) PlayerSet` and `PermanentsDealtDamageBy(DamageSourceRef, PermanentSet) PermanentSet` filter current recipients by positive game-long damage from one permanent-source incarnation. They reuse the captured-source resolver and the damage-system history queries. The sets work in resolving spell, activation, and trigger bodies through existing set consumers; together with `DamageRecipients` and `DealDamageEach`, they express The Fallen's upkeep recipient rule without a second damage executor. Spell sources have no permanent incarnation and select nothing. No production card or legacy export changed.

The Fallen's complete Oracle text was checked in `data/DRK.json`. Its production declaration remains unwired; the The Dark inventory and gap record stay open. Existing card-ID history callers remain on their legacy queries. This slice does not add turn-scoped history for Giant Shark, historical departed recipients as action targets, or source selection outside existing `DamageSourceRef` forms. The next dependency-ready work is a separate E event participant or P payment output after a fresh audit.

Focused tests first failed because the typed set constructors were absent. Synthetic and public DSL tests cover player and planeswalker selection, simultaneous damage, full prevention, source and recipient departure and return, phasing, durable LKI, clone isolation, inspection, invalid declarations, and a compile-time role error. `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh production search found no callers of either new set filter; The Fallen's production wiring remains on the legacy path.

## Current Slice: Permanent Damage History Identity

`(*Game).HasDealtDamageToPlayerFromSource(DamageSource, uuid.UUID) bool` and `(*Game).HasDealtDamageToPermanentFromSource(DamageSource, *Permanent) bool` read game-long damage history for one captured permanent source incarnation. The permanent query also identifies the recipient incarnation. The shared damage executor records positive final damage after replacement and prevention; game clones copy the records. Tests first failed because the queries were absent, then covered source and recipient departure and return, cleared LKI, prevention, phasing, simultaneous player and permanent damage, and clone isolation. The old card-ID history queries and their production callers remain unchanged.

The Fallen's complete text was checked in `data/DRK.json`. This prerequisite distinguishes its source and planeswalker history across battlefield visits, but its typed history selector, simultaneous damage action wiring, and production declaration remain separate work. Giant Shark's turn-scoped damage history is also outside this slice. No production card or gap record changed. The next dependency-ready work is a typed permanent-source damage-history selector or another isolated P payment output; spell-source and turn-scoped histories need separate contracts. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The first engine run exposed a clone allocation regression, which the lazy clone maps corrected before the passing runs.

## Current Slice: Graveyard Exile Payment Mana Value

`ExileMatchingCardFromGraveyardCostInto(CardFilter, string, *NumberBinding) Cost` locks a chosen matching card and its graveyard visit during cost preparation. A successful commit saves the card's pre-exile mana value in the spell or activation frame. The adapter reuses the existing graveyard-exile cost, total-cost transaction, and number-binding frame. It leaves the legacy cost and its callers intact. Necropolis's complete text was checked in `data/DRK.json`; its production wiring and The Dark gap record remain open. This slice does not add a card binding, a chosen graveyard, or a general payment-result API.

Focused tests first failed because the constructor was absent. Synthetic tests cover interleaved activations, stack copying, game cloning, stale graveyard visits, failed payment, inspection, and registration rejection for invalid or unproduced bindings. A public DSL test compiles and validates the declaration. The next dependency-ready work is another P payment output or a T divided-damage announcement contract after a fresh audit.

`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found one production caller of the legacy graveyard-exile cost (Necropolis) and no production caller of the new constructor; both exports remain available. No production card or JSON gap record changed.

## Current Slice: Fixed Counter-Payment Capture

`RemoveCountersCostInto(counter core.CounterType, count int, paid *NumberBinding) Cost` removes a fixed nonnegative number of counters from a battlefield source during activation payment and saves the committed count in that activation's number-binding frame. It uses the existing counter cost and total-cost transaction. Preparation locks the source incarnation; a failed commit leaves no capture. Zero requires a present source. The binding follows stack copies and game clones. This slice adds no chosen-count payment, counter replacement rule, card migration, or legacy API retirement. Existing `RemoveCountersCost` callers remain compatible.

The focused contract tests first failed because `RemoveCountersCostInto` was absent. Synthetic activations cover positive and zero payments, interleaved stack instances, source departure and return, phasing, insufficient counters, failed-payment life rollback, stack copying, game cloning, and registration rejection for nil, unnamed, negative, duplicate, and unproduced bindings and spell use. A public DSL declaration test checks the exported signature. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found `RemoveCountersCost` production calls in nine card files and no `RemoveCountersCostInto` production calls; the legacy export remains. The Legends Mana Battery inventory calls for a chosen count, which `RemoveAnyCountersCost` already supplies; this fixed-count adapter does not close that card wiring or any JSON gap. The next dependency-ready work is a separate P payment output or T divided-damage announcement after a fresh audit.

## Current Slice: Simultaneous Permanent Return to Hand

`ReturnEachToHand(PermanentSet) ResolvingEffect` uses the shared `Game.BouncePermanentsToHand` zone-change batch. It selects battlefield incarnations once at resolution and returns each eligible permanent to its owner's hand in one action. Phased-out, absent, returned, and duplicate incarnations are excluded; tokens leave but do not remain in hand. The effect is available in resolving spell, activation, and trigger bodies. Its inspection reports a mass bounce and validates the permanent-set declaration. No card was changed.

Hurkyl's Recall's complete text was checked in `data/ATQ.json`. `PermanentsOwnedBy(targetPlayer, IsArtifact)` and the new action express its collection and simultaneous return, but production wiring remains on the callback path. The Antiquities inventory and JSON gap record remain open. The existing singular `ReturnToHand` and legacy bounce exports remain for production callers. Focused tests first failed because the batch constructor was absent. A later contract test caught stale incarnations claiming a card ID before the current incarnation; the batch now checks incarnation before duplicate suppression. Tests cover owner/controller separation, phasing, incarnation change, tokens, duplicates, clone isolation, one-batch state observation, declaration rejection, and public DSL compilation. `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The next dependency-ready work is a separate P payment output or T divided-damage target contract after a fresh audit.

## Current Slice: Owner-Filtered Permanent Set

`PermanentsOwnedBy(PlayerRef, PermanentFilter) PermanentSet` selects current battlefield permanents owned by the resolved player, regardless of controller. It works in resolving spell, activation, and trigger bodies through existing set consumers. It reuses the battlefield filter, player binding, and permanent-set inspection paths. Registration rejects a missing or undeclared player reference. No engine zone contract or production card changed.

Hurkyl's Recall's complete text was checked in `data/ATQ.json`. This set supplies its owner-filtered artifact collection, but simultaneous return-to-hand and production wiring remain open. Its definition in `cards/antiquities/spells.go` still uses a targeted spell with a callback. The Antiquities inventory and gap record stay open; no legacy export was removed. Synthetic tests first failed because the constructor was absent. They cover owner/controller separation, type filtering, phasing, departure and return, target binding, stack-copy controller, game-clone isolation, inspection, and declaration validation. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The next dependency-ready work is the simultaneous return action or a separate P/T slice after a fresh audit.

## Current Slice: Permanent Owner Reference

`OwnerOf(PermanentRef) PlayerRef` is a typed resolving-time owner reference for one battlefield incarnation. It works in spell, activation, and trigger bodies through existing player-consuming instructions, including `SnapshotPlayer`. It reads the card owner from the resolved permanent and remains distinct from `ControllerOf`; absence, phasing, and a returned incarnation yield no player. The existing permanent resolver, player frame, inspection graph, and target scope validation supply the behavior. There is no new engine ownership contract or production card change.

Focused contract tests first failed because `OwnerOf` was undefined. They now cover differing owner and controller, phasing, leave and return, snapshot and game-clone isolation, target legality after phasing or return, registration rejection for nil and undeclared references, and a public DSL declaration. The complete Personal Incarnation text and its owner distinction were checked against `data/LEA.json` before selecting this acceptance case. Its death-trigger wiring, owner-only activation, damage redirection, and production migration remain open, so no gap record is closed. Existing `ControllerOf` and legacy ownership callers remain. The next dependency-ready work is a separate S owner-filtered permanent set or P payment output after a fresh audit.

`go test ./pkg/mage -run '^TestOwnerOf' -count=1`, the public DSL focused test, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. No production card or gap record changed.

## Previous Slice: Combat Opponent Set

`BlockingOrBlockedBy(PermanentRef) PermanentSet` selects the creatures currently blocking or blocked by one battlefield incarnation. It reads combat groups at the consuming instruction, excludes phased-out or returned objects, and returns each member once. The set can feed `CountPermanents`, `ForEach`, or a collection modification such as `Until(EndOfTurn, GrantEachKeyword(...))`. Registration rejects an absent declaration; the role excludes player and card references at compile time. It reuses the combat group and permanent-set services. No production card changed.

Spitting Slug's complete text was checked in `data/DRK.json`. This selector supplies the unpaid branch's creature set, but its optional payment and production wiring remain. The Dark inventory and gap record stay open. A fresh search found one production `BlocksOrBecomesBlockedTrigger` and one `ToBlockingOrBlockedBySource` caller, both in Spitting Slug; both legacy exports remain. The next dependency-ready work is a separate E event participant or P payment output after a fresh caller audit.

The focused tests first failed because the constructor was absent. Synthetic tests cover both combat directions, multiple blockers, phasing, leave and return, source incarnation, one grouped trigger, clone isolation, and registration validation. A public DSL declaration test compiles and validates. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Damage Source Controller Event Binding

`DealsDamageToPlayer(This or AttachedPermanent, player).WithSourceController(controller)` captures the damage source's controller from the final positive damage event. It composes with `ToController()` and optional `WithAmount(amount)` in battlefield `WhenEvent` and `WhenEventModes` abilities. The capture is an instance-local event participant, not a target. The adapter uses the existing damage event, attachment check, pending trigger, and player-binding frame; no damage engine contract changed. Registration rejects missing, duplicate, shared, unnamed, or out-of-scope bindings and use on other damage patterns. No production card changed.

The synthetic tests first failed because `WithSourceController` was absent. They cover full prevention, phasing, event-time control, source departure and return, stack copies, game clones, and declaration validation. A public DSL test validates the example. Backfire's complete text was checked against `data/LEG.json`: its "to you" filter, attached creature controller, and damage action can now be expressed, but its production declaration remains on the legacy path. The Legends inventory and JSON gap record still describe card wiring and remain open. The legacy `AttachedToDealsDamageToController` caller in Legends and `WheneverEnchantedPermanentDealsDamageToPlayerTrigger` caller in Jumpstart remain, so both exports stay. The next dependency-ready work is a separate E event participant or P payment output after a fresh caller audit.

Focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. No production card, legacy export, inventory, or JSON gap record changed.

## Current Slice: Damage to Trigger Controller Filter

`DealsDamageToPlayer(This, player).ToController()` and the `AttachedPermanent` form match positive damage to the battlefield trigger controller at event time. Optional `.WithAmount(amount)` and the player binding retain the final event participants in the pending trigger frame. The filter reuses `EventTargetIsController`, the shared damage event, attachment condition, and existing trigger capture. It adds no damage executor or production card migration. Registration rejects incompatible opponent/combat patterns and unsupported source references. The public DSL declaration compiles.

Synthetic tests first failed because `ToController` was absent. They cover other-player damage, full prevention, event-time control changes, phasing, source departure and return, amount/player capture, stack copies, clone isolation, and declaration validation. Backfire's complete Oracle text was checked in `data/LEG.json`. Its attached creature controller capture, damage action, and card wiring remain open, so the Legends gap record is unchanged. Curious Obsession in Jumpstart retains its legacy attached combat-damage trigger. A fresh search found one legacy `AttachedToDealsDamageToController` caller and one `WheneverEnchantedPermanentDealsDamageToPlayerTrigger` caller; both exports remain. The next dependency-ready E slice is an event-time attached permanent controller binding, after a fresh audit of all callers.

Focused engine and public DSL tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The older missing-package and C metadata blockers recorded below did not recur in this workspace.

## Current Slice: Attached Permanent Damage to Player Event

`DealsDamageToPlayer(AttachedPermanent, player).WithAmount(amount)` is a typed battlefield event pattern for positive damage from the currently attached permanent to a player. The player is an event participant, and the optional amount is the final amount after prevention. The pattern reuses the shared attachment condition, damage event, pending trigger, and binding frame. It also retains `DealsDamageToPlayer(This, player)`. Registration accepts only `This` or `AttachedPermanent` as the source and rejects invalid bindings. No production card changed. Backfire's complete text was checked in `data/LEG.json`; its "to you" filter, creature controller recipient, damage action, and card wiring remain separate. The Legends gap record stays open. The next dependency-ready E work is the "to you" filter or another event participant after a fresh audit.

The focused tests first failed at compilation because the constructor accepted only `This`. Synthetic tests cover player versus permanent recipients, unrelated sources, prevention, event-time player and amount, attachment change, source return, phasing, stack copying, game cloning, and registration. A public DSL declaration validates the documented form. Existing production callers remain on their current declarations.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `DealsDamageToPlayer` call, one Jumpstart legacy attached-damage-to-player trigger call, and Backfire's legacy trigger in Legends. Both cards remain unwired.

## Current Slice: Attached Permanent Damage Event

`DealsDamage(source PermanentRef, amount *NumberBinding) EventPattern` now accepts `AttachedPermanent` as well as its existing `This` source. On a battlefield Aura or Equipment, it matches final positive damage from the currently attached permanent to any recipient and saves the actual event amount in the trigger frame. It reuses the shared damage event, attachment condition, trigger, and binding services. Registration rejects other source references and invalid bindings. No production card or legacy export changed. Spirit Link's complete text was checked in `data/LEG.json`; its production declaration and the Legends gap record remain open for card migration. The next E work is an attached-permanent damage-to-player participant or another filtered event after a fresh audit.

Synthetic contract tests first failed at compilation because `DealsDamage` accepted only `This`. They cover player and permanent recipients, full prevention, unrelated sources, phasing, changed attachments, source departure and return, event-time controller and amount, stack-copy and game-clone isolation, and registration validation. A public DSL declaration test compiles the reference form. A fresh search found two production `DealsDamage(This, ...)` calls and no production `DealsDamage(AttachedPermanent, ...)` call; both existing callers remain compatible.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff contains only this adapter, synthetic and public tests, and supporting documentation. No gap record changed because Spirit Link remains unwired.

## Current Slice: Announced X Permanent Target Sets

`TargetPermanentsExactlyX(name, filter, options...)` and `TargetPermanentsUpToX(name, filter, options...)` declare typed permanent sets whose target count follows announced X. The underlying `targetDeclaration` now forwards `BoundsForX` to the existing target-count service. Spell and activation announcements check counts and distinct legal targets before costs, and stack-aligned target specifications retain group identity for copying and resolution. `ForEach` can consume the legal set. The typed names avoid the legacy `TargetXPermanents` export collision; that export and all production callers remain. The adapter uses the current target and payment paths and adds no new selection or action executor. A failing activation test also found that `ManaCost.IsZero` dropped a pure `{X}` cost before payment; the shared cost check now retains it.

The focused contract tests first failed because the constructors were absent. They cover exact and up-to counts, X=0, duplicate and excess targets, invalid announcement without payment, activation cost rollback, phased and returned targets, clone and stack-copy isolation, and declaration checks. Candelabra of Tawnos and Word of Binding were checked against `data/ATQ.json` and `data/4ED.json`. Their production wiring remains; neither card or its gap record is complete. Per-target payment and divided damage remain separate T work. The next dependency-ready slice is divided-damage announcement or another P payment result, after a fresh caller audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. A fresh search found four production calls to legacy X-target constructors in Fourth Edition and Legends and no production call to the new typed exports. No card or JSON gap record changed.

## Previous Slice: Player-or-Planeswalker Damage Target

`TargetPlayerOrPlaneswalkerDamage(name string) *DamageTargetBinding` declares one target player or planeswalker. It reuses the established legacy target eligibility rule and typed damage recipient binding, so creatures are excluded at announcement and a phased or returned planeswalker is absent at resolution. It works with `WithTargets` in spells, activations, and event triggers and can feed `DealDamageFrom` or `PreventNextDamage`. The broader any-target declaration remains available. No production card changed.

The focused contract test first failed because the constructor was absent. It now covers player and planeswalker selection, creature exclusion, loyalty damage, phasing, leave and return, clone isolation, missing declaration rejection, and empty label rejection. Land's Edge, Spiteful Prankster, and Hungry Flames were checked against `data/LEG.json` and `data/JMP.json`; their other abilities and production wiring remain. Their gap records stay open. A fresh search found three production `TargetPlayerOrPlaneswalker` callers and no production caller of the new adapter. The legacy export remains. The next dependency-ready slice is T recorded target-group boundaries or another P payment output after a fresh audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role checks, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass.

## Current Slice: Singular Paid Discard Card

`DiscardCardCostInto(*CardBinding)` saves the one chosen discarded card in its
new graveyard visit during committed total-cost payment. `FromCard` and
`IfCardHasType` can read it in the resolving spell or activation. A replacement
to the library gives an absent card reference, as does a later zone move.
The existing locked choice, shared discard replacement action, and
instance-local stack capture remain in use. Registration checks nil and
duplicate bindings and a dominating payment producer. `DiscardCost(n)` and
unmigrated production callers remain compatible.

Land's Edge's complete text was checked in `data/LEG.json`. Its land-type
branch can use this adapter for an ordinary graveyard discard, but its
player-or-planeswalker target, any-player activation, and card wiring remain
separate. The Legends gap record remains open. Random and name-constrained
discard costs and paid-card snapshots across other destination zones are
outside this singular capture. The next dependency-ready work is a typed
player-or-planeswalker damage target or another P payment output after a fresh
caller and rules audit.

The focused contract test first failed because the constructor was absent.
It covers activation, spell additional cost, stack copy and clone isolation,
graveyard-visit expiry, hidden-zone replacement, prevented payment rollback,
and declaration validation. A public DSL declaration compiles and validates.
A fresh search found nine production `DiscardCost` call sites and no
`DiscardCardCostInto` production caller. Focused tests, `go test
./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests,
`go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and
`git diff --check` pass. No production card or JSON gap record changed.

## Current Slice: Chosen Cost-Discard Replacement

`DiscardCost(n)` now submits each chosen card as a cost-origin `DiscardAction`
through the shared replacement pipeline during isolated payment validation and
live commit. A general discard replacement can change the destination; the
completed discard emits one event. The effect-only library replacement retains
its effect-only condition. The locked choice, source ID, and existing
transaction path remain in use. `PlayerDiscardByCost(Player, cardID, sourceID)`
is the engine entry point; the author signature stays `DiscardCost(n int)` in
activation costs and spell additional costs. No production card changed.

The contract tests first failed because cost discard bypassed the replacement
action. Focused tests cover replacement destination, replacement recheck and
prevention during isolated validation, other-cost rollback at validation,
effect-only exclusion, discard event count, and clone isolation. Land's Edge's
complete text was checked in `data/LEG.json`; paid-card type capture and its
player-or-planeswalker target still need separate slices. The Legends gap
record and all production wiring remain open. Existing legacy exports remain.
Nine production `DiscardCost` call sites remain. Random and name-constrained
discard costs still use their existing paths. The
next dependency-ready slice is a paid-discard characteristic binding, with
CR 701.9c hidden-zone behavior and instance-local payment capture.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration
and closed-role tests, `go build ./...`, `go vet ./...`, `make test`, `make lint`
(zero issues), and `git diff --check` pass.

## Current Slice: Unique Highest Life State Condition

`HasUniqueHighestLife() StateCondition` checks the current life totals of all
players and is true only for one strict leader. It uses the existing pure state
condition path in `ActivationIf`, `IfState`, `WhenState`, `WithInterveningIf`, and
`While`. It has no player-result binding and changes no production card. Tests
first failed because the constructor was absent. Focused tests cover a tie,
either controller, state changes, game-clone isolation, inspection, and
registration in an intervening-if trigger. Ghazbán Ogre's complete text was
checked against `data/ARN.json`; its winning-player binding, indefinite control
effect, and production wiring remain separate, so its gap record stays open.
No production caller uses the new condition. The existing control and discard
exports remain. Focused `TestUniqueHighestLife` tests, `go test ./pkg/mage/...`
with `-short -count=1`, the DSL registration and closed-role tests, `go build
./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff
--check` pass. The next dependency-ready slice is the cost-discard replacement
prerequisite or another independently scoped S condition after a fresh audit.

The next paid-discard slice has a verified prerequisite: `DiscardCost` uses
`PlayerDiscard`, which bypasses the replacement path used by effect discards.
CR 701.9c also makes discarded-card characteristics undefined when a
replacement moves the card to a hidden zone without revealing it. A paid-card
type adapter must wait for an exact transactional cost-discard replacement
contract. Existing discard costs and callers remain unchanged.

## Current Slice: Locked Discard Payment Choice

`DiscardCost(n)` now retains the payer's chosen hand-card IDs in the total-cost
transaction. Validation uses those IDs on the isolated game, and commit
discards the same cards without a second choice. It rejects absent or invalid
choices, checks stale choices before paying mana or life, and leaves the cost
declaration reusable. Direct legacy `Pay` calls keep their existing choice
path. This engine prerequisite adds no typed paid-card result, characteristic
snapshot, or production card wiring.

Land's Edge's complete text was checked in `data/LEG.json`, its Legends
inventory and JSON gap record, and the discard and cost rules were checked in
the indexed comprehensive rules. A typed paid-discard characteristic binding
is the next dependency-ready slice; its player-or-planeswalker damage target
and card migration remain separate. The Legends gap record stays blocked.
Existing production callers and legacy exports remain unchanged.

Contract tests first failed because commit selected a different card after
the validated choice left the hand. Focused tests now cover one choice across
validation and commit, a stale card, an invalid out-of-hand choice, and reuse
of a cost declaration. This slice uses the existing payment transaction and
discard event path.

`go test ./pkg/mage -run '^TestDiscardPayment' -count=1`,
`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role
run, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues),
and `git diff --check` pass. The full suite exposed a test-player duplicate-ID
choice for two same-name cards; the harness now returns distinct matching
cards, and the Zombie Infestation integration test passes. A fresh search
found nine production `DiscardCost` calls; they remain compatible and the
legacy export remains. No JSON gap record was closed.

## Current Slice: Controller Hand-Size State Condition

`ControllerHandSizeIs(int) StateCondition` compares the ability controller's
current hand size with an exact nonnegative count. It uses `GameReader.GetPlayer`
and the existing pure state-condition and activation paths. `ActivationIf`
checks it before payment. It also works in `IfState`, `WhenState`,
`WithInterveningIf`, and `While`. A missing controller does not match.
Registration rejects negative counts. No card or legacy export changed.

Library of Alexandria's complete Oracle text was checked in `data/ARN.json`.
This slice supplies its exact-seven activation restriction; its production
definition still uses a legacy callback and needs separate card wiring. A
fresh search found no production `ControllerHandSizeIs` caller and one legacy
hand-size callback in `cards/arabian/lands.go`. No gap record was closed.
Other hand-size arithmetic and paid-discard card characteristics remain
separate work. The next dependency-ready slice is a P paid-discard
characteristic capture for Land's Edge, after a fresh requirements audit.

Focused tests first failed because the constructor was absent. They cover
six, seven, and eight cards, current controller distinct from owner, failed
activation without payment, an accepted activation, source absence for a pure
state check, a missing controller, clone isolation, inspection, public DSL
registration, and invalid declaration rejection. Focused tests, short engine
and repository suites, DSL registration and closed-role tests, `go build ./...`,
`go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check`
pass.

## Current Slice: Singular Paid Sacrifice Card

`SacrificeMatchingCostWithCard(filter, text, card)` saves the card of one
chosen, sacrificed permanent in its new graveyard visit. `FromCard(card)` can
read that card in the resolving spell or activation. A token gives an absent
card reference, and any later zone move invalidates the visit. The existing
locked sacrifice choice and prepared payment transaction enforce the chosen
battlefield incarnation. Card captures now travel in stack frames, copies,
and game clones; a failed payment clears the capture. This slice adds no
general paid-permanent LKI, collection payment, or card wiring.

CR 400.7j permits the effect of a spell or ability to find a card moved to a
public zone by its cost. The Legends Sword of the Ages record requires cards
of multiple sacrificed creatures and its source, so it remains blocked on a
collection payment and linked exile. Existing production callers and legacy
exports remain. The next dependency-ready P work is a paid-object set with
pre-sacrifice characteristics, or a separate paid-discard card capture after
a fresh audit. No gap record is closed by this singular adapter.

Focused tests failed first because the constructor was absent. They cover
spell and activation payment, copy and clone isolation, the graveyard visit
boundary, phasing and returned choices, rollback on failed payment, and
registration validation. The public DSL example compiles and validates.
A fresh search found no production use of the new constructor and seven
production card files using related sacrifice-cost constructors. Their
declarations and complete card audits remain separate work. The focused tests,
short engine and repository suites, DSL registration and closed-role tests,
`go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and
`git diff --check` pass.

## Current Slice: Source-Inclusive Sacrifice Payment Stats

`SacrificeCreatureIncludingSourceCostWithStats(power, toughness)` captures the
chosen creature's power and toughness before a prepared activation or spell
additional cost sacrifices it. The source is eligible when it is a creature.
It uses the existing locked-choice transaction and instance-local paid-number
frame; no resolving action or global last-sacrificed slot supplies the values.
The legacy cost constructors and production callers remain unchanged.

Life Chisel's complete Oracle text in `data/LEG.json` requires its animated
source to be a legal sacrifice choice. This adapter supplies that payment
case, but the card still needs production wiring and its upkeep restriction.
Diamond Valley and Fling already have a source-excluding paid-stats cost for
their printed costs. No gap record is closed by this adapter alone. Focused
tests first failed at compile time because the constructor was absent; they
cover a source sacrificed with modified toughness, source return before
commit, copy and clone isolation, and registration errors. The next
dependency-ready P work is a general paid-object identity or a different
payment output after a fresh audit.

`go test ./pkg/mage -run '^TestPaidSourceSacrificeStats' -count=1`,
`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role
run, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass. Lint
reported zero issues. No production card changed.

## Current Slice: Source Attacked or Blocked This Combat

`SourceAttackedOrBlockedThisCombatCondition() StateCondition` reads declared attack or first-block history for the current battlefield source incarnation. It works in pure permanent-source state contexts through `IfState`, `While`, `WhenState`, `WithInterveningIf`, and `ActivationIf`. The existing event and turn tracker services record declarations. A beginning-of-combat event clears the combat record; the end-of-combat step keeps it through trigger resolution and clears it after priority. The separate turn record remains until cleanup. Phasing makes the query false while phased out, and leaving and returning cannot reuse the record. Game clones copy combat history independently. This slice does not add a target-bound condition, a refill counter limit, or card wiring.

Clockwork Beast's complete Oracle text was checked in `data/LEA.json` and `data/4ED.json`, along with its Limited inventory and JSON gap record. This condition supplies its end-of-combat history requirement. Its entry counters, refill-to-seven limit, and production declaration still need a separate migration, so the gap record remains blocked. Legacy callers and exports remain. The next dependency-ready work is a P paid-object snapshot or a separate S history predicate after a fresh audit.

Focused contract tests first failed because the constructor was absent. They cover attacker and blocker declarations, empty history, phasing, leave and return, a second combat in the same turn, end-of-combat trigger timing, turn-history preservation, clone isolation, inspection, and registration rejection for a spell source. No production card or JSON gap record changed.

`go test ./pkg/mage -run '^TestSourceAttackedOrBlockedThisCombat' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The public DSL example has a registration test. A fresh caller search found no production use of the new condition; Clockwork Beast still needs production wiring.

## Current Slice: Source Attacked or Blocked This Turn

`SourceAttackedOrBlockedThisTurnCondition() StateCondition` is a pure battlefield-source condition for `IfState`, `While`, `WhenState`, `WithInterveningIf`, and `ActivationIf`. Declared attacker and first declared blocker events save the source incarnation in the existing turn tracker. The query requires the same current incarnation; phasing makes it false until phase-in, leaving and returning makes it false, and turn reset clears it. Tracker cloning copies the record independently. The existing event dispatch and state-condition services supply the adapter. It does not supply "this combat" history, a shuffle action, or card wiring. Inferno Hellion's complete Oracle text was checked in `data/JMP.json` and its Jumpstart inventory and gap record. Its shuffle-to-owner's-library operation and production migration remain open, so the gap record remains blocked. Legacy caller exports remain.

Contract tests first failed because the constructor was absent. Focused tests cover attack and block declarations, absence before declaration, phasing, leave and return, turn reset, game clone isolation, trigger resolution, inspection, and registration rejection on a spell source. The next dependency-ready work is a P payment output or a distinct combat-scoped history prerequisite after a fresh audit.

`go test ./pkg/mage -run '^TestSourceAttackedOrBlockedThisTurn' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production caller of the new condition; Inferno Hellion still uses a direct tracker query, and the Limited Clockwork Beast still uses the legacy `SourceAttackedOrBlockedThisTurn` trigger condition. No production card or JSON gap record changed.

## Current Slice: Pure Your-Turn State Condition

`IsYourTurn() StateCondition` compares the current active player with the stored controller in the condition context. It works with `IfState` in spell, activation, and trigger bodies and with the existing pure condition combinators. It reuses `GameReader.ActivePlayerObj` and the existing state-condition evaluator. It adds no turn subsystem, cast restriction, payment, or card wiring. The source can be absent because the turn question concerns the player. Rapier Wit's complete text was checked in `data/SOS.json` and its Secrets of Strixhaven inventory and gap record. This condition supplies its resolution-time turn check; target legality, tap, stun counter, draw, and production card wiring remain a separate migration. The gap record stays open.

Contract tests first failed because `IsYourTurn` was undefined. They cover either active player, stored-controller identity, an absent spell source, clone isolation, inspection, and registration. Existing `OnlyOnYourTurn` and production callers remain. The next dependency-ready work is a P payment output or another small S state predicate after a fresh audit.

`go test ./pkg/mage -run '^TestIsYourTurn' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` (zero issues) pass. The reference example compiles. No production card or gap record changed.

## Current Slice: Source Combat Damage to Player Event

`DealsCombatDamageToPlayer(This, player).WithAmount(amount)` is a typed battlefield event pattern for `WhenEvent` and `WhenEventModes`. It reuses the shared positive damage event, its combat flag, and instance-local player and optional amount captures. The amount is final after prevention. It does not add a damage executor, grouped damage pattern, delayed subscription, or card wiring. Initiate's Companion's complete text was checked in `data/JMP.json` and the Jumpstart inventory and gap record. This pattern supplies its combat-only event requirement; its production declaration still needs a separate card audit and migration, so the gap record stays blocked. Legacy callers and exports remain.

Contract tests first failed because the constructor was absent. They now cover combat versus noncombat events, player versus permanent recipients, final amount after partial prevention, full prevention, phasing, source leave and return, stack copying, game clone isolation, inspection, and invalid declarations. The next dependency-ready work is another filtered E event participant or a P/T prerequisite after a fresh audit.

`go test ./pkg/mage -run '^TestSourceCombatDamageToPlayer' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A public DSL declaration test compiles and validates the reference example. A fresh search found one production `WheneverDealsCombatDamageToPlayerTrigger` caller and two `WheneverPermanentDealsCombatDamageToPlayerTrigger` callers; the latter need a different source-selection pattern. Initiate's Companion and other card wiring remain open. No production card, legacy export, or gap record changed.

## Current Slice: Permanent Receives Damage Event

`ReceivesDamage(This, amount).WithSourceController(controller)` is a typed battlefield event pattern for `WhenEvent` and `WhenEventModes`. The amount is required; the source-controller capture is optional. It matches each positive damage event whose final permanent recipient is the source and saves the final amount and captured damage source controller in that trigger instance. It reuses the shared damage event, trigger, and binding services. No new damage executor, target, or replacement rule was added.

Belltower Sphinx's complete Oracle text was checked in `data/JMP.json` and its Jumpstart inventory and gap record. This adapter supplies its receives-damage event, amount, and source-controller requirements. Its production declaration and complete migration audit remain open, so the gap record stays blocked. Legacy callers and exports remain unchanged. Focused tests failed first because the constructor was absent. They cover prevention, phasing, source and recipient departure and return, owner/controller separation, simultaneous damage, stack copies, game clones, inspection, and invalid declarations. The next dependency-ready slice is another E event participant or a P/T prerequisite after a fresh audit.

`go test ./pkg/mage -run '^TestReceivesDamage' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff contains only this adapter, its synthetic tests, and documentation. No production card, legacy export, gap record, or unrelated untracked file changed.


## Current Slice: Damage Event Source Controller

`core.GameEvent.SourceControllerID` now records the controller in the captured damage source on each positive `EvtDamageDealt`, for player and permanent recipients. The shared damage executor supplies this event-time identity after replacement and prevention. A source that leaves and returns under another controller cannot change it. No typed event adapter or card declaration changed in this prerequisite slice. Belltower Sphinx's complete text was checked in `data/JMP.json` and the Jumpstart inventory and gap record; its `ReceivesDamage` event pattern, amount and source-controller bindings, and card wiring remain open. The gap record remains blocked. Production callers and legacy exports remain unchanged.

Contract tests failed first because the field was absent. They now cover source departure and return, spell controller distinct from owner, full prevention, and a simultaneous batch with both player and permanent recipients. The next dependency-ready slice is a typed `ReceivesDamage(This)` event pattern with event-time amount and source-controller bindings.

`go test ./pkg/mage -run '^TestDamageEvent' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The post-lint diff contains only this engine prerequisite, its tests, and documentation. No production card, legacy export, or untracked inventory changed.

## Current Slice: Controller Receives Damage Event

`ControllerIsDealtDamage(*NumberBinding) EventPattern` matches a positive `EvtDamageDealt` whose final player recipient is the battlefield source's controller at event time. It saves the final amount for `FromNumber` in `WhenEvent` and `WhenEventModes`. The existing damage, replacement, prevention, trigger, and resolution-frame services supply the behavior; no new damage executor or result slot is needed. Registration rejects nil, unnamed, or out-of-scope amount bindings and use on a spell source. The damaged player is not a target.

Living Artifact's complete text was checked in `data/4ED.json` and the Limited inventory and gap record. This slice supplies its missing controller-damage event requirement. Both production definitions still use legacy behavior and need separate wiring and full card audits, so the card gap remains open. The new pattern has no production caller. Existing `WhenDamageDealtToThisTrigger` and other legacy exports remain available. Belltower Sphinx needs a different recipient pattern with damage-source controller LKI and is not covered here. The next dependency-ready E slice is that source-controller participant prerequisite, or a P payment output after a fresh audit.

Synthetic tests first failed because the constructor was absent. Focused tests cover actual post-prevention amount, redirection to a permanent, zero and prevented damage, controller change, phasing, source departure and return, simultaneous events, modal triggers, stack copies, game clones, inspection through registration, and binding scope. The shared event contract is documented in `docs/engine-rules-contracts.md`.

`go test ./pkg/mage -run '^TestControllerIsDealtDamage' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. No production card or untracked inventory was changed.

## Current Slice: Source Targeted by a Spell

`BecomesTargetOfSpell(This) EventPattern` works in `WhenEvent` and `WhenEventModes` for battlefield sources. It reuses `EvtBecomesTarget` and the existing source-target and spell-event conditions. It has no event participant, target, or stack-object binding. The event service already emits after a successful target announcement; the queued trigger keeps its event-time controller and source incarnation. This slice adds no replacement or payment path.

Departed Deckhand's complete text was checked in `data/JMP.json`. Its first clause needs this spell-only filter; its blocking restriction, activated ability, full migration audit, and production wiring remain separate. The Jumpstart gap record remains blocked. The production card still uses `NewTriggered(EvtBecomesTarget, ...)`, so the legacy surface stays available. Kira's first-time spell-or-ability trigger and stack-object counter remain separate work.

Focused tests failed first because the constructor was absent. Synthetic tests now cover actual spell casting, spell versus ability events, another target, source sacrifice, absence, phasing, source departure and return, game-clone isolation, inspection, registration rejection for a nonbattlefield source, and compile-time rejection of a player source. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The next dependency-ready slice is a T target bound or another E participant after a fresh audit.

## Current Slice: Opponent-Controlled Creature Death Event

`OpponentCreatureDies()` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It matches each creature that moves from the battlefield to a graveyard while an opponent of the trigger source controller controls it. The existing zone-change event supplies the former controller and pre-departure creature type. `.WithController(*PlayerBinding)` saves that player in the trigger instance; it is not a target. The pattern composes with existing death filters and captures. It adds no new zone event, result frame, or card policy.

Sangromancer's full text was checked in `data/JMP.json` and the Jumpstart gap record. This slice supplies its opponent-creature-death filter, and the preceding slice supplies its opponent-discard event. Its optional life gain, complete card audit, and production wiring remain separate. Eternal Thirst also needs granted-trigger authoring and card wiring. No production card or gap record changed. The gap record remains open. A fresh search found no production `OpponentCreatureDies` caller; Sangromancer and Eternal Thirst still use legacy trigger definitions, which remain available.

Contract tests first failed because the constructor was absent. They cover owner/controller separation, control changes, tokens and simultaneous deaths, source death, phasing, nondeath exclusion, event-time controller capture, stack copy, game clone, source departure and return, inspection, and invalid binding declarations. Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The next dependency-ready work is a T target-group boundary or another E event participant after a fresh audit.

## Current Slice: Opponent Discard Event Participant

`OpponentDiscardsCard(*PlayerBinding) EventPattern` matches each completed card discard by an opponent of the battlefield source controller. It captures the discarding player in the trigger instance for `WhenEvent` and `WhenEventModes`; `FromPlayer` reads that value at resolution. It reuses `EvtDiscard`, the existing opponent condition, and pending-trigger player capture. The event participant is not a target. The pattern does not capture the discarded card or create a general discard-result binding. Go rejects a non-player binding; registration rejects nil, unnamed, unbound, and out-of-scope bindings.

Fell Specter's complete text was checked in `data/JMP.json` and the Jumpstart gap record. Its opponent-discard clause is expressible, but its ETB choice and production wiring remain separate. Sangromancer and Megrim are other possible callers. No card or gap record changed. Synthetic tests failed first because the constructor was absent. They cover one trigger per card, own versus opposing discard, source control changes, replacement to library, impossible discard, phasing, source leave and return, stack copy, game clone, inspection, and declaration validation. Legacy discard-trigger exports remain for production callers. The next dependency-ready work is a T target-group boundary or another E participant after a fresh audit.

`go test ./pkg/mage -run '^TestOpponentDiscardsCard' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found two production `WheneverOpponentDiscardsTrigger` calls in Jumpstart; the legacy export remains. The older missing-package and lint blockers recorded below do not occur in this checkout.

## Current Slice: Land-Entry Actor Identity

`GameEvent.ActorID` records the player who causes a battlefield entry separately from the entering permanent's controller in `PlayerID`. `Game.PutOnBattlefieldBy(card, controller, actor)` supplies an explicit actor. Land play, permanent spell resolution, and engine effect paths with a known acting player now supply that field. The older `PutOnBattlefield` remains available for unmigrated callers and leaves actor unknown. `LandEnters().WithActor(*PlayerBinding)` uses the shared zone-entry event in `WhenEvent` and `WhenEventModes` on battlefield permanents. It composes with `OfSubtype` and `WithController`, matches only an entry with a known actor, and saves that player in the pending trigger. It is not a target. Registration rejects nil, unnamed, repeated, unbound, and actor/controller alias bindings; Go rejects a permanent binding at the method call.

Thelon's Chant and Tourach's Chant in `data/FEM.json` require the player who puts a Swamp or Forest onto the battlefield. This slice supplies that event participant for entries whose actor is recorded. Their optional counter-placement branch, production wiring, and complete card audit remain open. A fresh search found 65 direct production `PutOnBattlefield` calls and two production `WheneverLandEntersBattlefieldTrigger` calls; no production `LandEnters` call. Direct production calls to `PutOnBattlefield` cannot supply actor identity and remain a separate caller migration; the method is retained. No card or Fallen Empires gap record changed. Synthetic tests first failed because `WithActor` and `PutOnBattlefieldBy` were absent. They cover actor/controller/owner distinction, unknown actor, land play, subtype filtering, land return, stack copying, game cloning, and registration errors. The next dependency-ready work is the typed optional counter-placement choice for the Chant family or another E/P prerequisite after a fresh audit.

Focused engine and DSL tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, `go test ./pkg/mage/dsl -run 'TestRegisteredDefinitions|TestClosedExpressionRoles' -count=1`, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. The diff was reviewed after lint. No unrelated tracked or untracked file changed.

## Current Slice: Land-Entry Subtype Filter

`LandEnters().OfSubtype(string)` filters the existing land-entry event by one land subtype at detection. It composes with `.WithController` in `WhenEvent` and `WhenEventModes` on battlefield permanents. The subtype is checked after entry processing through the shared event condition, and a queued trigger keeps its event-time result if the land later leaves. Registration rejects empty, invalid, and repeated land subtypes. The sealed pattern remains an event role; no new event or zone executor is added.

Thelon's Chant and Tourach's Chant require Swamp and Forest entry filters, respectively, as confirmed in `data/FEM.json`. Their text refers to the player who puts the land onto the battlefield. Entry actor identity is supplied by the subsequent slice above for explicit entries; both cards still need their optional counter-payment branch. Their production definitions and Fallen Empires gap records remain open. Synthetic tests first failed because `OfSubtype` was absent. They cover matching and nonmatching subtypes, owner/controller distinction, land departure before resolution, game clone isolation, registration errors, and DSL inspection. Existing production callers and legacy exports remain unchanged.

Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass. Lint reports zero issues. No production card or gap record changed.

## Current Slice: Land-Entry Controller Participant

`LandEnters().WithController(binding)` is a battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It checks the entering permanent's land type after entry processing and saves its computed controller in the pending trigger. The binding is instance-local, is not a target, and survives later source or land departure, copying, and game cloning. The pattern uses the existing zone-entry event and player-binding frame; no new zone subsystem is added. Registration rejects nil, unnamed, duplicate, and unbound player bindings. The sealed role rejects a permanent binding at compile time.

Ankh of Mishra's full Oracle text in `data/4ED.json` requires damage to the entering land's controller. This slice supplies its event and controller participant; typed singular damage is already available. Its production definition still calls `WheneverLandEntersBattlefieldTrigger` and remains unchanged, so the Limited gap record stays open. Thelon's Chant and Tourach's Chant say a player *puts* a land onto the battlefield; this adapter does not claim that actor identity or their counter-payment branch. Their gap record remains open. Focused tests first failed because `LandEnters` was absent. They cover either controller distinct from owner, nonland entry, phased source, source and land return, stack copying, clone isolation, inspection, registration, and compile-time binding roles. No production card or legacy export changed. The next dependency-ready E slice is the entry actor identity prerequisite or another filtered entry participant after a fresh audit.

Focused tests, the short engine and repository suites, DSL registration and closed-role tests, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh caller search found no production `LandEnters` use and two production `WheneverLandEntersBattlefieldTrigger` uses. Card wiring remains separate.

## Current Slice: Resolution-Time Mana-Value Set Threshold

`ManaValueAtMost(PermanentSet, ResolutionValue) PermanentSet` filters a selected battlefield set by mana value at use time. `DestroyEach` consumes the result in one simultaneous destruction batch. The adapter reads `FromNumber` through the current resolution frame, so an announced X life payment supplies its own threshold. It uses the existing permanent mana-value and batch-destruction services. Registration rejects nil inputs and unbound number reads. Existing production callers and legacy exports remain unchanged.

Vicious Rivalry's complete text in `data/SOS.json` requires an additional cost of X life and destruction of all artifacts and creatures with mana value X or less. `LifePayXCostInto` already supplies the cost; this slice supplies the missing collection threshold. Production card wiring and a full card audit remain separate, so its Secrets of Strixhaven gap stays open. Synthetic tests first failed because `ManaValueAtMost` was absent. They cover the zero-cost and exact-threshold members, artifact/creature union, regeneration, phasing, clone isolation, and declaration errors. The next dependency-ready work is another E event participant or a separate P/B collection capability after a fresh audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production `ManaValueAtMost` caller. No card or JSON gap record changed.

## Current Slice: Artifact Ability Activation Without Tap Cost

`ArtifactAbilityActivatedWithoutTapCost().AttachedToSource().WithArtifact(binding).WithController(binding)` is a typed E event pattern for `WhenEvent` and `WhenEventModes` on battlefield permanents. The attachment filter and two captures are optional. The pattern uses the successful `EvtAbilityActivated` event, the artifact type at detection, and instance-local trigger bindings. It captures the artifact controller from departure LKI when an activation sacrifices its source. The direct and automatic mana activation paths now mark their `{T}` cost in that event. The pattern has no target. Registration checks bindings and scope; the sealed binding roles reject crossed types.

The complete Artifact Possession and Haunting Wind text was checked in `data/ATQ.json`. This slice supplies their ability-activation event requirement; both cards still need production wiring and a complete migration audit. The Antiquities gap record remains blocked. Synthetic tests first failed because the constructor was absent. They cover `{T}` filtering in direct and planned mana activation, attachment, controller and artifact captures, sacrificed-source LKI, failed-payment rollback, stack copying, clone isolation, and declaration errors. Existing production callers and legacy exports remain unchanged. The next dependency-ready work is another E event participant or the P/B collection filter and simultaneous destruction prerequisite for Vicious Rivalry.

Focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh search found no production caller of the new pattern and three existing Antiquities `EvtAbilityActivated` trigger definitions. No production card or JSON gap record changed.

## Current Slice: Artifact Tap Event Participants

`ArtifactBecomesTapped().WithArtifact(binding).WithController(binding)` is a typed E event pattern for `WhenEvent` and `WhenEventModes` on battlefield permanents. The two captures are optional and independent. The pattern reuses `EvtTapped`, artifact type checks, pending-trigger capture, permanent-incarnation and player binding frames, and stack/game copying. It triggers once for an untapped-to-tapped artifact transition, including an opponent's artifact, and does not match nonartifacts, entering tapped, or a second tap. The controller is fixed at event time; a returned or phased-out artifact is absent from `From(binding)` at resolution. It declares no target. Registration rejects invalid and unbound captures, and the DSL type checks reject crossed binding roles.

Haunting Wind's complete text in `data/ATQ.json` also requires an artifact ability-activation event without a tap cost and damage to the captured controller. This slice supplies only its tap-event participant contract; that card and all production callers remain unchanged. Its Antiquities gap record remains blocked. Synthetic tests first failed because the constructor was missing. They cover type and controller filtering, repeated taps, source and artifact phasing, artifact return, event-time player capture, pending source return, stack copying, game cloning, inspection, scope, and invalid declarations. The ability-activation pattern was added in the subsequent slice above; the P/B collection filter and simultaneous destruction for Vicious Rivalry remain open.

Focused `TestArtifactBecomesTapped` tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), the reference example build, and `git diff --check` pass. A fresh production search found no `ArtifactBecomesTapped` caller; legacy tap-event exports remain in use. No production card changed.

## Current Slice: Announced X Life-Payment Binding

`LifePayXCostInto(paid)` is a typed contextual cost for an activated ability or a spell additional cost. It locks the announced X as a fixed life payment before the shared total-cost transaction validates and commits it. The committed amount enters the stack instance's number-binding frame for `FromNumber(paid)` at resolution. Zero and exact-life payments follow CR 119.4; an unpayable or failed total cost leaves the live life total and stack unchanged. Registration checks the binding and producer scope. Existing fixed `LifePayCostInto` and legacy callers remain.

The complete Vicious Rivalry text in `data/SOS.json` requires this announcement-time payment, then destruction of all artifacts and creatures with mana value X or less. This slice supplies the payment capture only; its collection filter, production wiring, and full card audit remain. The Secrets of Strixhaven gap record stays blocked. Synthetic tests first failed because the constructor was missing. They cover zero, exact-life, excess-life rejection, separate pending activations, copy and clone isolation, failed-cast rollback, and declaration errors. The next dependency-ready work is the collection filtering and simultaneous destruction prerequisite or another E event participant, after a fresh audit.

Focused `TestPaidXLife` tests, the short engine and repository suites, DSL registration and closed-role tests, `go build ./...`, `go vet ./...`, `make test`, `make lint` (zero issues), and `git diff --check` pass. A fresh production search found no `LifePayXCostInto` caller and ten existing life-cost references; card wiring remains separate. No production card changed.

## Current Slice: Opponent Land Tap Participant

`OpponentLandBecomesTapped().WithLand(*PermanentBinding)` saves the tapped opponent-controlled land's battlefield incarnation at detection for a later `From(binding)` read. The optional method composes with `OfSubtype` in `WhenEvent` and `WhenEventModes`. It uses the existing tap event, pending-trigger capture, permanent binding frame, and clone service. The land is not a target; a returned or phased-out incarnation is absent at resolution. Existing production callers and legacy exports remain unchanged.

The Lifetap and Lifeblood inventories confirm the opponent land tap event family; their complete text in `data/4ED.json` and `data/LEG.json` does not require a land participant. This binding extends the shared event contract without completing either unmigrated card. Synthetic tests first failed because `WithLand` was absent. They cover presence, return, phasing, source return with changed controller, stack copying, clone isolation, scope, and invalid declarations. No gap record changed. The next dependency-ready work is another E event participant or P variable payment, after a fresh audit.

Focused tests, the short engine and repository suites, DSL registration and closed-role tests, build, vet, `make test`, `make lint` (zero issues), the standalone DSL example build, and `git diff --check` pass. A fresh production search finds no `OpponentLandBecomesTapped` or `WithLand` caller, so card wiring remains separate.

## Current Slice: Source Becomes Tapped Event

`BecomesTapped(This) EventPattern` is available in `WhenEvent` and `WhenEventModes` on battlefield permanents. It reads the shared `EvtTapped` transition with `EventSourceIsSelf`; `TapPermanent` emits that event only on an untapped-to-tapped change. It has no event-participant binding. Normal pending triggers keep the event-time controller and source incarnation, including through source departure, return, stack copying, and game cloning. Registration rejects a non-battlefield source; the sealed `sourceRef` argument rejects a player reference at compile time.

City of Brass in `data/ARN.json` requires a source tap event, damage to its controller, and any-color mana production. `DealDamageFrom(This, You, Fixed(1))` and `AddManaOfAnyOneColor(Fixed(1))` already supply the latter two actions; this slice supplies the event. Its production definition and the Arabian inventory remain open pending card wiring and audit. No production card or JSON gap record changed. A fresh caller search found no production `BecomesTapped` use; legacy tap-event exports remain in use. The next dependency-ready E work is a typed event participant or filtered tap event after a fresh audit; P variable life payment is also open.

Synthetic contract tests first failed because `BecomesTapped` was absent. They cover source filtering, entry tapped, repeated taps, phasing, absence, departure and return, pending controller and incarnation, target announcement, stack copying, game cloning, inspection, registration, and compile-time role rejection. Focused tests, the short engine and repository suites, DSL registration and closed-role tests, build, vet, and `make test` pass. The first lint run found a shadowed Go built-in in a test; it was renamed and the second lint run passed with zero issues. `git diff --check` passes.

## Current Slice: Fixed Life-Payment Amount Binding

`LifePayCostInto(amount, paid)` is a typed fixed-life cost for `Activate`, `ActivateModes`, and spell additional costs. It uses the existing prepared total-cost transaction and writes the committed payment to an instance-local `NumberBinding` for `FromNumber` at resolution. Zero and exact-life payments follow CR 119.4. The existing `LifePayCost` and all production callers remain unchanged. Registration rejects negative amounts, nil or unnamed bindings, duplicate paid-life producers, and reads without a producer. Resolution-time `IfPaid` and `UnlessPaid` costs do not support this output; variable X life costs, prevention or replacement of life loss, and card wiring are outside this slice.

Synthetic contract tests first failed because the constructor was absent. They cover activation and spell payment, zero and exact-life boundaries, insufficient-life failure, two pending activations, stack copying, game cloning, and registration scope. The shared cost executor, prepared transaction, stack capture map, and life-payment boundary are the reuse points. The complete Vicious Rivalry requirement in `data/SOS.json` includes a variable life payment that sets X while casting; this fixed-cost adapter does not satisfy it. No gap record or production card changed. A fresh caller search found no production `LifePayCostInto` caller. The next dependency-ready P work is a typed variable life payment with announcement-time X and transaction rollback, after a fresh contract audit.

Focused `TestPaidLifeAmount` tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, and `make lint` pass. Lint reported zero issues. The final focused test and diff check were repeated after the last test edit.

## Current Slice: Saved-Card Type Branch

`IfCardHasType(CardRef, core.CardType, ResolvingEffect)` and `IfCardLacksType(CardRef, core.CardType, ResolvingEffect)` are typed resolving branches for a saved hand or graveyard card or a legal graveyard target. Both read current card types through the existing zone-visit reference. An absent card matches neither branch. Inspection and registration reject nil roles, invalid types, missing producers, and a binding that escapes a branch. The existing resolution frame, target legality, and clone services supply identity and isolation. Production cards and legacy exports are unchanged.

Sindbad's complete Oracle text was checked in `data/ARN.json`: its drawn card is revealed, then discarded if it is not a land card. The existing draw-card result, reveal, and saved-card discard adapters together with this branch supply those actions; Sindbad still needs separate production wiring and a complete migration audit. The Arabian inventory remains open, and no JSON gap record is changed. Synthetic contracts first failed because the constructors were absent. They cover land and nonland cards, empty draw, hand departure and return, game clone isolation, legal and returned graveyard targets, registration errors, and branch scope. The next dependency-ready work is another P payment output or a separate E event participant, after a fresh audit.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, the reference example compile, and `git diff --check` pass. A fresh production search found no `IfCardHasType` or `IfCardLacksType` caller; the existing Sindbad definition and legacy exports remain.

## Current Slice: Public Reveal of a Saved Card

`RevealCard(CardRef) ResolvingEffect` resolves a saved card at instruction time and records a public, copied reveal-time card snapshot. `Game.PublicReveals()` gives all observers the same ordered history without exposing mutable engine state. A saved draw must still have its original hand visit; an absent card, replaced draw, or card that left and returned produces no reveal. Revealing does not move the card. This is a resolving action for spells, activations, and triggers, not an ongoing revealed-card effect or a cost. The `dsl.RevealCard` export has a sealed `CardRef` input; inspection and registration use the existing card-binding dominance checks. Existing `RevealHand`, draw, discard, and production callers remain.

The focused tests first failed because `RevealCard` and `PublicReveals` were absent. Synthetic tests cover public owner and land characteristics, zone preservation, absent and stale references, empty draw, immutable returned history, game clone isolation, registration errors, and compile-time player-reference rejection. Sindbad's complete Oracle text in `data/ARN.json` also requires a land-card condition and conditional discard of the same saved card; its production wiring remains open. No card or gap record was changed. A fresh search found no production `RevealCard` caller. The next dependency-ready slice is a typed card-type predicate for a saved card in a resolving branch, followed by separate card wiring.

Focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. Lint reported zero issues.

## Current Slice: Saved Hand-Card Discard

`DiscardCard(CardRef) ResolvingEffect` discards a saved card only while its original hand visit remains current. It uses `PlayerDiscardByEffect`, so an effect-discard replacement can change the destination and the normal discard event still occurs. An absent reference, a card in another zone, or a card that leaves and returns to hand does nothing. The adapter works in spell, activation, and trigger bodies. The public `dsl.DiscardCard` alias and its compile-time `CardRef` role are new; existing `Discard(PlayerRef, ResolutionValue)`, `DiscardAtRandom`, and legacy callers remain. No production card changed.

The focused tests first failed because `DiscardCard` was absent. Synthetic tests cover exact drawn-card identity, no second discard, return-to-hand visit identity, owner distinct from controller, graveyard exclusion, replacement routing, clone isolation, branch scope, registration errors, inspection, and a player-reference compile error. Sindbad's full Oracle text in `data/ARN.json` also requires revealing the drawn card and checking whether it is a land before discarding it. Those capabilities and production wiring remain open; its Arabian inventory entry is not complete. No JSON gap record was changed. A fresh caller search found no production `dsl.DiscardCard` use. The next dependency-ready slice is a truthful public reveal contract for a saved card, including reveal visibility and lifetime, before exposing a typed adapter.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass.

## Current Slice: Singular Drawn-Card Result

`Draw(You, Fixed(1), DrawCardInto(binding))` saves the card from one completed draw in the current resolution frame. `FromCard(binding)` now resolves either a saved hand visit from this result or a saved graveyard visit from a death event. It does not follow a card that leaves and returns. Registration rejects a non-single fixed amount, duplicate or mixed result options, nil bindings, and reads without a producer. The adapter uses `PlayerDrawCard`, hand visits, and the existing card-binding frame. It works in spell, activation, and trigger resolving bodies. `DrawEach`, multi-card result collections, hand reveal, and card wiring remain open. `DiscardCard(FromCard(binding))` now supplies the singular hand-card discard.

Focused tests first failed because `DrawCardInto` was absent. They cover successful and failed draws, replacement, absent and legal targets, hand departure and return, stale-result clearing, stack copying, game clone isolation, and registration errors. Sindbad's Oracle text in `data/ARN.json` and Read the Runes in `data/JMP.json` were checked. Sindbad still needs reveal and conditional discard of the saved card. Read the Runes needs a collection result and per-card choices. Their gap records remain blocked. A fresh production search found no `DrawCardInto` caller; legacy draw callers remain. The next dependency-ready work is a hand-card consumer for reveal or discard, or a multi-card result collection after a fresh audit.

Focused tests with `-count=1`, short engine and repository tests, DSL registration and closed-role tests, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. No production card or JSON gap record changed.

## Current Slice: Hand Visit Identity for Draw Results

`BasePlayer.handVisit(cardID)` is an internal read of a card's current hand visit. `AddToHand` creates a visit, `RemoveFromHand` ends it, and `SetHand` preserves visits for cards that remain while assigning new visits to added cards. Game cloning retains current visits and copies the map on later hand changes. The shared `PlayerDrawCard` path therefore gives a completed draw a hand identity, while a replaced or empty-library draw gives none. This is the zone-identity prerequisite for a typed drawn-card result; no new author export or production card changed. The existing `Draw(PlayerRef, ResolutionValue, ...DrawResultOption)` and `DrawAmountInto(*NumberBinding)` signatures remain the supported count-only API in spells, activations, and triggers.

The contract tests first failed because `handVisit` did not exist. They cover departure and return, unchanged-hand restore, draw and replacement, discard, and clone isolation. Sindbad's complete Oracle text in `data/ARN.json` needs the actually drawn card for reveal and conditional discard. Read the Runes' complete text in `data/JMP.json` needs per-card choices. Both remain blocked by card-result and choice adapters plus card wiring; their gap records are unchanged. A fresh production search found no typed drawn-card result caller because no such export exists. The next dependency-ready P slice is a typed drawn-card result using the recorded hand visit, with inspection, frame, copy, and registration contracts. This prerequisite has no target, phasing, or replacement-ordering declaration of its own; it uses the existing draw service for those boundaries.

## Current Slice: Singular Draw Count Result

`Draw(PlayerRef, ResolutionValue, ...DrawResultOption)` now accepts `DrawAmountInto(*NumberBinding)`. The option saves the number of successful individual draws from that instruction in the resolution frame. Empty-library attempts, replaced draws, and absent recipients contribute zero. The two-argument form, `DrawEach`, and all legacy exports remain. The adapter reuses `PlayerDrawCard`, its replacement and event service, and the existing number-binding scope and copy machinery. It does not bind drawn card identities or count a collection draw.

The new contract tests first failed because the option did not exist. They cover partial and empty draws, replacement, absent recipients, stale-result clearing, declaration validation, target absence, stack copies, and game clone isolation. Read the Runes' complete text in `data/JMP.json` needs the count of cards drawn this way, but also needs per-card choice and discard-or-sacrifice actions. Sindbad needs the actual card identity. No production card was changed, and neither gap is closed. Existing production `DrawFor` and `DrawEach` calls remain; no production `DrawAmountInto` caller exists. The next dependency-ready P slice is a card-result binding for draws or discards, after auditing zone-visit identity and replacement behavior.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass.

## Current Slice: Resolving Tap Action Result

`Tap(PermanentRef, ...TapResultOption)` now accepts `TapAmountInto(*NumberBinding)` and keeps its one-argument form. The option writes 1 only when the referenced battlefield incarnation changes from untapped to tapped through the shared tap action; it writes 0 for an already tapped, absent, phased-out, or returned object. `IfPositive(ResolutionValue, ResolvingEffect)` branches on that saved result inside a resolving `Sequence`. Both are exported through `dsl`. This is an action result, not a tap-cost output or a reflexive trigger. The normal tap event, target binding and legality, resolution-frame number storage, stack copy, clone, inspection, and registration services are reused.

Energy Tap's complete Oracle text was checked in `data/LEG.json`. Its successful-tap condition is now expressible, but its production definition and test remain unchanged. Its Legends gap stays blocked pending card wiring and a full migration audit. A fresh search found no production `TapAmountInto` or `IfPositive` caller; existing `Tap` and cost exports remain. Synthetic tests first failed because the typed option and branch were absent. They cover transition outcome, stale result clearing, phasing, leave and return, target legality, stack copy, game clone, branch scope, registration errors, and compile-time role rejection. The next dependency-ready work is another P payment or action output, after a fresh audit. No production card changed.

Focused tests, short engine and repository tests, DSL registration and closed-role tests, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass.

## Current Slice: Bounded Mixed Damage Target Group

`TargetDamageRecipients(name, minimum, maximum) *DamageTargetsBinding` declares one target group containing players, creatures, or planeswalkers. The binding is a sealed `DamageRecipientSet` for `DealDamageEach` and a `TargetBinding` for spell, activation, or trigger announcement. It reuses target count and legality checks, recorded target incarnations, stack copy alignment, and the existing simultaneous damage service. Registration rejects invalid bounds and undeclared reads. A returned, phased-out, or illegal permanent leaves the damage batch; legal player targets remain. The legacy singular target and damage exports remain.

Prismari Charm's full text was checked in `data/SOS.json`: its second mode requires one or two distinct damage recipients. This slice supplies that target and damage body, but modal card wiring and the other modes remain outside this run. The Secrets of Strixhaven gap stays open, and no production card changed. A fresh search finds no production `TargetDamageRecipients` caller and 130 occurrences across three legacy damage constructor names; the existing exports remain. Focused tests first failed on the missing constructor. Synthetic tests cover mixed choices, duplicate and illegal announcement before payment, partial legality, leave and return, phasing, prevention, planeswalker loyalty, batch observation, stack copying, cloning, scope validation, and collection-versus-singular compile rejection. The next dependency-ready slice is a P payment output or another T announcement contract after a fresh audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass.

## Current Slice: Grouped Activation Target Announcement

`WithActivationTargetGroups(bindings...) ActivationOption` declares independent target groups for a nonmodal `Activate` body. `Game.ActivateAbilityByIndexWithTargetGroups(playerID, permanentID, abilityIndex, groups)` accepts one UUID slice per declaration. The activation path validates each group's count and legality before payment, records target-to-declaration alignment on the stack, and carries that alignment through resolution, copying, and cloning. The flat activation path rejects grouped declarations. `WithTargets` and all production callers remain available; no production card was changed.

Focused contract tests first failed because both exports were absent. They cover an empty first optional group, legal activation with a tap cost, invalid groups before payment, duplicate choices within one group, the same choice in two groups, copy retargeting, target leave and return, clone isolation, registration rejection of empty declarations, failed-payment rollback, and rejection of a grouped announcement on a mana ability. The shared target, stack-copy, and payment services are the reuse boundary. Modal and delayed group announcement, UI input, and card wiring remain separate. No gap record is closed by this adapter alone. A fresh production search found no `WithActivationTargetGroups` caller; existing activation and legacy targeting callers remain. The next dependency-ready slice is payment-result outputs in P, after a fresh audit of current capabilities and callers.

The focused tests, `go test ./pkg/mage/... -short -count=1`, DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass.

## Current Slice: Grouped Spell Target Announcement

`WithTargetGroups(bindings...) SpellOption` marks independent spell target declarations. `Game.CastSpellByNameWithTargetGroups(playerID, name, groups, optionalX)` accepts one chosen UUID slice per declaration. The cast path checks each group's count, legality, and duplicate choices before payment, then records the exact target-to-declaration alignment on the stack object. The flat cast path rejects grouped spells before payment. Resolution, copying, retargeting, and cloning reuse the existing target-frame and target-specification services. `WithTargetGroups` has a sealed spell-only option role; registration rejects an empty declaration. `WithTargets`, the flat cast API, and all production callers remain available.

The focused tests first failed because the constructor and grouped cast entry point were absent. Synthetic spells cover an empty first group, a required player plus two optional creature groups, duplicate choices within one group, the same choice in separate groups, invalid target/count rejection before payment, payment rollback, copy and clone isolation, and target leave-and-return. Dissection Practice's complete text in `data/SOS.json` requires these independent group counts, but its production card and cast UI are unchanged. The Secrets of Strixhaven gap remains open, with card wiring and grouped UI input outstanding. No production caller uses `WithTargetGroups`; legacy target surfaces remain. The next dependency-ready T slice is grouped activation announcement, including the activation payment transaction and stack copy boundary. Modal spells and delayed triggers remain outside this slice.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, the standalone reference example, and `git diff --check` pass.

## Current Slice: Trigger Target-Group Announcement

`WhenEvent` and `WhenEventModes` now permit multiple variable-size typed target groups. Trigger placement records each group's chosen targets with their exact declaration, including an empty optional group before a selected one. Typed group choices are checked for count, legality, and duplicates before stack placement. Resolution, stack copying, retargeting, and game cloning use the existing target-aligned record. There is no new public constructor; `TargetPermanents`, `WithTargets`, `WhenEvent`, and `WhenEventModes` are the author API. Spell, activation, and delayed declarations retain the registration guard because their flat announcement paths do not yet record independent counts. No production card, legacy export, or JSON gap status changed.

The focused contract test first failed at registration. Synthetic tests cover ordinary and modal trigger placement, an empty first group, copied retargeting, return-to-battlefield target identity, clone isolation, invalid choices, a shared target in distinct groups, and the spell/activation registration guard. Dissection Practice's full text was checked in `data/SOS.json`; its spell target groups and production wiring remain blocked. Focused tests, short engine and repository tests, the DSL registration and role run, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. The next dependency-ready T slice is grouped spell and activation announcement, including payment rollback and copy-safe target records.

## Current Slice: Recorded Target Groups During Copy

Spell and ability copies now use the target-aligned `StackObject.TargetSpecs` record when they offer new targets. An empty optional group before a selected optional group keeps its position and count; the copy does not reinterpret the flat target list. This is a target-announcement prerequisite, with no new public DSL function. `TargetPermanents(name, minimum, maximum, filter, options...)`, `WithTargets`, and `Spell` remain the supported author signatures for one variable-size group. The existing count-inference fallback remains for legacy stack objects without an aligned record. Registration still rejects two variable-size typed target groups because normal flat cast and activation input cannot state their independent counts. No production card, legacy export, or JSON gap status changed.

The focused contract test first failed when a copied spell reassigned a choice from the second optional group to the empty first group. It now checks copied retargeting, effect routing, original and game-clone isolation, copied count preservation, and a target that leaves and returns. The existing registration test continues to reject ambiguous declarations. Dissection Practice's complete text was checked in `data/SOS.json`; its independent optional creature targets still require a grouped announcement input and production wiring. The Secrets of Strixhaven gap remains open. The next dependency-ready T slice is grouped target announcement and validation, including cast and activation transactions, before registration can accept those declarations. Production callers of the legacy targeting surface remain unchanged.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, the standalone reference example build, and `git diff --check` pass. A fresh search finds existing production `TargetPermanents` and spell-copy callers; none were changed.

## Current Slice: One Variable-Size Permanent Damage Target Group

`TargetPermanents(name, minimum, maximum, filter, options...)` can now supply more than one declared permanent to `DamageRecipients` in `DealDamageEach`. The existing target specification alignment gives one variable-size group an unambiguous boundary. Resolution excludes illegal, phased-out, or returned target incarnations; the shared damage service still applies all remaining damage in one batch. `Spell`, `Activate`, and triggered resolving bodies can use the adapter. There is no new public function or engine executor. Two variable-size target groups in one ability still fail registration because their separate chosen counts are not recorded.

The focused contract first failed at registration on the prior maximum-one guard. Synthetic tests cover two chosen permanents, mixed player recipients, partial target illegality, stack copying, clone isolation, acceptance of one variable group, and rejection of two. The public reference example builds. Prismari Charm's one-or-two-target damage mode in the Secrets of Strixhaven gap inventory is a candidate for separate card wiring, but it also needs a mixed creature-or-player target group, so this slice does not close its gap. No production card or legacy export changed. Focused tests, short engine and repository tests, the DSL registration and role run, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. The next dependency-ready T slice is recording independent variable-size target-group counts at announcement and carrying them through copies and resolution.

## Current Slice: Referenced Permanent Blocking State

`PermanentIsBlocking(PermanentRef) StateCondition` checks current blocker membership for `This` or `AttachedPermanent` on a battlefield ability. It works in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, and `IfState`. The sealed role rejects player references at compile time; registration rejects target and saved references in state contexts. It uses the existing blocker-incarnation service, including ordinary declarations and creatures put onto the battlefield blocking. An absent, phased-out, removed, or returned object does not match. The attached form requires the Aura and host incarnations. Clone combat state stays independent. No production card or legacy export changed.

The focused test first failed because the constructor was absent. Synthetic engine and DSL tests cover declaration, removal, attachment, static evaluation, phasing, leave and return, clone isolation, inspection, registration, and compile-time role rejection. Production cards still use legacy blocking conditions and filters or need additional policy such as combat-opponent selection; their gap records remain open. A fresh search found no `PermanentIsBlocking` production use. The next dependency-ready slice is T recorded target-group boundaries, after a fresh audit.

`go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. The standalone public DSL test validates the documented Aura form. No JSON gap record changed because no production card was wired.

## Current Slice: Referenced Permanent Attacking State

`PermanentIsAttacking(PermanentRef) StateCondition` checks current attacking status for `This` or `AttachedPermanent` on a battlefield ability. It is pure and works in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, and `IfState`. The combat service now records attacker incarnations on normal and simulated declarations and on creatures that enter attacking. A phased-out, absent, removed, or returned object does not satisfy the condition. The attached form checks both the Aura and host incarnations. Registration rejects target and saved references in this state context; the sealed `PermanentRef` role rejects player references at compile time. Legacy card-ID combat queries and production callers remain.

The contract first failed because the constructor was absent, then exposed a stale card-ID attack after leave and return. Focused tests cover that regression, removal, phasing, clone isolation, attachment state, static evaluation, registration, normal declaration, and attacking entry. The clone allocation test caught eager allocation of an empty attacker-identity map; lazy allocation restored its budget. `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. A fresh caller search found three `WhileSourceAttacking` uses in two production card files and no `PermanentIsAttacking` production use; other card-ID attacking filters and queries remain. No production card or gap record changed. The next dependency-ready slice is `PermanentIsBlocking(PermanentRef)` for attached and source state, after a fresh audit. Card migration remains separate.

## Current Slice: Referenced Permanent Damage Source

`DamageSourceOf(PermanentRef) DamageSourceRef` lets `DealDamageFrom` and `DealDamageEach` use a declared target or saved permanent as their damage source. Target and snapshot references retain their battlefield incarnation; a departed source uses durable departure LKI, while phasing without departure makes it absent. The adapter uses the shared damage executor, inspection, and target scope validation. It does not change production cards or the legacy `DealDamage` signature.

Tracker's complete Oracle text in `data/DRK.json` requires two sequential instructions with different creature sources. This slice supplies the referenced source for the reciprocal instruction; power lookup after a mid-resolution departure and production card wiring still need an exact audit. No card gap is closed by adapter availability. A fresh search found 19 production card files calling legacy `DealDamage` and none using `DamageSourceOf` or `DealDamageFrom`; those exports remain. The next dependency-ready work is recorded target-group boundaries in T, after a fresh audit.

The focused contract first failed because `DamageSourceOf` was absent. Tests now cover live spell targets, departure and return with old controller and lifelink, saved permanent identity, phasing, game clone isolation, scope and nil declaration rejection, and a player-source compile error. The documented DSL example builds. `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass.

## Current Slice: Each Controlled Attacker Event

`EachCreatureYouControlAttacks().WithAttacker(*PermanentBinding).WithDefendingPlayer(*PlayerBinding)` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It matches each newly declared creature controlled by the source controller and optionally saves that attacker incarnation and its defending player in the pending trigger. The captures use the existing event and instance-local binding services. A creature entering attacking does not match. The combat service currently records only player defenders; planeswalker and battle defender identity is separate work. No production card or legacy export changed.

Hellrider's complete Oracle text was checked in `data/JMP.json` and its Jumpstart gap record. This slice supplies its per-attacker condition and player-defender capture, but planeswalker-defender capture and production wiring remain open; the gap stays blocked. The focused contract first failed because the constructor was absent. Tests cover two independent attackers, actual declaration, controller matching, leave-and-return attacker identity, source phasing and return, attacker phasing, stack-copy and game-clone isolation, inspection, and registration errors. The next dependency-ready slice is T recorded target-group boundaries or the combat defender union needed for planeswalker attacks, after a fresh audit.

`go test ./pkg/mage -run '^TestEachControlledAttacker' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, the standalone reference example build, and `git diff --check` pass. A fresh card search finds nine files with legacy attack-trigger construction and no production use of the new typed export. Those callers and their exports remain. No JSON gap is closed by this prerequisite.

## Current Slice: Controlled Attack Group Event

`OneOrMoreCreaturesYouControlAttack().WithCount(*NumberBinding)` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It matches once when the source controller declares one or more attackers, and its optional count binding saves the number actually declared in that trigger instance. The normal and simulated attack paths send the same completed group event. Zero declarations and creatures put onto the battlefield attacking do not match. Registration rejects a nil or repeated count binding and reads outside its trigger scope. Existing legacy group-trigger exports and production callers remain.

Path of Bravery's full Oracle text was checked in `doc/designs/composable-dsl-migration-jumpstart-gap.json`: its group attack clause needs the count, while its static starting-life comparison and production wiring remain open. Duelist's Heritage also needs a target and optional effect. No JSON gap record is closed by this adapter. Synthetic engine and test-only card tests cover controller matching, one trigger for a two-creature group, the actual count, zero and nondeclaration events, phasing, source departure and return, game clone isolation, registration, normal and simulated declaration. The next dependency-ready slice is T recorded target-group boundaries or a per-attacker E participant after a fresh audit. No production card changed.

`go test ./pkg/mage -run '^TestAttackBatchPattern' -count=1`, the focused test-only card run, `go test ./pkg/mage/... -short -count=1`, the DSL registration and role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. A fresh caller search finds three production uses of the legacy controlled attack-group trigger and no production use of the typed export.


## Current Slice: Unblocked-Attacker Event

`AttacksUnblocked(This)` is a sealed `EventPattern` for `WhenEvent` and `WhenEventModes` on a battlefield source. The engine sends `core.EvtAttacksUnblocked` once for each live attacker that has no legal blocker after blocker declaration settles. The event includes creatures that entered attacking and records the defending player in `TargetID`. `.WithDefendingPlayer(*PlayerBinding)` saves that player in the trigger's instance-local frame. Existing `EvtBlockersDecl`, attack events, and legacy callers remain compatible. The pattern supplies no optional sacrifice or target declaration.

Mindstab Thrull and Necrite requirements were checked against `data/FEM.json` and the Fallen Empires inventory and gap record. Their optional sacrifice, conditional effect, Necrite target, and production wiring remain open; no gap record is closed. The focused contract first failed because `AttacksUnblocked` was absent. It now covers legal blocker settlement in normal and simulated declarations, event-time defender capture, a blocked attacker whose blocker leaves, phased source absence, source return, stack copy, clone isolation, registration errors, and a compile-time role error. No production card was changed. The next dependency-ready slice is the T recorded target-group boundary or the E attack-batch event pattern, after a fresh audit.

`go test ./pkg/mage -run '^TestUnblockedAttack' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. A fresh card search finds no `AttacksUnblocked` production declaration. Legacy attack and blocker completion callers remain; this adapter does not make those cards complete.

## Current Slice: Attack Defending-Player Capture

`Attacks(This).WithDefendingPlayer(*PlayerBinding)` saves the defending player from the declaration event for `WhenEvent` and `WhenEventModes`. Both normal and simulated attack declarations now put the defending player in `EvtDeclaredAttacker.TargetID`; the trigger captures that event value in its own resolution frame. This adds no target, attack-batch pattern, unblocked-after-blockers pattern, or production card wiring. Existing `Attacks(This)` and legacy exports remain. The engine's two-player attack service supplies a player defender. A future player-or-planeswalker recipient binding needs a distinct typed role.

The new focused tests first failed at compile time because the binding method was absent. They cover actual declaration, event-time capture after source departure and return, clone isolation, phasing, source absence, scope validation, and invalid declarations. The relevant inventory cases for Mindstab Thrull and Necrite still need an unblocked-attacker event and successful optional sacrifice; their JSON gaps stay open. Hellrider still needs an each-attacker event and recipient identity. The next dependency-ready work is the T recorded target-group boundary or the E unblocked-attacker event prerequisite, after a fresh audit.

`go test ./pkg/mage -run '^TestAttackDefendingPlayer|^TestSourceCombatEvents' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. No gap record is closed because no production card was migrated.

## Current Slice: Creature-You-Sacrifice Event

`CreatureYouSacrifice() EventPattern` matches each completed creature sacrifice made by the battlefield source's controller, including tokens and the source itself. It uses the existing `EvtSacrifice` payload and pending-trigger service. The pattern is available through `WhenEvent` and `WhenEventModes` and has no participant binding or target. The source's departure LKI supports self-sacrifice; a pending trigger retains its controller and source incarnation through a return. Legacy sacrifice-trigger exports and all production callers remain.

Kels, Fight Fixer's complete Oracle text was checked in `data/JMP.json` and its Jumpstart gap record. This slice supplies its sacrifice-event condition, but hybrid optional payment, its other ability, and production wiring remain separate. The Jumpstart gap record stays open. Focused tests first failed because the constructor was absent. They cover own, opponent, stolen, token, source, and noncreature sacrifices; phasing; pending return identity; copy and clone isolation; untargeted triggers; and registration. The next dependency-ready work is the T recorded target-group boundary or another E event participant after a fresh audit.

`go test ./pkg/mage -run '^TestCreatureYouSacrificeEvent' -count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. The earlier `cards/custom` and C metadata blockers do not reproduce in this working tree. No gap record is closed because Kels remains unmigrated.


## Current Slice: Spell-Cast Type Event

`SpellCast().OfType(core.CardType)` filters a battlefield source's cast-spell trigger by one nonland type on the event's spell stack object. It composes with color, opponent, and caster options in `WhenEvent` and `WhenEventModes`. Registration rejects land, invalid, or repeated type filters. The existing cast event, stack identity, trigger capture, and clone services supply the contract. The slice adds no spell binding, cast-cost metadata, or production card wiring.

Vedalken Archmage's full Oracle text was checked in `data/JMP.json` and its Jumpstart gap record. The artifact-cast condition is available, but its production definition remains unmigrated, so the gap stays open. Existing legacy cast-trigger exports and callers remain. Contract tests first failed because `OfType` was absent. They now cover type matching, a different stack object with the same source ID, event-time caster capture, source phasing and absence, real casts, failed-payment rollback, clone isolation, inspection, and registration. The next dependency-ready slice is the T recorded target-group boundary or another E event participant after a fresh audit.

`go test ./pkg/mage -run '^TestSpellCast(Type|Color|Caster)Event' -count=1`, the documented standalone example build, and `git diff --check` pass. Runnable packages in the engine and repository short tests and `make test` pass, but the full commands, DSL registration run, `go build ./...`, and `go vet ./...` fail setup because `cards/all.go` imports the absent `cards/custom` package. `git ls-tree HEAD cards/custom` is empty. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. Required gates did not pass, so this slice stays reviewable and uncommitted among pre-existing changes.


## Current Slice: Opponent Draw Event Participant

`OpponentDrawsCard(*PlayerBinding) EventPattern` matches each completed card draw by an opponent of the source controller at detection and saves that player in the trigger instance. It uses the existing `EvtCardDrawn`, opponent condition, pending-trigger capture, and damage action. It is supported by `WhenEvent` and `WhenEventModes` on battlefield sources. Replaced and empty-library draw attempts do not match. The existing `ControllerDrawsCard` behavior and all production callers remain. No production card changed.

Underworld Dreams's complete Oracle text was checked in `data/LEG.json` and its recorded requirement in `doc/designs/composable-dsl-migration-legends-gaps.json`. Its opponent-draw and typed singular-damage clauses are expressible, but its production definition remains unmigrated, so the Legends gap record stays open. Contract tests first failed because `OpponentDrawsCard` was absent. Focused tests cover repeated draws, the source controller's draw, draw replacement, empty-library attempts, event-time control, phasing, source leave and return, copied triggers, clone isolation, absence of targets, inspection, and registration errors. The next dependency-ready work is T recorded target-group boundaries or another small E participant after a fresh audit.

`go test ./pkg/mage -run 'TestOpponentDrawEvent|TestControllerDrawEvent' -count=1`, `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, the standalone documented example build, and `git diff --check` pass. Runnable packages pass under `go test ./pkg/mage/... -short -count=1`, `go test ./... -short -count=1`, and `make test`, but those commands and the DSL registration run stop at the absent `cards/custom` import in `cards/all.go`. `go build ./...` and `go vet ./...` stop on the same import. `git ls-tree HEAD cards/custom` confirms that package is absent at HEAD. `make lint` also reports absent C metadata for `cmd/pylib/decision_spec_ffi.go`. The required gates do not all pass, so this slice remains uncommitted among pre-existing workspace changes.

## Current Slice: Spell-Cast Caster Binding

`SpellCast().ByOpponent().WithCaster(*PlayerBinding)` adds an event-time opponent filter and an instance-local caster binding to the existing battlefield cast pattern. Either method can be used alone, and both compose with `OfColor`. The opponent check reuses `EvtSpellCast.PlayerID` and the source controller at detection. The capture uses the existing pending-trigger and player-binding services; it does not bind the spell, choose a target, or change payment behavior. Declaration inspection and validation reject absent or repeated bindings and out-of-scope reads. Production cards remain unchanged.

Rhystic Study's complete Oracle text in `data/JMP.json` requires an opponent cast event and that caster for its optional payment. This slice supplies the event participant; its optional draw/payment body and production wiring remain separate. The Jumpstart gap record stays open. Contract tests first failed because `ByOpponent` and `WithCaster` did not exist. Focused tests cover own and opposing casters, event-time control through source leave and return, separate casts, stack-copy and clone isolation, phasing, failed cast, inspection, and registration errors. Existing cast-trigger exports and their production callers remain. The next dependency-ready work is the recorded target-group boundary in T or another small event participant after a fresh audit.

`go test ./pkg/mage -run '^TestSpellCastCasterEvent' -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, the standalone authoring example build, and `git diff --check` pass. The runnable engine packages pass under `go test ./pkg/mage/... -short -count=1`; the DSL and two AI packages stop at setup. The required DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop at the absent `cards/custom` package imported by `cards/all.go`; `git ls-tree HEAD cards/custom` confirms that it is absent at HEAD. `make lint` stops on that import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. Required gates do not pass, so this slice is reviewable and uncommitted among pre-existing workspace changes.

## Current Slice: Spell-Cast Color Event

`SpellCast().OfColor(core.Color)` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. The color filter is optional and accepts one of the five Magic colors. It checks the cast spell's effective stack color when `EvtSpellCast` fires, after payment. A color change on the stack or a printed color override counts. The cast event now carries the spell stack ID in `TargetID`, so an older copy with the same source card ID cannot change the result. The existing pending-trigger service keeps the event-time source controller; the pattern adds no spell or caster binding and no target. The legacy cast-trigger exports and all production callers remain.

Crystal Rod's complete Oracle text in `data/4ED.json` and Sol'kanar the Swamp King's in `data/LEG.json` require a spell-cast color filter. The adapter supplies that event condition, but their production declarations remain unmigrated. Their gap records stay open. Tests first failed because `SpellCast` was absent. A second regression first failed when a blue stack copy was mistaken for a red spell cast from the same source card. Focused tests now cover either caster, multicolor spells, stack color changes, shared source IDs, source phasing, source leave and return, clone isolation, real casting, failed-payment rollback, inspection, and registration errors. The next dependency-ready E slice is a cast-event caster binding for opponent-cast triggers such as Rhystic Study, after its event-time identity contract is audited; T recorded target-group boundaries remain independent work.

`go test ./pkg/mage -run '^TestSpellCastColorEvent' -count=1`, `go test ./pkg/mage -short -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, the standalone DSL example build, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes runnable engine packages but fails DSL and two AI package setup steps. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail because `cards/all.go` imports the missing `cards/custom` package, absent from both the working tree and `HEAD`. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. Required gates do not pass, so this slice remains reviewable and uncommitted among pre-existing workspace changes.

## Current Slice: Opponent Land Tap Event

`OpponentLandBecomesTapped().OfSubtype(string)` is a typed battlefield event pattern for an opponent-controlled land changing from untapped to tapped. The subtype filter is optional and registration checks a supplied land subtype. It reuses `EvtTapped` and the existing trigger conditions. The event checks controller and characteristics when the tap occurs; the queued trigger keeps its original controller if the land or source later changes zones or control. It is available through `WhenEvent` and `WhenEventModes` and does not capture a participant or implement tapped-for-mana events. No production card changed.

Lifetap's Forest clause in `data/4ED.json` and Lifeblood's Mountain clause in `data/LEG.json` match this pattern. Both cards remain unmigrated, so their gap records stay open. The focused tests first failed because the constructor was absent. They cover opponent versus own control, subtype, tap transition, phasing, pending source and land identity, cloning, inspection, and registration. The next dependency-ready work is another filtered event participant or T recorded target-group boundary, after a fresh audit.

`go test ./pkg/mage -run '^TestOpponentLandBecomesTapped' -count=1`, `go test ./pkg/mage -short -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, the documented example build, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` and `go test ./... -short -count=1` pass their runnable packages but fail DSL and other packages at setup. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` confirms it is absent at HEAD. `make lint` stops on that missing package and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. Required gates do not pass, so this slice remains uncommitted and reviewable among existing workspace changes.

## Current Slice: Typed Source Leave Event

`LeavesBattlefield(This) EventPattern` is available in `WhenEvent` and `WhenEventModes` on a battlefield permanent. It reuses the zone-change event, pre-departure trigger snapshot, and `OnLeaveZone` executor. It matches a source move from the battlefield to any other zone, including destruction, exile, and return to hand. Phasing does not match. Pending triggers keep their event controller and source incarnation through a same-card return, stack copying, and game cloning. No production card changed. The pattern does not add a delayed watched-object subscription or a general participant binding.

Contract tests first failed because the constructor was absent. Focused tests cover the three destinations, phasing, return identity and controller, simultaneous source departures, copy and clone isolation, registration, and inspection. Thragtusk's complete Oracle text was checked in `data/JMP.json` and its recorded gap in `doc/designs/composable-dsl-migration-jumpstart-gap.json`. Its leave-event requirement is available, but production wiring remains and the JSON gap stays open. Existing `OnLeaveZone` production callers remain. The next dependency-ready work is a filtered nondeath event participant or T recorded target-group boundaries, after a fresh audit.

`go test ./pkg/mage -run '^TestSourceLeavesBattlefield' -count=1`, the runnable packages in `go test ./pkg/mage/... -short -count=1`, the standalone DSL example build, and `git diff --check` pass. The required full engine check, DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail at package setup because `cards/all.go` imports the absent `cards/custom` package. `git ls-tree HEAD cards/custom` confirms that package is also absent at HEAD. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. Required gates did not all pass, so the slice remains reviewable and uncommitted among pre-existing workspace changes.

## Current Slice: Attached-Permanent Tapped State

`PermanentIsTapped(PermanentRef)` and `PermanentIsUntapped(PermanentRef)` are sealed `StateCondition` constructors. Registration accepts `This` and `AttachedPermanent` on permanent abilities and rejects target or saved bindings because a state check has no resolution binding frame. The attached form reads the Aura's current attachment and both battlefield incarnations; absent and phased-out objects satisfy neither condition. It follows reattachment and remains pure in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, and `IfState`. The existing source-specific conditions remain available to production callers.

Focused tests first failed because the constructors were absent. Synthetic tests cover tap changes, reattachment, host and source leave-and-return, phasing, clone isolation, static layer recalculation, and registration rejection. Spectral Cloak's complete Oracle text was checked in `data/LEG.json`. Its legacy production `AttachedEffect` callback and all other card wiring remain unchanged; the Legends gap record remains open. This slice does not add arbitrary bound-permanent state checks, target-group boundaries, or target announcement changes. The next dependency-ready slice is T recorded target-group boundaries after a fresh audit.

`go test ./pkg/mage -run TestPermanentTappedAttachment -count=1`, `go test ./pkg/mage -short -count=1`, `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, a standalone build of the documented DSL example, `go test ./cards/legends -short -count=1`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail at setup because `cards/all.go` imports the absent `cards/custom` package. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `git ls-tree HEAD cards/custom` confirms the package is absent at HEAD. The required full checks did not pass, so this work remains uncommitted.

## Previous Slice: Typed One-Color Mana Production

`AddManaOfAnyOneColor(ResolutionValue) ResolvingEffect` exposes a resolution-time choice of one color for the full amount. It reuses the existing mana choice and production executor. Fixed positive amounts in tap-only mana activations are visible to the automatic payment planner as one `AnyColor` production. Dynamic amounts remain mana abilities but are not advertised as fixed production. The adapter works in resolving spell, activation, and trigger bodies. It does not add stored-color reads, any-combination choices, spending restrictions, or card wiring.

The focused tests first failed because the constructor was absent. They now cover the one-color amount, immediate mana ability timing, tap-for-mana event, fixed planning, refusal to split colors, failed-payment rollback, dynamic amount, stack copy controller, clone isolation, and declaration validation. The complete Oracle text of Implements of Sacrifice and Birds of Paradise was checked in `data/FEM.json` and `data/4ED.json`. Their production requirement is expressible, while their production cards and the other legacy mana callers remain unchanged. The Fallen Empires gap record stays open. The next dependency-ready slice is T recorded target-group boundaries after a fresh audit.

`go test ./pkg/mage -run '^TestAddManaOfAnyOneColor' -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail at setup because `cards/all.go` imports the absent `cards/custom` package. `git ls-tree HEAD cards/custom` confirms that package is absent at HEAD. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. The required gates did not all pass, so this slice remains reviewable and uncommitted. A fresh production search found 12 card files using `AddAnyMana` or `WithAnyColorMana`; those exports and all card wiring remain.


## Current Slice: Chosen Counter Payment Count

`RemoveAnyCountersCost(core.CounterType, *NumberBinding) Cost` is an activated-ability cost that chooses from zero through the source's current counter count. The payment transaction locks the choice and battlefield incarnation, validates it on an isolated game, removes the chosen counters at commit, and records the paid count for `FromNumber` in the resolving body. A returned, phased-out, absent, or counter-short source cannot pay. Stack copies and game clones retain independent captured numbers. This slice does not add mana production, counter-transfer actions, spell additional costs, or production card wiring.

The new contract tests first failed because the constructor was absent. Focused tests cover one choice, payment before resolution, zero and invalid choices, returned-source failure, registration checks, stack copying, and clone isolation. Black Mana Battery's complete Oracle text was checked in `data/LEG.json`; the Fallen Empires storage lands have the same chosen-counter requirement. Their remaining mana-production, untap, intervening-if, and card-wiring work remains open. No JSON gap record is closed. Legacy counter costs and all production callers remain. The next dependency-ready slice is T recorded target groups or M chosen-color mana, after a fresh audit.

`go test ./pkg/mage -run 'TestPaidCounterChoice|TestPaidSacrifice' -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. The runnable packages in `go test ./pkg/mage/... -short -count=1`, `go test ./... -short -count=1`, and `make test` pass, but those full commands, the DSL registration run, `go build ./...`, and `go vet ./...` fail at setup because `cards/all.go` imports the absent `cards/custom` package. `git ls-tree HEAD cards/custom` confirms the package is also absent at HEAD. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. Required gates did not pass, so this slice remains reviewable and uncommitted among earlier workspace changes.

## Current Slice: Typed Spell Mana Value

`ManaValueOfSpell(StackObjectRef) ResolutionValue` reads the current mana value of a targeted spell on the stack. It uses the existing stack target and mana-cost services, including the stack object's announced X for each `{X}`. A missing spell yields zero. The read is scoped to a declared target and follows copied target frames. It supports resolving effects; it does not add announcement-time mana-value target restrictions, delayed mana, ability targets, or card wiring. Mana symbols outside the current parser's representation remain a separate engine prerequisite.

The new focused tests first failed because the adapter was absent. They cover stack X, stack ID after recasting the same card, absence, clone isolation, copied retargeting, resolution-time target legality, and registration rejection. `Mana Drain` in `data/LEG.json` and `Spell Blast` in `doc/designs/composable-dsl-migration-limited-gaps.json` require stack mana value, but their remaining timing or target restrictions and production wiring remain open. No gap record is closed. Legacy exports remain for production callers. The next dependency-ready slice is a bounded target-group contract or another independent K or P unit after a fresh audit.

Focused tests, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes runnable packages but fails DSL and two AI packages at setup. The required DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` also fail at setup because `cards/all.go` imports `cards/custom`, which is absent from the working tree and `HEAD`. `make lint` stops on that import and missing C metadata in `cmd/pylib/decision_spec_ffi.go`. These are existing workspace blockers outside this slice. Required checks did not all pass, so this slice is reviewable and uncommitted among pre-existing changes.

## Current Slice: Typed Spell Controller Reference

`ControllerOfSpell(StackObjectRef) PlayerRef` reads the current controller of one targeted spell on the stack at resolution. `TargetSpell` supplies the sealed reference, and the existing target frame, stack lookup, and payment transaction supply the execution contract. It is available in resolving effect bodies such as `UnlessPaid` and `IfPaid`. It does not add stack mana value, ability targets, target groups, or card wiring. A missing stack object produces an absent player; a later cast of the same card has a different stack ID. Registration rejects a nil or out-of-scope spell binding, and the sealed roles reject a permanent or player argument at compile time.

Force Spike's complete Oracle text was checked in `data/LEG.json`, and Vodalian Mage's in `data/FEM.json`. Their controller-payment requirement is expressible with this adapter, but both cards remain unmigrated. Quandrix Charm also retains its production wiring. The gap records remain blocked until the complete card declarations are checked and migrated. Legacy payment and counter exports remain for production callers. The next dependency-ready work is a stack mana-value reference or a bounded target-group contract after a fresh audit.

The focused contract tests first failed because `ControllerOfSpell` was absent. They now pass with `-count=1` and cover owner versus controller, optional payment through the existing transaction, stack departure and recast, copied and retargeted frames, clone isolation, target legality at resolution, and declaration validation. A compile-failure fixture records the wrong-role case, but the DSL test harness cannot run in this workspace. `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes runnable engine packages but fails DSL and two AI package setup steps. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail at package setup because `cards/all.go` imports the absent `cards/custom` package, which is also absent at `HEAD`. `make lint` stops on that import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. Required checks did not all pass, so this slice remains reviewable and uncommitted among earlier workspace changes. No production card changed.

## Current Slice: Death-Event Graveyard Card Reference

`BindCard(name)`, `FromCard(binding)`, and `CreatureDies().WithCard(binding)` (also available on the other death patterns) bind the destination graveyard visit of a dying nontoken permanent. The typed `CardRef` works with existing graveyard actions and becomes absent if the card leaves that graveyard visit. The capture uses the completed destination state for simultaneous deaths, while the trigger still checks event-time battlefield characteristics. Tokens and replaced deaths bind absence. The binding is instance-local, copied with trigger and game state, and validated for name, producer, and scope. It is not a target. No production card changed.

The contract tests first failed because `BindCard`, `FromCard`, and `WithCard` were absent. Focused tests cover live and returned visits, simultaneous deaths including source death, a copied trigger, clone isolation, tokens, phased source, and registration errors. The reusable boundary is the existing graveyard visit and trigger capture services. A general card-zone choice, card-set binding, and production card wiring remain separate. The next dependency-ready work is a filtered nondeath event participant or the P/T extensions in delivery order after a fresh audit. Existing legacy exports remain because production callers have not migrated.

Puppet Master's full printed text was checked in `doc/designs/composable-dsl-migration-legends-gaps.json`, and Vastwood Zendikon's Oracle text in `data/JMP.json`. Both refer to the card that died. Puppet Master still needs the successful-return result and conditional payment path; Vastwood Zendikon still needs its Aura static animation and card wiring. Neither gap record is closed by this adapter.

`go test ./pkg/mage -run 'TestDeathEventCard|TestPermanentDies|TestCreatureDies' -count=1`, `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, and the standalone DSL example compile pass. `go test ./pkg/mage/... -short -count=1` passes all runnable packages but fails DSL and two AI package setup steps. `go test ./... -short -count=1`, the required DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail because `cards/all.go` imports `cards/custom`, which is absent from the working tree and `HEAD`. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. The required gates did not pass, so this slice is reviewable and uncommitted among earlier workspace changes. A fresh search found no production caller of `WithCard`, `BindCard`, or `FromCard`; no JSON gap record is closed by adapter availability.

## Current Slice: Typed Permanent-Death Type Filter

`PermanentDies()` and `PermanentDies().OfType(core.CardType)` are sealed battlefield `EventPattern` declarations for `WhenEvent` and `WhenEventModes`. They reuse the zone-change event, departure LKI, and existing death-event player bindings. A type filter reads the departing permanent before it leaves; `.WithController(binding)` saves its former controller. Registration rejects nonpermanent card types and repeated type filters. The pattern has no target and does not bind the dead card in its graveyard. No production card was changed.

Dingus Egg's complete Oracle text was checked in `data/4ED.json`, and Tablet of Epityr's in `data/ATQ.json`. The adapter covers their land/artifact death conditions and Dingus Egg's former-controller capture. Both cards still need separate wiring, and Tablet still needs its optional payment body migrated. Their gap records remain open. Existing legacy death exports and production callers remain. The next dependency-ready E slice is a zone-change card reference with visit identity, or another filtered event pattern after a fresh audit.

The new contract tests first failed because `PermanentDies` was absent. Focused tests cover land and artifact filters, simultaneous deaths with source departure, event-time controller, copy and clone isolation, phasing and absent source, graveyard replacement, unfiltered per-object events, inspection, and invalid registration. `go test ./pkg/mage -run 'TestPermanentDies|TestCreatureDies|TestAnotherNontokenCreatureDies' -count=1`, `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, and the standalone reference example pass. Runnable packages in `go test ./pkg/mage/... -short -count=1`, `go test ./... -short -count=1`, and `make test` pass, but the full commands, the DSL registration run, `go build ./...`, and `go vet ./...` fail at package setup because `cards/all.go` imports `cards/custom`, which is absent in the workspace and HEAD. `make lint` stops on that import and missing C metadata in `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. Required gates did not pass, so this slice remains reviewable and uncommitted among pre-existing changes.

## Current Slice: Typed Grouped Block Pattern

`BlocksOrBecomesBlockedBy(This).OfSubtype(string)` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It uses the existing `EvtBlockerGroup` service and tests the opposite creatures' subtypes at detection. One legal group with several matches makes one trigger. The source incarnation and event controller remain with the pending trigger. Registration rejects an empty or noncreature subtype. No production card was changed.

Dwarven Soldier's complete Oracle text was checked in `data/FEM.json`. The adapter supplies its Orc group condition; its temporary toughness modification and card wiring remain separate. The Fallen Empires gap record stays blocked. Synthetic tests cover both block directions, multiple matches, a nonmatch, source departure and return, clone isolation, and invalid declarations. The next dependency-ready slice is an event participant reference or another small E event pattern after a fresh audit. Existing legacy event exports and production callers remain.

`go test ./pkg/mage -run TestBlocksOrBecomesBlockedByGroup -count=1`, `go build ./pkg/mage/dsl`, and the runnable engine packages pass. The new tests first failed at compile time because the constructor was missing. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail at package setup because `cards/all.go` imports `cards/custom`, which is absent from the workspace and HEAD. `make lint` stops on that import and missing C metadata in `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. Required checks did not pass, so this slice stays reviewable and uncommitted among pre-existing changes.

## Current Slice: Paid Sacrifice Subtype

`SacrificeCreatureCostWithSubtype(subtype string, result *NumberBinding) Cost` saves whether the one creature chosen for the cost had a creature subtype immediately before sacrifice. The chosen creature may be the source. The existing locked transaction, number frame, declaration inspection, and copy services supply the contract. This is a P payment-output slice for `Activate`, `ActivateModes`, and spell additional costs. It does not add a general paid-object characteristic record, a multi-object sacrifice, or resolution-time payment output. Production cards and legacy callers are unchanged.

Ebon Praetor's complete text was checked in `data/FEM.json`. The saved value supports its Thrull condition, but its counter removal, upkeep-only once-per-turn restriction, and production wiring remain outside this slice. Its Fallen Empires gap remains open. Synthetic tests first failed on the missing constructor and then found the missing registration traversal. They cover positive and negative subtype results across pending activations, source sacrifice, stack copy, game clone, returned chosen creature, spell additional-cost validation, resolution-time cost rejection, and a wrong-binding compile case. The public DSL example compiles. No production card uses the new cost; existing sacrifice author exports and callers remain. The next dependency-ready work is another P output or a T target extension after a fresh audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass. Earlier missing-package and C-metadata failures do not recur in this tree.

## Current Slice: Grouped Block Event Prerequisite

`core.EvtBlockerGroup` now reports one event per creature for the new legal blocker pairs in a declaration. `GameEvent.RelatedIDs` records the opposing creature IDs in combat order; `Flag` distinguishes blocking from becoming blocked. The event fires after minimum-blocker restrictions and required blocks settle. A creature put onto the battlefield blocking reports only the attacker side. Existing `EvtDeclaredBlocker`, `EvtCreatureBlockedBy`, `EvtCreatureBecomesBlocked`, and production callers remain. This slice adds no typed event pattern or card wiring.

Dwarven Soldier's complete Oracle text in `data/FEM.json` requires one trigger when it blocks or becomes blocked by one or more Orcs. The grouped event supplies its new-pair identity and one-or-more boundary, but the subtype-filtered typed adapter and card wiring remain. Its gap record stays blocked. Synthetic tests first failed because the event and member list were absent. Focused tests now cover legal and removed pairs, both event roles, multiple blockers, a multi-blocking source, entry blocking, event-time subtype detection, old pairs, and clone isolation. The next dependency-ready E slice is the typed Orc-compatible group pattern with registration checks.

The public engine signature is `core.EvtBlockerGroup` with `core.GameEvent.RelatedIDs`; there is no new card-facing DSL signature. No production card uses this event. `go test ./pkg/mage -run TestBlockerGroupEvent -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and the standalone reference example pass. The runnable packages in `go test ./pkg/mage/... -short -count=1` and `go test ./... -short -count=1` pass, but both full commands, the DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail during setup because `cards/all.go` imports the absent `cards/custom` package. `git ls-tree HEAD cards/custom` confirms it is absent at HEAD. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. The required gates did not pass, so this slice remains uncommitted and reviewable among prior workspace changes.

This document tracks active, incomplete implementation items, engine blockers, and acceptance criteria for the composable DSL migration. All completed items and historical slice records have been archived in git history.

## Current Slice: Land Tapped for Mana Event Player

`LandTappedForMana().WithPlayer(*PlayerBinding)` is a sealed battlefield event pattern for `WhenEvent` and `WhenEventModes`. It matches the existing positive-production `EvtTappedForMana` only when the mana source is a land, and saves the activating player from the event in the pending trigger's instance frame. `DealDamageFrom(This, FromPlayer(player), Fixed(1))` supplies Manabarbs' complete instruction in a synthetic definition. Its Oracle text was checked in `data/4ED.json`; production card wiring remains unchanged. The public API does not expose a mana replacement, produced-color binding, or mana-trigger shortcut. Mana Flare, Gauntlet of Might, and Wild Growth retain their production requirements. Legacy event and damage callers remain.

The new focused tests first failed because the constructor was absent. They cover both player identities, land versus artifact, zero production, a phased source, source departure and return after a pending trigger, stack copying, clone isolation, inspection, and registration rejection for invalid or unproduced bindings. The existing mana event and damage pipelines supply timing and replacement behavior. The Limited JSON gap remains open because Manabarbs has not been migrated. The next dependency-ready work is a P or T extension from delivery step 3, such as recorded variable target-group boundaries, after a fresh contract audit.

Focused tests, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint`, and `git diff --check` pass in the current tree. A fresh caller search found no production `LandTappedForMana` use; Manabarbs still uses `NewTriggered(EvtTappedForMana, ...)` and positional damage. Earlier missing-package and C-metadata failures recorded below do not recur in this run.

## Current Run: Final Declared-Blocker Event Timing

`core.EvtDeclaredBlocker` now fires only for new legal blocker and attacker pairs after minimum-blocker restrictions and required blocks settle. Its first pair for a blocker in combat retains `Flag=true`; later pairs use `Flag=false`. Normal declaration and `ExecuteBlockers` use the same final-pair service. This is an engine prerequisite for a typed blocks-or-becomes-blocked-by-one-or-more pattern. It changes no author signature, production card, target, or replacement rule. Existing `Blocks(This)` in `WhenEvent` and `WhenEventModes` now observes only final legal declared blocks. Creatures put onto the battlefield blocking remain outside that pattern.

The new contract test first failed because a block removed by a three-blocker minimum had already queued `EvtDeclaredBlocker`. Focused tests now pass for that case through normal and simulated declaration, and for two legal pairs with one once-per-blocker trigger in an isolated clone. Dwarven Soldier's complete text was checked in `data/FEM.json`; Orc filtering and one-or-more grouping remain unimplemented. Spitting Slug and other legacy declared-blocker callers remain exported and unchanged. No JSON gap record is closed, and all card wiring remains open. The next dependency-ready E slice is the typed grouped Orc-compatible block pattern with event-time characteristics and one-or-more behavior.

`go test ./pkg/mage -run 'TestDeclaredBlockerEvent|TestNormalBlockDeclarationRejectsRemovedPairEvent' -count=1`, focused CR 509 integration tests, `go build ./pkg/mage/dsl`, and `git diff --check` pass. Runnable engine packages pass `go test ./pkg/mage/... -short -count=1`. The full engine and repository test commands, DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail at setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` stops on that missing import and absent C metadata in `cmd/pylib/decision_spec_ffi.go`. Required gates did not pass, so this slice remains reviewable and uncommitted among prior workspace changes.

## Current Run: Typed Per-Blocker Event Pattern

`BecomesBlockedBy(This).OfSubtype(string).WithBlocker(*PermanentBinding)` is a typed `EventPattern` for each new attacker and blocker pair. It uses the existing `EvtCreatureBlockedBy` combat event, checks the blocker subtype at event detection, and saves the blocker's battlefield incarnation in the pending trigger. The binding is instance-local and is not a target. Registration rejects an empty or noncreature subtype, an invalid binding, and a body read that has no producer. The existing combat, event capture, trigger, stack, and clone services supply the execution contract; no second event pipeline was added. This slice does not add Dwarven Soldier's combined blocks-or-becomes-blocked and one-or-more semantics, Battering Ram's delayed destruction or beginning-of-combat banding, or production card wiring.

The new contract tests first failed because `BecomesBlockedBy` was absent. Focused tests cover two blocker captures, subtype exclusion, real combat, blocker departure and return, phasing, source departure and return, stack copies, clone isolation, and declaration validation. The reference example also compiles as a standalone Go file. Battering Ram's complete text was checked in `data/ATQ.json`; Dwarven Soldier's requirement in `data/FEM.json` and its gap record remains broader. Both gap records remain blocked until their other engine dependencies and card wiring are complete. Existing legacy author exports and production callers remain. The next dependency-ready E slice is the blocks-or-becomes-blocked-by-one-or-more grouped pattern after a fresh contract audit.

`go test ./pkg/mage -run TestBlockedByPattern -count=1`, `go build ./pkg/mage/dsl`, and `git diff --check` pass. Runnable packages in `go test ./pkg/mage/... -short -count=1` and `go test ./... -short -count=1` pass, but both full commands, the DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail during setup because `cards/all.go` imports the absent `cards/custom` package. `git ls-tree HEAD cards/custom` is empty. `make lint` also reports absent C metadata for `cmd/pylib/decision_spec_ffi.go`. The required gates did not pass, so this slice remains reviewable and uncommitted with earlier workspace changes.

## Current Run: Per-Creature Blocked-By Event Prerequisite

`core.EvtCreatureBlockedBy` is a shared combat event for each new attacker and blocker pair. It reports the attacker in `SourceID`, the blocker in `TargetID`, and the attacker's event-time controller in `PlayerID`. Normal blocker declaration emits it after minimum-blocker restrictions and required blocks settle. Simulated declarations use the same final-pair check. A creature entering blocking emits it even if the attacker was already blocked. The existing once-per-attacker `EvtCreatureBecomesBlocked` and legacy `EvtDeclaredBlocker` remain separate. This slice adds no typed event pattern, blocker binding, subtype filter, delayed destruction, or production card change.

The contract test first failed because `EvtCreatureBlockedBy` was absent. Focused engine and test-only turn tests now pass with `-count=1`. They cover two blockers, pair identity, an existing pair, entry blocking, absent attacker, a rejected two-blocker declaration under a three-blocker restriction, source leave and return, and game-clone isolation. Battering Ram's complete text was checked in `data/ATQ.json`; it still needs a typed per-Wall pattern, captured blocker identity, delayed destruction, beginning-of-combat banding, and card wiring. Dwarven Soldier's text in `data/FEM.json` needs Orc filtering and one-or-more grouping, plus card wiring. Their gap records remain blocked. A fresh search found production `EvtDeclaredBlocker` uses in five card files and one `BlocksOrBecomesBlockedTrigger` use; those legacy paths remain. No production card uses the new event. The next dependency-ready E slice is a typed per-blocker pattern with event-time blocker capture and subtype filtering, after its instance-local binding contract is audited.

`go test ./pkg/mage -run TestBlockedByCreatureEvent -count=1`, the focused test-only turn test, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. The runnable packages in `go test ./pkg/mage/... -short -count=1` and `go test ./... -short -count=1` pass, but both full commands stop during DSL and AI package setup because `cards/all.go` imports `cards/custom`, which is absent from the workspace and `HEAD`. The required DSL registration run, `go build ./...`, `go vet ./...`, and `make test` stop on the same import. `make lint` stops on that import and absent C metadata in `cmd/pylib/decision_spec_ffi.go`. These gates did not pass, so this slice is reviewable but uncommitted among earlier workspace changes.

## Current Run: Typed Source Becomes Blocked Event

`BecomesBlocked(This)` is a sealed `EventPattern` for `WhenEvent` and `WhenEventModes` on battlefield sources. It uses the existing `EvtCreatureBecomesBlocked` transition event and shared pending-trigger path. One legal declaration with multiple blockers produces one trigger; a creature entering blocking an unblocked attacker also qualifies. The pattern does not expose blocker identity or a subtype filter. Its trigger keeps the event-time controller and source incarnation when the source leaves, returns, or phases; stack copies and game clones have independent resolution. Registration and inspection use the existing event-pattern checks. No production card was changed.

The focused tests first failed because `BecomesBlocked` was missing. `TestSourceCombatEvent*`, `TestSourceBecomesBlockedAbsentOrPhasedBeforeEvent`, and `TestDeclaredBlockersEmitOneBecomesBlockedEvent` now pass with `-count=1`; `go build ./pkg/mage/dsl` and `git diff --check` pass. The Antiquities Battering Ram text in `data/ATQ.json` also requires a per-Wall blocker reference and delayed destruction, so its gap remains blocked and its card wiring remains open. The Dark still uses `BlocksOrBecomesBlockedTrigger`; that legacy export remains. The next dependency-ready E work is a filtered per-blocker event with event-time participant identity after its shared event contract is audited.

`go test ./pkg/mage/... -short -count=1` passes runnable packages but fails DSL and AI package setup. The DSL registration command, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop because `cards/all.go` imports the absent `cards/custom` package, also absent at HEAD. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. Since `make test` and `make lint` did not pass, this slice is reviewable but uncommitted alongside earlier workspace changes.

## Previous Slice: Source Becomes Blocked Event Prerequisite

`core.EvtCreatureBecomesBlocked` reports the attacking permanent once when it first becomes blocked. Normal blocker declaration emits the event after restrictions and required blocks settle; simulated blocker declaration and a creature entering blocking use the same transition contract. The existing per-pair `EvtDeclaredBlocker` and production callers remain intact. Focused synthetic tests cover two simultaneous blockers, an already blocked attacker, entry blocking, game-clone isolation, normal turn integration, and a two-blocker declaration removed by a three-blocker minimum. This prerequisite added no participant binding, production card, or gap record change. Battering Ram, Dwarven Soldier, and the Legends blocks-or-becomes-blocked cards still need other typed patterns and dependencies; their card wiring remains open.

The focused tests first failed because the event constant was absent and now pass. `go test ./pkg/mage/... -short -count=1` passes its runnable packages but fails DSL and AI package setup because `cards/all.go` imports `cards/custom`, which is absent from the workspace and HEAD. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop on the same import. `make lint` stops on that import and missing C metadata in `cmd/pylib/decision_spec_ffi.go`. These gates did not pass, so this slice remains reviewable and uncommitted alongside earlier workspace changes.

## Current Run: Creature-Death Color Filter

`CreatureDies().WithoutColor(core.Color)` and the other creature-death patterns now filter the dying creature by its color immediately before departure. The filter uses the existing zone-change event and object/LKI view. It composes with `.Nontoken()` and the existing event captures in `WhenEvent` and `WhenEventModes` on battlefield permanents. Registration rejects non-Magic colors and repeated color filters. No production card was changed.

Necropolis of Azar's printed requirement was checked in `cards/astral/PAST.json`. The filter supplies its nonblack death condition, including multicolored black creatures and pre-departure color changes. Independent random token power and toughness and card wiring remain open; its JSON gap stays blocked. Synthetic tests cover colorless, mono- and multicolored creatures, continuous color changes, simultaneous source death, phasing, exile exclusion, leave and return, clone isolation, and registration. The next dependency-ready E slice is a death-event card reference with zone-change identity, or the source-becomes-blocked event prerequisite after a fresh audit.

The new tests first failed because `WithoutColor` was absent. Focused `TestCreatureDiesWithoutColor` tests, `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, `go test ./cards/astral -short -count=1`, and `git diff --check` pass. Runnable packages in `go test ./pkg/mage/... -short -count=1` and `go test ./... -short -count=1` pass, but the full commands and the DSL registration run stop during setup because `cards/all.go` imports `cards/custom`, which is absent at `HEAD`. `go build ./...`, `go vet ./...`, and `make test` fail for the same reason. `make lint` also reports absent C metadata in `cmd/pylib/decision_spec_ffi.go`. The required gates did not pass, so this slice remains reviewable and uncommitted alongside earlier workspace changes. No production card uses `WithoutColor`; existing death-event callers and exports remain for separate card wiring.

## Current Run: Creature-Death Owner Capture

`CreatureDies().WithOwner(*PlayerBinding)` and the other creature-death patterns save the dying creature's owner at event detection. The owner comes from the shared event-time object/LKI view and enters the trigger's instance-local player frame. It is distinct from `.WithController` and can combine with that capture and `.WithToughness`. Registration rejects a missing or empty binding and using the same player binding for owner and controller. The public constructor remains `WhenEvent` or `WhenEventModes` on a battlefield permanent; the existing death filters and legacy exports remain. No production card was changed.

Personal Incarnation's owner-at-death requirement was checked against `data/4ED.json`; its half-life arithmetic, damage replacement, activation restriction, and card wiring remain separate work. No JSON gap record is closed by this adapter. Synthetic tests cover owner versus controller, two simultaneous events, source death, phasing, stack copy, clone isolation, and declaration scope. The next dependency-ready E slice is a death-event card reference with zone-change identity, or the source-becomes-blocked event prerequisite after a fresh audit.

The new tests first failed because `WithOwner` did not exist. Focused `TestCreatureDiesOwnerCapture` and `go build ./pkg/mage/dsl` pass; the runnable engine and card packages in `go test ./pkg/mage/... -short -count=1` and `go test ./... -short -count=1` pass. The DSL registration run, full build, vet, and `make test` stop at package setup because `cards/all.go` imports the absent `cards/custom` package, which is also absent at `HEAD`. `make lint` stops on that import and missing C metadata in `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. Required full gates did not pass, so this slice remains reviewable and uncommitted alongside earlier workspace changes.

## Current Run: Paid Sacrifice Mana Value

`SacrificeMatchingCostWithManaValue(filter, text, binding)` captures the mana value of one chosen permanent sacrificed as an activation cost or spell additional cost. It reuses the locked sacrifice transaction and its instance-local number frame; the source is excluded. The binding is available to later resolving effects, including copied stack objects. Registration rejects missing text or binding; a zero-value filter matches all permanents. This slice does not add mana planning for a dynamic mana ability, chosen-color mana, resolution-time payment outputs, or production card wiring.

Priest of Yawgmoth in `data/ATQ.json` and Metamorphosis in `data/ARN.json` require a sacrificed object's mana value. Both need other adapters before migration: Priest needs dynamic mana-ability planning, and Metamorphosis needs chosen-color, creature-spell-restricted mana. Their gap records remain blocked. Existing sacrifice costs and legacy exports remain for unmigrated callers. The next dependency-ready P work is another distinct cost output after a fresh audit, or T recorded target groups.

The new tests failed first on the missing constructor. Focused `TestPaidSacrificeManaValue|TestPaidSacrificeStats|TestSpellPaidSacrifice` now passes with `-count=1`; the public DSL builds, `go vet ./pkg/mage` and `git diff --check` pass, and the wrong-binding compile case fails with the expected type error. `go test ./pkg/mage/... -short -count=1` passes its runnable packages but fails DSL and AI setup. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` confirms it is absent at HEAD. `make lint` also reports absent C metadata for `cmd/pylib/decision_spec_ffi.go`. Full validation has not passed, so this slice remains uncommitted. A fresh search found no production caller of the new cost; Priest of Yawgmoth and Metamorphosis still need card wiring after their other engine dependencies.

## Current Run: Creature Entry Subtype Filter

`AnotherCreatureYouControlEnters().OfSubtype(string)` filters the existing entry event by the entering creature's current subtype. It combines with `.Nontoken()` and `.WithPermanent(...)`, and works in `WhenEvent` and `WhenEventModes` on battlefield permanents. The event uses the shared entry and trigger services; no new event storage is needed. Registration rejects empty subtypes and known subtypes from another family. Synthetic tests cover matching and nonmatching subtype, token and opponent exclusion, phased source, clone isolation, and declaration errors. Lathliss's full Oracle text was checked in `data/JMP.json`; its token creation and boost ability, and all production wiring, remain card work. Existing legacy event exports remain. The next dependency-ready E slice is a death-event permanent binding or another event participant, after a fresh requirements audit.

The focused tests, `go build ./pkg/mage/dsl`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes its runnable packages but fails DSL and two AI package setup steps. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. The required full gates did not pass, so this slice remains reviewable and uncommitted among earlier workspace changes. No JSON gap record is closed by adapter availability alone.

## Current Run: Entering Creature Permanent Binding

`AnotherCreatureYouControlEnters().WithPermanent(*PermanentBinding)` saves the entering creature's battlefield incarnation at event detection. `From(binding)` can use it in the same trigger body if that incarnation remains present and phased in. The pattern works with `WhenEvent` and `WhenEventModes` on battlefield permanents, and with the existing `.Nontoken()` filter. It uses the shared event capture and resolution frame; no new global result state or card-specific behavior is added. Registration rejects nil, empty-name, or unproduced bindings. The captured permanent is not a target. Broader entry filters, death-event object bindings, and card wiring remain outside this slice.

The focused test first failed because `WithPermanent` did not exist. It now covers controlled versus opposing entry, ownership versus control, leave and return, phasing, clone isolation, source departure, stack copying, and registration validation. Existing entry filters and trigger lifecycle remain intact. No production card uses the new binding; legacy event exports and all card wiring remain. Soul of the Harvest and Lathliss requirements in `data/JMP.json` were checked; they do not require the new object binding, so no gap record is closed by it. The next dependency-ready work is an event-time subtype entry filter or a death-event permanent binding after a fresh requirement audit.

Focused tests, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes its runnable packages but fails setup in DSL and two AI packages. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` also fails on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. The required `make test` and `make lint` gates did not pass, so this slice is reviewable and uncommitted among earlier workspace changes.

## Current Run: Filtered Controlled Creature Entry

`AnotherCreatureYouControlEnters().Nontoken()` is a sealed battlefield `EventPattern` for `WhenEvent` and `WhenEventModes`. It reuses the existing battlefield entry event and trigger lifecycle. Creature type, controller, and token status are checked on the entering permanent when the event is detected; the trigger source is excluded. No event participant binding or subtype filter is added. Synthetic tests first failed because the constructor was absent. They now cover controlled and opposing entries, owner versus controller, token and noncreature exclusion, self entry, phasing, clone isolation, inspection, source leave and return, and stack-copy controller identity. Soul of the Harvest's complete entry condition was checked against `data/JMP.json`. Its optional draw and production wiring remain card work; this adapter does not complete its gap record. Legacy callers remain exported. The next E work is subtype or characteristic entry filters and event participant bindings, after a fresh requirement audit.

Focused tests, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes its runnable packages but fails DSL and two AI package setup steps. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. The required `make test` and `make lint` gates did not pass. This slice remains reviewable and uncommitted alongside earlier workspace changes.

## Current Run: Nontoken Other-Creature Death Pattern

`AnotherCreatureDies().Nontoken()` adds an event-time nontoken filter to the existing creature-death pattern. `AnotherCreatureDies()` excludes the trigger source without restricting the dying creature's controller. The filter reads pre-departure token status through the shared event/LKI view. Both declarations are valid in `WhenEvent` and `WhenEventModes` on battlefield permanents. Existing patterns and legacy callers remain. No production card was changed.

Harvester of Souls and Ogre Slumlord in `data/JMP.json` require another nontoken creature to die; their optional effect bodies and other abilities remain card wiring. The new test covers token exclusion, both controllers, self exclusion, simultaneous deaths, phasing, source return, clone isolation, and inspection. This slice does not add entry-event filters, subtype filters, event-permanent bindings, or card migrations. No gap record is closed by an available adapter alone. The next E slice is a filtered entry-event pattern with event-time characteristics, after a fresh contract audit.

The focused tests first failed because `AnotherCreatureDies` was absent and now pass with `-count=1`. `go build ./pkg/mage/dsl` and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes all runnable packages but fails package setup for DSL and two AI packages. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail setup because `cards/all.go` imports the absent `cards/custom` package, which is also absent at `HEAD`. `make lint` fails on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. The required `make test` and `make lint` gates did not pass, so this slice is reviewable and uncommitted. Earlier workspace changes remain in place.

## Current Run: Source Blocking State Predicate

`SourceBlockingCondition()` is a sealed `StateCondition` for a permanent that is blocking now. The shared combat group now records the blocking battlefield incarnation, and the predicate compares it with the current source. Phasing removes combat membership. Departure retains the old relationship for leave triggers, even if the same card returns; an absent or returned source does not satisfy the predicate for the old block. An attacker remains blocked after its blocker leaves. The condition is available in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, and `IfState` on permanent abilities; registration rejects a spell body. It does not supply attacked-or-blocked history, blocker identity, or a becomes-blocked event.

Synthetic tests first failed because the constructor was absent, then exposed stale combat membership after a leave and return. They now cover present and absent membership, phasing, leave and return, combat removal, clone isolation, static application, trigger resolution, intervening-if recheck, inspection, and registration. The Abu Ja'far card regression showed that departure must retain combat LKI until its death trigger resolves; its focused card test passes with incarnation-aware membership. Clockwork Beast and Clockwork Avian in `data/LEA.json` and `data/ATQ.json` need attacked-or-blocked-this-combat history and other counter behavior, so this predicate does not resolve their gap records. No production card was changed. A fresh caller search found no production `SourceBlockingCondition` use; legacy combat predicates and card wiring remain. The next dependency-ready work is a bound-permanent tapped predicate after instance-local state context and scope validation, or the full becomes-blocked event transition prerequisite.

Focused `TestSourceBlockingCondition|TestCombatDeparturePreservesAttackerBlockedStatus` and `TestAbuJafar` pass with `-count=1`. `go test ./pkg/mage -count=1`, `go build ./pkg/mage/dsl`, a standalone public DSL example build, `go vet ./pkg/mage`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes every runnable package but stops at setup for DSL and two AI packages. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop at setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. The required `make test` and `make lint` gates did not pass, so the slice remains reviewable and uncommitted. Earlier uncommitted work and unrelated files remain in place.

## Previous Slice: Source Attack History State Predicate

`SourceAttackedThisTurnCondition()` is a sealed `StateCondition` for a permanent that was declared as an attacker this turn. The existing turn tracker now also stores the attacking battlefield incarnation. The typed predicate rejects an absent, phased-out, or returned source and reads the same history in game clones. It is available in static, trigger, activation, intervening-if, and resolving condition contexts on permanent abilities; registration rejects it in a spell body. The older card-ID history query and its callers remain unchanged.

Synthetic tests cover before and after attack, clone isolation, phasing, absence, leave and return, turn reset, trigger resolution, inspection, and registration. Erg Raiders' full instruction was checked in `data/ARN.json`; this predicate supplies only its attack-history clause. Its control-time exception, conditional damage, and production card wiring remain open. No JSON gap record is resolved. The bound-permanent tapped predicate still needs instance-local state context and scope validation; it remains the next S slice. No production card was changed.

Focused `TestSourceAttackedThisTurn` tests, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. The runnable packages in `go test ./pkg/mage/... -short -count=1` pass, but DSL and two AI packages stop at setup. The required DSL registration run, `go build ./...`, `go vet ./...`, and `make test` stop at setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` stops on the same import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. This slice is reviewable and uncommitted because `make test` and `make lint` did not pass. The earlier uncommitted work and unrelated files remain in place.

## Previous Slice: Source Tapped State Predicate

`SourceTappedCondition()` and `SourceUntappedCondition()` are sealed `StateCondition` predicates for the current battlefield source. They work in static, trigger, activation, intervening-if, and resolving condition contexts. Both return false for an absent or phased-out source; negating the tapped predicate is not an equivalent untapped check. A captured resolution or static source incarnation prevents a returned card from satisfying the old condition. The older `SourceIsTapped` trigger condition and `SourceTapped` continuous-effect condition keep their signatures.

Synthetic tests cover tapped transitions, static power/toughness application, intervening-if recheck, clone isolation, phasing, absence, leave and return, and registration rejection in a spell body. Giant Tortoise in `data/ARN.json` and Howling Mine in `data/4ED.json` supply the complete acceptance text; neither card was modified. Their other declaration and card wiring needs remain open. The Arabian and Fourth Edition inventories still describe the older surface and are not card-completion evidence. No JSON gap record is resolved by this predicate alone. The next dependency-ready S slice is a typed bound-permanent tapped predicate with scope validation, after a fresh audit.

Focused `TestSourceTapped` tests, `go build ./pkg/mage/dsl`, a standalone public DSL example build, and `git diff --check` pass. The runnable engine packages pass `go test ./pkg/mage/... -short -count=1`. The required DSL registration run, `go build ./...`, `go vet ./...`, `go test ./... -short -count=1`, and `make test` fail at setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` fails on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. This run is reviewable but is not fully validated or committed. Earlier uncommitted work and unrelated files remain in place.

## Previous Slice: Spell Sacrifice Payment Characteristics

`SacrificeCreatureCostWithStats(power, toughness *NumberBinding)` now carries the chosen creature's payment-time power and toughness into a spell's instance-local stack frame. `FromNumber` can read either value in `Spell` and `SpellModes` bodies. Normal and alternative-cost casting use the same locked payment transaction. Stack copies and game clones retain independent frames. Registration validates card-level additional costs before spell bodies, including nil and duplicate binding checks; resolution-time `IfPaid` still rejects this capture. The export and legacy cost callers remain compatible.

Synthetic spell tests cover normal and alternative-cost casting, modal bodies, changed characteristics, two pending spells, copies, clones, absent creature rollback, and declaration failures. Fling and Momentous Fall requirements were checked against `data/JMP.json`; neither card was changed. A fresh card search found no production `SacrificeCreatureCostWithStats` callers; legacy matching-sacrifice costs remain in use. Their card wiring and any other card-specific dependencies remain open. No JSON gap record is marked complete. The next dependency-ready P slice is a paid-object reference or broader payment quantities after a fresh audit.

Focused `TestSpellPaidSacrifice|TestPaidSacrificeStats` tests, `go build ./pkg/mage/dsl`, and `go vet ./pkg/mage` pass. `go test ./pkg/mage/... -short -count=1` passes in runnable packages but fails setup in DSL and two AI packages. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` fail setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` fails on that import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. The required checks did not pass, so this slice remains reviewable and uncommitted. The pre-existing uncommitted work remains in place.

## Previous Slice: Activation Sacrifice Payment Characteristics

`SacrificeCreatureCostWithStats(power, toughness *NumberBinding)` uses the locked single-creature sacrifice cost and captures current power and toughness from the selected live incarnation immediately before payment. After successful activation payment, both numbers enter the activation's instance-local stack frame. `FromNumber` reads them in `Activate` and `ActivateModes` bodies. The cost still excludes the source, asks the payer once, and rejects a returned, phased-out, absent, or ineligible choice. Registration rejects nil or duplicate bindings; the role types reject non-number arguments at compile time. Existing `SacrificeCreatureCost` and other legacy callers remain unchanged.

Synthetic tests cover modified characteristics, chosen identity, failed commit after leave and return, two pending activations, stack-copy and clone isolation, and declaration checks. Diamond Valley's complete text in `data/ARN.json` and Life Chisel's complete text in `data/LEG.json` motivate this activation output. Fling's spell additional cost in `data/JMP.json` still needs a spell-stack capture path, so this slice does not resolve its gap record. No production card was changed; Diamond Valley and Life Chisel still need card wiring. A fresh search found 53 production occurrences of the four matching-sacrifice cost constructors and no new-cost callers; all legacy exports remain. The next dependency-ready slice is the spell additional-cost payment output, followed by broader payment objects and quantities.

Focused `TestPaidSacrificeStats|TestSacrificePayment` tests, `go test ./pkg/mage -short -count=1`, and `go build ./pkg/mage/dsl` pass. A direct compile check of the wrong-binding fixture fails for the intended Go type error. `go test ./pkg/mage/... -short -count=1` passes in runnable packages but stops at DSL and two AI packages. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` stop at package setup because `cards/all.go` imports the absent `cards/custom` package, also absent at HEAD. `make lint` stops on that import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. The required checks did not pass, so this slice remains reviewable and uncommitted. Earlier uncommitted work remains untouched.

## Previous Slice: Locked Sacrifice Payment Choices

Matching sacrifice costs now ask the payer once during transaction preparation, validate the choices on the isolated game, and retain their battlefield incarnations for commit. A different permanent cannot be substituted during live payment. Multi-permanent costs check all retained choices before sacrificing any of them. The shared payment transaction and `DoSacrifice` service remain the executors. Existing `SacrificeCreatureCost`, `SacrificeMatchingCost`, `SacrificeMatchingIncludingSourceCost`, and `SacrificeNMatchingCost` signatures and production callers remain unchanged. No typed paid-permanent output is exposed in this prerequisite slice.

Synthetic contract tests first failed because the clone's choice differed from live payment and because a returned permanent could be paid in place of the chosen incarnation. They now cover the payer's single choice, leave and return, phasing, multi-object payment, failure without a partial sacrifice, invalid choices, failed total-cost validation, and a test-only `Activate` declaration. Diamond Valley's complete text in `data/ARN.json` and Fling's complete text in `data/JMP.json` require a later typed characteristic output; neither card was changed. No gap record is resolved by choice locking. A fresh search found 53 production occurrences of the four matching-sacrifice cost constructors; they remain exported. The next dependency-ready slice is the instance-local paid-permanent characteristic output from the prepared payment transaction.

Focused `TestSacrificePayment|TestMatchingSacrificeCost` tests pass. `go test ./pkg/mage/... -short -count=1` passes in runnable packages but stops at DSL and two AI packages because `cards/all.go` imports the absent `cards/custom` package. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` stop at the same package setup error. `make lint` stops at that import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. The slice remains reviewable and uncommitted because the required `make test` and `make lint` checks cannot pass. Unrelated working-tree changes remain untouched.

## Previous Slice: Resolving Sacrifice Action Result

`Sacrifice(PermanentRef, ...SacrificeResultOption)` now accepts `SacrificeAmountInto(*NumberBinding)`. It writes 1 when the resolving controller actually sacrifices the present battlefield incarnation and 0 for an absent, phased-out, returned, or differently controlled permanent. The existing one-argument call remains valid. `FromNumber` reads the result in later instructions of the same resolving spell, activation, or trigger. Registration checks nil or duplicate options, binding names, and producer dominance. The shared `DoSacrifice` service still performs the action; the result uses instance-local number bindings and the existing clone/copy frame path.

Synthetic tests cover successful and impossible sacrifices, ownership versus control, phasing, leave-and-return, stale result clearing, stack copies, clone isolation, and invalid declarations. This is the action-result part of P, not a cost-payment capture. Diamond Valley, Life Chisel, and Fling still need a typed chosen-cost-object characteristic binding; Serendib Djinn still needs its complete optional sacrifice and condition wiring. No production card or gap record is complete because of this adapter alone. The next dependency-ready slice is a sacrifice payment output attached to the prepared payment transaction, after a fresh caller and contract review.

Focused `TestSacrificeAmountResult` tests, `go test ./pkg/mage -short -count=1`, `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1` passes in runnable packages but stops at setup in DSL and two AI packages. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop at setup because `cards/all.go` imports the absent `cards/custom` package. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `git ls-tree HEAD cards/custom` is empty. This slice is reviewable but not fully validated or committed. Existing uncommitted work remains untouched.

## Previous Slice: Typed Per-Action Damage Results

`BindDamageResults`, `DamageResultsInto`, `BindDamageResult`, and `ForEachDamageResult` expose each positive final action of `DealDamageEach` after the complete batch. `DamageResultAmount`, `DamageResultPlayer`, `DamageResultPermanent`, and `DamageResultSource` read its actual amount, final recipient, and captured source. The collection and iteration member live in the resolution frame, which is copied for game clones. Iteration effects run sequentially after damage, while the damage instruction remains simultaneous. Registration validation checks producers, scope, names, and duplicate or nil options. The typed option composes with `DamageTotalInto`. The existing public mixed-damage and legacy damage signatures remain intact.

Synthetic tests cover redirection, prevention, source departure and lifelink, zero damage clearing, player and permanent result roles, returned target exclusion, active-frame clone isolation, and invalid declarations. Inferno and Earthquake requirements were checked against `data/4ED.json`; their production wiring remains separate. No JSON card gap is resolved solely by this adapter. A fresh search found no production callers of the typed batch or result exports; the legacy damage exports remain. Next dependency-ready work is P payment outputs after a fresh review, or a distinct D target-group/recipient-history prerequisite when needed by a card.

Focused `TestDamageResults|TestDealDamageEach` tests, runnable engine package tests, a standalone DSL example, `go build ./pkg/mage/dsl`, and `go vet ./pkg/mage` pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop at package setup because `cards/all.go` imports the absent `cards/custom` package. `git ls-tree HEAD cards/custom` is empty. `make lint` stops on that import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. This slice is reviewable but uncommitted because the required full checks cannot pass. The working tree already held earlier uncommitted slices and unrelated files; these remain in place.

## Previous Slice: Mixed Damage Outcome Contract

The internal `Game.dealSimultaneousDamageOutcomes(...DamageAssignment) []damageOutcome` reuses the mixed damage batch and reports each final positive damage action with its actual amount, player or permanent incarnation, and captured source. Prevention and absent recipients have no result. Redirection changes the recorded recipient. The existing `Game.DealSimultaneousDamage` public signature and `DealDamageEach` total option remain compatible; the total is now the sum of the returned outcomes. The batch still dispatches events and callbacks after all damage actions. No typed per-recipient collection binding is exposed in this slice.

Synthetic contract tests cover redirection, full prevention, source departure and LKI, recipient return, phasing, clone isolation, and the existing mixed-batch timing and total contracts. Inferno's full damage instruction was checked in `data/4ED.json`. Inferno, Earthquake, and other production callers still need card wiring. No gap record is resolved by this internal contract. The next dependency-ready slice is a typed per-recipient result binding over these outcomes, after a fresh API and caller review; P payment outputs remain independent work.

The focused mixed-batch and typed damage tests pass. `go test ./pkg/mage/... -short -count=1` passes in runnable packages but stops at setup for DSL and two AI packages. The DSL registration run, `go build ./...`, `go vet ./...`, and `make test` stop at setup because `cards/all.go` imports the absent `cards/custom` package. `make lint` stops on the same import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. This slice is reviewable and uncommitted because `make test` and `make lint` did not pass. No production card changed.

## Previous Slice: Mixed Damage Aggregate Result

`DealDamageEach(source, recipients, amount, DamageTotalInto(binding))` saves the sum of damage actually dealt by the instruction in a `NumberBinding`. The batch executor returns this sum from its final executed actions while retaining the existing public `Game.DealSimultaneousDamage` contract. Replacement, redirection, and prevention affect the sum; zero damage and an absent source write zero. The binding is local to the resolution frame, available to later `FromNumber` expressions in `Spell`, `Activate`, and trigger bodies, and copied independently by game clones. Registration validation rejects nil or duplicate result options and reads without a dominating producer. The typed option does not expose per-recipient outcomes or change the legacy damage API.

Synthetic tests cover mixed player/creature damage with prevention, redirection, zero and absent-source clearing, returned target exclusion, clone isolation, and declaration validation. The underlying mixed batch tests cover recipient incarnation, phasing, and event ordering. The first batch requirement remains Inferno's exact text in `data/4ED.json`: “Inferno deals 6 damage to each creature and each player.” Inferno and Earthquake still need production card wiring. A fresh search found no production `DealDamageEach` or `DamageTotalInto` callers; the legacy damage exports remain. No JSON gap record is marked complete by this result binding. The next dependency-ready work is a P payment-output slice or per-recipient damage results after a fresh dependency review.

`go test ./pkg/mage -run 'TestDealDamageEach|TestMixedDamageBatch' -count=1`, `go test ./pkg/mage -short -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, the standalone DSL example compile, and `git diff --check` pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop at package setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` also stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. This slice is reviewable and uncommitted because `make test` and `make lint` did not pass. No production card changed.

## Previous Slice: Typed Mixed Damage Adapter

`DamageRecipients(PlayerSet, PermanentSet)` and `DealDamageEach(DamageSourceRef, DamageRecipientSet, ResolutionValue)` expose one simultaneous damage instruction over players and permanents. Both set inputs are required at registration. The source and amount are read once, and the existing `Game.DealSimultaneousDamage` service handles replacements, prevention, execution, and queued notifications. A targeted `PermanentTargetsBinding` with a maximum of one can supply the permanent set; its captured battlefield incarnation excludes a returned or phased-out target. Larger target groups are rejected at registration because their selection boundaries are not recorded yet. The adapter has no aggregate or per-recipient result binding and no general player target set. It is valid in resolving spell, activation, and trigger bodies.

Focused synthetic tests cover a mixed player/creature batch observed only after all results, prevention, phased permanents, lifelink and clone isolation, absent source, target leave-and-return, and registration errors for absent sets and undeclared targets. The first requirement is Inferno's exact text in `data/4ED.json`: “Inferno deals 6 damage to each creature and each player.” Earthquake also needs a filtered creature set and each player. Their production cards still use legacy paths; no card or JSON gap record is marked complete. No production caller uses `DealDamageEach`; the legacy damage exports remain. The next dependency-ready work is typed per-recipient or aggregate damage results, or the next P payment-output slice after a fresh dependency review.

`go test ./pkg/mage -run TestDealDamageEach -count=1`, `go test ./pkg/mage -short -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, the standalone DSL example compile, and `git diff --check` pass. The required engine short run, DSL registration run, full short run, `go build ./...`, `go vet ./...`, and `make test` stop at setup because `cards/all.go` imports the absent `cards/custom` package; `git ls-tree HEAD cards/custom` is empty. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. This slice remains reviewable and uncommitted because `make test` and `make lint` do not pass. No production card changed.

## Previous Slice: Mixed Damage Batch Prerequisite

`Game.DealSimultaneousDamage(...DamageAssignment)` accepts player and permanent assignments built with `PlayerDamageAssignment` and `PermanentDamageAssignment`. Each assignment carries a captured `DamageSource`. The batch prepares damage through the existing replacement and prevention pipeline, applies the resulting actions, and then dispatches damage events and callbacks. A permanent assignment captures its battlefield incarnation; a returned or phased-out recipient is skipped. The engine API is for engine and integration code. It does not yet declare a typed DSL recipient set or bind per-recipient or aggregate results.

Synthetic tests cover damage to both players and a creature in one event, a prevention shield, observers seeing all final results, source lifelink in a clone, and recipient leave-and-return and phasing. Inferno and Earthquake in `data/4ED.json` require mixed player/creature damage; their production wiring remains unchanged. Existing singular `DealDamageFrom` and legacy damage exports remain. The next dependency-ready work is a typed recipient-set declaration and `DealDamageEach` adapter built on this batch, with result and target legality contracts.

`go test ./pkg/mage -run TestMixedDamageBatch -count=1`, the runnable engine package short tests, `go vet ./pkg/mage`, `go build ./pkg/mage/dsl`, and `git diff --check` pass. The required full engine short run, DSL registration run, full short run, `go build ./...`, `go vet ./...`, and `make test` stop at package setup because `cards/all.go` imports the absent `cards/custom` package. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `cards/custom` is absent at HEAD. The slice remains uncommitted because `make test` and `make lint` do not pass. No card or JSON gap record is marked complete.

## Previous Slice: Attached Creature Death Event

`AttachedCreatureDies()` is a sealed battlefield event pattern for `WhenEvent` and `WhenEventModes`. It matches only the creature attached to the source at death, checks the host's battlefield incarnation, and composes with `WithController` and `WithToughness`. It reuses the zone-change observer and instance-local event captures. Focused synthetic tests cover an unrelated death, simultaneous Aura and creature departure, changed toughness, former controller distinct from owner, clone and stack-copy isolation, phasing, noncreature and exile exclusions, a returned host, and registration validation. The first complete acceptance example is Creature Bond's trigger and damage instruction, checked against `data/4ED.json`; production wiring and its existing printed-toughness/owner approximation remain separate card work. No card or JSON gap record is marked complete. The existing `DealDamageFrom` adapter supplies singular damage, while mixed simultaneous damage still needs a shared batch contract. The next dependency-ready slice is that damage batch prerequisite.

`go test ./pkg/mage -run TestAttachedCreatureDies -count=1`, `go test ./pkg/mage -short -count=1`, `go build ./pkg/mage/dsl`, `go vet ./pkg/mage`, and `git diff --check` pass. The required engine short run passes in runnable packages but stops for DSL and AI packages at setup. The DSL registration run, full short run, `go build ./...`, `go vet ./...`, and `make test` stop because `cards/all.go` imports the missing `cards/custom` package, which is also absent at HEAD. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. This slice remains reviewable and uncommitted because `make test` and `make lint` cannot pass. A fresh production search found no `AttachedCreatureDies` or `DealDamageFrom` callers; Creature Bond still uses its legacy callback.

The standalone `dsl` authoring example in `docs/dsl-reference.md` also compiles.

## Previous Slice: Creature-Death Event Toughness Capture

`CreatureDies()`, `CreatureYouControlDies()`, and `AnotherCreatureYouControlDies()` now support `.WithToughness(*NumberBinding)`. `FromNumber` reads the dying creature's toughness in the trigger body. Detection captures the pre-departure value from the batch observer or departure LKI, including continuous effects. The pending trigger stores the number before stack placement, and the instance-local resolution frame keeps it through cleanup and a same-card return. This reuses the existing zone-change trigger path and numeric event-capture effect; no new event or damage executor is added. `.WithController` composes with this method. The pattern is an event declaration for battlefield permanents, not a target, a general death filter, or a permanent reference to the dead creature.

Focused synthetic tests cover a same-card return after LKI cleanup, a source's own death, simultaneous deaths with distinct toughness, continuous effects, a copied trigger under another controller, game clone isolation, phased-out sources, noncreature departures, invalid nil or duplicate bindings, and a read outside the producing ability. The first card requirement is Creature Bond's pre-death toughness, checked against `data/4ED.json`; its attached-creature death filter was added in the subsequent slice, and its production card is unchanged. A fresh search found ten production uses of the typed creature-death patterns and `DiesCreatureTrigger`, and no production `.WithToughness` calls. No gap record was resolved by this narrower addition.

The focused tests, standalone authoring-example compile, `go test ./pkg/mage -short -count=1`, `go build ./pkg/mage/dsl`, and `go vet ./pkg/mage` pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail at package setup because `cards/all.go` imports the absent `cards/custom` package; it is absent at HEAD. `make lint` also reports absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. The required checks are blocked, so this slice remains reviewable and uncommitted.

## Previous Slice: Creature-Death Event Controller Capture

`CreatureDies()`, `CreatureYouControlDies()`, and `AnotherCreatureYouControlDies()` now return a sealed creature-death pattern with `.WithController(*PlayerBinding)`. `WhenEvent` and `WhenEventModes` can save the dying creature's controller before departure, then read it with `FromPlayer` during resolution. The pattern reuses the existing battlefield-to-graveyard event, its pre-departure `PlayerID`, the event-capture effect, and the instance-local resolution frame. It does not add a general event filter, a permanent reference to the dead creature, or a land-death pattern. The concrete return type enables the method without changing existing `EventPattern` callers.

The test-only watcher covers an opposing owner's creature under your control, an opposing controlled creature, source death in the same batch, two simultaneous deaths with distinct controllers, phasing, stack-copy controller changes, clone isolation, and invalid or non-dominating bindings. The first inventory cases that need this kind of participant are Dingus Egg and Creature Bond, but both still require additional event filters and Creature Bond also needs pre-death characteristics; no production card was changed. A fresh search found ten production calls across the three typed creature-death patterns and `DiesCreatureTrigger`; the legacy export remains. No gap record is marked resolved by this narrower adapter.

The focused `TestCreatureDies.*Controller` tests, the standalone authoring example compile, and the runnable packages in `go test ./pkg/mage/... -short -count=1` pass. The full engine run, DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail during package setup because `cards/all.go` imports the absent `cards/custom` package, also absent at HEAD. `make lint` fails on that import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. Required validation is blocked, so this slice remains reviewable and uncommitted. The next dependency-ready slice is simultaneous damage preparation and execution, or another small E event participant contract.

## Previous Slice: Actual Damage-Source Result

`BindDamageSource(name)`, `DamageSourceInto(binding)`, and `FromDamageSource(binding)` capture the final executed action's source when one `DealDamageFrom` instruction deals positive damage. A later instruction in the same resolution can deal damage from that captured source, including after its permanent leaves and returns. Zero, prevented, or absent damage clears a prior binding. The result combines with existing amount and recipient options. The binding has a sealed source role, declaration inspection, unique option validation, scope checks, and independent clone storage. Synthetic tests cover departed-source LKI, old controller and lifelink, spell source, redirection, prevention and stale clearing, clone isolation, and invalid declarations. Focused tests, `go build ./pkg/mage/dsl`, and `go vet ./pkg/mage` pass. No production card changed.

The required engine, DSL registration, full short-test, build, vet, and `make test` commands stop during package setup because `cards/all.go` imports the absent `cards/custom` package; that directory is also absent at HEAD. `make lint` stops on the same import and absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. The required checks are blocked, so this slice is reviewable but is not fully validated or committed. A fresh search found legacy damage calls in 29 production card files and no `DealDamageFrom` production calls. Card wiring remains separate. The next dependency-ready slice is a basic event-participant binding or simultaneous damage prerequisite; other source references and general filtered events also remain open.

## Previous Slice: Actual Damaged-Permanent Result

`DealDamageFrom(This, recipient, amount, DamagePermanentInto(binding))` writes the permanent incarnation that actually receives positive damage to an instance-local `PermanentBinding`. It follows redirection from a player to a permanent and clears a prior value for zero, prevented, or absent damage. The result can combine with `DamageAmountInto` and `DamagePlayerInto`; inspection and registration validation enforce declared, unique, in-scope bindings. Synthetic tests cover both redirection directions, actual amount composition, prevention, zero damage, target leave and return, phasing, clone isolation, and invalid declarations. No production card changed. Source-result identity, simultaneous damage batches, and event-participant bindings remain open. A fresh search found legacy damage calls in 29 production card files and no `DealDamageFrom` production calls; card wiring remains separate. The next dependency-ready slice is source-result identity or basic event-participant binding.

Focused `TestDealDamageFrom*` tests and `go build ./pkg/mage/dsl` pass. `go test ./pkg/mage/... -short -count=1` passes its runnable engine packages, but DSL and AI package setup fails because `cards/all.go` imports the absent `cards/custom` package; the directory is also absent at HEAD. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop on the same import. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. Required checks did not all pass, so this slice remains uncommitted. Earlier uncommitted slices and unrelated workspace files remain in place.

## Previous Slice: Actual Damaged-Player Result

`DealDamageFrom(This, recipient, amount, DamagePlayerInto(binding))` writes the player who actually receives positive damage to an instance-local `PlayerBinding`. It clears the binding if damage is prevented, redirected to a permanent, or absent. The option composes with `DamageAmountInto` and reuses the shared replacement and damage execution path. Definition inspection and registration validation reject missing, duplicate, and out-of-scope bindings. Focused synthetic tests cover player damage, redirecting player damage to a permanent and permanent damage to a player, prevention, stale binding clearing, clone isolation, and declaration validation. No production card changed. Modal Test Artifact and other simple damage callers can use the existing typed action in a separate migration. Drain Life and Syphon Soul can use actual amount results but retain other payment or batch prerequisites. Permanent-recipient and source-result identity, batch damage, and event-participant capture remain open. The next dependency-ready slice is permanent-recipient identity or filtered event-participant binding. Existing typed damage and legacy positional damage callers remain for separate card wiring.

Focused `TestDealDamageFrom*` tests pass with `-count=1`, and `go build ./pkg/mage/dsl` passes. `go test ./pkg/mage/... -short -count=1` passes the runnable engine packages, but DSL and AI package setup fails because `cards/all.go` imports the absent `cards/custom` package; the package is also absent at HEAD. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop on the same import. `make lint` stops on that import and missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. Required checks did not all pass, so this slice remains uncommitted. Existing workspace changes from earlier slices remain in place.

## Previous Slice: Singular Damage Amount Result

`DealDamageFrom(This, recipient, amount, DamageAmountInto(binding))` writes the amount actually dealt by that instruction to a `NumberBinding` in the current resolution frame. The shared damage executors report their post-replacement, post-prevention amount; redirected damage retains the captured source. Zero or absent damage writes zero. Definition inspection and registration validation cover the optional result and reject missing or duplicate result bindings. The option has a sealed role and does not change legacy `DealDamage(ValueSource)` callers. Synthetic tests cover prevention, redirection, lifelink source identity, zero damage, clone isolation, and declaration validation. This slice supplies only a singular amount result. It does not supply recipient/source result bindings, a batch total, or Drain Life's separate pre-damage recipient cap and black-only X payment. Drain Life and Syphon Soul still need card wiring and other prerequisites. The next dependency-ready slice is event-participant binding or a damage-recipient/source result identity contract.

Focused `TestDealDamageFrom*` tests and `go build ./pkg/mage/dsl` pass. `go test ./pkg/mage/... -short -count=1` passes its runnable engine packages but fails during DSL and AI package setup because `cards/all.go` imports the absent `cards/custom` package. The DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` hit the same missing package. `make lint` also reports absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. The required checks are blocked, so this slice is not fully validated or committed. Existing uncommitted engine and documentation changes remain in the working tree.

## Previous Slice: Filtered Other-Creature Death

`AnotherCreatureYouControlDies()` is a sealed `EventPattern` for `WhenEvent` and `WhenEventModes` on battlefield permanents. It reuses the existing battlefield-to-graveyard event, pre-departure creature and controller checks, source exclusion, and pending-trigger lifecycle. It does not bind the dying creature or add general event filters. Synthetic tests cover own and opposing creatures, stolen and lost control, self death, simultaneous death, token death, noncreature departure, exile, phasing, source leave and return, clone and stack-copy isolation, inspection, and registration validation. The first card candidate is Wraithbloom Cultivator; its production wiring remains unchanged. The `custom-other-controlled-death` engine gap is resolved, while the card migration remains open.

Focused tests and `go build ./pkg/mage/dsl` pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop at package setup because `cards/all.go` imports the missing `cards/custom` package. `make lint` also reports absent C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. This slice remains reviewable but is not fully validated or committed. A fresh production search found two `DiesCreatureTrigger` calls with additional card requirements; the legacy export remains. The custom card package and Wraithbloom Cultivator need separate card wiring.

## Previous Slice: Source Attack and Block Events

`Attacks(This)` and `Blocks(This)` are sealed `EventPattern` constructors for `WhenEvent` and `WhenEventModes` on battlefield permanents. They reuse the declared-attacker and first declared-blocker events, their existing trigger conditions, and the shared pending-trigger, stack, target, source-incarnation, and clone paths. A blocker assigned to multiple attackers triggers once. Entering attacking or blocking does not match. Tests cover real combat, source and other-creature filtering, once-per-blocker grouping, source leave and return, phasing, copies, clones, targets, inspection, and registration validation. Giant Badger and Mijae Djinn are first caller candidates; no production card was changed. A fresh production search found 27 `AttacksTrigger` calls and 5 `BlocksTrigger` calls, so both legacy exports remain. The `promo-block-event` engine gap is resolved; Giant Badger still needs card wiring. This slice does not add `BecomesBlocked`, blocks-or-becomes-blocked, unblocked-attacker, filtered death, or event-participant bindings. The next dependency-ready slice is filtered other-creature death.

Focused tests and the real-combat integration test pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail at package setup because `cards/all.go` imports the absent `cards/custom` package. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. The changed gap record validates against the engine-gap schema. This slice is reviewable but not fully validated or committed.

The singular-damage work below was already present in the working tree when this slice began.

## Previous Slice: Explicit Singular Damage

`DealDamageFrom(DamageSourceRef, DamageRecipientRef, ResolutionValue)` is the temporary typed damage constructor. `This` is the supported damage source. Player, permanent, and declared mixed damage targets are recipients. The adapter reads the resolving source incarnation or spell, then uses the shared replacement and prevention pipeline. It does not introduce a second damage executor. Effect-context target capture now preserves an announcement-time incarnation when the card leaves and returns. The legacy `DealDamage(ValueSource)` export and all production callers remain intact. Tests cover departed source LKI after a same-card return and cleanup, actual lifelink after prevention, permanent and player recipients, phased or absent recipients, target leave-and-return, absent source, clone isolation, spell source, inspection, and declaration validation. Focused tests pass. `go test ./pkg/mage/... -short -count=1`, the DSL registration run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` fail at package setup because `cards/all.go` imports the missing `cards/custom` package. `make lint` also reports absent C metadata for `cmd/pylib/decision_spec_ffi.go`. This slice is not fully validated or committed. Batch damage, other source references, actual damage result bindings, and card wiring remain open. Lightning Bolt, Prodigal Sorcerer, Mana Crypt, Modal Test Artifact, and Icatian Moneychanger are first caller candidates. The next dependency-ready slice is source attack/block event patterns or filtered other-creature death events.

The preceding damage-source engine work is retained below for context.

## Previous Slice: Damage Source Incarnation and LKI

The shared damage pipeline now carries a captured `DamageSource` through replacement and prevention. The capture keeps permanent incarnation, departure LKI, controller, colors, types, and keywords; resolving spells use their stack controller. Tests cover leave-and-return after cleanup, unrelated resolution sources, phasing, clone isolation, protection, deathtouch, and lifelink. Fight uses the same captured-source path. The existing planeswalker outcome change removes loyalty for unprevented damage and marks damage only if the permanent is also a creature; its tests cover prevention, history, dual types, state actions, and clones. No typed DSL damage constructor or production card changed. Lightning Bolt, Prodigal Sorcerer, Mana Crypt, Modal Test Artifact, and Icatian Moneychanger still need typed recipient/source wiring. The next dependency-ready slice is explicit singular `DealDamageFrom` with typed source and recipient roles, inspection, and validation.

Focused damage tests and `go test ./pkg/mage -short -count=1` pass. The required full checks cannot pass in this workspace: `cards/all.go` imports the absent `cards/custom` package, so `go test ./pkg/mage/... -short -count=1`, DSL registration tests, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make test` stop at package setup. `make lint` also reports missing C metadata for `cmd/pylib/decision_spec_ffi.go`. `git diff --check` passes. This slice is reviewable but is not fully validated or committed. Gap records stay blocked because no typed card adapter or card wiring was added.

The architectural specification is in [composable-dsl-design.md](composable-dsl-design.md), and authoring reference is in [dsl-reference.md](../dsl-reference.md).

## Current Run: Simultaneous Permanent Exile

`ExileEach(PermanentSet) ResolvingEffect` is available in spell, activation, and triggered resolving bodies. It selects the set once and uses `Game.ExilePermanents`, which shares the zone-change batch with set return and destruction. Each present, unphased battlefield incarnation moves at most once. Leave and state-trigger checks see the completed group. The shared `ExilePermanent` path now lets tokens leave without retaining them in exile. No production card or legacy author export changed.

Dust to Dust's complete Oracle text in `data/DRK.json` is "Exile two target artifacts." The typed declaration can use one exact-two artifact target group and `ExileEach`; its production card remains unchanged. The Dark inventory still requires card wiring. The test-only contract first failed because `ExileEach` was absent; a second test first failed because tokens stayed in exile. Tests cover a state trigger that must not observe an intermediate battlefield, duplicate and stale objects, phasing, target leave and return, clone isolation, source leave triggers, token removal, and registration rejection. The public example is in `docs/dsl-reference.md`.

This slice uses the existing exile operation and zone-change batch. General zone-change replacement ordering is still an engine gap; this adapter does not claim that contract or linked exile/return. Other B and Z work and all production card migrations remain open. A fresh search found no production `ExileEach` caller, six card files with `Exile`, and the one `ExileTargetStep` caller. Those exports remain. The next dependency-ready work is a separate zone-change replacement prerequisite or a P payment output after a fresh audit. The historical recipient-history blocker below is stale: incarnation-keyed recipient and source histories and their clone, phasing, and cleanup tests now exist. No JSON gap record was closed by this engine slice.

Validation passes: focused tests with `-count=1`, `go test ./pkg/mage/... -short -count=1`, the DSL registration and closed-role run, `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, `make test`, `make lint` with zero issues, and `git diff --check`. The standalone reference example builds. The post-lint diff was reviewed.

## Outstanding Card Migrations

- [ ] **Icatian Moneychanger**: Audit entry damage source identity. (Sacrifice activation with credit counter capture is complete).
- [ ] **Spore Creatures**: Migrate remaining activated definitions after token color, damage-source LKI, and sacrifice-cost audits. `cards/fallen_empires/creatures.go` already defines Saproling-sacrifice abilities for Elvish Farmer and Thallid Devourer; audit their exact behavior separately from upkeep migration.
- [ ] **Whirling Dervish**: Audit end-step timing, per-turn opponent-damage condition, and source identity before retiring `DealsDamageToOpponentTrigger`. (It is the remaining production caller; Nicol Bolas is already migrated).
- [ ] **Wheel of Fortune & Contract from Below**: Migrate before retiring the legacy `DiscardHand` selector surface. Audit simultaneous multi-player hand discards and the remaining draw/ante definitions. (Singular typed hand discard is available as `DiscardEntireHand(subject)`).
- [ ] **Modal Cards with Graveyard References**: Add typed graveyard references and remaining modal effects. Trusty Retriever, Entomber Exarch, and Biblioplex Tomekeeper retain their existing modal builders until their remaining effects have exact adapters.
- [ ] **Deep Spawn**: Audit optional upkeep mill payment before migrating the last production `MillTargetPlayer` caller. (Selhoff Occultist, Towering-Wave Mystic, Millstone, and Whimsy's mill branch are migrated; `MillstoneEffect` is retired).
- [ ] **Cauldron of Essence**: Audit retained reanimation behavior before declaring this card complete. `cards/secretsofstrixhaven/artifacts.go` already uses `ReturnCardToBattlefield` with a typed graveyard target, a sacrifice cost, and sorcery timing. Preserve target selection before payment, graveyard ownership, and return under the resolving controller. (Death trigger is migrated; `AnyCreatureDiesTrigger` is retired.)
- [ ] **First Main Phase Callers**: Migrate other first-main callers before retiring `BeginningOfFirstMainPhaseTrigger`: Scathing Shadelock needs typed prepared state and prepared casting audit; Abstract Paintmage needs missing instant/sorcery mana spending restriction.
- [ ] **Surrounding Card Definitions**: Migrate remaining surrounding definitions for Blindblast, Exclude, Thirst for Knowledge, and Rhystic Study. Damage-source LKI, stack references, hidden-zone choices, and event participants still need exact adapters.
- [ ] **Rhystic Study**: Audit optional draw and payer choice order when migrating the trigger. Legacy payment branches expose nested typed effects to inspection/validation; event participant capture remains open.
- [ ] **Elemental Uprising**: Audit land animation, target-controller restriction, color behavior, and lifetime.
- [ ] **Neyith of the Dire Hunt**: Audit combat trigger, optional hybrid payment, power doubling, and end-of-combat blocking requirement before removing `TargetMustBeBlockedIfAble`.
- [ ] **Glorious Decay**: Audit damage mode and graveyard target before retiring `DestroyTargetArtifact`.
- [ ] **Positional Return Helpers**: Continue positional return helper migrations. Witch Hunter's separate damage ability awaits damage-source LKI. Rakalite's delayed source identity audit and migration are complete; audit Prismari Charm's modes before retiring `ReturnToHandTarget`.
- [ ] **String-Bound Action Migrations**: Continue destruction, return, tap-or-untap, attachment, and other string-bound action migrations where exact typed forms exist.

## Engine Blockers

### Combat Removal
- [x] Preserve blocked status when the final blocker is removed. Correct the shared combat operation after a failing regression; keep explicit-unblocking legacy effects separate. See the typed combat-removal slice below.
- [ ] Audit explicit unblocking and historical sole-blocker tracking for False Orders and Ydwen Efreet before retiring their legacy removal helpers.

### Zone LKI Retention
- [x] Audit damage recipient incarnation and damage-condition histories. Incarnation-keyed game and turn histories retain old visits across leave and return, phasing, and cloning. Legacy card-ID queries remain for compatibility and reset at final cleanup. Damage source instructions retain incarnation LKI separately; typed counter and attachment histories have their own durable records.

### Mana-Cost Representation
- [ ] Support monocolored hybrid symbols in the shared parser and payment path. `ParseManaCost("{2/W}").CMC()` currently returns 0 instead of 2. The typed mana-value adapter reuses this service; it does not add symbol support.

### Prepared Spell Payment
- [ ] Resolve prepared spell mana payment before migrating Infirmary Healer's prepared Stream of Life factory. `WithPreparedSpell` installs a zero-cost activated ability, and `CastPreparedSpellCopy` pushes a copy without paying its mana cost. The current X=0 test supplies no mana; a failing cost regression is still required.
- [ ] Test the actual prepared casting entry path: {G} at X=0, {4}{G} at X=4, failed payment and target rollback, stored X, and copied targets.

### Damage-Source Identity & LKI
- [x] Correct unprevented planeswalker damage outcomes: `DamageSystem.ExecuteDamageToCreature` removes loyalty after damage replacements and prevention. Rakalite target eligibility and prevention continue through the existing pipeline; card wiring remains separate.
- [x] Retain source incarnation and last-known characteristics through the shared damage and replacement pipeline, including departure, return, phasing, and cloning. Damage actions use captured source characteristics for protection, lifelink, and deathtouch. Singular typed damage is available; recipient history and collection damage remain open.

### Destruction Regeneration Scope
- [x] Preserve turn-long regeneration prohibitions through ability removal. `CantRegenerate` is a rules restriction, not a removable ability; typed duration metadata reports a detrimental effect. See the Hurr Jackal slice below.
- [x] Resolve repeated destruction and intermediate state observations in the Disk helper. One union collection now uses the existing simultaneous destruction executor.
- [x] Keep the collection no-regeneration prohibition on each destruction action; preserve permanent characteristics and simultaneous death snapshots.
- [x] Correct singular typed, positional, step, and string-bound no-regeneration execution to use the action-scoped prohibition. Migrate Terror; see the completed slice below.
- [ ] Migrate remaining no-regeneration callers before retiring positional exports: Abu Ja'far needs combat-opponent capture and simultaneous destruction; Detonate needs X-constrained targets and damage-source LKI; Aswan Jaguar needs a chosen-subtype target adapter. Crumble's saved-value migration is complete and the string-bound no-regeneration constructor is retired.

### Temporary Blocking Restrictions
- [x] Preserve resolved blocking prohibitions through ability removal and source departure. Use the existing combat-restriction storage, keeping printed unblockable abilities removable. `TestBlockingProhibitionSurvivesAbilityRemoval` failed before the fix.
- [x] Migrate Dwarven Warriors and Suspicious Bookcase to typed targets, activation costs, and `Until(EndOfTurn, CantBeBlocked(creature))`. See the activated blocking slice below. No engine gap.
- [x] Migrate Teleport to typed `Spell`, `OnlyDuring(DeclareAttackers)`, and `Until(EndOfTurn, CantBeBlocked(creature))`. Enforce the casting restriction in normal, alternate, free, and card-copy casting paths and spell availability; retire `MakeUnblockableUntilEndOfTurn`. See the Teleport slice below.

### Untap Skipping
- [ ] Audit Goblin Rock Sled's last-turn attack condition, then migrate its attack trigger before retiring `StunCreature`. Deep Spawn's optional upkeep mill payment remains open under the milling audit.

### Life Gain & Payment Boundaries
- [x] Migrate the nine legacy replacement registration calls in `fasting_step_test.go`, `fasting_controller_test.go`, and `replacement_life_outcome_test.go`; retire `AddFastingReplacement`. Six integration registrations now use `Static(SkipDrawStepForLife(2))`. Three explicit lifetime registrations use private executor records, preserving absent incarnation, phasing, leave-and-return, and clone assertions. No engine gap or production card migration remains for this helper.
- [ ] Audit card callback `PlayerGainLife` callers and direct `GainLife` mutations for missing events. The low-level API intentionally retains its no-event contract; callers that already emit events must not receive duplicates. The singular DSL and string-variable executors report actual outcomes through one shared event path.
- [ ] Audit life-gain replacement ordering and recipient/source identity. Legacy amount modifiers still run before the general replacement pipeline.
- [ ] Establish simultaneous collection gain and replacement handling, including events that must observe all final life totals.

### Mill Replacement Semantics
- [ ] Integrate legacy mill modifiers with replacement choice and rechecks. Explicit registrations run first, followed by static abilities in battlefield and ability order. General replacement ordering remains unsupported.
- [ ] Define once-per-action replacement identities and actual mill results, including general zone replacements and events, before adding typed rules.

### Event & Delayed Capture
- [ ] General future event capture and general delayed events/reflexive triggers remain open.
- [x] Resolve the Blight attachment-tap audit: event-time permanent capture uses existing trigger records; `TapPermanent` now emits events only on a tap transition.
- [ ] Continue attachment-tap audits for Psychic Venom, Kudzu, Artifact Possession, and Relic Bind, plus remaining regeneration callers. Damage identity, attachment manipulation, and modal/player-or-planeswalker targeting remain separate work.

## Remaining DSL Families (Architectural Scope)

| Family | Outstanding Work |
| :--- | :--- |
| **Bindings** | Multiple variable target groups; card and stack references; other saved-value domains; divided-value declarations; explicit branch results. |
| **Actions** | Typed damage sources with LKI; payment/action result bindings and result amounts; typed token definitions; remaining action families and legacy callers. |
| **Milling** | General zone-change replacements, events, result bindings, and replacement ordering beyond the legacy amount modifier and direct player-zone updates. |
| **Choices & Hidden Zones** | Card choices; search, reveal, and order bindings; zone-lifetime references; visibility contracts; scry/surveil result bindings and surveil zone-change events/replacements. |
| **Continuous Effects** | Remaining modifications and durations; complete compound-effect identity; cross-layer condition continuity; dependency loops and audit; general multi-keyword summary behavior. |
| **Triggers** | Event participant bindings and other event patterns; event-based and reflexive delayed capture; linked state; legacy caller migrations. |
| **Spells & Activations** | Remaining legacy builders and callers; multi-mode selection; remaining casting and copy variants. |
| **Mana Abilities** | Dynamic amount planning; conditional, modal, and loop-based classification; automatic planning for mixed bodies and abilities without tap costs; fixed and string-bound legacy callers. |
| **Replacements** | General typed replacement/prevention rules and outcomes; retirement of legacy replacement registration test callers. |
| **Life Collections** | Collection life gain and its event/replacement audit; multiplayer team selection. |
| **Author Boundary** | Mutable aliases; callback declarations and cost conditions; remaining direct engine imports; obsolete author exports. |

## Legacy Helper Audits

- [ ] Migrate the remaining `ExileTargetStep` caller before retiring that export. (`ExileTarget` is retired from both author surfaces; internal executor remains in use by typed `Exile` and legacy step).
- [ ] Migrate Hazezon Tamar's `ExileTargetStep` loop and the string-bound `ExileGathered` callers in Tawnos's Coffin and Emeritus of Truce's Swords to Plowshares spell. Preserve timing, identity, and linked state. Audit graveyard-source exile separately.
- [ ] Audit remaining targeted, event-player, and string-variable life helpers before removing `GainLifeTarget`. The typed gain adapter still uses its executor.
- [ ] Migrate remaining UUID-based subtype helpers and positional destruction, return, modal, spell, PT, and keyword helpers before removing their exports.
- [ ] Close the immutable author boundary. Re-exporting mutable engine types through `dsl` does not complete it. Recheck direct imports and caller counts when each family is migrated.

## Outstanding Validation and Acceptance

- [ ] Maintain the High-Leverage 3-Tier Testing Model (exhaustive Tier 1 engine tests, 100% Tier 2 `TestRegisteredDefinitions` static validation, and Tier 3 risk-based card tests).
- [ ] Add exact caller migration evidence for each unfinished family, remove obsolete exports only after all callers have replacements, and close the mutable author surface before marking the full design complete.
