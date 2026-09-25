package dsl

import "github.com/benprew/mage-go/pkg/mage"

type NumberBinding = mage.NumberBinding

var (
	CreaturesDiedThisTurn         = mage.CreaturesDiedThisTurn
	CountersOn                    = mage.CountersOn
	SourceCounters                = mage.SourceCounters
	AddManaAmount                 = mage.AddManaAmount
	SnapshotNumber                = mage.SnapshotNumber
	CaptureNumberBeforePayment    = mage.CaptureNumberBeforePayment
	CapturePermanentBeforePayment = mage.CapturePermanentBeforePayment
	BindNumber                    = mage.BindNumber
	FromNumber                    = mage.FromNumber
)

type CoinBranch = mage.CoinBranch
type PermanentTargetOption = mage.PermanentTargetOption
type PlayerTargetOption = mage.PlayerTargetOption

var (
	FlipCoin        = mage.FlipCoin
	OnWin           = mage.OnWin
	OnLoss          = mage.OnLoss
	OnlyAsSorcery   = mage.OnlyAsSorcery
	OnlyDuring      = mage.OnlyDuring
	OnlyOnYourTurn  = mage.OnlyOnYourTurn
	ActivationLimit = mage.ActivationLimit
	ActivationIf    = mage.ActivationIf
	ControllerOf    = mage.ControllerOf
	UnlessPaid      = mage.UnlessPaid
	IfPaid          = mage.IfPaid
	ForEach         = mage.ForEach
)

// Typed subjects and declarations keep choices in the resolving instance.
type (
	PlayerRef                 = mage.PlayerRef
	PlayerSet                 = mage.PlayerSet
	PermanentSet              = mage.PermanentSet
	TargetBinding             = mage.TargetBinding
	PermanentTargetBinding    = mage.PermanentTargetBinding
	PlayerTargetBinding       = mage.PlayerTargetBinding
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
	TargetOpponentControls     = mage.TargetOpponentControls
	TargetChosenByOpponent     = mage.TargetChosenByOpponent
	MayTapOrUntap              = mage.MayTapOrUntap
	PreventCombatDamageToAndBy = mage.PreventCombatDamageToAndBy
	TargetPlayer               = mage.DeclarePlayerTarget
	TargetOpponentOnly         = mage.TargetOpponentOnly
	// WithTargets binds declared target bindings to a spell, activation, or trigger.
	WithTargets = mage.WithTargets
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
	Draw     = mage.DrawFor
	DrawEach = mage.DrawEach
	Discard  = mage.Discard
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
	WhenEventModes        = mage.WhenEventModes
	AtModes               = mage.AtModes
	DealsDamage           = mage.DealsDamage
	DealsDamageToPlayer   = mage.DealsDamageToPlayer
	DealsDamageToOpponent = mage.DealsDamageToOpponent
	OpponentLosesLife     = mage.OpponentLosesLife
	ControllerDrawsCard   = mage.ControllerDrawsCard
	// WhenEvent constructs an event-triggered ability that responds to game events.
	WhenEvent              = mage.WhenEvent
	Enters                 = mage.Enters
	Dies                   = mage.Dies
	CreatureYouControlDies = mage.CreatureYouControlDies
	CreatureDies           = mage.CreatureDies
	// AtNext creates a one-shot delayed trigger that executes at a future turn step.
	AtNext                   = mage.AtNext
	AtNextModes              = mage.AtNextModes
	AttachedControllerUpkeep = mage.AttachedControllerUpkeep
	AttachedBecomesTapped    = mage.AttachedBecomesTapped
	TurnStepBegins           = mage.TurnStepBegins
	EachEndStep              = mage.EachEndStep
	YourEndStep              = mage.YourEndStep
	EachEndOfCombat          = mage.EachEndOfCombat
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
	ModifyPT              = mage.ModifyPT
	ModifyEachPT          = mage.ModifyEachPT
	GrantKeyword          = mage.GrantKeywordModification
	CantBeBlocked         = mage.CantBeBlocked
	// Until installs temporary continuous modifications that expire at duration end (e.g. EndOfTurn).
	Until = mage.Until
	// UntilSourceLeaves installs continuous modifications that persist until the source permanent leaves the battlefield.
	UntilSourceLeaves = mage.UntilSourceLeaves
	// While constructs a static continuous effect active as long as the pure StateCondition holds.
	While = mage.While
	// Static installs static continuous effects and rules on a permanent.
	Static = mage.Static
	// AddCounters reads a typed amount before the counter action.
	AddCounters        = mage.AddCountersTo
	RemoveAllCounters  = mage.RemoveAllCounters
	RemoveCountersFrom = mage.RemoveCountersFrom
	SourceHasCounters  = mage.SourceHasCounters
	// Life amounts use ResolutionValue. Adapt legacy values with ValueAtResolution.
	GainLife = mage.GainLifeFor
	LoseLife = mage.LoseLifeFor
)

type (
	DamageRecipientRef  = mage.DamageRecipientRef
	DamageTargetBinding = mage.DamageTargetBinding
)

var (
	Tap                   = mage.TapSubject
	TapCost               = mage.TapCost
	TargetDamageRecipient = mage.DeclareDamageTarget
	PreventNextDamage     = mage.PreventNextDamage
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
	PowerOf           = mage.PowerOf
	ToughnessOf       = mage.ToughnessOf
	ValueAtResolution = mage.ValueAtResolution
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
