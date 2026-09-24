package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestComposableLibraryChoiceValidation(t *testing.T) {
	for _, build := range []func(ResolutionValue) ResolvingEffect{ScryAtResolution, SurveilAtResolution} {
		target := DeclarePermanentTarget("creature", IsCreature)
		saved := BindPermanent("saved")
		for _, tc := range []struct {
			value ResolutionValue
			path  string
		}{
			{nil, ".amount"},
			{PowerOf(target), ".amount.subject"},
			{PowerOf(From(saved)), ".amount.subject"},
			{MaxValue(Fixed(1), nil), ".amount.right"},
		} {
			card := NewSorcery("Invalid choice", "", WithAbility(Spell(build(tc.value))))
			err := ValidateCardDefinition(card.Name(), card)
			if err == nil || !strings.Contains(err.Error(), tc.path) {
				t.Fatalf("diagnostic = %v, want %s", err, tc.path)
			}
		}
		card := NewSorcery("Choice", "", WithAbility(Spell(Sequence(Snapshot(target, saved), build(PowerOf(From(saved)))), WithTargets(target))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		info := InspectDefinition(build(MaxValue(Fixed(1), PowerOf(target))))
		if len(info.Children) != 1 || info.Children[0].Role != "arithmetic-value" || info.Properties.Outcome != OutcomeBenefit {
			t.Fatalf("metadata = %+v", info)
		}
	}
}

type libraryChoiceValue struct{ calls int }

func (*libraryChoiceValue) Text() string { return "dynamic amount" }
func (v *libraryChoiceValue) Resolve(GameReader, uuid.UUID, uuid.UUID, []uuid.UUID) int {
	v.calls++
	return 2
}

func TestLibraryChoiceInspectionDoesNotResolve(t *testing.T) {
	for _, build := range []func(ValueSource) Effect{Scry, Surveil} {
		value := &libraryChoiceValue{}
		effect := build(value)
		info := InspectDefinition(effect)
		if value.calls != 0 || len(info.Children) != 1 || !strings.Contains(info.Text, "dynamic amount") {
			t.Fatalf("calls = %d, metadata = %+v", value.calls, info)
		}
		for _, amount := range []ValueSource{nil, (*libraryChoiceValue)(nil)} {
			card := NewSorcery("Nil choice", "", WithAbility(Spell(build(amount))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), ".amount") {
				t.Fatalf("diagnostic = %v", err)
			}
		}
	}
}

func TestComposableLibraryChoiceCloneAndAbsentBinding(t *testing.T) {
	for _, build := range []func(ResolutionValue) ResolvingEffect{ScryAtResolution, SurveilAtResolution} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		top := NewLand("Top")
		a.SetLibrary([]Card{top})
		clone := g.Clone()
		saved := BindPermanent("absent")
		if err := ApplyEffect(clone, Sequence(Snapshot(This, saved), build(PowerOf(From(saved)))), uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(clone.GetPlayer(a.PlayerID()).Library()) != 1 {
			t.Fatal("absent binding changed library")
		}
		if err := ApplyEffect(clone, Sequence(build(Fixed(4)), DrawFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 0 || len(a.Library()) != 1 {
			t.Fatal("clone or sequence lost isolation")
		}
		if err := ApplyEffect(clone, build(Fixed(4)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
}
