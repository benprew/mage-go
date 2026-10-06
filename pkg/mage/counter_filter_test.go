package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterFilters(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 2), a.PlayerID())
	for _, count := range []int{0, 1, 2, 3, 255} {
		g.MutablePermanent(p.ID()).Counters[core.Tide] = uint8(count)
		g.MutablePermanent(p.ID()).Counters[core.Net] = 3
		for _, want := range []int{0, 1, 3, 255, 256, -1} {
			if got := CounterCountEquals(core.Tide, want).Match(g.FindPermanent(p.ID()), g); got != (count == want) {
				t.Fatalf("count %d, threshold %d: %v", count, want, got)
			}
		}
		if got := HasCounter(core.Tide).Match(g.FindPermanent(p.ID()), g); got != (count > 0) {
			t.Fatalf("count %d: has counter %v", count, got)
		}
		if !HasCounter(core.Net).Match(g.FindPermanent(p.ID()), g) || !CounterCountEquals(core.Net, 3).Match(g.FindPermanent(p.ID()), g) {
			t.Fatal("counter kinds were combined")
		}
	}
	if HasCounter(core.Tide).Match(nil, g) || CounterCountEquals(core.Tide, 0).Match(nil, g) {
		t.Fatal("nil permanent matched")
	}
}

func TestCounterFilterConditionIdentity(t *testing.T) {
	for _, attached := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Subject", "", 2, 3)
		card.SetOwner(a.PlayerID())
		subject := g.PutOnBattlefield(card, a.PlayerID())
		source := subject
		var ref PermanentRef = This
		if attached {
			source = g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), b.PlayerID())
			g.Attach(source.ID(), subject.ID())
			ref = AttachedPermanent
		}
		condition := PermanentMatchesFilter(ref, CounterCountEquals(core.Net, 0))
		ctx := StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: source.ControllerID()}
		check := func(game *Game, want bool) {
			t.Helper()
			if got := condition.checkState(game, ctx); got != want {
				t.Fatalf("attached %v: got %v, want %v", attached, got, want)
			}
		}
		check(g, true)
		clone := g.Clone()
		g.MutablePermanent(subject.ID()).Counters[core.Net] = 1
		check(g, false)
		check(clone, true)
		g.MutablePermanent(subject.ID()).Counters[core.Net] = 0
		phase := g.PhaseOut(subject.ID())
		check(g, false)
		g.PhaseIn(phase)
		check(g, true)
		g.BouncePermanentToHand(g.FindPermanent(subject.ID()))
		check(g, false)
		a.RemoveFromHand(card.ID())
		g.PutOnBattlefield(card, a.PlayerID())
		check(g, false)
		check(clone, true)
		ctx.SourceIncarnation = uuid.New()
		check(clone, false)
	}
}
