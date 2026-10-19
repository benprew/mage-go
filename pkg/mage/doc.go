// Package mage implements the two-player Magic: The Gathering rules engine in Go.
//
// [WholeHandDiscardInstruction] retains one player or player collection.
// Its inspector reads no hands and calls no player references. Execution uses
// one shared discard batch, including replacements, ordering and later events.
// Singular discards retain delayed source-controller reads. Pure spell inspection
// validates complete native operands before legality gates and proofs. A prefix
// with selected players blocks later destruction proofs; a supported suffix
// preserves an earlier proof. Unknown operands remain unknown without actions.
//
// [PlayerDrawInstruction] retains one shared amount for a player collection.
// Its inspector selects no players and reads no amounts. Execution uses
// [DrawEach], individual draw replacements and active player order.
// Pure spell inspection accepts complete native collections and fixed
// nonnegative amounts. All children and bindings validate before proofs.
// A positive prefix with present players blocks later destruction proofs;
// zero amounts and empty sets permit them. Supported suffixes keep earlier
// proofs. Unknown operands remain unknown without draws or state changes.
//
// [NextUntapInstruction] retains a fixed controller-untap restriction for one
// typed permanent or a one-step restriction for a collection. Its inspector
// reads no subjects and installs no rules. Execution uses shared visit capture,
// current controller tracking, step counts, phasing and departure cleanup.
// Pure spell inspection validates all native declarations before legality gates
// and destruction proofs. A nonempty positive-count prefix blocks later proofs;
// a zero count or empty set permits them. A supported suffix preserves a proof.
// Opaque children and foreign bindings make the complete tree unknown.
//
// [GlobalCombatPreventionInstruction] retains all combat prevention until cleanup.
// Its inspector accepts the exact native declaration without installing effects.
// Execution uses [PreventAllCombatDamage] and the shared replacement service.
// Pure spell inspection validates the whole tree before target legality gates.
// A prevention prefix blocks later destruction proofs. A complete supported suffix
// preserves an earlier proof. Unknown children remain unknown without callbacks.
//
// [ObjectColorInstruction] retains one fixed color on a native union target.
// [InspectObjectColorInstruction] reads its declaration without changing state.
// [InspectObjectTarget] selects exact stack identity or a permanent visit with
// current native legality. Missing snapshots and opaque restrictions are unknown.
// RON's two fixed-color branches retain this one union declaration per instance.
// A legal color prefix blocks a later destruction proof; a complete supported
// suffix keeps an earlier proof. Execution uses shared color and layer services.
//
// [TemporaryColorInstruction] retains a collection color replacement until cleanup.
// Its inspector copies the color list without selecting members. An empty list
// makes the members colorless. Execution uses shared layer 5, captured membership,
// exact visits, timestamps and cleanup. Pure spell destruction inspection accepts
// complete native contextual collections, including plural permanent targets.
// A nonempty color prefix blocks later destruction proofs. A complete supported
// suffix preserves an earlier proof. Unknown children and foreign bindings remain
// unknown before target legality gates. No colors or game state change in inspection.
//
// [SavedPlayerInstruction] retains a player snapshot and its ordered body.
// Its inspector reads no player or saved value. Pure spell inspection accepts
// You or ActivePlayer before next-end-step destruction of native collections.
// Saved collection reads must use that exact local binding. Native exclusions,
// attack history and current-turn control history retain complete declarations.
// Other saved scopes and target-dependent collections need separate contracts.
// Scheduling cannot prove immediate destruction or block a later direct proof.
// Execution uses the shared snapshot, delayed frame and destruction services.
//
// [TurnAttackRuleInstruction] retains a live collection attack rule until cleanup.
// Require saves its player at resolution. Its inspector reads no operands.
// Pure spell inspection accepts native permanent filters and You or ActivePlayer.
// Opaque filters, saved players and player targets require other query contracts.
// Attack rules do not prove direct destruction. A prefix blocks later proofs;
// a complete supported suffix preserves an earlier proof. Execution uses the
// shared live membership, combat legality, payment and cleanup services.
//
// [InspectContextualPermanentCollection] retains [BlockedByThisTurn] with a
// complete native singular permanent target. Selection reads exact blocker and
// attacker visits from shared turn history. It uses the supplied legal target
// frame without callbacks, LKI reads or target rechecks. A present empty binding
// selects no objects; a missing or malformed binding is unknown. Phased attackers
// are absent. A phase cycle preserves history; a return creates a new visit.
// Final cleanup clears history. Source and saved references need other native
// inspection contracts.
//
// [CreatureDeathInstruction] retains a simultaneous destruction and its exact
// creature-death result. [CreatureDeathReturnInstruction] retains a required
// creature-card choice per saved visit, with owner control on entry. Their
// inspectors do not read results, choose cards or change state. Pure spell
// inspection validates preceding producers, exact member identities and native
// historical blocker targets before target gates or destruction proofs. A return
// continuation blocks later destruction proofs but preserves earlier proofs.
// Execution uses the shared destruction, history, graveyard visit and entry rules.
//
// [NextCombatThisTurnInstruction.Effect] retains one end-of-combat schedule
// before cleanup. [InspectNextCombatThisTurnInstruction] reads its body without
// a trigger. Pure spell inspection accepts complete direct destruction bodies
// with native singular targets or contextual collections. Every child and
// binding must be known before target gates or proofs. Delayed bodies do not
// prove immediate destruction or block later direct proofs. Inspection does
// not select future members. Execution uses the shared delayed frame, history,
// simultaneous destruction, replacement and cleanup services.
//
// [InspectPermanentTarget] and [InspectSpellTarget] retain native mana-value
// restrictions against the inspected object's announced X. Permanent mana value
// treats printed X as zero. Spell mana value includes each chosen X symbol.
// Selection uses current costs and exact announced identities without reading
// or changing the active resolution frame. Negative announced X is unknown.
// [InspectPermanentTargetDeclaration] reads a singular native target declaration
// without a resolving reference. Unknown filters and descendants still reject.
// RON uses one typed mana-value-X plan for permanent and spell target lowering.
// Spell Blast and Detonate support complete native inspection. Detonate's
// controller recipient uses the same exact target and current legality checks.
//
// [TemporaryMixedLayerInstruction.Effect] retains singular keyword and P/T
// changes through one until-end-of-turn operation with layer 6 and 7c parts.
// Its inspector copies both lists without value reads. Values are saved before
// installation. [NextEndStepInstruction.Effect] uses the shared one-shot delayed
// frame. Its inspector retains the complete body without scheduling a trigger.
// Pure spell inspection accepts complete native conditional destruction bodies
// with exact permanent targets and native filters, including attack history.
// Every child and declaration must be known before legality checks or proofs.
// A delayed body cannot prove immediate destruction or change a later proof.
// Mixed layers remain conservative barriers before later direct destruction.
// Existing visit capture, LKI, phasing, cloning and cleanup services apply.
//
// [PermanentBranchInstruction.Effect] retains a permanent condition and both
// resolution bodies. [InspectPermanentBranchInstruction] reads the declaration
// without callbacks. Pure spell inspection accepts complete native layer-only
// branches with exact native targets and native permanent filters, including
// toughness comparisons and presence or absence of rampage. Both bodies must
// be known. Inspection does not select a branch or predict characteristics.
// A legal conditional prefix blocks later destruction proofs.
// [TemporaryRampageInstruction.Effect] retains separate nonnegative rampage
// grants until end of turn through the shared trigger and layer services.
// [InspectTemporaryRampageInstruction] copies the amount list without grants.
// Exact visits, resolution-time conditions, copies, clones and cleanup use
// the existing services. Mixed layers and delayed bodies need other contracts.
//
// [TemporaryPTValueInstruction.Effect] retains singular P/T value adjustments
// until end of turn. [InspectTemporaryPTValueInstruction] copies the declaration
// without value reads or callbacks. Execution captures all amounts before one
// shared layer 7c installation. Pure spell inspection accepts fixed values,
// announced X, exact native target power, toughness and mana values, and native
// subtraction and multiplication trees. It checks each value target before
// legality gates or earlier proofs. Legal prefixes block later destruction
// proofs; supported suffixes preserve earlier proofs. Mixed parents need
// separate contracts. No future characteristics are predicted.
//
// [TemporaryKeywordInstruction.Effect] retains keyword grants until end of turn.
// Exactly one subject form and a nonempty keyword list are required.
// [InspectTemporaryKeywordInstruction] returns a copied declaration without
// callbacks or layer installation. Execution shares layer 6, exact visit capture,
// fixed collection membership and cleanup with [Until]. Pure spell inspection
// accepts exact native targets and contextual collections. A legal nonempty
// prefix blocks later destruction proofs; supported suffixes keep earlier proofs.
// Mixed layers, protection and other triggered grants need separate query contracts.
//
// [TemporaryPTInstruction.Effect] retains fixed P/T changes until end of turn.
// Exactly one subject form and a nonempty adjustment list are required.
// [InspectTemporaryPTInstruction] reads copied native declarations without
// callbacks or layer installation. Execution shares the existing layer 7c,
// exact visit capture, fixed collection membership and cleanup services.
// Pure spell inspection accepts exact native targets and contextual collections.
// A legal nonempty prefix blocks later destruction proofs, even for zero
// adjustments. Empty or illegal subjects permit later proofs. Supported suffixes
// preserve earlier proofs. Dynamic, mixed and other duration forms need separate
// inspection contracts. P/T changes do not prove direct destruction.
//
// [LibraryProcedureInstruction.Effect] retains a fixed private library look,
// optional top ordering or an optional shuffle, with explicit player operands.
// [InspectLibraryProcedureInstruction] reads these operands without library
// access or choices. Pure spell inspection accepts You, ActivePlayer and exact
// native player targets for both players. Complete trees and target identities
// are checked before proofs. A legal procedure prefix blocks later destruction
// proofs; an illegal operand skips it. Supported suffixes preserve earlier proofs.
// Dynamic look counts and other procedures need separate query contracts.
//
// [GraveyardBattlefieldInstruction.Effect] retains a mandatory normal entry.
// [InspectGraveyardBattlefieldInstruction] reads its card and controller without
// moving cards or running entry replacements. Choices, result bindings and entry
// options require separate contracts. Pure spell inspection accepts native
// singular graveyard targets under You, with exact declaration and visit identity.
// A legal entry prefix blocks later destruction proofs. An illegal prefix does
// not. A supported suffix preserves earlier proofs. Queries do not use LKI,
// run filter callbacks, make choices or change game state.
//
// [LibrarySearchInstruction.Effect] retains a complete single-card search.
// [InspectLibrarySearchInstruction] reads its kind without reading a library.
// Supported kinds move any card to hand or a basic land to the battlefield,
// then shuffle. Execution uses existing search, choice and entry services.
// Pure spell inspection accepts both kinds and reads no hidden card or choice.
// A search prefix blocks later destruction proofs, even for an empty library.
// A supported suffix preserves earlier proofs. Other search procedures remain
// unknown. Native declarations retain no selected card or mutable frame.
//
// [NextMainPhaseInstruction] retains a one-shot next-main-phase body.
// [InspectNextMainPhaseInstruction] reads that body without creating a trigger.
// Spell destruction inspection accepts complete fixed-color mana bodies for You.
// Amounts can use fixed values, saved X and bound spell mana values with LKI,
// with native sums and products. Exact spell declarations must belong to the
// inspected stack object. Unknown bodies or operands reject the complete tree
// before target gates and earlier proofs. Inspection reads no mana value and
// changes no frame, pool or schedule. A schedule prefix blocks later state
// proofs; a supported suffix keeps earlier proofs. Execution uses [AtNext].
//
// [ManaProductionInstruction.Effect] builds native fixed-type or chosen-color mana.
// [InspectManaProductionInstructions] retains complete native trees and operands.
// Spell destruction inspection accepts fixed and announced-X amounts with native
// player targets, You or ActivePlayer. It validates all operands before proofs.
// Production alone cannot directly destroy a permanent. A legal prefix blocks
// later state proofs. A supported suffix keeps earlier proofs. Queries do not
// add mana, run replacements or read amounts. Saved casting-sacrifice values
// use the producer contract below. Native one-color choices and creature/artifact
// spell restrictions retain their declarations without choices or callbacks.
// Fixed restricted batches keep one original event. Other saved amounts,
// restrictions and delayed production need separate query contracts.
//
// [CounterInstruction.Effect] builds a native spell counter action.
// [InspectCounterInstructions] retains complete native trees without callbacks.
// [InspectSpellTarget] reads exact singular stack targets with native card-type
// and color predicates and complete Boolean trees. Selection uses current spell
// types, colors and exact stack identity. Departure and recasting cannot restore
// an old target.
// Missing declarations and zone snapshots remain unknown. No LKI is used.
// Pure spell destruction inspection accepts these counter instructions. A legal
// counter prefix blocks later destruction proofs. An illegal prefix permits
// them when another target is legal. Supported suffixes preserve earlier proofs.
// Countering cannot directly destroy a permanent. The query does not counter
// spells, check counter outcomes, execute triggers or change game state.
//
// [ManaPaymentInstruction.Effect] retains an optional fixed or X mana cost,
// its player and both branches. [InspectManaPaymentInstruction] reads these
// operands without choices or amount reads. Execution uses shared transactions.
// Spell destruction inspection accepts a native singular spell's controller,
// fixed or X amounts, and complete counter branches for that exact target.
// A branch can save that controller with LKI, tap native collections and remove
// the saved player's unspent mana. Other branch children remain unknown.
// All branches validate before legality gates or earlier proofs. A legal payment
// prefix blocks later proofs, including zero costs. An illegal payer target
// skips the payment. Supported suffixes preserve earlier destruction proofs.
// [HasManaAbility] retains a native predicate and uses current ability classes.
// Inspection does not pay costs, counter spells, tap lands or remove mana.
//
// [RandomDiscardInstruction] retains a typed player and amount.
// [InspectRandomDiscardInstructions] reads complete native discard trees.
// Inspection does not select hand cards, use random results or move cards.
// Spell destruction inspection accepts fixed and X amounts with native players.
// A legal discard prefix blocks later proofs. A supported suffix preserves them.
// Execution uses the shared random selection and discard replacement batch.
//
// [Game.InspectSpellDestruction] checks native destruction, return, tap, damage,
// draw, discard, counter, exile, life, mana loss and finite prevention trees on live instant and sorcery spells.
// It checks every announced supported target
// before it reads exact destruction destinations. Permanent targets can combine
// card type, color, noncolor and subtype predicates. All illegal targets
// prevent the whole spell from resolving. Illegal instructions are skipped when
// another target is legal. Singular subjects keep declaration and visit identity;
// collections use the current spell controller. Current layers are required.
// A return can exclude exact watched visits from later destruction. Other
// nonempty groups block later state proofs unless a tap or destruction prefix
// cannot change state. Unknown trees and restrictions return unknown without callbacks,
// choices, shield use or state changes. This bounded
// query does not complete the broader prospective-spell contract for Equinox.
//
// [InspectDamageTarget] retains singular any-target and player-or-planeswalker
// declarations. Opponent-only declarations exclude the current controller as
// a player. They permit planeswalkers under either player. Pure selection uses
// exact declarations, current legality and saved battlefield visits. Copies
// use their own controller. Unknown wrappers and missing visits stay unknown.
// Shared packet, singular damage and finite prevention queries use this policy.
// Native live counts and half-counts use [InspectLivePermanentCount] for packets.
// Other dynamic amounts need separate query contracts.
//
// [InspectPermanentTargets] retains native fixed or exact-X permanent groups.
// [PermanentTargetsPredicate.Select] checks each announced visit with the same
// rules as singular targets. Duplicate identities, wrong counts and missing
// snapshots return unknown. Zero-count groups require the exact declaration in
// the spell definition. No targets differs from all announced targets being illegal.
// The spell query builds a complete legal target frame before it checks actions.
// Contextual collections can use these groups and [FilterPermanentSet] without
// callbacks. Missing frames remain unknown, including before an earlier proof.
// Copies and clones keep declaration identity and use their current controller.
// Optional counts require a separate contract. Native graveyard-count producers
// use [DestructionResultInstruction] and [InspectDestructionResult]. A native
// collection damage suffix can read that exact preceding number binding.
// The complete tree rejects missing, forward or duplicate producers before
// target gates and destruction proofs. Inspection never reads or predicts the
// saved count. Execution retains actual simultaneous graveyard moves after
// replacements. Other result kinds and arithmetic need separate contracts.
//
// [InspectPlayerTarget] retains a singular native player or opponent target,
// including [TargetPlayerAttackedThisTurn].
// [PlayerTargetPredicate.Select] checks the exact declaration and current player
// presence without callbacks or LKI. Missing and repeated declarations return
// unknown; an absent or ineligible player is an illegal target.
// Contextual owner and controller collections can read these target bindings.
// Selection reads current ownership, control, characteristics and visits. Supply
// a resolving target frame; a missing binding cannot supply an empty-set proof.
// The spell destruction query builds this frame after all target checks, before
// it applies the all-illegal gate. Attack-history targets read the current
// player history without callbacks. Opaque wrappers remain unknown.
// Return order and changed-state proof limits still apply.
//
// [CollectionDamageInstruction.Effect] retains one simultaneous damage action.
// [InspectCollectionDamageInstructions] reads complete native collection trees.
// [InspectDamageRecipientCollection] accepts native player sets alone or with
// complete contextual permanent collections. [InspectPlayerCollection] retains
// all players, one player or opponents of You, ActivePlayer or a native player
// target. Missing target frames remain unknown before target gates and proofs. Selection reads current keywords,
// combat membership and exact visits without callbacks or LKI. Layers must be current.
// [HasKeywordFilter] and [NotHasKeywordFilter] retain native keyword predicates.
// Pure spell inspection accepts source damage with fixed or X amounts. Damage
// alone does not destroy a permanent. A nonempty damage prefix blocks later
// destruction proofs, including a zero amount. Empty sets permit later proofs.
// Exact-count permanent target groups use [InspectPermanentTargets].
// Native filters read current properties after target legality is checked.
// Result bindings and division need separate contracts.
//
// [PacketDamageInstruction.Effect] retains one simultaneous native packet group.
// [InspectPacketDamageInstructions] copies complete native groups without callbacks.
// [InspectDamageRecipient] accepts singular any-target and native player targets,
// the resolving controller and the active player. Selection keeps exact target
// declarations and current legality. Missing bindings return unknown.
// [ControllerOfWithLKI] of a complete native singular permanent target retains
// that declaration. Selection reads current control after target legality checks.
// Illegal targets supply no player. Inspection does not read target LKI or simulate
// preceding destruction. Execution keeps the exact visit's current or last
// controller through the shared information service. A legal damage prefix
// blocks later destruction proofs; a supported suffix preserves earlier proofs.
// Spell inspection accepts fixed or X source packets and singular player damage.
// It checks all bindings before the all-illegal gate or any destruction proof.
// Damage prefixes keep the existing changed-state barrier, even at zero damage.
// Attack-history player targets use the shared native player query.
// Packet counts and repeated half-count rounding use [InspectLivePermanentCount].
// Its [LivePermanentCountPredicate.Read] checks current collection membership
// without callbacks or LKI. Bound players need a complete target frame.
// Saved recipients and other dynamic amounts need separate contracts.
//
// [FuturePreventionInstruction.Effect] retains one finite shield installation.
// [InspectFuturePreventionInstructions] reads complete native installation trees
// without callbacks or shield use. Execution uses [PreventNextDamage].
// Pure spell inspection accepts fixed/X amounts with native any-target or
// player targets, the resolving controller or the active player. It validates
// all declarations before target gates or destruction proofs. Prevention alone
// cannot directly destroy a permanent. A legal prevention prefix blocks later
// destruction proofs, including at zero or negative amounts. An illegal target
// is skipped when another target is legal. Supported suffixes keep earlier proofs.
// Saved recipients, dynamic amounts and other shield forms need separate policies.
//
// [PermanentExileInstruction.Effect] retains one singular or simultaneous group
// of battlefield exile actions. [InspectPermanentExileInstructions] reads complete
// native trees without callbacks. Execution uses [Exile] and [ExileEach].
// Pure spell inspection validates native target declarations and contextual
// collections before target gates or proofs. Exile alone cannot directly destroy
// a permanent. A nonempty exile prefix blocks later destruction proofs; an empty
// selection permits them. Supported suffixes preserve earlier proofs. Inspection
// does not move objects, use replacements or predict later state. Saved results,
// linked exile and other zones require separate query contracts.
//
// [DrawInstruction.Effect] retains one singular native draw action.
// [InspectDrawInstructions] reads complete native draw trees without callbacks.
// Execution uses [DrawFor] and the shared per-card draw replacement service.
// Pure spell inspection accepts fixed or X amounts with native player targets,
// the resolving controller or the active player. It validates every binding
// before target gates or proofs. Draw alone cannot directly destroy a permanent.
// A legal draw prefix blocks later destruction proofs, even at zero or negative
// amounts. A supported suffix preserves earlier proofs. Inspection does not
// draw cards, use replacements, read hidden cards or predict a later state.
// RON DrawACard saves a card result only when its exact scoped binding is read.
// Saved results and other dynamic amounts require separate query contracts.
//
// [DamageResultInstruction.Effect] retains singular or simultaneous damage with
// exact total and outcome declarations. [InspectDamageResultInstruction] reads
// the complete producer without callbacks. Pure spell inspection accepts fixed/X
// source damage and native recipients. A later native life action can read that
// total or a singular producer through [ClampedDamageAmount]. Missing, forward,
// duplicate and foreign declarations reject the complete tree before target gates.
// Queries do not read outcomes, predict damage, use shields or run replacements.
// Nonempty damage and legal life prefixes block later destruction proofs.
// Opponent recipient collections use the shared pure player selection contract.
//
// [LifeInstruction.Effect] retains one singular native gain or loss action.
// [InspectLifeInstructions] reads complete native trees without callbacks.
// Result bindings require separate declarations. Execution uses shared life
// replacements, permissions, history and events; damage prevention does not apply.
// Pure spell inspection accepts fixed or X amounts with native player targets,
// the resolving controller or the active player. All operands validate before
// any proof or all-illegal gate. Life alone does not directly destroy a permanent.
// A legal life prefix blocks later destruction proofs, even at zero or negative
// amounts. An illegal subject is skipped when another target is legal.
// [InspectPlayerDamageHistory] retains actual player damage totals from this
// turn, artifact-source totals and nested doubling. [PlayerDamageHistoryPredicate.Read]
// uses the same history as execution without reference callbacks. Native player
// targets, You and ActivePlayer use complete legal frames. Missing frames are
// unknown; absent players supply zero. Life and singular source damage trees
// validate these operands before target gates or destruction proofs, without
// reading totals or predicting damage. Singular permanent recipients reuse
// [InspectPermanentTarget] for exact visit identity and current legality.
// Other dynamic amounts, saved subjects and prevention need separate query contracts.
//
// [InspectTargetCharacteristicValue] retains native singular target power and
// mana values. [TargetCharacteristicValuePredicate.Read] uses the exact visit's
// current information or LKI without reference or filter callbacks. Layers must
// be current. Missing or malformed frames are unknown; empty frames supply zero.
// Pure spell inspection accepts these scalar amounts in native life gain/loss.
// It validates bindings before target gates or proofs without reading values.
// Legal life prefixes remain state barriers. Supported suffixes preserve proofs.
// Source, saved, toughness and arithmetic values need separate query contracts.
//
// [PermanentLifeInstruction.Effect] retains controller life gain from the same
// singular target's power or mana value. [InspectPermanentLifeInstruction] reads
// the native declaration without callbacks. Execution uses exact visit information
// from [ControllerOfWithLKI], [PowerOfWithLKI] and [ManaValueOfWithLKI]. A failed
// destruction does not suppress life gain. Pure spell inspection validates the
// target before any proof or all-illegal gate, without predicting the life value.
// A legal prefix blocks later proofs, even at zero. A suffix preserves earlier
// proofs. Owner, toughness, loss and saved results need separate query contracts.
//
// [ManaLossInstruction.Effect] retains one native unspent mana removal action.
// [InspectManaLossInstructions] reads complete native trees without callbacks.
// Saved mana results use separate declarations. Execution uses [LoseUnspentManaFor].
// Pure spell inspection accepts native player targets, the resolving controller
// and the active player. All bindings validate before target gates or proofs.
// Mana loss cannot directly destroy a permanent. A legal mana loss prefix blocks
// later destruction proofs, even with an empty pool. Supported suffixes preserve
// earlier proofs. Saved subjects need separate query contracts.
// Cloned players keep separate mana production history and payment history.
//
// [SingularDamageInstruction.Effect] retains one native damage instruction.
// [InspectSingularDamageInstructions] reads complete native trees without callbacks.
// Result bindings and instruction-local prevention require separate contracts.
// [InspectDamageTarget] retains singular native damage target declarations.
// [DamageTargetPredicate.Select] keeps player identity separate from permanent visits.
// Permanent legality uses shared current characteristics and source LKI checks.
// Different target domains and chooser wrappers remain unknown.
// The spell destruction query accepts native source damage with fixed or X amounts
// and these targets. Damage does not directly destroy a permanent (CR 120.5).
// A legal damage prefix blocks later destruction proofs because damage can change
// state. Even zero damage keeps this conservative barrier. Illegal recipients do
// not block later proofs if another target is legal. No damage or shield is used.
// These queries require current layers and do not complete Equinox's contract.
//
// [InspectPlayerHandSize] retains HandSizeOf with You, ActivePlayer or a native
// singular player target. [PlayerHandSizePredicate.Read] reads the current hand
// without reference callbacks. Missing frames are unknown; absent players supply
// zero. Pure spell inspection accepts this amount in native singular source
// damage. It validates bindings before target gates without reading hand sizes.
// A legal damage prefix remains a barrier, including an empty hand. Execution
// reads the hand at resolution and uses the shared damage service.
//
// [InspectPermanentTarget] retains a complete type, color and subtype singular target.
// Boolean trees use current types, effective colors and subtypes without filter callbacks.
// Colorless objects match noncolor predicates. Multicolor objects match each
// color they have. Unknown descendants reject the complete declaration.
// Required subtypes and complete subtype domains use the shared matcher.
// Native you/opponent controller and owner domains use current target information
// and the stack object's controller. Native selection uses the same matcher.
// Tap status and current attacking or blocking membership retain complete Boolean trees.
// Source exclusion uses the exact ability source visit, including saved source information.
// A later visit of the same card is a different source.
// Source-relative power and toughness targets use shared current or saved source power.
// Missing exact source information returns unknown. Strict and inclusive limits stay distinct.
// Fixed power equality, toughness equality and inclusive toughness limits retain
// complete Boolean trees. Native filters and pure queries share current P/T reads.
// Counters and continuous effects apply before selection. No target LKI is used.
// The shared collection filters retain these same fixed numeric predicates.
// Attachment target predicates retain complete current host filters. Both objects
// must be present and unphased, with the exact attached host visit. Host control
// restrictions use the stack controller. The host is not a target; its shroud
// does not restrict selection. Reattachment reads the new current host.
// History, combat relations and explicit player references need separate contracts.
// [PermanentTargetPredicate.Select] reads its exact announced visit and current
// legality. It uses shared source information and characteristic target checks.
// Layers must be current. Typed unconditional spell and ability restrictions
// use exact live sources and hosts without filter callbacks. Opaque target rules
// and opaque protection return unknown. Selection does not predict destruction.
// [AnyCardFilter], [HasCardTypeCardFilter], [HasColorCardFilter] and
// [HasSubTypeCardFilter] retain characteristic predicates. [AndCardFilter] and
// [OrCardFilter] retain complete Boolean trees and copy the supplied lists.
// [CanTargetOnlySubtypeFilter] retains a pure target-declaration predicate.
// Complete Boolean trees use the native subtype proof without filter callbacks.
// Source views keep their existing declaration and selected-mode contracts.
// RON retains these trees in static source filters. Conditional target rules
// can use native turn and source attack history conditions with complete Boolean
// trees. Opaque conditions remain unknown and their callbacks do not run.
//
// [InspectPermanentCollection] reads complete type, subtype and color collections,
// including mixed predicates, unions and intersections. Color queries read
// current effective colors without filter callbacks (CR 105.2, 613.1e).
// [PermanentCollectionPredicate.Select] returns
// exact current visits without collection or filter callbacks. Phased objects
// are absent. Layers must be current; this query does not predict destruction.
// RON retains these declarations in direct destruction groups.
//
// [InspectContextualPermanentCollection] retains complete type, subtype and
// color collections with owner or controller restrictions for [You] or
// [ActivePlayer], or complete singular player targets.
// [ContextualPermanentCollectionPredicate.Select] reads current
// visits from an explicit resolving context without callbacks or LKI.
// Layers must be current and the resolving controller must be present.
// Native [OpponentsOf] sets for [You] and [ActivePlayer] also retain complete
// controller collections. Selection reads present opponents without callbacks;
// ownership does not change membership. Unknown descendants reject the whole
// query. This does not predict destruction.
//
// Native [IsAttacking] and [IsBlocking] filters retain current combat queries.
// [Not], [And] and [Or] retain complete mixed characteristic and combat trees.
// Contextual collection inspection reads shared combat membership without
// callbacks. Phasing, departure and combat removal cannot supply old members.
// The context-free collection inspector still rejects combat declarations.
// RON retains IsAttacking, IsntAttacking and IsBlocking in these collections.
//
// [InspectContextualPermanentCollection] also retains [AttachmentsOn] over
// complete supported current host collections, including nested attachments.
// Selection checks the exact host incarnation without callbacks or LKI.
// Both the host and attachment must be present and unphased. Reattachment
// changes membership; a returned host cannot restore an old attachment.
// RON retains AttachedToAPermanent through its existing collection adapter.
// Singular and saved host references need separate inspection contracts.
// This query does not predict ordered actions or destruction certainty.
//
// [InspectPermanentCardTypeFilter] reads complete typed card-type predicates.
// [PermanentCardTypePredicate.Matches] tests a supplied type set without
// callbacks or game state. Opaque Boolean children reject the whole query.
// RON card-type filters retain these plans for pure analysis.
//
// [InspectReplacementActionDomains] reads complete proposal categories for
// supported native damage, damage life loss, destruction, tap, assignment, draw
// and life gain rules without callbacks.
// Pure destruction inspection ignores rules with proven unrelated domains in
// either list. Native face-up rules have damage, tap and assignment domains.
// Native player damage caps match damage proposals. Native life floors match
// only the life loss that results from damage. Neither matches destruction.
// Native protection prevention and temporary permanent or player redirection
// match only damage proposals. Their operands do not run during inspection.
// Native scoped and future draw replacements match only individual draws.
// Native life-gain-to-draw rules match only positive life gain proposals.
// Their bodies and player operands do not run during inspection.
// Unknown and nil rules remain unknown. Domain membership does
// not establish activation, applicability, destination or sequential outcomes.
//
// [Game.InspectDestructionDestinations] reads all possible destinations for one
// exact selected visit. It supports native regeneration, cancellation, saved
// death exile and current death-return declarations. Competing rules retain
// every possible destination without choices or callbacks. Live history rules
// read exact damage visits and retained card type filters. Opaque rules,
// opaque filters and custom activation conditions return unknown. Creature status
// at damage controls native history regeneration restrictions. Tokens report
// their proposed move before they cease to exist. This is not group-state analysis.
//
// [TapInstruction.Effect] builds a native tap, untap or optional singular choice.
// Collections keep one simultaneous group. [InspectTapInstructions] retains
// complete native trees without effect, operand or decision callbacks.
// Result bindings and unknown children reject the complete tree.
// Typed tap and untap instructions refresh layers after the full
// transition group and its events, before the next instruction. Stun counter
// removal also refreshes layers. Legacy actions and costs keep their contracts.
// The spell destruction query accepts these instructions with native permanent
// targets or pure contextual collections. They cannot prove direct destruction.
// [Game.InspectTapStateChange] checks exact selected visits without callbacks.
// Tap on tapped visits and untap on untapped visits permit later proofs because
// they cannot change state. Other nonempty tap instructions return unknown
// before later destruction. This includes optional choices and tapped visits
// with stun counters. Empty and illegal subjects retain the resolution gate.
// Untap on an untapped permanent does not remove stun counters. No later state
// is projected.
//
// [DirectReturnInstruction.Effect] builds a native singular or simultaneous
// return action. [InspectDirectReturn] retains complete return trees without
// operand callbacks. RON lowers both owner-hand return forms to this declaration.
// Pure spell inspection excludes returned exact visits from later destruction.
// If watched visits remain after a nonempty return, later state is unknown.
// Return-only tails cannot directly destroy a watched object. Selection reads
// current layers, legality and exact visits before any state proof.
//
// [DirectDestructionInstruction.Effect] builds the native destruction action.
// RON uses this descriptor for singular and simultaneous direct destruction.
// [Game.InspectDestructionProposal] checks one exact, already selected visit.
// It reports whether the proposed destruction continues, with a separate known
// result. Typed regeneration and cancellation do not consume their uses.
// Opaque replacements, local death-return rules and live regeneration filters
// return unknown. Inspection calls no replacement or filter callbacks.
// Layers must be current. This is not a sequential or zone-move prediction.
// Equinox still needs those separate contracts before its query can execute.
//
// [Game.InspectDestructionStateChange] checks exact selected visits without
// callbacks or shield use. Empty groups and current indestructible visits
// cannot propose destruction. They permit later ordered and spell proofs.
// All other visits retain a possible change, including canceled or regenerated
// destruction. Invalid, phased and stale visits return unknown. Layers and
// subject legality must be current. No later state is projected.
//
// [InspectOrderedDestruction] retains complete direct collection destruction
// sequences. [OrderedDestructionPredicate.Certain] checks exact watched visits
// against current simultaneous groups without callbacks or state changes.
// Empty groups and groups of indestructible visits permit later analysis.
// Other groups block it unless they already prove a watched graveyard move.
// Regeneration can change state without moving an object. Unknown remains
// distinct from a complete query with no proof. Singular references, source
// resolution and target legality need separate contracts.
//
// [InspectDirectDestruction] reads complete destruction instruction trees.
// It keeps typed subjects, simultaneous sets and regeneration restrictions.
// Inspection calls no effects or operand callbacks. Unknown trees reject.
// The declarations do not predict legality, replacements or destruction.
// RON lowers the four direct destruction actions to these shared declarations.
//
// [TurnResolvedPermanentFaceUpBeforeAction] installs an exact saved-spell rule.
// Before assignment, positive damage or tapping, its permanent turns face up.
// The event continues through shared replacement rechecks and batch services.
// Source departure cannot end it. Phasing suspends it; turning up ends it.
// Copies, returned objects, payment trials and game clones keep separate state.
// RON accepts the complete saved-cast CreateReplaceWouldMaskUntil shape.
//
// [CastHandCardFaceDownWithoutPayingInto] and
// [MayCastHandCardFaceDownWithoutPayingInto] save the exact successful spell.
// Failed and declined casts clear the result. [PermanentFromResolvedSpell]
// reads only the permanent that spell becomes. Copies and later visits cannot
// supply it. Control changes and face-up status keep it; phasing stops actions.
// Information readers use exact phase and departure LKI. Shared frames keep
// delayed bodies, copies and clones separate. RON binds
// ThePermanentSpellResolvedThisWay only after its local cast producer.
// [TurnResolvedPermanentFaceUpBeforeAction] supplies its face-up replacement.
//
// [MayCastHandCardFaceDownWithoutPaying] offers a separate cast choice for a
// saved hand visit. It checks legality and costs before the offer and again
// after acceptance. Declining does not spend mana, move or reveal the card.
// RON accepts MayAction around the chosen-hand PT(2, 2) cast action in ordinary
// resolving bodies. The choice does not cancel independent later instructions.
//
// [CastHandCardFaceDownWithoutPaying] casts a saved hand visit as a nameless,
// colorless 2/2 creature spell with no mana cost or printed abilities.
// Casting restrictions and external costs read these face-down characteristics.
// Shared payment, stack, entry and zone services retain the physical identity.
// Printed entry rules cannot apply. Face-down status precedes entry events.
// [StackObject.IsFaceDown] exposes status. [Game.InspectFaceDownObject] gives
// the controller a private copy, including while the permanent is phased out.
// Public reveal history records departure and exposes remaining cards at game end.
// Spell copies keep visible values without private identity or face-down status.
// Permanent copies retain status and physical departure identity. Clones isolate
// movement and reveal records. Continuous grants and summoning sickness apply.
// RON accepts the chosen-hand cast action with PT(2, 2) in resolving bodies.
// Face-up replacements use [TurnResolvedPermanentFaceUpBeforeAction].
//
// [WithActivationManaReceipt] saves actual mana types for one activation.
// [WithActivationManaXReceipt] saves the X part of one pure X payment.
// Fixed requirements receive mana first. Taxes do not enter the X receipt.
// [ChooseHandCardWithActivationManaOrElse] checks a private hand choice against
// that receipt. Hand X is zero; absent costs reject. Current spending rules
// cannot change the saved types. Copies and clones retain separate results.
// RON accepts ManaCostIsSubsetOfManaPaidForThisAbility in immediate optional
// hand choices of nonmodal nonmana activations with one pure announced X cost.
// Face-down casting uses [CastHandCardFaceDownWithoutPaying].
// [TurnResolvedPermanentFaceUpBeforeAction] replaces assignment, damage and taps.
//
// [ChooseHandCardOrElse] offers a private choice from one player's hand.
// A [CardBinding] saves the exact hand visit for the chosen branch only.
// Selection does not move or reveal the card. Empty candidate sets and declined,
// invalid or stale answers use the other branch. Current filters are checked
// again after the choice. Candidate copies cannot change the live cards.
// Nested calls restore outer bindings. Shared stack frames keep copies and
// clones separate. [FromCard] cannot follow a returned card or read hand LKI.
// RON accepts MayCost and PlayerMayCost with ChooseACardInHand in ordinary
// resolving bodies. If, Unless and IfElse keep the result on paid paths.
// Chosen hand visits can use [CastHandCardFaceDownWithoutPaying].
//
// [RandomizeBlockingPilesThisTurn] replaces blocker choices through cleanup.
// Defenders choose optional piles before the shared RNG assigns attackers.
// Extra blocking capacity permits extra piles. Empty piles are valid.
// The shared combat planner checks assigned pairs without adding requirements.
// Exact visits, group events, later combats and clones use the shared services.
// RON accepts this game operation with UntilEndOfTurn in resolving bodies.
// All five complete Camouflage records have casting and gameplay tests.
//
// [MaySkipYourTurnToUntap] replaces the current controller's turn by choice.
// Use [Static] with a state condition and This or AttachedPermanent.
// [Game.TryBeginTurn] applies turn replacements before a turn's actions.
// The affected player selects the replacement order, including scheduled skips.
// A paid skip saves an exact visit for untapping in the next actual turn.
// Skipped turns and steps retain it. Source changes cannot cancel it.
// The shared untap service checks stun counters and sends untap events.
// Returned and phased-out subjects cannot untap. Clones keep separate results.
// Native runners, application loops and gametest use this turn-start operation.
// RON accepts the conditional source turn-skip and paid-untap shape.
// All five complete Time Vault records have casting and gameplay tests.
//
// [RetainBlockerEligibility] saves eligible blocker visits at resolution.
// [SavedBlockers] saves set membership; [CurrentBlockers] checks current properties.
// [AllRetainedBlockers] and [AnyRetainedBlockers] combine those conditions.
// [Until] and the shared combat service enforce the rule and its duration.
// Source departure and ability removal do not end it. Phasing keeps saved visits.
// Returned objects cannot inherit membership. Clones keep separate effects.
// RON accepts the selected-pile blocker rule for a resolving loop member until
// end of combat. Complete Raging River records have casting and gameplay tests.
//
// [ChooseNamedPermanentPile] saves the exact visits in one required pile choice.
// Empty piles are valid choices. [FromPermanentSet] reads present members.
// RON accepts named pile choices after a partition in resolving permanent loops.
// Only the same loop member can read its selected pile in later instructions.
// Shared frames keep choices separate through iterations, delays and clones.
// Retained blocking rules use [RetainBlockerEligibility].
//
// [SeparatePermanentsIntoNamedPiles] saves exact battlefield visits in named
// [PermanentSetBinding] results. Each visit goes into one pile. Piles can be
// empty, and no member changes zones. [FromPermanentSet] reads present visits.
// Shared resolution frames keep results separate through copies and delays.
// RON accepts the partition action for saved attack-group defenders. Only its
// collection filter can read EachablePlayer. Selected piles have scoped readers.
//
// RON spell-targeted activation grants use [GrantActivatedAbility],
// [GrantEachActivatedAbility] and [ActivatedAbilityModification] with [WithTargets].
// Each target and body share the recipient declaration scope. Announcement,
// payment, resolution legality, copies and clones use the shared engine services.
// Source changes do not change the saved ability controller. RON accepts one
// TargetSpell declaration with fixed costs. Prospective destruction needs a
// separate contract; the complete Equinox record remains unsupported.
//
// [MayAnteTopEach] collects optional choices in APNAP order before one ante batch.
// [PlayerSetBinding] saves membership in the shared resolution frames.
// [FromPlayerSet] reads that result without another offer. Ineligible library
// tops cannot pay. Source departure and phasing do not cancel pending bodies.
// [SetLifeTotalEach] uses shared life gain and loss replacements and permissions.
// All ordinary life changes finish before their events. Copies and clones keep
// separate results. RON accepts the fixed-total PaidCost continuation in an
// ordinary resolving sequence after EachPlayerMayCost with library-top ante.
//
// [SpendAnyManaForOneSpellThisTurn] saves one optional player permission.
// The printed mana cost can use any mana type. Added costs keep their normal
// requirements. Shared cost locking applies reductions before this permission.
// Each mana unit keeps its actual type and spending restriction. Acceptance
// does not spend the permission until total payment succeeds. Cleanup ends it.
// Source departure, phasing and ability removal do not end a saved permission.
// Clones keep separate permissions. [Game.CanPaySpellManaCost] makes no choices
// and checks the printed cost with X. [Game.GetCastableSpells] uses that query.
// RON accepts MayActionOnce with AnySpell and UntilEndOfTurn in ordinary
// resolving player effects. North Star has complete casting and gameplay tests.
//
// [ReturnToHandInsteadOfDyingWithPlayLock] replaces the source's death.
// Shared replacement choices run before moves. Sacrifice remains a paid sacrifice.
// The completed hand visit stays revealed and cannot be played until its owner's
// next actual turn. A skipped turn cannot end the lock. Return and clones keep
// separate visits. [Game.IsHandCardRevealed] exposes only the current public card.
// RON supports the complete Firestorm Phoenix replacement and continuation.
//
// [ChooseSpellCastThisTurn] saves an exact cast from this turn at resolution.
// Cast facts survive departure. Uncast copies cannot supply a choice.
// [DamageDealtThisTurnBySource] reads actual damage from that spell identity.
// Shared prevention and redirection apply before totals change. Separate casts
// and copies keep separate totals. Cleanup clears history; clones copy it.
// RON accepts a sorcery type and explicit caster in ordinary resolving lists.
// The corrected complete Backdraft record has casting and gameplay tests.
//
// [MayDrawCardsThen] runs a body after the player accepts the draw instruction.
// Replaced draws keep the accepted continuation. Successful draw events save
// exact hand visits for this turn. Cleanup clears them; clones copy them.
// [PayLifeOrReturnDrawnCards] selects all cards before per-card life payments.
// Earlier draws count. Added cards and returned visits cannot supply a choice.
// Shared payment and zone replacements apply. Library returns are not discards.
// RON accepts fixed nonnegative counts and life in ordinary resolving bodies.
// The corrected complete Sylvan Library record has cast and gameplay evidence.
//
// [StaticAbilityPayment] offers a static ability grant during resolution.
// [PayActionOrElse] checks the saved visit again after the choice.
// [AcquireStaticAbilityUntil] uses the existing layer 6 grant service.
// A later ability removal can remove the granted attack restriction.
// [AttacksWithoutTappingUntil] saves the affected player and source visit.
// Its filter and condition read current state at attack declaration.
// The rule can affect later creatures and survives ability removal.
// Both attack paths use this rule without granting vigilance.
// Phasing preserves the visit. A returned source cannot supply its condition.
// RON supports Johan's complete optional grant and conditional attack tapping
// until end of combat. Complete casting and gameplay have test evidence.
//
// [SnapshotPermanentSet] saves exact battlefield visits for later instructions.
// [FromPermanentSet] selects present, unphased survivors of that result.
// New members and returned objects cannot replace the saved visits. Phasing
// preserves identity. Copies, clones and delayed bodies keep separate frames.
// [ChangeEachController] uses the shared duration and control layers for a set.
// All members are saved before control changes. Indefinite control survives
// source departure and ability removal. Later control effects use timestamps.
// RON accepts CreatePermanentsList(collection) and the lexical symbol
// ThePermanentsList in ordinary resolving action lists. GainControlOfEachPermanent
// uses the explicit action player. Complete Rohgahh gameplay covers payment,
// the untargeted tap and control continuation, and the current-controller bonus.
//
// [ExileGraveyardAsEntersWithCounterChoice] requires one entry exile payment.
// The entry controller selects exact graveyard visits before any move.
// Failed selection prevents entry. Replacements can change or omit paid moves.
// Only completed exile moves supply counters. Each card supplies one choice.
// Shared counter replacements apply before entry observations. Noncast entry
// has X equal to zero. Return and copies require separate payments and choices.
// RON accepts one MustCost graveyard exile with a CostWasPaid IfElse entry
// continuation, matching exile count, counter choices and failed graveyard entry.
// The complete Frankenstein's Monster record has casting and gameplay tests.
// Failed permanent spell entry uses shared zone replacements and zone events.
//
// [IfSacrificeWithManaValue] requires one matching sacrifice at resolution.
// Shared payment saves its mana value before departure. Replacement moves do
// not change the payment result. Only the paid body can read the saved value.
// [SearchLibraryWithManaDifference] searches for one matching library card.
// The player can find none. A positive mana value difference uses shared mana
// payment. An unpaid find moves to its owner's graveyard through replacements.
// Entry uses shared entry replacements. Exact library visits survive choices;
// returned visits cannot replace them. Movement precedes one shuffle.
// RON supports this complete artifact search and paid sacrifice continuation
// in ordinary resolving bodies. Transmute Artifact has paid-cast gameplay tests.
//
// [WithActivationManaXFromSpell] fixes X from the sole spell target before payment.
// Queries inspect legal spells without choices or changes to ambient X.
// Stack mana value includes the spell's X. Shared costs supply rollback.
// Copies keep payment X. Source departure and phasing cannot change it.
// [IfSpellTargetsPlayer] checks current targets without departure information.
// [ChangeSingleSpellTargetToPlayer] changes one target to another legal player.
// The original spell supplies target restrictions and keeps its controller,
// mode, X and allocation. Copies and clones keep separate targets.
// RON accepts target-derived twice-mana-value X and these resolving activation
// adapters. Complete Reflecting Mirror casting and gameplay have test evidence.
//
// [PreventNewAuraAttachmentsIf] keeps existing Auras on current subjects.
// A former attachment cannot attach again while the condition holds.
// [PreventControlGainsIf] keeps established control and blocks new control gains.
// An ended control effect can restore default or earlier control.
// Both rules read current subjects and source state. Phasing preserves visits.
// Ability removal disables the rules. Clones keep separate effect state.
// Control reconciliation applies later layers with the permitted controllers.
// RON accepts these rules in conditional EachPermanentRuleEffect bodies.
// Complete Guardian Beast has paid-cast and battlefield gameplay evidence.
//
// [PhaseOutUntilSourceLeaves] saves the source and subject battlefield visits.
// Source departure returns the subject and its indirect attachment group.
// The untap turn action cannot end the wait. An external phase-in cancels it.
// A later phase cycle cannot inherit it. Clones keep separate waits and epochs.
// Source phasing and ability removal do not end the wait. A source that left
// before resolution cannot start it. Simultaneous source moves finish first.
// The optional tap changes only the subject's phase-in status, with no tap event.
// RON supports PhaseOutPermanentUntilWithEffects with a typed permanent,
// UntilPermanentLeavesBattlefield(ThisPermanent) and TapAsPhasesIn in resolving
// battlefield-source bodies, including entry triggers. Oubliette has complete
// RON casting and gameplay evidence.
//
// [ThisCardDiscardedByOpponentEffect] saves the discard cause controller.
// Discard proposals retain the hand visit and cause facts before replacements.
// Resolving spells and abilities supply their saved controller and stack identity.
// Public graveyard and exile visits can supply the discarded card's own ability.
// Hidden library destinations cannot supply it. Costs and cleanup do not match.
// All moves in a discard batch finish before events. Pending triggers, copies
// and clones keep separate cause bindings after source departure or return.
// RON accepts FromAnyZone around the self-card opponent-effect discard TriggerA.
// Its action list can read Trigger_ControllerOfThatSpellOrAbility as a player.
// The complete Psychic Purge record has paid-cast and discard gameplay tests.
//
// [ProhibitDamagePreventionAndRedirection] changes damage rules for one visit.
// Install it with [Until] to retain the recipient until the stated duration ends.
// Ability removal cannot remove this rule. The shared damage service checks
// the current recipient before prevention or redirection. Prohibited damage
// does not use shield capacity. Other damage replacements still apply.
// Phasing keeps an installed rule. Return cannot inherit it. Clones keep
// separate effects and shields. RON accepts the bare operation
// DamageDealtToItCantBePreventedOrRedirected in ordinary resolving
// CreatePermanentRuleEffectUntil lists with UntilEndOfTurn.
// Whippoorwill uses a saved graveyard visit for its delayed exile.
//
// [CouldntAttackThisTurn] reads saved attack opportunities for one exact visit.
// Both declaration paths record before costs, taps and attack events.
// Shared restrictions, optional costs, limits and maximum requirements apply.
// Extra combats add history. Skipped combats add none. Later haste cannot
// create an earlier opportunity. Control and phasing keep the saved visit.
// Return cannot inherit it. Clones keep separate maps. Final cleanup clears it.
// RON accepts the bare symbol in ordinary resolving permanent collections.
// Complete Season of the Witch gameplay covers both triggers and paid casting.
//
// [DeclareGraveyardCardTargets] selects cards in an earlier player's graveyard.
// Zero cards is valid. Shared targets retain exact visits and copy counts.
// [ReturnGraveyardCardsToLibraryTop] saves visits before replacements.
// The selected chooser orders library moves. All moves finish before events.
// Returned visits cannot replace saved visits. Owner zones still apply.
// RON accepts this player and card group pair in ordinary spell bodies.
// Complete Drafna's Restoration tests cover paid casting and ordered return.
//
// [TrySwapBlockingAssignments] checks two present blocking creature visits.
// Each must be able to block the other creature's current attackers.
// Tapping, evasion, protection, block restrictions, capacity and bands apply.
// Block costs and declaration requirements do not apply. Failed checks keep
// both assignments. Success removes both blockers and adds the complete batch.
// Attackers stay blocked. New block and pair events see all new assignments.
// Empty assignments remain valid after all their attackers leave combat.
// RON accepts a fixed pair of opponent targets with ControlledByTheSamePlayer
// and TrySwappingBlockAssignmentsOfTwoBlockingCreatures in resolving bodies.
// [OnePlayer] supplies a typed singleton to existing damage batches.
// Complete Sorrow's Path gameplay covers its swap and independent tap trigger.
//
// [MayHavePermanentBlockAttacker] offers a block during resolution.
// The typed player chooses one present attacker for the exact blocker visit.
// Declaration restrictions, costs and limits do not apply to this effect.
// Shared combat services retain bands, block history and new block events.
// The action does not repeat the blocker step or change declaration snapshots.
// Absent, returned or phased visits cannot supply a block. Copies and clones
// keep separate choices. Waiting queries retain explicit source references.
// RON accepts MayAction(ChooseAnAttackingCreatureForBlockerToBlock(filter, ref))
// in ordinary resolving bodies. Complete False Orders gameplay covers all
// five records, casting, historical unblocking and the optional new block.
//
// [DamageReceivedThisTurnFromOtherSources] sums actual damage to one exact visit.
// It excludes damage from that visit. [DamageSourceHasName] reads source names.
// Historical source properties come from damage time under CR 608.2i.
// Shared damage services record final positive amounts before damage events.
// Phasing and marked damage removal keep history. Final cleanup clears it.
// Copies, clones and delayed queries retain exact references and separate state.
// RON accepts the named Other-source filter in ordinary resolving bodies.
// DeadPermanentDealsDamage requires a source death event and uses shared LKI.
// The complete Blazing Effigy record has paid-cast and chain-damage evidence.
//
// [TokenLegendary] gives a token declaration the legendary supertype.
// Direct creation and definition factories retain this copiable value.
// The legend rule ignores phased-out permanents. Players choose in turn order
// before one batch moves the other legends to their owners' graveyards.
// These moves are not destruction or sacrifice. RON accepts one Legendary
// symbol in TokenWithPT and NamedTokenWithPT supertype lists.
// Stangg has complete RON gameplay evidence with saved token sets and delays.
//
// [PayChosenExileOrElse] offers a subset of a typed permanent set at resolution.
// Zero is a valid payment. All selected visits are checked before any move.
// Shared exile services apply replacements in turn order and move one batch.
// Replaced actions still pay the cost. Only actual exile moves supply the count.
// Tokens count even when no card remains in exile. The set can include
// permanents controlled by other players.
// The count belongs to the paid branch. Nested payments restore prior results.
// Copies, clones and delayed bodies retain separate state and source visits.
// RON accepts MayCost(Exile([AnyNumberOfPermanents(TheCreatedTokens)],[]))
// with an adjacent If or Unless payment continuation in permanent resolving
// bodies. NumPermanentsExiledThisWay is bound only in that paid continuation.
// Entry and draw replacement bodies reject. Complete Tetravus gameplay covers
// both upkeep orders, casting, entry counters, token creation and token exile.

// [PlayerOrControllerOfDamageTarget] reads a player or exact visit's controller.
// Departed visits supply last known information. Return cannot replace a visit.
// [CopySpellFor] copies a present spell under a typed player's control.
// [MayCopySpellFor] adds a separate copy choice. Both permit new legal targets.
// [ThisSpell] can supply the resolving spell, including a resolving copy.
// Shared stack services retain choices and identity. Copies are not cast.
// RON accepts these self-copy forms in ordinary spell bodies and PlayerAction.
// The complete Chain Lightning record has paid-cast and copy-chain evidence.
//
// [SelectLibraryTopAndBottomRestRandomly] supplies private library selection.
// One inspected card stays on top. The rest go to the bottom in a random order.
// The count is read once. No card is drawn or moved to another zone.
// Shared inspection, player choices and library services retain private facts.
// [WithActivationMinimumX] checks announced X before costs are paid.
// RON uses both services for the complete Aladdin's Lamp draw replacement.
// Existing replacement rules retain saved X, source visits and separate uses.
//
// [PutHandCardInHandFor] moves one saved hand visit to its owner's hand.
// [PutSourceCardInGraveyardFromAnywhere] finds the physical source across zones.
// Both validate a typed destination player and apply the owner-zone rule.
// Nonbattlefield moves use [CardZoneMoveAction] through shared replacements.
// Rechecks exclude returned visits. Battlefield moves retain shared departure
// rules, LKI and death events. Phased sources cannot move. Events follow placement.
// RON accepts explicit player operands in ordinary resolving bodies.
// The complete Tempest Efreet record has paid-cast and activation evidence.
// Its missing graveyard player is corrected from local Oracle text.
//
// RON accepts one SetLandType(basicType) in CreatePermanentLayerEffectUntil
// with UntilPermanentLeavesBattlefield(ThisPermanent) in resolving permanent
// bodies. [UntilSourceLeaves] and [SetSubtypeFamily] keep the source and target
// visits. Source phasing preserves the duration. Source departure ends it.
// Shared land layers replace printed abilities and supply intrinsic mana.
// Copies, clones, delayed queries and overlapping effects keep separate state.
// Gaea's Liege has complete RON gameplay evidence across all six records.
// [DefendingPlayerOf] reads a present exact attacker during two-player combat.
// Live static counts use that player through shared collection services.
// Declaration uses nonattacking P/T before the attacking branch applies.
//
// RON accepts CombatDamageWouldBeDealtByACreature(SinglePermanent(ref))
// with one PreventThatDamage and UntilEndOfTurn in resolving bodies.
// The saved visit uses [PreventCombatDamageByEachThisTurn]. A permanent loop
// can offer a separate controller payment before it installs each rule.
// Heroism selects red attackers and offers payments when its ability resolves.
// Both paid casting and complete activation have RON gameplay evidence.
//
// [TargetOpponentOwns] checks current ownership at selection and resolution.
// Control does not change the owner test. Shared target services retain exact
// visits, phasing, protection, copied controllers and separate factory bindings.
// RON accepts OwnedByAPlayer(Opponent) in permanent targets, including And.
// Bronze Tablet uses these checks in its complete RON gameplay tests.
//
// RON source-death bodies can read OwnerOfDeadPermanent as a typed player.
// PermanentDies with SourceOnly and WithOwner saves this fact at detection.
// Pending triggers, copies and clones keep that owner through later returns.
// Repeated owner declarations fail definition validation.
// All six complete Personal Incarnation records have paid-cast, activation
// and death evidence. Life loss reads the owner's life at resolution and
// rounds up. Nonpositive totals cause no loss.
//
// [RedirectNextDamageToPlayer] saves one exact subject, player and finite budget.
// Damage can split between the subject and player until exhaustion or cleanup.
// Shared replacements keep source facts, combat status and prior rule uses.
// The subject's controller allocates simultaneous packets within legal bounds.
// Both portions stay in one damage instruction and one simultaneous event.
// Damage results include both portions. Separate added damage stays separate.
// Phasing suspends redirection. Return cannot inherit the old visit's budget.
// Waiting and delayed queries retain exact subjects without installing rules.
// RON accepts a positive fixed next amount, Permanent(ref), and one player redirect.
// This supplies Personal Incarnation's activation and its complete-card tests.
//
// [OwnerOfWithLKI] reads one exact permanent visit's owner.
// Present visits supply current ownership. Departure and phase information
// supply the saved owner. Return cannot replace the saved visit.
// Waiting delayed queries retain source reads without resolving the player.
// RON accepts OwnerOfPermanent(ref) through typed player references.
// Existing delayed-death owner bindings and activation permissions remain.
// Bronze Tablet reads the saved owner after target exile.
//
// [PreventDamageByThisTurn] saves one exact permanent visit until cleanup.
// Both combat and noncombat damage use shared prevention services.
// Departure keeps the saved source. Return creates a different visit.
// Control changes and phase cycles keep the same identity.
// Waiting queries retain exact references without installing rules.
// Delayed bodies can install prevention for their saved absent source.
// Clones keep separate rules and cleanup. Unpreventable damage keeps the rule.
// RON accepts DamageWouldBeDealtByPermanent with one PreventThatDamage
// operation and UntilEndOfTurn in ordinary resolving bodies.
// The complete Kry Shield record has cast, payment and resolution evidence.
//
// [ForcedManaLossInstruction.Effect] retains forced activations and mana loss.
// [InspectForcedManaLossInstruction] reads their native operands without choices.
// [LostManaInstruction] retains the exact saved loss or transfer declaration.
// [InspectLostManaInstruction] reads no saved mana and changes no pool.
// Spell destruction inspection validates native players, complete native filters
// and exact preceding lost-mana producers before target gates or earlier proofs.
// Legal forced activations and transfers block later state proofs. Supported
// suffixes preserve earlier proofs. Queries do not inspect ability outcomes,
// run payments, invoke replacements or change bindings, mana, history or RNG.
// Execution uses the existing player snapshot, activation and transfer services.
//
// [ActivateManaAbilityOfEach] saves matching visits controlled by one player.
// That player chooses the order, ability and output. Each visit can supply
// one activation. Shared activation services check costs and resolve mana
// abilities without the stack. Failed payments make no partial payment.
// Returned visits cannot replace saved visits. Phased sources are absent.
// Each activation has its own X. Copies and clones keep separate choices.
// RON accepts the forced land activation and mana loss in PlayerAction.
// The filter requires Land and an ActionPlayer controller in one And.
// A saved player supplies both activation and loss. The lost mana result
// uses [LostManaBinding]. Entry and draw replacement bodies reject.
// All six complete Drain Power records have cast and resolution evidence.
//
// [RedirectNextDamageToPermanentFromSelectedSource] saves one exact source,
// player and destination visit until use or cleanup. The destination must be
// a present creature or planeswalker at damage time. Later target restrictions
// do not apply. Phasing suspends the rule; return cannot restore the old visit.
// Selected permanent spells continue to their permanent. Shared replacements
// redirect all matching simultaneous packets, then consume the rule once.
// Source information, combat status, lifelink and damage events remain intact.
// RON accepts the chosen-source next-damage event for Player(playerRef) with
// one DealToCreatureOrPlaneswalkerInstead(permanentRef) in resolving bodies.
// Entry and draw replacement bodies reject. The corrected complete Nova Pentacle
// record has paid-cast, opponent-target and resolution-source choice evidence.
//
// RON accepts ReplaceWouldDiscard with AnEffectWouldCausePlayerToDiscardACard(You)
// and one MayAction(DiscardItToTopOfLibraryInstead) on battlefield permanents.
// [DiscardToLibraryReplacement] supplies the live source and current controller.
// Costs do not match. Each selected card has a separate private decision.
// Shared discard batches ask for library order before any card moves. Choices
// put the first choice on top and use a valid fallback. Discard events follow
// all moves.
// The destination prompt identifies the selected hand card without a reveal.
// All six complete Library of Leng records have cast and discard evidence.
//
// [ExchangeLifeTotals] saves both typed players and their current life totals.
// A prohibited gain or loss stops the complete exchange. Life replacements
// run in turn order before either ordinary life change. Events follow both
// changes and carry actual amounts. Equal totals and absent players do nothing.
// [Game.CanGainLife] and [Game.CanLoseLife] read current layer-cycle permission.
// [GameRules.AddCantGainLife] and [GameRules.AddCantLoseLife] set that permission.
// Life payments, direct life changes and damage life loss use these checks.
// Direct life loss uses [LifeLossAction] through shared replacement services.
// Damage retains its separate damage and life-result replacement contracts.
// Source information queries make no choices. Pending and delayed bodies keep
// exact source visits and their resolving player. Copies and clones stay separate.
// RON accepts ExchangeLifeTotalWithPlayer(playerRef) in ordinary resolving bodies,
// including single-player actions and delayed bodies. Entry and draw replacements
// reject. The complete Mirror Universe record has cast and activation evidence.
//
// [AnimateArtifactsUntilCleanupAfterExit] animates current noncreature artifacts.
// Each member loses all abilities and has base P/T equal to its mana value.
// Shared layer services keep each selection through layers 4, 6 and 7b.
// Type dependencies and later ability grants keep their normal order.
// Source departure retains the same effect and timestamp until cleanup.
// Membership stays live, including artifacts that enter after departure.
// Phasing suspends a present source without starting a departure duration.
// Return creates a separate source visit. Clones keep separate lifetimes.
// RON accepts EachPermanentStickyLayerEffect for this exact operation set,
// a conjunctive noncreature artifact filter and UntilEndOfTurn.
// Other sticky filters, operations, durations and resolving contexts reject.
// Both complete Titania's Song records have casting and gameplay evidence.
//
// [LimitDamageLifeLoss] limits a typed player's life loss from damage.
// Full damage remains available to events, history, outcomes and lifelink.
// A live static rule uses current control, exact source identity and phasing.
// Shared damage batches save life results before events are sent. Actual gains
// from lifelink and prevention enter the loss replacement's life-total test.
// Direct life loss and legacy minimum-life callers keep their current behavior.
// RON accepts PlayerEffect(You, one fixed positive damage life-floor operation).
// The complete Ali from Cairo record has paid-cast and damage gameplay evidence.
//
// [LimitPlayerDamageFromTypes] supplies one live static damage replacement.
// The current typed player and exact source types select each damage event.
// Simultaneous packets from one source to that player share the threshold
// and replacement amount. Separate visits and recipients stay separate.
// Shared damage services apply replacement order, rechecks, events and lifelink.
// Source departure, phasing and ability removal suspend the static rule.
// RON accepts a fixed positive GreaterThanOrEqualTo threshold, one IsCardtype
// or an Or of distinct types, Player(You) and one fixed DealDamageInstead.
// The replacement amount must be nonnegative and less than the threshold.
// The complete Forethought Amulet record has cast, upkeep and damage evidence.
//
// RON accepts ChooseACheckableAbility with at least two distinct fixed keywords,
// followed by one temporary LosesAbility(TheChosenAbility) operation.
// The duration must be UntilEndOfTurn in an ordinary resolving body.
// [ChooseAtResolution] asks the saved player when the body resolves.
// Both options remain available if the subject lacks those abilities.
// Each branch uses [RemoveKeyword] and [Until] with its own typed subject.
// Copies choose separately. Clones keep separate targets and layer effects.
// Source departure does not end the pending ability or installed removal.
// The complete Urborg record has land-play, mana and both-choice evidence.
//
// RON accepts Modal_ChooseOne with at least two action bodies in a
// WhenAPermanentBecomesTapped trigger for ThisPermanent or HostPermanent.
// Each mode has separate targets. Host modes can read Trigger_ThatPermanent.
// Shared modal triggers choose one legal mode at stack placement (CR 700.2b).
// Resolution keeps that mode. Copies keep the mode and can change targets.
// Pending bodies keep source information and exact event visits after departure.
// The complete Relic Bind record has paid-cast and both-mode gameplay evidence.
//
// [PlayersDealtDamageBy] and [PermanentsDealtDamageBy] select exact source history.
// Cleanup keeps that history. Phasing keeps visits but suspends recipients.
// Return cannot inherit an old visit. Current types select permanent recipients.
// Pending abilities keep their source visit and controller after departure.
// Waiting delayed queries retain explicit history source and candidate reads.
// [DamageRecipients] and [DealDamageEach] supply one simultaneous damage batch.
// RON accepts WasDealtDamageByPermanentThisGame(ref) in mixed recipient sets.
// A player And can add Opponent or AnyPlayer. A permanent And can add supported
// characteristic filters. Each set permits one history predicate.
// Other contexts and unbound references reject. The corrected complete The Fallen
// record has cast and upkeep gameplay evidence, including its opponent limit.
//
// [LoseUnspentManaInto] saves the full removed mana in a [LostManaBinding].
// [AddLostManaFor] transfers those values to a typed player (CR 106.13).
// Restrictions remain. The transfer does not produce new mana, apply production
// replacements, spend mana or activate an ability. Empty and absent players
// clear a producer result. Separate frames, loops, delayed bodies, clones and
// payment trials keep separate results. Definition checks require a producer.
// RON accepts AddMana(TheManaLostThisWay) after LoseUnspentMana in an ordinary
// action sequence. PlayerAction selects the donor or recipient. PlayerActions
// keeps local results inside its body. New losses replace the lexical result.
// Entry and draw replacement bodies reject this shape. Drain Power uses
// forced mana activation through [ActivateManaAbilityOfEach].
//
// [PutChosenHandCardOnLibrary] requires one current hand card at resolution.
// The typed player chooses. Empty hands and absent players make no choice.
// Invalid choices use a valid original candidate. Saved hand visits exclude
// returned objects. Canonical cards move to their owners' library tops.
// The move emits a zone event after completion, without discard or reveal.
// Source departure and phasing keep the saved resolving player. Waiting queries
// make no choice. Copies and clones keep separate hands and library state.
// RON accepts PutACardFromHandOnTopOfLibrary in ordinary resolving bodies with
// single-player scope. Entry/draw replacement and all-player bodies reject.
// The complete Conch Horn FEM record has cast, payment and resolution evidence.
//
// [ExileEachWithControllerCounts] saves actual exile counts by former controller.
// Exact visits and controllers are saved before batch replacement choices.
// Duplicate, absent, returned and phased subjects cannot add a count.
// Tokens count. Cancellation and other destinations do not count.
// Shared replacements run in turn order before any batch departure.
// [DrawEachPlayerNumber] reads both counts before ordinary draw actions.
// Number frames keep separate producers, copies, clones and payment trials.
// RON accepts a resolving EachPermanent exile and the per-player creature-count
// draw used by Martyr's Cry. The selection must guarantee creatures.
// Other count filters, consumers, delayed reads and replacement bodies reject.
// The complete DRK record has cast, exile and draw gameplay evidence.
//
// [SpellCast] with AfterFirstInstant matches a caster's second and later instants.
// Successful casts save CastInstantOrdinal from the shared turn tracker.
// Casts before source entry count. Other spell types and stack copies do not.
// Failed casts keep history. Turn start resets each player's count.
// The event filter reads saved facts. Pending triggers keep their typed caster
// and spell through later casts, control changes, source exit and phasing.
// Clones keep separate turn history and pending bodies. Damage uses shared LKI
// and prevention services. RON accepts the opponent instant ordinal TriggerA
// with NotEqualTo(Integer(1)). Other ordinal comparisons and filters reject.
// The complete Ichneumon Druid record has RON cast-and-damage evidence.
//
// [ControllerPlaysLand] saves the player, land visit and turn ordinal.
// Its AfterFirst filter reads the historical play, not current turn history.
// Shared event frames keep pending facts through control, zone and phase changes.
// Failed plays and direct entry supply no event. Permitted library plays do.
// RON accepts the controller land-play TriggerI with the negated first-land
// predicate. The six corrected Fastbond records have complete gameplay evidence.
//
// [AllowUnlimitedLandPlaysFor] removes a live player's land play limit.
// [Game.HasUnlimitedLandPlays] reads the current typed player without mutation.
// Printed source identity, current abilities and phasing control the permission.
// Control changes move You to the current controller. Cleanup keeps the rule.
// [Game.MaxLandPlays] returns math.MaxInt for unlimited plays, before additions.
// Existing global permissions remain available to current callers.
// Main-phase timing, active player, an empty stack and prohibitions still apply.
// Direct entry does not use a land play. Failed plays keep hand and play count.
// RON accepts PlayerEffect(You, [MayPlayAnyNumberOfLandsDuringThierTurn]) on
// permanents. All six Fastbond permission clauses have gameplay evidence.
//
// [CopyPermanent] replaces one exact visit's copiable rules at resolution.
// [Game.BecomeCopyOfPermanent] supplies the same native operation.
// Shared snapshots and physical-card builders supply fresh rules and statics.
// Status, counters, damage, control and attachments stay with the recipient.
// Entry rules do not run. Pending abilities keep their former rules.
// Later copies read the new values. Return restores the physical printed card.
// Phasing keeps the snapshot. Clones keep separate transitions and lifetimes.
// Existing word effects continue in order, once per object instruction.
// Current and cleanup definitions are checked before word effects start.
// Expiry keeps static order. Ordinary word changes stay outside copy values.
// RON accepts one IsACopyOfPermanent(ref, NoCopyEffects) or
// IsACopyOfPermanent(ref, CopyEffects([KeepColor])) in an ordinary resolving
// CreatePermanentLayerEffect. Other copy exceptions and replacement bodies
// reject. MayCopyPermanent supplies an optional resolution choice after both
// visits are checked. RON accepts MayAction around these copy forms.
// [BindTriggeredAbility] supplies one lexical [TriggeredAbilityBinding].
// [RetainCopyAbility] adds a fresh build of that declaration to copy values.
// Pending bodies keep their own declaration after other rules replace it.
// Self-copy adds an instance. Other copies keep all donor instances and add one.
// Nil, duplicate and unbound retention options fail definition checks.
// RON accepts AddAbilityVariable(ThisAbility), alone or with KeepColor, in
// resolving copy bodies of acquired controller-upkeep triggers. The reference
// cannot read donor bindings or a different ability. All six complete Vesuvan
// Doppelganger records have gameplay evidence.
//
// [PlayerLifeManaThisTurn] saves a player permission until cleanup.
// Positive fixed life payments add one fixed mana at mana-ability timing.
// [Game.ActivatePlayerManaAbility] resolves the payment without a stack object.
// [Game.GetPlayerManaAbilities] lists affordable permissions without payment.
// Direct callers supply priority. Shared mana payment services supply payment
// windows, including resolution. Other resolution instructions cannot activate it.
// All permissions share one life budget in the solver. Plans prefer less life
// payment before source preservation. Exact life is payable; zero life is not.
// Total-cost trials and stale-plan checks prevent partial payment on failure.
// Source departure, return, phasing and control changes keep the saved payer.
// Copies and clones keep separate permissions. Cleanup expires each game alone.
// RON accepts CreatePlayerEffectUntil(playerRef,
// [MayPayLifeToProduceManaAsManaAbility(Integer(positive), fixedManaType)],
// UntilEndOfTurn) in ordinary resolving bodies. Replacement bodies reject.
// All six complete Channel records have gameplay evidence. Legacy Channel
// spell-cost substitution remains available for current callers.
//
// [MayAttachPermanentToChoice] lets one saved player accept a legal host choice.
// Shared attachment and choice services check both exact visits before the move.
// The choice does not target. Protection and enchant limits still apply.
// Decline or failed movement keeps the attachment. Waiting queries make no choice.
// State actions run after resolution, so an Aura can move after its host leaves.
// RON accepts PlayerMayAction(playerRef, AttachPermanentToAPermanent) in ordinary
// resolving bodies. Kudzu uses its saved tapping land and that land's controller.
// All six complete records have gameplay evidence. Other optional actions reject.
//
// [DrawGame] ends the game without a winner or loser (CR 104.4c).
// Nested bodies return a stop signal. [ApplyEffect] consumes that signal.
// Remaining instructions, stack objects, state actions and turns do not run.
// [Game.IsGameOver], [Game.IsDraw] and [Game.Winner] report the saved draw.
// Game clones keep separate terminal state. A draw cannot replace an earlier win.
// Ante settlement requires a winner and rejects this result.
// RON accepts the strict DrawTheGame symbol in ordinary non-entry resolving
// bodies. Costs, player actions and entry or draw replacement bodies reject.
// The complete Divine Intervention record has gameplay evidence.
//
// [ControllerRemovesLastCounter] matches a positive removal from the source visit.
// Shared resolving removal and counter costs save the acting player, counter kind,
// actual count and remaining count. Layers update before event detection.
// Zero removal and zone departure supply no removal event. Payment trials keep
// events separate. The source controller must be the actor at event time.
// Pending bodies keep their actor through control, zone and phase changes.
// RON accepts WhenAPlayerRemovesTheLastCounterOfTypeFromAPermanent with
// SinglePlayer(You), a supported counter symbol and SinglePermanent(ThisPermanent).
// Its body can read Trigger_ThatPlayer. Divine Intervention uses DrawGame.
//
// [SacrificeFor] instructs a typed player to sacrifice one exact permanent.
// That player must control the present, unphased visit at resolution.
// [Sacrifice] keeps its saved ability controller. Both use shared sacrifice
// preparation, replacements and events. Waiting queries do not resolve operands.
// RON accepts PlayerAction(playerRef, SacrificePermanent(permanentRef)) in
// resolving bodies, including delayed source departure. Animate Dead reads
// its saved entrant's current controller. Aura control does not select that
// player. All six source records use the corrected creature-controller read.
//
// [AttachToPermanent] attaches one exact permanent to one exact host.
// The shared attachment service checks enchant limits and protection, updates
// both links and timestamps, and applies layers. Shroud and hexproof permit
// the move. Missing, phased and returned visits cannot supply saved objects.
// An illegal move keeps the current attachment. No choice or target is added.
// Waiting source queries retain explicit reads. Copies and clones stay separate.
// RON accepts AttachPermanentToPermanent in resolving bodies, including source
// entry triggers. Singular graveyard returns can save a local entrant in those
// bodies. A failed return clears its result. Entry and draw replacements reject.
// Animate Dead has grant, return, attachment and host-layer clause evidence.
// Its complete record still needs the departure-controller capture adapter.
//
// RON accepts EnteringPermanentPassesFilter(OnTheBattlefield) in a source
// entry TriggerI. [PermanentMatchesFilter] with This and AnyPermanent checks
// the exact source visit. [WithInterveningIf] checks at entry and resolution.
// Departure and phasing make the condition false. A phase cycle keeps the
// visit; return cannot supply the old visit. Control changes keep the saved
// trigger controller. Copies and clones check their own game state.
// This condition has no LKI fallback. Other entry properties need adapters.
//
// [GrantPermanentEnchant] gives one exact recipient a live enchant restriction.
// [Until] saves the recipient. The host query uses that recipient as This and
// its current controller as You. Donor bindings cannot enter the query.
// [RemoveGraveyardCreatureEnchant] removes the fixed graveyard enchant rule.
// The saved attachment stays available to later resolving instructions.
// All current enchant restrictions must permit a permanent host. Protection
// still applies. Shroud and hexproof do not restrict attachment.
// Layer 6 orders grants and removal. Expiry restores printed restrictions.
// Phasing preserves the visit. Return cannot inherit a grant. Clones keep
// separate grant state and saved graveyard attachments. Copiable rules stay
// unchanged. A host query can precede a later linked entry in one resolution.
// RON accepts the paired graveyard-enchant removal and exact linked-creature
// enchant grant in resolving source-Aura bodies, including source entry triggers.
// Entry and draw replacement bodies reject. Animate Dead still needs source
// presence, attachment-result and departure-player adapters for its full record.
//
// [RecordEntryFrom] saves an exact source before graveyard or exile entry choices.
// Successful entries record both visits before entry observations. Failed entries
// record nothing. [PermanentsEnteredWithSource] reads the current matching visits.
// The query can precede entry. Cleanup and source departure do not erase history.
// Phasing suspends recipient selection. Return cannot inherit either visit.
// Clones and payment trials keep separate maps. Waiting queries make no entry.
// RON accepts PutOntoBattlefieldByPermanent(ThisPermanent) in resolving
// graveyard-Aura collections, including source entry triggers and And/Or lists.
// Graveyard returns in that scope record the source through the shared service.
// Static enchant grants, other source references and replacement bodies reject.
// This supplies forward host linkage for Animate Dead. Its full record still
// needs the lasting enchant grant, source-presence and departure-player adapters.
//
// [PermanentsWithoutCountersRemovedBy] reads exact source and recipient visits.
// Only positive removals by resolving counter effects and counter payments enter
// history. Zero removal and zone departure do not enter history. Cleanup keeps it.
// Phasing suspends recipient selection. Return cannot inherit either visit.
// Clones and payment trials keep separate history. Failed payments publish none.
// RON accepts PermanentHasNotRemovedACounterOfTypeFromIt in resolving collections
// and AllCountersOfTypeFromAPermanent in ordinary resolving counter instructions.
// The action player must choose one member at resolution. A local typed binding
// retains the exact visit through removal. This choice does not target.
// Shared choice, removal and persistent upkeep services complete Cyclopean Tomb.
// Entry and draw replacement contexts need separate adapters.
//
// [PermanentsGivenCountersBy] reads counter placement history for an exact source
// and each exact recipient visit. Only positive placements after replacements
// and caps enter history. Counter removal and cleanup do not erase it. Phasing
// suspends recipient selection. Return cannot inherit either object's history.
// Shared single, batch, entry and payment services record placements. Resolving
// effects retain their saved source visit through departure and return. Clones
// and payment trials keep separate history. Raw ID APIs read a current source.
// RON accepts PermanentPutACounterOfTypeOnIt(permanentRef, counter) in ordinary
// resolving collections. Waiting queries retain source reads without placement.
// The shared removal history and choice services complete Tomb.
//
// [AtEach] saves a step subscription for the rest of the game.
// Each match queues a new body with the saved controller and typed references.
// Cleanup, source departure, phasing and control changes do not cancel it.
// A countered trigger does not cancel later matches. Copies and clones keep
// separate results. Existing information services supply exact source LKI.
// RON accepts CreateTrigger with AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))
// and a nonempty ActionList in ordinary resolving bodies. No new targets are
// declared in that body. Cyclopean Tomb uses the shared removal procedure.
//
// [UntilPermanentHasCounter] saves one watched visit and its counter condition.
// It captures layer recipients and values through the shared resolving service.
// No counter at resolution starts no effect. Last-counter removal, departure
// and phasing out end the duration. Later counters or phase-in cannot restart it.
// Counter origin and source departure do not change the saved duration.
// Counter-zero and phase epochs retain transitions between layer evaluations.
// Clones and speculative payments keep separate epochs and installed effects.
// RON accepts ForAsLongAsPermanentHasACounterOfTypeOnIt(EachablePermanent,
// counter) in ordinary resolving singular layer actions. The duration binds
// EachablePermanent to the recipient. SetLandType uses shared subtype layers.
// Cyclopean Tomb uses this duration in its complete RON record.
//
// [GraveyardCreatureEnchantTarget] declares a graveyard creature enchant rule.
// Aura spells save an exact target visit. Noncast entry chooses a graveyard host.
// Copy and entry replacements run before that choice. Attachment precedes entry
// observations. No legal host keeps a noncast Aura in its zone. Batch entry
// checks hosts again after choices. State actions remove illegal attachments.
// Phasing preserves identity and suspends state checks. Clones stay separate.
// [EnchantedGraveyardCard] reads that visit through shared attachment identity.
// Source departure supplies LKI. Returned cards and Auras cannot replace visits.
// RON accepts EnchantGraveyardCard(IsCardtype(Creature), AnyPlayer).
// Resolving hand and battlefield returns can read EnchantedGraveyardCard.
// Entry and draw replacement bodies require separate adapters. Animate Dead's
// complete record still needs linked enchant grants and trigger contracts.
//
// [PutExiledCardOntoBattlefieldOrElse] returns one exact exile visit under its
// owner's control and selects a branch from actual entry. Failed entry clears
// the typed result. Only the success body receives the new entrant binding.
// Shared exile entry supplies replacements, counters, tapped status and Aura
// legality before entry observations. A later departure cannot erase success.
// Delayed frames, stack copies and game clones keep separate saved visits.
// RON accepts required single grouped exile entry with owner control, optional
// tapped entry and saved counters, followed by an adjacent result branch.
// The entrant result is local to success. Coffin's complete record executes.
// Entry and draw replacement contexts require separate adapters.
//
// [WhenLeaves], [WhenUntaps] and [WhenLeavesOrUntaps] watch one exact visit.
// The union consumes one subscription at the first matching future event.
// Cleanup preserves it. Earlier events cannot replay. Return cannot match.
// Phasing preserves the visit. Stun replacement supplies no actual untap.
// Shared delayed frames keep bindings, copies and clones separate.
// RON accepts the source leave and untap events and their two-member Or union
// in ordinary resolving permanent-source bodies. The body gets no event players.
// Coffin uses single exile-entry continuation. Animate Dead still needs
// graveyard Aura contracts and a departure-controller capture adapter.
//
// [BindCounterSnapshot] declares saved counts for all counter kinds.
// [SnapshotCounters] reads one exact visit, with shared departure and phasing LKI.
// [EnterWithSavedCounters] restores counts in native graveyard and exile entry.
// Entry combines saved and native counts before counter replacements apply.
// Copy entry cannot erase external counts. Entry observations read the result.
// Typed number bindings supply cleanup, delayed frames and clone isolation.
// Waiting source queries retain reads without executing a capture.
// RON accepts resolving NoteCountersOnPermanent and a strict
// EntersWithNotedCounters symbol in owner-controlled plural exile returns.
// Branch-local captures cannot escape. Factories keep separate bindings.
// Coffin uses single-entry continuations and the source event union.
//
// [RemoveCreatureEnchant] removes fixed [CreatureEnchantTarget] declarations.
// [Until] saves the exact recipient. Layer 6 removes its enchant restriction.
// Other abilities and enchant restrictions remain. Printed and copiable rules
// stay unchanged. Expiry restores the restriction on the next layer pass.
// Phasing preserves the visit. Return cannot inherit removal. Clones stay separate.
// [Permanent.HasCreatureEnchantAbility] reads the computed fixed declaration.
// RON accepts LosesAbility(Enchant(IsCardtype(Creature))) in ordinary resolving
// CreatePermanentLayerEffect lists, alone or with supported acquired upkeep grants.
// Entry and draw replacements reject. Takklemaggot's complete record executes.
// Replacement enchant grants still need adapters.
//
// [TriggeredAbilityForPlayerModification] adds a typed event-player limit.
// [Until] saves the player and exact recipient when the grant starts.
// The player limit does not change when either permanent changes controller.
// The recipient supplies This and You. Event bindings stay in the new ability.
// Phasing suspends the grant. Return cannot inherit it. Clones stay separate.
// Waiting queries retain source reads without installing a grant.
// RON accepts SinglePlayer(playerRef) in lasting acquired upkeep triggers.
// You retains its current-recipient meaning. Other references are saved when
// the grant resolves. Donor bindings cannot enter the granted action body.
// Takklemaggot's complete record uses the fixed enchant-removal service.
//
// [EnterWithoutSubtype] removes one subtype from an exact entrant in layer 4.
// Entry replacements read the removal. Copy choices cannot erase it.
// Later layer effects use timestamp order. Phasing preserves the visit.
// Return cannot inherit it. Printed cards and copiable values stay unchanged.
// Cancelled entry and stale card visits discard the unused entry effect.
// [EnterAsNonAuraEnchantment] also sets the entrant's type to Enchantment.
// A copied Aura ability can return a card with a different printed type.
// RON accepts EntersAsNonAuraEnchantment in departed-Aura graveyard returns.
// That return and attached returns can supply a local exact entrant result.
// Shared [ReturnCardToBattlefieldInto] clears failed results and saves visits.
// Delayed bodies and clones keep separate frames. Takklemaggot's return now
// executes with its later fixed enchant removal.
//
// [ChoosePermanentForPlayerOrElse] selects a branch after a required choice.
// Shared choice rules save the chooser and exact visit. Empty results cannot
// supply an old choice. [LegalHostsForAuraCard] reads the exact current Aura
// card and its prospective entry controller through shared attachment rules.
// Shroud and hexproof permit the choice. Protection and enchant limits apply.
// The query makes no entry or move. Entry checks legality again.
// Waiting queries retain source reads. Copies and clones keep separate frames.
// RON accepts required ChooseAPermanent with adjacent payment-result branches.
// Only the success branch receives the new choice. Aura host-death bodies
// accept And([collection, CanBeEnchantedByDeadGuest]) for the saved Aura visit.
// Takklemaggot uses this choice in its complete record.
// Entry and draw replacement contexts need separate adapters.
//
// [ReturnCardToHandOrElse] moves one exact graveyard visit to its owner's hand.
// Success selects one branch after the move. Stale visits and tokens cannot move.
// Each call keeps its own result. Copies and clones use separate saved frames.
// Waiting queries retain source reads in both branches without moving a card.
// RON accepts required PutGraveyardCardIntoHand with adjacent If, Unless or
// IfElse payment-result branches in ordinary resolving bodies. Entry and draw
// replacements reject. This completes Puppet Master's LEG record. Exile entry
// uses PutExiledCardOntoBattlefieldOrElse with an exact entrant result.
//
// [EnterAttachedTo] saves one exact host before card entry replacements.
// Shared attachment rules read the entrant after copy changes. Shroud and
// hexproof do not prohibit this entry. An illegal host keeps an Aura in its
// original zone. Other permanents can enter unattached. Return cannot supply
// an old host visit. Registration and attachment precede entry observations.
// Graveyard returns and exile batches use the same typed entry option.
// Delayed queries retain explicit source-host reads without moving cards.
// RON accepts EntersAttachedToPermanent in resolving departed-Aura returns
// and existing plural exile returns. Other entry lists need separate adapters.
// This supplies a prerequisite for Takklemaggot and Tawnos's Coffin.
// Their complete records still reject. Copies and clones keep separate state.
//
// [AttachedPermanentDies] accepts WithDepartedAuraCard with a typed card binding.
// The exact Aura and host visits supply the capture. A simultaneous graveyard
// move or an orphan-Aura state action supplies its first graveyard visit.
// Other departures cannot supply it. Return cannot replace the saved visit.
// Pending clones keep separate captures. Stack copies retain the saved card.
// RON accepts ReturnDeadGuestGraveyardCardToHand in resolving Aura host-death
// bodies. Shared card movement returns it to its owner's hand. This supplies a
// prerequisite used by the complete Puppet Master and Takklemaggot records.
//
// [CreatureDeathsInto] saves exact creature visits from one destruction batch.
// Regeneration, indestructibility, phasing and other destinations exclude visits.
// Tokens that die count. Return cannot change the saved visit.
// [ReturnChosenGraveyardCardsForEachPermanent] binds each saved member while
// it reads a typed graveyard player. The effect controller chooses distinct
// matching cards before any entry. Shared entry rules return the chosen visits
// together under their owners' control. Failed entry keeps the card in its
// graveyard. Each member frame restores the outer bindings. Clones stay separate.
// RON pairs destruction with ActionForEachPermanentThatDiedThisWay and one
// creature-card return under its owner's control. The last-block controller
// expression reads EachablePermanent and one bound blocker in that body.
// Other loop bodies reject. Glyph of Reincarnation uses its complete record.
//
// [PermanentDies] with SourceOnly matches the source's move to a graveyard.
// It permits noncreatures and retains departure owner and controller bindings.
// Duplicate and conflicting source restrictions fail definition validation.
// Departure snapshots supply abilities and controller before the move.
// Return cannot change a queued event. Phasing and other destinations do not
// match. Copies and clones retain separate captured facts.
// RON accepts WhenAPermanentIsPutIntoAPlayersGraveyard with
// SinglePermanent(ThisPermanent) and the strict AnyPlayer symbol.
// The shared graveyard body supplies Trigger_ThatPlayer and
// ControllerOfDeadPermanent. Lich's complete records now execute.
//
// [LoseGameFor] records immediate game loss for one typed player.
// Life-total loss exemptions cannot prevent it. Life and poison do not change.
// An absent subject does nothing. Copies use their own resolving controller.
// Clones keep separate recorded loss. Later life gain cannot reverse loss.
// Definition checks reject nil or unbound players. Waiting queries retain
// explicit source reads without recording loss.
// RON accepts the strict LoseTheGame symbol in ordinary resolving bodies,
// including explicit player action scopes. Entry and replacement bodies reject.
// Lich's unpaid damage-payment branch now executes in all five reprints.
// Its exact-source graveyard event uses the shared death pattern.
//
// [RequiredSacrificeNumberOrElse] reads one count at resolution and requires
// that many matching permanents controlled by its typed payer. The shared
// transaction selects distinct exact visits and moves them simultaneously.
// Insufficient resources or invalid choices cause no partial payment. Phased
// subjects cannot pay. The source can pay if it matches. Zero and negative
// counts succeed without choices. Replacements can change the written actions
// while payment succeeds. Each call keeps its own branch result. Copies and
// clones keep separate payments. Waiting queries retain source reads in the
// payer, amount and branches without making payment.
// RON accepts resolving MustCost(SacrificeNumberPermanents(value, filter))
// with an adjacent If or Unless using CostWasPaid or CostWasntPaid. Existing
// resolution values supply event amounts and source LKI. Filters use the
// reference-free sacrifice grammar. Entry and draw replacements reject.
// Lich's payment now lowers; its loss operations still need adapters.
//
// [DrawInsteadOfLifeGain] replaces positive life gain with individual draws.
// Use [WithStaticAbility]. The typed subject reads the current source controller.
// Current abilities and exact source visits supply replacement candidates.
// Phasing and ability removal suspend the rule. Return starts a fresh visit.
// Shared replacements supply affected-player choices and live rechecks.
// The selected rule saves the player and amount before any draw. Source changes
// during those draws cannot change that instruction. Each draw can be replaced.
// No life is gained. Copies and game clones keep separate source state.
// RON accepts ReplaceWouldGainLife(PlayerWouldGainLife(You),
// [DrawNumberCards(WouldGainLife_LifeAmount)]) on permanent cards.
// The proposed amount is local to that operation. Other forms remain unsupported.
// This establishes Lich's gain clause; sacrifice and loss actions still reject.
//
// [PreventLifeTotalLossStatic] stops one player's nonpositive-life game loss.
// Use [WithStaticAbility]. You reads the current source controller. Departure,
// phasing and ability removal stop the rule through the shared source lifecycle.
// Copies, return and clones keep separate source visits and loss state.
// [Game.PreventsLifeTotalLoss] supplies the pure exemption query.
// [Player.IsAlive] reads recorded loss. [Game.CheckStateBasedActions] records
// life-total loss at priority boundaries. Life restored before that check can
// avoid loss. Later life gain cannot reverse recorded loss. Poison, empty draws
// and explicit loss still apply. Each batch saves loss decisions before moves.
// RON accepts PlayerEffect(You, [DoesntLoseTheGameForHaving0OrLessLife]).
// This establishes Lich's loss exemption. Its later clauses still need adapters.
// Other RON players, operation lists and contexts remain unsupported.
//
// [LoseLifeAsEnters] causes the entry controller to lose a typed amount.
// Shared entry replacements apply it once before battlefield observations.
// Earlier replacements can change the amount. Copy rules can replace the rule.
// A later cancelled entry keeps the loss. Earlier cancellation makes no loss.
// Shared life handling supplies events; damage prevention does not stop loss.
// Nonpositive amounts do nothing. Return reads again; phasing does not.
// Batch entry completes all losses before entry observations. Clones stay separate.
// RON accepts one source-entry LoseLife with Integer or LifeTotalOfPlayer(You).
// This establishes Lich's entry clause. Its other rules still need adapters.
// Other entry life values, subjects and operation lists remain unsupported.
//
// [AcquireStaticAbility] also accepts [PreventUntapDuringStep].
// The exact recipient keeps the grant. Its current counters and controller
// supply untap decisions. Ability removal uses layer order. Phasing suspends
// the rule; return cannot inherit it. Other effects can untap the recipient.
// RON accepts lasting AddAbility lists with conditional untap rules and
// controller-upkeep triggers in ordinary resolving bodies. Each granted body
// has a fresh recipient scope. ThisPermanent and You refer to that recipient.
// [TriggeredAbilityModification] retains each upkeep grant independently.
// This completes Glyph of Delusion. Copy and attachment grants need adapters.
//
// [WasDealtDamageThisTurn] tests positive damage to one present battlefield visit.
// Prevention and redirection supply only the final damage recipient. Spell
// and permanent sources use the same history. Damage removal and control
// changes keep it. Phased subjects are absent; a phase cycle keeps the visit.
// Return cannot supply an old visit. Final turn cleanup clears the history.
// Simultaneous damage records all recipients before observations. Clones keep
// separate maps. RON accepts the strict symbol in creature combat-pair filters.
// Detection checks the opposing visit once. Later damage cannot add a trigger.
// This completes Giant Shark. Other RON filter contexts need separate adapters.
//
// [PreventUntapWhileSourceTapped] saves one subject and source visit.
// The source must be present and tapped when the rule starts. Source untap,
// departure or phasing ends it permanently. Subject phasing keeps the rule.
// Return cannot supply an old visit. Ability removal does not end the rule.
// Other effects can untap the subject. Stun can keep the source tapped.
// Shared untap-step decisions precede simultaneous untap. Copies and clones
// keep separate rules. Waiting source queries do not install a rule.
// RON accepts one DoesntUntapDuringControllersUntap operation with a
// ForAsLongAsPermanentRemainsTapped(ThisPermanent) duration in ordinary
// permanent-source resolving bodies. Phyrexian Gremlins uses its target even
// if that artifact was already tapped. Other durations need separate adapters.
//
// [TappedForAbilitiesThisTurn] selects present payers for one exact source visit.
// Successful activation costs supply actual tap subjects. Ordinary taps and
// resolution payments supply no history. Copies do not pay again. Turn cleanup
// clears history. Untapping and control changes do not clear it. Phased payers
// are absent. Source departure keeps the facts. Return cannot supply old visits.
// Game clones keep separate history maps. Waiting queries do not make history.
// RON accepts WasTappedToPayForAbilitiesOfPermanent(ref) in ordinary resolving
// collections. This reads current-turn payments and completes Vodalian War Machine.
// Entry, target, static and cost contexts need separate adapters.
//
// RON accepts LifeTotalOfPlayer(playerRef) in ordinary resolving bodies.
// [LifeTotalOf] reads current life at each instruction. Negative life remains
// available to arithmetic. Nonpositive life changes do nothing.
// LoseLife_ThatPlayer is local to one LoseLife amount. [SnapshotPlayer] saves
// its explicit subject once. [LoseLifeFor] uses that same player. Factories,
// copies and clones keep separate bindings. Later actions cannot read it.
// Source reads use the existing exact visit, LKI and waiting-query services.
// Entry, replacement, cost and static-value contexts need separate adapters.
//
// [ForEachInTurnOrder] saves exact member visits in turn order.
// The resolving controller chooses the order within each control group.
// Later entries do not join. Return cannot supply an old visit.
// Each sequential body has separate bindings. Errors restore outer bindings.
// Damage results, sources, plural cards, spells and mana types stay local.
// Delayed bodies save their member. Copies and clones keep separate frames.
// Waiting source queries inspect the collection and body without execution.
// RON accepts ActionForEachPermanent(collection, [actions]) in ordinary
// resolving bodies. EachablePermanent is local to the body. Nested loops
// shadow it and restore the outer member. Entry and draw replacements reject.
// This completes Tidal Flats through existing payment and combat-pair services.
//
// [AddSpellDamageToNextSelectedSourceEvent] saves one selected-source event.
// The original damage remains. The saved spell deals the proposed amount to
// that source's controller in the same event. Replacement choices and rechecks
// precede all damage. Each new packet keeps prior replacement uses. The spell
// has its own source, cause and noncombat status. Original instruction results
// exclude its added damage. Local instruction prevention stays with that instruction.
// Exact source visits, selected permanent-spell continuity and retained spell
// information survive departure. Copies and clones keep separate saved rules.
// [Game.WaitingDamageExpansionSources] supplies their saved source references
// to general damage-source choices until consumption or cleanup.
// RON accepts a selected-source next-time Player event in an ordinary spell
// resolving body, with ContinueDealingDamage and one SpellDealsDamage operation.
// The amount and destination must be the proposed damage and source controller.
// This completes Eye for an Eye. Bounded redirection and replacement payments
// require separate adapters.
//
// [TargetPowerLessThanOrEqualToSourcePower] reads current target power and
// current source power. An absent source supplies exact retained power.
// [GainControlWhileSourceTappedAndPowerLE] saves one subject and source visit.
// Untapping, source absence or a failed power comparison ends it permanently.
// Both powers use completed layers. Target phasing ends the power duration.
// Source control changes and ability removal do not end the installed effect.
// [GainControlEachWhileSourceControlled] saves present collection members.
// Later entries and combat changes cannot add members. Source control loss,
// departure or phasing ends each duration. Target phasing preserves it.
// Return cannot supply an old source or subject. Copies and clones keep separate
// members, controllers and expiration state. Both use shared control layers.
// RON accepts source PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent)))
// in permanent targets, paired tapped and subject-power duration endpoints,
// and GainControlOfEachPermanentUntil with UntilPermanentChangesControl(You,
// ThisPermanent) in ordinary permanent-source resolving bodies. These forms
// complete Old Man of the Sea and The Wretched. Other endpoints need adapters.
//
// [RegenerateInsteadOfDestroy] replaces each eligible destruction of its source.
// Current abilities supply the rule. Phasing and ability removal suspend it.
// [ReplaceNextDestructionThisTurn] saves one permanent visit until cleanup.
// It cancels one destruction and removes damage without tapping or combat removal.
// No-regeneration clauses stop regeneration but do not stop damage removal.
// Both use shared replacement choices and rechecks. Clones keep separate state.
// Source departure keeps the saved rule. Return cannot supply the saved subject.
// Waiting persistent replacement queries retain that exact subject after departure.
// RON accepts source ReplaceWouldDestroy with regeneration, and resolving
// CreateFutureReplaceWouldDestroy with CancelDestroy and damage removal.
// Both bodies require WouldBeDestroyed_ThatPermanent. Other operations reject.
// OnlyOtherPlayersMayActivate(Opponent) uses the current source controller.
// These contexts complete Pyramids and Clergy of the Holy Nimbus.
//
// [ControllerControls] tests a present subject's current controller through
// [PlayerControls]. Matching uses present, unphased battlefield permanents.
// [PlayerControls] accepts [CombatDefendingPlayer] for pure combat queries.
// This reference has no value outside two-player combat. [CantAttackIf] checks
// this condition at declaration, after current power and toughness apply.
// Source ability removal, departure and phasing use the shared rule lifecycle.
// RON accepts ControlledByAPlayer(ControlsA(propertyFilter)) in permanent target
// filters, and source CantAttackIfDefendingPlayer with PlayerPassesFilter,
// DefendingPlayer and ControlsA(propertyFilter). Properties use the existing
// reference-free filter grammar. Target checks read the current controller at
// selection and resolution. They do not set a control effect's duration.
// This completes Seasinger and Orgg. Existing positive land checks remain.
//
// [MayRedirectDamageFromPermanentsToPlayerThisTurn] reads live recipients.
// The saved player can accept each redirect until cleanup. Later entries and
// current types use the filter. Phased recipients cannot receive damage.
// [RedirectSpellDamageToSourceControllerThisTurn] saves one exact spell.
// Each damage event supplies its source controller. Departure keeps that fact;
// another cast or spell copy cannot supply the saved identity.
// Shared replacement rules keep order, rechecks, cause, combat and source facts.
// Clones keep separate rules. Waiting spell queries retain the saved source.
// RON accepts resolving CreateReplaceWouldDealDamageUntil with a spell event
// and its source controller, or a permanent filter and optional redirect to You.
// Both forms require UntilEndOfTurn. Other budgets and operations need adapters.
//
// [CreateTokenCopiesOfPermanent] reads one present subject's copiable values.
// [CreateTokenCopiesOfPermanentLinked] also saves the maker's exact visit.
// Fresh token rules precede shared batch entry, replacements and observations.
// Copy exceptions and mana costs remain. Counters, ordinary layers, ownership,
// targets and entry choices do not transfer. Entry rules choose again.
// Zero counts, absent players and absent or phased subjects create nothing.
// Return cannot supply an old subject. Builder errors install no partial batch.
// Source departure keeps the maker link. Source return starts a separate group.
// Copies use their resolving controller. Clones keep separate token state.
// Waiting queries report source reads without building or creating tokens.
// RON accepts resolving TokenCopyOfPermanent(ref, NoTokenCopyEffects), with
// the existing NumberTokens wrapper and player scopes. This completes Dance
// of Many. Copy modifications and lexical creation results need separate adapters.
//
// [WhenAnyLeaves] saves present visits from one permanent set at resolution.
// The next member departure consumes one delayed trigger for the whole set.
// Shared departure snapshots keep simultaneous moves and exact identities.
// Cleanup, source departure and ability removal preserve the subscription.
// Phasing causes no departure. Copies and clones keep separate saved frames.
// Waiting source queries include the saved members without running the body.
// [WhenLeaves] also saves a phased source visit for a later actual departure.
// RON accepts TheTokensCreatedThisWay after one lexical CreateTokens producer
// with a creature prototype. Later producers and other branches keep separate
// results. Both delayed source-leave bodies and delayed token-set leave events
// can retain the result. No event binding is added. Token-copy results and
// combinations with persistent TheCreatedTokens links require separate adapters.
// Complete Stangg gameplay covers paid casting and both delayed leave rules.
//
// [CreatedTokenLeaves] matches one linked token's exact departure visit.
// Shared departure snapshots retain the link after the token leaves.
// Simultaneous moves read the former battlefield. Each member triggers once.
// Source return cannot inherit old members. Phasing causes no departure.
// Source phasing and ability removal stop detection. Queued triggers keep
// their source visit and controller through copies and separate game clones.
// RON accepts TriggerA(WhenAPermanentLeavesTheBattlefield(TheCreatedTokens),
// body) for one permanent-source producer. This event supplies no new binding.
// Lexical delayed token events use [WhenAnyLeaves] and saved result sets.
//
// [CreatedTokenLink] separates one token producer from other producers.
// [CreateTokenCopiesLinked] saves the declaration and source visit on accepted
// tokens before entry observations. Repeated creation adds members. Zero adds
// none. [TokensCreatedWith] selects present, unphased members of that group.
// Source departure, control changes and ability removal do not erase the link.
// Return starts a new source visit. Copies of token cards have no creation link.
// Game clones retain links in separate permanent state. Waiting queries declare
// source reads without creating tokens or selecting members.
// RON accepts one permanent-source CreateTokens producer and TheCreatedTokens
// in later resolving collections. Ambiguous producers and spell sources reject.
// This supplies the links for Dance of Many and a Tetravus prerequisite.
// Optional linked exile payments use [PayChosenExileOrElse].
//
// [CanTargetOnlySubtypeFilter] reads every applicable target declaration.
// [TargetSourceCanTargetOnlySubtype] requires a nonempty complete list and
// subtype proof for each group. Unknown and contradictory domains give no proof.
// Counts and choices cannot remove a group. Raw ability sources give no proof.
// [SpellTargetSourceWithDeclarations] retains a selected spell mode's groups.
// Ordinary spells and Auras use their casting declarations. Resolution and
// copies use the retained spell definition and selected mode. Existing live
// restrictions supply phasing, ability removal, attachment and visit checks.
// RON accepts CanTargetOnly(IsCreatureType(symbol)) in static spell and ability
// target restrictions. This completes Wall of Shadows. Other predicates and
// characteristic spell-filter contexts need separate adapters.
//
// [PermanentTargetDomain] retains immutable subtype predicates in declarations.
// [TargetWithDomain] enforces them in target and attachment checks.
// [TargetRequiresSubtype] proves a subtype from positive and negative tests,
// conjunctions and disjunctions. Opaque restrictions supply no proof.
// Contradictory subtype predicates supply no proof. Counts and chosen targets
// cannot change declarations. Copies and clones retain the same immutable facts.
// RON lowers permanent target predicates and Aura enchant predicates.
// [UnwrapTarget] accepts typed bindings as well as target declarations.
// Spell text checks retain domain structure. Subtype consumers use the proof.
//
// [Game.AbilityTargetSource] retains one selected ability's declaration ID and
// complete target list. [TargetSourceAbility] returns the ID, a separate list
// and a known flag. Empty known lists differ from unknown ability sources.
// Ordinary and selected-mode lists keep zero-choice groups. Current source
// characteristics and exact LKI use the shared source-information service.
// Announcement, pending triggers, resolution, copies and clones keep the facts.
// Graveyard activation checks targets before payment and during resolution.
// Existing RON lowering supplies fresh ability instances and complete groups.
// CanTargetOnly subtype consumers use these complete lists.
//
// [TargetHasSubtype] enforces a subtype in a permanent target declaration.
// [TargetRequiredSubtypes] returns a separate list of those constraints.
// An empty list supplies no proof about the domain. Opaque filters, negation
// and disjunction do not supply a constraint. Other filters can narrow it.
// Target choices, phasing and departure do not change the declaration.
// Live subtype changes still affect selection and resolution legality.
// Bindings, count wrappers, copies and clones keep the constraint.
// RON lowers IsCreatureType(symbol) in permanent target filters, directly
// or within And. This also retains the subtype for fixed and X target counts.
// Complete ability and subtype consumers use separate shared services.
//
// [SpellWithSingleTarget] counts one announced target occurrence on a spell.
// Target legality, phasing and departure do not remove the occurrence.
// Two occurrences of the same object are two targets. Changed target lists
// supply the current count. Copies and clones keep separate lists.
// RON accepts TargetsOnlyASingleTarget as a strict symbol in contextual spell
// target and cast-event filters. Card-only filters reject it. Reflecting Mirror
// still needs announcement X, player-target predicates and retargeting support.
//
// [PreventDamageCausedByTargetingThisTurn] saves one permanent visit until cleanup.
// A resolving spell or ability supplies its own stack identity and target visits.
// These facts stay separate from the damage source. Another permanent can deal
// the damage. Replacements keep the cause when they change amount or recipient.
// Combat and damage outside a resolving stack frame supply no targeting cause.
// Return cannot supply an old visit. Phasing keeps identity. Copies and clones
// keep separate target facts. Shared prevention rules supply order and rechecks.
// RON accepts one TargetsPermanent predicate, AnyDamageSource, the same singular
// permanent recipient, PreventThatDamage and UntilEndOfTurn. This completes
// Silhouette. Other cause predicates and recipients need separate adapters.
//
// [PermanentManaCostPayment] reads a present permanent's current mana cost.
// The shared activation transaction locks this cost before mana abilities.
// X is zero. An absent cost cannot be paid; a printed zero cost can be paid.
// Source and subject must be present and unphased at the read. Attachment
// references require the exact host visit. Payment does not use departure facts.
// Later departure or cost changes keep the locked amount. Other mana components
// join the same total before activation increases and reductions apply.
// Queries do not spend resources. Copies do not pay again. Clones keep separate
// zones, costs and resources. RON accepts PayManaCostOfPermanent(ThisPermanent)
// and PayManaCostOfPermanent(HostPermanent) in nonmana activation costs.
// The latter requires an Aura. Other payment contexts need separate adapters.
// This completes Merseine, including its host-controller activation permission.
//
// [PayChosenManaInto] saves a chosen generic mana payment in a [NumberBinding].
// Zero, invalid input, an absent payer and failed payment clear the result.
// Shared transactions supply mana abilities, spending checks and rollback.
// [NonnegativeNumberChooser] permits direct input. Other players enter digits
// through bounded choices. The amount has no pool or damage bound.
// [DealDamageWithPreventionFrom] prevents part of one damage instruction.
// Its local rule uses shared replacement order and follows redirected damage.
// It leaves no shield. Prevention restrictions still apply. Saved information,
// copies, clones and delays use their separate frames and exact source visits.
// RON accepts resolving PayAnyAmountOfMana, directly or in PlayerAction,
// AmountOfManaPaidThisWay and singular PermanentDealsDamageAndPreventSomeOfIt.
// This completes Power Leak. Entry, replacement and casting payments need
// separate adapters. Branch results cannot supply outer reads.
//
// [SingletonCard] adapts a typed card reference to a [CardSet].
// [ExileGraveyardCards] selects all groups before one shared exile batch.
// Only current graveyard visits move. All owners' cards leave before events.
// Duplicate subjects move once. Return cannot supply a saved visit.
// RON accepts saved plural sacrifice power and graveyard card consumers.
// A grouped exile can include the separate paid source card. Tokens supply
// power but no cards. Copies, clones and delays keep separate payment frames.
// This completes Sword of the Ages, including its source-card exile repair.
//
// [SacrificeSourceAndAnyMatchingCostInto] pays the source and a chosen subset
// in one sacrifice batch. Choices and power reads precede all movement.
// The mandatory source cannot enter the subset. A separate [CardBinding]
// saves its new public-zone visit. Tokens supply no card. The transaction
// locks the source before mana payment and rechecks all visits before commit.
// Return, phasing and changed control cannot replace a payment subject.
// Source and subset results use the existing copy, clone and delay frames.
// RON groups one source sacrifice and one optional-count matching sacrifice
// in an activation cost. Saved result consumers use separate typed bindings.
//
// [SacrificeAnyMatchingCostInto] saves one activation payment's total power
// and new public-zone card visits. The payer chooses zero or more distinct
// present matching permanents they control. The source can pay if it matches.
// Shared payment checks save visits before commit. Return, phasing and control
// changes cannot supply a saved choice. All subjects use one sacrifice batch.
// Replacement and graveyard order choices precede movement. Saved facts precede
// event observations. Tokens add power but supply no card. Exile replacements
// keep successful payment. Cancellation fails the complete transaction.
// Copies, clones and delays keep separate frames. Zero clears prior results.
// RON accepts one SacrificeAnyNumberOfPermanents in an activation cost, with
// one conjunctive ControlledByAPlayer(SinglePlayer(You)) filter.
// Entry payments and casting payments need separate adapters.
//
// [LimitAttackDeclarations] and [LimitBlockDeclarations] limit one declaration.
// The lowest active limit applies. Zero permits no declaration.
// Attack selection obeys the most requirements within the limit.
// Attack costs stay optional. Block selection counts creatures once, even
// when they block several attackers or an attacking band. Shared evasion,
// capacity and minimum blocker checks still apply. Queries do not change combat.
// Phasing, departure and ability removal stop the source rule. Return supplies
// a new rule. Copies and clones keep separate source visits and combat state.
// RON accepts top-level NoMoreThanNumberCreaturesCanAttack(Integer(n)) and
// NoMoreThanNumberCreaturesCanBlock(Integer(n)) with fixed nonnegative n.
// This completes Caverns of Despair. Filtered and temporary limits need adapters.
//
// [PutThenSacrificeFor] retains entry and saves a mandatory sacrifice.
// The player who puts the permanent supplies the count before entry.
// Current source control supplies You. Copies apply before type matching.
// Entry events precede the saved sacrifice. All batch entrants are present
// before observations and sacrifices. The actor can sacrifice the entrant.
// Each rule applies once. Nested sequences finish in reverse replacement order.
// Source departure after application cannot cancel the saved instruction.
// Choices require exact present visits. Return cannot replace a saved choice.
// Phasing and ability removal suspend the source rule. Clones keep separate lists.
// RON accepts an opponent count comparison against the source controller's
// typed permanent count, followed by retained entry and a matching sacrifice.
// This completes Land Equilibrium. Other player predicates and sequences
// need separate adapters.
//
// [EntersTappedFor] supplies a live static entry replacement for selected players.
// Current source control supplies You. Ability removal and phasing suspend it.
// Matching projects entry types and control through shared layers without entry.
// Copy replacements apply first. The affected controller chooses ordinary order.
// Each source visit and ability applies once; execution rechecks after choices.
// Tapped status precedes entry observations, including simultaneous token entry.
// RON accepts one opponent controller clause with nested card-type predicates
// and one EntersTapped symbol. This completes Kismet. Other tapped-entry
// operations require separate adapters.
//
// [SourceCounterCap] limits one counter type on the source's current visit.
// Shared entry, single and batch placement apply the cap after replacements.
// Current abilities supply the rule. Ability removal stops it. Phasing stops
// placement and state-based removal; return supplies a new visit and entry.
// State-based checks remove excess counters when the ability becomes active.
// Multiple caps use the lowest limit. Other counter types keep their counts.
// Clones and factory instances keep separate abilities and counter arrays.
// RON accepts source PermanentRuleEffect with one
// CantHaveMoreThanNumberCountersOfType(Integer(nonnegative), counterType).
// This completes Rasputin Dreamweaver. Other subjects and contexts need adapters.
//
// [UntappedLandCountAtTurnStart] reads a selected player's saved land count.
// [PermanentStartedTurnUntapped] reads the source's exact saved visit.
// [Game.BeginActiveTurn] captures both facts once, before phasing and untapping.
// Skipped untap steps keep the capture. Queries do not create history.
// Later taps, control changes and departure cannot change the saved facts.
// Return starts a new visit. Pending triggers keep the former visit's fact;
// permanent actions still require their saved present visit. Clones keep separate
// history. Extra turns capture again. RON accepts the untapped-land count with
// one SinglePlayer controller filter and a source upkeep history condition.
// This completes Power Surge and the history rule for Rasputin Dreamweaver.
//
// [Until] accepts [ChangeController] with core.Indefinite. Resolution saves the
// exact subject visit and selected player. Source departure and cleanup keep
// the effect. Phasing keeps identity; return starts a new visit. Layer 2 applies
// control effects in timestamp order. Removal restores earlier control.
// [PlayerWithUniqueHighestLife] selects no player on a tie. Later life changes
// cannot change an installed effect. Copies and clones keep separate frames.
// RON accepts singular resolving SetController and GainControlOfPermanent.
// Scoped action players supply the latter's recipient. Entry bodies reject.
// Source-tap mana with one delayed follow-up uses [NewManaAbility] metadata.
// Follow-up effects pass definition and binding checks. Queries schedule nothing.
// This completes Ghazbán Ogre and Rainbow Vale. Plural control needs an adapter.
//
// [DamageSource.TargetsPermanent] reads a spell's saved target visits.
// Stack departure and target illegality keep these facts. Return cannot match.
// Source snapshots, copies and clones keep separate target arrays.
// [PreventDamageFromSpellsTargeting] checks the current subject before prevention.
// [PreventDamageFromSpellsTargetingIf] also checks a live condition each time.
// The shared replacement process supplies rechecks and simultaneous packets.
// RON accepts source TargetsPermanent in static spell-source prevention.
// This completes Bronze Horse. Prevention by cause uses a separate service.
//
// [PayLifeUpToAsEnters] reads a typed cap before entry. The entry controller
// chooses zero through the cap and available life. Shared payment checks run
// before commit. Invalid answers or failed payments save zero. Copy entry rules
// choose again. Entry replacements use the affected player's order once each.
// [PTEqualsEntryLifePaid] applies the saved result in layer 7a. Other zones
// supply zero. [EntryLifePaidOf] reads current, phase or departure facts by visit.
// Return requires a new payment. Clones keep separate results and life totals.
// [CountGraveyardCardsAmong] reads distinct selected players' current graveyards.
// Waiting queries retain typed source reads without choices or payments.
// RON accepts one source PayAnyAmountOfLifeUpto entry operation and paired
// CDA_Power(TheLifePaid)/CDA_Toughness(TheLifePaid) declarations. Fixed bounds,
// sums, supported battlefield counts and one-player graveyard counts supply caps.
// This completes Nameless Race. Other entry payment producers need adapters.
//
// [MaySearchLibraryToHand] offers one bounded search at resolution.
// Acceptance can find zero cards and still shuffles. Decline has no effect.
// Saved library visits keep selected cards distinct. Return cannot replace them.
// Public reveal precedes owner hand movement. All moves precede zone events;
// shuffle follows. Copies and clones keep separate controller frames and zones.
// Waiting queries retain explicit player source reads without choices or movement.
// RON accepts resolving MayAction with a fixed nonnegative basic-land search,
// RevealFoundCards, PutFoundCardsIntoHand and Shuffle. This completes Land Tax.
// Other filters, dynamic counts, entry payments and saved results need adapters.
//
// [PermanentSetCountAtLeast] and [PermanentSetIsEmpty] read current membership.
// Intervening conditions receive event permanent facts before detection and
// separate saved facts at resolution. Queries keep the live frame unchanged.
// Combat sets require present visits; departure information cannot supply them.
// RON combat-pair TriggerI accepts nonempty And/Or conditions, fixed nonnegative
// GreaterThanOrEqualTo counts and absence checks over either blocking relation.
// Property collections accept Other with a typed source, host or event reference.
// Copies, clones and factory instances keep separate saved event subjects.
// This completes Wall of Caltrops. Other comparisons and event families require
// separate adapters.
//
// Combat-pair patterns accept WithSubject to save the watched battlefield visit.
// WithOpponent can save the opposing visit in a separate binding. Detection
// precedes trigger placement. Reattachment cannot change either saved subject.
// Return cannot supply an old visit. Phasing prevents actions but keeps identity.
// Pending events, copies, clones and nested delays keep separate saved frames.
// RON ordinary combat-pair TriggerA bodies accept Trigger_ThatCreature and
// Trigger_ThatPermanent for the watched visit. Shared destruction results and
// delayed counters complete Infinite Authority. Other intervening conditions,
// temporary rule references and subscription subjects need separate adapters.
//
// [core.P2P2] is one +2/+2 counter. Each counter adds two to power and
// toughness in layer 7c. Base values apply first; switches apply last.
// The counter remains separate from +1/+1 and -1/-1 counters.
// Shared counter services supply replacements, removal and departure facts.
// Phasing keeps counters. Return starts a new visit without the old counters.
// Clones keep separate arrays. RON accepts PTCounter(+2, +2) through the
// existing typed permanent-counter adapters. This completes Soul Exchange.
//
// [ExileMatchingPermanentCostWithFacts] saves each [PermanentPaymentFact]
// before the chosen permanent leaves. Current payment-time characteristics
// supply each match. [NumberBinding] results contain one or zero. The shared
// cost transaction publishes results only after success. Return, ability loss
// and later type changes cannot change them. Copies and clones keep saved facts.
// RON accepts APermanentWasExiledThisWay in resolving If branches after one
// nonmodal additional casting payment of APermanent. Supported characteristic
// filters and nonempty And/Or lists can read these facts in delayed bodies.
// Activation, entry, plural and resolving payment producers need adapters.
//
// [ReturnCardToBattlefieldInto] saves the exact successful entrant in a
// [PermanentBinding]. Failed entry clears the result. The shared entry service
// supplies replacements, choices, tapped status and events. [From] requires
// the saved present, unphased visit for actions. Information reads can use LKI.
// Copies, clones and delayed bodies keep separate result frames. RON accepts
// ThePermanentPutOnTheBattlefieldThisWay after a singular creature-target
// graveyard return in a resolving action list, including source entry triggers.
// Graveyard Auras can supply the result. Payment, exile-return
// and replacement producers need separate adapters.
//
// [NewSourceCounterManaAbility] adds one fixed mana type per source counter.
// Queries and activation read the same live count without spending counters.
// Zero counts have no production. Missing and phased sources have no production.
// The ability resolves immediately. Return uses new counters; clones read their
// own battlefield. Shared mana services supply payments, replacements and bonuses.
// RON accepts one source tap cost and one AddManaRepeated action with a fixed
// mana type and source SinglePermanent counter count. Other subjects, mixed
// actions, modifiers and resolving counter-mana bodies need separate adapters.
//
// [ExileMatchingPermanentCost] requires one present matching permanent that the
// payer controls. The source can pay. Queries make no choice. The shared cost
// transaction locks the chosen visit and checks the total payment before commit.
// Return, phasing and changed control cannot replace that visit. Exile keeps
// ownership and uses the shared departure events. Copies do not pay again.
// RON accepts Exile with one APermanent filter and no modifiers in activation
// and nonmodal additional casting costs. The filter requires one conjunctive
// ControlledByAPlayer(SinglePlayer(You)). Additional casting payments can
// save characteristic facts with [ExileMatchingPermanentCostWithFacts].
//
// [ExileGraveyardFor] reads one player's current graveyard at resolution.
// Saved visits cannot follow returned cards. All moves precede zone events.
// One graveyard departure event supplies the completed batch count. Cards added
// during event observations cannot join that batch. Existing single-graveyard
// exile payments use the same movement service. Copies and clones keep their
// player frames and zones separate. Waiting queries retain explicit source reads
// without movement. RON accepts EachCardInGraveyards with one InAPlayersGraveyard
// and SinglePlayer reference in resolving Exile with no modifiers. Entry payments
// and multi-graveyard player sets need adapters.
//
// [SacrificeAnyNumberAsEnters] offers a bounded permanent choice before entry.
// The controller can sacrifice only present matching visits they control.
// Shared replacements and owner ordering precede one simultaneous batch.
// Completed sacrifices supply the saved count, including exile destinations.
// [PTEqualsEntrySacrificeCount] applies that count in layer 7a. Other zones
// supply zero. [EntrySacrificeCountOf] can read exact departure or phase facts.
// Return and copied entry require new choices. Clones keep separate results.
// RON accepts one source entry sacrifice and matching dual source CDA reads.
// Activation payments and retained plural sacrifice results need adapters.
//
// [PlayerControlsAny] checks current control of live set members.
// [PreventDamageFromMatchingPermanentsToSetIf] checks live recipients before
// each replacement match. Both require a present exact source visit. The source
// controller supplies You. Phasing and return cannot supply a saved visit.
// Waiting prevention reads use the same membership and condition checks.
// RON accepts source SinglePermanent, Other and BandedWith, You control,
// supported characteristic filters and nonempty nested And/Or collections.
// Static ControlsA/ControlsNo and APermanent prevention recipients consume these
// queries. Resolving, entry and payment contexts keep their separate adapters.
//
// [ChooseNumberAsEnters] saves the controller's number before entry.
// [MayChooseNumber] offers an update to the resolving controller. Decline keeps
// the number. The subject must keep the same unphased visit through each choice.
// [ChosenNumberOf] reads current, phase or departure information for that visit.
// Missing choices supply zero. Return and copied entry make new choices.
// Clones keep separate integer and presence fields. Existing layers apply the
// live number before bonuses, counters and switches. Queries make no choices.
// RON accepts one source ChooseANumberBetween entry declaration, optional
// resolving battlefield updates, and separate static SetPower/SetToughness
// consumers. Other entry results and payment producers need separate adapters.
//
// [ChooseCopiableVariantAsEnters] requires a controller choice before entry.
// Each option applies a list of [CopyModification] values. Options and rules
// keep separate definitions. The shared entry replacement process applies each
// rule once. Copy replacements run first. Other replacements can read or replace
// the chosen values. Spell copies keep their token status and make new choices.
// [CopySetPowerToughness], [CopyAddSubTypes] and [CopyAddKeywords] change
// copiable base values. They keep existing types and abilities. Counters and
// later layer changes are not copied. Phasing keeps values; departure restores
// printed rules. Return requires a new choice. Clones keep separate objects.
// RON accepts EntersWithLayerEffectOfChoice in one source entry operation.
// Options contain fixed P/T, permanent type, creature type and keyword additions.
// Numeric entry/upkeep choices and other operations need separate adapters.
//
// [BandedWith] selects the present members of one current attacking band.
// The subject and members must match their saved combat visits. No departure
// facts can supply a member. Ability loss keeps the band; combat removal and
// phasing remove membership. A one-member band includes its subject.
// Queries do not change combat. Waiting delayed queries retain explicit source
// operands without selecting members. Clones keep separate combat and frames.
// RON accepts BandedWith(permanentRef) in resolving collections, including
// nested And and Or lists. Temporary grants save recipients at resolution.
// Static prevention accepts source band membership in a live recipient set.
// Entry and payment declarations need separate adapters.
//
// [PreventDamageFromMatchingPermanentsIf] checks a live static condition.
// Replacement rechecks and waiting source queries use the same condition.
// The current source controller and exact battlefield visit supply its context.
// RON static If, Unless and IfElse accept lists of supported rule, prevention
// and layer declarations. All branches must pass definition checks.
// Layer parts keep their existing membership and same-layer limits.
// Conditional prevention accepts one permanent source filter and PreventThatDamage.
// Recipients can use ThisPermanent or a supported live static collection.
//
// [YourMainPhase] matches each main phase on the controller's turn.
// [AtNext] waits for the next actual boundary, including extra main phases.
// Cleanup and skipped phases do not consume this schedule. The resolving
// controller stays saved after source departure or phasing. Copies and clones
// use the shared delayed frames. RON accepts CreateFutureTrigger with
// AtTheBeginningOfPlayersNextMainPhase(You) and a nonempty ActionList.
// Fixed-color AddManaRepeated accepts ManaValueOfSpell with a bound spell
// target or event reference. [ManaValueOfSpellWithLKI] keeps stack departure
// values by exact spell identity. A later cast cannot supply that value.
// Counter failure does not stop the independent delayed instruction.
// Waiting and pending bodies keep the referenced spell available to damage
// source choices. Queries use current or departure facts without adding mana.
//
// [WhenReceivesDamageThisTurn] watches one saved battlefield visit until cleanup.
// Each positive damage event saves its final amount before trigger placement.
// The creator keeps control. Departure and return cannot supply the watched
// visit. Phasing stops damage; a phase cycle keeps the subscription. Source
// filters use the shared damage source service and its departure information.
// Copies, clones and pending bodies keep separate amounts and saved frames.
// RON accepts CreateTriggerUntil with WhenAPermanentDealsDamageToAPermanent,
// a supported damage source filter, SinglePermanent, ActionList and
// UntilEndOfTurn. Trigger_ThatMuch is local to that delayed body's value scope.
// Other event subjects and durations need separate adapters.
//
// [DiscardCountInto] saves completed discards after one shared replacement batch.
// Required choices use distinct saved hand visits. Short hands supply only their
// available cards. Library replacements count; cancellation and returned visits
// do not. Zero amounts and missing players clear the saved count.
// [ReturnChosenGraveyardCardsToHand] selects distinct current graveyard visits
// at resolution. All choices precede movement. Departure and return cannot
// supply a saved visit. Cards enter their owners' hands before zone observations.
// One graveyard departure event reports the actual batch count.
// Existing callers of [Game.MoveCardsFromGraveyard] keep their event timing.
// RON accepts DiscardNumberCards with fixed values or bound ValueX, and the
// lexical NumCardsDiscardedThisWay result after a direct discard instruction.
// ReturnNumberCardsFromPlayersGraveyardToHand accepts that count, AnyCard and
// an existing singular player reference. Copies, clones and delayed bodies
// keep separate result frames. Other amounts and filters need adapters.
//
// [WhenCombatPairUntilEndOfCombat] saves one watched battlefield visit.
// Each new block pair can trigger it until combat priority ends.
// The creator keeps control. Source control changes and ability removal do not
// stop the saved rule. Departure and phasing cannot supply a new pair; a phase
// cycle keeps the visit. Detection saves each opponent before trigger placement.
// Pending bodies survive expiry and source departure. Return cannot supply a
// saved opponent. Copies and clones keep separate frames and effect lifetimes.
// RON accepts CreateTriggerUntil with the source block/becomes-blocked union,
// matching creature filters, ActionList and UntilEndOfCombat in resolving
// permanent-source bodies. Trigger_ThatOtherCreature reads the saved opponent.
// Host subjects, single directions and other durations need separate adapters.
// First-strike grants use the existing layer service until turn end.
//
// [BindAbility] and [FromAbility] keep an activated ability's exact stack identity.
// [ActivatedAbilityEvent] can require an attached source, a tap symbol and a
// nonmana ability. [GameEvent.ActivatedAbilityID] is empty for mana activations.
// Detection saves the identity before trigger placement. Missing or departed
// abilities supply no subject. Source departure, return, control and phasing
// cannot change the saved ability. Copies and clones keep separate frames.
// [CounterAbility] uses the shared counter service without a cost refund.
// RON accepts the host, tap and nonmana conjunction in battlefield TriggerA.
// Trigger_ThatActivatedAbility supplies CounterAbility and ControllerOfAbility
// in that body. Other event filters and delayed RON reads need adapters.
//
// [TargetOtherThanSavedPermanent] excludes one saved battlefield visit.
// Target placement, copy choices and resolution use separate event query frames.
// Missing bindings supply no targets. Departure and phasing keep the saved
// identity; a returned permanent cannot inherit the exclusion. Reattachment
// cannot change the event subject. Factories, pending events, copies and clones
// keep separate facts. Existing target protections and legality checks apply.
// RON accepts Other(bound event permanent) in singular permanent target filters,
// including And lists, in ordinary battlefield triggers. Resolving choices,
// prior targets, death-card references and delayed target scopes need adapters.
//
// [AttachedAttacks], [AttachedBlocks], [AttachedAttacksUnblocked] and
// [AttachedAttacksOrBlocks] save the attached creature at event detection.
// The union is one ability that matches either declaration event.
// A block declaration supplies one trigger even with multiple attackers.
// Reattachment cannot change a saved visit. Pending bodies retain their source
// controller. Departure, return and phasing use the shared reference services.
// Copies, clones and delayed bodies keep separate event frames.
// RON accepts these host events in battlefield TriggerA bodies. The attack/block
// Or list requires both events with SinglePermanent(HostPermanent).
// Trigger_ThatCreature and Trigger_ThatPermanent read the same saved host.
// Defending-player capture and host becomes-blocked events
// need separate adapters. Saved event target exclusion uses the shared query.
//
// [SwitchPowerToughness] uses layer 7d after base values, bonuses and counters.
// It has one subject and no numeric operands. Definition checks keep the subject
// requirement. [Until] saves the exact visit at resolution. Later bonuses still
// apply before the switch. Two switches cancel. Phasing keeps installed effects;
// departure and return cannot transfer them. Clones keep separate lifetimes.
// RON accepts SwitchPT in singular CreatePermanentLayerEffectUntil lists.
// Existing resolving references, scopes and temporary durations apply.
// Static, plural, entry and payment forms need separate adapters.
//
// [TargetChosenBy] uses a saved event player at trigger placement.
// The query keeps the live resolution frame. Legality uses the ability controller
// and source. Missing players cannot supply choices. Copies keep the event facts;
// their controller chooses new targets. Resolution checks the exact target visit.
// RON accepts TargetPermanentOfPlayersChoice(filter, Trigger_ThatPlayer) in
// bound step-trigger bodies. ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))
// uses the same binding with current control. Other chooser contexts reject.
//
// [PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl] returns creatures together.
// Selection saves current graveyard visits before entry choices in APNAP order.
// Returned visits and cards added during choices cannot join the batch.
// Shared entry rules supply copies, choices, counters and tapped status.
// Blocked or failed entries keep their graveyard visits. Uncast entry uses zero X.
// All entrants are registered before graveyard departure and entry observations.
// Each owner gets one graveyard departure notification with the actual count.
// RON accepts PutEachCardFromEachPlayersGraveyardOntoTheBattlefield with
// IsCardtype(Creature), AnyPlayer and one EntersNormally in resolving bodies.
// Other filters, players and entry options need separate adapters.
//
// [ThisSpell] reads the exact spell whose body is now resolving.
// [ExileSpell] and [ExileSpellWithCounters] move one present spell to face-up
// exile. Fixed counters are set before zone observations. Later instructions
// continue. Repeat actions cannot move a departed spell or its new visit.
// Cleanup cannot move it again. Copies use their own visit and cease at the
// next state-based check. Clones keep separate visits and counter arrays.
// RON accepts Exile with one Spell(ThisSpell) in a resolving instant or sorcery
// body. It accepts no modifiers or one ExilesWithNumberCounters with a fixed
// nonnegative amount. Targets, events, costs and delayed bodies need adapters.
//
// [RemoveCounterFromExiledCard] removes one available counter from a saved,
// present face-up exile visit. Missing, hidden or returned visits supply no
// subject. Later instructions still run. Counts and saved bodies stay separate
// in clones. Copies keep the saved visit and read its current count.
// RON accepts RemoveCounters with one ACounterOfTypeFromCardInExile instruction
// for ThisExiledCard in resolving FromExile owner-upkeep bodies, including If.
// Other references, amounts, costs and counter-removal events need adapters.
//
// [ExiledCardHasCounter] and [ExiledCardHasNoCounters] check one saved exile visit.
// Both require a present, face-up card. Departure and return cannot supply reads.
// Missing visits cannot supply a zero count. Detection and resolution read the
// current counters in separate query frames. [ExiledCard] stores integer counts;
// clones keep separate arrays. New exile visits start without counters.
// RON accepts HasACounterOfType and HasNoCountersOfType in
// ExiledCardPassesFilter(ThisExiledCard, filter) for owner-upkeep TriggerI and
// resolving If bodies under FromExile. [RemoveCounterFromExiledCard] supplies
// singular source removal. General placement needs a separate adapter.
//
// [NewManaAbilityOfColorsOpponentPermanentsCouldProduce] adds one mana of
// an available color. The source's current controller defines the opponent.
// Queries use present abilities and layers. They ignore costs and tapped status.
// A local fixed point includes recursive production and all replacement orders.
// Cycles without a defined color produce nothing. Phased subjects supply no color.
// Queries do not pay, choose or run riders. Source restrictions are not copied.
// Manual choices and automatic payment use the same finite color options.
// Empty options still permit the tap activation and produce no mana.
// Copies and clones read their own source, controller and battlefield.
// RON accepts one source tap activation with AddMana and
// AnyManaColorAPermanentCouldProduce over a characteristic filter and Opponent.
// Other costs, bodies, players and resolving-action contexts need adapters.
//
// [CopyInstantOrSorceryWithColors] sets colors within the spell copy process.
// The copy uses these colors before it can choose new legal targets.
// Empty lists make colorless copies. Color definitions are omitted.
// Later copies keep the exception. Ordinary color changes are not copied.
// Shared stack services keep X, modes, allocations, payments and source identity.
// Copies do not cast or pay again. Absent spells cannot use departure facts.
// RON accepts one fixed SetColor operation for an instant-or-sorcery target
// in resolving spell bodies. Other contexts and exceptions need adapters.
//
// [DiscardLastDrawnCardCost] requires the payer's last successful draw this turn.
// The exact hand visit must remain present. Return cannot supply that visit.
// Shared transactions lock the card and recheck history before payment.
// Queries and payment selection make no choices. Replaced draws keep history
// unless their bodies draw cards. Copies do not pay again. Clones keep separate
// history and zones. Turn cleanup clears history after pending actions finish.
// RON accepts DiscardCard(TheLastCardDrawnThisTurn) in activation costs,
// including And lists. Resolution payments and other card references reject.
//
// [ClampedDamageAmount] reads one saved singular damage outcome.
// Actual damage follows replacements and prevention. The final recipient supplies
// player life and loyalty before damage. Creature toughness uses the exact visit
// at the read, including departure and phase information. Return cannot replace
// that visit. Negative caps supply zero. Marked damage does not reduce toughness.
// RON accepts TheClampedAmountOfDamageDealtThisWay after one ordinary singular
// damage instruction. Results stay local to branches and factory instances.
// Delayed bodies and clones keep separate saved outcomes. Collections, entry
// operations and replacement bodies need separate adapters.
//
// [MayReturnNonAuraCardToBattlefield] offers normal entry for one exact
// non-Aura graveyard card.
// The selected player chooses at resolution and controls the returned card.
// Departure and return before or during choice cannot supply the saved visit.
// Missing players, absent cards, Auras and nonpermanent cards supply no offer.
// Shared entry rules run before entry events. Copies and clones keep separate
// frames. Decline does not stop later actions or later offers. Waiting queries
// retain explicit player source reads without choices or movement.
// RON accepts MayAction around PutGraveyardCardOntoBattlefield with
// ThisGraveyardCard and one EntersNormally in non-Aura graveyard source lists.
// Other references, entry options and player-action contexts need adapters.
//
// [WithXSpendingColor] limits mana spent on a spell's announced X.
// [SpellContextForCard] keeps the declaration in a fresh payment context.
// Shared total-cost locking saves the remaining restricted generic amount.
// Reductions remove restricted X first. Taxes and other generic costs remain
// unrestricted. Announced X does not change. Channel cannot replace this
// remaining amount with life. As-though permissions can supply its color;
// each mana unit keeps its actual color and spending restriction.
// Pool payment, automatic production and solver demand use the same rule.
// [Game.MaxXValue] applies it to the supplied cost without changing the context.
// It retains its existing supplied-cost contract for taxes and reductions.
// Copies do not pay again. Clones keep separate pools and announced values.
// RON accepts CastEffect(SpendOnlyColorManaOnX(color)) on one ordinary
// instant or sorcery with printed X. Modal and text-revision contexts need
// separate adapters. [ClampedDamageAmount] supplies Drain Life's life gain.
//
// [ControllerManaSpending] permits the source controller to pay with one
// mana color as another color. Active permissions combine, including chains.
// The payment planner and live payment keep actual colors and spending rules.
// Rule reads follow object layers and current control. Phasing and ability
// removal stop the rule. Return starts a new visit. Copies and clones keep
// separate rules and pools. Queries do not spend mana or make choices.
// RON accepts PlayerEffect(You, [MaySpendColoredManaAsThoughItWereAnotherColor
// (color, color)]) with one operation and two of the five mana colors.
// Other players and temporary permissions need separate adapters.
//
// [AddRestrictedMana] saves fixed productions with one spending rule.
// All productions use one shared mana replacement event. Each mana unit
// keeps its rule through type replacements, payment and game clones.
// Fixed metadata uses the same rule as manual production. Queries do not
// pay costs or change pools. RON accepts AddManaWithModifiers with fixed
// mana symbols, nonempty And lists and artifact-only or creature-only spending.
// The same spending adapter serves repeated one-color resolution production.
// Restricted X payment uses the separate spell spending contract.
//
// [WithSourceCardInZone] saves a graveyard or face-up exile source visit.
// Ordinary events use the owner as controller outside the battlefield.
// Source departure does not stop queued bodies. Return cannot supply a read.
// Source actions use [FromCard]; [ExiledCardFrom] uses the shared exile moves.
// Detection keeps separate event frames and does not change card declarations.
// Copies, clones and delayed bodies keep the saved card and player facts.
// RON accepts FromGraveyard or FromExile around an ordinary owner-upkeep
// TriggerA or TriggerI with ActionList. State events, targets and modes need
// separate contracts. Source damage and battlefield source reads reject.
// [WithInterveningIf] reads source-card facts in a separate query frame.
// [GraveyardCardHasAtLeastAbove] counts matching cards above a saved visit.
// [CardMatchesFilter] checks the current card at a saved visit.
// Both conditions require [FromCard]. Departure and return cannot supply reads.
// Detection and resolution use current facts without changes to game state.
// RON accepts a fixed nonnegative minimum and one card type for the ordered
// graveyard condition. Exile conditions accept one card type. Exile counters
// and optional source returns need separate adapters.
//
// [PlayerControlsChosenColor] checks live subjects against the source entry color.
// [PlayerControls] also accepts ChosenPlayerOf(This) for the entry player.
// Both choices stay with the source visit through control changes and phasing.
// Phased or departed sources cannot supply choices. Return starts a new visit.
// Subject colors and controllers use current layers. Clones keep separate choices.
// RON accepts PlayerPassesFilter with You or a source entry-chosen player,
// ControlsA or ControlsNo, and a chosen-color predicate in a conjunction.
// Choice disjunctions and target predicates stay separate.
//
// [PlayerControlsNone] checks absence only for a known player.
// [PlayerControlsNoneOfChosenColor] also requires a live source color choice.
// Missing choices, source departure and source phasing cannot supply absence.
// RON accepts a source entry-chosen player in no-permanent state triggers.
// Fixed filters and chosen-color conjunctions use current control and colors.
// [WhenState] keeps one pending or active trigger per source visit and rule.
// Pending bodies keep the detected controller and source visit. They do not
// check membership again. Return cannot supply that source. Copies and clones
// keep separate choices and pending bodies. Ability removal stops detection.
//
// [OpponentUpkeep] checks the source controller when an upkeep begins.
// It saves that upkeep player in a separate event binding. Pending bodies keep
// the player and controller through control changes, departure and phasing.
// Copies, clones and delayed bodies keep separate frames. Phasing and ability
// removal stop new detection. Return starts a new source visit.
// RON accepts AtTheBeginningOfAPlayersUpkeep(Opponent) in ordinary TriggerA.
// Opponent upkeep text revision and intervening-if adapters stay separate.
//
// [ChooseColorAsEnters] saves the entry controller's choice before entry.
// Only the five colors are legal. Invalid answers use white.
// Copies choose again. Phasing and control changes keep the saved choice.
// Return starts a new visit and a new choice. Clones keep separate storage.
// RON accepts one ChooseAColor(AnyColor) operation in one source entry rule.
// [PermanentsOfChosenColor] reads the exact reference visit's saved color.
// Departure and phasing use saved information. Return cannot replace the visit.
// Subjects use current colors and must be present and unphased.
// RON accepts IsColor(TheChosenColor) in resolving permanent collections
// with a source entry choice. Target predicates stay separate.
// [RequiredEntryOperations] runs one required sequence before entry.
// Children run in order. A rejected child stops the sequence and entry.
// RON accepts one source entry rule with a color choice then an opponent choice.
// Both choices use existing services and keep separate permanent fields.
// Copies and returns choose again. Control changes and phasing keep choices.
// Other joint entry operations need separate adapters.
//
// [PreventDamageToSourceByRemovingCounters] removes one matching counter for
// each damage point and prevents that point. Unpreventable damage still removes
// counters, but it keeps its amount and does not consume finite shields.
// Shared replacement choices apply each rule once per packet and check the
// current recipient again. Phasing and ability removal stop the static rule.
// Return requires a new source visit. Clones keep separate counters and rules.
// RON accepts EachDamageWouldBeDealtToRecipient(Permanent(ThisPermanent)) with
// one matching counter condition, one source counter removal and PreventThatDamage.
// Other recipients and replacement bodies need separate adapters.
//
// [ExileMatchingCardsFromSingleGraveyardCost] requires distinct matching cards
// from one eligible graveyard. The payer chooses; other players do not choose.
// Queries make no choices. Shared transactions save and recheck exact visits.
// A returned card cannot pay. All cards reach exile before zone events.
// Copies do not pay again. Separate activations and clones keep their choices.
// RON accepts fixed nonnegative counts and AnyPlayer in activation costs.
// RON entry graveyard payment uses a separate declaration. Other dynamic
// activation counts and plural payment reads need adapters.
//
// Permanent counters and exact departure snapshots use integer arrays.
// [EntersWithNCounters] and the shared counter replacement service keep counts
// above 255. Clones keep separate arrays. Phasing keeps counters; a zone return
// starts new counters. RON fixed entry amounts accept nonnegative integers.
// Typed entry X and entry payment rules use separate declarations.
//
// [AnnouncedXForEnteringPermanent] reads the entering spell's own X.
// [EntersWithCounters] uses this typed entry value before entry events.
// Direct entry supplies zero. Copies keep the original spell's X.
// Entry copy rules use the entering spell's X, not the donor's counters.
// [EntersWithXCounters] uses the same entry facts. Phasing keeps counters;
// return starts a new visit with zero X unless a new spell supplies it.
// RON accepts ValueX in one EntersWithNumberCounters entry operation.
// Use [ExileGraveyardAsEntersWithCounterChoice] for entry exile and counters.
//
// [TargetSharesCardTypeWith] compares a target with an earlier singular target.
// Both exact visits must be present and unphased. Current types supply the
// comparison; departure information cannot restore a target. A supplied type
// list contains distinct valid types. An omitted list permits every card type.
// [Game.PossibleTargetGroup] keeps prior choices in a separate query frame.
// Each candidate must permit a complete remaining selection. Queries do not
// choose, pay, use randomness or change the current resolution frame.
// Announcement, forced choices, triggers and copies use these declarations.
// Resolution checks each saved visit before actions. Copies can keep illegal
// old targets; explicit legal choices save current visits. Clones and factory
// instances keep separate choices. RON supports both card-type comparisons
// against earlier numbered singular permanent targets in conjunctions.
// Other reference kinds, target unions and plural groups need separate adapters.
//
// [HostOfPermanent] reads the host of an exact attachment visit.
// Current attachments use their current host. Departed attachments keep their
// last host. Return cannot replace either visit. Information reads can use host
// LKI; actions need a present, unphased host. RON binds HostPermanent in resolving
// spell bodies with one singular Aura target. Aura source bodies keep their
// existing source-host read.
//
// [AttachPermanentToChoice] makes a required legal host choice at resolution.
// The choice does not target and excludes the current host. Shared attachment
// rules check enchant limits, protection and attachment types. Both saved visits
// must remain present and unphased after the choice. A move keeps control and
// ownership, changes both host links and updates the static effect timestamp.
// Copies, clones and delays keep separate frames. Waiting queries make no choices.
// RON lowers AttachPermanentToAPermanent with a singular reference and a
// resolving host collection. Scoped player bodies keep their saved chooser.
// Entry and replacement operations need separate adapters.
//
// [PermanentsSharingCardTypes] selects present subjects with a shared card type.
// The reference reads current or exact departure and phase information.
// Return cannot replace the saved visit. With no type list, all types can match.
// Restricted lists contain distinct valid types. Absent and phased subjects
// cannot match. Existing frames keep targets, choices, copies and delays separate.
// Waiting queries retain explicit source reads without running the comparison.
// RON accepts both card-type comparison forms in resolving collections.
// Target predicates, entry operations and replacements need separate adapters.
//
// RON temporary singular control uses [ChangeController] with [Until] or
// [UntilSourceUntaps]. Subjects keep their exact visits and the effect controller.
// Cleanup ends end-of-turn control. Tapped-source control starts only with a
// present, unphased, tapped original source. Untapping, departure and source
// phasing end it permanently. Control changes and ability removal do not end it.
// Copies, clones and delayed bodies keep separate targets and saved frames.
// Entry and replacement declarations need separate adapters.
//
// [DamageOutcomesInto] saves actual singular damage outcomes after replacements.
// [IfDamageDealtToPlayer] checks those outcomes against an exact source and player.
// Prevention and redirection can stop the match. Departure does not erase facts;
// return cannot replace a saved visit. Zero damage clears older outcomes.
// [PlayerDamageRecipients] selects distinct players for the shared damage batch.
// Existing result frames keep delayed reads, copies and clones separate.
// RON reads TheAmountOfDamageDealtThisWay from its latest local damage producer.
// PermanentDealtDamageToPlayerThisWay checks that producer's saved outcomes.
// Branch-local producers cannot supply outer reads. Entry and replacement
// operations retain separate contracts.
//
// [PreventLandEntry] stops land entry through shared preparation.
// [Game.IsLandEntryProhibited] reads active source rules without changes.
// Phasing, departure and ability removal stop the rule. Return starts a new visit.
// Copies that become lands stop before later entry replacements.
// Blocked entry keeps hand, graveyard, library and exile cards in their zones.
// Token batches keep permitted members. Land plays check before zone removal.
// RON accepts CardsCantEnterTheBattlefield(IsCardtype(Land)) on permanents.
//
// [PayAnyAlternativeOrElseWithPayer] offers complete payment alternatives.
// Each eligible player chooses in turn order. The first payment selects paid.
// [CostPaymentOption] uses shared transactions and rollback.
// [ActionPaymentOption] saves values and visits before the offer.
// Accepted damage still pays when prevention stops all damage.
// The payer stays local to offers, options and the paid body.
// Copies, clones and delays keep separate frames and source information.
// RON accepts fixed mana, matching sacrifice and permanent damage options
// in AnyPlayerMayCost, including nested Or lists in resolving bodies.
// Entry, replacement and announcement payments keep separate contracts.
//
// [PermanentLifePaymentInstruction.Effect] retains fixed optional life offers,
// an exact unpaid-set declaration and the ordered body. Execution uses the
// shared payment transaction and resolution frame. Missing operands, unnamed
// results and negative amounts return nil. The inspector makes no offers.
// Pure spell inspection accepts complete collections without target reads and
// direct destruction trees that read only this unpaid binding. It validates
// every child before target gates or proofs. It does not predict payment choices.
// Optional destruction supplies no certainty proof. An optional prefix blocks
// later destruction proofs, including empty collections and zero life costs.
// Supported suffixes preserve earlier direct proofs. Other continuations keep
// their execution support but need separate pure inspection contracts.
//
// [OfferCostForEachPermanent] saves visits for which no eligible player pays.
// Each visit needs its own complete payment. All offers precede later actions.
// Subjects use turn order; the effect controller chooses order within a group.
// Offers name each subject. Shared transactions keep rechecks and rollback.
// [FromPermanentSet] reads saved present, unphased visits. Return cannot replace
// a visit. Empty results clear older values. Copies, clones and delays keep
// separate frames. Waiting queries make no offers. RON supports fixed life
// payments to AnyPlayer in ordinary resolving sequences. CostWasntPaidForIt
// reads the latest local producer. Branch-local results cannot escape.
//
// [UntilSourcePresent] saves subjects and values at resolution.
// An absent, returned or phased source cannot start the duration.
// Source departure and phasing end it. Phase-in cannot start it again.
// Control changes, untapping and ability removal do not end it.
// Saved subjects, copies and clones keep their exact visits and frames.
// RON uses this duration for GainControlOfPermanentUntil with
// UntilPermanentLeavesBattlefield(ThisPermanent) in resolving permanent bodies.
// Layer and replacement declarations keep their separate duration contracts.
// [UntilSourceLeaves] retains its duration through source phasing.
//
// [PayAnyOrElse] offers eligible players one cost in turn order.
// The first complete payment selects its paid body. No payment selects unpaid.
// Each call keeps its own result and the original effect controller.
// The shared cost transaction keeps failure separate from live resources.
// Saved sources, targets, copies, clones and delays use their existing frames.
// Waiting queries declare source reads without offers or payment.
// RON lowers AnyPlayerMayCost with Opponent or AnyPlayer and fixed PayMana.
// An adjacent If, Unless or IfElse keeps the result local to that pair.
// [PayAnyOrElseWithPayer] saves each candidate during offers and payment.
// The paid body can read that player. The unpaid body has no payer.
// The binding stays local; delayed bodies can save it in their frames.
// RON ActionPlayer reads this payer only on the paid path.
// Other costs, entry and replacement bodies stay separate.
//
// [WhenPermanentsPresent] saves exact matching visits when a state triggers.
// One pending or stacked trigger prevents another detection for that source
// visit and rule. [FromPermanentSet] reads the saved present, unphased visits.
// Later entries and type changes cannot change the saved set. Control changes
// use the current controllers for sacrifice. Return cannot supply an old visit.
// Pending bodies keep their source and controller after departure or phasing.
// Copies, clones and delays keep separate resolution frames.
// RON lowers WhenAnyNumberOfPermanentsAreOnTheBattlefield with nonempty filter
// groups and Other(ThisPermanent). Trigger_ThosePermanents is a plural read
// local to that trigger body. Other event contexts do not supply this binding.
//
// [PreventSpellCasts] prevents each player from casting matching cards.
// [Game.IsSpellCastProhibited] reads active rules without changing state.
// All casting paths and hand queries check before choices or payment.
// Free casts and alternate costs cannot override this rule. Stack copies,
// resolution, land plays and direct entry use their separate rules.
// Departure, phasing, face-down status and ability removal stop the rule.
// Return starts a new visit; clones keep separate source lifetimes.
// RON accepts CantCastSpells with fixed ARN or ATQ name predicates in
// nonempty AnyPlayer rule lists. All list operations must be supported.
//
// [PreventLandPlays] prevents each player from playing matching land cards.
// [Game.IsLandPlayProhibited] reads active source rules without changing state.
// PlayLand and GetPlayableLands use this check. A rejected play keeps hand and
// library order and land play history. Extra play permissions cannot override
// the restriction. Direct battlefield entry does not use this rule. Source
// departure, phasing, face-down status and ability removal stop the rule.
// A return starts a new visit; clones keep separate source lifetimes.
// RON accepts one AnyPlayer CantPlayLands or CantPlayLandsOfType operation.
// The filtered form accepts fixed ARN or ATQ original-printing name predicates.
// Entry prevention stays separate. Supported all-player operations can share a list.
//
// [ChooseUpToPermanentsEach] saves repeated optional choices in one set.
// Each player uses a saved count and chooses in turn order. A choice can
// include another player's permanents. Separate choices can share a visit.
// [PermanentSetBinding] uses the existing result frames; [CreatedTokensBinding]
// retains its compatible API. [FromPermanentSet] reads present saved visits.
// [WithoutPermanents] excludes those visits from a live collection. Phasing
// keeps identity; return cannot inherit a choice. Copies, clones and delays
// keep separate frames. Waiting queries do not make choices.
// RON lowers ChooseUptoNumberPermanentsForEach after a local tap producer.
// EachablePlayer is local to its repetition count. NotChosenThisWay reads
// that plural producer in resolving collections. Other counts stay separate.
//
// [TapAnyNumberEach] offers each player a count and distinct controlled subjects.
// All choices precede one tap batch. Present untapped visits supply actual counts
// in [PlayerNumbersBinding] before tap events. A return cannot supply an old visit.
// Phasing and loss of control prevent a tap. Saved counts keep historical facts.
// Copies, clones and delays use separate frames. Waiting queries make no choices.
// RON accepts AnyPlayer with one conjunctive ActionPlayer controller filter.
// NumberOfPermanentsTappedThisWayByPlayer reads the preceding local producer.
// Plural choices use [ChooseUpToPermanentsEach] with a local saved count.
//
// [ChoosePermanentForPlayer] saves a chooser and an exact permanent visit.
// [PermanentChosenByPlayer] reads the latest preceding choice by that player.
// Empty choices replace older results. Control changes do not change the chooser.
// Phasing prevents actions but keeps the visit. Return cannot replace it.
// Information reads use exact departure facts. Existing resolution frames keep
// delayed bodies, copies and clones separate. Each factory has fresh bindings.
// RON lowers ThePermanentChosenByPlayerThisWay(playerRef) after ChooseAPermanent
// or a direct PlayerAction choice in one resolving action sequence.
// Choices inside branches stay local. TheChosenPermanent keeps its local scope.
// PlayerActions lists and required payment choices need separate adapters.
//
// [ExchangePermanentControl] exchanges current controllers of two typed visits.
// Both subjects must be present and able to change control. Same controllers
// cause no change. Shared layer 2 effects apply both changes in one pass.
// Source departure does not end the control effects. A phase cycle keeps them;
// returned subjects cannot inherit them. Targets keep resolution legality.
// RON lowers ExchangeControl in resolving bodies with singular permanent reads.
// Player-keyed choices and exchange-result conditions need separate adapters.
// [ExchangePermanentControlInto] saves success or failure in a typed result.
// [BindControlExchangeResult] uses the shared resolution frames.
// [IfControlExchanged] reads that result, not later controller changes.
// Failures clear prior success. Copies, clones and delays keep separate frames.
// RON binds the latest local ExchangeControl for If(PermanentsChangedControlThisWay).
// Branch-local results cannot escape. Entry and replacement bodies stay separate.
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
// RON no-land state triggers use [PlayerControlsNone] with a live land filter.
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
//
// Use [Game.SetOutsideGameCards] to supply owned cards permitted by the game.
// Outside cards have no game zone. [Game.OutsideGameCards] returns private copies.
// [PutOutsideGameCardInHand] makes a required choice at resolution and checks
// the saved pool membership before entry. Ante restrictions still apply.
// [OutsideGameEntryAction] uses shared replacement ordering. Entry is not a draw.
// RON accepts PutACardFromOutsideGameInHand(AnyCard) in resolving bodies.
// Native human choices, AI choices and game clones keep the same contract.
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
// [UnlessManaPaidBeforeNextDraw] permits one payment before a player's next
// actual draw step. Payment stops only its saved delayed ability. Failed
// payments keep the permission. Cleanup and skipped steps keep the deadline.
// The draw-step event ends payment before priority; the unpaid body uses the
// shared delayed stack and saved references. Copies and clones stay separate.
// RON accepts CreateFutureTrigger_UnlessPlayerPaysManaBefore with one
// AtTheBeginningOfPlayersNextDrawStep, ActionList, the same player reference,
// and fixed mana in ordinary resolving bodies. Nafs Asp has complete RON
// casting, combat damage, payment and unpaid draw-step gameplay evidence.
//
// Use [PlayerManaPaymentThisTurn] for a repeatable player payment at instant timing.
// The player and source references are saved when the permission starts.
// [Game.GetPlayerPayments] lists affordable permissions without making choices.
// [Game.TakePlayerPayment] uses the shared cost transaction and no stack object.
// Each payment uses a separate copy of the saved resolution frame.
// Source departure and phasing do not end the permission. Cleanup ends it.
// [PriorityTakePlayerPayment] permits use through the priority handler.
// RON supports fixed mana payments and UntilEndOfTurn in resolving bodies.
// Life-to-mana uses [PlayerLifeManaThisTurn]. Other durations and one-use casting
// need separate adapters.
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
// Counter notes use [SnapshotCounters]. Entry options supply return attachments.
// Single-card exile entry still needs a separate adapter.
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
// RON acquisition also accepts bound singular grouped-exile results.
// Separate assignments do not require both cards. Saved visits exclude returns.
// Bronze Tablet uses independent assignments for absent and stolen sources.
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
// Use [CopyPermanent] for resolving copy replacement. Other exceptions reject.
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
// Use [CopyAddTriggeredAbility] to add a rule during the copy process.
// Its pure builder supplies fresh abilities, targets and bindings for each copy.
// Definition checks reject saved game state and shared declarations.
// The recipient supplies the rule source and controller. Later copies keep
// the rule. Ordinary ability removal cannot change these copiable values.
// RON accepts one controller-upkeep TriggerA in each AddAbility copy operation.
// Its body has a fresh permanent scope. Donor bindings cannot enter that scope.
// [CopyPermanent] supplies resolving copy replacement. ThisAbility still rejects.
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
// [CopyPermanent] supplies resolving copying with retained colors.
// Invoking-ability retention still needs a separate service.
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
// other turn-skip replacement declarations need separate adapters.
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
// [PTEqualsCountPlus], [PTEqualsControlledCountPlus] and
// [PTEqualsOpponentCountPlus] add a fixed value to those layer 7a counts.
// Opponent counts use the current source controller, or the card owner
// outside the battlefield. An absent controller supplies zero for the count.
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
// RON accepts fixed nonnegative PayLife(Integer(n)) with a two-branch IfElse.
// Both CostWasPaid and CostWasntPaid use the shared payment transaction.
// Zero life requires acceptance. An exact life total can pay the cost.
// Life-loss replacements keep payment acceptance; damage prevention has no effect.
// Bronze Tablet has payment-fragment evidence. Its absent-source case needs work.
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
// Use [PayRepeatedManaOrElse], [IfRepeatedManaPaid] or [UnlessRepeatedManaPaid]
// for a fixed generic and colored mana cost repeated at resolution.
// The count is read once before the offer. Nonpositive counts require a zero
// payment choice. The shared transaction supplies mana abilities and rollback.
// Activation reductions do not apply. Continuations keep separate payment
// results. Typed counts and both branches declare waiting source references.
// RON accepts PayManaForEach in optional resolving payments with an adjacent
// If, Unless or IfElse. Casting, activation costs and entry need other adapters.
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
// [AfterFirstCombat] requires actual combat completion in this turn.
// [Game.CombatHasEnded] supplies the pure history query. End-of-combat priority
// is still in combat. A skipped phase does not complete combat. Later combats
// keep the result. Turn changes reset it; clones keep separate history.
// RON accepts the strict IsAfterCombat symbol in spell casting restrictions.
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
// [TargetBlockedByThisTurn] requires history with an earlier singular target.
// Selection queries require a complete legal target list. Resolution reads the
// saved blocker visit even after departure or phasing. It reads no current
// blocker characteristics. A returned attacker cannot use its old history.
// RON accepts WasBlockedByDefenderThisTurn(Ref_TargetPermanentN) in a later
// singular permanent target, directly or within And. Other target forms reject.
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
// RON resolving collections accept ExceptFor(collection), including nested
// sets and bound references. [WithoutPermanents] removes exact present visits.
// Membership is read when the action resolves. Delays retain source and target
// identities. A return cannot replace a saved visit. Phased visits are absent.
// New exclusion forms require ordinary resolving bodies. Target filters keep
// their existing forms. Attack eligibility uses saved declaration opportunities.
//
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
// [ControllerOf], [OwnerOf], [OwnerOfWithLKI] and [ControllerOfWithLKI] of [This]
// declare the saved source in payment players and direct life or draw subjects.
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
// Use [RepeatWhilePositive] for complete resolving rounds with local results.
// Each round starts with separate copies of the caller's binding frame.
// A positive value repeats; a nonpositive value ends the process.
// Completion and errors restore the outer bindings. Game actions remain.
// Use [EachPlayerMayPutFromHandOntoBattlefield] for sequential optional entries.
// Seating order starts with the selected player. Each entry reads the current
// hand and uses normal entry rules. Its number binding counts actual entries.
// Failed entries do not count. Each round clears its count before choices.
// RON supports one entry round and its adjacent result-based repeat instruction.
// Queries retain typed source reads without making choices or reading results.
// Other repeated procedure adapters need separate contracts.
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
// Use [CantBeEnchanted] with [Static] to exclude all Auras from a subject.
// Token definitions use [TokenAbilities] to keep this rule as copiable text.
// RON accepts PermanentRuleEffect(ThisPermanent, [CantBeEnchanted]) in token
// definitions. Each token supplies its own source for the restriction.
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
// [PlayerCollectionDamageInstruction] retains that local amount declaration.
// [InspectRecipientPermanentCount] checks a native count with its exact member.
// Its pure read uses a present recipient without changing resolution bindings.
// Native spell inspection accepts these counts for source player damage.
// Legal nonempty damage remains a barrier before later destruction proofs.
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
// [CantAttackDuringNextTurn] saves one subject and one player at resolution.
// [BeginActiveTurn] starts the rule before that player's next actual turn actions.
// Cleanup ends the rule. Skipped turns do not start it; extra turns do.
// Source loss cannot end it. Control changes cannot select a different player.
// Return cannot supply the saved subject. Phasing keeps the installed rule.
// Copies and clones keep separate saved visits and windows.
// RON accepts one CantAttack in CreatePermanentRuleEffectUntil with a typed
// permanent subject and DuringPlayersNextTurn with a typed player reference.
// Other operations and duration contexts need separate adapters.
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
// per-recipient damage service. [DiscardUpToInstruction] retains the producer.
// [InspectRecipientDiscardAmount] retains fixed subtraction from the exact
// recipient's saved result. Spell inspection checks producer dominance without
// reading counts or making choices. Nonempty discard groups block later proofs.
// Coin rounds use [RepeatHeadsOrTailsDamage].
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
// Use [AcquireStaticAbility] for one captured layer 6 static ability or
// one [PreventUntapDuringStep] rule.
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
// [WhenDiesThisTurnWithCard] also saves the first graveyard visit at death.
// The card binding belongs only to that delayed body. A later graveyard visit
// cannot replace it. Tokens supply no card. Shared card-zone replacements
// apply before exile. Zone events observe the final destination after movement.
// Copies and clones keep separate captured card maps. RON accepts singular
// exile of the watched reference or Trigger_ThatPermanent in that death body.
// Trigger_ThatGraveyardCard supplies an explicit typed card reference there.
// Complete Whippoorwill gameplay covers both actions, paid casting and activation.
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
// [RequiredTapInstruction.Effect] retains a mandatory tap and separate branches.
// [InspectRequiredTapInstruction] reads native operands without cost payment.
// Spell destruction inspection accepts You, one native permanent target and
// complete mana production branches. Both branches validate before target gates.
// Native mana amounts can read the exact target mana value, fixed or announced-X
// values, saved casting values and native addition or multiplication trees.
// Queries validate target frames without reading values or predicting payment.
// They do not tap, produce mana, run replacements or change bindings.
// Payment and production prefixes block later state proofs. Supported suffixes
// keep earlier proofs. Destruction branches and other payers remain unknown.
// Execution retains the shared transaction, replaced payments and LKI reads.
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
// Use [ChoosePermanentsAPNAP] for grouped required permanent choices.
// Each [PermanentChoiceRequest] has a player, collection and separate result.
// Choosers are saved first. Choices use turn order, with declaration order for
// one player. Later players can see prior choices. Old results are cleared.
// The group makes no zone or control changes. Existing bindings keep visits,
// departure LKI, copies, clones and delays. Definition checks reject output
// reads within the group. Waiting queries keep player and collection sources.
// RON lowers explicit SimultaneousPermanentChoices in resolving sequences.
// The group exports keyed history but does not change TheChosenPermanent.
// Costs, static rules and arbitrary grouped actions need separate adapters.
//
// Use [SacrificeAllButEach] for required permanent choices in turn order.
// Each player chooses the subjects to keep. Later players see earlier choices.
// Counts and members are read before choices. The rest use shared simultaneous
// sacrifice replacements, owner destinations and departure information.
// [LeastControlledPermanentCount] counts present matching visits for each player.
// Empty player sets supply zero. Duplicate, phased and old visits cannot count.
// Copies and clones read their own battlefield at resolution. No result is saved.
// RON lowers EachPlayerAction(AnyPlayer, SacrificeAllPermanentsExceptForNum)
// with a fixed nonnegative count or an all-player least permanent count.
// Optional selection results and other RON player sets stay separate.
//
// Use [DiscardAllButEach] for required hand choices in turn order.
// The kept count is read once. All hand visits are saved before choices.
// Each player chooses cards to discard without revealing their names.
// Shared replacements and graveyard ordering precede one discard batch.
// Returned visits cannot move. Cancellation does not cause another choice.
// [LeastHandSize] includes empty hands and excludes absent or duplicate players.
// Empty player sets supply zero. Copies, clones and delays read current hands.
// RON lowers EachPlayerAction(AnyPlayer, DiscardAllButNumberCards) with a
// fixed nonnegative count or TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer).
// This count spelling is local to this resolving action. No result is saved.
//
// [CombatOpponentsWithLKI] reads an exact visit's combat opponents.
// Present sources use current combat. Departed sources use saved relations.
// Only present, unphased creatures of those opponent visits are selected.
// Return cannot replace a saved source or opponent. Clones keep separate state.
// RON accepts the pair BlockedDeadAttacker and WasBlockingDeadDefender in Or
// in source-death resolving collections. The event binds the exact dying visit.
// Shared simultaneous destruction retains replacements and forbids regeneration.
// The complete Abu Ja'far record has paid-cast and combat-death evidence.
//
// [CastingRequirement] checks an intrinsic condition before a spell is cast.
// It uses the shared casting checks without an instruction at resolution.
// Copies that are cast repeat the check. Stack copies and resolution do not.
// RON accepts CastEffect(CantBeCastUnless(NoPermanentsPassFilter(filter,
// OnTheBattlefield))) on nonland permanents with a reference-free filter.
// [BattlefieldHasNone] reads all present permanents, including tokens.
// Phased permanents are absent. Cast rejection preserves payment resources.
// The complete Tidal Influence record has casting and counter-cycle evidence.
//
// [CardPaymentFact] saves a filter result from one discard payment.
// [DiscardCardCostWithFacts] uses shared payment and discard services.
// Public destinations retain the match. An unrevealed hidden destination
// supplies no match. Failed payment publishes no result. Later card movement
// does not change the fact. Copies, clones and delays keep separate numbers.
// RON accepts DiscardedCardPassesFilter in an activated resolving body with
// exactly one DiscardACard cost. Complete Land's Edge has gameplay evidence.
//
// [CastSpellDuringLastTurn] and [PutNontokenPermanentDuringLastTurn] read
// the typed player's last completed turn. Final cleanup saves the active
// player's actions. Other turns and skipped turns keep that saved history.
// Entry history uses the actor, even with a different entering controller.
// Tokens and failed entries do not count. Later departure does not erase it.
// Clones and payment trials keep separate current and saved maps.
// [CantAttackEachIf] uses the shared live collection restriction at declaration.
// RON accepts CantAttackAPlayer(ExceptFor(test)) in EachPermanentRuleEffect
// on static permanents. The test is either supported last-turn activity or
// a nonempty Or list. Complete Arboria has paid-cast and attack evidence.
//
// [Game.InspectSpellDestruction] accepts saved sacrifice mana values in
// fixed-type mana production. The exact native additional cost must produce
// the captured number before resolution. Native sums and products retain
// their operands. Missing, foreign or overwritten bindings remain unknown.
// The query uses the current printed spell definition after a text revision.
// It does not pay costs, resolve values or add mana. Required tap branches,
// restricted production and delayed production need separate query contracts.
//
// [GraveyardHandInstruction] retains one graveyard-to-hand return. Execution
// uses shared graveyard movement and the card owner's hand. RON retains native
// card-type filters for singular graveyard targets. [InspectGraveyardTarget]
// checks the exact announced visit and current own or any graveyard scope.
// [Game.InspectSpellDestruction] validates these declarations before legality
// gates or earlier proofs. A legal return prefix blocks later state proofs.
// A supported suffix preserves an earlier proof. No filter callback, movement,
// replacement or LKI read runs during inspection. Optional and numeric targets,
// event cards and battlefield returns need separate query contracts.
//
// [CombatSourcePreventionInstruction] retains all combat prevention from one
// typed source until end of turn. [InspectCombatSourcePreventionInstruction]
// reads the complete singular declaration without installing a replacement.
// Execution captures the source visit and uses the shared replacement service.
// Spell inspection validates the exact native target before legality gates.
// A live prefix blocks later proofs; a supported suffix preserves earlier proofs.
// Source sets, other durations and recipient prevention have separate contracts.
//
// [CreatureDeathCountInstruction] retains a creature-death count from one
// simultaneous destruction. [InspectCreatureDeathCountInstruction] checks one
// exact native producer without reading a count. Execution uses shared creature
// last known information, replacement ordering and graveyard movement.
// [Game.InspectSpellDestruction] accepts saved number sums in singular source
// damage. Each saved number must have an exact preceding native producer.
// Forward, foreign and duplicate declarations remain unknown. Inspection reads
// no saved result and calls no amount operand. A supported damage suffix keeps
// an earlier destruction proof. Complete Hellfire has RON gameplay evidence.
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

// Native equal-division damage
//
// [EvenDamageInstruction] retains a source, recipient set and resolution total.
// [InspectEvenDamageInstructions] reads complete native trees without callbacks.
// Missing operands, result bindings and unknown children reject inspection.
// Execution uses [DealDamageEvenlyToRecipients] and shared simultaneous damage.
// [InspectDamageTargets] reads native fixed or optional damage target groups.
// Its query checks exact visits, current legality and declared empty groups.
// X bounds and chooser wrappers need separate query contracts.
// [InspectSpellDestruction] accepts native fixed/X equal-division actions with
// supported collections or legal target frames. Damage can cause state changes,
// so a nonempty damage prefix blocks a later destruction proof. This query does
// not divide amounts, deal damage, run replacements or predict state changes.
