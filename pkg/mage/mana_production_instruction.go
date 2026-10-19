package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// ManaProductionInstruction retains one mana production component.
// Saved amounts keep their exact operands and need separate query contracts.
type ManaProductionInstruction struct {
	Subject     PlayerRef
	Color       core.Color
	Amount      ResolutionValue
	Restriction core.ManaRestriction
}

// Effect builds the native action. Missing or invalid operands return nil.
func (instruction ManaProductionInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || nilDefinition(instruction.Amount) || instruction.Color < core.Colorless || instruction.Color > core.AnyColor || !nativeManaRestriction(instruction.Restriction) {
		return nil
	}
	if instruction.Color == core.AnyColor || instruction.Restriction != nil {
		if _, ok := instruction.Subject.(controllerRef); !ok {
			return nil
		}
		if instruction.Color == core.AnyColor {
			if instruction.Restriction != nil {
				return AddRestrictedManaOfAnyOneColor(instruction.Amount, instruction.Restriction)
			}
			return AddManaOfAnyOneColor(instruction.Amount)
		}
		amount, ok := instruction.Amount.(fixedValue)
		if !ok || amount.n <= 0 {
			return nil
		}
		return AddRestrictedMana([]ManaProduction{{Color: instruction.Color, Amount: amount.n}}, instruction.Restriction)
	}
	if _, ok := instruction.Subject.(controllerRef); ok {
		return AddManaAmount(instruction.Color, instruction.Amount)
	}
	return AddManaToPlayer(instruction.Subject, instruction.Color, instruction.Amount)
}

// nativeManaRestriction checks declarations without calling their methods.
func nativeManaRestriction(restriction core.ManaRestriction) bool {
	switch restriction.(type) {
	case nil, core.CreatureSpellsOnly, core.ArtifactSpellsOnly:
		return true
	default:
		return false
	}
}

// InspectManaProductionInstructions reads complete native production trees.
// Fixed restricted batches keep their event boundary in the original effect.
// It does not resolve operands, add mana or run replacements.
func InspectManaProductionInstructions(effect ResolvingEffect) ([]ManaProductionInstruction, bool) {
	var children []ResolvingEffect
	instruction := ManaProductionInstruction{Subject: You}
	switch effect := effect.(type) {
	case *sequenceEffect:
		if effect == nil {
			return nil, false
		}
		children = effect.effects
	case *compositeEffect:
		if effect == nil {
			return nil, false
		}
		children = effect.effects
	case *addManaEffect:
		if effect == nil || effect.color < core.Colorless || effect.color >= core.AnyColor {
			return nil, false
		}
		instruction.Color, instruction.Amount = effect.color, Fixed(effect.amount)
	case *manaAmountEffect:
		if effect == nil || effect.color < core.Colorless || effect.color >= core.AnyColor {
			return nil, false
		}
		instruction.Color, instruction.Amount = effect.color, effect.amount
	case *manaToPlayerEffect:
		if effect == nil || effect.color < core.Colorless || effect.color >= core.AnyColor {
			return nil, false
		}
		instruction.Subject, instruction.Color, instruction.Amount = effect.player, effect.color, effect.amount
	case *manaAnyOneColorEffect:
		if effect == nil || effect.restricted && nilDefinition(effect.restriction) || !effect.restricted && effect.restriction != nil {
			return nil, false
		}
		instruction.Color, instruction.Amount, instruction.Restriction = core.AnyColor, effect.amount, effect.restriction
	case *restrictedFixedManaEffect:
		if effect == nil || len(effect.productions) == 0 || nilDefinition(effect.restriction) || !nativeManaRestriction(effect.restriction) {
			return nil, false
		}
		out := make([]ManaProductionInstruction, 0, len(effect.productions))
		for _, production := range effect.productions {
			if production.AnyCombination || production.Color < core.Colorless || production.Color >= core.AnyColor || production.Amount <= 0 || production.Restriction != effect.restriction {
				return nil, false
			}
			out = append(out, ManaProductionInstruction{Subject: You, Color: production.Color, Amount: Fixed(production.Amount), Restriction: effect.restriction})
		}
		return out, true
	default:
		return nil, false
	}
	switch effect.(type) {
	case *sequenceEffect, *compositeEffect:
		var out []ManaProductionInstruction
		for _, child := range children {
			instructions, ok := InspectManaProductionInstructions(child)
			if !ok {
				return nil, false
			}
			out = append(out, instructions...)
		}
		return out, true
	default:
		if instruction.Effect() == nil {
			return nil, false
		}
		return []ManaProductionInstruction{instruction}, true
	}
}
