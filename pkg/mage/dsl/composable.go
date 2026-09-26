package dsl

import "github.com/benprew/mage-go/pkg/mage"

type NumberBinding = mage.NumberBinding

var (
	CountPermanents               = mage.CountPermanents
	CreaturesDiedThisTurn         = mage.CreaturesDiedThisTurn
	CountersOn                    = mage.CountersOn
	SourceCounters                = mage.SourceCounters
	AddManaAmount                 = mage.AddManaAmount
	AddManaOfAnyOneColor          = mage.AddManaOfAnyOneColor
	SnapshotNumber                = mage.SnapshotNumber
	CaptureNumberBeforePayment    = mage.CaptureNumberBeforePayment
	CapturePermanentBeforePayment = mage.CapturePermanentBeforePayment
	BindNumber                    = mage.BindNumber
	FromNumber                    = mage.FromNumber
)

type CoinBranch = mage.CoinBranch
type PermanentTargetOption = mage.PermanentTargetOption
type PlayerTargetOption = mage.PlayerTargetOption
type GraveyardTargetOption = mage.GraveyardTargetOption
type TokenDefinition = mage.TokenDefinition
type TokenOption = mage.TokenOption

var (
	FlipCoin             = mage.FlipCoin
	DefineToken          = mage.DefineToken
	CreateTokenCopies    = mage.CreateTokenCopies
	TokenTypes           = mage.TokenTypes
	TokenSubtypes        = mage.TokenSubtypes
	TokenColors          = mage.TokenColors
	TokenPT              = mage.TokenPT
	TokenKeywords        = mage.TokenKeywords
	TokenAbilities       = mage.TokenAbilities
	OnWin                = mage.OnWin
	OnLoss               = mage.OnLoss
	OnlyAsSorcery        = mage.OnlyAsSorcery
	OnlyDuring           = mage.OnlyDuring
	OnlyOnYourTurn       = mage.OnlyOnYourTurn
	ActivationLimit      = mage.ActivationLimit
	ActivationIf         = mage.ActivationIf
	AnyPlayerMayActivate = mage.AnyPlayerMayActivate
	ControllerOf         = mage.ControllerOf
	ControllerOfSpell    = mage.ControllerOfSpell
	UnlessPaid           = mage.UnlessPaid
	IfPaid               = mage.IfPaid
	ForEach              = mage.ForEach
)

// Typed subjects and declarations keep choices in the resolving instance.
type (
	PlayerRef                 = mage.PlayerRef
	StackObjectRef            = mage.StackObjectRef
	PlayerSet                 = mage.PlayerSet
	PermanentSet              = mage.PermanentSet
	TargetBinding             = mage.TargetBinding
	PermanentTargetBinding    = mage.PermanentTargetBinding
	PlayerTargetBinding       = mage.PlayerTargetBinding
	CardRef                   = mage.CardRef
	CardBinding               = mage.CardBinding
	CardTargetBinding         = mage.CardTargetBinding
	SpellOption               = mage.SpellOption
	ActivationOption          = mage.ActivationOption
	ActivationCaptureOption   = mage.ActivationCaptureOption
	ActivationTimingOption    = mage.ActivationTimingOption
	StepTimingOption          = mage.StepTimingOption
	ActivationConditionOption = mage.ActivationConditionOption
	ActivationLimitOption     = mage.ActivationLimitOption
	ModalActivationOption     = mage.ModalActivationOption
	GrantedActivationOption   = mage.GrantedActivationOption
	CostGroup                 = mage.CostGroup
)

var (
	// TargetPermanent declares a single permanent target matching the filter.
	TargetPermanent            = mage.DeclarePermanentTarget
	TargetYouControl           = mage.TargetYouControl
	TargetYouOwn               = mage.TargetYouOwn
	TargetAttachedToYouControl = mage.TargetAttachedToYouControl
	TargetOpponentControls     = mage.TargetOpponentControls
	TargetChosenByOpponent     = mage.TargetChosenByOpponent
	MayTapOrUntap              = mage.MayTapOrUntap
	PreventCombatDamageToAndBy = mage.PreventCombatDamageToAndBy
	TargetPlayer               = mage.DeclarePlayerTarget
	TargetSpell                = mage.DeclareSpellTarget
	TargetGraveyardCard        = mage.DeclareGraveyardCardTarget
	InAnyGraveyard             = mage.InAnyGraveyard
	GraveyardCardFilter        = mage.GraveyardCardFilter
	ReturnCardToHand           = mage.ReturnCardToHand
	DiscardCard                = mage.DiscardCard
	RevealCard                 = mage.RevealCard
	IfCardHasType              = mage.IfCardHasType
	IfCardLacksType            = mage.IfCardLacksType
	BindCard                   = mage.BindCard
	FromCard                   = mage.FromCard
	ReturnCardToBattlefield    = mage.ReturnCardToBattlefield
	Counter                    = mage.Counter
	TargetOpponentOnly         = mage.TargetOpponentOnly
	// WithTargets binds declared target bindings to a spell, activation, or trigger.
	WithTargets = mage.WithTargets
	// WithTargetGroups declares independent spell target groups.
	WithTargetGroups = mage.WithTargetGroups
	// WithActivationTargetGroups declares independent activation target groups.
	WithActivationTargetGroups = mage.WithActivationTargetGroups
	// Spell constructs the resolving ability of an instant or sorcery card.
	Spell = mage.Spell
	// Activate constructs an activated ability from costs, an effect body, and optional restrictions.
	Activate = mage.Activate
	// Costs groups cost components (mana, tap, sacrifice, life, counters) for an activation.
	Costs = mage.Costs
	Mana  = mage.ManaPayment
	// You refers to the stored resolving controller of the spell or ability.
	You                              = mage.You
	ActivePlayer                     = mage.ActivePlayer
	EachPlayer                       = mage.EachPlayer
	OpponentsOf                      = mage.OpponentsOf
	GrantLandPlays                   = mage.GrantLandPlays
	LoseLifeEach                     = mage.LoseLifeEach
	MatchingPermanents               = mage.MatchingPermanents
	ManaValueAtMost                  = mage.ManaValueAtMost
	WithoutPermanent                 = mage.WithoutPermanent
	ActivatedAbilityModification     = mage.ActivatedAbilityModification
	ActivatedModesModification       = mage.ActivatedModesModification
	EachActivatedAbilityModification = mage.EachActivatedAbilityModification
	EachActivatedModesModification   = mage.EachActivatedModesModification
	GrantEachActivatedAbility        = mage.GrantEachActivatedAbility
	GrantActivatedAbility            = mage.GrantActivatedAbility
	GrantActivatedModes              = mage.GrantActivatedModes
	GrantEachActivatedModes          = mage.GrantEachActivatedModes
	ControlledPermanents             = mage.ControlledPermanents
	// Regenerate installs a shield on the referenced battlefield incarnation.
	// Use Regenerate(AttachedPermanent) in an Aura-owned Activate definition.
	// The Aura controller pays; the instruction does not target the creature.
	// Use Activate(Costs(RemoveCountersCost(counter, count)), Regenerate(This))
	// for an activation that removes counters from its source.
	// A subtype-only target can also select a noncreature permanent.
	Regenerate = mage.Regenerate
	// Destroy destroys the targeted or referenced permanent (permits regeneration).
	Destroy = mage.Destroy
	// DestroyEach destroys all permanents in a PermanentSet simultaneously.
	DestroyEach = mage.DestroyEach
	// DestroyEachWithoutRegeneration destroys the set without regeneration.
	DestroyEachWithoutRegeneration = mage.DestroyEachWithoutRegeneration
	// DestroyWithoutRegeneration destroys the permanent and forbids regeneration.
	DestroyWithoutRegeneration = mage.DestroyWithoutRegeneration
	// Exile moves the referenced permanent to exile.
	Exile = mage.Exile
	// ReturnToHand bounces the referenced permanent to its owner's hand.
	ReturnToHand = mage.ReturnToHand
	// Untap untaps the referenced permanent incarnation.
	Untap = mage.UntapPermanent
	// SkipNextUntap prevents the permanent from untapping during its controller's next untap step.
	SkipNextUntap = mage.SkipNextUntap
	// SkipNextUntapOf prevents untapping during the specified player's next untap step.
	SkipNextUntapOf = mage.SkipNextUntapOf
	// Draw draws cards for the specified player.
	Draw           = mage.DrawFor
	DrawEach       = mage.DrawEach
	DrawAmountInto = mage.DrawAmountInto
	DrawCardInto   = mage.DrawCardInto
	Discard        = mage.Discard
	// DiscardAtRandom uses a typed player and amount. Ring of Renewal uses You and Fixed(1).
	DiscardAtRandom   = mage.DiscardAtRandom
	DiscardEntireHand = mage.DiscardEntireHand
	// Mill moves cards from the top of the library to the graveyard.
	Mill = mage.Mill
)

type (
	EventPattern           = mage.EventPattern
	TurnStepPattern        = mage.TurnStepPattern
	DelayedStepPattern     = mage.DelayedStepPattern
	DelayedStepOption      = mage.DelayedStepOption
	ModalDelayedStepOption = mage.ModalDelayedStepOption
	TurnStepEventPattern   = mage.TurnStepEventPattern
	TriggerOption          = mage.TriggerOption
)

var (
	WhenEventModes          = mage.WhenEventModes
	AtModes                 = mage.AtModes
	DealsDamage             = mage.DealsDamage
	ControllerIsDealtDamage = mage.ControllerIsDealtDamage
	ReceivesDamage          = mage.ReceivesDamage
	DealsDamageToPlayer     = mage.DealsDamageToPlayer
	DealsDamageToOpponent   = mage.DealsDamageToOpponent
	OpponentLosesLife       = mage.OpponentLosesLife
	ControllerDrawsCard     = mage.ControllerDrawsCard
	OpponentDrawsCard       = mage.OpponentDrawsCard
	OpponentDiscardsCard    = mage.OpponentDiscardsCard
	ControllerGainsLife     = mage.ControllerGainsLife
	// WhenEvent constructs an event-triggered ability that responds to game events.
	WhenEvent                          = mage.WhenEvent
	Enters                             = mage.Enters
	Dies                               = mage.Dies
	LeavesBattlefield                  = mage.LeavesBattlefield
	SpellCast                          = mage.SpellCast
	Attacks                            = mage.Attacks
	OneOrMoreCreaturesYouControlAttack = mage.OneOrMoreCreaturesYouControlAttack
	EachCreatureYouControlAttacks      = mage.EachCreatureYouControlAttacks
	AttacksUnblocked                   = mage.AttacksUnblocked
	Blocks                             = mage.Blocks
	BecomesBlocked                     = mage.BecomesBlocked
	BecomesTargetOfSpell               = mage.BecomesTargetOfSpell
	BecomesBlockedBy                   = mage.BecomesBlockedBy
	BlocksOrBecomesBlockedBy           = mage.BlocksOrBecomesBlockedBy
	CreatureYouControlDies             = mage.CreatureYouControlDies
	OpponentCreatureDies               = mage.OpponentCreatureDies
	AnotherCreatureYouControlDies      = mage.AnotherCreatureYouControlDies
	AnotherCreatureYouControlEnters    = mage.AnotherCreatureYouControlEnters
	LandEnters                         = mage.LandEnters
	AnotherCreatureDies                = mage.AnotherCreatureDies
	CreatureYouSacrifice               = mage.CreatureYouSacrifice
	AttachedCreatureDies               = mage.AttachedCreatureDies
	PermanentDies                      = mage.PermanentDies
	CreatureDies                       = mage.CreatureDies
	// AtNext creates a one-shot delayed trigger that executes at a future turn step.
	AtNext                                 = mage.AtNext
	AtNextModes                            = mage.AtNextModes
	AttachedControllerUpkeep               = mage.AttachedControllerUpkeep
	AttachedBecomesTapped                  = mage.AttachedBecomesTapped
	OpponentLandBecomesTapped              = mage.OpponentLandBecomesTapped
	ArtifactBecomesTapped                  = mage.ArtifactBecomesTapped
	ArtifactAbilityActivatedWithoutTapCost = mage.ArtifactAbilityActivatedWithoutTapCost
	BecomesTapped                          = mage.BecomesTapped
	TurnStepBegins                         = mage.TurnStepBegins
	EachEndStep                            = mage.EachEndStep
	YourEndStep                            = mage.YourEndStep
	EachEndOfCombat                        = mage.EachEndOfCombat
	// At constructs a recurring turn-step triggered ability (e.g. YourUpkeep, EachEndStep).
	At                      = mage.At
	YourFirstMainPhase      = mage.YourFirstMainPhase
	YourPostcombatMainPhase = mage.YourPostcombatMainPhase
	YourUpkeep              = mage.YourUpkeep
	EachUpkeep              = mage.EachUpkeep
	EachDrawStep            = mage.EachDrawStep
	WithInterveningIf       = mage.WithInterveningIf
)

// Continuous modifications and their installation lifecycles are distinct roles.
type (
	ContinuousModification = mage.ContinuousModification
	StaticRule             = mage.StaticRule
)

var (
	SetColors             = mage.SetColors
	SetEachColors         = mage.SetEachColors
	SetSubtypeFamily      = mage.SetSubtypeFamily
	AddSubtypeFamily      = mage.AddSubtypeFamily
	GrantEachKeyword      = mage.GrantEachKeyword
	SetBasePowerToughness = mage.SetBasePowerToughness
	SetBasePower          = mage.SetBasePower
	TargetOtherThanSource = mage.TargetOtherThanSource
	// ModifyPT also supports source bonuses paid with RemoveCountersCost.
	// Install the bonus with Until to set its duration.
	ModifyPT         = mage.ModifyPT
	ModifyEachPT     = mage.ModifyEachPT
	ChangeController = mage.ChangeController
	// GrantKeyword accepts a typed subject; Until supplies the duration.
	// Use it in Spell or Activate to grant flying to a creature target.
	GrantKeyword        = mage.GrantKeywordModification
	RemoveKeyword       = mage.RemoveKeyword
	RemoveLandwalk      = mage.RemoveLandwalk
	RemoveAbilities     = mage.RemoveAbilities
	RemoveAbilitiesEach = mage.RemoveAbilitiesEach
	CantBeBlocked       = mage.CantBeBlocked
	// Until installs temporary continuous modifications that expire at duration end (e.g. EndOfTurn).
	Until = mage.Until
	// UntilSourceLeaves installs continuous modifications that persist until the source permanent leaves the battlefield.
	UntilSourceLeaves = mage.UntilSourceLeaves
	// While constructs a static continuous effect active as long as the pure StateCondition holds.
	While = mage.While
	// Static installs static continuous effects and rules on a permanent.
	Static = mage.Static
	// AddCounters reads a typed amount before the counter action.
	AddCounters                     = mage.AddCountersTo
	RemoveAllCounters               = mage.RemoveAllCounters
	RemoveCountersFrom              = mage.RemoveCountersFrom
	SourceHasCounters               = mage.SourceHasCounters
	SourceTappedCondition           = mage.SourceTappedCondition
	PermanentIsTapped               = mage.PermanentIsTapped
	PermanentIsAttacking            = mage.PermanentIsAttacking
	PermanentIsBlocking             = mage.PermanentIsBlocking
	PermanentIsUntapped             = mage.PermanentIsUntapped
	SourceUntappedCondition         = mage.SourceUntappedCondition
	SourceBlockingCondition         = mage.SourceBlockingCondition
	SourceAttackedThisTurnCondition = mage.SourceAttackedThisTurnCondition
	// Life amounts use ResolutionValue. Adapt legacy values with ValueAtResolution.
	GainLife = mage.GainLifeFor
	LoseLife = mage.LoseLifeFor
)

type (
	DamageSourceRef        = mage.DamageSourceRef
	DamageRecipientRef     = mage.DamageRecipientRef
	DamageRecipientSet     = mage.DamageRecipientSet
	DamageTargetBinding    = mage.DamageTargetBinding
	DamageTargetsBinding   = mage.DamageTargetsBinding
	DamageResultOption     = mage.DamageResultOption
	SacrificeResultOption  = mage.SacrificeResultOption
	TapResultOption        = mage.TapResultOption
	DamageEachResultOption = mage.DamageEachResultOption
	DamageSourceBinding    = mage.DamageSourceBinding
	DamageResultsBinding   = mage.DamageResultsBinding
	DamageResultBinding    = mage.DamageResultBinding
)

var (
	Tap                    = mage.TapSubject
	Fight                  = mage.Fight
	TapCost                = mage.TapCost
	TargetDamageRecipient  = mage.DeclareDamageTarget
	TargetDamageRecipients = mage.DeclareDamageTargets
	DealDamageFrom         = mage.DealDamageFrom
	DamageSourceOf         = mage.DamageSourceOf
	DamageRecipients       = mage.DamageRecipients
	DealDamageEach         = mage.DealDamageEach
	DamageTotalInto        = mage.DamageTotalInto
	SacrificeAmountInto    = mage.SacrificeAmountInto
	TapAmountInto          = mage.TapAmountInto
	IfPositive             = mage.IfPositive
	BindDamageResults      = mage.BindDamageResults
	BindDamageResult       = mage.BindDamageResult
	DamageResultsInto      = mage.DamageResultsInto
	ForEachDamageResult    = mage.ForEachDamageResult
	DamageResultAmount     = mage.DamageResultAmount
	DamageResultPlayer     = mage.DamageResultPlayer
	DamageResultPermanent  = mage.DamageResultPermanent
	DamageResultSource     = mage.DamageResultSource
	DamageAmountInto       = mage.DamageAmountInto
	DamagePlayerInto       = mage.DamagePlayerInto
	DamagePermanentInto    = mage.DamagePermanentInto
	DamageSourceInto       = mage.DamageSourceInto
	BindDamageSource       = mage.BindDamageSource
	FromDamageSource       = mage.FromDamageSource
	PreventNextDamage      = mage.PreventNextDamage
)

var (
	ChoosePermanent       = mage.ChoosePermanent
	ChooseRandomPermanent = mage.ChooseRandomPermanent
	LowestPower           = mage.LowestPower
)

type (
	PermanentTargetsBinding = mage.PermanentTargetsBinding
	ModeDefinition          = mage.ModeDefinition
	ModeOption              = mage.ModeOption
)

var (
	TargetPermanents = mage.DeclarePermanentTargets
	ModeOf           = mage.ModeOf
	SpellModes       = mage.SpellModes
	ActivateModes    = mage.ActivateModes
)

// ResolutionValue retains typed bindings during value evaluation.
type ResolutionValue = mage.ResolutionValue

var (
	ManaValueOf       = mage.ManaValueOf
	ManaValueOfSpell  = mage.ManaValueOfSpell
	PowerOf           = mage.PowerOf
	ToughnessOf       = mage.ToughnessOf
	ValueAtResolution = mage.ValueAtResolution
	MultiplyValues    = mage.MultiplyValues
	SubtractValues    = mage.SubtractValues
	MinValue          = mage.MinValue
	MaxValue          = mage.MaxValue
)

type PlayerBinding = mage.PlayerBinding

var (
	SnapshotPlayer     = mage.SnapshotPlayer
	BindPlayer         = mage.BindPlayer
	FromPlayer         = mage.FromPlayer
	ChooseRandomPlayer = mage.ChooseRandomPlayer
)

// SkipDrawStepForLife creates an optional draw-step replacement for Static.
// The gain occurs first in the next step that occurs, even if the source leaves.
var SkipDrawStepForLife = mage.SkipDrawStepForLife
