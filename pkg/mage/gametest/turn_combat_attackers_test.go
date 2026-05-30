// Package gametest: turn-structure tests for the declare-attackers step.
// Covers CR 508.1a (attacker legality: summoning sickness, haste, untapped),
// CR 508.1f (tapping on attack is not a cost), CR 508.2a (post-declaration
// characteristic changes do not retro-trigger), and CR 508.8 (declare-blockers
// and combat-damage steps are skipped when no attackers are declared).
package gametest

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// TestDeclareAttackers covers CR 508 sub-rules that govern the declare-
// attackers step.
func TestDeclareAttackers(t *testing.T) {

	// CR 508.8 — The declare blockers and combat damage steps are skipped if
	// no creatures are declared as attackers.
	//
	// With no attacker declared, the EvtBlockersDecl event must not fire.
	// Contrast: when an attacker IS declared, it fires.
	t.Run("CR 508.8 no attackers skips declare-blockers step", func(t *testing.T) {
		registerCombatEventRecorder()

		// No attacker declared.
		combatRec.reset()
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Combat Event Recorder")
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		for _, e := range combatRec.snapshot() {
			if e == core.EvtBlockersDecl {
				t.Errorf("EvtBlockersDecl fired despite no attackers being declared")
			}
		}

		// Attacker declared — declare-blockers step runs, EvtBlockersDecl fires.
		combatRec.reset()
		tg2 := NewTestGame(t)
		tg2.AddCard(core.ZoneBattlefield, PlayerA, "Combat Event Recorder")
		tg2.Attack(1, PlayerA, "Combat Event Recorder")
		tg2.StopAt(1, core.PostcombatMain)
		tg2.Execute()
		sawBlockersDecl := false
		for _, e := range combatRec.snapshot() {
			if e == core.EvtBlockersDecl {
				sawBlockersDecl = true
			}
		}
		if !sawBlockersDecl {
			t.Errorf("EvtBlockersDecl did not fire when attacker was declared")
		}
	})

	// CR 508.1a — The active player chooses which creatures that they control,
	// if any, attack. The chosen creatures must be untapped, they can't have
	// summoning sickness, and each one must either have haste or have been
	// controlled by the active player continuously since the turn began.
	//
	// A creature cast this turn without haste cannot be declared as an
	// attacker. Contrast with a hasted creature.
	t.Run("CR 508.1a summoning-sick creature without haste cannot attack", func(t *testing.T) {
		name := "Combat Sick Bear"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 20) // summoning sickness blocked the attack
	})

	t.Run("CR 508.1a haste bypasses summoning sickness", func(t *testing.T) {
		name := "Combat Hasty Bear"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{1}{R}", 2, 2,
					mage.WithSubTypes("Bear"),
					mage.WithKeyword(core.Haste))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 18) // 2/2 with haste connected for 2
	})

	// CR 508.1a — Attacking creatures must be untapped. A tapped creature
	// cannot be declared as an attacker.
	t.Run("CR 508.1a tapped creature cannot attack", func(t *testing.T) {
		name := "Combat Tap Bear"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		permID := tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		perm := tg.FindPermanent(permID)
		if perm == nil {
			t.Fatalf("permanent not found after AddCard")
		}
		// Tap the permanent and prevent it from untapping during the untap step
		// so it is still tapped when the declare-attackers step runs.
		perm.Tapped = true
		perm.GrantBaseAttr(core.AttrDoesNotUntap)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertTapped(PlayerA, name, true) // still tapped at declare attackers
		tg.AssertLife(PlayerB, 20)           // tapped → couldn't attack
	})

	// CR 508.1f — The active player taps the chosen creatures. Tapping a
	// creature when it's declared as an attacker isn't a cost; attacking simply
	// causes creatures to become tapped.
	t.Run("CR 508.1f attacking causes creatures to become tapped", func(t *testing.T) {
		name := "Combat Tap-On-Attack Bear"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertTapped(PlayerA, name, true)
		tg.AssertLife(PlayerB, 18)
	})

	// CR 508.2a — Once an attacker has been declared, changing its
	// characteristics afterward does not retroactively cause any "whenever a
	// [type] creature attacks" ability to trigger. Triggers examining
	// characteristics evaluate them at declaration time; a subsequent color
	// change cannot bring back a trigger that did not fire.
	//
	// Setup: PlayerA controls a "Green Attack Watcher" whose triggered ability
	// reads "Whenever a green creature attacks, its controller gains 5 life"
	// (watcher-controller-agnostic; the trigger fires for any green attacker).
	// PlayerA attacks with a blue creature. In response to the declare-
	// attackers step, PlayerA turns the attacker green (via ChangeColorEffect).
	// Because color was blue at declaration, the watcher's trigger did not fire;
	// the post-declaration color change must not retro-trigger it.
	t.Run("CR 508.2a post-declaration color change does not retro-trigger attack ability", func(t *testing.T) {
		watcher := "508.2a Green Attack Watcher"
		attacker := "508.2a Blue Bear"
		lace := "508.2a Green Lace"

		if !mage.CardRegistered(watcher) {
			mage.Register(watcher, func() mage.Card {
				// "Whenever a green creature attacks, you gain 5 life."
				trig := mage.NewTriggered(core.EvtDeclaredAttacker, false,
					mage.GainLife(5)).
					SetCondition(func(evt *core.GameEvent, g mage.GameReader, _, _ uuid.UUID) bool {
						atk := g.FindPermanent(evt.SourceID)
						if atk == nil {
							return false
						}
						return slices.Contains(atk.Colors(), core.Green)
					})
				return mage.NewCreature(watcher, "{2}", 1, 1,
					mage.WithSubTypes("Spirit"),
					mage.WithAbility(trig))
			})
		}
		if !mage.CardRegistered(attacker) {
			mage.Register(attacker, func() mage.Card {
				return mage.NewCreature(attacker, "{1}{U}", 2, 2,
					mage.WithSubTypes("Bear"))
			})
		}
		if !mage.CardRegistered(lace) {
			mage.Register(lace, func() mage.Card {
				// Instant: target permanent becomes green (color override).
				return mage.NewInstant(lace, "{G}",
					mage.NewTargetedSpell(mage.TargetPermanent(),
						mage.ChangeColorEffect(core.Green)))
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, watcher)
		tg.AddCard(core.ZoneBattlefield, PlayerA, attacker)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
		tg.AddCard(core.ZoneHand, PlayerA, lace)
		tg.Attack(1, PlayerA, attacker)
		// Cast the lace in the declare-blockers step — AFTER the attacker has
		// already been declared. If the engine correctly scopes CR 508.2a, the
		// post-declaration color change must not retro-trigger the watcher.
		tg.CastSpell(1, core.DeclareBlockers, PlayerA, lace, attacker)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		// Attacker should now be green (color change took effect).
		perm := tg.FindPermanentByName(attacker, tg.GetPlayer(PlayerA).PlayerID())
		if perm == nil {
			t.Fatalf("attacker permanent not found")
		}
		sawGreen := false
		for _, c := range perm.Colors() {
			if c == core.Green {
				sawGreen = true
			}
		}
		if !sawGreen {
			t.Fatalf("color override did not apply; attacker colors=%v", perm.Colors())
		}
		// The critical assertion: watcher's trigger did NOT fire, so no 5 life
		// was gained. PlayerA's life must still be 20. CR 508.2a forbids the
		// post-declaration change from retro-triggering.
		tg.AssertLife(PlayerA, 20)
	})
}
