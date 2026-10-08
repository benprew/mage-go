package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedDamageOperandReferences(t *testing.T) {
	for _, ref := range []PlayerRef{ControllerOf(This), OwnerOf(This), ControllerOfWithLKI(This)} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control"} {
			for name, body := range map[string]ResolvingEffect{
				"single-recipient": DealDamageFrom(DamageSourceOf(AttachedPermanent), ref, Fixed(0)),
				"single-value":     DealDamageFrom(DamageSourceOf(AttachedPermanent), You, MultiplyValues(Fixed(0), HandSizeOf(ref))),
				"each-value":       DealDamageEach(DamageSourceOf(AttachedPermanent), PermanentDamageRecipients(MatchingPermanents(IsCreature)), HandSizeOf(ref)),
				"players-value":    DealDamageEachPlayer(DamageSourceOf(AttachedPermanent), EachPlayer, BindPlayer("member"), HandSizeOf(ref)),
				"packet-recipient": DealDamagePackets(DamageSourceOf(AttachedPermanent), DamagePacketTo(ref, Fixed(0))),
				"packet-value":     DealDamagePackets(DamageSourceOf(AttachedPermanent), DamagePacketTo(You, HandSizeOf(ref))),
			} {
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
							t.Fatal("typed player operand lost saved source")
						}
						if got[0].HasColor(core.Red) {
							t.Fatal("returned source supplied current facts")
						}
						if state == "control" && got[0].Controller() != b.PlayerID() {
							t.Fatal("lost current controller")
						}
						got[0].card.AddType(core.TypeEnchantment)
						if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
							t.Fatal("query shares card facts")
						}
					}
					if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 0 {
						t.Fatal("query executed body")
					}
					clone := g.Clone()
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("waiting lifetime shared with clone")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedDamageOperandPurityAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	source := DamageSourceOf(AttachedPermanent)
	for _, body := range []ResolvingEffect{
		DealDamageFrom(source, ControllerOfWithLKI(This), waitingMultiAmount{&reads}),
		DealDamageEach(source, PermanentDamageRecipients(waitingMultiEmptySet{&reads}), HandSizeOf(ControllerOfWithLKI(This))),
		DealDamagePackets(source, DamagePacketTo(ControllerOfWithLKI(This), waitingMultiAmount{&reads})),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query evaluated a value or collection")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{
		DealDamageFrom(source, ControllerOfWithLKI(AttachedPermanent), waitingMultiAmount{&reads}),
		DealDamageEachPlayer(source, EachPlayer, BindPlayer("member"), waitingMultiAmount{&reads}),
		AtNext(EachEndStep, DealDamageFrom(source, ControllerOfWithLKI(This), Fixed(0))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("inferred an opaque or future declaration")
	}
	for _, body := range []ResolvingEffect{
		DealDamageFrom(source, ControllerOf(nil), Fixed(0)),
		DealDamageFrom(source, You, nil),
		DealDamageEach(source, PermanentDamageRecipients(MatchingPermanents(IsCreature)), HandSizeOf(nil)),
		DealDamageEachPlayer(source, EachPlayer, BindPlayer("member"), nil),
		DealDamagePackets(source, DamagePacketTo(nil, Fixed(0))),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid damage declaration accepted")
		}
	}
}
