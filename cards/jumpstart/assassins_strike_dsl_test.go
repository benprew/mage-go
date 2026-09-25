package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestAssassinsStrikeOutcomes(t *testing.T) {
	for _, scenario := range []string{"destroy", "indestructible", "regenerate", "illegal", "return", "control", "own creature", "empty hand", "replacement"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			owner := b.PlayerID()
			if scenario == "own creature" {
				owner = a.PlayerID()
			}
			card := mage.NewCreature("Strike target", "", 2, 2)
			card.SetOwner(owner)
			target := g.PutOnBattlefield(card, owner)
			if scenario == "indestructible" {
				target.GrantBaseAttr(core.Indestructible)
			}
			if scenario == "regenerate" {
				g.AddRegenerationShield(target.ID())
			}
			discarder := gametest.PlayerB
			if scenario == "control" || scenario == "own creature" {
				discarder = gametest.PlayerA
			}
			var chosen uuid.UUID
			if scenario != "empty hand" {
				g.AddCard(core.ZoneHand, discarder, "Mountain")
				chosen = g.GetPlayer(discarder).Hand()[0].ID()
				g.AddCard(core.ZoneHand, discarder, "Forest")
				g.ChooseDiscard(discarder, "Mountain")
			}
			if scenario == "replacement" {
				g.AddCard(core.ZoneBattlefield, discarder, "Library of Leng")
			}
			events := 0
			watcher := mage.NewTriggered(core.EvtDiscard, false, mage.DrawFor(mage.You, mage.Fixed(1))).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
				events++
				if evt.PlayerID != g.GetPlayer(discarder).PlayerID() || evt.SourceID != chosen {
					t.Error("incorrect discard event player or card")
				}
				if scenario == "destroy" && g.FindPermanent(card.ID()) != nil {
					t.Error("discard occurred before destruction")
				}
				return false
			})
			g.PutOnBattlefield(mage.NewArtifact("Discard observer", "", mage.WithAbility(watcher)), a.PlayerID())
			spell, err := mage.CreateCard("Assassin's Strike")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Black, 6)
			g.SetStep(core.PrecombatMain)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
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
			wantEvents, wantHand, wantGraveyard := 1, 0, 1
			switch scenario {
			case "illegal", "return":
				wantEvents, wantHand, wantGraveyard = 0, 1, 0
			case "empty hand":
				wantEvents, wantGraveyard = 0, 0
			case "replacement":
				wantGraveyard = 0
				if len(b.Library()) != 1 || b.Library()[0].ID() != chosen {
					t.Error("discard replacement did not put the chosen card on the library")
				}
			}
			if events != wantEvents {
				t.Errorf("discard events = %d, want %d", events, wantEvents)
			}
			g.AssertHandCount(discarder, "Mountain", wantHand)
			g.AssertGraveyardCount(discarder, "Mountain", wantGraveyard)
			if scenario != "empty hand" {
				g.AssertHandCount(discarder, "Forest", 1)
			}
			survives := scenario == "indestructible" || scenario == "regenerate" || scenario == "return"
			if (g.FindPermanent(card.ID()) != nil) != survives {
				t.Error("wrong target destination")
			}
		})
	}
}
