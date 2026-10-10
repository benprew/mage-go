package mage

import (
	"fmt"
	"slices"
)

// ChangePrintedSpellText reads saved words for one exact pending spell.
// The checked revision service keeps payment and announcement facts fixed.
func ChangePrintedSpellText(subject StackObjectRef, changes ...TextChangeRef) ResolvingEffect {
	return &spellTextChange{subject: subject, changes: slices.Clone(changes)}
}

type spellTextChange struct {
	subject StackObjectRef
	changes []TextChangeRef
}

func (*spellTextChange) Text() string                 { return "change printed spell text" }
func (*spellTextChange) Properties() EffectProperties { return EffectProperties{} }
func (e *spellTextChange) Apply(ctx *EffectContext) error {
	obj, ok := e.subject.resolveStackObject(ctx)
	if !ok || obj == nil {
		return nil
	}
	changes := make([]TextChange, len(e.changes))
	for i, ref := range e.changes {
		if nilDefinition(ref) {
			return fmt.Errorf("spell text requires saved distinct words")
		}
		change, valid := ref.Resolve(ctx)
		if !valid {
			return fmt.Errorf("spell text requires saved distinct words")
		}
		changes[i] = change
	}
	return ctx.Game.RevisePrintedSpellText(obj.ID, changes...)
}

func (e *spellTextChange) inspectDefinition() definitionNode {
	n := definitionNode{Role: "spell-text-change", children: []definitionEdge{{"subject", e.subject}}}
	if len(e.changes) == 0 {
		n.problem = "supply word changes"
	}
	for i, change := range e.changes {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("change[%d]", i), change})
	}
	return n
}
