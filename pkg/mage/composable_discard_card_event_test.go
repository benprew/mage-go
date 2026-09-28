package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDiscardEventCardVisitAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := BindCard("discarded")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(BindPlayer("discarder")).WithCard(card),
		IfCardHasType(FromCard(card), core.TypeLand, GainLifeFor(You, Fixed(2))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land")
	b.AddToHand(land)
	g.PlayerDiscard(b, land.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatal("clone lost copied discard card")
	}
	b.RemoveFromGraveyard(land.ID())
	b.AddToHand(land)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("discard capture followed a later zone visit")
	}
}

func TestDiscardEventCardReplacementBatchAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := BindCard("discarded")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDiscardsCard(BindPlayer("player")).WithCard(card),
		IfCardHasType(FromCard(card), core.TypeLand, GainLifeFor(You, Fixed(1))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	b.AddToHand(NewLand("First"))
	b.AddToHand(NewLand("Second"))
	g.discardHandsTogether([]uuid.UUID{b.PlayerID()}, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("batch triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("batch life = %d", a.Life())
	}
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	b.AddToHand(NewLand("Replaced"))
	g.PlayerDiscardByEffect(b, b.Hand()[0].ID(), watcher.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("library replacement exposed a graveyard card")
	}
	phase := g.PhaseOut(watcher.ID())
	b.AddToHand(NewLand("Phased"))
	g.PlayerDiscard(b, b.Hand()[0].ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed discard")
	}
	g.PhaseIn(phase)
}

func TestDiscardEventCardDeclaration(t *testing.T) {
	card := BindCard("card")
	if info := InspectDefinition(OpponentDiscardsCard(BindPlayer("player")).WithCard(card)); info.Role != "event-pattern" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		body    ResolvingEffect
		want    string
	}{
		{"valid", PlayerDiscardsCard(BindPlayer("player")).WithCard(card), IfCardHasType(FromCard(card), core.TypeLand, GainLifeFor(You, Fixed(1))), ""},
		{"nil", PlayerDiscardsCard(BindPlayer("player")).WithCard(nil), GainLifeFor(You, Fixed(1)), "non-nil"},
		{"unnamed", PlayerDiscardsCard(BindPlayer("player")).WithCard(BindCard("")), GainLifeFor(You, Fixed(1)), "name"},
		{"duplicate", PlayerDiscardsCard(BindPlayer("player")).WithCard(card).WithCard(card), GainLifeFor(You, Fixed(1)), "once"},
		{"missing", PlayerDiscardsCard(BindPlayer("player")), IfCardHasType(FromCard(card), core.TypeLand, GainLifeFor(You, Fixed(1))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern, tc.body)))
			err := ValidateCardDefinition(definition.Name(), definition)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	outOfScope := NewEnchantment("Scope", "",
		WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("player")).WithCard(card), GainLifeFor(You, Fixed(1)))),
		WithAbility(Activate(Costs(), IfCardHasType(FromCard(card), core.TypeLand, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(outOfScope.Name(), outOfScope); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("out-of-scope validation = %v", err)
	}
}
