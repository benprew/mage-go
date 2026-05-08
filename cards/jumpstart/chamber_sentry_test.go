package jumpstart

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

// TestChamberSentry_EntersWithCountersForEachColorSpent verifies the ETB
// counter clause: "Chamber Sentry enters with a +1/+1 counter on it for each
// color of mana spent to cast it." Casting for X=3 with three different
// colors (W, U, R) puts 3 +1/+1 counters on it.
func TestChamberSentry_EntersWithCountersForEachColorSpent(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Chamber Sentry")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Chamber Sentry", 3)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Chamber Sentry", core.P1P1, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Chamber Sentry", 3, 3)
}

// TestChamberSentry_OneColorOnlyOneCounter verifies that paying X=3 with
// three Mountains (one distinct color) produces only one +1/+1 counter —
// it's "for each color of mana spent", not for each mana symbol.
func TestChamberSentry_OneColorOnlyOneCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Chamber Sentry")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Chamber Sentry", 3)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Chamber Sentry", core.P1P1, 1)
	g.AssertPowerToughness(gametest.PlayerA, "Chamber Sentry", 1, 1)
}

// TestChamberSentry_ZeroXDies verifies that casting Chamber Sentry for X=0
// (no mana spent, zero distinct colors) puts a 0/0 with no counters onto
// the battlefield, which dies to state-based actions.
func TestChamberSentry_ZeroXDies(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Chamber Sentry")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Chamber Sentry", 0)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Chamber Sentry", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Chamber Sentry", 1)
}

// TestChamberSentry_ActivatedDamageAbility verifies the activated ability
// {X}, {T}, Remove X +1/+1 counters from this creature: deals X damage to
// any target. With 3 counters and {3} available, activate for X=3 → 3
// damage to PlayerB; counters drop to 0.
func TestChamberSentry_ActivatedDamageAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	csID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Chamber Sentry")
	// Place 3 +1/+1 counters directly so the 0/0 base is a 3/3 from t=0,
	// avoiding state-based-action destruction before the scheduled counter
	// addition fires.
	if perm := g.FindPermanent(csID); perm != nil {
		perm.AddCounter(core.P1P1, 3)
	}
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Chamber Sentry", 3, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	// After paying the cost (remove all 3 +1/+1 counters), the now-0/0
	// Sentry dies to state-based actions.
	g.AssertPermanentCount(gametest.PlayerA, "Chamber Sentry", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Chamber Sentry", 1)
}

// TestChamberSentry_GraveyardReturnAbility verifies the graveyard-zone
// activated ability: {W}{U}{B}{R}{G}, return Chamber Sentry from your
// graveyard to your hand.
func TestChamberSentry_GraveyardReturnAbility(t *testing.T) {
	tg := gametest.NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Chamber Sentry")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	pid := tg.GetPlayer(gametest.PlayerA).PlayerID()

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	cs, idx, ok := tg.FindGraveyardActivatableCard(pid, "Chamber Sentry")
	if !ok {
		t.Fatal("graveyard activatable Chamber Sentry not found")
	}
	if err := tg.ActivateGraveyardAbility(pid, cs, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.ResolveStack()

	tg.AssertHandCount(gametest.PlayerA, "Chamber Sentry", 1)
	tg.AssertGraveyardCount(gametest.PlayerA, "Chamber Sentry", 0)
}
