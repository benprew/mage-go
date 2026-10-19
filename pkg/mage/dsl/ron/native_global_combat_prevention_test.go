package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeGlobalCombatPrevention(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEA", "Fog"}, {"LEB", "Fog"}, {"2ED", "Fog"},
		{"3ED", "Fog"}, {"CED", "Fog"}, {"CEI", "Fog"},
		{"LEG", "Darkness"}, {"LEG", "Holy Day"},
	} {
		for _, state := range []string{"normal", "clone", "copy"} {
			t.Run(tc.set+"/"+tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name))
				original := obj.ID
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete combat prevention tree is unknown")
					}
				}
				if g.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(obj.SourceID, b.PlayerID(), 2, true), g) == nil {
					t.Fatal("inspection installed prevention")
				}
				g.ResolveTopOfStack()
				if state == "copy" {
					g.CounterStackObject(original)
				}
				assertRONGlobalCombatPrevention(t, g, a.PlayerID(), b.PlayerID())
			})
		}
	}
}

func TestRONNativeGlobalCombatPreventionLowering(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEB", "Fog"}, {"LEG", "Darkness"}, {"LEG", "Holy Day"},
		{"FEM", "Spore Cloud"}, {"FEM", "Spore Flower"}, {"LEG", "Angus Mackenzie"},
	} {
		record := repositoryRecord(t, tc.set, tc.name, tc.set+".ron")
		node := contextualQueryNode(t, record, "CreatePreventDamageUntil")
		build, err := compileCombatPrevention(node, scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if _, ok := mage.InspectGlobalCombatPreventionInstruction(build(instanceBindings{})); !ok {
				t.Fatalf("%s lost its native declaration", tc.name)
			}
		}
	}
}

func TestRONNativeGlobalCombatPreventionMixedBoundary(t *testing.T) {
	g, a, _ := ronGame()
	obj := castRONSpell(t, g, a, repositoryCard(t, "FEM", "Spore Cloud"))
	if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
		t.Fatal("complete untap lock tree is unknown")
	}
	g.ResolveStack()
	if g.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(obj.SourceID, a.PlayerID(), 2, true), g) != nil {
		t.Fatal("mixed spell lost combat prevention")
	}
	g.RunStepWithPriority(core.Cleanup)
}
