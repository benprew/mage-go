package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestAnnouncedTargetCountKeepsIllegalTargetsAndCopyFrame(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareUpToXPermanentTargets("creatures", IsCreature)
	spell := NewInstant("Count targets", "{X}", WithAbility(Spell(
		GainLifeFor(You, AnnouncedTargetCount(targets)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), b.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(0, 2)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.PhaseOut(second.ID())
	clone.ResolveStackObject(copyObj)
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("copy life = %d, want 22", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	g.PutOnBattlefield(second.Card, b.PlayerID())
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("original life = %d, want 22", got)
	}
}

func TestAnnouncedTargetCountZeroAndRegistration(t *testing.T) {
	targets := DeclareUpToXPermanentTargets("creatures", IsCreature)
	value := AnnouncedTargetCount(targets)
	if got := value.resolveValue(&EffectContext{}); got != 0 {
		t.Fatalf("absent group count = %d, want 0", got)
	}
	good := NewInstant("Good", "{X}", WithAbility(Spell(GainLifeFor(You, value), WithTargets(targets))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Card{
		NewInstant("Undeclared", "", WithAbility(Spell(GainLifeFor(You, value)))),
		NewInstant("Nil", "", WithAbility(Spell(GainLifeFor(You, AnnouncedTargetCount((*PermanentTargetsBinding)(nil)))))),
	} {
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatalf("accepted invalid count in %s", bad.Name())
		}
	}
}

func TestAnnouncedTargetCountUsesOneRecordedGroup(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 2, IsCreature)
	second := DeclarePermanentTargets("second", 0, 2, IsCreature)
	obj := &StackObject{
		Targets:     []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		TargetSpecs: []Target{first.targetDeclaration, first.targetDeclaration, second.targetDeclaration},
	}
	defer g.resolution.Begin(obj)()
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if got := AnnouncedTargetCount(first).resolveValue(ctx); got != 2 {
		t.Fatalf("first group count = %d, want 2", got)
	}
	if got := AnnouncedTargetCount(second).resolveValue(ctx); got != 1 {
		t.Fatalf("second group count = %d, want 1", got)
	}
}
