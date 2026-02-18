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
				c := NewCreature(name, "{2}{G}", "Beast")
				c.Power_ = 3
				c.Toughness_ = 3
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
				c := NewCreature(name, "{1}{B}", "Zombie")
				c.Power_ = 1
				c.Toughness_ = 1
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
				c := NewCreature(victimName, "{1}{G}", "Bear")
				c.Power_ = 2
				c.Toughness_ = 2
				return c
			}},
			{boltName, func() Card {
				c := NewInstant(boltName, "{R}")
				sa := NewSpellAbility(DealDamage(3))
				sa.AddTarget(TargetCreature())
				c.AddAbility(sa)
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
				c := NewCreature(name, "{1}{R}", "Warrior")
				c.Power_ = 2
				c.Toughness_ = 2
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
