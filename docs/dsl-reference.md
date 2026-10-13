# DSL reference

## Saved sacrifice subtype results

`SacrificeChosenBy(player, chooser PlayerRef, subjects PermanentSet,
options ...SacrificeResultOption) ResolvingEffect` accepts
`SacrificeSubtypeInto(subtype string, result *NumberBinding)`.
`SacrificeMatchingIncludingSourceCostWithSubtype(filter PermanentFilter,
text, subtype string, result *NumberBinding) Cost` supplies the same saved
property through the existing activation payment transaction. The source is
eligible when it matches the cost filter. Existing callers keep their signatures.

Each output saves one for a completed sacrifice of a permanent with that subtype,
or zero otherwise. The property is read after replacement checks and before
departure. Replacement to exile still supplies a sacrifice; cancellation,
absence, phasing, a returned visit or loss of required control does not. A modified
cost can still be paid without supplying a successful sacrifice property
(CR 118.11). Payment validation and rollback use the shared transaction. Saved
numbers belong to the exact resolution or announced stack object; copies, clones
and delayed bodies keep separate values. Later objects and characteristics cannot
change the result. No historical result permits an action on an absent permanent.
Selection, owner destinations, event LKI and replacement ordering use existing
engine services. Waiting queries do not choose subjects or execute bodies.

RON supports resolving
`If(APermanentWasSacrificedThisWay(IsLandType(type)), [actions...])`
for Plains, Island, Swamp, Mountain, Forest and Desert, and the corresponding
`IsCreatureType(type)` form with subtype-family validation. A direct resolving
`SacrificeAPermanent(collection)` or
`SacrificeAPermanentOfAPlayersChoice(collection, chooser)` supplies the following
local continuation. A single matching `SacrificeAPermanent(filter)` activation
cost can instead supply its resolving body. Each producer supports one subtype;
repeated reads of that subtype are allowed. A later direct producer starts a
distinct result. Producers inside branches or scoped player bodies do not escape
their local sequence. Reads before producers, ambiguous costs, wrong subtype
families, additional casting-cost consumers, combined numeric characteristic and
subtype captures, other property filters and alternative-condition forms reject
at compilation. This is a resolving predicate, not a static condition.

## Complete global land-mana-event battlefield text

`Game.AddPrintedManaTapObjectTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` checks one complete global mana-event rule before
installing layer 3. The card must satisfy the existing
`WithManaTapSpellTextContinuation()` context: one marked battlefield
`GenericTriggered` or global `ManaBonusAbility`, without keywords, sibling rules,
casting targets, costs or modes. Attached and mixed rules use separate contracts.
Duration is `Indefinite` or `EndOfTurn`; changes must be nonempty and validated.

Preflight rebuilds the retained whole definition, checks its declaration group,
native ability kind, subtype context, targets, effect count, event capture flags
and bonus recipient/produced-type context, and checks the active runtime rule.
Fixed bonus mana symbols cannot change. Missing builders/groups, changed
contexts, absent or returned visits, face-down/phased sources and removed rules
reject before any layer is installed. A valid no-match substitution installs a
layer. `ChangePrintedPermanentText` now dispatches this complete context.

RON uses the existing resolving
`CreatePermanentLayerEffect(permanent_ref, [substitution, ...])` lowering and
lexically saved color or basic-land word pairs. Its shared `manaTapTextRules`
selector marks exactly one global
`TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer, IsCardtype(Land)), body)`;
existing action and scope validation still rejects unsupported declarations.
Stack continuation and battlefield dispatch use the same complete marker.

The existing declaration layer restores ordered baselines on expiry. Mana
bonuses keep the shared planner, actual replaced types and immediate production;
ordinary triggers keep stack timing and detection facts. Control, phasing,
cloning, pending copies, removal, LKI, printed return and copiable values use
existing services (CR 400.7, 113.7a, 605.1b/4a/5a, 612.1–3, 613.1c/f, 707.2).
No new cost, replacement-ordering or simultaneous-operation service is added.

Evidence: `TestRONTextManaTapObjectRecords` covers all 12 complete subject records
for Mana Flare and Manabarbs, with both real Magical Hack and Sleight of Mind
consumers (24 cases). It checks choices, payment, no-match layer installation,
IDs, mana symbols, timing, recipient, pending clones, LKI and recasts.
`QueuedSnapshots`, `ProducedReplacement`, `PaymentRollback` and `Boundaries`
cover revised pending target filters, copies/clones, replaced multi-type mana,
manual/planned agreement, failed payment rollback, invalid lexical declarations
and foreign object contexts. Native `TestTextManaTapObjectRejectsAtomic` covers
malformed definitions and runtimes; `Lifecycle` covers observable ordered word
changes, expiry while phased, control, removal/restoration and printed copies.
Compilation acceptance is unchanged; general subject coverage remains incomplete.

## Single global ability-grant spell text continuation

`WithGlobalStaticSpellTextContinuation() CardOption` now also accepts one
`StaticAbilityHolder.WithLayeredGrantTextRevision()` declaration on a nonland,
non-Aura permanent. The holder has battlefield ability-layer effects and may
include P/T modifications in layer 7c. Intrinsic keywords, sibling rules, modes,
casting targets and additional or alternate casting costs remain excluded.

RON selects exactly one `EachPermanentLayerEffect(filter, parts)` whose parts
are fixed `AdjustPT(integer, integer)` and nonempty `AddAbility([keyword, ...])`
or fixed basic `Landwalk` grants, with at least one grant. Existing lowering
checks filters and lexical scope. Activated, triggered and other grant contexts
keep their separate paths. The selector does not extend paired static/event or
static/activation continuation contexts.

`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
checks the exact stack visit, retained builder and declaration correspondence.
Fresh rules keep the grant context, effect count, ordered layers, sublayers,
durations and subtype context. Invalid changes, missing builders/groups and
foreign announced targets, bodies or casting results reject atomically.
Payment and mana symbols retain their printed values. Fresh entry preserves
physical identity, owner and printed departure rules.

The existing shared-subject grant service selects current recipients in layer 6
and keeps them through paired P/T in layer 7c. Control, color, source exclusions
and late entrants remain live. Later grant text layers use the continued rule
as their baseline; expiry restores it. Phasing keeps the visit and removal
suspends the donor. Clones and copied spells retain separate rules; spell copies
enter as tokens. Copiable values exclude ordinary word changes. LKI keeps final
results, and physical recasts use printed rules (CR 400.7, 612.1–3, 613.1c/f,
613.4c, 613.6 and 707.2).

Evidence: `TestRONTextGlobalGrantSpellRecords` checks every complete record for
seven names (17 records): Goblin King, Lord of Atlantis, Fortified Area, Kobold
Drill Sergeant, Concordant Crossroads, Goblin War Drums and Hidden Path.
`Lifecycle`, `CopiesAndMembership` and `SharedSubjects` cover layers, expiry,
phasing, suppression, evasion, LKI, printed recasts, copies, clones and live
membership. `Boundaries` retains unsupported sibling contexts and invalid
syntax rejection. Native `TestTextGlobalGrantSpellDeclaration` and
`RejectsAtomic` check definition and atomic revision boundaries.
This is a prerequisite for word-changing spell consumers. Whole-object dispatch,
multiple grants, intrinsic-keyword siblings and copy-added rules remain separate.

## Typed land-mana-event permanent-spell text continuation

`WithManaTapSpellTextContinuation() CardOption` marks a nonland, non-Aura
permanent with one `GenericTriggered.WithLayeredManaTapTextRevision()` rule
or one global `ManaBonusAbility.WithLayeredManaTapTextRevision()` rule.
Intrinsic keywords, sibling abilities, modes, casting targets, additional or
alternate costs and other continuation markers reject at definition validation.
Attached events and bonuses require a separate entry contract.

RON selects exactly one existing
`TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer, IsCardtype(Land)), body)`.
The existing compiler validates all filters, actions, targets and lexical scopes.
The exact produced-type bonus uses the shared mana planner and production
service; other supported bodies use their typed trigger and normal stack timing.
No unsupported body or declaration becomes accepted through this recognizer.

`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
checks the retained semantic builder, declaration group and exact stack visit.
Fresh rules must retain the native ability kind, event capture flags, event
attachment and type-filter context, target-group shape and body count. Bonuses
retain fixed/produced-type and recipient context. Subtype context stays fixed.
Empty or invalid changes, missing builders/groups, changed contexts, departed
visits and foreign announcement bodies, targets or casting results reject
atomically. Physical identity, owner, controller, payment and cast facts remain.
Each fresh build has separate targets and player/permanent/mana-type bindings.

Entry retains ordered changes with the physical card's identity and printed
departure rules (CR 400.7a, 608.3a). Spell copies enter as tokens; game clones
keep separate continued rules (CR 707.10f). Mana Flare retains immediate mana
production, actual replaced types and planner agreement. Manabarbs retains stack
timing and the exact saved event player (CR 605.1b, 605.4a, 605.5a).
Later declaration text layers use the continued baseline. Expiry restores it;
phasing retains the visit; ability removal suppresses the rule. Pending triggers
and copied triggers keep prior targets, rules and event facts after departure.
Copy values exclude ordinary word changes; departure LKI keeps final rules;
recasts use printed rules (CR 612.1–3, 613.1c/f, 707.2). A copy exception that
adds Land clears the spell-only marker while retaining its mana-event rule.

Evidence: `TestRONTextManaTapSpellRecords` casts and revises all 12 complete
records for Mana Flare and Manabarbs. Inventory maps exactly those two names.
Lifecycle covers revised targets, ordered changes, token copies, independent
stack/game clones, later layers, expiry while phased, removal/restoration,
control, copiable values, LKI and printed recasts. ManaProduction covers replaced
multi-type production, manual/planned agreement, cost rollback and clone/phasing
isolation. PendingEvents covers separate saved players, trigger copies and
source departure. IllegalTargets covers resolution legality. CopyExceptions
covers an added Land type. Boundaries rejects invalid syntax/scopes and retains
unsupported mixed and Aura contexts. `TestTextManaTapSpell*` covers native
markers, nil rules, incompatible contexts and rejection without mutation.
Mixed and Aura contexts retain separate contracts. Whole-object land-mana-event
dispatch uses AddPrintedManaTapObjectTextEffect. No RON source changed.

## Typed cast-event permanent-spell text continuation

`WithSpellCastSpellTextContinuation() CardOption` marks a permanent with one
`GenericTriggered.WithLayeredSpellCastTextRevision()` declaration.
`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
validates and retains fresh rules for that exact stack visit through entry.
The card must be a nonland, non-Aura permanent with no intrinsic keywords,
other abilities, modes, casting targets, additional or alternate costs, or
other permanent-spell continuation marker. RON selects exactly one existing
`TriggerA(WhenAPlayerCastsASpell(AnyPlayer | Opponent, spellFilter), body)`.
Existing filter, action, target and lexical binding checks remain mandatory.

Revisions require a retained semantic builder and matching printed declaration
group. They preserve caster/capture context, subtype context and target-group
shape. Announced targets, resolving bodies and saved casting results are outside
this contract. Invalid words, empty changes, missing builders/groups, changed
contexts and countered visits reject before mutation. The physical card,
controller, payment and cast facts remain unchanged. Fresh trigger bindings and
resolution payments are not read or paid at announcement.

Entry uses the shared permanent-spell pipeline with physical ID, owner and
printed departure rules (CR 400.7a, 608.3a/e/f). Spell copies enter as tokens;
clones keep separate continued rules, event captures and payment results.
Current control supplies the trigger controller at detection. Pending triggers
keep detection-time rules and references through later revisions, copies,
phasing and departure. Trigger targets use revised filters at placement and
recheck legality at resolution. Later layer-3 changes use the continued
baseline; expiry restores it, and layer-6 removal suppresses detection.
Phasing retains the visit. LKI keeps continued rules, while return uses printed
rules. Ordinary word changes stay outside copiable values (CR 612.1–3,
613.1c/f, 707.2–3, 707.10f).

The existing definition builder, typed cast event, transactional optional
payment, targeting, layer, clone, copy provenance and LKI services supply these
rules. No payment, replacement ordering or simultaneous-operation contract is
changed. Copy type additions clear this spell-only marker on Land prototypes
and retain their trigger and semantic builder. Authored invalid markers reject.
Mixed rules, intrinsic-plus-trigger spells, other event contexts, whole-object
word dispatch and the word-changing spells need separate contracts.

Evidence: `TestRONTextSpellCastSpellRecords` executes all 35 complete records
for Crystal Rod, Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere, Citanul
Druid, Urza's Chalice, In the Eye of Chaos, Nether Void and Presence of the Master.
`Lifecycle`, `EventBindingsAndTargets`, `IndependentPayments`, `CopyExceptions`,
`Inventory` and `Boundaries` cover continued gameplay and context checks.
Native `TestTextSpellCastSpellDeclaration` and `RejectsAtomic` cover malformed
markers and atomic revision. Existing `TestRONSpellCastLife*` payment rollback
and `TestRONTextSpellCastLayer*` contracts remain unchanged.

## Printed opponent-land-tap trigger text layers

`GenericTriggered.WithLayeredLandTapTextRevision() *GenericTriggered` opts in
one printed typed battlefield opponent-land-tap declaration.
`Game.AddPrintedLandTapTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` installs validated color
or basic-land word changes in layer 3. Duration must be `EndOfTurn` or
`Indefinite`; at least one change is required. A cloned probe checks the retained
semantic builder and printed group correspondence before live registration.
Invalid sources, visits, words, durations, foreign or granted abilities, missing
builders/groups, removed declarations and incompatible rebuilt contexts reject
without changing live rules. Rebuilds keep the layer marker, subtype-filter shape,
event-land capture shape, active UUID and declaration position.

RON marks its complete existing
`TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(subtype),
ControlledByAPlayer(Opponent)])), body)` context. Either filter order works.
Existing land subtype validation, resolving bodies, targets and lexical scopes
remain in `ron`; each factory rebuild supplies fresh declarations and bindings.
Other tap events and unbound event references retain their separate contracts.

Each layer pass restores the saved baseline, then applies surviving changes in
timestamp order (CR 612.1–3, 613.1c, 613.7). Expiry recomputes surviving changes
from that baseline. A prior direct revision can supply it. Direct revision cannot
change a layered result. Event-time land/control facts and pending rules keep
their detection-time text, targets and identities through installation, expiry,
copies and departure (CR 603.2–3, 113.7a). Current source-controller information
reads retain live or LKI behavior. Clones keep separate rules, controller contexts,
baselines and cleanup. Phasing keeps the visit, including expiry while phased.
Layer 6 removal suppresses detection; restoration uses current text. Departure
LKI keeps the final rule; returned objects use printed rules (CR 400.7).

Evidence: `TestRONTextLandTapLayerRecords` and `OrderingExpiry` execute all
six Lifetap records and Lifeblood. `Lifetime`, `RemovalAndTargets`,
`PendingTargets`, `ColorTargetsAndSibling`, `DirectBaseline`,
`ControllerIsolation`, `PaymentAndBatch` and `Boundaries` cover the remaining
lifecycle and legality contracts. Native `TestTextLandTapLayerDeclaration`,
`Rejects` and `Bindings` cover validation, atomic rejection and exact event-land
captures. Existing `TestRONOpponentLandTap*` remains unchanged. This prerequisite
adds layer gameplay coverage for two unique names / seven records, with zero
collection acceptance gain. Whole-object text dispatch, spell text and copy
rebinding remain separate capabilities.


## Printed combat-pair trigger text layers

`GenericTriggered.WithLayeredCombatPairTextRevision() *GenericTriggered` marks
one printed typed battlefield combat-pair declaration. It includes the existing
checked revision opt-in and validation.
`Game.AddPrintedCombatPairTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` installs ordered color or
basic land word changes in layer 3 with `EndOfTurn` or `Indefinite` duration.

RON marks its complete existing `TriggerA` pair context: two-argument
`WhenACreatureBlocksACreature`, `WhenACreatureBecomesBlockedByACreature`, or
their matching two-member `Or`. The subject is `SinglePermanent(ThisPermanent)`
or `SinglePermanent(HostPermanent)`. No new filter, body, event or zone shape is
accepted. RON keeps spelling, validation and fresh opponent/target bindings.

Each pass restores one baseline per source visit and declaration before ordered
changes (CR 612.1–3, 613.1c, 613.7). A prior direct revision can supply that
baseline. Expiry recomputes surviving changes. Direct revision cannot overwrite
a layer result. Rebuilds keep the active ID, declaration position, event
direction, source/host context and opponent capture shape. Installation checks
a cloned probe before registration. Invalid words/durations, empty changes,
foreign/granted/unmarked/removed declarations, absent/phased/face-down/stale
sources, missing builders/groups and incompatible rebuilds reject without
changing live rules or layer state.

Pending abilities keep detection-time text, targets and exact opponent
incarnations through installation, expiry, source departure, copies and delayed
end-of-combat bodies (CR 603.2–3, 400.7). Current source controller reads retain
the existing LKI contract. Layer 6 removal suspends detection; restoration uses
current text. Phasing keeps the visit, including expiry while phased. Clones
keep independent baselines, cleanup and controller contexts. Departure LKI keeps
the final rule; return uses printed rules. The existing definition rebuilder,
trigger retention map, layer/timestamp, event capture, target, delay and clone
services supply execution. No payment, replacement or simultaneous-operation
contract changes. Whole-object text, spell text and copy rebinding remain
separate capabilities.

Evidence: native `TestTextCombatPairLayerDeclaration` and `Rejects`; RON
`TestRONTextCombatPairLayerRecords`, `BodyAndLifetime`, `AuraAndRecords`,
`Boundaries`, `PendingTargets`, `PaidCastAndNativeBlocks`, `ControlAndLKI`,
`OrderingExpiry`, `BaselineController`, `DirectBaseline`, `BasicLandTargets`,
`WallDirections` and `AuraReattachment`. Complete records cover Abomination, Aisling Leprechaun,
Cockatrice, Thicket Basilisk, Infernal Medusa, Battering Ram and Venom: seven
unique names / 17 records. Wall restrictions, flying and the independent banding
rule remain covered. Compilation acceptance is unchanged.

## Printed spell-cast trigger text layers

`GenericTriggered.WithLayeredSpellCastTextRevision() *GenericTriggered` marks a
printed typed battlefield spell-cast trigger. It includes the existing checked
revision opt-in and declaration validation.
`Game.AddPrintedSpellCastTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` installs color or basic
land word changes in layer 3 with `EndOfTurn` or `Indefinite` duration.

RON marks the complete existing
`TriggerA(WhenAPlayerCastsASpell(AnyPlayer | Opponent, spellFilter), actions)`
context. Existing filters, bodies, targets and event scopes keep their checks.
No new trigger syntax is added. Complete object text uses
`Game.AddPrintedSpellCastObjectTextEffect`; mixed contexts stay separate.

Each pass restores the declaration baseline before timestamp-ordered changes.
A prior direct revision can supply the baseline. Expiry recomputes surviving
changes; direct revision rejects a current layer result. Semantic word changes
can affect event filters, body filters and targets; mana symbols retain their
values. Printed/copiable definitions and sibling declarations remain separate.
Installation probes a game clone before live registration. Missing source visits,
face-down/phased sources, foreign/granted/removed/unmarked declarations, missing
builders/groups, missing rebuilt markers, changed caster/capture contexts, invalid
changes, empty changes and unsupported durations reject without live mutation.

Detected triggers, placement targets, stack copies and optional payments retain
their detection-time rules and exact event bindings (CR 603.2–3). Layer 6 removal
suppresses detection; restoration uses retained text. Phasing keeps the visit,
including expiry before phase-in. Clones keep separate baselines, registrations,
cleanup and controller contexts. LKI keeps the final rule; return starts from
printed rules (CR 400.7). Shared text rebuilding, layers and timestamps implement
CR 612.1–3, 613.1c and 613.7. No cost, replacement or simultaneous-operation
contract changes. Whole-object text, spell text and copy rebinding remain separate.

Evidence: `TestRONTextSpellCastLayerRecords` executes all 30 charm records;
`RealCards`, `OtherRealCards` and `IdentityAndLifetime` cover 11 unique names
(36 records), including all six other records. `BaselineController`,
`BasicLandTargets`, `PendingTargetsAndClone`, `EventBindings` and
`IndependentDeclarations` cover baseline expiry, controllers, queued text,
clones, copies and payments. Native `TestTextSpellCastLayerDeclaration` and
`Rejects`, plus RON `Invalid`, cover declaration and atomic installation rejection.


## Printed static basic-land-type text layers

`StaticAbilityHolder.WithLayeredLandTypeTextRevision() *StaticAbilityHolder`
marks one printed global, fixed Aura-host or chosen Aura-host land-type rule.
`Game.AddPrintedStaticLandTypeTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` installs ordered layer-3
changes with `EndOfTurn` or `Indefinite` duration. The existing typed land-type
declaration validation remains in force. Missing markers, invalid words, empty
changes, other durations and changes of declaration context reject atomically.

RON opts in unconditional `EachPermanentLayerEffect(filter, [SetLandType(basic)])`,
Aura `PermanentLayerEffect(HostPermanent, [SetLandType(basic)])`, and Aura
`PermanentLayerEffect(HostPermanent, [SetLandTypeVariable(TheChosenLandType)])`
with a valid source entry-choice declaration. `basic` is Plains, Island, Swamp,
Mountain or Forest. Existing filter and scope checks remain. Mixed, temporary,
source-dependent global and other declarations do not opt in.

Each layer pass restores the saved baseline, then applies surviving text changes
in timestamp order. A prior direct revision can supply the baseline. Direct
revision rejects a declaration with a layered result. Printed land selection and
output words change; saved entry choices do not change or repeat (CR 612.2).
Layer 4 uses the shared land-type, dependency, attachment and intrinsic-mana
services (CR 305.7 and 613.8). Later lands and current hosts use live membership.
Layer-6 ability removal cannot undo the earlier type layer. Restoration uses the
registered rule. Phasing preserves the visit; expiry is recomputed on the next
present pass. Clones keep separate rules, baselines, cleanup and controller
contexts. Departure LKI retains final land types; returned sources and hosts
cannot inherit a former visit. IDs and registration positions remain stable.
This API changes one declaration. Whole-object text, spell text and copy-layer
rebinding still need separate services. Existing direct-revision callers remain.

Evidence: `TestRONTextStaticLandTypeLayerRecords` covers Blood Moon, Conversion,
Evil Presence and Phantasmal Terrain (4 names / 19 complete records), with ordered
changes, repeated passes, intrinsic mana, late entrants, attachment changes,
source and host phasing, expiry, suppression/restoration, clones, LKI and return.
`Dependencies` changes selection and output words in both source timestamp
orders. `PriorRevisionAndDeclarations` checks prior direct baselines, sibling IDs
and clone control. `Upkeep` retains Conversion's independent payment/sacrifice
rule under layered text. `Boundaries` retains invalid declarations and scope
checks. Native `TestTextStaticLandTypeLayerDeclaration`, `Rejects`, `Context`,
`RestoredCloneContext` and `RegistrationOrder` cover definition validation,
atomic rejection, all six cross-context changes, controller isolation and
registration order through installation and expiry. Existing RON direct tests
retain casting, legality, rollback and printed-ability suppression evidence.
No new payment, replacement ordering or simultaneous-operation contract is added.

## Targets of abilities from matching sources

`CantBeTargetedByAbilities(subject PermanentRef, filter CardFilter) StaticRule`
adds a live restriction through `Static` and the shared `EffectManager`.
Definitions require a bound permanent reference, a battlefield source and a
nonzero source filter. Source and Aura-host contexts use `This` and
`AttachedPermanent`. Reattachment changes the current subject. Departure,
phasing and removal of the granting ability suspend the rule; host ability
removal does not remove an Aura's rule. Returned hosts cannot supply the former
attachment. Clones retain independent registrations.

The shared `Permanent.CanBeTargetedBy` query applies the restriction to ability
sources at target selection and resolution. Spell queries retain their explicit
`SpellTargetSource` category. Nontargeted actions and direct attachments do not
use this restriction. Existing shroud, hexproof and protection checks still apply.
Illegal activation targets fail before costs are spent. If targets become
illegal after payment, costs stay paid. Legal members of a mixed target list
remain eligible (CR 115.1c–d, 608.2b).

Live ability sources supply current types and colors. Pending triggers, stack
resolution and retargeted copies use the source's saved incarnation, with current
or retained departure information (CR 113.7a, 608.2b). A returned card cannot
supply the former source's characteristics. Trigger target selection now uses
that same source view. Ability sources without a battlefield incarnation retain
their supplied card snapshot without following a later battlefield object.

RON supports `PermanentRuleEffect(ThisPermanent|HostPermanent,
[CantBeTheTargetOfAbilities(AbilityOfASource(filter))])`. Filters are
`IsCardtype(type)`, `IsColor(color)` for the five colors, and nonempty `And([…])`
or `Or([…])` compositions. A two-operation rule can combine this restriction
with `CantBeTheTargetOfSpells`. `CanTargetOnly`, unbound references, malformed
operands and unsupported filters remain compile errors. Conditional restriction
rules and target-declaration predicates need separate contracts.

Artifact Ward's complete ATQ record now executes all three rules. Raiding Party's
complete record now executes its static rule, optional tapping, saved tap counts,
repeated plural Plains choices and destruction exclusions. Local
`data/ATQ.json` and `data/FEM.json` confirm
these semantics. No RON source record changed.

## Land-subtype mana bonuses

`LandManaBonusThisTurn(filter PermanentFilter, color core.Color) ResolvingEffect`
installs an independent fixed bonus through `EffectManager`. Definition validation
requires a concrete mana type. The filter reads current lands under either
player, including nonbasic lands and later entries. Each matching tap adds one
mana to the activating player, regardless of the spell controller. Multiple
bonuses accumulate. Source departure, phasing and ability removal do not end the
resolved rule. Subject phasing prevents use; a returned land is a new current
member. Cleanup removes the registration (CR 514.2). Clones keep separate effect
registrations and cleanup. The effect uses the existing source-independent
lifetime contract and does not grant a removable ability to each land.

`NewControllerManaBonusAbility(filter PermanentFilter, bonusMana Color)
*ManaBonusAbility` supplies a fixed printed bonus to the matching land's current
controller. This recipient can differ from the activating player when the mana
ability permits another player to activate it. Existing constructors keep their
activating-player behavior. Checked printed revisions must retain this recipient
contract. Definition validation rejects nonconcrete, attached and produced-type
controller bonuses. Automatic payment uses controlled sources; bonus entries
retain recipient metadata for actual production.

`ManaSystem.ManaBonuses` includes these rules alongside printed fixed, attached
and produced-type bonuses. Planning, affordability, direct activation and actual
production use that same query. Only a tap-symbol mana activation that produces
positive mana receives the bonus (CR 106.12, 605.4a). Ordinary taps, zero
production and mana activations without a tap symbol do not qualify. A mixed
production receives one bonus per declaration, rather than one per mana unit.
Mana-type replacements change the original production; the fixed bonus keeps
its declared type. Failed cast payments use the existing transaction rollback.
These registrations change no damage or simultaneous-operation contract.

RON supports these complete contexts:

- `TriggerA(WhenAPermanentIsTappedForMana(IsLandType(type)),
  ActionList([PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),
  AddMana(fixed_type))]))` lowers to `NewControllerManaBonusAbility` and opts in checked
  printed mana-tap revision. Revision changes the land word, retains the active
  ability ID, and keeps mana symbols fixed. Existing clone, phasing,
  removal/restoration and returned-source contracts apply.
- `CreateTriggerUntil(WhenAPlayerTapsAPermanentForMana(AnyPlayer,
  IsLandType(type)), ActionList([AddMana(fixed_type)]), UntilEndOfTurn)` lowers
  to `LandManaBonusThisTurn`. The implicit recipient is the future activating
  player. It installs at resolution, without capturing the spell controller as
  that recipient. It supports exactly one fixed mana action.

Both forms require one known land subtype and one concrete mana unit. Other
players, durations, body shapes, amounts and unbound references reject during
compilation. RON owns spelling and shape checks; the engine owns production,
payment planning and lifetime. Printed declarations use the mana-event text
layer service below. General filtered mana events remain a separate capability.

Evidence: `TestRONSubtypeManaBonus*` maps complete Gauntlet of Might records in
LEA, LEB, 2ED, CED and CEI, plus High Tide in FEM. It covers casting, both
players, distinct controller/activator recipients, immediate timing,
current/nonbasic subtypes, payment planning,
stacking, cleanup, clones, printed revision identity and isolation,
phasing and ability restoration. `TestTemporaryManaBonus*` covers native
validation, mixed production, zero/non-tap production, membership, source
independence, cloning, replacement agreement and failed casting rollback.

## Printed mana-event text layers

`GenericTriggered.WithLayeredManaTapTextRevision() *GenericTriggered` and
`ManaBonusAbility.WithLayeredManaTapTextRevision() *ManaBonusAbility` mark the
complete declaration contexts of the direct mana-tap revision service.
`Game.AddPrintedManaTapTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` installs ordered layer-3
changes with EndOfTurn or Indefinite duration. It validates a cloned probe before
installation. Invalid changes, empty changes, unsupported durations, missing
builders/groups, foreign IDs, unmarked/granted declarations, missing markers in
fresh definitions, changed contexts and absent/phased/face-down sources reject
without mutation. Direct revision cannot change a current layered result.

Each layer pass restores a separate baseline for each declaration, then applies
active changes in timestamp order. A prior direct revision can supply that
baseline. Expiry recomputes surviving changes from it. IDs and declaration
positions remain stable. The printed card and copiable values stay unchanged.
Color words and basic land words change; mana symbols retain their values
(CR 612.1–3, 613.1c, 613.7). RON marks its existing complete subtype,
produced-type, Aura-host and typed non-mana land-event TriggerA paths.
No new filtered-event shape or temporary declaration context is accepted.

The existing mana bonus service supplies both planning and actual production.
Membership uses current land types. Produced-type bonuses use actual mana after
replacements. Controller-recipient bonuses retain the tapped permanent's
controller, while other bonuses retain their existing actor contract. Mana
bonuses resolve immediately; non-mana events retain stack timing (CR 605.1b,
605.4a, 605.5a). Queued triggers, target choices, copies and delays retain text
and event bindings from detection, even after expiry or source departure.

Clones keep separate baselines, registrations and controller contexts. Phasing
keeps the visit; expiry is reflected on the next present layer pass. Layer-6
removal suppresses the declaration; restoration uses its saved baseline and
active text effects. Departure LKI retains the final declaration. A returned
source uses printed rules and cannot accept the old visit (CR 400.7).
Whole-object changes, spell text and copy-layer rebinding remain separate gaps.
No cost transaction, replacement-order or simultaneous-operation rule changes.

Evidence: TestRONTextManaTapLayerRecords covers all 23 complete records for
Gauntlet of Might (5), Mana Flare (6), Wild Growth (6) and Manabarbs (6), plus
independent P/T, ordered changes, symbols, immediate/stack timing, expiry,
clones, phasing, removal/restoration and return. The QueuedSnapshots,
PaymentRollback, InstancesAndLiveTypes and ProducedReplacement tests cover
pending targets/copies, exact identities, source-relative recipients, nonbasic
and changed types, mana payment rollback and replacement/production agreement.
TestTextManaTapLayerDeclaration, Rejects and BaselineAndContext cover checked
native contexts, atomic rejection, missing markers, a prior direct baseline,
controller isolation, phased expiry, suppression/restoration and departure LKI.
Existing TestRONLandMana*, TestRONSubtypeManaBonus* and TestTextManaTapTrigger*
retain casting, malformed syntax, payment and event contract coverage.

## Printed mana-tap declaration text revision

`GenericTriggered.WithManaTapTextRevision() *GenericTriggered` opts in a typed
`LandTappedForMana` event on the battlefield. State triggers, other event types,
and extra source zones reject at definition validation. Existing typed event
validation checks bindings and filters. Targets retain normal stack timing;
untargeted mana bodies retain immediate execution (CR 605.1b, 605.4a, 605.5a).

`ManaBonusAbility.WithManaTapTextRevision() *ManaBonusAbility` opts in the shared
fixed, attached or produced-type bonus representation. Fixed colors must be
concrete. Attached bonuses cannot also match produced types or carry a global
filter. The zero global filter retains its all-permanents meaning. Existing
unmarked callers retain their behavior.

`Game.RevisePrintedManaTapTrigger(source *Permanent, abilityID uuid.UUID,
changes ...TextChange) error` checks the present face-up source incarnation,
printed declaration group, active marker and validated fresh definition before
mutation. Bonus revisions must retain attached/global and fixed/produced-type
contexts. The active ID and ability position remain. Each declaration keeps its
own ordered revision prototype; the printed card and copiable origin stay fixed.
Foreign, granted, phased, removed and returned declarations reject.

The trigger path reuses `revisePrintedTrigger` and its immutable event captures.
Queued bodies, targets, copies and delays keep prior text and facts. Mana bonuses
stay in `ManaSystem.ManaBonuses`, so current filters, actual production and payment
planning agree. Each revision uses separate ability and map storage. Clones and
sibling declarations remain independent. Phasing keeps revisions; ability removal
suspends them; restoration uses revised rules; return uses printed rules.

RON opts in its existing complete land-mana `TriggerA` contexts: AnyPlayer plus
IsCardtype(Land) non-mana bodies, the exact produced-type bonus, and the exact
Aura-host concrete bonus. Mana symbols are not color words (CR 612.2): Wild
Growth's ManaProduceG remains green after a Green-to-Blue word substitution.
The bounded subtype bonus forms above are also supported. Other filtered mana
events and mana bodies remain compiler errors.
Marked declarations use the printed mana-event text layer service above for
layer-3 order, lifetime and restoration. Spell text remains a separate gap.

Evidence: `TestTextManaTapTrigger*` and `TestRONTextManaTapTrigger*` cover checked
shapes, rejection without mutation, ordered changes, solver/production agreement,
failed payment queries, immediate and queued timing, event players, targets,
copies, clones, phasing, removal/restoration and returned sources. The real-record
mapping covers Mana Flare, Wild Growth and Manabarbs in all six core reprints
(three names / 18 records). Existing `TestRONLandMana*` retains full base-gameplay,
casting rollback, attachment, LKI, prevention and lifelink evidence. No new
replacement-order or simultaneous-operation service is established.

## Printed static basic-land-type text revision

`StaticAbilityHolder.WithLandTypeTextRevision() *StaticAbilityHolder` permits
checked revision through `Game.RevisePrintedStaticLandType(source *Permanent,
abilityID uuid.UUID, changes ...TextChange) error`. Definition validation requires
exactly one global `BecomesBasicLandsEffect`, fixed
`BecomesBasicLandAttachedEffect` with one valid basic land type, or
`BecomesChosenBasicLandTypeAttachedEffect` using the source's entry choice.
Empty, nil, opaque callback, duplicate, multi-type and mixed declarations reject.
Fresh revisions must retain the global, fixed-host or chosen-host context.
The zero-value permanent filter retains its normal all-permanents meaning.

RON opts in unconditional `EachPermanentLayerEffect(filter,
[SetLandType(Plains|Island|Swamp|Mountain|Forest)])` declarations. Existing filter
validation remains; source-dependent global selections, combined modifications
and temporary effects do not opt in. Aura scope also opts in the complete
`PermanentLayerEffect(HostPermanent, [SetLandType(basicType)])` and
`PermanentLayerEffect(HostPermanent, [SetLandTypeVariable(TheChosenLandType)])`
forms. The variable form requires a validated source entry-choice declaration.
Chosen types are saved choices, not printed land words (CR 612.2). Revision does
not change that choice or repeat the entry operation. Each rule reads the current
host through the shared attachment-incarnation check.

The shared printed-static revision service keeps the active declaration ID,
source incarnation, current controller and registration position. Semantic
builders change validated input and output land words in order. Each declaration
and clone keeps its own revision prototype. Names, mana cost and printed copy
information remain unchanged. Rebuilt effects use the existing layer-4 land
service and dependency detector, including selection changes that override
timestamps (CR 613.8). Current and later subjects use current membership.

Setting a basic land type removes old land types and printed abilities, supplies
intrinsic mana, and keeps granted abilities, other subtype families, card types
and supertypes (CR 305.6–305.8). Phasing suspends the rule and keeps its revision.
Layer-6 ability removal does not undo the earlier type layer. Restoration uses
the revised registration; departure information retains revised types. Returned
sources use printed rules and cannot accept an old visit. Granted, absent,
phased, face-down, unmarked and foreign declarations reject without mutation.
For layer-3 order, duration and original-text restoration, use
`AddPrintedStaticLandTypeTextEffect` on opted-in layered declarations.
Spell revisions remain a separate contract.

Evidence: `TestRONTextStaticLandTypeRecords` covers Conversion's six reprints and
Blood Moon (2 names / 7 records), with casting rollback, entry timing, revisions,
intrinsic mana, printed-ability suppression, clones, phasing, removal/restoration,
LKI and printed-origin return. `Dependencies` changes both selection and output
words in both timestamp orders. `Declarations` checks independent groups and
source control; `Upkeep` checks Conversion's separate payment/sacrifice rule.
`Boundaries` keeps unsupported contexts rejected. Native
`TestTextStaticLandTypeDeclaration`, `Rejects`, `RegistrationOrder` and
`RetainsGrantedRules` cover invalid declarations, failure atomicity, retained
positions and unrelated characteristics/abilities.

`TestRONTextAttachedLandTypeRecords` covers Evil Presence and Phantasmal Terrain
in all six core reprints (2 names / 12 records): casting legality and rollback,
entry timing, saved-choice exclusion, ordered revisions, mana, reattachment,
clones, source/host phasing, ability removal/restoration, LKI and printed return.
`Declarations` checks foreign IDs, sibling instances and controller changes.
`Boundaries` retains invalid syntax/scope checks and excludes mixed declarations.
Native `TestTextAttachedLandType*` covers typed declaration checks, all six
cross-context replacements without mutation, and retained registration order.
The typed attached effect keeps existing callers and land mechanics; no new
replacement or simultaneous-operation contract is established.


## Printed static animation text revision

`StaticAbilityHolder.WithAnimationTextRevision()` marks a printed declaration
for checked revision through `Game.RevisePrintedStaticAnimation(source, abilityID,
changes ...TextChange) error`. The declaration must contain exactly the ordered
layers from one `AnimateEach` rule: type (4), optional color (5), and base P/T
(7b), all with battlefield duration. Missing, duplicate, reordered, foreign or
mixed-rule effects fail definition validation.

RON opts in its complete `EachPermanentLayerEffect(filter, modifications)`
animation context: one `AddCardtype(Creature)`, one `SetPT(PT(integer, integer))`,
and optional `SetColor(SimpleColorList(colors))`. Existing static permanent
filter validation still applies. Other lists and sticky effects stay separate.
The semantic definition builder changes validated basic-land and Magic color
word roles; names, mana symbols, color indicators and copiable origin stay fixed.

The shared printed-static installer preserves declaration ID, registration
position, source visit and current controller, and installs fresh validated
rules. Ordered changes belong to each declaration. Each new layer pass selects
current subjects; all animation parts retain the same exact members within a
pass, including after layer-6 ability removal (CR 613.6). The existing animation
dependency detector, layer sorter and P/T sublayers remain in use.

Phasing suspends the rule and retains its revision. Ability restoration uses
revised rules. Clones and sibling declarations keep independent later changes.
Departure information retains revised characteristics. A returned source uses
printed rules and cannot accept an old visit. Granted, absent, phased,
face-down, unmarked and foreign declarations reject without mutation (CR 612.3).
Direct revision alone does not install a live text layer. The marked layer
service below supplies order, duration and baseline restoration. Spell revision
needs a separate contract.

Evidence: `TestRONTextStaticAnimationRecords` covers every Kormus Bell and Living
Lands reprint and Living Plane (3 names / 13 records), with paid casting,
rollback, entry timing, word changes, summoning sickness, clones, phasing,
removal/restoration, LKI and printed-origin return. `Bindings` covers independent
declarations, source-relative control and P/T timestamp order; `Boundaries`
keeps invalid and dependent contexts rejected. Native
`TestTextStaticAnimationDeclaration`, `Rejects` and `RegistrationOrder` check
complete rule identity, failure atomicity and retained registration positions.

## Printed static animation text layers

`StaticAbilityHolder.WithLayeredAnimationTextRevision() *StaticAbilityHolder`
marks a complete printed animation for layer 3. RON uses this marker for its
existing unconditional `EachPermanentLayerEffect(filter, modifications)` shape:
one `AddCardtype(Creature)`, one `SetPT(PT(integer, integer))`, and optional
`SetColor(SimpleColorList(colors))`. Static filter and declaration checks remain.

`Game.AddPrintedStaticAnimationTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` installs ordered color or
basic land word changes with `EndOfTurn` or `Indefinite` duration. Each pass
restores the saved baseline before applying surviving changes in layer 3.
A prior direct revision can supply the baseline. Direct revision rejects a
current layered result. Fresh replacements must retain the layer marker.
Invalid changes, durations, contexts and missing or foreign declarations reject
without state changes. Absent, phased, face-down and granted declarations reject.

The shared static text service retains declaration IDs, registration positions,
source incarnations and separate clone/controller contexts. The existing
animation service selects current subjects in layer 4 and keeps those exact
members through optional layer 5 and base P/T layer 7b, even after ability
removal in layer 6. Land-type dependencies and P/T timestamps remain active.
Phasing preserves the visit; expiry recomputes on the next present pass.
Departure information retains final results. Returned sources use printed rules.
Whole-object text, spell text, sticky animation and copy rebinding stay separate.
CR 612.1–3, 613.1c–g, 613.4b, 613.6–8 and 400.7 support this contract.

Evidence: `TestRONTextStaticAnimationLayerRecords` covers Kormus Bell (6),
Living Lands (6) and Living Plane (1), including repeated passes, color output,
late entrants, summoning sickness, phasing, removal/restoration, expiry, clones,
LKI and return. `LayerDependencies` checks Blood Moon in both source orders.
`LayerBindings` checks prior direct baselines, separate declarations, controller
changes and clone expiry. `LayerBoundaries` retains syntax and dependent-form
rejection. Native `TestTextStaticAnimationLayerDeclaration`, `LayerRejects`,
`LayerRestoredCloneContext` and `LayerRegistrationOrder` check complete typed
rules, failed replacement atomicity, marker retention and registration order.
Existing direct animation records retain paid casting, rollback and entry timing.

## Printed combat-pair trigger text revision

`GenericTriggered.WithCombatPairTextRevision() *GenericTriggered` opts in
`BlocksCreatureMatching`, `BecomesBlockedByCreatureMatching` or
`BlocksOrBecomesBlockedByCreatureMatching`. Definition validation requires the
matching event kind, a battlefield source, no extra zones, no state trigger,
and a checked source or attachment subject.

`Game.RevisePrintedCombatPairTrigger(source *Permanent, abilityID uuid.UUID,
changes ...TextChange) error` uses the shared checked printed-trigger installer.
It requires the exact present face-up source, an active unwrapped printed
ability, retained semantic builder and fresh declaration correspondence. The
replacement must keep its marker and event direction. Failure changes no runtime
ability or retention state. Success keeps the active ID and position with fresh
event filters, targets, bodies and per-instance opponent bindings.

RON marks the existing `TriggerA` combat-pair path: two-argument
`WhenACreatureBlocksACreature` and `WhenACreatureBecomesBlockedByACreature`,
or their matching two-member `Or` union. Subjects are
`SinglePermanent(ThisPermanent)` or `SinglePermanent(HostPermanent)`.
Existing creature, subtype, color and fixed toughness filters remain unchanged.
`TriggerI`, grouped combat events and temporary granted triggers are separate.
Color substitutions change validated filter and body word roles (CR 612.2).
This service does not add creature-subtype substitutions.

Detection and pending abilities keep the prior text, controller and exact
opponent capture (CR 603.2–3). Copies and future end-of-combat bodies retain
those bindings. A returned opponent cannot replace a saved incarnation.
Information reads use the existing source LKI services; they do not permit
movement of absent subjects. Phasing keeps revisions and suspends detection.
Ability removal suspends the declaration; restoration retrieves its revised
rules. Clones and sibling declarations retain separate subsequent changes.
A returned source uses printed rules. Granted abilities cannot qualify
(CR 612.3). Marked declarations use `Game.AddPrintedCombatPairTextEffect`
for layer-3 ordering, expiry and baseline restoration. Unmarked direct callers
must supply those services separately. Whole-object text remains separate.

Evidence: native `TestTextCombatPairTriggerDeclaration` and `Rejects`; RON
`TestRONTextCombatPairRecords`, `BodyAndLifetime`, `AuraAndRecords`,
`Boundaries`, `PendingTargets`, `PaidCastAndNativeBlocks` and `ControlAndLKI`.
Complete records supply bounded revision evidence for Abomination, Aisling
Leprechaun, Cockatrice, Thicket Basilisk, Infernal Medusa, Battering Ram and
Venom: seven unique names / 17 records. Existing combat-pair tests retain
full combat, regeneration and attachment evidence. Collection acceptance stays
unchanged. This mapping does not claim complete collection gameplay coverage.



## Printed intrinsic basic-landwalk text revision

`WithPrintedBasicLandwalk(keyword core.Attr) CardOption` adds a distinct
`PrintedBasicLandwalkAbility` and its intrinsic attribute seed. Definition
validation accepts only Plainswalk, Islandwalk, Swampwalk, Mountainwalk and
Forestwalk. RON uses this option only for top-level
`Landwalk(IsLandType(<basic land type>))`. Desert and legendary landwalk retain
their existing keyword representation. Nested grants use their existing adapters.

`Game.RevisePrintedBasicLandwalk(source *Permanent, abilityID uuid.UUID,
changes ...TextChange) error` requires the exact present face-up source visit,
an active unwrapped printed declaration, semantic builder and checked fresh
declaration correspondence. It preserves the active ID, position, source and
current controller. Each declaration retains its own ordered text prototype.
Only its intrinsic attribute contribution changes; duplicate declarations,
other base contributions and granted keywords remain separate (CR 612.2–3).
Failure leaves runtime declarations, attributes and retention unchanged.

The shared attribute, removal and combat services read the revised keyword
(CR 702.14c–e). Block declaration rechecks current evasion. Specific keyword
removal and complete ability removal suspend the contribution; restoration uses
the revised keyword and declaration. Per-incarnation state changes by copy, so
clones and sibling instances remain separate. Phasing keeps revisions but prevents
installation. LKI retains revised attributes; return uses the printed origin.
Printed card seeds and copiable definitions stay unchanged.

Evidence: native `TestTextLandwalkDeclaration` and `Rejects`; RON
`TestRONTextLandwalkRecords`, `Lifetime`, `DeclarationsAndGrants`, `Invalid`,
`CastAndRemoval` and `BlockDeclaration`. Complete accepted records give bounded
revision gameplay evidence for 10 unique names / 20 records. Three further
basic-landwalk records still reject on unrelated rules. No collection acceptance
gain. The caller still supplies layer-3 ordering, lifetime and original-text
restoration. Other intrinsic keywords, nonbasic landwalk and full live text
installation need separate contracts.


## Printed color-protection text layer

`ProtectionAbility.WithLayeredColorTextRevision() *ProtectionAbility` opts in
one printed, nonempty color-protection declaration without an Aura exception.
RON uses it for top-level `Protection(FromColor(Colors([...])))` rules.
`Game.AddPrintedProtectionTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` accepts validated color
changes with `Indefinite` or `EndOfTurn` duration. It checks the present face-up
incarnation, printed declaration, retained builder and replacement eligibility
on an isolated probe before registration. Failed installation leaves no effect.

The shared layer manager applies changes in layer 3 timestamp order (CR 612.1–3,
613.1c/f, 613.3). Each pass restores the declaration's saved baseline before
reapplication. Expiry therefore recomputes later changes from that baseline.
Earlier direct revisions can supply the baseline; direct revision of a currently
layered declaration is rejected. Each declaration keeps its UUID and position.
Printed copy information, sibling rules and granted protection stay separate.

Layer 6 can remove protection. Restoration uses the saved declaration and the
next layer pass reapplies its text effects. Phasing keeps the incarnation and
suspends application; the next present pass handles changes and expiry. Departure
LKI keeps the last result. A returned object uses printed rules (CR 400.7).
Clones keep independent registrations, baselines and controller contexts.
The existing protection service supplies targeting, damage prevention, blocking
and attachment checks (CR 702.16b–f). Printed protection has no Ward exception.

This API changes one declaration only. Whole-object and spell text consumers,
copy-layer rebinding, granted protection and Ward donor text require separate
contracts. It makes no payment or choice and changes no replacement ordering.

## Printed intrinsic color-protection text revision

`ProtectionAbility.WithColorTextRevision() *ProtectionAbility` opts in printed
color protection with a nonempty list of Magic colors and no Aura exception.
Definition validation rejects marked empty, colorless, non-color and exempt
protection. `Game.RevisePrintedProtection(source *Permanent, abilityID uuid.UUID,
changes ...TextChange) error` checks the exact present face-up source incarnation,
active printed declaration, retained semantic builder and fresh declaration-group
correspondence before mutation. A replacement must retain its marker and supported
protection contract. Failure leaves runtime declarations and retention unchanged.

RON marks only top-level `Protection(FromColor(Colors([...])))` declarations;
the list must be nonempty. Nested protection grants use existing adapters and
remain outside this contract.
The engine keeps the active UUID, position, source and controller with a fresh
protection predicate. Each declaration retains its own ordered semantic prototype.
It does not rewrite printed mana, colors, name or sibling declarations (CR 612.2).
Granted abilities cannot qualify (CR 612.3).

Shared targeting, attachment, prevention and blocking services read the revised
runtime ability (CR 702.16b–f). Queued spells recheck protection at resolution;
an illegal announcement does not spend mana. A Ward's exception cannot exempt
it from a separate printed protection ability (CR 702.16n). Shared simultaneous
damage and replacement ordering remain unchanged. The per-incarnation retention
map changes by copy: sibling instances, successive changes and cloned games stay
separate. Phasing keeps revisions but suspends source eligibility. Ability removal
suspends protection; restoration retrieves its revised declaration. Departure
saves revised protection in LKI, and return uses printed rules.

Evidence: native `TestTextProtectionDeclaration` and `Rejects`; RON
`TestRONTextProtectionRecords`, `Lifetime`, `IndependentDeclarations`, `Invalid`,
`CastTiming`, `AttachmentAndGrants` and `ColorSet`. Complete records supply
12 unique names / 22 records with bounded protection-revision evidence; this does
not establish new complete-card gameplay totals. No collection acceptance gain.
The caller still supplies layer-3 ordering, expiry and original-text restoration.
Other intrinsic keyword attributes and full live text consumers remain separate work.


## Printed opponent land-tap trigger text revision

`GenericTriggered.WithLandTapTextRevision() *GenericTriggered` opts in a
printed typed `OpponentLandBecomesTapped` declaration. Definition validation
requires `EvtTapped`, battlefield source, no extra zones and no state trigger.
`Game.RevisePrintedLandTapTrigger(source *Permanent, abilityID uuid.UUID,
changes ...TextChange) error` checks the exact present face-up incarnation,
active printed declaration, retained semantic builder and fresh correspondence
before mutation. The replacement must retain its marker and typed event contract.

RON opts in `TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(subtype),
ControlledByAPlayer(Opponent)])), body)` with the existing validated land subtype
and resolving body contexts. Either filter order works. Unbound event reads and
other tap-event forms retain their separate compiler contracts. No card-name
branch is used. The shared checked trigger installer preserves the active UUID
and declaration position, with fresh targets, effects and bindings per build.

CR 612.2 word roles and CR 612.3 printed/granted separation use retained RON
prototypes and printed group keys. Detection uses event-time subtype and control
(CR 603.2, 603.3a); pending and queued bodies keep earlier text and targets
(CR 603.3, 113.7a). Typed native land bindings keep exact incarnations through
copies and departure. Returned targets cannot replace saved targets. Phasing
suspends detection and keeps revision state. Ability restoration uses a retained
revised trigger on that battlefield incarnation. The shared installer also keeps
spell-cast revisions through removal and restoration. The retention map changes
by copy; game clones keep independent later revisions. Return uses printed rules.

Evidence: `TestRONTextLandTapTriggerRecords` covers complete Lifetap (six
reprints) and Lifeblood; `Lifetime`, `RemovalAndTargets` and `Boundaries` cover
identity, control, phasing, clones, restored rules, queued targets and rejection.
Native `TestTextLandTapTriggerDeclaration`, `Rejects` and `Bindings` cover
validation, rejection without mutation and per-event saved identities.
Existing `TestRONOpponentLandTap*` retains payment rollback, batch, entry-tapped
and subtype evidence. Collection acceptance does not change.
`Game.AddPrintedLandTapTextEffect` supplies layer order, expiry and baseline
restoration for marked declarations. Complete text consumers remain separate.


## Printed global ability and P/T grant text revision

`StaticAbilityHolder.WithGrantTextRevision() *StaticAbilityHolder` opts in a
printed static declaration with at least one layer-6 effect. Each effect must
use `WhileOnBattlefield`; additional effects can modify P/T in layer 7c.
Empty, nil, other-layer, set-P/T and other-duration declarations reject during
validation. `Game.RevisePrintedStaticGrant(source *Permanent, abilityID
uuid.UUID, changes ...TextChange) error` uses the shared checked static installer.
It requires the original present face-up source, an active printed declaration,
a retained semantic builder and complete registrations. The replacement must
keep its marker, effect count and each effect's layer, sublayer and duration.
Rejection leaves registrations and runtime declarations unchanged.

RON opts in unconditional `EachPermanentLayerEffect(filter, parts)` on the
basic static path when every part is fixed `AdjustPT(integer, integer)` or
`AddAbility(abilities)`, with at least one nonempty grant. Grants support validated
keyword symbols and fixed `Landwalk` filters. Existing global filters retain
control, exclusion, color, subtype and property validation. Conditional rules,
Aura rules, triggered/activated grants, BandsWithOthers and other modifications
use their separate contracts and do not acquire this marker.

Global grants use `WhileWithSharedSubjects` within layer 6 and through optional
layer 7c. Selection occurs
before the first modification, so granting an ability cannot remove a recipient
from another keyword grant or its later P/T part in that pass (CR 613.6). The printed registration checks
source identity before allowing a started shared rule to continue after later
ability removal. Each new pass reads current membership and controller; removal
that prevents the rule from starting suspends both parts. Existing dependency
and timestamp services retain their ordering (CR 613.7a, 613.8).

The installer retains active UUID and registration positions. Ordered changes,
sibling declarations, source instances and game clones remain separate. Phasing
suspends the grant; restoration after ability removal uses its revised holder.
Departure saves revised characteristics in LKI. Return uses printed origin and
cannot satisfy an earlier source reference. Granted recipient abilities remain
outside printed text revision (CR 612.3). The caller still supplies layer-3
ordering, expiry and original-text restoration. Full Magical Hack and Sleight
of Mind installation remains unfinished.

Evidence: `TestRONTextStaticGrantLords`, `Cards`, `Membership`, `Boundaries`,
`Records`, `CastTiming`, `MultipleKeywords` and `IndependentDeclarations`; native
`TestTextStaticGrantDeclaration`, `Rejects`, `RegistrationOrder` and
`CrossLayerContinuation`. Nine accepted unique cards / 24 records receive bounded
revision evidence. There is no new collection acceptance or global complete-card
coverage claim. Cost transactions, replacements and simultaneous movement use
unchanged services.

## Printed Aura ability-layer text revision

`StaticAbilityHolder.WithAbilityTextRevision() *StaticAbilityHolder` opts in a
printed static declaration with nonempty layer-6 effects and
`WhileOnBattlefield` duration. Definition validation rejects nil, empty, other
layers and other durations. `Game.RevisePrintedStaticAbility(source *Permanent,
abilityID uuid.UUID, changes ...TextChange) error` checks exact face-up source
incarnation, active printed declaration, retained semantic builder, declaration
correspondence and complete registrations before any mutation. Fresh effects
must keep the marker, layer, duration and count.

RON lowers this marker for unconditional
`PermanentLayerEffect(HostPermanent, [AddAbility(abilities)])` on the basic
`compileStaticLayer` path: validated keyword symbols, fixed Landwalk filters,
and Ward protection with `SinglePermanent(ThisPermanent)` as its exception.
The source must be an Aura. Mixed parts, global grants, conditional statics,
granted triggered/activated abilities and other layers do not opt in.

The shared static installer retains active UUID and registration positions
without changing timestamps (CR 613.7a). Each declaration keeps ordered changes
separately. Existing layer-6 grants read the current exact host, with phasing,
source-removal dependencies and current attachment timestamps. The Ward exception
exempts only its physical granting Aura, even when revised to white protection.
This implements semantic-word selection under CR 612.2 and grant execution under
CR 613.1f; it does not modify granted abilities on the recipient (CR 612.3).

Phasing suspends execution; restoration after ability removal keeps the revised
holder. Departure ends registrations and saves revised protection in LKI.
Return restores printed rules. Clone effect replacement and grant storage are
independent. The layer reset now allocates separate ability storage before
removing grants: shortening a shared slice must not permit later append to
replace a sibling game's granted ability.

Absent, phased, returned, face-down, unknown-ID, granted, removed, unmarked and
incomplete declarations reject without mutation. Invalid changes, builder errors
and changed contracts also reject. The caller supplies layer-3 order, expiry and
restoration of original text. Magical Hack and Sleight of Mind still reject their
unfinished full consumers. Payment, targeting, replacement choice and simultaneous
damage retain their shared services.

Evidence: RON `TestRONTextStaticAbilityWards` (five Wards, ordered changes, exact
exception, clones, phasing, reattachment, removal/restoration and returned source),
`Landwalk` (Burrowing/Fishliver Oil and defending-land phasing), `Keywords` (Fear,
Flight, Lance, Eternal Warrior, Consecrate Land), `Boundaries`,
`IndependentDeclarations` and `DamageAndLKI` (cast White Ward, resolution timing,
simultaneous prevention and departure snapshot). Native
`TestTextStaticAbilityDeclaration`, `Rejects`, `RegistrationOrder` and
`GrantCloneIsolation` cover definition contracts, transactional rejection,
multi-effect replacement and the shared grant-storage regression.


## Printed static P/T text revision

`StaticAbilityHolder.WithPTTextRevision() *StaticAbilityHolder` opts in a printed
static declaration whose nonempty effects all use layer 7c and
`WhileOnBattlefield`. `Game.RevisePrintedStaticPTAbility(source *Permanent,
abilityID uuid.UUID, changes ...TextChange) error` installs fresh checked effects
for the original present face-up source and active printed declaration.

RON opts in `EachPermanentLayerEffect(creatureFilter, [AdjustPT(integer,
integer)])` through `compileStaticLayer`. The existing validated creature filter
can include control, source exclusion, color, subtype and combat state. The
conditional source context specified below also opts in. Other parts, general
permanent filters, Aura hosts, other conditional statics and CDAs do not opt in. Invalid syntax and out-of-scope references retain their errors.

The service retains the active UUID and each effect's registration position.
It does not retimestamp a static rule when its text changes. Independent siblings
and source instances retain independent ordered changes. Current membership and
controller are read by the existing P/T effects on each layer pass. Source
phasing suspends execution. Temporary ability removal suspends the registration;
restoration uses the revised declaration. Departure ends registration; returning
uses printed rules. Game clones retain separate replacements and registrations.

Missing registrations, unknown IDs, absent/phased/returned/face-down sources,
granted or unmarked abilities, invalid changes, failed builders and changed
effect counts reject before mutation. Printed/copiable definitions stay separate.
The caller must coordinate layer-3 order and text-effect lifetime. This bounded
service does not establish complete Magical Hack or Sleight of Mind execution.


## Printed spell-cast trigger text revision

`(*mage.GenericTriggered).WithSpellCastTextRevision() *GenericTriggered` opts a
printed typed battlefield spell-cast trigger into checked runtime replacement.
Definition validation rejects other events, state triggers and additional or
non-battlefield source zones. RON lowers this option for
`TriggerA(WhenAPlayerCastsASpell(AnyPlayer | Opponent, spellFilter), actions)` on
the existing `compileTriggerRule` path. Existing spell filters and resolving
actions retain their validation and binding scopes.

`(*mage.Game).RevisePrintedSpellCastTrigger(source *Permanent,
abilityID uuid.UUID, changes ...TextChange) error` checks the source incarnation,
face-up presence, printed group/member, active opt-in and retained builder. It
validates fresh declarations before replacing one runtime trigger with the same
ability ID and current source/controller. Each trigger keeps its own ordered
changes. Fresh targets, choices, payments and event bindings remain independent.
Detected triggers, stacked abilities and copies retain their earlier rules,
captured controller, caster and spell. Event detection reads runtime abilities;
this context needs no static effect registration.

Phasing suspends detection and prevents revision; a phase cycle keeps an existing
revision. Departure preserves pending bodies; return restores the physical
card's printed origin. Removed, granted, unmarked, absent and incompatible
declarations reject without mutation. Clones keep separate runtime replacement
and resolution state. Printed/copiable definitions remain separate.

The caller must supply layer-3 order, lifetime and restoration after ability
removal. Other event kinds, state triggers, cross-zone triggers, static rules and
spell text remain separate capabilities. Magical Hack and Sleight of Mind still
reject. Invoke Prejudice still requires its unsupported shared-color filter.

Evidence: RON `TestRONTextSpellCastTriggerRealCards`, `OtherRealCards`,
`PendingTargetsAndClone`, `EventBindings`, `IdentityAndLifetime`,
`IndependentDeclarations`, `Invalid`; native `TestTextSpellCastTriggerDeclaration`
and `Rejects`.

## Printed activated-action text state

`mage.WithTextRevisionActivationState() ActionOption` opts a printed activated
`ActionDefinition` into checked runtime rebinding. Declaration validation rejects
this option on spell actions. RON lowers it on ordinary and modal `Activated`
and `ActivatedWithModifiers` actions produced by `compileActivatedRule`.
The separate conditional-mana compiler path is outside this contract.
RON now uses `WithLayeredTextRevisionActivationState` to include layer lifetime.

`(*mage.Game).RevisePrintedActivatedAbility(source *Permanent,
abilityID uuid.UUID, changes ...TextChange) error` checks the saved source
incarnation against the present face-up permanent, its retained definition,
declaration group and active printed action. It builds and validates fresh rules
before replacing that one runtime action. Removed, granted, unmarked, phased,
absent and returned sources/declarations reject without changing game state.

The replacement keeps the ability UUID, current source/controller, once-per-turn
flag, activation count and history lineage. Fresh targets, costs, choices and
result bindings belong to the rebuilt declaration. Queued original abilities and
copies retain their old bodies and frames. Repeated revisions use the ordered
changes of that action's prior revision. Other actions remain separate. History
uses a declaration lineage UUID plus source incarnation; clones keep separate
counts. Existing cleanup resets the rebuilt action's flags and history.

This direct service replaces one active action and rejects a declaration while
its layered result is present. It keeps the physical card's printed and copiable
definition. Other ability kinds and spell text have separate contracts.
Magical Hack and Sleight of Mind still reject as complete records.

Evidence: `TestRONTextActivationStateHistory`, `QueuedAndRepeated`,
`LimitsAndIdentity`, `HistoryQueueAndClone`, `Eligibility`,
`IndependentHistoriesAndCleanup`, `CopiedDefinitionAndResults`, `Modal`; native
`TestTextActivationStateDeclaration`,
`MissingSource` and `ContractMismatch`.

## Text definition ability correspondence

`mage.WithDefinitionAbilityGroup(group int, option CardOption) CardOption`
assigns a distinct nonnegative declaration group to the abilities added by one
option. Revision construction requires each ability to have exactly one group
and member. Definition validation rejects nil options/abilities, repeated or
negative groups, duplicate members and stale declaration keys. Existing callers
can still add ungrouped abilities, but such a definition cannot supply a revision.

`mage.BuildTextDefinitionRevision(card Card, changes ...TextChange)
(*TextDefinitionRevision, error)` uses the existing semantic text builder. The
revision's `Definition() Card` is an independent ownerless changed prototype.
`ReplacementAbility(id uuid.UUID) (Ability, bool)` maps an original definition's
ability to its fresh declaration. It checks exact group/member membership and
ability category; absent or incompatible correspondence returns an error.
Unknown IDs and nil revision lookups return no result. Saved target choices,
source/controller context and result bindings do not transfer. Successive
revisions map the immediately preceding definition. Copies retain separate key
storage; legacy `CloneFrom` clears provenance. Retained-color copy omissions
remove only the omitted member's key, without renumbering other declarations.

RON lowering assigns group zero to the combined count characteristic definition
and group `ruleIndex + 1` to each other compiled top-level rule option. Empty
options retain their group. This is independent of source locations and names.
The AST and lexical binding grammar retain their current validation.

This is preparation for active-rule installation. It does not assign the old
ability UUID to the replacement, migrate activation-history bindings, install a
live layer, update runtime abilities, or transfer spell text to battlefield entry.
Magical Hack and Sleight of Mind remain rejected in both target domains.

Evidence: `TestRONTextAbilityCorrespondence`, `QueuedActivation`,
`LandAndTrigger`, `CountDefinition`; `TestTextAbilityCorrespondenceContracts`,
`InvalidGroups`, `MembersAndCopy`, `CopyOmissionsAndTokens`.

## Semantic text definition prototypes

`mage.ReplaceColorWord(from, to core.Color) (TextChange, error)` and
`mage.ReplaceBasicLandWord(from, to core.BasicLandType) (TextChange, error)`
require two different legal words. Colorless, any-color, missing land types and
zero-value changes reject. `TextChange.ApplyColor` and `ApplyBasicLand` operate
only on words the caller has assigned to the corresponding semantic role.

`mage.BuildTextChangedDefinition(card Card, changes ...TextChange) (Card, error)`
builds an independent, ownerless prototype through the card's retained pure
`WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error))` callback.
The builder validates independent declarations and protected printed metadata.
It applies previously saved changes before new changes. It retains original
printed definition and copy information. `BaseCard.Copy` retains the immutable
text builder; `CloneFrom` clears it. No game object is modified by this API.

RON compilation retains a private AST. Definition transformation visits rules
through validated syntax roles and the type line's basic land subtypes. It
changes color filters, protection, explicit token colors, land filters, landwalk,
land-type operations and explicit token land subtypes where their role is known.
A matching word with an unknown role returns a positioned error. Card names,
named-card operands, token names, mana symbols and color indicators are excluded
(CR 612.2). Type-line substitution removes duplicate subtypes.

`compileTextSubstitution` lowers the two real two-argument
`ReplaceColorWordVariableWithNewColorWordVariable` and
`ReplaceLandTypeVariableWithNewLandTypeVariable` declarations using their
lexically bound first/second choice references. Wrong arity, identical references,
unbound references and cross-domain words reject. Mage's `ColorWordSubstitution`
and `BasicLandWordSubstitution` return `TextChangeRef`; missing or identical
resolved words cannot supply a change. Their definition edges expose both reads
for producer validation. They make no choices and retain no global result slot.

This is a printed-definition transformation prerequisite. It does not install
`CreateSpellEffect` or `CreatePermanentLayerEffect` text operations. Live layer-3
timestamps, active abilities, granted-ability exclusions (CR 612.3), copied
exceptions, exact object identity, spell-to-entry continuation and departure
restoration remain separate installation contracts. In particular, changed
subtype prototypes are not registered live text layers and do not pass the
existing current-characteristic copy-capture guard. Use retained printed plans
for prototype copy provenance; do not bypass that guard.

Evidence: `TestRONTextDefinitionGameplay`, `Ordering`, `Roles`,
`FiltersAndIsolation`, `SubstitutionDeclarations`, `SpellAndTrigger`,
`MetadataAndTypeSet`; `TestTextDefinitionInvalidWords`, `BuilderContracts`,
`TypedReads`. Real changed definitions cover Black Knight, Bog Wraith, Island,
Tropical Island, Bad Moon, Blue Elemental Blast and Lifetap. Black Ward has
complete-record AST transformation evidence. Magical Hack and Sleight of Mind
have their actual substitution declarations tested; their complete records
still reject. These tests add no complete-card unlocks for those two spells.

## Extra turns at resolution

Use `mage.ExtraTurnFor(player)` with a typed `mage.PlayerRef` to add one turn
after the current turn. It returns a resolving effect. An absent player gets no
turn. `mage.ExtraTurn()` keeps its existing controller behavior.

RON accepts `TakeAnExtraTurn` as a symbol in resolving action lists. The current
player scope supplies the recipient. `PlayerAction(player, TakeAnExtraTurn)`
and `PlayerActions(player, [TakeAnExtraTurn])` use their explicit player and
existing instance bindings. Spell, activation, trigger and delayed bodies use
the same adapter. Tuple spelling, costs, static/entry contexts and all-player
grants remain unsupported. Waiting queries declare typed source reads without
granting a turn. This syntax does not implement Time Vault's turn replacement.

After each completed turn, runners call `game.AdvanceToNextTurn()`. The shared
service selects the newest extra turn first, saves the next normal player
through nested extra turns, and increases the turn number. It does not execute
steps or cleanup. Clones retain separate pending turns and normal-order state.

## Ante deck declarations

RON supports `DeckConstruction(RemoveFromDeckIfNotPlayingForAnte)` once at the
top level of any card's rules. The operation must be a bare symbol. It is
pregame metadata and cannot occur in costs, actions, triggers or static branches.

The adapter uses `mage.WithAnteDeckRestriction()`. `mage.NewGame` excludes marked
cards from initial libraries before drawing; `mage.NewGameWithAnte` retains them.
Use `mage.PrepareCardsForAnte(cards, enabled)` for explicit deck and sideboard
preparation; it returns retained and excluded cards without changing the input.
There is no sideboard zone in the engine. This declaration supplies no ante
zone action or permission to bring a restricted card in from outside the game.


This checkout has the legacy facade in `pkg/mage/dsl/dsl.go`. It does not have
`pkg/mage/dsl/composable.go`. Proposed signatures in the migration plan are
not implemented APIs.

## Non-targeting permanent choice at resolution

`mage.HighestManaValue(subjects mage.PermanentSet) mage.PermanentSet` retains
all maximum-mana-value members, including ties, in collection order. It reads
current membership. X contributes zero on the battlefield; a face-down
permanent contributes zero. Use `mage.ChoosePermanent(player, subjects, binding)`
with `mage.BindPermanent` and `mage.From` to save a non-target choice at resolution.
No candidates clears the result. The binding keeps the exact incarnation.

RON supports `ChooseAPermanent(collection)` in resolution action sequences.
The argument uses the existing contextual permanent collection grammar,
including `APermanentWithTheHighestManaValue(collection)` and bare `IsUnblocked`.
`TheChosenPermanent` must be a bare reference after its producer in the same
sequence. Each producer has a distinct binding. A new choice changes the lexical
reference for its continuation. Neither abilities nor optional branches can
export their local choice to later outer instructions.

`PlayerAction(player, ChooseAPermanent(collection))` makes the selected player's
choice without exporting it. It cannot overwrite an outer `TheChosenPermanent`.
Player-keyed lookup with `ThePermanentChosenByPlayerThisWay` remains unsupported.
This adapter adds no announcement targets, entry choices, payments, prevention
body or exchange operation. Forcefield and Juxtapose still reject after their
supported choice operations.

## Optional operations before battlefield entry

`mage.MayEntryOperations(operations ...mage.AsEntersBattlefieldAbility)` returns
an entry ability. The entering permanent's controller chooses before battlefield
registration. Decline permits normal entry. Acceptance runs the children in
order; a child that rejects entry stops the sequence and uses the existing
failed-entry destination. Definition validation rejects empty bodies, nil
children and use on nonpermanent cards. The declaration stores no decision.

`mage.EnterTappedAsEnters()` supplies tapped entry inside this sequence.
Existing `ChooseBasicLandTypeAsEnters` and `ChooseOpponentAsEnters` abilities
can also be children. The sequence does not undo other required entry effects.

RON accepts `AsPermanentEnters(ThisPermanent, [MayActions([operations])])`.
The body must be nonempty. Supported children are bare `EntersTapped`, bare
`ChooseABasicLandType`, `ChooseAPlayer(Opponent)` and nested `MayActions` bodies
with these shapes. Each factory call creates fresh child abilities. Optional
choices do not declare `TheChosenPlayer` or `TheChosenLandType` for other rules.
Counters, payments, copy operations and replacement-context `MayActions` need
their own contracts and still reject. This adapter does not implement entry
copying for Clone, Copy Artifact or Vesuvan Doppelganger.

## Static spell targeting restrictions

`CantBeTargetedBySpells(PermanentRef, CardFilter) StaticRule` supplies a live
restriction on matching spells. Use it with `Static`. Supply `This` or
`AttachedPermanent` for source and Aura-host rules. Definition validation rejects
nil or unbound references and a zero filter. Spell filters use the spell's current
information view, including color changes. Ability sources remain separate.

For a direct pre-stack target query, pass `SpellTargetSource(card)` to
`Target.Possible` or `Permanent.CanBeTargetedBy`. The returned card view is
immutable and keeps the physical card identity. Engine casting, grouped/modal
choices, alternate-zone casting, copy retargeting, resolution, interactive spell
choices, AI spell choices and the test harness supply this context. Existing
ability queries retain their source contract.

RON supports `PermanentRuleEffect(subject, [CantBeTheTargetOfSpells(filter)])`
in the top-level static rule context. `subject` is
bare `ThisPermanent`, or bare `HostPermanent` on an Aura. `filter` reuses the
spell filter grammar: bare `AnySpell`, `IsEnchantmentType(Aura)`, `IsCardtype`,
`IsColor`, and nonempty nested `And`/`Or` lists. Wrong arity, constructor-form
references, unbound references and unsupported spell filters remain declaration
errors. One spell restriction can share its list with one other-Aura restriction,
as described below. Other combined lists and ability
targeting remain separate capabilities.

`CantBeTargetedBySpellsIf(PermanentRef, CardFilter, StateCondition) StaticRule`
adds a pure condition to the same rule. The condition is required and is checked
at each target query, with the source's current controller and incarnation.
Ability targets remain permitted. Use the existing `NotCondition` to invert it.

RON top-level static `If(condition, rules)`, `Unless(condition, rules)` and
`IfElse(condition, trueRules, falseRules)` accept nonempty lists of the existing
conditional modifications. Each branch keeps the existing same-layer limit.
A branch can instead contain exactly one
`PermanentRuleEffect(subject, [CantBeTheTargetOfSpells(filter)])`.
Existing conditional attack and untap bodies retain their support. Branches
cannot declare targets, costs, keywords as bare rules, triggers or resolving actions.
Both branches must compile, including the currently inactive branch.

`IsPlayersTurn(You)` reads the current controller's turn. The exact condition
`PermanentPassesFilter(ThisPermanent, Or([AttackedThisTurn, BlockedThisTurn]))`
reads the original source incarnation's turn history; the two operands can be
reversed. Host history, separate history predicates and other logical history
combinations remain unsupported. Static P/T counts also accept
`ControlledByAPlayer(Opponent)`, including within a nonempty `And` filter, through
the shared opponents and permanent-set services. The count reads current control,
types and phasing. It is not a saved resolution value or an all-zone CDA.
Angry Mob and Lurker use these forms with complete-record gameplay tests.

## Static Aura attachment restrictions

`CantBeEnchantedExceptBy(subject, exception PermanentRef) StaticRule` permits
only the specified Aura to enchant the subject. Both references must be valid
and bound. Use `Static` with `AttachedPermanent` and `This` for an Aura that
forbids other Auras. The rule follows the current attachment and its incarnation.
It checks Aura spell targets, direct attachment, optional hand entry, and existing
attachment legality. It does not restrict ability targets or Equipment.
Illegal attached Auras go to their owners' graveyards in one zone batch;
indestructibility and regeneration cannot prevent that state-based action.

RON supports `PermanentRuleEffect(subject,
[CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent),
IsEnchantmentType(Aura)]))])` at the top level. The two filter operands can be
reversed. The subject must be bare `ThisPermanent`, or bare `HostPermanent`
on an Aura. Exactly these two filter operands are supported. A list can combine
this operation with one `CantBeTheTargetOfSpells(filter)` in either order.
Unknown operations, duplicates, extra filter conditions, other exceptions,
conditional contexts and resolving durations remain rejected.
Anti-Magic Aura and Consecrate Land use these forms with complete-record tests.

## RON card loading

`pkg/mage/dsl/ron` parses the card records in `cards/mtgish/*.ron` and compiles
supported rules into the existing engine. It has these loading APIs:

```go
func Parse(reader io.Reader, filename string) ([]Node, error)
func Compile(card Node) (mage.CardFactory, error)
func LoadFS(files fs.FS, pattern string) (*mage.Registry, error)
func LoadInto(registry *mage.Registry, files fs.FS, pattern string) error
```

`LoadInto` replaces the entire destination registry only after every record
compiles. A failed load leaves its factories and name aliases unchanged. The
loader never calls an existing Go factory to substitute for unsupported RON.
Reprints with equal definitions share a factory. Conflicting definitions fail
the load. Every factory creates separate card, ability, target, and effect state.

This startup function compiles against the current API:

```go
package example

import (
    "os"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/dsl/ron"
)

func LoadCards(directory string) error {
    return ron.LoadInto(mage.DefaultRegistry, os.DirFS(directory), "*.ron")
}
```

The parser supports the syntax used by the supplied files: constructor names,
named fields, lists, tuples, strings, signed decimal integers, trailing commas,
and line and nested block comments. Records may span lines. Raw strings and
Rust Unicode string escapes are supported. This is a card-file parser, not a
general implementation of every RON extension.

The initial compiler supports card metadata, color indicators, simple keywords,
color protection, basic landwalk, tap/sacrifice/mana costs, single target
declarations, source damage to a bound target, batch destruction, drawing,
controller life changes, mana production, temporary boosts and keyword grants,
Aura boosts and grants, creature-filtered global boosts and grants, self entry
and death triggers, and controller upkeep triggers. It rejects unsupported
constructors, target-role mismatches, unbound references, and unknown fields.
It does not discard unsupported rules from otherwise supported cards.

The public adapters include `ActionList`, `Targeted`, `SpellActions`, `Activated`,
`TriggerA`, `Integer`, `PayMana`, `IsCardtype`, `IsCreatureType`,
`SpellDealsDamage`, `PermanentDealsDamage`, `DrawNumberCards`,
`DestroyPermanent`, `DestroyPermanentNoRegen`, `DestroyEachPermanent`, and
`DestroyEachPermanentNoRegen`. Their typed Go parameters use existing engine
values. The compiler validates the RON references before it calls these adapters.

Temporary RON layers use typed permanent references. In existing spell,
activation and trigger action lists, `CreatePermanentLayerEffectUntil/3`
accepts `ThisPermanent`, Aura `HostPermanent`, or the local
`Ref_TargetPermanent`, fixed `AdjustPT/2` and basic keyword `AddAbility/1`
modifications, and `UntilEndOfTurn`. One engine `Until` installs each action's
modifications together. Subjects are captured at resolution and retain their
incarnations until expiry. The compiler rejects empty modifications and invalid
or out-of-scope reads. Exact contracts and RON tests are in
[the temporary-layer slice](designs/mtgish-ron-temporary-layers.md).

In this end-of-turn context, `LosesAbility(keyword)` removes all instances of
one supported keyword. `Landwalk(IsLandType(symbol))` is accepted for both
`LosesAbility/1` and the existing `AddAbility([keyword, ...])` form. The supported
land symbols are Plains, Island, Swamp, Mountain, Forest and Desert.
`LosesAbility(AnyLandwalk)` uses the engine's complete landwalk removal service,
including legendary landwalk. These operations use layer 6 and timestamp order:
a later grant can restore an ability. Removal does not prohibit later grants.
Static removal, `AnyBandsWithOthers`, `TheChosenAbility`, non-keyword removal and
other durations remain unsupported. The tapped-source duration below also
accepts these modifications. Evidence is in
`TestRONTemporaryKeyword*` and `TestRONTemporaryLayersDeclareTypedReads`.

Temporary `CreatePermanentLayerEffectUntil/3` also accepts
`SetPower(Integer(n))` for any integer literal. It sets base power in layer 7b
through `SetBasePower` and preserves base toughness. Bonuses and counters still
apply in layer 7c. Existing typed subjects and durations apply. Dynamic values
and static `SetPower` remain unsupported. Island of Wak-Wak requires a flying
creature target; Singing Tree requires an attacking creature target. The engine
checks these restrictions at announcement and resolution. After resolution,
losing flying or leaving combat does not end the effect. Evidence:
`TestRONTemporaryPower*`.

Run the full collection check with:

```sh
go run ./cmd/loadcards -dir cards/mtgish
go run ./cmd/loadcards -dir cards/mtgish -json
```

Both commands return a nonzero exit status if any record fails. JSON diagnostics
include record counts and each failed card's location and first compilation
error. The current collection still has unsupported rules. Existing application
registration therefore remains in place until the complete collection compiles.
Remaining families include conditional costs and actions, other trigger forms,
replacement and prevention effects, modes, copy effects, zone movement,
permanent rule restrictions, dynamic values, and multiple result/target bindings.

## Damage attribution through retained callers

The existing engine services have these signatures:

```go
func (g *Game) GetArtifactDamageTaken(playerID uuid.UUID) int
func (g *Game) SetOnDamageDealt(callback func(sourceName, targetName string, amount int, isCombat bool))
func (ds *DamageSystem) CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int
```

These are engine services, not new sealed DSL expressions. Legacy resolving
damage effects use them through the shared damage system. Artifact history and
combat aggregation read the captured permanent incarnation's current information
or retained LKI. Callbacks report that source's name and the action's combat flag.
Copied abilities keep their original source attribution. History records damage
after prevention and redirection. History keys and event participants remain UUIDs.

This example compiles against the current engine and legacy DSL facade:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/dsl"
)

func ArtifactHistory(g *mage.Game, player uuid.UUID) int {
    return g.GetArtifactDamageTaken(player)
}

func DamageSource() dsl.Card {
    return dsl.NewCreature("Example artifact damage source", "", 1, 3,
        dsl.WithActivatedAbility(dsl.DealDamage(dsl.Fixed(2)),
            dsl.GenericCost(0), dsl.WithTarget(dsl.TargetDamageAnyTarget())))
}
```

`TestDamageAttributionSequence` makes a synthetic source an artifact, returns it
under another controller, and resolves its damage instruction. The history uses
the original source's artifact type. The example is a compiling declaration;
the test supplies the characteristic change and source departure.
No production cards or legacy exports are changed. Typed `DealDamageFrom`,
spell-source LKI, recipient identity, and immutable event captures remain open.

## Retained singular damage

```go
func DealDamage(amount ValueSource) Effect
func TargetDamageAnyTarget() Target
```

`dsl.DealDamage` retains its single-argument signature and positional target
behavior for production callers. The resolving effect can be used in a
targeted spell or a legacy activated or triggered ability with a declared
target. It uses the shared damage service. Damage to a planeswalker now removes
loyalty after prevention and replacement. A creature-planeswalker also receives
the existing creature damage result. Excess damage still counts in full for
damage events and lifelink. State-based actions handle zero loyalty.

This compiling declaration uses only the existing facade:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func DamageSpell() dsl.Card {
    return dsl.NewInstant("Example damage spell", "{R}",
        dsl.NewTargetedSpell(dsl.TargetDamageAnyTarget(),
            dsl.DealDamage(dsl.Fixed(3))),
    )
}
```

The test-only spell in `TestPlaneswalkerDamageSpellResolution` covers actual
announcement and resolution against a planeswalker. Engine tests cover
prevention, protection, replacement order, excess damage, and clone isolation.

`DealDamageFrom(source, recipient, amount)` remains proposed. Source LKI,
sealed expression roles, explicit recipient bindings, batches, and actual-damage
result bindings are not supplied by this change. No legacy export is retired.

## Lifelink from a departed permanent

The retained `DealDamage(amount ValueSource) Effect` also applies lifelink from
live permanent characteristics or retained battlefield LKI. This works in
legacy targeted activated abilities, including an ability whose source was
sacrificed to pay its cost. Player and permanent damage both use the shared
life-gain modifiers and replacements. Life-gain events report actual gain.

This compiling synthetic declaration exercises source departure during payment:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func LifelinkSource() dsl.Card {
    return dsl.NewCreature("Example lifelink source", "", 1, 1,
        dsl.WithKeyword(dsl.Lifelink),
        dsl.WithActivatedAbility(dsl.DealDamage(dsl.Fixed(2)),
            dsl.GenericCost(0),
            dsl.WithTarget(dsl.TargetDamageAnyTarget()),
            dsl.WithCost(dsl.SacrificeSourceCost())),
    )
}
```

No public signature changed. The later damage propagation prerequisite below
adds incarnation selection and retained LKI for captured permanent sources.
Spell-source LKI and `DealDamageFrom` remain absent. Production cards remain
unmigrated.

## Engine prerequisite: permanent incarnation references

Implemented in `mage`, without a new DSL export:

```go
func (g *Game) PermanentRef(id uuid.UUID) (PermanentRef, bool)
func (r PermanentRef) ID() uuid.UUID
func (g *Game) LookupPermanentRef(ref PermanentRef) LKIView
```

These engine queries capture a present battlefield incarnation and read its
current characteristics or retained LKI. They work outside resolution as well
as during engine actions. Capture rejects absent and phased-out permanents.
Existing references retain phase-time information while the object is absent;
phase-in resumes live reads without changing identity. Leave-and-return creates
a different reference. Departure snapshots survive cleanup. The zero reference
reads as absent. Reference values can be copied into instance-local storage and
used in cloned games; callers must retain the complete value, not only `ID()`.

This compiling engine example captures a source before its departure:

```go
package example

import "github.com/benprew/mage-go/pkg/mage"

func DepartedSourcePower(g *mage.Game, p *mage.Permanent) int {
    ref, ok := g.PermanentRef(p.ID())
    if !ok {
        return 0
    }
    g.BouncePermanentToHand(p)
    if view := g.LookupPermanentRef(ref); view != nil {
        return view.ViewPower()
    }
    return 0
}
```

This is an engine identity prerequisite, not a sealed author expression role.
The existing `LKIView` remains an engine reader with its existing read-only
ability convention. Returned departure snapshots have independent permanent
fields, but card and ability definitions follow existing shared-definition
ownership. Captured references retain snapshots for the game's lifetime; there
is no release API yet. Damage lifelink now consumes these references through
the action propagation described below. `DealDamageFrom` remains proposed.
No card wiring or gap status is complete.

## Engine prerequisite: ability source references

Implemented in `mage`; the legacy DSL signatures stay available:

```go
func (obj *StackObject) SourcePermanentRef() PermanentRef
func (g *Game) ResolvingSourcePermanentRef() PermanentRef
```

Activated abilities capture the battlefield source before payment. Ordinary
battlefield event and state triggers capture the source when they become
pending; leave triggers use the departing source. Pending-to-stack transfer,
ability copying, and game cloning preserve the complete reference. A copied
ability can have a different controller while retaining the original source.
This is identity capture, not an announcement-time characteristic snapshot:
`LookupPermanentRef` reads current characteristics until departure or phasing,
then retained LKI. Leave-and-return cannot substitute the new incarnation.

The resolution query works in engine effects and nested resolution; it restores
the previous reference on exit, including target failure. Outside resolution,
on spells, on delayed triggers, and on non-battlefield-source abilities it
returns zero. Manually constructed stack objects need engine capture to carry
an identity; a UUID alone does not supply one. Direct printed mana abilities do
not use the stack or this resolution query. The activation path for resolving
mana abilities does capture its source.

This complete example uses the retained facade and the new engine query:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/dsl"
    "github.com/google/uuid"
)

func AbilitySource() dsl.Card {
    effect := dsl.FuncEffect("Gain life equal to the source power",
        mage.EffectProperties{},
        func(g *mage.Game, source, controller uuid.UUID, targets []uuid.UUID) error {
            view := g.LookupPermanentRef(g.ResolvingSourcePermanentRef())
            if view != nil {
                g.PlayerGainLife(g.GetPlayer(controller), view.ViewPower())
            }
            return nil
        })
    return dsl.NewCreature("Example ability source", "", 2, 3,
        dsl.WithActivatedAbility(effect, dsl.GenericCost(0),
            dsl.WithCost(dsl.SacrificeSourceCost())))
}
```

`TestAbilitySourceRefSacrificeResolution` tests this pattern with a synthetic
card. Damage now carries the captured source as described below. Spell-source
identity, delayed-source capture, protection/deathtouch integration, typed roles,
and `DealDamageFrom` remain unimplemented. No card wiring or gap is complete.

## Engine prerequisite: damage action source propagation

Implemented in `mage`, with no new DSL author export:

```go
func (a *DamageToPlayerAction) SourcePermanentRef() PermanentRef
func (a *DamageToCreatureAction) SourcePermanentRef() PermanentRef
func (a *DamageToPlayerAction) RedirectToPermanent(id uuid.UUID) *DamageToCreatureAction
func (a *DamageToCreatureAction) RedirectToPlayer(id uuid.UUID) *DamageToPlayerAction
```

These queries and transformations are for engine damage replacements. Both
`WithAmount` methods preserve the captured source. The shared damage service
captures before replacements, preferring a matching resolving ability source
over the present incarnation. Built-in redirection preserves that identity.
Direct damage from a present permanent also captures its identity. UUID-only
action constructors keep their signatures and return zero references.

This complete engine example redirects a player damage action without losing
its original source:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/google/uuid"
)

func RedirectDamage(a *mage.DamageToPlayerAction, recipient uuid.UUID) *mage.DamageToCreatureAction {
    return a.RedirectToPermanent(recipient)
}
```

The retained `dsl.DealDamage(amount ValueSource) Effect` now uses the exact
permanent incarnation for lifelink in captured battlefield abilities, including
copies and triggers. Source departure/return, cleanup, and phasing retain the
original lifelink information. Current characteristics remain current until
departure. An explicitly different damage source does not inherit the resolving
ability's reference. Uncaptured absent sources keep legacy UUID lookup.

Protection, deathtouch, notification identity, history, and recipient lookup
retain their existing contracts. Spell/delayed-source identity, typed roles,
`DealDamageFrom`, simultaneous batches, and damage results remain unimplemented.
No production card is migrated and no inventory or gap is complete.

## Engine prerequisite: permanent source filters in prevention rules

The existing `WithFrom(from PermanentFilter)` option now uses captured permanent
source information in the shared damage-prevention rule. No signature changed.
The other relevant signatures remain:

```go
func (g *Game) AddDamagePreventionRule(opts ...damagePreventionRuleOption)
func (g *Game) PreventAllDamageFrom(sourceID uuid.UUID)
func WithFrom(from PermanentFilter) damagePreventionRuleOption
func WithToPlayer(playerID uuid.UUID) damagePreventionRuleOption
```

`damagePreventionRuleOption` is package-private; callers compose options without
naming its type. `AddDamagePreventionRule` installs a rule for the current effect
cycle. Use it during continuous-effect application, or when no intervening
cycle rebuild occurs before damage. `PreventAllDamageFrom` installs its existing
source-ID rule until end of turn. Neither function is a typed DSL declaration.

For captured damage, source filters read current characteristics while the same
permanent is present and retained characteristics after departure or phasing.
Captured spell sources and unknown battlefield incarnations do not match these
permanent filters. UUID-only actions keep live permanent lookup. The shared
lookup is internal; the public signatures above do not change.
Copies, redirections, and cleanup preserve that identity. Custom filters must
read the supplied permanent rather than perform another UUID lookup. Player and
permanent recipients retain their existing selection and legality contracts.
This complete engine example composes existing options:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func PreventRedDamageThisCycle(g *mage.Game, player uuid.UUID) {
    g.AddDamagePreventionRule(
        mage.WithFrom(mage.HasColorFilter(core.Red)),
        mage.WithToPlayer(player),
    )
}
```

At this earlier slice boundary, the separate color/type prevention executors
were unchanged. The contract below now extends their source reads. Protection,
deathtouch, spell-source identity, typed `DealDamageFrom`, and card migration
remain separate work. All production damage exports remain available.

## Engine prerequisite: color and card-type prevention source reads

Existing public signatures are unchanged:

```go
func (g *Game) AddColorPrevention(playerID uuid.UUID, color core.Color)
func (g *Game) AddTypePrevention(playerID uuid.UUID, ct core.CardType)
func AddColorPreventionStep(color core.Color) Effect
```

The two game methods install player-only, one-use prevention rules until end
of turn. They prevent the first matching damage event in full, then expire.
They can be called from resolving effects; the color step remains usable in
legacy sequences. They are engine installers, not sealed typed declarations.

Both rules read current color/type characteristics of the original permanent
incarnation, or retained departure/phase-time LKI. This includes gained and
lost characteristics, copied abilities, and source return followed by another
departure. Missing captured information rejects the match without using the
returned card. Nonmatching events do not consume the rule. Uncaptured live
permanents use current characteristics; uncaptured color reads reuse
`Game.EffectiveColors`, including spell color changes. Uncaptured card-type
reads retain the existing card lookup for nonpermanent sources. This does not
establish spell incarnation/LKI semantics.

This complete example compiles:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func PreventNextArtifactDamageThisTurn(g *mage.Game, player uuid.UUID) {
    g.AddTypePrevention(player, core.TypeArtifact)
}
```

These installers do not choose a damage source. They do not implement the
complete Circle of Protection requirements. Chosen-source identity, artifact
redirection source matching, protection, deathtouch, notifications, recipient
identity, typed damage adapters, and production card wiring remain open.

## Engine prerequisite: protection in the damage pipeline

Existing public signatures remain available:

```go
func ProtectionFromColor(c core.Color) *ProtectionAbility
func ProtectionFromColors(cs ...core.Color) *ProtectionAbility
func ProtectionFromCardType(ct core.CardType) *ProtectionAbility
func ProtectionFromSubType(subtype string) *ProtectionAbility
func ProtectionFromAll() *ProtectionAbility
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID)
```

Protection damage prevention now participates in recipient-chosen replacement
ordering and is checked again after redirection to a permanent. Permanent
sources use the action's captured incarnation: current characteristics while
present, retained departure or phase-time LKI while absent. A returned card
cannot replace the original source's characteristics. Color, card-type,
subtype, and everything protection reuse existing abilities, including wrapped
grants and Aura exemptions. Noncolor custom filters retain their CardFilter
contract; effective type and subtype queries use the supplied permanent view.
Uncaptured nonpermanent sources keep existing card and effective-color lookup;
spell incarnation/LKI is still open.

The same service supports legacy resolving damage effects, sequences, divided
damage, fight instructions, combat damage, and direct engine actions. The
existing `AttrDamageCantBePreventedOrRedirected` bypass now also bypasses
protection on direct permanent damage. This does not establish general CR
615.12 semantics. Targeting, blocking, and attachment checks remain separate.
No sealed typed declaration or `DealDamageFrom` adapter is introduced here.

This complete example compiles and uses nontargeted damage:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func DamageProtectedPermanent(g *mage.Game, controller mage.Player) {
    source := g.PutOnBattlefield(mage.NewCreature("Test red source", "{R}", 1, 3), controller.PlayerID())
    recipient := g.PutOnBattlefield(mage.NewCreature("Test protected recipient", "", 1, 3,
        mage.WithAbility(mage.ProtectionFromColor(core.Red))), controller.PlayerID())
    g.DealDamageToPermanent(recipient, 2, source.ID())
}
```

Production cards remain unmigrated. Deathtouch, notifications, spell and recipient
identity, batch damage, results, and typed adapters remain separate work.

## Engine prerequisite: deathtouch damage and destruction

Existing signatures are preserved:

```go
func DealDamage(amount ValueSource) Effect
func WithKeyword(kw core.Keyword) CardOption
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID)
func (g *Game) CheckStateBasedActions()
```

The damage service now keeps actual marked damage and recent deathtouch damage
as separate facts. Permanent sources use the captured incarnation's current
keywords or retained LKI, before notification callbacks. Returning the source
cannot replace it. Positive damage after replacements and prevention can cause
destruction at the next state-based check. That check uses the shared simultaneous
destruction service, including regeneration and indestructibility. Each check
consumes the fact, even when destruction cannot occur.

Supported contexts include existing resolving damage effects, sequences,
triggered and activated abilities (including copied abilities), fight and
combat damage, and direct engine actions. Target legality remains an
announcement/resolution requirement. A copy retains the original source card
for target checks. No sealed typed constructor or `DealDamageFrom` is introduced.

This complete engine example compiles:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func DealDeathtouchDamage(g *mage.Game, controller mage.Player) {
    source := g.PutOnBattlefield(mage.NewCreature("Test source", "", 1, 3,
        mage.WithKeyword(core.Deathtouch)), controller.PlayerID())
    recipient := g.PutOnBattlefield(mage.NewCreature("Test recipient", "", 1, 10), controller.PlayerID())
    g.DealDamageToPermanent(recipient, 1, source.ID())
    g.CheckStateBasedActions()
}
```

Spell-source identity/LKI, typed recipient bindings, damage batches and results,
source notifications, and production card migration remain open. The separate
legacy `BasiliskTouch` behavior is outside this contract.

## Engine prerequisite: artifact-source damage redirection

The existing public signature is unchanged:

```go
func (g *Game) SetArtifactDamageRedirect(controllerID, permID uuid.UUID)
```

This installer redirects artifact-source damage from the specified player to
an existing permanent for the current effect cycle. Use it in continuous-effect
application, or during resolution when no cycle rebuild precedes the damage.
It does not test whether the recipient is untapped; declarations must supply
that condition. An absent or phased recipient makes the rule inactive.

Source matching now reads the action's captured permanent incarnation, using
current types while present and retained departure/phase-time types while absent.
A later incarnation cannot supply those types. Missing captured information
rejects the match. Uncaptured live sources use current types; other uncaptured
sources keep the existing card lookup, without a spell-identity guarantee.
Replacement ordering, amount, combat flag, and original source reference use
the shared damage pipeline. This complete engine example compiles:

```go
package example

import "github.com/benprew/mage-go/pkg/mage"

func RedirectArtifactDamageThisCycle(g *mage.Game, player mage.Player, recipient *mage.Permanent) {
    g.SetArtifactDamageRedirect(player.PlayerID(), recipient.ID())
}
```

There is no new DSL export. Martyrs of Korlis still requires a typed replacement
and untapped condition. All production callers remain available. Notification
identity, spell/recipient identity, typed singular damage, batches, results, and
card migration remain separate work.

## Engine prerequisite: immutable damage-event source colors

Existing DSL signatures remain available:

```go
func (c EventSourceWasColor) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func (c EventSourceWasNotColor) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
```

For permanent-source damage, both predicates use colors captured from the damage
action's original incarnation before life and damage notifications. Current
colors apply while present; departure or phase-time LKI applies while absent.
Later changes, return, cleanup, event value copies, and game clones cannot change
this capture. Missing captured information satisfies neither predicate. Other
events and uncaptured nonpermanent sources retain legacy UUID lookup behavior.
These predicates remain usable with legacy triggered abilities; they are not
sealed composable declarations.

`core.GameEvent.DamageSourceColors` is an immutable `core.EventSourceColors`
value. Its engine inspection methods are `Captured() bool`, `Known() bool`,
`HasColor(Color) bool`, `SourceID() uuid.UUID`, and `Incarnation() uuid.UUID`.
`core.NewEventSourceColors(sourceID, incarnation uuid.UUID, colors []Color,
known bool) EventSourceColors` copies the color input into the value. A zero
value means there is no capture; a captured unknown source is distinct from a
known source with no colors. Damage executors produce this value, after
replacement processing and before result notifications. It is not a resolution
selector or a damage source declaration.

This complete example compiles:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func BlackDamageWatcher() mage.Card {
    condition := mage.EventSourceWasColor{Color: core.Black}
    return mage.NewEnchantment("Test black damage watcher", "",
        mage.WithAbility(mage.NewTriggered(core.EvtDamageDealt, false,
            mage.GainLife(1)).SetCondition(condition.CheckTriggerCond)))
}
```

Controller/type/subtype/filter and self-identity event predicates, spell LKI,
recipient identity, result bindings, typed damage, and production card migration
remain separate work. All production exports and callers remain available.

## Permanent damage-source controller capture

Implemented engine inspection API in `pkg/mage/core`:

```go
NewEventSourceController(sourceID, incarnation, controller uuid.UUID, known bool) EventSourceController
(EventSourceController).Captured() bool
(EventSourceController).Known() bool
(EventSourceController).SourceID() uuid.UUID
(EventSourceController).Incarnation() uuid.UUID
(EventSourceController).ControllerID() uuid.UUID
```

`GameEvent.DamageSourceController` contains this immutable value on damage
from a captured permanent. Both player and permanent damage paths save it
before life or damage notifications. Current information applies to the original
incarnation; retained departure or phase-time information applies while absent.
Returning the card cannot replace the controller. Unknown captured information
never falls back to UUID lookup. A source without a controller matches neither
controller predicate; its owner is not a substitute.

The existing engine conditions `EventSourceControlledByController{}` and
`EventSourceControlledByOpponent{}` retain their
`CheckTriggerCond(*GameEvent, GameReader, uuid.UUID, uuid.UUID) bool` signatures.
They use the saved controller for captured damage events. Other events and
uncaptured nonpermanent sources retain legacy lookup. These are event conditions,
not pure state predicates. The facade does not export these conditions here.
The scalar inspection API is not a sealed author expression or typed damage
adapter; type/subtype, self matching, and spell participants remain separate work.

This complete synthetic definition compiles with the current engine API:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func DamageControllerWatcher() mage.Card {
    condition := mage.EventSourceControlledByController{}
    ability := mage.NewTriggered(core.EvtDamageDealt, false, mage.GainLife(1)).
        SetCondition(condition.CheckTriggerCond)
    return mage.NewEnchantment("Test damage controller watcher", "", mage.WithAbility(ability))
}
```

This demonstrates only the controller condition. A creature/combat/opponent
restriction and optional draw require additional declarations for Coastal Piracy.

## Permanent damage-source type capture

Implemented engine inspection API in `pkg/mage/core`:

```go
NewEventSourceTypes(sourceID, incarnation uuid.UUID, types []CardType, subtypes []string, known bool) EventSourceTypes
(EventSourceTypes).Captured() bool
(EventSourceTypes).Known() bool
(EventSourceTypes).SourceID() uuid.UUID
(EventSourceTypes).Incarnation() uuid.UUID
(EventSourceTypes).HasType(CardType) bool
(EventSourceTypes).HasSubType(string) bool
```

`GameEvent.DamageSourceTypes` stores immutable copies of effective types and
subtypes from the original permanent incarnation. Both damage recipient paths
capture them before life or damage notifications. Current information applies
while the original incarnation is present; retained LKI applies during absence,
including phasing. Cleanup and a returned incarnation cannot replace the value.
Unknown captured information matches no type or subtype.

The existing `mage.EventSourceHasType`, `mage.EventSourceHasSubType`, and
`mage.EventSourceWasOfType` conditions retain their
`CheckTriggerCond(*GameEvent, GameReader, uuid.UUID, uuid.UUID) bool` signatures.
They use the capture for permanent-source damage; other events and uncaptured
nonpermanent sources retain existing lookup. This does not implement a general
permanent filter, spell-source capture, sealed expression, or typed damage
adapter. Legacy DSL exports remain compatible.

This complete synthetic definition compiles:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func CreatureDamageWatcher() mage.Card {
    condition := mage.EventSourceWasOfType{Type: core.TypeCreature}
    ability := mage.NewTriggered(core.EvtDamageDealt, false, mage.GainLife(1)).
        SetCondition(condition.CheckTriggerCond)
    return mage.NewEnchantment("Test creature damage watcher", "", mage.WithAbility(ability))
}
```

This example tests only source type. Coastal Piracy still needs controller,
combat, opponent, optional draw, and typed event declarations.

## Retained self-source damage conditions

These existing engine declarations keep their signatures:

```go
func (EventSourceIsSelf) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func (EventSourceIsSelfDamageToPlayer) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func (EventSourceIsSelfDamageToOpponent) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func DealsDamageToOpponentTrigger(effect Effect, optional bool) *GenericTriggered
```

For damage events with a permanent-source capture, self matching requires a
present source with the captured battlefield incarnation. Departure, returning,
phasing out, and missing captured information do not match. Phasing in keeps the
same identity. Copies preserve the source, and clones use their own battlefield.
Already pending triggers continue after source departure. Uncaptured events and
non-damage events retain UUID matching. These are retained trigger-time engine
conditions, not sealed typed expressions or resolution-time predicates.

This complete example compiles with the retained engine API:

```go
package example

import "github.com/benprew/mage-go/pkg/mage"

func SelfDamageSource() mage.Card {
    return mage.NewCreature("Example self damage source", "", 1, 4,
        mage.WithAbility(mage.DealsDamageToOpponentTrigger(mage.GainLife(1), false)),
        mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(2)),
            mage.GenericCost(0), mage.WithTarget(mage.TargetDamageAnyTarget())))
}
```

The ability deals damage through the shared damage service. If the source leaves
and returns before the activation resolves, its new trigger does not match the
old source's damage. Typed `DealDamageFrom`, spell and recipient identity,
full permanent-filter event captures, batches, and results remain proposed work.

## Damage-event permanent-filter identity

The retained engine condition has this signature:

```go
func (c EventSourceMatchesPermanentFilter) CheckTriggerCond(
    evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID,
) bool
```

For captured permanent damage, these existing filters use immutable event
information: `AnyPermanent`, the zero filter, `IsCreature`, `IsArtifact`,
`IsEnchantment`, `IsLand`, `IsPlaneswalker`, `HasSubType(string)`,
`HasColorFilter(core.Color)`, `ControlledBy(uuid.UUID)`,
`NotControlledBy(uuid.UUID)`, `IsID(uuid.UUID)`, `NotID(uuid.UUID)`,
`Named(string)`, `OwnedBy(uuid.UUID)`, `NotOwnedBy(uuid.UUID)`, and `IsToken`.
`Not(PermanentFilter)`, `And(...PermanentFilter)`, and `Or(...PermanentFilter)`
use captures when every leaf supports them. `And` and `Or` copy their inputs.
Their ordinary `Match(*Permanent, *Game)` behavior remains a current-state query.

Captures describe the original incarnation at damage execution, before life or
damage notifications. Later characteristic changes, departure, phasing, return,
cleanup, and cloned-game changes cannot change the match. Missing or inconsistent
required captures match nothing, including under negation or disjunction.
Controller filters require a known nonzero controller. ID filters compare the
saved UUID; they do not implement self-incarnation matching.

Filters outside this list, including custom filters, power, attachment, combat,
keyword and supertype filters, retain identity-safe lookup.
If a composition includes such a filter, the entire composition uses this
existing contract: current information for the same present incarnation or
retained LKI for departure/phasing, with the supplied game for state queries.
Uncaptured events and other event types keep current UUID lookup.

This is a retained engine condition; this checkout's `dsl` facade does not
export it. A complete compiling engine example is:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func Watcher() mage.Card {
    condition := mage.EventSourceMatchesPermanentFilter{
        Filter: mage.And(mage.IsCreature, mage.HasColorFilter(core.Green)),
    }
    return mage.NewEnchantment("Example damage filter watcher", "",
        mage.WithAbility(mage.NewTriggered(core.EvtDamageDealt, false,
            mage.GainLife(1)).SetCondition(condition.CheckTriggerCond)))
}
```

This establishes captured characteristic matching for the listed filters only.
Full permanent/game snapshots and exact typed declarations remain separate work.
Existing production callers stay compatible; no typed damage adapter is added.


## Damage-event source name, owner, and token status

Both shared damage executors save `core.GameEvent.DamageSourceIdentity` before
life and damage notifications. It contains an immutable `core.EventSourceIdentity`:

```go
func NewEventSourceIdentity(sourceID, incarnation, owner uuid.UUID,
    name string, token, known bool) EventSourceIdentity
func (s EventSourceIdentity) Captured() bool
func (s EventSourceIdentity) Known() bool
func (s EventSourceIdentity) SourceID() uuid.UUID
func (s EventSourceIdentity) Incarnation() uuid.UUID
func (s EventSourceIdentity) OwnerID() uuid.UUID
func (s EventSourceIdentity) Name() string
func (s EventSourceIdentity) IsToken() bool
```

`Named(name string) PermanentFilter`, `OwnedBy(playerID uuid.UUID) PermanentFilter`,
`NotOwnedBy(playerID uuid.UUID) PermanentFilter`, and the `IsToken PermanentFilter`
value now support captured damage matching through `EventSourceMatchesPermanentFilter`.
They work in triggered conditions and supported `And`/`Or`/`Not` compositions.
Ordinary permanent and prevention filtering retain their current/LKI contracts.
Captures use the existing live/LKI owner query, independently of ability control;
that service falls back to the permanent controller for ownerless engine fixtures.
A captured zero owner is unknown for owner filters. Empty names and nontoken status
remain known values. Missing, unknown, or mismatched captures never match, including
through negation. Uncaptured and non-damage events retain current lookup.

This complete engine example compiles:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func IdentityWatcher(owner uuid.UUID) mage.Card {
    condition := mage.EventSourceMatchesPermanentFilter{
        Filter: mage.And(mage.Named("Example source"), mage.OwnedBy(owner),
            mage.Not(mage.IsToken)),
    }
    return mage.NewEnchantment("Example identity watcher", "",
        mage.WithAbility(mage.NewTriggered(core.EvtDamageDealt, false,
            mage.GainLife(1)).SetCondition(condition.CheckTriggerCond)))
}
```

No production caller changes. Spell-source identity, recipient identity, typed
`DealDamageFrom`, other intrinsic filters, batches, and results remain open.

## RON single spell targets

`Targeted([TargetSpell(filter)], ActionList([CounterSpell(Ref_TargetSpell)]))`
uses a typed declaration local to the rule. Each factory creates a fresh target
binding. Supported filters are `AnySpell`, `IsCardtype(type)`, `IsColor(color)`,
`IsEnchantmentType(Aura)`, and recursive nonempty `And([filters])` and
`Or([filters])`. Filters match spell cards on the stack and exclude abilities. The engine checks legality at both
announcement and resolution. Countering uses the selected stack object and puts
its card in its owner's graveyard, unless it cannot be countered. Copies and
clones keep separate target choices. Undeclared, incompatible, malformed and
cross-rule reads fail compilation.

`TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))` is supported alone
or once in a top-level `And([filters])` for `TargetSpell`. Other conjuncts can
use recursive intrinsic spell filters. It uses
`mage.DeclareSpellTargetTargetingControlledPermanent` with the same per-factory
binding as `Ref_TargetSpell`. The selected spell must currently target a
permanent you control. A target that gained shroud can still qualify; absent,
returned and phased-out targets cannot. Checks run at announcement and
resolution. Copies use their own controllers and stack identities. This form
works in existing spell, mode, activation and trigger target declarations.

Other permanent predicates, players, duplicate target-history predicates and
history predicates nested inside logical operators remain compile errors.
Other enchantment subtypes, controller and mana-value spell filters remain
unsupported. No new modes or multiple spell target groups are added.
Contract: [controlled permanent spell targets](engine-rules-contracts.md#spell-targets-that-target-a-controlled-permanent).
Tests: `TestRONControlledPermanentCounter*`.

Contract and card tests: [RON spell scopes](designs/mtgish-ron-spell-scopes.md).

### RON controlled static creature layers

In `EachPermanentLayerEffect`, use
`ControlledByAPlayer(SinglePlayer(You))` inside an `And` filter
for fixed P/T changes or basic keyword grants. Nested nonempty conjunctions are
supported. The engine reads the source's current controller each time it applies
the layers. Matching sources are included. Current membership changes
with control, tap status, combat status, color and later entries. Controller
predicates inside `Or` and other controller reference forms remain rejected.
This adapter does not add controller support to target or collection filters.
See [the supported contract and tests](designs/mtgish-ron-static-controller.md).

### RON static source exclusion

In an `EachPermanentLayerEffect` conjunction, use
`Other(ThisPermanent)` to exclude the source from fixed P/T changes and basic
keyword grants. It can be combined with `ControlledByAPlayer(SinglePlayer(You))`
and nested nonempty `And` filters. Each source excludes only itself; other
instances can affect it. Source-dependent predicates below `Or`, other reference
operands remain rejected in this static context.
See [the contract and card tests](designs/mtgish-ron-static-other.md).

### RON captured temporary P/T values

`CreatePermanentLayerEffectUntil` supports `AdjustPTX(power, toughness, value)`
with `PlusX`, `MinusX`, `Zero` or `Integer(n)` coordinates. Values can be bound
`ValueX`, `Integer(n)`, or `ManaValueOfPermanent` of the modified permanent.
In that reader, `ActionPermanent` names the modification's subject. The binding
is local to the modification. Numeric values are captured at resolution and
remain fixed until end of turn. Other-object characteristic reads, power and
toughness readers, arithmetic and static AdjustPTXY remain unsupported.
See [the exact contexts and tests](designs/mtgish-ron-temporary-values.md).

## RON static landwalk grants

`AddAbility([Landwalk(IsLandType(subtype))])` is supported inside the existing
`PermanentLayerEffect(HostPermanent, modifications)` Aura context and
`EachPermanentLayerEffect(filter, modifications)` context.
The subtype must be a symbol: Plains, Island, Swamp, Mountain, Forest or Desert.
These grants use the same validated landwalk form as printed RON keywords.
They can be mixed with existing fixed P/T modifications and basic keywords.
Global source exclusion and controller conjunctions retain their existing
contracts. Each factory creates fresh continuous effects.

General permanent subjects are also supported as described below. Other
landwalk filters, temporary grants and rule restrictions remain separate
compiler contexts. See
[the static-landwalk slice](designs/mtgish-ron-static-landwalk.md) for evidence.

## RON general static permanent subjects

`EachPermanentLayerEffect(filter, modifications)` accepts the existing permanent
filters without an added creature restriction for fixed `AdjustPT(power,
toughness)` and basic keyword or supported landwalk `AddAbility(list)` grants.
`IsCreatureType(Goblin)` selects Goblin permanents; add `IsCardtype(Creature)`
only when the declaration requires creatures. Keywords can apply to
noncreatures. A P/T modification can apply when its recipient becomes a creature.

Nonempty recursive `And` and `Or` filters retain their logical membership.
`Other(ThisPermanent)` and `ControlledByAPlayer(SinglePlayer(You))` work at the
root or inside conjunctions. Source-dependent predicates within a disjunction
remain rejected. Each factory builds a fresh typed permanent set and static rule.
The engine reads current membership, controller, subtype, type and phase status
on each layer pass. No target or action-result binding is introduced.

General sets lower to `MatchingPermanents`, `ControlledPermanents`,
`WithoutPermanent`, `ModifyEachPT`, `GrantEachKeyword`, `While` and `Static`.
Existing explicit-creature and Aura adapters retain their services. Static
numeric P/T, independent base P/T/color/type changes and granted activated
abilities remain separate unsupported shapes. Controller-upkeep trigger grants use the
restricted contract under "RON granted controller-upkeep triggers" below.
Conditional rules have the
restricted contract below.
See [the exact contract and tests](designs/mtgish-ron-static-permanents.md).

## RON optional fixed mana payments

`AnyPlayerMayCost(Opponent, PayMana([...]))` and
`AnyPlayerMayCost(AnyPlayer, PayMana([...]))` accept fixed mana in ordinary
spell, activation, trigger and delayed action sequences. Pair the offer with
an adjacent `If` or `Unless` and a nonempty action list, or `IfElse` with two
nonempty lists. The condition must be symbol `CostWasPaid` or `CostWasntPaid`.
Both polarities use one local result. `PayAnyOrElse` offers eligible players in
active-player order and stops at the first complete payment. No successful
payment selects the unpaid path, including an empty eligible set. Both paths
keep the original effect controller. Each factory builds fresh costs and bodies.

On the paid path, `ActionPlayer` reads the successful payer through
`PayAnyOrElseWithPayer`. This applies to `If(CostWasPaid, ...)`,
`Unless(CostWasntPaid, ...)`, and the paid branch of either `IfElse` polarity.
The unpaid path cannot read or inherit `ActionPlayer`. The payer stays local
to this pair; subsequent actions retain their enclosing scope. Explicit
`PlayerAction(ActionPlayer, ...)` selects the payer. Ordinary player actions
keep their existing player context, and `You` keeps the effect controller.
A delayed body created on the paid path saves the payer in its own frame.
Nested offers have separate bindings and restore the enclosing paid scope.
Each factory builds a fresh binding. Costs and eligibility keep their existing
fixed-mana forms; the native API also supplies the candidate during payment.
Other player collections, X, nonmana/alternative costs, entry operations and
draw replacement bodies remain rejected. This is a shared prerequisite for
Scarwood Bandits, Worms of the Earth and Cleansing, not full-card coverage.
Scarwood has complete source-presence coverage; Worms needs entry prevention and
alternative payments; Cleansing needs repeated offers with per-subject results.
`TestRONEligiblePayerScope*` covers paid-path scope, delays, nested bindings,
active-player order, instance isolation and invalid unpaid reads.
`TestRONEligiblePlayerPayment*` covers lowering, gameplay, all polarities,
contexts, identity, copies, clones, inventory roles and atomic loading.

An adjacent `PlayerMayCost(player, PayMana([...]))` and
`Unless(CostWasPaid, [actions])` pair builds one `UnlessPaid` effect.
Each pair keeps its own result, including nested unpaid bodies. Payment uses
the engine's total-cost transaction during resolution. Spell-controller payer
references use the current controller of the declared stack target.
Variable mana payments use the separate adapter documented below. Fixed
mana/life alternatives and fixed life payments use the adapter at the end of
this document. Optional sacrifice payments use the adapter below; other
nonmana payments remain rejected.
Contract and tests: [RON optional payments](designs/mtgish-ron-optional-payment.md).

### RON optional sacrifice payments and branch polarity

In supported spell, activation and trigger action sequences,
`MayCost(SacrificeAPermanent(filter))` uses the resolving controller as payer.
`PlayerMayCost(player, SacrificeAPermanent(filter))` uses the existing typed
player-reference grammar. The filter uses the activation sacrifice-cost grammar,
including nonempty recursive `And`/`Or`. It selects one current, unphased
matching permanent the payer controls, including the source if it matches.
The selected card moves to its owner's graveyard. Regeneration does not apply.

Each payment must have an adjacent `If(condition, [actions])` or
`Unless(condition, [actions])`, with a nonempty action body. Symbol
`CostWasPaid` or `CostWasntPaid` supplies the condition. `If(CostWasPaid, ...)`
and `Unless(CostWasntPaid, ...)` run after successful payment;
`If(CostWasntPaid, ...)` and `Unless(CostWasPaid, ...)` run after decline or
failed payment. Both conditions also work with the existing fixed mana costs.
The payment result stays local to this pair, including nested pairs.
Detached result reads, plural sacrifice costs and other optional costs remain
errors. This does not add a general resolving `If` adapter.

The adapter uses `SacrificeMatchingIncludingSourceCost`, `IfPaid`, `UnlessPaid`
and the shared cost transaction. `TestRONOptionalSacrifice*` covers complete
Elder Spawn and Yawgmoth Demon records, declaration rejection, payment choices,
source identity, phasing, copies, clones, nested results and native loading.

## RON static animation

`EachPermanentLayerEffect` supports a combined static animation list with exactly
one `SetPT(PT(power, toughness))`, one `AddCardtype(Creature)`, and optionally one
`SetColor(SimpleColorList(colors))`. Values are signed integer literals. The
list order does not change layer order. Color symbols are White, Blue, Black,
Red and Green; an empty list makes the recipients colorless. Omission keeps
current colors. Other modifications cannot share this animation list.

The existing static permanent filters apply. Factories lower the complete
combination to `Static(AnimateEach(set, power, toughness, colors...))`.
The engine selects subjects in layer 4 and retains those incarnations for
layers 5 and 7b within the same application. Later applications use current
membership. Other static, Aura and temporary type/base/color shapes remain
separate compiler contexts. See [the contract and tests](designs/mtgish-ron-static-animation.md).

RON action sequences also accept `MayCost(PayMana([...]))` followed by
`Unless(CostWasPaid, [actions])`. This pair uses the stored resolving controller
as payer. Costs are fixed mana costs; an empty list still offers a choice.
`PermanentDealsDamage(ThisPermanent, value, Player(You))` uses that controller
as the recipient and the engine's damage source, LKI and prevention services.
Other explicit damage-player references remain unsupported. Shapes, limits and
tests: [RON upkeep payments](designs/mtgish-ron-upkeep-payment.md).

## RON fixed draw and activation life cost

Symbol `DrawACard` draws one card in supported action contexts. In
`PlayerAction(player, DrawACard)`, the typed player reference selects the player;
otherwise the resolving controller draws. Activation cost
`PayLife(Integer(n))` accepts a nonnegative fixed integer and uses the existing
atomic total-cost payment. Optional fixed life costs use the resolution adapter
at the end of this document. Dynamic life costs remain rejected.
Fixed `AnyManaColor` production asks for one color for the complete amount.
See [the exact shape and gameplay report](designs/mtgish-ron-fixed-draw.md).

## RON spell-cast life triggers and paid bodies

`TriggerA(WhenAPlayerCastsASpell(AnyPlayer, filter), actions)` uses the existing
spell filters and `SpellCast`. The opponent scope and event references have
separate contracts below. An adjacent fixed-mana MayCost or PlayerMayCost
followed by `If(CostWasPaid, [actions])` uses `IfPaid`. Each nonempty body has
its own payment result, including nested pairs. Payment is offered at resolution
to the stored ability controller or explicit typed payer. X and other nonmana
payments remain separate shapes; optional sacrifice support is described above.
See [the contract and tests](designs/mtgish-ron-spell-cast-life.md).

## RON fixed selected discard

In supported spell, activation and trigger action lists, symbol `DiscardACard`
and `DiscardNumberCards(Integer(n))` use `mage.Discard` with a fixed amount.
The integer must be nonnegative. Bare actions affect the resolving controller;
`PlayerAction(player, action)` uses an existing typed player reference, including
an explicitly declared `Ref_TargetPlayer`. Each factory builds fresh effects
and target bindings. A target declaration alone does not change the recipient
of a bare discard action.

The affected player selects cards at resolution, after preceding draws. Short
hands discard as many cards as possible; zero does not ask for a choice. Selected
cards move through the shared discard replacement batch before discard events
are sent. Dynamic amounts, saved discard results, and undeclared
event players remain unsupported RON action forms. The selected discard cost
contract is documented below. Evidence:
`TestRONFixedDiscardRealCards`, `TestRONFixedDiscardContexts`,
`TestRONFixedDiscardReplacementAndEvents`,
`TestRONFixedDiscardZeroEmptyAndContinuation`,
`TestRONFixedDiscardActivationRollback`, `TestRONFixedDiscardInvalid` and
`TestRONFixedDiscardInventory`.

## RON spell-cast event references

Inside `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, filter), actions)`, symbol
`Trigger_ThatSpell` reads the cast spell and symbol `Trigger_ThatPlayer` reads
its caster. CounterSpell accepts the event spell; supported draws/discards and
fixed PlayerMayCost accept the event player. ControllerOfSpell accepts the event
spell and reads its current controller. The event references do not add targets.
The compiler rejects these references outside this event and rejects explicit
event-player life actions. Existing spell filters and payment continuations retain
their shapes. See [the supported contract and tests](designs/mtgish-ron-spell-cast-references.md).

## RON temporary animation and base values

In the existing end-of-turn `CreatePermanentLayerEffectUntil/3` action, fixed
`SetPT(PT(integer, integer))` sets base P/T; `AddCardtype` supports Creature,
Artifact, Land and Enchantment. `AddCreatureType(symbol)` adds a creature subtype;
`SetCreatureType(symbol)` replaces the creature subtype family. Existing types
and abilities remain, and all modifications share one engine `Until` installation.
For becoming a named artifact creature, use replacement as required by CR 205.1b.
The compiler rejects invalid types/subtype families before building a factory.

Permanent-source `TargetPermanent` filters accept `Other(ThisPermanent)` alone
or under `And` with existing filters. They use the engine's source-exclusion
option at announcement and resolution. `Other` in disjunctions or with other
reference kinds remains unsupported. Exact scope, engine reuse, source corrections
and gameplay evidence: [temporary animation](designs/mtgish-ron-temporary-animation.md).
The supported collection animation and equal base-value extensions are described
in the temporary collection animation contract below. Other shapes remain gaps.

### RON temporary collection P/T

`CreateEachPermanentLayerEffectUntil(filter, [AdjustPT(integer, integer), ...],
UntilEndOfTurn)` accepts a nonempty modification list in supported resolving
action contexts. It reuses generic permanent filters and
`ControlledByAPlayer(SinglePlayer(You))` alone or under conjunction. Controller
selectors under disjunction, source exclusions, target collections and other
modifications/durations remain errors. You means the resolving controller.

The compiler reuses the immutable permanent-set filter plan and lowers to
`Until(EndOfTurn, ModifyEachPT(...))`. Members are captured at resolution;
later entries are excluded, and later control or combat changes do not change
membership. Phased objects are excluded. Installed effects keep incarnation
identity and expire at cleanup. See
[the collection report](designs/mtgish-ron-temporary-collection.md) for exact
contexts, full-record audit and gameplay mappings.

## RON activation timing

On permanent cards, `ActivatedWithModifiers(cost, actions, modifier)` accepts
`ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn))` and
`ActivateOnlyIf(PlayerPassesFilter(You, IsTheirUpkeep))`. Costs and action bodies
use the same builders as `Activated`. The engine checks the current activating
controller's turn, and the upkeep step where required, before payment. The
restriction does not repeat when the ability resolves. Every factory creates
fresh costs, targets, effects and abilities. The activation limit forms below
can be combined with this timing. Other modifier forms remain errors. See [the timing slice](designs/mtgish-ron-activation-timing.md)
for exact shapes and real-card evidence.


## RON activation step restrictions

`ActivatedWithModifiers/3` also accepts
`ActivateOnlyIf(IsDuringCombat)`, `ActivateOnlyIf(IsDuringEndOfCombatStep)`,
`ActivateOnlyIf(IsDuringDeclareBlockersStep)`, and `ActivateOnlyIf(IsDuringUpkeep)`.
The conditions are bare symbols. Combat includes beginning of combat, declare
attackers, declare blockers, both damage steps, and end of combat. Each form
permits either player's turn. Existing controller permissions still apply.

These forms lower to `WithActivationCondition` and can be combined through the
existing modifier `And([modifiers])`. Conditions, own-turn restrictions and use
limits remain conjunctive. Queries and mana planning check the same conditions
without payment. Resolution does not repeat the announcement restriction.
Malformed tuples and other step predicates remain errors.

In supported resolving bodies, `CreatePermanentLayerEffectUntil/3` accepts
bare `UntilEndOfCombat` for supported singular permanent modifications.
It reuses `Until(core.EndOfCombat, modifications...)`, captures each subject's
incarnation at resolution, and expires after end-of-combat priority. Singular
spell, trigger and delayed modification contexts also accept this duration. Collection modifications retain their separate duration restrictions.

Evidence: `TestRONActivationSteps*` covers Jade Statue and Desert through their
complete repository records, including animation, damage, timing, payment,
identity, phasing, copies, clones, combat and native expiry. Tolaria and Lesser
Werewolf have only the timing prerequisite; their other rules remain unsupported.

## RON activation limits

`ActivatedWithModifiers(cost, actions, ActivateOnlyAsASorcery)` accepts the bare
symbol as an activation modifier. Nonempty, nested `And` modifier lists can
combine it with existing timing conditions and limits. The compiler applies
`mage.WithSorcerySpeed()` after the other options, so an own-turn option cannot
erase the main-phase or empty-stack restriction. Costs, references and bodies
retain their existing declaration checks. Tuples such as
`ActivateOnlyAsASorcery()` and uses in actions or costs reject.

The existing engine permits activation in either main phase of the activating
player's turn with an empty stack. The priority loop supplies priority; this
adapter adds no priority permission. The restriction applies at activation,
before choices and payment, and does not apply again when the ability or a copy
resolves. Mana planning uses the same timing restriction as explicit activation.
Evidence: `TestRONSorceryActivation*`. Gaea's Touch and Illusionary Mask have
only this prerequisite supported; their complete records still reject at their
action bodies. The repository prerequisite tests use neutral test bodies and
do not establish complete gameplay for those cards.

Permanent `ActivatedWithModifiers` accepts the symbol `ActivateOnlyOnceEachTurn`
and `ActivateNoMoreThanNumberTimesEachTurn(Integer(n))` for a positive fixed
integer. A nonempty `And([modifier, ...])` combines supported limits and timing;
nested conjunctions are allowed. More than one numbered maximum is rejected.
Dynamic or nonpositive limits remain errors. Costs and actions retain their
existing context checks.

Limits belong to an ability of one permanent incarnation. Failed activation
consumes no use; control changes and phase cycles retain uses. Returned cards
start unused. Copies do not consume uses, cloned games keep separate counts,
and final cleanup resets phased sources too. The shared engine service copies
runtime action state before it records a successful activation; card definitions
remain unchanged. See [activation limits](designs/mtgish-ron-activation-limits.md)
for exact shapes, dependencies and real RON gameplay evidence.

RON activation costs accept `SacrificeAPermanent(filter)` through the existing
matching-sacrifice transaction. The activating player chooses one matching
permanent they control, including the source when it matches. Costs are fresh
per factory; nested filter `And`/`Or` lists must be nonempty. Saved sacrifice
results, source-relative filters and resolution/optional/casting contexts remain
unsupported. See [the exact contract and tests](designs/mtgish-ron-sacrifice-costs.md).

## RON single-permanent counter placement

`PutCounters([ACounterOfTypeOnPermanent(counter, permanentRef)])` places one
counter. `NumberCountersOfTypeOnPermanent(value, counter, permanentRef)` in the
same one-instruction list uses the typed resolving values described below.
Nonpositive amounts place nothing. Typed source, Aura host and declared permanent
target references use the same scope and incarnation rules as other actions.
Placement uses AddCountersTo and the shared replacement pipeline. Counters
remain through cleanup; returned objects do not receive pending placement.

Named permanent kinds and supported literal PTCounter pairs are validated before
factory construction. Multiple instructions, player counters, unsupported
values/kinds, entry/removal and placement-as-cost remain errors. See
[the exact shapes, inventory audit and tests](designs/mtgish-ron-counter-placement.md).

## RON fixed source-counter activation costs

`RemoveCounters([ACounterOfTypeFromPermanent(counter, ThisPermanent)])` removes
one counter as an activation cost. `NumberCountersOfTypeFromPermanent(Integer(n),
counter, ThisPermanent)` uses a nonnegative fixed amount, including zero. The list
must contain one instruction. Supported permanent counter types match placement.
Use these costs in `Activated` or `ActivatedWithModifiers`, including `And` costs.
Factories create fresh `RemoveCountersCost` instances; payment uses the shared
atomic transaction before resolution. Other source references, X/any/all amounts,
removal actions and optional counter payments remain errors. See
[the exact contexts and tests](designs/mtgish-ron-counter-costs.md).

## RON basic creature-token creation

RON `CreateTokens/2` lowers one creature-token instruction and an empty modifier
list to `mage.CreateTokenCopies(player, count, definition)`. The player is the
lexical player, with `You` as the default. Both `PlayerAction(player, CreateTokens(...))`
and `PlayerActions(player, [CreateTokens(...), ...])` support the same complete
token shape. `PlayerActions` saves its player once and supplies `ActionPlayer`.
Nested scopes retain the existing binding rules. Instructions accept
`TokenWithPT/6`, `NamedTokenWithPT/7`, or `NumberTokens/2`. Printed characteristics
use `DefineToken` with typed P/T, types, colors, subtypes and keyword options.
Counts use existing in-scope instance values at resolution, including fixed
integers, announced spell X and typed permanent information. Empty
optional options are omitted. Each action builds a fresh effect; the engine's
immutable token definition contains no chosen state.

Waiting delayed token creation declares explicit source reads in its player and
count, including typed permanent information and nested arithmetic. A zero count
keeps these declarations. Enumeration does not evaluate either operand or create
tokens. Token ability bodies use the created token's source scope and do not
declare the producer's source. Native `CreateOneToken` and `CreateTokenCopiesInto`
use the same operand contract; this does not add saved token results to RON.

See [the supported shapes and card tests](designs/mtgish-ron-token-creation.md).
Token source damage-to-player triggers are supported as described below. Other
token abilities, supertype options, copying, creation modifiers and saved token
results remain unsupported by this adapter.

### RON sacrifice payment values

In `Activated/2` and `ActivatedWithModifiers/3`, a single
`SacrificeAPermanent/1` cost can supply a saved number to its resolving actions.
`ToughnessOfTheSacrificedCreature` requires the exact filter
`IsCardtype(Creature)`. `ManaValueOfTheSacrificedPermanent` accepts the existing
matching-sacrifice filters. Both references are symbols. The number is captured
at payment, before sacrifice, and belongs to that activation.

The supported consumers are `GainLife/1`, `LoseLife/1`, `DrawNumberCards/1`, and
`NumberTokens/2` in the existing basic token-creation forms. These references do
not enable activation mana, damage, static values, or optional sacrifice
payments. A declaration cannot combine both characteristic
references or use more than one sacrifice cost, including source sacrifice.
Each factory builds the cost and its consumers with the same private engine
`NumberBinding`; copies keep the paid number without paying again.

For a nonmodal instant or sorcery, one
`CastEffect(AdditionalCastingCost(SacrificeAPermanent(filter)))` supplies the
same saved values to `SpellActions`. It uses the existing matching-sacrifice
filters and consumers above. `AddManaRepeated(production,
ManaValueOfTheSacrificedPermanent)` also accepts one fixed production symbol:
`ManaProduceW`, `ManaProduceU`, `ManaProduceB`, `ManaProduceR`, `ManaProduceG`, or
`ManaProduceC`. The spell pays its printed mana and sacrifice during casting;
the mana action runs during resolution. Each factory builds the additional cost
and effects with the same private binding. Copies retain the payment and use
their own controller. Goblin Grenade and Sacrifice use this form.

Malformed, duplicate, permanent-card and modal declarations remain errors.
Other additional costs, optional costs, activation mana from sacrifice values,
compound/any-color productions and restricted mana remain unsupported.

### RON source counter payment results

An activation can pay one
`RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(counter, ThisPermanent)])`
cost. Symbol `TheNumberOfCountersRemovedThisWay` reads its saved paid count in
that activation's life, draw, basic token-count or repeated-mana actions.
`AddManaRepeated(production, TheNumberOfCountersRemovedThisWay)` adds the count
in each declared production; `AnyManaColor` chooses one color for the amount.
Zero is a valid payment. Each factory builds a private typed number binding.
Unbound, tuple, cross-rule, non-source and ambiguous counter-payment forms fail
compilation. Optional resolution payments remain unsupported. The shared engine
transaction locks the choice and preserves rollback and source identity.
Mana production queries do not choose counter payments; choice-cost abilities
remain outside automatic mana planning. See the
[counter payment report](designs/mtgish-ron-counter-payment.md) for shapes and tests.

## RON end-step trigger adapter

Permanent-source RON rules accept
`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer), actions)` and
`TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You)), actions)`.
The first form matches either player's end step. The second form uses the
source's current controller when the event occurs. Both use the engine's step
patterns and pending-trigger path.

Within these action lists, symbol `Trigger_ThatPlayer` is a typed reference to
the player whose end step began. It is separate from `You`, which refers to the
pending ability's controller. It can supply existing player actions, including
life, draw, discard and optional fixed mana payment. Each factory builds a fresh
player binding and any declared targets. Targets are chosen at stack placement.

Other end-step player filters, `TriggerI`, and `CreateFutureTrigger` remain
unsupported. This adapter does not provide history queries or a delayed trigger.
Evidence: `TestRONEndStep*`, including complete Ball Lightning and Breeding Pit
RON gameplay; [slice report](designs/mtgish-ron-end-step.md).

### RON tapped entry

Permanent cards accept exactly
`AsPermanentEnters(ThisPermanent, [EntersTapped])`. The compiler lowers this
form to the engine's `EntersTapped` entry attribute. The permanent is tapped
before entry effects and triggers inspect it. Entry does not create a tap event
or put a tapped-entry ability on the stack. Normal untapping is permitted.
Re-entry applies the rule again; phasing does not. Card copies retain the rule,
and cloned games keep separate tapped states.

The source must be the symbol `ThisPermanent`, and the list must contain only
the symbol `EntersTapped`. Empty lists, other subjects, tuple-shaped symbols,
additional operations, entry choices and entry counters remain unsupported.
`EntersTapped` is not a general action or a top-level keyword declaration.
See `TestRONEntryTapped*` and the [slice report](designs/mtgish-ron-entry-tapped.md).

## RON creature death history in counter amounts

`PutCounters([NumberCountersOfTypeOnPermanent(amount, counter, subject)])`
accepts `NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature))`
as its amount in supported spell, activation and trigger action lists. This
exact filter lowers to `mage.CreaturesDiedThisTurn()`. The count is read when
placement resolves. It includes tokens and creatures from either player,
including deaths before the source entered. It does not count exile, return to
hand, phasing or prevented destruction. Other history filters remain errors.
The existing typed subject and counter replacement contracts still apply.
RON gameplay evidence: `TestRONDeathHistory*` for Khabál Ghoul and all six
Scavenging Ghoul records. Intervening-if history conditions remain unsupported.

## RON storage-land lifecycle

RON supports `PermanentRuleEffect(ThisPermanent,
[MayChooseNotToUntapDuringUntap])` and
`TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),
PermanentPassesFilter(ThisPermanent, IsTapped), actions)` on permanent cards.
The first uses the ability layer's optional untap attribute. The second checks
current tapped state at detection and resolution through WithInterveningIf.
Actions use the shared compiler and fresh instance bindings. Other operations,
subjects, conditions and events remain declaration errors. Full-card and invalid
shape evidence is recorded in the storage-land lifecycle entry in
[the progress ledger](designs/mtgish-ron-progress.md).

## RON next-untap restriction action

`PermanentDoesntUntapDuringControllersNextUntap/1` accepts an in-scope symbol
`ThisPermanent`, `HostPermanent`, or `Ref_TargetPermanent`. The shared resolving
action compiler supports it in spell, activation and supported trigger action
lists. The permanent source, Aura host or single typed target must exist in that
declaration context. Wrong arity, tuple references, undeclared reads and reads
from another rule fail compilation.

The action lowers to `mage.SkipNextUntap` with the instance's typed reference.
It does not tap the permanent or prevent other untap effects. The restriction
expires during the next untap step of the permanent's current controller, even
if the permanent is untapped. Control changes do not capture the old controller.
Repeated applications expire together. See
[the contract and tests](designs/mtgish-ron-next-untap.md).
Collection actions remain unsupported by this adapter. The self static form
is described below.

## RON self untap prohibition

The permanent-rule compiler accepts
`PermanentRuleEffect(ThisPermanent, [DoesntUntapDuringControllersUntap])`.
It creates a source-bound static ability through `Static`, `While`, and
`GrantKeywordModification(This, AttrDoesNotUntap)`. The existing optional-untap
form still uses `AttrMayNotUntap`. The operation list must contain exactly one
symbol. Only the source symbol `ThisPermanent` is accepted in this context.

The prohibition applies during each untap step of the current controller.
Other untap effects remain permitted. Ability removal disables the prohibition;
cleanup can restore it. Conditional, Aura-host, collection, and temporary rule
forms remain rejected. See [the slice ledger](designs/mtgish-ron-progress.md)
for the real-card tests and supported contexts.

## RON global landwalk blocking overrides

`EachPermanentRuleEffect(And([IsCardtype(Creature),
HasAbility(Landwalk(IsLandType(subtype)))]),
[CanBeBlockedAsThoughItDidntHave(Landwalk(IsLandType(subtype)))])`
uses the existing global landwalk blocking rule. The two filter terms may be
in either order. Both subtype operands must match. Supported subtypes are
Plains, Island, Swamp, Mountain, Forest and Desert.

The creature keeps its landwalk ability. Only blocking checks ignore that
landwalk; other evasion abilities still apply. The source must be phased in,
on the battlefield, and have its static ability. Each factory builds a fresh
effect. Extra filters, other ability operands, mixed operation lists, and
self-only or temporary contexts remain unsupported. This adapter does not add
general landwalk filter support to targets or other selectors.

## RON choose-one spell modes

`SpellActions(Modal_ChooseOne([mode, mode, ...]))` accepts at least two
mode bodies. Each body uses the supported `ActionList` or `Targeted` grammar.
Each mode has its own target scope. A target reference cannot read a declaration
from another mode. Each factory call builds fresh targets and effects. Internal
declaration names include a mode namespace; RON references keep their spelling.

The adapter uses `NewModalSpell`. The controller chooses one mode and only its
targets during casting. Resolution executes only that mode and checks its target
legality. Copies retain the mode and can choose new targets for that mode.
Mode labels use effect sequence text with readable target names. All mode bodies must compile;
an unsupported body rejects the entire record. Nested modes,
trigger modes, and `Targeted_Modal` remain unsupported. Supported player life
actions and finite prevention can run in each mode.

RON `PutPermanentIntoItsOwnersHand(reference)` uses `mage.ReturnToHand`.
The argument must be one in-scope permanent reference: `Ref_TargetPermanent`,
`ThisPermanent` on a permanent source, or `HostPermanent` on an Aura.
The existing action compiler accepts it in spells, choose-one spell modes,
activations and supported triggers. Target declarations are built per factory
instance. Unsupported enclosing forms still fail compilation.
The action reads the current object through its captured incarnation and moves
its card to its owner's hand. Phased or departed objects are absent; a returned
card is a new object. An Aura uses its current attachment, or the last attachment
of its captured source after departure. It cannot follow a returned source.

## RON typed single damage targets

Symbol `AnyTarget` in the existing single-target `Targeted` form now creates a
fresh `mage.DeclareDamageTarget` declaration. Symbol `Ref_AnyTarget` in
`SpellDealsDamage(ThisSpell, value, Ref_AnyTarget)` or
`PermanentDealsDamage(ThisPermanent, value, Ref_AnyTarget)` reads that instance's
recipient through `mage.DealDamageFrom`. The engine validates the declaration
and read together. The recipient can be a player, creature or planeswalker;
an ordinary land cannot be the target. Values retain the existing fixed/X
resolution contract.

Spell modes have separate declaration names and bindings. Copies retain their
mode and can change the recipient through the engine's target service. Target
legality, permanent incarnation, source LKI, damage replacement and prevention
remain engine behavior. This adapter also works in existing activation and
trigger action lists. Numbered references, multiple target groups and target unions
with arbitrary filters remain unsupported shapes. Evidence:
`TestRONDamageScope*` and the migration ledger's P0-TYPED-DAMAGE-RECIPIENT entry.

`TargetPlayerOrPermanent(AnyPlayer, IsCardtype(Planeswalker))` in the same
single-target form builds `mage.DeclarePlayerOrPlaneswalkerTarget`.
`TargetPlayerOrPermanent(Opponent, IsCardtype(Planeswalker))` instead builds
`mage.DeclareOpponentOrPlaneswalkerTarget`. Only the player branch excludes the
resolving controller; either player's planeswalker can be selected. Copies
check the player restriction against their own controller.
`Ref_TargetPlayerOrPermanent` reads that declaration as a damage recipient.
In the `AnyPlayer` form, either player or either player's planeswalker can be selected. A creature that
is also a planeswalker qualifies; a creature without that type does not.
This exact shape works in supported spells, choose-one spell modes, activations
and triggers, including damage packets. The reference is not a singular player
or permanent reference. It cannot be used as one. Other player filters, permanent
filters, arities and unbound or cross-mode reads fail compilation.
Each factory and spell mode builds its own declaration. Copies and clones use
the engine's target identity and legality services. This adapter adds no engine
API. Evidence: `TestRONPlayerPlaneswalker*`; complete records for Axelrod
Gunnarson and Witch Hunter supply the gameplay mappings.

## RON typed permanent damage recipients

RON damage actions also accept `Permanent(Ref_TargetPermanent)` as the recipient
in `SpellDealsDamage(ThisSpell, value, recipient)` and
`PermanentDealsDamage(ThisPermanent, value, recipient)`. The enclosing action
must declare its own single `TargetPermanent`. Spell, mode, activation and
supported trigger action lists use that declaration's fresh
`mage.PermanentTargetBinding` with `mage.DealDamageFrom`.
The wrapper does not select a new target or bypass the target filter at
resolution. Unbound, numbered, player, source and host operands remain rejected
in this shape. Existing fixed and announced-X values retain their contracts.
Evidence: `TestRONPermanentDamage*`.

## RON targeted temporary blocking restriction

`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked],
UntilEndOfTurn)` is supported in resolving action lists with a declared
`TargetPermanent`. Spell, choose-one spell mode, activated ability and supported
trigger contexts use the same per-instance target binding. Other rule operations,
subjects and durations remain rejected.

Permanent filters accept `PowerIs(LessThanOrEqualTo(Integer(n)))`,
`PowerIs(GreaterThanOrEqualTo(Integer(n)))` and `PowerIs(EqualTo(Integer(n)))`
for signed integer constants. These filters read current power at each query;
combine them with `IsCardtype(Creature)` when the subject must be a creature.
Dynamic bounds and other comparison forms remain unsupported.

Dwarven Warriors and Tawnos's Wand use the power-at-most-two target and the
same blocking restriction. Target legality is checked at announcement and
resolution. A later power change does not end a resolved restriction. The
restriction follows the captured incarnation until cleanup and survives source
departure, ability removal and a phase cycle. It does not follow a returned card.

## RON collection untap restrictions and tap actions

`EachPermanentRuleEffect(filter, [DoesntUntapDuringControllersUntap])` accepts
one bare restriction operation and a supported, source-independent permanent
filter. The compiler uses `PreventUntapDuringStepForMatching`. The engine checks
final characteristics after phasing and before the untap operation. It does not
grant an ability to the selected permanents. Other untap effects remain legal.
The rule stops when its source leaves, phases out, or loses its abilities.
Membership is selected again at each untap decision.

`IsSupertype(Legendary)` is supported in permanent filters, including nested
`And` and `Or`. Other supertypes and card/spell filter contexts remain unsupported.
The filter uses `IsLegendary`.

`TapEachPermanent(filter)` accepts a supported, source-independent permanent
filter in resolving spell, mode, activation and trigger action lists. It lowers
to `TapEach(MatchingPermanents(filter))`. Selection occurs at resolution, excludes
phased permanents, and taps the selected permanents together before tap events.
Target-group, chosen-result and player-dependent collection selectors remain
unsupported by this adapter.

Source-entry triggers now use `WhenEvent(Enters(This), body)`. An event object is
not a target declaration. A nontargeted trigger can resolve after its source
leaves or phases out. See the progress ledger for complete real-card mappings.

### RON random discard actions

In resolving spell, mode, activation and supported trigger action lists,
`DiscardACardAtRandom` is a bare symbol. `DiscardNumberCardsAtRandom(Integer(n))`
requires a nonnegative integer; `DiscardNumberCardsAtRandom(ValueX)` requires an
announced X scope. Other amounts and tuple forms of the singular action remain
rejected. These actions lower to `mage.DiscardAtRandom`. Random discard costs
have the separate declaration contract below.

Direct actions affect the resolving controller. `PlayerAction(player, action)`
uses the existing typed player reference plan: `You`, declared `Ref_TargetPlayer`,
or an available step/cast event or spell-controller reference. Each factory
builds its own target/event bindings. A player read without its declaration is
an error. A target declaration alone does not change the direct action's player.
Selection occurs at resolution before the shared simultaneous discard batch;
it does not ask the affected player to choose cards.

Full-record gameplay mappings: Hymn to Tourach, Mind Twist (all six reprints),
Ring of Renewal and Gwendlyn Di Corci in `TestRONRandomDiscard*`. The engine
contract is documented in the random-discard section of
[engine-rules-contracts.md](engine-rules-contracts.md).

### RON printed rampage

A permanent's top-level `Rules` list accepts `Rampage(n)` with one nonnegative
integer literal. It lowers to `mage.RampageTrigger(n)` with a fresh ability per
factory construction. Zero is legal and still creates the trigger. Bare symbols,
negative values, `Integer(n)`, X, extra arguments, and resolving-action uses are
rejected during compilation. `AddAbility([Rampage(n)])` and selected ability
choices remain unsupported contexts.

Rampage counts blockers at resolution and grants its fixed bonus until cleanup.
Its source remains the original incarnation. Full-record gameplay mappings:
`TestRONRampageRealCards` covers Aerathi Berserker, Chromium, Craw Giant, Frost
Giant, Hunding Gjornersen, Marhault Elsdragon and Wolverine Pack.
`TestRONRampageNativeCombat` checks declared-blocker legality, flying and trample.
Chromium also uses the shared `TestRONUpkeepPayment*` lifecycle tests.

## RON resolving continuous durations

### End-of-combat duration

`CreatePermanentLayerEffectUntil(subject, modifications, UntilEndOfCombat)`
accepts the existing singular permanent references and temporary modifications
in all supported resolving bodies, including spells, triggers and delayed bodies.
The duration is a bare symbol. It does not grant casting permission or declare a
combat target. Collection layers, rule effects and installed event subscriptions
retain their separate adapters and restrictions.

Lowering reuses `mage.Until(core.EndOfCombat, modifications...)`. Resolution
captures each subject's incarnation and amounts. Source departure and removal
from combat do not end the effect. Phasing suspends application; a phase cycle
keeps identity. A returned permanent cannot inherit the effect. Each copied
spell resolves its own target frame; game clones retain separate expiry state.

The effect remains active during end-of-combat trigger resolution and priority.
Native `RunStepWithPriority(EndCombat)` removes it when the combat phase ends,
before the next combat. This does not expire independent turn-long prevention
or next-end-step destruction.

`TestRONEndCombatLayerGlyph` executes the complete Glyph of Destruction record,
including payment, blocking-Wall targets, +10/+0, damage prevention, next-end-step
destruction, regeneration, copies, clones and incarnation checks.
`TestRONEndCombatLayerDelayedResolution` covers delayed creation during the
end-of-combat step. `TestRONEndCombatLayerUpkeep` covers a noncombat trigger.
`TestRONEndCombatLayerInvalid` rejects malformed durations,
missing bindings and invalid modifications. Native
`TestEndCombatLayerPhaseBoundary` checks expiry after trigger resolution.

### Next-upkeep duration

`CreatePermanentLayerEffectUntil(subject, modifications, UntilNextUpkeep(You))`
accepts the existing singular permanent references and temporary modifications.
It applies in resolving action bodies, including spells, activations, supported
triggers, and supported delayed actions. The duration operand must be bare `You`;
other players and malformed declarations are rejected. Collection modifications,
static operations, costs, and control changes gain no new support.

The adapter uses
`mage.UntilNextUpkeep(player mage.PlayerRef, modifications ...mage.ContinuousModification)`.
The engine supports validated typed player references, saves the player at
resolution, and captures the existing modification subjects and values. An absent
player starts no effect. Changes to the source or recipient controller do not
change the saved player. Source departure, return, ability removal, and phasing
do not end the duration. Recipient departure breaks the captured identity;
recipient phasing suspends application but does not postpone expiry.

The effect ends as the next actual upkeep of the saved player begins, before
upkeep event detection and priority. It lasts through cleanup and other players'
upkeeps. A skipped upkeep does not expire it; an extra upkeep can expire it in
the same turn. This operation creates no delayed trigger or stack object.
`Until(UntilYourNextTurn, ...)` remains an invalid declaration; the explicit
player adapter supplies the lifetime policy.

Erhnam Djinn uses this duration with its controller-upkeep trigger and opposing
non-Wall creature target. Xenic Poltergeist still needs `SetCardtypes` and
`SetPowerAndToughnessBoth` adapters. Halfdane uses the separate end-of-next-upkeep adapter documented below.

### Tapped-source duration

`CreatePermanentLayerEffectUntil(subject, modifications,
ForAsLongAsPermanentRemainsTapped(ThisPermanent))` supports the existing typed
subjects and temporary modifications in resolving permanent abilities, including
activations and supported triggers. The source operand must be the symbol
`ThisPermanent`. Host, target, event and numbered source operands are rejected.
Spell-only, static, cost, collection-layer and control-duration contexts are not
added. Existing `UntilEndOfTurn` declarations retain their behavior.

The compiler lowers to this engine service:

```go
func UntilSourceUntaps(source sourceRef, modifications ...ContinuousModification) ResolvingEffect
```

Call it with `mage.This`. The service captures the source incarnation and
modification subjects/values at resolution. It starts only if that source is
present, phased in and tapped. An untap replaced by removal of a stun counter
does not end it. Actual untapping, source departure and source phasing end it
permanently; retapping or phase-in cannot restart it. Control changes do not
end this tapped-state duration. Target phasing suspends application, and target
return does not change the captured membership.

Tawnos's Weaponry, Spirit Shield and Zelyon Sword use this duration with their
printed mana/tap costs and P/T bonuses. Their separate permission to remain
tapped during untap still permits the choice after the target leaves.
Evidence: `TestRONSourceTappedDuration*` and `TestUntilSourceUntaps*`.

## RON opponent-land tap triggers

`TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(subtype),
ControlledByAPlayer(Opponent)])), actions)` uses
`mage.OpponentLandBecomesTapped().OfSubtype(subtype)`. The conjunction must
contain exactly these two filters, in either order. The subtype must be a symbol
that the engine accepts as a land subtype. Existing subtype spelling conversion
applies, including `Urzas` and `PowerPlant`. Extra filters, disjunctions,
source/host selectors, other controller selectors and event-result reads remain
unsupported in this context. The existing trigger action compiler supplies
per-instance actions and local targets; this event does not declare a
`Trigger_ThatPermanent` or `Trigger_ThatPlayer` binding.

Matching uses the land's type, subtype and controller when it becomes tapped.
The pending trigger retains its controller. Mana tapping and other tap operations
can trigger it; entry tapped and tapping an already tapped land cannot. Source
or land departure and later control changes do not erase a pending trigger.
Phased sources do not trigger. Copies use the copy controller for `You`.

Real RON evidence: Lifetap (all six reprints) and Lifeblood in
`TestRONOpponentLandTapRealCards` and `TestRONOpponentLandTapPaymentAndBatch`.
Declaration and syntax evidence: `TestRONOpponentLandTapInvalidAndInventory`
and `TestRONOpponentLandTapSubtypeAndOrder`.

### RON Aura host tap events

On an Aura, `TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent)),
actions)` declares `Trigger_ThatPermanent` for that action body. The reference
reads the permanent captured when the enchanted permanent becomes tapped.
It is distinct from `HostPermanent`, which reads the attachment at resolution.
Existing typed permanent actions can use this binding. It does not declare
`Trigger_ThatPlayer` or supply a controller query or damage-source adapter.

Each factory call creates its own `BindPermanent` declaration. Lowering uses
`AttachedBecomesTapped` and `From`; the engine retains separate captures for
pending events, copies and clones. The exact form executes Blight and Spirit
Shackle. Other `SinglePermanent` tap-event subjects remain unsupported.

## RON global combat prevention

In a resolving action list, the RON form
`CreatePreventDamageUntil(CombatDamageWouldBeDealt, [PreventThatDamage], UntilEndOfTurn)`
uses `mage.PreventAllCombatDamage`. The event and duration must be symbols.
The operation list must contain exactly the symbol `PreventThatDamage`.
The adapter rejects other events, operations and durations before factory creation.
It does not declare a source, recipient, target or result binding.

The shared engine installs prevention at resolution and removes it at cleanup.
It applies to all combat damage for the rest of the turn, including later combats,
and leaves noncombat damage unchanged. Fog, Darkness and Holy Day use this
form in spells; Spore Flower uses it after its three-counter activation cost.
Tests: `TestRONGlobalCombatPrevention*`. Other `CreatePreventDamageUntil` shapes
and future amount shields remain separate gaps.

## RON source combat prevention

`CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(ref), [PreventThatDamage], UntilEndOfTurn)`
accepts a bound typed permanent reference in a resolving action. The event is
an exact one-argument tuple; its operation list has exactly the symbol
`PreventThatDamage`, and the duration is exactly the symbol `UntilEndOfTurn`.
Other events, operations, durations, unbound references and cost/static/rule
contexts remain declaration errors. The existing target filter checks creature
legality at announcement and resolution.

The adapter builds `mage.PreventCombatDamageByThisTurn(PermanentRef)` with
per-instance bindings. It captures the resolved incarnation and installs the
existing source prevention replacement until cleanup. It prevents combat damage
by that source, including later combats, but allows damage to that source and
noncombat damage. A returned permanent is a new source; phasing and control
changes preserve the captured identity. Captured old-source damage keeps its
identity after departure. `PreventCombatDamageBy` and `PreventCombatDamageByEach`
still end at end of combat.

Real RON tests cover Horn of Deafening, Lady Evangela and both Subdue instructions.
Telekinesis still requires its separate future untap action. Filtered source sets,
recipient prevention and finite shields are outside this supported shape.

## RON controller restriction on permanent targets

`TargetPermanent(ControlledByAPlayer(SinglePlayer(You)))` restricts a target
by the controller of the spell or ability. It can occur inside nested `And`
filters with existing permanent filters and `Other(ThisPermanent)`.
The adapter uses `mage.TargetYouControl()` on each fresh target declaration.
This restriction is available in supported spell, mode, activation and trigger
contexts. Ownership does not determine target legality. The target must remain
under that controller at resolution. A later control change does not end an
already resolved characteristic modification.

Only the exact `SinglePlayer(You)` shape is supported here. Opponent expressions,
other player references, ownership restrictions and disjunctions containing
controller restrictions remain errors. Static and collection filters retain
their separate adapters. Ashnod's Battle Gear uses this target restriction with
the existing source-tapped duration and optional untap services.

## RON combat prevention to and by one permanent

The resolving action has this exact form:

```ron
CreatePreventDamageUntil(
    Or([
        CombatDamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent)),
        CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent),
    ]),
    [PreventThatDamage],
    UntilEndOfTurn,
)
```

`Or/1` takes a list with exactly one recipient event and one source event,
in either order. Both events require one argument. The recipient uses the
`Permanent/1` wrapper. Both references must identify the same lexical permanent
plan. Supported references are the existing scoped `Ref_TargetPermanent`,
`ThisPermanent`, `HostPermanent`, and bound `Trigger_ThatPermanent` forms.
Unbound, numbered, player, different-subject, duplicate-event, additional-event,
wrong-operation and wrong-duration declarations fail compilation.

This action is available in supported spells, modes, activations and trigger
bodies. It is not a cost, layer modification or static `PreventDamage` rule.
The compiler builds `mage.PreventCombatDamageToAndBy(PermanentRef)` with fresh
instance bindings. The engine owns incarnation capture, replacement selection,
source snapshots and cleanup. This is a union of two events for one object;
it does not introduce a general event-union executor.

Real RON tests cover every rule of Maze of Ith, Ebony Horse (ARN and 3ED), and
Elvish Scout. They retain attacking-creature target checks, the controller
restriction for Horse and Scout, costs, untap and prevention. Untap does not
remove an attacker from combat. Failure to untap due to a stun counter does not
stop prevention. Neither source departure nor later recipient control changes
end installed prevention. Returned permanents have new incarnations.

Finite shields, static Aura prevention (including Gaseous Form), different
source/recipient unions and other durations remain separate gaps.

## RON hand-size values and ordinary upkeep events

`TheNumberOfCardsInPlayersHand(player)` reads the selected player's current hand
when the action resolves. It accepts the existing typed player references,
including `You`, declared `Ref_TargetPlayer`, and `Trigger_ThatPlayer` in a
matching event scope. `Minus(left, right)` subtracts two supported resolution
values recursively. Intermediate results retain their sign; life and damage
services treat nonpositive final amounts as zero.

These values are supported where `compileInstanceValue` is used: resolving
life, draw, damage and existing token-count actions. They do not establish
static values, casting costs, chosen-player references, damage-result players,
or arbitrary temporary P/T value support. Unbound reads remain errors.

`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer), body)` and the
`SinglePlayer(You)` form use the same per-instance player-binding service as
ordinary end-step events. `Trigger_ThatPlayer` saves the player whose upkeep
began; `You` remains the resolving ability's controller. Other player filters
and intervening-if upkeep declarations retain their separate adapters.

`SpellDealsDamage` and `PermanentDealsDamage` now build typed damage actions
with resolution values. `Player(player)` accepts a valid typed player reference;
existing bare target recipients and targeted `Permanent(...)` remain supported.
The shared engine supplies target checks, source snapshots and prevention.
Full-record evidence: Ivory Tower (ATQ/3ED), Storm World (LEG), and Copper Tablet
(LEA/LEB/2ED/CED/CEI) in `TestRONHandSize*`.

## RON static filtered blocking restriction

`PermanentRuleEffect(ThisPermanent, [CantBeBlockedExceptByDefenders(filter)])`
and the Aura form with `HostPermanent` support exactly one operation. The filter
uses the existing source-independent permanent-filter grammar, including nested
`And`/`Or`, type, subtype, color and keyword queries. Each possible blocker must
pass this filter and every other combat restriction. `Defenders` here means
blocking creatures; it does not require the defender keyword. Reach does not
satisfy a filter that requires flying.

The compiler uses `mage.CantBeBlockedExceptBy(subject, filter)` in a static
`While` rule. The Aura reads its current attachment each layer pass. Source
ability removal, source departure or source phasing stops the rule; recipient
ability removal does not remove an Aura's rule. The rule does not grant an
ability to the recipient. Blocker characteristics are queried when block
legality is checked. A restriction change after a legal block does not remove
that block.

The typed engine modification also supports `Until(EndOfTurn, ...)`: it captures
its subject incarnation at resolution, survives source departure and a phase
cycle, and expires at cleanup. A returned card does not inherit the restriction.
This run adds only the static RON spelling, not new temporary RON operations.
Existing source and target constructors remain available. Evidence:
`TestFilteredBlocking*` and `TestRONFilteredBlocking*`.

### RON Aura host-controller upkeep

`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(HostController)), body)`
is supported on Auras. It uses `AttachedControllerUpkeep` with fresh event
bindings in each factory instance. `Trigger_ThatPlayer` is the player whose
upkeep caused the trigger. This binding is available only inside `body`.
A later control or attachment change does not change that player.

`HostPermanent` keeps its existing resolution-time attachment reference. Thus
Unstable Mutation puts its counter on the creature enchanted at resolution.
The event's internal host capture does not replace that reference. A departed
Aura uses the existing attachment LKI and cannot follow its new incarnation.

Complete-record tests cover Cursed Land, Feedback, Wanderlust, Warp Artifact,
and Unstable Mutation. Host-death triggers and host-controller end-step patterns
remain unsupported. Resolving references use the separate contract below.

## RON static collection attack restrictions

`EachPermanentRuleEffect(filter, [CantAttack])` accepts one bare `CantAttack`
symbol on a permanent. It lowers the static filter plan to
`Static(CantAttackEach(PermanentSet))`. The collection reads final characteristics
at attack declaration, including later flying grants or removal. Source control
and collection membership are current. The rule does not grant an ability to
recipients and does not remove creatures already in combat.

Static filters also accept `ExceptFor(filter)` for a pure complement and
`IsNamed(NamedCard("name"))` with a nonempty string. A complement with a contextual
controller or source exclusion is rejected. Existing conjunctions,
`ControlledByAPlayer(SinglePlayer(You))`, and `Other(ThisPermanent)` keep their
scope rules; source-dependent disjunctions remain errors. These adapters also
serve existing static layer contexts and the existing end-of-turn temporary
collection P/T adapter. Temporary collections keep their existing resolution-time
membership capture. No current RON record uses these new filters in that temporary
context. They do not add generic target filters,
card-zone filters or name choices.

Complete RON gameplay evidence covers Moat, Evil Eye of Orms-by-Gore, and
Akron Legionnaire in `TestRONAttackRuleCards`, `Lifecycle`, `FlyingChanges`,
`Native`, `NativeCast`, `StaticFilterScopes`, and `StrictLoadAndInventory`.
Other `CantAttack` contexts, temporary rules, attacking-player restrictions,
and mixed operation lists remain unsupported.

## RON static Aura control

On an Aura with `EnchantPermanent/1`, `PermanentLayerEffect/2` accepts
`HostPermanent` and a modification list containing `SetController(You)/1`.
`You` must be a bare symbol. This modification can accompany the existing
supported Aura modifications. Each factory creates a separate `ControlAttached`
continuous effect through `WithStaticAbility`. The controller is the Aura's
current controller in layer 2, not a player saved when the spell was cast.

`ThisPermanent`, collection control, chosen/target players, resolving actions,
and other control contexts remain unsupported. Control Magic and Steal Artifact
are covered by `TestRONStaticControl*`; their creature/artifact enchant filters
still govern casting and attachment legality. Collection loading remains strict.

## Static basic land type prerequisite

The existing engine constructors `BecomesBasicLandsEffect(filter, subtypes...)`
and `BecomesBasicLandAttachedEffect(subtypes...)` are available for fixed basic
land types. The collection effect now detects layer-4 selection dependencies;
the shared executor reevaluates remaining dependencies after each application.
The attachment effect requires the host incarnation saved by `Game.Attach`.
Do not construct attachments by setting `AttachedTo` alone.

Each new basic type supplies its intrinsic mana ability. The change removes old
land types and printed rules-text abilities, retains other subtype families and
supertypes, and permits separately granted abilities. Details and test evidence
are in [the engine contract](engine-rules-contracts.md#static-basic-land-types-dependency-and-attachment-identity).

RON supports `PermanentLayerEffect(HostPermanent, [SetLandType(BasicType)])`
on Auras and `EachPermanentLayerEffect(PureFilter, [SetLandType(BasicType)])`.
`BasicType` is a symbol: Plains, Island, Swamp, Mountain or Forest. Host effects
can include the existing P/T and keyword modifications. Collection effects must
contain only the type operation; combined operations need a shared membership
capture contract. Source-dependent collection filters remain rejected.

`IsSupertype(Basic)` and `IsNonSupertype(Basic)` read the basic supertype, not the
land subtype. Both filter forms also support Legendary. These forms use the
existing pure predicate and negation services. Conversion, Evil Presence and
Blood Moon have RON gameplay evidence. `SetLandTypeVariable`, resolving type
changes and entry choices remain unsupported.

## RON conditional static modifications

`If(condition, [layer_rules...])` on a permanent accepts a nonempty body of
`PermanentLayerEffect(subject, modifications)` and
`EachPermanentLayerEffect(filter, modifications)`. Singular subjects use
`ThisPermanent` or, on an Aura, `HostPermanent`. Collections use the existing
static filter adapter, including controller selection and source exclusion.

Conditions accept `PlayerPassesFilter(You, ControlsA(filter))`,
`PlayerPassesFilter(You, ControlsNo(filter))`,
`APlayerPassesFilter(Opponent, ControlsA(filter))`, or
`PermanentPassesFilter(subject, IsUntapped)`. Controller filters accept existing
card-type, subtype, color, supertype, tap/combat-state and `AnyPermanent` filters,
with nonempty recursive `And`/`Or`. Symbol `IsNonToken` is supported in these
static condition filters and lowers to `Not(IsToken)`. They reject ability and
P/T queries. `Opponent` is a player filter here, not a player reference. Only
`ControlsA` is supported on this opponent form.

Modifications accept fixed `AdjustPT(power, toughness)` or nonempty
`AddAbility([keyword...])` with existing basic keywords and supported landwalk
forms. One `If` body must use only layer 6 grants or only layer 7c bonuses.
Nested conditions, other body rules, mixed layers, chosen/event/target references,
counter modifications, other player conditions, type/base-value changes and replacements remain
errors. No resolving `If` action support is implied.

The adapter reuses `PlayerControls`, `OpponentControls`, `ControlsNone`, `PermanentIsUntapped`,
`While`, `Static`, typed source/attachment references, permanent sets and
continuous modifications. Conditions and membership are live at each layer pass.
Factories create separate static rules and modifications without chosen state.
`TestRONConditionalStatic*` covers Kird Ape, Sedge Troll, Giant Tortoise,
Spectral Cloak and Angelic Voices, their reprints and invalid declarations.
`TestRONOpponentStatic*` covers Water Wurm, Beasts of Bogardan and Ivory
Guardians, including token exclusion, current colors, named recipients on both
battlefields, cumulative bonuses, protection and layer ordering.

`mage.OpponentControls(PermanentFilter) StateCondition` is a pure condition for
`While`, `IfState` and other state-condition consumers. It uses the context
controller's current opponent and current unphased battlefield membership.
An absent or unknown controller does not satisfy the condition. The declaration
holds no chosen state and is safe to reuse across factories and cloned games.

## RON static blocking prohibitions

`PermanentRuleEffect(ThisPermanent, [CantBlockAttackers(filter)])` prevents the
source from blocking attackers that match the filter. It lowers to
`SourceCanBlockOnly(Not(filter))`. The source restriction uses exactly one
operation; `HostPermanent`, target and event subjects remain rejected for this
form.

`PermanentRuleEffect(subject, [CantBeBlockedByDefenders(filter)])` prevents
creatures that match the filter from blocking the subject. Its subject is
`ThisPermanent` or the current `HostPermanent` of an Aura. It lowers to the
existing typed static `CantBeBlockedExceptBy(subject, Not(filter))` rule. This
form also uses exactly one operation. Here, `Defenders` means candidate blocking
creatures, regardless of whether they have defender.

Both forms reuse the existing permanent-filter grammar, including nested
`And`/`Or`, card type, creature subtype, color and fixed power comparisons.
`PowerIs(GreaterThanOrEqualTo(Integer(n)))`, `LessThanOrEqualTo` and `EqualTo`
read current power when block legality is checked. Filters do not create target
bindings or choices. All other combat restrictions still apply. Removing the
source ability stops the static rule. Removing a host's abilities does not stop
an Aura's rule. Phasing suspends the rule; attachment changes select the new
host, and a departed host is not followed through its return.

`TestRONBlockingProhibitionCards`, `CurrentCharacteristics`, `Lifecycle`,
`Host`, `NativeCombat`, `DeclaredBlock`, `Invalid`, `Inventory` and
`StrictCollection` (each with prefix `TestRONBlockingProhibition`) cover these
forms. The five complete-card mappings are Ironclaw Orcs, Brassclaw Orcs,
Orcish Veteran, Amrou Kithkin and Bog Rats. Other records with these operations
can still have unsupported rules. Temporary forms are unchanged.

## RON static attack permissions

Permanent rules accept these exact forms:

```ron
PermanentRuleEffect(HostPermanent, [CanAttackAsThoughItDidntHaveDefender])
PermanentRuleEffect(HostPermanent, [CanAttackAsThoughItHadHaste])
```

Each operation is a symbol in a one-element list. The subject is symbol
`ThisPermanent` or, only on an Aura, symbol `HostPermanent`. Target/event reads,
operation arguments, mixed lists and temporary or collection contexts remain
rejected. Both forms use typed current references and fresh static abilities per
factory, through the shared `While` service. The first reuses `AttrCanAttack`;
the second uses `mage.AttackAsThoughHaste` and preserves tap-cost restrictions.
They do not grant haste or remove defender from the host's characteristics.

Animate Wall and Instill Energy now compile in all six reprints each. Instill
Energy's separate zero-mana untap activation uses the existing controller-turn
and once-per-turn checks. No Go definition or production registry is replaced.

### RON temporary combat rules

`CreatePermanentRuleEffectUntil(subject, [operation], UntilEndOfTurn)` accepts
exactly one of these additional operations:

- Symbol `CanAttackAsThoughItDidntHaveDefender`.
- Symbol `CanAttackAsThoughItHadHaste`.
- Tuple `CantBeBlockedByDefenders(permanent_filter)`.

These forms accept symbol `ThisPermanent` on a permanent or a declared
`Ref_TargetPermanent`. They work in resolving activation, spell, mode and trigger
bodies. Aura hosts, event references, multiple operations and other durations
remain unsupported. The existing symbol `CantBeBlocked` keeps its target-only
boundary.

RON builds each typed target and effect in the same factory instance. It uses
`Until(EndOfTurn, ...)` with `GrantKeywordModification(ref, AttrCanAttack)`,
`AttackAsThoughHaste(ref)`, or `CantBeBlockedExceptBy(ref, Not(filter))`.
The engine captures the subject incarnation at resolution. A blocking filter
reads current blocker characteristics at declaration. Each attack permission
leaves other attack restrictions in force. Attack-only haste does not permit
summoning-sick tap costs or grant the haste ability.

Tower of Coireall uses the filtered rule with its creature target and tap cost.
Wall of Wonder uses the source defender permission after its temporary +4/-4.
`TestRONTemporaryCombatRule*` supplies real-record gameplay, native combat,
invalid-declaration, payment, target-copy, lifecycle and context evidence.

### RON static prevention from permanent sources

The top-level permanent rule supports this complete shape:

```ron
PreventDamage(
    DamageWouldBeDealtByAPermanentToRecipient(
        source_filter,
        Permanent(ThisPermanent)
    ),
    [PreventThatDamage]
)
```

`source_filter` uses the existing permanent filter grammar. The recipient is
only the source of the static ability. The operation list must contain exactly
the symbol `PreventThatDamage`. Player recipients, Aura hosts, event or target
references, general damage sources, finite amounts and additional operations
remain unsupported in this shape. Static combat events have the separate
contract below. These declarations
fail compilation before a factory is returned.

Each factory builds a fresh typed static ability using
`PreventDamageFromMatchingPermanents(This, source_filter)`. It prevents all matching damage to that permanent while the ability
is active. It covers combat and noncombat damage without a finite shield.
Current source characteristics and retained source incarnations use the engine
source filter contract. This does not make permanent filters match spell sources.
The rule follows the recipient across control changes and phase cycles; a
returned recipient gets its own new static ability. Removing that ability
removes the prevention. Other instances and cloned games keep separate state.

Tests: `TestRONStaticPreventionCards`, `TestRONStaticPreventionLifecycle`,
`TestRONStaticPreventionPendingActivation`, `TestRONStaticPreventionCombatAndSpell`,
`TestRONStaticPreventionReplacementOrder`, `TestRONStaticPreventionOtherRules`,
`TestRONStaticPreventionNativeLandwalk`, `TestRONStaticPreventionStrictLoadAndInstances`
and `TestRONStaticPreventionInvalid`. These execute the complete Argothian Pixies,
Desert Nomads and Uncle Istvan records, including their other rules.

## RON mixed-recipient damage

In resolving action lists, the existing `SpellDealsDamage(ThisSpell, amount,
recipient)` and `PermanentDealsDamage(ThisPermanent, amount, recipient)` now
accept this recipient shape:

```ron
MultipleRecipients([
    EachPermanent(And([IsCardtype(Creature), DoesntHaveAbility(Flying)])),
    EachPlayer(AnyPlayer)
])
```

Supply exactly one `EachPermanent/1` and one `EachPlayer/1`, in either order.
The permanent operand uses the existing context-free permanent filter grammar;
the player operand must be symbol `AnyPlayer`. Duplicate sets, singular members,
empty lists, player-history filters, and unbound references are rejected during
compilation. This does not add multiple damage amounts or target groups.

The adapter builds `mage.DamageRecipients(mage.EachPlayer,
mage.MatchingPermanents(filter))` and passes it to
`mage.DealDamageEach(mage.This, recipients, amount)`. Each card factory builds
fresh effects. Existing amount forms, action contexts, and source restrictions
remain in force. The engine selects recipients at resolution and deals the
same amount to all of them in one simultaneous batch. No target declaration is
created. Earthquake, Hurricane, and Inferno use this complete shape.

Evidence: `TestRONMixedDamage*` in `pkg/mage/dsl/ron/mixed_damage_test.go`.

## RON simultaneous damage packets

Resolving action lists accept these forms:

```ron
SpellDealsMultipleDamage(ThisSpell, [
    DamageToRecipients(Integer(4), Ref_AnyTarget),
    DamageToRecipients(Integer(2), Player(You))
])
PermanentDealsMultipleDamage(ThisPermanent, [
    DamageToRecipients(Integer(2), Ref_AnyTarget),
    DamageToRecipients(Integer(3), Permanent(ThisPermanent))
])
```

The action has arity two. Its second operand is a nonempty list of
`DamageToRecipients(amount, recipient)/2`. The source must match the enclosing
spell or permanent context. Each amount uses the existing resolution-value
compiler. A recipient is the in-scope symbol `Ref_AnyTarget`, `Player(ref)` with
an existing typed player reference, or `Permanent(ref)` with an existing typed
permanent reference. Collections, numbered target groups and undeclared reads
remain errors. This adds no target declaration or target-choice service.
Spells, supported activations, supported triggers and supported modal action
bodies use the same lowering and retain their existing declaration scopes.

The compiler builds fresh `mage.DamagePacketTo(recipient, amount)` values for
`mage.DealDamagePackets(mage.This, packets...)`. The engine reads all recipients
and positive amounts before damage. Overlapping recipient identities receive
one combined amount before replacement processing. All assignments use one
captured source and one existing simultaneous batch. Nontargeted self damage
still requires the enclosing targeted spell or ability to resolve. The source
as a damage recipient must remain its original, unphased incarnation; as a
damage source it can use retained LKI after departure.

Orcish Artillery, Psionic Entity, Brothers of Fire and Psionic Blast use this
complete shape. Evidence: `TestRONDamagePackets*` and `TestDamagePackets*`.
The general `DamageToRecipients` family remains partial: recipient sets,
opponent-chosen targets, history target filters and rounded/counting values
still require their own adapters and contracts.

## RON draw-step intervening-if triggers

Permanent cards support this exact rule shape:

```ron
TriggerI(AtTheBeginningOfAPlayersDrawStep(players),
    PermanentPassesFilter(ThisPermanent, state), actions)
```

`players` is symbol `AnyPlayer` or `SinglePlayer(You)`. `state` is symbol
`IsTapped` or `IsUntapped`. The body uses the existing action-list compiler and
may read `Trigger_ThatPlayer` as a typed player reference. Each factory creates
its own binding; independent events, copies and cloned games keep their captured
player. `You` in the body uses the resolving ability controller. The controller
filter uses the source controller at detection. Other event players, conditions,
subjects and spell contexts remain declaration errors.

The adapter lowers to `TurnStepBegins(EachDrawStep, binding)` and
`WithInterveningIf(SourceTappedStateWithLKI(state == IsTapped))`. The engine
checks the condition before the normal draw and again at resolution. A departed
source uses the last-known tapped state of its captured incarnation. A returned
card cannot replace that information. A source that phases out after detection
keeps its tapped state for this information read. Phased sources do not trigger.
Normal draws and trigger draws use the shared replacement pipeline.

The complete-card mappings are Howling Mine and Mana Vault, each in all six
reprints. `TestRONDrawStepCards`, `ConditionsAndIdentity`, `VaultPayment`,
`Bindings`, `Invalid`, `InventoryAndReprints`, `BeforeNormalDrawReplacement` and
`VaultDamagePrevention` (all with prefix `TestRONDrawStep`) cover their rules
and interactions. The vocabulary family remains partial: this does not add
other intervening-if events or conditions.

`mage.SourceTappedStateWithLKI(bool) StateCondition` is a source-only condition
for information reads at resolution. At detection, with no captured incarnation,
it requires a live phased-in source. With a captured incarnation, it reads that
object's current tapped state or its departure snapshot. Existing
`PermanentIsTapped`, `PermanentIsUntapped`, `SourceTappedCondition` and
`SourceUntappedCondition` keep their live-only semantics.

## RON permanent information and singular exile

In supported resolving action bodies, `ControllerOfPermanent(ref)` is a typed
player reference. `PowerOfPermanent(ref)` and `ManaValueOfPermanent(ref)` are
resolution values. They read the exact incarnation's current information, or
its saved departure/phase information. `ref` can be the local permanent target,
a captured `Trigger_ThatPermanent`, `ThisPermanent` on a permanent, or
`HostPermanent` on an Aura. An unbound or wrong-role reference fails compilation.
Each factory builds reads against its own declarations; modes and abilities
cannot share a target scope.

`PlayerAction(ControllerOfPermanent(ref), GainLife(value))` and its `LoseLife`
form use the indicated player. Other player contexts retain their existing
support limits. `Exile([Permanent(ref)], [])` supports one permanent with no
modifiers in spell, modal, activation and ordinary trigger bodies. It performs
no action on a departed or phased incarnation. Larger lists, card subjects,
collections, face-down modifiers and linked results remain unsupported.
See [the rules contract](engine-rules-contracts.md#permanent-information-through-typed-references)
for engine APIs, lifecycle behavior, and test evidence.

## RON fixed entry counters

`AsPermanentEnters(ThisPermanent, [EntersWithNumberCounters(Integer(n), counter)])`
accepts one entry operation on a permanent card. The source is the symbol
`ThisPermanent`. The amount is a nonnegative integer literal that fits the
engine's current counter storage (0–255); larger declarations are rejected.
`counter` uses the same validated permanent counter types as `PutCounters`,
including `PTCounter(+1, +0)`, `PTCounter(+1, +1)` and named counters.
The adapter uses `WithAbility(EntersWithNCounters(counter, n))`, with a fresh
ability for each card factory invocation. Counters exist before entry triggers
are detected. Entry is not a stack action; the permanent spell still resolves
through the stack. X, computed entry values, other source references and combined
entry-operation lists remain unsupported. Existing `EntersTapped` support remains.

Triskelion's complete ATQ record is exercised by `TestRONEntryCountersTriskelion`,
`Lifecycle`, `Replacements` and `TargetRecheckAndPrevention` in the RON package.
`TestRONEntryCountersCompleteRecordInventory` exercises only entry clauses from
all 13 records of eight fixed-counter names; it does not establish their remaining
rules. See the [rules contract](engine-rules-contracts.md#ron-fixed-entry-counters)
and [progress ledger](designs/mtgish-ron-progress.md).

## RON player-target life actions

`PlayerAction(Ref_TargetPlayer, GainLife(value))` and
`PlayerAction(Ref_TargetPlayer, LoseLife(value))` use the declared singular
`TargetPlayer(AnyPlayer)` binding. `value` uses the existing resolution-value
compiler, including `Integer(n)` and announced `ValueX`. The target can belong
to a spell, activation, trigger, or selected spell mode. Each factory creates
its own target declaration. `PlayerAction(You, ...)` and bare life actions
still use the action controller, even inside a player-target action list.
Sibling actions do not inherit an explicit player's scope.

Life actions now also honor existing non-event spell-controller references.
Step players and permanent-controller LKI reads retain their established
behavior. Spell-cast event players remain rejected in this slice. Unbound,
incompatible and mode-external references remain declaration errors.
The engine executes life changes through `GainLifeFor` and `LoseLifeFor`,
with its normal replacement and notification services. Life loss does not
use damage prevention. `TestRONPlayerLife*` covers these contracts.

Stream of Life has complete RON gameplay evidence for all six reprints.
Healing Salve and Alabaster Potion have life-clause evidence only; their
complete records remain rejected because finite prevention is unsupported.

## RON opponent targets and spell-cast events

The symbol `Opponent` is supported in `TargetPlayer(Opponent)`,
`TargetPermanent(ControlledByAPlayer(Opponent))`, and
`WhenAPlayerCastsASpell(Opponent, spellFilter)`. Permanent target filters can
combine the controller restriction with other supported filters in nested
`And` lists. These target forms work in spells, activations, triggers, and
selected spell modes. Each factory creates separate typed target declarations.

Targets use the spell or ability controller at announcement and resolution.
An opponent-controlled copy that keeps its controller as its target cannot
resolve. Source control changes or departure do not change a pending ability's
controller. Spell-cast events use `SpellCast().ByOpponent()` to check the source
controller at the event. The existing spell and caster bindings stay separate
for each pending trigger. Tuple-shaped `Opponent()`, player references used as
filters, and controller-dependent target disjunctions remain rejected.
This does not add general opponent player-reference or collection support.
Evidence: `TestRONOpponent*`, including complete Citanul Druid gameplay.

## RON optional tap or untap

`MayAction(TapOrUntapPermanent(ref))` accepts a typed permanent reference in
its current action scope: `ThisPermanent`, Aura `HostPermanent`, declared
`Ref_TargetPermanent`, or bound `Trigger_ThatPermanent`. The nested action must
have exactly one reference. Other `MayAction` shapes and a bare
`TapOrUntapPermanent` remain unsupported.

The compiler lowers this pair to `mage.MayTapOrUntap` with fresh instance
bindings. It works in spell actions, selected spell modes, activations, and
trigger actions. The resolving controller chooses whether to change the
permanent's current tapped state. Target choice and costs occur before this
choice. Decline does not refund costs or stop later instructions. An absent or
illegal subject receives no choice. Normal tap/untap events and replacements
still apply. See `TestRONTapChoice*` for complete Twiddle and Hyperion Blacksmith
RON gameplay and shared scope, copy, clone, event, and rejection evidence.

## RON resolution-time battlefield counts

`TheNumberOfPermanentsOnTheBattlefield(filter)` supplies a resolution value.
The filter accepts the existing source-independent permanent filters. Nested
`And([...])` can include one `ControlledByAPlayer(SinglePlayer(player))` clause.
The player uses the current typed reference scope, including `You`, a declared
`Ref_TargetPlayer`, or an event-bound `Trigger_ThatPlayer`. Controller clauses
inside `Or`, multiple controller clauses, collection players, unbound references,
and damage-loop players remain rejected.

`Twice(value)` multiplies any supported instance resolution value by two.
It composes with counts and subtraction and retains signed intermediate values.
These forms work in the resolving action consumers of `compileInstanceValue`;
they do not add static, CDA, entry, cost, beginning-of-turn, or temporary P/T
value support. Each instruction reads current collection membership. Phased
permanents are absent; tokens and returned permanents can be current members.
Ownership does not determine control.

The adapter uses `CountPermanents`, `MatchingPermanents`,
`PermanentsControlledBy`, and `MultiplyValues`. Typed references keep factory,
copy, event, and target bindings separate. Full-card evidence is Karma
(LEA/LEB/2ED/3ED/CED/CEI) and Jovial Evil (LEG), in `TestRONBattlefieldCount*`.

### RON land mana events

RON supports `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,
IsCardtype(Land)), body)` for non-mana resolving actions. Each factory has its
own typed event bindings. `Trigger_ThatPlayer` saves the activating player;
`Trigger_ThatPermanent` saves the land incarnation. Information reads use the
existing LKI and phase-information contracts. These names do not escape the
trigger body.

Two complete mana-bonus bodies use the shared mana-bonus service so automatic
payment and actual production agree:

```ron
TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer, IsCardtype(Land)),
  ActionList([PlayerAction(Trigger_ThatPlayer,
    AddMana(Trigger_AnyManaTypeProduced))]))
TriggerA(WhenAPermanentIsTappedForMana(SinglePermanent(HostPermanent)),
  ActionList([PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),
    AddMana(ManaProduceG))]))
```

The attached form needs an Aura scope and one concrete production. White, blue,
black, red, green and colorless productions work; `And` with exactly one concrete
production also works. The bonus goes to the player who taps the host for mana.
These bonuses add one mana, resolve immediately, and do not create stack objects.
The produced-type form permits only a type produced by the triggering ability;
it does not copy the amount produced or the types of other bonuses.

Other mana-producing bodies in these events remain declaration errors. Subtype
events, other player filters, temporary triggers, source/target mana events and
attached non-mana bodies remain unsupported. This does not add general
`PlayerAction(..., AddMana(...))` support outside this event scope.

### RON enchanted permanent filter

The bare symbol `IsEnchanted` is a permanent filter. It uses the shared
`mage.IsEnchanted` query: a permanent must have an attached, present Aura.
Equipment alone does not qualify. The filter composes with the supported
`And` and `Or` forms and works in existing permanent target and static damage
source filters. Target legality is checked at announcement and resolution;
prevention reads source information when damage would occur. The query does
not choose or save an attachment. Constructor forms such as `IsEnchanted()`
and `IsEnchanted(ThisPermanent)` are invalid.

RON gameplay tests: `TestRONEnchantedFilter*` cover Ramses Overdark and Wall of
Putrid Flesh. Enchanted Being still requires an unsupported combat-only static
prevention event; this filter does not establish support for that event.

## RON land-entry events

Top-level permanent `TriggerA` rules accept these events:

```ron
WhenAPermanentEntersTheBattlefield(IsCardtype(Land))
WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer, IsLandType(Swamp))
```

The second form accepts the five basic land subtypes. Other filters, player
restrictions, malformed envelopes and tuple-spelled references are rejected.
Both events declare `Trigger_ThatPermanent` only inside their action body. The
actor form also declares `Trigger_ThatPlayer`, the player who caused entry.
An entry without a known actor does not match that form. Setup entry can still
match the first form. Playing a land and putting a land onto the battlefield
both cause entry; phasing does not.

Each factory uses a fresh `PermanentBinding` with
`LandEnters().WithPermanent(binding)`. Actor events also use `OfSubtype` and
`WithActor` with a separate `PlayerBinding`. Each pending event saves its own
land incarnation and actor. A returned land cannot replace an old reference.
`ControllerOfPermanent(Trigger_ThatPermanent)` reads the controller at resolution
through `ControllerOfWithLKI`; it does not use the actor or freeze entry control.
Actions on the land require the original, present incarnation. Copies and clones
retain independent resolution state.

`TestRONLandEntry*` covers Ankh of Mishra's complete six records and both Chant
event prerequisites. The Chants still reject their optional counter-placement
payments; these tests do not establish complete Chant gameplay.

## RON land and artifact graveyard-event adapters

`TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(filter, AnyPlayer), actions)`
accepts `IsCardtype(Land)` or `IsCardtype(Artifact)`. It also accepts
`And([IsCardtype(type), ControlledByAPlayer(SinglePlayer(You))])` in either
order, with exactly one type and one controller restriction. The type and
controller are checked before the permanent leaves the battlefield. A move
from another zone, phasing, regeneration, or an exile destination does not
match. Tokens and the source itself can match.

Inside this trigger, `Trigger_ThatPlayer` is the owner of the destination
graveyard. `ControllerOfDeadPermanent` is the permanent's controller before
it left. Both use separate `PlayerBinding` values created for each factory
instance. Each pending trigger saves its own values. A later return or control
change does not redirect them. These symbols are not available in other event
contexts. Singular source/host selectors, other types, restricted destination
players and `TriggerI` remain unsupported for this event.

The adapter uses `PermanentDies().OfType(type).WithOwner(owner)` and
`WithController(controller)`, with `YouControlled()` for the controller
restriction. Existing action lists, optional mana payments and typed targets
apply. No new engine API is required.

Dingus Egg's six records use `ControllerOfDeadPermanent`, in accord with the
local Oracle text in `data/LEA.json`. Tablet of Epityr uses the existing optional
payment action. Full-card evidence is in `TestRONGraveyardEvent*`, including
strict loading and native casting. This does not establish support for Lich,
Cyclopean Tomb or Urza's Miter.

### RON end-step intervening-if triggers

`TriggerI(AtTheBeginningOfAPlayersEndStep(players), condition, actions)` accepts
`AnyPlayer` or `SinglePlayer(You)`. The actions can read the captured
`Trigger_ThatPlayer`; that binding belongs to this trigger and this factory.
Supported conditions are:

- `NoPermanentsPassFilter(filter, OnTheBattlefield)`, with an existing permanent
  filter. The query includes both players and excludes phased permanents.
- `PermanentPassesFilter(ThisPermanent,
  DealtDamageToAnyPlayerThisTurn(Opponent))`. Damage history belongs to the
  source incarnation. Zero damage and prevented damage do not satisfy it.

The engine checks the condition at the event and at resolution. A false event
condition creates no trigger. A false resolution condition performs no actions.
The pending trigger keeps its controller and event player after a control
change. Source actions cannot affect a returned incarnation. Sacrifice still
requires the trigger controller to control the permanent. History and counters
remain separate between factories and cloned games. Other end-step conditions,
including tapped state, death history and attack history, remain unsupported.
Evidence: `TestRONEndStepIntervening*` covers the complete Pestilence and
Whirling Dervish records, reprints, native casting and engine gameplay.

### RON artifact event adapters

In `TriggerA(event, actions)`, the following event forms create a fresh typed
`Trigger_ThatPermanent` binding per rule and factory:

- `WhenAPermanentBecomesTapped(IsCardtype(Artifact))`.
- `WhenAPermanentBecomesTapped(And([IsCardtype(Artifact), ControlledByAPlayer(Opponent)]))`;
  the two conjunction members may occur in either order.
- `WhenAPlayerActivatesAnAbility(player, And([AbilityOfAPermanent(IsCardtype(Artifact)), DoesntHaveTapSelfInCost]))`,
  where `player` is symbol `AnyPlayer` or `Opponent`.
- The same activation form with `AbilityOfPermanent(HostPermanent)` in an Aura.

The two activation conjunction members may occur in either order. Arity,
source kind, non-tap symbol spelling, duplicates and lexical reads are checked
before factory construction. Other activation filters remain unsupported.
`ControllerOfPermanent(Trigger_ThatPermanent)` uses the saved incarnation's
current controller or departure LKI at resolution. `Trigger_ThatPlayer` is not
introduced by these forms. Haunting Wind, Powerleech and Artifact Possession
use these adapters. The attached tap adapter and land tap adapters remain
available with their existing contracts.


### RON random discard costs

`DiscardACardAtRandom` is a bare cost symbol in `Activated/2` and
`ActivatedWithModifiers/3`, including nested cost `And([costs])` lists.
It creates `mage.DiscardRandomCost(1)` for each factory instance. Tuple forms,
random-number discard costs and optional random payments remain rejected.
The payer is the activation controller; declaring a target does not change it.

The engine locks a random hand selection after deterministic payments. It uses
hand-visit identity and the game's random service, with no hand-choice prompt.
Shared discard replacements and simultaneous batch events apply to the payment.
An unpaid activation changes no live resources or scripted random results.
Copies resolve their effects without paying the original cost again.

Complete-record mappings: Coral Helm and Draconian Cylix in
`TestRONRandomDiscardCosts` and `TestRONRandomDiscardCostNativeLoadAndInstances`.
The latter loads a strict native fixture collection; production cutover remains
pending complete collection support.

### RON land dependence

`PermanentRuleEffect(ThisPermanent, [CantAttackUnlessDefendingPlayer(
PlayerPassesFilter(DefendingPlayer, ControlsA(IsLandType(Island))))])` uses
`Static(CanAttackOnlyIfDefendingPlayerControls(filter))`. In this two-player
engine, the nonactive player is the defender. The rule checks current subtype
and control for attack legality; it does not remove a declared attacker.

`TriggerA(WhenAPlayerControlsNoPermanents(SinglePlayer(You),
IsLandType(Island)), actions)` uses `WhenState(ControlsNone(filter), actions)`.
The condition uses the current source controller. Once triggered, the ability
resolves even if that player acquires a matching permanent. This is a state
trigger, with no intervening-if check and no event-player binding.

These forms accept `IsLandType` with Plains, Island, Swamp, Mountain, Forest or
Desert. The attack rule accepts only the source, one operation, `DefendingPlayer`
and `ControlsA`. The state event accepts only `SinglePlayer(You)`. Actions and
targets use the existing action compiler and fresh factory declarations.
Other filters, chosen players, Aura hosts and inverse attack conditions remain
unsupported in these contexts. The entire collection still fails strict loading.

## RON source attack triggers

`TriggerA(WhenACreatureAttacks(SinglePermanent(ThisPermanent)), actions)`
uses `mage.Attacks(mage.This)`. Replace the event with
`WhenACreatureAttacksAndIsntBlocked` to use `mage.AttacksUnblocked(mage.This)`.
Only the source permanent is supported in these two event contexts.
`Trigger_ThatCreature` is a typed source reference inside the action body. The
engine captures the source incarnation for each pending ability. The reference
is not available in another rule, and it does not select a returned permanent.
`PermanentDealsDamage` accepts this reference as its source in this context.

Temporary layer modifications in these action bodies accept the bare duration
`UntilEndOfCombat` as well as the existing `UntilEndOfTurn` and source-tapped
forms. Combat expiry uses `mage.Until(core.EndOfCombat, modifications...)`.
Other temporary-modification contexts retain their existing duration support.
Host events, event defending-player references, delayed events and non-mana
optional payments need separate adapters. These forms remain rejected.

### RON one-shot delayed step actions

`CreateFutureTrigger(event, ActionList([...]))` accepts bare
`AtTheBeginningOfTheNextEndStep`, bare `AtTheNextEndOfCombat`, and
`AtTheBeginningOfPlayersNextUpkeep(You)`. It lowers to `mage.AtNext` with
`EachEndStep`, `EachEndOfCombat`, or `YourUpkeep`. The nonempty body uses the
existing action compiler and may schedule another supported delayed action.

The body can read existing typed target declarations, source references, saved
number/permanent/player bindings, and announced X. Each scheduling resolution
captures its own frame. The delayed body declares no new targets. A captured
permanent reference keeps its original incarnation and does not repeat the
original target filter. Source departure and ability removal do not cancel a
schedule. Actions still require a present, phased-in subject; information reads
use their existing LKI contracts.

Other events and new target declarations or modes in the delayed body remain
compiler errors. Bound cast-event spell reads use the saved spell frame described
in "Delayed spell-event references" below. Next upkeep accepts only bare `You`; other players need
an explicit scheduling contract. `AtNextEndOfCombatThisTurn` is distinct from
`AtTheNextEndOfCombat`; its bounded schedule is described below. This adapter
does not add casting restrictions, combat events, choices, histories, links,
or finite prevention.

### RON combat-pair triggers

`TriggerA` accepts `WhenACreatureBlocksACreature(subject, filter)` and
`WhenACreatureBecomesBlockedByACreature(subject, filter)` with
`SinglePermanent(ThisPermanent)` or, in an Aura, `SinglePermanent(HostPermanent)`.
The filter is `IsCardtype(Creature)`, `IsCreatureType(subtype)`, or a
one-argument `And` list. A conjunction must contain a creature type condition
or a required creature subtype. It permits one of each, one
`IsNonCreatureType(subtype)`, one color condition, and one fixed toughness
comparison. The color condition is `IsColor(color)` or a nonempty
`Or([IsColor(color), ...])`. Each subtype must pass creature-family validation.
`IsCreatureType` also requires the creature card type. It cannot match a
noncreature with that subtype. `ToughnessIs` accepts `EqualTo(Integer(n))`,
`LessThanOrEqualTo(Integer(n))`, or `GreaterThanOrEqualTo(Integer(n))` and reads
current toughness at detection. Other type, history, numeric, controller and
nested conjunction forms remain unsupported. Duplicate conditions of one kind
are errors.

An event union is exactly `Or([blockingEvent, blockedByEvent])`, in either order.
Both branches must have the same position-free subject and filter declarations.
It lowers to the engine's per-opponent combat-pair event, not two abilities or a
once-per-combat group. Arbitrary unions and unequal branch declarations remain
errors. Inventory walks nested events even in unsupported unions.

The blocking event provides `Trigger_ThatOtherCreature`; the blocked-by event
provides `Trigger_TheBlockingCreature`; the union provides
`Trigger_ThatOtherCreature`. All three pair-event forms also provide
`Trigger_ThatOtherPermanent` as an alias for the same opposing incarnation.
The alias is available in resolving action bodies and supported delayed step
actions. It does not grant support to action contexts that reject event
references, such as temporary combat rules. Each is a typed event binding
available only in its body, including supported delayed step actions. Each factory builds a fresh
binding. The matching engine patterns save the opposing incarnation at detection.
Aura events read the current attachment at detection. Later attachment, color,
subtype, toughness or controller changes do not recheck filters or redirect
the saved object.
Departure of the source does not end pending or delayed abilities. A returned
opponent is a new object; a phased-out opponent cannot be destroyed.

The compiler uses `BlocksCreatureMatching`, `BecomesBlockedByCreatureMatching`,
and `BlocksOrBecomesBlockedByCreatureMatching` with a typed source/host reference,
immutable `PermanentFilter`, and `WithOpponent(binding)`. Previous source-only
engine patterns remain available. Source-only `WhenACreatureBlocks` uses the
adapter described below. Host block events, group events and other
event-reference spellings still require separate adapters.

### RON indefinite type changes

`CreatePermanentLayerEffect(reference, [AddCardtype(type)])` accepts Creature,
Artifact, Enchantment or Land. `CreatePermanentLayerEffect(reference,
[SetLandType(type)])` accepts Plains, Island, Swamp, Mountain or Forest.
Each form requires exactly one operation and a bound singular permanent
reference in a resolving action body. Empty or combined operations, unsupported
card types, nonbasic land types and malformed or unbound references are errors.
Other indefinite modification families remain unsupported.

The compiler builds each subject with the action instance's bindings and uses
`Until(Indefinite, AddCardType(...))` or
`Until(Indefinite, SetSubtypeFamily(..., SubtypeLand, ...))`. Resolution captures
one incarnation. The effect survives cleanup and source departure. Phasing
preserves an installed effect; departure and return do not preserve it. An
absent or illegal target at resolution receives no modification. Basic land
subtype replacement uses the engine's printed-ability removal and intrinsic
mana rules; it preserves other card types, subtype families and supertypes.

Complete RON gameplay evidence covers Ashnod's Transmogrant and Thelonite Monk
in `TestRONIndefiniteType*`. The latter can sacrifice itself as its green creature
payment. Counter replacement on Transmogrant does not stop its next instruction.

## RON attachment target filters

`TargetPermanent(AttachedToAPermanent(hostFilter))` selects an attachment by its
current battlefield host. Combine it with `IsEnchantmentType(Aura)` in `And`
to select Auras. This adapter works in spell, activation and trigger targets.
The host filter accepts existing pure permanent filters and nonempty nested
`And`/`Or` lists. A conjunctive `ControlledByAPlayer(SinglePlayer(You))` restricts
the host to the stack object's controller. It does not restrict the Aura's
controller or owner. The host controller restriction can occur once.

The compiler lowers the host query to `IsAttachedTo` and the controller rule to
`TargetAttachedToYouControl`. The target remains the attachment; the host is
read again at resolution. Reattachment to a qualifying host keeps the target
legal. Source departure does not change the activation controller. Copies use
their own controller. Each card factory builds a separate target declaration.

Opponent/event/target player references, controller rules inside a disjunction,
source exclusions and nested attachment queries are rejected in this host
context. This slice does not add collection attachment filters, movement,
activated modes or replacement operations. Miracle Worker and Savaen Elves have
complete RON gameplay tests in `attachment_target_test.go`.

### RON plural targets and temporary colors

The RON compiler accepts `OneOrMoreTargetPermanents(filter)` as the sole target
group of a spell, mode, activation, or trigger. Its typed collection reference
is `Ref_TargetPermanents`; the singular `Ref_TargetPermanent` cannot read this
group. Existing permanent target filters and options apply to each member.

`CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,
[SetColor(SimpleColorList([colors]))], UntilEndOfTurn)` replaces the colors of
legal targets at resolution. The list can contain White, Blue, Black, Red, and
Green. An empty list makes the targets colorless. Supply one modification.
The compiler rejects delayed-action use of this form until that context has
contract tests. Collection P/T effects retain their existing filter context.

The adapter uses `DeclarePermanentTargets`, `SetEachColors`, and `Until`.
Targets are chosen at announcement. The color effect captures each legal
incarnation at resolution and ends at cleanup. The compiler creates fresh target
bindings for every factory call.

### RON fixed and X permanent target collections

`NumberTargetPermanents(Integer(n), filter)` declares exactly `n` targets.
The integer must be nonnegative. `NumberTargetPermanents(ValueX, filter)`
declares exactly the announced X targets. Both use the sole target group of
`Targeted` and bind the typed collection `Ref_TargetPermanents`. Spell, mode,
activation and trigger bodies share this declaration adapter. X payment remains
an independent cost contract; this adapter does not add activation-X costs.

`TapEachPermanent(Ref_TargetPermanents)` taps the legal group in one engine
operation. `Exile([EachPermanent(Ref_TargetPermanents)], [])` exiles the legal
group in one zone-change batch. The exile form requires one subject and no
modifiers. These actions also accept existing one-or-more target declarations.
They reject singular, unbound and delayed-context collection reads. Existing
filter-based tap and singular exile forms keep their contracts.

The adapters use `DeclarePermanentTargets`, `DeclareXPermanentTargets`,
`TapEach` and `ExileEach`. Zero targets permit resolution; if a nonempty group
has no legal targets, the spell does not resolve. Factory calls and spell copies
keep separate chosen targets and announced values.

## RON conditional fixed mana

The supported context is an unmodified `Activated` rule with the sole cost
`TapPermanent(ThisPermanent)` and the sole action:

```ron
IfElse(
    PlayerPassesFilter(You, And([
        ControlsA(And([IsLandType(Urzas), IsLandType(Mine)])),
        ControlsA(And([IsLandType(Urzas), IsLandType(PowerPlant)])),
    ])),
    [AddMana(And([ManaProduceC, ManaProduceC, ManaProduceC]))],
    [AddMana(ManaProduceC)],
)
```

`ControlsA(filter)` may stand alone or occur in nonempty, nested `And` conditions.
Its permanent filter accepts nonempty `And`/`Or` combinations of `AnyPermanent`,
card types, positive/negative land or creature subtypes, Aura subtype,
positive/negative colors, and positive/negative supertypes. Subtype validation
uses the engine family contract, which permits custom unknown subtypes and
rejects known family mismatches. Tap and combat status filters are rejected
because solver queries precede the tap cost.

Each branch has exactly one `AddMana`, with a concrete mana symbol or a nonempty
`And` of concrete symbols. Repeated colors and multiple fixed colors are valid.
Empty branches, additional actions or costs, modifiers, player choices, dynamic
amounts, bound references and other condition forms remain unsupported.
Factories create fresh `ManaAbility` instances and use the shared engine
`NewConditionalManaAbility` service. General `IfElse` action support is still a
gap. Tests: `TestRONConditionalMana*`; engine contract: `TestConditionalMana*`.

## RON source and attachment filter conditions

`PermanentPassesFilter(subject, filter)` in a static `If` accepts bare
`ThisPermanent`, or bare `HostPermanent` on an Aura. It lowers to
`mage.PermanentMatchesFilter(PermanentRef, PermanentFilter) StateCondition`.
The existing `IsUntapped` form retains `PermanentIsUntapped`.

Static condition filters also accept `HasACounterOfType(counter)` and
`HasNumberCountersOfType(EqualTo(Integer(n)), counter)`, where `n` is a
nonnegative integer literal. They use `mage.HasCounter` and
`mage.CounterCountEquals`. Counter types use the existing permanent-counter
grammar, including supported P/T pairs. Counts above 255 remain integer
comparisons; they do not wrap to the engine's counter storage type.
These filters compose with nonempty `And`/`Or` and also work in the existing
`ControlsA`/`ControlsNo` static player conditions. Dynamic thresholds, other
comparisons, resolving conditions and counter-dependent durations are not added.
Conditional body restrictions remain unchanged, including rejection of
`PermanentRuleEffect` bodies.

`TestRONStaticCounter*` covers the predicate contract and the exact Homarid
and Tidal Influence static P/T clauses. Merseine and Venarian Gold use their
exact parsed predicates with test-only keyword bodies. These four complete
records still reject other unsupported rules; none is a full-card gameplay
coverage claim.

The filter uses the existing static condition grammar: nonempty nested
`And([filters])`/`Or([filters])`, `IsNonToken`, permanent type/subtype,
color/supertype predicates, tapped state, and combat membership. Operands must
have the existing exact shapes. Ability and P/T filters remain rejected so a
conditional grant or bonus cannot read the characteristics it changes. Target,
event, saved and unresolved references remain rejected. No resolving `If` action
or new conditional body shape is added.

The engine checks the live source incarnation and, for attachments, the live
attachment incarnation before applying the filter. It uses the shared battlefield
query service and stores no choices or captured game state. Conditions read the
current attachment and characteristics on each layer pass. Departure or phasing
makes the condition false; a phase cycle restores it if the same objects still
qualify. Control changes alone do not end the condition. No LKI is read.

`TestPermanentMatchesFilter*` covers the engine contract.
`TestRONPermanentFilter*` covers complete Goblin Caves gameplay, Aura legality
and costs, attachment/type changes, source and recipient membership, phasing,
ability removal, independent factory instances and clone isolation. Goblin
Shrine shares the condition but remains rejected for its leave-battlefield
trigger; its complete gameplay is not verified.

## RON unlimited recipient prevention

Resolving actions accept these two complete forms:

```ron
CreatePreventDamageUntil(
    DamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent)),
    [PreventThatDamage], UntilEndOfTurn)
CreatePreventDamageUntil(
    DamageWouldBeDealtByAPermanentToRecipient(
        And([IsCardtype(Creature), HasAbility(Flying)]), Player(You)),
    [PreventThatDamage], UntilEndOfTurn)
```

Recipients reuse typed damage-packet references: `Permanent(reference)`,
`Player(reference)`, or a bound `Ref_AnyTarget`. Existing source, Aura attachment,
event, saved, and singular target scopes apply. Unbound or incompatible reads
are compile errors. Each factory builds references with its own bindings.
These actions work in the existing resolving spell, mode, activation, trigger,
and delayed-action contexts; they add no static or replacement-operation scope.
The operation list must contain exactly one bare `PreventThatDamage`, and the
only supported duration is bare `UntilEndOfTurn`.

Source filters reuse supported type, subtype, supertype, color, keyword, token,
and tapped-state filters with nonempty nested `And`/`Or`. Bare `IsAttacking`
uses `DamageSourceIsAttacking()` so absent sources use saved combat information.
`IsBlocking` and `PowerIs` remain rejected in this context. A zero engine filter
means any source; RON uses the first form to express that choice. The second form
requires a permanent source and does not match a spell source.

The engine API is `PreventDamageThisTurn(DamageRecipientRef, PermanentFilter)`.
It captures the recipient at resolution, preserves permanent incarnation,
rechecks source properties at damage time, and expires at cleanup. It prevents
combat and noncombat damage without consuming a finite shield. It does not
prevent damage dealt by the recipient to other objects or players.

Complete gameplay evidence covers Indestructible Aura, Scarecrow, and
Al-abara's Carpet. The latter two records include their Oracle source
restrictions: flying creatures, and attacking creatures without flying,
respectively. Other finite, chosen-source, redirection, and additional-operation
prevention declarations remain unsupported.

### RON all-player hand and fixed draw actions

Resolving action lists accept `EachPlayerAction(AnyPlayer, DiscardHand)` and
`EachPlayerAction(AnyPlayer, DrawNumberCards(Integer(n)))`, where `n` is a
nonnegative integer. `AnyPlayer` and `DiscardHand` are bare symbols. These forms
use `DiscardEach(EachPlayer)` and `DrawEach(EachPlayer, Fixed(n))`.

Discard selects both current hands at resolution and uses one simultaneous
batch. All discard moves finish before discard events. Draw uses individual
draw actions; the active player completes all draws before the other player.
Both operations use the shared replacement pipeline. Empty hands and zero
draws permit later instructions. Existing spell, mode, activation, trigger and
delayed action lists use the same lowering. The forms create no targets or
loop-local player binding and preserve surrounding target scopes.

Other player filters, action-list bodies, optional discards,
X and player-dependent draw amounts remain compile errors in this context.
Complete Wheel of Fortune gameplay evidence is in `TestRONEachPlayerWheel`,
`TestRONEachPlayerCopies` and `TestRONEachPlayerReplacements`.

### RON all-player library shuffle actions

`EachPlayerAction(AnyPlayer, ShuffleHandAndGraveyardIntoLibrary)` moves both
players' hands and graveyards into their own libraries, then shuffles each
library. Later fixed draw actions remain separate instructions. This uses
`ShuffleHandsAndGraveyardsIntoLibraries(EachPlayer)`.

The adjacent pair `EachPlayerAction(AnyPlayer, ShuffleHandIntoLibrary)` followed
by `EachPlayerAction(AnyPlayer, DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`
uses `ShuffleHandsIntoLibrariesAndDraw(EachPlayer)`. The engine keeps each
player's moved count local to that resolution. It finishes all shuffles before
any draw. This pair creates no readable result outside its continuation.
Missing, nonadjacent, incompatible, or malformed continuations are compile
errors. Other shuffle-result consumers remain unsupported.

Both forms use resolving action lists, including mode and activation bodies.
They create no targets. They shuffle empty libraries and hands, preserve
owners, and exclude battlefield, exile, and stack cards. Movement is not
discard. Zone events see all completed moves; graveyard departures report one
batch per player. Draws use the existing replacement pipeline and active-player
order. The resolving spell moves to the graveyard after its instructions.
Evidence: `TestRONLibraryRecycle*` and engine `TestLibraryRecycle*`.

### RON current-hand operations

`PlayerAction(player, RevealHand)` reveals that player's current hand at
resolution. `RevealHand` must be a bare symbol.
`PlayerAction(player, DiscardCards(filter))` discards all matching cards in one
batch. `TheNumberOfCardsOfTypeInPlayersHand(filter, player)` is a current-hand
resolution value. Player references use the existing typed lexical scope;
these forms create no new binding or target declaration.

Hand filters accept `IsCardtype(type)`, `IsNonCardtype(type)`, `IsColor(color)`,
`IsNonColor(color)` and nonempty nested `And([...])` or `Or([...])` forms. They
read card characteristics, not permanent characteristics. Unbound players,
malformed declarations, battlefield predicates and empty logical filters are
compile errors. The adapters apply in resolving spell, mode, activation,
trigger and delayed action scopes; they do not add cost, entry, static or
replacement-operation forms. The count uses existing resolution-value contexts.

The engine primitives are `RevealHandFor`, `DiscardMatchingHand` and
`CountMatchingHand`. Complete card evidence is in `TestRONHandOperationsCards`,
`TestRONHandOperationsCopiesAndPrevention` and
`TestRONHandOperationsEmptyAndReplacements` for Amnesia and Inquisition.

## RON damage-amount triggers

`TriggerA(WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent)), body)`
uses `DealsDamage(This, amount)`. An Aura can use `HostPermanent` in the same
shape, which uses `DealsDamage(AttachedPermanent, amount)`.
`Trigger_AmountOfDamageDealt` is a bare value symbol in that trigger's resolving
body. Each factory call creates a number binding. The value is the actual damage
after prevention, for combat or noncombat damage to a player or permanent.
Zero and fully prevented damage do not trigger. Each event keeps its own amount.
The body has no damaged-player or damaged-permanent binding in this shape.
Other source collections remain unsupported. Opponent-damage events use the
separate shape below.

The engine captures the event before stack placement. Later source departure,
phasing, control changes, or attachment changes do not change the saved amount.
The trigger keeps its controller; a copied trigger uses its copy controller for
`You`. `TestRONDamageAmountCards`, `BatchAndInstances`, `Combat`, `Casting`,
`AuraTargetLegality`, `Invalid`, and `Inventory` provide the RON evidence.


## RON opponent-damage player triggers

`TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),
Opponent), body)` uses `DealsDamageToOpponent(This, player)`. Each factory
creates a separate player binding. Bare `Trigger_ThatPlayer` reads the damaged
opponent in the event body's resolving scopes. It is not a target. The event
checks the source controller when actual combat or noncombat damage occurs.
Zero or fully prevented damage does not trigger. Later source control changes,
departure, return or phasing cannot redirect the captured player. Pending
triggers, copies and clones use the existing resolution frame services.
This shape supplies no amount or permanent binding. Other source references
and player filters remain unsupported.

`PlayerAction(player, DiscardHand)` requires a bare action symbol and a bound
player reference. It uses `DiscardMatchingHand(player, CardFilter{})` to select
all cards in that player's current hand at resolution. Replacements prepare all
moves before discard events. This adds no bare action, EachPlayerAction, cost,
entry, static or replacement-operation context.

`TestRONDamagePlayerCards`, `Casting`, `Combat`, `Instances`,
`DiscardReplacement`, `HandBatch`, `Invalid` and `Inventory` cover complete
Hypnotic Specter and Nicol Bolas records. `TestRONUpkeepPaymentRealCards`
also covers Nicol Bolas's complete upkeep payment and sacrifice rule.

## RON player poison counters and token damage triggers

`PutCounters([ACounterOfTypeOnPlayer(PoisonCounter, player)])` places one counter.
`NumberCountersOfTypeOnPlayer(Integer(n), PoisonCounter, player)` places a fixed,
nonnegative amount, including zero. The list must contain one instruction.
These resolving actions accept existing typed player references, including
`You`, declared player targets and event-local `Trigger_ThatPlayer`. Other player
counter types, dynamic amounts, costs and entry operations remain unsupported.
They use `mage.AddPoisonFor(subject PlayerRef, amount ResolutionValue)`; the
engine resolves the player and amount in the current frame and adds positive
amounts through the existing player poison state. Nil or unbound declarations
fail definition validation. Absent references and nonpositive amounts do nothing.
Poison placement is separate from damage and life loss.

`TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),
AnyPlayer), body)` extends the existing source/opponent form. It uses
`DealsDamageToPlayer(This, binding)`, captures the actual damaged player, and
permits damage to the source controller. It supplies no amount or permanent
binding. Other source and player filters remain unsupported.

The ability list of `TokenWithPT/6` and `NamedTokenWithPT/7` can contain this
source damage-player `TriggerA/2` with bare `AnyPlayer` or `Opponent`, alongside
existing keywords. The token trigger body uses existing resolving action forms
in a fresh permanent scope. It cannot read targets, X, payment results or event
bindings from the creating ability. Each card factory builds a fresh immutable
token declaration with fresh trigger bindings; `TokenAbilities` gives created
tokens separate runtime ability identities. Token creation and copying of the
creating stack ability retain their existing controller and batch contracts.

Evidence: `TestAddPoisonFor*`, `TestRONPoisonCards`, `CastingAndCombat`,
`GeneratorPaymentAndCopies`, `FactoryInstances`, `TokenScope`, `PlacementAndLoss`,
`Invalid` and `Inventory` in their poison test files. The complete RON card
mappings are Marsh Viper, Pit Scorpion and Serpent Generator.

## RON casting windows

Nonmodal instant and sorcery records can declare one
`CastEffect(CantBeCastUnless(condition))` beside `SpellActions`. Rule order does
not change the restriction. Conditions support nonempty nested `And` lists,
`IsAPlayersTurn(You|Opponent)` and `IsPlayersTurn(You|Opponent)`, and bare
`IsDuringUpkeep`, `IsDuringDeclareAttackersStep`, `IsDuringDeclareBlockersStep`
and `IsAfterUpkeep`. The last form checks whether the first upkeep in this turn has ended.
It permits later upkeep steps (CR 503.2); it does not infer history from a step
number. A skipped upkeep does not supply a completed upkeep. Turn references use the caster, not the card owner.

Lowering uses `WithCastingCondition(StateCondition)` with `IsDuringSteps`,
`AfterFirstUpkeep`, `IsYourTurn`, `NotCondition` and `AllConditions`. The condition is checked before
casting choices and payment on each casting path. It grants no priority or
sorcery timing permission. Copying a spell on the stack and resolving it do not
repeat this check. Casting a copy of a card does repeat it. Modal spells,
permanent cards, other CastEffect operations, other predicates and duplicate
CastEffect declarations remain errors.

Combat conditions also support bare `IsDuringCombat`, `IsBeforeAttackersDeclared`,
`IsBeforeBlockersDeclared`, and `IsBeforeCombatDamageStep`.
The before conditions use the first combat of the turn. A nonempty nested `And`
that includes `IsDuringCombat` uses the current combat for its before conditions,
regardless of operand order. This permits the window in an extra combat phase
(CR 506.7c/d). `IsAfterCombat` remains unsupported.
Skipped blockers/damage close their windows when the attackers step ends;
a skipped combat phase closes them when the precombat main phase ends.
First-strike damage closes the combat-damage window. Tuple forms remain errors.

`ActivateOnlyIf(IsBeforeCombatDamageStep)` uses the same first-combat history
through the existing activation gate. It does not grant priority or remove other
activation restrictions. Angus Mackenzie uses this form with its mana/tap cost
and existing end-of-turn combat prevention. Activation turn and combat window forms are documented in
"RON activation turn and combat windows" below.

Resolving `UntapEachPermanent(filter)` uses `UntapEach` and the existing typed
collection services. Filters support the existing pure permanent filters and
controller restrictions expressed as
`ControlledByAPlayer(SinglePlayer(player-reference))` inside `And`. One
controller restriction is allowed. Selection occurs at resolution and excludes
phased permanents. Declared player targets and event players use their own
instance bindings. Chosen-payment and plural-target collections remain errors
in this action context. Reset combines this adapter with its casting window;
Teleport combines its window with the existing temporary blocking restriction.

### Numeric RON permanent targets

Use `PowerAndToughnessIsEqualTo(PT(p,t))` for exact current P/T.
Use `ToughnessIs(EqualTo(Integer(n)))`, `LessThanOrEqualTo`, or
`GreaterThanOrEqualTo` for a fixed current toughness comparison.
In permanent target declarations, use
`ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent)))` for a strict source
power comparison. The compiler requires a permanent source scope and composes
this restriction with `And` filters and controller restrictions. The engine
checks it at announcement and resolution with retained source information.
The Go API is `TargetToughnessLessThanSourcePower()` in
`DeclarePermanentTarget` or `DeclarePermanentTargets`.

## RON win/loss coin branches

Resolving actions support `FlipACoin_OnLose([actions])` and
`FlipACoin_OnWinAndLose([win-actions], [loss-actions])`. Each supplied list must
contain at least one action. The compiler checks both branches, including
unbound references and adjacent optional-payment continuations. Branches use
the surrounding lexical target, event, X and payment scopes. They cannot declare
targets or export payment results. Nested coin branches are supported.

Lowering uses `FlipCoin(You, OnWin(Sequence(...)), OnLoss(Sequence(...)))`,
omitting OnWin for the loss-only form. One flip occurs at resolution, and only
the matching branch runs. An omitted win branch does nothing. You is the
resolving controller, including on a copied ability. Independent instances use
their own target declarations; the engine owns random outcomes and clone state.

These forms work in supported spell/mode, activation, trigger and delayed
resolving action lists. Delayed bodies retain captured target incarnations and
do not target again. No cost, static, entry, replacement or PlayerAction body
context is added. Bare heads/tails `FlipACoin`, result predicates, repeat
procedures and additional combat actions remain unsupported.

Complete-card evidence: `TestRONCoinBottle`, `BottleDamageAndLKI`, `Captain`,
`Kites`, `CastingAndInstances`, `NestedSequencing`, and `InvalidAndInventory`.
The three mappings are Bottle of Suleiman, Goblin Kites and Orcish Captain.


## RON source block events

`TriggerA(WhenACreatureBlocks(SinglePermanent(ThisPermanent)), body)` uses
`mage.Blocks(mage.This)`. The body is a supported resolving action list or
single target declaration with actions. `Trigger_ThatCreature` denotes the
captured source incarnation in this body and its supported delayed actions.
The binding cannot escape to another ability. Host subjects and filtered
collections remain unsupported. The event has exactly one positional argument;
its selector has exactly one symbol reference.

The event triggers once when the source blocks, including when it blocks
multiple attackers. Entering the battlefield blocking does not trigger it.
The shared engine selects final legal block assignments and preserves pending
trigger controllers, source identity, copies and clone isolation.

`IsntEnchanted` is a zero-argument symbol permanent filter. It lowers to
`mage.Not(mage.IsEnchanted)` and reads current Aura attachments. A phased Aura
does not count. Targets use this filter at announcement and resolution;
equipment does not count as an Aura. Tuple spellings remain invalid.

`TestRONSourceBlock*` maps the complete Time Elemental record to native combat,
next-end-of-combat sacrifice and damage, casting, activation payments, target
rechecks, source LKI, identity, phasing, copies, clones and factory isolation.
Other block consumers retain their separate unsupported actions.

## RON static and indefinite keyword removal

`EachPermanentLayerEffect(IsCardtype(Creature), [LosesAbility(keyword)])`
removes one fixed keyword from all current creatures in layer 6. The list must
contain exactly one modification. Fixed keyword symbols and fixed
`Landwalk(IsLandType(...))` declarations use the existing keyword validator.
Other selectors, Aura hosts, `AnyLandwalk`, chosen abilities and combined static
modifications remain unsupported in this context.

`CreatePermanentLayerEffect(reference, [LosesAbility(keyword)])` installs an
indefinite layer-6 removal on the permanent selected at resolution. This action
uses the existing source, host, target and event reference scopes. It
accepts the fixed keywords and `AnyLandwalk` supported by temporary removal.
The list must contain exactly one modification. References must be bound in the
current scope. An absent or phased subject receives no effect. A returned card
cannot inherit the removal; a phase cycle retains an installed removal.

Engine API: `mage.RemoveEachKeyword(mage.PermanentSet, core.Attr)` returns a
`ContinuousModification`. `Until` captures the set at resolution; `While` reads
current membership at each layer application. Both reuse keyword removal and
timestamp order. A later grant can restore the keyword.

Evidence: `TestRemoveEachKeyword*` and `TestRONLastingKeyword*`. The complete
Gravity Sphere and Elder Land Wurm records have native RON gameplay tests.

## RON batch return to hand

`PutEachPermanentIntoItsOwnersHand(filter)`, `DestroyEachPermanent(filter)` and
`DestroyEachPermanentNoRegen(filter)` accept contextual battlefield collection
predicates. Nonempty `And([filters])` and `Or([filters])` nest freely and keep
player restrictions local to each operand. Repeated or overlapping predicates
select each incarnation once. Characteristic leaves use existing permanent
filters. `OwnedByAPlayer(SinglePlayer(player-reference))` and
`ControlledByAPlayer(SinglePlayer(player-reference))` use typed player references
from the current resolving scope. `ControlledByAPlayer(Opponent)` selects the
resolving controller's opponents. `AttachedToAPermanent(host-filter)` uses the
same collection grammar for hosts, including nested ownership/control predicates.
It checks the current attachment and the exact host incarnation.

Collections read current membership at each instruction. Absent or phased
objects do not qualify; an untargeted returned card can qualify as a new object.
LKI does not restore collection membership. Selection does not target members:
shroud does not protect them. Return and destruction use the existing simultaneous
batches, replacement pipeline, indestructibility and regeneration contracts.
Remove Enchantments returns its owned collection before selecting the destruction
collection. No collection or chosen state is saved in the compiled factory.

For permanent targets, `OwnedByAPlayer(SinglePlayer(You))` lowers to
`TargetYouOwn()` and can be combined with existing conjunction restrictions.
Target ownership and control are checked at announcement and resolution against
the stack object's controller. Other target owner references and contextual
ownership in target disjunctions remain unsupported. Existing attachment target
syntax retains its separate contract.

Engine primitives are `UnionPermanents(sets ...PermanentSet) PermanentSet`,
`IntersectPermanents(sets ...PermanentSet) PermanentSet`,
`AttachmentsOn(hosts PermanentSet) PermanentSet`, and
`PermanentsControlledByPlayers(players PlayerSet, filter PermanentFilter)
PermanentSet`. Union/intersection copy their input slices and emit live members
once in battlefield order. Empty compositions, nil children and unbound nested
references fail definition validation. Existing RON scopes construct bindings
separately for each factory. `TestPermanentSet*`, `TestRONContextualCollection*`
and `TestRONReturnCollection*` supply contract and gameplay evidence.

## RON combat removal and temporary blocking prohibition

Resolving actions `RemoveCreatureFromCombat(ref)/1` and
`RemoveCreatureFromCombatAndUnblockBlockers(ref)/1` use existing scoped typed
permanent references. They are available in supported spell, mode, activation
and trigger action lists, including coin branches. They declare no targets.
Wrong arity, unbound and incompatible references, and cost/static contexts are
rejected. The ordinary action lowers to `RemovePermanentFromCombat`. The second
lowers to `RemovePermanentFromCombatAndUnblock` and explicitly makes qualifying
attackers unblocked. It checks both current blocking membership and the set of
blocker incarnations recorded for that attacker during this combat. Removing a
different blocker does not make the remaining blocker historically sole.

Both removal actions accept the same typed references in delayed action lists.
An explicit `ThisPermanent` subject declares the saved source incarnation to
`Game.WaitingDelayedPermanentSources`. Queries do not resolve the subject or
change combat. Departure and phasing retain the saved reference until detection;
a returned object cannot replace it. An implicit attachment and a future delay
body do not infer source reads. Saved target bindings keep their separate frame
references. `TestRONWaitingDelayedCombatExit*` covers delayed execution,
identities, history, copies, clones, scope rejection and protection consumers.

A returned card cannot replace an old blocker. Combat reset clears the history;
game clones keep independent history maps. No attack-unblocked trigger is fired
by removal after blocker declaration.

`CreatePermanentRuleEffectUntil(ref, [CantBlock], UntilEndOfTurn)/3` accepts the
symbol operation, one scoped source or singular target reference, and the symbol
duration. It lowers to `Until(EndOfTurn, CantBlock(ref))`. This rules restriction
uses the shared blocker-filter service, survives recipient ability removal and
phase cycles, and applies in later combats this turn. It keeps the captured
incarnation and expires at cleanup. It does not remove an already declared
block. Other operation lists and durations retain their current support limits.

Complete records for Mijae Djinn (ARN and 3ED) and Ydwen Efreet (ARN) use these
forms with existing coin and source-combat triggers. Evidence:
`TestRONCombatExitNative`, `PendingIdentity`, `Vigilance`, `TargetContexts`,
`Invalid` and `InventoryAndReprints`. False Orders, Imprison and Disharmony gain
shared prerequisites but retain other compiler gaps; no full-card support is
claimed for those records.

## RON announced X activation payments

`Activated/2` and `ActivatedWithModifiers/3` accept the cost
`PayManaX(manaSymbols, ValueX)/2`. The first operand is a mana list with at
least one bare `ManaCostX`; fixed generic and colored symbols can share the
list. The second operand must be bare `ValueX`. Multiple X symbols and X
costs use the same announced value. Non-X `PayMana/1` costs can compose with
these costs through `And([costs])/1`.

The activation declares X for its action and target scope. A
`NumberTargetPermanents(ValueX, filter)/2` target requires that declaration.
`UntapEachPermanent(Ref_TargetPermanents)/1` reads that instance's typed target
collection and lowers to `mage.UntapEach`. Existing filter-based untap forms
keep their behavior. Undeclared, malformed, singular and out-of-scope collection
reads fail compilation.

Payments lower to `mage.ManaCostPayment`. The shared transaction substitutes
announced X before applying an activation cost reduction, then plans and pays
mana and nonmana costs atomically. The stack object retains X even if another
action changes the game's announcement state. Reductions do not change the
number of X targets or the effect's X value. Optional resolution-time payments,
defined-X modifiers, and rounded-half values still require separate adapters.

Evidence: `TestRONXActivation*` and `TestXActivationPayment*`. The complete
Candelabra of Tawnos record has native casting, activation and resolution tests.

## RON recipient-relative player damage

`SpellDealsDamage(ThisSpell, amount, Player(player-reference))` and
`PermanentDealsDamage(ThisPermanent, amount, Player(player-reference))` bind
bare `DealsDamage_ThatPlayer` within `amount` to that typed player reference.
Existing player references and resolution values retain their contracts.
The binding is unavailable to the recipient declaration, other actions,
other modes, delayed bodies, and permanent or mixed recipient amounts.

The same damage actions accept `EachPlayer(AnyPlayer)` or
`EachPlayer(Opponent)`. Each factory creates its own amount-member binding.
`DealsDamage_ThatPlayer` reads that member during amount evaluation. The engine
selects players in turn order, evaluates all amounts before any damage, and
uses one simultaneous batch. Opponents are relative to the resolving controller,
including copied spells. Nonpositive amounts deal no damage. Prevention and
redirection operate through the existing replacement pipeline; changing a
recipient there does not recalculate the original amount.

Malformed player collections and unbound reads are compile errors. General
player filters, permanent-recipient bindings and mixed collections remain
unsupported. Discard result values use the per-player contract below.
`TestRONDamageRecipientValue*` covers both
complete Legends records, Storm Seeker and Typhoon, plus declaration, scope,
instance, activation, copy, clone, payment, timing and simultaneous contracts.
Mind Bomb uses the per-player discard result contract below.

## Simultaneous allocation for finite recipient shields

The existing `mage.PreventNextDamage(recipient, amount)` and
`Game.AddPreventionShield(recipientID, amount)` APIs now support affected-player
allocation among simultaneous damage sources. `DealSimultaneousDamage` and
native combat steps retain all assignments until replacement choices finish.
The affected player or permanent's current controller uses `ChooseNumber` with
legal bounds to allocate limited shield capacity. Replacement ordering uses
`ChooseMode`, including a choice that applies to a later packet. Source
incarnations, shield consumption, cleanup and game clones use the existing
engine services. Single-damage callers keep their current signatures.

This is an engine prerequisite. The RON compiler still rejects finite
`CreateFuturePreventDamage` declarations. No new RON shape or card factory is
accepted. See the finite-shield contract and test mappings in
[engine-rules-contracts.md](engine-rules-contracts.md).

## Source identity for incoming damage to an attacking band

The optional engine player service `mage.BandingDamageAssigner` supplies a
separate assignment for each blocker. `GetBandingDamageAssignment` receives the
game, blocking source, eligible band members and exact source amount. The result
must assign that amount among those members. Nil or invalid results use bounded
`Player.ChooseNumber` choices. This service has no RON spelling.

Existing `BandingDamageDistributor` callers can still supply one map of
recipient totals. The engine preserves valid totals while assigning each
blocking source separately. Sources keep their own identities and abilities
through finite prevention, redirection, actual damage and cloned games. Losing
banding restores the blocking controller's assignment choice. Blocker links are
recorded for every band member so a departed member does not erase other links.

Only incoming damage to ordinary attacking bands has this new contract.
Outgoing band aggregation remains a migration prerequisite. See
[engine-rules-contracts.md](engine-rules-contracts.md) for signatures and tests.
No additional RON declaration is accepted.

## RON finite recipient prevention

The resolving action
`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(amount, recipient), [PreventThatDamage])`
now lowers to `mage.PreventNextDamage`. `amount` is `Integer(n)` with a
nonnegative literal, or bound bare `ValueX`. Zero installs no effective shield.
`recipient` uses existing typed `Permanent(reference)`, `Player(reference)`, or
bound bare `Ref_AnyTarget`. Targets must be declared in the enclosing action
scope. Spell modes keep separate target bindings. Resolving activations,
triggers and delayed bodies use the same adapter and their existing scopes.

Resolution captures the recipient and reads the amount once. The engine owns
shield consumption, incarnation checks, phasing, simultaneous allocation,
replacement choices, game cloning and cleanup. Unbound or incompatible
references, wrong arity, extra operations, source-specific events, and negative
literal amounts remain compile errors. This action grants no activated modes,
player permissions or additional prevention operations. Guardian Angel and
Balm of Restoration still fail complete-record compilation.

Evidence: `TestRONFinitePrevention*` uses complete records for ten cards and
synthetic scope declarations. The outgoing attacking-band source assignment
limitation remains recorded in the engine contract; these adapters do not
establish that separate combat behavior.

## RON single-card library searches

The resolving action compiler accepts these complete sequences:

```ron
SearchLibrary([FindAGenericCard, PutFoundCardsIntoHand, Shuffle])
SearchLibrary([FindACardOfType(And([IsSupertype(Basic), IsCardtype(Land)])),
               PutFoundCardsOntoBattlefield([EntersNormally]), Shuffle])
```

The basic/land conjunction permits either operand order. No other filter,
number of found cards, destination, reveal instruction, entry modifier, or
instruction order is enabled. Symbols must be bare symbols. Declarations fail
at compilation for malformed or unsupported sequences. Searches are resolving
actions in the existing spell, activation, trigger and delayed action scopes;
they are not costs, static rules, entry operations or explicit player actions.
They use the resolving object's controller and do not produce a result binding.

Lowering reuses `SearchLibraryToHand` and `SearchLibraryToBattlefield`. The engine
chooses at resolution from that controller's current library. An unqualified
search must find a card if one exists; a search for a basic land may find none.
The library is shuffled even if nothing is found. Hand movement and normal
battlefield entry precede the shuffle. Normal entry restrictions and counter
replacements still apply. A prohibited entry leaves the card in the library.
Neither shape reveals a card moved to hand. Complete Demonic Tutor and Untamed
Wilds records have gameplay tests in `TestRONLibrarySearchCards` and
`TestRONLibrarySearchCopiesAndEntry`; declaration, inventory, source-controller
and entry contracts have separate tests in the same file.

## Source identity for outgoing damage from an attacking band

Ordinary attacking bands now assign outgoing damage separately for each member.
The existing `CombatDamageAssigner` receives the member and its own amount.
When an ordinary banding blocker changes the chooser, `BandingDamageAssigner`
or `BandingDamageDistributor` receives each attacking source separately.
Source trample permits an unassigned remainder for the defender only after the
complete band's assignments supply lethal damage to all blockers. Other members'
assigned damage and deathtouch can supply lethal damage; they cannot grant trample.
Invalid maps use a legal default plan. Caller maps remain unchanged.

Every packet keeps its own incarnation and damage information through prevention,
redirection, source departure, actual damage histories, lifelink and game cloning.
`TestRONBandOutgoingDamage` maps complete Benalish Hero, Timber Wolves, Mesa
Pegasus, War Elephant, Icatian Phalanx, War Mammoth and Amulet of Kroog records to
this native combat interaction. This resolves the outgoing aggregation prerequisite
listed above. No additional RON declaration is accepted. See
[engine-rules-contracts.md](engine-rules-contracts.md) for the exact context,
validation rules and engine tests.

## Normal nontrample combat division

The existing `mage.CombatDamageAssigner.GetCombatDamageAssignment` also permits
free division for a normal blocked attacker without trample. With multiple
eligible blockers, return nonnegative amounts for their IDs totaling the exact
source amount. No lethal order applies (CR 510.1c). The engine does not call
`GetBlockerOrder` for this context. Nil or invalid maps use the legal default;
one blocker receives all damage without a choice. Zero and negative sources
make no assignment. No remaining blocker means no player damage.

Combat membership keeps the original incarnations. Separate captured source
packets enter the shared simultaneous damage service, including finite shield
allocation and actual lifelink. Existing player implementations keep their
signatures. The trample and banding contexts keep their existing behavior.
No RON shape is added. `TestRONNontrampleAssignment` supplies native combat
interaction evidence for six complete RON creature records with Wall of Air
and Amulet of Kroog. See the normal nontrample contract in
[engine-rules-contracts.md](engine-rules-contracts.md) for exact test mappings
and remaining contexts.

## RON choose-one activated modes

`Activated(cost, Modal_ChooseOne([mode, mode, ...]))` and
`ActivatedWithModifiers(cost, Modal_ChooseOne([mode, mode, ...]), modifiers)`
accept at least two bodies with the supported `ActionList` or `Targeted` grammar.
All bodies must compile. Each mode has a separate target namespace and fresh
factory bindings. Activation costs and supported modifiers apply to the ability.
Payment-result bindings are available in each mode; target declarations stay
local to their mode. Nested modes and triggered modes remain unsupported.

The adapter uses `NewModalActivated` and the existing activation transaction.
The controller chooses a legal mode and its targets before payment. Only that
mode resolves. Copies keep the selected mode and can select new targets for it.
Source sacrifice does not stop resolution. Target legality and incarnation
checks still apply. Finite shields start at resolution and expire at cleanup.
Balm of Restoration has RON gameplay evidence for both modes, casting, costs,
rollback, copies, clones, recipient identity and simultaneous damage.

## RON resolution-time variable mana payments

`MayCost(PayManaX(mana, amount))` and
`PlayerMayCost(player, PayManaX(mana, amount))` require an adjacent nonempty
`If` or `Unless` body testing bare `CostWasPaid` or `CostWasntPaid`. `mana` is
an existing mana-symbol list containing at least one X. Fixed colored/generic
symbols and multiple X symbols retain their normal payment requirements.
`amount` is `Integer(n)` with n >= 0, a bound bare `ValueX`, or
`ManaValueOfSpell(typed-spell-reference)`. Spell references can be the current
scope's `Ref_TargetSpell` or cast event's `Trigger_ThatSpell`. They are built
with the current factory's target/event declarations. Other values, missing
continuations and unbound/incompatible references reject compilation.

These resolving forms work in supported spell, activation, trigger and delayed
lists; bound delayed cast-event spell reads use the saved frame. They do not
announce X or add a cost to casting/activation. The engine reads the amount once at the
payment action, then uses `IfManaPaid`/`UnlessManaPaid` and the shared cost
transaction. An unavailable cost offers no choice. A zero cost still requires
acceptance. Each adjacent pair keeps its own outcome. Activation-cost reductions
do not apply. RON `ManaValueOfSpell` uses `ManaValueOfSpellWithLKI` in the engine.
It includes the referenced spell's announced X,
uses current information while it is on the stack, and retains its departure
mana value by stack object ID. A recast card does not replace this information.

`TestRONResolutionMana*` covers the complete In the Eye of Chaos record,
explicit casters, source lifetime, spell departure, payments, copies, clones,
multiple targets, declaration errors and inventory roles. Power Sink still
requires its unpaid `PlayerActions` body and is not a complete supported card.

### Player-scoped mana loss and tap collections

`LoseUnspentManaFor(player PlayerRef) ResolvingEffect` removes all mana from that
player's pool, including restricted mana. It preserves life, mana-production
history and other players' pools. Supply a typed player reference; definition
validation rejects missing targets and bindings.

`HasManaAbility PermanentFilter` checks current runtime abilities with the shared
activated and triggered mana classifiers. Tapped state, payment availability and
current production do not change the classification. Ability removal and grants
do change the result.

`ControllerOfSpellWithLKI(spell StackObjectRef) PlayerRef` reads the live spell's
controller or the controller saved at departure. It uses the stack object's
identity, so a later cast of the same card cannot replace that information.
`ControllerOfSpell` remains a live-only query.

RON resolving `PlayerActions(player, [actions])` captures its player once with
`SnapshotPlayer`. `ActionPlayer` is available only in that body. Nested bodies
have separate bindings; a sibling action keeps its prior scope. This slice
supports nested `PlayerActions`, explicit `PlayerAction`, `TapEachPermanent`,
`LoseUnspentMana`, and the existing draw, life, discard and hand-reveal player
actions. Other body actions remain compile errors. Spells, activations, triggers
and delayed resolving lists use the same adapter; costs, static declarations
and entry operations do not.

`TapEachPermanent(filter)` accepts one `ControlledByAPlayer(SinglePlayer(ref))`
restriction, directly or within nested `And` filters, through the existing
controlled collection service. Controller restrictions inside `Or`, duplicate
controller restrictions and unbound player reads remain errors.
`HasAbility(ManaAbility)` and `DoesntHaveAbility(ManaAbility)` use the current
mana-ability filter. `LoseUnspentMana` must be a bare symbol, either in a player
body, an explicit `PlayerAction`, or a resolving list for its controller.

Mana Short and Power Sink use these shared forms. Their complete RON records
are tested for all six reprints by `TestRONManaDenial*`. Compilation and complete
collection gameplay remain separate measurements.

## RON live static combat prevention

Top-level `PreventDamage(event, [PreventThatDamage])` on permanents accepts:

- `CombatDamageWouldBeDealtByCreature(reference)`.
- `CombatDamageWouldBeDealtToRecipient(Permanent(reference))`.
- `CombatDamageWouldBeDealtByACreatureToRecipient(filter, Permanent(reference))`.
- `Or([CombatDamageWouldBeDealtToRecipient(Permanent(reference)),
  CombatDamageWouldBeDealtByCreature(reference)])`, in either order, with
  exactly the same reference in both events.

References are `ThisPermanent` and, on an Aura, `HostPermanent`. Source filters accept `And`/`Or`, type/subtype,
color and supertype filters, `AnyPermanent`, `IsTapped`, `IsUntapped`,
`IsEnchanted`, `IsntEnchanted` and `IsAttacking`. The filtered source event
also requires a creature. `IsBlocking` and other relational/numeric source
filters remain declaration errors. Enchanted and attacking sources use retained
incarnation information after departure or phasing; live sources use current
state.
Target/event references, player recipients, mismatched unions, finite amounts,
and additional operations fail compilation. Resolving prevention retains its
separate duration contract.

These adapters use `Static(PreventCombatDamageFrom(PermanentRef))` and
`Static(PreventCombatDamageTo(PermanentRef, PermanentFilter))`. They read the
current subject for every proposed event and prevent repeated combat packets.
They do not prevent noncombat damage. Source phasing or departure suspends the
rule; changing an Aura's host redirects its rule. No subject is saved at entry.
The engine replacement pipeline owns ordering, rechecks, and unpreventable
packets. `PermanentRuleEffect(reference, [CantAttack])` uses
`Static(CantAttack(PermanentRef))` with the same reference restrictions. This
attack rule remains separate from the host's abilities and does not remove a
creature already in combat.

Evidence: `TestStaticCombatPrevention*` and `TestRONStaticCombatPrevention*`.
Complete RON records: Demonic Torment, Gaseous Form, Enchanted Being.


## RON controller spell-cost reductions

Permanent rules accept `PlayerEffect(You,
[DecreaseSpellCost(filter, [CostReduceGeneric(n)])])`. The amount is one
nonnegative integer literal. Filters accept bare `AnySpell`,
`IsCardtype(Creature|Artifact|Enchantment|Instant|Sorcery)`, and nonempty nested
`And([filters])` or `Or([filters])`. A union applies its reduction once even
when a spell has both matching types. Other player selectors, operations,
reduction kinds, values and contexts remain declaration errors.

The compiler uses `ReduceSpellCostStatic`, `FixedAmount`, `SpellAny`,
`SpellHasType`, `SpellsAnd` and `SpellsOr`. Each factory builds a fresh static
ability and effect. The current controller receives the reduction while the
source is present and has its ability. Phasing makes the source inactive.
Different sources add their reductions. Only generic mana is reduced, including
announced X and additional mana costs. Colored mana and nonmana costs remain
required. Cost increases precede reductions, and the final cost cannot be
negative. The existing payment transaction locks the total before payment.

Real RON tests: `TestRONSpellCostReductionCards`, `Casting`, `Lifecycle`,
`Totals`, `StackingAndColoredCosts`, `Shapes`, `StrictRecords`,
`LockedBeforeSourcePayment`, `Invalid` and `Inventory`
(all names have the `TestRONSpellCostReduction` prefix).

## Fixed-count RON sacrifice costs

`SacrificeNumberPermanents(Integer(n), filter)` accepts a positive integer literal
and the existing sacrifice filter vocabulary, including nonempty nested `And`
and `Or`. It works in activation costs (including `And` cost groups) and in
`MayCost`/`PlayerMayCost` paired with an adjacent `If`/`Unless` payment result.
Dynamic counts, empty logical filters, and action/static contexts remain errors.

The adapter uses `SacrificeNMatchingIncludingSourceCost(n, filter, text)`.
Matching permanents must belong to the payer's current control and be phased in.
The source is eligible when it matches. All distinct choices are locked before
any sacrifice and rechecked by incarnation before payment. One count cost is
one simultaneous departure batch; separate cost declarations remain separate
payments. The total-cost service protects mana and other resources when a locked
count payment becomes stale. Existing `SacrificeNMatchingCost` excludes the
source as before. Positive counts are checked during definition validation.

`ThatEnteringPermanent` is a typed source reference only within
`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), body)`.
It uses the stack source's saved incarnation, including in copied triggers and
clones. Other events, activation bodies, and tuple spellings reject this name.
This permits Mold Demon's unpaid sacrifice and Electric Eel's entry damage.

### RON all-player untap limits

The compiler accepts
`EachPlayerEffect(AnyPlayer, [CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(n), filter)])`
in a permanent's rules, with exactly one operation and nonnegative literal `n`.
`filter` uses the existing permanent-filter grammar. It lowers to
`Static(LimitUntapsDuringStep(filter, n))`; it does not grant a recipient ability.
Other player filters, dynamic limits and other operations remain unsupported.
Smoke and Damping Field use this form. The engine saves membership and lets the
active player choose a legal set before simultaneous untap. Overlapping filters
apply together. The spell, cost, entry and resolving-action contexts do not gain
this declaration.

### RON library-top inspection

Resolving action lists accept:

- `LookAtTheTopNumberCardsOfPlayersLibrary(player, value,
  [LeaveRemainingCardsOnTopOfLibraryInSameOrder])`.
- The same form with `[PutTheRemainingCardsOnTopOfLibraryInAnyOrder]`.
- `MayHavePlayerAction(player, Shuffle)`.

`value` is a scoped resolution value. Direct integer literals must be nonnegative;
computed nonpositive counts inspect no cards. Each form reads an existing typed
player reference; it does not declare a target. Unbound references,
other ordering operations, multiple operations, and other optional actions
remain errors. Rules, costs and entry operations do not gain these actions.
The resolving controller sees the cards and chooses their order or optional
shuffle, even when the library belongs to another player. Factories build fresh
target references. Copies can retain or change their own targets.

Lowering uses `LookAtLibraryTopNumber(You, player, value, reorder)` and
`MayShuffleLibrary(You, player)`. Short libraries supply all remaining cards;
empty libraries permit a later optional shuffle. Inspection neither reveals
cards publicly nor changes their zones. Evidence: `TestRONLibraryInspection*`.

### Library inspection engine API

`LookAtLibraryTop(viewer, library PlayerRef, count int, reorder bool)` records
private card snapshots at resolution. `Game.LibraryLooks(viewerID)` returns
independent copies of that viewer's history. History records past observations;
it does not grant access to the current library order. Optional
`LibraryLookObserver` callers receive private copies before the next choice.
`LibraryTopOrderChooser` callers choose a complete order of the inspected cards.
Other players use the existing scry choice service; both returned partitions
stay on top. Invalid orders retain the complete original order.

`MayShuffleLibrary(chooser, library PlayerRef)` offers a resolution decision to
the chooser and uses the selected player's existing shuffle service on acceptance.
Both APIs validate their player references. Negative inspection counts fail
card-definition validation. Missing players perform no actions or choices.
HumanPlayer displays private cards and supplies ordering choices through its
own choice channels. The Player interface remains unchanged.

## RON milling

`MillNumberCards(Integer(n))` supports a nonnegative fixed count in resolving
player actions, including `PlayerAction(player, ...)` and `PlayerActions`.
The enclosing player scope supplies the subject. A bare action uses the ability
controller. Typed player targets and local `ActionPlayer` bindings remain
separate between instances and copies. `MillNumberCards(value)` also accepts
scoped resolution values through the shared value compiler, including
`PowerOfPermanent`, `ManaValueOfPermanent`, arithmetic and bound number reads.
The amount is read once at resolution before mill modifiers apply. Nonpositive
computed amounts move no cards. All-player milling remains unsupported.

Waiting delayed mill bodies declare explicit source reads from their player and
amount without evaluating either operand. A zero amount retains the declaration.
Departure and phasing preserve the saved incarnation; a returned card cannot
replace it. Implicit attachments and future delayed bodies supply no inferred
source read.

`MayCost(MillNumberCards(Integer(n)))` and
`PlayerMayCost(player, MillNumberCards(Integer(n)))` require a positive fixed
count and an adjacent `If` or `Unless` payment continuation. They use `MillCost`
through the existing optional payment transaction. A mill action moves as many
cards as possible; a mill cost requires its complete adjusted count. Millstone
and Deep Spawn use these contexts.

`Mill` and `MillCost` use `ChooseCardFromLibrary` to let the library owner order
cards that move together to the graveyard. The choice reason identifies the
graveyard operation. All order choices precede the moves. An invalid selection
uses the first remaining card; zero or one card needs no order choice. Payment
queries and transaction validation make no live order choices. Existing library
order remains the default for `BasePlayer` and cloned players.

### Mana value targets with announced X

Use `TargetManaValueEqualX()` with a typed permanent target to compare its
current mana value with the action's announced X. Use
`DeclareSpellTargetManaValueX(name, filters...)` for a typed spell target.
The target spell's own X contributes to its mana value; X in a permanent's cost
is zero. Target checks and copy choices use the action's stored X.

RON lowers `TargetPermanent(And([IsCardtype(Artifact),
ManaValueIs(EqualTo(ValueX))]))` and
`TargetSpell(And([AnySpell, ManaValueIs(EqualTo(ValueX))]))` to these services.
The compiler requires announced X in the scope. Other comparisons, static
filters, disjunctions and combined spell mana value/permanent-target restrictions
remain unsupported. Detonate and Spell Blast have complete RON gameplay tests.

## RON singular graveyard exile and source-death cards

`Exile([CardInGraveyards(ref)], [])` lowers to
`mage.ExileGraveyardCard(CardRef) ResolvingEffect`. It accepts one declared
graveyard target or `Trigger_ThatGraveyardCard` in the body of
`WhenAPermanentDies(SinglePermanent(ThisPermanent))`. Modifiers, plural subjects,
unbound references and callable reference symbols are rejected.

The death body receives a fresh card binding per factory. Detection saves the
first graveyard visit through `PermanentDies().WithCard(binding)`, restricted by
`EventSourceIsSelf`. Copies retain the captured visit; a card that leaves and
returns is a new object. Existing `PutGraveyardCardIntoHand` can also read this
typed binding. Battlefield return still requires a creature card target.
Host death has the separate binding contract below. General filtered death-card
bindings remain unsupported.

Grave Robbers uses the existing any-graveyard artifact target, mana and tap
payment, and life action. An illegal sole target prevents both exile and life
gain. Cyclopean Mummy uses its captured death card without a target. Phasing
does not cause a death event. See `TestRONGraveRobbers*`,
`TestRONCyclopeanMummyDeathCard`, `TestRONGraveyardExile*` and
`TestRONDeathCard*`.

## RON fixed landwalk keyword filters

`Landwalk(IsSupertype(Legendary))` lowers to the engine's legendary landwalk
keyword. The existing printed, static grant, temporary grant and removal
contexts use the same validator. `HasAbility(Landwalk(filter))` and
`DoesntHaveAbility(Landwalk(filter))` now accept fixed `IsLandType` filters
for Plains, Island, Swamp, Mountain, Forest and Desert, and
`IsSupertype(Legendary)`. They query the current keyword, including layer
changes, and targeted abilities check the filter again at resolution.
The existing `ManaAbility` predicate keeps its separate ability-class behavior.
Unknown, variable, combined and callable-symbol forms remain rejected.
Legendary landwalk blocking overrides remain rejected: the existing override
service supports subtype landwalk only.

`TestRONFixedLandwalkLivonya` and `NativeCombat` execute Livonya Silone's full
record, including first strike. `TestRONFixedLandwalkAssassin`,
`AssassinRollback` and `InstancesAndRetargetedCopy` execute Merfolk Assassin's
full record with live keyword legality, source and target identity, phasing,
regeneration, costs, copies and clone isolation. `Filters`, `GrantRemoval`,
`Invalid` and `Inventory` cover the shared shapes and semantic roles.

## RON rounded resolving values

`HalfRoundedDown(value)` and `HalfRoundedUp(value)` each require one supported
resolving value. They can nest around announced `ValueX`, battlefield counts,
supported captured results, characteristic reads and existing arithmetic.
Bindings remain local to the enclosing action and must exist before the read.
The compiler lowers these forms through the shared engine division services
with a fixed divisor of two. These forms are supported by the consumers of
`compileInstanceValue`, including damage packets. Static layer, entry, cost and
replacement value contexts do not gain support from this adapter.

Banshee uses the announced activation X for half-down targeted damage and
half-up self damage. Eternal Flame uses the resolving controller's current
Mountain count for full targeted damage and half-up self damage. Both use one
simultaneous damage instruction. The corrected Eternal Flame source matches
`data/DRK.json`; it previously halved targeted damage and rounded self damage
down. Evidence: `TestRONRoundedDamage*` and `TestOpponentOrPlaneswalkerTarget*`.

## RON source-conditioned control

`GainControlOfPermanentUntil(subject, duration)` supports typed permanent
subjects in existing resolving permanent action scopes. `duration` is exactly
`UntilPermanentChangesControl(You, ThisPermanent)` or
`Or([UntilPermanentChangesControl(You, ThisPermanent),
ForAsLongAsPermanentRemainsTapped(ThisPermanent)])`. `Or` gives the two endpoints:
either loss of source control or loss of its tapped state ends the duration.
It is not a disjunction of conditions that keep the effect active. Other players,
sources, durations, missing endpoints and incompatible bindings remain errors.
Spell-source, static, entry, cost and replacement-operation contexts gain no
support. Each factory creates its own typed target declarations.

The compiler lowers these forms to
`mage.GainControlWhileSourceControlled(subject, requireTapped)`. The action
keeps the resolving controller and the original source and target incarnations.
It starts only if the source duration is valid at resolution. Source departure,
phasing and loss of control end both forms; untapping also ends the second.
Target phasing preserves a valid duration. Ability removal does not erase an
installed effect. Copies keep their source identity and use their own controller
and target frame. No source LKI or returned object can start the old duration.

Aladdin and Thrull Champion use the control-only endpoint. Rubinia Soulsinger
and Willow Satyr use both endpoints and retain their independent optional untap
permission after the target leaves. Thrull Champion's global Thrull bonus uses
the existing static creature layer. Tests: `TestRONSourceControlCards`,
`RollbackAndCasting`, `UntapAndInstances`, `CopiesAndContexts`, `Invalid`, and
`Inventory`. Compilation counts remain separate from gameplay coverage.

## RON spell-or-permanent color changes

On a nonpermanent spell, `SpellActions(Targeted_Modal(branches))` supports
exactly two `Targeted` branches for this shape: one `TargetSpell(AnySpell)`
with `CreateSpellEffect(Ref_TargetSpell, [SetColor(SimpleColorList([color]))])`,
and one `TargetPermanent(AnyPermanent)` with
`CreatePermanentLayerEffect(Ref_TargetPermanent, [SetColor(SimpleColorList([color]))])`.
Each branch has one target and one action in its `ActionList`. Both branches
must use the same single actual color. Branch order is free. Extra actions,
filters, unbound or cross-category references, different colors, empty or plural
colors, casting modifiers and permanent-card contexts remain errors.

These alternatives lower to one union target, not a modal spell. No mode choice
occurs. A copied spell can choose a permanent in place of a spell, or a spell in
place of a permanent. Each factory creates its own declaration. This shape
covers Chaoslace, Deathlace, Lifelace, Purelace and Thoughtlace, including all
six reprints. `TestRONLace*` covers their complete records. Text-changing
`Targeted_Modal` consumers remain unsupported.

The engine API is `DeclareSpellOrPermanentTarget(name string)
*SpellOrPermanentTargetBinding` and `SetObjectColor(object
*SpellOrPermanentTargetBinding, color core.Color) ResolvingEffect`, with
`WithTargets(object)`. Declaration validation requires a declared nonnil binding
and one of White, Blue, Black, Red or Green. The target uses stack object IDs
and battlefield incarnation IDs. Resolution reuses `Game.ChangeObjectColor`.
The color replaces existing colors without changing mana symbols. It lasts
through cleanup, control changes and phasing, and carries from a permanent
spell to the permanent it becomes. Later zone changes end it. Color changes
are not copiable values. Existing layer timestamps and protection checks apply.

## RON spell-or-permanent resolving bodies

`SpellActions(Targeted_Modal([Targeted([TargetSpell(AnySpell)], ActionList(actions)),
Targeted([TargetPermanent(AnyPermanent)], ActionList(actions))]))` accepts two
unfiltered branches in either order on a nonpermanent spell. Each body uses the
existing resolving action compiler. Spell bodies bind `Ref_TargetSpell`;
permanent bodies bind `Ref_TargetPermanent`. Cross-category references, duplicate
categories, nested targets, empty bodies, filters, permanent contexts and casting
modifiers reject. Unsupported actions remain errors. In particular, text changes
remain unsupported. The existing exact Lace color form retains its contract.

One union declaration supplies the target; this syntax makes no mode choice.
`ResolveObjectTarget(target *SpellOrPermanentTargetBinding, spell, permanent
ResolvingEffect) ResolvingEffect` selects the matching body at resolution.
`SpellFromObjectTarget(target) *SpellTargetBinding` and
`PermanentFromObjectTarget(target) *PermanentTargetBinding` read that declaration
without adding a target. Results and typed references belong to each factory and
branch. A copy can retarget across categories. The body requires a legal saved
object at its start; information reads after its departure retain existing LKI.

`TestRONObjectTargetBody*` covers these adapters and both actual word-choice
bodies from Magical Hack and Sleight of Mind with their text consumers removed.
This is partial-card evidence; neither complete card is accepted yet.

## RON activation history at resolution

`BindActivationHistory(name string) *ActivationHistoryBinding` declares history
for one activated ability. Supply it once with
`WithActivationHistory(binding *ActivationHistoryBinding) ActionOption` on
`NewAction(ActionActivated, ...)`. Use
`IfActivationsAtLeast(binding *ActivationHistoryBinding, minimum int,
body ResolvingEffect) ResolvingEffect` to read the current turn's count during
resolution. Validation rejects missing declarations, nil or empty bindings,
negative thresholds, duplicate options, spell declarations and one binding
shared by different abilities.

The RON compiler supports this resolving action in nonmodal activated abilities
that use the stack:

```ron
If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))),
   [CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,
       ActionList([SacrificePermanent(ThisPermanent)]))])
```

The threshold can be any nonnegative integer. The branch must have at least one
supported action. Its declarations use the enclosing ability's lexical scope.
A delayed body cannot read this enclosing history condition. Spell, static,
trigger and modal contexts remain unsupported. Mana bodies support fixed-color
production with history-gated delayed instructions, as specified below.
The obsolete resolution-count spelling remains rejected.

Dragon Whelp and Nalathni Dragon use this shape. All identical records now count
activations, as Oracle requires. Farrelite Priest and Initiates of the Ebon Hand
use the same per-ability history with immediate mana resolution and automatic
payment planning. Their complete RON gameplay is covered by `TestRONHistoryMana*`.

## Mandatory resolution sacrifice payments

`IfRequiredPaid(player PlayerRef, costs CostGroup, body ResolvingEffect)` and
`UnlessRequiredPaid(player PlayerRef, costs CostGroup, body ResolvingEffect)`
use the shared total-cost transaction without an optional payment choice.
The supported group contains one or more matching sacrifice or uncaptured discard
costs. Empty groups, other cost types, and captured payment values are rejected by definition
validation. This restriction does not require a player to activate mana abilities
for a mandatory cost (CR 118.3c).

A source-excluding cost excludes the source incarnation saved by the resolving
ability. A returned card is a new object and can pay that old ability.
Payment branches depend on the start of a legal payment, including a sacrifice
whose move is replaced. Failed total payments leave the game unchanged.

RON supports adjacent `MustCost(SacrificeAPermanent(filter))` and
`If(condition, [actions])` or `Unless(condition, [actions])` in supported spell,
activation, and trigger action sequences. The condition is the symbol
`CostWasPaid` or `CostWasntPaid`. Bodies must be nonempty. Each pair keeps its
own result, including nested pairs. The current resolving controller pays.
The filter uses the existing sacrifice filter grammar, with
`Other(ThisPermanent)` also permitted at the root or in nested `And` filters
when the source is a permanent. Source exclusions inside `Or` remain rejected.
Mandatory tap, mana, plural sacrifice, saved sacrifice values, and
`IfElse` continuations remain unsupported RON shapes.

`SacrificePermanentAction` sends sacrifice through the shared replacement
pipeline. `WithExile(sourceID)` redirects the move to exile. The controller
chooses among replacements in the same priority tier; each replacement applies
once and the pipeline checks the changed action again. A redirected sacrifice
fires sacrifice and zone-change events with the final destination. It does not
fire a death event or update creature-death history. Regeneration does not
replace sacrifice.

Evidence: `TestRequiredPayment*` and `TestRONRequiredPayment*`. The complete RON
Lord of the Pit records use this adapter; Transmute Artifact still requires
search and saved-payment-value support.

## RON live static P/T bonuses

The RON compiler supports singular static `AdjustPTXY(PlusX, PlusY, x, y)` and
`AdjustPTForEach(power, toughness, count)` on `ThisPermanent` or a declared
`HostPermanent`. The per-member multipliers are signed integer literals. Values
use integer constants, `Twice`, `HalfRoundedDown`, `HalfRoundedUp`, and supported
battlefield counts. The exact Aura/attachment count uses `AttachedAuras`.

These adapters reuse `Static`, `While`, `ModifyPT`, `CountPermanents`,
`PermanentsControlledBy`, `MatchingPermanents`, `AttachedAuras`, `MultiplyValues`,
`DivideRoundDown` and `DivideRoundUp`. Values are live on each layer pass; they
are not wrapped in `ValueAtResolution`. P/T-dependent values, X and event
results are rejected in this static context. Combined modification lists and
collection subjects remain separate contexts.

Native `CopySpellOnStack` can retarget Aura casting declarations. A permanent
spell copy enters as a token with separate ability and static-effect state.
The normal entry path applies replacements and events; the copy has no mana
spent to cast it. Illegal and countered copies leave no card in another zone.

## RON singular entry counters

`AsPermanentEnters(ThisPermanent, [EntersWithACounter(counter)])` accepts one
entry operation on a permanent. It uses the same validated permanent counter
types as fixed entry counters, including named counters and supported
`PTCounter` pairs. The operation creates a fresh `EntersWithNCounters(counter,
1)` ability for each factory call. The source must be the bare `ThisPermanent`
symbol. Additional operations, other subjects and resolving-action contexts
remain rejected. Entry and counter replacements use the existing engine service.

`TestRONSingularEntryCounter*` executes the complete Icatian Javelineers record
and the exact entry clauses of Homarid and Tidal Influence. Homarid now has
complete counter-cycle coverage. Tidal Influence still rejects its independent casting restriction. See the
[rules contract](engine-rules-contracts.md#ron-singular-entry-counters).

## RON source counter-state cycles

`TriggerA(WhenAPermanentHasNumberCountersOfType(SinglePermanent(ThisPermanent),
GreaterThanOrEqualTo(Integer(n)), counter), body)` supports a nonnegative literal
threshold and the validated permanent counter kinds. Other comparisons, dynamic
thresholds, host subjects and collections remain declaration errors. The body
can read `Trigger_ThatPermanent` as the original source through the engine's
typed source reference. The read is local to this trigger; other rules cannot
use it. Each factory creates a fresh state-trigger ability.

`RemoveCounters([AllCountersOfTypeFromPermanent(counter, subject)])` supports
one resolving instruction with a typed source, Aura host, declared permanent
target or bound permanent event reference. It uses `RemoveAllCounters`, reads
the live count at resolution and removes only that kind. It is not a payment.
`RemoveCounters([ACounterOfTypeFromPermanent(counter, subject)])` uses the same
typed references and removes one available counter with `RemoveCountersFrom`
and `Fixed(1)`. No counter is required for the action to resolve. Later
instructions still run when no counter remains. Plural instruction lists and
numbered or chosen removal shapes remain rejected in actions; existing counter
payment forms remain available in costs.

`WhenState` and `SourceHasCounters` supply arming and the native stack lifecycle.
The threshold is not an intervening-if condition. Reducing the count after the
trigger queues does not cancel its reset. A returned source cannot replace the
original source; phasing makes it absent for the action. Native ability copies,
permanent-spell copies and game clones retain separate execution state.

Evidence: `TestRONCounterState*` in `counter_state_test.go`; full Homarid gameplay,
exact Tidal Influence reset clause, declaration rejection, inventory roles,
entry replacements, targets, hosts, state arming and instance identity. See the
[rules contract](engine-rules-contracts.md#ron-source-counter-state-cycles).

## RON source and Aura-host untap rules

`PermanentRuleEffect(ThisPermanent,[DoesntUntapDuringControllersUntap])` and
`PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap])` lower to
`mage.PreventUntapDuringStep`. The host reference requires Aura scope. Static
`If(condition,[PermanentRuleEffect(subject,[DoesntUntapDuringControllersUntap])])`
uses the same rule and the existing static condition compiler, including live
source or host counter checks. The body must contain exactly this one operation
and one rule. Returned objects do not inherit old attachment references.

The restriction is checked before simultaneous untap, after characteristic layers.
Source removal, phasing and ability removal disable it. Removing the recipient's
abilities does not disable a separate source's rule. Other untap effects remain
legal. Optional source untap retains its existing adapter; optional host untap and
mixed conditional rule/layer bodies remain unsupported. `TestRONHostStaticUntap*`
and `TestSubjectUntap*` supply declaration and execution evidence. The four
inventory consumers retain other blockers; this is not full-card support.

## RON resolving single-counter removal

The single-counter action uses the existing source, Aura host, declared target
and bound event scopes. Each factory builds its own references. It adds no
payment, replacement or entry context. Invalid counter kinds, arity, lists and
unbound references fail compilation.

Evidence: `TestRONSingleCounterRemoval*` in `counter_removal_test.go` covers
complete Venarian Gold gameplay and eight exact removal clauses from rejected
records. See the [rules contract](engine-rules-contracts.md#ron-resolving-single-counter-removal).

### RON completed card draws

`TriggerA(WhenAPlayerDrawsACard(players), actions)` accepts `AnyPlayer`,
`Opponent`, or `SinglePlayer(You)`. The event supplies the typed player reference
`Trigger_ThatPlayer` only within its action body. Each factory builds a separate
binding. The adapter uses `PlayerDrawsCard`, `OpponentDrawsCard`, or
`ControllerDrawsCard`; it does not declare a target. Other player filters and
card references remain unsupported in this context.

### RON resolving Aura host controller

Bare `HostController` is a typed player reference in an Aura's resolving actions.
It lowers to `ControllerOfWithLKI(AttachedPermanent)`. `HostController()` and
uses outside an Aura reject during compilation. The reference reads the current
attachment and its controller when the action runs. It does not read the Aura's
controller or the upkeep event's saved `Trigger_ThatPlayer`.

A departed Aura supplies its last attachment. A departed or phased host supplies
its retained controller information. Returned objects cannot replace the saved
incarnations. These information reads do not permit actions on absent hosts.
The existing `HostPermanent` action still requires a present host.

Paralyze's complete record now compiles, including entry tap, normal untap
restriction, and optional four-mana payment by the resolving host controller.
Payment remains optional even when the host is already untapped or absent.
Curse Artifact still rejects optional `SacrificePermanent(HostPermanent)`;
Erosion still rejects alternative `Or` mana/life costs. Neither full card is
claimed supported. Evidence: `TestRONHostController*` in `pkg/mage/dsl/ron`.


## RON fixed alternative resolution payments

In supported spell, activation and trigger action lists, `MayCost(cost)` or
`PlayerMayCost(player, cost)` accepts `PayLife(Integer(n))`, with a nonnegative
integer literal, and `Or([cost, cost, ...])`. Every alternative must be fixed
`PayMana([...])`, fixed `PayLife(Integer(n))`, or another `Or` of those costs.
Each `Or` requires at least two alternatives. X, dynamic values, conjunctions,
other cost types and malformed declarations remain rejected in this context.
Activation costs and spell additional costs do not gain this RON grammar.

The adjacent nonempty `If` or `Unless` continuation uses symbol `CostWasPaid`
or `CostWasntPaid`. Its result belongs to that payment, including nested and
consecutive pairs. Zero payments still require acceptance. After acceptance,
the engine offers payable alternatives to the resolving payer; one payable
option needs no mode choice. Mana abilities can fund mana options. Only the
selected option is paid, through the existing total-cost transaction. Resolution
payments ignore activation cost reductions during affordability and locking.

`Or` lowers to fresh `EitherCost` declarations, and life payment lowers to
`LifePayCost`. Existing `IfPaid`/`UnlessPaid`, typed player and spell references,
attachment information, countering, destruction, phasing and copying services
keep their ownership. The engine's private mana-policy check is shared by
resolution affordability and payment locking. Existing casting and activation
callers retain their policies.

Full-record behavior mappings: `TestRONAlternativePaymentErosion`,
`ThrullWizard`, `InstancesAndRetarget`, and `NativeUpkeepAndLifeClause` in
`pkg/mage/dsl/ron/alternative_payment_test.go`. The last test covers only the
exact upkeep clause of Season of the Witch; its full record remains rejected
for the independent end-step history filter. `Declarations`,
`NestedAndInventory`, and `PolarityAndZero` establish grammar, scope, result and
zero-payment contracts. Inventory role labels do not assert gameplay support.

### RON optional sacrifice of a declared permanent

`MayCost(SacrificePermanent(ref))` and
`PlayerMayCost(player, SacrificePermanent(ref))` accept a bound singular permanent
reference in resolving action lists. The adjacent nonempty `If` or `Unless`
continuation uses the symbols `CostWasPaid` or `CostWasntPaid`; all four branch
polarities work. Results stay local to each pair, including nested pairs.
`MustCost`, composite/alternative costs, and collections gain no new support.

The adapter uses `IfSacrificePaid` or `UnlessSacrificePaid`. These engine effects
resolve the subject once and use the existing matching-sacrifice transaction
with its captured ID and incarnation. The payer must control the subject.
Missing, phased, returned, or uncontrolled subjects cannot pay. A declined or
failed payment does not sacrifice another object. Zone replacements can change
the destination without changing successful payment.

An Aura host-controller upkeep exposes `Trigger_ThatPermanent` as the captured
host and `Trigger_ThatPlayer` as the upkeep player. These bindings are local to
the trigger and are fresh in each factory instance. Curse Artifact uses these
references because its Oracle text refers to that artifact and player. Its
complete record is covered by `TestRONTypedSacrificeCurseArtifact` and
`TestRONTypedSacrificeNativeUpkeep`. The remaining `TestRONTypedSacrifice*` tests
cover rejection, independent targets, copies, nested results and the source
unblocked-event clause. Mindstab Thrull's defending-player binding is supported
by the contract below.

## RON defending-player references and singular targets

`WhenACreatureAttacks(SinglePermanent(ThisPermanent))` and
`WhenACreatureAttacksAndIsntBlocked(SinglePermanent(ThisPermanent))` expose
`Trigger_DefendingPlayer` in their action bodies. The compiler lowers this
reference to a fresh `PlayerBinding` supplied to `WithDefendingPlayer`. It
retains the event player after source departure, return, control changes,
ability removal and copies. Other events and separate abilities cannot read it.

`TargetPermanent(ControlledByPlayer(Trigger_DefendingPlayer))` selects a permanent
that this saved player currently controls. The restriction can appear inside
nested `And` filters with existing restrictions. `ControlledByPlayer(You)` uses
the ability controller. Other player expressions, `Or` combinations and plural
target contexts remain unsupported by this adapter. Each target factory receives
its instance's bindings before it constructs the declaration.

The engine API `TargetControlledBy(PlayerRef) PermanentTargetOption` accepts
`You` and `FromPlayer` of a saved event binding. Definition validation checks
the event producer before checking target reads and rejects missing or invalid
references. Trigger target selection reads a private query frame made from the
existing event captures, without running effects or offering payment. The stack
object retains those player values for legality checks and retargeting copies.
Clones and copies have separate maps. Random target choices consume the live
random stream; candidate queries leave the live resolution frame unchanged.

Mindstab Thrull and Necrite use their complete FEM records. Targets are chosen
when Necrite's trigger goes on the stack; sacrifice is offered at resolution.
If its target becomes illegal, the ability fails to resolve and no sacrifice is
offered. Mindstab Thrull discards from the saved defender's current hand only
after payment. Source return or loss of control prevents payment of the old
source. Existing zone replacements can exile a sacrificed source. Necrite's
destruction forbids regeneration and still respects indestructibility.

Evidence: `TestRONDefendingPlayerNative`, `Pending`, `InstancesAndCopy`,
`ScopeAndInventory` and `Declarations` in `pkg/mage/dsl/ron`, and
`TestTargetPlayerReferenceEvent`, `Definition` and `SnapshotsAndQueries` in mage.

## RON targets that share a blocking pair with the source

Permanent target filters accept `IsBlockingAttacker(ThisPermanent)` and
`IsBlockedByDefender(ThisPermanent)`, alone or in `And` filters. The former
selects a blocker of the source; the latter selects an attacker blocked by the
source. `Or([IsBlockingAttacker(ThisPermanent),
IsBlockedByDefender(ThisPermanent)])` permits either direction, in either order.
Other combat disjunctions, host/event/target references, and non-target contexts
remain unsupported. An empty target disjunction is invalid.

The compiler lowers these forms to the engine's `TargetBlocksSource`,
`TargetBlockedBySource`, and `TargetInCombatWithSource` options on fresh typed
target declarations. No target choice is stored in the compiled plan. The engine
checks live combat membership and both incarnations at selection and resolution.
LKI cannot restore an ended relation. Tapping preserves it; phasing, control
changes, and combat exit end it. Copies and clones use the existing target services.

Evidence: `TestRONCombatTargetRealDeclarations`, `Declarations`, and
`NativeFactories`. Lesser Werewolf uses the resolving property branch described
below. Sentinel still requires indefinite dynamic base toughness. The target
clauses alone do not establish full-card gameplay support.

### Resolving permanent property conditions

RON resolving action lists support
`If(PermanentPassesFilter(subject, filter), [actions])`. `subject` uses the
existing singular permanent references and their lexical scope: source, Aura
host, declared target, or a supplied event reference. Each factory builds fresh
bindings. Branch bodies use the existing action sequence compiler, including
nested branches and adjacent payment continuations. The body must be nonempty.

Pure property filters accept `AnyPermanent`, tapped/untapped, the existing
supported supertype, card type, subtype, color and keyword predicates, fixed
power/toughness comparisons and fixed P/T equality. Nonempty `And` and `Or`
compose these predicates. Numeric limits are fixed integers; their supported
comparison forms are the existing `compileFilter` forms. Historical, controller,
attachment and combat-relation filters remain errors in this context.
`IsAttacking` and `IsBlocking` are supported as bare complete conditions through
`IfPermanentAttacking` and `IfPermanentBlocking`; they require current combat
membership and cannot occur inside a property conjunction or disjunction.

Pure properties lower to
`mage.IfPermanentMatches(PermanentRef, PermanentFilter, ResolvingEffect)
ResolvingEffect`. The predicate receives current information or the retained
snapshot for the same incarnation. It evaluates at its position in the resolving
sequence, rather than at announcement. LKI permits information reads, not actions
on absent objects. Invalid/unbound references, malformed filters and empty bodies
are declaration errors. Static `If`, intervening-if, activation restrictions,
unsupported historical conditions do not gain support here. Resolving property
`Unless` and `IfElse` use the alternatives described below.
Conditional mana production in activation branches is rejected until mana
classification and solver planning support the same condition.

Evidence: `TestIfPermanentMatches*`, `TestRONResolvingCondition*` and the existing
permanent-information, target, combat-state and payment regression tests.

## RON combat relation collections and captured prevention

Resolving `TapEachPermanent(And([IsCardtype(Creature),
IsBlockingAttacker(permanent-ref)]))` selects current blockers with `BlockersOf`.
`IsBlockedByDefender(permanent-ref)` selects current blocked attackers with
`BlockedAttackersOf`. The conjunction requires exactly the creature type and
one relation; either order is accepted. Additional filters and generic unions
remain unsupported in this collection context. Existing typed source, host,
target and event references retain their scope checks and per-instance bindings.

`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([
SinglePermanent(permanent-ref), relation-collection])), [PreventThatDamage],
duration)` accepts exactly one singular reference and one collection in either
order. Duration is `UntilEndOfCombat` or `UntilEndOfTurn`. Each instruction
captures its selected objects at resolution. The prevention sequence uses
`PreventCombatDamageBy` / `PreventCombatDamageByEach` for combat duration, or
`PreventCombatDamageByThisTurn` / `PreventCombatDamageByEachThisTurn` for turn
duration. This adapter applies in existing resolving action scopes, including
conditional bodies; it does not add static prevention, costs or entry operations.

`PreventCombatDamageByEachThisTurn(subjects PermanentSet) ResolvingEffect` uses
the existing installation and replacement services. Definition validation checks
its collection and bound references. Each source keeps its original incarnation;
departure LKI, phasing and control changes do not substitute a returned object.
The effect excludes noncombat damage and expires in native cleanup. Collections
are selected once, and duplicate incarnations install one rule per instruction.

Complete RON gameplay evidence: `TestRONCombatCollectionCards`,
`SewersAttacking`, `AnnouncementLegality`, `Invalid`, and `StrictInventory`.

## RON permanent collection damage

In resolving action lists, `SpellDealsDamage/3` and `PermanentDealsDamage/3`
accept `EachPermanent(filter)`, `EachPermanent(Ref_TargetPermanents)`, and
`EachPermanent(And([Ref_TargetPermanents, property-filter]))`. The intersection
requires exactly one plural reference and one property filter, in either order.
The property filter uses the supported resolving property grammar. A live
battlefield collection uses the supported permanent filters, including attacking
and blocking status and nonempty nested `And`/`Or`. This action declares no targets.

`FilterPermanentSet(PermanentSet, PermanentFilter)` keeps current matching
members from a typed set. `PermanentDamageRecipients(PermanentSet)` places that
set in the existing damage batch without player recipients. Both services retain
definition validation and the original set's identity and target checks. A zero
engine filter keeps all members. RON empty logical filters are errors.

Sandstorm selects attacking creatures when it resolves. Winter Blast first taps
its legal X creature targets, then reads flying on those same legal targets.
A ground creature remains a legal target and is tapped. Later flying changes
change damage eligibility without changing the target declaration. Plural target
references remain local to each spell, mode, activation or trigger; the existing
rejection of delayed plural target reads remains in force. These additions do
not add collection recipients to replacement operations or damage packets.
Evidence: `TestRONCollectionDamage*` and `TestFilterPermanentSet*`.

## Captured temporary characteristic values

`CreatePermanentLayerEffectUntil(subject, [SetPT(ManualPT(power, toughness))],
 duration)` accepts two resolution values in spell, activation and supported
trigger action bodies. `AdjustPTX` accepts the same characteristic operands.
`PowerOfPermanent(ref)` and `ToughnessOfPermanent(ref)` read typed permanent
references with exact-incarnation information. `ActionPermanent` refers only to
the modified subject inside these values. `Minus(left, right)` can nest these
reads and supported integer/X values. Existing `ManaValueOfPermanent` restrictions
remain in force. Static, CDA, entry and indefinite `ManualPT` forms remain errors.

The engine captures both coordinates before it installs the modifications.
Later characteristic changes cannot recalculate those values. Existing layer
7b/7c ordering, target legality, identity, phasing and copy isolation apply.
`ToughnessOfWithLKI(PermanentRef) ResolutionValue` provides the reusable toughness
query; it reads current layered toughness, retained departure/phase toughness,
or zero for an unknown incarnation. It does not permit actions on absent objects.

`UntilEndOfNextUpkeep(You)` saves the resolving controller and ends after priority
in that player's next actual upkeep. A duration created during that upkeep waits
for a later upkeep. Skipped upkeeps do not expire it; extra upkeeps can. Source
control changes do not redirect expiry. The engine API is
`UntilEndOfNextUpkeep(PlayerRef, ...ContinuousModification) ResolvingEffect`.
`UntilNextUpkeep(You)` retains its beginning-of-upkeep endpoint.

Halfdane's complete record uses `ManualPT` and the end-of-upkeep duration.
`TestRONTemporaryCharacteristicHalfdane` covers its casting, trigger, targets,
values, identity and duration; `Values`, `Invalid`, `InvalidShapes` and
`LoadAndInventory` cover shared value forms and strict loading. Berserk still
needs its `AttackedThisTurn` delayed condition; Blood Lust still needs resolving
`IfElse`. Their partial compilation progress is not full-card gameplay evidence.

## RON paid selection and simultaneous untap

The compiler accepts this adjacent pair in an `ActionList`:

```text
PlayerAction(player, ChooseAnyNumberPermanentsAndPayManaForEach(
  And([filter, ControlledByPlayer(PlayerAction_ThatPlayer)]), mana)),
PlayerAction(player, UntapEachPermanent(TheChosenPermanents))
```

Both player references must identify the same typed plan. The controller token
is local to the choice. `filter` must require `IsTapped` through a bare predicate
or an `And` conjunction and otherwise uses the supported permanent filters.
The outer `And` contains exactly one such filter and one controller clause,
in either order. `mana` is the existing nonnegative fixed mana-symbol list;
X is rejected. The result must be a bare `TheChosenPermanents` in the adjacent
untap. Other consumers, intervening actions, unbound reads and different player
references remain errors. This is a complete choice/payment/untap procedure,
not general support for saved permanent collections.

The pair lowers to `mage.UntapWithManaPayment(player, filter, cost)` in resolving
spell, activation, trigger and branch action lists. The engine owns the local
selection, incarnation capture, total mana transaction and untap batch. Choices
are made at resolution, not announcement. They are not targets; shroud does not
exclude them. An upkeep trigger retains its event player through source control
changes, departure, stack copying and game cloning. Each resolution has its own
choice and payment result. Magnetic Mountain (ARN and 3ED) and Thelon's Curse
(FEM) have complete-record gameplay tests in `TestRONPaidUntapCards`.

### RON exile activation payments

`Exile([Permanent(ThisPermanent)], [])` is an activation cost. It uses
`ExileSourceCost` and can be combined with mana, tap, life, and other supported
costs in `And`. Exile happens during payment; the body runs at resolution.

`Exile([ACardInGraveyards(And([cardFilter,
InAPlayersGraveyard(SinglePlayer(You))]))], [])` chooses one matching card from
the activator's graveyard. Exactly one graveyard exile cost is supported in an
activation. The controller clause must occur once in the outer conjunction.
Other conjuncts use the existing intrinsic card type/color filters and nonempty
logical combinations. Plural, foreign-graveyard, and permanent-choice payments
remain unsupported. The object list has one element and the operation list is
empty. These forms are not resolving actions or resolution costs.

`ManaValueOfExiled(TheCardExiledThisWay)` reads a typed number saved by that
activation's graveyard payment. Each factory creates a fresh number binding;
pending activations, modes, copies, and clones retain their own paid values.
Source exile cannot produce this result. Undeclared, malformed, duplicate, and
cross-activation results are compile errors. `NumberCountersOfTypeOnPermanent`
accepts this value without extending other counter-amount contexts.

Bare `ShuffleGraveyardIntoLibrary` in a resolving action list lowers to
`ShuffleGraveyardFor(You)`. It moves only the resolving controller's graveyard,
then shuffles that library even when the graveyard is empty. A changed-controller
copy uses its own controller. This action makes no target declaration.
Evidence: `TestRONExilePayment*` and `TestGraveyardOnlyShuffleBatch`.

## RON opponent entry choice

`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])` lowers to
`mage.ChooseOpponentAsEnters()`. It saves the entry controller's opponent on
the permanent before battlefield registration. The choice stays with that
incarnation through control changes and phasing. A returned card chooses again.

With this declaration, an upkeep trigger can select
`SinglePlayer(TheChosenPlayer)`. Its body reads the saved event player through
`Trigger_ThatPlayer`. Black Vise and The Rack use that reference for both
`TheNumberOfCardsInPlayersHand` and damage, so their amounts use the current hand
at resolution and their recipients survive source departure and stack copying.

Exactly one opponent choice operation and declaration are supported. The rules
may list the upkeep consumer before its entry declaration. Duplicate and unbound
declarations are errors. Combined entry choices, other player filters, resolving
choices, chosen-player action reads and chosen-player end-step triggers remain
unsupported. The legacy opponent ETB helpers retain their current behavior.

## RON end-of-combat triggers

`TriggerA(AtTheEndOfCombat, body)` accepts the bare event symbol and a supported
action list, including its standard target declaration. It lowers to
`mage.At(mage.EachEndOfCombat, ...)`. Both players' combats match. This event
supplies no `Trigger_ThatPlayer` or `Trigger_ThatPermanent` binding.

The supported intervening shape is
`TriggerI(AtTheEndOfCombat, ACombatPermanentPassesFilter(ThisPermanent,
Or([AttackedThisCombat, BlockedThisCombat])), body)`. The two history symbols can
occur in either order, once each. Other subjects, predicates, combinations and
event tuple forms remain errors. It lowers to
`mage.WithInterveningIf(mage.SourceAttackedOrBlockedThisCombatCondition())`.
The original incarnation must have attacked or blocked in the current combat
at detection and resolution. Entering attacking does not count as attacking
(CR 508.4). Leaving combat does not erase a declaration. Phasing prevents the
condition from matching; a phase cycle preserves history. A new incarnation
has no history from the old one. Each factory builds separate abilities and
target declarations through the existing action-plan service.

Evidence: `TestRONEndCombat*` tests the entry and combat rules extracted from
complete Clockwork Avian and Clockwork Beast records, plus synthetic plain-event,
identity and invalid-declaration contracts. Their restoration rules remain
unsupported. Neither clockwork nor The Wretched has complete gameplay coverage
from this adapter, and their complete records still fail strict loading.

## RON bounded optional counter placement

`PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(amount,
 counter, subject)], CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(
 counter, subject, Integer(limit)))` accepts one placement and one restriction.
The counter and typed permanent reference must match. The limit is an integer
literal from zero through 255; the amount uses existing resolving counter values,
including
announced `ValueX` only in an X-enabled scope. Permanent counter types use the
existing placement validator. Source, attachment, target and event references
retain their existing scope requirements. This is a resolving action, not a
cost, static modification, entry operation or replacement declaration.

The adapter builds `mage.AddCountersUpTo` with fresh instance bindings. At
resolution, the effect controller chooses zero through the smaller of the
amount and available counter room. An absent or phased subject, nonpositive
amount or full subject causes no choice. Invalid choice responses place nothing.
The action limit applies to its counter on its original subject after counter
replacements; it does not remove excess counters supplied by other actions.

Clockwork Avian and Clockwork Beast now compile from their complete records.
`TestRONBoundedCounters*` covers casting, choices, payment, upkeep timing,
identity, target legality, copies, clones, replacement ordering and instances.
`TestRONEndCombatClockwork` and `TestRONEndCombatInterveningIdentity` now use
complete records for entry, characteristics and combat-history interactions.
Other counter instructions and restriction shapes remain errors.

## Static prevention by source card type

`mage.PreventDamageFromCardType(subject mage.PermanentRef, kind core.CardType)
mage.StaticRule` prevents combat and noncombat damage from sources with the
specified card type. Compose it with `mage.Static`. The subject is live: `This`
protects the ability source, and `AttachedPermanent` follows the current host.
Invalid card types, nil subjects, and undeclared bindings fail definition
validation. Source types use the exact damage source's current or retained
information. Captured spells can match; a returned permanent cannot replace an
older source. The rule uses the existing static lifecycle and damage pipeline.

RON accepts the static rule
`PreventDamage(DamageWouldBeDealtByASourceToRecipient(IsCardtype(type),
Permanent(subject)), [PreventThatDamage])`, where `subject` is `ThisPermanent`
or a declared Aura's `HostPermanent`. Other filter shapes and resolving
contexts remain unsupported. Argothian Treefolk uses the complete shape.
Artifact Ward uses its host context, but its separate ability-target restriction
still prevents complete loading.

## RON prevention from blocked creatures

A permanent static rule accepts
`PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(And([IsCardtype(Creature), IsBlockedByDefender(ThisPermanent)]), Permanent(ThisPermanent)), [PreventThatDamage])`.
The two filter operands can occur in either order. Both references must name the
permanent source. Other types, references, recipients, operations, filter shapes
and resolving/cost contexts fail compilation. This shape does not extend the
ordinary permanent filter or target compiler.

Each factory builds `Static(PreventDamageFromBlockedCreatures(This))` with its
own ability. The engine checks the current blocking pair for each proposed
combat or noncombat damage event. Former pairs cannot match through LKI.
Wall of Vapor has complete RON gameplay coverage in
`TestRONBlockedDamagePreventionWall`, `StrictInventory` and `Ordering`.
Wall of Shadows shares this rule but still fails its separate targeting rule.

RON resolving `CreatePermanentLayerEffect(subject, [SetToughness(value)])`
sets base toughness indefinitely through `mage.SetBaseToughness`. It preserves
base power, bonuses and counters. The value is captured at resolution.
Supported values are integer literals, binary nested `Plus`/`Minus`, typed
permanent power/toughness reads, and a matching graveyard-card count with one
`InAPlayersGraveyard(SinglePlayer(playerRef))` and one card filter in `And`.
The operands can appear in either order. Existing lexical bindings apply.
Static and temporary RON forms and chosen-number values remain unsupported.
Sentinel and Wall of Tombstones have complete-record gameplay evidence in
`TestRONIndefiniteToughness*`; shared engine contracts are documented in
`engine-rules-contracts.md` under “RON indefinite base toughness.”

### Optional RON entry from hand

`MayAction(PutACardFromHandOnBattlefield(filter, [EntersNormally]))` is a
resolving action. It uses `mage.MayPutFromHandOntoBattlefield(mage.You, filter)`.
The controller makes the optional choice during resolution. Copies use their
own controller. The action has no target and does not cast or play the card.

The card filter accepts bare `IsPermanent`, `IsSupertype(Basic|Legendary|Snow|World)`,
`IsLandType(subtype)`, `IsCreatureType(subtype)`, existing card-type and color
filters, and nonempty nested `And`/`Or` lists. Subtype families are validated.
Only one bare `EntersNormally` entry operation is supported. Repeated sequential
entry rounds use the contract below. Other explicit player loops, entry operations
and action-result reads remain separate contracts. The engine excludes instants
and sorceries from entry.

The engine keeps each offered hand visit through both choices. Invalid, absent
or returned choices cannot move a card. Entry uses the existing preparation,
registration, attachment and entry-event services. A land-entry prohibition
keeps the card in hand with its original visit. Auras use the existing permanent
host contract: legal attachment choices ignore shroud, obey protection and
exclude phased hosts. No legal host leaves the Aura in hand.

Complete Gaea's Touch behavior is covered by `TestRONOptionalHandEntryGaeasTouch`
and `TestRONOptionalHandEntryTimingAndRollback`. Goblin Wizard's temporary
protection and Eureka's repeated player procedure remain unsupported. Their
complete records still reject. Triassic Egg now has a corrected graveyard mode
and live activation-condition support; see the activation-property section below.


## RON controller restrictions on enchant abilities

On an Aura, `EnchantPermanent(filter)` accepts
`ControlledByAPlayer(SinglePlayer(You))` or `ControlledByAPlayer(Opponent)`.
One controller restriction can occur alone or in a nonempty nested `And`
with existing permanent property filters. Operand order does not matter.
A property-only `Or` retains the existing filter grammar; a controller
restriction inside `Or` is rejected. Duplicate or conflicting controller
restrictions, other player references, malformed constructors and enchant
rules on non-Auras are compile errors.

RON lowers these restrictions to fresh `TargetPermanentYouControl(filter)`
or `TargetPermanentOpponentControls(filter)` cast targets. Every factory
has its own casting target state. Casting and resolution use the spell's
controller; a copied spell uses its own controller. Direct attachment,
optional hand entry and state-based actions use the Aura's current or
prospective controller. Ownership does not determine the restriction.
Shroud and hexproof affect spell targeting, but do not prevent untargeted
attachment or invalidate an existing attachment.

Evidence: `TestRONEnchantControllerClauses`, `Invalid`, `NestedFilters` and
`Reporting`. Gameplay tests execute the real enchant clauses from Cocoon,
Relic Bind and Tourach's Gate, including casting costs and metadata. They
are clause evidence: all three complete records remain rejected for later
unsupported operations. No full-card behavior claim follows from these tests.

## RON resolving counter conditions

Inside a resolving `If(PermanentPassesFilter(subject, filter), [actions...])`,
counter filters accept `HasACounterOfType(counter)`,
`HasNoCountersOfType(counter)`, and
`HasNumberCountersOfType(comparison(Integer(n)), counter)`. The comparison is
`EqualTo` or `GreaterThanOrEqualTo`; `n` must be a nonnegative integer literal.
The counter uses the existing permanent counter grammar, including supported
`PTCounter` coordinates. Nonempty `And` and `Or` combine these with the existing
resolving property filters. Other comparisons, dynamic thresholds, malformed
constructors and unbound subjects are declaration errors.

The condition reads the count when its action runs, after earlier instructions.
It uses current information or LKI for the exact subject incarnation. Retained
information permits a condition read, but cannot permit an action on an absent,
returned or phased-out object. Each factory constructs its own typed targets;
copies and clones retain their own bindings and state. Conditional mana
activations remain rejected until solver planning supports their complete body.
Static conditions, activation restrictions and exiled-card conditions retain
their separate grammars.

`mage.CounterCountAtLeast(counter, minimum)` compares the stored count as an
`int`; a threshold above 255 cannot wrap to a smaller count. Presence and absence
reuse `HasCounter` and `CounterCountEquals`. Execution reuses
`IfPermanentMatches`, counter actions and the existing reference service.

Evidence: `TestCounterCountAtLeast` and `TestRONResolvingCounter*`. The actual
Fasting upkeep and Tourach's Gate counter-decay clauses have gameplay coverage.
Fasting still rejects its draw-step replacement, and Gate still rejects its
host tap cost. These tests do not establish complete-card support.

### RON live activation property restrictions

`ActivatedWithModifiers(cost, body, ActivateOnlyIf(PermanentPassesFilter(subject,
filter)))` checks the current source or Aura host before activation payment.
`subject` is bare `ThisPermanent`, or bare `HostPermanent` on an Aura. Targets,
event bindings and other subjects are not available in this declaration scope.
Ordinary and modal bodies use the same restriction. Nested modifier `And` lists
combine property, turn, step and usage restrictions without replacing them.

Filters use the existing resolving property filter grammar: current types,
subtypes, colors, supertypes, tapped state, keywords and supported P/T comparisons;
nonempty nested `And`/`Or`; counter presence/absence; and counter counts with
`EqualTo(Integer(n))` or `GreaterThanOrEqualTo(Integer(n))`. Counter thresholds
must be nonnegative literals, and counter kinds must be known. This filter
reads live information, without departure LKI. Static and resolving condition
contexts retain their separate reference and evaluation contracts.

Lowering uses `mage.WithActivationStateCondition(mage.PermanentMatchesFilter(...))`.
The engine validates the typed declaration and uses the same pure activation
check for mana planning and actual payment. Restrictions do not repeat at
resolution or when a stack ability is copied. Changing counters or tapping the
host after activation does not stop the ability. Phased, absent and stale hosts
cannot qualify; changing the attachment changes the live subject.

`TestRONActivationProperty*` covers these contracts and both complete Triassic
Egg modes. Its corrected graveyard mode targets a creature card in your
graveyard at announcement; the hand mode makes an optional choice at resolution.
Tourach's Gate has activation-condition clause evidence only; its complete record
still rejects the separate host tap cost.

## RON last-turn attack conditions

Static `PermanentPassesFilter(subject, AttackedDuringLastPlayersTurn(player))`
accepts bare `ThisPermanent` or an Aura's `HostPermanent`. The player is bare
`You` or `ItsController`: the latter reads the current subject controller,
which can differ from the Aura controller. The condition reads the player's
most recent completed turn and the live subject's exact incarnation. Current
turn attacks do not replace that history. Phasing preserves history; absent,
phased and returned objects cannot use an old incarnation's history.

Static `If(condition, [PermanentRuleEffect(subject, [CantAttack])])` now uses
`mage.CantAttackIf(PermanentRef, StateCondition)`. The body has exactly one rule
and one bare operation. It reuses the existing static condition grammar,
including the last-turn condition. Existing conditional untap bodies keep
`PreventUntapDuringStep`. Mixed bodies and additional operations still reject.
History predicates in resolving, activation, cost, target and nested property
filter contexts are not added by this slice.

Engine `PermanentAttackedDuringLastTurn(PermanentRef, PlayerRef) StateCondition`
accepts `This` or `AttachedPermanent`; the player is `You` or `ControllerOf`
a live source/attachment. Invalid declarations fail definition validation.
Turn cleanup archives incarnation history per player; cloning copies all maps.
The existing ID-only history API remains available to its callers.
Evidence: `TestLastTurnAttack*` and `TestRONLastTurnAttack*`, including complete
Giant Turtle, Goblin Rock Sled and Tangle Kelp records.

## RON static source combat attributes

`PermanentRuleEffect(ThisPermanent, [MustAttack])` and
`PermanentRuleEffect(ThisPermanent, [CanBlockAnAdditionalCreature])` accept one
bare operation on the live permanent source. These declarations lower to
`Static(While(condition, GrantKeywordModification(This, attribute)))` with
`AttrMustAttack` or `CanBlockAdditional`. Each factory builds a fresh ability.
Tuple operations, tuple references, target references, Aura hosts, collections,
conditional bodies and combined operation lists remain unsupported here.

Native attacker declaration includes a source with `MustAttack` when it can
attack. Tapping, summoning sickness, phasing, attack restrictions and unpaid
attack costs still prevent declaration. The requirement applies in each combat
and reads the controller, rather than the owner. Additional blocking permits
this source to block two distinct attackers; it does not permit blocking a
flying attacker without flying or reach. It does not require a block. Ability
removal changes later declarations and leaves existing combat pairs intact.

Complete-record evidence: Juggernaut and Two-Headed Giant of Foriys map to
`TestRONSourceCombatAttributeCast`, `RequiredAttack`, `AdditionalBlock`,
`AttackControlAndFinality`, `DeclaredBlockFinality`, `GiantTrample`, `Lifecycle`,
`Invalid` and `Reprints` (all have the `TestRONSourceCombatAttribute` prefix).
Primordial Ooze remains rejected for its unresolved `CreateValueX` action.
This slice does not establish additive multiple extra-block grants, all-zone
CDA values, arbitrary combat requirements or new combat decision services.

## RON damage-received triggers

Permanent rules support `TriggerA` with either
`WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent))` or
`WhenAPlayerIsDealtDamage(SinglePlayer(You))`. The first form uses
`ReceivesDamage(This, amount)`; the second uses
`ControllerIsDealtDamage(amount)`. Both save `Trigger_AmountOfDamageDealt`
through a fresh `NumberBinding` for each factory instance. Counter placement
can use this saved number in `NumberCountersOfTypeOnPermanent`.

Only the permanent event supplies `Trigger_ThatCreatureOrPlaneswalker`. It
resolves through `This` and keeps the original source incarnation. It is not a
target. The player event checks the source controller when damage occurs; it
does not supply `Trigger_ThatPlayer`. Other subjects, malformed references and
reads outside the matching event body remain declaration errors.

Fungusaur's complete RON record is supported. Living Artifact's damage clause
is supported, but its complete record still rejects optional counter payment.
Lich retains its other unsupported contracts. This adapter does not grant
support for general damage-recipient filters or optional counter payments.

RON optional counter-placement payment accepts
`PlayerMayCost(payer, PutCounters([ACounterOfTypeOnAPermanent(counter,
And([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(payer))]))]))`
and implicit `MayCost` with `You` as payer. The two filters can appear in either
order. The controller reference must be the same bound player reference as the
payer. Counter types use the existing permanent counter vocabulary. The cost
requires one instruction and one counter; plural amounts, other filters, and
other controller references remain rejected.

These payments require an adjacent `If` or `Unless` with `CostWasPaid` or
`CostWasntPaid`, in the existing resolving spell, activation, mode, or trigger
action sequence. The cost and continuation stay together. A later payment or
sibling action cannot read that result. Selection occurs during resolution,
without targeting. A shrouded creature can pay. The engine locks its incarnation
and verifies current control and presence before payment. Counter replacements
can change or prevent placement without changing successful payment (CR 118.11–12).

`PutCounterOnMatchingCost(counter core.CounterType, filter PermanentFilter) Cost`
is the shared engine primitive. It uses the payer's current permanents, permits
the source, and treats a zero filter as unrestricted. Availability queries do
not choose or place counters. Preparation locks one choice; commit does not
choose again. The shared cost transaction and counter replacement pipeline
supply rollback, game isolation, and replacement ordering.

Evidence: `TestPutCounterCost*`, `TestRONCounterPlacementPayment*` and the existing
`TestRONLandEntry*` event contracts. The complete FEM records for Thelon's Chant
and Tourach's Chant now load and have gameplay tests for both their rules.

## Beginning-of-combat RON triggers

`TriggerA(AtTheBeginningOfCombatDuringAPlayersTurn(players), actions)` supports
bare `AnyPlayer` and `SinglePlayer(You)`. Each combat begins a separate event,
including extra combats. `You` uses the source controller at detection.
`Trigger_ThatPlayer` is a typed, local event-player reference. Each factory call
creates a separate binding, and each pending event saves its player.

The adapter uses `TurnStepBegins` with `EachBeginCombat` or `YourBeginCombat`.
Supported action lists retain their existing target and payment contracts.
Temporary permanent layer effects in this immediate body can use
`UntilEndOfCombat`. Singular layer modifications in a nested delayed-step body
also accept that duration.
Other player filters, malformed tuples, and event references outside their body
are rejected during compilation. This does not add `TriggerI`, combat-scoped
installed triggers, optional nonmana effect payments, or conditional attack tap
rules.

`TestRONBeginCombatBinding`, `BatteringRamClause`, `NativeAndControl`,
`ConsumerBoundaries`, `Invalid`, `ScopeRejection`, and `Inventory` cover this
contract. The Battering Ram test executes its complete combat-start RON clause;
the rest of that card remains rejected. Goblin Flotilla and Johan tests execute
their original event declarations with test-only bodies. These mappings do not
claim complete gameplay coverage for any of the three cards.

## RON granted controller-upkeep triggers

Supported declarations are
`PermanentLayerEffect(HostPermanent, [AddAbility([TriggerA(event, body)])])`
for an Aura and
`EachPermanentLayerEffect(filter, [AddAbility([TriggerA(event, body)])])`.
Each contains exactly one modification and one granted ability. `event` must be
`AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))`. Other events, player
selectors and combined modifications remain rejected.

The collection uses the existing static permanent filter adapter, including
its controller and source-exclusion forms. A host grant follows the current
attachment. The grant operates in layer 6. `You` and `ThisPermanent` inside
`body` refer to the recipient's trigger controller and source, respectively.
The body receives a fresh permanent scope with the ordinary upkeep event's
`Trigger_ThatPlayer` binding. It can declare its own targets. It cannot read the
donor's attachment, entry choices, X, targets or event/payment results.
Existing action and adjacent-payment restrictions still apply.

The adapter uses `GrantTriggeredAbility(PermanentRef, *GenericTriggered)` or
`GrantEachTriggeredAbility(PermanentSet, *GenericTriggered)`. Each factory builds
fresh declarations. The engine keeps ability identity separate for each grant
and each pair of donor/recipient incarnations. Layer passes and control changes
do not change that identity. Phasing suspends the grant without changing it.
Pending triggers keep the recipient incarnation and detection controller even
if the grant ends before resolution.

Complete record mappings are Energy Flux (artifact sacrifice unless {2} is
paid), Farmstead (optional {W}{W}, then gain 1 life), and The Tabernacle at
Pendrell Vale (creature destruction unless {1} is paid).
`TestRONGrantedUpkeep*` covers their casting or land play, native upkeep,
payments, recipient identity, attachment, control, phasing, copies, clones,
ability removal, later members, multiple donors, malformed declarations and
strict loading. Compilation and collection-wide gameplay coverage stay separate.

## RON continuous public visibility

In a permanent declaration, the compiler supports exactly one operation in:

- `EachPlayerEffect(AnyPlayer, [PlaysWithHandRevealed])`
- `EachPlayerEffect(AnyPlayer, [PlaysWithTopOfLibraryRevealed])`

These are static player rules, not actions or one-time reveals. The operation
and `AnyPlayer` must be symbols. Tuple operations, extra arguments, combined
operations and other player selectors are rejected. Existing all-player untap
limit declarations keep their compiler and engine behavior.

The compiler lowers the declarations to `mage.Static(mage.RevealAllHands())` or
`mage.Static(mage.RevealAllLibraryTops())`. Each factory builds its own ability
and effect state. Visibility follows current contents while the printed ability
is active, regardless of its controller. The engine owns source lifecycle,
phasing, ability suppression and cloned visibility flags. Public hands appear
in interactive snapshots without enabling the debug hand-display option.

Complete-card mappings: Field of Dreams and Revelation use
`TestRONPublicVisibilityCards`, `Lifecycle`, and `Contents`;
`TestRONPublicVisibilityInvalid` and `InventoryAndAtomicLoad` cover declaration
validation, nested semantic roles and strict loading. See the continuous public
visibility contract for engine and snapshot evidence.

## RON referenced permanent damage and source tap events

`PermanentDealsDamage(source, amount, recipient)` accepts typed permanent
references for `source` with singular `Permanent(ref)`, `Player(ref)`, or bound
`Ref_AnyTarget` / `Ref_TargetPlayerOrPermanent` recipients. Source and recipient
references use their own source, Aura host, target or event scope. The compiler
uses `DealDamageFrom(DamageSourceOf(source), recipient, amount)` and existing
resolution values. Explicit player recipients also provide the existing
recipient-value scope. Collection damage retains its existing supported forms.

`WhenAPermanentBecomesTapped(SinglePermanent(ThisPermanent))` uses
`BecomesTapped(This)`. Its body may read `Trigger_ThatPermanent` as the exact
trigger source. That name is unavailable outside the event's body. The existing
host-tap form keeps its captured host binding. Factories build separate target
and event declarations; pending copies and clones use the engine's stack frames.

Evidence: `TestRONReferencedDamage*` covers the complete City of Brass and
Tracker records, typed host/event/target sources, invalid declarations,
incarnation and clone isolation, announcement costs, prevention and sequential
resolution. These adapters add no source-choice or damage-result operation.

## RON destruction result branches

In supported resolving action sequences, `MustCost(DestroyPermanent(ref))`
followed immediately by `If(condition, [actions])` or
`Unless(condition, [actions])` attempts destruction and branches on its result.
`condition` is the symbol `CostWasPaid` or `CostWasntPaid`. Bodies are nonempty.
The reference must be a bound permanent reference in that context. This includes
source, host, target, and supported event references; it does not introduce new
bindings. Activation costs and `IfElse` do not accept this mandatory shape.
The optional `MayCost` shape uses the choice contract below.

This RON spelling denotes the actual destruction result, as Cosmic Horror's
"destroyed this way" clause requires. It uses `DestroyWithResult`,
`DestroyedThisWayInto`, `FromNumber` and `IfPositive`, rather than a mandatory
cost transaction. Regeneration, indestructibility, absence and phasing produce
zero. Each declaration builds its own typed result binding with a position-based
name. Nested pairs and copied or cloned resolution frames retain separate state.
Existing delayed effects retain the original references. Cosmic Horror executes
its complete record; Infinite Authority still requires a captured combat subject
for `Trigger_ThatCreature` and remains rejected.

### RON selected single-card discard costs

Bare `DiscardACard` in `Activated/2` or `ActivatedWithModifiers/3`, including
nested cost `And([costs])`, lowers to a fresh `mage.DiscardCost(1)`. The
activation controller chooses one card during payment. The whole mana, tap and
discard payment must succeed before the ability enters the stack. Copies do
not pay an activation cost again.

Resolving action sequences accept `MayCost(DiscardACard)` and
`PlayerMayCost(player, DiscardACard)` immediately followed by `If/2` or
`Unless/2` with bare `CostWasPaid` or `CostWasntPaid` and a nonempty action list.
The explicit payer must be a bound typed player reference. These forms reuse
`IfPaid`/`UnlessPaid` and the same total-cost transaction. Payment and choice
occur at resolution; each adjacent pair has its own outcome. Changing or
returning the source does not replace the saved ability controller. A copied
ability offers its own payment to its own controller.

Tuple discard declarations, plural costs, filtered costs, saved discarded-card
reads, mandatory resolving discards, optional cost conjunctions and discard
alternatives are not added. Unsupported complete records still reject.
Dwarven Armorer's cost/target/first counter-choice clause and Mishra's War
Machine's upkeep/payment/damage clause have engine tests in
`TestRONChosenDiscardCostArmorerClause` and `TestRONChosenDiscardCostWarMachineClause`.
Armorer's complete choice now uses the resolving-choice adapter below. War
Machine's actual-damage-result rule remains a gap. Land's Edge
still needs a typed discarded-card result. Its shared activation permission
is supported by the adapter described below.
Other evidence: `TestRONChosenDiscardCostOptional`, `IndependentPayers`,
`ReplacementsAndInstances`, `Invalid` and `StrictBoundaries`.


## RON choices during resolution

`ChooseAnAction/1` accepts at least two nonempty lists of resolving actions.
Each list uses the shared action-sequence compiler. Nested choices, local
payment continuations, draw results and created values retain their existing
contracts. All branches inherit the enclosing typed references and execution
context. Producers stay inside their branch; no result escapes to a sibling or
the enclosing continuation. Targets must be declared outside the choice.
Every branch must compile, even when a player would select another branch.
The lexical player chooses when the body runs, including immediate draw
replacements and created delayed bodies. Actions use their normal replacement,
batch and transaction services. No selected index is saved on a permanent.

Dwarven Armorer retains complete-record gameplay coverage. Island Sanctuary's
choice transport has focused evidence in `TestRONActionChoiceSanctuaryContext`,
with its unsupported attack restriction replaced by a life action. Its unchanged
complete record remains rejected until the restriction and duration are supported.
`TestRONActionChoiceSequence`, `PaymentAndDraw`, `DelayedIdentity`,
`ReplacementAncestry` and `Invalid`
cover general bodies, nested player scope, branch results, timing, identity,
LKI, copies, clones and invalid declarations.

`ChooseAnAbility/1` accepts at least two distinct keywords supported by the
existing temporary keyword adapter. It must be followed immediately by
`CreatePermanentLayerEffectUntil/3`, whose modification list contains
`AddAbilityVariable(TheChosenAbility)`. The reference is a bare symbol and is
bound only in that adjacent continuation. Other supported modifications, such
as `AdjustPT`, may share the list. Subjects and durations retain their existing
typed contracts. Fixed branches lower the selected ability without a mutable
result slot. Temporary grants and choices also accept `Rampage(n)` with a bare
nonnegative integer. Different amounts are distinct alternatives. Saved choices
outside the continuation, wrong references, duplicate options and missing
continuations remain errors. This shape executes Urza's Avenger and Gabriel
Angelfire. `HasAbility(AnyRampage)` and `DoesntHaveAbility(AnyRampage)` read current
triggered abilities through `mage.HasRampage`; Rapid Fire checks at resolution
after its first-strike grant. Rampage is not an Attr: its temporary grant uses
`mage.TriggeredAbilityModification` with the shared becomes-blocked pattern and
rampage effect. Static RON rampage grants and rampage removal remain unsupported.

Both forms use `mage.ChooseAtResolution(PlayerRef, ...ResolutionOption)`.
`ResolutionOption` has `Label string`, `Body ResolvingEffect` and optional
`When StateCondition`. The condition supplies any choice restrictions; arbitrary
action legality is not inferred. Zero available options do nothing. One option
runs without a prompt. Multiple options use the selected player's `ChooseMode`;
an invalid answer uses the first available option. Definition validation checks
all bodies, conditions and references, and prevents branch results from escaping.

Choices occur during resolution after targets and payments. The selected branch
uses the current frame, including target groups, controller, source incarnation,
event bindings and saved information. Copies make separate choices. Each factory
builds fresh effects and target bindings. Existing counters, layers, transaction,
replacement, copy and clone services own the resulting actions.

Evidence: `TestResolutionChoice*`, `TestRONResolutionChoice*`, existing
`TestRONChosenDiscardCostArmorerClause`, counter placement, temporary keyword
and next-upkeep duration contracts. Modal/spell and ordinary activation target
bindings have definition tests; the ordinary trigger has resolution and expiry
tests. These mappings do not establish collection-wide gameplay coverage.

## Mana ability activation payment plans

`ManaTap.ActivationPayment` is an optional `*ManaActivationPayment`, a fixed
array of counts indexed by color. `SolveMana` now searches all legal color
payments for each supported mana ability's activation cost. It saves the chosen
payment with that action's production choices. For example, a Celestial Prism
can spend red and green mana for its `{2}` activation and retain white mana for
a final `{W}{U}` cost. Generic, colored, hybrid and existing color conversion
payments use the same mana-pool legality rules. Spell-restricted mana cannot
pay an activation cost.

`ManaTap.SourceIncarnation` saves the discovered source's incarnation. Execution
checks that incarnation, current ability, production, controller, tap legality,
current reduced cost and saved payment before it spends mana or taps the source.
A phase cycle keeps the incarnation. A departed or returned source cannot execute
an old discovered plan. `Game.Clone` can execute the same plan independently.
Callers that omit either new field retain the previous behavior for that field.

The RON syntax and supported cost contexts do not change. Celestial Prism,
Apprentice Wizard and Fire Sprites reuse their existing full-record compilation
and the shared solver. This service is a prerequisite for repeatable non-tap
conversion abilities. Those abilities, mana bodies with activation history, and
Standing Stones' additional life cost still need planner support.

## Temporary declared triggered abilities

`TriggeredAbilityModification(PermanentRef, *GenericTriggered)
ContinuousModification` installs a declared battlefield trigger in layer 6.
Use it with `Until`, source-bound durations, saved-player upkeep durations, or
`While`. The subject uses the outer context; the trigger has an independent
permanent-source and binding scope. Spell grants cannot read the spell's source
as a permanent. Granted bodies cannot read outer targets or casting facts.
Nil declarations, unbound references, legacy event declarations and triggers
from other zones fail definition validation.

Each installation captures recipient incarnation and supplies a separate ability
object and stable ability UUID. Layer reapplication does not accumulate triggers.
The recipient's controller at detection controls the trigger. Copies of the
source effect install separate grants. Phasing suspends the grant and preserves
identity; leaving and returning ends its application. Pending triggers survive
expiry and ability removal, with their original event frame. Existing layer
ordering, durations, APNAP, target legality and cloning services apply.

Evidence: `TestTemporaryTrigger*`, `TestRONTemporaryRampage*`, existing
`TestGrantedTrigger*` and `TestRampage*`. Gabriel Angelfire's four choices and
saved-player next-upkeep duration, and Rapid Fire's casting restriction,
resolution-time rampage condition and end-of-turn grants have complete RON tests.

## Permanent-source next-event prevention prerequisite

`PreventNextDamageFromPermanent(subject PermanentRef, recipient PlayerRef,
filter PermanentFilter) ResolvingEffect` installs one shield until cleanup.
The subject must be present and pass the filter when the effect resolves.
The recipient is saved at resolution. Targets and saved references use their
existing declaration and incarnation checks.

The next matching damage event to that player is prevented in full. Current
source properties are checked again; departure or phasing uses the retained
view for that exact incarnation. A returned card cannot use or consume the old
shield. Nonmatching and zero damage do not consume it. All matching packets
from one simultaneous event are prevented with one use. Separate installations,
copied activations and cloned games keep separate consumption state.

The zero filter matches all permanent sources. Filter callbacks must read the
supplied permanent, rather than look up its UUID again. This prerequisite does
not choose a source or support spell-to-permanent source continuity. RON
`ChooseADamageSource` and its continuation remain rejected. The existing UUID
`Game.AddSourcePrevention` installer retains its caller behavior.

### Engine planning prerequisite: repeatable fixed mana conversion

The existing `Activate(Costs(ManaPayment(...)), AddMana(...))` engine API can
participate in automatic payment without a tap cost when the body is fixed and
mana-only (or includes the deferred history shape below), and the reduced cost
consumes at least its produced mana count.
The source may be tapped or summoning-sick. Automatic planning excludes X costs,
activation limits, arbitrary activation conditions, targets, non-mana costs and
mixed bodies except history-gated delayed instructions. Explicit activation
retains the existing broader behavior.
No tap bonus or tapped-for-mana event is generated for a conversion.

RON vocabulary is unchanged. History-bearing fixed-color conversions now use
the synchronous execution contract below.
`TestRepeatableMana*` and `TestRONRepeatableManaPayment` establish this engine
prerequisite; they do not establish full Farrelite Priest or Initiates of the
Ebon Hand gameplay support.

### Fixed mana production with activation history

RON supports nonmodal `Activated` bodies with an `ActionList` containing fixed
`AddMana(ManaProduceW|ManaProduceU|ManaProduceB|ManaProduceR|ManaProduceG|ManaProduceC)`
actions and `If(TimesThisAbilityHasBeenActivatedThisTurnIs(
GreaterThanOrEqualTo(Integer(n))), [CreateFutureTrigger(event, ActionList(...))])`.
The threshold is nonnegative. Each branch contains one or more supported delayed
step instructions. Existing step-event and delayed-body restrictions apply.
Immediate branch actions, conditional production, any-color production, extra
immediate actions and history reads inside delayed bodies are rejected.

Farrelite Priest and Initiates of the Ebon Hand each pay `{1}`, produce one fixed
mana, and create a next-end-step sacrifice delay at four or more activations.
Explicit activation and automatic payment share the same per-instance history.
Each successful activation resolves immediately without entering the stack.
The planner admits these bodies under its existing cost and timing restrictions,
reads fixed output without executing effects, then executes the complete body
through normal immediate resolution after payment. No new public API is needed.
`TestHistoryMana*` and `TestRONHistoryMana*` cover payment, event timing, thresholds,
instance/clone isolation, cleanup, departure, control, phasing and delayed timing.

## RON attached permanent death bindings

`TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)), body)` uses
`mage.AttachedPermanentDies()`. The source must be an Aura. The event matches
its attached incarnation moving from the battlefield to a graveyard, including
simultaneous Aura/host departure. It does not require the host to be a creature.
`mage.AttachedCreatureDies()` retains its existing creature restriction.

Each factory builds separate typed bindings. Within this trigger body,
`ControllerOfDeadPermanent` and `OwnerOfDeadPermanent` are saved player references,
`ToughnessOfDeadPermanent` is a saved number, `Trigger_ThatPermanent` is the
original departing permanent, and `Trigger_ThatGraveyardCard` is its first
graveyard visit. Callable symbols and reads outside this body are compile errors.
Existing player actions, permanent-source damage and graveyard-card actions can
read these bindings. Source-death and general filtered death contexts do not
receive the new host bindings.

`DeadGuestPermanentDealsDamage(value, recipient)` is a resolving action in this
host-death body. It lowers to `DealDamageFrom(DamageSourceOf(This), recipient,
value)`: the Aura deals the damage. The amount uses supported instance values;
the recipient uses the existing typed singular damage-recipient declarations.
It declares no target of its own. Values from the host death are captured at
detection, not recalculated from a returned card or a later attachment.

Creature Bond has complete RON gameplay evidence in `TestRONHostDeath*`.
Puppet Master and Takklemaggot can use the event bindings but retain unsupported
payment, choice, return and Aura-change instructions. Their complete records
still fail strict compilation.

## RON fixed matching tap payments

`TapAPermanent(filter)` and `TapNumberPermanents(Integer(n), filter)` are fixed
activation costs; `n` must be positive. They use
`mage.TapNMatchingIncludingSourceCost(n, filter, text) Cost`. This constructor
uses the existing exact-count payment transaction and permits the source when
it matches. `TapNOtherMatchingCost` keeps its source exclusion.

The payer chooses distinct untapped matching permanents they control. Each
choice retains its incarnation. A selected permanent can have summoning sickness;
a separate `TapPermanent(ThisPermanent)` cost still checks the tap symbol rule.
All selected permanents tap before their tap events are sent. Failed payments
leave mana, life and taps unchanged. Queries do not make choices.

Filters can use the existing fixed permanent filters and nonempty conjunctions.
A conjunction can contain one `ControlledByAPlayer(SinglePlayer(You))`; the engine
already checks the payer's control. Other players, unbound references, duplicate
controller clauses, or controller clauses inside disjunctions are rejected.
These adapters do not provide paid-object history or resolution tap payments.
Hand of Justice has complete RON gameplay tests. Vodalian War Machine's two
activation clauses have tests, but its death rule still needs paid-object history.


## RON activation permission for any player

`ActivatedWithModifiers(cost, actions, OtherPlayersMayActivate(AnyPlayer))`
permits either player to activate the ability. This modifier also works inside
nested `And([...])` modifier lists and with `Modal_ChooseOne` action bodies.
It lowers to `mage.WithAnyPlayerMay()` on each fresh action definition.
The compiler requires one bare `AnyPlayer` symbol and rejects duplicate
permissions, including duplicates in nested modifier lists. Other player
filters, top-level rules, action/cost uses and the
separate conditional-mana activation context remain unsupported.

The activator pays, chooses modes and targets, controls the stack ability and
supplies resolving `You`. Target controller restrictions are checked against
that player at announcement and resolution. Source characteristics and damage
ownership use the original source's current or retained information, rather
than the stack ability's controller. Permission does not bypass payment,
phasing, activation limits, timing or target legality.

`TestRONActivationPermissionEfreet` proves the complete Ifh-Bíff Efreet ARN
record, native casting, both activators, payment rollback, flying recipients,
shroud, phasing and simultaneous damage. `Identity` covers source control,
departure/return, phase cycles, ability removal, lifelink, copies and clones.
`ActorsAndTiming`, `ModalChoices`, `IndependentInstances`, `CostRollback`,
`TargetsAndPresence`, `EfreetPreventionAndMembership`, `Invalid` and
`StrictLoad` cover the remaining permission contracts and atomic loading.
Land's Edge shares the supported permission and discard cost, but its full
record still rejects `DiscardedCardPassesFilter`. It has no complete gameplay
mapping from this slice.

### RON grouped source combat events

`TriggerA` supports this exact event union, in either order:

```text
Or([
  WhenACreatureBlocksAnyNumberOfCreatures(SinglePermanent(ThisPermanent), IsCreatureType(S)),
  WhenACreatureBecomesBlockedByAnyNumberOfCreatures(SinglePermanent(ThisPermanent), IsCreatureType(S))
])
```

Both entries must use the same source and creature subtype. The compiler lowers
this union to `BlocksOrBecomesBlockedBy(This).OfSubtype(S)`. One matching new
group creates one trigger, even when multiple members match. These forms do not
support a standalone direction, an Aura host, or a singular opponent binding.

`WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent))` lowers to
`BecomesBlocked(This)`. It fires on the source's first transition to blocked,
including a transition caused by entry of a blocker. Later blockers do not
repeat this event. Both adapters permit `Trigger_ThatCreature` as the original
source in the resolving body. They do not supply `Trigger_DefendingPlayer` or
an opposing-creature reference. Each factory builds fresh actions and abilities.

Dwarven Soldier has complete RON gameplay coverage in `TestRONGroupedCombat*`.
Spitting Slug still rejects its unsupported `MayCost`/`IfElse` continuation;
this event adapter does not establish payment branches or collection grants.

## RON two-branch fixed mana payments

In resolving action sequences, adjacent `MayCost(PayMana([...]))` or
`PlayerMayCost(player, PayMana([...]))` and
`IfElse(condition, [true_actions], [false_actions])` form one payment operation.
The condition is the bare symbol `CostWasPaid` or `CostWasntPaid`. Both lists
must contain at least one supported action. Each branch can contain another
payment pair. The result cannot be read outside the pair. Fixed costs include
an explicit zero cost; X, mandatory payments and nonmana costs in this
three-argument continuation remain errors. Existing one-branch forms retain
their contracts. The adapter lowers to `mage.PayOrElse` with typed player and
body references, built separately for each factory instance.

`CreateEachPermanentLayerEffectUntil(collection, [AddAbility([keywords])],
UntilEndOfTurn)` supports nonempty lists of supported temporary keywords.
The collection uses the contextual collection grammar, including nonempty
`And`/`Or` and typed `IsBlockingAttacker(permanent)` and
`IsBlockedByDefender(permanent)`. These relations use current combat pairs.
An absent, phased or returned source cannot restore a pair through LKI.
Membership is captured when the grant begins. Leaving combat does not end it;
a recipient's zone change ends it, and cleanup removes it. Mixed keyword/P/T
modifications and other durations remain errors in this shape.

`TestRONPaymentBranches*` maps complete Spitting Slug RON to casting, both combat
triggers, multiple blockers, payment, first-strike damage, duration, identity,
phasing, copies and cloning. Synthetic tests cover nested results, polarity,
target legality, copy retargeting and invalid declarations. This does not claim
complete Cyclone, Imprison, Bronze Tablet or Transmute Artifact support.

## RON activated ability targets

`Targeted([TargetAbility(And([ActivatedAbility,
AbilityOfASource(IsCardtype(Artifact))]))], ActionList([CounterAbility(Ref_TargetAbility)]))`
declares one activated ability target. The two filter operands can appear in
either order. Exactly one bare `ActivatedAbility` and one
`AbilityOfASource(IsCardtype(type))` are supported. The source type uses the
existing card-type grammar. Other ability/source filters remain errors.

`ControllerOfAbility(Ref_TargetAbility)` is a typed player reference. It reads
the controller of the ability on the stack, independently of source control.
It can supply the payer of `PlayerMayCost`; the existing adjacent `Unless` or
`If` payment continuation selects its branch at resolution. `CounterAbility`
requires a bound ability reference in the same rule. Spell, permanent, delayed,
undeclared and cross-rule references are rejected. Factories build fresh target
bindings. Stack objects store their own choices; copies can select other targets.

The engine API is `DeclareActivatedAbilityTarget(name string, sourceType
core.CardType) *AbilityTargetBinding`, `ControllerOfAbility(AbilityRef) PlayerRef`,
and `CounterAbility(AbilityRef) ResolvingEffect`. The compiler imports these
services directly from `mage`; no legacy DSL export is required. Rust and Ayesha
Tanaka use their complete LEG RON records. Trigger-event ability references,
including Imprison's `Trigger_ThatActivatedAbility`, remain unsupported.

## RON source departure events and live exile collections

`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)), body)`
is supported on permanent definitions. The event and `SinglePermanent` each take
one argument. The source must be the bare `ThisPermanent` symbol. Other subject
filters and Aura-host events remain unsupported. The compiler uses
`WhenEvent(LeavesBattlefield(This), ...)` and creates a fresh ability per factory.
Within this body, `Trigger_ThatPermanent` is a typed reference to the original
source. It has the same incarnation and LKI behavior as `ThisPermanent`; it is
unbound in a separate rule. It supplies no card reference in the destination zone.

`Exile([EachPermanent(collection)], [])` supports the shared contextual collection
compiler in resolving actions. This includes permanent filters, nonempty `And`
and `Or`, controller/owner selections, combat-pair collections and attachments.
A declared `Ref_TargetPermanents` retains its typed target set. Unbound target
sets, malformed filters and exile modifiers are rejected. `ExileEach` captures
live collection members at resolution and uses one zone-change batch. This action
has no target unless the surrounding action declares one.

Complete-card mappings: `TestRONSourceLeaveShrine*` covers Goblin Shrine's
conditional basic-Mountain bonus and departure damage. `TestRONSourceLeaveHazezon*`
covers Hazezon Tamar's entry trigger, next-upkeep tokens and departure exile.
`TestRONSourceLeaveCasting`, `IndependentEvents`, `Invalid`, `Inventory` and
`StrictLoad` cover payment, declarations, independent pending events, role
reporting, fresh factories and atomic rejection. Existing `TestExileEach*` tests
supply the reusable batch, identity, phasing and token contracts.

## RON ordinary draw-step triggers

Permanent definitions support
`TriggerA(AtTheBeginningOfAPlayersDrawStep(players), actions)` with exactly one
event argument: bare `AnyPlayer` or `SinglePlayer(You)`. The existing action
compiler supplies the body; this declaration does not add missing body operations.
`Trigger_ThatPlayer` is a typed player reference local to that body. Each factory
creates a separate binding, and each pending event saves its player. `You` reads
the resolving ability controller. The controller filter reads the source at
detection; it is not checked again at resolution.

The adapter reuses `compileBoundStepAbility`, `TurnStepBegins(EachDrawStep, binding)`
and `EventPlayerIsController` for the `SinglePlayer(You)` restriction. It has no
intervening-if condition. Other players, malformed declarations, cross-rule
reads and instant/sorcery rule contexts fail compilation. Detection precedes the
normal draw; priority and trigger resolution follow it. Phased sources cannot
trigger, but departure or phasing after detection does not cancel a pending rule.
Copies retain the event player and use their own controller; clones keep separate
resolution state.

`TestRONOrdinaryDrawStep*` covers declarations, bindings, copies, clones, source
lifecycle and native timing. Armageddon Clock's unchanged RON event is tested
with an explicit probe body. Its complete record still rejects the source-counter
value. Sylvan Library still rejects optional drawing payment and has a separate
source-data defect in its drawn-card disposition. Neither has a complete-card
mapping from this prerequisite. Existing `TestRONDrawStep*` mappings for Howling
Mine and Mana Vault remain valid.

## RON counter information values

`TheTotalNumberOfCountersOfTypeAmongPermanents(counter, SinglePermanent(ref))`
is supported by the resolving value compiler. `counter` uses the validated
permanent counter vocabulary. `ref` is a typed source, Aura host, singular target,
or permanent supplied by the current event. A reference must be bound in that
rule and context. Collections and static, entry, cost and announced-X contexts
are not established by this adapter.

The adapter uses `mage.CountersOnWithLKI` with each factory's own bindings.
It reads current counters at execution or retained information for the exact
incarnation after departure or phasing. It cannot restore an absent target's
legality. A prior instruction can move a legal target and a later information
read can still use its retained counters. A sacrifice cost uses departure
information. No mutable count is stored in the compiled plan.

Complete-card evidence: Armageddon Clock, in ATQ and 3ED. Tests with prefix
`TestRONCounterInformation` cover its casting, upkeep placement, any-player
upkeep activation, payment rollback, countered payment, removal, draw timing,
simultaneous damage, prevention, copies, clones and incarnation changes.
Other consumers remain rejected at independent gaps. See
[the counter information contract](engine-rules-contracts.md#counter-information-values).

## RON restricted activation players

`OnlyPlayerMayActivate(OwnerOfPermanent(ref))` and
`OnlyPlayerMayActivate(ControllerOfPermanent(ref))` are activation modifiers.
`ref` must be the bare `ThisPermanent` symbol or, for an Aura,
`HostPermanent`. They work in ordinary and modal `ActivatedWithModifiers`
abilities and nested modifier `And` lists. Exactly one permission modifier is
permitted, including when combined with `OtherPlayersMayActivate`.
Announcement targets and event bindings cannot supply the permission subject.
Other player references, action/cost/rule uses and conditional-mana activations
remain unsupported. This does not add resolving `OwnerOfPermanent` support.

Lowering combines `WithAnyPlayerMay()` and `WithActivationStateCondition()`
with `PlayerIsPermanentOwner(PermanentRef) StateCondition` or
`PlayerIsPermanentController(PermanentRef) StateCondition`. These engine
conditions accept `This` and `AttachedPermanent`, compare
`StateContext.ControllerID`, and require the current source and attachment
incarnations. They use live information; absent or phased objects cannot grant
permission. Control and attachment changes affect future activations.

The eligible player pays costs, chooses modes and targets, controls the stack
ability, and supplies resolving `You`. Timing, limits and target legality still
apply. Permission is not repeated during resolution or for a stack copy.
No mutable choice is stored in the condition. Engine declaration validation
rejects nil, target and saved-binding subjects before gameplay.

Evidence: `TestActivationPlayerConditionIdentity` and `Declarations`;
`TestRONOnlyPlayerActivationRealModifierProbes`, `PresenceAndReattachment`,
`TimingTargetsAndInstances`, `RollbackAndModes`, `Inventory`, and
`InvalidAndBoundaries`. The probes retain the unchanged permission nodes from
Merseine and Personal Incarnation, with explicitly synthetic costs and bodies.
Neither complete card is supported: Merseine still needs
`PayManaCostOfPermanent(HostPermanent)` and Personal Incarnation still needs
`CreateFutureReplaceWouldDealDamage` and its independent death body.

### Counted and collection next-untap actions

Use `mage.SkipNextUntaps(subject, count)` for a fixed nonnegative number of
future untap steps of the subject's current controller. Use
`mage.SkipNextUntapEach(subjects)` to capture a typed collection at resolution
and suppress each member's next controller untap. Neither action taps subjects
or prevents untaps from other effects. Repeated restrictions overlap; they do
not add steps. Control changes and phasing preserve remaining steps. Departure
ends the restriction, and clones keep separate counts.

RON supports resolving
`PermanentDoesntUntapDuringControllersNextNumberUntaps(ref, Integer(n))` and
`EachPermanentDoesntUntapDuringControllersNextUntap(filter)`. Counts must be
nonnegative literals. References use the existing instance scope; filters use
contextual collection queries. These are actions, not costs or static rules.
Complete gameplay mappings cover Telekinesis and Spore Cloud. See the counted
untap contract in `engine-rules-contracts.md` for timing and identity evidence.

## RON resolution counter payments

Resolving permanent-source action lists accept `MayCost`, `PlayerMayCost`
and `MustCost` with `RemoveCounters([instruction])`, followed immediately by
`If` or `Unless` with `CostWasPaid` or `CostWasntPaid` and a nonempty action
list. Fixed instructions are `ACounterOfTypeFromPermanent(counter,
ThisPermanent)` and `NumberCountersOfTypeFromPermanent(Integer(n), counter,
ThisPermanent)` with nonnegative n. Optional payments also accept
`AnyNumberOfCountersOfTypeFromPermanent(counter, ThisPermanent)`.

A chosen payment's paid continuation can read the symbol
`TheNumberOfCountersRemovedThisWay`. The compiler makes a fresh NumberBinding
per factory instance and payment. The value is unavailable before payment,
after the continuation, or on its unpaid path. Fixed resolution payments do
not produce this value. Spell sources, other subjects, mandatory chosen counts,
multiple instructions, dynamic fixed counts and IfElse continuations remain
rejected. Existing activation counter payments retain their contract.

Lowering uses `PayCountersOrElse` or `PayChosenCountersOrElse` and the engine's
shared transaction. The payment occurs at resolution, with presence, phasing
and original-incarnation checks. It cannot use LKI to remove counters from an
absent object. Copies pay separately; clones keep separate state.

Living Artifact's six complete reprints compile and pass definition validation.
Its gameplay mapping covers enchant/casting costs and legality, actual damage
counter accumulation, controller upkeep timing, decline, insufficient counters,
source identity, control, phasing, copied triggers and clones. Cocoon and
Tetravus still fail strict compilation at their remaining rule adapters.
Synthetic mandatory and token-count tests establish the payment prerequisite;
they do not establish either card's complete gameplay. Evidence:
`TestRONResolutionCounter*` and `TestResolutionCounterPayment*`.

## RON untargeted activated ability grants

A static `PermanentLayerEffect(subject, [AddAbility([Activated(cost, body)])])`
can grant to `ThisPermanent` or an Aura's `HostPermanent`.
`EachPermanentLayerEffect(filter, [AddAbility([Activated(cost, body)])])` uses
an existing static permanent collection. A resolving
`CreatePermanentLayerEffect(reference, [AddAbility([Activated(cost, body)])])`
captures one typed permanent for an indefinite grant.

Each form requires exactly one modification and one `Activated` declaration.
The cost uses existing fixed source-cost adapters; X costs are rejected.
The body must be a nonempty, untargeted `ActionList` in a fresh permanent-source
scope. `ThisPermanent` and `You` refer to the recipient and activating player.
Donor targets, attachment references, events and payment results cannot enter
that scope. Targeted bodies, modes and `ActivatedWithModifiers` in the granted
declaration remain unsupported. The outer Life Matrix activation can still
have its own target and upkeep restriction.

The adapter uses `GrantActivatedAbility`, `GrantEachActivatedAbility` and
`ActivatedAbilityModification`. Each factory creates its own costs and body.
Static grants follow current membership; resolving grants keep captured
incarnations. Zombie Master and Life Matrix have complete RON test mappings
in `activated_grant_test.go`. Equinox still needs a targeted grant and a
spell-outcome condition.

## RON static attack payments

`PermanentRuleEffect(subject, [CantAttackUnlessCost(cost)])` supports
`ThisPermanent` on a permanent and `HostPermanent` on an Aura. The operation
requires exactly one `PayMana(list)` with fixed mana symbols or
`SacrificeNumberPermanents(Integer(n), filter)` with positive fixed `n` and an
existing sacrifice filter. X, combined costs, target references, other cost
forms and multiple operations remain compile errors in this context.

Each factory creates a fresh cost. The adapter uses `SourceCantAttackUnlessPays`
or `AttachedCantAttackUnlessPays`; payment uses `Game.PayAttackCosts` during
attack declaration. The attacker controller pays, even when an Aura has another
controller. Multiple live rules contribute separate costs to one transaction.
The rule follows current attachment, source abilities and phasing. It does not
use retained information to tax a former host. Casting and resolving the card
do not pay the attack cost. Leviathan's optional upkeep payment is separate.

Complete Brainwash and Leviathan records are mapped by `TestRONAttackCost*`:
casting and rollback, entry and upkeep, native attack choices, trample damage,
control, attachment changes, phase cycles, returned objects, copies, independent
clones, aggregate payment and invalid or stale sacrifice choices. Inventory
visits the operation's operand as a cost, including rejected descendants.

Attack payment may sacrifice the attacker when it matches the cost filter.
Both declaration paths require the paid source's original incarnation and
current control before they put it in combat. An animated Leviathan that is
also an Island can pay with itself; it then remains absent from combat.

### RON delayed departure actions through end of turn

`CreateTriggerUntil(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ref)),
ActionList(actions), UntilEndOfTurn)` and the corresponding
`WhenAPermanentDies(SinglePermanent(ref))` form lower to `WhenLeavesThisTurn`
and `WhenDiesThisTurn`. They use resolving spell, activation, trigger and nested
supported delayed action scopes. `ref` must be a bound singular permanent
reference. The body uses the existing delayed action restrictions, captures the
outer bindings and declares no new targets. Delayed death-event bindings are
described below. Leave-event bodies do not declare departure-event bindings.

The duration is the bare `UntilEndOfTurn` symbol. Other events, durations,
plural subjects, wrong arities, unbound reads, empty bodies, modes and new body
target declarations remain compile errors. This shape does not add generic
`CreateTriggerUntil` event support. War Barge and Sandals of Abdallah use the
complete records, with their separate regeneration restrictions.

## RON hand and continuous-control activation restrictions

`ActivateOnlyIf(PermanentPassesFilter(subject,
ControlledSinceBeginningOfMostRecentTurn))` accepts a bare predicate with
`ThisPermanent` or an Aura's `HostPermanent`. It uses
`mage.ControlledSinceTurnStart(subject)`. The subject must remain present,
phased in, and under the activating player's control. An opponent's turn or
extra turn cannot grant permission before the controller's next turn begins.
Control changes and zone changes restart the requirement; phasing preserves
control history. Resolving and static filters do not accept this predicate.

`ActivateOnlyIf(PlayerPassesFilter(You,
NumCardsInHandIs(EqualTo(Integer(n)))))` accepts a nonnegative literal `n`.
It uses `mage.ControllerHandSizeIs(n)` before payment. The activating player's
current hand supplies the count, including when another player owns the source.
Other comparisons, dynamic counts, and other player references remain errors.
Both restrictions combine with existing modifiers and use the shared pure check
for mana planning. Stack copies and resolution do not repeat the check.

`TestRONActivationHistory*` maps the complete Library of Alexandria and Rocket
Launcher records to gameplay and declaration tests. Casting, land play, tap/mana
payment, target legality, delayed destruction, copies, clones, source return,
phasing, and current controller behavior use their existing engine services.

## Spell damage-source identity prerequisite

The engine APIs are:

```go
func (g *Game) CaptureSpellDamageSource(stackID uuid.UUID) DamageSource
func DamageSourceOfSpell(spell StackObjectRef) DamageSourceRef
func SnapshotDamageSource(source DamageSourceRef, binding *DamageSourceBinding) ResolvingEffect
func (s DamageSource) StackID() uuid.UUID
func (s DamageSource) SameObject(other DamageSource) bool
```

Capture uses a live stack object ID, including permanent spells, and rejects
abilities or absent spells. `DamageSourceOfSpell` accepts an existing typed spell
target or saved spell binding. Snapshot saves independent information in the
existing resolution environment; `FromDamageSource` can read it after departure.
The normal definition validator checks declarations and producer dominance.
An unsuccessful snapshot replaces the binding with an empty source.

Native spell damage retains its exact stack ID while resolving. A saved source
keeps identity, controller, owner, effective colors and card types through
departure and cloning. `SameObject` distinguishes separate casts, copied spells
and permanent incarnations. It does not link a permanent spell to the permanent
it becomes; that future prevention contract needs an explicit link. A source
without a stack ID or permanent incarnation cannot match by UUID alone.

These APIs are engine services. No new RON shape is accepted. In particular,
`ChooseADamageSource` remains rejected until enumeration, source bindings,
property rechecks and shield installation are complete. Existing callers and
the permanent-source prevention API remain available.


## RON attack history references

`AttackedThisTurn` and `DidntAttackThisTurn` are bare permanent-filter symbols.
They are supported in existing permanent-filter contexts, including target
filters and battlefield collections, and in resolving
`If(PermanentPassesFilter(reference, filter), [actions])` branches. Nonempty
`And`/`Or` branches retain the existing scope rules. A delayed branch keeps its
original permanent reference through `AtNext`; it reads history when the delayed
body resolves. No new intervening-if, static-condition or player-collection
filter context is added.

`TargetPlayer(AttackedThisTurn)` declares one player target. It lowers to
`DeclarePlayerTarget(name, TargetPlayerAttackedThisTurn())`; either player can
qualify. The engine checks the restriction at announcement and resolution.
Tuple forms, arguments, `TargetPlayer(DidntAttackThisTurn)` and unbound or
incompatible permanent references remain declaration errors.

The engine filters `AttackedThisTurn` and `DidntAttackThisTurn` read declaration
history for an exact permanent incarnation. `IfPermanentMatches` can read that
history through current or retained information. Absence does not permit an
action on that object. A returned card has separate history, even when both
incarnations attack in the same turn. Phasing and combat exit do not erase
history. A creature put into combat without declaration did not attack.

`TargetPlayerAttackedThisTurn()` composes with `TargetOpponentOnly()` in either
order. It uses the existing player target and turn tracker, without a separate
choice service. Targets and delayed bindings remain private to each factory;
copies and cloned games retain their existing identity contracts.

Evidence: `TestAttackHistoryReferenceIdentity`,
`TestAttackHistoryReferencePhasingAndCombat`, `TestAttackHistoryPlayerTarget`,
`TestAttackHistoryBothDeclarationPaths`, `TestRONAttackHistoryBerserk`,
`TestRONAttackHistoryFireAndBrimstone`, `TestRONAttackHistoryInvalidAndInventory`,
`TestRONAttackHistoryBerserkTimingAndLegality`,
`TestRONAttackHistoryBerserkCopyAndExpiry`,
`TestRONAttackHistoryFireAndBrimstoneBatch`, and
`TestRONAttackHistoryFilterContextsAndInstances`.
Erg Raiders still needs its source intervening-if and continuous-control
conditions. Season of the Witch still needs its inability-to-attack history
and exclusion filter. Their full records remain rejected.

## RON filtered compulsory blockers

RON supports
`PermanentRuleEffect(ThisPermanent, [AllCreaturesMustBlockIt(filter)])` and,
for an Aura with an enchant declaration,
`PermanentRuleEffect(HostPermanent, [AllCreaturesMustBlockIt(filter)])`.
The operation has exactly one permanent-filter argument. It uses the existing
filter compiler, including supported nonempty conjunctions/disjunctions.
Unbound targets, non-Aura hosts, malformed operations and incompatible filters
remain declaration errors. No temporary, collection, cost or entry context is
added by this adapter.

The compiler lowers the operation to
`Static(While(condition, AllCreaturesMustBlock(subject, filter)))`.
The engine owns live attachment identity, rule storage and the maximum legal
blocking assignment. It uses both native and search declaration paths and does
not redirect every block to the first compulsory attacker. Lure requires all
able creatures to block its current host. Marble Priest requires only able
Walls and retains its separate combat-damage prevention rule.

For Go engine consumers, `AllCreaturesMustBlock(PermanentRef, PermanentFilter)`
can also use the existing `Until` duration contract. The pure
`Game.CompleteBlockAssignments(uuid.UUID, []BlockAssignment) []BlockAssignment`
query keeps legal maximum proposals and completes unmet requirements without
writing combat state. The engine contract documents capacity, bands, minimum
blocker counts, identity and search behavior. Evidence is in
`TestBlockingRequirement*` and `TestRONBlockingRequirement*`.


## RON temporary collection animation and equal base values

`CreateEachPermanentLayerEffectUntil(collection,
[SetPT(PT(integer, integer)), AddCardtype(Creature)], UntilEndOfTurn)` accepts
exactly one fixed P/T pair and one Creature addition, in either order. Collections
use the existing contextual collection compiler, including typed controller,
intersection and union queries. Both operations share one `Until` installation.
It captures current members and their incarnations at resolution; later entries
and phased-out members do not join. Prior types, subtypes, colors and abilities
remain. This is the complete Thelonite Druid record, including its transactional
mana, tap and creature-sacrifice costs; the source can pay its own sacrifice.

In `CreatePermanentLayerEffectUntil/3`,
`SetCardtypes([Artifact, Creature])` accepts exactly those two types once each,
in either order. It lowers to two type additions because becoming an artifact
creature retains prior types under CR 205.1b. Other type-setting lists remain
errors; this does not establish general type replacement.
`SetPowerAndToughnessBoth(value)` lowers to the existing base-value operation
with the temporary-value compiler. Its supported values are fixed integers,
bound announced X, subtraction of supported values, and typed permanent
power/toughness or mana-value reads. `ActionPermanent` means this modification's
subject. A mana-value read must refer to that subject. The engine captures values
at resolution, rather than recalculating them on each layer pass.

Xenic Poltergeist uses these operations with the existing
`UntilNextUpkeep(You)` duration. Its noncreature-artifact target is checked at
announcement and resolution; the saved caster/controller selects the expiry
upkeep. Source departure does not end the installed effect. These additions
apply only to resolving temporary modifications, not static, entry or cost
contexts. Invalid references and declarations fail compilation.

Evidence: `TestRONExtendedAnimationCards`, `XenicLifetime`, `DruidMembership`,
`Copy`, `Invalid` and `StrictLoad` in `extended_animation_test.go`.

## RON numbered singular permanent groups

`Targeted([TargetPermanent(filter1), TargetPermanent(filter2), ...], actions)`
accepts two or more ordinary singular permanent declarations. The resolving body
uses `Ref_TargetPermanent1`, `Ref_TargetPermanent2`, and subsequent one-based
names. Each declaration has a separate typed plan and each factory call builds
separate engine target bindings. Single declarations retain the unnumbered
`Ref_TargetPermanent` spelling. A multiple-declaration body cannot use that alias.

This shape works in existing spell, activation, triggered and mode-local action
contexts. Existing delayed bodies capture each numbered reference separately.
Filters reuse the singular permanent target compiler. Each group requires one
legal permanent at announcement. A permanent can be chosen once in each separate
group when it meets both filters. The normal target service checks each group
again at resolution. Legal groups still act when another group is illegal; all
illegal groups prevent resolution. An action cannot act on an illegal target.
Copies can retarget each group, and clones keep their own resolving frames.
Delayed actions retain the captured incarnation and existing information reads.

Empty lists, mixed declaration kinds, plural groups, opponent-choice declarations,
unbound or malformed numbered references, mode-scope leaks, nested declarations,
and conflicting `You`/`Opponent` target controller restrictions are rejected.
This adds no dependent target filter, exchange, fight or graveyard group adapter.
Gauntlets of Chaos and Glyph of Delusion now reach their dependent-filter gaps;
Arena, Cuombajj Witches and Drafna's Restoration still need other target forms.
No complete-card gameplay mapping is added by this prerequisite.

Evidence: `TestRONSingularGroupsGameplay`, `InstancesAndRepeatedObject`,
`CopyRetarget`, `DelayedLKI`, `TriggeredAndModal`, `Invalid`, and
`StrictLoadAndConsumers`. Execution reuses `DeclarePermanentTarget`, `WithTargets`,
existing target legality, payment transactions, saved references and stack copies.

## RON opponent-chosen permanent targets

`TargetPermanentOfAPlayersChoice(filter, Opponent)` and
`TargetPermanentOfAPlayersChoiceTheyControl(filter, Opponent)` declare one
permanent target in `Targeted`. The second form requires the opponent to control
that permanent. The first form permits any matching permanent. The chooser
argument must be the bare `Opponent` symbol. Other chooser references and
`TargetPermanentOfPlayersChoice` remain unsupported.

These forms can occur alone or in numbered singular permanent groups with
ordinary `TargetPermanent` declarations. They use the existing target filters,
scopes, spell, activation, trigger and mode contexts. A single declaration uses
`Ref_TargetPermanent`; multiple declarations use one-based numbered references.
Conflicting controller restrictions fail compilation before factory construction.
Factories create separate typed target bindings for each declaration.

Lowering uses `DeclarePermanentTarget`, `TargetChosenByOpponent` and, for
`TheyControl`, `TargetOpponentControls`. The opponent chooses at announcement
or trigger placement. Target legality reads the action's controller and source;
the chooser does not bypass hexproof or other targeting restrictions. Targets
are checked again at resolution. Modal activation preserves the initial target
choice. Copy permissions use the existing copy service: the copy's controller
can choose new legal targets under CR 707.10c.

Evidence: `TestRONOpponentTargets*` and
`TestModalActivationOpponentChoosesOnce`. Preacher, Arena and Nova Pentacle have
source-record target-fragment tests and explicit next-blocker checks. Those
fragments establish targeting and original activation-cost behavior. They do
not establish complete-card gameplay or permit partial production loading.


## RON reference collections for resolving actions

`SinglePermanent(ref)` is a typed contextual collection operand. Symbolic
`Ref_TargetPermanents` uses its scoped numbered target group. Both can occur in
nonempty `And([...])` and `Or([...])` compositions with existing live filters,
controller/owner sets, attachments and combat relations. The action consumers
are `TapEachPermanent`, `UntapEachPermanent`, `DestroyEachPermanent`,
`DestroyEachPermanentNoRegen` and `PutEachPermanentIntoItsOwnersHand`.

These forms use `mage.SingletonPermanent(PermanentRef) PermanentSet`, existing
target sets, `IntersectPermanents` and `UnionPermanents`. The reference keeps its
identity and legality checks; collection selection does not add targets. Each
factory builds separate bindings. Action selection excludes absent and phased
members and uses existing batch services. Existing saved attachment and singular
delayed capture behavior remains in effect. Plural delayed target collections,
unbound or incompatible references and empty compositions reject. These forms
do not add support in costs, entry operations, static filters or damage sets.

Evidence: `TestSingletonPermanentSet*`, `TestRONReferenceCollection*`, and the
existing full RON tap/untap and contextual collection gameplay tests. Arena still
needs Fight; Volcanic Eruption still needs its actual destruction-result count.

### RON destruction counts

A resolving action sequence can pair `DestroyEachPermanent(selection)/1` or
`DestroyEachPermanentNoRegen(selection)/1` with its next action. That action can
read `TheNumberOfPermanentsThatDiedThisWay(IsCardtype(Creature))/1` or
`TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay(filter, AnyPlayer)/2`.
The creature count uses the characteristics before destruction. The graveyard
count requires the selection to guarantee the requested filter: the same
position-free declaration, or an operand of its nonempty `And` tree. A bound
`Ref_TargetPermanents` uses its declared target filter for this check. It retains
all target legality and incarnation checks. This proof permits the existing
engine count of all selected graveyard moves without ignoring a filter.

Each pair creates a fresh `NumberBinding`. Only the adjacent continuation can
read it. Reads before the producer, later actions, delayed bodies, mixed result
kinds, other death filters and restricted graveyard players are rejected. Each
factory instance has separate bindings. `Plus(left, right)/2` now composes two
supported resolution values with `AddValues`; announcement and static value
contexts do not gain this adapter.

Volcanic Eruption and Hellfire execute their complete collection records through
`DestroyEach`, `PermanentsPutIntoGraveyardThisWayInto`,
`CreaturesDiedThisWayInto`, `FromNumber` and the existing damage services.
No engine action or global result slot is added.

## Damage-source choice prerequisite

`mage.ChooseDamageSourceFrom(player, []mage.DamageSourceRef{...}, binding)`
saves a required non-targeted choice in `mage.BindDamageSource`. Use
`mage.FromDamageSource(binding)` in the subsequent resolving instructions.
Each reference supplies its own identity and information contract. The declared
list must be nonempty and contain only sources permitted by the calling rule.
An empty runtime list clears the binding. The operation copies its input slice,
keeps distinct spell stack objects, and removes exact duplicate sources.

Players can implement `mage.DamageSourceChooser` to choose among typed source
snapshots. Existing players use numbered options. Choice values remain in the
existing resolution frame and survive game cloning without sharing mutable
snapshot data.

RON `ChooseADamageSource` remains unsupported. Its adapter needs complete
CR 609.7a enumeration, property filters, lexical bindings and the consuming
shield/replacement operations. This prerequisite does not change the production
loader, accepted records, or existing UUID-based callers.

## Typed damage-source property filters

`ChooseDamageSourceMatchingFrom(player PlayerRef, sources []DamageSourceRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect`
filters an explicit legal list before the existing required choice. The caller
must supply the complete legal list. The zero filter permits any valid source.
`DamageSourceHasColor(Color)` accepts one of the five Magic colors;
`DamageSourceHasType(CardType)` accepts a valid engine card type.
`AllDamageSources(...DamageSourceFilter)` and
`AnyOfDamageSources(...DamageSourceFilter)` require nonempty operands and copy
the supplied slice. Invalid colors, types and nested declarations fail engine
validation. `Match(DamageSource) bool` is a pure information query; a missing
source never matches. The original unfiltered API remains available.

Choice evaluates each reference at resolution. Live references supply current
properties; saved/event references supply their retained information. The filter
does not make a fresh battlefield lookup by card ID. Exact duplicate removal,
source snapshots, local results, no-choice clearing, player compatibility and
cloning use the existing choice service. The source need not deal damage.

RON now validates the filter argument of `ChooseADamageSource/1`:
`AnyDamageSource` as a symbol, `IsColor/1`, `IsCardtype/1`, and recursive nonempty
`And([filters])/1` or `Or([filters])/1`. The RON card types use the current
compiler vocabulary. Nested semantic inventory assigns source-filter, color
and card-type roles even inside rejected records. The choice action still
rejects after filter validation: complete CR 609.7a enumeration, per-action
bindings and complete consuming shields remain prerequisites. Filter validation
alone cannot produce a partial production registry.


### RON combat defending player targets

For singular permanent targets in instant/sorcery `SpellActions`,
`ControlledByAPlayer(SpellDefendingPlayer)` selects current permanents controlled
by the nonactive player during combat. The `SinglePlayer(SpellDefendingPlayer)`
wrapper is also supported. Either player can cast the spell. The target control
restriction does not follow the caster. Nested `And` filters, modal spell targets
and numbered singular target groups reuse their existing target declarations.
Outside combat the engine supplies no defending player, so no target qualifies.
Use the card's independent `CastEffect` to establish its exact casting window.

For source attack or unblocked-attack trigger targets,
`ControlledByAPlayer(SinglePlayer(DefendingPlayer))` uses the event's saved
player. The bare `DefendingPlayer` form is also supported in this target context.
The existing `ControlledByPlayer(Trigger_DefendingPlayer)` form is unchanged.
The player is available before target selection and stays fixed through copies,
source departure and later turn changes. Control, incarnation, phasing and target
protection are checked again at resolution.

RON spell-defender symbols are accepted only in these spell target filters;
event-defender symbols require the matching event scope. Other references,
malformed wrappers and tuple symbols remain declaration errors.
`CombatDefendingPlayer() mage.PlayerRef` and `mage.TargetControlledBy` supply
the reusable engine contract. RON keeps the syntax and scope checks.

Blaze of Glory, False Orders and Floral Spuzzem have other unsupported operations.
The target prerequisite does not make their complete records executable.
`TestRONCombatDefender*` maps the supported target contracts; the existing
`TestRONDefendingPlayer*` covers accepted full-card regressions.

## RON upkeep player conditions

`TriggerI(AtTheBeginningOfAPlayersUpkeep(AnyPlayer | SinglePlayer(You)),
condition, actions)` supports these player conditions:

- `PlayerPassesFilter(You | Trigger_ThatPlayer, ControlsA(permanent_filter))`.
- `APlayerPassesFilter(Opponent, ControlsMorePermanentThanPlayer(You, permanent_filter))`.
- `APlayerPassesFilter(AnyPlayer, SinglePlayer(ThePlayerWithTheMostLife))`.

Filters use the existing static-condition filter compiler. Empty conjunctions,
unbound players, tuple symbols and other selectors are rejected. The event
player and action body share one fresh binding per factory. The engine checks
current conditions at detection and resolution. A tie does not satisfy the
highest-life condition. An opponent count must be strictly greater; ownership,
LKI and phased permanents cannot supply current control.

Spiritual Sanctuary has a complete RON gameplay mapping in
`TestRONSpiritualSanctuary` and `TestRONSpiritualSanctuaryLifecycle`. Land Tax
still needs its multi-card search; Ghazbán Ogre still needs its indefinite
control action. Their condition probes do not establish complete gameplay.

## RON end-step source state and creature death conditions

In `TriggerI(AtTheBeginningOfAPlayersEndStep(players), condition, actions)`,
`players` can be the symbol `AnyPlayer` or `SinglePlayer(You)`. This slice adds:

- `PermanentPassesFilter(ThisPermanent, IsUntapped)`.
- `PermanentPassesFilter(ThisPermanent, DidntAttackThisTurn)`.
- `ACreatureOrPlaneswalkerDiedThisTurn(IsCardtype(Creature))`.

The first two filters are symbols. Other subjects, filter tuples, death filters,
and contexts remain unsupported. The spelling of the third condition does not
permit a planeswalker-only death filter. All declarations are checked before
factory construction.

Lowering uses `SourceTappedStateWithLKI(false)`,
`SourceAttackHistoryThisTurn(false)` and `CreatureDiedThisTurnCondition()` with
`TurnStepBegins` and `WithInterveningIf`. Each condition is read at detection and
resolution. The resolving source keeps its original incarnation. An absent source
can supply information but cannot receive counters or a bonus.

Osai Vultures has a complete gameplay mapping in `TestRONOsaiVulturesEndStep`,
`Activation` and `NativeSteps`. `TestRONEndStepSourceState` and `StateRecords`
provide prerequisite evidence for Erg Raiders and Voodoo Doll. Their complete
records still reject at resolving `Unless` and activation `XIs`, respectively.
`TestRONEndStepStateDeclarations` preserves strict declaration rejection;
`StateRecords` covers all four selected records, independent factories,
semantic roles and atomic loading. No complete gameplay claim is made for
those two blocked names.

### Typed player damage-history values

```go
mage.DamageTakenThisTurn(subject mage.PlayerRef) mage.ResolutionValue
mage.ArtifactDamageTakenThisTurn(subject mage.PlayerRef) mage.ResolutionValue
```

These pure resolution-time values read actual damage taken this turn. Prevention
and redirection affect the saved amounts; ordinary life loss does not. Artifact
classification uses the source's type at damage time. Later type or zone changes
do not change history. An absent player supplies zero. Definition validation
checks the typed reference. Clones keep separate totals, and cleanup clears them.

RON `DamageDealtToPlayerThisTurn(player)` uses the first value.
`DamageDealtToPlayerThisTurnByPermanents(player, IsCardtype(Artifact))` uses the
second. Other source filters remain errors. References use existing resolution
bindings, including target, scoped action and delayed player references. These
values do not supply casting costs, entry amounts or static characteristics.

### Static fixed mana increases

Use `mage.IncreaseSpellCostStatic(filter, increase, controllerOnly)` with
`WithStaticAbility` to add fixed generic or colored mana to matching spells.
The increase uses the live source controller when `controllerOnly` is true;
otherwise, it applies to all casters. Increases precede reductions in the shared
spell payment transaction. It accepts nonnegative generic and W/U/B/R/G counts,
including zero; X, hybrid mana and nil filters are invalid declarations.
Source departure, phasing and ability removal stop the tax. A returned source
starts a new instance. Stack copies require no extra payment.

RON permanent rules support `PlayerEffect(You, [IncreaseSpellCost(filter,
PayMana(mana))])` and `EachPlayerEffect(AnyPlayer, [IncreaseSpellCost(filter,
PayMana(mana))])`. The operation list contains exactly one rule, and `mana`
contains fixed mana symbols. Spell-cost filters support `AnySpell`, nonland
`IsCardtype`, `IsColor` for W/U/B/R/G, and nonempty nested `And`/`Or`.
`IsColor` reads card colors, including an indicator. It also works in the existing
static generic-reduction context. This support does not include an
`ActivatedAbilityEffect` tax. Derelor is a complete RON consumer; Gloom still
requires that separate activation contract.

## RON static activation mana modifiers

Permanent definitions accept `ActivatedAbilityEffect(subject, operation)/2`:

- `AbilityOfAPermanent(filter)/1` with `IncreaseManaCost(manaSymbols)/1`.
  The filter accepts `IsCardtype(type)/1`, `IsColor(color)/1`, and nonempty
  nested `And(list)/1` or `Or(list)/1`. Mana symbols use fixed, nonnegative
  generic and W/U/B/R/G costs. X is rejected.
- An Aura accepts `AbilityOfPermanent(HostPermanent)/1` with
  `ReduceManaCostNotLessThanOne([CostReduceGeneric(n)])/1`, where `n` is a
  nonnegative integer literal. Other references and reduction lists fail
  declaration checks.

These lower to `IncreaseActivationCostStatic(filter PermanentFilter,
increase core.ManaCost) ContinuousEffect` and
`ReduceActivationCostStatic(subject PermanentRef, reduction, minimum int)
ContinuousEffect`. They use live source incarnations and typed Aura host
references. Sources can affect either player's abilities. Source departure,
phasing and printed ability removal disable them. Reattachment changes the
subject; a returned host does not inherit an old attachment.

The shared activation transaction combines all mana components and announced
X, adds increases once, then reduces generic mana. Colored mana is retained.
A minimum limits the reduction; it does not add mana to a zero cost. Multiple
sources accumulate. The legacy `AddActivationCostReduction` caller remains
supported. Ordinary activation affordability, actual payment, native mana
abilities and solver plans use the shared total. Spells and resolution payments
do not acquire these modifiers. Copies of abilities do not pay again.

Evidence: `TestStaticActivationCost*`, `TestRONActivationCost*`. Complete RON
Gloom tests cover both its spell and activation taxes. Complete Power Artifact
casts and Basalt Monolith interactions cover its artifact enchant restriction,
generic reduction, stacking, zero mana and one-mana minimum. All six Gloom
reprints and the Power Artifact record pass declaration and strict subset
loading tests; the production collection still rejects remaining gaps.

### Current card P/T queries for count CDAs

Use `g.CardPowerToughness(card)` when the current value of a card matters.
`card.Power()` and `card.Toughness()` remain printed values for copy construction.
The existing `PTEqualsCount`, `PTEqualsControlledCount` and
`ToughnessEqualsControlledCount` declarations work with this query outside the
battlefield. Counts use current nonphased permanents. A spell uses its controller;
a card without a controller uses its owner. Battlefield queries use layer results,
including ability removal, base overrides, bonuses, counters and switching.
Use saved-information services for LKI; this current-object query cannot preserve
a departed incarnation. See the count-CDA contract in `engine-rules-contracts.md`.
RON count-CDA lowering is described below.

### RON battlefield count CDAs

Creature rules can declare `CDA_Power(TheNumberOfPermanentsOnTheBattlefield(filter))`
and `CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(filter))` together with
identical position-free count expressions. They lower to one fresh
`PTEqualsCount` or `PTEqualsControlledCount` declaration per factory instance.
A toughness-only declaration requires a `ControlledByAPlayer(SinglePlayer(You))`
restriction and lowers to `ToughnessEqualsControlledCount`. Duplicate axes,
unequal expressions, power-only counts and noncreature declarations reject.

Count filters accept nonempty `And`/`Or`, explicit card names through
`IsNamed(NamedCard(nonempty_string))`, card types, creature/land/enchantment
subtypes, colors and supertypes, including their existing negative forms.
Controller restrictions must occur in conjunctions and use `You`; disjunctions
cannot contain a controller restriction. Object references, current P/T,
ability filters, arithmetic and entry/payment results remain unsupported.
The engine supplies owner/current spell controller context outside the
battlefield, layer 7a execution, phasing, per-incarnation binding and copying.

Evidence: `TestRONCharacteristicCount*` executes the complete Dakkon Blackblade,
Keldon Warlord, Nightmare, Plague Rats and People of the Woods records and checks
all twenty records, including reprints. The strict full collection still rejects
remaining gaps. Compilation counts do not establish full collection gameplay.


## Live-object damage-source enumeration prerequisite

`mage.Game.LiveDamageSources() []mage.DamageSource` reads current unphased
battlefield permanents, followed by live spells in stack insertion order.
Permanent spells and spell copies qualify. An ability on the stack does not
qualify as a spell. Objects need not have a damage ability. Shroud does not
restrict this non-targeted source choice. The query uses the current layer
results and does not run a layer pass, make decisions, or change game state.
Each returned source has independent snapshot data.

`mage.ChooseDamageSourceFromLiveObjects(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` enumerates those objects at resolution and uses the
existing filtered choice service. The result uses the current resolution frame.
The explicit-reference choice APIs retain their existing behavior.

This is a complete contract for the two live-object categories, not complete
CR 609.7a enumeration. Objects referred to by stack objects, waiting replacement
or prevention effects, or waiting delayed triggers, and face-up command-zone
objects require further engine contracts. RON `ChooseADamageSource/1` remains
rejected. Its consumer cannot use this restricted list as its whole legal list.
No source files or production registry change in this slice.

### Captured blocking rule permissions

Use `Until(EndOfTurn, BlockAnyNumber(subject), BlockEachAttacker(subject))` for a
permanent that can block every legal attacker and must do so. These operations
change rules; they do not grant abilities. The shared blocker planner respects
capacity, evasion and minimum blockers. Captured subjects keep their incarnation
through control changes and phasing. Ability removal does not remove the rules.

RON `CreatePermanentRuleEffectUntil(subject, operations, UntilEndOfTurn)` accepts
nonempty lists of the bare operations `CantBeRegenerated`,
`CanBlockAnyNumberOfCreatures`, and `MustBlockEachAttacker`. The subject is a
permanent source or declared permanent target. Regeneration prohibition uses
`Until` with `GrantKeywordModification(subject, CantRegenerate)`; it does not
prevent creation of a regeneration shield. Each factory keeps separate target
bindings. Unsupported operations and durations fail compilation.

### RON static untap-step rules

Permanent rules accept `EachPlayerEffect(AnyPlayer, [SkipsUntapStep])`, with
exactly one bare operation. It lowers to `Static(SkipUntapSteps())` and replaces
the whole step before phasing and untap actions. The existing replacement
pipeline chooses and rechecks competing replacements. Other player selectors,
tuple operations, multiple operations and resolving/cost/entry contexts remain
errors. Stasis also uses the existing controller-upkeep payment and sacrifice
adapters; the skip does not skip that upkeep.

A static `If(condition, [EachPlayerEffect(AnyPlayer,
[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(n), filter)])])`
uses `LimitUntapsDuringStepIf(filter, n, condition)`. The condition uses the
existing static-condition grammar and typed source/Aura-host references;
unbound targets and incompatible references remain errors. The limit is one
nonnegative literal and uses the existing permanent filter grammar. No other
all-player operation is supported inside this conditional body, and mixed
bodies remain rejected. This form does not add a target or mutable binding.

Winter Orb's complete record checks `PermanentPassesFilter(ThisPermanent,
IsUntapped)`. The engine reads that condition after phasing but before the untap
selection. If a tapped Orb untaps in the same operation, the operation keeps
its unrestricted selection. The next untap step reads its new state. Separate
instances, control changes and cloned games retain independent source contexts.
Stasis and Winter Orb have complete RON gameplay mappings in
`static_player_untap_test.go`; identical reprints also pass definition validation.


## RON resolving numeric consumers

`GainLifeForEach(value, count)` resolves both typed values, multiplies them with
`mage.MultiplyValues`, and sends one life-gain action through `mage.GainLifeFor`.
It uses the current action's player, including explicit `PlayerAction` and
`PlayerActions` scopes. Spell, mode, activation and ordinary trigger bodies use
the same instance value compiler as `GainLife`. The existing restriction on
life actions in a directly captured event-player scope still applies. This form
adds no static, entry, payment, or replacement-operation declaration.

`NumberCountersOfTypeOnPermanent(value, counter, ref)` now uses that same typed
instance value compiler. It supports arithmetic and in-scope permanent
mana-value, power, counter-information and saved payment/event values, along
with existing integer and announced-X values. The existing top-level
`NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature))` form remains
supported. `UptoNumberCountersOfTypeOnPermanent` uses the same amount compiler;
its fixed cap, optional number choice and other restrictions stay in force.
Player counter placement retains its fixed-integer amount restriction.

Values are read when each action executes. A legal target's current mana value
supplies Living Armor's +0/+1 counter count. Icatian Moneychanger reads the
original source's retained credit counters after its sacrifice cost. Returned
objects cannot replace that information. An information read cannot make an
illegal, absent or phased recipient eligible for placement. Nonpositive final
amounts do nothing. Unknown, wrong-type and out-of-scope references fail at
compilation; each factory keeps its own target and payment bindings.

`TestRONNumericConsumer*` covers both complete RON records, sequencing,
resolution-time values, supported contexts, invalid declarations, LKI, timing,
target legality, copy retargeting, independent instances and clones, and
replacement ordering. These adapters add no engine service or new simultaneous
operation contract.


### RON resolving source exclusion

`SacrificeAPermanent(filter)` in a permanent's `Activated` or
`ActivatedWithModifiers` cost accepts `Other(ThisPermanent)` alone or in nested
nonempty `And` filters. Ordinary nonempty `Or` filters can select payment types;
source exclusion below `Or` remains rejected. Payment uses
`SacrificeMatchingCost`, so the source cannot pay. Existing costs without the
exclusion still permit the source. Sacrifice characteristic result bindings and
granted abilities retain their prior supported contexts.

`CreateEachPermanentLayerEffectUntil(filter, [AdjustPT(power, toughness), ...],
UntilEndOfTurn)` accepts the same source exclusion for a permanent source.
The engine captures matching recipients when the effect starts. Both players'
matching permanents qualify unless a controller filter restricts them. A
returned source is a new object and can receive the bonus. Later entries and
returned recipients cannot join the captured set. Phasing preserves a captured
recipient's identity. Other references, source-dependent disjunctions and a
spell's `ThisPermanent` remain declaration errors.

Evidence: `TestRONOrcGeneral`, `TestRONOrcGeneralRollback`,
`TestRONSourceExclusionInstancesAndCopy`, `TestRONSourceExclusionInvalid`.

## RON live nonattacking filters

Bare `IsntAttacking` is a permanent filter and lowers to
`mage.Not(mage.IsAttacking)`. It can appear in existing permanent target,
collection and static filter contexts, including nested `And`/`Or`.
Static `PermanentPassesFilter` conditions accept it on `ThisPermanent` or an
Aura's `HostPermanent`. Tuple forms are invalid. Existing reference and
source-dependent filter restrictions still apply. Card-zone and chosen damage
source filters do not gain this predicate.

Static membership reads current combat status at each layer pass. Untapping an
attacker does not make it nonattacking; removal from combat does. This predicate
is separate from attack history. Target filters check announcement and resolution.
`TestRONNonattacking*` covers the complete Arcades Sabboth record, live source
and attachment conditions, target legality, identity and clone isolation.
`TestRONUpkeepPayment*` includes its upkeep payment and sacrifice behavior.
Gaea's Liege still lacks conditional base-value and defending-player adapters;
this filter does not establish its gameplay support.

## Engine prerequisite for permanent property alternatives

The engine now exposes
`mage.IfPermanentMatchesElse(PermanentRef, PermanentFilter, ResolvingEffect,
ResolvingEffect) ResolvingEffect` and
`mage.UnlessPermanentMatches(PermanentRef, PermanentFilter, ResolvingEffect)
ResolvingEffect`. These services use one current or retained information read
for the exact object. Missing information runs neither body. Both branch bodies
are required and validated; branch-local producers do not escape their paths.
Both new services execute their conditions in AI simulations.

RON resolving action lists now accept
`IfElse(PermanentPassesFilter(subject, propertyFilter), [matchingActions],
[nonmatchingActions])` and
`Unless(PermanentPassesFilter(subject, propertyFilter), [nonmatchingActions])`.
They use the existing resolving property and counter filters, including nested
nonempty `And`/`Or`. Both bodies must be nonempty. Each branch compiles in its
own action sequence with fresh factory bindings. Payment results remain local
to their adjacent continuation; they cannot escape into another branch.
Static and combat-membership alternatives remain unsupported. Conditional mana
production in activation branches remains a declaration error.

The exact historical exception shape is
`Unless(PermanentPassesFilter(subject, CameUnderControlOfPlayerThisTurn(playerRef)),
[actions])`. It uses
`mage.UnlessPermanentCameUnderControlThisTurn(PermanentRef, PlayerRef,
ResolvingEffect) ResolvingEffect`. Each original incarnation retains the latest
acquisition turn for every player. Entry and actual control changes record it;
phasing and ability removal do not erase it. A later controller does not erase
another player's acquisition earlier in the same turn. Haste and summoning
sickness do not supply this history. Current, phased and departed information
use the same identity service; returned objects cannot replace it. Missing
object or player information runs no body. Other history shapes remain errors.

Evidence: `TestRONPropertyAlternatives*` covers the complete Blood Lust and
corrected Erg Raiders records, both property branches, nested conditions,
current reads, target legality, casting payments and rollback, retargeted copies,
factory and clone isolation, source LKI, phasing, control changes, damage
prevention and cleanup. `TestUnlessPermanentCameUnderControl*` supplies the
engine history and declaration contracts. Existing `TestPermanentPropertyBranch*`
and `TestRONResolvingCondition*` retain their behavior.

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

### RON resolving blocking-history collections

`WasBlockedByDefenderThisTurn(permanentRef)` accepts one bound singular
permanent reference in a resolving collection. It lowers to
`mage.BlockedByThisTurn` and composes through the existing collection `And/Or`
adapters. For example, both Glyph destruction clauses use
`And([IsCardtype(Creature), WasBlockedByDefenderThisTurn(Ref_TargetPermanent)])`.
The existing `DestroyEach` and `DestroyEachWithoutRegeneration` actions retain
batch destruction, regeneration and replacement behavior.

The reference belongs to its rule instance and keeps its original incarnation.
Saved blockers can supply history after departure or phasing. Collection members
must still be live and phased in. Invalid arity, unbound references, player
references and cross-rule reads are declaration errors. Inventory gives the
operand a permanent-reference role even below unsupported delayed actions.

Target-filter use, including Glyph of Delusion's dependent second target, is
still unsupported. After-combat casting, death-result
loops and the RON last-block controller expression remain separate gaps. The
engine's typed `ControllerWhenLastBlockedThisTurn` supplies the controller
history prerequisite. Glyph of Delusion and Glyph of Reincarnation still reject;
Glyph of Doom uses the bounded schedule described below. Tests
of extracted clauses do not claim complete gameplay coverage for those cards.

## RON resolving collection attack rules

`CreateEachPermanentRuleEffectUntil(filter, [CantAttack], UntilEndOfTurn)`
accepts the supported source-independent static permanent filters, including
nonempty logical combinations. Controller and source-exclusion selectors are
not supported in this restriction shape.

`CreateEachPermanentRuleEffectUntil(And([filter, ...,
ControlledByPlayer(TheActivePlayer)]), [MustAttack], UntilEndOfTurn)` accepts
exactly one active-player selector and one or more supported permanent filters.
It saves the active player at resolution and checks current creature membership
at attack declaration. Filter order does not matter. Other player selectors,
operation lists, durations and unbound references remain compile errors.
Both forms work in resolving action lists. They lower to
`CantAttackMatchingThisTurn` and `MustAttackControlledThisTurn`; they do not use
captured layer membership or give creatures an ability (CR 611.2c).

Festival's complete RON record is tested. Siren's Call's exact attack clause is
tested in a test-only extracted record; its full record still rejects on the
delayed destruction collection. Runesword's regeneration and exile clauses
remain unsupported. These forms do not establish other collection rule support.

### Turn-scoped unblocked attacker subscription

`WhenAttacksUnblockedThisTurn(subject PermanentRef, body ResolvingEffect)
ResolvingEffect` saves the watched incarnation and resolving controller. It uses
legal blocker-declaration events in each combat until cleanup. A phase cycle
keeps identity; a returned object cannot use the subscription. Source departure,
phasing, control changes and ability removal do not end the subscription.
Pending triggers retain the body after cleanup. Use `Snapshot` and `From` when
the body must read the watched object independently of the creator.

RON supports `CreateTriggerUntil(WhenACreatureAttacksAndIsntBlocked(
SinglePermanent(ref)), ActionList([actions...]), UntilEndOfTurn)` in resolving
contexts. The subject must be a bound singular permanent reference. The nonempty
body can use existing supported actions and the local `Trigger_ThatCreature`
reference. This reference names the watched incarnation; it does not name the
artifact that created the subscription. Parent references retain their bindings.
Each installation saves its own frame, including original targets and X.
Other delayed combat events, durations, target declarations inside this body,
and unsupported body operations still reject.

Delif's Cone and Delif's Cube use this event. Real-record clause tests establish
Cone's power/life expression and Cube's counter instruction with their actual
activation costs and target restrictions. Cone remains unsupported on optional
life gain. Cube's complete gameplay uses the assignment rule described below.

### Combat damage assignment suppression

`mage.AssignsNoCombatDamage(subject mage.PermanentRef) mage.ContinuousModification` can be installed with `mage.Until`. It changes a
combat rule and remains effective after ability removal. The resolved subject
keeps its incarnation; source departure and subject control changes do not end
an end-of-turn installation. A phase cycle keeps it. Cleanup ends it, including
while the subject is phased out. An absent or phased-out subject at installation
receives no rule. This service does not prevent damage or change power.

RON resolving action lists accept
`CreatePermanentRuleEffectUntil(permanentRef, [AssignsNoCombatDamage],
UntilEndOfTurn)`. There are exactly three arguments, one symbol operation, and
one symbol duration. The permanent reference must be bound in that action's
scope: source, target, Aura host, or a supported permanent event binding.
Factory construction uses that instance's typed references. Other durations and
mixed operation lists are not added by this adapter.

`TestRONCombatAssignmentCube` executes the complete Delif's Cube record,
including casting, its delayed combat trigger, cube counter payment and
regeneration. `RollbackAndIdentity`, `EventAndInstances`, `SourceAndHost` and
`Invalid` establish the supported scopes, legality, copies and declaration
rejections. The optional payment contract below adds three consumers.
Farrel's Mantle still needs its host unblocked event contract.

## RON optional resolution action payments

`MayCost(action)` and `PlayerMayCost(player, action)` accept these complete
singular action shapes in resolving sequences:

- `GainLife(value)` uses the current action player and a resolution value.
- `HavePermanentDealDamage(permanent, value, recipient)` uses a typed permanent
  source and one existing `Permanent(ref)`, `Player(ref)`, or singular union
  recipient. Source information follows `DamageSourceOf`, including LKI.
- `DestroyPermanent(permanent)` requires the original live, unphased object
  without indestructible. Regeneration can replace the paid instruction.

The payment must immediately precede `If(condition, [actions])`,
`Unless(condition, [actions])`, or `IfElse(condition, [actions], [actions])`.
The condition is the symbol `CostWasPaid` or `CostWasntPaid`; each listed
branch is nonempty. References and values must already be bound in the current
context. The adapter creates no new event, choice, or target binding.
`MayCost` asks the resolving controller. `PlayerMayCost` asks its explicit
player; it does not change the action's player or damage source.

Lowering uses `PayActionOrElse` with a typed `ResolutionActionPayment`.
The choice determines the branch (CR 118.11–12). Prevented damage, replaced
life gain, and regeneration still count as payment. Indestructible and absent
subjects cannot pay destruction. The payment captures its amount and object
identity before the choice, then rechecks the recipient before execution.
Nested payments cannot replace an outer result. Each payment is one action;
these forms do not add composite or announcement costs.

`TestRONActionPaymentRealCards`, `NativeCombat`, `Invalid`, `StrictLoad`, and
`BranchesAndChooser` cover Delif's Cone, Farrel's Zealot, and Floral Spuzzem.
Farrel's Mantle still needs its host event binding and a source-data review.

## RON indefinite fixed color and keyword changes

`CreatePermanentLayerEffect(reference, [SetColor(SimpleColorList([colors...]))])`
accepts distinct Magic colors. An empty list sets colorless. It replaces the
subject's colors in layer 5. Choices and unbound references are rejected.

`CreatePermanentLayerEffect(reference, [AddAbility([keywords...])])` accepts a
nonempty list of the existing fixed keyword declarations, including fixed
landwalk. The separate activated-ability grant form remains supported.
These resolving forms use `Until(Indefinite)`, `SetColors`, and
`GrantKeywordModification`. They capture the exact live subject at resolution.
Source departure does not end them. Returned subjects do not inherit them;
phasing preserves an installed effect. Later layer timestamps still apply.

Full-card evidence: `TestRONAislingIndefiniteColor`,
`TestRONAislingNativeCombat`, and `TestRONCocoonCompleteLifecycle`.
Declaration and recipient contracts are in `TestRONIndefiniteCharacteristic*`
and `TestRONIndefiniteKeywordIdentity`. This slice adds no engine service.

## RON scoped color choices

Resolving action lists support these adjacent pairs:

```ron
ChooseAColor(AnyColor),
CreatePermanentLayerEffect(permanent_reference, [SetColor(TheChosenColor)])

ChooseColors,
CreatePermanentLayerEffect(permanent_reference, [SetColor(TheChosenColors)])
```

The first pair offers exactly the five Magic colors. The second offers all 32
subsets, including no colors. Each choice is made at resolution by the resolving
ability or spell's controller. These are resolution choices, not announced modes.
The result belongs only to the adjacent color-setting continuation. Unbound,
wrong-kind, nonadjacent and escaped result reads are rejected. Entry choices,
chosen-color filters, other modifications and other durations remain unsupported
by this adapter. The existing typed permanent reference compiler supplies the
subject; the choice introduces no target declaration.

The compiler lowers each legal option to a fixed `SetColors` modification with
`Until(Indefinite)`, through `ChooseAtResolution`. No choice is stored on a source
permanent or in shared mutable declaration state. The existing layer capture
service owns incarnation identity, phase cycles, timestamps and game cloning.
Accepted Alchor's Tomb and Dream Coat records have gameplay mappings in
`color_choice_test.go`; this does not certify all accepted collection records.

## RON temporary ability grants

`CreatePermanentLayerEffectUntil(ref, [AddAbility([Protection(FromColor(
Colors([White])))])], UntilEndOfTurn)` accepts one or more of the five Magic
colors. Empty lists, colorless, unknown qualities, malformed constructors and
unbound subjects fail compilation. Protection is part of the existing temporary
ability plan, including its scoped `ChooseAnAbility` continuation. The plan keeps
an immutable color set; factory calls build separate target declarations.
Existing permanent references and temporary durations keep their scope checks.

`CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,
[AddAbility([Landwalk(IsLandType(Island))])], UntilEndOfTurn)` accepts nonempty
lists of existing temporary keywords, including fixed landwalk. It uses the
existing plural target binding from `NumberTargetPermanents` or
`OneOrMoreTargetPermanents`. Multiple `AddAbility` operations share one captured
installation. Mixed color/keyword modifications, plural protection/rampage and
delayed plural target reads remain unsupported.

`mage.GrantProtectionColors(subject mage.PermanentRef, colors ...core.Color)
mage.ContinuousModification` is the typed engine primitive. Definition validation
rejects missing subjects, undeclared targets and invalid color sets. Use it with
`Until` to capture the live subject incarnation in layer 6. Normal protection
rules apply to targeting, attachment, blocking and damage; there is no Ward
attachment exception. Repeated layer passes do not accumulate abilities.

Goblin Wizard and Part Water have complete RON gameplay mappings in
`temporary_ability_grant_test.go`. Their native costs, choices, targets, copies,
clones, object identity and cleanup are tested separately from loader acceptance.

### RON live static base values

RON supports `PermanentLayerEffect(subject, [SetPowerAndToughnessBoth(value)])`
for `ThisPermanent` and a declared Aura's `HostPermanent`, alone or inside an
existing static `If` body. This singular operation uses `SetBasePowerToughness`
and `While` in layer 7b. Its value is read on each layer pass, before bonuses,
counters and switches. It is not a characteristic-defining ability.

Static values accept `Integer/1`, `Plus/2`, `Minus/2`, `Twice/1`,
`HalfRoundedDown/1`, `HalfRoundedUp/1`, and the existing restricted battlefield
counts. `ManaValueOfPermanent/1` accepts the source or declared Aura host and
uses current information. These values also work in the existing singular
static `AdjustPTXY` and `AdjustPTForEach` adapters. Targets, event references,
announced X and P/T reads remain invalid in this context.

Combined modifications and collective base-value operations remain rejected.
This prerequisite does not accept the complete Animate Artifact, Angry Mob or
Gaea's Liege records. Their remaining rules require separate slices.

### RON basic land type choices

`AsPermanentEnters(ThisPermanent,[ChooseABasicLandType])` declares one saved
entry choice on a permanent. The operation must be a symbol. Duplicate choices,
combined operations and nonpermanent declarations are errors. Rule order does
not change this declaration's scope. On an Aura,
`PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`
uses that entry declaration and accepts exactly one modification.

The engine runs `ChooseBasicLandTypeAsEnters` before battlefield registration.
The entry controller chooses Plains, Island, Swamp, Mountain or Forest through
`ChooseString`. The permanent stores `core.BasicLandType` separately from its
chosen color and other entry choices. `BecomesChosenBasicLandTypeAttachedEffect`
uses the existing attachment and land-type services. It replaces land subtypes
and printed abilities, supplies intrinsic mana, preserves supertypes and follows
reattachment. Source phasing and departure stop the effect. Ability removal in
layer 6 does not undo the type change already applied in layer 4.

In an action sequence, bare `ChooseABasicLandType` must immediately precede
`CreatePermanentLayerEffectUntil(subject,
[AddAbility([Landwalk(IsLandTypeVariable(TheChosenLandType))])],duration)`.
The subject and duration use their existing supported contexts, including the
body of a supported delayed step trigger. This bounded pair declares no target
and exposes no result outside its continuation. The compiler builds five fixed
branches through the existing temporary keyword compiler. `ChooseAtResolution`
selects one branch for each invocation; no mutable compiler or global result
slot stores the choice. The controller chooses at resolution, even if the saved
source has left. A grant still requires its original live incarnation.

The complete Phantasmal Terrain records (six reprints) map to
`TestRONBasicLandChoiceTerrain`, `TestRONBasicLandChoiceTerrainIdentity` and
`TestRONBasicLandChoiceInventoryAndReprints`. Giant Slug maps to
`TestRONBasicLandChoiceSlug` and `TestRONBasicLandChoiceSlugIdentity`.
`TestRONBasicLandChoiceInvalid` and `TestRONBasicLandChoiceStrictLoading` cover
scope errors and atomic rejection. General subtype choices, plural choice
consumers, mixed modifications and other saved-land-choice reads remain errors.

### Conditional static damage redirection

Use `mage.RedirectDamageToPermanentIf(player, destination, sourceFilter,
condition)` with `mage.Static` for player damage from permanent sources.
Use `mage.RedirectDamageFromCardTypeToPermanentIf(player, destination, cardType,
condition)` for a type restriction that includes sources outside the battlefield.
Both constructors require a condition. The rule reads current controller and
attachment references, and the destination must remain a creature or planeswalker.
The shared replacement pipeline preserves source information and damage facts.
`mage.DamageSourceIsUnblocked()` reads current or retained combat status. An
attacker becomes unblocked only after blocker declaration; removal of its last
blocker does not make a blocked attacker unblocked.

RON `If`, `Unless` and `IfElse` static branches can contain one
`ReplaceWouldDealDamage(DamageWouldBeDealtByAPermanentToRecipient(filter,
Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)])`.
`HostPermanent` is also valid in an Aura. Permanent filters accept nonempty
`And`/`Or`, bare `IsUnblocked`, and the supported type, subtype, supertype and
color predicates. `DamageWouldBeDealtByASourceToRecipient(IsCardtype(type),
Player(You))` includes captured spell sources. This does not add resolving
replacement operations or target declarations.


### Original permanent sources of stack abilities

`mage.Game.StackAbilityPermanentSources() []mage.DamageSource` reads original
permanent sources of abilities still on the stack, in insertion order with
exact duplicates removed. It keeps saved incarnation identities through
source departure, return and phasing. Live sources use current information;
absent sources use the shared phase/departure information reader. Returned
cards cannot replace the original source. Missing identities are excluded.
The query is pure and returns separate snapshot data.

`mage.ChooseDamageSourceFromStackAbilities(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` makes a filtered non-targeted choice from that category
at resolution. The shared choice service owns validation, decisions and
per-resolution output bindings. Empty lists and absent players clear the
result. Clones retain independent results.

This source-only API covers one CR 609.7a context. Other stack references,
waiting replacement/prevention and delayed-trigger references, and face-up
command-zone objects remain dependencies. General RON `ChooseADamageSource`
remains rejected; do not combine this and the live-object list and treat them
as the complete legal list. Existing Go callers remain compatible.

### Permanent targets referred to by stack objects

`mage.Game.StackTargetPermanentSources() []mage.DamageSource` reads exact
permanent target incarnations referred to by spells and abilities still on the
stack. Results follow stack insertion order, then target-list order, with exact
duplicates removed. Target legality does not remove this reference: shroud,
phasing or departure can make a target illegal while the object still refers
to it. Current information applies to live incarnations; phase information or
departure LKI applies to absent incarnations. Returned cards cannot replace them.

`mage.ChooseDamageSourceFromStackTargets(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` makes a filtered non-targeted choice from this category
at resolution. It uses the shared decision and per-resolution binding services.
Empty lists and absent players clear only this output. Definition validation
checks players, filters, bindings and producer-before-consumer ordering.

Copies retain original target incarnations unless retargeted. Only current
`Targets` entries qualify; old map entries cannot add candidates. Players,
spell targets, graveyard card visits and missing/unknown identities do not
qualify as permanent targets. Removing the final referring stack object ends
eligibility after the final resolving body completes. Queries are pure,
return separate snapshot data and preserve independent clones.

This establishes one CR 609.7a category. Other saved stack references, waiting
replacement/prevention and delayed-trigger references, and face-up command
objects remain dependencies. General RON `ChooseADamageSource` stays rejected.
No existing caller changes and no new production RON shape is added.

### Spell targets referred to by stack objects

`mage.Game.StackTargetSpellSources() []mage.DamageSource` reads spell targets
of spells and abilities still on the stack. Results follow stack insertion and
target-list order, with exact duplicates removed. A spell need not be capable
of damage. Permanent spells qualify. Ability, player, permanent and source-card
IDs cannot substitute for spell stack IDs.

Live spells supply current characteristics and controller. Each referring stack
object saves the target spell's information. Every native departure path refreshes
that information before removal. A later cast or battlefield entry cannot replace
the old spell. Copies keep the saved reference; retargeted copies use their own
selected targets. Removing the final referring object ends this category's
reference. Queries return separate card data and make no choices or state changes.
Stack clones share immutable maps and copy them before departure updates.

`mage.ChooseDamageSourceFromStackSpellTargets(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses the shared filtered resolution choice service.
Declaration validation, producer-before-consumer checks, separate bindings,
empty/absent-player clearing and clone behavior remain the same. This service
covers only spell targets, not the complete CR 609.7a source list. Other saved
bindings, waiting effects, command objects and chosen-source shields remain
separate dependencies. No additional production RON shape is accepted.

### Permanent bindings referred to by stack objects

`mage.Game.StackBoundPermanentSources() []mage.DamageSource` reads exact
permanent incarnations saved by event, announcement, payment and delayed-frame
bindings on spells and abilities still on the stack. It reads `capturedPermanents`,
immutable event captures and captured delayed bodies in their sequence wrappers.
It does not execute an effect or evaluate an unproduced binding. Results follow stack insertion
order, then permanent ID and incarnation byte order within each object, with
exact duplicates removed. Queries return separate card snapshots.

Live incarnations use current information; phased and departed incarnations
use their separate retained information. Returned objects cannot replace them.
Shroud and damage capability do not restrict this non-targeted source category.
Copies keep saved references. Removal of the last referring stack object ends
eligibility after body completion. Pending triggers are excluded.

`mage.ChooseDamageSourceFromStackBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` reuses the shared filtered resolution choice service.
Definition validation checks the player, filter, output and binding dominance.
Empty categories and absent players clear only that output. Choices and cloned
resolution frames stay separate.

This category includes saved permanent bindings, target groups and created-token
sets in delayed bodies after stack placement. It is not the complete CR 609.7a
source list. Other binding types, waiting replacement/prevention/delayed references
and face-up command objects remain separate dependencies. General RON
`ChooseADamageSource` remains rejected. Existing Go callers stay compatible.

### Damage-source event bindings referred to by stack objects

`mage.Game.StackBoundDamageSources() []mage.DamageSource` reads exact permanent
and spell identities saved by `eventDamageSourceCapture` within stack objects,
including nested sequence wrappers. It does not execute captures, evaluate future
action results, or read the active resolution frame. Results follow stack
insertion order and capture order, with exact duplicates removed.

Live permanent incarnations and spell stack IDs supply current information.
Permanent departure and phase information use the existing exact-incarnation
reader. Spell departure information is saved separately on each referring object,
with copy-on-write updates for clones. A returned permanent or recast spell cannot
replace the original object. The trigger body's event-time source binding remains
unchanged. Query results own separate card snapshots.

`mage.ChooseDamageSourceFromStackDamageBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses the shared filtered non-targeted resolution choice
service. Definition validation, independent results, empty/absent-player clearing,
invalid-answer fallback and cloned resolution frames retain their existing rules.

Pending triggers are outside this category. Resolving frames retain references.
Removing the final referring stack object ends eligibility. This adds one source
category required by CR 609.7a; saved delayed frames, other typed references,
waiting effects and face-up command objects remain separate dependencies. General
RON `ChooseADamageSource` and the seven chosen-source shields remain rejected.
Existing production declarations and callers do not change.

### Spell-event bindings referred to by stack objects

`mage.Game.StackBoundSpellSources() []mage.DamageSource` reads spell references
saved by typed cast-event bindings on stack objects. Detection saves identity and
information before placement. Nested sequences expose immutable captures without
running them. Live spells use current information; departing spells use the shared
copy-on-write stack snapshot service. A spell absent before placement uses its
detection snapshot. Recasts and permanent entry retain separate identities.

The query returns copied card data in stack insertion/capture traversal order,
with exact duplicates removed. Nil, invalid and incompatible captures are
excluded. Pending triggers are outside the category; resolving frames retain
references through body completion. Copies retain references; clones keep independent departure information.

`mage.ChooseDamageSourceFromStackSpellBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` performs a filtered, non-targeted choice at resolution.
It uses shared definition validation, producer order, independent output bindings,
empty/absent-player clearing, invalid-answer fallback and cloned result frames.

This establishes the typed spell-event context of CR 609.7a. Other delayed bindings,
other reference types, waiting effects, face-up command objects and consuming
shields remain separate dependencies. General RON `ChooseADamageSource` remains
rejected; no new RON spelling or compiler adapter is established here.

## Delayed permanent frames referred to by stack objects

`StackBoundPermanentSources` and `ChooseDamageSourceFromStackBindings` also
read immutable `capturedDelayedBody` frames after delayed trigger placement.
Saved permanent bindings, singular/plural target groups and created-token sets
supply exact permanent incarnations. The shared reader sorts and deduplicates
them with other references in each stack object. Nil declarations and invalid
identities are excluded. Reading does not execute the body or a future producer.

Current objects supply current characteristics and control; absent and phased
objects use exact retained information. Copies preserve captures; game clones
keep independent state. Query and choice snapshots cannot mutate saved cards.
A saved target does not become a new target. Waiting and pending triggers
stay outside this category; resolving frames retain their references.

No RON syntax or context is added. Stone Giant, Goblin Kites and War Barge use
existing delayed adapters; `TestRONStackDelayedPermanentSources` and
`TestRONStackDelayedPermanentSourceInstances` verify their native gameplay and
source choice through a synthetic category consumer. General `ChooseADamageSource`
and the complete shields still need other source categories and shield contracts.

## Permanent references held by waiting delayed abilities

`mage.Game.WaitingDelayedPermanentSources() []mage.DamageSource` reads saved
permanent bindings, singular/plural target groups, created-token sets and exact
permanent event subjects from delayed abilities that still wait to trigger.
Captured modal bodies and sequence wrappers use the same frame reader. The query
uses registration order, then sorted ID/incarnation pairs within each ability,
and removes exact duplicates. It does not run effects or make choices.

Live objects supply current properties. Departed and phased objects use the
shared exact-incarnation information reader. Returned objects cannot replace
saved references. Each result owns separate card data. One-shot detection ends
waiting eligibility before stack placement. A delay with a duration keeps waiting
until its existing lifecycle expires. Pending and stack objects have separate
categories; copies on the stack do not create new waiting abilities. Game clones
retain independent waiting state.

`mage.ChooseDamageSourceFromWaitingDelayedBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses the shared filtered resolution-choice service,
declaration validation and binding dominance. Empty or absent-player choices
clear only that output. Invalid answers use the existing first-choice fallback.

This is a permanent-frame and exact-event-subject contract. Other typed saved
references, source reads in future bodies, untyped legacy IDs, waiting replacement
and prevention effects and command objects still need separate contracts. No RON
syntax is added. General `ChooseADamageSource` remains rejected.


## Source references in waiting delayed action bodies

`Game.WaitingDelayedPermanentSources` also reads explicit `This` subjects in
`Destroy`, `DestroyWithoutRegeneration`, `Sacrifice` and `ReturnToHand` bodies.
This context includes saved `capturedDelayedBody` wrappers, nested `Sequence`
bodies and each saved modal body. The typed reference interface does not execute
an action. A source ID alone does not qualify. An action that will create another
delayed ability does not supply its future body's references in this context.
Other source-reading action kinds remain separate forms. Property branches have
the reference contract below; other conditional wrappers remain separate.

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

## RON static maximum hand size

Top-level permanent rules support these complete shapes:

```text
PlayerEffect(You, [HasNoMaximumHandSize])
PlayerEffect(You, [SetMaximumHandSize(Integer(n))])
PlayerEffect(TheChosenPlayer, [HasNoMaximumHandSize])
PlayerEffect(TheChosenPlayer, [SetMaximumHandSize(Integer(n))])
```

`n` is a nonnegative integer literal. `TheChosenPlayer` requires the existing
`AsPermanentEnters(ThisPermanent, [ChooseAPlayer(Opponent)])` declaration on the
same card. Each list has exactly one operation. These are static declarations;
resolving actions, combined operation lists, target/event player references,
constructor-form symbols, dynamic amounts and negative maxima remain errors.
The inventory assigns the fixed amount the `value` role.

RON lowers these forms to `SetMaximumHandSizeStatic(PlayerRef, int)` and
`NoMaximumHandSizeStatic(PlayerRef)`, both returning `ContinuousEffect`.
`ChosenPlayerOf(PermanentRef) PlayerRef` supplies the saved entry choice through
a live incarnation; it reads no LKI. Use `WithStaticAbility` for native callers.
`Game.MaximumHandSize` and cleanup share the live rule query and timestamp order.
The existing direct hand-size setters remain available to current callers.
Cursed Rack is covered as a complete RON card. Library of Leng's discard
replacement remains unsupported; its hand-size rule alone has gameplay evidence.

### Delayed spell-event references

`AtNext`, `AtNextModes` and delayed permanent-event bodies now capture and restore
`SpellBinding` values. A `FromSpell` read keeps the original stack identity; a
counter action cannot affect a later cast of the same card. Other typed frames
retain their existing behavior.

`mage.Game.WaitingDelayedSpellSources() []mage.DamageSource` reads saved spell
frames only while their abilities wait to trigger. It uses current information
or exact departure information, returns independent snapshots, and deduplicates
in deterministic order. Placed delayed bodies use `StackBoundSpellSources`.

`mage.ChooseDamageSourceFromWaitingDelayedSpellBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` chooses from this category through the shared filtered
choice service. Missing choices and absent players clear only this output.
General RON `ChooseADamageSource` remains rejected until its complete source list
and shield contracts are available.

In supported RON delayed bodies, bound `CounterSpell(Trigger_ThatSpell)` and
other existing spell-reference consumers now use the saved cast-event frame.
`Trigger_ThatSpell` requires an enclosing cast event. It does not become a target
and cannot refer to a later cast. This adds no RON spelling or production registry.

### Delayed damage-source frames

`AtNext`, `AtNextModes` and delayed permanent-event operations save the current
`DamageSourceBinding` frame. `FromDamageSource` reads the saved facts during
body execution. Separate executions receive separate maps, and the outer frame
is restored afterward.

`mage.Game.WaitingDelayedDamageSources() []mage.DamageSource` reads exact source
references held by waiting delayed frames. Queries use current or exact departure
information, preserve return/recast identity, and return separate snapshots.
Placed bodies supply `StackBoundDamageSources`. One-shot detection ends waiting
eligibility; duration-based delays retain it until cleanup.

`mage.ChooseDamageSourceFromWaitingDelayedDamageBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` chooses from this category through the existing filtered
choice service. Empty categories and absent players clear only the named output.

RON's supported controller/source received-damage events create fresh internal
source captures that supported delayed bodies can retain. Existing amount and
recipient spellings keep their scope. No new source-reference spelling is added.
General RON `ChooseADamageSource` still requires the other source categories and
complete shield contracts.

### Waiting prevention permanent references

`mage.Game.WaitingPreventionPermanentSources() []mage.DamageSource` reads exact
permanent recipients saved by `PreventNextDamage` and `PreventDamageThisTurn`,
and exact source subjects saved by `PreventNextDamageFromPermanent`. It scans
active persistent and cycle replacements in registration order, removes exact
duplicates, and returns separate snapshots. Player IDs, legacy source IDs
without incarnations, filters over unspecified future members, unknown objects,
consumed shields and removed effects do not supply references.

`mage.ChooseDamageSourceFromWaitingPreventionReferences(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses this category with the shared non-targeted choice and
binding service. Missing players and empty filtered categories clear that output.

Existing RON lowering of
`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(amount,
recipient), [PreventThatDamage])` and
`CreatePreventDamageUntil(DamageWouldBeDealtToRecipient(recipient),
[PreventThatDamage], UntilEndOfTurn)` now supplies these references after
resolution. The permanent-source-filter variant of the second declaration keeps
its existing behavior; its filter does not declare individual source objects.
No new RON syntax or partial `ChooseADamageSource` adapter is introduced.
Other prevention/replacement contexts, spell references and command-zone sources
remain separate prerequisites for general source choice.


### Waiting references from resolved combat prevention

`Game.WaitingPreventionPermanentSources() []DamageSource` also includes exact
subjects retained by `PreventCombatDamageToAndBy`, `PreventCombatDamageBy`,
`PreventCombatDamageByThisTurn`, `PreventCombatDamageByEach`, and
`PreventCombatDamageByEachThisTurn`. Collection members are saved at resolution;
later members cannot enter the reference list. A to-and-by subject appears once.
The shared query and `ChooseDamageSourceFromWaitingPreventionReferences` retain
the existing identity, information, deduplication, filtering and binding contract.
Combat prevention remains available after matching damage. End-of-combat or
end-of-turn removal ends its eligibility according to its declared duration.

RON uses the existing `CreatePreventDamageUntil` lowering:
`CombatDamageWouldBeDealtByCreature(permanent)` for one source;
`Or([CombatDamageWouldBeDealtToRecipient(Permanent(permanent)),
CombatDamageWouldBeDealtByCreature(permanent)])` for a to-and-by subject;
and `CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(permanent),
collection]))` for the supported combat relation collections. Operations must be
`[PreventThatDamage]`. Singular forms permit `UntilEndOfTurn`; supported
collection unions also permit `UntilEndOfCombat`. Existing scope checks remain.
There is no new spelling or general `ChooseADamageSource` adapter.

### Waiting references from typed static prevention

`Game.WaitingPreventionPermanentSources() []DamageSource` also reads the current
subject of `PreventCombatDamageFrom`, `PreventCombatDamageTo`,
`PreventDamageFromCardType`, and `PreventDamageFromBlockedCreatures`.
`This` resolves the live rule source; `AttachedPermanent` resolves its current
host. The source incarnation must still match. Phased or absent sources and
subjects supply no reference. Reattachment changes the subject. Ability removal
uses the existing layer lifecycle; removing a host's abilities does not remove
an Aura's rule. Queries remain read-only and return separate, deduplicated
snapshots. Source filters and blocked attackers do not declare extra subjects.

Existing RON `PreventDamage(event, [PreventThatDamage])` lowering is reused:
`CombatDamageWouldBeDealtByCreature(subject)`,
`CombatDamageWouldBeDealtToRecipient(Permanent(subject))`, their matching `Or`
union, and `CombatDamageWouldBeDealtByACreatureToRecipient(filter,
Permanent(subject))` use the static combat rules. The supported source-card-type
and blocked-creature events use the other two typed rules. Existing declaration,
scope and filter restrictions remain. This adds no new spelling or general
`ChooseADamageSource` adapter. Legacy filter-factory static prevention and other
waiting replacements still need separate reference contracts.

## Typed static prevention from matching permanent sources

`PreventDamageFromMatchingPermanents(subject PermanentRef, from PermanentFilter)
StaticRule` prevents combat and noncombat damage to its current subject. `This`
and `AttachedPermanent` use the existing static subject validation. A zero filter
permits any permanent source. Spell sources cannot match, even when their card
characteristics match the filter. Exact departed sources use the existing LKI
service; a returned card cannot supply the old source's information.

RON lowers the existing top-level `PreventDamage(
DamageWouldBeDealtByAPermanentToRecipient(source_filter,
Permanent(ThisPermanent)), [PreventThatDamage])` to this rule. The supported RON
context stays self-only; Aura hosts, event/target references, player recipients,
finite amounts and additional operations stay rejected in this event shape.
The native service permits a live attachment subject without extending RON syntax.

The rule supplies its exact live subject through
`Game.WaitingPreventionPermanentSources`. Source filters do not enumerate their
matching members. Current subject resolution, source incarnation validation,
phasing, ability removal, clones and replacement rechecks reuse typed static
prevention. Legacy `PreventDamageFromTo` callers keep their existing contract.

Tests: `TestStaticPermanentFilterPrevention*`,
`TestWaitingStaticPermanentFilterPreventionStaleCycle`, and the expanded
`TestRONWaitingStaticPreventionReferences`. Complete Argothian Pixies, Desert
Nomads and Uncle Istvan records retain `TestRONStaticPrevention*` gameplay evidence.


### Live destinations of waiting static damage redirection

`mage.Game.WaitingRedirectionPermanentSources() []mage.DamageSource` reads the
exact current destination of each active typed static damage redirection rule.
`mage.ChooseDamageSourceFromWaitingRedirectionReferences(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` saves one matching destination through the shared choice
service. This category is separate from prevention references and from complete
CR 609.7a source enumeration.

RON retains conditional `ReplaceWouldDealDamage` lowering for the permanent-source
and single-card-type source events described above. The destination plan permits
`ThisPermanent`, or `HostPermanent` in an Aura, and builds its typed reference in
each factory call. No resolving choice syntax or additional static scope is added.
A false condition, phased or departed source, missing host, or destination that
is neither a creature nor a planeswalker supplies no reference. A returned source
cannot restore a saved cycle rule. Reattachment supplies the new live host.
Queries neither execute damage nor test a hypothetical matching source.

Complete Veteran Bodyguard and Martyrs of Korlis records map to
`TestRONWaitingRedirectionReferences` and the existing `TestRONStaticRedirect*`
tests. Other replacement categories, source-choice shields, command objects and
complete `ChooseADamageSource` lowering remain separate capabilities.

### Exact-object source prevention prerequisite

`PreventNextDamageFromExactSource(source DamageSourceRef, recipient PlayerRef,
filter DamageSourceFilter) ResolvingEffect` saves one exact permanent incarnation
or spell stack identity at resolution. It prevents the next positive damage event
to that player this turn. All matching simultaneous packets use one shield.
Nonmatching properties, recipients, returned objects and recasts do not consume
it. Cleanup expires it; clones keep separate consumption state. The filter uses
current object information when available and retained information after departure.

The RON lowering prerequisite now uses `PreventNextDamageFromSelectedSource`
with the same typed arguments. It accepts
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(player)), [PreventThatDamage])` only with an existing
typed `chosenDamageSourcePlan` in the lexical scope and its factory-local binding.
The plan supplies the choice filter. It rejects unbound references, permanent
recipients, extra operations and call forms of the two symbols. `Player(player)`
uses the existing typed player reference grammar.

No production declaration creates that chosen-source scope yet. The seven
protection records remain rejected at `ChooseADamageSource`. Tests lower their
shield expressions with explicit native producers. The selected-source service
continues an active shield from an exact spell stack identity to the exact
permanent that spell becomes (CR 400.7c, 609.7a). This includes a chosen spell
copy that becomes a token with a new card ID. It links the shield after successful
battlefield registration and before entry effects and events. Failed entry, a
countered spell, a separate copy, direct entry and a later cast cannot create
that link. The original spell and its continued permanent share one consumption
state. Current properties are checked before damage; departure and phasing use
the existing exact information services. Returned objects cannot inherit it.
The continued permanent joins waiting prevention references. Clones copy both
identities and keep separate consumption. Complete source enumeration must
precede production source-choice lowering. Other prevention operations and
recipients require separate contracts.

### Waiting prevention spell choice prerequisite

`Game.WaitingPreventionSpellSources()` reads exact spell identities held by active
typed source shields. `ChooseDamageSourceFromWaitingPreventionSpellReferences(
player, filter, binding)` selects from this category through the shared resolution
choice service. Current spell data or retained departure data supplies properties;
the saved shield snapshot is the fallback. Recasts and copies keep separate stack
identities. Consumption and cleanup remove the reference. A continued permanent
supplies a separate incarnation through the permanent-reference query.

RON `compileChosenDamageSourceChoice` validates exactly
`ChooseADamageSource(filter)` and produces an immutable `chosenDamageSourcePlan`.
The category lowering `waitingPreventionSpellChoice(instanceBindings)` uses the
plan's filter, `You`, and its own factory-local chosen-source binding. It supports
synthetic category consumers alongside the existing scoped
`CreateFuturePreventDamage` lowering. Invalid arity/filter and absent bindings
fail declaration validation. The production compiler still rejects general
`ChooseADamageSource`: remaining replacement references and face-up command
objects need contracts before complete enumeration can run. No partial registry
is published. This establishes subrule evidence for the seven protection records,
not complete-card gameplay or new collection compilation coverage.


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

## RON optional draw-step skip for life

Permanent rules support the exact operation sequence
`ReplaceWouldBeginDrawStep(PlayerWouldBeginTheirDrawStep(You),
[MayCost(SkipThisDrawStep), If(CostWasPaid, [GainLife(Integer(n))])])`,
where `n` is a nonnegative integer literal. Each factory constructs
`Static(SkipDrawStepForLife(n))`. Other players, operations and dynamic amounts
remain declaration errors. The optional replacement skips the whole step;
the gain occurs first in the next step that actually occurs (CR 614.10b).
`CostWasPaid` is local to the replacement and supplies no action-list result.

`Game.WaitingDrawStepLifePermanentSources` and
`ChooseDamageSourceFromWaitingDrawStepLifeReferences` supply one damage-source
category from active draw-step life replacements. Execution and enumeration
share source incarnation, phasing and ability-lifecycle checks. Inactive rules
and pending life gains cannot supply former source objects. Queries use current
characteristics and control, copy their data and merge duplicates. The choice
uses typed player/filter/binding declarations and independent resolution results.
RON `chosenDamageSourcePlan.waitingDrawStepLifeChoice` lowers the validated
filter with `You` and the factory-local binding. Production general
`ChooseADamageSource` still rejects until all legal categories exist.

Tests: `TestRONDrawStepLifeFasting`, `TestRONDrawStepLifeFastingUpkeep`,
`TestRONDrawStepLifeInstances`, `TestRONDrawStepLifeProtection`,
`TestRONDrawStepLifeInvalid`, `TestRONDrawStepLifeFixedAmounts`,
`TestWaitingDrawStepLifeReferences` and `TestWaitingDrawStepLifeChoices`.
Full Fasting gameplay and protection subrule evidence are separate claims.

### Waiting counter-prevention subjects

`PreventDamageToSourceByRemovingCounters(counterType core.CounterType)
ContinuousEffect` keeps its existing public signature. Use it in a static
ability. `Game.WaitingPreventionPermanentSources() []DamageSource` now includes
its exact source while that ability remains active, even with zero counters.
The query excludes phased or departed sources, removed rules and stale rules
for returned objects.
It uses current control and returns independent information snapshots.

RON `chosenDamageSourcePlan.waitingPreventionChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` to the shared waiting-prevention choice
service with `You` and the instance's typed damage-source binding. This is a
category-specific prerequisite. General production `ChooseADamageSource` stays
rejected until complete legal-source enumeration is available.

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

`mage.RedirectDamageFromPermanentThisTurn(source mage.PermanentRef,
player mage.PlayerRef, destination mage.PermanentRef) mage.ResolvingEffect`
captures all three references at resolution. It redirects each positive damage
packet from that exact source to that saved player until cleanup. The original
destination must be present as a creature or planeswalker when damage occurs.
Control changes do not change the saved player; a phase cycle keeps identity;
a returned source or destination cannot inherit the effect. Unbound identities at
installation produce no rule. Captured identities permit installation after
departure or phasing, without making those objects available for damage. Source information on a pending damage packet can
survive source departure. The effect is unlimited and is not consumed by damage.

RON supports the resolving action shape
`CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(
<permanent reference>, Player(<player reference>)),
[DealToCreatureOrPlaneswalkerInstead(<permanent reference>)], UntilEndOfTurn)`.
References use the existing lexical scopes and fresh factory bindings. Other
events, optional operations, collections and durations remain rejected.
Complete Shimian Night Stalker gameplay maps to `TestRONTemporaryRedirectGameplay`
and `TestRONTemporaryRedirectCopyAndInstances`.

`Game.WaitingRedirectionPermanentSources` also reads the saved source and
destination of this typed rule. Both remain referred objects until cleanup,
even when the destination cannot currently receive redirected damage. Current
or exact retained information supplies choice filters. The existing collector
deduplicates identities and returns isolated snapshots.
`chosenDamageSourcePlan.waitingRedirectionChoice(instanceBindings)` connects
this category to the existing RON choice/prevention plans. It is category-only;
general `ChooseADamageSource` stays rejected pending complete enumeration.
Legacy ID-only redirection installers retain their behavior and are not enumerated.

### Selected-source permanent-recipient redirection

The resolving RON shape is:

```text
CreateFutureReplaceWouldDealDamage(
  NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
    TheChosenDamageSource, Permanent(<permanent reference>)),
  [DealToPlayerInstead(<player reference>)])
```

`compileSelectedSourceRedirect(Node, scope) (effectFactory, error)` requires a
lexically bound source-choice plan, one typed permanent recipient and exactly
one player-redirection operation. Each factory uses its own source binding and
existing target/player bindings. It lowers to
`mage.RedirectNextDamageFromSelectedSource(DamageSourceRef, PermanentRef,
PlayerRef, DamageSourceFilter) ResolvingEffect`. The choice plan supplies the
source filter for installation and damage-time rechecks.

The engine keeps exact identities, saves the destination player, redirects one
positive damage event until cleanup and continues selected permanent spells
before entry effects. All matching simultaneous packets use the rule together.
Other events, recipients, operations, unbound references and wrong arities remain
errors. `ChooseADamageSource` remains rejected in production until all its source
categories exist; the adapter does not provide a partial source list.

`Game.WaitingRedirectionSpellSources` supplies exact selected spells to the shared
waiting-reference choice service alongside existing permanent redirection refs.
Native and RON tests are mapped in `docs/engine-rules-contracts.md`. No RON source
record or vocabulary hash changed.

### Face-up command-object damage sources

`mage.ChooseDamageSourceFromCommandObjects(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses only face-up engine objects installed in the command
zone. The choice occurs at resolution and does not target. An object need not
have a damage ability. Invalid declarations fail definition validation; empty
choices and absent players clear only this result binding.

`Game.AddCommandObject(Card, bool) (uuid.UUID, error)` supplies a fresh visit ID.
The caller must remove the object from its previous zone first. Nil objects,
missing IDs and duplicate command IDs fail. `RemoveCommandObject(visit)` and
`SetCommandObjectFaceDown(visit, bool)` use that visit, so an old reference cannot
remove or change a returned object. These services do not grant casting
permission or install command abilities. `Game.CommandDamageSources()` returns
isolated face-up snapshots in insertion order. `DamageSource.CommandVisit()`
keeps command entries separate from permanent incarnations and stack spells.
Departure information and game clones keep exact visits separate.

RON `chosenDamageSourcePlan.commandChoice(instanceBindings)` lowers the validated
`ChooseADamageSource(filter)` AST to this engine service with `You` and its own
binding. Exact and selected-source prevention and selected-source redirection
accept these identities and use the existing damage and replacement pipeline.
This is one source category. Complete source enumeration, including explicit
references held by waiting rules, still precedes unrestricted production RON
source choices. No full-card unlock or production registry change is claimed.

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

### Waiting callback modifier source choice

`mage.ChooseDamageSourceFromWaitingModifierReferences(player, filter, binding)`
chooses among the explicit sources of active registered mill/life-gain modifiers
and their static factories. `Game.WaitingModifierPermanentSources()` reads that
category without calling callbacks or factories. Registered sources need their
saved live unphased incarnation; static sources also need their current ability.
The result uses the existing typed `DamageSourceBinding` and shared source filters.

RON `ChooseADamageSource(filter)` has a category adapter
`chosenDamageSourcePlan.waitingModifierChoice(instanceBindings)` for this context.
It uses `You` and the plan's instance binding. This prerequisite does not accept
unrestricted production source choices. Objects hidden in callback closures and
other legal categories need separate reference contracts.


### Waiting empty-library replacement source choice

`mage.ChooseDamageSourceFromWaitingEmptyLibraryReferences(player, filter, binding)`
chooses among the explicit sources of active empty-library draw replacements.
`Game.WaitingEmptyLibraryPermanentSources()` reads this category without calling
callbacks. Registration saves the current live source incarnation. A phased,
departed or returned source cannot supply the old registration. Phase-in restores
the same registration; a returned source requires a fresh registration.

The RON `chosenDamageSourcePlan.waitingEmptyLibraryChoice(instanceBindings)`
adapter lowers validated `ChooseADamageSource(filter)` with `You` and its own
binding. It completes this category only. Full production source choices remain
rejected until the legal source list is complete.

### Persistent legacy replacement source references

`mage.ChooseDamageSourceFromWaitingPersistentReplacementReferences(player,
filter, binding)` supplies one CR 609.7a source category.
`Game.WaitingPersistentReplacementPermanentSources()` reads the exact explicit
sources of registered life-gain-to-draw, sanctuary normal-draw and counter-doubling
replacements. General filters supply no matching-object references. Queries do
not execute replacements, make choices or modify registrations.

`SetLichActive(playerID, sourceID)`,
`AddIslandSanctuaryReplacement(playerID, sourceID)` and
`AddCounterDoubler(sourceID, counterType, filter)` save the live incarnation at
registration. Execution and enumeration require that incarnation to remain live
and unphased. Phasing suspends the registration; phase-in restores it. Departure
ends it, and a return needs a new registration. Registration before entry cannot
follow a later entry. These separate registrations keep their existing explicit
player and filter contracts; source ability removal does not remove them.

RON `ChooseADamageSource(filter)` has a validated per-instance
`waitingPersistentReplacementChoice` lowering for this category. Full production
source choices remain rejected until the complete legal list is established.
Evidence: native `TestWaitingPersistentReplacementSources*` and RON
`TestRONWaitingPersistentReplacementSource*`. The latter use full-record costs
and choice/prevention ASTs for six Circles and Greater Realm of Preservation in
a test-only category context. They do not establish full-card loading.

### Typed untap-step replacement source references

`mage.ChooseDamageSourceFromWaitingUntapSkipReferences(player, filter, binding)`
supplies the explicit source category of active typed `SkipUntapSteps` rules.
`Game.WaitingUntapSkipPermanentSources() []DamageSource` reads both replacement
lists without running replacements, choices, factories or layer passes. Each
source must keep the registered incarnation and the matching live static ability.
Phasing suspends availability; phase-in restores it. Departure and ability
removal stop it; a returned permanent needs its new static registration.

RON `chosenDamageSourcePlan.waitingUntapSkipChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and its per-instance binding.
The common source filter and definition validation still apply. Empty or
nonmatching lists and absent players clear only that result. This category does
not complete CR 609.7a enumeration; production declarations remain rejected.
Evidence: `TestWaitingUntapSkipSources*` and
`TestRONWaitingUntapSkipSource*`, including actual RON Stasis as a source and
full-record protection costs and prevention ASTs in a test-only category context.

### Typed discard replacement source references

`mage.DiscardToLibraryReplacement() ContinuousEffect` retains its existing
calling form and now constructs a typed ability-layer effect. Its replacement
requires the original live, unphased incarnation and matching static ability.
Each effect-caused discard reads the current source controller. Declining keeps
the normal discard; cost discards cannot use the optional replacement.

`Game.WaitingDiscardPermanentSources() []DamageSource` reads exact explicit
sources from both replacement lists without choices, callbacks or layer passes.
`mage.ChooseDamageSourceFromWaitingDiscardReferences(player, filter, binding)`
uses shared source filters, decisions and resolution bindings.
RON `chosenDamageSourcePlan.waitingDiscardChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and its instance binding.
Empty lists and absent players clear only that binding. Complete production
source choices still reject until the remaining CR 609.7a audit is complete.
This does not lower Library of Leng's `ReplaceWouldDiscard` declaration.

Evidence: `TestWaitingDiscardSourcesLifecycle`, `Choices`, `Execution` and
`InvalidRegistrations`; `TestRONWaitingDiscardSourceProtection`, `Invalid` and
`TimingAndRechecks`. Seven protection records supply their complete cost,
choice and prevention ASTs in a test-only category context. This establishes
category gameplay evidence, not complete production card loading.

### Exact subjects of waiting exile replacements

`Game.WaitingExileSources() []DamageSource` reads exact spell and permanent
subjects of active `AddExileIfWouldGoToGraveyardThisTurn` rules. A live or
resolving spell registration saves its stack identity. Its continuation saves
only the permanent that spell becomes, before entry effects run. Departure and
phasing retain both references until cleanup. Queries use current or retained
information, deduplicate exact identities and return separate snapshots.
Legacy registrations before casting retain their calling form and card marker;
they supply no exact waiting reference.

`mage.ChooseDamageSourceFromWaitingExileReferences(player, filter, binding)`
uses shared source filters, decisions, declaration validation and resolution
frames. RON `chosenDamageSourcePlan.waitingExileChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and its instance binding.
Missing or nonmatching subjects and absent players clear only that result.
Full production source choices still reject until the remaining audit is complete.

Evidence: `TestWaitingExileSources*` and `TestRONWaitingExileSource*`.
The seven protection names supply their complete activation cost, choice and
prevention ASTs in a test-only category context. The spell test also proves
selected-source prevention continuation through the normal engine entry path.
This is category gameplay evidence; it does not establish full-card loading.

## RON conditional type and equal base values

A conditional static body may contain one
`PermanentLayerEffect(ThisPermanent | HostPermanent, [modifications])` with
one `SetPowerAndToughnessBoth(value)` and one or more
`AddCardtype(Artifact | Creature)` operations, including Creature. HostPermanent
requires an Aura declaration. The existing live static P/T value grammar
applies, including `ManaValueOfPermanent(HostPermanent)`. Other modifications,
duplicate base declarations, unbound references and resolution-only values
reject before factory publication. Other mixed body rules still reject.

The compiler lowers the complete list to one `Static(WhileWithSharedSubjects(condition, ...))`
rule. The engine selects the condition and exact subjects before the first
layer, retains them for the pass, and reads values in their applicable layer.
Animate Artifact therefore keeps its base P/T after its own type addition
makes the host a creature. A host that was already a creature is unaffected.
A fresh pass checks the condition again. This does not add sticky effects or
collective dynamic base-value lowering.

### RON static bands-with-other grants

The compiler accepts
`EachPermanentLayerEffect(filter, [AddAbility([BandsWithOthers(quality)])])`.
The collection selector uses existing static permanent filters, including
`ControlledByAPlayer(SinglePlayer(You))`. The quality is a pure permanent filter;
source/controller bindings, target references and empty compositions are
rejected. `And`, `IsSupertype`, `IsCardtype` and `IsNamed(NamedCard("name"))`
use their existing validated filter plans. Each factory builds its own grant.

This shape lowers to `GrantEachBandsWithOther(PermanentSet, PermanentFilter)`
inside `Static(While(...))`. The engine checks qualities at band formation and
combat damage assignment. All members of a quality band must match the quality;
only one matching member must hold the ability. Static land recipients must
also match the grant's color and current controller selection.

The supported context is an unconditional static collection layer with one or
more bands-with-other grants. Combined unrelated modifications, Aura/singular
subjects, printed declarations, token abilities and temporary grants remain
declaration errors. Ordinary banding removal uses the shared engine rule and
removes bands-with-other grants too.

### RON selective bands-with-other removal

`CreatePermanentLayerEffectUntil(subject, [LosesAbility(AnyBandsWithOthers)],
UntilEndOfTurn)` lowers to `RemoveBandsWithOther(PermanentRef)` in the existing
singular resolving modification context. `AnyBandsWithOthers` is a symbol;
call forms and use in `AddAbility` are invalid. Typed subject scope checks and
supported duration checks apply. Static collection removal remains unsupported.

The operation removes every quality grant in layer 6, preserves ordinary
banding, and uses the saved subject incarnation until cleanup. A later layer
grant can restore the ability. A formed band persists, while damage assignment
checks the abilities that remain. Shelkin Brownie uses selective removal;
Tolaria combines it with ordinary banding removal and an upkeep restriction.

### Precast exile references in damage-source choices

`mage.Game.CaptureExiledDamageSource(visit)` and
`mage.Game.StackBoundExileSources()` expose exact exile visits and references
saved by stack card/card-set payment bindings or nested event-card captures.
Use `mage.ChooseDamageSourceFromStackExileBindings(player, filter, binding)`
only for that category. A face-down card supplies no public characteristics.
Departure preserves the old visit's information; a return creates a new object.
A choice of the exile object does not follow a later spell or permanent.
`GrantCastFromExile` keeps its existing signature and grants permission only for
the present exile visit; departure or re-exile ends that permission.

RON's private `chosenDamageSourcePlan.stackExileChoice(instanceBindings)`
adapter consumes validated `ChooseADamageSource(filter)` plans with separate
instance bindings. It shares the existing exact-source prevention lowering.
General source choice remains unsupported in complete RON records until the
full CR 609.7a source list and remaining reference contracts are established.

### Precast graveyard references on the stack

`mage.Game.CaptureGraveyardDamageSource(visit)` and
`mage.Game.StackGraveyardSources()` read exact graveyard visits referred to by
stack targets, card/card-set payments and nested event-card captures.
`mage.ChooseDamageSourceFromStackGraveyardReferences(player, filter, binding)`
uses only that category. Current facts are read while the visit is present;
departure facts stay separate from returned cards. Copies retain original target
visits; retargeting captures the new target. Final referring stack departure
ends category availability. A graveyard source does not become a later spell
or permanent source. Game clones and queries keep separate mutable facts.

RON's private `chosenDamageSourcePlan.stackGraveyardChoice(instanceBindings)`
lowers `ChooseADamageSource(filter)` with separate instance bindings and the
existing selected-source prevention body. General production source choice still
rejects until the complete CR 609.7a list is supported. Waiting references and
delayed card frames remain separate contexts.

## RON hand source references on the stack

Validated `ChooseADamageSource(filter)` has an internal
`chosenDamageSourcePlan.stackHandChoice(instanceBindings)` lowering for exact
hand visits in singular/plural payment card bindings and nested event captures
on stack objects. It calls
`mage.ChooseDamageSourceFromStackHandReferences(mage.You, filter, binding)`.
The separate instance binding is consumed by the existing
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(You)), [PreventThatDamage])` lowering.

Hidden hands supply no public characteristics. Public hands use current
visibility; saved departures keep only the facts public at departure. Cost
reveals keep their exact card public until the paid stack object leaves, including
resolution. Copies retain cost references without extending the original reveal. Returned cards, later spells
and permanents keep separate identities. Invalid declarations and premature
reads reject. Empty choices clear the result. General production source choice
remains unsupported until complete enumeration and its opaque-reference audit.


## RON choices with private hand inspection

Validated `ChooseADamageSource(filter)` plans also have a private
`chosenDamageSourcePlan.inspectableStackHandChoice(instanceBindings)` lowering.
It calls `mage.ChooseDamageSourceFromInspectableStackHandReferences` with `You`
and the instance's own binding. The choosing player can inspect their referenced
hand cards; other players receive public information only. Filters use that
player's information at resolution. Inspection does not reveal the hand.

`mage.Game.CaptureHandDamageSourceFor(viewer, visit)` reads a live visit and
`mage.Game.StackHandSourcesFor(viewer)` reads the existing stack-reference
category. Departures retain private facts and separate public visibility.
Queries and clones preserve exact visits and separate mutable snapshots.
The original public query and choice APIs remain available.

The chosen-source future prevention and native selected-source redirection
recheck actual hand characteristics through the engine's rules reader.
A private matching choice can therefore install a shield without revealing the
card. Returned visits, later spells and permanents remain separate objects.
Invalid declarations, missing bindings and reads before producers still reject;
empty choices clear the binding. Complete production `ChooseADamageSource`
remains unsupported pending the remaining reference categories and aggregation.


## Waiting damage-rule exile references

`Game.WaitingPreventionExileSources() []DamageSource` and
`Game.WaitingRedirectionExileSources() []DamageSource` read exact exile visits
saved by active exact/selected-source prevention and selected-source redirection.
Both reuse the command category's shared active-replacement walker and the
existing exile information service. They do not execute replacement callbacks.
Queries deduplicate exact visits and return separate mutable snapshots.
Consumption ends availability; end-of-turn cleanup removes temporary rules.
Cycle registrations keep their existing layer-cycle lifetime.

`ChooseDamageSourceFromWaitingPrecastExileReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` combines
these categories and resolves the choosing player at resolution. An absent player
or empty matching set clears the result. Definition validation rejects invalid
players, filters, bindings and reads before the producer. Other categories,
including exile-instead replacement subjects, retain their separate APIs.

Current exile facts apply while the visit remains. Departure retains the facts
at that visit's departure. Face-down exile has no characteristics, even when a
player can inspect the card. A later visit, spell or permanent cannot inherit
the source identity (CR 400.7, 406.3a, 609.7a–b). Existing damage replacement
ordering, rechecks, simultaneous packets, shield consumption and clone services
own execution; this query adds no replacement policy or zone transition.

RON `chosenDamageSourcePlan.waitingPrecastExileChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` to this category with `You` and a fresh
instance binding. Complete production enumeration remains rejected until other
required contexts are complete. No new RON spelling or source edit is needed.

Evidence: `TestWaitingExileSources`, `Declarations`, `VisibilityAndBoundaries`
and `RechecksAndBatch`; `TestRONWaitingPrecastExileProtection` maps the real
choice, payment and future-prevention declarations of all six Circles of
Protection and Greater Realm of Preservation to live/departed/returned, copied
and cloned waiting references. `TestRONWaitingPrecastExileInvalid` covers
scope/declaration rejection. These are subrule mappings, not full-card coverage.


## Waiting damage-rule hand references

`Game.WaitingPreventionHandSourcesFor(viewer uuid.UUID) []DamageSource` and
`Game.WaitingRedirectionHandSourcesFor(viewer uuid.UUID) []DamageSource` read
exact hand visits retained by active exact/selected-source prevention and
selected-source redirection. The shared replacement walker checks ordinary and
cycle rule lifetime without calling replacement callbacks, excludes malformed
and other-zone identities, deduplicates visits, and returns separate snapshots.
Consumption and cleanup end availability; cycle rules retain their layer lifetime.
No stack reference is required once the waiting rule retains the visit.

The existing hand information reader supplies current facts with the viewer's
inspection rights. The hand player can inspect private characteristics; other,
nil and unknown viewers receive public facts only. Departure keeps that visit's
private facts and separate public visibility. Later reveals and returns cannot
rewrite it. A waiting reference does not extend a cost reveal after the paying
stack object leaves. Inspection adds no unreferenced card to the source list.

`ChooseDamageSourceFromWaitingPrecastHandReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` combines
both categories using the choosing player's information at resolution, even when
that player differs from the controller. Empty matches or an absent player clear
the result. Definition validation checks player, filter, binding and producer
order. Options and saved choices keep separate mutable data.

RON `chosenDamageSourcePlan.waitingPrecastHandChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and the instance binding.
Existing future prevention consumes it. General production enumeration still
rejects pending other categories; this API supplies only waiting hand references.
The engine's rules information reader still rechecks actual characteristics for
prevention and redirection. Hidden information does not stop an installed shield.
A returned visit, later spell or permanent cannot inherit it. Replacement order,
simultaneous damage, consumption, phasing and clone behavior reuse existing rules.

Rules: CR 400.2/400.7, 402.3, 609.7a–b, 615.9 and 701.20a.
Evidence: native `TestWaitingHandSources`, `Declarations`,
`VisibilityAndBoundaries`, `RechecksAndBatch`, `Inspection` and `RevealLifetime`;
RON `TestRONWaitingPrecastHandProtection` and `Invalid`. All six Circles of
Protection and Greater Realm of Preservation map their real payment, choice and
future-prevention declarations to this waiting-reference context. These mappings
establish subrule gameplay, not full-card acceptance.

## RON source choices from delayed hand and exile card frames

Validated `ChooseADamageSource(filter)` plans have an internal
`chosenDamageSourcePlan.waitingDelayedCardChoice(instanceBindings)` lowering.
It calls `mage.ChooseDamageSourceFromWaitingDelayedCardBindings(mage.You,
filter, binding)` with a separate binding for the factory instance. Existing
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(You)), [PreventThatDamage])` consumes that binding.

The source category includes singular and plural hand/exile card bindings held
by waiting delayed bodies. Existing `CreateFutureTrigger/2` lowering reuses
`AtNext`, which now saves these frames. Trigger placement transfers saved card
references to stack queries. Each delayed execution restores its card frame
without changing the outer frame. Hand choices use the choosing player's private
inspection rights; other hands and face-down exile keep hidden characteristics.
Departure retains exact visit facts, and a returned card cannot replace them.
A saved reference cannot extend a cost reveal. Empty choices clear their binding;
invalid declarations and premature reads reject through shared validation.

`TestRONWaitingDelayedCardProtection` maps these declarations to Circle of
Protection: Black, Blue, Green, Red, White, Artifacts and Greater Realm of
Preservation. It executes hand draw-event and exile payment producers, RON
next-end-step bodies and paid protection activations across live, departed,
returned, copied and cloned contexts. `TestRONWaitingDelayedCardInvalid` covers
invalid syntax and binding declarations. General production source choice stays
rejected until all CR 609.7a categories and the opaque-reference audit are complete.
Waiting graveyard card frames have a separate contract below. Remaining precast
target contexts still prevent a complete legal source list.

## RON source choices from delayed graveyard card frames

Validated `ChooseADamageSource(filter)` plans support internal
`chosenDamageSourcePlan.waitingDelayedGraveyardChoice(instanceBindings)` lowering
to `mage.ChooseDamageSourceFromWaitingDelayedGraveyardBindings(mage.You,
filter, binding)`. Each instance supplies its own binding. Existing
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(You)), [PreventThatDamage])` consumes the result.

`mage.Game.WaitingDelayedGraveyardSources() []mage.DamageSource` supplies exact
singular/plural graveyard visits saved by waiting delayed bodies and modes.
Current public facts or retained departure facts cannot follow a returned card.
Existing `CreateFutureTrigger/2` lowering uses shared card capture and restoration.
Detection, placement and final stack departure retain their existing lifetimes.

`TestRONWaitingDelayedGraveyardProtection` maps all six Circles of Protection and
Greater Realm of Preservation to this context using a real discard payment,
compiled RON delayed body, real protection payment and prevention declarations.
It covers live, departed, returned, copied and cloned choices, resolution timing,
failed payments, shield consumption and empty-category clearing. `Invalid`
checks malformed syntax, missing instance bindings and premature reads.
General production source choice remains rejected until all remaining CR 609.7a
categories and opaque references are audited. No full-card unlock is claimed.


## Exact hand-target source references

`Game.StackHandSources() []DamageSource` and
`Game.StackHandSourcesFor(viewer uuid.UUID) []DamageSource` now include exact
hand targets, alongside their existing payment/event/reveal references.
Announcement captures the physical card ID, hand visit and zone before payment.
A target discarded as a cost keeps its hand visit; it supplies no graveyard
reference. Failed payment installs no target frame. Stack insertion retains the
saved visit and also captures targets for direct stack insertion. Target
incarnations also use the hand visit for resolution legality. An absent visit
or a conflicting saved incarnation supplies no source. Queries only read
identities listed in the stack object's current targets; old retarget snapshots
cannot add choices. Sorting and exact deduplication use the existing reader.

A live visit supplies current facts. Departure supplies the same visit's saved
facts and public visibility. The hand player can inspect private facts; other
players receive only public information. Targeting does not reveal a hand card.
A later reveal cannot expose a hidden departed visit. Queries return separate
snapshots. No source continues to a returned card, spell or permanent.

Copies retain exact targets. An explicit legal new choice saves its current
visit, including a returned card with the same physical ID. Cancellation keeps
the original visit. Copy maps are separate; game clones share immutable target
identities through copy-on-write. The original returned hand target is illegal
at resolution. Illegal targets remain available as source references until the
last referring stack object leaves. Resolving hand frames retain these target
references until resolution ends, including cloned frames.

RON `chosenDamageSourcePlan.inspectableStackHandChoice(instanceBindings)`
reuses `ChooseDamageSourceFromInspectableStackHandReferences(You, filter,
binding)` for validated `ChooseADamageSource(filter)` in this context. Each
instance retains its own binding. The existing exact future-prevention adapter
consumes it. The public `stackHandChoice` lowering remains available. General
production source choice remains rejected pending complete source aggregation.
No additional source-choice syntax or targeting declaration is introduced.

Rules: CR 400.7, 402.3, 601.2c, 608.2b, 609.7a–b and 707.10c.
Phasing does not apply to hand cards. Payment rollback, damage batches,
replacement ordering, source-property rechecks and shield consumption use their
existing engine services.

Evidence: native `TestHandTargetSourcesIdentity`, `LegalityAndRetarget`,
`Boundaries`, `Visibility`, `ResolutionAndClone`, `SpellCopy`, and
`PaymentDeparture` and `FailedPayment`; RON
`TestRONHandTargetSourcesProtection` and `Invalid`. The RON test maps all six
Circles of Protection and Greater Realm of Preservation to their complete real
cost/choice/future-prevention declarations in this hand-target context. It does
not claim complete-card gameplay or compilation gains.

### RON stack exile target source context

The internal `chosenDamageSourcePlan.stackExileChoice(instanceBindings)` adapter
for `ChooseADamageSource(filter)` now includes exact exile targets on stack and
resolving objects. It reuses `ChooseDamageSourceFromStackExileBindings` and the
existing per-instance `DamageSourceBinding`; no RON spelling changes.
Announcement saves the exile visit before payment. Copies keep it unless a new
legal choice saves a current visit. Returned targets are illegal. Face-down
references expose no characteristics. Current/departure information, clones,
reference lifetime and selected-source shields use the native services.
General `ChooseADamageSource` compilation remains rejected until enumeration
is complete. No new exile target declaration is introduced.

RON graveyard damage-source choices reuse exact visits from targets and card
bindings on stack and resolving objects. The body retains its own references
through nested resolution and cloning; completing the body ends availability.
`ChooseDamageSourceFromStackGraveyardReferences` uses these visits with a
per-instance `DamageSourceBinding`. General RON `ChooseADamageSource` still
requires complete source enumeration. See the resolving graveyard source
contract and `TestRONResolvingGraveyardSourcesProtection` for this context.

## Resolving permanent and spell source references

`Game.StackAbilityPermanentSources`, `StackTargetPermanentSources`,
`StackBoundPermanentSources`, `StackTargetSpellSources`, `StackBoundSpellSources`
and `StackBoundDamageSources` include stack objects and retained resolving
frames. Their signatures remain `() []DamageSource`. The queue removes an
object before its body runs; this does not end its reference lifetime.
`Stack.referenceObjects()` supplies both contexts. Nested resolution restores
outer frames, and clones copy each retained frame with the existing stack clone
service. Final body completion ends this category's availability.

The queries retain exact incarnations and spell identities, remove duplicates,
and return separate card facts. Current information, phasing snapshots and
departure information use the existing readers. A returned permanent cannot
replace a saved incarnation. Spell departure also updates target snapshots on
resolving referrers through `Stack.saveSpellInformation`; later casts cannot
replace the saved spell. Invalid identities and nil bindings supply no choices.
No query invokes an effect callback or invents a reference from its closure.
Opaque callback references still need an explicit audit before general source
aggregation. CR 400.7, 608.2n and 609.7a–b define these lifetimes and identities.

RON's private `chosenDamageSourcePlan.stackAbilityChoice(instanceBindings)
mage.ResolvingEffect` lowers validated `ChooseADamageSource(filter)` with
`mage.You` and the instance's `*mage.DamageSourceBinding` to
`ChooseDamageSourceFromStackAbilities(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect`. General source choice remains rejected.

Evidence: `TestResolvingObjectSourcesFrames` covers all six queries, duplicate
references, query isolation, departure during resolution, returned permanents,
phasing, nested resolution and clones. `TestResolvingObjectSourcesInvalidReferences`
excludes invalid and unbound references. `TestRONResolvingObjectSourcesProtection`
adds 49 subrule scenarios from the six Circles of Protection and Greater Realm
of Preservation: real filter, payment and future prevention lowering, own
resolving ability source, failed payment, return, departure, phase cycles,
copies, clones and shield consumption. Metadata is synthetic to make the
ability source match each real record's filter. This is category evidence,
not complete-card gameplay evidence. Invalid order and absent instance bindings
remain errors. Existing event/death/delayed tests now require the resolving
object's own reference and still require its removal after body completion.

### Waiting delayed damage-body source references

`Game.WaitingDelayedPermanentSources() []DamageSource` includes the exact
permanent source when a waiting body contains single-recipient
`DealDamageFrom(This, recipient, amount)` or
`DealDamageFrom(DamageSourceOf(This), recipient, amount)`. The recipient can
be a player or permanent. Zero damage still declares the source reference.
Sequences, captured delayed bodies and modes use the existing declaration
traversal. Enumeration does not resolve the recipient, amount or body.

RON's private
`chosenDamageSourcePlan.waitingDelayedPermanentChoice(instanceBindings)
mage.ResolvingEffect` lowers validated `ChooseADamageSource(filter)` with
`mage.You` and the instance's `*mage.DamageSourceBinding` to
`ChooseDamageSourceFromWaitingDelayedBindings(PlayerRef, DamageSourceFilter,
*DamageSourceBinding) ResolvingEffect`. Existing `CreateFutureTrigger` and
`PermanentDealsDamage(ThisPermanent, value, recipient)` lowering supplies the
delayed source read. General source choice remains rejected until all required
reference categories and their aggregation have complete contracts.

The original source incarnation supplies current or retained information after
phasing or departure. A returned card cannot replace it. Registration occurs
at resolution, and one-shot detection ends this waiting category. Clones keep
separate frames. Other bound damage sources keep their separate saved-frame
contract. A future nested delay does not declare its source for an outer delay.

Evidence: `TestWaitingDelayedDamageBodySources`, `Boundaries` and
`PermanentRecipient` cover both native source forms, sequences, modes, exact
identity, LKI, control and color changes, phasing, clones, invalid declarations,
zero amounts and permanent recipients. `TestRONWaitingDelayedDamageBodyProtection`
maps the real filters, costs and future shields of all six Circles and Greater
Realm to 49 scenarios with a compiled synthetic RON damage-only delayed source.
These are subrule scenarios, not complete-card coverage.

## Waiting delayed sources of multi-recipient damage

RON permanent collection, per-player and packet damage lowers its scoped self
source through `compileSelfDamageSource(Node, scope, bool)` into
`mage.DamageSourceOf(PermanentRef)`. Spell self sources keep `mage.This`.
The supported self aliases use the existing permanent scope. In particular,
`Trigger_ThatPermanent` in a source-tapped trigger keeps that source through a
nested future step trigger. Other permanent sources retain their separate
single-recipient adapter; this helper does not admit target or attachment reads.

Inside `CreateFutureTrigger(event, ActionList([...]))`, these existing forms now
supply the exact permanent source to `Game.WaitingDelayedPermanentSources`:

- `PermanentDealsDamage(ThisPermanent, amount, EachPermanent(filter))`
- `PermanentDealsDamage(ThisPermanent, amount, EachPlayer(AnyPlayer|Opponent))`
- `PermanentDealsDamage(ThisPermanent, amount, MultipleRecipients([...]))`
- `PermanentDealsMultipleDamage(ThisPermanent, [DamageToRecipients(amount,
  recipient), ...])`

Each form keeps its existing amount, filter, recipient and lexical validation.
Sequences and saved modes preserve the declaration. Source enumeration evaluates
neither recipients nor amounts. Empty recipient collections and zero damage do
not remove the source. A nested future body does not declare an outer self read.
The existing scoped `chosenDamageSourcePlan.waitingDelayedPermanentChoice`
adapter can select these sources, with independent bindings per instance.
General `ChooseADamageSource` remains rejected until every category is complete.

Evidence: `TestRONWaitingDelayedMultiDamageProtection` maps the six Circle of
Protection names and Greater Realm of Preservation to 147 real-record subrule
scenarios; `TestRONWaitingDelayedMultiDamageInstances` proves separate factory
instances and source-tapped aliases; `TestRONWaitingDelayedMultiDamageInvalid`
proves declaration rejection. This is subrule coverage, not complete-card proof.

### RON waiting delayed property-branch references

In a supported `CreateFutureTrigger(event, ActionList([...]))` body,
`If(PermanentPassesFilter(permanent, filter), [actions])`, the corresponding
`Unless`, and `IfElse(condition, [matching], [nonmatching])` retain explicit
permanent self references from the property condition and both action branches.
`compileResolvingPermanentBranch(Node, scope)` continues to build fresh scoped
references and lower to `IfPermanentMatches`, `UnlessPermanentMatches`, or
`IfPermanentMatchesElse`. Existing property filters and declaration restrictions
apply. Source aliases use the existing permanent-reference scope.

`Game.WaitingDelayedPermanentSources` reads these declarations without executing
the condition or a branch. False conditions and unselected branches still refer
to their declared objects. Saved sequences and modes keep the exact source
incarnation. The current or retained information and one-shot waiting lifetime
use the existing delayed services. Nested future producers do not expose their
future bodies through the outer branch. No new syntax or general source-choice
adapter is added. Other conditional implementations still need their own
reference contracts. Combat branches have the explicit contract below.

Evidence: `TestRONWaitingDelayedConditionalProtection`,
`TestRONWaitingDelayedConditionalInstances`, and
`TestRONWaitingDelayedConditionalDeclarations`. The seven protection records
supply real filters, costs and shields; their complete general source selection
remains rejected. These tests establish subrule interactions, not complete-card
coverage.


### RON waiting delayed self-recipient references

The existing `compileReferencedPermanentDamage(Node, scope)` and
`compileDamagePackets(Node, scope)` adapters use
`compileDamagePacketRecipient(Node, scope)` to lower `Permanent(ThisPermanent)`
to `mage.This`. Within a supported `CreateFutureTrigger` body, that recipient
now declares the saved source to `Game.WaitingDelayedPermanentSources`, even
when the damage source is another permanent. Source-tapped aliases retain their
existing lexical lowering; unbound and incompatible references remain errors.

Supported examples are `PermanentDealsDamage(Ref_TargetPermanent, Integer(2),
Permanent(ThisPermanent))` with a declared permanent target, and
`PermanentDealsMultipleDamage(ThisPermanent, [DamageToRecipients(Integer(1),
Permanent(ThisPermanent)), ...])`. Packet sources retain the existing self-source
restriction. No new spellings or scope permissions are introduced.

Native declaration reads do not resolve recipients or amounts. Saved sequences,
property branches and modes propagate the declaration. Zero amounts still refer
to the recipient. Bound and attached recipients do not imply self, and a nested
future body does not supply an outer reference. Exact captures, current/LKI facts,
phasing, copies, clones and one-shot waiting lifetime use the existing services.
General source selection still rejects incomplete protection records.

Evidence: `TestRONWaitingDelayedRecipientProtection` covers 98 real-record
subrule scenarios for six Circles of Protection and Greater Realm of Preservation.
`TestRONWaitingDelayedRecipientInvalid` covers declaration rejection. Native
`TestWaitingDelayedRecipientReferences`, `Boundaries` and `PacketGameplay` prove
recipient-only declarations, identity, pure queries and simultaneous execution.


### RON waiting delayed combat-branch references

Within a supported `CreateFutureTrigger(event, ActionList([...]))` body,
`If(PermanentPassesFilter(permanent, IsAttacking), [actions])` and the
`IsBlocking` form retain explicit self references from the subject and body.
`compileResolvingPermanentBranch(Node, scope) (effectFactory, error)` reuses
`compilePermanentRef` and fresh instance bindings to lower these forms to
`mage.IfPermanentAttacking` and `mage.IfPermanentBlocking`. Existing target,
attachment and event scope restrictions apply. `Unless`, `IfElse`, tuple combat
filters, empty bodies and unbound subjects remain rejected in this context.

`Game.WaitingDelayedPermanentSources` reads declarations without checking combat
or running actions. Nested supported branches and sequences propagate body-only
reads; modes retain them without a selection. Nested future producers keep their
own boundary. Queries retain exact current/departure information after phasing or
departure; a return cannot replace the saved incarnation. The body checks current
combat membership at resolution. A false branch keeps its waiting reference but
does not consume a damage shield. Copies and clones use existing saved frames.

Evidence: `TestRONWaitingDelayedCombatProtection` supplies 98 real-record filter,
payment and shield interactions across seven protection names.
`TestRONWaitingDelayedCombatBodyInstances` proves body-only reads and instance/
clone independence. `TestRONWaitingDelayedCombatInvalid` covers declaration
rejection. Native `TestWaitingDelayedCombatReferences`, `Boundaries` and
`QueryPurity` cover declaration traversal, exact identity and pure queries.
Complete general source selection and complete protection-card coverage remain
unfinished. No new RON spelling or scope permission is introduced.

## Self references in waiting delayed payment branches

`Game.WaitingDelayedPermanentSources() []DamageSource` includes the saved source
when the body of `IfPaid` or `UnlessPaid`, or either body of `PayOrElse`, has an
explicit typed self read. Nested sequences, payment branches and saved modes
preserve the declaration. Enumeration does not resolve the payer, test a cost,
read an amount, execute a body or make a payment choice. It includes both bodies
before the payment result exists. A payer or cost reference alone does not declare
this body category. Future delayed producers keep their own lifetime boundary.

Existing RON lowering stays in `compileResolutionPayment` and
`compilePaymentBranches`. Supported adjacent `MayCost` / `PlayerMayCost` and
`If` / `Unless` payment continuations use their existing typed scopes. The fixed
mana `IfElse(CostWasPaid, [paid], [unpaid])` and inverse `CostWasntPaid` forms
lower to `PayOrElse`. This contract adds declaration traversal to their native
node; it adds no syntax or cost support. Dynamic mana, exact sacrifice, action
payment, payer and cost references need separate declaration contracts.

Current and departure information, phasing, return identity, deduplication,
query snapshot isolation, one-shot detection and clone lifetimes use the existing
waiting-reference services. Payment legality, rollback and body execution keep
the existing transaction. General `ChooseADamageSource` remains rejected.

Evidence: native `TestWaitingDelayedPaymentReferences`, `Boundaries` and
`QueryPurity`; RON `TestRONWaitingDelayedPaymentProtection`, `Invalid` and
`Instances`. Protection evidence uses complete records for six Circles of
Protection and Greater Realm of Preservation with the isolated waiting category.
It establishes subrule coverage, not complete card coverage.

## Self references in waiting delayed action payments

`PayActionOrElse(PlayerRef, ResolutionActionPayment, ResolvingEffect,
ResolvingEffect) ResolvingEffect` declares direct self references in either result
body. It also declares `This` in the source or recipient of `DamagePayment`, or
the subject of `DestroyPayment`. A source wrapped in `DamageSourceOf(This)`
qualifies. The declaration remains available before a payment result exists,
even for zero damage or a payment that cannot be made. Enumeration does not
prepare a payment, read an amount, resolve a player or run an action.

Existing `compileResolutionActionPayment` lowers adjacent `MayCost` or
`PlayerMayCost` with `GainLife`, `HavePermanentDealDamage` or `DestroyPermanent`
into this typed node. Supported continuations are `If`, `Unless` and `IfElse`
with `CostWasPaid` or `CostWasntPaid`. This slice adds the engine declaration
contract; it changes no RON spelling, lowering permission or payment semantics.
Player references, value references, opaque payment callbacks and other payment
nodes still need separate audits. A nested future producer keeps its own lifetime.

Saved sequences and modes keep the original incarnation. Current or departure
facts, phasing, return identity, deduplication and clones use the existing waiting
reference services. General `ChooseADamageSource` remains rejected.

Evidence: `TestWaitingDelayedActionPaymentReferences` and
`TestWaitingDelayedActionPaymentPurityAndBoundaries`; RON
`TestRONWaitingDelayedActionPaymentProtection`, `Invalid` and `Instances`.
The protection tests use real-record filters, costs and shields for six Circles
of Protection and Greater Realm of Preservation. They establish subrule coverage.

## Self references in waiting delayed counter payments

Existing `compileResolutionCounterPayment` lowering and `CreateFutureTrigger`
step bodies retain their supported syntax and scope. `RemoveCounters` accepts
one `ACounterOfTypeFromPermanent`, fixed nonnegative
`NumberCountersOfTypeFromPermanent(Integer(n), ...)`, or optional
`AnyNumberOfCountersOfTypeFromPermanent` instruction on `ThisPermanent`.
An adjacent `If` or `Unless` with `CostWasPaid` or `CostWasntPaid` selects a
nonempty continuation. Chosen results remain local to the paid continuation.

The lowered native `PayCountersOrElse` and `PayChosenCountersOrElse` now declare
the direct source subject and source reads in both native result branches to
`Game.WaitingDelayedPermanentSources`. The compiler does not add a separate
runtime. Queries do not make payment decisions, even for zero or impossible
payments. Saved source identity survives departure and phasing; counter payment
cannot follow a returned object. Future delayed producers and unaudited
player/value reads need separate contracts.

Evidence: `TestRONWaitingDelayedCounterPaymentProtection`, `Invalid` and
`Instances`, plus native `TestWaitingDelayedCounterPaymentReferences` and
`Boundaries`. The seven protection names gain subrule coverage; their complete
records still reject general damage-source aggregation.

## Self references in waiting delayed exact sacrifice payments

Existing `compileResolutionPayment(payment, continuation Node, ctx scope)
(effectFactory, error)` lowering accepts
`MayCost(SacrificePermanent(ThisPermanent))` and
`PlayerMayCost(player, SacrificePermanent(ThisPermanent))` in supported
`CreateFutureTrigger` resolving lists. The adjacent nonempty `If` or `Unless`
uses `CostWasPaid` or `CostWasntPaid`. All four polarities retain their existing
local result and per-instance scope.

The native `IfSacrificePaid(PlayerRef, PermanentRef, ResolvingEffect)` and
`UnlessSacrificePaid(PlayerRef, PermanentRef, ResolvingEffect)` now declare a
direct `This` payment subject or a typed source read in their continuation to
`Game.WaitingDelayedPermanentSources() []DamageSource`. A paid or unpaid
continuation qualifies before a payment decision exists. Sequences, modes and
nested audited payments preserve the declaration. Queries do not resolve the
payer or subject, check payment legality or offer a choice. Attached subjects,
player/value reads and future delayed producers need separate contracts.

The saved incarnation uses current or exact departure facts, including after
phasing. A returned object cannot inherit it. Sacrifice execution still requires
the original present subject under the payer's control and uses the existing
transaction and zone replacements.

Evidence: native `TestWaitingDelayedSacrificePaymentReferences` and `Boundaries`;
RON `TestRONWaitingDelayedSacrificePaymentProtection`, `Invalid` and `Instances`.
The seven protection names gain subrule coverage. General source aggregation
and their complete declarations remain unsupported.


## Waiting delayed dynamic mana-payment references

`IfManaPaid(PlayerRef, ManaCost, ResolutionValue, ResolvingEffect)` and
`UnlessManaPaid` declare source reads in their amount and continuation to
`Game.WaitingDelayedPermanentSources() []DamageSource`. Paid and unpaid
continuations qualify before payment. Sequences, modes and nested audited
payments preserve the declaration. Typed direct `This` power, toughness, mana
value and counter reads qualify, with either live-only or exact-information
semantics. Arithmetic traverses both operands without reading them; a zero
multiplier does not remove a reference.

Queries do not read amounts or players, inspect affordability, offer payment or
execute a body. Attached subjects, player-derived values, opaque legacy/custom
values and nested future producers are outside this contract. Fixed and
announced-X wrappers contain no permanent reference. Spell-value amounts keep
their separate spell reference contract. The engine retains the exact saved
source through departure, return and phasing. A returned incarnation cannot
replace it. Queries copy characteristic information. Trigger detection ends
waiting eligibility before pending placement; cloned games keep separate
lifetimes. Payment still uses the existing transaction and reads its amount once
before payment choices. No replacement ordering or simultaneous action changes.
CR 400.7, 603.7a–e, 608.2h, 609.7a–b and 702.26b establish these boundaries.

RON reuses `compileResolutionPayment`, `compileResolutionManaCost` and
`compileDelayedStep`. Supported resolving `MayCost` / `PlayerMayCost` with
`PayManaX(mana, amount)` require an adjacent nonempty `If` / `Unless` using bare
`CostWasPaid` / `CostWasntPaid`. Amounts remain nonnegative `Integer`, a bound
`ValueX`, or typed `ManaValueOfSpell`. This slice adds no new amount syntax or
`IfElse` support. Binding plans are built per factory instance.

Evidence: `TestWaitingDelayedManaPaymentReferences` covers 189 declaration,
nesting, mode and lifecycle cases. `Boundaries` checks pure queries, excluded
operands and invalid definitions. `TestRONWaitingDelayedManaPaymentProtection`
covers 98 real-record subrule scenarios for Circle of Protection: Artifacts,
Black, Blue, Green, Red, White and Greater Realm of Preservation, including
costs, shields, copies, clones and actual dynamic payment. `Invalid` rejects nine
malformed or unbound delayed declarations. `Instances` checks nested and adjacent
payments, separate targets, opposite results, mana spending and clone lifetime.
Existing `TestResolutionManaPayment*` and `TestRONResolutionMana*` retain amount
locking, mana solver, rollback, spell identity and complete In the Eye of Chaos
coverage. General source aggregation still rejects the seven protection cards.


## Waiting delayed typed player and value references

`Game.WaitingDelayedPermanentSources() []DamageSource` declares direct `This`
references inside `ControllerOf`, `OwnerOf` and `ControllerOfWithLKI` player
operands. This extends the preceding payment contracts: typed payer reads now
qualify in `IfPaid`, `UnlessPaid`, `PayOrElse`, required payments,
`IfManaPaid`, `UnlessManaPaid`, both counter-payment forms, both sacrifice-payment
forms and `PayActionOrElse`. Existing signatures and execution remain unchanged.

`LifeTotalOf`, `HandSizeOf`, `LifeGainedThisTurn`, `DamageTakenThisTurn` and
`ArtifactDamageTakenThisTurn` propagate their typed player declaration through
arithmetic. Direct `GainLifeFor`, `LoseLifeFor` and `DrawFor` bodies declare their
player and amount reads. Action payments declare the life-gain player and amount,
or damage amount and typed player recipient, as well as their existing source
and permanent recipient reads. Sequences, captured bodies, modes and existing
payment branches retain these declarations without evaluating any operand.

The internal helpers are `playerRefReadsSourcePermanent(PlayerRef) bool` and
`resolutionValueReadsSourcePermanent(ResolutionValue) bool`. They inspect only
known typed declarations. Attached, bound and opaque player wrappers remain
outside this direct-source contract. Ordinary damage-body player/amount operands,
other action families, filtered counts and cast/state/coin conditions still need
separate audits before general source aggregation. Nested future producers do
not expose their future body through the current waiting category.

Queries do not resolve players, calculate values, check affordability, call
callbacks, offer payment or execute a body. Zero values and impossible payments
retain references. Exact incarnation capture, current/departure information,
phasing information, snapshot copies, deduplication and clone lifetimes reuse the
existing delayed and source services. Returned objects cannot replace the saved
source (CR 400.7); detection ends one-shot waiting availability (CR 603.7a–e).
CR 609.7a allows the reference as a chosen source even after departure. Actual
payment, damage and prevention retain their existing transactions and replacement
ordering. Native live-only player reads remain live-only; RON controller reads
use `ControllerOfWithLKI` and retain the exact departure controller.

RON reuses `compilePlayerRef(Node, scope) (playerReferencePlan, error)`,
`compileInstanceValue(Node, scope) (valueFactory, error)`,
`compileResolutionPayment` and `compileDelayedStep`. Supported controller syntax
is `ControllerOfPermanent(permanent-ref)` in a resolving player context.
`PlayerAction(player, action)` and `PlayerMayCost(player, cost)` retain their
existing scoped lowering. The tested value syntax is
`TheNumberOfCardsInPlayersHand(player)`, `DamageDealtToPlayerThisTurn(player)` and
`DamageDealtToPlayerThisTurnByPermanents(player, IsCardtype(Artifact))` in existing
value contexts. This engine contract adds no RON owner, life-total or life-gain
value adapter, and does not expand restricted `PayManaX` amount syntax. Plans
still build fresh references per factory instance and reject invalid scopes.

Evidence: native `TestWaitingDelayedPlayerValueReferences` (324 typed operand /
identity cases), `Boundaries` and `Purity`; RON
`TestRONWaitingDelayedPlayerValueProtection` (98 real-record subrule cases),
`Invalid` (seven rejection cases), and `Instances` (three value families with
independent factories, departure information and cloned gameplay). The cloned
simulation player declines optional payments under its existing decision policy.
The seven protection names gain subrule evidence, with zero full-card unlocks.
Their complete records still reject `ChooseADamageSource`; strict collection
loading and runtime/removal gates are unchanged.

Waiting delayed damage bodies retain direct source declarations in typed player
recipients and value operands. `DealDamageFrom`, `DealDamageEach`,
`DealDamageEachPlayer` and each `DealDamagePackets` packet use the same typed
player/value traversal as waiting payments. Queries preserve exact identity
without reading amounts or selecting recipients. Zero damage and empty sets
keep an explicit declaration. Attached, opaque and filtered-count operands need
separate contracts. This extends `WaitingDelayedPermanentSources`; it does not
provide the complete legal source list for a general source choice.

### Waiting delayed filtered counts

`CountMatchingHand(player, filter)` and `CountGraveyardCards(player, filter)`
retain direct `This` reads through typed controller, owner and LKI-controller
players in `Game.WaitingDelayedPermanentSources`. `CountPermanents(subjects)`
retains those reads in typed controller/owner sets and their filter,
union/intersection and mana-value-limit wrappers. The query does not call
filters, resolve players, select members or calculate values. Zero arithmetic
and empty collections keep explicit declarations. Counts read current membership
at resolution. Saved incarnation, departure information, phasing and clone
lifetimes use the shared delayed services. Direct source operands in singleton,
attachment, combat and history counts now have the contract below. Opaque,
bound and implicit membership do not declare a direct source read.

RON resolution values now share the graveyard-count adapter used by indefinite
toughness:

```ron
TheNumberOfGraveyardCards(And([
    IsCardtype(Creature),
    InAPlayersGraveyard(SinglePlayer(ControllerOfPermanent(ThisPermanent)))
]))
```

The two operands can appear in either order. Supply exactly one supported card
filter and one single-player graveyard declaration. Player scope is validated at
compile time. The adapter builds fresh references for each factory instance.
Existing hand and battlefield count syntax is unchanged. This is a prerequisite
for complete source aggregation, with no complete protection-card unlocks.

### Waiting delayed hand actions

`RevealHandFor(player)`, `DiscardMatchingHand(player, filter)`,
`Discard(player, amount)` and `DiscardAtRandom(player, amount)` retain direct
`This` reads through typed controller, owner and LKI-controller players in
`Game.WaitingDelayedPermanentSources`. Both discard amounts retain the existing
typed value declarations. Queries do not resolve players, evaluate amounts or
filters, reveal hands, choose cards, move cards or consume random results.
Empty hands and nonpositive amounts keep the saved source reference.

RON player actions keep their existing scope:

```ron
PlayerAction(ControllerOfPermanent(ThisPermanent), RevealHand)
PlayerAction(ControllerOfPermanent(ThisPermanent), DiscardCards(IsCardtype(Creature)))
PlayerAction(ControllerOfPermanent(ThisPermanent), DiscardHand)
PlayerAction(ControllerOfPermanent(ThisPermanent), DiscardACard)
PlayerAction(ControllerOfPermanent(ThisPermanent), DiscardACardAtRandom)
DiscardNumberCardsAtRandom(TheNumberOfCardsOfTypeInPlayersHand(
    IsCardtype(Creature), ControllerOfPermanent(ThisPermanent)
))
```

`DiscardNumberCardsAtRandom(value)` now accepts scoped resolution values through
`compileInstanceValue`. Fixed nonnegative literals and announced X keep their
behavior. Negative literals remain declaration errors. A dynamic nonpositive
result moves no cards and consumes no randomness. Each instruction reads its
amount once when it resolves; copies read separately. The engine selects random
cards before movement and sends discard events after the complete batch.
Discard replacements use the shared pipeline. `DiscardNumberCards` retains its
fixed-literal restriction.

Exact source identity, departure information, phasing, copied snapshots and clone
lifetime use the existing delayed services. RON controller reads retain LKI.
Opaque, attached, bound and nested future declarations keep their separate
contracts. This supplies a prerequisite for complete damage-source aggregation;
the seven protection records still reject incomplete general source choices.

## Waiting delayed tap and untap subjects

`TapSubject(PermanentRef, ...TapResultOption) ResolvingEffect` and
`UntapPermanent(PermanentRef) ResolvingEffect` declare a direct `This` subject to
`Game.WaitingDelayedPermanentSources() []DamageSource`. The declaration also
applies to the shared untap effect used by `UntapPermanentWithResult`.
Saved sequences, modes and captured delayed bodies retain the declaration.
Queries do not resolve subjects, change tapped state, emit events, write results
or consume RNG. A subject already in the requested state keeps its reference.
Attached, opaque and bound subjects do not imply a direct source declaration;
bound subjects retain their separate saved-frame contract. A body that creates
a later delay does not supply that future body's references.

RON `TapPermanent(permanent)` and `UntapPermanent(permanent)` resolving actions
use `compileTapAction(Node, scope) (effectFactory, error)` and the existing
`compilePermanentRef` plans. Each factory builds its own typed subject.
Malformed, incompatible and unbound references reject before execution.
The same actions work inside `CreateFutureTrigger(event, ActionList([...]))`.
There is no new RON syntax, payment form or general source aggregation.

The shared source-information service keeps current or exact departure facts.
Departure, return and phasing retain waiting availability; returned cards cannot
replace the saved incarnation. Detection ends one-shot waiting availability.
Execution requires the original present incarnation, including after control
changes or a phase cycle (CR 400.7, 603.7a–e, 609.7a, 701.26a–b, 702.26b–d).
Copies and clones retain separate delayed lifetimes. No tap/untap batch or
replacement behavior changes.

Evidence: `TestWaitingDelayedTapReferences` covers 48 native action, nesting,
mode and lifecycle cases, pure queries, retained facts and independent clones.
`TestWaitingDelayedTapBoundaries` checks attached/bound/future exclusions and
invalid native declarations. `TestRONWaitingDelayedTapProtection` covers 112
real-record subrule cases for all six Circles of Protection and Greater Realm
of Preservation, including payment failure, resolution choice, exact prevention,
shield consumption, stack copies and clones. `TestRONWaitingDelayedTapInvalid`
rejects 16 declarations. `TestRONWaitingDelayedTapInstances` checks two factory
instances, saved target incarnations, later shroud and clone execution for both
actions. Complete protection records still reject general source aggregation;
these scenarios do not establish complete-card gameplay coverage.

## Waiting delayed counter-action references

`AddCountersTo(PermanentRef, CounterType, ResolutionValue) ResolvingEffect`,
`AddCountersUpTo(PermanentRef, CounterType, ResolutionValue, int) ResolvingEffect`,
`RemoveCountersFrom(PermanentRef, CounterType, ResolutionValue) ResolvingEffect`
and `RemoveAllCounters(PermanentRef, CounterType) ResolvingEffect` declare direct
`This` subjects to `Game.WaitingDelayedPermanentSources() []DamageSource`.
The result-producing `RemoveCountersFromWithResult` shares the removal declaration.
The first three also declare typed source reads in their amounts. Existing typed
permanent, player, arithmetic and count declarations apply. `SourceCounters`
now declares its source without reading the count. Zero and negative amounts,
empty counters and full bounded limits keep the reference.

Queries do not resolve subjects, evaluate amounts, offer number choices, place
or remove counters, write results, emit events or consume random results.
Attached, bound and opaque subjects do not imply a direct source read. Saved
binding frames retain their separate contracts. A nested delay producer does
not declare its future body before creation. Sequences, modes and captured
bodies propagate the current body's declaration.

RON `compileCounterAction(Node, scope) (effectFactory, error)` routes resolving
`PutCounters/1`, `PutCountersWithRestriction/2` and `RemoveCounters/1` through
existing typed lowering. Supported singular permanent instructions are
`ACounterOfTypeOnPermanent(counter, permanent)`,
`NumberCountersOfTypeOnPermanent(amount, counter, permanent)`,
`UptoNumberCountersOfTypeOnPermanent(amount, counter, permanent)` with its
matching `CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan` limit,
and `ACounterOfTypeFromPermanent(counter, permanent)` or
`AllCountersOfTypeFromPermanent(counter, permanent)`. Each instruction list
has one member. Scope and binding validation precede fresh factory construction.
This contract applies inside `CreateFutureTrigger(event, ActionList([...]))`.
Player poison placement keeps its existing lowering but gains no declaration
contract in this slice. Collection counter operands remain separate work.

Current and exact departure information use the existing source service.
Departure, return and phasing retain waiting availability. Returned objects
cannot replace the saved incarnation. One-shot detection ends availability
before stack placement (CR 400.7, 603.7a–e, 609.7a, 702.26b–d).
Execution needs the original present subject; phase cycles and control changes
keep that identity. Saved delayed targets are not targeted again after shroud.
Counter placement, replacement limits and removal results reuse existing
counter services (CR 122). Copies and clones keep independent delayed lifetimes.
No cost, replacement ordering or simultaneous batch behavior changes.

Evidence: `TestWaitingDelayedCounterActionReferences` has 96 native action,
nesting, mode and lifecycle cases. `TestWaitingDelayedCounterActionOperands`
checks five typed amount forms independently, attached/bound/future exclusions
and invalid subjects. `TestWaitingDelayedCounterActionZeroAndResult` checks
nonpositive amounts, full limits, exact removed-count results and clone isolation.
`TestRONWaitingDelayedCounterActionProtection` has 224 real-record subrule cases
for the six Circles of Protection and Greater Realm of Preservation: costs,
rollback, resolution choice, retained source prevention, consumed shields,
stack copies and clones. `TestRONWaitingDelayedCounterActionInvalid` rejects
11 declarations. `TestRONWaitingDelayedCounterActionInstances` has six typed
amount/placement scenarios with distinct factories and targets, zero arithmetic,
departure counter information, returned-target exclusion and later shroud.
Complete protection records still reject unfinished general source aggregation;
this evidence does not establish complete-card gameplay coverage.


## Waiting delayed collection count operands

`CountPermanents(PermanentSet) ResolutionValue` declares direct `This` operands
in `SingletonPermanent`, `AttachedAuras`, `BlockersOf`, `BlockedAttackersOf`,
`BlockingOrBlockedBy` and `BlockedByThisTurn`. `AttachmentsOn`, `LowestPower`,
filters, union, intersection and mana-value limits retain nested declarations.
`WithoutPermanent` also declares its direct excluded source. A typed
`PermanentsControlledByPlayers(OpponentsOf(player), filter)` declares source
reads in its player. Queries do not resolve references, run filters, select
members, read values or execute the body. Empty collections retain references.
An attachment subject or bound reference does not imply the direct source.

RON `TheNumberOfPermanentsOnTheBattlefield(collection)` at resolution now uses
`compileContextualCollection`. It accepts `SinglePermanent(permanent-ref)`,
`AttachedToAPermanent(collection)`, `IsBlockingAttacker(permanent-ref)`,
`IsBlockedByDefender(permanent-ref)`, `WasBlockedByDefenderThisTurn(permanent-ref)`,
`Ref_TargetPermanents`, typed controller/owner sets, ordinary filters and nested
nonempty `And([...])` / `Or([...])`. Intersections can repeat controller operands;
unions count each incarnation once. Existing target scopes and per-instance
bindings apply. Counts read membership at resolution, including attachments
created after the delayed ability was installed. Static P/T count restrictions
and singular counter-information reads keep their separate contexts.

This is one prerequisite for full CR 609.7a source aggregation. The compiler
still rejects general `ChooseADamageSource` declarations until all source
categories have contracts.


### Waiting delayed collection action source declarations

Waiting source enumeration traverses explicit source operands in `TapEach`,
`UntapEach`, `ReturnEachToHand`, `DestroyEach`, `DestroyEachWithoutRegeneration`
and permanent recipients of `DealDamageEach`. Nested typed sets use the same
pure declaration traversal as `CountPermanents`; no selection or action occurs.
Empty sets retain source declarations. Existing capture, incarnation, LKI,
phasing and clone services keep waiting references until trigger detection.
Resolution uses current members and the existing simultaneous batches.

RON resolving `EachPermanent(collection)` damage recipients now use contextual
collections: singleton, attachments, combat/history, controller/owner, filters,
and nonempty unions/intersections. Existing bound target/property filters remain
supported. Plural delayed target references retain their rejection boundary.
Counter collections and other collection action declarations remain separate.
See the collection action contract in `docs/engine-rules-contracts.md` for tests.


## Counter collection totals and waiting declarations

`CountCountersAmong(PermanentSet, core.CounterType) ResolutionValue` sums the
specified counters on distinct current, present collection members when used.
Stale incarnations, departed objects, duplicates and phased objects supply no
counters. Counters are converted to int before summation. This is a collection
query, not saved singleton information. `CountersOnWithLKI` retains its existing
singleton contract (CR 122.2, 400.7, 608.2h, 702.26b).

RON `TheTotalNumberOfCountersOfTypeAmongPermanents(counter, collection)` lowers
non-singleton operands through `compileContextualCollection(Node, scope)` with
fresh instance bindings. Supported resolving contexts include typed attachment,
combat/history, controller/owner and nonempty union/intersection collections.
`SinglePermanent(ref)` retains its prior LKI lowering. A singleton inside a
union/intersection is instead a current collection member. Invalid counter
types, shapes, unbound references and empty compositions reject before execution.
Plural delayed targets still reject under their existing capture boundary.

`Game.WaitingDelayedPermanentSources() []DamageSource` traverses explicit source
reads in `CountCountersAmong` collections and native `AddCountersEach` subjects
and amounts. The query does not select members, run filters, read counters,
execute placements or make choices. Empty sets, zero arithmetic and saved modes
retain declarations. Opaque callbacks, implicit matching members and future
bodies not yet installed do not infer references. Existing exact source capture,
departure facts, snapshot isolation, copy/clone lifetime and one-shot detection
remain in use (CR 603.7a–e, 609.7a–b). Counter placement continues through the
existing simultaneous preparation and replacement pipeline.

Evidence: `TestWaitingDelayedCounterCollectionOperands` covers 24 native
subject/amount declarations and exact departure/return identity.
`TestWaitingDelayedCounterCollectionTotal`, `Purity` and
`TestCounterCollectionCurrentMembers` cover pure queries, zero/mode declarations,
duplicate/stale/phased members, clone isolation and invalid native definitions.
`TestRONCounterCollectionProtection` covers 294 real-record subrule cases: seven
protection names, six collection forms and seven lifecycle/copy states, real
payment failure, resolution choice and prevention consumption. RON composition,
invalid and instance tests cover late attachments, totals, saved targets, later
shroud, returned-object exclusion and independent factories/clones. Existing
`TestRONCounterInformation` tests retain real singleton card regressions.
Full protection records still reject general source aggregation. These tests
increase subrule coverage, not complete-card coverage.

### Waiting delayed exile references

`Exile(This)` and `ExileEach(subjects)` declare explicit source reads to
`Game.WaitingDelayedPermanentSources`. Typed collection operands use the same
source-read service as counter and other collection actions. Queries do not
select members or exile objects; an empty collection keeps explicit source reads.
Saved references retain their incarnation after departure, return or phasing.
Execution still excludes absent/phased subjects and uses the existing exile batch.

RON `Exile([Permanent(ref)], [])` and
`Exile([EachPermanent(collection)], [])` use the existing scoped references and
shared contextual collection compiler. Nested union/intersection, attachment,
combat/history and controller/owner operands retain explicit source reads.
Invalid scopes, empty compositions, plural delayed targets and unsupported
modifiers remain compilation errors. See the waiting delayed exile contract in
`docs/engine-rules-contracts.md` for evidence and source-aggregation limits.

## Waiting delayed untap-restriction declarations

Resolving `PermanentDoesntUntapDuringControllersNextUntap(ref)`,
`PermanentDoesntUntapDuringControllersNextNumberUntaps(ref, Integer(n))` and
`EachPermanentDoesntUntapDuringControllersNextUntap(collection)` share
`compileNextUntaps(Node, scope)` and fresh instance bindings. The same forms
can run inside supported delayed bodies. Counts remain nonnegative literals;
saved singular targets keep their exact incarnation. Delayed plural targets
and unbound or incompatible references still reject.

Native `SkipNextUntap`, `SkipNextUntaps` and `SkipNextUntapEach` declare explicit
source reads to `Game.WaitingDelayedPermanentSources`. `SkipNextUntapOf` also
checks its typed player operand. Zero counts and empty collections retain the
source declaration. Queries do not resolve subjects, read membership, install
restrictions or run filters. Departure and phasing retain the saved reference;
a returned card cannot replace it. Resolution still requires a present subject
and reads collection membership then. These declarations do not enable general
`ChooseADamageSource` aggregation.

## RON waiting delayed combat-prevention references

In a `CreateFutureTrigger` body, supported `CreatePreventDamageUntil/3` forms
retain explicit `ThisPermanent` subjects as exact waiting damage-source
references. This includes to-and-by unions and source reads inside blocking
relation collections. Queries do not select members or install prevention.
Global `CombatDamageWouldBeDealt` does not infer a source operand.

Collection unions keep one typed set containing the singleton and relation.
Both operands retain their own instance bindings. At delayed resolution, current
present members receive prevention for the declared duration. An empty relation
still declares its source operand while waiting. A returned object cannot replace
that saved operand. Clones keep separate membership and waiting lifetimes.
Existing duration, operation, scope and union restrictions still apply; general
`ChooseADamageSource` remains unsupported until all categories are complete.
See the waiting delayed combat-prevention contract in
`docs/engine-rules-contracts.md` for signatures and test mappings.

### Waiting delayed temporary layers

`CreatePermanentLayerEffectUntil/3` lowers to one typed `Until` installation.
The waiting source query reads explicit source declarations in its subject,
collection, values and duration without executing the effect. Saved incarnations
remain available after departure or phasing. Granted ability bodies keep their
own future recipient scope. Supported syntax and duration restrictions remain
unchanged; see the temporary-layer declaration contract in
[engine-rules-contracts.md](engine-rules-contracts.md).

### Waiting delayed player collections

`EachPlayerAction(AnyPlayer, DrawNumberCards(value))` accepts scoped resolution
values through `compileInstanceValue`, including permanent power, arithmetic and
bound values. Negative integer literals remain invalid. The draw amount is read
once at execution, before either player draws. `ActionPlayer` is not introduced
by this form. The adjacent shuffle-result form keeps its separate result scope.

Typed native player sets declare explicit source reads to
`Game.WaitingDelayedPermanentSources`. Opponent sets, damage-history sets and
singular library-recycling sets retain their typed operands. Draw, discard, life,
library recycling, random choice, damage and player-controlled permanent
collections use these declarations. Queries do not execute the operation or
select members. General damage-source aggregation remains unsupported.

## Waiting delayed conditional operands and scoped coin branches

`Game.WaitingDelayedPermanentSources() []DamageSource` retains explicit source
reads through typed card-type, cast, enchantment-subtype, positive-value, state,
activation-history and coin branches. State conditions declare typed source
subjects, counter/combat/history reads and nested all/any/not operands.
`UnlessPermanentCameUnderControlThisTurn` also declares its player operand.
`SnapshotPlayer` declares source reads in the binding producer. Waiting delayed
intervening-if conditions use the same typed traversal.

Both branches declare references, even when a condition is false or a branch
cannot run. Queries do not check conditions, read amounts, flip coins or run
callbacks. Opaque callbacks, implicit attachment reads and future nested delays
remain excluded. The existing saved incarnation, LKI, phasing, deduplication,
trigger detection and clone services retain their behavior.

RON `compileCoinBranch(Node, scope) (effectFactory, error)` lowers the current
player scope with `mage.FlipCoin(PlayerRef, ...CoinBranch) ResolvingEffect`.
Both `FlipACoin_OnLose([actions])` and
`FlipACoin_OnWinAndLose([win_actions], [loss_actions])` now support `PlayerAction`
and `PlayerActions` contexts, including delayed bodies. Branch lists must be
nonempty. `PlayerActions` supplies its existing local `ActionPlayer` binding;
`PlayerAction` supplies no new binding. Nested actions retain the same lexical
player scope. Unbound or incompatible references remain compile errors.

Evidence: `TestWaitingDelayedBranchOperands`, `TestWaitingDelayedBranchPurity`,
`TestWaitingDelayedBranchIntervening` and `TestRONWaitingDelayedBranch*`.
The seven protection names gain subrule evidence only. General source aggregation
still rejects their complete RON records.

### Waiting resolution choices and scoped action choices

RON `ChooseAnAction([[PutCounters([ACounterOfTypeOnPermanent(PTCounter(p,t),
Ref_TargetPermanent)])], ...])` remains supported by the general nonempty
action-list body contract and requires at least two alternatives. It works inside
`PlayerAction`
and `PlayerActions`, including already-created delayed trigger bodies. The
lexical player chooses at resolution; `ActionPlayer` is declared only by
`PlayerActions`. Targets remain typed, bound and captured by the existing delay.

`ChooseAnAbility([ability, ...])` with its adjacent
`CreatePermanentLayerEffectUntil(..., [AddAbilityVariable(TheChosenAbility)],
...)` continuation retains its existing supported abilities and contexts.
Both choice adapters build fresh options and use the lexical player plan.
The shared helper for color and basic-land choices keeps its current callers.

Mage declares source reads in every option body and condition, plus the chooser,
for waiting delayed source enumeration. The query does not make a choice or
check option availability. Unselected options retain explicit source references.
General `ChooseAnAction` bodies use the shared sequence compiler.
General `ChooseADamageSource` retains its separate supported contexts.

### Created resolution values

RON `CreateValueX(value)` saves an existing typed resolution value at its place
in an action sequence. Subsequent `ValueX` reads use a private number binding in
that sequence and its nested action bodies. A second producer can read the first
saved value in its operand, then supplies a fresh binding for later instructions.
A branch or future delay does not export a producer to its enclosing sequence.
An announced X is available before a producer. Saved X takes precedence after it.
Unbound reads and malformed producers are compile errors. Contexts that require
announcement values or legacy value sources reject saved-number reads.

The compiler lowers the producer to
`mage.SnapshotNumber(ResolutionValue, *NumberBinding) ResolvingEffect`.
Dynamic resolution `PayManaX([ManaCostX], ValueX)` uses this binding for its amount.
Bindings are built per factory instance and values stay in the engine resolution
frame. Saved values pass through existing delayed capture and copy services.
Primordial Ooze uses the producer after its counter instruction and before its
optional payment. Its unpaid damage reads the same saved counter total.

A waiting delayed snapshot declares explicit source reads in its operand.
Queries do not evaluate the value or write the binding. Zero arithmetic keeps
explicit references. An already-saved number does not infer an object reference.
Future delays and implicit attachment reads keep their separate source contracts.


### Waiting delayed prevention operands

`PreventNextDamage(recipient, amount)` declares explicit source reads in its
recipient and typed resolution amount to `Game.WaitingDelayedPermanentSources`.
`PreventDamageThisTurn(recipient, from)` declares recipient reads.
`PreventNextDamageFromPermanent(subject, player, filter)` declares source and
player reads. `PreventNextDamageFromExactSource(source, player, filter)` and
`PreventNextDamageFromSelectedSource(source, player, filter)` declare their typed
source and player reads. These declarations use the existing source traversal;
implicit attachments, opaque values and future delays do not infer references.

RON `CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(
amount, recipient), [PreventThatDamage])` uses `compileInstanceValue` for its
amount. Fixed integers, announced X, created `ValueX` and existing typed
resolution values retain their normal scope rules. Literal negative amounts
remain declaration errors; computed nonpositive amounts install no effective
shield. The amount is read when the shield is installed. Subsequent changes
cannot change that installed amount. The existing recipient compiler keeps
permanent, player and any-target references separate. General damage-source
choice remains unsupported.

### Source operands of waiting redirection installations

`RedirectNextDamageFromSelectedSource(DamageSourceRef, PermanentRef, PlayerRef,
DamageSourceFilter) ResolvingEffect` and
`RedirectDamageFromPermanentThisTurn(PermanentRef, PlayerRef, PermanentRef)
ResolvingEffect` declare their explicit source reads to
`Game.WaitingDelayedPermanentSources() []DamageSource`. The source, subject or
destination, and typed source-derived player all participate. `This` and
`DamageSourceOf(This)` retain the delayed source's exact incarnation. Implicit
attachments and opaque references do not declare it.

Existing RON lowering through `compileSelectedSourceRedirect` and
`compileTemporaryDamageRedirect` uses these declarations in a waiting body.
`CreateReplaceWouldDealDamageUntil` retains its permanent-to-player event, one
`DealToCreatureOrPlaneswalkerInstead` operation and `UntilEndOfTurn` duration.
`CreateFutureReplaceWouldDealDamage` retains its bound chosen-source event,
permanent recipient and one `DealToPlayerInstead` operation. Typed targets,
source-derived players and per-instance source bindings keep their existing
scope checks. A source-choice producer in a delayed body must precede the read
in that body; this slice adds no cross-body binding permission.

Queries do not resolve operands, select damage sources or install redirection.
Departure and phasing keep the saved reference; returned objects cannot replace
it. Detection ends one-shot waiting availability before stack placement. Native
and RON tests cover lifecycle, copying, cloning, payment and declaration errors.
The seven protection records gain waiting-redirection subrule coverage. General
`ChooseADamageSource` still requires the remaining category audit and aggregation.


## Source operands of waiting damage-source choices

`ChooseDamageSourceFrom(PlayerRef, []DamageSourceRef, *DamageSourceBinding)
ResolvingEffect` and `ChooseDamageSourceMatchingFrom(PlayerRef,
[]DamageSourceRef, DamageSourceFilter, *DamageSourceBinding) ResolvingEffect`
declare explicit permanent source reads in their player and fixed source list
through `damageSourceChoice.refersToSourcePermanent() bool`.
Category choices use the same typed player declaration. The query
`Game.WaitingDelayedPermanentSources() []DamageSource` retains the saved exact
incarnation when the player reads its controller or owner, or the list contains
`This` or `DamageSourceOf(This)`. Lists with duplicate operands still supply
one reference. Filters that cannot match do not remove declarations.

Queries do not resolve references or players, apply filters, select candidates,
write choice bindings or install damage rules. Implicit attachments, opaque
references, category enumeration alone and future delayed bodies do not infer
source reads. Departure and phasing retain the saved incarnation and current or
exact departure facts; a return cannot replace it. One-shot detection ends
waiting availability before stack placement. These boundaries follow CR 400.7,
603.7a–e, 609.7a–b and 702.26b.

Execution reuses source choice, exact-object deduplication, per-instance binding
frames, clone isolation and the shared damage/replacement services. RON reuses
`compileChosenDamageSourceChoice`, `compileSelfDamageSource`, `compilePlayerRef`,
`compileActions`, `compileExactSourcePrevention` and the existing category
lowering. No new RON spelling or general source aggregation is enabled.
`ChooseADamageSource/1` continues to reject in production.

`TestWaitingDelayedDamageChoiceOperands`, `PurityAndBoundaries` and `Invalid`
establish lifecycle declarations, snapshot isolation, query purity, implicit and
opaque boundaries, clone lifetime and definition validation.
`TestRONWaitingDelayedDamageChoiceOperands`, `Protection` and `Invalid` establish
RON typed operand and consuming-operation lowering, copies, failed payment,
resolution timing, exact identity, independent consumption and rejection of
invalid declarations. Jade Monolith supplies the target/redirection body; six
Circles and Greater Realm supply actual costs, filters and prevention bodies.
Their explicit-list/category test seams establish subrules, not complete-card
gameplay or complete CR 609.7a aggregation. Existing choice and redirection tests
continue to establish simultaneous packets, replacement ordering and legality.

## Token quality abilities

`TokenWithPT` and `NamedTokenWithPT` accept `BandsWithOthers(quality)` in their
ability lists. The quality uses the existing source-independent static permanent
filters, including `And`, `Or`, type, color and `IsNamed(NamedCard(name))`.
Empty filters, malformed constructors, target references and source-dependent
controller/exclusion filters remain errors. Multiple quality abilities remain
independent; ordinary banding is not granted.

The compiler creates a fresh static ability for each declaration and uses
`GrantBandsWithOther(This, quality)`. Token construction copies each ability's
continuous effects. A spell producer cannot supply its source or bindings to
the token ability. Creation, resolution controllers, simultaneous entry, layer
application, removal, phasing and combat reuse the native token and layer services.

`TestRONTokenBandsMaster`, `Combat` and `Lifecycle` use the complete Master of
the Hunt record. `SpellBatch` covers two independent qualities on a batch created
by a spell; `Invalid` checks declaration rejection. Native
`TestTokenBandsSingularGrant` establishes singular subjects, batch isolation,
multiple qualities, phasing, removal, cloning and definition validation.

## Waiting delayed library operands

Library inspection declares explicit source reads in the viewer, library player
and typed count. Optional shuffle declares them in the chooser and library player.
Waiting queries do not read values, inspect cards, order cards or offer choices.
Zero counts keep source references. Saved incarnations survive departure and
phasing until trigger detection; returned objects cannot replace them.
Implicit attachments, opaque values and future delayed bodies keep separate scopes.

`LookAtLibraryTopNumber(viewer, library PlayerRef, count ResolutionValue,
reorder bool)` reads the count once at resolution. Missing players perform no
inspection. Nil counts and negative direct Fixed counts fail definition validation.
`LookAtLibraryTop` preserves its integer API through this service.
Evidence: `TestRONWaitingDelayedLibrary*` and `TestWaitingDelayedLibrary*`.

## Waiting delayed mana-loss player operands

Bare `LoseUnspentMana` lowers to `LoseUnspentManaFor(player PlayerRef)` in the
current resolving player scope. `PlayerAction(player, LoseUnspentMana)` keeps
its typed player operand; `PlayerActions(player, [LoseUnspentMana])` uses its
existing player snapshot and local binding. Both work in delayed bodies.
Tuple spelling, unbound targets, wrong reference roles and out-of-scope event
or action players remain errors. Each factory keeps separate bindings.

`LoseUnspentManaFor` declares explicit source reads from its typed player to
`Game.WaitingDelayedPermanentSources`. Queries do not resolve players or clear
mana. Empty pools keep the declaration. Saved incarnations survive departure,
return and phasing until trigger detection. Bound player values, implicit
attachments and future delayed bodies do not infer a source reference.
Resolution uses the existing player reference and mana-pool services, including
restricted mana removal without life loss. General source choice still requires
the complete legal source aggregation.

Evidence: `TestWaitingDelayedManaOperands`, `Boundaries`,
`TestRONWaitingDelayedManaGameplay`, `Instances`, `Invalid` and `Protection`.

## Waiting delayed poison operands

`AddPoisonFor(subject PlayerRef, amount ResolutionValue)` declares explicit
source reads from both operands to `Game.WaitingDelayedPermanentSources`.
`ControllerOf(This)`, `OwnerOf(This)` and `ControllerOfWithLKI(This)` qualify.
Typed source values and their arithmetic retain the declaration. Queries do not
resolve players, read amounts or add counters. Zero amounts keep the reference.
Sequences and modes retain it; implicit attachments, opaque operands and future
delay bodies do not infer a source read.

Departure and phasing keep the exact saved incarnation until trigger detection.
A returned object cannot replace it. Execution retains the existing player and
value semantics: live-only references require presence; typed information reads
can use phase or departure facts. Copies and game clones keep separate frames
and waiting lifetimes (CR 400.7, 603.7a–e, 609.7a, 702.26b–d, 122.1f).

RON `compilePlayerCounterPlacement(Node, scope) (effectFactory, error)` lowers
one `ACounterOfTypeOnPlayer(PoisonCounter, player)/2` or
`NumberCountersOfTypeOnPlayer(Integer(n), PoisonCounter, player)/3` instruction
in `PutCounters/1`. The count is a nonnegative literal. Typed player declarations
and their lexical scope are checked before fresh factory construction. This
includes supported delayed resolving bodies. Dynamic RON poison counts and
other player-counter types remain unsupported.

Evidence: `TestWaitingDelayedPoisonOperands`, `Boundaries` and `Resolution`;
`TestRONWaitingDelayedPoisonGameplay`, `Invalid`, `Instances` and `Protection`.
The seven protection consumers cover this source category only. General
`ChooseADamageSource` remains rejected until complete source aggregation has
contract evidence. Existing `TestRONPoison*` retains complete Pit Scorpion,
Marsh Viper and Serpent Generator coverage.

## Waiting delayed mana production operands

`AddManaAmount(color Color, amount ResolutionValue)`,
`AddManaOfAnyOneColor(amount ResolutionValue)` and
`AddRestrictedManaOfAnyOneColor(amount ResolutionValue, restriction ManaRestriction)`
declare explicit source reads in their amounts.
`AddManaToPlayer(player PlayerRef, color Color, amount ResolutionValue)` and
`AddManaOfProducedTypeToPlayer(player PlayerRef, types *ManaTypesBinding, amount ResolutionValue)`
also declare player reads to `Game.WaitingDelayedPermanentSources()`.
The shared typed source-read services inspect operands without resolving players,
reading amounts, choosing colors, adding mana or testing spending restrictions.
Zero amounts retain explicit reads. Sequences and modes preserve declarations;
implicit attachments, opaque operands and future delayed bodies do not infer them.

Saved exact incarnations survive departure, transient LKI clearing and phasing
until detection. Returned objects cannot replace saved references. Execution
retains current and departure information rules and the saved resolving controller.
A delayed frame now captures produced mana-type bindings with separate slices.
Step, modal and permanent-event delays restore those bindings only for their body.
Independent events, stack copies and game clones retain separate frames and pools.
Existing payment, mana-event, source-filter and prevention services are reused
(CR 400.7, 603.7a–e, 609.7a, 702.26b–d).

RON `compileResolvingMana(Node, scope) ([]effectFactory, error)` isolates existing
`AddMana/1` and `AddManaRepeated/2` lowering. Existing literal and paid-result
restrictions remain. `AddManaRepeatedWithModifiers(AnyManaColor, value,
CanOnlySpendOnSpells(IsCardtype(Creature)))/3` uses typed instance values,
including source power, source counters and arithmetic in delayed bodies.
No new production syntax or broader explicit-player mana context is introduced.
Land-mana event lowering keeps its existing immediate solver contract.

Evidence: native `TestWaitingDelayedManaProductionOperands`, `Boundaries`,
`Resolution`, `EventTypes` and `CapturedTypes`; RON
`TestRONWaitingDelayedManaProductionGameplay`, `Invalid`, `Instances` and
`Protection`. Seven complete protection records supply their actual source filters,
costs and one-use shields in category-specific tests. This is subrule evidence;
general `ChooseADamageSource` still rejects full records until complete aggregation.

## Waiting delayed regeneration subjects

`Regenerate(subject PermanentRef) ResolvingEffect` declares an explicit `This`
subject to `Game.WaitingDelayedPermanentSources()`. The query uses the shared
source-read declaration and saved delayed incarnation. It does not resolve a
subject, create a shield, consume a shield or change combat. Sequences and modes
preserve the declaration. Departure, cleared transient LKI and phasing keep the
reference until trigger detection. A returned object cannot replace it.
Implicit attachments, opaque references and nested future bodies do not infer
source reads. Target and saved bindings use the existing captured frame service.

At resolution the shared regeneration service protects only a present,
phased-in subject with the saved identity. Current control does not change that
identity. Copies create separate shields; clones keep separate frames and shield
lifetimes. Destruction consumes one shield, removes damage, taps the subject and
removes it from combat. Later shroud does not invalidate an already saved delayed
subject. A departed or returned target receives no shield. Implicit Aura hosts
use the attachment at body resolution (CR 400.7, 603.7a–e, 609.7a, 701.19a–c,
702.26b–d). No replacement ordering or simultaneous-operation contract changes.

RON `compileRegeneration(Node, scope) (effectFactory, error)` lowers
`RegeneratePermanent(permanent)/1` through `compilePermanentRef`. Existing source,
Aura host, target and event scopes remain unchanged, including supported delayed
resolving bodies. Wrong arity, wrong roles and unbound references reject before
factory publication. Fresh factories retain distinct target bindings.

Evidence: native `TestWaitingDelayedRegenerationOperands`, `Boundaries` and
`ModesAndPurity`; RON `TestRONWaitingDelayedRegenerationGameplay`, `Instances`,
`Invalid`, `Host`, `Records` and `Protection`. Seven complete protection records
supply actual costs, source filters and one-use prevention in category-specific
tests. Full general source choice remains rejected. Unchanged Drudge Skeletons,
Living Wall, Uthden Troll, Wall of Bone, Wall of Brambles, Will-o'-the-Wisp and
Regeneration records retain immediate gameplay evidence. This slice adds no
new complete-card compilation claim.

### RON Fight operands

`Fight(first, second)/2` lowers resolving actions to
`mage.Fight(first, second PermanentRef) ResolvingEffect`. Both operands use the
existing permanent scope: explicit source, implicit Aura host, declared singular
or numbered targets, and supported event bindings. Each card factory creates
its own bindings. Wrong-role, missing and out-of-scope operands reject compilation.

Both present creatures deal current power as simultaneous noncombat damage.
If either operand is absent, phased out, no longer a creature, a returned object,
or an illegal resolving target, neither fights. Self-fight deals twice power.
Earlier independent instructions still resolve for legal targets. Saved delayed
operands preserve identity and do not recheck shroud as new targets.

A direct source in either operand declares that source to waiting delayed damage
source queries. Queries do not resolve operands or deal damage. Implicit hosts,
opaque references and future delayed bodies supply no inferred source read.
Arena now compiles and has real-record gameplay tests; general damage-source
aggregation remains rejected.

### RON graveyard entry controller

`PutGraveyardCardOntoBattlefield(card, [EntersUnderPlayersControl(player)])`
lowers through `compileGraveyardReturn(Node, scope) (effectFactory, error)`
to `mage.ReturnCardToBattlefield(card CardRef, controller PlayerRef,
options ...BattlefieldEntryOption) ResolvingEffect`. The card must be an existing
singular graveyard target whose filter requires a creature. One bare
`EntersNormally` still uses `You`. One `EntersUnderPlayersControl(player)/1`
uses the existing typed player scope and fresh factory bindings. Multiple entry
operations, invalid roles and unbound players reject at compilation. This adds
no graveyard-Aura attachment, source-zone card or action-result binding.

The controller is read when the return resolves. `ControllerOfPermanent` uses
current or retained exact-incarnation information; `You` uses the resolving
ability's controller. An implicit `HostController` reads the current attachment.
The exact graveyard visit must still be present. Normal entry replacements run
before entry events; entry triggers resolve separately.

A direct source read in the controller supplies the saved source to
`Game.WaitingDelayedPermanentSources` until trigger detection. Queries do not
resolve players or move cards. Departure, phasing, return exclusion, sequences,
modes and clone isolation use existing engine services. Implicit attachments,
opaque players and future delayed bodies do not infer source reads.

Evidence: `TestRONWaitingDelayedReturnGameplay`, `Instances`, `Host`, `Invalid`,
`Inventory` and `Protection`; native `TestWaitingDelayedReturnController` and
`Boundaries`. Protection evidence uses bounded category choices with real-record
costs, filters and shields. General `ChooseADamageSource` remains rejected.

### Source reads in delayed library-choice amounts

`ScryAtResolution(amount ResolutionValue)` and
`SurveilAtResolution(amount ResolutionValue)` declare explicit source reads in
`amount` to `Game.WaitingDelayedPermanentSources`. Nested typed operands retain
these reads even when the resulting amount is zero. Enumeration does not resolve
values, inspect libraries, choose placement, move cards or emit action events.
Sequences and delayed modes retain the declaration. Implicit attachments,
opaque values and future delayed bodies do not infer a source read.

The waiting frame retains the exact source through departure, cleared transient
LKI and phasing. A returned object cannot replace it. One-shot trigger detection
ends waiting availability before body resolution; game clones have separate
waiting lifetimes. The body reads the amount once in its saved frame and uses
the delayed ability's controller with the existing scry/surveil services.
Current source control does not change that player. Nonpositive amounts make
no placement choice. Card inspection remains private.

Rules: local CR 603.7a–e, 609.7a, 701.22, 701.25 and 702.26b–d.
Evidence: `TestWaitingDelayedLibraryChoice*` and
`TestRONWaitingDelayedLibraryChoice*`. The RON tests use existing typed value,
cost, chosen-source filter and shield lowering with a native delayed library
body. There is no scry/surveil spelling in the collection, and this contract
adds none. Seven protection subrules gain bounded source-choice evidence;
general source aggregation and complete-card coverage remain separate gates.


## General damage-source choice

`mage.ChooseDamageSource(player PlayerRef, filter DamageSourceFilter,
binding *DamageSourceBinding) ResolvingEffect` chooses from the CR 609.7a
categories at resolution. `(*Game).DamageSourcesFor(playerID uuid.UUID)
[]DamageSource` reads the same categories without a choice. It combines live
permanents and spells, exact references on stack and resolving objects, waiting
replacement/prevention and delayed references, and face-up command objects.
Graveyard activated ability sources supply their exact saved visit.
An object need not be able to deal damage. Exact identities remove duplicates.
Hand references use the viewer's inspection rights; inspection adds no source.
Other category-specific choice APIs retain their bounded behavior.

RON `ChooseADamageSource(filter)` in an `ActionList` declares
`TheChosenDamageSource` for its following actions. Each factory call allocates
its own binding. A later choice has its own binding and shadows the prior choice
for its continuation. A branch-local choice does not declare a result outside
that branch. Filters support `AnyDamageSource`, `IsColor(color)`,
`IsCardtype(type)`, and nonempty `And([filters])` / `Or([filters])` lists.

The six Circles of Protection and Greater Realm of Preservation lower their
next-source shield to `PreventNextDamageFromSelectedSource`. Jade Monolith uses
`RedirectNextDamageFromSelectedSource` with its exact creature target. The
existing engine services own property rechecks, simultaneous damage,
one-use consumption, cleanup, and permanent-spell continuation.
Replacement bodies for Reverse Damage, Dark Sphere and Eye for an Eye still
reject. Earlier bounded-source prerequisite sections describe their original
scope; this section establishes the general aggregation contract.

## Player choice at resolution

`mage.ChoosePlayer(player PlayerRef, subjects PlayerSet, binding *PlayerBinding)
ResolvingEffect` records one required choice in the current resolution frame.
It uses the existing player decision interface with bounds of one. A sole
candidate needs no decision; an empty set or absent chooser clears the result.
An invalid, empty or multiple-member answer uses the first candidate. The
candidate list passed to the decision is separate from the validated list.
`mage.PlayersWhoCastSorceryThisTurn() PlayerSet` reads the existing turn tracker
in active-player-first order. Countered and unresolved casts qualify. Failed
casting payments and spell copies do not count. Final cleanup clears history.

RON `ChooseAPlayer/1` in a resolution `ActionList` supports bare `AnyPlayer`,
bare `Opponent`, and `CastASpellThisTurn(IsCardtype(Sorcery))`. The chooser is the
current explicit player scope. Opponent is relative to that same player.
Bare `TheChosenPlayer` is available only in the choice's continuation. Each
producer has a separate typed binding, and a later producer shadows the earlier
one in its continuation. Factories allocate fresh declarations. Delayed bodies
capture existing bindings or perform their own choices at delayed resolution.
Spell and activation sequences use the same lowering. A terminal
`PlayerAction(player, ChooseAPlayer(filter))` makes that player's choice without
exporting its result. Entry choices retain their separate permanent lifetime.
Other spell-history filters and player-choice contexts remain unsupported.

Evidence: native `TestPlayerChoice*` and RON `TestRONPlayerChoice*`. Tests execute
the exact choice nodes from Backdraft and Rainbow Vale with an observation body;
they do not claim complete gameplay for either card. Their full records still
reject at resolved-spell history and explicit-player control transfer.

## Retained printed-definition builders

Every successful RON `Compile(Node) (mage.CardFactory, error)` now retains a
`mage.CardDefinitionPlan` with the lowered immutable construction plan. Compilation
validates two independent builds before publishing the factory. The same option
and instance-binding factories build reconstructed abilities, costs, target
groups, entry declarations, static rules and delayed bodies. No new RON spelling
or binding scope is introduced. Invalid engine declarations return positioned
compiler errors. Unsupported copy expressions still reject.

`mage.RebuildCardDefinition(card mage.Card) (mage.Card, error)` creates a fresh
printed definition without name lookup or a Go registry. `BaseCard.Copy` retains
that plan; legacy `CloneFrom` clears it when it replaces rules. Reconstruction
returns a fresh card identity and declarations with no owner, ability source,
controller, target choices or saved resolution state. Printed metadata, keyword
seeds, additional casting costs and Aura target declarations come from the plan.
The caller must supply the new instance's owner and lifecycle context.

This is a prerequisite for copiable declarations. It does not capture current
copiable values, install a copy layer, transfer announcement choices, or schedule
entry replacements. Clone, Copy Artifact and Vesuvan Doppelganger remain
unsupported until those contracts are implemented.

Evidence: `TestCardDefinitionPlan*` and `TestRONDefinitionPlan*`. The RON tests
rebuild every currently accepted record and execute reconstructed Icatian
Javelineers, Swords to Plowshares and Holy Strength. Birds of Paradise and Wall
of Wood cover printed metadata and keyword seeds. Synthetic player choices
cover independent delayed bindings and clones.

## RON copiable provenance and saved prototypes

Each successful RON `Compile(Node) (mage.CardFactory, error)` now lowers printed
mana-cost presence and retains immutable `mage.CopiableValues` beside its
`mage.CardDefinitionPlan`. `ManaCost` absent is separate from `ManaCost: []` or an
explicit zero cost. Metadata comes from the validated definition; live source
choices are never retained in that provenance. The compiler captures field
presence at lowering, so later mutation of the input AST cannot change it.
No new RON spelling or lexical scope is added in this prerequisite slice.

`game.CaptureCopiableValues(source)` returns saved values for one present exact
incarnation. `values.Build()` reconstructs independent face-up prototypes with
fresh rule declarations. Ordinary layers, counters, controller, ownership,
attachments, paid costs and target/entry choices do not transfer. A prototype
built from saved values supplies the same provenance to another capture.
Standard face-down sources supply a nameless colorless 2/2 creature with no mana
cost or abilities; the prototype is face up. Values remain fixed after the source
changes, phases or leaves. Capture itself does not run the construction builder.

Missing provenance, untracked changes, inactive subjects and active legacy copy
layers return errors. Nil or uninitialized snapshots cannot build. Mutable card
cost/color data is separate in ordinary copies. Rules added after construction
and legacy `CloneFrom` clear copy provenance. The existing builder contract still
requires pure construction and immutable lowered declarations.

This service does not grant casting/entry permission, schedule entry replacements,
install live copy layers or implement `CopyEffects` exceptions. Those are the
remaining contracts for Clone, Copy Artifact and Vesuvan Doppelganger. These
complete records remain rejected rather than running approximate copy behavior.

Evidence: `TestCopiableValues*`, `TestRONCopiableValues*`, and the retained
`TestRONDefinitionPlan*` reconstruction/accepted-collection regressions.


### Entry-copy physical identity prerequisite

Every accepted RON factory retains a printed definition plan separately from its
copiable effective plan. `mage.BuildEntryCopy(card, values)` uses those plans to
build independent entry rules and printed departure rules with the same physical
identity. It changes no RON spelling or binding scope. The supplied snapshot can
come from a different permanent or another copy; the physical origin remains
that of `card`. Normal entry, LKI, zone moves and cloning use the engine services.

This builder grants no entry permission and does not schedule replacements.
`EnterAsACopyOfAPermanent` and `CopyEffects` still reject until their complete
entry-copy and exception contracts are available. No RON copy declaration is
accepted through a partial adapter.

### RON token copiable-value prerequisite

`mage.NewTokenDefinitionFactory(build mage.TokenDefinitionFactory)
(mage.TokenDefinitionFactory, error)` checks a pure builder of fresh token
characteristics, abilities and bindings. The returned factory supplies independent
`TokenDefinition` declarations with one immutable `CardDefinitionPlan`.
RON lowering uses it for the existing supported `TokenWithPT/6` and
`NamedTokenWithPT/7` forms in `CreateTokens/2`, including `NumberTokens/2`.
No syntax, binding scope or supported token characteristic is added.

`CreateTokenCopies`, `CreateTokenCopiesInto` and `CreateOneToken` build fresh cards
from this retained plan before the shared entry batch. Build failures return
errors without installing a partial batch; result bindings clear first. Existing
`DefineToken` callers retain their old path. The public card-plan constructor
keeps its signature; its internal builder can now return an error.

Created tokens supply exact snapshots through `CaptureCopiableValues`.
Snapshots retain token name, types, subtypes, colors, printed P/T, keywords and
fresh rules, with an absent mana cost. Saved values omit token status, owner,
counters, controller and choices. `values.Build()` produces a fresh nontoken
prototype; `BuildEntryCopy` preserves the physical copying object's token status.
Phasing or departure prevents a new capture but does not invalidate saved values.
This prerequisite does not accept Clone or add entry-copy scheduling.

Evidence: `TestTokenDefinitionPlan*`, `TestRONTokenCopiableValuesGameplay`,
`TestRONTokenCopiableValuesDeclarations`, and existing token creation/banding,
copy snapshot and entry-copy identity suites.

## Optional early entry copy

Use `mage.MayCopyAsEnters(filter mage.PermanentFilter) mage.Ability` with
`WithAbility` on a permanent whose physical definition has a validated plan.
The entry controller can choose a present matching permanent without targeting.
The copy uses `CaptureCopiableValues` and `BuildEntryCopy`, so the selected
source must have supported copy provenance or standard face-down values.

RON supports this complete entry shape:

```ron
AsPermanentEnters(ThisPermanent, [MayActions([
    EnterAsACopyOfAPermanent(IsCardtype(Creature), NoCopyEffects)
])])
```

The filter uses the existing context-free permanent-filter compiler. The copy
operation must be the sole child of the outer `MayActions`. Other copy exceptions,
required copying, nested optional copying and mixed copy/ordinary entry sequences
remain declaration errors. Ordinary optional entry sequences keep their existing
support. No resolving action or target binding is introduced.

The copy precedes tapped entry, counters and entry choices. Those operations and
ETB observers read the new definition. Chosen values, counters and ordinary layer
changes on the source do not transfer. A physical card keeps its ID and owner,
and uses its printed rules after departure. Phasing keeps the copy.

## Copiable type additions

`mage.CopyAddCardTypes(types ...core.CardType) mage.CopyModification` adds
permanent types within the copy process. It keeps existing types and abilities.
`(*mage.CopiableValues).WithModifications(...mage.CopyModification)
(*mage.CopiableValues, error)` creates an immutable, validated snapshot.
`mage.MayCopyAsEntersWithModifications(mage.PermanentFilter,
...mage.CopyModification) mage.Ability` uses these modifications before later
entry replacements, counters and choices. `MayCopyAsEnters` remains compatible.

RON accepts a sole optional entry-copy operation with this modification shape:

```ron
AsPermanentEnters(ThisPermanent, [MayActions([
    EnterAsACopyOfAPermanent(IsCardtype(Artifact),
        CopyEffects([AddCardtypes([Enchantment])]))
])])
```

`CopyEffects` and each `AddCardtypes` list must be nonempty. Multiple additive
operations run in order; duplicate types are retained only once. The RON type
vocabulary supports Creature, Artifact, Enchantment and Land in this context.
The native API also supports Planeswalker. Instant, Sorcery, unknown types,
malformed declarations and other modification operations reject before gameplay.
`NoCopyEffects` retains its existing symbol shape. Target, resolving, attachment,
required-copy and mixed entry-operation contexts do not acquire support.

Modified snapshots build independent abilities, targets and costs. Subsequent
copies inherit the added types; ordinary layers and counters do not transfer.
Type additions keep the original declaration identities for replacement ordering.
Physical departure rules and phasing use the existing entry-copy services.
Added abilities and live copying remain separate capabilities.

## Retained colors in entry copies

`mage.CopyRetainColors(colors ...core.Color) mage.CopyModification` retains a
saved color set in a copy snapshot. The constructor copies and deduplicates its
input. An empty set or `core.Colorless` represents colorless; `AnyColor` and
invalid values reject during definition validation. Copied mana cost remains.
`mage.MayCopyAsEntersKeepingColors(mage.PermanentFilter,
...mage.CopyModification) mage.Ability` captures the entering object's copiable colors
when the copy replacement applies, before replacing its definition. Other copy
modifications use the existing immutable snapshot service.

RON accepts the zero-argument **symbol** `KeepColor` in the existing sole optional
entry-copy context:

```ron
AsPermanentEnters(ThisPermanent, [MayActions([
    EnterAsACopyOfAPermanent(IsCardtype(Creature), CopyEffects([KeepColor]))
])])
```

`KeepColor` can appear with `AddCardtypes`; repeated occurrences are idempotent.
Tuple forms such as `KeepColor()` reject. Unsupported operations in the same
list still reject the whole definition. Entry context, choice, target exclusion,
filter legality and physical departure rules retain their existing contracts.
This adapter does not enable resolving or live copy-layer contexts.

Mage replaces the donor's color indicator with the saved colors. A colorless
exception remains distinct from absent color metadata and therefore overrides
the donor's colored mana cost. A static continuous effect that defines color
must implement `mage.ColorCharacteristicDefinition` with `DefinesColor() bool`.
A true result omits its entire static ability from the copy (CR 707.9d).
Ordinary color changes must not use this marker. P/T definitions and ordinary
color-changing abilities remain. This is a copy classification contract, not a
new color-CDA authoring or all-zone evaluation service.

Later copies inherit the saved colors and omitted abilities. Ordinary color
layers remain effective and stay outside the snapshot. Existing native and RON
callers remain compatible. Vesuvan Doppelganger advances to its unsupported
`AddAbility` exception; its complete definition is still rejected.

## Next combat event with an unprevented amount

`mage.PreventNextCombatDamageFromPermanentExcept(subject PermanentRef,
recipient PlayerRef, amount ResolutionValue) ResolvingEffect` saves an exact
creature, player and nonnegative amount at resolution. It prevents damage above
that amount in the next matching combat event this turn. Matching simultaneous
packets share one allowance. If no damage is prevented, the shield remains
(CR 609.7b); the allowance is not a finite damage-prevention budget.

RON lowers this resolving shape:

```ron
CreateFuturePreventDamage(
  NextTimeCombatDamageWouldBeDealtThisTurnByCreatureToRecipient(
    TheChosenPermanent, Player(You)),
  [PreventAllButSomeOfThatDamage(Integer(1))])
```

The source uses the existing typed permanent-reference adapter, including a
preceding `ChooseAPermanent` result; the player and amount use their existing
scoped adapters. Exactly one operation is required. Nil, unbound, wrong-role,
malformed and negative fixed declarations reject before gameplay. Dynamic
negative amounts become zero, as in other resolving amount operations.
Permanent recipients, other events and other prevention operations gain no
support from this adapter. Spell actions, activated/triggered actions and delayed
bodies use the same resolving effect and retain their own bindings.

Forcefield chooses an unblocked creature at resolution without targeting.
Shroud does not prohibit that choice. Blocker declaration sets unblocked status;
activation before that declaration is legal but an empty choice installs no rule.
The installed shield checks creature type and exact identity before damage.
Noncombat damage and damage to other recipients cannot consume it. A phase cycle
keeps identity; departure keeps exact damage-source facts; return cannot replace
that object. Cleanup ends the rule. Game and stack copies keep independent state.

Evidence: `TestRONNextCombatPreventionForcefield`, `Lifecycle`, `Invalid` and
`Inventory`, and native `TestNextCombatPrevention*`. All five complete Forcefield
records compile. Full collection execution and runtime cutover remain separate.

## Selected-source prevention operations

Use `PreventNextDamageFromSelectedSourceWithOperations(source, player, filter,
PreventProposedDamage(amount))` for the next matching source event this turn.
Use `ProposedDamageAmount()` for all of the proposed damage or
`HalfProposedDamageRoundedDown()` for half. Matching simultaneous packets share
one amount, read after earlier replacements. Prevention of zero keeps the rule.
Append `GainLifeFromPreventedDamage(player PlayerRef)` to gain life for a
separately saved player from this rule's actual
prevention after the original damage event through the shared life service. The
outcome stays local to that rule.
Supply one prevention operation and at most one following life gain operation.
Definition validation rejects malformed programs and unbound references.

The saved source uses existing selected-source identity, property rechecks and
permanent-spell continuity. Cleanup expires the rule; phasing keeps its exact
reference, and a returned object cannot inherit it. Existing next-source APIs
remain available. These operations do not use `ResolutionValue` or a global
last-result slot.

RON uses `CreateFuturePreventDamage(
NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,
Player(playerRef)), operations)` after a damage-source choice in the same scope.
Supported lists contain `PreventThatDamage` or
`PreventSomeOfThatDamage(WouldDealDamage_ThatMuchDamage)` or
`PreventSomeOfThatDamage(HalfRoundedDown(WouldDealDamage_ThatMuchDamage))`,
optionally followed by `GainLife(TheAmountOfDamagePreventedThisWay)`. The proposed
amount and actual result cannot escape this prevention list. Implicit `GainLife`
uses `You`, independent of the event recipient. Dark Sphere and
Reverse Damage have complete-record gameplay tests for these shapes.

### Temporary mana type replacement

Use `ReplaceTappedManaTypeForPlayerThisTurn(player, filter, to)` to change all
mana types from matching permanents controlled by the saved player when that
player taps them for mana. Cleanup ends the effect. Use
`ReplaceTappedManaTypeUntilLeaves(subject, from, to)` to change one type from a
saved permanent until it leaves. A phase cycle keeps that identity. Neither
operation changes amounts or spending restrictions.

`Game.ManaTypesAbilityCouldProduce(ability, sourceID)` reads possible types after
replacement, with no payment or choice. Raw production metadata remains separate.
`Game.WaitingManaReplacementPermanentSources()` reads exact subjects of active
indefinite rules, including phased subjects, for damage-source choices.

RON uses `CreateReplaceWouldProduceManaUntil/3` in resolving actions. The player
form supports `PlayerTappingAPermanentWouldProduceMana(You, controlledFilter)`,
one `WouldProduceMana_ReplaceType(concreteType)` operation, and `UntilEndOfTurn`.
The filter must include `ControlledByAPlayer(SinglePlayer(You))`.
The exact permanent form supports `TappingPermanentWouldProduceMana(ref)`, one
`WouldProduceMana_ReplaceTypeOfType(from, to)` operation, and
`UntilPermanentLeavesBattlefield(ref)` with the same reference. These shapes
implement Deep Water and Quarum Trench Gnomes. The latter replaces only white
mana, including when the Plains has an ability that also produces another type.

## RON library-top ante actions

Use bare `AnteTopCardOfLibrary` in a resolving action list to ante the action
player's current library top. The default player is the action controller.
`PlayerAction(playerRef, AnteTopCardOfLibrary)` supplies a typed explicit player;
`PlayerActions(playerRef, [AnteTopCardOfLibrary, ...])` uses its saved scoped
player. `EachPlayerAction(AnyPlayer, AnteTopCardOfLibrary)` performs one shared
batch for all players. All eligible cards move before zone events.

Empty libraries and ineligible tops do not stop later instructions. Disabled
or settled ante games move no cards. The action does not reveal a choice, draw,
pay a cost or change ownership. Tuple spellings and cost, entry or static uses
reject. Optional ante payments and `PaidCost` players remain unsupported.
Bare `DiscardHand` in an ordinary action uses the existing scoped hand-discard
service, so Contract from Below can discard, ante and draw in order.

Engine services are `AnteTopFor`, `AnteTopEach` and `Game.AnteLibraryTops`.
They keep typed player bindings and existing delayed source-reference contracts.
See `TestRONAnteTop*` and the library-top ante engine contract.

## Typed resolution ante payments

`AnteTopCost() Cost` pays with the payer's current library top. The shared
payment transaction locks each successive top on its validation game. It checks
all locked payments on a separate game before spending live mana or other
resources. Empty libraries, foreign-owned cards, tokens, disabled ante and
settled games cannot pay. Affordability queries do not move cards or offer choices.
Direct payment and transactional payment use `AnteLibraryTops` for movement.

`PayPermanentAnteOrElse(player PlayerRef, subject PermanentRef, paid, unpaid
ResolvingEffect) ResolvingEffect` offers a payment with one exact permanent.
`AntePermanentOrElse` has the same signature and performs a mandatory payment
without a may choice. The permanent is captured before an optional offer.
The payer must own it, but need not control it (CR 407.4). Phased or departed
subjects cannot pay. A returned card cannot replace the captured incarnation
(CR 400.7). The transaction uses `MoveToAnte`, with existing departure/LKI,
attachment and zone-event behavior. This is neither destruction nor sacrifice;
regeneration and graveyard-destination replacements do not apply.

Each result is local to its invocation and selects exactly one branch after
payment (CR 118.3, 118.10–12). Missing players perform no branch. A nil branch
performs no action; two nil branches are invalid. Definitions validate typed
players, subjects and non-nil branch declarations. Delayed queries enumerate
explicit source operands and bodies without resolving them or offering payment.
Pending copies use their own payer, saved source and frame; clones keep separate
zone storage and delayed lifetimes.

RON supports `MayCost(AnteTopCardOfLibrary)` and
`PlayerMayCost(playerRef, AnteTopCardOfLibrary)`; `MayCost`, `PlayerMayCost` and
`MustCost` support `AnteAPermanent(SinglePermanent(permanentRef))`. These forms
require an adjacent `If`, `Unless` or `IfElse` on bare `CostWasPaid` or
`CostWasntPaid`, with nonempty action lists. Existing resolving spell, activation,
trigger and delayed contexts use the same adapter. Per-factory references retain
lexical bindings. No result is available outside its continuation. Activation
costs, entry/static operations, mandatory top payments, arbitrary permanent
collections and `EachPlayerMayCost` are not supported by this adapter.

Evidence: native `TestAntePayment*` covers owner/control distinctions, token,
phasing, return, offer-time return, disabled/settled games, exact physical IDs,
transaction rollback and stale preflight. `TestRONAntePayment*` covers actual
casting and activation, branch-local nested results, explicit target payers,
permanent subjects, clone/copy behavior, delayed identity and invalid contexts.
Jeweled Bird's retained payment subrule uses a draw continuation. Its complete
record and ante movement now have separate `TestRONAnteMovement*` coverage.
Rebirth's complete record still rejects paid-player collection dispatch. These
are prerequisite mappings, not complete-card gameplay claims.

## Exact ante card targets

`DeclareAnteCardTarget(name string) *CardTargetBinding` declares exactly one
card in the shared, public ante zone. It has no ownership restriction. The
existing `CardRef` and target declaration validation supply typed reads and
per-instance identity. Disabled or settled ante games supply no candidates.

Each ante arrival has a separate visit. `SetAnte` keeps visits of cards that
remain in the zone, including copy-on-write ownership changes. Removing a card
ends its visit; adding that physical card again creates a new visit. Existing
initial ante, `MoveToAnte` and `AnteLibraryTops` use this same storage. Clones
keep visits and separate subsequent mutations.

Casting and activation save ante targets before payment. The existing stack,
resolution and copy services retain the saved zone and visit. An absent or
returned target is illegal (CR 400.7 and 608.2b). No LKI makes it available.
An ownership change alone keeps it legal. Copies without new target choices
retain the old visit. A legal new choice can select the current visit of the
same physical card. Queries do not move cards, make choices or consume RNG.

RON supports `TargetAnteCard(AnyCard)` in the existing singular `Targeted`
spell, activation and trigger contexts. It declares local `Ref_TargetAnteCard`
as a typed card reference with a fresh binding per factory instance. Other
filter shapes and incompatible permanent/player/graveyard reads reject at
compilation. Inventory descends into the still-unsupported exchange action.
Targeting alone does not supply movement or ownership exchange. The movement
contract below establishes a separate ante-to-graveyard operation.

Native `TestAnteReference*` covers visits, current ownership, typed reads,
prepayment capture, zone changes during payment, clone isolation, copies,
legal retargeting and all existing ante arrival paths. `TestRONAnteReference*`
covers casting, failed payment, return illegality, independent activations,
entry triggers, invalid declarations and semantic inventory. Darkpact's actual
target subrule runs with a test-only life continuation; all six complete
records still reject their exchange action. This is prerequisite gameplay
coverage, not complete Darkpact coverage. Complete Jeweled Bird coverage is
recorded in the ante-to-graveyard movement contract below.


## Typed ante-to-graveyard movement

`AnteCardsOwnedBy(owner PlayerRef) AnteCardSet` selects exact current ante
visits with that current owner, across both players' storage. Selection is pure.
`OtherAnteCards(subjects AnteCardSet) AnteCardSet` excludes the physical source
card after payment moves it to ante (CR 400.7j).
`PutAnteCardsInGraveyards(subjects AnteCardSet) ResolvingEffect` saves the
selection once, removes duplicate visits, and moves each available card to its
current owner's graveyard. Ownership changes do not replace a visit. Departure
and return do. Disabled or settled ante games and missing owners move nothing.

Owner-controlled graveyard order uses the existing card-choice service and the
same helper as milling. Choices follow APNAP order (CR 101.4), use card copies,
and precede all movement. Invalid answers use the first remaining card. The
operation preserves prior graveyard order (CR 400.5 and 404.3). Exact visits and
ownership are checked again after choices. All moves finish before zone events;
events identify the destination owner. New graveyard visits use existing player
storage. This is neither a death nor milling; battlefield-departure replacements
and mill modifiers do not apply. No general zone replacement contract is added.
Clones retain separate zone storage. No absent-card LKI enables movement.

RON action shape: `PutEachCardFromAnteInGraveyard(OwnedByAPlayer(SinglePlayer(playerRef)))`.
The owner uses existing typed player references in resolving spell, activation
and trigger action contexts. `Other(OwnedByAPlayer(SinglePlayer(playerRef)))`
is supported only in the paid branch of an adjacent source-permanent ante
payment and its immediate nested actions. It is rejected outside that branch,
on unpaid paths and in delayed bodies. Entry, cost and static contexts reject.
Other filter shapes, invalid arity and unbound/incompatible references reject
at compilation. Inventory classifies the operands as ante card filters, player
filters and player references, including descendants of rejected declarations.

Jeweled Bird's full ARN record uses explicit `Other(...)` to preserve Oracle's
“all other cards.” `TestRONAnteMovementJeweledBird` covers payment, source
exclusion, ownership/control distinctions, departed/returned/phased sources,
clones, ability copies, empty other-card sets and drawing. `BirdSequenceAndOrder`
covers deck preparation, tap payment, owner order, simultaneous movement and
payment-before-movement-before-draw. `ExplicitPlayerAndInventory`, `Declarations`
and `SourceScope` cover typed owners, reporting and invalid declarations.
Native `TestAnteMovement*` covers saved visits, ownership changes, duplicates,
multiple owners, APNAP decisions, invalid choices, return during a decision,
clone storage and invalid definitions. Existing mill-order tests protect the
shared helper's callers. Darkpact still rejects its ownership/exchange operation;
the exact ante-departure helper is available for that later atomic operation.


## Typed ante ownership and library exchange

`AcquireCardOwnership(card CardRef, owner PlayerRef) ResolvingEffect` changes
ownership through `Game.ChangeOwner`. It requires a present nontoken card and
an existing player in an enabled, unsettled ante game (CR 407.3). It does not
move the card, change its visit or alter a controller. Copy-on-write replacement
keeps game clones separate and preserves original-owner settlement history.
Graveyard replacement now keeps the selected and neighboring cards' visits;
`SetGraveyard` retains its existing whole-zone replacement behavior.
Hand, exile and ante replacement retain their existing visit preservation.
Saved card references supply no LKI permission to act on departed objects.

`ExchangeAnteCardWithLibraryTop(card CardRef, player PlayerRef) ResolvingEffect`
requires a current exact ante visit owned by the selected player. A nonempty
library must have a nontoken top owned by that same player (CR 701.12a/d).
The action reads the top at resolution. It moves the ante card to the top of
that library and the former top to ante, with both moves complete before zone
events. An empty library receives the ante card (CR 701.12d). The library tail
keeps its order. Ante departure ends its visit; the incoming top gets a fresh
ante visit (CR 400.7). An invalid operand prevents the entire exchange.
This operation does not acquire ownership. A preceding ownership instruction
still succeeds if a foreign-owned or token library top prevents exchange.
No draw, reveal, shuffle, payment or battlefield event is added.

RON shapes are `AcquireOwnershipOfCard(anteCardRef, playerRef)` and
`ExchangeAnteCardWithTopCardOfPlayersLibrary(anteCardRef, playerRef)` in resolving
spell, activation and trigger actions. `anteCardRef` must be the bound
`Ref_TargetAnteCard` from `TargetAnteCard(AnyCard)`. Player operands use the
existing typed player grammar and per-instance bindings. Wrong arity, kind,
zone, unbound references and cost, entry or static uses reject at compilation.
Semantic inventory classifies both operands as card/player references.
Explicit source reads in the player operand are declared to waiting delayed
queries; the queries do not resolve operands or change ownership or zones.

All six Darkpact records now contain the missing independent Oracle ownership
instruction before exchange. Local `data/LEA.json` supplies the Oracle evidence.
`TestRONAnteExchangeDarkpact` covers every reprint, payment rollback, timing,
foreign ante ownership, empty and foreign-owned libraries, target departure and
return, spell copies, clone isolation and the spell's final graveyard move.
`TestRONAnteExchangeContextsAndInstances` covers independent activations,
triggers, source phasing and source return; declaration/role tests reject invalid
forms. `TestAnteExchange*` covers native atomic events, exact visits, current
tops, disabled/settled games, missing players, token tops, independent ownership,
other nonbattlefield zones, settlement and definition validation.

This contract does not implement two-card ownership exchange, result-bound exile
or reveal adapters, paid-player collections or general zone replacements.
Bronze Tablet and Tempest Efreet remain rejected; Rebirth needs paid-player
collection dispatch. Go definitions and the legacy DSL remain until the runtime
and removal gates pass.

### Complete name catalog and required resolution choice

RON `ChooseACardName(AnyOracleCard)` lowers to
`mage.ChooseCardName(player mage.PlayerRef, catalog *mage.CardNameCatalog,
result *mage.CardNameBinding) mage.ResolvingEffect`. The chooser comes from the
current action player: controller by default, or the explicit `PlayerAction`
or `PlayerActions` player. Spell, activation and trigger resolution bodies are
supported. Cost, entry and static contexts reject this operation. Other name
filters, malformed tuples and `AnyOracleCard()` reject at compilation.

`mage.OracleCardNameCatalog() *mage.CardNameCatalog` supplies an immutable,
embedded snapshot independent of registered/executable cards, decks, formats,
and filesystem paths. `Names() []string` returns separate sorted storage;
`Contains(string) bool` tests an exact English face name. The source and hashes
are in `pkg/mage/oracle_card_names_source.json`. The Scryfall nontoken English
name catalog supplies canonical names, with combined names split into separate
faces under CR 201.4b–f. The interchangeable-name supplement below adds the
original English names omitted by that endpoint. A later update must retain
these checks.

`mage.BindCardName(string) *mage.CardNameBinding` declares one result.
`(*mage.CardNameBinding).Resolve(*mage.EffectContext) (string, bool)` reads the
current frame. Each RON factory constructs separate declarations. Choices run
at resolution through `Player.ChooseMode`, with a copy of the name list.
Invalid indices select the first legal name; an absent chooser clears the result.
This is a required choice and has no decline branch. Frame cleanup, cloning,
nested application and loops keep separate result storage.

This establishes the catalog and choice prerequisite for Nebuchadnezzar and
Petra Sphinx. Typed `TheChosenName` operand lowering and unconditional
`PlayerActions` result scope are established below. Typed
`IsNamed(TheChosenName)` matching and a resolving hand-discard consumer are
established below. Result-bound reveal checks and result capture across
delayed bodies need separate typed compiler/engine consumers. Neither
real card is accepted or claimed as complete gameplay by this prerequisite.

### Typed chosen-name operands and lexical scope

`mage.FromCardName(*mage.CardNameBinding) mage.CardNameRef` supplies a pure,
typed operand. `CardNameRef.Resolve(*mage.EffectContext) (string, bool)` reads
only the current frame. An unwritten or cleared result is absent. Native
validation checks the operand's binding and requires a producer on every path.
The existing binding API and name-choice behavior remain available.

RON `compileCardNameRef(Node, scope)` lowers the exact symbol `TheChosenName`
to an instance factory for this operand. A declaration requires a preceding
choice in its lexical environment. Tuples, other reference types and unbound
names reject. Choice plans and binding maps give each factory independent
name declarations; the engine frame supplies each resolution's value.

Action sequences retain the most recent unconditional choice. Nested
`PlayerActions(player, [actions])` inherits the outer name environment and
exports its final unconditional name choice to later outer actions. A scope
with no new choice retains the outer choice. New choices keep separate
bindings, so previously compiled reads keep their original producer. The
scoped player does not replace the outer action player. Conditional branches
receive the outer environment, but cannot export a choice from only one path.

This is an operand/scope prerequisite for Nebuchadnezzar and Petra Sphinx.
It does not accept their reveal consumers. The matching service below uses
retained interchangeable-name evidence under CR 201.3 and 201.4g. Prior-name
capture by delayed consumers, result export from a single
`PlayerAction`, optional branches and loops remain separate contexts.

Evidence: `TestRONCardNameScopeOperand`, `PlayerActions`, `Sequence`,
`BranchBoundary`, `RealOperands`; native `TestCardNameRefContract`,
`Declarations`, `Frames`, `PendingResolution`. Existing RON name-choice
spell/activation/trigger/payment/copy gameplay tests remain in force.

### Resolving chosen-name filters

`IsNamed(TheChosenName)` lowers through
`compileResolvingCardNameFilter(Node, scope) (cardNameMatcherFactory, error)`.
Its factory builds `mage.MatchesChosenCardName(mage.FromCardName(binding))` with
the same per-instance declaration as the dominating choice. The matcher checks
supplied characteristic names under CR 201.2a/201.3/201.4g. It does not select
objects or expose private information. A missing result cannot match.

`PlayerAction(player, DiscardCards(IsNamed(TheChosenName)))` and
`PlayerActions(player, [DiscardCards(IsNamed(TheChosenName))])` use
`mage.DiscardHandMatchingName(player mage.PlayerRef, matcher mage.CardNameMatcher)
mage.ResolvingEffect`. Membership comes from the current hand at resolution.
Replacements and simultaneous discard events use the existing shared batch.
The name remains local to its declaration and frame. RON action lists pass
through `mage.Sequence` so native validation checks instruction order.

The independent legal-choice catalog now contains 35,950 names, including the
31 original English names from 31 interchangeable face pairs. The supplement
and per-Oracle-identity source evidence are retained in
`pkg/mage/oracle_name_equivalences.txt` and its `_source.json` file. English
`printed_name` on these printings supplies original names; artwork flavor titles
are excluded. Both names in a pair can be chosen and match either member.

Cost/static/entry uses, nested boolean name filters and reveal consumers remain
unsupported. This slice does not accept the full Nebuchadnezzar or Petra Sphinx
records. Evidence: `TestRONCardNameMatch*` and native `TestCardNameMatch*`.

### RON random hand reveal results

The resolving action sequence supports
`PlayerAction(player, RevealACardFromHandAtRandom)`. The reveal operation must be
a bare symbol. It reveals one card from the selected player's current hand at
resolution and leaves that card in the hand. Empty hands and absent players
clear the result without consuming randomness. This is a public reveal, not a
private inspection or a cost.

The continuation has a lexical typed card result,
`TheCardRevealedFromHandThisWay`, lowered to `mage.FromCard(binding)` by
`compileRevealedHandCardRef`. The current supported RON consumer is
`PlayerAction(player, DiscardTheCardRevealedThisWay)`. It requires the same exact
hand visit and membership in the explicit player's hand, then uses the shared
discard replacement and event service. A different player, departed card, or
returned visit cannot supply the discard. The native API is
`RevealRandomHandCardInto(PlayerRef, *CardBinding) ResolvingEffect` and
`DiscardCardFromHandFor(PlayerRef, CardRef) ResolvingEffect`.

The producer owns its remaining action sequence. Each factory build has separate
bindings; a later producer shadows the earlier result only in its continuation.
Optional payment bodies retain their local scope. Unbound reads, reads before
production, malformed symbols, and incompatible player/card references reject at
compile time. Direct `PlayerActions` reveal production and export, drawn-card
results, library reveals, reveal predicates, mana-value results, plural random
reveal, and ownership/movement consumers remain unsupported. Spell, mode,
activation, trigger and delayed resolving action sequences share this adapter;
no cost, static, entry or replacement-operation adapter is added.

Native resolution frames, saved card visits, public reveal snapshots, scripted
randomness and clone handling are reused. Permanent loops now isolate singular
card bindings as well as their other existing saved values. Typed player source
reads participate in waiting delayed queries without resolving operands or
consuming randomness. Public history records the revealed visit's facts; it
does not expose the rest of the hand or a later returned visit.

Evidence: `TestRONRandomHandReveal*` and native `TestRandomHandReveal*`.
Complete Wand of Ith and Tempest Efreet records supply real producer ASTs to
engine-backed tests; both full records remain rejected. This is subrule coverage,
not complete-card gameplay coverage. Rules: local CR 400.2, 400.7, 608.2c,
701.20a–c. Reveal itself causes no zone change; discard replacements remain in
the shared pipeline.

### RON reveal-result predicates

Resolving `If`, `Unless` and `IfElse` accept
`ACardWasRevealedThisWay(filter)` after the singular random hand producer.
The condition has one operand. Supported filters are `IsCardtype(type)`,
`IsNonCardtype(type)` and `IsNamed(TheChosenName)`, each with one operand.
Types use the compiler's existing card-type vocabulary. The name requires a
prior lexical name choice and uses the shared interchangeable-name matcher.
Unsupported filters, missing producers or choices, wrong arities, empty explicit
branches and branch-local results used outside their branch reject at compilation.
Mana solver bodies remain unsupported.

The predicate checks the name and card types saved when the card was revealed.
A missing or empty reveal never matches, including `IsNonCardtype`. `Unless`
then runs its body; `IfElse` runs its second body. The matcher runs once before
its selected body. No predicate moves, reveals or chooses another card.
`TheCardRevealedFromHandThisWay` retains its separate exact-visit permission.
Historical facts cannot make a returned hand card available for discard.

The engine owns `RevealedCardMatcher`, `RevealedCardHasType(CardType)`,
`RevealedCardLacksType(CardType)`, `RevealedCardNamed(CardNameMatcher)` and
`IfRevealedCardMatches(*CardBinding, RevealedCardMatcher, ResolvingEffect,
ResolvingEffect)`. RON owns spelling, lexical scope and branch lowering.
A nil native body makes no change; at least one body is required. These are
resolving history predicates, not static or event-trigger conditions.

Evidence: `TestRONRevealResultGameplay`, `Invalid`, `RealOperands`, `Contexts`,
`DistinctProducers`, `WandLandBranch` and native `TestRevealResult*`. Actual Wand
of Ith, Petra Sphinx and Sindbad predicates execute through a bounded hand
producer in subrule tests. Wand's complete land-payment branch executes unchanged.
Wand's second predicate is corrected to `IsNonCardtype(Land)` using retained
`data/DRK.json` Oracle text. Complete Wand still needs dynamic life payment;
Petra Sphinx needs library results and Sindbad needs drawn-card results. No full
card is newly accepted. Earlier statements that reveal predicates are unsupported
are superseded only for this singular hand context.

### Saved hand reveal from a single draw

Use `DrawFor(player, Fixed(1), DrawCardInto(drawn))`, then
`RevealHandCardInto(FromCard(drawn), revealed)` to retain a reveal result.
`IfRevealedCardMatches` reads saved name/type facts from `revealed`.
`DiscardCardFromHandFor(player, FromCard(drawn))` keeps the explicit player
and original hand visit. A replacement draw produces no original result.
Reveal moves nothing; departure/return invalidates actions but cannot rewrite
historical facts. Each declaration must have a dominating producer.

RON resolving action lists support `DrawACard`, `RevealTheCardDrawnThisWay`
and `DiscardTheCardDrawnThisWay` with separate lexical bindings. Draw and
discard permit explicit `PlayerAction` subjects. Draw results stay local to
branches and PlayerActions bodies; only the supported outer action-list
continuation can read them. Cost/static/entry/replacement and plural/library
producers remain separate contexts. Sindbad's complete ARN record supplies
end-to-end gameplay evidence in `TestRONDrawnCardRevealSindbad`.

### Library-top reveal results

Use `RevealLibraryTopInto(player, result)` with a `CardBinding` for a public
reveal of the current library top at resolution. `FromCard(result)` requires
its original library visit. Departure, return, shuffle or reordering end that
reference. A continuously public top also gets a new visit when it stops
being revealed (CR 401.6). Removal of another card preserves the visit.
`IfRevealedCardMatches` reads
immutable name/type facts after identity ends. Empty libraries clear the result;
reveal does not draw, mill or move a card.

RON supports the symbol `RevealTopCardOfLibrary` in resolving action lists,
`PlayerAction(playerRef, RevealTopCardOfLibrary)`, and unconditional nested
`PlayerActions(playerRef, [..., RevealTopCardOfLibrary])`. The scoped reveal
result can reach the outer continuation; the player scope stays local. Existing
`ACardWasRevealedThisWay` type/name predicates inspect that result. Results are
separate for each producer and factory. Conditional producers stay in their
branch. Hand-discard operands cannot consume a library result. Cost/static/entry/
replacement and tuple forms remain unsupported.

`TestRONLibraryTopRevealPetraPrerequisite` retains producer/predicate evidence.
`TestRONLibraryTopMovePetra` now executes the complete Petra Sphinx record.
This supersedes earlier library-producer and Petra movement gap statements.


### Resolving library-top movement

Use `PutLibraryTopInHand(player PlayerRef) ResolvingEffect` or
`PutLibraryTopInGraveyard(player PlayerRef) ResolvingEffect` to move the selected
player's current library top. Each action reads the top when it resolves, routes
it to its owner's corresponding zone, and sends one library-to-destination zone
event after movement. Empty libraries and absent players have no effect. The
hand action is not a draw; the graveyard action is not a mill. Draw replacements,
draw history, empty-library draw loss, and mill count modifiers do not apply.

RON supports the bare symbols `PutTopOfLibraryInHand` and
`PutTopOfLibraryInGraveyard`, explicit `PlayerAction(playerRef, action)`, and
nested `PlayerActions` bodies in spell, activation, trigger and delayed resolving
lists. They need no reveal producer and do not consume a saved reveal binding.
Tuple forms, arguments, unbound players, costs, static effects, entry operations
and replacement bodies remain rejected. Typed source reads use the existing
waiting delayed-source declaration service without inspecting or moving cards.

Complete Petra Sphinx gameplay maps to `TestRONLibraryTopMovePetra`. Contexts,
player scopes, rollback and clones map to `TestRONLibraryTopMoveScopes` and
`TestRONLibraryTopMoveContexts`; declaration rejection maps to
`TestRONLibraryTopMoveInvalid`. Native contracts map to `TestLibraryTopMove*`.
Linked exile and plural library selection remain separate capabilities.

### RON random hand reveal and matching discard

Resolving player actions accept:

- `RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(value,
  IsNamed(TheChosenName))`: reveal distinct random cards, then discard all
  matching revealed cards. `TheChosenName` requires a dominating lexical choice;
  `ValueX` requires its existing announcement or created-value scope. Fixed
  `Integer(n)` declarations must be nonnegative. Existing typed resolution
  values retain their value and scope checks.
- `RevealHandAndDiscardACardOfTypeAtRandom(filter)`: reveal the whole hand,
  then discard one random matching card. Filters use the existing hand-card
  grammar: card type, color, their negative forms, and nonempty `And`/`Or` lists.

Both work in spell, activation, trigger and delayed resolving lists, bare player
scope, explicit `PlayerAction`, and nested `PlayerActions`. Each instance builds
its own name binding. Unsupported filters, wrong shapes, unbound players or
names, and cost, static, entry and replacement-operation contexts reject.
These compound actions export no reveal result. Nebuchadnezzar and Rag Man use
the shared engine operations; neither needs a card-specific branch.
Evidence: `TestRONRandomHandDiscardRealCards`, `TestRONRandomHandDiscardCast`,
`TestRONRandomHandDiscardContexts` and `TestRONRandomHandDiscardInvalid`.

## Linked exile producer declarations

`mage.NewExileLink() *mage.ExileLink` creates an immutable declaration identity.
Supply the same declaration to linked native producer and query operations:

- `ExilePermanentLinked(subject PermanentRef, link *ExileLink) ResolvingEffect`
- `ExileLibraryTopLinked(player PlayerRef, link *ExileLink, faceDown bool) ResolvingEffect`
- `CardsExiledWith(link *ExileLink, source PermanentRef) ExiledCardSet`
- `CountExiledCards(subjects ExiledCardSet) ResolutionValue`

The query selects current exile visits belonging to both the declaration and the
exact source incarnation. It can supply engine transport of face-down cards; it
does not grant inspection permission. A fresh acquired group must get a fresh
link. Game clones retain immutable declarations and separate mutable exile cards.
The separate ExiledCardSet type returns exact visit references internally and
cannot feed general CardSet choosers or characteristic filters. Nil declarations
and operands fail engine definition validation.

RON supports `Exile([TheTopCardOfLibrary], [])` and
`Exile([TheTopCardOfLibrary], [ExilesFaceDown])` in resolving permanent-source
bodies, including activated, triggered and delayed bodies. The top is read at
resolution. Empty libraries have no effect. This is neither a draw nor a reveal.
`Exile([Permanent(ref)], [])` in these bodies uses linked transport with the
existing typed subject and target legality. Unassociated producers have separate links.
Spell-source permanent exile retains the existing ordinary exile operation.
Costs, entry operations and static operations do not accept library-top exile.
Plural subjects and face-down permanent exile remain unsupported.

RON associates a single unambiguous producer declaration with
`ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(ThisPermanent))` and
`PutExiledCardsIntoOwnersGraveyards(WasExiledWithPermanent(ref))` in resolving
activated, triggered and delayed bodies. The reference is `ThisPermanent`, or
`Trigger_ThatPermanent` from a source-departure event. Missing or multiple
producer declarations and incompatible references reject. The immutable link
belongs to the compiled definition; the exact source incarnation separates
physical instances, pending resolutions and returned objects. No mutable result
is stored in the declaration.

Native `ReturnExiledCardsToOwnersHands(ExiledCardSet) ResolvingEffect` and
`PutExiledCardsInOwnersGraveyards(ExiledCardSet) ResolvingEffect` select current
exile visits at resolution. They check each visit again after graveyard-order
choices, move cards to their current owners, and send zone events after all
moves. Face-down transport gives no inspection permission while cards remain
in exile. Waiting delayed queries retain explicit source reads without selection,
choices or movement. Knowledge Vault has complete-record gameplay evidence.
Safe Haven's battlefield consumer uses the owner-controlled entry contract below.
Multiple linked producer groups still require separate adapters.


## Mandatory payment of one typed permanent

`IfRequiredSacrificePaid(player PlayerRef, subject PermanentRef, body ResolvingEffect)`
and `UnlessRequiredSacrificePaid(player PlayerRef, subject PermanentRef, body ResolvingEffect)`
use the exact-subject sacrifice service with mandatory payment. The payer must
control the present, unphased incarnation. A saved reference cannot select a
returned card. Ownership controls the destination, not permission to sacrifice.
An absent payer runs no body. Declaration validation rejects nil operands,
nil bodies and unbound subjects.

The service reads the subject once, then uses the existing exact-incarnation
filter and matching sacrifice transaction. It makes no optional payment offer.
The shared replacement pipeline can change or prevent movement after payment
starts; this does not undo success (CR 118.11–12). Regeneration and indestructible
cannot prevent sacrifice (CR 701.21a). Each invocation selects its own body;
nested payments, stack copies and game clones do not share payment results.
Waiting delayed queries retain explicit source reads in the subject, player and
body without resolving operands, testing affordability or making payment.
No new replacement order, zone storage or simultaneous-operation rule is added.

RON supports `MustCost(SacrificePermanent(ref))` followed immediately by
`If` or `Unless` with symbol `CostWasPaid` or `CostWasntPaid` and a nonempty
action list. `ref` uses the existing typed permanent scope: source, declared
target, event, choice or Aura host where available. Activated, triggered and
delayed resolving bodies use their existing identity and legality rules.
An unbound reference, invalid symbol/arity, separated continuation, empty body,
static or entry operation rejects. Two-branch `IfElse` remains unsupported.
Optional typed sacrifice and required matching sacrifice keep their contracts.

Evidence: `TestRequiredTypedSacrificeIdentity`, `Definition` and
`ReplacementOrdering` and `Rollback` cover legality, owner destination, phasing, return,
control, native declaration errors, movement prevention, competing replacements,
sacrifice LKI triggers, death exclusion and invalid-choice rollback. `TestRONRequiredTypedSacrifice*`
covers Knowledge Vault's real payment rule with an observable test continuation,
announcement/resolution timing, changed-controller copies, cloned pending
resolution, branch polarity, independent nested results, resolving contexts,
waiting-source queries and target legality. This is payment-rule evidence;
Knowledge Vault uses the linked-exile consumer contract for complete-record gameplay.


## Linked exile entry under owners' control

`PutExiledCardsOntoBattlefieldUnderOwnersControl(subjects ExiledCardSet) ResolvingEffect`
selects exact exile visits once at resolution. Each present nontoken permanent
card enters under its current owner's control. Entry replacements, copy choices,
tapped status and counters use the shared battlefield entry pipeline. This is
not casting; announced X and cast colors are zero during entry, and the enclosing
resolution's X is restored afterward. Each prepared definition is an independent
card copy with the physical card ID and printed departure provenance.

Entry choices use active-player order while the cards remain in exile. Each
owner chooses the order of their simultaneous entry choices. All saved
visits, owners, Aura hosts and land-entry prohibitions are checked again after
choices. A new exile visit cannot replace the saved subject. Prohibited lands,
nonpermanent cards, tokens and Auras without legal hosts remain in exile. Aura
host selection uses the existing permanent attachment legality service without
targeting; shroud does not prevent entry. Hosts must already be on the battlefield
and retain their incarnation. Existing entry operations that return failure keep
their graveyard destination, with owner ordering before movement. All successful
entries and graveyard moves finish before zone events. Clones have separate state.
Waiting delayed queries retain the linked source without choices or movement.

RON supports exactly
`PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ref), [EntersUnderOwnersControl])`
in activated, triggered and delayed resolving permanent bodies. The existing
one-producer association and source-reference checks apply. `ref` is
`ThisPermanent` or the source-departure `Trigger_ThatPermanent`. Costs, static
rules, entry declarations, other sets and other entry options reject. Safe Haven
has complete-record gameplay evidence. Tawnos's Coffin still needs result-bound
exile, counter notes, explicit attachments and conditional return payments.

## RON saved reveal mana value and resolution life payment

`ManaValueOfTheCardRevealedThisWay` is a symbol in resolving numeric operands.
It reads the current lexical singular reveal: random hand, revealed drawn card,
or library top. It lowers to `RevealedCardManaValue(result *CardBinding)
ResolutionValue`. Each reveal saves mana value with its immutable name/type
facts. X contributes zero outside the stack. Missing reveal facts supply zero;
a saved value cannot authorize an action on an absent or returned card.

Optional resolution payments support exactly
`MayCost(PayLife(ManaValueOfTheCardRevealedThisWay))` and
`PlayerMayCost(player, PayLife(ManaValueOfTheCardRevealedThisWay))`, followed by
an adjacent nonempty `If` or `Unless` body using `CostWasPaid` or `CostWasntPaid`.
These forms work in supported spell, activated, triggered and delayed action
lists. Fixed payments retain their existing behavior. Unbound results, reads
before production, branch-local results used outside their branch, tuple value
spelling, activation/additional costs, mandatory payments and dynamic alternatives
reject before execution. Two-branch dynamic payments and other dynamic life
operands remain separate adapters.

`IfLifePaid(player PlayerRef, amount ResolutionValue, effect ResolvingEffect)`
and `UnlessLifePaid` read the amount once before offering payment. The shared
life cost and payment transaction enforce affordability, recheck after the choice
and subtract the saved amount. Nonpositive amounts become zero; zero still
requires acceptance. An absent payer runs no body. Payment results stay local.
Delays and clones retain separate result maps and choices. Explicit source reads
in payer, amount and body supply waiting references without reading amounts,
choosing payment, revealing cards or executing the continuation.

Evidence: `TestRONResolutionLifeWand` executes the complete DRK record;
`TestRONResolutionLifeContexts`, `DistinctProducers` and `Invalid` cover the
reusable shape and its scope. Native `TestResolutionLifePayment*` and
`TestRevealedCardManaValue` establish payment, reveal identity and frame contracts.

## Grouped permanent exile results

Native declarations:

- `PermanentExileGroup{Subjects PermanentSet, Result *CardsBinding}`.
- `ExilePermanentGroupsInto(groups ...PermanentExileGroup) ResolvingEffect`.
- `ExiledCardsFrom(binding *CardsBinding) ExiledCardSet`.
- `SingleExiledCardFrom(binding *CardsBinding) CardRef`.

Select all groups before one exile batch. Use separate named bindings for each
group. Overlapping members move once; every matching group saves the same visit.
Empty results clear. Tokens supply no retained card. Only current unphased
incarnations move. Result writes precede batch events. Saved visits expire on
exit or re-exile; a plural result cannot become a singular reference when one
member remains. Existing frames, delays and clones keep separate result storage.
Transport sets do not grant inspection; singular card reads require visibility.

RON resolving sequences accept two or more `Permanent(ref)`/`EachPermanent(filter)`
subject groups with empty modifiers. `AttachedToPermanent(ref)` selects current
attachments through the existing set service. Two singular declarations supply
first/second card references; one singular plus one collection supplies
single/other references. One explicit source group supplies `ThisExiledCard`.
These symbol references are lexical and cannot escape conditional producers.

Singular results can feed `PutExiledCardIntoOwnersGraveyard`. The other collection
can feed the existing plural hand/graveyard/owner-entry adapters; owner entry
permits only `[EntersUnderOwnersControl]`. `compileCardRef` lowers singular
results for future card consumers and preserves zone restrictions. This contract
does not supply noted counters, attachment entry, ownership exchange, exile
replacement rules or linked accumulation across separate activations.

See the grouped permanent exile contract in `engine-rules-contracts.md` and
`TestGroupedExile*`/`TestRONGroupedExile*` for identity and gameplay evidence.

### Two-card ownership exchange

`ExchangeCardOwnership(first CardRef, second CardRef) ResolvingEffect` exchanges
the current owners of two present nontoken cards in an active, unsettled ante
game. Both references and both owners must be valid before either change.
The same card or the same owner on both operands makes no change. The operation
keeps zones, zone visits, controllers and attachments, and sends no zone events.
It uses `Game.ChangeOwner` for copy-on-write storage and settlement history.

RON accepts `ExchangeOwnershipOfTwoCards(cardRef, cardRef)` in resolving action
sequences, including spell, activated, triggered, conditional and delayed bodies.
Operands support bound singular grouped-exile results, the lexical
`TheCardRevealedFromHandThisWay`, graveyard targets/death results, and ante targets.
Factory instances build separate references and bindings. Producers must dominate
reads. Unknown symbols, wrong arity, unbound results, branch escape, plural exile
results, permanent operands and cost/static/entry contexts reject during compile.
`ThisPermanent` and `ThisCard` do not become physical source-card references.
Explicit destinations and physical source transport remain separate capabilities.

`TestRONCardOwnershipExchangeGameplay` executes Bronze Tablet's actual grouped
exile and exchange nodes in a test-only activation probe. Its target filter and
payment continuation are not included. `TestRONCardOwnershipExchangeRevealContexts`
uses Tempest Efreet's actual reveal/exchange forms with an explicit player and a
graveyard target in place of its unsupported physical source. It covers spell,
activation, trigger and delayed execution. These tests establish prerequisites,
not complete-card gameplay. The complete records still reject.

## RON nontoken permanent filters

The symbol `IsNonToken` lowers to `mage.Not(mage.IsToken)` through the shared
permanent-filter compiler. It works in existing target, collection, fixed
sacrifice-cost, static-selection and resolving-property contexts, including
supported `And`/`Or` compositions. Static condition filters use this same adapter.
It takes no arguments; tuple forms and use as an action, cost or player filter
remain declaration errors. This adds no card-in-zone or spell filter.

Token status belongs to the physical permanent, independently of its name,
owner, controller or copied characteristics. Targets use the existing
announcement and resolution checks. Collections select current unphased
members; static layers select live members each pass. Resolving property reads
retain exact-incarnation LKI. Payments keep the shared transaction and rollback.

`TestRONNontokenFilter*` covers real Bronze Tablet and Dance of Many target
nodes and Lich's filter in test-only probes, plus collections, layers, LKI,
rollback and invalid declarations. These probes do not establish complete-card
support. `TestNontokenFilter*` proves physical copy status, clone isolation and
simultaneous exile with the existing engine services.

## Controller sacrifice of resolving permanent collections

RON `ControllersSacrificeEachPermanent(collection)` accepts one positional
operand in resolving action lists. The operand uses the existing contextual
permanent collection adapter: supported filters, union/intersection, singleton
references, target sets, control/owner sets and attachment/combat collections.
It lowers to `mage.ControllersSacrificeEach(subjects PermanentSet) ResolvingEffect`.
Spelling and binding checks stay in RON; engine services select and move subjects.
This is an effect, not a payment or static modification.

Selection reads current membership once at resolution. Each current controller
sacrifices their selected permanents, even when different from the effect
controller or owner. Exact incarnations, target legality and phasing still apply.
Duplicates move once. Replacement choices and owner graveyard ordering finish
before all movements; the shared zone batch then sends events. Waiting delayed
queries retain explicit source reads without evaluating the collection.

Golgothian Sylex still requires its original-printing name filter. City in a
Bottle still requires its state-trigger collection and player restrictions;
`Trigger_ThosePermanents` remains unbound outside that missing event contract.
No complete-card unlock is claimed. Evidence: `TestRONControllersSacrifice*`
and native `TestControllersSacrifice*`.

### Original-printing name filters

RON symbols `SharesANameOriginallyPrintedInAntiquities` and
`SharesANameOriginallyPrintedInArabianNights` lower to the engine's
`OriginalPrintingNameCatalog(setCode string) (*CardNameCatalog, error)` and
`HasNameIn(catalog *CardNameCatalog) PermanentFilter`. Only symbol forms are
supported; tuples and action/payment/player-filter roles reject. Supported
permanent-filter consumers include targets, resolving collections, matching
sacrifice costs, static selections and resolving property conditions.

The embedded, immutable CR 206.3a/b snapshots contain 77 ARN and 85 ATQ names.
`CardNameCatalog.Matches(names []string) bool` checks supplied current or
retained English names, including the existing interchangeable-name groups.
`Names()` returns separate storage. An unknown set code returns an error;
a nil catalog and nameless objects do not match. Membership is independent of
executable cards and the mutable set catalog. Mountain is excluded from ARN.
`HasNameIn` excludes face-down names but leaves token status to the caller.

Targets keep announcement identity and recheck names at resolution. Collections
read current membership; a returned permanent can be a new collection member.
Static selections read names each layer pass. Resolving information predicates
retain the exact object's departure facts. No information read permits a move
of an absent object. Face-down subjects have their computed P/T fields reset
each layer pass, while their face-down base values remain fixed.

The complete Golgothian Sylex RON record is now executable and has casting,
activation, payment, timing, reprint, copy, clone, owner/controller, phasing,
nameless/token exclusion and simultaneous-sacrifice tests. City in a Bottle's
exact nested filter has a resolving prerequisite probe; its state trigger and
spell/land restrictions remain rejected. This does not establish full Bottle
execution. Source evidence and snapshot hashes are in
`pkg/mage/original_printing_names_source.json`.

## Defined source-counter activation mana X

On an activated `ActionDefinition`, use
`WithActivationManaXFromSourceCounters(counter)` with an X `ManaCostPayment`.
The engine derives payment X from the live source before target checks and
payment. Affordability queries use the same count without changing state.
Substitution precedes cost reductions; counter payment and sacrifice cannot
change the locked mana amount. Stack copies retain payment X without paying.

Resolving counter amounts must use `CountersOnWithLKI(This, counter)` separately.
This API does not supply dynamic resolving X. RON supports the exact modifier
`XIs(TheTotalNumberOfCountersOfTypeAmongPermanents(counter,
SinglePermanent(ThisPermanent)))`, also inside `And([...])`, with `PayManaX`.
Bodies and targets that read `ValueX`, target-derived definitions and plural
counter totals remain rejected. Voodoo Doll uses the supported shape; Reflecting
Mirror needs a separate target-derived announcement contract and a source audit.

## RON temporary player mana payments

In resolving spell, activation and trigger action lists:

```ron
CreatePlayerEffectUntil(You,
    [MayPayManaAnyTimeTheyCouldCastAnInstant([ManaCostGeneric(1)],
        [CreateFuturePreventDamage(
            NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1), Ref_AnyTarget),
            [PreventThatDamage])])],
    UntilEndOfTurn)
```

The player reference and typed lexical environment are saved at installation.
The body retains its original target visits and X; taking a payment selects no
new target. Each payment uses fresh local result storage. Source departure,
phasing and control changes do not revoke the permission. Cleanup revokes it.
The selected player can repeat the fixed mana payment at instant timing. It
uses the shared transaction, resolves immediately and creates no stack object.
Unpaid attempts do not execute the body. Existing automatic mana production
can supply payment. Activation taxes and reductions do not change this cost.

Native APIs are `mage.PlayerManaPaymentThisTurn(player, mana, body)`,
`Game.GetPlayerPayments(player)` and `Game.TakePlayerPayment(player, id)`.
Priority handlers return `PriorityAction{Type: PriorityTakePlayerPayment,
PlayerPaymentID: id}`. Direct callers supply a legal instant-timing opportunity;
queries and execution reject use during stack resolution. Fixed nonnegative
ordinary mana and nonempty operation/body lists are supported. X, hybrid mana,
other durations, life-to-mana and one-use casting are separate capabilities.
Guardian Angel is the complete collection card covered by this adapter.

## RON source-card references after payment

`ExchangeOwnershipOfTwoCards(ThisPermanent, cardRef)` can read the source card
saved by one unconditional `SacrificePermanent(ThisPermanent)` or
`Exile([Permanent(ThisPermanent)], [])` activation cost. The order of the two
exchange operands can be reversed. The source card has its first new public-zone
visit, including sacrifice replaced by exile. A later visit cannot replace it.
Each factory builds a separate `CardBinding` for the producer and consumers.

After a grouped resolving `Exile`, the same operand can read its unambiguous
singular source group through the existing exile result. No arbitrary permanent
operand converts to a card reference. Unpaid or ambiguous sources and tuple
spellings are rejected. `ThisCard` and movement from anywhere remain unsupported.

Native callers use `SacrificeSourceCostWithCard(*CardBinding) Cost` or
`ExileSourceCostWithCard(*CardBinding) Cost`, then `FromCard` in the resolving
body. These result-producing costs require an activated ability declaration.
The engine checks source identity, payment rollback and saved visits; RON owns
spelling and lexical binding. Tests are `TestSourceCardPayment*` and
`TestRONSourceCard*`. This establishes a prerequisite, with no newly accepted
complete card or complete-card gameplay mapping.

### Paired text-word choices at resolution

RON accepts `ChooseTwoColorWords` and `ChooseTwoBasicLandTypes` as symbols in
resolving action lists and explicit player bodies. Each operation saves two
different words using the selected action player and two `Player.ChooseMode`
decisions. It does not run during announcement or cost payment.

The engine APIs are `ChooseColorWordPair(PlayerRef, *ColorWordPairBinding)` and
`ChooseBasicLandWordPair(PlayerRef, *BasicLandWordPairBinding)`, both returning
`ResolvingEffect`. Declare results with `BindColorWordPair(string)` or
`BindBasicLandWordPair(string)`. `FirstColorWord` and `SecondColorWord` return
`ColorWordRef`; `FirstBasicLandWord` and `SecondBasicLandWord` return
`BasicLandWordRef`. Each typed `Resolve` returns its value and a presence flag.
The supported colors exclude colorless. Basic land words exclude nonbasic types.
The second decision excludes the first word. Invalid answers use the first legal
option. Missing choosers clear results. Bindings and saved frames remain separate
across factory instances, copies, clones, loops and delayed execution.

RON lexical operands are `TheFirstChosenColorWord`, `TheSecondChosenColorWord`,
`TheFirstChosenLandType` and `TheSecondChosenLandType`. They require the matching
prior producer in the same supported lexical context. They cannot be amounts,
card names or ordinary entry choices. Pair producers inside player bodies remain
local to those bodies. The compiler adapters retain typed operands for subsequent
text-effect work; `CreateSpellEffect` and color/land text modifications remain
unsupported. No complete-card support gain is claimed for this prerequisite.

### Private hand inspection

`LookAtHand(viewer, player PlayerRef) ResolvingEffect` shows the selected player's
current hand to the viewer during resolution. `Game.HandLooks(viewerID)` returns
copied historical observations; `HandLookObserver.ObserveHandLook([]Card,
GameReader)` receives independent card copies. Neither interface grants later
access to the hand. Empty hands and missing players produce no observation.

RON `LookAtPlayersHand(player)` uses the resolving action controller as viewer.
Its operand uses the current typed player scope, including declared targets,
event players and source-controller reads. It is an action, not a cost, static
modification or entry operation. Glasses of Urza has complete RON gameplay tests.
Word of Command's combined inspection, selection and controlled play remains
unsupported until its separate decision and player-control contracts are ready.

## Repeated optional hand-entry rounds (RON)

The supported resolving shape is:

```ron
RepeatableActions([
    EachPlayerStartingWithMayAction(
        AnyPlayer, You,
        PutACardFromHandOnBattlefield(IsPermanent, [EntersNormally])
    ),
    Unless(NoOneTookAnActionThisWay, [RepeatThisProcess])
])
```

The starting operand accepts an in-scope typed player reference. The hand filter
uses the existing optional-entry filters. The body must contain exactly one
round and its adjacent continuation. The continuation requires the bare result
condition and one bare `RepeatThisProcess`. Other producers, result conditions,
extra actions, tuple-shaped symbols and escaped result/repeat reads reject.

Each factory builds a fresh `NumberBinding`. Lowering uses
`EachPlayerMayPutFromHandOntoBattlefield(PlayerSet, PlayerRef, CardFilter,
*NumberBinding)` with `RepeatWhilePositive(ResolvingEffect, ResolutionValue)`.
The engine counts actual entries, rather than accepted offers. Players act in
seating order from the starting player and read their current hands in each
round. A player can accept after declining a previous round. Normal entry rules
and attachment legality apply; entering a land does not play it.

Round-local binding maps and target captures are separate copies of the outer
frame. Nested execution, errors, stack copies and game clones keep separate
results. Source-read queries do not execute a round. This supports Eureka's full
record; Mana Clash still requires coin-result and damage adapters.

Evidence: `TestRONRepeatedEntry*` and native `TestRepeatedEntry*`.

## Printed conditional source P/T text revision

`SourcePTWhileControllerControlsBasicLand(landType string, power, toughness int)
ContinuousEffect` creates one typed source P/T modification. Definition validation
requires a basic land subtype: Plains, Island, Swamp, Mountain or Forest.
`StaticAbility(effect).WithPTTextRevision()` enables checked installation through
`Game.RevisePrintedStaticPTAbility(source *Permanent, abilityID uuid.UUID,
changes ...TextChange) error`. A conditional declaration must contain exactly
one effect. Mixed or multiple conditional effects reject. Replacement cannot
change between the existing global context and the conditional source context.

RON opts in exactly `If(PlayerPassesFilter(You, ControlsA(IsLandType(type))),
[PermanentLayerEffect(ThisPermanent, [AdjustPT(power, toughness)])])`, with a
fixed basic subtype and fixed integer adjustments. Existing Unless, IfElse,
ControlsNo, nonbasic subtype, collection, host and multiple-modification forms
retain their existing execution without this opt-in. Malformed syntax and
unbound references remain compilation errors.

CR 611.3 and 613.4c use the existing `While`, `PlayerControls`, `You`,
`HasSubType`, `ModifyPT`, `This` and `Fixed` services. Control is read after
layer 2; land subtypes are read after layer 4. Opponent lands and phased lands
do not satisfy the condition. Typed effects clone their installed state.
CR 612.2 changes the land word in the retained semantic AST in order. Mana
costs and the other printed declarations keep their separate contracts.

The shared revision service keeps active identity, source visit, printed group
and registration position. It validates the complete rebuilt definition and
context before mutation. Siblings and clones keep independent changes. Phasing
keeps revisions; ability removal suspends execution and restoration keeps the
revised rule. Departure saves current P/T in LKI. Return uses the printed rule.
The caller still supplies layer-3 order, lifetime and original-text restoration.
This contract does not establish live Magical Hack execution.

Evidence: `TestRONTextConditionalSourcePTRecords` covers all eight Sedge Troll
and Kird Ape records, current controller, siblings, clones, phase cycles,
removal/restoration, LKI, return and paid regeneration. `Boundaries` covers
invalid declarations and unmarked contexts. `OrderedRevisions` covers foreign
IDs, successive changes, land control and layer-4 subtype changes. Native
`TestTextConditionalSourcePTDeclaration`, `Context`, `RegistrationOrder` cover
five legal subtypes, invalid values, mixed effects, both context transitions,
atomic rejection and unchanged registration position. Existing
`TestRONConditionalStaticController` covers normal casting, entry timing and
unpaid regeneration; `TestTextStaticPTRejects` retains common rejection evidence.


## Printed opponent-conditioned P/T text revision

Native constructors return `ContinuousEffect`:

- `SourcePTWhileOpponentControlsBasicLand(landType string, power, toughness int)`.
- `SourcePTWhileOpponentControlsNontokenColor(color core.Color, power, toughness int)`.
- `NamedCreaturesPTWhileOpponentControlsNontokenColor(name string, color core.Color, power, toughness int)`.

Use `StaticAbility(effect).WithPTTextRevision()` and the existing
`Game.RevisePrintedStaticPTAbility` service. Definition validation requires one
typed effect, a basic land subtype or one of the five colors, and a nonempty
name for the named-creature context. A revision must keep its condition kind,
source or collection recipient kind, and fixed recipient name. Invalid context
changes reject before mutation. Unconditional global rules and controller-land
source rules keep their existing contracts.

RON opts in `If(APlayerPassesFilter(Opponent, ControlsA(filter)), [body])`:
`filter` is `IsLandType(basic_type)` for a source recipient, or
`And([IsNonToken, IsColor(color)])` for either supported recipient. `body` is
`PermanentLayerEffect(ThisPermanent, [AdjustPT(integer, integer)])` or
`EachPermanentLayerEffect(And([IsCardtype(Creature),
IsNamed(NamedCard(nonempty_string))]), [AdjustPT(integer, integer)])`.
Both two-member conjunctions can use either order. Unless, IfElse, nonbasic
land types, color conditions without nontoken exclusion, general collections,
hosts and multiple adjustments keep existing execution without this opt-in.
Malformed declarations and unbound references remain compilation errors.

CR 604.2/7, 611.3 and 613.1b/d/e/g, 613.4c use existing `While`,
`OpponentControls`, `HasSubType`, `HasColorFilter`, `Not(IsToken)`, `ModifyPT`,
`ModifyEachPT`, `MatchingPermanents`, `IsCreature`, `Named` and `Fixed` services.
The source's current controller determines the opponent; neither owner nor
active player determines it. Conditions read current unphased permanents after
control, type and color layers. Named recipients can belong to either player
and can be tokens; nontoken exclusion applies only to the condition.
Membership stays live on each layer pass. No condition uses departure LKI.

CR 612.2 and 613.7 retain ordered semantic substitutions, fixed card names,
active declaration IDs and registration positions. Sibling declarations and
clones keep separate changes. Phasing keeps revisions; ability removal suspends
execution; restoration uses the revised rule. Departure saves current P/T in
LKI; CR 400.7 return uses printed rules. The caller still supplies layer-3
ordering, lifetime and original-text restoration. This bounded installation
contract does not establish complete Magical Hack or Sleight of Mind gameplay.

Evidence: `TestRONTextOpponentPTRecords` covers all three complete Water Wurm,
Beasts of Bogardan and Ivory Guardians records, ordered clone revisions,
current source/condition controller, phasing, removal/restoration, LKI, return,
printed costs and separate protection. `NamedRecipients` covers independent
sources, foreign IDs, both players, token recipients, token condition exclusion,
noncreature exclusion and later entries. `Layers` covers type/color order and
successive changes; `IndependentDeclarations` covers separate rule histories
and reversed condition order; `Boundaries` covers invalid and unmarked shapes.
Native `TestTextOpponentPTDeclaration`, `Context`, `RegistrationOrder` cover
legal and invalid values, mixed effects, all 30 context transitions with atomic
rejection, and identity/registration preservation in all three new contexts.
Existing `TestRONOpponentStatic*` retains casting, protection legality, failed
cost rollback and collection loading evidence; common revision rejection tests
remain in `TestTextStaticPTRejects`.


## Printed Aura-host basic-land P/T text revision

`CreaturesPTWhileAttachedToBasicLand(landType, creatureType string, power,
toughness int) ContinuousEffect` creates one typed global creature-subtype
P/T rule with a current Aura-host condition. The host must be Basic and have
the supplied basic land subtype. `StaticAbility(effect).WithPTTextRevision()`
opts in to `Game.RevisePrintedStaticPTAbility`. Definition validation requires
one effect, one of the five basic land types, and a nonempty creature subtype
that passes `ValidateSubtypeFamily(SubtypeCreature, ...)`. Revision must keep
this condition/recipient context and the fixed creature subtype. Transitions
from or to other P/T contexts reject before mutation.

RON opts in exactly:

```ron
If(PermanentPassesFilter(HostPermanent,
    And([IsSupertype(Basic), IsLandType(Mountain)])), [
    EachPermanentLayerEffect(And([IsCreatureType(Goblin),
        IsCardtype(Creature)]), [AdjustPT(0, 2)])
])
```

Both conjunctions permit either order; the land and creature subtype are fixed
symbols and adjustments are integers. `HostPermanent` requires Aura scope.
Unless, IfElse, missing Basic, nonbasic land subtypes, general collections and
multiple adjustments retain existing execution without revision opt-in. Invalid
syntax and unbound references remain compilation errors.

CR 604.2/4/7, 612.2–3 and 613.1d/f/g, 613.4c, 613.7a/e use the existing
`PermanentMatchesFilter`, `AttachedPermanent`, `While`, `MatchingPermanents`,
`ModifyEachPT`, `HasSubType`, `IsCreature` and fixed-value services. The Basic
filter uses the same supertype read as existing RON filters. Reattachment reads
the current host; an absent, phased or returned old host supplies no condition
and no LKI substitute. Earlier type layers determine matching land/creature
subtypes. Current recipients under either player, including tokens and later
entries, receive the adjustment. Control and ownership do not restrict them.

The shared checked revision service retains active IDs, declaration groups,
source visits and registration order. Reattachment still receives its normal
Aura timestamp; revision does not retimestamp it. Clones and sibling declarations
keep independent histories. Phasing keeps revisions; ability removal suspends
execution; restoration uses the revised rule. Recipient departure saves revised
P/T in LKI. Returned sources use printed rules. Mana costs and Goblin Shrine's
separate departure damage trigger keep their existing contracts. The caller
supplies layer-3 ordering, lifetime and original-text restoration. Full Magical
Hack execution remains separate.

Evidence: `TestRONTextHostPTRecords` maps complete DRK Goblin Caves and Goblin
Shrine records to ordered revisions, live hosts, Basic exclusion, both players,
noncreature exclusion, control, clones, phase cycles, removal/restoration,
earlier type layers, later/token recipients, departure damage and source return.
`VisitsAndLKI` covers recipient LKI, host departure/return and changing recipient
subtypes. `IndependentDeclarations` covers reversed conjunctions, foreign IDs,
sibling factories and independent declarations. `Boundaries` covers invalid and
unmarked shapes. Native `TestTextHostPTDeclaration`, `Context`,
`RegistrationOrder` cover five legal types, invalid types/subtypes, mixed
effects, all 22 new context/subtype transitions, atomic rejection and retained
registration positions. Existing `TestRONPermanentFilter*`,
`TestRONSourceLeaveShrine*` and `TestRONSourceLeaveCasting` retain normal casting,
Aura legality, failed-cost rollback, simultaneous damage, prevention, copied
triggers and old-source information evidence.

## Printed Aura ability-grant text layer

`StaticAbilityHolder.WithLayeredAbilityTextRevision() *StaticAbilityHolder` adds
layer-3 installation to the checked printed static ability contract. RON uses
this marker for unconditional `PermanentLayerEffect(HostPermanent,
[AddAbility(abilities)])`: keyword symbols, fixed landwalk, and Ward protection
with `SinglePermanent(ThisPermanent)` as the exact Aura exception. Mixed parts,
global rules and grants of activated or triggered abilities do not opt in.

`Game.AddPrintedStaticAbilityTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` accepts validated color
or basic-land word changes, and only `Indefinite` or `EndOfTurn` duration. It
checks the present face-up source incarnation, active eligible printed holder,
retained semantic builder, correspondence and complete registrations in an
isolated probe before registration. Invalid input cannot change live rules.

Each layer pass restores a separate baseline, then applies text changes in
timestamp order before layer-6 grants. Expiry recomputes surviving changes from
that baseline. Prior direct revision can supply the baseline; direct revision
cannot overwrite a currently layered result. Holder UUIDs, declaration groups,
effect positions, source context and grant timestamps stay intact. The donor
changes; granted recipient abilities remain separate (CR 612.2–3, 613.1c/f).

Ability removal suppresses the donor. Restoration uses the registered holder.
Phasing preserves the incarnation and delays baseline reset until a present
pass. Expiry during suppression or phasing restores the original grant on
return. Departure LKI retains the last grant; a new battlefield visit uses
printed rules. Clones keep separate baselines, registrations and controller
contexts, including clones made after expiry. Copy-layer declaration replacement,
whole-object text and spell text remain separate capabilities.

Evidence: `TestRONTextStaticAbilityLayerWards`, `Landwalk`, `Keywords`,
`Boundaries`, `IndependentDeclarations`, `DamageAndLKI`, `Records`, `Expiry`,
`SuppressedAndPhasedExpiry`, `PriorRevisionAndLandExpiry`, and
`InvalidInstallation`; native `TestTextStaticAbilityLayerDeclaration`, `Rejects`
and `RestoredCloneContext`. These tests cover twelve unique donor cards and all
61 records. Compilation gain is zero; full card gameplay is tracked separately.


## Printed global static-grant text layers

`StaticAbilityHolder.WithLayeredGrantTextRevision() *StaticAbilityHolder` opts
in a printed global grant for layer-3 installation. It includes the checked
`WithGrantTextRevision` contract: nonempty battlefield ability-layer effects,
with optional P/T modifications in layer 7c. RON uses this marker for unconditional
`EachPermanentLayerEffect(filter, parts)` on the basic static path, where each
part is fixed `AdjustPT(integer, integer)` or nonempty `AddAbility(abilities)`
and at least one grant is present. Grants accept keyword symbols and fixed
landwalk filters. Conditional, activated, triggered and BandsWithOthers grants
retain their separate contracts. Aura grants keep their separate layer marker.

`Game.AddPrintedStaticGrantTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` accepts validated color
and basic-land substitutions with `Indefinite` or `EndOfTurn` duration. It checks
the exact present face-up source visit, active opted-in printed declaration,
retained semantic builder, declaration group and complete registrations on an
isolated probe before registration. The replacement keeps the marker, count,
ordered layers, sublayers, durations and registration positions. Rejection does
not change the live declaration, baseline or effects.

The shared static text service restores a saved baseline on each pass, then
applies text effects in timestamp order in layer 3. Each grant selects current
subjects in layer 6 and keeps them through its optional layer-7c modification
(CR 613.6). Current control, color and other filter properties remain live.
Expiry recomputes surviving substitutions from the baseline; this can be a prior
direct revision. Direct revision cannot overwrite the layered result. Printed
copy information, independent declarations and granted recipient abilities stay
separate (CR 612.2–3). Clones keep separate effects, baselines and controllers,
including clones after expiry. Phasing retains the visit and checks expiry on
the next present pass. Removal suspends the grant; restoration keeps the saved
text. Departure LKI retains recipient results; a new source visit uses printed
rules. Full-object/spell text and copy-layer rebinding remain separate.

Evidence: `TestRONTextStaticGrantLayerRecords` covers nine unique names and all
24 complete RON records with recipient grants, P/T, control, repeated passes,
expiry and departure. `Lords`, `Membership`, `SuppressedExpiry`, `PriorRevision`,
`IndependentDeclarations`, `CastTiming` and `Boundaries` cover ordered changes,
clones, identity, LKI, evasion, phasing, removal/restoration, shared membership,
payment timing and invalid declarations. Native `TestTextStaticGrantLayer*`
covers definition validation, atomic rejection, registration order, expiry and
clone controller isolation. This is bounded donor lifecycle evidence; it does
not establish complete Magical Hack or Sleight of Mind execution.

## Printed P/T text layers

`StaticAbilityHolder.WithLayeredPTTextRevision() *StaticAbilityHolder` opts in
a printed P/T declaration. It includes `WithPTTextRevision`:
nonempty layer-7c modifications with `WhileOnBattlefield` duration. Typed
controller-, opponent- and host-conditioned P/T declarations can use this
marker when they satisfy their existing typed direct-revision contracts.
RON applies it to unconditional `EachPermanentLayerEffect(filter,
[AdjustPT(integer, integer)])` on the creature collection path. Existing
source exclusion, controller, color, subtype and property filters retain their
contracts. RON also opts in these complete conditional shapes:

- `If(PlayerPassesFilter(You, ControlsA(IsLandType(basic))),
  [PermanentLayerEffect(ThisPermanent, [AdjustPT(integer, integer)])])`.
- `If(APlayerPassesFilter(Opponent, ControlsA(IsLandType(basic))),
  [PermanentLayerEffect(ThisPermanent, [AdjustPT(integer, integer)])])`.
- The opponent condition can instead use `And([IsNonToken, IsColor(color)])`.
  Its body can modify the source or creatures with one fixed `NamedCard` name.
- An Aura condition can use `PermanentPassesFilter(HostPermanent,
  And([IsSupertype(Basic), IsLandType(basic)]))` with one global creature-subtype
  collection and one fixed `AdjustPT`.

Each typed conditional declaration contains exactly one effect. Replacement
keeps the condition kind, recipient kind, fixed name and creature subtype.
Only color and basic-land words change. Other conditional shapes are not
opted in. All existing direct APIs remain available before layered installation.

`Game.AddPrintedStaticPTTextEffect(source *Permanent, abilityID uuid.UUID,
duration core.Duration, changes ...TextChange) error` accepts validated color
and basic-land word changes with `Indefinite` or `EndOfTurn` duration. The exact
present face-up source visit must have an active opted-in printed declaration,
a retained semantic builder, declaration group and complete registrations.
An isolated probe checks the replacement before live installation. Replacement
preserves the marker, count, layers, sublayers, durations and registration order.
Rejection leaves live effects, declarations and baseline maps unchanged.

The shared static text-layer service restores its saved baseline before applying
ordered substitutions in layer 3 (CR 612.2, 613.1c). A prior direct revision can
be that baseline. Expiry recomputes surviving changes. Direct revision cannot
replace a layered result. Subjects are selected from current characteristics in
layer 7c, after control, type, color and ability layers (CR 613.1, 613.4c).
Source ability removal suspends the rule before P/T applies. Later entrants,
control changes, tap state and combat membership use existing live queries.
Sibling declarations and printed/copiable information remain separate.

Clones keep separate effects, baselines and holder/controller contexts, including
a clone created after expiry. Phasing retains the visit and checks expiry on
the next present pass. Removal/restoration retains the baseline, including expiry
while suppressed. Departure LKI keeps the final recipient P/T. A returned source
uses printed rules and cannot satisfy an earlier visit (CR 400.7).
Full-object/spell text and copy-layer rebinding remain separate contracts.
No new cost, replacement-order or simultaneous-operation service is introduced.

Evidence: `TestRONTextStaticPTLayerRecords` covers all 36 accepted complete
records across 12 names. `OrderedExpiry`, `IndependentDeclarations`,
`PriorRevisionAndCasting`, `LiveMembership`, `SiblingMana` and `Boundaries`
cover order, identity, expiry, phasing, suppression, LKI, clones, casting/payment,
current membership, declaration isolation and rejection. Native
`TestTextStaticPTLayer*` checks definition validation, atomic rejection,
registration positions, expiry and clone controller isolation. This is P/T
lifecycle evidence; it does not establish complete text-spell execution.

Conditional evidence: `TestRONTextConditionalPTLayerRecords` checks all 13
complete records across Kird Ape, Sedge Troll, Water Wurm, Beasts of Bogardan,
Ivory Guardians, Goblin Caves and Goblin Shrine. The tests cover ordered expiry,
clones, phasing, suppression, LKI, source return and separate abilities.
`LiveConditions`, `NamedRecipients`, `PriorRevisionAndVisits`,
`IndependentDeclarations` and `Boundaries` cover current control, earlier
type/color layers, token exclusions, host identity, prior baselines, independent
declarations and rejection. `TestTextConditionalPTLayer*` checks typed definition
validation and atomic rejection of changed condition or recipient contexts.


## Printed activated-action text layers

`mage.WithLayeredTextRevisionActivationState() ActionOption` includes the checked
activation-state contract. RON uses it on ordinary and modal `Activated` and
`ActivatedWithModifiers` actions. The conditional-mana path stays separate.

`(*mage.Game).AddPrintedActivatedTextEffect(source *Permanent,
abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error`
installs nonempty validated color/basic-land word changes with `EndOfTurn` or
`Indefinite` duration. Installation checks the exact present face-up source,
printed declaration, retained builder, layer marker and history shape on a clone
before it adds an effect. Rebuilds keep timing, limits, permissions and declaration
shape (costs, targets, modes, restrictions, captures and defined mana X).
Granted, absent, removed, foreign and phased declarations
reject without a live change. Each layer pass restores the saved baseline before
ordered changes. Expiry recomputes from that baseline. A prior direct revision
can supply the baseline; direct revision cannot replace a layered result.

Active IDs, limits and history lineage stay fixed. Fresh cost, target, modal,
choice and result declarations serve future activations. Pending activations and
copies keep prior text and bindings. Mana symbols do not change. Ability removal
stops activation. Restoration keeps current text and paid limits. Cleanup resets
limits even while the ability is removed or phased out. Phasing keeps the visit;
the next present layer pass restores expired text. Departure keeps final LKI;
return uses printed rules. Clones keep separate state and controller contexts.

Evidence: `TestRONTextActivationLayerTargets`, `Limits`, `History`, `Choices`,
`Boosts`, `OrderingLifetime`, `Boundaries`, `BaselineControllerAndLKI`,
`TimingAndResults`, `Modal` and `LandWords`; native `TestTextActivationLayerDeclaration`
and `Rejects`. Whole-object dispatch and spell continuation stay separate.

### Ordinary pending spell text

RON's ordinary `SpellActions(body)` path without an additional casting cost
marks `WithSpellTextContinuation`. `Game.RevisePrintedSpellText` rebuilds that
pending instant or sorcery's semantic rules while retaining announced targets,
X and cast facts. New target declarations use the same saved groups; revised
restrictions control resolution legality. Resolution choices and results use
fresh per-instance bindings. Copies retain current target declarations, and
clones keep separate revisions. The physical card retains printed text.

Typed permanent, player, opponent, any-damage and spell target contexts are
supported, including counted permanent groups. Permanent spells, other target
kinds and other captured casting results need separate services. Saved sacrifice
mana values use the checked contract below. Modal declarations
use the checked contract below.
This checked engine prerequisite does not lower the complete `CreateSpellEffect`
or whole-object permanent text operations. Magical Hack and Sleight of Mind
remain rejected as complete records.

### Modal pending spell text

RON marks the complete `SpellActions(Modal_ChooseOne([body, body, ...]))` path
with `(*ModalSpellAbility).WithTextContinuation() *ModalSpellAbility`.
`Game.RevisePrintedSpellText` rebuilds every mode with independent bindings and
checks each mode's declaration context. The announced mode and targets stay
fixed; only that mode's fresh effects resolve. Typed target support matches the
ordinary continuation contract. Revised legality, phasing and exact zone visits
use existing target services. Results produced at resolution use fresh frames.

Copies keep the selected mode and current semantic definition. New targets use
that definition's mode declarations. Clones and subsequent revisions of either
stack object stay separate. Physical cards keep their printed rules on departure.
Additional/alternate costs, casting-result transfer, casting windows, permanent
spells and union-target declarations require separate contracts. This marker
does not lower complete Magical Hack or Sleight of Mind records.

Evidence: `TestRONTextModalSpellContinuation*` covers both modes of Blue Elemental
Blast, Red Elemental Blast and Healing Salve in all six reprints, plus revised
collections, partial legality, copies, clones and fresh mode-local results.
`TestTextModalSpellContinuation*` covers declaration validation and atomic
rejection, including incompatible unchosen modes.

### Saved sacrifice mana values in pending spell text

`mage.WithSpellSacrificeManaValueContinuation(paid *mage.NumberBinding)
ActionOption` marks an ordinary instant/sorcery with exactly one additional
matching-sacrifice cost that produces that binding's mana value. RON opts in
`CastEffect(AdditionalCastingCost(SacrificeAPermanent(filter)))` with ordinary
`SpellActions(body)` that reads symbol `ManaValueOfTheSacrificedPermanent`.
Arithmetic, restricted resolving mana and resolution-result bodies retain their
existing lowering. Each factory supplies one private cost/body binding.

`Game.RevisePrintedSpellText` checks the marker, producer and exact saved map,
then transfers the number to the rebuilt declaration's fresh binding. Cost kind,
count, source permission and label stay fixed. Semantic filters rebuild, but
payment is not repeated. Targets, X, cast facts and physical printed rules stay
fixed. Copies keep the original payment with their own controller; later copy
revisions and game clones have separate maps. Results created at resolution
remain in fresh frames. Zero is a present saved value. Absent, foreign or extra
numbers, other capture types, duplicate/nil markers, alternative/multiple costs
and changed producer contexts reject atomically. Toughness captures, modal
additional costs and permanent entry remain separate contracts.

Evidence: `TestRONTextSacrificeSpellContinuationRecords` covers all six Sacrifice
records and Metamorphosis, including restricted mana, zero values, returned and
phased payment cards, repeated revisions, copies and clones. `FreshResults`,
`Legality` and `Instances` check result frames, changed cost/target filters, exact
visits and independent payments. Native `TestTextSacrificeSpellContinuation*`
checks definition validation, exact binding transfer and atomic rejection.
Complete Magical Hack and Sleight of Mind remain rejected.


### Text builders for copy exceptions

The existing entry-copy lowering for `CopyEffects([AddCardtypes([...])])`
and `CopyEffects([KeepColor])`, including their ordered combination, now keeps
those exceptions when copied rules are rebuilt for word changes. No RON
spelling or scope changes. `mage.CopiableValues.WithModifications` composes the
saved exceptions with the source's checked semantic text builder.

`BuildTextChangedDefinition` and `BuildTextDefinitionRevision` keep added types,
retained colors (including colorless) and omitted color-defining abilities.
Fresh target and result bindings still come from the RON definition plan.
Later copies retain copy exceptions and exclude ordinary text changes.
Unknown semantic word roles and invalid builders remain errors. Existing
per-declaration text-layer services supply lifetime and active rule identity.
Whole-object text dispatch, permanent-spell continuation and added copy abilities
remain separate contracts; Magical Hack, Sleight of Mind and Vesuvan Doppelganger
still reject as complete records.

Evidence: `TestRONCopyTextRebindingRecords` casts each of the six Copy Artifact
reprints, changes both copied Gauntlet of Might rules, and checks all six Clone
reprints as later copies. `RetainedColors` exercises combined KeepColor and
AddCardtypes with fresh activated target bindings. Existing copy declaration
rejection tests retain their exact shape checks.


### Land-type Aura spell text entry continuation

`mage.WithLandTypeAuraSpellTextContinuation() CardOption` marks one ordinary
Aura with one permanent enchant target and one attached basic-land-type rule.
A fixed rule permits no entry choice. A chosen rule requires exactly one
`ChooseBasicLandTypeAsEnters` declaration. Duplicate markers, additional or
alternative costs, extra types/subtypes, keywords, modes, unrelated abilities,
missing/nil targets and mismatched choices fail definition validation.

RON marks complete `EnchantPermanent(filter)` plus
`PermanentLayerEffect(HostPermanent, [SetLandType(basicType)])`, or the chosen
variant with `AsPermanentEnters(ThisPermanent, [ChooseABasicLandType])` and
`SetLandTypeVariable(TheChosenLandType)`. Filters keep their existing validated
permanent/enchant roles. No new RON spelling or card-name dispatch is used.

`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
checks the retained marker, builder, ability groups and target context before
atomic replacement. Ordered semantic word changes preserve the announced target,
exact visit, X, controller, owner, payment and cast facts. Revised enchant filters
control resolution legality and copy retargeting. Entry choices happen only when
an eligible spell resolves; saved choices are not printed words (CR 612.2).

Fresh revised definitions enter through the shared battlefield entry service
(CR 400.7a, 608.3b/c). The physical card keeps its ID, owner and printed departure
rules. Ordinary changes do not enter its copiable values. Spell copies enter as
tokens (CR 608.3f, 707.10f), retain current revised rules, and choose independently
at entry (CR 707.10). Clones keep separate revisions and source contexts.
Phasing keeps text, attachment and choices without repeating entry. Later
per-declaration text layers use the continued baseline, including after expiry.
Layer 6 ability removal cannot undo an earlier layer 4 land-type change; restoring
the ability keeps its continued text (CR 613.1d/f). Departure saves revised LKI
and returns the physical printed definition (CR 400.7).

This is a bounded entry prerequisite. Copy-entry, whole-object permanent word
changes, other Aura rules, casting-result transfer and general spell text
lifetime remain separate. It does not lower complete Magical Hack or Sleight
of Mind. No replacement priority, cost transaction or batch operation changes.

Evidence: `TestRONTextLandAuraSpellRecords` covers Evil Presence and Phantasmal
Terrain in all six reprints (2 names, 12 records). `Legality`, `Retarget`,
`CopiesAndLayers`, `ChoiceCopies` and `Boundaries` cover ordered target filters,
exact visits, phasing, copy-controller choices, token copies, separate clones,
layer expiry/removal/restoration, copiable values and invalid/unsupported forms.
`TestTextLandAuraSpellDeclaration` and `RejectsAtomic` check native declaration
validation and rejection without mutation. Compilation gain is zero.


### Intrinsic creature spell text entry continuation

`WithIntrinsicCreatureSpellTextContinuation() CardOption` permits the checked
`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
path for one creature-only definition. RON selects a nonempty intrinsic context
with top-level `Protection(FromColor(Colors([color, ...])))` and/or
`Landwalk(IsLandType(basicType))`, plus ordinary keyword symbols. The color list
must be nonempty. Landwalk must use a fixed basic land type. Each declaration
uses its existing definition group and layer-3 marker. Extra card types,
targets, modes, costs, entry operations and other abilities require separate
contracts. Invalid markers and malformed declarations fail definition validation.

Ordered changes build fresh rules without changing the physical card, mana,
announced X, cast facts, controller or owner. Declaration correspondence keeps
protection and landwalk kinds and independent keyword contributions. Invalid
words, empty changes, changed contexts and foreign saved casting results reject
before any stack update. Entry uses the shared permanent-spell and entry paths
(CR 400.7a, 608.3a/f). Spell copies enter as tokens with their own text; game
clones keep separate changes. Ordinary layer-5 spell color remains independent.

Protection uses existing target, damage, blocking and attachment services
(CR 702.16); landwalk uses existing combat services (CR 702.14). Later text
layers use the continued definition as their baseline. Ability removal and
restoration, phasing and departure retain the existing lifecycle. Departure LKI
keeps final rules; the physical card returns with printed rules (CR 400.7).

A private immutable entry-characteristic snapshot lets
`Game.CaptureCopiableValues(source *Permanent) (*CopiableValues, error)` check
continued landwalk seeds without accepting later untracked mutations. The
returned values keep the printed definition (CR 707.2). Permanent-spell token
wrappers use the same checked provenance. No new payment, event, replacement
ordering or simultaneous-operation service is introduced.

Evidence: `TestRONTextIntrinsicSpellRecords` covers all 34 complete records of
14 unique names; `Copies`, `Layers`, `ProtectionGameplay` and `Boundaries` cover
spell-copy token entry/provenance, ordered substitutions, separate duplicate
landwalk contributions, protection color sets, later layer expiry, phasing,
ability restoration, grants, damage batches and illegal targeting/attachment.
Native `TestTextIntrinsicSpellDeclaration`, `RejectsAtomic` and
`CopyProvenance` cover marker validation, changed contexts, countered visits,
foreign casting results and untracked mutations. Copy-entry and other permanent
rules, whole-object text dispatch, Magical Hack and Sleight of Mind remain open.

### Copy-entry spell text continuation

RON marks a permanent whose only rule is
`AsPermanentEnters(ThisPermanent, [MayActions([EnterAsACopyOfAPermanent(filter, modifications)])])`.
The existing compiler validates the permanent filter and `NoCopyEffects`, or
nonempty `CopyEffects` containing `AddCardtypes` and/or `KeepColor`. Added copy
abilities remain unsupported. The inventory supplies typed permanent-filter,
copy-modification and card-type roles for these arguments.

The marker lowers to `mage.WithCopyEntrySpellTextContinuation()`. The existing
`Game.RevisePrintedSpellText` checks the exact stack visit and declaration
correspondence before replacing its retained rules. Targets, casting results,
extra abilities, keywords, modes and extra casting costs are outside this
context. Revision retains payment and physical identity without making choices.

At resolution, the entry replacement pipeline carries an immutable list of
ordered substitutions through donor acquisition. Fresh donor text is built by
`mage.BuildEntryCopyWithText(original, values, changes...)`. Each acquisition
starts from its saved copy values, including copy exceptions; the continuing
changes are applied once to that new definition before acquired entry operations
and ETB detection. A donor requires a checked text builder and matching
declaration groups. Invalid builders return errors. Face-down copy values have
no words or rules to change. Nested copy choices keep the same continuing list
and use the existing replacement identities and priorities.

Continued words do not enter copiable values. Later text layers use the continued
baseline, and expiry restores it. Clones, spell-copy token entry, phasing, LKI
and physical printed departure retain their separate lifecycle contracts.

Evidence: `TestRONTextCopySpellRecords` casts every Clone and Copy Artifact
reprint (12 records), copies complete Bog Wraith and Gauntlet of Might rules,
and checks payment, entry, copied mana events, landwalk, exceptions, copy values,
phasing, LKI and printed recasts. Copies, DonorBindings, Layers, EntryShapes,
RetainedColors, FaceDownExceptions and FilterRevision cover independent token/clone state, fresh
cost/target/ETB bindings, nested acquisition, absent and returned donors,
face-down values, later layers and typed selection. Boundaries and native
`TestTextCopySpell*` tests cover declaration and atomic mutation failures.
Magical Hack, Sleight of Mind and Vesuvan Doppelganger remain unsupported.

### Global static permanent-spell text continuation

RON marks a nonland, non-Aura permanent whose sole rule is
`EachPermanentLayerEffect(filter, [AdjustPT(integer, integer)])`, where the
filter requires creatures, or `EachPermanentLayerEffect(filter,
[SetLandType(basicLandType)])`. Existing static lowering validates each filter
and operation. Controller and source-exclusion selectors remain live. Extra
rules, keywords, targets, casting costs, conditional rules and sticky rules
remain outside this context.

`mage.WithGlobalStaticSpellTextContinuation() CardOption` declares the engine
contract. For these shapes, validation requires one marked global static effect, no
other abilities or intrinsic keywords, and no modes or casting costs/targets.
The existing `Game.RevisePrintedSpellText(stackID, changes...)` checks the exact
stack visit, fresh declaration correspondence, static context and subtype
context before replacing retained rules. It rejects resolving bodies and
captured casting results. Missing builders/groups, invalid words, countered
visits and changed contexts fail without mutation.

Entry rebuilds independent continued rules through the existing text builder.
It retains physical identity, owner, token status and printed departure rules.
Existing static registration, type dependencies, intrinsic land mana and P/T
layers execute the result. Later layer 3 effects start from the continued
baseline; expiry restores it. Layer 6 removal retains the established ordering:
P/T rules stop, while an earlier land-type rule keeps its layer 4 result.
Adding a Land copy exception clears this spell eligibility marker on the copied
prototype while retaining its rules and checked text builder. Original
definitions keep their marker. Phasing keeps the visit and continued rules. Control and subject membership
remain live. Copies of permanent spells enter as independent tokens; game
clones keep separate stack revisions and layers. Ordinary text changes stay
outside copiable values. Departure LKI retains continued rules, and physical
recasts use printed rules (CR 400.7a, 608.3a/e/f, 612.1–3, 613.1c/d/f/g,
707.2–3, 707.10f).

Evidence: `TestRONTextGlobalStaticSpellRecords` covers all 29 complete records
of Blood Moon, Bad Moon, Castle, Crusade, Orcish Oriflamme, Jacques le Vert,
Kobold Taskmaster, Mightstone and Weakstone. `PTLifecycle`, `LandLifecycle`,
`Control`, `CopyExceptions`, `Boundaries` and `Inventory` cover copies, clones, printed copy
values, later text layers, phasing, LKI, printed recasts, live control, invalid
syntax and exact record mappings. Native `TestTextGlobalStaticSpellDeclaration`
and `RejectsAtomic` cover invalid markers, builders, groups and stack state.
Gauntlet of Might, Conversion and Sunken City use the mixed continuation
contract below. Titania's Song still fails compilation at its sticky rule. This adds
no full-card compiler unlocks. Whole-object word dispatch, remaining spell
contexts and both word-changing spells require further capabilities.


### Mixed global-static/event permanent-spell text continuation

`mage.WithMixedStaticEventSpellTextContinuation() CardOption` permits exactly
one marked global fixed P/T or basic-land-type rule paired with either a marked
fixed controller-recipient land mana bonus or a typed controller-upkeep fixed
mana payment with unpaid self-sacrifice. Either declaration order is permitted.
The permanent must be nonland and non-Aura, with no other rules, intrinsic
keywords, modes, casting costs or casting targets.

RON recognizes the existing global `EachPermanentLayerEffect` shape together
with `TriggerA(WhenAPermanentIsTappedForMana(IsLandType(subtype)), body)` or
`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),
ActionList([MayCost(PayMana(symbols)), Unless(CostWasPaid,
[SacrificePermanent(ThisPermanent)])]))`. Existing adapters validate filters,
bonus recipient/amount and lexical payment scope. No card-name dispatch occurs.

`Game.RevisePrintedSpellText(stackID, changes...)` checks both retained
printed groups, their order, static context, bonus recipient/production context,
upkeep event capture presence, unchanged fixed payment and subtype context.
Foreign casting targets, results and bodies, invalid words, missing builders or
groups and departed stack visits reject atomically. Independent entry builds
retain physical identity, owner, controller, token status and printed departure
rules. The shared static layers, mana query/production and transactional payment
services execute both rules. Mana symbols remain unchanged. Upkeep uses the
pending trigger controller and exact source incarnation; returned cards cannot
supply that sacrifice. Copies of triggers use their own controller.

Spell copies enter as independent tokens; cloned stack revisions and payment
resources stay separate. Later text layers start from the continued baseline.
Phasing and layer-6 removal/restoration use existing rule lifecycles. Ordinary
word changes stay outside copiable values; LKI keeps continued rules and physical
recasts use printed rules. A Land copy exception clears the spell marker while
retaining both rules and the text builder. CR 400.7a, 605.1b/4a, 608.3a/e,
612.1–3, 613.1c/d/f/g, 707.2 and 707.10f supply the applicable rules.

Evidence: `TestRONTextMixedStaticEventSpellRecords` and `Inventory` map all
12 complete records: Gauntlet of Might (5), Conversion (6), Sunken City (1).
`Payments`, `Lifecycle`, `CopyExceptions` and `Boundaries` cover both rules,
recipient and timing, payment/decline/rollback, pending control and return,
trigger copies, spell copies/clones, later layers, phased expiry, ability
removal, copiable values, LKI, printed recasts and invalid contexts. Native
`TestTextMixedStaticEventSpellDeclaration`, `RejectsAtomic`,
`UpkeepDeclaration` and `UpkeepRejectsAtomic` check marked shapes and atomic
context rejection. This adds continuation coverage, with no compiler unlocks.
Other mixed declarations and whole-object word dispatch remain separate.

## Intrinsic creature and cast-event spell text continuation

`mage.WithIntrinsicCastEventSpellTextContinuation() mage.CardOption` permits
one creature definition with marked intrinsic color protection/basic landwalk
and one marked typed battlefield spell-cast trigger. Ordinary keywords remain.
RON selects intrinsic rules plus one `TriggerA(WhenAPlayerCastsASpell(AnyPlayer
or Opponent, filter), body)`. Existing lowering checks filters, targets, bodies
and lexical bindings. Rule order and separate intrinsic contributions remain.
Casting targets, additional/alternate costs, modes, Aura/extra types, other
rules and conflicting continuation markers are outside this contract.

`Game.RevisePrintedSpellText` retains declaration groups, trigger event context,
target shapes, intrinsic contribution counts, subtype context and ordinary
keywords. Invalid changes reject before stack mutation. Mana symbols retain
printed values. Entry builds fresh rules with physical identity and printed
departure rules. Copies enter as tokens; game clones keep separate rules.
Later declaration text layers restore the continued baseline after expiry.
Phasing keeps the visit. Pending triggers retain detection text and controller.
LKI keeps final rules; a recast uses printed rules. Ordinary word changes stay
outside copiable values. Added copy types retain rules and clear this marker.

Evidence: `TestRONTextIntrinsicCastSpellRecords`, `Lifecycle`, `Inventory`,
`EventBindingsAndTargets`, `IntrinsicGroups`, `CopyExceptions`, `Boundaries`;
`TestTextIntrinsicCastSpellDeclaration` and `RejectsAtomic`.
Whole-object text dispatch and word-changing spell consumers remain separate.


## Conditional source P/T spell text continuation

`WithGlobalStaticSpellTextContinuation() CardOption` also accepts one marked
`StaticAbility(effect).WithLayeredPTTextRevision()` whose sole effect is
`SourcePTWhileControllerControlsBasicLand` or
`SourcePTWhileOpponentControlsBasicLand`. `Game.RevisePrintedSpellText(stackID
uuid.UUID, changes ...TextChange) error` retains this conditional rule through
fresh permanent entry. The card must satisfy the existing single-static
permanent contract: no Aura, land, casting targets, extra costs, keywords,
modes or sibling abilities. Color-conditioned, named-recipient, host and
mixed static/event contexts remain outside this extension.

RON selects one complete rule:
`If(PlayerPassesFilter(You, ControlsA(IsLandType(type))),
[PermanentLayerEffect(ThisPermanent, [AdjustPT(power, toughness)])])`, or
`If(APlayerPassesFilter(Opponent, ControlsA(IsLandType(type))),
[PermanentLayerEffect(ThisPermanent, [AdjustPT(power, toughness)])])`.
The subtype is fixed and basic; adjustments are fixed integers. Existing typed
lowering supplies the effects. Native revision requires unchanged condition
side, recipient context, subtypes and paired declaration identity. Invalid
builders and context changes reject before stack mutation. The mixed-static
selector continues to accept only its previous global contexts.

CR 400.7a carries changed text through entry. CR 611.3 and 613.4c use live
control and land types in the existing layer service. CR 612.1–3 keeps mana
symbols and names separate. Copies enter as tokens; clones keep separate
changes. Later text layers restore the continued baseline on expiry; phasing
keeps it, and ability removal suspends it. CR 707.2 excludes ordinary word
changes from copy values. Departure keeps final LKI and restores the physical
card's printed rules for a later cast. No new cost, replacement or simultaneous
operation service is introduced.

Evidence: `TestRONTextConditionalPTSpellRecords` covers both Kird Ape records
and Water Wurm, payment/timing/identity, old and new conditions, controller
changes, copied spells, clone isolation, text-layer expiry while phased,
ability removal, copiable values, LKI and printed recasts. `Boundaries` and
`MixedBoundary` cover malformed declarations, Sedge Troll, Beasts of Bogardan,
Ivory Guardians, Goblin Caves, Goblin Shrine and synthetic conditional/upkeep
pairs. Native `TestTextConditionalPTSpellDeclaration` and
`RejectsContextAtomic` cover allowed contexts, invalid types, and condition,
recipient, opt-in and sibling changes. This is a prerequisite for whole-object
text dispatch; it does not execute Magical Hack or Sleight of Mind.


## Protection and conditional opponent-color P/T spell continuation

`WithProtectionPTSpellTextContinuation() CardOption` marks a creature with
exactly two printed declarations: nonempty color protection, marked with
`WithLayeredColorTextRevision()`, and one fixed P/T rule, marked with
`WithLayeredPTTextRevision()`. The P/T service is
`SourcePTWhileOpponentControlsNontokenColor(color, power, toughness)` or
`NamedCreaturesPTWhileOpponentControlsNontokenColor(name, color, power, toughness)`.
The condition reads the current controller's opponent. Named recipients include
matching creatures controlled by either player. Tokens cannot satisfy the
condition. Other abilities, keywords, card types, Auras, casting targets,
additional/alternate costs and conflicting continuation markers are excluded.

RON selects `Protection(FromColor(Colors([color, ...])))` beside
`If(APlayerPassesFilter(Opponent, ControlsA(And([IsNonToken, IsColor(color)]))),
[PermanentLayerEffect(ThisPermanent, [AdjustPT(power, toughness)])])`, or the
same condition with `EachPermanentLayerEffect(And([IsCardtype(Creature),
IsNamed(NamedCard(name))]), [AdjustPT(power, toughness)])`. Adjustments are
integers; colors are concrete. Existing protection and conditional P/T lowering
build separate declarations and groups. Valid broader cards still compile
without this marker; unsupported declarations remain errors.

`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
checks both old and fresh contexts before replacing the exact stack visit.
Protection cardinality, P/T condition/recipient context, named recipient,
subtypes, and specific sibling declaration pairing must stay fixed. Both rules
have static ability type, so pairing checks their concrete sibling identity.
Invalid changes, missing builders/groups and foreign announcement state reject
atomically. Mana symbols and names retain their printed values (CR 612.1–3).

The shared entry builder keeps physical identity, owner and printed departure
rules (CR 400.7a). Copies enter as tokens; clones keep separate changes. Live
control, condition colors, token status and recipients use existing layer
services. Later declaration text layers restore the continued baseline on expiry;
phasing keeps the visit. Ability removal suppresses both rules and restoration
uses their continued text. Departure keeps final LKI; recasts use printed rules.
Ordinary word changes stay outside copiable values (CR 707.2). Added copy types
clear the creature-only marker while retaining independently rebuilt rules.
No payment, damage, replacement-order or simultaneous-operation service changes.

Evidence: `TestTextProtectionPTSpellDeclaration` and `RejectsAtomic`;
`TestRONTextProtectionPTSpellRecords`, `Lifecycle`, `Copies`, `Inventory`,
`Boundaries` and `CopyExceptions`. Complete LEG records map Beasts of Bogardan
and Ivory Guardians to these tests. This shared prerequisite adds two names of
continuation coverage with zero collection acceptance gain. Complete battlefield
dispatch uses the protection/P/T object text contract below. Complete Sleight of
Mind execution remains separate work.

## P/T static and activated spell text continuation

`WithStaticActivationSpellTextContinuation() CardOption` marks a creature with
exactly two printed declarations, in either order: one layered P/T static rule
and one layered ordinary activated action. The static rule is an unconditional
global P/T adjustment, or a source P/T adjustment while its controller or
opponent controls a specified basic land type. The action has no modes,
activation history, source-counter X, captures, state conditions or opaque
restrictions. Other keywords, types, Auras, casting targets, additional and
alternate casting costs, and conflicting continuation markers are excluded.

RON selects `EachPermanentLayerEffect(creature_filter, [AdjustPT(p, t)])` or
`If(PlayerPassesFilter(You, ControlsA(IsLandType(basic))),
[PermanentLayerEffect(ThisPermanent, [AdjustPT(p, t)])])`, including the existing
opponent condition spelling, beside `Activated(cost, body)`. Adjustments are
fixed integers. Existing action lists and ordinary targeted bodies keep their
established typed lowering. `ActivatedWithModifiers` and modal bodies are not
selected. Broader valid records retain ordinary compilation.

`Game.RevisePrintedSpellText` validates both prior and fresh definitions,
P/T condition and recipient context, subtypes, action timing, cost cardinality
and target shapes, body cardinality and exact sibling declaration groups. Invalid changes
reject without replacing the stack visit. Fresh entry retains physical identity,
owner and printed departure rules, with independent activation bindings. Mana
symbols stay fixed. Copies enter as tokens; game clones keep separate rules.
Later declaration text layers restore this entry baseline. Phasing retains it;
ability removal stops both declarations. LKI retains final rules and P/T.
Returned physical cards use printed rules. Ordinary text changes do not enter
copiable values. Added copy types clear the creature-only marker.

Evidence: `TestTextStaticActivationSpellDeclaration`, `RejectsAtomic` and
`TestRONTextStaticActivationSpellRecords`, `Inventory`, `Boundaries`,
`Lifecycle`, `Copies`, `BothWords`, `ThrullLifecycle`, `CopyExceptions`.
Sedge Troll maps six complete records to regeneration and conditional P/T
coverage; Thrull Champion maps its complete FEM record to global Thrull P/T,
targeted control and source-control duration coverage. Complete battlefield
dispatch uses the static P/T and activated object text contract below.

## RON intrinsic creature object-text actions

Resolving `CreatePermanentLayerEffect(permanent_ref, [substitution, ...])` accepts
a nonempty list containing only
`ReplaceColorWordVariableWithNewColorWordVariable(from, to)` and/or
`ReplaceLandTypeVariableWithNewLandTypeVariable(from, to)`. Each operand must have
a matching earlier word-pair choice in its lexical scope. Mixed ordinary layer
operations, missing choices, wrong operand roles and equal references reject at
compilation. The compiler builds fresh typed references for each rule instance.
This form has indefinite duration.

The action calls `mage.ChangePrintedIntrinsicCreatureText`, which validates the
complete target context before installation. Supported targets have only ordinary
intrinsic keywords, marked color protection and/or basic landwalk, through the
existing intrinsic creature continuation contract. The engine changes every
matching printed declaration, preserves granted abilities, and rejects other
contexts atomically. Native callers can use
`Game.AddPrintedIntrinsicCreatureTextEffect` with Indefinite or EndOfTurn.
The established declaration-level services supply layer order, restoration,
cloning, phasing and exact-incarnation behavior.

This supports a complete bounded target context, not the full consumers Magical
Hack or Sleight of Mind. Their spell-target `CreateSpellEffect` branches, other
permanent rule contexts and live-copy rebinding remain unsupported.

## Complete global static object text dispatch

`Game.AddPrintedGlobalStaticTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` accepts one complete marked global P/T or
basic-land-type definition. `ChangePrintedPermanentText(subject PermanentRef,
duration core.Duration, changes ...TextChangeRef) ResolvingEffect` reads saved
words at resolution and dispatches to this contract or the existing complete
intrinsic creature contract. A missing subject does nothing.

RON lowers resolving `CreatePermanentLayerEffect(permanentRef, [substitution,
...])` with preceding typed word-pair choices through this dispatcher. Scope,
spelling and fresh bindings stay in RON. Complete Magical Hack and Sleight of
Mind still reject their unsupported spell branch; extracted permanent branches
are test fixtures only.

The source must be the present face-up incarnation. The complete retained and
rebuilt definitions must have one marked global static rule, matching declaration
groups, subtypes and context, without other rules, keywords, casting costs or
targets. Conditional source P/T, grants, attached rules and mixed contexts reject.
The shared declaration service validates runtime registrations and words before
installation. Indefinite and EndOfTurn durations are supported. Invalid calls
install no effects. Layer 3 baselines, timestamp order, live recipients/control,
layer 4 land types/intrinsic mana, layer 6 removal, layer 7c P/T, clone isolation,
phasing, copiable values, LKI and printed return retain their existing contracts
(CR 400.7, 612.1–3, 613.1c/d/f/g, 707.2).

Evidence: `TestRONTextGlobalStaticObjectRecords` runs the real permanent consumer
branches against all 29 complete records of Blood Moon, Bad Moon, Castle,
Crusade, Orcish Oriflamme, Jacques le Vert, Kobold Taskmaster, Mightstone and
Weakstone. `PTLifecycle`, `LandLifecycle`, `Control`, `PendingIsolation` and
`Atomic` cover continued baselines, expiry, removal, clones, identity and scope.
Existing `TestRONTextIntrinsicObjectDeclarations` covers invalid RON roles,
producers and mixed operations. Native `TestTextGlobalStaticObjectRejectsAtomic`
and `BindingDeclarations` cover complete-definition preflight and invalid bindings.

## Complete global grant object text dispatch

`Game.AddPrintedGlobalGrantTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` checks a complete definition with one or more
marked global grants and optional fixed intrinsic keywords. Each holder must
use `WithLayeredGrantTextRevision`. Grants can supply keywords, fixed basic
landwalk, paired fixed P/T modifications, or an untargeted activated ability.
Intrinsic protection/landwalk, attached rules, other sibling rules, casting
costs/targets and live copy rebinding require separate whole-object contracts.

RON marks its existing complete global activated-grant shape:
`EachPermanentLayerEffect(filter, [AddAbility([Activated(cost, ActionList(...))])])`.
The existing cost and action adapters still reject X, targeting, modifiers and
unbound donor/event/host references. The recipient owns the activation scope.
Resolving `CreatePermanentLayerEffect(permanentRef, [wordSubstitution, ...])`
uses `ChangePrintedPermanentText` to dispatch to this complete grant contract.
Typed pair choices, spelling and lexical scopes remain in RON.

The API requires the exact present face-up incarnation, saved builder and
matching declaration groups, fixed keywords and subtypes. It validates every
holder and runtime registration in an isolated probe before installing any
layer. Invalid words, missing declarations or registrations, changed grant
forms and unsupported durations leave the live game unchanged. Supported
durations are `Indefinite` and `EndOfTurn`.

Each donor rule retains its own baseline and applies ordered substitutions in
layer 3. Its recipients stay live in layer 6, with paired P/T in layer 7c.
Activated-grant effect identities survive rebuilds, repeated passes and expiry,
so recipient activation limits keep their state. Pending activations retain
their original recipient incarnation, controller and body. Mana symbols remain
fixed. Granted words on the target stay outside printed text changes (CR 612.3).
Phasing retains the visit; removal suspends the donor; clones isolate text and
control. Expiry recomputes remaining substitutions. Recipient LKI retains final
rules. Physical return and ordinary copy values use printed rules (CR 400.7,
612.1–3, 613.1c/f/g, 707.2). No payment or replacement service was added.

Evidence: `TestRONTextGlobalGrantObjectRecords` exercises the actual permanent
branches of Magical Hack/Sleight of Mind against all 24 complete records of
Goblin King, Lord of Atlantis, Zombie Master, Hidden Path, Fortified Area,
Kobold Drill Sergeant, Kobold Overlord, Concordant Crossroads and Goblin War
Drums. `ZombieLifecycle`, `Control`, `PendingBindings`, `ActivatedBody` and `Boundaries`
cover sibling filters, fixed payments, changed and pending bodies, ordered expiry,
grants, clones, returned targets,
removal, phasing, LKI and copy values. Native `TestTextGlobalGrantObjectAtomic`
checks all-or-nothing validation, including a missing second declaration or
registration. `ActivationIdentity` checks recipient limits and pending bodies.
Existing RON declaration tests retain producer/role/nested-binding rejection.

The extracted permanent consumer branches are explicit test fixtures. Complete
Magical Hack and Sleight of Mind still reject `CreateSpellEffect`; this contract
adds target interactions without claiming complete consumer gameplay or any
new collection acceptance. The existing intrinsic and global P/T/land dispatch
APIs retain their contracts.


## Complete ordinary activated object text

`Game.AddPrintedActivatedObjectTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` changes every printed ordinary activated declaration
on one exact face-up battlefield visit. `ChangePrintedPermanentText(subject
PermanentRef, duration core.Duration, changes ...TextChangeRef) ResolvingEffect`
now dispatches this context as well as its existing intrinsic and global contexts.

A complete definition must contain one or more `ActionActivated` declarations
marked with `WithLayeredTextRevisionActivationState`, optionally with fixed
intrinsic keywords. Land, Aura, instant/sorcery, mixed static/event definitions,
casting targets/costs, unmarked/granted declarations and live copy rebinding are
outside this context. Duration is `Indefinite` or `EndOfTurn`. A removed, phased,
returned or face-down source cannot receive a new installation.

Preflight checks the complete retained and rebuilt definitions, printed groups,
fixed keywords, subtypes, activation shapes and all active runtime declarations.
An isolated game probe runs the existing declaration-level service for each rule.
Only after every check passes are the text layers installed and recomputed.
A missing second runtime declaration leaves no partial first-declaration change.

Each declaration reuses `printedActivatedTextEffect` and its ordered baseline.
The existing activation service retains IDs, timing, limits, history, payment X,
targets, choices and result bindings. Pending bodies retain their earlier rules,
selected modes, targets and controller. Ability removal and restoration retain
state; cleanup resets limits even while phased. Clones have separate rule state.
Phasing keeps the visit. Departure retains final LKI; physical return and captured
copy values use printed rules. The shared cost transaction checks legality before
payment. No new payment, event, replacement or zone subsystem is introduced.

RON keeps the existing resolving shape
`CreatePermanentLayerEffect(permanentRef, [wordSubstitution, ...])`; typed preceding
color/basic-land pair choices supply each substitution. `compilePermanentObjectText`
lowers this shape through the complete-context dispatcher. Missing producers,
wrong word roles, equal-word reads and mixed modifications still reject at compile
time. Mana symbols and names stay fixed under CR 612.2; granted rules are excluded
under CR 612.3. Layers, departure and copies use CR 613.1c, 400.7 and 707.2.

Evidence: native `TestTextActivatedObjectAtomic`; RON
`TestRONTextActivatedObjectRecords` (Frozen Shade, Granite Gargoyle, Shivan Dragon,
Prodigal Sorcerer, Northern Paladin, Circle of Protection: Red; all 36 reprints),
`Siblings`, `TimingRemovalAndControl`, `PendingChoices`, `LandWordsAndResults`,
`Modal` and `Boundaries`. Complete Magical Hack and Sleight of Mind still reject
`CreateSpellEffect`; their extracted permanent branches are explicit test fixtures.
This contract establishes another shared prerequisite, not complete consumer play.

## Complete mixed static/event object text

`Game.AddPrintedMixedStaticEventTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` applies color and basic-land word substitutions to
one complete paired permanent context. `ChangePrintedPermanentText` dispatches
this context at resolution through the existing RON typed word bindings.

The definition must retain `WithMixedStaticEventSpellTextContinuation`, exactly
one marked global fixed P/T or basic-land-type static rule, and one controller
mana bonus or controller-upkeep fixed mana payment with unpaid self-sacrifice.
The mana bonus must use its layered text marker and the tapped land's controller
as recipient. No Aura, Land, extra keyword, mode, casting target/cost or other
rule is permitted. Duration is `Indefinite` or `EndOfTurn`. The installation
requires the original present face-up battlefield visit.

Preflight checks the retained and fresh definitions, declaration groups, static
kind, subtypes, event shape, fixed payment symbols and bonus mana symbol. Both
runtime declarations must be present. An isolated game probe checks the existing
static registrations and every required declaration layer before live changes.
All layers are installed together, followed by one live recomputation. Failure
leaves live declarations, registrations and effects intact. The supported upkeep
rule has no replaceable words; it retains its original trigger and payment body.

Static and mana-event layers keep independent ordered baselines, IDs and current
subjects. Existing services retain mana planning/production, event recipients,
upkeep timing and pending controller/source captures. Payment rollback, trigger
copies, clones, phasing, removal/restoration, LKI, printed return and copiable
values retain their shared contracts. No replacement or simultaneous-operation
policy changes. Rules: CR 612.1–3, 613.1c/d/f/g, 400.7 and 707.2.

RON retains `CreatePermanentLayerEffect(permanentRef, [wordSubstitution, ...])`
with preceding typed color/basic-land choices. `compilePermanentObjectText`
already lowers that shape through the dispatcher; no new spelling or alternate
execution path is needed. Invalid producers still reject. Complete Magical Hack
and Sleight of Mind retain their unsupported `CreateSpellEffect` branch.

Evidence: `TestTextMixedStaticEventObjectAtomic`; RON
`TestRONTextMixedStaticEventObjectRecords` (Gauntlet of Might: five records;
Conversion: six; Sunken City: one), `Payments`, `Lifecycle`, `Ordering` and
`Boundaries`. Extracted consumer permanent branches are explicit test fixtures.
This establishes a shared engine prerequisite with zero compilation unlocks.
Attached contexts, other mixed contexts and live copy rebinding remain separate.

## Complete creature Aura object text

`mage.CreatureEnchantTarget() mage.Target` declares a fixed creature enchant
restriction with typed provenance. `mage.WithAttachedObjectText() mage.CardOption`
marks one complete Aura context. `Game.AddPrintedAttachedObjectTextEffect(source
*Permanent, duration core.Duration, changes ...TextChange) error` checks the
retained definition and fresh declaration correspondence before it installs the
existing single-declaration text layer. `ChangePrintedPermanentText` selects this
context at resolution. Only `Indefinite` and `EndOfTurn` durations are supported.

RON marks exactly two rules, in either order: `EnchantPermanent(IsCardtype(Creature))`
and either `PermanentLayerEffect(HostPermanent, [AddAbility(abilities)])`, or one
`Activated(PayMana([fixedManaSymbol, ...]), ActionList([hostAction]))`. Grants accept
validated keyword symbols, fixed basic landwalk, and
`ProtectionAndDoesntRemovePermanents(FromColor(Colors([...])), SinglePermanent(ThisPermanent))`.
The action is `RegeneratePermanent(HostPermanent)` or
`CreatePermanentLayerEffectUntil(HostPermanent, [AdjustPT(power, toughness)], UntilEndOfTurn)`
with fixed integer P/T. The card must be an Enchantment with only the Aura subtype.
There are no other printed abilities, intrinsic keywords, modes or casting costs.
Unbound word choices, wrong binding types, malformed grants and extra text-layer
parts still fail compilation. Other complete Aura shapes do not acquire this marker.

The engine checks the original present face-up Aura, the fixed enchant target,
one eligible printed rule, fresh context, groups and effect counts. Native
ordinary actions also retain cost declaration counts, body counts, modes, timing
and limits.
The underlying installer checks active runtime declarations and complete grant
registrations on an isolated clone. Failure leaves live rules and layers intact.
Typed nil runtime declarations reject before the installer reads their IDs.

Existing layers apply ordered changes from each declaration's saved baseline.
Ward grants keep only the exact donor exempt from protection; other matching
Auras remain illegal. Grant subjects follow current attachment. Aura actions
belong to the donor controller and read the host when they resolve; a departed
donor uses its saved attachment, while returned or phased hosts cannot receive
an action. Pending activations keep their earlier bodies and controller.
Text on a recipient does not change granted rules. Copies and game clones keep
separate state. Phasing preserves visits, removal suspends rules, expiry restores
surviving layers, departure retains final LKI, and new entries use printed rules.
Ordinary text changes stay outside copiable values.

Rules: CR 303.4a/c/e/m, 400.7, 612.1–3, 613.1c/f/g, 702.16b–f/n and 707.2.
Cost transactions, replacement order and simultaneous operations use existing
services; this slice changes none of those contracts. Mixed Auras, changing
enchant filters, stack Aura text, live-copy rebinding and complete word-changing
consumer spells remain separate capabilities.

Evidence: `TestRONTextAttachedObjectRecords`, `Lifecycle`, `ActivationBindings`,
`CopiesAndLegality`, `Boundaries` and `Inventory`, plus native
`TestTextAttachedObjectAtomic`. Interaction evidence covers 14 names / 74 complete
records: five color Wards, Burrowing, Fishliver Oil, Flight, Fear, Lance, Eternal
Warrior, Firebreathing, Blessing and Regeneration. The permanent branches extracted
from actual Sleight of Mind and Magical Hack records are test fixtures; the
complete consumer cards still reject. Existing single-declaration grant/activation
contract tests retain damage, targeting, blocking, payment and lifecycle evidence.
This adds no full-card compilation accepts or collection-wide gameplay claim.

## Complete static P/T and activated object text

`Game.AddPrintedStaticActivationTextEffect(source *Permanent, duration
core.Duration, changes ...TextChange) error` accepts one complete marked creature:
one layered global P/T adjustment, or a source P/T adjustment conditional on its
controller/opponent controlling a basic land type, paired with one ordinary
layered activated action. Either declaration order is valid. The existing
`WithStaticActivationSpellTextContinuation` contract marks both stack and
battlefield use. Modal actions, history, captures, source-counter X, extra rules,
keywords, casting targets/costs and other types remain excluded.

`ChangePrintedPermanentText` selects this context at resolution. RON retains its
complete declaration selector, rule-position groups, fresh factory bindings and
existing `CreatePermanentLayerEffect(ref, [substitution, ...])` lowering. Typed
color/basic-land word choices must dominate their reads in the same lexical
scope. No new spelling or card-name dispatch is required.

The engine checks the old and fresh complete definitions, condition/recipient
context, subtypes, activation timing/limits, cost/target/body cardinality, target
shapes and exact sibling groups. Both existing declaration layer installers must
succeed on an isolated game before either live layer is added. Missing builders,
runtime declarations or static registrations, typed nil declarations, stale or
phased visits, face-down sources, empty/invalid changes and unsupported durations
reject without changing the live source or effect count. Durations are Indefinite
and EndOfTurn. A valid no-match operation checks and installs both layers.

Each declaration retains its baseline, ordered changes and ID. Expiry recomputes
remaining changes, including a baseline continued from stack text. Static
conditions and recipients stay live. Pending activated bodies retain prior text
and targets; legality is checked again at resolution. Mana symbols and names stay
fixed. Copies and clones retain independent declarations and effects. Phasing
keeps the visit; layer 6 removal suppresses both rules. Departure keeps final LKI;
returned cards and copiable values use printed text (CR 400.7, 612.1–3,
613.1c/f/g, 613.4c, 707.2).

Existing regeneration, control-duration, cost transaction, replacement ordering
and simultaneous-operation contracts are reused without changes. Evidence:
`TestTextStaticActivationObjectAtomic`; `TestRONTextStaticActivationObjectRecords`,
`BothWords`, `Inventory`, `Boundaries`, `TargetVisit`, `PendingVisit` and
`ContinuedBaseline`. Sedge Troll maps all six complete records to conditional P/T
and regeneration; Thrull Champion maps its FEM record to global Thrull P/T,
targeted control and source-control duration. Extracted real Magical Hack and
Sleight of Mind permanent branches supply interaction evidence. Their complete
records still reject `CreateSpellEffect`; attached land rules and live-copy
rebinding require separate contracts.

## Complete protection/P/T object text

`Game.AddPrintedProtectionPTTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` checks a complete creature with two printed rules:
one marked color protection declaration and one marked P/T declaration conditional
on the opponent controlling a nontoken permanent of a color. The recipient is
this source or creatures with one fixed name under either player. Both rule
orders are supported. Additional rules, keywords, types, casting costs/targets
and Aura contexts are excluded by the existing protection/P/T continuation
contract.

`ChangePrintedPermanentText` dispatches this context at resolution. RON uses the
shared `protectionPTTextRules` selector, position-based declaration groups and
existing `compilePermanentObjectText` lowering. Its nonempty resolving
`CreatePermanentLayerEffect(permanent_ref, [substitution, ...])` list contains
only typed color/basic-land word substitutions. Compatible distinct saved word
choices must dominate their uses. RON supplies Indefinite; native callers may
also supply EndOfTurn. The compiler creates fresh references for each instance.

The engine validates old and rebuilt complete contexts and exact sibling
correspondence before checking live protection and registered P/T on an isolated
game. Missing builders/groups, changed recipient names/contexts, typed nil
runtime declarations, missing registrations, stale/phased visits, face-down
sources, invalid durations and invalid words reject without live mutation.
Basic-land substitutions have no matching words in this context, but still
validate both live declarations. Color substitutions install two existing layer
3 effects together, then run one layer pass. Each declaration keeps its active
ID and baseline. Only semantic words change; mana symbols and card names retain
their values (CR 612.1–3).

The shared layers keep ordered changes and expiry, separate clones, live control,
opponent nontoken conditions and named recipients. Phasing keeps the visit;
layer 6 removal suppresses both rules and restoration keeps changed text.
Departure LKI keeps final protection and P/T; return uses printed rules
(CR 400.7, 613.1c/f/g, 613.4c). Copiable values exclude ordinary text changes
(CR 707.2). Stack-changed entry text remains the later layer restoration baseline.
No cost, replacement-order, damage-batch or simultaneous-operation service changes.

Evidence: native `TestTextProtectionPTObjectAtomic`; RON
`TestRONTextProtectionPTObjectRecords`, `OrderAndContinuation`, `TargetVisit`,
`Boundaries`, `Inventory` and `PendingBindings`, with the existing protection/P/T
spell declaration and lifecycle tests. The complete LEG records for Beasts of
Bogardan and Ivory Guardians gain battlefield interaction coverage. Complete
Sleight of Mind/Magical Hack records still reject `CreateSpellEffect`; tests use
an explicitly extracted real permanent branch for that interaction. Attached
land text, live-copy text and complete word-changing consumers remain separate.


## Complete attached land-type object text

`Game.AddPrintedLandAuraObjectTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` checks one complete land-type Aura before it installs
ordered text changes in layer 3. `LandEnchantTarget() Target` marks a fixed
“enchant land” restriction. The retained definition must use
`WithLandTypeAuraSpellTextContinuation`, one layered attached land-type rule,
and exactly the required entry-choice declaration. Both complete definitions and
every declaration group must correspond. Changed contexts, mutable enchant
filters, missing builders, runtime rules or registrations, invalid changes,
unsupported durations and stale, phased or face-down sources reject atomically.

RON lowers exact `EnchantPermanent(IsCardtype(Land))` to `LandEnchantTarget`.
Complete fixed `PermanentLayerEffect(HostPermanent,[SetLandType(basic)])` and
chosen `SetLandTypeVariable(TheChosenLandType)` contexts reuse the existing
land-Aura marker. The chosen form requires exactly one
`AsPermanentEnters(ThisPermanent,[ChooseABasicLandType])`. Existing
`CreatePermanentLayerEffect(permanent_ref,[word_substitution,...])` lowering
uses typed saved word pairs and `ChangePrintedPermanentText`; its dispatcher
now selects this complete Aura context. No card names select execution.

The transaction reuses `AddPrintedStaticLandTypeTextEffect`. Each layer pass
restores its baseline and applies surviving substitutions in order. Saved entry
choices are not printed words and never change or repeat (CR 612.2). Layer 4
uses the current exact attachment, land-type dependencies and intrinsic mana
(CR 305.7). Layer 6 removal cannot reverse that earlier type layer. Control,
phasing, expiry and clones keep separate rule state; returned sources and hosts
cannot inherit old visits. Spell-entry text remains the baseline. Copiable values
exclude ordinary text changes; departure keeps final LKI and printed return
(CR 400.7, 613.1c/d/f and 707.2).

Evidence: `TestTextLandAuraObjectAtomic` and `TestRONTextLandAuraObject*` cover
atomic rejection, all twelve Evil Presence/Phantasmal Terrain records, rule
order, real Magical Hack permanent-branch resolution, payment, choice timing,
independent pending results, mana, entry choices, ordered expiry, attachment
identity, control, phasing, removal/restoration, clone isolation, LKI and printed
copies/return. Complete Magical Hack/Sleight of Mind spell execution, mutable
enchant restrictions, other mixed Aura contexts and live copy rebinding remain
separate capabilities.

## Resolving spell word substitutions

`ChangePrintedSpellText(subject StackObjectRef, changes ...TextChangeRef)
ResolvingEffect` reads a nonempty ordered list of saved word pairs when its body
resolves. It resolves the subject's current exact stack visit and makes no new
choices. An absent visit makes no change; LKI cannot restore a departed spell.
Every pair must resolve before `Game.RevisePrintedSpellText` runs, so a missing
later pair cannot install an earlier change. The caller's slice is copied.
Definition validation checks the subject, nonempty list and all lexical reads.

RON lowers `CreateSpellEffect(spell_ref, [substitution, ...])` in resolving action
contexts. Supported substitutions are
`ReplaceColorWordVariableWithNewColorWordVariable(first, second)` and
`ReplaceLandTypeVariableWithNewLandTypeVariable(first, second)`. Each requires
compatible distinct references to a preceding lexical word-pair choice. Empty
lists, mixed nontext operations, unbound references and wrong arities reject.
Existing Lace `SetColor` union forms keep their separate adapter.

Complete `Targeted_Modal` records for Magical Hack and Sleight of Mind now compile.
Their unfiltered spell/permanent union chooses the category by the announced
object, without a mode choice. Each branch uses its own choice bindings. Copies
can retarget to the other category and choose fresh words; clones keep independent
stack changes and resolution frames. The engine revision service preserves
announced targets, payment, X, declaration groups, semantic word roles and
printed departure rules. Ordered changes accumulate on the same stack visit.
Revised target rules participate in later resolution legality (CR 608.2b,
612.1–3). Return creates another visit (CR 400.7). Phasing keeps a permanent's
visit but makes it unavailable while phased (CR 702.26).

Evidence: `TestRONTextSpellActionRecords` casts both complete consumers in all
six printings against pending intrinsic creatures and battlefield intrinsic
creatures. `CopyAndClone`, `Legality` and `OrderAndDeparture` cover independent
choices, cross-category copy retargeting, exact visits, phasing, shroud, ordered
land words, ordinary spell bodies and printed recasts. `MixedList` covers compatible color and land pairs in one list;
`RevisedTargetLegality` checks the pending Terror restriction. `Invalid` covers lexical
and shape rejection. Native `TestTextSpellActionDeclaration`, `Atomic` and
`SavedList` cover declaration checks, missing pairs, unsupported unmarked
subjects, atomic list validation, exact saved spell identity and valid no-match
changes. Existing spell continuation tests retain payment/result/LKI and copy
contracts. No payment, replacement or simultaneous-operation service changes.

Compilation acceptance is separate from complete target-context gameplay.
This action uses only the spell contexts supported by `RevisePrintedSpellText`.
The permanent branch uses only the complete contexts supported by
`ChangePrintedPermanentText`. Unsupported subjects still return explicit errors;
remaining complete mana-event, other mixed permanent and live-copy contexts need
separate contracts. These two cards do not yet have exhaustive general-target
coverage, and runtime cutover and Go-card/DSL removal remain gated.

## Complete battlefield cast-event object text

`Game.AddPrintedSpellCastObjectTextEffect(source *Permanent, duration core.Duration,
changes ...TextChange) error` checks one complete printed cast-event rule before
installing its existing layer-3 effect. The nonland, non-Aura permanent requires
`WithSpellCastSpellTextContinuation` and one
`GenericTriggered.WithLayeredSpellCastTextRevision` declaration. Intrinsic
keywords, sibling rules, modes, optional trigger declarations, intervening-if
conditions, casting targets and additional/alternate costs need separate contexts.
Resolution-time optional payments in ordinary action bodies remain supported.

Both retained definitions, declaration correspondence, subtype context, event
capture flags, target shapes, body count and active runtime declaration pass
preflight before installation. Empty/invalid changes, unsupported durations,
missing builders/groups, removed rules and absent, phased, face-down or returned
visits reject without installing a layer. Durations are `Indefinite` and
`EndOfTurn`. Valid substitutions with no matching word still install a layer.

`ChangePrintedPermanentText` selects this context at resolution. RON uses the
shared `spellCastTextRules` selector for exactly one
`TriggerA(WhenAPlayerCastsASpell(AnyPlayer | Opponent, spellFilter), body)`.
Existing filter, target, action and lexical checks apply. Resolving nonempty
`CreatePermanentLayerEffect` word lists use typed saved choices and instance
bindings through `compilePermanentObjectText`; no new RON spelling is added.

The service reuses `BuildTextDefinitionRevision`, declaration groups and
`AddPrintedSpellCastTextEffect`. Ordered changes and expiry use the saved layer
baseline, including prior spell text continuation. Pending triggers retain
detection-time rules, targets, controllers and event captures. Optional payment
choices occur at resolution and use the shared cost transaction. Clones and
copies retain separate bindings. Phasing retains the visit; ability removal
suspends detection. LKI retains final rules, return restores printed rules and
ordinary changes stay outside copiable values (CR 113.7a, 603.2–3, 612.1–3,
613.1c/f). No new replacement or simultaneous-operation service is introduced.

Evidence: native `TestTextSpellCastObjectRejectsAtomic`; RON
`TestRONTextSpellCastObjectRecords`, `Lifecycle`, `Targets`, `QueuedTargets`,
`IndependentPayments` and `Boundaries`. Complete subjects cover Crystal Rod,
Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere, Citanul Druid, Urza's Chalice,
In the Eye of Chaos, Nether Void and Presence of the Master (10 names / 35 records),
with both complete Sleight of Mind and Magical Hack consumers (70 cases).
This establishes this subject context; general text subjects and live-copy
rebinding remain incomplete.


## Printed ordinary upkeep declaration text revision

`GenericTriggered.WithUpkeepTextRevision() *GenericTriggered` marks an ordinary
battlefield upkeep declaration. `Game.RevisePrintedUpkeepTrigger(source
*Permanent, abilityID uuid.UUID, changes ...TextChange) error` replaces that
printed declaration after checked definition rebuilding. It requires nonempty
valid changes, the original present face-up source, its printed declaration
group, an active marked rule, and a retained text builder. Invalid requests leave
runtime declarations and revision state unchanged. Typed-nil runtime rules reject.

RON lowers `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer |
SinglePlayer(You) | SinglePlayer(TheChosenPlayer)), actions)` on its existing
bound-step path. The chosen-player form requires the existing opponent entry
choice in lexical scope and uses `ChosenPlayerUpkeep(player *PlayerBinding)
EventPattern`. This typed event reads the saved entry choice at detection and
captures the matching event player. It does not repeat the entry choice. Other
upkeep forms retain their existing adapters. Host-controller, intervening-if,
optional-trigger, modal, state and source-zone contexts cannot use this service.

A revision keeps the any-player, controller or chosen-player context, one body,
and target declaration kinds, counts and bindings. Target filters and semantic
words can change. Each fresh body owns its event, target and payment bindings.
The runtime declaration keeps its ability ID and current source/controller.
Ordered revisions use the declaration's retained definition, independently of
siblings and other instances. Names and mana symbols keep their printed values.

Detection captures the event player and controller before stack placement
(CR 603.2b, 603.3a). Pending bodies and trigger copies keep their prior rules and
facts (CR 603.3, 113.7a). Choices and optional payments occur at resolution;
current hand and land counts are read then. Returned targets remain illegal.
Phasing suspends detection and revision, but preserves the visit and its revision.
Ability removal stops detection; restoration uses the revised printed rule.
Departure leaves pending bodies and exact source LKI available. Return rebuilds
printed rules and makes a new entry choice. Game clones retain separate runtime
rules, bindings, payments and actions. Shared trigger, target, payment, damage,
life, zone and clone services retain APNAP and replacement ordering.

This is a declaration prerequisite. The caller must provide layer-3 order,
duration and baseline restoration. Whole-object text dispatch, spell text
continuation and live-copy rebinding need separate contracts. Granted recipient
abilities cannot be revised as printed declarations (CR 612.3).

Evidence: native `TestTextUpkeepTriggerDeclaration` and `RejectsAtomic`; RON
`TestRONTextUpkeepTriggerRecords` (all ten Black Vise, The Rack and Ivory Tower
records, four hand sizes), `Karma` (six complete reprints with changed land words),
`Payments` (Force of Nature), `PendingTargets`, `Identity`, `ChosenIdentity`,
`IndependentDeclarations` and `Invalid`. Existing opponent-entry, hand-size,
upkeep-payment and intervening-if tests retain their prior contracts.


## Layered ordinary upkeep declaration text

`GenericTriggered.WithLayeredUpkeepTextRevision() *GenericTriggered` opts in
an ordinary printed battlefield upkeep declaration. It includes the existing
checked revision marker. `Game.AddPrintedUpkeepTextEffect(source *Permanent,
abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error`
accepts nonempty validated color/basic-land word changes with `EndOfTurn` or
`Indefinite` duration. The source must be the original present face-up visit,
and the declaration must be active, printed, grouped and backed by a retained
text builder. Invalid requests install no effects and change no declarations.
Typed-nil rules, granted wrappers, foreign events, missing markers/builders,
changed event context/capture/targets/body shape and other durations reject.

RON opts in `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer |
SinglePlayer(You) | SinglePlayer(TheChosenPlayer)), actions)` through its typed
bound-step lowering. The chosen-player form still requires the opponent entry
choice. Host-controller, intervening-if, optional/modal/state and source-zone
contexts keep their separate services.

Each layer pass restores a saved declaration baseline, then applies current
changes in timestamp order (CR 613.1c, 613.3, 613.7). A prior direct revision can
supply the baseline. Cleanup recomputes surviving effects from that baseline;
it cannot retain a word introduced only by an expired effect. Direct revision
rejects while a layered result is present. Sibling declarations and other source
instances retain their own baselines and IDs. Names, mana symbols and saved
entry choices remain fixed (CR 612.2). Granted rules are excluded (CR 612.3).

Detection keeps earlier bodies and captured event players/controllers before
stack placement. Targets, trigger copies, optional payment and live counts use
the shared resolution services. Layer-6 removal stops detection; restoration
uses current text. Phasing keeps the visit and baseline; expiry during phasing
is recomputed on phase-in. Clones retain separate effects, cleanup, baselines
and controller contexts. Departure retains the final rule in LKI and pending
bodies. Return rebuilds printed rules and makes fresh entry choices.

This is a declaration layer prerequisite. Whole-object dispatch, upkeep spell
continuation and live-copy rebinding remain separate capabilities. No new
replacement, payment, zone, simultaneous-operation or trigger subsystem is added.

Evidence: native `TestTextUpkeepLayerDeclaration` and `Rejects`; RON
`TestRONTextUpkeepLayerRecords`, `ChosenIdentity`, `Karma`, `Payments`,
`PendingTargets`, `Identity`, `IndependentDeclarations`, `Invalid`, `Inventory`
and `OrderAndExpiry`. Real gameplay maps Black Vise, The Rack and Ivory Tower
(all ten records), Karma (six records), and Force of Nature (one record).
The 52-name / 108-record inventory proves declaration-layer acceptance, not
complete gameplay for every subject.


## Complete ordinary upkeep object text

`WithUpkeepObjectTextRevision() CardOption` marks one complete ordinary upkeep
context. `Game.AddPrintedUpkeepObjectTextEffect(source *Permanent, duration
core.Duration, changes ...TextChange) error` accepts validated nonempty color or
basic-land word changes with `EndOfTurn` or `Indefinite` duration. The definition
must be a nonland, non-Aura permanent with one marked ordinary battlefield
upkeep trigger. A chosen-player trigger requires exactly one
`ChooseOpponentAsEnters()` declaration; the other contexts require none.
Fixed intrinsic keyword symbols can accompany the upkeep trigger. Other
sibling rules, casting costs/targets, modal/optional/intervening/state triggers
and other source zones are outside this contract.

RON selects this complete context for `TriggerA(AtTheBeginningOfAPlayersUpkeep(
AnyPlayer | SinglePlayer(You) | SinglePlayer(TheChosenPlayer)), actions)`, with
exactly `AsPermanentEnters(ThisPermanent, [ChooseAPlayer(Opponent)])` for the
chosen-player form. `CreatePermanentLayerEffect` word lists keep their existing
lexical pair validation and lower to `ChangePrintedPermanentText`. This service
now dispatches to the upkeep context after resolving the saved word pairs.

Both complete definitions and all declaration groups pass preflight before any
layer is installed. The active printed trigger must keep its event-player role,
context, target shape and declaration ID. Missing builders/groups, typed nil,
granted wrappers, absent/phased/face-down/returned visits and invalid revisions
reject atomically. The entry choice is checked but never repeated by a text
change. Names and mana symbols stay fixed (CR 612.2); granted abilities keep
their separate rules (CR 612.3).

The existing upkeep declaration service supplies ordered layer-3 baselines,
expiry, control and clone isolation (CR 613.1c, 613.3, 613.7). Pending triggers
keep detection text, event players/controllers, targets and payment bindings
(CR 603.2b, 603.3a). Shared services preserve target legality, payment rollback,
damage prevention, phasing, ability removal/restoration, final LKI and printed
return with a fresh entry choice. No new trigger or replacement pipeline is added.
Upkeep spell continuation and live-copy rebinding remain separate capabilities.

Evidence: native `TestTextUpkeepObjectRejectsAtomic` and `Declarations`; RON
`TestRONTextUpkeepObjectRecords` (Black Vise, The Rack and Ivory Tower: all ten
records, four hand sizes and both real word consumers), `Karma` (six reprints
with pending old words), `PendingTargets`, `Identity`, `ChosenIdentity`,
`OrderAndExpiry`, `Payments`, `Prevention`, `Boundaries` and `Inventory`.
The inventory proves complete-context installation for 10 names / 26 records;
it does not prove full gameplay for every subject. New consumer gameplay maps
four names / 16 records. Compilation coverage is unchanged.

### Ordinary upkeep spell text continuation

RON marks a nonland, non-Aura permanent whose complete rules are one
`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer | SinglePlayer(You)), actions)`.
The actions use existing validated action, target, choice and payment scopes.
The compiler adds `mage.WithUpkeepSpellTextContinuation()` with the complete
object marker and layered trigger marker. Fixed intrinsic keywords can stay.
One paired opponent entry choice can supply chosen-player timing. Other entry
rules and mixed rules require separate contexts.

`Game.RevisePrintedSpellText` checks both retained definitions and declaration
correspondence before replacement. Revised spells enter with fresh bindings;
fixed mana symbols, physical identity and printed departure rules stay intact.
Copies become tokens with independent rules. Later upkeep text layers start
from the continued baseline. See `TestRONTextUpkeepSpell*` for real RON word
consumers, all eight subject names, targets, payments and lifecycle evidence.

### Fixed intrinsic keywords beside an upkeep trigger

The complete upkeep object context also permits the fixed symbols `Flying`,
`FirstStrike`, `Trample`, `Vigilance`, `Haste`, `Reach`, `Banding`, `Defender`,
`Fear`, `Deathtouch`, `Lifelink`, `Shroud`, `Hexproof`, `Menace` and
`Indestructible`. RON retains each rule position and uses the existing keyword
lowering. Parameterized keywords, protection, landwalk, extra events and other
sibling rules require separate complete contexts. All action bodies must still
pass their existing lexical and engine validation.

`Game.AddPrintedUpkeepObjectTextEffect` validates both complete definitions and
all declaration groups before installing its layer. Intrinsic keyword seed maps,
including duplicate contribution counts, must be equal. Added, removed, changed
or invalid contributions reject without installing an effect or changing live
rules. Ordinary color and basic-land word substitutions cannot change these
fixed keywords (CR 612.2). Layer 6 can still remove them; text layers cannot
restore an ability removed in that layer (CR 613.1c/f). Granted abilities keep
their separate rules (CR 612.3).

The existing upkeep declaration layer handles ordered changes, expiry, pending
bodies, exact visits, control, phasing, clones, removal/restoration, final LKI and
printed return. Payments retain their mana symbols and use the existing cost
transaction. This adds no payment, event, zone, replacement or combat pipeline.
`WithUpkeepSpellTextContinuation` also supports these fixed contributions;
the spell extension below establishes that entry contract.

Evidence: native `TestTextUpkeepKeywordsAtomic` and `Declarations`; RON
`TestRONTextUpkeepKeywordsRecords`, `Wall`, `Inventory`, `Lifecycle`, `Payments`
and `Boundaries`. Real word consumers cover Force of Nature (6), Phantasmal
Forces (6), Serendib Efreet (2), Lord of the Pit (6), Junún Efreet (1),
Palladia-Mors (1), Cosmic Horror (1), Yawgmoth Demon (1) and Wall of Tombstones
(1): nine names / 25 records. Paid/unpaid branches, mandatory and optional
sacrifice, fixed mana, toughness updates, flying legality and trample damage
retain their engine behavior. Mishra's War Machine remains rejected for its
unsupported damage-result condition; keyword eligibility cannot bypass it.
These are new text-context gameplay mappings for already compiled cards.
Compilation counts do not increase.

### Fixed-keyword upkeep spell continuation

`WithUpkeepSpellTextContinuation() CardOption` now permits the fixed intrinsic
keywords listed above beside its single ordinary upkeep trigger. RON
`upkeepSpellTextRules([]Node) bool` accepts those symbols in any rule position;
it retains the complete object-context check and permits one paired opponent
entry choice. Other entry rules stay separate.
Unknown or parameterized keywords and foreign sibling rules remain errors.

`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
checks both complete definitions and requires equal intrinsic seed maps,
including duplicate counts, before replacing the exact stack visit. Existing
`BuildTextDefinitionRevision` checks every declaration group, including keyword
groups without an ability member. Added, removed, changed or invalid keyword
contributions reject atomically. The entry builder creates fresh trigger,
target and payment bindings while retaining all fixed keyword contributions.
Names and mana symbols remain fixed (CR 612.2).

Existing stack-copy and clone services keep separate revisions. Copies enter as
tokens (CR 707.10f); ordinary text changes remain outside copiable values.
Later object text layers use the continued baseline and restore it on expiry.
Phasing keeps the visit, layer-6 removal suppresses keywords and detection,
and restoration keeps the continued rule (CR 613.1c/f). Pending triggers keep
their detection text and controller. Departure retains final LKI and the
physical printed card; a new entry uses printed rules and makes a fresh opponent
choice when declared. Other entry rules, mixed siblings and live-copy rebinding
require separate contracts.

Evidence: native `TestTextUpkeepSpellKeywordsAtomic` and `Declarations`; RON
`TestRONTextUpkeepSpellKeywordsRecords`, `Wall`, `Inventory`, `Payments`,
`Targets`, `Lifecycle`, `Identity` and `Boundaries`. These cover all nine names /
25 records in the fixed-keyword upkeep object mapping above through real
Sleight of Mind and Magical Hack on the stack, fresh entry, and subsequent
battlefield text changes. Shared gameplay helpers also retain the earlier
object and ordinary-spell checks. Mishra's War Machine remains rejected for
its nested damage-result condition. Compilation gains are zero.


### Chosen-player upkeep spell continuation

`WithUpkeepSpellTextContinuation() CardOption` also permits one
`ChooseOpponentAsEnters() AsEntersBattlefieldAbility` paired with
`ChosenPlayerUpkeep(player *PlayerBinding) EventPattern`. The complete upkeep
object and layered trigger markers remain required. RON accepts exactly
`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])` with one
`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),actions)`.
Existing fixed intrinsic keywords can stay in any rule position. The compiler
validates the entry declaration before body lowering, regardless of rule order.
Malformed, duplicate and unbound choices reject. Other entry operations and
foreign siblings do not acquire this complete text context.

`Game.RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error`
checks both complete definitions, every ability's declaration group and order,
fixed keyword counts, target shapes and event-player context before atomic
replacement. Added, removed or changed choice groups cannot replace a pending
spell. Announced targets, bodies and casting results remain forbidden in this
permanent spell context. No new exported API is needed.

Checked entry rebuilds fresh rules and event bindings. The shared entry service
saves the entry controller's opponent before registration (CR 614.12a). In this
two-player engine that opponent is unique; no prompt, target, payment or trigger
is added. Copies enter as tokens and choose for their own controller; game
clones keep independent state (CR 707.10/707.10f). Ordinary word changes and
saved choices remain outside copiable values. Control changes and phasing keep
the choice; later layers use the continued text baseline. Expiry and ability
restoration keep that baseline and declaration identity. Removal and phasing
stop new detection. Pending triggers and trigger copies keep the event player
and detection text, even after source departure or return (CR 603.3/603.3a).
Departure retains final LKI and the physical printed card. A new entry uses
printed words and makes a fresh opponent choice (CR 400.7/400.7a).

Evidence: native `TestTextUpkeepSpellChosenDeclarations`, `Atomic` and `Entry`;
RON `TestRONTextUpkeepSpellChosenRecords`, `Lifecycle`, `RuleOrder`, `Payments`,
`Targets` and `Boundaries`. `Records` covers all six Black Vise and two The Rack
records through real Sleight of Mind and Magical Hack, copies, clone isolation,
resolution hand-size changes, prevention, control, phasing, removal and return.
`Lifecycle` uses actual changing land words to distinguish ordered stack changes,
copy and clone rules, pending detection text, later layer expiry and printed
copiable values. `Targets` checks pending filters and returned target illegality;
`Payments` checks saved-event-player payment, fixed symbols and rollback. These
are two additional complete text-context mappings, with zero compilation gain.


## Resolving chosen sacrifice

`SacrificeChosenBy(player, chooser PlayerRef, subjects PermanentSet) ResolvingEffect`
keeps the sacrificing player separate from the chooser. At resolution it reads
that player, restricts the typed collection to their current controlled objects,
and uses `ChoosePermanent` for one required, non-target choice. No candidates or
an absent player/chooser causes no movement. Invalid answers use the shared
choice service's first legal candidate. A final exact-incarnation, phasing and
control check precedes `controllersSacrificePermanents`. Ownership controls the
destination; regeneration, indestructible and targeting restrictions cannot stop
sacrifice (CR 701.21a). The existing preparation, replacement ordering, graveyard
ordering, zone batch and LKI event services remain in use. No payment or optional
offer occurs. Bindings belong to the current resolution frame; clones and copies
keep separate results. Waiting queries retain explicit source reads without
choices or movement. Definition validation rejects nil or unbound operands.

RON supports `SacrificeAPermanent(collection)` and
`SacrificeAPermanentOfAPlayersChoice(collection, chooser)` in resolving spell,
activation, trigger, delayed and scoped player bodies. Existing typed permanent
collections supply the selection; the sacrificing player is the current action
player. The implicit chooser is that player. An explicit chooser uses an
in-scope typed player reference or bare `Opponent`, selected from the sacrificing
player's opponents with a fresh lexical player binding. Arity, malformed symbols,
unbound references, empty collection expressions and non-resolving roles reject.
Activation sacrifice costs retain their existing behavior. These actions do not
produce `CostWasPaid` or `APermanentWasSacrificedThisWay` results.

Evidence: native `TestChosenSacrificePlayers`, `Definition`, `Identity` and
`Waiting`; RON `TestRONChosenSacrificeRecords`, `Declarations`, `SelfChoice`,
`PlayerScope`, `Spell` and `Inventory`. Demonic Hordes has complete gameplay
coverage for all six records, including its land destruction and upkeep payment.
Serendib Djinn has action-only evidence; its result predicate remains unsupported.

## RON per-player discard results

Resolving action sequences accept
`EachPlayerAction(AnyPlayer, DiscardUptoNumberCards(Integer(n)))`, with a
nonnegative fixed maximum. Each player chooses a number from zero through the
smaller of that maximum and their current hand size, then selects those cards.
All choices follow active-player order before one shared discard batch.

`TheNumberOfCardsDiscardedByPlayerThisWay(player-reference)` reads the latest
unconditional producer in that lexical sequence. A branch-local producer cannot
escape its branch; reads before a producer, malformed declarations and unbound
player references fail compilation. `DealsDamage_ThatPlayer` remains local to
the amount of a player damage instruction. Mind Bomb uses the existing
recipient-relative damage batch to read a separate count for each player.

Native services are `BindPlayerNumbers(string) *PlayerNumbersBinding`,
`PlayerNumber(*PlayerNumbersBinding, PlayerRef) ResolutionValue`, and
`DiscardUpToEach(PlayerSet, ResolutionValue, *PlayerNumbersBinding) ResolvingEffect`.
The collection declares one immutable NumberBinding for each of the engine's
two player seats. Existing saved-number frames supply clone, copy, delayed-body,
loop and cleanup behavior without another runtime storage system.

Counts include completed discards redirected to a library; cancelled moves and
stale hand visits contribute zero. Later card movement does not change a saved
count. Returned cards cannot replace the selected hand visit. Other discard
shapes and general player loops need separate adapters. Raw coin rounds use
the heads-or-tails damage contract below. No RON source changed.

Evidence: `TestRONPlayerDiscardResultsMindBomb`, `CopiesAndClones`,
`TimingRollbackAndPrevention`, `IndependentProducers`, `ActivationSources`,
`Invalid`, and `Inventory`;
`TestPlayerDiscardResultsBatch`, `Declaration`, `Identity`,
`CloneClearAndCancel`, `DelayedAndLoopFrames`, `WaitingReads`, and
`OrderEmptyAndDuplicateSets`.

## Future one-shot draw replacement

Use `mage.ReplaceNextDrawThisTurn(player, body)` in a resolving body. The player
is a typed PlayerRef and the body is a ResolvingEffect. The replacement saves
its player, source visit, controller, X, bindings and targets until one use or
cleanup. It survives source departure and phasing. It runs immediately through
the shared replacement pipeline and restores the enclosing resolution frame.
Consumption precedes body execution. Clones keep independent use and bindings.
Returned objects cannot replace saved visits; information reads use existing LKI.

RON lowers `CreateFutureReplaceWouldDraw(
NextTimePlayerWouldDrawACardThisTurn(player), [actions])`. Supply a supported
player reference and a nonempty supported action list. Body results stay local;
this event does not supply a new lexical drawing-player reference. Library
selection, outside-game entry and other future draw events remain separate gaps.


## RON required player payments and draw replacement discards

`PlayerMustCost(player, DiscardACard)` accepts one typed player reference and
an adjacent `If`, `Unless`, or `IfElse`. The condition is the symbol
`CostWasPaid` or `CostWasntPaid`. Every declared branch is a nonempty action
list. The result belongs to that pair; it cannot be read by a later action,
another payment, or a sibling branch. Existing player bindings and reference
legality remain in force. `ChooseAPermanent` is a separate unsupported cost.

Ordinary spell, activated, triggered and delayed bodies lower to
`IfRequiredPaid`, `UnlessRequiredPaid`, or
`PayRequiredOrElse(player PlayerRef, costs CostGroup, paid, unpaid ResolvingEffect)`.
The native required-payment group now permits uncaptured discard costs as well
as matching sacrifice costs. It remains nonempty and rejects other cost families
and captured payment values. Existing optional payments keep their contracts.
Payment occurs at resolution without an optional payment offer, through the
shared transaction. Full-payment failure leaves resources unchanged.

Inside a static or future draw replacement body, the same RON form denotes an
instructed discard. It lowers to
`DiscardOneOrElse(player PlayerRef, discarded, notDiscarded ResolvingEffect)`.
That service uses an effect-caused discard and selects the branch from the
actual discard result. Library destinations still count; cancellation does not.
Library of Leng can replace that discard. A delayed trigger created in such a
body retains its saved player binding but uses ordinary resolution-payment
semantics when it runs. RON keeps the execution context separate from bindings.

Neither service stores a global payment result or adds a stack object.
Copies, clones and nested bodies keep independent outcomes. Complete Chains
still rejects its `MillACard` body; Takklemaggot still rejects its distinct
creature choice and linked Aura return. Tests cover these real contexts with
explicit supported test bodies; this is prerequisite evidence only.

## RON damage union bindings

`Targeted([AnyTarget], body)` and
`Targeted([AnyTargetOfAPlayersChoice(Opponent)], body)` bind `Ref_AnyTarget`.
A homogeneous list of these declarations binds `Ref_AnyTarget1`,
`Ref_AnyTarget2`, etc. Each is a separate singular group and can select the
same legal recipient. Opponent choices use the action controller's targeting
permissions. Copy retargeting uses the copy controller (CR 707.10c).
Damage packets, singular damage and existing resolving prevention actions share
these typed recipient bindings. Mixed group kinds remain unsupported.

`AnyTargetAsAPermanent` projects one singular union binding for resolving
permanent operations and information reads. A player supplies no permanent.
A permanent keeps its exact saved incarnation; information can use departure
LKI, while actions require presence. Numbered groups cannot supply this
unnumbered projection. Invalid arity, wrong choosers, unbound/ambiguous reads
and cross-mode reads reject during compilation.

Engine callers use `DeclareOpponentChosenDamageTarget(name)` with `WithTargets`
and `PermanentFromDamageTarget(target)` with existing permanent actions.
The projection shares the declaration; it does not declare another target.
Evidence: native `TestDamageTargetProjection*` and RON
`TestRONUnionDamageTarget*`, including complete Cuombajj Witches gameplay.
Disintegrate death replacement and Pyrotechnics allocation remain unsupported.


### RON temporary death replacement

Resolving actions support exactly
`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ref)), [ExileItInstead], UntilEndOfTurn)`.
`ref` must be a bound typed permanent reference in that action's lexical context.
Ordinary, triggered and delayed action bodies use the same lowering. A singular
AnyTarget declaration can supply `AnyTargetAsAPermanent`; player recipients
supply no permanent. Each factory retains its own target declarations.
Wrong arities, unbound references, unsupported events, operation lists and
durations fail compilation. Damage-history collections and static graveyard
replacement declarations remain unsupported.

The adapter uses `mage.ExileIfWouldDieThisTurn`. The engine saves one present
incarnation and uses shared replacement choice, graveyard proposals and zone
batches. Disintegrate installs this rule even with zero or prevented damage when
the target is still a creature. State-based actions run after the complete spell
body, so lethal damage cannot move the recipient before this rule is installed.


## RON fixed spell damage distribution

The compiler accepts ordinary instant/sorcery
`SpellActions(TargetedDistributed([AnyNumberOfAnyTargets],
DistributeNumberAmongTargets(Integer(n)),
ActionList([SpellDealsDistributedDamage(ThisSpell)])))`, where `n` is a positive
integer. The body has exactly one instruction and one union target group.
It rejects nested declarations, modal/activated/triggered contexts, variable or
nonpositive totals, incompatible sources, extra operations and wrong arities
with positioned errors. The implicit recipient set belongs only to this plan.

A fresh `mage.DeclareAnyNumberDamageTargets("Ref_AnyTargets")` and
`mage.DealAllocatedDamage(mage.This, recipients, n)` share one typed declaration
per factory call. The existing engine chooses shares before payment and stack
publication, requires at least one per chosen target, rejects duplicate and
illegal targets without spending resources, and permits zero targets without an
allocation prompt. Payment failure preserves the hand and leaves no stack object.
No new engine execution or payment service is introduced.

Shared stack services retain saved visits, legal-target rechecks, clone isolation
and copy retargeting. Copies keep shares by target position and make no new
allocation choice. Illegal, phased or returned targets lose their shares without
redistribution. A phase-out/in cycle keeps the visit. Damage uses the shared
simultaneous batch and prevention pipeline, including planeswalker loyalty.
The native union-batch observer test checks that every callback sees all applied
packets, after prevention. Existing allocation and damage replacement tests supply
regression evidence for unchanged shared rules.

`TestRONDistributedDamagePyrotechnics` covers the complete LEG RON record and
real {4}{R} payment, empty and four-target groups, creatures/players/planeswalkers,
partial/all illegality, return, phasing, prevention, direct copies and clones.
`Announcement` checks invalid target choices, too many positive shares and failed
payment. `RetargetAndInstances` checks retargeted spell copies and reconstructed
factory bindings. `Invalid` and `Inventory` cover declarations and semantic roles.
`TestDistributedDamageUnionBatchContract` checks the exact native engine shape.
Rules: CR 601.2c–d/h, 608.2b/d, 400.7 and 702.26b/d.

Pyrotechnics gains complete RON gameplay coverage. Fireball retains separate
casting-surcharge and rounded-division adapters. No RON source changed.


## RON ordinary spell equal damage

`SpellDealsDamageDividedAmongRecipientsRoundedDown(ThisSpell, value, recipients)`
accepts existing typed resolution values in ordinary instant/sorcery action lists.
Recipients are either `EachPermanent(collection)` using the resolving permanent
collection compiler, or the symbol `Ref_AnyTargets` produced by one
`Targeted([AnyNumberOfAnyTargets], body)` declaration. Each factory constructs a
fresh `DamageTargetsBinding`, shared only by its own target declaration and
consumers. Empty target groups are legal. Singular and plural references are
separate types; unbound reads and invalid kinds/arity fail compilation.

The collection form calls `DealDamageEvenlyToRecipients`; the target form calls
`DealDamageEvenly`. Both use the existing engine damage batch and resolution X.
Modal, activated, triggered, delayed and replacement bodies and numbered plural
union declarations remain unsupported contexts. `TargetedDistributed` retains its announcement allocation
contract. Equal division never offers an allocation choice.

The complete Dwarven Catapult record executes. The TARGET-COUNT-SPELL-SURCHARGE
slice below also executes all six complete Fireball records.
No RON data or vocabulary source hash changed.


## RON target-count spell surcharge

Ordinary nonmodal instant and sorcery records accept one
`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(n)])))`
with a positive integer literal `n`. The payment must contain exactly one generic
symbol. Wrong kinds, arities, colored or X payments, duplicate CastEffect rules,
permanent declarations, and modal bodies fail at compilation with a source position.
The form is a casting declaration; it is not a resolving action or activation cost.

RON lowers the validated amount to
`mage.WithAdditionalManaPerTargetBeyondFirst(n)` on its fresh spell action.
The existing shared announcement service counts target occurrences before payment.
Zero or one target adds no cost. Separate target groups can select the same subject
and each occurrence counts. Targets, X and damage bindings remain per instance.
Ordinary untargeted, singular, grouped and distributed spell bodies retain their
existing action contracts. Modal adapters and spell text continuation with this
cost remain separate capabilities.

All six complete Fireball records have RON gameplay evidence in
`TestRONTargetSurchargeFireball`. Payment, timing, copies, independent instances,
invalid declarations and inventory roles use the other `TestRONTargetSurcharge*`
tests. Compilation acceptance and gameplay evidence remain separate measures.

## Zone-event intervening conditions

RON accepts `TriggerI(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),
PermanentPassesFilter(ThisPermanent | HostPermanent, HasAbility(keyword)), body)`.
The host reference requires an Aura declaration. The condition uses
`PermanentMatchesFilterWithLKI(PermanentRef, PermanentFilter) StateCondition`
with `WithInterveningIf` at detection and resolution. A live source reads its
current attachment; a departed source reads its final attachment. Source and
host incarnations remain exact through departure, return, phasing and cloning.
Information does not permit actions on departed subjects.

RON also accepts `TriggerI(WhenAPermanentIsPutIntoAPlayersGraveyard(filter, AnyPlayer),
PermanentPutInGraveyardPassesFilter(WasntSacrificed), body)` with the existing
Land/Artifact and optional `ControlledByAPlayer(SinglePlayer(You))` event filters.
`WithDeathNotSacrificed() deathNotSacrificedOption (implements TriggerOption)` checks the historical death cause at
both times. The engine retains immutable event facts per pending trigger and
copied ability. Other event contexts, duplicate guards, modal or appended bodies, and combinations with
another intervening condition fail definition validation. Existing event player
bindings and optional payment bodies keep their shared services.

Complete Urza's Miter is covered by `TestRONZoneInterveningMiter` and
`TestRONZoneInterveningNativeLoadAndCast`. Earthbind's condition is covered as
an explicit prerequisite; its nested acquired static ability remains rejected.

Resolving Aura spells now attach through the shared spell-entry service before
entry effects and the entry event (CR 303.4, 603.6a/b). This includes spell copies.
Spell-to-permanent damage/exile replacement continuation still precedes entry
notifications. Target legality uses the existing resolved-target service.
`TestRONZoneInterveningNativeEntry` and `NativeAuraCopy` verify native casting,
entry-time host checks, copied Auras and clone isolation. `SourceEntry` checks
source keyword changes, departure, return and phasing.


## Acquired layer 6 static abilities

`AcquireStaticAbility(subject PermanentRef, rule StaticRule) ResolvingEffect`
reads one live recipient at resolution and saves its exact battlefield visit.
It accepts a `While` continuous rule with layer 6 modifications and no nested
activated or triggered grants. Declaration validation gives the acquired rule
a fresh binding scope and a battlefield source context. Unsupported rule kinds,
empty rules, missing conditions, other layers and unbound outer bindings fail
validation. Each acquisition has a separate static declaration identity.

The recipient supplies the rule's live source and controller. Its attachments
and set membership remain live. The independent acquisition effect grants the
declaration in timestamp order; the declaration's own effects use shared
dependency ordering and require that ability to remain present. An earlier
removal can be followed by acquisition; a later removal stops the static rule.
The layer scheduler retains inactive effects until an earlier grant can activate
them in the same pass. Reattachment retimestamps the acquired static effect.
Phasing suspends the rule and preserves acquisition. Departure ends that visit;
returned recipients cannot inherit it. Donor departure does not end a different
recipient's acquisition. Clones keep separate battlefield and layer state.
Acquired rules do not enter printed or copiable card definitions. Explicit This recipients are declared to
`Game.WaitingDelayedPermanentSources` without resolving them or executing the
static rule. The nested rule has its own future recipient context; its reads and
implicit attachments do not supply reads by the waiting action. Saved source
visits and cloned waiting lifetimes use the shared delayed-reference service.

RON accepts the exact resolving shape
`CreatePermanentLayerEffect(ThisPermanent, [AddAbility([PermanentLayerEffect(HostPermanent, [LosesAbility(keyword)])])])`
on an Aura, with one supported fixed keyword. The nested host is evaluated in
the acquired static context, not saved as the host at acquisition. Other subjects,
combined grants, nested rule effects and temporary acquisition remain rejected.
Earthbind uses this adapter after its existing intervening-if entry condition
and shared damage operation. Prevention does not prevent acquisition. Damage
uses source LKI after departure, but acquisition requires the original live Aura.
A copied trigger rechecks flying: an earlier resolution can make it fail.

CR 400.7, 604.2/4/7, 611.2a/c, 611.3a, 613.1f, 613.7a/b/e and 613.8a-c
supply identity, duration, live attachment, timestamp and dependency contracts.
`TestAcquiredStatic*` covers engine lifecycle, declaration isolation, recipient
context and invalid rules. `TestRONAcquiredStatic*` covers all six complete
Earthbind records, native casting and rollback, trigger/spell copies, pending
clones, prevention, entry/resolution timing, source and host returns, phasing,
control, removal, reattachment, strict loading and semantic inventory.

## RON live turn damage-history rules

Resolving permanent collections accept
`WasDealtDamageByPermanentThisTurn(permanentRef)`. Selection uses actual damage
this turn and both original battlefield visits. Collections can combine this
subject with live filters through `And` and `Or`.

The two-argument form
`WasDealtDamageByPermanentThisTurn(permanentRef, IsCardtype(Creature))` selects
recipients that were creatures when damage occurred. Later type changes keep
that fact. No other damage-time filter is supported.

`CreateEachPermanentRuleEffectUntil(subject, [CantBeRegenerated], UntilEndOfTurn)`
accepts one history subject, or its one-argument form combined with live filters
through `And`. The rule reads later damage until cleanup. It does not grant an
ability. `APermanentWouldDie(subject)` with one `ExileItInstead` operation and
`UntilEndOfTurn` accepts the one-argument history form and live-filter conjunctions.
Damage-time filters in that replacement context remain rejected.

Each action saves its own source visit at resolution. Factory instances, copies
and clones do not share chosen source state. Existing singleton death replacements
and collection attack rules keep their behavior. Runesword uses creature-at-damage
history for regeneration and a current creature filter for death replacement.

### RON bounded next-combat schedule

`CreateFutureTrigger(AtNextEndOfCombatThisTurn, ActionList(actions))` accepts a
bare event symbol and a nonempty action list in supported resolving contexts.
It lowers to `AtNextThisTurn(EachEndOfCombat, Sequence(actions...))`. No targets
are declared inside the delayed body. Existing typed references and bindings
use the same capture rules as `AtNext`.

The schedule triggers once at the next end-of-combat event after installation.
An extra combat in the same turn can supply that event. Installation during
end-of-combat priority cannot see the event that already occurred. Cleanup
removes unused schedules; they cannot wait for another turn. Detection consumes
the schedule before stack placement. Countering cannot rearm it. Pending bodies
keep their captured frame after cleanup. Ordinary next-step schedules keep
their existing unbounded lifetime.

Glyph of Doom uses the complete LEG RON record. Its saved Wall visit supplies
live turn history after departure or phasing. A returned Wall cannot replace
that visit. Destruction selects current creature members when the body resolves;
returned and phased attackers are excluded. Shared batch destruction permits
regeneration and respects indestructibility. Source departure does not cancel
the schedule. Factories, spell copies and game clones keep separate frames.
Target-history predicates and other Glyph card prerequisites remain unsupported.

### Indefinite captured RON base P/T collections

`CreateEachPermanentLayerEffect(collection, [SetPT(PT(power, toughness))])`
accepts one fixed integer pair in resolving action lists. It lowers to
`Until(Indefinite, SetEachBasePowerToughness(subjects, Fixed(power), Fixed(toughness)))`.
The existing contextual collection compiler supplies typed members. Empty or
compound modifications, dynamic pairs and unbound references reject at compile
time. This shape does not add a static layer or animate noncreature members.

`DealtDamageToPermanentThisTurn(permanent)` lowers to
`PermanentsThatDealtDamageToThisTurn(permanent, MatchingPermanents(AnyPermanent))`.
It can participate in existing collection intersections and unions. The source
death event now binds `Trigger_ThatPermanent` to its exact departing visit,
separately from `Trigger_ThatGraveyardCard`. Each factory owns its declarations.
The history query reads identity after departure or phasing without permitting
an absent permanent to perform an action. Current creature membership is read
when Brine Hag's trigger resolves; its resulting set stays fixed indefinitely.
Evidence: `TestRONCapturedBasePT*` and `TestCapturedBasePT*`.

Reverse turn-damage collections also declare explicit source reads to the shared
waiting-delay query. That query does not select members or resolve the effect.
`TestCapturedBasePTDelayedSource` checks the declaration and saved departure.


### Complete RON heads-or-tails damage rounds

Ordinary spell actions support
`RepeatableActions([EachPlayerAction(Or([SinglePlayer(reference), ...]), FlipACoin),
SpellDealsDamage(ThisSpell, Integer(n), EachPlayer(CoinCameUpTails)),
Unless(AllCoinsCameUpHeads, [RepeatThisProcess])])`.
The player union must be nonempty; every reference must be bound in the enclosing
scope. The three instructions and bare result/control symbols must match this
shape exactly. Dynamic amounts, other round actions, activated rules, modal
bodies, delays and replacement contexts reject at compilation.

The compiler lowers the complete procedure to `RepeatHeadsOrTailsDamage`.
All flips precede one simultaneous tails-damage batch; all heads ends repetition.
Prevention changes damage, not coin results. Each invocation owns its local
outcomes. Spell targeting, source information, payments, copies, clones and RNG
use existing engine services. Mana Clash's complete DRK record is covered by
`TestRONCoinRound*`; engine contracts are covered by `TestCoinRound*`.

## Active player control predicates

RON accepts `ControlledByPlayer(TheActivePlayer)` in singular
`TargetPermanent` filters, alone or within nested `And` filters. The factory
lowers it to `TargetControlledBy(ActivePlayer)` with fresh target declarations.
The engine accepts this typed reference in definition validation. Existing
restrictions, target legality and exact incarnation checks still apply.

Resolving collections accept the same predicate and combine it with existing
`And` and `Or` collection operands. It lowers to
`PermanentsControlledBy(ActivePlayer, AnyPermanent)`. Delayed actions read the
active player and current members when their body resolves. Target declarations
read the active player at selection and each later legality check. Neither
operation captures the earlier active player or substitutes the action controller.

The predicate has exactly one symbol operand. Malformed symbols, missing or
extra operands and unsupported player references remain errors. This adapter
does not establish control-since-turn-start predicates, activation timing,
static control filters or plural target declarations. Nettling Imp remains
rejected as a complete record. Both cards have active-player clause evidence
across all six reprints. Siren's Call also has complete-record evidence below.

Evidence: `TestRONActivePlayerControlTargets`, `Collections`, `Invalid`,
`Native`, `RepositoryClauses`, and `Inventory`; engine
`TestTargetActivePlayerContract`. No full-card gameplay gain is claimed.

### RON current-turn control history

`ControlledSinceBeginningOfTurn` and `NotControlledSinceBeginningOfTurn` are
symbols in singular target filters and resolving permanent collections. The
collection form `ExceptFor(NotControlledSinceBeginningOfTurn)` selects subjects
whose current controller has kept control continuously since this turn began.
Use `mage.ControlledSinceCurrentTurnStart` for the shared predicate. The
controller's most recent own turn uses the separate existing condition.

Targets recheck current history at resolution. Collections read current members,
including entries after announcement. Phasing preserves history; a returned
permanent starts new history. Siren's Call has RON gameplay coverage for both
same-turn end steps and a next end step on another player's turn. Its corrected
records save the earlier active player in a scoped player binding.
Nettling Imp's exact target clause is covered; its complete records now use
the activation windows documented below. Other exclusions and static contexts
are not added by this adapter.

### RON saved active player in delayed bodies

`PlayerActions(TheActivePlayer, [CreateFutureTrigger(event, ActionList(actions))])`
saves the active player when the outer instruction resolves. `ActionPlayer` is
bound only inside that player body, including its future delayed body. Nested
player bodies have independent bindings; sibling actions retain their prior
scope. `ControlledByPlayer(ActionPlayer)` is supported in resolving permanent
collections, including existing intersections and unions. Explicit
`ControlledByPlayer(TheActivePlayer)` continues to read the live active player.
Target, static and entry `ActionPlayer` contexts remain separate adapters.

The shared `SnapshotPlayer`, `FromPlayer`, `AtNext` and
`PermanentsControlledBy` services supply execution. No new public engine API is
needed. Step-event shapes, body validation and one-shot lifetimes retain the
existing delayed adapter. Invalid or escaped bindings reject before loading.

All six Siren's Call records save the player before creating their delay. Local
Oracle in `data/LEA.json` says "that player" in the delayed instruction. A skipped
end step can move that instruction to another turn; its permanent membership
and current-turn control history are still read when the delay resolves.
Evidence: `TestRONSavedActivePlayerScopes`, `Siren`, `Invalid`, `Native`,
`TestSavedActivePlayerDelayedFrames` and existing `TestRONCurrentTurnControlSiren`.


## RON singular temporary attack requirements

Use `CreatePermanentRuleEffectUntil(ref, [MustAttack], UntilEndOfTurn)` in a
resolving body. `ref` must be a bound typed singular permanent reference.
Ordinary, numbered and union-derived permanent targets keep their declaration
identity; source, attachment and saved references use their existing scopes.
One bare `MustAttack` symbol and `UntilEndOfTurn` are required. Mixed lists and
other durations need separate adapters.

The engine service is `mage.MustAttackThisTurn(subject mage.PermanentRef)`.
It captures a present subject once and requires that exact battlefield visit to
attack if able through cleanup. Current control determines who attacks. A phase
cycle keeps the rule; return cannot inherit it. Source departure and ability
removal do not end it. Restrictions still apply and attack costs stay optional.
Copies, clones and delayed frames keep separate reference state.

Nettling Imp uses this rule in all six records. Its target, tap cost, requirement
and delayed destruction have clause evidence in `TestRONSingularAttackRuleImpClauses`.
Its complete records also have timing and lifecycle evidence in
`TestRONActivationWindowImpGameplay`.

## RON activation turn and combat windows

`ActivatedWithModifiers` and its modal activation path accept
`ActivateOnlyIf(IsAPlayersTurn(You|Opponent))` and the identical
`IsPlayersTurn` spelling. They also accept bare `IsDuringCombat`,
`IsBeforeAttackersDeclared`, `IsBeforeBlockersDeclared` and
`IsBeforeCombatDamageStep`. Nonempty nested `And` condition lists can combine
these window predicates. Other condition families keep their separate adapters.
Modifier-level nested `And` lists can combine windows with existing costs,
permissions, limits and sorcery timing.

Lowering uses `WithActivationStateCondition(StateCondition)`, `IsYourTurn`,
`NotCondition`, `AllConditions`, `IsDuringSteps` and
`BeforeCombatStep(point PhaseStep, during bool)`. The acting player supplies
`You`; ownership does not. A before window uses first-combat history unless
any window in the same modifier conjunction contains `IsDuringCombat`.
That conjunction uses current-combat history, including extra combats, regardless
of operand order or nesting. Skipped steps close the existing engine windows;
first-strike damage closes the damage window. New turns reset history.

Queries are pure. Legality is checked before targets and payment. These conditions
grant no priority or permission and cannot replace sorcery timing. A failed
activation leaves resources and the stack unchanged. Once announced, the body,
stack copies and pending clones do not repeat the timing check. Source departure,
phasing and control changes cannot cancel an announced body. The shared target,
payment, saved-reference, attack and delayed-trigger services retain their rules.
Malformed tuples, empty conjunctions, unknown predicates, `Or` and unbound
player references fail compilation. Strict loading remains atomic.

Evidence: `TestRONActivationWindowTiming`, `ConjunctionHistory`,
`CostsControlAndNativeSkip`, `ModalPermission`, `InvalidAndNative`, `Inventory`
and `ImpGameplay`;
engine `TestActivationWindowTypedContracts` and existing `TestCombatTiming*`.
All six complete Nettling Imp records run without removing any rule. Existing
Angus Mackenzie activation and spell casting-window regressions remain supported.

## Required graveyard choice and battlefield return

`ReturnChosenGraveyardCard(graveyard, controller PlayerRef, filter CardFilter)
ResolvingEffect` makes a required non-target choice for the resolving effect
controller. `ReturnChosenGraveyardCardToOwner(graveyard PlayerRef, filter CardFilter)
ResolvingEffect` instead uses the selected card's owner as its entry controller.
The graveyard and explicit controller references are read once before the choice.
Candidates are current matching cards in that graveyard. Empty sets do nothing.
One candidate needs no decision; multiple candidates use `ChooseCardFromGraveyard`
without a `ChooseMayAbility` offer. Nil and foreign answers use the first eligible
candidate, as with the shared required permanent-choice contract.

Each candidate's graveyard visit is saved before the decision. The selected card
must still occupy that exact visit in the same graveyard. Departure and return
cannot supply the move. Choices are local to each operation. Shared
`returnGraveyardCardToBattlefield` retains zone replacements, entry restrictions,
entry operations, ownership and fresh battlefield identity. This is one move;
it does not change simultaneous multi-card movement or payment transactions.
Source-derived player references declare waiting reads without making a choice.
Existing capture frames retain source LKI, phasing and departed visits for delayed
bodies; copies and cloned games keep their frames separate.

RON lowers
`ReturnACardFromPlayersGraveyardToBattlefield(IsCardtype(Creature), player,
[EntersUnderOwnersControl])` or one
`EntersUnderPlayersControl(controller)` entry. Player operands use existing typed
lexical references in ordinary resolving bodies and supported nested/delayed
bodies. Other filters, entry lists and unbound operands reject at compilation.
This slice supplies the shared return action for Reincarnation and Glyph of
Reincarnation; their complete records remain unsupported. It does not supply
owner-information reads, delayed death-event bindings or death-result loops.

Evidence: `TestChosenGraveyardReturnContracts`, `Definition`, `WaitingSource`;
`TestRONChosenGraveyardReturnGameplay`, `RealClauses`, `DelayedTiming`, `InvalidAndNative`,
and `Inventory`. These cover chooser versus graveyard player, required-choice
fallback, exact visits, owners and entry controllers, current resolution
membership, delayed source LKI/phasing/departure, copies, clones, declaration
errors, nested inventory and atomic collection rejection. Rules: CR 115.10,
400.7, 603.7 and 608.2c/d/h. These are clause and engine contracts, not complete
card gameplay claims.


## RON delayed death-event bindings

`CreateTriggerUntil(WhenAPermanentDies(SinglePermanent(ref)),
ActionList(actions), UntilEndOfTurn)` declares `Trigger_ThatPermanent` inside
its body. `ref` uses an existing singular permanent reference. The declaration
works in ordinary spell, activation and supported nested delayed bodies.
`OwnerOfPermanent(Trigger_ThatPermanent)` and
`ControllerOfPermanent(Trigger_ThatPermanent)` read separate players saved at
death detection. The permanent reference keeps the departed battlefield visit
for existing LKI readers; actions cannot affect a returned visit.

Each factory makes fresh bindings. The adapter uses
`WhenDiesThisTurnWithBindings(subject PermanentRef, permanent *PermanentBinding,
owner, controller *PlayerBinding, body ResolvingEffect) ResolvingEffect`.
The three bindings belong only to the saved body frame. Both player bindings
must be present and distinct. Death facts are saved before trigger placement,
then installed in a separate copy of that frame. Waiting and stack queries keep
the exact watched visit. Cleanup removes subscriptions but keeps pending bodies.
Phasing sends no death; exile replacements do not trigger a death subscription.

Reincarnation now uses its dead creature's owner for both the graveyard and
entry controller. The source correction follows `data/LEG.json` Oracle text.
`TestRONDelayedDeathCapture*` covers its complete record, resolution choices,
legality, copies, simultaneous deaths, independent instances and cloned games.
Leaves events, other owner operands, plural watched subjects, new targets,
other durations and out-of-scope event reads remain separate contracts.

## RON specific-permanent tap payments

`MustCost(TapPermanent(ref))` followed immediately by
`If(CostWasPaid, actions)` or `Unless(CostWasPaid, actions)` makes a mandatory
payment at resolution. `CostWasntPaid` supplies the opposite condition. The
nonempty continuation keeps its own result; another payment cannot change it.
`ref` must be a bound singular permanent reference in that action context.
Ordinary spell, activation and supported delayed bodies use the same lowering.
Optional payments, explicit `PlayerMustCost` and `IfElse` forms remain separate.

The adapter uses `RequiredTapOrElse(player PlayerRef, subject PermanentRef,
paid, unpaid ResolvingEffect) ResolvingEffect`. The effect controller pays, as with other implicit `MustCost` payments.
Only the exact present, unphased, untapped subject can pay. Tapping does not
require control unless the declaration supplies that restriction. A written tap
instruction permits summoning sickness. Payment makes no optional offer and
uses the shared transaction and tap event service.

`TapPermanent(HostPermanent)` in an Aura's activated cost uses the existing
`TapAttachedPermanentCost() Cost`. The transaction locks both source and host
visits. It rejects departure, return, phasing, changed attachment and an already
tapped host before spending live resources. The adapter does not change the
existing `TapPermanent(ThisPermanent)` tap-symbol behavior.

Energy Tap's `AddManaRepeated(fixedProduction, ManaValueOfPermanent(ref))`
uses the existing typed mana-value reader and `AddManaAmount`. This context
accepts one fixed-color production; multiple productions and a color choice
remain rejected. The information read permits exact departure LKI, while the
payment itself requires a present object. Mana is added during resolution.

`TestRONSpecificTapPayment*` maps complete Energy Tap (LEG:76) and Tourach's
Gate (FEM:97) gameplay. It covers announcement, payment, legality, upkeep,
cleanup, attachments, independent instances, copies, clones, delayed bodies,
invalid declarations, inventory and atomic loading. RON source is unchanged.

### RON contextual spell filters

Spell targets and battlefield cast events accept
`ControlledByAPlayer(SinglePlayer(You))`,
`SharesAColorWithAPermanent(permanentFilter)`, and `ExceptFor(spellFilter)`.
Nonempty `And` and `Or` compose these predicates with existing spell card-type,
Aura and color predicates. `You` is the action controller. A color-sharing
member filter can include one conjunctive controller restriction and existing
pure permanent filters. Unbound players and controller restrictions inside a
member disjunction remain errors.

Targets read current spell control, color and battlefield members both at
announcement and resolution. Cast-event filters read those facts at detection
only. Colorless objects cannot share a color, and phased members do not qualify.
Pending bodies keep their exact spell, caster and independent payment bindings.
Goblin Artisans still requires its separate ability-target exclusion rule.

Engine adapters: `SpellFilter`, `SpellCardMatches`, `SpellControlledByYou`,
`SpellSharesColorWithPermanents`, `AllSpellFilters`, `AnySpellFilter`,
`NotSpellFilter`, `SpellTargetBinding.WithFilter` and `SpellCast().Matching`.
Existing pure card filters and cost-filter APIs remain supported.


## RON controller-cast triggers and optional drawing

`WhenAPlayerCastsASpell(SinglePlayer(You), filter)` lowers to
`SpellCast().ByController().Matching(filter).WithSpell(...).WithCaster(...)`.
The selector must contain the symbol `You`. Other singular player references
remain declaration errors. Existing AnyPlayer and Opponent contexts stay valid.
The source controller at detection supplies You (CR 603.2, 603.3a). Later source
or spell control changes do not cancel a pending trigger or change its captured
caster. Spell copies cause no cast event; failed casting causes no event.

`MayDrawCard(player PlayerRef) ResolvingEffect` offers one choice to the selected
player at resolution, then uses `Game.PlayerDrawCard` if accepted. An absent
player makes no choice. An empty library still permits the choice and preserves
normal draw failure. Draw replacements and events use the existing service
(CR 121.1, 603.5). Player references retain explicit delayed source reads.
RON lowers exactly `MayAction(DrawACard)` in resolving ActionList and PlayerActions bodies;
argument-bearing DrawACard forms remain invalid. Each choice stays local and
produces no payment or drawn-card result binding.

Evidence: `TestRONControllerCastEnchantress` executes the complete unchanged
Verduran Enchantress record. `Records` validates all six reprints and strict
atomic loading. `Invalid` and `ScopedOptionalDraw` cover declaration errors and
independent player scope. Native `TestControllerCastDetectionAndBindings`,
`Declarations` and `OptionalDraw` cover composition, saved event identities,
copies, clones, typed validation, accept/decline, absence and draw replacement.
Stack-source Mana Vortex uses the self-cast contract below; general optional action lists remain unsupported.

## RON controller permanent-presence state trigger

`TriggerA(WhenAPlayerControlsAPermanent(SinglePlayer(You), filter), actions)`
lowers to `mage.WhenState(mage.Controls(filter), body)` in ordinary battlefield
rules. The player must be exactly `SinglePlayer(You)`. The permanent filter
uses the existing context-free filter compiler. Nested `And` and `Or` groups
must be nonempty. Source, target, chosen and event references in the filter
remain unsupported. This event supplies no player or permanent bindings.
The existing action compiler builds targets and bodies afresh per factory.

The condition reads the source's current controller and present matching
permanents, including tokens and the source. It uses current subtype layers.
Phasing and ability removal stop detection. A pending body keeps its controller
and source visit; a later false condition does not cancel it. The shared state
trigger service prevents repetition while pending or on the stack, then permits
another trigger if the state still holds. See `TestRONControllerPresence*` and
native `TestControllerPresenceStateBatchAndReplacement`.

## RON stack ability target exclusion

Permanent activated `TargetSpell` declarations accept
`IsntTheTargetOfAnAbility(AbilityOfAPermanent(filter))`. The source filter uses
context-free permanent predicates, fixed names and nonempty `And`/`Or` lists.
A conjunctive `Other(ThisPermanent)` excludes the activating source's exact
battlefield visit. Contextual controller selection, source-dependent disjunctions,
unbound references, spell actions and cast-event declarations remain rejected.

`mage.SpellNotTargetedByPermanentAbility(filter mage.PermanentFilter,
excludeSource bool) mage.SpellFilter` supplies the engine predicate. Target
queries use abilities currently on the stack, including copies and triggered
abilities. Matching reads the current source or its exact departure/phase
information. Returned sources cannot replace saved visits. Resolution checks
again; illegal targets stop the coin and draw instructions through shared
resolution services. The predicate makes no choices or state changes.

Goblin Artisans uses this adapter with a creature, fixed-name and other-source
conjunction. The ATQ RON record includes the Oracle creature restriction.
`TestRONSpellAbilityExclusion*` covers the complete record and declaration
boundaries; `TestSpellAbilityExclusionIdentity` covers native identity queries.

## RON global battlefield-absence state trigger

`TriggerA(WhenNoPermanentsAreOnTheBattlefield(filter), actions)` lowers to
`mage.WhenState(mage.BattlefieldHasNone(filter), body)` in ordinary battlefield
rules. The filter uses the context-free permanent-filter compiler. Nested
`And` and `Or` lists must be nonempty. Source, controller, target, choice and
event references in the filter are rejected. The event supplies no bindings.
Targets and body declarations are built afresh for each factory instance.

The condition reads current unphased permanents under both players, including
tokens and the source. Current layers determine membership. Source phasing,
departure and ability removal stop detection. One trigger can be pending or
stacked for each source visit and rule. A later matching entry does not cancel
the body. Resolution and countering permit a new trigger if the state still
holds. Pending bodies retain their controller and exact source visit. Shared
target, sacrifice, replacement, copy and clone services retain their contracts.

`TestRONBattlefieldAbsence*` covers the exact Mana Vortex and Drop of Honey
subrules, declaration rejection, atomic loading, semantic inventory, membership,
target legality and lifecycle. The full cards remain rejected for `FromStack`
and `DestroyAPermanentNoRegen`, respectively. Native mechanics are covered by
`TestBattlefieldAbsenceBatchAndReplacement`; no new engine API is needed.

## Stack-source self-cast triggers

`SourceSpellCast() spellCastPattern` uses the stack as its source zone and
matches only the exact source spell visit identified by the cast event.
`WithSpell` and `WithCaster` keep the existing typed captures. Stack detection
now saves those captures before queuing, using the same service as battlefield
triggers. Copies on the stack cannot inherit another visit's cast event. Real
spell copies and failed casting do not emit a cast event. Pending bodies keep
the detected controller, caster and spell identity after source departure or
control changes. Re-casting a physical card cannot replace the saved stack visit.
The normal APNAP, counter, clone and resolution-payment services remain shared
(CR 113.6k, 113.7a, 400.7, 601.2i, 603.2/3a, 707.10).

RON accepts exactly
`FromStack(TriggerA(WhenAPlayerCastsASpell(SinglePlayer(You), SingleSpell(ThisSpell)), actions))`
on a permanent card definition. Every constructor has its exact tuple arity;
`You` and `ThisSpell` must be symbols. This context supplies only the existing
cast spell/caster bindings, not a battlefield source or Aura host. Bodies use
the existing validated action compiler and fresh per-factory declarations.
Other source-zone events, instant/sorcery declarations and other self-spell
reference consumers remain separate. RON owns spelling and scope validation;
the engine owns exact-visit matching, capture timing and execution.

Evidence: `TestRONStackSelfCastManaVortex` executes the complete real record for
payment, decline, absent payment, failed casting, copies, departure and controller
changes. `ManaVortexBattlefield` covers both upkeep players, phasing and global
absence. `CloneAndRecast` uses an explicit later stack visit to prove old captures
cannot follow the physical card, and clone execution leaves original choices
unchanged. `RecordsAndInventory` checks fresh declarations, semantic reference
roles and strict atomic loading; `Invalid` checks malformed and foreign contexts.
Native `TestStackSelfCastExactVisit` places a copy beside the cast visit before
an event check, then verifies the captured counter removes only the cast visit.
Optional sacrifice transactions and replacement ordering retain the existing
`TestRONOptionalSacrifice*` and native cost/zone-batch contracts.

## RON minimum-power destruction choice

`DestroyAPermanentNoRegen(collection)` makes a required non-target choice for
its scoped player at resolution, then destroys that exact visit without
regeneration. Ordinary spell, activated, triggered and explicit player action
bodies use the shared action compiler. This operation has a private binding;
it does not produce `TheChosenPermanent` for later instructions.

Resolving collections accept
`PowerIs(EqualTo(TheLeastPowerAmongPermanents(collection)))`, including within
`And` and `Or`. The comparison collection is independent of the outer selection.
An empty comparison collection matches nothing. Other comparisons, target and
static predicate contexts remain unsupported for this dynamic value.

`mage.PowerEqualToLeastOf(subjects, compared) PermanentSet` supplies the typed
selector. Compose `ChoosePermanent` with `DestroyWithoutRegeneration` to execute
the choice. The complete Drop of Honey record uses the controller upkeep trigger
and the separate battlefield-absence trigger. See `TestRONLeastPowerChoice*`.

## RON typed permanent-control exchange

`ExchangeControl(firstPermanentRef, secondPermanentRef)` accepts exactly two
bound singular permanent references in resolving action bodies. Spells,
activations, triggers, delayed bodies and existing player/choice scopes use the
same adapter. Numbered targets remain distinct. Tuple forms of symbol references,
players, collections and unbound choices reject during compilation.

`mage.ExchangePermanentControl(first, second PermanentRef) ResolvingEffect`
resolves both exact visits and calls the existing `Game.ExchangeControl` service.
No result binding is produced. `ThePermanentChosenByPlayerThisWay` and
`PermanentsChangedControlThisWay` remain unsupported. Juxtapose and Gauntlets of
Chaos therefore remain rejected as complete records. `TestRONControlExchange*`
provides prerequisite gameplay evidence, including the actual Gauntlets exchange
AST in a supported synthetic declaration.

### Player-keyed permanent choices

`BindPermanentChoice(name)` declares one result. Use
`ChoosePermanentForPlayer(player, subjects, result)` during resolution.
`PermanentChosenIn(result)` reads that exact choice.
`PermanentChosenByPlayer(player, precedingChoices...)` reads the latest
preceding choice made by the specified player. Supply the choices in instruction
order. An empty latest choice gives no subject; it cannot reuse an older choice.
The chooser is saved before the choice. Later control changes do not change it.
All choices must precede the read in the same declared resolution path.

RON accepts `ThePermanentChosenByPlayerThisWay(playerRef)` after
`ChooseAPermanent(collection)` and direct
`PlayerAction(playerRef, ChooseAPermanent(collection))` in resolving sequences.
A direct player choice exports its keyed result without changing the enclosing
`TheChosenPermanent`. Branch choices stay local. Delayed bodies keep the saved
frame. `PlayerActions` list producers and payment choices remain unsupported.

### Grouped permanent choices

Use `ChoosePermanentsAPNAP(requests...)` with `PermanentChoiceRequest` values.
Each request has a `Player`, a `Subjects` collection and a separate `Result`.
Choosers are saved first. Required choices follow APNAP order and declaration
order for one player. Read results with `PermanentChosenIn` or
`PermanentChosenByPlayer` after the group. The group does not move subjects.

RON action sequences accept:

```ron
SimultaneousPermanentChoices([
    PlayerAction(You, ChooseAPermanent(collection)),
    PlayerAction(Ref_TargetPlayer, ChooseAPermanent(collection)),
])
```

Use `ThePermanentChosenByPlayerThisWay(playerRef)` in later instructions.
The group does not assign `TheChosenPermanent`. Group-local output reads,
empty groups, arbitrary actions, costs and static declarations are rejected.
Existing sequential choices retain instruction order.


## Required permanent cardinality choices and simultaneous sacrifice

`SacrificeAllButEach(players PlayerSet, subjects PermanentSet,
keep ResolutionValue) ResolvingEffect` reads the kept count once and saves all
present candidates by controller before choices. Counts below zero keep zero;
counts above available members keep all. Each selected player chooses the
permanents to **keep**, in APNAP order, with prior players' kept names and IDs
visible in the prompt. Selection uses distinct exact visits. Invalid chooser
responses receive a legal required-choice fallback. No choice moves a subject.

`LeastControlledPermanentCount(players PlayerSet, subjects PermanentSet)
ResolutionValue` supplies the minimum count, including zero-member players.
Absent and duplicate player IDs are excluded. Empty player sets supply zero.
Duplicate permanent visits, phased subjects and old incarnations cannot count.
The private `choosePermanentSubset` helper uses the existing `ChoosePermanent`
decision API for an exact bounded count; it can also supply future optional
count adapters once those adapters choose and save their own count/results.

Unkept visits still controlled by the choosing player enter the existing
`controllersSacrificePermanents` batch. Replacement choices and owner graveyard
ordering precede all moves. Indestructible and regeneration do not stop sacrifice.
Cancellation and exile replacement do not cause new choices. Shared zone events
retain departure LKI and the former battlefield. CR 101.4, 101.4b, 608.2d and
701.21a supply the choice, ordering and sacrifice contracts. No persistent choice
or last-action state is added. Copies, clones and delays use their own current
resolution context. Waiting queries inspect declarations without making choices.

RON accepts `EachPlayerAction(AnyPlayer,
SacrificeAllPermanentsExceptForNum(collection, count))` in resolving action lists.
`count` is `Integer(n)` with n >= 0, or
`TheLowestNumberOfPermanentsControlledAmongPlayers(collection, AnyPlayer)`.
Collections use existing typed resolving adapters, including saved singular
targets. Target selection and resolution legality remain with their declarations.
Other player sets, cost/static contexts, hand equalization, optional selections,
and saved plural choice results remain unsupported. The least-count spelling
is supported in this count context only.

Evidence: `TestPermanentCardinalityChoicesAndBatch`, `LeastAndDeclaration`,
`BoundsIdentityAndFallback`, `ReplacementsAndWaiting`; RON
`TestRONPermanentCardinalityBalanceInstructions`, `Invalid`,
`TargetsCopiesAndDelay`, `RolesAndInventory`. All six Balance reprints supply
unchanged land and creature instructions. The full card remains rejected at
`DiscardAllButNumberCards`; instruction coverage is not full-card gameplay.

### RON required hand cardinality

In resolving action lists,
`EachPlayerAction(AnyPlayer, DiscardAllButNumberCards(count))` makes each player
choose cards to discard down to the kept count. Supported counts are a
nonnegative `Integer(n)` and
`TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer)`. The minimum is read once
when this instruction resolves, including empty hands. All required choices
precede simultaneous discard; replacements do not cause a new choice.

The compiler lowers these forms directly to `mage.DiscardAllButEach` and
`mage.LeastHandSize`. Hidden choices preserve exact hand visits. The count
spelling is local to this action; other player sets, cost/static contexts and
saved discard results require separate adapters. Complete Balance gameplay is
covered by `TestRONHandCardinalityBalance` for all six reprints.

## Optional per-player tap results

RON resolving action lists support
`EachPlayerAction(AnyPlayer, TapAnyNumberOfPermanents(filter))`. The filter must
contain exactly one conjunctive
`ControlledByAPlayer(SinglePlayer(ActionPlayer))` clause, either alone or inside
nested `And` lists. Other clauses use existing contextual permanent collections.
The engine restricts the resulting set to the chooser's present untapped
permanents. Disjunctive controller bindings and other player collections reject.
No tap choices occur during casting or payment.

`NumberOfPermanentsTappedThisWayByPlayer(playerRef)` reads the most recent
preceding optional-tap producer in that lexical action sequence through
`PlayerNumber`. Ordinary typed player references and a bound `ActionPlayer` from
`PlayerActions` can select the saved seat. Branch producers stay in their branch;
independent producers use distinct declarations. Delayed bodies retain saved
counts. Activation costs, static rules, unbound reads and `EachablePlayer` outside
its plural-choice repetition scope reject. No global last-operation result is used.

`TestRONPlayerTapResultsRaidingPartyInstruction` executes the exact first
instruction of Raiding Party's activated body, including its Orc sacrifice cost,
failed-payment check, white/untapped/creature restrictions, both players' optional
choices and source departure. Complete-card evidence now also includes
`TestRONPermanentSetChoiceRaidingParty` and the existing target-restriction test.
`ScopesCopiesClonesAndDelay`, `ActivationSources`, `Invalid`, and
`InventoryAndAtomicLoad` cover saved counts, independent factories and producers,
stack copies, game clones, source control/phasing/return, delayed reads, declaration
errors, typed inventory roles and strict atomic loading. Existing Raiding Party
spell/ability target-restriction tests retain coverage for its other rule.


## Repeated optional permanent choices and saved exclusions

`ChooseUpToPermanentsEach(players PlayerSet, subjects PermanentSet,
limit ResolutionValue, repetitions *PlayerNumbersBinding,
result *PermanentSetBinding) ResolvingEffect` reads the limit, candidate visits
and each selected player's saved count before choices. It makes that many
separate bounded choices per player, in APNAP order. Each choice offers zero
through the smaller of the limit and present candidate count. Negative limits
and repetition counts mean zero. Distinct members are required within one
choice; repeated choices and different players can select the same visit.
Subjects do not need to be controlled by the chooser. Later players receive
earlier players' public selections. The existing subset selector supplies legal
fallbacks for malformed answers. Choices do not move or tap subjects.

`BindPermanentSet(name string) *PermanentSetBinding` declares the union result.
Each invocation clears the old result, including empty choices. The saved union
contains exact visits, with no duplicate visit. `FromPermanentSet` reads surviving,
unphased visits. `WithoutPermanents(subjects, excluded PermanentSet) PermanentSet`
reads both sets at resolution and excludes matching visits. Control changes and
phase cycles retain identity; departure and return cannot supply an old choice.
Existing simultaneous destruction, regeneration and replacements apply later.

`CreatedTokensBinding` is a compatibility alias for `PermanentSetBinding`.
Token entry and choice producers retain separate named declarations. Both use
the existing exact-visit result maps, definition dominance validation, resolution
frames, deep cloning, delayed captures and waiting/stack source queries.
No new action-result storage or token behavior is introduced.

RON supports the complete resolving shape
`EachPlayerAction(AnyPlayer, ChooseUptoNumberPermanentsForEach(Integer(n),
filter, NumberOfPermanentsTappedThisWayByPlayer(EachablePlayer)))` with n >= 0
and a preceding local tap producer. `EachablePlayer` is permitted only in that
count position. `NotChosenThisWay` is a symbol in later contextual permanent
collections. Branch results remain local, later producers use fresh declarations,
and delayed bodies retain saved choices. Other player sets, repetition forms,
costs, targets and static predicates remain unsupported.

CR 101.4/101.4b/101.4c, 400.7 and 608.2d–f establish choice order, exact identity
and resolution timing. Local `data/FEM.json` confirms that Raiding Party permits
any player's Plains and one up-to-two choice for each tapped creature.
Evidence: `TestPermanentSetChoiceRepeatedAPNAPAndOverlap`,
`VisitsResetCloneAndDelay`, `InvalidDefinitions`, `BoundsDistinctVisitsAndWaiting`;
`TestRONPermanentSetChoiceRaidingParty`, `Invalid`,
`CopiesClonesDelayAndScope`, `InventoryAndAtomicLoad`, and
`TestRONAbilityTargetRestrictionRaidingParty` with the complete factory.

## RON static land play restrictions

In a permanent's top-level Rules list, RON accepts
`EachPlayerEffect(AnyPlayer, [CantPlayLands])` and
`EachPlayerEffect(AnyPlayer, [CantPlayLandsOfType(predicate)])`.
Each declaration has a nonempty supported operation list. The filtered form accepts the
symbols `SharesANameOriginallyPrintedInArabianNights` and
`SharesANameOriginallyPrintedInAntiquities`, using the embedded CR 206.3 catalogs
against the candidate card's name. Physical set membership does not control the
result. Mountain is not in the ARN name list. RON owns these spellings and
lowers them to `mage.PreventLandPlays(CardFilter)`.

Tuple spellings for symbols, wrong arity, other player scopes, resolving/cost
contexts, permanent references, arbitrary filter shapes and unsupported operations
reject at compilation. Inventory reports the filtered operand as
`land_play_card_filter`. Strict collection loading still rejects unsupported
records and does not publish a partial registry. City in a Bottle still needs
captured state members; Worms of the Earth still
needs land entry prevention and alternative optional payment. No source RON
record changed.


## RON filtered casting restrictions and player-rule lists

`EachPlayerEffect(AnyPlayer, [CantCastSpells(predicate), ...])` accepts the
symbols `SharesANameOriginallyPrintedInArabianNights` and
`SharesANameOriginallyPrintedInAntiquities`. The fixed CR 206.3 catalogs match
card names, independent of physical printing. RON lowers the predicate to
`mage.PreventSpellCasts(CardFilter)` and reports `casting_card_filter` in the
semantic inventory. No permanent binding or result scope is needed.

Nonempty all-player lists compose the supported casting, land-play, public
visibility, untap-skip, untap-limit and spell-cost-increase operations. Each
operation must validate before a factory is returned. Declaration order and
repeated operations are retained. Factories create separate cards and abilities.
Unsupported operands, tuple spellings for symbols, wrong arity, other player
scopes, action/cost contexts and battlefield rules on instants reject. Invalid
collections publish no registry.

`TestRONSpellCastRestrictionCity` executes City's exact complete player-rule
subrule under both active players. The unsupported captured state trigger keeps
the full record rejected. `CompositionAndInventory` and `ListOrderAndInstances`
cover ATQ names, compound visibility/land rules, ordering, independent lifetimes
and atomic loading. `Invalid` covers declaration rejection. No RON source changed.

## RON captured permanent-set state triggers

`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(subjects), actions)`
is supported for ordinary battlefield declarations. Subjects accept the
context-free permanent filters, nonempty nested `And`/`Or` groups and
`Other(ThisPermanent)`. Dynamic player, target, choice and event reads in the
subjects remain rejected. Every tuple has its exact arity; the source reference
must be a symbol. The adapter creates a fresh `PermanentSetBinding` per factory
and lowers through `mage.WhenPermanentsPresent`.

The body supplies `Trigger_ThosePermanents` as a plural collection symbol.
It can be read by existing collection actions, including
`ControllersSacrificeEachPermanent`. It is unavailable in sibling rules,
other trigger events, singular references, costs or static declarations.
The ordinary action compiler retains target, player and delay scope checks.

Detection saves exact unphased visits, including both players' matching members.
One pending or stacked instance suppresses further detection for that source
visit and declaration. Later entries wait for a new detection; later type
changes do not change the pending set. Phasing prevents actions while preserving
identity; return cannot replace a saved visit. Pending bodies survive source
departure and phasing. Sacrifice uses current controllers and owner destinations
through the shared simultaneous replacement service. Copies and clones keep
separate resolution frames.

City in a Bottle's complete unchanged ARN record now compiles and executes both
its captured trigger and its compound casting/land-play restriction.
`TestRONStatePermanentSetCity` covers both active players, name membership,
source/token/basic-name exclusion, current controllers, owner destinations,
late entry and repeat detection. `Pending` covers identity, phasing, source
lifetimes, copying, countering and clones. `InstancesAndInventory` covers
separate factories, source exclusion, APNAP and semantic reference roles.
`Invalid` covers malformed syntax, unsupported subjects, result scope,
wrong reference type, source zone and atomic loader rejection.


## RON source-presence control duration

`GainControlOfPermanentUntil(subject,
UntilPermanentLeavesBattlefield(ThisPermanent))` is supported in ordinary
resolving permanent bodies. The subject uses the existing singular permanent
references. Only `ThisPermanent` supplies the duration endpoint. Other endpoints,
malformed declarations, spell sources, entry and replacement bodies remain
errors. This adapter does not extend the same spelling in layer declarations.

Lowering uses `mage.UntilSourcePresent(mage.This,
mage.ChangeController(subject, mage.You))`. The exact source must be present
and phased in at resolution. Later departure or phasing ends the duration
permanently. Source control changes, untapping and ability removal preserve it.
Target phasing preserves identity; return cannot receive the old effect.
The resolving controller stays fixed. Existing targets, payment transactions,
layer order, control restrictions, copies and clone frames remain in use.

`mage.UntilSourcePresent` also accepts captured characteristic modifications.
`mage.UntilSourceLeaves` remains separate and survives source phasing. The RON
layer and phasing lifecycle forms require their own adapters and contracts.
Evidence: `TestUntilSourcePresent*` and `TestRONSourcePresenceDuration*`;
Scarwood Bandits now has complete RON-backed gameplay coverage.

### Per-permanent optional life payments in RON

In ordinary resolving action sequences,
`ForEachPermanentAnyPlayerMayCost(collection, AnyPlayer, PayLife(Integer(n)))`
offers each eligible player a separate payment for each saved matching visit.
`n` is fixed and nonnegative. The first player who pays protects that visit;
the result contains visits for which no player pays. Prompts name the subject.
`CostWasntPaidForIt` reads the latest preceding producer's unpaid permanent set
in the same lexical sequence. Branch-local producers cannot escape their branch.

The native API is `OfferCostForEachPermanent(subjects PermanentSet,
players PlayerSet, costs CostGroup, unpaid *PermanentSetBinding) ResolvingEffect`.
Use `BindPermanentSet` and `FromPermanentSet` for its result. All offers precede
later actions; empty results replace old results. Existing exact visits and
resolution frames preserve phasing, return identity, clones, copies and delays.
Cleansing uses the existing `DestroyEach` batch on this result. Other RON
payment forms, dynamic life and entry/replacement contexts need separate adapters.
