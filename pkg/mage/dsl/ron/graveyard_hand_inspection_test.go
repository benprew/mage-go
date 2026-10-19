package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONGraveyardHandInspection(t *testing.T) {
	for _, name := range []string{"Raise Dead", "Regrowth", "Reconstruction"} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if name == "Reconstruction" {
			sets = []string{"ATQ", "3ED"}
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "absent", "returned", "copy", "clone"} {
				t.Run(set+"/"+name+"/"+state, func(t *testing.T) {
					g, a, b := ronGame()
					victim := mage.NewCreature("Victim", "", 1, 1, mage.WithCardType(core.TypeArtifact))
					victim.SetOwner(a.PlayerID())
					a.AddToGraveyard(victim)
					obj := castRONSpell(t, g, a, repositoryCard(t, set, name), victim.ID())
					if state == "absent" || state == "returned" {
						card, ok := a.RemoveFromGraveyard(victim.ID())
						if !ok {
							t.Fatal("missing victim")
						}
						if state == "returned" {
							a.AddToGraveyard(card)
						} else {
							a.AddToHand(card)
						}
					}
					if state == "copy" {
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					}
					if state == "clone" {
						g = g.Clone()
						obj = g.StackPeek()
					}
					before := g.StackSize()
					for range 2 {
						if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
							t.Fatal("complete graveyard return tree is unknown")
						}
					}
					if g.StackSize() != before || g.GetPlayer(a.PlayerID()).Life() != 20 {
						t.Fatal("inspection changed state")
					}
					g.ResolveTopOfStack()
					wantHand := state != "returned" && state != "copy"
					found := false
					for _, card := range g.GetPlayer(a.PlayerID()).Hand() {
						found = found || card.ID() == victim.ID()
					}
					if found != wantHand {
						t.Fatal("wrong visit or controller returned a card")
					}
					if state == "copy" {
						g.ResolveTopOfStack()
						if len(g.GetPlayer(a.PlayerID()).Hand()) != 1 {
							t.Fatal("copy consumed the original's target")
						}
					}
					if state == "clone" && len(a.Hand()) != 0 {
						t.Fatal("clone changed original hand")
					}
					if g.FindPermanent(victim.ID()) != nil {
						t.Fatal("hand return entered battlefield")
					}
				})
			}
		}
	}
}
