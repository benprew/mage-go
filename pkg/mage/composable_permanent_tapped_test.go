package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentTappedAttachmentState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := NewCreature("Host", "", 2, 2)
	other := NewCreature("Other", "", 2, 2)
	aura := NewAura("Aura", "")
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	source := g.FindPermanent(aura.ID())
	ctx := StateContext{SourceID: aura.ID(), ControllerID: a.PlayerID(), SourceIncarnation: source.incarnationID}
	untapped := PermanentIsUntapped(AttachedPermanent)
	if untapped.checkState(g, ctx) {
		t.Fatal("unattached source matched")
	}
	g.Attach(aura.ID(), host.ID())
	if !untapped.checkState(g, ctx) || PermanentIsTapped(AttachedPermanent).checkState(g, ctx) {
		t.Fatal("untapped host did not match")
	}
	clone := g.Clone()
	clone.MutablePermanent(host.ID()).Tapped = true
	if !PermanentIsTapped(AttachedPermanent).checkState(clone, ctx) || !untapped.checkState(g, ctx) {
		t.Fatal("clone shares host tapped state")
	}
	g.MutablePermanent(host.ID()).Tapped = true
	if !PermanentIsTapped(AttachedPermanent).checkState(g, ctx) || untapped.checkState(g, ctx) {
		t.Fatal("tapped host did not match")
	}
	g.Attach(aura.ID(), other.ID())
	if !untapped.checkState(g, ctx) {
		t.Fatal("condition did not follow reattachment")
	}
	g.MutablePermanent(other.ID()).PhasedOut = true
	if untapped.checkState(g, ctx) || PermanentIsTapped(AttachedPermanent).checkState(g, ctx) {
		t.Fatal("phased host matched")
	}
	g.MutablePermanentIncludingPhased(other.ID()).PhasedOut = false
	if !untapped.checkState(g, ctx) {
		t.Fatal("phased-in host lost identity")
	}
	g.PhaseOut(aura.ID())
	if untapped.checkState(g, ctx) || PermanentIsTapped(AttachedPermanent).checkState(g, ctx) {
		t.Fatal("phased source matched")
	}
	g.PhaseIn([]uuid.UUID{aura.ID()})
	if !untapped.checkState(g, ctx) {
		t.Fatal("phased-in source lost identity")
	}
	g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
	g.PutOnBattlefield(other, b.PlayerID())
	if untapped.checkState(g, ctx) {
		t.Fatal("returned host matched old attachment")
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), other.ID())
	if untapped.checkState(g, ctx) {
		t.Fatal("returned source matched old incarnation")
	}
}

func TestPermanentTappedAttachmentStaticAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aura := NewAura("Aura", "", WithAbility(Static(While(PermanentIsUntapped(AttachedPermanent), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition("Aura", aura); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCardDefinition("Source", NewCreature("Source", "", 1, 1,
		WithAbility(WhenState(PermanentIsTapped(This), GainLifeFor(You, Fixed(1)))))); err != nil {
		t.Fatal(err)
	}
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("untapped host did not gain shroud")
	}
	g.MutablePermanent(host.ID()).Tapped = true
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("tapped host retained shroud")
	}
	target := DeclarePermanentTarget("target", IsCreature)
	invalid := NewAura("Invalid", "", WithAbility(Static(While(PermanentIsTapped(target), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition("Invalid", invalid); err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("unsupported state reference accepted: %v", err)
	}
}
