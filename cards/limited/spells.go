package limited

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	Register("Swords to Plowshares", func() Card {
		return NewInstant("Swords to Plowshares", "{W}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"exile target creature. Its controller gains life equal to its power",
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
						g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: permController, Amount: power})
					}
					return nil
				})),
		)
	})

	Register("Disenchant", func() Card {
		return NewInstant("Disenchant", "{1}{W}",
			NewTargetedSpell(TargetArtifactOrEnchantment(), DestroyTargetPermanent()),
		)
	})

	Register("Healing Salve", func() Card {
		c := NewInstant("Healing Salve", "{W}",
			NewTargetedSpell(TargetAnyTarget(), FuncEffect(
				"target player gains 3 life or prevent the next 3 damage that would be dealt to any target this turn",
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					if g.CurrentMode == 0 {
						for _, pl := range g.Players {
							if pl.PlayerID() == targets[0] {
								g.PlayerGainLife(pl, 3)
								return nil
							}
						}
					} else {
						g.Effects.AddPreventionShield(targets[0], 3)
					}
					return nil
				},
			)),
		)
		c.SetModes([]string{
			"Target player gains 3 life",
			"Prevent the next 3 damage that would be dealt to any target this turn",
		})
		return c
	})

	Register("Armageddon", func() Card {
		return NewSorcery("Armageddon", "{3}{W}",
			NewSpellAbility(DestroyAllLands()),
		)
	})

	Register("Balance", func() Card {
		return NewSorcery("Balance", "{1}{W}",
			NewSpellAbility(BalanceEffect()),
		)
	})

	Register("Resurrection", func() Card {
		return NewSorcery("Resurrection", "{2}{W}{W}",
			NewTargetedSpell(TargetCreatureInYourGraveyard(), ReturnFromGraveyardToBattlefield()),
		)
	})

	Register("Reverse Damage", func() Card {
		return NewInstant("Reverse Damage", "{1}{W}{W}",
			NewSpellAbility(FuncEffect(
				"prevent the next source of damage to you and gain that much life",
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					g.Effects.AddPreventionShield(controller, 1000)
					g.Effects.AddReverseDamageShield(controller)
					return nil
				})),
		)
	})

	Register("Death Ward", func() Card {
		return NewInstant("Death Ward", "{W}",
			NewTargetedSpell(TargetCreature(), RegenerateTarget()),
		)
	})

	Register("Guardian Angel", func() Card {
		return NewInstant("Guardian Angel", "{X}{W}",
			// Prevent the next X damage that would be dealt to any target this turn
			NewTargetedSpell(TargetAnyTarget(), PreventDamageToTarget(XValue())),
		)
	})

	// ===== BLUE SPELLS =====

	Register("Ancestral Recall", func() Card {
		return NewInstant("Ancestral Recall", "{U}",
			NewTargetedSpell(TargetPlayer(), DrawCards(Fixed(3))),
		)
	})

	Register("Braingeyser", func() Card {
		return NewSorcery("Braingeyser", "{X}{U}{U}",
			NewTargetedSpell(TargetPlayer(), DrawCards(XValue())),
		)
	})

	Register("Counterspell", func() Card {
		return NewInstant("Counterspell", "{U}{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpell()),
		)
	})

	Register("Unsummon", func() Card {
		return NewInstant("Unsummon", "{U}",
			NewTargetedSpell(TargetCreature(), ReturnToHandTarget()),
		)
	})

	Register("Spell Blast", func() Card {
		return NewInstant("Spell Blast", "{X}{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpellIfXMeetsCMC()),
		)
	})

	Register("Power Sink", func() Card {
		return NewInstant("Power Sink", "{X}{U}",
			NewTargetedSpell(TargetSpellOnStack(), PowerSinkEffect()),
		)
	})

	Register("Blue Elemental Blast", func() Card {
		return NewInstant("Blue Elemental Blast", "{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpellIfColor(Red)),
		)
	})

	Register("Twiddle", func() Card {
		return NewInstant("Twiddle", "{U}",
			NewTargetedSpell(TargetPermanent(), TapOrUntapTarget()),
		)
	})

	Register("Timetwister", func() Card {
		return NewSorcery("Timetwister", "{2}{U}",
			NewSpellAbility(ShuffleGraveyardIntoLibraryAndDraw(7)),
		)
	})

	Register("Time Walk", func() Card {
		return NewSorcery("Time Walk", "{1}{U}",
			NewSpellAbility(ExtraTurn()),
		)
	})

	Register("Sleight of Mind", func() Card {
		return NewInstant("Sleight of Mind", "{U}",
			NewTargetedSpell(TargetPermanent(), ReplaceKeywordEffect(Swampwalk, Forestwalk)),
		)
	})

	Register("Stasis", func() Card {
		return NewEnchantment("Stasis", "{1}{U}",
			// Players skip their untap steps.
			WithStaticAbility(PreventAllUntaps()),
			// At the beginning of your upkeep, sacrifice Stasis unless you pay {U}.
			WithAbility(SacrificeAtUpkeepUnlessPay("{U}")),
		)
	})

	// ===== BLACK SPELLS =====

	Register("Dark Ritual", func() Card {
		return NewInstant("Dark Ritual", "{B}",
			NewSpellAbility(AddMana(Black, 3)),
		)
	})

	Register("Terror", func() Card {
		return NewInstant("Terror", "{1}{B}",
			NewTargetedSpell(TargetCreature(
				Not(HasColorFilter(Black)),
				Not(IsArtifact),
			), DestroyTarget()),
		)
	})

	Register("Raise Dead", func() Card {
		return NewSorcery("Raise Dead", "{B}",
			NewTargetedSpell(TargetCreatureInYourGraveyard(), ReturnFromGraveyardToHandTarget()),
		)
	})

	Register("Demonic Tutor", func() Card {
		return NewSorcery("Demonic Tutor", "{1}{B}",
			NewSpellAbility(SearchLibraryToHand()),
		)
	})

	Register("Drain Life", func() Card {
		return NewSorcery("Drain Life", "{X}{1}{B}",
			NewTargetedSpell(TargetAnyTarget(), FuncEffect(
				"deal X damage to target and gain X life",
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
						g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: controller, Amount: amount})
					}
					return nil
				})),
		)
	})

	Register("Mind Twist", func() Card {
		return NewSorcery("Mind Twist", "{X}{B}",
			NewTargetedSpell(TargetPlayer(), DiscardCards(XValue())),
		)
	})

	Register("Sinkhole", func() Card {
		return NewLandDestruction("Sinkhole", "{B}{B}")
	})

	Register("Animate Dead", func() Card {
		return NewAura("Animate Dead", "{1}{B}",
			WithAbility(NewSpellAbility(ReturnFromGraveyardToBattlefield())),
			WithStaticAbility(
				BoostAttached(-1, 0, AttachAura),
			),
		)
	})

	Register("Pestilence", func() Card {
		return NewEnchantment("Pestilence", "{2}{B}{B}",
			// {B}: Deal 1 damage to each creature and each player
			WithActivatedAbility(
				CompositeEffects("deal 1 damage to each creature and each player",
					DealDamageToAllCreatures(Fixed(1), nil),
					DealDamageToPlayers(Fixed(1), SelectEachPlayer()),
				),
				ManaCostOf("{B}"),
			),
		)
	})

	Register("Red Elemental Blast", func() Card {
		return NewInstant("Red Elemental Blast", "{R}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpellIfColor(Blue)),
		)
	})

	// ===== RED SPELLS =====

	Register("Lightning Bolt", func() Card {
		return NewInstant("Lightning Bolt", "{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(3))),
		)
	})

	Register("Fireball", func() Card {
		return NewSorcery("Fireball", "{X}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(XValue())),
		)
	})

	Register("Disintegrate", func() Card {
		return NewSorcery("Disintegrate", "{X}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(XValue())),
		)
	})

	Register("Earthquake", func() Card {
		return NewSorcery("Earthquake", "{X}{R}",
			NewSpellAbility(CompositeEffects(
				"deal X damage to each creature without flying and each player",
				DealDamageToAllCreatures(XValue(), NotHasKeywordFilter(Flying)),
				DealDamageToPlayers(XValue(), SelectEachPlayer()),
			)),
		)
	})

	Register("Shatter", func() Card {
		return NewInstant("Shatter", "{1}{R}",
			NewTargetedSpell(TargetArtifact(), DestroyTargetArtifact()),
		)
	})

	Register("Stone Rain", func() Card {
		return NewLandDestruction("Stone Rain", "{2}{R}")
	})

	Register("Flashfires", func() Card {
		return NewSorcery("Flashfires", "{3}{R}",
			NewSpellAbility(DestroyAllMatching(
				HasSubType("Plains"),
				"destroy all Plains",
			)),
		)
	})

	Register("Wheel of Fortune", func() Card {
		return NewSorcery("Wheel of Fortune", "{2}{R}",
			NewSpellAbility(DiscardHandAndDraw(7)),
		)
	})

	Register("Fork", func() Card {
		return NewInstant("Fork", "{R}{R}",
			NewTargetedSpell(TargetSpellOnStack(), CopySpellOnStack()),
		)
	})

	Register("Berserk", func() Card {
		return NewInstant("Berserk", "{G}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"Target creature's power is doubled. Destroy it at end of turn.",
				DoubleTargetPower(),
				DestroyTargetAtEndOfTurn(),
			)),
		)
	})

	// ===== GREEN SPELLS =====

	Register("Giant Growth", func() Card {
		return NewInstant("Giant Growth", "{G}",
			NewTargetedSpell(TargetCreature(), BoostUntilEndOfTurn(Fixed(3), Fixed(3), SelectTarget)),
		)
	})

	Register("Regrowth", func() Card {
		return NewSorcery("Regrowth", "{1}{G}",
			NewTargetedSpell(TargetCardInYourGraveyard(), ReturnFromGraveyardToHandTarget()),
		)
	})

	Register("Hurricane", func() Card {
		return NewSorcery("Hurricane", "{X}{G}",
			NewSpellAbility(CompositeEffects(
				"deal X damage to each creature with flying and each player",
				DealDamageToAllCreatures(XValue(), HasKeywordFilter(Flying)),
				DealDamageToPlayers(XValue(), SelectEachPlayer()),
			)),
		)
	})

	Register("Tranquility", func() Card {
		return NewSorcery("Tranquility", "{2}{G}",
			NewSpellAbility(DestroyAllEnchantments()),
		)
	})

	Register("Tsunami", func() Card {
		return NewSorcery("Tsunami", "{3}{G}",
			NewSpellAbility(DestroyAllMatching(
				HasSubType("Island"),
				"destroy all Islands",
			)),
		)
	})

	Register("Ice Storm", func() Card {
		return NewLandDestruction("Ice Storm", "{2}{G}")
	})

	Register("Stream of Life", func() Card {
		return NewSorcery("Stream of Life", "{X}{G}",
			NewTargetedSpell(TargetPlayer(), GainLifeTarget(XValue())),
		)
	})

	Register("Fog", func() Card {
		return NewInstant("Fog", "{G}",
			NewSpellAbility(PreventAllCombatDamage()),
		)
	})

	Register("Channel", func() Card {
		return NewSorcery("Channel", "{G}{G}",
			NewSpellAbility(FuncEffect(
				"until end of turn, pay 1 life to add {C}",
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					g.Effects.SetChannelActive(controller)
					return nil
				})),
		)
	})

	Register("Howl from Beyond", func() Card {
		return NewInstant("Howl from Beyond", "{X}{B}",
			NewTargetedSpell(TargetCreature(), BoostUntilEndOfTurn(XValue(), Fixed(0), SelectTarget)),
		)
	})

	Register("Righteousness", func() Card {
		return NewInstant("Righteousness", "{W}",
			NewTargetedSpell(TargetCreature(), BoostUntilEndOfTurn(Fixed(7), Fixed(7), SelectTarget)),
		)
	})

	// ===== COLORLESS SPELLS =====

	Register("Chaos Orb", func() Card {
		return NewArtifact("Chaos Orb", "{2}",
			// {1}, {T}: Destroy a random nontoken permanent you don't control, then destroy Chaos Orb.
			WithActivatedAbility(
				ChaosOrbEffect(),
				GenericCost(1),
				WithCost(TapSourceCost()),
			),
		)
	})

	Register("Wrath of God", func() Card {
		return NewSorcery("Wrath of God", "{2}{W}{W}",
			NewSpellAbility(DestroyAllCreatures()),
		)
	})
}
