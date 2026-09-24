package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticGrantedTiming(t *testing.T) {
	for _, timing := range []struct {
		name   string
		option GrantedActivationOption
		step   PhaseStep
	}{{"sorcery", OnlyAsSorcery(), PrecombatMain}, {"step", OnlyDuring(Upkeep), Upkeep}, {"own turn", OnlyOnYourTurn(), Upkeep}} {
		for _, scenario := range []string{"allowed", "wrong turn", "wrong step", "stack occupied"} {
			t.Run(timing.name+"/"+scenario, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), GainLifeFor(You, Fixed(2)), timing.option))))
				g.PutOnBattlefield(donor, a.PlayerID())
				recipient := NewArtifact("Recipient", "")
				g.PutOnBattlefield(recipient, b.PlayerID())
				g.ApplyContinuousEffects()
				g.turns.SetActivePlayerIndex(1)
				g.turns.SetStep(timing.step)
				if scenario == "wrong turn" {
					g.turns.SetActivePlayerIndex(0)
				}
				if scenario == "wrong step" {
					g.turns.SetStep(EndStep)
				}
				if scenario == "stack occupied" {
					filler := NewArtifact("Filler", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
					g.PutOnBattlefield(filler, b.PlayerID())
					if err := g.ActivateAbilityByIndex(b.PlayerID(), filler.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				b.ManaPool().Add(Colorless, 1)
				before := g.StackSize()
				allowed := (scenario != "wrong turn" || timing.name == "step") && (scenario != "wrong step" || timing.name == "own turn") && (scenario != "stack occupied" || timing.name != "sorcery")
				err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
				if !allowed {
					if err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != before {
						t.Fatal("invalid activation changed payment or stack", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if b.ManaPool().TotalMana() != 0 || g.StackSize() != before+1 {
					t.Fatal("incorrect payment or stack timing")
				}
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
				clone.turns.SetActivePlayerIndex(0)
				clone.turns.SetStep(EndStep)
				clone.ApplyContinuousEffects()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 22+before || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("resolution rechecked timing or used donor controller")
				}
				if b.Life() != 20 || g.StackSize() != before+1 {
					t.Fatal("clone changed original")
				}
			})
		}
	}
}

func TestStaticGrantedTimingValidation(t *testing.T) {
	for _, options := range [][]GrantedActivationOption{{OnlyAsSorcery(), OnlyDuring(Upkeep)}, {OnlyOnYourTurn(), OnlyAsSorcery()}, {OnlyDuring(PhaseStep(-1))}} {
		if err := validateFactory("Invalid grant", func() Card {
			return NewEnchantment("Invalid grant", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), Regenerate(This), options...))))
		}); err == nil {
			t.Fatal("invalid timing accepted")
		}
	}
}
