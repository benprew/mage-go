package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithStaticActivationSpellTextContinuation permits paired P/T and activated rules.
// Both declarations keep changed words through permanent entry.
func WithStaticActivationSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.staticActivationText++ }
}

func staticActivationSpellParts(base *BaseCard) (*StaticAbilityHolder, *ActionDefinition) {
	if len(base.abilities) != 2 {
		return nil, nil
	}
	var pt *StaticAbilityHolder
	var action *ActionDefinition
	for _, ability := range base.abilities {
		switch rule := ability.(type) {
		case *StaticAbilityHolder:
			if rule == nil || pt != nil || len(rule.Effects) != 1 || !rule.ptTextLayer || !rule.ptTextRevision || !rule.supportsLayeredPTTextRevision() {
				return nil, nil
			}
			kind := rule.ptTextRevisionContext().kind
			if kind != globalPTContext && kind != controllerBasicLandSourcePT && kind != opponentBasicLandSourcePT {
				return nil, nil
			}
			pt = rule
		case *ActionDefinition:
			if rule == nil || action != nil || !layeredActivatedEligible(rule) || len(rule.modes) != 0 || rule.activationHistory != nil || rule.activationManaX != nil || rule.activationMinimumXOptions != 0 || len(rule.dslCaptures) != 0 || len(rule.activationConds) != 0 || len(rule.dslRestrictions) != 0 {
				return nil, nil
			}
			action = rule
		default:
			return nil, nil
		}
	}
	return pt, action
}

func staticActivationTextContext(card Card) error {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.staticActivationText != 1 || !slices.Equal(base.types, []core.CardType{core.TypeCreature}) || base.HasSubType("Aura") ||
		len(base.attrSeeds) != 0 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.protectionPTSpellText != 0 || base.intrinsicCastText != 0 || base.intrinsicSpellText != 0 || base.spellCastSpellText != 0 || base.staticEventSpellText != 0 || base.manaTapSpellText != 0 || base.globalStaticSpellText != 0 || base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 {
		return fmt.Errorf("static/activation spell text requires one marked creature without other rules or casting costs or targets")
	}
	pt, action := staticActivationSpellParts(base)
	if pt == nil || action == nil {
		return fmt.Errorf("static/activation spell text requires marked global or basic-land source P/T and one ordinary activated rule")
	}
	return nil
}

func sameStaticActivationSpellContext(old, fresh *BaseCard, revision *TextDefinitionRevision) bool {
	oldPT, oldAction := staticActivationSpellParts(old)
	newPT, newAction := staticActivationSpellParts(fresh)
	if oldPT == nil || newPT == nil || oldAction == nil || newAction == nil || !slices.Equal(old.subTypes, fresh.subTypes) || oldPT.ptTextRevisionContext() != newPT.ptTextRevisionContext() ||
		!sameActivatedTextContext(oldAction, newAction) || len(oldAction.effects) != len(newAction.effects) {
		return false
	}
	for i, target := range oldAction.targets {
		if !sameSpellTextTarget(target, newAction.targets[i]) {
			return false
		}
	}
	ptReplacement, ptPaired := revision.ReplacementAbility(oldPT.AbilityID())
	actionReplacement, actionPaired := revision.ReplacementAbility(oldAction.AbilityID())
	if !ptPaired || ptReplacement != newPT || !actionPaired || actionReplacement != newAction {
		return false
	}
	for i, ability := range old.abilities {
		replacement, paired := revision.ReplacementAbility(ability.AbilityID())
		if !paired || replacement != fresh.abilities[i] {
			return false
		}
	}
	return true
}

func buildStaticActivationSpellEntry(obj *StackObject) (Card, error) {
	if err := staticActivationTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	return buildIntrinsicTextSpellEntry(obj)
}
