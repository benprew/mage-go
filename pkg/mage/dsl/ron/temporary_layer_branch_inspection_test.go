package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryLayerBranchInspection(t *testing.T) {
	for _, name := range []string{"Blood Lust", "Rapid Fire"} {
		for _, state := range []string{"normal", "clone", "copy", "phased", "phase-in", "returned", "shroud"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "", 3, 5)
				p := g.PutOnBattlefield(card, b.PlayerID())
				obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", name), p.ID())
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "phase-in":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(card, b.PlayerID())
				case "shroud":
					g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete conditional layer tree is unknown")
					}
				}
				if g.FindPermanentIncludingPhased(p.ID()).CurrentPower(g) != 3 || g.FindPermanentIncludingPhased(p.ID()).HasKeyword(core.FirstStrike) {
					t.Fatal("query changed subject")
				}
				g.ResolveTopOfStack()
				present := g.FindPermanentIncludingPhased(p.ID())
				applies := state != "phased" && state != "returned" && state != "shroud"
				if name == "Blood Lust" {
					want := 3
					if applies {
						want = 7
					}
					if present.CurrentPower(g) != want {
						t.Fatal("wrong P/T branch")
					}
				} else if present.HasKeyword(core.FirstStrike) != applies || mage.HasRampage.Match(present, g) != applies {
					t.Fatal("wrong ability branch")
				}
				g.PhaseIn(g.PhaseOut(p.ID()))
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				present = g.FindPermanentIncludingPhased(p.ID())
				if present.CurrentPower(g) != 3 || present.CurrentToughness(g) != 5 || present.HasKeyword(core.FirstStrike) || mage.HasRampage.Match(present, g) {
					t.Fatal("branch survived cleanup")
				}
			})
		}
	}
}

func TestRONTemporaryLayerBranchBloodLustCapture(t *testing.T) {
	for _, toughness := range []int{1, 2, 4, 5, 6, 9} {
		t.Run(fmt.Sprint(toughness), func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 3, 3), b.PlayerID())
			castRONSpell(t, g, a, repositoryCard(t, "LEG", "Blood Lust"), p.ID())
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(0), mage.Fixed(toughness-3))), p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			want := 1
			if toughness >= 5 {
				want = toughness - 4
			}
			p = g.FindPermanent(p.ID())
			if p.CurrentPower(g) != 7 || p.CurrentToughness(g) != want {
				t.Fatalf("got %d/%d", p.CurrentPower(g), p.CurrentToughness(g))
			}
			g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 2)
			g.ApplyContinuousEffects()
			if g.FindPermanent(p.ID()).CurrentToughness(g) != want+2 {
				t.Fatal("saved reduction changed after resolution")
			}
		})
	}
}

func TestRONTemporaryLayerBranchRejectsDeclarations(t *testing.T) {
	for _, condition := range []string{
		`PermanentPassesFilter(Ref_TargetPermanent)`,
		`PermanentPassesFilter(Ref_TargetPermanent,DoesntHaveAbility(AnyRampage()))`,
		`PermanentPassesFilter(Ref_TargetPermanent,ToughnessIs(GreaterThanOrEqualTo(ValueX)))`,
		`PermanentPassesFilter(You,ToughnessIs(GreaterThanOrEqualTo(Integer(5))))`,
		`PermanentPassesFilter(Ref_TargetPermanent2,AnyPermanent)`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([If(` + condition + `,[CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbility([Rampage(2)])],UntilEndOfTurn)])])))])`
		nodes, err := Parse(strings.NewReader(source), "invalid-branch.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", condition)
		}
	}
}
