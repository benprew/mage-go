package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSourceIdentityAfterReturnAndCleanup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 2, WithKeyword(Lifelink), WithKeyword(Deathtouch))
	old := g.PutOnBattlefield(card, a.PlayerID())
	source := g.CaptureDamageSource(old.ID())
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.zones.ClearLKI()
	if source.Incarnation() == returned.incarnationID || source.Controller() != a.PlayerID() || !source.HasKeyword(Lifelink) || !source.HasKeyword(Deathtouch) {
		t.Fatal("damage source changed with the returned card")
	}
	g.DealDamageToPlayerFromSource(b, 2, source)
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatalf("life totals after old source damage = %d, %d", a.Life(), b.Life())
	}
}

func TestDamageSourceIdentityCloneAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewCreature("Source", "", 1, 2, WithKeyword(Lifelink)), a.PlayerID())
	source := g.CaptureDamageSource(old.ID())
	g.PhaseOut(old.ID())
	if g.CaptureDamageSource(old.ID()).Valid() {
		t.Fatal("phased-out permanent supplied a new damage source")
	}
	clone := g.Clone()
	clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, source)
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("clone damage source affected the original game")
	}
}

func TestDamageSourceIdentityUnrelatedResolutionSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
	g.resolution.sourceCaptured = true
	g.resolution.sourceIncarnation = first.incarnationID
	source := g.CaptureDamageSource(other.ID())
	if source.ID() != other.ID() || source.Incarnation() != other.incarnationID || source.HasKeyword(Lifelink) {
		t.Fatal("damage used an unrelated resolving source")
	}
	g.DealDamageToPlayerFromSource(a, 1, source)
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
}

func TestDamageSourceIdentityDepartedResolutionSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 1, WithKeyword(Lifelink))
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.resolution.sourceCaptured = true
	g.resolution.sourceIncarnation = old.incarnationID
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(card, b.PlayerID())
	g.zones.ClearLKI()
	source := g.CaptureDamageSource(card.ID())
	if source.Incarnation() != old.incarnationID || source.Controller() != a.PlayerID() {
		t.Fatal("resolution lost its departed source")
	}
	g.DealDamageToPlayerFromSource(b, 1, source)
	if a.Life() != 21 || b.Life() != 19 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
	if g.CaptureDamageSource(uuid.New()).Valid() {
		t.Fatal("unknown source used resolution LKI")
	}
}

func TestDamageSourceIdentitySpellController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := NewInstant("Borrowed Spell", "", nil)
	spell.SetOwner(a.PlayerID())
	end := g.resolution.Begin(&StackObject{Card: spell, Controller: b.PlayerID(), SourceID: spell.ID()})
	defer end()
	source := g.CaptureDamageSource(spell.ID())
	if source.Controller() != b.PlayerID() || source.Owner() != a.PlayerID() {
		t.Fatalf("spell source controller = %s, owner = %s", source.Controller(), source.Owner())
	}
}

func TestDamageSourceIdentityProtectionAndDeathtouchAfterReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 1, WithKeyword(Deathtouch))
	old := g.PutOnBattlefield(card, a.PlayerID())
	source := g.CaptureDamageSource(old.ID())
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(card, b.PlayerID())
	protected := g.PutOnBattlefield(NewCreature("Protected", "", 2, 4, WithAbility(ProtectionFromCardType(TypeCreature))), b.PlayerID())
	g.DealDamageToPermanentFromSource(protected, 1, source)
	if protected.Damage != 0 {
		t.Fatal("protection did not prevent old-source damage")
	}
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 4), b.PlayerID())
	g.DealDamageToPermanentFromSource(victim, 1, source)
	if victim.Damage != 1 {
		t.Fatalf("deathtouch marked damage = %d, want 1", victim.Damage)
	}
	if !victim.DeathtouchDamage {
		t.Fatal("expected deathtouch damage flag to be set")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("expected victim to be destroyed by state-based actions")
	}
}

func TestDamageSourceIdentityTypeAndCardCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Animated Source", "")
	perm := g.PutOnBattlefield(card, a.PlayerID())
	perm.GrantBaseAttr(AttrIsCreature)
	source := g.CaptureDamageSource(perm.ID())
	if !source.HasType(TypeCreature) || source.card.ID() != perm.ID() {
		t.Fatal("source did not preserve effective type and identity")
	}
	perm.RevokeBaseAttr(AttrIsCreature)
	if !source.HasType(TypeCreature) {
		t.Fatal("source type changed after capture")
	}
	victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 2, WithAbility(ProtectionFromCardType(TypeCreature))), b.PlayerID())
	g.DealDamageToPermanentFromSource(victim, 1, source)
	if victim.Damage != 0 {
		t.Fatal("protection did not use the captured type")
	}
}

func TestDamageSourceIdentityUnknownLegacySource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 2, WithAbility(ProtectionFromCardType(TypeCreature))), b.PlayerID())
	sourceID := uuid.New()
	g.DealDamageToPermanent(victim, 1, sourceID)
	if victim.Damage != 1 || !g.HasDealtDamageToPermanent(sourceID, victim.ID()) {
		t.Fatal("unknown legacy source lost its ID or damage outcome")
	}
}
