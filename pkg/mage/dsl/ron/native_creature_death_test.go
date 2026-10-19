package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeCreatureDeath(t *testing.T) {
	for range 2 {
		g, a, _ := ronGame()
		wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), a.PlayerID())
		g.SetStep(core.EndCombat)
		g.SetStep(core.PostcombatMain)
		obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Glyph of Reincarnation"), wall.ID())
		for range 2 {
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
				t.Fatal("complete native death-result tree is unknown")
			}
		}
		g.ResolveStack()
		if g.FindPermanent(wall.ID()) == nil || g.StackSize() != 0 {
			t.Fatal("inspection changed resolution")
		}
	}
}
