package combatsolver

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestTargetingAbilityCombatSolver(t *testing.T) {
	g, a, _ := makeGame()
	target := &abilityIdentityTarget{Target: mage.TargetCreature()}
	ability := mage.NewActivatedAbility(mage.GainLifeFor(mage.You, mage.Fixed(1)), mage.ManaPayment("{0}"), mage.WithTarget(target))
	target.expected = ability.AbilityID()
	source := g.PutOnBattlefield(mage.NewArtifact("Source", "", mage.WithAbility(ability)), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), a.PlayerID())
	units := applicablePumpUnits(g, a.PlayerID(), victim, []abilityPump{{power: 1, toughness: 1, cost: 1, source: source, target: target, ability: ability}})
	if len(units) != 1 {
		t.Fatal("combat solver lost targeting ability")
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
