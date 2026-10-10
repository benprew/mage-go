package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithProtectionPTSpellTextContinuation permits paired protection and P/T rules.
// Both printed declarations keep changed words through permanent entry.
func WithProtectionPTSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.protectionPTSpellText++ }
}

func protectionPTSpellParts(base *BaseCard) (*ProtectionAbility, *StaticAbilityHolder) {
	if len(base.abilities) != 2 {
		return nil, nil
	}
	var protection *ProtectionAbility
	var pt *StaticAbilityHolder
	for _, ability := range base.abilities {
		switch rule := ability.(type) {
		case *ProtectionAbility:
			if rule == nil || protection != nil || !rule.colorTextRevision || !rule.textLayerEligible || !rule.supportsColorTextRevision() {
				return nil, nil
			}
			protection = rule
		case *StaticAbilityHolder:
			if rule == nil || pt != nil || len(rule.Effects) != 1 || !rule.ptTextLayer || !rule.ptTextRevision || !rule.supportsPTTextRevision() {
				return nil, nil
			}
			kind := rule.ptTextRevisionContext().kind
			if kind != opponentNontokenColorSourcePT && kind != opponentNontokenColorNamedPT {
				return nil, nil
			}
			pt = rule
		default:
			return nil, nil
		}
	}
	return protection, pt
}

func protectionPTSpellTextContext(card Card) error {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.protectionPTSpellText != 1 || !slices.Equal(base.types, []core.CardType{core.TypeCreature}) || base.HasSubType("Aura") ||
		len(base.attrSeeds) != 0 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.intrinsicCastText != 0 || base.intrinsicSpellText != 0 || base.spellCastSpellText != 0 || base.staticEventSpellText != 0 || base.manaTapSpellText != 0 || base.globalStaticSpellText != 0 || base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 {
		return fmt.Errorf("protection/P/T spell text requires one marked creature without other rules or casting costs or targets")
	}
	protection, pt := protectionPTSpellParts(base)
	if protection == nil || pt == nil {
		return fmt.Errorf("protection/P/T spell text requires marked color protection and one opponent nontoken color P/T rule")
	}
	return nil
}

func sameProtectionPTSpellContext(old, fresh *BaseCard, revision *TextDefinitionRevision) bool {
	oldProtection, oldPT := protectionPTSpellParts(old)
	newProtection, newPT := protectionPTSpellParts(fresh)
	if oldProtection == nil || newProtection == nil || oldPT == nil || newPT == nil || !slices.Equal(old.subTypes, fresh.subTypes) || len(oldProtection.FromColors) != len(newProtection.FromColors) || oldPT.ptTextRevisionContext() != newPT.ptTextRevisionContext() {
		return false
	}
	protectionReplacement, protectionPaired := revision.ReplacementAbility(oldProtection.AbilityID())
	ptReplacement, ptPaired := revision.ReplacementAbility(oldPT.AbilityID())
	if !protectionPaired || protectionReplacement != newProtection || !ptPaired || ptReplacement != newPT {
		return false
	}
	for i, rule := range old.abilities {
		replacement, paired := revision.ReplacementAbility(rule.AbilityID())
		if !paired || replacement != fresh.abilities[i] {
			return false
		}
	}
	return true
}

func buildProtectionPTSpellEntry(obj *StackObject) (Card, error) {
	if err := protectionPTSpellTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	return buildIntrinsicTextSpellEntry(obj)
}
