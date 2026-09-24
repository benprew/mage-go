package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestNumberSnapshotDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("count")
	body := Sequence(SnapshotNumber(CountersOn(This, Charge), n),
		RemoveAllCounters(This, Charge),
		AtNext(EachEndStep, GainLifeFor(You, FromNumber(n))),
		SnapshotNumber(Fixed(99), n))
	c := NewCreature("Source", "", 2, 2, WithAbility(Activate(Costs(), body)))
	c.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(c, a.PlayerID())
	for _, count := range []int{3, 5} {
		g.MutablePermanent(c.ID()).AddCounter(Charge, count)
		if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatalf("stack = %d", game.StackSize())
		}
		for _, object := range game.StackObjects() {
			game.CopyStackObjectDirect(object, b.PlayerID(), false)
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 28 || game.GetPlayer(b.PlayerID()).Life() != 28 {
			t.Fatalf("life A=%d B=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
		if FromNumber(n).resolveValue(&EffectContext{Game: game}) != 0 {
			t.Fatal("snapshot escaped resolution")
		}
		if game == clone && a.Life() != 20 {
			t.Fatal("clone changed original")
		}
	}
}

func TestNumberSnapshotValidation(t *testing.T) {
	n := BindNumber("number")
	read := GainLifeFor(You, FromNumber(n))
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"valid", Sequence(SnapshotNumber(Fixed(2), n), read), ""},
		{"read before write", Sequence(read, SnapshotNumber(Fixed(2), n)), "producer"},
		{"self read", SnapshotNumber(FromNumber(n), n), "producer"},
		{"overwrite", Sequence(SnapshotNumber(Fixed(2), n), SnapshotNumber(SubtractValues(FromNumber(n), Fixed(1)), n), read), ""},
		{"branch escape", Sequence(IfState(Controls(IsCreature), SnapshotNumber(Fixed(2), n)), read), "producer"},
		{"loop escape", Sequence(ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), SnapshotNumber(Fixed(2), n)), read), "producer"},
		{"delayed escape", Sequence(AtNext(EachEndStep, SnapshotNumber(Fixed(2), n)), read), "producer"},
		{"duplicate", Sequence(SnapshotNumber(Fixed(2), n), SnapshotNumber(Fixed(3), BindNumber("number"))), "duplicate"},
		{"nil value", SnapshotNumber(nil, n), "non-nil"},
		{"nil binding", SnapshotNumber(Fixed(2), nil), "non-nil"},
		{"empty name", SnapshotNumber(Fixed(2), BindNumber("")), "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), tc.body)))
			err := ValidateCardDefinition(c.Name(), c)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestNumberSnapshotOverwriteAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("number")
	ctx := &EffectContext{Game: g}
	for _, amount := range []int{7, -3, 0} {
		if err := SnapshotNumber(Fixed(amount), n).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if got := FromNumber(n).resolveValue(ctx); got != amount {
			t.Fatalf("saved = %d, want %d", got, amount)
		}
	}
	if err := SnapshotNumber(Fixed(4), n).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := SnapshotNumber(Fixed(8), n).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromNumber(n).resolveValue(ctx) != 4 {
		t.Fatal("clone shared binding")
	}
	body := Sequence(SnapshotNumber(SubtractValues(FromNumber(n), Fixed(1)), n), GainLifeFor(You, FromNumber(n)))
	if err := ApplyEffect(g, body, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || FromNumber(n).resolveValue(ctx) != 4 {
		t.Fatal("nested resolution failed to isolate and restore binding")
	}
}

func TestNumberSnapshotLoopIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 3, 3), a.PlayerID())
	n, member := BindNumber("amount"), BindPermanent("member")
	body := Sequence(SnapshotNumber(Fixed(4), n),
		ForEach(MatchingPermanents(IsCreature), member, Sequence(
			SnapshotNumber(PowerOf(From(member)), n), GainLifeFor(You, FromNumber(n)))),
		GainLifeFor(You, FromNumber(n)))
	if err := ApplyEffect(g, body, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 29 {
		t.Fatalf("life = %d", a.Life())
	}
}
