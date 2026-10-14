package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestTargetingAbilityCombat(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	target := &abilityIdentityTarget{Target: mage.TargetCreature()}
	ability := mage.NewActivatedAbility(mage.Regenerate(mage.This), mage.ManaPayment("{0}"), mage.WithTarget(target))
	target.expected = ability.AbilityID()
	source := g.PutOnBattlefield(mage.NewArtifact("Source", "", mage.WithAbility(ability)), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), a.PlayerID())
	if chosen, ok := pumpAbilityTargets(ability, source, victim, a.PlayerID(), g); !ok || len(chosen) != 1 || chosen[0] != victim.ID() {
		t.Fatal("pump query lost targeting ability")
	}
	strategy := &Strategy{}
	action := strategy.regenerationAbilityAction(a.PlayerID(), g, g.GetActivatableAbilities(a.PlayerID()), victim)
	if action == nil || len(action.Targets) != 1 || action.Targets[0] != victim.ID() {
		t.Fatal("regeneration query lost targeting ability")
	}
}

type abilityIdentityTarget struct {
	mage.Target
	expected uuid.UUID
}

func (t *abilityIdentityTarget) Possible(controller uuid.UUID, card mage.Card, g *mage.Game) []uuid.UUID {
	id, declared, known := mage.TargetSourceAbility(card)
	if !known || id != t.expected || len(declared) != 1 || declared[0] != t {
		return nil
	}
	return t.Target.Possible(controller, card, g)
}
