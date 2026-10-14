package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileCounterRemovalVisits(t *testing.T) {
	for _, scenario := range []string{"zero", "one", "wide", "depart", "return", "hidden", "wrong-zone", "unwritten", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			card.SetOwner(a.PlayerID())
			g.ExileCard(card, card.ID())
			object := g.FindExiledCard(card.ID())
			object.Counters[core.Scream] = 300
			object.Counters[core.Charge] = 2
			binding := BindCard("source")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: object.Visit, zone: core.ZoneExile}}
			if scenario == "zero" || scenario == "one" {
				object.Counters[core.Scream] = 0
				if scenario == "one" {
					object.Counters[core.Scream] = 1
				}
			}
			clone := g.Clone()
			if scenario == "depart" || scenario == "return" || scenario == "wrong-zone" {
				g.RemoveFromExile(card.ID())
				if scenario == "return" {
					g.ExileCard(card, card.ID())
					g.FindExiledCard(card.ID()).Counters[core.Scream] = 300
				}
				if scenario == "wrong-zone" {
					a.AddToGraveyard(card)
					g.resolution.cardBindings[binding] = capturedCard{id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}
				}
			}
			if scenario == "hidden" {
				object.FaceDown = true
				g.RevealExiledCardTo(card.ID(), a.PlayerID())
			}
			if scenario == "unwritten" {
				g.resolution.cardBindings = nil
			}
			effect := RemoveCounterFromExiledCard(FromCard(binding), core.Scream)
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if current := g.FindExiledCard(card.ID()); current != nil {
				want := 300
				switch scenario {
				case "zero", "one":
					want = 0
				case "wide", "clone":
					want = 299
				}
				if current.Counters[core.Scream] != want {
					t.Fatalf("counters = %d, want %d", current.Counters[core.Scream], want)
				}
				if scenario != "return" && current.Counters[core.Charge] != 2 {
					t.Fatal("removal changed another counter type")
				}
			}
			if scenario == "clone" {
				if clone.FindExiledCard(card.ID()).Counters[core.Scream] != 300 {
					t.Fatal("removal changed the clone")
				}
				if err := effect.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if clone.FindExiledCard(card.ID()).Counters[core.Scream] != 299 || g.FindExiledCard(card.ID()).Counters[core.Scream] != 299 {
					t.Fatal("clone removal lost its saved visit")
				}
			}
		})
	}
}

func TestExileCounterRemovalDefinition(t *testing.T) {
	for _, scenario := range []string{"valid", "nil", "nil-binding", "unwritten", "target", "negative", "outside"} {
		t.Run(scenario, func(t *testing.T) {
			binding := BindCard("source")
			ref := FromCard(binding)
			counter := core.Scream
			switch scenario {
			case "nil":
				ref = nil
			case "nil-binding":
				ref = FromCard(nil)
			case "unwritten":
				ref = FromCard(BindCard("unwritten"))
			case "target":
				ref = DeclareGraveyardCardTarget("target")
			case "negative":
				counter = -1
			case "outside":
				counter = core.NumCounters
			}
			card := NewArtifact("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), RemoveCounterFromExiledCard(ref, counter), WithSourceCardInZone(core.ZoneExile, binding))))
			err := ValidateCardDefinition(card.Name(), card)
			if (err == nil) != (scenario == "valid") {
				t.Fatalf("definition validation = %v", err)
			}
		})
	}
}
