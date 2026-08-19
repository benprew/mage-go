package mage

import (
	"slices"
	"testing"
)

func TestTargetSpellOrPermanentIncludesBothZones(t *testing.T) {
	g, a, b := newRandomTargetGame()
	permanent := addRandomTargetCreature(g, b, "Permanent Target")
	spell := NewInstant("Spell Target", "{0}", NewSpellAbility())
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatalf("cast spell target: %v", err)
	}

	possible := TargetSpellOrPermanent().Possible(a.PlayerID(), spell, g)
	if !slices.Contains(possible, permanent.ID()) {
		t.Fatalf("possible targets %v do not include permanent %v", possible, permanent.ID())
	}
	if !slices.Contains(possible, spell.ID()) {
		t.Fatalf("possible targets %v do not include spell %v", possible, spell.ID())
	}
}

func TestTargetSpellOrPermanentExcludesAbilities(t *testing.T) {
	g, a, _ := newRandomTargetGame()
	source := NewArtifact("Ability Source", "{0}", WithActivatedAbility(GainLife(1), GenericCost(0)))
	source.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), permanent.ID(), 0, nil); err != nil {
		t.Fatalf("activate ability: %v", err)
	}

	possible := TargetSpellOrPermanent().Possible(a.PlayerID(), source, g)
	if len(possible) != 1 || possible[0] != permanent.ID() {
		t.Fatalf("possible targets = %v, want only source permanent %v", possible, permanent.ID())
	}
}
