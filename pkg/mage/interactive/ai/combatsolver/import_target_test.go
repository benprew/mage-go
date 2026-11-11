package combatsolver

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestImportPumpSpellTarget(t *testing.T) {
	card := mage.NewInstant("Pump", "{G}", mage.NewTargetedSpell(mage.TargetPermanent(mage.ObjectAnd(mage.IsCardType(core.TypeCreature), mage.OtherThan(mage.SourceObject()))), mage.Boost(mage.Fixed(3), mage.Fixed(3))))
	p, tough, creature := pumpSpellBoost(card)
	if p != 3 || tough != 3 || !creature {
		t.Fatalf("pump was not recognized: %d/%d creature=%v", p, tough, creature)
	}
}

func TestImportPumpAbilityTarget(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.PutOnBattlefield(mage.NewArtifact("Pump Source", "{1}", mage.WithAbility(mage.NewActivated(
		mage.ManaCostOf("{G}"), mage.WithTarget(mage.TargetPermanent(mage.IsCardType(core.TypeCreature))), mage.Boost(mage.Fixed(2), mage.Fixed(2)),
	))), a.PlayerID())
	if pumps := gatherAbilityPumps(g, a.PlayerID()); len(pumps) != 1 {
		t.Fatalf("compiled pump ability count=%d", len(pumps))
	}
}
