package gametest

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// registerOnce guards one-time test card registration for replacement tests.
var replacementRegistered sync.Once

func registerReplacementTestCards() {
	replacementRegistered.Do(func() {
		// A vanilla 5/5 for testing damage scenarios.
		if !mage.CardRegistered("Test Giant") {
			mage.Register("Test Giant", func() mage.Card {
				return mage.NewCreature("Test Giant", "{3}{R}{R}", 5, 5,
					mage.WithSubTypes("Giant"),
				)
			})
		}

		// A 3/3 with regeneration for testing regeneration replacement.
		if !mage.CardRegistered("Regenerating Troll") {
			mage.Register("Regenerating Troll", func() mage.Card {
				return mage.NewCreature("Regenerating Troll", "{2}{G}", 3, 3,
					mage.WithSubTypes("Troll"),
					mage.WithActivatedAbility(
						mage.RegenerateSource(),
						mage.ManaCostOf("{G}"),
					),
				)
			})
		}

		// A 2/2 creature whose damage source is red (for color prevention).
		if !mage.CardRegistered("Red Goblin") {
			mage.Register("Red Goblin", func() mage.Card {
				return mage.NewCreature("Red Goblin", "{1}{R}", 2, 2,
					mage.WithSubTypes("Goblin"),
				)
			})
		}

		// A sorcery that deals 6 damage for testing prevention shields.
		if !mage.CardRegistered("Big Blast") {
			mage.Register("Big Blast", func() mage.Card {
				return mage.NewSorcery("Big Blast", "{4}{R}{R}",
					mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(6))),
				)
			})
		}

		// A simple destroy spell (allows regeneration) for testing regen replacement.
		if !mage.CardRegistered("Test Destroy") {
			mage.Register("Test Destroy", func() mage.Card {
				return mage.NewInstant("Test Destroy", "{1}{B}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
				)
			})
		}

		// An artifact creature for testing type prevention.
		if !mage.CardRegistered("Iron Golem") {
			mage.Register("Iron Golem", func() mage.Card {
				return mage.NewCreature("Iron Golem", "{4}", 3, 3,
					mage.WithSubTypes("Golem"),
					mage.WithCardType(core.TypeArtifact),
				)
			})
		}

		if !mage.CardRegistered("Counter Shield Creature") {
			mage.Register("Counter Shield Creature", func() mage.Card {
				return mage.NewCreature("Counter Shield Creature", "{3}{G}", 5, 5,
					mage.WithAbility(mage.EntersWithNCounters(core.P1P1, 2)),
					mage.WithStaticAbility(mage.PreventDamageToSourceByRemovingCounters(core.P1P1)),
				)
			})
		}

		if !mage.CardRegistered("Self Shield Creature") {
			mage.Register("Self Shield Creature", func() mage.Card {
				return mage.NewCreature("Self Shield Creature", "{2}{R}", 3, 3,
					mage.WithActivatedAbility(
						mage.PreventDamageToTarget(mage.Fixed(1)).Targeting(mage.ToSource()),
						mage.ManaCostOf("{R}"),
					),
				)
			})
		}
	})
}

// ===== Replacement Pipeline Edge Cases =====

// TestPreventionShieldPartial tests that a prevention shield absorbs partial
// damage and lets the remainder through.
func TestPreventionShieldPartial(t *testing.T) {
	registerReplacementTestCards()

	t.Run("partial_prevention", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneHand, PlayerA, "Big Blast")

		// Give PlayerB a prevention shield of 4 (Big Blast deals 6)
		g.AddPreventionShield(g.AllPlayers()[1].PlayerID(), 4)

		g.CastSpell(1, core.PrecombatMain, PlayerA, "Big Blast", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 6 damage - 4 prevented = 2 damage taken
		g.AssertLife(PlayerB, 18)
	})

	t.Run("full_prevention", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

		// Give PlayerB a prevention shield of 5 (Lightning Bolt deals 3)
		g.AddPreventionShield(g.AllPlayers()[1].PlayerID(), 5)

		g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 3 damage fully prevented
		g.AssertLife(PlayerB, 20)
	})
}

func TestPreventDamageToSourceByRemovingCounters(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Shield Creature")
	g.AddCard(core.ZoneHand, PlayerB, "Big Blast")
	g.CastSpell(2, core.PrecombatMain, PlayerB, "Big Blast", "Counter Shield Creature")
	g.StopAt(2, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Counter Shield Creature", 1)
	g.AssertCounterCount(PlayerA, "Counter Shield Creature", core.P1P1, 0)
	creature := g.FindPermanentByName("Counter Shield Creature", g.GetPlayer(PlayerA).PlayerID())
	if creature == nil {
		t.Fatal("Counter Shield Creature is not on the battlefield")
	}
	if creature.Damage != 4 {
		t.Fatalf("damage after preventing two by removing counters = %d, want 4", creature.Damage)
	}
}

func TestPreventDamageToTargetTargetingSource(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Self Shield Creature")
	g.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Self Shield Creature")
	g.CastSpell(1, core.PrecombatMain, PlayerB, "Lightning Bolt", "Self Shield Creature")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Self Shield Creature", 1)
	creature := g.FindPermanentByName("Self Shield Creature", g.GetPlayer(PlayerA).PlayerID())
	if creature == nil {
		t.Fatal("Self Shield Creature is not on the battlefield")
	}
	if creature.Damage != 2 {
		t.Fatalf("damage after source shield = %d, want 2", creature.Damage)
	}
}

// TestFogNonCombatPassesThrough verifies that fog only prevents combat damage,
// not non-combat damage.
func TestFogNonCombatPassesThrough(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

	// Register fog
	g.SetPreventCombatDamage()

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Lightning Bolt is non-combat damage; fog doesn't prevent it
	g.AssertLife(PlayerB, 17)
}

// TestMultipleReplacementsInteract tests two replacements applying to the same
// action: prevention shield + minimum life together.
func TestMultipleReplacementsInteract(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.SetLife(PlayerB, 5)
	g.AddCard(core.ZoneHand, PlayerA, "Big Blast")

	// Give PlayerB a prevention shield of 2 AND minimum life protection
	g.AddPreventionShield(g.AllPlayers()[1].PlayerID(), 2)
	g.SetMinimumLife(g.AllPlayers()[1].PlayerID())

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Big Blast", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Big Blast deals 6. Prevention absorbs 2 → 4 remaining.
	// Minimum life caps damage so life stays >= 1: life=5, max_damage=4, so 4 goes through.
	// Result: 5 - 4 = 1
	g.AssertLife(PlayerB, 1)
}

// TestRedirectionChangesActionType verifies that a replacement can change
// DamageToPlayerAction into DamageToCreatureAction (bodyguard pattern).
func TestRedirectionChangesActionType(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Giant")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant") // 3/3

	// Redirect all damage from Test Giant to Hill Giant
	testGiantPerm := findPermanentByName(g, "Test Giant")
	hillGiantPerm := findPermanentByName(g, "Hill Giant")
	g.SetAttackerDamageRedirect(testGiantPerm.ID(), hillGiantPerm.ID())

	g.Attack(1, PlayerA, "Test Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	// Test Giant's 5 combat damage redirected from PlayerB to Hill Giant
	g.AssertLife(PlayerB, 20) // no player damage
	// Hill Giant took 5 damage on a 3/3 → dead
	g.AssertGraveyardCount(PlayerB, "Hill Giant", 1)
}

// TestCreatureDamageRedirectToPlayer tests the Jade Monolith pattern:
// redirect damage from creature to player.
func TestCreatureDamageRedirectToPlayer(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant") // 3/3
	g.AddCard(core.ZoneBattlefield, PlayerB, "Test Giant") // 5/5
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

	// Redirect damage dealt to Test Giant → PlayerB instead
	tg := findPermanentByName(g, "Test Giant")
	g.SetCreatureDamageRedirect(tg.ID(), g.AllPlayers()[1].PlayerID())

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "Test Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Lightning Bolt's 3 damage redirected from Test Giant to PlayerB
	g.AssertLife(PlayerB, 17)
	// Test Giant survives
	g.AssertPermanentCount(PlayerB, "Test Giant", 1)
}

// TestRegenerationSurvivesDestroy verifies that regeneration replaces
// destruction with tap + remove damage.
func TestRegenerationSurvivesDestroy(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Regenerating Troll")
	g.AddCard(core.ZoneHand, PlayerB, "Test Destroy")

	// Activate regeneration to get a shield, then destroy it
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Regenerating Troll")
	g.CastSpell(1, core.PrecombatMain, PlayerB, "Test Destroy", "Regenerating Troll")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Regeneration shield should prevent destruction
	g.AssertPermanentCount(PlayerA, "Regenerating Troll", 1)
	g.AssertTapped(PlayerA, "Regenerating Troll", true)
}

// TestColorPreventionConsumed verifies that color prevention is one-shot:
// it prevents the first matching source then expires.
func TestColorPreventionConsumed(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

	// Add one-shot red prevention
	g.AddColorPrevention(g.AllPlayers()[1].PlayerID(), core.Red)

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// First bolt prevented, second bolt deals 3
	g.AssertLife(PlayerB, 17)
}

// TestSkipDrawReplacement verifies that skip-draw prevents the normal draw
// step draw but not effect-based draws.
func TestSkipDrawReplacement(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	// Both players start with 0 cards in hand, some in library
	g.AddCard(core.ZoneLibrary, PlayerA, "Hill Giant")
	g.AddCard(core.ZoneLibrary, PlayerA, "Hill Giant")
	g.AddCard(core.ZoneLibrary, PlayerA, "Hill Giant")

	// Skip PlayerA's next normal draw
	g.SetSkipNextDraw(g.AllPlayers()[0].PlayerID())

	// Turn 1 draw step: PlayerA's draw should be skipped
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHandCount(PlayerA, "Hill Giant", 0) // draw was skipped
}

// TestMinimumLifeDoesNotGoBelow1 verifies that with minimum life active,
// damage cannot reduce life below 1. Sets the Rules flag directly (as Ali
// from Cairo's continuous effect does each cycle).
func TestMinimumLifeDoesNotGoBelow1(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.SetLife(PlayerB, 3)
	g.AddCard(core.ZoneHand, PlayerA, "Big Blast") // deals 6

	// Set minimum life via the Rules flag directly (the same path Ali from Cairo uses).
	// The fallback in executeDamageToPlayer checks this flag.
	g.SetMinimumLife(g.AllPlayers()[1].PlayerID())

	// Normally this flag is reset each Apply() cycle and re-set by the continuous effect.
	// For the test, set it right before the spell resolves using a static ability.
	// Instead, we use a test creature that re-sets it each cycle.
	if !mage.CardRegistered("Min Life Bearer") {
		mage.Register("Min Life Bearer", func() mage.Card {
			return mage.NewCreature("Min Life Bearer", "{2}{R}{R}", 0, 1,
				mage.WithStaticAbility(mage.FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield,
					func(g *mage.Game, sourceID uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						g.SetMinimumLife(perm.ControllerID())
						return nil
					})),
			)
		})
	}
	g.AddCard(core.ZoneBattlefield, PlayerB, "Min Life Bearer")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Big Blast", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Big Blast deals 6, but life can't go below 1. life=3, maxDamage=2.
	g.AssertLife(PlayerB, 1)
}

// TestReplacementPipelineDeduplication verifies that each replacement fires
// at most once per event, preventing infinite loops.
func TestReplacementPipelineDeduplication(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

	// Register two prevention shields on PlayerB
	pid := g.AllPlayers()[1].PlayerID()
	g.AddPreventionShield(pid, 1) // absorbs 1
	g.AddPreventionShield(pid, 1) // absorbs 1

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// 3 damage. First shield absorbs 1 → 2 remaining. Second shield absorbs 1 → 1 remaining.
	// 1 damage gets through.
	g.AssertLife(PlayerB, 19)
}

// TestForcefieldDoesNotAffectBlockedDamage verifies that forcefield only
// reduces UNBLOCKED combat damage. Blocked creatures deal normal damage.
func TestForcefieldDoesNotAffectBlockedDamage(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Giant") // 5/5
	g.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant") // 3/3

	// Give PlayerB forcefield against Test Giant
	tgPerm := g.FindPermanentByName("Test Giant", g.AllPlayers()[0].PlayerID())
	g.AddForcefieldShield(g.AllPlayers()[1].PlayerID(), tgPerm.ID())

	g.Attack(1, PlayerA, "Test Giant")
	g.Block(1, PlayerB, "Hill Giant", "Test Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	// Test Giant is blocked, so forcefield doesn't apply.
	// Hill Giant takes 5 damage (dies), Test Giant takes 3 damage (survives at 5/5 with 3 damage).
	g.AssertLife(PlayerB, 20) // no player damage (blocked)
	g.AssertGraveyardCount(PlayerB, "Hill Giant", 1)
	g.AssertPermanentCount(PlayerA, "Test Giant", 1) // survives with 3 damage
}

// TestTypePreventionArtifact tests that type prevention blocks damage from
// artifact sources.
func TestTypePreventionArtifact(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Iron Golem") // artifact creature 3/3

	// Prevent artifact damage to PlayerB
	g.AddTypePrevention(g.AllPlayers()[1].PlayerID(), core.TypeArtifact)

	g.Attack(1, PlayerA, "Iron Golem")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	// Iron Golem's combat damage prevented by type prevention
	g.AssertLife(PlayerB, 20)
}

// TestCustomReplacementEffect tests registering a card-level custom
// ReplacementEffect via the *Game API.
func TestCustomReplacementEffect(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

	// Register a custom replacement that doubles all damage to PlayerB
	pid := g.AllPlayers()[1].PlayerID()
	g.AddReplacementEffect(&doubleDamageReplacement{playerID: pid})

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Lightning Bolt deals 3, doubled to 6
	g.AssertLife(PlayerB, 14)
}

// doubleDamageReplacement is a custom test replacement that doubles damage.
type doubleDamageReplacement struct {
	playerID uuid.UUID
}

func (r *doubleDamageReplacement) Matches(a mage.Action, _ mage.GameReader) bool {
	act, ok := a.(*mage.DamageToPlayerAction)
	return ok && act.PlayerID() == r.playerID
}

func (r *doubleDamageReplacement) Replace(a mage.Action, _ *mage.Game) mage.Action {
	act := a.(*mage.DamageToPlayerAction)
	return act.WithAmount(act.Amount() * 2)
}

func (r *doubleDamageReplacement) SourceID() uuid.UUID             { return uuid.Nil }
func (r *doubleDamageReplacement) IsActive(_ mage.GameReader) bool { return true }
func (r *doubleDamageReplacement) GetDuration() core.Duration      { return core.EndOfTurn }
func (r *doubleDamageReplacement) Clone() mage.ReplacementEffect {
	c := *r
	return &c
}

// TestPreventionShieldOnCreature tests that prevention shields work on
// creature targets, not just players.
func TestPreventionShieldOnCreature(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant") // 3/3
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

	// Add prevention shield to Hill Giant
	hg := findPermanentByName(g, "Hill Giant")
	g.AddPreventionShield(hg.ID(), 2)

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// 3 damage - 2 prevented = 1 damage. Hill Giant survives (3/3 with 1 damage).
	g.AssertPermanentCount(PlayerB, "Hill Giant", 1)
}

// ===== Replacement Duration / End-of-Turn Cleanup =====

// TestForcefieldExpiresAtEndOfTurn verifies that a forcefield replacement
// added during turn 1 does not persist into turn 2.
func TestForcefieldExpiresAtEndOfTurn(t *testing.T) {
	registerReplacementTestCards()

	t.Run("forcefield_active_on_turn_1", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Test Giant") // 5/5

		tgPerm := g.FindPermanentByName("Test Giant", g.AllPlayers()[0].PlayerID())
		g.AddForcefieldShield(g.AllPlayers()[1].PlayerID(), tgPerm.ID())

		g.Attack(1, PlayerA, "Test Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(PlayerB, 19) // 5 reduced to 1
	})

	t.Run("forcefield_gone_on_turn_3", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Test Giant") // 5/5

		tgPerm := g.FindPermanentByName("Test Giant", g.AllPlayers()[0].PlayerID())
		g.AddForcefieldShield(g.AllPlayers()[1].PlayerID(), tgPerm.ID())

		g.Attack(1, PlayerA, "Test Giant")
		// Turn 3 (PlayerA's next turn): no new forcefield. Full damage.
		g.Attack(3, PlayerA, "Test Giant")
		g.StopAt(3, core.EndCombat)
		g.Execute()
		// Turn 1: 5 reduced to 1. Turn 3: full 5 damage. Total: 6.
		g.AssertLife(PlayerB, 14)
	})
}

// TestFogExpiresAtEndOfTurn verifies that fog only prevents combat damage
// for the turn it was activated and does not carry over.
func TestFogExpiresAtEndOfTurn(t *testing.T) {
	registerReplacementTestCards()

	t.Run("fog_active_on_turn_1", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant") // 3/3

		g.SetPreventCombatDamage()

		g.Attack(1, PlayerA, "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(PlayerB, 20) // prevented
	})

	t.Run("fog_gone_on_turn_3", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant") // 3/3

		g.SetPreventCombatDamage()

		g.Attack(1, PlayerA, "Hill Giant")
		// Turn 3: fog expired. Full damage.
		g.Attack(3, PlayerA, "Hill Giant")
		g.StopAt(3, core.EndCombat)
		g.Execute()
		// Turn 1: prevented. Turn 3: 3 damage.
		g.AssertLife(PlayerB, 17)
	})
}

// TestRegenerationShieldExpiresAtEndOfTurn verifies that an unused
// regeneration shield does not persist into the next turn.
func TestRegenerationShieldExpiresAtEndOfTurn(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Regenerating Troll")
	g.AddCard(core.ZoneHand, PlayerB, "Test Destroy")
	g.AddCard(core.ZoneHand, PlayerB, "Test Destroy")

	// Activate regeneration on turn 1 but don't use it
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Regenerating Troll")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(PlayerA, "Regenerating Troll", 1)

	// Turn 2: destroy the troll without a fresh shield — it should die
	g.CastSpell(2, core.PrecombatMain, PlayerB, "Test Destroy", "Regenerating Troll")
	g.StopAt(2, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(PlayerA, "Regenerating Troll", 0)
	g.AssertGraveyardCount(PlayerA, "Regenerating Troll", 1)
}

// TestPreventionShieldExpiresAtEndOfTurn verifies that an unused prevention
// shield is cleared at end of turn.
func TestPreventionShieldExpiresAtEndOfTurn(t *testing.T) {
	registerReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")

	// Add a prevention shield on turn 1 but don't trigger any damage.
	// Cast bolt on turn 2 — shield should be gone.
	g.AddPreventionShield(g.AllPlayers()[1].PlayerID(), 5)
	g.CastSpell(2, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(2, core.BeginCombat)
	g.Execute()
	g.AssertLife(PlayerB, 17)
}

// findPermanentByName is a test helper to find a permanent on the battlefield.
func findPermanentByName(g *TestGame, name string) *mage.Permanent {
	for _, p := range g.AllBattlefield() {
		if p.Name() == name {
			return p
		}
	}
	return nil
}
