package interactive

import (
	"slices"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAvailableActionsManaValueTargets(t *testing.T) {
	for _, includeLabels := range []bool{false, true} {
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		a.ManaPool().Add(core.Red, 4)
		zero := g.PutOnBattlefield(mage.NewArtifact("Zero", "{0}"), b.PlayerID())
		three := g.PutOnBattlefield(mage.NewArtifact("Three", "{3}"), b.PlayerID())
		g.PutOnBattlefield(mage.NewArtifact("Too expensive", "{9}"), b.PlayerID())
		spell := mage.NewSorcery("Removal", "{X}{R}", mage.NewTargetedSpell(mage.TargetArtifactWithManaValueX(), mage.DestroyTarget()))
		a.AddToHand(spell)
		g.SetXValue(9)

		found := false
		for _, action := range getAvailableActions(g, a.PlayerID(), includeLabels) {
			if action.CardID != spell.ID() {
				continue
			}
			found = true
			if !slices.Equal(action.ValidTargets, []uuid.UUID{zero.ID(), three.ID()}) {
				t.Fatalf("labels %t: targets = %v, want affordable artifacts", includeLabels, action.ValidTargets)
			}
			if !action.NeedsX || action.MaxXValue != 3 {
				t.Fatalf("wrong X options: %+v", action)
			}
		}
		if !found {
			t.Fatalf("labels %t: missing affordable spell", includeLabels)
		}
		if g.XValue() != 9 || a.ManaPool().TotalMana() != 4 {
			t.Fatal("action query changed announcement state")
		}
	}
}
