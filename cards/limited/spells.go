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
		c := mage.NewInstant("Swords to Plowshares", "{W}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
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
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Disenchant", func() mage.Card {
		c := mage.NewInstant("Disenchant", "{1}{W}")
		sa := mage.NewTargetedSpell(mage.TargetArtifactOrEnchantment(), mage.DestroyTargetPermanent())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Healing Salve", func() mage.Card {
		c := mage.NewInstant("Healing Salve", "{W}")
		// Mode 1: Target player gains 3 life
		// Mode 2: Prevent the next 3 damage to any target
		// Auto-detect: if targeting a player, gain life; otherwise prevent damage.
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.FuncEffect(
			"gain 3 life or prevent 3 damage",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				if len(targets) == 0 {
					return nil
				}
				// If target is a player, gain 3 life
				for _, pl := range g.Players {
					if pl.PlayerID() == targets[0] {
						g.PlayerGainLife(pl, 3)
						return nil
					}
				}
				// Otherwise prevent 3 damage to target permanent
				g.Effects.AddPreventionShield(targets[0], 3)
				return nil
			},
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Armageddon", func() mage.Card {
		c := mage.NewSorcery("Armageddon", "{3}{W}")
		sa := mage.NewSpellAbility(mage.DestroyAllLands())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Balance", func() mage.Card {
		c := mage.NewSorcery("Balance", "{1}{W}")
		sa := mage.NewSpellAbility(mage.BalanceEffect())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Resurrection", func() mage.Card {
		c := mage.NewSorcery("Resurrection", "{2}{W}{W}")
		sa := mage.NewTargetedSpell(mage.TargetCreatureInYourGraveyard(), mage.ReturnFromGraveyardToBattlefield())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Reverse Damage", func() mage.Card {
		c := mage.NewInstant("Reverse Damage", "{1}{W}{W}")
		sa := mage.NewSpellAbility(mage.FuncEffect(
			"prevent the next source of damage to you and gain that much life",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				// Set a large prevention shield on the controller and mark reverse damage
				g.Effects.AddPreventionShield(controller, 1000)
				g.Effects.AddReverseDamageShield(controller)
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Death Ward", func() mage.Card {
		c := mage.NewInstant("Death Ward", "{W}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.RegenerateTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Guardian Angel", func() mage.Card {
		c := mage.NewInstant("Guardian Angel", "{X}{W}")
		// Prevent the next X damage that would be dealt to any target this turn
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.PreventDamageToTarget(mage.XValue()))
		c.AddAbility(sa)
		return c
	})

	// ===== BLUE SPELLS =====

	mage.Register("Ancestral Recall", func() mage.Card {
		c := mage.NewInstant("Ancestral Recall", "{U}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawCards(mage.Fixed(3)))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Braingeyser", func() mage.Card {
		c := mage.NewSorcery("Braingeyser", "{X}{U}{U}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawCards(mage.XValue()))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Counterspell", func() mage.Card {
		c := mage.NewInstant("Counterspell", "{U}{U}")
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpell())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Unsummon", func() mage.Card {
		c := mage.NewInstant("Unsummon", "{U}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.ReturnToHandTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Spell Blast", func() mage.Card {
		c := mage.NewInstant("Spell Blast", "{X}{U}")
		// Counter target spell with mana value X
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfXMeetsCMC())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Power Sink", func() mage.Card {
		c := mage.NewInstant("Power Sink", "{X}{U}")
		// Counter target spell unless its controller pays {X}
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.PowerSinkEffect())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Blue Elemental Blast", func() mage.Card {
		c := mage.NewInstant("Blue Elemental Blast", "{U}")
		// Counter target red spell (only counters if the spell is red)
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfColor(core.Red))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Twiddle", func() mage.Card {
		c := mage.NewInstant("Twiddle", "{U}")
		// You may tap or untap target artifact, creature, or land
		sa := mage.NewTargetedSpell(mage.TargetPermanent(), mage.TapOrUntapTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Timetwister", func() mage.Card {
		// Each player shuffles their hand and graveyard into library, then draws 7
		c := mage.NewSorcery("Timetwister", "{2}{U}")
		sa := mage.NewSpellAbility(mage.ShuffleGraveyardIntoLibraryAndDraw(7))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Time Walk", func() mage.Card {
		c := mage.NewSorcery("Time Walk", "{1}{U}")
		sa := mage.NewSpellAbility(mage.ExtraTurn())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Sleight of Mind", func() mage.Card {
		c := mage.NewInstant("Sleight of Mind", "{U}")
		// Change the text of target permanent by replacing all instances of one
		// color word with another. Default: swamp->forest (swampwalk->forestwalk).
		sa := mage.NewTargetedSpell(mage.TargetPermanent(), mage.ReplaceKeywordEffect(core.Swampwalk, core.Forestwalk))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Stasis", func() mage.Card {
		c := mage.NewEnchantment("Stasis", "{1}{U}")
		// Players skip their untap steps.
		c.AddAbility(mage.StaticAbility(
			mage.PreventAllUntaps(),
		))
		// At the beginning of your upkeep, sacrifice Stasis unless you pay {U}.
		c.AddAbility(mage.SacrificeAtUpkeepUnlessPay("{U}"))
		return c
	})

	// ===== BLACK SPELLS =====

	mage.Register("Dark Ritual", func() mage.Card {
		c := mage.NewInstant("Dark Ritual", "{B}")
		sa := mage.NewSpellAbility(mage.AddMana(core.Black, 3))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Terror", func() mage.Card {
		c := mage.NewInstant("Terror", "{1}{B}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(
			mage.Not(mage.HasColorFilter(core.Black)),
			mage.Not(mage.IsArtifact),
		), mage.DestroyTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Raise Dead", func() mage.Card {
		c := mage.NewSorcery("Raise Dead", "{B}")
		sa := mage.NewTargetedSpell(mage.TargetCreatureInYourGraveyard(), mage.ReturnFromGraveyardToHandTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Demonic Tutor", func() mage.Card {
		c := mage.NewSorcery("Demonic Tutor", "{1}{B}")
		sa := mage.NewSpellAbility(mage.SearchLibraryToHand())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Drain Life", func() mage.Card {
		c := mage.NewSorcery("Drain Life", "{X}{1}{B}")
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.FuncEffect(
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
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Mind Twist", func() mage.Card {
		c := mage.NewSorcery("Mind Twist", "{X}{B}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.DiscardCards(mage.XValue()))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Sinkhole", func() mage.Card {
		return mage.NewLandDestruction("Sinkhole", "{B}{B}")
	})

	mage.Register("Animate Dead", func() mage.Card {
		c := mage.NewAura("Animate Dead", "{1}{B}")
		// Return target creature card from a graveyard to the battlefield.
		// Animate Dead attaches to it. Enchanted creature gets -1/-0.
		sa := mage.NewSpellAbility(mage.ReturnFromGraveyardToBattlefield())
		c.AddAbility(sa)
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(-1, 0, core.AttachAura),
		))
		return c
	})

	mage.Register("Pestilence", func() mage.Card {
		c := mage.NewEnchantment("Pestilence", "{2}{B}{B}")
		// {B}: Deal 1 damage to each creature and each player
		ab := mage.NewActivatedAbility(
			mage.CompositeEffects("deal 1 damage to each creature and each player",
				mage.DealDamageToAllCreatures(mage.Fixed(1), nil),
				mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectEachPlayer()),
			),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Red Elemental Blast", func() mage.Card {
		c := mage.NewInstant("Red Elemental Blast", "{R}")
		// Counter target blue spell (only counters if the spell is blue)
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfColor(core.Blue))
		c.AddAbility(sa)
		return c
	})

	// ===== RED SPELLS =====

	mage.Register("Lightning Bolt", func() mage.Card {
		c := mage.NewInstant("Lightning Bolt", "{R}")
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3)))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Fireball", func() mage.Card {
		c := mage.NewSorcery("Fireball", "{X}{R}")
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue()))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Disintegrate", func() mage.Card {
		c := mage.NewSorcery("Disintegrate", "{X}{R}")
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue()))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Earthquake", func() mage.Card {
		c := mage.NewSorcery("Earthquake", "{X}{R}")
		// Deal X damage to each creature without flying and each player
		sa := mage.NewSpellAbility(mage.CompositeEffects(
			"deal X damage to each creature without flying and each player",
			mage.DealDamageToAllCreatures(mage.XValue(), mage.NotHasKeywordFilter(core.Flying)),
			mage.DealDamageToPlayers(mage.XValue(), mage.SelectEachPlayer()),
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Shatter", func() mage.Card {
		c := mage.NewInstant("Shatter", "{1}{R}")
		sa := mage.NewTargetedSpell(mage.TargetArtifact(), mage.DestroyTargetArtifact())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Stone Rain", func() mage.Card {
		return mage.NewLandDestruction("Stone Rain", "{2}{R}")
	})

	mage.Register("Flashfires", func() mage.Card {
		c := mage.NewSorcery("Flashfires", "{3}{R}")
		// Destroy all Plains
		sa := mage.NewSpellAbility(mage.DestroyAllMatching(
			mage.HasSubType("Plains"),
			"destroy all Plains",
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Wheel of Fortune", func() mage.Card {
		c := mage.NewSorcery("Wheel of Fortune", "{2}{R}")
		// Each player discards their hand, then draws seven cards
		sa := mage.NewSpellAbility(mage.DiscardHandAndDraw(7))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Fork", func() mage.Card {
		// Copy target instant or sorcery spell
		c := mage.NewInstant("Fork", "{R}{R}")
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CopySpellOnStack())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Berserk", func() mage.Card {
		c := mage.NewInstant("Berserk", "{G}")
		// Double target creature's power until end of turn, destroy it at end of turn
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.CompositeEffects(
			"Target creature's power is doubled. Destroy it at end of turn.",
			mage.DoubleTargetPower(),
			mage.DestroyTargetAtEndOfTurn(),
		))
		c.AddAbility(sa)
		return c
	})

	// ===== GREEN SPELLS =====

	mage.Register("Giant Growth", func() mage.Card {
		c := mage.NewInstant("Giant Growth", "{G}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostUntilEndOfTurn(mage.Fixed(3), mage.Fixed(3), mage.SelectTarget))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Regrowth", func() mage.Card {
		c := mage.NewSorcery("Regrowth", "{1}{G}")
		sa := mage.NewTargetedSpell(mage.TargetCardInYourGraveyard(), mage.ReturnFromGraveyardToHandTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Hurricane", func() mage.Card {
		c := mage.NewSorcery("Hurricane", "{X}{G}")
		// Deal X damage to each creature with flying and each player
		sa := mage.NewSpellAbility(mage.CompositeEffects(
			"deal X damage to each creature with flying and each player",
			mage.DealDamageToAllCreatures(mage.XValue(), mage.HasKeywordFilter(core.Flying)),
			mage.DealDamageToPlayers(mage.XValue(), mage.SelectEachPlayer()),
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Tranquility", func() mage.Card {
		c := mage.NewSorcery("Tranquility", "{2}{G}")
		sa := mage.NewSpellAbility(mage.DestroyAllEnchantments())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Tsunami", func() mage.Card {
		c := mage.NewSorcery("Tsunami", "{3}{G}")
		// Destroy all Islands
		sa := mage.NewSpellAbility(mage.DestroyAllMatching(
			mage.HasSubType("Island"),
			"destroy all Islands",
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Ice Storm", func() mage.Card {
		return mage.NewLandDestruction("Ice Storm", "{2}{G}")
	})

	mage.Register("Stream of Life", func() mage.Card {
		c := mage.NewSorcery("Stream of Life", "{X}{G}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.GainLifeTarget(mage.XValue()))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Fog", func() mage.Card {
		c := mage.NewInstant("Fog", "{G}")
		sa := mage.NewSpellAbility(mage.PreventAllCombatDamage())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Channel", func() mage.Card {
		c := mage.NewSorcery("Channel", "{G}{G}")
		// Until end of turn, you may pay 1 life to add {C}.
		// Simplified: add a large pool of colorless mana and lose life when used.
		// We implement by setting a channel flag that charges life for X costs.
		sa := mage.NewSpellAbility(mage.FuncEffect(
			"until end of turn, pay 1 life to add {C}",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				g.Effects.SetChannelActive(controller)
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Howl from Beyond", func() mage.Card {
		c := mage.NewInstant("Howl from Beyond", "{X}{B}")
		// Target creature gets +X/+0 until end of turn
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostUntilEndOfTurn(mage.XValue(), mage.Fixed(0), mage.SelectTarget))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Righteousness", func() mage.Card {
		c := mage.NewInstant("Righteousness", "{W}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostUntilEndOfTurn(mage.Fixed(7), mage.Fixed(7), mage.SelectTarget))
		c.AddAbility(sa)
		return c
	})

	// ===== COLORLESS SPELLS =====

	mage.Register("Chaos Orb", func() mage.Card {
		c := mage.NewArtifact("Chaos Orb", "{2}")
		// {1}, {T}: Destroy a random nontoken permanent you don't control, then destroy Chaos Orb.
		ab := mage.NewActivatedAbility(
			mage.ChaosOrbEffect(),
			mage.GenericCost(1),
			mage.WithCost(mage.TapSourceCost()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Wrath of God", func() mage.Card {
		c := mage.NewSorcery("Wrath of God", "{2}{W}{W}")
		sa := mage.NewSpellAbility(mage.DestroyAllCreatures())
		c.AddAbility(sa)
		return c
	})
}
