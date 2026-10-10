package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CreatureEnchantTarget keeps a fixed creature restriction for Aura text checks.
func CreatureEnchantTarget() Target {
	target := TargetPermanent(IsCreature).(*PermanentTarget)
	target.fixedCreatureEnchant = true
	return target
}

// WithAttachedObjectText marks one complete creature Aura rule for text changes.
// The enchant declaration must stay fixed.
func WithAttachedObjectText() CardOption {
	return func(c *BaseCard) { c.attachedObjectText++ }
}

func attachedObjectTextContext(card Card) (bool, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.attachedObjectText != 1 || !slices.Equal(base.types, []core.CardType{core.TypeEnchantment}) || !slices.Equal(base.subTypes, []string{"Aura"}) || len(base.abilities) != 1 || len(base.attrSeeds) != 0 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 1 {
		return false, fmt.Errorf("attached object text requires one marked creature Aura rule without other casting rules")
	}
	target, ok := base.castTargets[0].(*PermanentTarget)
	if !ok || target == nil || !target.fixedCreatureEnchant || !sameSpellTextTargetSpec(target, CreatureEnchantTarget()) {
		return false, fmt.Errorf("attached object text requires one fixed creature enchant target")
	}
	if nilDefinition(base.abilities[0]) {
		return false, fmt.Errorf("attached object text requires a non-nil printed declaration")
	}
	if layeredActivatedEligible(base.abilities[0]) {
		return true, nil
	}
	holder, ok := base.abilities[0].(*StaticAbilityHolder)
	if !ok || holder == nil || !layeredStaticAbilityEligible(holder) {
		return false, fmt.Errorf("attached object text requires a marked host grant or ordinary action")
	}
	for _, effect := range holder.Effects {
		if _, ok := effect.(*attachedEffect); !ok {
			return false, fmt.Errorf("attached object text requires attached grant effects")
		}
	}
	return false, nil
}

// AddPrintedAttachedObjectTextEffect checks the whole Aura before installation.
// Its host and donor must keep separate rule bindings.
func (g *Game) AddPrintedAttachedObjectTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("attached object text requires a source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("attached object text requires the original face-up Aura")
	}
	action, err := attachedObjectTextContext(current.Card)
	if err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	nextAction, err := attachedObjectTextContext(revision.Definition())
	if err != nil {
		return err
	}
	old := current.Card.(*BaseCard).abilities[0]
	next, paired := revision.ReplacementAbility(old.AbilityID())
	if !paired || next != revision.Definition().(*BaseCard).abilities[0] || action != nextAction {
		return fmt.Errorf("attached object text changed its declaration context")
	}
	if action {
		if !sameActivatedObjectTextContext(old.(*ActionDefinition), next.(*ActionDefinition)) {
			return fmt.Errorf("attached object text changed its action context")
		}
	} else {
		first, second := old.(*StaticAbilityHolder), next.(*StaticAbilityHolder)
		if len(first.Effects) != len(second.Effects) {
			return fmt.Errorf("attached object text changed its grant count")
		}
	}
	for _, ability := range current.RuntimeAbilities {
		if nilDefinition(ability) {
			return fmt.Errorf("attached object text requires non-nil runtime declarations")
		}
	}
	if action {
		return g.AddPrintedActivatedTextEffect(current, old.AbilityID(), duration, changes...)
	}
	return g.AddPrintedStaticAbilityTextEffect(current, old.AbilityID(), duration, changes...)
}
