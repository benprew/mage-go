package mage

import "testing"

func TestCanBlockOnlyClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2, WithStaticAbility(SourceCanBlockOnly(Not(HasPowerGTE(2))))), a.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 5), b.PlayerID())
	if CanBlock(blocker, attacker, g) {
		t.Fatal("source restriction absent")
	}
	clone := g.Clone()
	if CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(attacker.ID()), clone) {
		t.Fatal("clone lost restriction before layer pass")
	}
	clone.effects.canBlockOnlyRules[blocker.ID()][0] = AnyPermanent
	if CanBlock(blocker, attacker, g) {
		t.Fatal("clone shared filter slice")
	}
	clone.effects.resetCombatRestrictions()
	if CanBlock(blocker, attacker, g) {
		t.Fatal("clone reset changed original")
	}
	clone.ApplyContinuousEffects()
	if CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(attacker.ID()), clone) {
		t.Fatal("clone did not rebuild source restriction")
	}
	ids := g.PhaseOut(blocker.ID())
	g.ApplyContinuousEffects()
	if len(g.effects.canBlockOnlyRules[blocker.ID()]) != 0 {
		t.Fatal("phased source still supplies a restriction")
	}
	if CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(attacker.ID()), clone) {
		t.Fatal("phasing changed clone restriction")
	}
	g.PhaseIn(ids)
	g.ApplyContinuousEffects()
	if CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(attacker.ID()), g) {
		t.Fatal("phase cycle lost source restriction")
	}
}
