# Engine rules contracts

## Printed basic land types: intrinsic mana

CR 305.6 grants the corresponding tap-for-mana ability to an object with the
land card type and a basic land subtype, even without a printed mana ability.
After type-layer effects apply, the engine updates these intrinsic abilities
before ability-layer effects apply. Repeated layer passes do not accumulate
abilities. Each basic subtype supplies its own color. Removing a subtype removes
its intrinsic mana ability. A game clone cannot change the original's abilities.

This permits RON basic and dual land definitions to omit mana abilities.
Evidence: `TestPrintedBasicLandTypesHaveIntrinsicMana` and
`TestRONBasicLandTypesSupplyMana`, plus the existing basic land type effect tests.

## Damage: permanent-source attribution

The player and permanent damage executors save source attribution before damage
results emit life or damage events. A captured permanent source supplies its
current name, artifact type, and controller while that incarnation is present.
After departure or phasing, retained information supplies those fields. Cleanup,
a returned card, and that card's later departure do not change the captured
source. Missing captured information supplies an unknown name, no artifact
classification, and no controller; it does not fall back to another incarnation.

Player artifact-damage history uses effective types, including gained or lost
artifact type (CR 608.2h). Combat aggregation uses the damage source's controller,
including its last controller after departure, rather than its owner or the
controller of a copied ability. Player and permanent callbacks use the action's
combat flag and actual damage after prevention and redirection. Fully prevented
damage emits no callback and enters no history. Redirected permanent damage does
not enter player damage history. UUID-only callers retain card lookup for sources
without a live permanent; this does not establish spell-source LKI.

Existing signatures remain unchanged: `Game.SetOnDamageDealt`,
`Game.GetArtifactDamageTaken`, and
`DamageSystem.CombatDamageSourcesThisStep`. History keys and `EvtDamageDealt`
participants still use UUIDs. Immutable event participant captures, incarnation
history keys, source-dependent follow-up effects, spell identity, typed recipients,
damage batches, and damage result bindings remain separate contracts.

Evidence: `TestDamageAttributionSource`,
`TestDamageAttributionPermanentCallback`,
`TestDamageAttributionCopyCloneAndPrevention`, `TestDamageAttributionRedirection`,
and the test-harness `TestDamageAttributionSequence`.

## Permanent damage: planeswalker results

`Game.DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID)`
uses the shared damage replacement and prevention pipeline. The retained
`DamageToCreatureAction` name also covers planeswalker recipients; it is not a
new author-facing typed role.

After replacement and prevention, damage removes loyalty counters from a
planeswalker, to a minimum of zero (CR 120.3c, 306.8). If the permanent is also
a creature, the same damage has both results: loyalty removal and the existing
creature damage result. A noncreature planeswalker receives no marked damage
or creature deathtouch result.

Damage events, history, and lifelink use the amount dealt, even when it exceeds
the available loyalty. The damage event retains the action's combat flag.
Zero damage and fully prevented damage do not produce damage events. The
recipient's controller chooses the order of applicable replacements through
the existing replacement service (CR 616.1).

Damage does not itself move the recipient. The existing state-based action
puts a zero-loyalty planeswalker into its owner's graveyard (CR 120.5, 704.5i).
Regeneration does not replace this move. Mutable permanent access preserves
copy-on-write isolation for game clones.

Evidence: `TestPlaneswalkerDamageResults`, `TestPlaneswalkerDamageCloneIsolation`,
`TestPlaneswalkerDamageLegacyEffect`, `TestPlaneswalkerDamageReplacementOrder`,
`TestPlaneswalkerDamageProtectionAndDeathtouch`, and
`TestPlaneswalkerDamageSpellResolution`.

This contract does not establish damage-source incarnation/LKI, typed recipient
bindings, simultaneous damage batches, damage result bindings, or a complete
creature damage keyword contract. Those remain separate prerequisites.

## Damage: permanent-source lifelink and LKI

For damage to players and permanents, the shared damage executors read the
source's live characteristics or retained battlefield LKI before damage
notifications. Lifelink completes before damage notifications. A departed source with lifelink still causes its last controller
to gain life (CR 608.2h, 702.15b–d). The owner is used only when that source view
has no controller. Gained and lost keywords are read from the permanent view,
not from the printed card. The beneficiary is saved before damage callbacks.

Lifelink uses the amount dealt after prevention and replacement. It uses the
shared life-gain modifiers and replacements for both recipient categories.
`EvtLifeGained` reports the amount actually gained. Full prevention and life
gain replaced with card draws produce no life-gain event. Game clones change
only their own life totals. Absent and phased-out recipients have no damage
result.

Evidence: `TestDamageLifelinkSourceLKI`,
`TestDamageLifelinkCurrentCharacteristics`, `TestDamageLifelinkReplacement`,
`TestDamageLifelinkCloneAndAbsence`,
`TestDamageLifelinkCapturesControllerBeforeCallbacks`,
`TestDamageLifelinkAbilityAfterSourceDeparture`,
`TestDamageLifelinkNotificationResults`, `TestDamageLifelinkNoControllerUsesOwner`,
`TestDamageLifelinkIllegalTarget`, and
`TestDamageLifelinkSacrificedSourceResolution`.

This was a limited prerequisite, not the complete source-identity contract.
The damage action contract below now adds incarnation selection and retained
LKI for permanent lifelink. Spell-source LKI, protection, and deathtouch still
require separate work. No typed damage adapter or simultaneous damage
instruction is supplied.

## Captured permanent identity and retained information

`Game.PermanentRef` captures a present permanent's card UUID and battlefield
incarnation in an opaque, comparable value. `Game.LookupPermanentRef` prefers
a live permanent only when both identities match. Departure records retain
information separately for each captured incarnation. Returning and departing
the same card again does not overwrite an older reference. The zero reference
returns nil. Capture of an absent or phased-out permanent fails.

Phase-out captures information for the source and its indirectly phased
attachments before any member of the group phases out. While absent, a captured
reference reads that information. Phase-in restores live reads with the same
identity. A later phase-out replaces the phase-time information for that same
incarnation. Phasing does not write legacy zone-leave LKI or fire zone events.

The zone subsystem retains only explicitly captured incarnations, for the
lifetime of the game. Cleanup still clears legacy UUID LKI. Clone maps are
independent; retained snapshots can be shared internally because lookup returns
a copy of the snapshot's permanent fields. Existing card and ability definition
ownership remains shared and read-only. Live reads use the existing `LKIView`
contract. This API is an engine service, not the eventual sealed author reader.

Evidence: `TestPermanentRefIncarnations`, `TestPermanentRefPhasing`,
`TestPermanentRefCloneAndReadIsolation`, `TestPermanentRefSimultaneousDeparture`,
and `TestPermanentRefSacrificePayment`. Rules: CR 400.7, 608.2h, 702.26b–g.
Synthetic cards exercise current and departed keywords, controller versus owner,
a static boost during simultaneous death, source sacrifice payment, cleanup,
and clone isolation.

At this earlier slice boundary, no damage executor consumed these references. Source action capture,
spell-source identity, protection and deathtouch integration, target legality,
replacement ordering, damage outcomes, and batch damage remain later contracts.
Existing production damage and other UUID-based callers remain compatible.

## Ability instances: original permanent source identity

Ordinary battlefield event/state triggers capture a `PermanentRef` at trigger
time, before later zone changes and stack announcement. Self leave triggers
capture the departure snapshot; simultaneous leave triggers capture the source
from the existing pre-batch observer in the receiving game. The retained map
belongs to that game, so clone departures remain independent. Activated
abilities capture after successful payment preparation and before payment
commit, including source-sacrifice costs. Failed preparation creates neither
an instance nor a retained reference and leaves the source present. This does
not change the existing payment service's handling of errors during commit.

`StackObject.SourcePermanentRef()` exposes the immutable reference value.
Pending-to-stack transfer, ability copying (CR 707.10b), and cloning preserve
it. Copy controller and source controller remain distinct. Characteristics
are read at query time (CR 113.7a, 608.2h); current information is used only for
the original incarnation. Departure and phase-time LKI survive cleanup. A
returned card does not replace the source (CR 400.7).

`Game.ResolvingSourcePermanentRef()` exposes the instance reference throughout
resolution, including target checks. Deferred scope restoration covers normal
exit, target failure, and nested resolution. Resolution-state clones retain
the value. Outside resolution, and for uncaptured/non-battlefield sources,
the query returns zero. Target legality still controls whether effects execute.

Evidence: `TestAbilitySourceRefActivation`, `TestAbilitySourceRefPendingTriggers`,
`TestAbilitySourceRefPhasingAndFailure`, `TestAbilitySourceRefResolutionScope`,
`TestAbilitySourceRefIllegalTarget`, and the synthetic
`TestAbilitySourceRefSacrificeResolution`. They cover owner/controller separation,
source payment/departure/return, current keywords, pending and stacked clones,
copies with another controller, simultaneous deaths, cleanup, phasing, failed
payment preparation, target failure, nested resolution, and absence.

This earlier slice extended the identity service only. It superseded the earlier statement
that no ability instances consume permanent references. At that boundary, damage executors and
legacy damage effects still selected UUID sources. Spell, non-battlefield, and
delayed-trigger identities, damage replacement/action propagation, typed
registration/scope rules, damage batches/results, and card migrations remain
separate work. No replacement, payment output, or AI decision contract changed.

## Damage actions: permanent source identity and lifelink

The shared damage service captures the resolving ability's `PermanentRef` when
the requested source UUID matches it. Otherwise it captures a present permanent.
It saves the reference before replacement processing. A different requested
source cannot inherit the resolving source's identity. An absent source without
an ability capture retains the legacy UUID lookup contract.

Both damage action types expose `SourcePermanentRef() PermanentRef`.
`WithAmount` preserves it. `DamageToPlayerAction.RedirectToPermanent` and
`DamageToCreatureAction.RedirectToPlayer` return new actions that preserve the
reference, source UUID, amount, and combat flag. All five built-in redirections
use these methods. Existing UUID-only constructors remain compatible and return
a zero reference; engine replacements must use the redirection methods to retain
a captured identity. No mutable reference setter is exported.

The lifelink executors read the captured incarnation's current or retained
information (CR 113.7a, 400.7, 608.2h, 702.15b–d). Departure, return, later
departure, cleanup, and source phasing cannot substitute a different incarnation.
A nonzero reference with no information never falls back to UUID lookup.
The source's controller, or owner only if it has no controller, gains life;
copy controller and source controller remain separate. The existing prevention,
replacement order, life-gain replacement, and target legality services still
determine whether damage occurs and how much life is gained.

Evidence: `TestDamageSourceRefAbilityLifelink`, `TestDamageSourceRefCopyClone`,
`TestDamageSourceRefDifferentSource`, `TestDamageSourceRefActionTransforms`,
`TestDamageSourceRefReplacementOrder`, `TestDamageSourceRefRedirectedAbility`,
`TestDamageSourceRefSimultaneousDeath`, and synthetic
`TestDamageSourceRefReturnedSourceResolution`. Existing lifelink, target-failure,
payment-failure, phasing, and source-reference tests remain regression coverage.
This establishes action identity propagation and exact permanent lifelink only.
Protection, deathtouch, notification source identity, UUID history, recipient
incarnations, spell sources, delayed sources, batches, results, typed declarations,
and card migration remain separate contracts. Payment transactions are unchanged.

## Damage prevention: permanent source filters

`damagePreventionRuleReplacement` now evaluates its `from` filter against the
permanent incarnation captured by the damage action. A present source supplies
current characteristics. A departed or phased source supplies a copy of its
retained battlefield snapshot, including effective colors and controller.
Cleanup does not discard that captured information. Returning the same card,
even followed by another departure, cannot replace the original snapshot.

This applies to player-scoped, source-only player, and permanent-recipient
rules. Existing recipient filters, combat restrictions, player restrictions,
one-shot consumption, and recipient-chosen replacement ordering remain in the
shared pipeline. Redirection and amount changes retain the source reference.
Copies and game clones use their own state and prevention consumption. Missing
captured sources do not fall back to a later incarnation. UUID-only actions
without a captured reference retain live-permanent lookup.

Filters must read the supplied permanent for source characteristics. A custom
callback that looks up that UUID again in the game can read another incarnation;
this change does not rewrite arbitrary callbacks or create a frozen game.
Captured spell sources have no battlefield incarnation and do not match these
permanent filters. Unknown captured incarnations cannot use a live UUID match.
The phase view uses a copy of the retained permanent; the departure view uses
`ZoneSystem.LKIByIncarnation`, which survives ordinary LKI cleanup. Tests check
that mutations of either departure view or clone view cannot change retained
counters or types.

The separate color/type prevention implementations, protection,
deathtouch, source notifications, recipient incarnation, batch damage, and typed
adapters remain separate work. No payment or action transaction changed.

Evidence: `TestDamageSourceFilterIncarnation`, `TestDamageSourceFilterCopyClone`,
`TestDamageSourceFilterRedirect`, `TestDamageSourceFilterMissingSource`,
`TestDamageSourceFilterReplacementOrder`, `TestDamageSourceFilterSimultaneousDeath`,
`TestDamageSourceFilterCurrentAndRetainedViews`, and
`TestRONPreventionSourcePrerequisite`. Rules: CR 113.7a, 608.2h,
609.7b–c, 615.6/615.9, 616.1/616.1f, and 702.26b–d.

The 2026-10-04 implementation supplies this contract. Before this slice, this
section described behavior that was absent from `Matches`, and the named
source-filter tests were absent. The new RON integration test executes complete
Prodigal Sorcerer and Grapeshot Catapult records with existing engine prevention
rules. It covers departure, return to another controller, phasing, copied
activations and cloned pending resolution. It does not compile the collection's
unsupported `PreventDamage` declarations. The finite-shield allocation contract
in CR 615.7 remains separate.

## Damage prevention: color and card-type source characteristics

`Game.AddColorPrevention` and `Game.AddTypePrevention` now match the permanent
incarnation saved in `DamageToPlayerAction.SourcePermanentRef`. Current
characteristics apply while that incarnation is present. Retained departure or
phase-time characteristics apply after absence, including after cleanup and
a later incarnation's departure (CR 608.2h, 609.7b, 702.26b–d). Missing captured
information never falls back to a returned card. Color overrides and current
type attributes apply instead of printed mana colors and printed card types.

Rules remain restricted to the installed player. The first matching damage
event is fully prevented and consumes the rule. A nonmatching event does not
consume it; cleanup removes unused end-of-turn rules (CR 615.3/615.6). Existing
replacement choice and rechecking supply CR 616.1/616.1f ordering. Amount
changes and redirection back to the protected player keep source identity.
Ability copies retain their original source independently of copy controller;
game clones consume only their own rules.

Uncaptured live permanents use current information. Color matching for other
uncaptured sources reuses `Game.EffectiveColors`, including current stack and
resolving spell overrides. Type matching for nonpermanent uncaptured sources
keeps the existing card lookup. This is not a spell-source identity contract.

Evidence: `TestDamageCharacteristicPreventionIncarnation`,
`TestDamageCharacteristicPreventionCopyClone`,
`TestDamageCharacteristicPreventionRedirectAndAbsence`,
`TestDamageCharacteristicPreventionReplacementOrder`,
`TestDamageCharacteristicPreventionExcludesReturnedSource`,
`TestDamageCharacteristicPreventionLegacyAndLifetime`,
`TestDamageCharacteristicPreventionSimultaneousDeath`, and synthetic
`TestDamageCharacteristicPreventionSequence`.

Source selection, chosen-source shield identity, artifact redirection matching,
protection, deathtouch, source notifications, recipient incarnation, batches,
results, typed declarations, and card wiring remain separate work. Existing
payment, target legality, registration, and AI contracts are unchanged.

## Damage prevention: permanent protection

Protection is a prevention candidate in `EffectManager.ApplyReplacements`, not
an early damage-service return (CR 702.16e, 615.1, 616.1/616.1f). The affected
recipient chooses between protection and other applicable replacements. After
redirection, the pipeline reads the new recipient's current protection. It
reuses current runtime abilities, including wrapped grants, face-down handling,
and the existing Aura exemption. A phased or absent recipient supplies no
protection. Fully prevented damage produces no damage history, loyalty removal,
or lifelink gain (CR 615.6).

Source characteristics come from the action's `SourcePermanentRef`. Current
characteristics apply to that incarnation while present; retained LKI applies
after departure or phasing (CR 608.2h, 609.7c, 702.26b–d). Missing captured
information does not fall back to a returned card. Effective colors, card types,
and subtypes supply the built-in protection qualities (CR 702.16a).
Uncaptured nonpermanent sources retain prior card/effective-color lookup.
Custom CardFilter callbacks retain their own query contracts. No spell-source
identity guarantee is added.

The local protection candidate has no stored game binding or consumable state.
Each action constructs its own candidate. Stack copies keep their original
source; cloned games read only their own current state and LKI. Legacy damage
and fight effects delegate protection to the shared damage pipeline. Target
legality still applies before resolving a targeted instruction. The existing
`AttrDamageCantBePreventedOrRedirected` bypass includes protection for direct
permanent damage; general CR 615.12 and redirected bypass ordering remain
outside this contract. No payment transaction or simultaneous-action service
changed.

Evidence: `TestDamageProtectionIncarnation`, `TestDamageProtectionOrdering`,
`TestDamageProtectionRedirectAndBypass`, `TestDamageProtectionCopyClone`,
`TestDamageProtectionAbsenceAndOutcomes`, `TestDamageProtectionSimultaneousDeath`,
`TestDamageProtectionExcludesReturnedCharacteristics`, and synthetic
`TestDamageProtectionSequence`.

## Damage: permanent-source deathtouch and state-based destruction

The shared permanent-damage executor saves whether positive damage was dealt
by a source with deathtouch, separately from actual marked damage. It reads the
action's captured `PermanentRef` through the existing current/LKI service before
lifelink or damage notifications (CR 120.3e, 120.5, 702.2b/d/e). Current gained
or lost keywords apply to a present source; departure, return, later departure,
cleanup, and phasing cannot substitute another incarnation. Missing captured
information supplies no keyword. UUID-only actions keep live-source lookup.
This establishes permanent-source behavior, not spell-source identity/LKI.

Replacement and prevention determine the final amount and recipient first.
Zero and fully prevented damage record no deathtouch fact. Partial prevention
retains the exact marked amount. Loyalty, lifelink, and damage events retain
their existing amount contracts. A noncreature planeswalker loses loyalty but
is not destroyed by deathtouch. A recipient that becomes a creature before the
check can be destroyed; a creature that stops being a creature is not.

Each state-based check consumes the fact, including for indestructible,
noncreature, and phased-out recipients. A current creature with positive
toughness is destroyed in the existing simultaneous zone-change batch
(CR 704.3, 704.5h). Increasing toughness does not save it. Indestructible prevents
destruction. Regeneration replaces destruction once, clears damage and the
fact, taps the recipient, and uses existing combat removal. Zero toughness
uses the existing graveyard action and cannot be replaced by regeneration.
Damage itself does not destroy a recipient during resolution.

The fact belongs to a permanent incarnation. Return creates a fresh permanent.
Copy-on-write and payment-validation cloning preserve the fact and isolate its
consumption. Ability copies preserve source identity independently of copy
controller. Copies of abilities now also retain their original `TargetSource`
when there is no copied spell card; this preserves creature-target legality
and prevents a nil-card panic found by the synthetic copy acceptance test.
No payment transaction or declaration validation contract is changed.

Evidence: `TestDamageDeathtouchResults`, `TestDamageDeathtouchSourceIdentity`,
`TestDamageDeathtouchCloneAndCheckLifetime`,
`TestDamageDeathtouchSimultaneousDeaths`, `TestDamageDeathtouchBeforeNotifications`,
`TestDamageDeathtouchCopyClone`, `TestDamageDeathtouchCheckBoundary`,
`TestDamageDeathtouchCapturedAbsenceAndTargets`,
`TestDamageDeathtouchRedirectAndProtection`, `TestDamageDeathtouchReplacementOrder`,
`TestDamageDeathtouchRecipientTypes`,
`TestDamageDeathtouchBecomesCreatureBeforeCheck`,
`TestDamageDeathtouchPaymentClone`, and synthetic `TestDamageDeathtouchSequence`.

Typed damage roles, spell-source identity, notifications, history identity,
recipient references, batches, result bindings, and production card wiring
remain separate work. The legacy `BasiliskTouch` marked-damage approximation
is explicitly unchanged; it is not deathtouch and needs a separate rules audit.

## Damage redirection: permanent artifact-source matching

`Game.SetArtifactDamageRedirect` retains its current-cycle lifetime and player
restriction. Its replacement reads the damage action's captured permanent
incarnation through `damageSourcePermanentView`. Current effective card types
apply while present; retained departure or phase-time LKI applies while absent
(CR 113.7a, 608.2h, 609.7b–c, 702.26b–d). A returned incarnation, including
its later departure, cannot replace that view. Missing captured information
rejects matching. Uncaptured live sources use current types. Other uncaptured
sources retain card lookup; this does not establish spell identity/LKI.

The existing replacement service preserves recipient choice and rechecking
(CR 616.1/616.1f). Redirection keeps source reference, amount and combat flag,
then applies the recipient's prevention/protection rules. An absent or phased
recipient makes the rule inactive. The installer supplies no untapped condition;
that remains a declaration responsibility. Copy controller does not select a
different source. Game clones read their own source state and damage recipients.
No payment transaction, UUID history, notification, batch, or target contract
changed.

Evidence: `TestDamageArtifactRedirectSource`,
`TestDamageArtifactRedirectAbsenceAndLegacy`,
`TestDamageArtifactRedirectCopyClone`, `TestDamageArtifactRedirectOrdering`,
`TestDamageArtifactRedirectSourceChangeDuringReplacement`,
`TestDamageArtifactRedirectPrintedTypeLoss`,
`TestDamageArtifactRedirectSimultaneousDeath`, and synthetic
`TestDamageArtifactRedirectSequence`. Production cards remain unmigrated.

## Damage events: immutable permanent-source colors

Both shared damage executors capture source colors before applying damage
results that emit life or damage notifications (CR 120.4, 603.2, 608.2h).
The action's permanent reference supplies current colors for the original
incarnation, or retained departure/phase-time LKI. A returned incarnation
cannot supply these colors (CR 400.7, 702.26b–d). The event stores the colors
as an immutable scalar value with source UUID and incarnation, independently
of subsequent source changes, cleanup, event copying, and game cloning.

`EventSourceWasColor` and `EventSourceWasNotColor` use this capture. Missing
captured information matches neither; known colorless information can match a
negative color condition. Events without a capture retain their existing UUID
lookup contract. Nonpermanent sources are outside the new capture guarantee.
Other event predicates remain unchanged and must not be assumed to have the
same guarantee. No spell identity or complete event participant API is supplied.

Replacements still determine the action and actual amount before execution.
Fully prevented damage emits no event. This slice changes no replacement
ordering, target selection, payment, batch, or AI decision contract.

Evidence: `TestDamageEventColors`, `TestDamageEventColorsNotificationAndCopy`,
`TestEventSourceColorsValueIsolation`, and the synthetic test-harness
`TestDamageEventColorsSequence`. They cover both recipient paths, gained/lost
colors, departure/return, cleanup, phasing/phase-in, missing captures, life
notification changes, copied abilities, clone isolation, partial/full prevention,
and immutable event copies. Existing source, target, replacement, and payment
contract tests remain in the required engine suite.

## Damage events: immutable permanent-source controller

Both shared damage executors save the source controller before damage results
can emit life or damage notifications (CR 120.4, 603.2, 608.2h). The action's
captured `PermanentRef` selects the original incarnation's current controller
or retained departure/phase-time LKI. A returned card cannot substitute its
controller (CR 400.7, 702.26b–d). Cleanup does not remove the saved event value.
Owner and copied ability controller are independent of source controller.

`core.GameEvent.DamageSourceController` is an immutable scalar
`core.EventSourceController`, with source UUID, incarnation, controller UUID,
and captured/known flags. Copies and clones can share this value without
sharing mutable fields. `EventSourceControlledByController` and
`EventSourceControlledByOpponent` read it. Unknown captures and known sources
without a controller match neither. Uncaptured events retain prior UUID lookup.
Nonpermanent source identity and other event predicates remain separate work.

The existing replacement service supplies the final action and amount before
capture. Fully prevented damage emits no event. No target, payment, simultaneous
batch, declaration, or AI contract changes. Source type/subtype filters and
self matching must not be assumed to use this controller capture.

Evidence: `TestDamageEventController`,
`TestDamageEventControllerNotificationAndCopy`,
`TestDamageEventControllerLegacyAndUnknown`,
`TestDamageEventControllerPermanentNotification`,
`TestEventSourceControllerValueIsolation`, and the synthetic
`TestDamageEventControllerSequence`. Cases cover both recipient paths,
departure/return, cleanup, phasing/phase-in, owner/controller separation,
missing captures, later control changes, life-loss/lifelink notifications,
actual amount after prevention, copied abilities, clone isolation, immutable
value copies, and legacy event lookup.

## Damage events: immutable permanent-source types and subtypes

Both damage executors capture effective source card types and subtypes before
life or damage notifications (CR 120.4, 603.2, 608.2h). The shared captured
permanent reference supplies current original-incarnation information or
retained departure/phase-time LKI. Returned incarnations cannot replace it
(CR 400.7, 702.26b–d). Face-down and continuous subtype/type changes use the
existing permanent characteristic queries.

`core.GameEvent.DamageSourceTypes` contains `core.EventSourceTypes`. Its
constructor copies input slices; private storage and predicate-only inspection
let event copies and cloned pending resolutions share the value safely.
`Captured` distinguishes unknown captured sources from legacy events. Unknown
captures match no type/subtype and never fall back to UUID lookup.
`EventSourceHasType`, `EventSourceHasSubType`, and `EventSourceWasOfType` use
this value when captured. Their existing lookup remains for other events and
uncaptured nonpermanent sources.

Evidence: `TestDamageEventTypes`, `TestDamageEventTypesNotificationAndCopy`,
`TestDamageEventTypesLegacy`, `TestDamageEventTypesCurrentCharacteristics`,
`TestEventSourceTypesImmutable`, and synthetic
`TestDamageEventTypesSequence`. They cover both recipients, gained artifact
type, subtype overrides, absence, return, phasing, cleanup, unknown LKI,
notification-time changes, actual damage after partial prevention, full
prevention, copy/clone isolation, and immutable input copies. Existing target,
replacement ordering, simultaneous departure, and payment transaction services
are unchanged and remain covered by the full regression suite.

General permanent filters, spell-source and recipient identity, typed damage,
batches, and result bindings remain open. Self-incarnation matching is described
in the following contract.
Production cards remain unmigrated.

## Damage: self-source trigger identity

The retained `EventSourceIsSelf`, `EventSourceIsSelfDamageToPlayer`, and
`EventSourceIsSelfDamageToOpponent` predicates compare a captured permanent
source's identity with the present ability source. They reuse
`GameEvent.DamageSourceTypes` for its immutable source ID and incarnation.
A source must be present, with known captured information and the same
incarnation. A returned card cannot match damage from its previous incarnation
(CR 400.7). An absent or phased-out permanent cannot match; phasing in preserves
identity (CR 702.26b–d). Unknown captures never fall back to UUID matching.

Damage to either recipient category uses this check. Player and opponent
restrictions remain separate checks. A copied ability retains its original
source, and each cloned game tests against its own battlefield. Source changes
in damage-result notifications cannot change the saved event identity.
An already pending trigger survives source departure and retains its original
controller (CR 603.3a); this change adds no resolution-time identity recheck.

Non-damage events and damage events without a permanent capture retain their
legacy UUID behavior. This does not establish spell-source identity, recipient
identity, general permanent-filter capture, or typed damage declarations.
Target legality, replacement order, simultaneous actions, and payment rollback
continue to use their existing services without changes.

Evidence: `TestDamageEventSelfIdentity`, `TestDamageEventSelfLegacy`,
`TestDamageEventSelfCopyClone`, `TestDamageEventSelfNotification`,
`TestDamageEventSelfPendingSurvivesDeparture`, and the synthetic
`TestDamageEventSelfSequence`.

## Damage events: permanent-filter source identity

`EventSourceMatchesPermanentFilter` now resolves a captured damage-source
incarnation through the same `damageSourceMatches` service as prevention filters.
A present incarnation supplies current information; departure and phasing supply
retained LKI. Returned incarnations cannot replace it (CR 400.7, 608.2h,
702.26b–d). Unknown captures, inconsistent source IDs, and missing incarnations
match no filter, including the zero filter. Non-damage and uncaptured events
retain current permanent UUID lookup, including failure after departure.

This contract concerns identity only. No immutable full permanent snapshot is
stored in the event. Filters that read power or game state still evaluate with
the supplied game, and later characteristic changes can affect a present source.
Exact event-time filter snapshots remain a prerequisite under CR 120.4 and 603.2.
No replacement ordering, target, batch, payment, or declaration service changed.

Evidence: `TestDamageEventFilter`, `TestDamageEventFilterLegacyAndInvalid`,
`TestDamageEventFilterCopyCloneAndNotification`,
`TestDamageEventFilterSimultaneousDeath`, and synthetic
`TestDamageEventFilterSequence`. Both recipient paths cover live, departed,
returned, phased, phased-in, and missing sources after cleanup. Copies keep the
original source/controller; cloned resolution leaves the original untouched.
Partial prevention preserves actual amounts; full prevention emits no event.

## Damage events: captured characteristic filters

The retained `EventSourceMatchesPermanentFilter` now evaluates type, subtype,
color, controller, and UUID filters from immutable captures. It reuses
`DamageSourceTypes`, `DamageSourceColors`, and `DamageSourceController`; neither
executor nor event storage changes. The zero filter and `AnyPermanent` also
support captures. `Not`, `And`, and `Or` propagate this capability only when all
leaves support it. Both composition constructors own a copy of their inputs.
These predicates read the source at damage execution, before result notifications
(CR 120.4, 603.2, 608.2h). Later changes, departure, phasing, return, cleanup,
event copies, and independent clone changes do not affect saved matches.

Required captures must be known and have the same source ID and incarnation.
Controller predicates also require a nonzero controller. Missing or inconsistent
information matches nothing, even under `Not` or `Or`. An unknown value is not a
negative characteristic. ID predicates compare UUIDs and do not replace the
separate self-source incarnation check. Ordinary permanent filtering and damage
prevention continue to read their existing live/LKI inputs.

Custom and other filters retain the previous identity-safe current/LKI contract.
A composition with any unsupported leaf retains that contract for all leaves;
it never mixes capture times with current-state queries. This is a compatibility
boundary, not an exact event-time adapter for arbitrary filters. Power, supertype, keyword, attachment, combat, and general game-state filter
capture remain open. Owner, name, and token capture are implemented below. No target, replacement, payment, batch, registration, or
AI service changed.

Evidence: `TestDamageEventCharacteristicFilter`,
`TestDamageEventCharacteristicFilterCompositionIsolation`,
`TestDamageEventCharacteristicFilterUnknownAndLegacy`, and synthetic
`TestDamageEventCharacteristicFilterNotification`, plus the existing
`TestDamageEventFilter` copy/clone, prevention, LKI, and simultaneous-death tests.


## Damage events: immutable source name, owner, and token status

The shared damage executors save `DamageSourceIdentity` from the action's original
permanent incarnation before life or damage notifications (CR 120.4, 603.2,
608.2h). The existing permanent-reference live/LKI service supplies name, owner,
and token status. Ownership is independent of control (CR 108.3, 111.2);
the service's existing ownerless-fixture fallback to controller remains unchanged.
Departure, return, phasing, cleanup, and later name/status changes cannot replace
event data. The capture contains only scalar values, so event copies and game
clones share no mutable capture storage.

`Named`, `OwnedBy`, `NotOwnedBy`, and `IsToken` use this information in
`EventSourceMatchesPermanentFilter`, including supported compositions. Capture
identity must agree with `DamageSourceTypes` and the event source ID. Missing or
unknown information cannot match under negation or disjunction. Owner filters
also reject a zero owner. A known empty name or nontoken status is a valid value.
Uncaptured events and other event types retain current lookup. Compositions with
unsupported filters retain the prior whole-composition live/LKI fallback.

Evidence: `TestDamageEventIdentityFilters`, `TestDamageEventIdentityUnknownAndLegacy`,
`TestDamageEventIdentityNotificationAndClone`, and synthetic
`TestDamageEventIdentityNotification`. Both recipient paths cover departure,
return, phasing/phase-in, cleanup, owner/controller distinction, composition,
notification timing, scalar event copies, and clone-local source information.
Existing damage-filter tests retain prevention, simultaneous-death, and pending
ability copy/clone coverage. Target legality, replacement ordering, payment
transactions, and batch execution are unchanged. Spell sources and other
intrinsic/game-state filter captures remain separate contracts.

## Control duration during phasing and printed keyword dependencies

A control effect on a phased target suspends application without expiring its
otherwise valid duration. Phase-in resumes the effect. Turn-limited durations
can expire during phasing; conditions that require an absent source or target
fact still end their duration (CR 611.2b, 702.26b/d/f). Clone expiration latches
are independent. Evidence: `TestControlEffectSurvivesTargetPhasing` and
`TestConditionalControlStillExpiresDuringPhasing`.

The targeted and global `RemoveAllAbilities` services expose their removal
membership to printed static effects. In layer 6, a matching removal changes
the existence of a printed grant, so the grant depends on removal (CR 613.8a/b).
The shared sorter applies removal first and execution checks current activity.
This applies to printed or directly installed removal services. Removal expiry
state is deep-cloned. It does not establish complete dependency detection for
all layer forms. Evidence: `TestAbilityRemovalOrdersBeforePrintedKeywordGrant`,
`TestPrintedAbilityRemovalOrdersBeforePrintedGrant`, and the RON Fortified Area/Goblin War Drums source-removal tests.

## Landwalk and phased defending lands

`HasLandwalkEvasion` tests lands controlled by the defending player, regardless
of their owner. Phased-out lands do not enable subtype or legendary landwalk
(CR 702.14c, 702.26b/d). Phase-in restores eligibility; changing control or
leaving the battlefield changes eligibility immediately. A clone's phasing
state cannot change the original query. Evidence:
`TestLandwalkEvasionPhasingAndController` covers all six supported subtype
landwalk attributes and legendary landwalk. RON Aura tests exercise this query
with real grants and phased defending lands. This fixes the query used by the
shared blocker-selection and declaration checks; it adds no replacement,
payment, target or event behavior.

## General static permanent sets

`MatchingPermanents` and `ControlledPermanents` select current, phased-in
permanents without an implicit creature restriction. `WithoutPermanent(set,
This)` excludes only the current source. In a `While` static rule,
`ModifyEachPT` and `GrantEachKeyword` use these sets on each layer pass.
Controller membership follows the source's current controller; owner does not
select the set. Source departure, phasing and printed ability removal suspend
its static effects. Returned objects have new incarnations and obtain effects
from current membership (CR 400.7, 611.3a–c, 702.26b/d).

A subtype-only selector can match a noncreature permanent (CR 109.2, 205.3m).
Keywords do not require a creature recipient. A P/T modification remains
applicable if the permanent becomes a creature; layer 4 animation and layer 7
base values precede the layer 7c modification (CR 208.3a, 613.1d, 613.4b/c).
An explicit creature filter excludes noncreatures until animation. Subtype
changes in layer 4 change later membership. This contract reuses existing
engine behavior and does not add Kindred metadata support or noncreature P/T
readers.

Evidence: `TestRONStaticPermanentNoncreaturesAndInstances`,
`TestRONStaticPermanentRootExclusionAndCreatureBoundary`,
`TestRONStaticPermanentCurrentMembership`, `TestRONStaticPermanentLords` and
`TestRONStaticPermanentCastTiming`. Tests distinguish independent factories and
sources, global/controlled sets, accumulated modifications, later entries,
animation, subtype expiry, ability removal/restoration, leave/return, phasing,
landwalk, cloned games and cloned stack resolution. No new cost, replacement,
LKI or simultaneous zone-operation contract is established.

## Static permanent animation

`AnimateEach(set, power, toughness, colors...)` is a `StaticRule` for `Static`.
It adds Creature in layer 4, optionally replaces colors in layer 5, and sets
base P/T in layer 7b. Existing types, subtypes, supertypes and abilities remain.
Nil colors preserve colors; a non-nil empty variadic slice makes the recipients
colorless. Printed static wrappers forward their P/T sublayer, so base sets
follow timestamp order before layer 7c bonuses/counters and layer 7d switches.

The effect manager retains selected incarnation references from layer 4 through
the later parts of the animation for one application (CR 613.6). Type addition
or layer-6 source ability removal does not change that selection. Each new
application selects current subjects. Phasing suspends source/subject membership;
departure ends the source effect, and returned objects have new incarnations.
Clone effect managers retain independent cycle state.

The animation detects membership-changing layer-4 dependencies by testing the
other type effect on an isolated game/effect clone. This uses the existing
sorter and does not establish complete dependency detection for all engine
forms. Ordinary creature combat eligibility applies to animated noncreatures;
animation does not supply an exception to Defender or an attack/block
prohibition. Summoning sickness and intrinsic land mana rules still apply.

Evidence: all `TestStaticAnimation*` engine tests and `TestRONStaticAnimation*`
RON tests, mapped in [the slice report](designs/mtgish-ron-static-animation.md).
Creature entry detection sees the animated power before its trigger is queued.
The event keeps that value after departure. No new replacement, payment-result,
APNAP, general LKI reader or simultaneous zone-operation contract is introduced.

## RON controller upkeep payment adapter

RON lowers adjacent fixed `MayCost(PayMana(...))` and
`Unless(CostWasPaid, [...])` declarations to `UnlessPaid(You, Costs(...), ...)`.
The stacked trigger keeps its controller as payer. Source control changes do
not transfer that payment; an unpaid sacrifice still requires that player to
control the original source incarnation. RON controller damage uses
`DealDamageFrom(This, You, ValueAtResolution(...))`, preserving source LKI and
prevention instead of applying life loss. Tests cover departure, return,
phasing, control changes and damage-event identity. This adds compiler support
for existing engine contracts. See the [shape and test report](designs/mtgish-ron-upkeep-payment.md).

## RON fixed draw and activation life cost adapters

RON symbol `DrawACard` uses `DrawFor(player, Fixed(1))`. Typed player targets
and the stacked controller remain valid after source departure or control
change. Draws use the shared replacement and empty-library services (CR 121).
Fixed nonnegative activation `PayLife(Integer(n))` uses fresh `LifePayCost(n)`
objects through the existing total-cost transaction. Life is paid at activation;
damage prevention does not prevent payment. Combined life costs must be
payable together; failure restores mana-source and tap costs (CR 119.4, 602.2).
Fixed `AnyManaColor` production uses `AddManaOfAnyOneColor`, which chooses one
color for the whole amount and retains immediate mana-ability execution.
This establishes compiler adapters for existing services, with no engine rule
change. Evidence: all `TestRONFixed*` tests, mapped in
[the slice report](designs/mtgish-ron-fixed-draw.md).

## RON spell-cast life trigger adapters

RON any-player spell-cast filters use `SpellCastMatchesCardFilters`; matching is
checked at cast time. The pending ability keeps its controller (CR 603.3a).
Optional fixed mana payment is offered during resolution (CR 603.5), even when
the source departs, returns, changes controller or phases out. A copy placed
on the stack does not emit a cast event. Failed casting does not trigger.
Adjacent paid continuations use `IfPaid` and the shared cost transaction;
refusal or inability gives no paid body. Nested pairs keep separate results.
This adds compiler support without a new engine contract. Evidence:
`TestRONSpellCastLife*`, mapped in [the slice report](designs/mtgish-ron-spell-cast-life.md).

## Selected discard without result bindings: simultaneous moves

`Discard(subject PlayerRef, amount ResolutionValue)` without result options
uses the same executor as legacy `DiscardCards(amount ValueSource)`.
The executor collects the selected cards, prepares each proposed effect-caused
`DiscardAction` through the shared replacement pipeline, moves the resulting
cards, then sends discard events. An event therefore sees all destinations
from that instruction. A replacement sees the hand before any selected card
moves. A replacement can prevent one discard without stopping other selected
cards or later instructions. Replacement rechecks remain in the shared service.

The existing affected-player choice, captured resolving controller, short-hand
behavior, graveyard visits, tracker events and clone state remain in use.
Nonpositive amounts do not ask for a choice. These changes do not establish a
new contract for discard result options, random discard or discard costs.
Rules: CR 101.3, 608.2c/d/f, 616.1/f and 701.9a/b.
Evidence: `TestSelectedDiscardBatchEvents`,
`TestSelectedDiscardBatchReplacementPreparation`, and
`TestRONFixedDiscardReplacementAndEvents`. Existing typed-discard tests cover
choice, short hands, absent subjects, amounts, declaration validation and copies.

## RON spell-cast event references

`WhenAPlayerCastsASpell(AnyPlayer, filter)` lowers to `SpellCast` with fresh
spell and caster bindings. The trigger captures the exact stack UUID and caster
at the event. `Trigger_ThatSpell` does not select a later cast or a copy;
`Trigger_ThatPlayer` does not change when the spell changes controller.
`ControllerOfSpell(Trigger_ThatSpell)` reads the current stack controller.
Event references remain distinct from target declarations (CR 608.2k).
Countering, copied triggers, clones and fixed optional payment use existing
engine services. Source departure or phasing does not erase a queued capture.
No engine execution contract changed. Evidence: `TestRONSpellCastReference*`;
[exact compiler contexts and mappings](designs/mtgish-ron-spell-cast-references.md).

## Temporary typed card-type additions

`AddCardType(PermanentRef, core.CardType) ContinuousModification` adds Creature,
Artifact, Land or Enchantment in layer 4 through the existing `Until` executor.
Definition validation rejects other card types. Existing types, colors, subtypes
and abilities remain; Creature also supplies power/toughness identity. In one
captured type effect, type additions precede subtype checks. Subjects retain
captured incarnations; absent/phased subjects are not captured, later phasing
suspends application, and returned objects do not inherit the effect. Cleanup
and cloned games use the existing independent effect lifecycle.

Combine `SetBasePowerToughness` (layer 7b), `ModifyPT` (7c) and subtype operations
in one installation. `AddSubtypeFamily` retains earlier subtypes;
`SetSubtypeFamily` replaces the selected family. CR 205.1b requires replacement
for becoming a named creature-type artifact creature. RON's explicit
`SetCreatureType` correction preserves this distinction for Factory and Statue.
`ValidateSubtypeFamily(SubtypeFamily, ...string) error` exposes the existing
constructor validation before factories are built; constructor panic behavior
is unchanged. No new exhaustive creature-subtype dictionary is claimed.

Evidence: `TestTemporaryCardType*` and `TestRONTemporaryAnimation*`, with exact
contracts and coverage in [the slice report](designs/mtgish-ron-temporary-animation.md).
No new event, replacement, payment or general LKI contract is introduced.

## RON temporary collection capture

`Until(EndOfTurn, ModifyEachPT(set, ...))` captures the selected phased-in
incarnations at resolution (CR 611.2a/c), then installs the adjustments in
layer 7c (CR 613.4c). `ControlledPermanents` uses the resolving controller.
A copied spell uses its own controller. Later entries, returned objects and
objects phased out during capture do not join the set. Later control changes,
combat removal and phase cycles do not remove captured members. Source departure
does not end the installed effect. Multiple adjustments install before
state-based actions use their result. Cleanup and cloned effects retain the
existing isolation contract. This uses existing engine services.

Evidence: `TestRONTemporaryCollectionRealCards`,
`TestRONTemporaryCollectionCopyController`,
`TestRONTemporaryCollectionSharedContexts`,
`TestRONTemporaryCollectionOpponentCombat`, and
`TestRONTemporaryCollectionHolyLightDeaths`. Bone Flute's transactional payment
regression is `TestRONTemporaryCollectionBoneFluteRollback`. No new replacement,
LKI query or simultaneous zone-operation contract is established.

## RON own-turn and upkeep activation timing

RON `ActivatedWithModifiers/3` with
`ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn|IsTheirUpkeep))` lowers to
existing action timing and condition services. The engine's activation path
checks the current controller's turn before costs. Upkeep also requires the
upkeep step. Timing is an activation instruction (CR 602.1b, 602.5), so a
pending ability resolves after a step or turn change. Normal activation
permission and tap/summoning-sickness checks still apply (CR 117.1b, 602.5a).
The mana solver uses the same restrictions and its queries do not pay costs.

Evidence: `TestRONActivationTimingLegality`, `TestRONActivationTimingScepter`,
`TestRONActivationTimingPriest`, `TestRONActivationTimingInstancesAndCopy`,
`TestRONActivationTimingRepeatAndInventory` and
`TestRONActivationTimingManaPlanning`. These reuse the existing transactional
payment, source/target capture, phasing, copying, clone and cleanup contracts.
No new engine execution service is introduced.


## Activation limit isolation

Intrinsic ActionDefinition limits are state of one ability on one permanent.
After successful payment, markActivatedInGame obtains the mutable source and
copies runtime abilities before it records the use. It leaves the card's
ability declaration and other game clones unchanged. The stack and mana
activation paths share this service. An action whose source has left needs no
lasting runtime use count. Granted-ability tracker handling stays unchanged.

Control changes and phasing retain limits (CR 602.5b, 702.26). Returning to the
battlefield creates an unused object (CR 400.7). Stack copies do not activate
abilities. Rejected activation does not consume a use (CR 602.2). Final cleanup
resets runtime counts, including phased sources, with clone isolation.

Evidence: TestActivationLimitIsolation, TestRONActivationLimitRealCards,
TestRONActivationLimitInstancesCopiesAndClone,
TestRONActivationLimitConjunctionAndRollback,
TestRONActivationLimitManaAndInventory and TestRONActivationLimitPhasedReset.
The RON adapter supports fixed positive maxima and once-only limits with
supported timing conjunctions. It adds no replacement or simultaneous-operation
contract.

## RON matching sacrifice activation cost adapter

`SacrificeAPermanent(filter)` in activation costs uses
`SacrificeMatchingIncludingSourceCost` and the shared total-cost transaction.
The activating controller chooses a matching controlled permanent, including
the source. Payment locks its incarnation and moves it to its owner's graveyard;
regeneration does not prevent sacrifice (CR 602.1a, 602.2, 701.21a). Failed
preparation leaves mana, tap state, permanents and stack unchanged. Costs are
built separately per factory. Existing ability-source, target and clone services
preserve pending resolution after payment, source departure and copying.

Evidence: `TestRONSacrificeCostRealCards`, `TestRONSacrificeCostSourceAndTargets`,
`TestRONSacrificeCostRollbackAndChoice`, `TestRONSacrificeCostInstancesAndCopy`,
`TestRONSacrificeCostGateTimingAndLimits` and `TestRONSacrificeCostDamageReplacement`.
`TestRONSacrificeCostManualManaAndQueries` covers immediate explicit mana activation
and the existing tap-source planner exclusion for sacrifice costs. No new engine
API, saved paid-result, automatic sacrifice planning, batch sacrifice or
replacement-order contract is added by this adapter.

## RON single-permanent counter placement

RON PutCounters with one supported permanent instruction lowers to AddCountersTo
through typed references and ValueAtResolution. It uses the original source or
target incarnation, and the existing Aura attachment/LKI reference. Departed,
returned and phased subjects cannot receive placement through stale references.
P/T counters use layer 7c. Counters persist through cleanup (CR 122.1a/122.2, 400.7).

Placement enters the shared AddCountersAction replacement pipeline. The target
controller chooses among competing replacements; each applicable replacement
acts once and conditions are checked again (CR 616.1/1f). A prevented placement
does not prevent later independent actions. Costs still use the shared transaction;
copies and clones preserve private target/source state. No engine execution
contract changed and no new multi-object, entry or removal contract is claimed.
Evidence: TestRONCounterPlacement*, with
[exact contexts and card mappings](designs/mtgish-ron-counter-placement.md).

## RON fixed source-counter activation cost adapter

Fixed source-counter costs use `RemoveCountersCost` and the shared total-cost
transaction. Counters are paid during activation; insufficient counters or later
mana failure restores the complete payment (CR 602.1a, 602.2/602.2b). Repeated
costs cannot reuse already paid counters. Source permission, phasing, timing and
activation limits use the existing activation service. Ability copies pay no new
cost, and source departure does not stop independent targeted damage. Source
regeneration keeps the original incarnation; a returned or phased source does
not receive a pending shield. Damage uses existing target checks and prevention.

Evidence: `TestRONCounterCostRealCards`, `TestRONCounterCostRollbackAndComposition`,
`TestRONCounterCostFactoryIsolationAndManaRollback`,
`TestRONCounterCostTargetsCopiesAndInstances`, and `TestRONCounterCostFormsAndTiming`.
See [the exact RON shapes and mappings](designs/mtgish-ron-counter-costs.md).
No engine service or simultaneous-operation contract changes in this adapter.

## RON basic token batches

The RON token adapter uses `CreateTokenCopies` with `You` and a validated immutable
`DefineToken` declaration. The resolving controller creates and owns the tokens
(CR 111.2). Source departure, phasing or later control change does not redirect a
pending creation. Copied stack objects use their own controller and retain X.
Nonpositive counts create no tokens. Printed token characteristics remain
independent of the creating spell or ability (CR 111.3).

RON derives an unnamed token's name from its subtypes plus ` Token` (CR 111.4).
Named tokens retain their specified names. The existing engine token batch makes
all accepted tokens visible together before entry events. RON gameplay evidence
includes Icatian Town's four-token batch, Thallid copied abilities, clone isolation,
all-illegal target rejection, payment rollback and token phasing/destruction.
No replacement, entry-counter or token-result contract is added by this slice.
Exact forms and tests: [basic token creation](designs/mtgish-ron-token-creation.md).

## RON saved sacrifice characteristics

The activation adapter uses `SacrificeCreatureIncludingSourceCostWithStats` or
`SacrificeMatchingIncludingSourceCostWithManaValue`, with per-factory number
bindings and `FromNumber`. The shared payment transaction saves current
characteristics before departure. Counters affect saved toughness. A returned
card, another payment, or a later characteristic change cannot replace the
number (CR 400.7, 602.2b, 608.2h). The shared engine keeps separate paid values in
pending activations, ability copies and game clones. A copy uses its own
controller for life or token creation. Nonpositive life gain does nothing.
Life Chisel's upkeep restriction applies during activation, not resolution.

Evidence: `TestRONSacrificeValueRealCards`,
`TestRONSacrificeValuePendingAndRollback`,
`TestRONSacrificeValueFactoriesAndSelfPayment`,
`TestRONSacrificeValueRealRestrictions`, `TestRONSacrificeValueCopyController`,
and `TestRONSacrificeValueInvalid`. These tests use complete RON records for
Diamond Valley, Life Chisel and Homarid Spawning Bed. No engine service, new
replacement operation, or simultaneous-operation contract is introduced.

## RON source counter payment adapters

RON activation counter choices use `RemoveAnyCountersCost` with a fresh
`NumberBinding`. The shared payment transaction locks one choice from zero to
current source counters and captures the amount only after successful payment
(CR 602.2/2b). RON result reads use `FromNumber`, so counter changes, source
return/control/phasing, copied stack abilities and clones cannot replace the
saved count. Ambiguous or unbound result declarations fail compilation.
`AddManaAmount` and `AddManaOfAnyOneColor` retain engine mana classification:
untargeted activated mana resolves immediately, including a zero count
(CR 605.1a, 605.2, 605.3b); targeted production uses the stack (605.5a).
Automatic mana queries exclude these choice costs and never make the choice.
This establishes compiler support for existing engine contracts, with no engine
rule change. Evidence: `TestRONCounterPayment*`, mapped in the
[counter payment report](designs/mtgish-ron-counter-payment.md).

## RON ordinary end-step events

The RON end-step adapter reuses `EachEndStep`, `YourEndStep`, `TurnStepBegins`,
`WhenEvent`, `BindPlayer` and `FromPlayer`. It establishes no new engine rule.
At the event boundary, the current source controller determines whether a
controller-only event matches (CR 603.2b). The pending ability keeps that
controller through placement and resolution (CR 603.3a). APNAP placement remains
in the shared trigger service (CR 603.3b). Event player and ability controller
are separate references, including on copies and in cloned games.

No end-step ability executes on detection. Entry after the boundary does not
repeat it, and cleanup is a different event (CR 513.2). Phased sources do not
trigger; a pending ability survives source phasing. Source references retain the
captured incarnation and cannot follow a returned card (CR 400.7). A pending
sacrifice cannot sacrifice a source now controlled by another player (CR 701.21a).
Typed targets are selected at placement and checked again at resolution
(CR 603.3d, 608.2b).

Evidence: `TestRONEndStep*`, mapped in
[the slice report](designs/mtgish-ron-end-step.md). Intervening-if, histories,
delayed triggers and other step player filters remain separate prerequisites.

## RON tapped-entry adapter

`AsPermanentEnters(ThisPermanent, [EntersTapped])` reuses the engine's existing
entry attribute and `prepareBattlefieldEntry`. It establishes compiler support,
with no engine rule change. The source enters tapped before entry observers
inspect it (CR 614.1d, 614.12). It does not become tapped from an untapped state,
so entry does not fire a tap transition event. Phasing keeps the current status
and does not repeat entry (CR 702.26b–d). A zone return applies entry to a new
permanent (CR 400.7); card copies retain the declaration. Entry does not impose
an untap restriction.

The six newly supported full records also exercise shared mana ability timing
(CR 605.1a, 605.3b), atomic activation costs, owner-directed sacrifice
(CR 701.21a), and batch destruction. Nevinyrral's Disk allows regeneration;
an artifact creature is selected once by its union filter. Death trigger conditions use
the complete battlefield before the batch. Phased permanents and lands are excluded;
indestructible permanents survive. These are existing engine services, not new
entry replacement ordering rules. General entry choices, copies, counters and
competing entry replacements remain outside this adapter.

Evidence: `TestRONEntryTapped*`, mapped in
[the tapped-entry report](designs/mtgish-ron-entry-tapped.md).

## RON creature death history counter amount

RON's exact `NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature))`
counter amount uses the existing `CreaturesDiedThisTurn` resolution value and
`AddCountersTo`. The value reads the current game's turn tracker at resolution,
including tokens, both players, earlier deaths and later deaths while the trigger
is on the stack (CR 700.4, 603.2). Copies and cloned games read their own history.
Simultaneous destruction uses the shared batch; regeneration does not cause a
death (CR 701.19a). Cleanup clears turn history while counters persist. The
subject still uses incarnation identity and excludes phased permanents
(CR 400.7, 702.26b). No engine execution contract changed.

Evidence: `TestRONDeathHistoryCards`, `TestRONDeathHistoryEvents`, and
`TestRONDeathHistoryCastingAndInstances`; retained engine evidence includes
`TestCreatureDeathCountActions`, `TestCreatureDeathCountResolutionCopyAndClone`
and `TestCreatureDeathCountSnapshot`. Other card types, combined filters,
intervening-if conditions and event-time history snapshots are not supported
by this RON slice.

## RON storage-land lifecycle adapter

The storage-land adapter reuses the existing untap decision and ability layer
services for a self-only optional untap rule. Ability removal disables the rule;
untap restrictions take precedence over the optional choice. The current
controller makes the untap choice (CR 502.3). A controller-upkeep trigger with
PermanentIsTapped(This) uses WithInterveningIf for detection and resolution
checks (CR 603.4). The source must still be the same phased-in incarnation;
control change alone does not invalidate its tapped-state condition.

TestRONStorageLands covers all five full FEM records, native land play/untap/
upkeep, counter accumulation and immediate transactional mana payments.
TestRONStorageInterveningIf and TestRONStorageUntapInstances cover source
return, control, phasing, ability removal, copies and clones. No new engine
execution rule or replacement/batch contract is introduced. Exact accepted RON
shapes and coverage limits are in [the ledger](designs/mtgish-ron-progress.md).

## RON next-untap restriction adapter

RON `PermanentDoesntUntapDuringControllersNextUntap/1` reuses `SkipNextUntap`
and the existing typed permanent references. Resolution installs a one-step
restriction on the present incarnation. Source departure does not cancel a
restriction on a target. A returned card cannot inherit it (CR 400.7).
Target legality is checked before resolution (CR 608.2b). Phased subjects cannot
receive it; phasing after installation preserves it (CR 702.26b/d).
The current controller's next untap step consumes the restriction, including
when the permanent is already untapped. Other untap effects remain legal.
Repeated applications expire during the same step (CR 502.3).

`TestRONSkipUntapRealCards`, `TestRONSkipUntapHomaridWarrior`,
`TestRONSkipUntapTargetLifecycle`, `TestRONSkipUntapCopiesAndInstances` and
`TestRONSkipUntapSharedContexts` cover RON execution through the existing
engine. `TestRONSkipUntapIllegalAnnouncementRollback` verifies retained
transactional payment behavior. No engine rule, replacement ordering or
simultaneous operation changes. Exact declaration limits are in
[the adapter contract](designs/mtgish-ron-next-untap.md).

## RON self static untap prohibition adapter

RON `PermanentRuleEffect(ThisPermanent,
[DoesntUntapDuringControllersUntap])` reuses the existing source-bound static
ability service and `AttrDoesNotUntap`. The turn service reads that attribute
when it determines which permanents untap (CR 502.3). It does not prohibit
untapping from other effects. The source must retain its static ability
(CR 604.2); ability removal disables this rule through the existing ability
layer (CR 613.1f). Each returned incarnation acquires its own static rule,
while pending untap actions retain the old incarnation (CR 400.7). Phase-in
restores the rule before untap selection (CR 502.1, 702.26b/d).

`TestRONStaticUntapBasaltMonolith`, `TestRONStaticUntapBrassMan`, and
`TestRONStaticUntapColossus` execute the complete real RON records, including
mana, optional payment, and upkeep-only activation behavior.
`TestRONStaticUntapNativeUpkeep` verifies native upkeep execution.
`TestRONStaticUntapLifecycle` verifies current control, phase-in, removal and
restoration of abilities, returned objects, factory isolation and game clones.
`TestRONStaticUntapInvalid` keeps unsupported roles and scopes rejected.
No engine execution rule, replacement ordering, or simultaneous operation was
changed. Aura-host, collection, conditional and temporary forms remain open.

## RON global landwalk blocking overrides

The RON global creature/landwalk shape lowers to `NullifyLandwalkEffect` through
`WithStaticAbility`. `HasLandwalkEvasion` ignores only that subtype's evasion;
it does not remove the keyword. `CanBlock` still checks other restrictions.
The native declare-blockers step uses both queries (CR 509.1b, 702.14c).
A legal block remains in place when the rule source leaves before damage.

`TestRONLandwalkRuleCards` covers the eight complete Legends records, including
Lord Magnus's two overrides and the three creatures' first strike and P/T.
It checks both defenders, source control, departure/return, ability removal,
phasing and clone isolation (CR 400.7, 604.2, 702.26b/d).
`TestRONLandwalkRuleInstancesAndLegality` covers all six supported subtypes,
independent factory instances, late entry, flying and another landwalk ability.
`TestRONLandwalkRuleNativeCombat` uses real Hidden Path and Deadfall records to
prove native block declaration and damage, including source departure after
blocking. These tests establish adapter reuse of existing engine behavior;
no new target, payment, replacement or event contract is added.

## RON choose-one spell mode adapter

`SpellActions(Modal_ChooseOne([...]))` lowers independent action plans to
`NewModalSpell`. Mode-local declarations use distinct internal names and fresh
binding instances. This uses existing announcement, payment, legality,
regeneration and copy services; it adds no engine execution service.

`TestRONModalBlastCards` executes both modes of Blue Elemental Blast and Red
Elemental Blast from all six RON reprints. It checks announcement, owner zones,
clone isolation and one mode prompt. `TestRONModalAnnouncementRollback` checks
wrong-mode targets, wrong colors, invalid mode choice and failed payment.
`TestRONModalResolutionLegality` checks color changes, departure/return,
phasing, regeneration and uncounterable spells. `TestRONModalCopyTargets` checks
retargeting with the chosen mode retained and independent original/clone state.
`TestRONModalInstances` checks mode and factory binding isolation;
`TestRONModalUntargetedAndPlayerModes` checks unchosen targets, explicit player
draws and untargeted life gain. The relevant rules are CR 601.2b-c, 608.2b,
700.2a/c/f/g, 400.7 and 702.26b. Declaration tests reject invalid arity, short
mode lists, incompatible reads, cross-mode reads and unsupported contexts.

RON return-to-hand adapter evidence: `PutPermanentIntoItsOwnersHand/1` lowers
to the existing typed `ReturnToHand` effect. No engine rule changed.
`TestRONReturnHandCards` executes Unsummon, Boomerang, Active Volcano and
Flash Flood, including both modal instructions and owner/controller separation.
`TestRONReturnHandIdentityAndLegality` verifies CR 400.7 and 608.2b incarnation
and target checks, CR 702.26b phasing, control changes and regeneration independence.
`TestRONReturnHandCopyAndInstances` verifies CR 700.2g retained mode, retargeting,
fresh declarations and clone isolation. `TestRONReturnHandSourceAndHostContexts`
verifies activation/entry-trigger source reads and attachment LKI after payment.
This adds no simultaneous-operation or replacement-order contract.

## RON single damage recipient bindings

RON's single `AnyTarget` declaration and `Ref_AnyTarget` damage read now use
`DeclareDamageTarget`, per-instance `DamageTargetBinding`, and `DealDamageFrom`.
The adapter preserves the shared engine contracts for announcement, rollback,
resolution legality, incarnation identity, source LKI, mode copying and clone
isolation. `TestRONDamageScope*` verifies real RON damage to players, creatures
and planeswalkers; missing declaration rejection; departure/return; phasing;
source sacrifice; announced X; and recipient choice between competing damage
replacements. No engine execution behavior changed in this slice.

This does not establish a simultaneous shared-shield allocation contract.
The migration ledger records the CR 615.7 gap in `DamageSystem` that must be
resolved before RON future recipient prevention can be accepted.

`Permanent(Ref_TargetPermanent)` in RON single damage actions uses the local
`PermanentTargetBinding` as a `DamageRecipientRef`. `DealDamageFrom` retains
the target's incarnation and uses the shared source-LKI and replacement services.
`TestRONPermanentDamageRealCards` covers all rules of Crimson Manticore,
D'Avenant Archer, Grapeshot Catapult, Lady Caleria and Tor Wauki: exact damage,
combat/flying filters at announcement and resolution, tap/sickness/mana costs,
rollback, target return/phasing/control, source LKI with lifelink, prevention,
copies and clones. `TestRONPermanentDamageSpellAndTriggerBindings` covers
spell/mode/trigger targets, source departure, retargeted copies and independent
factory instances. `TestRONPermanentDamageReplacementOrder` verifies the
recipient controller's choice between redirection and amount prevention.
Declaration tests require a typed read and reject mode-scope leaks.
This adapter changes no engine execution contract and adds no damage batch.

## RON fixed-power targeting and temporary unblockability

RON lowers fixed `PowerIs` comparisons to `HasPowerLTE`, `HasPowerGTE` and
`HasPowerEqual`. The filters read current layered power at announcement and
resolution; they do not capture power when the factory is built. RON lowers
`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked],
UntilEndOfTurn)` to `Until(EndOfTurn, CantBeBlocked(typedTarget))`.
The existing per-instance permanent target, transactional cost, resolution,
combat restriction and duration services supply execution. No engine API changed.

The target must be legal at resolution (CR 608.2b). Once installed, the rule
keeps its subject incarnation (CR 400.7), survives source departure and ability
removal, and applies to later blockers (CR 611.2c). Phasing preserves identity
(CR 702.26b-d); cleanup ends the restriction even while the subject is phased
out (CR 514.2, 702.26f).

Evidence: `TestRONTemporaryBlockingRealCards` covers complete Dwarven Warriors
records from all six sets and Tawnos's Wand, including costs, resolution,
clones and cleanup. `TestRONTemporaryBlockingLegality`,
`TestRONTemporaryBlockingRollback`, `TestRONTemporaryBlockingCopiesAndBindings`
and `TestRONTemporaryBlockingPhaseAndSourceLifetime` cover power rechecks,
returned targets, phasing, source LKI, control, ability removal, failed payment,
summoning sickness, shroud, copied targets and fresh declarations.
`TestRONTemporaryBlockingSpellAndTriggerContexts` covers spell, mode and entry
trigger lowering; `TestRONFixedPowerFilters` checks signed bounds and changing
layered power. Invalid declarations and nested inventory roles have separate
tests. These tests establish adapter reuse of existing engine rules, not new
replacement or simultaneous-operation behavior.

## Matching untap restrictions at the turn-based decision

`PreventUntapDuringStepForMatching(PermanentFilter) ContinuousEffect` keeps a
static restriction separate from a granted ability. `Game.doUntap` selects the
restricted battlefield objects after phasing in and layer evaluation, before
any object untaps (CR 502.1, 502.3, 613.5). All matching restrictions contribute
to one local decision snapshot. A change during the untap operation does not
change that snapshot. The next untap decision uses current membership.

The check uses final power, including bonuses, reductions and switching. The
layer reset includes switched P/T state, so repeated evaluations give the same
result. An active printed source must remain on the battlefield, phased in,
and retain its abilities. Recipient ability removal does not remove this rule.
Changing control changes which player's untap step affects the object. Source
return creates a new active instance. Other untap effects remain permitted.
The decision snapshot belongs to the current call; it stores no shared game or
factory state. Existing `PreventUntapForMatching` callers retain their behavior.

Evidence: `TestMatchingUntapDecision` and `TestMatchingUntapDecisionSnapshot`.
Real-RON evidence: `TestRONCollectionUntapMeekstone` (all six reprints),
`TestRONCollectionUntapArena`, `TestRONCollectionUntapKarakas`,
`TestRONCollectionTapRiptide`, `TestRONCollectionUntapInvalid` and
`TestRONCollectionUntapDeclarationsAndInventory`. Collection taps reuse the
existing simultaneous tap contract (`TestTapEachTransitionsTogether`). Source
entry lowering uses typed events to avoid treating event context as a target;
source departure, phasing, return and copied triggers are covered by Arena.
This slice adds no payment, replacement-ordering or optional-choice contract.

## Random discard without result bindings: simultaneous moves

`DiscardAtRandom(subject PlayerRef, amount ResolutionValue)` resolves its player
and reads the amount once. It selects distinct cards from a local copy of that
player's hand with the game's random source. Each selected card consumes one
random value, including the last remaining card. Nonpositive amounts and empty
hands consume no random values. A short hand limits selection to its size.
Selection does not invoke the player's hand-choice interface.

The action uses `discardActionsTogether`: prepare effect-caused replacements
before movement, move the resulting cards, then send discard events. A prevented
card is not selected again to fill the amount. Other selected cards and later
instructions still execute. Rechecks and affected-player replacement choices
stay in the shared replacement service. A discard event sees all destinations
from this instruction; its graveyard visit is absent for a library destination.
The local hand copy and random queue remain isolated in game clones.
The existing API, definition validation and outcome metadata are unchanged.
This contract does not change random discard costs or legacy random helpers.

Rules: CR 101.3, 608.2b/c/f, 616.1/f and 701.9a/b.
Evidence: `TestRandomDiscardBatchEventsAndReplacements`, retained
`TestTypedRandomDiscard*`, and `TestRONRandomDiscard*`.

## Rampage: blocked transition and original source

`RampageTrigger(int) *GenericTriggered` listens to the source's transition to
blocked, including a creature entering as a blocker (CR 509.3c, 702.23a).
Adding another blocker to an already blocked attacker does not trigger it again.
Each printed instance has its own trigger. A single blocker still triggers the
ability, but supplies no bonus.

`RampageEffect(int) Effect` keeps its existing representation for current callers
and rampage detection. It reads the current combat group when the ability
resolves (CR 702.23b), then uses `Until(EndOfTurn, ModifyPT(This, ...))` to retain
the original source incarnation. Blockers removed before resolution do not
count; later changes do not alter the resolved bonus. A source removed from
combat has no group. A departed, returned, or phased source cannot receive the
pending bonus. A resolved bonus survives a phase cycle and expires at cleanup;
it cannot follow a leave-and-return cycle. Changing control removes an attacker
from combat, so its pending rampage has no blockers to count. Ability removal
does not cancel an already pending trigger or a resolved bonus.

The shared trigger, copy, clone, source-reference and continuous-effect services
own identity and lifecycle. Evidence: `TestRampageBlockedTransition`,
`TestRampageOriginalIncarnation`, `TestRampagePendingLifecycle`,
`TestRONRampageRealCards`, and `TestRONRampageNativeCombat`. The existing API
signatures and `RampageEffectData` remain available; a standalone blocker-declaration event no longer substitutes for a becomes-blocked transition.

## RON temporary keyword removal

RON now lowers `LosesAbility` inside the existing end-of-turn permanent layer
action to `RemoveKeyword(PermanentRef, Attr) ContinuousModification` or
`RemoveLandwalk(PermanentRef) ContinuousModification`. Temporary fixed landwalk
grants use `GrantKeywordModification`. No engine API or execution rule changed.
`Until` captures the subject's incarnation at resolution (CR 611.2c), and layer 6
uses the existing timestamp order (CR 613.1f, 613.7). Removal affects all earlier
instances of the selected keyword; a later grant can restore it.

RON evidence: `TestRONTemporaryKeywordRealCards` covers target legality,
summoning sickness, copies, departure, control, phasing, returned incarnations
and clone cleanup. `TestRONTemporaryKeywordScarwoodOrder` covers colored payment
rollback, repeated grants, timestamp order and landwalk evasion (CR 702.14).
`TestRONTemporaryKeywordSourceLandwalk` covers source-bound grants, independent
subsequent damage, source LKI, captured ability controller and damage prevention.
`TestRONTemporaryKeywordCopyRetarget` checks separate copied target frames.
The shared engine's `TestComposableKeywordRemoval` and
`TestComposableKeywordRemovalLifetime` remain the engine contract tests.

## Tapped-source continuous duration

`UntilSourceUntaps(This, modifications...)` uses the same capture and layer
installation as `Until`, with an incarnation-bound source lifetime. CR 611.2b
requires a present, phased-in, tapped original source when the instruction
resolves. An absent, untapped, phased or returned source installs nothing.
Untapping then retapping before resolution can start a new duration at resolution.

Once installed, actual untapping, source departure or source phasing ends every
installed layer permanently (CR 611.2a/b, 702.26f). Transition hooks record
expiration before untap notifications or another tap/phase-in can occur. The
simultaneous untap service completes all untaps, expires all affected layers,
and reapplies them before sending any untap event. A stun counter replacing
untap preserves the lifetime and emits no untap event. Indirect source phasing
also ends it. Layer activity provides a second check for absent, changed or
untapped sources. Expiration state is independent in cloned games.

Captured recipients and values use the existing CR 611.2c/d contracts. Target
phasing suspends application without ending a duration that tracks the source;
source expiration while the target is phased still ends the effect. A returned
target is a new incarnation and receives no old bonus. Source control changes,
ability removal and end-of-turn cleanup do not end this duration. Control layer
modifications still require their separate lifetime adapter and are rejected.

This service does not register a target-dependent untap policy. A separate
printed permission to remain tapped remains a choice even after a recipient
leaves. Existing legacy tap-maintained callers retain their behavior.

Engine evidence: `TestUntilSourceUntapsLifetime`,
`TestUntilSourceUntapsStartAndValidation`, and
`TestUntilSourceUntapsTransitionsBeforeNotifications`. Real RON evidence for
Tawnos's Weaponry, Spirit Shield and Zelyon Sword:
`TestRONSourceTappedDurationCards`,
`TestRONSourceTappedDurationUntapChoiceAndInstances`,
`TestRONSourceTappedDurationInvalidAndInventory`, and
`TestRONSourceTappedDurationTriggerAndDefinition`.

## RON opponent-land tap event adapter

The RON compiler now lowers the exact land-subtype/opponent conjunction in
`WhenAPermanentBecomesTapped` to the existing
`OpponentLandBecomesTapped().OfSubtype(...)` event pattern. It validates the
subtype against the same engine event declaration contract before returning a
factory. No engine event, timing, replacement or identity behavior changes.

CR 603.2e and 701.26a require an untapped-to-tapped transition on the
battlefield. CR 603.3a fixes the trigger controller at the event. The RON tests
verify these rules through real Lifetap and Lifeblood definitions, including
pending source/land departure, control changes, phasing, trigger copies and
clone isolation. Mana payment tests distinguish successful taps from failed
transactional casts. `TapEach` tests retain the existing simultaneous tap
service and queue one trigger per matching land.

Evidence: `TestRONOpponentLandTap*`, with retained engine tests
`TestOpponentLandBecomesTapped*` and `TestTapEachTransitionsTogether`.

## RON Aura host tap event binding

The RON Aura adapter lowers
`WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent))` to
`AttachedBecomesTapped(BindPermanent(...))`. Its action body reads the event
incarnation through `From`, using `Trigger_ThatPermanent`. Attachment changes
and source departure cannot redirect this reference. Host departure and return
produce a different incarnation; a phased host cannot be affected, while a
phase cycle retains identity. Trigger copies and cloned games keep their own
resolution captures. This adapter adds no engine rule.

The trigger is not targeted. Shroud does not stop its actions. Blight uses the
existing destruction service, including regeneration and subsequent Aura state
checks. Spirit Shackle uses the existing counter service with one -0/-2 counter.
CR 603.2e and 701.26a require a transition to tapped; entry tapped and a second
tap without untapping do not trigger. CR 603.3a retains the event controller.
Mana taps queue a normal trigger, and failed casting payments roll back taps
and their pending events. Simultaneous taps use the shared batch service.

Evidence: `TestRONHostTapRealCards`, `TestRONHostTapNativeCastingAndPayment`,
`TestRONHostTapPendingBindingsAndBatch` and `TestRONHostTapInvalid`, with retained
engine contracts `TestAttachedBecomesTappedCapture`,
`TestAttachedBecomesTappedValidation` and `TestTapEachTransitionsTogether`.
Controller queries that require LKI, source tap events and new damage-source
forms are outside this adapter.

## RON global combat prevention adapter

`CreatePreventDamageUntil(CombatDamageWouldBeDealt, [PreventThatDamage], UntilEndOfTurn)`
lowers to the existing `PreventAllCombatDamage` effect. The adapter changes no
engine rule. Resolution installs an end-of-turn `fogReplacement` in the shared
replacement pipeline. It has no battlefield source lifetime or recipient binding.
It prevents combat damage from current and later sources to either player or
permanents, through first-strike and normal damage steps and later combats.
Noncombat damage remains legal. Cleanup ends the effect (CR 514.2, 615.3).
A later resolution cannot change past damage (CR 615.4).

`TestRONGlobalCombatPreventionRealSpells` checks all Fog reprints, Darkness and
Holy Day through native casting and combat, including simultaneous attacker and
blocker damage, trample, first strike, lifelink, later sources, repeat combats,
noncombat damage and independent clone cleanup. `TestRONGlobalCombatPreventionSporeFlower`
checks upkeep counters, failed and successful counter payment, source departure,
return, phasing, control changes, copied activations and pending clone isolation.
`TestRONGlobalCombatPreventionLateCopyAndCounter` checks opponent-turn casting,
past damage, copied spells and countered spells. `TestRONGlobalCombatPreventionReplacementOrder`
checks affected-player choice with redirection and the existing unpreventable
recipient attribute (CR 615.6, 615.12, 616.1). Declaration and inventory tests
check exact argument kinds, arity, operation count, contexts and nested roles.

This adapter does not establish finite shield allocation, source-specific
prevention durations or new LKI reads. The CR 615.7 prerequisite remains open.

## Turn-long source combat prevention

`PreventCombatDamageByThisTurn(PermanentRef)` uses the same typed reference,
incarnation capture and `combatSourcePreventionReplacement` as the existing
end-of-combat constructors. Only its duration differs. Resolution installs the
replacement; payment and announcement install nothing (CR 615.3–615.4).
It matches combat packets by source ID and incarnation, for players and
permanents. Source departure does not erase a captured old-source packet;
a returned source gets no shield (CR 400.7). Control changes and a phase cycle
preserve identity (CR 702.26). Cleanup expires the replacement (CR 514.2).
A target's creature filter applies at announcement and resolution; this is
prevention by a captured object, not a dynamic creature-source filter.

`TestCombatSourcePreventionThisTurn` checks later-combat lifetime, old-source
packets after departure, return, phase cycles, control changes, player/permanent
recipients, unrelated sources, noncombat damage and independent clone cleanup.
`TestCombatSourcePreventionThisTurnDeclarations` checks bound, nil and unbound
references. Existing end-of-combat constructor tests still pass.
`TestRONSourceCombatPreventionCards` checks complete Horn of Deafening and Lady
Evangela records through native activation and combat, double strike, trample,
lifelink, simultaneous opposing damage, rollback, legality, copies and clones.
`TestRONSourceCombatPreventionSubdue` checks both spell instructions, independent
retargeted copy bonuses, countering and shared cleanup. Replacement-order tests
check recipient choice, redirection rechecks and unpreventable damage
(CR 615.6, 615.12, 616.1). Declaration tests check shapes, scopes and inventory.

No finite shield allocation or new damage batch contract is established here.
The CR 615.7 prerequisite remains open. No runtime registry cutover occurs.

## RON permanent targets controlled by the ability controller

`ControlledByAPlayer(SinglePlayer(You))` in `TargetPermanent` lowers to the
existing `TargetYouControl` option. Nested conjunctions preserve characteristic
filters and source exclusions. Each factory call creates fresh target bindings;
only immutable declaration options are shared. No engine rule changes.

Announcement and resolution use the stack object's controller, including for
copied spells and abilities (CR 601.2c, 602.2b, 603.3a, 608.2b).
Source departure or a later source control change does not change that
controller. Ownership is independent. A target control change before resolution
can make the target illegal. After resolution, a characteristic modification
keeps its captured recipient even if its controller changes (CR 611.2c).
Return creates a new incarnation; phasing preserves identity but can make a
pending target absent (CR 400.7, 702.26).

Evidence: `TestRONControlledTargetBattleGear` checks the complete Ashnod's Battle
Gear record, mana and tap payment, rollback, layered +2/-2, ownership, legality,
return, phasing, copies, clones, and source-tapped lifetime.
`TestRONControlledTargetUntapChoice` checks both optional untap decisions.
`TestRONControlledTargetBattleGearLethal` checks the toughness reduction with
native state-based actions. `TestRONControlledTargetDeclarationsAndInstances`
checks nested filters, source exclusion, distinct factory bindings and invalid
shapes. `TestRONControlledTargetSpellAndTriggerContexts` checks spell and trigger
controllers, source departure/control changes, opponent copies and clone
isolation. Opponent/owner expressions and controller restrictions inside `Or`
remain unsupported. This adapter adds no replacement or simultaneous-operation
contract.

## Captured combat prevention to and by one permanent

`PreventCombatDamageToAndBy(PermanentRef)` and
`Game.PreventCombatDamageToAndBy(uuid.UUID)` retain their signatures and
end-of-turn duration. Installation requires a current subject. A departed
subject cannot install a shield on a returned incarnation. This existing typed
reference rule is retained.

`creatureCombatDamagePreventionReplacement.Matches` now checks the captured
`DamageSource` ID and incarnation for outgoing combat damage. It no longer
requires that damage source to remain on the battlefield. Incoming combat
damage instead requires the live recipient's captured incarnation. A shield
on a returned permanent cannot match an old source packet with the same UUID;
an old shield cannot protect the returned recipient (CR 400.7, 609.7, 615.1–4).
A captured damage packet retains its source after departure, phasing and LKI
cleanup. This uses the existing damage snapshot service; no new history or
parallel damage pipeline is introduced.

The shield prevents combat damage to and by its subject until cleanup, including
later combats. It permits noncombat damage and does not change combat membership
(CR 506.4b). Control and a phase cycle preserve identity (CR 702.26); phasing
before resolution still makes an attacking target illegal. Prevented damage
produces no damage outcome or lifelink gain. Simultaneous packets retain
independent source/recipient checks. Existing affected-player choice and
replacement rechecks apply with redirection, and unpreventable damage passes
through (CR 615.6, 615.12, 616.1/1f).

Evidence: `TestCombatPreventionToAndByCapturedSource`,
`TestCombatPreventionToAndByRejectsOldSourceForNewShield`,
`TestCombatPreventionToAndByDepartedSubjectAtInstallation`,
`TestCombatPreventionToAndBySimultaneousPackets`, retained typed combat tests,
and `TestRONTargetCombatPrevention*`. The RON tests cover complete records for
Maze of Ith, Ebony Horse and Elvish Scout plus scoped spell/mode/trigger probes,
copy retargeting and fresh factory bindings. CR 615.7 finite-shield allocation
remains open; this unbounded shield introduces no allocation decision.

## RON hand-size upkeep execution

The adapter uses `HandSizeOf`, `SubtractValues`, `EachUpkeep`, `YourUpkeep`,
`TurnStepBegins`, `BindPlayer`, `FromPlayer`, and `DealDamageFrom`. No engine
rule or public API changes. Upkeep detection saves the event player and the
ability controller separately (CR 603.2b, 603.3a). Source control changes,
departure and phasing after detection do not redirect the pending event.
Copies keep the event player while `You` uses the copy's controller. Clones
keep separate pending resolution state.

Hand size is read at the instruction's resolution, after prior instructions
(CR 608.2c, 608.2h). Subtraction retains signed intermediate values; final
nonpositive life and damage amounts have no effect (CR 107.1b). Damage uses
native source LKI, prevention and lifelink. A returned source cannot replace
the captured source incarnation (CR 400.7, 608.2h).

`TestRONHandSize*` verifies full records for Ivory Tower, Storm World and Copper
Tablet, invalid declarations, lexical isolation, independent factory instances
and events, resolution-time hand changes, signed arithmetic, native boundaries,
source lifecycle, copies, clones, and prevention. Chosen entry players and
intervening-if upkeep conditions remain separate capabilities.

## Typed filtered blocking restrictions

`CantBeBlockedExceptBy(PermanentRef, PermanentFilter)` uses the continuous
modification and combat restriction services. It is a rule change, not a granted
keyword. Every filter registered for an attacker must pass for the blocker,
and `CanBlock` still checks flying, reach, protection and other restrictions
(CR 509.1b). The filter reads the blocker's current characteristics at the query.
Changing a restriction after a legal declaration does not remove the block
(CR 506.4a, 509.1b).

`While` supplies current source/attachment membership on each layer pass.
Removing the source ability stops its static rule. Removing a recipient's
abilities does not remove another source's rule. Source phasing suspends a
static rule; an Aura does not follow a departed and returned host. `Until` keeps
the captured recipient incarnation and the filter until cleanup, including
across a phase cycle; it does not follow a returned card (CR 400.7, 702.26).

`Game.Clone` copies the computed `cantBeBlockedExceptByRules` map and each
filter slice. Immediate block queries on a clone retain these rules before a
new layer pass. Clearing or rebuilding that map in either game cannot change
the other game. Filters are immutable declarations, as with the existing
combat constructors; this change introduces no game state in a filter.

Evidence: `TestFilteredBlockingCapturedRestriction` covers typed temporary
identity, ability removal, phase cycle, immediate clone queries and independent
cleanup. `TestFilteredBlockingDefinitionValidation` covers nil/unwritten
subjects and a valid typed target. `TestRONFilteredBlockingRealCards`,
`Lifecycle`, `CurrentFiltersAndIntersection`, `NativeCombat`, and
`InstancesAndStrictLoad` (with the same prefix) cover complete Invisibility,
Seeker and Elven Riders records, current filters, static lifetime, casting
rollback, native block declaration/combat, existing-block finality and separate
factory instances. Existing combat services still own declaration execution.

## RON Aura host-controller upkeep

The RON adapter reuses `AttachedControllerUpkeep`, `BindPermanent`, `BindPlayer`,
`FromPlayer`, `DealDamageFrom`, `AttachedPermanent`, and `AddCountersTo`. No
engine API or rule changes. The trigger checks the current attachment and its
controller when upkeep begins (CR 503.1a, 603.2b). It saves the event player
before stack placement. Its controller is the Aura's controller at detection
(CR 603.3a); subsequent control, attachment, phase, and zone changes do not
redirect the captured player. Copies keep that event player. Factory instances,
pending events, and clones keep separate binding state.

The resolving `HostPermanent` reference remains distinct from the internal
event host capture. It reads the current attachment, or the departed source's
attachment LKI. It cannot follow a returned source or returned host incarnation
(CR 400.7, 608.2h). A phased host is absent to the counter action; a phase cycle
retains its identity. Ability removal before detection prevents the trigger;
removal after detection does not cancel it (CR 113.7a). Damage uses the shared
source LKI and prevention services.

`TestRONHostUpkeepGameplay`, `IndependentBindings`, `DamagePrevention`,
`NativeCast`, `NativeStep`, `Reprints`, and `Invalid` in `pkg/mage/dsl/ron`
verify these contracts through complete RON records. The supported event is
`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(HostController)), body)`
on an Aura. Resolving `HostController`, optional payments, host-death events,
intervening-if host conditions, and host-controller end-step events remain
separate compiler work.

## Static collection attack restrictions

`CantAttackEach(PermanentSet) StaticRule` checks current members through
`Permanent.CanDeclareAsAttacker(*Game)`. Its static effect uses the existing
printed-ability lifetime service. Membership is queried after characteristic
layers, rather than encoded as a layer-6 attribute removal (CR 613.11). This
keeps flying, types, names and controllers current regardless of effect
timestamp order. Each active restriction can prohibit declaration (CR 508.1c).
The query does not make choices or mutate the collection.

`ControlledPermanents` reads the source's current controller. Removing a
recipient's abilities does not remove another source's restriction. Removing
the source ability or phasing/removing the source suspends the restriction
(CR 611.3a–b). Returned and newly entered objects use current membership;
there is no saved recipient list or LKI read. Changes after declaration do not
remove attackers from combat (CR 506.4a). Phased recipients cannot be declared.
Other attack restrictions, summoning sickness, tapping and Defender still apply.

The declaration validator sees the typed set and rejects missing sets and
unbound references. Factories create separate continuous effects; game clones
copy their source storage and retain immutable collection declarations.
`TestCantAttackEachFinalCharacteristics` covers both timestamp orders, recipient
ability removal and clone isolation; `TestCantAttackEachDefinition` covers
invalid declarations. RON tests named `TestRONAttackRule*` cover all three real
consumers, source and recipient control, phasing, return, suppression, independent
instances, strict loading, native casting/payment rollback and native combat.
No new optional costs, damage replacements or simultaneous operations are added.

## Static attachment control identity

`ControlAttached() ContinuousEffect` uses the source's current attachment and
current layer-2 controller. The host must match both the attachment ID and its
saved incarnation. A host that leaves and returns cannot receive control from
the old attachment, including the interval before state-based actions remove
an invalid Aura (CR 400.7, 303.4c, 611.3a–b). A new legal attachment records the
new incarnation and receives a new timestamp (CR 613.7e).

A directly phased source cannot supply this effect. Phasing the host also phases
its Aura; their attachment identity survives the phase cycle (CR 702.26b–d, g).
Tests that use the low-level phase methods explicitly recompute continuous
effects before reading characteristics. Layer-6 ability removal does not erase
control already applied in layer 2 of the same computation (CR 613.1b/f).

`TestControlAttachedKeepsHostIncarnation` covers departure/return, a new
attachment, phase cycles and clone isolation with synthetic engine cards.
`TestControlAttachedDirectPhaseAndLayers` covers repeated source phase cycles
and ability removal across repeated layer computations.
`TestRONStaticControlLifecycle`, `TestRONStaticControlNativeCasting` and
`TestRONStaticControlFactories` exercise these services through both real RON
Aura definitions, including current controller, competing timestamps,
can't-change-control restrictions, payment rollback and target legality.

## Static basic land types: dependency and attachment identity

`BecomesBasicLandsEffect(filter PermanentFilter, subtypes ...string) ContinuousEffect`
uses current, unphased lands selected by a pure filter. It replaces only the land
subtype family, suppresses printed and copied rules-text abilities, and supplies
intrinsic mana for each new basic type (CR 305.6–8). Other card types, supertypes,
creature subtypes, and abilities granted by separate effects remain. Each
application starts from printed characteristics. Constructor inputs are copied;
empty lists and nonbasic subtype declarations retain their declaration-time
rejection. The public signature is unchanged.

In layer 4, this effect depends on an effect that changes its selected lands.
A printed layer-4 effect also depends on a type effect that removes its source's
printed ability. Dependency trials apply an isolated effect to a cloned game;
they do not change live permanent state, chosen objects, mana, or transient
animation membership. Stateful effects use their existing clone contract.
Filters and effect callbacks must keep their writes in the supplied game.

The shared layer 3–6 executor reevaluates remaining dependencies after each
application (CR 613.8c). Independent effects use timestamp order. Dependencies
inside a strongly connected loop use timestamp order, while dependencies from
outside that loop still wait for it (CR 613.8a–b). This slice does not establish
CDA ordering or new layer-7 dependency behavior.

`AttachedEffect(layer Layer, apply AttachedApplyFunc) ContinuousEffect` now
requires the source's saved attachment incarnation in both its activity query
and its apply method. A departed and returned host cannot receive an old effect,
even before state-based actions remove its Aura. `Game.Attach` records a new
legal attachment. Direct source phasing suspends the static effect; indirect
phasing preserves the attachment. These static effects use live objects, not LKI.

Evidence: `TestBasicLandDependency*`, `TestBasicLandSetDeclarationIsolation`,
`TestBasicLandAttachedKeepsHostIncarnation`, and the retained basic-land-type,
attachment-timestamp, dependency, and clone tests. RON-backed regression evidence
is `TestRONLandDependencyPrerequisite` (Forest, Tropical Island, Savannah) and
`TestRONAttachedEffectsRejectReturnedHost` (Holy Strength, Flight, Grizzly Bears).
The RON static adapters now reuse these services for Conversion, Evil Presence
and Blood Moon. `TestRONStaticLandTypeDependencies` proves Blood Moon and
Conversion use dependency order in both timestamp orders, with current entries,
source departure, phasing, intrinsic mana and clone isolation.
`TestRONEvilPresenceCasting` covers announcement legality, payment rollback,
resolution legality, attachment incarnations and direct/indirect phasing.
`TestRONStaticLandTypeRemovesPrintedRules` proves printed rules disappear and
return, including failed activation without payment. `TestRONStaticLandTypeCombinedFactories`
checks all five basic types, separate Aura instances and retained creature
characteristics. `TestRONStaticLandTypeNativeSpells` covers native spell payment
and resolution. Conversion also runs every scenario in
`TestRONUpkeepPaymentRealCards`, including captured payer/source identity.

The collection adapter accepts one type operation and pure filters. It rejects
source-dependent selection and combined collection operations. The host adapter
uses the saved attachment identity and can use the existing host modifications.
No new engine, replacement, payment, or batch-operation API is added.

## RON conditional static layer contract

RON static `If` uses the existing `While` contract. Controller conditions read
current control of unphased permanents; `ControlsNo` uses the same current
membership with an absence check. An untapped condition reads the current source
or its current attachment with the attachment incarnation check. Static effects
use no last known information (CR 604.4, 604.7, 611.3a–c).

Conditional fixed bonuses apply in layer 7c; keyword grants apply in layer 6
(CR 613.1f, 613.4c). Source ability removal stops the static effect. Earlier land
type changes affect controller conditions before bonuses apply. P/T counters
remain separate from bonuses. Source phasing suspends grants, phase return
restores them, and a returned host cannot inherit an old attachment's grant.
The RON adapter rejects mixed layers until a shared membership contract supports
those bodies. Static layer execution and replacement services are unchanged.

`TestRONConditionalStatic*` verifies real casting and target legality, failed
payment rollback, current controller/membership and tap changes, regeneration,
layer ordering with Blood Moon and ability removal, attachment changes and
returned host identity, phasing, independent factories and cloned games.

`OpponentControls(PermanentFilter) StateCondition` checks the current opponent
of the condition context's controller (CR 102.2), not the active player or card
owner. The controller must identify a player in the game. The condition uses
the same opponent lookup and unphased battlefield query as
`WhileOpponentControls`; it holds no source, target or chosen game state.
`TestOpponentControlsState` and `TestOpponentControlsDefinition` establish
current control, absent/invalid controller rejection, phasing, departure and
return without LKI, clone isolation, definition inspection and resolution use.

RON `APlayerPassesFilter(Opponent, ControlsA(filter))` lowers to this condition.
Symbol `IsNonToken` in static condition filters uses `Not(IsToken)` and can
occur inside nonempty recursive `And`/`Or`. It reads token status, not the
card's name or characteristics (CR 111.6). Earlier type/color effects apply
before conditional bonuses; removing the source ability stops those bonuses.
Named collection recipients remain live and include either player's matching
creatures. Each source contributes a separate bonus.
`TestRONOpponentStatic*` verifies the complete RON records for Water Wurm,
Beasts of Bogardan and Ivory Guardians, protection legality/prevention,
transactional illegal-target rejection, tokens, multicolor noncreature
qualifiers, live source/qualifier control, phase cycles, source departure,
returned source identity, colors, ability removal, Blood Moon layer ordering,
independent factory instances and clones. No delayed binding, optional payment,
replacement ordering or simultaneous-operation contract is added.

## Static blocking prohibitions and clone queries

`SourceCanBlockOnly(Not(filter))` prohibits the source from blocking a matching
attacker. `CantBeBlockedExceptBy(subject, Not(filter))` prohibits a matching
creature from blocking the subject. These reuse the existing per-blocker and
per-attacker filter maps and `CanBlock`; every applicable restriction must pass
(CR 509.1b). The fixed power and color filters read final current characteristics
at the legality query. A later filter or characteristic change does not remove
a block that was legally declared (CR 509.1b, 509.1g–h). Creatures put onto the
battlefield blocking do not undergo declaration restrictions (CR 509.4b).

`Game.Clone` now copies `canBlockOnlyRules` and each filter slice, as it already
does for `cantBeBlockedExceptByRules`. Immediate clone queries preserve the
computed restrictions before the next layer pass. Resetting, rebuilding or
changing a clone's filter slice cannot change the original game. Filters remain
immutable declarations; no separate combat execution path is added.

Evidence: `TestCanBlockOnlyClone` tests immediate queries, filter slice isolation,
reset and reconstruction. The RON `TestRONBlockingProhibition*` suite tests
current power, color and subtype changes, cumulative flying/reach restrictions,
source ability removal and restoration, native block declarations, declared
block persistence, controller changes, phase cycles, source return, independent
factories and Aura attachment identity. Orcish Veteran's separate activation
also resolves and expires through the existing temporary keyword service.

## Attack permissions: haste only for attacking

`AttackAsThoughHaste(subject PermanentRef) ContinuousModification` permits the
subject to attack without continuous control since the turn began (CR 302.6,
702.10b, 609.4). It does not grant haste and does not permit tap or untap symbol
costs while the creature has summoning sickness (CR 702.10c). Tapping, defender,
phasing and other attack restrictions still apply. Combine it with
`GrantKeywordModification(subject, core.AttrCanAttack)` to permit attacks
through defender as well. Neither rule permission is an ability of the recipient.
Removing the recipient's abilities does not remove a permission from another
source; removing the source's static ability stops its permission.

The new `core.AttrAttackAsThoughHaste` is appended without changing existing attr
values. `IsKeywordAttr` and `IsAbilityAttr` exclude it. Current attack legality
reads it; `CanTapForEffect` retains its existing haste check. Existing layer,
attachment, source ability, clone and incarnation services supply its lifecycle.
A static rule follows the current attachment; an `Until` rule captures the current
incarnation. A returned permanent cannot inherit a captured permission. Changing
legality after declaration does not remove an attacker (CR 508.1).

Evidence: `TestAttackOnlyHastePermission`, `TestAttackOnlyHasteOtherRestrictions`,
and `TestRONAttackPermission*`. The latter covers real Aura casting, native
attacks and combat damage, payment and target rejection, source/host lifecycle,
and Instill Energy's complete untap activation. No replacement, simultaneous
action, LKI, or payment service is changed.

## RON temporary combat rule adapters

The temporary combat adapters reuse the established `Until`, typed reference,
blocking-filter and attack-permission contracts. They add no engine state or
rules API. The exact RON signatures and contexts are in the temporary combat
rules section of `docs/dsl-reference.md`.

Resolution captures the selected subject incarnation (CR 400.7, 611.2c).
Control changes and phase cycles retain that identity; departure and return
create a new object. A departed activation source cannot transfer a source
permission to its returned card. A departed Tower does not end a resolved
restriction on another creature. A phased or otherwise illegal target at
resolution receives no restriction (CR 608.2b). Cleanup expires the rule even
while the subject is phased out (CR 514.2, 702.26).

Blocking filters read current blocker characteristics and intersect with other
combat restrictions (CR 509.1b). A later restriction does not remove a block
already declared. Defender and attack-only haste permissions apply only to the
specified attack restriction (CR 609.4, 702.3, 702.10); summoning sickness, tap
costs and other independent restrictions keep their normal checks.

`TestRONTemporaryCombatRuleCards`, `Lifecycle`, `Legality`, `Copy`,
`AttackContexts`, `FilteredContexts`, and `NativeBlocking` establish paid native
casting/activation, effect timing, real combat damage, declared-block retention,
current subtype changes, independent targets, retargeted stack copies, clone
isolation, phase/zone/control changes, recipient ability removal and cleanup.
Wall of Wonder also exercises the shared P/T action before its source permission.
No replacement, simultaneous-operation, event-LKI or new payment contract is
introduced by this slice.

## RON static prevention with a permanent source filter

RON now lowers
`PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(filter, Permanent(ThisPermanent)), [PreventThatDamage])`
to the existing `PreventDamageFromTo(filter, IsID)` static service. The adapter
validates the event, recipient and operation list at compilation. Each factory
creates an independent static ability. No engine signature or replacement
algorithm changes.

The shared continuous cycle installs prevention only while the recipient has
this ability on the battlefield. Source filters use the captured source contract
above: current characteristics while present and retained incarnation data after
departure or phasing. The recipient is the current source of the static ability;
control changes do not remove it. A phase cycle restores the ability, and a
returned recipient has a fresh ability. Clones and separate factory instances
keep independent continuous state. Combat and noncombat packets use the same
prevention pipeline. Spell sources do not match this permanent-source event.
Finite shield allocation under CR 615.7 remains outside this adapter.

Evidence: `TestRONStaticPreventionCards`, `TestRONStaticPreventionLifecycle`,
`TestRONStaticPreventionPendingActivation`, `TestRONStaticPreventionCombatAndSpell`,
`TestRONStaticPreventionReplacementOrder`, `TestRONStaticPreventionStrictLoadAndInstances`
and `TestRONStaticPreventionInvalid`. Replacement choice tests show prevention
first cancels the packet, while redirection first moves it outside the recipient
filter. Rules: CR 113.6, 400.7, 609.7b–c, 615.1/615.2/615.6, 616.1/616.1f,
and 702.26b–d. Full-card tests also cover Pixies' blocking restriction and
Nomads' landwalk with the existing combat services.

## RON mixed-recipient damage execution

RON `MultipleRecipients([EachPermanent(filter), EachPlayer(AnyPlayer)])`, in
either list order, uses the existing `DamageRecipients` and `DealDamageEach`
services. The engine selects current recipients when the action resolves
(CR 608.2h). Flying changes, later entries, and returned creatures affect this
selection. Phased permanents are absent. Shroud does not exclude a recipient
because this instruction does not target. Source and amount use the existing
resolving-context contracts; copies keep announced X and their source identity.

One shared batch prepares replacement/prevention actions, applies damage
results, then sends damage notifications. Notification observers see the whole
batch (CR 120.4). Recipient replacement choices retain the shared recheck
contract (CR 616.1). Nonpositive amounts produce no damage event (CR 120.8).
Damage is not life loss: prevention, redirection, source lifelink, marked damage,
and damage notifications still apply. Lethal creatures remain present during
batch notifications; stack resolution then performs state-based actions
(CR 120.5, 704.3/704.5g). Death triggers resolve separately.

Permanent-source abilities keep their original incarnation and LKI after source
return or phasing (CR 113.7a, 400.7, 608.2h). A returned source can produce its
own entry trigger; it cannot replace the old ability's source or lifelink
beneficiary. Clones and copied abilities retain independent execution state.
The RON adapter introduces no new damage engine, allocation decision, saved
result, replacement priority, cost, or zone contract.

Evidence: `TestRONMixedDamageCards`, `TestRONMixedDamageResolutionMembership`,
`TestRONMixedDamageXCopiesAndRollback`, `TestRONMixedDamageReplacementOrder`,
`TestRONMixedDamageAbilitySourceIdentity`, and
`TestRONMixedDamageSimultaneousLethal`. Invalid declarations, strict reprint
loading, factory identity, and nested inventory roles have separate tests in
that file. Finite shields across simultaneous packets from multiple sources,
history filters, singular/overlapping recipient unions, and multiple damage
amounts remain separate capabilities.

## Simultaneous damage with separate amounts

`DamagePacketTo(DamageRecipientRef, ResolutionValue) DamagePacket` and
`DealDamagePackets(DamageSourceRef, ...DamagePacket) ResolvingEffect` define
one simultaneous instruction from one source. The action copies the packet
slice at construction. Definition inspection visits every recipient and value,
so undeclared targets and unbound values are rejected before gameplay. Empty
instructions and nil source, recipient or value declarations are invalid.

The action captures its damage source once, then resolves all recipient
identities and amounts before submitting any damage. Nonpositive packets have
no event (CR 120.8). Absent or phased recipients are omitted. Positive packets
for the same recipient are added before replacements, so one source deals one
combined amount to that recipient. Selection order is stable. Typed reference
resolution keeps permanent incarnation identity and existing target legality.
A copied ability uses its own resolving controller for `You`, while source
characteristics and lifelink retain the original source's controller or LKI.

The action submits the prepared assignments to `Game.DealSimultaneousDamage`.
Existing prevention, redirection, protection, lifelink and damage results apply
through that service (CR 120.3–4, 616.1). Damage callbacks see all results;
state-based lethal removal follows resolution. A sole illegal target prevents
the entire spell or ability from resolving, including nontargeted packets
(CR 608.2b). Two successive resolutions remain two instructions: if a copy kills
a self recipient, the original cannot deal its self packet to the departed
incarnation, but can still deal its legal targeted packet.

Evidence: `TestDamagePacketsSimultaneousAndMerged`,
`TestDamagePacketsDefinitionValidation`, `TestDamagePacketsValuesBeforeDamage`,
and `TestRONDamagePackets*`. These cover different amounts, overlapping
recipients, construction isolation, reads before damage, declaration errors,
real RON spell/activation gameplay, placement-time trigger targets, copies,
clone isolation, rollback, phasing, source return/LKI, controller changes,
planeswalker results, lethal self damage and affected-player replacement order.
This contract adds neither finite multi-source prevention allocation nor new
recipient collections, event bindings or announcement choices.

## Draw-step detection and source tapped-state information

CR 603.2b and 603.4 require beginning-of-draw-step trigger detection and its
intervening-if check before the normal draw changes the game state. CR 504.1
requires the normal draw before priority. The engine now dispatches `EvtDrawStep`
first, performs the normal draw through replacements, then places all pending
triggers on the stack in the existing APNAP path. No priority or trigger
resolution occurs between detection and the normal draw. Skipped draw steps
still produce neither the event nor the draw.

`SourceTappedStateWithLKI(tapped bool) StateCondition` implements a pure source
information check. Without a captured incarnation it requires a live phased-in
source. Resolution uses the captured incarnation and reads its current state,
or the ZoneSystem departure snapshot under CR 608.2h. Another incarnation of
the same card cannot supply the result (CR 400.7). The existing retained
incarnation snapshot survives transient LKI cleanup and later departures.
Phasing after detection permits a tapped-state information read from the retained
object, with no action on that phased object (CR 702.26b/d). Phasing before
an event prevents detection. The condition requires a permanent source in
engine definition validation; it does not change live-only conditions used by
static membership rules. It stores no game state and clones share only the
immutable declaration.

`TestSourceTappedStateWithLKILifecycle` covers both states, live changes,
departure/return, a second departure, phasing, unknown identity, transient LKI
cleanup and clones. `TestSourceTappedStateWithLKITrigger` covers detection,
resolution rechecks, departed states, returned objects, copies, clones and
invalid spell context. `TestDrawStepDetectionBeforeNormalDraw` and
`TestDrawStepPriorityAfterNormalDraw` establish the native timing contract.
`TestRONDrawStepBeforeNormalDrawReplacement` proves both directions with a
real Howling Mine record and a normal-draw replacement. The other
`TestRONDrawStep*` tests cover all Howling Mine and Mana Vault rules, including
optional upkeep payment, mana classification, failed payment rollback, damage
prevention, event identity and current controller. Existing Fasting tests cover
whole-step skipping. No new simultaneous-operation or replacement-order service
is introduced.

## RON graveyard targets and returns

RON `TargetGraveyardCard(filter)` declares one `CardTargetBinding` per card
factory instance. `IsCardtype(type)` and nonempty nested `And([filters])` read
card characteristics. `InAPlayersGraveyard(SinglePlayer(You))` restricts the
zone to the announcing controller's graveyard. Without that restriction, or
with `InAPlayersGraveyard(AnyPlayer)`, both graveyards are available. A
conjunction that includes the controller restriction retains that restriction.
Other player filters and disjunctions remain unsupported.

`Ref_TargetGraveyardCard` is a card reference within its target body. It cannot
be read as a permanent, player, or spell, or escape to a different mode or
ability. The adapter uses `DeclareGraveyardCardTarget`, `GraveyardCardFilter`,
`InAnyGraveyard`, and the existing target declaration and resolution services.
Targets are chosen before payment and checked again at resolution. Graveyard
visit identity prevents a selected card from following a leave-and-return cycle
(CR 400.7, 601.2c, 602.2b, 608.2b).

`PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)` uses `ReturnCardToHand` and
puts the card into its owner's hand at resolution. The battlefield form is
`PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard, [EntersNormally])`.
This slice requires a target filter that includes `IsCardtype(Creature)`.
Other card types and entry modifiers remain unsupported in this context.
The adapter uses `ReturnCardToBattlefield(card, You)` for normal entry under
the resolving controller. Entry effects and ETB triggers use the shared engine
path. A later instruction cannot use the old graveyard reference after the
first instruction moves the card (CR 400.7, 608.2c). Source departure, phasing,
or a control change does not change the controller of a pending ability.
Copies keep independent targets and recheck legality for their controller.

Evidence: `TestRONGraveyardReturnCards`, `AnnouncementRollback`,
`CopiesAndInstances`, `Invalid`, `ReprintsAndInventory`, `SharedContexts`,
`NormalEntry`, `BattlefieldRequiresCreatureTarget`,
`ConjunctionAndCopyLegality`, and `Caretaker` in
`pkg/mage/dsl/ron/graveyard_return_test.go` (all names have the
`TestRONGraveyardReturn` prefix). All eight complete card names and 24 reprint
records compile and validate. Gameplay evidence covers seven return cards and
Hell's Caretaker's upkeep, tap, sacrifice, and target contracts. No engine
execution rule, simultaneous batch, replacement order, or LKI service changes.

## Permanent information through typed references

`ControllerOfWithLKI(PermanentRef) PlayerRef`,
`PowerOfWithLKI(PermanentRef) ResolutionValue`, and
`ManaValueOfWithLKI(PermanentRef) ResolutionValue` read information about an
exact battlefield incarnation (CR 400.7, 608.2h). Supported subjects are `This`,
`AttachedPermanent`, `From(PermanentBinding)`, and a singular
`PermanentTargetBinding`. Definitions reject nil or unsupported subjects, and
retain target and snapshot dominance checks.

While phased in, a matching incarnation supplies its current controller,
layered power, or mana value. After departure, the incarnation-indexed retained
snapshot supplies controller and mana value; the saved final power supplies
power. A returned card and its later departure cannot overwrite the old
information. Unknown identities supply no player and zero for numeric reads.
An uncaptured `This` can read only the live source. A departed source's
attachment retains its last host incarnation.

Before direct and indirect phase-out, `Game.PhaseOut` saves each member's
information with the existing `PermanentLKI` snapshot builder. `ZoneSystem`
owns a separate incarnation-indexed phase-information map; this map does not
publish a departure to existing LKI readers. These explicit information queries
use the saved phase information while the object is absent (CR 702.26b/d/f/g).
Phase-in discards it and restores live reads. Phase snapshots and group status
remain isolated in clones. Indirect phase-in obtains mutable attachment pointers
before changing their status. Phasing creates no zone-change events.

These queries never make a departed or phased object available for an action.
Existing `ControllerOf`, `PowerOf`, `ManaValueOf`, mutation references, subtype
conditions, and typed damage-source queries retain their contracts. No new
payment, replacement order, event scheduling or simultaneous zone operation is
introduced. Evidence: `TestPermanentInformationLifecycle`,
`TestPermanentInformationDeclaration`, `TestPermanentInformationPhaseSnapshotAndClone`,
`TestPermanentInformationAttachmentDepartureAndUnboundSource`,
`TestPermanentInformationCloneAfterPhaseIn`, and the existing
`TestDamageSourceOfAbsentPhasedAndDeclaration` and
`TestPermanentSubtypeBranchEntryAndDeparture` regressions.

The RON compiler lowers `ControllerOfPermanent(ref)`, `PowerOfPermanent(ref)`,
and `ManaValueOfPermanent(ref)` to these services. References remain local to
their source, Aura, target, mode or captured permanent event. Life actions with
an explicit permanent-controller player use `GainLifeFor` / `LoseLifeFor`.
`Exile([Permanent(ref)], [])` lowers one live subject to `Exile`; other subject
lists and modifiers remain rejected. The declaration works in spell, modal,
activation and ordinary trigger action bodies. It does not establish graveyard
exile, collections, links, exile results, or replacement-context operations.

Evidence: `TestRONPermanentInformationSpells`, `Venom`, `VenomCasting`,
`DivineOffering`, `CopiesAndRollback`, `Contexts`, `Invalid`, `ReplacementOrder`,
and `ReprintsAndInventory` (all names have the `TestRONPermanentInformation`
prefix). Real Swords to Plowshares and Crumble read information after the zone
action; failed destruction still permits the life instruction. Divine Offering
permits regeneration and gains life for the resolving controller. Psychic Venom
reads the captured land's current or last controller when the trigger resolves,
not a new attachment's controller or the land's controller at event time.

## RON creature-death filters

`TriggerA(WhenAPermanentDies(IsCardtype(Creature)), actions)` uses
`CreatureDies()`. The exact `And([IsCardtype(Creature),
WasDealtDamageByPermanentThisTurn(ThisPermanent)])` filter uses
`CreatureDies().DamagedByThisTurn()`. Either order of the two conditions is valid.
These adapters work in ordinary permanent triggered abilities. Other type,
source, composition, delayed-trigger and replacement forms remain rejected.
These events do not declare `Trigger_ThatPermanent` or `Trigger_ThatPlayer`.

The existing engine event and damage-history contracts apply. Death means
battlefield to graveyard (CR 700.4); creature status is checked before departure
(CR 603.10a), including animated artifacts and tokens. Phasing, exile, bounce and
regeneration are not death. The damage condition checks actual damage this turn
from the source incarnation to the dying incarnation. Prevention, cleanup and
leave-and-return can prevent a match; a phase cycle retains history. Each death
in a simultaneous batch can trigger separately, including when the source dies
in that batch (CR 603.2c). A pending source counter action cannot affect a returned
source or a currently phased source. Control changes do not change that identity.

Soul Net payments occur at resolution and use the trigger controller. Decline or
failed affordability consumes no mana and gives no life. A copied trigger uses
its copy controller for this payment. Life gain still uses the replacement
pipeline and the affected player's ordering choice. Search clones make their own
optional decisions through the existing `SearchPlayer` policy; cloning does not
copy an interactive player's future decisions.

Evidence: `TestRONCreatureDeathSengir`, `SoulNet`, `Declarations`,
`ReprintsAndCasting`, `InventoryAndRegistry`, `FilterOrder`, and
`LifeReplacementOrder`. These tests execute complete RON records for Sengir
Vampire and Soul Net, including all six reprints, cost/timing rollback,
incarnation history, simultaneous deaths, copied triggers and clone isolation.
The existing `TestCreatureDiesDamagedByThisTurn*` engine tests retain their
contracts. No engine execution API changed in this slice.

## RON fixed entry counters

RON lowers a single `EntersWithNumberCounters(Integer(n), counter)` entry
operation on `ThisPermanent` to the existing `EntersWithNCounters` ability.
`prepareBattlefieldEntry` routes placement through `AddCountersWithReplacement`
before entry observers run (CR 122.6, 603.6a, 614.1c). The entering controller
chooses between applicable counter-placement replacements (CR 616.1); the
adapter adds no replacement service. Phasing retains counters without repeating
entry (CR 702.26d). A zone return starts a new incarnation with its declared
entry counters (CR 400.7, 122.2).

Evidence: `TestRONEntryCountersTriskelion` executes the complete ATQ card:
casting/payment rollback, 4/4 entry, one-counter activation cost, any-target
player/creature/planeswalker/self damage, controller legality and exhausted costs.
`TestRONEntryCountersLifecycle` covers source departure, stack/card copies,
cloned pending abilities, phase cycles, return, independent instances and source
death before damage resolution. `TestRONEntryCountersTargetRecheckAndPrevention`
covers target return, shroud, phasing and prevention after payment.
`TestRONEntryCountersReplacements` verifies add-then-double (8) and
double-then-add (7) on real Triskelion entry, with the entering controller as
chooser. `TestRONEntryCountersObserverAndFactoryIsolation` verifies entry
observation, zero placement and fresh factory ability declarations.

This establishes the fixed-entry compiler adapter, with no engine execution API
change. It introduces no event or saved-result bindings and no batch operation.
General entry choices, X, copy-on-entry replacement ordering and token batches
remain outside this slice. The engine stores counter totals in `uint8`;
declarations above 255 are rejected here. Unbounded counter totals, including
replacement-expanded totals beyond that storage, remain an engine limitation;
the observed collection counts (2–7) and tested replacement totals (7–8) fit.

## RON player-target life lowering

The RON life adapter supplies the existing typed `PlayerRef` to `GainLifeFor`
and `LoseLifeFor`. It does not change engine execution. The player target keeps
its declaration identity in separate factories, stack copies and game clones.
The chosen player and announced X are read at resolution. Activation source
control changes, departure, return and phasing do not redirect the recipient.
Each explicit `PlayerAction` has a local scope; a sibling `You` read uses the
resolving action controller. Existing spell-controller references also supply
their explicit player instead of the action controller.

CR 119.3 and 119.9–10 govern life changes and zero gain; CR 608.2b governs target
rechecks; CR 608.2c governs instruction order. The shared life action applies
normal life-gain replacement services. Damage shields do not prevent life loss.
`TestRONPlayerLifeStreamOfLife`, `ScopesAndCopies`, `TargetRecheckAndReplacement`,
`ModeLocalTargets`, `SpellController` and `InvalidDeclarations` in the RON package
provide evidence. `RealModalClauses` covers the life clauses of Healing Salve
and Alabaster Potion only. No new replacement-order or simultaneous-operation
contract is established. Spell-cast event life references remain unsupported.
