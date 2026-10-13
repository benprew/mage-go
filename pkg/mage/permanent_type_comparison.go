package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PermanentsSharingCardTypes selects present subjects with a shared card type.
// The reference uses current or exact last-known information. With no types,
// all card types can match. This information cannot restore an absent subject.
func PermanentsSharingCardTypes(reference PermanentRef, subjects PermanentSet, types ...core.CardType) PermanentSet {
	return sharingCardTypesSet{reference, subjects, slices.Clone(types)}
}

type sharingCardTypesSet struct {
	reference PermanentRef
	subjects  PermanentSet
	types     []core.CardType
}

func (sharingCardTypesSet) permanentSet() {}

func (s sharingCardTypesSet) inspectDefinition() definitionNode {
	n := permanentInformationNode(s.reference, "permanent-set", "permanents with a shared card type")
	n.children = append(n.children, definitionEdge{"subjects", s.subjects})
	seen := make(map[core.CardType]bool)
	for _, kind := range s.types {
		if !slices.Contains(core.CardTypeValues(), kind) || seen[kind] {
			n.problem = "card type comparison requires distinct valid card types"
		}
		seen[kind] = true
	}
	return n
}

func (s sharingCardTypesSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	p, lki := readPermanentInformation(s.reference, ctx)
	if p == nil && lki == nil {
		return nil
	}
	var result []*Permanent
	for _, candidate := range s.subjects.resolvePermanents(ctx) {
		if candidate == nil || candidate.PhasedOut {
			continue
		}
		current := ctx.Game.FindPermanent(candidate.ID())
		if current == nil || current.incarnationID != candidate.incarnationID {
			continue
		}
		if sharesPermanentCardType(current, p, lki, s.types) && !slices.Contains(result, current) {
			result = append(result, current)
		}
	}
	return result
}

func sharesPermanentCardType(subject, reference *Permanent, lki *PermanentLKI, types []core.CardType) bool {
	if len(types) == 0 {
		types = core.CardTypeValues()
	}
	return slices.ContainsFunc(types, func(kind core.CardType) bool {
		return subject.HasType(kind) && ((reference != nil && reference.HasType(kind)) || (lki != nil && lki.HasType(kind)))
	})
}
