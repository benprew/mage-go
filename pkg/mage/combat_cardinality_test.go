package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatCardinalityRequirements(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitAttackDeclarations(1))), WithAbility(Static(LimitBlockDeclarations(1)))), a.PlayerID())
	optional := g.PutOnBattlefield(NewCreature("Optional", "", 1, 8), a.PlayerID())
	required := g.PutOnBattlefield(NewCreature("Required", "", 1, 8, WithKeyword(core.MustAttack)), a.PlayerID())
	optional.RevokeBaseAttr(core.AttrSummonSick)
	required.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{optional.ID()})
	if g.combat.IsAttacking(optional.ID()) || !g.combat.IsAttacking(required.ID()) {
		t.Fatal("attack requirement did not replace optional attack")
	}
	required.GrantBaseAttr(core.AttrMustBeBlockedIfAble)
	required.GrantBaseAttr(core.Menace)
	one := g.PutOnBattlefield(NewCreature("One", "", 1, 8), b.PlayerID())
	two := g.PutOnBattlefield(NewCreature("Two", "", 1, 8), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{one.ID(), required.ID()}, {two.ID(), required.ID()}})
	if len(g.combat.GroupFor(required.ID()).BlockerIDs) != 0 {
		t.Fatal("requirement bypassed cap or menace")
	}
}

func TestCombatCardinalityDefinition(t *testing.T) {
	for _, rule := range []StaticRule{LimitAttackDeclarations(-1), LimitBlockDeclarations(-1)} {
		card := NewEnchantment("Invalid", "", WithAbility(Static(rule)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("negative limit accepted")
		}
	}
	for _, rule := range []StaticRule{LimitAttackDeclarations(0), LimitBlockDeclarations(0)} {
		card := NewInstant("Invalid", "", WithAbility(Static(rule)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("spell static rule accepted")
		}
	}
}

func TestCombatCardinalityBothDeclarationPaths(t *testing.T) {
	for _, native := range []bool{false, true} {
		for _, limit := range []int{0, 1, 2, 4} {
			a := &playerRestrictionAttacker{BasePlayer: NewBasePlayer("A")}
			b := &fixedBlockPlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitAttackDeclarations(limit))), WithAbility(Static(LimitBlockDeclarations(limit)))), a.PlayerID())
			for range 3 {
				p := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				a.attackers = append(a.attackers, p.ID())
				q := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
				b.assignments = append(b.assignments, BlockAssignment{q.ID(), a.attackers[0]})
			}
			a.attackers = append(a.attackers, a.attackers[0])
			b.assignments = append(b.assignments, b.assignments[0])
			if native {
				g.doDeclareAttackers()
			} else {
				g.ExecuteAttackers(a.PlayerID(), a.attackers)
			}
			if got := len(g.combat.Groups); got != min(limit, 3) {
				t.Fatalf("native=%v limit=%d attackers=%d", native, limit, got)
			}
			if native {
				g.doDeclareBlockers()
			} else {
				g.ExecuteBlockers(b.assignments)
			}
			if limit > 0 && len(g.combat.Groups[0].BlockerIDs) != min(limit, 3) {
				t.Fatal("block cardinality")
			}
		}
	}
}

func TestCombatCardinalityDistinctBlockersAndRequirements(t *testing.T) {
	for _, band := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitBlockDeclarations(1)))), a.PlayerID())
		first := requirementAttacker(g, a, "First", IsCreature, WithKeyword(core.Banding))
		second := requirementAttacker(g, a, "Second", IsCreature)
		g.combat.AddAttackerIncarnation(first.ID(), first.incarnationID, b.PlayerID())
		g.combat.AddAttackerIncarnation(second.ID(), second.incarnationID, b.PlayerID())
		if band {
			if err := g.FormAttackingBand([]uuid.UUID{first.ID(), second.ID()}); err != nil {
				t.Fatal(err)
			}
		}
		one := g.PutOnBattlefield(NewCreature("One", "", 1, 8, WithKeyword(core.CanBlockAny)), b.PlayerID())
		two := g.PutOnBattlefield(NewCreature("Two", "", 1, 8), b.PlayerID())
		proposed := []BlockAssignment{{two.ID(), first.ID()}, {one.ID(), second.ID()}}
		before := g.Clone()
		result := g.CompleteBlockAssignments(b.PlayerID(), proposed)
		if len(g.combat.Groups[0].BlockerIDs) != 0 || len(before.combat.Groups[0].BlockerIDs) != 0 {
			t.Fatal("query changed combat")
		}
		g.ExecuteBlockers(result)
		for _, group := range g.combat.Groups {
			if len(group.BlockerIDs) != 1 || group.BlockerIDs[0] != one.ID() {
				t.Fatal("multi-blocker did not maximize requirements within distinct cap")
			}
		}
		if before.combat.IsBlocking(one.ID()) {
			t.Fatal("clone combat changed")
		}
	}
}

func TestCombatCardinalityAttackWeightsAndCosts(t *testing.T) {
	for _, native := range []bool{false, true} {
		a := &playerRestrictionAttacker{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitAttackDeclarations(1)))), a.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 8, WithKeyword(core.MustAttack)), a.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 8, WithKeyword(core.MustAttack), WithKeyword(core.MustAttack)), a.PlayerID())
		taxed := g.PutOnBattlefield(NewCreature("Taxed", "", 1, 8, WithKeyword(core.MustAttack), WithKeyword(core.MustAttack), WithKeyword(core.MustAttack), WithStaticAbility(SourceCantAttackUnlessPays(GenericCost(1)))), a.PlayerID())
		for _, p := range []*Permanent{first, second, taxed} {
			p.RevokeBaseAttr(core.AttrSummonSick)
		}
		a.ManaPool().Add(core.Colorless, 1)
		a.attackers = []uuid.UUID{first.ID()}
		if native {
			g.doDeclareAttackers()
		} else {
			g.ExecuteAttackers(a.PlayerID(), a.attackers)
		}
		if len(g.combat.Groups) != 1 || !g.combat.IsAttacking(second.ID()) || a.ManaPool().TotalMana() != 1 || taxed.Tapped {
			t.Fatal("weighted requirements or optional tax")
		}
	}
}

func TestCombatCardinalityLifecycleAndMinimum(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewEnchantment("Limit", "", WithAbility(Static(LimitAttackDeclarations(2))), WithAbility(Static(LimitBlockDeclarations(2))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewEnchantment("Other", "", WithAbility(Static(LimitAttackDeclarations(1))), WithAbility(Static(LimitBlockDeclarations(1)))), b.PlayerID())
	clone := g.Clone()
	if g.combatDeclarationLimit(false) != 1 || g.combatDeclarationLimit(true) != 1 {
		t.Fatal("minimum")
	}
	g.PhaseOut(other.ID())
	if g.combatDeclarationLimit(false) != 2 || clone.combatDeclarationLimit(false) != 1 {
		t.Fatal("phase clone isolation")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if g.combatDeclarationLimit(false) != -1 {
		t.Fatal("departure")
	}
	g.PutOnBattlefield(card, b.PlayerID())
	if g.combatDeclarationLimit(false) != 2 {
		t.Fatal("returned ability did not register")
	}
	removal := RemoveAllAbilities(source.ID(), core.EndOfTurn)
	g.AddContinuousEffect(removal)
	g.effects.Apply(g)
	if g.combatDeclarationLimit(false) != -1 {
		t.Fatal("ability removal")
	}
	g.effects.RemoveEndOfTurn()
	g.effects.Apply(g)
	if g.combatDeclarationLimit(false) != 2 {
		t.Fatal("restoration")
	}
}

func TestCombatCardinalityAttackLegality(t *testing.T) {
	a := &playerRestrictionAttacker{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitAttackDeclarations(1)))), a.PlayerID())
	g.effects.Rules.SetSanctuaryActive(b.PlayerID())
	ground := g.PutOnBattlefield(NewCreature("Ground", "", 1, 8, WithKeyword(core.MustAttack), WithKeyword(core.MustAttack)), a.PlayerID())
	air := g.PutOnBattlefield(NewCreature("Air", "", 1, 8, WithKeyword(core.MustAttack), WithKeyword(core.Flying)), a.PlayerID())
	ground.RevokeBaseAttr(core.AttrSummonSick)
	air.RevokeBaseAttr(core.AttrSummonSick)
	g.doDeclareAttackers()
	if len(g.combat.Groups) != 1 || !g.combat.IsAttacking(air.ID()) {
		t.Fatal("illegal required creature displaced a legal attacker")
	}
}
