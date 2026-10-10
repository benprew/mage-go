package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// PlayerNumbersBinding declares separate saved numbers for the two player seats.
// Its members use the shared number frames for copies, clones and delayed bodies.
type PlayerNumbersBinding struct{ members [2]*NumberBinding }

// BindPlayerNumbers declares a collection with a diagnostic name.
func BindPlayerNumbers(name string) *PlayerNumbersBinding {
	result := &PlayerNumbersBinding{}
	for i := range result.members {
		label := name
		if label != "" {
			label = fmt.Sprintf("%s player %d", name, i+1)
		}
		result.members[i] = BindNumber(label)
	}
	return result
}
func (b *PlayerNumbersBinding) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-numbers-binding", children: []definitionEdge{{"first", b.members[0]}, {"second", b.members[1]}}}
}
func (b *PlayerNumbersBinding) member(g *Game, id uuid.UUID) *NumberBinding {
	for i, p := range g.players {
		if p.PlayerID() == id {
			return b.members[i]
		}
	}
	return nil
}

// PlayerNumber reads the selected player's saved number. Absent results are zero.
func PlayerNumber(binding *PlayerNumbersBinding, player PlayerRef) ResolutionValue {
	return playerNumberValue{binding, player}
}

type playerNumberValue struct {
	binding *PlayerNumbersBinding
	player  PlayerRef
}

func (playerNumberValue) Text() string { return "saved player number" }
func (v playerNumberValue) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"player", v.player}, {"binding", v.binding}}}
	if v.binding != nil {
		for i, b := range v.binding.members {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("read[%d]", i), FromNumber(b)})
		}
	}
	return n
}
func (v playerNumberValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.player.resolvePlayer(ctx)
	if !ok {
		return 0
	}
	return ctx.Game.resolution.numberBindings[v.binding.member(ctx.Game, id)]
}

// DiscardUpToEach saves actual discard counts after one simultaneous batch.
// All player choices precede replacements and card movement.
func DiscardUpToEach(subjects PlayerSet, maximum ResolutionValue, result *PlayerNumbersBinding) ResolvingEffect {
	return &discardUpToEachEffect{subjects, maximum, result}
}

type discardUpToEachEffect struct {
	subjects PlayerSet
	maximum  ResolutionValue
	result   *PlayerNumbersBinding
}

func (*discardUpToEachEffect) Text() string { return "each player may discard up to a number of cards" }
func (*discardUpToEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *discardUpToEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"maximum", e.maximum}, {"result", e.result}}}
	if e.result != nil {
		for _, b := range e.result.members {
			n.bindingWrites = append(n.bindingWrites, b)
		}
	}
	return n
}
func (e *discardUpToEachEffect) Apply(ctx *EffectContext) error {
	values := ctx.Game.resolution.numberBindings
	if values == nil {
		values = make(map[*NumberBinding]int)
		ctx.Game.resolution.numberBindings = values
	}
	for _, b := range e.result.members {
		values[b] = 0
	}
	limit := max(0, e.maximum.resolveValue(ctx))
	selected := e.subjects.resolvePlayers(ctx)
	var actions []*DiscardAction
	seen := make(map[uuid.UUID]bool)
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if !slices.Contains(selected, id) {
			continue
		}
		player := ctx.Game.GetPlayer(id)
		if player == nil || seen[id] {
			continue
		}
		seen[id] = true
		maximum := min(limit, len(player.Hand()))
		if maximum == 0 {
			continue
		}
		count := max(0, min(maximum, player.ChooseNumber(0, maximum, "cards to discard")))
		if count == 0 {
			continue
		}
		chosen := player.ChooseCardsFromHand(count, "discard", ctx.Game)
		held := make(map[uuid.UUID]Card, len(player.Hand()))
		for _, c := range player.Hand() {
			held[c.ID()] = c
		}
		ids := make(map[uuid.UUID]bool)
		for _, c := range chosen {
			if c == nil || held[c.ID()] == nil || ids[c.ID()] {
				continue
			}
			ids[c.ID()] = true
			action := NewDiscardAction(ctx.SourceID, id, c.ID())
			action.handVisit = ctx.Game.handVisit(c.ID())
			actions = append(actions, action)
			if len(ids) == count {
				break
			}
		}
		for _, c := range player.Hand() {
			if len(ids) == count {
				break
			}
			if ids[c.ID()] {
				continue
			}
			ids[c.ID()] = true
			action := NewDiscardAction(ctx.SourceID, id, c.ID())
			action.handVisit = ctx.Game.handVisit(c.ID())
			actions = append(actions, action)
		}
	}
	for _, event := range ctx.Game.discardActionsTogetherWithEvents(actions, false) {
		if b := e.result.member(ctx.Game, event.PlayerID); b != nil {
			values[b]++
		}
	}
	return nil
}
func (e *discardUpToEachEffect) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.maximum)
}
