package mage

type lowestPowerSet struct{ subjects PermanentSet }

// LowestPower keeps all members tied for the least current power.
func LowestPower(subjects PermanentSet) PermanentSet { return lowestPowerSet{subjects} }
func (lowestPowerSet) permanentSet()                 {}
func (s lowestPowerSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	members := s.subjects.resolvePermanents(ctx)
	if len(members) == 0 {
		return nil
	}
	least := members[0].CurrentPower(ctx.Game)
	for _, p := range members[1:] {
		least = min(least, p.CurrentPower(ctx.Game))
	}
	result := make([]*Permanent, 0, len(members))
	for _, p := range members {
		if p.CurrentPower(ctx.Game) == least {
			result = append(result, p)
		}
	}
	return result
}
func (s lowestPowerSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "least-power-set", children: []definitionEdge{{"subjects", s.subjects}}}
}

type permanentChoice struct {
	player   PlayerRef
	subjects PermanentSet
	binding  *PermanentBinding
}

// ChoosePermanent makes a required non-target choice at resolution.
// No candidates or an absent player records absence in the binding.
func ChoosePermanent(player PlayerRef, subjects PermanentSet, binding *PermanentBinding) ResolvingEffect {
	return &permanentChoice{player, subjects, binding}
}
func (e *permanentChoice) Text() string                 { return "choose a permanent" }
func (e *permanentChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *permanentChoice) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	delete(ctx.Game.resolution.permanentBindings, e.binding)
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	candidates := e.subjects.resolvePermanents(ctx)
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[0]
	if len(candidates) > 1 {
		chosen := ctx.Game.GetPlayer(id).ChoosePermanent(candidates, "Choose "+e.binding.name, ctx.Game)
		if chosen != nil {
			for _, p := range candidates {
				if p.ID() == chosen.ID() && p.incarnationID == chosen.incarnationID {
					selected = p
					break
				}
			}
		}
	}
	ctx.Game.resolution.permanentBindings[e.binding] = capturedPermanent{selected.ID(), selected.incarnationID}
	return nil
}
func (e *permanentChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-choice", bindingWrite: e.binding, children: []definitionEdge{{"player", e.player}, {"subjects", e.subjects}, {"binding", e.binding}}}
}

// ChooseRandomPermanent saves one random member at resolution.
// An empty set clears the binding and does not use a random value.
func ChooseRandomPermanent(subjects PermanentSet, binding *PermanentBinding) ResolvingEffect {
	return &randomPermanentChoice{subjects, binding}
}

type randomPermanentChoice struct {
	subjects PermanentSet
	binding  *PermanentBinding
}

func (*randomPermanentChoice) Text() string                 { return "choose a random permanent" }
func (*randomPermanentChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *randomPermanentChoice) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	delete(ctx.Game.resolution.permanentBindings, e.binding)
	candidates := e.subjects.resolvePermanents(ctx)
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[ctx.Game.RandIntn(len(candidates))]
	ctx.Game.resolution.permanentBindings[e.binding] = capturedPermanent{selected.ID(), selected.incarnationID}
	return nil
}
func (e *randomPermanentChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "random-permanent-choice", bindingWrite: e.binding, children: []definitionEdge{{"subjects", e.subjects}, {"binding", e.binding}}}
}
