package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestCombatStateBranchCurrentIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 2, 2)
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), g.FindPermanent(attacker.ID()).incarnationID, b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
	attack := IfPermanentAttacking(This, GainLifeFor(You, Fixed(2)))
	block := IfPermanentBlocking(This, GainLifeFor(You, Fixed(3)))
	if err := ApplyEffect(g, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, block, blocker.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 23 {
		t.Fatalf("combat branches: %d, %d", a.Life(), b.Life())
	}
	clone := g.Clone()
	phased := g.PhaseOut(attacker.ID())
	if err := ApplyEffect(g, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(clone, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatal("phasing or clone changed combat membership")
	}
	g.PhaseIn(phased)
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	g.PutOnBattlefield(attacker, a.PlayerID())
	if err := ApplyEffect(g, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatal("returned object inherited combat membership")
	}
}

func TestCombatStateBranchTargetScope(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := NewInstant("Valid", "", Spell(Sequence(
		IfPermanentAttacking(target, GainLifeFor(You, Fixed(1))),
		IfPermanentBlocking(target, GainLifeFor(You, Fixed(1))),
	), WithTargets(target)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		IfPermanentAttacking(nil, GainLifeFor(You, Fixed(1))),
		IfPermanentBlocking(target, GainLifeFor(You, Fixed(1))),
		IfPermanentAttacking(target, nil),
	} {
		bad := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid combat branch accepted")
		}
	}

	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Target", "", 2, 2)
	g.PutOnBattlefield(c, b.PlayerID())
	g.combat.AddAttackerIncarnation(c.ID(), g.FindPermanent(c.ID()).incarnationID, a.PlayerID())
	ability := valid.Abilities()[0].(*SpellAbility)
	obj := &StackObject{ID: uuid.New(), SourceID: valid.ID(), Card: valid, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("target combat branch life = %d", a.Life())
	}
	obj = &StackObject{ID: uuid.New(), SourceID: valid.ID(), Card: valid, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 21 {
		t.Fatal("cloned target branch changed the original frame")
	}
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	g.PutOnBattlefield(c, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("stale target matched a returned object")
	}
}
