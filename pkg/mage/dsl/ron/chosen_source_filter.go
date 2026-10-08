package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileChosenDamageSourceFilter(n Node) (mage.DamageSourceFilter, error) {
	switch n.Name {
	case "AnyDamageSource":
		if _, err := symbol(n); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		return mage.DamageSourceFilter{}, nil
	case "IsColor":
		if err := arity(n, 1); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		value, err := color(n.Args[0])
		if err != nil {
			return mage.DamageSourceFilter{}, err
		}
		if value == core.Colorless {
			return mage.DamageSourceFilter{}, problem(n.Args[0], "source color requires one of the five Magic colors")
		}
		return mage.DamageSourceHasColor(value), nil
	case "IsCardtype":
		if err := arity(n, 1); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		value, err := cardType(n.Args[0])
		if err != nil {
			return mage.DamageSourceFilter{}, err
		}
		return mage.DamageSourceHasType(value), nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.DamageSourceFilter{}, err
		}
		if len(parts) == 0 {
			return mage.DamageSourceFilter{}, problem(n, "supply at least one source filter")
		}
		filters := make([]mage.DamageSourceFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileChosenDamageSourceFilter(part)
			if err != nil {
				return mage.DamageSourceFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.AllDamageSources(filters...), nil
		}
		return mage.AnyOfDamageSources(filters...), nil
	default:
		return mage.DamageSourceFilter{}, unsupported(n, "chosen damage source filter")
	}
}

func compileChosenDamageSourceChoice(n Node) (*chosenDamageSourcePlan, error) {
	if n.Name != "ChooseADamageSource" {
		return nil, unsupported(n, "damage source choice")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter, err := compileChosenDamageSourceFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	return &chosenDamageSourcePlan{filter: filter}, nil
}

func (p *chosenDamageSourcePlan) waitingPreventionSpellChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingPreventionSpellReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingRegenerationChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingRegenerationReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingDrawStepLifeChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingDrawStepLifeReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingPreventionChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingPreventionReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

// The engine keeps resolved legacy references until consumption or cleanup.
// Static registrations require their live subjects.
func (p *chosenDamageSourcePlan) waitingRedirectionChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingRedirectionReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingCounterPlacementChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingCounterPlacementReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) commandChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromCommandObjects(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) stackExileChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromStackExileBindings(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingCommandChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingCommandReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingModifierChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingModifierReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingEmptyLibraryChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingEmptyLibraryReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingPersistentReplacementChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingPersistentReplacementReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingUntapSkipChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingUntapSkipReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingDiscardChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingDiscardReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingExileChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingExileReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) stackGraveyardChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromStackGraveyardReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) stackHandChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromStackHandReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) inspectableStackHandChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromInspectableStackHandReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingPrecastExileChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingPrecastExileReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}

func (p *chosenDamageSourcePlan) waitingPrecastHandChoice(bindings instanceBindings) mage.ResolvingEffect {
	return mage.ChooseDamageSourceFromWaitingPrecastHandReferences(mage.You, p.filter, bindings.chosenDamageSources[p])
}
