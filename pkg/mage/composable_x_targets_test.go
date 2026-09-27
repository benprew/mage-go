package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTypedXTargetsSpellAnnouncementAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXPermanentTargets("creatures", IsCreature)
	member := BindPermanent("member")
	spell := NewInstant("X tap", "{X}", WithAbility(Spell(ForEach(targets, member, TapSubject(From(member))), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), b.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	for _, chosen := range [][]uuid.UUID{{first.ID()}, {first.ID(), first.ID()}, {first.ID(), second.ID(), uuid.New()}} {
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), chosen, 2); err == nil {
			t.Fatalf("accepted invalid X targets %v", chosen)
		}
		if g.StackSize() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 0 {
			t.Fatal("invalid announcement changed state")
		}
	}
	a.ManaPool().Add(0, 4)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.PhaseOut(second.ID())
	clone.ResolveStackObject(copyObj)
	if !clone.FindPermanent(first.ID()).Tapped || clone.FindPermanentIncludingPhased(second.ID()).Tapped {
		t.Fatal("copy lost legal target or affected phased target")
	}
	if first.Tapped || second.Tapped || len(original.TargetSpecs) != 2 {
		t.Fatal("clone changed original or lost target count")
	}
	g.ResolveStackObject(original)
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("X targets did not resolve")
	}
}

func TestTypedXTargetsZeroAndReturnedTarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXPermanentTargets("creatures", IsCreature)
	member := BindPermanent("member")
	spell := NewInstant("X tap", "{X}", WithAbility(Spell(ForEach(targets, member, TapSubject(From(member))), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	second := NewInstant("X tap", "{X}", WithAbility(Spell(ForEach(targets, member, TapSubject(From(member))), WithTargets(targets))))
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	old := g.PutOnBattlefield(NewCreature("Old", "", 2, 2), b.PlayerID())
	a.ManaPool().Add(0, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{old.ID()}, 1); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(old.ID()))
	g.PutOnBattlefield(old.Card, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(old.ID()).Tapped {
		t.Fatal("target followed a returned incarnation")
	}
}

func TestTypedXTargetsActivationAndUpToX(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareUpToXPermanentTargets("lands", IsLand)
	member := BindPermanent("member")
	source := g.PutOnBattlefield(NewArtifact("X untap", "", WithAbility(Activate(Costs(ManaPayment("{X}"), TapCost(This)),
		ForEach(targets, member, UntapPermanent(From(member))), WithTargets(targets)))), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source.Card); err != nil {
		t.Fatal(err)
	}
	land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	g.TapPermanent(land)
	g.SetXValue(1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID(), uuid.New()}); err == nil {
		t.Fatal("accepted more than X targets")
	}
	if source.Tapped || !land.Tapped || g.StackSize() != 0 {
		t.Fatal("invalid activation changed state")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err == nil {
		t.Fatal("accepted unpaid X activation")
	}
	if source.Tapped || !land.Tapped || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(0, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if land.Tapped || !source.Tapped {
		t.Fatal("activation lost announced target")
	}
}

func TestTypedXTargetsDeclaration(t *testing.T) {
	targets := DeclareXPermanentTargets("creatures", IsCreature)
	if err := ValidateCardDefinition("missing", NewSorcery("Missing", "", WithAbility(Spell(DestroyEach(targets))))); err == nil {
		t.Fatal("accepted undeclared X targets")
	}
	unnamed := DeclareXPermanentTargets("", IsCreature)
	if err := ValidateCardDefinition("unnamed", NewSorcery("Unnamed", "", WithAbility(Spell(DestroyEach(unnamed), WithTargets(unnamed))))); err == nil {
		t.Fatal("accepted unnamed X targets")
	}
	first := DeclareUpToXPermanentTargets("first", IsCreature)
	second := DeclareUpToXPermanentTargets("second", IsCreature)
	card := NewInstant("Grouped", "{X}", WithAbility(Spell(DestroyEach(first), WithTargetGroups(first, second))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
