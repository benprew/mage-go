package mage

// PlayerDamageHistoryPredicate retains a native history value and its doublings.
// It stores no game state. The zero value is unknown.
type PlayerDamageHistoryPredicate struct {
	player   PlayerCollectionPredicate
	artifact bool
	inner    *PlayerDamageHistoryPredicate
}

// InspectPlayerDamageHistory accepts native player damage totals and doubling.
// Players can be You, ActivePlayer or a singular native player target.
// Unknown operands reject the complete tree without callbacks or history reads.
func InspectPlayerDamageHistory(value ResolutionValue) (PlayerDamageHistoryPredicate, bool) {
	switch value := value.(type) {
	case playerDamageHistoryValue:
		player, ok := InspectPlayerCollection(OnePlayer(value.subject))
		if ok {
			return PlayerDamageHistoryPredicate{player: player, artifact: value.artifact}, true
		}
	case arithmeticValue:
		if value.operation != valueMultiply {
			break
		}
		child := value.right
		factor, ok := value.left.(fixedValue)
		if !ok || factor.n != 2 {
			factor, ok = value.right.(fixedValue)
			child = value.left
		}
		if !ok || factor.n != 2 {
			break
		}
		inner, ok := InspectPlayerDamageHistory(child)
		if ok {
			return PlayerDamageHistoryPredicate{inner: &inner}, true
		}
	}
	return PlayerDamageHistoryPredicate{}, false
}

func (predicate PlayerDamageHistoryPredicate) bindingsKnown(ctx *EffectContext) bool {
	if predicate.inner != nil {
		return predicate.inner.bindingsKnown(ctx)
	}
	_, known := predicate.player.Select(ctx)
	return known
}

// Read uses actual damage totals from this turn without reference callbacks.
// Bound players require a complete legal resolving target frame. Empty or absent
// players supply zero; missing or malformed frames are unknown. Players use no LKI.
func (predicate PlayerDamageHistoryPredicate) Read(ctx *EffectContext) (int, bool) {
	if predicate.inner != nil {
		amount, known := predicate.inner.Read(ctx)
		return amount * 2, known
	}
	players, known := predicate.player.Select(ctx)
	if !known {
		return 0, false
	}
	total := 0
	for _, id := range players {
		if predicate.artifact {
			total += ctx.Game.GetArtifactDamageTaken(id)
		} else {
			total += ctx.Game.DamageTakenByPlayer(id)
		}
	}
	return total, true
}
