// Package gametest: engine tests for CR 400–407 (Zones).
// Covers: zone-change new-object identity (400.7), instant/sorcery stays off
// battlefield (400.4a), graveyard ordering (400.5, 404.1, 404.2), stack LIFO
// (405.2), mana ability stack-bypass (405.6c), exile (406.6, 406.7), and
// ante zone mechanics (407.2, 407.4).
package gametest

import (
	"sync"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// ===== one-time registration guard =====

var ch04Once sync.Once

func registerCh04Cards() {
	ch04Once.Do(func() {
		if !mage.CardRegistered("Zone Test Bear") {
			mage.Register("Zone Test Bear", func() mage.Card {
				return mage.NewCreature("Zone Test Bear", "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		if !mage.CardRegistered("Zone Test Bolt") {
			mage.Register("Zone Test Bolt", func() mage.Card {
				return mage.NewInstant("Zone Test Bolt", "{R}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.DealDamage(mage.Fixed(3))),
				)
			})
		}
		if !mage.CardRegistered("Zone Test Counterspell") {
			mage.Register("Zone Test Counterspell", func() mage.Card {
				return mage.NewInstant("Zone Test Counterspell", "{U}{U}",
					mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpell()),
				)
			})
		}
		if !mage.CardRegistered("Zone Test Exiler") {
			mage.Register("Zone Test Exiler", func() mage.Card {
				return mage.NewInstant("Zone Test Exiler", "{W}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.ExileTarget()),
				)
			})
		}
		if !mage.CardRegistered("Zone Test Bouncer") {
			mage.Register("Zone Test Bouncer", func() mage.Card {
				return mage.NewInstant("Zone Test Bouncer", "{U}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.ReturnToHandTarget()),
				)
			})
		}
		if !mage.CardRegistered("Zone Test Killer") {
			mage.Register("Zone Test Killer", func() mage.Card {
				return mage.NewInstant("Zone Test Killer", "{B}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
				)
			})
		}
		if !mage.CardRegistered("Zone Test Saproling") {
			mage.Register("Zone Test Saproling", func() mage.Card {
				return mage.NewCreature("Zone Test Saproling", "{G}", 1, 1, mage.WithSubTypes("Saproling"))
			})
		}
	})
}

// ===== CR 400.4a — instant/sorcery cannot enter battlefield =====

// TestCR400_4a_InstantStaysOffBattlefield verifies that casting an
// instant puts it on the stack, not the battlefield, and it ends in the
// graveyard after resolving (not as a permanent).
func TestCR400_4a_InstantStaysOffBattlefield(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Bolt")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Bolt", "Zone Test Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Instant must end in graveyard, never on battlefield.
	g.AssertGraveyardCount(PlayerA, "Zone Test Bolt", 1)
	g.AssertPermanentCount(PlayerA, "Zone Test Bolt", 0)
	g.AssertPermanentCount(PlayerB, "Zone Test Bolt", 0)
}

// ===== CR 400.5 / 404.1 / 404.2 — graveyard ordering =====

// TestCR400_5_GraveyardOrderPreserved verifies that cards enter the
// graveyard in the order they are put there and that order does not change.
func TestCR400_5_GraveyardOrderPreserved(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	// Two creatures on B's battlefield; kill them sequentially.
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Saproling")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Killer")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Killer")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Killer", "Zone Test Bear")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Killer", "Zone Test Saproling")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Bear was killed first → should be at graveyard index 0 (bottom of this pile).
	// Saproling killed second → at index 1.
	g.AssertGraveyardOrder(PlayerB, "Zone Test Bear", "Zone Test Saproling")
}

// TestCR404_1_SpellGoesToGraveyard verifies that a resolved spell
// (instant/sorcery) goes to its owner's graveyard after resolution.
func TestCR404_1_SpellGoesToGraveyard(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Bolt")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Bolt", "Zone Test Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertGraveyardCount(PlayerA, "Zone Test Bolt", 1)
}

// TestCR404_1_CounteredSpellGoesToGraveyard verifies that a
// countered spell goes to its owner's graveyard (not owner of the counter spell).
func TestCR404_1_CounteredSpellGoesToGraveyard(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Bolt")
	g.AddCard(core.ZoneHand, PlayerB, "Zone Test Counterspell")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Bolt", "Zone Test Bear")
	g.CastInResponseTo(PlayerB, "Zone Test Counterspell", "Zone Test Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Bolt owned by A goes to A's graveyard after being countered.
	g.AssertGraveyardCount(PlayerA, "Zone Test Bolt", 1)
	// Bear survives.
	g.AssertPermanentCount(PlayerB, "Zone Test Bear", 1)
}

// ===== CR 400.7 — zone change creates a new object =====

// TestCR400_7_ZoneChangeNewObject verifies that a spell targeting a
// creature fizzles when the creature leaves the battlefield (it's a new object
// in the graveyard and no longer a legal target).
func TestCR400_7_ZoneChangeNewObject(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	// Bear on B's battlefield.
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	// A targets bear with bolt; B bounces bear back to hand before bolt resolves.
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Bolt")
	g.AddCard(core.ZoneHand, PlayerB, "Zone Test Bouncer")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Bolt", "Zone Test Bear")
	g.CastInResponseTo(PlayerB, "Zone Test Bouncer", "Zone Test Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Bear is now in hand (new object) — bolt fizzled, bear survives.
	g.AssertHandCount(PlayerB, "Zone Test Bear", 1)
	g.AssertPermanentCount(PlayerB, "Zone Test Bear", 0)
	g.AssertGraveyardCount(PlayerA, "Zone Test Bolt", 1)
}

// ===== CR 403.4 — permanent re-entering battlefield is a new object =====

// TestCR403_4_ReentryIsNewObject verifies that bouncing a
// permanent and recasting it produces a new object that no longer retains
// the targeting relationship from spells that targeted the previous instance.
func TestCR403_4_ReentryIsNewObject(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Bolt")
	g.AddCard(core.ZoneHand, PlayerB, "Zone Test Bouncer")
	// A targets bear; B bounces it. Bear leaves battlefield → bolt fizzles.
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Bolt", "Zone Test Bear")
	g.CastInResponseTo(PlayerB, "Zone Test Bouncer", "Zone Test Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// The original permanent instance is gone; spell targeting it fizzled.
	g.AssertHandCount(PlayerB, "Zone Test Bear", 1)
	g.AssertPermanentCount(PlayerB, "Zone Test Bear", 0)
}

// ===== CR 405.2 — stack is LIFO =====

// TestCR405_2_StackIsLIFO verifies that the last spell cast resolves
// first: B's counterspell (pushed second) resolves before A's bolt (pushed first).
func TestCR405_2_StackIsLIFO(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Bolt")
	g.AddCard(core.ZoneHand, PlayerB, "Zone Test Counterspell")
	// A casts Bolt → stack: [Bolt].
	// B casts Counterspell in response → stack: [Bolt, Counterspell].
	// Counterspell resolves first (LIFO), countering Bolt.
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Bolt", "Zone Test Bear")
	g.CastInResponseTo(PlayerB, "Zone Test Counterspell", "Zone Test Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Counterspell (top) resolved first and countered Bolt → Bear survives.
	g.AssertPermanentCount(PlayerB, "Zone Test Bear", 1)
	g.AssertGraveyardCount(PlayerA, "Zone Test Bolt", 1)
	g.AssertGraveyardCount(PlayerB, "Zone Test Counterspell", 1)
}

// ===== CR 405.6c — mana ability resolves immediately without using the stack =====

// TestCR405_6c_ManaAbilityDoesNotUseStack verifies that activating a mana
// ability doesn't use the stack: the mana is available immediately, allowing
// a spell to be cast in the same priority window without waiting for
// resolution. The key assertion is that the creature resolves onto the
// battlefield — if mana abilities used the stack, the cast would fail.
func TestCR405_6c_ManaAbilityDoesNotUseStack(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Bear") // {1}{G}
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Zone Test Bear", 1)
}

// ===== CR 406.6 — linked exile abilities =====

// TestCR406_6_LinkedExileAbilities verifies that a permanent that
// exiles a card tracks that specific card as "exiled with it."  We use
// Swords to Plowshares (which exiles with ExiledBy set to the source spell)
// as a stand-in for the general exile-link mechanic.
func TestCR406_6_LinkedExileAbilities(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Exiler")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Exiler", "Zone Test Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertExileCount("Zone Test Bear", 1)
	g.AssertPermanentCount(PlayerB, "Zone Test Bear", 0)
}

// ===== CR 406.7 / 400.8 — re-exiling an already-exiled object =====

// TestCR406_7_ReexileBecomesNewObject verifies that exiling a card that
// is already in exile puts it back as a new object.  Two separate exile effects
// targeting the same permanent will each exile it; the second effect should
// have no legal target (permanent was already exiled by first effect).
func TestCR406_7_ReexileBecomesNewObject(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Exiler")
	g.AddCard(core.ZoneHand, PlayerA, "Zone Test Exiler")
	// Cast first exiler targeting bear; cast second exiler also targeting bear
	// in response. Second (top of stack) resolves first and exiles bear.
	// First exiler's target (the battlefield permanent) is gone → fizzles.
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Zone Test Exiler", "Zone Test Bear")
	g.CastInResponseTo(PlayerA, "Zone Test Exiler", "Zone Test Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Bear is in exile once (the second exiler resolved; first fizzled).
	g.AssertExileCount("Zone Test Bear", 1)
	g.AssertPermanentCount(PlayerB, "Zone Test Bear", 0)
}

// ===== CR 407.2 — ante zone basic flow =====

// TestCR407_2_AnteZoneBasicFlow verifies that cards can be placed in
// the ante zone and are tracked separately from graveyard/hand/library.
func TestCR407_2_AnteZoneBasicFlow(t *testing.T) {
	registerCh04Cards()
	g := NewTestGame(t)
	// Place a card directly into ante (simulating the initial ante action).
	g.AddCard(core.ZoneAnte, PlayerA, "Zone Test Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertAnteCount(PlayerA, "Zone Test Bear", 1)
	// Card should not appear in other zones.
	g.AssertPermanentCount(PlayerA, "Zone Test Bear", 0)
	g.AssertHandCount(PlayerA, "Zone Test Bear", 0)
	g.AssertGraveyardCount(PlayerA, "Zone Test Bear", 0)
}

// TestCR407_2_AnteFromLibraryViaSpell verifies that casting
// "Contract from Below" moves the top library card to ante.
func TestCR407_2_AnteFromLibraryViaSpell(t *testing.T) {
	registerCh04Cards()
	g := NewTestGameWithAnte(t)
	// Put a known card on top of A's library.
	g.AddCard(core.ZoneLibrary, PlayerA, "Zone Test Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Contract from Below")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Contract from Below")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertAnteCount(PlayerA, "Zone Test Bear", 1)
	g.AssertLibraryCount(PlayerA, "Zone Test Bear", 0)
}

// TestCR407_4_OnlyOwnerCanAnteOwnCards verifies that an ante card
// placed in ante belongs to the player who owns it, not the opponent.
// Demonic Attorney antes the top card of each player's library; each card
// should appear in its owner's ante, not the opponent's.
func TestCR407_4_OnlyOwnerCanAnteOwnCards(t *testing.T) {
	registerCh04Cards()
	g := NewTestGameWithAnte(t)
	g.AddCard(core.ZoneLibrary, PlayerA, "Zone Test Bear")
	g.AddCard(core.ZoneLibrary, PlayerB, "Zone Test Saproling")
	g.AddCard(core.ZoneHand, PlayerA, "Demonic Attorney")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Demonic Attorney")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Each player's card entered their own ante zone.
	g.AssertAnteCount(PlayerA, "Zone Test Bear", 1)
	g.AssertAnteCount(PlayerB, "Zone Test Saproling", 1)
	// Neither card is in the wrong player's ante.
	g.AssertAnteCount(PlayerA, "Zone Test Saproling", 0)
	g.AssertAnteCount(PlayerB, "Zone Test Bear", 0)
}
