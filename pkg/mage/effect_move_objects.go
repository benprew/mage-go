package mage

import (
	"fmt"
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type battlefieldEntryOptions struct {
	tapped bool
	from   Zone
}

type moveOptions struct {
	bottom      bool
	sacrifice   bool
	sacrificeBy PlayerSelector
	tapped      bool
	controller  PlayerSelector
}

// MoveOpt configures the destination or method of a zone change.
type MoveOpt func(*moveOptions)

// ToLibraryBottom puts moved cards on the bottom of the library.
func ToLibraryBottom() MoveOpt { return func(options *moveOptions) { options.bottom = true } }

// AsSacrifice requires the resolving controller to sacrifice selected permanents.
func AsSacrifice() MoveOpt { return func(options *moveOptions) { options.sacrifice = true } }

// EnterTapped makes moved permanents enter tapped.
func EnterTapped() MoveOpt { return func(options *moveOptions) { options.tapped = true } }

// UnderControlOf selects the controller of permanents that enter the battlefield.
func UnderControlOf(player PlayerSelector) MoveOpt {
	return func(options *moveOptions) { options.controller = player }
}

type moveObjectsEffect struct {
	objects     ObjectSelector
	destination Zone
	options     moveOptions
}

// MoveObjects moves the selected objects to the supplied zone.
func MoveObjects(objects ObjectSelector, destination Zone, options ...MoveOpt) Effect {
	effect := &moveObjectsEffect{objects: objects, destination: destination}
	for _, option := range options {
		option(&effect.options)
	}
	return effect
}
func (e *moveObjectsEffect) Text() string {
	_, target := e.objects.(legacyFirstReference)
	if e.options.sacrifice {
		if reference, ok := e.objects.(objectReference); ok && reference.kind == "source" {
			return "sacrifice this permanent"
		}
		if target {
			return "sacrifice target permanent"
		}
		return "sacrifice that permanent"
	}
	switch e.destination {
	case ZoneHand:
		if target {
			return "return target permanent to its owner's hand"
		}
		return "return those objects to their owners' hands"
	case ZoneExile:
		if target {
			return "exile target permanent"
		}
		return "exile those objects"
	case ZoneGraveyard:
		return "put those objects into their owners' graveyards"
	case ZoneLibrary:
		if e.options.bottom {
			return "put those objects on the bottom of their owners' libraries"
		}
		return "put those objects on top of their owners' libraries"
	case ZoneBattlefield:
		if e.options.tapped {
			return "put those cards onto the battlefield tapped"
		}
		return "put those cards onto the battlefield"
	}
	return "move those objects"
}
func (e *moveObjectsEffect) Properties() EffectProperties {
	if e.options.sacrifice {
		if reference, ok := e.objects.(objectReference); ok && reference.kind == "source" {
			return EffectProperties{SacrificesSource: true}
		}
		return EffectProperties{}
	}
	return EffectProperties{Outcome: OutcomeDetriment, IsBounce: e.destination == ZoneHand}
}

// AsSacrificeBy restricts sacrifices to permanents controlled by the selected players.
func AsSacrificeBy(player PlayerSelector) MoveOpt {
	return func(options *moveOptions) { options.sacrifice = true; options.sacrificeBy = player }
}

func (e *moveObjectsEffect) Apply(ctx *EffectContext) error {
	switch e.destination {
	case ZoneHand, ZoneGraveyard, ZoneExile, ZoneLibrary, ZoneBattlefield:
	default:
		return fmt.Errorf("unsupported object destination %s", e.destination)
	}
	if e.options.sacrifice && e.destination != ZoneGraveyard {
		return fmt.Errorf("sacrifice must move to the graveyard")
	}
	if e.destination == ZoneBattlefield {
		controller := ctx.Controller
		if e.options.controller != nil {
			ids := e.options.controller.Select(ctx)
			if len(ids) != 1 {
				return fmt.Errorf("battlefield entry requires one controller")
			}
			controller = ids[0]
		}
		if ctx.Game.GetPlayer(controller) == nil {
			return ErrPlayerNotFound
		}
	}
	ids := slices.Clone(currentObjectIDs(ctx, e.objects))
	graveyardCounts := make(map[uuid.UUID]int)
	seen := make(map[uuid.UUID]bool)
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		if permanent := ctx.Game.FindPermanent(id); permanent != nil {
			if e.options.sacrifice {
				allowed := []uuid.UUID{ctx.Controller}
				if e.options.sacrificeBy != nil {
					allowed = e.options.sacrificeBy.Select(ctx)
				}
				if slices.Contains(allowed, permanent.ControllerID()) {
					ctx.Game.DoSacrifice(permanent)
				}
				continue
			}
			switch e.destination {
			case ZoneHand:
				ctx.Game.BouncePermanentToHand(permanent)
			case ZoneGraveyard:
				ctx.Game.PutPermanentIntoGraveyard(permanent)
			case ZoneExile:
				ctx.Game.ExilePermanent(permanent)
			case ZoneBattlefield:
				continue
			case ZoneLibrary:
				owner := permanent.Card.Owner()
				controller := permanent.ControllerID()
				if owner == uuid.Nil() {
					owner = controller
				}
				ctx.Game.RemoveFromBattlefield(permanent)
				if !permanent.IsToken {
					e.putCard(ctx, permanent.Card, owner, ZoneBattlefield)
				}
				event := GameEvent{Type: EvtZoneChange, SourceID: id, PlayerID: controller, FromZone: ZoneBattlefield, ToZone: ZoneLibrary}
				ctx.Game.FireEvent(event)
				ctx.Game.checkAbilitiesForEvent(ctx.Game.LKIAbilities(id), &event, id, controller)
			}
			continue
		}
		if e.options.sacrifice {
			continue
		}
		object := ctx.Game.stack.FindBySourceID(id)
		if object == nil {
			object = ctx.Game.stack.FindByID(id)
		}
		if object != nil && !object.IsAbility && object.Card != nil {
			move := *e
			if object.ExileOnLeaveStack {
				move.destination = ZoneExile
			}
			if move.destination == ZoneBattlefield && (object.Card.HasType(TypeInstant) || object.Card.HasType(TypeSorcery)) {
				continue
			}
			if move.destination == ZoneBattlefield && object.Card.HasType(TypeLand) && ctx.Game.effects.Rules.LandsCantEnter() {
				continue
			}
			ctx.Game.stack.RemoveByID(object.ID)
			if !object.IsCopy {
				owner := object.Card.Owner()
				if owner == uuid.Nil() {
					owner = object.Controller
				}
				move.putCard(ctx, object.Card, owner, ZoneStack, object.ColorOverride)
				if move.destination != ZoneBattlefield {
					ctx.Game.FireEvent(GameEvent{Type: EvtZoneChange, SourceID: object.Card.ID(), PlayerID: object.Controller, FromZone: ZoneStack, ToZone: move.destination})
				}
			}
			continue
		}
		card := ctx.Game.FindCardAnywhere(id)
		if card == nil {
			continue
		}
		owner := card.Owner()
		from := ctx.Game.CardZone(id)
		if from == e.destination {
			continue
		}
		var removed *Card
		holder := owner
		for _, player := range ctx.Game.AllPlayers() {
			if ctx.Game.findCardInZoneOpt(player.PlayerID(), id, from, true) != nil {
				holder = player.PlayerID()
				break
			}
		}
		if e.destination == ZoneBattlefield && card.HasType(TypeLand) && ctx.Game.effects.Rules.LandsCantEnter() {
			continue
		}
		if from == ZoneGraveyard {
			if player := ctx.Game.GetPlayer(holder); player != nil {
				removed, _ = player.RemoveFromGraveyard(id)
			}
			if removed != nil {
				graveyardCounts[holder]++
			}
		} else {
			removed = ctx.Game.removeCardFromZone(holder, id, from)
		}
		if removed == nil {
			continue
		}
		if owner == uuid.Nil() {
			owner = holder
		}
		e.putCard(ctx, removed, owner, from)
		if e.destination != ZoneBattlefield {
			ctx.Game.FireEvent(GameEvent{Type: EvtZoneChange, SourceID: id, PlayerID: holder, FromZone: from, ToZone: e.destination})
		}
	}
	for _, player := range ctx.Game.AllPlayers() {
		if count := graveyardCounts[player.PlayerID()]; count > 0 {
			ctx.Game.FireEvent(GameEvent{Type: EvtCardsLeftGraveyard, PlayerID: player.PlayerID(), Amount: count})
		}
	}
	return nil
}
func (e *moveObjectsEffect) putCard(ctx *EffectContext, card *Card, owner uuid.UUID, from Zone, colors ...*[]Color) {
	if card.Owner() == uuid.Nil() {
		card.SetOwner(owner)
	}
	player := ctx.Game.GetPlayer(owner)
	if player == nil {
		return
	}
	switch e.destination {
	case ZoneHand:
		player.AddToHand(card)
	case ZoneGraveyard:
		player.AddToGraveyard(card)
	case ZoneExile:
		ctx.Game.ExileCard(card, ctx.SourceID)
	case ZoneLibrary:
		if e.options.bottom {
			player.SetLibrary(append(slices.Clone(player.Library()), card))
		} else {
			player.SetLibrary(append([]*Card{card}, player.Library()...))
		}
	case ZoneBattlefield:
		controller := ctx.Controller
		if e.options.controller != nil {
			ids := e.options.controller.Select(ctx)
			if len(ids) == 0 {
				return
			}
			controller = ids[0]
		}
		var override *[]Color
		if len(colors) > 0 {
			override = colors[0]
		}
		ctx.Game.putOnBattlefield(card, controller, override, card.ID(), false, battlefieldEntryOptions{tapped: e.options.tapped, from: from})
	}
}
