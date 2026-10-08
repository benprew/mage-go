package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedActionPaymentReferences(t *testing.T) {
	for _, payment := range []string{"damage-source", "damage-recipient", "destroy", "paid-body", "unpaid-body"} {
		for _, location := range []string{"body", "nested", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(fmt.Sprintf("%s/%s/%s", payment, location, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					read := DealDamageFrom(This, You, Fixed(0))
					noop := GainLifeFor(You, Fixed(0))
					cost := GainLifePayment(You, Fixed(0))
					paid, unpaid := noop, noop
					switch payment {
					case "damage-source":
						cost = DamagePayment(DamageSourceOf(This), You, Fixed(0))
					case "damage-recipient":
						cost = DamagePayment(DamageSourceOf(AttachedPermanent), This, Fixed(0))
					case "destroy":
						cost = DestroyPayment(This)
					case "paid-body":
						paid = read
					case "unpaid-body":
						unpaid = read
					}
					body := PayActionOrElse(You, cost, paid, unpaid)
					if location == "nested" {
						body = PayActionOrElse(You, GainLifePayment(You, Fixed(0)), Sequence(body), noop)
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

func TestWaitingDelayedActionPaymentPurityAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	payerReads, amountReads, paymentReads := 0, 0, 0
	payer := waitingPaymentPayer{PlayerRef: You, calls: &payerReads}
	payment := DamagePayment(DamageSourceOf(This), You, waitingMultiAmount{&amountReads})
	body := PayActionOrElse(payer, payment, GainLifeFor(You, Fixed(0)), nil)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || payerReads != 0 || amountReads != 0 {
			t.Fatal("query prepared or resolved payment")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if payerReads != 1 || amountReads != 1 {
		t.Fatal("payment did not prepare only at resolution")
	}
	for _, effect := range []ResolvingEffect{
		PayActionOrElse(ControllerOf(AttachedPermanent), GainLifePayment(You, Fixed(0)), GainLifeFor(You, Fixed(0)), nil),
		PayActionOrElse(You, GainLifePayment(ControllerOf(AttachedPermanent), Fixed(0)), GainLifeFor(You, Fixed(0)), nil),
		PayActionOrElse(You, GainLifePayment(You, Fixed(0)), AtNext(EachEndStep, Destroy(This)), nil),
		PayActionOrElse(You, waitingOpaqueActionPayment{payment, &paymentReads}, GainLifeFor(You, Fixed(0)), nil),
	} {
		g.triggers.SetDelayed(nil)
		if err := ApplyEffect(g, AtNext(EachEndStep, effect), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 || paymentReads != 0 {
			t.Fatal("query inferred an unaudited reference or called a payment")
		}
	}
	for _, effect := range []ResolvingEffect{
		PayActionOrElse(nil, DestroyPayment(This), Destroy(This), nil),
		PayActionOrElse(You, nil, Destroy(This), nil),
		PayActionOrElse(You, DestroyPayment(nil), Destroy(This), nil),
		PayActionOrElse(You, DamagePayment(nil, You, Fixed(1)), Destroy(This), nil),
		PayActionOrElse(You, GainLifePayment(You, Fixed(0)), nil, nil),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, effect))))) == nil {
			t.Fatal("invalid action payment accepted")
		}
	}
}

type waitingOpaqueActionPayment struct {
	ResolutionActionPayment
	calls *int
}

func (p waitingOpaqueActionPayment) prepareResolutionAction(ctx *EffectContext) (preparedResolutionAction, bool) {
	*p.calls++
	return p.ResolutionActionPayment.prepareResolutionAction(ctx)
}
