package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomCounterDistributionIsChosenAtCastTimeAndFrozen(t *testing.T) {
	g, a, b := newRandomTargetGame()
	spell := NewInstant("Random Counters", "{X}", NewSpell(
		RandomCounterDistribution(P1P1, XValue()),
		WithTarget(TargetRandomCount(TargetOneToXCreatures())),
	))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Colorless, 4)
	first := addRandomTargetCreature(g, b, "First")
	addRandomTargetCreature(g, b, "Second")
	third := addRandomTargetCreature(g, b, "Third")
	g.SetRandomResults([]int{1, 2, 0, 1, 1})

	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 4); err != nil {
		t.Fatalf("cast random-counter spell: %v", err)
	}
	obj := g.Stack().Peek()
	want := map[uuid.UUID]int{third.ID(): 1, first.ID(): 3}
	if !maps.Equal(obj.CounterDistribution, want) {
		t.Fatalf("counter distribution = %v, want %v", obj.CounterDistribution, want)
	}
	clonedObj := cloneStackObject(obj)
	if !maps.Equal(clonedObj.CounterDistribution, want) {
		t.Fatalf("cloned counter distribution = %v, want %v", clonedObj.CounterDistribution, want)
	}
	clonedObj.CounterDistribution[first.ID()] = 98
	if obj.CounterDistribution[first.ID()] != 3 {
		t.Fatal("clone shares counter distribution map with original")
	}

	copyObj := g.CopySpellOnStack(spell.ID(), a.PlayerID(), false)
	if copyObj == nil || !maps.Equal(copyObj.CounterDistribution, want) {
		t.Fatalf("copied counter distribution = %v, want %v", copyObj.CounterDistribution, want)
	}
	copyObj.CounterDistribution[first.ID()] = 99
	if obj.CounterDistribution[first.ID()] != 3 {
		t.Fatal("copy shares counter distribution map with original")
	}
	g.stack.Pop()

	g.DestroyPermanent(first)
	g.ResolveTopOfStack()
	if got := g.FindPermanent(third.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("surviving target counters = %d, want 1", got)
	}
}

func TestRandomCounterDistributionHandlesZeroTotalAndTargets(t *testing.T) {
	g, a, _ := newRandomTargetGame()
	spell := NewInstant("Zero Random Counters", "{X}", NewSpell(
		RandomCounterDistribution(P1P1, XValue()),
		WithTarget(TargetRandomCount(TargetOneToXCreatures())),
	))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)

	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 0); err != nil {
		t.Fatalf("cast zero-counter spell: %v", err)
	}
	if got := g.Stack().Peek().CounterDistribution; len(got) != 0 {
		t.Fatalf("zero distribution = %v, want empty", got)
	}
	g.ResolveTopOfStack()
}
