package mage

import (
	. "github.com/benprew/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// TriggerConditionData is a composable, data-driven predicate for trigger
// conditions. Each implementation checks one aspect of the event/game state.
// Compose with AndTriggerCond, OrTriggerCond, NotTriggerCond.
//
// Use AsTriggerCondition() to bridge into the closure-based TriggerCondition type.
type TriggerConditionData interface {
	CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
}

// AsTriggerCondition wraps a TriggerConditionData into a TriggerCondition closure.
func AsTriggerCondition(d TriggerConditionData) TriggerCondition {
	return func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
		return d.CheckTriggerCond(evt, g, sourceID, controllerID)
	}
}

// ---------------------------------------------------------------------------
// Atomic predicates
// ---------------------------------------------------------------------------

// EventSourceIsSelf checks evt.SourceID == sourceID.
type EventSourceIsSelf struct{}

func (EventSourceIsSelf) CheckTriggerCond(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
	return evt.SourceID == sourceID
}

// EventTargetIsSelf checks evt.TargetID == sourceID.
type EventTargetIsSelf struct{}

func (EventTargetIsSelf) CheckTriggerCond(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
	return evt.TargetID == sourceID
}

// EventPlayerIsController checks evt.PlayerID == controllerID.
type EventPlayerIsController struct{}

func (EventPlayerIsController) CheckTriggerCond(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
	return evt.PlayerID == controllerID
}

// EventPlayerIsNotController checks evt.PlayerID != controllerID.
type EventPlayerIsNotController struct{}

func (EventPlayerIsNotController) CheckTriggerCond(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
	return evt.PlayerID != controllerID
}

// EventPlayerIsOpponent is an alias for EventPlayerIsNotController, kept for
// readability when expressing "an opponent did X" triggers in 2-player games.
type EventPlayerIsOpponent struct{}

func (EventPlayerIsOpponent) CheckTriggerCond(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
	return evt.PlayerID != controllerID
}

// EventSacrificedPermanentIsCreature checks the Flag bit set by Game.Sacrifice
// to indicate the sacrificed permanent was a creature. Used for "whenever you
// sacrifice another creature" triggers.
type EventSacrificedPermanentIsCreature struct{}

func (EventSacrificedPermanentIsCreature) CheckTriggerCond(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return evt.Flag
}

// EventSourceNotSelf checks evt.SourceID != sourceID ("another" creature).
type EventSourceNotSelf struct{}

func (EventSourceNotSelf) CheckTriggerCond(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
	return evt.SourceID != sourceID
}

// EventSourceControlledByController checks that the permanent referenced by
// evt.SourceID is controlled by the trigger's controller. For events fired
// after the permanent has left the battlefield (e.g. EvtCreatureDied), it
// falls back to the LKI snapshot recorded by RemoveFromBattlefield.
type EventSourceControlledByController struct{}

func (EventSourceControlledByController) CheckTriggerCond(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	game, ok := g.(*Game)
	if !ok {
		return false
	}
	view := game.LookupObject(evt.SourceID)
	return view != nil && view.ViewController() == controllerID
}

// EventSourceControlledByOpponent checks that the permanent referenced by
// evt.SourceID is NOT controlled by the trigger's controller. For events
// fired after the permanent has left the battlefield (e.g. EvtCreatureDied),
// it falls back to the LKI snapshot.
type EventSourceControlledByOpponent struct{}

func (EventSourceControlledByOpponent) CheckTriggerCond(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	game, ok := g.(*Game)
	if !ok {
		return false
	}
	view := game.LookupObject(evt.SourceID)
	return view != nil && view.ViewController() != controllerID
}

// EventSourceHasType checks that the permanent at evt.SourceID has a card type.
type EventSourceHasType struct {
	Type CardType
}

func (c EventSourceHasType) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	if perm := g.FindPermanent(evt.SourceID); perm != nil {
		return perm.HasType(c.Type)
	}
	if game, ok := g.(*Game); ok {
		if lki := game.LKI(evt.SourceID); lki != nil {
			return lki.HasType(c.Type)
		}
	}
	return false
}

// EventSourceHasSubType checks that the permanent at evt.SourceID has a subtype.
type EventSourceHasSubType struct {
	SubType string
}

func (c EventSourceHasSubType) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	if perm := g.FindPermanent(evt.SourceID); perm != nil {
		return perm.HasSubType(c.SubType)
	}
	if game, ok := g.(*Game); ok {
		if lki := game.LKI(evt.SourceID); lki != nil {
			return lki.HasSubType(c.SubType)
		}
	}
	return false
}

// EventTargetHasSubType checks that the permanent at evt.TargetID has a subtype.
type EventTargetHasSubType struct {
	SubType string
}

func (c EventTargetHasSubType) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	perm := g.FindPermanent(evt.TargetID)
	return perm != nil && perm.HasSubType(c.SubType)
}

// EventSourceMatchesPermanentFilter checks the permanent at evt.SourceID against a filter.
type EventSourceMatchesPermanentFilter struct {
	Filter PermanentFilter
}

func (c EventSourceMatchesPermanentFilter) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	perm := g.FindPermanent(evt.SourceID)
	return perm != nil && c.Filter.Match(perm, g.(*Game))
}

// EventSourceWasOfType checks the type of an event's source object via the
// unified LKIView, transparently consulting either the live permanent or
// the LKI snapshot (CR 603.6c / 603.10). Used by leave-/dies-triggers
// fired on EvtZoneChange where the permanent may already be gone from the
// battlefield, and equally usable for ETB triggers where the source is
// still live.
type EventSourceWasOfType struct {
	Type CardType
}

func (c EventSourceWasOfType) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	game, ok := g.(*Game)
	if !ok {
		return false
	}
	view := game.LookupObject(evt.SourceID)
	if view == nil {
		return false
	}
	return view.ViewHasType(c.Type)
}

// EventSourceWasColor checks the current color of a live event source or its
// last-known color if it has left the battlefield.
type EventSourceWasColor struct {
	Color Color
}

func (c EventSourceWasColor) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	game, ok := g.(*Game)
	if !ok {
		return false
	}
	view := game.LookupObject(evt.SourceID)
	return view != nil && view.ViewHasColor(c.Color)
}

// EventSourceWasNotColor checks that a live event source, or its last-known
// battlefield state, did not have the specified color. Missing objects do not
// match.
type EventSourceWasNotColor struct {
	Color Color
}

func (c EventSourceWasNotColor) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	game, ok := g.(*Game)
	if !ok {
		return false
	}
	view := game.LookupObject(evt.SourceID)
	return view != nil && !view.ViewHasColor(c.Color)
}

// EventZoneChangeMatches checks that an EvtZoneChange event's FromZone and
// ToZone match. Pass ZoneAny in either field to skip that side of the check
// — useful for "when ~ leaves the battlefield" (To=ZoneAny) and "when ~
// enters the battlefield" (From=ZoneAny) triggers (CR 603.6c, 603.6d).
//
// Per CR 603.10 a single zone change is one event, so any leaves-/enters-
// trigger can be expressed as a ZoneChangeMatches over EvtZoneChange.
type EventZoneChangeMatches struct {
	From Zone // ZoneAny to skip the source check
	To   Zone // ZoneAny to skip the destination check
}

func (c EventZoneChangeMatches) CheckTriggerCond(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	if c.From != ZoneAny && evt.FromZone != c.From {
		return false
	}
	if c.To != ZoneAny && evt.ToZone != c.To {
		return false
	}
	return true
}

// SourceIsAttachedToEventSource checks that the source permanent is attached
// to the permanent in evt.SourceID (for aura triggers like "when enchanted
// creature becomes tapped").
type SourceIsAttachedToEventSource struct{}

func (SourceIsAttachedToEventSource) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return false
	}
	return src.AttachedTo == evt.SourceID
}

// SourceIsTapped checks that the source permanent is tapped.
type SourceIsTapped struct{}

func (SourceIsTapped) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && perm.Tapped
}

// SourceIsUntapped checks that the source permanent is untapped.
type SourceIsUntapped struct{}

func (SourceIsUntapped) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && !perm.Tapped
}

// SourceNotSummonSick checks the source doesn't have summoning sickness.
type SourceNotSummonSick struct{}

func (SourceNotSummonSick) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && !perm.HasAttr(AttrSummonSick)
}

// EventSourceDamagedBySource checks if the creature that died (evt.SourceID)
// was damaged by sourceID this turn (Sengir Vampire pattern).
type EventSourceDamagedBySource struct{}

func (EventSourceDamagedBySource) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	sources := g.GetDamageSources(evt.SourceID)
	return sources[sourceID]
}

// EventIsChosenPlayerUpkeep checks evt.PlayerID matches the source permanent's ChosenPlayer.
type EventIsChosenPlayerUpkeep struct{}

func (EventIsChosenPlayerUpkeep) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	perm := g.FindPermanent(sourceID)
	return perm != nil && evt.PlayerID == perm.ChosenPlayer
}

// HasAttackedThisTurnCond checks if the source has attacked this turn.
type HasAttackedThisTurnCond struct{}

func (HasAttackedThisTurnCond) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return g.HasAttackedThisTurn(sourceID)
}

// EventFlagIsFalse checks that evt.Flag is false (used to distinguish tap-cost
// abilities from non-tap abilities).
type EventFlagIsFalse struct{}

func (EventFlagIsFalse) CheckTriggerCond(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return !evt.Flag
}

// ResolvingSpellCastFromZone is true when the spell currently resolving was
// cast from the named zone (CR 601.2a). For ETB triggers on a permanent
// entering the battlefield as a result of casting it, this checks the zone
// the spell was cast from — e.g. ResolvingSpellCastFromZone{Zone: ZoneHand}
// implements "if you cast it from your hand". Returns false outside of spell
// resolution (abilities and other paths).
type ResolvingSpellCastFromZone struct {
	Zone Zone
}

func (c ResolvingSpellCastFromZone) CheckTriggerCond(_ *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	return g.ResolvingCastZone() == c.Zone
}

// ---------------------------------------------------------------------------
// Combinators
// ---------------------------------------------------------------------------

// AndTriggerCond requires all inner conditions to be true.
type AndTriggerCond struct {
	Conditions []TriggerConditionData
}

func (c AndTriggerCond) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	for _, inner := range c.Conditions {
		if !inner.CheckTriggerCond(evt, g, sourceID, controllerID) {
			return false
		}
	}
	return true
}

// OrTriggerCond requires any inner condition to be true.
type OrTriggerCond struct {
	Conditions []TriggerConditionData
}

func (c OrTriggerCond) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	for _, inner := range c.Conditions {
		if inner.CheckTriggerCond(evt, g, sourceID, controllerID) {
			return true
		}
	}
	return false
}

// NotTriggerCond negates an inner condition.
type NotTriggerCond struct {
	Inner TriggerConditionData
}

func (c NotTriggerCond) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	return !c.Inner.CheckTriggerCond(evt, g, sourceID, controllerID)
}

// ---------------------------------------------------------------------------
// Combat predicates
// ---------------------------------------------------------------------------

// SourceIsBlockedAttacker checks that the source is an attacker with at least
// one blocker assigned.
type SourceIsBlockedAttacker struct{}

func (SourceIsBlockedAttacker) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	group := g.CombatGroupFor(sourceID)
	return group != nil && group.Blocked
}

// SourceIsUnblockedAttacker checks that the source is an attacker with no
// blockers assigned.
type SourceIsUnblockedAttacker struct{}

func (SourceIsUnblockedAttacker) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	if !g.IsAttackingInCombat(sourceID) {
		return false
	}
	group := g.CombatGroupFor(sourceID)
	return group != nil && !group.Blocked
}

// SourceIsBlockingInCombat checks that the source is a declared blocker.
type SourceIsBlockingInCombat struct{}

func (SourceIsBlockingInCombat) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return g.IsBlockingInCombat(sourceID)
}

// SourceInCombat checks that the source is either attacking or blocking.
type SourceInCombat struct{}

func (SourceInCombat) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return g.IsAttackingInCombat(sourceID) || g.IsBlockingInCombat(sourceID)
}

// SourceBlockedByCreatureMatching checks that the source is an attacker and at
// least one of its blockers matches the given PermanentFilter.
type SourceBlockedByCreatureMatching struct {
	Filter PermanentFilter
}

func (c SourceBlockedByCreatureMatching) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	group := g.CombatGroupFor(sourceID)
	if group == nil || len(group.BlockerIDs) == 0 {
		return false
	}
	for _, bid := range group.BlockerIDs {
		blocker := g.FindPermanent(bid)
		if blocker != nil && c.Filter.Match(blocker, g.(*Game)) {
			return true
		}
	}
	return false
}

// SourceInCombatWithMatchingCreature checks that the source is in combat
// (attacking or blocking) and the opponent creature in the combat group
// matches the given PermanentFilter. When attacking, checks blockers; when
// blocking, checks the attacker.
type SourceInCombatWithMatchingCreature struct {
	Filter PermanentFilter
}

func (c SourceInCombatWithMatchingCreature) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	for _, group := range g.CombatGroups() {
		if group.AttackerID == sourceID {
			for _, bid := range group.BlockerIDs {
				blocker := g.FindPermanent(bid)
				if blocker != nil && c.Filter.Match(blocker, g.(*Game)) {
					return true
				}
			}
		}
		for _, bid := range group.BlockerIDs {
			if bid == sourceID {
				attacker := g.FindPermanent(group.AttackerID)
				if attacker != nil && c.Filter.Match(attacker, g.(*Game)) {
					return true
				}
			}
		}
	}
	return false
}

// SourceOnBattlefield checks that the source permanent is still on the battlefield.
type SourceOnBattlefield struct{}

func (SourceOnBattlefield) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return g.FindPermanent(sourceID) != nil
}

// CombatGroupCountEquals checks that there are exactly N combat groups.
type CombatGroupCountEquals struct {
	N int
}

func (c CombatGroupCountEquals) CheckTriggerCond(_ *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	return len(g.CombatGroups()) == c.N
}

// OpponentCastNthSpellOfType checks that an opponent cast a spell of the given
// type and it is the Nth or later such spell they cast this turn.
type OpponentCastNthSpellOfType struct {
	Type     CardType
	MinCount int
}

func (c OpponentCastNthSpellOfType) CheckTriggerCond(evt *GameEvent, g GameReader, _ uuid.UUID, controllerID uuid.UUID) bool {
	if evt.PlayerID == controllerID {
		return false
	}
	card := g.FindCardAnywhere(evt.SourceID)
	if card == nil || !card.HasType(c.Type) {
		return false
	}
	return g.GetInstantsCastThisTurn(evt.PlayerID) >= c.MinCount
}

// SourceAttackedOrBlockedThisTurn checks if the source attacked or blocked
// this turn (for "end of combat" triggers like Clockwork Beast).
type SourceAttackedOrBlockedThisTurn struct{}

func (SourceAttackedOrBlockedThisTurn) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return g.HasAttackedThisTurn(sourceID) || len(g.GetBlockedThisTurn(sourceID)) > 0
}

// ---------------------------------------------------------------------------
// Damage predicates
// ---------------------------------------------------------------------------

// EventSourceIsSelfDamageToPlayer checks evt.SourceID == sourceID and the
// target is a player (any player).
type EventSourceIsSelfDamageToPlayer struct{}

func (EventSourceIsSelfDamageToPlayer) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	return evt.SourceID == sourceID && g.GetPlayer(evt.TargetID) != nil
}

// EventSourceIsSelfDamageToOpponent checks evt.SourceID == sourceID and the
// target is an opponent (not the controller).
type EventSourceIsSelfDamageToOpponent struct{}

func (EventSourceIsSelfDamageToOpponent) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	if evt.SourceID != sourceID {
		return false
	}
	p := g.GetPlayer(evt.TargetID)
	return p != nil && p.PlayerID() != controllerID
}

// EventTargetIsPlayer checks the event's TargetID identifies a player.
type EventTargetIsPlayer struct{}

func (EventTargetIsPlayer) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	return g.GetPlayer(evt.TargetID) != nil
}

// EventIsCombatDamage checks evt.Flag, which is set by combat damage events.
type EventIsCombatDamage struct{}

func (EventIsCombatDamage) CheckTriggerCond(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return evt.Flag
}

// EventSourceIsAttachedTo checks that the source is attached to the event's
// SourceID (used for "enchanted creature deals damage" patterns where the
// source of the damage event must be the permanent the aura is attached to).
type EventSourceIsAttachedTo struct{}

func (EventSourceIsAttachedTo) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return false
	}
	return evt.SourceID == src.AttachedTo
}

// ---------------------------------------------------------------------------
// Becomes-target predicates (CR 603.6c, 119.5)
// ---------------------------------------------------------------------------

// EventTargetIsSelfFirstTimeThisTurn checks that the event's TargetID is the
// source permanent AND the source has been targeted exactly once this turn
// (i.e. this is the first time it has become the target this turn). Used for
// Kira, Great Glass-Spinner ("the first time each turn").
type EventTargetIsSelfFirstTimeThisTurn struct{}

func (EventTargetIsSelfFirstTimeThisTurn) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	if evt.TargetID != sourceID {
		return false
	}
	// fireBecomesTargetEvents increments the counter BEFORE firing the event,
	// so "first time" means the counter is exactly 1 when this condition runs.
	return g.TimesTargetedThisTurn(sourceID) == 1
}

// EventBecomesTargetSourceIsNotSelf checks that the event was caused by a
// spell or ability whose source is not the trigger's own source (so a creature
// doesn't trigger when it targets itself with its own ability).
type EventBecomesTargetSourceIsNotSelf struct{}

func (EventBecomesTargetSourceIsNotSelf) CheckTriggerCond(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
	return evt.SourceID != sourceID
}

// ---------------------------------------------------------------------------
// Battlefield state predicates
// ---------------------------------------------------------------------------

// ControllerHasNoPermanentMatching checks that the controller has no permanent
// on the battlefield matching the given filter.
type ControllerHasNoPermanentMatching struct {
	Filter PermanentFilter
}

func (c ControllerHasNoPermanentMatching) CheckTriggerCond(_ *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	for _, p := range g.FilterBattlefield(c.Filter) {
		if p.ControllerID() == controllerID {
			return false
		}
	}
	return true
}

// NoBattlefieldPermanentMatching checks that no permanent on the battlefield
// matches the given filter (global, any controller).
type NoBattlefieldPermanentMatching struct {
	Filter PermanentFilter
}

func (c NoBattlefieldPermanentMatching) CheckTriggerCond(_ *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	return !g.AnyBattlefield(c.Filter)
}

// CreatureDeathsOccurred checks that at least one creature died this turn.
type CreatureDeathsOccurred struct{}

func (CreatureDeathsOccurred) CheckTriggerCond(_ *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	return g.CreatureDeaths() > 0
}

// ---------------------------------------------------------------------------
// Spell-cast predicates
// ---------------------------------------------------------------------------

// SpellCastMatchesCardFilters checks that the spell referenced by evt.SourceID
// matches all given CardFilter predicates.
type SpellCastMatchesCardFilters struct {
	Filters []CardFilter
}

func (c SpellCastMatchesCardFilters) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	card := g.FindCardAnywhere(evt.SourceID)
	if card == nil {
		return false
	}
	for _, f := range c.Filters {
		if !f.Match(card) {
			return false
		}
	}
	return true
}

// SpellCastIsType checks that the spell referenced by evt.SourceID has a
// specific card type.
type SpellCastIsType struct {
	Type CardType
}

func (c SpellCastIsType) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	card := g.FindCardAnywhere(evt.SourceID)
	return card != nil && card.HasType(c.Type)
}

// OpponentCastSpellOfType checks that an opponent cast a spell of a given type.
type OpponentCastSpellOfType struct {
	Type CardType
}

func (c OpponentCastSpellOfType) CheckTriggerCond(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	if evt.PlayerID == controllerID {
		return false
	}
	card := g.FindCardAnywhere(evt.SourceID)
	return card != nil && card.HasType(c.Type)
}

// ControllerCastSpellOfType checks that the controller cast a spell of a given type.
type ControllerCastSpellOfType struct {
	Type CardType
}

func (c ControllerCastSpellOfType) CheckTriggerCond(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	if evt.PlayerID != controllerID {
		return false
	}
	card := g.FindCardAnywhere(evt.SourceID)
	return card != nil && card.HasType(c.Type)
}

// ---------------------------------------------------------------------------
// Event amount / flag predicates
// ---------------------------------------------------------------------------

// EventAmountGreaterThan checks evt.Amount > N.
type EventAmountGreaterThan struct {
	N int
}

func (c EventAmountGreaterThan) CheckTriggerCond(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return evt.Amount > c.N
}

// EventFlagIsTrue checks that evt.Flag is true.
type EventFlagIsTrue struct{}

func (EventFlagIsTrue) CheckTriggerCond(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return evt.Flag
}

// ---------------------------------------------------------------------------
// Aura/attachment predicates
// ---------------------------------------------------------------------------

// EventIsAttachedControllerUpkeep checks that the event player is the
// controller of the permanent the source is attached to (for aura upkeep triggers).
type EventIsAttachedControllerUpkeep struct{}

func (EventIsAttachedControllerUpkeep) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return false
	}
	host := g.FindPermanent(src.AttachedTo)
	if host == nil {
		return false
	}
	return evt.PlayerID == host.ControllerID()
}

// AttachedToDealsDamageToController checks that the permanent the source is
// attached to (evt.SourceID) dealt damage to the controller (evt.TargetID).
type AttachedToDealsDamageToController struct{}

func (AttachedToDealsDamageToController) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	if src == nil || src.AttachedTo == uuid.Nil {
		return false
	}
	return evt.SourceID == src.AttachedTo && evt.TargetID == controllerID
}

// AttachedToIsEventSource checks that the source is attached to the permanent
// in evt.SourceID, and that the event had no tap cost (Flag == false).
// Used for "whenever enchanted artifact is activated" patterns.
type AttachedToIsEventSourceNoTapCost struct{}

func (AttachedToIsEventSourceNoTapCost) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
	if evt.Flag {
		return false
	}
	src := g.FindPermanent(sourceID)
	if src == nil {
		return false
	}
	return evt.SourceID == src.AttachedTo
}

// SourceInOwnGraveyardWithCreaturesAbove checks that the source card is in its
// owner's graveyard with at least N creature cards above it. Used as the
// intervening-if for graveyard-functional triggers like Nether Shadow.
type SourceInOwnGraveyardWithCreaturesAbove struct {
	N int
}

func (c SourceInOwnGraveyardWithCreaturesAbove) CheckTriggerCond(_ *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
	p := g.GetPlayer(controllerID)
	if p == nil {
		return false
	}
	graveyard := p.Graveyard()
	idx := -1
	for i, card := range graveyard {
		if card.ID() == sourceID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	above := 0
	for j := idx + 1; j < len(graveyard); j++ {
		if graveyard[j].HasType(TypeCreature) {
			above++
		}
	}
	return above >= c.N
}

// OpponentActivatedArtifactNoTapCost checks that an opponent activated an
// artifact ability without a tap cost.
type OpponentActivatedArtifactNoTapCost struct{}

func (OpponentActivatedArtifactNoTapCost) CheckTriggerCond(evt *GameEvent, g GameReader, _ uuid.UUID, controllerID uuid.UUID) bool {
	if evt.Flag {
		return false
	}
	if evt.PlayerID == controllerID {
		return false
	}
	perm := g.FindPermanent(evt.SourceID)
	return perm != nil && perm.HasType(TypeArtifact)
}
