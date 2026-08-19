package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func tapThenFightPipeline() Effect {
	return Pipeline("tap those creatures; they fight each other",
		EffectProperties{Outcome: OutcomeDetriment, Taps: true},
		SnapshotTarget(0, "first"),
		SnapshotTarget(1, "second"),
		TapGathered("first"),
		TapGathered("second"),
		FightGathered("first", "second"),
	)
}

func TestTapThenFightPipeline_TapsAndFightsTwoTargets(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := NewCreature("First", "{1}", 2, 3)
	first.SetOwner(a.PlayerID())
	firstPerm := g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "{1}", 3, 4)
	second.SetOwner(b.PlayerID())
	secondPerm := g.PutOnBattlefield(second, b.PlayerID())

	err := ApplyEffect(g, tapThenFightPipeline(), uuid.New(), a.PlayerID(), []uuid.UUID{firstPerm.ID(), secondPerm.ID()})
	if err != nil {
		t.Fatalf("ApplyEffect: %v", err)
	}

	if !firstPerm.Tapped || !secondPerm.Tapped {
		t.Fatalf("tapped = (%v, %v), want both true", firstPerm.Tapped, secondPerm.Tapped)
	}
	if firstPerm.Damage != 3 || secondPerm.Damage != 2 {
		t.Fatalf("damage = (%d, %d), want (3, 2)", firstPerm.Damage, secondPerm.Damage)
	}
}

func TestTapThenFightPipeline_UsesNormalProtectionAndPrevention(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := NewCreature("Red Fighter", "{R}", 3, 4)
	first.SetOwner(a.PlayerID())
	firstPerm := g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Protected Fighter", "{G}", 2, 4,
		WithAbility(ProtectionFromColor(Red)))
	second.SetOwner(b.PlayerID())
	secondPerm := g.PutOnBattlefield(second, b.PlayerID())
	g.AddPreventionShield(firstPerm.ID(), 1)

	err := ApplyEffect(g, tapThenFightPipeline(), uuid.New(), a.PlayerID(), []uuid.UUID{firstPerm.ID(), secondPerm.ID()})
	if err != nil {
		t.Fatalf("ApplyEffect: %v", err)
	}

	if firstPerm.Damage != 1 {
		t.Fatalf("first fighter damage = %d, want 1 after prevention", firstPerm.Damage)
	}
	if secondPerm.Damage != 0 {
		t.Fatalf("protected fighter damage = %d, want 0", secondPerm.Damage)
	}
}

func TestTapThenFightPipeline_OneIllegalTapsLegalTargetWithoutFight(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	first := NewCreature("First", "{1}", 2, 3)
	first.SetOwner(a.PlayerID())
	firstPerm := g.PutOnBattlefield(first, a.PlayerID())

	err := ApplyEffect(g, tapThenFightPipeline(), uuid.New(), a.PlayerID(), []uuid.UUID{firstPerm.ID(), uuid.Nil})
	if err != nil {
		t.Fatalf("ApplyEffect: %v", err)
	}

	if !firstPerm.Tapped {
		t.Fatal("legal target was not tapped")
	}
	if firstPerm.Damage != 0 {
		t.Fatalf("legal target damage = %d, want no fight damage", firstPerm.Damage)
	}
}

func TestFightGathered_NoncreatureDoesNotFight(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := NewCreature("First", "{1}", 2, 3)
	first.SetOwner(a.PlayerID())
	firstPerm := g.PutOnBattlefield(first, a.PlayerID())
	second := NewArtifact("No Longer a Creature", "{1}")
	second.SetOwner(b.PlayerID())
	secondPerm := g.PutOnBattlefield(second, b.PlayerID())

	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), Vars: map[string]any{
		"first":  firstPerm.ID(),
		"second": secondPerm.ID(),
	}}
	err := FightGathered("first", "second").Apply(ctx)
	if err != nil {
		t.Fatalf("ApplyEffect: %v", err)
	}

	if secondPerm.Tapped {
		t.Fatal("fight effect tapped a permanent")
	}
	if firstPerm.Damage != 0 {
		t.Fatalf("legal target damage = %d, want no fight damage", firstPerm.Damage)
	}
}
