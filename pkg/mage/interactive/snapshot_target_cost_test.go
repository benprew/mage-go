package interactive

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAvailableActionsReportXByTargetCount(t *testing.T) {
	for _, tc := range []struct {
		name      string
		red       int
		reduction int
		life      int
		want      []int
	}{
		{name: "reserve mana for each extra target", red: 6, want: []int{5, 4, 3}},
		{name: "omit target counts that cost too much", red: 2, want: []int{1, 0}},
		{name: "apply reductions to the total cost", red: 4, reduction: 2, want: []int{5, 4, 3}},
		{name: "use Channel and mana", red: 2, life: 4, want: []int{3, 3, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			a.ManaPool().Add(core.Red, tc.red)
			g.PutOnBattlefield(mage.NewCreature("Test creature", "{1}", 1, 1), b.PlayerID())
			if tc.reduction > 0 {
				g.AddSpellTypeCostReduction(core.TypeSorcery, tc.reduction)
			}
			if tc.life > 0 {
				a.SetLife(tc.life)
				g.SetChannelActive(a.PlayerID())
			}
			spell := mage.NewSorcery("Fireball", "{X}{R}",
				mage.NewTargetedSpell(mage.TargetDamageAnyNumberOfTargets(), mage.DealDamageDividedEvenly(mage.XValue())),
				mage.WithAdditionalCost(mage.ManaCostPerTargetBeyondFirst("{1}")),
			)
			a.AddToHand(spell)
			for _, action := range GetAvailableActions(g, a.PlayerID()) {
				if action.CardID == spell.ID() {
					if !slices.Equal(action.MaxXByTargetCount, tc.want) {
						t.Fatalf("X limits = %v, want %v", action.MaxXByTargetCount, tc.want)
					}
					return
				}
			}
			t.Fatal("missing Fireball action")
		})
	}
}
