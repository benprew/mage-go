package mage

type conditionalSourcePTEffect struct {
	*composedLayerEffect
	landType string
}

// SourcePTWhileControllerControlsBasicLand modifies the source's P/T while
// its current controller controls a permanent with the supplied basic land type.
func SourcePTWhileControllerControlsBasicLand(landType string, power, toughness int) ContinuousEffect {
	rule := While(PlayerControls(You, HasSubType(landType)), ModifyPT(This, Fixed(power), Fixed(toughness)))
	return &conditionalSourcePTEffect{
		composedLayerEffect: rule.staticEffects()[0].(*composedLayerEffect),
		landType:            landType,
	}
}

func (e *conditionalSourcePTEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.composedLayerEffect = e.composedLayerEffect.cloneEffect().(*composedLayerEffect)
	return &clone
}

func (e *conditionalSourcePTEffect) inspectDefinition() definitionNode {
	n := e.composedLayerEffect.inspectDefinition()
	if _, valid := basicLandTypeColor(e.landType); !valid {
		n.problem = "conditional source P/T requires one basic land type"
	}
	return n
}

func (a *StaticAbilityHolder) conditionalSourcePTContext() bool {
	for _, effect := range a.Effects {
		if _, ok := effect.(*conditionalSourcePTEffect); ok {
			return true
		}
	}
	return false
}
