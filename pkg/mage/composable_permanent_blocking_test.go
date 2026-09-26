package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentBlockingStateIdentityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := NewCreature("Blocker", "", 2, 2)
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(blocker, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	old := g.FindPermanent(blocker.ID()).incarnationID
	ctx := StateContext{SourceID: blocker.ID(), SourceIncarnation: old}
	condition := PermanentIsBlocking(This)
	if condition.checkState(g, ctx) {
		t.Fatal("matched before block")
	}
	g.combat.AddAttackerIncarnation(attacker.ID(), g.FindPermanent(attacker.ID()).incarnationID, a.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), old, attacker.ID())
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("current blocker did not match")
	}
	clone := g.Clone()
	clone.RemoveFromCombat(blocker.ID())
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone shared combat state")
	}
	g.FindPermanent(blocker.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased blocker matched")
	}
	g.FindPermanentIncludingPhased(blocker.ID()).PhasedOut = false
	phase := g.PhaseOut(blocker.ID())
	g.PhaseIn(phase)
	if condition.checkState(g, ctx) {
		t.Fatal("phased blocker stayed in combat")
	}
	g.combat.AddBlockerIncarnation(blocker.ID(), old, attacker.ID())
	g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent blocker matched")
	}
	g.PutOnBattlefield(blocker, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: blocker.ID()}) || condition.checkState(g, StateContext{SourceID: uuid.Nil}) {
		t.Fatal("returned or missing blocker matched")
	}
}

func TestPermanentBlockingAttachedAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aura := NewAura("Watcher", "", WithAbility(Static(While(PermanentIsBlocking(AttachedPermanent), GrantKeywordModification(AttachedPermanent, Shroud)))))
	host := NewCreature("Host", "", 2, 2)
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(host, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	ctx := StateContext{SourceID: aura.ID(), SourceIncarnation: g.FindPermanent(aura.ID()).incarnationID}
	condition := PermanentIsBlocking(AttachedPermanent)
	g.combat.AddAttackerIncarnation(attacker.ID(), g.FindPermanent(attacker.ID()).incarnationID, a.PlayerID())
	g.combat.AddBlockerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, attacker.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("unattached source matched")
	}
	g.Attach(aura.ID(), host.ID())
	if !condition.checkState(g, ctx) {
		t.Fatal("blocking host did not match")
	}
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("static rule did not see blocking host")
	}
	g.RemoveFromCombat(host.ID())
	g.ApplyContinuousEffects()
	if condition.checkState(g, ctx) || g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("removed host still matched")
	}
	g.combat.AddBlockerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, attacker.ID())
	g.FindPermanent(aura.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased attachment matched")
	}
	g.FindPermanentIncludingPhased(aura.ID()).PhasedOut = false
	g.FindPermanent(host.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased host matched")
	}
	g.FindPermanentIncludingPhased(host.ID()).PhasedOut = false
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("returned host inherited attachment or block")
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "attached permanent is blocking" {
		t.Fatalf("inspection = %+v", got)
	}
	target := DeclarePermanentTarget("target", IsCreature)
	bad := NewAura("Bad", "", WithAbility(Static(While(PermanentIsBlocking(target), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("invalid state reference accepted: %v", err)
	}
	spell := NewInstant("Bad spell", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("nonbattlefield source accepted")
	}
}

func TestPermanentBlockingNormalDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 2, 2)
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.FindPermanent(attacker.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	ctx := StateContext{SourceID: blocker.ID(), SourceIncarnation: g.FindPermanent(blocker.ID()).incarnationID}
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
	if !PermanentIsBlocking(This).checkState(g, ctx) {
		t.Fatal("declared blocker did not match")
	}
	entered := NewCreature("Entering blocker", "", 1, 1)
	g.PutOnBattlefieldBlocking(entered, b.PlayerID(), attacker.ID())
	entry := StateContext{SourceID: entered.ID(), SourceIncarnation: g.FindPermanent(entered.ID()).incarnationID}
	if !PermanentIsBlocking(This).checkState(g, entry) {
		t.Fatal("creature entering blocking did not match current state")
	}
}
