package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// TrySwapBlockingAssignments exchanges two creatures' blocks if both can block the new sets.
// A failed check leaves both creatures in combat with their previous assignments.
func TrySwapBlockingAssignments(subjects PermanentSet) ResolvingEffect {
	return &blockAssignmentSwap{subjects: subjects}
}

type blockAssignmentSwap struct{ subjects PermanentSet }

func (*blockAssignmentSwap) Text() string {
	return "swap two blocking creatures' assignments if each could block the other set"
}
func (*blockAssignmentSwap) Properties() EffectProperties { return EffectProperties{} }
func (e *blockAssignmentSwap) inspectDefinition() definitionNode {
	return definitionNode{Role: "block-assignment-swap", children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *blockAssignmentSwap) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}

func (g *Game) presentBlockingCreature(p *Permanent) bool {
	return p != nil && g.FindPermanent(p.ID()) == p && !p.PhasedOut && p.HasType(core.TypeCreature) && g.combat.IsBlockingIncarnation(p.ID(), p.incarnationID)
}

func (g *Game) blockedGroups(blocker *Permanent) []*CombatGroup {
	var groups []*CombatGroup
	for _, group := range g.combat.Groups {
		if g.liveCombatAttacker(group) == nil {
			continue
		}
		for i, id := range group.BlockerIDs {
			if id == blocker.ID() && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == blocker.incarnationID {
				groups = append(groups, group)
				break
			}
		}
	}
	return groups
}

func (g *Game) couldBlockGroups(blocker *Permanent, groups []*CombatGroup) bool {
	if len(groups) == 0 {
		return true
	}
	if !blocker.CanDeclareAsBlocker(g) {
		return false
	}
	bands := make(map[uuid.UUID][]*CombatGroup)
	for _, group := range groups {
		key := group.AttackerID
		if band := g.combat.Bands[key]; len(band) > 0 {
			key = band[0]
		}
		bands[key] = append(bands[key], group)
	}
	if !g.effects.blockAnyNumberRules[blocker.ID()] && len(bands) > blockCapacity(blocker, len(bands)) {
		return false
	}
	for _, members := range bands {
		legal := false
		for _, group := range members {
			attacker := g.liveCombatAttacker(group)
			defender := group.DefenderID
			if p := g.FindPermanent(defender); p != nil {
				defender = p.ControllerID()
			}
			if attacker == nil || defender != blocker.ControllerID() || attacker.ControllerID() == blocker.ControllerID() {
				continue
			}
			minimum := max(1, g.effects.MinBlockers(attacker.ID()))
			if attacker.HasKeyword(core.Menace) {
				minimum = max(minimum, 2)
			}
			if len(group.BlockerIDs) >= minimum && CanBlock(blocker, attacker, g) && !HasLandwalkEvasion(attacker, defender, g) {
				legal = true
			}
		}
		if !legal {
			return false
		}
	}
	return true
}

func (e *blockAssignmentSwap) Apply(ctx *EffectContext) error {
	subjects := e.subjects.resolvePermanents(ctx)
	if len(subjects) != 2 || subjects[0].ID() == subjects[1].ID() {
		return nil
	}
	g := ctx.Game
	first, second := subjects[0], subjects[1]
	if !g.presentBlockingCreature(first) || !g.presentBlockingCreature(second) || first.ControllerID() != second.ControllerID() {
		return nil
	}
	firstGroups, secondGroups := g.blockedGroups(first), g.blockedGroups(second)
	if !g.couldBlockGroups(first, secondGroups) || !g.couldBlockGroups(second, firstGroups) {
		return nil
	}
	assignments := make([]effectBlockAssignment, 0, len(firstGroups)+len(secondGroups))
	for i, groups := range [][]*CombatGroup{secondGroups, firstGroups} {
		for _, group := range groups {
			attacker := g.liveCombatAttacker(group)
			assignments = append(assignments, effectBlockAssignment{blocker: capturedPermanent{subjects[i].ID(), subjects[i].incarnationID}, attacker: capturedPermanent{attacker.ID(), attacker.incarnationID}})
		}
	}
	g.combat.RemoveFromCombat(first.ID())
	g.combat.RemoveFromCombat(second.ID())
	g.addEffectBlockers(assignments)
	return nil
}
