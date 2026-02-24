package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
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
				NewTriggered(EvtCreatureDied, false,
					FuncEffect("destroy all creatures blocking or blocked by Abu Ja'far",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(g.CombatGroups()) == 0 {
								return nil
							}
							var toDestroy []uuid.UUID
							for _, group := range g.CombatGroups() {
								if group.AttackerID == sourceID {
									toDestroy = append(toDestroy, group.BlockerIDs...)
								}
								for _, bid := range group.BlockerIDs {
									if bid == sourceID {
										toDestroy = append(toDestroy, group.AttackerID)
									}
								}
							}
							for _, id := range toDestroy {
								p := g.FindPermanent(id)
								if p != nil {
									g.DestroyPermanent(p)
								}
							}
							return nil
						}),
				).SetCondition(IsThisSource),
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
				TapSourceCost(),
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
			WithAbility(NewTriggered(EvtLeavesBattlefield, false, SacrificeSource()).
				SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.Controller == controllerID && p.HasSubType("Island") {
							return false
						}
					}
					return true
				})),
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
			WithActivatedAbility(
				UntapSource(),
				ManaCostOf("{U}{U}{U}"),
			),
			WithStaticAbility(PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island"))),
			WithAbility(NewTriggered(EvtLeavesBattlefield, false, SacrificeSource()).
				SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.Controller == controllerID && p.HasSubType("Island") {
							return false
						}
					}
					return true
				})),
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
					FuncEffect("gain 2 life when unblocked",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p != nil {
								p.GainLife(2)
							}
							return nil
						}),
				).SetCondition(func(_ *GameEvent, g *Game, sourceID, _ uuid.UUID) bool {
					for _, group := range g.CombatGroups() {
						if group.AttackerID == sourceID && len(group.BlockerIDs) == 0 {
							return true
						}
					}
					return false
				}),
			),
			WithAbility(NewTriggered(EvtLeavesBattlefield, false, SacrificeSource()).
				SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.Controller == controllerID && p.HasSubType("Island") {
							return false
						}
					}
					return true
				})),
		)
	})

	// Oracle: "You may choose not to untap Old Man of the Sea during your untap step.
	// {T}: Gain control of target creature with power less than or equal to Old Man of
	// the Sea's power for as long as Old Man of the Sea remains tapped and that creature's
	// power remains less than or equal to Old Man of the Sea's power."
	Register("Old Man of the Sea", func() Card {
		return NewCreature("Old Man of the Sea", "{1}{U}{U}", 2, 3, WithSubTypes("Djinn"))
	})

	// Oracle: "Flying. At the beginning of your upkeep, sacrifice a land. If you sacrifice
	// an Island this way, Serendib Djinn deals 3 damage to you.
	// When you control no lands, sacrifice Serendib Djinn."
	Register("Serendib Djinn", func() Card {
		return NewCreature("Serendib Djinn", "{2}{U}{U}", 5, 6,
			WithSubTypes("Djinn"),
			WithKeyword(Flying),
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("sacrifice a land or self-destruct",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						lands := g.FilterBattlefield(And(ControlledBy(controller), IsLand))
						if len(lands) == 0 {
							// No lands: take 3 damage and sacrifice Djinn
							p := g.GetPlayer(controller)
							if p != nil {
								g.DealDamageToPlayer(p, 3, sourceID)
							}
							perm := g.FindPermanent(sourceID)
							if perm != nil {
								g.Sacrifice(perm)
							}
							return nil
						}
						// Choose a land to sacrifice
						player := g.GetPlayer(controller)
						chosen := player.ChoosePermanent(lands, "sacrifice", g)
						if chosen == nil {
							chosen = lands[0]
						}
						wasIsland := chosen.HasSubType("Island")
						g.Sacrifice(chosen)
						if wasIsland {
							if player != nil {
								g.DealDamageToPlayer(player, 3, sourceID)
							}
						}
						return nil
					}), false,
			)),
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
				FuncEffect("draw and reveal; discard if not land",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						card, ok := p.DrawCard()
						if !ok {
							return nil
						}
						if !card.HasType(TypeLand) {
							p.RemoveFromHand(card.ID())
							p.AddToGraveyard(card)
						}
						return nil
					}),
				TapSourceCost(),
			),
		)
	})

	// ===== BLACK CREATURES =====

	// Oracle: "{T}: Cuombajj Witches deals 1 damage to any target and 1 damage to any
	// target of an opponent's choice."
	Register("Cuombajj Witches", func() Card {
		return NewCreature("Cuombajj Witches", "{B}{B}", 1, 3, WithSubTypes("Human", "Wizard"))
	})

	Register("El-Hajjâj", func() Card {
		// Oracle text: "Whenever El-Hajjâj deals damage, you gain that much life."
		// Functionally equivalent to lifelink.
		return NewCreature("El-Hajjâj", "{1}{B}{B}", 1, 1,
			WithSubTypes("Human", "Wizard"),
			WithKeyword(Lifelink),
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
				).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					return evt.PlayerID == controllerID && !g.HasAttackedThisTurn(sourceID)
				}),
			),
		)
	})

	// Oracle: "As long as Guardian Beast is untapped, noncreature artifacts you control
	// can't be enchanted, they have indestructible, and other players can't gain control
	// of them."
	Register("Guardian Beast", func() Card {
		return NewCreature("Guardian Beast", "{3}{B}", 2, 4, WithSubTypes("Beast"))
	})

	// Oracle: "Whenever Hasran Ogress attacks, it deals 3 damage to you unless you pay {2}."
	Register("Hasran Ogress", func() Card {
		return NewCreature("Hasran Ogress", "{B}{B}", 3, 2,
			WithSubTypes("Ogre"),
			WithAbility(AttacksTrigger(
				FuncEffect("pay {2} or take 3 damage",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if g.TryPayCostFromLands(controller, "{2}") {
							return nil
						}
						p := g.GetPlayer(controller)
						if p != nil {
							g.DealDamageToPlayer(p, 3, sourceID)
						}
						return nil
					}), false,
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
					FuncEffect("put +1/+1 counters for deaths",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							count := g.CreatureDeaths()
							if count > 0 {
								perm := g.FindPermanent(sourceID)
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
				SetPTUntilEndOfTurn(0, 2, SelectTarget),
				TapSourceCost(),
				WithTarget(TargetCreature()),
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
		return NewCreature("Aladdin", "{2}{R}{R}", 1, 1, WithSubTypes("Human", "Rogue"))
	})

	// Oracle: "{R}: Tap target Wall."
	Register("Ali Baba", func() Card {
		return NewCreature("Ali Baba", "{R}", 1, 1,
			WithSubTypes("Human", "Rogue"),
			WithActivatedAbility(
				TapTarget(),
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
					g.Effects.SetMinimumLife(perm.Controller)
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
				GrantKeywordUntilEndOfTurn(CantRegenerate, SelectTarget),
				TapSourceCost(),
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
				FuncEffect("flip coin or remove from combat",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if !g.FlipCoin(controller) {
							g.RemoveFromCombat(sourceID)
							perm := g.FindPermanent(sourceID)
							if perm != nil {
								perm.Tapped = true
							}
						}
						return nil
					}), false,
			)),
		)
	})

	// Oracle: "When Rukh Egg dies, create a 4/4 red Bird creature token with flying
	// at the beginning of the next end step."
	Register("Rukh Egg", func() Card {
		return NewCreature("Rukh Egg", "{3}{R}", 0, 3,
			WithSubTypes("Bird", "Egg"),
			WithAbility(
				NewTriggered(EvtCreatureDied, false,
					CreateToken("Bird", 4, 4, []CardType{TypeCreature}, []string{"Bird"}, Flying),
				).SetCondition(IsThisSource),
			),
		)
	})

	// Oracle: "Whenever Ydwen Efreet blocks, flip a coin. If you lose the flip,
	// remove Ydwen Efreet from combat and it can't block this turn."
	Register("Ydwen Efreet", func() Card {
		return NewCreature("Ydwen Efreet", "{R}{R}{R}", 3, 6,
			WithSubTypes("Efreet"),
			WithAbility(BlocksTrigger(
				FuncEffect("flip coin or remove from combat",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if !g.FlipCoin(controller) {
							g.RemoveFromCombat(sourceID)
						}
						return nil
					}), false,
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
				FuncEffect("grant forestwalk to opponent creature",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Find non-Wall creatures opponents control
						candidates := g.FilterBattlefield(And(
							NotControlledBy(controller),
							IsCreature,
							Not(HasSubType("Wall")),
						))
						if len(candidates) == 0 {
							return nil
						}
						// Pick one (first candidate — AI/test deterministic)
						target := candidates[0]
						eff := TargetEffect(LayerAbility, UntilYourNextTurn, target.ID(),
							func(g *Game, target *Permanent) error {
								g.Effects.GrantAttr(target.ID(), Forestwalk)
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
			// Static continuous effect at LayerControl: each EM cycle, set controller
			// to the player with the most life (if strictly more than all others).
			WithStaticAbility(FuncContinuousEffect(LayerControl, WhileOnBattlefield,
				func(g *Game, sourceID uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					players := g.Players
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
						perm.Controller = maxPlayer.PlayerID()
					}
					return nil
				})),
		)
	})

	// Oracle: "Flying. {G}: Ifh-Bíff Efreet deals 1 damage to each creature with flying
	// and each player. Any player may activate this ability."
	Register("Ifh-Bíff Efreet", func() Card {
		return NewCreature("Ifh-Bíff Efreet", "{2}{G}{G}", 3, 3, WithSubTypes("Efreet"))
	})

	// Oracle: "Whenever Nafs Asp deals damage to a player, that player loses 1 life at
	// the beginning of their next draw step unless they pay {1} before that draw step."
	Register("Nafs Asp", func() Card {
		return NewCreature("Nafs Asp", "{G}", 1, 1,
			WithSubTypes("Snake"),
			WithAbility(DealsDamageToOpponentTrigger(
				FuncEffect("register delayed draw-step penalty",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
								func(g GameMutator, sourceID2, controller2 uuid.UUID, _ []uuid.UUID) error {
									p := g.GetPlayer(damagedPlayerID)
									if p == nil {
										return nil
									}
									if !g.TryPayCostFromLands(damagedPlayerID, "{1}") {
										g.DealDamageToPlayer(p, 1, sourceID2)
									}
									return nil
								})},
						})
						return nil
					}), false)),
		)
	})

	// Oracle: "{T}: Target attacking creature has base power 0 until end of turn."
	Register("Singing Tree", func() Card {
		return NewCreature("Singing Tree", "{3}{G}", 0, 3,
			WithSubTypes("Plant"),
			WithActivatedAbility(
				SetPowerUntilEndOfTurn(0, SelectTarget),
				TapSourceCost(),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	})

	// Oracle: "{T}: Target creature gets +1/+1 until end of turn."
	Register("Wyluli Wolf", func() Card {
		return NewCreature("Wyluli Wolf", "{1}{G}", 1, 1,
			WithSubTypes("Wolf"),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(1), SelectTarget),
				TapSourceCost(),
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
				FuncEffect("pay {1} to untap", EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if g.TryPayCostFromLands(controller, "{1}") {
							perm := g.FindPermanent(sourceID)
							if perm != nil {
								perm.Tapped = false
							}
						}
						return nil
					}), false,
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
