package mage

import "github.com/google/uuid"

// PermanentVisit identifies one battlefield object. A return has a new visit.
type PermanentVisit struct {
	ID          uuid.UUID
	Incarnation uuid.UUID
}

// PermanentCollectionPredicate describes a complete type, subtype and color collection.
// The zero value supplies no members. Inspection must succeed before use.
type PermanentCollectionPredicate struct {
	filter *permanentCollectionFilterNode
}

// InspectPermanentCollection reads type, subtype and color matching, union and intersection
// declarations. Unknown children reject the whole query without callbacks.
func InspectPermanentCollection(subjects PermanentSet) (PermanentCollectionPredicate, bool) {
	filter, ok := inspectPermanentCollectionMembers(subjects)
	if !ok {
		return PermanentCollectionPredicate{}, false
	}
	return PermanentCollectionPredicate{filter: filter}, true
}

func inspectPermanentCollectionMembers(subjects PermanentSet) (*permanentCollectionFilterNode, bool) {
	switch subjects := subjects.(type) {
	case matchingPermanentSet:
		if subjects.controlled {
			return nil, false
		}
		return inspectPermanentCollectionFilter(subjects.filter)
	case composedPermanentSet:
		if len(subjects.members) == 0 {
			return nil, false
		}
		operation := collectionFilterAny
		if subjects.intersection {
			operation = collectionFilterAll
		}
		children := make([]*permanentCollectionFilterNode, len(subjects.members))
		for i, member := range subjects.members {
			predicate, ok := inspectPermanentCollectionMembers(member)
			if !ok {
				return nil, false
			}
			children[i] = predicate
		}
		return &permanentCollectionFilterNode{operation: operation, children: children}, true
	default:
		return nil, false
	}
}

// Select reads current characteristics and presence without collection callbacks.
// Layers must be current. The result does not predict destruction or use LKI.
func (p PermanentCollectionPredicate) Select(g *Game) []PermanentVisit {
	if p.filter == nil || g == nil {
		return nil
	}
	var result []PermanentVisit
	for _, permanent := range g.zones.battlefield {
		if permanent.PhasedOut {
			continue
		}
		if p.filter.matches(permanent, g) {
			result = append(result, PermanentVisit{ID: permanent.ID(), Incarnation: permanent.incarnationID})
		}
	}
	return result
}
