package mage

import "testing"

func TestReverseDamageReplacementMatchesChosenSource(t *testing.T) {
	protected := NewBasePlayer("protected")
	opponent := NewBasePlayer("opponent")
	g := NewGame(protected, opponent)
	chosen := g.PutOnBattlefield(NewCreature("chosen", "{1}{R}", 2, 2), opponent.PlayerID())
	unchosen := g.PutOnBattlefield(NewCreature("unchosen", "{1}{R}", 2, 2), opponent.PlayerID())

	g.AddReverseDamageShield(protected.PlayerID(), chosen.ID())
	g.DealDamageToPlayer(protected, 2, unchosen.ID())
	g.DealDamageToPlayer(protected, 3, chosen.ID())

	if got := protected.Life(); got != 21 {
		t.Fatalf("life = %d, want 21", got)
	}
}
