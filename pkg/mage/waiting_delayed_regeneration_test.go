package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedRegenerationOperands(t *testing.T) {
	for name, action := range map[string]func(PermanentRef) ResolvingEffect{"single": Regenerate, "sequence": func(ref PermanentRef) ResolvingEffect { return Sequence(Regenerate(ref), Regenerate(ref)) }} {
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
			if !g.combat.IsAttacking(p.ID()) || len(g.WaitingRegenerationPermanentSources()) != 0 {
				t.Fatal("query created a shield or removed attacker")
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

func TestWaitingDelayedRegenerationBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 10), a.PlayerID())
	for _, action := range []func(PermanentRef) ResolvingEffect{Regenerate} {
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

type waitingRegenerationOpaqueRef struct {
	PermanentRef
	reads *int
}

func (r waitingRegenerationOpaqueRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	*r.reads++
	return r.PermanentRef.resolvePermanent(ctx)
}

func TestWaitingDelayedRegenerationModesAndPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 4), a.PlayerID())
	reads := 0
	body := AtNextModes(EachEndStep, ModeOf("source", Regenerate(This)), ModeOf("opaque", Regenerate(waitingRegenerationOpaqueRef{This, &reads})))
	if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || g.HasRegenerationShield(p.ID()) {
			t.Fatal("mode query resolved a subject or created a shield")
		}
	}
	g.triggers.SetDelayed(nil)
	if err := ApplyEffect(g, AtNext(EachEndStep, Regenerate(waitingRegenerationOpaqueRef{This, &reads})), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("opaque operand inferred a source or resolved")
	}
}
