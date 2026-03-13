package mage

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// TestNewCreature_HasAttrs_CanAttack_CanBlock_HasPT_IsCreature verifies that a
// freshly created creature permanent has all the expected capability and identity attrs.
func TestNewCreature_HasAttrs_CanAttack_CanBlock_HasPT_IsCreature(t *testing.T) {
	card := NewCreature("Test Creature", "{2}{G}", 2, 2)
	perm := NewPermanent(card, uuid.New())

	for _, a := range []Attr{AttrCanAttack, AttrCanBlock, AttrHasPowerToughness, AttrIsCreature, AttrSummonSick} {
		if !perm.HasAttr(a) {
			t.Errorf("expected HasAttr(%v) true for creature", a)
		}
	}
}

// TestNewCreature_NoAttr_IsLand_IsArtifact ensures creature doesn't have land/artifact attrs.
func TestNewCreature_NoAttr_IsLand_IsArtifact(t *testing.T) {
	card := NewCreature("Test Creature", "{2}{G}", 2, 2)
	perm := NewPermanent(card, uuid.New())

	if perm.HasAttr(AttrIsLand) {
		t.Error("creature should not have AttrIsLand")
	}
	if perm.HasAttr(AttrIsArtifact) {
		t.Error("creature should not have AttrIsArtifact")
	}
}

// TestNewLand_HasAttr_IsLand verifies land permanents have AttrIsLand.
func TestNewLand_HasAttr_IsLand(t *testing.T) {
	card := NewLand("Forest")
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrIsLand) {
		t.Error("expected HasAttr(AttrIsLand) true for land")
	}
}

// TestNewLand_NoAttr_IsCreature verifies lands don't have creature attrs.
func TestNewLand_NoAttr_IsCreature(t *testing.T) {
	card := NewLand("Forest")
	perm := NewPermanent(card, uuid.New())
	if perm.HasAttr(AttrIsCreature) {
		t.Error("land should not have AttrIsCreature")
	}
	if perm.HasAttr(AttrCanAttack) {
		t.Error("land should not have AttrCanAttack")
	}
	if perm.HasAttr(AttrSummonSick) {
		t.Error("land should not have AttrSummonSick")
	}
}

// TestNewArtifact_HasAttr_IsArtifact verifies artifact permanents have AttrIsArtifact.
func TestNewArtifact_HasAttr_IsArtifact(t *testing.T) {
	card := NewArtifact("Sol Ring", "{1}")
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrIsArtifact) {
		t.Error("expected HasAttr(AttrIsArtifact) true for artifact")
	}
}

// TestNewEnchantment_HasAttr_IsEnchantment verifies enchantment permanents have AttrIsEnchantment.
func TestNewEnchantment_HasAttr_IsEnchantment(t *testing.T) {
	card := NewEnchantment("Test Enchantment", "{2}")
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrIsEnchantment) {
		t.Error("expected HasAttr(AttrIsEnchantment) true for enchantment")
	}
}

// TestNewAura_HasAttr_IsEnchantment verifies aura permanents have AttrIsEnchantment.
func TestNewAura_HasAttr_IsEnchantment(t *testing.T) {
	card := NewAura("Test Aura", "{1}{W}")
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrIsEnchantment) {
		t.Error("expected HasAttr(AttrIsEnchantment) true for aura")
	}
}

// TestNewToken_HasAttr_IsCreature verifies token creatures have creature attrs.
func TestNewToken_HasAttr_IsCreature(t *testing.T) {
	card := NewToken("Bird", 1, 1, []CardType{TypeCreature}, []string{"Bird"}, Flying)
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrIsCreature) {
		t.Error("expected HasAttr(AttrIsCreature) true for token creature")
	}
	if !perm.HasAttr(AttrCanAttack) {
		t.Error("expected HasAttr(AttrCanAttack) true for token creature")
	}
}

// TestWithKeyword_Flying_ReflectedInHasAttr verifies that WithKeyword populates
// the attrSeeds so NewPermanent reflects it in baseAttrs.
func TestWithKeyword_Flying_ReflectedInHasAttr(t *testing.T) {
	card := NewCreature("Angel", "{4}{W}{W}", 4, 4, WithKeyword(Flying))
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) true for creature with WithKeyword(Flying)")
	}
}

// TestWithKeyword_DoesNotUntapKW_ReflectedInHasAttr verifies backward-compat alias.
func TestWithKeyword_DoesNotUntapKW_ReflectedInHasAttr(t *testing.T) {
	card := NewCreature("Frozen Creature", "{2}{U}", 2, 2, WithKeyword(DoesNotUntapKW))
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrDoesNotUntap) {
		t.Error("expected HasAttr(AttrDoesNotUntap) true for creature with DoesNotUntapKW")
	}
}

// TestWithKeyword_EntersTapped_ReflectedInHasAttr verifies EntersTapped alias.
func TestWithKeyword_EntersTapped_ReflectedInHasAttr(t *testing.T) {
	card := NewCreature("Tapped Entry", "{3}", 2, 3, WithKeyword(EntersTapped))
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrEntersTapped) {
		t.Error("expected HasAttr(AttrEntersTapped) true for creature with EntersTapped")
	}
}

// TestWithKeyword_MultipleKeywords_AllReflected ensures multiple keywords are all set.
func TestWithKeyword_MultipleKeywords_AllReflected(t *testing.T) {
	card := NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
		WithKeyword(Flying), WithKeyword(Vigilance))
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(Flying) {
		t.Error("expected Flying")
	}
	if !perm.HasAttr(Vigilance) {
		t.Error("expected Vigilance")
	}
}

// TestHasType_Creature_DelegatesToAttrIsCreature verifies that granting
// AttrIsCreature via grantedAttrs makes HasType(TypeCreature) return true
// even on a non-creature card.
func TestHasType_Creature_DelegatesToAttrIsCreature(t *testing.T) {
	card := NewLand("Forest")
	perm := NewPermanent(card, uuid.New())
	// Initially not a creature
	if perm.HasType(TypeCreature) {
		t.Error("land should not be a creature initially")
	}
	// Grant creature attr via grantedAttrs (as EffectManager will do)
	perm.grantedAttrs[AttrIsCreature] = 1
	if !perm.HasType(TypeCreature) {
		t.Error("expected HasType(TypeCreature) true after granting AttrIsCreature")
	}
}

// TestHasType_Land_DelegatesToAttrIsLand verifies land type delegation.
func TestHasType_Land_DelegatesToAttrIsLand(t *testing.T) {
	card := NewLand("Forest")
	perm := NewPermanent(card, uuid.New())
	if !perm.HasType(TypeLand) {
		t.Error("expected HasType(TypeLand) true for land")
	}
}

// TestHasType_CardLevel_StillUsesTypesSlice verifies that BaseCard.HasType
// still works via the types slice (not affected by Permanent.HasType).
func TestHasType_CardLevel_StillUsesTypesSlice(t *testing.T) {
	card := NewCreature("Test", "{1}", 1, 1)
	if !card.HasType(TypeCreature) {
		t.Error("card.HasType(TypeCreature) should be true via types slice")
	}
	if card.HasType(TypeLand) {
		t.Error("card.HasType(TypeLand) should be false for creature card")
	}
}

// TestHasKeyword_DelegatesToHasAttr verifies that setting baseAttrs[Flying]
// makes HasKeyword(Flying) return true.
func TestHasKeyword_DelegatesToHasAttr(t *testing.T) {
	card := NewCreature("Test", "{1}", 1, 1)
	perm := NewPermanent(card, uuid.New())
	// Set directly in baseAttrs (bypassing WithKeyword for this isolated test)
	perm.baseAttrs[Flying] = 1
	if !perm.HasKeyword(Flying) {
		t.Error("expected HasKeyword(Flying) true when baseAttrs[Flying]=1")
	}
}

// TestNewCreature_SummonSick_SetOnETB verifies creatures start summoning sick.
func TestNewCreature_SummonSick_SetOnETB(t *testing.T) {
	card := NewCreature("Test", "{1}", 1, 1)
	perm := NewPermanent(card, uuid.New())
	if !perm.HasAttr(AttrSummonSick) {
		t.Error("expected AttrSummonSick set on creature ETB")
	}
	// Clearing via RevokeBaseAttr should remove it
	perm.RevokeBaseAttr(AttrSummonSick)
	if perm.HasAttr(AttrSummonSick) {
		t.Error("expected AttrSummonSick cleared after RevokeBaseAttr")
	}
}
