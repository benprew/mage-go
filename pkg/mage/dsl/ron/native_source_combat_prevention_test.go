package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeSourceCombatPreventionSubdue(t *testing.T) {
	for _, state := range []string{"normal", "clone", "copy", "returned", "phased", "phase cycle", "shroud"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := mage.NewCreature("Subject", "{4}{G}{G}", 3, 8)
			p := g.PutOnBattlefield(card, b.PlayerID())
			obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Subdue"), p.ID())
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			switch state {
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			}
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
					t.Fatal("complete Subdue tree is unknown or proves destruction")
				}
			}
			if g.FindPermanentIncludingPhased(p.ID()).CurrentToughness(g) != 8 {
				t.Fatal("inspection installed layer")
			}
			if state != "phased" && sourceCombatPrevented(g, p.ID(), a.PlayerID()) {
				t.Fatal("inspection installed prevention")
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{p.ID()})
			affected := state != "returned" && state != "phased" && state != "shroud"
			want := 8
			if affected {
				want = 14
				if state == "copy" {
					want = 20
				}
			}
			if g.FindPermanent(p.ID()).CurrentToughness(g) != want || sourceCombatPrevented(g, p.ID(), a.PlayerID()) != affected {
				t.Fatal("incorrect complete spell execution")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(p.ID()).CurrentToughness(g) != 8 || sourceCombatPrevented(g, p.ID(), a.PlayerID()) {
				t.Fatal("effects survived cleanup")
			}
		})
	}
}
