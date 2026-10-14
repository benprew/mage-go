package search

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

func TestTargetingAbilitySearch(t *testing.T) {
	g, a, _ := makeGame()
	target := &abilityIdentityTarget{Target: mage.TargetCreature()}
	ability := mage.NewActivatedAbility(mage.GainLifeFor(mage.You, mage.Fixed(1)), mage.ManaPayment("{0}"), mage.WithTarget(target))
	target.expected = ability.AbilityID()
	source := g.PutOnBattlefield(mage.NewArtifact("Source", "", mage.WithAbility(ability)), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), a.PlayerID())
	combinations := topAbilityTargetCombinations(g, a.PlayerID(), source, ability, eval.TargetGeneric, 0)
	if len(combinations) != 1 || len(combinations[0]) != 1 || combinations[0][0] != victim.ID() {
		t.Fatal("search lost targeting ability identity")
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
