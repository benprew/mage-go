package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeNextUntap(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEG", "Telekinesis"}, {"FEM", "Spore Cloud"}} {
		for _, state := range []string{"normal", "clone", "copy", "return", "phased", "cycle", "shroud", "control"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "", 2, 20)
				p := g.PutOnBattlefield(card, b.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{p.ID()})
				var targets []uuid.UUID
				if tc.name == "Telekinesis" {
					targets = []uuid.UUID{p.ID()}
				}
				obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name), targets...)
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
					g.CounterStackObject(g.GetStack().Objects()[0].ID)
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				switch state {
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(card, b.PlayerID())
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "shroud":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete next-untap tree is unknown")
					}
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
				g.UntapPermanent(g.FindPermanent(p.ID()))
				g.TapPermanent(g.FindPermanent(p.ID()))
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Untap)
				if g.FindPermanent(p.ID()).Tapped {
					t.Fatal("inspection installed an untap lock")
				}
				if state == "phased" {
					g.PhaseOut(p.ID())
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{p.ID()})
				affected := state != "return" && state != "phased" && (state != "shroud" || tc.name == "Spore Cloud") && (state != "cycle" || tc.name != "Spore Cloud")
				if state == "control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					g.SetActivePlayerIndex(0)
				}
				count := 1
				if tc.name == "Telekinesis" {
					count = 2
				}
				for step := 0; step <= count; step++ {
					g.TapPermanent(g.FindPermanent(p.ID()))
					g.RunStepWithPriority(core.Untap)
					if g.FindPermanent(p.ID()).Tapped != (affected && step < count) {
						t.Fatalf("wrong untap restriction at step %d", step)
					}
				}
			})
		}
	}
}

func TestRONNativeNextUntapLowering(t *testing.T) {
	for _, tc := range []struct {
		set, name, action string
		count             int
	}{
		{"LEG", "Telekinesis", "PermanentDoesntUntapDuringControllersNextNumberUntaps", 2},
		{"FEM", "Spore Cloud", "EachPermanentDoesntUntapDuringControllersNextUntap", 1},
		{"DRK", "Barl's Cage", "PermanentDoesntUntapDuringControllersNextUntap", 1},
		{"FEM", "Elvish Hunter", "PermanentDoesntUntapDuringControllersNextUntap", 1},
		{"FEM", "Deep Spawn", "PermanentDoesntUntapDuringControllersNextUntap", 1},
		{"FEM", "Homarid Warrior", "PermanentDoesntUntapDuringControllersNextUntap", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := repositoryRecord(t, tc.set, tc.name, "lock.ron")
			node := contextualQueryNode(t, record, tc.action)
			plan := &permanentTargetPlan{name: "Ref_TargetPermanent"}
			build, err := compileNextUntaps(node, scope{resolving: true, permanent: true, permanentTarget: plan})
			if err != nil {
				t.Fatal(err)
			}
			var previous mage.PermanentRef
			for range 2 {
				target := mage.DeclarePermanentTarget("subject", mage.IsCreature)
				effect := build(instanceBindings{permanents: map[*permanentTargetPlan]*mage.PermanentTargetBinding{plan: target}})
				declaration, ok := mage.InspectNextUntapInstruction(effect)
				if !ok || declaration.Count != tc.count {
					t.Fatal("lost native lock declaration")
				}
				if node.Args[0].Name == "Ref_TargetPermanent" {
					if declaration.Subject != target || declaration.Subject == previous {
						t.Fatal("shared target instance")
					}
					previous = declaration.Subject
				}
			}
		})
	}
}
