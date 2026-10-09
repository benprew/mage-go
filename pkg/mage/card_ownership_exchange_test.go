package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCardOwnershipExchange(t *testing.T) {
	for _, state := range []string{"normal", "same-owner", "same-card", "missing", "returned", "token", "disabled", "settled", "invalid-owner"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := anteEffectTestGame(t)
			first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(b.PlayerID(), "second")
			if state == "token" {
				second = NewToken("token", 1, 1, []core.CardType{core.TypeCreature}, nil)
				second.SetOwner(b.PlayerID())
			}
			a.AddToHand(first)
			b.AddToGraveyard(second)
			if state == "same-owner" {
				second.SetOwner(a.PlayerID())
			}
			if state == "invalid-owner" {
				second.SetOwner(uuid.New())
			}
			one, two := BindCard("first"), BindCard("second")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{
				one: {id: first.ID(), visit: a.handVisit(first.ID()), zone: core.ZoneHand},
				two: {id: second.ID(), visit: g.graveyardVisit(second.ID()), zone: core.ZoneGraveyard},
			}
			switch state {
			case "missing", "returned":
				b.RemoveFromGraveyard(second.ID())
				if state == "returned" {
					b.AddToGraveyard(second)
				}
			case "disabled":
				g.anteEnabled = false
			case "settled":
				g.anteSettled = true
			}
			beforeFirst, beforeSecond := first.Owner(), second.Owner()
			clone := g.Clone()
			events := 0
			watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(*core.GameEvent, GameReader, uuid.UUID, uuid.UUID) bool { events++; return false })
			g.PutOnBattlefield(NewArtifact("watcher", "", WithAbility(watcher)), a.PlayerID())
			events = 0
			right := FromCard(two)
			if state == "same-card" {
				right = FromCard(one)
			}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := ExchangeCardOwnership(FromCard(one), right).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			swapped := state == "normal"
			want := beforeFirst
			if swapped {
				want = beforeSecond
			}
			if g.FindCardAnywhere(first.ID()).Owner() != want {
				t.Fatal("first owner or partial exchange")
			}
			if c := g.FindCardAnywhere(second.ID()); c != nil {
				want = beforeSecond
				if swapped {
					want = beforeFirst
				}
				if c.Owner() != want {
					t.Fatal("second owner or partial exchange")
				}
			}
			if clone.FindCardAnywhere(first.ID()).Owner() != beforeFirst || events != 0 || g.CardZone(first.ID()) != core.ZoneHand {
				t.Fatal("exchange changed clone or zone")
			}
			if _, ok := FromCard(one).resolveCard(ctx); !ok {
				t.Fatal("ownership ended hand visit")
			}
			if state != "missing" && state != "returned" {
				if _, ok := FromCard(two).resolveCard(ctx); !ok {
					t.Fatal("ownership ended graveyard visit")
				}
			}
			if swapped {
				b.SetLife(0)
				changes, err := g.AnteResult()
				if err != nil || len(changes) != 2 {
					t.Fatal("settlement lost ownership changes", changes, err)
				}
			} else if len(g.originalOwners) != 0 {
				t.Fatal("failed exchange changed ownership history")
			}
		})
	}
}

func TestCardOwnershipExchangeDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{ExchangeCardOwnership(nil, DeclareAnteCardTarget("other")), ExchangeCardOwnership(DeclareAnteCardTarget("one"), nil), ExchangeCardOwnership(FromCard(BindCard("unbound")), DeclareAnteCardTarget("other"))} {
		if ValidateCardDefinition("invalid", NewSorcery("invalid", "", WithAbility(Spell(effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestCardOwnershipExchangeExileAndAnte(t *testing.T) {
	for _, state := range []string{"normal", "returned-exile", "returned-ante", "hidden", "same-owner", "current-owner"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := anteEffectTestGame(t)
			first, second := ownedTestCard(a.PlayerID(), "exile"), ownedTestCard(b.PlayerID(), "ante")
			g.ExileCard(first, a.PlayerID())
			b.AddToAnte(second)
			binding := BindCard("exile")
			visit := g.FindExiledCard(first.ID()).Visit
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: first.ID(), visit: visit, zone: core.ZoneExile}}
			target := DeclareAnteCardTarget("ante")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: second.ID(), incarnation: g.anteVisit(second.ID())}}}}
			switch state {
			case "returned-exile":
				g.RemoveFromExile(first.ID())
				g.ExileCard(first, a.PlayerID())
			case "returned-ante":
				b.RemoveFromAnte(second.ID())
				b.AddToAnte(second)
			case "hidden":
				g.FindExiledCard(first.ID()).FaceDown = true
			case "same-owner":
				if err := g.ChangeOwner(second.ID(), a.PlayerID()); err != nil {
					t.Fatal(err)
				}
			case "current-owner":
				if err := g.ChangeOwner(first.ID(), b.PlayerID()); err != nil {
					t.Fatal(err)
				}
				if err := g.ChangeOwner(second.ID(), a.PlayerID()); err != nil {
					t.Fatal(err)
				}
			}
			oneOwner, twoOwner := g.FindCardAnywhere(first.ID()).Owner(), g.FindCardAnywhere(second.ID()).Owner()
			clone := g.Clone()
			if err := ExchangeCardOwnership(FromCard(binding), target).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			swapped := state == "normal" || state == "current-owner"
			wantOne, wantTwo := oneOwner, twoOwner
			if swapped {
				wantOne, wantTwo = twoOwner, oneOwner
			}
			if g.FindCardAnywhere(first.ID()).Owner() != wantOne || g.FindCardAnywhere(second.ID()).Owner() != wantTwo {
				t.Fatal("wrong current owners or invalid visit used")
			}
			if clone.FindCardAnywhere(first.ID()).Owner() != oneOwner || g.CardZone(first.ID()) != core.ZoneExile || g.CardZone(second.ID()) != core.ZoneAnte {
				t.Fatal("zone or clone changed")
			}
			if swapped && (g.FindExiledCard(first.ID()).Visit != visit || g.anteVisit(second.ID()) != clone.anteVisit(second.ID())) {
				t.Fatal("exchange changed visits")
			}
		})
	}
}

func TestCardOwnershipExchangeDelayedFrames(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(b.PlayerID(), "second")
	a.AddToHand(first)
	b.AddToHand(second)
	one, two := BindCard("first"), BindCard("second")
	g.resolution.cardBindings = map[*CardBinding]capturedCard{
		one: {id: first.ID(), visit: a.handVisit(first.ID()), zone: core.ZoneHand},
		two: {id: second.ID(), visit: b.handVisit(second.ID()), zone: core.ZoneHand},
	}
	source := g.PutOnBattlefield(NewArtifact("source", ""), a.PlayerID())
	effect := ExchangeCardOwnership(FromCard(one), FromCard(two))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}
	if err := AtNext(EachEndStep, effect).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.cardBindings = nil
	clone := g.Clone()
	g.PhaseOut(source.ID())
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 0 || g.FindCardAnywhere(first.ID()).Owner() != a.PlayerID() {
			t.Fatal("waiting query executed or inferred source")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindCardAnywhere(first.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(second.ID()).Owner() != a.PlayerID() {
		t.Fatal("delayed frame lost bindings")
	}
	if len(g.resolution.cardBindings) != 0 || clone.FindCardAnywhere(first.ID()).Owner() != a.PlayerID() {
		t.Fatal("frame restoration or clone isolation failed")
	}
}
