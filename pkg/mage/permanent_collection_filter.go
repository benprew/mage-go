package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentCollectionFilterOperation uint8

const (
	collectionFilterAll permanentCollectionFilterOperation = iota
	collectionFilterAny
	collectionFilterNot
	collectionFilterTypes
	collectionFilterSubtype
	collectionFilterColor
	collectionFilterAttacking
	collectionFilterBlocking
	collectionFilterTapped
	collectionFilterPowerEqual
	collectionFilterToughnessEqual
	collectionFilterToughnessGTE
	collectionFilterAttachment
	collectionFilterKeyword
	collectionFilterRampage
	collectionFilterAttackedThisTurn
	collectionFilterCurrentTurnControl
	collectionFilterManaAbility
)

type permanentCollectionFilterNode struct {
	operation  permanentCollectionFilterOperation
	types      PermanentCardTypePredicate
	subtype    string
	color      core.Color
	keyword    core.Attr
	limit      int
	children   []*permanentCollectionFilterNode
	contextual bool
}

func inspectPermanentCollectionFilter(filter PermanentFilter) (*permanentCollectionFilterNode, bool) {
	node, ok := inspectContextualPermanentCollectionFilter(filter)
	if !ok || node.contextual {
		return nil, false
	}
	return node, true
}

func inspectContextualPermanentCollectionFilter(filter PermanentFilter) (*permanentCollectionFilterNode, bool) {
	if types, ok := InspectPermanentCardTypeFilter(filter); ok {
		return &permanentCollectionFilterNode{operation: collectionFilterTypes, types: types}, true
	}
	return filter.query, filter.query != nil
}

func composePermanentCollectionFilters(operation permanentCollectionFilterOperation, filters []PermanentFilter) *permanentCollectionFilterNode {
	children := make([]*permanentCollectionFilterNode, len(filters))
	contextual := false
	for i, filter := range filters {
		child, ok := inspectContextualPermanentCollectionFilter(filter)
		if !ok {
			return nil
		}
		children[i] = child
		contextual = contextual || child.contextual
	}
	return &permanentCollectionFilterNode{operation: operation, children: children, contextual: contextual}
}

func (n *permanentCollectionFilterNode) matches(p *Permanent, g *Game) bool {
	switch n.operation {
	case collectionFilterManaAbility:
		return permanentHasManaAbility(p)
	case collectionFilterCurrentTurnControl:
		return p != nil && g != nil && !p.PhasedOut && p.turnControlGained < g.CurrentTurn()
	case collectionFilterAttackedThisTurn:
		return p != nil && g != nil && g.HasAttackedThisTurnIncarnation(p.ID(), p.incarnationID)
	case collectionFilterRampage:
		return permanentHasRampage(p, g)
	case collectionFilterKeyword:
		return p.HasKeyword(n.keyword)
	case collectionFilterAttachment:
		host := currentAttachmentHost(p, g)
		return host != nil && n.children[0].matches(host, g)
	case collectionFilterTypes:
		return n.types.root.matchesCurrentPermanent(p)
	case collectionFilterSubtype:
		return p.HasSubType(n.subtype)
	case collectionFilterColor:
		return slices.Contains(p.Colors(), n.color)
	case collectionFilterPowerEqual:
		return p.CurrentPower(g) == n.limit
	case collectionFilterToughnessEqual:
		return p.CurrentToughness(g) == n.limit
	case collectionFilterToughnessGTE:
		return p.CurrentToughness(g) >= n.limit
	case collectionFilterTapped:
		return p.Tapped
	case collectionFilterAttacking:
		return g != nil && g.combat != nil && g.combat.IsAttacking(p.ID())
	case collectionFilterBlocking:
		return g != nil && g.combat != nil && g.combat.IsBlocking(p.ID())
	case collectionFilterNot:
		return !n.children[0].matches(p, g)
	case collectionFilterAll:
		for _, child := range n.children {
			if !child.matches(p, g) {
				return false
			}
		}
		return true
	case collectionFilterAny:
		for _, child := range n.children {
			if child.matches(p, g) {
				return true
			}
		}
	}
	return false
}

func (n *permanentCardTypeNode) matchesCurrentPermanent(p *Permanent) bool {
	if n == nil {
		return true
	}
	switch n.operation {
	case cardTypesHas:
		return p.HasType(n.kind)
	case cardTypesNot:
		return !n.children[0].matchesCurrentPermanent(p)
	case cardTypesAll:
		for _, child := range n.children {
			if !child.matchesCurrentPermanent(p) {
				return false
			}
		}
		return true
	case cardTypesAny:
		for _, child := range n.children {
			if child.matchesCurrentPermanent(p) {
				return true
			}
		}
	}
	return false
}
