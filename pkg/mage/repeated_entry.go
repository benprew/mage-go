package mage

import (
	"slices"
)

// RepeatWhilePositive keeps round results separate from its caller.
// It tests the value after each complete round. Nonpositive values end the process.
func RepeatWhilePositive(body ResolvingEffect, again ResolutionValue) ResolvingEffect {
	return &repeatedResolution{body: body, again: again}
}

type repeatedResolution struct {
	body  ResolvingEffect
	again ResolutionValue
}

func (*repeatedResolution) Text() string { return "repeat the process while its result is positive" }
func (e *repeatedResolution) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *repeatedResolution) inspectDefinition() definitionNode {
	return definitionNode{Role: "repeated-resolution", isolatedBindings: true, children: []definitionEdge{{"round", e.body}, {"repeat value", e.again}}}
}
func (e *repeatedResolution) Apply(ctx *EffectContext) error {
	outer := ctx.Game.resolution
	defer func() { ctx.Game.resolution.installRoundBindings(outer) }()
	for {
		ctx.Game.resolution.installRoundBindings(outer.Clone())
		child := *ctx
		child.targetBindings = cloneDelayedTargets(ctx.targetBindings)
		if err := e.body.Apply(&child); err != nil {
			return err
		}
		if e.again.resolveValue(&child) <= 0 {
			return nil
		}
	}
}
func (e *repeatedResolution) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.body) || resolutionValueReadsSourcePermanent(e.again)
}

// EachPlayerMayPutFromHandOntoBattlefield saves the number of actual entries.
// Each choice uses the current hand after the preceding player's entry.
func EachPlayerMayPutFromHandOntoBattlefield(subjects PlayerSet, starting PlayerRef, filter CardFilter, result *NumberBinding) ResolvingEffect {
	return &optionalHandEntryRound{subjects: subjects, starting: starting, filter: filter, result: result}
}

type optionalHandEntryRound struct {
	subjects PlayerSet
	starting PlayerRef
	filter   CardFilter
	result   *NumberBinding
}

func (*optionalHandEntryRound) Text() string {
	return "each player may put a card from hand onto the battlefield, starting with the selected player"
}
func (*optionalHandEntryRound) Properties() EffectProperties { return EffectProperties{} }
func (e *optionalHandEntryRound) inspectDefinition() definitionNode {
	return definitionNode{Role: "optional-hand-entry-round", bindingWrite: e.result, children: []definitionEdge{{"players", e.subjects}, {"starting player", e.starting}, {"entries", e.result}}}
}
func (e *optionalHandEntryRound) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	ctx.Game.resolution.numberBindings[e.result] = 0
	start, ok := e.starting.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	selected := e.subjects.resolvePlayers(ctx)
	first := -1
	for i, p := range ctx.Game.players {
		if p.PlayerID() == start {
			first = i
			break
		}
	}
	if first < 0 {
		return nil
	}
	players := append([]Player(nil), ctx.Game.players[first:]...)
	players = append(players, ctx.Game.players[:first]...)
	for _, p := range players {
		if !slices.Contains(selected, p.PlayerID()) {
			continue
		}
		entry := &optionalHandEntry{filter: e.filter}
		entered, err := entry.applyPlayer(ctx, p.PlayerID())
		if err != nil {
			return err
		}
		if entered {
			ctx.Game.resolution.numberBindings[e.result]++
		}
	}
	return nil
}
func (e *optionalHandEntryRound) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects) || playerRefReadsSourcePermanent(e.starting)
}

func (s *ResolutionState) installRoundBindings(frame ResolutionState) {
	s.numberBindings = frame.numberBindings
	s.playerBindings = frame.playerBindings
	s.permanentBindings = frame.permanentBindings
	s.createdTokenBindings = frame.createdTokenBindings
	s.cardBindings = frame.cardBindings
	s.cardsBindings = frame.cardsBindings
	s.spellBindings = frame.spellBindings
	s.wordPairBindings = frame.wordPairBindings
	s.cardNameBindings = frame.cardNameBindings
	s.manaTypesBindings = frame.manaTypesBindings
	s.lostManaBindings = frame.lostManaBindings
	s.damageSourceBindings = frame.damageSourceBindings
	s.damageResultsBindings = frame.damageResultsBindings
	s.damageResultBindings = frame.damageResultBindings
}
