package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONTemporaryPTInspection(t *testing.T) {
	type spellCase struct {
		set, name        string
		power, toughness int
	}
	cases := make([]spellCase, 0, 19)
	cases = append(cases, []spellCase{
		{"ARN", "Army of Allah", 2, 0}, {"ARN", "Piety", 0, 3},
		{"DRK", "Holy Light", -1, -1}, {"DRK", "Marsh Gas", -2, 0}, {"DRK", "Morale", 1, 1},
		{"LEG", "Hell Swarm", -1, 0}, {"LEG", "Shield Wall", 0, 2},
	}...)
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		cases = append(cases, spellCase{set, "Giant Growth", 3, 3})
	}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		cases = append(cases, spellCase{set, "Righteousness", 7, 7})
	}

	for _, tc := range cases {
		states := []string{"normal", "copy", "clone", "phased", "returned"}
		if tc.name == "Righteousness" {
			states = append(states, "attacker left")
		}
		for _, state := range states {
			t.Run(tc.name+"/"+tc.set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "{U}", 4, 4)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if tc.name == "Piety" || tc.name == "Righteousness" {
					attacker := g.PutOnBattlefield(mage.NewCreature("Other attacker", "", 4, 4), b.PlayerID())
					g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
					g.GetCombat().AddBlocker(p.ID(), attacker.ID())
					if state == "attacker left" {
						g.RemoveFromBattlefield(attacker)
					}
				} else {
					g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
				}
				var targets []uuid.UUID
				if tc.name == "Giant Growth" || tc.name == "Righteousness" {
					targets = append(targets, p.ID())
				}
				obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name), targets...)
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "phased" {
					g.PhaseOut(p.ID())
				}
				if state == "returned" {
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(card, a.PlayerID())
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("native temporary P/T tree is unknown")
					}
				}
				if g.FindPermanentIncludingPhased(p.ID()).CurrentPower(g) != 4 {
					t.Fatal("query installed a layer")
				}
				g.ResolveTopOfStack()
				affected := state != "phased"
				if state == "returned" && (tc.name == "Giant Growth" || tc.name == "Righteousness" || tc.name == "Army of Allah" || tc.name == "Morale" || tc.name == "Piety") {
					affected = false
				}
				if state == "copy" && tc.name == "Shield Wall" {
					affected = false
				}
				wantPower, wantToughness := 4, 4
				if affected {
					wantPower += tc.power
					wantToughness += tc.toughness
				}
				current := g.FindPermanentIncludingPhased(p.ID())
				if current.CurrentPower(g) != wantPower || current.CurrentToughness(g) != wantToughness {
					t.Fatalf("got %d/%d want %d/%d", current.CurrentPower(g), current.CurrentToughness(g), wantPower, wantToughness)
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if current.CurrentPower(g) != 4 || current.CurrentToughness(g) != 4 {
					t.Fatal("effect survived cleanup")
				}
			})
		}
	}
}
