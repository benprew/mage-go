package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDestructionResultQuery(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "zero", "regeneration", "indestructible", "exile", "phase", "cycle", "return", "shroud", "all illegal", "copy", "clone"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, "Volcanic Eruption")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
				q := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
				survivor := g.PutOnBattlefield(mage.NewCreature("Survivor", "", 1, 10), b.PlayerID())
				ids, x := []uuid.UUID{p.ID(), q.ID()}, 2
				if state == "zero" {
					ids, x = nil, 0
				}
				a.ManaPool().Add(core.Blue, 5)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, x); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				collection, _ := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.IsLand))
				watched := collection.Select(g)[:1]
				want, amount := true, 2
				switch state {
				case "zero":
					want, amount = false, 0
				case "regeneration":
					g.AddRegenerationShield(p.ID())
					want, amount = false, 1
				case "indestructible":
					p.GrantBaseAttr(core.Indestructible)
					want, amount = false, 1
				case "exile":
					if err := mage.ApplyEffect(g, mage.ExileIfWouldDieThisTurn(mage.This), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					want, amount = false, 1
				case "phase":
					g.PhaseOut(p.ID())
					want, amount = false, 1
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(p.Card, b.PlayerID())
					want, amount = false, 1
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
					want, amount = false, 1
				case "all illegal":
					p.GrantBaseAttr(core.Shroud)
					q.GrantBaseAttr(core.Shroud)
					want, amount = false, 0
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				case "clone":
					g = g.Clone()
					obj = g.StackPeek()
				}
				for range 2 {
					if got, known := g.InspectSpellDestruction(obj, watched); got != want || !known {
						t.Fatalf("got %v/%v want %v/true", got, known, want)
					}
				}
				if a.Life() != 20 || b.Life() != 20 || survivor.Damage != 0 {
					t.Fatal("query changed state")
				}
				g.ResolveTopOfStack()
				if g.GetPlayer(a.PlayerID()).Life() != 20-amount || g.GetPlayer(b.PlayerID()).Life() != 20-amount || g.FindPermanent(survivor.ID()).Damage != amount {
					t.Fatal("actual graveyard count differs")
				}
			})
		}
	}
}
