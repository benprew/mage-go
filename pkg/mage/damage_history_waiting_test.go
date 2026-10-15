package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageHistoryRecipientsWaitingSources(t *testing.T) {
	for _, shape := range []string{"direct", "wrapped", "nested"} {
		t.Run(shape, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("History source", "", 2, 4)
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			var source DamageSourceRef = This
			if shape == "wrapped" {
				source = DamageSourceOf(This)
			}
			subjects := PermanentsDealtDamageBy(source, MatchingPermanents(IsPlaneswalker))
			if shape == "nested" {
				subjects = PermanentsDealtDamageBy(DamageSourceOf(AttachedPermanent), subjects)
			}
			reads := 0
			body := DealDamageEach(DamageSourceOf(AttachedPermanent), DamageRecipients(EachPlayer, subjects), waitingMultiAmount{&reads})
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			for _, state := range []string{"present", "phased", "returned"} {
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "returned":
					g.PhaseIn([]uuid.UUID{p.ID()})
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					returned := g.PutOnBattlefield(card, b.PlayerID())
					g.ChangeObjectColor(returned.ID(), core.Red)
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || got[0].HasColor(core.Red) || reads != 0 {
						t.Fatalf("%s: waiting history reference lost identity or made a query", state)
					}
				}
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
				t.Fatal("clone shared delayed lifetime or queried damage")
			}
		})
	}
}
