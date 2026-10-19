package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RandomizeBlockingPilesThisTurn replaces blocker declarations until cleanup.
// Defenders choose piles before the engine assigns them to attackers at random.
func RandomizeBlockingPilesThisTurn() ResolvingEffect { return randomBlockingPilesEffect{} }

type randomBlockingPilesEffect struct{}

func (randomBlockingPilesEffect) Text() string {
	return "defenders choose random blocking piles this turn"
}
func (randomBlockingPilesEffect) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e randomBlockingPilesEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "random-blocking-piles", Text: e.Text()}
}
func (randomBlockingPilesEffect) Apply(ctx *EffectContext) error {
	ctx.Game.turns.randomBlockingPiles = true
	return nil
}

func (g *Game) randomBlockingAssignments(defender uuid.UUID) []BlockAssignment {
	player := g.GetPlayer(defender)
	if player == nil {
		return nil
	}
	var attackers []capturedPermanent
	for _, group := range g.combat.Groups {
		target := group.DefenderID
		if p := g.FindPermanent(target); p != nil {
			target = p.ControllerID()
		}
		if p := g.liveCombatAttacker(group); p != nil && target == defender {
			attackers = append(attackers, capturedPermanent{p.ID(), p.incarnationID})
		}
	}
	if len(attackers) == 0 {
		return nil
	}
	piles := make([][]capturedPermanent, len(attackers))
	var creatures []capturedPermanent
	for _, p := range g.AllBattlefield() {
		if !p.PhasedOut && p.HasType(core.TypeCreature) && p.ControllerID() == defender {
			creatures = append(creatures, capturedPermanent{p.ID(), p.incarnationID})
		}
	}
	for _, visit := range creatures {
		creature := g.FindPermanent(visit.id)
		if creature == nil || creature.incarnationID != visit.incarnation {
			continue
		}
		capacity := blockCapacity(creature, len(piles))
		if g.effects.blockAnyNumberRules[creature.ID()] {
			capacity = len(piles)
		}
		available := make([]int, len(piles))
		for i := range available {
			available[i] = i
		}
		for range capacity {
			modes := []string{"Do not add this creature to another pile"}
			for _, index := range available {
				modes = append(modes, fmt.Sprintf("Pile %d", index+1))
			}
			var contents []string
			for i, pile := range piles {
				for _, member := range pile {
					if p := g.FindPermanent(member.id); p != nil {
						contents = append(contents, fmt.Sprintf("%s (%s): pile %d", p.Name(), p.ID(), i+1))
					}
				}
			}
			reason := fmt.Sprintf("Choose a blocking pile for %s (%s). Piles: %s", creature.Name(), creature.ID(), strings.Join(contents, "; "))
			chosen := player.ChooseMode(modes, reason)
			if chosen <= 0 || chosen >= len(modes) {
				break
			}
			index := available[chosen-1]
			piles[index] = append(piles[index], visit)
			available = slices.Delete(available, chosen-1, chosen)
		}
	}
	for i := len(attackers) - 1; i > 0; i-- {
		j := g.RandIntn(i + 1)
		attackers[i], attackers[j] = attackers[j], attackers[i]
	}
	var proposed []BlockAssignment
	for i, pile := range piles {
		attacker := g.FindPermanent(attackers[i].id)
		if attacker == nil || attacker.incarnationID != attackers[i].incarnation {
			continue
		}
		for _, visit := range pile {
			p := g.FindPermanent(visit.id)
			if p != nil && p.incarnationID == visit.incarnation {
				proposed = append(proposed, BlockAssignment{BlockerID: p.ID(), AttackerID: attacker.ID()})
			}
		}
	}
	// Requirements cannot add creatures to piles that the defender left empty.
	assignments, _ := g.newBlockingPlan(defender, proposed).preferredLegal()
	return assignments
}
