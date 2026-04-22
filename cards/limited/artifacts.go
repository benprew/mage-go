package limited

import (
	"math/rand"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
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
			// At the beginning of your upkeep, you may pay {4}. If you do, untap Mana Vault.
			WithAbility(NewTriggered(EvtUpkeep, false, Pipeline(
				"pay {4} to untap Mana Vault",
				EffectProperties{},
				SnapshotPermanent(SelectSource, "self"),
				IfElse("pay {4} to untap",
					&TryPayManaCond{Cost: "{4}"},
					UntapGathered("self"),
					nil,
				),
			)).SetConditionData(EventPlayerIsController{})),
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
				WithCost(TapSourceCost()),
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
		// XXX: missing once-per-turn restriction on untap ability
		return NewAura("Instill Energy", "{G}",
			WithStaticAbility(
				GrantAbilityToAttached(Haste, AttachAura),
				GrantActivatedAbilityToAttached(
					UntapSource(),
					GenericCost(0),
					AttachAura,
				),
			),
		)
	})

	Register("Mana Flare", func() Card {
		return NewEnchantment("Mana Flare", "{2}{R}",
			WithAbility(NewManaFlareAbility(IsLand)),
		)
	})

	Register("Sacrifice", func() Card {
		return NewInstant("Sacrifice", "{B}",
			NewTargetedSpell(TargetCreature(), Pipeline(
				"sacrifice creature and add black mana equal to its CMC",
				EffectProperties{},
				SnapshotPermanent(SelectTarget, "t"),
				SacrificeGathered("t"),
				AddManaFromVar(Black, "t.cmc"),
			)),
		)
	})

	Register("Word of Command", func() Card {
		return NewInstant("Word of Command", "{B}{B}",
			// TODO: convert to pipeline — complex hand/cast manipulation
			NewTargetedSpell(TargetPlayer(), FuncEffect(
				"look at opponent's hand and force them to play a card",
				EffectProperties{},
				func(g *Game, _, controller uuid.UUID, targets []uuid.UUID) error {
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
			// TODO: convert to pipeline — needs PreventBlockingUntilEndOfCombat step
			NewSpellAbility(FuncEffect(
				"you assign blockers this combat",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
			// TODO: convert to pipeline — needs PreventBlockingUntilEndOfCombat step
			WithAbility(NewTriggered(EvtDeclaredAttacker, false, FuncEffect(
				"split blockers into piles",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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
				})).
				SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventPlayerIsController{},
					SourceOnBattlefield{},
					CombatGroupCountEquals{N: 1},
				}})),
		)
	})

	Register("Natural Selection", func() Card {
		return NewInstant("Natural Selection", "{G}",
			// TODO: convert to pipeline — needs library manipulation steps
			NewTargetedSpell(TargetPlayer(), FuncEffect(
				"look at top 3 cards of target player's library, rearrange them",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetPlayer := g.GetPlayer(targets[0])
					if targetPlayer == nil {
						return nil
					}
					lib := targetPlayer.Library()
					if len(lib) < 2 {
						return nil
					}
					n := 3
					if len(lib) < n {
						n = len(lib)
					}
					// TODO it's not a random shuffle, it's the controller choosing the order
					// TODO the controller may also choose to shuffle the library
					rand.Shuffle(n, func(i, j int) {
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
			// TODO: convert to pipeline — needs Lich-specific game rule steps
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"lose life equal to your life total",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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
			// TODO: convert to pipeline — needs Lich-specific game rule steps
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(FuncEffect(
				"you lose the game",
				EffectProperties{},
				func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
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
			// TODO: convert to pipeline — needs skip-draw and sanctuary game rule steps
			WithAbility(NewTriggered(EvtDrawStep, false, FuncEffect(
				"skip draw, only flying/islandwalk can attack you until your next turn",
				EffectProperties{},
				func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
					g.SetSkipNextDraw(controller)
					g.SetSanctuaryActive(controller)
					return nil
				})).SetConditionData(EventPlayerIsController{})),
		)
	})

	Register("Power Surge", func() Card {
		return NewEnchantment("Power Surge", "{R}{R}",
			// TODO need to count number of land untapped at start of turn (before untap)
			WithAbility(BeginningOfEachUpkeepTrigger(
				DealDamageToPlayers(
					CountBattlefield(SelectActivePlayer(), And(IsLand, Not(IsTapped))),
					SelectActivePlayer(),
				), false)),
		)
	})

	Register("Mana Short", func() Card {
		// TODO Also drain player's mana pool
		return NewInstant("Mana Short", "{2}{U}",
			NewTargetedSpell(TargetPlayer(), TapAllLands()),
		)
	})

	// TODO implement
	// Target player activates a mana ability of each land they control. Then that player loses all unspent mana and you add the mana lost this way.
	Register("Drain Power", func() Card {
		return NewSorcery("Drain Power", "{U}{U}",
			NewTargetedSpell(TargetPlayer(), TapAllLands()),
		)
	})

	Register("Simulacrum", func() Card {
		return NewInstant("Simulacrum", "{1}{B}",
			// TODO: convert to pipeline — needs DamageTakenByPlayer as ValueSource
			NewTargetedSpell(TargetCreatureYouControl(), FuncEffect(
				"gain life and deal damage equal to damage taken this turn",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

	// TODO implement "Also blocks if able"
	Register("Blaze of Glory", func() Card {
		return NewInstant("Blaze of Glory", "{W}",
			NewTargetedSpell(TargetCreature(), GrantKeywordUntilEndOfTurn(CanBlockAny, SelectTarget)),
		)
	})

	// TODO implement
	// "Text": "Cast this spell only during the declare blockers step.\nRemove target creature defending player controls from combat. Creatures it was blocking that had become blocked by only that creature this combat become unblocked. You may have it block an attacking creature of your choice.",
	Register("False Orders", func() Card {
		return NewInstant("False Orders", "{R}",
			NewTargetedSpell(TargetCreature(), RemoveFromCombat()),
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
			// XXX: missing "Activated abilities of white enchantments cost {3} more to activate" — no engine support for increasing activated ability costs
		)
	})

	Register("Magnetic Mountain", func() Card {
		return NewEnchantment("Magnetic Mountain", "{1}{R}{R}",
			WithStaticAbility(
				PreventUntapForMatching(And(IsCreature, HasColorFilter(Blue))),
			),
			// TODO: convert to pipeline — needs ForEach + TryPayMana + untap per creature
			WithAbility(BeginningOfEachUpkeepTrigger(
				FuncEffect("pay {4} to untap blue creatures",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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

	// TODO implement
	//   "Text": "Enchant land\nEnchanted land has indestructible and can't be enchanted by other Auras.",
	Register("Consecrate Land", func() Card {
		return NewAura("Consecrate Land", "{W}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				GrantAbilityToAttached(Indestructible, AttachAura),
			),
		)
	})

	// TODO implement
	// Fastbond's effects only apply to caster, not all players
	Register("Fastbond", func() Card {
		return NewEnchantment("Fastbond", "{G}",
			WithStaticAbility(AllowUnlimitedLandPlays()),
			WithAbility(NewTriggered(EvtLandPlayed, false,
				DealDamageToPlayers(Fixed(1), SelectController()),
			).
				SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{EventPlayerIsController{}, EventAmountGreaterThan{N: 1}}})),
		)
	})

	Register("Kudzu", func() Card {
		return NewAura("Kudzu", "{1}{G}{G}",
			WithCastTarget(TargetLand()),
			// TODO: convert to pipeline — complex attachment manipulation
			WithAbility(WhenAttachedBecomesTappedTrigger(FuncEffect(
				"destroy enchanted land; attach Kudzu to another land",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
		// XXX: missing cast timing restriction and "attack if able" forced attack effect
		return NewInstant("Siren's Call", "{U}",
			// TODO: convert to pipeline — needs delayed trigger pipeline support
			NewSpellAbility(FuncEffect(
				"destroy non-attacking non-Wall creatures at end of turn",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					active := g.ActivePlayerObj()
					activeID := active.PlayerID()
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:  EvtEndStep,
						SourceID:   sourceID,
						Controller: controller,
						Effects: []Effect{FuncEffect(
							"destroy non-attackers",
							EffectProperties{},
							func(g2 *Game, srcID, ctrlID uuid.UUID, _ []uuid.UUID) error {
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

	// Smoke {R}{R}
	// Enchantment
	// Players can't untap more than one creature during their untap steps.
	Register("Smoke", func() Card {
		return NewEnchantment("Smoke", "{R}{R}",
			WithStaticAbility(LimitCreatureUntaps(1)),
		)
	})

	// Manabarbs {3}{R}
	// Enchantment
	// Whenever a player taps a land for mana, Manabarbs deals 1 damage to that player.
	Register("Manabarbs", func() Card {
		return NewEnchantment("Manabarbs", "{3}{R}",
			WithAbility(NewTriggered(EvtTapped, false,
				DealDamageToPlayers(Fixed(1), SelectEventController()),
			).SetConditionData(EventSourceHasType{Type: TypeLand})),
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
				WithCost(TapSourceCost()),
			),
		)
	})

	// Conservator {4}
	// Artifact
	// {3}, {T}: Prevent the next 2 damage that would be dealt to you this turn.
	Register("Conservator", func() Card {
		return NewArtifact("Conservator", "{4}",
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(2)),
				GenericCost(3),
				WithCost(TapSourceCost()),
				WithTarget(TargetController()),
			),
		)
	})

	// Animate Wall {W}
	// Enchantment — Aura
	// Enchant Wall
	// Enchanted Wall can attack as though it didn't have defender.
	Register("Animate Wall", func() Card {
		return NewAura("Animate Wall", "{W}",
			WithCastTarget(TargetCreature(HasSubType("Wall"))),
			WithStaticAbility(
				GrantAbilityToAttached(AttrCanAttack, AttachAura),
			),
		)
	})
}
