package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTurnTrackers_RecordAndReset(t *testing.T) {
	tt := NewTurnTrackers()
	p1 := uuid.New()
	p2 := uuid.New()
	perm1 := uuid.New()

	tt.RecordDamageTaken(p1, 5, false)
	tt.RecordDamageTaken(p1, 2, true)
	tt.RecordAttacked(perm1)
	tt.RecordBlocked(perm1, uuid.New())
	tt.RecordInstantCast(p1)
	tt.RecordSorceryCast(p2)
	tt.RecordCreatureDeath()
	tt.RecordLandPlayed()
	tt.AddExtraLandPlays(p1, 1)
	tt.SetOptionalCostPaid(perm1, true)
	tt.SetUntappedLandsAtTurnStart(p1, 4)
	tt.IncrementCleanupPriorityRounds()

	// Verify queries before reset
	if tt.DamageTaken(p1) != 7 {
		t.Errorf("expected DamageTaken 7, got %d", tt.DamageTaken(p1))
	}
	if tt.ArtifactDamageTaken(p1) != 2 {
		t.Errorf("expected ArtifactDamageTaken 2, got %d", tt.ArtifactDamageTaken(p1))
	}
	if !tt.Attacked(perm1) {
		t.Errorf("expected Attacked true")
	}
	if len(tt.Blocked(perm1)) != 1 {
		t.Errorf("expected 1 blocked entry")
	}
	if tt.InstantsCast(p1) != 1 {
		t.Errorf("expected InstantsCast 1, got %d", tt.InstantsCast(p1))
	}
	if tt.SorceriesCast(p2) != 1 {
		t.Errorf("expected SorceriesCast 1, got %d", tt.SorceriesCast(p2))
	}
	if tt.CreatureDeaths() != 1 {
		t.Errorf("expected CreatureDeaths 1, got %d", tt.CreatureDeaths())
	}
	if tt.LandsPlayed() != 1 {
		t.Errorf("expected LandsPlayed 1, got %d", tt.LandsPlayed())
	}
	if tt.ExtraLandPlays(p1) != 1 {
		t.Errorf("expected ExtraLandPlays 1, got %d", tt.ExtraLandPlays(p1))
	}
	if !tt.OptionalCostPaid(perm1) {
		t.Errorf("expected OptionalCostPaid true")
	}
	if tt.UntappedLandsAtTurnStart(p1) != 4 {
		t.Errorf("expected UntappedLandsAtTurnStart 4, got %d", tt.UntappedLandsAtTurnStart(p1))
	}
	if tt.CleanupPriorityRounds() != 1 {
		t.Errorf("expected CleanupPriorityRounds 1, got %d", tt.CleanupPriorityRounds())
	}

	// Reset for new turn (active player p1)
	tt.ResetForNewTurn(p1)

	// Verify attackedLastTurn contains perm1 for p1
	if !tt.AttackedLastTurn(p1)[perm1] {
		t.Errorf("expected attackedLastTurn[p1][perm1] to be true")
	}

	// Verify turn-scoped fields are reset
	if tt.DamageTaken(p1) != 0 {
		t.Errorf("expected DamageTaken reset to 0, got %d", tt.DamageTaken(p1))
	}
	if tt.Attacked(perm1) {
		t.Errorf("expected Attacked reset to false")
	}
	if len(tt.Blocked(perm1)) != 0 {
		t.Errorf("expected Blocked reset to empty")
	}
	if tt.InstantsCast(p1) != 0 {
		t.Errorf("expected InstantsCast reset to 0")
	}
	if tt.CreatureDeaths() != 0 {
		t.Errorf("expected CreatureDeaths reset to 0")
	}
	if tt.LandsPlayed() != 0 {
		t.Errorf("expected LandsPlayed reset to 0")
	}
}

func TestTrackerSystem_CloneIndependence(t *testing.T) {
	ts := NewTrackerSystem()
	p1 := uuid.New()
	p2 := uuid.New()
	perm1 := uuid.New()

	ts.Turn.RecordDamageTaken(p1, 3, false)
	ts.Turn.RecordAttacked(perm1)
	ts.Duel.RecordCreatureDeath(p2)

	clone := ts.Clone()

	// Mutate original
	ts.Turn.RecordDamageTaken(p1, 10, true)
	ts.Turn.RecordAttacked(uuid.New())
	ts.Duel.RecordCreatureDeath(p2)

	// Verify clone is unchanged
	if clone.Turn.DamageTaken(p1) != 3 {
		t.Errorf("expected clone DamageTaken 3, got %d", clone.Turn.DamageTaken(p1))
	}
	if clone.Turn.ArtifactDamageTaken(p1) != 0 {
		t.Errorf("expected clone ArtifactDamageTaken 0, got %d", clone.Turn.ArtifactDamageTaken(p1))
	}
	if len(clone.Turn.AttackedMap()) != 1 {
		t.Errorf("expected clone AttackedMap len 1, got %d", len(clone.Turn.AttackedMap()))
	}
	obj := clone.Duel.ObjectivesFor(p1, p2)
	if obj.OpponentCreaturesDestroyed != 1 {
		t.Errorf("expected clone OpponentCreaturesDestroyed 1, got %d", obj.OpponentCreaturesDestroyed)
	}
}
