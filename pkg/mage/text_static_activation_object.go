package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddPrintedStaticActivationTextEffect checks both rules before installation.
// Each declaration must keep its own baseline and pending actions.
func (g *Game) AddPrintedStaticActivationTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("static/activation object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("static/activation object text requires the original face-up source")
	}
	if err := staticActivationTextContext(current.Card); err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	if err := staticActivationTextContext(revision.Definition()); err != nil {
		return err
	}
	old, fresh := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	if !sameStaticActivationSpellContext(old, fresh, revision) {
		return fmt.Errorf("static/activation object text changed its paired declaration context")
	}
	for _, ability := range current.RuntimeAbilities {
		if nilDefinition(ability) {
			return fmt.Errorf("static/activation object text requires non-nil runtime declarations")
		}
	}
	pt, action := staticActivationSpellParts(old)
	probe := g.Clone()
	if err := probe.AddPrintedStaticPTTextEffect(probe.FindPermanent(current.ID()), pt.AbilityID(), duration, changes...); err != nil {
		return err
	}
	if err := probe.AddPrintedActivatedTextEffect(probe.FindPermanent(current.ID()), action.AbilityID(), duration, changes...); err != nil {
		return err
	}
	capture := capturedPermanent{id: current.ID(), incarnation: current.incarnationID}
	pending := []ContinuousEffect{
		&printedStaticAbilityTextEffect{subject: capture, abilityID: pt.AbilityID(), duration: duration, changes: slices.Clone(changes), contract: staticPTTextContract},
		&printedActivatedTextEffect{subject: capture, abilityID: action.AbilityID(), duration: duration, changes: slices.Clone(changes)},
	}
	for _, effect := range pending {
		effect.SetSourceID(current.ID())
		g.effects.Add(effect)
	}
	g.ApplyContinuousEffects()
	return nil
}
