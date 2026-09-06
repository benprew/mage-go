package thedark

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// Blood Moon {2}{R}
	// Enchantment
	// Nonbasic lands are Mountains.
	Register("Blood Moon", func() Card {
		return NewEnchantment("Blood Moon", "{2}{R}",
			WithStaticAbility(
				BecomesBasicLandsEffect(
					NewPermanentFilter("nonbasic lands", func(permanent *Permanent, _ *Game) bool {
						return permanent.HasType(TypeLand) && !permanent.Card.HasSuperType(SuperBasic)
					}),
					"Mountain",
				),
			),
		)
	})

	// Curse Artifact {2}{B}{B}
	// Enchantment — Aura
	// Enchant artifact
	// At the beginning of the upkeep of enchanted artifact's controller, this Aura deals 2 damage to that player unless they sacrifice that artifact.
	Register("Curse Artifact", func() Card {
		return NewAura("Curse Artifact", "{2}{B}{B}",
			WithCastTarget(TargetArtifact()),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				FuncEffect(
					"deals 2 damage unless controller sacrifices enchanted artifact",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						aura := g.FindPermanent(sourceID)
						if aura == nil || !aura.IsAttached() {
							return nil
						}
						target := g.FindPermanent(aura.AttachedTo)
						if target == nil {
							return nil
						}
						p := g.GetPlayer(target.ControllerID())
						if p == nil {
							return nil
						}
						if p.ChooseMayAbility("sacrifice enchanted artifact to prevent 2 damage") {
							g.DoSacrifice(target)
						} else {
							g.DealDamageToPlayer(p, 2, sourceID)
						}
						return nil
					},
				),
				false,
			)),
		)
	})

	// Dance of Many {U}{U}
	// Enchantment
	// When this enchantment enters, create a token that's a copy of target nontoken creature.
	// When this enchantment leaves the battlefield, exile the token.
	// When the token leaves the battlefield, sacrifice this enchantment.
	// At the beginning of your upkeep, sacrifice this enchantment unless you pay {U}{U}.
	Register("Dance of Many", func() Card {
		return NewEnchantment("Dance of Many", "{U}{U}",
			WithAbility(SacrificeAtUpkeepUnlessPay("{U}{U}")),
			WithAbility(NewTriggered(EvtZoneChange, false,
				FuncEffect(
					"create token copy of target nontoken creature",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := g.FindPermanent(targets[0])
						if target == nil {
							return nil
						}
						tokenCard := target.Card.Copy()
						tokenPerm := g.PutOnBattlefield(tokenCard, controller)
						if tokenPerm == nil {
							return nil
						}
						tokenPerm.IsToken = true
						tokenPerm.CreatedBy = sourceID

						// When the token leaves the battlefield, sacrifice this enchantment.
						tokenPerm.RuntimeAbilities = append(tokenPerm.RuntimeAbilities,
							OnLeaveZone(ZoneBattlefield, ZoneAny, FuncEffect(
								"sacrifice Dance of Many",
								EffectProperties{Outcome: OutcomeDetriment},
								func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
									dance := g.FindPermanent(sourceID)
									if dance != nil {
										g.DoSacrifice(dance)
									}
									return nil
								},
							), false),
						)
						return nil
					},
				),
			).
				SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventSourceIsSelf{},
					EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
				}}).
				AddTarget(TargetCreature(Not(IsToken)))),
			// When this enchantment leaves the battlefield, exile the token.
			WithAbility(OnLeaveZone(ZoneBattlefield, ZoneAny, FuncEffect(
				"exile the token",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
					for _, perm := range g.FilterBattlefield(CreatedByFilter(sourceID)) {
						g.ExilePermanent(perm)
					}
					return nil
				},
			), false)),
		)
	})

	// Dark Heart of the Wood {B}{G}
	// Enchantment
	// Sacrifice a Forest: You gain 3 life.
	Register("Dark Heart of the Wood", func() Card {
		return NewEnchantment("Dark Heart of the Wood", "{B}{G}",
			WithActivatedAbility(
				GainLife(3),
				GenericCost(0),
				WithCost(SacrificeMatchingCost(And(IsLand, HasSubType("Forest")), "Sacrifice a Forest")),
			),
		)
	})

	// Deep Water {U}{U}
	// Enchantment
	// {U}: Until end of turn, if you tap a land you control for mana, it produces {U} instead of any other type.
	Register("Deep Water", func() Card {
		return NewEnchantment("Deep Water", "{U}{U}",
			WithActivatedAbility(
				FuncEffect(
					"until end of turn, if you tap a land you control for mana, it produces {U} instead of any other type",
					EffectProperties{},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						g.SetDeepWaterActive(controller)
						return nil
					},
				),
				ManaCostOf("{U}"),
			),
		)
	})

	// Fasting {W}
	// Enchantment
	// At the beginning of your upkeep, put a hunger counter on this enchantment. Then destroy this enchantment if it has five or more hunger counters on it.
	// If you would begin your draw step, you may skip that step instead. If you do, you gain 2 life.
	// When you draw a card, destroy this enchantment.
	Register("Fasting", func() Card {
		return NewEnchantment("Fasting", "{W}",
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect(
					"put a hunger counter on Fasting, then destroy it if it has five or more hunger counters",
					EffectProperties{},
					func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						perm.AddCounter(Hunger, 1)
						if perm.Counters[Hunger] >= 5 {
							g.DestroyPermanent(perm)
						}
						return nil
					},
				),
				false,
			)),
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"register Fasting draw replacement",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					g.AddFastingReplacement(controller, sourceID)
					return nil
				},
			), false)),
			WithAbility(NewTriggered(EvtCardDrawn, false,
				FuncEffect(
					"destroy Fasting",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							g.DestroyPermanent(perm)
						}
						return nil
					},
				),
			).SetConditionData(EventPlayerIsController{})),
		)
	})

	// Gaea's Touch {G}{G}
	// Enchantment
	// {0}: You may put a basic Forest card from your hand onto the battlefield. Activate only as a sorcery and only once each turn.
	// Sacrifice this enchantment: Add {G}{G}.
	Register("Gaea's Touch", func() Card {
		return NewEnchantment("Gaea's Touch", "{G}{G}",
			WithActivatedAbility(
				FuncEffect(
					"put a basic Forest card from your hand onto the battlefield",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						var forests []Card
						for _, c := range p.Hand() {
							if c.HasType(TypeLand) && c.HasSuperType(SuperBasic) && c.HasSubType("Forest") {
								forests = append(forests, c)
							}
						}
						if len(forests) == 0 {
							return nil
						}
						if !p.ChooseMayAbility("put a basic Forest from your hand onto the battlefield") {
							return nil
						}
						chosen := p.ChooseCardFromHand(forests, "choose a basic Forest to put onto the battlefield", g)
						if chosen == nil {
							return nil
						}
						if c, ok := p.RemoveFromHand(chosen.ID()); ok {
							g.PutOnBattlefield(c, controller)
						}
						return nil
					},
				),
				GenericCost(0),
				WithSorcerySpeed(),
				WithOncePerTurn(),
			),
			WithActivatedAbility(
				AddMana(Green, 2),
				GenericCost(0),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Goblin Caves {1}{R}{R}
	// Enchantment — Aura
	// Enchant land
	// As long as enchanted land is a basic Mountain, Goblin creatures get +0/+2.
	Register("Goblin Caves", func() Card {
		return NewAura("Goblin Caves", "{1}{R}{R}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					aura := g.FindPermanent(sourceID)
					if aura == nil || !aura.IsAttached() {
						return nil
					}
					land := g.FindPermanent(aura.AttachedTo)
					if land == nil || !land.HasSubType("Mountain") || !land.Card.HasSuperType(SuperBasic) {
						return nil
					}
					for _, p := range g.FilterBattlefield(And(IsCreature, HasSubType("Goblin"))) {
						p.BoostPT(0, 2)
					}
					return nil
				}),
			),
		)
	})

	// Goblin Shrine {1}{R}{R}
	// Enchantment — Aura
	// Enchant land
	// As long as enchanted land is a basic Mountain, Goblin creatures get +1/+0.
	// When this Aura leaves the battlefield, it deals 1 damage to each Goblin creature.
	Register("Goblin Shrine", func() Card {
		return NewAura("Goblin Shrine", "{1}{R}{R}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					aura := g.FindPermanent(sourceID)
					if aura == nil || !aura.IsAttached() {
						return nil
					}
					land := g.FindPermanent(aura.AttachedTo)
					if land == nil || !land.HasSubType("Mountain") || !land.Card.HasSuperType(SuperBasic) {
						return nil
					}
					for _, p := range g.FilterBattlefield(And(IsCreature, HasSubType("Goblin"))) {
						p.BoostPT(1, 0)
					}
					return nil
				}),
			),
			WithAbility(OnLeaveZone(ZoneBattlefield, ZoneAny, FuncEffect(
				"deals 1 damage to each Goblin creature",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
					for _, p := range g.FilterBattlefield(And(IsCreature, HasSubType("Goblin"))) {
						g.DealDamageToPermanent(p, 1, sourceID)
					}
					return nil
				},
			), false)),
		)
	})

	// Hidden Path {2}{G}{G}{G}{G}
	// Enchantment
	// Green creatures have forestwalk. (They can't be blocked as long as defending player controls a Forest.)
	Register("Hidden Path", func() Card {
		return NewEnchantment("Hidden Path", "{2}{G}{G}{G}{G}",
			WithStaticAbility(
				GrantKeywordToAllIncludingSource(Forestwalk, And(IsCreature, HasColorFilter(Green))),
			),
		)
	})

	// Mana Vortex {1}{U}{U}
	// Enchantment
	// When you cast this spell, counter it unless you sacrifice a land.
	// At the beginning of each player's upkeep, that player sacrifices a land of their choice.
	// When there are no lands on the battlefield, sacrifice this enchantment.
	Register("Mana Vortex", func() Card {
		return NewEnchantment("Mana Vortex", "{1}{U}{U}",
			WithAbility(NewTriggered(EvtSpellCast, false,
				FuncEffect(
					"counter Mana Vortex unless you sacrifice a land",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						lands := g.FilterBattlefield(And(ControlledBy(controller), IsLand))
						sacrificed := false
						if len(lands) > 0 && p.ChooseMayAbility("sacrifice a land to prevent countering Mana Vortex") {
							chosen := p.ChoosePermanent(lands, "choose land to sacrifice for Mana Vortex", g)
							if chosen != nil {
								g.DoSacrifice(chosen)
								sacrificed = true
							}
						}
						if !sacrificed {
							g.CounterSpellOnStack(sourceID)
						}
						return nil
					},
				),
			).InZone(ZoneStack).SetConditionData(EventSourceIsSelf{})),
			WithAbility(BeginningOfEachUpkeepTrigger(
				FuncEffect(
					"active player sacrifices a land of their choice",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
						active := g.ActivePlayerObj()
						if active == nil {
							return nil
						}
						lands := g.FilterBattlefield(And(ControlledBy(active.PlayerID()), IsLand))
						if len(lands) > 0 {
							chosen := active.ChoosePermanent(lands, "choose a land to sacrifice", g)
							if chosen != nil {
								g.DoSacrifice(chosen)
							}
						}
						return nil
					},
				),
				false,
			)),
			WithAbility(NewStateTriggered(false,
				FuncEffect(
					"sacrifice Mana Vortex when there are no lands on the battlefield",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							g.DoSacrifice(perm)
						}
						return nil
					},
				),
			).SetConditionData(NoBattlefieldPermanentMatching{Filter: IsLand})),
		)
	})

	// Psychic Allergy {3}{U}{U}
	// Enchantment
	// As this enchantment enters, choose a color.
	// At the beginning of each opponent's upkeep, this enchantment deals X damage to that player, where X is the number of nontoken permanents of the chosen color they control.
	// At the beginning of your upkeep, destroy this enchantment unless you sacrifice two Islands.
	Register("Psychic Allergy", func() Card {
		return NewEnchantment("Psychic Allergy", "{3}{U}{U}",
			WithETBEffect(ChooseColorStep("Choose a color for Psychic Allergy")),
			WithAbility(BeginningOfEachUpkeepTrigger(
				FuncEffect(
					"deals X damage to opponent where X is number of nontoken permanents of chosen color they control",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						active := g.ActivePlayerObj()
						if active == nil || active.PlayerID() == controller {
							return nil
						}
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						color := perm.ChosenColor
						count := 0
						for _, p := range g.FilterBattlefield(ControlledBy(active.PlayerID())) {
							if !p.IsToken && slices.Contains(p.Colors(), color) {
								count++
							}
						}
						if count > 0 {
							g.DealDamageToPlayer(active, count, sourceID)
						}
						return nil
					},
				),
				false,
			)),
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect(
					"destroy Psychic Allergy unless you sacrifice two Islands",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						islands := g.FilterBattlefield(And(ControlledBy(controller), IsLand, HasSubType("Island")))
						sacced := false
						if len(islands) >= 2 && p.ChooseMayAbility("sacrifice two Islands to keep Psychic Allergy") {
							first := p.ChoosePermanent(islands, "choose first Island to sacrifice", g)
							if first != nil {
								g.DoSacrifice(first)
								var remaining []*Permanent
								for _, isl := range islands {
									if isl.ID() != first.ID() {
										remaining = append(remaining, isl)
									}
								}
								second := p.ChoosePermanent(remaining, "choose second Island to sacrifice", g)
								if second != nil {
									g.DoSacrifice(second)
									sacced = true
								}
							}
						}
						if !sacced {
							g.DestroyPermanent(perm)
						}
						return nil
					},
				),
				false,
			)),
		)
	})

	// Season of the Witch {B}{B}{B}
	// Enchantment
	// At the beginning of your upkeep, sacrifice this enchantment unless you pay 2 life.
	// At the beginning of the end step, destroy all untapped creatures that didn't attack this turn, except for creatures that couldn't attack.
	Register("Season of the Witch", func() Card {
		return NewEnchantment("Season of the Witch", "{B}{B}{B}",
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect(
					"sacrifice Season of the Witch unless you pay 2 life",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						if p.Life() > 2 && p.ChooseMayAbility("pay 2 life to keep Season of the Witch") {
							g.PlayerLoseLife(p, 2)
						} else {
							g.DoSacrifice(perm)
						}
						return nil
					},
				),
				false,
			)),
			WithAbility(BeginningOfEachEndStepTrigger(
				FuncEffect(
					"destroy all untapped creatures that didn't attack this turn, except for creatures that couldn't attack",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
						var toDestroy []*Permanent
						for _, p := range g.FilterBattlefield(IsCreature) {
							if p.Tapped {
								continue
							}
							// Didn't attack this turn
							if g.HasAttackedThisTurn(p.ID()) {
								continue
							}
							// Except creatures that couldn't attack
							if !p.CanDeclareAsAttacker(g) {
								continue
							}
							toDestroy = append(toDestroy, p)
						}
						for _, p := range toDestroy {
							g.DestroyPermanent(p)
						}
						return nil
					},
				),
				false,
			)),
		)
	})

	// Tangle Kelp {U}
	// Enchantment — Aura
	// Enchant creature
	// When this Aura enters, tap enchanted creature.
	// Enchanted creature doesn't untap during its controller's untap step if it attacked during its controller's last turn.
	Register("Tangle Kelp", func() Card {
		return NewAura("Tangle Kelp", "{U}",
			WithCastTarget(TargetCreature()),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"tap enchanted creature",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						aura := g.FindPermanent(sourceID)
						if aura != nil && aura.IsAttached() {
							target := g.FindPermanent(aura.AttachedTo)
							if target != nil {
								g.TapPermanent(target)
							}
						}
						return nil
					},
				),
				false,
			)),
			WithStaticAbility(
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					if g.AttackedDuringLastTurn(target.ControllerID(), target.ID()) {
						g.GrantAttr(target.ID(), AttrDoesNotUntap)
					}
					return nil
				}),
			),
		)
	})

	// Worms of the Earth {2}{B}{B}{B}
	// Enchantment
	// Players can't play lands.
	// Lands can't enter the battlefield.
	// At the beginning of each upkeep, any player may sacrifice two lands of their choice or have this enchantment deal 5 damage to that player. If a player does either, destroy this enchantment.
	Register("Worms of the Earth", func() Card {
		return NewEnchantment("Worms of the Earth", "{2}{B}{B}{B}",
			WithStaticAbility(
				WormsOfTheEarthEffect(),
			),
			WithAbility(BeginningOfEachUpkeepTrigger(
				FuncEffect(
					"any player may sacrifice two lands or take 5 damage to destroy Worms of the Earth",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						// Active player gets first choice, then non-active player
						for _, p := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
							if p == nil {
								continue
							}
							lands := g.FilterBattlefield(And(ControlledBy(p.PlayerID()), IsLand))
							if len(lands) >= 2 && p.ChooseMayAbility("sacrifice two lands to destroy Worms of the Earth") {
								first := p.ChoosePermanent(lands, "choose first land to sacrifice", g)
								if first != nil {
									g.DoSacrifice(first)
									var remaining []*Permanent
									for _, l := range lands {
										if l.ID() != first.ID() {
											remaining = append(remaining, l)
										}
									}
									second := p.ChoosePermanent(remaining, "choose second land to sacrifice", g)
									if second != nil {
										g.DoSacrifice(second)
										g.DestroyPermanent(perm)
										return nil
									}
								}
							}
							if p.ChooseMayAbility("have Worms of the Earth deal 5 damage to you to destroy it") {
								g.DealDamageToPlayer(p, 5, sourceID)
								g.DestroyPermanent(perm)
								return nil
							}
						}
						return nil
					},
				),
				false,
			)),
		)
	})

}
