package arabian

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerCreatures()
}

func registerCreatures() {
	// ===== WHITE CREATURES =====

	// Oracle: "When Abu Ja'far dies, destroy all creatures blocking or blocked by it.
	// They can't be regenerated."
	Register("Abu Ja'far", func() Card {
		return NewCreature("Abu Ja'far", "{W}", 0, 1,
			WithSubTypes("Human"),
			WithAbility(
				DiesTrigger(
					ForEachCombatOpponent(
						DestroyTargetNoRegenStep(),
						"destroy all creatures blocking or blocked by Abu Ja'far",
					),
					false,
				),
			),
		)
	})

	// Oracle: "Banding. As long as Camel is attacking, prevent all damage Deserts
	// would deal to Camel and to creatures banded with Camel."
	Register("Camel", func() Card {
		return NewCreature("Camel", "{W}", 0, 1,
			WithSubTypes("Camel"),
			WithKeyword(Banding),
			WithStaticAbility(PreventDamageFromTo(
				HasSubType("Desert"),
				func(sourceID uuid.UUID) PermanentFilter {
					return Or(IsID(sourceID), IsBandedWith(sourceID))
				},
				WhileSourceAttacking,
			)),
		)
	})

	// Oracle: "{T}: Destroy target Djinn or Efreet."
	Register("King Suleiman", func() Card {
		return NewCreature("King Suleiman", "{1}{W}", 1, 1,
			WithSubTypes("Human", "Noble"),
			WithActivatedAbility(
				DestroyTarget(),
				Tap(),
				WithTarget(TargetCreature(
					Or(HasSubType("Djinn"), HasSubType("Efreet")),
				)),
			),
		)
	})

	// Oracle: "Trample"
	Register("Moorish Cavalry", func() Card {
		return NewCreature("Moorish Cavalry", "{2}{W}{W}", 3, 3,
			WithSubTypes("Human", "Knight"),
			WithKeyword(Trample),
		)
	})

	// Oracle: "Protection from red"
	Register("Repentant Blacksmith", func() Card {
		return NewCreature("Repentant Blacksmith", "{1}{W}", 1, 2,
			WithSubTypes("Human"),
			WithAbility(ProtectionFromColor(Red)),
		)
	})

	// Oracle: "Trample; banding"
	Register("War Elephant", func() Card {
		return NewCreature("War Elephant", "{3}{W}", 2, 2,
			WithSubTypes("Elephant"),
			WithKeyword(Trample),
			WithKeyword(Banding),
		)
	})

	// ===== BLUE CREATURES =====

	// Oracle: "Dandân can't attack unless defending player controls an Island.
	// When you control no Islands, sacrifice Dandân."
	Register("Dandân", func() Card {
		return NewCreature("Dandân", "{U}{U}", 4, 1,
			WithSubTypes("Fish"),
			WithStaticAbility(PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island"))),
			WithAbility(NewTriggered(EvtZoneChange, false, SacrificeSource()).AndConditionData(EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneAny}).
				AndConditionData(ControllerHasNoPermanentMatching{Filter: HasSubType("Island")})),
		)
	})

	// Oracle: "Flying"
	Register("Flying Men", func() Card {
		return NewCreature("Flying Men", "{U}", 1, 1,
			WithSubTypes("Human"),
			WithKeyword(Flying),
		)
	})

	// Oracle: "Giant Tortoise gets +0/+3 as long as it's untapped."
	Register("Giant Tortoise", func() Card {
		return NewCreature("Giant Tortoise", "{1}{U}", 1, 1,
			WithSubTypes("Turtle"),
			WithStaticAbility(BoostSelf(0, 3, WhileSourceUntapped)),
		)
	})

	// Oracle: "Island Fish Jasconius doesn't untap during your untap step.
	// At the beginning of your upkeep, you may pay {U}{U}{U}. If you do, untap it.
	// It can't attack unless defending player controls an Island.
	// When you control no Islands, sacrifice it."
	Register("Island Fish Jasconius", func() Card {
		return NewCreature("Island Fish Jasconius", "{4}{U}{U}{U}", 6, 8,
			WithSubTypes("Fish"),
			WithKeyword(DoesNotUntapKW),
			// "At the beginning of your upkeep, you may pay {U}{U}{U}. If you do, untap it."
			WithAbility(BeginningOfUpkeepTrigger(
				IfElse("Only if tapped",
					SourceIsTapped{},
					EffectIfPaid(
						ManaCostOf("{U}{U}{U}"),
						UntapSource()),
					nil,
				), false,
			)),
			WithStaticAbility(PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island"))),
			WithAbility(NewTriggered(EvtZoneChange, false, SacrificeSource()).AndConditionData(EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneAny}).
				AndConditionData(ControllerHasNoPermanentMatching{Filter: HasSubType("Island")})),
		)
	})

	// Oracle: "Merchant Ship can't attack unless defending player controls an Island.
	// Whenever Merchant Ship attacks and isn't blocked, you gain 2 life.
	// When you control no Islands, sacrifice Merchant Ship."
	Register("Merchant Ship", func() Card {
		return NewCreature("Merchant Ship", "{U}", 0, 2,
			WithSubTypes("Human"),
			WithStaticAbility(PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island"))),
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					GainLife(2),
				).SetConditionData(SourceIsUnblockedAttacker{}),
			),
			WithAbility(NewTriggered(EvtZoneChange, false, SacrificeSource()).AndConditionData(EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneAny}).
				AndConditionData(ControllerHasNoPermanentMatching{Filter: HasSubType("Island")})),
		)
	})

	// Oracle: "You may choose not to untap Old Man of the Sea during your untap step.
	// {T}: Gain control of target creature with power less than or equal to Old Man of
	// the Sea's power for as long as Old Man of the Sea remains tapped and that creature's
	// power remains less than or equal to Old Man of the Sea's power."
	Register("Old Man of the Sea", func() Card {
		return NewCreature("Old Man of the Sea", "{1}{U}{U}", 2, 3,
			WithSubTypes("Djinn"),
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				GainControl().While(
					ControlSourceControlledByEffectController{},
					ControlSourceTapped{},
					ControlTargetPowerLESource{},
				).TapMaintained(),
				Tap(),
				WithTarget(TargetCreatureWithPowerLESource()),
			),
		)
	})

	// Oracle: "Flying. At the beginning of your upkeep, sacrifice a land. If you sacrifice
	// an Island this way, Serendib Djinn deals 3 damage to you.
	// When you control no lands, sacrifice Serendib Djinn."
	Register("Serendib Djinn", func() Card {
		return NewCreature("Serendib Djinn", "{2}{U}{U}", 5, 6,
			WithSubTypes("Djinn"),
			WithKeyword(Flying),
			// Upkeep: sacrifice a land, 3 damage if Island; if no lands, sacrifice self
			WithAbility(BeginningOfUpkeepTrigger(
				// TODO: convert to pipeline — needs sacrifice-chosen-land + conditional damage based on subtype
				FuncEffect("sacrifice a land",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						lands := g.FilterBattlefield(And(ControlledBy(controller), IsLand))
						if len(lands) == 0 {
							// "When you control no lands, sacrifice this creature."
							perm := g.FindPermanent(sourceID)
							if perm != nil {
								g.DoSacrifice(perm)
							}
							return nil
						}
						player := g.GetPlayer(controller)
						chosen := player.ChoosePermanent(lands, "sacrifice", g)
						if chosen == nil {
							chosen = lands[0]
						}
						wasIsland := chosen.HasSubType("Island")
						g.DoSacrifice(chosen)
						if wasIsland {
							if player != nil {
								g.DealDamageToPlayer(player, 3, sourceID)
							}
						}
						return nil
					}), false,
			)),
			// "When you control no lands, sacrifice Serendib Djinn."
			WithAbility(NewStateTriggered(false, SacrificeSource()).
				SetConditionData(ControllerHasNoPermanentMatching{Filter: IsLand})),
		)
	})

	// Oracle: "Flying. At the beginning of your upkeep, Serendib Efreet deals 1 damage to you."
	Register("Serendib Efreet", func() Card {
		return NewCreature("Serendib Efreet", "{2}{U}", 3, 4,
			WithSubTypes("Efreet"),
			WithKeyword(Flying),
			WithAbility(BeginningOfUpkeepTrigger(
				DealDamageToPlayers(Fixed(1), SelectController()), false,
			)),
		)
	})

	// Oracle: "{T}: Draw a card and reveal it. If it isn't a land card, discard it."
	Register("Sindbad", func() Card {
		return NewCreature("Sindbad", "{1}{U}", 1, 1,
			WithSubTypes("Human"),
			WithActivatedAbility(
				SindbadEffect(),
				Tap(),
			),
		)
	})

	// ===== BLACK CREATURES =====

	// Oracle: "{T}: Cuombajj Witches deals 1 damage to any target and 1 damage to any
	// target of an opponent's choice."
	Register("Cuombajj Witches", func() Card {
		return NewCreature("Cuombajj Witches", "{B}{B}", 1, 3,
			WithSubTypes("Human", "Wizard"),
			WithActivatedAbility(
				// TODO: convert to pipeline — needs opponent-chooses-target primitive
				FuncEffect("deal 1 to target, 1 to opponent's choice",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						// Deal 1 damage to controller's chosen target
						if len(targets) > 0 {
							targetID := targets[0]
							dealt := false
							for _, pl := range g.AllPlayers() {
								if pl.PlayerID() == targetID {
									g.DealDamageToPlayer(pl, 1, sourceID)
									dealt = true
									break
								}
							}
							if !dealt {
								perm := g.FindPermanent(targetID)
								if perm != nil {
									g.DealDamageToPermanent(perm, 1, sourceID)
								}
							}
						}
						// Opponent chooses any target for the second 1 damage
						opp := g.GetOpponent(controller)
						if opp == nil {
							return nil
						}
						// Opponent can choose any creature or player as second target
						creatures := g.FilterBattlefield(IsCreature)
						if len(creatures) > 0 {
							chosen := opp.ChoosePermanent(creatures, "Cuombajj Witches", g)
							if chosen != nil {
								g.DealDamageToPermanent(chosen, 1, sourceID)
							} else {
								// Opponent declined creature — deal to opponent themselves
								g.DealDamageToPlayer(opp, 1, sourceID)
							}
						} else {
							// No creatures — deal to opponent
							g.DealDamageToPlayer(opp, 1, sourceID)
						}
						return nil
					}),
				Tap(),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	// Oracle: "Whenever El-Hajjâj deals damage, you gain that much life."
	Register("El-Hajjâj", func() Card {
		return NewCreature("El-Hajjâj", "{1}{B}{B}", 1, 1,
			WithSubTypes("Human", "Wizard"),
			WithAbility(NewTriggered(EvtDamageDealt, false,
				GainLifeAmount(EventAmountValue()),
			).SetConditionData(EventSourceIsSelf{})),
		)
	})

	// Oracle: "At the beginning of your end step, if Erg Raiders didn't attack this turn,
	// Erg Raiders deals 2 damage to you unless it came under your control this turn."
	Register("Erg Raiders", func() Card {
		return NewCreature("Erg Raiders", "{1}{B}", 2, 3,
			WithSubTypes("Human", "Warrior"),
			WithAbility(
				NewTriggered(EvtEndStep, false,
					DealDamageToPlayers(Fixed(2), SelectController()),
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventPlayerIsController{},
					NotTriggerCond{Inner: HasAttackedThisTurnCond{}},
					SourceNotSummonSick{},
				}}),
			),
		)
	})

	// Oracle: "As long as Guardian Beast is untapped, noncreature artifacts you control
	// can't be enchanted, they have indestructible, and other players can't gain control
	// of them."
	Register("Guardian Beast", func() Card {
		return NewCreature("Guardian Beast", "{3}{B}", 2, 4,
			WithSubTypes("Beast"),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
				func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					// Grant indestructible, can't be enchanted, and can't change control
					// to noncreature artifacts owned by the same player.
					// Uses Owner (not Controller) because a stealing effect at LayerControl
					// may have already changed Controller before this LayerAbility runs.
					// The post-layer AttrCantChangeControl check reverts any such steal.
					for _, p := range g.AllBattlefield() {
						if p.Card.Owner() == src.ControllerID() &&
							p.HasType(TypeArtifact) && !p.HasType(TypeCreature) &&
							p.ID() != sourceID {
							g.GrantAttr(p.ID(), Indestructible)
							g.GrantAttr(p.ID(), AttrCantBeEnchanted)
							g.GrantAttr(p.ID(), AttrCantChangeControl)
						}
					}
					return nil
				}, SourceUntapped)),
		)
	})

	// Oracle: "Whenever Hasran Ogress attacks, it deals 3 damage to you unless you pay {2}."
	Register("Hasran Ogress", func() Card {
		return NewCreature("Hasran Ogress", "{B}{B}", 3, 2,
			WithSubTypes("Ogre"),
			WithAbility(AttacksTrigger(
				DamageUnlessPayMana("{2}", 3), false,
			)),
		)
	})

	// Oracle: "Flying. At the beginning of your upkeep, sacrifice Junún Efreet unless you pay {B}{B}."
	Register("Junún Efreet", func() Card {
		return NewCreature("Junún Efreet", "{1}{B}{B}", 3, 3,
			WithSubTypes("Efreet"),
			WithKeyword(Flying),
			WithAbility(SacrificeAtUpkeepUnlessPay("{B}{B}")),
		)
	})

	// Oracle: "At the beginning of your upkeep, Juzám Djinn deals 1 damage to you."
	Register("Juzám Djinn", func() Card {
		return NewCreature("Juzám Djinn", "{2}{B}{B}", 5, 5,
			WithSubTypes("Djinn"),
			WithAbility(BeginningOfUpkeepTrigger(
				DealDamageToPlayers(Fixed(1), SelectController()), false,
			)),
		)
	})

	// Oracle: "At the beginning of each end step, put a +1/+1 counter on Khabál Ghoul
	// for each creature that died this turn."
	Register("Khabál Ghoul", func() Card {
		return NewCreature("Khabál Ghoul", "{2}{B}", 1, 1,
			WithSubTypes("Zombie"),
			WithAbility(
				NewTriggered(EvtEndStep, false,
					// TODO: convert to pipeline — needs CreatureDeaths() access primitive
					FuncEffect("put +1/+1 counters for deaths",
						EffectProperties{},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							count := g.CreatureDeaths()
							if count > 0 {
								perm := g.MutablePermanent(sourceID)
								if perm != nil {
									perm.AddCounter(P1P1, count)
								}
							}
							return nil
						}),
				),
			),
		)
	})

	// Oracle: "{T}: Target creature other than Sorceress Queen has base power and
	// toughness 0/2 until end of turn."
	Register("Sorceress Queen", func() Card {
		return NewCreature("Sorceress Queen", "{1}{B}{B}", 1, 1,
			WithSubTypes("Human", "Wizard", "Sorcerer"),
			WithActivatedAbility(
				SorceressQueenEffect(),
				Tap(),
				WithTarget(TargetOtherCreature()),
			),
		)
	})

	// Oracle: "First strike"
	Register("Stone-Throwing Devils", func() Card {
		return NewCreature("Stone-Throwing Devils", "{B}", 1, 1,
			WithSubTypes("Devil"),
			WithKeyword(FirstStrike),
		)
	})

	// ===== RED CREATURES =====

	// Oracle: "{1}{R}{R}, {T}: Gain control of target artifact for as long as you
	// control Aladdin."
	Register("Aladdin", func() Card {
		return NewCreature("Aladdin", "{2}{R}{R}", 1, 1,
			WithSubTypes("Human", "Rogue"),
			WithActivatedAbility(
				GainControl().While(ControlSourceControlledByEffectController{}),
				Tap(),
				WithCost(ManaCostOf("{1}{R}{R}")),
				WithTarget(TargetArtifact()),
			),
		)
	})

	// Oracle: "{R}: Tap target Wall."
	Register("Ali Baba", func() Card {
		return NewCreature("Ali Baba", "{R}", 1, 1,
			WithSubTypes("Human", "Rogue"),
			WithActivatedAbility(
				Tap(),
				ManaCostOf("{R}"),
				WithTarget(TargetCreature(HasSubType("Wall"))),
			),
		)
	})

	// Oracle: "Damage that would reduce your life total to less than 1 reduces it to 1 instead."
	Register("Ali from Cairo", func() Card {
		return NewCreature("Ali from Cairo", "{2}{R}{R}", 0, 1,
			WithSubTypes("Human"),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
				func(g *Game, sourceID uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					g.SetMinimumLife(perm.ControllerID())
					return nil
				})),
		)
	})

	// Oracle: "Flying"
	Register("Bird Maiden", func() Card {
		return NewCreature("Bird Maiden", "{2}{R}", 1, 2,
			WithSubTypes("Human", "Bird"),
			WithKeyword(Flying),
		)
	})

	// Oracle: "Desertwalk. Prevent all damage that would be dealt to Desert Nomads by Deserts."
	Register("Desert Nomads", func() Card {
		return NewCreature("Desert Nomads", "{2}{R}", 2, 2,
			WithSubTypes("Human", "Nomad"),
			WithKeyword(Desertwalk),
			WithStaticAbility(PreventDamageFromTo(
				HasSubType("Desert"),
				func(sourceID uuid.UUID) PermanentFilter {
					return IsID(sourceID)
				},
			)),
		)
	})

	// Oracle: "{T}: Target creature can't be regenerated this turn."
	Register("Hurr Jackal", func() Card {
		return NewCreature("Hurr Jackal", "{R}", 1, 1,
			WithSubTypes("Jackal"),
			WithActivatedAbility(
				HurrJackalEffect(),
				Tap(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Oracle: "Kird Ape gets +1/+2 as long as you control a Forest."
	Register("Kird Ape", func() Card {
		return NewCreature("Kird Ape", "{R}", 1, 1,
			WithSubTypes("Ape"),
			WithStaticAbility(BoostSelf(1, 2, WhileControlling(HasSubType("Forest")))),
		)
	})

	// Oracle: "Whenever Mijae Djinn attacks, flip a coin. If you lose the flip,
	// remove Mijae Djinn from combat and tap it."
	Register("Mijae Djinn", func() Card {
		return NewCreature("Mijae Djinn", "{R}{R}{R}", 6, 3,
			WithSubTypes("Djinn"),
			WithAbility(AttacksTrigger(
				Pipeline("flip coin or remove from combat",
					EffectProperties{},
					SnapshotPermanent(SelectSource, "self"),
					IfElse("remove from combat and tap if lost",
						NotTriggerCond{Inner: FlipCoinCond{}},
						&PipelineData{Steps: []Effect{
							RemoveFromCombatGathered("self"),
							TapGathered("self"),
						}},
						nil,
					),
				), false,
			)),
		)
	})

	// Oracle: "When Rukh Egg dies, create a 4/4 red Bird creature token with flying
	// at the beginning of the next end step."
	Register("Rukh Egg", func() Card {
		return NewCreature("Rukh Egg", "{3}{R}", 0, 3,
			WithSubTypes("Bird", "Egg"),
			WithAbility(
				DiesTrigger(
					RegisterDelayedTriggerStep(EvtEndStep, "",
						CreateColoredToken("Bird", 4, 4, []Color{Red}, []CardType{TypeCreature}, []string{"Bird"}, Flying),
					),
					false,
				),
			),
		)
	})

	// Oracle: "Whenever Ydwen Efreet blocks, flip a coin. If you lose the flip,
	// remove Ydwen Efreet from combat and it can't block this turn."
	Register("Ydwen Efreet", func() Card {
		return NewCreature("Ydwen Efreet", "{R}{R}{R}", 3, 6,
			WithSubTypes("Efreet"),
			WithAbility(BlocksTrigger(
				Pipeline("flip a coin; if you lose, remove from combat and it can't block this turn",
					EffectProperties{},
					SnapshotPermanent(SelectSource, "self"),
					IfElse("on lost flip, remove from combat and revoke can-block",
						FlipCoinCond{},
						nil,
						Pipeline("remove from combat and revoke can-block",
							EffectProperties{},
							RemoveFromCombatGathered("self"),
							RevokeKeyword(AttrCanBlock).Targeting(ToSource()),
						),
					),
				), false,
			)),
		)
	})

	// ===== GREEN CREATURES =====

	// Oracle: "At the beginning of your upkeep, target non-Wall creature an opponent
	// controls gains forestwalk until your next upkeep."
	Register("Erhnam Djinn", func() Card {
		return NewCreature("Erhnam Djinn", "{3}{G}", 4, 5,
			WithSubTypes("Djinn"),
			WithAbility(BeginningOfUpkeepTrigger(
				// TODO: convert to pipeline — needs grant-keyword-until-your-next-turn + choose-opponent-creature primitives
				FuncEffect("grant forestwalk to opponent creature",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Find non-Wall creatures opponents control
						candidates := g.FilterBattlefield(And(
							NotControlledBy(controller),
							IsCreature,
							Not(HasSubType("Wall")),
						))
						if len(candidates) == 0 {
							return nil
						}
						// Controller chooses target
						p := g.GetPlayer(controller)
						var target *Permanent
						if p != nil {
							target = p.ChoosePermanent(candidates, "Erhnam Djinn", g)
						}
						if target == nil {
							target = candidates[0]
						}
						eff := TargetEffect(LayerAbility, UntilYourNextTurn, target.ID(),
							func(g *Game, target *Permanent) error {
								g.GrantAttr(target.ID(), Forestwalk)
								return nil
							})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}), false,
			)),
		)
	})

	// Oracle: "At the beginning of your upkeep, if a player has more life than each
	// other player, the player with the most life gains control of Ghazbán Ogre."
	Register("Ghazbán Ogre", func() Card {
		return NewCreature("Ghazbán Ogre", "{G}", 2, 2,
			WithSubTypes("Ogre"),
			// Upkeep trigger: determine who should control Ghazbán Ogre based on life totals.
			// Stores result in ChosenPlayer so the continuous effect can persist it.
			WithAbility(BeginningOfUpkeepTrigger(
				// TODO: convert to pipeline — needs life total comparison + ChosenPlayer assignment primitives
				FuncEffect("check life totals for control change",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.MutablePermanent(sourceID)
						if perm == nil {
							return nil
						}
						players := g.AllPlayers()
						if len(players) < 2 {
							return nil
						}
						var maxLife int
						var maxPlayer Player
						tied := false
						for _, p := range players {
							if maxPlayer == nil || p.Life() > maxLife {
								maxLife = p.Life()
								maxPlayer = p
								tied = false
							} else if p.Life() == maxLife {
								tied = true
							}
						}
						if !tied && maxPlayer != nil {
							perm.ChosenPlayer = maxPlayer.PlayerID()
						}
						return nil
					}), false,
			)),
			WithStaticAbility(ControlSourceByChosenPlayer()),
		)
	})

	// Oracle: "Flying. {G}: Ifh-Bíff Efreet deals 1 damage to each creature with flying
	// and each player. Any player may activate this ability."
	Register("Ifh-Bíff Efreet", func() Card {
		return NewCreature("Ifh-Bíff Efreet", "{2}{G}{G}", 3, 3,
			WithSubTypes("Efreet"),
			WithKeyword(Flying),
			WithActivatedAbility(
				CompositeEffects(
					"Ifh-Bíff Efreet deals 1 damage to each creature with flying and each player",
					DealDamageToAllCreatures(Fixed(1), HasKeywordFilter(Flying)),
					DealDamageToPlayers(Fixed(1), SelectEachPlayer()),
				),
				ManaCostOf("{G}"),
				WithAnyPlayerMay(),
			),
		)
	})

	// Oracle: "Whenever Nafs Asp deals damage to a player, that player loses 1 life at
	// the beginning of their next draw step unless they pay {1} before that draw step."
	Register("Nafs Asp", func() Card {
		return NewCreature("Nafs Asp", "{G}", 1, 1,
			WithSubTypes("Snake"),
			WithAbility(NewTriggered(EvtDamageDealt, false,
				// TODO: convert to pipeline — needs RegisterDelayedTrigger primitive
				FuncEffect("register delayed draw-step penalty",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						// targets[0] is the damaged player (from EvtDamageDealt.TargetID)
						if len(targets) == 0 {
							return nil
						}
						damagedPlayerID := targets[0]
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:     EvtDrawStep,
							MatchPlayerID: damagedPlayerID,
							SourceID:      sourceID,
							Controller:    controller,
							Effects: []Effect{FuncEffect("lose 1 life unless pay {1}",
								EffectProperties{Outcome: OutcomeDetriment},
								func(g *Game, sourceID2, controller2 uuid.UUID, _ []uuid.UUID) error {
									p := g.GetPlayer(damagedPlayerID)
									if p == nil {
										return nil
									}
									if !g.TryPayMana(damagedPlayerID, "{1}") {
										p.LoseLife(1)
									}
									return nil
								})},
						})
						return nil
					}),
			).
				SetConditionData(EventSourceIsSelfDamageToPlayer{})),
		)
	})

	// Oracle: "{T}: Target attacking creature has base power 0 until end of turn."
	Register("Singing Tree", func() Card {
		return NewCreature("Singing Tree", "{3}{G}", 0, 3,
			WithSubTypes("Plant"),
			WithActivatedAbility(
				SetPowerUntilEndOfTurn(0, SelectTarget),
				Tap(),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	})

	// Oracle: "{T}: Target creature gets +1/+1 until end of turn."
	Register("Wyluli Wolf", func() Card {
		return NewCreature("Wyluli Wolf", "{1}{G}", 1, 1,
			WithSubTypes("Wolf"),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)),
				Tap(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// ===== ARTIFACT CREATURES =====

	// Oracle: "Brass Man doesn't untap during your untap step. At the beginning of your
	// upkeep, you may pay {1}. If you do, untap Brass Man."
	Register("Brass Man", func() Card {
		return NewCreature("Brass Man", "{1}", 1, 3,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithKeyword(DoesNotUntapKW),
			WithAbility(BeginningOfUpkeepTrigger(
				IfElse("Only if tapped",
					SourceIsTapped{},
					EffectIfPaid(
						ManaCostOf("{1}"),
						UntapSource()),
					nil,
				), false,
			)),
		)
	})

	// Oracle: "Flying"
	Register("Dancing Scimitar", func() Card {
		return NewCreature("Dancing Scimitar", "{4}", 1, 5,
			WithSubTypes("Spirit"),
			WithCardType(TypeArtifact),
			WithKeyword(Flying),
		)
	})
}
