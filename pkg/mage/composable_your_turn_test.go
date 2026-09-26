package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsYourTurnResolutionUsesCurrentTurnAndStoredController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewInstant("Turn check", "")
	effect := IfState(IsYourTurn(), GainLifeFor(You, Fixed(2)))
	g.SetActivePlayerIndex(1)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("effect ran during opponent's turn")
	}
	clone := g.Clone()
	clone.SetActivePlayerIndex(0)
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("clone did not use its own active player")
	}
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 22 {
		t.Fatal("effect did not use the stored controller")
	}
}

func TestIsYourTurnDefinitionAndAbsence(t *testing.T) {
	condition := IsYourTurn()
	card := NewInstant("Turn check", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "it is your turn" {
		t.Fatalf("condition metadata = %+v", meta)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if !condition.checkState(g, StateContext{SourceID: uuid.New(), ControllerID: a.PlayerID()}) {
		t.Fatal("absent spell source changed turn check")
	}
	if condition.checkState(g, StateContext{ControllerID: uuid.Nil}) {
		t.Fatal("missing controller matched active player")
	}
}
