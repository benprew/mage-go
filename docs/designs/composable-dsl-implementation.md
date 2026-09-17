# Composable DSL implementation inventory

The full design is not complete. This inventory distinguishes implemented
adapters from unfinished families. The existing binding-slice document records
the earlier foundation, not the current completion state.

Rules reference: repository Comprehensive Rules, effective February 27, 2026.
Relevant contracts are CR 400.7, 601–602, 603, 608, 611–613, 614–616, and the
sacrifice and coin-flip actions in CR 701.

## Coverage by slice

| Slice | Implemented surface and executor | Remaining work |
| --- | --- | --- |
| State triggers and sacrifice | Pure conditions, `WhenState`, `This`, `Sacrifice`; existing observation, pending triggers, incarnation checks, and sacrifice action | The first-slice card tests and registry validation remain the baseline. |
| Typed bindings | Permanent/player/damage target declarations, one variable collection group, saved permanents, modal scope, copy and retarget adapters | Multiple variable groups, card/stack references, other saved-value domains, divided-value declarations, explicit branch results |
| Actions | Typed sacrifice, destruction, simultaneous destruction, exile, tap, untap, draw, discard, life, counters, and controller lookup | Typed damage sources with LKI, all remaining legacy callers, result amounts, other action families |
| Choices | Typed permanent choice, lowest-power selection, coin branches, sequential loops, and paid/unpaid branches over total-cost transactions | Card choices, search/reveal/order bindings, visibility contracts, payment/action result bindings |
| Continuous effects | Typed PT, keyword, color, and subtype-family modifications; captured temporary membership; dynamic static membership; source-exit duration | Other modifications and durations, dependency audit, complete compound-effect identity contract |
| Triggers | Source enters/dies, upkeep/draw/end-step patterns, typed targets, intervening-if, captured `AtNext` turn-step triggers | Event participant bindings, other event patterns, reflexive/event-based delayed capture, linked state |
| Spells and activations | Closed spell/activation/target/mode options, explicit costs, timing/limit/state restrictions, sequential mana inspection and immediate activation | Remaining legacy builders and callers, complete mana-ability surface, all casting/copy variants |
| Replacements | Typed recipient prevention shield over the existing pipeline | General typed replacement/prevention rules and outcomes |
| Author boundary | DSL exports, generator import test, compile-fail fixtures, definition validation | Mutable aliases, callback declarations, remaining direct engine imports, obsolete exports |

Production still has direct engine imports in 47 files, excluding tests and
registration-only `test.go` files. Re-exporting the same mutable types through
`dsl` does not complete the package boundary. Legacy helpers remain where
callers do not yet have exact replacements.

## Added adapter contracts

| Adapter | Evaluation, absence, and ownership | Evidence |
| --- | --- | --- |
| `FlipCoin(player, OnWin(effect), OnLoss(effect))` | Resolve the player once; flip once through `Game.FlipCoin`; execute only the matching branch. An omitted branch or absent player has no effect. Definitions do not store random state. | `TestComposableCoinFlipBranches`, `TestComposableCoinFlipValidation` |
| `ForEach(subjects, member, body)` | Capture incarnations once; execute sequentially in selector order. Each iteration inherits outer bindings and has independent writes. Restore the outer frame on return or error. A returned physical card is a different member. | `TestComposableLoopIncarnations`, `TestComposableLoopScope` |
| `ControllerOf(subject)` | Read the current controller at consumption. An absent battlefield incarnation produces absence. It does not mean a player captured by an earlier event. | `TestComposablePaymentSavedSubject` |
| `UnlessPaid(player, costs, effect)` | Offer a payment at resolution, lock and validate the whole cost group through the existing transaction, then commit. Declined or failed payment executes the body with the original ability controller. No player means no instruction. | `TestComposablePaymentAtomicity`, `TestComposablePaymentSavedSubject` |
| `IfPaid(player, costs, effect)` | Use the same transaction; run the body only after successful payment. Acceptance without payment does not execute the body. | `TestComposableIfPaid`, affected card-package tests |
| Activation restrictions | Check through `CanActivate` before payment; use existing timing and per-turn counters. Pure conditions are part of definition inspection. Conflicting timing/limit declarations fail at the factory boundary. | `TestComposableActivationRestrictions`, `TestComposableActivationOptionValidation` |

The activation regression exposed a zero-value bug in the existing step check:
`Untap` was treated as an unspecified step. `TimingStepOnly` now always checks
the declared step. This does not grant priority during the untap step.

## Removed random predicate

`FlipCoinCond` is removed from the engine and DSL. It is replaced by `FlipCoin`
with closed `OnWin`/`OnLoss` branches. `GameReader` no longer exposes random
mutation. The executor remains the existing game random source. Branch
metadata retains both alternatives; the legacy summary is derived from them.
No target, LKI, or source-lifetime behavior is added by the coin operation.

| Previous caller | Replacement and Oracle clause | Regression |
| --- | --- | --- |
| Mijae Djinn | `OnLoss`: remove from combat and tap | `TestMijaeDjinn` |
| Ydwen Efreet | `OnLoss`: remove from combat and prevent blocking | `TestYdwenEfreet` |
| Goblin Artisans | `OnWin`: draw; `OnLoss`: counter | `TestGoblinArtisans` |
| Orcish Captain | Typed Orc target; `OnWin`: +2/+0; `OnLoss`: -0/-2 until end of turn | `TestOrcishCaptain_CoinFlipBoostsOrShrinksTarget` |
| Mana Crypt | `OnLoss`: deal 3 damage to the controller | `TestManaCrypt_UpkeepCoinFlip` |
| Bottle of Suleiman helper | `OnWin`: token; `OnLoss`: damage | `TestBottleOfSuleiman` |

`EffectIfPaid` is also removed after all eight callers migrated to
`IfPaid(You, Costs(...), effect)`. The replacements use the same mana and
sacrifice cost executors, with aggregate transaction validation. They keep
the existing surrounding trigger conditions. The instruction subject is
explicit: life and draw effects use `You`; untap effects use `This` and its
captured incarnation. Structural metadata records the payment branch and
its body; the legacy summary comes from the body.

| Caller | Payment and body | Existing evidence |
| --- | --- | --- |
| Farmstead | {W}{W}; gain 1 life | `TestFarmstead` |
| Mana Vault | {4}; untap source | `TestManaVault` |
| Island Fish Jasconius | {U}{U}{U}; untap source | `TestIslandFishJasconius` |
| Brass Man | {1}; untap source | `TestBrassMan` |
| Leviathan | Sacrifice two Islands; untap source | Fourth Edition package suite |
| Tablet of Epityr | {1}; gain 1 life | `TestTabletOfEpityr`, `TestTabletOfEpityrSelfTrigger` |
| Urza's Chalice | {1}; gain 1 life | Antiquities package suite |
| Urza's Miter | {3}; draw one card | Antiquities package suite |

These migrations preserve surrounding legacy behavior; they do not certify
unrelated restrictions in those cards. Goblin Artisans retains its existing
documented targeting gap.

Cocoon's entry effect now composes `Tap(AttachedPermanent)` and
`AddCounters(This, Pupa, Fixed(3))` through `WhenEvent(Enters(This), ...)`.
`TestCocoon` covers the migrated entry and later counter behavior. The counter
instruction now uses the normal counter action and replacement path.

## Continuous layer extension

The later continuous-effect slice now also exports `SetColors`, `SetEachColors`,
`SetSubtypeFamily`, `AddSubtypeFamily`, and `GrantEachKeyword` through `dsl`.
These are modifications for `Until` or `While`, not installed effects.
The existing effect manager still applies and expires them.

| Adapter | Rules and execution contract | Evidence |
| --- | --- | --- |
| `SetColors`, `SetEachColors` | Layer 5 replacement; an empty list means colorless. Input lists are copied. Temporary membership is captured before any layer applies; static membership uses the current layer state. | `TestComposableTypeAndColorCapture`, `TestComposableStaticColorUsesLayerMembership`, `TestComposableColorlessAndSubtypeApplicability` |
| `SetSubtypeFamily` | Layer 4 replacement in one family through `applySetSubtypes`; other families remain. CR 205.3d and 305.7 apply, including printed-ability removal and intrinsic basic-land mana. | `TestComposableTypeAndColorCapture`, `TestComposableColorlessAndSubtypeApplicability`, existing basic-land subtype tests |
| `AddSubtypeFamily` | Layer 4 addition through the shared `applyAddSubtypes`; existing subtypes and rules-text abilities remain. | `TestComposableAddedSubtypePreservesAbilities` |
| `GrantEachKeyword` | Layer 6 grant through the existing attribute manager. Static selectors see changes from preceding layers. | `TestComposableStaticColorUsesLayerMembership` |

Captured references reject returned cards and retain identity through phasing.
Expiration in a clone does not change the original game. Invalid colors,
known mismatched subtype families, and absent subjects fail at the definition
boundary. Public composition and compile-fail fixtures cover the new surface.
Inspection includes each modification's operation and literal values.

No legacy exports or production callers are removed in this extension. Legacy
UUID-based subtype helpers still have callers and remain pending migration.
General compound-effect identity, cross-layer condition continuity, dependency
loops, and additional lifetime policies remain open acceptance work.

## Mana composition extension

`Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1), AddMana(Green, 1)))`
now retains its mana production profile. The production reader and automatic
planner share sequential inspection for both `Sequence` and legacy
`CompositeEffects`. Nested instructions keep their order and separate
`AddAnyMana` color choices. Targeted abilities are not mana sources.

CR 605.1a and 605.3b do not require a tap cost. Targetless activations with
unconditional mana instructions now resolve immediately with or without one.
Other sequential instructions also resolve immediately. Automatic planning
continues to exclude those mixed bodies because its executor only produces
mana. A tapped-for-mana event requires a tap cost and fires once per activation.
Non-mana activation restrictions permit these mana abilities. The planner's
step restriction now treats the zero-valued untap step as an actual step.

| Contract | Evidence |
| --- | --- |
| Nested production and immediate resolution; one tap event | `TestComposableManaProductionAndResolution` |
| Planning and cloned-game isolation | `TestComposableManaPlanningAndClone` |
| Target exclusion and immediate mixed instructions | `TestComposableManaTargetAndMixedEffects` |
| Explicit untap-step restriction | `TestComposableManaPlanningUntapRestriction` |
| Mana payment without a tap cost; no tap event | `TestComposableManaWithoutTapCost` |
| Non-mana activation restriction | `TestComposableManaNonManaRestriction` |
| One-color versus independent color choices | `TestComposableManaChoiceGroups` |
| Unpayable costs leave resources unchanged | `TestComposableManaCostFailureLeavesState` |
| Public DSL composition and registration | `TestTypedManaComposition` |

This extends the existing mana executors; no production card syntax or exports
are removed. Conditional, modal, and loop-based mana classification remain open,
as does automatic planning for mixed bodies and abilities without tap costs.

### Damage-source audit blocker

Typed damage-source work still needs an incarnation-aware engine path.
`DamageSystem.ExecuteDamageToPlayer` and `ExecuteDamageToCreature` use
`FindPermanent(sourceID)` for lifelink and deathtouch. The protection check in
`DealDamageToPermanent` also retrieves the source by physical card ID. A source
that leaves and returns can therefore be confused with the old damage source;
a departed source's granted keywords are not read from its LKI. `LookupObject`
also prefers a live object with that ID. These are code-audit findings, not new
passing regression coverage. A correct typed adapter must first retain the
applicable source incarnation and last-known characteristics through the
existing replacement and damage pipeline.

## Delayed turn-step extension

`AtNext(pattern, body)` creates a one-shot delayed trigger through the existing
trigger system. It supports the closed `TurnStepPattern` family, including the
new `EachEndStep`, `YourEndStep`, and `EachEndOfCombat` patterns. Those patterns
also work with `At` for ordinary recurring triggers.

The adapter captures the resolving controller, source incarnation, saved
permanent bindings, typed target references, and X. Your-step patterns match
that stored controller. Captured references retain object identity through
phasing, but reject a leave-and-return. Targets referenced by the delayed body
refer to the original objects; this adapter does not declare new delayed
targets. Other dynamic expressions evaluate when the body resolves. Arbitrary
legacy pipeline variables and positional target/value conventions are not
captured by this adapter.

Each installation stores its own immutable capture. Each resolution copies the
frame, including in cloned games. Later assignments in the creating sequence
cannot change the scheduled body. Registration permits outer reads only when
the producer precedes installation; writes in a delayed body cannot supply a
value to the creating sequence.

A separate engine regression demonstrated that legacy delayed triggers bypassed
pending placement. They now join ordinary pending triggers and use the same
APNAP stack-placement path (CR 603.3b). Their event amount, event source, legacy
subject, controller, and captured source incarnation are retained. One-shot
triggers are consumed when the event matches, before stack placement. Cloning
copies both scheduled and pending delayed records. The prior source-lifetime
test now explicitly places pending triggers before inspecting the stack.

| Contract | Evidence |
| --- | --- |
| Pending placement, APNAP, one-shot consumption, pending clone | `TestDelayedTriggersWaitForPriorityAndAPNAP` |
| Captured saved values, source exit, returned host, scheduled clone | `TestComposableDelayedCaptureAndClone` |
| No earlier-event observation; stored-controller step match | `TestComposableDelayedStepAndCreationTime` |
| Producers before installation; no delayed writes escaping; nil validation | `TestComposableDelayedBindingScope` |
| Repeated schedules, separate target frames, captured X, phasing | `TestComposableDelayedTargetFrames` |
| Source already returned before installation; direct-context capture | `TestComposableDelayedSourceAlreadyReturned`, `TestComposableDelayedDirectContextCapture` |
| Failed body restores outer saved values and X without changing its capture | `TestComposableDelayedRestoresFrameAfterError` |
| Public composition, metadata, rejected event-pattern argument | `TestTypedDelayedComposition`, `TestClosedExpressionRoles` |

No legacy delayed constructors are removed. Event-based and reflexive capture,
new delayed targets, linked state, and remaining caller migrations are open.

## Acceptance still open

The current target contract rejects multiple variable groups instead of
guessing their boundaries. Damage still needs an adapter that preserves the
source incarnation and LKI through death, return, and copying. Search and reveal
need zone-lifetime references and visibility tests. Temporary/static effect
coverage is limited to the implemented modifications and supported durations.
Event-based delayed/reflexive bindings and static replacement rules are unfinished.

Erosion's migration needs event participant capture: “that player” and “that
land” cannot be replaced by unqualified current-state lookups. Thrull Retainer
needs capture before its sacrifice cost. Keep their existing helpers until
those adapters are available and tested.

The full design must remain in progress until every family has exact caller
migrations, its obsolete exports are removed, and the mutable author surface
is closed. Passing the repository suite is not evidence for unimplemented
acceptance criteria.

## Verification

The delayed turn-step extension passed:

- Focused pending/APNAP, delayed-frame, source-lifetime, public composition,
  compile-fail, and CR 603.7 integration tests.
- `go test ./pkg/mage/... -short -count=1`.
- `go test ./... -count=1`, including long AI simulations and every card package.
- `go build ./...` and `go vet ./...`.
- `make lint`: zero issues.
- `git diff --check`.

The mana composition extension passed:

- Focused mana composition, tap-event, production, and public DSL tests.
- `go test ./pkg/mage/... -short -count=1`.
- `go test ./... -count=1`, including the long AI simulations.
- `go build ./...` and `go vet ./...`.
- `make lint`: zero issues.
- `git diff --check`.

The continuous layer extension passed:

- Focused composable, subtype, basic-land, public composition, and compile-fail tests.
- `go test ./pkg/mage/... -short -count=1`.
- `go test ./... -short -count=1`, including registered-definition validation.
- `go build ./...` and `go vet ./...`.
- `make lint`: zero issues.
- `git diff --check`.

Long AI simulations were not rerun for this extension.

Previously passed for the implementation before the continuous layer extension:

- Focused adapter tests, including payment rollback, clone isolation, loop
  scope and error restoration, and activation validation.
- `TestTypedResolutionComposition` through the public DSL and the
  `go/types` compile-fail suite.
- All registered definitions through `TestRegisteredDefinitions`.
- `go test ./... -count=1`, including the long AI simulations.
- `go build ./...` and `go vet ./...`.
- `make lint`: zero issues.
- `git diff --check`.


## Controller-life action family

Completed the [controller-life slice](composable-dsl-life-slice.md): all callers
of the fixed/dynamic controller-life and life Step constructors now use typed
subjects, and the five old constructors and four private effect types are
removed. All production replacements compile through `dsl`. Dynamic amount
inspection, original value context, and nonpositive gain behavior have focused
regressions. The parity document lists every migrated production and test file.
Targeted, event-player, and string-variable life helpers remain open.


## Verification for the controller-life slice

- Focused life behavior, public composition, and compile-fail tests passed.
- All ten affected card-package suites passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -count=1` passed, including the long AI simulations.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues. The sandbox prevented some optional
  cache writes; checks still completed successfully.
- `git diff --check` passed.

## Source-exit duration family

Completed the [source-exit duration slice](composable-dsl-source-duration-slice.md)
within later continuous-effect slice 4. `UntilSourceLeaves(This, ...)` uses the
existing layer installation and source-removal paths. It captures source and
subject incarnations, membership, and amounts, survives phasing, and keeps cloned
games isolated. Gaea's Liege now uses a typed land binding and this duration.
All callers of `BecomesBasicLandTargetUntilSourceLeaves` migrated, and its engine
and DSL exports are removed. Source absence or return before resolution now
correctly prevents installation; focused failing regressions preceded the fix.
Other continuous-effect families remain open.

## Verification for the source-exit duration slice

- Focused duration, card, public composition, and compile-fail tests passed.
- `go test ./cards/fourthedition -count=1` passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -short -count=1` passed, including all card packages and
  registered-definition validation.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues. Its unrelated import formatting change
  in Secrets of Strixhaven was reverted.
- `git diff --check` passed.

Long AI simulations were not rerun; this slice changes no AI executor.

## Source-untap action family

Completed the [source-untap slice](composable-dsl-source-untap-slice.md) within
later action slice 2. Every production caller now uses `dsl.Untap(This)`;
`UntapSource` and its private executor are removed. A failing regression showed
that the old helper could untap a returned source. The typed adapter preserves
the captured incarnation through control changes and phasing, and rejects a
returned or absent source. Stun counters, untap events, cloned games, metadata,
validation, and public composition have focused coverage.

Positional-target untap helpers were later removed by the target-untap slice
below. Aura tap migration remains open:
`AttachedPermanent` currently reads a live source and needs a separate audit of
last-known attachment information for departed Auras under CR 608.2h.

## Verification for the source-untap slice

- Focused behavior, public composition, and compile-fail tests passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -short -count=1` and `go test ./... -count=1` passed, including
  all card packages, registered definitions, and long AI simulations.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues; optional cache writes were blocked by
  the sandbox. Its unrelated Secrets of Strixhaven import change was reverted.
- `git diff --check` passed.

## Target-untap action family

Completed the [target-untap slice](composable-dsl-target-untap-slice.md) within
later action slice 2. All seven production callers and the ward test fixture
now use typed target or saved-member references. `UntapTarget`,
`UntapTargetStep`, their DSL exports, and the private positional executor are
removed. Untap resolves a typed reference directly through `Game.UntapPermanent`.

Whimsy uses the bounded `ChooseRandomPermanent` saved-binding adapter from
choice slice 3. Legacy event triggers can add typed declarations with
`GenericTriggered.WithTargets`. A failing regression exposed a missing target
incarnation map in stack clones; the clone now retains that immutable map.
The parity document records every caller, metadata changes, and remaining gaps.
Disharmony and Reset's documented timing restrictions remain outside this slice.
Tap-or-untap, attachment, gathered-string, and untap-skipping families remain open.

## Verification for the target-untap slice

- Focused untap, random binding, copy, trigger, ward, public composition, and
  compile-fail tests passed.
- All five affected card-package suites passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -count=1` passed, including registered-definition validation,
  all card packages, and long AI simulations.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues. Optional cache writes were blocked by
  the sandbox. Its unrelated Secrets of Strixhaven import change was reverted.
- `git diff --check` passed. No calls or definitions of either removed untap
  helper remain; the forge command's unrelated `UntapTarget` enum remains.

## Gathered untap and combat-prevention family

Completed the [gathered-untap slice](composable-dsl-gathered-untap-slice.md).
Ebony Horse, the only string-bound untap caller, now uses a typed target and
`Untap`. `UntapGathered` and `UntapGatheredData` are removed. The unused
positional combat-prevention constructor is replaced by the typed
`PreventCombatDamageToAndBy(subject)` instruction. `TargetYouControl()` adds
an inspected controller restriction to singular and collection target bindings.

A failing card regression proved that removing Ebony Horse's target from combat
was wrong. Its target now stays in combat, with combat damage to and by it
prevented for the turn. A separate failing engine regression proved that the
old prevention shield followed a returned card. The shield now retains and
checks its battlefield incarnation. Stun replacement does not stop the following
prevention instruction. The existing replacement, clone, and duration paths remain.

Attachment tap, tap-or-untap, untap-skipping, other string-bound actions, and
general damage-source LKI remain open. The capability and caller table records
the exact scope; this does not complete the general replacement family.

## Verification for the gathered-untap slice

- Focused Ebony Horse, combat-prevention lifetime, controlled-target,
  public composition, and compile-fail tests passed.
- `go test ./cards/arabian -count=1` passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -count=1` passed, including registered definitions, all card
  packages, and long AI simulations.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues. Optional cache writes were blocked by
  the sandbox. Its unrelated Secrets of Strixhaven import change was reverted.
- `git diff --check` passed. No Go references to the removed constructors or
  executor types remain.
