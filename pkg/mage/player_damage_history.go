package mage

type playerDamageHistoryValue struct {
	subject  PlayerRef
	artifact bool
}

// DamageTakenThisTurn reads the selected player's actual damage this turn.
func DamageTakenThisTurn(subject PlayerRef) ResolutionValue {
	return playerDamageHistoryValue{subject: subject}
}

// ArtifactDamageTakenThisTurn reads actual damage from artifact sources this turn.
// The source's type at the time of damage determines the saved total.
func ArtifactDamageTakenThisTurn(subject PlayerRef) ResolutionValue {
	return playerDamageHistoryValue{subject: subject, artifact: true}
}

func (v playerDamageHistoryValue) Text() string {
	if v.artifact {
		return "damage taken from artifacts this turn"
	}
	return "damage taken this turn"
}

func (v playerDamageHistoryValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.subject.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return 0
	}
	if v.artifact {
		return ctx.Game.GetArtifactDamageTaken(id)
	}
	return ctx.Game.DamageTakenByPlayer(id)
}

func (v playerDamageHistoryValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}
