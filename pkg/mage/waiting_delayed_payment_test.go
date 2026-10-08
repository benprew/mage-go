package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedPaymentReferences(t *testing.T) {
	for _, paid := range []bool{true, false} {
		for _, location := range []string{"body", "both-paid", "both-unpaid", "nested", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(fmt.Sprintf("paid=%t/%s/%s", paid, location, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					branch := UnlessPaid
					if paid {
						branch = IfPaid
					}
					costs := Costs(LifePayCost(21))
					read := DealDamageFrom(This, You, Fixed(0))
					body := branch(You, costs, read)
					switch location {
					case "both-paid":
						body = PayOrElse(You, costs, read, GainLifeFor(You, Fixed(0)))
					case "both-unpaid":
						body = PayOrElse(You, costs, GainLifeFor(You, Fixed(0)), read)
					case "nested":
						body = branch(You, costs, Sequence(body))
					}
					delay := AtNext(EachEndStep, Sequence(body))
					if location == "mode" {
						delay = AtNextModes(EachEndStep, ModeOf("Condition", body), ModeOf("Duplicate", body))
					}
					card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(), delay)))
					if err := ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					saved := g.CaptureDamageSource(p.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("unresolved delay supplied references")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(p)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
							g.ChangeObjectColor(card.ID(), core.Red)
						}
					case "phased":
						g.PhaseOut(p.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					case "color":
						g.ChangeObjectColor(p.ID(), core.Red)
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) {
						t.Fatal("conditional reference lost source incarnation")
					}
					if a.Life() != 20 {
						t.Fatal("query evaluated a condition or body")
					}
					if got[0].HasColor(core.Red) != (state == "color") {
						t.Fatal("wrong current or departure facts")
					}
					if state == "control" && got[0].Controller() != b.PlayerID() {
						t.Fatal("wrong current controller")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
						t.Fatal("query shares mutable facts")
					}
					clone := g.Clone()
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("waiting lifetime or clone isolation")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					if a.Life() != 20 || b.Life() != 20 {
						t.Fatal("zero damage changed life")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedPaymentBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	for _, body := range []ResolvingEffect{
		IfPaid(ControllerOf(AttachedPermanent), Costs(LifePayCost(1)), GainLifeFor(You, Fixed(0))),
		UnlessPaid(You, Costs(), AtNext(EachEndStep, Destroy(This))),
		PayOrElse(You, Costs(), GainLifeFor(You, Fixed(0)), AtNext(EachEndStep, Destroy(This))),
	} {
		g.triggers.SetDelayed(nil)
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("attached payer or future body implied a branch self read")
		}
	}
	for _, body := range []ResolvingEffect{
		IfPaid(nil, Costs(), Destroy(This)),
		UnlessPaid(You, CostGroup{}, Destroy(This)),
		IfPaid(You, Costs(), nil),
		PayOrElse(You, Costs(), Destroy(This), nil),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid payment declaration accepted")
		}
	}
}

func TestWaitingDelayedPaymentQueryPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	payerReads, costReads, amountReads := 0, 0, 0
	payer := waitingPaymentPayer{PlayerRef: You, calls: &payerReads}
	cost := waitingPaymentCost{Cost: LifePayCost(21), calls: &costReads}
	body := PayOrElse(payer, Costs(cost), DealDamageFrom(This, You, waitingMultiAmount{&amountReads}), GainLifeFor(You, Fixed(0)))
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || payerReads != 0 || costReads != 0 || amountReads != 0 {
			t.Fatal("query evaluated payment or body")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if payerReads != 1 || costReads != 1 || amountReads != 0 {
		t.Fatal("payment did not run only at resolution")
	}
}

type waitingPaymentPayer struct {
	PlayerRef
	calls *int
}

func (p waitingPaymentPayer) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	*p.calls++
	return p.PlayerRef.resolvePlayer(ctx)
}

type waitingPaymentCost struct {
	Cost
	calls *int
}

func (c waitingPaymentCost) CanPay(source, controller uuid.UUID, g *Game) bool {
	*c.calls++
	return c.Cost.CanPay(source, controller, g)
}
