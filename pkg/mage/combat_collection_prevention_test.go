package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatCollectionPreventionTurnDuration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 10), b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
	effect := PreventCombatDamageByEachThisTurn(BlockersOf(This))
	if err := ApplyEffect(g, effect, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.effects.RemoveEndOfCombat()
	g.combat.Reset()
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(a, 2, blocker.ID())
	if a.Life() != 20 {
		t.Fatal("turn prevention ended with combat or followed current membership")
	}
	clone := g.Clone()
	clone.RunStepWithPriority(core.Cleanup)
	clone.damage.SetResolvingCombatDamage(true)
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, blocker.ID())
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || a.Life() != 20 {
		t.Fatal("cleanup or clone isolation")
	}
	g.damage.SetResolvingCombatDamage(false)
	g.DealDamageToPlayer(a, 1, blocker.ID())
	if a.Life() != 19 {
		t.Fatal("noncombat damage prevented")
	}
}

func TestCombatCollectionPreventionTurnDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "nil set", Card: NewInstant("Bad", "", Spell(PreventCombatDamageByEachThisTurn(nil))), WantErr: true},
		{Name: "unbound", Card: NewInstant("Bad", "", Spell(PreventCombatDamageByEachThisTurn(BlockersOf(target)))), WantErr: true},
		{Name: "bound", Card: NewInstant("Good", "", Spell(PreventCombatDamageByEachThisTurn(BlockersOf(target)), WithTargets(target)))},
	})
}

func TestCombatCollectionPreventionTurnIdentity(t *testing.T) {
	for _, change := range []string{"return", "phase", "control", "leave-pair"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), a.PlayerID())
			card := NewCreature("Blocker", "", 2, 10)
			blocker := g.PutOnBattlefield(card, b.PlayerID())
			g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
			g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
			if err := ApplyEffect(g, PreventCombatDamageByEachThisTurn(BlockersOf(This)), attacker.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			captured := g.CaptureDamageSource(blocker.ID())
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseIn(g.PhaseOut(blocker.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: attacker.ID(), TargetID: blocker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "leave-pair":
				g.RemoveFromCombat(blocker.ID())
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayerFromSource(a, 2, captured)
			if a.Life() != 20 {
				t.Fatal("captured source lost prevention")
			}
			g.DealDamageToPlayer(a, 1, blocker.ID())
			want := 20
			if change == "return" {
				want = 19
			}
			if a.Life() != want {
				t.Fatal("prevention followed another incarnation")
			}
		})
	}
}

func TestCombatCollectionPreventionTurnDeduplicates(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 10), a.PlayerID())
	set := fixedTapSet{permanents: []*Permanent{p, p}}
	if err := ApplyEffect(g, PreventCombatDamageByEachThisTurn(set), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.replacements) != 1 {
		t.Fatal("duplicate source rules")
	}
}
