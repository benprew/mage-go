package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeEvenDamage(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Fireball"}, {"LEB", "Fireball"}, {"2ED", "Fireball"}, {"3ED", "Fireball"}, {"CED", "Fireball"}, {"CEI", "Fireball"}, {"FEM", "Dwarven Catapult"}} {
		for _, state := range []string{"normal", "partial", "phase", "cycle", "return", "prevention", "clone", "copy", "empty", "zero", "zero-share", "all-illegal"} {
			t.Run(tc.set+"/"+tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				p := g.PutOnBattlefield(mage.NewCreature("Animated land", "", 1, 20), b.PlayerID())
				p.GrantBaseAttr(core.AttrIsLand)
				q := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 20), b.PlayerID())
				ids := []uuid.UUID{p.ID(), q.ID()}
				if tc.name == "Dwarven Catapult" {
					ids = []uuid.UUID{b.PlayerID()}
				}
				x := 5
				if state == "zero" {
					x = 0
				}
				if state == "zero-share" {
					x = 1
				}
				if state == "empty" {
					if tc.name == "Fireball" {
						ids = nil
					} else {
						g.BouncePermanentToHand(p)
						g.BouncePermanentToHand(q)
					}
				}
				a.ManaPool().Add(core.Red, 20)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, x); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				instructions, ok := mage.InspectEvenDamageInstructions(obj.Effects[0])
				if !ok || len(instructions) != 1 || instructions[0].Total == nil {
					t.Fatal("lost native division")
				}
				wantP, wantQ := x/2, x/2
				switch state {
				case "partial":
					p.GrantBaseAttr(core.Shroud)
					if tc.name == "Fireball" {
						wantP, wantQ = 0, x
					}
				case "phase":
					g.PhaseOut(p.ID())
					wantP, wantQ = 0, x
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.BouncePermanentToHand(p)
					b.RemoveFromHand(p.Card.ID())
					p = g.PutOnBattlefield(p.Card, b.PlayerID())
					if tc.name == "Fireball" {
						wantP, wantQ = 0, x
					}
				case "prevention":
					g.AddPreventionShield(p.ID(), 1)
					wantP = max(0, wantP-1)
				case "copy":
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				case "empty":
					wantP, wantQ = 0, 0
				case "all-illegal":
					if tc.name == "Fireball" {
						p.GrantBaseAttr(core.Shroud)
						q.GrantBaseAttr(core.Shroud)
						wantP, wantQ = 0, 0
					}
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("division tree is unknown")
					}
				}
				if p.Damage != 0 || q.Damage != 0 || b.Life() != 20 {
					t.Fatal("inspection changed state")
				}
				g.ResolveTopOfStack()
				if state != "empty" && (g.FindPermanentIncludingPhased(p.ID()).Damage != wantP || g.FindPermanent(q.ID()).Damage != wantQ) {
					t.Fatalf("damage=%d/%d want=%d/%d", g.FindPermanentIncludingPhased(p.ID()).Damage, g.FindPermanent(q.ID()).Damage, wantP, wantQ)
				}
			})
		}
	}
}
