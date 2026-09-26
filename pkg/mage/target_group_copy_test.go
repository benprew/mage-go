package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestCopyRetargetUsesRecordedOptionalGroupBoundary(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	member := BindPermanent("member")
	ability := Spell(Sequence(DestroyEach(first), ForEach(second, member, TapSubject(From(member)))), WithTargets(first, second))
	spell := NewInstant("Two optional groups", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	old := g.PutOnBattlefield(NewCreature("Old", "", 2, 2), b.PlayerID())
	replacement := g.PutOnBattlefield(NewCreature("Replacement", "", 2, 2), b.PlayerID())
	original := &StackObject{
		ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(),
		Targets: []uuid.UUID{old.ID()}, TargetSpecs: []Target{second.targetDeclaration},
		Effects: ability.Effects(),
	}
	g.pushStack(original)
	clone := g.Clone()
	a.chooseQueue = [][]uuid.UUID{{replacement.ID()}}
	copyObj := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copyObj == nil || len(copyObj.TargetSpecs) != 1 || copyObj.TargetSpecs[0] != second.targetDeclaration {
		t.Fatalf("copy lost the second group boundary: %+v", copyObj)
	}
	if len(copyObj.Targets) != 1 || copyObj.Targets[0] != replacement.ID() {
		t.Fatalf("copy targets = %v, want only replacement", copyObj.Targets)
	}
	if len(original.Targets) != 1 || original.Targets[0] != old.ID() || clone.StackPeek().Targets[0] != old.ID() {
		t.Fatal("retargeting changed the original or cloned game")
	}
	g.ResolveStackObject(copyObj)
	if p := g.FindPermanent(replacement.ID()); p == nil || !p.Tapped {
		t.Fatal("the second group's effect missed its copied target")
	}
	if g.FindPermanent(old.ID()) == nil {
		t.Fatal("the empty first group destroyed the original target")
	}
	clonedOriginal := clone.StackPeek()
	clonedCopy := clone.CopyStackObjectDirect(clonedOriginal, a.PlayerID(), false)
	if clonedCopy == nil || len(clonedCopy.TargetSpecs) != 1 || clonedCopy.TargetSpecs[0] != second.targetDeclaration {
		t.Fatal("game clone or stack copy lost the recorded group")
	}
	clone.RemoveFromBattlefield(clone.FindPermanent(old.ID()))
	clone.PutOnBattlefield(old.Card, b.PlayerID())
	clone.ResolveStackObject(clonedCopy)
	if p := clone.FindPermanent(old.ID()); p == nil || p.Tapped {
		t.Fatal("the copied target reached a returned incarnation")
	}
}
