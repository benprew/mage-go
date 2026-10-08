package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedFilteredCountReferences(t *testing.T) {
	for _, ref := range []PlayerRef{ControllerOf(This), OwnerOf(This), ControllerOfWithLKI(This)} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control"} {
			for name, body := range map[string]ResolvingEffect{
				"hand":        GainLifeFor(You, CountMatchingHand(ref, CardFilter{})),
				"graveyard":   GainLifeFor(You, CountGraveyardCards(ref, CardFilter{})),
				"battlefield": GainLifeFor(You, CountPermanents(PermanentsControlledBy(ref, IsCreature))),
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
							t.Fatal("filtered count lost saved source")
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

func TestWaitingDelayedFilteredCountPurityAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	a.AddToHand(NewArtifact("Hand", ""))
	a.AddToGraveyard(NewArtifact("Graveyard", ""))
	reads := 0
	cardFilter := NewCardFilter("must not run", func(Card) bool { reads++; return true })
	permanentFilter := NewPermanentFilter("must not run", func(*Permanent, *Game) bool { reads++; return true })
	ref := ControllerOfWithLKI(This)
	controlled := PermanentsControlledBy(ref, permanentFilter)
	for name, value := range map[string]ResolutionValue{
		"hand":         CountMatchingHand(ref, cardFilter),
		"graveyard":    CountGraveyardCards(ref, cardFilter),
		"controller":   CountPermanents(controlled),
		"owner":        CountPermanents(PermanentsOwnedBy(ref, permanentFilter)),
		"filter":       CountPermanents(FilterPermanentSet(controlled, permanentFilter)),
		"union":        CountPermanents(UnionPermanents(MatchingPermanents(permanentFilter), controlled)),
		"intersection": CountPermanents(IntersectPermanents(controlled, MatchingPermanents(permanentFilter))),
		"limit":        CountPermanents(ManaValueAtMost(waitingMultiEmptySet{&reads}, CountMatchingHand(ref, cardFilter))),
	} {
		t.Run(name, func(t *testing.T) {
			g.triggers.SetDelayed(nil)
			body := GainLifeFor(You, MultiplyValues(Fixed(0), value))
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
					t.Fatal("query evaluated filtered count or lost zero-value declaration")
				}
			}
		})
	}
	g.triggers.SetDelayed(nil)
	for _, value := range []ResolutionValue{
		CountMatchingHand(ControllerOfWithLKI(AttachedPermanent), cardFilter),
		CountGraveyardCards(You, cardFilter),
		CountPermanents(waitingMultiEmptySet{&reads}),
		CountPermanents(MatchingPermanents(permanentFilter)),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, value)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred an attached, implicit or opaque source")
	}
	for _, value := range []ResolutionValue{
		CountMatchingHand(nil, cardFilter), CountGraveyardCards(nil, cardFilter), CountPermanents(nil), CountPermanents(PermanentsControlledBy(nil, permanentFilter)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, GainLifeFor(You, value)))))) == nil {
			t.Fatal("invalid count accepted")
		}
	}
}
