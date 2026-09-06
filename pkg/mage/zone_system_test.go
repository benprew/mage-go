package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestZoneSystem_BattlefieldAndQueries(t *testing.T) {
	zs := NewZoneSystem()
	pA := uuid.New()
	pB := uuid.New()

	card1 := NewCreature("Bear", "1G", 2, 2)
	card1.SetOwner(pA)
	perm1 := NewPermanent(card1, pA)

	card2 := NewCreature("Giant", "3R", 3, 3)
	card2.SetOwner(pB)
	perm2 := NewPermanent(card2, pB)

	zs.AddToBattlefield(perm1, perm2)

	if len(zs.Battlefield()) != 2 {
		t.Fatalf("expected 2 permanents, got %d", len(zs.Battlefield()))
	}
	if zs.FindPermanent(perm1.ID()) != perm1 {
		t.Errorf("FindPermanent failed for perm1")
	}
	if zs.FindPermanentByName("Bear", pA) != perm1 {
		t.Errorf("FindPermanentByName failed for Bear")
	}
	if zs.FindPermanentByName("Giant", pA) != nil {
		t.Errorf("FindPermanentByName should return nil for wrong controller")
	}

	// Test removal
	removed, ok := zs.RemovePermanent(perm1.ID())
	if !ok || removed != perm1 {
		t.Errorf("RemovePermanent failed")
	}
	if len(zs.Battlefield()) != 1 {
		t.Errorf("expected 1 permanent after removal, got %d", len(zs.Battlefield()))
	}
	if zs.FindPermanent(perm1.ID()) != nil {
		t.Errorf("FindPermanent should return nil after removal")
	}
}

func TestZoneSystem_Phasing(t *testing.T) {
	zs := NewZoneSystem()
	pA := uuid.New()

	creatureCard := NewCreature("Knight", "WW", 2, 2)
	creatureCard.SetOwner(pA)
	creature := NewPermanent(creatureCard, pA)

	auraCard := NewEnchantment("Holy Strength", "W")
	auraCard.SetOwner(pA)
	auraCard.AddSubType("Aura")
	aura := NewPermanent(auraCard, pA)
	aura.AttachedTo = creature.ID()

	zs.AddToBattlefield(creature, aura)

	phased := zs.PhaseOut(creature.ID())
	if len(phased) != 2 {
		t.Fatalf("expected 2 phased permanents, got %d", len(phased))
	}
	if !creature.PhasedOut || !aura.PhasedOut {
		t.Errorf("expected both creature and aura to be phased out")
	}
	if zs.FindPermanent(creature.ID()) != nil {
		t.Errorf("FindPermanent should not find phased-out creature")
	}
	if zs.FindPermanentIncludingPhased(creature.ID()) != creature {
		t.Errorf("FindPermanentIncludingPhased should find phased-out creature")
	}

	zs.PhaseIn(phased)
	if creature.PhasedOut || aura.PhasedOut {
		t.Errorf("expected creature and aura to be phased in")
	}
	if zs.FindPermanent(creature.ID()) != creature {
		t.Errorf("FindPermanent should find phased-in creature")
	}
}

func TestZoneSystem_Exile(t *testing.T) {
	zs := NewZoneSystem()
	pA := uuid.New()
	pB := uuid.New()
	sourceID := uuid.New()

	card1 := NewCreature("Spell Card", "U", 1, 1)
	card1.SetOwner(pA)

	zs.ExileCard(card1, sourceID)
	if len(zs.Exile()) != 1 {
		t.Fatalf("expected 1 exiled card, got %d", len(zs.Exile()))
	}
	ec := zs.FindExiledCard(card1.ID())
	if ec == nil || ec.Card != card1 {
		t.Fatalf("FindExiledCard failed")
	}
	if ec.FaceDown {
		t.Errorf("expected face up exile")
	}

	// Face-down exile with restricted visibility
	card2 := NewCreature("Secret Card", "B", 2, 2)
	card2.SetOwner(pB)
	zs.ExileCardFaceDown(card2, sourceID, pA)

	ec2 := zs.FindExiledCard(card2.ID())
	if ec2 == nil || !ec2.FaceDown {
		t.Fatalf("expected face-down exiled card")
	}
	if !ec2.VisibleTo(pA) {
		t.Errorf("expected visible to pA")
	}
	if ec2.VisibleTo(pB) {
		t.Errorf("expected not visible to pB")
	}

	zs.RevealExiledCardTo(card2.ID(), pB)
	if !ec2.VisibleTo(pB) {
		t.Errorf("expected visible to pB after reveal")
	}

	// Remove by source
	removed := zs.RemoveExiledCardBySource(sourceID)
	if len(removed) != 2 {
		t.Errorf("expected 2 removed cards by source, got %d", len(removed))
	}
	if len(zs.Exile()) != 0 {
		t.Errorf("expected empty exile after RemoveExiledCardBySource, got %d", len(zs.Exile()))
	}
}

func TestZoneSystem_CopyOnWriteCloning(t *testing.T) {
	zs := NewZoneSystem()
	pA := uuid.New()

	card := NewCreature("Bear", "1G", 2, 2)
	card.SetOwner(pA)
	perm := NewPermanent(card, pA)
	perm.Damage = 1
	zs.AddToBattlefield(perm)

	clone := zs.Clone()

	// Initial shared state
	if len(clone.Battlefield()) != 1 {
		t.Fatalf("expected 1 permanent in clone")
	}
	if clone.Battlefield()[0] != perm {
		t.Fatalf("expected initial pointer sharing")
	}

	// Mutate clone via MutablePermanent
	clonePerm := clone.MutablePermanent(perm.ID())
	if clonePerm == perm {
		t.Errorf("MutablePermanent on clone should return a distinct copy")
	}
	clonePerm.Damage = 2

	// Verify original is untouched
	if perm.Damage != 1 {
		t.Errorf("original perm damage mutated: got %d, want 1", perm.Damage)
	}
	if clonePerm.Damage != 2 {
		t.Errorf("clone perm damage not updated: got %d, want 2", clonePerm.Damage)
	}

	// Mutate original via MutablePermanent and verify clone is untouched
	origPerm := zs.MutablePermanent(perm.ID())
	origPerm.Damage = 3
	if clonePerm.Damage != 2 {
		t.Errorf("clone perm damage affected by original mutation: got %d, want 2", clonePerm.Damage)
	}
}

func TestZoneSystem_CastFromExileAndExileInstead(t *testing.T) {
	zs := NewZoneSystem()
	pA := uuid.New()
	cardID := uuid.New()
	sourceID := uuid.New()

	zs.GrantCastFromExile(pA, cardID, true)
	perm := zs.CastFromExilePermissionFor(pA, cardID)
	if perm == nil || !perm.AnyColorMana {
		t.Fatalf("GrantCastFromExile failed")
	}

	zs.ClearCastFromExilePermission(cardID)
	if zs.CastFromExilePermissionFor(pA, cardID) != nil {
		t.Errorf("ClearCastFromExilePermission failed")
	}

	zs.AddExileInstead(cardID, sourceID)
	if !zs.IsExileInstead(cardID) {
		t.Errorf("IsExileInstead should be true")
	}
	zs.ClearEndOfTurn()
	if zs.IsExileInstead(cardID) {
		t.Errorf("IsExileInstead should be false after ClearEndOfTurn")
	}
}
