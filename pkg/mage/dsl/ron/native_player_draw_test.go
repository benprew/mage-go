package ron

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONNativePlayerDraw(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Wheel of Fortune"}, {"LEB", "Wheel of Fortune"}, {"2ED", "Wheel of Fortune"}, {"3ED", "Wheel of Fortune"}, {"CED", "Wheel of Fortune"}, {"CEI", "Wheel of Fortune"}, {"LEA", "Timetwister"}, {"LEB", "Timetwister"}, {"2ED", "Timetwister"}, {"CED", "Timetwister"}, {"CEI", "Timetwister"}} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			record := repositoryRecord(t, tc.set, tc.name, "draw.ron")
			node := contextualQueryNode(t, record, "EachPlayerAction")
			for _, action := range contextualQueryNode(t, record, "ActionList").Args[0].Args {
				if action.Name == "EachPlayerAction" && len(action.Args) == 2 && action.Args[1].Name == "DrawNumberCards" {
					node = action
				}
			}
			builds, err := compileEachPlayerAction(node, scope{resolving: true})
			if err != nil {
				t.Fatal(err)
			}
			effect := builds[0](instanceBindings{})
			declaration, ok := mage.InspectPlayerDrawInstruction(effect)
			if !ok || declaration.Subjects != mage.EachPlayer {
				t.Fatal("lost native draw declaration")
			}
			g, a, b := ronGame()
			fillRONLibrary(t, a, 10)
			fillRONLibrary(t, b, 10)
			if err := mage.ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 7 || len(b.Hand()) != 7 {
				t.Fatal("real RON draw did not execute")
			}
			obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name))
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || known != (tc.name == "Wheel of Fortune") {
				t.Fatal("complete native Wheel must be known; shuffle remains unknown")
			}
		})
	}
}

func TestRONNativePlayerDrawQuery(t *testing.T) {
	for _, amount := range []int{0, 2} {
		for _, state := range []string{"only", "prefix", "suffix", "clone", "copy"} {
			t.Run(fmt.Sprintf("%d/%s", amount, state), func(t *testing.T) {
				action := fmt.Sprintf("EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(%d)))", amount)
				body := action
				if state == "prefix" {
					body += ",DestroyEachPermanent(IsCardtype(Land))"
				}
				if state == "suffix" {
					body = "DestroyEachPermanent(IsCardtype(Land))," + body
				}
				card := permanentScopeFactory(t, "Sorcery", "SpellActions(ActionList(["+body+"]))")()
				g, a, b := ronGame()
				fillRONLibrary(t, a, 10)
				fillRONLibrary(t, b, 10)
				obj := castRONSpell(t, g, a, card)
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				}
				certain, known := g.InspectSpellDestruction(obj, nil)
				if certain || known != (state != "prefix" || amount == 0) {
					t.Fatalf("query=%v/%v", certain, known)
				}
				if len(g.GetPlayer(a.PlayerID()).Hand()) != 0 || len(g.GetPlayer(b.PlayerID()).Hand()) != 0 {
					t.Fatal("query drew cards")
				}
				g.ResolveTopOfStack()
				if len(g.GetPlayer(a.PlayerID()).Hand()) != amount || len(g.GetPlayer(b.PlayerID()).Hand()) != amount {
					t.Fatal("draw did not resolve")
				}
			})
		}
	}
}
