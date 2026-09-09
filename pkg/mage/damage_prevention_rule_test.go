package mage

import "testing"

func TestDamagePreventionRuleCanProtectOnePlayer(t *testing.T) {
	protected := NewBasePlayer("protected")
	opponent := NewBasePlayer("opponent")
	g := NewGame(protected, opponent)
	source := NewCreature("damage source", "{1}{R}", 2, 2)
	source.SetOwner(opponent.PlayerID())
	perm := g.PutOnBattlefield(source, opponent.PlayerID())

	g.AddDamagePreventionRule(WithFrom(IsID(perm.ID())), WithToPlayer(protected.PlayerID()))
	g.DealDamageToPlayer(opponent, 2, perm.ID())
	g.DealDamageToPlayer(protected, 2, perm.ID())

	if got := opponent.Life(); got != 18 {
		t.Errorf("unprotected player life = %d, want 18", got)
	}
	if got := protected.Life(); got != 20 {
		t.Errorf("protected player life = %d, want 20", got)
	}
}
