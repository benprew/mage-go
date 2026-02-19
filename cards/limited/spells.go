package limited

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	mage.Register("Swords to Plowshares", func() mage.Card {
		return mage.NewInstant("Swords to Plowshares", "{W}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
				"exile target creature. Its controller gains life equal to its power",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return fmt.Errorf("no target")
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					power := perm.CurrentPower(g)
					permController := perm.Controller
					g.ExilePermanent(perm)
					p := g.GetPlayer(permController)
					if p != nil && power > 0 {
						p.GainLife(power)
						g.FireEvent(core.GameEvent{Type: core.EvtLifeGained, PlayerID: permController, Amount: power})
					}
					return nil
				}))),
		)
	})

	mage.Register("Disenchant", func() mage.Card {
		return mage.NewInstant("Disenchant", "{1}{W}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetArtifactOrEnchantment(), mage.DestroyTargetPermanent())),
		)
	})

	mage.Register("Healing Salve", func() mage.Card {
		return mage.NewInstant("Healing Salve", "{W}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.FuncEffect(
				"gain 3 life or prevent 3 damage",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					for _, pl := range g.Players {
						if pl.PlayerID() == targets[0] {
							g.PlayerGainLife(pl, 3)
							return nil
						}
					}
					g.Effects.AddPreventionShield(targets[0], 3)
					return nil
				},
			))),
		)
	})

	mage.Register("Armageddon", func() mage.Card {
		return mage.NewSorcery("Armageddon", "{3}{W}",
			mage.WithAbility(mage.NewSpellAbility(mage.DestroyAllLands())),
		)
	})

	mage.Register("Balance", func() mage.Card {
		return mage.NewSorcery("Balance", "{1}{W}",
			mage.WithAbility(mage.NewSpellAbility(mage.BalanceEffect())),
		)
	})

	mage.Register("Resurrection", func() mage.Card {
		return mage.NewSorcery("Resurrection", "{2}{W}{W}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreatureInYourGraveyard(), mage.ReturnFromGraveyardToBattlefield())),
		)
	})

	mage.Register("Reverse Damage", func() mage.Card {
		return mage.NewInstant("Reverse Damage", "{1}{W}{W}",
			mage.WithAbility(mage.NewSpellAbility(mage.FuncEffect(
				"prevent the next source of damage to you and gain that much life",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					g.Effects.AddPreventionShield(controller, 1000)
					g.Effects.AddReverseDamageShield(controller)
					return nil
				}))),
		)
	})

	mage.Register("Death Ward", func() mage.Card {
		return mage.NewInstant("Death Ward", "{W}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.RegenerateTarget())),
		)
	})

	mage.Register("Guardian Angel", func() mage.Card {
		return mage.NewInstant("Guardian Angel", "{X}{W}",
			// Prevent the next X damage that would be dealt to any target this turn
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.PreventDamageToTarget(mage.XValue()))),
		)
	})

	// ===== BLUE SPELLS =====

	mage.Register("Ancestral Recall", func() mage.Card {
		return mage.NewInstant("Ancestral Recall", "{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawCards(mage.Fixed(3)))),
		)
	})

	mage.Register("Braingeyser", func() mage.Card {
		return mage.NewSorcery("Braingeyser", "{X}{U}{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawCards(mage.XValue()))),
		)
	})

	mage.Register("Counterspell", func() mage.Card {
		return mage.NewInstant("Counterspell", "{U}{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpell())),
		)
	})

	mage.Register("Unsummon", func() mage.Card {
		return mage.NewInstant("Unsummon", "{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.ReturnToHandTarget())),
		)
	})

	mage.Register("Spell Blast", func() mage.Card {
		return mage.NewInstant("Spell Blast", "{X}{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfXMeetsCMC())),
		)
	})

	mage.Register("Power Sink", func() mage.Card {
		return mage.NewInstant("Power Sink", "{X}{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.PowerSinkEffect())),
		)
	})

	mage.Register("Blue Elemental Blast", func() mage.Card {
		return mage.NewInstant("Blue Elemental Blast", "{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfColor(core.Red))),
		)
	})

	mage.Register("Twiddle", func() mage.Card {
		return mage.NewInstant("Twiddle", "{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPermanent(), mage.TapOrUntapTarget())),
		)
	})

	mage.Register("Timetwister", func() mage.Card {
		return mage.NewSorcery("Timetwister", "{2}{U}",
			mage.WithAbility(mage.NewSpellAbility(mage.ShuffleGraveyardIntoLibraryAndDraw(7))),
		)
	})

	mage.Register("Time Walk", func() mage.Card {
		return mage.NewSorcery("Time Walk", "{1}{U}",
			mage.WithAbility(mage.NewSpellAbility(mage.ExtraTurn())),
		)
	})

	mage.Register("Sleight of Mind", func() mage.Card {
		return mage.NewInstant("Sleight of Mind", "{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPermanent(), mage.ReplaceKeywordEffect(core.Swampwalk, core.Forestwalk))),
		)
	})

	mage.Register("Stasis", func() mage.Card {
		return mage.NewEnchantment("Stasis", "{1}{U}",
			// Players skip their untap steps.
			mage.WithAbility(mage.StaticAbility(mage.PreventAllUntaps())),
			// At the beginning of your upkeep, sacrifice Stasis unless you pay {U}.
			mage.WithAbility(mage.SacrificeAtUpkeepUnlessPay("{U}")),
		)
	})

	// ===== BLACK SPELLS =====

	mage.Register("Dark Ritual", func() mage.Card {
		return mage.NewInstant("Dark Ritual", "{B}",
			mage.WithAbility(mage.NewSpellAbility(mage.AddMana(core.Black, 3))),
		)
	})

	mage.Register("Terror", func() mage.Card {
		return mage.NewInstant("Terror", "{1}{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(
				mage.Not(mage.HasColorFilter(core.Black)),
				mage.Not(mage.IsArtifact),
			), mage.DestroyTarget())),
		)
	})

	mage.Register("Raise Dead", func() mage.Card {
		return mage.NewSorcery("Raise Dead", "{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreatureInYourGraveyard(), mage.ReturnFromGraveyardToHandTarget())),
		)
	})

	mage.Register("Demonic Tutor", func() mage.Card {
		return mage.NewSorcery("Demonic Tutor", "{1}{B}",
			mage.WithAbility(mage.NewSpellAbility(mage.SearchLibraryToHand())),
		)
	})

	mage.Register("Drain Life", func() mage.Card {
		return mage.NewSorcery("Drain Life", "{X}{1}{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.FuncEffect(
				"deal X damage to target and gain X life",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return fmt.Errorf("no target for drain life")
					}
					amount := g.CurrentX
					if amount <= 0 {
						return nil
					}
					target := targets[0]
					perm := g.FindPermanent(target)
					if perm != nil {
						g.DealDamageToPermanent(perm, amount, sourceID)
					} else {
						p := g.GetPlayer(target)
						if p != nil {
							g.DealDamageToPlayer(p, amount, sourceID)
						}
					}
					caster := g.GetPlayer(controller)
					if caster != nil {
						caster.GainLife(amount)
						g.FireEvent(core.GameEvent{Type: core.EvtLifeGained, PlayerID: controller, Amount: amount})
					}
					return nil
				}))),
		)
	})

	mage.Register("Mind Twist", func() mage.Card {
		return mage.NewSorcery("Mind Twist", "{X}{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.DiscardCards(mage.XValue()))),
		)
	})

	mage.Register("Sinkhole", func() mage.Card {
		return mage.NewLandDestruction("Sinkhole", "{B}{B}")
	})

	mage.Register("Animate Dead", func() mage.Card {
		return mage.NewAura("Animate Dead", "{1}{B}",
			mage.WithAbility(mage.NewSpellAbility(mage.ReturnFromGraveyardToBattlefield())),
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAttached(-1, 0, core.AttachAura),
			)),
		)
	})

	mage.Register("Pestilence", func() mage.Card {
		return mage.NewEnchantment("Pestilence", "{2}{B}{B}",
			// {B}: Deal 1 damage to each creature and each player
			mage.WithAbility(mage.NewActivatedAbility(
				mage.CompositeEffects("deal 1 damage to each creature and each player",
					mage.DealDamageToAllCreatures(mage.Fixed(1), nil),
					mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectEachPlayer()),
				),
				mage.ManaCostOf("{B}"),
			)),
		)
	})

	mage.Register("Red Elemental Blast", func() mage.Card {
		return mage.NewInstant("Red Elemental Blast", "{R}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfColor(core.Blue))),
		)
	})

	// ===== RED SPELLS =====

	mage.Register("Lightning Bolt", func() mage.Card {
		return mage.NewInstant("Lightning Bolt", "{R}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3)))),
		)
	})

	mage.Register("Fireball", func() mage.Card {
		return mage.NewSorcery("Fireball", "{X}{R}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue()))),
		)
	})

	mage.Register("Disintegrate", func() mage.Card {
		return mage.NewSorcery("Disintegrate", "{X}{R}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue()))),
		)
	})

	mage.Register("Earthquake", func() mage.Card {
		return mage.NewSorcery("Earthquake", "{X}{R}",
			mage.WithAbility(mage.NewSpellAbility(mage.CompositeEffects(
				"deal X damage to each creature without flying and each player",
				mage.DealDamageToAllCreatures(mage.XValue(), mage.NotHasKeywordFilter(core.Flying)),
				mage.DealDamageToPlayers(mage.XValue(), mage.SelectEachPlayer()),
			))),
		)
	})

	mage.Register("Shatter", func() mage.Card {
		return mage.NewInstant("Shatter", "{1}{R}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetArtifact(), mage.DestroyTargetArtifact())),
		)
	})

	mage.Register("Stone Rain", func() mage.Card {
		return mage.NewLandDestruction("Stone Rain", "{2}{R}")
	})

	mage.Register("Flashfires", func() mage.Card {
		return mage.NewSorcery("Flashfires", "{3}{R}",
			mage.WithAbility(mage.NewSpellAbility(mage.DestroyAllMatching(
				mage.HasSubType("Plains"),
				"destroy all Plains",
			))),
		)
	})

	mage.Register("Wheel of Fortune", func() mage.Card {
		return mage.NewSorcery("Wheel of Fortune", "{2}{R}",
			mage.WithAbility(mage.NewSpellAbility(mage.DiscardHandAndDraw(7))),
		)
	})

	mage.Register("Fork", func() mage.Card {
		return mage.NewInstant("Fork", "{R}{R}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CopySpellOnStack())),
		)
	})

	mage.Register("Berserk", func() mage.Card {
		return mage.NewInstant("Berserk", "{G}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.CompositeEffects(
				"Target creature's power is doubled. Destroy it at end of turn.",
				mage.DoubleTargetPower(),
				mage.DestroyTargetAtEndOfTurn(),
			))),
		)
	})

	// ===== GREEN SPELLS =====

	mage.Register("Giant Growth", func() mage.Card {
		return mage.NewInstant("Giant Growth", "{G}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostUntilEndOfTurn(mage.Fixed(3), mage.Fixed(3), mage.SelectTarget))),
		)
	})

	mage.Register("Regrowth", func() mage.Card {
		return mage.NewSorcery("Regrowth", "{1}{G}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCardInYourGraveyard(), mage.ReturnFromGraveyardToHandTarget())),
		)
	})

	mage.Register("Hurricane", func() mage.Card {
		return mage.NewSorcery("Hurricane", "{X}{G}",
			mage.WithAbility(mage.NewSpellAbility(mage.CompositeEffects(
				"deal X damage to each creature with flying and each player",
				mage.DealDamageToAllCreatures(mage.XValue(), mage.HasKeywordFilter(core.Flying)),
				mage.DealDamageToPlayers(mage.XValue(), mage.SelectEachPlayer()),
			))),
		)
	})

	mage.Register("Tranquility", func() mage.Card {
		return mage.NewSorcery("Tranquility", "{2}{G}",
			mage.WithAbility(mage.NewSpellAbility(mage.DestroyAllEnchantments())),
		)
	})

	mage.Register("Tsunami", func() mage.Card {
		return mage.NewSorcery("Tsunami", "{3}{G}",
			mage.WithAbility(mage.NewSpellAbility(mage.DestroyAllMatching(
				mage.HasSubType("Island"),
				"destroy all Islands",
			))),
		)
	})

	mage.Register("Ice Storm", func() mage.Card {
		return mage.NewLandDestruction("Ice Storm", "{2}{G}")
	})

	mage.Register("Stream of Life", func() mage.Card {
		return mage.NewSorcery("Stream of Life", "{X}{G}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.GainLifeTarget(mage.XValue()))),
		)
	})

	mage.Register("Fog", func() mage.Card {
		return mage.NewInstant("Fog", "{G}",
			mage.WithAbility(mage.NewSpellAbility(mage.PreventAllCombatDamage())),
		)
	})

	mage.Register("Channel", func() mage.Card {
		return mage.NewSorcery("Channel", "{G}{G}",
			mage.WithAbility(mage.NewSpellAbility(mage.FuncEffect(
				"until end of turn, pay 1 life to add {C}",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					g.Effects.SetChannelActive(controller)
					return nil
				}))),
		)
	})

	mage.Register("Howl from Beyond", func() mage.Card {
		return mage.NewInstant("Howl from Beyond", "{X}{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostUntilEndOfTurn(mage.XValue(), mage.Fixed(0), mage.SelectTarget))),
		)
	})

	mage.Register("Righteousness", func() mage.Card {
		return mage.NewInstant("Righteousness", "{W}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostUntilEndOfTurn(mage.Fixed(7), mage.Fixed(7), mage.SelectTarget))),
		)
	})

	// ===== COLORLESS SPELLS =====

	mage.Register("Chaos Orb", func() mage.Card {
		return mage.NewArtifact("Chaos Orb", "{2}",
			// {1}, {T}: Destroy a random nontoken permanent you don't control, then destroy Chaos Orb.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.ChaosOrbEffect(),
				mage.GenericCost(1),
				mage.WithCost(mage.TapSourceCost()),
			)),
		)
	})

	mage.Register("Wrath of God", func() mage.Card {
		return mage.NewSorcery("Wrath of God", "{2}{W}{W}",
			mage.WithAbility(mage.NewSpellAbility(mage.DestroyAllCreatures())),
		)
	})
}
