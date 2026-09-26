package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type fixedBlockPlayer struct {
	*BasePlayer
	assignments []BlockAssignment
}

func (p *fixedBlockPlayer) DeclareBlockers(*Game) []BlockAssignment { return p.assignments }

func TestDeclaredBlockerEventUsesFinalLegalPairs(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 4, 4, WithStaticAbility(SourceCantBeBlockedByFewerThan(3)))
	blocker := NewCreature("Blocker", "", 1, 1, WithAbility(
		NewTriggered(EvtDeclaredBlocker, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{}),
	))
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 20 {
		t.Fatalf("removed block triggered: life = %d, want 20", got)
	}
}

func TestNormalBlockDeclarationRejectsRemovedPairEvent(t *testing.T) {
	a := NewBasePlayer("A")
	b := &fixedBlockPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 4, 4, WithStaticAbility(SourceCantBeBlockedByFewerThan(3)))
	blocker := NewCreature("Blocker", "", 1, 1, WithAbility(
		NewTriggered(EvtDeclaredBlocker, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{}),
	))
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	b.assignments = []BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}}
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 20 {
		t.Fatalf("removed block triggered in normal combat: life = %d", got)
	}
}

func TestDeclaredBlockerEventGroupsFinalPairsInClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	blocker := NewCreature("Blocker", "", 1, 5, WithKeyword(CanBlockAny),
		WithAbility(BlocksTrigger(GainLifeFor(You, Fixed(1)), false)),
		WithAbility(NewTriggered(EvtDeclaredBlocker, false, GainLifeFor(You, Fixed(1))).SetConditionData(EventSourceIsSelf{})))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttacker(first.ID(), b.PlayerID())
	g.combat.AddAttacker(second.ID(), b.PlayerID())
	clone := g.Clone()
	clone.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}, {BlockerID: blocker.ID(), AttackerID: second.ID()}})
	if got := clone.GetPlayer(b.PlayerID()).Life(); got != 23 {
		t.Fatalf("two legal pairs gave life %d, want 23", got)
	}
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 20 {
		t.Fatalf("clone changed original life: %d", got)
	}
}
