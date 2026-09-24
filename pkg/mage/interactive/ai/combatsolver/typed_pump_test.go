package combatsolver

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedHandPumpTargetLegality(t *testing.T) {
	for _, opponentOnly := range []bool{false, true} {
		name := "own creature allowed"
		if opponentOnly {
			name = "opponent only"
		}
		t.Run(name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			ours := mage.NewCreature("Ours", "", 2, 2)
			enemy := mage.NewCreature("Enemy", "", 2, 3)
			g.PutOnBattlefield(ours, a.PlayerID())
			g.PutOnBattlefield(enemy, b.PlayerID())
			target := dsl.TargetPermanent("creature", dsl.IsCreature)
			if opponentOnly {
				target = dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetOpponentControls())
			}
			card := dsl.NewInstant("Pump", "{G}", dsl.WithAbility(dsl.Spell(
				dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.Fixed(3), dsl.Fixed(3))), dsl.WithTargets(target))))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 1)
			pumps := gatherHandPumps(g, a.PlayerID())
			if len(pumps) != 1 {
				t.Fatalf("pump count = %d, want 1", len(pumps))
			}
			applyPumpToFlip(g, a.PlayerID(), g.FindPermanent(ours.ID()), g.FindPermanent(enemy.ID()), &pumps, nil, 1)
			want := 5
			if opponentOnly {
				want = 2
			}
			if got := g.FindPermanent(ours.ID()).CurrentPower(g); got != want {
				t.Fatalf("power = %d, want %d", got, want)
			}
		})
	}
}
