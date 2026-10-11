// Package mage implements the two-player Magic: The Gathering rules engine in Go.
//
// [ExchangePermanentControl] exchanges current controllers of two typed visits.
// Both subjects must be present and able to change control. Same controllers
// cause no change. Shared layer 2 effects apply both changes in one pass.
// Source departure does not end the control effects. A phase cycle keeps them;
// returned subjects cannot inherit them. Targets keep resolution legality.
// RON lowers ExchangeControl in resolving bodies with singular permanent reads.
// Player-keyed choices and exchange-result conditions need separate adapters.
//
// [PowerEqualToLeastOf] compares two current permanent sets at resolution.
// Empty comparison sets supply no power. Negative power and ties keep their
// normal values. Selection does not exclude an indestructible subject.
// Combine [ChoosePermanent] and [DestroyWithoutRegeneration] for a required
// choice that keeps its exact visit. RON lowers DestroyAPermanentNoRegen with
// resolving collections and PowerIs(EqualTo(TheLeastPowerAmongPermanents)).
// Shared choices, destruction and waiting queries keep player and source facts.
// Targets, static predicates and other comparisons need separate adapters.
//
// [SourceSpellCast] matches the source spell at its exact cast stack visit.
// Stack detection saves typed event bindings before trigger placement. Copies
// do not inherit another visit's event. Pending bodies keep the caster and
// spell identity after departure or control changes. A later cast cannot
// supply the saved spell. RON accepts the explicit FromStack self-cast rule.
//
// RON global absence state triggers use [BattlefieldHasNone] and [WhenState].
// Present permanents under either player can match, including tokens and the
// source. Current layers determine membership. Phasing stops selection and
// source detection. Each source visit and rule keeps one pending or active
// trigger. Later entries do not cancel it. Pending bodies keep their controller
// and source visit. Shared services keep target legality, sacrifice replacements,
// copies and clones. Return cannot supply an earlier source visit.
//
// Use [SpellNotTargetedByPermanentAbility] for current stack ability targets.
// The source filter reads the exact permanent visit or its retained information.
// Source exclusion keeps that visit through departure, phasing and return.
// Shared targets check again at resolution. Copies keep the original source;
// clones keep separate state. The query makes no choices or state changes.
// RON accepts this predicate in permanent activated spell targets.
// Other declaration contexts and source controller filters stay separate.
//
// RON controller-presence state triggers use [Controls] and [WhenState].
// The condition reads present permanents and the current source controller.
// Tokens and the source can match. Phasing stops selection and detection.
// Each source visit and rule keeps one pending or active trigger. A later
// condition change does not cancel it. Pending bodies keep their controller
// and source visit. Shared sacrifice services keep replacement choices,
// owner destinations, copies and clones. Return cannot inherit an old body.
//
// RON controller-cast triggers use [SpellCast] with ByController. Detection
// saves the source controller, caster and exact spell visit. Pending bodies
// keep these facts after source control changes, departure or phasing.
// [MayDrawCard] offers one draw to its typed player at resolution. The shared
// draw service keeps replacements, events and empty-library failure.
//
// RON activation turn and combat windows use [WithActivationStateCondition].
// [IsYourTurn] reads the acting player. [BeforeCombatStep] keeps history from
// the first combat unless the same modifier conjunction requires combat. That
// context uses history from the current combat, including extra combats. Nested
// lists and their order keep the same meaning. Shared history closes skipped
// windows and resets each turn. Queries make no changes. Payment and other
// restrictions still apply. Pending bodies, copies and clones do not repeat
// the timing check.
//
// [MustAttackThisTurn] saves one present permanent until cleanup.
// The shared attack planner requires that exact visit to attack if able.
// Control changes and a phase cycle keep the rule. A return cannot inherit it.
// Source departure and ability removal do not end it. Attack costs stay optional.
// Copies, clones and delayed frames keep their separate subjects and lifetimes.
// RON lowers one MustAttack symbol in CreatePermanentRuleEffectUntil with
// a typed singular permanent reference and UntilEndOfTurn.
//
// RON PlayerActions can save [ActivePlayer] with [SnapshotPlayer] before a delay.
// Each [AtNext] frame keeps its own saved player after source departure.
// ActionPlayer reads that saved binding in the scoped body. Nested scopes keep
// separate bindings. Explicit TheActivePlayer reads still use the current turn.
// Resolving ControlledByPlayer(ActionPlayer) collections read current members
// through [PermanentsControlledBy]. Phasing excludes members; return starts
// new control history. Copies and clones keep separate saved frames.
//
// [ControlledSinceCurrentTurnStart] checks the current controller's continuous
// control since the current turn began. Entry and control changes this turn
// fail the check, including control loss and regain. This differs from
// [Permanent.ControlledSinceTurnStart], which uses the controller's last turn.
// RON lowers the positive and negative symbols in singular targets and
// resolving collections, plus ExceptFor(NotControlledSinceBeginningOfTurn)
// in collections. Target legality reads history again at resolution.
// Collections read current members. Phasing keeps history but stops selection;
// return starts new history. Shared batches keep destruction and regeneration.
// Nettling Imp uses the shared typed activation windows.
//
// [TargetControlledBy] accepts [ActivePlayer] for current turn control.
// Target selection and resolution read the active player each time.
// The action controller and source controller do not change that player.
// [PermanentsControlledBy] uses the same reference for live collections.
// Saved targets keep their exact visits; collections read current members.
// RON lowers ControlledByPlayer(TheActivePlayer) in singular target filters
// and resolving collections. Control history needs a separate predicate.
//
// Use [RepeatHeadsOrTailsDamage] for complete heads-or-tails damage rounds.
// Player references are read once. Duplicate and absent players are excluded.
// Every player flips before one simultaneous tails-damage batch. Each round
// reads its amount and exact source once. All heads ends the process, even
// when damage is prevented. Nonpositive damage does not stop repetition.
// These flips have no winner or loser. Results remain local to each round.
// Source LKI, replacement order, copies and clones use shared services.
// RON supports a player union, fixed spell damage to CoinCameUpTails and an
// adjacent AllCoinsCameUpHeads repeat condition in one ordinary spell body.
// Other round actions and declaration contexts need separate adapters.
//
// Use [DealDamageEvenlyToRecipients] for an equal division at resolution.
// It reads the total and selects the typed recipients once. Each gets a share
// rounded down before shared damage replacements and prevention. Empty sets
// and nonpositive shares cause no damage. Source LKI and simultaneous outcomes
// use the existing damage service. [DealDamageEvenly] keeps its target API.
// RON lowers ordinary spell division among live permanent collections or one
// plural union target binding. Ordinary spells can use target surcharges.
//
// Use [WithAdditionalManaPerTargetBeyondFirst] for an ordinary spell action.
// It uses the same announcement cost as [AdditionalManaPerTargetBeyondFirst].
// The count includes each announced target occurrence, before cost reductions.
// Shared payment keeps X, locked costs, rollback, copies and alternate casting.
// Definition validation rejects this cost on an activated action.
// RON accepts one positive fixed generic payment in a nonmodal CastEffect.
// Modal forms and spell text continuation need separate adapters.
//
// RON fixed spell distributions use [DeclareAnyNumberDamageTargets] and
// [DealAllocatedDamage]. Each factory makes a fresh declaration and effect.
// Announcement fixes positive shares before payment. An empty group needs
// no shares. Copies keep shares by target position when targets change.
// Shared services keep legality, saved visits, phasing, prevention and
// simultaneous damage. Illegal recipients lose their shares; no new division
// is made. Modal, activated and variable-total RON forms stay separate.
//
// Use [ExileIfWouldDieThisTurn] for one saved permanent until cleanup.
// Destruction, sacrifice and other graveyard moves use shared replacements.
// Zero toughness permits this rule but does not permit regeneration.
// Mixed lethal and zero-toughness batches make choices before departure.
// Source departure and ability removal do not end the rule. Phasing keeps
// the saved visit; return cannot replace it. Clones keep separate rules.
// [Game.WaitingExileSources] retains its exact subject and departure facts.
// RON lowers APermanentWouldDie with SinglePermanent and ExileItInstead.
// Live damage-history sets use [ExileDealtDamageByThisTurn].
// Return-to-hand operations need separate services.
//
// Use [DeclareOpponentChosenDamageTarget] for an opponent target choice.
// Target legality uses the action controller and source. Copies keep the
// controller's permission to choose new targets. Separate groups can share
// one legal recipient. [PermanentFromDamageTarget] keeps the same declaration
// and saved incarnation. Players supply no permanent or permanent information.
// Phasing prevents actions; a return cannot replace the saved visit. Information
// reads can use exact departure LKI. RON lowers singular and numbered AnyTarget
// groups and AnyTargetOfAPlayersChoice(Opponent). AnyTargetAsAPermanent requires
// one singular union binding. Damage and prevention use shared typed services.
//
// RON no-land state triggers use [ControlsNone] with a current land-type filter.
// [WhenState] keeps one pending or active trigger per source visit and rule.
// A new land does not cancel a pending trigger. Countering permits a new trigger.
// Phasing and ability removal stop detection. Saved controllers and source visits
// keep pending sacrifice rules separate. Copies and clones use shared services.
//
// Use [PayRequiredOrElse] for two mandatory resolution-payment branches.
// [IfRequiredPaid] and [UnlessRequiredPaid] also accept uncaptured discard costs.
// Required groups permit discard and matching sacrifice costs only.
// The shared transaction keeps full-payment failure separate from live resources.
// No optional payment offer is made. Each operation keeps its own result.
// Use [DiscardOneOrElse] for an actual effect-caused discard outcome.
// Library destinations count; cancellation and returned hand visits do not.
// RON PlayerMustCost with DiscardACard uses this outcome in draw replacements.
// Ordinary and delayed bodies use resolution payments. Saved player bindings
// keep their values across that context change. Choices and Aura returns stay
// separate. Copies, clones and waiting queries use the shared typed contracts.
//
// Use [ReplaceNextDrawThisTurn] for one saved player's next draw until cleanup.
// The body keeps its resolution bindings, X, controller and source visit.
// Source departure, phasing and ability removal do not end the rule.
// Returned sources cannot replace the saved visit. Information reads use LKI.
// The affected player chooses among rules through the shared replacement pipeline.
// Each rule is consumed before its body runs. Generated draws keep event ancestry.
// Bodies run immediately and restore the outer frame. Clones keep separate use.
// RON lowers CreateFutureReplaceWouldDraw with a typed player and action list.
// Library selection and outside-game entry need separate services.
//
// Use [ReplaceDraw] with [Static] for battlefield draw replacement rules.
// [ControllerDrawStep] includes effect draws during the controller's draw step.
// [ExceptFirstDrawStepDraw] exempts the first actual draw in a player's draw step.
// Replaced proposals do not count. New steps reset history; clones keep it separate.
// The body keeps a local player binding and restores the enclosing resolution.
// Generated draws exclude that source visit and declaration through their ancestry.
// Copies can still replace them. Departure, phasing and removal stop the rule.
// Replaced draws supply no card result to the original instruction.
// RON lowers both ReplaceWouldDraw events and WouldDrawACard_ThatPlayer reads.
// Complete Sanctuary and Chains bodies and future replacements need more contracts.
//
// Use [WithUpkeepSpellTextContinuation] for one ordinary upkeep spell rule.
// The complete object marker and trigger layer marker must also be present.
// Any-player, controller and chosen-player events keep fresh entry bindings.
// Targets and fixed mana symbols keep their declarations. Copies enter as tokens;
// clones keep separate changes. Later text layers use the entry baseline.
// Phasing keeps the visit. Removal stops detection; pending bodies keep facts.
// Departure keeps final LKI and the physical printed card. Return uses printed
// rules. Fixed intrinsic keywords keep their counts and declaration groups.
// One opponent choice runs at entry. Other entry rules and mixed rules stay separate.
//
// Use [WithUpkeepObjectTextRevision] for one complete ordinary upkeep rule.
// [Game.AddPrintedUpkeepObjectTextEffect] checks all printed declarations.
// One opponent entry choice can supply the chosen-player context.
// Saved choices, names and mana symbols keep their values. Existing layers
// keep ordered baselines, expiry, clones, control, phasing, removal and LKI.
// Pending bodies keep detection facts; returned cards use printed rules.
// [ChangePrintedPermanentText] selects this context at resolution.
// Fixed intrinsic keywords keep their counts and declaration groups.
// Mixed rules and live copy need separate contracts.
//
// Use [GenericTriggered.WithLayeredUpkeepTextRevision] for printed upkeep rules.
// [Game.AddPrintedUpkeepTextEffect] changes one declaration in layer 3.
// Each pass restores its baseline before ordered color or basic land changes.
// Expiry restores surviving changes. Direct revision cannot change a layer result.
// RON marks ordinary any-player, controller and saved chosen-player contexts.
// Pending bodies keep prior text, targets, payment and captured event facts.
// Clones keep separate baselines, cleanup and controller contexts.
// Phasing keeps the visit. Removal stops detection; restoration keeps the text.
// Departure keeps final LKI. Return uses printed rules and new entry choices.
// Whole-object text, spell text and live copy rules need separate services.
//
// Use [GenericTriggered.WithUpkeepTextRevision] for an ordinary upkeep rule.
// [Game.RevisePrintedUpkeepTrigger] keeps its printed declaration identity.
// The event must use typed player capture on the battlefield.
// [ChosenPlayerUpkeep] keeps the source's saved entry choice at detection.
// Fresh rules keep the player context, targets and independent bindings.
// Pending bodies keep prior text and event facts. Phasing keeps the visit.
// Removal stops detection; restoration keeps revisions. Clones stay separate.
// Departure keeps pending bodies and LKI. Return uses printed rules.
// The caller must supply layer order, duration and baseline restoration.
// Other event contexts, spell text and whole-object text need separate services.
//
// Use [ChangePrintedSpellText] to apply saved word pairs to a pending spell.
// All pairs must resolve before the checked revision changes live rules.
// The action makes no choices. An absent spell cannot use departure information
// for this action. [Game.RevisePrintedSpellText] keeps declaration groups,
// ordered words, targets, payment and printed departure rules. RON lowers
// nonempty CreateSpellEffect lists with compatible lexical word references.
// Complete word-changing consumers use separate spell and permanent branches.
// General target contexts and live copy rules still need separate contracts.
//
// Use [Game.AddPrintedSpellCastObjectTextEffect] for one complete cast-event rule.
// The shared spell marker and trigger layer marker identify this context.
// Both definitions and the active declaration pass checks before installation.
// The nonland, non-Aura permanent must have one ordinary nonmodal trigger.
// Saved event facts, pending payments and fixed mana symbols keep their values.
// Shared services keep ordered layers, expiry, control, phasing, clones,
// removal, LKI and printed return. [ChangePrintedPermanentText] selects this
// context at resolution. Mixed rules and live copy rules stay separate.
//
// Use [Game.AddPrintedManaTapObjectTextEffect] for one complete global mana rule.
// The definition and declaration group pass checks before layer installation.
// The rule must have the shared mana-event spell marker and layer marker.
// Fixed mana symbols and event capture context keep their values.
// Pending triggers keep prior text and event facts. Shared services keep layer
// order, expiry, control, phasing, clones, removal, LKI and printed return.
// [ChangePrintedPermanentText] selects this context at resolution. Attached
// and mixed mana rules keep their separate contracts.
//
// Use [Game.AddPrintedLandAuraObjectTextEffect] for one complete land-type Aura.
// [LandEnchantTarget] keeps its enchant restriction fixed. Both definitions and
// every declaration group pass checks before layer installation. Fixed rules
// change printed output words. Saved entry choices stay fixed. Shared services
// keep ordered baselines, expiry, attachment identity, intrinsic mana, clones,
// control, phasing, removal, LKI and printed return. Spell entry text stays the
// baseline. Mutable enchant filters and live copy rules need separate contracts.
//
// Use [Game.AddPrintedProtectionPTTextEffect] for complete paired creature rules.
// One color protection rule shares text with one opponent nontoken color P/T rule.
// The recipient is the source or creatures with one fixed name. Both declarations
// pass checks before either layer is installed. Each keeps its baseline and ID.
// Ordered changes, expiry, clones, control, phasing, removal and LKI use shared
// services. Return and copy values use printed rules. Stack entry text stays
// the layer baseline. [ChangePrintedPermanentText] selects this context.
// Other mixed rules and live copy changes need separate contracts.
//
// Use [Game.AddPrintedStaticActivationTextEffect] for paired creature rules.
// One global or basic-land source P/T rule shares text with one ordinary action.
// Both declarations pass checks before either layer is installed. They keep
// separate baselines, IDs, targets, payments and pending bodies. RON uses the
// same complete declaration marker for stack and battlefield text. Clones,
// copies, expiry, phasing, removal, LKI and printed return use shared services.
// Other paired rules and live copy changes need separate contracts.
//
// Use [Game.AddPrintedAttachedObjectTextEffect] for one complete creature Aura.
// [CreatureEnchantTarget] keeps its enchant restriction fixed. RON marks one
// host keyword, landwalk or Ward grant, or one fixed-mana host action.
// The whole definition and declaration groups pass checks before installation.
// Existing text layers keep ordered baselines, expiry and active declaration IDs.
// Ward protection keeps the exact donor exception. Actions read the current host
// at resolution, or its saved attachment after departure. Returned hosts cannot
// replace that visit. Clones, copies, phasing, removal, LKI and printed return
// use shared services. Mixed Auras and spell text need separate contracts.
//
// Use [Game.AddPrintedMixedStaticEventTextEffect] for complete paired rules.
// One global static rule can share text with a controller mana bonus or a fixed
// controller-upkeep payment and unpaid self-sacrifice. Both rules pass preflight
// before any layer is installed. Mana symbols and pending upkeep bodies stay fixed.
// Static and mana rules keep separate ordered baselines and declaration IDs.
// Shared services keep expiry, clones, control, phasing, removal and final LKI.
// Return and copy values use printed rules. [ChangePrintedPermanentText] selects
// this context at resolution. Attached and other mixed contexts stay separate.
//
// Use [Game.AddPrintedActivatedObjectTextEffect] for complete ordinary actions.
// Every printed ability must permit layered activation text revision. Fixed
// intrinsic keywords can stay. [ChangePrintedPermanentText] checks this context.
// All declarations pass preflight before installation. Each keeps its baseline,
// targets, payments, modes, limits, history and pending bodies. Shared services
// keep ordered layers, expiry, clones, control, phasing, removal and LKI.
// Return uses printed rules. Mixed, attached and live copy rules stay separate.
//
// Use [Game.AddPrintedGlobalGrantTextEffect] for complete global grants.
// Every printed rule must be a marked grant. Fixed intrinsic keywords can stay.
// [ChangePrintedPermanentText] checks this context before it installs layers.
// Each donor keeps its declaration groups, ordered baselines and live subjects.
// Activated grants keep recipient identity, limits, costs and pending bodies.
// Granted recipient words stay separate. Clones, phasing, removal, expiry,
// LKI and printed return use shared services. Mixed and attached rules and
// live copy changes need separate contracts.
//
// Use [Game.AddPrintedGlobalStaticTextEffect] for one complete global rule.
// It checks the retained definition before it changes a printed P/T or land rule.
// [ChangePrintedPermanentText] reads saved words and selects a supported context.
// Intrinsic creature rules keep their existing contract. Mixed rules, attached
// rules and live copy changes need separate contracts.
// Layer order, expiry, clones, phasing, LKI and printed return use shared services.
//
// Use [Game.AddPrintedIntrinsicCreatureTextEffect] for complete intrinsic rules.
// It checks all printed protection and basic landwalk rules before installation.
// Ordinary keywords, names, mana symbols and grants keep their values.
// Each declaration keeps its ID, ordered changes and restoration baseline.
// Phasing keeps the visit. Clones keep separate layers. Removal applies in layer 6.
// LKI keeps final rules; return uses printed rules. Copy values exclude changes.
// [ChangePrintedIntrinsicCreatureText] reads typed word pairs at resolution.
// RON lowers resolving CreatePermanentLayerEffect lists of word substitutions.
// Other rule contexts and live copy changes require separate contracts.
//
// Use [WithGlobalStaticSpellTextContinuation] for one global ability grant.
// The holder must use [StaticAbilityHolder.WithLayeredGrantTextRevision].
// Fixed grants and paired P/T keep changed words through checked spell entry.
// Fresh rules keep declaration groups and ordered effect layers and durations.
// Shared subjects remain live. Later text layers use the continued baseline.
// Copies and clones keep separate rules. Phasing keeps the visit. Removal stops
// the donor. LKI keeps final results; recasts use printed rules. Copy values
// exclude ordinary word changes. Other mixed and whole-object rules stay separate.
//
// Use [WithStaticActivationSpellTextContinuation] for paired creature rules.
// One global P/T rule or basic-land source P/T condition can share a spell
// with one ordinary activated rule. Checked revision keeps both groups and
// target contexts. Entry builds fresh bindings. Mana symbols stay fixed.
// Copies and clones keep separate rules. Later text layers restore the entry
// baseline. Phasing keeps the visit. Removal stops both rules. Departure keeps
// LKI; recasts use printed rules. Other mixed contexts need separate contracts.
//
// Use [WithProtectionPTSpellTextContinuation] for paired creature rules.
// One color protection rule and one opponent nontoken color P/T rule keep
// changed words through checked stack revision and fresh permanent entry.
// The P/T recipient is the source or creatures with one fixed name.
// Declaration groups keep each sibling separate. Mana symbols stay fixed.
// Copies and clones keep separate rules. Later text layers restore the entry
// baseline. Phasing keeps the visit; removal suspends both rules.
// Departure keeps final LKI. Recasts use printed rules. Ordinary changes stay
// outside copiable values. Other mixed rules need separate contracts.
//
// Use [WithGlobalStaticSpellTextContinuation] for one conditional source P/T rule.
// The condition must check the controller or opponent for one basic land type.
// [Game.RevisePrintedSpellText] keeps that condition through fresh entry.
// Changes must keep the condition side and source recipient. Costs stay fixed.
// Copies and clones keep separate rules. Later text layers restore the entry
// baseline. Phasing keeps the visit. Removal stops the rule. Departure keeps
// LKI; return uses printed rules. Mixed rules need separate contracts.
//
// Use [WithIntrinsicCastEventSpellTextContinuation] for paired creature rules.
// RON marks intrinsic protection/basic landwalk with one battlefield cast trigger.
// [Game.RevisePrintedSpellText] checks both groups before fresh entry builds.
// Mana symbols keep their values. Copies and clones keep separate rules.
// Later text layers restore the continued baseline. Phasing keeps the visit.
// Pending triggers keep detection text and controller. LKI keeps final rules;
// return uses printed rules. Ordinary changes stay outside copiable values.
// Other mixed rules and whole-object text require separate contracts.
//
// Use [WithMixedStaticEventSpellTextContinuation] for paired permanent rules.
// It permits one global static rule with a controller mana bonus or a fixed
// controller-upkeep mana payment and unpaid self-sacrifice. Both declarations
// keep their groups through checked stack revision and fresh entry builds.
// Mana symbols keep their values. Copies and clones keep separate rules.
// Later layers start from the continued baseline. Phasing keeps the visit.
// Departure keeps LKI; the physical card returns to its printed rules.
// Other mixed rules and whole-object text require separate contracts.
//
// Use [WithManaTapSpellTextContinuation] for one global land mana event rule.
// [Game.RevisePrintedSpellText] keeps fresh rules through permanent entry.
// RON marks one AnyPlayer TriggerA land event or its produced-type bonus.
// Each build has fresh event bindings and targets. Mana bonuses use the shared
// planner and production service. Other triggers keep their stack timing.
// Copies enter as tokens. Clones keep separate rules. Phasing keeps the visit.
// Later text layers use the continued baseline. Removal stops the rule.
// Pending triggers keep prior rules and event facts. Copy values exclude word
// changes. LKI keeps final rules; returned cards use printed rules.
// Mixed, attached and other spell rules require separate contracts.
//
// Use [WithSpellTextContinuation] for one ordinary instant or sorcery action.
// [Game.RevisePrintedSpellText] changes its rules during the exact stack visit.
// RON marks its ordinary SpellActions path without additional casting costs.
// Fresh effects keep announced target groups, X, payment and cast facts.
// Revised targets can become illegal. Resolution results use fresh bindings.
// Copies use the retained target declarations. Clones keep separate revisions.
// The physical card keeps printed rules on departure. Invalid changes are atomic.
// [ModalSpellAbility.WithTextContinuation] marks a choose-one spell.
// Each revision checks all modes and keeps the selected mode and its targets.
// Copies use current mode declarations. Results and clones keep separate state.
// [WithSpellSacrificeManaValueContinuation] marks one saved sacrifice mana value.
// The additional cost and spell body share its binding. Rebuilt rules use a
// fresh binding with the paid value. Copies and clones keep separate maps.
// Revision does not repeat payment or read a returned sacrificed card.
// Other permanent spells and casting results need separate services.
// Whole-object text dispatch and layer installation need separate services.
//
// Use [WithSpellCastSpellTextContinuation] for one typed cast-event spell rule.
// [Game.RevisePrintedSpellText] keeps fresh rules through permanent entry.
// RON marks one battlefield TriggerA with an AnyPlayer or Opponent cast event.
// Other rules, keywords and casting costs or targets need separate contracts.
// Entry keeps physical identity, owner and printed departure rules.
// Copies enter as tokens. Clones keep separate rules and payment results.
// Event bindings and trigger targets belong to each fresh declaration.
// Phasing keeps the visit. Later text layers use the continued baseline.
// Removal stops detection. Pending triggers keep their rules and event facts.
// Copy values exclude ordinary word changes. LKI keeps the final rules.
// Return uses printed rules. [Game.AddPrintedSpellCastObjectTextEffect] checks
// complete battlefield rules. Mixed rules and live copy rules stay separate.
//
// Use [WithGlobalStaticSpellTextContinuation] for one global static spell rule.
// [Game.RevisePrintedSpellText] keeps fresh rules through permanent entry.
// This context marks one fixed creature P/T or basic-land-type rule.
// Casting targets, costs, modes and other rules need separate contracts.
// Entry keeps physical identity, owner and printed departure rules.
// Copies enter as tokens. Clones keep separate rules. Phasing keeps the visit.
// Control and subjects remain live. Later text layers use the continued baseline.
// Expiry restores it. Ability removal keeps the existing layer order.
// Copy values exclude ordinary word changes. LKI keeps the final rules.
// Return uses printed rules. Mixed, sticky and whole-object rules remain separate.
//
// Use [WithCopyEntrySpellTextContinuation] for one optional entry-copy spell.
// [Game.RevisePrintedSpellText] retains ordered words through donor acquisition.
// [BuildEntryCopyWithText] checks fresh donor rules and declaration groups.
// Each entry keeps its own changes through nested choices and copy exceptions.
// Acquired entry rules and triggers use the changed text. Copy values exclude it.
// Missing donor builders and invalid groups return errors. Clones keep separate
// rules. Spell copies enter as tokens. Phasing keeps the visit and its text.
// Later text layers restore the continued baseline. Departure keeps printed
// physical rules and final LKI. Other permanent rules need separate contracts.
//
// Use [WithIntrinsicCreatureSpellTextContinuation] for intrinsic creature rules.
// [Game.RevisePrintedSpellText] carries fresh protection and basic landwalk
// through entry. RON marks creature rules with these declarations and ordinary
// keywords. Each declaration keeps its contribution. Copies enter as tokens.
// Clones keep separate rules. Phasing keeps the visit. Later text layers restore
// the continued baseline on expiry. Departure keeps LKI and printed rules.
// [Game.CaptureCopiableValues] checks the saved entry characteristics, then
// returns printed copy values. Ordinary word changes stay outside those values.
// Copy entry, other rules and whole-object word changes need separate services.
//
// Use [WithLandTypeAuraSpellTextContinuation] for a land-type Aura spell.
// [Game.RevisePrintedSpellText] keeps its announced target and cast payment.
// RON marks one enchant rule and one fixed or chosen host land-type rule.
// A chosen rule requires one basic-land-type entry choice. Entry uses fresh
// revised rules with physical identity and printed departure information.
// Choices happen at entry and are not printed words. Copies enter as tokens
// and choose separately. Clones keep separate rules. Phasing keeps the visit.
// Later text layers use the continued baseline. Departure keeps LKI and
// restores physical printed rules. Ordinary changes stay outside copy values.
// Copy-entry, other Aura rules and whole-object text need separate services.
//
// Use [WithLayeredTextRevisionActivationState] for a printed activated action.
// [Game.AddPrintedActivatedTextEffect] changes one declaration in layer 3.
// Each pass restores its baseline before ordered color or basic land changes.
// Expiry recomputes surviving changes. Mana symbols keep their printed values.
// Costs, targets, choices and results use fresh declarations. Pending bodies keep
// prior text. Limits and history keep the same declaration and source visit.
// Cleanup resets limits even while an ability is removed or phased out.
// Clones keep separate baselines, state and controller contexts.
// Layer 6 removal stops activation; restoration keeps current text and limits.
// LKI keeps the final rule. Return uses printed rules. RON marks its ordinary
// and modal Activated and ActivatedWithModifiers action contexts.
// Conditional mana, whole-object text and spell text need separate services.
//
// Use [GenericTriggered.WithLayeredLandTapTextRevision] for printed land events.
// [Game.AddPrintedLandTapTextEffect] changes one declaration in layer 3.
// Each pass restores its baseline before ordered color or basic land changes.
// Expiry restores surviving changes. Direct revision cannot change a result.
// RON marks its complete typed battlefield opponent land-tap trigger context.
// Pending triggers keep earlier rules, targets and event facts.
// Clones keep separate baselines, cleanup and controller contexts.
// Phasing keeps the visit. Removal stops detection; restoration keeps the text.
// LKI keeps the final rule. Return uses printed rules.
// Whole-object text, spell text and copy rebinding need separate services.
//
// Use [GenericTriggered.WithLayeredCombatPairTextRevision] for printed pair events.
// [Game.AddPrintedCombatPairTextEffect] changes one declaration in layer 3.
// Each pass restores its baseline before ordered color or basic land changes.
// Expiry recomputes surviving changes. Direct revision cannot change a result.
// RON opts in its complete source and Aura-host TriggerA combat-pair context.
// Directions and captures stay fixed. Pending targets and delays keep prior text.
// Clones keep separate baselines, cleanup and controller contexts.
// Phasing keeps the visit. Removal stops detection; restoration keeps the text.
// LKI keeps the final rule. Return uses printed rules.
// Whole-object text, spell text and copy rebinding need separate services.
//
// Use [GenericTriggered.WithLayeredSpellCastTextRevision] for printed cast events.
// [Game.AddPrintedSpellCastTextEffect] changes one declaration in layer 3.
// Each pass restores its baseline before ordered color or basic land changes.
// Expiry restores surviving changes. Direct revision cannot change a layer result.
// RON marks its complete typed battlefield spell-cast trigger context.
// Pending triggers keep earlier rules, targets, payments and event facts.
// Clones keep separate baselines, cleanup and controller contexts.
// Phasing keeps the visit. Removal stops detection; restoration keeps the text.
// LKI keeps the final rule. Return uses printed rules.
// Complete object text uses [Game.AddPrintedSpellCastObjectTextEffect].
// Mixed rules and copy rebinding need separate services.
//
// Use [StaticAbilityHolder.WithLayeredAnimationTextRevision] for printed animation.
// [Game.AddPrintedStaticAnimationTextEffect] changes one declaration in layer 3.
// Each pass restores its baseline before ordered color or basic land changes.
// RON opts in complete static animation with fixed P/T and optional colors.
// The changed rule keeps one selection through type, color and base P/T layers.
// Expiry restores surviving changes. Direct revision cannot change layered text.
// Clones keep separate baselines, cleanup and controller contexts.
// Phasing keeps the visit. LKI keeps final results; return uses printed rules.
// Whole-object text, spell text and copy rebinding need separate services.
//
// Use [StaticAbilityHolder.WithLayeredLandTypeTextRevision] for a printed land rule.
// [Game.AddPrintedStaticLandTypeTextEffect] changes one declaration in layer 3.
// RON opts in its global, fixed-host and chosen-host basic-land-type contexts.
// Each pass restores its baseline before ordered color or basic land changes.
// Saved entry choices do not change. Layer 4 keeps live subjects, dependencies
// and intrinsic mana. Layer 6 cannot undo an earlier type change.
// Expiry restores surviving changes. Direct revision cannot change a layered rule.
// Clones keep separate baselines, cleanup and controller contexts.
// Phasing keeps the visit. LKI keeps final types; return uses printed rules.
// Whole-object text, spell text and copy rebinding need separate services.
//
// Use [ProtectionAbility.WithLayeredColorTextRevision] for printed protection.
// [Game.AddPrintedProtectionTextEffect] changes one declaration in layer 3.
// It accepts color word changes with EndOfTurn or Indefinite duration.
// Each pass restores the baseline before ordered changes. Expiry removes them.
// Layer 6 can remove protection. Restoration uses the saved declaration.
// Clones keep separate effects and controller contexts. Phasing keeps the visit.
// Departure LKI keeps the result. Return uses printed rules. Grants stay separate.
// RON opts in top-level nonempty color protection without an Aura exception.
// Direct revision cannot change a declaration while its layered result is present.
// Whole-object text, spell text and Ward donor text need separate services.
//
// Use [WithLayeredPrintedBasicLandwalk] for checked layer 3 landwalk changes.
// [Game.AddPrintedBasicLandwalkTextEffect] selects one printed declaration.
// It accepts basic land word changes with EndOfTurn or Indefinite duration.
// Each layer pass restores the baseline, then applies changes in timestamp order.
// Expiry removes the change from later passes. Layer 6 can remove the keyword.
// Granted keywords and sibling declarations stay separate. Clones keep separate
// effects and baselines. Phasing keeps the visit; the next present layer pass
// recomputes the text. Departure LKI keeps the last result. Return uses printed
// rules. Direct revision of the layered declaration is rejected while its layer
// result is present. RON opts in top-level fixed basic landwalk declarations.
// Whole-object text changes, copy-layer rebinding and spell text are separate.
//
// Use [CantBeTargetedByAbilities] with [Static] for a live target restriction.
// Its filter reads the ability source. Spell targets use a separate rule.
// [This] and [AttachedPermanent] keep source and host identity checks.
// Source departure, phasing and ability removal stop the restriction.
// Pending triggers, resolution and copies use current or retained information
// for their exact source. Returned objects cannot supply earlier source facts.
// RON supports card-type and color source filters with nonempty And or Or lists.
// It permits one static restriction or a pair with a spell target restriction.
// Target-declaration filters and conditional rules need separate contracts.
//
// Use [LandManaBonusThisTurn] for one fixed bonus from matching lands.
// The shared mana query keeps planning and production in agreement.
// Each positive tap-symbol mana activation adds one bonus to its player.
// Current land types control membership, including later and nonbasic lands.
// Resolved rules survive source departure, phasing and ability removal.
// Cleanup ends them. Clones keep separate registrations and cleanup.
// RON supports one known land subtype, concrete mana type and UntilEndOfTurn.
// Its printed subtype bonus uses [NewControllerManaBonusAbility].
// That bonus pays the land controller and permits checked text revision.
// General filtered events need separate adapters.
//
// Use [Game.AddPrintedManaTapTextEffect] for one printed mana event in layer 3.
// [GenericTriggered.WithLayeredManaTapTextRevision] marks a typed land event.
// [ManaBonusAbility.WithLayeredManaTapTextRevision] marks a shared mana bonus.
// Each pass restores its baseline before ordered color or basic land changes.
// Expiry recomputes surviving changes. Direct revision cannot change a result.
// Mana symbols keep their values. Planning and production use the same rule.
// Pending triggers keep prior text and event facts. IDs and positions stay fixed.
// Clones keep separate rules and controller contexts. Phasing keeps the visit.
// Removal suspends the rule; restoration keeps its text. LKI keeps the result.
// Return uses printed rules. RON opts in its complete land-mana event contexts.
// Whole-object text, spell text and copy rebinding need separate services.
//
// Use [Game.RevisePrintedManaTapTrigger] for a printed mana-tap declaration.
// [GenericTriggered.WithManaTapTextRevision] opts in typed land mana events.
// [ManaBonusAbility.WithManaTapTextRevision] opts in shared mana bonuses.
// Fresh validated rules keep the active ID and declaration position.
// Queued bodies keep prior text and event facts. Each declaration keeps its
// ordered changes. Clones and sibling declarations keep separate revisions.
// Shared mana services keep payment planning and production together.
// Phasing keeps revisions; restoration uses them; return uses printed rules.
// RON opts in its complete land mana event and bonus contexts.
// Mana symbols keep their printed values. Marked declarations use
// [Game.AddPrintedManaTapTextEffect] for layer order, lifetime and restoration.
// Other event adapters and spell changes need separate services.
//
// Use [StaticAbilityHolder.WithLandTypeTextRevision] for a printed land rule.
// [Game.RevisePrintedStaticLandType] checks its declaration and source visit.
// Exactly one global or attached effect must set a basic land type in layer 4.
// An attached effect can use its source's saved entry choice. A revision must
// keep its global, fixed-host or chosen-host context. A saved choice is not a
// printed land word. Revision does not change it or repeat the entry operation.
// RON opts in unconditional global and Aura-host SetLandType lists with one
// fixed type, and Aura-host SetLandTypeVariable with a valid entry declaration.
// Input and output words change in order. Fresh rules keep the active ID and
// registration position. The shared land service keeps intrinsic mana, printed
// ability removal and type dependencies. Clones and siblings stay separate.
// Phasing keeps revisions; restoration uses revised rules. Return uses printed
// rules. [Game.AddPrintedStaticLandTypeTextEffect] supplies layer installation
// for marked declarations. Temporary rules and spell changes need separate services.
//
// Use [StaticAbilityHolder.WithAnimationTextRevision] for a printed animation.
// [Game.RevisePrintedStaticAnimation] checks its declaration and source visit.
// All ordered effects must come from one [AnimateEach] rule. Fresh rules keep
// the active ID, registration positions, layers and battlefield duration.
// Each pass keeps the same subjects through type, color and base P/T layers.
// Clones and sibling declarations keep separate changes. Phasing keeps them.
// Ability restoration uses revised rules. Return restores printed rules.
// RON opts in its complete unconditional static animation context.
// [Game.AddPrintedStaticAnimationTextEffect] supplies layer installation
// for marked declarations. Sticky rules and spell changes need separate services.
//
// Use [WithPrintedBasicLandwalk] for a printed basic-landwalk declaration.
// [Game.RevisePrintedBasicLandwalk] checks its active identity and source visit.
// It changes only that declaration's intrinsic attribute contribution.
// Duplicate declarations, granted keywords and other base attributes stay separate.
// Clones keep separate changes. Phasing keeps them; removal suspends them.
// Restoration uses revised rules. LKI keeps them; return uses the printed origin.
// RON opts in top-level Landwalk with one fixed basic land type.
// Desert and legendary landwalk keep their existing keyword representation.
// The caller must supply text-layer order, lifetime and original-text restoration.
// Other intrinsic keywords and full live text installation need separate services.
//
// Use [ProtectionAbility.WithColorTextRevision] for printed color protection.
// [Game.RevisePrintedProtection] checks its declaration and source visit.
// It keeps the active ID with fresh rules and ordered semantic changes.
// Granted protection and Aura exceptions cannot use this service.
// Clones and sibling declarations keep separate changes. Phasing keeps them.
// Ability restoration uses revised rules; return uses the printed origin.
// Shared protection rules check targeting, damage, blocking and attachment.
// RON opts in top-level Protection with a nonempty FromColor color list.
// The caller must supply text-layer order, lifetime and original-text restoration.
// Other intrinsic keywords and full text consumers need separate services.
//
// Use [StaticAbilityHolder.WithGrantTextRevision] for a printed static grant.
// [Game.RevisePrintedStaticGrant] checks its source visit and declaration.
// Effects must use layer 6, with optional P/T modifications in layer 7c.
// Each effect must have battlefield duration. Fresh effects keep each layer,
// sublayer, duration and registration position. Eligible RON globals use
// [WhileWithSharedSubjects] to keep members through all grants in one pass.
// A started shared rule continues after later ability removal. Each new pass
// reads current membership. Clones and sibling declarations stay separate.
// Phasing suspends rules; restoration keeps revisions. Return uses printed rules.
// The caller must supply text-layer order, lifetime and original-text restoration.
// Conditional rules, other layers and full text consumers need separate services.
//
// Use [StaticAbilityHolder.WithLayeredGrantTextRevision] for a printed global grant.
// [Game.AddPrintedStaticGrantTextEffect] changes one declaration in layer 3.
// RON opts in unconditional keyword and landwalk grants with optional fixed P/T.
// Each pass restores its saved rule before ordered color or basic land changes.
// Expiry restores the prior rule. Direct revision cannot change a layered result.
// Shared subjects keep each grant and its P/T part together in layers 6 and 7c.
// Control and membership remain live. Clones and declarations stay separate.
// Phasing keeps the visit; removal suspends the grant; restoration keeps its text.
// LKI keeps the final grant. Return uses printed rules. Other text consumers and
// copy-layer replacement need separate services.
//
// Use [StaticAbilityHolder.WithLayeredAbilityTextRevision] for a printed donor rule.
// [Game.AddPrintedStaticAbilityTextEffect] installs one declaration in layer 3.
// RON opts in unconditional Aura-host keyword, landwalk and Ward grants.
// Each pass restores its baseline before ordered color or basic land changes.
// Expiry recomputes surviving changes. Prior direct revision can be the baseline.
// Direct revision cannot change a layered result. Grants apply in layer 6.
// Ward protection keeps its exact Aura exception. Ability removal suppresses
// the donor; restoration uses its registered rule. Phasing keeps the visit.
// Clones keep separate registrations, baselines and controller contexts.
// Departure LKI keeps the last grant. Return uses printed rules.
// Full object text, spell text and copy-layer replacement need separate services.
//
// Use [StaticAbilityHolder.WithAbilityTextRevision] for a printed ability-layer rule.
// [Game.RevisePrintedStaticAbility] checks its declaration and source visit.
// It replaces fresh effects with the same active ID and registration positions.
// Each effect must use layer 6 with battlefield duration. Clones stay separate.
// RON opts in unconditional Aura-host keyword, landwalk and Ward grants.
// The exact granting Aura keeps its protection exception after each revision.
// Phasing suspends the grant. Removal and restoration keep revised rules.
// Return restores printed rules. The caller must supply text-layer order and
// lifetime. Other layers, conditional rules and full text consumers need
// separate services.
//
// Use [StaticAbilityHolder.WithLayeredPTTextRevision] for a printed P/T rule.
// [Game.AddPrintedStaticPTTextEffect] changes one declaration in layer 3.
// RON opts in unconditional creature collections with one fixed AdjustPT.
// Typed controller, opponent and host conditions use the same text layer.
// Their conditions and subjects read current state in layer 7c.
// Each pass restores its saved baseline before ordered word changes.
// Current subjects receive P/T in layer 7c. Expiry restores surviving changes.
// Direct revision cannot change a layered result. Clones keep separate rules
// and controller contexts. Phasing keeps the visit. Removal suspends the rule;
// restoration keeps its text. LKI keeps final P/T. Return uses printed rules.
// Other conditional contexts, layers and full text consumers need separate services.
//
// Use [StaticAbilityHolder.WithPTTextRevision] for a printed static P/T rule.
// [Game.RevisePrintedStaticPTAbility] checks its declaration and source visit.
// It keeps the active ID and registration order with fresh validated effects.
// Each effect must modify P/T while its source is on the battlefield.
// Each declaration keeps its own ordered text changes. Clones stay separate.
// Phasing suspends the rule. Ability removal suspends it; restoration keeps
// the revised declaration. A returned source uses its printed rules.
// RON opts in global creature collections with one fixed AdjustPT action.
// [SourcePTWhileControllerControlsBasicLand] adds one conditional source rule.
// Its controller and basic land condition remain live at each layer pass.
// [SourcePTWhileOpponentControlsBasicLand] uses the current opponent's lands.
// [SourcePTWhileOpponentControlsNontokenColor] uses nontoken color membership.
// [NamedCreaturesPTWhileOpponentControlsNontokenColor] affects named creatures
// under either player. Each typed conditional rule must contain one effect.
// Revision keeps its condition kind, recipient kind and fixed recipient name.
// Land types and colors remain live after their earlier layers.
// [CreaturesPTWhileAttachedToBasicLand] adds a current-host Basic condition
// and a global creature-subtype collection. Revision keeps that subtype.
// The host uses attachment identity; returned hosts cannot replace it.
// The caller must supply text-layer order and lifetime. Other static layers,
// other conditional or Aura rules and full text consumers need separate services.
//
// Use [GenericTriggered.WithCombatPairTextRevision] for a printed pair trigger.
// [Game.RevisePrintedCombatPairTrigger] checks its declaration and source visit.
// The typed event must keep its block direction and battlefield source.
// It keeps the active ID with fresh filters, targets and opponent bindings.
// Pending triggers, copies and delayed bodies keep their prior text and facts.
// Phasing keeps revisions. Ability restoration uses the revised declaration.
// Clones and sibling declarations keep separate changes. Return uses printed rules.
// RON opts in its source and Aura-host TriggerA combat-pair path.
// The caller must supply text-layer order, lifetime and original-text restoration.
// Grouped events, creature-type substitutions and full text consumers are separate.
//
// Use [GenericTriggered.WithLandTapTextRevision] for a printed land-tap trigger.
// [Game.RevisePrintedLandTapTrigger] checks its declaration and source visit.
// RON opts in its typed opponent land-subtype tap event. The active ID stays.
// Pending bodies keep earlier text, targets and captured event identities.
// Phasing keeps revisions. Ability restoration uses the revised declaration.
// Clones keep separate later changes. Return restores printed rules.
// [Game.AddPrintedLandTapTextEffect] supplies layer installation for marked rules.
// Other event kinds and complete text consumers need separate services.
//
// Use [GenericTriggered.WithSpellCastTextRevision] for a printed spell-cast trigger.
// [Game.RevisePrintedSpellCastTrigger] checks its declaration and source visit.
// It keeps the active ID with fresh rules, targets and event bindings.
// Detected triggers and copies keep their earlier text and captured players.
// Each active declaration keeps its own ordered changes. Clones stay separate.
// Phasing keeps a revision; returned sources cannot inherit it.
// RON lowers the option for its typed battlefield spell-cast trigger path.
// The caller must supply text-layer order, lifetime and ability restoration.
// Other trigger kinds, static registrations and spell text need separate services.
//
// Use [WithTextRevisionActivationState] for a printed activated action.
// [Game.RevisePrintedActivatedAbility] checks its declaration and source visit.
// It keeps the active ID, limits and activation history with fresh rules.
// Queued bodies keep their earlier targets and result bindings. Cleanup resets
// limits and history; phasing keeps the revised action. Returned sources cannot
// inherit it. RON lowers this option on its ActionDefinition activation path.
// The caller must supply text-layer order and lifetime. Other active ability
// kinds, spell text and full source text installation need separate services.
//
// Use [BuildTextDefinitionRevision] for checked ability correspondence.
// [WithDefinitionAbilityGroup] keeps declaration groups across fresh builds.
// [TextDefinitionRevision.ReplacementAbility] pairs original ability IDs with
// new declarations. Targets, choices, source and controller remain separate.
// RON assigns groups by rule position; count definitions use a separate group.
// Copy omissions keep the other members in their original groups.
// This preparation does not install rules on a present object.
//
// Use [BuildTextChangedDefinition] for an independent text prototype.
// [ReplaceColorWord] and [ReplaceBasicLandWord] require different legal words.
// RON retains a private AST and changes only validated semantic word roles.
// Names, mana symbols and color indicators keep their printed values.
// Each build has fresh declarations. Successive changes keep their order.
// Printed copy information stays separate. Legacy rule replacement clears
// the text builder. Unknown matching word roles return an error.
// [ColorWordSubstitution] and [BasicLandWordSubstitution] read typed pairs
// from the current resolution frame without making choices.
// This builder does not install a text layer or change a present object.
// Live timestamps, ability instances and spell entry need separate services.
//
// Use [ResolveObjectTarget] for two resolving bodies with one union target.
// [SpellFromObjectTarget] and [PermanentFromObjectTarget] share its declaration.
// The saved category selects one body at resolution, without a mode choice.
// Each body keeps its own result declarations. The target must still be present
// when the body starts. Later instructions can read its departure information.
// Copies can choose a target in the other category. RON supports unfiltered
// spell/permanent branches in nonpermanent spell actions. Text effects require
// separate adapters. Existing Lace color forms keep their validated behavior.
//
// Use [SacrificeSourceCostWithCard] or [ExileSourceCostWithCard] for an
// activated ability that needs its paid source card. The payment saves its
// first new public-zone visit before zone events. Tokens supply no card.
// The transaction saves the source incarnation and checks it before payment.
// [FromCard] cannot follow a later visit. Copies and delays keep the result;
// clones keep separate cards and owners. RON ownership exchanges accept
// ThisPermanent after one source payment or one singular source exile group.
// This reference does not permit movement from anywhere.
//
// Use [ChooseColorWordPair] or [ChooseBasicLandWordPair] for two different words.
// The selected player makes both choices at resolution. Typed first and second
// references read one saved pair. Missing choosers clear the result.
// The second choice excludes the first. Invalid answers use a legal default.
// Resolution frames, copies, clones, loops and delays keep separate results.
// Waiting queries retain typed chooser source reads without making choices.
// RON supports paired choice symbols in resolving action and player bodies.
// Text changes require separate consumers and layer contracts.
//
// Use [PlayerManaPaymentThisTurn] for a repeatable player payment at instant timing.
// The player and source references are saved when the permission starts.
// [Game.GetPlayerPayments] lists affordable permissions without making choices.
// [Game.TakePlayerPayment] uses the shared cost transaction and no stack object.
// Each payment uses a separate copy of the saved resolution frame.
// Source departure and phasing do not end the permission. Cleanup ends it.
// [PriorityTakePlayerPayment] permits use through the priority handler.
// RON supports fixed mana payments and UntilEndOfTurn in resolving bodies.
// Life-to-mana, one-use casting and other durations need separate adapters.
//
// Use [ExilePermanentGroupsInto] with distinct [PermanentExileGroup] results.
// All groups select their current subjects before one simultaneous exile batch.
// Overlapping subjects move once and supply the same visit to each group.
// Results use [CardsBinding] in the resolution frame. Empty groups clear results.
// [ExiledCardsFrom] permits transport of the saved current exile visits.
// [SingleExiledCardFrom] requires one original result and public card visibility.
// A plural result cannot become a singular reference after a member leaves.
// Returned and phased permanents cannot supply a move. Tokens leave the
// battlefield but supply no retained exile card. Results are saved before events.
// Delays and clones retain separate result storage. Waiting queries declare
// explicit source reads without selecting subjects or moving permanents.
// RON supports plural permanent subject groups and lexical result references.
// Counter notes, ownership exchange and return attachments need separate adapters.
//
// Use [NewExileLink] for a distinct group of linked exile declarations.
// [ExilePermanentLinked] and [ExileLibraryTopLinked] save that declaration
// with the exact source incarnation on the new exile visit before zone events.
// [CardsExiledWith] selects only current visits from that link and source.
// [ExiledCardSet] permits transport without a card inspection interface.
// [CountExiledCards] counts these visits without reading characteristics.
// Source departure, control changes and phasing keep an announced instruction.
// A returned source or re-exiled card cannot inherit an earlier link.
// Face-down library exile grants no inspection permission. Engine transport
// can select these cards without exposing their characteristics to players.
// Clones keep the declaration but copy exile cards and visibility storage.
// Waiting queries retain source reads without moving cards or reading a library.
// [ReturnExiledCardsToOwnersHands] and [PutExiledCardsInOwnersGraveyards]
// move selected visits to their current owners' zones before zone events.
// Graveyard order uses the existing owner choices before movement.
// RON associates one unambiguous producer with source-local consumers in
// resolving permanent bodies. Knowledge Vault uses both destinations.
// [PutExiledCardsOntoBattlefieldUnderOwnersControl] uses the shared entry pipeline.
// Entry choices use active-player order before movement. Saved visits and Aura
// hosts are checked again before all successful entries reach the battlefield.
// Prohibited entries keep their exile visits. Failed entry operations keep the
// existing graveyard destination. Copy, counter and tapped entry rules apply.
// No casting payment or announced X transfers to this entry.
// Multiple producer groups and explicit entry attachments need separate adapters.
//
// Use [WithAnteDeckRestriction] for a physical card that requires ante.
// [PrepareCardsForAnte] separates these cards from non-ante decks and sideboards.
// The results keep card identity, ownership and order, with separate slice storage.
// [NewGame] removes these cards from initial libraries before game actions.
// [NewGameWithAnte] keeps them eligible for explicit initial ante selection.
// Copies of game characteristics do not change the physical deck restriction.
// This pregame contract does not permit outside-game entry or execute ante actions.
//
// Use [AnteTopFor] or [AnteTopEach] for library-top ante actions.
// [Game.AnteLibraryTops] moves all eligible tops before zone events.
// Empty libraries, tokens and cards owned by another player cannot contribute.
// Duplicate players contribute once. Disabled or settled ante games do not move
// cards. Each action reads current tops at resolution and keeps physical identity.
// Typed player operands use the existing bindings, LKI and waiting declarations.
// Legacy [AnteLibraryTop] uses the same batch service.
//
// Use [AnteTopCost] for a library-top payment through the cost transaction.
// The transaction saves each top before payment and checks the full payment
// before it spends live resources. Queries do not move cards or offer choices.
// Use [PayPermanentAnteOrElse] for an optional exact-permanent payment.
// Use [AntePermanentOrElse] for the same payment without an optional choice.
// The payer must own the nontoken permanent; control is not required.
// Departure, phasing and return cannot supply an available payment subject.
// Each result selects one local branch. Typed operands and bodies declare
// explicit source reads to waiting delayed queries without executing payment.
// Paid-player collections and ante-to-graveyard actions need separate services.
//
// Use [DeclareAnteCardTarget] for one card in the shared ante zone.
// Either player's card can qualify. The target keeps its exact zone visit
// from announcement, before payment. Departure and return end its legality.
// An ownership change keeps the visit and supplies the current owner.
// Copies retain saved visits; legal new target choices save current visits.
// Clones keep separate ante storage and target state. No LKI permits an
// action on an absent card. Ante movement and exchange need separate services.
//
// Use [ExchangeCardOwnership] for two present typed cards in an ante game.
// Both nontoken cards and their current owners are checked before either change.
// The exchange keeps zones, visits and controllers. It sends no zone events.
// Two cards with one owner make no change. Returned visits cannot qualify.
// Existing ownership storage keeps clones and final settlement separate.
// RON accepts bound singular exile, hand reveal, graveyard and ante operands.
// Paid source references use typed card results. Explicit destination moves
// and references from anywhere need separate adapters.
//
// Use [AcquireCardOwnership] for a present typed card in an ante game.
// Ownership uses [Game.ChangeOwner] and does not move the card or change its
// visit. Graveyard ownership updates keep visits of all stored cards.
// Use [ExchangeAnteCardWithLibraryTop] for a separate cross-zone exchange.
// The ante card and library top must belong to the selected player.
// An empty library receives the ante card. An ineligible top prevents both
// moves, but cannot reverse a prior ownership action. Both moves finish before
// zone events. Target departure and return end the saved reference.
// Clones keep separate ownership and movement. The action reads the current
// library top at resolution. It does not draw, reveal, shuffle or make choices.
// Typed player operands declare explicit source reads to waiting delayed queries.
// Two-card ownership exchanges and result-bound cards need separate adapters.
//
// Use [AnteCardsOwnedBy] for typed current-owner selection in shared ante.
// [OtherAnteCards] excludes the physical source after its payment moves it.
// [PutAnteCardsInGraveyards] saves exact visits and uses current owners.
// All owner order choices precede movement and use active-player order.
// All movement precedes zone events. A returned visit cannot replace a saved
// subject. Clones keep separate zones. This movement is not a death or mill.
// RON source exclusion is local to a paid source-ante continuation.
// Ownership acquisition and atomic library exchange need separate operations.
//
// Use [NewCardDefinitionPlan] for a pure builder of printed definitions.
// It checks separate builds for shared abilities, targets, costs and bindings.
// [RebuildCardDefinition] creates new declarations without a registry lookup.
// The retained plan supplies independent printed declarations.
// Saved owners, targets, source identities and resolution choices do not transfer.
// [BaseCard.Copy] retains the plan. [BaseCard.CloneFrom] clears it because
// legacy rule replacement cannot establish the new definition's construction plan.
// RON factories retain their validated plans and copiable metadata.
// Use [Game.CaptureCopiableValues] for a present permanent with known provenance.
// The immutable [CopiableValues] rebuilds independent face-up prototypes.
// Ordinary layers, counters, ownership and saved choices do not transfer.
// Standard face-down status supplies a nameless, colorless 2/2 creature.
// [CopiableValues.HasManaCost] keeps an absent cost separate from a zero cost.
// Returned and phased subjects cannot supply a new capture. Saved values remain.
// Legacy copy layers and untracked characteristic changes return errors.
// The snapshot does not supply casting or battlefield entry permission.
// Live copy layers and other copy exceptions need separate services.
// [BuildEntryCopy] builds fresh battlefield rules for a physical card's next
// entry. It keeps that card's ID, owner, token status and printed origin.
// Departure uses fresh printed rules; LKI and pending abilities keep copy rules.
// RON factories retain this origin with [WithPrintedCardDefinitionPlan].
// The caller must supply entry permission, copy choice and replacement order.
// This builder does not schedule entry-copy replacements or install live layers.
// [NewTokenDefinitionFactory] checks pure token builders and retains their
// copiable definition plans. RON token declarations use this factory.
// Creation builds all independent token cards before the shared entry batch.
// Builder errors stop the batch and clear its result bindings.
// Token snapshots have no mana cost, owner, source identity or saved choices.
// Their prototypes are not tokens; [BuildEntryCopy] keeps physical token status.
// Legacy [DefineToken] callers keep their existing construction behavior.
//
// Use [MayCopyAsEnters] for an optional copy before other entry operations.
// The entry controller chooses a present matching permanent without targeting.
// The shared replacement pipeline applies the copy priority and checks new rules.
// Stable definition positions prevent a rebuilt rule from repeating in one event.
// Copying uses validated provenance, fresh rules and the physical entry builder.
// Acquired entry counters, choices and triggers use the copied definition.
// Departure restores printed rules; phasing does not repeat the copy choice.
// Use [MayCopyAsEntersWithModifications] with [CopyAddCardTypes] to add permanent
// types within the copy process. [CopiableValues.WithModifications] keeps the
// modified types in a validated snapshot with fresh rules for each build.
// Later copies keep these types; ordinary type layers do not enter the snapshot.
// Type additions keep abilities and the entry rules' original identities.
// Color exceptions, added abilities and live copy layers need separate services.
//
// CopiableValues.WithModifications keeps copy exceptions in text builders.
// BuildTextChangedDefinition and BuildTextDefinitionRevision use fresh rules.
// Added types, retained colors and color-definition omissions stay in order.
// Protected printed values are checked before copy exceptions are applied.
// Ordinary word changes do not enter the retained copiable values.
// RON uses its existing AddCardtypes and KeepColor entry-copy lowering.
// Pending permanent spells and whole-object text still need separate services.
//
// Use [MayCopyAsEntersKeepingColors] to retain its copiable colors.
// [CopyRetainColors] keeps a saved color set in an immutable copy snapshot.
// Empty sets are colorless, even with a colored copied mana cost.
// Color-defining effects use [ColorCharacteristicDefinition]. Their abilities
// are omitted from this copy; ordinary color-changing abilities remain.
// Later copies retain the exception. Ordinary color layers still apply.
// Entry replacement identities survive the omission of color definitions.
// Added abilities and live copy installation need separate services.
//
// Use [OracleCardNameCatalog] for legal English names without card factories.
// The retained nontoken snapshot includes separate faces and excludes combined
// names. [CardNameCatalog.Names] returns separate storage for a choice.
// [ChooseCardName] uses the specified player at resolution and saves a typed
// [CardNameBinding]. An absent chooser clears the result. An invalid answer
// uses the first legal name. Clones and resolution frames keep separate results.
// RON supports ChooseACardName(AnyOracleCard) in resolving action contexts.
// [FromCardName] supplies a typed [CardNameRef] for the current frame.
// Definition checks require a producer before each read on every path.
// RON name operands keep lexical choice identity through PlayerActions.
// Conditional choices stay in their branch. [MatchesChosenCardName] checks
// supplied object names with the retained interchangeable English names.
// A missing choice or a nameless object cannot match. [DiscardHandMatchingName]
// selects the current hand before the shared discard batch and its replacements.
// RON DiscardCards(IsNamed(TheChosenName)) uses this service at resolution.
// Reveal results and their consumers need separate compiler adapters.
//
// Use [RevealedCardHasType], [RevealedCardLacksType] and [RevealedCardNamed]
// with [IfRevealedCardMatches] to check facts saved by a reveal operation.
// [RevealRandomHandCardInto] saves an immutable name and card-type snapshot.
// Missing results cannot match, including a negative card-type filter.
// [RevealedCardManaValue] reads the mana value saved by each singular reveal.
// Missing facts supply zero. X in a revealed hand or library card supplies zero.
// This value keeps historical facts after departure, return or later changes.
// [IfLifePaid] and [UnlessLifePaid] read a typed amount once before the choice.
// Nonpositive amounts require acceptance of a zero payment. The shared cost
// transaction checks affordability again and pays the selected player's life.
// Each result stays local. Delayed queries retain explicit source reads in the
// payer, amount and body without reading amounts or offering payments.
// RON optional resolution PayLife(ManaValueOfTheCardRevealedThisWay) uses the
// current lexical reveal with an adjacent If or Unless payment continuation.
// Activation, mandatory and alternative payments require separate adapters.
// These history reads keep reveal facts after departure or return; they do not
// permit an action on an absent card. [FromCard] keeps exact-visit checks.
// The name filter uses the shared chosen-name matcher and its saved binding.
// Frames, delays and clones keep separate result maps with immutable facts.
// Waiting queries retain source reads from both bodies without checking a match.
// Library producers and current mana-value reads need separate
// contracts. This operation supplies resolving predicates, not static conditions.
//
// Use [RevealHandCardInto] for a public reveal of a typed, saved hand card.
// [DrawFor] with [DrawCardInto] saves the original successful draw.
// Replacement draws cannot supply this result. The reveal saves separate facts
// and an exact hand visit without movement, choices or random selection.
// An absent or returned input clears the result. Historical predicates retain
// facts after departure; [FromCard] still requires the saved visit for an action.
// RON DrawACard, RevealTheCardDrawnThisWay and DiscardTheCardDrawnThisWay
// use separate lexical draw and reveal bindings in resolving action lists.
// Branch-local producers cannot supply results to an outer continuation.
//
// Use [ChoosePlayer] for a required player choice at resolution.
// The specified player chooses from a typed set. The saved [PlayerBinding]
// belongs to its resolution frame. An empty set or absent chooser clears it.
// [PlayersWhoCastSorceryThisTurn] reads the existing turn history. Countered
// and unresolved casts qualify; failed payments and spell copies do not.
// Cleanup clears that history. Delayed bodies retain their saved player bindings.
// Explicit source reads in the chooser and set supply waiting references.
// Waiting queries do not select players or make choices.
//
// Use [HighestManaValue] to keep all maximum-mana-value members of a set.
// Selection reads current membership and keeps its order. Face-down permanents
// have mana value zero. Combine it with [ChoosePermanent] for a required choice
// at resolution. The saved binding keeps the chosen incarnation through control
// changes and phasing. It cannot follow a returned card. Empty choices clear it.
// Delayed collection bodies retain explicit source reads through this selector.
// Waiting queries do not select members or read their mana values.
// [ChoosePermanent] also declares explicit source reads in its player and set.
// Waiting queries do not resolve the player or make a choice.
//
// Use [Game.WaitingPreventionPermanentSources] for exact permanent references
// retained by finite recipient, unlimited recipient and next-source prevention.
// Resolved combat prevention supplies its exact source or to-and-by subject.
// Collection prevention supplies each member saved at resolution.
// Typed static prevention supplies its current subject. The live rule must
// remain active; a former attachment cannot supply that subject.
// [PreventDamageFromMatchingPermanents] supplies its current exact subject.
// Its permanent filter uses current or retained source information.
// [ChooseDamageSourceFromWaitingPreventionReferences] uses this category only.
// [Game.WaitingRedirectionPermanentSources] reads live destinations of typed
// static damage redirection. It uses the same subject check as damage execution.
// Legacy cycle redirection also supplies its exact live destination.
// Each registration keeps the destination incarnation. A returned card needs
// a new registration. Cycle owners still control registration removal.
// [ChooseDamageSourceFromWaitingRedirectionReferences] uses this category only.
// Use [Game.WaitingPreventionSpellSources] for exact spell references in
// active source shields. Live or retained spell data keeps separate casts apart.
// [ChooseDamageSourceFromWaitingPreventionSpellReferences] uses this category only.
// Consumption and cleanup end its availability. A continued permanent keeps
// its separate incarnation reference.
// [Game.WaitingDrawStepLifePermanentSources] reads exact sources of active
// draw-step life replacements. Departure, phasing and ability removal exclude
// the source. A pending life gain does not keep a permanent reference.
// [ChooseDamageSourceFromWaitingDrawStepLifeReferences] uses this category only.
// [Game.WaitingRegenerationPermanentSources] reads exact subjects of active
// regeneration shields. Departure and phasing keep their saved references.
// [ChooseDamageSourceFromWaitingRegenerationReferences] uses this category only.
// [PreventDamageToSourceByRemovingCounters] supplies its exact live subject to
// [Game.WaitingPreventionPermanentSources], even with no counters. A saved rule
// cannot follow a returned source or an ability that was removed.
// [Game.AddSourcePreventionShield] and [Game.AddHalfDamageFromSourcePreventionShield]
// retain a live permanent source's exact incarnation. Both supply that source
// to [Game.WaitingPreventionPermanentSources] until consumption or cleanup.
// Departure and phasing keep the reference. A returned source cannot inherit
// the shield. Registrations without a live permanent keep their legacy matching
// behavior and cannot supply an exact waiting reference.
// [Game.AddReverseDamageShield] retains the same exact permanent source and
// supplies it to the waiting prevention query until consumption or cleanup.
// Positive damage from that incarnation consumes the shield before life gain.
// [RedirectDamageFromPermanentThisTurn] saves a permanent source, player and
// destination until cleanup. Its exact source and destination supply waiting
// redirection references, even after departure or phasing. Damage requires the
// original destination to be present as a creature or planeswalker.
// [RedirectNextDamageFromSelectedSource] saves an exact source, subject and player.
// It redirects one damage event to the saved player until cleanup. All matching
// simultaneous packets use it together. A selected permanent spell continues
// to the exact permanent it becomes before entry effects run.
// Its source filter checks current or retained information before damage.
// [Game.WaitingRedirectionPermanentSources] includes its permanent references.
// [Game.WaitingRedirectionSpellSources] includes its exact selected spell.
// [ChooseDamageSourceFromWaitingRedirectionReferences] reads both categories.
// [Game.WaitingPreventionExileSources] and
// [Game.WaitingRedirectionExileSources] read exact visits in active damage rules.
// Departure keeps public facts; face-down cards keep no characteristics.
// A returned card cannot replace the saved visit. Consumption and cleanup end
// availability. Cycle registrations retain their existing cycle lifetime.
// [ChooseDamageSourceFromWaitingPrecastExileReferences] uses these two categories.
// [Game.WaitingPreventionHandSourcesFor] and
// [Game.WaitingRedirectionHandSourcesFor] read exact hand visits in active rules.
// The hand player can inspect private facts. Other viewers receive public facts.
// Departure keeps separate private facts and public visibility for that visit.
// A waiting reference cannot extend a cost reveal or follow a returned card.
// [ChooseDamageSourceFromWaitingPrecastHandReferences] uses the choosing player.
// Consumption and cleanup end availability. Queries and clones keep separate data.
// [Game.WaitingPreventionCommandSources] and
// [Game.WaitingRedirectionCommandSources] keep exact command visits in active rules.
// [ChooseDamageSourceFromWaitingCommandReferences] uses these two categories.
// Departure keeps the visit; a return cannot replace it. Hidden information
// stays hidden. Consumption and cleanup end availability.
// [Game.WaitingModifierPermanentSources] reads explicit sources of active
// registered mill and life-gain modifiers and their static factories.
// Queries do not call callbacks or factories. Phasing suspends availability.
// A saved registration cannot follow a returned source. Static factories use
// current abilities; ability removal stops their references.
// [ChooseDamageSourceFromWaitingModifierReferences] uses this category only.
// [Game.WaitingEmptyLibraryPermanentSources] reads exact sources of active
// empty-library draw replacements without calling their callbacks.
// [Game.AddEmptyLibraryDrawReplacement] saves the live source incarnation.
// Phasing suspends the rule. Departure ends it; a return needs a new registration.
// [ChooseDamageSourceFromWaitingEmptyLibraryReferences] uses this category only.
// [Game.WaitingPersistentReplacementPermanentSources] reads exact sources of
// registered life-gain, sanctuary draw and counter-doubling replacements.
// Their registrations save the live incarnation. Phasing suspends them;
// departure ends them. A return needs a new registration.
// [ChooseDamageSourceFromWaitingPersistentReplacementReferences] uses this category only.
// [Fight] declares explicit source reads in both permanent operands to
// [Game.WaitingDelayedPermanentSources]. Queries do not resolve operands or
// deal damage. Saved incarnations survive departure and phasing until detection.
// Returned sources cannot replace them. Implicit hosts and future delayed bodies
// do not infer source reads. Bound operands use the saved frame.
//
// [ScryAtResolution] and [SurveilAtResolution] declare explicit source reads
// in their typed amounts to [Game.WaitingDelayedPermanentSources]. Queries
// do not read amounts, inspect cards or make choices. Zero amounts keep reads.
// Departure and phasing keep the saved incarnation until trigger detection.
// Returned objects cannot replace it. Implicit attachments, opaque values
// and future delayed bodies do not infer reads. Each body uses its saved frame
// and controller with the existing scry or surveil service.
//
// [ReturnCardToBattlefield] declares explicit source reads in its controller
// to [Game.WaitingDelayedPermanentSources]. Queries do not read the controller
// or move a card. Departure and phasing keep the saved incarnation until
// detection. A returned source cannot replace it. Implicit attachments,
// opaque players and future delayed bodies do not infer source reads.
// Entry options do not change the declaration. Saved graveyard visits and
// controller information use their existing frame and information services.
//
// [Regenerate] declares an explicit source subject to
// [Game.WaitingDelayedPermanentSources]. Queries do not resolve the subject
// or create a shield. Departure and phasing keep the saved incarnation until
// detection. Returned objects cannot inherit it. Implicit attachments and
// future delayed bodies do not infer reads; bound subjects use saved frames.
//
// [AddManaAmount], [AddManaOfAnyOneColor] and [AddRestrictedManaOfAnyOneColor]
// declare explicit source reads in their amount to waiting delayed queries.
// [AddManaToPlayer] and [AddManaOfProducedTypeToPlayer] also declare player reads.
// Queries do not read amounts, choose colors or add mana. Zero amounts keep
// explicit reads. Departure and phasing keep the saved incarnation until detection.
// Returned sources cannot inherit a saved reference. Delayed frames save produced
// mana types for each instance. Resolution and clones keep separate type slices.
// Implicit attachments, opaque operands and future bodies do not infer reads.
//
// [Game.WaitingUntapSkipPermanentSources] reads exact sources of active
// typed untap-step replacements. Queries use the replacement's live ability
// and incarnation checks. Phasing, departure and ability removal stop them.
// [ChooseDamageSourceFromWaitingUntapSkipReferences] uses this category only.
// [Game.WaitingDiscardPermanentSources] reads exact sources of active typed
// discard-to-library replacements. The saved incarnation and live ability
// control execution and enumeration. Phasing and ability removal stop the rule.
// Each discard reads the current controller. A return needs a new registration.
// [ChooseDamageSourceFromWaitingDiscardReferences] uses this category only.
// [Game.WaitingExileSources] reads exact spell and permanent subjects of active
// exile-instead rules. A permanent spell keeps its exact continuation before
// entry effects run. Phasing and departure keep the saved references until
// cleanup. Returned permanents and later casts cannot inherit a saved rule.
// [ChooseDamageSourceFromWaitingExileReferences] uses this category only.
// Legacy registrations before casting supply no exact waiting reference.
// Other waiting replacement references need separate contracts.
//
// [Game.SetCreatureDamageRedirect] saves a live subject until use or cleanup.
// [Game.SetAttackerDamageRedirect] saves live source and destination identities
// until cleanup. Both supply exact waiting redirection references after departure
// or phasing. Damage still requires the original live destination. A returned
// source cannot inherit the attacker redirect. An absent installation adds no rule.
//
// Use [PreventNextDamageFromSelectedSource] for a chosen-source shield.
// A shield on a permanent spell continues to the exact permanent it becomes.
// Entry effects run after the link. Separate copies and returned objects cannot
// inherit it. [PreventNextDamageFromExactSource] keeps exact-object behavior.
// Both services require a typed source; source enumeration is separate.
//
// Use [Game.WaitingCounterPlacementPermanentSources] for explicit source
// exclusions in active entry-counter replacements. General filters do not
// supply references. A saved registration cannot follow a returned source.
// [ChooseDamageSourceFromWaitingCounterPlacementReferences] uses this category only.
//
// Use [Game.AddCommandObject] to install an object with a fresh command visit.
// The caller must remove the object from its former zone first.
// [Game.CommandDamageSources] supplies face-up objects to source choices.
// [ChooseDamageSourceFromCommandObjects] uses this category only.
// [DamageSource.CommandVisit] keeps a return separate from its former visit.
// Removal retains departure information. Hidden objects supply no public
// characteristics. Queries and game clones keep separate object snapshots.
// This service does not grant casting permission or install command abilities.
//
// [Game.CaptureHandDamageSource] reads one live hand visit.
// [DamageSource.HandVisit] keeps returned cards separate from saved sources.
// Hidden hands supply identity and owner only. Public hands supply current facts.
// [Game.StackHandSources] reads exact hand targets and payment card bindings
// and nested event card captures on stack objects. Departure keeps the facts
// available at that visit's departure; final stack departure ends this category.
// [ChooseDamageSourceFromStackHandReferences] uses this category only.
// A cost reveal keeps its exact card public until that stack object leaves.
// Copies keep the reference without extending the original reveal.
// A selected hand source cannot follow a later spell or permanent.
// Hand targets save their exact visit at announcement. Copies keep that visit.
// A new target choice saves the current visit, even with the same card ID.
// Returned targets are illegal for the original spell or ability.
// Waiting card frames use their separate reference contracts.
// [Game.CaptureHandDamageSourceFor] and [Game.StackHandSourcesFor] add the
// hand player's private inspection information. Other viewers keep public facts.
// [ChooseDamageSourceFromInspectableStackHandReferences] uses the choosing player.
// Inspection does not reveal a card. Departure saves separate public visibility;
// returned visits cannot inherit it. Damage rechecks use actual current or
// departure facts, independent of the information shown to a player.
//
// [Game.CaptureGraveyardDamageSource] reads one live graveyard visit.
// [DamageSource.GraveyardVisit] keeps returned cards separate from saved sources.
// [Game.StackGraveyardSources] reads exact targets, payment card bindings
// and nested event card captures on stack objects. Departure keeps public facts;
// the final referring stack departure ends this category's availability.
// Resolving frames retain these references until body completion.
// Queries and game clones keep separate mutable card facts.
// [ChooseDamageSourceFromStackGraveyardReferences] uses this category only.
// A selected graveyard source cannot follow a later spell or permanent.
// Waiting and delayed card frames need separate reference contracts.
//
// [Game.CaptureExiledDamageSource] reads one live exile visit.
// [DamageSource.ExileVisit] keeps each visit separate from later objects.
// Face-down cards supply no characteristics, even with inspection permission.
// [Game.StackBoundExileSources] reads exact exile targets, card payment bindings
// and event card captures on stack objects. Departure keeps exact information.
// [ChooseDamageSourceFromStackExileBindings] uses only this category.
// Exile targets save their visit before payment. Copies keep the visit until
// a legal new target choice saves the current visit. Returned targets are illegal.
// The resolving stack frame keeps references until resolution ends.
// A selected exile source cannot follow a later cast or battlefield entry.
// [Game.GrantCastFromExile] saves the current exile visit. Departure ends
// its permission. A failed payment keeps the visit and its permission.
// Re-exile creates a new visit. Clones keep separate exile card information.
//
// [Game.WaitingDelayedGraveyardSources] reads exact visits in waiting card frames.
// Current or departure facts keep returned visits separate. Singular and plural
// bindings share the existing delayed capture, stack transfer and clone services.
// [ChooseDamageSourceFromWaitingDelayedGraveyardBindings] uses this category only.
//
// [Game.StackAbilityPermanentSources], [Game.StackTargetPermanentSources],
// [Game.StackBoundPermanentSources], [Game.StackTargetSpellSources],
// [Game.StackBoundSpellSources] and [Game.StackBoundDamageSources] include
// resolving frames until body completion. Nested resolution and clones keep
// each frame. Spell departure updates retained target information. Saved
// incarnations and spell identities cannot follow returned objects.
// These categories do not enumerate references hidden in effect callbacks.
//
// [Mill] declares source reads in its typed player and amount to
// [Game.WaitingDelayedPermanentSources]. Queries do not read the amount,
// call mill modifiers or move cards. Zero amounts keep the reference.
// Departure and phasing keep the saved incarnation; a return cannot replace it.
// Implicit attachments and future delayed bodies keep separate boundaries.
//
// [Game.WaitingDelayedPermanentSources] includes the saved source when
// [DealDamageFrom] or a packet in [DealDamagePackets] uses [This] as recipient.
// Queries do not read amounts or resolve recipients. A zero amount keeps the
// reference. Departure and phasing keep its saved incarnation; a return cannot
// replace it. Bound and attached recipients do not declare a source reference.
// Saved sequences and modes keep the declaration until the delay triggers.
//
// [RedirectNextDamageFromSelectedSource] declares explicit source reads in its
// source, subject and player to [Game.WaitingDelayedPermanentSources].
// [RedirectDamageFromPermanentThisTurn] declares reads in its source,
// destination and player. Queries do not resolve operands or install rules.
// Implicit attachments and future delayed bodies keep separate boundaries.
// Saved incarnations survive departure and phasing until trigger detection.
//
// [ChooseDamageSourceFrom] and [ChooseDamageSourceMatchingFrom] declare
// explicit source reads in their typed source list and player to
// [Game.WaitingDelayedPermanentSources]. Category choices also declare their
// typed player reads. Queries do not resolve operands, check filters or choose
// a source. Empty runtime lists and nonmatching filters keep explicit reads.
// Saved incarnations survive departure and phasing until trigger detection.
// Implicit attachments, opaque references and future delays do not infer reads.
//
// [ChooseAtResolution] declares explicit source reads in its player, all
// option conditions and all option bodies to the same waiting query.
// Queries do not evaluate conditions or choose an option. Unavailable and
// unselected options keep their reads until the delayed ability triggers.
// Future delays in option bodies keep their separate creation lifetime.
//
// [CreateTokenCopies] and [CreateTokenCopiesInto] declare source reads in their
// player and count to [Game.WaitingDelayedPermanentSources]. [CreateOneToken]
// declares reads in its player. Zero counts keep these reads. Queries do not
// resolve players, read counts or create tokens. Token ability bodies use the
// created token's scope and do not declare reads of the producer's source.
// Saved incarnations survive departure and phasing until the delay triggers.
//
// [SnapshotNumber] declares explicit source reads in its typed value to
// [Game.WaitingDelayedPermanentSources]. Zero values keep these declarations.
// Queries do not evaluate the value or write the binding. Saved numbers do not
// infer an object reference. Departure and phasing keep the saved incarnation;
// a returned source cannot replace it. Future delays keep a separate lifetime.
//
// [TapEach], [UntapEach], [ReturnEachToHand] and [DestroyEach] declare
// explicit source reads in their typed collections to the same waiting query.
// [DestroyEachWithoutRegeneration] uses the same declaration.
// [DealDamageEach] declares these reads in [PermanentDamageRecipients].
// Nested set operands use the same declaration rules as [CountPermanents].
// Queries do not select members or run filters. Empty collections keep reads.
// Resolution selects current members through the existing batch services.
// [AddCountersEach] declares source reads in its subjects and amount.
// [CountCountersAmong] declares source reads in its typed collection.
// Queries do not select members or read counters. Zero amounts keep the reads.
// Totals select current, distinct, present incarnations at resolution.
// RON singleton counter information retains [CountersOnWithLKI].
// [Exile] declares a direct [This] subject to the same waiting query.
// [ExileEach] declares source reads in its typed collection. Queries do not
// select members or move objects. Empty collections keep explicit reads.
// Resolution uses current, present members and the existing exile batch.
// [SkipNextUntap], [SkipNextUntaps] and [SkipNextUntapEach] declare explicit
// source reads in their subjects to the same waiting query. Zero counts and
// empty collections keep the saved source. Queries do not install restrictions
// or select members. [SkipNextUntapOf] also declares a source read in its player.
// Resolution still requires present subjects and reads current collection members.
// [Until], [UntilNextTurn], [UntilNextUpkeep], [UntilEndOfNextUpkeep], [UntilSourceLeaves]
// and [UntilSourceUntaps] declare explicit source reads in their layer subjects,
// sets, values, control recipients and upkeep players to the same waiting query.
// Source-bound durations declare their source even with another layer subject.
// Queries do not read values, select members or install layers. Empty sets and
// zero values keep explicit reads. Granted ability bodies keep their own scope.
// [DrawEach], [DiscardEach], [GainLifeEach], [LoseLifeEach] and library
// recycling declare explicit source reads in their player sets and values.
// [OpponentsOf], [PlayersDealtDamageBy] and singular recycling player sets
// retain typed operand reads. Damage recipient sets, per-player damage,
// random player choices and player-controlled permanent sets use this rule.
// Queries do not select players, read history, draw cards or consume randomness.
// Empty sets and zero amounts keep the saved source until the delay triggers.
// Other collection actions need separate contracts.
// Typed conditional bodies declare explicit source reads in every branch to
// [Game.WaitingDelayedPermanentSources]. [IfState] and delayed intervening
// conditions retain typed source operands without checking the condition.
// [IfPositive] retains its value and body. [FlipCoin] retains its player and
// both outcomes. [IfActivationsAtLeast] retains its source incarnation history.
// [SnapshotPlayer] retains explicit source reads in the binding producer.
// Queries do not read values, check conditions, run branches or consume randomness.
// Opaque callbacks, implicit attachments and future delays do not infer reads.
// [PreventCombatDamageToAndBy], [PreventCombatDamageBy] and
// [PreventCombatDamageByThisTurn] declare direct [This] subjects to the same
// waiting query. [PreventCombatDamageByEach] and
// [PreventCombatDamageByEachThisTurn] declare source reads in typed sets.
// Empty sets keep those reads. Queries do not select members or install rules.
// Global combat prevention does not declare a source subject.
//
// [PayCountersOrElse] and [PayChosenCountersOrElse] declare a direct [This]
// payment subject and source reads in both result branches to
// [Game.WaitingDelayedPermanentSources]. Queries do not resolve players or
// make payment choices. Zero and impossible payments keep the reference.
// A saved incarnation survives departure and phasing until the delay triggers.
// Counter removal still requires the original subject to be present.
// [IfSacrificePaid] and [UnlessSacrificePaid] declare a direct [This] payment
// subject and source reads in their continuation to the same waiting query.
// Queries do not resolve the subject or payer, or offer a sacrifice payment.
// Departure and phasing keep the saved reference; a return cannot replace it.
// Sacrifice still requires the original present subject under the payer's control.
// [IfManaPaid] and [UnlessManaPaid] declare source reads in their amount and body.
// Typed permanent power, toughness, mana value and counter reads qualify.
// Arithmetic keeps declarations from both operands, even with a zero multiplier.
// Queries do not read values or offer payment. Typed payer reads also qualify.
// The saved incarnation survives departure and phasing until the delay triggers.
//
// [PreventNextDamage] declares explicit source reads in its recipient and amount
// to [Game.WaitingDelayedPermanentSources]. [PreventDamageThisTurn] declares
// reads in its recipient. [PreventNextDamageFromPermanent] declares reads in
// its source and player. [PreventNextDamageFromExactSource] and
// [PreventNextDamageFromSelectedSource] declare reads in their source and player.
// Queries do not read amounts or install shields. Zero amounts keep the saved
// incarnation. Departure and phasing keep the reference until detection.
// A returned card cannot replace it. Future delays keep a separate lifetime.
//
// [WhileWithSharedSubjects] reads its condition and exact subjects at its first layer.
// Later layers keep those objects for the same pass, even after ability removal.
// Each new pass reads current state. Values are read in their applicable layer.
// Type dependencies compare selections in separate game probes.
//
// Use [GrantBandsWithOther] for one subject and [GrantEachBandsWithOther]
// for a set of subjects. Both grant a combat ability with a quality filter.
// Current subjects and qualities control formation and damage assignment.
// [Game.FormAttackingBand] checks distinct attackers with one defender before
// blocker declaration. Ability loss keeps the band; combat removal removes
// its member. Losing banding also removes all bands-with-other grants.
// A matching holder and another matching creature transfer the damage choice.
// The choice can assign damage to any eligible combat opponent.
// [RemoveBandsWithOther] removes all quality grants in layer 6.
// Ordinary banding stays available. Later grants can restore a quality.
// With [Until], the saved incarnation keeps removal through a phase cycle.
// A returned permanent cannot inherit it. Removal keeps a formed band.
// Combat damage choices use the abilities that remain at assignment.
// A blocker across several groups divides one amount among its opponents.
// The shared damage batch keeps each source and its own keywords separate.
//
// Use [ExtraTurnFor] for one selected player after the current turn.
// Resolution adds the turn; announcement and waiting queries do not add it.
// An absent player gets no turn. Typed players retain bindings and source reads.
// [ExtraTurn] keeps its controller behavior through the same action.
// [Game.AdvanceToNextTurn] selects the most recent extra turn first (CR 500.7).
// It saves the next normal player through nested and skipped extra turns.
// Native runners, application loops and gametest use this shared service.
// Clones keep separate pending turns and the saved normal player.
// The caller must finish the current turn before advancing. The service does
// not execute steps, cleanup or turn replacements. Multiple-player grants and
// turn-skip replacement declarations need separate adapters.
//
// Use [CantBeAttackedExceptByUntilNextTurn] for saved player protection.
// Both players are read at resolution. Each attacker must pass the live filter.
// Separate rules all apply. Later entries and ability changes affect legality.
// Source changes do not end protection. Clones keep separate lifetimes.
// [BeginActiveTurn] ends it before the saved player's next actual turn actions.
// RON lowers CantBeAttackedExceptBy in resolving CreatePlayerEffectUntil
// lists with UntilPlayersNextTurn and typed player references.
//
// # Subsystem Architecture and Ownership
//
// [Game] serves as the aggregate root and coordinator of the engine. Rather than
// implementing every domain rule directly, [Game] composes cohesive, stateful
// subsystems that each own their state invariants and cloning behavior:
//
//   - [ZoneSystem]: Owns battlefield storage with copy-on-write semantics for AI
//     search, exile storage with visibility metadata, entering-permanent tracking,
//     and Last-Known Information (LKI) snapshots.
//   - [TurnSystem]: Owns turn counters, current phase and step, active player
//     index, extra turns, [TurnSchedule] lifecycle, and one-shot untap restrictions.
//   - [ResolutionState]: Owns transient resolution scratch state including X values,
//     chosen modes, event source/amount context, cast-time snapshots ([CastContext]),
//     targets, damage/counter distributions, and cost payment reveals/sacrifices.
//   - [TriggerSystem]: Owns pending triggers waiting for stack placement, active
//     [DelayedTrigger] instances, and armed state triggers (CR 603.8).
//   - [ManaSystem]: Owns mana source discovery, mana ability solver planning,
//     and payment scratch buffers.
//   - [DamageSystem]: Owns damage execution, combat damage step aggregation,
//     damage history tracking, and damage reflection.
//   - [TrackerSystem]: Owns turn-scoped observations ([TurnTrackers]) and duel-scoped
//     observations ([DuelTrackers]).
//   - [RandomSource]: Owns deterministic integer and coin flip queues for tests
//     and AI simulation.
//
// In addition to these subsystems, [Game] coordinates the [Stack], [Combat],
// [EffectManager] (continuous and replacement effects), and [Player] instances.
//
// Read-only queries are organized into focused domain interfaces:
// [PlayerReader], [BattlefieldReader], [ResolutionReader], [TurnReader],
// [CombatReader], [TrackerReader], and [StackReader], which compose into [GameReader].
//
// Use [Game.CardPowerToughness] for current card P/T with count CDAs.
// Cards without a controller use their owner. Spells use their current
// controller. Battlefield reads use the existing layer results.
// [PTEqualsCount], [PTEqualsControlledCount] and [ToughnessEqualsControlledCount]
// keep their count declarations outside the battlefield. Copies keep those
// declarations; each battlefield instance has a separate source binding.
// Saved object information must use the existing LKI services.
// Use [CantBeTargetedBySpellsIf] for a live conditional spell target restriction.
// The condition reads the current source controller at each target query.
// Ability targets use their existing rules. Source departure, phasing and
// ability removal stop the restriction through the shared effect lifecycle.
//
// Spell definitions can use OnlyDuring to restrict casting to one or more turn steps.
// Game checks the restriction before choices and payment. Copies of stack spells
// and spell resolution do not repeat the casting check.
// Use [ReduceSpellCostStatic] for a generic cost reduction on matching spells
// cast by the current controller. A phased or departed source cannot supply
// the reduction. The saved source incarnation prevents a returned card from
// restoring an old entry before the next layer pass.
// Use [IncreaseActivationCostStatic] for fixed mana increases on abilities
// of matching live permanents. Use [ReduceActivationCostStatic] with
// [AttachedPermanent] for a generic reduction with a minimum mana payment.
// The payment service combines mana costs and announced X before modifiers.
// Increases precede reductions. Mana planning uses the same total.
// Phasing, source departure and ability removal stop the modifier.
// Aura reattachment changes the subject; a returned host cannot restore it.
//
// Use [IncreaseSpellCostStatic] for fixed generic and colored spell taxes.
// Increases apply before reductions in the shared total spell cost.
// With controllerOnly, the source's current controller must cast the spell.
// Source departure, phasing and ability removal stop the tax.
// Use [DamageTakenThisTurn] or [ArtifactDamageTakenThisTurn] for a typed
// player damage total at resolution. The total uses actual damage this turn
// and the source types at the time of damage. Later type changes do not change it.
// Use [PayOrElse] for one optional payment with two resolution branches.
// The payment result stays local to the operation.
// Use [PayActionOrElse] with [GainLifePayment], [DamagePayment], or
// [DestroyPayment] for one optional action paid during resolution.
// The accepted payment selects the branch even if prevention or replacement
// changes its outcome. An absent subject or an indestructible destruction
// subject cannot pay. Each payment keeps its own references and amount.
// Resolution counter payments use PayCountersOrElse for a fixed count and
// PayChosenCountersOrElse for an optional chosen count. Each payment requires
// the original live permanent. The chosen count is available only in its paid
// branch. Counter choices and removal use the shared payment transaction.
//
// Use [UnlessManaPaid] or [IfManaPaid] for a mana payment during resolution.
// Supply a mana cost with X and a [ResolutionValue]. The amount is read once
// before the payment choice. Activation cost reductions do not change it.
// [ManaValueOfSpellWithLKI] retains the value when a spell leaves the stack.
// A later cast of the same card cannot replace that information.
// Use [WithCastingCondition] for a pure restriction before casting choices
// and payment. Conditions read the caster. [IsDuringSteps] reads the current
// step through [TurnReader]. [AfterFirstUpkeep] permits later upkeeps after
// the first upkeep ends. A restriction does not grant casting permission.
// Use [BeforeCombatStep] for attackers, blockers, or combat damage boundaries.
// A first-combat window stays closed during later combats. Set during to true
// for a window in each combat phase.
// Skipped combat points close their windows at the applicable phase boundary.
//
// Use [BlockAnyNumber] and [BlockEachAttacker] with [Until] for captured
// blocking permission and requirements. These rules survive ability removal.
// The shared blocker planner still checks evasion and minimum blocker counts.
//
// Use [CantAttackMatchingThisTurn] for a live attack restriction until cleanup.
// Use [MustAttackControlledThisTurn] for a saved player's live attack requirement.
// These rules remain separate from creature abilities. Later entries and control
// changes can change membership. Source departure does not end the rules.
// Attack requirements do not force a player to pay an attack cost.
//
// Use [AssignsNoCombatDamage] with [Until] to stop combat assignment for
// the resolved permanent. The rule survives ability removal and source departure.
// A phase cycle keeps its identity; a returned card cannot inherit it.
// Noncombat damage and incoming damage still use the damage services.
//
// Use [AllCreaturesMustBlock] with [While] for a live filtered blocking rule.
// [AttachedPermanent] follows the current host. Separate requirements share
// block capacity and cannot bypass evasion or minimum blocker counts.
// [Game.CompleteBlockAssignments] reads a proposal without changing combat.
// Both declaration paths commit a legal maximum before they send events.
//
// Use [TargetBlocksSource], [TargetBlockedBySource], or [TargetInCombatWithSource]
// for a target in a current blocking pair with the original source. Both
// incarnations must remain in combat. LKI cannot restore the pair.
// Use [BlockedByThisTurn] for live members of an exact blocker's turn history.
// Saved blocker references keep their identity after departure and phasing.
// [ControllerWhenLastBlockedThisTurn] reads the attacker controller at the last
// matching block. Returned objects cannot use an earlier object's history.
// Both services use turn history that remains until final cleanup.
//
// Use [BlockersOf] or [BlockedAttackersOf] for a collection at resolution.
// [PreventCombatDamageByEach] captures that collection for this combat.
// [PreventCombatDamageByEachThisTurn] captures it until cleanup.
// Later combat changes cannot add sources to the captured collection.
//
// Use [TargetToughnessLessThanSourcePower] on typed permanent targets for
// a strict current toughness comparison. Source departure uses retained power;
// a returned source cannot replace that information. Combine [HasPowerEqual]
// and [HasToughnessEqual] for exact current P/T targets.
//
// Use [SetBasePowerToughness] with [Until] to set captured base values.
// Use [SetEachBasePowerToughness] and [AddEachCardType] for a set animation.
// Until saves the members before any type change. Later entries cannot join it.
// Combine [AddCardType] and [AddSubtypeFamily] with these base values for
// temporary animation. The added types permit the subtype checks in one effect.
// [ValidateSubtypeFamily] permits declaration checks before factory construction.
// [SetBasePower] changes only base power. Bonuses and counters still apply. [TargetOtherThanSource] excludes the
// source from a typed permanent target.
//
// Use [WithActivationStateCondition] with [PermanentMatchesFilter] for a
// live source or attachment restriction before activation payment. Mana planning
// uses the same pure check. Copies and resolution do not repeat it.
// Use [PlayerIsPermanentOwner] or [PlayerIsPermanentController] with
// [WithActivationStateCondition] for a live player restriction.
// Combine this condition with [WithAnyPlayerMay] when the permitted player
// can differ from the source controller. Use [This] or [AttachedPermanent].
//
// Use [EachBeginCombat] or [YourBeginCombat] for combat-start triggers.
// Combine [TurnStepBegins] with a player binding to save the event player.
// Extra combats generate separate events; skipped combats generate none.
//
// Use [BindActivationHistory] and [WithActivationHistory] to keep activation
// counts separate for each ability and source incarnation. [IfActivationsAtLeast]
// reads the count at resolution. Countered activations count; copies do not add
// activations. Clones keep separate history. Final cleanup clears the count.
//
// Use [Activate] with [Mana] costs and [Until] with [ModifyPT] of [This]
// for a temporary source bonus. Each resolution adds a separate bonus to
// the original permanent. A returned card does not receive that bonus.
// Jumpstart and Legends source bonuses use this form with explicit mana costs.
// Osai Vultures uses [RemoveCountersCost] to pay two carrion counters
// before its temporary source bonus resolves.
// Use [TapNMatchingIncludingSourceCost] for a fixed matching tap payment.
// The source can pay when it matches. Each choice keeps its incarnation.
// A written tap cost permits summoning sickness; a tap symbol does not.
// The shared payment transaction checks all choices before it spends resources.
// Use [SacrificeNMatchingIncludingSourceCost] for a fixed count payment that
// permits the source. All choices precede one simultaneous sacrifice batch.
// Stale count choices fail before mana and other resources are spent.
// RON Goblin Warrens and Mold Demon use this cost in activation and resolution.
// Fallen Angel uses [SacrificeMatchingIncludingSourceCost] with [IsCreature].
// The source can pay this cost. The bonus uses its original incarnation.
// Use [OpponentControls] with [While] for a static condition that reads
// the current opponent's permanents. Combine [Not] and [IsToken] to exclude
// tokens. Control changes and phasing change this condition immediately.
// RON IsNonToken uses the same filter for targets, collections, payments,
// static selections and resolving property conditions. Copying characteristics
// does not change physical token status. Targets keep their legality checks;
// information reads can use LKI without permitting actions on absent objects.
// Use [IfPermanentMatches] for a resolving property condition. It uses the
// current object or information saved for its exact incarnation. An information
// read does not make an absent object available for an action.
// Use [UnlessPermanentCameUnderControlThisTurn] for a resolving player control history
// exception. Each incarnation retains acquisition turns for all players. Later
// control changes cannot erase an earlier acquisition in the same turn.
// Use [IfPermanentMatchesElse] to select one of two resolution bodies.
// Use [UnlessPermanentMatches] for a body on a known nonmatching object.
// Both read the condition once. Missing information runs neither body.
// [AttackedThisTurn] and [DidntAttackThisTurn] keep each incarnation separate.
// Entering combat without an attack declaration does not satisfy attack history.
// [TargetPlayerAttackedThisTurn] checks the player's history at target selection
// and resolution. Creature departure and control changes do not erase that history.
//
// Use [PermanentMatchesFilter] with [This] or [AttachedPermanent] for a
// live static condition. A returned permanent cannot satisfy an old reference.
// Use [HasCounter] or [CounterCountEquals] for a live counter condition.
// These filters read one counter kind. Exact counts do not use a narrow cast.
// Use [CounterCountAtLeast] for a minimum count of one counter kind.
// Resolving RON counter conditions use current or retained information for
// the original permanent. An absent permanent cannot receive an action.
// Use a typed target with [WithTargets] for a targeted bonus.
// Dwarven Lieutenant combines [IsCreature] with [HasSubType] for its Dwarf target.
// Either player's Dwarf creature can be the target, including the source.
// Angelic Page combines [IsCreature] with [IsAttacking] or [IsBlocking].
// Check combat status again at resolution. After resolution, leaving combat
// does not end the bonus.
// For an entry bonus, put [WithTargets] on [WhenEvent] with [Enters].
// Choose the target when the trigger goes on the stack. The creature spell
// has no target, and the entered creature can target itself.
// [ValueAtResolution] captures X when the bonus starts. A blocking restriction
// belongs to the target filter; it does not end a bonus after resolution.
//
// Icatian Scout and Svyelunite Priest use typed creature targets with
// [GrantKeywordModification] and [Until] for temporary keyword grants.
// Combine [OnlyDuring] with [OnlyOnYourTurn] for a controller upkeep restriction.
// Jump and Pixie Queen use the same typed flying grant in [Spell] and [Activate].
// Either player's creature can be the target. The grant expires at cleanup.
//
// Use [UntilSourceUntaps] with continuous modifications for a bonus that lasts
// while the source remains tapped. The duration starts at resolution. Untapping,
// source phasing, and source departure end it permanently. A phased recipient
// keeps its captured identity. Target departure does not remove a source's
// separate choice to remain tapped during its untap step.
//
// Use [DeclareSpellTargetTargetingControlledPermanent] for a spell that targets
// a permanent you control. It checks current control and the saved incarnation
// at announcement and resolution. A target that gained shroud can still qualify.
// Absent, returned and phased-out targets do not qualify. Copies use their own
// stack identity and controller. Combine card filters to restrict the spell type.
//
// Use [PreventDamageFromBlockedCreatures] with [Static] for damage from
// creatures that the current subject blocks. Combat and noncombat damage
// use the current pair. Retained information cannot restore a former pair.
//
// Delayed bodies keep spell-event bindings for the original stack identity.
// Use [Game.WaitingDelayedSpellSources] and
// [ChooseDamageSourceFromWaitingDelayedSpellBindings] for saved waiting spells.
// Queries use current or exact departure information. A later cast cannot
// replace a saved spell. These services supply one source category.
//
// Delayed bodies keep singular and plural card bindings for their saved visits.
// [Game.WaitingDelayedHandSourcesFor] uses the viewer's hand inspection rights.
// [Game.WaitingDelayedExileSources] keeps face-down characteristics hidden.
// [ChooseDamageSourceFromWaitingDelayedCardBindings] uses these two categories.
// Trigger placement transfers the references to the existing stack queries.
// Each execution restores the outer card frame, including after an error.
// A saved reference cannot extend a reveal or follow a returned card.
//
// Delayed bodies also keep typed damage-source bindings and their saved facts.
// Use [Game.WaitingDelayedDamageSources] and
// [ChooseDamageSourceFromWaitingDelayedDamageBindings] for these waiting frames.
// Queries read current or exact departure information. Body execution restores
// the saved event information without changing the outer resolution frame.
// Placed delayed bodies use [Game.StackBoundDamageSources].
//
// [Game.DamageSourcesFor] reads the CR 609.7a categories for a choosing player.
// [ChooseDamageSource] saves one filtered, non-targeted choice at resolution.
// Live objects, stack references, waiting replacement and delayed references,
// and face-up command objects share exact identity checks. Duplicate references
// supply one option. Hand references use the choosing player's inspection rights.
// Queries do not make choices or extend reference lifetimes. Returned objects,
// later casts and later zone visits cannot replace saved references.
// Source shields use their existing property rechecks and spell continuation.
//
// Use [ChooseDamageSourceFrom] for a non-targeted choice from explicit legal
// source references. [DamageSourceChooser] permits typed player decisions.
// The result uses [DamageSourceBinding] in the current resolution frame.
// The caller must supply the complete legal source list.
// [Game.LiveDamageSources] reads current unphased permanents and live spells.
// [ChooseDamageSourceFromLiveObjects] makes a filtered choice from those objects.
// These live-object services exclude referred and command-zone objects.
// [Game.StackAbilityPermanentSources] reads original permanent sources of
// abilities still on the stack. It uses current or retained information for
// each exact incarnation, including information saved before phasing.
// [ChooseDamageSourceFromStackAbilities] chooses only from this category.
// [Game.StackTargetPermanentSources] reads exact permanent target incarnations
// referred to by spells and abilities still on the stack. Target legality
// does not remove the saved reference. Copies keep their own target list.
// [ChooseDamageSourceFromStackTargets] chooses only from this category.
// [Game.StackTargetSpellSources] reads exact spell targets referred to by
// objects still on the stack. Departure information remains available until
// the final referring object leaves. Recasts keep separate stack identities.
// [ChooseDamageSourceFromStackSpellTargets] chooses only from this category.
// [Game.StackBoundPermanentSources] reads saved event and payment incarnations
// referred to by objects still on the stack. Announcement captures also qualify.
// Immutable event captures supply references without running their effects.
// Delayed bodies on the stack also supply saved permanent, target and token
// identities. Waiting and pending delayed triggers do not enter this category.
// [ChooseDamageSourceFromStackBindings] chooses only from these binding contexts.
// [Game.WaitingDelayedPermanentSources] reads saved permanent, target and token
// frames and exact event subjects while delayed abilities wait to trigger.
// Direct [This] reads in destruction, sacrifice and return bodies also qualify.
// [AddCountersTo], [AddCountersUpTo], [RemoveCountersFrom] and
// [RemoveAllCounters] declare direct [This] subjects in waiting bodies.
// Typed amount reads also qualify, including [SourceCounters].
// Zero amounts and full counter limits keep the saved reference.
// Queries do not read amounts, choose counters or write removal results.
// Counter execution still requires the original present incarnation.
// [TapSubject] and [UntapPermanent] declare their direct [This] subject, even
// when its tapped state needs no change. Queries do not resolve the subject
// or change its status. Execution still needs the original present incarnation.
// Single-recipient [DealDamageFrom] bodies qualify with [This] or
// [DamageSourceOf] of [This], even when the damage amount is zero.
// [DealDamageEach], [DealDamageEachPlayer] and [DealDamagePackets] bodies
// also declare these source reads without evaluating recipients or amounts.
// Empty collections and zero amounts keep the source reference.
// [IfPermanentMatches], [UnlessPermanentMatches] and [IfPermanentMatchesElse]
// declare self references in the condition and both bodies without checking
// the condition. A false condition does not remove a declared reference.
// [IfPermanentAttacking] and [IfPermanentBlocking] also declare self references
// in the subject and body. Queries do not check combat or execute the body.
// Resolution checks current combat membership for the saved incarnation.
// [IfPaid], [UnlessPaid] and [PayOrElse] declare self reads in their bodies.
// Both payment branches qualify before payment. Queries do not read the payer,
// test costs or execute actions. Cost references need separate contracts.
// [PayActionOrElse] declares self reads in either body and direct [This] reads
// in a [DamagePayment] source or recipient, or a [DestroyPayment] subject.
// Queries do not prepare the payment. Typed player and amount reads also qualify.
// [ControllerOf], [OwnerOf] and [ControllerOfWithLKI] of [This] declare the
// saved source in payment players and direct life or draw subjects.
// [LifeTotalOf], [HandSizeOf], [LifeGainedThisTurn], [DamageTakenThisTurn] and
// [ArtifactDamageTakenThisTurn] retain these declarations in typed values.
// Arithmetic reads both declarations without calculating a value.
// Queries do not resolve players, read amounts or make payment choices.
// Damage bodies declare these typed player reads in singular recipients and
// these typed values in single, collection, per-player and packet amounts.
// Queries do not select recipients, read amounts or execute damage batches.
// [CountMatchingHand] and [CountGraveyardCards] declare direct [This] reads
// in their typed player. [CountPermanents] declares these reads in typed
// controller and owner sets. Filters, unions, intersections and mana-value
// limits retain the declarations. Queries do not run filters or select members.
// Singleton, attachment, combat and block-history sets declare direct [This]
// operands. Exclusions, lowest-power sets and typed opponent-player sets keep
// those declarations. Queries do not read attachments, combat or history.
// Counts still read current membership when the delayed body resolves.
// [RevealHandFor], [DiscardMatchingHand], [Discard] and [DiscardAtRandom]
// declare direct source reads in their typed player. Both discard amounts
// retain typed source reads. Queries do not reveal, choose or move cards,
// run filters or use random results. Each body reads the current hand when
// it resolves. Zero amounts and empty hands keep the saved reference.
// Attached, bound and opaque player references need separate contracts.
// Saved sequences and modes retain the original source incarnation.
// One-shot detection ends this category; duration-based delays can keep waiting.
// [ChooseDamageSourceFromWaitingDelayedBindings] uses only these references.
// [Game.StackBoundDamageSources] reads permanent and spell source identities
// saved by event captures and delayed frames on stack objects. Queries stay separate from
// the event information used by the trigger body. Spell departure information
// uses the exact stack identity and remains separate in cloned games.
// [ChooseDamageSourceFromStackDamageBindings] chooses only from this category.
// [Game.StackBoundSpellSources] reads spell identities saved by cast events.
// Detection saves information before a pending trigger reaches the stack.
// Current queries and departure snapshots keep the exact spell identity.
// [ChooseDamageSourceFromStackSpellBindings] chooses only from this category.
// Other saved binding types need separate source contracts.
// [ChooseDamageSource] combines these services with the waiting categories.
// Use [ChooseDamageSourceMatchingFrom] for a property filter on that list.
// [DamageSourceHasColor] and [DamageSourceHasType] read the supplied information.
// [AllDamageSources] and [AnyOfDamageSources] combine those filters.
// Live references read current properties. Saved references read saved properties.
//
// Use [PreventDamageFromCardType] with [Static] for damage from sources
// with a specified card type. The current subject can be [This] or
// [AttachedPermanent]. Source types use current or retained information for
// the exact object, including spell sources. This rule has no damage limit.
// Use [DamageSourceOfSpell] for a live typed spell reference and
// [SnapshotDamageSource] to save its information before departure.
// [DamageSource.StackID] keeps separate casts and copies apart.
// [DamageSource.SameObject] compares a saved source with an exact object.
//
// Use [PreventCombatDamageFrom] and [PreventCombatDamageTo] with [Static]
// for live combat prevention. [AttachedPermanent] follows the current host.
// The rule checks source presence and incarnation for each damage event.
// [DamageSourceIsEnchanted] uses the saved Aura status for absent sources.
// Recipient ability removal does not remove a separate Aura's rule.
// Use [CantAttack] for the same live subject attack restriction.
//
// Use [MayPutFromHandOntoBattlefield] for one optional matching permanent
// card at resolution. The choice keeps its hand visit. Entry uses the selected
// player as controller and actor. This action does not cast or play the card.
//
// Use [Mill] for a player action and [MillCost] for a full payment.
// The library owner can order cards that move together to the graveyard.
// All order choices precede the moves. Payment queries make no choices.
// RON MillACard uses the typed player scope and a fixed count of one.
// A failed effect discard can select mill in a draw replacement body.
// A replaced draw from an empty library does not cause a draw failure.
//
// Use [ExileGraveyardCard] with a typed card target or [FromCard] for exile
// from a graveyard. A saved visit prevents the effect from moving a returned
// card. [PermanentDies] with WithCard saves the first graveyard visit before
// trigger placement. RON source-death bodies use this capture with
// [EventSourceIsSelf]; the capture stays separate for each pending event.
//
// Use [GainControlWhileSourceControlled] for control that ends when you lose
// control of the original source. Set requireTapped when untapping must also
// end the effect. A returned source or target cannot restore the old effect.
// Target phasing preserves a valid duration; source phasing ends it.
// The separate permission to remain tapped stays optional after target departure.
//
// Use [CantBeEnchantedExceptBy] with [Static] to permit one Aura on a subject.
// Use [AttachedPermanent] and [This] to exclude the granting Aura from the rule.
// The rule checks casting and attachment legality. Illegal attached Auras move
// to the graveyard as a state-based action; this action does not destroy them.
//
// Use [TargetPermanentYouControl] or [TargetPermanentOpponentControls] with
// [WithCastTarget] for a controller restriction on an Aura's enchant ability.
// Attachment and state checks read the Aura's current controller. Shroud does
// not prevent a direct attachment. A control change can make the Aura illegal.
//
// Use [DeclareSpellOrPermanentTarget] with [WithTargets] for one target across
// the stack and battlefield. Copies can change the target category.
// [SetObjectColor] uses this declaration to keep the original object identity.
// A permanent spell keeps its changed color on entry. Later zone changes end it.
//
// Use [UntapWithManaPayment] for an optional selection of controlled, tapped
// permanents followed by one total mana payment and one untap batch.
// All choices precede payment. Failed payment cannot untap part of the selection.
// The operation keeps each selected incarnation through mana abilities.
//
// Use [UnionPermanents] and [IntersectPermanents] for contextual collections.
// Each live incarnation appears once, in battlefield order. [AttachmentsOn]
// selects attachments on current host-set members with the same incarnation.
// [PermanentsControlledByPlayers] permits a [PlayerSet] controller restriction.
// Queries use the resolving context and cannot restore absent members with LKI.
// Use [SingletonPermanent] to use a typed permanent reference in a collection.
// It retains the reference's identity and legality checks. RON resolving tap,
// untap, return and destruction collections can combine these references with
// target collections and live filters through union and intersection.
//
// Mana payment plans save activation colors in [ManaTap] with
// [ManaActivationPayment]. The saved payment prevents an activation from
// spending colors required by the final cost. Execution checks the current
// cost and source incarnation before payment. A returned source cannot use
// its former plan. Payment queries do not spend mana or make choices.
//
// Use [DeclareActivatedAbilityTarget] with [WithTargets] to select an activated
// ability by its stack identity and source card type. Source types use current
// information or retained information for the original incarnation or zone visit.
// Mana abilities do not use the stack. Triggered abilities do not match.
// [ControllerOfAbility] reads the selected ability's controller, including copies.
// [CounterAbility] uses the shared counter operation and does not refund costs.
//
// Attack costs use [SourceCantAttackUnlessPays] or [AttachedCantAttackUnlessPays].
// Both attack declaration paths read the current source after payment. The
// original incarnation must remain present and under the declaring player's
// control. Payment can sacrifice the attacker when it matches the cost filter.
// An absent source cannot become an attacker or produce an attack event.
//
// Use [RedirectDamageToPermanentIf] for conditional player damage redirection
// from matching permanent sources. [DamageSourceIsUnblocked] reads the status
// set after blocker declaration, or the saved departure or phasing status.
// Use [RedirectDamageFromCardTypeToPermanentIf] to include matching sources
// outside the battlefield. Both rules use live player and destination references.
// Redirection preserves the source, amount and combat status for later replacements.
//
// # Engine Principles & Invariants
//
//   - Pure Queries: Readers and query methods never mutate game state or consume RNG.
//   - Clone Isolation: Cloned games ([Game.Clone]) must have strictly independent
//     state for tree search and AI simulation; mutations in one game never affect another.
//   - Transactional Cost Payments: Spell casting and ability activations verify and pay
//     costs atomically on an isolated clone before committing to the live game state.
//   - Durable Incarnations: Battlefield permanents track a unique incarnation ID
//     independent of their physical card UUID to prevent leave-and-return reference confusion.
//
// Draw-step replacements use [Static] with [SkipDrawStepForLife]. The rule
// follows the current controller and remains active only while the source has
// the ability. A pending gain keeps the player who skipped the step.
//
// Use [ChooseRandomPermanent] with [MatchingPermanents] to save a random
// battlefield reference. Use [TapSubject] with [From] to tap that reference.
// An empty collection does not consume a random value.
// Use [PreventDamageThisTurn] with a typed recipient for unlimited prevention.
// A zero source filter permits all sources. [DamageSourceIsAttacking] reads
// current combat status or the saved departure or phasing status of the source.
// Use [Until] with [GrantKeywordModification] and CantRegenerate for a
// prohibition that lasts until end of turn, even after its source leaves.
//
// Use [DealDamageEachPlayer] for amounts that depend on each selected player.
// Its player binding is local to the amount. All amounts are read before one
// simultaneous damage batch. Replacement effects do not recalculate them.
// Use [DivideRoundDown] and [DivideRoundUp] with [Fixed] of two for half values.
// [DeclareOpponentOrPlaneswalkerTarget] excludes the controller only from
// the player targets. A copied spell checks against its own controller.
//
// [PreventNextDamage] permits allocation across simultaneous damage sources.
// The affected player or recipient controller chooses within legal bounds.
// Native combat saves all assignments before the damage batch starts.
// Use [PreventNextCombatDamageFromPermanentExcept] to leave a saved amount
// in the next combat event from one exact creature to one player this turn.
// Matching simultaneous packets share one amount. The shield remains if it
// prevents no damage. A returned creature cannot use the saved shield.
// First-strike and normal damage steps use separate batches.
// [BandingDamageAssigner] lets a player assign each blocking source separately
// to an attacking band. Each source keeps its own damage information.
// The old [BandingDamageDistributor] can still set the total for each recipient.
// Normal nontrample attackers use [CombatDamageAssigner] to divide their
// complete amount freely among current blockers. There is no lethal order.
// Invalid maps use a legal default. Single blockers need no choice.
// Outgoing attacking-band damage uses [CombatDamageAssigner] for each source.
// A blocking band uses [BandingDamageAssigner] or the existing distributor.
// Each source keeps its own trample and deathtouch. Shared assignments can
// supply lethal damage before a source with trample damages the defender.
//
// Use [PreventNextDamageFromExactSource] for one damage event from an exact
// permanent, spell or command reference to a player this turn. Property checks use
// current or retained information for that object. A returned permanent or
// another cast cannot consume the shield. Bindings can change after installation;
// each shield keeps its own source. [PreventNextDamageFromSelectedSource] adds
// permanent-spell continuity for a source chosen before that spell resolves.
//
// Use [PreventNextDamageFromSelectedSourceWithOperations] for an event amount.
// [PreventProposedDamage] reads [ProposedDamageAmount] or
// [HalfProposedDamageRoundedDown] when the rule applies. Matching simultaneous
// packets share one amount. Replacement order can change that amount.
// No actual prevention keeps the rule for a later event.
// [GainLifeFromPreventedDamage] can follow prevention for a separate saved player.
// It uses that rule's actual prevention after the original damage event.
// The rule is consumed before this gain can cause further damage.
// Life modifiers, replacements, history and events use the shared life service.
// Each saved rule keeps its own source, player and outcome. Values do not escape
// the event or read resolution bindings. Selected spell continuity still applies.
//
// Use [PreventNextDamageFromPermanent] for one damage event from a typed
// permanent source to a player this turn. The filter is checked at installation
// and before damage. A returned source cannot consume the old shield.
// All matching packets in one simultaneous event use the shield together.
//
// Use [MultiplyValues] to multiply two resolution values. Procrastinate uses
// twice the announced X with [AddCountersTo] after [TapSubject].
//
// Use [IfSacrificePaid] or [UnlessSacrificePaid] for an optional payment of
// one typed permanent. The payer must control the original incarnation.
// Returned and phased permanents cannot pay. Zone replacements can change
// the destination without changing successful payment.
//
// Use [PlayerControls] with [FromPlayer] for a saved event player condition.
// Detection and resolution keep separate player bindings for each event.
// [OpponentControlsMoreThanYou] compares current matching permanent counts.
// Phased permanents and equal counts cannot satisfy that comparison.
//
// Use [TargetControlledBy] with [You] or [FromPlayer] of an event binding.
// [CombatDefendingPlayer] selects the nonactive player during two-player combat.
// Target control checks use this player even when the defending player casts
// the spell. An attack event can keep its separate saved defending player.
// Target selection and later checks read the saved event player. Source
// control changes and copies do not change that player. [AttacksUnblocked]
// with WithDefendingPlayer saves the player before target selection.
//
// Use [IfRequiredSacrificePaid] or [UnlessRequiredSacrificePaid] for mandatory
// payment of one typed permanent at resolution. The payer must control the
// exact present incarnation. Missing, phased and returned subjects cannot pay.
// Payment has no optional offer. The shared transaction and replacement
// pipeline determine success. Each invocation keeps its own payment result.
// Waiting delayed queries retain explicit source reads without making payment.
// RON MustCost(SacrificePermanent(ref)) supports an adjacent If or Unless
// with CostWasPaid or CostWasntPaid in resolving action lists.
//
// Use [IfRequiredPaid] or [UnlessRequiredPaid] for a mandatory matching
// sacrifice payment during resolution. The branch keeps its own result.
// Source exclusion uses the original incarnation, so a returned source can pay.
// [SacrificePermanentAction] permits exile replacements through the shared
// pipeline. A replaced sacrifice can pay without causing a death event.
//
// Use [ManaValueOf] to read a permanent through its typed reference. Use
// [SnapshotNumber] before [Destroy] when a later instruction needs that value.
// The saved number remains available after the permanent leaves.
// Swords to Plowshares saves [PowerOf] and [ControllerOf] before [Exile],
// then uses the saved values with [GainLifeFor]. Exile metadata stays distinct
// from destruction metadata when effects are combined.
// Use [SnapshotPlayer] with [ControllerOf] to save its controller before
// destruction. Crumble combines both snapshots with [DestroyWithoutRegeneration]
// and [GainLifeFor]; the saved controller gains life even if destruction fails.
//
// Use [WhenEvent] with [Enters] and [TapSubject] of [AttachedPermanent]
// to tap an enchanted permanent on entry. The attachment is read at resolution;
// a departed source uses its last attachment and cannot follow a new incarnation.
//
// [AttachedBecomesTapped] saves the event permanent before trigger placement.
// Read it with [From] so attachment changes cannot redirect the effect.
//
// Targeted destruction uses [Destroy] with a typed permanent target. Declare
// that target with [WithTargets] on the spell, mode, trigger, or activation.
// The target filter supplies the restriction; the action allows regeneration.
// Use [UnlessPaid] with [You] and [Destroy] of [This] for a granted
// upkeep payment. The trigger keeps its controller and source incarnation.
// Land destruction uses [IsLand] with the same target and action constructors.
// To destroy all lands, use [DestroyEach] with [MatchingPermanents] and [IsLand].
// Use [HasSubType] for a land subtype. Combine [IsCreature] and
// [HasColorFilter] with [And] to restrict destruction by creature color.
// Combine type filters with [Or] in one collection to destroy each matching
// permanent once. Separate destruction actions can consume extra shields and
// expose intermediate states to triggers.
// The collection is selected at resolution and does not include phased permanents.
// Use [DestroyEachWithoutRegeneration] when the instruction forbids regeneration.
// This restriction applies to the action and does not change permanent keywords.
//
// Use [CantBeBlocked] with [Until] for a temporary blocking restriction.
// It uses combat rules, so ability removal does not erase the restriction.
// The subject keeps its captured incarnation; source departure does not end it.
// Use [Activate] with [Costs] and [WithTargets] for tap-cost blocking effects.
// Put power limits in the target filter. Later power changes do not end the effect.
//
// Use [RemovePermanentFromCombat] with a typed reference to remove combat
// membership. An attacker stays blocked when its last blocker is removed.
// Use [RemovePermanentFromCombatAndUnblock] when the instruction makes
// historically sole-blocked attackers unblocked. Both actions declare an
// explicit [This] subject to [Game.WaitingDelayedPermanentSources]. Queries
// do not resolve subjects or change combat. Departure and phasing keep the
// saved incarnation until trigger detection. A returned object cannot replace
// it. Implicit attachments and future delays keep separate scopes.
// [CantBlock] with [Until] restricts later blocks and keeps its captured
// incarnation until cleanup.
//
// Use [ManaCostPayment] for announced X activation mana. The payment
// transaction substitutes X before an activation cost reduction. The stack
// keeps the announced value for target counts and resolution.
//
// Use [AddRestrictedManaOfAnyOneColor] for a resolution amount with one
// spending rule. The controller chooses one color for the full amount.
// The mana pool and payment planner keep the rule on each mana unit.
// A nonpositive amount adds no mana and requires no color choice.
//
// Use [GrantLandPlays] to add to a player's land allowance for this turn.
// Grants accumulate and remain separate in cloned games. The action does not
// play a land or change land timing. Explore grants the allowance before drawing.
//
// Use [Activate] with [ModifyPT] and [Until] for targeted temporary boosts.
// Put mana, tap, sacrifice, and discard costs in [Costs]. A source sacrificed
// to pay a cost does not end the boost on the target.
// Use [DiscardRandomCost] for a random hand payment after deterministic costs.
// The payment keeps hand visits and random results separate for each use.
// Failed payment checks do not spend live resources or scripted random results.
// Put subtype and current power/toughness limits in the target filter.
// These limits are checked again at resolution, not during the boost.
// Icatian Lieutenant uses a Soldier creature target; Icatian Priest uses any
// creature target. Both use mana costs and an end-of-turn duration.
// Ghosts of the Damned and Pradesh Gypsies use negative power modifiers.
// Tuknir Deathlock uses the same target and duration with a +2/+2 bonus.
//
// Use [Until] with EndOfCombat for captured singular layer modifications.
// The effect stays active through end-of-combat triggers and priority.
// Native turn execution ends it when the combat phase ends. Source departure
// and removal from combat do not end it. A phase cycle keeps the subject;
// a returned permanent cannot inherit the modification. Clones expire separately.
// Glyph of Destruction keeps its turn prevention and delayed destruction
// separate from this duration.
//
// Use [Until] with [ModifyEachPT] for a temporary collection boost.
// [ControlledPermanents] with [IsCreature] selects your creatures at resolution.
// Shield Wall uses this collection with a +0/+2 bonus until end of turn.
// [MatchingPermanents] with [IsCreature] and [IsAttacking] or [IsBlocking]
// selects combat creatures for either player. Combat removal does not end
// the boost. Later entries and returned cards do not receive the boost.
// Use [MatchingPermanents] with [IsCreature] for global P/T reductions.
// Fixed negative P/T summaries report a detrimental outcome.
//
// Use [AnimateEach] with [Static] to add the creature type and set base values.
// Supply colors to replace current colors. The selection stays the same across
// layers in one application, even if the source loses its ability in layer 6.
// Each later application uses current membership.
//
// Use [SkipUntapSteps] with [Static] to replace each player's whole untap step.
// Source departure, phasing and ability removal stop the replacement.
// Turn-start control checks still permit creatures controlled from earlier turns.
// Use [LimitUntapsDuringStepIf] to check a condition before untap selection.
// Phasing precedes this check. Untapping cannot change the saved selection.
// Use [LimitUntapsDuringStep] with [Static] for a fixed untap-step limit.
// The active player chooses a set that satisfies all limits before untap.
// Type overlap counts against each limit. Other untap effects remain legal.
// Use [PreventUntapDuringStep] with a typed subject and state condition for
// a live source or Aura-host restriction. The untap decision reads both before
// any permanent untaps. Recipient ability removal does not remove this rule.
// Other untap effects can still untap the subject.
//
// Use [PreventUntapDuringStepForMatching] for a static untap-step restriction.
// It checks final characteristics before the untap operation. Other untap
// effects remain legal. Recipient ability removal does not remove the rule.
//
// Use [PreventCombatDamageByThisTurn] with a typed permanent reference for
// combat damage from that source until cleanup. It keeps the captured
// incarnation after control changes or a phase cycle. A returned permanent
// has a new incarnation. [PreventCombatDamageBy] still ends at end of combat.
//
// Use [CantBeBlockedExceptBy] with [While] for a static blocking restriction.
// With [Until], the rule keeps the selected incarnation until cleanup.
// Each blocker must pass its filter and all other combat restrictions.
//
// Use [CantAttackEach] with [Static] for a current collection attack restriction.
// It checks final characteristics at declaration. [ControlledPermanents] reads
// the current source controller. Later rule changes do not remove attackers.
//
// Use [SourceCanBlockOnly] with [Not] to prevent the source from blocking
// matching attackers. Use [CantBeBlockedExceptBy] with [Not] for matching
// blockers. Filters read current characteristics at the block legality check.
// Clones keep both filter maps for queries before the next layer pass.
//
// Use [AttackAsThoughHaste] for an attack permission that leaves tap costs
// restricted by summoning sickness. It does not grant the haste ability.
// The permission keeps its subject incarnation with [Until] and reads the
// current subject with [While]. Use [GrantKeywordModification] with
// AttrCanAttack to permit attacks through defender.
//
// Use [UntilNextTurn] to save the duration player at resolution.
// [BeginActiveTurn] ends the effects before that player's next turn actions.
// Skipped turns do not end the duration. Extra turns do.
// Source changes and phasing do not change the endpoint. Clones stay separate.
// [NextTurnExpiration] gives installed layers and replacements the same endpoint.
// RON lowers UntilPlayersNextTurn(player) in temporary permanent layer actions.
// Temporary player operations still require their own checked consumers.
//
// Use [UntilNextUpkeep] to save the duration player at resolution.
// The effect ends before that player's next upkeep triggers are detected.
// Source control changes, departure, and phasing do not change this duration.
// Subject references keep their captured incarnations.
// Use [UntilEndOfNextUpkeep] when the effect must last through upkeep priority.
// [ToughnessOfWithLKI] keeps the exact incarnation for a saved toughness read.
//
// Use [SourceTappedStateWithLKI] for a source tapped-state check that needs
// departure information at resolution. Live-only state conditions remain
// available for static membership checks. Draw-step events are detected before
// the normal draw; pending triggers go on the stack after that draw.
//
// Use [ControllerOfWithLKI], [PowerOfWithLKI], and [ManaValueOfWithLKI]
// when an information read must survive departure. They use the exact source,
// attachment, saved permanent, or singular target incarnation. Phase information
// remains separate from departure information. These reads do not permit actions
// on absent objects. The existing live-only queries keep their behavior.
//
// Use [BlocksCreatureMatching], [BecomesBlockedByCreatureMatching], and
// [BlocksOrBecomesBlockedByCreatureMatching] for creature pair events.
// Supply [This] or [AttachedPermanent] and a permanent filter. WithOpponent
// saves the opposing incarnation before trigger placement. Later attachment
// and characteristic changes do not change that saved object.
//
// Use [NewConditionalManaAbility] for fixed mana branches with one tap cost.
// Payment queries and activation use the same current-state condition.
// The condition must stay the same when the source taps.
//
// Use [RevealHandFor] to show a player's current hand to all players.
// [DiscardMatchingHand] moves all matching hand cards before discard events.
// [CountMatchingHand] reads the current count without reveal history or LKI.
//
// Use [RevealRandomHandAndDiscardMatchingName] for a partial random hand reveal.
// It selects distinct cards, then discards only matching revealed cards.
// Use [RevealHandAndDiscardRandomMatching] to reveal the whole hand and discard
// one random matching card. Both operations read the current hand at resolution.
// All reveals precede shared discard replacements, moves and batch events.
// Owners can order simultaneous graveyard arrivals before any card moves.
// Empty selections use no randomness. Clones keep separate hands and results.
// These operations do not produce a saved card result. Delayed bodies retain
// chosen-name bindings and restore the outer name frame after execution.
// Typed players and amounts declare source reads without executing during queries.
//
// Use [RevealRandomHandCardInto] for a public random hand reveal at resolution.
// [FromCard] keeps the exact saved hand visit. Departure and return end that
// reference. Empty hands and absent players clear it without random selection.
// [DiscardCardFromHandFor] checks the explicit player before shared discard
// replacements and events. Public reveal history keeps separate card snapshots.
// Nested execution, permanent loops and clones keep separate card bindings.
// Typed players declare source reads to waiting delayed queries. These queries
// do not reveal cards, read the hand or use randomness.
// Result predicates, saved characteristics and library reveals need separate
// contracts. A saved card reference does not permit an action on an absent card.
//
// Use [RevealLibraryTopInto] for a public top-card reveal at resolution.
// It saves name and type facts with an exact library visit in a [CardBinding].
// [FromCard] cannot follow departure, return, shuffle or a reordered region.
// A continuously revealed top gets a new visit when its visibility ends.
// Removal of other cards keeps retained visits. Reveal predicates retain their
// historical facts after identity ends. An empty library clears the result.
// Clones and saved resolution frames keep separate visits and results.
// Typed player operands declare source reads without revealing during queries.
// RON supports RevealTopCardOfLibrary in resolving player action lists.
// Unconditional PlayerActions producers retain their lexical result outside
// that player scope. Linked exile needs separate adapters.
// Use [PutLibraryTopInHand] or [PutLibraryTopInGraveyard] for a current top.
// Both move to the owner's zone before the zone event. Empty libraries do
// nothing. Neither action draws or mills; draw and mill replacements do not
// apply. The shared library and destination services create the new visits.
// Typed players declare source reads without movement during waiting queries.
// RON supports PutTopOfLibraryInHand and PutTopOfLibraryInGraveyard as symbols
// in resolving player actions. These actions do not read a saved reveal result.
//
// Use [LookAtHand] to show a current hand only to a typed viewer.
// [Game.HandLooks] returns independent snapshots of past observations.
// [HandLookObserver] receives separate card copies during resolution.
// Inspection keeps hand visits, public visibility and zones unchanged.
// Copies use their own controller as viewer. Clones keep separate history.
// Both player operands declare source reads to waiting delayed queries.
// Queries do not inspect hands. Missing players and empty hands do nothing.
// RON supports LookAtPlayersHand in resolving bodies with typed player scope.
// Controlled play requires separate choices and player-control services.
//
// Use [LookAtLibraryTop] for private inspection and optional top ordering.
// [LookAtLibraryTopNumber] reads a typed count once at resolution.
// Both forms and [MayShuffleLibrary] declare explicit source reads in their
// players and count to [Game.WaitingDelayedPermanentSources]. Queries do not
// inspect cards, read counts or offer choices. Zero counts keep the reference.
// Departure and phasing keep the saved incarnation until trigger detection.
// The viewer and library player are separate typed references.
// [MayShuffleLibrary] lets the chooser decide after the inspection.
// [Game.LibraryLooks] returns copied history for the selected viewer.
// Optional [LibraryLookObserver] and [LibraryTopOrderChooser] interfaces
// supply private display and ordering without a change to [Player].
//
// Use [SearchLibraryToHand] for a required single-card search, followed by
// a shuffle. Use [SearchLibraryToBattlefield] with a card filter for a search
// that can find no card. A zero filter requires a card when one is available.
// The destination move occurs before the shuffle. A prohibited battlefield
// entry keeps the card in the library. These searches do not reveal hand cards.
//
// Use [AddPoisonFor] with a typed player reference to add poison counters.
// Its typed player and amount declare explicit source reads to
// [Game.WaitingDelayedPermanentSources]. Queries do not resolve operands
// or add counters. Zero amounts keep the saved incarnation until detection.
// Departure and phasing keep the reference; a return cannot replace it.
// Implicit attachments and future delay bodies keep separate scopes.
// Nonpositive amounts make no change. The existing state check handles loss
// at ten poison counters. [TokenAbilities] gives each token its own ability
// identity; event bindings keep their values in the resolution frame.
// Static token abilities keep separate continuous effects and source identities.
// Definition validation uses the token as the source of its own abilities.
//
// Use [ShuffleHandsAndGraveyardsIntoLibraries] before a separate draw action.
// Use [ShuffleHandsIntoLibrariesAndDraw] to draw each player's moved count.
// All moves finish before zone events, and all shuffles finish before draws.
// Counts stay local to each resolution. Empty hands still cause a shuffle.
//
// Use [LoseUnspentManaFor] to clear one selected player's mana pool.
// Its typed player declares explicit source reads to
// [Game.WaitingDelayedPermanentSources]. Queries do not resolve the player
// or clear mana. An empty pool keeps the declaration. Departure and phasing
// keep the saved incarnation until trigger detection. A returned source
// cannot replace it. Implicit attachments and future delays infer no reads.
// Use [HasManaAbility] to check current abilities without testing payment
// or current output. [ControllerOfSpellWithLKI] keeps departure control
// for instructions that run after a spell is countered.
//
// Use [TargetManaValueEqualX] for a permanent target with mana value equal to X.
// [DeclareSpellTargetManaValueX] includes the target spell's own X in its value.
// Target checks use the action's announced X before resolution begins.
//
// Use [ModifyPT] with [While] for live static P/T values.
// [CountPermanents] and [AttachedAuras] read current membership on each layer pass.
// [CopySpellOnStack] can retarget Aura casting declarations. Permanent spell
// copies enter as tokens with separate ability and static effect state.
//
// Use [FilterPermanentSet] to keep current matching members of a typed set.
// The original set keeps its target and incarnation checks.
// Use [PermanentDamageRecipients] with [DealDamageEach] for one simultaneous
// damage batch to permanents. Players are excluded from this recipient set.
//
// Use [CantBeTargetedBySpells] with [Static] for a live spell restriction.
// [SpellTargetSource] identifies spells during target queries before stack entry.
// Ability sources use their separate source information. An Aura spell targets;
// a direct attachment does not. Source ability removal ends its own rule.
//
// Use [ShuffleGraveyardFor] so graveyard moves finish before zone events.
// The library is shuffled even when that graveyard is empty.
// RON exile payments use [ExileMatchingCardFromGraveyardCostInto] to keep
// a separate paid mana value for each pending activation.
// Use [ChooseOpponentAsEnters] for an opponent choice before battlefield entry.
// Each permanent keeps its choice through control changes and phasing.
// [EventIsChosenPlayerUpkeep] checks that choice when the upkeep event occurs.
// [ChosenPlayerOf] reads a live permanent's entry choice without LKI.
// Use [SetMaximumHandSizeStatic] or [NoMaximumHandSizeStatic] with a typed
// player reference. [Game.MaximumHandSize] and cleanup read active rules in
// timestamp order after the object layers. Phasing and source departure stop
// the rule. An entry choice stays fixed when the source changes controller.
// Use [ChooseBasicLandTypeAsEnters] for a land choice before battlefield entry.
// [Permanent.ChosenBasicLandType] keeps the typed choice separate from color.
// [BecomesChosenBasicLandTypeAttachedEffect] applies it to the current host.
// Use [MayEntryOperations] for an optional sequence before battlefield entry.
// The entry controller makes the choice. Decline permits normal entry.
// Accepted operations run in order. A rejected child stops the entry.
// [EnterTappedAsEnters] sets tapped status within such a sequence.
// Phasing keeps the result; a new entry requires a new choice.
//
// Use [AddCountersUpTo] for an optional counter amount with an action limit.
// The effect controller chooses at resolution. The limit still applies after
// replacements, because this action cannot cause an excess on its subject.
// Other actions can add counters above this limit.
//
// Use [GrantTriggeredAbility] or [GrantEachTriggeredAbility] with [Static]
// for a live trigger grant. The selection uses the donor's context.
// The trigger uses each recipient as its source and reads its controller
// at detection. Bindings belong to the granted ability. A pending trigger
// keeps its source incarnation after the grant ends.
//
// Use [TriggeredAbilityModification] with a duration for a captured trigger grant.
// The recipient has its own source and binding context. A returned permanent
// cannot receive the old grant. [HasRampage] reads current rampage abilities.
// RON Gabriel Angelfire and Rapid Fire use the shared rampage effect and event.
//
// Use [PermanentAttackedDuringLastTurn] for the live source or attachment's
// attack history during one player's last completed turn. Cleanup saves exact
// incarnations. Phasing keeps history; returned objects cannot use old history.
// Use [CantAttackIf] for a live conditional attack restriction. A later condition
// change does not remove attackers already declared.
//
// Use [RevealAllHands] or [RevealAllLibraryTops] with [Static] for public
// visibility while the source ability is active. Control changes do not change
// the affected players. [Game.IsHandRevealed] reads current hand visibility.
// Clones retain separate hand and library-top visibility flags.
//
// Use [ChooseAtResolution] with [ResolutionOption] for a choice made while an
// effect resolves. Each option runs in the existing resolution frame.
// Supply a pure When condition if an option has a legality restriction.
// Costs and targets are selected before this choice. Copies choose separately.
// RON ChooseAnAction lowers nonempty action lists with the shared sequence
// compiler. Each branch keeps local payment, draw and number results.
// Draw replacements and delayed bodies keep their execution context.
//
// Automatic mana payment can use fixed mana-only conversions without a tap.
// The reduced cost must consume at least the produced mana count. Repeated
// conversion keeps the source available, even when it is tapped or newly entered.
// It does not receive tap bonuses or emit a tapped-for-mana event. X costs,
// activation limits and arbitrary conditions are not planned.
// Exact payments keep the final cost colors. The shared cost transaction
// keeps failed casting payments separate from live resources.
// Fixed-color mana bodies can include [IfActivationsAtLeast] with [AtNext].
// Planned payment runs the complete body after it records the activation.
// Each successful activation can install its own delay. Queries and failed
// payments do not change history or install delays.
//
// Use [AttachedPermanentDies] to capture an attached permanent's death.
// It permits any permanent type. [AttachedCreatureDies] still requires a creature.
// Controller, owner, toughness, permanent and card bindings save event facts.
// The card binding keeps the first graveyard visit. A later visit cannot replace it.
// RON Creature Bond uses these facts with [DealDamageFrom] and the Aura as source.
//
// Use [SkipNextUntaps] for a fixed count of future controller untap steps.
// Overlapping restrictions count the same steps. Other untaps remain legal.
// Use [SkipNextUntapEach] to capture the recipients of one-step restrictions.
// Control changes and phasing preserve counts; departure ends them.
//
// Use [WhenAttacksUnblockedThisTurn] to watch one incarnation this turn.
// Each legal blocker declaration can trigger it. Source departure and control
// changes do not change the saved controller. Cleanup ends the subscription;
// pending triggers retain their saved body and references.
//
// Use [WhenDiesThisTurn] or [WhenLeavesThisTurn] for a delayed departure.
// Each installation keeps the watched incarnation, controller and body bindings.
// Cleanup removes the subscription. A pending trigger can still resolve.
// Destruction replaced by exile sends a leave event, but no death event.
//
// Use [TargetChosenByOpponent] for an opponent choice at announcement or
// trigger placement. [TargetOpponentControls] adds a separate control condition.
// Target legality uses the action controller and source. Modal activation keeps
// the first target choice. A copy uses its controller's permission to retarget.
//
// Use [SourceAttackHistoryThisTurn] for the original source's current-turn
// attack history. A captured incarnation keeps that history after departure.
// Use [CreatureDiedThisTurnCondition] for a creature death under either player.
// Both conditions read current history at detection and resolution.
// Cleanup clears this history. Information reads do not permit absent actions.
//
// Use [ReplaceTappedManaTypeForPlayerThisTurn] to change all mana types from
// matching permanents controlled by a saved player when that player taps them
// for mana. Cleanup ends the rule. Source departure does not end it.
// [ReplaceTappedManaTypeUntilLeaves] changes one type from a saved incarnation.
// Phasing keeps that rule; departure ends it permanently. Amounts and spending
// restrictions remain. Actual types supply tap bonuses and mana events.
// Manual production and automatic payment share replacement candidates and order.
// Planning makes no choices or mutations. Stable rule IDs survive game cloning.
// [Game.ManaTypesAbilityCouldProduce] includes all possible replacement orders
// without checking payment or tapped status. Raw metadata stays separate.
// [Game.WaitingManaReplacementPermanentSources] supplies exact active subjects
// to general damage-source choices. Phasing keeps references; departure ends them.
//
// Use [SacrificeChosenBy] for a required sacrifice choice at resolution.
// The sacrificing player and chooser have separate typed references.
// The choice includes only present subjects controlled by that player.
// Empty sets do nothing. Saved choices cannot follow returned objects.
// Phased subjects cannot move. Shared services keep replacement order,
// owner destinations, source information and sacrifice events.
// RON supports resolving SacrificeAPermanent and the explicit chooser form.
// Saved sacrifice results need a separate contract.
//
// Use [ControllersSacrificeEach] for a typed permanent collection at resolution.
// Each present subject uses its current controller. The effect controller and
// owner do not restrict selection. Subjects keep their exact incarnations.
// Phased and returned subjects cannot supply a move. Duplicate subjects move once.
// Replacement choices use active-player order before any movement. Owners order
// their simultaneous graveyard arrivals before the shared zone batch moves cards.
// Tokens supply no retained graveyard card. Events keep the sacrificing player.
// Leave triggers retain the former battlefield and source abilities.
// Waiting queries declare explicit source reads without selecting or moving.
// RON supports ControllersSacrificeEachPermanent with existing resolving
// permanent collections. State-trigger collections need a separate contract.
//
// [OriginalPrintingNameCatalog] supplies the fixed ARN and ATQ lists from
// CR 206.3a and 206.3b. These lists do not use physical expansion symbols,
// registry membership or mutable catalogs. Unsupported set codes return errors.
// [CardNameCatalog.Matches] checks supplied names and interchangeable names.
// [HasNameIn] reads a permanent's current name; face-down subjects cannot match.
// Token status is separate. Collections, targets, layers and LKI readers keep
// their existing identity and timing contracts. Computed P/T fields are reset
// for face-down subjects each layer pass; face-down base values stay fixed.
// RON supports both original-printing symbols in permanent-filter contexts.
// Spell and land restriction adapters need separate contracts.
//
// Use [WithActivationManaXFromSourceCounters] for source-counter mana X.
// The live source defines payment X before target checks and payment.
// Queries do not change the resolution frame. All X symbols share the count.
// Cost reductions apply after X. Later counter payment cannot change the cost.
// Copies retain the payment X; clones keep separate counters and stack objects.
// Resolving counts use [CountersOnWithLKI] for the original incarnation.
// RON supports one source-counter XIs modifier with an X mana payment.
// Dynamic ValueX reads in bodies and targets need a separate contract.
//
// Use damage event WithSourcePermanent to save the exact damaging permanent.
// DealsDamage and DealsDamageToPlayer keep this reference separate from the
// recipient and damage-source bindings. Detection saves its incarnation.
// Reattachment and return cannot replace it. Current controller reads and LKI
// use ControllerOfWithLKI. Pending triggers, copies and clones keep the result.
// RON supports source or Aura-host damage to AnyPlayer or SinglePlayer(You).
// These bodies can read Trigger_ThatPermanent and Trigger_AmountOfDamageDealt.
// Existing source-to-Opponent bodies also support the captured damage amount.
// Host-to-Opponent events and combat-only source captures remain unsupported.
//
// Use [SacrificeChosenBy] with [SacrificeSubtypeInto] for a saved subtype result.
// [SacrificeMatchingIncludingSourceCostWithSubtype] supplies the same cost result.
// Each [NumberBinding] keeps a completed sacrifice match before departure.
// Exile replacements keep the result. Cancellation and absent subjects clear it.
// Copies, clones and delayed bodies keep separate saved numbers. Payment uses
// the shared transaction. Returned objects cannot change historical results.
// RON permits one subtype per local producer in resolving If branches.
// Basic land types, Desert and creature subtypes use separate family checks.
// Casting results, other properties and alternative branches stay separate.
//
// Use [BindPlayerNumbers] to declare separate results for the two players.
// [PlayerNumber] reads one member by player identity. [DiscardUpToEach]
// saves completed discards after one shared batch. All choices precede moves.
// Saved hand visits cannot follow returned cards. Library replacements count;
// cancellation clears the result. Existing number frames keep copies, clones,
// delayed bodies, loop frames and cleanup separate. RON lowers all-player
// fixed-maximum discard and lexical result reads. Mind Bomb uses the shared
// per-recipient damage service. Coin rounds use [RepeatHeadsOrTailsDamage].
//
// Use [PermanentMatchesFilterWithLKI] for a source or attachment condition.
// It reads the exact current object or its saved departure or phase information.
// [WithInterveningIf] checks this information at detection and resolution.
// Saved source visits keep their last attachment after departure. Live sources
// read their current attachment. Returned objects cannot replace saved visits.
// Use [WithDeathNotSacrificed] for one typed permanent death event.
// Each pending trigger keeps its own death cause for the resolution check.
// Copies and clones keep those facts. Other event contexts, duplicate guards
// and a second intervening condition fail definition validation.
// RON supports source-entry keyword conditions and WasntSacrificed death guards.
// Acquired host keyword removal uses [AcquireStaticAbility].
//
// Use [AcquireStaticAbility] for one captured layer 6 static ability.
// The live recipient supplies its source, controller and attachment context.
// Each acquisition keeps a separate declaration and original battlefield visit.
// Shared layer order permits an earlier removal followed by acquisition.
// Later removal stops the rule. Reattachment changes its static timestamp.
// Phasing suspends the rule; return cannot inherit it. Clones stay separate.
// Explicit source recipients supply waiting references without rule execution.
// RON lowers one Aura-host keyword removal in an acquired PermanentLayerEffect.
// Other layers, nested grants and temporary acquisition need separate adapters.
//
// Use [CantRegenerateCreaturesDealtDamageByThisTurn] for creature damage history.
// Recipient types are saved at actual damage. Later type changes keep the rule.
// [CantRegenerateDealtDamageByThisTurn] uses a live recipient filter.
// [ExileDealtDamageByThisTurn] uses live history through shared death replacements.
// Both save the damaging source visit at resolution and end at cleanup.
// [Game.CanRegeneratePermanent] checks restrictions before replacement choices.
// Source departure, phasing and ability removal do not end resolved rules.
// Returned objects cannot inherit them. Clones keep separate history and rules.
// RON supports resolving history collections and the two Runesword consumers.
//
// Use [AtNextThisTurn] for one future step trigger before cleanup.
// Its schedule expires separately from repeat behavior. Detection consumes it;
// a pending body keeps its saved frame after cleanup or source departure.
// Clones keep separate schedules. [AtNext] retains its unbounded lifetime.
// RON lowers AtNextEndOfCombatThisTurn with the existing capture service.
//
// Use [PermanentsThatDealtDamageToThisTurn] with a saved recipient visit.
// The query keeps history after departure or phasing without retained LKI.
// Present candidates use current membership; returned visits cannot match.
// [SetEachBasePowerToughness] with [Until] and Indefinite captures that set
// at resolution. Base values use layer 7b before bonuses and counters.
// Cleanup does not end this effect. Phasing keeps its saved subjects.
// RON lowers one fixed SetPT pair in CreateEachPermanentLayerEffect.
// Source death rules keep separate permanent and graveyard-card bindings.
//
// Use [ReturnChosenGraveyardCard] for a required graveyard choice and return.
// The effect controller chooses from the saved player's current graveyard.
// [ReturnChosenGraveyardCardToOwner] uses the selected card's owner for entry.
// Empty sets do nothing. Invalid answers use the first eligible card.
// A changed graveyard visit cannot supply the return. Choices stay local.
// Shared entry services keep replacement order, entry rules and new identity.
// Typed player references keep source information and waiting queries.
// RON supports one creature filter, one bound player and one entry operation.
// Delayed death bindings and death-result loops need separate adapters.
//
// Use [WhenDiesThisTurnWithBindings] to save one watched death event.
// The body gets separate permanent, owner and controller bindings.
// Detection saves the exact visit and player facts before trigger placement.
// Shared death capture reads the former battlefield for simultaneous moves.
// Each pending body keeps its facts through cleanup, source departure and clones.
// Waiting and stack queries keep the exact watched visit. Returned objects
// cannot replace it. Saved body frames restore outer bindings after execution.
// RON lowers these reads in one UntilEndOfTurn delayed death ActionList.
// OwnerOfPermanent and ControllerOfPermanent of Trigger_ThatPermanent read
// the saved event players. Other owner contexts need separate adapters.
//
// Use [RequiredTapOrElse] for mandatory payment with one saved permanent.
// Each operation keeps its own paid and unpaid body. The shared transaction
// requires that exact present, unphased, untapped visit. Return cannot pay.
// A written tap instruction permits summoning sickness. Control restrictions
// belong to the declaration. Payment makes no optional offer or subject choice.
// Delayed queries retain source reads without paying. Copies and clones keep
// separate outcomes. RON lowers MustCost(TapPermanent(ref)) with an adjacent
// If or Unless condition. Aura activation costs can tap HostPermanent through
// [TapAttachedPermanentCost]. Existing source tap-symbol costs stay unchanged.
// Energy Tap reads [ManaValueOfWithLKI] for one fixed-color mana production.
//
// Use [SpellFilter] for immutable spell predicates shared by targets and events.
// [SpellCardMatches] keeps current card information. [SpellControlledByYou]
// reads the action controller, even after source departure or a control change.
// [SpellSharesColorWithPermanents] selects present battlefield members at each
// check. Returned members can qualify; phased members cannot. Only the five
// colors can match. [AllSpellFilters], [AnySpellFilter] and [NotSpellFilter]
// keep nested predicates. Zero filters and empty groups fail definition checks.
// [SpellTargetBinding.WithFilter] keeps exact stack targets and checks current
// legality. SpellCast().Matching checks only at cast detection. Queued
// triggers keep their saved spell and caster. Copies and clones stay separate.
// RON lowers contextual filters in spell targets and battlefield cast events.
// Cost and ability-source filters keep their separate contexts.
//
// # Further Documentation
//
// For specialized guides and comprehensive specifications, refer to:
//
//   - docs/dsl-reference.md: Complete card author reference for the composable DSL
//     (subjects, selectors, actions, triggers, spells, activations, and continuous effects).
//   - docs/engine-rules-contracts.md: Detailed Comprehensive Rules (CR) execution contracts
//     including Continuous Layers 1–7, LKI, state triggers, replacement pipelines, and cleanup.
//   - docs/architecture-4plus1.md: 4+1 architectural view of the rules engine and subsystems.
//   - pkg/mage/dsl: Public card-author export package providing the canonical authoring grammar.
package mage
