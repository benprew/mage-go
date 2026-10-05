package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatSourcePreventionThisTurn(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Source", "", 3, 8), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 8), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 8), b.PlayerID())
			captured := g.CaptureDamageSource(p.ID())
			if err := PreventCombatDamageByThisTurn(This).Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.effects.RemoveEndOfCombat()
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayerFromSource(b, 2, captured)
			g.DealDamageToPermanentFromSource(victim, 2, captured)
			g.DealDamageToPlayer(b, 1, p.ID())
			g.DealDamageToPlayer(b, 1, other.ID())
			want := 19
			if change == "return" {
				want = 18
			}
			if b.Life() != want || victim.Damage != 0 {
				t.Fatalf("life=%d damage=%d", b.Life(), victim.Damage)
			}
			g.damage.SetResolvingCombatDamage(false)
			g.DealDamageToPlayerFromSource(b, 1, captured)
			if b.Life() != want-1 {
				t.Fatal("noncombat damage prevented")
			}
			clone := g.Clone()
			clone.RunStepWithPriority(core.Cleanup)
			clone.damage.SetResolvingCombatDamage(true)
			clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, captured)
			if clone.GetPlayer(b.PlayerID()).Life() != want-2 {
				t.Fatal("cleanup did not expire prevention")
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayerFromSource(b, 1, captured)
			if b.Life() != want-1 {
				t.Fatal("clone cleanup changed original")
			}
		})
	}
}

func TestCombatSourcePreventionThisTurnDeclarations(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewInstant("Valid", "", Spell(PreventCombatDamageByThisTurn(target), WithTargets(target)))},
		{Name: "nil", Card: NewInstant("Bad", "", Spell(PreventCombatDamageByThisTurn(nil))), WantErr: true},
		{Name: "unbound", Card: NewInstant("Bad", "", Spell(PreventCombatDamageByThisTurn(target))), WantErr: true},
	})
}
