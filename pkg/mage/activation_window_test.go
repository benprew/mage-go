package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationWindowTypedContracts(t *testing.T) {
	for _, during := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetTurn(1)
		g.SetStep(core.PrecombatMain)
		condition := AllConditions(NotCondition(IsYourTurn()), BeforeCombatStep(core.DeclareAttackers, during))
		if during {
			condition = AllConditions(condition, IsDuringSteps(core.BeginCombat, core.DeclareAttackers, core.DeclareBlockers, core.FirstStrikeDamage, core.CombatDamage, core.EndCombat))
		}
		card := NewArtifact("Window", "", WithAbility(NewActivated(GenericCost(1), GainLifeFor(You, Fixed(1)), WithActivationStateCondition(condition))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		card.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
			t.Fatal("own-turn rejection spent resources")
		}
		g.SetActivePlayerIndex(1)
		g.SetStep(core.BeginCombat)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		g.SetStep(core.DeclareAttackers)
		g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
		g.ResolveStack()
		if a.Life() != 21 {
			t.Fatal("departure or closed window stopped pending body")
		}
		clone.ResolveStack()
		clone.SetStep(core.EndCombat)
		clone.SetStep(core.PostcombatMain)
		clone.SetStep(core.BeginCombat)
		clone.GetPlayer(a.PlayerID()).ManaPool().Add(core.Colorless, 1)
		err := clone.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil)
		if (err == nil) != during {
			t.Fatalf("extra combat: %v, during=%v", err, during)
		}
		if during {
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 21 {
				t.Fatal("clone results shared")
			}
		}
	}
}
