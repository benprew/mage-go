package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestLaunchPartyOutcomes(t *testing.T) {
	for _, scenario := range []string{"destroy", "indestructible", "regenerate", "illegal", "return", "control", "sacrifice target", "no sacrifice"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Exquisite Blood")
			owner := b.PlayerID()
			if scenario == "sacrifice target" {
				owner = a.PlayerID()
			} else if scenario != "no sacrifice" {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
			}
			card := mage.NewCreature("Launch Party target", "", 2, 2)
			card.SetOwner(owner)
			target := g.PutOnBattlefield(card, owner)
			if scenario == "indestructible" {
				target.GrantBaseAttr(core.Indestructible)
			}
			if scenario == "regenerate" {
				g.AddRegenerationShield(target.ID())
			}
			spell, err := mage.CreateCard("Launch Party")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Black, 4)
			err = g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0)
			if scenario == "no sacrifice" {
				if err == nil {
					t.Fatal("cast without the sacrifice payment")
				}
				g.AssertHandCount(gametest.PlayerA, "Launch Party", 1)
				if a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 {
					t.Fatal("failed cast changed mana or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "sacrifice target" {
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("target was not sacrificed during payment")
				}
			} else {
				g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
			}
			if scenario == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if scenario == "illegal" || scenario == "return" {
				g.RemoveFromBattlefield(target)
				if scenario == "return" {
					g.PutOnBattlefield(card, owner)
				}
			}
			g.ResolveStack()
			g.PutTriggersOnStack()
			g.ResolveStack()
			wantA, wantB := 22, 18
			switch scenario {
			case "illegal", "return", "sacrifice target":
				wantA, wantB = 20, 20
			case "control":
				wantA, wantB = 18, 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life A=%d B=%d, want A=%d B=%d", a.Life(), b.Life(), wantA, wantB)
			}
			survives := scenario == "indestructible" || scenario == "regenerate" || scenario == "return"
			if (g.FindPermanent(card.ID()) != nil) != survives {
				t.Fatal("wrong target destination")
			}
		})
	}
}
