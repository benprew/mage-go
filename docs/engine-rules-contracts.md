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
