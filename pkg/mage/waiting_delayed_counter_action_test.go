package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func delayedCounterAction(kind string, subject PermanentRef, amount ResolutionValue) ResolvingEffect {
	switch kind {
	case "add":
		return AddCountersTo(subject, core.Charge, amount)
	case "bounded":
		return AddCountersUpTo(subject, core.Charge, amount, 7)
	case "remove":
		return RemoveCountersFrom(subject, core.Charge, amount)
	default:
		return RemoveAllCounters(subject, core.Charge)
	}
}

func TestWaitingDelayedCounterActionReferences(t *testing.T) {
	for _, kind := range []string{"add", "bounded", "remove", "all"} {
		for _, shape := range []string{"direct", "sequence", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "color", "empty"} {
				t.Run(kind+"/"+shape+"/"+state, func(t *testing.T) {
					a := &boundedCounterPlayer{BasePlayer: NewBasePlayer("A"), choice: -1}
					b := NewBasePlayer("B")
					g := NewGame(a, b)
					body := delayedCounterAction(kind, This, Fixed(1))
					delay := AtNext(EachEndStep, body)
					if shape == "sequence" {
						delay = AtNext(EachEndStep, Sequence(Sequence(body)))
					}
					if shape == "mode" {
						delay = AtNextModes(EachEndStep, ModeOf("First", body), ModeOf("Second", body))
					}
					card := NewArtifact("Source", "", WithAbility(Activate(Costs(), delay)))
					if err := ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					if state != "empty" {
						p.AddCounter(core.Charge, 2)
					}
					saved := g.CaptureDamageSource(p.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("unresolved delay supplies source")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(p)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							p = g.PutOnBattlefield(card, b.PlayerID())
							p.AddCounter(core.Charge, 2)
						}
					case "phased":
						g.PhaseOut(p.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "color":
						g.ChangeObjectColor(p.ID(), core.Red)
					}
					before := p.Counters[core.Charge]
					g.SetRandomResults([]int{7})
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 1 || !got[0].SameObject(saved) {
							t.Fatal("counter declaration lost saved source")
						}
						if got[0].HasColor(core.Red) != (state == "color") {
							t.Fatal("wrong current or departure facts")
						}
						if state == "control" && got[0].Controller() != b.PlayerID() {
							t.Fatal("wrong current controller")
						}
						if p.Counters[core.Charge] != before || a.calls != 0 || g.StackSize() != 0 {
							t.Fatal("query executes counter action")
						}
						got[0].card.AddType(core.TypeEnchantment)
						if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
							t.Fatal("query shares facts")
						}
					}
					if g.RandIntn(10) != 7 {
						t.Fatal("query consumes random results")
					}
					clone := g.Clone()
					for _, game := range []*Game{clone, g} {
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						if len(game.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("detected delay still waits")
						}
						game.PutTriggersOnStack()
						game.ResolveStack()
						want := before + 1
						if kind == "remove" {
							want = max(before-1, 0)
						}
						if kind == "all" {
							want = 0
						}
						if state == "returned" || state == "phased" {
							want = before
						}
						if live := game.FindPermanentIncludingPhased(p.ID()); live != nil && live.Counters[core.Charge] != want {
							t.Fatalf("counters %d want %d", live.Counters[core.Charge], want)
						}
						if game == clone && (len(g.WaitingDelayedPermanentSources()) != 1 || p.Counters[core.Charge] != before || a.calls != 0) {
							t.Fatal("clone shares lifetime or counters")
						}
					}
				})
			}
		}
	}
}

func TestWaitingDelayedCounterActionOperands(t *testing.T) {
	for _, kind := range []string{"add", "bounded", "remove", "all"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			for _, ref := range []PermanentRef{AttachedPermanent, From(BindPermanent("other"))} {
				if err := ApplyEffect(g, AtNext(EachEndStep, delayedCounterAction(kind, ref, Fixed(1))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, delayedCounterAction(kind, This, Fixed(1)))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("nonself or future body supplies source")
			}
			for _, ref := range []PermanentRef{nil, From(nil)} {
				card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, delayedCounterAction(kind, ref, Fixed(1))))))
				if ValidateCardDefinition(card.Name(), card) == nil {
					t.Fatal("invalid counter reference accepted")
				}
			}
			if kind == "all" {
				return
			}
			for _, amount := range []ResolutionValue{PowerOfWithLKI(This), CountersOnWithLKI(This, core.Charge), CountersOn(This, core.Charge), SourceCounters(core.Charge), MultiplyValues(Fixed(0), ManaValueOf(This))} {
				if err := ApplyEffect(g, AtNext(EachEndStep, delayedCounterAction(kind, AttachedPermanent, amount)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if len(g.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("typed amount lost source", amount.Text())
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("detected operand still waits")
				}
			}
		})
	}
}

func TestWaitingDelayedCounterActionZeroAndResult(t *testing.T) {
	for _, kind := range []string{"add", "bounded", "remove"} {
		for _, amount := range []int{0, -1, 1} {
			if amount == 1 && kind != "bounded" {
				continue
			}
			a := &boundedCounterPlayer{BasePlayer: NewBasePlayer("A"), choice: -1}
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			p.AddCounter(core.Charge, 7)
			if err := ApplyEffect(g, AtNext(EachEndStep, delayedCounterAction(kind, This, Fixed(amount))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 1 || a.calls != 0 || p.Counters[core.Charge] != 7 {
				t.Fatal("zero or negative amount changed declaration or state")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if p.Counters[core.Charge] != 7 || a.calls != 0 {
				t.Fatal("nonpositive action changed counters or offered choice")
			}
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	p.AddCounter(core.Charge, 2)
	result := BindNumber("removed")
	body := Sequence(RemoveCountersFromWithResult(This, core.Charge, Fixed(5), result), GainLifeFor(You, FromNumber(result)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || p.Counters[core.Charge] != 2 || a.Life() != 20 {
			t.Fatal("query wrote counter result")
		}
	}
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.FindPermanent(p.ID()).Counters[core.Charge] != 0 {
		t.Fatal("removed-count result missing")
	}
	if a.Life() != 20 || p.Counters[core.Charge] != 2 || len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("result or lifetime shared with clone")
	}
}
