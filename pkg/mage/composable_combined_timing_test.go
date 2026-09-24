package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombinedActivationTiming(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, granted := range []bool{false, true} {
			for _, mana := range []bool{false, true} {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				first, second := OnlyDuring(Upkeep), OnlyOnYourTurn()
				if reverse {
					first, second = second, first
				}
				var body = GainLifeFor(You, Fixed(1))
				if mana {
					body = AddMana(Blue, 1)
				}
				source := NewArtifact("Source", "")
				if granted {
					donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(TapCost(This)), body, first, second))))
					g.PutOnBattlefield(donor, a.PlayerID())
				} else {
					source = NewArtifact("Source", "", WithAbility(Activate(Costs(TapCost(This)), body, first, second)))
				}
				g.PutOnBattlefield(source, b.PlayerID())
				g.ApplyContinuousEffects()
				for _, state := range []struct {
					active  int
					step    PhaseStep
					allowed bool
				}{{0, Upkeep, false}, {1, PrecombatMain, false}, {1, Upkeep, true}} {
					g.SetActivePlayerIndex(state.active)
					g.SetStep(state.step)
					_, planned := manaSourceAbilityForPlanning(g.FindPermanent(source.ID()).RuntimeAbilities[0], source.ID(), 0, g)
					if mana && planned != state.allowed {
						t.Fatal("mana planning ignored combined timing")
					}
					err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil)
					if !state.allowed {
						if err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
							t.Fatal("invalid activation paid costs")
						}
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					g.SetActivePlayerIndex(0)
					g.SetStep(EndStep)
					g.ResolveStack()
					if mana {
						if b.ManaPool().TotalMana() != 1 {
							t.Fatal("mana ability failed")
						}
					} else if b.Life() != 21 {
						t.Fatal("timing rechecked at resolution")
					}
				}
			}
		}
	}
}
