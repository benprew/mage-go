package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryPTValueInspection(t *testing.T) {
	type spellCase struct{ set, name string }
	cases := make([]spellCase, 0, 7)
	cases = append(cases, spellCase{"LEG", "Great Defender"})
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		cases = append(cases, spellCase{set, "Howl from Beyond"})
	}
	for _, tc := range cases {
		for _, state := range []string{"normal", "zero", "opponent", "copy", "clone", "phased", "phase-in", "returned", "shroud"} {
			t.Run(tc.name+"/"+tc.set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "{2}{U}", 4, 5)
				card.SetOwner(a.PlayerID())
				controller := a.PlayerID()
				if state == "opponent" {
					controller = b.PlayerID()
				}
				p := g.PutOnBattlefield(card, controller)
				spell := repositoryCard(t, tc.set, tc.name)
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				x := 3
				if tc.name == "Great Defender" || state == "zero" {
					x = 0
				}
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, x); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("unpaid cast changed state")
				}
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.White, 1)
				a.ManaPool().Add(core.Colorless, x)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, x); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					if obj.XValue != x {
						t.Fatal("copy lost announced X")
					}
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "phase-in":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(card, controller)
				case "shroud":
					g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("native dynamic P/T tree is unknown")
					}
				}
				if g.FindPermanentIncludingPhased(p.ID()).CurrentPower(g) != 4 || g.FindPermanentIncludingPhased(p.ID()).CurrentToughness(g) != 5 {
					t.Fatal("query installed a layer")
				}
				if state == "phased" {
					g.PhaseIn([]uuid.UUID{p.ID()})
					g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
					g.PhaseOut(p.ID())
				} else {
					g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
				}
				g.ResolveTopOfStack()
				power, toughness := 5, 6
				if state != "phased" && state != "returned" && state != "shroud" {
					if tc.name == "Great Defender" {
						toughness += 3
					} else {
						power += x
					}
				}
				current := g.FindPermanentIncludingPhased(p.ID())
				if current.CurrentPower(g) != power || current.CurrentToughness(g) != toughness {
					t.Fatalf("got %d/%d want %d/%d", current.CurrentPower(g), current.CurrentToughness(g), power, toughness)
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
				g.PhaseIn(g.PhaseOut(p.ID()))
				if g.FindPermanent(p.ID()).CurrentPower(g) != power || g.FindPermanent(p.ID()).CurrentToughness(g) != toughness {
					t.Fatal("phase cycle changed installation")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(p.ID()).CurrentPower(g) != 5 || g.FindPermanent(p.ID()).CurrentToughness(g) != 6 {
					t.Fatal("layer survived cleanup")
				}
			})
		}
	}
}

func TestRONTemporaryPTValueCapture(t *testing.T) {
	for _, expression := range []string{`PowerOfPermanent(ActionPermanent)`, `Minus(ToughnessOfPermanent(ActionPermanent),Integer(3))`, `Minus(Minus(ToughnessOfPermanent(Ref_TargetPermanent),Integer(1)),Integer(2))`} {
		t.Run(expression, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(1,0), AdjustPTX(PlusX,Zero,`+expression+`), AdjustPTX(PlusX,Zero,PowerOfPermanent(ActionPermanent))], UntilEndOfTurn)])))`)
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 3, 6), b.PlayerID())
			obj := castRONSpell(t, g, a, factory(), p.ID())
			if _, known := g.InspectSpellDestruction(obj, nil); !known {
				t.Fatal("native arithmetic tree is unknown")
			}
			g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
			clone := g.Clone()
			g.ResolveTopOfStack()
			if g.FindPermanent(p.ID()).CurrentPower(g) != 13 || clone.FindPermanent(p.ID()).CurrentPower(clone) != 4 {
				t.Fatal("amounts were not captured together at resolution")
			}
			g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 2)
			if g.FindPermanent(p.ID()).CurrentPower(g) != 15 {
				t.Fatal("amounts followed later characteristics")
			}
			clone.ResolveTopOfStack()
			if clone.FindPermanent(p.ID()).CurrentPower(clone) != 13 {
				t.Fatal("clone lost operands")
			}
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			returned := g.PutOnBattlefield(mage.NewCreature("Replacement", "", 3, 6), b.PlayerID())
			if returned.CurrentPower(g) != 3 {
				t.Fatal("layer followed another visit")
			}
		})
	}
}

func TestRONTemporaryPTValueRejectsDeclarations(t *testing.T) {
	for _, modification := range []string{`AdjustPTX(PlusX,Zero)`, `AdjustPTX(PlusX(),Zero,Integer(1))`, `AdjustPTX(PlusX,Zero,ValueX)`, `AdjustPTX(PlusX,Zero,ToughnessOfPermanent(You))`, `AdjustPTX(PlusX,Zero,Minus(Integer(1)))`} {
		source := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[%s],UntilEndOfTurn)])))])`, modification)
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", modification)
		}
	}
	for _, tc := range []struct{ set, name string }{{"LEG", "Subdue"}} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 4, 5), a.PlayerID())
		obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name), p.ID())
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatalf("complete %s tree failed native inspection", tc.name)
		}
	}
}
