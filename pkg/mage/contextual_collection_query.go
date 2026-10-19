package mage

import "github.com/google/uuid"

// ContextualPermanentCollectionPredicate retains a complete collection declaration.
// It has no saved members. The zero value cannot select objects.
type ContextualPermanentCollectionPredicate struct {
	root *contextualCollectionNode
}

type contextualCollectionPlayer uint8

const (
	collectionAnyPlayer contextualCollectionPlayer = iota
	collectionController
	collectionActivePlayer
	collectionBoundPlayer
	collectionExplicitPlayer
)

type contextualCollectionNode struct {
	filter          *permanentCollectionFilterNode
	hosts           *contextualCollectionNode
	player          contextualCollectionPlayer
	explicitPlayer  uuid.UUID
	playerTarget    *targetDeclaration
	permanentTarget *targetDeclaration
	blockedByTarget *targetDeclaration
	owned           bool
	opponents       bool
	children        []*contextualCollectionNode
	intersection    bool
	difference      bool
}

// InspectContextualPermanentCollection reads complete type, subtype, color and combat
// collections with owner or controller restrictions. Player references can be
// You, ActivePlayer or a complete singular player target, including their native
// opponent sets. Unknown descendants reject the whole declaration.
// AttachmentsOn can use these complete collections as current host sets.
// Exact-count permanent targets and native filters can use a resolving target frame.
// BlockedByThisTurn accepts a complete native singular permanent target.
// The target frame supplies the exact blocker visit for turn-history reads.
// Saved and singular host references need separate inspection contracts.
// Inspection does not call collection, player or filter callbacks.
func InspectContextualPermanentCollection(subjects PermanentSet) (ContextualPermanentCollectionPredicate, bool) {
	root, ok := inspectContextualCollection(subjects)
	if !ok {
		return ContextualPermanentCollectionPredicate{}, false
	}
	return ContextualPermanentCollectionPredicate{root: root}, true
}

func inspectContextualCollection(subjects PermanentSet) (*contextualCollectionNode, bool) {
	return inspectContextualCollectionWithMember(subjects, nil)
}

func inspectContextualCollectionWithMember(subjects PermanentSet, member *PlayerBinding) (*contextualCollectionNode, bool) {
	switch s := subjects.(type) {
	case blockedThisTurnSet:
		predicate, ok := InspectPermanentTarget(s.blocker)
		if !ok {
			return nil, false
		}
		return &contextualCollectionNode{blockedByTarget: predicate.declaration}, true
	case *PermanentTargetsBinding:
		predicate, ok := InspectPermanentTargets(s)
		if !ok {
			return nil, false
		}
		return &contextualCollectionNode{permanentTarget: predicate.target.declaration}, true
	case filteredPermanentSet:
		subjects, ok := inspectContextualCollectionWithMember(s.subjects, member)
		if !ok {
			return nil, false
		}
		filter, ok := inspectContextualPermanentCollectionFilter(s.filter)
		if !ok {
			return nil, false
		}
		return &contextualCollectionNode{children: []*contextualCollectionNode{subjects, {filter: filter}}, intersection: true}, true
	case matchingPermanentSet:
		player := collectionAnyPlayer
		if s.controlled {
			player = collectionController
		}
		return contextualCollectionLeaf(s.filter, player, false)
	case controlledByPermanentSet:
		player, target, ok := inspectContextualCollectionPlayerWithMember(s.controller, member)
		if !ok {
			return nil, false
		}
		node, ok := contextualCollectionLeaf(s.filter, player, false)
		if ok {
			node.playerTarget = target
		}
		return node, ok
	case ownedPermanentSet:
		player, target, ok := inspectContextualCollectionPlayerWithMember(s.owner, member)
		if !ok {
			return nil, false
		}
		node, ok := contextualCollectionLeaf(s.filter, player, true)
		if ok {
			node.playerTarget = target
		}
		return node, ok
	case playerSetPermanents:
		opponents, ok := s.players.(*opponentsRef)
		if !ok || opponents == nil {
			return nil, false
		}
		player, target, ok := inspectContextualCollectionPlayerWithMember(opponents.player, member)
		if !ok {
			return nil, false
		}
		node, ok := contextualCollectionLeaf(s.filter, player, false)
		if !ok {
			return nil, false
		}
		node.opponents = true
		node.playerTarget = target
		return node, true
	case differencePermanentSet:
		included, ok := inspectContextualCollectionWithMember(s.subjects, member)
		if !ok {
			return nil, false
		}
		excluded, ok := inspectContextualCollectionWithMember(s.excluded, member)
		if !ok {
			return nil, false
		}
		return &contextualCollectionNode{children: []*contextualCollectionNode{included, excluded}, difference: true}, true
	case composedPermanentSet:
		if len(s.members) == 0 {
			return nil, false
		}
		children := make([]*contextualCollectionNode, len(s.members))
		for i, subject := range s.members {
			child, ok := inspectContextualCollectionWithMember(subject, member)
			if !ok {
				return nil, false
			}
			children[i] = child
		}
		return &contextualCollectionNode{children: children, intersection: s.intersection}, true
	case attachmentHostSet:
		hosts, ok := inspectContextualCollectionWithMember(s.hosts, member)
		if !ok {
			return nil, false
		}
		return &contextualCollectionNode{hosts: hosts}, true
	default:
		return nil, false
	}
}

func inspectContextualCollectionPlayerWithMember(player PlayerRef, member *PlayerBinding) (contextualCollectionPlayer, *targetDeclaration, bool) {
	if ref, ok := player.(savedPlayerRef); ok && member != nil && ref.binding == member {
		return collectionExplicitPlayer, nil, true
	}
	return inspectContextualCollectionPlayer(player)
}

func inspectContextualCollectionPlayer(player PlayerRef) (contextualCollectionPlayer, *targetDeclaration, bool) {
	switch player.(type) {
	case controllerRef:
		return collectionController, nil, true
	case activePlayerRef:
		return collectionActivePlayer, nil, true
	default:
		if predicate, ok := InspectPlayerTarget(player); ok {
			return collectionBoundPlayer, predicate.declaration, true
		}
		return collectionAnyPlayer, nil, false
	}
}

func contextualCollectionLeaf(filter PermanentFilter, player contextualCollectionPlayer, owned bool) (*contextualCollectionNode, bool) {
	query, ok := inspectContextualPermanentCollectionFilter(filter)
	if !ok {
		return nil, false
	}
	return &contextualCollectionNode{filter: query, player: player, owned: owned}, true
}

// Select reads current visits in battlefield order without callbacks or LKI.
// Supply current layers and a present resolving controller. Phased objects are
// absent. Combat membership is current. The query does not predict destruction
// or change game state.
// Attachments require a present host with the exact attached incarnation.
// Bound player and permanent targets require a resolving target frame.
// Permanent frames must contain only legal targets from the current resolution.
// Blocker history reads the supplied visit after departure or phasing. It does
// not check target legality again. Returned attackers cannot use earlier history.
// Missing bindings select
// no objects; proof queries must also check that all bindings are present.
func (p ContextualPermanentCollectionPredicate) Select(ctx *EffectContext) []PermanentVisit {
	result, _ := p.selectKnown(ctx)
	return result
}

func (p ContextualPermanentCollectionPredicate) selectKnown(ctx *EffectContext) ([]PermanentVisit, bool) {
	if p.root == nil || ctx == nil || ctx.Game == nil || ctx.Game.GetPlayer(ctx.Controller) == nil {
		return nil, false
	}
	if !p.root.bindingsKnown(ctx) {
		return nil, false
	}
	var result []PermanentVisit
	for _, permanent := range ctx.Game.zones.battlefield {
		if !permanent.PhasedOut && p.root.matches(permanent, ctx) {
			result = append(result, PermanentVisit{ID: permanent.ID(), Incarnation: permanent.incarnationID})
		}
	}
	return result, true
}

func (n *contextualCollectionNode) matches(permanent *Permanent, ctx *EffectContext) bool {
	if n.blockedByTarget != nil {
		values := ctx.targetBindings[n.blockedByTarget]
		return len(values) == 1 && ctx.Game.wasBlockedByThisTurn(values[0], permanent)
	}
	if n.permanentTarget != nil {
		for _, visit := range ctx.targetBindings[n.permanentTarget] {
			if visit.id == permanent.ID() && visit.incarnation == permanent.incarnationID {
				return true
			}
		}
		return false
	}
	if n.hosts != nil {
		for _, host := range ctx.Game.zones.battlefield {
			if !host.PhasedOut && host.ID() == permanent.AttachedTo && host.incarnationID == permanent.attachedIncarnation {
				return n.hosts.matches(host, ctx)
			}
		}
		return false
	}
	if n.filter != nil {
		if !n.filter.matches(permanent, ctx.Game) {
			return false
		}
		if n.player == collectionAnyPlayer {
			return true
		}
		id := ctx.Controller
		if n.player == collectionExplicitPlayer {
			id = n.explicitPlayer
		}
		if n.player == collectionActivePlayer {
			active := ctx.Game.ActivePlayerObj()
			if active == nil {
				return false
			}
			id = active.PlayerID()
		}
		if n.player == collectionBoundPlayer {
			values := ctx.targetBindings[n.playerTarget]
			if len(values) == 0 {
				return false
			}
			id = values[0].id
			if ctx.Game.GetPlayer(id) == nil {
				return false
			}
		}
		if n.opponents {
			controller := permanent.ControllerID()
			return controller != id && ctx.Game.GetPlayer(controller) != nil
		}
		if n.owned {
			return permanent.Card.Owner() == id
		}
		return permanent.ControllerID() == id
	}
	if n.difference {
		return n.children[0].matches(permanent, ctx) && !n.children[1].matches(permanent, ctx)
	}
	for _, child := range n.children {
		if child.matches(permanent, ctx) != n.intersection {
			return !n.intersection
		}
	}
	return n.intersection
}

func (n *contextualCollectionNode) bindingsKnown(ctx *EffectContext) bool {
	if n == nil {
		return false
	}
	if n.blockedByTarget != nil {
		values, ok := ctx.targetBindings[n.blockedByTarget]
		if !ok || len(values) > 1 {
			return false
		}
		if len(values) == 1 && (values[0].id == uuid.Nil || values[0].incarnation == uuid.Nil) {
			return false
		}
	}
	if n.permanentTarget != nil {
		values, ok := ctx.targetBindings[n.permanentTarget]
		if !ok {
			return false
		}
		for i, value := range values {
			if value.id == uuid.Nil || value.incarnation == uuid.Nil {
				return false
			}
			for _, previous := range values[:i] {
				if previous.id == value.id {
					return false
				}
			}
		}
	}
	if n.player == collectionBoundPlayer {
		values, ok := ctx.targetBindings[n.playerTarget]
		if !ok || len(values) > 1 {
			return false
		}
	}
	if n.hosts != nil && !n.hosts.bindingsKnown(ctx) {
		return false
	}
	for _, child := range n.children {
		if !child.bindingsKnown(ctx) {
			return false
		}
	}
	return true
}
