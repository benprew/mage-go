package ron

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeCollectionColor(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{
		{"Dwarven Song", core.Red}, {"Heaven's Gate", core.White}, {"Sea Kings' Blessing", core.Blue}, {"Sylvan Paradise", core.Green}, {"Touch of Darkness", core.Black},
	} {
		for _, state := range []string{"normal", "shroud", "phase", "cycle", "return", "all illegal", "clone", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "{W}{U}", 2, 3)
				p := g.PutOnBattlefield(card, b.PlayerID())
				q := g.PutOnBattlefield(mage.NewCreature("Other", "{W}{U}", 2, 3), a.PlayerID())
				obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", tc.name), p.ID(), q.ID())
				switch state {
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "phase":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, b.PlayerID())
				case "all illegal":
					p.GrantBaseAttr(core.Shroud)
					q.GrantBaseAttr(core.Shroud)
				case "clone":
					g = g.Clone()
					obj = g.StackPeek()
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete color spell tree is unknown")
					}
				}
				if !slices.Equal(g.FindPermanentIncludingPhased(p.ID()).Colors(), []core.Color{core.White, core.Blue}) {
					t.Fatal("inspection executed layer")
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{p.ID()})
				want := []core.Color{tc.color}
				if state == "shroud" || state == "phase" || state == "return" || state == "all illegal" {
					want = []core.Color{core.White, core.Blue}
				}
				if !slices.Equal(g.FindPermanent(p.ID()).Colors(), want) {
					t.Fatal("lost visit or legality")
				}
				if state != "all illegal" && !slices.Equal(g.FindPermanent(q.ID()).Colors(), []core.Color{tc.color}) {
					t.Fatal("lost legal member")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if !slices.Equal(g.FindPermanent(q.ID()).Colors(), []core.Color{core.White, core.Blue}) {
					t.Fatal("cleanup retained colors")
				}
			})
		}
	}
}

func TestRONNativeCollectionColorLowering(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Dwarven Song", "colors.ron")
	node := contextualQueryNode(t, record, "CreateEachPermanentLayerEffectUntil")
	plan := &permanentTargetsPlan{name: "Ref_TargetPermanents"}
	build, err := compilePluralColor(node, scope{resolving: true, permanentTargets: plan})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		binding := mage.DeclarePermanentTargets("colors", 1, 2, mage.IsCreature)
		effect := build(instanceBindings{permanentSets: map[*permanentTargetsPlan]*mage.PermanentTargetsBinding{plan: binding}})
		got, ok := mage.InspectTemporaryColorInstruction(effect)
		if !ok || got.Subjects != binding || !slices.Equal(got.Colors, []core.Color{core.Red}) {
			t.Fatal("lost native operands")
		}
	}
}
