package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedPoisonOperands(t *testing.T) {
	for name, body := range map[string]ResolvingEffect{
		"controller": AddPoisonFor(ControllerOf(This), Fixed(0)),
		"owner":      AddPoisonFor(OwnerOf(This), Fixed(0)),
		"lki":        AddPoisonFor(ControllerOfWithLKI(This), Fixed(0)),
		"amount":     AddPoisonFor(You, SourceCounters(core.Charge)),
		"arithmetic": AddPoisonFor(You, AddValues(Fixed(0), PowerOfWithLKI(This))),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("poison", Sequence(body)), ModeOf("other", AddPoisonFor(You, Fixed(0)))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || a.PoisonCounters() != 0 || b.PoisonCounters() != 0 {
					t.Fatal("query lost source or added poison", got)
				}
			}
			check()
			ids := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(ids)
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			if g.CaptureDamageSource(returned.ID()).SameObject(saved) {
				t.Fatal("return reused incarnation")
			}
			check()
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("shared waiting lifetime")
			}
		})
	}
}

func TestWaitingDelayedPoisonBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	reads := 0
	if err := ApplyEffect(g, AtNext(EachEndStep, AddPoisonFor(ControllerOf(This), waitingMultiAmount{&reads})), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query evaluated poison amount")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{
		AddPoisonFor(You, Fixed(1)),
		AddPoisonFor(ControllerOf(AttachedPermanent), Fixed(1)),
		AddPoisonFor(You, ValueAtResolution(Fixed(1))),
		AtNext(EachEndStep, AddPoisonFor(ControllerOf(This), Fixed(1))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred implicit or future source")
	}
	for _, body := range []ResolvingEffect{
		AddPoisonFor(nil, Fixed(1)), AddPoisonFor(You, nil),
		AddPoisonFor(FromPlayer(BindPlayer("missing")), Fixed(1)),
		AddPoisonFor(ControllerOf(DeclarePermanentTarget("missing", IsCreature)), Fixed(1)),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid delayed poison accepted")
		}
	}
}

func TestWaitingDelayedPoisonResolution(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "control"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			body := AddPoisonFor(ControllerOfWithLKI(This), AddValues(PowerOfWithLKI(This), Fixed(-1)))
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p.AddCounter(core.P1P1, 1)
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					returned := g.PutOnBattlefield(card, b.PlayerID())
					returned.AddCounter(core.P1P1, 5)
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("lost poison amount reference")
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			wantA, wantB := 2, 0
			if state == "control" {
				wantA, wantB = 0, 2
			}
			if clone.GetPlayer(a.PlayerID()).PoisonCounters() != wantA || clone.GetPlayer(b.PlayerID()).PoisonCounters() != wantB {
				t.Fatal("amount used stale or returned information")
			}
			if a.PoisonCounters() != 0 || b.PoisonCounters() != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone changed original")
			}
		})
	}
}
