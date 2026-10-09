package interactive_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestHumanHandInspection(t *testing.T) {
	human := interactive.NewHumanPlayer("A")
	other := mage.NewBasePlayer("B")
	g := mage.NewGame(human, other)
	g.SetActivePlayerIndex(1)
	other.AddToHand(mage.NewLand("Private card"))
	done := make(chan error, 1)
	go func() {
		done <- mage.ApplyEffect(g, mage.LookAtHand(mage.You, mage.ActivePlayer), uuid.Nil, human.PlayerID(), nil)
	}()
	req := <-human.ChoiceRequests()
	if len(req.Options) != 1 || req.Options[0].Label != "Continue" || !strings.Contains(req.Reason, "Private card") {
		t.Fatal("missing private hand display")
	}
	human.ChoiceResponses() <- interactive.ChoiceResponse{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(g.HandLooks(other.PlayerID())) != 0 || len(g.PublicReveals()) != 0 || len(other.Hand()) != 1 {
		t.Fatal("private display changed visibility or zones")
	}
}

func TestHumanLibraryInspectionAndOrder(t *testing.T) {
	human := interactive.NewHumanPlayer("A")
	other := mage.NewBasePlayer("B")
	g := mage.NewGame(human, other)
	cards := []mage.Card{mage.NewLand("First"), mage.NewLand("Second"), mage.NewLand("Third")}
	requests := human.ChoiceRequests()
	responses := human.ChoiceResponses()
	other.SetLibrary(cards)
	g.SetActivePlayerIndex(1)
	done := make(chan error, 1)
	go func() {
		done <- mage.ApplyEffect(g, mage.LookAtLibraryTop(mage.You, mage.ActivePlayer, 3, true), uuid.Nil, human.PlayerID(), nil)
	}()
	req := <-requests
	if len(req.Options) != 1 || req.Options[0].Label != "Continue" {
		t.Fatal("missing private inspection acknowledgement")
	}
	responses <- interactive.ChoiceResponse{}
	for _, index := range []int{2, 0} {
		req = <-requests
		if req.Type != interactive.ChoiceCardFromLibrary {
			t.Fatal("missing order selection")
		}
		responses <- interactive.ChoiceResponse{SelectedIDs: []uuid.UUID{cards[index].ID()}}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	order := other.Library()
	if len(order) != 3 || order[0].ID() != cards[2].ID() || order[1].ID() != cards[0].ID() || order[2].ID() != cards[1].ID() {
		t.Fatal("wrong human order")
	}
}
