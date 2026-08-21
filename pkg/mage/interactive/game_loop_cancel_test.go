package interactive_test

import (
	"context"
	"testing"
	"time"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestRunGameLoopContextCancelsWhileWaitingForHumanAction(t *testing.T) {
	human := interactive.NewHumanPlayer("Human")
	opponent := mage.NewBasePlayer("Opponent")
	human.AddToHand(mage.NewLand("Forest"))
	for range 20 {
		human.AddToLibrary(mage.NewLand("Forest"))
		opponent.AddToLibrary(mage.NewLand("Mountain"))
	}
	g := mage.NewGame(human, opponent)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		interactive.RunGameLoopContext(ctx, g, 0, 0)
	}()

	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	for {
		select {
		case msg, ok := <-human.ToTUI():
			if !ok {
				t.Fatal("game loop exited before requesting a human action")
			}
			if msg.Prompt != interactive.PromptMainPhaseAction {
				continue
			}
			cancel()
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				t.Fatal("game loop did not exit after cancellation")
			}
		case <-timeout.C:
			t.Fatal("game loop did not request a human action")
		}
	}
}
