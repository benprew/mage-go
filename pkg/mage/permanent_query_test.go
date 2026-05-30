package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// makeTestGame returns a minimal game for query method tests.
func makeTestGame() *Game {
	return &Game{
		stack:   NewStack(),
		combat:  NewCombat(),
		effects: NewEffectManager(),
	}
}

// TestCreature_CanDeclareAsAttacker_WhenUntappedAndNotSummonSick verifies baseline.
func TestCreature_CanDeclareAsAttacker_WhenUntappedAndNotSummonSick(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New())
	p.RevokeBaseAttr(AttrSummonSick) // cleared after first untap
	if !p.CanDeclareAsAttacker(g) {
		t.Error("expected CanDeclareAsAttacker true for untapped, not sick creature")
	}
}

// TestCreature_CannotDeclareAsAttacker_WhenTapped verifies tapped creatures can't attack.
func TestCreature_CannotDeclareAsAttacker_WhenTapped(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New())
	p.RevokeBaseAttr(AttrSummonSick)
	p.Tapped = true
	if p.CanDeclareAsAttacker(g) {
		t.Error("expected CanDeclareAsAttacker false when tapped")
	}
}

// TestCreature_CannotDeclareAsAttacker_WhenSummonSick verifies summoning sickness.
func TestCreature_CannotDeclareAsAttacker_WhenSummonSick(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New()) // SummonSick is set by NewPermanent
	if p.CanDeclareAsAttacker(g) {
		t.Error("expected CanDeclareAsAttacker false when summoning sick")
	}
}

// TestCreature_CanDeclareAsAttacker_WhenSummonSickButHasHaste verifies Haste bypass.
func TestCreature_CanDeclareAsAttacker_WhenSummonSickButHasHaste(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2, WithKeyword(Haste))
	p := NewPermanent(card, uuid.New()) // SummonSick is set, but Haste bypasses it
	if !p.CanDeclareAsAttacker(g) {
		t.Error("expected CanDeclareAsAttacker true when summoning sick but has Haste")
	}
}

// TestCreature_CannotDeclareAsAttacker_WhenHasDefender verifies Defender restriction.
func TestCreature_CannotDeclareAsAttacker_WhenHasDefender(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Wall", "{2}", 0, 4, WithKeyword(Defender))
	p := NewPermanent(card, uuid.New())
	p.RevokeBaseAttr(AttrSummonSick)
	if p.CanDeclareAsAttacker(g) {
		t.Error("expected CanDeclareAsAttacker false when has Defender")
	}
}

// TestCreature_CannotDeclareAsAttacker_WhenAttackRevokedByEffect verifies that
// RevokeAttr(AttrCanAttack) prevents attack declaration.
func TestCreature_CannotDeclareAsAttacker_WhenAttackRevokedByEffect(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New())
	p.RevokeBaseAttr(AttrSummonSick)
	g.battlefield = append(g.battlefield, p)
	// Simulate an effect revoking AttrCanAttack
	p.grantedAttrs[AttrCanAttack] = -1 // net = baseAttrs(1) + granted(-1) = 0
	if p.CanDeclareAsAttacker(g) {
		t.Error("expected CanDeclareAsAttacker false when AttrCanAttack revoked")
	}
}

// TestLand_CannotDeclareAsAttacker verifies lands can't attack.
func TestLand_CannotDeclareAsAttacker(t *testing.T) {
	g := makeTestGame()
	card := NewLand("Forest")
	p := NewPermanent(card, uuid.New())
	if p.CanDeclareAsAttacker(g) {
		t.Error("expected CanDeclareAsAttacker false for land")
	}
}

// TestCreature_CanDeclareAsBlocker_WhenUntapped verifies baseline blocking.
func TestCreature_CanDeclareAsBlocker_WhenUntapped(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New())
	g.battlefield = append(g.battlefield, p)
	if !p.CanDeclareAsBlocker(g) {
		t.Error("expected CanDeclareAsBlocker true for untapped creature")
	}
}

// TestCreature_CannotDeclareAsBlocker_WhenTapped verifies tapped creatures can't block.
func TestCreature_CannotDeclareAsBlocker_WhenTapped(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New())
	p.Tapped = true
	g.battlefield = append(g.battlefield, p)
	if p.CanDeclareAsBlocker(g) {
		t.Error("expected CanDeclareAsBlocker false when tapped")
	}
}

// TestCreature_CannotDeclareAsBlocker_WhenBlockPrevented verifies prevention via attr.
// PreventBlockingUntilEndOfCombat revokes AttrCanBlock via grantedAttrs; simulate that
// by directly writing grantedAttrs (the continuous effect does this each Apply cycle).
func TestCreature_CannotDeclareAsBlocker_WhenBlockPrevented(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New())
	g.battlefield = append(g.battlefield, p)
	// Simulate RevokeAttr(AttrCanBlock): net = base(1) + granted(-1) = 0 → HasAttr false
	p.grantedAttrs[AttrCanBlock] = -1
	if p.CanDeclareAsBlocker(g) {
		t.Error("expected CanDeclareAsBlocker false when blocking prevented")
	}
}

// TestLand_CannotDeclareAsBlocker verifies lands can't block.
func TestLand_CannotDeclareAsBlocker(t *testing.T) {
	g := makeTestGame()
	card := NewLand("Forest")
	p := NewPermanent(card, uuid.New())
	g.battlefield = append(g.battlefield, p)
	if p.CanDeclareAsBlocker(g) {
		t.Error("expected CanDeclareAsBlocker false for land")
	}
}

// TestCreature_CannotTapForEffect_WhenSummonSick verifies summoning sickness for mana.
func TestCreature_CannotTapForEffect_WhenSummonSick(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2)
	p := NewPermanent(card, uuid.New()) // SummonSick set
	if p.CanTapForEffect(g) {
		t.Error("expected CanTapForEffect false for summoning-sick creature")
	}
}

// TestCreature_CanTapForEffect_WhenSummonSickButHasHaste verifies Haste bypass for mana.
func TestCreature_CanTapForEffect_WhenSummonSickButHasHaste(t *testing.T) {
	g := makeTestGame()
	card := NewCreature("Test", "{2}", 2, 2, WithKeyword(Haste))
	p := NewPermanent(card, uuid.New())
	if !p.CanTapForEffect(g) {
		t.Error("expected CanTapForEffect true when summoning sick but has Haste")
	}
}

// TestLand_CanTapForEffect_EvenWhenJustPlayed verifies lands are never summoning sick.
func TestLand_CanTapForEffect_EvenWhenJustPlayed(t *testing.T) {
	g := makeTestGame()
	card := NewLand("Forest")
	p := NewPermanent(card, uuid.New())
	if !p.CanTapForEffect(g) {
		t.Error("expected CanTapForEffect true for land (no summoning sickness)")
	}
}

// TestArtifact_CanTapForEffect_Immediately verifies non-creature artifacts have no sickness.
func TestArtifact_CanTapForEffect_Immediately(t *testing.T) {
	g := makeTestGame()
	card := NewArtifact("Sol Ring", "{1}")
	p := NewPermanent(card, uuid.New())
	if !p.CanTapForEffect(g) {
		t.Error("expected CanTapForEffect true for artifact (no HasPowerToughness)")
	}
}
