package mage

import "github.com/google/uuid"

// ExileEachWithControllerCounts saves actual exile counts by former controller.
// Tokens count even though they cannot remain in exile.
func ExileEachWithControllerCounts(subjects PermanentSet, result *PlayerNumbersBinding) ResolvingEffect {
	return &exileControllerCountsEffect{subjects, result}
}

type exileControllerCountsEffect struct {
	subjects PermanentSet
	result   *PlayerNumbersBinding
}

func (*exileControllerCountsEffect) Text() string {
	return "exile permanents and save former controller counts"
}
func (*exileControllerCountsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true, TargetPurposeOverride: AITargetExile}
}
func (e *exileControllerCountsEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}
func (e *exileControllerCountsEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"result", e.result}}}
	if e.result != nil {
		for _, b := range e.result.members {
			n.bindingWrites = append(n.bindingWrites, b)
		}
	}
	return n
}
func (e *exileControllerCountsEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	if g.resolution.numberBindings == nil {
		g.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	for _, b := range e.result.members {
		g.resolution.numberBindings[b] = 0
	}
	for _, moved := range g.exilePermanents(e.subjects.resolvePermanents(ctx)) {
		if b := e.result.member(g, moved.controller); b != nil {
			g.resolution.numberBindings[b]++
		}
	}
	return nil
}

type exiledPermanentFact struct {
	visit      capturedPermanent
	controller uuid.UUID
}

// DrawEachPlayerNumber draws each player's saved count in turn order.
// All counts are read before the first draw replacement can run.
func DrawEachPlayerNumber(result *PlayerNumbersBinding) ResolvingEffect {
	return &drawPlayerNumbersEffect{result}
}

type drawPlayerNumbersEffect struct{ result *PlayerNumbersBinding }

func (*drawPlayerNumbersEffect) Text() string { return "each player draws their saved count" }
func (*drawPlayerNumbersEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *drawPlayerNumbersEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"result", e.result}}}
	if e.result != nil {
		for _, b := range e.result.members {
			n.children = append(n.children, definitionEdge{"count", FromNumber(b)})
		}
	}
	return n
}
func (e *drawPlayerNumbersEffect) Apply(ctx *EffectContext) error {
	ids := EachPlayer.resolvePlayers(ctx)
	counts := make([]int, len(ids))
	for i, id := range ids {
		counts[i] = ctx.Game.resolution.numberBindings[e.result.member(ctx.Game, id)]
	}
	for i, id := range ids {
		child := *ctx
		child.Targets = []uuid.UUID{id}
		if err := DrawCards(Fixed(counts[i])).Apply(&child); err != nil {
			return err
		}
	}
	return nil
}
