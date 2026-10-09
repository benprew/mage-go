package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type ptRuleContext uint8

const (
	globalPTContext ptRuleContext = iota
	controllerBasicLandSourcePT
	opponentBasicLandSourcePT
	opponentNontokenColorSourcePT
	opponentNontokenColorNamedPT
)

type opponentPTEffect struct {
	*composedLayerEffect
	context  ptRuleContext
	landType string
	color    core.Color
	name     string
}

// SourcePTWhileOpponentControlsBasicLand modifies the source's P/T while
// its current controller's opponent controls the supplied basic land type.
func SourcePTWhileOpponentControlsBasicLand(landType string, power, toughness int) ContinuousEffect {
	rule := While(OpponentControls(HasSubType(landType)), ModifyPT(This, Fixed(power), Fixed(toughness)))
	return &opponentPTEffect{composedLayerEffect: rule.staticEffects()[0].(*composedLayerEffect), context: opponentBasicLandSourcePT, landType: landType}
}

// SourcePTWhileOpponentControlsNontokenColor modifies the source's P/T while
// its current controller's opponent controls a nontoken permanent of one color.
func SourcePTWhileOpponentControlsNontokenColor(color core.Color, power, toughness int) ContinuousEffect {
	rule := While(OpponentControls(And(Not(IsToken), HasColorFilter(color))), ModifyPT(This, Fixed(power), Fixed(toughness)))
	return &opponentPTEffect{composedLayerEffect: rule.staticEffects()[0].(*composedLayerEffect), context: opponentNontokenColorSourcePT, color: color}
}

// NamedCreaturesPTWhileOpponentControlsNontokenColor modifies named creatures
// while the source's opponent controls a nontoken permanent of one color.
func NamedCreaturesPTWhileOpponentControlsNontokenColor(name string, color core.Color, power, toughness int) ContinuousEffect {
	rule := While(OpponentControls(And(Not(IsToken), HasColorFilter(color))), ModifyEachPT(MatchingPermanents(And(IsCreature, Named(name))), Fixed(power), Fixed(toughness)))
	return &opponentPTEffect{composedLayerEffect: rule.staticEffects()[0].(*composedLayerEffect), context: opponentNontokenColorNamedPT, color: color, name: name}
}

func (e *opponentPTEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.composedLayerEffect = e.composedLayerEffect.cloneEffect().(*composedLayerEffect)
	return &clone
}

func (e *opponentPTEffect) validContext() bool {
	if e == nil || e.composedLayerEffect == nil {
		return false
	}
	switch e.context {
	case opponentBasicLandSourcePT:
		_, valid := basicLandTypeColor(e.landType)
		return valid
	case opponentNontokenColorSourcePT:
		return e.color >= core.White && e.color <= core.Green
	case opponentNontokenColorNamedPT:
		return e.color >= core.White && e.color <= core.Green && e.name != ""
	default:
		return false
	}
}

func (e *opponentPTEffect) inspectDefinition() definitionNode {
	n := e.composedLayerEffect.inspectDefinition()
	if !e.validContext() {
		n.problem = "opponent P/T requires a basic land type or one color and a nonempty named recipient"
	}
	return n
}

type ptRevisionContext struct {
	kind ptRuleContext
	name string
}

func (a *StaticAbilityHolder) ptTextRevisionContext() ptRevisionContext {
	if a.conditionalSourcePTContext() {
		return ptRevisionContext{kind: controllerBasicLandSourcePT}
	}
	for _, effect := range a.Effects {
		if typed, ok := effect.(*opponentPTEffect); ok && typed != nil {
			return ptRevisionContext{kind: typed.context, name: typed.name}
		}
	}
	return ptRevisionContext{}
}
