package mage

import "github.com/google/uuid"

// TriggeredAbility checks events and produces effects.
type TriggeredAbility interface {
	Ability
	CheckEventType(EventType) bool
	CheckTrigger(*GameEvent, *Game) bool
	IsOptional() bool
	Effects() []Effect
	Targets() []Target
}

// TriggerCondition is a predicate that determines whether a triggered ability
// should fire for a given event. It receives the event, game state, the source
// permanent's ID, and the controller's ID. Return true to trigger.
// A nil condition always triggers.
type TriggerCondition func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool

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

func (t *GenericTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	if t.Condition == nil {
		return true
	}
	return t.Condition(evt, g, t.source, t.controller)
}

func (t *GenericTriggered) IsOptional() bool { return t.Optional }
func (t *GenericTriggered) Effects() []Effect { return t.effects }
func (t *GenericTriggered) Targets() []Target { return t.targets }

// ---------------------------------------------------------------------------
// Convenience constructors: thin wrappers around NewTriggered that preserve
// the existing API. Each returns *GenericTriggered with the appropriate
// event type and condition baked in.
// ---------------------------------------------------------------------------

// AttacksTrigger fires when the source creature is declared as an attacker.
func AttacksTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDeclaredAttacker, optional, effect).
		SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
			return evt.SourceID == sourceID
		})
}

// DiesCreatureTrigger fires when another creature you control dies.
// The filter parameter is reserved for future use.
func DiesCreatureTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtCreatureDied, optional, effect).
		SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
			if evt.SourceID == sourceID {
				return false // "another" creature — not itself
			}
			return evt.PlayerID == controllerID
		})
}

// EntersBattlefieldTrigger fires when the source permanent enters the battlefield.
func EntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtEntersBattlefield, optional, effect).
		SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
			return evt.SourceID == sourceID
		})
}

// PutIntoGraveyardFromBattlefieldTrigger fires when the source goes to graveyard
// from the battlefield (e.g. Rancor).
func PutIntoGraveyardFromBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtPutIntoGraveyardFromBattlefield, optional, effect).
		SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
			return evt.SourceID == sourceID
		})
}

// BeginningOfUpkeepTrigger fires at the beginning of the controller's upkeep.
func BeginningOfUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect).
		SetCondition(func(evt *GameEvent, _ *Game, _, controllerID uuid.UUID) bool {
			return evt.PlayerID == controllerID
		})
}

// BeginningOfEachUpkeepTrigger fires at the beginning of every player's upkeep.
func BeginningOfEachUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect)
	// nil condition = always fires
}

// DealsDamageToOpponentTrigger fires when the source deals damage to an opponent.
func DealsDamageToOpponentTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
			if evt.SourceID != sourceID {
				return false
			}
			targetPlayer := g.GetPlayer(evt.TargetID)
			if targetPlayer == nil {
				return false // damage was to a creature, not a player
			}
			return targetPlayer.PlayerID() != controllerID
		})
}

// WheneverSpellCastTrigger fires whenever a spell of the matching color is cast.
// Pass nil for colorFilter to trigger on any spell.
func WheneverSpellCastTrigger(effect Effect, optional bool, colorFilter *Color) *GenericTriggered {
	return NewTriggered(EvtSpellCast, optional, effect).
		SetCondition(func(evt *GameEvent, g *Game, _, _ uuid.UUID) bool {
			if colorFilter == nil {
				return true
			}
			card := g.FindCardAnywhere(evt.SourceID)
			if card == nil {
				return false
			}
			for _, c := range card.ManaCost().Colors() {
				if c == *colorFilter {
					return true
				}
			}
			return false
		})
}

// WheneverEnchantmentCastTrigger fires whenever the controller casts an enchantment.
func WheneverEnchantmentCastTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtSpellCast, optional, effect).
		SetCondition(func(evt *GameEvent, g *Game, _, controllerID uuid.UUID) bool {
			if evt.PlayerID != controllerID {
				return false
			}
			card := g.FindCardAnywhere(evt.SourceID)
			if card == nil {
				return false
			}
			return card.HasType(TypeEnchantment)
		})
}

// WhenDamageDealtToThisTrigger fires when damage is dealt to the source creature.
func WhenDamageDealtToThisTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
			return evt.TargetID == sourceID
		})
}

// BeginningOfAttachedControllerUpkeepTrigger fires at the beginning of the
// upkeep of the player who controls the permanent this aura is attached to.
func BeginningOfAttachedControllerUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect).
		SetCondition(func(evt *GameEvent, g *Game, sourceID, _ uuid.UUID) bool {
			src := g.FindPermanent(sourceID)
			if src == nil || !src.IsAttached() {
				return false
			}
			host := g.FindPermanent(src.AttachedTo)
			if host == nil {
				return false
			}
			return evt.PlayerID == host.Controller
		})
}

// WheneverLandEntersBattlefieldTrigger fires whenever any land enters the battlefield.
func WheneverLandEntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtEntersBattlefield, optional, effect).
		SetCondition(func(evt *GameEvent, g *Game, _, _ uuid.UUID) bool {
			perm := g.FindPermanent(evt.SourceID)
			if perm == nil {
				return false
			}
			return perm.HasType(TypeLand)
		})
}

// AnyCreatureDiesTrigger fires when any creature dies (regardless of controller).
func AnyCreatureDiesTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtCreatureDied, optional, effect)
	// nil condition = always fires
}

// CreatureDealtDamageBySourceDiesTrigger fires when a creature that was dealt
// damage by the source permanent this turn dies (e.g. Sengir Vampire).
func CreatureDealtDamageBySourceDiesTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtCreatureDied, optional, effect).
		SetCondition(func(evt *GameEvent, g *Game, sourceID, _ uuid.UUID) bool {
			sources := g.DamageDealtBy[evt.SourceID]
			return sources != nil && sources[sourceID]
		})
}
