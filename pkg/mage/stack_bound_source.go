package mage

import "slices"

// StackBoundPermanentSources reads event and payment bindings on stack objects.
// A saved incarnation prevents a returned card from replacing the referred object.
// This category alone cannot supply the complete CR 609.7a source list.
func (g *Game) StackBoundPermanentSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.stack.objects {
		identities := make([]capturedPermanent, 0, len(object.capturedPermanents))
		for binding, identity := range object.capturedPermanents {
			if binding != nil {
				identities = append(identities, identity)
			}
		}
		for _, effect := range object.Effects {
			if references, ok := effect.(stackPermanentBindingReferences); ok {
				identities = append(identities, references.stackPermanentBindings()...)
			}
		}
		// Map order must not change the options presented to the player.
		slices.SortFunc(identities, func(a, b capturedPermanent) int {
			if order := slices.Compare(a.id[:], b.id[:]); order != 0 {
				return order
			}
			return slices.Compare(a.incarnation[:], b.incarnation[:])
		})
		for _, identity := range identities {
			var source DamageSource
			p, lki := g.readPermanentIncarnationInformation(identity)
			if p != nil {
				source = damageSourceFromPermanent(p)
			} else if lki != nil {
				source = damageSourceFromLKI(lki)
			}
			if source.Valid() && !slices.ContainsFunc(sources, source.SameObject) {
				sources = append(sources, source)
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackBindings chooses a permanent saved in a stack binding.
// Other referred objects need separate contracts for a general source choice.
func ChooseDamageSourceFromStackBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackBindings: true}
}

type stackPermanentBindingReferences interface {
	stackPermanentBindings() []capturedPermanent
}

func (e *eventPermanentCapture) stackPermanentBindings() []capturedPermanent {
	if e.binding == nil {
		return nil
	}
	return []capturedPermanent{e.value}
}

func (e *sequenceEffect) stackPermanentBindings() []capturedPermanent {
	var identities []capturedPermanent
	for _, effect := range e.effects {
		if references, ok := effect.(stackPermanentBindingReferences); ok {
			identities = append(identities, references.stackPermanentBindings()...)
		}
	}
	return identities
}

func (e *capturedDelayedBody) stackPermanentBindings() []capturedPermanent {
	var identities []capturedPermanent
	for binding, identity := range e.permanents {
		if binding != nil {
			identities = append(identities, identity)
		}
	}
	for binding, values := range e.targets {
		if binding != nil {
			identities = append(identities, values...)
		}
	}
	for binding, values := range e.createdTokens {
		if binding != nil {
			identities = append(identities, values...)
		}
	}
	return identities
}
