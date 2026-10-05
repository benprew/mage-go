package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCantAttackEachFinalCharacteristics(t *testing.T) {
	for _, before := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		c := NewCreature("Subject", "", 2, 3)
		p := g.PutOnBattlefield(c, a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		source := NewEnchantment("Rule", "", WithAbility(Static(CantAttackEach(MatchingPermanents(And(IsCreature, Not(HasKeywordFilter(core.Flying))))))))
		grant := NewEnchantment("Grant", "", WithAbility(Static(While(StateConditionFunc("always", nil, func(StateReader, StateContext) bool { return true }), GrantEachKeyword(MatchingPermanents(IsCreature), core.Flying)))))
		if before {
			g.PutOnBattlefield(grant, b.PlayerID())
		}
		rule := g.PutOnBattlefield(source, b.PlayerID())
		if !before {
			g.PutOnBattlefield(grant, b.PlayerID())
		}
		if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("final flying grant did not permit attack")
		}
		g.RemoveFromBattlefield(g.FindPermanent(grant.ID()))
		if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("nonflying creature can attack")
		}
		if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("recipient ability removal erased rule")
		}
		clone := g.Clone()
		clone.RemoveFromBattlefield(clone.FindPermanent(rule.ID()))
		if !clone.FindPermanent(p.ID()).CanDeclareAsAttacker(clone) || g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("clone rule isolation")
		}
	}
}

func TestCantAttackEachDefinition(t *testing.T) {
	target := DeclarePermanentTarget("target", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewEnchantment("Good", "", WithAbility(Static(CantAttackEach(ControlledPermanents(IsCreature)))))},
		{Name: "nil", Card: NewEnchantment("Bad", "", WithAbility(Static(CantAttackEach(nil)))), WantErr: true},
		{Name: "unbound", Card: NewEnchantment("Bad", "", WithAbility(Static(CantAttackEach(WithoutPermanent(MatchingPermanents(IsCreature), target))))), WantErr: true},
	})
}
