package mage

// SkipNextUntaps prevents step untaps for the subject's current controller.
// Overlapping restrictions consume the same steps. Other untaps remain legal.
func SkipNextUntaps(subject PermanentRef, count int) ResolvingEffect {
	return &skipNextUntapsEffect{subject: subject, count: count}
}

type skipNextUntapsEffect struct {
	subject PermanentRef
	count   int
}

func (*skipNextUntapsEffect) Text() string { return "skip the next controller untap steps" }
func (*skipNextUntapsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *skipNextUntapsEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.count < 0 {
		n.problem = "supply a nonnegative untap step count"
	}
	return n
}
func (e *skipNextUntapsEffect) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.turns.skipNextUntaps(p.ID(), e.count)
	}
	return nil
}

// SkipNextUntapEach captures the set before it installs the restrictions.
// Later membership changes cannot add a subject to the operation.
func SkipNextUntapEach(subjects PermanentSet) ResolvingEffect {
	return &skipNextUntapEachEffect{subjects: subjects}
}

type skipNextUntapEachEffect struct{ subjects PermanentSet }

func (*skipNextUntapEachEffect) Text() string {
	return "skip each subject's next controller untap step"
}
func (*skipNextUntapEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *skipNextUntapEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *skipNextUntapEachEffect) Apply(ctx *EffectContext) error {
	for _, p := range e.subjects.resolvePermanents(ctx) {
		if p == nil {
			continue
		}
		live := ctx.Game.FindPermanent(p.ID())
		if live != nil && live.incarnationID == p.incarnationID {
			ctx.Game.SkipNextUntap(live.ID())
		}
	}
	return nil
}
