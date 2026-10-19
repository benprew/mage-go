package interactive_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestOutsideGameHumanChoice(t *testing.T) {
	hp := interactive.NewHumanPlayer("Human")
	g := mage.NewGame(hp, mage.NewBasePlayer("B"))
	cards := []mage.Card{mage.NewArtifact("First", ""), mage.NewArtifact("Second", "")}
	for _, c := range cards {
		c.SetOwner(hp.PlayerID())
	}
	if err := g.SetOutsideGameCards(hp.PlayerID(), cards); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- mage.ApplyEffect(g, mage.PutOutsideGameCardInHand(mage.You, mage.CardFilter{}), uuid.Nil, hp.PlayerID(), nil)
	}()
	simulateTUI(t, hp, func(req interactive.ChoiceRequest) {
		if req.Type != interactive.ChoiceCardFromOutsideGame || len(req.Options) != 2 {
			t.Error("wrong outside-game choice", req)
		}
	}, interactive.ChoiceResponse{SelectedIDs: []uuid.UUID{cards[1].ID()}})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(hp.Hand()) != 1 || hp.Hand()[0].ID() != cards[1].ID() {
		t.Fatal("human answer was not used")
	}
}
