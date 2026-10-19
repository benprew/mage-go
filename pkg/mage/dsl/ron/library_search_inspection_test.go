package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLibrarySearchInspection(t *testing.T) {
	for _, name := range []string{"Demonic Tutor", "Untamed Wilds"} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if name == "Untamed Wilds" {
			sets = []string{"LEG"}
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "empty", "copy", "clone"} {
				t.Run(name+"/"+set+"/"+state, func(t *testing.T) {
					a := &ronSearchPlayer{BasePlayer: mage.NewBasePlayer("A")}
					b := &ronSearchPlayer{BasePlayer: mage.NewBasePlayer("B")}
					g := mage.NewGame(a, b)
					g.SetActivePlayerIndex(0)
					g.SetTurn(1)
					g.SetStep(core.PrecombatMain)
					card := repositoryCard(t, set, name)
					obj := castRONSpell(t, g, a.BasePlayer, card)
					controller := a
					if state == "copy" {
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
						controller = b
					}
					found := repositoryCard(t, "LEB", "Forest")
					found.SetOwner(controller.PlayerID())
					if state != "empty" {
						controller.AddToLibrary(found)
					}
					if state == "clone" {
						g = g.Clone()
						obj = g.StackPeek()
					}
					for range 2 {
						if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
							t.Fatal("native search tree is unknown")
						}
					}
					if a.choices != 0 || b.choices != 0 || a.shuffles != 0 || b.shuffles != 0 {
						t.Fatal("inspection searched or shuffled")
					}
					g.ResolveTopOfStack()
					player := g.GetPlayer(controller.PlayerID())
					if len(player.Library()) != 0 {
						t.Fatal("search did not move the selected card")
					}
					if state != "empty" {
						if name == "Demonic Tutor" && len(player.Hand()) != 1 {
							t.Fatal("search hand destination")
						}
						if name == "Untamed Wilds" && g.FindPermanent(found.ID()) == nil {
							t.Fatal("search battlefield destination")
						}
					}
					if state == "clone" && len(a.Library()) != 1 {
						t.Fatal("clone changed original library")
					}
				})
			}
		}
	}
}
