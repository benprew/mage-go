package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddPrintedGlobalStaticTextEffect checks the complete rule before installation.
// Other rule contexts need separate contracts.
func (g *Game) AddPrintedGlobalStaticTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("global static object text requires a source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("global static object text requires the original face-up source")
	}
	if _, err := globalStaticSpellTextContext(current.Card); err != nil {
		return err
	}
	base := current.Card.(*BaseCard)
	kind, err := globalStaticSpellRuleContext(base.abilities[0])
	if err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	if _, err := globalStaticSpellTextContext(revision.Definition()); err != nil {
		return err
	}
	fresh := revision.Definition().(*BaseCard)
	freshKind, err := globalStaticSpellRuleContext(fresh.abilities[0])
	if err != nil {
		return err
	}
	replacement, paired := revision.ReplacementAbility(base.abilities[0].AbilityID())
	if kind != freshKind || !paired || replacement != fresh.abilities[0] || !slices.Equal(base.subTypes, fresh.subTypes) {
		return fmt.Errorf("global static object text changed its declaration context")
	}
	switch kind {
	case staticPTTextContract:
		return g.AddPrintedStaticPTTextEffect(current, base.abilities[0].AbilityID(), duration, changes...)
	case staticLandTypeTextContract:
		return g.AddPrintedStaticLandTypeTextEffect(current, base.abilities[0].AbilityID(), duration, changes...)
	default:
		return fmt.Errorf("global static object text requires one global P/T or land-type rule")
	}
}

// ChangePrintedPermanentText uses only complete supported rule contexts.
// Saved word choices are read when the body resolves.
func ChangePrintedPermanentText(subject PermanentRef, duration core.Duration, changes ...TextChangeRef) ResolvingEffect {
	return &permanentObjectTextChange{intrinsicObjectTextChange{subject: subject, duration: duration, changes: slices.Clone(changes)}}
}

type permanentObjectTextChange struct{ intrinsicObjectTextChange }

func (*permanentObjectTextChange) Text() string { return "change printed permanent text" }
func (e *permanentObjectTextChange) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok || p == nil {
		return nil
	}
	changes := make([]TextChange, len(e.changes))
	for i, ref := range e.changes {
		change, valid := ref.Resolve(ctx)
		if !valid {
			return fmt.Errorf("permanent object text requires saved distinct words")
		}
		changes[i] = change
	}
	if _, err := attachedObjectTextContext(p.Card); err == nil {
		return ctx.Game.AddPrintedAttachedObjectTextEffect(p, e.duration, changes...)
	}
	if intrinsicSpellTextContext(p.Card) == nil {
		return ctx.Game.AddPrintedIntrinsicCreatureTextEffect(p, e.duration, changes...)
	}
	if activatedObjectTextContext(p.Card) == nil {
		return ctx.Game.AddPrintedActivatedObjectTextEffect(p, e.duration, changes...)
	}
	if globalGrantObjectTextContext(p.Card) == nil {
		return ctx.Game.AddPrintedGlobalGrantTextEffect(p, e.duration, changes...)
	}
	if protectionPTSpellTextContext(p.Card) == nil {
		return ctx.Game.AddPrintedProtectionPTTextEffect(p, e.duration, changes...)
	}
	if staticActivationTextContext(p.Card) == nil {
		return ctx.Game.AddPrintedStaticActivationTextEffect(p, e.duration, changes...)
	}
	if _, _, err := mixedStaticEventSpellTextContext(p.Card); err == nil {
		return ctx.Game.AddPrintedMixedStaticEventTextEffect(p, e.duration, changes...)
	}
	return ctx.Game.AddPrintedGlobalStaticTextEffect(p, e.duration, changes...)
}
func (e *permanentObjectTextChange) inspectDefinition() definitionNode {
	n := e.intrinsicObjectTextChange.inspectDefinition()
	n.Role = "permanent-object-text"
	return n
}
