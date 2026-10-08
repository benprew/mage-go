package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedFightOperands(t *testing.T) {
	for name, action := range map[string]func(PermanentRef) ResolvingEffect{"single": func(ref PermanentRef) ResolvingEffect { return Fight(ref, AttachedPermanent) }, "sequence": func(ref PermanentRef) ResolvingEffect {
		return Sequence(Fight(AttachedPermanent, ref), Fight(ref, ref))
	}} {
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
			if !g.combat.IsAttacking(p.ID()) || g.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("query dealt damage or removed attacker")
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

type waitingFightOpaqueRef struct {
	PermanentRef
	reads *int
}

func (r waitingFightOpaqueRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	*r.reads++
	return r.PermanentRef.resolvePermanent(ctx)
}

func TestWaitingDelayedFightBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 10), a.PlayerID())
	reads := 0
	for _, body := range []ResolvingEffect{Fight(AttachedPermanent, AttachedPermanent), AtNext(EachEndStep, Fight(This, This)), Fight(waitingFightOpaqueRef{This, &reads}, AttachedPermanent)} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("inferred host, opaque operand or future body")
	}
	for _, ref := range []PermanentRef{nil, From(BindPermanent("missing")), DeclarePermanentTarget("missing", IsCreature)} {
		for _, body := range []ResolvingEffect{Fight(ref, This), Fight(This, ref)} {
			card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))
			if ValidateCardDefinition(card.Name(), card) == nil {
				t.Fatal("accepted invalid fight operand")
			}
		}
	}
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("first", Fight(This, AttachedPermanent)), ModeOf("second", Fight(AttachedPermanent, This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || g.FindPermanent(p.ID()).Damage != 0 {
			t.Fatal("modal query executed body or lost source")
		}
	}
}
