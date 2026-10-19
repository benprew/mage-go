package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentCardTypeOperation uint8

const (
	cardTypesAll permanentCardTypeOperation = iota
	cardTypesAny
	cardTypesNot
	cardTypesHas
)

type permanentCardTypeNode struct {
	operation permanentCardTypeOperation
	kind      core.CardType
	children  []*permanentCardTypeNode
}

// PermanentCardTypePredicate is an immutable predicate on a supplied type set.
// The zero value matches every set. It does not select battlefield objects.
type PermanentCardTypePredicate struct{ root *permanentCardTypeNode }

// InspectPermanentCardTypeFilter reads a complete typed card-type predicate.
// Opaque filters return false, including unknown Boolean children.
// Inspection does not call filter callbacks or read game state.
func InspectPermanentCardTypeFilter(filter PermanentFilter) (PermanentCardTypePredicate, bool) {
	if filter.fn == nil {
		return PermanentCardTypePredicate{}, true
	}
	if filter.cardTypes == nil {
		return PermanentCardTypePredicate{}, false
	}
	return PermanentCardTypePredicate{root: filter.cardTypes}, true
}

// Matches tests the supplied card types without callbacks or game state.
// The caller supplies current types after applicable continuous effects.
func (p PermanentCardTypePredicate) Matches(types ...core.CardType) bool {
	return p.root.matches(types)
}

func (n *permanentCardTypeNode) matches(types []core.CardType) bool {
	if n == nil {
		return true
	}
	switch n.operation {
	case cardTypesHas:
		return slices.Contains(types, n.kind)
	case cardTypesNot:
		return !n.children[0].matches(types)
	case cardTypesAll:
		for _, child := range n.children {
			if !child.matches(types) {
				return false
			}
		}
		return true
	case cardTypesAny:
		for _, child := range n.children {
			if child.matches(types) {
				return true
			}
		}
	}
	return false
}

// HasCardTypeFilter tests the current card type and retains a pure predicate.
func HasCardTypeFilter(kind core.CardType) PermanentFilter {
	return permanentCardTypeFilter(kind.String(), &permanentCardTypeNode{operation: cardTypesHas, kind: kind})
}

func permanentCardTypeFilter(label string, node *permanentCardTypeNode) PermanentFilter {
	return PermanentFilter{label: label, cardTypes: node, fn: func(p *Permanent, _ *Game) bool { return node.matchesPermanent(p) }}
}

func (n *permanentCardTypeNode) matchesPermanent(p *Permanent) bool {
	if n.operation == cardTypesHas {
		return p.HasType(n.kind)
	}
	return n.matches(nil)
}

func composePermanentCardTypes(operation permanentCardTypeOperation, filters []PermanentFilter) *permanentCardTypeNode {
	children := make([]*permanentCardTypeNode, len(filters))
	for i, filter := range filters {
		predicate, ok := InspectPermanentCardTypeFilter(filter)
		if !ok {
			return nil
		}
		children[i] = predicate.root
	}
	return &permanentCardTypeNode{operation: operation, children: children}
}
