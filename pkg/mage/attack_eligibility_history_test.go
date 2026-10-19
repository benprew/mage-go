package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackEligibilityHistoryBoundaries(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit", true: "native"}[native], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			ready := g.PutOnBattlefield(NewCreature("Ready", "", 1, 8), a.PlayerID())
			sick := g.PutOnBattlefield(NewCreature("Sick", "", 1, 8), a.PlayerID())
			tapped := g.PutOnBattlefield(NewCreature("Tapped", "", 1, 8), a.PlayerID())
			defender := g.PutOnBattlefield(NewCreature("Defender", "", 1, 8, WithKeyword(core.Defender)), a.PlayerID())
			taxed := g.PutOnBattlefield(NewCreature("Taxed", "", 1, 8, WithStaticAbility(SourceCantAttackUnlessPays(ManaCostOf("{1}")))), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), b.PlayerID())
			for _, p := range []*Permanent{ready, tapped, defender, taxed, other} {
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			tapped.Tapped = true
			for _, p := range []*Permanent{ready, sick, tapped, defender, taxed, other} {
				if !CouldntAttackThisTurn.Match(p, g) {
					t.Fatal("history existed before declaration")
				}
			}
			if native {
				g.RunStepWithPriority(core.DeclareAttackers)
			} else {
				g.ExecuteAttackers(a.PlayerID(), nil)
			}
			g.UntapPermanent(tapped)
			sick.RevokeBaseAttr(core.AttrSummonSick)
			for _, p := range []*Permanent{sick, tapped, defender, taxed, other} {
				if !CouldntAttackThisTurn.Match(g.FindPermanent(p.ID()), g) {
					t.Fatalf("%s gained historical eligibility", p.Name())
				}
			}
			if CouldntAttackThisTurn.Match(g.FindPermanent(ready.ID()), g) {
				t.Fatal("ready creature has no history")
			}
			if a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
				t.Fatal("snapshot made a payment or trigger")
			}
			g.MutablePermanent(ready.ID()).Tapped = true
			if CouldntAttackThisTurn.Match(g.FindPermanent(ready.ID()), g) {
				t.Fatal("later tap erased history")
			}
			clone := g.Clone()
			g.PhaseIn(g.PhaseOut(ready.ID()))
			if CouldntAttackThisTurn.Match(g.FindPermanent(ready.ID()), g) {
				t.Fatal("phase cycle erased history")
			}
			old := g.FindPermanent(ready.ID())
			g.RemoveFromBattlefield(old)
			returned := g.PutOnBattlefield(old.Card, a.PlayerID())
			returned.RevokeBaseAttr(core.AttrSummonSick)
			if !CouldntAttackThisTurn.Match(returned, g) || CouldntAttackThisTurn.Match(clone.FindPermanent(ready.ID()), clone) {
				t.Fatal("returned visit or clone history")
			}
			g.GetCombat().Reset()
			g.ExecuteAttackers(a.PlayerID(), nil)
			if CouldntAttackThisTurn.Match(g.FindPermanent(sick.ID()), g) || CouldntAttackThisTurn.Match(returned, g) {
				t.Fatal("extra combat did not add visits")
			}
			clone.RunStepWithPriority(core.Cleanup)
			if !CouldntAttackThisTurn.Match(clone.FindPermanent(ready.ID()), clone) || CouldntAttackThisTurn.Match(g.FindPermanent(sick.ID()), g) {
				t.Fatal("cleanup or clone isolation")
			}
		})
	}
}

func TestAttackEligibilityHistoryRequirements(t *testing.T) {
	for _, limit := range []int{0, 1, 2, 3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitAttackDeclarations(limit)))), a.PlayerID())
		optional := g.PutOnBattlefield(NewCreature("Optional", "", 1, 8), a.PlayerID())
		required := g.PutOnBattlefield(NewCreature("Required", "", 1, 8, WithKeyword(core.MustAttack)), a.PlayerID())
		equal := g.PutOnBattlefield(NewCreature("Equal", "", 1, 8, WithKeyword(core.MustAttack)), a.PlayerID())
		for _, p := range []*Permanent{optional, required, equal} {
			p.RevokeBaseAttr(core.AttrSummonSick)
		}
		g.ExecuteAttackers(a.PlayerID(), nil)
		if CouldntAttackThisTurn.Match(g.FindPermanent(optional.ID()), g) != (limit < 3) {
			t.Fatalf("optional limit %d", limit)
		}
		for _, p := range []*Permanent{required, equal} {
			if CouldntAttackThisTurn.Match(g.FindPermanent(p.ID()), g) != (limit == 0) {
				t.Fatalf("equal maximum declaration limit %d", limit)
			}
		}
	}
}

func TestAttackEligibilityHistoryRestrictions(t *testing.T) {
	for _, kind := range []string{"rule", "sanctuary", "phased", "noncreature"} {
		g := newPriorityTestGame()
		a, b := g.players[0], g.players[1]
		p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 8), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		switch kind {
		case "rule":
			mustApplyEligibility(t, g, p, a.PlayerID(), CantAttackMatchingThisTurn(IsCreature))
		case "sanctuary":
			g.effects.Rules.SetSanctuaryActive(b.PlayerID())
		case "phased":
			g.PhaseOut(p.ID())
		case "noncreature":
			p = g.PutOnBattlefield(NewArtifact("Artifact", ""), a.PlayerID())
		}
		g.ExecuteAttackers(a.PlayerID(), nil)
		if !CouldntAttackThisTurn.Match(g.FindPermanentIncludingPhased(p.ID()), g) {
			t.Fatalf("%s bypassed", kind)
		}
	}
}

func mustApplyEligibility(t *testing.T, g *Game, p *Permanent, controller uuid.UUID, e ResolvingEffect) {
	t.Helper()
	if err := ApplyEffect(g, e, p.ID(), controller, nil); err != nil {
		t.Fatal(err)
	}
}

type eligibilityHistoryPlayer struct {
	*BasePlayer
	offers int
}

func (p *eligibilityHistoryPlayer) ChooseMayAbility(string) bool { p.offers++; return false }

func TestAttackEligibilityHistoryCostsAndPurity(t *testing.T) {
	a := &eligibilityHistoryPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	free := g.PutOnBattlefield(NewCreature("Free", "", 1, 8), a.PlayerID())
	taxed := g.PutOnBattlefield(NewCreature("Taxed", "", 1, 8, WithStaticAbility(SourceCantAttackUnlessPays(ManaCostOf("{1}")))), a.PlayerID())
	for _, p := range []*Permanent{free, taxed} {
		p.RevokeBaseAttr(core.AttrSummonSick)
	}
	a.ManaPool().Add(core.Red, 1)
	g.doDeclareAttackers()
	if a.offers != 0 || a.ManaPool().TotalMana() != 1 || free.Tapped || taxed.Tapped {
		t.Fatal("snapshot made choices or paid costs")
	}
	for range 3 {
		if CouldntAttackThisTurn.Match(free, g) || !CouldntAttackThisTurn.Match(taxed, g) {
			t.Fatal("wrong optional-cost history")
		}
	}
	clone := g.Clone()
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{taxed.ID()})
	if !g.HasAttackedThisTurn(taxed.ID()) || CouldntAttackThisTurn.Match(g.FindPermanent(taxed.ID()), g) || a.ManaPool().TotalMana() != 0 {
		t.Fatal("actual paid attack not recorded")
	}
	if !CouldntAttackThisTurn.Match(clone.FindPermanent(taxed.ID()), clone) || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 1 {
		t.Fatal("payment leaked into clone")
	}
	g.RemoveFromBattlefield(g.FindPermanent(taxed.ID()))
	returned := g.PutOnBattlefield(taxed.Card, a.PlayerID())
	if !CouldntAttackThisTurn.Match(returned, g) {
		t.Fatal("paid attack followed returned visit")
	}
}
