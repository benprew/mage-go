package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlocksOrBecomesBlockedByGroup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 3, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).OfSubtype("Orc"), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	human := NewCreature("Human", "", 1, 2, WithSubTypes("Human"))
	orc1 := NewCreature("Orc 1", "", 1, 2, WithSubTypes("Orc"))
	orc2 := NewCreature("Orc 2", "", 1, 2, WithSubTypes("Orc"))
	for _, card := range []Card{human, orc1, orc2} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: human.ID(), AttackerID: watcher.ID()}, {BlockerID: orc1.ID(), AttackerID: watcher.ID()}, {BlockerID: orc2.ID(), AttackerID: watcher.ID()}})
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("two matching blockers produced life %d, want 21", got)
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: a.PlayerID(), RelatedIDs: []uuid.UUID{human.ID()}})
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("unmatched group produced life %d", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone life changed to %d", got)
	}
}

func TestBlocksOrBecomesBlockedByGroupValidation(t *testing.T) {
	for _, subtype := range []string{"", "Island"} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
			BlocksOrBecomesBlockedBy(This).OfSubtype(subtype), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "subtype") {
			t.Fatalf("subtype %q accepted: %v", subtype, err)
		}
	}
}

func TestBlocksOrBecomesBlockedByGroupBlockingAndTiming(t *testing.T) {
	a := NewBasePlayer("A")
	b := &fixedBlockPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2, WithSubTypes("Orc"))
	second := NewCreature("Second", "", 2, 2, WithSubTypes("Orc"))
	watcher := NewCreature("Watcher", "", 2, 5, WithKeyword(core.CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).OfSubtype("Orc"), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.combat.AddAttacker(first.ID(), b.PlayerID())
	g.combat.AddAttacker(second.ID(), b.PlayerID())
	b.assignments = []BlockAssignment{{BlockerID: watcher.ID(), AttackerID: first.ID()}, {BlockerID: watcher.ID(), AttackerID: second.ID()}}
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 21 {
		t.Fatalf("two attacked Orcs produced life %d, want 21", got)
	}
	g.FireEvent(core.GameEvent{Type: core.EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: b.PlayerID(), Flag: true, RelatedIDs: []uuid.UUID{first.ID()}})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 22 {
		t.Fatalf("old trigger controller changed after source returned: %d", got)
	}
}

func TestBlocksOrBecomesBlockedByGroupEventTimeSubtype(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 3, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).OfSubtype("Orc"), GainLifeFor(You, Fixed(1)))))
	orc := NewCreature("Orc", "", 1, 1, WithSubTypes("Orc"))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(orc, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: a.PlayerID(), RelatedIDs: []uuid.UUID{orc.ID()}})
	g.PhaseOut(orc.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("later absence changed event-time match: %d", got)
	}
	g.FireEvent(core.GameEvent{Type: core.EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: a.PlayerID(), RelatedIDs: []uuid.UUID{orc.ID()}})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("phased-out creature matched group: %d", got)
	}
}
