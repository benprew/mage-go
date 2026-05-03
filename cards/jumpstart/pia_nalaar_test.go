package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// mageCreate is a tiny alias around mage.CreateCard for readable test setup.
func mageCreate(name string) (mage.Card, error) { return mage.CreateCard(name) }

// Pia Nalaar, Consul of Revival {2}{R}
// Legendary Creature — Human Artificer  3/2
// When this creature enters, create a 1/1 colorless Thopter artifact creature
// token with flying.
// Sacrifice an artifact: Creatures you control get +1/+0 until end of turn.
// At the beginning of your end step, if an opponent was dealt 3 or more damage
// this turn, you may pay {R}. If you do, return this card from your graveyard
// to the battlefield.

func TestPiaNalaar_EntersWithThopter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pia Nalaar, Consul of Revival")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Thopter", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 3, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Thopter", 1, 1)
	g.AssertHasAbility(gametest.PlayerA, "Thopter", core.Flying, true)
}

func TestPiaNalaar_SacArtifactBoost(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pia Nalaar, Consul of Revival")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mox Ruby")
	g.ChoosePermanent(gametest.PlayerA, "Mox Ruby")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Pia Nalaar, Consul of Revival")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 1)
	// Pia's ETB also creates a Thopter; both Pia and the Bears (and the Thopter)
	// should each have +1/+0 until end of turn.
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 4, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Thopter", 2, 1)
}

func TestPiaNalaar_GraveyardReturnAfterOpponent3Damage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Pia Nalaar, Consul of Revival")
	// Two Mountains: one taps to cast Lightning Bolt, the other pays the
	// {R} of Pia's may-pay end-step trigger.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	// Cast Lightning Bolt on PlayerB → 3 damage this turn.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.Cleanup)
	g.Execute()

	// At PlayerA's end step, the trigger resolves; PlayerA accepts the
	// may-pay (default true) and pays {R} from the Mountain. Pia returns.
	g.AssertPermanentCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 0)
	g.AssertLife(gametest.PlayerB, 17)
}

func TestPiaNalaar_NoReturnWhenOpponentDamageBelowThreshold(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Pia Nalaar, Consul of Revival")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	// 2 damage to PlayerB (below the 3-damage threshold). Use Shock.
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "PlayerB")
	g.StopAt(1, core.Cleanup)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 0)
	g.AssertLife(gametest.PlayerB, 18)
}

func TestPiaNalaar_DeclineMayPay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Pia Nalaar, Consul of Revival")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	// Decline the optional {R} payment when the trigger resolves.
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.StopAt(1, core.Cleanup)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 0)
}

func TestPiaNalaar_NoTriggerFromExile(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	// Place Pia in exile via the engine (AddCard does not support ZoneExile).
	pia, err := mageCreate("Pia Nalaar, Consul of Revival")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	pia.SetOwner(g.GetPlayer(gametest.PlayerA).PlayerID())
	g.Game.ExileCard(pia, g.GetPlayer(gametest.PlayerA).PlayerID())

	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.Cleanup)
	g.Execute()

	// Trigger functions while in graveyard only, not exile.
	g.AssertPermanentCount(gametest.PlayerA, "Pia Nalaar, Consul of Revival", 0)
	g.AssertExileCount("Pia Nalaar, Consul of Revival", 1)
}
