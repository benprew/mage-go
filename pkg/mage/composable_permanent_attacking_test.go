package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentAttackingStateAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	old := g.FindPermanent(card.ID()).incarnationID
	ctx := StateContext{SourceID: card.ID(), SourceIncarnation: old}
	condition := PermanentIsAttacking(This)
	if condition.checkState(g, ctx) {
		t.Fatal("matched before combat")
	}
	g.combat.AddAttackerIncarnation(card.ID(), old, b.PlayerID())
	if !condition.checkState(g, ctx) {
		t.Fatal("current attacker did not match")
	}
	clone := g.Clone()
	clone.RemoveFromCombat(card.ID())
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone shared combat state")
	}
	g.FindPermanent(card.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased attacker matched")
	}
	g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
	if !condition.checkState(g, ctx) {
		t.Fatal("phased-in attacker lost identity")
	}
	phase := g.PhaseOut(card.ID())
	g.PhaseIn(phase)
	if condition.checkState(g, ctx) {
		t.Fatal("phasing kept attacker in combat")
	}
	g.combat.AddAttackerIncarnation(card.ID(), old, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent attacker matched")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) || condition.checkState(g, StateContext{SourceID: uuid.Nil}) {
		t.Fatal("returned or missing attacker matched")
	}
}

func TestPermanentAttackingAttachedAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aura := NewAura("Watcher", "", WithAbility(Static(While(PermanentIsAttacking(AttachedPermanent), GrantKeywordModification(AttachedPermanent, Shroud)))))
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(host, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	ctx := StateContext{SourceID: aura.ID(), SourceIncarnation: g.FindPermanent(aura.ID()).incarnationID}
	condition := PermanentIsAttacking(AttachedPermanent)
	g.combat.AddAttackerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, b.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("unattached source matched")
	}
	g.Attach(aura.ID(), host.ID())
	if !condition.checkState(g, ctx) {
		t.Fatal("attacking attachment did not match")
	}
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("static rule did not see attacking host")
	}
	g.RemoveFromCombat(host.ID())
	g.ApplyContinuousEffects()
	if condition.checkState(g, ctx) || g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("removed attacker still matched")
	}
	g.combat.AddAttackerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("returned host inherited attack or attachment identity")
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "attached permanent is attacking" {
		t.Fatalf("inspection = %+v", got)
	}
	target := DeclarePermanentTarget("target", IsCreature)
	bad := NewAura("Bad", "", WithAbility(Static(While(PermanentIsAttacking(target), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("invalid state reference accepted: %v", err)
	}
	spell := NewInstant("Bad spell", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("nonbattlefield source accepted")
	}
}

func TestPermanentAttackingEntryAndNormalDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := NewCreature("Entered attacker", "", 2, 2)
	g.PutOnBattlefieldAttacking(entered, a.PlayerID(), b.PlayerID())
	entry := StateContext{SourceID: entered.ID(), SourceIncarnation: g.FindPermanent(entered.ID()).incarnationID}
	if !PermanentIsAttacking(This).checkState(g, entry) {
		t.Fatal("creature entering attacking did not match current state")
	}
	declared := NewCreature("Declared attacker", "", 2, 2)
	g.PutOnBattlefield(declared, a.PlayerID())
	g.FindPermanent(declared.ID()).RevokeBaseAttr(AttrSummonSick)
	declaration := StateContext{SourceID: declared.ID(), SourceIncarnation: g.FindPermanent(declared.ID()).incarnationID}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{declared.ID()})
	if !PermanentIsAttacking(This).checkState(g, declaration) {
		t.Fatal("declared attacker did not match current state")
	}
}
