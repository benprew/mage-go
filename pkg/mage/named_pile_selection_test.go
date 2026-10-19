package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNamedPileSelection(t *testing.T) {
	for _, answer := range []int{0, 1, -1, 9} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		chooser := &partitionPlayer{Player: a, answers: []int{answer}}
		g := NewGame(chooser, b)
		subject := g.PutOnBattlefield(NewCreature("Subject", "", 1, 4), a.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 4), b.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 4), b.PlayerID())
		left, right, result := BindPermanentSet("left"), BindPermanentSet("right"), BindPermanentSet("selected")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: subject.ID()}
		if err := seedNamedPile(left, "First").Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if err := seedNamedPile(right, "Second").Apply(ctx); err != nil {
			t.Fatal(err)
		}
		piles := []NamedPermanentPile{{"Left", left}, {"Right", right}}
		effect := ChooseNamedPermanentPile(You, This, result, piles...)
		g.PhaseOut(first.ID())
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		g.PhaseIn([]uuid.UUID{first.ID()})
		selected := FromPermanentSet(result).resolvePermanents(ctx)
		want := first.ID()
		if answer == 1 {
			want = second.ID()
		}
		if len(selected) != 1 || selected[0].ID() != want {
			t.Fatal("wrong selected pile", answer, selected)
		}
		if len(chooser.reasons) != 1 {
			t.Fatal("wrong choice count")
		}
		if !strings.Contains(chooser.reasons[0], "Second") || !strings.Contains(chooser.reasons[0], "Right") {
			t.Fatal("pile membership missing from decision", chooser.reasons)
		}
		g.RemoveFromBattlefield(g.FindPermanent(want))
		g.PutOnBattlefield(selected[0].Card, b.PlayerID())
		if len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
			t.Fatal("selected pile followed returned object")
		}
		clone.PhaseIn([]uuid.UUID{first.ID()})
		if len(FromPermanentSet(result).resolvePermanents(&EffectContext{Game: clone})) != 1 {
			t.Fatal("clone lost selected visits")
		}
		if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 || len(chooser.reasons) != 1 {
			t.Fatal("missing subject retained result or made a choice")
		}
	}
}

func TestNamedPileSelectionEmptyAndSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &partitionPlayer{Player: a, answers: []int{1, 0}}
	g := NewGame(chooser, b)
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 1, 4), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 4), b.PlayerID())
	left, right, result := BindPermanentSet("left"), BindPermanentSet("right"), BindPermanentSet("selected")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: subject.ID()}
	if err := seedNamedPile(left, "First").Apply(ctx); err != nil {
		t.Fatal(err)
	}
	effect := ChooseNamedPermanentPile(You, This, result, NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right})
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
		t.Fatal("empty pile was not selectable")
	}
	chooser.during = func() {
		if err := seedNamedPile(left, "Absent").Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	selected := FromPermanentSet(result).resolvePermanents(ctx)
	if len(selected) != 1 || selected[0].ID() != first.ID() {
		t.Fatal("choice callback changed saved inputs")
	}
}

func TestNamedPileSelectionDefinition(t *testing.T) {
	left, right, result := BindPermanentSet("left"), BindPermanentSet("right"), BindPermanentSet("selected")
	producer := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right})
	piles := []NamedPermanentPile{{"Left", left}, {"Right", right}}
	for _, effect := range []Effect{
		ChooseNamedPermanentPile(You, This, result),
		ChooseNamedPermanentPile(You, This, result, piles[:1]...),
		ChooseNamedPermanentPile(You, This, nil, piles...),
		ChooseNamedPermanentPile(You, This, left, piles...),
		ChooseNamedPermanentPile(You, This, result, NamedPermanentPile{"", left}, piles[1]),
		ChooseNamedPermanentPile(You, This, result, piles[0], NamedPermanentPile{"Left", right}),
		ChooseNamedPermanentPile(You, This, result, piles[0], piles[0]),
		ChooseNamedPermanentPile(You, This, result, piles[0], NamedPermanentPile{"Right", nil}),
		ChooseNamedPermanentPile(nil, This, result, piles...),
		ChooseNamedPermanentPile(You, nil, result, piles...),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Sequence(producer, effect))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted invalid selection", effect)
		}
	}
	selection := ChooseNamedPermanentPile(You, This, result, piles...)
	for _, body := range []Effect{selection, Sequence(TapEach(FromPermanentSet(result)), producer, selection)} {
		card := NewArtifact("Invalid dominance", "", WithAbility(Activate(Costs(), body)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted read before producer")
		}
	}
	card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(producer, selection, TapEach(FromPermanentSet(result))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestNamedPileSelectionResolutionFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &partitionPlayer{Player: a, answers: []int{0, 1, 0, 0, 1, 1}}
	g := NewGame(chooser, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 4), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 4), b.PlayerID())
	left, right, result := BindPermanentSet("left"), BindPermanentSet("right"), BindPermanentSet("selected")
	body := Sequence(SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right}), ChooseNamedPermanentPile(You, This, result, NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right}), AtNext(EachEndStep, TapEach(FromPermanentSet(result))))
	card := NewArtifact("Choice source", "", WithAbility(Activate(Costs(), body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if len(chooser.reasons) != 6 || first.Tapped || second.Tapped {
		t.Fatal("wrong choice timing")
	}
	g.BouncePermanentToHand(source)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if !game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped {
			t.Fatal("selected results leaked between resolutions")
		}
	}
}

func seedNamedPile(result *PermanentSetBinding, name string) Effect {
	return FuncEffect("seed pile", EffectProperties{}, func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
		if g.resolution.createdTokenBindings == nil {
			g.resolution.createdTokenBindings = make(map[*PermanentSetBinding][]capturedPermanent)
		}
		members := MatchingPermanents(Named(name)).resolvePermanents(&EffectContext{Game: g, Controller: controller})
		saved := make([]capturedPermanent, 0, len(members))
		for _, p := range members {
			saved = append(saved, capturedPermanent{p.ID(), p.incarnationID})
		}
		g.resolution.createdTokenBindings[result] = saved
		return nil
	})
}

func TestNamedPileSelectionUnavailableSubject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &partitionPlayer{Player: a}
	g := NewGame(chooser, b)
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 1, 4), a.PlayerID())
	binding := BindPermanent("subject")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {subject.ID(), subject.incarnationID}}
	left, right, result := BindPermanentSet("left"), BindPermanentSet("right"), BindPermanentSet("result")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := ChooseNamedPermanentPile(You, From(binding), result, NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right})
	g.PhaseOut(subject.ID())
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(chooser.reasons) != 0 {
		t.Fatal("phased subject made a choice")
	}
	g.PhaseIn([]uuid.UUID{subject.ID()})
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(chooser.reasons) != 1 {
		t.Fatal("phase cycle lost subject")
	}
	g.RemoveFromBattlefield(g.FindPermanent(subject.ID()))
	g.PutOnBattlefield(subject.Card, a.PlayerID())
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(chooser.reasons) != 1 {
		t.Fatal("returned subject replaced saved visit")
	}
}
