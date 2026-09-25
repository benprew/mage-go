package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestTakeHeartResolution(t *testing.T) {
	for _, scenario := range []struct {
		name string
		gain int
	}{
		{"two attackers", 2},
		{"removed attacker", 1},
		{"opponent attackers", 0},
		{"returned target", 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			targetCard := mage.NewCreature("Heart target", "", 2, 2)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			attackerController, defender := a.PlayerID(), b.PlayerID()
			if scenario.name == "opponent attackers" {
				attackerController, defender = b.PlayerID(), a.PlayerID()
			}
			first := g.PutOnBattlefield(mage.NewCreature("First attacker", "", 2, 2), attackerController)
			second := g.PutOnBattlefield(mage.NewCreature("Second attacker", "", 2, 2), attackerController)
			g.GetCombat().AddAttacker(first.ID(), defender)
			g.GetCombat().AddAttacker(second.ID(), defender)
			g.PutOnBattlefield(mage.NewCreature("Idle creature", "", 2, 2), a.PlayerID())
			spell, err := mage.CreateCard("Take Heart")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 1)
			g.SetStep(core.DeclareAttackers)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if scenario.name == "removed attacker" {
				g.RemoveFromCombat(first.ID())
			}
			if scenario.name == "returned target" {
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(targetCard, b.PlayerID())
			}
			g.ResolveStack()
			g.AssertLife(gametest.PlayerA, 20+scenario.gain)
			g.AssertLife(gametest.PlayerB, 20)
			wantPT := 4
			if scenario.name == "returned target" {
				wantPT = 2
			}
			g.AssertPowerToughness(gametest.PlayerB, "Heart target", wantPT, wantPT)
		})
	}
}
