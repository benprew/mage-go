package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatPreventionToAndByCapturedSource(t *testing.T) {
	for _, change := range []string{"depart", "return", "phase", "clear-LKI"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 3, 10, WithKeyword(Lifelink))
			p := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 10), b.PlayerID())
			old := g.CaptureDamageSource(p.ID())
			g.PreventCombatDamageToAndBy(p.ID())
			if change == "phase" {
				g.PhaseOut(p.ID())
			} else {
				g.RemoveFromBattlefield(p)
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
				if change == "clear-LKI" {
					g.zones.ClearLKI()
				}
			}
			clone := g.Clone()
			clone.damage.SetResolvingCombatDamage(true)
			clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 2, old)
			clone.DealDamageToPermanentFromSource(clone.FindPermanent(victim.ID()), 2, old)
			if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatal("captured combat source lost prevention after departure")
			}
			if change == "return" {
				clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 1, p.ID())
				clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 1, victim.ID())
				if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.FindPermanent(p.ID()).Damage != 1 {
					t.Fatal("returned permanent inherited prevention")
				}
			}
			before := clone.GetPlayer(b.PlayerID()).Life()
			clone.damage.SetResolvingCombatDamage(false)
			clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, old)
			if clone.GetPlayer(b.PlayerID()).Life() != before-1 {
				t.Fatal("noncombat damage was prevented")
			}
			clone.effects.ClearReplacementsEndOfTurn()
			clone.damage.SetResolvingCombatDamage(true)
			clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, old)
			if clone.GetPlayer(b.PlayerID()).Life() != before-2 {
				t.Fatal("prevention survived cleanup")
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayerFromSource(b, 1, old)
			if b.Life() != 20 {
				t.Fatal("clone cleanup changed original prevention")
			}
		})
	}
}

func TestCombatPreventionToAndByRejectsOldSourceForNewShield(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 3, 10)
	p := g.PutOnBattlefield(card, a.PlayerID())
	old := g.CaptureDamageSource(p.ID())
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.PreventCombatDamageToAndBy(returned.ID())
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayerFromSource(b, 2, old)
	if b.Life() != 18 {
		t.Fatal("new shield prevented damage from old incarnation")
	}
	g.DealDamageToPlayer(b, 1, returned.ID())
	if b.Life() != 18 {
		t.Fatal("new shield did not prevent its source")
	}
}

func TestCombatPreventionToAndByDepartedSubjectAtInstallation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 3, 10)
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := PreventCombatDamageToAndBy(This).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 1, returned.ID())
	if b.Life() != 19 {
		t.Fatal("departed subject installed shield on returned incarnation")
	}
}

func TestCombatPreventionToAndBySimultaneousPackets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 3, 10, WithKeyword(Lifelink)), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), b.PlayerID())
	subjectSource, otherSource := g.CaptureDamageSource(subject.ID()), g.CaptureDamageSource(other.ID())
	g.PreventCombatDamageToAndBy(subject.ID())
	g.damage.SetResolvingCombatDamage(true)
	outcomes := g.damage.dealSimultaneousDamageOutcomes(g, []damageRequest{
		{playerID: b.PlayerID(), sourceID: subject.ID(), source: subjectSource, amount: 3},
		{permanent: other, incarnation: other.incarnationID, sourceID: subject.ID(), source: subjectSource, amount: 3},
		{permanent: subject, incarnation: subject.incarnationID, sourceID: other.ID(), source: otherSource, amount: 2},
		{playerID: a.PlayerID(), sourceID: other.ID(), source: otherSource, amount: 2},
	})
	if len(outcomes) != 1 || outcomes[0].playerID != a.PlayerID() || a.Life() != 18 || b.Life() != 20 || subject.Damage != 0 || other.Damage != 0 {
		t.Fatal("batch did not isolate protected source and recipient")
	}
}
