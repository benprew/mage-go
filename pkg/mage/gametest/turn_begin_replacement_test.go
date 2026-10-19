package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTurnBeginReplacementHarness(t *testing.T) {
	tg := NewTestGame(t)
	a := tg.GetPlayer(PlayerA)
	source := tg.PutOnBattlefield(mage.NewArtifact("Turn skip fixture", "", mage.WithAbility(mage.Static(mage.MaySkipYourTurnToUntap(mage.This, mage.PermanentMatchesFilter(mage.This, mage.IsTapped))))), a.PlayerID())
	tg.TapPermanent(source)
	a.QueueMayAbilityChoices(true, false)
	tg.SkipNextTurnFor(tg.GetPlayer(PlayerB).PlayerID())
	tg.StopAt(3, core.Upkeep)
	tg.Execute()
	if tg.CurrentTurn() != 3 || tg.ActivePlayerObj().PlayerID() != a.PlayerID() || tg.FindPermanent(source.ID()).Tapped || len(a.Hand()) != 0 || len(tg.GetPlayer(PlayerB).Hand()) != 0 {
		t.Fatal("harness ran a skipped turn or lost continuation")
	}
}
