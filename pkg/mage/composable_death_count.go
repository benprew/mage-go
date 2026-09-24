package mage

type creatureDeathCountValue struct{}

// CreaturesDiedThisTurn reads the current turn's creature death count.
// It includes tokens and creatures controlled by any player.
func CreaturesDiedThisTurn() ResolutionValue { return creatureDeathCountValue{} }

func (creatureDeathCountValue) Text() string { return "creatures that died this turn" }

func (creatureDeathCountValue) resolveValue(ctx *EffectContext) int {
	return ctx.Game.CreatureDeaths()
}

func (v creatureDeathCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "creature-death-count", Text: v.Text()}
}
