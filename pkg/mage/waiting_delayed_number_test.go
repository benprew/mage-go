package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedNumberOperands(t *testing.T) {
	for name, value := range map[string]ResolutionValue{
		"power":      PowerOfWithLKI(This),
		"zero":       MultiplyValues(Fixed(0), PowerOfWithLKI(This)),
		"controller": HandSizeOf(ControllerOfWithLKI(This)),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{U}", 3, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			n := BindNumber("saved")
			body := Sequence(SnapshotNumber(value, n), GainLifeFor(You, FromNumber(n)))
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || a.Life() != 20 {
					t.Fatal("snapshot lost source or query ran body")
				}
			}
			check()
			g.PhaseIn(g.PhaseOut(p.ID()))
			check()
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			want := 20
			if name == "power" {
				want = 23
			}
			if a.Life() != want || len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("snapshot followed returned source or shared clone")
			}
		})
	}
}

func TestWaitingDelayedNumberPurity(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	reads := 0
	n := BindNumber("number")
	body := Sequence(SnapshotNumber(AddValues(PowerOfWithLKI(This), ValueAtResolution(waitingTokenCount{&reads})), n), GainLifeFor(You, FromNumber(n)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query evaluated snapshot")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 || a.Life() != 23 {
		t.Fatal("snapshot did not evaluate once")
	}
}

func TestWaitingDelayedNumberScope(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	for _, value := range []ResolutionValue{Fixed(1), PowerOfWithLKI(AttachedPermanent), FromNumber(BindNumber("opaque"))} {
		if err := ApplyEffect(g, AtNext(EachEndStep, SnapshotNumber(value, BindNumber("result"))), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, SnapshotNumber(PowerOfWithLKI(This), BindNumber("future")))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("implicit or future source read inferred")
	}
}
