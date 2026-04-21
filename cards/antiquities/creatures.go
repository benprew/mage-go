package antiquities

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

func init() {
	registerCreatures()
}

// sacrificeArtifactCaptureCMCCost sacrifices an artifact and stores its CMC
// in g.CurrentX so the effect can read it via g.XValue().
type sacrificeArtifactCaptureCMCCost struct{}

func (c *sacrificeArtifactCaptureCMCCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.AllBattlefield() {
		if p.Controller == controller && p.HasType(TypeArtifact) && p.ID() != sourceID {
			return true
		}
	}
	return false
}

func (c *sacrificeArtifactCaptureCMCCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.AllBattlefield() {
		if p.Controller == controller && p.HasType(TypeArtifact) && p.ID() != sourceID {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no artifact to sacrifice")
	}
	player := g.GetPlayer(controller)
	chosen := player.ChoosePermanent(candidates, "sacrifice artifact", g)
	if chosen == nil {
		return fmt.Errorf("no artifact to sacrifice")
	}
	g.SetXValue(chosen.Card.ManaCost().CMC())
	g.Sacrifice(chosen)
	return nil
}

func (c *sacrificeArtifactCaptureCMCCost) Text() string { return "Sacrifice an artifact" }

func registerCreatures() {
	// ===== WHITE CREATURES =====

	// Argivian Archaeologist {1}{W}{W}
	// Creature — Human Artificer
	// {W}{W}, {T}: Return target artifact card from your graveyard to your hand.
	Register("Argivian Archaeologist", func() Card {
		return NewCreature("Argivian Archaeologist", "{1}{W}{W}", 1, 1,
			WithSubTypes("Human", "Artificer"),
			WithActivatedAbility(
				ReturnFromGraveyardToHandTarget(),
				ManaCostOf("{W}{W}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCardInYourGraveyard(IsArtifactCard)),
			),
		)
	})

	// Argivian Blacksmith {1}{W}{W}
	// Creature — Human Artificer
	// {T}: Prevent the next 2 damage that would be dealt to target artifact creature this turn.
	Register("Argivian Blacksmith", func() Card {
		return NewCreature("Argivian Blacksmith", "{1}{W}{W}", 2, 2,
			WithSubTypes("Human", "Artificer"),
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(2)),
				TapSourceCost(),
				WithTarget(TargetCreature(IsArtifact)),
			),
		)
	})

	// Martyrs of Korlis {3}{W}{W}
	// Creature — Human
	// As long as Martyrs of Korlis is untapped, all damage that would be dealt to you by
	// artifacts is dealt to Martyrs of Korlis instead.
	Register("Martyrs of Korlis", func() Card {
		return NewCreature("Martyrs of Korlis", "{3}{W}{W}", 1, 6,
			WithSubTypes("Human"),
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					g.SetArtifactDamageRedirect(src.Controller, src.ID())
					return nil
				}, SourceUntapped),
			),
		)
	})

	// ===== BLUE CREATURES =====

	// Sage of Lat-Nam {1}{U}
	// Creature — Human Artificer
	// {T}, Sacrifice an artifact: Draw a card.
	Register("Sage of Lat-Nam", func() Card {
		return NewCreature("Sage of Lat-Nam", "{1}{U}", 1, 2,
			WithSubTypes("Human", "Artificer"),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				TapSourceCost(),
				WithCost(SacrificeArtifactCost()),
			),
		)
	})

	// ===== BLACK CREATURES =====

	// Phyrexian Gremlins {2}{B}
	// Creature — Phyrexian Gremlin
	// You may choose not to untap Phyrexian Gremlins during your untap step.
	// {T}: Tap target artifact. It doesn't untap during its controller's untap step for as long as
	// Phyrexian Gremlins remains tapped.
	Register("Phyrexian Gremlins", func() Card {
		return NewCreature("Phyrexian Gremlins", "{2}{B}", 1, 1,
			WithSubTypes("Phyrexian", "Gremlin"),
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				FuncEffect("tap target artifact; it doesn't untap while ~ remains tapped",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						target := g.FindPermanent(targetID)
						if target == nil {
							return nil
						}
						g.TapPermanent(target)
						// Create continuous effect: target doesn't untap while source is tapped
						eff := FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
							func(g *Game, srcID uuid.UUID) error {
								src := g.FindPermanent(srcID)
								if src == nil || !src.Tapped {
									return nil
								}
								t := g.FindPermanent(targetID)
								if t == nil {
									return nil
								}
								g.GrantAttr(t.ID(), AttrDoesNotUntap)
								return nil
							})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				TapSourceCost(),
				WithTarget(TargetPermanent(IsArtifact)),
			),
		)
	})

	// Priest of Yawgmoth {1}{B}
	// Creature — Phyrexian Human Cleric
	// {T}, Sacrifice an artifact: Add an amount of {B} equal to the sacrificed artifact's mana value.
	Register("Priest of Yawgmoth", func() Card {
		return NewCreature("Priest of Yawgmoth", "{1}{B}", 1, 2,
			WithSubTypes("Phyrexian", "Human", "Cleric"),
			WithActivatedAbility(
				FuncEffect("add {B} equal to sacrificed artifact's CMC",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// CMC was stored by the custom cost's Pay method
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						cmc := g.XValue() // repurpose CurrentX for the captured CMC
						if cmc > 0 {
							p.ManaPool().Add(Black, cmc)
						}
						return nil
					}),
				TapSourceCost(),
				WithCost(&sacrificeArtifactCaptureCMCCost{}),
			),
		)
	})

	// Xenic Poltergeist {1}{B}{B}
	// Creature — Spirit
	// {T}: Until your next upkeep, target noncreature artifact becomes an artifact creature with
	// power and toughness each equal to its mana value.
	Register("Xenic Poltergeist", func() Card {
		return NewCreature("Xenic Poltergeist", "{1}{B}{B}", 1, 1,
			WithSubTypes("Spirit"),
			WithActivatedAbility(
				FuncEffect("animate target noncreature artifact until your next upkeep",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						cmc := perm.Card.ManaCost().CMC()
						eff := TemporaryAnimateUntilNextUpkeep(perm.ID(), cmc, cmc)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				TapSourceCost(),
				WithTarget(TargetPermanent(And(IsArtifact, Not(IsCreature)))),
			),
		)
	})

	// Yawgmoth Demon {4}{B}{B}
	// Creature — Phyrexian Demon
	// Flying, first strike
	// At the beginning of your upkeep, you may sacrifice an artifact. If you don't, tap
	// Yawgmoth Demon and it deals 2 damage to you.
	Register("Yawgmoth Demon", func() Card {
		return NewCreature("Yawgmoth Demon", "{4}{B}{B}", 6, 6,
			WithSubTypes("Phyrexian", "Demon"),
			WithKeyword(Flying),
			WithKeyword(FirstStrike),
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("sacrifice an artifact or tap and take 2 damage",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						player := g.GetPlayer(controller)
						if player == nil {
							return nil
						}
						candidates := g.FilterBattlefield(And(ControlledBy(controller), IsArtifact))
						if len(candidates) > 0 && player.ChooseMayAbility("sacrifice an artifact") {
							chosen := player.ChoosePermanent(candidates, "sacrifice", g)
							if chosen != nil {
								g.Sacrifice(chosen)
								return nil
							}
						}
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							g.TapPermanent(perm)
						}
						g.DealDamageToPlayer(player, 2, sourceID)
						return nil
					}), false,
			)),
		)
	})

	// ===== RED CREATURES =====

	// Atog {1}{R}
	// Creature — Atog
	// Sacrifice an artifact: Atog gets +2/+2 until end of turn.
	Register("Atog", func() Card {
		return NewCreature("Atog", "{1}{R}", 1, 2,
			WithSubTypes("Atog"),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(2), Fixed(2), SelectSource),
				SacrificeArtifactCost(),
			),
		)
	})

	// Dwarven Weaponsmith {1}{R}
	// Creature — Dwarf Artificer
	// {T}, Sacrifice an artifact: Put a +1/+1 counter on target creature.
	// Activate only during your upkeep.
	Register("Dwarven Weaponsmith", func() Card {
		return NewCreature("Dwarven Weaponsmith", "{1}{R}", 1, 1,
			WithSubTypes("Dwarf", "Artificer"),
			WithActivatedAbility(
				AddCounters(P1P1, Fixed(1), SelectTarget),
				TapSourceCost(),
				WithCost(SacrificeArtifactCost()),
				WithTarget(TargetCreature()),
				WithUpkeepOnly(),
			),
		)
	})

	// Goblin Artisans {R}
	// Creature — Goblin Artificer
	// {T}: Choose target artifact spell you control that isn't the target of an ability from
	// another creature named Goblin Artisans. Flip a coin. If you win the flip, draw a card.
	// If you lose the flip, counter that spell.
	// XXX: The "not targeted by another Goblin Artisans" restriction is not enforced (extremely
	// rare edge case requiring stack-targeting tracking).
	Register("Goblin Artisans", func() Card {
		return NewCreature("Goblin Artisans", "{R}", 1, 1,
			WithSubTypes("Goblin", "Artificer"),
			WithActivatedAbility(
				FuncEffect("flip coin: win=draw, lose=counter own artifact spell",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						if g.FlipCoin(controller) {
							p.DrawCard()
						} else {
							// Counter target artifact spell on stack
							if len(targets) > 0 {
								g.CounterSpellOnStack(targets[0])
							}
						}
						return nil
					}),
				TapSourceCost(),
				WithTarget(TargetOwnSpellOnStack(IsArtifactCard)),
			),
		)
	})

	// Orcish Mechanics {2}{R}
	// Creature — Orc
	// {T}, Sacrifice an artifact: Orcish Mechanics deals 2 damage to any target.
	Register("Orcish Mechanics", func() Card {
		return NewCreature("Orcish Mechanics", "{2}{R}", 1, 1,
			WithSubTypes("Orc"),
			WithActivatedAbility(
				DealDamage(Fixed(2)),
				TapSourceCost(),
				WithCost(SacrificeArtifactCost()),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

	// ===== GREEN CREATURES =====

	// Argothian Pixies {1}{G}
	// Creature — Faerie
	// Argothian Pixies can't be blocked by artifact creatures.
	// Prevent all damage that would be dealt to Argothian Pixies by artifact creatures.
	Register("Argothian Pixies", func() Card {
		return NewCreature("Argothian Pixies", "{1}{G}", 2, 1,
			WithSubTypes("Faerie"),
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					// Can't be blocked by artifact creatures
					for _, blocker := range g.AllBattlefield() {
						if blocker.HasType(TypeArtifact) && blocker.HasType(TypeCreature) {
							g.PreventBlockPair(blocker.ID(), sourceID)
						}
					}
					return nil
				}),
			),
			// Prevent all damage from artifact creatures
			WithStaticAbility(PreventDamageFromTo(
				And(IsArtifact, IsCreature),
				func(sourceID uuid.UUID) PermanentFilter {
					return IsID(sourceID)
				},
			)),
		)
	})

	// Argothian Treefolk {3}{G}{G}
	// Creature — Treefolk
	// Prevent all damage that would be dealt to Argothian Treefolk by artifact sources.
	Register("Argothian Treefolk", func() Card {
		return NewCreature("Argothian Treefolk", "{3}{G}{G}", 3, 5,
			WithSubTypes("Treefolk"),
			// Prevent all damage from artifact sources
			WithStaticAbility(PreventDamageFromTo(
				IsArtifact,
				func(sourceID uuid.UUID) PermanentFilter {
					return IsID(sourceID)
				},
			)),
		)
	})

	// Citanul Druid {1}{G}
	// Creature — Human Druid
	// Whenever an opponent casts an artifact spell, put a +1/+1 counter on Citanul Druid.
	Register("Citanul Druid", func() Card {
		return NewCreature("Citanul Druid", "{1}{G}", 1, 1,
			WithSubTypes("Human", "Druid"),
			WithAbility(
				NewTriggered(EvtSpellCast, false,
					AddCounters(P1P1, Fixed(1), SelectSource),
				).SetCondition(func(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
					if evt.PlayerID == controllerID {
						return false
					}
					card := g.FindCardAnywhere(evt.SourceID)
					if card == nil {
						return false
					}
					return card.HasType(TypeArtifact)
				}),
			),
		)
	})

	// Gaea's Avenger {1}{G}{G}
	// Creature — Treefolk
	// Power and toughness are each equal to 1 plus the number of artifacts opponents control.
	Register("Gaea's Avenger", func() Card {
		return NewCreature("Gaea's Avenger", "{1}{G}{G}", 1, 1,
			WithSubTypes("Treefolk"),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					count := g.CountBattlefield(And(NotControlledBy(src.Controller), IsArtifact))
					src.BoostPT(count, count)
					return nil
				}),
			),
		)
	})

	// ===== ARTIFACT CREATURES =====

	// Battering Ram {2}
	// Artifact Creature — Construct
	// At the beginning of combat on your turn, Battering Ram gains banding until end of combat.
	// Whenever Battering Ram becomes blocked by a Wall, destroy that Wall at end of combat.
	Register("Battering Ram", func() Card {
		return NewCreature("Battering Ram", "{2}", 1, 1,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithAbility(
				NewTriggered(EvtBeginCombat, false,
					FuncEffect("gain banding until end of combat",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.FindPermanent(sourceID)
							if perm == nil {
								return nil
							}
							eff := TargetEffect(LayerAbility, EndOfCombat, perm.ID(), func(g *Game, target *Permanent) error {
								g.GrantAttr(target.ID(), Banding)
								return nil
							})
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
							g.ApplyContinuousEffects()
							return nil
						}),
				).SetCondition(func(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
					return evt.PlayerID == controllerID
				}),
			),
			// When blocked by a Wall, destroy that Wall at end of combat
			WithAbility(
				NewTriggered(EvtDeclaredBlocker, false,
					FuncEffect("destroy blocking Wall at end of combat",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							wallID := targets[0]
							g.RegisterDelayedTrigger(&DelayedTrigger{
								EventType:  EvtEndOfCombat,
								TargetID:   wallID,
								Effects:    []Effect{DestroyTarget()},
								SourceID:   sourceID,
								Controller: controller,
							})
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
					// Battering Ram is the attacker being blocked
					if evt.TargetID != sourceID {
						return false
					}
					blocker := g.FindPermanent(evt.SourceID)
					return blocker != nil && blocker.HasSubType("Wall")
				}),
			),
		)
	})

	// Clay Statue {4}
	// Artifact Creature — Golem
	// {2}: Regenerate Clay Statue.
	Register("Clay Statue", func() Card {
		return NewCreature("Clay Statue", "{4}", 3, 1,
			WithSubTypes("Golem"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				RegenerateSource(),
				GenericCost(2),
			),
		)
	})

	// Clockwork Avian {5}
	// Artifact Creature — Bird
	// Flying
	// Clockwork Avian enters the battlefield with four +1/+0 counters on it.
	// At end of combat, if Clockwork Avian attacked or blocked this combat, remove a +1/+0 counter.
	// {X}, {T}: Put up to X +1/+0 counters on Clockwork Avian (max 4 total). Activate only
	// during your upkeep.
	Register("Clockwork Avian", func() Card {
		return NewCreature("Clockwork Avian", "{5}", 0, 4,
			WithSubTypes("Bird"),
			WithCardType(TypeArtifact),
			WithKeyword(Flying),
			WithAbility(EntersWithNCounters(P1P0, 4)),
			// At end of combat, if Clockwork Avian attacked or blocked, remove a +1/+0 counter
			WithAbility(AttacksTrigger(
				FuncEffect("schedule counter removal at end of combat",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:  EvtEndOfCombat,
							TargetID:   sourceID,
							Effects:    []Effect{RemoveCountersFromSource(P1P0, 1)},
							SourceID:   sourceID,
							Controller: controller,
						})
						return nil
					}), false,
			)),
			WithAbility(BlocksTrigger(
				FuncEffect("schedule counter removal at end of combat",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:  EvtEndOfCombat,
							TargetID:   sourceID,
							Effects:    []Effect{RemoveCountersFromSource(P1P0, 1)},
							SourceID:   sourceID,
							Controller: controller,
						})
						return nil
					}), false,
			)),
			// {X}, {T}: Put up to X +1/+0 counters on Clockwork Avian (max 4 total). Upkeep only.
			WithActivatedAbility(
				AddCountersUpToMax(P1P0, 4),
				XManaCost(),
				WithCost(TapSourceCost()),
				WithUpkeepOnly(),
			),
		)
	})

	// Colossus of Sardia {9}
	// Artifact Creature — Golem
	// Trample
	// Colossus of Sardia doesn't untap during your untap step.
	// {9}: Untap Colossus of Sardia. Activate only during your upkeep.
	Register("Colossus of Sardia", func() Card {
		return NewCreature("Colossus of Sardia", "{9}", 9, 9,
			WithSubTypes("Golem"),
			WithCardType(TypeArtifact),
			WithKeyword(Trample),
			WithKeyword(DoesNotUntapKW),
			WithActivatedAbility(
				UntapSource(),
				GenericCost(9),
				WithUpkeepOnly(),
			),
		)
	})

	// Dragon Engine {3}
	// Artifact Creature — Construct
	// {2}: Dragon Engine gets +1/+0 until end of turn.
	Register("Dragon Engine", func() Card {
		return NewCreature("Dragon Engine", "{3}", 1, 3,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(0), SelectSource),
				GenericCost(2),
			),
		)
	})

	// Grapeshot Catapult {4}
	// Artifact Creature — Construct
	// {T}: Grapeshot Catapult deals 1 damage to target creature with flying.
	Register("Grapeshot Catapult", func() Card {
		return NewCreature("Grapeshot Catapult", "{4}", 2, 3,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapSourceCost(),
				WithTarget(TargetCreature(HasKeywordFilter(Flying))),
			),
		)
	})

	// Mishra's War Machine {7}
	// Artifact Creature — Juggernaut
	// Banding
	// At the beginning of your upkeep, Mishra's War Machine deals 3 damage to you unless you
	// discard a card. If it deals damage to you this way, tap it.
	Register("Mishra's War Machine", func() Card {
		return NewCreature("Mishra's War Machine", "{7}", 5, 5,
			WithSubTypes("Juggernaut"),
			WithCardType(TypeArtifact),
			WithKeyword(Banding),
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("discard a card or take 3 damage and tap",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						player := g.GetPlayer(controller)
						if player == nil {
							return nil
						}
						if len(player.Hand()) > 0 && player.ChooseMayAbility("discard a card") {
							chosen := player.ChooseCardsFromHand(1, "discard", g)
							if len(chosen) > 0 {
								for _, card := range chosen {
									player.RemoveFromHand(card.ID())
									player.AddToGraveyard(card)
								}
								return nil
							}
						}
						g.DealDamageToPlayer(player, 3, sourceID)
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							g.TapPermanent(perm)
						}
						return nil
					}), false,
			)),
		)
	})

	// Onulet {3}
	// Artifact Creature — Construct
	// When Onulet dies, you gain 2 life.
	Register("Onulet", func() Card {
		return NewCreature("Onulet", "{3}", 2, 2,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithAbility(
				NewTriggered(EvtCreatureDied, false,
					GainLife(2),
				).SetCondition(IsThisSource),
			),
		)
	})

	// Ornithopter {0}
	// Artifact Creature — Thopter
	// Flying
	Register("Ornithopter", func() Card {
		return NewCreature("Ornithopter", "{0}", 0, 2,
			WithSubTypes("Thopter"),
			WithCardType(TypeArtifact),
			WithKeyword(Flying),
		)
	})

	// Primal Clay {4}
	// Artifact Creature — Shapeshifter
	// As Primal Clay enters the battlefield, it becomes your choice of a 3/3 artifact creature,
	// a 2/2 artifact creature with flying, or a 1/6 Wall artifact creature with defender.
	Register("Primal Clay", func() Card {
		c := NewCreature("Primal Clay", "{4}", 3, 3,
			WithSubTypes("Shapeshifter"),
			WithCardType(TypeArtifact),
			WithAbility(ETBEffect(FuncEffect("choose form on ETB",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					mode := g.ModeValue()
					switch mode {
					case 1: // 2/2 with flying
						perm.Card.SetBasePT(2, 2)
						perm.GrantBaseAttr(Flying)
					case 2: // 1/6 Wall with defender
						perm.Card.SetBasePT(1, 6)
						if bc, ok := perm.Card.(*BaseCard); ok {
							bc.AddSubType("Wall")
						}
						perm.GrantBaseAttr(Defender)
						// case 0: 3/3 is the default
					}
					return nil
				}))),
		)
		c.SetModes([]string{"3/3 artifact creature", "2/2 artifact creature with flying", "1/6 Wall artifact creature with defender"})
		return c
	})

	// Shapeshifter {6}
	// Artifact Creature — Shapeshifter
	// As Shapeshifter enters the battlefield, choose a number between 0 and 7.
	// At the beginning of your upkeep, you may choose a number between 0 and 7.
	// Shapeshifter's power is equal to the last chosen number and its toughness is equal to
	// 7 minus that number.
	shapeshifterModes := []string{"0", "1", "2", "3", "4", "5", "6", "7"}
	Register("Shapeshifter", func() Card {
		c := NewCreature("Shapeshifter", "{6}", 0, 7, // base P/T overridden by continuous effect
			WithSubTypes("Shapeshifter"),
			WithCardType(TypeArtifact),
			// ETB: choose a number 0-7
			WithAbility(ETBEffect(FuncEffect("choose a number between 0 and 7",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					choice := g.ModeValue() // 0-7
					if choice < 0 {
						choice = 0
					}
					if choice > 7 {
						choice = 7
					}
					perm.StoredValue = choice
					return nil
				}))),
			// Upkeep: may re-choose
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("you may choose a new number between 0 and 7",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						if p.ChooseMayAbility("choose a new number for Shapeshifter") {
							choice := p.ChooseMode(shapeshifterModes, "choose number 0-7")
							if choice < 0 {
								choice = 0
							}
							if choice > 7 {
								choice = 7
							}
							perm.StoredValue = choice
						}
						return nil
					}), false,
			)),
			// Static: P/T = chosen / (7 - chosen)
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					chosen := src.StoredValue
					// Override base P/T to chosen / (7-chosen)
					src.BasePTOverride = &[2]int{chosen, 7 - chosen}
					return nil
				}),
			),
		)
		c.SetModes(shapeshifterModes)
		return c
	})

	// Su-Chi {4}
	// Artifact Creature — Construct
	// When Su-Chi dies, add {C}{C}{C}{C}.
	Register("Su-Chi", func() Card {
		return NewCreature("Su-Chi", "{4}", 4, 4,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithAbility(
				NewTriggered(EvtCreatureDied, false,
					FuncEffect("add {C}{C}{C}{C}",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p != nil {
								p.ManaPool().Add(Colorless, 4)
							}
							return nil
						}),
				).SetCondition(IsThisSource),
			),
		)
	})

	// Tetravus {6}
	// Artifact Creature — Construct
	// Flying
	// Tetravus enters the battlefield with three +1/+1 counters on it.
	// At the beginning of your upkeep, you may remove any number of +1/+1 counters from Tetravus.
	// If you do, create that many 1/1 colorless Tetravite artifact creature tokens with flying.
	// At the beginning of your upkeep, you may exile any number of tokens created with Tetravus.
	// If you do, put that many +1/+1 counters on Tetravus.
	Register("Tetravus", func() Card {
		return NewCreature("Tetravus", "{6}", 1, 1,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithKeyword(Flying),
			WithAbility(EntersWithNCounters(P1P1, 3)),
			// Remove counters → create Tetravite tokens
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("remove +1/+1 counters and create Tetravite tokens",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						maxCounters := int(src.Counters[P1P1])
						if maxCounters == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						n := p.ChooseNumber(0, maxCounters, "Remove how many +1/+1 counters from Tetravus?")
						if n == 0 {
							return nil
						}
						src.Counters[P1P1] -= uint8(n)
						for range n {
							token := NewToken("Tetravite", 1, 1,
								[]CardType{TypeArtifact, TypeCreature},
								[]string{"Tetravite"},
								Flying, AttrCantBeEnchanted,
							)
							token.SetOwner(controller)
							perm := g.PutOnBattlefield(token, controller)
							perm.CreatedBy = sourceID
						}
						return nil
					}), true,
			)),
			// Exile Tetravite tokens → add +1/+1 counters
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("exile Tetravite tokens and add +1/+1 counters",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						var tetravites []*Permanent
						tetravites = append(tetravites, g.FilterBattlefield(And(
							ControlledBy(controller),
							IsToken,
							CreatedByFilter(sourceID),
						))...)
						if len(tetravites) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						n := p.ChooseNumber(0, len(tetravites), "Exile how many Tetravite tokens?")
						if n == 0 {
							return nil
						}
						for i := range n {
							if i < len(tetravites) {
								g.ExilePermanent(tetravites[i])
							}
						}
						src.Counters[P1P1] += uint8(n)
						return nil
					}), true,
			)),
		)
	})

	// Triskelion {6}
	// Artifact Creature — Construct
	// Triskelion enters the battlefield with three +1/+1 counters on it.
	// Remove a +1/+1 counter from Triskelion: It deals 1 damage to any target.
	Register("Triskelion", func() Card {
		return NewCreature("Triskelion", "{6}", 1, 1,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			WithAbility(EntersWithNCounters(P1P1, 3)),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				RemoveCountersCost(P1P1, 1),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

	// Urza's Avenger {6}
	// Artifact Creature — Shapeshifter
	// {0}: Urza's Avenger gets -1/-1 and gains your choice of banding, flying, first strike,
	// or trample until end of turn.
	Register("Urza's Avenger", func() Card {
		c := NewCreature("Urza's Avenger", "{6}", 4, 4,
			WithSubTypes("Shapeshifter"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				FuncEffect("-1/-1 and gain chosen keyword until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						// Apply -1/-1 until end of turn
						debuff := TemporaryBoost(perm.ID(), -1, -1)
						debuff.SetSourceID(sourceID)
						g.AddContinuousEffect(debuff)
						// Choose keyword
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						modes := []string{"Banding", "Flying", "First strike", "Trample"}
						choice := p.ChooseMode(modes, "choose keyword")
						var attr Attr
						switch choice {
						case 0:
							attr = Banding
						case 1:
							attr = Flying
						case 2:
							attr = FirstStrike
						default:
							attr = Trample
						}
						kwEff := TargetEffect(LayerAbility, EndOfTurn, perm.ID(), func(g *Game, target *Permanent) error {
							g.GrantAttr(target.ID(), attr)
							return nil
						})
						kwEff.SetSourceID(sourceID)
						g.AddContinuousEffect(kwEff)
						g.ApplyContinuousEffects()
						return nil
					}),
				GenericCost(0),
			),
		)
		return c
	})

	// Wall of Spears {3}
	// Artifact Creature — Wall
	// Defender, first strike
	Register("Wall of Spears", func() Card {
		return NewCreature("Wall of Spears", "{3}", 2, 3,
			WithSubTypes("Wall"),
			WithCardType(TypeArtifact),
			WithKeyword(Defender),
			WithKeyword(FirstStrike),
		)
	})

	// Yotian Soldier {3}
	// Artifact Creature — Soldier
	// Vigilance
	Register("Yotian Soldier", func() Card {
		return NewCreature("Yotian Soldier", "{3}", 1, 4,
			WithSubTypes("Soldier"),
			WithCardType(TypeArtifact),
			WithKeyword(Vigilance),
		)
	})
}
