package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// LandManaBonusThisTurn adds one mana when a matching land is tapped for mana.
// The shared bonus query keeps payment planning and production in agreement.
func LandManaBonusThisTurn(filter PermanentFilter, color core.Color) ResolvingEffect {
	return &landManaBonusInstallation{filter: filter, color: color}
}

type landManaBonusInstallation struct {
	filter PermanentFilter
	color  core.Color
}

func (*landManaBonusInstallation) Text() string                 { return "add a land mana bonus this turn" }
func (*landManaBonusInstallation) Properties() EffectProperties { return EffectProperties{} }
func (e *landManaBonusInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "land-mana-bonus", Text: e.Text()}
	if e.color < core.Colorless || e.color >= core.AnyColor {
		n.problem = "land mana bonus requires one concrete mana type"
	}
	return n
}
func (e *landManaBonusInstallation) Apply(ctx *EffectContext) error {
	ctx.Game.effects.Add(&temporaryLandManaBonus{effectSource: effectSource{sourceID: ctx.SourceID}, filter: e.filter, color: e.color})
	return nil
}

type temporaryLandManaBonus struct {
	effectSource
	filter PermanentFilter
	color  core.Color
}

func (*temporaryLandManaBonus) survivesSourceExit() bool             { return true }
func (*temporaryLandManaBonus) GetLayer() core.Layer                 { return core.LayerAbility }
func (*temporaryLandManaBonus) GetDuration() core.Duration           { return core.EndOfTurn }
func (*temporaryLandManaBonus) IsActive(*Game) bool                  { return true }
func (*temporaryLandManaBonus) Text() string                         { return "land mana bonus this turn" }
func (*temporaryLandManaBonus) Properties() EffectProperties         { return EffectProperties{} }
func (*temporaryLandManaBonus) ApplyContinuous(*EffectContext) error { return nil }
func (e *temporaryLandManaBonus) cloneEffect() ContinuousEffect      { clone := *e; return &clone }
