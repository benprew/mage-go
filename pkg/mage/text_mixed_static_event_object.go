package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddPrintedMixedStaticEventTextEffect checks both printed rules before installation.
// Each rule keeps its declaration and its own baseline.
func (g *Game) AddPrintedMixedStaticEventTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("mixed object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("mixed object text requires the original face-up source")
	}
	index, kind, err := mixedStaticEventSpellTextContext(current.Card)
	if err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	freshIndex, freshKind, err := mixedStaticEventSpellTextContext(revision.Definition())
	if err != nil {
		return err
	}
	old, fresh := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	if index != freshIndex || kind != freshKind || !slices.Equal(old.subTypes, fresh.subTypes) || !sameMixedObjectEventContext(old.abilities[1-index], fresh.abilities[1-index]) {
		return fmt.Errorf("mixed object text changed its declaration context")
	}
	for i, rule := range old.abilities {
		replacement, paired := revision.ReplacementAbility(rule.AbilityID())
		if !paired || replacement != fresh.abilities[i] {
			return fmt.Errorf("mixed object text lost its paired declarations")
		}
	}
	event := old.abilities[1-index]
	if !slices.ContainsFunc(current.RuntimeAbilities, func(rule Ability) bool {
		return rule != nil && sameMixedObjectEventContext(event, rule) && rule.AbilityID() == event.AbilityID()
	}) {
		return fmt.Errorf("mixed object text requires its active printed event declaration")
	}
	probe := g.Clone()
	subject := probe.FindPermanent(current.ID())
	staticID := old.abilities[index].AbilityID()
	if err := probe.addPrintedStaticTextEffect(subject, staticID, duration, kind, changes...); err != nil {
		return err
	}
	capture := capturedPermanent{id: current.ID(), incarnation: current.incarnationID}
	pending := []ContinuousEffect{&printedStaticAbilityTextEffect{subject: capture, abilityID: staticID, duration: duration, changes: slices.Clone(changes), contract: kind}}
	if _, mana := event.(*ManaBonusAbility); mana {
		if err := probe.AddPrintedManaTapTextEffect(probe.FindPermanent(current.ID()), event.AbilityID(), duration, changes...); err != nil {
			return err
		}
		pending = append(pending, &printedManaTapTextEffect{subject: capture, abilityID: event.AbilityID(), duration: duration, changes: slices.Clone(changes)})
	}
	for _, effect := range pending {
		effect.SetSourceID(current.ID())
		g.effects.Add(effect)
	}
	g.ApplyContinuousEffects()
	return nil
}

func sameMixedObjectEventContext(old, fresh Ability) bool {
	switch rule := fresh.(type) {
	case *ManaBonusAbility:
		if rule == nil {
			return false
		}
	case *GenericTriggered:
		if rule == nil {
			return false
		}
	}
	if !sameMixedEventSpellContext(old, fresh) {
		return false
	}
	if first, mana := old.(*ManaBonusAbility); mana {
		return first.BonusMana == fresh.(*ManaBonusAbility).BonusMana
	}
	return true
}
