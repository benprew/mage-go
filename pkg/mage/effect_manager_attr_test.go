package mage

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// makeTestGameWithPerm creates a minimal game state with one permanent for EM tests.
func makeTestGameWithPerm(t *testing.T) (*Game, *Permanent) {
	t.Helper()
	card := NewCreature("Test", "{1}", 1, 1)
	perm := NewPermanent(card, uuid.New())
	g := &Game{
		Battlefield: []*Permanent{perm},
		Stack:       NewStack(),
		Combat:      NewCombat(),
		Effects:     NewEffectManager(),
	}
	return g, perm
}

// TestEffectManager_GrantAttr_AddsToAttrDeltas verifies GrantAttr writes to attrDeltas.
func TestEffectManager_GrantAttr_AddsToAttrDeltas(t *testing.T) {
	_, perm := makeTestGameWithPerm(t)
	em := NewEffectManager()
	em.GrantAttr(perm.ID(), Flying)
	if em.attrDeltas[perm.ID()][Flying] != 1 {
		t.Errorf("expected attrDeltas[Flying]=1, got %d", em.attrDeltas[perm.ID()][Flying])
	}
}

// TestEffectManager_RevokeAttr_SubtractsFromAttrDeltas verifies RevokeAttr writes negative.
func TestEffectManager_RevokeAttr_SubtractsFromAttrDeltas(t *testing.T) {
	_, perm := makeTestGameWithPerm(t)
	em := NewEffectManager()
	em.RevokeAttr(perm.ID(), Flying)
	if em.attrDeltas[perm.ID()][Flying] != -1 {
		t.Errorf("expected attrDeltas[Flying]=-1, got %d", em.attrDeltas[perm.ID()][Flying])
	}
}

// TestEffectManager_Apply_ResetsGrantedAttrsFirst verifies grantedAttrs is cleared
// before effects are re-applied.
func TestEffectManager_Apply_ResetsGrantedAttrsFirst(t *testing.T) {
	g, perm := makeTestGameWithPerm(t)
	// Set a stale value
	perm.grantedAttrs[Flying] = 99
	g.Effects.Apply(g)
	if perm.grantedAttrs[Flying] != 0 {
		t.Errorf("expected grantedAttrs[Flying]=0 after Apply reset, got %d", perm.grantedAttrs[Flying])
	}
}

// TestEffectManager_Apply_WritesAttrDeltasToPerms verifies that attrDeltas accumulated
// during Apply() are written to perm.grantedAttrs.
func TestEffectManager_Apply_WritesAttrDeltasToPerms(t *testing.T) {
	g, perm := makeTestGameWithPerm(t)
	// Use Indefinite duration so the effect isn't culled by source-on-battlefield filter.
	ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, sourceID uuid.UUID) error {
		g.Effects.GrantAttr(perm.ID(), Flying)
		return nil
	}, func(g *Game, _ uuid.UUID) bool { return true })
	g.Effects.Add(ce)
	g.Effects.Apply(g)
	if perm.grantedAttrs[Flying] != 1 {
		t.Errorf("expected grantedAttrs[Flying]=1 after Apply, got %d", perm.grantedAttrs[Flying])
	}
}

// TestEffectManager_GrantAttr_MakesHasAttrTrue_AfterApply verifies that GrantAttr
// from an effect makes HasAttr return true after Apply().
func TestEffectManager_GrantAttr_MakesHasAttrTrue_AfterApply(t *testing.T) {
	g, perm := makeTestGameWithPerm(t)
	// perm starts with no Flying in baseAttrs
	if perm.HasAttr(Flying) {
		t.Fatal("precondition: perm should not have Flying")
	}
	ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, sourceID uuid.UUID) error {
		g.Effects.GrantAttr(perm.ID(), Flying)
		return nil
	}, func(g *Game, _ uuid.UUID) bool { return true })
	g.Effects.Add(ce)
	g.Effects.Apply(g)
	if !perm.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) true after GrantAttr effect applied")
	}
}

// TestEffectManager_RevokeAttr_NegatesBaseAttr_AfterApply verifies that RevokeAttr
// from an effect negates a baseAttr (base=1, revoke=-1 → net=0 → false).
func TestEffectManager_RevokeAttr_NegatesBaseAttr_AfterApply(t *testing.T) {
	g, perm := makeTestGameWithPerm(t)
	perm.GrantBaseAttr(Flying)
	if !perm.HasAttr(Flying) {
		t.Fatal("precondition: perm should have Flying")
	}
	ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, sourceID uuid.UUID) error {
		g.Effects.RevokeAttr(perm.ID(), Flying)
		return nil
	}, func(g *Game, _ uuid.UUID) bool { return true })
	g.Effects.Add(ce)
	g.Effects.Apply(g)
	if perm.HasAttr(Flying) {
		t.Error("expected HasAttr(Flying) false after RevokeAttr negates base")
	}
}

// TestEffectManager_GrantedAttr_ClearedBetweenApplyCycles verifies that
// if an effect is removed, its attr grant is gone after the next Apply().
func TestEffectManager_GrantedAttr_ClearedBetweenApplyCycles(t *testing.T) {
	g, perm := makeTestGameWithPerm(t)
	sourceID := uuid.New()
	ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, sid uuid.UUID) error {
		g.Effects.GrantAttr(perm.ID(), Flying)
		return nil
	}, func(g *Game, _ uuid.UUID) bool { return true })
	ce.(*funcContinuousEffect).sourceID = sourceID
	g.Effects.Add(ce)
	g.Effects.Apply(g)
	if !perm.HasAttr(Flying) {
		t.Fatal("should have Flying after first Apply")
	}
	// Remove the effect and apply again
	g.Effects.Remove(sourceID)
	g.Effects.Apply(g)
	if perm.HasAttr(Flying) {
		t.Error("expected Flying gone after removing effect and re-applying")
	}
}

// TestEffectManager_ReplacePreventAttack_WithRevokeAttrCanAttack verifies that
// RevokeAttr(AttrCanAttack) makes HasAttr(AttrCanAttack) false after Apply().
func TestEffectManager_ReplacePreventAttack_WithRevokeAttrCanAttack(t *testing.T) {
	g, perm := makeTestGameWithPerm(t)
	// Creature starts with AttrCanAttack
	if !perm.HasAttr(AttrCanAttack) {
		t.Fatal("precondition: creature should have AttrCanAttack")
	}
	ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
		g.Effects.RevokeAttr(perm.ID(), AttrCanAttack)
		return nil
	}, func(g *Game, _ uuid.UUID) bool { return true })
	g.Effects.Add(ce)
	g.Effects.Apply(g)
	if perm.HasAttr(AttrCanAttack) {
		t.Error("expected AttrCanAttack false after RevokeAttr")
	}
}

// TestEffectManager_ReplaceRemovedKW_WithRevokeAttrKeyword verifies the RevokeAttr
// pattern for keyword removal (equivalent to removedKW).
func TestEffectManager_ReplaceRemovedKW_WithRevokeAttrKeyword(t *testing.T) {
	card := NewCreature("Air Elemental", "{3}{U}{U}", 4, 4, WithKeyword(Flying))
	perm := NewPermanent(card, uuid.New())
	g := &Game{
		Battlefield: []*Permanent{perm},
		Stack:       NewStack(),
		Combat:      NewCombat(),
		Effects:     NewEffectManager(),
	}
	if !perm.HasAttr(Flying) {
		t.Fatal("precondition: creature should have Flying")
	}
	ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
		g.Effects.RevokeAttr(perm.ID(), Flying)
		return nil
	}, func(g *Game, _ uuid.UUID) bool { return true })
	g.Effects.Add(ce)
	g.Effects.Apply(g)
	if perm.HasAttr(Flying) {
		t.Error("expected Flying false after RevokeAttr negates base")
	}
}
