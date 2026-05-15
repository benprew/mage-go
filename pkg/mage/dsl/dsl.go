// Package dsl is the card-author-facing surface of the mage engine.
//
// The intent: card files under cards/{arabian,antiquities,...} should ONLY
// import this package (and pkg/mage/core for value types). Anything not
// re-exported here is by definition an engine concern that cards must not
// reach into directly.
//
// This file is a starting-point inventory derived from the symbols actually
// used by cards/arabian/creatures.go. It is intentionally a flat list of
// aliases — not curated or grouped for ergonomics yet. Compile cards against
// this package (with pkg/mage/ moved under internal/) and the resulting
// errors are the ground-truth list of what's still missing or what's
// leaking.
//
// SECTIONS:
//  1. Domain types cards must name in signatures
//  2. Registration entry point
//  3. Card construction (NewCreature, With*)
//  4. Abilities (NewTriggered, AttacksTrigger, ...)
//  5. Trigger condition data
//  6. Effects (the high-value DSL surface)
//  7. Pipeline / composition
//  8. Filters
//  9. Targets
//  10. Costs
//  11. Static / continuous-effect helpers
//  12. Selectors and value sources
//  13. Core enum re-exports (events, layers, durations, keywords, attrs,
//     counters, types, colors)
//  14. ESCAPE HATCHES — flagged for removal
package dsl

import (
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// =============================================================================
// 1. Domain types
// =============================================================================
// Cards must be able to name these in factory signatures and effect closures.
// Exposing the type does NOT expose engine mutators — those live as methods
// on *Game and *Player and stay in pkg/mage proper. Once the package split
// is done these should point at pkg/mage/domain instead.

type (
	Card                   = mage.Card
	Game                   = mage.Game
	GameReader             = mage.GameReader
	Player                 = mage.Player
	Permanent              = mage.Permanent
	Action                 = mage.Action
	ReplacementEffect      = mage.ReplacementEffect
	GenericTriggered       = mage.GenericTriggered
	DestroyPermanentAction = mage.DestroyPermanentAction
	DamageToCreatureAction = mage.DamageToCreatureAction
	DamageToPlayerAction   = mage.DamageToPlayerAction
	GameEvent              = core.GameEvent
	ExiledCard             = mage.ExiledCard
	BaseCard               = mage.BaseCard
	Effect                 = mage.Effect
	EffectProperties       = mage.EffectProperties
	PipelineData           = mage.PipelineData
	PermanentFilter        = mage.PermanentFilter
	TriggeredAbility       = mage.TriggeredAbility
	Cost                   = mage.Cost
	Target                 = mage.Target
	DelayedTrigger         = mage.DelayedTrigger
	ValueSource            = mage.ValueSource
	PlayerSelector         = mage.PlayerSelector
	PermanentSelector      = mage.PermanentSelector
	ContinuousEffect       = mage.ContinuousEffect
	SourceCondition        = mage.SourceCondition
	ActiveCondition        = mage.ActiveCondition
	CardOption             = mage.CardOption
	AbilityOption          = mage.AbilityOption
	ActionOption           = mage.ActionOption

	// Trigger condition data (interface + concrete types)
	TriggerConditionData                   = mage.TriggerConditionData
	EventSourceIsSelf                      = mage.EventSourceIsSelf
	EventPlayerIsController                = mage.EventPlayerIsController
	EventPlayerIsNotController             = mage.EventPlayerIsNotController
	SourceIsAttachedToEventSource          = mage.SourceIsAttachedToEventSource
	EventSourceIsSelfDamageToPlayer        = mage.EventSourceIsSelfDamageToPlayer
	EventSourceIsSelfDamageToOpponent      = mage.EventSourceIsSelfDamageToOpponent
	SourceNotSummonSick                    = mage.SourceNotSummonSick
	SourceIsUnblockedAttacker              = mage.SourceIsUnblockedAttacker
	HasAttackedThisTurnCond                = mage.HasAttackedThisTurnCond
	ControllerHasNoPermanentMatching       = mage.ControllerHasNoPermanentMatching
	NoBattlefieldPermanentMatching         = mage.NoBattlefieldPermanentMatching
	SourceIsTapped                         = mage.SourceIsTapped
	SourceIsUntapped                       = mage.SourceIsUntapped
	EventFlagIsFalse                       = mage.EventFlagIsFalse
	SourceOnBattlefield                    = mage.SourceOnBattlefield
	CombatGroupCountEquals                 = mage.CombatGroupCountEquals
	EventAmountGreaterThan                 = mage.EventAmountGreaterThan
	EventSourceHasType                     = mage.EventSourceHasType
	EventSourceWasOfType                   = mage.EventSourceWasOfType
	EventZoneChangeMatches                 = mage.EventZoneChangeMatches
	SourceInOwnGraveyardWithCreaturesAbove = mage.SourceInOwnGraveyardWithCreaturesAbove
	SourceAttackedOrBlockedThisTurn        = mage.SourceAttackedOrBlockedThisTurn
	SpellCastIsType                        = mage.SpellCastIsType
	OpponentCastNthSpellOfType             = mage.OpponentCastNthSpellOfType
	OpponentCastSpellOfType                = mage.OpponentCastSpellOfType
	EventTargetIsSelf                      = mage.EventTargetIsSelf
	EventSourceHasSubType                  = mage.EventSourceHasSubType
	AttachedToIsEventSourceNoTapCost       = mage.AttachedToIsEventSourceNoTapCost
	AttachedToDealsDamageToController      = mage.AttachedToDealsDamageToController
	OpponentActivatedArtifactNoTapCost     = mage.OpponentActivatedArtifactNoTapCost
	CreatureDeathsOccurred                 = mage.CreatureDeathsOccurred
	SourceInCombatWithMatchingCreature     = mage.SourceInCombatWithMatchingCreature
	SourceIsBlockingInCombat               = mage.SourceIsBlockingInCombat
	SourceBlockedByCreatureMatching        = mage.SourceBlockedByCreatureMatching
	SourceIsBlockedAttacker                = mage.SourceIsBlockedAttacker
	SourceInCombat                         = mage.SourceInCombat
	ManaProduction                         = mage.ManaProduction
	NotTriggerCond                         = mage.NotTriggerCond
	AndTriggerCond                         = mage.AndTriggerCond
	OrTriggerCond                          = mage.OrTriggerCond
	EventTargetHasSubType                  = mage.EventTargetHasSubType

	// Pipeline conditions
	TryPayManaCond = mage.TryPayManaCond
	FlipCoinCond   = mage.FlipCoinCond
	NotCond        = mage.NotCond

	RegisterDelayedTriggerData = mage.RegisterDelayedTriggerData
	HasMatchingPermanentCond   = mage.HasMatchingPermanentCond
	SourceHasCounterCond       = mage.SourceHasCounterCond
	TargetHasRampageCond       = mage.TargetHasRampageCond
	VarGTCond                  = mage.VarGTCond
)

// =============================================================================
// 2. Registration
// =============================================================================

var Register = mage.Register

// =============================================================================
// 3. Card construction
// =============================================================================

var (
	NewCreature                  = mage.NewCreature
	NewArtifact                  = mage.NewArtifact
	NewEnchantment               = mage.NewEnchantment
	NewInstant                   = mage.NewInstant
	NewSorcery                   = mage.NewSorcery
	NewLand                      = mage.NewLand
	NewLandDestruction           = mage.NewLandDestruction
	NewAura                      = mage.NewAura
	NewBoostAura                 = mage.NewBoostAura
	NewLuckyCharm                = mage.NewLuckyCharm
	NewToken                     = mage.NewToken
	WithSubTypes                 = mage.WithSubTypes
	WithCardType                 = mage.WithCardType
	WithKeyword                  = mage.WithKeyword
	WithAbility                  = mage.WithAbility
	WithStaticAbility            = mage.WithStaticAbility
	WithActivatedAbility         = mage.WithActivatedAbility
	WithCost                     = mage.WithCost
	WithTarget                   = mage.WithTarget
	WithAnyPlayerMay             = mage.WithAnyPlayerMay
	WithETBEffect                = mage.WithETBEffect
	WithSuperTypes               = mage.WithSuperTypes
	WithAdditionalCost           = mage.WithAdditionalCost
	WithCastTarget               = mage.WithCastTarget
	WithOncePerTurn              = mage.WithOncePerTurn
	WithEffect                   = mage.WithEffect
	WithUpkeepOnly               = mage.WithUpkeepOnly
	WithControlledSinceTurnStart = mage.WithControlledSinceTurnStart
	WithOpponentOnlyMay          = mage.WithOpponentOnlyMay
	WithMaxActivationsPerTurn    = mage.WithMaxActivationsPerTurn
	WithYourTurnOnly             = mage.WithYourTurnOnly
	WithManaAbility              = mage.WithManaAbility
	WithMultiManaAbility         = mage.WithMultiManaAbility
	WithAnyColorMana             = mage.WithAnyColorMana
	WithStepOnly                 = mage.WithStepOnly
)

// =============================================================================
// 4. Abilities
// =============================================================================

var (
	NewTriggered                               = mage.NewTriggered
	NewStateTriggered                          = mage.NewStateTriggered
	BeginningOfUpkeepTrigger                   = mage.BeginningOfUpkeepTrigger
	AttacksTrigger                             = mage.AttacksTrigger
	BlocksTrigger                              = mage.BlocksTrigger
	EntersBattlefieldTrigger                   = mage.EntersBattlefieldTrigger
	StaticAbility                              = mage.StaticAbility
	GrantAbilityToAttached                     = mage.GrantAbilityToAttached
	GrantActivatedAbilityToAttached            = mage.GrantActivatedAbilityToAttached
	GrantActivatedAbilityToAll                 = mage.GrantActivatedAbilityToAll
	GrantTriggeredAbilityToAll                 = mage.GrantTriggeredAbilityToAll
	NewManaFlareAbility                        = mage.NewManaFlareAbility
	NewAttachedManaBonusAbility                = mage.NewAttachedManaBonusAbility
	BoostAttached                              = mage.BoostAttached
	BoostAllCreaturesIncludingSelf             = mage.BoostAllCreaturesIncludingSelf
	BoostAllCreatures                          = mage.BoostAllCreatures
	BoostOtherControlledCreatures              = mage.BoostOtherControlledCreatures
	GrantKeywordToAll                          = mage.GrantKeywordToAll
	GrantKeywordToOtherControlled              = mage.GrantKeywordToOtherControlled
	LimitLandUntaps                            = mage.LimitLandUntaps
	PreventUntapForMatching                    = mage.PreventUntapForMatching
	TemporaryAnimateUntilEndOfCombat           = mage.TemporaryAnimateUntilEndOfCombat
	TemporaryAnimate                           = mage.TemporaryAnimate
	ManaConversion                             = mage.ManaConversion
	AnimateLands                               = mage.AnimateLands
	GrantColorToAll                            = mage.GrantColorToAll
	CyclopeanTombEffect                        = mage.CyclopeanTombEffect
	PreventBlockingUntilEndOfCombat            = mage.PreventBlockingUntilEndOfCombat
	IncreaseSpellCostForColor                  = mage.IncreaseSpellCostForColor
	ChangeSubTypesForAll                       = mage.ChangeSubTypesForAll
	AllowUnlimitedLandPlays                    = mage.AllowUnlimitedLandPlays
	LimitCreatureUntaps                        = mage.LimitCreatureUntaps
	BoostControlledCreatures                   = mage.BoostControlledCreatures
	RevokeAttrFromControlled                   = mage.RevokeAttrFromControlled
	PreventAllUntaps                           = mage.PreventAllUntaps
	BoostAttachedByCount                       = mage.BoostAttachedByCount
	BodyguardContinuous                        = mage.BodyguardContinuous
	AnimateArtifact                            = mage.AnimateArtifact
	ForTarget                                  = mage.ForTarget
	Attached                                   = mage.Attached
	TemporaryBoost                             = mage.TemporaryBoost
	TemporaryKeyword                           = mage.TemporaryKeyword
	SetBasePT                                  = mage.SetBasePT
	ForAll                                     = mage.ForAll
	ColorOverride                              = mage.ColorOverride
	NullifyLandwalkEffect                      = mage.NullifyLandwalkEffect
	WrapGrantedAbility                         = mage.WrapGrantedAbility
	AttachedEffect                             = mage.AttachedEffect
	PreventAttachedFromUntapping               = mage.PreventAttachedFromUntapping
	PreventAttachedFromAttacking               = mage.PreventAttachedFromAttacking
	GrantKeywordToControlled                   = mage.GrantKeywordToControlled
	RevokeKeywordFromAll                       = mage.RevokeKeywordFromAll
	RemoveKeywordFromAttached                  = mage.RemoveKeywordFromAttached
	ChangeAttachedSubTypes                     = mage.ChangeAttachedSubTypes
	ChangeAttachedSubTypesByChosenColor        = mage.ChangeAttachedSubTypesByChosenColor
	GrantProtectionToAttached                  = mage.GrantProtectionToAttached
	NewManaBonusAbility                        = mage.NewManaBonusAbility
	ControlChangeContinuous                    = mage.ControlChangeContinuous
	BeginningOfAttachedControllerUpkeepTrigger = mage.BeginningOfAttachedControllerUpkeepTrigger
	DealsDamageToOpponentTrigger               = mage.DealsDamageToOpponentTrigger
	SacrificeUnlessLand                        = mage.SacrificeUnlessLand
	CreatureDealtDamageBySourceDiesTrigger     = mage.CreatureDealtDamageBySourceDiesTrigger
	DiesTrigger                                = mage.DiesTrigger
	OnLeaveZone                                = mage.OnLeaveZone
	DiesCreatureTrigger                        = mage.DiesCreatureTrigger
	EntersWithNCounters                        = mage.EntersWithNCounters
	EntersWithXCounters                        = mage.EntersWithXCounters
	BeginningOfEachEndStepTrigger              = mage.BeginningOfEachEndStepTrigger
	CopyCreatureOnETB                          = mage.CopyCreatureOnETB
	ETBEffect                                  = mage.ETBEffect
	AnyCreatureDiesTrigger                     = mage.AnyCreatureDiesTrigger
	BeginningOfEachDrawStepTrigger             = mage.BeginningOfEachDrawStepTrigger
	BeginningOfEachUpkeepTrigger               = mage.BeginningOfEachUpkeepTrigger
	WheneverLandEntersBattlefieldTrigger       = mage.WheneverLandEntersBattlefieldTrigger
	WhenOpponentPermanentBecomesTappedTrigger  = mage.WhenOpponentPermanentBecomesTappedTrigger
	WhenAttachedBecomesTappedTrigger           = mage.WhenAttachedBecomesTappedTrigger
	WhenDamageDealtToThisTrigger               = mage.WhenDamageDealtToThisTrigger
	RampageTrigger                             = mage.RampageTrigger
	WheneverEnchantmentCastTrigger             = mage.WheneverEnchantmentCastTrigger
	WheneverSpellCastTrigger                   = mage.WheneverSpellCastTrigger
	ChooseOpponentOnETB                        = mage.ChooseOpponentOnETB
	ChosenPlayerUpkeepTrigger                  = mage.ChosenPlayerUpkeepTrigger
	PutIntoGraveyardFromBattlefieldTrigger     = mage.PutIntoGraveyardFromBattlefieldTrigger
	NewSpellAbility                            = mage.NewSpellAbility
	NewTargetedSpell                           = mage.NewTargetedSpell
	ToAllMatching                              = mage.ToAllMatching
	RegisterDelayedTriggerStep                 = mage.RegisterDelayedTriggerStep
	SacrificeAtUpkeepUnlessPay                 = mage.SacrificeAtUpkeepUnlessPay
)

// =============================================================================
// 6. Effects
// =============================================================================

var (
	// Removal / combat
	DestroyTarget                             = mage.DestroyTarget
	DestroyTargetPermanent                    = mage.DestroyTargetPermanent
	DestroyTargetNoRegen                      = mage.DestroyTargetNoRegen
	DestroyTargetArtifact                     = mage.DestroyTargetArtifact
	DestroyTargetNoRegenStep                  = mage.DestroyTargetNoRegenStep
	RegenerateTarget                          = mage.RegenerateTarget
	RegenerateGathered                        = mage.RegenerateGathered
	PreventDamageToTarget                     = mage.PreventDamageToTarget
	SacrificeSource                           = mage.SacrificeSource
	Tap                                       = mage.Tap
	UntapSource                               = mage.UntapSource
	RemoveFromCombatGathered                  = mage.RemoveFromCombatGathered
	TapGathered                               = mage.TapGathered
	UntapTarget                               = mage.UntapTarget
	UntapTargetStep                           = mage.UntapTargetStep
	UntapGathered                             = mage.UntapGathered
	MakeUnblockableUntilEndOfTurn             = mage.MakeUnblockableUntilEndOfTurn
	TapAttachedCreature                       = mage.TapAttachedCreature
	TapOrUntapTarget                          = mage.TapOrUntapTarget
	ReturnFromGraveyardToBattlefield          = mage.ReturnFromGraveyardToBattlefield
	ReturnFromGraveyardToHandTarget           = mage.ReturnFromGraveyardToHandTarget
	ExileSourceFromGraveyard                  = mage.ExileSourceFromGraveyard
	ReturnSourceToHand                        = mage.ReturnSourceToHand
	SearchLibraryToHand                       = mage.SearchLibraryToHand
	SearchLibraryToBattlefield                = mage.SearchLibraryToBattlefield
	ShuffleHandAndGraveyardIntoLibraryAndDraw = mage.ShuffleHandAndGraveyardIntoLibraryAndDraw
	ExtraTurn                                 = mage.ExtraTurn
	PowerSinkEffect                           = mage.PowerSinkEffect
	SacrificeGathered                         = mage.SacrificeGathered
	AddManaFromVar                            = mage.AddManaFromVar
	ChooseColorStep                           = mage.ChooseColorStep
	SnapshotSourceCounter                     = mage.SnapshotSourceCounter
	DealDamageToPlayersFromVar                = mage.DealDamageToPlayersFromVar
	ShuffleGraveyardIntoLibrary               = mage.ShuffleGraveyardIntoLibrary
	SetVarFromHandSize                        = mage.SetVarFromHandSize
	AddColorPreventionStep                    = mage.AddColorPreventionStep
	ExileGathered                             = mage.ExileGathered
	GainLifeFromVar                           = mage.GainLifeFromVar
	GainLifeControllerFromVar                 = mage.GainLifeControllerFromVar
	ModalEffect                               = mage.ModalEffect
	AddPreventionShieldToControllerStep       = mage.AddPreventionShieldToControllerStep
	AddReverseDamageShieldStep                = mage.AddReverseDamageShieldStep
	SacrificeSourceStep                       = mage.SacrificeSourceStep
	AddContinuousEffectsStep                  = mage.AddContinuousEffectsStep
	VarPlayer                                 = mage.VarPlayer
	TapAllLands                               = mage.TapAllLands
	RemoveFromCombat                          = mage.RemoveFromCombat
	RegenerateSource                          = mage.RegenerateSource
	StunCreature                              = mage.StunCreature
	ReplaceKeywordEffect                      = mage.ReplaceKeywordEffect
	ForEachCombatOpponent                     = mage.ForEachCombatOpponent
	ForEachPermanent                          = mage.ForEachPermanent
	ForEachAttackerBlockedBySource            = mage.ForEachAttackerBlockedBySource
	ForEachBlockerOfSourceMatching            = mage.ForEachBlockerOfSourceMatching
	ForEachControlledPermanent                = mage.ForEachControlledPermanent
	DrawCards                                 = mage.DrawCards
	DrawCardsActivePlayer                     = mage.DrawCardsActivePlayer
	MillTargetPlayer                          = mage.MillTargetPlayer
	DiscardHandAndDraw                        = mage.DiscardHandAndDraw
	CopySpellOnStack                          = mage.CopySpellOnStack
	DoubleTargetPower                         = mage.DoubleTargetPower
	PreventAllCombatDamage                    = mage.PreventAllCombatDamage
	ForcefieldEffect                          = mage.ForcefieldEffect
	CloneTarget                               = mage.CloneTarget
	BlackViseEffect                           = mage.BlackViseEffect
	TheRackEffect                             = mage.TheRackEffect
	CompositeEffects                          = mage.CompositeEffects
	ChangeColorEffect                         = mage.ChangeColorEffect
	DestroyAllCreatures                       = mage.DestroyAllCreatures
	DestroyAllCreaturesNoRegen                = mage.DestroyAllCreaturesNoRegen
	DestroyAllEnchantments                    = mage.DestroyAllEnchantments
	ChaosOrbEffect                            = mage.ChaosOrbEffect
	DestroyAllLands                           = mage.DestroyAllLands
	DestroyAllMatching                        = mage.DestroyAllMatching
	DestroyAllMatchingNoRegen                 = mage.DestroyAllMatchingNoRegen
	DestroyGatheredNoRegen                    = mage.DestroyGatheredNoRegen
	DestroyGathered                           = mage.DestroyGathered
	DestroyTargetStep                         = mage.DestroyTargetStep
	ExileTargetStep                           = mage.ExileTargetStep
	DestroyAttachedStep                       = mage.DestroyAttachedStep
	BalanceEffect                             = mage.BalanceEffect
	CounterSpell                              = mage.CounterSpell
	CounterSpellIfColor                       = mage.CounterSpellIfColor
	CounterSpellIfXMeetsCMC                   = mage.CounterSpellIfXMeetsCMC
	CounterUnlessPay                          = mage.CounterUnlessPay
	HandSizeDamageEffect                      = mage.HandSizeDamageEffect
	ReturnToHandTarget                        = mage.ReturnToHandTarget

	// Damage / life
	GainLife                  = mage.GainLife
	GainLifeAmount            = mage.GainLifeAmount
	LoseLifeAmount            = mage.LoseLifeAmount
	GainLifeTarget            = mage.GainLifeTarget
	PoisonTargetPlayer        = mage.PoisonTargetPlayer
	DealDamage                = mage.DealDamage
	DealDamageToPlayers       = mage.DealDamageToPlayers
	DealDamageToPlayersStep   = mage.DealDamageToPlayersStep
	DealDamageStep            = mage.DealDamageStep
	DealDamageToSourceStep    = mage.DealDamageToSourceStep
	GainLifeStep              = mage.GainLifeStep
	DealDamageToAllCreatures  = mage.DealDamageToAllCreatures
	SacrificeCreatureOrDamage = mage.SacrificeCreatureOrDamage

	// P/T modification
	Boost                                  = mage.Boost
	BoostSelf                              = mage.BoostSelf
	SetPTUntilEndOfTurn                    = mage.SetPTUntilEndOfTurn
	SetPowerUntilEndOfTurn                 = mage.SetPowerUntilEndOfTurn
	GrantKeyword                           = mage.GrantKeyword
	GrantAbility                           = mage.GrantAbility
	GrantType                              = mage.GrantType
	RevokeKeyword                          = mage.RevokeKeyword
	AddCounters                            = mage.AddCounters
	RemoveCounters                         = mage.RemoveCounters
	ToAttached                             = mage.ToAttached
	ToTarget                               = mage.ToTarget
	ToSource                               = mage.ToSource
	ToGathered                             = mage.ToGathered
	ToMatching                             = mage.ToMatching
	GrantAttrToGathered                    = mage.GrantAttrToGathered
	DiscardRandom                          = mage.DiscardRandom
	ReturnSourceFromGraveyardToBattlefield = mage.ReturnSourceFromGraveyardToBattlefield
	PTEqualsCount                          = mage.PTEqualsCount
	PTEqualsControlledCount                = mage.PTEqualsControlledCount

	// Tokens / snapshot
	CreateColoredToken = mage.CreateColoredToken
	CreateToken        = mage.CreateToken
	CreateTokens       = mage.CreateTokens
	AddMana            = mage.AddMana
	AddAnyMana         = mage.AddAnyMana
	DiscardCards       = mage.DiscardCards
	SnapshotPermanent  = mage.SnapshotPermanent
	SnapshotAttached   = mage.SnapshotAttached

	// Bundled static abilities
	ProtectionFromColor                           = mage.ProtectionFromColor
	PreventDamageFromTo                           = mage.PreventDamageFromTo
	PreventFromAttackingIfDefendingPlayerControls = mage.PreventFromAttackingIfDefendingPlayerControls
)

// =============================================================================
// 7. Pipeline / composition
// =============================================================================

var (
	Pipeline      = mage.Pipeline
	IfElse        = mage.IfElse
	UnwrapAbility = mage.UnwrapAbility
	ApplyEffect   = mage.ApplyEffect
)

// =============================================================================
// 8. Filters
// =============================================================================

var (
	And                 = mage.And
	Or                  = mage.Or
	Not                 = mage.Not
	HasSubType          = mage.HasSubType
	HasKeywordFilter    = mage.HasKeywordFilter
	NotHasKeywordFilter = mage.NotHasKeywordFilter
	IsID                = mage.IsID
	IsBandedWith        = mage.IsBandedWith
	IsCreature          = mage.IsCreature
	IsArtifact          = mage.IsArtifact
	IsEnchantment       = mage.IsEnchantment
	IsLand              = mage.IsLand
	IsLegendary         = mage.IsLegendary
	IsToken             = mage.IsToken
	NewPermanentFilter  = mage.NewPermanentFilter
	NewCardFilter       = mage.NewCardFilter
	IsArtifactCard      = mage.IsArtifactCard
	CreatedByFilter     = mage.CreatedByFilter
	HasPowerGTE         = mage.HasPowerGTE
	HasPowerLTE         = mage.HasPowerLTE
	IsTapped            = mage.IsTapped
	IsUntapped          = mage.IsUntapped
	Named               = mage.Named
	IsAttacking         = mage.IsAttacking
	IsBlocking          = mage.IsBlocking
	ControlledBy        = mage.ControlledBy
	NotControlledBy     = mage.NotControlledBy
	PrintedInSet        = mage.PrintedInSet
	IsAuraOnLand        = mage.IsAuraOnLand
	HasColorFilter      = mage.HasColorFilter
	HasColorCardFilter  = mage.HasColorCardFilter
	AnyPermanent        = mage.AnyPermanent

	// Static-ability "while" guards (functions, not filters, but used the
	// same way at the call site)
	WhileSourceAttacking = mage.WhileSourceAttacking
	WhileSourceUntapped  = mage.WhileSourceUntapped
	WhileControlling     = mage.WhileControlling
	SourceUntapped       = mage.SourceUntapped
	SourceTapped         = mage.SourceTapped
	SourceAttached       = mage.SourceAttached
)

// =============================================================================
// 9. Targets
// =============================================================================

var (
	TargetCreature                          = mage.TargetCreature
	TargetOtherCreature                     = mage.TargetOtherCreature
	TargetArtifact                          = mage.TargetArtifact
	TargetDamageAnyTarget                         = mage.TargetDamageAnyTarget
	TargetCreatureWithPowerLESource         = mage.TargetCreatureWithPowerLESource
	TargetCreatureYouControl                = mage.TargetCreatureYouControl
	TargetPermanent                         = mage.TargetPermanent
	TargetPlayer                            = mage.TargetPlayer
	TargetCreatureInHand                    = mage.TargetCreatureInHand
	TargetSpellOnStack                      = mage.TargetSpellOnStack
	TargetOwnSpellOnStack                   = mage.TargetOwnSpellOnStack
	TargetLand                              = mage.TargetLand
	TargetController                        = mage.TargetController
	TargetArtifactOrEnchantment             = mage.TargetArtifactOrEnchantment
	TargetCreatureInYourGraveyard           = mage.TargetCreatureInYourGraveyard
	TargetCardInYourGraveyard               = mage.TargetCardInYourGraveyard
	TargetControlledCreature                = mage.TargetControlledCreature
	TargetControlledPermanent               = mage.TargetControlledPermanent
	TargetArtifactWithManaValueX            = mage.TargetArtifactWithManaValueX
	TargetCreatureBlockingOrBlockedBySource = mage.TargetCreatureBlockingOrBlockedBySource
	TargetPermanentOpponentControls         = mage.TargetPermanentOpponentControls
	TargetOpponent                          = mage.TargetOpponent
)

// =============================================================================
// 10. Costs
// =============================================================================

var (
	SacrificeSourceCost   = mage.SacrificeSourceCost
	ManaCostOf            = mage.ManaCostOf
	WithFrom              = mage.WithFrom
	WithTo                = mage.WithTo
	WithCombatOnly        = mage.WithCombatOnly
	WithPlayerOnly        = mage.WithPlayerOnly
	GenericCost           = mage.GenericCost
	XManaCost             = mage.XManaCost
	RemoveCountersCost    = mage.RemoveCountersCost
	RequireCountersCost   = mage.RequireCountersCost
	SacrificeCreatureCost = mage.SacrificeCreatureCost
	DiscardRandomCost     = mage.DiscardRandomCost
	ExileSourceCost       = mage.ExileSourceCost
	SacrificeArtifactCost = mage.SacrificeArtifactCost
	SacrificeMatchingCost = mage.SacrificeMatchingCost
	LifePayCost           = mage.LifePayCost
	DiscardCost           = mage.DiscardCost
)

// =============================================================================
// 12. Selectors and value sources
// =============================================================================

var (
	Fixed                           = mage.Fixed
	XValue                          = mage.XValue
	EventAmountValue                = mage.EventAmountValue
	CountBattlefield                = mage.CountBattlefield
	UntappedLandsAtTurnStart        = mage.UntappedLandsAtTurnStart
	HalfRoundUp                     = mage.HalfRoundUp
	PlayerLifeValue                 = mage.PlayerLifeValue
	Mul                             = mage.Mul
	SelectController                = mage.SelectController
	SelectTarget                    = mage.SelectTarget
	SelectSource                    = mage.SelectSource
	SelectActivePlayer              = mage.SelectActivePlayer
	SelectEventController           = mage.SelectEventController
	SelectAttachedController        = mage.SelectAttachedController
	SelectEachPlayer                = mage.SelectEachPlayer
	SelectEachOpponent              = mage.SelectEachOpponent
	SelectTargetPermanentController = mage.SelectTargetPermanentController
	SelectTargetPlayer              = mage.SelectTargetPlayer
	SelectDefendingPlayer           = mage.SelectDefendingPlayer
)

// Outcome enum values used in EffectProperties literals.
const (
	OutcomeDetriment = mage.OutcomeDetriment
	OutcomeBenefit   = mage.OutcomeBenefit
	OutcomeUnknown   = mage.OutcomeUnknown
)

// =============================================================================
// 13. Core enum re-exports
// =============================================================================
// All of these live in pkg/mage/core today. Cards currently dot-import core
// directly; once cards import only this package, these aliases let them keep
// the bare names.

// Layers (Rule 613)
const (
	LayerControl = core.LayerControl
	LayerAbility = core.LayerAbility
	LayerType    = core.LayerType
	LayerPT      = core.LayerPT
)

// Attach kinds
const (
	AttachAura = core.AttachAura
)

// Core helpers (functions exported from core)
var (
	LandwalkAttr  = core.LandwalkAttr
	LandwalkAttrs = core.LandwalkAttrs
)

// Turn phase steps
const (
	EndCombat = core.EndCombat
)

// Durations
const (
	WhileOnBattlefield = core.WhileOnBattlefield
	EndOfTurn          = core.EndOfTurn
	EndOfCombat        = core.EndOfCombat
	UntilYourNextTurn  = core.UntilYourNextTurn
	Indefinite         = core.Indefinite
)

// Zones
const (
	ZoneLibrary     = core.ZoneLibrary
	ZoneHand        = core.ZoneHand
	ZoneBattlefield = core.ZoneBattlefield
	ZoneGraveyard   = core.ZoneGraveyard
	ZoneExile       = core.ZoneExile
	ZoneStack       = core.ZoneStack
	ZoneAny         = core.ZoneAny
)

// Event types
const (
	EvtZoneChange       = core.EvtZoneChange
	EvtBlockersDecl     = core.EvtBlockersDecl
	EvtEndStep          = core.EvtEndStep
	EvtDrawStep         = core.EvtDrawStep
	EvtDamageDealt      = core.EvtDamageDealt
	EvtTapped           = core.EvtTapped
	EvtUpkeep           = core.EvtUpkeep
	EvtDeclaredAttacker = core.EvtDeclaredAttacker
	EvtLifeGained       = core.EvtLifeGained
	EvtLandPlayed       = core.EvtLandPlayed
	EvtEndOfCombat      = core.EvtEndOfCombat
	EvtBecameUntapped   = core.EvtBecameUntapped
	EvtSpellCast        = core.EvtSpellCast
	EvtBeginCombat      = core.EvtBeginCombat
	EvtDeclaredBlocker  = core.EvtDeclaredBlocker
	EvtAbilityActivated = core.EvtAbilityActivated
	EvtCardDrawn        = core.EvtCardDrawn
)

// Card types
const (
	TypeCreature    = core.TypeCreature
	TypeArtifact    = core.TypeArtifact
	TypeLand        = core.TypeLand
	TypeEnchantment = core.TypeEnchantment
	TypeInstant     = core.TypeInstant
	TypeSorcery     = core.TypeSorcery

	SuperBasic     = core.SuperBasic
	SuperLegendary = core.SuperLegendary
	SuperWorld     = core.SuperWorld
)

// Colors
const (
	White     = core.White
	Blue      = core.Blue
	Black     = core.Black
	Red       = core.Red
	Green     = core.Green
	Colorless = core.Colorless
)

// Keywords (these are Attr values in core/attr.go)
const (
	Flying                     = core.Flying
	Reach                      = core.Reach
	FirstStrike                = core.FirstStrike
	Trample                    = core.Trample
	Vigilance                  = core.Vigilance
	Haste                      = core.Haste
	Menace                     = core.Menace
	Fear                       = core.Fear
	Deathtouch                 = core.Deathtouch
	Lifelink                   = core.Lifelink
	Defender                   = core.Defender
	Banding                    = core.Banding
	Hexproof                   = core.Hexproof
	Shroud                     = core.Shroud
	Forestwalk                 = core.Forestwalk
	Islandwalk                 = core.Islandwalk
	Swampwalk                  = core.Swampwalk
	Mountainwalk               = core.Mountainwalk
	Plainswalk                 = core.Plainswalk
	Desertwalk                 = core.Desertwalk
	BasiliskTouch              = core.BasiliskTouch
	CantBeBlockedByWalls       = core.CantBeBlockedByWalls
	CantBeBlockedExceptByWalls = core.CantBeBlockedExceptByWalls
	MustAttack                 = core.MustAttack
	AttrMustAttack             = core.AttrMustAttack
	MustBeBlocked              = core.MustBeBlocked
	UnblockableKW              = core.UnblockableKW
	AttrIsCreature             = core.AttrIsCreature
	LegendaryLandwalk          = core.LegendaryLandwalk
	Indestructible             = core.Indestructible
	CanBlockAny                = core.CanBlockAny
	CanBlockAdditional         = core.CanBlockAdditional
	AttrCanAttack              = core.AttrCanAttack
	CantRegenerate             = core.CantRegenerate
	DoesNotUntapKW             = core.DoesNotUntapKW
)

// Attrs (non-keyword permanent flags)
const (
	AttrCanBlock                  = core.AttrCanBlock
	AttrDoesNotUntap              = core.AttrDoesNotUntap
	AttrCantBeEnchanted           = core.AttrCantBeEnchanted
	AttrCantChangeControl         = core.AttrCantChangeControl
	AttrMayNotUntap               = core.AttrMayNotUntap
	AttrCantBeTargetedByArtifacts = core.AttrCantBeTargetedByArtifacts
	EntersTapped                  = core.EntersTapped
)

// Counters
const (
	P1P1         = core.P1P1
	P1P0         = core.P1P0
	P1P2         = core.P1P2
	Spore        = core.Spore
	M1M1         = core.M1M1
	M0M1         = core.M0M1
	M0M2         = core.M0M2
	Dream        = core.Dream
	Sleep        = core.Sleep
	Glyph        = core.Glyph
	Pupa         = core.Pupa
	Intervention = core.Intervention
	Credit       = core.Credit
	Javelin      = core.Javelin
	Tide         = core.Tide
	Storage      = core.Storage
	Wind         = core.Wind
	Mire         = core.Mire
	Charge       = core.Charge
	Corpse       = core.Corpse
	Doom         = core.Doom
	Hatchling    = core.Hatchling
	Matrix       = core.Matrix
	Pin          = core.Pin
	Carrion      = core.Carrion
	Vitality     = core.Vitality
	NumCounters  = core.NumCounters
)

// Type aliases for core value types named in card signatures.
type (
	Color              = core.Color
	CardType           = core.CardType
	Attr               = core.Attr // Keyword is an alias for Attr in core
	Layer              = core.Layer
	Duration           = core.Duration
	AttachType         = core.AttachType
	PhaseStep          = core.PhaseStep
	EventType          = core.EventType
	CounterType        = core.CounterType
	SuperType          = core.SuperType
	ManaRestriction    = core.ManaRestriction
	ArtifactSpellsOnly = core.ArtifactSpellsOnly
	CreatureSpellsOnly = core.CreatureSpellsOnly
)

// =============================================================================
// 14. ESCAPE HATCHES — flagged for removal
// =============================================================================
// These are the leak sites. Every cards/arabian/creatures.go FuncEffect
// reaches into *Game directly through these. Each represents missing DSL
// coverage. Goal: get this section to zero entries.
//
// Counts below are uses inside cards/arabian/creatures.go alone:
//   FuncEffect           x18
//   FuncContinuousEffect x4 (Old Man, Guardian Beast, Ali from Cairo, Aladdin)
//   TargetEffect         x2 (Ydwen Efreet, Erhnam Djinn)
//
// Either remove these aliases (and the affected cards stop compiling until
// each TODO in creatures.go is converted to a Pipeline), or keep them with
// an explicit lint check that flags any new use under cards/.

var (
	FuncEffect           = mage.FuncEffect           // ESCAPE: arbitrary *Game access in resolution
	FuncContinuousEffect = mage.FuncContinuousEffect // ESCAPE: arbitrary *Game access in layer apply
	TargetEffect         = mage.TargetEffect         // ESCAPE: arbitrary per-target *Game access
)
