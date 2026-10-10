package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerAttackRestrictionLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, CantBeAttackedExceptByUntilNextTurn(You, HasKeywordFilter(core.Flying), You), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	g.RunStepWithPriority(core.Cleanup)
	p := g.PutOnBattlefield(NewCreature("Late", "", 2, 3), b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("late attacker bypassed player rule")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, GrantKeywordModification(This, core.Flying)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("current flying not read")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("ability removal erased player rule")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("phase cycle erased player rule")
	}
	clone := g.Clone()
	g.AdvanceToNextTurn()
	g.BeginActiveTurn()
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("opponent turn expired rule")
	}
	g.AdvanceToNextTurn()
	g.BeginActiveTurn()
	if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) || clone.FindPermanent(p.ID()).CanDeclareAsAttacker(clone) {
		t.Fatal("next turn or clone expiry")
	}
}

func TestPlayerAttackRestrictionDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		CantBeAttackedExceptByUntilNextTurn(nil, AnyPermanent, You),
		CantBeAttackedExceptByUntilNextTurn(You, AnyPermanent, nil),
		CantBeAttackedExceptByUntilNextTurn(FromPlayer(BindPlayer("unbound")), AnyPermanent, You),
	} {
		c := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid player declaration")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	p := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 3), b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	for _, effect := range []ResolvingEffect{
		CantBeAttackedExceptByUntilNextTurn(FromPlayer(BindPlayer("absent")), Not(AnyPermanent), You),
		CantBeAttackedExceptByUntilNextTurn(You, Not(AnyPermanent), FromPlayer(BindPlayer("absent"))),
	} {
		if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if !p.CanDeclareAsAttacker(g) {
		t.Fatal("absent player installed rule")
	}
}

type playerRestrictionAttacker struct {
	*BasePlayer
	attackers []uuid.UUID
}

func (p *playerRestrictionAttacker) DeclareAttackers(*Game) []uuid.UUID { return p.attackers }

func TestPlayerAttackRestrictionDeclarationPaths(t *testing.T) {
	for _, path := range []string{"native", "explicit"} {
		t.Run(path, func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &playerRestrictionAttacker{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			g.SetTurn(3)
			p := g.PutOnBattlefield(NewCreature("Ground", "", 2, 3), b.PlayerID())
			q := g.PutOnBattlefield(NewCreature("Air", "", 2, 3, WithKeyword(core.Flying)), b.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			q.RevokeBaseAttr(core.AttrSummonSick)
			b.attackers = []uuid.UUID{p.ID(), q.ID()}
			if err := ApplyEffect(g, CantBeAttackedExceptByUntilNextTurn(You, HasKeywordFilter(core.Flying), You), uuid.New(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.SetActivePlayerIndex(1)
			if path == "native" {
				g.doDeclareAttackers()
			} else {
				g.ExecuteAttackers(b.PlayerID(), b.attackers)
			}
			if g.GetCombat().IsAttacking(p.ID()) || !g.GetCombat().IsAttacking(q.ID()) || g.FindPermanent(p.ID()).Tapped {
				t.Fatal("attack restriction bypassed declaration")
			}
			if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), q.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if !g.GetCombat().IsAttacking(q.ID()) {
				t.Fatal("later ability loss removed attacker")
			}
		})
	}
}
