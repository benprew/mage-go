package limited

import (
	"math/rand"

	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
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
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	Register("Sol Ring", func() Card {
		return NewArtifact("Sol Ring", "{1}",
			// {T}: Add {C}{C}
			WithActivatedAbility(
				AddMana(Colorless, 2),
				TapSourceCost(),
			),
		)
	})

	Register("Basalt Monolith", func() Card {
		return NewArtifact("Basalt Monolith", "{3}",
			WithKeyword(DoesNotUntapKW),
			// {T}: Add {C}{C}{C}
			WithActivatedAbility(
				AddMana(Colorless, 3),
				TapSourceCost(),
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
				TapSourceCost(),
			),
			// At the beginning of your upkeep, deal 1 damage to you
			WithAbility(BeginningOfUpkeepTrigger(DealDamageToPlayers(Fixed(1), SelectController()), false)),
		)
	})

	// ===== UTILITY ARTIFACTS =====

	Register("Jayemdae Tome", func() Card {
		return NewArtifact("Jayemdae Tome", "{4}",
			// {4}, {T}: Draw a card
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				GenericCost(4),
				WithCost(TapSourceCost()),
			),
		)
	})

	Register("Disrupting Scepter", func() Card {
		return NewArtifact("Disrupting Scepter", "{3}",
			// {3}, {T}: Target player discards a card
			WithActivatedAbility(
				DiscardCards(Fixed(1)),
				GenericCost(3),
				WithCost(TapSourceCost()),
				WithTarget(TargetPlayer()),
			),
		)
	})

	Register("Icy Manipulator", func() Card {
		return NewArtifact("Icy Manipulator", "{4}",
			// {1}, {T}: Tap target permanent
			WithActivatedAbility(
				TapTarget(),
				GenericCost(1),
				WithCost(TapSourceCost()),
				WithTarget(TargetPermanent()),
			),
		)
	})

	Register("Rod of Ruin", func() Card {
		return NewArtifact("Rod of Ruin", "{4}",
			// {3}, {T}: Deal 1 damage to any target
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				GenericCost(3),
				WithCost(TapSourceCost()),
				WithTarget(TargetAnyTarget()),
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
				WithCost(TapSourceCost()),
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
		return NewArtifact("Forcefield", "{3}",
			WithActivatedAbility(
				ForcefieldEffect(),
				GenericCost(1),
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

	// ===== STEAL ARTIFACT =====

	Register("Steal Artifact", func() Card {
		return NewAura("Steal Artifact", "{2}{U}{U}",
			WithCastTarget(TargetArtifact()),
			WithStaticAbility(ControlChangeContinuous()),
		)
	})

	Register("Copy Artifact", func() Card {
		return NewArtifact("Copy Artifact", "{1}{U}",
			WithAbility(NewTargetedSpell(TargetArtifact(), CloneTarget(TypeEnchantment))),
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

	Register("Jade Monolith", func() Card {
		return NewArtifact("Jade Monolith", "{4}",
			WithActivatedAbility(
				FuncEffect(
					"redirect next damage to target creature to target player instead",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) < 2 {
							return nil
						}
						creatureID := targets[0]
						playerID := targets[1]
						g.SetCreatureDamageRedirect(creatureID, playerID)
						return nil
					}),
				GenericCost(1),
				WithTarget(TargetCreature()),
				WithTarget(TargetPlayer()),
			),
		)
	})

	Register("Jade Statue", func() Card {
		return NewArtifact("Jade Statue", "{4}",
			// {2}: Jade Statue becomes a 3/6 artifact creature until end of combat.
			WithActivatedAbility(
				FuncEffect("become a 3/6 artifact creature until end of combat",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
				FuncEffect("look at target player's hand", EffectProperties{}, func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					return nil
				}),
				TapSourceCost(),
				WithTarget(TargetPlayer()),
			),
		)
	})

	Register("Helm of Chatzuk", func() Card {
		return NewArtifact("Helm of Chatzuk", "{1}",
			// {1}, {T}: Target creature gains banding until end of turn.
			WithActivatedAbility(
				GrantKeywordUntilEndOfTurn(Banding, SelectTarget),
				GenericCost(1),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	Register("Sunglasses of Urza", func() Card {
		return NewArtifact("Sunglasses of Urza", "{3}",
			WithStaticAbility(ManaConversion(Red, White)),
		)
	})

	Register("Kormus Bell", func() Card {
		return NewArtifact("Kormus Bell", "{4}",
			WithStaticAbility(
				AnimateLands(And(IsLand, HasSubType("Swamp")), 1, 1),
			),
		)
	})

	Register("Cyclopean Tomb", func() Card {
		return NewArtifact("Cyclopean Tomb", "{4}",
			WithStaticAbility(CyclopeanTombEffect()),
			// {2}, {T}: Put a mire counter on target non-Swamp land.
			WithActivatedAbility(
				AddCounters(Mire, Fixed(1), SelectTarget),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithTarget(TargetPermanent(IsLand, Not(HasSubType("Swamp")))),
			),
		)
	})

	Register("Illusionary Mask", func() Card {
		return NewArtifact("Illusionary Mask", "{2}",
			WithActivatedAbility(
				FuncEffect(
					"put creature from hand onto battlefield face down as 0/1",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
				WithCost(TapSourceCost()),
			),
		)
	})

	// ===== DEATHGRIP / LIFEFORCE =====

	Register("Deathgrip", func() Card {
		return NewEnchantment("Deathgrip", "{B}{B}",
			WithActivatedAbility(
				CounterSpellIfColor(Green),
				ManaCostOf("{B}{B}"),
				WithTarget(TargetSpellOnStack()),
			),
		)
	})

	Register("Lifeforce", func() Card {
		return NewEnchantment("Lifeforce", "{G}{G}",
			WithActivatedAbility(
				CounterSpellIfColor(Black),
				ManaCostOf("{G}{G}"),
				WithTarget(TargetSpellOnStack()),
			),
		)
	})

	// ===== LIVING LANDS / INSTILL ENERGY =====

	Register("Living Lands", func() Card {
		return NewEnchantment("Living Lands", "{3}{G}",
			WithStaticAbility(
				AnimateLands(And(IsLand, HasSubType("Forest")), 1, 1),
			),
		)
	})

	Register("Instill Energy", func() Card {
		return NewAura("Instill Energy", "{G}",
			WithStaticAbility(
				GrantAbilityToAttached(Haste, AttachAura),
			),
		)
	})

	Register("Mana Flare", func() Card {
		return NewEnchantment("Mana Flare", "{2}{R}",
			WithAbility(NewManaBonusAbility(HasSubType("Plains"), White)),
			WithAbility(NewManaBonusAbility(HasSubType("Island"), Blue)),
			WithAbility(NewManaBonusAbility(HasSubType("Swamp"), Black)),
			WithAbility(NewManaBonusAbility(HasSubType("Mountain"), Red)),
			WithAbility(NewManaBonusAbility(HasSubType("Forest"), Green)),
		)
	})

	Register("Sacrifice", func() Card {
		return NewInstant("Sacrifice", "{B}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"sacrifice creature and add black mana equal to its CMC",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
						p.ManaPool().Add(Black, cmc)
					}
					return nil
				})),
		)
	})

	Register("Word of Command", func() Card {
		return NewInstant("Word of Command", "{B}{B}",
			NewTargetedSpell(TargetPlayer(), FuncEffect(
				"look at opponent's hand and force them to play a card",
				EffectProperties{},
				func(g GameMutator, _, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetPlayer := g.GetPlayer(targets[0])
					if targetPlayer == nil {
						return nil
					}
					hand := targetPlayer.Hand()
					for _, card := range hand {
						targetPlayer.ManaPool().Add(Red, 10)
						targetPlayer.ManaPool().Add(Blue, 10)
						targetPlayer.ManaPool().Add(Black, 10)
						targetPlayer.ManaPool().Add(White, 10)
						targetPlayer.ManaPool().Add(Green, 10)
						targetPlayer.ManaPool().Add(Colorless, 10)
						autoTargets := []uuid.UUID{controller}
						err := g.CastSpellByName(targetPlayer.PlayerID(), card.Name(), autoTargets)
						if err == nil {
							return nil
						}
					}
					return nil
				})),
		)
	})

	Register("Camouflage", func() Card {
		return NewInstant("Camouflage", "{G}",
			NewSpellAbility(FuncEffect(
				"you assign blockers this combat",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.Controller != controller && p.HasType(TypeCreature) {
							eff := PreventBlockingUntilEndOfCombat(p.ID())
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
						}
					}
					g.ApplyContinuousEffects()
					return nil
				})),
		)
	})

	Register("Raging River", func() Card {
		return NewEnchantment("Raging River", "{R}{R}",
		WithAbility(NewTriggered(EvtDeclaredAttacker, false, FuncEffect(
			"split blockers into piles",
			EffectProperties{},
			func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				var nonFlyers []*Permanent
				for _, p := range g.FilterBattlefield(AnyPermanent) {
					if p.Controller != controller && p.HasType(TypeCreature) &&
						!p.HasKeyword(Flying) {
						nonFlyers = append(nonFlyers, p)
					}
				}
				for _, p := range nonFlyers {
					eff := PreventBlockingUntilEndOfCombat(p.ID())
					eff.SetSourceID(sourceID)
					g.AddContinuousEffect(eff)
				}
				g.ApplyContinuousEffects()
				return nil
			})).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
			return evt.PlayerID == controllerID && g.FindPermanent(sourceID) != nil && len(g.CombatGroups()) == 1
		})),
		)
	})

	Register("Natural Selection", func() Card {
		return NewInstant("Natural Selection", "{G}",
			NewTargetedSpell(TargetPlayer(), FuncEffect(
				"look at top 3 cards of target player's library and shuffle",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
				})),
		)
	})

	Register("Lich", func() Card {
		return NewEnchantment("Lich", "{B}{B}{B}{B}",
			// ETB: lose life equal to your life total
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"lose life equal to your life total",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					life := p.Life()
					if life > 0 {
						p.LoseLife(life)
					}
					g.SetLichActive(controller, sourceID)
					return nil
				}), false)),
			// When Lich is put into a graveyard from the battlefield, you lose the game.
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(FuncEffect(
				"you lose the game",
				EffectProperties{},
				func(g GameMutator, _, controller uuid.UUID, _ []uuid.UUID) error {
					g.ClearLich(controller)
					p := g.GetPlayer(controller)
					if p != nil {
						p.LoseLife(9999)
					}
					return nil
				}), false)),
		)
	})

	Register("Island Sanctuary", func() Card {
		return NewEnchantment("Island Sanctuary", "{1}{W}",
			WithAbility(NewTriggered(EvtDrawStep, false, FuncEffect(
				"skip draw, only flying/islandwalk can attack you until your next turn",
				EffectProperties{},
				func(g GameMutator, _, controller uuid.UUID, _ []uuid.UUID) error {
					g.SetSkipNextDraw(controller)
					g.SetSanctuaryActive(controller)
					return nil
				})).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
				src := g.FindPermanent(sourceID)
				return src != nil && evt.PlayerID == controllerID
			})),
		)
	})

	Register("Power Surge", func() Card {
		return NewEnchantment("Power Surge", "{R}{R}",
			WithAbility(BeginningOfEachUpkeepTrigger(
				DealDamageToPlayers(
					CountBattlefield(SelectActivePlayer(), And(IsLand, Not(IsTapped))),
					SelectActivePlayer(),
				), false)),
		)
	})

	Register("Mana Short", func() Card {
		return NewInstant("Mana Short", "{2}{U}",
			NewTargetedSpell(TargetPlayer(), TapAllLands()),
		)
	})

	Register("Drain Power", func() Card {
		return NewSorcery("Drain Power", "{U}{U}",
			NewTargetedSpell(TargetPlayer(), TapAllLands()),
		)
	})

	Register("Simulacrum", func() Card {
		return NewInstant("Simulacrum", "{1}{B}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"gain life and deal damage equal to damage taken this turn",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					dmg := g.DamageTakenByPlayer(controller)
					if dmg > 0 {
						p := g.GetPlayer(controller)
						if p != nil {
							p.GainLife(dmg)
							g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: controller, Amount: dmg})
						}
						if len(targets) > 0 {
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.DealDamageToPermanent(perm, dmg, sourceID)
							}
						}
					}
					return nil
				})),
		)
	})

	Register("Blaze of Glory", func() Card {
		return NewInstant("Blaze of Glory", "{W}",
			NewTargetedSpell(TargetCreature(), GrantKeywordUntilEndOfTurn(CanBlockAny, SelectTarget)),
		)
	})

	Register("False Orders", func() Card {
		return NewInstant("False Orders", "{R}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"remove target creature from combat",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					g.RemoveFromCombat(targets[0])
					return nil
				})),
		)
	})

	Register("Lifetap", func() Card {
		return NewEnchantment("Lifetap", "{U}{U}",
			WithAbility(WhenOpponentPermanentBecomesTappedTrigger(
				GainLife(1), false,
				And(IsLand, HasSubType("Forest")),
			)),
		)
	})

	Register("Conversion", func() Card {
		return NewEnchantment("Conversion", "{2}{W}{W}",
			WithStaticAbility(
				ChangeSubTypesForAll([]string{"Mountain"}, []string{"Plains"}),
			),
			WithAbility(SacrificeAtUpkeepUnlessPay("{W}{W}")),
		)
	})

	Register("Gloom", func() Card {
		return NewEnchantment("Gloom", "{2}{B}",
			WithStaticAbility(
				IncreaseSpellCostForColor(White, 3),
			),
		)
	})

	Register("Magnetic Mountain", func() Card {
		return NewEnchantment("Magnetic Mountain", "{1}{R}{R}",
			WithStaticAbility(
				PreventUntapForMatching(And(IsCreature, HasColorFilter(Blue))),
			),
			WithAbility(BeginningOfEachUpkeepTrigger(
				FuncEffect("pay {4} to untap blue creatures",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Find the active player (whose upkeep it is)
						activePlayer := g.ActivePlayerObj().PlayerID()
						blues := g.FilterBattlefield(And(IsCreature, HasColorFilter(Blue), ControlledBy(activePlayer), IsTapped))
						for _, blue := range blues {
							if g.TryPayCostFromLands(activePlayer, "{4}") {
								blue.Tapped = false
							}
						}
						return nil
					}), false,
			)),
		)
	})

	Register("Consecrate Land", func() Card {
		return NewAura("Consecrate Land", "{W}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				GrantAbilityToAttached(Indestructible, AttachAura),
			),
		)
	})

	Register("Fastbond", func() Card {
		return NewEnchantment("Fastbond", "{G}",
			WithStaticAbility(AllowUnlimitedLandPlays()),
			WithAbility(NewTriggered(EvtLandPlayed, false,
				DealDamageToPlayers(Fixed(1), SelectController()),
			).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
				return evt.PlayerID == controllerID && evt.Amount > 1
			})),
		)
	})

	Register("Kudzu", func() Card {
		return NewAura("Kudzu", "{1}{G}{G}",
			WithCastTarget(TargetLand()),
			WithAbility(WhenAttachedBecomesTappedTrigger(FuncEffect(
				"destroy enchanted land; attach Kudzu to another land",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.HasType(TypeLand) && p.ID() != attachedID {
							g.Attach(sourceID, p.ID())
							return nil
						}
					}
					return nil
				}), false)),
		)
	})

	Register("Regeneration", func() Card {
		return NewAura("Regeneration", "{1}{G}",
			WithStaticAbility(
				GrantActivatedAbilityToAttached(
					RegenerateSource(),
					ManaCostOf("{G}"),
					AttachAura,
				),
			),
		)
	})

	Register("Siren's Call", func() Card {
		return NewInstant("Siren's Call", "{U}",
			NewSpellAbility(FuncEffect(
				"destroy non-attacking non-Wall creatures at end of turn",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					active := g.ActivePlayerObj()
					activeID := active.PlayerID()
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:  EvtEndStep,
						SourceID:   sourceID,
						Controller: controller,
						Effects: []Effect{FuncEffect(
							"destroy non-attackers",
							EffectProperties{},
							func(g2 GameMutator, srcID, ctrlID uuid.UUID, _ []uuid.UUID) error {
								var toDestroy []*Permanent
								for _, p := range g2.FilterBattlefield(AnyPermanent) {
									if p.Controller == activeID && p.HasType(TypeCreature) &&
										!p.HasSubType("Wall") && !g2.HasAttackedThisTurn(p.ID()) {
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
				})),
		)
	})

	Register("Magical Hack", func() Card {
		return NewInstant("Magical Hack", "{U}",
			NewTargetedSpell(TargetPermanent(), ReplaceKeywordEffect(Swampwalk, Forestwalk)),
		)
	})
}
