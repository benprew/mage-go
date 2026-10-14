package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceZoneConditionTiming(t *testing.T) {
	for _, scenario := range []string{"false-event", "present", "false-resolution", "depart", "return", "copy", "same-controller-copy", "pending-clone"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			binding := BindCard("source")
			condition := GraveyardCardHasAtLeastAbove(FromCard(binding), You, 2, NewCardFilter("creature", func(c Card) bool { return c.HasType(core.TypeCreature) }))
			trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithSourceCardInZone(core.ZoneGraveyard, binding), WithInterveningIf(condition))
			card := NewArtifact("Watcher", "", WithAbility(trigger))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			below := NewCreature("Below", "", 1, 1)
			below.SetOwner(a.PlayerID())
			a.AddToGraveyard(below)
			a.AddToGraveyard(card)
			above := make([]Card, 0, 2)
			for i := range 2 {
				c := NewCreature("Above", "", 1, 1)
				c.SetOwner(a.PlayerID())
				above = append(above, c)
				if scenario != "false-event" || i == 0 {
					a.AddToGraveyard(c)
				}
			}
			sentinel := BindCard("unrelated")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{sentinel: {id: below.ID(), visit: g.graveyardVisit(below.ID()), zone: core.ZoneGraveyard}}
			saved := g.resolution.Clone()
			declaration := trigger.(*GenericTriggered)
			source, controller := declaration.source, declaration.controller
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			if !reflect.DeepEqual(g.resolution, saved) || declaration.source != source || declaration.controller != controller {
				t.Fatal("query changed resolution frame")
			}
			clone := g.Clone()
			if scenario == "false-resolution" {
				a.RemoveFromGraveyard(above[0].ID())
			}
			if scenario == "depart" || scenario == "return" {
				a.RemoveFromGraveyard(card.ID())
				if scenario == "return" {
					a.AddToGraveyard(card)
					for _, c := range above {
						a.RemoveFromGraveyard(c.ID())
						a.AddToGraveyard(c)
					}
				}
			}
			g.PutTriggersOnStack()
			wantStack := 1
			if scenario == "false-event" {
				wantStack = 0
			}
			if g.StackSize() != wantStack {
				t.Fatalf("stack = %d", g.StackSize())
			}
			if scenario == "copy" || scenario == "same-controller-copy" {
				copyController := b.PlayerID()
				if scenario == "same-controller-copy" {
					copyController = a.PlayerID()
				}
				if g.CopyStackObjectDirect(g.StackPeek(), copyController, false) == nil {
					t.Fatal("copy missing")
				}
			}
			g.ResolveStack()
			want := 20
			if scenario == "present" || scenario == "copy" || scenario == "pending-clone" {
				want = 21
			}
			if scenario == "same-controller-copy" {
				want = 22
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("life = %d/%d", a.Life(), b.Life())
			}
			if scenario == "pending-clone" {
				a.RemoveFromGraveyard(card.ID())
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 21 {
					t.Fatal("clone lost visit")
				}
			}
		})
	}
}

func TestSourceZoneConditionDeclarations(t *testing.T) {
	target := DeclareGraveyardCardTarget("target")
	unsupported := NewArtifact("Unsupported", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithTargets(target), WithInterveningIf(CardMatchesFilter(target, CardFilter{})))))
	if err := ValidateCardDefinition(unsupported.Name(), unsupported); err == nil {
		t.Fatal("target condition without a detection frame accepted")
	}

	for _, condition := range []StateCondition{
		CardMatchesFilter(DeclareGraveyardCardTarget("target"), CardFilter{}),
		GraveyardCardHasAtLeastAbove(DeclareGraveyardCardTarget("target"), You, 0, CardFilter{}),
		GraveyardCardHasAtLeastAbove(FromCard(BindCard("unwritten")), You, 0, CardFilter{}),
		GraveyardCardHasAtLeastAbove(FromCard(BindCard("source")), You, -1, CardFilter{}),
		CardMatchesFilter(FromCard(nil), CardFilter{}),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithSourceCardInZone(core.ZoneGraveyard, BindCard("source")), WithInterveningIf(condition))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid condition accepted")
		}
	}
	for _, factory := range []func(*CardBinding) StateCondition{
		func(b *CardBinding) StateCondition {
			return GraveyardCardHasAtLeastAbove(FromCard(b), You, 0, CardFilter{})
		},
		func(b *CardBinding) StateCondition { return CardMatchesFilter(FromCard(b), CardFilter{}) },
	} {
		binding := BindCard("source")
		card := NewArtifact("Valid", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithSourceCardInZone(core.ZoneGraveyard, binding), WithInterveningIf(factory(binding)))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceZoneConditionOrderedQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	binding := BindCard("source")
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	creature := NewCreature("Above", "", 1, 1)
	creature.SetOwner(a.PlayerID())
	a.AddToGraveyard(creature)
	other := NewArtifact("Other", "")
	other.SetOwner(a.PlayerID())
	a.AddToGraveyard(other)
	state := StateContext{ControllerID: a.PlayerID(), cards: map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}}
	condition := GraveyardCardHasAtLeastAbove(FromCard(binding), You, 2, NewCardFilter("creature", func(c Card) bool { return c.HasType(core.TypeCreature) }))
	if condition.checkState(g, state) {
		t.Fatal("nonmatching card counted")
	}
	zero := GraveyardCardHasAtLeastAbove(FromCard(binding), You, 0, CardFilter{})
	if !zero.checkState(g, state) {
		t.Fatal("zero threshold failed for present source")
	}
	a.RemoveFromGraveyard(card.ID())
	a.AddToGraveyard(card)
	if zero.checkState(g, state) {
		t.Fatal("zero threshold accepted a returned source")
	}
}

func TestSourceZoneConditionExile(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		binding := BindCard("exile")
		condition := CardMatchesFilter(FromCard(binding), NewCardFilter("artifact", func(c Card) bool { return c.HasType(core.TypeArtifact) }))
		card := NewArtifact("Exile watcher", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithInterveningIf(condition), WithSourceCardInZone(core.ZoneExile, binding))))
		card.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		g.ExileCard(card, card.ID())
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("exile condition had no frame")
		}
		if returned {
			g.RemoveFromExile(card.ID())
			g.ExileCard(card, card.ID())
		}
		g.ResolveStack()
		want := 21
		if returned {
			want = 20
		}
		if a.Life() != want {
			t.Fatalf("exile life = %d", a.Life())
		}
	}
}

func TestSourceZoneConditionHiddenExile(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	binding := BindCard("source")
	card := NewArtifact("Hidden watcher", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1)), WithSourceCardInZone(core.ZoneExile, binding), WithInterveningIf(CardMatchesFilter(FromCard(binding), CardFilter{})))))
	card.SetOwner(a.PlayerID())
	g.ExileCardFaceDown(card, card.ID(), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("private source characteristics armed a trigger")
	}
}
