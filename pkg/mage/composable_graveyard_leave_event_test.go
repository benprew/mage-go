package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardLeaveBatchEventCountAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("left")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCardsLeaveYourGraveyard().WithCount(count), GainLifeFor(You, FromNumber(count)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first, second, other := NewLand("First"), NewLand("Second"), NewLand("Other")
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	b.AddToGraveyard(other)
	g.MoveCardsFromGraveyard(a.PlayerID(), []uuid.UUID{first.ID(), second.ID(), uuid.New()}, core.ZoneExile)
	g.MoveFromGraveyard(b.PlayerID(), other.ID(), core.ZoneExile)
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("batch triggers = %d, want 1", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatalf("clone changed original or lost count: %d, %d", a.Life(), clone.GetPlayer(a.PlayerID()).Life())
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("resolved batch count = %d, want 24", a.Life())
	}
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	g.MoveFromGraveyard(a.PlayerID(), first.ID(), core.ZoneHand)
	g.MoveFromGraveyard(a.PlayerID(), second.ID(), core.ZoneHand)
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("separate moves = %d triggers, want 2", got)
	}
	g.ResolveStack()
	if a.Life() != 26 {
		t.Fatalf("separate count = %d, want 26", a.Life())
	}
}

func TestGraveyardLeaveEventSourceLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCardsLeaveYourGraveyard(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewLand("Card")
	a.AddToGraveyard(card)
	phase := g.PhaseOut(watcher.ID())
	g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	a.AddToGraveyard(card)
	g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("active source did not trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("queued controller changed: %d, %d", a.Life(), b.Life())
	}
	a.AddToGraveyard(card)
	g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("returned source followed former controller")
	}
}

func TestGraveyardLeaveEventDeclaration(t *testing.T) {
	count := BindNumber("left")
	pattern := OneOrMoreCardsLeaveYourGraveyard().WithCount(count)
	if info := InspectDefinition(pattern); info.Role != "event-pattern" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewEnchantment("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(count))))), ""},
		{"nil", NewEnchantment("Nil", "", WithAbility(WhenEvent(OneOrMoreCardsLeaveYourGraveyard().WithCount(nil), GainLifeFor(You, Fixed(1))))), "count binding"},
		{"duplicate", NewEnchantment("Duplicate", "", WithAbility(WhenEvent(pattern.WithCount(count), GainLifeFor(You, Fixed(1))))), "count binding"},
		{"unnamed", NewEnchantment("Unnamed", "", WithAbility(WhenEvent(OneOrMoreCardsLeaveYourGraveyard().WithCount(BindNumber("")), GainLifeFor(You, Fixed(1))))), "name"},
		{"missing", NewEnchantment("Missing", "", WithAbility(WhenEvent(OneOrMoreCardsLeaveYourGraveyard(), GainLifeFor(You, FromNumber(count))))), "producer"},
		{"wrong scope", NewEnchantment("Scope", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))), WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(count))))), "producer"},
		{"wrong zone", NewInstant("Instant", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))), "battlefield"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}
