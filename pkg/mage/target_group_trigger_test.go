package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTriggerOptionalTargetGroupsKeepAnnouncementBoundary(t *testing.T) {
	for _, modal := range []bool{false, true} {
		g, a, b := newTriggerTargetGame()
		first := DeclarePermanentTargets("first", 0, 1, IsCreature)
		second := DeclarePermanentTargets("second", 0, 1, IsCreature)
		member := BindPermanent("member")
		body := Sequence(DestroyEach(first), ForEach(second, member, TapSubject(From(member))))
		var trigger TriggeredAbility
		if modal {
			trigger = WhenEventModes(Enters(This),
				ModeOf("choose", body, WithTargets(first, second)),
				ModeOf("other", GainLifeFor(You, Fixed(1))))
		} else {
			trigger = WhenEvent(Enters(This), body, WithTargets(first, second))
		}
		source := NewCreature("Watcher", "", 1, 1, WithAbility(trigger))
		source.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatalf("modal=%v: %v", modal, err)
		}
		victim := NewCreature("Victim", "", 2, 2)
		victim.SetOwner(b.PlayerID())
		g.PutOnBattlefield(victim, b.PlayerID())
		a.chooseQueue = [][]uuid.UUID{{}, {victim.ID()}}
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutTriggersOnStack()
		obj := g.StackPeek()
		if obj == nil || len(obj.Targets) != 1 || obj.Targets[0] != victim.ID() || len(obj.TargetSpecs) != 1 || obj.TargetSpecs[0] != second.targetDeclaration {
			t.Fatalf("modal=%v: targets or group = %+v", modal, obj)
		}
		clone := g.Clone()
		g.ResolveStack()
		if p := g.FindPermanent(victim.ID()); p == nil || !p.Tapped {
			t.Fatalf("modal=%v: second target did not receive its effect", modal)
		}
		if p := clone.FindPermanent(victim.ID()); p == nil || p.Tapped {
			t.Fatal("resolution changed the cloned game")
		}
	}
}

func TestTriggerTargetGroupsCopyAndReturnedIncarnation(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	member := BindPermanent("member")
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This),
		Sequence(DestroyEach(first), ForEach(second, member, TapSubject(From(member)))), WithTargets(first, second))))
	source.SetOwner(a.PlayerID())
	old := NewCreature("Old", "", 2, 2)
	old.SetOwner(b.PlayerID())
	replacement := NewCreature("Replacement", "", 2, 2)
	replacement.SetOwner(b.PlayerID())
	g.PutOnBattlefield(old, b.PlayerID())
	g.PutOnBattlefield(replacement, b.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{}, {old.ID()}}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutTriggersOnStack()
	original := g.StackPeek()
	clone := g.Clone()
	a.chooseQueue = [][]uuid.UUID{{replacement.ID()}}
	copyObj := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copyObj == nil || len(copyObj.TargetSpecs) != 1 || copyObj.TargetSpecs[0] != second.targetDeclaration || copyObj.Targets[0] != replacement.ID() {
		t.Fatal("stack copy lost the second target group")
	}
	if original.Targets[0] != old.ID() || clone.StackPeek().Targets[0] != old.ID() {
		t.Fatal("copy changed the original or game clone")
	}
	g.RemoveFromBattlefield(g.FindPermanent(replacement.ID()))
	g.PutOnBattlefield(replacement, b.PlayerID())
	g.ResolveStackObject(copyObj)
	if p := g.FindPermanent(replacement.ID()); p == nil || p.Tapped {
		t.Fatal("returned target received the copied effect")
	}
	g.ResolveStackObject(original)
	if p := g.FindPermanent(old.ID()); p == nil || !p.Tapped {
		t.Fatal("original lost its independent target")
	}
}

func TestTriggerTargetGroupsRejectInvalidChoice(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This),
		DestroyEach(first), WithTargets(first, second))))
	source.SetOwner(a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{}, {uuid.New()}}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("invalid group choice put a trigger on the stack")
	}
}

func TestTriggerTargetGroupsAllowSamePermanentInSeparateGroups(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This),
		Sequence(DestroyEach(first), DestroyEach(second)), WithTargets(first, second))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{victim.ID()}, {victim.ID()}}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutTriggersOnStack()
	obj := g.StackPeek()
	if obj == nil || len(obj.Targets) != 2 || obj.Targets[0] != victim.ID() || obj.Targets[1] != victim.ID() || len(obj.TargetSpecs) != 2 || obj.TargetSpecs[0] != first.targetDeclaration || obj.TargetSpecs[1] != second.targetDeclaration {
		t.Fatal("one legal target in each group was rejected or merged")
	}
}

func TestFlatActionInputStillRejectsTwoVariableGroups(t *testing.T) {
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	spell := NewInstant("Spell", "", WithAbility(Spell(DestroyEach(first), WithTargets(first, second))))
	activation := NewArtifact("Ability", "", WithAbility(Activate(Costs(), DestroyEach(first), WithTargets(first, second))))
	for _, card := range []Card{spell, activation} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("%s accepted two variable groups with flat announcement", card.Name())
		}
	}
}
