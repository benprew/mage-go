package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type cardCharacteristicOperation uint8

const (
	cardCharacteristicAll cardCharacteristicOperation = iota
	cardCharacteristicAny
	cardCharacteristicType
	cardCharacteristicColor
	cardCharacteristicSubtype
	cardCharacteristicTargetOnlySubtype
)

type cardCharacteristicNode struct {
	operation cardCharacteristicOperation
	kind      core.CardType
	color     core.Color
	subtype   string
	children  []*cardCharacteristicNode
}

func (n *cardCharacteristicNode) matches(card Card) bool {
	switch n.operation {
	case cardCharacteristicType:
		return card.HasType(n.kind)
	case cardCharacteristicColor:
		return slices.Contains(cardColors(card), n.color)
	case cardCharacteristicTargetOnlySubtype:
		return TargetSourceCanTargetOnlySubtype(card, n.subtype)
	case cardCharacteristicSubtype:
		return card.HasSubType(n.subtype)
	case cardCharacteristicAll:
		for _, child := range n.children {
			if !child.matches(card) {
				return false
			}
		}
		return true
	case cardCharacteristicAny:
		for _, child := range n.children {
			if child.matches(card) {
				return true
			}
		}
	}
	return false
}

func typedCardFilter(label string, node *cardCharacteristicNode) CardFilter {
	return CardFilter{label: label, characteristics: node, fn: func(card Card) bool { return node.matches(card) }}
}

// AnyCardFilter retains an unrestricted characteristic predicate.
func AnyCardFilter() CardFilter {
	return typedCardFilter("any card", &cardCharacteristicNode{operation: cardCharacteristicAll})
}

// HasCardTypeCardFilter retains a predicate on the source card's type.
func HasCardTypeCardFilter(kind core.CardType) CardFilter {
	return typedCardFilter(kind.String()+" card", &cardCharacteristicNode{operation: cardCharacteristicType, kind: kind})
}

func composeCardCharacteristics(operation cardCharacteristicOperation, filters []CardFilter) *cardCharacteristicNode {
	children := make([]*cardCharacteristicNode, len(filters))
	for i, filter := range filters {
		if filter.fn == nil {
			children[i] = &cardCharacteristicNode{operation: cardCharacteristicAll}
			continue
		}
		if filter.characteristics == nil {
			return nil
		}
		children[i] = filter.characteristics
	}
	return &cardCharacteristicNode{operation: operation, children: children}
}

// OrCardFilter accepts a card that matches at least one complete predicate.
func OrCardFilter(filters ...CardFilter) CardFilter {
	filters = slices.Clone(filters)
	return CardFilter{label: "matching card", characteristics: composeCardCharacteristics(cardCharacteristicAny, filters), fn: func(card Card) bool {
		for _, filter := range filters {
			if filter.Match(card) {
				return true
			}
		}
		return false
	}}
}
