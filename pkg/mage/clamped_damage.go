package mage

// ClampedDamageAmount limits one saved damage outcome for a life gain.
// Player life and loyalty use facts from before damage. Creature toughness
// uses the exact visit at this read, including departure and phase information.
func ClampedDamageAmount(results *DamageResultsBinding) ResolutionValue {
	return clampedDamageAmount{results}
}

type clampedDamageAmount struct{ results *DamageResultsBinding }

func (clampedDamageAmount) Text() string { return "clamped damage amount" }
func (v clampedDamageAmount) resolveValue(ctx *EffectContext) int {
	outcomes := ctx.Game.resolution.damageResultsBindings[v.results]
	if len(outcomes) != 1 {
		return 0
	}
	outcome := outcomes[0]
	amount := min(outcome.amount, outcome.gainLimit)
	if outcome.creatureRecipient {
		permanent, lki := ctx.Game.readPermanentIncarnationInformation(outcome.permanent)
		if permanent != nil {
			amount = min(amount, permanent.CurrentToughness(ctx.Game))
		} else if lki != nil {
			amount = min(amount, lki.Toughness)
		} else {
			return 0
		}
	}
	return max(0, amount)
}
func (v clampedDamageAmount) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"results", damageResultsRead{v.results}}}}
}
