package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithIntrinsicCastEventSpellTextContinuation permits paired creature rules at entry.
// Other rule kinds and casting results require separate contracts.
func WithIntrinsicCastEventSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.intrinsicCastText++ }
}

func intrinsicCastSpellParts(base *BaseCard) (*BaseCard, *GenericTriggered) {
	intrinsic := *base
	intrinsic.abilities = make([]Ability, 0, len(base.abilities))
	var event *GenericTriggered
	for _, ability := range base.abilities {
		if rule, ok := ability.(*GenericTriggered); ok {
			if event != nil || rule == nil || !layeredSpellCastEligible(rule) || rule.Optional || len(rule.modes) != 0 || rule.interveningDefinition != nil || rule.stateDefinition != nil {
				return nil, nil
			}
			event = rule
		} else {
			intrinsic.abilities = append(intrinsic.abilities, ability)
		}
	}
	return &intrinsic, event
}

func intrinsicCastTextContext(card Card) error {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.intrinsicCastText != 1 || !slices.Equal(base.types, []core.CardType{core.TypeCreature}) || base.HasSubType("Aura") ||
		len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.intrinsicSpellText != 0 || base.spellCastSpellText != 0 || base.staticEventSpellText != 0 || base.manaTapSpellText != 0 || base.globalStaticSpellText != 0 || base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 {
		return fmt.Errorf("intrinsic/cast spell text requires one marked creature without casting costs or targets")
	}
	intrinsic, event := intrinsicCastSpellParts(base)
	if intrinsic == nil || event == nil {
		return fmt.Errorf("intrinsic/cast spell text requires one marked typed battlefield cast trigger")
	}
	return intrinsicCreatureTextRulesContext(intrinsic)
}

func sameIntrinsicCastSpellContext(old, fresh *BaseCard, revision *TextDefinitionRevision) bool {
	oldIntrinsic, oldEvent := intrinsicCastSpellParts(old)
	newIntrinsic, newEvent := intrinsicCastSpellParts(fresh)
	if oldIntrinsic == nil || newIntrinsic == nil || oldEvent == nil || newEvent == nil || !sameIntrinsicCreatureSpellContext(oldIntrinsic, newIntrinsic, revision) ||
		!sameSpellCastTextContext(oldEvent.eventDefinition.(spellCastPattern), newEvent.eventDefinition.(spellCastPattern)) || len(oldEvent.targets) != len(newEvent.targets) || len(oldEvent.effects) != len(newEvent.effects) {
		return false
	}
	for i, target := range oldEvent.targets {
		if !sameSpellTextTarget(target, newEvent.targets[i]) {
			return false
		}
	}
	for i, rule := range old.abilities {
		replacement, paired := revision.ReplacementAbility(rule.AbilityID())
		if !paired || replacement != fresh.abilities[i] {
			return false
		}
	}
	return true
}

func buildIntrinsicCastSpellEntry(obj *StackObject) (Card, error) {
	if err := intrinsicCastTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	return buildIntrinsicTextSpellEntry(obj)
}
