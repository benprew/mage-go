package mage

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// LandEnchantTarget keeps a fixed land restriction for Aura text checks.
func LandEnchantTarget() Target {
	target := TargetPermanent(IsLand).(*PermanentTarget)
	target.fixedLandEnchant = true
	return target
}

func landAuraObjectTextContext(card Card) (*StaticAbilityHolder, error) {
	if _, err := landAuraSpellTextContext(card); err != nil {
		return nil, err
	}
	base := card.(*BaseCard)
	target := UnwrapTarget(base.castTargets[0]).(*PermanentTarget)
	if !target.fixedLandEnchant || !sameSpellTextTargetSpec(target, LandEnchantTarget()) {
		return nil, fmt.Errorf("land Aura object text requires a fixed land enchant target")
	}
	for _, ability := range base.abilities {
		if holder, ok := ability.(*StaticAbilityHolder); ok {
			if !holder.landTypeTextLayer || !holder.landTypeTextRevision {
				return nil, fmt.Errorf("land Aura object text requires a layered land declaration")
			}
			return holder, nil
		}
	}
	return nil, fmt.Errorf("land Aura object text requires an attached land declaration")
}

// AddPrintedLandAuraObjectTextEffect checks the whole Aura before installation.
// Saved entry choices must stay separate from printed words.
func (g *Game) AddPrintedLandAuraObjectTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("land Aura object text requires a source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("land Aura object text requires the original face-up source")
	}
	old, err := landAuraObjectTextContext(current.Card)
	if err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	fresh, err := landAuraObjectTextContext(revision.Definition())
	if err != nil {
		return err
	}
	if old.landTypeTextRevisionContext() != fresh.landTypeTextRevisionContext() {
		return fmt.Errorf("land Aura object text changed its land declaration context")
	}
	base, rebuilt := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	for i, ability := range base.abilities {
		replacement, paired := revision.ReplacementAbility(ability.AbilityID())
		if !paired || replacement != rebuilt.abilities[i] {
			return fmt.Errorf("land Aura object text changed its declaration correspondence")
		}
	}
	for _, ability := range current.RuntimeAbilities {
		if nilDefinition(ability) {
			return fmt.Errorf("land Aura object text requires non-nil runtime declarations")
		}
	}
	return g.AddPrintedStaticLandTypeTextEffect(current, old.AbilityID(), duration, changes...)
}
