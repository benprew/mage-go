package limited

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// ===== MOX CYCLE =====

	moxen := []struct {
		name  string
		color Color
	}{
		{"Mox Pearl", White},
		{"Mox Sapphire", Blue},
		{"Mox Jet", Black},
		{"Mox Ruby", Red},
		{"Mox Emerald", Green},
	}

	for _, m := range moxen {
		name := m.name
		color := m.color
		Register(name, func() Card {
			return NewArtifact(name, "{0}", WithManaAbility(color))
		})
	}

	// ===== MANA ARTIFACTS =====

	Register("Black Lotus", func() Card {
		return NewArtifact("Black Lotus", "{0}",
			// {T}, Sacrifice: Add 3 mana of any one color
			WithActivatedAbility(
				AddAnyMana(3, Green),
				Tap(),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	Register("Sol Ring", func() Card {
		return NewArtifact("Sol Ring", "{1}",
			// {T}: Add {C}{C}
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}),
		)
	})

	Register("Basalt Monolith", func() Card {
		return NewArtifact("Basalt Monolith", "{3}",
			WithKeyword(DoesNotUntapKW),
			// {T}: Add {C}{C}{C}
			WithActivatedAbility(
				AddMana(Colorless, 3),
				Tap(),
			),
			// {3}: Untap Basalt Monolith
			WithActivatedAbility(
				UntapSource(),
				GenericCost(3),
			),
		)
	})

	Register("Mana Vault", func() Card {
		return NewArtifact("Mana Vault", "{1}",
			WithKeyword(DoesNotUntapKW),
			// {T}: Add {C}{C}{C}
			WithActivatedAbility(
				AddMana(Colorless, 3),
				Tap(),
			),
			// At the beginning of your upkeep, you may pay {4}. If you do, untap Mana Vault.
			WithAbility(BeginningOfUpkeepTrigger(
				IfElse(
					"Only if tapped",
					SourceIsTapped{},
					EffectIfPaid(
						ManaCostOf("{4}"),
						UntapSource(),
					), nil),
				true)),
			// At the beginning of your draw step, if Mana Vault is tapped, it deals 1 damage to you.
			WithAbility(NewTriggered(EvtDrawStep, false, DealDamageToPlayers(Fixed(1), SelectController())).
				SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{EventPlayerIsController{}, SourceIsTapped{}}})),
		)
	})

	// ===== UTILITY ARTIFACTS =====

	Register("Jayemdae Tome", func() Card {
		return NewArtifact("Jayemdae Tome", "{4}",
			// {4}, {T}: Draw a card
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				GenericCost(4),
				WithCost(Tap()),
			),
		)
	})

	Register("Disrupting Scepter", func() Card {
		// XXX: missing "Activate only during your turn" restriction
		return NewArtifact("Disrupting Scepter", "{3}",
			// {3}, {T}: Target player discards a card
			WithActivatedAbility(
				DiscardCards(Fixed(1)),
				GenericCost(3),
				WithCost(Tap()),
				WithTarget(TargetPlayer()),
			),
		)
	})

	Register("Icy Manipulator", func() Card {
		return NewArtifact("Icy Manipulator", "{4}",
			// {1}, {T}: Tap target permanent
			WithActivatedAbility(
				Tap(),
				GenericCost(1),
				WithCost(Tap()),
				WithTarget(TargetPermanent(Or(IsArtifact, IsCreature, IsLand))),
			),
		)
	})

	Register("Rod of Ruin", func() Card {
		return NewArtifact("Rod of Ruin", "{4}",
			// {3}, {T}: Deal 1 damage to any target
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				GenericCost(3),
				WithCost(Tap()),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	Register("The Hive", func() Card {
		return NewArtifact("The Hive", "{5}",
			// {5}, {T}: Create a 1/1 colorless Insect artifact creature token with flying named Wasp.
			WithActivatedAbility(
				CreateToken("Wasp", 1, 1,
					[]CardType{TypeArtifact, TypeCreature},
					[]string{"Insect"},
					Flying,
				),
				GenericCost(5),
				WithCost(Tap()),
			),
		)
	})

	Register("Winter Orb", func() Card {
		return NewArtifact("Winter Orb", "{2}",
			WithStaticAbility(LimitLandUntaps(1)),
		)
	})

	Register("Meekstone", func() Card {
		return NewArtifact("Meekstone", "{1}",
			WithStaticAbility(
				PreventUntapForMatching(And(IsCreature, HasPowerGTE(3))),
			),
		)
	})

	Register("Howling Mine", func() Card {
		return NewArtifact("Howling Mine", "{2}",
			WithAbility(BeginningOfEachDrawStepTrigger(DrawCardsActivePlayer(Fixed(1)), false)),
		)
	})

	Register("Forcefield", func() Card {
		// Oracle: "{1}: The next time an unblocked creature of your choice
		// would deal combat damage to you this turn, prevent all but 1 of
		// that damage."
		// XXX: target should be restricted to "unblocked attacker" — using
		// IsAttacking is a coarser approximation; if the chosen attacker is
		// blocked, the shield is wasted (matches Oracle's "next time" wording
		// in spirit, since blocked attackers don't deal damage to the player).
		return NewArtifact("Forcefield", "{3}",
			WithActivatedAbility(
				ForcefieldEffect(),
				GenericCost(1),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	})

	// ===== GAUNTLET =====

	Register("Gauntlet of Might", func() Card {
		return NewArtifact("Gauntlet of Might", "{4}",
			// Red creatures get +1/+1
			WithStaticAbility(
				BoostAllCreaturesIncludingSelf(1, 1, HasColorFilter(Red)),
			),
			// Whenever a Mountain is tapped for mana, its controller adds an additional {R}
			WithAbility(NewManaBonusAbility(HasSubType("Mountain"), Red)),
		)
	})

	// ===== MISC ARTIFACTS =====

	Register("Ankh of Mishra", func() Card {
		return NewArtifact("Ankh of Mishra", "{2}",
			WithAbility(WheneverLandEntersBattlefieldTrigger(
				DealDamageToPlayers(Fixed(2), SelectEventController()), false,
			)),
		)
	})

	Register("Copper Tablet", func() Card {
		return NewArtifact("Copper Tablet", "{2}",
			WithAbility(BeginningOfEachUpkeepTrigger(DealDamageToPlayers(Fixed(1), SelectActivePlayer()), false)),
		)
	})

	// Black Vise {1}
	// Artifact
	// As this artifact enters, choose an opponent.
	// At the beginning of the chosen player's upkeep, this artifact deals X damage
	// to that player, where X is the number of cards in their hand minus 4.
	Register("Black Vise", func() Card {
		return NewArtifact("Black Vise", "{1}",
			WithAbility(ChooseOpponentOnETB()),
			WithAbility(ChosenPlayerUpkeepTrigger(BlackViseEffect(), false)),
		)
	})

	Register("Jade Monolith", func() Card {
		return NewArtifact("Jade Monolith", "{4}",
			WithActivatedAbility(
				// TODO: convert to pipeline when damage redirect step is available
				FuncEffect(
					"redirect next damage to target creature to you instead",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) < 1 {
							return nil
						}
						creatureID := targets[0]
						g.SetCreatureDamageRedirect(creatureID, controller)
						return nil
					}),
				GenericCost(1),
				WithTarget(TargetCreature()),
			),
		)
	})

	Register("Jade Statue", func() Card {
		// XXX: missing "Activate only during combat" restriction
		return NewArtifact("Jade Statue", "{4}",
			// {2}: Jade Statue becomes a 3/6 artifact creature until end of combat.
			WithActivatedAbility(
				// TODO: convert to pipeline when animate step is available
				FuncEffect("become a 3/6 artifact creature until end of combat",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						eff := TemporaryAnimateUntilEndOfCombat(sourceID, 3, 6)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				GenericCost(2),
			),
		)
	})

	Register("Glasses of Urza", func() Card {
		return NewArtifact("Glasses of Urza", "{1}",
			WithActivatedAbility(
				FuncEffect("look at target player's hand", EffectProperties{}, func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					return nil
				}),
				Tap(),
				WithTarget(TargetPlayer()),
			),
		)
	})

	Register("Helm of Chatzuk", func() Card {
		return NewArtifact("Helm of Chatzuk", "{1}",
			// {1}, {T}: Target creature gains banding until end of turn.
			WithActivatedAbility(
				GrantKeyword(Banding),
				GenericCost(1),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
		)
	})

	Register("Sunglasses of Urza", func() Card {
		return NewArtifact("Sunglasses of Urza", "{3}",
			WithStaticAbility(ManaConversion(White, Red)),
		)
	})

	Register("Kormus Bell", func() Card {
		return NewArtifact("Kormus Bell", "{4}",
			WithStaticAbility(
				AnimateLands(And(IsLand, HasSubType("Swamp")), 1, 1),
				GrantColorToAll(Black, And(IsLand, HasSubType("Swamp"))),
			),
		)
	})

	Register("Cyclopean Tomb", func() Card {
		return NewArtifact("Cyclopean Tomb", "{4}",
			WithStaticAbility(CyclopeanTombEffect()),
			// {2}, {T}: Put a mire counter on target non-Swamp land.
			WithActivatedAbility(
				AddCounters(Mire, Fixed(1)),
				GenericCost(2),
				WithCost(Tap()),
				WithTarget(TargetPermanent(IsLand, Not(HasSubType("Swamp")))),
			),
		)
	})

	Register("Illusionary Mask", func() Card {
		return NewArtifact("Illusionary Mask", "{2}",
			WithActivatedAbility(
				// TODO: convert to pipeline — complex face-down mechanic
				FuncEffect(
					"put creature from hand onto battlefield face down as 2/2",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						card, ok := p.RemoveFromHand(targets[0])
						if !ok {
							return nil
						}
						perm := g.PutOnBattlefield(card, controller)
						perm.FaceDown = true
						perm.BasePTOverride = &[2]int{2, 2}
						perm.RuntimeAbilities = nil
						return nil
					}),
				GenericCost(0),
				WithTarget(TargetCreatureInHand()),
			),
		)
	})

	Register("Nevinyrral's Disk", func() Card {
		return NewArtifact("Nevinyrral's Disk", "{4}",
			WithKeyword(EntersTapped),
			// {1}, {T}: Destroy all artifacts, creatures, and enchantments
			WithActivatedAbility(
				CompositeEffects("destroy all artifacts, creatures, and enchantments",
					DestroyAllCreatures(),
					DestroyAllEnchantments(),
					DestroyAllMatching(IsArtifact, "destroy all artifacts"),
				),
				GenericCost(1),
				WithCost(Tap()),
			),
		)
	})

	// Celestial Prism {3}
	// Artifact
	// {2}, {T}: Add one mana of any color.
	Register("Celestial Prism", func() Card {
		return NewArtifact("Celestial Prism", "{3}",
			WithActivatedAbility(
				AddAnyMana(1, Colorless),
				GenericCost(2),
				WithCost(Tap()),
			),
		)
	})

	// Conservator {4}
	// Artifact
	// {3}, {T}: Prevent the next 2 damage that would be dealt to you this turn.
	Register("Conservator", func() Card {
		return NewArtifact("Conservator", "{4}",
			WithActivatedAbility(
				AddPreventionShieldToControllerStep(2),
				GenericCost(3),
				WithCost(Tap()),
			),
		)
	})

	// ===== LUCKY CHARMS =====

	Register("Crystal Rod", func() Card {
		return NewLuckyCharm("Crystal Rod", "{1}", Blue)
	})

	Register("Iron Star", func() Card {
		return NewLuckyCharm("Iron Star", "{1}", Red)
	})

	Register("Ivory Cup", func() Card {
		return NewLuckyCharm("Ivory Cup", "{1}", White)
	})

	Register("Throne of Bone", func() Card {
		return NewLuckyCharm("Throne of Bone", "{1}", Black)
	})

	Register("Wooden Sphere", func() Card {
		return NewLuckyCharm("Wooden Sphere", "{1}", Green)
	})

	Register("Soul Net", func() Card {
		return NewArtifact("Soul Net", "{1}",
			WithAbility(AnyCreatureDiesTrigger(GainLife(1), true)),
		)
	})

	Register("Chaos Orb", func() Card {
		return NewArtifact("Chaos Orb", "{2}",
			// {1}, {T}: Destroy a random nontoken permanent you don't control, then destroy Chaos Orb.
			WithActivatedAbility(
				ChaosOrbEffect(),
				GenericCost(1),
				WithCost(Tap()),
			),
		)
	})

}
