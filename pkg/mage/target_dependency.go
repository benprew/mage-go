package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// TargetSharesCardTypeWith compares a target with an earlier singular target.
// Both visits must be present. An omitted list permits every card type.
func TargetSharesCardTypeWith(reference *PermanentTargetBinding, types ...core.CardType) PermanentTargetOption {
	return targetTypeDependency{reference, slices.Clone(types)}
}

type targetTypeDependency struct {
	reference *PermanentTargetBinding
	types     []core.CardType
}

func (targetTypeDependency) permanentTargetOption() {}
func (o targetTypeDependency) applyPermanentTarget(t *PermanentTarget) {
	t.typeDependencies = append(t.typeDependencies, o)
}
func (o targetTypeDependency) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-type-dependency", children: []definitionEdge{{"reference", o.reference}}}
	seen := make(map[core.CardType]bool)
	for _, kind := range o.types {
		if !slices.Contains(core.CardTypeValues(), kind) || seen[kind] {
			n.problem = "target comparison requires distinct valid card types"
		}
		seen[kind] = true
	}
	return n
}
func (o targetTypeDependency) matches(p *Permanent, ctx *EffectContext) bool {
	if o.reference == nil || o.reference.targetDeclaration == nil {
		return false
	}
	ref, ok := o.reference.resolvePermanent(ctx)
	if !ok {
		return false
	}
	return sharesPermanentCardType(p, ref, nil, o.types)
}

func hasTargetDependencies(specs []Target) bool {
	return slices.ContainsFunc(specs, func(spec Target) bool {
		if nilDefinition(spec) {
			return false
		}
		if _, ok := UnwrapTarget(spec).(*playerGraveyardTargets); ok {
			return true
		}
		t, ok := UnwrapTarget(spec).(*PermanentTarget)
		return ok && t != nil && (len(t.typeDependencies) != 0 || len(t.blockHistoryDependencies) != 0)
	})
}

func (g *Game) targetBindingsQuery(specs []Target, chosen []uuid.UUID, visits map[uuid.UUID]uuid.UUID, x int) *Game {
	query := *g
	query.resolution.SetX(x)
	query.resolution.targetQueryBindings = cloneDelayedTargets(g.resolution.targetQueryBindings)
	if query.resolution.targetQueryBindings == nil {
		query.resolution.targetQueryBindings = make(map[*targetDeclaration][]capturedPermanent)
	}
	for _, spec := range specs {
		if b, ok := spec.(*targetDeclaration); ok {
			delete(query.resolution.targetQueryBindings, b)
		}
	}
	aligned := expandTargetSpecs(specs, chosen, x)
	for i, id := range chosen {
		if i >= len(aligned) {
			break
		}
		b, ok := aligned[i].(*targetDeclaration)
		if !ok || id == uuid.Nil {
			continue
		}
		visit := visits[id]
		if visit == uuid.Nil {
			if p := g.FindPermanent(id); p != nil {
				visit = p.incarnationID
			}
		}
		query.resolution.targetQueryBindings[b] = append(query.resolution.targetQueryBindings[b], capturedPermanent{id, visit})
	}
	return &query
}

// PossibleTargetGroup reads one group after the supplied earlier choices.
// A player target can precede one optional graveyard card collection.
// The query does not choose targets or change the current resolution frame.
// Each candidate must permit a complete selection for the remaining groups.
func (g *Game) PossibleTargetGroup(controller uuid.UUID, card Card, specs []Target, group int, chosen []uuid.UUID, x int) []uuid.UUID {
	if mixedPlayerGraveyardTargets(specs) {
		if group < 0 || group >= len(specs) || len(chosen) != group {
			return nil
		}
		if err := g.validateActionTargets(controller, card, specs[:group], chosen, x, "target prefix"); err != nil {
			return nil
		}
		return g.targetBindingsQuery(specs[:group], chosen, nil, x).possibleTargetsWithX(specs[group], controller, card, x)
	}
	if group < 0 || group >= len(specs) || len(chosen) != group {
		return nil
	}
	for _, spec := range specs {
		if nilDefinition(spec) {
			return nil
		}
		minimum, maximum := TargetBounds(spec, x)
		if minimum != 1 || maximum != 1 {
			return nil
		}
	}
	if err := g.validateActionTargets(controller, card, specs[:group], chosen, x, "target prefix"); err != nil {
		return nil
	}
	candidates := g.targetBindingsQuery(specs[:group], chosen, nil, x).possibleTargetsWithX(specs[group], controller, card, x)
	if !hasTargetDependencies(specs) {
		return candidates
	}
	var result []uuid.UUID
	for _, id := range candidates {
		if g.targetSelectionCanComplete(controller, card, specs, append(slices.Clone(chosen), id), x) {
			result = append(result, id)
		}
	}
	return result
}

func (g *Game) targetSelectionCanComplete(controller uuid.UUID, card Card, specs []Target, chosen []uuid.UUID, x int) bool {
	if len(chosen) == len(specs) {
		return true
	}
	query := g.targetBindingsQuery(specs[:len(chosen)], chosen, nil, x)
	for _, id := range query.possibleTargetsWithX(specs[len(chosen)], controller, card, x) {
		if g.targetSelectionCanComplete(controller, card, specs, append(slices.Clone(chosen), id), x) {
			return true
		}
	}
	return false
}

func (g *Game) targetCandidatesForList(controller uuid.UUID, card Card, specs []Target, group int, chosen []uuid.UUID, x int) []uuid.UUID {
	if hasTargetDependencies(specs) {
		return g.PossibleTargetGroup(controller, card, specs, group, chosen, x)
	}
	return g.possibleTargetsWithX(specs[group], controller, card, x)
}

func (g *Game) hasLegalTargetSelection(controller uuid.UUID, card Card, specs []Target, x int) bool {
	if !hasTargetDependencies(specs) {
		for _, spec := range specs {
			minimum, _ := TargetBounds(spec, x)
			if len(uniqueTargetCandidates(g.possibleTargetsWithX(spec, controller, card, x))) < minimum {
				return false
			}
		}
		return true
	}
	return len(g.PossibleTargetGroup(controller, card, specs, 0, nil, x)) != 0
}

func targetDependencyProblem(specs []Target) string {
	if mixedPlayerGraveyardTargets(specs) {
		return ""
	}
	for _, spec := range specs {
		if _, ok := UnwrapTarget(spec).(*playerGraveyardTargets); ok {
			return "graveyard collection requires an earlier singular player target"
		}
	}
	if !hasTargetDependencies(specs) {
		return ""
	}
	prior := make(map[*targetDeclaration]bool)
	for i, spec := range specs {
		b, ok := spec.(*targetDeclaration)
		if !ok || b == nil {
			return "dependent targets require singular permanent declarations"
		}
		t, ok := b.spec.(*PermanentTarget)
		if !ok || t == nil || t.Min() != 1 || t.Max() != 1 {
			return "dependent targets require singular permanent declarations"
		}
		for _, dependency := range t.typeDependencies {
			if dependency.reference == nil || !prior[dependency.reference.targetDeclaration] {
				return fmt.Sprintf("target group %d requires an earlier singular permanent declaration", i+1)
			}
		}
		for _, dependency := range t.blockHistoryDependencies {
			if dependency.reference == nil || !prior[dependency.reference.targetDeclaration] {
				return fmt.Sprintf("target group %d requires an earlier singular permanent declaration", i+1)
			}
		}
		prior[b] = true
	}
	return ""
}

func mixedPlayerGraveyardTargets(specs []Target) bool {
	if len(specs) != 2 {
		return false
	}
	first, ok := specs[0].(*targetDeclaration)
	if !ok {
		return false
	}
	second, ok := UnwrapTarget(specs[1]).(*playerGraveyardTargets)
	if !ok || second.player == nil {
		return false
	}
	return second.player.targetDeclaration == first
}
