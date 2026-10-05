package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackOnlyHastePermission(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
	if host.CanDeclareAsAttacker(g) || host.CanTapForEffect(g) {
		t.Fatal("new creature has no summoning restriction")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, AttackAsThoughHaste(This)), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	host = g.FindPermanent(host.ID())
	if !host.CanDeclareAsAttacker(g) || host.CanTapForEffect(g) || host.HasKeyword(core.Haste) {
		t.Fatal("attack permission changed tap legality or haste")
	}
	clone := g.Clone()
	g.AddContinuousEffect(RemoveAllAbilities(host.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("ability removal erased a rule permission")
	}
	g.MutablePermanent(host.ID()).Tapped = true
	if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("permission ignored tapping")
	}
	if !clone.FindPermanent(host.ID()).CanDeclareAsAttacker(clone) {
		t.Fatal("clone lost permission")
	}
	g.MutablePermanent(host.ID()).Tapped = false
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) || !clone.FindPermanent(host.ID()).CanDeclareAsAttacker(clone) {
		t.Fatal("expiration or clone isolation")
	}
	if core.IsKeywordAttr(core.AttrAttackAsThoughHaste) || core.IsAbilityAttr(core.AttrAttackAsThoughHaste) {
		t.Fatal("rule permission classified as ability")
	}
}

func TestAttackOnlyHasteOtherRestrictions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Host", "", 2, 4, WithKeyword(core.Defender))
	host := g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, AttackAsThoughHaste(This)), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("attack-only haste bypassed defender")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, GrantKeywordModification(This, core.AttrCanAttack)), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("permissions did not combine")
	}
	clone := g.Clone()
	phased := g.PhaseOut(host.ID())
	g.ApplyContinuousEffects()
	if clone.FindPermanent(host.ID()).CanTapForEffect(clone) {
		t.Fatal("clone has full haste")
	}
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("phase cycle lost captured rule")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
	if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) || g.FindPermanent(host.ID()).CanTapForEffect(g) {
		t.Fatal("control change changed rule scope")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.CanDeclareAsAttacker(g) {
		t.Fatal("rule followed a returned incarnation")
	}
	if !clone.FindPermanent(host.ID()).CanDeclareAsAttacker(clone) {
		t.Fatal("departure changed clone")
	}
	full := g.PutOnBattlefield(NewCreature("Full haste", "", 2, 4, WithKeyword(core.Haste)), a.PlayerID())
	if !full.CanDeclareAsAttacker(g) || !full.CanTapForEffect(g) {
		t.Fatal("full haste changed")
	}
}

func TestAttackOnlyHasteTapCostRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 4, WithAbility(Activate(Costs(Tap()), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, AttackAsThoughHaste(This)), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), host.ID(), 0, nil); err == nil {
		t.Fatal("summoning-sick tap cost accepted")
	}
	if g.FindPermanent(host.ID()).Tapped || g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed tap cost changed state")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, GrantKeywordModification(This, core.Haste)), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), host.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(host.ID()).Tapped || a.Life() != 21 {
		t.Fatal("full haste activation failed")
	}
}
