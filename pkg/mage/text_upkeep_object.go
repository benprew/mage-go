package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithUpkeepObjectTextRevision permits complete ordinary upkeep text changes.
// Spell entry and mixed rules need separate contracts.
func WithUpkeepObjectTextRevision() CardOption {
	return func(card *BaseCard) { card.upkeepObjectText++ }
}

func upkeepObjectTextContext(card Card) (*GenericTriggered, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.upkeepObjectText != 1 || !isEntryCopyPermanent(base) || base.HasType(core.TypeLand) ||
		base.HasType(core.TypeInstant) || base.HasType(core.TypeSorcery) || base.HasSubType("Aura") || len(base.attrSeeds) != 0 ||
		len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 {
		return nil, fmt.Errorf("upkeep object text requires a marked nonland permanent without casting costs, targets or keywords")
	}
	var trigger *GenericTriggered
	choice := false
	for _, ability := range base.abilities {
		switch rule := ability.(type) {
		case *GenericTriggered:
			if rule == nil || trigger != nil || !layeredUpkeepEligible(rule) {
				return nil, fmt.Errorf("upkeep object text requires one ordinary marked upkeep trigger")
			}
			trigger = rule
		case *opponentEntryChoiceAbility:
			if rule == nil || choice {
				return nil, fmt.Errorf("upkeep object text requires at most one opponent entry choice")
			}
			choice = true
		default:
			return nil, fmt.Errorf("upkeep object text contains an unsupported sibling rule")
		}
	}
	if trigger == nil {
		return nil, fmt.Errorf("upkeep object text requires one upkeep trigger")
	}
	context, _ := trigger.upkeepTextContext()
	if context.chosen != choice {
		return nil, fmt.Errorf("upkeep object text requires the matching opponent entry declaration")
	}
	return trigger, nil
}

func upkeepObjectPlayer(t *GenericTriggered) *PlayerBinding {
	switch p := t.eventDefinition.(type) {
	case turnStepEventPattern:
		return p.player
	case chosenPlayerUpkeepPattern:
		return p.player
	}
	return nil
}

func sameUpkeepObjectTextContext(old, fresh *GenericTriggered) bool {
	a, aOK := old.upkeepTextContext()
	b, bOK := fresh.upkeepTextContext()
	if !aOK || !bOK || a != b || upkeepObjectPlayer(old).name != upkeepObjectPlayer(fresh).name || len(old.targets) != len(fresh.targets) {
		return false
	}
	for i, target := range old.targets {
		if !sameSpellTextTarget(target, fresh.targets[i]) {
			return false
		}
	}
	return true
}

// AddPrintedUpkeepObjectTextEffect checks all printed rules before installation.
// Saved entry choices and pending triggers keep their prior facts.
func (g *Game) AddPrintedUpkeepObjectTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("upkeep object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("upkeep object text requires the original face-up source")
	}
	old, err := upkeepObjectTextContext(current.Card)
	if err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	fresh, err := upkeepObjectTextContext(revision.Definition())
	if err != nil {
		return err
	}
	oldBase, newBase := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	if !slices.Equal(oldBase.subTypes, newBase.subTypes) || len(oldBase.abilities) != len(newBase.abilities) || !sameUpkeepObjectTextContext(old, fresh) {
		return fmt.Errorf("upkeep object text changed its declaration context")
	}
	for i, ability := range oldBase.abilities {
		replacement, paired := revision.ReplacementAbility(ability.AbilityID())
		if !paired || replacement != newBase.abilities[i] {
			return fmt.Errorf("upkeep object text changed its declaration group")
		}
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(ability Ability) bool {
		rule, ok := ability.(*GenericTriggered)
		return ok && rule != nil && rule.AbilityID() == old.AbilityID() && layeredUpkeepEligible(rule) && sameUpkeepObjectTextContext(old, rule)
	}) {
		return fmt.Errorf("upkeep object text requires its active printed declaration")
	}
	return g.AddPrintedUpkeepTextEffect(current, old.AbilityID(), duration, changes...)
}
