package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedDamageBodySources(t *testing.T) {
	for _, source := range []struct {
		name string
		ref  DamageSourceRef
	}{{"this", This}, {"permanent", DamageSourceOf(This)}} {
		for _, shape := range []string{"direct", "sequence", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(source.name+"/"+shape+"/"+state, func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					body := DealDamageFrom(source.ref, You, Fixed(2))
					var delay = AtNext(EachEndStep, body)
					switch shape {
					case "sequence":
						delay = AtNext(EachEndStep, Sequence(Sequence(body)))
					case "mode":
						delay = AtNextModes(EachEndStep, ModeOf("First damage", body), ModeOf("Second damage", body))
					}
					card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(), delay)))
					if err := ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					original := g.CaptureDamageSource(p.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("unresolved activation supplied a waiting reference")
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
					case "phased", "cycle":
						ids := g.PhaseOut(p.ID())
						if state == "cycle" {
							g.PhaseIn(ids)
						}
					case "color":
						g.ChangeObjectColor(p.ID(), core.Red)
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(original) {
						t.Fatal("damage body lost its exact source")
					}
					if got[0].HasColor(core.Red) != (state == "color") {
						t.Fatal("wrong current or departure information")
					}
					if state == "control" && got[0].Controller() != b.PlayerID() {
						t.Fatal("query lost current controller")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) || b.Life() != 20 {
						t.Fatal("query mutated facts or executed the body")
					}
					clone := g.Clone()
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("waiting lifetime or clone isolation")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					clone.PutTriggersOnStack()
					clone.ResolveStack()
					if a.Life() != 18 || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 18 {
						t.Fatal("delayed damage lost its original controller or source")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedDamageBodyBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	for _, body := range []ResolvingEffect{
		GainLifeFor(You, Fixed(1)),
		DealDamageFrom(DamageSourceOf(From(BindPermanent("other"))), You, Fixed(0)),
		DealDamageFrom(FromDamageSource(BindDamageSource("other")), You, Fixed(0)),
		AtNext(EachEndStep, DealDamageFrom(This, You, Fixed(0))),
	} {
		g.triggers.SetDelayed(nil)
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("unrelated or future source supplied a reference")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, incarnation := range []uuid.UUID{uuid.Nil, uuid.New()} {
		g.RegisterDelayedTrigger(&DelayedTrigger{SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: incarnation, Effects: []Effect{DealDamageFrom(This, You, Fixed(0))}})
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{SourceID: p.ID(), sourceIncarnation: p.incarnationID, Effects: []Effect{DealDamageFrom(This, You, Fixed(0))}})
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("invalid or uncaptured incarnation supplied a reference")
	}
	if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(DealDamageFrom(This, You, Fixed(0)), DealDamageFrom(DamageSourceOf(This), You, Fixed(0)))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || a.Life() != 20 {
		t.Fatal("zero damage or duplicate source declaration")
	}
	for _, body := range []ResolvingEffect{DealDamageFrom(nil, You, Fixed(1)), DealDamageFrom(DamageSourceOf(nil), You, Fixed(1))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid source declaration accepted")
		}
	}
}

func TestWaitingDelayedDamageBodyPermanentRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 3, WithAbility(Activate(Costs(), AtNext(EachEndStep, DealDamageFrom(DamageSourceOf(This), This, Fixed(1))))))
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(p.ID())) || p.Damage != 0 {
		t.Fatal("permanent recipient changed source enumeration")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Damage != 1 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("permanent damage body failed")
	}
}
