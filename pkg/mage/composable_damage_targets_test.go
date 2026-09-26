package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageTargetGroupMixedBatchAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Mixed damage", "", WithAbility(Spell(
		DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 3, 4), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 3, 4), b.PlayerID())
	a.AddToHand(spell)
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID(), creature.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.RemoveFromBattlefield(clone.FindPermanent(creature.ID()))
	clone.PutOnBattlefield(creature.Card, b.PlayerID())
	clone.ResolveStackObject(copyObj)
	if clone.GetPlayer(b.PlayerID()).Life() != 18 || clone.FindPermanent(creature.ID()).Damage != 0 || clone.FindPermanent(other.ID()).Damage != 0 {
		t.Fatal("copy followed a returned target or damaged an unselected one")
	}
	if b.Life() != 20 || creature.Damage != 0 || len(original.Targets) != 2 {
		t.Fatal("clone changed the original stack or battlefield")
	}
	g.ResolveStackObject(original)
	if b.Life() != 18 || g.FindPermanent(creature.ID()).Damage != 2 || g.FindPermanent(other.ID()).Damage != 0 {
		t.Fatalf("mixed batch: life=%d creature=%d other=%d", b.Life(), g.FindPermanent(creature.ID()).Damage, g.FindPermanent(other.ID()).Damage)
	}
}

func TestDamageTargetGroupAnnouncementAndValidation(t *testing.T) {
	for _, count := range [][2]int{{-1, 2}, {2, 1}, {0, 0}} {
		targets := DeclareDamageTargets("invalid", count[0], count[1])
		card := NewInstant("Invalid", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(1)), WithTargets(targets))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted counts %v", count)
		}
	}
	targets := DeclareDamageTargets("recipients", 1, 2)
	missing := NewInstant("Missing", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(1)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "target binding requires") {
		t.Fatalf("undeclared group = %v", err)
	}
	g, a, b := newSpellCopyGame()
	card := NewInstant("Validated", "{1}", WithAbility(Spell(DealDamageEach(This, targets, Fixed(1)), WithTargets(targets))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(0, 1)
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
	for _, chosen := range [][]uuid.UUID{nil, {b.PlayerID(), b.PlayerID()}, {artifact.ID()}, {uuid.New()}, {creature.ID(), artifact.ID()}} {
		before := a.ManaPool().TotalMana()
		if err := g.CastSpellByName(a.PlayerID(), card.Name(), chosen); err == nil {
			t.Fatalf("accepted invalid targets %v", chosen)
		}
		if a.ManaPool().TotalMana() != before || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("failed announcement changed payment or stack")
		}
	}
}

func TestDamageTargetGroupPreventionAndPhasing(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Batch", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 3, 4), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID(), creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.AddPreventionShield(b.PlayerID(), 1)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if b.Life() != 19 || g.FindPermanent(creature.ID()).Damage != 2 {
			t.Errorf("damage callback saw life=%d damage=%d", b.Life(), g.FindPermanent(creature.ID()).Damage)
		}
	})
	g.ResolveStack()
	if observed != 2 || b.Life() != 19 || g.FindPermanent(creature.ID()).Damage != 2 {
		t.Fatal("prevention changed the wrong recipient or broke the batch")
	}
	g.damage.SetOnDamageDealt(nil)

	second := NewInstant("Second", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	if err := g.CastSpellByName(a.PlayerID(), second.Name(), []uuid.UUID{b.PlayerID(), creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(creature.ID())
	g.ResolveStack()
	if b.Life() != 17 || g.FindPermanentIncludingPhased(creature.ID()).Damage != 2 {
		t.Fatal("phased recipient received damage or legal player was lost")
	}
}

func TestDamageTargetGroupPlaneswalker(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Batch", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{a.PlayerID(), walker.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 18 || g.FindPermanent(walker.ID()).Counters[core.Loyalty] != 3 {
		t.Fatal("player or planeswalker received incorrect damage")
	}
}
