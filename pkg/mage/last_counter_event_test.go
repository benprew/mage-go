package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func lastCounterWatcher() Card {
	actor := BindPlayer("remover")
	return NewArtifact("Watcher", "", WithAbility(WhenEvent(
		ControllerRemovesLastCounter(core.Intervention, actor), GainLifeFor(FromPlayer(actor), Fixed(1)),
	)))
}

func TestLastCounterEventFacts(t *testing.T) {
	for _, scenario := range []string{"last", "partial", "zero", "wrong player", "wrong kind", "other permanent", "phase", "phase cycle", "departure", "return", "control before", "control after", "ability removed"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			card := lastCounterWatcher()
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(core.Intervention, 2)
			actor, counter, amount := a.PlayerID(), core.Intervention, 2
			want := 1
			switch scenario {
			case "partial":
				amount, want = 1, 0
			case "zero":
				amount, want = 0, 0
			case "wrong player":
				actor, want = b.PlayerID(), 0
			case "wrong kind":
				counter, want = core.Charge, 0
				p.AddCounter(counter, 2)
			case "other permanent":
				p = g.PutOnBattlefield(NewArtifact("Other", ""), a.PlayerID())
				p.AddCounter(counter, 2)
				want = 0
			case "phase", "phase cycle":
				g.PhaseOut(p.ID())
				want = 0
				if scenario == "phase cycle" {
					g.PhaseIn([]uuid.UUID{p.ID()})
					want = 1
				}
			case "control before":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 0
			case "ability removed":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
				want = 0
			}
			if err := ApplyEffect(g, RemoveCountersFrom(This, counter, Fixed(amount)), p.ID(), actor, nil); err != nil {
				t.Fatal(err)
			}
			pending := g.triggers.Pending()
			if len(pending) != want {
				t.Fatalf("pending=%d want=%d", len(pending), want)
			}
			if want == 1 {
				event := pending[0].event
				if event.PlayerID != actor || event.TargetID != p.ID() || event.TargetIncarnation != p.incarnationID || event.CounterType != counter || event.Amount != 2 || event.CountersRemaining != 0 {
					t.Fatalf("wrong removal facts: %+v", event)
				}
			}
			switch scenario {
			case "departure", "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				if scenario == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control after":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 20+want || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("event binding or clone isolation")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20+want {
				t.Fatal("clone lost event player")
			}
		})
	}
}

func TestLastCounterEventPayments(t *testing.T) {
	for _, kind := range []string{"fixed", "X", "chosen"} {
		for _, commit := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/failed", true: "/paid"}[commit], func(t *testing.T) {
				a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
				g := NewGame(a, NewBasePlayer("B"))
				p := g.PutOnBattlefield(lastCounterWatcher(), a.PlayerID())
				p.AddCounter(core.Intervention, 2)
				cost := RemoveCountersCost(core.Intervention, 2)
				switch kind {
				case "X":
					cost = RemoveXCountersFromSourceCost(core.Intervention)
				case "chosen":
					cost = RemoveAnyCountersCost(core.Intervention, BindNumber("paid"))
				}
				g.resolution.SetX(2)
				for range 3 {
					if !cost.CanPay(p.ID(), a.PlayerID(), g) {
						t.Fatal("cost unavailable")
					}
				}
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: p.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if len(g.triggers.Pending()) != 0 || g.FindPermanent(p.ID()).Counters[core.Intervention] != 2 {
					t.Fatal("payment trial published removal")
				}
				if !commit {
					g.MutablePermanent(p.ID()).RemoveCounter(core.Intervention, 1)
				}
				err = tx.Commit()
				want := 0
				if commit {
					want = 1
				}
				if (err == nil) != commit || len(g.triggers.Pending()) != want {
					t.Fatalf("commit=%v err=%v pending=%d", commit, err, len(g.triggers.Pending()))
				}
			})
		}
	}
}

func TestLastCounterEventInvalid(t *testing.T) {
	for _, counter := range []core.CounterType{core.CounterType(-1), core.NumCounters} {
		actor := BindPlayer("actor")
		card := NewArtifact("Bad", "", WithAbility(WhenEvent(ControllerRemovesLastCounter(counter, actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid counter accepted")
		}
	}
	card := NewArtifact("Bad", "", WithAbility(WhenEvent(ControllerRemovesLastCounter(core.Intervention, nil), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil actor accepted")
	}
}

func TestLastCounterEventAttributionAndIdentity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := lastCounterWatcher()
	p := g.PutOnBattlefield(card, a.PlayerID())
	cause := g.PutOnBattlefield(NewArtifact("Cause", ""), b.PlayerID())
	p.AddCounter(core.Intervention, 1)
	if err := ApplyEffect(g, RemoveCounters(core.Intervention, 1).Targeting(ToTarget()), cause.ID(), a.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.triggers.Pending()) != 1 || g.triggers.Pending()[0].event.SourceID != cause.ID() {
		t.Fatal("remover was derived from the cause's controller")
	}
	event := *g.triggers.Pending()[0].event
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(event)
	if len(g.triggers.Pending()) != 0 {
		t.Fatal("an old visit supplied a new event")
	}
	returned.AddCounter(core.Intervention, 1)
	g.RemoveFromBattlefield(returned)
	if len(g.triggers.Pending()) != 0 {
		t.Fatal("zone departure was treated as removal")
	}
	if core.EvtCountersRemoved.String() != "CountersRemoved" || !core.EvtCountersRemoved.IsAEventType() {
		t.Fatal("event enumeration is incomplete")
	}
}

func TestLastCounterEventPostRemovalLayers(t *testing.T) {
	for _, kind := range []string{"control", "abilities"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			p := g.PutOnBattlefield(lastCounterWatcher(), a.PlayerID())
			p.AddCounter(core.Intervention, 1)
			modification := ChangeController(This, You)
			if kind == "abilities" {
				modification = RemoveAbilities(This)
			}
			if err := ApplyEffect(g, UntilPermanentHasCounter(This, core.Intervention, modification), p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, RemoveAllCounters(This, core.Intervention), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.triggers.Pending()) != 1 {
				t.Fatal("removal event did not use completed layers")
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("post-removal trigger controller")
			}
		})
	}
}
