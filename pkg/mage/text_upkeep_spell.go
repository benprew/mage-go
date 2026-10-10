package mage

import (
	"fmt"
	"maps"
	"slices"
)

// WithUpkeepSpellTextContinuation permits one upkeep rule at entry.
// Fixed intrinsic keywords keep their contributions through entry.
// Entry choices and casting results require separate contracts.
func WithUpkeepSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.upkeepSpellText++ }
}

func upkeepSpellTextContext(card Card) (*GenericTriggered, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.upkeepSpellText != 1 || len(base.abilities) != 1 ||
		base.spellCastSpellText != 0 || base.manaTapSpellText != 0 || base.staticEventSpellText != 0 ||
		base.globalStaticSpellText != 0 || base.intrinsicSpellText != 0 || base.intrinsicCastText != 0 ||
		base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 || base.staticActivationText != 0 || base.protectionPTSpellText != 0 {
		return nil, fmt.Errorf("upkeep spell text requires one marked ordinary permanent rule")
	}
	rule, err := upkeepObjectTextContext(card)
	if err != nil {
		return nil, err
	}
	context, _ := rule.upkeepTextContext()
	if context.chosen {
		return nil, fmt.Errorf("upkeep spell text does not permit entry choices")
	}
	return rule, nil
}

func (g *Game) reviseUpkeepSpellText(obj *StackObject, changes []TextChange) error {
	old, err := upkeepSpellTextContext(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 || len(obj.capturedNumbers) != 0 ||
		len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("upkeep spell text requires no announced targets, bodies or casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh, err := upkeepSpellTextContext(revision.Definition())
	if err != nil {
		return err
	}
	oldBase, newBase := obj.spellTextDefinition.(*BaseCard), revision.Definition().(*BaseCard)
	replacement, paired := revision.ReplacementAbility(old.AbilityID())
	if !paired || replacement != fresh || !maps.Equal(oldBase.attrSeeds, newBase.attrSeeds) || !slices.Equal(oldBase.subTypes, newBase.subTypes) ||
		!sameUpkeepObjectTextContext(old, fresh) {
		return fmt.Errorf("upkeep spell text changed its declaration context")
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = revision.Definition()
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("upkeep spell text lost its stack visit")
}

func buildUpkeepSpellEntry(obj *StackObject) (Card, error) {
	if _, err := upkeepSpellTextContext(obj.spellTextDefinition); err != nil {
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
