package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddPrintedSpellCastObjectTextEffect checks the complete rule before installation.
// Pending triggers keep their prior text, payments and saved event facts.
func (g *Game) AddPrintedSpellCastObjectTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("cast-event object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("cast-event object text requires the original face-up source")
	}
	old, err := spellCastObjectTextContext(current.Card)
	if err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	fresh, err := spellCastObjectTextContext(revision.Definition())
	if err != nil {
		return err
	}
	oldBase, newBase := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	replacement, paired := revision.ReplacementAbility(old.AbilityID())
	if !paired || replacement != fresh || !slices.Equal(oldBase.subTypes, newBase.subTypes) || !sameSpellCastObjectTextContext(old, fresh) {
		return fmt.Errorf("cast-event object text changed its declaration context")
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(ability Ability) bool {
		rule, ok := ability.(*GenericTriggered)
		return ok && rule != nil && rule.AbilityID() == old.AbilityID() && layeredSpellCastEligible(rule) && sameSpellCastObjectTextContext(old, rule)
	}) {
		return fmt.Errorf("cast-event object text requires its active printed declaration")
	}
	return g.AddPrintedSpellCastTextEffect(current, old.AbilityID(), duration, changes...)
}

func sameSpellCastObjectTextContext(old, fresh *GenericTriggered) bool {
	if !sameSpellCastTextContext(old.eventDefinition.(spellCastPattern), fresh.eventDefinition.(spellCastPattern)) ||
		old.Optional != fresh.Optional || len(old.modes) != len(fresh.modes) ||
		(old.interveningDefinition == nil) != (fresh.interveningDefinition == nil) ||
		(old.stateDefinition == nil) != (fresh.stateDefinition == nil) ||
		len(old.targets) != len(fresh.targets) || len(old.effects) != len(fresh.effects) {
		return false
	}
	for i, target := range old.targets {
		if !sameSpellTextTarget(target, fresh.targets[i]) {
			return false
		}
	}
	return true
}

func spellCastObjectTextContext(card Card) (*GenericTriggered, error) {
	rule, err := spellCastSpellTextContext(card)
	if err != nil {
		return nil, err
	}
	if rule.Optional || len(rule.modes) != 0 || rule.interveningDefinition != nil || rule.stateDefinition != nil {
		return nil, fmt.Errorf("cast-event object text requires one ordinary nonmodal trigger")
	}
	return rule, nil
}
