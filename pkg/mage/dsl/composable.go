package dsl

import "github.com/benprew/mage-go/pkg/mage"

type CoinBranch = mage.CoinBranch
type PermanentTargetOption = mage.PermanentTargetOption

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
	PlayerRef              = mage.PlayerRef
	PlayerSet              = mage.PlayerSet
	PermanentSet           = mage.PermanentSet
	TargetBinding          = mage.TargetBinding
	PermanentTargetBinding = mage.PermanentTargetBinding
	PlayerTargetBinding    = mage.PlayerTargetBinding
	SpellOption            = mage.SpellOption
	ActivationOption       = mage.ActivationOption
	CostGroup              = mage.CostGroup
)

var (
	TargetPermanent            = mage.DeclarePermanentTarget
	TargetYouControl           = mage.TargetYouControl
	PreventCombatDamageToAndBy = mage.PreventCombatDamageToAndBy
	TargetPlayer               = mage.DeclarePlayerTarget
	WithTargets                = mage.WithTargets
	Spell                      = mage.Spell
	Activate                   = mage.Activate
	Costs                      = mage.Costs
	Mana                       = mage.ManaPayment
	You                        = mage.You
	ActivePlayer               = mage.ActivePlayer
	EachPlayer                 = mage.EachPlayer
	MatchingPermanents         = mage.MatchingPermanents
	ControlledPermanents       = mage.ControlledPermanents
	Destroy                    = mage.Destroy
	DestroyEach                = mage.DestroyEach
	DestroyWithoutRegeneration = mage.DestroyWithoutRegeneration
	Exile                      = mage.Exile
	Untap                      = mage.UntapPermanent
	Draw                       = mage.DrawFor
	DrawEach                   = mage.DrawEach
	Discard                    = mage.Discard
)

type (
	EventPattern    = mage.EventPattern
	TurnStepPattern = mage.TurnStepPattern
	TriggerOption   = mage.TriggerOption
)

var (
	WhenEvent         = mage.WhenEvent
	Enters            = mage.Enters
	Dies              = mage.Dies
	AtNext            = mage.AtNext
	EachEndStep       = mage.EachEndStep
	YourEndStep       = mage.YourEndStep
	EachEndOfCombat   = mage.EachEndOfCombat
	At                = mage.At
	YourUpkeep        = mage.YourUpkeep
	EachUpkeep        = mage.EachUpkeep
	EachDrawStep      = mage.EachDrawStep
	WithInterveningIf = mage.WithInterveningIf
)

// Continuous modifications and their installation lifecycles are distinct roles.
type (
	ContinuousModification = mage.ContinuousModification
	StaticRule             = mage.StaticRule
)

var (
	SetColors         = mage.SetColors
	SetEachColors     = mage.SetEachColors
	SetSubtypeFamily  = mage.SetSubtypeFamily
	AddSubtypeFamily  = mage.AddSubtypeFamily
	GrantEachKeyword  = mage.GrantEachKeyword
	ModifyPT          = mage.ModifyPT
	ModifyEachPT      = mage.ModifyEachPT
	GrantKeyword      = mage.GrantKeywordModification
	Until             = mage.Until
	UntilSourceLeaves = mage.UntilSourceLeaves
	While             = mage.While
	Static            = mage.Static
	AddCounters       = mage.AddCountersTo
	GainLife          = mage.GainLifeFor
	LoseLife          = mage.LoseLifeFor
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
