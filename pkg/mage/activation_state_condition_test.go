package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationStateCondition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	c := NewArtifact("Condition", "", WithAbility(NewActivated(GenericCost(1), GainLifeFor(You, Fixed(1)), WithActivationStateCondition(PermanentMatchesFilter(This, CounterCountAtLeast(core.Charge, 2))))))
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("false condition spent resources")
	}
	g.MutablePermanent(p.ID()).Counters[core.Charge] = 2
	clone := g.Clone()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(p.ID()).Counters[core.Charge] = 0
	g.ResolveStack()
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("condition repeated at resolution or clone shared state")
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 21 {
		t.Fatal("clone lost condition")
	}
	for _, condition := range []StateCondition{nil, PermanentMatchesFilter(nil, IsCreature)} {
		invalid := NewArtifact("Invalid", "", WithAbility(NewActivated(GenericCost(0), GainLifeFor(You, Fixed(1)), WithActivationStateCondition(condition))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("invalid condition accepted")
		}
	}
}
