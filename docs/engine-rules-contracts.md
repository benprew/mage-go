# Engine rules contracts

## Live restrictions on spell targeting

`CantBeTargetedBySpellsIf(PermanentRef, CardFilter, StateCondition)` adds a
required pure condition to the existing restriction. Definition validation
checks the condition and its reference dependencies. Target queries check the
condition with the live source's current controller and exact incarnation.
Turn changes and turn-history changes can therefore change legality before
selection or resolution, without a saved branch choice (CR 611.3a).
The existing source lifecycle, attachment, spell information, copy and target
services apply. Ability targets remain separate; no replacement operation is
introduced. The unconditional constructor retains its behavior and callers.

`TestCantBeTargetedBySpellsIf*` covers invalid conditions/references/filters,
live turn and control changes, phasing, ability removal/restoration, clone
isolation, host reattachment, returned hosts and departed sources.
`TestRONStaticBranch*` covers complete Angry Mob and Lurker records, payment
rollback, opponent counts, base/modify/counter/switch layers, native trample,
attack/block history, phase cycles, final cleanup, source return, target rechecks,
spell/ability copies, departed ability-source information, strict loading and
independent factories. Existing history services distinguish declarations from
objects put into combat and retain history until final cleanup (CR 508.1,
509.1, 514.3a, 400.7, 702.26). Static branch modifiers use `While` and
`NotCondition`; no new continuous-effect ordering or history service is added.

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

## Land-entry permanent capture

`LandEnters().WithPermanent(binding)` saves the entering land's card UUID and
battlefield incarnation when the event is detected, before trigger placement.
It can be combined with `WithActor` and `WithController`; all three declarations
write separate typed bindings. Nil, empty-name, duplicate and unbound declarations
fail definition validation. Existing callers without this capture keep their
behavior. This adds no entry replacement or alternative event dispatcher.

`From(binding)` uses the shared permanent reference service. Live actions require
the saved incarnation to be present. Departure and return cannot redirect an old
action. Phasing suspends actions; a phase cycle preserves identity. Information
reads use current or retained information for that incarnation, as required by
CR 400.7, 603.2–3, 603.6a–b, 608.2h and 702.26b. Pending events, trigger copies
and game clones retain their own captured values.

Evidence: `TestLandEntryPermanentDeclaration`, `TestLandEntryPermanentIdentity`,
`TestLandEntryPermanentCombinedCaptures` and `TestRONLandEntry*`. Existing
`TestLandEntryController*`, `TestLandEntryActor*` and `TestLandEntrySubtype*`
continue to cover entry controller snapshots, known actors and event-time subtype
checks. RON Ankh tests cover current versus retained controller reads, source
departure, native land play, casting rollback, prevention, ability removal,
independent sources and pending events. Complete Chant gameplay is covered by
`TestRONCounterPlacementPayment*`; its payment contract is below.

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

RON also accepts the bare `ActivateOnlyAsASorcery` modifier through the existing
`WithSorcerySpeed` action option (CR 307.5, 602.5d). It requires either main
phase, the activating player's turn, and an empty stack. Normal priority
handling remains in the priority loop. The player need not hold a sorcery card.
The compiler keeps this timing option after other combined options because
`WithYourTurnOnly` uses the same timing field. Other activation conditions and
limits continue to apply. Timing is checked before modes and payment; pending
abilities and copies can resolve in a later step or turn. The mana solver and
explicit mana activation keep the same restriction and immediate mana behavior.
`TestRONSorceryActivationTiming`, `CombinedTiming`, `Modal`, `ManaPlanning` and
`RepositoryPrerequisites` verify these contracts without changing the engine.
This does not establish the unsupported bodies of Gaea's Touch or Illusionary
Mask, or grant priority during another action's payment.

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
to `PreventDamageFromMatchingPermanents(This, filter)` in the typed static
prevention service. The adapter
validates the event, recipient and operation list at compilation. Each factory
creates an independent static ability. The source filter and replacement
algorithm retain their existing behavior.

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

RON combat-pair filters now accept a required creature subtype and a fixed
toughness comparison. They reuse `IsCreature`, `HasSubType`,
`HasToughnessEqual`, and `HasToughnessGTE` through the existing matching patterns.
The event reads final current characteristics at detection (CR 509.3f).
A later subtype or toughness change cannot change a saved event.
`TestRONCombatCharacteristicShapes`, `Identity`, and `DetectionAndInstances`
cover these filters, typed opponent capture, delayed destruction, phasing,
returned objects, independent factories, copies, regeneration and clones.
`NativeClauses` executes the original Battering Ram and Infinite Authority event
declarations with test bodies through native combat. `Consumers` inspects their
complete records and Dwarven Soldier's original Orc filter. These are clause
and prerequisite tests; none establishes complete gameplay for these cards.
`Invalid` covers malformed filters, duplicate conditions and wrong subtype
families. No new engine rule or grouped combat event is established.

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

The RON adapter supports nonmodal stack abilities and the fixed mana context
specified below. Existing `AtNext` captures the source incarnation and resolving
controller (CR 603.7). A delay
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
service. Fixed mana conversions with history-gated delayed instructions use
the synchronous execution contract below.

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

## Bounded optional counter placement

`AddCountersUpTo(subject PermanentRef, counter CounterType,
amount ResolutionValue, limit int) ResolvingEffect` gives the effect controller
a resolution choice from zero through `min(amount, max(limit-current, 0))`.
The declaration requires a valid typed reference and value, a valid permanent
counter type and a limit from zero through 255. Larger limits are rejected
because the current engine stores each counter count in one byte. Nil and
unbound declarations fail engine validation. Invalid player responses place no counters. No choice is made for
an absent or phased subject, a nonpositive amount, or a full subject.

The subject retains the reference service's captured incarnation. Control
changes and phase cycles preserve it; departure, return and unresolved phasing
prevent placement. The amount and current room are read during resolution,
not announcement. Announced X comes from the individual stack frame. Copies
use their controller for choices. Game clones retain independent counter and
stack state; native cloned players use their normal simulation decisions.

Placement uses the existing `AddCountersAction` replacement pipeline and its
affected-player order and rechecks (CR 614–616). A private optional limit leaves
existing `AddCountersWithReplacement` callers unchanged. After replacements,
placement on the original subject requires its original incarnation. The final
amount for the restricted counter on that subject is limited to its current room.
Replacements that cancel placement still allow later instructions. A replacement
that changes the counter or destination does not transfer the original subject's
restriction. The limit belongs to this instruction, not a static counter maximum:
other actions can exceed it, and this action does not remove existing excess.

The resolution choice follows CR 608.2c–d; counter identity and zone changes
follow CR 122.1–2 and 400.7. Clockwork source records and local Oracle data specify
that the restoration ability cannot cause a total above four or seven. This is
not the static maximum in CR 122.4. No collection or simultaneous placement
shape is added. `TestAddCountersUpTo*` and `TestRONBoundedCounters*` establish
bounds, choice timing, incarnation, phasing, clones and replacement behavior.

## Static prevention from sources with a card type

`PreventDamageFromCardType(subject PermanentRef, kind core.CardType) StaticRule`
uses the same live subject, source incarnation, cycle replacement, definition
validation and clone services as static combat prevention. It prevents combat
and noncombat damage with no amount limit. Nil or undeclared subjects and card
types outside the engine enum fail validation. The RON adapter accepts only
`IsCardtype(type)` with a source or declared Aura host in a static rule.

The damage-source type check reads current characteristics while the original
permanent is present, retained characteristics after departure or phasing, and
the captured card for a spell source. Unknown captured permanent incarnations
cannot use a later UUID match. UUID-only legacy actions retain live lookup.
An Aura follows its current attachment. Recipient ability removal does not erase
an Aura's rule; source ability removal does. Source departure or phasing stops
the rule, and a returned source cannot restore an old replacement entry.

Matching is repeated after other replacements. Source-type changes, source
phasing, and redirection can change applicability. The shared damage pipeline
retains affected-player choices, simultaneous packets, unpreventable damage and
recipient incarnation checks. This slice adds no choice, cost, allocation,
source-enumeration, next-event shield, or spell-to-permanent link contract.
Existing permanent-only prevention filters retain their spell exclusion.

Evidence: `TestStaticSourceTypePrevention`, `Declarations`, `LiveHost` and
`Recheck`; `TestRONStaticSourceTypePreventionTreefolk`, `SpellsAndActivations`,
`Host`, `Invalid`, `Ordering`, `StrictInventory`, `Combat` and `Instances`.
The RON Treefolk tests use its entire repository record. Host tests use a
synthetic RON Aura fixture; they do not claim complete Artifact Ward support.
Rules: CR 400.7, 609.7c, 611.3a–b, 615.1–6, 615.12, 616.1/616.1f and 702.26b–i.

## Live prevention from blocked creatures

`PreventDamageFromBlockedCreatures(subject PermanentRef) StaticRule` prevents
all combat and noncombat damage to the current subject from creatures it
currently blocks (CR 509.1g, 611.3a–b, 615.1). Compose it with `Static`.
`This` protects the source; `AttachedPermanent` follows an Aura's current host.
Nil and undeclared subjects fail definition validation.

The rule reuses the static prevention lifecycle and shared damage replacement
pipeline. It checks the source incarnation, current subject, current creature
type and `Combat.BlocksIncarnations` on each proposed event. Damage-source
identity uses the existing current/LKI matcher, but a saved source must still
be the live original incarnation in the current pair. LKI alone cannot restore
combat. Departure, return, phasing and removal from combat end that pair
(CR 400.7, 506.4). A phase cycle can restore the static ability, but it does not
restore a combat pair. Removing the rule source's abilities ends prevention;
removing a separate Aura host's abilities does not remove the Aura rule.

There is no shield consumption. Repeated damage and simultaneous packets use
the same check. Unrelated recipients, players, reverse pairs and noncreature
sources do not match. Replacement choices and rechecks after pair changes or
redirection remain in the shared pipeline (CR 616.1). Unpreventable damage uses
that pipeline's existing bypass (CR 615.12).

Evidence: `TestBlockedDamagePrevention`, `Declarations`, `Recheck` and `Host`;
`TestRONBlockedDamagePreventionWall`, `Invalid`, `ShapeAndContext`,
`StrictInventory`, `ShadowsGap` and `Ordering`. These cover native combat,
real RON casting/payment rollback, noncombat activations and copies, separate
factory instances, clones, live attachments, ability removal, source return,
phasing, current type changes, batches, both affected-player replacement orders
and strict collection rejection. No timing, cost, LKI storage or damage batch
service changes in this slice.

## RON indefinite base toughness

`SetBaseToughness(PermanentRef, ResolutionValue) ContinuousModification` sets
only base toughness in layer 7b (CR 613.4b). It preserves printed/CDA power and
earlier base-power settings. Layer 7c bonuses/counters and layer 7d switches
still apply. `Until(Indefinite, ...)` captures the value and subject incarnation
at resolution (CR 608.2h, 611.2a–d). Later source departure and control changes
do not end an installed effect. Phasing preserves it; a returned card cannot
receive the old effect. `While` uses the same modification with live values.

`AddValues(ResolutionValue, ResolutionValue) ResolutionValue` uses the shared
arithmetic service. `CountGraveyardCards(PlayerRef, CardFilter) ResolutionValue`
is a pure count of the selected player's current matching cards. An absent
player gives zero; a zero filter counts all cards. A resolving ability's `You`
is its saved controller, even after source control changes. Counts captured by
`Until` do not change when more cards enter the graveyard. This API does not add
all-zone CDA evaluation or change existing card-characteristic queries.

RON accepts `CreatePermanentLayerEffect(subject, [SetToughness(value)])` only
in resolving action contexts with typed subjects and a single modification.
Values accept integer literals, nested binary `Plus`/`Minus`, typed
`PowerOfPermanent`/`ToughnessOfPermanent` reads through the existing current/LKI
service, and `TheNumberOfGraveyardCards(And([cardFilter,
InAPlayersGraveyard(SinglePlayer(playerRef))]))` with either operand order.
Card filters use the existing type/color and nonempty `And`/`Or` grammar;
player references must be valid in the enclosing scope. Static or temporary
RON `SetToughness`, chosen numbers and undeclared X remain unsupported.

Sentinel checks its combat-pair target at announcement and resolution. A source
or target removed from combat cannot retain the pair through LKI (CR 506.4,
608.2b). Once installed, its setting lasts indefinitely. Wall of Tombstones
triggers only on its controller's upkeep, counts at resolution, and keeps the
count until another setting supersedes it. Defender remains in force.
Evidence: `TestBaseToughness*`, `TestGraveyardCountAndAdditionQueries`, and
`TestRONIndefiniteToughness*`. Existing incarnation, target, stack-copy, cloning,
transaction and layer services retain ownership. No replacement ordering or
simultaneous-operation contract changes.

## Optional hand entry (RON migration slice)

`MayPutFromHandOntoBattlefield(PlayerRef, CardFilter) ResolvingEffect` offers one
matching permanent card from the selected player's current hand at resolution.
It makes no announcement choice. A nil player reference fails definition
validation. No candidates means no decision. Decline means no card selection.
Selections must be offered cards with the same hand visit and matching current
characteristics. Each application has local selection state; copies and clones
cannot share it. No permanent source presence or source LKI is needed.

The shared entry pipeline applies normal tapped entry, entry counters, static
registration and entry events. The selected player is both controller and actor;
ownership stays with the card. This is neither casting nor a land play. A land
prohibition preserves the hand visit. For Auras in the engine's permanent-host
model, the controller chooses a legal current host before entry. Shroud does
not restrict that choice; protection and enchant restrictions do. Phased hosts
are absent. A missing or invalid host response uses a legal offered host; its
identity and legality are checked again. Attachment precedes entry events.
No legal host preserves the hand.

Evidence: `TestOptionalHandEntryChoices`, `TestOptionalHandEntryDeclarationAndClone`,
`TestOptionalHandEntryAura`, `TestOptionalHandEntryAuraRequiredHost`, and the RON `TestRONOptionalHandEntry*` tests.
Contracts use CR 400.4a/400.7, 608.2d/h and 303.4f/g. This single-card action
adds no batch, APNAP or replacement-order policy; existing entry services retain
those responsibilities. It does not establish repeated entry procedures or
player-enchanting Auras.

## Live Aura attachment restriction and exception

`CantBeEnchantedExceptBy(PermanentRef, PermanentRef) StaticRule` implements a
live attachment restriction under CR 303.4, 303.4a/c/f/j and 704.5m. It uses the
existing active printed-effect lifecycle and typed permanent references. The
subject and exception use current incarnation identities, without LKI. Source
ability removal, departure and phasing suspend the rule; a phase cycle preserves
attachment identity. Control changes do not change its scope. Removing host
abilities does not erase a rule supplied by its Aura. Moving the granting Aura
moves the restriction. Returning a host cannot inherit its old attachment.
Clones read their own effect, source, suppression and attachment state.

Aura spell target queries use the existing spell source view. Ability targets
remain legal under this restriction. The shared `auraHostIsLegal` query supplies
direct attachment, entry selection and state-based attachment checks. The
exception identifies one Aura incarnation, so a separate factory or spell copy
cannot use the original Aura's exemption. Casting rejects an illegal target
before payment; resolution rechecks the rule. Shroud and spell-only restrictions
remain separate from direct attachment legality.

Illegal attached Auras are put into their owners' graveyards, rather than
destroyed, under CR 303.4c and 704.5m. All selected illegal Auras move through
one existing zone batch. The batch retains the former battlefield for trigger
checks and sends events after the moves. Indestructibility and regeneration do
not prevent these moves. This change establishes no new zone replacement API.

`TestCantBeEnchantedExceptByDefinition/Lifecycle/StateAction` covers invalid
references, exception identity, ability removal and cleanup, attachment/control/
phase/return changes, cloning, indestructibility, regeneration, simultaneous
moves and retained event information. `TestRONAuraRestrictionCards` covers the
complete Anti-Magic Aura and Consecrate Land records, paid casting, rollback,
target legality, both controllers, lifecycle, ability targets and indestructible
land behavior. `CopiesAndResolution` covers permanent spell copies, clones and
new restrictions at resolution. `ExistingAuras` covers previously attached
indestructible Auras. `Invalid` and `AtomicLoadAndInventory` cover declaration
rejection, bounded composition, strict loading and semantic reporting.


## Controller-aware enchant legality

`TargetPermanentYouControl(...PermanentFilter) Target` provides the raw target
counterpart to the existing `TargetYouControl` declaration option.
`TargetPermanentOpponentControls` and `TargetOpponentControls` supply the
opponent restriction. These permanent-target restrictions also constrain
an Aura's enchant ability when used with `WithCastTarget`. The shared
attachment query unwraps typed target declarations before reading their
filters and these controller restrictions.

Under CR 109.5, 303.4a/c/e/f/g/j and 702.5a, casting checks use the spell
controller, while attachment and state-based checks use the Aura's current
controller. Optional entry checks use the prospective controller before
entry, without substituting the card owner. Ownership can differ from control.
Changing either the Aura's control or its host's control can make the
attachment illegal. Direct illegal attachment attempts retain the old
attachment. Targeting restrictions such as shroud do not affect untargeted
attachment. Phasing retains valid attachment identity; unavailable spell
targets remain illegal at resolution. No LKI query can make an absent host
available for attachment.

Illegal Auras reuse the existing simultaneous state-action move batch and
former-battlefield event information. The moves go to each Aura owner's
graveyard and bypass indestructibility and regeneration. This change adds
no replacement-order, APNAP or zone-movement policy. Existing targeting,
copying, entry, incarnation and clone services retain their contracts.

Evidence: `TestAuraControllerAttachment`, `CurrentAuraControl`,
`UntargetedEntry`, `SimultaneousStateAction` and `TestRONEnchantController*`.
The engine tests cover raw and typed declarations, owner/controller
separation, live control changes, legal phase cycles, cloning, shroud,
blocked hand-visit preservation and batch departure information. The RON
tests also cover spell payment rollback, original and returned targets,
resolution timing, separate factory state and copies with either controller.
This slice does not add arbitrary player-binding restrictions to enchant
abilities, ownership restrictions or player-enchanting Auras.

## Resolving counter information

`CounterCountAtLeast(counter core.CounterType, minimum int) PermanentFilter`
compares one counter kind. It is a pure query, returns false for a nil permanent,
and promotes the stored count to `int` before comparison. Zero, negative and
above-storage-range thresholds keep ordinary integer comparison semantics.
RON declarations accept only nonnegative literal thresholds. Exact and absent
counts reuse `CounterCountEquals`; presence reuses `HasCounter`.

The RON resolving `PermanentPassesFilter` adapter accepts presence, absence,
`EqualTo(Integer(n))` and `GreaterThanOrEqualTo(Integer(n))` counter tests,
including nonempty nested `And` and `Or`. It uses `IfPermanentMatches` and its
existing current/LKI reference contract (CR 608.2c/h). Earlier counter actions
complete before the condition reads its count. LKI keeps the original counter
information after departure or phasing; CR 400.7 and 122.2 prevent a returned
object from supplying the old object's information or receiving its actions.
No state trigger, intervening-if, replacement, cost or batch contract changes.

Evidence: `TestCounterCountAtLeast` covers independent kinds, nil, zero,
negative and 256 thresholds, and clone isolation. `TestRONResolvingCounter*`
covers real upkeep sequencing, threshold boundaries, counter-kind isolation,
source departure/return/phasing/control, copies, retargeted spell copies,
per-factory bindings, clones, invalid declarations, casting rollback, native
upkeep progression, semantic roles and strict collection rejection. The
existing `TestIfPermanentMatches*` contracts remain applicable. Real-card
tests establish clauses of Fasting and Tourach's Gate; both full records still
reject their later unsupported rules.

## Typed activation state conditions for RON

`WithActivationStateCondition(StateCondition) ActionOption` supplies the same
restriction used by `ActivationIf`, for existing ordinary and modal action
builders. It records the typed condition as a definition child, so invalid
references and nil conditions fail definition validation. A nil condition is
also false during a check; it does not cause a game-time declaration panic.
Valid existing `ActivationIf` callers keep their behavior.

The shared activation path checks the condition before payment (CR 602.1b,
602.2, 602.5). Failed checks preserve mana, permanents, counters and stack state.
The mana solver uses the same pure check. No choices, random values, new payment
results or history state are introduced. The condition does not become an
additional resolution check (CR 608.2c); copies resolve without new activation
payment. Existing payment transactions and stack cloning retain their isolation.

`PermanentMatchesFilter(This|AttachedPermanent, filter)` supplies live property
checks. The current source must be present; host lookup checks attachment
incarnation. Phased or departed subjects do not match, and departure LKI cannot
make them match (CR 400.7, 702.26b). A new attachment supplies a new live subject.
Counter counts use the existing exact and minimum filters without narrowing the
threshold. No replacement ordering or simultaneous-operation service changes.

Evidence: `TestActivationStateCondition` covers definition validation, failed
payment, resolution timing and clones. `TestRONActivationPropertyClauses` uses
both actual source/host declarations and covers identity, attachment changes,
phasing, current controller, copies and pending clones. `ThresholdsAndModifiers`
covers boundary counts, independent counter kinds, logical filters and combined
timing. `ManaPlanning`, `Invalid` and `InventoryAndStrictLoad` cover pure solver
queries, declaration errors, nested reporting and atomic collection rejection.
`TriassicEgg` covers casting, both activated abilities and both modes, counter
production, optional hand entry, targeted graveyard entry, failed activation
rollback, graveyard visit identity, source return, copies and clones. Existing
`TestRONGraveyardReturn*` and `TestRONOptionalHandEntry*` supply the shared zone,
entry and choice contracts.

## Last completed turn attack history

`PermanentAttackedDuringLastTurn(subject PermanentRef, player PlayerRef)` reads
one live source or attachment and the specified player's most recent completed
turn. It uses the same live subject resolver as `PermanentMatchesFilter`.
`You` reads the current ability controller; `ControllerOf(This)` and
`ControllerOf(AttachedPermanent)` read current subject control. Neither query
uses LKI. Invalid subjects and players fail definition validation.

The shared turn tracker archives both card IDs and incarnations at final cleanup.
Current turn attacks and new combat phases do not overwrite the prior archive.
Another player's cleanup does not overwrite it. A later completed turn with no
attacks clears that player's history. Multiple combats contribute to one turn.
Both native and explicit attack declarations already record incarnations.
Clones copy the nested archive maps independently. Existing ID-only queries and
callers retain their behavior. CR 400.7 prevents returned cards from inheriting
history. CR 702.26b–d preserves history through phasing while excluding phased
subjects from live queries. CR 611.3a–b requires current attachment and control.

`CantAttackIf(subject PermanentRef, condition StateCondition)` extends the shared
attack restriction service with a pure condition. Source presence and printed
ability wrappers retain their existing lifecycle. The condition reads current
state at attack declaration (CR 508.1c); it does not remove a declared attacker.
Removing a recipient's abilities does not remove a separate Aura's rule.
Conditional untap continues to use the shared simultaneous decision snapshot
before untapping (CR 502.3); other untap effects remain permitted.

Evidence: `TestLastTurnAttackIdentity`, `ConditionalRule`, `Declaration` and
`MultipleCombatsAndPlayers`; `TestRONLastTurnAttackCards`, `Lifecycle`,
`KelpEntryTimingAndLegality`, `ConditionalBodyReuse`, `Invalid` and
`InventoryAndStrictLoad`. Complete records cover Giant Turtle's attack rule,
Goblin Rock Sled's trample, Mountain requirement and history untap rule, and
Tangle Kelp's creature enchant, entry tap and current-host history untap rule.
No new payment, replacement, simultaneous move or damage contract is introduced.

## RON static source combat attributes

The RON source declarations for `MustAttack` and
`CanBlockAnAdditionalCreature` reuse `GrantKeywordModification`, `While` and
`Static`. Printed effects follow the source incarnation and the existing
layer-6 ability-removal lifecycle. Source departure ends the old rule; a returned
source has a fresh printed rule. Phasing preserves identity and prevents a
phased source from participating in combat. Factories and game clones keep
separate source state.

`Game.RunStepWithPriority(DeclareAttackers)` uses the existing `AttrMustAttack`
check. It reads current control and attack legality, and it applies in each
combat (CR 508.1a/c/d). An attack requirement does not require payment of an
attack cost. `PayAttackCosts` retains its transactional contract; native step
completion separately empties unspent mana (CR 500.5).
`Game.RunStepWithPriority(DeclareBlockers)` uses `CanBlockAdditional` to permit
two attackers, subject to ordinary blocking restrictions (CR 509.1a/b).
Removing the ability after declaration leaves the declared combat pairs intact.
These adapters add no replacement, LKI, cost, damage or simultaneous-operation
service. Existing native combat damage and trample execute the complete records.

Evidence: `TestRONSourceCombatAttributeRequiredAttack`,
`TestRONSourceCombatAttributeAdditionalBlock`,
`TestRONSourceCombatAttributeAttackControlAndFinality`,
`TestRONSourceCombatAttributeDeclaredBlockFinality`,
`TestRONSourceCombatAttributeGiantTrample`,
`TestRONSourceCombatAttributeLifecycle` and
`TestRONSourceCombatAttributeReprints`. This evidence covers the two source
forms; it does not establish additive extra-block grants or a general solver
for interacting combat requirements.

## RON damage-received event lowering

RON source-permanent and controller-player damage events reuse `ReceivesDamage`
and `ControllerIsDealtDamage`. They observe positive damage after prevention
and redirection (CR 120.4b, 120.8). Life loss is not a damage event. Each pending
packet retains its actual amount; simultaneous sources do not overwrite one
another. Native trigger detection, APNAP placement, source incarnation and
resolution use the existing engine contracts (CR 603.2–3, 400.7, 608.2h).

The permanent event's recipient alias uses `This`, so a returned source cannot
receive counters from an old trigger. Phased or absent subjects cannot receive
an action; a phase cycle preserves the incarnation. Controller-player matching
uses control at detection. Later control changes do not change saved damage.
A copied trigger keeps its source identity and amount. Each factory builds its
own number binding, and cloned games keep independent pending and stack state.

Evidence: `TestRONDamageReceivedFungusaur`, `CastAndCombat`,
`ControllerClause`, `Batch`, `FinalRecipient`, `PermanentAmountsAndInstances`,
`ReprintsAndInventory` and `Invalid` (all with the `TestRONDamageReceived`
prefix). Fungusaur tests use its complete record and all six reprints. Living
Artifact tests isolate its exact damage clause from the complete AST and assert
that the full record remains rejected. No complete-card claim applies to it.

## Optional counter-placement payments

`PutCounterOnMatchingCost(core.CounterType, PermanentFilter) Cost` places one
counter on one matching permanent controlled by the payer. A zero filter accepts
any permanent. The source can pay. The choice is a payment choice, so shroud and
summoning sickness do not exclude it. Current control, presence, filter membership
and incarnation determine eligibility. Ownership does not determine the payer.
Phased objects cannot pay; a phase cycle preserves a prepared choice. A returned
card cannot replace a previously selected incarnation (CR 400.7, 702.26).

`CanPay` makes no choices or state changes. Payment preparation copies the cost,
asks the live payer once, verifies the candidate and saves its typed captured
permanent. Validation places counters only in the payment game. Commit checks
the saved choice before spending mana or life and uses the shared full-payment
preflight. A later failed cost, stale candidate, control change or failed filter
check cannot spend part of the cost. Cost declarations keep no chosen state.
Nested cost alternatives retain the same locked choice. The source need not
remain present for a resolving payment that chooses another permanent.

Placement uses `AddCountersWithReplacement`, with the causing source and
`onEntry=false`. Existing affected-player replacement ordering applies.
Modified counter type, increased amount, or prevented placement still pays the
cost (CR 118.11–12, 614, 616). `IfPaid` and `UnlessPaid` retain each invocation's
branch result. This introduces no new target, counter, event or replacement
service and no collection-wide simultaneous operation.

`TestPutCounterCostTransaction` covers pure queries, owner/control distinction,
locked choices, invalid choices, phase/return/filter changes, total rollback and
doubling. `ResolutionAndDeclaration`, `ReplacementPayment`, `StaleChoice` and
`AlternativeTransaction` and `ReplacementOrder` cover optional branches, source permission, cloned
payments, invalid counter types in definition validation, non-targeting shroud,
replacement prevention/type changes, nested alternatives and both affected-player
orders for competing increase/doubling replacements.

`TestRONCounterPlacementPaymentChants` uses complete FEM records for Thelon's
Chant and Tourach's Chant: saved entry actor, payment at resolution, decline,
missing/opposing/phased creatures, source departure/return/phasing/control,
phase cycles, ability removal, trigger copies, clones, doubling and damage
prevention. `CastingAndUpkeep` covers full mana costs, casting timing, rollback,
entry timing and each colored upkeep payment/sacrifice. `IndependentEvents`
covers two sources and both saved actors with four pending triggers.
`NativeLand` covers real RON Swamp/Forest play and illegal second land exclusion.
`Invalid` and `Inventory` cover declaration restrictions, lexical payment
results, complete nested roles and strict loading. Existing land-entry tests
retain actor/controller distinction, subtype checks and capture/LKI contracts.

## Beginning-of-combat step patterns

`EachBeginCombat` matches `EvtBeginCombat` for either player.
`YourBeginCombat` also checks the source controller at event detection.
Both use the shared `TurnStepPattern` service and support `At`, `TurnStepBegins`,
and `AtNext`. They do not restrict detection to the first combat of a turn.
Native combat-start execution emits the event before priority (CR 507.2,
603.2b). A skipped combat emits no event (CR 603.2g).

`TurnStepBegins` captures the event player for each pending trigger. Source
control changes, departure, return, or ability removal after detection do not
change that saved player or the original trigger controller (CR 603.3a).
Copies retain event captures and use their own controllers. Clones retain
separate pending state. The existing APNAP stack placement applies (CR 603.3b).
Inactive or phased sources cannot generate new ordinary triggers. An absent or
phased source cannot receive a temporary layer effect; a returned source cannot
receive an effect for its previous incarnation (CR 400.7). End-of-combat layer
bonuses end at the end of the combat phase (CR 511.2).

Evidence: `TestBeginCombatPatterns`, `TestBeginCombatDelayedAndAPNAP`,
`TestBeginCombatSkippedNativeStep`, and the `TestRONBeginCombat` tests.
The engine delayed patterns are tested independently; the RON adapter does not
add a new delayed combat-start spelling.

## Live grants of declared triggered abilities

`GrantTriggeredAbility(PermanentRef, *GenericTriggered)` and
`GrantEachTriggeredAbility(PermanentSet, *GenericTriggered)` are static rules.
They require a declared event or step pattern and a battlefield trigger.
Definition validation checks the subject/set and the complete granted ability
in a fresh binding scope. Nil declarations, unbound reads, undeclared targets,
nil effects and zone-source triggers are rejected. These declarations follow
the engine's immutable-definition convention.

The donor context selects current recipients in layer 6 (CR 613.1f). Each
recipient receives its own ability object, source and controller. Ability UUIDs
are derived from the grant identity and both incarnations. Repeated layer
applications do not accumulate abilities or change UUIDs. Separate donors grant
separate instances; returned donors or recipients produce new identities.
Phasing preserves identities but suspends application. Attachment movement
changes a host grant's recipient. Controller restrictions use the donor's current
controller; a trigger's `You` uses the recipient's controller at detection
(CR 603.3a). Removal and grant order use the existing layer lifecycle and
timestamps (CR 613.7); removal after a grant removes the granted trigger.

Patterns, target declarations, event capture, APNAP placement, intervening-if
checks, source incarnation capture, copied stack objects and resolution frames
remain in the shared trigger services. Pending triggers survive grant removal
(CR 113.7a). Their permanent actions cannot affect returned or phased recipients
(CR 400.7, 702.26b), while independent player instructions can still occur.
Resolution payments use the existing transaction and saved-controller contracts.
Energy Flux sacrifices; Tabernacle destroys and therefore permits regeneration
and indestructibility. A copied sacrifice controlled by another player cannot
sacrifice a recipient that player does not control. Destruction has no such
controller restriction. No simultaneous sacrifice batch is created by separate
upkeep triggers. This service adds no replacement ordering or LKI query.

`TestGrantedTrigger*` covers declaration validation, recipient/event identity,
stable UUIDs, new incarnations, controller-dependent selection, phasing,
ability removal, multiple donors, target scopes, intervening-if checks and
independent cloned pending frames. `TestRONGrantedUpkeep*` adds complete RON
consumer mappings, casting/payment rollback, native upkeep, attachment/control
changes, original and copied triggers, destruction/regeneration distinctions
and collection rejection. Low-level event fixtures apply layers after phase
changes before detection, as native step execution does.

## Continuous public visibility

`RevealAllHands() StaticRule` marks every player's current hand as public.
`RevealAllLibraryTops() StaticRule` reuses `RevealTopCardsOfAllLibraries()`.
Both use the existing battlefield static-effect lifecycle and layer-6 source
ability checks. They introduce no resolving action, target, choice, event,
replacement, zone move, or saved object reference. Control changes do not change
which players are affected. Phasing or departure suspends visibility; phasing in
or a new entry restores the live rule. Source ability removal suspends the rule
until the removal expires. Multiple active sources retain visibility when one
leaves. The World rule can remove an older source.

Hand visibility is rebuilt with other per-cycle `GameRules` state.
`Game.IsHandRevealed(uuid.UUID) bool` and `GameRules.IsHandRevealed(uuid.UUID) bool`
read that state without mutation. `GameRules.AddRevealedHand(uuid.UUID)` registers
it for the current pass. Clones copy the hand and library-top maps independently,
so queries before a new layer pass retain the original visibility. Existing
`Game.IsTopCardRevealed` and library inspection primitives retain their contracts.
No private inspection history is made public by these rules.

Interactive snapshots include the current hand when it is public and redact it
again when the rule ends; hand counts remain available. Draws, discard and other
hand moves change the next snapshot's contents. No hand card moves because of
visibility (CR 400.2, 402.3, 611.3, 701.20b). These rules do not grant permission
to cast cards or change the order of a library. Library announcement timing and
library object visits are unchanged; this slice adds no library-target reference
or intermediate announcement display.

Evidence: `TestPublicVisibilityLifecycle`, `TestSnapshotPublicHands`, and
`TestRONPublicVisibilityCards`, `Lifecycle`, `Contents`, `Invalid`, and
`InventoryAndAtomicLoad`. The RON tests compile complete Field of Dreams and
Revelation records. They cover casting payment rollback, pre-resolution timing,
both players, current contents, empty libraries, phase cycles, control changes,
ability suppression and cleanup, departure/return, independent clones, World
state checks, semantic roles and atomic rejection of invalid collections.

## RON referenced damage sources and source tap events

The RON singular `PermanentDealsDamage` adapter reuses `DamageSourceOf` and
`DealDamageFrom` (CR 120.2b, 120.3f, 608.2h). Its specified permanent supplies
source characteristics and lifelink controller. The action controller does not
replace the damage source. Source, host, bound target and captured event reads
retain their existing incarnation contracts. Information reads use
`PowerOfWithLKI`; an absent recipient cannot receive damage. A target that is
illegal at resolution still follows the normal target legality rules.

Tracker's two actions execute in order. Each power is read when its action runs.
State-based actions occur after the ability resolves, so lethal damage from the
first action does not remove the target before the second action. These actions
are not a simultaneous damage batch. Existing prevention and replacement
services process each damage event.

The source tap adapter reuses `BecomesTapped(This)` (CR 701.26a). A repeated tap
of a tapped object and entry in a tapped state do not produce this transition.
The trigger uses its detection controller and source incarnation after departure,
return, control change or phasing. A mana ability resolves immediately; its tap
trigger waits for normal trigger placement and resolution.

`TestRONReferencedDamageTracker`, `City`, `BindingsAndLKI`,
`CastingAndLandPlay`, `Invalid` and `InventoryAndAtomicLoad` supply RON evidence.
Existing `TestDamageSourceOf*` and `TestSourceBecomesTapped*` supply engine
contracts. This slice changes no engine service or replacement ordering.

## RON adapter for singular destruction outcomes

RON `MustCost(DestroyPermanent(ref))` with an adjacent paid-result branch uses
existing `DestroyWithResult(ref, DestroyedThisWayInto(binding))`. It does not
change the mandatory sacrifice payment contract. Actual destruction writes one;
regeneration, indestructibility and an unavailable original incarnation write
zero (CR 701.8, 701.19). This distinction implements Cosmic Horror's "destroyed
this way" clause. Damage after destruction uses the shared saved source
information and prevention pipeline. A pending upkeep retains its controller;
a copied ability uses its own controller and result frame.

The adapter creates a fresh typed binding for each declaration and factory build.
Nested branches cannot replace an outer result. Illegal targets prevent the
spell from resolving before either branch runs (CR 608.2b). Delayed contexts
continue to use the engine's saved-reference rules (CR 603.7c). This slice adds
no combat participant capture, destruction cost type, or replacement rule.

## RON selected discard payment reuse

RON bare `DiscardACard` costs use the existing `DiscardCost(1)` transaction.
The payer selects a hand card once; the transaction saves its hand visit,
checks the full payment on an isolated game, and commits the locked selection.
Failed mana, tap, target, hand choice or discard prevention leaves live payment
resources unchanged. Effect-only discard replacements do not apply to costs.
Copies do not repeat activation payments. No new engine execution service is
introduced.

Optional resolution payments use `IfPaid` or `UnlessPaid` with the bound payer.
They require acceptance and full payment. Decline, an empty hand, an invalid
selection or prevented discard takes the unpaid path. Each nested invocation
keeps its own outcome; a nested payment cannot replace an outer result. The
resolving frame supplies the original ability controller through departure,
control changes and phasing. Copies use their own controller; player targets
use their declared identity. These adapters add no saved discarded-card result.

Rules: CR 118.3, 118.10–12, 601.2h, 602.2b, 608.2c/g, 701.9 and 732.1.
Evidence: `TestRONChosenDiscardCost*`, existing selected-discard transaction,
visit, replacement and batch tests. Real RON clauses are tested separately
from complete-record coverage. Armorer now has complete-record tests in
`TestRONResolutionChoiceArmorer`; War Machine and Land's Edge still reject.


## Fixed alternatives during resolution

`ChooseAtResolution(PlayerRef, ...ResolutionOption) ResolvingEffect` selects
one available alternative while the effect resolves (CR 608.2c-d). This service
is for instructed resolution choices. Modal spells and modal abilities keep
their announcement mode contracts (CR 700.2a-b). It does not add targets,
change payments, or provide priority during resolution.

The declaration copies its option slice and stores no selected index. Each
`ResolutionOption.When` is a pure condition checked before the choice, using
the resolving source incarnation and controller. Callers must supply all
restrictions that make an option unavailable. The chosen player uses the typed
player reference. No available option or missing player performs no action;
one available option needs no prompt. Invalid indices select the first
available option, following the existing engine choice policy.

The body receives the same `EffectContext`. It retains target group identity,
source incarnation, saved information and event bindings. Copies and clones
use their own frames. Conditions precede bodies in definition inspection;
branch binding writes cannot validate reads in another branch or after the
choice. Declaration validation rejects nil references/bodies, empty labels,
empty option lists and unbound target/result reads. It inspects every branch.
Existing action pipelines own replacements, batches and rollback.

RON lowers the complete Dwarven Armorer and Urza's Avenger records through
this service. Armorer's payments occur before its counter choice; illegal
resolution targets prevent the choice. Avenger's choice and P/T change occur
at resolution, with a separate choice on each copy or activation. Ability
choices are bound to one adjacent layer continuation. Temporary effects keep
the original permanent incarnation and their normal expiry (CR 400.7,
611.2a-c, 613.1f, 613.4c, 702.26). Phasing and departure use the existing
reference rules; no new LKI or simultaneous-operation policy is introduced.

Evidence: `TestResolutionChoiceDispatch`, `AvailableOptions`, `Declarations`;
`TestRONResolutionChoiceArmorer`, `Avenger`, `PendingInstances`, `Contexts`,
`Invalid`, `InventoryAndAtomicLoad`; existing chosen-discard transaction,
counter replacement-order, temporary keyword and next-upkeep tests.

## Contextual battlefield collection composition

`UnionPermanents` and `IntersectPermanents` combine typed `PermanentSet` operands.
Their copied operand slices contain declarations, not selected objects. Queries
read each operand in the supplied EffectContext, then emit each live incarnation
once in battlefield order. They exclude phased and stale members. Empty operand
lists and nil/unbound nested declarations reject in `ValidateCardDefinition`.
`AttachmentsOn` selects current attachments whose saved host incarnation matches
a current host-set member. It does not infer host control from attachment control.
`PermanentsControlledByPlayers` combines a typed PlayerSet with a characteristic
filter; `OpponentsOf(You)` reads the current resolving controller. These queries
make no choices, pay no costs and keep no selected state in declarations.

`TestPermanentSetComposition` verifies ownership/control separation, duplicate
elimination, input-slice isolation, live control changes, phasing and clone
isolation. `TestPermanentSetAttachments` verifies independent host and attachment
control, live reattachment, clone isolation, host phasing and stale attachment
incarnation rejection. `TestPermanentSetPlayerControllers` verifies pure repeated
queries, controller-local opponents, departure without LKI membership, new-object
membership and nil nested player declarations. `Declarations` verifies all new
set nodes reject empty/nil operands and unbound player targets.

RON lowers contextual predicates in resolving tap, untap, return and destruction collections;
its permanent target owner adapter uses the existing `TargetYouOwn` service.
`TestRONContextualCollectionObelisk`, `ObeliskLegality` and
`CastingAndInventory` cover the full ATQ Obelisk of Undoing record: casting,
metadata, six-mana/tap payment, illegal ownership and unpaid rollback, self target,
source departure, changed control, shroud, phasing, returned incarnations, copied
controller legality, pending clones and resolution timing.
`RemoveEnchantments` and `RemoveLiveMembership` cover the full LEG record:
owned return before destruction, independently controlled Auras, owned and foreign
Auras on controlled/attacking hosts, removal from combat, reattachment, control,
phasing, indestructibility, copied controllers and clone-local resolution.
`Batches` verifies overlapping union branches do not expose intermediate return
or destruction states to state-trigger observers. `Declarations`, `StrictLoad`
and `CastingAndInventory` verify syntax rejection, newly valid Boolean ownership,
factory isolation, engine validation, atomic rejected loads and nested syntax roles.

Rules: CR 108.3, 110.2, 303.4b/e, 400.7, 608.2b/c/f/h and 702.26b-d.
Existing ReturnEachToHand, DestroyEach, DestroyEachWithoutRegeneration and
battlefield/target services perform all actions. This slice changes no replacement
ordering, optional choice, target identity or LKI policy. It does not establish
static, entry, replacement-operation or cost contexts for contextual collections.

## Saved mana ability activation payments

`ManaActivationPayment` and `ManaTap.ActivationPayment` preserve the exact color
counts chosen by `SolveMana`. Payment search considers all legal allocations,
including generic costs, colored pips, hybrid pips and existing color conversions.
It deduplicates equivalent allocations. Queries make no player choices and do
not change pools, RNG, activation counts or tapped state. Restricted mana remains
unavailable for ability costs, even during payment for an eligible spell.

Before one planned activation starts, execution checks the current source
incarnation, ability, controller, tap legality, output choices and saved payment
against the current reduced cost. A bad or stale allocation fails before any
resource is spent. Source departure and return invalidate a discovered plan;
phasing out prevents activation and a phase cycle retains identity. A nil payment
or zero source incarnation preserves compatibility with earlier callers.
Cloned games execute a saved plan with independent pools and permanent state.
Casting and activation keep their existing payment transaction and rollback
boundary. Direct `ApplyManaSolution` retains its sequential execution contract;
it does not add a transaction around the entire plan.

Evidence: `TestManaActivationPaymentPreservesFinalColors`, `Enumeration`,
`ConvertedPip` and `PlanLifecycle`; full RON `TestRONManaActivationPaymentPrism`,
`ColoredSources`, `Rollback`, `Invalid` and `StrictLoad`. These cover legal
alternative payments, duplicate allocations, restrictions, stale costs and
incarnations, ability removal, phasing, control, clones, compatibility, immediate
production and casting rollback. CR 400.7, 601.2f-h, 602.2 and 605.3 apply.
No replacement order or simultaneous-operation contract changes.

## Temporary triggered grants and rampage reads

`TriggeredAbilityModification(PermanentRef, *GenericTriggered)` extends the
shared continuous-modification service with a declared battlefield trigger.
The outer reference remains in the resolving or static context; the granted
ability has a fresh binding and permanent-source scope. Definition validation
rejects nil, legacy or zone-source triggers, outer target reads and cast-context
reads. A spell can grant a trigger to its declared permanent target without
turning the spell's own source into a permanent (CR 113.3, 603).

Resolution captures the recipient incarnation (CR 400.7). Layer 6 installs a
fresh ability object with a stable per-installation UUID and current recipient
controller (CR 613.1f). Static installation derives its UUID from the donor,
recipient and modification identities. Prototypes remain immutable. Separate
installations remain separate; repeated layer passes do not accumulate grants.
Ability removal and later grants follow timestamps (CR 613.7). Phasing suspends
application without changing identity (CR 702.26b). Pending triggers retain their
source/event frame after grant expiry or removal (CR 113.7a); clones retain
independent pending and continuous state.

`HasRampage PermanentFilter` reads current triggered abilities, including base
and granted rampage effects. RON `AnyRampage` is a bare ability-class symbol in
`HasAbility/1` and `DoesntHaveAbility/1`. RON temporary `Rampage/1` grants use a
nonnegative integer and the existing becomes-blocked and rampage services.
Rampage amounts are read once; bonuses count blockers at trigger resolution
and each instance triggers separately (CR 702.23a-c). No replacement policy,
LKI fallback for actions, or new damage batch is introduced.

Evidence: `TestTemporaryTriggerLifecycle`, `Declarations`, `SpellScope`,
`RecipientController`, `StaticLayerOrder`; `TestRONTemporaryRampageCompleteRecords`,
`RapidFireLifecycle`, `GabrielChoices`, `Invalid`, `StrictLoad`, `TimingAndShapes`;
existing `TestGrantedTrigger*`, `TestRampage*` and RON rampage tests. The full
Gabriel Angelfire and Rapid Fire records have separate behavior mappings.

## Next-event prevention for a captured permanent source

`PreventNextDamageFromPermanent(PermanentRef, PlayerRef, PermanentFilter)` uses
the shared source-prevention replacement and damage batch pipeline. Installation
requires a live, matching permanent and a valid player. Definition validation
rejects nil, unbound and undeclared references. Resolution saves the source
incarnation and recipient. It cannot restore a departed or phased subject with
LKI at installation.

Damage matches the saved source incarnation. Current characteristics or retained
departure/phase information supply the property check (CR 113.7a, 400.7,
609.7b, 615.8–9, 702.26b). Nonmatching sources, other recipients, missing source
information and nonpositive amounts do not consume the shield. Control changes
and phase cycles preserve it; returning the physical card cannot inherit it.
Cleanup removes it. Consumption remains separate in game clones.

The affected player chooses among competing replacements through the existing
pipeline (CR 616.1). If another prevention removes all damage, this shield stays
unused. If this shield is selected, it prevents all currently matching packets
from the source to that player in the simultaneous event before damage executes.
Other sources and recipients remain independent. Later events can use other
installed shields, including shields from copies of activated abilities.

Evidence: `TestNextPermanentSourcePreventionIdentity`, `Declarations`,
`PropertyRechecks`, `ReplacementOrder`, `Simultaneous`, `OneSimultaneousEvent`,
`IndependentBindings`, `MissingInformation`, and
`TestRONNextPermanentSourcePrevention` for complete Prodigal Sorcerer and Orcish
Artillery RON records. These tests establish the permanent-source prerequisite;
source choice, spell sources, source continuity on spell entry, command-zone and
pending-object candidates remain separate contracts. The engine currently models
unpreventable damage with a permanent-recipient rule; this player-recipient
service adds no unpreventable-player or unpreventable-source rule.

## Repeatable fixed mana conversion planning

`ManaSystem` now discovers non-tap activated abilities with fixed, unconditional
mana-only bodies and fixed mana-only costs. The reduced cost must consume at
least as many mana units as the body produces. X costs, activation limits,
arbitrary activation conditions, targets, choice costs, and mixed bodies other
than the deferred history shape below remain outside automatic conversion
planning. Explicit activation keeps its existing behavior.

The solver can use a conversion repeatedly, before or after a tap ability on the
same source. Only a tap consumes that source's tap availability. Exact activation
payments preserve the colors needed for the final cost. Conversions cannot
increase total mana, and tap sources have finite availability, so visited pool
and tap-availability states give finite search without an arbitrary depth limit.
Preservation score and activation count use lexicographic comparison; repeated
paths can be longer than the number of sources. Source-equivalence pruning stays
active for tap-only sources. Conversion sources retain distinct availability.

Summoning sickness restricts tap-symbol costs. It does not restrict these
mana-only costs, and a tapped source can still pay them (CR 302.6, 605.3).
Execution rechecks the exact incarnation, current controller, source presence,
activation restrictions, current reduced cost and exact saved payment. A phase
cycle preserves identity; departure and return invalidate an old plan (CR 400.7,
702.26). Conversion produces mana immediately and emits an ability-activation
event, but supplies no tapped-for-mana bonus or event (CR 106.12, 605.3b).
Hypothetical mana and X upper bounds do not count conversion as new mana.
Queries and game clones do not change live resources. The existing total-cost
transaction owns casting rollback. This slice adds no new replacement, LKI,
choice, target, or simultaneous-operation service.

Evidence: `TestRepeatableManaPlanning`, `ExactPayment`, `Cycles`,
`ExcludedShapes`, `PlanLifecycle`, `TapEventsAndBonuses`, `CostRollback`, and
`BoundAndPreference`. `TestRONRepeatableManaPayment` uses complete Mountain,
Swamp, Forest, Sinkhole and Hymn to Tourach RON records with a synthetic engine
conversion fixture. It covers automatic repeated payment, target rejection,
stack timing, destruction, random discard, and pending game-clone isolation.
It does not verify the two history-bearing conversion cards.

## Synchronous history-bearing mana conversion

The planner can read fixed concrete-color production alongside
`IfActivationsAtLeast` branches bound to that same ability. These branches must
contain only untargeted, nonmodal `AtNext` instructions, or nonempty sequences
of them. The delayed body runs at its future event, so it cannot change the
current payment. Other immediate side effects, conditional production and
any-color output remain excluded from this planning shape. Existing finite
conversion search, cost bounds and exact saved payments remain in force.

Planned execution pays the saved activation cost, records one activation, emits
one activation event, and resolves the complete original body immediately
(CR 602.2, 605.1a, 605.3b). Fixed output is not also inserted by the fast mana path.
The event's tap flag reflects the actual cost. Non-tap conversion emits no
`EvtTappedForMana` event and receives no tap bonus. Each qualifying activation
creates its own one-shot delay (CR 603.7). Explicit and planned activations share
history; queries, failed payments and copies do not add activations. The existing
total-cost transaction isolates speculative history and delays from live state.

`AtNext` retains the activating controller and source incarnation. Departure
and return cannot substitute a new object (CR 400.7). Phasing preserves identity,
but a phased object cannot pay a sacrifice at the delayed event (CR 702.26).
Control changes retain history, while the old delayed controller cannot sacrifice
a source it no longer controls (CR 701.17a). Ability removal does not remove an
installed delay. Final cleanup resets activation history, and a delay created
after an end step begins waits for a later end step.

Evidence: `TestHistoryManaPlanning`, `MixedExecution`, `ExcludedBodies` and
`ActivationEvents`; `TestRONHistoryManaCards`, `PaymentAndLifecycle`, `Invalid`,
`CostRollback`, `NativeSpellPayment`, `HistoryIdentity` and
`InventoryAndStrictLoad`. Complete real Farrelite Priest and Initiates of the Ebon Hand records supply the conversion
abilities; real Armageddon and Sinkhole records exercise native casting payment.
This adds no target, replacement, LKI or simultaneous-operation service.

## Attached permanent death and RON event captures

`AttachedPermanentDies() creatureDiesPattern` extends the shared death-pattern service
to any attached permanent type. `AttachedCreatureDies()` retains its creature
filter. Both match the source's attachment incarnation before the battlefield
object moves to a graveyard. The shared zone batch preserves attachment,
abilities and characteristics when the Aura and host leave simultaneously.
Phasing, exile, an unattached source and a returned host do not supply a matching
death event. Removing the Aura's ability before detection prevents its trigger;
removal, departure, phasing or reattachment after detection does not erase it.

`WithController`, `WithOwner`, `WithToughness`, `WithPermanent` and `WithCard`
reuse the existing typed event captures. Controller, owner and toughness are
saved for each event. The permanent reference keeps its departing incarnation
for source information. The card reference keeps its first graveyard visit.
It cannot move the same physical card from a later visit. Pending triggers,
copies and cloned games retain separate captured frames. The change adds no
new zone, replacement, damage, payment or LKI subsystem.

RON lowers `WhenAPermanentDies(SinglePermanent(HostPermanent))` to this pattern.
Creature Bond's `DeadGuestPermanentDealsDamage` uses the original Aura as its
damage source and the saved host toughness and controller. It uses the shared
damage pipeline, including prevention and retained source characteristics.
Casting still uses the creature-only Aura target and the normal cost transaction.

Evidence: `TestAttachedPermanentDiesCaptures`, `VisitAndPhase`, `Invalid`, existing
`TestAttachedCreatureDies*`, and `TestRONHostDeathCreatureBond`, `Casting`,
`CastingLegality`, `IndependentEvents`, `RepeatedEvents`, `SavedReferences`,
`DamageSource`, `Invalid`, `ReprintsAndInventory`. All six Creature Bond records
compile and pass definition validation. One complete unique-card gameplay mapping
is added; synthetic event tests do not establish full Puppet Master or
Takklemaggot gameplay. Rules: CR 400.7e, 603.2, 603.6c, 603.10a, 608.2h and 702.26.

## Fixed matching tap costs that permit the source

`TapNMatchingIncludingSourceCost(count int, filter PermanentFilter, text string) Cost`
uses the existing exact-count tap cost with source inclusion enabled. It requires
positive `count` and nonempty text during definition validation. Existing
`TapNOtherMatchingCost` and its binding constructor keep their old behavior.

Payment queries make no choices. The payment transaction chooses exactly N
separate, current, untapped, unphased matching permanents controlled by the payer.
A selected source is permitted. Its summoning sickness does not restrict this
written tap instruction (CR 302.6); a separate tap-symbol cost does. The same
permanent cannot pay two tap instructions (CR 118.3, 118.10). All choices save
incarnations and are checked again before commit. A tap, phasing, return, control
change or filter change can invalidate a choice. Failure spends no other payment
resources (CR 602.2). All selected permanents tap before any tap event is sent.
The shared payment transaction and tap batch own these rules; RON owns no payment
state. No replacement ordering or APNAP contract changes here.

Evidence: `TestTapNMatchingIncludingSource*` covers source eligibility, pure
queries, definition rejection, distinct choices, both combined-cost orders,
stale choices, life rollback, simultaneous event visibility and clone isolation.
`TestTapNOtherMatching*` retains the old exclusion contract.
`TestRONTapMatchingHandOfJustice` covers full real RON casting, native costs,
legality, resolution timing, target identity/phasing, shroud, regeneration,
indestructibility, source departure/phasing, copies and clones.
`TestRONTapMatchingWarMachineActivationClauses` covers both real activation
clauses and expiry, with explicit rejection of the incomplete full record.


## RON any-player activation permission

RON `OtherPlayersMayActivate(AnyPlayer)` reuses
`WithAnyPlayerMay() ActionOption` for ordinary and modal activated actions.
No engine state or payment service changes. The compiler rejects duplicate
permissions before it builds a factory. Each factory builds its own action.

The activating player pays the costs and controls the stack ability
(CR 113.8, 602.1a, 602.2a–b). Resolving `You`, modes and target-controller
restrictions use this player. Damage still comes from the source object
(CR 120.7), with current or saved information for its original incarnation
(CR 400.7, 608.2h). Lifelink therefore uses the source controller, including
when the opponent activates or copies the ability. A returned source cannot
replace its saved information. Source phasing or departure after activation
does not remove the pending ability (CR 113.7a); absent/phased sources cannot
start a new battlefield activation. Existing timing, target checks, resource
transactions and per-source ability limits remain in force (CR 602.2, 602.5).

Evidence: `TestRONActivationPermissionEfreet`, `Identity`, `ActorsAndTiming`,
`ModalChoices`, `IndependentInstances`, `CostRollback`, `TargetsAndPresence`,
`EfreetPreventionAndMembership`, `Invalid` and `StrictLoad`. These tests use
the complete Efreet RON record and synthetic declaration contracts. They prove
independent actor contexts, invalid discard rollback, source LKI and lifelink,
cloned pending resolution, target-control rechecks, simultaneous damage,
protection/prevention, independent factories and strict atomic collection
rejection. Existing mixed-damage tests retain replacement and batch evidence.
This slice adds no new replacement ordering or mana-solver contract.

## RON combat-pair permanent reference alias

RON `Trigger_ThatOtherPermanent` reads the same typed opponent binding as
`Trigger_ThatOtherCreature` (blocking and pair unions) or
`Trigger_TheBlockingCreature` (blocked-by events). The existing engine patterns
and `WithOpponent` capture remain unchanged. Each factory and pending event
keeps its own binding; `AtNext` retains that event frame for end-of-combat
destruction (CR 509.3b/d/f, 603.7c/e, 400.7). There is no attacker alias or
permission to read these references outside the matching event body.

`TestRONCombatPairReferenceCards` uses complete Battering Ram and Infernal
Medusa records with native combat, both directions, Wall inclusion/exclusion,
combat damage, end-of-combat destruction and Ram's banding duration.
`Identity` covers departure, return, phasing, control/subtype changes,
regeneration, indestructibility, copied triggers and pending clones.
`IndependentEvents` checks two factories and two pending opponents each, shroud
and clone isolation. `Casting` covers costs, rollback, entry and metadata.
`Declarations`, `Union` and `StrictLoad` cover alias direction, supported union
capture, invalid scope/shape, fresh factories, reprints and atomic rejection.
`RemainingGap` keeps Wall of Dust rejected for its separate temporary combat
rule context; its next-turn duration also remains unsupported. These tests have
the prefix `TestRONCombatPairReference`. No new engine API is established.

## RON grouped source combat trigger adapters

The paired RON `WhenACreatureBlocksAnyNumberOfCreatures` and
`WhenACreatureBecomesBlockedByAnyNumberOfCreatures` union accepts one common
`SinglePermanent(ThisPermanent)` subject and one common `IsCreatureType(S)`
filter. It reuses `BlocksOrBecomesBlockedBy(This).OfSubtype(S)` and the existing
new-pair group events. Multiple matching members in one group create one
trigger. A later new group can create another trigger. Subtypes are checked at
detection; later subtype changes or opposing-object departure do not change
a pending trigger (CR 509.3f, 603.2). Illegal pairs produce no trigger. Entry
while blocking produces no blocking trigger, but can trigger the attacked
source's blocked-by group (CR 509.4).

RON `WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent))` reuses
`BecomesBlocked(This)` and the first transition to blocked (CR 509.3c).
Additional blockers and per-pair events do not repeat it. The body can read
`Trigger_ThatCreature` through the pending ability's source incarnation.
Neither adapter invents one opponent from a group or supplies a defending
player. Source return cannot redirect a pending modification. A phased source
cannot receive that modification; a phase cycle retains identity. Control
changes do not redirect it. Copies and clones retain separate execution state.

No engine event, replacement, cost, LKI or simultaneous-operation service changes.
`TestRONGroupedCombatSoldier`, `Identity`, `CastingAndEntry`, `Damage`,
`LegalityAndMatching`, `Declarations`, `StrictLoad`, `BecomesBlocked`,
`BecomesBlockedIdentity`, `InvalidBlockedDeclarations`, and `SpittingSlugBoundary`
cover the adapters, complete Dwarven Soldier record and remaining source gap.

## Two-branch optional resolution payment

`PayOrElse(PlayerRef, CostGroup, paid ResolvingEffect, unpaid ResolvingEffect)`
uses the same affordability checks and atomic total-cost transaction as
`IfPaid` and `UnlessPaid`. It offers one payment and executes exactly one branch
in the existing effect context. Decline, lack of resources or transaction
failure selects the unpaid branch (CR 118.2–3, 118.5, 118.12). A missing payer
executes neither branch. Zero costs still require acceptance. Branch results
are local to the invocation; nested payments cannot change the outer result.
No payment-result slot or separate payment service is added.

Both bodies contribute effect properties. Declaration validation checks the
payer, explicit cost group and both bodies, and keeps assignments inside their
own branch. Existing one-branch callers retain their behavior. The RON adapter
currently permits fixed mana costs for two-branch continuations. This engine
service accepts the existing supported resolution cost groups.

`TestResolutionPaymentBranches*` covers success, decline, insufficient resources,
zero payment, nested outcomes, invalid declarations, branch scope, combined
cost rollback, absent payers and independent clones. RON gameplay and copying
are covered by `TestRONPaymentBranches*`. Combat collection grants reuse
`BlockersOf`, `BlockedAttackersOf`, union/intersection, `GrantEachKeyword` and
`Until`; they add no LKI, replacement-order or simultaneous-operation contract.

## Activated ability targets with source types

`DeclareActivatedAbilityTarget` declares a singular typed target with a source
card-type restriction. Native battlefield and graveyard activations mark their
stack object as activated; triggers and spells do not match. Mana abilities
resolve immediately and provide no target (CR 605.3b). The graveyard activation
path now uses the same mana classification and immediate resolution as the
battlefield path. Copies and game clones
preserve this classification. Target identity is the stack object's ID, never
the source card's ID. Multiple abilities from one source remain distinct.

The existing target service checks legality before payment and at resolution
(CR 608.2b). The original battlefield source uses current characteristics until
it departs or phases out, then the shared incarnation-indexed information service.
A returned source cannot replace this information (CR 113.7a, 400.7). Graveyard
activations retain their source visit and an independent card snapshot before
payment; self-exile costs cannot erase the source type. A card still in that
visit supplies current information. Target queries do not mutate state or choose.

`AbilityRef` keeps ability reads separate from spell/permanent reads.
`ControllerOfAbility` reads the live ability controller, including copied
abilities with another controller. Source control changes do not change the
payer. An absent ability supplies no player. `CounterAbility` uses the existing
counter service (CR 701.6a–b); it cancels only that object, leaves its source
alone, and does not refund activation costs. No replacement, APNAP, or
simultaneous-operation rule changes in this contract.

Evidence: `TestActivatedAbilityTargetContract`, `Validation`, `Graveyard`,
`CopiesAndController`, `SacrificeCost`, `GraveyardMana`, and `SourceTypesAndRollback` cover native
activation, exclusion, current/retained types, incarnation identity, phasing,
zone costs, controller reads, independent copy targets, cloning, declaration
validation and rollback. `TestRONAbilityTargetCards`, `AyeshaCasting`,
`Declarations`, `RustCopy`, `Invalid`, and `BadPlayers` cover complete source
records and compiler scopes. Ayesha's Banding grant also maps to existing
`TestBandIncomingDamage*` and `TestBandOutgoingDamage*` engine contracts.

## RON source-leaves event integration

RON source departure declarations use the existing `LeavesBattlefield(This)`
pattern. The event reads the departing object's abilities and controller before
movement (CR 603.6c, 603.10a). Bounce, destruction and exile can trigger it.
Phasing does not trigger it (CR 702.26d). Ability removal before departure
prevents detection. Removal after detection does not cancel the pending ability.
The pending source incarnation and retained characteristics remain separate from
any later incarnation of the same card (CR 400.7, 608.2h).

Goblin Shrine's `Trigger_ThatPermanent` lowers to the typed source reference in
this one event context. `DealDamageEach` uses source LKI and one simultaneous
recipient batch. A returned Shrine cannot replace the original source's lifelink
or controller. The conditional bonus reads the current Aura host, requires both
Basic and Mountain, and applies only to Goblin creatures of either player.

Hazezon Tamar's entry trigger uses the existing `AtNext(YourUpkeep, ...)` service.
The delayed ability keeps the entry trigger's resolving controller even if the
source departs, phases or returns under another controller (CR 603.7b, 603.7e).
Its token count reads that player's live lands at delayed resolution. Departure
exile selects every live Sand Warrior at resolution, including another source's
creatures and creatures that entered after departure. `ExileEach` uses one batch;
shroud and indestructible do not stop this nontargeted action. Phased members are
absent. Tokens leave the battlefield and then cease to exist under the existing
state checks.

Evidence: `TestRONSourceLeaveCards`, `Casting`, `Shrine`,
`ShrineSourceInformation`, `ShrineMembershipAndBonus`, `IndependentEvents`,
`Hazezon`, `HazezonCollection`, `Invalid`, `StrictLoad` and `Inventory`.
`TestExileEachUsesOneZoneChangeBatch`, `IdentityAndRegistration`,
`SkipsPhasedAndReturnedTargets`, `QueuesSourceLeaveTrigger` and
`TokenDoesNotRemainInExile` remain the shared engine evidence. This integration
adds no replacement ordering, APNAP, cost or zone-movement service.

## RON ordinary draw-step event integration

RON `TriggerA(AtTheBeginningOfAPlayersDrawStep(players), body)` reuses the existing
`TurnStepBegins(EachDrawStep, binding)` event service. `AnyPlayer` is unrestricted;
`SinglePlayer(You)` adds `EventPlayerIsController` at detection. A fresh factory
creates a fresh player binding. Pending events and copies save the event player;
`You` in resolution reads the ability controller. No condition is checked again
for this ordinary trigger.

The native timing contract remains CR 603.2b, 603.3a and 504.1–2: detection at
step start, normal draw, APNAP placement, then priority and resolution. A phased
source cannot detect an event (CR 702.26b); its departure or phasing after
detection does not cancel an independent triggered ability (CR 113.7a).
`TestRONOrdinaryDrawStepBindings`, `EventLifecycle`, `NativeTiming`, `Copy`,
`Declarations` and `InventoryAndBoundaries` establish this adapter's integration.
The lifecycle test uses the unchanged Armageddon Clock event with a test body;
it does not establish Clock's missing counter-value or activation contracts.
No new engine rule, damage batch, replacement ordering or payment service is added.

## Counter information values

`CountersOnWithLKI(subject PermanentRef, counter core.CounterType) ResolutionValue`
reads one counter kind for the exact permanent incarnation (CR 608.2h).
It uses the existing permanent information reference and snapshot service.
Supported subjects are `This`, `AttachedPermanent`, a saved permanent binding,
and a singular permanent target. Nil, unsupported and unbound subjects and
invalid counter kinds fail definition validation. Missing information supplies
zero. This declaration does not panic for an invalid counter kind.

A live object supplies its current count when the value executes. Departure
and phase information supplies the saved count when the original object is
absent. Counters cease to exist on zone change (CR 122.2); the retained count
is information and does not put counters on the card or its new incarnation.
A later return and second departure cannot replace the old count. Phase-in
restores current reads. The query cannot make a departed or phased object
available for an action. Existing `CountersOn` keeps its live-only contract.

Counts from zero through 255 remain integers without a narrow conversion.
Queries make no choices and change no state. Factories, stack copies and clones
keep their reference identities and independent game state. Counter placement
and damage continue through shared replacement and simultaneous batch services.
This slice changes no replacement, payment or event scheduling contract.

Evidence: `TestCounterInformationLifecycle`, `Declaration`, and
`BoundsAndPurity` in `pkg/mage/composable_counter_information_test.go`;
`TestRONCounterInformationSource`, `Targets`, `HostAndEvent`,
`ReplacementOrder`, `InventoryAndBoundaries`, `Invalid`, and the Clock tests
in `pkg/mage/dsl/ron/counter_information_test.go`. Target legality is checked
before resolution (CR 608.2b); information reads after an earlier instruction
moves a legal target use the retained count (CR 608.2c/h).

## Live player conditions for restricted RON activations

`PlayerIsPermanentOwner(PermanentRef) StateCondition` and
`PlayerIsPermanentController(PermanentRef) StateCondition` compare the player
in `StateContext.ControllerID` with the current owner or controller of `This`
or `AttachedPermanent`. Definition validation rejects all other subjects.
The shared `resolveStatePermanent` query checks source presence, phasing,
optional source incarnation and the attachment's saved host incarnation.
An unknown player, absent source, stale attachment or phased subject fails.
No LKI grants activation permission (CR 602.2, 602.5, 400.7, 702.26b/d).

RON `OnlyPlayerMayActivate` combines these pure conditions with the existing
any-player permission. The source's controller does not acquire permission
unless they satisfy the condition. The activator still pays transactionally,
chooses modes and targets, and controls the stack object (CR 113.8,
602.2a/b). Checks apply before payment; control or attachment changes after
activation do not cancel the stack object, and copies do not repeat permission.
Source information for damage and resolving attachment references retain their
existing contracts. This slice changes no damage, replacement, cost, APNAP,
simultaneous-operation or mana-solver service.

Engine evidence: `TestActivationPlayerConditionIdentity` covers both predicates
and both subjects, owner versus controller, control changes, phasing and phase
cycles, stale host/source incarnations, absence, return, unknown players and
clones. `Declarations` covers nil, target and saved-binding rejection and valid
source/attachment declarations. RON evidence:
`TestRONOnlyPlayerActivationRealModifierProbes` covers actual source permission
nodes, payment, control changes, departure, copies and clones;
`PresenceAndReattachment` covers live hosts and returned attachments;
`TimingTargetsAndInstances` covers all four supported reference combinations,
sorcery timing, activation limits, independent factories and target legality;
`RollbackAndModes` covers payment rollback and mode ownership;
`Inventory` and `InvalidAndBoundaries` cover nested semantic reporting,
declaration rejection, independent complete-card gaps and strict atomic loading.
Complete-card coverage for Merseine and Personal Incarnation remains open.

## Counted and collection next-untap restrictions

`SkipNextUntaps(subject PermanentRef, count int) ResolvingEffect` extends the
existing next-untap service. A nonnegative fixed count applies to the subject's
current controller's next actual untap steps (CR 502.3). Zero has no effect.
Overlapping restrictions count the same future steps; their maximum remaining
count is sufficient. A later one-step restriction cannot shorten a longer one.
The count decreases even if the subject is already untapped or another rule
also prohibits its untap. Skipped steps do not decrease the count. Untapping
through another effect remains legal and does not decrease the count.

Installation requires the original live, phased-in incarnation. Saved
information cannot restore an absent subject. Control changes preserve the
count and change the applicable player. Phasing preserves it; native phase-in
precedes the untap decision. Departure clears it before a returned card can
use it (CR 400.7, 702.26b/d). Clones retain independent remaining counts.
`SkipNextUntap` and player-captured `SkipNextUntapOf` retain their contracts.
The existing decision snapshot and simultaneous untap service remain in use.

`SkipNextUntapEach(subjects PermanentSet) ResolvingEffect` resolves membership
once and installs one-step restrictions for those live incarnations. Later
membership changes do not change the recipients. It has no choices, zone moves
or events during installation. Definition validation rejects nil subjects,
unbound references and negative counts before gameplay.

RON resolving actions add
`PermanentDoesntUntapDuringControllersNextNumberUntaps(ref, Integer(n))` and
`EachPermanentDoesntUntapDuringControllersNextUntap(filter)`. The first uses
existing typed permanent bindings; the second uses contextual collection
queries, including nested nonempty And/Or and supported player restrictions.
Dynamic counts, static/entry-operation/cost declarations and new reference
producers are not established. Targets retain announcement and resolution
legality (CR 608.2b); collection membership is read in instruction order
(CR 608.2c/f). No damage replacement or cost transaction contract changes.

Evidence: `TestNextUntapsCountAndOverlap`, `IdentityAndTiming`,
`CollectionAndDeclarations`, `SavedInformationCannotRestoreObject`;
`TestRONNextUntapsTelekinesis`, `SporeCloud`, `InvalidAndInventory`,
`SharedContextsAndInstances`. Real-card tests execute unchanged complete LEG
Telekinesis and FEM Spore Cloud records through casting, prevention, cleanup
and native untap steps. Existing single-step regression tests remain in use.

## Resolution counter payments

`PayCountersOrElse(player, subject, counter, count, optional, paid, unpaid)`
pays a fixed count from a typed permanent reference during resolution.
`PayChosenCountersOrElse(player, subject, counter, result, paid, unpaid)`
offers a choice from zero through the available count. Both use the shared
counter-cost transaction. A nil branch means no action in that branch.
Invalid counter kinds, negative fixed counts, nil references and unbound
result reads fail definition validation. The chosen result is available only
inside the paid branch, including nested actions; it cannot escape to an
unpaid branch or a later sibling.

The subject must be present and phased in with its original incarnation.
Retained information cannot supply counters for payment. Control changes do
not change subject identity or the resolving ability's player. Optional
payment can be declined; mandatory fixed payment gives no choice. A zero
count is a successful payment from a present subject. Insufficient counters,
out-of-range choices and stale subjects fail without removing counters.
The payment checks identity after the optional decision; the transaction locks
identity across the number choice and checks it again at commit. This uses
CR 118.3, 122.2, 400.7, 608.2d/f and 702.26. A copied ability makes a separate
resolution payment. Game clones keep separate choices, counts and counters.

The count is written only after successful commit and restored after its paid
branch. Nested payments use distinct bindings. No activation payment contract,
replacement order or counter-removal batch contract changes.
Evidence: `TestResolutionCounterPayment*` and `TestRONResolutionCounter*`.

## RON untargeted activation grant adapter

RON static and indefinite activation grants use the existing layer-6 grant
services. The donor selects recipients. Each recipient supplies the source,
controller and costs of its granted ability (CR 602.1a, 602.2, 613.1f).
A pending activation keeps its own actor and source incarnation after donor
removal, attachment movement or control changes (CR 113.7a, 400.7).
No retained information can restore an absent recipient for regeneration.

Static membership follows live selection (CR 611.3); phasing or removal of
the donor's ability suspends its grant. Indefinite resolving grants capture
recipients at resolution (CR 611.2c), survive donor departure, and expire for
a departed recipient. Phasing preserves the captured incarnation (CR 702.26).
Repeated layer passes do not accumulate abilities. Separate grants remain
separate, including copies of Life Matrix's activation. Shared cost transactions
remove counters or mana before the granted body resolves; missing resources
reject activation without changing live state.

`TestRONActivatedGrantZombieMaster`, `LifeMatrix`, `Casting`, `Swampwalk`,
`NoncreatureZombieAndMembership`, `RecipientScopeAndInstances`,
`StaticHostAndSource`, and `CopyRetarget` cover the RON integration.
`Invalid` and `DeclarationsAndStrictLoading` cover lexical isolation, malformed
and unsupported declarations, complete-record definition validation, reprints,
nested inventory roles and atomic collection rejection. Engine grant,
regeneration, payment and layer services are unchanged.

## RON static attack payment adapter

The RON `CantAttackUnlessCost` adapter uses the existing source and attachment
attack-cost rules. Fixed mana
and fixed-count matching sacrifices use `PayAttackCosts`, including its reserved
attacker, aggregate transaction and optional native declaration decision.
Source or Aura ability removal suspends its rule. Removing abilities from the
host does not remove the Aura's separate rule. Current host and control determine
which creature is taxed and which player pays. Source departure and phasing
suspend the rule; a returned host does not inherit its former attachment.

`TestRONAttackCostBrainwash`, `Leviathan`, `LeviathanUpkeep`,
`NativeDeclaration`, `AtomicChoices`, `InvalidAndInventory` and `StrictLoading`
cover both complete records and the supported adapter contexts. Native decline
leaves the creature out of combat; no partial sacrifice or mana payment is
committed. Upkeep payment does not pay Leviathan's later attack cost. Existing
simultaneous sacrifice, replacement, incarnation and transaction services remain
responsible for payment execution. These tests do not claim a new general
multi-attacker declaration rollback contract.

Both native and search attack declaration now read the current attacker after
payment. The original incarnation must still be on the battlefield and under
the declaring player's control (CR 508.1k). A sacrificed, phased, returned or
control-changed source cannot become an attacker through its old pointer. A
successful payment remains paid when its source leaves; no attack event or
attack history is recorded for that source. `TestAttackPaymentSourceSacrifice`
covers both declaration paths. `TestAttackPaymentCurrentIdentity` covers current
control, return, phase cycles and independent clones.
`TestRONAttackCostSacrificedAttacker` covers an animated Leviathan that is also
an Island and legally pays its attack cost with itself. No LKI read can supply
a live attacker. The shared payment and sacrifice replacement contracts remain
unchanged.

### Delayed departure actions and destruction replaced by exile

`WhenLeavesThisTurn` and `WhenDiesThisTurn` keep a watched permanent incarnation
and capture the installing resolution's controller, source and body bindings
(CR 400.7, 603.7b–e). Control and characteristic changes preserve that identity.
Phasing sends no departure event (CR 702.26d). A returned object cannot satisfy
an old subscription or receive an action intended for the old incarnation.
Cleanup removes subscriptions; pending stack triggers remain independent.

Destruction replaced by exile uses the same battlefield-to-exile operation as
explicit exile. It retains the replacement's exile source and sends the normal
zone-change event. Leave subscriptions observe this event; death subscriptions
and creature-death counts do not. Simultaneous destruction uses the existing
zone batch and its former battlefield snapshot. `TestDelayedDepartureReplacedDestruction`
checks single and simultaneous operations, pending timing, destination, counts,
returned incarnations, expiry and cloned games. This establishes no new
replacement ordering or generic delayed-event API.

## Continuous control through each player's turns

`Permanent.ControlledSinceTurnStart(GameReader)` compares the turn of the
latest control gain with `MostRecentTurnStart(currentController)`. It does not
compare with the current global turn. An opponent's turn, including extra turns,
cannot restore the permission. The existing `ControlledSinceTurnStart` state
condition and `WithControlledSinceTurnStart` activation option share this read.
The state condition also requires the original, live, phased-in subject and
matching controller. Haste does not bypass this explicit restriction.

`TurnSystem` owns a fixed two-player array of observed turn starts. Native
`RunTurn` records the active turn before scheduling steps; `RunStepWithPriority`
also records it for direct step execution. Skipping untap still starts a turn;
turns skipped by `Run` execute neither path and do not update this history.
`MostRecentTurnStart` is pure: it reads the current turn for the active player,
the stored start for the inactive player, and zero for an unknown player or a
player with no recorded turn. The public test/setup setters save the previous
active turn before changing turn or player. The array is copied by value with
`TurnSystem.Clone`, without new clone allocations.

Battlefield entry and control changes retain the existing control-gain stamp.
Returning cards start with a new incarnation and stamp. Phasing preserves the
stamp and turn history. No retained information can grant activation permission
to an absent or phased subject. Activation restrictions run before transactional
payment and do not repeat at resolution or when an ability is copied.

Evidence: `TestContinuousControlUsesControllersTurn` and
`TestContinuousControlNativeTurnHistory` cover opponent and extra turns,
control changes, returned objects, phase cycles, direct native steps with skipped
untap, and clone isolation. Existing typed declaration, attached subject,
activation, hand-size, and clone-allocation contracts remain in force.
RON gameplay and pure mana-planning evidence is in `TestRONActivationHistory*`.
Rules: CR 400.7, 500.7, 500.11, 602.1b, 602.2 and 602.5.

## Spell damage-source identity prerequisite

`Game.CaptureSpellDamageSource(stackID)` reads a live spell by its stack object
ID. It rejects abilities, card IDs and absent objects. It saves the card ID,
stack ID, current controller, owner, effective colors and card types in a
`DamageSource`. Permanent spells also qualify; damage capability is not a
condition for this information read. The query makes no player choices and
changes no game state.

`DamageSourceOfSpell(StackObjectRef)` uses the shared typed spell reference and
live stack query. `SnapshotDamageSource(DamageSourceRef, *DamageSourceBinding)`
saves independent information in the existing per-resolution binding map.
`FromDamageSource` reads it after departure. An unsuccessful snapshot writes an
empty source, so a previous value cannot survive. Definition validation checks
nil or ability references, missing declarations, binding names and producer
dominance. The snapshot uses the damage-source role for `This`, so a resolving spell is
valid without a battlefield source.

`DamageSource.StackID()` identifies the exact cast or copy. Native resolution
keeps that ID after popping the spell, and clears it with the resolution context.
Resolution cloning and source snapshots retain it. `SameObject` requires a
matching card ID and either the exact permanent incarnation or the exact spell
stack ID. Unknown identities cannot match. Color and control changes preserve
identity; saved information remains fixed. A later cast or copied spell has a
separate identity. The shared damage actions, simultaneous packets, damage-result
bindings and damage-event source captures carry the source value unchanged.

This establishes a prerequisite for CR 120.7, 609.7a and 615.9 source choices.
It does not yet establish source enumeration, property rechecks, chosen-source
shields, or the special continuation from a permanent spell to the permanent it
becomes. A spell and that permanent remain separate objects under `SameObject`
(CR 400.7); a future shield must link them explicitly. Permanent phasing keeps
its incarnation; departure and return change it. Legacy synthetic resolving
objects without a stack ID keep their information but cannot prove exact spell
identity. Existing permanent-only damage-history APIs retain their contracts.

Evidence: `TestSpellDamageSourceLiveAndResolving`,
`TestSpellDamageSourceSnapshotLifecycle`,
`TestSpellDamageSourceCopiesAndPermanentBoundary`,
`TestSpellDamageSourceDefinitionContract`, and real RON
`TestRONSpellDamageIdentityNativeResolution`, `CopyAndRecast`, `EventSnapshot`.
The RON tests cover Lightning Bolt, Psionic Blast, Earthquake and Hurricane,
including paid casting, rollback, simultaneous player damage, copies, later
casts, source departure, pending event captures and clone isolation. Collection
compilation and whole-card behavior coverage do not increase in this slice.


## Attack history for exact objects and player targets

`AttackedThisTurn` and `DidntAttackThisTurn` use the existing attack-declaration
tracker for each permanent incarnation. Both native declaration paths record
only successful attacks (CR 508.1). Entering combat without declaration does not
count as an attack (CR 508.4). History remains after combat exit, control changes
and phasing. A new zone visit has a new incarnation (CR 400.7).

The turn tracker now retains every attacking incarnation, including two attacks
by different incarnations of the same card. Its existing latest-incarnation
archive and ID-only callers remain in place. Current-turn information reads
cannot lose an earlier attack when a returned card attacks. Clones copy the
additional history; final native cleanup resets it with the other turn history.
History remains available during cleanup priority under the existing lifecycle.

`IfPermanentMatches` reads current or retained information for the original
object (CR 608.2h–i). History does not restore a departed or phased permanent as
an action recipient. Berserk's delayed body checks attack history at resolution;
it is not an intervening-if trigger (CR 603.4). The original target can gain
shroud or change controller without preventing its delayed destruction. Native
regeneration and indestructible rules still apply. Its power bonus is captured
at spell resolution and its trample and bonus end at cleanup.

`TargetPlayerAttackedThisTurn()` restricts the existing typed player target at
announcement and resolution (CR 608.2b). Creature departure, return and control
changes do not change the player who declared the attack. A history reset makes
a pending target illegal. `TargetOpponentOnly()` retains the history restriction
in either option order. Failed target checks occur before casting payment.
Fire and Brimstone retains simultaneous damage, prevention and native spell
movement; overlapping player packets combine through the shared damage service.

Evidence: `TestAttackHistoryReferenceIdentity` (two incarnations, saved-reference
LKI, clone/reset); `TestAttackHistoryReferencePhasingAndCombat` (combat entry,
phase cycle, native cleanup); `TestAttackHistoryPlayerTarget` (option order,
declaration validation, resolution recheck); `TestAttackHistoryBothDeclarationPaths`
(illegal and successful declarations); the complete-record
`TestRONAttackHistoryBerserk*` and `TestRONAttackHistoryFireAndBrimstone*` tests.
No new cost, replacement, combat declaration or intervening-if service is added.

## Filtered blocking requirements

`AllCreaturesMustBlock(subject PermanentRef, filter PermanentFilter)` is a
continuous rule modification. `While` reads the current source or attachment;
`Until` captures the recipient's incarnation. Each installation contributes
one independent requirement for every matching blocker. Layer reapplication
resets the requirement map. Clones copy its filter slices. The rule does not
add an ability to the recipient, so removing the host's abilities does not
remove a separate Aura's requirement. Removing or phasing the source suspends
its static rule; a returned host cannot inherit its former attachment.
Definition validation rejects nil and undeclared subjects.

`Game.CompleteBlockAssignments(defender, proposed)` is a pure declaration query.
It retains a legal proposal if that proposal satisfies the maximum number of
blocking requirements. Otherwise it returns a legal maximum, in deterministic
battlefield and combat order. Both native and search declarations use this
service before combat/history writes and declaration events. This preserves the
engine's existing completion of compulsory declarations; the query itself does
not ask the player to make choices or pay costs. Existing combat pairs remain
fixed. Later rule changes do not undo declared blocks.

Candidate pairs require a current, phased-in attacking creature incarnation, a current
untapped creature controlled by the defender, the existing pair restrictions,
evasion and protection checks, and landwalk legality. Duplicate pairs do not
use extra capacity. Normal, additional and unlimited block capacities are
shared by all requirements. Minimum-blocker restrictions, including menace,
apply before requirements are counted. Multiple compulsory attackers compete
for capacity; independent repeated requirements contribute independently.
The existing `MustBeBlocked` and `AttrMustBeBlockedIfAble` attributes use the
same service. A single-blocker menace declaration is now rejected; its previous
regression fixture intentionally asserted the old engine gap and now checks the
rules result.

A block declared against one legal band member blocks the other members under
CR 702.22h. It uses one declaration slot and satisfies applicable requirements
for those members. Evasion and minimum-blocker restrictions on an indirectly
blocked member do not prevent choosing a different legal member. The planner
keeps direct declarations separate from the resulting band pairs.

The exact search uses admissible requirement bounds and lazy subsets. It has
no depth limit or partial-result cutoff. Complex combinations can require
exponential work; normal proposals with no unmet requirement avoid the search.
The service adds no new blocking cost or player-choice API.

Evidence: `TestBlockingRequirementsBothPaths`, `MaximumAndMinimum`,
`DefinitionAndIdentity`, `CapturedDuration`, `RepeatedAttributes`, `CreatureTypeLoss`,
`Band`, `BandRestrictions`,
`AgainstExhaustiveAssignments` (96 independent small assignment problems),
and `LargeMultiBlockAndSingleCoverage` (32 attackers).
`TestRONBlockingRequirementCards`, `LureLifetime`, `LureAnnouncement`,
`Invalid`, and `StrictRecordsAndInventory` supply complete Lure and Marble
Priest records, all reprints, payment/target rollback, combat damage, prevention,
source/host identity, phasing, ability removal, copy/clone isolation, declaration
roles and atomic loading. Existing combat-event and pair regressions remain in
force. Rules: CR 400.7, 506.4, 509.1a–c/g–i, 611.3a–b, 702.22g–h,
702.26b–g and 702.111.


## Captured collection animation

`SetEachBasePowerToughness(PermanentSet, ResolutionValue, ResolutionValue)
ContinuousModification` sets both base values in layer 7b.
`AddEachCardType(PermanentSet, core.CardType) ContinuousModification` adds one
supported permanent type in layer 4. Both use the existing `layerModification`
executor and definition inspector. Nil sets/values and unsupported type additions
fail definition validation. Existing singular callers are unchanged.

One `Until` installation captures the collection and all values before layer
application. No member is added because of the type change, a later entry, or
later control/subtype changes (CR 611.2c-d, 613.6). Phased-out members are absent
at capture. A captured member retains its effect through control changes and a
phase cycle; a returned incarnation does not inherit it (CR 400.7, 702.26).
Existing types, subtypes, abilities and colors remain (CR 205.1b). Creature type
addition supplies P/T identity. Base values apply before bonuses, counters and
switches (CR 613.4b-d). Clones and cleanup use the existing independent lifecycle.

Evidence: `TestCollectionAnimationCapture` and `TestCollectionAnimationInvalid`.
Real RON gameplay and factory/target isolation are covered by
`TestRONExtendedAnimation*`. Xenic Poltergeist's mana-value read uses the current
card characteristics when its ability resolves, including a full characteristic
copy before resolution; later changes cannot alter the captured base values.
No general card-copy, all-zone characteristic, type replacement, replacement
ordering or new LKI service is established by this slice.

## RON numbered singular target binding reuse

RON's multiple ordinary `TargetPermanent` declarations reuse the existing fixed
singular target services; no engine rule or state is added. Per-declaration plans
map `Ref_TargetPermanent1`, `Ref_TargetPermanent2`, and subsequent names to
separate `PermanentTargetBinding` instances at each factory build. Separate
instances of the word "target" can select the same legal permanent (CR 601.2c).
Target rechecks, partial legality, all-illegal resolution and payment rollback
remain engine operations (CR 601.2e/h, 602.2b, 608.2b). Spell and ability copies
can retarget groups independently. Mode bodies cannot read another mode's plans.

Delayed bodies reuse saved reference frames. A returned object cannot replace a
captured incarnation; a phase cycle preserves it. Information reads keep the
existing exact-object LKI contract (CR 400.7, 603.7c, 608.2h). The new tests cover
separate group actions, repeated-object choices, cast/activation failures,
shroud, phasing, return, control changes, copies with new targets, factory and
clone isolation, trigger-source departure, mode scope and delayed LKI capture.
No new replacement ordering or simultaneous operation is established.

Evidence: `TestRONSingularGroups*`. Complete collection coverage remains separate
from this compiler prerequisite's synthetic RON gameplay evidence.

## Opponent choice during modal activation

A modal activated ability selects its mode and targets before payment. The
opponent selects a declaration marked with `TargetChosenByOpponent` at that
point (CR 602.2b, 602.3). The later announcement path preserves that choice and
validates it without asking the opponent again. The flat stack targets and the
selected `ModalTargets` entry retain the same choice. Affordability checks can
reject an activation before any choice. Failed payment publishes no ability.

Target candidates and resolution legality still use the ability controller and
source. An opponent's choice does not change control restrictions or hexproof.
Returned incarnations, phased targets and control changes can invalidate the
chosen target (CR 400.7, 608.2b, 702.26b/d). Game cloning retains separate pending
target state. Copies use the existing CR 707.10c permission for the copy's
controller to choose new targets; they do not repeat activation payment.

`TestModalActivationOpponentChoosesOnce` covers one prompt, committed mode
identity, affordability, control changes, return, phasing and clone isolation.
`TestRONOpponentTargets*` covers the RON adapters, singular and numbered groups,
partial and all-illegal resolution, malformed choices, source departure, copies,
factory isolation and trigger/mode contexts. Replacement ordering, simultaneous
operations and information reads retain their existing engine contracts.


## Typed singleton references in resolving collections

`SingletonPermanent(subject PermanentRef) PermanentSet` exposes the existing
singleton selection used by granted activations. It stores a declaration, not a
selected object. Nil and unbound target references fail definition validation.
Selection delegates to the reference: source and target references retain their
original incarnation; a live attachment reads its current host, and a departed
or phased source uses the existing saved attachment relation. The selected host
must still be a live original incarnation. LKI cannot supply an absent member.

RON `SinglePermanent(ref)/1` and symbolic `Ref_TargetPermanents` are operands in
contextual resolving collections. `And([collections])/1` intersects and
`Or([collections])/1` unions references, live filters and existing contextual
operands. Results contain each live incarnation once in battlefield order.
`TapEachPermanent`, `UntapEachPermanent`, `DestroyEachPermanent`,
`DestroyEachPermanentNoRegen`, and `PutEachPermanentIntoItsOwnersHand` share this
compiler service. Existing typed target declarations are constructed per factory.
The engine performs tap/untap and zone actions through existing simultaneous
batches. No new payment, replacement, targeting or LKI policy is introduced.

A singular reference can use the existing delayed capture service. Its identity
is saved when the delayed trigger is created; later shroud does not make that
saved reference a new target. Plural delayed target collections remain rejected.
References cannot escape their ability or mode. This adapter does not add entry,
cost, static, damage-collection or target-filter syntax.

Evidence: `TestSingletonPermanentSet*` covers definition validation, source
incarnation, phasing, control changes, pure selection, clone isolation and
simultaneous tap. `TestRONReferenceCollection*` covers all four action consumers,
union duplicates, current filtered plural membership, scope rejection, copied
and independent targets, source/host identities, delayed captures and full RON
Winter Blast partial legality. Existing real-card tests for Candelabra of Tawnos,
Word of Binding, Mana Short, Reset, Feint, Arena of the Ancients and contextual
return/destruction consumers remain required regression evidence. Arena and
Volcanic Eruption remain rejected at Fight and destruction-result count,
respectively. This prerequisite verifies neither complete card.

Rules: CR 113.7a, 400.7, 608.2b/h and 702.26b-d. Existing target bindings,
attachment relations, delayed captures and collection batches supply these rules.

### RON destruction count adapters

RON destruction-and-continuation pairs use existing typed result services.
They do not change engine destruction behavior. CR 608.2b preserves partial
and all-illegal target checks; CR 608.2c preserves instruction order. CR 608.2h–i
and 700.4 supply the pre-destruction creature information and actual deaths.
CR 701.8 and 702.12b exclude regenerated and indestructible permanents.
The shared replacement pipeline applies affected-player choices under CR 616.1.
Exile replacements contribute zero to both graveyard moves and creature deaths.
Tokens contribute to the appropriate count even though they do not remain in
the graveyard. Phased objects and returned target incarnations are excluded.

`TestRONDestructionCount*` covers full Hellfire and Volcanic Eruption records,
X payment rollback, illegal targets, partial and all-illegal resolution, phasing,
returned targets, shroud, regeneration, indestructibility, token moves, competing
regeneration/exile choices, adjacent-result scope, separate instructions and
instances, retargeted copies, pending clones, simultaneous damage callbacks,
no-regeneration destruction and declared filter guarantees. Existing engine
`TestDestroyEachGraveyardResult*` and `TestDestroyEachCreatureDeathsResult*`
remain the reusable destruction-result contract tests.

## Explicit typed damage-source choice

`ChooseDamageSourceFrom(player PlayerRef, sources []DamageSourceRef,
binding *DamageSourceBinding) ResolvingEffect` makes a required choice at
resolution from explicit legal source references. The caller supplies the legal
list. This service is a prerequisite for CR 609.7a enumeration; it does not
establish complete source enumeration or a prevention shield.

References retain their existing current-state, saved-information and LKI
contracts. Missing references are omitted. Exact duplicate objects appear once,
in input order. Spell copies and separate stack visits remain separate even
when their card IDs match. Shroud does not restrict this choice. An explicit
reference to a departed or phased object can still supply information; this
service does not grant permission to include that object in a legal list.

The optional `DamageSourceChooser.ChooseDamageSource([]DamageSource, *Game) int`
player interface returns a zero-based index. Each option has a separate copy of
its card data. Existing players use a bounded numbered choice with source names.
An invalid answer selects the first legal option, as with the existing required
permanent choice. Empty runtime lists and absent players clear the output and
make no choice. Declaration validation rejects an empty declared list, nil
references, nil players or bindings, unnamed bindings, undeclared targets, and
reads before their producer.

The output uses the existing resolution frame and `DamageSourceBinding` map.
Input references are read before the output is cleared, so a previously produced
binding can also be the output. Repeated choices replace the local result.
Clones keep independent maps and card snapshots. The constructor copies the
reference slice; it does not store chosen state in a declaration.

Evidence: `TestDamageSourceChoiceIdentity`, `PermanentAndAbsence`, `Declaration`,
`InvalidAnswersAndRepeatedUse`, `LegacyPlayer`, and `InputOutputOverlap`.
`TestRONDamageSourceChoicePrerequisite` executes the complete Prodigal Sorcerer
RON record with a synthetic event consumer. It covers queued choice timing,
source departure/return/phasing, copied triggers, source-attributed damage, and
pending clone isolation. This adds prerequisite evidence, not complete gameplay
coverage for the eleven still-rejected chosen-source consumers.

## Typed property filtering before damage-source choice

`DamageSourceFilter` and `ChooseDamageSourceMatchingFrom` establish property
filtering over explicit legal source references at resolution. CR 609.7a permits
sources that cannot deal damage and makes this a choice rather than a target.
The caller remains responsible for complete legal enumeration. Color reads use
`DamageSource.HasColor`, including stack color overrides; type reads use its
captured card information, including effective permanent types. The filter
itself makes no game query and does not change state or make decisions.

Current and saved reference contracts remain distinct. A current reference
reads current source properties. An event/snapshot reference reads the supplied
saved information. Departure, return and phasing retain the reference's exact
identity and LKI semantics (CR 400.7, 608.2h and 702.26). Filtering never resolves
a saved card ID to a later incarnation. This establishes choice-time property
reads; it does not establish shield rechecks under CR 609.7b/615.9 or
permanent-spell shield continuity under CR 400.7c.

Engine validation checks color/type operands and nonempty recursive boolean
compositions. Missing sources never match. A list with no matching sources
clears its result without a player decision. Matching references retain input
order; exact duplicates are removed. The existing choice service preserves
separate stack objects, copies, per-frame bindings, input/output overlap,
legacy players and independent clone snapshots.

Evidence: `TestDamageSourcePropertyMatching`, `Choice`, `Declarations` and
`LiveAndRetained`, plus existing `TestDamageSourceChoice*` contracts.
`TestRONChosenSourceFilterRecords` checks complete source records for all eleven
consumers while confirming they remain rejected. `Declarations` checks malformed
RON shapes, unbound operands and invalid color/type arguments. `Inventory`
checks nested roles. `Gameplay` executes complete Prodigal Sorcerer RON events
with the Circle of Protection: Blue filter in a synthetic consumer, covering
queued resolution, recoloring, source departure/return/phasing, copies and
pending clones. This is prerequisite evidence and adds no complete card coverage.


## Combat defending player target prerequisite

`CombatDefendingPlayer() PlayerRef` selects the nonactive player during the
combat phase of a two-player game (CR 506.2). It reads the current turn, not the
spell or ability controller. It is available in every combat step, including
beginning of combat before attackers exist and first-strike damage. Outside
combat, without an active player, or without exactly two players it supplies no
player. Queries do not change game state. Clones use their own turn state.

`TargetControlledBy` also accepts this reference. Existing target enumeration
and resolution checks compare the permanent's current controller with this
player (CR 601.2c, 608.2b). Ownership does not satisfy the restriction. The normal
incarnation, phasing, shroud and hexproof checks still apply. LKI cannot make an
absent or returned target legal. Retargeting a copy uses its own controller for
ordinary target protection, but still uses the nonactive player for this control
restriction. No target, combat, cost or event subsystem is replaced.

RON singular permanent target filters support
`ControlledByAPlayer(SpellDefendingPlayer)` and
`ControlledByAPlayer(SinglePlayer(SpellDefendingPlayer))` in instant/sorcery
`SpellActions` target scopes, including mode and numbered singular target scopes.
The symbol is rejected in activation and trigger targets, static filters,
resolving player references and collections. Nested nonempty `And` uses the
existing target-filter composition. Tuple spell-defender reads and malformed
player wrappers are rejected before factory construction.

Attack and unblocked-attack trigger target filters also support
`ControlledByAPlayer(DefendingPlayer)` and
`ControlledByAPlayer(SinglePlayer(DefendingPlayer))`. These lower to the existing
saved `WithDefendingPlayer` event binding, not the live combat reference
(CR 508.5, 603.3d, 608.2b). Each pending event, factory, stack copy and clone keeps
its own captured player. Later turn and source changes cannot replace it.
No additional reference becomes available outside that event scope.

Engine evidence: `TestCombatDefendingPlayerQuery`, `Missing`, and the existing
`TestTargetPlayerReference*`. RON evidence: `TestRONCombatDefenderSpellTargets`,
`CapturedTarget`, `IndependentTargets`, `ModeAndGroups`, `CopyRetarget`,
`Declarations` and `Records`.
The gameplay probes use all three consumers' exact target declarations and
casting windows with a test-only resolving body. They establish the prerequisite,
not full gameplay for Blaze of Glory, False Orders or Floral Spuzzem. All eleven
complete consumer records still reject other operations. Existing complete RON
Mindstab Thrull and Necrite gameplay remains covered by
`TestRONDefendingPlayer*`. Replacement ordering and simultaneous operations are
unchanged and are not new contracts of this slice.

## Saved player conditions and opposing permanent counts

`PlayerControls(PlayerRef, PermanentFilter)` now accepts `FromPlayer(binding)`
as well as `You` and `ActivePlayer`. Definition validation requires a producer
for the saved binding. Trigger detection reads the event pattern's player
captures without running effects, making choices or changing the resolution
frame. Resolution reads the current frame populated by the existing capture
service. Each pending event, copied stack object and game clone retains its own
player identity. Neither current turn nor source control can replace that player.
An absent player or missing binding returns false. Current matching battlefield
permanents supply the control check; departed and phased objects supply no LKI.

`OpponentControlsMoreThanYou(PermanentFilter) StateCondition` compares the
current matching permanent counts for the condition's controller and opponent.
Equal counts return false. Missing controller or opponent returns false. The
query makes no choices and stores no mutable state. It works in the existing
state-condition contexts, including `WithInterveningIf` and `IfState`.

CR 603.4 and 608.2a require checks at detection and resolution. CR 603.3a keeps
the triggering controller after source departure or control change. No cost,
replacement, simultaneous-operation, targeting or zone-movement behavior changes.
Evidence: `TestPlayerControlsSavedIntervening`, `TestPlayerControlsSavedDeclarations`,
`TestOpponentControlsMoreThanYou`, and the RON upkeep-player tests. Existing
`PlayerControls`, opposing-control, storage-upkeep and event-capture tests remain
regressions. Land Tax and Ghazbán Ogre's unresolved bodies remain compile errors.

## End-step history conditions

`SourceAttackHistoryThisTurn(attacked bool) StateCondition` reads attack
history for the exact source incarnation. With no captured incarnation it
requires a live, unphased source. With a captured incarnation it permits the
original live or phased source, or its retained departure information. Unknown
incarnations fail both the positive and negative condition. Control changes,
combat exit and source departure do not erase an attack declaration. Direct
placement in combat does not record an attack declaration. A returned card's
history cannot replace the old history (CR 400.7, 508.1, 608.2i, 702.26).
The existing live-only `SourceAttackedThisTurnCondition` retains its contract.

`CreatureDiedThisTurnCondition() StateCondition` checks the shared current-turn
creature death count. It includes tokens and either player's creatures. It does
not count noncreatures, bounce, exile, prevented destruction or regeneration.
It makes no choices and does not save a count. Cleanup clears the shared history;
a clone retains its own history. Existing simultaneous death tracking is reused.

End-step intervening conditions run at detection and resolution (CR 603.4,
608.2a). `SourceTappedStateWithLKI(false)` supplies current or retained untapped
state (CR 608.2h); source return cannot replace its old incarnation. A copied
trigger reads its own controller and retains its original source identity.
Information reads do not give permission to act on departed or phased objects.
No new payment, targeting, replacement-order or simultaneous-movement contract
is added.

Evidence: `TestSourceAttackHistoryThisTurn`, `TestCreatureDiedThisTurnCondition`,
existing `TestSourceTappedStateWithLKI*`, and RON `TestRONEndStepSourceState`.
Complete Osai Vultures behavior uses `TestRONOsaiVulturesEndStep`, `Activation`
and `NativeSteps`: native steps for both controllers, one counter per end step,
zero/late/multiple/token deaths, counter payments and rollback, independent
pending bonuses, cleanup, phasing, departure/return, copies and clones.

## Resolution values: player damage this turn

`DamageTakenThisTurn(PlayerRef) ResolutionValue` and
`ArtifactDamageTakenThisTurn(PlayerRef) ResolutionValue` read the selected
player's current turn damage totals. They are pure queries and return zero for
an absent or unresolved player. Definition validation rejects nil and unbound
player references. Each factory, target, copied stack object and delayed context
keeps its own typed player binding.

The values reuse `TurnTrackers` and the damage executor. Damage enters history
after prevention and redirection; life loss and life gain do not enter it.
Artifact history uses the source's effective type when damage occurs, including
retained permanent and spell information. Later source type, control, zone or
phasing changes cannot change a recorded amount. Simultaneous packets contribute
their actual amounts. Clones have independent histories. Native cleanup clears
both totals. Each application reads the current total, rather than a casting-time
snapshot (CR 120.3–4, 608.2b–c/h).

RON resolution values support `DamageDealtToPlayerThisTurn(player)/1` and
`DamageDealtToPlayerThisTurnByPermanents(player, IsCardtype(Artifact))/2`.
The latter is the collection's artifact-source history spelling, including
retained source information; it does not enumerate current permanents.
Other source filters remain unsupported. Existing `Twice` lowers the multiplier.
No static, entry, cost or announcement history-value context is added.

Evidence: `TestPlayerDamageHistoryValues`, `SourceInformation`, `TypesAndSpells`
and `ReplacementOrdering`; `TestRONPlayerDamageHistoryCards`, `Bindings`,
`CopyController`, `DelayedBinding`, `Invalid` and `RecordsAndInventory`.
The real RON mappings cover all rules of Simulacrum and Reverse Polarity.

## Fixed mana increases for static spell costs

`IncreaseSpellCostStatic(filter SpellPredicate, increase core.ManaCost,
controllerOnly bool) ContinuousEffect` adds fixed generic and colored mana to
matching spell costs. With `controllerOnly`, the caster must be the source's
current controller. Otherwise, the rule applies to every caster. Each source
instance contributes once per matching spell, even if several filter alternatives
match. Separate sources accumulate. Definition validation rejects nil filters,
negative mana, X and hybrid declarations. A zero increase is valid.

The shared continuous-effect pass registers each tax with its source incarnation.
Payment ignores absent, phased or returned sources. Control changes use the live
controller. Ability removal suppresses the printed tax through the existing layer
and dependency service. Clones keep separate registration slices. No LKI permits
an absent source to tax a new spell.

The shared spell transaction adds these increases to the complete mana total
before reductions (CR 601.2f). Generic reductions cannot pay colored increases.
Additional costs, announced X and alternate costs keep their existing contracts.
The total stays locked if the tax source leaves during payment. Failed payment
cannot commit mana, sacrifice or stack changes. Copying a spell on the stack
requires no payment. This changes no activation or resolution-payment costs and
adds no replacement effect.

RON supports `PlayerEffect(You, [IncreaseSpellCost(filter, PayMana(mana))])`
and `EachPlayerEffect(AnyPlayer, [IncreaseSpellCost(filter, PayMana(mana))])`
on permanent definitions. Each list must have one operation. `mana` is a list
of fixed generic and W/U/B/R/G symbols. Filters reuse the spell-cost predicate
compiler: `AnySpell`, nonland `IsCardtype`, nonempty nested `And`/`Or`, and
`IsColor` for one of the five Magic colors. Color uses the existing card-color
filter, including color indicators; it does not derive color solely from cost.
The shared predicate compiler also permits these color filters for static generic
reductions. Other player scopes, dynamic increases, nonmana costs, resolving
durations, and activated-ability taxes remain unsupported.

Evidence: `TestStaticSpellCostIncreasePayment`, `Lifecycle`, `Declarations`,
and `StackingAndLock`; `TestRONSpellCostIncreaseDerelor`, `Lifecycle`, `Filters`,
`Invalid`, and `RecordsAndInventory`. Derelor has complete gameplay evidence.
Gloom's spell-tax rule compiles, but all six complete records remain rejected
for `ActivatedAbilityEffect`. No complete Gloom gameplay claim is made.
Rules: CR 400.7, 601.2f-h, 601.7, 611.3a-b and 702.26b.

## Static activation mana modifiers

`IncreaseActivationCostStatic(PermanentFilter, core.ManaCost)` and
`ReduceActivationCostStatic(PermanentRef, reduction, minimum int)` register
per-source rules through the existing continuous-effect service. Definition
validation rejects missing subjects/filters, negative amounts and nonfixed
increases. Each rule reads its exact live source incarnation, current matching
permanent characteristics, or current typed attachment. No LKI grants a cost
modifier. Source phasing, departure and printed ability removal stop it. Host
phasing and return use the existing attachment identity service. Control changes
cannot change which object is enchanted. Separate sources accumulate; cloned
rule slices are independent.

The total mana cost is computed after announced X and before payment. Fixed
increases apply once to the sum of all mana components, followed by generic
reductions. The minimum cannot increase the original cost. Colored mana is
retained, and reductions cannot change X stored on the stack. Ordinary ability
queries and transactional payment, native mana activation, and mana-source
planning use the shared calculator. A mana ability with a tax cannot use the
solver's interchangeable free-source path. The transaction locks the total;
later source departure does not recalculate it. Failed payment leaves mana,
tap state and stack unchanged. Copied abilities pay nothing again. Existing
spell and resolution-payment policies are retained.

Rules: CR 118.7a, 400.7, 601.2f-h, 602.2b, 605.3, 611.3a-b and 702.26b.
Evidence: `TestStaticActivationCostTotal`, `Lifecycle`, `Payment`,
`Declarations`, `SplitAndX`, `HostIdentity`, `ManaPlanning`,
`ColoredAndLocked`; `TestRONActivationCostRealCards`, `Invalid`,
`RecordsAndInventory`, `GloomBothRules`, `PowerArtifactManaAndFloor`.
No replacement ordering or simultaneous zone-action contract is changed.

## Current card P/T from count CDAs

`Game.CardPowerToughness(card Card) (int, int)` is a pure current-object query.
It returns zeroes for a nil card. A live battlefield object uses
`CurrentPower`/`CurrentToughness`, including layers, counters and switching.
A phased object uses its retained layer results. This query is not an LKI or
saved-incarnation reference; a caller that needs departed-object information
must use the existing snapshots and typed references.

`PTEqualsCount(PermanentFilter)`, `PTEqualsControlledCount(PermanentFilter)`
and `ToughnessEqualsControlledCount(PermanentFilter)` now retain inspectable
count declarations. Their signatures and battlefield behavior remain available.
Outside the battlefield, the query evaluates these declarations against current
nonphased battlefield membership. “You” uses the current spell controller on
the stack and the card owner when there is no controller (CR 109.5, 208.2a,
604.3). Unzoned cards, library, hand, graveyard and exile cards use the same
owner context. This includes cards before entry and copied definitions.
Printed `Card.Power`/`Card.Toughness` still supply copiable printed values;
queries do not overwrite them. This service does not add other all-zone
continuous effects or an announcement-time would-be-controller query.

The effect manager applies these counts in layer 7a before setting, modifying,
and switching P/T (CR 613.4). Each battlefield entry or face-up registration
binds a separate effect to the source incarnation. Returning a card cannot
reactivate an old bound effect. Game clones copy bound effects independently.
The layer reset includes prior CDA values, so layer-6 ability removal does not
leave an old layer-7a value on the object. Phased members do not contribute to
counts. Existing filter and zone services perform the queries; there is no
second battlefield store. A zero filter retains its existing match-all meaning.
The native snapshot uses this query for visible hand, graveyard and exile cards.

`TestCharacteristicPTZonesAndController`, `SpellAndCopies`,
`LayersRemovalAndPhase`, `SharedDeclarationsAndClone`, `KindsAndPurity`,
`CountsMembership`, `CopiedDefinitionAndStaleEffect` and `DirectEffectCaller`
cover these contracts.
`TestSnapshotCharacteristicPT` covers native display reads.
`TestRONCharacteristicPrerequisiteGameplay` runs complete RON Giant Growth,
Unsummon and Swords to Plowshares against a synthetic count-CDA fixture,
including casting rollback, cleanup, original-controller LKI and clones.
`TestRONCharacteristicCountReprints` now validates and strictly loads all twenty
records for the five count-CDA consumers. `TestRONCharacteristicCountGameplay`,
`Membership`, `ZonesLayersAndLKI`, and `SpellController` execute complete RON
Dakkon Blackblade, Keldon Warlord, Nightmare, Plague Rats and People of the Woods.
They establish owner and controller queries, current membership, Wall/name/type
filters, dual lands, spell payment rollback and entry timing, flying and legendary
metadata, phasing, copy and clone isolation, source return, ability removal,
bonuses, counters and original power/controller LKI for exile life gain.
`TestRONCharacteristicCountInvalid` rejects malformed, duplicate, unbound,
incompatible and unsupported declaration contexts. Inventory tests retain strict
collection rejection. This adapter adds no new engine execution or payment rule;
existing layer-set/switch and incarnation contracts remain covered by the engine
tests above. RON arithmetic and entry-result CDAs remain unfinished.


## Live-object damage-source enumeration prerequisite

`Game.LiveDamageSources() []DamageSource` establishes the permanent and live
spell categories in CR 120.7 and 609.7a. It returns unphased permanents in
battlefield order, then spells in stack insertion order. It includes lands,
objects without damage abilities, and permanent spells. Abilities are not
spells. Spell copies and separate stack visits retain separate stack identities,
even with the same card ID. The query reads current layer results without
changing state, paying costs, making choices or running a layer pass.

Source information reuses `damageSourceFromPermanent` and
`CaptureSpellDamageSource`. Colors include current overrides and permanent
types/subtypes use current characteristics. Snapshots have independent card
data. Face-down permanents have no name, color, printed keyword or subtype;
the shared permanent and LKI snapshot helper now hides the underlying card
characteristics (CR 708.2a). Ownership and incarnation identity remain available.
Phased permanents are excluded (CR 702.26b). Departed objects are excluded from
this live list even when the engine retains their LKI. A returned permanent is
a new selectable object (CR 400.7).

`ChooseDamageSourceFromLiveObjects(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` enumerates at resolution, then reuses
`damageSourceChoice` for property filtering, exact duplicate removal, typed or
legacy player decisions, invalid-answer fallback, snapshots, and local results.
An empty matching list or absent player clears only its result and makes no
choice. Validation checks the player, filter, named output binding and producer
order. Separate declarations and cloned resolution frames keep separate results.
No stored query result or global last-choice state is added.

Evidence: `TestLiveDamageSourcesObjects`, `TestLiveDamageSourcesFaceDown`,
`TestLiveDamageSourceChoiceResolution`, `Declarations`, and
`IndependentBindings`; existing `TestDamageSourceChoice*` and
`TestDamageSourceProperty*` retain explicit-reference coverage.
`TestRONLiveDamageSourcePrerequisite` executes the complete Prodigal Sorcerer
record and the real Circle of Protection: Blue filter with a synthetic consumer.
It checks queued resolution, recoloring, shroud, phase cycles, source departure
and return, copied triggers, source-attributed damage and independent pending
clones. `TestRONLiveDamageSourceSpell` executes the complete Lightning Bolt
record with the Circle of Protection: Red filter while the spell is still on
the stack. The original spell and synthetic continuation both resolve normally.

This service is a prerequisite, not the complete CR 609.7a legal list. Referred
objects and face-up command-zone objects remain missing engine contracts.
RON source-choice bindings, consuming shields, property rechecks (CR 609.7b,
615.9), permanent-spell continuity, finite prevention results and redirection
remain unsupported. These tests do not add complete Circle/Greater Realm card
coverage or establish new payment, replacement-order or simultaneous-operation
contracts. Source data, startup wiring and Go-card/DSL removal are unchanged.

## Captured blocking permissions and blocker requirements

`BlockAnyNumber(PermanentRef) ContinuousModification` permits the subject to
block any number of attackers. `BlockEachAttacker(PermanentRef)
ContinuousModification` supplies one requirement for each attacking creature.
Install these with `Until(EndOfTurn, ...)` to capture the resolved incarnation.
Both modify rules without granting abilities. Ability removal cannot erase
them. Absent or phased subjects cannot receive a resolving effect; control
changes and phase cycles retain an installed rule. A returned card is a new
object (CR 400.7, 611.2c, 702.26).

The existing blocking planner combines these rules with capacity, evasion,
minimum blocker counts, attacking bands, and independent requirements. It
maximizes satisfied requirements without violating restrictions (CR 509.1b–c).
A single creature can satisfy each requirement only within its legal capacity.
Multiple installations supply separate requirements. Both blocker declaration
paths use the planner. Computed rule maps are rebuilt on each layer pass and
copied for game clones. Cleanup ends the captured effects.

RON `CreatePermanentRuleEffectUntil(permanentRef, operations, UntilEndOfTurn)`
accepts a nonempty list of bare `CantBeRegenerated`,
`CanBlockAnyNumberOfCreatures`, and `MustBlockEachAttacker` operations. Source
and declared permanent-target references use existing per-instance bindings.
Regeneration prohibition reuses `GrantKeywordModification(..., CantRegenerate)`
and the existing destruction replacement check; shield creation remains legal
(CR 701.19c). Other operation families retain their existing supported shapes.
Damage prevention/redirection prohibitions and other durations remain rejected.

Evidence: `TestBlockEachAttackerRequirements`, `TestBlockEachAttackerIdentity`,
and `TestRONTemporaryRuleList*`. Complete Blaze of Glory and Hurr Jackal records
cover real casting/activation, timing and legality, rollback, captured identities,
phasing, ability removal, independent pending targets, copied spell retargeting,
clone isolation, native blocking/destruction, and cleanup. Whippoorwill and
Runesword retain separate unsupported operations.

## Static untap-step skipping and conditional limits

`SkipUntapSteps() StaticRule`, used with `Static`, replaces each player's whole
untap step (CR 500.11, 614.10). `RunStepWithPriority` applies layers and submits
`beginUntapStepAction` to the shared replacement pipeline before setting the
step or executing turn-based actions. The affected player chooses between
competing replacements; source presence, incarnation and runtime ability are
rechecked after each replacement. Source departure, phasing or ability removal
stops the rule. No LKI permits an absent static source to keep it active.
Multiple sources do not queue skips for later steps.

A skipped step does not phase objects in, consume per-permanent untap skips,
remove stun counters, untap objects, send untap events or empty mana pools.
Native turn-start control checks still clear summoning sickness for objects
controlled from an earlier turn (CR 302.6), including phased objects. Entries
and control changes during the current turn remain sick. Test setters only
record turn history; they do not run this control check. Other untap effects
remain legal. Upkeep still occurs, and its triggers use their saved controller
and source incarnation through the existing payment and sacrifice services.

`LimitUntapsDuringStepIf(filter PermanentFilter, maximum int,
condition StateCondition) StaticRule` extends the existing untap limit service.
Definition validation rejects negative limits, nil conditions and invalid
condition references. The condition reads the current source context after
phasing and layer evaluation, before any selections or untaps (CR 502.1–3).
Each true condition contributes a quota to the existing simultaneous selection.
Untapping the source cannot change that operation's quotas. A later step reads
new conditions. Overlapping limits and stun replacements keep their existing
contracts. Effect clones copy source metadata; selections remain local.

Evidence: `TestStaticUntapSkipWholeStep`, `TestStaticUntapSkipReplacementRecheck`,
`TestStaticUntapSkipTurnControl`, `TestConditionalUntapLimitDecision` and
`Declarations`; existing `TestUntapLimits*` and `TestContinuousControl*`.
RON `TestRONStaticUntapCards`, `Casting`, `Interactions`,
`ReprintsAndInventory`, `Declarations`, `TestRONStasisUpkeep` and
`TestRONStasisNativeTurn` map every rule of Stasis and Winter Orb. These tests
establish no additional draw, damage, entry or resolving-action replacement shape.


## RON numeric consumer reuse

The RON resolving numeric-consumer adapters use the existing `ResolutionValue`
services. `GainLifeForEach/2` lowers to `MultiplyValues` and `GainLifeFor`;
single-permanent counter amounts lower to `AddCountersTo`, and bounded amounts
to `AddCountersUpTo`. Both operands are read at resolution, in the action's
private binding frame. This establishes adapter coverage; it changes no engine
API, information lifetime, payment, replacement, or simultaneous-operation rule.

Living Armor uses `ManaValueOfWithLKI` with its declared permanent target.
Target legality is checked first (CR 608.2b), so an illegal target cannot use LKI
to receive counters. A live legal target supplies its mana value when placement
executes (CR 202.3, 608.2h). Icatian Moneychanger uses `CountersOnWithLKI` with
the sacrificed source's captured incarnation; each departure keeps its own
counter snapshot (CR 400.7, 608.2h). Copies use their own controller and preserve
bindings, while retargeted copies read their own target's value. P/T counters
use the existing layer 7c service and persist through cleanup (CR 122.1a/122.2).

Evidence: `TestRONNumericConsumerLivingArmor`, `LivingArmorRetarget`,
`Moneychanger`, `MoneychangerPendingIncarnations`, `ResolutionAndInstances`,
`ReplacementOrder`, `Contexts`, `BoundedPlacement`, `Nonpositive`, `Invalid` and
`Inventory` in `pkg/mage/dsl/ron/numeric_consumers_test.go`. These tests keep
compilation counts separate from complete-card gameplay evidence.


## RON source exclusion in activation payments and temporary P/T collections

RON activation costs reuse the required matching-sacrifice adapter for costs
without a sacrifice characteristic result. `Other(ThisPermanent)` in a
conjunction selects `SacrificeMatchingCost`; other matching costs retain
`SacrificeMatchingIncludingSourceCost`. The shared payment transaction validates
choices before spending the tap cost. Wrong types, opposing control, phasing,
summoning sickness and stale incarnations fail without payment (CR 602.2b).

Temporary fixed P/T collection modifications reuse the static collection plan's
`WithoutPermanent(subjects, This)` through `ModifyEachPT` and `Until`.
`This` retains the resolving ability's original source incarnation. A returned
source is eligible; an absent or phased source does not stop resolution.
Recipients are captured at resolution under CR 611.2c. Control changes, ability
removal and a phasing cycle preserve the captured bonus. A returned recipient
and a later entry do not receive it (CR 400.7). Cleanup ends the bonus
(CR 514.2). Copies keep their original source without paying again; factories
and game clones keep separate state. No engine execution contract is changed.

Card mapping: the complete DRK Orc General record is exercised by
`TestRONOrcGeneral` and `TestRONOrcGeneralRollback`. Synthetic source instances
and copied abilities use `TestRONSourceExclusionInstancesAndCopy`.
`TestRONSourceExclusionInvalid` covers malformed, unbound, incompatible and
unsupported disjunctive declarations. Bronze Horse still needs a typed static
existence condition and targeted-spell prevention; this slice does not claim
its gameplay coverage.

## RON live nonattacking filter contract

RON bare `IsntAttacking` uses the existing `Not(IsAttacking)` permanent query.
It reads current combat membership, including creatures put onto the battlefield
attacking (CR 508.4). Untapping an attacker does not remove it from combat
(CR 506.4b). Removal from combat or the end of combat changes membership
(CR 508.1k). Past attack declarations do not control this query.

Static layer membership and source/attachment conditions remain live and use
no LKI (CR 604.2, 611.3a–c). Source ability removal and phasing suspend the
bonus; return creates a new source. A phased recipient returns outside combat
and can qualify again. Arcades Sabboth's bonus applies in layer 7c, separately
from its captured end-of-turn activation bonus and counters (CR 613.4c).
Targets retain the existing incarnation and legality checks at announcement
and resolution (CR 601.2c, 608.2b). The RON adapter adds no engine state or
combat, layer, cost, target or replacement service.

`TestRONNonattackingArcadesCombat`, `Identity`, `Activation`,
`ConditionsAndDeclarations` and `TargetLegality` establish these contracts.
The existing `TestRONUpkeepPayment*` scenarios cover the complete card's
upkeep payer, timing, color payment, sacrifice, copy and identity contracts.

### Permanent property alternatives at resolution

`IfPermanentMatchesElse(subject PermanentRef, filter PermanentFilter,
matching, nonmatching ResolvingEffect) ResolvingEffect` selects exactly one body
from one property read at its position in the resolving sequence.
`UnlessPermanentMatches(subject PermanentRef, filter PermanentFilter,
body ResolvingEffect) ResolvingEffect` runs its body on a known nonmatch.
They extend `IfPermanentMatches` through the same executor and information
service. Source, attachment, saved-permanent and singular-target references keep
current, phase-retained or departure information for the exact incarnation.
An absent object with no usable information runs neither body, including the
negated form. Failure to obtain required information is not evidence of a
nonmatch (CR 608.2b/h). A zero filter matches known information.

The condition runs once before either body. A body's changes cannot cause the
other body to run. Bodies retain the resolution controller and bindings; actions
still require live legal objects. LKI cannot restore a returned target. Stack
legality, payment rollback, copying and game cloning use existing services.
Errors from the selected body stop its surrounding sequence. Both new forms
have no unconditional AI shortcut properties; their conditions must execute.
The existing one-body form keeps its properties and callers.

Definition validation requires every body and a supported, bound reference.
Both alternative bodies are inspected. A producer in one branch cannot supply
a read in another branch or after the branch. A branch-local sequence can use
its own producer. No conditional result is promoted to an outer scope.
No costs, replacement ordering, simultaneous batches or events are changed.
RON resolving property `IfElse` and `Unless` now use this contract. Static
and combat-membership alternatives remain separate contracts.

Evidence: `TestPermanentPropertyBranchSelection`, `Information`, `Definitions`,
`ErrorAndSequence`, `ReadsOnce`, `Stack` and `BindingScope`; existing
`TestIfPermanentMatches*` and `TestRONResolvingCondition*` retain the one-body
contract and real Lesser Werewolf / Eater of the Dead gameplay evidence.


### Player control acquisition at resolution

`UnlessPermanentCameUnderControlThisTurn(subject PermanentRef, player PlayerRef,
body ResolvingEffect) ResolvingEffect` reads the exact object's information and
one selected player's acquisition history at its instruction position. Entry
and actual control changes retain the latest acquisition turn for every player
on that incarnation. Later control changes do not erase another player's
acquisition in the same turn. Haste does not grant or erase history. Phasing
preserves it (CR 702.26d); a new zone incarnation starts fresh (CR 400.7).
Permanent cloning and LKI snapshots copy the history independently.

The body runs only on a known nonmatch. Missing object or player information
runs no body. The effect uses the shared permanent information branch, its
source/attachment/saved/target identity checks, and definition visitor. Nil
subjects, players or bodies, unsupported references and unbound players are
definition errors. Information does not permit actions on absent objects.

RON owns `CameUnderControlOfPlayerThisTurn(playerRef)` spelling and scope in a
resolving `Unless(PermanentPassesFilter(...), [actions])`. It does not implement
a separate history service. Both Erg Raiders reprints replace their former
current-control conjunction with this historical exception. Local ARN Oracle
text says “unless it came under your control this turn”; losing control after
that acquisition cannot restore the damage. The end-step intervening-if still
checks attack history at event and resolution (CR 603.4); the exception reads
at resolution (CR 603.5, 608.2c/h/i). The trigger's saved controller remains
its “you” after source control changes (CR 603.3a).

Evidence: `TestUnlessPermanentCameUnderControl*` covers entry, earlier turns,
haste, changed and regained control, departure, return, phasing, unknown
information, clone divergence and invalid declarations.
`TestRONPropertyAlternativesErgRaiders` and `PaymentAndRetarget` cover complete
RON casting, attack history, controller end steps, changed control, control
acquisition followed by loss, LKI damage, returned sources, phasing, independent
clones, copied trigger controllers and actual damage prevention. Blood Lust's
complete record uses the existing alternatives and temporary P/T services;
`TestRONPropertyAlternativesBloodLust`, `NestedAndInventory`, `Invalid` and
`PaymentAndRetarget` cover both toughness branches, resolution changes,
cleanup, legality, retargeting copies, binding isolation and payment rollback.

## Sacrifice-derived mana and restricted resolution production

`mage.AddRestrictedManaOfAnyOneColor(amount ResolutionValue, restriction
ManaRestriction) ResolvingEffect` reads the amount once at resolution. A positive
amount uses one controller color choice for all units. Zero or negative amounts
produce no mana and make no choice. Definition validation rejects a missing
amount or restriction. Production uses the existing mana-pool service and
production tracking. Fixed production metadata retains the restriction, so the
shared payment planner cannot use it for an incompatible spell or activation.
`CreatureSpellsOnly` uses the existing spell payment context (CR 106.6).

RON accepts `AddManaRepeatedWithModifiers(AnyManaColor, value,
CanOnlySpendOnSpells(IsCardtype(Creature)))` in resolving action lists, with the
existing supported resolution-value shapes. Other production and restriction
forms remain compile errors. The inventory reports the restriction role and its
nested spell filter independently of compiler success.

`AddManaRepeated(fixedProduction, ManaValueOfTheSacrificedPermanent)` now
accepts the existing unique matching sacrifice activation binding as well as an
additional casting cost binding. An absent, ambiguous, malformed or unrelated
binding remains an error. Each factory instance has its own typed number binding.
The shared payment transaction saves mana value before sacrifice. Copies retain
the paid value without another payment and use their own resolving controller.
Game clones keep separate pools and payment frames. Returned payment objects
cannot change the number (CR 400.7, 202.3, 707.10).

Priest of Yawgmoth remains an untargeted mana ability, including zero production:
payment and production finish immediately without the stack (CR 605.1a, 605.2,
605.3b). A tap symbol still checks summoning sickness. The matching sacrifice
permits its source if it is an artifact. Metamorphosis pays its additional cost
at casting and produces mana only when the spell resolves (CR 605.5b).
Automatic planning does not select arbitrary sacrifice payments; the existing
choice-cost exclusion remains in effect.

Evidence: `TestRestrictedResolutionMana`, `Declarations`, and `Planning` in
`pkg/mage/restricted_resolution_mana_test.go`; complete RON gameplay in
`TestRONSacrificeManaPriest`, `PriestLegality`, `Metamorphosis`,
`MetamorphosisRollback`, `ActualSpending`, `PendingValues`, `Invalid`, and
`RegistryAndInventory` in `pkg/mage/dsl/ron/sacrifice_mana_test.go`.

### Blocking history for an exact permanent

`BlockedByThisTurn(blocker PermanentRef) PermanentSet` selects live, phased-in
permanents whose exact battlefield incarnations were blocked by that exact
blocker this turn. It does not require a current combat relation or a current
creature type; callers can intersect it with `MatchingPermanents(IsCreature)`.
`ControllerWhenLastBlockedThisTurn(attacker, blocker PermanentRef) PlayerRef`
reads the attacker's controller when the last matching block pair formed.
It does not read the current controller. An absent pair supplies no player.

The shared legal block commit records all new pairs before it emits events.
Both normal declaration and `ExecuteBlockers` use it. A block of an attacking
band records every band member (CR 702.22h). Pure proposal checks, rejected
blocks and duplicate existing pairs do not record or change history. A creature
that enters blocking has not blocked for effects (CR 509.4), so that entry does
not record this history. Direct edits to the low-level `Combat` fixture do not
commit a game action and do not record history.

The turn tracker stores both object IDs and incarnations, plus the attacker
controller at that event. Control changes, combat removal and later combats
cannot erase earlier pairs. A later matching block updates only its pair's
controller. Source, attachment, saved permanent and singular target references
use the existing information identity service. Saved identities remain readable
after departure or phasing, even if LKI has been cleared; they do not make an
absent permanent available for an action. A returned attacker or blocker is a
new object (CR 400.7). Phased attackers cannot join the selected action set;
phasing in restores the original object's eligibility. Clones have separate
history maps. Final cleanup resets the history; cleanup priority retains it
with other turn history (CR 514.3a).

Evidence: `TestBlockHistoryLifecycle`, `LegalPairsAndLastController`,
`BandsAndNormalDeclaration`, `CleanupPriority`, `SeparateObjects` and
`Declarations`. RON evidence uses the exact destruction clauses from Glyph of
Doom and Glyph of Reincarnation in named contract fixtures, not full-card
support claims. No new replacement, payment or destruction service is added.

## Resolving live attack rules for this turn

`CantAttackMatchingThisTurn(PermanentFilter) ResolvingEffect` prohibits attack
declarations by current matching permanents. `MustAttackControlledThisTurn(
PlayerRef, PermanentFilter) ResolvingEffect` saves the player at resolution and
requires that player's current matching creatures to attack if able. These
rules change attack legality, not creature abilities (CR 508.1c/d, 611.2c).
Entry, return, control and characteristic changes can change membership.
Phased-out creatures cannot attack. Ability removal does not remove either rule.
Neither rule removes a creature already in combat.

The effect manager owns the independent end-of-turn entries and clones them.
Source departure, source control changes and source phasing do not end them.
Native cleanup removes them, including while a recipient is phased out
(CR 514.2). Attack restrictions use the shared final-characteristic query.
Normal declaration and `ExecuteAttackers` complete live attack requirements
before the existing legality and payment checks. An omitted creature with an
attack cost is not added solely by this resolving requirement; a proposed
attacker still uses the shared payment transaction. Existing printed
`AttrMustAttack` behavior in normal declaration is retained.

`TestAttackRulesThisTurnMembership`, `Requirements`, `Definitions`,
`PlayerIdentityAndCosts` and `AlreadyInCombat` cover late and returned objects,
current properties/control, phasing, source independence, ability removal,
clone isolation, cleanup, both declaration paths, cost choice and legality.
This adds no LKI, replacement-order, damage-batch or cost-payment implementation.
RON Festival has complete behavior evidence in
`TestRONCollectionAttackTurnFestival`. `TestRONCollectionAttackTurnSirenClause`
covers only the exact attack requirement extracted from Siren's Call.

### Turn-scoped unblocked attacker subscription

`WhenAttacksUnblockedThisTurn` uses the shared delayed permanent-event service,
`capturedDelayedBody`, `TriggerSystem`, and `EvtAttacksUnblocked`. The watched
permanent must be live and phased in at installation. Matching requires the
saved incarnation and a live creature; an explicit stale event incarnation,
LKI-only object, phased object or noncreature cannot restore a match.

The event occurs after legal blockers are committed (CR 509.1h, 509.3g).
A blocked attacker does not become unblocked when its blockers depart.
Installation cannot match an earlier event (CR 603.7a). The stated turn duration
permits repeated triggers in later combats (CR 603.7b). Control and phase cycles
keep identity; zone return creates a new object (CR 400.7, 603.7c).

The source and controller are saved when the creating action resolves
(CR 603.7d/e). Removal of the source's abilities does not remove this independent
subscription. Existing APNAP placement applies. Copies keep saved references
and use the copy controller for `You`. The shared body capture retains targets,
permanent/player/number/token bindings and X, and restores the outer resolution
frame after each body. Current or retained information reads use existing typed
reference services; this does not permit actions on absent objects.

Native cleanup ends the subscription (CR 514.2). It does not remove pending
triggers. Clones keep independent subscription lists, pending triggers and saved
frames. No replacement or cost service is added. Evidence:
`TestUnblockedDelay*` and `TestRONUnblockedDelay*`, including native RON combat,
rollback and resolution target legality, repeated installations, absent/stale
identity, LKI, APNAP, phasing, pending cleanup, copies and clones.

### Combat damage assignment suppression

`AssignsNoCombatDamage(PermanentRef) ContinuousModification` uses the existing
`Until` capture, layer installation, source-independent duration and definition
validation services. It installs a rule separately from abilities (CR 611.2).
The effect manager rebuilds an ID-to-incarnation map on each layer pass and
copies the map for game clones. Returned objects cannot inherit an old rule
(CR 400.7); control changes and phasing cycles preserve the original identity
(CR 702.26). An absent or phased subject receives no installation. Cleanup
expires the rule (CR 514.2).

Every engine combat assignment path checks the rule before choices or damage
packets: unblocked attackers, blocked attackers, trample with or without live
blockers, blocking bands, and outgoing and incoming attacking-band damage.
First/double-strike eligibility still records the first-step history
(CR 510.4). The rule stops assignment, so no damage event, lifelink, prevention
consumption or replacement-order choice can result from the suppressed source.
Unpreventable damage does not bypass it. Incoming and noncombat damage still
use the existing simultaneous damage and replacement services (CR 510.1–2).

`TestCombatAssignmentRulePaths`, `IdentityAndCleanup`, `StrikeSteps`,
`Declarations`, `NoDamageEvent` and `Installation` establish these contracts.
RON `TestRONCombatAssignment*` adds complete Delif's Cube gameplay, both
abilities, casting/payment rollback, target rechecks, source departure,
control/ability changes, phase/return identity, copied activations, pending
clones, event and factory isolation, and source/host bindings. Existing
`TestRONUnblockedDelay*` retains delayed event and clause evidence. The other
three consumers now use the optional payment contract below. Farrel's Mantle
still needs its host-event contract. The assignment rule does not change
replacement ordering.

## Optional action payments during resolution

`ResolutionActionPayment` is a sealed declaration for one gain-life, damage,
or destruction instruction. Constructors are `GainLifePayment(PlayerRef,
ResolutionValue)`, `DamagePayment(DamageSourceRef, DamageRecipientRef,
ResolutionValue)`, and `DestroyPayment(PermanentRef)`.
`PayActionOrElse(PlayerRef, ResolutionActionPayment, ResolvingEffect,
ResolvingEffect)` offers the payment to one player and runs the paid or unpaid
branch. Either branch may be nil; both absent is invalid. Nil payment inputs
and unbound references fail definition validation.

CR 118.11–12 makes the branch depend on accepted payment, not actual life,
damage, or destruction. Regeneration and damage prevention do not cancel
payment. CR 702.12b and 614.17b prevent an indestructible subject from paying
a destruction cost. Missing and phased recipients are unavailable. A phase
cycle preserves identity; leaving and returning does not. Negative calculated
amounts use zero (CR 107.1b); zero amounts cause no life or damage event.

Preparation reads the amount once and saves the recipient's incarnation and
source information. The choice is followed by an identity and availability
recheck. Execution reuses the life, damage, and destruction services, including
replacement processing, source LKI, lifelink and event generation. No state-based
action or trigger priority boundary is inserted within resolution. Single-action
payment has no earlier resources to roll back. Existing casting and activation
transactions remain responsible for announcement costs.

Prepared values and branch selection belong to one invocation. They are not
stored on the reusable declaration or in a global result slot. Copies and
clones resolve their own frames. `TestResolutionActionPayment*` covers branch
choice, target identity, phasing, prevention, regeneration, indestructibility,
zero/replaced life, nested choices, source LKI/lifelink/control, definition
rejection and clone execution. RON `TestRONActionPayment*` supplies real-card
casting, activation, target legality, combat timing and cleanup evidence.

## RON indefinite fixed characteristic changes

RON fixed `SetColor(SimpleColorList(...))` and nonempty fixed keyword
`AddAbility([...])` resolving modifications reuse `Until(Indefinite)` with
`SetColors` and `GrantKeywordModification`. No engine rules changed.
CR 611.2a/c fixes duration and subject membership at resolution; CR 400.7
separates returned objects. CR 702.26d/e preserves installed effects across
phasing but excludes absent subjects when the effect starts. CR 613.1e/f and
613.7 govern layer and timestamp order. The typed event and attachment services
retain the intended subject after the ability's source leaves.

`TestRONAislingIndefiniteColor` covers both event directions, non-targeted
recipients, pending source departure, recipient return, phasing before
resolution, control, copied triggers, cloned pending effects, cleanup and later
color timestamps. `TestRONAislingNativeCombat` exercises both native declaration
paths. `TestRONCocoonCompleteLifecycle` covers legal host casting, illegal-host
cost rollback, entry tapping and three counters, untap gating, three successful
upkeep payments, the fourth unpaid branch, Aura sacrifice, retained attachment,
+1/+1 counter and indefinite flying. It also checks upkeep player, clone,
cleanup, phase cycle and returned host identity.
`TestRONIndefiniteKeywordIdentity` checks pending copies/clones, target legality,
control, departure and later ability removal. Declaration tests reject malformed
keywords, missing references, color choices, repeated colors and empty grants.

## RON color-choice adapter evidence

`TestRONColorChoice*` establishes RON use of the existing resolution-choice and
color-layer services (CR 105.3–4, 608.2b–d, 611.2a and 613.1e). Single-color
choices exclude colorless. Plural choices include every subset of the five
colors, including the empty set. Choices occur after target legality checks.
Each fixed branch sets all colors and keeps its original permanent incarnation.
Indefinite effects survive source departure, cleanup and a subject's phase cycle;
a returned subject receives no old effect. Later color timestamps take priority.
Copied activations ask their own controller. Pending choices and resolved layers
remain separate in game clones; search clones use their existing choice defaults.
Alchor's Tomb retains its controlled-target and payment rules. Dream Coat retains
its Aura subject and once-per-turn activation rule. No engine API changes.

## Typed temporary protection and replacement ordering

`GrantProtectionColors(PermanentRef, ...core.Color) ContinuousModification`
uses the shared layer capture service. Its color list is copied at declaration.
Definition validation requires a subject and at least one of the five Magic
colors. `Until` binds the live incarnation at resolution. A departed or phased
subject cannot receive a new grant. Phasing after installation keeps the captured
identity; departure and return do not. Source departure does not end the grant.
Wrapped protection is reset on each layer pass, obeys later ability removal,
and expires with its duration. Clones retain independent effect and runtime state.

Permanent protection is now an applicable prevention candidate in both the
single damage and simultaneous damage replacement selectors. The affected
player can choose protection before or after another shield (CR 616.1).
The candidate reads the current recipient after each replacement and retains
the proposed damage source information. Redirecting to a protected recipient
can prevent damage; redirecting away can permit it. Unpreventable damage uses
the existing bypass. Legacy damage and self-fight paths also reach this service.
Protection no longer discards damage before those choices. Targeting, Aura
legality, blocking and current color reads reuse CR 702.16 services. Equipment
and Fortification attachment rejects matching protection. Existing attachments
become unattached through state-based actions, keep the attachment permanent on
the battlefield, and update both sides of the attachment relation (CR 702.16d).

Evidence: `TestProtectionModificationCaptureAndLayers`,
`TestProtectionModificationValidation`,
`TestProtectionModificationReplacementOrder`,
`TestProtectionModificationRedirectionAndBatch`,
`TestProtectionModificationAttachments`, and the real RON
`TestRONTemporaryAbility*` tests. Existing source-identity, color, prevention
allocation, fight and replacement tests remain regression evidence.

## RON static base-value adapter

`SetPowerAndToughnessBoth/1` in a singular static RON layer lowers to existing
`SetBasePowerToughness`, `Static` and `While`. This establishes compiler use of
the existing live layer 7b contract (CR 611.3a-b, 613.4b-d). Values and conditions
are read on each pass, using the source's current controller and original
incarnation. A phased or departed source supplies no effect; ability removal
in layer 6 stops the effect. Aura subjects follow current attachment through
`AttachedPermanent`. Current mana-value reads use `ManaValueOf`, without LKI.
The compiler adds no engine service or parallel layer state.

`TestRONStaticBaseValueRealBranches` tests the exact nonattacking Liege branch
in all six reprints and each Mob base-value branch in isolation. It covers
native casting, failed payment, factory bindings, live membership/control,
phasing, cloning, ability removal and set/modify/counter/switch order.
`TestRONStaticBaseValueHost` tests Animate Artifact's exact mana-value operation
in isolation, including reattachment, returned objects and source departure.
`TestRONStaticBaseValueArithmeticAndCondition` tests signed arithmetic and live
conditions. Invalid declaration and inventory tests cover strict shape/context
rejection and role reporting. These tests establish subrule contracts, not
complete-card gameplay evidence. Complete records still fail strict compilation.

## Basic land type choice before entry and at delayed resolution

`ChooseBasicLandTypeAsEnters() AsEntersBattlefieldAbility` uses the existing
entry preparation hook before battlefield registration (CR 614.12a). The entry
controller chooses exactly one of the five basic land subtypes (CR 205.3e,i).
The caller receives fresh string options through `Player.ChooseString`.
An invalid answer uses Plains, consistent with the engine's deterministic
choice fallback policy. The choice is stored as `core.BasicLandType` in
`Permanent.ChosenBasicLandType`; zero denotes no choice. It does not overwrite
chosen color, player or creature subtype. Definition validation rejects the
entry ability on an instant or sorcery.

Each incarnation makes a new choice. Control changes and phase cycles retain
it; a copied permanent spell makes its own choice when it enters (CR 400.7,
702.26, 707). `Game.Clone` copies the typed field as independent permanent state.
`BecomesChosenBasicLandTypeAttachedEffect() ContinuousEffect` reads it during
layer 4. Missing or invalid values have no effect. The shared `AttachedEffect`
and `applyBecomesBasicLand` services own attachment identity, phasing, layer
ordering, dependencies, printed ability removal and intrinsic mana (CR 305.7,
613.1d,f). A layer 6 ability-removal effect cannot undo a layer 4 type change.
These APIs do not change the legacy chosen-color or ETB-choice APIs.

RON resolving basic-land choices use five fixed `ResolutionOption` branches
and `ChooseAtResolution`, then existing `Until` and keyword modifications.
The choice is local to that resolution. Giant Slug's complete RON activation
reuses `AtNext(YourUpkeep,...)`: payment and scheduling occur first, the choice
occurs when the next controller-upkeep trigger resolves, and the captured
original source gains that landwalk until cleanup. Opponent upkeep does not
consume the schedule. Source departure, control change, phasing or ability
removal does not cancel it; an absent or returned source cannot receive the
old grant (CR 113.7a, 400.7, 603.7, 702.14). Copies and clones retain separate
resolution choices. No new scheduling, cost or layer system is introduced.

Evidence: `TestChooseBasicLandTypeAsEnters` covers all five options, invalid
answers, defensive option copying, controller versus owner, pre-entry timing,
phase/control persistence, new entry, declaration validation and clone isolation.
`TestRONBasicLandChoice*` covers both complete cards, all reprints, native casts
and costs, illegal announcement and Aura resolution, all five choices, intrinsic
mana, evasion, expiry, reattachment, source identity, phasing, control, ability
removal, copied permanent spells, copied delayed triggers and cloned choices.
The existing land-type, dependency, fixed-landwalk and delayed-step regressions
supply the underlying service contracts.

## Conditional static player damage redirection

`RedirectDamageToPermanentIf(PlayerRef, PermanentRef, PermanentFilter,
StateCondition) StaticRule` redirects damage from matching permanent sources.
`RedirectDamageFromCardTypeToPermanentIf(PlayerRef, PermanentRef, core.CardType,
StateCondition) StaticRule` also includes matching sources outside the battlefield.
The condition is required. Definition validation checks references, condition
bindings and card types. A zero permanent filter permits every permanent source.

The shared continuous-effect lifecycle installs cycle replacements. Each keeps
its source incarnation and reads the current source controller, condition, player
and destination at each proposed damage event. Departure, phasing and ability
removal stop the rule. A returned source cannot activate the original rule.
Attached destinations follow the current attachment and retain host incarnation.
A missing, phased or returned host cannot receive the old redirection. The
resolved destination must be a creature or planeswalker (CR 614.9).

The shared source query uses live characteristics or retained information for the
original incarnation, including phasing snapshots. Card-type redirection also
reads captured spell information. The replacement preserves amount, combat status
and the complete damage source. The shared pipeline then rechecks recipient
prevention and protection, permits affected-player ordering, and applies each
replacement once. Simultaneous damage, actual damage, lifelink and damage history
use the existing damage executor (CR 120.3, 609.7c, 614.5–9, 616.1/616.2).

`CombatGroup.Unblocked` distinguishes attackers before blocker declaration from
unblocked creatures after declaration (CR 509.1h). Both native blocker paths set
it before block events. Adding a blocker clears it; removing blockers alone does
not restore it. Explicit unblock effects set it. Removal from combat ends the
status. `Combat.IsUnblockedIncarnation` requires the exact live attacker.
`PermanentLKI.Unblocked` retains departure and phasing information, and
`DamageSourceIsUnblocked() PermanentFilter` uses the current or retained status.
Phase-in uses current combat state. Returned objects cannot replace old facts.
Game clones preserve independent combat and source information (CR 400.7,
506.4, 509.1h, 702.26).

RON conditional static bodies accept exactly one
`ReplaceWouldDealDamage(event,[DealToCreatureOrPlaneswalkerInstead(reference)])`.
The recipient is exactly `Player(You)`; the destination is the source or a valid
Aura host. Permanent-source events use supported property predicates and nonempty
`And`/`Or`; bare `IsUnblocked` uses the retained query. All-source events accept
exactly `IsCardtype(type)`. Other operations, dynamic targets, resolving contexts,
empty filters and incompatible references remain errors. Top-level redirection
and other replacement families remain unsupported.

Evidence: `TestStaticDamageRedirectLifecycle`, `Invalid`, `SourceInformation`,
`Ordering`, `DestinationAndRechecks`, `OriginalInstance`; real RON
`TestRONStaticRedirectCards`, `ArtifactSpell`, `Lifecycle`, `Combat`, `Invalid`,
`Inventory`, and `TestInventoryRepositoryCollection`. These establish complete
Martyrs of Korlis and Veteran Bodyguard mappings. They do not establish general
source choices, temporary redirection or collection-wide gameplay validation.


## Permanent sources referred to by stack abilities

`Game.StackAbilityPermanentSources() []DamageSource` reads the original
permanent sources of activated or triggered abilities still on the stack.
This establishes one referred-object context of CR 609.7a, not its complete
legal source list. Spells, pending triggers, targets, saved action/event objects,
waiting replacements/prevention, delayed triggers and command-zone objects
are outside this query. A caller cannot use this list alone for unrestricted
RON `ChooseADamageSource`.

Stack objects already save `SourceID` and `sourceIncarnation`. The query reads
those identities in insertion order, removes exact duplicates, and ignores
missing or unknown incarnations. Copies refer to the same source; two visits
of one card remain distinct. Removing the last stack reference removes that
source from this category. A source need not have a damage ability.

The shared permanent-information reader supplies current characteristics and
controller for a live incarnation, information saved before phasing for a
phased source, and departure LKI for a departed source. A returned card cannot
replace the old source. Phasing preserves identity; phase-in resumes current
reads. Face-down sources use the existing hidden-information snapshot helper.
The query does not run layers, make choices, pay costs, or change game state.
Each result has separate card data. Existing stack cloning and copying preserve
the references without new state. CR 400.7, 609.7a, 702.26 and 708.2 apply.

`ChooseDamageSourceFromStackAbilities(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` makes a non-targeted choice from this
category at resolution. It reuses the existing choice service, property filters,
validation, independent output bindings, empty-list/absent-player clearing,
choice fallback, snapshots and cloned resolution frames. It grants no new
production RON support. General source enumeration and property rechecks by
chosen-source shields remain separate dependencies.

Evidence: `TestStackAbilityPermanentSources`, `SourceMissingIdentity`,
`SourceIncarnations`, `SourceDeathTrigger`, `SourceChoice` and
`SourceDeclarations` in `stack_ability_source_test.go` (all have the common
`TestStackAbilityPermanent` prefix). `TestRONStackAbilitySourcePrerequisite`
executes the complete Prodigal Sorcerer and Rod of Ruin RON records through
native casting, activation and damage resolution. Synthetic consumers use the
real Circle of Protection: Blue and Circle of Protection: Artifacts filters;
those complete shield records remain rejected. Tests cover unpaid casting,
sick/unpaid activation rollback, departure/return, phasing, current recoloring,
ability copies, pending clones and original damage attribution. Existing
prevention and damage-choice tests retain replacement-order, simultaneous
packet and invalid-answer coverage; this query changes none of those services.

## Permanent targets referred to by stack objects

`Game.StackTargetPermanentSources() []DamageSource` establishes the permanent
target context of CR 609.7a. Each spell or ability still on the stack refers
to its saved target incarnations even if those targets are now illegal. The
query uses `Targets` and `targetIncarnations`, without a target-legality check.
It does not treat current objects with the same card ID as the original object
(CR 400.7). Missing or unknown identities and nonpermanent target categories
are excluded. Objects need no damage ability to qualify.

Results follow stack insertion and target-list order, deduplicated by exact
identity. A copy with unchanged targets retains the original incarnations,
including after departure; a retargeted copy adds only its selected targets.
Unused identity-map entries cannot add candidates. Removing the last referring
object removes a target from this category. The object currently resolving has
already left the stack. Pending triggers and other saved event/action bindings
are outside this query.

The existing exact-incarnation information reader supplies current information,
phase snapshots or departure LKI. Control and color changes affect live reads;
return cannot replace retained information. Phase-in resumes live reads
(CR 702.26). Face-down snapshots hide printed characteristics (CR 708.2).
The query does not mutate game state or run layers. Each result owns separate
card data; game clones retain independent stack and information state.

`ChooseDamageSourceFromStackTargets(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` uses the shared filtered choice service,
definition validation and resolution frame. Empty lists and absent players
clear only the named output. Separate bindings and cloned frames stay separate.
This is not a complete source list: other referred stack objects, waiting
replacement/prevention/delayed references and face-up command objects remain.
RON shield adapters and property rechecks remain separate blockers.

Evidence: `TestStackTargetPermanentSources`, `SourceIdentity`, `SourceChoice`,
`SourceDeclarations` and `SourceRetarget` in `stack_target_source_test.go`
(common prefix `TestStackTargetPermanent`). `TestRONStackTargetSourcePrerequisite`
executes complete Giant Growth, Lightning Bolt and Prodigal Sorcerer records
through native casting/activation, using a synthetic engine choice consumer
with the real Circle of Protection: Blue filter. Tests cover unpaid/sick rollback,
live/departed/returned/phased targets, phase cycles, recoloring, shroud, copies,
clones, retained identity and actual RON bonus/damage resolution. These tests
strengthen accepted-card coverage; they do not establish shield gameplay.
Existing damage/prevention tests retain simultaneous and replacement ordering
contracts; enumeration adds no operation to those pipelines.

## Spell targets referred to by stack objects

`Game.StackTargetSpellSources() []DamageSource` supplies the spell-target
context of CR 609.7a. `pushStack` captures exact spell stack IDs and information
on each referring object. Queries use only its current `Targets` list and saved
spell references. Current target legality does not erase a reference. Missing
identities, source-card IDs and non-spell categories are excluded. Permanent
spells and spell copies qualify without a damage ability.

Live reads use `CaptureSpellDamageSource`. `Stack.saveSpellInformation` refreshes
saved references with the departing spell's current characteristics, color override,
owner and controller before pop, ID removal, source-ID removal or stack-to-ante
movement. Departure snapshots are scoped to referring stack objects, not a global
last-action slot or an unbounded spell history. Recasts and battlefield entries
cannot substitute for the departed spell (CR 400.7). The future shield contract
for spell-to-permanent identity remains separate (CR 400.7c and 609.7a).

Results follow stack insertion then target-list order, deduplicated by exact
identity. Stack copies keep original references; changed targets capture their
own spell IDs (CR 707.10). Stale map entries cannot add sources. Removal of the
last referring stack object ends the reference; a resolving object has already
left the stack. The query is pure and returns separate card snapshots. Clones
share immutable saved maps and copy them before updates. Spell targets have no
phasing operation; permanent targets retain their separate established contract.

`ChooseDamageSourceFromStackSpellTargets(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` reuses filtered non-targeted decisions,
definition validation, binding dominance, empty/absent-player clearing, result
snapshots and resolution-frame cloning. It grants no general source choice or
production RON shield support. Other saved stack bindings, waiting replacement,
prevention and delayed references and face-up command objects remain gaps.

Evidence: `TestStackTargetSpellSources`, `TestStackTargetSpellSourceDepartureAndCopy`,
`Boundaries`, `Choice`, `Declarations` and `CopiesAndRemoval` (the latter tests share
`TestStackTargetSpellSource`). `TestRONStackTargetSpellSourcePrerequisite` executes
complete Counterspell, Spell Blast, Chaoslace and Lightning Bolt records through
native casting and resolution. It covers unpaid rollback, live/departed/recast
identity, current recoloring, copies, resolution timing and pending clones. A
synthetic engine consumer uses the real Circle of Protection: Red source filter;
complete shields remain rejected. This query adds no replacement or simultaneous
operation; existing damage and prevention tests retain those contracts.

## Permanent event and payment bindings referred to by stack objects

`Game.StackBoundPermanentSources() []DamageSource` establishes the CR 609.7a
category for exact permanent incarnations retained by event, announcement and
payment bindings on spells and abilities still on the stack. `capturedPermanents`
supplies announcement/payment captures; the private typed
`stackPermanentBindingReferences` interface supplies immutable
`eventPermanentCapture` values through `sequenceEffect` wrappers. Neither path
runs capture effects or reads a global resolution result. Future action results
cannot create candidates before their producers run.

Results follow stack insertion order and sorted permanent ID/incarnation bytes
within each object. Exact identities are deduplicated. Nil declarations and
missing, unknown or incompatible identities cannot produce sources. Pending
triggers are excluded until placement. The object currently resolving has
already left the stack. Removal or countering of the final referring object
ends eligibility. Copies retain the saved identities; existing stack copying
and cloning already preserve the maps and immutable event effects.

The shared exact-incarnation information reader supplies live characteristics
and controller, phase information or departure LKI (CR 400.7, 608.2h, 702.26).
Returned cards cannot replace old references. Face-down snapshots hide printed
characteristics. Shroud, later subtype changes and lack of a damage ability do
not remove a referred source. Queries return separate snapshots, make no
choices, use no RNG and do not change state.

`ChooseDamageSourceFromStackBindings(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` reuses the existing non-targeted choice,
filter, definition validation, producer/consumer, fallback, snapshot and cloned
resolution-frame contracts. Empty categories and absent players clear only the
named result. This adds no damage batch or replacement operation. Other binding
types, waiting effects and command objects remain outside this category; unrestricted RON source choices and shields stay rejected.

Evidence: `TestStackBoundPermanentSources`, `SourceOrderAndBoundaries`,
`SourceChoice`, `SourceDeclarations`, `SourceNativeCaptures`,
`SourceNativePayment`, `SourceResolvingLifetime` and `SourceAnnouncement`
(all use the `TestStackBoundPermanent` prefix). These cover simultaneous death
captures, APNAP placement, event/announcement/payment identities, native rollback,
source departure/return/phasing, current control/color, copying, cloning, snapshot
isolation and final-reference lifetime. `TestRONStackBoundSourcePrerequisite`
executes complete Battering Ram and Infernal Medusa records through native
casting and normal blocker declaration with priority. It uses complete Wall of
Air and Water Elemental records as blockers and a synthetic choice consumer
with Circle of Protection: Blue's real filter. It verifies retained choices,
actual damage and native delayed destruction. These strengthen accepted-card
evidence; they do not establish complete shield gameplay.

## Damage-source event bindings referred to by stack objects

`Game.StackBoundDamageSources() []DamageSource` establishes the CR 609.7a
category for permanent and spell source references retained in immutable
`eventDamageSourceCapture` effects on spells and abilities still on the stack.
The private typed `stackDamageBindingReferences` reader traverses sequence
wrappers without executing effects. It reads no global resolution slot and does
not evaluate future producers. Nil captures, missing permanent incarnations and
incompatible identity kinds are excluded. Other saved binding types and delayed
resolution frames are outside this contract.

Current information applies to exact live objects; permanent departure/phase
information comes from the shared incarnation reader (CR 400.7, 608.2h, 702.26).
Spell reads use exact stack IDs. Native stack pop, ID removal, source-ID removal,
countering and stack-to-ante movement save the departing spell's information on
each referring object. `boundSpellSources` uses copy-on-write, and copies retain
its saved references. These query snapshots do not alter the trigger body's
immutable event-time binding. A spell that already departed before trigger
placement uses its event snapshot. Recasts and permanent entry cannot replace
that spell; spell-to-permanent shield continuity is a separate prerequisite.

Results follow stack insertion and capture traversal order. Exact duplicates
are removed. No damage ability or targeting permission is required. Face-down
permanent information remains hidden. Queries make no choices, use no RNG, do
not change state, and return separate card data. Pending triggers qualify only
after placement; the currently resolving object is no longer on the stack.
Removal of the final referer ends this category's eligibility.

`ChooseDamageSourceFromStackDamageBindings(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` reuses the shared source choice service,
filter/definition validation, binding dominance and per-resolution results.
Empty/absent-player clearing, invalid-choice fallback, separate outputs and
cloned frames apply. This service adds no replacement or damage batch operation.
The existing replacement and simultaneous damage services retain ownership.
General source-choice and shield adapters remain blocked by other CR 609.7a
categories and shield installation/property rechecks.

Evidence: `TestStackBoundDamageSourcesPermanent`, `SpellDeparture`, `Boundaries`,
`ChoiceAndDeclarations`, `NativeBatch`, and `ActivationAndResolving` (all share
`TestStackBoundDamageSources`). Tests cover exact incarnation and spell identity,
live versus retained properties, phase cycles, face-down information, deterministic
ordering, duplicate captures and copies, native simultaneous damage and APNAP,
activation sacrifice/payment rollback, pending/resolving boundaries, invalid
declarations, immutable event facts, snapshot isolation and reference lifetime.
`TestRONStackBoundDamageSourcePrerequisite` executes complete Prodigal Sorcerer
and Lightning Bolt records through native casting/activation and damage, with
an engine observer and category-only consumer using real Circle of Protection
filters. It covers sickness/payment rollback, return/recast, phasing, current
control/color, copied controllers, resolution-only choice and independent clones.
These strengthen accepted-card evidence without claiming complete shield gameplay.

## Spell-event bindings referred to by stack objects

`Game.StackBoundSpellSources() []DamageSource` supplies the CR 609.7a category
for typed `SpellBinding` event captures in objects still on the stack. Cast
detection saves the exact stack ID and source information through the existing
pending-trigger capture service. Placement installs the immutable capture in
sequence wrappers. Queries traverse those wrappers without executing effects,
evaluating future producers, or reading global resolution bindings.

Live spells supply current characteristics and control. Native pop, ID removal,
source-ID removal, countering and stack-to-ante removal retain departure data
in the existing copy-on-write `boundSpellSources` map. The same departure service
now checks both damage-source and spell-event captures. A source absent before
placement uses its detection snapshot. Recasts and permanent entry cannot replace
the saved spell (CR 400.7, 608.2h). Shield continuity from spell to permanent
remains a separate CR 400.7c contract. The body still uses its saved spell ID;
query snapshots do not change its counter or current-controller behavior.

Results follow stack insertion and capture traversal order, with exact identity
deduplication. Nil declarations, invalid sources, mismatched stack IDs and
permanent identities are excluded. No damage ability or targeting permission is
required. Queries make no choices, use no RNG, change no state and return copied
card data. Pending triggers qualify only after placement; the resolving object
has left the stack. Removal of the final referrer ends eligibility. Stack copies
and game clones preserve references and separate departure information.

`ChooseDamageSourceFromStackSpellBindings(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` uses the shared resolution choice service.
Definition validation checks typed player/filter/output declarations and binding
producer order. Independent results, empty/absent-player clearing, invalid-answer
fallback, copied options and cloned frames retain their established contracts.
This adds no replacement, prevention, damage-batch or payment behavior.

Evidence: `TestStackBoundSpellSourcesDeparture`, `ChoiceAndDeclarations`,
`Boundaries`, `PendingAndResolving`, and `PermanentSpellAndCopy` cover this category,
native departure paths, pure reads, copies, clones and exact source information.
`TestRONStackBoundSpellSourcePrerequisite` executes full Nether Void and Presence
of the Master records with Crusade, unpaid casting rollback, source departure
and phasing, recast separation, copied triggers, and a synthetic category choice
using Circle of Protection: White's real filter. `IndependentEvents` uses two
RON Nether Void instances and two Lightning Bolt casts to check independent
captures, APNAP placement, deduplication, native counters and clones. Existing
`TestRONSpellCastReference*` covers invalid scopes, legality, optional payment,
current versus event controller and copied counters. Complete shield gameplay
and the full CR 609.7a source list remain unfinished.

## Saved delayed permanent frames referred to by stack objects

`Game.StackBoundPermanentSources` now includes the immutable permanent, target
and created-token maps in `capturedDelayedBody` effects on stack objects. This
extends the existing `stackPermanentBindingReferences` service and its sequence
traversal. It does not run delayed bodies, read global resolution slots, or
evaluate future producers. `AtNext` and delayed permanent-event operations
retain their existing capture, controller, timing and execution contracts.

Within each stack object, saved ID/incarnation pairs share deterministic sorting
and exact deduplication with announcement, payment and event captures. Nil
binding declarations and missing, unknown or incompatible identities produce
no candidate. Live reads use current color, types and controller. Departed and
phased reads use the exact retained information, including face-down information
hiding. Returned objects cannot replace saved objects (CR 400.7, 603.7c, 608.2h).

Waiting and pending delayed triggers are excluded until stack placement. Copies
retain their captures; game clones preserve independent query and body state.
The currently resolving object has left the stack. A second referring object
can keep the source eligible; removing the final reference ends eligibility.
The existing `ChooseDamageSourceFromStackBindings` service supplies filtered,
non-targeted choices with declaration validation, producer order, empty-result
clearing and separate choice snapshots. No replacement, payment or damage batch
contract changes. General source enumeration remains incomplete under CR 609.7a.

Evidence: `TestStackDelayedPermanentSourcesEventFrame`, `Tokens`, `Boundaries`,
`ChoiceLifetimeAndDeclarations` and `PluralTargets` cover native event/target/token
captures, live and retained identity, source departure, phasing, current color
and control, hidden information, copies/clones, ordering, duplicate references,
future/global exclusion, invalid declarations and resolving lifetime.
`TestRONStackDelayedPermanentSources` executes complete Stone Giant, Goblin Kites
and War Barge records with Prodigal Sorcerer targets, unpaid activation checks,
native delay placement, source choices using Circle of Protection: Blue's real
filter, actual damage and delayed destruction/sacrifice.
`TestRONStackDelayedPermanentSourceInstances` checks separate Goblin Kites factory
bindings, APNAP placement, countering and independent cloned sacrifice bodies.
These strengthen accepted-card evidence; they do not verify complete shields.

## Permanent references held by waiting delayed abilities

`Game.WaitingDelayedPermanentSources() []DamageSource` implements the saved
permanent-frame and exact-event-subject contexts of CR 609.7a for delayed
abilities still registered as waiting. It reads immutable `capturedDelayedBody`
permanent, target and created-token maps through sequence wrappers and modal
bodies, and `MatchEventID` with its saved `matchIncarnation`. It reuses the stack
reference reader's sorting, exact deduplication and incarnation information
service. Nil declarations, invalid identities and unknown incarnations cannot
supply candidates. A source ID alone, a global resolution frame, an unproduced
future action or an untyped legacy matcher cannot create a candidate.

Queries return separate card snapshots, make no choices, consume no RNG and do
not change state. Current live information, phased information and departure
LKI retain their existing rules. Leave-and-return cannot replace a saved object
(CR 400.7, 603.7c, 702.26). Target legality and ability to deal damage do not
restrict this non-targeted category.

One-shot detection removes the waiting referer before pending placement.
Duration-based delays remain registered after a match and stop supplying this
category when their lifecycle expires (CR 603.7b). Pending and stack references
are separate categories. Stack copies do not create waiting abilities. Game
clones keep independent trigger lists and immutable saved frames.

`ChooseDamageSourceFromWaitingDelayedBindings(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect` uses shared definition validation,
producer-before-consumer checks, independent resolution results, empty and
absent-player clearing, invalid-answer fallback and cloned frames. This adds
no payment, damage batch, prevention or replacement operation.

Evidence: `TestWaitingDelayedPermanentSourcesLifecycle`, `FramesAndMatchers`,
`ChoiceAndBoundaries` and `NativeFrames` cover native target/event/token/modal
captures, plural groups, exact matchers, current properties, departure/return,
phasing, hidden data, shroud, deduplication/order, copied snapshots, waiting versus
pending/stack lifetimes, duration expiry, clone isolation and invalid declarations.
`TestRONWaitingDelayedPermanentSources` executes complete Stone Giant, Goblin
Kites and War Barge records, with Prodigal Sorcerer targets and Circle of
Protection: Blue's real filter in a synthetic category consumer. It covers native
payment failure, resolution-only choice, delayed actions and clone execution.
`TestRONWaitingDelayedPermanentSourceInstances` covers separate factory bindings,
registration order, APNAP placement and independent delayed sacrifices.

Other typed references, source reads in future bodies, untyped legacy IDs,
waiting replacement/prevention references, command objects and shield continuity
and property rechecks remain separate gaps. This does not establish the complete
CR 609.7a source list or accept general RON source choices.


## Source references in waiting delayed action bodies

`Game.WaitingDelayedPermanentSources` also reads explicit `This` subjects in
`Destroy`, `DestroyWithoutRegeneration`, `Sacrifice` and `ReturnToHand` bodies.
This context includes saved `capturedDelayedBody` wrappers, nested `Sequence`
bodies and each saved modal body. The typed reference interface does not execute
an action. A source ID alone does not qualify. An action that will create another
delayed ability does not supply its future body's references in this context.
Other source-reading action kinds and conditional wrappers remain separate forms.

The ability must have a captured source and a known permanent incarnation.
The reader uses the saved source ID/incarnation, the existing incarnation
information service and the existing deterministic deduplication service.
Current color and control, departure LKI, phasing and hidden characteristics use
the same contracts as saved target references. Returned cards cannot replace the
original source (CR 400.7, 603.7c, 603.7e, 609.7a and 702.26).

`ChooseDamageSourceFromWaitingDelayedBindings` can consume this category through
its existing filtered choice, validation and output services. One-shot detection
ends waiting eligibility before pending placement. Copies on the stack do not
create waiting abilities; clones keep independent waiting lifetimes. The actions
keep their existing legality: a player cannot sacrifice a source controlled by
another player. This slice adds no cost or replacement operation.

Evidence: `TestWaitingDelayedSourceBodies`, `TestWaitingDelayedSourceBodyBoundaries`
and `TestWaitingDelayedSourceBodyChoicesAndInstances` cover exact identity,
current and retained information, query snapshot isolation, modal and sequence
bodies, declaration rejection, independent filtered outputs, empty/absent-player
clearing, APNAP, stack copies and clone execution. `TestRONWaitingSourceBodies`
executes complete Dragon Whelp, Nalathni Dragon, Rocket Launcher and Rakalite
records, payment failure, delayed creation and native delayed actions.
These are added contracts for accepted records, not complete prevention-card
coverage. General RON `ChooseADamageSource` remains rejected.

## Static maximum hand size

`SetMaximumHandSizeStatic(PlayerRef, int) ContinuousEffect` sets a nonnegative
maximum. `NoMaximumHandSizeStatic(PlayerRef) ContinuousEffect` removes the
maximum. Use `WithStaticAbility`. Definition validation rejects nil references,
unbound resolution bindings and negative fixed sizes. Each factory has its own
effect source. Clones copy that source state.

`ChosenPlayerOf(PermanentRef) PlayerRef` reads the live incarnation's entry-chosen
player. It supplies no player for an absent, phased, returned or unchosen object,
or an unknown player ID. It does not use LKI. `You` reads the source's current
controller for these static rules; the entry choice stays fixed through control
changes and phasing. A new entry gets a new choice.

`Game.MaximumHandSize` evaluates the active typed rules in existing continuous
source timestamp order after object characteristics are established (CR 613.11).
The last matching set or no-maximum rule wins. This query uses the existing
printed-ability suppression and phasing lifecycle without a layer pass or a
mutation. A phase cycle can restore an earlier rule. Removing or returning a
source cannot leave its old rule active. Legacy direct setters retain their
existing computed-map behavior and supply the baseline for this typed query.

Cleanup uses the same query for the active player (CR 402.2, 514.1). A zero
maximum requires discarding the whole hand; no maximum requires no discard.
Inactive players do not discard. The shared cleanup/discard service still owns
card choice, movement, replacements and trigger placement. This slice adds no
new discard replacement or payment operation.

Evidence: `TestMaximumHandSizeStatic*`, `TestChosenPlayerOfLiveIncarnation` and
`TestRONMaximumHandSize*`. The RON tests cover the complete Cursed Rack record,
casting payment rollback, pre-entry timing, cleanup, choice and source lifetime,
independent factories, cloning, declaration rejection and atomic loader rejection.
Library of Leng's hand-size rule is tested from its real record in isolation;
its full record still rejects its unsupported discard replacement.

## Delayed spell-event frames

`AtNext`, `AtNextModes`, and delayed permanent-event bodies capture the current
`SpellBinding` frame when the delayed ability is created. Each body restores its
own frame for execution, then restores the outer frame. `FromSpell` reads the
original stack identity. An absent spell cannot be countered; a later cast of
the same card cannot replace it (CR 400.7 and 603.7c). Bindings remain subject to
normal definition validation and producer scope.

`Game.WaitingDelayedSpellSources() []DamageSource` supplies only spell references
in saved waiting frames, including nested sequences and modal bodies. It excludes
nil bindings, mismatched identities, ability objects, future delayed producers,
and global resolution frames. Registration order and sorted stack identities
make deduplication deterministic. One-shot detection ends waiting eligibility
before pending placement. A placed body supplies the existing
`StackBoundSpellSources` category. Copies do not create waiting abilities.

The stack saves exact spell departure information alongside its existing saved
mana values and controllers. Queries use current information for a live spell,
then departure information for that stack identity, then the saved frame.
Departure information does not change event bindings. Returned query values own
their characteristic snapshots. Stack clones own separate departure maps;
delayed frames are immutable and shared safely until execution.

`ChooseDamageSourceFromWaitingDelayedSpellBindings(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` uses the shared
non-targeted decision, filtering, validation, and output-clearing service.
It supplies this category only. The full CR 609.7a source list still needs waiting
damage-source frames, replacement/prevention references, other referred forms,
and command-zone objects. No payment or replacement-order contract changes.

RON now lowers a bound `Trigger_ThatSpell` inside supported `CreateFutureTrigger`
and `CreateTriggerUntil` bodies through the existing typed spell reference.
An unbound reference or a player/permanent reference in this role remains an
error. This establishes a shared prerequisite with no collection unlocks.

Evidence: `TestWaitingDelayedSpellFrames`, `ChoicesAndBoundaries`, and
`IndependentEvents` cover live/departed/recast spells, color/control changes,
modal and sequence frames, query isolation, choice clearing, invalid declarations,
frame restoration, APNAP, stack copies and clones. `TestRONWaitingDelayedSpellFrame`
executes the complete Lightning Bolt record with a synthetic RON delayed-event
consumer, including unpaid casting rollback, timing, source departure, recasting,
copying and clone execution. `TestRONWaitingDelayedSpellInvalid` covers invalid
reference shapes and types. Existing stack spell-binding tests remain required.

## Damage-source references in waiting delayed frames

`AtNext`, `AtNextModes`, `WhenDiesThisTurn`, `WhenLeavesThisTurn`, and
`WhenAttacksUnblockedThisTurn` capture typed `DamageSourceBinding` maps at delayed
creation. Captures copy each source snapshot. Execution restores a separate
copy of the saved map and restores the outer map afterward, including on an
error. Body reads retain event-time facts; current source queries cannot change
those facts. Modal bodies share immutable captured declarations and receive
separate execution maps. Existing cloning and stack copying preserve these
immutable frames.

`Game.WaitingDelayedDamageSources() []DamageSource` reads source references in
waiting frames through the typed `stackDamageBindingReferences` service. It
traverses saved sequences and modes without executing them. Results follow
registration/traversal order, then sorted object ID/incarnation/stack identity
within a frame, with exact deduplication. Nil bindings, invalid or incompatible
identities, unknown permanent incarnations, global resolution slots, source IDs
alone, and future producers cannot supply candidates. One-shot detection ends
waiting eligibility before pending placement. Persistent delays retain it until
cleanup. Placed bodies supply `StackBoundDamageSources`; copies do not create
new waiting abilities.

Both waiting and stack damage-source queries now reuse the exact information
reader. Permanent reads use current, phase, or departure information for the
saved incarnation. Spell reads use current stack information, exact departure
history, then the saved snapshot. This preserves departure updates that occurred
before a waiting body reached the stack. Return/recast cannot supply another
object's information. Queries are pure and return separate snapshots; face-down
permanents retain hidden characteristics. These services follow CR 400.7,
603.7a–e, 609.7a, and the existing phasing information contract.

`ChooseDamageSourceFromWaitingDelayedDamageBindings(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` uses shared filtered
non-targeted choices, definition validation, binding dominance and output
isolation. Empty categories or absent players clear only this output. This
category alone cannot implement general RON source choice. Waiting replacement
and prevention references, command objects, other referred forms, and complete
shield installation/property rechecks remain separate gaps. This slice changes
no payment, replacement ordering, or simultaneous damage rules.

RON `WhenAPlayerIsDealtDamage(SinglePlayer(You))` and
`WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent))` now lower with a
fresh internal damage-source binding per factory. This event capture can pass
into supported `CreateFutureTrigger` and `CreateTriggerUntil` frames. The source
capture adds no public RON reference spelling or targeting permission. Existing
amount and recipient bindings keep their established scope and execution.
Invalid received-event shapes and unbound amount reads remain errors.

Evidence: `TestWaitingDelayedDamageFrames` covers ordinary/modal replay,
current control/color, departure/return/recast, phase cycles, hidden information,
immutable queries, stack transition, copies, clones and outer-frame restoration.
`TestWaitingDelayedDamageBoundaries` covers persistent departure delays, cleanup,
filtered choices, output clearing, invalid definitions and future/global source
exclusion. `TestWaitingDelayedDamageIndependentEvents` covers separate factories,
four simultaneous event frames, deduplication, APNAP and clone gameplay.
`TestWaitingDelayedDamageSavedFacts` proves capture isolation and event-time
lifelink controller replay despite a different current source controller.
`TestWaitingDelayedDamageErrorRestoration` proves independent execution snapshots
and outer-frame restoration when the delayed body returns an error.
`TestWaitingDelayedDamageInvalidFrames` excludes bad identities without masking
valid frames. `TestRONWaitingDelayedDamageFrames` executes complete Lightning
Bolt and Prodigal Sorcerer records with synthetic RON received-event consumers,
including payment/sickness rollback, prevention, waiting timing, source departure,
return/recast, phasing, copied triggers, filtered choices and cloned delayed life
gain. `TestRONWaitingDelayedDamageInvalid` covers declaration rejection. Existing
complete Fungusaur and Living Artifact received-damage regressions also pass.
This adds accepted-card interaction coverage, with no full shield unlock.

### Waiting prevention references: captured permanent contexts

`Game.WaitingPreventionPermanentSources` supplies the CR 609.7a referred-object
category for three exact contexts: finite recipient shields, resolved unlimited
recipient rules, and next-permanent-source prevention. Existing RON AST lowering
through `PreventNextDamage` and `PreventDamageThisTurn` remains the adapter;
no duplicate compiler or game-time declaration service is needed.

Unlimited recipient installation now retains an explicit `capturedPermanent`
identity beside its matching filter. Finite recipients and next-source rules
reuse their existing ID/incarnation pairs. Only active effects expose them.
Query order follows persistent then cycle registration; exact duplicates are
removed. Player recipients, incomplete IDs and unknown incarnations are excluded.
Source properties use the shared current/exact departure/phasing information
reader. A returned object cannot substitute its identity or information, even
when the old short-lived LKI cache is cleared. Snapshots have separate card data.
Choices use existing filter validation, player decisions, fallback indices,
per-output binding writes and definition checks. Clones retain independent
replacement lifetimes. Consumption and cleanup end eligibility; unlimited
prevention remains eligible after matching damage. CR 400.7, 609.7a, 615.1–4,
615.6–7 and the existing phasing information contract apply.

Evidence: `TestWaitingPreventionPermanentSources` covers all three contexts,
departure/return, cleared LKI, phase cycles, control, cleanup, consumption and
clones. `TestWaitingPreventionChoicesAndBoundaries` covers invalid identities,
exact deduplication, independent outputs, empty-choice clearing and declaration
rejection. `TestWaitingPreventionInformationAndChoice` covers current and retained
colors, filtered choices and separate query snapshots.
`TestRONWaitingPreventionReferences` executes complete Samite Healer, Healing
Salve, Alabaster Potion and Indestructible Aura records through their existing
AST lowering, payment/sickness rollback, resolution timing, target legality,
departure/return, phasing, consumption, cleanup and clone gameplay. Selected
absent recipients can supply damage sources through the shared damage executor.
Existing invalid RON prevention declarations remain rejected.

This establishes no new replacement ordering or payment behavior. General
`ChooseADamageSource`, other waiting prevention/replacement references, saved
spell references and command-zone categories remain unfinished. No full-card
compilation gain or complete CR 609.7a enumeration is claimed.


### Exact waiting combat-prevention subjects

Resolved `combatSourcePreventionReplacement` and
`creatureCombatDamagePreventionReplacement` supply their saved ID/incarnation
pairs to the existing waiting-prevention reference query. This establishes the
CR 120.7 / 609.7a referred-object category for exact resolved combat prevention,
including collection members saved at resolution. The replacement's registration
source is not an additional subject. Global prevention and unspecified filter
members do not supply exact references. Static subjects and other replacement
contexts need separate contracts.

Current characteristics and control, exact departure information, and phased
information use `damageReferenceInformation`. CR 400.7 identity prevents a
returned card from replacing the saved object. Clearing the short LKI cache does
not discard the exact departure information. Queries return separate snapshots,
remove exact duplicates, and reject incomplete or unknown identities. Matching
damage does not consume this unlimited prevention. Existing duration removal
ends reference eligibility. Clones retain independent replacement lifetimes.
No damage, replacement-ordering, target, payment or simultaneous-operation rule
changes in this slice.

`TestWaitingCombatPreventionReferences` covers singular to-and-by, singular source,
turn collection and combat collection contexts; capture, departure/return,
cleared LKI, phasing/cycles, current control, repeated prevention, duration removal
and clone isolation. `TestWaitingCombatPreventionChoiceBoundaries` covers filtered
independent choices, empty output clearing, duplicate/incomplete/unknown subjects,
global-prevention exclusion and query snapshot isolation.
`TestRONWaitingCombatPreventionReferences` executes complete Maze of Ith, Ebony
Horse, Elvish Scout, Horn of Deafening, Lady Evangela, Subdue, Telekinesis, Feint
and Sewers of Estark records. It covers payment failure, resolution timing,
phased target rejection, saved identity after departure/return, phasing, stack
copies, combat/turn duration boundaries, clone queries, and selected retained
subjects as noncombat damage sources. Existing invalid declaration, combat
membership, legality, payment rollback and simultaneous combat tests still apply.
This is added interaction evidence for accepted cards, not complete general source
choice or a full-card compilation gain.

### Live subjects of waiting typed static prevention

`staticDamagePreventionReplacement` supplies its current exact subject to
`Game.WaitingPreventionPermanentSources`. Its private `currentSubject(*Game)`
resolver is shared with damage matching. It checks the source incarnation,
current controller and current `PermanentRef` before returning a subject.
It does not test a hypothetical damage event or execute the prevention body.
CR 120.7 and 609.7a permit a referred object even if it cannot deal damage.
Source filters and current blocked attackers do not add unspecified members.

This contract covers `PreventCombatDamageFrom`, `PreventCombatDamageTo`,
`PreventDamageFromCardType` and `PreventDamageFromBlockedCreatures`, with `This`
or `AttachedPermanent`. Static subjects are live references: reattachment changes
membership; source or subject phasing/departure removes it (CR 702.26b, 611.3).
A returned source cannot satisfy an old cycle entry (CR 400.7). Source ability
removal uses the existing continuous layer lifecycle. Host ability removal does
not remove a separate Aura rule. A fresh rule on a returned creature supplies
the new incarnation. These semantics differ from saved resolved shields.

`TestWaitingStaticPreventionReferences` covers all four native rule kinds.
`TestWaitingStaticPreventionStaleCycle` directly retains an old cycle entry to
check reattachment, source/host phase, source/host return and changed control.
`TestWaitingStaticPreventionChoiceBoundaries` covers duplicate subjects, exclusion
of registration sources/global prevention/unrelated permanents, independent
filtered bindings, empty output clearing, separate snapshots and cleared LKI.
`TestRONWaitingStaticPreventionReferences` executes complete Demonic Torment,
Gaseous Form, Enchanted Being, Argothian Treefolk and Wall of Vapor records. It
checks live/reattached identity, phase/cycle, source/host departure and return,
source/host ability removal, control, repeated query after damage, cleanup,
independent clones and choice followed by damage from that subject.
Existing static prevention tests retain declaration rejection, casting and
payment rollback, actual prevention, simultaneous damage and replacement order.

Wall of Shadows still fails its separate `CanTargetOnly` declaration. Legacy
filter-factory static rules and other waiting replacement contexts remain gaps.
No payment, damage batch, replacement order or general source-choice rule changes.

### Typed permanent-filter static prevention

`PreventDamageFromMatchingPermanents(PermanentRef, PermanentFilter) StaticRule`
uses the typed static prevention lifecycle and current exact subject resolver.
It delegates permanent source filtering to `damagePreventionRuleReplacement`
and `Game.damageFilterSource`. A zero filter becomes `AnyPermanent`, so it cannot
bypass permanent-source identity checks and prevent spell-source damage.
Combat and noncombat packets use the same filter. Live source characteristics,
phase information and exact departure LKI retain the existing source contract;
a returned card cannot replace the original incarnation (CR 400.7, 609.7c).

Subjects are live `This` or `AttachedPermanent` references. Reattachment changes
the subject; phasing, departure or source ability removal ends its current rule.
A returned source cannot restore an old cycle entry (CR 611.3, 702.26b).
The exact current subject also supplies the waiting-prevention reference category
(CR 120.7, 609.7a). Registration sources and unspecified filter members are not
additional references. No hypothetical damage event runs during the query.

`TestStaticPermanentFilterPrevention*` covers valid/invalid declarations, zero
filters, matching and nonmatching sources, source characteristic changes,
departure/return/phasing, spell exclusion, unknown identities, unpreventable
packets, repeated simultaneous combat/noncombat damage, live Aura lifecycle,
control, ability removal, clone isolation and replacement rechecks/redirection.
`TestWaitingStaticPermanentFilterPreventionStaleCycle` tests saved cycle entries.
`TestRONWaitingStaticPreventionReferences` now covers complete Argothian Pixies,
Desert Nomads and Uncle Istvan records through exact choices and damage execution.
`TestRONStaticPrevention*` retains declaration rejection, pending activation/copy,
combat, casting, strict loading, instance isolation and recipient replacement order.

This does not establish general source choice or the seven chosen-source shields.
Other waiting replacement contexts and legacy filter-factory callers remain gaps.
No costs, damage batch ordering or existing replacement priorities change.


### Exact live destinations of waiting static damage redirection

`Game.WaitingRedirectionPermanentSources() []DamageSource` establishes the typed
static damage redirection destination part of the referred-object category in
CR 120.7 and 609.7a. It queries the existing `staticDamageRedirectReplacement`
context resolver used by actual damage execution. Current source incarnation,
controller, static condition, attachment and creature/planeswalker type are
checked once per rule. Phasing and source departure exclude a live rule; a
returned object cannot satisfy an old cycle entry. Source ability removal uses
the shared continuous lifecycle. No hypothetical damage, source-filter match,
player choice, RNG call or replacement execution occurs during the query.

The query and `WaitingPreventionPermanentSources` share one exact-reference
collector. It visits active persistent then cycle replacements, excludes missing
identities/information, deduplicates by object identity and returns isolated
snapshots. The two public categories remain separate. Registration sources,
players, arbitrary filter members, legacy redirection closures and unrelated
permanents are not inferred as destinations. Current live information supplies
properties; departed destination LKI cannot revive a live static rule.

`ChooseDamageSourceFromWaitingRedirectionReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` reuses the existing
filtered non-targeted decision service, declaration validation, empty-result
clearing, independent per-output bindings and clone state. It enumerates only
this category. It cannot implement unrestricted `ChooseADamageSource` alone.

Evidence: `TestWaitingRedirectionReferences` covers both native rule forms,
condition transitions, control, phase cycles, departure/return, cleared LKI,
ability removal, repeated damage, cleanup and clones.
`TestWaitingRedirectionHostAndStaleCycle` covers reattachment, current conditions,
source/host phasing, departure/return, destination type changes and saved cycle
identity. `TestWaitingRedirectionChoicesAndBoundaries` covers exact deduplication,
registration/unrelated/prevention exclusion, isolated query snapshots, filter
reads, valid/invalid declarations, independent outputs, empty-result clearing
and clones. `TestRONWaitingRedirectionReferences` executes both complete card
records through their typed AST lowering and proves that the exact selected
destination can supply damage. Existing `TestRONStaticRedirect*` and
`TestStaticDamageRedirect*` retain legality/payment, combat and simultaneous
packets, pending ability/copy LKI, replacement order and destination rechecks.

This establishes no new damage ordering or payment behavior. Other waiting
replacement/source/spell contexts, selected-source shields, face-up command
objects and complete source enumeration remain gaps. No RON record changes or
full-card compilation gain is claimed.

### Exact-object source shields with typed properties

`PreventNextDamageFromExactSource(DamageSourceRef, PlayerRef, DamageSourceFilter)`
establishes an exact-object prevention installation at resolution, separate from
source enumeration and spell-to-permanent continuity. It resolves the source and
recipient once and copies source information into each replacement instance.
Sources require exactly one permanent incarnation or spell stack identity. A
missing binding, incomplete identity, absent player or nonmatching installation
filter creates no replacement. Definition validation checks typed source/player
references, binding dominance and the filter declaration.

`sourcePreventionReplacement` preserves legacy UUID and permanent-filter callers.
The new typed context uses `DamageSource.SameObject` and the shared exact
information reader. Live permanent/spell properties take precedence over a saved
packet. Departure and phasing use the existing exact LKI and spell-retention
services; if permanent information is absent, the matching damage packet supplies
its source information. Recasts, returned objects and unidentified packets cannot
substitute another source. A property mismatch or zero damage does not consume
the shield. Matching positive player damage consumes it. The shared simultaneous
replacement planner treats exact spell sources like exact permanent sources and
prevents all matching packets in one event. Affected-player replacement choice,
rechecks and cleanup retain their existing contracts. Cloning copies the saved
source information and keeps replacement consumption separate. Exact permanent
subjects also join the existing waiting-prevention permanent-reference category.

The scoped RON adapter validates the exact two-argument future event, typed
`Player` recipient and single bare `PreventThatDamage` operation. Its source plan
and factory-local binding carry the choice filter without a global result slot.
Production `ChooseADamageSource` remains rejected. The scoped adapter now uses
the selected-source continuation service described below. The exact-object API
retains its original boundary. Complete enumeration remains a prerequisite for
general chosen-source shields.
CR 400.7, 609.7b, 615.8–9 and 616.1 apply to the established context.

Evidence: `TestExactSourcePreventionIdentity`, `Spells`, `Declarations`,
`RechecksAndOrder` and `IndependentBindings` cover identity, cleared LKI,
phasing, cleanup, stack recasts, live properties, simultaneous events, invalid
and unbound declarations, competing shields and clone isolation.
`TestRONExactSourcePreventionLowering` maps the shield expressions in all seven
protection cards; `TestRONExactSourcePreventionInvalid` covers declaration
rejection. `TestRONExactSourcePreventionGameplay` uses complete RON Lightning
Bolt, Prodigal Sorcerer and Orcish Artillery records for real casting/activation,
payment/sickness legality, ability-source departure/return/phasing, copies,
independent artillery recipients and pending-game clones. These tests do not
establish complete gameplay for the rejected protection cards.


### Selected-source prevention across permanent spell resolution

`PreventNextDamageFromSelectedSource(DamageSourceRef, PlayerRef,
DamageSourceFilter) ResolvingEffect` adds the CR 400.7c and 609.7a continuation
to the exact-source installation contract. The caller supplies a typed selected
source; this API does not enumerate source choices. Existing
`PreventNextDamageFromExactSource` callers keep their exact-object behavior.
Both APIs use the same definition validation, property checks and replacement
planner. Nil references, unbound reads and invalid filters remain errors.

When a permanent spell enters through resolution, successful entry preparation
and battlefield registration precede the shield link. Entry effects and events
follow it. Each active selected-source shield compares the resolving spell's
exact identity and captures the new permanent's exact identity. No global
last-action result or persistent card-ID correspondence is used. A chosen spell
copy can become a token with a different card ID. Separate copies, recasts,
failed entry and direct battlefield entry cannot inherit another spell's shield.

The shield matches its original spell or its continued permanent. Both consume
the same next-event shield; consumption before entry prevents continuation.
Current permanent properties override saved packet properties. Nonmatching
properties, zero damage and other recipients preserve the shield. Departure and
phasing retain exact-object information; missing information falls back to the
exact matching damage packet. Returned permanents cannot match. The shared
simultaneous replacement service, affected-player ordering and cleanup keep
their existing behavior. Clones copy both saved source snapshots and consumption
state. Continued subjects join `WaitingPreventionPermanentSources`. Exact
waiting spell references and complete source enumeration remain separate gaps.

`TestSelectedSourcePreventionContinuity`, `CopyAndClone`, `ChosenCopy`,
`EntryTimingAndCounter`, `RejectedEntry` and `Declarations` cover the native
contract. `TestRONSelectedSourcePreventionContinuity` lowers the exact shield
expressions of all seven protection records and casts complete RON Frozen Shade,
Prodigal Sorcerer, Llanowar Elves, Orcish Artillery, Samite Healer and Grapeshot
Catapult. Sorcerer and Artillery use real activation, sickness checks, resolution
and separate self-damage; the other four supply source-identity interaction
evidence through native damage. These are subrule tests for the rejected
protection cards, not full-card gameplay evidence.

### Exact spell references held by waiting prevention shields

`Game.WaitingPreventionSpellSources() []DamageSource` supplies the exact spell
references held by active typed source-prevention shields in both replacement
lists. CR 609.7a permits these objects after they leave the stack. Only a valid
spell identity with a stack ID and no permanent incarnation qualifies; legacy
card IDs and malformed mixed identities do not. Duplicate references are merged
in replacement-list order. Queries copy card data and do not change game state.

Current stack information takes precedence over exact retained departure data,
with the shield's saved snapshot as fallback. Recasting the same card and separate
spell copies cannot replace the original identity. A selected permanent spell's
continuation retains the original spell reference and separately supplies the
entered incarnation through `WaitingPreventionPermanentSources`. Consumption or
cleanup removes eligibility; source departure, control changes and phasing of the
continued permanent do not end an unconsumed shield. Clones keep independent
snapshots and lifetimes. Existing CR 400.7c, 609.7b, 615 and 616 mechanics remain
unchanged.

`ChooseDamageSourceFromWaitingPreventionSpellReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` uses the shared typed
choice service for this category only. It filters current or retained information,
clears its own result when no candidate exists, and validates player, filter and
binding declarations. It does not grant a general source enumeration contract.

`TestWaitingPreventionSpellReferences` covers live properties/control, departure,
recast, copy, missing departure history, exact-object callers, continuation,
phasing, consumption, cleanup, copied query data and clones.
`TestWaitingPreventionSpellChoice` covers both lists, deduplication, invalid
identities, legacy exclusion, independent bindings, empty choice and invalid
engine definitions. `TestRONWaitingPreventionSpell` lowers the seven protection
records' choice plans and shields with this explicit category consumer, executes
complete RON source casts, then checks waiting identity and independent shield
consumption in original and cloned games. `TestRONWaitingPreventionSpellInvalid`
checks syntax and missing factory binding rejection. This is subrule evidence;
the seven protection cards remain rejected pending complete source enumeration.


### Exact subjects held by waiting regeneration shields

`Game.WaitingRegenerationPermanentSources() []DamageSource` supplies exact
permanent subjects of active regeneration shields in both replacement lists.
CR 609.7a permits these referred objects even after departure; CR 701.19a keeps
the shield waiting until destruction or cleanup. Phasing and an inability to
regenerate do not remove the reference. The query reads current characteristics
or exact retained departure information. Returned incarnations cannot replace
old subjects. Invalid identities are excluded, duplicates are merged in list
order, and query results contain copied card data. Clones retain separate shield
lifetimes. This query uses the shared waiting permanent-reference collector.

`ChooseDamageSourceFromWaitingRegenerationReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` supplies this category
through the shared choice service. Pure filters read current or retained data.
Each choice writes only its own binding; no matching candidate clears that
binding. Definition validation checks the player, filter, binding and producer
order. It introduces no new targeting, payment, destruction or replacement
ordering behavior.

RON `chosenDamageSourcePlan.waitingRegenerationChoice(instanceBindings)` lowers
an already validated `ChooseADamageSource(filter)` plan with `You` and its
factory-local binding. The existing scoped `CreateFuturePreventDamage` consumes
that binding. Production general source choices remain rejected until all legal
categories are implemented. No partial registry is installed.

Evidence: `TestWaitingRegenerationReferences`, `TestWaitingRegenerationChoices`
and `TestWaitingRegenerationSeparateSubjects` cover lifecycle, retained data,
phasing, control, duplicate and invalid identities, independent choices, shield
counts, returned objects and clones. `TestRONWaitingRegeneration` executes full
Drudge Skeletons, Death Ward and Living Wall records through payment and
resolution, including illegal targets and retained-source damage. The protection
subrule test `TestRONWaitingRegenerationProtection` lowers all seven protection
records, chooses a departed Living Wall retained by its regeneration shield,
and checks independent prevention consumption in original and cloned games.
`TestRONWaitingRegenerationInvalid` checks invalid choice syntax. These are
additional interaction and subrule tests, not new full-card compilation coverage.

### Optional draw-step skip with a fixed life gain in RON

RON permanent rules support this complete replacement shape:

```ron
ReplaceWouldBeginDrawStep(PlayerWouldBeginTheirDrawStep(You), [
    MayCost(SkipThisDrawStep),
    If(CostWasPaid, [GainLife(Integer(n))]),
])
```

`n` must be a nonnegative integer literal. The compiler checks the complete
operation sequence before it returns a factory. Each factory builds a fresh
`Static(SkipDrawStepForLife(n))`. Other event players, costs, branches, values,
extra operations and instant/sorcery contexts fail compilation. `CostWasPaid`
belongs only to this replacement; it creates no resolving payment binding.

The shared engine replaces the whole proposed draw step before detection,
normal drawing or priority. Declining leaves the step intact. CR 614.10b puts
the gain first in the next step that actually occurs. The saved player and
amount survive source departure and stay separate in cloned games. The rule
follows its current controller; departure, phasing or ability removal stops it.
The existing replacement pipeline owns choices, competing replacements and
rechecks. This adapter does not change their order.

`Game.WaitingDrawStepLifePermanentSources() []DamageSource` reads exact source
references of active `drawStepLifeReplacement` entries in both replacement
lists. It uses the same `IsActive` contract as execution, including saved
incarnation and static rule identity. Stale registrations cannot follow a
returned permanent or an absent ability. A new registration can refer to the
returned incarnation. Current control and characteristics supply source data.
Inactive rules supply no retained source, and pending gains hold only a player
and amount. CR 609.7a permits objects referred to by waiting replacements;
it does not restore an inactive replacement. The shared collector excludes
invalid identities, merges duplicates in list order and copies query data.

`ChooseDamageSourceFromWaitingDrawStepLifeReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` uses the shared pure
filter and choice services. Independent bindings keep independent results;
absence clears only the selected binding. Definition validation checks player,
filter, binding and producer order. RON
`chosenDamageSourcePlan.waitingDrawStepLifeChoice(instanceBindings)` lowers a
validated `ChooseADamageSource(filter)` plan with `You` and its own instance
binding. Full general source choices remain rejected pending all categories.

Evidence: `TestWaitingDrawStepLifeReferences` and
`TestWaitingDrawStepLifeChoices` cover stale registrations, live information,
control, phase cycles, departure, return, ability removal/restoration, duplicate
and invalid references, pending gains, independent bindings and clones.
`TestRONDrawStepLifeFasting`, `TestRONDrawStepLifeFastingUpkeep` and
`TestRONDrawStepLifeInstances` execute the complete Fasting record through
casting, optional skipping, deferred life gain, all five upkeep counters,
destruction on drawing, control and clone changes, and Howling Mine interaction.
`TestRONDrawStepLifeProtection` lowers choice/prevention subrules of all seven
protection names using a live Fasting source with changed color/type; those
records remain unsupported in full. `TestRONDrawStepLifeInvalid` and
`TestRONDrawStepLifeFixedAmounts` cover invalid declarations and fixed amounts.

### Exact waiting counter-prevention subjects

`PreventDamageToSourceByRemovingCounters` now registers a replacement with a
saved source incarnation and static-rule identity. Execution and enumeration
share `counterDamagePreventionReplacement.IsActive`: the original unphased
source must retain that specific runtime ability. A rule retained across a layer
cycle cannot follow a returned object or another counter-prevention ability.
New registration can refer to the returned incarnation. Clones keep independent
counter state and rule lifetimes. Existing callers and the public signature stay.

The rule supplies its source to `Game.WaitingPreventionPermanentSources` through
the shared reference provider. Zero counters prevent a damage match, but do not
remove the active reference. Query data uses current characteristics/control,
exact deduplication across both replacement lists and copied snapshots. LKI cannot
make an inactive static ability active. Nonpositive damage does not match.
CR 400.7, 609.7a, 611.3b, 615.1–6 and 702.26 supply these contracts.

`TestWaitingCounterPreventionReferences` covers stale cycles, departure/return,
phasing, control, ability removal/restoration, snapshots and clone lifetime.
`TestWaitingCounterPreventionBoundaries` covers zero counters, duplicate and invalid
references, distinct rule identity, nonpositive damage and clone counter isolation.
`TestRONWaitingCounterPreventionProtection` executes the choice/prevention subrules
from seven complete protection records against a native counter-rule fixture.
It checks filter lowering, per-instance binding, prevention consumption and empty
choice clearing. `TestRONWaitingCounterPreventionInvalid` rejects malformed filters
and a consumer before its producer. This establishes subrule gameplay evidence;
it does not establish full Rock Hydra or protection-card gameplay. Counter removal,
damage packets, payment and affected-player replacement ordering keep their
existing mechanics. Other waiting replacement categories remain separate gaps.

### Exact permanent sources of legacy finite and half-damage shields

`Game.AddSourcePreventionShield(playerID, sourceID uuid.UUID, amount int)` and
`Game.AddHalfDamageFromSourcePreventionShield(playerID, sourceID uuid.UUID)`
retain the incarnation when `sourceID` is a live permanent at registration.
Their damage match requires that exact incarnation and positive damage to the
specified player. Return cannot transfer the shield to the new object.
Departure and phasing do not end a resolved shield (CR 400.7, 609.7a, 615).
The finite shield retains its remaining amount across damage events and the
shared simultaneous packet pipeline. The half shield prevents the amount rounded
down in its next matching packet and then ends. Existing ordering stays in the
shared replacement service; this change does not establish a new half-shield
simultaneous-event contract.

Both rules supply their exact permanent source to
`Game.WaitingPreventionPermanentSources() []DamageSource` in ordinary and cycle
replacement lists. Current or retained exact information, deduplication, copied
query snapshots and cloned lifetime use the existing collector. Consumption and
cleanup end eligibility. Nonpositive finite amounts install no rule. Registrations
without a live permanent preserve their existing ID-based matching but do not
supply a waiting permanent or spell reference. Typed spell shields keep their
separate services; no permanent-spell continuation is added here.

RON uses the existing `chosenDamageSourcePlan.waitingPreventionChoice` lowering
for validated `ChooseADamageSource(filter)`, its factory-local binding and
`compileExactSourcePrevention` for the chosen-source shield. No duplicate adapter
is required. The seven protection records have choice/prevention subrule tests
against both native shield forms, including departure, return and phasing.
`TestRONWaitingLegacyPreventionInvalid` checks malformed declarations and binding
order. Complete source enumeration and complete protection records remain
unsupported. `TestWaitingLegacyPrevention*` covers native lifetime, information,
return exclusion, clone isolation, finite simultaneous damage and odd half damage.

### Exact permanent sources held by legacy reverse-damage shields

`Game.AddReverseDamageShield(playerID, sourceID uuid.UUID)` now saves the
incarnation of a live permanent at registration. Positive damage to that player
must come from the same incarnation. Departure and phasing keep the shield;
a returned permanent cannot consume it. Nonpositive damage and a consumed
rule cannot match. The existing replacement order, prevention, consumption
before life gain, life-gain replacements and event/history services remain in use.
This change adds no new simultaneous-event or unpreventable-damage contract.

The source joins `Game.WaitingPreventionPermanentSources() []DamageSource`.
The shared collector reads current or exact departure information, including
after LKI cleanup, and deduplicates ordinary and cycle replacement lists.
Queries return independent snapshots. Consumption and end-of-turn cleanup end
eligibility; game clones retain independent lifetimes. Unknown/nonpermanent IDs
retain their existing matching behavior but supply no exact waiting reference.
No spell capture or permanent-spell continuation is established by this API.

RON reuses `compileChosenDamageSourceChoice`, the per-instance
`chosenDamageSourcePlan.waitingPreventionChoice(instanceBindings)` lowering,
and `compileExactSourcePrevention` for the chosen-source consumer. Its supported
category now includes these shield references. `TestRONWaitingReverseDamageProtection`
executes the complete choice/prevention subrules from the six Circle of Protection
names and Greater Realm of Preservation with live, departed, returned and phased
sources and cloned games. `TestRONWaitingReverseDamageInvalid` checks malformed
choices, missing bindings and consumer-before-producer order.
`TestWaitingReverseDamageReferences`, `TestWaitingReverseDamageBoundaries` and
`TestWaitingReverseDamageDeduplication` cover native identity, information,
consumption, histories, compatibility and cloning. Existing reverse-damage and
life-outcome tests cover life-gain replacements and nested damage.

CR 400.7, 609.7a, 615.4–8 and 702.26 support the lifetime and reference contract.
Complete source enumeration remains unsupported. The seven protection names gain
subrule evidence only; the full Reverse Damage record, including its prevented
amount result, is not implemented by this change.

### Temporary exact permanent-source redirection

`RedirectDamageFromPermanentThisTurn(PermanentRef, PlayerRef, PermanentRef)
ResolvingEffect` installs an ordinary end-of-turn replacement through the shared
manager. Typed references must supply an exact identity at installation through the
existing permanent information-reference service. Departure or phasing before
resolution retains that identity, without permitting damage to an absent object. The rule saves source and destination incarnations
and the player separately. Declaration inspection rejects nil references and
uses existing binding dominance validation. No new payment or target service is
introduced, and existing ID-only installers keep their contracts.

Matches require positive damage to the saved player from the exact source
identity carried by the packet. Source departure information can still supply
that identity (CR 400.7, 609.7). The source need not still be attacking after
resolution. The destination must be the original live, unphased creature or
planeswalker (CR 614.9, 702.26b); return cannot restore it. Replace checks these
conditions again. Source, amount and combat status remain unchanged for the
shared replacement/prevention pipeline (CR 120.4, 614.5, 616). The rule is not
consumed by damage. Cleanup removes it; clones copy its captured state.

The rule supplies both saved permanent references to
`WaitingRedirectionPermanentSources`, using the same collector and information
service as other waiting categories (CR 120.7, 609.7a). Missing destinations,
phasing and type changes do not erase a resolved rule's referred objects. Queries
use current or exact retained information without executing replacement actions.
Cleanup ends eligibility. Static redirection still uses its live context.

Evidence: `TestTemporaryRedirectIdentity` covers departure/return, cleared LKI,
source and destination phasing, phase cycles, control/type changes, wrong-player
exclusion, repeated damage, cleanup and clones. `TestTemporaryRedirectBoundaries`
covers nonpositive packets, absent installation, invalid declarations and
same-object deduplication. `TestTemporaryRedirectDamageBatch` covers simultaneous
packets, prevention after redirection, actual outcomes, lifelink, source/combat
identity and destination rechecks. `TestRONTemporaryRedirectGameplay` executes the
complete Shimian Night Stalker record with real activation costs and target
checks at announcement/resolution. `TestRONTemporaryRedirectCopyAndInstances`
covers per-instance targets/destinations and a copy with a different controller.
`TestRONTemporaryRedirectInvalid` rejects unsupported and unbound declarations.
`TestRONTemporaryRedirectProtection*` maps the seven protection records' actual
choice/prevention subrules to this producer category; complete source enumeration
and complete protection-card gameplay remain unimplemented.

Creature-recipient-to-player redirection, other legacy replacement references,
spell-source redirection, optional redirection and face-up command
objects remain separate capabilities. No RON source change or new general source
choice is established here.

### Selected-source permanent-recipient redirection

`RedirectNextDamageFromSelectedSource(source DamageSourceRef, subject PermanentRef,
player PlayerRef, filter DamageSourceFilter) ResolvingEffect` saves an exact source,
permanent incarnation and destination player at resolution. A zero filter permits
all sources. The filter checks current or retained exact-source information at
installation and damage time. A mismatch does not consume the rule (CR 609.7b).

Positive damage from the selected source to the original live creature or
planeswalker becomes damage to the saved player (CR 614.7a, 614.9). Subject phasing,
departure or loss of both types stops applicability; a returned object cannot
restore it. A phase cycle preserves it. Source departure or phasing can still
supply captured damage. Source and subject control changes cannot change the
saved player. The shared pipeline preserves source attribution, amount and combat
status, affected-player ordering, recipient prevention and actual damage results.
A prohibition on redirection keeps the rule unused.

One matching simultaneous damage event consumes the rule after all its matching
packets are redirected (CR 614.3–5). Later damage passes normally. Cleanup ends
unused rules. Clones retain independent state and copied source information.
A selected permanent spell continues to the exact permanent it becomes before
entry effects run (CR 609.7a). Separate spell copies and later returns do not
inherit that continuation. A rule consumed by spell damage cannot continue.

`WaitingRedirectionPermanentSources` includes the subject, selected permanent
source and continued permanent source until consumption or cleanup.
`WaitingRedirectionSpellSources() []DamageSource` includes the selected spell,
using the same exact-stack information and snapshot collector as waiting
prevention. `ChooseDamageSourceFromWaitingRedirectionReferences` now reads both
redirection categories. It does not enumerate other legal source categories.
Legacy ID-only replacements retain their existing behavior.

Evidence: `TestSelectedSourceRedirectIdentity`, `SpellContinuity`, `SpellCopy`,
`WaitingSpell`, `Rechecks`, `DamageContract` and `Invalid`; RON
`TestRONSelectedSourceRedirectAction`, `Invalid` and `Protection`. The RON action
fixture executes Jade Monolith's unchanged target and replacement AST with an
explicit source list. Seven protection records supply actual choice/prevention
ASTs for this producer. These are subrule evidence; all eight complete names
remain rejected pending complete source enumeration. No full-card claim is added.

### Waiting counter-placement references: explicit source exclusions

`Game.WaitingCounterPlacementPermanentSources() []DamageSource` supplies only
an `etbAdditionalCountersReplacement` source explicitly excluded by
`excludeSelf=true`. A general counter filter or counter doubler does not name
all its possible members and supplies no references through this category.
CR 609.7a permits objects referred to by waiting replacements even when they
cannot deal damage. The source need not match the entry filter.

`AddETBAdditionalCounters` retains the live source incarnation at registration.
`IsActive`, `Matches`, `Replace` and entry-counter synthesis use that identity.
Departure, phasing or absent registration stops execution and enumeration;
a phase cycle restores the same object, while a returned card needs a new
registration. `Replace` rechecks applicability. Cleanup does not expire this
battlefield-duration rule. This direct registration API retains its existing
explicit-registration lifetime; it does not become a typed static ability or
acquire an ability-removal contract. Static registration owners still manage
registration removal. Counter amounts, entry timing, filters, source exclusion,
affected-player ordering and counter batches use the existing pipeline.

The shared collector reads both replacement lists, rejects invalid identities,
deduplicates exact references and supplies isolated current-information snapshots.
Clones retain separate registration state. General filters and rule provenance
alone do not create referred objects.

`ChooseDamageSourceFromWaitingCounterPlacementReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` uses the shared choice,
filter and binding services. The RON adapter
`chosenDamageSourcePlan.waitingCounterPlacementChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and the instance's own binding.
Missing bindings and reads before a producer fail definition validation. Empty
or mismatched choices clear only that binding. General RON source enumeration
remains rejected pending the remaining categories.

Evidence: `TestWaitingCounterPlacementReferences` covers lifecycle, stale
registrations, current control, entry execution, cleanup and clones.
`TestWaitingCounterPlacementChoices` covers excluded/generic/invalid/duplicate
references, filters, independent results, declaration validation and self/entry
restrictions. `TestRONWaitingCounterPlacementProtection` executes the actual
choice/prevention ASTs of the six Circles and Greater Realm with this category,
including consumption, phasing, clearing and clones. These are subrule tests;
none establishes complete source enumeration. `TestRONWaitingCounterPlacementInvalid`
rejects malformed declarations and invalid producer order.


### Waiting legacy cycle redirection destinations

`Game.WaitingRedirectionPermanentSources() []DamageSource` also reads exact
permanent destinations in `bodyguardReplacement`,
`playerDamageRedirectReplacement` and `artifactDamageRedirectReplacement`.
`BodyguardContinuous() ContinuousEffect`,
`PersonalIncarnationRedirect() ContinuousEffect` and
`Game.SetArtifactDamageRedirect(controllerID, permID uuid.UUID)` retain their
signatures and save the live destination incarnation at registration.

The query, `IsActive`, `Matches` and direct `Replace` agree on that identity.
Phasing and departure stop the rule. A phase cycle restores the same object;
a returned card needs a new registration. The bodyguard rule also requires its
live destination to be untapped. The artifact installer does not impose that
condition. General source filters and registration provenance do not add named
references. Current source information and isolated snapshots use the shared
collector, including exact deduplication across both replacement lists.

These legacy registrations retain their explicit cycle ownership. Layer rebuilds
clear cycle entries and owners must register them again. This change does not
convert legacy continuous callbacks into typed static abilities or give retained
manual registrations an ability-removal contract. Existing ID-only native test
fixtures retain their matching behavior but cannot supply exact references.
The legacy player redirect remains its existing engine operation; it is not an
implementation of the Personal Incarnation RON declaration.

RON `chosenDamageSourcePlan.waitingRedirectionChoice(instanceBindings)` reuses
`ChooseDamageSourceFromWaitingRedirectionReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect`. Validated
`ChooseADamageSource(filter)` plans lower with `You` and a per-instance binding.
Invalid filters, missing bindings and a consumer before its producer fail
validation. Empty choices clear only their own binding. The existing exact-source
prevention action consumes the chosen reference. General source enumeration
continues to reject full protection declarations.

CR 400.7, 609.7a and 702.26b establish the exact-object and phasing boundaries.
Damage amount, combat status, source attribution, replacement choices and
simultaneous packets retain the shared pipeline (CR 614.5, 616.1).
Evidence: native `TestWaitingLegacyRedirectionReferences`, `Choices`,
`DamageAndCycle` and `ReplacementOrder`; RON
`TestRONWaitingLegacyRedirectionProtection` and `Invalid`. The seven protection
names supply their complete real choice/shield ASTs for this source category.
These are subrule tests, not full-card gameplay claims.

### Waiting references in resolved legacy temporary redirection

`Game.SetCreatureDamageRedirect(creatureID, playerID uuid.UUID)` captures the
live subject incarnation for its existing one-use redirect. The subject remains
an exact waiting reference until the rule is consumed or cleanup removes it.
`Game.SetAttackerDamageRedirect(attackerID, absorberID uuid.UUID)` captures both
live incarnations. Its source and destination remain waiting references until
cleanup, including after departure or phasing. Neither installer registers an
absent permanent. The creature installer also requires a present player.

`Game.WaitingRedirectionPermanentSources() []DamageSource` includes these
references through the shared collector. Queries use current, phase or retained
incarnation information and deduplicate exact objects. Clearing ordinary LKI
cannot erase retained damage-source information. A returned permanent cannot
replace that information. Query snapshots and cloned games stay independent.
These resolved registrations differ from live static registrations: an absent
destination stops damage redirection, but does not erase the resolved reference
(CR 400.7, 609.7a, 702.26b).

Both `Matches` and direct `Replace` check action kind, positive damage, a present
player and the original live damageable destination (CR 614.9). The creature
redirect also checks consumption. The attacker redirect checks the packet's
exact source identity, including departure damage, and does not require the
source to remain attacking. Its legacy scope continues to permit damage to
either player. Amount, source and combat status remain unchanged. Existing
replacement ordering, prevention and batch execution stay in the shared damage
services (CR 614.5, 616.1). The creature service retains its existing single-packet
consumption contract; this is not a new RON next-time redirection operation.
Internal ID-only replacement fixtures retain legacy matching and supply no exact
waiting references.

RON `chosenDamageSourcePlan.waitingRedirectionChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` through the existing typed
`ChooseDamageSourceFromWaitingRedirectionReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` service. The existing
exact-source prevention lowering consumes that choice. No additional spelling
adapter or complete source enumeration is introduced. Invalid declarations and
missing or premature bindings remain errors; empty choices clear their binding.

Evidence: native `TestWaitingTemporaryLegacyRedirectReferences`, `Rechecks`,
`SourceIdentity`, `ConsumptionAndBatch`, `MissingAndDuplicate`, `ReplacementOrder`
and `AbsentInstallation`; RON `TestRONWaitingTemporaryLegacyRedirectProtection`
and `Invalid`. The RON tests execute the real choice/prevention subrules of the
six Circles of Protection and Greater Realm of Preservation, with live, departed,
returned and phased references in original and cloned games. The attacker fixture
phases its destination after choice so redirection cannot intercept the damage
used to check prevention. Complete protection declarations still fail compilation.
Jade Monolith and Reverse Damage remain downstream consumers with separate
operation gaps. Other explicit legacy replacement references and face-up command
objects still need contracts before general source choice can be installed.

### Face-up command objects as damage-source choices

`Game.AddCommandObject(card Card, faceDown bool) (uuid.UUID, error)` installs
an engine object in the command zone with a fresh visit identity. The caller
must remove it from its former zone first. Nil objects, missing IDs and duplicate
IDs in the command zone fail without a state change. This placement service does
not grant casting permission or install command-zone abilities or variant rules.
`Game.RemoveCommandObject(visit uuid.UUID) (Card, bool)` removes only that visit.
`Game.SetCommandObjectFaceDown(visit uuid.UUID, faceDown bool) bool` changes its
visibility without changing identity. Unknown visits have no effect.

`Game.CommandDamageSources() []DamageSource` supplies all face-up installed
objects in insertion order (CR 609.7a, 408.1). An object needs no damage ability.
The query excludes face-down objects and does not inspect the battlefield or
stack. The engine uses its existing `Card` representation for these objects.
Command objects are not permanents; permanent destruction and phasing cannot
find them. `FindCardAnywhere` includes command storage with an isolated snapshot.

`DamageSource.CommandVisit() uuid.UUID` distinguishes command entries from
permanent incarnations and stack spell identities. `SameObject` requires exactly
one of those three identities, together with the same object ID. A return cannot
inherit a chosen source or shield. Current face-up information supplies the
stored object's name, owner, colors and types. A command card has no controller
(CR 108.4). The visibility service gives hidden objects no public characteristics.
Removal retains the information at departure by visit; a hidden departure cannot
restore a previous face-up snapshot. Game clones copy live and departed storage
and query snapshots independently. Retained visits survive ordinary LKI cleanup.

`ChooseDamageSourceFromCommandObjects(player PlayerRef, filter DamageSourceFilter,
binding *DamageSourceBinding) ResolvingEffect` queries at resolution and uses
shared filters, choice validation and per-instance result bindings. Empty choices
and absent players clear the selected binding. Definition validation checks the
player, filter, binding and producer order. RON
`chosenDamageSourcePlan.commandChoice(instanceBindings) mage.ResolvingEffect`
lowers validated `ChooseADamageSource(filter)` with `You` and its own binding.
The compiler keeps unrestricted source choices rejected until all categories
and explicit references have been composed and audited.

Exact-source prevention, selected-source prevention and selected-source
redirection now accept command visits through the shared identity check.
Their existing replacement order, event consumption and simultaneous damage
services remain authoritative. Property checks use current command information
or exact departure information. Captured delayed damage bindings include command
visits with deterministic identity ordering and the same information service.
No new cost or combat rule is introduced.

Evidence: `TestCommandDamageSourceVisits`, `ChoiceAndPrevention`, `Declarations`,
`RetainedVisibility`, `BatchAndOrder`, `Redirect`, `ZoneLookup` and
`CapturedReferences` cover identities,
visibility, departure/return, isolated storage, choices, declarations, replacement
order, simultaneous damage, consumption and clones. All names have the
`TestCommandDamageSource` prefix. `TestRONCommandDamageSourceProtection` uses
actual choice/prevention bodies from all six Circles and Greater Realm of
Preservation. `TestRONCommandDamageSourceTimingAndCopies` uses the Red Circle's
actual cost and body with a test-only category restriction, including failed
payment, resolution-time choice, stack copies and clones.
`TestRONCommandDamageSourceInvalid` checks invalid syntax and result scope.
These are subrule tests; they do not establish full-card gameplay or new loader
acceptance. Waiting rules that retain command visits need an explicit reference
collector before complete CR 609.7a enumeration is enabled.

### Waiting command visits in prevention and redirection

`Game.WaitingPreventionCommandSources() []DamageSource` reads exact command
visits from active exact-source and selected-source prevention shields.
`Game.WaitingRedirectionCommandSources() []DamageSource` reads the selected
command source of active typed redirection rules. The collector examines both
ordinary and cycle replacement storage, excludes other identity kinds and
invalid identities, deduplicates visits, and returns isolated snapshots.

CR 609.7a permits these references after departure. CR 400.7 keeps a returned
object separate. Current or exact departure information supplies properties;
hidden objects expose no characteristics. A retained reference does not require
the redirection recipient to remain present or unphased. Consumption and cleanup
remove availability. Existing replacement ordering and damage batches apply.

`ChooseDamageSourceFromWaitingCommandReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` chooses
from both query results at resolution through the shared choice service.
Definition validation checks player, filter, binding and producer order.
Nonmatching or empty queries and absent players clear only that result binding.
Game clones retain independent references and consumption.

RON `chosenDamageSourcePlan.waitingCommandChoice(instanceBindings)
mage.ResolvingEffect` lowers validated `ChooseADamageSource(filter)` with `You`
and the plan's per-instance binding. Full source choices stay rejected until the
remaining explicit-reference audit and category composition are complete.
No RON record or source hash changes.

Evidence: native `TestWaitingCommandSources`, `VisibilityAndCleanup`,
`Declarations`, and `ExcludeOtherIdentities` (all use the
`TestWaitingCommandSources` prefix); RON `TestRONWaitingCommandSourceProtection`,
`Invalid`, and `TimingAndCopies`. The RON tests use the complete records of six
Circles and Greater Realm of Preservation for their actual choice/prevention
bodies, plus the Red Circle's activation cost. These tests establish subrule
coverage, not full-card acceptance or complete gameplay validation.

### Explicit sources of waiting callback modifiers

`Game.WaitingModifierPermanentSources() []DamageSource` supplies the explicit
permanent-source context of registered mill and life-gain modifiers and their
static factories. These services are legacy amount modifiers, with their existing
registration order; this collector does not give them CR 616 replacement choices.
CR 609.7a permits an object referred to by a waiting replacement. A callback's
explicit registration source, or the source passed to its static factory, supplies
that reference. Arbitrary objects hidden in closures do not gain a reference.

Registered modifiers require a nonnil function, a saved nonzero incarnation and
the same live unphased permanent. A legacy life modifier registered without a
source cannot supply an exact reference. Departure ends the registration's
availability; retained information and a returned card cannot restore it
(CR 400.7). A fresh registration can supply the new incarnation. Removing an
ability does not remove a separately registered modifier. Static factories use
current runtime abilities; departure, phasing and ability removal stop their
availability. A returned static source supplies its new incarnation.

Queries do not call functions or factories, prune registrations, make choices,
or write game state. References use current properties, shared deterministic
identity ordering and deduplication, and isolated card snapshots. Multiple
registrations or static factories for the same object supply one option. Clones
retain independent source state and registrations through existing clone services.

`ChooseDamageSourceFromWaitingModifierReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` reads
this category at resolution through the shared source-choice service. Declaration
validation checks the player, filter, result binding and producer order. Empty or
nonmatching lists and absent players clear only the selected result. The RON
`chosenDamageSourcePlan.waitingModifierChoice(instanceBindings)
mage.ResolvingEffect` adapter uses `You` and the per-instance binding from validated
`ChooseADamageSource(filter)`. Unrestricted production choices remain rejected.

Evidence: `TestWaitingModifierSourcesLifecycle` and `Choices` cover registered
and static mill/life services, no callback execution during queries, malformed
references, duplicate registration, phase cycles, departure/return, control and
ability changes, isolated query data, clone execution and declaration rejection.
`TestRONWaitingModifierSourceProtection` maps the six Circles and Greater Realm
of Preservation to their actual cost and choice/prevention bodies in this category.
`TestRONWaitingModifierSourceTimingAndRechecks` covers the Red Circle's
resolution-time registration, property rechecks, simultaneous damage, consumption
and clone isolation. `TestRONWaitingModifierSourceInvalid` covers invalid syntax
and binding order. These establish subrules, not full-card acceptance. Opaque
callback-held references, other replacement families and complete source-list
composition still need contracts.


### Exact sources of waiting empty-library draw replacements

`Game.WaitingEmptyLibraryPermanentSources() []DamageSource` reads the explicit
source passed to active `AddEmptyLibraryDrawReplacement` callbacks. CR 609.7a
permits this referred object as a damage source. Queries do not call callbacks,
make choices, prune registrations or change game state. A nonempty library does
not end a waiting rule. A nil callback supplies no reference. Objects hidden in
callback closures are outside this contract.

Registration captures the live permanent's incarnation. Both replacement
execution and source enumeration require that same live unphased incarnation
(CR 400.7, 702.26b). Phase-in restores availability; departure or return cannot
restore a saved rule. Registration before source entry cannot follow that entry.
Control changes update queried characteristics without changing the registered
draw player. Removing abilities does not remove a separate registration.
Callbacks retain their existing signature, and matching retains the explicit
player and empty-library check for normal and effect draws. The shared CR 616
pipeline selects among competing replacements; this slice adds no ordering path.

`ChooseDamageSourceFromWaitingEmptyLibraryReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` uses
shared definition validation, source filters, result clearing and isolated card
snapshots. Clones retain independent registration and permanent state. The RON
`chosenDamageSourcePlan.waitingEmptyLibraryChoice(instanceBindings)` adapter
uses `You` and its per-instance binding. Unrestricted choices remain rejected.

Evidence: `TestWaitingEmptyLibrarySourcesLifecycle`, `Choices`,
`SavedRegistrations`, and `ReplacementOrderAndClones` cover query purity,
deduplication, identity, phasing, departure/return, library contents, player and
control checks, separate registration after ability removal, invalid declarations,
absent-player and nonmatching result clearing, replacement choices and clones.
`TestRONWaitingEmptyLibrarySourceProtection` maps all six Circles and Greater
Realm of Preservation to their actual RON payment and choice/prevention bodies.
`TimingAndRechecks` adds resolution timing, property rechecks and simultaneous
packets; `Invalid` checks syntax and producer order. These are category/subrule
tests, not complete source enumeration or full-card gameplay evidence.
