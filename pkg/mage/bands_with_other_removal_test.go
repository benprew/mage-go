package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBandsWithOtherRemoval(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	holder := g.PutOnBattlefield(NewCreature("Member", "", 1, 20, WithKeyword(core.Banding), WithKeyword(core.Flying)), a.PlayerID())
	partner := g.PutOnBattlefield(NewCreature("Member", "", 1, 20), a.PlayerID())
	ids := []uuid.UUID{holder.ID(), partner.ID()}
	grant := func(quality PermanentFilter) {
		t.Helper()
		if err := ApplyEffect(g, Until(core.EndOfTurn, GrantEachBandsWithOther(ControlledPermanents(IsCreature), quality)), holder.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	grant(Named("Member"))
	grant(IsCreature)
	if len(g.bandsQualities(g.FindPermanent(holder.ID()))) != 2 {
		t.Fatal("missing independent qualities")
	}
	holder.RevokeBaseAttr(core.AttrSummonSick)
	partner.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), ids)
	if err := g.FormAttackingBand(ids); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveBandsWithOther(This)), holder.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.bandsQualities(g.FindPermanent(holder.ID()))) != 0 || len(g.bandsQualities(g.FindPermanent(partner.ID()))) != 2 {
		t.Fatal("removal did not select one exact subject")
	}
	if !g.FindPermanent(holder.ID()).HasKeyword(core.Banding) || !g.FindPermanent(holder.ID()).HasKeyword(core.Flying) {
		t.Fatal("ordinary keywords removed")
	}
	if !g.combat.IsBandedWith(ids[0], ids[1]) || !g.hasBandingAssignment(ids) {
		t.Fatal("ordinary band or permission lost")
	}
	g.MutablePermanent(holder.ID()).RevokeBaseAttr(core.Banding)
	g.MutablePermanent(partner.ID()).RevokeBaseAttr(core.Banding)
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveBandsWithOther(This)), partner.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.hasBandingAssignment(ids) || !clone.hasBandingAssignment(ids) {
		t.Fatal("quality removal did not stop live assignment or changed clone")
	}
	if !g.combat.IsBandedWith(ids[0], ids[1]) {
		t.Fatal("removal erased formed band")
	}
	grant(Named("Member"))
	if len(g.bandsQualities(g.FindPermanent(holder.ID()))) != 1 || !g.hasBandingAssignment(ids) {
		t.Fatal("later grant lost timestamp order")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveBandsWithOther(This)), holder.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveBandsWithOther(This)), partner.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.bandsQualities(g.FindPermanent(holder.ID()))) != 0 || g.hasBandingAssignment(ids) {
		t.Fatal("later removal lost timestamp order")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if len(g.bandsQualities(g.FindPermanent(holder.ID()))) != 0 || len(clone.bandsQualities(clone.FindPermanent(holder.ID()))) != 2 {
		t.Fatal("cleanup or clone isolation")
	}
}
