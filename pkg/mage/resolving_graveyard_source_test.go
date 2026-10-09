package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolvingGraveyardSourcesFrames(t *testing.T) {
	for _, kind := range []string{"target", "card", "cards", "event", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Saved visit", "{R}")
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			visit := g.graveyardVisit(card.ID())
			saved := g.CaptureGraveyardDamageSource(visit)
			identity := capturedCard{id: card.ID(), visit: visit, zone: core.ZoneGraveyard}
			object := &StackObject{ID: uuid.New(), IsAbility: true, Controller: a.PlayerID()}
			switch kind {
			case "target":
				object.Targets = []uuid.UUID{card.ID(), card.ID()}
			case "card":
				object.capturedCards = map[*CardBinding]capturedCard{BindCard("card"): identity}
			case "cards":
				object.capturedCardSets = map[*CardsBinding][]capturedCard{BindCards("cards"): {identity, identity}}
			case "event":
				object.Effects = []Effect{Sequence(Sequence(&eventCardCapture{binding: BindCard("event"), value: identity}))}
			case "delayed":
				object.Effects = []Effect{&capturedDelayedBody{body: GainLifeFor(You, Fixed(0)), cardSets: map[*CardsBinding][]capturedCard{BindCards("delayed"): {identity}}}}
			}
			runs := 0
			check := func(game *Game) {
				t.Helper()
				got := game.StackGraveyardSources()
				if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Red) {
					t.Fatalf("resolving %s lost exact visit: %v", kind, got)
				}
				got[0].card.AddType(core.TypeLand)
				if game.StackGraveyardSources()[0].HasType(core.TypeLand) {
					t.Fatal("query changed retained facts")
				}
			}
			object.Effects = append(object.Effects, FuncEffect("Check resolving references", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
				runs++
				check(game)
				clone := game.Clone()
				check(clone)
				game.GetPlayer(a.PlayerID()).RemoveFromGraveyard(card.ID())
				card.SetColorOverride([]core.Color{core.Blue})
				game.GetPlayer(a.PlayerID()).AddToGraveyard(card)
				check(game)
				if !clone.CaptureGraveyardDamageSource(visit).HasColor(core.Red) {
					t.Fatal("clone shared zone facts")
				}
				inner := &StackObject{ID: uuid.New(), IsAbility: true, Controller: a.PlayerID(), Effects: []Effect{FuncEffect("Check outer frame", EffectProperties{}, func(nested *Game, _, _ uuid.UUID, _ []uuid.UUID) error { check(nested); return nil })}}
				game.PushStack(inner)
				game.ResolveTopOfStack()
				check(game)
				return nil
			}))
			g.PushStack(object)
			g.ResolveTopOfStack()
			if runs != 1 || len(g.StackGraveyardSources()) != 0 {
				t.Fatal("frame escaped resolution")
			}
		})
	}
}

func TestResolvingGraveyardSourcesInvalidReferences(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Unreferenced", "{R}")
	a.AddToGraveyard(card)
	identity := capturedCard{id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}
	object := &StackObject{ID: uuid.New(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{nil: identity, BindCard("wrong visit"): {id: card.ID(), visit: uuid.New(), zone: core.ZoneGraveyard}, BindCard("wrong zone"): {id: card.ID(), visit: identity.visit, zone: core.ZoneHand}}}
	object.Effects = []Effect{FuncEffect("Check absent references", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		if len(game.StackGraveyardSources()) != 0 {
			t.Fatal("invalid or unbound visit became available")
		}
		return nil
	})}
	g.PushStack(object)
	g.ResolveTopOfStack()
}
