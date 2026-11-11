package mage

import (
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TriggerSpec tests an event without storing an ability binding.
type TriggerSpec interface {
	MatchesEventType(EventType) bool
	MatchesEvent(*GameEvent, AbilityContext, GameReader) bool
}

type eventTriggerSpec struct {
	event    EventType
	from, to Zone
	objects  ObjectFilter
	players  PlayerSelector
	spell    bool
}

func (s eventTriggerSpec) MatchesEventType(t EventType) bool { return t == s.event }
func (s eventTriggerSpec) MatchesEvent(evt *GameEvent, ctx AbilityContext, g GameReader) bool {
	if evt == nil || !s.MatchesEventType(evt.Type) {
		return false
	}
	if s.event == EvtZoneChange && ((s.from != ZoneAny && evt.FromZone != s.from) || (s.to != ZoneAny && evt.ToZone != s.to)) {
		return false
	}
	if s.players != nil && !slices.Contains(s.players.Select(filterEffectContext(ctx, g)), evt.PlayerID) {
		return false
	}
	if s.event == EvtUpkeep || s.event == EvtEndStep {
		return true
	}
	if s.spell {
		card := g.FindCardAnywhere(evt.SourceID)
		if card == nil {
			return false
		}
		return s.objects.Match(spellObjectView{Card: card, controller: evt.PlayerID}, g, ctx)
	}
	object := g.LookupObject(evt.SourceID)

	return s.objects.Match(object, g, ctx)
}

// WhenPermanentEntersBattlefield tests the object after entry.
func WhenPermanentEntersBattlefield(filter ObjectFilter) TriggerSpec {
	return eventTriggerSpec{event: EvtZoneChange, from: ZoneAny, to: ZoneBattlefield, objects: filter}
}

// WhenPermanentDies tests the object before it leaves the battlefield.
func WhenPermanentDies(filter ObjectFilter) TriggerSpec {
	return eventTriggerSpec{event: EvtZoneChange, from: ZoneBattlefield, to: ZoneGraveyard, objects: filter}
}

// WhenPermanentLeavesBattlefield tests the object before it leaves.
func WhenPermanentLeavesBattlefield(filter ObjectFilter) TriggerSpec {
	return eventTriggerSpec{event: EvtZoneChange, from: ZoneBattlefield, to: ZoneAny, objects: filter}
}

// WhenCreatureAttacks tests each declared attacking creature.
func WhenCreatureAttacks(filter ObjectFilter) TriggerSpec {
	return eventTriggerSpec{event: EvtDeclaredAttacker, objects: ObjectAnd(IsCardType(TypeCreature), filter)}
}

// WhenPlayerCastsSpell tests the caster and the spell.
func WhenPlayerCastsSpell(player PlayerSelector, spell ObjectFilter) TriggerSpec {
	return eventTriggerSpec{event: EvtSpellCast, players: player, objects: spell, spell: true}
}

// AtBeginningOfUpkeep tests the active player at upkeep.
func AtBeginningOfUpkeep(player PlayerSelector) TriggerSpec {
	return eventTriggerSpec{event: EvtUpkeep, players: player}
}

// AtBeginningOfEndStep tests the active player at the end step.
func AtBeginningOfEndStep(player PlayerSelector) TriggerSpec {
	return eventTriggerSpec{event: EvtEndStep, players: player}
}

type orTriggerSpec struct{ specs []TriggerSpec }

// TriggerOr matches an event if any child specification matches it.
func TriggerOr(specs ...TriggerSpec) TriggerSpec { return orTriggerSpec{specs: slices.Clone(specs)} }
func (s orTriggerSpec) MatchesEventType(t EventType) bool {
	for _, child := range s.specs {
		if child.MatchesEventType(t) {
			return true
		}
	}
	return false
}
func (s orTriggerSpec) MatchesEvent(evt *GameEvent, ctx AbilityContext, g GameReader) bool {
	for _, child := range s.specs {
		if child.MatchesEvent(evt, ctx, g) {
			return true
		}
	}
	return false
}

// NewTriggeredAbility combines an event specification with effects and targets.
func NewTriggeredAbility(spec TriggerSpec, parts ...any) *GenericTriggered {
	if spec == nil {
		panic("mage: nil trigger specification")
	}
	action := NewAction(ActionSpell, actionPartsToOptions(parts...)...)
	trigger := NewTriggered(0, false, action.effects...)
	trigger.spec = spec
	trigger.targets = action.targets
	return trigger
}
