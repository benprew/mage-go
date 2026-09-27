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
	outerNumbers := ctx.Game.resolution.numberBindings
	defer func() { ctx.Game.resolution.numberBindings = outerNumbers }()
	outerPlayers := ctx.Game.resolution.playerBindings
	defer func() { ctx.Game.resolution.playerBindings = outerPlayers }()
	outer := ctx.Game.resolution.permanentBindings
	defer func() { ctx.Game.resolution.permanentBindings = outer }()
	outerCreated := ctx.Game.resolution.createdTokenBindings
	defer func() { ctx.Game.resolution.createdTokenBindings = outerCreated }()
	for _, member := range members {
		ctx.Game.resolution.createdTokenBindings = cloneCreatedTokenBindings(outerCreated)
		ctx.Game.resolution.numberBindings = maps.Clone(outerNumbers)
		ctx.Game.resolution.playerBindings = maps.Clone(outerPlayers)
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
