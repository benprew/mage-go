package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// LeastControlledPermanentCount reads the smallest selected player's count.
// An empty player set supplies zero. Phased and old visits cannot count.
func LeastControlledPermanentCount(players PlayerSet, subjects PermanentSet) ResolutionValue {
	return leastControlledPermanentCount{players, subjects}
}

type leastControlledPermanentCount struct {
	players  PlayerSet
	subjects PermanentSet
}

func (leastControlledPermanentCount) Text() string { return "least controlled permanent count" }
func (v leastControlledPermanentCount) resolveValue(ctx *EffectContext) int {
	groups := controlledPermanentGroups(ctx, v.players, v.subjects)
	if len(groups) == 0 {
		return 0
	}
	least := len(groups[0].members)
	for _, group := range groups[1:] {
		least = min(least, len(group.members))
	}
	return least
}
func (v leastControlledPermanentCount) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"players", v.players}, {"subjects", v.subjects}}}
}

// SacrificeAllButEach lets each selected player choose which subjects to keep.
// All choices precede the shared simultaneous sacrifice operation.
func SacrificeAllButEach(players PlayerSet, subjects PermanentSet, keep ResolutionValue) ResolvingEffect {
	return &sacrificeAllButEach{players, subjects, keep}
}

type sacrificeAllButEach struct {
	players  PlayerSet
	subjects PermanentSet
	keep     ResolutionValue
}

func (*sacrificeAllButEach) Text() string {
	return "each player sacrifices all but a number of permanents"
}
func (*sacrificeAllButEach) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *sacrificeAllButEach) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"players", e.players}, {"subjects", e.subjects}, {"keep", e.keep}}}
}
func (e *sacrificeAllButEach) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.players) || permanentSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.keep)
}
func (e *sacrificeAllButEach) Apply(ctx *EffectContext) error {
	count := max(0, e.keep.resolveValue(ctx))
	groups := controlledPermanentGroups(ctx, e.players, e.subjects)
	var sacrifices []*Permanent
	previous := make([]string, 0, len(groups))
	for _, group := range groups {
		reason := "Choose permanents to keep"
		if len(previous) > 0 {
			reason += ". Previous choices: " + strings.Join(previous, "; ")
		}
		kept := choosePermanentSubset(group.player, group.members, count, reason, ctx.Game)
		names := make([]string, 0, len(kept))
		for _, p := range kept {
			names = append(names, fmt.Sprintf("%s (%s)", p.Name(), p.ID()))
		}
		previous = append(previous, group.player.Name()+": "+strings.Join(names, ", "))
		for _, p := range group.members {
			if slices.Contains(kept, p) {
				continue
			}
			current := ctx.Game.FindPermanent(p.ID())
			if current != nil && !current.PhasedOut && current.incarnationID == p.incarnationID && current.ControllerID() == group.player.PlayerID() {
				sacrifices = append(sacrifices, p)
			}
		}
	}
	ctx.Game.controllersSacrificePermanents(sacrifices)
	return nil
}

type controlledPermanentGroup struct {
	player  Player
	members []*Permanent
}

func controlledPermanentGroups(ctx *EffectContext, players PlayerSet, subjects PermanentSet) []controlledPermanentGroup {
	ids := players.resolvePlayers(ctx)
	members := subjects.resolvePermanents(ctx)
	var groups []controlledPermanentGroup
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if !slices.Contains(ids, id) {
			continue
		}
		group := controlledPermanentGroup{player: ctx.Game.GetPlayer(id)}
		seen := make(map[uuid.UUID]bool)
		for _, p := range members {
			if p == nil || seen[p.ID()] {
				continue
			}
			current := ctx.Game.FindPermanent(p.ID())
			if current != nil && !current.PhasedOut && current.incarnationID == p.incarnationID && current.ControllerID() == id {
				seen[p.ID()] = true
				group.members = append(group.members, current)
			}
		}
		groups = append(groups, group)
	}
	return groups
}

func choosePermanentSubset(player Player, candidates []*Permanent, count int, reason string, g *Game) []*Permanent {
	count = min(max(0, count), len(candidates))
	if count == len(candidates) {
		return slices.Clone(candidates)
	}
	remaining := slices.Clone(candidates)
	selected := make([]*Permanent, 0, count)
	for range count {
		chosen := player.ChoosePermanent(slices.Clone(remaining), reason, g)
		index := max(slices.IndexFunc(remaining, func(p *Permanent) bool {
			return chosen != nil && p.ID() == chosen.ID() && p.incarnationID == chosen.incarnationID
		}), 0)
		selected = append(selected, remaining[index])
		remaining = slices.Delete(remaining, index, index+1)
	}
	return selected
}
