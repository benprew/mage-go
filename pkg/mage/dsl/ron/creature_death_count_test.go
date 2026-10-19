package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeCreatureDeathCount(t *testing.T) {
	for _, state := range []string{"normal", "token", "exile", "noncreature", "regenerate", "indestructible", "phase", "cycle", "return", "control", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			var card mage.Card = mage.NewCreature("Victim", "", 2, 2)
			if state == "token" {
				card = mage.NewToken("Victim", 2, 2, []core.CardType{core.TypeCreature}, nil)
			}
			if state == "noncreature" {
				card = mage.NewLand("Victim")
			}
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Hellfire"))
			want, damage := true, 4
			switch state {
			case "exile":
				if err := mage.ApplyEffect(g, mage.ExileIfWouldDieThisTurn(mage.This), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want, damage = false, 3
			case "noncreature":
				want, damage = false, 3
			case "regenerate":
				g.AddRegenerationShield(p.ID())
				want, damage = false, 3
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				want, damage = false, 3
			case "phase":
				g.PhaseOut(p.ID())
				want, damage = false, 3
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			}
			query, _ := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
			visits := query.Select(g)
			for range 2 {
				if proof, known := g.InspectSpellDestruction(obj, visits); !known || proof != want {
					t.Fatalf("proof %v/%v, want %v/true", proof, known, want)
				}
			}
			if g.GetPlayer(a.PlayerID()).Life() != 20 || g.GetPlayer(b.PlayerID()).Life() != 20 || g.FindPermanentIncludingPhased(p.ID()) == nil {
				t.Fatal("inspection changed state")
			}
			g.ResolveTopOfStack()
			controller := a.PlayerID()
			if state == "copy" {
				controller = b.PlayerID()
			}
			if g.GetPlayer(controller).Life() != 20-damage {
				t.Fatal("actual death count differs")
			}
			absent := state != "noncreature" && state != "regenerate" && state != "indestructible" && state != "phase"
			if (g.FindPermanentIncludingPhased(p.ID()) == nil) != absent {
				t.Fatal("destruction differs")
			}
		})
	}
}
