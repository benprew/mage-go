package mage

import (
	"testing"

	"github.com/google/uuid"
)

// TestGenericTriggered verifies that GenericTriggered can replace all
// bespoke triggered ability types with a single parameterized struct.
func TestGenericTriggered(t *testing.T) {
	t.Run("ETB trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL ETB Creature"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewCreature(name, "{2}{G}", 3, 3, "Beast")
				c.AddAbility(NewTriggered(
					EvtEntersBattlefield,
					false,
					GainLife(3),
				).SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == sourceID
				}))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerA, 23) // 20 + 3 from ETB trigger
		tg.AssertPermanentCount(PlayerA, name, 1)
	})

	t.Run("upkeep trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL Upkeep Damager"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewArtifact(name, "{2}")
				c.AddAbility(NewTriggered(
					EvtUpkeep,
					false,
					DealDamageToPlayers(Fixed(1), SelectController()),
				).SetCondition(func(evt *GameEvent, _ *Game, _, controllerID uuid.UUID) bool {
					return evt.PlayerID == controllerID
				}))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.StopAt(2, PrecombatMain)
		tg.Execute()

		tg.AssertLife(PlayerA, 19) // took 1 damage on turn 1 upkeep
	})

	t.Run("creature dies trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL Death Counter"
		victimName := "DSL Victim"
		boltName := "DSL Test Bolt"
		for _, reg := range []struct {
			n string
			f func() Card
		}{
			{name, func() Card {
				c := NewCreature(name, "{1}{B}", 1, 1, "Zombie")
				c.AddAbility(NewTriggered(
					EvtCreatureDied,
					false,
					AddCounters(P1P1, Fixed(1), SelectSource),
				).SetCondition(func(evt *GameEvent, _ *Game, sourceID, controllerID uuid.UUID) bool {
					if evt.SourceID == sourceID {
						return false
					}
					return evt.PlayerID == controllerID
				}))
				return c
			}},
			{victimName, func() Card {
				c := NewCreature(victimName, "{1}{G}", 2, 2, "Bear")
				return c
			}},
			{boltName, func() Card {
				c := NewInstant(boltName, "{R}")
				c.AddAbility(NewTargetedSpell(TargetCreature(), DealDamage(Fixed(3))))
				return c
			}},
		} {
			if !CardRegistered(reg.n) {
				Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.AddCard(ZoneBattlefield, PlayerA, victimName)
		tg.AddCard(ZoneHand, PlayerA, boltName)
		tg.CastSpell(1, PrecombatMain, PlayerA, boltName, victimName)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertCounterCount(PlayerA, name, P1P1, 1)
	})

	t.Run("attacks trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL Attack Pumper"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewCreature(name, "{1}{R}", 2, 2, "Warrior")
				c.AddAbility(NewTriggered(
					EvtDeclaredAttacker,
					false,
					BoostUntilEndOfTurn(Fixed(2), Fixed(0), SelectSource),
				).SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == sourceID
				}))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerB, 16) // 2+2 = 4 damage
	})
}

// TestDestroyAllCollapse verifies that DestroyAllCreatures, DestroyAllLands,
// and DestroyAllEnchantments are thin aliases over DestroyAllMatching.
func TestDestroyAllCollapse(t *testing.T) {
	t.Run("DestroyAllCreatures still works", func(t *testing.T) {
		name := "DSL Wrath"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{2}{W}{W}")
				c.AddAbility(NewSpellAbility(DestroyAllCreatures()))
				return c
			})
		}
		creatureName := "DSL Bear"
		if !CardRegistered(creatureName) {
			Register(creatureName, func() Card {
				c := NewCreature(creatureName, "{1}{G}", 2, 2, "Bear")
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, creatureName)
		tg.AddCard(ZoneBattlefield, PlayerB, creatureName)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, creatureName, 0)
		tg.AssertPermanentCount(PlayerB, creatureName, 0)
	})

	t.Run("DestroyAllLands still works", func(t *testing.T) {
		name := "DSL Armageddon"
		landName := "DSL Test Land"
		for _, reg := range []struct {
			n string
			f func() Card
		}{
			{name, func() Card {
				c := NewSorcery(name, "{3}{W}")
				c.AddAbility(NewSpellAbility(DestroyAllLands()))
				return c
			}},
			{landName, func() Card {
				c := NewLand(landName, "Forest")
				c.AddAbility(NewManaAbility(Green))
				return c
			}},
		} {
			if !CardRegistered(reg.n) {
				Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, landName)
		tg.AddCard(ZoneBattlefield, PlayerB, landName)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, landName, 0)
		tg.AssertPermanentCount(PlayerB, landName, 0)
	})

	t.Run("DestroyAllEnchantments still works", func(t *testing.T) {
		name := "DSL Tranquility"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{2}{G}")
				c.AddAbility(NewSpellAbility(DestroyAllEnchantments()))
				return c
			})
		}
		enchName := "DSL Test Enchantment"
		if !CardRegistered(enchName) {
			Register(enchName, func() Card {
				c := NewEnchantment(enchName, "{W}")
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, enchName)
		tg.AddCard(ZoneBattlefield, PlayerB, enchName)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, enchName, 0)
		tg.AssertPermanentCount(PlayerB, enchName, 0)
	})
}

// TestXVariantCollapse verifies that X-variant effects still work after
// being collapsed into the base effect types (e.g., DealXDamage → DealDamage with useX).
func TestXVariantCollapse(t *testing.T) {
	t.Run("DealXDamage still works", func(t *testing.T) {
		name := "DSL X Damage"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{X}{R}")
				c.AddAbility(NewTargetedSpell(TargetAnyTarget(), DealDamage(XValue())))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpellWithX(1, PrecombatMain, PlayerA, name, 5, "PlayerB")
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerB, 15) // 20 - 5 = 15
	})

	t.Run("DealDamage fixed amount still works", func(t *testing.T) {
		eff := DealDamage(Fixed(3))
		if eff.Text() != "deal 3 damage to target" {
			t.Errorf("DealDamage(Fixed(3)).Text() = %q", eff.Text())
		}
	})

	t.Run("DrawXCards still works", func(t *testing.T) {
		name := "DSL X Draw"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{X}{U}")
				c.AddAbility(NewTargetedSpell(TargetPlayer(), DrawCards(XValue())))
				return c
			})
		}

		tg := NewTestGame(t)
		// Put cards in library
		for i := 0; i < 5; i++ {
			tg.getPlayer(PlayerA).AddToLibrary(NewSorcery("Library Card", "{0}"))
		}
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpellWithX(1, PrecombatMain, PlayerA, name, 3, "PlayerA")
		tg.StopAt(1, EndCombat)
		tg.Execute()

		// Started with 0 hand cards (name was cast), drew 3
		if len(tg.getPlayer(PlayerA).Hand()) != 3 {
			t.Errorf("expected 3 cards in hand, got %d", len(tg.getPlayer(PlayerA).Hand()))
		}
	})
}

// TestFuncEffect verifies that anonymous functions can be used as effects.
func TestFuncEffect(t *testing.T) {
	t.Run("inline effect via FuncEffect", func(t *testing.T) {
		name := "DSL Func Spell"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{R}")
				sa := NewSpellAbility(FuncEffect(
					"deal 3 damage to each player",
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						for _, p := range g.Players {
							g.DealDamageToPlayer(p, 3, sourceID)
						}
						return nil
					},
				))
				c.AddAbility(sa)
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerA, 17)
		tg.AssertLife(PlayerB, 17)
	})

	t.Run("FuncEffect Text() returns description", func(t *testing.T) {
		eff := FuncEffect("do something cool", func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			return nil
		})
		if eff.Text() != "do something cool" {
			t.Errorf("FuncEffect.Text() = %q, want %q", eff.Text(), "do something cool")
		}
	})
}

// TestSourceTargetUnification verifies that Source/Target effect pairs have been
// unified into single parameterized structs that determine their target via
// an applyToSource flag.
func TestSourceTargetUnification(t *testing.T) {
	t.Run("BoostSourceUntilEndOfTurn pumps source", func(t *testing.T) {
		name := "DSL Boost Source"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewCreature(name, "{1}{R}", 2, 2, "Warrior")
				c.AddAbility(NewTriggered(
					EvtDeclaredAttacker,
					false,
					BoostUntilEndOfTurn(Fixed(1), Fixed(0), SelectSource),
				).SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == sourceID
				}))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerB, 17) // 2+1 = 3 damage
	})

	t.Run("BoostTargetUntilEndOfTurn pumps target", func(t *testing.T) {
		eff := BoostUntilEndOfTurn(Fixed(2), Fixed(2), SelectTarget)
		if eff.Text() != "target creature gets +2/+2 until end of turn" {
			t.Errorf("BoostTargetUntilEndOfTurn(2,2).Text() = %q", eff.Text())
		}
	})

	t.Run("AddCountersToSource adds to source", func(t *testing.T) {
		eff := AddCounters(P1P1, Fixed(2), SelectSource)
		if eff.Text() != "put 2 +1/+1 counter(s) on it" {
			t.Errorf("AddCountersToSource.Text() = %q", eff.Text())
		}
	})

	t.Run("AddCountersToTarget adds to target", func(t *testing.T) {
		eff := AddCounters(P1P1, Fixed(1), SelectTarget)
		if eff.Text() != "put 1 +1/+1 counter(s) on target" {
			t.Errorf("AddCountersToTarget.Text() = %q", eff.Text())
		}
	})

	t.Run("GrantKeywordSourceUntilEndOfTurn Text()", func(t *testing.T) {
		eff := GrantKeywordUntilEndOfTurn(Flying, SelectSource)
		if eff.Text() != "~ gains Flying until end of turn" {
			t.Errorf("GrantKeywordSource.Text() = %q", eff.Text())
		}
	})

	t.Run("GrantKeywordTargetUntilEndOfTurn Text()", func(t *testing.T) {
		eff := GrantKeywordUntilEndOfTurn(Flying, SelectTarget)
		if eff.Text() != "target creature gains Flying until end of turn" {
			t.Errorf("GrantKeywordTarget.Text() = %q", eff.Text())
		}
	})
}

// TestContinuousEffectMerge verifies that boostAllCreaturesEffect and
// boostAllCreaturesIncludingSelfEffect have been merged into a single
// struct with an includeSelf flag.
func TestContinuousEffectMerge(t *testing.T) {
	t.Run("BoostAllCreatures excludes self (lord)", func(t *testing.T) {
		lordName := "DSL Lord"
		bearName := "DSL Bear CE"
		for _, reg := range []struct {
			n string
			f func() Card
		}{
			{lordName, func() Card {
				c := NewCreature(lordName, "{2}{W}", 1, 1, "Human")
				c.AddAbility(StaticAbility(BoostAllCreatures(1, 1, nil)))
				return c
			}},
			{bearName, func() Card {
				c := NewCreature(bearName, "{1}{G}", 2, 2, "Bear")
				return c
			}},
		} {
			if !CardRegistered(reg.n) {
				Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, lordName)
		tg.AddCard(ZoneBattlefield, PlayerA, bearName)
		tg.Attack(1, PlayerA, bearName)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		// Bear gets +1/+1 from lord = 3/3, deals 3 damage
		tg.AssertLife(PlayerB, 17)
	})

	t.Run("BoostAllCreaturesIncludingSelf includes self", func(t *testing.T) {
		selfBoosterName := "DSL Self Booster"
		if !CardRegistered(selfBoosterName) {
			Register(selfBoosterName, func() Card {
				c := NewCreature(selfBoosterName, "{2}{G}", 1, 1, "Beast")
				c.AddAbility(StaticAbility(BoostAllCreaturesIncludingSelf(2, 2, nil)))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, selfBoosterName)
		tg.Attack(1, PlayerA, selfBoosterName)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		// Self Booster gets +2/+2 from itself = 3/3, deals 3 damage
		tg.AssertLife(PlayerB, 17)
	})
}

// TestCardTemplateHelpers verifies that card template helpers produce
// correctly configured cards with minimal boilerplate.
func TestCardTemplateHelpers(t *testing.T) {
	t.Run("NewLuckyCharm creates color-triggered life gain artifact", func(t *testing.T) {
		name := "DSL Lucky Charm"
		if !CardRegistered(name) {
			Register(name, func() Card {
				return NewLuckyCharm(name, "{1}", Blue)
			})
		}

		spellName := "DSL Blue Spell"
		if !CardRegistered(spellName) {
			Register(spellName, func() Card {
				return NewSorcery(spellName, "{U}")
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.AddCard(ZoneHand, PlayerA, spellName)
		tg.CastSpell(1, PrecombatMain, PlayerA, spellName)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerA, 21) // 20 + 1 from lucky charm trigger
	})

	t.Run("NewLandDestruction creates land destruction sorcery", func(t *testing.T) {
		name := "DSL Land Destroy"
		landName := "DSL Target Land"
		for _, reg := range []struct {
			n string
			f func() Card
		}{
			{name, func() Card {
				return NewLandDestruction(name, "{2}{R}")
			}},
			{landName, func() Card {
				c := NewLand(landName, "Mountain")
				c.AddAbility(NewManaAbility(Red))
				return c
			}},
		} {
			if !CardRegistered(reg.n) {
				Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerB, landName)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name, landName)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertPermanentCount(PlayerB, landName, 0)
	})

	t.Run("NewBoostAura creates aura that boosts attached creature", func(t *testing.T) {
		auraName := "DSL Boost Aura"
		bearName := "DSL Aura Bear"
		for _, reg := range []struct {
			n string
			f func() Card
		}{
			{auraName, func() Card {
				return NewBoostAura(auraName, "{G}", 2, 2)
			}},
			{bearName, func() Card {
				c := NewCreature(bearName, "{1}{G}", 2, 2, "Bear")
				return c
			}},
		} {
			if !CardRegistered(reg.n) {
				Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, bearName)
		tg.AddCard(ZoneHand, PlayerA, auraName)
		tg.CastSpell(1, PrecombatMain, PlayerA, auraName, bearName)
		tg.Attack(1, PlayerA, bearName)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		// Bear 2/2 + aura +2/+2 = 4/4, deals 4 damage
		tg.AssertLife(PlayerB, 16)
	})
}

// TestPermanentPropertyMigration verifies that permanent property flags have
// been successfully migrated from ad-hoc BaseCard fields to the ability system.
func TestPermanentPropertyMigration(t *testing.T) {
	t.Run("CantBeBlockedByWalls keyword on Juggernaut", func(t *testing.T) {
		name := "DSL CantBlock Wall Test"
		wallName := "DSL Wall Blocker"
		for _, reg := range []struct {
			n string
			f func() Card
		}{
			{name, func() Card {
				c := NewCreature(name, "{4}", 5, 3, "Juggernaut")
				c.AddType(TypeArtifact)
				c.AddAbility(NewKeywordAbility(CantBeBlockedByWalls))
				return c
			}},
			{wallName, func() Card {
				c := NewCreature(wallName, "{2}{W}", 0, 5, "Wall")
				c.AddAbility(NewKeywordAbility(Defender))
				return c
			}},
		} {
			if !CardRegistered(reg.n) {
				Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.AddCard(ZoneBattlefield, PlayerB, wallName)
		tg.Attack(1, PlayerA, name)
		tg.Block(1, PlayerB, wallName, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		// Wall can't block Juggernaut, so 5 damage goes through
		tg.AssertLife(PlayerB, 15)
	})

	t.Run("EntersTapped keyword", func(t *testing.T) {
		name := "DSL Enters Tapped"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewArtifact(name, "{4}")
				c.AddAbility(NewKeywordAbility(EntersTapped))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertTapped(PlayerA, name, true)
	})

	t.Run("DoesNotUntapKW keyword", func(t *testing.T) {
		name := "DSL Does Not Untap"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewArtifact(name, "{3}")
				c.AddAbility(NewKeywordAbility(DoesNotUntapKW))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.StopAt(1, PrecombatMain)
		tg.Execute()

		tg.AssertHasAbility(PlayerA, name, DoesNotUntapKW, true)
	})

	t.Run("EntersWithXCounters ability", func(t *testing.T) {
		name := "DSL X Counter Creature"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewCreature(name, "{X}{R}{R}", 0, 0, "Hydra")
				c.AddAbility(EntersWithXCounters(P1P1))
				return c
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpellWithX(1, PrecombatMain, PlayerA, name, 3)
		tg.StopAt(1, EndCombat)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, name, 1)
		tg.AssertCounterCount(PlayerA, name, P1P1, 3)
	})

	t.Run("SacrificeUnlessLand ability", func(t *testing.T) {
		name := "DSL Sacrifice Unless Island"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewCreature(name, "{5}{U}", 5, 5, "Serpent")
				c.AddAbility(SacrificeUnlessLand("Island"))
				return c
			})
		}

		// Without an Island, the creature should be sacrificed
		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, name)
		tg.StopAt(1, PrecombatMain)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, name, 0)
	})

	t.Run("GraveyardReturnIfCreaturesAbove ability", func(t *testing.T) {
		name := "DSL Graveyard Return"
		bearName := "DSL GY Bear"
		for _, reg := range []struct {
			n string
			f func() Card
		}{
			{name, func() Card {
				c := NewCreature(name, "{B}{B}", 1, 1, "Spirit")
				c.AddAbility(NewKeywordAbility(Haste))
				c.AddAbility(GraveyardReturnIfCreaturesAbove(2))
				return c
			}},
			{bearName, func() Card {
				c := NewCreature(bearName, "{1}{G}", 2, 2, "Bear")
				return c
			}},
		} {
			if !CardRegistered(reg.n) {
				Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		// Put the returner in graveyard, then 2 creatures above it
		tg.AddCard(ZoneGraveyard, PlayerA, name)
		tg.AddCard(ZoneGraveyard, PlayerA, bearName)
		tg.AddCard(ZoneGraveyard, PlayerA, bearName)
		tg.StopAt(2, PrecombatMain) // upkeep of turn 2 checks graveyard
		tg.Execute()

		// Should have returned to battlefield
		tg.AssertPermanentCount(PlayerA, name, 1)
	})
}

// TestValueSource verifies Fixed and XValue resolve correctly.
func TestValueSource(t *testing.T) {
	t.Run("Fixed returns constant", func(t *testing.T) {
		v := Fixed(7)
		got := v.Resolve(nil, uuid.Nil, uuid.Nil)
		if got != 7 {
			t.Errorf("Fixed(7).Resolve() = %d, want 7", got)
		}
	})

	t.Run("Fixed text shows number", func(t *testing.T) {
		v := Fixed(3)
		if v.Text() != "3" {
			t.Errorf("Fixed(3).Text() = %q, want %q", v.Text(), "3")
		}
	})

	t.Run("XValue reads CurrentX", func(t *testing.T) {
		g := &Game{CurrentX: 5}
		v := XValue()
		got := v.Resolve(g, uuid.Nil, uuid.Nil)
		if got != 5 {
			t.Errorf("XValue().Resolve() with CurrentX=5 = %d, want 5", got)
		}
	})

	t.Run("XValue text is X", func(t *testing.T) {
		v := XValue()
		if v.Text() != "X" {
			t.Errorf("XValue().Text() = %q, want %q", v.Text(), "X")
		}
	})
}

// TestPlayerSelector verifies all PlayerSelector implementations.
func TestPlayerSelector(t *testing.T) {
	t.Run("SelectController returns controller", func(t *testing.T) {
		controllerID := uuid.New()
		s := SelectController()
		ids := s.Select(nil, uuid.Nil, controllerID, nil)
		if len(ids) != 1 || ids[0] != controllerID {
			t.Errorf("SelectController().Select() = %v, want [%v]", ids, controllerID)
		}
	})

	t.Run("SelectActivePlayer returns active player", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.StopAt(1, PrecombatMain)
		tg.Execute()
		s := SelectActivePlayer()
		ids := s.Select(tg.Game, uuid.Nil, uuid.Nil, nil)
		if len(ids) != 1 {
			t.Fatalf("SelectActivePlayer returned %d IDs, want 1", len(ids))
		}
		activeID := tg.Game.ActivePlayerObj().PlayerID()
		if ids[0] != activeID {
			t.Errorf("SelectActivePlayer returned %v, want %v", ids[0], activeID)
		}
	})

	t.Run("SelectEachPlayer returns all players", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.StopAt(1, PrecombatMain)
		tg.Execute()
		s := SelectEachPlayer()
		ids := s.Select(tg.Game, uuid.Nil, uuid.Nil, nil)
		if len(ids) != 2 {
			t.Errorf("SelectEachPlayer returned %d IDs, want 2", len(ids))
		}
	})

	t.Run("SelectEachOpponent excludes controller", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.StopAt(1, PrecombatMain)
		tg.Execute()
		controllerID := tg.Game.Players[0].PlayerID()
		s := SelectEachOpponent()
		ids := s.Select(tg.Game, uuid.Nil, controllerID, nil)
		if len(ids) != 1 {
			t.Fatalf("SelectEachOpponent returned %d IDs, want 1", len(ids))
		}
		if ids[0] == controllerID {
			t.Errorf("SelectEachOpponent should not return the controller")
		}
	})

	t.Run("SelectEventController reads targets[0]", func(t *testing.T) {
		playerID := uuid.New()
		s := SelectEventController()
		ids := s.Select(nil, uuid.Nil, uuid.Nil, []uuid.UUID{playerID})
		if len(ids) != 1 || ids[0] != playerID {
			t.Errorf("SelectEventController() = %v, want [%v]", ids, playerID)
		}
	})

	t.Run("SelectEventController with no targets returns nil", func(t *testing.T) {
		s := SelectEventController()
		ids := s.Select(nil, uuid.Nil, uuid.Nil, nil)
		if len(ids) != 0 {
			t.Errorf("SelectEventController() with no targets = %v, want empty", ids)
		}
	})
}

// TestPermanentSelector verifies SelectTarget vs SelectSource constants.
func TestPermanentSelector(t *testing.T) {
	if SelectTarget == SelectSource {
		t.Error("SelectTarget and SelectSource should be different values")
	}
}

// TestDealDamageToPlayers verifies the unified damage-to-players effect
// works with different selectors.
func TestDealDamageToPlayers(t *testing.T) {
	t.Run("damage to each player", func(t *testing.T) {
		name := "DTP Each Player Test"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{R}")
				c.AddAbility(NewSpellAbility(DealDamageToPlayers(Fixed(3), SelectEachPlayer())))
				return c
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerA, 17)
		tg.AssertLife(PlayerB, 17)
	})

	t.Run("damage to each opponent", func(t *testing.T) {
		name := "DTP Each Opponent Test"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{R}")
				c.AddAbility(NewSpellAbility(DealDamageToPlayers(Fixed(4), SelectEachOpponent())))
				return c
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name)
		tg.StopAt(1, EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerA, 20)
		tg.AssertLife(PlayerB, 16)
	})
}

// TestValueSourceEffects verifies that effects using ValueSource resolve
// both Fixed and XValue correctly.
func TestValueSourceEffects(t *testing.T) {
	t.Run("DealDamage with Fixed", func(t *testing.T) {
		name := "VS DealDamage Fixed"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewInstant(name, "{R}")
				sa := NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(3)))
				c.AddAbility(sa)
				return c
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name, "PlayerB")
		tg.StopAt(1, EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 17)
	})

	t.Run("GainLifeTarget with Fixed", func(t *testing.T) {
		name := "VS GainLife Fixed"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewInstant(name, "{W}")
				sa := NewTargetedSpell(TargetPlayer(), GainLifeTarget(Fixed(5)))
				c.AddAbility(sa)
				return c
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name, "PlayerA")
		tg.StopAt(1, EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerA, 25)
	})

	t.Run("BoostTarget with Fixed", func(t *testing.T) {
		name := "VS Boost Fixed"
		bearName := "Boost Test Bear"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewInstant(name, "{G}")
				sa := NewTargetedSpell(TargetCreature(), BoostUntilEndOfTurn(Fixed(3), Fixed(3), SelectTarget))
				c.AddAbility(sa)
				return c
			})
			Register(bearName, func() Card {
				c := NewCreature(bearName, "{1}{G}", 2, 2, "Bear")
				return c
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(ZoneBattlefield, PlayerA, bearName)
		tg.AddCard(ZoneHand, PlayerA, name)
		tg.CastSpell(1, PrecombatMain, PlayerA, name, bearName)
		tg.StopAt(1, EndCombat)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, bearName, 5, 5)
	})
}
