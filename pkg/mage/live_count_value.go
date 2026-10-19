package mage

// LivePermanentCountPredicate retains a native count and its rounding operations.
// It has no saved members or game state. The zero value is unknown.
type LivePermanentCountPredicate struct {
	subjects ContextualPermanentCollectionPredicate
	inner    *LivePermanentCountPredicate
	roundUp  bool
}

// InspectLivePermanentCount accepts a complete native permanent count tree.
// Each division must use a fixed divisor of two. Unknown children reject the tree.
// Inspection does not execute value, collection, player or filter callbacks.
func InspectLivePermanentCount(value ResolutionValue) (LivePermanentCountPredicate, bool) {
	return inspectLivePermanentCount(value, nil)
}

func inspectLivePermanentCount(value ResolutionValue, member *PlayerBinding) (LivePermanentCountPredicate, bool) {
	switch value := value.(type) {
	case permanentCountValue:
		root, ok := inspectContextualCollectionWithMember(value.subjects, member)
		subjects := ContextualPermanentCollectionPredicate{root: root}
		if ok {
			return LivePermanentCountPredicate{subjects: subjects}, true
		}
	case arithmeticValue:
		if value.operation != valueDivideUp && value.operation != valueDivideDown {
			break
		}
		divisor, ok := value.right.(fixedValue)
		if !ok || divisor.n != 2 {
			break
		}
		inner, ok := inspectLivePermanentCount(value.left, member)
		if ok {
			return LivePermanentCountPredicate{inner: &inner, roundUp: value.operation == valueDivideUp}, true
		}
	}
	return LivePermanentCountPredicate{}, false
}

// Read counts current visits and applies rounding without callbacks or LKI.
// Layers must be current. Bound players need a complete resolving target frame.
func (predicate LivePermanentCountPredicate) Read(ctx *EffectContext) (int, bool) {
	if predicate.inner != nil {
		value, known := predicate.inner.Read(ctx)
		if !known {
			return 0, false
		}
		half := value / 2
		if predicate.roundUp {
			half += value % 2
		}
		return half, true
	}
	visits, known := predicate.subjects.selectKnown(ctx)
	return len(visits), known
}
