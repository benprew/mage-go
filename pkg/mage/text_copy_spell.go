package mage

import (
	"fmt"
	"slices"
)

// WithCopyEntrySpellTextContinuation permits continued text after copy entry.
// The declaration must contain one optional entry copy without other rules.
func WithCopyEntrySpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.copyEntrySpellText++ }
}

func copyEntrySpellTextContext(card Card) (*entryCopyAbility, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.copyEntrySpellText != 1 || !isEntryCopyPermanent(base) ||
		len(base.abilities) != 1 || nilDefinition(base.abilities[0]) || len(base.attrSeeds) != 0 ||
		len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.landAuraSpellText != 0 || base.intrinsicSpellText != 0 {
		return nil, fmt.Errorf("copy-entry spell text requires one marked permanent with only an entry copy")
	}
	rule, ok := base.abilities[0].(*entryCopyAbility)
	if !ok || nilDefinition(rule.filter) {
		return nil, fmt.Errorf("copy-entry spell text requires a typed entry-copy declaration")
	}
	return rule, validateCopyModifications(rule.modifications)
}

func sameCopySpellModifications(old, fresh *entryCopyAbility) bool {
	if old.keepColors != fresh.keepColors || len(old.modifications) != len(fresh.modifications) {
		return false
	}
	for i, modification := range old.modifications {
		switch first := modification.(type) {
		case copyTypeAddition:
			second, ok := fresh.modifications[i].(copyTypeAddition)
			if !ok || !slices.Equal(first.types, second.types) {
				return false
			}
		case copyRetainedColors:
			second, ok := fresh.modifications[i].(copyRetainedColors)
			if !ok || !slices.Equal(first.colors, second.colors) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (g *Game) reviseCopyEntrySpellText(obj *StackObject, changes []TextChange) error {
	old, err := copyEntrySpellTextContext(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 || len(obj.capturedNumbers) != 0 ||
		len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("copy-entry spell text requires no targets or casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh, err := copyEntrySpellTextContext(revision.Definition())
	if err != nil {
		return err
	}
	replacement, ok := revision.ReplacementAbility(old.AbilityID())
	oldBase, newBase := obj.spellTextDefinition.(*BaseCard), revision.Definition().(*BaseCard)
	if !ok || replacement != fresh || !sameCopySpellModifications(old, fresh) || !slices.Equal(oldBase.subTypes, newBase.subTypes) {
		return fmt.Errorf("copy-entry spell text changed its declaration context")
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = revision.Definition()
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("copy-entry spell text lost its stack visit")
}

func buildCopyEntrySpellEntry(obj *StackObject) (Card, error) {
	if _, err := copyEntrySpellTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	fresh, err := BuildTextChangedDefinition(obj.spellTextDefinition)
	if err != nil {
		return nil, err
	}
	base := fresh.(*BaseCard)
	base.id, base.owner, base.isToken = obj.Card.ID(), obj.Card.Owner(), obj.Card.IsToken()
	base.departureCard = obj.Card
	characteristics := printedCopiableCharacteristics(base)
	base.textEntryValues = &characteristics
	return base, nil
}

func entryCopyTextChanges(card Card) []TextChange {
	if copied, ok := card.(permanentSpellCopyCard); ok {
		card = copied.Card
	}
	base, ok := card.(*BaseCard)
	if !ok || base.copyEntrySpellText != 1 || base.textDefinition == nil {
		return nil
	}
	return slices.Clone(base.textDefinition.changes)
}

// BuildEntryCopyWithText applies continued text to fresh donor rules.
// Ordinary text changes stay outside the saved copy values and printed card.
func BuildEntryCopyWithText(original Card, values *CopiableValues, changes ...TextChange) (Card, error) {
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return nil, fmt.Errorf("entry copy text requires validated words")
		}
	}
	copied, err := BuildEntryCopy(original, values)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 || values.faceDown {
		return copied, nil
	}
	revision, err := BuildTextDefinitionRevision(copied, changes...)
	if err != nil {
		return nil, fmt.Errorf("entry copy donor text: %w", err)
	}
	base, result := copied.(*BaseCard), revision.Definition().(*BaseCard)
	result.id, result.owner, result.isToken = base.id, base.owner, base.isToken
	result.printedDefinitionPlan, result.departureCard = base.printedDefinitionPlan, base.departureCard
	result.requiresAnte = base.requiresAnte
	characteristics := printedCopiableCharacteristics(result)
	result.textEntryValues = &characteristics
	return result, nil
}
