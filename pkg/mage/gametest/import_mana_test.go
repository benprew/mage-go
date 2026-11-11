package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestImportManaHarnessUsesAbilityType(t *testing.T) {
	tg := NewTestGame(t)
	owner := tg.playerA.PlayerID()
	p := tg.PutOnBattlefield(mage.NewArtifact("Source", "{1}",
		mage.WithAbility(mage.NewActivated(mage.Tap(), mage.AddMana(core.Green, 1), mage.WithOncePerTurn())),
		mage.WithAbility(mage.NewActivated(mage.LifePayCost(1), mage.GainLife(2))),
	), owner)
	if supply := tg.untappedManaSupply(owner); supply[core.Green] != 1 {
		t.Fatalf("compiled mana supply=%v", supply)
	}
	_, index, err := tg.findActivatableAbilityByName(owner, "Source", nil)
	if err != nil || index != 0 {
		t.Fatalf("expected the first full activated action, got index=%d err=%v", index, err)
	}
	if err := tg.TapForMana(owner, p.ID()); err != nil {
		t.Fatal(err)
	}
	p.Tapped = false
	_, index, err = tg.findActivatableAbilityByName(owner, "Source", nil)
	if err != nil || index != 1 {
		t.Fatalf("expected utility action after mana activation limit, index=%d err=%v", index, err)
	}
	if supply := tg.untappedManaSupply(owner); supply[core.Green] != 0 {
		t.Fatalf("used mana ability remains available: %v", supply)
	}
}
