package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachRejectsIllegalAuraMoveWithoutChangingAttachment(t *testing.T) {
	for _, tc := range []struct {
		name     string
		makeHost func() Card
		modify   func(*Permanent)
	}{
		{"wrong type", func() Card { return NewLand("Land") }, nil},
		{"cannot be enchanted", func() Card { return NewCreature("Host", "", 2, 2) }, func(p *Permanent) { p.GrantBaseAttr(core.AttrCantBeEnchanted) }},
		{"protection", func() Card { return NewCreature("Host", "", 2, 2, WithAbility(ProtectionFromColor(core.White))) }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			old := g.PutOnBattlefield(NewCreature("Old", "", 2, 2), a.PlayerID())
			card := NewAura("Aura", "{W}")
			card.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), old.ID())
			host := g.PutOnBattlefield(tc.makeHost(), b.PlayerID())
			if tc.modify != nil {
				tc.modify(g.MutablePermanent(host.ID()))
			}
			g.Attach(aura.ID(), host.ID())
			if got := g.FindPermanent(aura.ID()).AttachedTo; got != old.ID() {
				t.Fatalf("attachment moved to %v", got)
			}
			if len(g.FindPermanent(old.ID()).Attachments) != 1 || len(g.FindPermanent(host.ID()).Attachments) != 0 {
				t.Fatal("host attachment lists changed")
			}
		})
	}
}

func TestAttachRejectsInvalidSourceAndSameHost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID())
	g.Attach(other.ID(), host.ID())
	if other.AttachedTo != uuid.Nil {
		t.Fatal("ordinary creature attached")
	}
	aura := g.PutOnBattlefield(NewAura("Aura", ""), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.Attach(aura.ID(), host.ID())
	if got := len(g.FindPermanent(host.ID()).Attachments); got != 1 {
		t.Fatalf("same-host move added %d attachments", got)
	}
	g.PhaseOut(other.ID())
	g.Attach(aura.ID(), other.ID())
	if got := g.FindPermanent(aura.ID()).AttachedTo; got != host.ID() {
		t.Fatal("attached to phased-out host")
	}
	clone := g.Clone()
	clone.Attach(aura.ID(), clone.PutOnBattlefield(NewCreature("New", "", 2, 2), a.PlayerID()).ID())
	if got := g.FindPermanent(aura.ID()).AttachedTo; got != host.ID() {
		t.Fatal("clone changed original attachment")
	}
}

func TestAttachUsesAuraEnchantRestrictionAndIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second"), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Land aura", "", WithCastTarget(TargetLand())), a.PlayerID())
	g.Attach(aura.ID(), first.ID())
	g.RemoveFromBattlefield(first)
	g.Attach(aura.ID(), first.ID())
	if got := g.FindPermanent(aura.ID()).AttachedTo; got != first.ID() {
		t.Fatalf("departed host record changed to %v", got)
	}
	g.Attach(aura.ID(), second.ID())
	if got := g.FindPermanent(aura.ID()).AttachedTo; got != second.ID() {
		t.Fatalf("legal land attachment = %v", got)
	}
	returned := g.PutOnBattlefield(first.Card, a.PlayerID())
	g.Attach(aura.ID(), returned.ID())
	if got := g.FindPermanent(aura.ID()).attachedIncarnation; got != returned.incarnationID {
		t.Fatal("attachment kept old host incarnation")
	}
}
