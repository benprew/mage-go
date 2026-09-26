package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationTargetGroupsKeepEmptyFirstGroup(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	member := BindPermanent("member")
	source := g.PutOnBattlefield(NewArtifact("Grouped", "", WithAbility(Activate(Costs(TapCost(This)),
		Sequence(DestroyEach(first), ForEach(second, member, TapSubject(From(member)))),
		WithActivationTargetGroups(first, second)))), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source.Card); err != nil {
		t.Fatal(err)
	}
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
	if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0, [][]uuid.UUID{{}, {victim.ID()}}); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	if obj == nil || len(obj.TargetSpecs) != 1 || obj.TargetSpecs[0] != second.targetDeclaration || !source.Tapped {
		t.Fatal("activation lost target group or payment")
	}
	clone := g.Clone()
	g.ResolveStack()
	if p := g.FindPermanent(victim.ID()); p == nil || !p.Tapped {
		t.Fatal("second target group did not resolve")
	}
	if p := clone.FindPermanent(victim.ID()); p == nil || p.Tapped {
		t.Fatal("resolution changed clone")
	}
}

func TestActivationTargetGroupsRejectBeforePayment(t *testing.T) {
	for _, groups := range [][][]uuid.UUID{{{}}, {{}, {uuid.New()}}, {{uuid.New(), uuid.New()}, {}}} {
		g, a, b := newSpellCopyGame()
		first := DeclarePermanentTargets("first", 0, 1, IsCreature)
		second := DeclarePermanentTargets("second", 0, 1, IsCreature)
		source := g.PutOnBattlefield(NewArtifact("Grouped", "", WithAbility(Activate(Costs(TapCost(This)),
			DestroyEach(first), WithActivationTargetGroups(first, second)))), a.PlayerID())
		g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
		if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0, groups); err == nil {
			t.Fatal("accepted invalid target groups")
		}
		if source.Tapped || g.StackSize() != 0 {
			t.Fatal("invalid announcement paid activation cost")
		}
	}
}

func TestActivationTargetGroupsRequireGroupedEntry(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	source := g.PutOnBattlefield(NewArtifact("Grouped", "", WithAbility(Activate(Costs(TapCost(This)),
		DestroyEach(first), WithActivationTargetGroups(first, second)))), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("flat activation accepted grouped declaration")
	}
	if source.Tapped || g.StackSize() != 0 {
		t.Fatal("flat activation changed game state")
	}
	bad := NewArtifact("Empty", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), WithActivationTargetGroups())))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("accepted empty group declaration")
	}
}

func TestActivationTargetGroupsCopyAndReturnedTarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	member := BindPermanent("member")
	source := g.PutOnBattlefield(NewArtifact("Grouped", "", WithAbility(Activate(Costs(),
		ForEach(second, member, TapSubject(From(member))),
		WithActivationTargetGroups(first, second)))), a.PlayerID())
	old := g.PutOnBattlefield(NewCreature("Old", "", 2, 2), b.PlayerID())
	replacement := g.PutOnBattlefield(NewCreature("Replacement", "", 2, 2), b.PlayerID())
	if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0, [][]uuid.UUID{{}, {old.ID()}}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.chooseQueue = [][]uuid.UUID{{replacement.ID()}}
	copyObj := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copyObj == nil || len(copyObj.TargetSpecs) != 1 || copyObj.TargetSpecs[0] != second.targetDeclaration {
		t.Fatal("copy moved the second group into the first")
	}
	g.ResolveStackObject(copyObj)
	if p := g.FindPermanent(replacement.ID()); p == nil || !p.Tapped {
		t.Fatal("copy did not apply the second group")
	}
	g.RemoveFromBattlefield(g.FindPermanent(old.ID()))
	g.PutOnBattlefield(old.Card, b.PlayerID())
	g.ResolveStackObject(original)
	if p := g.FindPermanent(old.ID()); p == nil || p.Tapped {
		t.Fatal("returned target received the original effect")
	}
}

func TestActivationTargetGroupsPaymentFailureRollsBack(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	source := g.PutOnBattlefield(NewArtifact("Unpayable", "", WithAbility(Activate(
		Costs(LifePayCost(100), TapCost(This)), DestroyEach(first),
		WithActivationTargetGroups(first, second)))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	life := a.Life()
	if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0, [][]uuid.UUID{{}, {creature.ID()}}); err == nil {
		t.Fatal("accepted unpayable grouped activation")
	}
	if source.Tapped || a.Life() != life || g.StackSize() != 0 {
		t.Fatal("failed payment changed live state")
	}
}

func TestActivationTargetGroupsDuplicateChoiceBoundary(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 2, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	source := g.PutOnBattlefield(NewArtifact("Grouped", "", WithAbility(Activate(Costs(TapCost(This)),
		GainLifeFor(You, Fixed(1)), WithActivationTargetGroups(first, second)))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0,
		[][]uuid.UUID{{creature.ID(), creature.ID()}, {}}); err == nil {
		t.Fatal("one target group accepted the same object twice")
	}
	if source.Tapped || g.StackSize() != 0 {
		t.Fatal("invalid duplicate paid the cost")
	}
	if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0,
		[][]uuid.UUID{{creature.ID()}, {creature.ID()}}); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	if len(obj.TargetSpecs) != 2 || obj.TargetSpecs[0] != first.targetDeclaration || obj.TargetSpecs[1] != second.targetDeclaration {
		t.Fatal("separate target words lost their declarations")
	}
}

func TestActivationTargetGroupsRejectManaAbility(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	land := g.PutOnBattlefield(NewLand("Source", WithManaAbility(core.Green)), a.PlayerID())
	if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), land.ID(), 0, [][]uuid.UUID{{}}); err == nil {
		t.Fatal("accepted grouped announcement for a mana ability")
	}
	if land.Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("invalid grouped announcement produced mana")
	}
}
