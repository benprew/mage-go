package fourthedition

import (
	"math/rand"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

// stoneGiantTarget targets a creature its controller controls whose toughness
// is strictly less than the source's current power. It captures source state
// at Possible() time, which is the closest the engine's filter API allows for
// "less than this creature's power" (filters can't see the source).
type stoneGiantTarget struct {
	chosen []uuid.UUID
}

func (t *stoneGiantTarget) Min() int            { return 1 }
func (t *stoneGiantTarget) Max() int            { return 1 }
func (t *stoneGiantTarget) Chosen() []uuid.UUID { return t.chosen }
func (t *stoneGiantTarget) IsChosen() bool      { return len(t.chosen) > 0 }
func (t *stoneGiantTarget) Reset()              { t.chosen = nil }

func (t *stoneGiantTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	src := g.FindPermanent(sourceCard.ID())
	if src == nil {
		return nil
	}
	srcPower := src.CurrentPower(g)
	var result []uuid.UUID
	for _, p := range g.AllBattlefield() {
		if !p.HasType(TypeCreature) {
			continue
		}
		if p.Controller != controller {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		if p.CurrentToughness(g) < srcPower {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *stoneGiantTarget) Choose(_ uuid.UUID, _ Card, _ *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

func init() {
	registerCreatures()
}

func registerCreatures() {

	// ===== WHITE CREATURES =====

	// Angry Mob {2}{W}{W}
	// Creature — Human
	// 2+*/2+*
	// Trample
	// During your turn, Angry Mob's power and toughness are each equal to 2 plus the number of Swamps your opponents control. During turns other than yours, Angry Mob's power and toughness are each 2.
	Register("Angry Mob", func() Card {
		return NewCreature("Angry Mob", "{2}{W}{W}", 0, 0,
			WithSubTypes("Human"),
			WithKeyword(Trample),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					bonus := 2
					ap := g.ActivePlayerObj()
					if ap != nil && ap.PlayerID() == src.Controller {
						bonus += g.CountBattlefield(And(NotControlledBy(src.Controller), HasSubType("Swamp")))
					}
					src.BoostPT(bonus, bonus)
					return nil
				}),
			),
		)
	})

	// Pikemen {1}{W}
	// Creature — Human Soldier
	// 1/1
	// First strike; banding
	Register("Pikemen", func() Card {
		return NewCreature("Pikemen", "{1}{W}", 1, 1,
			WithSubTypes("Human", "Soldier"),
			WithKeyword(FirstStrike),
			WithKeyword(Banding),
		)
	})

	// ===== BLUE CREATURES =====

	// Apprentice Wizard {1}{U}{U}
	// Creature — Human Wizard
	// 0/1
	// {U}, {T}: Add {C}{C}{C}.
	Register("Apprentice Wizard", func() Card {
		return NewCreature("Apprentice Wizard", "{1}{U}{U}", 0, 1,
			WithSubTypes("Human", "Wizard"),
			WithActivatedAbility(
				AddMana(Colorless, 3),
				ManaCostOf("{U}"),
				WithCost(Tap()),
			),
		)
	})

	// Ghost Ship {2}{U}{U}
	// Creature — Spirit
	// 2/4
	// Flying
	// {U}{U}{U}: Regenerate Ghost Ship.
	Register("Ghost Ship", func() Card {
		return NewCreature("Ghost Ship", "{2}{U}{U}", 2, 4,
			WithSubTypes("Spirit"),
			WithKeyword(Flying),
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{U}{U}{U}"),
			),
		)
	})

	// Leviathan {5}{U}{U}{U}{U}
	// Creature — Leviathan
	// 10/10
	// Trample
	// Leviathan enters the battlefield tapped and doesn't untap during your untap step.
	// At the beginning of your upkeep, you may sacrifice two Islands. If you do, untap Leviathan.
	// Leviathan can't attack unless you sacrifice two Islands.
	// TODO: implement — needs sacrifice-to-untap and sacrifice-to-attack
	Register("Leviathan", func() Card {
		return NewCreature("Leviathan", "{5}{U}{U}{U}{U}", 10, 10,
			WithSubTypes("Leviathan"),
		)
	})

	// ===== BLACK CREATURES =====

	// Bog Imp {1}{B}
	// Creature — Imp
	// 1/1
	// Flying
	Register("Bog Imp", func() Card {
		return NewCreature("Bog Imp", "{1}{B}", 1, 1,
			WithSubTypes("Imp"),
			WithKeyword(Flying),
		)
	})

	// Murk Dwellers {3}{B}
	// Creature — Zombie
	// 2/2
	// Whenever Murk Dwellers attacks and isn't blocked, it gets +2/+0 until end of combat.
	Register("Murk Dwellers", func() Card {
		return NewCreature("Murk Dwellers", "{3}{B}", 2, 2,
			WithSubTypes("Zombie"),
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					Boost(Fixed(2), Fixed(0)).Targeting(ToSource()).Until(EndOfCombat),
				).SetConditionData(SourceIsUnblockedAttacker{}),
			),
		)
	})

	// Rag Man {2}{B}{B}
	// Creature — Human Minion
	// 2/1
	// {B}{B}{B}, {T}: Target opponent reveals their hand and discards a creature card at random. Activate only during your turn.
	Register("Rag Man", func() Card {
		return NewCreature("Rag Man", "{2}{B}{B}", 2, 1,
			WithSubTypes("Human", "Minion"),
			WithActivatedAbility(
				FuncEffect(
					"target opponent reveals their hand and discards a creature card at random",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(targets[0])
						if p == nil {
							return nil
						}
						var creatures []Card
						for _, c := range p.Hand() {
							if c.HasType(TypeCreature) {
								creatures = append(creatures, c)
							}
						}
						if len(creatures) == 0 {
							return nil
						}
						chosen := creatures[rand.Intn(len(creatures))]
						p.DiscardCard(chosen.ID())
						return nil
					},
				),
				ManaCostOf("{B}{B}{B}"),
				WithCost(Tap()),
				WithTarget(TargetOpponent()),
				WithYourTurnOnly(),
			),
		)
	})

	// Uncle Istvan {1}{B}{B}{B}
	// Creature — Human
	// 1/3
	// Prevent all damage that would be dealt to Uncle Istvan by creatures.
	Register("Uncle Istvan", func() Card {
		return NewCreature("Uncle Istvan", "{1}{B}{B}{B}", 1, 3,
			WithSubTypes("Human"),
			WithStaticAbility(
				PreventDamageFromTo(
					IsCreature,
					func(sourceID uuid.UUID) PermanentFilter { return IsID(sourceID) },
				),
			),
		)
	})

	// ===== RED CREATURES =====

	// Ball Lightning {R}{R}{R}
	// Creature — Elemental
	// 6/1
	// Trample
	// Haste
	// At the beginning of the end step, sacrifice Ball Lightning.
	Register("Ball Lightning", func() Card {
		return NewCreature("Ball Lightning", "{R}{R}{R}", 6, 1,
			WithSubTypes("Elemental"),
			WithKeyword(Trample),
			WithKeyword(Haste),
			WithAbility(
				BeginningOfEachEndStepTrigger(SacrificeSource(), false),
			),
		)
	})

	// Brothers of Fire {1}{R}{R}
	// Creature — Human Shaman
	// 2/2
	// {1}{R}{R}: Brothers of Fire deals 1 damage to any target and 1 damage to you.
	Register("Brothers of Fire", func() Card {
		return NewCreature("Brothers of Fire", "{1}{R}{R}", 2, 2,
			WithSubTypes("Human", "Shaman"),
			WithActivatedAbility(
				CompositeEffects("deal 1 damage to any target and 1 damage to you",
					DealDamage(Fixed(1)),
					DealDamageToPlayers(Fixed(1), SelectController()),
				),
				ManaCostOf("{1}{R}{R}"),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

	// Cave People {1}{R}{R}
	// Creature — Human
	// 1/4
	// Whenever Cave People attacks, it gets +1/-2 until end of turn.
	// {1}{R}{R}, {T}: Target creature gains mountainwalk until end of turn.
	Register("Cave People", func() Card {
		return NewCreature("Cave People", "{1}{R}{R}", 1, 4,
			WithSubTypes("Human"),
			WithAbility(
				AttacksTrigger(
					Boost(Fixed(1), Fixed(-2)).Targeting(ToSource()),
					false,
				),
			),
			WithActivatedAbility(
				GrantKeyword(Mountainwalk).Targeting(ToTarget()),
				ManaCostOf("{1}{R}{R}"),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Goblin Rock Sled {1}{R}
	// Creature — Goblin
	// 3/1
	// Trample
	// Goblin Rock Sled doesn't untap during your untap step if it attacked during your last turn.
	// Goblin Rock Sled can't attack unless defending player controls a Mountain.
	Register("Goblin Rock Sled", func() Card {
		return NewCreature("Goblin Rock Sled", "{1}{R}", 3, 1,
			WithSubTypes("Goblin"),
			WithKeyword(Trample),
			WithStaticAbility(
				PreventFromAttackingIfDefendingPlayerControls(HasSubType("Mountain")),
			),
			WithAbility(AttacksTrigger(FuncEffect(
				"doesn't untap during your next untap step",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					expiryTurn := g.CurrentTurn() + 2
					ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							g.GrantAttr(perm.ID(), AttrDoesNotUntap)
						}
						return nil
					}, func(g *Game, _ uuid.UUID) bool {
						return g.CurrentTurn() <= expiryTurn && g.FindPermanent(sourceID) != nil
					})
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					return nil
				},
			), false)),
		)
	})

	// Sisters of the Flame {1}{R}{R}
	// Creature — Human Shaman
	// 2/2
	// {T}: Add {R}.
	Register("Sisters of the Flame", func() Card {
		return NewCreature("Sisters of the Flame", "{1}{R}{R}", 2, 2,
			WithSubTypes("Human", "Shaman"),
			WithManaAbility(Red),
		)
	})

	// Stone Giant {2}{R}{R}
	// Creature — Giant
	// 3/4
	// {T}: Target creature you control with toughness less than Stone Giant's power gains flying until end of turn. Destroy that creature at the beginning of the next end step.
	Register("Stone Giant", func() Card {
		return NewCreature("Stone Giant", "{2}{R}{R}", 3, 4,
			WithSubTypes("Giant"),
			WithActivatedAbility(
				FuncEffect(
					"target creature gains flying until end of turn; destroy it at the next end step",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						kw := TemporaryKeyword(targets[0], Flying)
						kw.SetSourceID(sourceID)
						g.AddContinuousEffect(kw)
						g.ApplyContinuousEffects()
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:  EvtEndStep,
							TargetID:   targets[0],
							Effects:    []Effect{DestroyTarget()},
							SourceID:   sourceID,
							Controller: controller,
						})
						return nil
					},
				),
				Tap(),
				WithTarget(&stoneGiantTarget{}),
			),
		)
	})

	// ===== GREEN CREATURES =====

	// Carnivorous Plant {3}{G}
	// Creature — Plant Wall
	// 4/5
	// Defender
	Register("Carnivorous Plant", func() Card {
		return NewCreature("Carnivorous Plant", "{3}{G}", 4, 5,
			WithSubTypes("Plant", "Wall"),
			WithKeyword(Defender),
		)
	})

	// Gaea's Liege {3}{G}{G}{G}
	// Creature — Avatar
	// */*
	// As long as Gaea's Liege isn't attacking, its power and toughness are each equal to the number of Forests you control. As long as Gaea's Liege is attacking, its power and toughness are each equal to the number of Forests defending player controls.
	// {T}: Target land becomes a Forest until Gaea's Liege leaves the battlefield.
	Register("Gaea's Liege", func() Card {
		return NewCreature("Gaea's Liege", "{3}{G}{G}{G}", 0, 0,
			WithSubTypes("Avatar"),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					var who PermanentFilter
					if g.IsAttackingInCombat(sourceID) {
						who = NotControlledBy(src.Controller)
					} else {
						who = ControlledBy(src.Controller)
					}
					count := g.CountBattlefield(And(who, HasSubType("Forest")))
					src.BoostPT(count, count)
					return nil
				}),
			),
			WithActivatedAbility(
				FuncEffect(
					"target land becomes a Forest until this creature leaves the battlefield",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						ce := FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
							perm := g.FindPermanent(targetID)
							if perm == nil {
								return nil
							}
							perm.SubTypeOverride = []string{"Forest"}
							return nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						g.ApplyContinuousEffects()
						return nil
					},
				),
				Tap(),
				WithTarget(TargetPermanent(IsLand)),
			),
		)
	})

	// Land Leeches {1}{G}{G}
	// Creature — Leech
	// 2/2
	// First strike
	Register("Land Leeches", func() Card {
		return NewCreature("Land Leeches", "{1}{G}{G}", 2, 2,
			WithSubTypes("Leech"),
			WithKeyword(FirstStrike),
		)
	})

	// Marsh Viper {3}{G}
	// Creature — Snake
	// 1/2
	// Whenever Marsh Viper deals damage to a player, that player gets two poison counters.
	Register("Marsh Viper", func() Card {
		return NewCreature("Marsh Viper", "{3}{G}", 1, 2,
			WithSubTypes("Snake"),
			WithAbility(NewTriggered(EvtDamageDealt, false, PoisonTargetPlayer(2)).
				SetConditionData(EventSourceIsSelfDamageToPlayer{})),
		)
	})

	// ===== COLORLESS CREATURES =====

	// Diabolic Machine {7}
	// Artifact Creature — Construct
	// 4/4
	// {3}: Regenerate Diabolic Machine.
	Register("Diabolic Machine", func() Card {
		return NewCreature("Diabolic Machine", "{7}", 4, 4,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				RegenerateSource(),
				GenericCost(3),
			),
		)
	})
}
