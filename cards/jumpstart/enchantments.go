package jumpstart

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

// spellCastCasterSelector picks the player who cast the spell that triggered
// an EvtSpellCast trigger. The trigger machinery passes targets[0] as the
// spell's stack ID and targets[1] as the caster's player ID.
type spellCastCasterSelector struct{}

func (spellCastCasterSelector) Select(_ GameReader, _, _ uuid.UUID, targets []uuid.UUID) []uuid.UUID {
	if len(targets) < 2 {
		return nil
	}
	return []uuid.UUID{targets[1]}
}

func (spellCastCasterSelector) Text() string { return "that player" }

func registerEnchantments() {

	// Assault Formation {1}{G}
	// Enchantment
	// Each creature you control assigns combat damage equal to its toughness rather than its power.
	// {G}: Target creature with defender can attack this turn as though it didn't have defender.
	// {2}{G}: Creatures you control get +0/+1 until end of turn.
	Register("Assault Formation", func() Card {
		return NewEnchantment("Assault Formation", "{1}{G}",
			WithStaticAbility(AssignsDamageEqualToToughnessForCreaturesYouControl()),
			WithActivatedAbility(
				FuncEffect(
					"target creature with defender can attack this turn as though it didn't have defender",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 || targets[0] == uuid.Nil {
							return nil
						}
						eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g *Game, target *Permanent) error {
							g.RevokeAttr(target.ID(), Defender)
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					},
				),
				ManaCostOf("{G}"),
				WithTarget(TargetCreature(HasKeywordFilter(Defender))),
			),
			WithActivatedAbility(
				BoostMatchingUntilEndOfTurn(Fixed(0), Fixed(1), IsCreature),
				ManaCostOf("{2}{G}"),
			),
		)
	})

	// Barrage of Expendables {R}
	// Enchantment
	// {R}, Sacrifice a creature: This enchantment deals 1 damage to any target.
	Register("Barrage of Expendables", func() Card {
		return NewEnchantment("Barrage of Expendables", "{R}",
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				ManaCostOf("{R}"),
				WithCost(SacrificeCreatureCost()),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	// Black Market {3}{B}{B}
	// Enchantment
	// Whenever a creature dies, put a charge counter on this enchantment.
	// At the beginning of your first main phase, add {B} for each charge counter on this enchantment.
	Register("Black Market", func() Card {
		return NewEnchantment("Black Market", "{3}{B}{B}",
			WithAbility(AnyCreatureDiesTrigger(
				AddCounters(Charge, Fixed(1)).Targeting(ToSource()),
				false,
			)),
			WithAbility(BeginningOfFirstMainPhaseTrigger(
				FuncEffect("add {B} for each charge counter on this enchantment",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						n := int(src.Counters[Charge])
						for range n {
							p.ManaPool().Add(Black, 1)
						}
						return nil
					}),
				false,
			)),
		)
	})

	// Blessed Sanctuary {3}{W}{W}
	// Enchantment
	// Prevent all noncombat damage that would be dealt to you and creatures you control.
	// Whenever a nontoken creature you control enters, create a 2/2 white Unicorn creature token.
	Register("Blessed Sanctuary", func() Card {
		nontokenCreatureYouControl := NewPermanentFilter("nontoken creature you control", func(p *Permanent, _ *Game) bool {
			return p.HasType(TypeCreature) && !p.Card.IsToken()
		})
		return NewEnchantment("Blessed Sanctuary", "{3}{W}{W}",
			WithStaticAbility(PreventNoncombatDamageToControllerAndCreatures()),
			WithAbility(WheneverPermanentEntersBattlefieldTrigger(
				CreateColoredToken("Unicorn", 2, 2, []Color{White}, []CardType{TypeCreature}, []string{"Unicorn"}),
				false,
				nontokenCreatureYouControl,
			).AndConditionData(EventSourceControlledByController{})),
		)
	})

	// Branching Evolution {2}{G}
	// Enchantment
	// If one or more +1/+1 counters would be put on a creature you control, twice that many +1/+1 counters are put on that creature instead.
	Register("Branching Evolution", func() Card {
		return NewEnchantment("Branching Evolution", "{2}{G}",
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("register +1/+1 counter doubler for creatures you control",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						g.AddCounterDoubler(sourceID, P1P1, And(IsCreature, ControlledBy(controller)))
						return nil
					}), false),
			),
		)
	})

	// Cathars' Crusade {3}{W}{W}
	// Enchantment
	// Whenever a creature you control enters, put a +1/+1 counter on each creature you control.
	Register("Cathars' Crusade", func() Card {
		return NewEnchantment("Cathars' Crusade", "{3}{W}{W}",
			WithAbility(
				NewTriggered(EvtZoneChange, false,
					FuncEffect("put a +1/+1 counter on each creature you control",
						EffectProperties{Outcome: OutcomeBenefit, Mass: true},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							for _, p := range g.FilterBattlefield(And(IsCreature, ControlledBy(controller))) {
								p = g.MutablePermanent(p.ID())
								if p == nil {
									continue
								}
								p.AddCounter(P1P1, 1)
							}
							return nil
						}),
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
					EventSourceControlledByController{},
					EventSourceMatchesPermanentFilter{Filter: IsCreature},
				}}),
			),
		)
	})

	// Celestial Mantle {3}{W}{W}{W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +3/+3.
	// Whenever enchanted creature deals combat damage to a player, double its controller's life total.
	Register("Celestial Mantle", func() Card {
		return NewAura("Celestial Mantle", "{3}{W}{W}{W}",
			WithStaticAbility(BoostAttached(3, 3, AttachAura)),
			WithAbility(
				NewTriggered(EvtDamageDealt, false,
					FuncEffect("double enchanted creature's controller's life total",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
							src := g.FindPermanent(sourceID)
							if src == nil || src.AttachedTo == uuid.Nil {
								return nil
							}
							host := g.FindPermanent(src.AttachedTo)
							if host == nil {
								return nil
							}
							p := g.GetPlayer(host.Controller)
							if p == nil {
								return nil
							}
							gain := p.Life()
							if gain <= 0 {
								return nil
							}
							g.PlayerGainLife(p, gain)
							return nil
						}),
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventSourceIsAttachedTo{},
					EventTargetIsPlayer{},
					EventIsCombatDamage{},
				}}),
			),
		)
	})

	// Coastal Piracy {2}{U}{U}
	// Enchantment
	// Whenever a creature you control deals combat damage to an opponent, you may draw a card.
	Register("Coastal Piracy", func() Card {
		return NewEnchantment("Coastal Piracy", "{2}{U}{U}",
			WithAbility(WheneverPermanentDealsCombatDamageToPlayerTrigger(
				DrawCards(Fixed(1)), true, IsCreature)),
		)
	})

	// Cradle of Vitality {3}{W}
	// Enchantment
	// Whenever you gain life, you may pay {1}{W}. If you do, put a +1/+1 counter on target creature for each 1 life you gained.
	Register("Cradle of Vitality", func() Card {
		return NewEnchantment("Cradle of Vitality", "{3}{W}",
			WithAbility(WheneverYouGainLifeTrigger(
				MayPayMana("{1}{W}",
					"put a +1/+1 counter on target creature for each 1 life you gained",
					FuncEffect("put a +1/+1 counter on target creature for each 1 life you gained",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							n := g.EventAmount()
							if n <= 0 {
								return nil
							}
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							candidates := g.FilterBattlefield(IsCreature)
							if len(candidates) == 0 {
								return nil
							}
							chosen := p.ChoosePermanent(candidates, "target creature for Cradle of Vitality counters", g)
							if chosen == nil {
								return nil
							}
							g.AddCountersWithReplacement(chosen, P1P1, n, sourceID, false)
							return nil
						}),
				),
				false,
			)),
		)
	})

	// Curiosity {U}
	// Enchantment — Aura
	// Enchant creature
	// Whenever enchanted creature deals damage to an opponent, you may draw a card.
	Register("Curiosity", func() Card {
		return NewAura("Curiosity", "{U}",
			WithAbility(WheneverEnchantedPermanentDealsDamageToPlayerTrigger(
				DrawCards(Fixed(1)), true)),
		)
	})

	// Curious Obsession {U}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +1/+1 and has "Whenever this creature deals combat damage to a player, you may draw a card."
	// At the beginning of your end step, if you didn't attack with a creature this turn, sacrifice this Aura.
	Register("Curious Obsession", func() Card {
		return NewAura("Curious Obsession", "{U}",
			WithStaticAbility(
				BoostAttached(1, 1, AttachAura),
				GrantTriggeredAbilityToAttached(
					EvtDamageDealt, true,
					AndTriggerCond{Conditions: []TriggerConditionData{
						EventSourceIsSelf{},
						EventTargetIsPlayer{},
						EventIsCombatDamage{},
					}},
					DrawCards(Fixed(1)),
				),
			),
			WithAbility(
				NewTriggered(EvtEndStep, false,
					SacrificeSourceStep(),
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventPlayerIsController{},
					controllerDidNotAttackThisTurn{},
				}}),
			),
		)
	})

	// Death's Approach {B}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets -X/-X, where X is the number of creature cards in its controller's graveyard.
	Register("Death's Approach", func() Card {
		return NewAura("Death's Approach", "{B}",
			WithStaticAbility(
				AttachedEffect(LayerPT, func(g *Game, source, target *Permanent) error {
					owner := g.GetPlayer(target.Controller)
					if owner == nil {
						return nil
					}
					n := 0
					for _, c := range owner.Graveyard() {
						if c.HasType(TypeCreature) {
							n++
						}
					}
					target.BoostPT(-n, -n)
					return nil
				}),
			),
		)
	})

	// Duelist's Heritage {2}{W}
	// Enchantment
	// Whenever one or more creatures attack, you may have target attacking creature gain double strike until end of turn.
	Register("Duelist's Heritage", func() Card {
		return NewEnchantment("Duelist's Heritage", "{2}{W}",
			WithAbility(WheneverOneOrMoreCreaturesAttackTrigger(
				FuncEffect("target attacking creature gains double strike until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						eff := TemporaryKeyword(targets[0], DoubleStrike)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				true,
			).AddTarget(TargetCreature(IsAttacking))),
		)
	})

	// Eternal Thirst {1}{B}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature has lifelink and "Whenever a creature an opponent controls dies, put a +1/+1 counter on this creature." (Damage dealt by a creature with lifelink also causes its controller to gain that much life.)
	Register("Eternal Thirst", func() Card {
		return NewAura("Eternal Thirst", "{1}{B}",
			WithStaticAbility(
				GrantAbilityToAttached(Lifelink, AttachAura),
				GrantTriggeredAbilityToAttached(
					EvtZoneChange, false,
					AndTriggerCond{Conditions: []TriggerConditionData{
						EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
						EventSourceWasOfType{Type: TypeCreature},
						EventSourceControlledByOpponent{},
					}},
					AddCounters(P1P1, Fixed(1)).Targeting(ToSource()),
				),
			),
		)
	})

	// Exquisite Blood {4}{B}
	// Enchantment
	// Whenever an opponent loses life, you gain that much life.
	Register("Exquisite Blood", func() Card {
		return NewEnchantment("Exquisite Blood", "{4}{B}",
			WithAbility(WheneverOpponentLosesLifeTrigger(
				GainLifeTarget(EventAmountValue()), false)),
		)
	})

	// Face of Divinity {2}{W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +2/+2.
	// As long as another Aura is attached to enchanted creature, it has first strike and lifelink.
	Register("Face of Divinity", func() Card {
		return NewAura("Face of Divinity", "{2}{W}",
			WithStaticAbility(
				BoostAttached(2, 2, AttachAura),
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					if anotherAuraAttachedTo(g, source, target) {
						g.GrantAttr(target.ID(), FirstStrike)
						g.GrantAttr(target.ID(), Lifelink)
					}
					return nil
				}),
			),
		)
	})

	// Feral Invocation {2}{G}
	// Enchantment — Aura
	// Flash (You may cast this spell any time you could cast an instant.)
	// Enchant creature
	// Enchanted creature gets +2/+2.
	Register("Feral Invocation", func() Card {
		return NewAura("Feral Invocation", "{2}{G}",
			WithKeyword(Flash),
			WithStaticAbility(BoostAttached(2, 2, AttachAura)),
		)
	})

	// Forced Worship {1}{W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature can't attack.
	// {2}{W}: Return this Aura to its owner's hand.
	Register("Forced Worship", func() Card {
		return NewAura("Forced Worship", "{1}{W}",
			WithStaticAbility(
				PreventAttachedFromAttacking(AttachAura),
			),
			WithActivatedAbility(
				FuncEffect("return this Aura to its owner's hand",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						owner := g.GetPlayer(src.Card.Owner())
						if owner == nil {
							return nil
						}
						g.RemoveFromBattlefield(src)
						owner.AddToHand(src.Card)
						return nil
					}),
				ManaCostOf("{2}{W}"),
			),
		)
	})

	// Indomitable Will {1}{W}
	// Enchantment — Aura
	// Flash (You may cast this spell any time you could cast an instant.)
	// Enchant creature
	// Enchanted creature gets +1/+2.
	Register("Indomitable Will", func() Card {
		return NewAura("Indomitable Will", "{1}{W}",
			WithKeyword(Flash),
			WithStaticAbility(BoostAttached(1, 2, AttachAura)),
		)
	})

	// Knightly Valor {4}{W}
	// Enchantment — Aura
	// Enchant creature
	// When this Aura enters, create a 2/2 white Knight creature token with vigilance. (Attacking doesn't cause it to tap.)
	// Enchanted creature gets +2/+2 and has vigilance.
	Register("Knightly Valor", func() Card {
		return NewAura("Knightly Valor", "{4}{W}",
			WithAbility(EntersBattlefieldTrigger(
				CreateToken("Knight", 2, 2, []CardType{TypeCreature}, []string{"Knight"}, Vigilance),
				false,
			)),
			WithStaticAbility(
				BoostAttached(2, 2, AttachAura),
				GrantAbilityToAttached(Vigilance, AttachAura),
			),
		)
	})

	// Lawmage's Binding {1}{W}{U}
	// Enchantment — Aura
	// Flash
	// Enchant creature
	// Enchanted creature can't attack or block, and its activated abilities can't be activated.
	Register("Lawmage's Binding", func() Card {
		return NewAura("Lawmage's Binding", "{1}{W}{U}",
			WithKeyword(Flash),
			WithStaticAbility(
				PreventAttachedFromAttacking(AttachAura),
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					g.RevokeAttr(target.ID(), AttrCanBlock)
					return nil
				}),
				PreventAttachedFromActivatingNonManaAbilities(AttachAura),
			),
		)
	})

	// Lightning Diadem {5}{R}
	// Enchantment — Aura
	// Enchant creature
	// When this Aura enters, it deals 2 damage to any target.
	// Enchanted creature gets +2/+2.
	Register("Lightning Diadem", func() Card {
		return NewAura("Lightning Diadem", "{5}{R}",
			WithStaticAbility(BoostAttached(2, 2, AttachAura)),
			WithAbility(
				EntersBattlefieldTrigger(DealDamage(Fixed(2)), false).
					AddTarget(TargetDamageAnyTarget()),
			),
		)
	})

	// Lurking Predators {4}{G}{G}
	// Enchantment
	// Whenever an opponent casts a spell, reveal the top card of your library. If it's a creature card, put it onto the battlefield. Otherwise, you may put that card on the bottom of your library.
	Register("Lurking Predators", func() Card {
		return NewEnchantment("Lurking Predators", "{4}{G}{G}",
			WithAbility(NewTriggered(EvtSpellCast, false,
				FuncEffect("reveal top; if creature, put onto battlefield; otherwise may put on bottom",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						revealed := g.RevealTopN(p, 1)
						if len(revealed) == 0 {
							return nil
						}
						top := revealed[0]
						taken := g.RemoveTopN(p, 1)
						if len(taken) == 0 {
							return nil
						}
						if top.HasType(TypeCreature) {
							g.PutOnBattlefield(top, controller)
							return nil
						}
						if p.ChooseMayAbility("put revealed card on the bottom of your library") {
							g.PutOnBottomInRandomOrder(p, taken)
						} else {
							g.PutOnTopInChosenOrder(p, taken)
						}
						return nil
					}),
			).SetConditionData(EventPlayerIsOpponent{})),
		)
	})

	// Makeshift Munitions {1}{R}
	// Enchantment
	// {1}, Sacrifice an artifact or creature: This enchantment deals 1 damage to any target.
	Register("Makeshift Munitions", func() Card {
		return NewEnchantment("Makeshift Munitions", "{1}{R}",
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				GenericCost(1),
				WithCost(SacrificeMatchingCost(Or(IsArtifact, IsCreature), "Sacrifice an artifact or creature")),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	// Mark of the Vampire {3}{B}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +2/+2 and has lifelink.
	Register("Mark of the Vampire", func() Card {
		return NewAura("Mark of the Vampire", "{3}{B}",
			WithStaticAbility(
				BoostAttached(2, 2, AttachAura),
				GrantAbilityToAttached(Lifelink, AttachAura),
			),
		)
	})

	// Narcolepsy {1}{U}
	// Enchantment — Aura
	// Enchant creature
	// At the beginning of each upkeep, if enchanted creature is untapped, tap it.
	Register("Narcolepsy", func() Card {
		return NewAura("Narcolepsy", "{1}{U}",
			WithAbility(
				NewTriggered(EvtUpkeep, false,
					FuncEffect("tap enchanted creature if untapped",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							src := g.FindPermanent(sourceID)
							if src == nil || src.AttachedTo == uuid.Nil {
								return nil
							}
							target := g.FindPermanent(src.AttachedTo)
							if target != nil && !target.Tapped {
								g.TapPermanent(target)
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
					src := g.FindPermanent(sourceID)
					if src == nil || src.AttachedTo == uuid.Nil {
						return false
					}
					target := g.FindPermanent(src.AttachedTo)
					return target != nil && !target.Tapped
				}),
			),
		)
	})

	// New Horizons {2}{G}
	// Enchantment — Aura
	// Enchant land
	// When this Aura enters, put a +1/+1 counter on target creature you control.
	// Enchanted land has "{T}: Add two mana of any one color."
	Register("New Horizons", func() Card {
		return NewAura("New Horizons", "{2}{G}",
			WithCastTarget(TargetLand()),
			WithAbility(
				EntersBattlefieldTrigger(
					AddCounters(P1P1, Fixed(1)),
					false,
				).AddTarget(TargetCreatureYouControl()),
			),
			WithStaticAbility(
				GrantManaAbilityToAttached(ManaProduction{Color: AnyColor, Amount: 2}),
			),
		)
	})

	// Pacifism {1}{W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature can't attack or block.
	Register("Pacifism", func() Card {
		return NewAura("Pacifism", "{1}{W}",
			WithStaticAbility(
				PreventAttachedFromAttacking(AttachAura),
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					g.RevokeAttr(target.ID(), AttrCanBlock)
					return nil
				}),
			),
		)
	})

	// Parasitic Implant {3}{B}
	// Enchantment — Aura
	// Enchant creature
	// At the beginning of your upkeep, enchanted creature's controller sacrifices it and you create a 1/1 colorless Phyrexian Myr artifact creature token.
	Register("Parasitic Implant", func() Card {
		return NewAura("Parasitic Implant", "{3}{B}",
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect(
					"enchanted creature's controller sacrifices it; create a 1/1 colorless Phyrexian Myr artifact creature token",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						host := g.FindPermanent(src.AttachedTo)
						if host != nil {
							g.Sacrifice(host)
						}
						token := NewToken("Myr", 1, 1,
							[]CardType{TypeArtifact, TypeCreature}, []string{"Phyrexian", "Myr"})
						token.SetOwner(controller)
						g.PutOnBattlefield(token, controller)
						return nil
					},
				),
				false,
			)),
		)
	})

	// Path of Bravery {2}{W}
	// Enchantment
	// As long as your life total is greater than or equal to your starting life total, creatures you control get +1/+1.
	// Whenever one or more creatures you control attack, you gain life equal to the number of attacking creatures.
	Register("Path of Bravery", func() Card {
		return NewEnchantment("Path of Bravery", "{2}{W}",
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					ctrl := g.GetPlayer(src.Controller)
					if ctrl == nil || ctrl.Life() < ctrl.StartingLife() {
						return nil
					}
					for _, p := range g.FilterBattlefield(And(IsCreature, ControlledBy(src.Controller))) {
						p = g.MutablePermanent(p.ID())
						if p == nil {
							continue
						}
						p.BoostPT(1, 1)
					}
					return nil
				}),
			),
			WithAbility(WheneverOneOrMoreCreaturesYouControlAttackTrigger(
				GainLifeAmount(EventAmountValue()), false,
			)),
		)
	})

	// Phyrexian Reclamation {B}
	// Enchantment
	// {1}{B}, Pay 2 life: Return target creature card from your graveyard to your hand.
	Register("Phyrexian Reclamation", func() Card {
		return NewEnchantment("Phyrexian Reclamation", "{B}",
			WithActivatedAbility(
				ReturnFromGraveyardToHandTarget(),
				ManaCostOf("{1}{B}"),
				WithCost(LifePayCost(2)),
				WithTarget(TargetCardInYourGraveyard(IsCreatureCard)),
			),
		)
	})

	// Presence of Gond {2}{G}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature has "{T}: Create a 1/1 green Elf Warrior creature token."
	Register("Presence of Gond", func() Card {
		return NewAura("Presence of Gond", "{2}{G}",
			WithStaticAbility(
				GrantActivatedAbilityToAttached(
					CreateToken("Elf Warrior", 1, 1, []CardType{TypeCreature}, []string{"Elf", "Warrior"}),
					Tap(),
					AttachAura,
				),
			),
		)
	})

	// Primeval Bounty {5}{G}
	// Enchantment
	// Whenever you cast a creature spell, create a 3/3 green Beast creature token.
	// Whenever you cast a noncreature spell, put three +1/+1 counters on target creature you control.
	// Landfall — Whenever a land you control enters, you gain 3 life.
	Register("Primeval Bounty", func() Card {
		noncreatureCard := NewCardFilter("noncreature card", func(c Card) bool {
			return !c.HasType(TypeCreature)
		})
		return NewEnchantment("Primeval Bounty", "{5}{G}",
			WithAbility(WheneverYouCastSpellTrigger(
				CreateColoredToken("Beast", 3, 3, []Color{Green}, []CardType{TypeCreature}, []string{"Beast"}),
				false, IsCreatureCard,
			)),
			WithAbility(WheneverYouCastSpellTrigger(
				AddCounters(P1P1, Fixed(3)).Targeting(ToTarget()),
				false, noncreatureCard,
			).AddTarget(TargetCreatureYouControl())),
			WithAbility(WheneverLandEntersBattlefieldTrigger(
				GainLife(3), false,
			).AndConditionData(EventSourceControlledByController{})),
		)
	})

	// Rhystic Study {2}{U}
	// Enchantment
	// Whenever an opponent casts a spell, you may draw a card unless that player pays {1}.
	Register("Rhystic Study", func() Card {
		return NewEnchantment("Rhystic Study", "{2}{U}",
			WithAbility(WheneverSpellCastTrigger(
				UnlessTargetPays(
					spellCastCasterSelector{},
					ManaCostOf("{1}"),
					"Pay {1} to prevent the opponent's Rhystic Study draw?",
					drawSelfCard(1),
				),
				true,
			).SetConditionData(EventPlayerIsOpponent{})),
		)
	})

	// Sarkhan's Unsealing {3}{R}
	// Enchantment
	// Whenever you cast a creature spell with power 4, 5, or 6, this enchantment deals 4 damage to any target.
	// Whenever you cast a creature spell with power 7 or greater, this enchantment deals 4 damage to each opponent and each creature and planeswalker they control.
	Register("Sarkhan's Unsealing", func() Card {
		creaturePower4to6 := NewCardFilter("creature with power 4, 5, or 6", func(c Card) bool {
			if !c.HasType(TypeCreature) {
				return false
			}
			p := c.Power()
			return p >= 4 && p <= 6
		})
		creaturePower7plus := NewCardFilter("creature with power 7 or greater", func(c Card) bool {
			return c.HasType(TypeCreature) && c.Power() >= 7
		})
		return NewEnchantment("Sarkhan's Unsealing", "{3}{R}",
			WithAbility(
				WheneverYouCastSpellTrigger(DealDamage(Fixed(4)), false, creaturePower4to6).
					AddTarget(TargetDamageAnyTarget()),
			),
			WithAbility(
				WheneverYouCastSpellTrigger(FuncEffect(
					"deal 4 damage to each opponent and each creature and planeswalker they control",
					EffectProperties{Outcome: OutcomeBenefit, Mass: true},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						opponent := g.GetOpponent(controller)
						if opponent != nil {
							g.DealDamageToPlayer(opponent, 4, sourceID)
						}
						for _, perm := range g.FilterBattlefield(NewPermanentFilter(
							"creature or planeswalker opponent controls",
							func(p *Permanent, _ *Game) bool {
								return p.Controller != controller && (p.HasType(TypeCreature) || p.HasType(TypePlaneswalker))
							},
						)) {
							g.DealDamageToPermanent(perm, 4, sourceID)
						}
						return nil
					},
				), false, creaturePower7plus),
			),
		)
	})

	// Sky Tether {W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature has defender and loses flying.
	Register("Sky Tether", func() Card {
		return NewAura("Sky Tether", "{W}",
			WithStaticAbility(
				GrantAbilityToAttached(Defender, AttachAura),
				RemoveKeywordFromAttached(Flying, AttachAura),
			),
		)
	})

	// Stab Wound {2}{B}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets -2/-2.
	// At the beginning of the upkeep of enchanted creature's controller, that player loses 2 life.
	Register("Stab Wound", func() Card {
		return NewAura("Stab Wound", "{2}{B}",
			WithStaticAbility(
				BoostAttached(-2, -2, AttachAura),
			),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				DealDamageToPlayers(Fixed(2), SelectAttachedController()), false,
			)),
		)
	})

	// Vastwood Zendikon {4}{G}
	// Enchantment — Aura
	// Enchant land
	// Enchanted land is a 6/4 green Elemental creature. It's still a land.
	// When enchanted land dies, return that card to its owner's hand.
	Register("Vastwood Zendikon", func() Card {
		return NewAura("Vastwood Zendikon", "{4}{G}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				AnimateAttachedLand(AnimateLandOptions{
					Power:     6,
					Toughness: 4,
					SubTypes:  []string{"Elemental"},
					Colors:    []Color{Green},
				}),
			),
			WithAbility(
				NewTriggered(EvtZoneChange, false,
					FuncEffect("return enchanted land to its owner's hand",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							evtSource := uuid.Nil
							if len(targets) > 0 {
								evtSource = targets[0]
							}
							if evtSource == uuid.Nil {
								return nil
							}
							for _, pl := range g.AllPlayers() {
								for _, c := range pl.Graveyard() {
									if c.ID() == evtSource {
										if removed, ok := pl.RemoveFromGraveyard(c.ID()); ok {
											ownerP := g.GetPlayer(c.Owner())
											if ownerP == nil {
												ownerP = pl
											}
											ownerP.AddToHand(removed)
										}
										return nil
									}
								}
							}
							return nil
						}),
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
					EventSourceWasOfType{Type: TypeCreature},
					SourceIsAttachedToEventSource{},
				}}),
			),
		)
	})

	// Verdant Embrace {3}{G}{G}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +3/+3 and has "At the beginning of each upkeep, create a 1/1 green Saproling creature token."
	Register("Verdant Embrace", func() Card {
		return NewAura("Verdant Embrace", "{3}{G}{G}",
			WithStaticAbility(
				BoostAttached(3, 3, AttachAura),
				GrantTriggeredAbilityToAttached(
					EvtUpkeep, false, nil,
					CreateToken("Saproling", 1, 1, []CardType{TypeCreature}, []string{"Saproling"}),
				),
			),
		)
	})

	// Waterknot {1}{U}{U}
	// Enchantment — Aura
	// Enchant creature
	// When this Aura enters, tap enchanted creature.
	// Enchanted creature doesn't untap during its controller's untap step.
	Register("Waterknot", func() Card {
		return NewAura("Waterknot", "{1}{U}{U}",
			WithAbility(EntersBattlefieldTrigger(TapAttachedCreature(), false)),
			WithStaticAbility(
				PreventAttachedFromUntapping(AttachAura),
			),
		)
	})

	// Zendikar's Roil {3}{G}{G}
	// Enchantment
	// Landfall — Whenever a land you control enters, create a 2/2 green Elemental creature token.
	Register("Zendikar's Roil", func() Card {
		return NewEnchantment("Zendikar's Roil", "{3}{G}{G}",
			WithAbility(
				NewTriggered(EvtZoneChange, false,
					CreateColoredToken("Elemental", 2, 2, []Color{Green}, []CardType{TypeCreature}, []string{"Elemental"}),
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
					EventSourceControlledByController{},
					EventSourceMatchesPermanentFilter{Filter: IsLand},
				}}),
			),
		)
	})

	// Zombie Infestation {1}{B}
	// Enchantment
	// Discard two cards: Create a 2/2 black Zombie creature token.
	Register("Zombie Infestation", func() Card {
		return NewEnchantment("Zombie Infestation", "{1}{B}",
			WithActivatedAbility(
				CreateToken("Zombie", 2, 2, []CardType{TypeCreature}, []string{"Zombie"}),
				DiscardCost(2),
			),
		)
	})

}
