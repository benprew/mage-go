package mage

import (
	"fmt"
	"slices"
	"strings"
)

// NamedPermanentPile declares one pile and its saved battlefield visits.
type NamedPermanentPile struct {
	Name   string
	Result *PermanentSetBinding
}

// SeparatePermanentsIntoNamedPiles saves a required partition at resolution.
// Piles can be empty. Their members stay on the battlefield.
func SeparatePermanentsIntoNamedPiles(player PlayerRef, subjects PermanentSet, piles ...NamedPermanentPile) ResolvingEffect {
	return &namedPermanentPartition{player, subjects, slices.Clone(piles)}
}

type namedPermanentPartition struct {
	player   PlayerRef
	subjects PermanentSet
	piles    []NamedPermanentPile
}

func (*namedPermanentPartition) Text() string                 { return "separate permanents into named piles" }
func (*namedPermanentPartition) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *namedPermanentPartition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "named-permanent-partition", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"subjects", e.subjects}}}
	if len(e.piles) < 2 {
		n.problem = "supply at least two named piles"
	}
	names := make(map[string]bool, len(e.piles))
	results := make(map[*PermanentSetBinding]bool, len(e.piles))
	for _, pile := range e.piles {
		if strings.TrimSpace(pile.Name) == "" || names[pile.Name] {
			n.problem = "supply distinct nonempty pile names"
		}
		if pile.Result == nil || results[pile.Result] {
			n.problem = "supply distinct non-nil pile results"
		}
		names[pile.Name] = true
		results[pile.Result] = true
		n.bindingWrites = append(n.bindingWrites, pile.Result)
		n.children = append(n.children, definitionEdge{"pile result", pile.Result})
	}
	return n
}
func (e *namedPermanentPartition) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || permanentSetReadsSourcePermanent(e.subjects)
}
func (e *namedPermanentPartition) Apply(ctx *EffectContext) error {
	members := e.subjects.resolvePermanents(ctx)
	if ctx.Game.resolution.createdTokenBindings == nil {
		ctx.Game.resolution.createdTokenBindings = make(map[*PermanentSetBinding][]capturedPermanent)
	}
	for _, pile := range e.piles {
		delete(ctx.Game.resolution.createdTokenBindings, pile.Result)
	}
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	visits := make([]capturedPermanent, 0, len(members))
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
			visits = append(visits, visit)
		}
	}
	names := make([]string, len(e.piles))
	saved := make([][]capturedPermanent, len(e.piles))
	for i, pile := range e.piles {
		names[i] = pile.Name
	}
	previous := make([]string, 0, len(visits))
	for _, visit := range visits {
		live := ctx.Game.FindPermanent(visit.id)
		if live == nil || live.PhasedOut || live.incarnationID != visit.incarnation {
			continue
		}
		reason := fmt.Sprintf("Choose a pile for %s (%s)", live.Name(), live.ID())
		if len(previous) > 0 {
			reason += ". Previous choices: " + strings.Join(previous, "; ")
		}
		choice := player.ChooseMode(slices.Clone(names), reason)
		if choice < 0 || choice >= len(names) {
			choice = 0
		}
		live = ctx.Game.FindPermanent(visit.id)
		if live == nil || live.PhasedOut || live.incarnationID != visit.incarnation {
			continue
		}
		saved[choice] = append(saved[choice], visit)
		previous = append(previous, fmt.Sprintf("%s (%s): %s", live.Name(), live.ID(), names[choice]))
	}
	for i, pile := range e.piles {
		ctx.Game.resolution.createdTokenBindings[pile.Result] = saved[i]
	}
	return nil
}
