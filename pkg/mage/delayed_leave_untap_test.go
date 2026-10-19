package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedLeaveUntap(t *testing.T) {
	for _, event := range []string{"leave", "untap", "stun", "phase", "return-before-install", "leave-before-install", "phase-before-install", "untap-before-install", "control", "simultaneous"} {
		t.Run(event, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.TapPermanent(p)
			saved := BindPermanent("source")
			captures := map[*PermanentBinding]capturedPermanent{saved: {p.ID(), p.incarnationID}}
			amount := BindNumber("amount")
			effect := WhenLeavesOrUntaps(From(saved), GainLifeFor(You, FromNumber(amount)))
			switch event {
			case "leave-before-install", "return-before-install":
				g.BouncePermanentToHand(p)
				if event == "return-before-install" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase-before-install":
				g.PhaseOut(p.ID())
			case "untap-before-install":
				g.UntapPermanent(p)
			}
			if err := applyEffectWithCardSetCaptures(g, effect, p.ID(), a.PlayerID(), nil, map[*NumberBinding]int{amount: 3}, captures, nil, nil); err != nil {
				t.Fatal(err)
			}
			if event == "leave-before-install" || event == "return-before-install" {
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("absent saved visit subscribed")
				}
				return
			}
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("union must register one subscription")
			}
			g.triggers.ClearEndOfTurn()
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("cleanup removed unbounded trigger")
			}
			if event == "phase-before-install" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			clone := g.Clone()
			switch event {
			case "leave":
				g.ExilePermanent(g.FindPermanent(p.ID()))
			case "simultaneous":
				g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID())
				if err := ApplyEffect(g, DestroyEach(MatchingPermanents(AnyPermanent)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase":
				ids := g.PhaseOut(p.ID())
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("phasing triggered departure")
				}
				g.PhaseIn(ids)
				g.UntapPermanent(g.FindPermanent(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.UntapPermanent(g.FindPermanent(p.ID()))
			case "stun":
				g.MutablePermanent(p.ID()).AddCounter(core.Stun, 1)
				g.UntapPermanent(g.FindPermanent(p.ID()))
				g.PutTriggersOnStack()
				if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
					t.Fatal("replaced untap triggered")
				}
				g.UntapPermanent(g.FindPermanent(p.ID()))
			case "untap-before-install":
				g.UntapPermanent(g.FindPermanent(p.ID()))
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("earlier untap replayed")
				}
				g.TapPermanent(g.FindPermanent(p.ID()))
				g.UntapPermanent(g.FindPermanent(p.ID()))
			default:
				g.UntapPermanent(g.FindPermanent(p.ID()))
			}
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("first event did not consume union")
			}
			if live := g.FindPermanent(p.ID()); live != nil {
				g.ExilePermanent(live)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.Life() != 20 {
				t.Fatal("wrong pending timing or duplicate union trigger")
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			pending := g.Clone()
			g.ResolveStack()
			if a.Life() != 23 || b.Life() != 23 {
				t.Fatal("body controller or copy frame")
			}
			pending.ResolveStack()
			if pending.GetPlayer(a.PlayerID()).Life() != 23 {
				t.Fatal("pending clone lost body")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.DelayedTriggers()) != 1 {
				t.Fatal("clone changed with original")
			}
			clone.ExilePermanent(clone.FindPermanent(p.ID()))
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 23 {
				t.Fatal("waiting clone lost union")
			}
		})
	}
}

func TestDelayedLeaveUntapDeclarations(t *testing.T) {
	saved := BindPermanent("unbound")
	for _, effect := range []ResolvingEffect{
		WhenLeaves(nil, GainLifeFor(You, Fixed(1))), WhenUntaps(This, nil),
		WhenLeavesOrUntaps(From(saved), GainLifeFor(You, Fixed(1))),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid delayed declaration accepted")
		}
	}
	for _, effect := range []ResolvingEffect{WhenLeaves(This, GainLifeFor(You, Fixed(1))), WhenUntaps(This, GainLifeFor(You, Fixed(1)))} {
		card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDelayedLeaveUntapEventIsolation(t *testing.T) {
	for _, leave := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewArtifact("Watched", "")
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		other := g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID())
		effect := WhenUntaps(This, GainLifeFor(You, Fixed(1)))
		if leave {
			effect = WhenLeaves(This, GainLifeFor(You, Fixed(1)))
		}
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.TapPermanent(other)
		g.UntapPermanent(other)
		g.ExilePermanent(other)
		g.TapPermanent(g.FindPermanent(p.ID()))
		if leave {
			g.UntapPermanent(g.FindPermanent(p.ID()))
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
			t.Fatal("unrelated event fired subscription")
		}
		g.BouncePermanentToHand(g.FindPermanent(p.ID()))
		a.RemoveFromHand(card.ID())
		g.PutOnBattlefield(card, a.PlayerID())
		g.TapPermanent(g.FindPermanent(p.ID()))
		g.UntapPermanent(g.FindPermanent(p.ID()))
		g.PutTriggersOnStack()
		g.ResolveStack()
		want := 20
		if leave {
			want = 21
		}
		if a.Life() != want {
			t.Fatal("returned visit matched old subscription or lost pending departure")
		}
	}
}

func TestDelayedLeaveUntapNestedWaitingSource(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewArtifact("Source", "")
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		original := g.CaptureDamageSource(p.ID())
		effect := AtNext(EachEndStep, WhenLeavesOrUntaps(This, GainLifeFor(You, Fixed(1))))
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if returned {
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, a.PlayerID())
		}
		for range 2 {
			sources := g.WaitingDelayedPermanentSources()
			if len(sources) != 1 || !sources[0].SameObject(original) {
				t.Fatal("nested installation lost exact source read")
			}
			if a.Life() != 20 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("query installed or executed delayed event")
			}
		}
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		g.ResolveStack()
		if returned {
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("nested installation followed returned source")
			}
		} else {
			g.ExilePermanent(g.FindPermanent(p.ID()))
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("nested installation failed")
			}
		}
	}
}
