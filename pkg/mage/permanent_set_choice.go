package mage

import (
	"fmt"
	"slices"
	"strings"
)

// PermanentSetBinding names one saved set of exact permanent visits.
// Shared resolution frames keep each result separate through copies and delays.
type PermanentSetBinding struct{ name string }

// BindPermanentSet declares one instance-local permanent set result.
func BindPermanentSet(name string) *PermanentSetBinding { return &PermanentSetBinding{name: name} }

// FromPermanentSet selects present, unphased visits from a saved result.
func FromPermanentSet(binding *PermanentSetBinding) PermanentSet { return savedPermanentSet{binding} }

// WithoutPermanents removes the exact members of the excluded set.
func WithoutPermanents(subjects, excluded PermanentSet) PermanentSet {
	return differencePermanentSet{subjects, excluded}
}

type differencePermanentSet struct{ subjects, excluded PermanentSet }

func (differencePermanentSet) permanentSet() {}
func (s differencePermanentSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "set difference", children: []definitionEdge{{"subjects", s.subjects}, {"excluded", s.excluded}}}
}
func (s differencePermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	excluded := s.excluded.resolvePermanents(ctx)
	result := slices.Clone(s.subjects.resolvePermanents(ctx))
	return slices.DeleteFunc(result, func(p *Permanent) bool {
		return slices.ContainsFunc(excluded, func(other *Permanent) bool {
			return p != nil && other != nil && p.ID() == other.ID() && p.incarnationID == other.incarnationID
		})
	})
}

// ChooseUpToPermanentsEach saves the union of repeated optional choices.
// Each player uses their saved repetition count. Other players' subjects can match.
func ChooseUpToPermanentsEach(players PlayerSet, subjects PermanentSet, limit ResolutionValue, repetitions *PlayerNumbersBinding, result *PermanentSetBinding) ResolvingEffect {
	return &chooseUpToPermanentsEach{players, subjects, limit, repetitions, result}
}

type chooseUpToPermanentsEach struct {
	players     PlayerSet
	subjects    PermanentSet
	limit       ResolutionValue
	repetitions *PlayerNumbersBinding
	result      *PermanentSetBinding
}

func (*chooseUpToPermanentsEach) Text() string {
	return "each player chooses up to a number of permanents for each saved result"
}
func (*chooseUpToPermanentsEach) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *chooseUpToPermanentsEach) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingWrite: e.result, children: []definitionEdge{{"players", e.players}, {"subjects", e.subjects}, {"limit", e.limit}, {"repetitions", PlayerNumber(e.repetitions, You)}, {"result", e.result}}}
}
func (e *chooseUpToPermanentsEach) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.players) || permanentSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.limit)
}
func (e *chooseUpToPermanentsEach) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.createdTokenBindings == nil {
		ctx.Game.resolution.createdTokenBindings = make(map[*CreatedTokensBinding][]capturedPermanent)
	}
	delete(ctx.Game.resolution.createdTokenBindings, e.result)
	limit := max(0, e.limit.resolveValue(ctx))
	selected := e.players.resolvePlayers(ctx)
	members := e.subjects.resolvePermanents(ctx)
	candidates := make([]*Permanent, 0, len(members))
	for _, p := range members {
		if p == nil {
			continue
		}
		current := ctx.Game.FindPermanent(p.ID())
		if current != nil && !current.PhasedOut && current.incarnationID == p.incarnationID && !slices.Contains(candidates, current) {
			candidates = append(candidates, current)
		}
	}
	type choiceGroup struct {
		player Player
		count  int
	}
	groups := make([]choiceGroup, 0, len(selected))
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if slices.Contains(selected, id) {
			groups = append(groups, choiceGroup{ctx.Game.GetPlayer(id), max(0, ctx.Game.resolution.numberBindings[e.repetitions.member(ctx.Game, id)])})
		}
	}
	saved := make([]capturedPermanent, 0, len(candidates))
	previous := make([]string, 0, len(groups))
	for _, group := range groups {
		names := []string{}
		for range group.count {
			available := make([]*Permanent, 0, len(candidates))
			for _, p := range candidates {
				current := ctx.Game.FindPermanent(p.ID())
				if current != nil && !current.PhasedOut && current.incarnationID == p.incarnationID {
					available = append(available, current)
				}
			}
			maximum := min(limit, len(available))
			if maximum == 0 {
				continue
			}
			reason := group.player.Name() + ": Choose permanents to keep"
			if len(previous) > 0 {
				reason += ". Previous choices: " + strings.Join(previous, "; ")
			}
			count := max(0, min(maximum, group.player.ChooseNumber(0, maximum, reason)))
			for _, p := range choosePermanentSubset(group.player, available, count, reason, ctx.Game) {
				visit := capturedPermanent{p.ID(), p.incarnationID}
				if !slices.Contains(saved, visit) {
					saved = append(saved, visit)
				}
				names = append(names, fmt.Sprintf("%s (%s)", p.Name(), p.ID()))
			}
		}
		previous = append(previous, group.player.Name()+": "+strings.Join(names, ", "))
	}
	ctx.Game.resolution.createdTokenBindings[e.result] = saved
	return nil
}
