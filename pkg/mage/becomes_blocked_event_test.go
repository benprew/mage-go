package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestBecomesBlockedEventTransition(t *testing.T) {
	for _, tc := range []struct {
		name string
		step func(*Game, *Permanent, *Permanent, *Permanent)
	}{
		{"declared together", func(g *Game, attacker, first, second *Permanent) {
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: attacker.ID()}})
		}},
		{"enters blocking", func(g *Game, attacker, _, _ *Permanent) {
			card := NewCreature("Entering", "", 1, 1)
			g.PutOnBattlefieldBlocking(card, g.players[1].PlayerID(), attacker.ID())
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := NewCreature("Attacker", "", 4, 4, WithAbility(NewTriggered(EvtCreatureBecomesBlocked, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{})))
			attacker.SetOwner(a.PlayerID())
			first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
			g.PutOnBattlefield(attacker, a.PlayerID())
			g.PutOnBattlefield(first, b.PlayerID())
			g.PutOnBattlefield(second, b.PlayerID())
			g.combat.AddAttacker(attacker.ID(), b.PlayerID())
			clone := g.Clone()
			tc.step(clone, clone.FindPermanent(attacker.ID()), clone.FindPermanent(first.ID()), clone.FindPermanent(second.ID()))
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
				t.Fatalf("life after first transition = %d, want 21", got)
			}
			tc.step(clone, clone.FindPermanent(attacker.ID()), clone.FindPermanent(first.ID()), clone.FindPermanent(second.ID()))
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
				t.Fatalf("already blocked attacker triggered again: life = %d", got)
			}
		})
	}
}
