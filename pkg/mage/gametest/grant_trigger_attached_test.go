package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var grantTrigAttachedOnce sync.Once

func registerGrantTriggerAttachedCards() {
	grantTrigAttachedOnce.Do(func() {
		// Test aura that grants the enchanted creature an "at the beginning of
		// your upkeep, you gain 1 life" triggered ability — modelled after
		// Stalwart Aven's "enchanted creature gains <triggered ability>".
		if !mage.CardRegistered("Test Trig Aura") {
			mage.Register("Test Trig Aura", func() mage.Card {
				return mage.NewAura("Test Trig Aura", "{W}",
					mage.WithStaticAbility(
						mage.GrantTriggeredAbilityToAttached(
							core.EvtUpkeep, false,
							mage.EventPlayerIsController{},
							mage.GainLife(1),
						),
					),
				)
			})
		}
	})
}

// TestGrantTriggeredAbilityToAttached verifies the granted trigger fires
// from the enchanted creature, not the aura.
func TestGrantTriggeredAbilityToAttached(t *testing.T) {
	registerGrantTriggerAttachedCards()

	g := NewTestGame(t)
	bear := g.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	aura := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Trig Aura")
	g.Attach(aura, bear)
	g.SetLife(PlayerA, 20)
	g.StopAt(2, core.PrecombatMain)
	g.Execute()

	g.AssertLife(PlayerA, 21)
}
