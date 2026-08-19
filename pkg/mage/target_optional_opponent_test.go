package mage

import "testing"

func TestTargetUpToNCreaturesOpponentControls(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	source := NewSorcery("Source", "{1}", NewSpellAbility())
	source.SetOwner(a.PlayerID())
	mine := NewCreature("Mine", "{1}", 1, 1)
	mine.SetOwner(a.PlayerID())
	g.PutOnBattlefield(mine, a.PlayerID())
	theirs := NewCreature("Theirs", "{1}", 1, 1)
	theirs.SetOwner(b.PlayerID())
	theirsPerm := g.PutOnBattlefield(theirs, b.PlayerID())

	target := TargetUpToNCreaturesOpponentControls(1)
	if target.Min() != 0 || target.Max() != 1 {
		t.Fatalf("bounds = %d..%d, want 0..1", target.Min(), target.Max())
	}
	possible := target.Possible(a.PlayerID(), source, g)
	if len(possible) != 1 || possible[0] != theirsPerm.ID() {
		t.Fatalf("possible = %v, want only opponent creature %v", possible, theirsPerm.ID())
	}
}
