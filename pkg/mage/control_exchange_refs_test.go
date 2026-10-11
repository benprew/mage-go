package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestReferencedControlExchangeIdentity(t *testing.T) {
	for _, change := range []string{"none", "same", "blocked", "phase", "phase-cycle", "return", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, first := controllerTestGame(t)
			card := NewArtifact("Second", "")
			card.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(card, b.PlayerID())
			firstBinding, secondBinding := BindPermanent("first"), BindPermanent("second")
			ctx := &EffectContext{Game: g, SourceID: first.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, firstBinding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			ctx.SourceID = second.ID()
			if err := Snapshot(This, secondBinding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "same":
				g.AddControlEffect(ControlEffectSpec{TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "blocked":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.AttrCantChangeControl)
			case "phase":
				g.PhaseOut(second.ID())
			case "phase-cycle":
				g.PhaseOut(second.ID())
				g.PhaseIn([]uuid.UUID{second.ID()})
			case "return":
				g.BouncePermanentToHand(second)
				b.RemoveFromHand(card.ID())
				second = g.PutOnBattlefield(card, b.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			beforeFirst, beforeSecond := g.FindPermanentIncludingPhased(first.ID()).ControllerID(), g.FindPermanentIncludingPhased(second.ID()).ControllerID()
			if err := ExchangePermanentControl(From(firstBinding), From(secondBinding)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			success := change == "none" || change == "phase-cycle"
			wantFirst, wantSecond := beforeFirst, beforeSecond
			if success {
				wantFirst, wantSecond = beforeSecond, beforeFirst
			}
			if g.FindPermanentIncludingPhased(first.ID()).ControllerID() != wantFirst || g.FindPermanentIncludingPhased(second.ID()).ControllerID() != wantSecond {
				t.Fatal("exchange ignored identity or changed only one controller")
			}
		})
	}
}

func TestReferencedControlExchangeDeclaration(t *testing.T) {
	for _, refs := range [][2]PermanentRef{{nil, This}, {This, nil}, {This, DeclarePermanentTarget("missing", AnyPermanent)}} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), ExchangePermanentControl(refs[0], refs[1]))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid reference accepted")
		}
	}
}

func TestReferencedControlExchangeWaitingSource(t *testing.T) {
	for _, side := range []string{"first", "second"} {
		t.Run(side, func(t *testing.T) {
			g, a, b, p := controllerTestGame(t)
			target := DeclarePermanentTarget("other", AnyPermanent)
			first, second := PermanentRef(This), PermanentRef(target)
			if side == "second" {
				first, second = second, first
			}
			if !effectReadsSourcePermanent(ExchangePermanentControl(first, second)) {
				t.Fatal("source dependency missing")
			}
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNext(EachEndStep, ExchangePermanentControl(This, This)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(p.ID())
			returned := g.PutOnBattlefield(p.Card, b.PlayerID())
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				got := game.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("waiting query lost exact source")
				}
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if clone.FindPermanent(returned.ID()).ControllerID() != b.PlayerID() || len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("return or clone changed lifetime")
			}
		})
	}
}

func TestReferencedControlExchangeNativeDuration(t *testing.T) {
	g, a, b, first := controllerTestGame(t)
	first.Card.SetOwner(a.PlayerID())
	card := NewArtifact("Second", "")
	card.SetOwner(b.PlayerID())
	second := g.PutOnBattlefield(card, b.PlayerID())
	if !g.ExchangeControl(first.ID(), second.ID(), first.ID()) {
		t.Fatal("exchange failed")
	}
	clone := g.Clone()
	clone.BouncePermanentToHand(clone.FindPermanent(first.ID()))
	clone.GetPlayer(a.PlayerID()).RemoveFromHand(first.ID())
	returned := clone.PutOnBattlefield(first.Card, a.PlayerID())
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(second.ID()).ControllerID() != a.PlayerID() || returned.ControllerID() != a.PlayerID() {
		t.Fatal("clone lost source independence or target identity")
	}
	if g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(second.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("clone changed original effects")
	}
	g.BouncePermanentToHand(second)
	b.RemoveFromHand(card.ID())
	returned = g.PutOnBattlefield(card, b.PlayerID())
	g.ApplyContinuousEffects()
	if returned.ControllerID() != b.PlayerID() || g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("returned target inherited exchange or ended the other effect")
	}
}
