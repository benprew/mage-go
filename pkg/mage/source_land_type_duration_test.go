package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceLandTypeDurationLayerContracts(t *testing.T) {
	g, a, source, land, _ := basicLandTypeTestGame(t)
	target := DeclarePermanentTarget("land", IsLand)
	g.resolution.targetSpecs = []Target{target.targetSpec()}
	if err := ApplyEffect(g, Until(EndOfTurn, GrantKeywordModification(target, Flying)), source.ID(), a.PlayerID(), []uuid.UUID{land.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	if err := ApplyEffect(g, UntilSourceLeaves(This, SetSubtypeFamily(target, SubtypeLand, "Forest")), source.ID(), a.PlayerID(), []uuid.UUID{land.ID()}); err != nil {
		t.Fatal(err)
	}
	if !land.HasSubType("Forest") || countManaAbilityColor(land, Green) != 1 || countManaAbilityColor(land, Red) != 0 || !land.HasKeyword(Flying) {
		t.Fatal("layer four lost subtype, mana or granted ability")
	}
	clone := g.Clone()
	ids := g.PhaseOut(land.ID())
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PhaseIn(ids)
	g.ApplyContinuousEffects()
	restored := g.FindPermanent(land.ID())
	if !restored.HasSubType("Mountain") || countManaAbilityColor(restored, Red) != 1 || countManaAbilityColor(restored, Green) != 0 || !restored.HasKeyword(Flying) {
		t.Fatalf("expired type effect: types %v, red %d, green %d, flying %v", restored.SubTypes(), countManaAbilityColor(restored, Red), countManaAbilityColor(restored, Green), restored.HasKeyword(Flying))
	}
	if !clone.FindPermanent(land.ID()).HasSubType("Forest") {
		t.Fatal("clone shared expiration")
	}
}
