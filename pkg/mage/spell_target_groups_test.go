package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestSpellTargetGroupsPreserveEmptyFirstGroup(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	member := BindPermanent("member")
	spell := NewInstant("Grouped", "", WithAbility(Spell(Sequence(DestroyEach(first),
		ForEach(second, member, TapSubject(From(member)))), WithTargetGroups(first, second))))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	old := g.PutOnBattlefield(NewCreature("Old", "", 2, 2), b.PlayerID())
	replacement := g.PutOnBattlefield(NewCreature("Replacement", "", 2, 2), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{{}, {old.ID()}}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	if original == nil || len(original.TargetSpecs) != 1 || original.TargetSpecs[0] != second.targetDeclaration {
		t.Fatalf("cast lost second target group: %+v", original)
	}
	clone := g.Clone()
	a.chooseQueue = [][]uuid.UUID{{replacement.ID()}}
	copyObj := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copyObj == nil || len(copyObj.TargetSpecs) != 1 || copyObj.TargetSpecs[0] != second.targetDeclaration {
		t.Fatal("copy lost second target group")
	}
	g.ResolveStackObject(copyObj)
	if p := g.FindPermanent(replacement.ID()); p == nil || !p.Tapped {
		t.Fatal("copy did not route second group")
	}
	if original.Targets[0] != old.ID() || clone.StackPeek().Targets[0] != old.ID() {
		t.Fatal("copy changed original or clone")
	}
	g.RemoveFromBattlefield(g.FindPermanent(old.ID()))
	g.PutOnBattlefield(old.Card, b.PlayerID())
	g.ResolveStackObject(original)
	if p := g.FindPermanent(old.ID()); p == nil || p.Tapped {
		t.Fatal("returned target received the effect")
	}
}

func TestSpellTargetGroupsRejectInvalidAnnouncementBeforePayment(t *testing.T) {
	for _, tc := range []struct {
		name   string
		groups func(uuid.UUID, uuid.UUID) [][]uuid.UUID
	}{
		{"missing group", func(a, b uuid.UUID) [][]uuid.UUID { return [][]uuid.UUID{{a}} }},
		{"too many in group", func(a, b uuid.UUID) [][]uuid.UUID { return [][]uuid.UUID{{}, {a, b}} }},
		{"duplicate within group", func(a, b uuid.UUID) [][]uuid.UUID { return [][]uuid.UUID{{a, a}, {}} }},
		{"illegal target", func(a, b uuid.UUID) [][]uuid.UUID { return [][]uuid.UUID{{}, {uuid.New()}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			first := DeclarePermanentTargets("first", 0, 2, IsCreature)
			second := DeclarePermanentTargets("second", 0, 1, IsCreature)
			spell := NewInstant("Grouped", "{1}", WithAbility(Spell(DestroyEach(first), WithTargetGroups(first, second))))
			spell.SetOwner(a.PlayerID())
			a.ManaPool().Add(0, 1)
			a.AddToHand(spell)
			one := g.PutOnBattlefield(NewCreature("One", "", 2, 2), b.PlayerID())
			two := g.PutOnBattlefield(NewCreature("Two", "", 2, 2), b.PlayerID())
			before := a.ManaPool().TotalMana()
			if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), tc.groups(one.ID(), two.ID())); err == nil {
				t.Fatal("accepted invalid grouped targets")
			}
			if a.ManaPool().TotalMana() != before || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("invalid announcement changed game state")
			}
		})
	}
}

func TestSpellTargetGroupsRequireGroupedCastAndDeclaration(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	spell := NewInstant("Grouped", "", WithAbility(Spell(DestroyEach(first), WithTargetGroups(first, second))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("flat cast accepted grouped declaration")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("flat cast changed game state")
	}
	plain := NewInstant("Plain", "", WithAbility(Spell(DestroyEach(first), WithTargets(first, second))))
	if err := ValidateCardDefinition(plain.Name(), plain); err == nil {
		t.Fatal("flat declaration accepted independent variable groups")
	}
	empty := NewInstant("Empty", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)), WithTargetGroups())))
	if err := ValidateCardDefinition(empty.Name(), empty); err == nil {
		t.Fatal("accepted empty target group declaration")
	}
}

func TestSpellTargetGroupsMixedRequiredAndOptionalTargets(t *testing.T) {
	g, a, b := newSpellCopyGame()
	player := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	member := BindPermanent("member")
	spell := NewInstant("Mixed", "", WithAbility(Spell(Sequence(
		LoseLifeFor(player, Fixed(1)),
		DestroyEach(first),
		ForEach(second, member, TapSubject(From(member))),
	), WithTargetGroups(player, first, second))))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{{b.PlayerID()}, {}, {victim.ID()}}); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	if len(obj.TargetSpecs) != 2 || obj.TargetSpecs[0] != player.targetDeclaration || obj.TargetSpecs[1] != second.targetDeclaration {
		t.Fatalf("mixed group alignment = %v", obj.TargetSpecs)
	}
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatalf("opponent life = %d, want 19", b.Life())
	}
	if p := g.FindPermanent(victim.ID()); p == nil || !p.Tapped {
		t.Fatal("optional second creature group was not applied")
	}
}

func TestSpellTargetGroupsAllowOneObjectInTwoGroups(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	spell := NewInstant("Twice", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)), WithTargetGroups(first, second))))
	spell.SetOwner(a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{{creature.ID()}, {creature.ID()}}); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	if len(obj.TargetSpecs) != 2 || obj.TargetSpecs[0] != first.targetDeclaration || obj.TargetSpecs[1] != second.targetDeclaration {
		t.Fatal("duplicate choice across separate groups lost its declarations")
	}
}

func TestSpellTargetGroupsKeepLegalGroupAfterPhasing(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	member := BindPermanent("member")
	spell := NewInstant("Partial", "", WithAbility(Spell(Sequence(DestroyEach(first),
		ForEach(second, member, TapSubject(From(member)))), WithTargetGroups(first, second))))
	spell.SetOwner(a.PlayerID())
	one := g.PutOnBattlefield(NewCreature("One", "", 2, 2), b.PlayerID())
	two := g.PutOnBattlefield(NewCreature("Two", "", 2, 2), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{{one.ID()}, {two.ID()}}); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(one.ID())
	g.ResolveStack()
	if p := g.FindPermanent(two.ID()); p == nil || !p.Tapped {
		t.Fatal("legal second group did not resolve")
	}
	g.PhaseIn(phased)
	if g.FindPermanent(one.ID()) == nil {
		t.Fatal("phased first target was destroyed")
	}
}

func TestSpellTargetGroupsPaymentFailureRollsBack(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 1, IsCreature)
	second := DeclarePermanentTargets("second", 0, 1, IsCreature)
	spell := NewInstant("Unpayable", "", WithAdditionalCost(LifePayCost(100)),
		WithAbility(Spell(DestroyEach(first), WithTargetGroups(first, second))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	life := a.Life()
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{{}, {creature.ID()}}); err == nil {
		t.Fatal("accepted unpayable grouped spell")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 || a.Life() != life {
		t.Fatal("payment failure changed game state")
	}
}
