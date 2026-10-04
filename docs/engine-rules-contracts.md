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
Spell sources, the separate color/type prevention implementations, protection,
deathtouch, source notifications, recipient incarnation, batch damage, and typed
adapters remain separate work. No payment or action transaction changed.

Evidence: `TestDamageSourceFilterIncarnation`, `TestDamageSourceFilterCopyClone`,
`TestDamageSourceFilterRedirect`, `TestDamageSourceFilterMissingSource`,
`TestDamageSourceFilterReplacementOrder`, `TestDamageSourceFilterSimultaneousDeath`,
and synthetic `TestDamageSourceFilterSequence`. Rules: CR 113.7a, 608.2h,
609.7b–c, 615.6/615.9, 616.1/616.1f, and 702.26b–d.

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
