package mage

import (
	"testing"

	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
)

// TestAttrConstantsAreNonZeroAndUnique verifies all named Attr constants are
// non-zero (so zero value means "no attr") and that no two share a value.
func TestAttrConstantsAreNonZeroAndUnique(t *testing.T) {
	attrs := []Attr{
		AttrCanAttack, AttrCanBlock, AttrHasPowerToughness, AttrSummonSick,
		AttrDoesNotUntap, AttrEntersTapped, AttrMustAttack, AttrMustBeBlocked,
		AttrIsCreature, AttrIsLand, AttrIsArtifact, AttrIsEnchantment,
		Flying, Reach, FirstStrike, DoubleStrike, Trample, Vigilance, Haste,
		Menace, Fear, Deathtouch, Lifelink, Defender, Banding, Indestructible,
		Hexproof, Shroud, Forestwalk, Islandwalk, Swampwalk, Mountainwalk,
		Plainswalk, UnblockableKW, CantBeBlockedByWalls, CantBeBlockedExceptByWalls,
		CanBlockAny, CanBlockAdditional, BasiliskTouch,
	}

	seen := make(map[Attr]bool)
	for _, a := range attrs {
		if a == 0 {
			t.Errorf("attr %v has zero value", a)
		}
		if seen[a] {
			t.Errorf("attr %v is duplicated", a)
		}
		seen[a] = true
	}
}

// TestKeywordIsAttrAlias verifies that Keyword is an alias for Attr.
func TestKeywordIsAttrAlias(t *testing.T) {
	var _ Keyword = Flying        // must compile
	var _ Attr = Flying           // Flying is an Attr
	if Keyword(Flying) != Attr(Flying) {
		t.Errorf("Keyword(Flying) != Attr(Flying)")
	}
}

// TestAliasesMatchCapabilityAttrs verifies backward-compat aliases.
func TestAliasesMatchCapabilityAttrs(t *testing.T) {
	if DoesNotUntapKW != AttrDoesNotUntap {
		t.Errorf("DoesNotUntapKW != AttrDoesNotUntap")
	}
	if EntersTapped != AttrEntersTapped {
		t.Errorf("EntersTapped != AttrEntersTapped")
	}
	if MustAttack != AttrMustAttack {
		t.Errorf("MustAttack != AttrMustAttack")
	}
	if MustBeBlocked != AttrMustBeBlocked {
		t.Errorf("MustBeBlocked != AttrMustBeBlocked")
	}
}

// TestPermanentHasAttr_FalseWhenEmpty ensures a freshly created permanent
// (maps initialized but empty) returns false for any attr.
func TestPermanentHasAttr_FalseWhenEmpty(t *testing.T) {
	card := NewCreature("Test", "{1}", 1, 1)
	p := &Permanent{
		Card:         card,
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
	}
	if p.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) false for empty permanent")
	}
	if p.HasAttr(AttrIsCreature) {
		t.Error("expected HasAttr(AttrIsCreature) false for empty permanent")
	}
}

// TestPermanentGrantBaseAttr_IsAdditive verifies that granting the same attr
// twice increments the count but HasAttr still returns true.
func TestPermanentGrantBaseAttr_IsAdditive(t *testing.T) {
	p := &Permanent{
		Card:         NewCreature("Test", "{1}", 1, 1),
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
	}
	p.GrantBaseAttr(Flying)
	p.GrantBaseAttr(Flying)
	if p.baseAttrs[Flying] != 2 {
		t.Errorf("expected base count 2, got %d", p.baseAttrs[Flying])
	}
	if !p.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) true after two grants")
	}
}

// TestPermanentRevokeBaseAttr_DoesNotGoNegative ensures revoking a zero attr
// does not produce a negative count.
func TestPermanentRevokeBaseAttr_DoesNotGoNegative(t *testing.T) {
	p := &Permanent{
		Card:         NewCreature("Test", "{1}", 1, 1),
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
	}
	p.RevokeBaseAttr(Flying) // nothing to revoke
	if p.baseAttrs[Flying] != 0 {
		t.Errorf("expected 0 after revoking unset attr, got %d", p.baseAttrs[Flying])
	}
	p.GrantBaseAttr(Flying)
	p.RevokeBaseAttr(Flying)
	if p.baseAttrs[Flying] != 0 {
		t.Errorf("expected 0 after grant+revoke, got %d", p.baseAttrs[Flying])
	}
	if p.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) false after grant+revoke")
	}
}

// TestPermanentHasAttr_TrueFromGrantedAttrs verifies that setting grantedAttrs
// directly (as EffectManager.Apply will do) makes HasAttr return true.
func TestPermanentHasAttr_TrueFromGrantedAttrs(t *testing.T) {
	p := &Permanent{
		Card:         NewCreature("Test", "{1}", 1, 1),
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
	}
	p.grantedAttrs[Flying] = 1
	if !p.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) true from grantedAttrs")
	}
}

// TestPermanentHasAttr_SumOfBaseAndGranted verifies additive/subtractive composition:
// base=1, granted=-1 → net=0 → false; base=1, granted=0 → net=1 → true.
func TestPermanentHasAttr_SumOfBaseAndGranted(t *testing.T) {
	p := &Permanent{
		Card:         NewCreature("Test", "{1}", 1, 1),
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
	}
	p.GrantBaseAttr(Flying)
	p.grantedAttrs[Flying] = -1
	if p.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) false when base=1, granted=-1 (net=0)")
	}

	p.grantedAttrs[Flying] = 0
	if !p.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) true when base=1, granted=0 (net=1)")
	}
}

// TestPermanentHasAttr_FaceDown_OnlyBasicCreatureAttrsVisible verifies that a
// face-down permanent only exposes the basic creature attrs.
func TestPermanentHasAttr_FaceDown_OnlyBasicCreatureAttrsVisible(t *testing.T) {
	p := &Permanent{
		Card:         NewCreature("Test", "{1}", 1, 1),
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
		FaceDown:     true,
	}
	// These should return true for face-down
	for _, a := range []Attr{AttrIsCreature, AttrCanAttack, AttrCanBlock, AttrHasPowerToughness} {
		if !p.HasAttr(a) {
			t.Errorf("expected HasAttr(%v) true for face-down permanent", a)
		}
	}
}

// TestPermanentHasAttr_FaceDown_FlyingHidden verifies that keywords are hidden
// on face-down permanents even if present in baseAttrs.
func TestPermanentHasAttr_FaceDown_FlyingHidden(t *testing.T) {
	p := &Permanent{
		Card:         NewCreature("Test", "{1}", 1, 1),
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
		FaceDown:     true,
	}
	p.GrantBaseAttr(Flying)
	if p.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) false for face-down permanent even with baseAttr set")
	}
}

// TestIsKeywordAttr verifies the Flying boundary.
func TestIsKeywordAttr(t *testing.T) {
	if IsKeywordAttr(AttrIsCreature) {
		t.Error("AttrIsCreature should not be a keyword attr")
	}
	if !IsKeywordAttr(Flying) {
		t.Error("Flying should be a keyword attr")
	}
	if !IsKeywordAttr(BasiliskTouch) {
		t.Error("BasiliskTouch should be a keyword attr")
	}
}

// TestNewPermanentInitializesMaps verifies that NewPermanent initializes both maps.
func TestNewPermanentInitializesMaps(t *testing.T) {
	card := NewCreature("Test", "{1}", 1, 1)
	p := NewPermanent(card, uuid.New())
	if p.baseAttrs == nil {
		t.Error("baseAttrs should be initialized")
	}
	if p.grantedAttrs == nil {
		t.Error("grantedAttrs should be initialized")
	}
}
