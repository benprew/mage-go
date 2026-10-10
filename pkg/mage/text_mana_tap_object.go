package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddPrintedManaTapObjectTextEffect checks the complete rule before installation.
// Pending events keep their prior text and saved facts.
func (g *Game) AddPrintedManaTapObjectTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("mana-event object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("mana-event object text requires the original face-up source")
	}
	old, err := manaTapSpellTextContext(current.Card)
	if err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	fresh, err := manaTapSpellTextContext(revision.Definition())
	if err != nil {
		return err
	}
	oldBase, newBase := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	replacement, paired := revision.ReplacementAbility(old.AbilityID())
	if !paired || replacement != fresh || !slices.Equal(oldBase.subTypes, newBase.subTypes) || !sameManaTapObjectTextContext(old, fresh) {
		return fmt.Errorf("mana-event object text changed its declaration context")
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(rule Ability) bool {
		return manaTapSpellRulePresent(rule) && rule.AbilityID() == old.AbilityID() && layeredManaTapEligible(rule) && sameManaTapObjectTextContext(old, rule)
	}) {
		return fmt.Errorf("mana-event object text requires its active printed declaration")
	}
	return g.AddPrintedManaTapTextEffect(current, old.AbilityID(), duration, changes...)
}

func sameManaTapObjectTextContext(old, fresh Ability) bool {
	if !sameManaTapSpellTextContext(old, fresh) {
		return false
	}
	if bonus, ok := old.(*ManaBonusAbility); ok {
		return bonus.BonusMana == fresh.(*ManaBonusAbility).BonusMana
	}
	return true
}
