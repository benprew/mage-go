package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type attackPaymentIdentityPlayer struct {
	*BasePlayer
	attacker uuid.UUID
}

func (p *attackPaymentIdentityPlayer) DeclareAttackers(*Game) []uuid.UUID {
	return []uuid.UUID{p.attacker}
}
func (*attackPaymentIdentityPlayer) ChooseMayAbility(string) bool { return true }

func TestAttackPaymentSourceSacrifice(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprint(native), func(t *testing.T) {
			a := &attackPaymentIdentityPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(1)
			card := NewCreature("Payment source", "", 2, 3, WithStaticAbility(SourceCantAttackUnlessPays(SacrificeNMatchingIncludingSourceCost(1, IsCreature, "Sacrifice a creature"))))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			a.attacker = p.ID()
			if native {
				g.RunStepWithPriority(core.DeclareAttackers)
			} else {
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			}
			if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("source did not pay")
			}
			if g.GetCombat().IsAttacking(p.ID()) || g.HasAttackedThisTurn(p.ID()) {
				t.Fatal("absent source declared an attack")
			}
		})
	}
}

func TestAttackPaymentCurrentIdentity(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	p := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 3), a.PlayerID())
	original := *p
	if g.attackerAfterPayment(&original, a.PlayerID()) == nil {
		t.Fatal("live identity rejected")
	}
	clone := g.Clone()
	clone.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if clone.attackerAfterPayment(&original, a.PlayerID()) != nil || g.attackerAfterPayment(&original, a.PlayerID()) == nil {
		t.Fatal("control or clone identity")
	}
	phased := g.PhaseOut(p.ID())
	if g.attackerAfterPayment(&original, a.PlayerID()) != nil {
		t.Fatal("phased source retained")
	}
	g.PhaseIn(phased)
	if g.attackerAfterPayment(&original, a.PlayerID()) == nil {
		t.Fatal("phase cycle changed incarnation")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.PutOnBattlefield(p.Card, a.PlayerID())
	if g.attackerAfterPayment(&original, a.PlayerID()) != nil {
		t.Fatal("returned source replaced paid source")
	}
}
