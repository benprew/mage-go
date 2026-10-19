package mage

import "github.com/google/uuid"

type playerCollectionKind uint8

const (
	playerCollectionInvalid playerCollectionKind = iota
	playerCollectionNone
	playerCollectionAll
	playerCollectionSingle
	playerCollectionOpponents
)

// PlayerCollectionPredicate retains a complete native player set declaration.
// The zero value cannot select players.
type PlayerCollectionPredicate struct {
	kind   playerCollectionKind
	player contextualCollectionPlayer
	target *targetDeclaration
}

// InspectPlayerCollection reads native all, single and opponent player sets.
// References can be You, ActivePlayer or a singular native player target.
// Unknown references reject the whole set. No callback is called.
func InspectPlayerCollection(players PlayerSet) (PlayerCollectionPredicate, bool) {
	var reference PlayerRef
	var kind playerCollectionKind
	switch set := players.(type) {
	case allPlayersRef:
		return PlayerCollectionPredicate{kind: playerCollectionAll}, true
	case noDamagePlayers:
		return PlayerCollectionPredicate{kind: playerCollectionNone}, true
	case singlePlayerSet:
		kind, reference = playerCollectionSingle, set.player
	case *opponentsRef:
		if set == nil {
			return PlayerCollectionPredicate{}, false
		}
		kind, reference = playerCollectionOpponents, set.player
	default:
		return PlayerCollectionPredicate{}, false
	}
	player, target, ok := inspectContextualCollectionPlayer(reference)
	if !ok {
		return PlayerCollectionPredicate{}, false
	}
	return PlayerCollectionPredicate{kind: kind, player: player, target: target}, true
}

func (p PlayerCollectionPredicate) bindingsKnown(ctx *EffectContext) bool {
	if p.kind == playerCollectionInvalid || ctx == nil || ctx.Game == nil || ctx.Game.GetPlayer(ctx.Controller) == nil {
		return false
	}
	if p.player == collectionBoundPlayer {
		values, ok := ctx.targetBindings[p.target]
		if !ok || len(values) > 1 {
			return false
		}
		if len(values) == 1 && (values[0].id == uuid.Nil || values[0].incarnation != uuid.Nil) {
			return false
		}
	}
	return true
}

// Select reads present players in the same order as native execution.
// Bound targets require a complete legal resolving target frame. An empty
// binding is known and selects no players, including for an opponent set.
// Missing bindings are unknown. The query uses no LKI and changes no state.
func (p PlayerCollectionPredicate) Select(ctx *EffectContext) ([]uuid.UUID, bool) {
	if !p.bindingsKnown(ctx) {
		return nil, false
	}
	if p.kind == playerCollectionNone {
		return nil, true
	}
	var subject uuid.UUID
	if p.kind != playerCollectionAll {
		switch p.player {
		case collectionController:
			subject = ctx.Controller
		case collectionActivePlayer:
			if active := ctx.Game.ActivePlayerObj(); active != nil {
				subject = active.PlayerID()
			}
		case collectionBoundPlayer:
			if values := ctx.targetBindings[p.target]; len(values) == 1 {
				subject = values[0].id
			}
		}
		if ctx.Game.GetPlayer(subject) == nil {
			return nil, true
		}
		if p.kind == playerCollectionSingle {
			return []uuid.UUID{subject}, true
		}
	}
	var result []uuid.UUID
	appendPlayer := func(id uuid.UUID) {
		if p.kind == playerCollectionAll || id != subject {
			result = append(result, id)
		}
	}
	active := ctx.Game.ActivePlayerObj()
	if active != nil {
		appendPlayer(active.PlayerID())
	}
	for _, player := range ctx.Game.players {
		if active == nil || player.PlayerID() != active.PlayerID() {
			appendPlayer(player.PlayerID())
		}
	}
	return result, true
}
