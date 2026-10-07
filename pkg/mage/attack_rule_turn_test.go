package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackRulesThisTurnMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, CantAttackMatchingThisTurn(And(IsCreature, Not(HasKeywordFilter(core.Flying)))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	c := NewCreature("Late", "", 2, 3)
	c.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(c, b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("late creature bypassed restriction")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, GrantKeywordModification(This, core.Flying)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("rule ignored current characteristics")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("ability removal erased rule")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("phase cycle erased rule")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	p = g.PutOnBattlefield(c, b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("returned creature bypassed live rule")
	}
	clone := g.Clone()
	clone.RunStepWithPriority(core.Cleanup)
	if !clone.FindPermanent(p.ID()).CanDeclareAsAttacker(clone) || g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("cleanup or clone isolation")
	}
}

func TestAttackRulesThisTurnRequirements(t *testing.T) {
	for _, path := range []string{"normal", "explicit"} {
		t.Run(path, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(1)
			if err := ApplyEffect(g, MustAttackControlledThisTurn(You, IsCreature), uuid.New(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(NewCreature("Late", "", 2, 3), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			q := g.PutOnBattlefield(NewCreature("Tapped", "", 2, 3), a.PlayerID())
			q.RevokeBaseAttr(core.AttrSummonSick)
			g.TapPermanent(q)
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 3), b.PlayerID())
			other.RevokeBaseAttr(core.AttrSummonSick)
			if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if path == "normal" {
				g.RunStepWithPriority(core.DeclareAttackers)
			} else {
				g.ExecuteAttackers(a.PlayerID(), nil)
			}
			if !g.GetCombat().IsAttacking(p.ID()) || g.GetCombat().IsAttacking(q.ID()) || g.GetCombat().IsAttacking(other.ID()) {
				t.Fatal("live requirement or legality")
			}
			g.RemoveFromCombat(p.ID())
			g.UntapPermanent(g.FindPermanent(p.ID()))
			if err := ApplyEffect(g, CantAttackMatchingThisTurn(IsCreature), uuid.New(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ExecuteAttackers(a.PlayerID(), nil)
			if g.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("requirement bypassed restriction")
			}
		})
	}
}

func TestAttackRulesThisTurnDefinitions(t *testing.T) {
	target := DeclarePlayerTarget("player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "global", Card: NewInstant("Global", "", WithAbility(Spell(CantAttackMatchingThisTurn(IsCreature))))},
		{Name: "controlled", Card: NewInstant("Controlled", "", WithAbility(Spell(MustAttackControlledThisTurn(You, IsCreature))))},
		{Name: "nil", Card: NewInstant("Nil", "", WithAbility(Spell(MustAttackControlledThisTurn(nil, IsCreature)))), WantErr: true},
		{Name: "unbound", Card: NewInstant("Unbound", "", WithAbility(Spell(MustAttackControlledThisTurn(target, IsCreature)))), WantErr: true},
	})
}

func TestAttackRulesThisTurnPlayerIdentityAndCosts(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, MustAttackControlledThisTurn(You, IsCreature), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, ChangeController(This, You)), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	taxed := g.PutOnBattlefield(NewCreature("Taxed", "", 2, 3, WithStaticAbility(SourceCantAttackUnlessPays(GenericCost(1)))), a.PlayerID())
	taxed.RevokeBaseAttr(core.AttrSummonSick)
	p := g.PutOnBattlefield(NewCreature("Free", "", 2, 3), b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	a.ManaPool().Add(core.Colorless, 1)
	g.ExecuteAttackers(a.PlayerID(), nil)
	if g.GetCombat().IsAttacking(taxed.ID()) || g.GetCombat().IsAttacking(p.ID()) || a.ManaPool().TotalMana() != 1 {
		t.Fatal("rule changed player or forced a payment")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, ChangeController(This, You)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.FindPermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	clone := g.Clone()
	g.ExecuteAttackers(a.PlayerID(), nil)
	if !g.GetCombat().IsAttacking(p.ID()) || clone.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("current control or clone isolation")
	}
	clone.RunStepWithPriority(core.Cleanup)
	clone.ExecuteAttackers(a.PlayerID(), nil)
	if clone.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("expired requirement survived cleanup")
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{taxed.ID()})
	if !g.GetCombat().IsAttacking(taxed.ID()) || a.ManaPool().TotalMana() != 0 {
		t.Fatal("voluntary attack did not use shared payment")
	}
}

func TestAttackRulesThisTurnAlreadyInCombat(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 3), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	if err := ApplyEffect(g, CantAttackMatchingThisTurn(IsCreature), uuid.New(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("restriction removed a declared attacker")
	}
}
