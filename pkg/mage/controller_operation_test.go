package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func controllerTestGame(t *testing.T) (*Game, Player, Player, *Permanent) {
	t.Helper()
	a := NewBasePlayer("Alice")
	b := NewBasePlayer("Bob")
	g := NewGame(a, b)
	card := NewCreature("Borrowed Creature", "{2}{U}", 2, 2)
	card.SetOwner(b.PlayerID())
	perm := g.PutOnBattlefield(card, a.PlayerID())
	return g, a, b, perm
}

func TestControllerBaseAndLayerReconciliation(t *testing.T) {
	g, a, b, perm := controllerTestGame(t)

	g.ApplyContinuousEffects()
	g.ApplyContinuousEffects()
	if got := perm.ControllerID(); got != a.PlayerID() {
		t.Fatalf("cross-owner battlefield controller = %s, want %s", got, a.PlayerID())
	}

	if err := ApplyEffect(g, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{perm.ID()}); err != nil {
		t.Fatal(err)
	}
	if got := perm.ControllerID(); got != b.PlayerID() {
		t.Fatalf("temporary controller = %s, want %s", got, b.PlayerID())
	}

	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if got := perm.ControllerID(); got != a.PlayerID() {
		t.Fatalf("expired control fell back to %s, want base controller %s", got, a.PlayerID())
	}
}

func TestControlEffectsUseTimestampOrder(t *testing.T) {
	g, a, b, perm := controllerTestGame(t)

	g.AddControlEffect(ControlEffectSpec{
		SourceID: b.PlayerID(), TargetID: perm.ID(), ControllerID: b.PlayerID(), Duration: Indefinite,
	})
	g.AddControlEffect(ControlEffectSpec{
		SourceID: a.PlayerID(), TargetID: perm.ID(), ControllerID: a.PlayerID(), Duration: EndOfTurn,
	})
	if got := perm.ControllerID(); got != a.PlayerID() {
		t.Fatalf("latest effect controller = %s, want %s", got, a.PlayerID())
	}

	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if got := perm.ControllerID(); got != b.PlayerID() {
		t.Fatalf("controller after latest effect expires = %s, want %s", got, b.PlayerID())
	}
}

func TestConditionalControlExpiresPermanently(t *testing.T) {
	g, a, b, target := controllerTestGame(t)
	sourceCard := NewCreature("Controller", "{1}{U}", 3, 3)
	sourceCard.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(sourceCard, b.PlayerID())
	source.Tapped = true

	g.AddControlEffect(ControlEffectSpec{
		SourceID: source.ID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: Indefinite,
		Conditions: []ControlConditionData{
			ControlSourceControlledByEffectController{},
			ControlSourceTapped{},
			ControlTargetPowerLESource{},
		},
		TapMaintained: true,
	})
	if got := target.ControllerID(); got != b.PlayerID() {
		t.Fatalf("conditional controller = %s, want %s", got, b.PlayerID())
	}

	source.Tapped = false
	g.ApplyContinuousEffects()
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("controller after condition failed = %s, want %s", got, a.PlayerID())
	}
	source.Tapped = true
	g.ApplyContinuousEffects()
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("expired condition restarted: got %s, want %s", got, a.PlayerID())
	}
}

func TestControlChangeRestoresSummoningSickness(t *testing.T) {
	g, _, b, perm := controllerTestGame(t)
	g.SetTurn(3)
	perm.RevokeBaseAttr(AttrSummonSick)

	if err := ApplyEffect(g, GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{perm.ID()}); err != nil {
		t.Fatal(err)
	}
	if !perm.HasAttr(AttrSummonSick) {
		t.Fatal("control change did not restore summoning sickness")
	}
	if perm.ControlledSinceTurnStart(g) {
		t.Fatal("new controller should not count as controlling since turn start")
	}

	perm.GrantBaseAttr(Haste)
	if !perm.CanDeclareAsAttacker(g) {
		t.Fatal("haste should bypass summoning sickness after a control change")
	}
}

func TestControlAttachedFollowsAuraController(t *testing.T) {
	g, a, b, target := controllerTestGame(t)
	auraCard := NewAura("Control Aura", "{2}{U}{U}", WithStaticAbility(ControlAttached()))
	auraCard.SetOwner(a.PlayerID())
	aura := g.PutOnBattlefield(auraCard, a.PlayerID())
	g.Attach(aura.ID(), target.ID())
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("attached controller = %s, want %s", got, a.PlayerID())
	}

	g.AddControlEffect(ControlEffectSpec{
		SourceID: b.PlayerID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: Indefinite,
	})
	if got := aura.ControllerID(); got != b.PlayerID() {
		t.Fatalf("aura controller = %s, want %s", got, b.PlayerID())
	}
	if got := target.ControllerID(); got != b.PlayerID() {
		t.Fatalf("enchanted permanent did not follow aura controller: got %s, want %s", got, b.PlayerID())
	}

	g.RemoveFromBattlefield(aura)
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("controller after aura left = %s, want base %s", got, a.PlayerID())
	}
}

func TestCantChangeControlPreservesLegalNonOwnerController(t *testing.T) {
	g, a, b, perm := controllerTestGame(t)
	perm.GrantBaseAttr(AttrCantChangeControl)
	g.AddControlEffect(ControlEffectSpec{
		SourceID: b.PlayerID(), TargetID: perm.ID(), ControllerID: b.PlayerID(), Duration: Indefinite,
	})
	if got := perm.ControllerID(); got != a.PlayerID() {
		t.Fatalf("can't-change-control restored %s, want preexisting non-owner controller %s", got, a.PlayerID())
	}
}

func TestFaceDownControllerResetsAndCloneIsIsolated(t *testing.T) {
	g, a, b, perm := controllerTestGame(t)
	perm.FaceDown = true
	g.AddControlEffect(ControlEffectSpec{
		SourceID: b.PlayerID(), TargetID: perm.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn,
	})
	if got := perm.ControllerID(); got != b.PlayerID() {
		t.Fatalf("face-down controller = %s, want %s", got, b.PlayerID())
	}

	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if got := clone.FindPermanent(perm.ID()).ControllerID(); got != a.PlayerID() {
		t.Fatalf("clone controller after expiration = %s, want %s", got, a.PlayerID())
	}
	if got := perm.ControllerID(); got != b.PlayerID() {
		t.Fatalf("clone controller mutation leaked: got %s, want %s", got, b.PlayerID())
	}
}
