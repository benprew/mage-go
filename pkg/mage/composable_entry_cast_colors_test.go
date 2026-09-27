package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEntryCastColorsCounters(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := NewCreature("Converge", "{3}", 2, 2, WithAbility(
		EntersWithCounters(core.P1P1, ColorsSpentToCastEnteringPermanent())))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	if g.StackSize() != 0 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("failed payment changed entry state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*Game{g, clone} {
		p := game.FindPermanent(card.ID())
		if p == nil || p.Counters[core.P1P1] != 2 {
			t.Fatalf("cast entry counters = %v", p)
		}
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.Counters[core.P1P1] != 0 {
		t.Fatalf("uncast entry counters = %d", returned.Counters[core.P1P1])
	}
}

func TestEntryCastColorsDeclaration(t *testing.T) {
	valid := NewCreature("Valid", "", 1, 1, WithAbility(
		EntersWithCounters(core.P1P1, ColorsSpentToCastEnteringPermanent())))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(valid.Abilities()[0]).Children; len(got) == 0 {
		t.Fatal("entry replacement missing from inspection")
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"nil", NewCreature("Nil", "", 1, 1, WithAbility(EntersWithCounters(core.P1P1, nil))), "entry amount"},
		{"counter", NewCreature("Counter", "", 1, 1, WithAbility(EntersWithCounters(core.CounterType(999), ColorsSpentToCastEnteringPermanent()))), "counter type"},
	} {
		if err := ValidateCardDefinition(tc.name, tc.card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestEntryCastColorsCounterReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	doubler := g.PutOnBattlefield(NewCreature("Doubler", "", 1, 1), a.PlayerID())
	g.AddCounterDoubler(doubler.ID(), core.P1P1, ControlledBy(a.PlayerID()))
	card := NewCreature("Converge", "{2}", 2, 2, WithAbility(
		EntersWithCounters(core.P1P1, ColorsSpentToCastEnteringPermanent())))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if got := g.FindPermanent(card.ID()).Counters[core.P1P1]; got != 4 {
		t.Fatalf("replacement counters = %d", got)
	}
}

func TestEntryCastColorsZeroAllowsAdditionalCounters(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := g.PutOnBattlefield(NewCreature("Donor", "", 1, 1), a.PlayerID())
	g.AddETBAdditionalCounters(donor.ID(), core.P1P1, 1,
		And(IsCreature, ControlledBy(a.PlayerID())), false)
	card := NewCreature("Uncast", "", 2, 2, WithAbility(
		EntersWithCounters(core.P1P1, ColorsSpentToCastEnteringPermanent())))
	entry := g.PutOnBattlefield(card, a.PlayerID())
	if got := entry.Counters[core.P1P1]; got != 1 {
		t.Fatalf("additional counters after zero base = %d", got)
	}
}
