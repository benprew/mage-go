package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileCounterConditionVisits(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	g.ExileCard(card, card.ID())
	object := g.FindExiledCard(card.ID())
	object.Counters[core.Scream] = 300
	object.Counters[core.Charge] = 2
	binding := BindCard("source")
	state := StateContext{ControllerID: a.PlayerID(), cards: map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: object.Visit, zone: core.ZoneExile}}}
	has := ExiledCardHasCounter(FromCard(binding), core.Scream)
	none := ExiledCardHasNoCounters(FromCard(binding), core.Scream)
	assert := func(game *Game, wantHas, wantNone bool) {
		t.Helper()
		if has.checkState(game, state) != wantHas || none.checkState(game, state) != wantNone {
			t.Fatalf("counter conditions = %v/%v, want %v/%v", has.checkState(game, state), none.checkState(game, state), wantHas, wantNone)
		}
	}
	assert(g, true, false)
	clone := g.Clone()
	g.FindExiledCard(card.ID()).Counters[core.Scream] = 0
	assert(g, false, true)
	assert(clone, true, false)
	if clone.FindExiledCard(card.ID()).Counters[core.Scream] != 300 || clone.FindExiledCard(card.ID()).Counters[core.Charge] != 2 {
		t.Fatal("clone lost counters")
	}
	g.RemoveFromExile(card.ID())
	assert(g, false, false)
	g.ExileCard(card, card.ID())
	if g.FindExiledCard(card.ID()).Counters[core.Charge] != 0 {
		t.Fatal("return retained counters")
	}
	g.FindExiledCard(card.ID()).Counters[core.Scream] = 1
	assert(g, false, false)
	state.cards[binding] = capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}
	assert(g, true, false)
	g.ExileCard(card, card.ID())
	assert(g, false, false)
	if g.FindExiledCard(card.ID()).Counters[core.Scream] != 0 {
		t.Fatal("re-exile retained counters")
	}
}

func TestExileCounterConditionTiming(t *testing.T) {
	for _, scenario := range []string{"absent", "present", "clear-before-resolution", "depart", "return", "clone", "copy", "hidden", "two-instances"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			binding := BindCard("source")
			trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithSourceCardInZone(core.ZoneExile, binding), WithInterveningIf(ExiledCardHasCounter(FromCard(binding), core.Scream)))
			card := NewArtifact("Watcher", "", WithAbility(trigger))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if scenario == "hidden" {
				g.ExileCardFaceDown(card, card.ID(), a.PlayerID())
			} else {
				g.ExileCard(card, card.ID())
			}
			if scenario != "absent" {
				g.FindExiledCard(card.ID()).Counters[core.Scream] = 1
			}
			if scenario == "two-instances" {
				second := card.Copy()
				second.SetOwner(a.PlayerID())
				g.ExileCard(second, second.ID())
			}
			saved := g.resolution.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep armed a trigger")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			if !reflect.DeepEqual(saved, g.resolution) {
				t.Fatal("condition query changed the resolution frame")
			}
			clone := g.Clone()
			if scenario == "clear-before-resolution" {
				g.FindExiledCard(card.ID()).Counters[core.Scream] = 0
			}
			if scenario == "depart" || scenario == "return" || scenario == "clone" {
				g.RemoveFromExile(card.ID())
				if scenario == "return" {
					g.ExileCard(card, card.ID())
					g.FindExiledCard(card.ID()).Counters[core.Scream] = 1
				}
			}
			g.PutTriggersOnStack()
			wantStack := 1
			if scenario == "absent" || scenario == "hidden" {
				wantStack = 0
			}
			if g.StackSize() != wantStack {
				t.Fatalf("stack = %d, want %d", g.StackSize(), wantStack)
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy missing")
			}
			g.ResolveStack()
			want := 20
			if scenario == "present" || scenario == "copy" || scenario == "two-instances" {
				want = 21
			}
			wantB := 20
			if scenario == "copy" {
				wantB = 21
			}
			if a.Life() != want || b.Life() != wantB {
				t.Fatalf("life = %d/%d, want %d/%d", a.Life(), b.Life(), want, wantB)
			}
			if scenario == "clone" {
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 21 {
					t.Fatal("clone lost pending visit or counters")
				}
			}
		})
	}
}

func TestExileCounterConditionInvalid(t *testing.T) {
	for _, build := range []func(*CardBinding) StateCondition{
		func(*CardBinding) StateCondition { return ExiledCardHasCounter(FromCard(nil), core.Scream) },
		func(*CardBinding) StateCondition {
			return ExiledCardHasNoCounters(FromCard(BindCard("unwritten")), core.Scream)
		},
		func(*CardBinding) StateCondition {
			return ExiledCardHasCounter(DeclareGraveyardCardTarget("target"), core.Scream)
		},
		func(binding *CardBinding) StateCondition {
			return ExiledCardHasCounter(FromCard(binding), core.CounterType(-1))
		},
		func(binding *CardBinding) StateCondition {
			return ExiledCardHasNoCounters(FromCard(binding), core.NumCounters)
		},
	} {
		binding := BindCard("source")
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithSourceCardInZone(core.ZoneExile, binding), WithInterveningIf(build(binding)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid counter condition accepted")
		}
	}
	for _, build := range []func(CardRef, core.CounterType) StateCondition{ExiledCardHasCounter, ExiledCardHasNoCounters} {
		binding := BindCard("source")
		card := NewArtifact("Valid", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), IfState(build(FromCard(binding), core.Scream), GainLifeFor(You, Fixed(1))), WithSourceCardInZone(core.ZoneExile, binding))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExileCounterConditionMissingFacts(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	binding := BindCard("source")
	for _, build := range []func(CardRef, core.CounterType) StateCondition{ExiledCardHasCounter, ExiledCardHasNoCounters} {
		condition := build(FromCard(binding), core.Scream)
		if condition.checkState(g, StateContext{}) {
			t.Fatal("unwritten binding supplied counter information")
		}
		a.AddToGraveyard(card)
		state := StateContext{ControllerID: a.PlayerID(), cards: map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}}
		if condition.checkState(g, state) {
			t.Fatal("graveyard supplied exile counters")
		}
		a.RemoveFromGraveyard(card.ID())
		g.ExileCardFaceDown(card, card.ID(), a.PlayerID())
		state.cards[binding] = capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}
		if condition.checkState(g, state) {
			t.Fatal("face-down source supplied a condition")
		}
		g.RemoveFromExile(card.ID())
	}
}
