package mage

import "maps"

type permanentLoop struct {
	subjects PermanentSet
	member   *PermanentBinding
	body     ResolvingEffect
}

// ForEach saves the member list once and runs the body in battlefield order.
// Each iteration has its own saved values. The actions are not simultaneous.
func ForEach(subjects PermanentSet, member *PermanentBinding, body ResolvingEffect) ResolvingEffect {
	return &permanentLoop{subjects, member, body}
}
func (*permanentLoop) Text() string { return "for each permanent" }
func (e *permanentLoop) Properties() EffectProperties {
	var props EffectProperties
	if !nilDefinition(e.body) {
		props = e.body.Properties()
	}
	props.Mass = true
	return props
}
func (e *permanentLoop) Apply(ctx *EffectContext) error {
	subjects := e.subjects.resolvePermanents(ctx)
	members := make([]capturedPermanent, 0, len(subjects))
	for _, permanent := range subjects {
		members = append(members, capturedPermanent{permanent.ID(), permanent.incarnationID})
	}
	outer := ctx.Game.resolution.permanentBindings
	defer func() { ctx.Game.resolution.permanentBindings = outer }()
	for _, member := range members {
		frame := maps.Clone(outer)
		if frame == nil {
			frame = make(map[*PermanentBinding]capturedPermanent)
		}
		frame[e.member] = member
		ctx.Game.resolution.permanentBindings = frame
		if err := CompositeEffects("", e.body).Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}
func (e *permanentLoop) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-loop", isolatedBindings: true, children: []definitionEdge{
		{"subjects", e.subjects}, {"member", loopMember{e.member}}, {"body", e.body},
	}}
}

type loopMember struct{ binding *PermanentBinding }

func (m loopMember) inspectDefinition() definitionNode {
	return definitionNode{Role: "loop-member", bindingWrite: m.binding, children: []definitionEdge{{"binding", m.binding}}}
}
