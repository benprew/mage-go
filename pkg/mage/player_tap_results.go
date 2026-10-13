package mage

import (
	"fmt"
	"slices"
	"strings"
)

// TapAnyNumberEach saves actual tap counts after simultaneous optional choices.
// Each player can select only present untapped subjects under their control.
func TapAnyNumberEach(players PlayerSet, subjects PermanentSet, result *PlayerNumbersBinding) ResolvingEffect {
	return &tapAnyNumberEach{players, subjects, result}
}

type tapAnyNumberEach struct {
	players  PlayerSet
	subjects PermanentSet
	result   *PlayerNumbersBinding
}

func (*tapAnyNumberEach) Text() string {
	return "each player may tap any number of matching permanents"
}
func (*tapAnyNumberEach) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *tapAnyNumberEach) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"players", e.players}, {"subjects", e.subjects}, {"result", e.result}}}
	if e.result != nil {
		for _, b := range e.result.members {
			n.bindingWrites = append(n.bindingWrites, b)
		}
	}
	return n
}
func (e *tapAnyNumberEach) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.players) || permanentSetReadsSourcePermanent(e.subjects)
}
func (e *tapAnyNumberEach) Apply(ctx *EffectContext) error {
	values := ctx.Game.resolution.numberBindings
	if values == nil {
		values = make(map[*NumberBinding]int)
		ctx.Game.resolution.numberBindings = values
	}
	for _, b := range e.result.members {
		values[b] = 0
	}
	groups := controlledPermanentGroups(ctx, e.players, FilterPermanentSet(e.subjects, IsUntapped))
	previous := make([]string, 0, len(groups))
	selected := make([]controlledPermanentGroup, 0, len(groups))
	for _, group := range groups {
		reason := group.player.Name() + ": Choose permanents to tap"
		if len(previous) > 0 {
			reason += ". Previous choices: " + strings.Join(previous, "; ")
		}
		count := 0
		if len(group.members) > 0 {
			count = max(0, min(len(group.members), group.player.ChooseNumber(0, len(group.members), reason)))
		}
		chosen := choosePermanentSubset(group.player, group.members, count, reason, ctx.Game)
		names := make([]string, 0, len(chosen))
		for _, p := range chosen {
			names = append(names, fmt.Sprintf("%s (%s)", p.Name(), p.ID()))
		}
		previous = append(previous, group.player.Name()+": "+strings.Join(names, ", "))
		selected = append(selected, controlledPermanentGroup{group.player, chosen})
	}
	taps := make([]*Permanent, 0)
	for _, group := range selected {
		for _, p := range group.members {
			current := ctx.Game.FindPermanent(p.ID())
			if current == nil || current.PhasedOut || current.Tapped || current.incarnationID != p.incarnationID || current.ControllerID() != group.player.PlayerID() || slices.Contains(taps, current) {
				continue
			}
			taps = append(taps, current)
			values[e.result.member(ctx.Game, group.player.PlayerID())]++
		}
	}
	ctx.Game.tapPermanentsTogether(taps)
	return nil
}
