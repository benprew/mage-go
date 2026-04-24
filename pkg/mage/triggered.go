package mage

import (
	"fmt"

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

// Indicates the triggered ability should fire if the event source is the card
// that this ability is attached to.
func IsThisSource(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return evt.SourceID == sourceID
}

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
// Convenience constructors: thin wrappers around NewTriggered that preserve
// the existing API. Each returns *GenericTriggered with the appropriate
// event type and condition baked in.
// ---------------------------------------------------------------------------

// AttacksTrigger fires when the source creature is declared as an attacker.
func AttacksTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDeclaredAttacker, optional, effect).
		SetCondition(func(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
			return evt.SourceID == sourceID
		})
}

// BlocksTrigger fires when the source creature is declared as a blocker.
func BlocksTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDeclaredBlocker, optional, effect).
		SetCondition(func(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
			return evt.SourceID == sourceID
		})
}

// DiesCreatureTrigger fires when another creature you control dies.
// The filter parameter is reserved for future use.
func DiesCreatureTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtCreatureDied, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
			if evt.SourceID == sourceID {
				return false // "another" creature — not itself
			}
			return evt.PlayerID == controllerID
		})
}

// EntersBattlefieldTrigger fires when the source permanent enters the battlefield.
func EntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtEntersBattlefield, optional, effect).
		SetCondition(func(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
			return evt.SourceID == sourceID
		})
}

// PutIntoGraveyardFromBattlefieldTrigger fires when the source goes to graveyard
// from the battlefield (e.g. Rancor).
func PutIntoGraveyardFromBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtPutIntoGraveyardFromBattlefield, optional, effect).
		SetCondition(func(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
			return evt.SourceID == sourceID
		})
}

// ChooseOpponentOnETB sets the permanent's ChosenPlayer to the opponent on ETB.
// Used by Black Vise, The Rack, and similar "as this enters, choose an opponent" cards.
func ChooseOpponentOnETB() *GenericTriggered {
	return EntersBattlefieldTrigger(FuncEffect(
		"choose an opponent",
		EffectProperties{},
		func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
			perm := g.FindPermanent(sourceID)
			if perm == nil {
				return false
			}
			return evt.PlayerID == perm.ChosenPlayer
		})
}

// BeginningOfUpkeepTrigger fires at the beginning of the controller's upkeep.
func BeginningOfUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect).
		SetCondition(func(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
			return evt.PlayerID == controllerID
		})
}

// BeginningOfEachUpkeepTrigger fires at the beginning of every player's upkeep.
func BeginningOfEachUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect)
	// nil condition = always fires
}

// BeginningOfEachDrawStepTrigger fires at the beginning of every player's draw step.
// The effect receives the active player's ID in the event's PlayerID field.
// Only triggers while the source permanent is untapped.
func BeginningOfEachDrawStepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDrawStep, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
			src := g.FindPermanent(sourceID)
			return src != nil && !src.Tapped
		})
}

// BeginningOfEachEndStepTrigger fires at the beginning of every player's end step.
func BeginningOfEachEndStepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtEndStep, optional, effect)
}

// BeginningOfEachCleanupStepTrigger fires at the beginning of every player's
// cleanup step (CR 514). Per CR 514.3a, a trigger that fires during cleanup
// causes players to receive priority and a new cleanup step to begin after
// the triggered ability resolves.
func BeginningOfEachCleanupStepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtCleanup, optional, effect)
}

// DealsDamageToOpponentTrigger fires when the source deals damage to an opponent.
func DealsDamageToOpponentTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
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

// WheneverSpellCastTrigger fires whenever any player casts a spell matching the
// given CardFilter predicates. Pass no filters to trigger on any spell.
func WheneverSpellCastTrigger(effect Effect, optional bool, filters ...CardFilter) *GenericTriggered {
	return NewTriggered(EvtSpellCast, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
			if len(filters) == 0 {
				return true
			}
			card := g.FindCardAnywhere(evt.SourceID)
			if card == nil {
				return false
			}
			for _, f := range filters {
				if !f.Match(card) {
					return false
				}
			}
			return true
		})
}

// WheneverYouCastSpellTrigger fires whenever the controller casts a spell matching
// the given CardFilter predicates. Pass no filters to trigger on any of your spells.
func WheneverYouCastSpellTrigger(effect Effect, optional bool, filters ...CardFilter) *GenericTriggered {
	return NewTriggered(EvtSpellCast, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
			if evt.PlayerID != controllerID {
				return false
			}
			card := g.FindCardAnywhere(evt.SourceID)
			if card == nil {
				return false
			}
			for _, f := range filters {
				if !f.Match(card) {
					return false
				}
			}
			return true
		})
}

// WheneverEnchantmentCastTrigger fires whenever the controller casts an enchantment.
// Convenience wrapper for WheneverYouCastSpellTrigger with IsEnchantmentCard filter.
func WheneverEnchantmentCastTrigger(effect Effect, optional bool) *GenericTriggered {
	return WheneverYouCastSpellTrigger(effect, optional, IsEnchantmentCard)
}

// WhenDamageDealtToThisTrigger fires when damage is dealt to the source creature.
func WhenDamageDealtToThisTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetCondition(func(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
			return evt.TargetID == sourceID
		})
}

// BeginningOfAttachedControllerUpkeepTrigger fires at the beginning of the
// upkeep of the player who controls the permanent this aura is attached to.
func BeginningOfAttachedControllerUpkeepTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtUpkeep, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
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

// WheneverPermanentEntersBattlefieldTrigger fires whenever a permanent matching
// the filter enters the battlefield.
func WheneverPermanentEntersBattlefieldTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtEntersBattlefield, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
			perm := g.FindPermanent(evt.SourceID)
			if perm == nil {
				return false
			}
			return filter.Match(perm, g.(*Game))
		})
}

// WheneverLandEntersBattlefieldTrigger fires whenever any land enters the battlefield.
// Convenience wrapper for WheneverPermanentEntersBattlefieldTrigger with IsLand filter.
func WheneverLandEntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return WheneverPermanentEntersBattlefieldTrigger(effect, optional, IsLand)
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
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
			sources := g.GetDamageSources(evt.SourceID)
			return sources != nil && sources[sourceID]
		})
}

// SacrificeAtUpkeepUnlessPay creates a trigger that sacrifices the source at
// the beginning of the controller's upkeep unless the mana cost can be paid
// by tapping lands. If cost is non-empty, the engine auto-pays from untapped
// lands; otherwise the permanent is always sacrificed.
func SacrificeAtUpkeepUnlessPay(cost string) *GenericTriggered {
	return NewTriggered(EvtUpkeep, false, FuncEffect(
		"Sacrifice unless pay "+cost,
		EffectProperties{},
		func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			if cost != "" && g.TryPayCostFromLands(controller, cost) {
				return nil // paid, keep the permanent
			}
			perm := g.FindPermanent(sourceID)
			if perm != nil {
				g.Sacrifice(perm)
			}
			return nil
		},
	)).SetCondition(func(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
		return evt.PlayerID == controllerID
	})
}

// WhenAttachedBecomesTappedTrigger fires when the permanent this aura is
// attached to becomes tapped (e.g. Psychic Venom, Kudzu).
func WhenAttachedBecomesTappedTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtTapped, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
			src := g.FindPermanent(sourceID)
			if src == nil || !src.IsAttached() {
				return false
			}
			return evt.SourceID == src.AttachedTo
		})
}

// RampageTrigger creates a triggered ability for Rampage N. When the source
// becomes blocked, it gets +N/+N until end of turn for each creature blocking
// it beyond the first.
func RampageTrigger(n int) *GenericTriggered {
	return NewTriggered(EvtBlockersDecl, false, FuncEffect(
		fmt.Sprintf("rampage %d", n),
		EffectProperties{},
		func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			// Count how many creatures are blocking the source
			blockerCount := 0
			for _, group := range g.CombatGroups() {
				if group.AttackerID == sourceID {
					blockerCount = len(group.BlockerIDs)
					break
				}
			}
			if blockerCount <= 1 {
				return nil // not blocked or only 1 blocker — no rampage bonus
			}
			bonus := n * (blockerCount - 1)
			ce := TemporaryBoost(sourceID, bonus, bonus)
			ce.SetSourceID(sourceID)
			g.AddContinuousEffect(ce)
			return nil
		},
	)).SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
		group := g.CombatGroupFor(sourceID)
		return group != nil && len(group.BlockerIDs) > 0
	})
}

// WhenOpponentPermanentBecomesTappedTrigger fires when a permanent matching the
// filter that an opponent controls becomes tapped (e.g. Lifetap).
func WhenOpponentPermanentBecomesTappedTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtTapped, optional, effect).
		SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
			perm := g.FindPermanent(evt.SourceID)
			if perm == nil {
				return false
			}
			if perm.Controller == controllerID {
				return false // not an opponent's permanent
			}
			return filter.Match(perm, g.(*Game))
		})
}
