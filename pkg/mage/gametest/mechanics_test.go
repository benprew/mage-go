package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// ===== Counter Annihilation SBA (704.5q) =====

func TestCounterAnnihilation(t *testing.T) {
	name := "Counter Test Bear"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	tg := NewTestGame(t)
	id := tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	perm := tg.FindPermanent(id)
	perm.AddCounter(core.P1P1, 3)
	perm.AddCounter(core.M1M1, 2)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// 3 +1/+1 and 2 -1/-1 should annihilate to 1 +1/+1 remaining
	tg.AssertCounterCount(PlayerA, name, core.P1P1, 1)
	tg.AssertCounterCount(PlayerA, name, core.M1M1, 0)
	tg.AssertPowerToughness(PlayerA, name, 3, 3) // 2/2 + 1 from counter
}

// ===== Zero Toughness SBA =====

func TestZeroToughnessDies(t *testing.T) {
	name := "Fragile Creature"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{G}", 1, 1, mage.WithSubTypes("Insect"))
		})
	}

	tg := NewTestGame(t)
	id := tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	perm := tg.FindPermanent(id)
	perm.AddCounter(core.M1M1, 1)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// 1/1 with -1/-1 counter = 0 toughness, should be in graveyard
	tg.AssertPermanentCount(PlayerA, name, 0)
	tg.AssertGraveyardCount(PlayerA, name, 1)
}

// ===== Search Library To Hand (shuffles) =====

func TestSearchLibraryToHand(t *testing.T) {
	name := "Test Tutor"
	target1 := "Library Card A"
	target2 := "Library Card B"
	target3 := "Library Card C"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{B}",
				mage.NewSpellAbility(mage.SearchLibraryToHand()),
			)
		})
	}
	for _, n := range []string{target1, target2, target3} {

		if !mage.CardRegistered(n) {
			mage.Register(n, func() mage.Card {
				return mage.NewCreature(n, "{1}", 1, 1, mage.WithSubTypes("Test"))
			})
		}
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.AddCard(core.ZoneLibrary, PlayerA, target1)
	tg.AddCard(core.ZoneLibrary, PlayerA, target2)
	tg.AddCard(core.ZoneLibrary, PlayerA, target3)
	tg.ChooseFromLibrary(PlayerA, target2)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertHandCount(PlayerA, target2, 1)
	tg.AssertLibraryCount(PlayerA, target1, 1)
	tg.AssertLibraryCount(PlayerA, target3, 1)
}

// ===== Search Library To Top =====

func TestSearchLibraryToTop(t *testing.T) {
	name := "Test Tutor Top"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{B}",
				mage.NewSpellAbility(mage.SearchLibraryToTop()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")
	tg.ChooseFromLibrary(PlayerA, "Island")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLibraryTop(PlayerA, "Island")
	tg.AssertHandCount(PlayerA, "Island", 0)
}

// ===== Discard Cost =====

func TestDiscardCost(t *testing.T) {
	name := "Discard Spell"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{R}",
				mage.NewSpellAbility(mage.DealDamageToPlayers(mage.Fixed(3), mage.SelectEachOpponent())),
				mage.WithAdditionalCost(mage.DiscardCost(1)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.AddCard(core.ZoneHand, PlayerA, "Mountain") // card to discard
	tg.ChooseDiscard(PlayerA, "Mountain")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerB, 17) // 3 damage dealt
	tg.AssertGraveyardCount(PlayerA, "Mountain", 1)
	tg.AssertGraveyardCount(PlayerA, name, 1)
}

// ===== Equipment Falling Off Non-Creature SBA =====

func TestEquipmentStaysOnCreature(t *testing.T) {
	equipName := "Test Sword"
	crName := "Equip Target"
	if !mage.CardRegistered(equipName) {
		mage.Register(equipName, func() mage.Card {
			return mage.NewEquipment(equipName, "{2}",
				mage.WithAbility(mage.StaticAbility(mage.BoostAttached(2, 2, core.AttachEquipment))),
			)
		})
	}
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{W}", 1, 1, mage.WithSubTypes("Soldier"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, equipName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
	equip := tg.FindPermanentByName(equipName, tg.GetPlayer(PlayerA).PlayerID())
	host := tg.FindPermanentByName(crName, tg.GetPlayer(PlayerA).PlayerID())
	equip.AttachedTo = host.ID()
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertAttachedTo(PlayerA, equipName, crName)
}

// ===== Legend Rule SBA (704.5j) =====

func TestLegendRule(t *testing.T) {
	name := "Test Legend"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{W}", 3, 3,
				mage.WithSuperTypes(core.SuperLegendary),
				mage.WithSubTypes("Human", "Knight"),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, name, 1)
	tg.AssertGraveyardCount(PlayerA, name, 1)
}

// ===== World Rule SBA (704.5k) =====

func TestWorldRule(t *testing.T) {
	name1 := "Test World Enchantment A"
	name2 := "Test World Enchantment B"
	if !mage.CardRegistered(name1) {
		mage.Register(name1, func() mage.Card {
			return mage.NewEnchantment(name1, "{2}{G}",
				mage.WithSuperTypes(core.SuperWorld),
			)
		})
	}
	if !mage.CardRegistered(name2) {
		mage.Register(name2, func() mage.Card {
			return mage.NewEnchantment(name2, "{3}{R}",
				mage.WithSuperTypes(core.SuperWorld),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name1)
	tg.AddCard(core.ZoneBattlefield, PlayerB, name2)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, name1, 0)
	tg.AssertPermanentCount(PlayerB, name2, 1)
}

// ===== Cumulative Upkeep =====

func TestCumulativeUpkeep(t *testing.T) {
	name := "Cumulative Test Creature"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{U}", 4, 4,
				mage.WithSubTypes("Beast"),
				mage.WithCumulativeUpkeep("{U}"),
			)
		})
	}

	t.Run("pays upkeep with mana", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Island", 5)
		tg.StopAt(1, core.PrecombatMain)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, name, 1)
		tg.AssertCounterCount(PlayerA, name, core.Age, 1)
	})

	t.Run("sacrificed when cant pay", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.StopAt(1, core.PrecombatMain)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, name, 0)
		tg.AssertGraveyardCount(PlayerA, name, 1)
	})
}

// ===== Tap Creature Cost =====

func TestTapCreatureCost(t *testing.T) {
	name := "Tap Creature Artifact"
	helperName := "Tap Helper"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewArtifact(name, "{2}",
				mage.WithAbility(mage.NewActivatedAbility(
					mage.GainLife(2),
					mage.TapCreatureCost(),
				)),
			)
		})
	}
	if !mage.CardRegistered(helperName) {
		mage.Register(helperName, func() mage.Card {
			return mage.NewCreature(helperName, "{G}", 1, 1, mage.WithSubTypes("Elf"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.AddCard(core.ZoneBattlefield, PlayerA, helperName)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerA, 22)
	tg.AssertTapped(PlayerA, helperName, true)
}

// ===== Protection from Non-Color Attributes =====

func TestProtectionFromCreatureType(t *testing.T) {
	protName := "Protected Spirit"
	if !mage.CardRegistered(protName) {
		mage.Register(protName, func() mage.Card {
			return mage.NewCreature(protName, "{2}{W}", 2, 2,
				mage.WithSubTypes("Spirit"),
				mage.WithAbility(mage.ProtectionFromCardType(core.TypeCreature)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, protName)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, protName, 1)
	// Verify the permanent has a ProtectionAbility
	perm := tg.FindPermanentByName(protName, tg.GetPlayer(PlayerA).PlayerID())
	hasProtection := false
	for _, a := range perm.Card.Abilities() {
		if _, ok := a.(*mage.ProtectionAbility); ok {
			hasProtection = true
			break
		}
	}
	if !hasProtection {
		t.Error("expected permanent to have ProtectionAbility")
	}
}

func TestProtectionFromSubType(t *testing.T) {
	name := "Anti-Goblin Knight"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{1}{W}", 2, 2,
				mage.WithSubTypes("Human", "Knight"),
				mage.WithAbility(mage.ProtectionFromSubType("Goblin")),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, name, 1)
}

// ===== Deck-out Loss SBA (704.5b) =====

func TestDeckOutLoss(t *testing.T) {
	name := "Mass Draw"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{U}",
				mage.NewSpellAbility(mage.DrawCards(mage.Fixed(100))),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(2, core.PrecombatMain)
	tg.Execute()

	tg.AssertGameOver(true)
	tg.AssertWinner(PlayerB)
}

// ===== Target Re-validation / Fizzle =====

func TestSpellFizzlesWhenTargetGone(t *testing.T) {
	boltName := "Test Bolt"
	bearName := "Fizzle Bear"
	killName := "Test Kill"
	if !mage.CardRegistered(boltName) {
		mage.Register(boltName, func() mage.Card {
			return mage.NewInstant(boltName, "{R}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DealDamage(mage.Fixed(3))),
			)
		})
	}
	if !mage.CardRegistered(bearName) {
		mage.Register(bearName, func() mage.Card {
			return mage.NewCreature(bearName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered(killName) {
		mage.Register(killName, func() mage.Card {
			return mage.NewInstant(killName, "{B}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, bearName)
	tg.AddCard(core.ZoneHand, PlayerA, boltName)
	tg.AddCard(core.ZoneHand, PlayerA, killName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, boltName, bearName)
	tg.CastInResponseTo(PlayerA, killName, bearName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// Bear destroyed by kill spell; bolt fizzles
	tg.AssertGraveyardCount(PlayerB, bearName, 1)
	tg.AssertGraveyardCount(PlayerA, boltName, 1)
	tg.AssertGraveyardCount(PlayerA, killName, 1)
}

// ===== Exile Zone Metadata =====

func TestExileZoneMetadata(t *testing.T) {
	exilerName := "Test Exiler"
	targetName := "Exile Target"
	if !mage.CardRegistered(exilerName) {
		mage.Register(exilerName, func() mage.Card {
			return mage.NewSorcery(exilerName, "{W}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.ExileTarget()),
			)
		})
	}
	if !mage.CardRegistered(targetName) {
		mage.Register(targetName, func() mage.Card {
			return mage.NewCreature(targetName, "{G}", 1, 1, mage.WithSubTypes("Saproling"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, targetName)
	tg.AddCard(core.ZoneHand, PlayerA, exilerName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, exilerName, targetName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, targetName, 0)
	tg.AssertExileCount(targetName, 1)
}

// ===== Cost Reduction =====

func TestCostReduction(t *testing.T) {
	reducerName := "Cost Reducer"
	spellName := "Reduced Spell"
	if !mage.CardRegistered(reducerName) {
		mage.Register(reducerName, func() mage.Card {
			return mage.NewEnchantment(reducerName, "{1}{U}",
				mage.WithStaticAbility(mage.ReduceSpellCostForColor(core.Blue, 1)),
			)
		})
	}
	if !mage.CardRegistered(spellName) {
		mage.Register(spellName, func() mage.Card {
			return mage.NewSorcery(spellName, "{2}{U}",
				mage.NewSpellAbility(mage.GainLife(5)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, reducerName)
	tg.AddCard(core.ZoneHand, PlayerA, spellName)
	// With reduction: {2}{U} -> {1}{U} — need 1 generic + 1 blue = 2 total
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Island", 2)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, spellName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerA, 25) // gained 5 life
	tg.AssertGraveyardCount(PlayerA, spellName, 1)
}

// ===== Put From Hand Onto Battlefield =====

func TestPutFromHandOntoBattlefield(t *testing.T) {
	name := "Test Cheat Spell"
	bigCreature := "Cheated Creature"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{G}",
				mage.NewSpellAbility(mage.PutFromHandOntoBattlefield(mage.CardFilter{})),
			)
		})
	}
	if !mage.CardRegistered(bigCreature) {
		mage.Register(bigCreature, func() mage.Card {
			return mage.NewCreature(bigCreature, "{8}{G}{G}", 10, 10, mage.WithSubTypes("Beast"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.AddCard(core.ZoneHand, PlayerA, bigCreature)
	tg.ChooseFromLibrary(PlayerA, bigCreature) // reuses library chooser for "put onto battlefield" card choice
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, bigCreature, 1)
	tg.AssertPowerToughness(PlayerA, bigCreature, 10, 10)
}

// ===== Choose Color Effect =====

func TestChooseColor(t *testing.T) {
	name := "Color Chooser"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewEnchantment(name, "{1}{W}",
				mage.WithAbility(mage.EntersBattlefieldTrigger(
					mage.ChooseColor("choose a color"), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.ChooseManaColor(PlayerA, core.Red)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, name, 1)
	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if perm.ChosenColor != core.Red {
		t.Errorf("ChosenColor: got %v, want Red", perm.ChosenColor)
	}
}

// ===== SuperTypes (HasSuperType) =====

func TestHasSuperType(t *testing.T) {
	name := "Legendary Beast"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{3}{G}", 4, 4,
				mage.WithSuperTypes(core.SuperLegendary),
				mage.WithSubTypes("Beast"),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if !perm.Card.HasSuperType(core.SuperLegendary) {
		t.Error("expected HasSuperType(Legendary) to be true")
	}
	if perm.Card.HasSuperType(core.SuperBasic) {
		t.Error("expected HasSuperType(Basic) to be false")
	}
}

// ===== Additional Costs at Cast Time =====

func TestAdditionalCostLifePay(t *testing.T) {
	name := "Life Pay Spell"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{B}",
				mage.NewSpellAbility(mage.DrawCards(mage.Fixed(3))),
				mage.WithAdditionalCost(mage.LifePayCost(3)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerA, 17) // paid 3 life
	tg.AssertGraveyardCount(PlayerA, name, 1)
}

// ===== Shuffle Library Effect =====

func TestShuffleLibraryEffect(t *testing.T) {
	name := "Shuffle Spell"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{G}",
				mage.NewSpellAbility(mage.ShuffleLibrary()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// Library should still contain the same 3 cards (just shuffled)
	tg.AssertLibraryCount(PlayerA, "Mountain", 1)
	tg.AssertLibraryCount(PlayerA, "Forest", 1)
	tg.AssertLibraryCount(PlayerA, "Island", 1)
}

// ===== Targeted Spells Not Castable Without Targets =====

func TestTargetedSpellNotCastableWithoutTargets(t *testing.T) {
	auraName := "Test Aura No Targets"
	crName := "Test Aura Bear"
	if !mage.CardRegistered(auraName) {
		mage.Register(auraName, func() mage.Card {
			return mage.NewBoostAura(auraName, "{0}", 1, 1)
		})
	}
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{0}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	t.Run("aura not castable with no creatures", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, auraName)
		tg.StopAt(1, core.PrecombatMain)
		tg.Execute()

		playerID := tg.GetPlayer(PlayerA).PlayerID()
		castable := tg.GetCastableSpells(playerID)
		for _, card := range castable {
			if card.Name() == auraName {
				t.Errorf("aura should not be castable with no creatures on the battlefield")
			}
		}
	})

	t.Run("aura castable with creature on battlefield", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, auraName)
		tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
		tg.StopAt(1, core.PrecombatMain)
		tg.Execute()

		playerID := tg.GetPlayer(PlayerA).PlayerID()
		castable := tg.GetCastableSpells(playerID)
		found := false
		for _, card := range castable {
			if card.Name() == auraName {
				found = true
			}
		}
		if !found {
			t.Errorf("aura should be castable when a creature is on the battlefield")
		}
	})
}

// ===== Attachment cleanup on host removal =====

func TestAttachmentsOnSacrifice(t *testing.T) {
	auraName := "Sac Test Aura"
	equipName := "Sac Test Sword"
	crName := "Sac Test Bear"
	if !mage.CardRegistered(auraName) {
		mage.Register(auraName, func() mage.Card {
			return mage.NewAura(auraName, "{W}")
		})
	}
	if !mage.CardRegistered(equipName) {
		mage.Register(equipName, func() mage.Card {
			return mage.NewEquipment(equipName, "{2}",
				mage.WithAbility(mage.StaticAbility(mage.BoostAttached(1, 1, core.AttachEquipment))),
			)
		})
	}
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	t.Run("aura goes to graveyard when host is sacrificed", func(t *testing.T) {
		tg := NewTestGame(t)
		crID := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
		auraID := tg.AddCard(core.ZoneBattlefield, PlayerA, auraName)
		tg.Attach(auraID, crID)

		perm := tg.FindPermanent(crID)
		tg.DoSacrifice(perm)
		tg.CheckStateBasedActions()

		tg.AssertPermanentCount(PlayerA, crName, 0)
		tg.AssertPermanentCount(PlayerA, auraName, 0)
		tg.AssertGraveyardCount(PlayerA, auraName, 1)
	})

	t.Run("equipment detaches but stays on battlefield when host is sacrificed", func(t *testing.T) {
		tg := NewTestGame(t)
		crID := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
		equipID := tg.AddCard(core.ZoneBattlefield, PlayerA, equipName)
		tg.Attach(equipID, crID)

		perm := tg.FindPermanent(crID)
		tg.DoSacrifice(perm)
		tg.CheckStateBasedActions()

		tg.AssertPermanentCount(PlayerA, crName, 0)
		tg.AssertPermanentCount(PlayerA, equipName, 1)
		equip := tg.FindPermanentByName(equipName, tg.GetPlayer(PlayerA).PlayerID())
		if equip.IsAttached() {
			t.Errorf("equipment should be detached after host is sacrificed")
		}
	})
}

func TestAttachmentsOnExile(t *testing.T) {
	auraName := "Exile Test Aura"
	equipName := "Exile Test Sword"
	crName := "Exile Test Bear"
	if !mage.CardRegistered(auraName) {
		mage.Register(auraName, func() mage.Card {
			return mage.NewAura(auraName, "{W}")
		})
	}
	if !mage.CardRegistered(equipName) {
		mage.Register(equipName, func() mage.Card {
			return mage.NewEquipment(equipName, "{2}",
				mage.WithAbility(mage.StaticAbility(mage.BoostAttached(1, 1, core.AttachEquipment))),
			)
		})
	}
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	t.Run("aura goes to graveyard when host is exiled", func(t *testing.T) {
		tg := NewTestGame(t)
		crID := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
		auraID := tg.AddCard(core.ZoneBattlefield, PlayerA, auraName)
		tg.Attach(auraID, crID)

		perm := tg.FindPermanent(crID)
		tg.ExilePermanent(perm)
		tg.CheckStateBasedActions()

		tg.AssertPermanentCount(PlayerA, crName, 0)
		tg.AssertPermanentCount(PlayerA, auraName, 0)
		tg.AssertGraveyardCount(PlayerA, auraName, 1)
		tg.AssertExileCount(crName, 1)
	})

	t.Run("equipment detaches but stays on battlefield when host is exiled", func(t *testing.T) {
		tg := NewTestGame(t)
		crID := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
		equipID := tg.AddCard(core.ZoneBattlefield, PlayerA, equipName)
		tg.Attach(equipID, crID)

		perm := tg.FindPermanent(crID)
		tg.ExilePermanent(perm)
		tg.CheckStateBasedActions()

		tg.AssertPermanentCount(PlayerA, crName, 0)
		tg.AssertPermanentCount(PlayerA, equipName, 1)
		equip := tg.FindPermanentByName(equipName, tg.GetPlayer(PlayerA).PlayerID())
		if equip.IsAttached() {
			t.Errorf("equipment should be detached after host is exiled")
		}
		tg.AssertExileCount(crName, 1)
	})
}
