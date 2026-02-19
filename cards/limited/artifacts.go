package limited

import (
	"math/rand"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// ===== MOX CYCLE =====

	moxen := []struct {
		name  string
		color core.Color
	}{
		{"Mox Pearl", core.White},
		{"Mox Sapphire", core.Blue},
		{"Mox Jet", core.Black},
		{"Mox Ruby", core.Red},
		{"Mox Emerald", core.Green},
	}

	for _, m := range moxen {
		name := m.name
		color := m.color
		mage.Register(name, func() mage.Card {
			return mage.NewArtifact(name, "{0}", mage.WithManaAbility(color))
		})
	}

	// ===== MANA ARTIFACTS =====

	mage.Register("Black Lotus", func() mage.Card {
		return mage.NewArtifact("Black Lotus", "{0}",
			// {T}, Sacrifice: Add 3 mana of any one color
			mage.WithAbility(mage.NewActivatedAbility(
				mage.AddAnyMana(3, core.Green),
				mage.TapSourceCost(),
				mage.WithCost(mage.SacrificeSourceCost()),
			)),
		)
	})

	mage.Register("Sol Ring", func() mage.Card {
		return mage.NewArtifact("Sol Ring", "{1}",
			// {T}: Add {C}{C}
			mage.WithAbility(mage.NewActivatedAbility(
				mage.AddMana(core.Colorless, 2),
				mage.TapSourceCost(),
			)),
		)
	})

	mage.Register("Basalt Monolith", func() mage.Card {
		return mage.NewArtifact("Basalt Monolith", "{3}",
			mage.WithKeyword(core.DoesNotUntapKW),
			// {T}: Add {C}{C}{C}
			mage.WithAbility(mage.NewActivatedAbility(
				mage.AddMana(core.Colorless, 3),
				mage.TapSourceCost(),
			)),
			// {3}: Untap Basalt Monolith
			mage.WithAbility(mage.NewActivatedAbility(
				mage.UntapSource(),
				mage.GenericCost(3),
			)),
		)
	})

	mage.Register("Mana Vault", func() mage.Card {
		return mage.NewArtifact("Mana Vault", "{1}",
			mage.WithKeyword(core.DoesNotUntapKW),
			// {T}: Add {C}{C}{C}
			mage.WithAbility(mage.NewActivatedAbility(
				mage.AddMana(core.Colorless, 3),
				mage.TapSourceCost(),
			)),
			// At the beginning of your upkeep, deal 1 damage to you
			mage.WithAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController()), false)),
		)
	})

	// ===== UTILITY ARTIFACTS =====

	mage.Register("Jayemdae Tome", func() mage.Card {
		return mage.NewArtifact("Jayemdae Tome", "{4}",
			// {4}, {T}: Draw a card
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DrawCards(mage.Fixed(1)),
				mage.GenericCost(4),
				mage.WithCost(mage.TapSourceCost()),
			)),
		)
	})

	mage.Register("Disrupting Scepter", func() mage.Card {
		return mage.NewArtifact("Disrupting Scepter", "{3}",
			// {3}, {T}: Target player discards a card
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DiscardCards(mage.Fixed(1)),
				mage.GenericCost(3),
				mage.WithCost(mage.TapSourceCost()),
				mage.WithTarget(mage.TargetPlayer()),
			)),
		)
	})

	mage.Register("Icy Manipulator", func() mage.Card {
		return mage.NewArtifact("Icy Manipulator", "{4}",
			// {1}, {T}: Tap target permanent
			mage.WithAbility(mage.NewActivatedAbility(
				mage.TapTarget(),
				mage.GenericCost(1),
				mage.WithCost(mage.TapSourceCost()),
				mage.WithTarget(mage.TargetPermanent()),
			)),
		)
	})

	mage.Register("Rod of Ruin", func() mage.Card {
		return mage.NewArtifact("Rod of Ruin", "{4}",
			// {3}, {T}: Deal 1 damage to any target
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DealDamage(mage.Fixed(1)),
				mage.GenericCost(3),
				mage.WithCost(mage.TapSourceCost()),
				mage.WithTarget(mage.TargetAnyTarget()),
			)),
		)
	})

	mage.Register("The Hive", func() mage.Card {
		return mage.NewArtifact("The Hive", "{5}",
			// {5}, {T}: Create a 1/1 colorless Insect artifact creature token with flying named Wasp.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.CreateToken("Wasp", 1, 1,
					[]core.CardType{core.TypeArtifact, core.TypeCreature},
					[]string{"Insect"},
					core.Flying,
				),
				mage.GenericCost(5),
				mage.WithCost(mage.TapSourceCost()),
			)),
		)
	})

	mage.Register("Winter Orb", func() mage.Card {
		return mage.NewArtifact("Winter Orb", "{2}",
			mage.WithAbility(mage.StaticAbility(mage.LimitLandUntaps(1))),
		)
	})

	mage.Register("Meekstone", func() mage.Card {
		return mage.NewArtifact("Meekstone", "{1}",
			mage.WithAbility(mage.StaticAbility(
				mage.PreventUntapForMatching(mage.And(mage.IsCreature, mage.HasPowerGTE(3))),
			)),
		)
	})

	mage.Register("Howling Mine", func() mage.Card {
		return mage.NewArtifact("Howling Mine", "{2}",
			mage.WithAbility(mage.BeginningOfEachDrawStepTrigger(mage.DrawCardsActivePlayer(mage.Fixed(1)), false)),
		)
	})

	mage.Register("Forcefield", func() mage.Card {
		return mage.NewArtifact("Forcefield", "{3}",
			mage.WithAbility(mage.NewActivatedAbility(
				mage.ForcefieldEffect(),
				mage.GenericCost(1),
			)),
		)
	})

	// ===== GAUNTLET =====

	mage.Register("Gauntlet of Might", func() mage.Card {
		return mage.NewArtifact("Gauntlet of Might", "{4}",
			// Red creatures get +1/+1
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAllCreaturesIncludingSelf(1, 1, mage.HasColorFilter(core.Red)),
			)),
			// Whenever a Mountain is tapped for mana, its controller adds an additional {R}
			mage.WithAbility(mage.NewManaBonusAbility(mage.HasSubType("Mountain"), core.Red)),
		)
	})

	// ===== STEAL ARTIFACT =====

	mage.Register("Steal Artifact", func() mage.Card {
		return mage.NewAura("Steal Artifact", "{2}{U}{U}",
			mage.WithAbility(mage.StaticAbility(mage.ControlChangeContinuous())),
		)
	})

	mage.Register("Copy Artifact", func() mage.Card {
		return mage.NewArtifact("Copy Artifact", "{1}{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetArtifact(), mage.CloneTarget(core.TypeEnchantment))),
		)
	})

	// ===== MISC ARTIFACTS =====

	mage.Register("Ankh of Mishra", func() mage.Card {
		return mage.NewArtifact("Ankh of Mishra", "{2}",
			mage.WithAbility(mage.WheneverLandEntersBattlefieldTrigger(
				mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectEventController()), false,
			)),
		)
	})

	mage.Register("Jade Monolith", func() mage.Card {
		return mage.NewArtifact("Jade Monolith", "{4}",
			mage.WithAbility(mage.NewActivatedAbility(
				mage.FuncEffect(
					"redirect next damage to target creature to target player instead",
					func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) < 2 {
							return nil
						}
						creatureID := targets[0]
						playerID := targets[1]
						g.Effects.SetCreatureDamageRedirect(creatureID, playerID)
						return nil
					}),
				mage.GenericCost(1),
				mage.WithTarget(mage.TargetCreature()),
				mage.WithTarget(mage.TargetPlayer()),
			)),
		)
	})

	mage.Register("Jade Statue", func() mage.Card {
		return mage.NewArtifact("Jade Statue", "{4}",
			// {2}: Jade Statue becomes a 3/6 artifact creature until end of combat.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.FuncEffect("become a 3/6 artifact creature until end of combat",
					func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						eff := mage.TemporaryAnimateUntilEndOfCombat(sourceID, 3, 6)
						eff.SetSourceID(sourceID)
						g.Effects.Add(eff)
						g.Effects.Apply(g)
						return nil
					}),
				mage.GenericCost(2),
			)),
		)
	})

	mage.Register("Glasses of Urza", func() mage.Card {
		return mage.NewArtifact("Glasses of Urza", "{1}",
			mage.WithAbility(mage.NewActivatedAbility(
				mage.FuncEffect("look at target player's hand", func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					return nil
				}),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetPlayer()),
			)),
		)
	})

	mage.Register("Helm of Chatzuk", func() mage.Card {
		return mage.NewArtifact("Helm of Chatzuk", "{1}",
			// {1}, {T}: Target creature gains banding until end of turn.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.GrantKeywordUntilEndOfTurn(core.Banding, mage.SelectTarget),
				mage.GenericCost(1),
				mage.WithCost(mage.TapSourceCost()),
				mage.WithTarget(mage.TargetCreature()),
			)),
		)
	})

	mage.Register("Sunglasses of Urza", func() mage.Card {
		return mage.NewArtifact("Sunglasses of Urza", "{3}",
			mage.WithAbility(mage.StaticAbility(mage.ManaConversion(core.Red, core.White))),
		)
	})

	mage.Register("Kormus Bell", func() mage.Card {
		return mage.NewArtifact("Kormus Bell", "{4}",
			mage.WithAbility(mage.StaticAbility(
				mage.AnimateLands(mage.And(mage.IsLand, mage.HasSubType("Swamp")), 1, 1),
			)),
		)
	})

	mage.Register("Cyclopean Tomb", func() mage.Card {
		return mage.NewArtifact("Cyclopean Tomb", "{4}",
			mage.WithAbility(mage.StaticAbility(mage.CyclopeanTombEffect())),
			// {2}, {T}: Put a mire counter on target non-Swamp land.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.AddCounters(core.Mire, mage.Fixed(1), mage.SelectTarget),
				mage.GenericCost(2),
				mage.WithCost(mage.TapSourceCost()),
				mage.WithTarget(mage.TargetPermanent(mage.IsLand, mage.Not(mage.HasSubType("Swamp")))),
			)),
		)
	})

	mage.Register("Illusionary Mask", func() mage.Card {
		return mage.NewArtifact("Illusionary Mask", "{2}",
			mage.WithAbility(mage.NewActivatedAbility(
				mage.FuncEffect(
					"put creature from hand onto battlefield face down as 0/1",
					func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
						perm.BasePTOverride = &[2]int{0, 1}
						perm.RuntimeAbilities = nil
						return nil
					}),
				mage.GenericCost(0),
				mage.WithTarget(mage.TargetCreatureInHand()),
			)),
		)
	})

	mage.Register("Nevinyrral's Disk", func() mage.Card {
		return mage.NewArtifact("Nevinyrral's Disk", "{4}",
			mage.WithKeyword(core.EntersTapped),
			// {1}, {T}: Destroy all artifacts, creatures, and enchantments
			mage.WithAbility(mage.NewActivatedAbility(
				mage.CompositeEffects("destroy all artifacts, creatures, and enchantments",
					mage.DestroyAllCreatures(),
					mage.DestroyAllEnchantments(),
					mage.DestroyAllMatching(mage.IsArtifact, "destroy all artifacts"),
				),
				mage.GenericCost(1),
				mage.WithCost(mage.TapSourceCost()),
			)),
		)
	})

	// ===== DEATHGRIP / LIFEFORCE =====

	mage.Register("Deathgrip", func() mage.Card {
		return mage.NewEnchantment("Deathgrip", "{B}{B}",
			mage.WithAbility(mage.NewActivatedAbility(
				mage.CounterSpellIfColor(core.Green),
				mage.ManaCostOf("{B}{B}"),
				mage.WithTarget(mage.TargetSpellOnStack()),
			)),
		)
	})

	mage.Register("Lifeforce", func() mage.Card {
		return mage.NewEnchantment("Lifeforce", "{G}{G}",
			mage.WithAbility(mage.NewActivatedAbility(
				mage.CounterSpellIfColor(core.Black),
				mage.ManaCostOf("{G}{G}"),
				mage.WithTarget(mage.TargetSpellOnStack()),
			)),
		)
	})

	// ===== LIVING LANDS / INSTILL ENERGY =====

	mage.Register("Living Lands", func() mage.Card {
		return mage.NewEnchantment("Living Lands", "{3}{G}",
			mage.WithAbility(mage.StaticAbility(
				mage.AnimateLands(mage.And(mage.IsLand, mage.HasSubType("Forest")), 1, 1),
			)),
		)
	})

	mage.Register("Instill Energy", func() mage.Card {
		return mage.NewAura("Instill Energy", "{G}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.Haste, core.AttachAura),
			)),
		)
	})

	mage.Register("Mana Flare", func() mage.Card {
		return mage.NewEnchantment("Mana Flare", "{2}{R}",
			mage.WithAbility(mage.NewManaBonusAbility(mage.HasSubType("Plains"), core.White)),
			mage.WithAbility(mage.NewManaBonusAbility(mage.HasSubType("Island"), core.Blue)),
			mage.WithAbility(mage.NewManaBonusAbility(mage.HasSubType("Swamp"), core.Black)),
			mage.WithAbility(mage.NewManaBonusAbility(mage.HasSubType("Mountain"), core.Red)),
			mage.WithAbility(mage.NewManaBonusAbility(mage.HasSubType("Forest"), core.Green)),
		)
	})

	mage.Register("Sacrifice", func() mage.Card {
		return mage.NewInstant("Sacrifice", "{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
				"sacrifice creature and add black mana equal to its CMC",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					cmc := perm.Card.ManaCost().CMC()
					g.Sacrifice(perm)
					p := g.GetPlayer(controller)
					if p != nil {
						p.ManaPool().Add(core.Black, cmc)
					}
					return nil
				}))),
		)
	})

	mage.Register("Word of Command", func() mage.Card {
		return mage.NewInstant("Word of Command", "{B}{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.FuncEffect(
				"look at opponent's hand and force them to play a card",
				func(g *mage.Game, _, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetPlayer := g.GetPlayer(targets[0])
					if targetPlayer == nil {
						return nil
					}
					hand := targetPlayer.Hand()
					for _, card := range hand {
						targetPlayer.ManaPool().Add(core.Red, 10)
						targetPlayer.ManaPool().Add(core.Blue, 10)
						targetPlayer.ManaPool().Add(core.Black, 10)
						targetPlayer.ManaPool().Add(core.White, 10)
						targetPlayer.ManaPool().Add(core.Green, 10)
						targetPlayer.ManaPool().Add(core.Colorless, 10)
						autoTargets := []uuid.UUID{controller}
						err := g.CastSpellByName(targetPlayer.PlayerID(), card.Name(), autoTargets)
						if err == nil {
							return nil
						}
					}
					return nil
				}))),
		)
	})

	mage.Register("Camouflage", func() mage.Card {
		return mage.NewInstant("Camouflage", "{G}",
			mage.WithAbility(mage.NewSpellAbility(mage.FuncEffect(
				"you assign blockers this combat",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					for _, p := range g.Battlefield {
						if p.Controller != controller && p.HasType(core.TypeCreature) {
							g.Effects.PreventFromBlocking(p.ID())
						}
					}
					return nil
				}))),
		)
	})

	mage.Register("Raging River", func() mage.Card {
		return mage.NewEnchantment("Raging River", "{R}{R}",
		mage.WithAbility(mage.NewTriggered(core.EvtDeclaredAttacker, false, mage.FuncEffect(
			"split blockers into piles",
			func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
				var nonFlyers []*mage.Permanent
				for _, p := range g.Battlefield {
					if p.Controller != controller && p.HasType(core.TypeCreature) &&
						!p.HasKeyword(core.Flying) {
						nonFlyers = append(nonFlyers, p)
					}
				}
				for _, p := range nonFlyers {
					g.Effects.PreventFromBlocking(p.ID())
				}
				return nil
			})).SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
			return evt.PlayerID == controllerID && g.FindPermanent(sourceID) != nil && len(g.Combat.Groups) == 1
		})),
		)
	})

	mage.Register("Natural Selection", func() mage.Card {
		return mage.NewInstant("Natural Selection", "{G}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.FuncEffect(
				"look at top 3 cards of target player's library and shuffle",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetPlayer := g.GetPlayer(targets[0])
					if targetPlayer == nil {
						return nil
					}
					lib := targetPlayer.Library()
					rand.Shuffle(len(lib), func(i, j int) {
						lib[i], lib[j] = lib[j], lib[i]
					})
					targetPlayer.SetLibrary(lib)
					return nil
				}))),
		)
	})

	mage.Register("Lich", func() mage.Card {
		return mage.NewEnchantment("Lich", "{B}{B}{B}{B}",
			// ETB: lose life equal to your life total
			mage.WithAbility(mage.EntersBattlefieldTrigger(mage.FuncEffect(
				"lose life equal to your life total",
				func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					life := p.Life()
					if life > 0 {
						p.LoseLife(life)
					}
					g.Effects.SetLichActive(controller)
					return nil
				}), false)),
			// When Lich is put into a graveyard from the battlefield, you lose the game.
			mage.WithAbility(mage.PutIntoGraveyardFromBattlefieldTrigger(mage.FuncEffect(
				"you lose the game",
				func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
					g.Effects.ClearLich(controller)
					p := g.GetPlayer(controller)
					if p != nil {
						p.LoseLife(9999)
					}
					return nil
				}), false)),
		)
	})

	mage.Register("Island Sanctuary", func() mage.Card {
		return mage.NewEnchantment("Island Sanctuary", "{1}{W}",
			mage.WithAbility(mage.NewTriggered(core.EvtDrawStep, false, mage.FuncEffect(
				"skip draw, only flying/islandwalk can attack you until your next turn",
				func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
					g.Effects.SetSkipNextDraw(controller)
					g.Effects.SetSanctuaryActive(controller)
					return nil
				})).SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
				src := g.FindPermanent(sourceID)
				return src != nil && evt.PlayerID == controllerID
			})),
		)
	})

	mage.Register("Power Surge", func() mage.Card {
		return mage.NewEnchantment("Power Surge", "{R}{R}",
			mage.WithAbility(mage.BeginningOfEachUpkeepTrigger(mage.FuncEffect(
				"deal damage equal to untapped lands",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					active := g.ActivePlayerObj()
					activeID := active.PlayerID()
					untapped := 0
					for _, p := range g.Battlefield {
						if p.Controller == activeID && p.HasType(core.TypeLand) && !p.Tapped {
							untapped++
						}
					}
					if untapped > 0 {
						g.DealDamageToPlayer(active, untapped, sourceID)
					}
					return nil
				}), false)),
		)
	})

	mage.Register("Mana Short", func() mage.Card {
		return mage.NewInstant("Mana Short", "{2}{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.TapAllLands())),
		)
	})

	mage.Register("Drain Power", func() mage.Card {
		return mage.NewSorcery("Drain Power", "{U}{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPlayer(), mage.TapAllLands())),
		)
	})

	mage.Register("Simulacrum", func() mage.Card {
		return mage.NewInstant("Simulacrum", "{1}{B}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
				"gain life and deal damage equal to damage taken this turn",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					dmg := g.DamageTakenThisTurn[controller]
					if dmg > 0 {
						p := g.GetPlayer(controller)
						if p != nil {
							p.GainLife(dmg)
							g.FireEvent(core.GameEvent{Type: core.EvtLifeGained, PlayerID: controller, Amount: dmg})
						}
						if len(targets) > 0 {
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.DealDamageToPermanent(perm, dmg, sourceID)
							}
						}
					}
					return nil
				}))),
		)
	})

	mage.Register("Blaze of Glory", func() mage.Card {
		return mage.NewInstant("Blaze of Glory", "{W}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.GrantKeywordUntilEndOfTurn(core.CanBlockAny, mage.SelectTarget))),
		)
	})

	mage.Register("False Orders", func() mage.Card {
		return mage.NewInstant("False Orders", "{R}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
				"remove target creature from combat",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					g.Combat.RemoveFromCombat(targets[0])
					return nil
				}))),
		)
	})

	mage.Register("Lifetap", func() mage.Card {
		return mage.NewEnchantment("Lifetap", "{U}{U}",
			mage.WithAbility(mage.WhenOpponentPermanentBecomesTappedTrigger(
				mage.GainLife(1), false,
				mage.And(mage.IsLand, mage.HasSubType("Forest")),
			)),
		)
	})

	mage.Register("Conversion", func() mage.Card {
		return mage.NewEnchantment("Conversion", "{2}{W}{W}",
			mage.WithAbility(mage.StaticAbility(
				mage.ChangeSubTypesForAll([]string{"Mountain"}, []string{"Plains"}),
			)),
			mage.WithAbility(mage.SacrificeAtUpkeepUnlessPay("{W}{W}")),
		)
	})

	mage.Register("Gloom", func() mage.Card {
		return mage.NewEnchantment("Gloom", "{2}{B}",
			mage.WithAbility(mage.StaticAbility(
				mage.IncreaseSpellCostForColor(core.White, 3),
			)),
		)
	})

	mage.Register("Magnetic Mountain", func() mage.Card {
		return mage.NewEnchantment("Magnetic Mountain", "{1}{R}{R}",
			mage.WithAbility(mage.StaticAbility(
				mage.PreventUntapForMatching(mage.And(mage.IsCreature, mage.HasColorFilter(core.Blue))),
			)),
		)
	})

	mage.Register("Consecrate Land", func() mage.Card {
		return mage.NewAura("Consecrate Land", "{W}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.Indestructible, core.AttachAura),
			)),
		)
	})

	mage.Register("Fastbond", func() mage.Card {
		return mage.NewEnchantment("Fastbond", "{G}",
			mage.WithAbility(mage.StaticAbility(mage.AllowUnlimitedLandPlays())),
			mage.WithAbility(mage.NewTriggered(core.EvtLandPlayed, false,
				mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController()),
			).SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
				return evt.PlayerID == controllerID && evt.Amount > 1
			})),
		)
	})

	mage.Register("Kudzu", func() mage.Card {
		return mage.NewAura("Kudzu", "{1}{G}{G}",
			mage.WithAbility(mage.WhenAttachedBecomesTappedTrigger(mage.FuncEffect(
				"destroy enchanted land; attach Kudzu to another land",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					kudzu := g.FindPermanent(sourceID)
					if kudzu == nil {
						return nil
					}
					attached := g.FindPermanent(kudzu.AttachedTo)
					if attached == nil {
						return nil
					}
					attachedID := attached.ID()
					kudzu.AttachedTo = uuid.Nil
					filtered := attached.Attachments[:0]
					for _, id := range attached.Attachments {
						if id != sourceID {
							filtered = append(filtered, id)
						}
					}
					attached.Attachments = filtered
					g.DestroyPermanent(attached)
					for _, p := range g.Battlefield {
						if p.HasType(core.TypeLand) && p.ID() != attachedID {
							g.Attach(sourceID, p.ID())
							return nil
						}
					}
					return nil
				}), false)),
		)
	})

	mage.Register("Regeneration", func() mage.Card {
		return mage.NewAura("Regeneration", "{1}{G}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantActivatedAbilityToAttached(
					mage.RegenerateSource(),
					mage.ManaCostOf("{G}"),
					core.AttachAura,
				),
			)),
		)
	})

	mage.Register("Siren's Call", func() mage.Card {
		return mage.NewInstant("Siren's Call", "{U}",
			mage.WithAbility(mage.NewSpellAbility(mage.FuncEffect(
				"destroy non-attacking non-Wall creatures at end of turn",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					active := g.ActivePlayerObj()
					activeID := active.PlayerID()
					g.RegisterDelayedTrigger(&mage.DelayedTrigger{
						EventType:  core.EvtEndStep,
						SourceID:   sourceID,
						Controller: controller,
						Effects: []mage.Effect{mage.FuncEffect(
							"destroy non-attackers",
							func(g2 *mage.Game, srcID, ctrlID uuid.UUID, _ []uuid.UUID) error {
								var toDestroy []*mage.Permanent
								for _, p := range g2.Battlefield {
									if p.Controller == activeID && p.HasType(core.TypeCreature) &&
										!p.HasSubType("Wall") && !g2.AttackedThisTurn[p.ID()] {
										toDestroy = append(toDestroy, p)
									}
								}
								for _, p := range toDestroy {
									g2.DestroyPermanent(p)
								}
								return nil
							}),
						},
					})
					return nil
				}))),
		)
	})

	mage.Register("Magical Hack", func() mage.Card {
		return mage.NewInstant("Magical Hack", "{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetPermanent(), mage.ReplaceKeywordEffect(core.Swampwalk, core.Forestwalk))),
		)
	})
}
