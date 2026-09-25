package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTokenCreationCopiesDefinitionInputs(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	types := []CardType{TypeCreature}
	subtypes := []string{"Citizen"}
	colors := []Color{White}
	keywords := []Keyword{Flying}
	effect := CreateColoredToken("Citizen", 1, 1, colors, types, subtypes, keywords...)
	types[0] = TypeArtifact
	subtypes[0] = "Goblin"
	colors[0] = Red
	keywords[0] = Haste
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	permanents := g.FilterBattlefield(IsToken)
	if len(permanents) != 1 {
		t.Fatalf("got %d tokens, want one", len(permanents))
	}
	p := permanents[0]
	if !p.HasType(TypeCreature) || p.HasType(TypeArtifact) || !p.HasSubType("Citizen") || p.HasSubType("Goblin") || !p.HasKeyword(Flying) || p.HasKeyword(Haste) || len(p.Colors()) != 1 || p.Colors()[0] != White {
		t.Fatal("token changed after the caller changed its definition slices")
	}
}

func TestTokenCreationHandlesProhibitedEntry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.effects.Rules.SetLandsCantEnter(true)
	for _, effect := range []Effect{
		CreateToken("Land creature", 1, 1, []CardType{TypeLand, TypeCreature}, nil),
		CreateTokenAttacking("Land creature", 1, 1, []CardType{TypeLand, TypeCreature}, nil),
		CreateTokenBlocking("Land creature", 1, 1, []CardType{TypeLand, TypeCreature}, nil),
	} {
		if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(g.FilterBattlefield(IsToken)); got != 0 {
		t.Fatalf("got %d tokens despite entry prohibition", got)
	}
}
