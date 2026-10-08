package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedManaPaymentReferences(t *testing.T) {
	for _, payment := range []string{"power", "toughness", "mana", "information", "counters", "counter-information", "arithmetic", "paid-body", "unpaid-body"} {
		for _, location := range []string{"body", "nested", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(fmt.Sprintf("%s/%s/%s", payment, location, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					noop := GainLifeFor(You, Fixed(0))
					amount := PowerOf(This)
					switch payment {
					case "toughness":
						amount = ToughnessOf(This)
					case "mana":
						amount = ManaValueOf(This)
					case "information":
						amount = PowerOfWithLKI(This)
					case "counters":
						amount = CountersOn(This, core.Charge)
					case "counter-information":
						amount = CountersOnWithLKI(This, core.Charge)
					case "arithmetic":
						amount = MaxValue(Fixed(0), AddValues(Fixed(1), MultiplyValues(Fixed(0), ManaValueOfWithLKI(This))))
					case "paid-body", "unpaid-body":
						amount = Fixed(100)
					}
					body := IfManaPaid(You, core.ParseManaCost("{X}"), amount, noop)
					if payment == "paid-body" {
						body = IfManaPaid(You, core.ParseManaCost("{X}"), amount, DealDamageFrom(This, You, Fixed(0)))
					}
					if payment == "unpaid-body" {
						body = UnlessManaPaid(You, core.ParseManaCost("{X}"), amount, DealDamageFrom(This, You, Fixed(0)))
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

func TestWaitingDelayedManaPaymentBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", "{U}"), a.PlayerID())
	payerReads, amountReads := 0, 0
	body := IfManaPaid(waitingPaymentPayer{PlayerRef: You, calls: &payerReads}, core.ParseManaCost("{X}"), waitingMultiAmount{&amountReads}, DealDamageFrom(This, You, Fixed(0)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || payerReads != 0 || amountReads != 0 {
			t.Fatal("query resolved a payment operand")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if payerReads != 1 || amountReads != 1 {
		t.Fatal("operands were not read once at resolution")
	}
	for _, amount := range []ResolutionValue{
		PowerOf(AttachedPermanent), ManaValueOfWithLKI(AttachedPermanent),
		CountersOnWithLKI(AttachedPermanent, core.Charge),
		LifeTotalOf(ControllerOf(This)), ValueAtResolution(XValue()),
		AddValues(Fixed(0), Fixed(1)), waitingMultiAmount{&amountReads},
	} {
		g.triggers.SetDelayed(nil)
		body := IfManaPaid(ControllerOf(This), core.ParseManaCost("{X}"), amount, AtNext(EachEndStep, Destroy(This)))
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("unrelated or unaudited operand declared the source")
		}
	}
	for _, body := range []ResolvingEffect{
		IfManaPaid(nil, core.ParseManaCost("{X}"), Fixed(0), Destroy(This)),
		UnlessManaPaid(You, core.ParseManaCost("{X}"), nil, Destroy(This)),
		IfManaPaid(You, core.ParseManaCost("{X}"), PowerOf(This), nil),
		IfManaPaid(You, core.ParseManaCost("{1}"), PowerOf(This), Destroy(This)),
		IfManaPaid(You, core.ParseManaCost("{X}"), AddValues(PowerOf(This), nil), Destroy(This)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid mana payment accepted")
		}
	}
}
