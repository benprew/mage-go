package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnyDrawEventCardVisitsAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDrawsCard(drawer).WithCard(drawn),
		IfCardHasType(FromCard(drawn), core.TypeLand, GainLifeFor(FromPlayer(drawer), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	landA, landB := NewLand("A land"), NewLand("B land")
	a.AddToLibrary(landA)
	b.AddToLibrary(landB)
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("draw triggers = %d", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatalf("clone life = %d, %d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
	if _, ok := b.DiscardCard(landB.ID()); !ok {
		t.Fatal("drawn land absent")
	}
	b.AddToHand(landB)
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("stale draw visit counted: %d, %d", a.Life(), b.Life())
	}
}

func TestOpponentDrawEventCardAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDrawsCard(drawer).WithCard(drawn),
		IfCardHasType(FromCard(drawn), core.TypeLand, GainLifeFor(You, Fixed(1))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	a.AddToLibrary(NewLand("Own"))
	b.AddToLibrary(NewLand("Other"))
	g.PlayerDrawCard(a)
	phase := g.PhaseOut(watcher.ID())
	g.PlayerDrawCard(b)
	g.PhaseIn(phase)
	b.AddToLibrary(NewLand("Later"))
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("opponent draw triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("queued draw trigger life = %d", a.Life())
	}
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(watcher.ID(), b.PlayerID(), func(*Game, uuid.UUID) { replaced++ })
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if replaced != 1 || g.StackSize() != 0 {
		t.Fatalf("replaced draw: replacement = %d, stack = %d", replaced, g.StackSize())
	}
}

func TestAnyDrawEventCardDeclaration(t *testing.T) {
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	if info := InspectDefinition(PlayerDrawsCard(drawer).WithCard(drawn)); info.Role != "event-pattern" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
	body := IfCardHasType(FromCard(drawn), core.TypeLand, GainLifeFor(You, Fixed(1)))
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		body    ResolvingEffect
		want    string
	}{
		{"valid", PlayerDrawsCard(drawer).WithCard(drawn), body, ""},
		{"nil", PlayerDrawsCard(drawer).WithCard(nil), GainLifeFor(You, Fixed(1)), "non-nil"},
		{"unnamed", OpponentDrawsCard(drawer).WithCard(BindCard("")), GainLifeFor(You, Fixed(1)), "name"},
		{"repeated", PlayerDrawsCard(drawer).WithCard(drawn).WithCard(drawn), body, "once"},
		{"missing", PlayerDrawsCard(drawer), body, "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern, tc.body)))
			err := ValidateCardDefinition(definition.Name(), definition)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	other := NewEnchantment("Scope", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer).WithCard(drawn), body)),
		WithAbility(Activate(Costs(), body)))
	if err := ValidateCardDefinition(other.Name(), other); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope validation = %v", err)
	}
}
