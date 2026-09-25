package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestGrantLandPlaysLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := GrantLandPlays(You, Fixed(1))
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.MaxLandPlays() != 3 {
		t.Fatal("grants did not accumulate")
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.MaxLandPlays() != 4 || g.MaxLandPlays() != 3 {
		t.Fatal("clone changed original allowance")
	}
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.MaxLandPlays() != 3 || g.ExtraLandPlaysGrantedThisTurn(b.PlayerID()) != 1 {
		t.Fatal("grant used active player instead of resolving controller")
	}
	g.doCleanupActions()
	if g.MaxLandPlays() != 1 || g.ExtraLandPlaysGrantedThisTurn(b.PlayerID()) != 0 {
		t.Fatal("allowance survived cleanup")
	}
}

func TestGrantLandPlaysAmountAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindNumber("allowance")
	body := Sequence(SnapshotNumber(Fixed(2), saved), GrantLandPlays(You, FromNumber(saved)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, -1} {
		if err := ApplyEffect(g, GrantLandPlays(You, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.MaxLandPlays() != 3 {
		t.Fatal("incorrect saved or nonpositive amount")
	}
	calls := 0
	if err := ApplyEffect(g, GrantLandPlays(ControllerOf(This), countedLifeAmount{&calls}), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("absent subject evaluated amount")
	}
}

func TestGrantLandPlaysValidation(t *testing.T) {
	for _, body := range []ResolvingEffect{
		GrantLandPlays(nil, Fixed(1)),
		GrantLandPlays(You, nil),
		GrantLandPlays(You, FromNumber(BindNumber("missing"))),
		GrantLandPlays(DeclarePlayerTarget("undeclared"), Fixed(1)),
	} {
		card := NewSorcery("Invalid allowance", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid allowance accepted")
		}
	}
	info := InspectDefinition(GrantLandPlays(You, Fixed(1)))
	if info.Properties.Outcome != OutcomeBenefit || info.Role != "resolving-effect" {
		t.Fatalf("incorrect metadata: %+v", info)
	}
}
