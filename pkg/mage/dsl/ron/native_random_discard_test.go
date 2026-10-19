package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeRandomDiscard(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		x, count  int
	}{
		{"LEA", "Mind Twist", 3, 3}, {"LEB", "Mind Twist", 3, 3},
		{"2ED", "Mind Twist", 3, 3}, {"3ED", "Mind Twist", 3, 3},
		{"CED", "Mind Twist", 3, 3}, {"CEI", "Mind Twist", 3, 3},
		{"FEM", "Hymn to Tourach", 0, 2}, {"LEB", "Mind Twist", 0, 0},
	} {
		for _, state := range []string{"normal", "clone", "copy"} {
			t.Run(tc.set+"/"+tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				randomRONHand(t, b, 4)
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.Colorless, tc.x)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, tc.x); err != nil {
					t.Fatal(err)
				}
				g.SetRandomResults([]int{2, 0, 1, 7})
				obj := g.StackPeek()
				original := obj.ID
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				baseline := g.Clone()
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete random discard tree is unknown")
					}
				}
				if len(g.GetPlayer(b.PlayerID()).Hand()) != 4 || g.StackSize() != baseline.StackSize() {
					t.Fatal("inspection changed hand or stack")
				}
				g.ResolveTopOfStack()
				baseline.ResolveTopOfStack()
				if state == "copy" {
					g.CounterStackObject(original)
					baseline.CounterStackObject(original)
				}
				got, want := g.GetPlayer(b.PlayerID()).Graveyard(), baseline.GetPlayer(b.PlayerID()).Graveyard()
				if len(got) != tc.count || len(want) != tc.count {
					t.Fatal("wrong discard amount")
				}
				for i := range got {
					if got[i].ID() != want[i].ID() {
						t.Fatal("inspection consumed random results")
					}
				}
				if g.RandIntn(10) != baseline.RandIntn(10) {
					t.Fatal("inspection changed random queue")
				}
				if len(g.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
					t.Fatal("spell did not leave stack")
				}
			})
		}
	}
}

func TestRONNativeRandomDiscardLowering(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEA", "Mind Twist"}, {"LEB", "Mind Twist"}, {"2ED", "Mind Twist"},
		{"3ED", "Mind Twist"}, {"CED", "Mind Twist"}, {"CEI", "Mind Twist"},
		{"FEM", "Hymn to Tourach"},
	} {
		record := repositoryRecord(t, tc.set, tc.name, "native-discard.ron")
		node := contextualQueryNode(t, record, "DiscardNumberCardsAtRandom")
		builds, err := compileRandomDiscard(node, scope{x: true, resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			instructions, ok := mage.InspectRandomDiscardInstructions(builds[0](instanceBindings{}))
			if !ok || len(instructions) != 1 || instructions[0].Subject != mage.You {
				t.Fatalf("%s lost its native operands", tc.name)
			}
		}
	}
}
