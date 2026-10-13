package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterPointPreventionUnpreventable(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, amount := range []int{1, 5, 300, 0, -1} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 1000, WithStaticAbility(PreventDamageToSourceByRemovingCounters(core.Charge))), a.PlayerID())
			p.AddCounter(core.Charge, 3)
			p.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			g.AddPreventionShield(p.ID(), 2)
			clone := g.Clone()
			var actualDamage int
			if batch {
				g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(p.ID()), amount, g.CaptureDamageSource(uuid.New())))
				actualDamage = g.FindPermanent(p.ID()).Damage
			} else {
				result := g.effects.ApplyReplacements(NewDamageToCreatureAction(uuid.New(), p.ID(), amount, false), g)
				actualDamage = damageActionAmount(result)
			}
			got := g.FindPermanent(p.ID())
			if got.Counters[core.Charge] != 3-min(3, max(0, amount)) || actualDamage != max(0, amount) {
				t.Fatalf("batch=%v amount=%d counters=%d damage=%d", batch, amount, got.Counters[core.Charge], actualDamage)
			}
			if clone.FindPermanent(p.ID()).Counters[core.Charge] != 3 || clone.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("clone shared counters or damage")
			}
			got.RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			got.RemoveCounter(core.Charge, got.Counters[core.Charge])
			before := got.Damage
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 2, uuid.New())
			if g.FindPermanent(p.ID()).Damage != before {
				t.Fatal("unpreventable damage consumed the finite shield")
			}
		}
	}
}
