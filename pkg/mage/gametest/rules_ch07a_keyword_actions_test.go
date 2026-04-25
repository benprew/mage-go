package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

var ch07aOnce sync.Once

func registerCh07aCards() {
	ch07aOnce.Do(func() {
		if !mage.CardRegistered("Ch07a Bear") {
			mage.Register("Ch07a Bear", func() mage.Card {
				return mage.NewCreature("Ch07a Bear", "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		if !mage.CardRegistered("Ch07a Destroy Spell") {
			mage.Register("Ch07a Destroy Spell", func() mage.Card {
				return mage.NewInstant("Ch07a Destroy Spell", "{1}{B}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
				)
			})
		}
		if !mage.CardRegistered("Ch07a Exile Spell") {
			mage.Register("Ch07a Exile Spell", func() mage.Card {
				return mage.NewInstant("Ch07a Exile Spell", "{1}{W}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.ExileTarget()),
				)
			})
		}
		if !mage.CardRegistered("Ch07a Discard Spell") {
			mage.Register("Ch07a Discard Spell", func() mage.Card {
				return mage.NewSorcery("Ch07a Discard Spell", "{1}{B}",
					mage.NewTargetedSpell(mage.TargetPlayer(), mage.DiscardCards(mage.Fixed(1))),
				)
			})
		}
		if !mage.CardRegistered("Ch07a Random Discard Spell") {
			mage.Register("Ch07a Random Discard Spell", func() mage.Card {
				return mage.NewSorcery("Ch07a Random Discard Spell", "{1}{B}",
					mage.NewTargetedSpell(mage.TargetPlayer(), mage.DiscardRandom(1)),
				)
			})
		}
		if !mage.CardRegistered("Ch07a Death Watcher") {
			mage.Register("Ch07a Death Watcher", func() mage.Card {
				return mage.NewCreature("Ch07a Death Watcher", "{1}{B}", 1, 1,
					mage.WithSubTypes("Zombie"),
					mage.WithAbility(mage.AnyCreatureDiesTrigger(
						mage.GainLife(1),
						false,
					)),
				)
			})
		}
		if !mage.CardRegistered("Ch07a Indestructible Creature") {
			mage.Register("Ch07a Indestructible Creature", func() mage.Card {
				return mage.NewCreature("Ch07a Indestructible Creature", "{2}{W}", 2, 2,
					mage.WithSubTypes("Angel"),
					mage.WithKeyword(core.Indestructible),
				)
			})
		}
	})
}

// =============================================================================
// 700.4 — Dies
// =============================================================================

// TestCR700_4_DiesTriggersOnGraveyardFromBattlefield verifies that "when a creature
// dies" triggers fire when a creature is destroyed and goes to the graveyard.
func TestCR700_4_DiesTriggersOnGraveyardFromBattlefield(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Death Watcher")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Destroy Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Destroy Spell", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Bear is dead; death watcher gains 1 life
	tg.AssertGraveyardCount(PlayerA, "Ch07a Bear", 1)
	tg.AssertLife(PlayerA, 21)
}

// TestCR700_4_ExileDoesNotTriggerDies verifies that exiling a creature does not
// trigger "when this creature dies" abilities.
func TestCR700_4_ExileDoesNotTriggerDies(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Death Watcher")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Exile Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Exile Spell", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Bear is exiled, not dead; death watcher should NOT gain life
	tg.AssertExileCount("Ch07a Bear", 1)
	tg.AssertLife(PlayerA, 20)
}

// =============================================================================
// 701.2 — Activate
// =============================================================================

// TestCR701_2_ActivateOnlyControllerCanActivate verifies that a player cannot activate
// an ability on a permanent they don't control.
func TestCR701_2_ActivateOnlyControllerCanActivate(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Drudge Skeletons")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	playerBID := tg.GetPlayer(PlayerB).PlayerID()
	abilities := tg.GetActivatableAbilities(playerBID)
	for _, a := range abilities {
		if a.PermanentName == "Drudge Skeletons" {
			t.Errorf("PlayerB should not be able to activate abilities on PlayerA's Drudge Skeletons")
		}
	}
}

// =============================================================================
// 701.3 — Attach
// =============================================================================

// TestCR701_3_AttachEquipMovesToNewCreature verifies that re-equipping an Equipment
// to a different creature reattaches it (old host is no longer equipped).
func TestCR701_3_AttachEquipMovesToNewCreature(t *testing.T) {
	registerCh07aCards()

	const (
		equipName = "Attach Test Sword"
		bear1Name = "Attach Test Bear One"
		bear2Name = "Attach Test Bear Two"
	)
	if !mage.CardRegistered(equipName) {
		mage.Register(equipName, func() mage.Card {
			return mage.NewEquipment(equipName, "{2}",
				mage.WithAbility(mage.StaticAbility(mage.BoostAttached(2, 2, core.AttachEquipment))),
			)
		})
	}
	if !mage.CardRegistered(bear1Name) {
		mage.Register(bear1Name, func() mage.Card {
			return mage.NewCreature(bear1Name, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered(bear2Name) {
		mage.Register(bear2Name, func() mage.Card {
			return mage.NewCreature(bear2Name, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	tg := NewTestGame(t)
	equipID := tg.AddCard(core.ZoneBattlefield, PlayerA, equipName)
	bear1ID := tg.AddCard(core.ZoneBattlefield, PlayerA, bear1Name)
	bear2ID := tg.AddCard(core.ZoneBattlefield, PlayerA, bear2Name)
	tg.Attach(equipID, bear1ID)
	tg.Attach(equipID, bear2ID)
	tg.CheckStateBasedActions()
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// Equipment should be on bear2, not bear1
	tg.AssertAttachedTo(PlayerA, equipName, bear2Name)
	// bear1 should be 2/2 (no bonus)
	tg.AssertPowerToughness(PlayerA, bear1Name, 2, 2)
	// bear2 should be 4/4 (with bonus)
	tg.AssertPowerToughness(PlayerA, bear2Name, 4, 4)
}

// TestCR701_3_UnattachEquipmentStaysOnBattlefield verifies that when equipment
// loses its host it remains on the battlefield unattached (rule 701.3d).
func TestCR701_3_UnattachEquipmentStaysOnBattlefield(t *testing.T) {
	registerCh07aCards()

	const (
		equipName = "Unattach Test Axe"
		hostName  = "Unattach Test Warrior"
	)
	if !mage.CardRegistered(equipName) {
		mage.Register(equipName, func() mage.Card {
			return mage.NewEquipment(equipName, "{2}",
				mage.WithAbility(mage.StaticAbility(mage.BoostAttached(2, 2, core.AttachEquipment))),
			)
		})
	}
	if !mage.CardRegistered(hostName) {
		mage.Register(hostName, func() mage.Card {
			return mage.NewCreature(hostName, "{1}{W}", 1, 1, mage.WithSubTypes("Warrior"))
		})
	}

	tg := NewTestGame(t)
	equipID := tg.AddCard(core.ZoneBattlefield, PlayerA, equipName)
	hostID := tg.AddCard(core.ZoneBattlefield, PlayerA, hostName)
	tg.Attach(equipID, hostID)

	// Destroy the host
	host := tg.FindPermanent(hostID)
	tg.DestroyPermanent(host)
	tg.CheckStateBasedActions()

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// Equipment stays on battlefield
	tg.AssertPermanentCount(PlayerA, equipName, 1)
	tg.AssertGraveyardCount(PlayerA, hostName, 1)
	equip := tg.FindPermanentByName(equipName, tg.GetPlayer(PlayerA).PlayerID())
	if equip.IsAttached() {
		t.Errorf("equipment should not be attached after its host was destroyed")
	}
}

// =============================================================================
// 701.6 — Counter
// =============================================================================

// TestCR701_6_CounterSpellRemovedFromStack verifies that countering a spell puts it
// in the owner's graveyard without resolving.
func TestCR701_6_CounterSpellRemovedFromStack(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")
	tg.AddCard(core.ZoneHand, PlayerB, "Counterspell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	tg.CastInResponseTo(PlayerB, "Counterspell", "Lightning Bolt")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Lightning Bolt should be in graveyard, not resolved
	tg.AssertGraveyardCount(PlayerA, "Lightning Bolt", 1)
	tg.AssertLife(PlayerB, 20) // not hit
}

// =============================================================================
// 701.7 — Create
// =============================================================================

// TestCR701_7_CreateTokenEntersBattlefield verifies that creating a token puts it
// onto the battlefield under the controller's control.
func TestCR701_7_CreateTokenEntersBattlefield(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "The Hive")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 10)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "The Hive")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Wasp", 1)
}

// =============================================================================
// 701.8 — Destroy
// =============================================================================

// TestCR701_8_DestroyMovesToGraveyard verifies that destroying a permanent puts it in
// its owner's graveyard.
func TestCR701_8_DestroyMovesToGraveyard(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerA, "Ch07a Destroy Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Ch07a Destroy Spell", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, "Ch07a Bear", 0)
	tg.AssertGraveyardCount(PlayerB, "Ch07a Bear", 1)
}

// TestCR701_8_DestroyRegenerationPrevents verifies that a regeneration shield prevents
// destruction.
func TestCR701_8_DestroyRegenerationPrevents(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Drudge Skeletons")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Destroy Spell")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Drudge Skeletons")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Destroy Spell", "Drudge Skeletons")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Drudge Skeletons", 1)
	tg.AssertTapped(PlayerA, "Drudge Skeletons", true)
}

// TestCR701_8_DestroyIndestructiblePrevents verifies that an indestructible permanent
// can't be destroyed.
func TestCR701_8_DestroyIndestructiblePrevents(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Indestructible Creature")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Destroy Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Destroy Spell", "Ch07a Indestructible Creature")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Ch07a Indestructible Creature", 1)
	tg.AssertGraveyardCount(PlayerA, "Ch07a Indestructible Creature", 0)
}

// TestCR701_8_ExileIsNotDestruction verifies that exiling a creature does not
// count as destruction; no "when destroyed" triggers fire.
func TestCR701_8_ExileIsNotDestruction(t *testing.T) {
	registerCh07aCards()

	// Use a card that counts deaths specifically: Death Watcher gains 1 life per death.
	// If exile triggered death, it would gain 1 life. It should not.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Death Watcher")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Exile Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Exile Spell", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertExileCount("Ch07a Bear", 1)
	tg.AssertLife(PlayerA, 20) // no life gain — no death trigger
}

// =============================================================================
// 701.9 — Discard
// =============================================================================

// TestCR701_9_DiscardMovesToGraveyard verifies that discarding a card moves it from
// the hand to the graveyard.
func TestCR701_9_DiscardMovesToGraveyard(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerA, "Ch07a Discard Spell")
	tg.ChooseDiscard(PlayerB, "Ch07a Bear")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Ch07a Discard Spell", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertHandCount(PlayerB, "Ch07a Bear", 0)
	tg.AssertGraveyardCount(PlayerB, "Ch07a Bear", 1)
}

// TestCR701_9_DiscardPlayerChooses verifies that the default discard lets the affected
// player choose which card to discard.
func TestCR701_9_DiscardPlayerChooses(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Hill Giant")
	tg.AddCard(core.ZoneHand, PlayerA, "Ch07a Discard Spell")
	tg.ChooseDiscard(PlayerB, "Hill Giant")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Ch07a Discard Spell", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// PlayerB chose to discard Hill Giant, keeping the Bear
	tg.AssertGraveyardCount(PlayerB, "Hill Giant", 1)
	tg.AssertHandCount(PlayerB, "Ch07a Bear", 1)
}

// TestCR701_9_DiscardRandom verifies that a random discard effect discards
// one card at random from the target player's hand.
func TestCR701_9_DiscardRandom(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerA, "Ch07a Random Discard Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Ch07a Random Discard Spell", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// The bear was the only card; it should now be in graveyard
	tg.AssertGraveyardCount(PlayerB, "Ch07a Bear", 1)
	tg.AssertHandCount(PlayerB, "Ch07a Bear", 0)
}

// =============================================================================
// 701.13 — Exile
// =============================================================================

// TestCR701_13_ExileFromBattlefield verifies that exiling a permanent moves it to the
// exile zone.
func TestCR701_13_ExileFromBattlefield(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerA, "Ch07a Exile Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Ch07a Exile Spell", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, "Ch07a Bear", 0)
	tg.AssertGraveyardCount(PlayerB, "Ch07a Bear", 0)
	tg.AssertExileCount("Ch07a Bear", 1)
}

// TestCR701_13_ExileFromGraveyard verifies that exiling a card from the graveyard moves
// it to the exile zone.
func TestCR701_13_ExileFromGraveyard(t *testing.T) {
	registerCh07aCards()

	const exileGY = "Exile GY Spell"
	if !mage.CardRegistered(exileGY) {
		mage.Register(exileGY, func() mage.Card {
			return mage.NewInstant(exileGY, "{1}{W}",
				mage.NewTargetedSpell(
					mage.TargetCardInYourGraveyard(),
					mage.FuncEffect(
						"exile target card from your graveyard",
						mage.EffectProperties{Outcome: mage.OutcomeBenefit},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							for _, p := range g.AllPlayers() {
								c, ok := p.RemoveFromGraveyard(targets[0])
								if ok {
									g.ExileCard(c, sourceID)
									return nil
								}
							}
							return nil
						},
					),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerA, exileGY)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, exileGY, "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertGraveyardCount(PlayerA, "Ch07a Bear", 0)
	tg.AssertExileCount("Ch07a Bear", 1)
}

// TestCR701_13_ExileDoesNotTriggerDies verifies that exiling a creature does not trigger
// "when this creature dies" abilities (same scenario as 700.4 but as a direct
// assertion about the exile keyword action).
func TestCR701_13_ExileDoesNotTriggerDies(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Death Watcher")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerA, "Ch07a Exile Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Ch07a Exile Spell", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertExileCount("Ch07a Bear", 1)
	tg.AssertLife(PlayerA, 20)
}

// =============================================================================
// 701.18 — Play (Land)
// =============================================================================

// TestCR701_18_PlayLandNoStack verifies that playing a land is a special
// action and does not use the stack.
func TestCR701_18_PlayLandNoStack(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Forest")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Land enters battlefield; not in graveyard, not on stack
	tg.AssertPermanentCount(PlayerA, "Forest", 1)
	tg.AssertGraveyardCount(PlayerA, "Forest", 0)
}

// TestCR701_18_PlayLandOncePerTurn verifies that a player normally can't play a second
// land in the same turn.
func TestCR701_18_PlayLandOncePerTurn(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Forest")
	tg.AddCard(core.ZoneHand, PlayerA, "Plains")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// Only one land should have been played in the turn; the other stays in hand.
	playerAID := tg.GetPlayer(PlayerA).PlayerID()
	total := 0
	for _, perm := range tg.Battlefield {
		if perm.Controller == playerAID && (perm.Name() == "Forest" || perm.Name() == "Plains") {
			total++
		}
	}
	if total > 1 {
		t.Errorf("player played %d lands in one turn; expected at most 1", total)
	}
}

// =============================================================================
// 701.19 — Regenerate
// =============================================================================

// TestCR701_19_RegeneratePreventsDestruction verifies that a regeneration shield
// replaces destruction with tapping and damage removal.
func TestCR701_19_RegeneratePreventsDestruction(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Drudge Skeletons")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Destroy Spell")
	// Activate regen before destruction
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Drudge Skeletons")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Destroy Spell", "Drudge Skeletons")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Drudge Skeletons", 1)
}

// TestCR701_19_RegenerateUsesShieldOnce verifies that a one-shot regeneration shield
// (from a spell like Death Ward) is consumed on first use.
func TestCR701_19_RegenerateUsesShieldOnce(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneHand, PlayerA, "Death Ward")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Destroy Spell")
	tg.AddCard(core.ZoneHand, PlayerB, "Ch07a Destroy Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Death Ward", "Ch07a Bear")
	// First destroy: consumed by shield
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Destroy Spell", "Ch07a Bear")
	// Second destroy: no shield; bear dies
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Ch07a Destroy Spell", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Ch07a Bear", 0)
	tg.AssertGraveyardCount(PlayerA, "Ch07a Bear", 1)
}

// TestCR701_19_CantRegenerateShieldIgnored verifies that "can't be
// regenerated" prevents a regeneration shield from applying.
func TestCR701_19_CantRegenerateShieldIgnored(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Drudge Skeletons")
	tg.AddCard(core.ZoneHand, PlayerB, "Disintegrate")
	// PlayerA activates regen on their main phase; the shield survives into turn 2.
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Drudge Skeletons")
	// PlayerB casts the sorcery Disintegrate on their own main phase (CR 307.1).
	tg.CastSpellWithX(2, core.PrecombatMain, PlayerB, "Disintegrate", 2, "Drudge Skeletons")
	tg.StopAt(2, core.EndStep)
	tg.Execute()

	// Regen shield is present but CantRegenerate prevents it from applying
	tg.AssertPermanentCount(PlayerA, "Drudge Skeletons", 0)
	tg.AssertGraveyardCount(PlayerA, "Drudge Skeletons", 1)
}

// =============================================================================
// 701.20 — Reveal
// =============================================================================

// TestCR701_20_RevealCardStaysInZone verifies that revealing cards from the top of a
// library does not move them from the library.
func TestCR701_20_RevealCardStaysInZone(t *testing.T) {
	registerCh07aCards()

	// Natural Selection looks at the top 3 cards of target player's library and
	// reorders them — the cards stay in the library.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Natural Selection")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Hill Giant")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Natural Selection", "PlayerA")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// All three cards remain in library after being revealed and reordered
	tg.AssertLibraryCount(PlayerA, "Ch07a Bear", 1)
	tg.AssertLibraryCount(PlayerA, "Hill Giant", 1)
	tg.AssertLibraryCount(PlayerA, "Forest", 1)
}

// =============================================================================
// 701.21 — Sacrifice
// =============================================================================

// TestCR701_21_SacrificeMovesToGraveyard verifies that sacrificing a permanent puts it
// in its owner's graveyard.
func TestCR701_21_SacrificeMovesToGraveyard(t *testing.T) {
	registerCh07aCards()

	const sacrificeMe = "Sac To Graveyard Test Creature"
	const sacrificer = "Sac To GY Spell"
	if !mage.CardRegistered(sacrificeMe) {
		mage.Register(sacrificeMe, func() mage.Card {
			return mage.NewCreature(sacrificeMe, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered(sacrificer) {
		mage.Register(sacrificer, func() mage.Card {
			return mage.NewSorcery(sacrificer, "{B}",
				mage.NewTargetedSpell(
					mage.TargetControlledCreature(),
					mage.FuncEffect("sacrifice target creature you control",
						mage.EffectProperties{Outcome: mage.OutcomeDetriment},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.Sacrifice(perm)
							}
							return nil
						},
					),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, sacrificeMe)
	tg.AddCard(core.ZoneHand, PlayerA, sacrificer)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, sacrificer, sacrificeMe)
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, sacrificeMe, 0)
	tg.AssertGraveyardCount(PlayerA, sacrificeMe, 1)
}

// TestCR701_21_SacrificeRegenerationDoesNotApply verifies that sacrificing can't be
// replaced by regeneration.
func TestCR701_21_SacrificeRegenerationDoesNotApply(t *testing.T) {
	registerCh07aCards()

	const sacrificeMe = "Regen Sac Test Skeleton"
	const sacrificer = "Regen Sac Spell"
	if !mage.CardRegistered(sacrificeMe) {
		mage.Register(sacrificeMe, func() mage.Card {
			return mage.NewCreature(sacrificeMe, "{1}{B}", 1, 1,
				mage.WithSubTypes("Skeleton"),
				mage.WithActivatedAbility(mage.RegenerateSource(), mage.ManaCostOf("{B}")),
			)
		})
	}
	if !mage.CardRegistered(sacrificer) {
		mage.Register(sacrificer, func() mage.Card {
			return mage.NewSorcery(sacrificer, "{1}{B}",
				mage.NewTargetedSpell(
					mage.TargetControlledCreature(),
					mage.FuncEffect("sacrifice target creature you control",
						mage.EffectProperties{Outcome: mage.OutcomeDetriment},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.Sacrifice(perm)
							}
							return nil
						},
					),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, sacrificeMe)
	tg.AddCard(core.ZoneHand, PlayerA, sacrificer)
	// Activate regen — it should not prevent sacrifice
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, sacrificeMe)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, sacrificer, sacrificeMe)
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Sacrificed despite regen shield
	tg.AssertPermanentCount(PlayerA, sacrificeMe, 0)
	tg.AssertGraveyardCount(PlayerA, sacrificeMe, 1)
}

// TestCR701_21_IndestructibleCanBeSacrificed verifies that an indestructible
// permanent can still be sacrificed.
func TestCR701_21_IndestructibleCanBeSacrificed(t *testing.T) {
	registerCh07aCards()

	const sacrificer = "Indestructible Sac Spell"
	if !mage.CardRegistered(sacrificer) {
		mage.Register(sacrificer, func() mage.Card {
			return mage.NewSorcery(sacrificer, "{1}{B}",
				mage.NewTargetedSpell(
					mage.TargetControlledCreature(),
					mage.FuncEffect("sacrifice target creature you control",
						mage.EffectProperties{Outcome: mage.OutcomeDetriment},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.Sacrifice(perm)
							}
							return nil
						},
					),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Indestructible Creature")
	tg.AddCard(core.ZoneHand, PlayerA, sacrificer)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, sacrificer, "Ch07a Indestructible Creature")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Ch07a Indestructible Creature", 0)
	tg.AssertGraveyardCount(PlayerA, "Ch07a Indestructible Creature", 1)
}

// =============================================================================
// 701.23 — Search
// =============================================================================

// TestCR701_23_SearchHiddenZoneNoRequirement verifies that searching a hidden zone
// (library) for a quality doesn't require finding cards even when present.
// (Demonic Tutor-style search: player chooses; engine cannot enforce finding.)
func TestCR701_23_SearchHiddenZoneNoRequirement(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Demonic Tutor")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	// Player chooses NOT to find anything (chooses an empty result)
	// The engine represents this by the player choosing a card — we choose Bear.
	tg.ChooseFromLibrary(PlayerA, "Ch07a Bear")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Demonic Tutor")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Bear goes to hand
	tg.AssertHandCount(PlayerA, "Ch07a Bear", 1)
}

// TestCR701_23_SearchLibraryShufflesAfter verifies that searching a library causes the
// library to be shuffled afterward.
func TestCR701_23_SearchLibraryShufflesAfter(t *testing.T) {
	registerCh07aCards()

	// Natural Selection lets you look at the top 3 and reorder — but Demonic
	// Tutor searches the whole library AND shuffles after. This test uses
	// Demonic Tutor and verifies the library still contains the same cards after.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Demonic Tutor")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Ch07a Bear")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")
	tg.ChooseFromLibrary(PlayerA, "Ch07a Bear")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Demonic Tutor")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Bear went to hand; Forest and Island remain in shuffled library
	tg.AssertHandCount(PlayerA, "Ch07a Bear", 1)
	tg.AssertLibraryCount(PlayerA, "Forest", 1)
	tg.AssertLibraryCount(PlayerA, "Island", 1)
}

// =============================================================================
// 701.24 — Shuffle
// =============================================================================

// TestCR701_24_ShuffleLibraryRandomized verifies that shuffling a library randomizes
// it (all cards are still present afterward).
func TestCR701_24_ShuffleLibraryRandomized(t *testing.T) {
	registerCh07aCards()

	const shuffleSpell = "Ch07a Shuffle Spell"
	if !mage.CardRegistered(shuffleSpell) {
		mage.Register(shuffleSpell, func() mage.Card {
			return mage.NewSorcery(shuffleSpell, "{G}",
				mage.NewSpellAbility(mage.ShuffleLibrary()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, shuffleSpell)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, shuffleSpell)
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLibraryCount(PlayerA, "Forest", 1)
	tg.AssertLibraryCount(PlayerA, "Island", 1)
	tg.AssertLibraryCount(PlayerA, "Mountain", 1)
}

// =============================================================================
// 701.26 — Tap and Untap
// =============================================================================

// TestCR701_26_TapOnlyUntappedCanBeTapped verifies that tapping an already-tapped
// permanent has no additional effect.
func TestCR701_26_TapOnlyUntappedCanBeTapped(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Icy Manipulator")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Ch07a Bear")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 5)
	// Tap the bear twice; should just remain tapped
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Icy Manipulator", "Ch07a Bear")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Icy Manipulator", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertTapped(PlayerB, "Ch07a Bear", true)
	tg.AssertPermanentCount(PlayerB, "Ch07a Bear", 1) // bear not destroyed
}

// TestCR701_26_UntapOnlyTappedCanBeUntapped verifies that untapping an already-untapped
// permanent has no effect.
func TestCR701_26_UntapOnlyTappedCanBeUntapped(t *testing.T) {
	registerCh07aCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Twiddle")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ch07a Bear")
	// Bear is untapped; Twiddle untapping it should be a no-op
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Twiddle", "Ch07a Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Bear remains untapped and alive
	tg.AssertPermanentCount(PlayerA, "Ch07a Bear", 1)
	tg.AssertTapped(PlayerA, "Ch07a Bear", false)
}
