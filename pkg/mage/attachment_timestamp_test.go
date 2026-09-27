package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachmentMoveUpdatesStaticEffectOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), a.PlayerID())
	grant := g.PutOnBattlefield(NewAura("Grant", "", WithStaticAbility(AttachedEffect(LayerAbility, func(g *Game, _, target *Permanent) error {
		g.effects.GrantAttr(target.ID(), Flying)
		return nil
	}))), a.PlayerID())
	remove := g.PutOnBattlefield(NewAura("Remove", "", WithStaticAbility(AttachedEffect(LayerAbility, func(g *Game, _, target *Permanent) error {
		g.effects.RevokeAttr(target.ID(), Flying)
		return nil
	}))), b.PlayerID())
	g.Attach(grant.ID(), host.ID())
	g.Attach(remove.ID(), host.ID())
	if g.FindPermanent(host.ID()).HasAttr(Flying) {
		t.Fatal("later attachment did not remove flying")
	}
	g.Attach(grant.ID(), other.ID())
	g.Attach(grant.ID(), host.ID())
	if !g.FindPermanent(host.ID()).HasAttr(Flying) {
		t.Fatal("reattached aura did not apply after earlier aura")
	}
	g.Attach(remove.ID(), host.ID())
	if !g.FindPermanent(host.ID()).HasAttr(Flying) {
		t.Fatal("same-host attach changed effect order")
	}
	land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	g.Attach(remove.ID(), land.ID())
	if !g.FindPermanent(host.ID()).HasAttr(Flying) {
		t.Fatal("illegal attachment changed effect order")
	}
	g.PhaseOut(remove.ID())
	g.Attach(remove.ID(), other.ID())
	g.PhaseIn([]uuid.UUID{remove.ID()})
	if !g.FindPermanent(host.ID()).HasAttr(Flying) {
		t.Fatal("phased-out attachment changed effect order")
	}
	clone := g.Clone()
	clone.Attach(remove.ID(), other.ID())
	clone.Attach(remove.ID(), host.ID())
	if clone.FindPermanent(host.ID()).HasAttr(Flying) {
		t.Fatal("clone reattachment did not update its effect order")
	}
	if !g.FindPermanent(host.ID()).HasAttr(Flying) {
		t.Fatal("clone reattachment changed original effect order")
	}
}
