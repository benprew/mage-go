package interactive_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestHumanChosenManaPayment(t *testing.T) {
	human := interactive.NewHumanPlayer("A")
	g := mage.NewGame(human, mage.NewBasePlayer("B"))
	human.ManaPool().Add(core.Green, 123)
	result := mage.BindNumber("paid")
	done := make(chan error, 1)
	go func() {
		done <- mage.ApplyEffect(g, mage.Sequence(mage.PayChosenManaInto(mage.You, result), mage.GainLifeFor(mage.You, mage.FromNumber(result))), uuid.Nil, human.PlayerID(), nil)
	}()
	for _, answer := range []int{1, 1, 1, 2, 1, 3, 0} {
		select {
		case request := <-human.ChoiceRequests():
			if len(request.Options) > 10 {
				t.Fatal("unbounded menu")
			}
			human.ChoiceResponses() <- interactive.ChoiceResponse{SelectedIndex: answer}
		case <-time.After(time.Second):
			t.Fatal("missing number choice")
		}
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("payment did not finish")
	}
	if human.Life() != 143 || human.ManaPool().TotalMana() != 0 {
		t.Fatal("human payment lost chosen amount")
	}
}
