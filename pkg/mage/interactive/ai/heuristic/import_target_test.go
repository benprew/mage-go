package heuristic

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func TestImportBeneficialPermanentTargetAvoidsEnemy(t *testing.T) {
	g, pa, pb := makeGame()
	g.AddToBattlefield(makePerm("Enemy", "{G}", 2, 2, pb.PlayerID()))
	card := mage.NewInstant("Pump", "{G}", mage.NewTargetedSpell(mage.TargetPermanent(mage.IsCardType(core.TypeCreature)), mage.Boost(mage.Fixed(3), mage.Fixed(3))))
	card.SetOwner(pa.PlayerID())
	s := &Strategy{Personality: ai.MidrangePersonality}
	if targets := s.autoSelectTargets(pa, g, card); len(targets) != 0 {
		t.Fatalf("pump targets an enemy: %v", targets)
	}
}

func TestImportManaSourceColors(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewLand("Compiled Land", mage.WithAbility(mage.NewActivated(mage.Tap(), mage.AddMana(core.Green, 1))))
	g.PutOnBattlefield(card, pa.PlayerID())
	colors, total := boardManaSources(g, pa.PlayerID())
	if total != 1 || colors[core.Green] != 1 {
		t.Fatalf("compiled source colors=%v total=%d", colors, total)
	}
	landColors, anyColor := landManaColors(card)
	if anyColor || len(landColors) != 1 || landColors[0] != core.Green {
		t.Fatalf("compiled land colors=%v any=%v", landColors, anyColor)
	}
}
