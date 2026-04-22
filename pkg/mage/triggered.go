package mage

import (
	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TriggeredAbility checks events and produces effects.
type TriggeredAbility interface {
	Ability
	CheckEventType(EventType) bool
	CheckTrigger(*GameEvent, GameReader) bool
	IsOptional() bool
	Effects() []Effect
	Targets() []Target
}

// TriggerCondition is a predicate that determines whether a triggered ability
// should fire for a given event. It receives the event, a read-only game view,
// the source permanent's ID, and the controller's ID. Return true to trigger.
// A nil condition always triggers.
type TriggerCondition func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool

// GenericTriggered is a universal triggered ability that replaces bespoke trigger
// types. It listens for a single EventType and applies an optional condition
// function to decide whether to fire. All existing trigger constructors
// (AttacksTrigger, BeginningOfUpkeepTrigger, etc.) are thin wrappers that
// create a GenericTriggered with the appropriate condition.
type GenericTriggered struct {
	BaseAbility
	eventType EventType
	Optional  bool
	Condition TriggerCondition
	effects   []Effect
	targets   []Target
}

// NewTriggered creates a GenericTriggered ability that fires on the given event type.
// Use SetCondition to add a predicate that filters which events actually trigger it.
func NewTriggered(eventType EventType, optional bool, effects ...Effect) *GenericTriggered {
	return &GenericTriggered{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityTriggered,
		},
		eventType: eventType,
		Optional:  optional,
		effects:   effects,
	}
}

// SetCondition sets the trigger condition and returns the trigger for chaining.
func (t *GenericTriggered) SetCondition(cond TriggerCondition) *GenericTriggered {
	t.Condition = cond
	return t
}

// SetConditionData sets the trigger condition from a composable TriggerConditionData
// and returns the trigger for chaining.
func (t *GenericTriggered) SetConditionData(cond TriggerConditionData) *GenericTriggered {
	t.Condition = AsTriggerCondition(cond)
	return t
}

// AddEffect appends an effect to the trigger. Returns the trigger for chaining.
func (t *GenericTriggered) AddEffect(e Effect) *GenericTriggered {
	t.effects = append(t.effects, e)
	return t
}

// AddTarget appends a target requirement. Returns the trigger for chaining.
func (t *GenericTriggered) AddTarget(tgt Target) *GenericTriggered {
	t.targets = append(t.targets, tgt)
	return t
}

func (t *GenericTriggered) CheckEventType(et EventType) bool {
	return et == t.eventType
}

func (t *GenericTriggered) CheckTrigger(evt *GameEvent, g GameReader) bool {
	if t.Condition == nil {
		return true
	}
	return t.Condition(evt, g, t.source, t.controller)
}

func (t *GenericTriggered) IsOptional() bool  { return t.Optional }
func (t *GenericTriggered) Effects() []Effect { return t.effects }
func (t *GenericTriggered) Targets() []Target { return t.targets }

// ---------------------------------------------------------------------------
// Convenience constructors: thin wrappers around NewTriggered that use
// composable TriggerConditionData predicates instead of closures.
// ---------------------------------------------------------------------------

// AttacksTrigger fires when the source creature is declared as an attacker.
func AttacksTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDeclaredAttacker, optional, effect).
		SetConditionData(EventSourceIsSelf{})
}

// BlocksTrigger fires when the source creature is declared as a blocker.
func BlocksTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDeclaredBlocker, optional, effect).
		SetConditionData(EventSourceIsSelf{})
}

// DiesCreatureTrigger fires when another creature you control dies.
// The filter parameter is reserved for future use.
func DiesCreatureTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtCreatureDied, optional, effect).
		SetConditionData(AndTriggerCond{[]TriggerConditionData{
			EventSourceNotSelf{},
			EventPlayerIsController{},
		}})
}

// EntersBattlefieldTrigger fires when the source permanent enters the battlefield.
func EntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtEntersBattlefield, optional, effect).
		SetConditionData(EventSourceIsSelf{})
}

// PutIntoGraveyardFromBattlefieldTrigger fires when the source goes to graveyard
// from the battlefield (e.g. Rancor).
func PutIntoGraveyardFromBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtPutIntoGraveyardFromBattlefield, optional, effect).
		SetConditionData(EventSourceIsSelf{})
}

// ChooseOpponentOnETB sets the permanent's ChosenPlayer to the opponent on ETB.
// Used by Black Vise, The Rack, and similar "as this enters, choose an opponent" cards.
func ChooseOpponentOnETB() *GenericTriggered {
	return EntersBattlefieldTrigger(FuncEffect(
		"choose an opponent",
		EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			perm := g.FindPermanent(sourceID)
			if perm == nil {
				return nil
			}
			opponent := g.GetOpponent(controller)
			if opponent != nil {
				perm.ChosenPlayer = opponent.PlayerID()
			}
			return nil
		}), false)
}

// ChosenPlayerUpkeepTrigger fires at the beginning of the chosen player's upkeep.
// Requires ChooseOpponentOnETB to have set ChosenPlayer.
func ChosenPlayerUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect).
		SetConditionData(EventIsChosenPlayerUpkeep{})
}

// BeginningOfUpkeepTrigger fires at the beginning of the controller's upkeep.
func BeginningOfUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect).
		SetConditionData(EventPlayerIsController{})
}

// BeginningOfEachUpkeepTrigger fires at the beginning of every player's upkeep.
func BeginningOfEachUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect)
	// nil condition = always fires
}

// BeginningOfEachDrawStepTrigger fires at the beginning of every player's draw step.
// Only triggers while the source permanent is untapped.
func BeginningOfEachDrawStepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDrawStep, optional, effect).
		SetConditionData(SourceIsUntapped{})
}

// BeginningOfEachEndStepTrigger fires at the beginning of every player's end step.
func BeginningOfEachEndStepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtEndStep, optional, effect)
}

// DealsDamageToOpponentTrigger fires when the source deals damage to an opponent.
func DealsDamageToOpponentTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetConditionData(EventSourceIsSelfDamageToOpponent{})
}

// WheneverSpellCastTrigger fires whenever any player casts a spell matching the
// given CardFilter predicates. Pass no filters to trigger on any spell.
func WheneverSpellCastTrigger(effect Effect, optional bool, filters ...CardFilter) *GenericTriggered {
	if len(filters) == 0 {
		return NewTriggered(EvtSpellCast, optional, effect)
	}
	return NewTriggered(EvtSpellCast, optional, effect).
		SetConditionData(SpellCastMatchesCardFilters{Filters: filters})
}

// WheneverYouCastSpellTrigger fires whenever the controller casts a spell matching
// the given CardFilter predicates. Pass no filters to trigger on any of your spells.
func WheneverYouCastSpellTrigger(effect Effect, optional bool, filters ...CardFilter) *GenericTriggered {
	conds := []TriggerConditionData{EventPlayerIsController{}}
	if len(filters) > 0 {
		conds = append(conds, SpellCastMatchesCardFilters{Filters: filters})
	}
	return NewTriggered(EvtSpellCast, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: conds})
}

// WheneverEnchantmentCastTrigger fires whenever the controller casts an enchantment.
func WheneverEnchantmentCastTrigger(effect Effect, optional bool) *GenericTriggered {
	return WheneverYouCastSpellTrigger(effect, optional, IsEnchantmentCard)
}

// WhenDamageDealtToThisTrigger fires when damage is dealt to the source creature.
func WhenDamageDealtToThisTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetConditionData(EventTargetIsSelf{})
}

// BeginningOfAttachedControllerUpkeepTrigger fires at the beginning of the
// upkeep of the player who controls the permanent this aura is attached to.
func BeginningOfAttachedControllerUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect).
		SetConditionData(EventIsAttachedControllerUpkeep{})
}

// WheneverPermanentEntersBattlefieldTrigger fires whenever a permanent matching
// the filter enters the battlefield.
func WheneverPermanentEntersBattlefieldTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtEntersBattlefield, optional, effect).
		SetConditionData(EventSourceMatchesPermanentFilter{Filter: filter})
}

// WheneverLandEntersBattlefieldTrigger fires whenever any land enters the battlefield.
func WheneverLandEntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return WheneverPermanentEntersBattlefieldTrigger(effect, optional, IsLand)
}

// AnyCreatureDiesTrigger fires when any creature dies (regardless of controller).
func AnyCreatureDiesTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtCreatureDied, optional, effect)
}

// CreatureDealtDamageBySourceDiesTrigger fires when a creature that was dealt
// damage by the source permanent this turn dies (e.g. Sengir Vampire).
func CreatureDealtDamageBySourceDiesTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtCreatureDied, optional, effect).
		SetConditionData(EventSourceDamagedBySource{})
}

// SacrificeAtUpkeepUnlessPay creates a trigger that sacrifices the source at
// the beginning of the controller's upkeep unless the mana cost can be paid.
func SacrificeAtUpkeepUnlessPay(cost string) *GenericTriggered {
	return NewTriggered(EvtUpkeep, false,
		DataEffect(IfElse(
			"Sacrifice unless pay "+cost,
			&TryPayManaCond{Cost: cost},
			nil,
			SacrificeSourceStep(),
		)),
	).SetConditionData(EventPlayerIsController{})
}

// WhenAttachedBecomesTappedTrigger fires when the permanent this aura is
// attached to becomes tapped (e.g. Psychic Venom, Kudzu).
func WhenAttachedBecomesTappedTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtTapped, optional, effect).
		SetConditionData(SourceIsAttachedToEventSource{})
}

// RampageTrigger creates a triggered ability for Rampage N.
func RampageTrigger(n int) *GenericTriggered {
	return NewTriggered(EvtBlockersDecl, false,
		DataEffect(RampageEffect(n)),
	).SetConditionData(SourceIsBlockedAttacker{})
}

// WhenOpponentPermanentBecomesTappedTrigger fires when a permanent matching the
// filter that an opponent controls becomes tapped.
func WhenOpponentPermanentBecomesTappedTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtTapped, optional, effect).
		SetConditionData(AndTriggerCond{[]TriggerConditionData{
			EventSourceControlledByOpponent{},
			EventSourceMatchesPermanentFilter{Filter: filter},
		}})
}
