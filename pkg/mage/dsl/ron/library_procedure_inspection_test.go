package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLibraryProcedureInspection(t *testing.T) {
	for _, name := range []string{"Natural Selection", "Visions"} {
		sets := []string{"LEA", "LEB", "2ED", "CED", "CEI"}
		count := 3
		if name == "Visions" {
			sets = []string{"LEG"}
			count = 5
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "empty", "short", "accept", "copy", "clone"} {
				t.Run(name+"/"+set+"/"+state, func(t *testing.T) {
					a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A"), accept: state == "accept"}
					b := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("B")}
					g := mage.NewGame(a, b)
					g.SetStep(core.PrecombatMain)
					n := 7
					if state == "empty" {
						n = 0
					}
					if state == "short" {
						n = 2
					}
					for range n {
						b.AddToLibrary(mage.NewLand("hidden"))
					}
					obj := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, set, name), b.PlayerID())
					viewer := a.PlayerID()
					if state == "copy" {
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
						viewer = b.PlayerID()
					}
					if state == "clone" {
						g = g.Clone()
						obj = g.StackPeek()
					}
					first := g.GetPlayer(b.PlayerID()).Library()
					for range 2 {
						if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
							t.Fatal("native library procedure tree is unknown")
						}
					}
					if len(g.LibraryLooks(viewer)) != 0 || a.orders+b.orders+a.offers+b.offers+a.shuffles+b.shuffles != 0 {
						t.Fatal("query executed library procedure")
					}
					g.ResolveTopOfStack()
					if len(g.LibraryLooks(viewer)) != min(n, count) || len(g.PublicReveals()) != 0 {
						t.Fatal("wrong private look")
					}
					remaining := g.GetPlayer(b.PlayerID()).Library()
					if len(remaining) != n {
						t.Fatal("procedure moved library cards")
					}
					if state != "accept" {
						for i := range remaining {
							index := i
							if name == "Natural Selection" && state != "clone" && i < min(n, count) {
								index = min(n, count) - 1 - i
							}
							if remaining[i].ID() != first[index].ID() {
								t.Fatal("wrong library order")
							}
						}
					}
					if state == "clone" && (a.offers != 0 || len(g.LibraryLooks(a.PlayerID())) != min(n, count)) {
						t.Fatal("clone changed original choices")
					}
					if state != "clone" {
						chooser := a
						if state == "copy" {
							chooser = b
						}
						if chooser.offers != 1 || b.shuffles != map[bool]int{false: 0, true: 1}[state == "accept"] {
							t.Fatal("wrong optional shuffle")
						}
					}
				})
			}
		}
	}
}
