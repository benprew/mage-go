package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestMireTritonEntryResolution(t *testing.T) {
	for _, scenario := range []string{"normal", "empty library", "one card", "double mill", "double gain", "prevent gain", "source leaves"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			cards := []mage.Card{mage.NewArtifact("top", ""), mage.NewArtifact("second", ""), mage.NewArtifact("third", ""), mage.NewArtifact("bottom", "")}
			if scenario == "empty library" {
				cards = nil
			}
			if scenario == "one card" {
				cards = cards[:1]
			}
			a.SetLibrary(cards)
			modifier := mage.NewArtifact("Modifier", "")
			g.PutOnBattlefield(modifier, b.PlayerID())
			wantMill := min(2, len(cards))
			if scenario == "double mill" {
				g.AddMillReplacement(modifier.ID(), mage.DoubleMillForOpponents(modifier.ID(), b.PlayerID()))
				wantMill = 4
			}
			wantGain := 2
			if scenario == "double gain" {
				wantGain = 4
			}
			if scenario == "prevent gain" {
				wantGain = 0
			}
			g.AddLifeGainModifier(modifier.ID(), func(_ *mage.Game, _ uuid.UUID, amount int) int {
				if len(a.Graveyard()) != wantMill {
					t.Errorf("milled %d cards before gain, want %d", len(a.Graveyard()), wantMill)
				}
				if scenario == "double gain" {
					return amount * 2
				}
				if scenario == "prevent gain" {
					return 0
				}
				return amount
			})
			card, err := mage.CreateCard("Mire Triton")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("stack size = %d, want 1", g.StackSize())
			}
			if scenario == "source leaves" {
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			}
			g.ResolveStack()
			g.AssertLife(gametest.PlayerA, 20+wantGain)
			g.AssertLife(gametest.PlayerB, 20)
			if got := g.PlayerLifeGainedThisTurn(a.PlayerID()); got != wantGain {
				t.Errorf("life gain event amount = %d, want %d", got, wantGain)
			}
			if len(a.Graveyard()) != wantMill || len(a.Library()) != len(cards)-wantMill {
				t.Fatalf("graveyard = %d, library = %d", len(a.Graveyard()), len(a.Library()))
			}
			for i, milled := range a.Graveyard() {
				if milled.ID() != cards[i].ID() {
					t.Error("did not mill from the top in order")
				}
			}
			if a.DrewFromEmpty() || len(b.Graveyard()) != 0 {
				t.Error("mill caused a draw or changed the opponent's graveyard")
			}
		})
	}
}
