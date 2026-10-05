package mage

import (
	"fmt"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaBonusLifecycle(t *testing.T) {
	for _, attached := range []bool{false, true} {
		for _, state := range []string{"live", "source-phased", "land-phased", "returned", "control", "clone", "removed"} {
			t.Run(fmt.Sprint(attached)+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				landCard := NewLand("Land", WithManaAbility(Blue))
				land := g.PutOnBattlefield(landCard, a.PlayerID())
				ability := NewManaFlareAbility(IsLand)
				if attached {
					ability = NewAttachedManaBonusAbility(Green)
				}
				source := g.PutOnBattlefield(NewEnchantment("Bonus", "", WithSubTypes("Aura"), WithAbility(ability)), b.PlayerID())
				if attached {
					g.Attach(source.ID(), land.ID())
				}
				want := 1
				switch state {
				case "source-phased":
					g.PhaseOut(source.ID())
					want = 0
				case "land-phased":
					g.PhaseOut(land.ID())
					want = 0
				case "returned":
					g.RemoveFromBattlefield(land)
					land = g.PutOnBattlefield(landCard, a.PlayerID())
					if attached {
						want = 0
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
				case "clone":
					clone := g.Clone()
					clone.PhaseOut(source.ID())
					if len(clone.mana.ManaBonuses(clone, land.ID())) != 0 {
						t.Fatal("clone retained phased bonus")
					}
				case "removed":
					g.AddContinuousEffect(RemoveAllAbilities(source.ID(), Indefinite))
					want = 0
				}
				if got := len(g.mana.ManaBonuses(g, land.ID())); got != want {
					t.Fatalf("bonuses = %d, want %d", got, want)
				}
			})
		}
	}
}
