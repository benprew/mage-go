package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONGraveyardBattlefieldInspection(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "absent", "returned", "copy", "clone"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				victim := mage.NewCreature("Victim", "", 1, 1)
				victim.SetOwner(a.PlayerID())
				a.AddToGraveyard(victim)
				obj := castRONSpell(t, g, a, repositoryCard(t, set, "Resurrection"), victim.ID())
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
						t.Fatal("native reanimation tree is unknown")
					}
				}
				if g.StackSize() != before || g.FindPermanent(victim.ID()) != nil {
					t.Fatal("inspection changed stack or entered a card")
				}
				g.ResolveTopOfStack()
				wantEntry := state == "normal" || state == "clone"
				entered := g.FindPermanent(victim.ID())
				if (entered != nil) != wantEntry {
					t.Fatal("wrong graveyard visit returned")
				}
				if entered != nil && (entered.ControllerID() != a.PlayerID() || entered.Tapped) {
					t.Fatal("wrong normal entry")
				}
				if state == "copy" {
					g.ResolveTopOfStack()
					if entered := g.FindPermanent(victim.ID()); entered == nil || entered.ControllerID() != a.PlayerID() {
						t.Fatal("copy consumed original target")
					}
				}
				if state == "clone" && len(a.Graveyard()) != 1 {
					t.Fatal("clone changed original graveyard")
				}
			})
		}
	}
}
