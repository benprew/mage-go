package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGenericTriggered(t *testing.T) {
	t.Run("ETB trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL ETB Creature"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{2}{G}", 3, 3,
					mage.WithSubTypes("Beast"),
					mage.WithAbility(mage.EntersBattlefieldTrigger(mage.GainLife(3), false)),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerA, 23)
		tg.AssertPermanentCount(PlayerA, name, 1)
	})

	t.Run("upkeep trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL Upkeep Damager"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewArtifact(name, "{2}",
					mage.WithAbility(mage.NewTriggered(
						core.EvtUpkeep,
						false,
						mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController()),
					).SetConditionData(mage.EventPlayerIsController{})),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.StopAt(2, core.PrecombatMain)
		tg.Execute()

		tg.AssertLife(PlayerA, 19)
	})

	t.Run("creature dies trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL Death Counter"
		victimName := "DSL Victim"
		boltName := "DSL Test Bolt"
		for _, reg := range []struct {
			n string
			f func() mage.Card
		}{
			{name, func() mage.Card {
				return mage.NewCreature(name, "{1}{B}", 1, 1,
					mage.WithSubTypes("Zombie"),
					mage.WithAbility(mage.DiesCreatureTrigger(
						mage.AddCounters(core.P1P1, mage.Fixed(1)).Targeting(mage.ToSource()),
						false, mage.AnyPermanent,
					)),
				)
			}},
			{victimName, func() mage.Card {
				return mage.NewCreature(victimName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			}},
			{boltName, func() mage.Card {
				return mage.NewInstant(boltName, "{R}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.DealDamage(mage.Fixed(3))),
				)
			}},
		} {
			if !mage.CardRegistered(reg.n) {
				mage.Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, victimName)
		tg.AddCard(core.ZoneHand, PlayerA, boltName)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, boltName, victimName)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertCounterCount(PlayerA, name, core.P1P1, 1)
	})

	t.Run("attacks trigger via GenericTriggered", func(t *testing.T) {
		name := "DSL Attack Pumper"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{1}{R}", 2, 2,
					mage.WithSubTypes("Warrior"),
					mage.WithAbility(mage.NewTriggered(
						core.EvtDeclaredAttacker,
						false,
						mage.Boost(mage.Fixed(2), mage.Fixed(0)).Targeting(mage.ToSource()),
					).SetConditionData(mage.EventSourceIsSelf{})),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerB, 16)
	})
}

func TestDestroyAllCollapse(t *testing.T) {
	t.Run("DestroyAllCreatures still works", func(t *testing.T) {
		name := "DSL Wrath"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewSorcery(name, "{2}{W}{W}",
					mage.NewSpellAbility(mage.DestroyAllCreatures()),
				)
			})
		}
		creatureName := "DSL Bear"
		if !mage.CardRegistered(creatureName) {
			mage.Register(creatureName, func() mage.Card {
				return mage.NewCreature(creatureName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, creatureName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, creatureName)
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, creatureName, 0)
		tg.AssertPermanentCount(PlayerB, creatureName, 0)
	})
}

func TestXVariantCollapse(t *testing.T) {
	t.Run("DealXDamage still works", func(t *testing.T) {
		name := "DSL X Damage"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewSorcery(name, "{X}{R}",
					mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.XValue())),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpellWithX(1, core.PrecombatMain, PlayerA, name, 5, "PlayerB")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerB, 15)
	})

	t.Run("DrawXCards still works", func(t *testing.T) {
		name := "DSL X Draw"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewSorcery(name, "{X}{U}",
					mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawCards(mage.XValue())),
				)
			})
		}

		tg := NewTestGame(t)
		for range 5 {
			tg.GetPlayer(PlayerA).AddToLibrary(mage.NewSorcery("Library Card", "{0}", nil))
		}
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpellWithX(1, core.PrecombatMain, PlayerA, name, 3, "PlayerA")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		if len(tg.GetPlayer(PlayerA).Hand()) != 3 {
			t.Errorf("expected 3 cards in hand, got %d", len(tg.GetPlayer(PlayerA).Hand()))
		}
	})
}

func TestFuncEffect(t *testing.T) {
	t.Run("inline effect via FuncEffect", func(t *testing.T) {
		name := "DSL Func Spell"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewSorcery(name, "{R}",
					mage.NewSpellAbility(mage.FuncEffect(
						"deal 3 damage to each player",
						mage.EffectProperties{Outcome: mage.OutcomeDetriment, DamageValue: mage.Fixed(3)},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							for _, p := range g.AllPlayers() {
								g.DealDamageToPlayer(p, 3, sourceID)
							}
							return nil
						},
					)),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerA, 17)
		tg.AssertLife(PlayerB, 17)
	})
}

func TestSourceTargetUnification(t *testing.T) {
	t.Run("BoostSourceUntilEndOfTurn pumps source", func(t *testing.T) {
		name := "DSL Boost Source"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewCreature(name, "{1}{R}", 2, 2,
					mage.WithSubTypes("Warrior"),
					mage.WithAbility(mage.NewTriggered(
						core.EvtDeclaredAttacker,
						false,
						mage.Boost(mage.Fixed(1), mage.Fixed(0)).Targeting(mage.ToSource()),
					).SetConditionData(mage.EventSourceIsSelf{})),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerB, 17)
	})
}

func TestConditionalTargetContinuousEffects(t *testing.T) {
	registerCh07bTestCards()
	for _, reg := range []struct {
		n string
		f func() mage.Card
	}{
		{"DSL Shatter", func() mage.Card {
			return mage.NewInstant("DSL Shatter", "{1}{R}",
				mage.NewTargetedSpell(mage.TargetArtifact(), mage.DestroyTargetArtifact()),
			)
		}},
		{"DSL Terror", func() mage.Card {
			return mage.NewInstant("DSL Terror", "{1}{B}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTargetNoRegen()),
			)
		}},
		{"Mountain", func() mage.Card {
			return mage.NewLand("Mountain",
				mage.WithManaAbility(core.Red),
				mage.WithSubTypes("Mountain"),
			)
		}},
	} {
		if !mage.CardRegistered(reg.n) {
			mage.Register(reg.n, reg.f)
		}
	}

	t.Run("target boost applies only while source is tapped", func(t *testing.T) {
		name := "DSL Tapped Source Weapon"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				return mage.NewArtifact(name, "{2}",
					mage.WithKeyword(core.AttrMayNotUntap),
					mage.WithActivatedAbility(
						mage.Boost(mage.Fixed(1), mage.Fixed(1)).
							Targeting(mage.ToTarget()).
							Until(core.WhileOnBattlefield).
							While(mage.SourceTapped),
						mage.GenericCost(2),
						mage.WithCost(mage.Tap()),
						mage.WithTarget(mage.TargetCreature()),
					),
				)
			})
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, "Grizzly Bears")
		tg.StopAt(1, core.BeginCombat)
		tg.Execute()

		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 3, 3)
		tg.AssertTapped(PlayerA, name, true)
	})

	t.Run("target boost ends when source untaps", func(t *testing.T) {
		name := "DSL Tapped Source Weapon"

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, "Grizzly Bears")
		tg.StopAt(3, core.PrecombatMain)
		tg.Execute()

		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
	})

	t.Run("ended target boost does not resume if source taps again", func(t *testing.T) {
		name := "DSL Tapped Source Weapon"

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, "Grizzly Bears")
		tg.ActivateAbility(3, core.PrecombatMain, PlayerA, name, "Hill Giant")
		tg.StopAt(3, core.BeginCombat)
		tg.Execute()

		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
		tg.AssertPowerToughness(PlayerA, "Hill Giant", 4, 4)
	})

	t.Run("target boost ends when source leaves", func(t *testing.T) {
		name := "DSL Tapped Source Weapon"

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.AddCard(core.ZoneHand, PlayerB, "DSL Shatter")
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, "Grizzly Bears")
		tg.CastSpell(1, core.BeginCombat, PlayerB, "DSL Shatter", name)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, name, 0)
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
	})

	t.Run("target boost ends when target leaves", func(t *testing.T) {
		name := "DSL Tapped Source Weapon"

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.AddCard(core.ZoneHand, PlayerB, "DSL Terror")
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, "Grizzly Bears")
		tg.CastSpell(1, core.BeginCombat, PlayerB, "DSL Terror", "Grizzly Bears")
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	})

	t.Run("revoke landwalk removes all landwalk until end of turn", func(t *testing.T) {
		name := "DSL Hammerheim Effect"
		walkerName := "DSL Double Landwalker"
		for _, reg := range []struct {
			n string
			f func() mage.Card
		}{
			{name, func() mage.Card {
				return mage.NewLand(name,
					mage.WithActivatedAbility(
						mage.RevokeLandwalk().Targeting(mage.ToTarget()).Until(core.EndOfTurn),
						mage.Tap(),
						mage.WithTarget(mage.TargetCreature()),
					),
				)
			}},
			{walkerName, func() mage.Card {
				return mage.NewCreature(walkerName, "{3}{B}", 3, 3,
					mage.WithKeyword(core.Swampwalk),
					mage.WithKeyword(core.Mountainwalk),
				)
			}},
		} {
			if !mage.CardRegistered(reg.n) {
				mage.Register(reg.n, reg.f)
			}
		}

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Swamp")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.AddCard(core.ZoneBattlefield, PlayerB, walkerName)
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, walkerName)
		tg.Attack(1, PlayerB, walkerName)
		tg.Block(1, PlayerA, "Grizzly Bears", walkerName)
		tg.StopAt(1, core.EndStep)
		tg.Execute()

		tg.AssertLife(PlayerA, 20)
	})

	t.Run("revoked landwalk returns after cleanup", func(t *testing.T) {
		name := "DSL Hammerheim Effect"
		walkerName := "DSL Double Landwalker"

		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Swamp")
		tg.AddCard(core.ZoneBattlefield, PlayerB, walkerName)
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, walkerName)
		tg.StopAt(2, core.PrecombatMain)
		tg.Execute()

		perm := tg.FindPermanentByName(walkerName, tg.GetPlayer(PlayerB).PlayerID())
		if perm == nil {
			t.Fatalf("expected %s on battlefield", walkerName)
		}
		if !perm.HasKeyword(core.Swampwalk) {
			t.Fatalf("expected swampwalk to return after cleanup")
		}
	})
}

func TestValueSource(t *testing.T) {
	t.Run("Fixed returns constant", func(t *testing.T) {
		v := mage.Fixed(7)
		got := v.Resolve(nil, uuid.Nil, uuid.Nil, nil)
		if got != 7 {
			t.Errorf("Fixed(7).Resolve() = %d, want 7", got)
		}
	})

	t.Run("XValue reads CurrentX", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.SetXValue(5)
		v := mage.XValue()
		got := v.Resolve(tg.Game, uuid.Nil, uuid.Nil, nil)
		if got != 5 {
			t.Errorf("XValue().Resolve() with CurrentX=5 = %d, want 5", got)
		}
	})
}

func TestPermanentSelector(t *testing.T) {
	if mage.SelectTarget == mage.SelectSource {
		t.Error("SelectTarget and SelectSource should be different values")
	}
}
