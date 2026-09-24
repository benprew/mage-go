package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomPlayerBindingResolution(t *testing.T) {
	for _, index := range []int{0, 1} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLibrary([]Card{NewArtifact("A top", "")})
		b.SetLibrary([]Card{NewArtifact("B top", "")})
		saved := BindPlayer("recipient")
		body := Sequence(ChooseRandomPlayer(EachPlayer, saved), Mill(FromPlayer(saved), Fixed(1)))
		g.SetRandomResults([]int{index})
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(a.Graveyard()) != 1-index || len(b.Graveyard()) != index {
			t.Fatal("random choice milled the wrong player")
		}
		if _, ok := FromPlayer(saved).resolvePlayer(&EffectContext{Game: g}); ok {
			t.Fatal("binding leaked past resolution")
		}
	}
}

func TestRandomPlayerBindingAbsenceAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.SetRandomResults([]int{1, 0})
	if err := ChooseRandomPlayer(EachPlayer, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	cloneCtx := &EffectContext{Game: clone, Controller: a.PlayerID()}
	if err := ChooseRandomPlayer(OpponentsOf(ControllerOf(This)), saved).Apply(cloneCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(cloneCtx); ok {
		t.Fatal("empty set retained prior binding")
	}
	if got, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || got != b.PlayerID() {
		t.Fatal("clone changed original binding")
	}
	if clone.RandIntn(2) != 0 || g.RandIntn(2) != 0 {
		t.Fatal("empty set consumed randomness or clone shared queue")
	}
	clone.resolution.Clear()
	if len(clone.resolution.playerBindings) != 0 {
		t.Fatal("clear retained player bindings")
	}
}

func TestRandomPlayerBindingNestedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	body := Sequence(ChooseRandomPlayer(OpponentsOf(You), saved),
		FuncEffect("nested", EffectProperties{}, func(g *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
			return ApplyEffect(g, GainLifeFor(FromPlayer(saved), Fixed(7)), source, controller, nil)
		}), GainLifeFor(FromPlayer(saved), Fixed(2)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("nested frame leaked: %d %d", a.Life(), b.Life())
	}
}

func TestPlayerBindingValidation(t *testing.T) {
	saved := BindPlayer("saved")
	choose := ChooseRandomPlayer(EachPlayer, saved)
	read := Mill(FromPlayer(saved), Fixed(1))
	for _, tt := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"valid", Sequence(choose, read), ""},
		{"missing", read, "producer"},
		{"before", Sequence(read, choose), "producer"},
		{"branch", Sequence(IfState(Controls(IsCreature), choose), read), "producer"},
		{"duplicate", Sequence(choose, ChooseRandomPlayer(EachPlayer, BindPlayer("saved"))), "duplicate"},
		{"cross category duplicate", Sequence(choose, Snapshot(This, BindPermanent("saved"))), "duplicate"},
		{"nil set", ChooseRandomPlayer(nil, saved), "non-nil"},
		{"nil binding", ChooseRandomPlayer(EachPlayer, nil), "non-nil"},
		{"nil read", Mill(FromPlayer(nil), Fixed(1)), "non-nil"},
		{"empty name", ChooseRandomPlayer(EachPlayer, BindPlayer("")), "name"},
		{"loop escape", Sequence(ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), choose), read), "producer"},
		{"delayed escape", Sequence(AtNext(EachEndStep, choose), read), "producer"},
		{"set dependency", ChooseRandomPlayer(OpponentsOf(FromPlayer(saved)), saved), "producer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), tt.body)))
			err := ValidateCardDefinition(card.Name(), card)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	info := InspectDefinition(choose)
	if info.Role != "random-player-choice" || len(info.Children) != 2 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestRandomPlayerBindingLoopAndDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	saved := BindPlayer("recipient")
	body := Sequence(ChooseRandomPlayer(OpponentsOf(You), saved),
		ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), Sequence(
			GainLifeFor(FromPlayer(saved), Fixed(1)),
			ChooseRandomPlayer(OpponentsOf(FromPlayer(saved)), saved))),
		AtNext(EachEndStep, GainLifeFor(FromPlayer(saved), Fixed(3))),
		ChooseRandomPlayer(OpponentsOf(FromPlayer(saved)), saved))
	card := NewSorcery("Capture", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("loop failed to restore the outer player binding")
	}
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatal("delayed trigger lost its captured player")
		}
		if game == clone && b.Life() != 22 {
			t.Fatal("delayed resolution changed original game")
		}
	}
}

func TestRandomPlayerBindingRepeatedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	body := Sequence(ChooseRandomPlayer(EachPlayer, saved), GainLifeFor(FromPlayer(saved), Fixed(1)))
	g.SetRandomResults([]int{0, 1})
	card := NewSorcery("Random", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	InspectDefinition(body)
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("inspection consumed randomness or repeated resolution reused a binding")
	}
}

func TestRandomPlayerBindingSpellCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	ability := Spell(Sequence(ChooseRandomPlayer(OpponentsOf(You), saved), GainLifeFor(FromPlayer(saved), Fixed(2))))
	card := NewSorcery("Random", "", WithAbility(ability))
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("copy did not choose from its controller's opponents")
	}
	g.ResolveStackObject(original)
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("original reused copy binding")
	}
}
