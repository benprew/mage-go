package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatOpponentsSelectCurrentIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 4)
	first := NewCreature("First", "", 1, 2)
	second := NewCreature("Second", "", 1, 2)
	for _, card := range []Card{source, first, second} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.combat.AddAttackerIncarnation(source.ID(), g.FindPermanent(source.ID()).incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(first.ID(), g.FindPermanent(first.ID()).incarnationID, source.ID())
	g.combat.AddBlockerIncarnation(second.ID(), g.FindPermanent(second.ID()).incarnationID, source.ID())
	set := BlockingOrBlockedBy(This)
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceIncarnation: g.FindPermanent(source.ID()).incarnationID, sourceCaptured: true}
	if got := len(set.resolvePermanents(ctx)); got != 2 {
		t.Fatalf("blockers = %d, want 2", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	g.PutOnBattlefield(second, b.PlayerID())
	if got := len(set.resolvePermanents(ctx)); got != 1 {
		t.Fatalf("blockers after return = %d, want 1", got)
	}
	g.PhaseOut(first.ID())
	if got := len(set.resolvePermanents(ctx)); got != 0 {
		t.Fatalf("blockers after phasing = %d, want 0", got)
	}
	g.PhaseIn([]uuid.UUID{first.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if got := len(set.resolvePermanents(ctx)); got != 0 {
		t.Fatalf("old source found %d opponents", got)
	}
}

func TestCombatOpponentsBlockingAndTriggerResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 2, 4, WithKeyword(core.CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This), GainLifeFor(You, CountPermanents(BlockingOrBlockedBy(This))))))
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(source, b.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.combat.AddAttackerIncarnation(first.ID(), g.FindPermanent(first.ID()).incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(second.ID(), g.FindPermanent(second.ID()).incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(source.ID(), g.FindPermanent(source.ID()).incarnationID, first.ID())
	g.combat.AddBlockerIncarnation(source.ID(), g.FindPermanent(source.ID()).incarnationID, second.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtBlockerGroup, SourceID: source.ID(), RelatedIDs: []uuid.UUID{first.ID(), second.ID()}, Flag: true})
	g.PutTriggersOnStack()
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 22 {
			t.Fatalf("life after one grouped trigger = %d, want 22", got)
		}
	}
}

func TestCombatOpponentsDeclaration(t *testing.T) {
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This), GainLifeFor(You, CountPermanents(BlockingOrBlockedBy(This))))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This), GainLifeFor(You, CountPermanents(BlockingOrBlockedBy(nil))))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("bad declaration: %v", err)
	}
}
