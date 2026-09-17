package mage

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestComposableLoopRestoresOuterFrameOnError(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Member", "", 1, 1)
	card.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	member, saved := BindPermanent("member"), BindPermanent("outer")
	outer := capturedPermanent{card.ID(), permanent.incarnationID}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: outer}
	want := errors.New("body error")
	body := FuncEffect("fail", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		delete(g.resolution.permanentBindings, saved)
		return want
	})
	err := ForEach(MatchingPermanents(IsCreature), member, body).Apply(&EffectContext{Game: g, Controller: a.PlayerID()})
	if !errors.Is(err, want) || g.resolution.permanentBindings[saved] != outer || len(g.resolution.permanentBindings) != 1 {
		t.Fatal("loop did not restore the outer frame on error")
	}
}

func TestComposableLoopIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	member := BindPermanent("member")
	visited := 0
	effect := ForEach(MatchingPermanents(IsCreature), member, Sequence(
		FuncEffect("return other member", EffectProperties{}, func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
			visited++
			if visited == 1 {
				current := g.resolution.permanentBindings[member].id
				other := first
				if current == first.ID() {
					other = second
				}
				g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
				g.PutOnBattlefield(other, controller)
			}
			return nil
		}),
		Destroy(From(member)),
	))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if visited != 2 || len(g.FilterBattlefield(IsCreature)) != 1 {
		t.Fatal("loop followed a later incarnation or changed its member list")
	}
}

func TestComposableLoopScope(t *testing.T) {
	member, saved := BindPermanent("member"), BindPermanent("saved")
	loop := ForEach(MatchingPermanents(IsCreature), member, Sequence(Snapshot(From(member), saved), Destroy(From(saved))))
	card := func(effect ResolvingEffect) Card {
		return NewCreature("Loop", "", 1, 1, WithAbility(Activate(Costs(), effect)))
	}
	if err := ValidateCardDefinition("Loop", card(loop)); err != nil {
		t.Fatal(err)
	}
	for _, escaped := range []*PermanentBinding{member, saved} {
		if err := ValidateCardDefinition("Scope", card(Sequence(loop, Destroy(From(escaped))))); err == nil {
			t.Fatal("loop-local binding escaped")
		}
	}
	outer := Sequence(Snapshot(This, saved), ForEach(MatchingPermanents(IsCreature), member, Destroy(From(saved))))
	if err := ValidateCardDefinition("Outer", card(outer)); err != nil {
		t.Fatal(err)
	}
}
