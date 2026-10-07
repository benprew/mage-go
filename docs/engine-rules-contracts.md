# Engine rules contracts

## Live restrictions on spell targeting

`CantBeTargetedBySpells(PermanentRef, CardFilter)` is a static rule evaluated
against the current subject at each target query. It uses the existing active
continuous-effect lifecycle and printed-ability suppression. A source must be
present, unphased and retain its ability. A returned source supplies only its new
rule. An Aura host is resolved through the current attachment incarnation;
moving the Aura moves the restriction, and a returned host cannot inherit the
old attachment. Removing the host's abilities does not remove the Aura's rule.
Control changes do not limit either player's spell targeting prohibition.
Clones copy the effect source and read their own source, host and suppression
state without a layer pass being required for the query itself.

`SpellTargetSource(Card) Card` creates an immutable spell view for pre-stack
queries. The existing retained source view records spell identity at resolution,
even when the object has been popped or is a copy. Ability views remain abilities
after departure, return or phasing of their Aura source. No inference from source
card types or zones replaces this distinction (CR 115.1b–d, 707.10). Existing
shroud, hexproof, protection, target bounds and incarnation checks still apply.
Spell targeting restrictions do not prevent nontargeted attachment (CR 115.1b).

Casting and target selection reject matching spells before payment (CR 601.2c).
Resolution checks the current restriction and current spell information again
(CR 608.2b, 608.3b). A color filter uses the retained spell color override, even
after stack removal. Copies use their own characteristics and controller;
ordinary color changes are not copiable values (CR 707.2). All-illegal targets
prevent resolution; other legal targets remain independent. Copy retargeting
offers only legal new candidates (CR 707.10c). The service creates no events,
replacement rules, payment results or simultaneous-operation ordering.

`TestCantBeTargetedBySpells*` covers declaration rejection, source phasing and
return, source ability removal/restoration, both controllers, payment rollback,
original and copied spells, copy retargeting, partial legality, color changes,
ability-source departure/LKI, nontargeted attachment and clone isolation.
`TestRONSpellTargetRestriction*` covers complete Bartel Runeaxe and Tetsuo
Umezawa casting and gameplay, shared host lifecycle and filter syntax, invalid
RON, strict complete-record test loading and atomic invalid-collection rejection.
Bartel's native attack verifies vigilance. Tetsuo's tapped target destruction uses
the existing transactional activation and destruction pipeline. Existing engine
and RON destruction/target tests retain the broader regeneration, blocking,
source-departure and target-legality contracts.

## Captured next-upkeep continuous duration

`UntilNextUpkeep(PlayerRef, ...ContinuousModification)` establishes an explicit
player lifetime policy for resolving continuous modifications. It saves the
selected player and each modification's subjects/values at resolution. The
existing layer service installs all layers together and retains incarnation
identity (CR 611.2a, 611.2c–d). Player references participate in definition
validation; missing players start no duration. Nil/empty modifications, unbound
player or permanent references, and unsupported control durations are rejected.

Each installed composed layer effect holds the saved player UUID. Both source
removal and layer pruning honor its existing `survivesSourceExit` contract.
Source departure, return, control changes, ability removal, and phasing cannot
redirect expiry. Recipient departure makes the captured modification inapplicable;
phasing suspends application without changing the endpoint. Clones copy the
saved player with each installed effect; expiry in one game changes no other game.

At the start of an actual upkeep, `RemoveUntilYourNextTurn` removes composed
effects for the saved player. Layers are applied again before `EvtUpkeep`, so
upkeep conditions see expired characteristics (CR 500.4 and 500.6). There is no
delayed trigger, priority window, or replacement operation for expiry. Cleanup,
untap, and another player's upkeep do not expire this duration. Skipped upkeeps
do not execute this hook; extra upkeeps do, even within the same turn.

Existing noncomposed `UntilYourNextTurn` callers retain their source-controller
policy. `Until(UntilYourNextTurn, ...)` remains invalid because it has no explicit
player policy. Existing transactional costs, targeting, trigger ownership
(CR 109.5), copies, and layer ordering supply their usual contracts. This slice
adds no LKI read, action result, simultaneous operation, or replacement ordering.

`TestUntilNextUpkeep*` covers identity, validation, independent players, explicit
player selection, absent players, cloning, expiry before intervening-if event
detection, cleanup/untap timing, phasing, and skipped/extra upkeeps.
`TestRONNextUpkeep*` maps the full Erhnam Djinn record to casting/payment rollback,
target legality, forestwalk queries, native upkeep execution, source departure,
control changes, phased/returned targets, copies, and cloned pending effects.
Xenic Poltergeist's modifications remain unsupported. Halfdane's `ManualPT`
remains unsupported and its RON duration needs a separate source correction:
`data/LEG.json` specifies the end of the next upkeep, unlike its current RON.

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

## RON optional sacrifice payment adapter

RON optional `SacrificeAPermanent(filter)` payments use the same transaction
with `SacrificeMatchingIncludingSourceCost`. The current payer controls the
chosen permanent; its owner receives the card (CR 701.21a). An unavailable cost
offers no choice. Decline or an invalid/stale selection runs the unpaid body
without consuming a payment permanent. Regeneration does not replace sacrifice.
No payment result is read from global resolution state. Adjacent `If`/`Unless`
branches support both `CostWasPaid` and `CostWasntPaid` (CR 118.10–12a).

The trigger's stored controller remains the payer and the damage recipient
after source control changes. Copies use their own controller. Unpaid actions
use the original source incarnation. A returned source is not tapped or
sacrificed by the old trigger; independent damage still uses source LKI
(CR 400.7, 608.2c/h). Phasing prevents detection but does not cancel a pending
payment. Phased payment candidates are unavailable (CR 702.26b).
Elder Spawn's sacrifice and damage are separate ordered instructions.
This adapter adds no engine API or replacement-order contract.
Evidence: `TestRONOptionalSacrificeCards`, `BranchesAndScope`,
`InstancesCopiesAndClones`, `ExplicitPlayerAndNestedResults`,
`DetectionAndCurrentFilters`, `OtherCardRules`, `Inventory` and `NativeLoad`
under the same `TestRONOptionalSacrifice` prefix.

## All-player hand and graveyard library shuffles

`ShuffleHandsAndGraveyardsIntoLibraries(PlayerSet)` selects current hands and
graveyards at resolution. All selected zone moves finish before events, and
all shuffles finish before a later instruction. It sends per-card zone events
and one graveyard-departure batch per affected player. Hand movement is not
discard. The player zone services preserve ownership and clone isolation.
Empty sets of cards still cause a library shuffle (CR 701.24c-e).

`ShuffleHandsIntoLibrariesAndDraw(PlayerSet)` shuffles hands only, then draws
the number each player moved. Its count map belongs to one invocation and
cannot be changed by another instance, copy, clone, or later hand change.
Draws occur individually through the existing replacement pipeline, with the
active player completing all draws first (CR 121.2, 608.2c/e/f). The typed
player collection is checked by definition validation. Nil collections fail
before execution. These APIs create no general saved-count binding.

The retained `ShuffleHandAndGraveyardIntoLibraryAndDraw(n)` constructor uses
the same shuffle service followed by `DrawEach`. This fixes its former early
draw and seating-order behavior while preserving its public signature.
Source departure, return, phasing, or control change does not alter an
all-player instruction already on the stack. The resolving spell remains on
the stack until its instructions finish (CR 608.2n).

Evidence: engine `TestLibraryRecycleBatch`, `EmptyAndDeclaration`,
`LegacyOrder`; RON `TestRONLibraryRecycleCards`, `Declarations`,
`ReplacementsAndEmpty`, `CountsBeforeDraws`, and `Contexts`.

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

### RON additional sacrifice casting costs

One `CastEffect(AdditionalCastingCost(SacrificeAPermanent(filter)))` on a
nonmodal instant or sorcery uses `WithAdditionalCost` and the existing matching
sacrifice costs. The caster must control the payment permanent; ownership does
not grant payment permission. The cost moves it to its owner's graveyard.
Indestructible and regeneration do not prevent sacrifice (CR 701.21a).
Targets and casting timing are checked before payment. Printed mana and the
additional cost use one transaction; an unpaid or invalid cast keeps the card,
mana and permanent (CR 601.2c, e–h). Costs remain paid if the spell is countered
or its target becomes illegal.

Saved toughness or mana value uses the same typed number bindings as activation
payments. Each stack object keeps its own captured number. Spell copies keep
that number and do not pay again (CR 707.10). Resolution uses the copy's
controller. Mana value uses the cost before sacrifice, with battlefield X equal
to zero; counters do not increase it (CR 202.3, 608.2h). Zero adds no mana.
This adapter does not change engine payment, damage, replacement or batch rules.

Evidence: `TestRONAdditionalSacrificeRealCards`, `Rollback`, `PendingValues`,
`GrenadeTargets`, `ZeroAndRecast`, `FactoriesAndQueries`, `Declarations`, and
`StrictLoadAndInventory` in `pkg/mage/dsl/ron/additional_sacrifice_test.go`.
The complete Goblin Grenade and Sacrifice records supply the real-card cases.
Other additional costs and sacrifice activation mana remain separate contexts.

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
[DoesntUntapDuringControllersUntap])` uses the source-bound static ability
service and `PreventUntapDuringStep`. The turn service checks this rule
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
changed. The later typed subject contract below adds Aura-host and conditional
forms. Temporary forms remain open.

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
on an Aura. Host-death events, intervening-if host conditions, and
host-controller end-step events remain separate compiler work. Resolving
`HostController` uses the separate contract below.

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

## RON player-or-planeswalker target adapter

RON `TargetPlayerOrPermanent(AnyPlayer, IsCardtype(Planeswalker))` uses the
existing `DeclarePlayerOrPlaneswalkerTarget` declaration. Its typed recipient
read keeps the declaration and permanent incarnation. Target choice occurs at
announcement or trigger placement (CR 601.2c, 603.3d). Resolution checks current
type, shroud, phasing and incarnation (CR 400.7, 608.2b). An illegal sole target
prevents the complete ability from resolving, including an independent life
gain. A legal target with prevented damage still permits the life gain.
Damage to a planeswalker removes loyalty (CR 306.8). The existing damage and
replacement pipeline handles amounts, prevention and affected-player order.

`TestRONPlayerPlaneswalkerAxelrod` covers the complete LEG record: source-specific
damage history, actual prevented damage, turn cleanup, simultaneous deaths with
the source, two pending triggers, target choice, independent life gain, source
departure/return/control/phasing, copied controller, clones and native trample
combat. `AxelrodCasting` checks the creature's full mana payment and metadata.
`WitchHunter` covers the complete DRK damage ability, target exclusions, tap and
sickness checks, player/planeswalker damage, source information, copies and
clones. `WitchHunterBounceAndCast` covers casting and the separate paid bounce,
current opponent control, return to the owner, rollback and target rechecks.
`TargetIdentity`, `SpellModesInstancesAndRetarget`, `PacketsAndReplacementOrder`,
`Declarations` and `StrictLoadAndInventory` cover the shared target contracts.
No engine rule, replacement service or simultaneous-operation service is added.

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

## Player target specification checks at resolution

`Game.isTargetStillLegal` checks an existing player against the target
specification's `Possible` query before resolution (CR 102.2, 109.5, 608.2b).
The query uses the stack object's controller. If an opponent-controlled copy
keeps that player as its target, an opponent-only specification makes the
player illegal. When all targets are illegal, no instruction runs, including
an independent draw. Targets with no specification retain their existing
player-existence check. Typed and legacy player target declarations use the
same check. No target API changes.

`TestPlayerTargetResolutionChecksSpecification` covers typed and legacy
specifications, any-player and opponent-only targets, both controllers,
all-illegal resolution, and clone isolation. The earlier drain-copy test had
opposing life instructions that could cancel and hide incorrect resolution.
`TestRONOpponentPlayerTargets` adds separate life-loss evidence through native
RON spells, activations, entry triggers, and selected modes.

RON cast-event `Opponent` lowering uses the existing `SpellCast().ByOpponent()`
event-time check (CR 603.2, 603.3a). `TestRONOpponentCitanulDruid` covers artifact
casts, failed casts, spell copies, source control changes, departure, returned
incarnations, phasing, independent pending events, copied triggers, and clones.
The filter is not an intervening-if condition and is not checked again at
resolution. Counter actions still require the captured source incarnation.

## RON optional tap/untap adapter

`MayAction(TapOrUntapPermanent(ref))` reuses `MayTapOrUntap`; no engine behavior
changes. The resolving controller makes the optional decision at resolution
(CR 608.2d). Only untapped permanents can be tapped and only tapped permanents
can be untapped (CR 701.26a–b), so the current state determines the offered
operation. Decline affects only this instruction. Shared target checks suppress
all instructions when every target is illegal (CR 608.2b).

The adapter uses typed instance references and shared tap/untap operations.
Departure and return invalidate an old incarnation; a phase cycle preserves
it. Source departure or later source control does not transfer an activation's
decision. Copies use their own controller and target identity. Game clones
retain independent targets and state; their SearchPlayer defaults can decline
independently of the live player. Untap replacement with a stun counter removes
the counter and keeps the permanent tapped; a declined untap leaves it intact.

`TestRONTapChoiceRealCards`, `TargetFiltersAndRollback`, `CopiesAndInstances`,
`Contexts`, `SequenceAndEvents`, `Invalid`, and `Inventory` in `dsl/ron` cover
this adapter. Existing `TestComposableTapChoice*` tests retain the shared engine
contracts. This slice adds no LKI read, finite prevention, replacement-ordering,
or simultaneous-operation contract.

## RON resolution-time battlefield counts

`TestRONBattlefieldCount*` establishes RON lowering to the existing
`CountPermanents`, `MatchingPermanents`, `PermanentsControlledBy`, and
`MultiplyValues` services. No engine rule or public API changes.

Counts read current battlefield membership and final characteristics when an
instruction resolves (CR 608.2c, 608.2h). Later instructions recount after prior
zone actions. Control and ownership remain separate; phased members are absent
(CR 702.26b/d). A returned permanent is a new object but can join a current
collection (CR 400.7). This count is not a historical snapshot.

Karma retains its upkeep event player and pending controller through source
control changes, departure and phasing (CR 603.2b, 603.3a). Copies preserve the
event player. Jovial Evil checks opponent target legality at announcement and
resolution (CR 608.2b); a copy can select its own opponent without changing the
original target binding. Native damage retains prevention and source LKI,
including lifelink. Counts in a cloned game use that game's members. Definition
validation still detects undeclared targets nested inside the count.

The tests cover full RON records, reprints, paid casting, rollback, current color
and type filters, tokens, empty counts, signed multiplication, native upkeep,
source and member lifecycle, copies, clones, declaration errors, and action
sequencing. All-zone CDA, historical counts and recipient-local damage counts
remain separate contracts. No new replacement ordering or simultaneous-operation
contract is established by this adapter.

## RON land mana event and bonus contracts

CR 106.12/106.12a and 605.1b/605.4a distinguish tapping for mana from an ordinary
tap. The new RON adapter uses `LandTappedForMana` for ordinary triggered actions.
The engine saves the activating player and land incarnation for each event.
Manabarbs damage waits for trigger placement and resolution. A copied trigger
keeps the original event actor even when its controller changes. Controller
information about the saved land uses current information, departure LKI or
phase information through `ControllerOfWithLKI`; it cannot follow a returned
incarnation. No new LKI or phase-information rule is introduced.

The exact single-instruction RON mana bonuses use `NewManaFlareAbility` and
`NewAttachedManaBonusAbility`. Their shared production and payment-planning
service adds one mana immediately to the activating player, regardless of the
bonus source's controller. A produced-type choice uses only the original
ability's produced types, including colorless. Multiple bonus instances each
add one mana. Bonus mana does not cause another tapped-for-mana event.

`ManaSystem.ManaBonuses` now excludes phased sources and checks both card ID and
incarnation for attached bonuses. An Aura cannot supply a bonus to a returned
host through a stale attachment. Removal of source abilities removes its bonus.
Clones retain independent phasing, attachments, pools and payment state. Existing
transactional casting keeps failed payment from leaking taps, mana or triggers.
There is no new API or replacement-ordering contract.

Evidence: `TestManaBonusLifecycle` and `TestRONLandMana*`. The RON tests cover
native casts for all 18 reprints of Mana Flare, Manabarbs and Wild Growth,
automatic payment and rollback, Aura legality, source/host lifecycle, fixed and
produced-type choices, copies, independent instances/events, native prevention
and lifelink, typed LKI/phase reads, invalid scopes and nested inventory roles.
Subtype events and temporary triggers remain compiler gaps.

### RON enchanted filter adapter

RON `IsEnchanted` now uses the existing `mage.IsEnchanted` permanent query.
CR 303.4b defines the enchanted object by its attached Aura. The adapter adds
no engine behavior. Existing target services apply CR 608.2b at announcement
and resolution, including attachment departure, Aura phasing, target phasing,
and target incarnation changes. Equipment does not satisfy the filter.

`TestRONEnchantedFilterRamses` and `LegalityRollback` verify real RON activation,
source departure, control changes, copied abilities, regeneration and clone
isolation. `TestRONEnchantedFilterWall*` verify static prevention through the
shared damage-source/LKI query and replacement pipeline (CR 608.2h, 615.1),
including repeated and simultaneous damage, Aura departure/phasing, source
return, ability removal, combat damage, protection from white and spell damage.
`TestRONEnchantedFilterReplacementOrder` verifies affected-player ordering and
recipient rechecks. No finite prevention allocation contract is added.

## RON temporary base power

RON `SetPower(Integer(n))` in `CreatePermanentLayerEffectUntil/3` lowers to
`SetBasePower(PermanentRef, ResolutionValue) ContinuousModification`. This uses
the existing layer 7b service (CR 613.4b); base toughness stays unchanged, and
layer 7c bonuses and counters still apply. The compiler creates a typed subject
for each factory instance. Existing spell, activation and trigger action lists
and temporary durations apply. Dynamic and static values remain unsupported.

Island of Wak-Wak and Singing Tree use the existing tap cost and target
services. Flying or attacking status is checked again at resolution (CR 608.2b).
Phasing and controller changes remove a creature from combat (CR 506.4), so
Singing Tree cannot resolve on that creature after either operation. Once the
effect starts, later combat removal, control changes or a phase cycle do not
end it (CR 611.2c). Returned objects have new incarnations. Source departure
does not end the effect; cleanup and cloned games use the existing independent
lifecycle. No new LKI, payment, replacement or batch-operation contract is
introduced. Evidence: `TestRONTemporaryPowerDeclarations`,
`TestRONTemporaryPowerRealCards`, `TestRONTemporaryPowerLegality`,
`TestRONTemporaryPowerStrictLoadAndNativeGame`,
`TestRONTemporaryPowerSharedContexts` and `TestRONTemporaryPowerSpellAndInventory`.

## RON land and artifact graveyard events

The RON `WhenAPermanentIsPutIntoAPlayersGraveyard` adapter uses the existing
`PermanentDies` event service. `OfType` checks the type before departure;
`YouControlled` checks the last controller against the trigger controller.
`WithOwner` saves the destination graveyard owner and `WithController` saves
the last battlefield controller in separate typed bindings. The compiler
exposes these as `Trigger_ThatPlayer` and `ControllerOfDeadPermanent` only
inside the matching trigger body. Each factory and pending event keeps its
own bindings. Returned objects and later control changes do not redirect them.

The supported RON context is `TriggerA` with land/artifact type filters and an
optional controller restriction, and `AnyPlayer` as the destination filter.
The existing engine detects each occurrence in simultaneous deaths, including
self-death, and uses its pre-batch observer when the source dies too. Tokens
can trigger without leaving a card in the graveyard. APNAP placement, trigger
copies, cloned games, source LKI for damage, prevention and optional mana
payment remain in their existing services. No engine implementation changed.

Rules: CR 400.3, 400.7, 603.2c/g, 603.3a/b, 603.6c, 603.10a and 700.4.
Evidence: `TestRONGraveyardEventDingusEgg`, `TestRONGraveyardEventTablet`,
`TestRONGraveyardEventBatchInstancesAndCopy`,
`TestRONGraveyardEventDeathKindsAndLKI`,
`TestRONGraveyardEventOwnerControllerAndAPNAP`,
`TestRONGraveyardEventNonEvents`, `TestRONGraveyardEventDeclarations`,
`TestRONGraveyardEventInventoryAndScopes` and
`TestRONGraveyardEventNativeLoad`. The local Oracle data requires Dingus Egg
to damage the last controller, not the graveyard owner; all six RON reprints
now use the separate controller binding.

## RON end-step intervening-if adapters

End-step `TriggerI` uses `TurnStepBegins`, `EachEndStep`/`YourEndStep`,
`BindPlayer`, `WhenEvent` and `WithInterveningIf`. Battlefield absence lowers
to `BattlefieldHasNone`; source damage-to-opponent history lowers to
`SourceDamagedOpponentThisTurnCondition`. No engine service changes.
CR 603.4 and 608.2a require checks at detection and resolution. CR 603.3b
supplies APNAP placement. Current battlefield queries exclude phased objects
(CR 702.26b). Captured source incarnations prevent a returned card from
receiving counters or paying the original trigger's sacrifice (CR 400.7).
A phase cycle retains damage history; a new incarnation does not. The pending
trigger retains its controller, so source control changes cannot change the
opponent used by its condition. Sacrifice can fail after a source control
change because the trigger controller no longer controls it (CR 701.21a).

`TestRONEndStepInterveningPestilence` proves both condition checks, phasing,
source return, sacrifice permission, copies and clone isolation.
`TestRONEndStepInterveningPestilenceDamage` proves native paid activation,
failed payment, source departure/control/phasing, black protection, finite
player prevention and simultaneous creature/player damage visibility.
`TestRONEndStepInterveningDervish` proves actual damage history, cleanup,
source identity, control changes, copies and phasing.
`TestRONEndStepInterveningDervishProtectionAndInstances` proves black damage,
blocking, spell and Aura protection, plus per-instance history. It proves one
counter per end step regardless of
the number of damage events. `TestRONEndStepInterveningAPNAPAndAbilityRemoval`
proves separate factories, APNAP and retained pending triggers.
`TestRONEndStepInterveningDeclarations`, `TestRONEndStepInterveningInventory`
and `TestRONEndStepInterveningNativeLoad` prove declaration errors, event scope,
semantic roles, strict reprint loading, unpaid casting rejection and native
execution. These adapters establish no new LKI, payment, damage, replacement
or history service.

## Artifact tap and non-tap activation events in RON

`ArtifactAbilityActivatedWithoutTapCost` matches the activation source's exact
`SourceIncarnation`. It reads current characteristics for that incarnation or
its departure LKI. A returned physical card cannot change the match or capture.
`WithArtifact` saves that incarnation even when the source was sacrificed during
payment. `WithController` saves the controller of the same incarnation at
detection. A later `ControllerOfWithLKI(From(artifact))` reads the current
controller or departure information at resolution; it is distinct from the
saved event controller. Sources without an event incarnation do not match.

RON `TriggerA` lowers artifact tap events, optional opponent control filters,
and artifact non-tap activation events through the existing event patterns.
Each rule and each factory gets a separate permanent binding. Attached-host
activation filters require an Aura scope. Unsupported activation filters and
undeclared event reads remain compilation errors. The tap and activation rules
are separate: a tap-symbol activation emits the tap trigger once and is excluded
from the non-tap activation trigger. Mana activations still emit these events;
failed payments emit neither. Copies do not repeat activation or pay costs.

These adapters reuse trigger detection, APNAP placement, pending captures,
source/controller LKI, damage prevention, phasing and clone services. They add
no new payment, replacement-order or simultaneous-operation rules.
Rules: CR 113.7a, 602.2/2b, 603.2/2e/3a/3b, 605.3 and 608.2h.
Evidence: `TestArtifactActivationCapture*`, retained
`TestArtifactAbilityWithoutTapCost*` and `TestArtifactBecomesTapped*`, and
`TestRONArtifactEvent*` for complete Haunting Wind, Powerleech and Artifact
Possession records, native loading/casting, independent factories and tap batches.


## Random discard costs: locked selection and payment order

`DiscardRandomCost(n int) Cost` requires a nonnegative count and enough hand
cards to pay the full count. Definition validation rejects a negative count.
`CanPay` is a pure affordability query. Selection never calls the payer's hand
choice interface. Each distinct card selected consumes one game random result,
including the last card. Zero consumes none and does not lock unrelated hand
visits or random state.

The shared action/spell payment transaction moves direct random discard costs
after deterministic costs while preserving order within both groups. It locks
selection on the validation game after mana and deterministic payments. The
locked payment saves the candidate hand visits, selected hand visits and the
scripted random state before and after selection. The reusable declaration
stores no payment choice. Commit does not select again. A changed candidate
hand or scripted random state rejects the prepared payment. Clones keep their
own hand visits and random queues.

A transaction with random discard repeats the locked total payment on an
isolated game before spending live resources. A stale visit or changed discard
replacement therefore rejects the total payment before mana, tap or life costs
are spent. Direct `Pay` uses the same transaction. The random state advances
before discard replacements, so replacement choices see the post-selection
state. The existing game random service supplies unscripted outcomes; this
change does not add a seeded process random generator.

Random discard uses the existing cost-discard action and batch service:
prepare replacements before movement, check every proposed payment, move all
selected cards, then send events. Library destinations still pay this
unfiltered cost; prevention rejects the full payment. Effect-only discard
replacements do not apply. A prevented random card is not replaced with another
selection. Existing replacement ordering and affected-player decisions remain
owned by the shared service. No new LKI or phasing service is introduced.

Rules: CR 118.3/11, 601.2h, 602.2b, 616.1, 701.9a/b/c and 732.1.
Evidence: `TestRandomDiscardCostPayment`, `TestRandomDiscardCostOrderAndBatch`,
`TestRandomDiscardCostInvalidAndDirect`, `TestRandomDiscardCostCommitRecheck`,
`TestRandomDiscardCostRepeatedPayments`,
`TestRandomDiscardCostFailedCommitClearsResults`,
`TestRandomDiscardCostZeroIgnoresHandAndRandomChanges`, `TestRONRandomDiscardCosts`,
`TestRONRandomDiscardCostInvalid`,
`TestRONRandomDiscardCostModifierAndNestedContext` and
`TestRONRandomDiscardCostNativeLoadAndInstances`. Existing selected-discard
batch/replacement tests remain the shared-service evidence.

### RON land-dependent creatures

The RON adapter reuses `CanAttackOnlyIfDefendingPlayerControls`, `ControlsNone`
and `WhenState`. It adds no engine execution service. The defending-player
restriction reads current control and land subtype in the two-player combat
model (CR 508.1c). Phased matching permanents do not satisfy either player's
condition. Removing source abilities removes its static restriction and stops
new state triggers. A restriction that becomes false after attack declaration
does not remove that attacker.

The no-land condition is a state trigger (CR 603.8). It queues once while an
instance is pending or on the stack, and can trigger again after countering.
Acquiring a matching permanent before resolution does not cancel it. Its source
and controller use the existing instance capture contract. A returned source
cannot be sacrificed by an old trigger (CR 400.7); a phased source is absent
for sacrifice. Control changes do not transfer an existing trigger's controller.
A copied sacrifice ability uses its copy controller. Game clones keep pending
state triggers and their resolution independent.

Evidence: `TestRONLandDependenceCards`, `TestRONLandDependenceStateInstances`,
`TestRONLandDependenceInactiveSources`, `TestRONLandDependenceNativeAttack`,
`TestRONLandDependenceSubtypes`, `TestRONLandDependenceCasting`,
`TestRONLandDependenceOtherAbilities`, `TestRONLandDependenceInvalid` and
`TestRONLandDependenceReprints` in `pkg/mage/dsl/ron`. Complete RON gameplay
mappings cover Dandân, Sea Serpent, Pirate Ship, Island Fish Jasconius and
Vodalian Knights. Seasinger, Merchant Ship and Giant Shark retain other blockers.
This adapter does not establish general state filters or multiplayer combat.

## RON source attack event lowering

The compiler now lowers source-only `WhenACreatureAttacks/1` and
`WhenACreatureAttacksAndIsntBlocked/1` to the existing `Attacks(This)` and
`AttacksUnblocked(This)` patterns. The argument must be
`SinglePermanent(ThisPermanent)`. Each factory builds fresh abilities, effects
and target declarations. No new engine event or state service is introduced.

The first event occurs on attacker declaration (CR 508.3a). The second occurs
when blockers are declared and the attacker is unblocked (CR 509.1h, 509.3g).
Entry attacking does not constitute an attack declaration (CR 508.4).
`Trigger_ThatCreature` denotes the ability's captured source incarnation, through
`mage.This`; it does not use a separate mutable event slot. Pending abilities
keep their controller and source after control changes or departure (CR 603.3,
608.2h). A returned permanent has a new incarnation (CR 400.7). Phasing before
detection disables the source; phasing after detection leaves the pending
ability but prevents actions on the absent object (CR 702.26).

`UntilEndOfCombat` in a source attack action body reuses the engine's temporary
continuous effect expiry. It ends as combat ends, after end-of-combat priority,
while `UntilEndOfTurn` remains until cleanup. Removing a creature from combat
before this time does not terminate either bonus. Target selection, legality,
optional mana payment, damage prevention and source LKI retain their existing
shared contracts; no replacement-order or simultaneous-operation contract is
changed.

Evidence: `TestRONSourceAttackNative` runs real Cave People, Hasran Ogress,
Merchant Ship and Murk Dwellers through native attack/block steps, checking both
blocked and unblocked attacks and both durations. `Identity` checks source
incarnations, pending control changes, phasing, ability removal, two factories,
cloned games and copied triggers. `OtherRules` checks Ogress resolution payment,
decline, affordability, prevention and returned source, Cave People activation
cost rollback and landwalk cleanup, and Merchant Ship's no-Island state trigger.
`CastingAndRegistry` checks all four complete declarations, fresh factory
instances, paid casting and failed payment rollback in an isolated test registry.
`NoDeclaration`, `Invalid` and `Inventory` cover entry attacking, rejected
contexts and nested semantic roles. All these names have prefix
`TestRONSourceAttack`. Collection-wide runtime loading remains incomplete.

### RON delayed step capture

The RON adapter for `CreateFutureTrigger/2` uses the existing `AtNext` contract.
It creates a schedule on resolution (CR 603.7a), consumes it at the next matching
step (CR 603.7b), and retains the source incarnation and resolving controller
(CR 603.7c–e, 400.7). A schedule created after a step has begun waits for the
next matching step; cleanup does not consume an end-step schedule (CR 513.2).
Detection queues the trigger for the next priority opportunity. The body does
not run at registration or detection.

The adapter preserves captured typed permanent targets without a new target
choice at placement. Incarnation and phasing checks still apply to actions.
Sacrifice requires the original ability controller to control the subject when
it resolves; a later control change can prevent sacrifice without cancelling
an independent damage instruction. The existing damage pipeline uses source
LKI after sacrifice and applies prevention. Clone and stack-copy execution use
the shared capture services. No replacement ordering or simultaneous-operation
engine contract changes in this slice.

`TestRONDelayedStepRukhEgg` covers real RON casting, death, token characteristics,
next-step timing, cleanup, and clones. `TestRONDelayedStepRukhEggNonDeathsAndCopies`
covers exile, bounce, phasing, regeneration, copies, and distinct death
controllers. The other `TestRONDelayedStep*` tests cover source and target
incarnations, controller changes, ability removal, source damage after sacrifice,
independent X and target frames, captured event players, nesting, countered
triggers, all-illegal original targets, and invalid declarations. Event spell
bindings are not captured by the engine and are rejected in delayed RON bodies.

## Filtered source and attachment combat-pair capture

`BlocksCreatureMatching(subject PermanentRef, filter PermanentFilter)`,
`BecomesBlockedByCreatureMatching(subject PermanentRef, filter PermanentFilter)`
and `BlocksOrBecomesBlockedByCreatureMatching(subject PermanentRef, filter
PermanentFilter)` return event patterns with `WithOpponent(*PermanentBinding)`.
Subjects are `This` or `AttachedPermanent`; other references, nil/empty captures
and repeated captures fail definition validation. Existing patterns retain their
API. A zero-value filter follows the existing match-all filter contract, while
both event creatures must still be present, phased in and creatures.

Each pattern checks the matching direction, subject identity and opponent filter
at detection (CR 509.3b/d/f). Attachment lookup uses the current source and host
incarnations. Explicit event incarnations, when supplied, must match the live
objects. The filter is a pure query. It is not a target restriction or an
intervening-if condition. Shroud does not stop these untargeted triggers.
`WithOpponent` declares a typed producer and captures the exact opposing
incarnation before stack placement. Pending events, copied triggers and delayed
frames keep independent values. Later attachment or characteristic changes do
not recheck the event condition (CR 608.2k, 603.7c). Departure of the source does
not cancel pending or delayed abilities. Ordinary destruction still permits
regeneration and honors indestructibility. Return invalidates the captured
identity; phasing preserves it but prevents actions while phased out.

`TestCombatPairMatchingEngine` covers source/host and all three directions with
attachment movement, source departure, opponent return, phasing, control/color
changes, copying, regeneration and clone isolation. `Declarations` rejects invalid
subjects/captures; `Detection` covers color unions (including multicolor), Wall
exclusion, noncreatures, wrong direction/host, stale incarnations, source/host/
opponent phasing, source departure, ability removal and Aura-controller ownership.
These names have the prefix `TestCombatPairMatching`.

`TestRONCombatPairExtendedRealCards` covers complete Abomination and Venom RON
records in both combat directions, all color/Wall combinations, combat damage
and end-of-combat destruction. `Identity` covers saved opponents through stack
copies, pending clones, attachment changes, source departure, return, phasing,
color/Wall/control changes, regeneration and indestructibility. `Casting`
covers both costs and characteristics, Aura attachment/legality and rollback.
`IndependentEvents` checks two factory instances, two events each, multicolor
opponents, shroud and clone isolation. `Invalid` checks union shape, lexical
bindings and unsupported filters; `Inventory` checks nested event/filter/reference
roles under an unsupported body. These names have the prefix
`TestRONCombatPairExtended`. The existing `TestRONCombatPair*` suite retains
Cockatrice and Thicket Basilisk coverage for the shared compiler path.

No new LKI read, cost service, replacement ordering, simultaneous-operation or
combat event generation is introduced. This adapter does not support history
filters, group unions or conditional destruction results.

## RON indefinite resolving type changes

`CreatePermanentLayerEffect` with one `AddCardtype` or basic `SetLandType`
operation reuses the existing `Until(Indefinite)` capture contract (CR 611.2a,
611.2c). The source does not control its lifetime. Subject departure invalidates
its captured incarnation; phasing preserves that incarnation. Targets must pass
resolution legality before capture. No absent-object action uses LKI.

`SetSubtypeFamily(subject, SubtypeLand, basicType)` applies CR 305.6–7 in layer
4: replace land subtypes, remove printed abilities, grant intrinsic mana and
retain other card types, subtype families, supertypes and later granted abilities.
`AddCardType` retains existing types and abilities. No engine service changes.

Evidence: `TestRONIndefiniteTypeRealCards` exercises both full RON records with
source departure, target return, phasing, control changes, cloned pending and
installed effects, and cleanup. `Casting` checks native mana payment and
rollback, metadata and fresh factories. `MonkSelfPayment` checks summoning
sickness rollback and source sacrifice. `LegalityAndReplacement` checks a target
becoming an artifact before resolution, counter prevention/doubling, and copied
abilities. `InstanceBindings` checks distinct sources and recipients.
`AllBasicTypes` checks all five mana colors, retained creature characteristics
and legendary supertype, removed printed flying and retained granted vigilance.
`Invalid` and `Inventory` cover declaration rejection and nested reporting.
These suffixes share the prefix `TestRONIndefiniteType`.

## RON attachment target lowering

The RON `AttachedToAPermanent(hostFilter)` target adapter reuses `IsAttachedTo`
and `TargetAttachedToYouControl`. No engine rule changes. CR 303.4b/e separates
the Aura from its host and their controllers. CR 601.2c, 602.2b and 608.2b require
target legality at announcement and resolution. The host query uses current
characteristics and attachment, including after reattachment. Only the Aura
is targeted; host shroud does not prevent the activation. The ability controller
stays fixed after source control changes or departure. A copied ability uses
its own controller for the host restriction.

The target uses the existing captured incarnation (CR 400.7). A returned Aura
cannot be affected by an older activation. Returning the host and attaching the
original Aura to it does not invalidate that Aura target if the new host
qualifies. Phasing follows CR 702.26b/d: phased targets are illegal; a phase
cycle preserves identity. Destruction uses normal regeneration, indestructible
and owner-zone services. Transactional cost payment and clone isolation remain
in the shared engine services.

`TestRONAttachmentTargetRealCards`, `Rollback`, `Casting`,
`InstancesAndCopyController`, `ShapesAndInventory` and `TriggerPlacement`
(all with the `TestRONAttachmentTarget` prefix) cover these contracts. The real
DRK records for Miracle Worker and Savaen Elves supply complete behavior tests.
No LKI read is used to act on an absent object. No batch or replacement-order
contract is added by this adapter.

### RON plural target color adapter

`OneOrMoreTargetPermanents(filter)` lowers to a typed
`DeclarePermanentTargets(name, 1, math.MaxInt, filter, options...)` declaration.
The upper bound imposes no additional game limit. The declaration has no chosen
state. Factory-local bindings connect `Ref_TargetPermanents` to this group.
Spell, mode, activation, and trigger contexts use the same adapter. Delayed
color actions remain compile errors.

The one-operation `SetColor(SimpleColorList(colors))` action on this collection
uses `Until(EndOfTurn, SetEachColors(binding, colors...))`. Target legality is
checked at announcement and resolution (CR 115, 601.2c, 608.2b). Legal members
are captured together when the effect starts (CR 611.2c). Layer 5 replaces their
colors (CR 613.1e); it does not change types, abilities, or P/T. Departure ends
the captured incarnation's participation (CR 400.7). Phasing retains identity
but excludes a phased target from resolution (CR 702.26b). Cleanup ends the
effect. These engine services retain their existing contracts.

`TestRONPluralColorRealCards` verifies complete Dwarven Song, Heaven's Gate,
Sea Kings' Blessing, Sylvan Paradise, and Touch of Darkness records, their mana
payments, plural selection, replacement colors, timing, clones, and cleanup.
`TestRONPluralColorLegalityAndIdentity` checks partial/all-illegal targets,
return, phasing, control, copies, and clone isolation. `Announcement` checks
zero/duplicate targets, wrong types, shroud, phasing, and payment rollback.
`InstancesAndLayerOrder` checks independent factory instances, multiple colors,
later layer-5 effects, and departure after resolution. `RetargetCopy` checks
separate copy choices. `Contexts` checks modes, activation source departure,
trigger placement, and an empty color list. `Invalid` and `Inventory` check
rejected declarations, mode scope, and nested semantic roles. All test names
have the prefix `TestRONPluralColor`.

### RON fixed/X target batch adapters

`NumberTargetPermanents(Integer(n), filter)` uses equal minimum and maximum
bounds in `DeclarePermanentTargets`; `ValueX` uses `DeclareXPermanentTargets`.
Announcement fixes X and the selected incarnations (CR 115, 601.2b–c).
At resolution, the engine removes illegal targets from the bound collection;
all-illegal nonempty groups prevent the whole spell from resolving (CR 608.2b).
An empty zero-count group does not stop other instructions.

`TapEachPermanent(Ref_TargetPermanents)` uses `TapEach`.
`Exile([EachPermanent(Ref_TargetPermanents)], [])` uses `ExileEach`.
The existing engine batches prevent triggers from observing an intermediate
member transition (CR 608.2f). Departure and return do not preserve the target
incarnation (CR 400.7); phasing excludes a target while phased but preserves
its identity across a phase cycle. Collection actions require live legal
objects and do not use LKI to act on absent objects. Existing damage prevention
applies to Ashes to Ashes after the exile instruction.

`TestRONNumberedTargetsRealCards` covers complete Ashes to Ashes, Dust to Dust,
and Word of Binding records, payments, resolution timing, both controllers,
unselected objects and pending clones. `Legality` covers partial/all-illegal
resolution, departure, phasing, control, copies and preventable damage.
`Announcement` covers exact bounds, duplicate/illegal targets, artifact
creatures, rollback and X=0/1/3. `Batch` checks simultaneous exile and tap.
`Contexts` checks mode, activation with source departure, and trigger actions.
`InstancesAndCopy` checks independent factory calls, separate X values and copy
retargeting. `FixedZero` checks empty fixed groups and continued resolution.
`Invalid` and `Inventory` check declarations and nested roles under rejected
parents. All test names have the prefix `TestRONNumberedTargets`.

## Conditional fixed mana production

`NewConditionalManaAbility(condition StateCondition, whenTrue, whenFalse []ManaProduction) *ManaAbility`
provides a source tap mana ability with two fixed production branches. Both
branches require at least one production, concrete colors, positive amounts,
and no restrictions or color choices. Definition validation checks these
requirements and the condition, including invalid or undeclared player reads.
The constructor copies branch slices; production queries return fresh slices.

The condition reads the current source controller and battlefield at each query
and activation. It must remain unchanged by the source tap cost: payment queries
precede that cost. The RON adapter therefore excludes tap and combat status
filters. A missing or phased source has no production. A phased prerequisite
is absent from battlefield membership; a returned prerequisite is a new object
that can satisfy the current collection condition. There is no LKI or saved
condition result. Queries do not tap, add mana, choose, or consume randomness.

The existing `ManaAbility`, `ManaSystem`, mana solver and transactional casting
paths consume the same production service. Activation resolves immediately
without a stack object (CR 605.1a, 605.3a–b). Bonus mana applies once per tap,
regardless of the number of production entries. Ability removal suppresses
activation through existing ability membership. Failed payments keep live
resources unchanged. Conditions and branch declarations hold no game state;
cloned games query their own battlefield and mana pools.

Evidence: `TestConditionalMana`, `TestConditionalManaInvalid`, and
`TestRONConditionalMana*`. The complete ATQ records for Urza's Mine, Urza's Power
Plant and Urza's Tower exercise current land subtypes, rather than card names.
This contract does not establish general conditional resolving actions,
choice-based or restricted mana, extra costs, or conditions changed by tap costs.

## Live source and attachment filter conditions

`HasCounter(core.CounterType) PermanentFilter` checks for one or more counters
of one kind. `CounterCountEquals(core.CounterType, int) PermanentFilter` checks
the exact count. Nil permanents never match, including an exact zero query.
Negative or greater-than-storage-range counts never match. Both queries are
pure and read the supplied permanent; zone presence and identity remain the
responsibility of the selector or `PermanentMatchesFilter`.

The counter predicates do not use departure LKI. They reuse current counter
storage (CR 122.1), live static membership (CR 611.3a), and the existing
incarnation and phasing checks. A returned permanent cannot satisfy an old
condition, even if it has the same count. A phase cycle retains counters.
No new replacement, counter-placement, cost, trigger, or batch service is added.
`TestCounterFilters` covers independent counter kinds, zero, storage boundaries,
and nil. `TestCounterFilterConditionIdentity` covers source/host absence,
phasing, return, stale source incarnations and clone isolation.

`PermanentMatchesFilter(PermanentRef, PermanentFilter) StateCondition` accepts
`This` or `AttachedPermanent`. Definition validation rejects nil, target and
saved references for this static context. The zero filter matches any live
subject. Execution checks the source incarnation from `StateContext` when one
is supplied, then the attachment incarnation for an attached subject. Absent,
phased or returned subjects cannot satisfy the old reference (CR 400.7,
702.26b). There is no departure LKI fallback.

The condition uses the shared `FilterBattlefield` query and has no mutable
chosen state. Attachment and characteristic changes update the condition;
control changes preserve subject identity. `While` and the existing layer
service apply static membership again on each pass (CR 611.3a–c). Type and color
changes precede layer 7c bonuses, while layer 6 ability removal disables the
source rule (CR 613.1d–g, 613.4c). This primitive adds no target, replacement,
transactional cost, event, or simultaneous-operation service.

Evidence: `TestPermanentMatchesFilterIdentity` and
`TestPermanentMatchesFilterDeclarationsAndLayers` cover identity, missing/stale
references, source and host phasing, changing attachments, control, return,
clone isolation, declaration rejection, color changes, ability removal and
source destruction. `TestRONPermanentFilterGoblinCaves`, `Invalid` and
`SourceInstances` add RON scope validation, actual Aura casting, cost and target
legality, basic versus nonbasic Mountains, prior-layer type changes, both
players' Goblins, later entries, recipient subtype changes and independent
factory instances. The RON adapter retains its existing restricted static
filter and conditional body grammars.

## Unlimited prevention for a captured recipient

`PreventDamageThisTurn(recipient DamageRecipientRef, from PermanentFilter)`
installs an unlimited end-of-turn prevention rule through the existing
`damagePreventionRuleReplacement`. The recipient is resolved when the effect
resolves. Player references keep that player; permanent references keep the
selected incarnation. Control changes and a phase cycle preserve that identity.
A departed, returned, or phased recipient cannot receive a new rule from an old
reference. A rule already installed resumes after its recipient phases in.
Source departure or ability removal does not remove the installed rule.
Cleanup removes it (CR 400.7, 514.2, 615.3–4, 702.26).

A zero source filter permits any source, including spells. Nonzero permanent
filters use the existing `damageFilterSource` live/departure/phasing views.
They recheck properties when damage would occur (CR 608.2h, 609.7b).
`DamageSourceIsAttacking()` reads the current source incarnation's combat status,
or the status saved in `PermanentLKI.Attacking` before departure or phasing.
It does not read the combat status of a returned card. Phasing in returns to
live queries; it does not restore combat membership. The normal `IsAttacking`
filter keeps its live battlefield behavior. Blocking and numeric source filters
have no new retained-information contract.

Both combat and noncombat damage use the shared replacement pipeline. Each
simultaneous packet can be fully prevented without consuming the rule. Finite
shield allocation remains separate (CR 615.7). Redirection and prevention use
the affected player's choice and recheck the new recipient (CR 616.1). Existing
unpreventable-damage handling remains responsible for bypassing prevention.
Declarations reject missing or unbound recipients before execution. Instances,
stack copies, and game clones keep their own bindings and replacement state.

Evidence: `TestPreventDamageThisTurnIdentity`, `SourceAndBatch`, `Declarations`,
`AttackingSourceLKI`, and `ReplacementOrder` under the same test prefix;
`TestRONRecipientPreventionAura`, `Artifacts`, `ArtifactCasting`, `Combat`,
`Instances`, `CopyRetarget`, and `InvalidAndInventory`.

## Typed current-hand operations

`RevealHandFor(PlayerRef)` records each card in the resolved player's current
hand with the existing public reveal service. It does not move cards or make a
choice (CR 701.20a–c). History entries are snapshots; later changes and game
clones do not change earlier entries. `CountMatchingHand(PlayerRef, CardFilter)`
is a pure resolution query of current hand membership and card characteristics.
It does not use reveal history or LKI. An unresolved or absent player supplies
no cards and a count of zero.

`DiscardMatchingHand(PlayerRef, CardFilter)` selects every matching card at
resolution, then uses `discardActionsTogetherWithEvents`. All proposed effect discards
are prepared through the existing affected-player replacement pipeline before
any move. Discard events follow all moves (CR 608.2f, 701.9). A replaced or
prevented discard does not stop other selected discards. No hand-choice prompt,
random selection, payment or result binding is introduced. A zero card filter
matches every card, as in the existing `CardFilter` contract.

Player references retain the resolving controller and target binding. A source
control change, departure or phasing does not change a captured ability's
controller. Fresh definitions, stack copies and cloned games use the existing
binding and resolution frame services. Declaration validation rejects nil and
unbound player references. `TestHandOperations*` establishes engine evidence;
`TestRONHandOperations*` covers complete Amnesia and Inquisition records, their
casting, copies, replacement effects and supported resolving contexts.

## RON damage amount capture

The RON `WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent))` event
uses the existing `DealsDamage(This, amount)` pattern. An Aura's
`SinglePermanent(HostPermanent)` uses `AttachedPermanent`. This adapter adds
no damage or event rule. CR 603.2c/g and 603.3a/b require actual event amounts,
separate occurrences, deferred resolution, and the event-time controller.
The engine saves each actual post-replacement amount in its pending trigger.
Each RON factory creates a separate number binding. `Trigger_AmountOfDamageDealt`
reads that binding only in the trigger's resolving scope.

`TestRONDamageAmountCards` covers prevention, event/controller capture,
attachment changes, source departure/return/phasing, ability removal, copies and
clone isolation. `TestRONDamageAmountBatchAndInstances` covers independent
factory bindings, a pending-trigger clone, and simultaneous player/permanent
packets from two sources. `TestRONDamageAmountCombat` covers native combat to
players and blockers. `Casting` and `AuraTargetLegality` cover payment rollback,
attachment and resolution target legality. `Invalid` rejects malformed subjects,
amount tuples, cross-event amount reads and unavailable player bindings.
These tests map the complete El-Hajjâj and Spirit Link RON records.


## RON opponent-damage event lowering

The exact RON source/opponent shape documented in `docs/dsl-reference.md`
uses `DealsDamageToOpponent(This, binding)`. RON adds no event rule. The engine
checks the event-time source controller, saves the damaged player before stack
placement, and retains separate facts for separate events (CR 603.2c/g,
603.3a/b). The player is not a target. Each RON factory has its own typed binding.
Copied triggers retain the captured player even when their controller changes.

Bare `PlayerAction(player, DiscardHand)` reuses the zero-filter
`DiscardMatchingHand` contract above. The hand is selected at resolution;
all moves precede discard events. Existing discard replacements remain active.
`TestRONDamagePlayerCards` covers prevention, opponent identity, control,
departure/return/phasing, ability removal, copies and pending clones.
`Instances` covers simultaneous damage from independent factories/controllers.
`DiscardReplacement` and `HandBatch` cover replacement and complete-batch event
visibility. `Casting`, `Combat` and `TestRONUpkeepPaymentRealCards` complete the
Hypnotic Specter and Nicol Bolas gameplay mappings. `Invalid` and `Inventory`
cover declaration rejection and semantic roles. No engine contract is changed.

## Typed player poison placement and RON token triggers

`AddPoisonFor(PlayerRef, ResolutionValue)` uses the existing typed resolution
frame, player poison state and clone service. It resolves a player first, then
reads the amount without changing targets. An absent player reference or a
nonpositive amount causes no change. Definition validation rejects nil subjects,
nil values and unbound reads. It does not deal damage, change life, or run
state-based actions during the instruction. Ten poison counters cause loss at
the existing state-based action check (CR 122.1f, 704.5c). No new player-counter
replacement family is introduced.

RON source damage-player events with bare `AnyPlayer` reuse
`DealsDamageToPlayer(This, binding)`. Zero and fully prevented damage cannot
trigger; partial prevention changes damage but does not scale the fixed poison
amount (CR 603.2g). Separate events retain separate players. Source departure,
return, phasing and control changes cannot redirect a queued trigger. Copies
retain the player even when their resolving controller changes.

RON token damage-player triggers use a fresh lexical permanent scope. They
cannot borrow the creator's targets, announced X or saved results. Token
characteristics and abilities are printed characteristics (CR 111.3). Each
factory creates a fresh `DefineToken` declaration with `TokenAbilities`;
existing engine cloning gives each token a distinct runtime ability ID and
source while immutable binding declarations use per-resolution values.
Token owners/controllers use the resolving creation controller (CR 111.2).
Payment rollback, copied creation, simultaneous damage, APNAP placement and
pending clone isolation reuse existing engine services.

Evidence: `TestAddPoisonFor`, `Validation` and `EventValues`; the RON
`TestRONPoison*` suite covers the three complete cards, real casting/combat,
prevention and self damage, source/recipient identity, phase cycles, ability
removal, independent factory/token instances, copies/clones, payment rollback,
poison loss, invalid declarations and nested inventory roles. No static, entry,
cost or new replacement-operation adapter is established.

## Casting conditions before announcement

`WithCastingCondition(condition StateCondition) ActionOption` adds a pure
casting restriction to an `ActionSpell`. Multiple conditions all must pass.
The condition receives the casting card ID and the caster ID in `StateContext`.
Definition inspection visits these conditions before target declarations and
payment captures. Nil conditions, invalid condition trees, unavailable bindings,
battlefield-source reads and casting conditions on activated actions fail
validation. This service supplies no game-time choices or mutable result slots.

`IsDuringSteps(steps ...PhaseStep) StateCondition` reads `TurnReader.GetStep`.
The declaration copies its input slice. Empty lists and invalid enum values
fail definition validation. Immutable conditions remain safe in copied cards,
stack objects and cloned games. Existing `IsYourTurn` reads the current active
player relative to the caster. These APIs impose restrictions; they do not
supply priority or casting permission (CR 117, 601.2, 601.3).

The shared `checkSpellTiming` gate applies conditions and the retained step
restrictions before modes, target acquisition and payment. Its callers include
hand casting, free and alternate-cost zone casting, cast copies and castability
queries. A foreign-owned card uses its caster. Ordinary sorcery timing still
applies on the hand path. Existing zone-casting permissions keep their behavior.
Copying an existing stack spell is not casting (CR 707.10), and resolution does
not repeat the condition (CR 608).

Tests: `TestSpellCastingConditionPaths`, `CopiesAndValidation`,
`TestSpellCastingStepsAndConjunction`, and the retained `TestSpellStep*` suite.
RON evidence: `TestRONCastingWindows`, `WindowGameplay`,
`WindowTeleportIdentity`, `WindowOrderScopeAndInstances`, `WindowStackCopies`,
`WindowInvalid` and `WindowInventory`. Reset's collection action reuses
`UntapEach`; existing `TestUntapEachBatchAndStun`, `IdentityCloneAndDeclaration`
and `TargetGroupKeepsAnnouncementIncarnation` retain the simultaneous operation,
stun replacement, identity and clone contracts. No new untap rule is established.

`AfterFirstUpkeep() StateCondition` uses turn-owned completion state (CR 503.2).
The state becomes true when an actual upkeep finishes. It stays true through
later upkeep steps, extra phases and repeated cleanup steps in that turn.
Turn changes clear it. Cloning copies it without sharing mutable state.
Skipped upkeeps do not complete an upkeep. `SetStep` records departure from an
upkeep in the same turn for direct engine/harness callers; `RunStepWithPriority`
records completion even when the next step is another upkeep. The stored step's
turn identity prevents a new turn from inheriting the old turn's final step.
`TurnReader.UpkeepHasEnded` supplies the pure history query. Tests:
`TestSpellAfterFirstUpkeep` and `TestRONCastingWindowResetExtraUpkeep`.

## Numeric permanent target restrictions

`TargetToughnessLessThanSourcePower()` composes with typed permanent target
filters and controller restrictions. It reads current target toughness and
source power at announcement and resolution (CR 601.2c, 602.2b, 608.2b).
Equal values are illegal. A departed source supplies retained incarnation power,
including continuous bonuses at departure. A returned source cannot replace it.
A phased source supplies phase-time power; phase-in restores current queries.
The existing target-source information service owns this data (CR 400.7,
608.2h, 702.26b). Copies retain the source but use their own ability controller.
Missing source information permits no targets. The option adds no target state.

RON `PowerAndToughnessIsEqualTo(PT(p,t))` checks both current characteristics.
`ToughnessIs(EqualTo(Integer(n)))`, `LessThanOrEqualTo`, and
`GreaterThanOrEqualTo` reuse exact engine filters, including negative values.
Only permanent target contexts support
`ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent)))`; the source reference
must be available in that scope. Other comparisons and references are rejected.
Stone Giant's later destruction uses the existing delayed captured reference;
changes to target toughness, controller or shroud do not repeat targeting.
Departure/return and phasing retain their existing delayed identity contracts.

Evidence: `TestTargetToughnessBelowSourcePower`, `LayerLKI`, and
`TestRONNumericTargetsCards`, `Legality`, `MetadataAndPayment`,
`DelayedIdentity`, `Instances`, `Invalid`, `FixedQueries`, `Inventory`.

## RON coin branch execution

RON win/loss branches reuse `FlipCoin`, `OnWin`, `OnLoss` and `Sequence`.
The resolving controller makes one random choice at resolution (CR 705.1–2).
Only the selected branch runs, in order (CR 608.2c). No outcome is chosen during
compilation, casting, activation payment or delayed-trigger installation.
All-illegal targets stop resolution before the flip (CR 608.2b). Stack copies
flip separately and use their own controllers. `RandomSource.Clone` keeps
scripted outcomes independent between cloned games.

Goblin Kites uses the existing `AtNext` captured delayed body. Its coin is
flipped at the next end-step trigger's resolution, even if the saved creature
has left or phased out. A returned creature has a new incarnation (CR 400.7).
Phasing preserves identity but prevents sacrifice while absent (CR 702.26b).
The trigger has no new target and does not recheck toughness or shroud
(CR 603.7c). The original resolving controller cannot sacrifice a creature now
controlled by the opponent (CR 701.21a). The one-shot trigger is consumed even
when its branch cannot perform the sacrifice.

Bottle of Suleiman's sacrifice is a payment. Its later damage uses the existing
source-information and prevention pipeline (CR 608.2h, 615); it cannot follow a
returned source's controller. Token creation uses the resolving controller.
Orcish Captain uses existing target legality and temporary layer modifications.
This adapter adds no engine API, replacement order, simultaneous-operation or
transaction contract. Supplied actions retain their established services.

Evidence: `TestRONCoin*` and retained `TestComposableCoinFlipBranches`,
`TestComposableCoinFlipValidation`, `TestRONDelayedStep*` and token/damage tests.


## RON source block event adapter

RON `WhenACreatureBlocks(SinglePermanent(ThisPermanent))/1` reuses `Blocks(This)`
and the engine's final legal block assignments (CR 509.1, 509.3a). The adapter
adds no event system. Multiple pairs produce one source block trigger; entering
blocking produces none (CR 509.4). `Trigger_ThatCreature` uses the pending
ability's source incarnation, not a global event slot. Delayed actions keep
that source and the resolving controller through `AtNext` (CR 400.7, 603.7).
Departure, control changes and phasing retain the existing source-reference,
sacrifice and damage LKI contracts. Damage still occurs if sacrifice cannot
occur. Copies and clones keep independent pending actions.

RON symbol filter `IsntEnchanted` reuses `Not(IsEnchanted)`. Only current,
unphased Aura attachments count. Casting and resolution use the existing target
legality checks (CR 601.2c, 608.2b). No new engine API or replacement policy is
established. `TestRONSourceBlockTimeElementalNative`, `Identity`, `Activation`,
`LegalPairsAndEntry`, `FactoryAndFilterRecheck` and `InvalidAndInventory` provide
real-record and declaration evidence for these adapter contracts.

## Collection keyword removal and composed ability-removal dependencies

`RemoveEachKeyword(PermanentSet, Attr) ContinuousModification` removes all
instances of the selected keyword in layer 6 (CR 613.1f, 613.9). Other keywords
and nonmembers retain their abilities. It uses the shared collection selection,
incarnation capture, attribute removal, timestamps and layer installation.
`Until` captures membership at resolution (CR 611.2c); `While` selects current
members (CR 611.3a). Captured removals survive source departure and subject
phasing, but cannot follow returned incarnations (CR 400.7, 702.26).

Printed static effects now also detect dependencies on composed
`RemoveAbilities` and `RemoveAbilitiesEach` actions. The existing isolated layer
probe checks whether the proposed removal changes the printed effect's active
state. This permits removal to suppress the source before its static effect
applies in the same layer (CR 613.8a/b). The legacy ability-removal dependency
path remains available. Probes do not change live or cloned game state.

`TestRemoveEachKeywordSelectionAndLifetime` checks current versus captured
membership, later entries, preserved keywords, noncreatures, source departure,
phasing, return and clone isolation. `TestRemoveEachKeywordSourceAbilityDependency`
checks both composed removal forms and restoration after cleanup.
`TestRONLastingKeywordGravitySphere` covers casting payment rollback, global
selection, World metadata, source lifecycle, current control, later entries,
timestamp order and clones. `TestRONLastingKeywordWurmNative` covers full casting,
P/T, defender/trample, native blocking, indefinite duration and attack legality.
`TestRONLastingKeywordWurmIdentity` covers pending source incarnations, absence,
phasing, control, copied triggers and cloned pending work.
`TestRONLastingKeywordTargetInstances` checks independent factory bindings,
invalid-target cost rollback, copies, source departure and clones.
`TestRONLastingKeywordInvalid` checks declaration and context rejection.
No damage, LKI read, replacement ordering or cost execution service changes.

## RON owner-scoped batch return

`TestRONReturnCollectionRecall` executes Hurkyl's Recall from its complete RON
record through casting and resolution. It verifies owner/controller separation,
current resolution membership, late entry, new incarnations, type changes,
phasing, nontargeted shroud/indestructible recipients, tokens, retargeted copies,
cost rollback and clone isolation. `TestRONReturnCollectionBatchAndInstances`
checks independent activation controllers and an observer that must not see an
intermediate battlefield state during a simultaneous return. Declaration and
semantic-role tests are `TestRONReturnCollectionInvalid` and `Inventory`.

The adapter reuses `ReturnEachToHand`, `PermanentsOwnedBy`,
`MatchingPermanents` and `BouncePermanentsToHand`. It changes no engine service.
Rules: CR 400.3, 400.7, 608.2b/c/f/h and 702.26b–d. Current membership has no
saved permanent or LKI read. Existing target and zone services retain legality,
replacement and identity handling. No new replacement ordering is established.

## Combat removal with historical sole-blocker unblocking

`RemovePermanentFromCombatAndUnblock(PermanentRef) ResolvingEffect` removes the
resolved live incarnation and makes its historically sole-blocked attackers
unblocked. Each combat group retains the set of blocker ID/incarnation pairs
added during that combat. The operation requires a matching current blocker and
exactly that one historical pair. A blocker removed before resolution, including
by departure, control change or phasing, cannot unblock its former attackers.
A different blocker that left earlier still disqualifies sole-blocker status.
Returned incarnations remain distinct. Multiple attacker groups are checked
independently. Combat reset discards history and `Game.Clone` copies each map.
Existing ordinary removal keeps attackers blocked (CR 509.1h); legacy unblocking
callers retain their existing API behavior.

Battlefield departure now removes live combat membership after its LKI snapshot
(CR 506.4). The snapshot retains attacking status, while historical blockers and
the attacker's blocked status remain separate from current combat membership.
The old departure regression now asserts this separation for both incarnations.

`CantBlock(PermanentRef) ContinuousModification` uses the existing per-blocker
filter service with an always-false filter. With `Until(EndOfTurn, ...)`, it keeps
its captured incarnation through control changes and phase cycles, survives
ability removal, and expires at cleanup. It cannot affect a returned object.
It restricts later declarations and does not undo a legal block (CR 506.4a,
509.1b). Nil and unbound references fail definition validation.

Evidence: `TestCombatExitSoleBlockerHistory`, `DepartureKeepsHistory`,
`MultipleGroupsAndCloneHistory`, `Declarations`, `TestCantBlockDurationAndIdentity`
and `TestCombatDeparturePreservesAttackerBlockedStatus`; RON native gameplay and
pending/copy/target tests are mapped in the progress ledger. This slice adds no
new payment, damage, replacement, LKI-action or APNAP service.

The departure snapshot also retains the exact current combat-opponent
incarnations for `ForEachCombatOpponent`, an existing engine caller of combat
state after death. It uses the source's captured incarnation to read that
snapshot, then resolves only still-present, unphased opponents of those exact
incarnations. A returned source or opponent cannot replace a saved object.
`TestCombatExitDepartedOpponentCompatibility` verifies both directions, source
and opponent return, opponent phasing and cloned departure information. Existing
Abu Ja'far regressions pass without a Go card-definition change. This establishes
compatibility, not RON compilation or full behavior coverage for Abu Ja'far.

## Spell targets that target a controlled permanent

`DeclareSpellTargetTargetingControlledPermanent(name, filters...)` declares one
spell target. Optional card filters restrict its current card characteristics.
The selected stack object must target at least one permanent controlled by the
player who chooses this spell target. Abilities and objects without spell cards
are excluded. Existing `DeclareSpellTarget` callers keep their behavior.

The check uses the selected stack object's target zones and captured permanent
incarnations, plus each permanent's current controller. It runs at announcement
and resolution through the normal target service. CR 115.9b permits a matching
permanent whose target legality has changed, such as one that gained shroud.
Absent, returned and phased-out targets do not qualify. A phase cycle preserves
identity. Departure information is not used. Another matching target can keep
the spell eligible when one target departs. Each copy is checked by its own stack
identity and current target list. A counter copy uses its own controller.

The declaration uses existing definition validation, target bindings, stack
copying, game cloning, transactional costs, resolution checks and `Counter`.
Source departure, phasing or control changes do not change a pending ability's
controller. An uncounterable spell can qualify but `Counter` cannot remove it.
Tests: `TestSpellTargetTargetingControlledPermanent`, its `Identity` test,
and the RON `TestRONControlledPermanentCounter*` tests. Relevant rules:
CR 115.9b, 400.7, 601.2c, 608.2b and 702.26b/d.

## Announced X activation mana and reductions

The activation payment transaction substitutes its announced X into a
`ManaCostPayment` before applying the existing activation cost reduction.
Multiple X symbols multiply that value; generic and colored components retain
their payment requirements. The reduction changes the locked mana payment,
not the saved stack X or target count (CR 107.3a/i, 601.2b/f, 602.2b).
The existing reduction's minimum-total-mana restriction still applies.

`TestXActivationPaymentReduction` covers zero, single and double X, colored and
fixed components, reduction ordering, exact payment and retained effect values.
`TestXActivationPaymentAggregateReduction` checks composed costs and existing
`XManaCost` callers. `TestXActivationPaymentNegativeRollback` checks rejection
without live cost or stack changes. Existing total-payment and target-dependent
mana regressions remain in force.

RON `PayManaX(manaSymbols, ValueX)` supplies the typed activation's declaration;
`UntapEachPermanent(Ref_TargetPermanents)` uses the existing typed set service.
`TestRONXActivationCandelabra`, `AutotapAndReduction`, and `InstancesAndCopies`
cover actual casting/activation payment, rollback, announcement target bounds,
partial/all-illegal resolution, source departure/control/phasing, target return
and phasing, legal phase cycles, automatic mana planning, inactive-turn timing,
retargeted copies, independent factories and cloned pending abilities. Identity
and phasing use CR 400.7, 608.2b and 702.26b/d. Absent targets are not actions
on LKI. This slice adds no replacement ordering or simultaneous-operation rule;
existing mana and untap services supply those operations.

## RON activation step adapter and combat-duration reuse

Bare activation step conditions use `WithActivationCondition` with an immutable
list of allowed steps. They add no engine timing or permission service.
`IsDuringCombat` covers `BeginCombat` through `EndCombat`, including the optional
first-strike damage step (CR 506.1). The other supported predicates select
`EndCombat`, `DeclareBlockers`, or `Upkeep`. They permit either player's turn.
The existing activation gate checks these instructions before payment; copies
and resolution do not repeat them (CR 602.1b, 602.2, 602.5). Modifier conjunctions
preserve own-turn restrictions and activation limits. Mana queries remain pure.

Immediate singular activation modifications with `UntilEndOfCombat` reuse
`Until(core.EndOfCombat, ...)`. Source departure does not end the captured
modification. Subject return creates a new object; phase cycles keep identity.
Native end-of-combat execution removes effects after its priority round and
before combat reset (CR 511.2–3). No new replacement-order or simultaneous-operation
contract is established. Existing damage and target services execute Desert.

Evidence: `TestRONActivationSteps`, `Invalid`, `ConjunctionAndMana`,
`JadeStatue`, `Desert`, `CombatDurationTargets`, `CombatDurationInvalid`,
`JadeCombatAndInstances`, and `Inventory` in `dsl/ron/activation_step_test.go`.

## Player-relative amounts in one damage instruction

`DealDamageEachPlayer(source DamageSourceRef, subjects PlayerSet,
member *PlayerBinding, amount ResolutionValue) ResolvingEffect` selects current
player membership once. Each distinct existing player receives one assignment.
The member is a local producer for `amount` only. Definition validation rejects
nil inputs, unnamed bindings, unbound reads and reads that escape into the
player set or later actions. Outer bindings are restored before replacement
processing and damage notifications.

All amounts are evaluated before the first replacement or damage action
(CR 608.2h). Nonpositive amounts produce no assignment. The shared
`DealSimultaneousDamage` service processes replacements, prevention and damage
(CR 120.4, 615.6, 616.1); redirection does not reevaluate the amount. Damage
notifications observe the complete batch. Existing damage-source capture retains
incarnations and departure information (CR 400.7, 608.2h). A pending activation
keeps its controller; lifelink uses its source's current or retained controller.
Phasing does not create a new source incarnation.

`TestPlayerDamageAmounts*` covers definition validation, local scope restoration,
empty/duplicate/absent membership, nonpositive amounts, pre-batch information,
prevention, affected-player replacement ordering and rechecks, source
control/departure/return/phasing, lifelink and cloned pending activations.
`TestRONDamageRecipientValue*` covers the compiler adapter and complete
Storm Seeker and Typhoon records. No optional-payment or discard-result
contract is established here.

## Finite recipient shields in simultaneous damage batches

`Game.AddPreventionShield(uuid.UUID, int)` and
`PreventNextDamage(DamageRecipientRef, ResolutionValue)` keep their signatures.
The ordinary recipient shield now allocates its capacity across the current
simultaneous damage packets before any damage executes (CR 615.7). With multiple
source identities and insufficient capacity, the affected player or the
permanent's current controller chooses each prevention amount through the
existing `Player.ChooseNumber` service. Bounds reserve enough capacity for the
remaining packets, so all applicable shield capacity is used. Capacity sums are
bounded by the shield amount to avoid integer overflow. A forced amount,
a single source, or sufficient capacity requires no allocation prompt. Invalid
answers use the legal maximum. Source identity includes the saved incarnation;
labels use retained source information, without a live-source requirement.

Batch replacement choices use the existing priority tiers, effect labels and
`Player.ChooseMode` service (CR 616.1). Choices occur in APNAP order. A player can
select a replacement for a later packet before allocating a shield. Each
packet keeps its own applied-effect set; redirection cannot reapply the same
rule to that packet. Matching and priority are checked again after each selected
replacement (CR 616.1f). Redirected packets can join a recipient's shield group.
A shield allocation changes the damage amounts, not the saved damage sources.
Shield capacity, captured recipient incarnation, cleanup expiry and clone
ownership remain in the existing replacement service. Damage with the existing
unpreventable/unredirectable attribute does not consume shield capacity.

Native `Combat.ResolveDamage` now collects its assignments before calling
`DealSimultaneousDamage`, once for each damage step (CR 510.2, 510.4). Normal,
blocking-band and attacking-band paths use this collection. This change keeps
the existing combat assignment and band source-attribution behavior; it does
not establish a new banding assignment contract. All damage and lifelink in the
batch precede damage events and notifications. First-strike and normal steps
remain separate batches. Single-damage callers retain their existing API.

Evidence: `TestPreventionAllocationSplit`, `APNAP`, `Rechecks`,
`BoundsAndCapacity`, `IdentityAndClone`, `OneSource`, `RedirectIntoShield`,
`CombatStepBatch`, `Priority`, `ChooseLaterPacket`, `SourceIncarnations`, and `LargeAmounts`. These cover split
prevention and source outcomes, both active players, replacement order and
rechecks, later-packet choices, invalid answers, spare capacity, nonpositive
packets, control versus ownership, departure/return, phasing, clone isolation,
retained lifelink, native combat batches and cleanup. Retained
`TestTypedPrevention*` tests cover declaration validation and resolution-time
amounts. `TestRONPreventionAllocationCombat` uses the complete Savannah Lions
and War Mammoth records in native combat; `SpellRollbackAndClone` uses the
complete Hurricane record with announced X, payment rollback, pending clones
and independent shield consumption. `KeepsDeclarationsStrict` proves that the
engine prerequisite does not authorize unsupported RON declarations.

No finite-prevention RON action is enabled by this slice. In particular,
`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(...),
[PreventThatDamage])` remains rejected. Source-specific shields, additional
prevention operations and full finite-prevention card gameplay mappings require
separate adapter and contract evidence.

## Incoming damage to attacking bands: separate sources

`BandingDamageAssigner.GetBandingDamageAssignment(g *Game, source *Permanent,
recipients []*Permanent, amount int) map[uuid.UUID]int` is an optional player
service for the incoming side of ordinary attacking bands. It runs once for
each positive blocking source that assigns damage in the current damage step.
Each map must contain only eligible recipient IDs, nonnegative amounts and the
source's exact damage total. Validation subtracts from the available amount to
avoid overflow. Nil and invalid maps use `Player.ChooseNumber`, with bounds that
assign the complete source amount. Invalid number answers use the legal maximum;
forced choices need no prompt. Assignment makes no payments or zone changes.

While a current member has banding, the attacking player chooses the incoming
assignments. After the members lose banding, each blocking source's controller
chooses instead (CR 510.1d, 702.22e/g/k). Formation alone does not preserve the
assignment permission. The existing `BandingDamageDistributor` signature remains
valid: when no source-aware service exists, one valid aggregate recipient map
sets the recipient totals for the step. Bounded source choices divide those
totals into individual source packets, without changing the caller's map.
Invalid aggregate maps use unrestricted per-source number choices. An aggregate
total that exceeds the integer range also uses the per-source path.

`Combat.AddBlockerIncarnation` records each blocker for all members of the
attacking band (CR 702.22h). Duplicate declarations do not add duplicate packets.
Removing the originally named member cannot erase surviving members' blocking
relationships. Departure, return and phasing remove the object from combat;
phase-in and a new incarnation do not restore its membership (CR 400.7, 506.4,
702.22f, 702.26). Current attack incarnations limit eligible recipients.

Every blocking source retains its own amount, ID, incarnation, controller,
owner, characteristics and keywords in `DamageSource`. Zero and negative source
amounts are excluded individually (CR 510.1a). First strike and double strike
use the existing step service. All packets join the existing combat damage
batch before replacement choices. Finite prevention can therefore distinguish
sources (CR 615.7); redirection, rechecks and priority remain in the existing
replacement pipeline (CR 616.1). Actual damage supplies source histories and
lifelink; notifications follow the complete damage batch. A replacement-time
source departure or return cannot replace saved source information. Game clones
keep their own combat, shield and damage state.

Evidence: `TestBandIncomingDamageSources`, `IndependentAssignments`,
`InvalidAssignments`, `LegacyTotals`, `Steps`, `IdentityAndClone`,
`RetainedSource`, `ReplacementOrder`, `Unpreventable` and `NonpositiveSources`.
`TestRONBandIncomingDamage` casts complete Benalish Hero, Timber Wolves and
War Elephant records, then uses complete Savannah Lions and War Mammoth records
as blockers. It checks unpaid rollback, casting timing, band metadata, native
combat, split shields, both source histories and clone isolation. It adds
incoming-combat evidence rather than complete-card gameplay certification.

This contract does not change outgoing attacking-band assignment, blocking-band
assignment, "bands with other", or the old lethal-order policy for other combat
paths. Outgoing aggregation and shared trample/deathtouch handling remain open
migration prerequisites. The RON compiler accepts no new shape in this slice;
finite `CreateFuturePreventDamage` declarations remain rejected.

## RON finite recipient shields

The RON compiler now enables the fixed/X recipient-shield action documented in
[dsl-reference.md](dsl-reference.md#ron-finite-recipient-prevention). This
supersedes the earlier finite-shield prerequisite's statement that all
`CreateFuturePreventDamage` actions are rejected. No engine signature or
replacement policy changes. `PreventNextDamage` reads the typed recipient and
amount at resolution, then uses `AddPreventionShield`. Spells, mode targets,
activations, triggers and delayed bodies retain their existing scoped bindings.

`TestRONFinitePreventionModes` covers both complete Healing Salve and Alabaster
Potion modes, X including zero, payment rollback, player/permanent recipients,
resolution timing and pending clones. `Activations` covers eight complete
records, mana/tap/sickness legality, source departure, control and copies,
independent shield capacity, clones and native cleanup. `ActivationTargets`
covers current target restrictions and resolution rechecks. `Identity` covers
shroud, return, phase cycles, control and unpreventable damage. `RetargetAndInstances`
covers separate factory/copy recipients. `RakaliteDelay` covers multiple
one-shot delays, owner return, phasing, changed control, new incarnations and
pending clones. `BatchAndCombat` covers native non-band combat, allocation,
retained lifelink and both replacement orders. `ResolvingContexts` covers
trigger controllers and delayed X/recipient capture. Invalid declarations and
semantic inventory roles are tested separately.

This is compiler and RON gameplay evidence for the listed contracts. The
outgoing attacking-band assignment gap remains. Collection-wide gameplay,
production loading and runtime cutover are not established by these tests.

## Single-card library search ordering and legality

`SearchLibraryToHand` requires a card from a nonempty library (CR 701.23d).
`SearchLibraryToBattlefield` permits failure to find with a nonzero quality
filter (CR 701.23b); its zero filter requires a card. Both validate the returned
choice against candidates by ID and use the canonical candidate object. A nil
or foreign choice uses the first candidate for a required search and finds
nothing for a qualified search. Empty libraries do not prompt for a card.

Both services shuffle once even if they find nothing. A found card moves to
hand or enters the battlefield before the shuffle. This ordering implements
the instruction sequence of the supported RON records (CR 608.2c). A failed
`PutOnBattlefieldBy` restores the library card before shuffling, so an entry
prohibition cannot remove it from the game. Normal entry uses the shared entry
pipeline, including tapped entry and counter replacement. The search does not
cast or play the found card and does not use the land-play allowance. No hand
reveal is added (CR 701.23e).

`TestLibrarySearchRequiredFindAndOrder`, `TestLibrarySearchQualifiedCanFail` and
`TestLibrarySearchProhibitedEntryKeepsCard` cover required/optional selection,
foreign decisions, empty libraries, both destinations and shuffle order.
`TestRONLibrarySearchCards` executes all six Demonic Tutor reprints and the
complete Untamed Wilds record, including payment, hidden information and
resolution-time library membership. `TestRONLibrarySearchCopiesAndEntry`
covers copied controllers and pending clones. `NormalEntryRules` combines
tapped entry and doubled entry counters before shuffle. `ActivationKeepsController`
checks departure, return, phasing and control changes after activation.

These services do not establish multiple-card searches, reveal/result bindings,
restricted search zones or new replacement types. Strict collection loading,
collection-wide gameplay coverage and runtime cutover remain unfinished.

## Outgoing damage from ordinary attacking bands: separate sources

The outgoing attacking-band path now saves each current member's positive damage
amount, source incarnation, controller and damage information before replacement
choices (CR 510.1a, 510.2, 702.22g). Each packet uses that source's own keywords.
The band does not share trample, deathtouch or lifelink. Returned, phased and
removed members cannot assign damage; a phase cycle does not restore combat
membership. First strike and double strike use the existing step service.

`CombatDamageAssigner.GetCombatDamageAssignment` keeps its signature and receives
one member and that member's amount per call. Outgoing assignments have no lethal
order (CR 510.1c). A current blocker with ordinary banding instead gives the
choice to the defending player (CR 702.22j), through `BandingDamageAssigner`,
then the existing `BandingDamageDistributor`. A missing or invalid nontrample
blocking-band answer uses bounded number choices. The latter interface now
receives separate source requests on this path; its signature stays unchanged.
Incoming aggregate recipient requests retain their prior contract.

An assignment map contains only eligible blocker IDs and nonnegative amounts.
Validation subtracts from the source amount to prevent integer overflow. A
nontrample source must assign its complete amount. Only a source with trample
may leave a remainder for the defender. The complete band assignment must supply
lethal damage to every blocker before any remainder can go to the defender
(CR 702.19b). Marked damage and assignments from other band members count;
positive damage assigned by a source with deathtouch supplies lethal damage.
Prevention does not change this check. Caller maps are copied before another
source is queried. Invalid complete plans use a legal default plan: nontrample
sources first, then trample sources, retaining shared lethal assignments. No
payment, resource expenditure or live damage occurs during assignment.

A blocked band with no remaining blockers assigns only each trampling member's
own amount to the defender (CR 510.1c, 702.19d). All outgoing and incoming packets
join the existing simultaneous combat batch. Prevention allocation, single-use
redirection, affected-player replacement order, rechecks, actual damage histories,
lifelink and notifications use the shared damage pipeline. Replacement-time
source departure and return cannot change captured source information. Pending
clones retain separate combat and shield state.

Evidence: `TestBandOutgoingDamageSources`, `TrampleIsPerSource`, `SharedLethal`,
`InvalidAssignments`, `BlockingBandChooser`, `Steps`, `IdentityAndClone`,
`RetainedSource`, `IndependentAssignments`, `ReplacementOrder`, `Unpreventable`,
`LegacyDefendingChoices` and `KeywordsStayWithSource`. `TestRONBandOutgoingDamage`
casts full Benalish Hero records with Timber Wolves, Mesa Pegasus, War Elephant
or Icatian Phalanx, and activates a full Amulet of Kroog record to protect a
RON War Mammoth blocker. It checks unpaid rollback, casting/activation timing,
source histories, finite prevention and native cloned combat. Malformed banding
rules and incompatible contexts still fail compilation.

This section supersedes the outgoing aggregation gap recorded in the incoming
band contract above. It establishes the ordinary attacking-band path. The
separate non-band and blocking-band combat paths, their old lethal-order policy,
"bands with other", and multi-blocker damage choices outside this path are not
changed. No new RON syntax is accepted and no collection loading gate changes.

## Normal nontrample attacking damage assignments

For a blocked, non-band attacker without trample, CR 510.1c permits any division
of its complete positive damage amount among current blockers. There is no
lethal order. With two or more eligible blockers, the current controller's
`CombatDamageAssigner.GetCombatDamageAssignment` receives that source, the
blockers, and its amount. `GetBlockerOrder` is not called on this path. A single
blocker receives the complete amount without a choice. Zero or negative amounts
make no assignment and require no choice. No remaining blocker means no damage
to the defender. Existing damage-equal-to-toughness and damage-step services
still determine the source amount and first/double-strike timing.

Maps may contain only eligible blocker IDs and nonnegative amounts whose sum
is exactly the source amount. The shared map validator subtracts available
amounts to avoid overflow. Missing or invalid maps use the existing legal
default distribution. Each map is consumed into separate immutable damage
assignments before another source is queried; caller maps remain unchanged.
No payment or live damage occurs during assignment.

Source and blocker incarnations must match combat membership. Departure,
control changes, combat removal and phasing remove membership; a return or
phase-in does not restore it. Legacy nil-incarnation combat declarations retain
their compatibility. Damage sources are captured before replacement choices.
Replacement-time source departure and return cannot change damage attribution,
keywords or lifelink's beneficiary. All packets enter the existing simultaneous
combat batch; prevention allocation, redirection order, damage histories,
notifications and cloned state use the shared pipeline (CR 510.2, 615.7, 616.1).

Evidence: `TestNontrampleAssignmentFreeDivision`, `Invalid`,
`TimingAndIdentity`, `BatchAndRetainedSource`, `ReplacementOrder`, and
`Deathtouch`. `TestRONNontrampleAssignment` casts complete Craw Wurm, Hill Giant,
Savannah Lions, Grizzly Bears, Serra Angel and Air Elemental records, then uses
Wall of Air blockers and an activated Amulet of Kroog shield. It checks unpaid
rollback, payment, timing, flying/vigilance interactions, free assignment,
actual prevention, histories and independent cloned combat. Malformed keywords,
costs and incompatible target references remain compile errors in
`TestRONNontrampleAssignmentDeclarations`.

This establishes the normal nontrample attacking assignment context. Normal
trample's existing lethal-order policy, blocking-band decisions, and assignment
by a blocker to multiple attackers remain separate contracts. No RON syntax,
production registry, casting permission or banding policy changes.

## RON choose-one activated mode adapter

`Activated` and `ActivatedWithModifiers` with `Modal_ChooseOne` reuse
`NewModalActivated`, activation transactions, per-mode target validation, payment
snapshots and stack copying. The compiler shares mode-plan construction with
spells. Each factory builds separate targets in each mode. This establishes no
new engine service. Supported modifiers remain on the complete activation.

`TestRONModalActivationBalm` executes the complete FEM record: artifact casting,
both modes, mana/tap/sacrifice timing, failed payment and illegal-target rollback,
source departure, target shroud/phasing/return, phase cycles, copied retargeting,
pending clones, shield consumption and cleanup. `BalmSimultaneous` checks the
finite amount across simultaneous sources after recipient control and phasing
changes, plus independent cloned shields. `Declarations` rejects malformed,
incompatible and cross-mode reads and checks independent factory/mode targets,
mode labels and once-per-turn modifiers. `PaymentBindings` checks announced X,
sacrifice characteristics after return, source departure, zero-cost declarations
and cloned pending resolution. `CounterBindings` checks two pending instances'
separate counter payments. `StrictLoad` checks the complete accepted record,
semantic inventory and collection rejection with an invalid mode.

Rules: CR 602.2a-b, 601.2b-c/f-h, 700.2a/c/f/g, 608.2b/h, 400.7,
702.26b/d, 615.6-7 and 514.2. Existing finite-prevention ordering and allocation
tests remain applicable; the adapter does not create another replacement path.
Pyramids still needs its destruction replacement; its attached-land filter
is already supported.
Triassic Egg still needs entry/graveyard operations, its counter condition and
correction of its second mode. Neither card gains complete behavior coverage.

## Resolution mana payment and spell mana-value information

`UnlessManaPaid(player PlayerRef, cost ManaCost, amount ResolutionValue,
body ResolvingEffect)` and `IfManaPaid` offer a complete mana payment at
resolution. Definition validation requires all typed operands and a nonnegative
mana template containing X. Each X uses the same amount, evaluated once before
payment choices or mana abilities; nonpositive dynamic amounts count as zero.
The immutable declaration copies hybrid symbols. Existing `unlessPaidEffect`
and the total-cost transaction own the choice, affordability, mana solver,
payment and conditional body. Decline or failed payment spends no resources;
zero mana requires acknowledgment (CR 118.3, 118.5, 118.12, 608.2g–h).
An absent player has no effect. Activation reductions do not apply to resolution
payments, including existing fixed `IfPaid` and `UnlessPaid` mana payments.
Their affordability check now agrees with the transaction's locked cost.

`ManaValueOfSpellWithLKI` reads current stack mana cost and announced X (CR 202.3).
For a departed typed target or captured spell it reads the value retained by
stack object ID (CR 608.2h). Pop, counter/removal by ID or source, and stack-to-ante
removal all save the departure value. Retained information does not restore
stack membership or allow countering the departed spell. A new cast uses a new
stack object ID. The stack's value map is cloned independently. This contract
adds mana-value information only. The existing `ManaValueOfSpell` keeps its
zero-on-departure contract. This does not add general spell LKI, a saved
controller query, or delayed spell-event bindings.

Evidence: `TestResolutionManaPayment*` checks costs, colored/multiple X,
zero acceptance, affordability, mana production, independent outcomes, clone
isolation, amount capture and transaction rollback. `TestSpellManaValue*` checks
live changes, all removal paths, recasting and cloned departure information.
`TestRONResolutionMana*` checks lowering, typed event/target identity and actual
RON gameplay. This slice adds no replacement or simultaneous-operation service;
those paths retain the shared payment and mana contracts.

### Player-scoped mana denial and retained spell controllers

`LoseUnspentManaFor` clears the selected player's current pool (CR 106.4),
including restricted mana. It causes no life loss, mana activation, mana
production or change to another pool. `ManaPool.Clear` owns the operation;
production history remains available. Typed player reads and engine definition
validation retain the existing declaration rules.

`HasManaAbility` reads current runtime abilities. It reuses
`IsActivatedManaAbility` and the triggered classification shared with immediate
mana-trigger resolution (CR 605.1–2). It does not test whether a cost can be
paid or whether the ability currently produces mana. Ability removal and grants
change membership; controlled battlefield collections exclude phased objects.

`ControllerOfSpellWithLKI` reads current stack control, then saved departure
control (CR 608.2h). Stack removal records the controller under the unique stack
object ID; pop, ID removal, source removal and ante removal share that service.
A recast uses another ID. Cloned stacks own separate departure maps. The existing
`ControllerOfSpell` still fails to read an absent spell. Retained information
does not grant permission to counter an absent spell or make an illegal target
legal.

RON `PlayerActions` lowers to `SnapshotPlayer` and an ordered `Sequence` with
per-instance, nested player bindings. Its player reads cannot escape the body.
Tap collections use `PermanentsControlledBy`, `MatchingPermanents` and `TapEach`;
all selected taps finish before their events (CR 608.2f). Tapping this way does
not activate mana abilities. Power Sink's unpaid sequence continues after an
unsuccessful counter attempt (CR 608.2c); an all-illegal target prevents the
entire spell from resolving (CR 608.2b). Its X and optional payment retain the
existing announced value and payment transaction.

Evidence: `TestLoseUnspentManaFor`, `TestHasManaAbilityFilter`,
`TestSpellControllerWithLKIDeparture`, `TestSpellControllerWithLKIRemovalPaths`
and `TestRONManaDenial*` cover typed declarations, pool scope and history,
current membership, restrictions, payment branches, departure information,
recasts, phasing, copies, clones and real RON casting. Existing
`TestTapEachTransitionsTogether`, `TestTapEachCloneAndDeclaration`,
`TestTappedForManaTrigger*` and `TestSpellManaValue*` retain shared batch,
classification and compatibility evidence. No new replacement or damage
contract is established here.

## Live static combat prevention and subject attack restriction

`PreventCombatDamageFrom(subject PermanentRef) StaticRule` prevents outgoing
combat damage to any recipient. `PreventCombatDamageTo(subject PermanentRef,
from PermanentFilter) StaticRule` prevents incoming matching combat damage.
A zero source filter permits all sources. Both use the current subject under
the static ability's source context. `This` and `AttachedPermanent` have no
announcement choices. Nil and undeclared references fail definition validation.

Each ability instance installs its own cycle replacement. Each proposed event
checks the current source incarnation and resolves its current subject again.
This check is required because phasing need not start another layer cycle.
No previous attachment, returned source, or returned host can replace that
identity. Static prevention stops while its source is absent or phased out and
returns with its active ability. Recipient ability removal cannot erase an
Aura's separate rule. Source ability removal does erase that rule. Source
filters reuse current/LKI damage-source matching. `DamageSourceIsEnchanted()`
reads current Aura status while the source is live, and the saved `Enchanted`
fact from `PermanentLKI` after departure or phasing. Snapshot capture uses the
same Aura query as the unchanged live `IsEnchanted` filter. A returned source
uses its own current status; it cannot overwrite the earlier incarnation.
The shared pipeline handles simultaneous packets, affected-player replacement
choices, rechecks after redirection, and damage that cannot be prevented.
There is no shield consumption and no noncombat prevention.

`CantAttack(subject PermanentRef) StaticRule` lowers to `CantAttackEach` through
the shared single-reference set. It checks current subject identity at attack
declaration, remains separate from recipient abilities, and does not remove
existing attackers. Existing collection callers retain their behavior.

RON owns spelling, arity, static contexts and source/host reference restrictions.
Its union requires one incoming and one outgoing event for the same subject.
The filtered incoming event additionally requires a creature damage source.
Demonic Torment, Gaseous Form and Enchanted Being are complete RON consumers.
`TestStaticCombatPreventionLiveHost`, `TestStaticCombatPreventionFilterAndDeclarations`
`TestStaticCombatPreventionEnchantedSourceInformation`
and `TestStaticCombatPreventionSimultaneousAndUnpreventable` establish engine
identity, live filters, source/host ability removal, phasing, clones, repeated
and simultaneous packets, noncombat exclusions and invalid declarations.
`TestRONStaticCombatPreventionCards`, `Casting`, `Lifecycle`, `Invalid`,
`Ordering`, `StrictInventory` and `IllegalAuraResolution` (with the common
`TestRONStaticCombatPrevention` prefix) establish complete record behavior,
casting costs and rollback, target legality, attachment changes, native combat,
factory/clone separation and strict loading.
Rules: CR 120.2, 400.7, 508.1, 611.3a–b, 615.1–6, 615.12, 616.1/616.1f,
609.7c, and 702.26b–i. No timing, payment, damage allocation or replacement ordering
algorithm is changed.


## Live identity for static spell-cost reducers

`ReduceSpellCostStatic` and `ReduceSpellCostStaticLabeled` register the source
incarnation with each reducer. A cost query or payment ignores that entry when
the source is absent, phased out, or has a different incarnation. Phase-in can
restore the same entry before another layer pass. A returned card cannot restore
its old entry. The controller must match the current source and the registered
player. Normal layer passes update registration after control or ability changes.
Explicit `GameRules.AddSpellCostReducer` entries without a captured incarnation
keep their existing behavior.

This establishes CR 400.7, 601.2f–h, 611.3a–b and 702.26b for static generic
spell-cost reductions. It adds no new cost-payment or replacement system.
`TestSpellCostReducerSourceIdentity` covers phase-out/in without a layer pass,
source departure/return, controller identity and clone isolation.
`TestSpellCostReducerPhasedCastingRollback` covers native casting, rejected
payment rollback and restoration after phase-in. RON tests cover source ability
removal, control changes, independent sources, cost increases, X and added costs.

## Fixed-count sacrifice payment

`SacrificeNMatchingIncludingSourceCost(n, filter, text) Cost` permits a matching
source among n distinct phased-in permanents under the payer's current control.
The count must be positive. `SacrificeNMatchingCost` retains its source exclusion.
Definition validation now visits ordinary matching sacrifice costs as well as
costs with captured results. Unpayable counts fail before gameplay.

Matching sacrifice costs choose every permanent before any leaves. The payment
transaction saves each ID and incarnation, with no new choice on commit. A stale
incarnation, control change, or phasing fails the payment. Count costs use the
existing transaction preflight before spending mana or other resources. A second
invalid choice leaves all live resources unchanged (CR 118.3, 601.2h, 602.2b).

One matching count cost uses `withZoneChangeBatch`. Zone-change and sacrifice
events observe the completed departure and detect battlefield abilities against
the prior battlefield (CR 603.6c, 603.10). Departing observers can see every
member of the same batch. All choices see the same battlefield, so sacrificing
a static source cannot change eligibility during selection. Separate cost
objects are separate payments. Sacrifice goes to the owner and bypasses
indestructible and regeneration (CR 701.21a). The existing sacrifice path and
LKI service remain responsible for departure information; this slice introduces
no new replacement operation, result slot, or sacrifice replacement contract.

Evidence: `TestSacrificeCountIncludingSourceBatch`, `Rollback`, `Invalid`, and
`Activation`, plus existing `TestSacrificePayment*` and RON sacrifice regressions.
RON mappings: `TestRONSacrificeNumberWarrens` covers complete Goblin Warrens
payment, token characteristics, copies and pending clones; `MoldDemon` covers
complete optional entry payment and original/returned source identity;
`Casting` covers native casting and entry. `SelfAndExplicitPlayer` covers
source inclusion, recursive filters and separate payment results. `Invalid`
and `Inventory` cover rejected declarations and semantic reporting.
`TestRONEnteringSourceElectricEel` covers the additional complete consumer's
entry damage, source departure/return/phasing/control, prevention, trigger copy,
clone isolation, paid boost and cleanup. `TestRONEnteringSourceScopeRejected`
keeps the entry reference local to the supported self-entry event.

## Untap limits and simultaneous selection

`LimitUntapsDuringStep(filter PermanentFilter, maximum int) StaticRule` limits
matching permanents for each player's untap step. Definition validation rejects
negative limits. A zero filter matches all permanents; zero is a valid limit.
Printed source presence, phasing and ability suppression use the shared effect
lifecycle. Recipient ability removal does not remove this rule.

After phase-in and final layer evaluation, the engine saves each limit's members
and selects the active player's eligible tapped permanents. Skip-next-untap,
captured restrictions and other untap prohibitions exclude candidates. The player
can decline `ChooseMayAbility` to stop, then selects each object through
`ChoosePermanent`. Nil also stops; an invalid object uses the first legal
candidate. Each selected object consumes every applicable quota. Artifact lands
and artifact creatures do not bypass a second limit. Existing land, artifact
and creature limit fields use this same selection service; their callers remain.

All selections precede `untapPermanentsTogether` (CR 502.3). Stun replacement
removes a counter from a selected object without backfilling the selection.
Untap observers see the completed operation. Membership cannot change during
that operation; the next step makes a new selection. Other untap effects do not
use these limits. Each decision uses local state; effect clones own their source
metadata. No LKI is used for a departed static source.

Evidence: `TestUntapLimitsChoiceAndOverlap`, `TestUntapLimitsSnapshotAndSimultaneous`,
`TestUntapLimitInvalid`, and real-RON `TestRONUntapLimitsRealCards`, `Casting`,
`Reprints`, `Invalid` and `Inventory`. Smoke and Damping Field have complete
record mappings. No new damage replacement, cost, trigger or APNAP contract is
established.

## Private library-top inspection and optional shuffle

`LookAtLibraryTop(viewer, library PlayerRef, count int, reorder bool)` reads the
current top `min(count, library size)` cards during resolution (CR 401.2,
701.20e, 608.2c). It records independent private snapshots before any ordering
choice. Only the viewer receives `LibraryLookObserver` notifications. The library
owner receives no notification unless that player is the viewer. Public reveal
history is unchanged. `Game.LibraryLooks(viewerID)` returns copies of the past
observations, including copied card definitions and type slices; it does not
expose later library changes. Games clone all observation maps, slices and cards.

When reordering is permitted, the viewer supplies a permutation of the inspected
cards through `LibraryTopOrderChooser`. Existing players can use the scry choice
service; its two partitions are joined into one top order. Unknown, duplicate,
missing or extra IDs leave the library order unchanged. All inspected cards stay
on top, and the uninspected suffix stays in the same order. The engine does not
remove/reinsert cards, change their zone identities, or emit zone-change events.
No order decision is made for an empty inspection. A negative count or invalid
player declaration fails definition validation. An absent resolved player causes
no inspection or choice.

`MayShuffleLibrary(chooser, library PlayerRef)` reads the selected player at
resolution. The chooser decides after preceding instructions. Acceptance calls
the existing `Player.ShuffleLibrary`, including for zero or one cards
(CR 701.24a/e). Decline preserves order. Player ownership and stack controller
are distinct: a copied spell or activation uses its copy controller as viewer
and chooser. Departure, return, control change or phasing of an activation source
does not redirect the queued library instruction. No permanent LKI, replacement
operation, simultaneous zone movement or cost-payment contract is added.

HumanPlayer uses private display and top-order choices; the existing Player
interface and shuffle callers remain valid. Evidence: engine
`TestLibraryInspectionPrivacyOrderAndClone`, `BoundsInvalidOrderAndShuffle`,
`DeclarationsAndMissingPlayers`; RON `TestRONLibraryInspectionCards`, `Reprints`,
`Invalid`, `InstancesAndInventory`, `SpyCastingAndSickness`, `RetargetAndControl`,
`SorceryTiming`; interactive `TestHumanLibraryInspectionAndOrder`.
`TestLibraryInspectionOrderValidation` also covers duplicate, missing, extra,
and unknown IDs without loss of library cards.

## Fixed RON mill actions and resolution costs

`MillNumberCards(Integer(n))` lowers to `Mill(player, Fixed(n))` in resolving
player scopes. Its optional resolution-cost context lowers to `MillCost(n)`
inside `IfPaid` or `UnlessPaid`. The compiler checks arity, integer shape, scope,
nonnegative action counts and positive declared cost counts before factories
run. No announced X, action-result, all-player or activation-cost mill adapter
is established here.

CR 701.17a-b distinguishes milling as much as possible from full cost payment.
CR 118.3 and 118.12a require a complete optional payment at resolution. Existing
mill modifiers adjust the count before affordability and execution. A modified
zero cost can be accepted without moving cards. Independent adjacent payment
continuations retain separate outcomes. The legacy modifier service retains
its registration order; this slice adds no CR 616 replacement-choice policy.

CR 404.3 permits the owner to order cards placed together into their graveyard.
The shared `millCards` service now selects the full top-of-library batch, asks
`ChooseCardFromLibrary` for each next graveyard card, then removes the complete
batch from the library before appending the chosen order. Prior graveyard cards
retain their order. Candidate cards are copies that keep their IDs. Nil or
foreign selections use the first remaining card. The existing player interface
supports native human choices; `BasePlayer` retains library order. Pure
`CanPay` queries and isolated payment validation do not call the live chooser.
Commit checks affordability again before resources or order choices are used.

CR 603.3a keeps Deep Spawn's original trigger controller as payer. A copied
trigger uses the copy controller. CR 701.21a prevents the old controller from
sacrificing a source whose control changed. CR 400.7 and 702.26 preserve the
existing source-incarnation and phasing rules. Its other ability adds temporary
shroud, suppresses its next controller untap, then taps the source at resolution.

Evidence: engine `TestMillOwnerOrderAndBatch`,
`TestMillCostOrderQueriesAndRollback`, `TestMillCostOrderStaleRecheck`; RON
`TestRONMillstone`, `TestRONDeepSpawnPayment`, `TestRONDeepSpawnUnpaidIdentity`,
`TestRONDeepSpawnShroudUntap`, `TestRONMillModifiersAndPaymentFrames`,
`TestRONDeepSpawnNativeUpkeep`, `TestRONMillRealCardOwnerOrder`,
`TestRONMillDeclarations`, `TestRONMillScopesAndInstances`, and
`TestRONMillStrictLoadAndInventory`. Existing mill modifier, payment, source
lifetime, target and untap regressions remain in the required suites.

## Announced X in mana value target restrictions

`TargetManaValueEqualX()` restricts a typed permanent target to its current mana
value. X in the target permanent's mana cost is zero (CR 202.3e).
`DeclareSpellTargetManaValueX(name, filters...)` restricts a typed spell target
by its current mana value on the stack, including that spell's own announced X
and each X symbol in its cost. The action's X and the target spell's X are
separate values (CR 107.3a, 202.3e).

Announcement validation, target prompts, random target choices, copy retargeting
and resolution legality supply the relevant action's X explicitly. A query uses
a separate game view with its own X field; it cannot overwrite the current
resolution's X. Resolution checks each object's stored X before effect execution
begins. The existing target, incarnation, stack identity, transaction, copy and
clone services retain ownership of choices and state (CR 601.2b-c, 608.2b).
Target restrictions read current characteristics and do not use target LKI.

RON supports `ManaValueIs(EqualTo(ValueX))` as a direct permanent or spell target
restriction and within its supported `And` conjunction. The scope must supply
announced X. Static filters, disjunctions, fixed amounts, other comparisons and
combined spell mana value/current-permanent-target restrictions remain rejected.
Detonate uses existing controller LKI after destruction; failed destruction does
not suppress its damage. Spell Blast uses the target stack object's identity,
so a returned card or another spell copy cannot replace it.

Evidence: `TestTargetManaValueXQueries`, `AnnouncementAndCopy`,
`TestRONDetonateManaValueX`, `TestRONSpellBlastManaValueX`,
`TestRONManaValueXRetargetAndInstances`, `IndependentPendingCounters`,
`Declarations`, and `StrictLoadAndInventory`.

## Singular typed graveyard exile

`ExileGraveyardCard(CardRef) ResolvingEffect` resolves the supplied reference and
moves that card from its graveyard to face-up exile. It uses
`MoveFromAnyGraveyard` and `Game.ExileCard`, preserves the owner, and records the
effect source as `ExiledBy`. Target and saved-card references retain their
graveyard visit. Absent cards, later visits, and references to hand or exile
cards cause no move. Definition validation rejects nil and unbound references.
Graveyard departure events use the existing player attribution and count of one.
This action adds no batch or replacement ordering contract.

RON source-death bodies use the existing destination capture in
`PermanentDies().WithCard` with `EventSourceIsSelf`. This retains the first
graveyard visit before trigger placement and uses departure abilities and
controller information. Pending triggers, copied abilities and cloned games
retain independent captures. Phasing and a replaced graveyard destination do
not produce death triggers. Rules: CR 400.7e, 603.6c, 608.2b and 701.13a.

Evidence: `TestExileGraveyardCardVisit`,
`TestExileGraveyardCardDeclarations`, `TestExileGraveyardCardDepartureEvents`,
`TestRONGraveRobbersExile`, `TestRONGraveRobbersRollback`,
`TestRONGraveRobbersRetargetedCopy`, `TestRONCyclopeanMummyDeathCard`,
`TestRONGraveyardExileIndependentDeaths` and `TestRONDeathCardReturnUsesFirstVisit`.

## RON fixed landwalk declarations and keyword queries

The compiler now reuses `LegendaryLandwalk`, `HasKeywordFilter`, `Not`, typed
permanent targets and `Destroy`. No engine execution behavior changes.
Legendary landwalk requires a legendary land controlled by the defender;
a legendary nonland does not qualify (CR 702.14c). Phased lands do not qualify.
Printed first strike remains a separate ability (CR 702.7).

`TestRONFixedLandwalkLivonya` covers current land control, departure, phasing,
ability removal/restoration and clones. `TestRONFixedLandwalkNativeCombat`
checks actual block declaration and first-strike combat with the real record.
`TestRONFixedLandwalkAssassin` checks keyword legality at announcement and
resolution (CR 608.2b), permanent incarnation changes (CR 400.7), source
independence, phasing, clones, indestructibility and regeneration (CR 701.8,
701.19). `AssassinRollback` checks failed activation resource preservation.
`InstancesAndRetargetedCopy` checks independent activations and copied targets.
`Filters` checks positive and negative current keyword predicates for all seven
fixed variants. `GrantRemoval` covers legendary static grants and temporary
grant/removal expiry through native cleanup. Existing replacement ordering and
simultaneous-operation contracts are reused; this slice establishes no new
ordering or batching behavior. Legendary blocking overrides remain rejected.

## Opponent-or-planeswalker targets and rounded RON damage

`DeclareOpponentOrPlaneswalkerTarget(name string) *DamageTargetBinding` is a
single typed damage target. Its player candidates exclude the action's current
controller. Its permanent candidates include either player's planeswalkers,
subject to existing targeting restrictions and phasing. A creature qualifies
only if it is also a planeswalker. Existing `DeclarePlayerOrPlaneswalkerTarget`
continues to include both players. Empty labels and missing target declarations
fail definition validation. No new player or permanent reference type is added.

Announcement and resolution use the same candidate service. Copies with a new
controller recheck retained player targets against that controller; choosing new
targets also uses that controller. Permanent control changes do not invalidate
the planeswalker branch. Target return invalidates the saved incarnation, while
a completed phase cycle preserves it. A sole illegal target suppresses the
whole action, including its nontargeted self damage (CR 400.7, 608.2b).

RON `HalfRoundedDown(value)` and `HalfRoundedUp(value)` lower recursive resolving
values through `DivideRoundDown(value, Fixed(2))` and `DivideRoundUp(value,
Fixed(2))`. Existing arithmetic handles signed values; nonpositive damage has
no event (CR 107.1, 120.8). This establishes no static, entry, cost or replacement
value adapter. Existing `DealDamagePackets` captures all amounts and recipients
before its simultaneous batch. Existing announcement X, source LKI, prevention,
lifelink, target binding, transactional payment and clone services still own
execution. No new replacement ordering or damage batching rule is introduced.

Evidence: `TestOpponentOrPlaneswalkerTarget` and `CastingAndCopies` establish
the engine declaration, legal candidates, rollback, copies and clones.
`TestRONRoundedDamageCards`, `EternalFlameResolutionCounts`, `TargetIdentity`,
`BansheeSourceAndCopies`, `EternalFlameCopies`, `Rollback`, `BansheeLKI`,
`IndependentInstances`, `Invalid`, `NestedValues` and `Inventory` execute the
complete Banshee and Eternal Flame records and the shared value shapes.
The existing `TestDivideValues*`, `TestDamagePackets*` and `TestRONDamagePackets*`
retain arithmetic validation and replacement/batch contracts.

## Source-conditioned captured control

`GainControlWhileSourceControlled(subject PermanentRef, requireTapped bool)
ResolvingEffect` creates a Layer 2 control effect for the resolving controller.
It uses typed reference resolution and the existing control condition and layer
reconciliation services. Definition validation requires a permanent source and a
bound, non-nil subject. The declaration contains no chosen object or game state.

The duration starts only while the original source is present, phased in and
controlled by the resolving controller, and tapped when required (CR 611.2b).
Source LKI cannot start this duration. A returned source cannot substitute for
the captured source. Each installed effect saves both source and target
incarnations and the resolving controller. Source departure, phasing or loss of
control ends it permanently. The tapped form also ends on actual untapping.
Ability removal and cleanup do not end an installed duration. Losing and then
regaining control, retapping, or phasing back in does not restart it.

Untap transition hooks expire these effects before untap notifications. The
simultaneous untap service expires all affected durations before notifications.
A stun replacement that prevents untapping preserves the duration. Phasing
expires the source duration before an immediate phase-in (CR 702.26f). Target
phasing suspends application and preserves a valid source duration; target
return cannot receive the old effect (CR 400.7, 611.2c). Cloned games copy the
captured identities and expiration state independently.

The separate printed permission to remain tapped stays optional after target
departure or phasing. This action does not register the legacy target-dependent
untap policy. `GainControl`, `ControlEffectSpec`, and legacy tap-maintained
callers retain their behavior. The existing control layer owns timestamp order,
control-change events, summoning sickness and prohibitions on changing control.
The adapter adds no replacement-ordering, APNAP or simultaneous zone contract.

Evidence: `TestGainControlWhileSourceControlled`,
`TestGainControlWhileSourceControlledStartAndValidation`, the existing
`TestControl*` suites, and `TestRONSourceControl*` with the full Aladdin,
Thrull Champion, Rubinia Soulsinger and Willow Satyr records.

## Combat timing history (RON migration slice)

`BeforeCombatStep(point PhaseStep, during bool) StateCondition` supports
`DeclareAttackers`, `DeclareBlockers`, and `CombatDamage`. Invalid points fail
engine definition validation. `during=false` reads the first combat boundary in
the turn; `during=true` requires a current combat and reads its boundary.
`TurnReader.CombatWindowOpen(PhaseStep, bool)` supplies pure
queries. These conditions grant no casting or activation permission.

TurnSystem owns the value-only history and copies it during game cloning.
SetTurn and IncrementTurn reset it. Entering attackers or blockers closes that
point even when no creature is declared. First-strike damage is the first damage
boundary when that step exists (CR 506.7a/b). Native execution does not enter a
skipped first-strike, blockers, or damage step. It checks current continuous
abilities before deciding whether the first-strike step exists. An empty combat
closes the blockers/damage windows when its attackers step ends (CR 506.7e).
EndCombat completion or transition out of combat records the phase boundary.
Later combats restore only current-combat windows (CR 506.7c/d/g).

TurnSchedule retains a consumed full combat-phase skip for the precombat-main
boundary. Both construction-time and execution-time skips close first-combat
windows before an inserted extra combat. Skipping only BeginCombat does not
skip the phase. Schedule clones retain this information independently.

The RON compiler adds four symbol casting predicates and propagates the explicit
IsDuringCombat condition across a nonempty nested And tree. Activation lowering
adds only IsBeforeCombatDamageStep. Shared casting and activation gates retain
payment rollback; stack copies and resolution do not repeat timing checks.
No target, replacement-order, source-LKI, or simultaneous-damage rule is changed.
Angus Mackenzie reuses existing global combat prevention through cleanup.

Evidence: TestCombatTimingHistory, SkippedPoints, NativeSkippedSteps,
SkippedFirstThenExtraCombat, NativeDamageSteps and SkippedBeginningStep;
TestRONCombatTimingCasting, ExtraCombat, Invalid, Angus and AngusLifetime.
The existing casting-window and global-combat-prevention suites retain their
identity, target legality, copy, clone, damage and cleanup contracts.

## Typed spell-or-permanent color targets

`DeclareSpellOrPermanentTarget` declares one target that can be either a spell
on the stack or a permanent on the battlefield. Spell candidates use stack
object IDs; abilities, players and physical spell-card IDs are excluded.
The declaration keeps no chosen state. `WithTargets` captures choices in each
stack frame. Native target validation checks zone, incarnation, phasing, shroud,
hexproof and protection at announcement and resolution. Copies can retarget
across both categories without a mode choice.

`SetObjectColor` reads that typed declaration and calls the existing
`Game.ChangeObjectColor` service only for its current legal object. A missing
stack object is not replaced by a later cast of the same physical card.
A permanent requires its captured incarnation and must be phased in when the
change starts. The installed color change follows that incarnation through
control changes and phasing and persists after cleanup or source departure.
CR 400.7a carries a permanent spell's changed color to its resolved permanent;
other zone changes end the change. CR 611.2a and 613 layer-5 timestamps retain
indefinite duration and order later color changes. Mana symbols and other
copiable characteristics do not change. Copies of a changed spell use their
copiable colors. No damage or replacement ordering service changes.

Evidence: `TestSetObjectColorDefinition`, `TestSetObjectColorIdentity`, existing
`TestChangeColorEffectChangesSpellAndCarriesToPermanent`,
`TestSpellColorOverrideCloneAndCopyIsolation`, and RON `TestRONLaceCards`,
`RetargetedCopy`, `Legality`, `OrderingAndProtection`, `Invalid`, and
`RegistryAndInventory`. The RON tests cover all five lace names and 30 records.

## Activation history read during resolution

`WithActivationHistory` opts one `ActionDefinition` into a game-owned turn
tracker. The key contains the declaration binding and original battlefield
incarnation. A successful payment records one activation before stack placement
or immediate mana resolution. A failed declaration, target choice or payment
records none. A source sacrificed as a cost keeps its original incarnation.
This tracker does not alter activation limits or existing callers.

`IfActivationsAtLeast` reads this history when its instruction executes. The
current activation and countered activations count. Multiple pending activations
can all see the same later count. A copied ability reads the original history;
copying does not activate it (CR 707.10). Source departure does not erase the
old count. A returned source has separate history (CR 400.7). Control changes
and phasing retain history (CR 702.26). `TurnTrackers.Clone` copies the map;
final cleanup clears it with other turn history (CR 514.3).

The RON adapter supports only the nonmodal stack context. Existing `AtNext`
captures the source incarnation and resolving controller (CR 603.7). A delay
created after an end step begins waits for a later end step. Each qualifying
resolution creates its own delay. A returned permanent cannot replace the
captured source. A phased source cannot be sacrificed when the delay resolves;
the one-shot delay then ends. A player who no longer controls the source cannot
sacrifice it (CR 701.17a).

`TestActivationHistory*` covers declaration checks, independent abilities,
source incarnation, clones, turn reset, countered activations, copies, payment
failure and sacrifice costs. `TestRONActivationHistory*` covers all seven dragon
records, native casting/payment, temporary bonuses, thresholds, pending and
countered activations, copies, delayed timing, source return/phasing/control,
strict registry rejection and semantic roles. Existing flying and banding engine
contracts remain in force. This change adds no replacement or simultaneous-action
service. The mana-conversion compiler context remains rejected; synchronous mana
execution and planning require their own complete integration.

## Mandatory resolution sacrifices and replacement destinations

`IfRequiredPaid` and `UnlessRequiredPaid` accept a nonempty group of matching
sacrifice costs. They use the total-cost transaction at resolution and do not
offer a may choice. The branch remains local to its payment. Definition
validation rejects other cost families and captured payment values. The service
preserves current optional-payment callers.

The payer controls each current, unphased payment permanent; its owner receives
the card unless a replacement changes the destination (CR 701.21a). A cost that
excludes the source excludes the resolving ability's original incarnation.
A returned source can therefore pay the old ability (CR 400.7). Copies keep the
source identity but use their own controller as payer. Source departure or
phasing does not cancel the pending payment. Failed full payments leave all
payment objects in place. Independent and nested payments do not overwrite a
result.

`DoSacrifice` now creates a `SacrificePermanentAction` before the move. The
existing replacement pipeline supplies priority, affected-controller choices,
cloning and rechecks (CR 616.1). The existing
`AddExileIfWouldGoToGraveyardThisTurn` service also matches this action.
The action retains the chosen incarnation. An exile destination still causes a
sacrifice event, including the source's LKI trigger, but does not cause a death
event or increase creature-death counts. If a replacement prevents the move,
a legal mandatory payment that started still takes the paid branch (CR 118.12).
Regeneration cannot replace the sacrifice. Matching multi-permanent sacrifice costs select all replacements before
any payment object moves. Existing batch services preserve the pre-move
battlefield and defer events until the sacrifice batch finishes.

Evidence: `TestRequiredPayment`, `NestedAndClone`, `Definition`,
`ReplacementEventsAndOrdering`, `ReplacementPreventsMove`, `SourceIdentity`, `SimultaneousSacrifices`, and
`SimultaneousReplacementSource`;
`TestRONRequiredPaymentLordOfThePit`, `Scopes`, `Invalid`,
`AllReprintsAndNativeCasting`, `InvalidChoice`, and `InventoryAndStrictLoad`.
These establish the mandatory matching-sacrifice context, not general mandatory
mana, destruction, tap payments, or saved resolution-payment values.

## RON live static P/T values and permanent spell copies

`PermanentLayerEffect(subject, [AdjustPTXY(PlusX, PlusY, x, y)])` and
`PermanentLayerEffect(subject, [AdjustPTForEach(power, toughness, count)])`
now lower to `Static(While(condition, ModifyPT(...)))`. The subject is the
current source or a declared Aura host. Each layer pass reads current battlefield
membership and the source's current controller (CR 611.3a, 611.3c).
Values support integer constants, `Twice`, signed half rounding, and battlefield
counts with type, subtype, color, supertype and combat/tap filters. A conjunction
can restrict the count to `ControlledByAPlayer(SinglePlayer(You))`.
The exact conjunction of `IsEnchantmentType(Aura)` and
`AttachedToPermanent(source-or-host)` uses `AttachedAuras` and keeps the current
host incarnation. Equipment and phased objects are excluded. Static P/T reads,
announced X, event results, collection modifications and combined operation lists
remain rejected by this adapter.

The bonus applies in layer 7c, after base values and before switching P/T.
Counters remain additive. Removing the source's abilities in layer 6 suppresses
its bonus. Removing the host's abilities does not remove a separate Aura's
bonus. Control, attachment and phase changes cause a fresh count; returned
objects do not keep old attachments. These static queries do not use LKI.
`TestRONStaticValue*` maps the complete Aspect of Wolf and Rabid Wombat records,
including reprints, casting rollback, resolution legality, controller changes,
phasing, attachment changes, clones, copied Auras and native combat.

`CopySpellOnStack` now uses `Card.CastTargets()` for nonmodal retargeting, so
Aura casting targets use the same target-group and incarnation checks as other
spells. Permanent spell copies enter through the normal battlefield path as
token permanents (CR 608.3f, 707.10f). Token status is visible before entry
replacements and events. Entry uses zero colors spent to cast the copy; the
original cast facts are preserved. No token-creation operation is invoked.
The token keeps the copied spell's characteristics and controller/owner, with
separate ability identities and static effect source state. Shared stack card
definitions are not changed, including when a cloned game resolves first.
Illegal or countered copies cease to exist; instant, sorcery and ability copies
keep their existing resolution behavior. `TestPermanentSpellCopy*` covers token
entry, Aura retargeting, original bindings, return/phasing/shroud legality,
countering, clone isolation, token departure, cast and entry facts, entry
counter replacements and prohibited entry. No new replacement ordering or
simultaneous-operation service is added.

## RON singular entry counters

`EntersWithACounter(counter)` in a single `AsPermanentEnters(ThisPermanent,
[operation])` lowers to `EntersWithNCounters(counter, 1)`. The compiler adds no
engine API or execution state. Entry counters pass through the shared counter
replacement pipeline before entry observers (CR 122.6, 614.1c). The entering
controller chooses replacement order (CR 616.1). Phasing keeps counters;
returning from another zone starts a new incarnation with a new entry counter
(CR 122.2, 400.7, 702.26d).

Evidence in `pkg/mage/dsl/ron/singular_entry_counter_test.go`:

- `Javelineers` executes its complete RON record: white casting payment, entry
  timing, summoning sickness, controller and any-target legality, tap/counter
  payment, rollback, exhaustion and player/creature/planeswalker/self damage.
- `Lifecycle` covers card copies, copied and cloned pending abilities, source
  departure, phase cycles, return and independent instances.
- `TargetRecheck` covers target return, phasing, shroud and damage prevention.
- `Replacements` checks add-then-double (4) and double-then-add (3), with the
  entering controller choosing. `Observer` checks counters before entry triggers.
- `Declarations` rejects malformed shapes, incompatible counter types, subjects
  and contexts; factory declarations stay separate. `CompleteRecords` executes
  exact entry clauses for all three consumers and retains complete-record
  rejection for Tidal Influence. Homarid now compiles with the source counter-state
  adapter below. `Inventory` checks descendant roles.

No new target, LKI, simultaneous-operation or cost contract is introduced.
General entry choices, multiple operations and counter-state triggers remain
outside this adapter. Existing counter storage limits remain unchanged.

## RON source counter-state cycles

The source-only `WhenAPermanentHasNumberCountersOfType` form lowers to
`WhenState(SourceHasCounters(counter, minimum), body)`. It uses the existing
per-source, per-ability, per-incarnation arming key. One instance can be pending
or on the stack at a time. Countering or resolving permits another trigger if
the condition still holds (CR 603.8). The condition checks current counters;
phased sources and sources without the ability cannot queue new instances.
It is not an intervening-if condition (CR 603.4).

For this source-only shape, `Trigger_ThatPermanent` uses `This` and the original
source reference captured by the engine when the trigger queues. Control changes
cannot redirect it. Departure and return cannot substitute a new incarnation
(CR 400.7). Reset does not use LKI counters: an absent or phased subject is not
modified. A phase cycle before resolution retains the original incarnation.
Copied abilities retain that source even when their controller differs.

`AllCountersOfTypeFromPermanent` lowers to `RemoveAllCounters` on the compiler's
typed permanent reference. Its count is read at resolution, including counters
added after the trigger queued. Different counter kinds remain. Ordinary target
legality and Aura attachment reference contracts still apply. No new payment,
replacement, simultaneous-operation or zone service is introduced. Entry
counter replacements can satisfy the state condition before entry observers.

Evidence in `counter_state_test.go`:

- `Homarid`: complete real RON casting, payment rollback, entry, exact P/T states,
  own versus opponent upkeep and two native counter cycles.
- `Instances`, `Inactive`: pending/stack arming, countered/resolved rearming,
  changed counts, control, departure, return, phasing, copy and clone isolation,
  independent factories and APNAP source identity.
- `PermanentSpellCopy`: native copied permanent entry, static bonus isolation
  and counter reset without changing the original permanent.
- `TidalClause`: the exact source reset clause; the complete card still rejects
  its casting restriction. This is not full-card coverage.
- `ResetTarget`, `ResetHost`, `EntryReplacements`: legality, current and last
  attachment, incarnation, phasing and replaced entry counts.
- `Declarations`, `Inventory`: scope, type, arity and threshold rejection;
  supported comparison/value/counter/reference descendant roles.

## RON live subject untap restrictions

`PreventUntapDuringStep(subject PermanentRef, condition StateCondition) StaticRule`
checks a typed subject and pure condition before the simultaneous untap decision
(CR 502.3). It shares the collection restriction pipeline. All conditions and
subjects are read before any permanent untaps. The rule does not grant a host
ability; removing the host's abilities cannot remove the source's restriction.
Removing the source's static ability disables it. Source presence, phasing and
incarnation checks use the printed static effect service. Current control and
attachment are read at the decision; returned hosts cannot inherit old attachments.
Static conditions use current state without LKI (CR 611.3a–b). Other untap effects
remain legal. Clones copy source state and effect wrappers independently.

RON adds `PermanentRuleEffect(subject,[DoesntUntapDuringControllersUntap])`
with `ThisPermanent` or an Aura's `HostPermanent`, either as a top-level rule or
as the sole body rule of static `If(condition,[rule])`. The condition uses the
existing static condition grammar. Mixed conditional rule/layer bodies, optional
host untap, collections and resolving/cost contexts are not added by this adapter.

`TestSubjectUntapRestriction`, `TestSubjectUntapDefinition` and
`TestSubjectUntapDecisionSnapshot` establish the engine contract. RON
`TestRONHostStaticUntap*` covers exact Cocoon, Merseine, Venarian Gold and Paralyze
clauses, live counter changes, native untap, attachment/control/phase/zone changes,
source and recipient ability removal, independent factories and clones, invalid
references/shapes and semantic roles. These are clause mappings; the full records
still reject their other unsupported rules. Existing `TestRONStaticUntap*` proves
that previously accepted source restrictions and their optional payments remain
functional. No replacement ordering or zone operation changes.

## RON resolving single-counter removal

`RemoveCounters([ACounterOfTypeFromPermanent(counter, subject)])` lowers to
`RemoveCountersFrom(subject, counter, Fixed(1))`. The action removes one counter
of that kind if available; it does not require a counter as payment. It reads
current counters at resolution, preserves other kinds, and continues later
instructions when none remains (CR 101.3, 122.1, 608.2c).

The subject uses the existing typed source, Aura attachment, target or event
reference. Absent and phased subjects are not modified; LKI is not used to
change counters. A returned subject has a new incarnation (CR 400.7, 702.26b/d).
Host actions follow the current attachment at resolution, or the original
source's retained attachment after departure. A saved event reference remains
the event object even after reattachment. Illegal targets prevent resolution
through the normal stack checks (CR 608.2b). Native copies retain their source
and use their own ability controller; game clones isolate counter mutations.

This adapter adds no removal event, replacement order, simultaneous batch or
cost contract. In particular, Divine Intervention's last-counter trigger is
still unsupported. Existing transactional counter costs remain separate.

Evidence: `TestRONSingleCounterRemovalVenarianGold` covers real Aura casting,
unpaid rollback, announced X, entry tapping, conditional untap and the correct
upkeep through zero counters. `VenarianCopies` covers copied permanent spells,
X, token entry, independent upkeep removals and clone isolation. `Source`,
`Target`, `Host`, `Event` and `Declarations` cover identity, legality, phasing,
attachments, pending bindings, copies, clones, continuation and invalid forms.
`Clauses` executes exact removal actions from Divine Intervention, Ebon Praetor,
Merseine, Tourach's Gate, Armageddon Clock, Clockwork Avian, Clockwork Beast and
Glyph of Delusion; their complete records still reject independent blockers.

## RON completed-draw event adapter

`WhenAPlayerDrawsACard` uses the existing typed completed-draw patterns for any
player, the current opponent, or the current controller. CR 121.2 requires one
event per card drawn. Replaced draws produce no draw event (CR 603.2g).
`Trigger_ThatPlayer` keeps the event player through control changes, source
departure, ability copies and game clones. Filtering occurs at detection; it
is not checked again at resolution. Source phasing prevents detection but does
not cancel an existing trigger. Damage remains source damage through the shared
replacement pipeline. No new engine rule or replacement order is introduced.

Evidence: `TestRONDrawEventUnderworldDreams`, `Bindings`, `ActualDraws`,
`SeparateInstances`, `FastingClause`, `Declarations`, and `Inventory` in
`pkg/mage/dsl/ron/draw_event_test.go`. Underworld Dreams uses its complete RON
record; Fasting has clause coverage only and remains rejected as a full card.

## RON resolving Aura host controller

The bare Aura-scoped `HostController` reference reuses
`ControllerOfWithLKI(AttachedPermanent)`. No engine API or rules change.
CR 303.4b and 608.2h require the attachment and its controller at the information
read, separate from the player saved by `AttachedControllerUpkeep` when the event
occurs. The Aura's controller owns the trigger; the resolving host controller
receives its payment choice. A control or attachment change before resolution
can change that payer without changing the captured event player (CR 603.3a).

Exact-incarnation attachment, departure and phase information reuse the existing
engine service. Source departure or return retains the old attachment; host
return cannot substitute a new controller. Host phasing retains information,
but `UntapSubject` cannot affect the absent host. An optional payment can still
occur when a later untap action cannot change the host (CR 603.5, 608.2c/d/g).
Cost transactions and mana-ability payment remain in the shared cost service.
Copies keep their source reference independently of the new ability controller.
Search clones use their normal noninteractive choice policy and isolated state.
No replacement order or simultaneous-operation contract changes.

`TestRONHostControllerParalyze`, `ResolutionIdentity`, `NativeUpkeep`, and
`InstancesAndReprints` cover complete Paralyze, its six reprints, casting
rollback, creature legality, entry timing, untap restriction, paid/declined and
unaffordable choices, mana abilities, control/attachment changes, source and
host identity, phasing, ability removal, independent pending instances, copies
and clones. `CurseDamageClause` covers the exact unpaid damage clause from
Curse Artifact with current-controller reads, LKI, prevention, copies and clone
isolation; the full card remains rejected for its optional sacrifice cost.
Erosion remains rejected for its alternative mana/life cost. `Activation` and
`Declarations` verify shared action context and declaration rejection.


## Fixed alternatives in resolution payments

`IfPaid` and `UnlessPaid` can use `EitherCost` with fixed mana/life options,
including nested alternatives. CR 118.2, 118.3, 118.5, 118.10 and 118.12a require
resolution-time mana ability access, full payment, explicit acceptance of a
zero cost, separate outcomes and an unpaid continuation after decline. CR
608.2c-d places the payment and option choice at resolution. These are payments
during resolution, so an activation cost reduction cannot reduce a mana option.

The shared private affordability function recursively checks alternatives with
the transaction's mana policy. Resolution checks and option locking use raw
mana costs; activation transactions retain activation reductions. Casting and
legacy locking callers retain their current behavior. Option locking precedes
the existing atomic total-cost validation and commit. Failed validation spends
no live mana or life and runs the unpaid branch. Independent payments cannot
consume each other's outcomes. No new global payment-result state is added.

`TestResolutionAlternativeManaPolicy` covers mana-only, life-only, both options,
choice, decline, neither option, nested alternatives, mana abilities, combined
cost rollback and false affordability caused by activation reductions.
`TestResolutionAlternativeKeepsActivationPolicy` covers unchanged activation
reductions and clone isolation. RON `TestRONAlternativePayment*` covers complete
Erosion and Thrull Wizard records through casting, activation, native upkeep,
target legality, payer control changes, pending source departure, return and
phasing, changed-controller and retargeted copies, prevention of countering,
regeneration, clone isolation and declaration rejection. Existing exact
attachment/LKI and stack identities determine recipients; these changes add
no new identity, replacement-order or simultaneous-operation rule.

## Optional sacrifice of a typed subject

`IfSacrificePaid(PlayerRef, PermanentRef, ResolvingEffect)` and
`UnlessSacrificePaid(PlayerRef, PermanentRef, ResolvingEffect)` offer a payment
at resolution. Definition validation rejects nil or unbound inputs. The subject
is read once before the offer. Its ID and incarnation restrict the existing
`SacrificeMatchingIncludingSourceCost`; `IfPaid`/`UnlessPaid` payment behavior
supplies affordability, the payer's choice and the atomic action transaction.
The payer must control that incarnation (CR 701.21a). Missing and phased subjects
cannot pay; phase cycles preserve identity, while zone changes do not (CR 400.7).
A change during the offer cannot make the transaction sacrifice a returned card.

Sacrifice uses the shared action and zone replacement pipeline. Regeneration
cannot replace it. A graveyard-to-exile replacement still pays the cost
(CR 118.11–12). Each invocation creates its own cost; no choice or payment result
is stored in the declaration. Pending copies and cloned games retain their own
bindings and state. Clone search players make their own optional choices.
An absent payer runs no branch, as in the existing resolution payment service.
This contract adds no mandatory, multiple-subject, simultaneous payment or
replacement-ordering service.

`TestResolutionSacrificeIdentity` and `TestResolutionSacrificeDefinition` cover
payment polarity, legality, decline, control, owner destination, absence,
leave/return, phasing, stale payment, replacements, cloning and invalid inputs.
RON gameplay and target/copy/result contracts map to `TestRONTypedSacrifice*`.

## Event player references in permanent target restrictions

`TargetControlledBy(PlayerRef) PermanentTargetOption` accepts `You` or
`FromPlayer` of a saved event binding. It compares current permanent control
with that player at target selection and resolution (CR 603.3d, 608.2b).
Generic trigger validation visits the event declaration before target
restrictions, so a saved player read must have its producer in the same ability.
Nil, unbound and unsupported references fail definition validation.

Target queries collect player values from the existing immutable event capture
instructions. They do not execute an ability or its costs. Separate pending
events retain separate maps, even with the same binding declaration. Stack
copies preserve these values when their controller changes, and retargeting
uses the captured player with current permanent control and ordinary legality.
Copies and game clones copy the maps. Query frames do not change live resolution
bindings or consume randomness. Random target selection consumes the live
random stream after candidate enumeration. Other event data, damage batches,
cost transactions, replacement ordering and APNAP ordering retain their existing
services.

The RON source attack and unblocked attack events now expose
`Trigger_DefendingPlayer`. These preserve declaration/blocker timing
(CR 508.3a, 509.1h, 509.3g). Entry attacking is not an attack declaration.
Only singular `TargetPermanent` filters accept `ControlledByPlayer` with this
binding or `You`, alone or in nested `And` filters. Other contexts remain errors.
`Trigger_ThatCreature` still reads the original source incarnation. A returned
source cannot pay sacrifice (CR 400.7, 701.21a); a phased source is absent for
payment. A phase cycle preserves its identity. Source departure or control
changes do not change the pending ability's controller or saved defender.

Mindstab Thrull offers source sacrifice at resolution and, on success, makes
the saved defender discard three cards from their current hand. Fewer than
three cards means discarding the available cards. Necrite chooses a defending
player's creature before offering payment. An illegal target prevents resolution
and payment. Its destruction forbids regeneration but does not override
indestructibility. Existing sacrifice replacements can change the source's
destination while payment succeeds. A copy cannot sacrifice the same source
twice. Copies with a different controller retain the defender but must still
control the source to sacrifice it.

Engine evidence: `TestTargetPlayerReferenceEvent`, `Definition` and
`SnapshotsAndQueries`. Complete-card RON evidence:
`TestRONDefendingPlayerNative`, `Pending`, `InstancesAndCopy`,
`ScopeAndInventory` and `Declarations`. These also cover target control,
return/phasing/shroud, inactive sources, ability removal, casting rollback,
source payment identity, current hand size, copies, clone isolation and syntax.

## Live combat pairs in typed permanent targets

`TargetBlocksSource`, `TargetBlockedBySource`, and `TargetInCombatWithSource`
return `PermanentTargetOption` values. They require a battlefield source in
engine definition validation. The last option permits either direction; separate
options require all supplied restrictions. Target selection and resolution check
current combat pairs with both battlefield incarnations. `Combat.BlocksIncarnations`
uses the existing groups and incarnation maps, including blocks of attacking
bands and multiple blockers (CR 509.1g/h, 702.22).

A source or target that leaves combat no longer qualifies. A departed, returned,
or phased source cannot supply the old relation. Source information retained for
target queries now includes its incarnation; LKI does not create a combat pair.
Phasing removes combat membership, so a phase cycle preserves object identity
but does not restore the block (CR 506.4). Tapping does not end the relation
(CR 506.4b). Control changes remove combat membership. Ordinary shroud and target
identity checks still apply. Removing source abilities does not remove an ability
already on the stack. Copies retain the source incarnation even when their
controller changes; clones retain separate combat and target state.

Evidence: `TestTargetCombatRelation`, `TestTargetCombatRelationDefinition`, and
`TestTargetCombatRelationBandsAndMultiplePairs`. RON evidence:
`TestRONCombatTargetRealDeclarations`, `TestRONCombatTargetDeclarations`, and
`TestRONCombatTargetNativeFactories`. The real-record tests cover only the target
clauses of Lesser Werewolf and Sentinel. The resolving property branch below
completes Lesser Werewolf; Sentinel still has an unsupported toughness effect.
Costs, replacement ordering, simultaneous actions, and event ordering use the
existing services without changes.

### Resolving permanent property branch

`IfPermanentMatches(subject PermanentRef, filter PermanentFilter,
body ResolvingEffect) ResolvingEffect` evaluates a pure property predicate when
its instruction resolves (CR 608.2c/h). It reuses `readPermanentInformation`:
current information for the exact live incarnation, the retained phase snapshot
for a phased object, or departure LKI for the original incarnation. Missing
information makes the condition false, including a zero filter. A returned
object cannot supply information for the original. Phase-in restores current
information. The predicate receives the retained object's properties and the
current game; queries of live combat relations or unrelated game state are not
historical queries. RON uses the separate live combat branch APIs for membership.

Source, attachment, saved-permanent and singular-target information references
are supported. Definition validation rejects nil or unsupported references,
unbound targets and nil bodies. The body runs once through the existing executor
only when the predicate matches. The body retains its controller and reference
bindings. Information does not permit mutation of absent, phased or returned
objects. Target legality is checked by the normal stack resolution service before
the branch executes; an illegal target does not become valid through LKI.

No cost, event, replacement, simultaneous-operation or static-state service is
changed. Those operations inside a body keep their existing contracts. Properties
and definition children propagate through the branch for copying and validation.
`PermanentMatchesFilter` remains the distinct static condition API. RON rejects
conditional mana production in activation branches until classification and
solver planning have a matching condition contract.

Evidence: `TestIfPermanentMatchesInformation` tests source/target/saved/host,
current power and keywords, control, departure/return, phasing, unknown identity
and clones. `TestIfPermanentMatchesSequenceAndSnapshot` tests sequential timing,
effective power retained after cleanup and return, and action identity.
`TestIfPermanentMatchesDeclaration` tests invalid definitions.
`TestRONResolvingConditionWerewolf`, `RepeatedWerewolf`, `Eater`,
`RollbackAndCasting`, `InstancesAndRetarget`, `CombatAndInventory`, `Inventory`
and `Invalid` establish the RON adapter and complete Lesser Werewolf and Eater of
the Dead gameplay mappings. Sewers of Estark has condition-clause evidence only.

## RON captured combat relation collection prevention

`PreventCombatDamageByEachThisTurn(PermanentSet)` adds a turn-duration constructor
for the existing `combatSourcePreventionInstallation`. It adds no replacement
ordering, source lookup or collection state. `BlockersOf` and `BlockedAttackersOf`
read current blocking pairs and exact incarnations at resolution (CR 506.4,
509.1g/h); LKI cannot restore an absent pair. The installation then saves each
selected incarnation in `combatSourcePreventionReplacement`. Later departure,
control changes, phasing and combat exit preserve installed prevention, while
returning the physical card creates a different source. Captured departure
damage retains the old prevention (CR 400.7, 609.7, 615.2–4). Combat duration
expires at the combat boundary; turn duration expires in native cleanup.
Noncombat damage remains unaffected. Competing replacements and simultaneous
damage still use the shared pipeline (CR 615.6, 616.1).

The RON compiler lowers the exact creature/relation conjunction and singular plus
relation union documented in `dsl-reference.md`. Feint selects current blockers
for tapping, then captures its target and current blockers for turn-long combat
prevention. Sewers of Estark checks attacking/blocking status at resolution; its
attacking restriction does not erase a block already declared (CR 509.1b), and
its blocking branch captures the blocker and current attackers for this combat.
No live static combat-relation prevention is established here.

Evidence: `TestCombatCollectionPreventionTurnDuration`, `TurnIdentity`,
`TurnDeduplicates`, `TurnDeclaration`; complete-record
`TestRONCombatCollectionCards`, `SewersAttacking`, `AnnouncementLegality`,
`Invalid`, `StrictInventory`. Existing `TestBlockersOf*`,
`TestBlockedAttackersOf*`, `TestCombatSourcePrevention*`,
`TestDamageSourceFilterReplacementOrder` and native combat regression tests retain
the collection, source, ordering and batch contracts. Payment queries and cost
transactions are unchanged; RON illegal/unpaid casts check resource rollback.

## Filtered permanent sets and permanent-only damage batches

`FilterPermanentSet(subjects PermanentSet, filter PermanentFilter) PermanentSet`
reads the set once per operation and keeps its current matching, unphased members
(CR 608.2h, 702.26b). It preserves the source set's order and identity checks.
It does not declare targets, broaden a target filter, capture absent objects,
or use LKI to select recipients. With plural target bindings, only surviving
legal target incarnations are available (CR 400.7, 608.2b). A later control
change does not exclude an otherwise legal target. The zero filter keeps all
members. Nil subjects and undeclared target reads fail definition validation.
The wrapper has no mutable state, so cloned games use their own current members.

`PermanentDamageRecipients(subjects PermanentSet) DamageRecipientSet` supplies
an empty player set to the existing mixed recipient service. It preserves strict
validation of the existing two-set `DamageRecipients` constructor. Damage uses
`DealDamageEach` and the existing simultaneous batch, source LKI, lifelink,
prevention, replacement ordering and notification contracts (CR 120.4, 120.8,
608.2h, 615, 616). No separate damage executor is added.

The RON collection adapter reads live filters at resolution. The restored
Winter Blast record filters its legal target set by flying after the tap
instruction. Flying is a damage condition, not a target restriction.
`TestFilterPermanentSetDeclaration` and `Targets` cover declaration checks,
current flying, target return/phasing/shroud/control, and pending clones.
`TestRONCollectionDamageRealCards`, `WinterIdentity`, `WinterAnnouncement`,
`SandstormMembership`, `InstancesCopyAndInventory`, `SimultaneousAndPrevention`
and `SourceIdentity` cover complete real records, X, rollback, live combat
membership, copies, factories, atomic loading, simultaneous notifications,
finite prevention, source LKI and lifelink. `Invalid` covers rejected shapes
and scope errors. Existing replacement-order tests remain the ordering evidence;
this slice changes no replacement choice or priority contract.

## Captured characteristic values and end-of-next-upkeep expiry

`ToughnessOfWithLKI(PermanentRef) ResolutionValue` uses the same exact-incarnation
information service as power. Live reads include layers and counters. Departure
and phase snapshots use the saved final toughness, without recalculating layers
on an absent object (CR 400.7, 608.2h, 702.26). A returned object cannot replace
that value. Unknown incarnations return zero. Definition validation rejects nil,
unsupported and undeclared references. `TestPermanentInformationLifecycle`,
`Declaration` and `PhaseSnapshotAndClone` now cover toughness as well as power.

`UntilEndOfNextUpkeep(PlayerRef, ...ContinuousModification)` captures its player,
subjects and values with the existing layer service. At the next actual upkeep
of the saved player, installed effects are marked for end-of-step expiry.
Effects created during that upkeep are not marked. After the priority round,
marked effects expire before final state-based actions and layer refresh
(CR 500.5, 611.2a, 613.4). There is no delayed trigger or replacement for expiry.
Skipped upkeeps cannot mark effects; extra upkeeps can. Source departure,
control changes and recipient phasing cannot delay or redirect the endpoint.
Clones copy both the saved player and the marking state. Existing beginning-of-
upkeep durations and legacy source-controller policies retain their behavior.

`TestUntilEndOfNextUpkeepBoundaries`, `NewEffect`, `Declarations` and
`SkippedControlAndPhasing` establish these contracts. RON
`TestRONTemporaryCharacteristicHalfdane` proves complete-record gameplay,
resolution-time P/T capture, layer/counter order, legality, return/phasing,
control, copied abilities and clone isolation. `Values` proves simultaneous
capture of multiple modifiers before installation. Target announcement,
transactional casting, replacement ordering and action batches use the existing
services; this slice does not add payment or replacement behavior.

## Paid permanent selection followed by simultaneous untap

`UntapWithManaPayment(player PlayerRef, filter PermanentFilter, cost ManaCost)
ResolvingEffect` selects any number of tapped, unphased matching permanents
controlled by the resolved player. The player may decline each additional
selection and may select zero. Selection is not targeting and ignores shroud.
Every selection is distinct. The operation captures ID and incarnation, checks
selection legality before payment, multiplies the fixed cost by the chosen count,
and pays the total through one existing action-payment transaction. A declined,
invalid or stale selection, absent player, or failed payment spends no resources
and untaps nothing. Integer overflow also fails before payment.

All choices precede payment; all payment precedes the untap batch. Thus an untap
result cannot fund its own payment. Mana abilities can fund the total and may
move a chosen object. After successful payment, only surviving, unphased captured
incarnations are acted on. Characteristics and control are not reselected after
payment. No LKI makes a departed object available for untapping. Untapping uses
`untapPermanentsTogether`: stun counters replace individual untaps, and all
changes finish before the first untap event. Untap-step restrictions do not
block this resolving untap. Payment is independent of activation cost reductions.
The API copies the mana template's hybrid list. Definition validation requires
a nonnil typed player, a nonzero filter and a nonnegative fixed cost without X.
No mutable chosen state is retained in the declaration or in a global result.

Evidence: `TestPaidUntapSelectionAndTotalPayment`, `InvalidAndStaleChoice`,
`Definition`, `ManaSourcesAndBatch`, `CannotFundPaymentWithResult`, and
`PaymentDeparture`; `TestRONPaidUntapCards`, `PlayerAndProcedureIsolation`,
`Invalid`, and `InventoryAndStrictLoading`. These cover subset/zero selection,
shroud, one total rollback, mana sources, source departure, event player,
control, returned/phased objects, copies, clones, batch events, stun replacement,
declaration rejection and atomic collection rejection. Rules: CR 118.3,
118.12, 400.7, 502.3, 608.2d/g/h, 701.26b and 702.26b.

## Graveyard-only shuffle and RON exile payment reuse

`ShuffleGraveyardFor(PlayerRef) ResolvingEffect` uses the shared library recycler
with hand movement disabled. It resolves the player at execution, moves that
player's whole current graveyard to their library, then reports each zone change
and one `EvtCardsLeftGraveyard` event with the moved count. All moves precede
all events. It shuffles once after the events, including for an empty graveyard
(CR 701.24d). Hand cards and other players' zones are preserved. Nil or unbound
player declarations fail definition validation. An absent player has no effect.
The legacy `ShuffleGraveyardIntoLibrary` effect delegates to this service for
`You`; current callers remain intact. `TestGraveyardOnlyShuffleBatch` covers
native and legacy paths, complete-batch event observations, empty shuffles,
controller scope, hand preservation, nil declarations, and clone isolation.

RON source exile and matching-card graveyard exile use `ExileSourceCost` and
`ExileMatchingCardFromGraveyardCostInto`. The existing action payment transaction
locks a selected card and its graveyard visit, validates the total cost, and
commits atomically (CR 118.3, 602.2). The saved mana value belongs to that pending
activation, not the game's last exiled card. It survives source departure and
later departure of the paid card. Copies retain the saved payment value without
paying again. A different copy controller does not change that value.
Counter placement still requires the original live source incarnation; phasing
or leaving and returning cannot restore that action through LKI (CR 400.7,
608.2h, 702.26b). A phase cycle before resolution preserves the incarnation.
This adapter adds no new payment, counter, zone-replacement, or ordering service.

`TestRONExilePayment*` covers complete Necropolis and Feldon's Cane records,
native casting, costs, defender, announcement timing, wrong-zone/type/payment
rejection, source and graveyard total rollback, locked choices, independent
instances and pending activations, modes, copied controllers, cloned resolution,
source control/departure/return/phasing, paid-card departure, and strict loading.
Existing `TestExiledGraveyardCostManaValue*` and `TestPaidExile*` supply the reused
payment's stale-visit, result-declaration, and transaction regression evidence.

## Opponent choice before entry and RON chosen-player upkeep

`ChooseOpponentAsEnters() AsEntersBattlefieldAbility` saves the opponent of the
entry controller in the entering permanent's `ChosenPlayer` field. The existing
`prepareBattlefieldEntry` hook runs it before battlefield registration, subsequent
`AsEntersBattlefieldAbility` operations, and zone/entry observers (CR 614.12a).
The two-player engine has one eligible opponent, so this choice needs no prompt,
target, stack object, or payment. The owner does not determine the choice.
Definition validation rejects the ability on an instant or sorcery.

Each permanent owns its saved choice. Control changes and phase cycles preserve
it (CR 702.26); a new entry makes a new choice (CR 400.7). Permanent spell copies
choose for their own entry controllers. Clones keep independent permanent state.
The legacy `ETBChooseOpponent` and `ChooseOpponentOnETB` APIs retain their existing
timing for current callers. This constructor does not add a general player-choice
service, entry replacement ordering service, or multiplayer decision policy.

RON supports exactly `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`
with one operation and one declaration per card. This declaration permits
`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),body)`.
The compiler checks the entry declaration before compiling the other rules;
their textual order does not change the static declaration's scope. Other choice
filters, combined entry operations, resolving choices, chosen-player action reads,
and chosen-player end-step filters remain unsupported.

The trigger reuses `TurnStepBegins(EachUpkeep,playerBinding)` and
`EventIsChosenPlayerUpkeep`. It checks the permanent's choice when the event
occurs and captures that event's player before trigger placement. The check is
not an intervening-if condition. Changing control, removing abilities, phasing,
departing, or returning after the event cannot erase or redirect a queued trigger
(CR 113.7a). Inactive sources cannot generate new triggers. Copies keep the saved
event player while using their own controller. Black Vise and The Rack read that
player's current hand size at resolution, then deal damage through the existing
damage, source LKI, prevention and lifelink services (CR 608.2h).

Evidence: `TestChooseOpponentAsEntersTiming`, `Identity`, `Declaration` and
`TestRONOpponentEntryRealCards`, `Identity`, `InstancesAndLoading`,
`NativeAndCopy`, `DeclarationScopeAndInventory`, `Invalid`. The RON tests use the
complete Black Vise and The Rack records. Cursed Rack's maximum-hand-size effect
and Jihad's combined choices remain separate prerequisites.

## RON end-of-combat event and history reuse

The RON adapter uses `EachEndOfCombat` for the beginning of the end-of-combat
step (CR 511.2). Clockwork history conditions use
`SourceAttackedOrBlockedThisCombatCondition` through `WithInterveningIf`, so
CR 603.4 and 608.2a apply at detection and resolution. Native step execution
resolves these triggers before combat history resets. History follows the
original incarnation, survives removal from combat and control changes, and
resets for the next combat. A phased or returned source cannot satisfy an old
trigger's condition. Ability removal after detection does not remove the
pending trigger. Copies retain the source identity; clones retain independent
history and counters. Plain triggers do not add a source-presence condition.

`TestRONEndCombatClockwork` covers source entry counters, flying, attack and
block declarations, idle sources, other instances, entering attacking,
removal from combat, native end steps, extra combats and clones.
`TestRONEndCombatInterveningIdentity` covers both condition checks, phasing,
return, control, ability removal and stack copies. `TestRONEndCombatPlain`
covers event selection, departure and copied controllers. Existing
`TestSourceAttackedOrBlockedThisCombat*` tests guard the reused engine service.
No engine execution, replacement ordering, cost or simultaneous-operation
contract changes in this slice. Full-card loading remains blocked by bounded
counter restoration and blocking-collection control effects.
