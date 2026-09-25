package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatedTokenAbilitiesHaveIndependentRuntimeState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := NewTriggered(EvtZoneChange, false)
	effect := TokenWithAbilities(CreateTokens(2, "Spirit", 1, 1, []CardType{TypeCreature}, nil), ability)
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 2 {
		t.Fatalf("token count = %d, want 2", len(tokens))
	}
	first, second := tokens[0].RuntimeAbilities[0], tokens[1].RuntimeAbilities[0]
	if first == second || first.AbilityID() == second.AbilityID() {
		t.Fatal("tokens share an ability instance or identity")
	}
	if first.Source() != tokens[0].ID() || second.Source() != tokens[1].ID() {
		t.Fatal("token ability has the wrong source")
	}
	if ability.Source() != uuid.Nil || ability.Controller() != uuid.Nil {
		t.Fatal("entry changed the ability declaration")
	}
}
