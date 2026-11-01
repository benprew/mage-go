package interactive_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestHumanPlayer_ChooseTargets(t *testing.T) {
	human := interactive.NewHumanPlayer("Human")
	opponent := mage.NewBasePlayer("Opponent")
	game := mage.NewGame(human, opponent)
	possible := []uuid.UUID{human.PlayerID(), opponent.PlayerID()}
	result := make(chan []uuid.UUID, 1)
	go func() { result <- human.ChooseTargets(possible, 1, 1, game) }()
	select {
	case req := <-human.ChoiceRequests():
		if len(req.Options) != 2 || req.Options[1].ID != opponent.PlayerID() || req.Options[1].Label != "Opponent" {
			t.Fatalf("unexpected target options: %+v", req.Options)
		}
		human.ChoiceResponses() <- interactive.ChoiceResponse{SelectedIDs: []uuid.UUID{opponent.PlayerID()}}
	case targets := <-result:
		t.Fatalf("selected targets without a prompt: %v", targets)
	case <-time.After(time.Second):
		t.Fatal("no target prompt")
	}
	select {
	case targets := <-result:
		if len(targets) != 1 || targets[0] != opponent.PlayerID() {
			t.Fatalf("targets = %v, want opponent", targets)
		}
	case <-time.After(time.Second):
		t.Fatal("target selection did not complete")
	}
}

func TestHumanPlayer_ModalPreventionTarget(t *testing.T) {
	human := interactive.NewHumanPlayer("Human")
	opponent := mage.NewBasePlayer("Opponent")
	game := mage.NewGame(human, opponent)
	spell := dsl.NewInstant("Modal prevention", "", dsl.WithAbility(dsl.NewModalSpell([]dsl.Mode{
		{
			Label:   "Gain life",
			Targets: []dsl.Target{dsl.TargetPlayer()},
			Effects: []dsl.Effect{dsl.GainLifeTarget(dsl.Fixed(3))},
		},
		{
			Label:   "Prevent damage",
			Targets: []dsl.Target{dsl.TargetDamageAnyTarget()},
			Effects: []dsl.Effect{dsl.PreventDamageToTarget(dsl.Fixed(3))},
		},
	})))
	spell.SetOwner(human.PlayerID())
	human.AddToHand(spell)
	finished := make(chan error, 1)
	go func() { finished <- game.CastSpellByName(human.PlayerID(), spell.Name(), nil) }()
	for _, response := range []interactive.ChoiceResponse{
		{SelectedIndex: 1},
		{SelectedIDs: []uuid.UUID{opponent.PlayerID()}},
	} {
		select {
		case <-human.ChoiceRequests():
			human.ChoiceResponses() <- response
		case err := <-finished:
			t.Fatalf("cast finished before selection: %v", err)
		case <-time.After(time.Second):
			t.Fatal("missing choice prompt")
		}
	}
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cast did not complete")
	}
	game.ResolveStack()
	game.DealDamageToPlayer(opponent, 5, uuid.Nil)
	if opponent.Life() != 18 || human.Life() != 20 {
		t.Fatalf("life after damage: opponent=%d human=%d", opponent.Life(), human.Life())
	}
}
