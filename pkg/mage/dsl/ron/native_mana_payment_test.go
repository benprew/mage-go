package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeManaPayment(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEG", "Force Spike"}, {"LEA", "Power Sink"}, {"LEB", "Power Sink"},
		{"2ED", "Power Sink"}, {"3ED", "Power Sink"}, {"CED", "Power Sink"}, {"CEI", "Power Sink"},
	} {
		for _, state := range []string{"paid", "declined", "clone", "copy", "illegal", "zero"} {
			t.Run(tc.set+"/"+tc.name+"/"+state, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state == "paid" || state == "zero"}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				victim := castRONSpell(t, g, b.BasePlayer, mage.NewInstant("Victim", ""))
				b.ManaPool().Clear()
				card := repositoryCard(t, tc.set, tc.name)
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Colorless, 2)
				x := 2
				if state == "zero" {
					x = 0
				}
				if tc.name == "Force Spike" {
					x = 0
				}
				obj := castNativePaymentRON(t, g, a.BasePlayer, card, victim.ID, x)
				b.ManaPool().Add(core.Red, 3)
				land := g.PutOnBattlefield(mage.NewLand("Mana land", mage.WithManaAbility(core.Green)), b.PlayerID())
				empty := g.PutOnBattlefield(mage.NewLand("No mana"), b.PlayerID())
				if state == "illegal" {
					g.GetStack().RemoveByID(victim.ID)
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				}
				for range 2 {
					certain, known := g.InspectSpellDestruction(obj, nil)
					if certain || !known {
						t.Fatal("complete payment tree is unknown")
					}
				}
				if b.offers != 0 || a.offers != 0 || g.FindPermanent(land.ID()).Tapped || g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 3 {
					t.Fatal("inspection changed payment state")
				}
				g.ResolveTopOfStack()
				paid := state == "paid" || state == "zero"
				if state == "illegal" {
					if b.offers != 0 {
						t.Fatal("illegal target offered payment")
					}
					return
				}
				if (g.GetStack().FindByID(victim.ID) != nil) != paid {
					t.Fatal("wrong payment branch")
				}
				if tc.name == "Power Sink" {
					if g.FindPermanent(land.ID()).Tapped == paid || g.FindPermanent(empty.ID()).Tapped {
						t.Fatal("wrong tap continuation")
					}
					want := 0
					if paid {
						want = 3 - x
					}
					if g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != want {
						t.Fatal("wrong mana continuation")
					}
				}
			})
		}
	}
}

func castNativePaymentRON(t *testing.T, g *mage.Game, p *mage.BasePlayer, c mage.Card, target uuid.UUID, x int) *mage.StackObject {
	t.Helper()
	c.SetOwner(p.PlayerID())
	p.AddToHand(c)
	if err := g.CastSpellByID(p.PlayerID(), c.ID(), []uuid.UUID{target}, x); err != nil {
		t.Fatal(err)
	}
	return g.StackPeek()
}

func TestRONNativeManaPaymentLowering(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEG", "Force Spike"}, {"LEA", "Power Sink"}, {"LEB", "Power Sink"}, {"2ED", "Power Sink"}, {"3ED", "Power Sink"}, {"CED", "Power Sink"}, {"CEI", "Power Sink"}} {
		record := repositoryRecord(t, tc.set, tc.name, "native-payment.ron")
		payment := contextualQueryNode(t, record, "PlayerMayCost")
		continuation := contextualQueryNode(t, record, "Unless")
		plan := &spellTargetPlan{name: "Ref_TargetSpell"}
		build, err := compileResolutionPayment(payment, continuation, scope{spellTarget: plan, x: true, resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			target := mage.DeclareSpellTarget("spell")
			effect := build(instanceBindings{spells: map[*spellTargetPlan]*mage.SpellTargetBinding{plan: target}})
			instruction, ok := mage.InspectManaPaymentInstruction(effect)
			if !ok || instruction.Paid != nil || instruction.Unpaid == nil || instruction.Cost.HasX != (tc.name == "Power Sink") {
				t.Fatal("lost payment declaration")
			}
			if tc.name == "Force Spike" && instruction.Cost.Generic != 1 {
				t.Fatal("wrong fixed cost")
			}
		}
	}
}
