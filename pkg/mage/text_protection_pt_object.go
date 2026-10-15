package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddPrintedProtectionPTTextEffect checks both rules before installation.
// Each declaration must keep its own baseline and recipient context.
func (g *Game) AddPrintedProtectionPTTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("protection/P/T object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("protection/P/T object text requires the original face-up source")
	}
	if err := protectionPTSpellTextContext(current.Card); err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	if err := protectionPTSpellTextContext(revision.Definition()); err != nil {
		return err
	}
	old, fresh := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	if !sameProtectionPTSpellContext(old, fresh, revision) {
		return fmt.Errorf("protection/P/T object text changed its paired declaration context")
	}
	protection, pt := protectionPTSpellParts(old)
	for _, ability := range current.RuntimeAbilities {
		if nilDefinition(ability) {
			return fmt.Errorf("protection/P/T object text requires non-nil runtime declarations")
		}
		if ability.AbilityID() == protection.AbilityID() {
			active, ok := ability.(*ProtectionAbility)
			if !ok || !active.textLayerEligible {
				return fmt.Errorf("protection/P/T object text requires layered printed protection")
			}
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedProtection(probe.FindPermanent(current.ID()), protection.AbilityID(), false, changes...); err != nil {
		return err
	}
	contract := &printedStaticAbilityTextEffect{contract: staticPTTextContract}
	if err := probe.revisePrintedStaticAbilityWithRefresh(probe.FindPermanent(current.ID()), pt.AbilityID(), contract.eligible, false, changes...); err != nil {
		return err
	}
	colors := make([]TextChange, 0, len(changes))
	for _, change := range changes {
		if change.kind == 1 {
			colors = append(colors, change)
		}
	}
	if len(colors) == 0 {
		return nil
	}
	capture := capturedPermanent{id: current.ID(), incarnation: current.incarnationID}
	pending := []ContinuousEffect{
		&printedProtectionTextEffect{subject: capture, abilityID: protection.AbilityID(), duration: duration, changes: slices.Clone(colors)},
		&printedStaticAbilityTextEffect{copyTextOmitted: true, subject: capture, abilityID: pt.AbilityID(), duration: duration, changes: slices.Clone(colors), contract: staticPTTextContract},
	}
	if err := g.prepareAddedResolvingCopyText(current, pending[0]); err != nil {
		return err
	}
	for _, effect := range pending {
		effect.SetSourceID(current.ID())
		g.effects.Add(effect)
	}
	g.ApplyContinuousEffects()
	return nil
}
