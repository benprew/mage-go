package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// MayHavePermanentBlockAttacker offers a block during resolution.
// Declaration restrictions do not apply to this effect.
func MayHavePermanentBlockAttacker(player PlayerRef, blocker PermanentRef, attackers PermanentSet) ResolvingEffect {
	return &effectBlockingChoice{player: player, blocker: blocker, attackers: attackers}
}

type effectBlockingChoice struct {
	player    PlayerRef
	blocker   PermanentRef
	attackers PermanentSet
}

func (*effectBlockingChoice) Text() string {
	return "you may have a creature block an attacking creature"
}
func (*effectBlockingChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *effectBlockingChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "effect-blocking-choice", children: []definitionEdge{
		{"player", e.player}, {"blocker", e.blocker}, {"attackers", e.attackers},
	}}
}
func (e *effectBlockingChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || permanentRefReadsSourcePermanent(e.blocker) || permanentSetReadsSourcePermanent(e.attackers)
}

func (g *Game) canBlockByEffect(blocker, attacker *Permanent) bool {
	if blocker == nil || attacker == nil || blocker.PhasedOut || !blocker.HasType(core.TypeCreature) || g.combat.IsAttacking(blocker.ID()) {
		return false
	}
	group := g.combat.GroupFor(attacker.ID())
	if group == nil || g.liveCombatAttacker(group) != attacker || attacker.ControllerID() == blocker.ControllerID() {
		return false
	}
	defender := group.DefenderID
	if permanent := g.FindPermanent(defender); permanent != nil {
		defender = permanent.ControllerID()
	}
	if defender != blocker.ControllerID() {
		return false
	}
	for i, id := range group.BlockerIDs {
		if id == blocker.ID() && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == blocker.incarnationID {
			return false
		}
	}
	return true
}

func (e *effectBlockingChoice) Apply(ctx *EffectContext) error {
	playerID, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(playerID)
	blocker, ok := e.blocker.resolvePermanent(ctx)
	if player == nil || !ok {
		return nil
	}
	blockerVisit := capturedPermanent{blocker.ID(), blocker.incarnationID}
	var candidates []*Permanent
	visits := make(map[capturedPermanent]bool)
	for _, attacker := range e.attackers.resolvePermanents(ctx) {
		if ctx.Game.canBlockByEffect(blocker, attacker) {
			visit := capturedPermanent{attacker.ID(), attacker.incarnationID}
			if !visits[visit] {
				candidates = append(candidates, attacker)
				visits[visit] = true
			}
		}
	}
	if len(candidates) == 0 || !player.ChooseMayAbility("have "+blocker.Name()+" block an attacking creature") {
		return nil
	}
	chosen := player.ChoosePermanent(candidates, "Choose an attacking creature to block", ctx.Game)
	if chosen == nil || !visits[capturedPermanent{chosen.ID(), chosen.incarnationID}] {
		return nil
	}
	attackerVisit := capturedPermanent{chosen.ID(), chosen.incarnationID}
	blocker = ctx.Game.FindPermanent(blockerVisit.id)
	attacker := ctx.Game.FindPermanent(attackerVisit.id)
	if blocker == nil || blocker.incarnationID != blockerVisit.incarnation || attacker == nil || attacker.incarnationID != attackerVisit.incarnation || !ctx.Game.canBlockByEffect(blocker, attacker) {
		return nil
	}
	ctx.Game.addEffectBlockers([]effectBlockAssignment{{blocker: blockerVisit, attacker: attackerVisit}})
	return nil
}

type effectBlockAssignment struct{ blocker, attacker capturedPermanent }

func (g *Game) addEffectBlockers(assignments []effectBlockAssignment) bool {
	for _, assignment := range assignments {
		blocker := g.FindPermanent(assignment.blocker.id)
		attacker := g.FindPermanent(assignment.attacker.id)
		if blocker == nil || blocker.incarnationID != assignment.blocker.incarnation || attacker == nil || attacker.incarnationID != assignment.attacker.incarnation || !g.canBlockByEffect(blocker, attacker) {
			return false
		}
	}
	blockedBefore := g.blockedAttackersBeforeDeclaration()
	pairsBefore := g.blockerPairsBeforeDeclaration()
	for _, assignment := range assignments {
		g.combat.AddBlockerIncarnation(assignment.blocker.id, assignment.blocker.incarnation, assignment.attacker.id)
	}
	for pair := range g.blockerPairsBeforeDeclaration() {
		if !pairsBefore[pair] {
			g.recordBlockHistory(g.FindPermanent(pair.blocker), g.FindPermanent(pair.attacker))
		}
	}
	g.fireDeclaredBlockerPairs(pairsBefore)
	g.fireNewBlockerPairs(pairsBefore)
	g.fireNewBlockerGroups(pairsBefore)
	g.fireNewlyBlockedAttackers(blockedBefore)
	return true
}
