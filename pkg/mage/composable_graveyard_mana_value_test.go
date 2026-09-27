package mage

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardManaValueBoundTracksControllerHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cheap := NewCreature("Cheap", "{1}", 1, 1)
	cheap.SetOwner(a.PlayerID())
	expensive := NewCreature("Expensive", "{3}", 3, 3)
	expensive.SetOwner(a.PlayerID())
	other := NewCreature("Other", "{1}", 1, 1)
	other.SetOwner(b.PlayerID())
	a.AddToGraveyard(cheap)
	a.AddToGraveyard(expensive)
	b.AddToGraveyard(other)
	target := DeclareGraveyardCardTarget("card", GraveyardCardFilter(IsCreatureCard), GraveyardCardManaValueAtMost(ControllerLifeGainedThisTurn()))
	if got := target.Possible(a.PlayerID(), nil, g); len(got) != 0 {
		t.Fatalf("zero gain candidates = %v", got)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(1)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := target.Possible(a.PlayerID(), nil, g); !slices.Equal(got, []uuid.UUID{cheap.ID()}) {
		t.Fatalf("one gain candidates = %v", got)
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, GainLifeFor(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := target.Possible(a.PlayerID(), nil, clone); !slices.Contains(got, expensive.ID()) {
		t.Fatalf("clone gain candidates = %v", got)
	}
	if got := target.Possible(a.PlayerID(), nil, g); slices.Contains(got, expensive.ID()) {
		t.Fatalf("clone changed original candidates = %v", got)
	}
	if got := target.Possible(b.PlayerID(), nil, g); len(got) != 0 {
		t.Fatalf("wrong controller candidates = %v", got)
	}
	fromAny := DeclareGraveyardCardTarget("any", InAnyGraveyard(), GraveyardCardManaValueAtMost(FixedTargetNumber(1)))
	if got := fromAny.Possible(a.PlayerID(), nil, g); len(got) != 2 || !slices.Contains(got, other.ID()) {
		t.Fatalf("any-graveyard fixed candidates = %v", got)
	}
}

func TestGraveyardManaValueBoundRechecksTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "{2}", 2, 2)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(ControllerLifeGainedThisTurn()))
	ability := Spell(ReturnCardToHand(target), WithTargets(target))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Recover", spell); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(target.Possible(a.PlayerID(), spell, g), card.ID()) {
		t.Fatal("card was not legal at announcement")
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	if !g.isTargetStillLegal(card.ID(), spell, a.PlayerID(), g.zoneOfTarget(card.ID()), target) {
		t.Fatal("target became illegal without a state change")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if g.isTargetStillLegal(card.ID(), spell, a.PlayerID(), g.zoneOfTarget(card.ID()), target) {
		t.Fatal("target remained legal after the value changed")
	}
	g.ResolveTopOfStack()
	if len(a.Hand()) != 0 {
		t.Fatal("illegal target was returned")
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	obj = &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	// A new graveyard visit must not restore the old target.
	moved, _ := g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand)
	a.AddToHand(moved)
	returned, _ := a.RemoveFromHand(card.ID())
	a.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if len(a.Hand()) != 0 {
		t.Fatal("old graveyard visit was returned")
	}
}

func TestGraveyardManaValueBoundTriggerAnnouncement(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	cheap := NewCreature("Cheap", "{1}", 1, 1)
	cheap.SetOwner(a.PlayerID())
	expensive := NewCreature("Expensive", "{3}", 3, 3)
	expensive.SetOwner(a.PlayerID())
	a.AddToGraveyard(cheap)
	a.AddToGraveyard(expensive)
	target := DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(ControllerLifeGainedThisTurn()))
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This), ReturnCardToHand(target), WithTargets(target))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(1)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.chooseQueue = [][]uuid.UUID{{expensive.ID()}}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackPeek() != nil {
		t.Fatal("trigger accepted an illegal target")
	}
	a.chooseQueue = [][]uuid.UUID{{cheap.ID()}}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if obj := g.StackPeek(); obj == nil || !slices.Equal(obj.Targets, []uuid.UUID{cheap.ID()}) {
		t.Fatalf("trigger target = %v", obj)
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != cheap.ID() {
		t.Fatal("trigger did not use its legal card target")
	}
}

func TestGraveyardManaValueBoundRejectsInvalidDeclaration(t *testing.T) {
	for _, options := range [][]GraveyardTargetOption{
		{GraveyardCardManaValueAtMost(nil)},
		{GraveyardCardManaValueAtMost(FixedTargetNumber(-1))},
		{GraveyardCardManaValueAtMost(FixedTargetNumber(1)), GraveyardCardManaValueAtMost(FixedTargetNumber(2))},
	} {
		target := DeclareGraveyardCardTarget("card", options...)
		card := NewSorcery("Invalid", "", WithAbility(Spell(ReturnCardToHand(target), WithTargets(target))))
		err := ValidateCardDefinition("Invalid", card)
		if err == nil || !strings.Contains(err.Error(), "mana value") {
			t.Fatalf("invalid options %v: %v", options, err)
		}
	}
}
