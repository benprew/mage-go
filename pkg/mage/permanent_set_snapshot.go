package mage

// SnapshotPermanentSet saves exact battlefield visits for later instructions.
// New members and returned objects cannot replace the saved visits.
func SnapshotPermanentSet(subjects PermanentSet, result *PermanentSetBinding) ResolvingEffect {
	return &snapshotPermanentSetEffect{subjects, result}
}

type snapshotPermanentSetEffect struct {
	subjects PermanentSet
	result   *PermanentSetBinding
}

func (*snapshotPermanentSetEffect) Text() string                 { return "save a permanent set" }
func (*snapshotPermanentSetEffect) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *snapshotPermanentSetEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set-snapshot", Text: e.Text(), bindingWrite: e.result, children: []definitionEdge{{"subjects", e.subjects}, {"result", e.result}}}
}
func (e *snapshotPermanentSetEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}
func (e *snapshotPermanentSetEffect) Apply(ctx *EffectContext) error {
	members := e.subjects.resolvePermanents(ctx)
	saved := make([]capturedPermanent, 0, len(members))
	seen := make(map[capturedPermanent]bool, len(members))
	for _, p := range members {
		if p == nil {
			continue
		}
		live := ctx.Game.FindPermanent(p.ID())
		if live == nil || live.PhasedOut || live.incarnationID != p.incarnationID {
			continue
		}
		visit := capturedPermanent{live.ID(), live.incarnationID}
		if !seen[visit] {
			seen[visit] = true
			saved = append(saved, visit)
		}
	}
	if ctx.Game.resolution.createdTokenBindings == nil {
		ctx.Game.resolution.createdTokenBindings = make(map[*PermanentSetBinding][]capturedPermanent)
	}
	ctx.Game.resolution.createdTokenBindings[e.result] = saved
	return nil
}
