package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddPrintedIntrinsicCreatureTextEffect changes all matching printed intrinsic rules.
// Each declaration must pass its layer contract before any effect is installed.
func (g *Game) AddPrintedIntrinsicCreatureTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("intrinsic object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("intrinsic object text requires the original face-up source")
	}
	if err := intrinsicSpellTextContext(current.Card); err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	if err := intrinsicSpellTextContext(revision.Definition()); err != nil {
		return err
	}
	if !sameIntrinsicCreatureSpellContext(current.Card.(*BaseCard), revision.Definition().(*BaseCard), revision) {
		return fmt.Errorf("intrinsic object text changed its declaration context")
	}
	colors, lands := []TextChange{}, []TextChange{}
	for _, change := range changes {
		switch change.kind {
		case 1:
			colors = append(colors, change)
		case 2:
			lands = append(lands, change)
		default:
			return fmt.Errorf("intrinsic object text requires validated words")
		}
	}
	probe := g.Clone()
	pending := []ContinuousEffect{}
	subject := capturedPermanent{id: current.ID(), incarnation: current.incarnationID}
	for _, ability := range current.Card.Abilities() {
		id := ability.AbilityID()
		switch ability.(type) {
		case *ProtectionAbility:
			if len(colors) == 0 {
				continue
			}
			if err := probe.AddPrintedProtectionTextEffect(probe.FindPermanent(current.ID()), id, duration, colors...); err != nil {
				return err
			}
			pending = append(pending, &printedProtectionTextEffect{subject: subject, abilityID: id, duration: duration, changes: slices.Clone(colors)})
		case *PrintedBasicLandwalkAbility:
			if len(lands) == 0 {
				continue
			}
			if err := probe.AddPrintedBasicLandwalkTextEffect(probe.FindPermanent(current.ID()), id, duration, lands...); err != nil {
				return err
			}
			pending = append(pending, &printedLandwalkTextEffect{subject: subject, abilityID: id, duration: duration, changes: slices.Clone(lands)})
		}
	}
	for _, effect := range pending {
		effect.SetSourceID(current.ID())
		g.effects.Add(effect)
	}
	if len(pending) > 0 {
		g.ApplyContinuousEffects()
	}
	return nil
}

// ChangePrintedIntrinsicCreatureText reads saved words when its body resolves.
// Unsupported complete rule contexts return an error before installation.
func ChangePrintedIntrinsicCreatureText(subject PermanentRef, duration core.Duration, changes ...TextChangeRef) ResolvingEffect {
	return &intrinsicObjectTextChange{subject: subject, duration: duration, changes: slices.Clone(changes)}
}

type intrinsicObjectTextChange struct {
	subject  PermanentRef
	duration core.Duration
	changes  []TextChangeRef
}

func (*intrinsicObjectTextChange) Text() string                 { return "change printed intrinsic creature text" }
func (*intrinsicObjectTextChange) Properties() EffectProperties { return EffectProperties{} }
func (e *intrinsicObjectTextChange) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok || p == nil {
		return nil
	}
	changes := make([]TextChange, len(e.changes))
	for i, ref := range e.changes {
		change, ok := ref.Resolve(ctx)
		if !ok {
			return fmt.Errorf("intrinsic object text requires saved distinct words")
		}
		changes[i] = change
	}
	return ctx.Game.AddPrintedIntrinsicCreatureTextEffect(p, e.duration, changes...)
}
func (e *intrinsicObjectTextChange) inspectDefinition() definitionNode {
	n := definitionNode{Role: "intrinsic-object-text", children: []definitionEdge{{"subject", e.subject}}}
	if len(e.changes) == 0 || (e.duration != core.Indefinite && e.duration != core.EndOfTurn) {
		n.problem = "supply word changes and an indefinite or end-of-turn duration"
	}
	for i, change := range e.changes {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("change[%d]", i), change})
	}
	return n
}
func (e *intrinsicObjectTextChange) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}
