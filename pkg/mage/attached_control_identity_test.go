package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControlAttachedKeepsHostIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	hostCard := NewCreature("Host", "", 2, 3)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	auraCard := NewAura("Control", "", WithStaticAbility(ControlAttached()))
	auraCard.SetOwner(a.PlayerID())
	aura := g.PutOnBattlefield(auraCard, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	returned := g.PutOnBattlefield(hostCard, b.PlayerID())
	if returned.incarnationID == host.incarnationID {
		t.Fatal("host incarnation did not change")
	}
	if g.FindPermanent(returned.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("old attachment controlled returned host")
	}
	if clone.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("host departure changed clone")
	}
	g.Attach(aura.ID(), returned.ID())
	if g.FindPermanent(returned.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("new attachment did not control returned host")
	}
	phased := g.PhaseOut(returned.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if g.FindPermanent(returned.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("phase cycle changed attachment identity")
	}
}

func TestControlAttachedDirectPhaseAndLayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 3), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Control", "", WithStaticAbility(ControlAttached())), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	for range 2 {
		ids := g.PhaseOut(aura.ID())
		g.ApplyContinuousEffects()
		if g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("phased source retained control effect")
		}
		g.PhaseIn(ids)
		g.ApplyContinuousEffects()
		if g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("phase-in did not restore effect")
		}
	}
	g.AddContinuousEffect(RemoveAllAbilities(aura.ID(), core.Indefinite))
	for range 2 {
		g.ApplyContinuousEffects()
		if g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("layer 6 removal erased layer 2 control")
		}
	}
}
