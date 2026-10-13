package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentPaymentPlayer struct {
	*BasePlayer
	offers     []string
	accept     func(string) bool
	chooseLast bool
}

func (p *permanentPaymentPlayer) ChooseMayAbility(reason string) bool {
	p.offers = append(p.offers, reason)
	return p.accept != nil && p.accept(reason)
}
func (p *permanentPaymentPlayer) ChoosePermanent(candidates []*Permanent, reason string, g GameReader) *Permanent {
	if p.chooseLast {
		return candidates[len(candidates)-1]
	}
	return p.BasePlayer.ChoosePermanent(candidates, reason, g)
}

func TestPermanentPaymentOffersOrderAndRollback(t *testing.T) {
	for _, active := range []int{0, 1} {
		a := &permanentPaymentPlayer{BasePlayer: NewBasePlayer("A"), chooseLast: true}
		b := &permanentPaymentPlayer{BasePlayer: NewBasePlayer("B")}
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		g.PutOnBattlefield(NewLand("First"), a.PlayerID())
		g.PutOnBattlefield(NewLand("Second"), a.PlayerID())
		a.accept = func(reason string) bool { return strings.Contains(reason, "Second") }
		b.accept = func(reason string) bool { return strings.Contains(reason, "First") }
		result := BindPermanentSet("unpaid")
		effect := OfferCostForEachPermanent(UnionPermanents(MatchingPermanents(IsLand), MatchingPermanents(IsLand)), EachPlayer, Costs(LifePayCost(1)), result)
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 19 || b.Life() != 19 || len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
			t.Fatal("cross-controller payments or distinct visits")
		}
		all := append(append([]string{}, a.offers...), b.offers...)
		if len(all) != 3 {
			t.Fatal("offer did not stop at first successful payment", all)
		}
		if !strings.Contains(a.offers[0], "Second") || !strings.Contains(b.offers[0], "Second") && active == 1 {
			t.Fatal("controller's subject order was ignored")
		}
		a.accept = func(string) bool { return true }
		b.accept = func(string) bool { return true }
		a.offers = nil
		b.offers = nil
		a.SetLife(20)
		b.SetLife(20)
		if err := OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, Costs(LifePayCost(15), LifePayCost(15)), result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 20 || b.Life() != 20 || len(FromPermanentSet(result).resolvePermanents(ctx)) != 2 {
			t.Fatal("partial cost committed or unpaid visits lost")
		}
		a.accept = nil
		b.accept = nil
		if err := OfferCostForEachPermanent(SingletonPermanent(From(BindPermanent("missing"))), EachPlayer, Costs(), result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
			t.Fatal("empty producer retained older results")
		}
	}
}

func TestPermanentPaymentIdentityDelayAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Saved")
	card.SetOwner(b.PlayerID())
	subject := g.PutOnBattlefield(card, b.PlayerID())
	phased := g.PutOnBattlefield(NewLand("Phased"), a.PlayerID())
	g.PhaseOut(phased.ID())
	result := BindPermanentSet("unpaid")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, Costs(LifePayCost(1)), result)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(FromPermanentSet(result).resolvePermanents(ctx)) != 1 {
		t.Fatal("phased subject offered")
	}
	if err := AtNext(EachEndStep, DestroyEach(FromPermanentSet(result))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.PhaseOut(subject.ID())
	if len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
		t.Fatal("phased saved visit actionable")
	}
	g.PhaseIn([]uuid.UUID{subject.ID()})
	if len(FromPermanentSet(result).resolvePermanents(ctx)) != 1 {
		t.Fatal("phase cycle lost identity")
	}
	g.RemoveFromBattlefield(subject)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(returned.ID()) == nil {
		t.Fatal("return inherited old result")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.ResolveStack()
	if clone.FindPermanent(subject.ID()) != nil || g.FindPermanent(returned.ID()) == nil {
		t.Fatal("clone lost saved frame or changed original")
	}
}

func TestPermanentPaymentInvalid(t *testing.T) {
	result := BindPermanentSet("unpaid")
	for _, body := range []ResolvingEffect{
		OfferCostForEachPermanent(nil, EachPlayer, Costs(), result),
		OfferCostForEachPermanent(MatchingPermanents(IsLand), nil, Costs(), result),
		OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, CostGroup{}, result),
		OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, Costs(nil), result),
		OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, Costs(), nil),
		OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, Costs(), BindPermanentSet("")),
		DestroyEach(FromPermanentSet(result)),
	} {
		if err := ValidateCardDefinition("Bad", NewSorcery("Bad", "", Spell(body))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	body := Sequence(OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, Costs(), result), DestroyEach(FromPermanentSet(result)))
	if err := ValidateCardDefinition("Good", NewSorcery("Good", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
}

func TestPermanentPaymentSnapshotRecheckAndSimultaneous(t *testing.T) {
	a := &permanentPaymentPlayer{BasePlayer: NewBasePlayer("A")}
	b := &permanentPaymentPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
	old := g.PutOnBattlefield(NewLand("Old"), b.PlayerID())
	var returned, later *Permanent
	a.accept = func(reason string) bool {
		if strings.Contains(reason, "First") {
			a.SetLife(0)
			g.RemoveFromBattlefield(old)
			returned = g.PutOnBattlefield(old.Card, b.PlayerID())
			later = g.PutOnBattlefield(NewLand("Later"), b.PlayerID())
		}
		return true
	}
	b.accept = func(string) bool { return true }
	result := BindPermanentSet("unpaid")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := OfferCostForEachPermanent(MatchingPermanents(IsLand), EachPlayer, Costs(LifePayCost(1)), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 0 || b.Life() != 19 || len(a.offers) != 1 || len(b.offers) != 1 || len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
		t.Fatal("failed payment did not continue or collection changed during offers")
	}
	if g.FindPermanent(returned.ID()) == nil || g.FindPermanent(later.ID()) == nil {
		t.Fatal("changed visits were affected")
	}
	a.accept = nil
	b.accept = nil
	second := g.PutOnBattlefield(NewLand("Second"), a.PlayerID())
	events := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if event.SourceID != first.ID() && event.SourceID != second.ID() {
			return false
		}
		events++
		if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || reader.FindPermanent(first.ID()) == nil || reader.FindPermanent(second.ID()) == nil {
			t.Fatal("destruction was not simultaneous or lost departure information")
		}
		return false
	}))), a.PlayerID())
	effect := Sequence(OfferCostForEachPermanent(PermanentsControlledBy(You, IsLand), EachPlayer, Costs(LifePayCost(1)), result), DestroyEach(FromPermanentSet(result)))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if events != 2 || g.FindPermanent(later.ID()) == nil {
		t.Fatal("wrong batch or controller collection", events)
	}
}
