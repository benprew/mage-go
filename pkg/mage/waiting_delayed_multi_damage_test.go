package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedMultiDamageSources(t *testing.T) {
	for _, source := range []struct {
		name string
		ref  DamageSourceRef
	}{{"this", This}, {"permanent", DamageSourceOf(This)}} {
		for _, kind := range []string{"collection", "players", "packets"} {
			for _, shape := range []string{"direct", "sequence", "mode"} {
				for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
					t.Run(source.name+"/"+kind+"/"+shape+"/"+state, func(t *testing.T) {
						a, b := NewBasePlayer("A"), NewBasePlayer("B")
						g := NewGame(a, b)
						body := waitingMultiDamageBody(kind, source.ref, Fixed(2))
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
						wantA, wantB := 18, 18
						if kind == "packets" {
							wantA, wantB = 16, 20
						}
						if a.Life() != wantA || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
							t.Fatal("delayed damage lost its original controller or source")
						}
					})
				}
			}
		}
	}
}

func waitingMultiDamageBody(kind string, source DamageSourceRef, amount ResolutionValue) ResolvingEffect {
	switch kind {
	case "collection":
		return DealDamageEach(source, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), amount)
	case "players":
		return DealDamageEachPlayer(source, EachPlayer, BindPlayer("member"), amount)
	default:
		return DealDamagePackets(source, DamagePacketTo(You, amount), DamagePacketTo(You, amount))
	}
}

func TestWaitingDelayedMultiDamageBoundaries(t *testing.T) {
	for _, kind := range []string{"collection", "players", "packets"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			for _, source := range []DamageSourceRef{DamageSourceOf(From(BindPermanent("other"))), FromDamageSource(BindDamageSource("other"))} {
				if err := ApplyEffect(g, AtNext(EachEndStep, waitingMultiDamageBody(kind, source, Fixed(0))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("bound source implied a self reference")
			}
			if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, waitingMultiDamageBody(kind, This, Fixed(0)))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("nested future body supplied an outer reference")
			}
			calls := 0
			if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(waitingMultiDamageBody(kind, This, waitingMultiAmount{&calls}), waitingMultiDamageBody(kind, DamageSourceOf(This), Fixed(0)))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 1 || calls != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("query executed an amount or lost a zero-damage source")
			}
			for _, source := range []DamageSourceRef{nil, DamageSourceOf(nil)} {
				if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, waitingMultiDamageBody(kind, source, Fixed(1))))))) == nil {
					t.Fatal("invalid source accepted")
				}
			}
		})
	}
}

type waitingMultiAmount struct{ calls *int }

func (v waitingMultiAmount) resolveValue(*EffectContext) int { *v.calls++; return 0 }
func (waitingMultiAmount) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value"}
}

func TestWaitingDelayedMultiDamageEmptyRecipients(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	calls := 0
	body := DealDamageEach(This, PermanentDamageRecipients(waitingMultiEmptySet{&calls}), Fixed(2))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || calls != 0 {
		t.Fatal("query evaluated recipients")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if calls != 1 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("empty collection changed damage")
	}
}

type waitingMultiEmptySet struct{ calls *int }

func (waitingMultiEmptySet) permanentSet()                                   {}
func (v waitingMultiEmptySet) resolvePermanents(*EffectContext) []*Permanent { *v.calls++; return nil }
func (waitingMultiEmptySet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set"}
}

func (waitingMultiAmount) Text() string { return "zero" }
