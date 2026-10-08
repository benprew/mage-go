package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCounterPaymentReferences(t *testing.T) {
	for _, payment := range []string{"fixed", "chosen", "mandatory", "paid-body", "unpaid-body"} {
		for _, location := range []string{"body", "nested", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(fmt.Sprintf("%s/%s/%s", payment, location, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					noop := GainLifeFor(You, Fixed(0))
					subject := PermanentRef(This)
					paid, unpaid := noop, noop
					if payment == "paid-body" || payment == "unpaid-body" {
						subject = AttachedPermanent
						if payment == "paid-body" {
							paid = Destroy(This)
						} else {
							unpaid = Destroy(This)
						}
					}
					body := PayCountersOrElse(You, subject, core.Charge, 0, payment != "mandatory", paid, unpaid)
					if payment == "chosen" {
						body = PayChosenCountersOrElse(You, subject, core.Charge, BindNumber("paid counters"), paid, unpaid)
					}
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
						t.Fatal("counter reference query changed life")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedCounterPaymentBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	payer := waitingPaymentPayer{PlayerRef: You, calls: &reads}
	body := PayCountersOrElse(payer, This, core.Charge, 100, true, GainLifeFor(You, Fixed(0)), nil)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query resolved payer or checked affordability")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 {
		t.Fatal("payer was not read at resolution")
	}
	for _, effect := range []ResolvingEffect{
		PayCountersOrElse(ControllerOf(AttachedPermanent), AttachedPermanent, core.Charge, 0, true, GainLifeFor(You, Fixed(0)), nil),
		PayCountersOrElse(You, AttachedPermanent, core.Charge, 0, true, AtNext(EachEndStep, Destroy(This)), nil),
	} {
		g.triggers.SetDelayed(nil)
		if err := ApplyEffect(g, AtNext(EachEndStep, effect), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("unrelated or future reference declared")
		}
	}
	for _, effect := range []ResolvingEffect{
		PayCountersOrElse(nil, This, core.Charge, 0, true, Destroy(This), nil),
		PayCountersOrElse(You, nil, core.Charge, 0, true, Destroy(This), nil),
		PayCountersOrElse(You, This, core.NumCounters, 0, true, Destroy(This), nil),
		PayCountersOrElse(You, This, core.Charge, -1, true, Destroy(This), nil),
		PayChosenCountersOrElse(You, This, core.Charge, nil, Destroy(This), nil),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, effect))))) == nil {
			t.Fatal("invalid counter declaration accepted")
		}
	}
}
