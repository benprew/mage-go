package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCombatExitOperands(t *testing.T) {
	for name, action := range map[string]func(PermanentRef) ResolvingEffect{"remove": RemovePermanentFromCombat, "unblock": RemovePermanentFromCombatAndUnblock} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", 1, 10)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			g.combat.AddAttackerIncarnation(p.ID(), saved.Incarnation(), b.PlayerID())
			if err := ApplyEffect(g, AtNext(EachEndStep, action(This)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("lost saved source", got)
				}
			}
			check()
			if !g.combat.IsAttacking(p.ID()) {
				t.Fatal("query removed attacker")
			}
			ids := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(ids)
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(c.ID())
			returned := g.PutOnBattlefield(c, a.PlayerID())
			if g.CaptureDamageSource(returned.ID()).SameObject(saved) {
				t.Fatal("return retained incarnation")
			}
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("shared waiting lifetime")
			}
		})
	}
}

func TestWaitingDelayedCombatExitBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 10), a.PlayerID())
	for _, action := range []func(PermanentRef) ResolvingEffect{RemovePermanentFromCombat, RemovePermanentFromCombatAndUnblock} {
		for _, body := range []ResolvingEffect{action(AttachedPermanent), AtNext(EachEndStep, action(This))} {
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		for _, ref := range []PermanentRef{nil, From(BindPermanent("missing")), DeclarePermanentTarget("missing", IsCreature)} {
			card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AtNext(EachEndStep, action(ref)))))
			if ValidateCardDefinition(card.Name(), card) == nil {
				t.Fatal("accepted undeclared subject")
			}
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred attachment or future body source")
	}
}
