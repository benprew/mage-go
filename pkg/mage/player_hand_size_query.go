package mage

// PlayerHandSizePredicate retains a native hand-size value without game state.
// The zero value is unknown.
type PlayerHandSizePredicate struct {
	player PlayerCollectionPredicate
}

// InspectPlayerHandSize accepts HandSizeOf with a native singular player.
// References can be You, ActivePlayer or a native player target.
// Inspection does not call reference or value callbacks.
func InspectPlayerHandSize(value ResolutionValue) (PlayerHandSizePredicate, bool) {
	if value, ok := value.(playerHandSizeValue); ok {
		if player, ok := InspectPlayerCollection(OnePlayer(value.subject)); ok {
			return PlayerHandSizePredicate{player: player}, true
		}
	}
	return PlayerHandSizePredicate{}, false
}

func (predicate PlayerHandSizePredicate) bindingsKnown(ctx *EffectContext) bool {
	return predicate.player.bindingsKnown(ctx)
}

// Read returns the current hand size without reference callbacks.
// Empty or absent players supply zero. Missing or malformed frames are unknown.
func (predicate PlayerHandSizePredicate) Read(ctx *EffectContext) (int, bool) {
	players, known := predicate.player.Select(ctx)
	if !known {
		return 0, false
	}
	total := 0
	for _, id := range players {
		total += len(ctx.Game.GetPlayer(id).Hand())
	}
	return total, true
}
