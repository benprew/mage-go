package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestMixedDamageBatchReportsFinalOutcomes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink)), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(source)
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.AddPreventionShield(a.PlayerID(), 2)
	outcomes := g.dealSimultaneousDamageOutcomes(
		PlayerDamageAssignment(a, 2, captured),
		PlayerDamageAssignment(b, 3, captured),
		PermanentDamageAssignment(victim, 1, captured),
	)
	if len(outcomes) != 2 {
		t.Fatalf("outcomes = %d, want 2 positive damage actions", len(outcomes))
	}
	if outcomes[0].amount != 3 || outcomes[0].permanent.id != victim.ID() || outcomes[0].permanent.incarnation != victim.incarnationID || outcomes[0].playerID != uuid.Nil || outcomes[0].source.Controller() != a.PlayerID() {
		t.Fatalf("redirected outcome = %+v", outcomes[0])
	}
	if outcomes[1].amount != 1 || outcomes[1].permanent.id != victim.ID() || outcomes[1].source.ID() != source.ID() {
		t.Fatalf("permanent outcome = %+v", outcomes[1])
	}
	if a.Life() != 24 || b.Life() != 20 || victim.Damage != 4 {
		t.Fatalf("life = %d, %d; damage = %d", a.Life(), b.Life(), victim.Damage)
	}
}

func TestMixedDamageBatchOutcomesExcludeAbsentRecipients(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Victim", "", 2, 9)
	old := g.PutOnBattlefield(card, b.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	assignment := PermanentDamageAssignment(old, 2, captured)
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	outcomes := clone.dealSimultaneousDamageOutcomes(assignment, PlayerDamageAssignment(clone.GetPlayer(b.PlayerID()), 1, captured))
	if len(outcomes) != 1 || outcomes[0].playerID != b.PlayerID() || outcomes[0].amount != 1 {
		t.Fatalf("outcomes = %+v", outcomes)
	}
	if returned.Damage != 0 || b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatal("returned recipient or original game changed")
	}
	g.PhaseOut(returned.ID())
	if got := g.dealSimultaneousDamageOutcomes(PermanentDamageAssignment(returned, 2, captured)); len(got) != 0 {
		t.Fatalf("phased recipient outcomes = %+v", got)
	}
}
