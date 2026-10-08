package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedSacrificePaymentReferences(t *testing.T) {
	for _, payment := range []string{"paid", "unpaid", "paid-body", "unpaid-body"} {
		for _, location := range []string{"body", "nested", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(fmt.Sprintf("%s/%s/%s", payment, location, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					noop := GainLifeFor(You, Fixed(0))
					subject := PermanentRef(This)
					continuation := noop
					if payment == "paid-body" || payment == "unpaid-body" {
						subject = AttachedPermanent
						continuation = Destroy(This)
					}
					body := IfSacrificePaid(You, subject, continuation)
					if payment == "unpaid" || payment == "unpaid-body" {
						body = UnlessSacrificePaid(You, subject, continuation)
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
				})
			}
		}
	}
}

func TestWaitingDelayedSacrificePaymentBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	payer := waitingPaymentPayer{PlayerRef: You, calls: &reads}
	body := IfSacrificePaid(payer, This, GainLifeFor(You, Fixed(1)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || a.Life() != 20 {
			t.Fatal("query resolved an impossible payment")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 || a.Life() != 20 {
		t.Fatal("payment did not read its payer at resolution")
	}
	for _, effect := range []ResolvingEffect{
		IfSacrificePaid(ControllerOf(AttachedPermanent), AttachedPermanent, GainLifeFor(You, Fixed(0))),
		UnlessSacrificePaid(You, AttachedPermanent, AtNext(EachEndStep, Destroy(This))),
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
		IfSacrificePaid(nil, This, GainLifeFor(You, Fixed(1))),
		IfSacrificePaid(You, nil, GainLifeFor(You, Fixed(1))),
		UnlessSacrificePaid(You, This, nil),
		IfSacrificePaid(You, From(BindPermanent("unbound")), GainLifeFor(You, Fixed(1))),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, effect))))) == nil {
			t.Fatal("invalid sacrifice declaration accepted")
		}
	}
}
