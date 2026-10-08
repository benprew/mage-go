package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedHandActionReferences(t *testing.T) {
	for _, ref := range []PlayerRef{ControllerOf(This), OwnerOf(This), ControllerOfWithLKI(This)} {
		for name, body := range map[string]ResolvingEffect{
			"reveal":        RevealHandFor(ref),
			"matching":      DiscardMatchingHand(ref, CardFilter{}),
			"chosen":        Discard(ref, Fixed(1)),
			"random":        DiscardAtRandom(ref, Fixed(1)),
			"chosen amount": Discard(You, MultiplyValues(Fixed(0), HandSizeOf(ref))),
			"random amount": DiscardAtRandom(You, MultiplyValues(Fixed(0), HandSizeOf(ref))),
		} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control"} {
				t.Run(fmt.Sprintf("%T/%s/%s", ref, name, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					card := NewArtifact("Source", "{U}")
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					original := g.CaptureDamageSource(p.ID())
					delay := AtNext(EachEndStep, Sequence(body))
					if err := ValidateCardDefinition("Source", NewArtifact("Source", "", WithAbility(Activate(Costs(), delay)))); err != nil {
						t.Fatal(err)
					}
					if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
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
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 1 || !got[0].SameObject(original) {
							t.Fatal("hand action lost saved source")
						}
						if got[0].HasColor(core.Red) {
							t.Fatal("returned source supplied new facts")
						}
						if state == "control" && got[0].Controller() != b.PlayerID() {
							t.Fatal("lost current controller")
						}
						got[0].card.AddType(core.TypeEnchantment)
						if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
							t.Fatal("shared query snapshot")
						}
					}
					if len(g.PublicReveals()) != 0 || len(a.Graveyard()) != 0 || g.StackSize() != 0 {
						t.Fatal("query ran hand action")
					}
					clone := g.Clone()
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("shared waiting lifetime")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedHandActionPurityAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	a.AddToHand(NewCreature("Hand", "", 1, 1))
	reads := 0
	filter := NewCardFilter("must not run", func(Card) bool { reads++; return true })
	ref := ControllerOfWithLKI(This)
	probe := waitingPaymentPayer{PlayerRef: ref, calls: &reads}
	for _, body := range []ResolvingEffect{
		DiscardMatchingHand(ref, filter),
		DiscardAtRandom(You, MultiplyValues(Fixed(0), CountMatchingHand(ref, filter))),
		Discard(You, MultiplyValues(Fixed(0), CountMatchingHand(ref, filter))),
	} {
		g.triggers.SetDelayed(nil)
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.SetRandomResults([]int{7})
		for range 3 {
			if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || len(a.Hand()) != 1 {
				t.Fatal("query evaluated hand operand")
			}
		}
		if g.RandIntn(10) != 7 {
			t.Fatal("query consumed randomness")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{
		RevealHandFor(probe), DiscardMatchingHand(probe, filter), Discard(probe, Fixed(1)), DiscardAtRandom(probe, Fixed(1)),
		RevealHandFor(ControllerOfWithLKI(AttachedPermanent)), DiscardAtRandom(You, HandSizeOf(You)),
		AtNext(EachEndStep, RevealHandFor(ref)),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("inferred opaque, attached or future reference")
	}
	for _, body := range []ResolvingEffect{RevealHandFor(nil), DiscardMatchingHand(nil, filter), Discard(nil, Fixed(1)), DiscardAtRandom(nil, Fixed(1)), DiscardAtRandom(You, nil)} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid hand operand accepted")
		}
	}
}
