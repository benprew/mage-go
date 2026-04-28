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

// AndConditionData composes the given predicate with any existing trigger
// condition (logical AND). If no condition is set, this becomes the
// condition. Use AndConditionData (rather than SetConditionData) when adding
// a refinement on top of a constructor-provided filter — for example when
// WheneverPermanentEntersBattlefieldTrigger has already installed the filter
// check, and you want to additionally require the source not to be self.
func (t *GenericTriggered) AndConditionData(cond TriggerConditionData) *GenericTriggered {
	if t.Condition == nil {
		t.Condition = AsTriggerCondition(cond)
		return t
	}
	prev := t.Condition
	added := AsTriggerCondition(cond)
	t.Condition = func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
		return prev(evt, g, sourceID, controllerID) && added(evt, g, sourceID, controllerID)
	}
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

// WheneverOneOrMoreCreaturesAttackTrigger fires once per combat after all
// attackers have been declared (CR 506.4 / 603.6e), if any creature attacked.
// evt.PlayerID is the attacking (active) player; evt.Amount is the attacker
// count. Used by Duelist's Heritage and similar "whenever one or more
// creatures attack" effects that should resolve once per combat regardless
// of attacker count.
func WheneverOneOrMoreCreaturesAttackTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtAttackersDeclared, optional, effect)
}

// WheneverOneOrMoreCreaturesYouControlAttackTrigger restricts the once-per-
// combat trigger to combats in which the controller is the attacking player.
func WheneverOneOrMoreCreaturesYouControlAttackTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtAttackersDeclared, optional, effect).
		SetConditionData(EventPlayerIsController{})
}

// WheneverOneOrMoreCreaturesYouControlDealCombatDamageToPlayerTrigger fires
// once per combat damage step (CR 510.2) for each opposing player that took
// any combat damage from creatures controlled by the source's controller.
// Used by Keeper of Fables and other "whenever one or more creatures you
// control deal combat damage to a player" triggers. evt.PlayerID is the
// damaging-creatures' controller, evt.TargetID is the player taking damage,
// evt.Amount is the total combat damage dealt to that player by that
// controller's creatures this step.
func WheneverOneOrMoreCreaturesYouControlDealCombatDamageToPlayerTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtCombatDamageDealt, optional, effect).
		SetConditionData(EventPlayerIsController{})
}

// AttacksTrigger fires when the source creature is declared as an attacker.
func AttacksTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDeclaredAttacker, optional, effect).
		SetConditionData(EventSourceIsSelf{})
}

// BlocksTrigger fires once per combat when the source creature blocks one or
// more attackers (CR 509.3a). For "Whenever [creature] blocks a creature"
// (per-attacker) triggers, construct a NewTriggered on EvtDeclaredBlocker
// directly.
func BlocksTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtCreatureBlocks, optional, effect).
		SetConditionData(EventSourceIsSelf{})
}

// DiesCreatureTrigger fires when another creature you control dies.
// The filter parameter is reserved for future use.
// Fires on EvtZoneChange (BF -> GY) with a was-creature LKI predicate
// (CR 700.4 / 603.6c).
func DiesCreatureTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtZoneChange, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
			EventSourceWasOfType{Type: TypeCreature},
			EventSourceNotSelf{},
			EventPlayerIsController{},
		}})
}

// OnEnterZone fires when the source permanent enters `to`. Generic
// zone-change trigger over EvtZoneChange (CR 603.10): a single zone change
// is one event, so any "when ~ enters X" can be expressed this way. For the
// common ZoneBattlefield case prefer EntersBattlefieldTrigger which is a
// thin alias.
func OnEnterZone(to Zone, effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtZoneChange, optional, effect).
		SetConditionData(AndTriggerCond{[]TriggerConditionData{
			EventSourceIsSelf{},
			EventZoneChangeMatches{From: ZoneAny, To: to},
		}})
}

// OnLeaveZone fires when the source permanent leaves `from`, optionally
// constrained to also entering `to`. Pass ZoneAny for `to` to fire on any
// destination (CR 603.6c — leaves-the-battlefield triggers consult LKI;
// the destination is irrelevant to whether the event fired).
//
// The engine captures the source's runtime abilities into the LKI snapshot
// at RemoveFromBattlefield time, so this trigger fires uniformly regardless
// of which leave path (destroy / sacrifice / SBA / bounce / exile) moved
// the permanent.
func OnLeaveZone(from, to Zone, effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtZoneChange, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventSourceIsSelf{},
			EventZoneChangeMatches{From: from, To: to},
		}})
}

// EntersBattlefieldTrigger fires when the source permanent enters the
// battlefield. Implemented as an OnEnterZone(ZoneBattlefield) — fires on
// EvtZoneChange with ToZone=Battlefield (CR 603.6d).
func EntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return OnEnterZone(ZoneBattlefield, effect, optional)
}

// PutIntoGraveyardFromBattlefieldTrigger fires when the source goes to graveyard
// from the battlefield (e.g. Rancor). Battlefield-only — does not fire on
// the Sacrifice path. Use LeavesBattlefieldToGraveyardTrigger if you want
// "when ~ is put into a graveyard from the battlefield" Oracle wording that
// catches sacrifices too.
func PutIntoGraveyardFromBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtZoneChange, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventSourceIsSelf{},
			EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
		}})
}

// LeavesBattlefieldToGraveyardTrigger fires whenever the source is put into a
// graveyard from the battlefield, via any path (lethal damage, destroy,
// sacrifice, state-based effect, etc.). Oracle wording "When CARDNAME is put
// into a graveyard from the battlefield..." (Terrarion-style artifacts) per
// CR 603.6c (leaves-the-battlefield triggers look back at the permanent's
// LKI). With CR 700.4 collapsing sacrifice into "put into a graveyard from
// the battlefield," this constructor and PutIntoGraveyardFromBattlefieldTrigger
// now fire on identical sets of paths and could be unified.
func LeavesBattlefieldToGraveyardTrigger(effect Effect, optional bool) *GenericTriggered {
	return OnLeaveZone(ZoneBattlefield, ZoneGraveyard, effect, optional)
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

// BeginningOfFirstMainPhaseTrigger fires at the beginning of the controller's
// precombat (first) main phase (CR 505). Used by cards like Black Market that
// say "at the beginning of your first main phase". Per CR 505, EvtMainPhase
// has Flag=true for precombat and Flag=false for postcombat.
func BeginningOfFirstMainPhaseTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtMainPhase, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventPlayerIsController{},
			EventFlagIsTrue{},
		}})
}

// BeginningOfPostcombatMainPhaseTrigger fires at the beginning of the
// controller's postcombat main phase.
func BeginningOfPostcombatMainPhaseTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtMainPhase, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventPlayerIsController{},
			EventFlagIsFalse{},
		}})
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
// the filter enters the battlefield. Fires on EvtZoneChange (To=Battlefield)
// — see CR 603.10 / 603.6d.
func WheneverPermanentEntersBattlefieldTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtZoneChange, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
			EventSourceMatchesPermanentFilter{Filter: filter},
		}})
}

// WheneverLandEntersBattlefieldTrigger fires whenever any land enters the battlefield.
func WheneverLandEntersBattlefieldTrigger(effect Effect, optional bool) *GenericTriggered {
	return WheneverPermanentEntersBattlefieldTrigger(effect, optional, IsLand)
}

// AnyCreatureDiesTrigger fires when any creature dies (regardless of controller).
// Fires on EvtZoneChange (BF -> GY) with a was-creature LKI predicate
// (CR 700.4 / 603.6c).
func AnyCreatureDiesTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtZoneChange, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
			EventSourceWasOfType{Type: TypeCreature},
		}})
}

// CreatureDealtDamageBySourceDiesTrigger fires when a creature that was dealt
// damage by the source permanent this turn dies (e.g. Sengir Vampire).
func CreatureDealtDamageBySourceDiesTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtZoneChange, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
			EventSourceWasOfType{Type: TypeCreature},
			EventSourceDamagedBySource{},
		}})
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

// WheneverYouGainLifeTrigger fires whenever the controller gains life
// (CR 119.9). evt.Amount is the amount gained.
func WheneverYouGainLifeTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtLifeGained, optional, effect).
		SetConditionData(EventPlayerIsController{})
}

// WheneverPlayerGainsLifeTrigger fires whenever any player gains life.
// evt.PlayerID identifies the gaining player; evt.Amount is the amount gained.
func WheneverPlayerGainsLifeTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtLifeGained, optional, effect)
}

// WheneverYouLoseLifeTrigger fires whenever the controller loses life
// (CR 119.9). evt.Amount is the amount lost.
func WheneverYouLoseLifeTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtLifeLost, optional, effect).
		SetConditionData(EventPlayerIsController{})
}

// WheneverOpponentLosesLifeTrigger fires whenever an opponent loses life
// (CR 119.9; used by Exquisite Blood). evt.Amount is the amount lost.
func WheneverOpponentLosesLifeTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtLifeLost, optional, effect).
		SetConditionData(EventPlayerIsOpponent{})
}

// WheneverPlayerLosesLifeTrigger fires whenever any player loses life.
// evt.PlayerID identifies the losing player; evt.Amount is the amount lost.
func WheneverPlayerLosesLifeTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtLifeLost, optional, effect)
}

// WheneverPlayerDiscardsTrigger fires whenever any player discards a card.
// evt.PlayerID is the discarding player; evt.SourceID is the discarded card ID.
func WheneverPlayerDiscardsTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDiscard, optional, effect)
}

// WheneverOpponentDiscardsTrigger fires whenever an opponent discards a card
// (used by Fell Specter, Sangromancer).
func WheneverOpponentDiscardsTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDiscard, optional, effect).
		SetConditionData(EventPlayerIsOpponent{})
}

// WheneverYouDiscardTrigger fires whenever the controller discards a card.
func WheneverYouDiscardTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDiscard, optional, effect).
		SetConditionData(EventPlayerIsController{})
}

// WheneverYouSacrificeAnotherCreatureTrigger fires whenever the controller
// sacrifices a creature other than the source permanent. The sacrificed
// permanent's identity is reported via evt.SourceID, the controller via
// evt.PlayerID, and creature-ness via evt.Flag.
func WheneverYouSacrificeAnotherCreatureTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtSacrifice, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventPlayerIsController{},
			EventSacrificedPermanentIsCreature{},
			EventSourceNotSelf{},
		}})
}

// WheneverYouSacrificeTrigger fires whenever the controller sacrifices any
// permanent (creature or otherwise), excluding the source itself.
func WheneverYouSacrificeTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtSacrifice, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventPlayerIsController{},
			EventSourceNotSelf{},
		}})
}

// WheneverBecomesTargetTrigger fires when the source permanent becomes the
// target of a spell or activated ability (CR 603.6c, 119.5). Used by cards
// like Departed Deckhand ("When this creature becomes the target of a spell
// or ability, sacrifice it").
func WheneverBecomesTargetTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtBecomesTarget, optional, effect).
		SetConditionData(EventTargetIsSelf{})
}

// WheneverBecomesTargetFirstTimeEachTurnTrigger fires the first time each turn
// the source permanent becomes the target of a spell or activated ability
// (CR 603.6c, 119.5). Used by Kira, Great Glass-Spinner.
func WheneverBecomesTargetFirstTimeEachTurnTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtBecomesTarget, optional, effect).
		SetConditionData(EventTargetIsSelfFirstTimeThisTurn{})
}

// WheneverDealsCombatDamageToPlayerTrigger fires when the source permanent
// deals combat damage to any player (CR 119.5). Used by Coastal Piracy-style
// "whenever a creature you control deals combat damage to a player" cards
// when targeted at the source itself; for filter-based triggers see
// WheneverPermanentDealsCombatDamageToPlayerTrigger.
func WheneverDealsCombatDamageToPlayerTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventSourceIsSelf{},
			EventTargetIsPlayer{},
			EventIsCombatDamage{},
		}})
}

// WheneverPermanentDealsCombatDamageToPlayerTrigger fires whenever any
// permanent matching the given filter (and controlled by the trigger's
// controller) deals combat damage to a player. Used by Coastal Piracy
// ("whenever a creature you control deals combat damage to a player") and
// Sharding Sphinx ("whenever an artifact creature you control deals combat
// damage to a player").
func WheneverPermanentDealsCombatDamageToPlayerTrigger(effect Effect, optional bool, filter PermanentFilter) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventTargetIsPlayer{},
			EventIsCombatDamage{},
			EventSourceControlledByController{},
			EventSourceMatchesPermanentFilter{Filter: filter},
		}})
}

// WheneverEnchantedPermanentDealsDamageToPlayerTrigger fires whenever the
// permanent this aura is attached to deals damage (combat or otherwise) to a
// player. Used by Curiosity ("when enchanted creature deals damage to a
// player, draw a card").
func WheneverEnchantedPermanentDealsDamageToPlayerTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtDamageDealt, optional, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventTargetIsPlayer{},
			EventSourceIsAttachedTo{},
		}})
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
