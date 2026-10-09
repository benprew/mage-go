package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PermanentExileGroup declares one subject set and its saved exile visits.
type PermanentExileGroup struct {
	Subjects PermanentSet
	Result   *CardsBinding
}

// ExilePermanentGroupsInto selects all groups before one simultaneous exile.
func ExilePermanentGroupsInto(groups ...PermanentExileGroup) ResolvingEffect {
	return &groupedExileEffect{groups: append([]PermanentExileGroup(nil), groups...)}
}

type groupedExileEffect struct{ groups []PermanentExileGroup }

func (*groupedExileEffect) Text() string { return "exile permanent groups with saved visits" }
func (*groupedExileEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true, TargetPurposeOverride: AITargetExile}
}
func (e *groupedExileEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text()}
	if len(e.groups) == 0 {
		n.problem = "supply at least one exile group"
	}
	seen := make(map[*CardsBinding]bool)
	for i, group := range e.groups {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("groups[%d].subjects", i), group.Subjects}, definitionEdge{fmt.Sprintf("groups[%d].result", i), group.Result})
		if group.Result == nil {
			n.problem = "supply a non-nil result for each exile group"
		} else {
			if seen[group.Result] {
				n.problem = "supply a distinct result for each exile group"
			}
			seen[group.Result] = true
			n.bindingWrites = append(n.bindingWrites, group.Result)
		}
	}
	return n
}
func (e *groupedExileEffect) refersToSourcePermanent() bool {
	for _, group := range e.groups {
		if permanentSetReadsSourcePermanent(group.Subjects) {
			return true
		}
	}
	return false
}
func (e *groupedExileEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	if g.resolution.cardsBindings == nil {
		g.resolution.cardsBindings = make(map[*CardsBinding][]capturedCard)
	}
	selected := make([][]capturedPermanent, len(e.groups))
	var all []*Permanent
	seen := make(map[uuid.UUID]bool)
	for i, group := range e.groups {
		delete(g.resolution.cardsBindings, group.Result)
		for _, p := range group.Subjects.resolvePermanents(ctx) {
			if p == nil {
				continue
			}
			current := g.FindPermanent(p.ID())
			if current == nil || current.incarnationID != p.incarnationID || current.PhasedOut {
				continue
			}
			selected[i] = append(selected[i], capturedPermanent{id: p.ID(), incarnation: p.incarnationID})
			if !seen[p.incarnationID] {
				seen[p.incarnationID] = true
				all = append(all, current)
			}
		}
	}
	g.withZoneChangeBatch(func() {
		g.ExilePermanents(all)
		moved := make(map[uuid.UUID]capturedCard)
		for _, p := range all {
			if exiled := g.FindExiledCard(p.ID()); exiled != nil && !p.IsToken {
				moved[p.incarnationID] = capturedCard{id: p.ID(), visit: exiled.Visit, zone: core.ZoneExile}
			}
		}
		for i, group := range e.groups {
			saved := make(map[uuid.UUID]bool)
			for _, subject := range selected[i] {
				if visit, ok := moved[subject.incarnation]; ok && !saved[subject.incarnation] {
					g.resolution.cardsBindings[group.Result] = append(g.resolution.cardsBindings[group.Result], visit)
					saved[subject.incarnation] = true
				}
			}
		}
	})
	return nil
}

// ExiledCardsFrom reads only current exile visits in a saved result group.
func ExiledCardsFrom(binding *CardsBinding) ExiledCardSet {
	return savedExiledCardsSet{binding: binding}
}

type savedExiledCardsSet struct{ binding *CardsBinding }

func (savedExiledCardsSet) exiledCardSet() {}
func (s savedExiledCardsSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "exiled-card-set", bindingRead: s.binding, children: []definitionEdge{{"binding", s.binding}}}
}
func (s savedExiledCardsSet) resolveExiledCards(ctx *EffectContext) []capturedCard {
	var result []capturedCard
	for _, visit := range ctx.Game.resolution.cardsBindings[s.binding] {
		if visit.zone != core.ZoneExile || visit.visit == uuid.Nil {
			continue
		}
		if exiled := ctx.Game.FindExiledCard(visit.id); exiled != nil && exiled.Visit == visit.visit {
			result = append(result, visit)
		}
	}
	return result
}

// SingleExiledCardFrom requires one original result and its current exile visit.
func SingleExiledCardFrom(binding *CardsBinding) CardRef {
	return singleExiledCardRef{binding: binding}
}

type singleExiledCardRef struct{ binding *CardsBinding }

func (singleExiledCardRef) cardRef() {}
func (r singleExiledCardRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-ref", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}
func (r singleExiledCardRef) resolveCard(ctx *EffectContext) (Card, bool) {
	saved := ctx.Game.resolution.cardsBindings[r.binding]
	if len(saved) != 1 || saved[0].zone != core.ZoneExile || saved[0].visit == uuid.Nil {
		return nil, false
	}
	exiled := ctx.Game.FindExiledCard(saved[0].id)
	if exiled == nil || exiled.Visit != saved[0].visit || !exiled.VisibleTo(ctx.Controller) {
		return nil, false
	}
	return exiled.Card, true
}
