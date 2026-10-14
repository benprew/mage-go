package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PermanentsOfChosenColor selects present subjects with the reference's chosen color.
// The exact reference visit supplies current or last-known information.
func PermanentsOfChosenColor(reference PermanentRef, subjects PermanentSet) PermanentSet {
	return chosenColorPermanentSet{reference, subjects}
}

type chosenColorPermanentSet struct {
	reference PermanentRef
	subjects  PermanentSet
}

func (chosenColorPermanentSet) permanentSet() {}

func (s chosenColorPermanentSet) inspectDefinition() definitionNode {
	n := permanentInformationNode(s.reference, "permanent-set", "permanents of the chosen color")
	n.children = append(n.children, definitionEdge{"subjects", s.subjects})
	return n
}

func (s chosenColorPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	p, lki := readPermanentInformation(s.reference, ctx)
	if lki != nil {
		p = lki.Snapshot
	}
	if p == nil || p.ChosenColor < core.White || p.ChosenColor > core.Green {
		return nil
	}
	color := p.ChosenColor
	var result []*Permanent
	for _, candidate := range s.subjects.resolvePermanents(ctx) {
		if candidate == nil || candidate.PhasedOut {
			continue
		}
		current := ctx.Game.FindPermanent(candidate.ID())
		if current != nil && current.incarnationID == candidate.incarnationID && slices.Contains(current.Colors(), color) && !slices.Contains(result, current) {
			result = append(result, current)
		}
	}
	return result
}
