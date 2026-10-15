package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardHandContinuation(t *testing.T) {
	for _, state := range []string{"present", "missing", "returned", "hand", "unbound", "token"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 1, 1)
			if state == "token" {
				card = NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil)
			}
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.DestroyPermanent(p)
			binding := BindCard("card")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}
			switch state {
			case "missing", "returned", "hand":
				moved, ok := g.MoveFromGraveyard(b.PlayerID(), card.ID(), core.ZoneHand)
				if !ok {
					t.Fatal("missing fixture")
				}
				if state == "returned" {
					b.AddToGraveyard(moved)
				} else {
					b.AddToHand(moved)
				}
				if state == "hand" {
					g.resolution.cardBindings[binding] = capturedCard{id: card.ID(), visit: b.handVisit(card.ID()), zone: core.ZoneHand}
				}
			case "unbound":
				delete(g.resolution.cardBindings, binding)
			}
			effect := ReturnCardToHandOrElse(FromCard(binding), GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(3)))
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 23
			if state == "present" {
				want = 22
				if len(b.Hand()) != 1 || len(b.Graveyard()) != 0 || len(a.Hand()) != 0 {
					t.Fatal("wrong destination")
				}
			}
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if a.Life() != want+3 {
				t.Fatal("result leaked between instructions")
			}
		})
	}
}

func TestGraveyardHandContinuationInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		ReturnCardToHandOrElse(nil, GainLifeFor(You, Fixed(1)), nil),
		ReturnCardToHandOrElse(FromCard(BindCard("unbound")), GainLifeFor(You, Fixed(1)), nil),
		ReturnCardToHandOrElse(FromCard(BindCard("card")), nil, nil),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestGraveyardHandContinuationWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("dead")
	body := ReturnCardToHandOrElse(FromCard(dead), GainLifeFor(ControllerOf(This), Fixed(2)), GainLifeFor(You, Fixed(3)))
	source := NewArtifact("Watcher", "", WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), AtNext(EachEndStep, body))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.DestroyPermanent(g.PutOnBattlefield(victim, b.PlayerID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.WaitingDelayedGraveyardSources()) != 1 || len(b.Hand()) != 0 || a.Life() != 20 {
			t.Fatal("waiting query lost bindings or executed")
		}
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 25 || len(b.Hand()) != 1 || clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 1 {
		t.Fatal("copy or clone shared result")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 {
		t.Fatal("clone lost frame")
	}
}
