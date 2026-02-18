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
					DealDamageToSourceController(1),
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
					AddCountersToSource(P1P1, 1),
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
				c.AddAbility(NewTargetedSpell(TargetCreature(), DealDamage(3)))
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
					BoostSourceUntilEndOfTurn(2, 0),
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
				c.AddAbility(NewTargetedSpell(TargetAnyTarget(), DealXDamage()))
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
		eff := DealDamage(3)
		if eff.Text() != "deal 3 damage to target" {
			t.Errorf("DealDamage(3).Text() = %q", eff.Text())
		}
	})

	t.Run("DrawXCards still works", func(t *testing.T) {
		name := "DSL X Draw"
		if !CardRegistered(name) {
			Register(name, func() Card {
				c := NewSorcery(name, "{X}{U}")
				c.AddAbility(NewTargetedSpell(TargetPlayer(), DrawXCards()))
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
					BoostSourceUntilEndOfTurn(1, 0),
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
		eff := BoostTargetUntilEndOfTurn(2, 2)
		if eff.Text() != "target creature gets +2/+2 until end of turn" {
			t.Errorf("BoostTargetUntilEndOfTurn(2,2).Text() = %q", eff.Text())
		}
	})

	t.Run("AddCountersToSource adds to source", func(t *testing.T) {
		eff := AddCountersToSource(P1P1, 2)
		if eff.Text() != "put 2 +1/+1 counter(s) on it" {
			t.Errorf("AddCountersToSource.Text() = %q", eff.Text())
		}
	})

	t.Run("AddCountersToTarget adds to target", func(t *testing.T) {
		eff := AddCountersToTarget(P1P1, 1)
		if eff.Text() != "put 1 +1/+1 counter(s) on target" {
			t.Errorf("AddCountersToTarget.Text() = %q", eff.Text())
		}
	})

	t.Run("GrantKeywordSourceUntilEndOfTurn Text()", func(t *testing.T) {
		eff := GrantKeywordSourceUntilEndOfTurn(Flying)
		if eff.Text() != "~ gains Flying until end of turn" {
			t.Errorf("GrantKeywordSource.Text() = %q", eff.Text())
		}
	})

	t.Run("GrantKeywordTargetUntilEndOfTurn Text()", func(t *testing.T) {
		eff := GrantKeywordTargetUntilEndOfTurn(Flying)
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
