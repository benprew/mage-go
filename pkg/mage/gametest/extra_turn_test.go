package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExtraTurnHarnessNormalResume(t *testing.T) {
	tg := NewTestGame(t)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	a, b := tg.GetPlayer(PlayerA).PlayerID(), tg.GetPlayer(PlayerB).PlayerID()
	tg.GrantExtraTurn(a)
	tg.GrantExtraTurn(b)
	tg.StopAt(2, core.PrecombatMain)
	tg.Execute()
	if tg.ActivePlayerObj().PlayerID() != b {
		t.Fatal("harness did not select the newest extra turn")
	}
	cloned := tg.Clone()
	tg.StopAt(3, core.PrecombatMain)
	tg.Execute()
	if tg.ActivePlayerObj().PlayerID() != a {
		t.Fatal("harness lost the older extra turn")
	}
	tg.StopAt(4, core.PrecombatMain)
	tg.Execute()
	if tg.ActivePlayerObj().PlayerID() != b || tg.CurrentTurn() != 4 {
		t.Fatal("harness lost the next normal player or turn synchronization")
	}
	cloned.AdvanceToNextTurn()
	cloned.AdvanceToNextTurn()
	if cloned.ActivePlayerObj().PlayerID() != b || cloned.CurrentTurn() != 4 {
		t.Fatal("native and harness turn advancement differ")
	}
}
