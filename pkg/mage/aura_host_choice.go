package mage

import "slices"

// LegalHostsForAuraCard selects legal hosts without an entry or a target.
// The controller supplies the Aura's enchant restrictions before entry.
func LegalHostsForAuraCard(card CardRef, controller PlayerRef, hosts PermanentSet) PermanentSet {
	return auraCardHostSet{card, controller, hosts}
}

type auraCardHostSet struct {
	card       CardRef
	controller PlayerRef
	hosts      PermanentSet
}

func (auraCardHostSet) permanentSet() {}
func (s auraCardHostSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "legal hosts for Aura card", children: []definitionEdge{{"card", s.card}, {"controller", s.controller}, {"hosts", s.hosts}}}
}
func (s auraCardHostSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	card, ok := s.card.resolveCard(ctx)
	if !ok || card.IsToken() || !card.HasSubType("Aura") {
		return nil
	}
	controller, ok := s.controller.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(controller) == nil {
		return nil
	}
	aura := NewPermanent(card, controller)
	var result []*Permanent
	for _, host := range s.hosts.resolvePermanents(ctx) {
		if host == nil {
			continue
		}
		current := ctx.Game.FindPermanent(host.ID())
		if current != nil && !current.PhasedOut && current.incarnationID == host.incarnationID && ctx.Game.canAttachEntering(aura, current) && !slices.Contains(result, current) {
			result = append(result, current)
		}
	}
	return result
}

// ChoosePermanentForPlayerOrElse selects one branch after a required choice.
// An empty choice clears the result and selects the failure branch.
func ChoosePermanentForPlayerOrElse(player PlayerRef, subjects PermanentSet, result *PermanentChoiceBinding, chosen, notChosen ResolvingEffect) ResolvingEffect {
	return Sequence(ChoosePermanentForPlayer(player, subjects, result), &permanentChoiceContinuation{PermanentChosenIn(result), chosen, notChosen})
}

type permanentChoiceContinuation struct {
	subject           PermanentRef
	chosen, notChosen ResolvingEffect
}

func (*permanentChoiceContinuation) Text() string                 { return "select permanent choice branch" }
func (*permanentChoiceContinuation) Properties() EffectProperties { return EffectProperties{} }
func (e *permanentChoiceContinuation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"choice", e.subject}}}
	if !nilDefinition(e.chosen) {
		n.children = append(n.children, definitionEdge{"chosen", e.chosen})
	}
	if !nilDefinition(e.notChosen) {
		n.children = append(n.children, definitionEdge{"not-chosen", e.notChosen})
	}
	if nilDefinition(e.chosen) && nilDefinition(e.notChosen) {
		n.problem = "supply at least one choice branch"
	}
	return n
}
func (e *permanentChoiceContinuation) Apply(ctx *EffectContext) error {
	body := e.notChosen
	if p, ok := e.subject.resolvePermanent(ctx); ok && !p.PhasedOut {
		body = e.chosen
	}
	if !nilDefinition(body) {
		return body.Apply(ctx)
	}
	return nil
}
func (e *permanentChoiceContinuation) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || effectReadsSourcePermanent(e.chosen) || effectReadsSourcePermanent(e.notChosen)
}
