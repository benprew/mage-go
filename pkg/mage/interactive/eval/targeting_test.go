package eval

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// Destroy-based removal must avoid indestructible creatures, while exile-based
// removal (TargetExile) treats them as ordinary high-value targets.
func TestPermanentValueForTargeting_ExileBypassesIndestructible(t *testing.T) {
	g, _, pb := makeGame()
	perm := makePerm("Avatar", "{4}{R}{R}", 5, 5, pb.PlayerID(), mage.WithKeyword(core.Indestructible))
	g.AddToBattlefield(perm)

	if got := PermanentValueForTargeting(g, perm, TargetRemoval); got != -20 {
		t.Fatalf("TargetRemoval on indestructible should be penalized (-20), got %d", got)
	}
	if got := PermanentValueForTargeting(g, perm, TargetExile); got <= 0 {
		t.Fatalf("TargetExile on indestructible should be a positive target value, got %d", got)
	}
}

// An effect tagged AITargetExile is classified as TargetExile so the AI scores
// its targets with the exile (indestructibility-ignoring) heuristic.
func TestTargetPurposeFromAI_Exile(t *testing.T) {
	if got := TargetPurposeFromAI(mage.AITargetExile); got != TargetExile {
		t.Fatalf("AITargetExile should map to TargetExile, got %v", got)
	}
}
