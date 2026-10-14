package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLandTypeAuraSpellTextContinuation permits land-type rules at spell entry.
// The declaration must have one attached rule and its required entry choice.
func WithLandTypeAuraSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.landAuraSpellText++ }
}

func landAuraSpellTextContext(card Card) (landTypeRevisionContext, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.landAuraSpellText != 1 || !slices.Equal(base.types, []core.CardType{core.TypeEnchantment}) ||
		!slices.Equal(base.subTypes, []string{"Aura"}) || len(base.attrSeeds) != 0 || len(base.modes) != 0 ||
		len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 1 {
		return 0, fmt.Errorf("land Aura spell text requires one marked Aura without additional casting costs or other rules")
	}
	target, ok := UnwrapTarget(base.castTargets[0]).(*PermanentTarget)
	if !ok || target == nil || target.Min() != 1 || target.Max() != 1 {
		return 0, fmt.Errorf("land Aura spell text requires one permanent enchant target")
	}
	var context landTypeRevisionContext
	choices := 0
	for _, ability := range base.abilities {
		if nilDefinition(ability) {
			return 0, fmt.Errorf("land Aura spell text requires non-nil declarations")
		}
		switch declaration := ability.(type) {
		case *basicLandEntryChoiceAbility:
			choices++
		case *StaticAbilityHolder:
			if context != 0 {
				return 0, fmt.Errorf("land Aura spell text requires one attached land rule")
			}
			context = declaration.landTypeTextRevisionContext()
			if context != attachedLandTypeRevision && context != chosenAttachedLandTypeRevision {
				return 0, fmt.Errorf("land Aura spell text requires a fixed or chosen attached land rule")
			}
		default:
			return 0, fmt.Errorf("land Aura spell text has an unsupported declaration")
		}
	}
	if context == 0 || context == attachedLandTypeRevision && choices != 0 || context == chosenAttachedLandTypeRevision && choices != 1 {
		return 0, fmt.Errorf("land Aura spell text requires the matching entry choice")
	}
	return context, nil
}

func (g *Game) reviseLandAuraSpellText(obj *StackObject, changes []TextChange) error {
	oldContext, err := landAuraSpellTextContext(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 1 || len(obj.TargetSpecs) != 1 || len(obj.capturedNumbers) != 0 ||
		len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("land Aura spell text requires one announced target without casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh := revision.Definition()
	newContext, err := landAuraSpellTextContext(fresh)
	if err != nil {
		return err
	}
	oldTarget, newTarget := obj.spellTextDefinition.CastTargets()[0], fresh.CastTargets()[0]
	if newContext != oldContext || obj.TargetSpecs[0] != oldTarget || !sameSpellTextTargetSpec(oldTarget, newTarget) {
		return fmt.Errorf("land Aura spell text changed its declaration context")
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = fresh
	rebuilt.TargetSpecs = []Target{newTarget}
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("land Aura spell text lost its stack visit")
}

func buildLandAuraSpellEntry(obj *StackObject) (Card, error) {
	if _, err := landAuraSpellTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	fresh, err := BuildTextChangedDefinition(obj.spellTextDefinition)
	if err != nil {
		return nil, err
	}
	base := fresh.(*BaseCard)
	base.id, base.owner, base.isToken = obj.Card.ID(), obj.Card.Owner(), obj.Card.IsToken()
	base.departureCard = obj.Card
	return base, nil
}
