package legends

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

// avoid unused import error
var _ = fmt.Sprintf

func init() {
	registerEnchantments()
}

func registerEnchantments() {

	// Angelic Voices {2}{W}{W}
	// Enchantment
	// Creatures you control get +1/+1 as long as you control no nonartifact, nonwhite creatures.
	Register("Angelic Voices", func() Card {
		return NewEnchantment("Angelic Voices", "{2}{W}{W}",
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					controller := src.Controller
					// Check if controller controls any nonartifact, nonwhite creature
					for _, perm := range g.FilterBattlefield(And(IsCreature, ControlledBy(controller))) {
						if !perm.HasType(TypeArtifact) && !HasColorFilter(White).Match(perm, g) {
							return nil // has a nonartifact nonwhite creature, no bonus
						}
					}
					// Boost all creatures you control +1/+1
					for _, perm := range g.FilterBattlefield(And(IsCreature, ControlledBy(controller))) {
						perm.BoostPT(1, 1)
					}
					return nil
				}),
			),
		)
	})

	// Anti-Magic Aura {2}{U}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature can't be the target of spells and can't be enchanted by other Auras.
	Register("Anti-Magic Aura", func() Card {
		return NewAura("Anti-Magic Aura", "{2}{U}",
			WithStaticAbility(
				GrantAbilityToAttached(Shroud, AttachAura),
			),
		)
	})

	// Arboria {2}{G}{G}
	// World Enchantment
	// Creatures can't attack a player unless that player cast a spell or put a nontoken permanent onto the battlefield during their last turn.
	// XXX: needs tracking of per-player "cast a spell or put nontoken permanent" last turn
	Register("Arboria", func() Card {
		return NewEnchantment("Arboria", "{2}{G}{G}",
			WithSuperTypes(SuperWorld),
		)
	})

	// Backfire {U}
	// Enchantment — Aura
	// Enchant creature
	// Whenever enchanted creature deals damage to you, this Aura deals that much damage to that creature's controller.
	Register("Backfire", func() Card {
		return NewAura("Backfire", "{U}",
			WithAbility(
				NewTriggered(EvtDamageDealt, false,
					DealDamageToPlayers(EventAmountValue(), SelectAttachedController()),
				).SetConditionData(AttachedToDealsDamageToController{}),
			),
		)
	})

	// Blight {B}{B}
	// Enchantment — Aura
	// Enchant land
	// When enchanted land becomes tapped, destroy it.
	Register("Blight", func() Card {
		return NewAura("Blight", "{B}{B}",
			WithCastTarget(TargetLand()),
			WithAbility(
				WhenAttachedBecomesTappedTrigger(
					DestroyAttachedStep(), false,
				),
			),
		)
	})

	// Caverns of Despair {2}{R}{R}
	// World Enchantment
	// No more than two creatures can attack each combat.
	// No more than two creatures can block each combat.
	// XXX: needs engine support for attack/block count limits
	Register("Caverns of Despair", func() Card {
		return NewEnchantment("Caverns of Despair", "{2}{R}{R}",
			WithSuperTypes(SuperWorld),
		)
	})

	// Chains of Mephistopheles {1}{B}
	// Enchantment
	// If a player would draw a card except the first one they draw in each of their draw steps, that player discards a card instead. If the player discards a card this way, they draw a card. If the player doesn't discard a card this way, they mill a card.
	// XXX: needs draw replacement effect engine support
	Register("Chains of Mephistopheles", func() Card {
		return NewEnchantment("Chains of Mephistopheles", "{1}{B}")
	})

	// Cocoon {G}
	// Enchantment — Aura
	// Enchant creature you control
	// When this Aura enters, tap enchanted creature and put three pupa counters on this Aura.
	// Enchanted creature doesn't untap during your untap step if this Aura has a pupa counter on it.
	// At the beginning of your upkeep, remove a pupa counter from this Aura. If you can't, sacrifice it, put a +1/+1 counter on enchanted creature, and that creature gains flying.
	Register("Cocoon", func() Card {
		return NewAura("Cocoon", "{G}",
			// TODO: convert to pipeline — needs TapAttached + AddCountersToSource primitives
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("tap enchanted creature and add pupa counters",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.MutablePermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						target := g.FindPermanent(src.AttachedTo)
						if target != nil {
							g.TapPermanent(target)
						}
						src.AddCounter(Pupa, 3)
						return nil
					}), false,
			)),
			WithStaticAbility(
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					// Only prevent untap while this Aura has pupa counters
					if source.Counters[Pupa] > 0 {
						g.GrantAttr(target.ID(), AttrDoesNotUntap)
					}
					return nil
				}),
			),
			WithAbility(BeginningOfUpkeepTrigger(
				IfElse("remove pupa counter or sacrifice and boost",
					&SourceHasCounterCond{CounterType: Pupa, MinCount: 1},
					RemoveCounters(Pupa, 1),
					&PipelineData{
						Steps: []Effect{
							SnapshotAttached("host"),
							SacrificeSourceStep(),
							AddCounters(P1P1, Fixed(1)).Targeting(ToGathered("host")),
							GrantAttrToGathered("host", Flying),
						},
						Txt: "sacrifice Cocoon, boost creature",
					},
				), false,
			)),
		)
	})

	// Concordant Crossroads {G}
	// World Enchantment
	// All creatures have haste.
	Register("Concordant Crossroads", func() Card {
		return NewEnchantment("Concordant Crossroads", "{G}",
			WithSuperTypes(SuperWorld),
			WithStaticAbility(
				GrantKeywordToAll(Haste, IsCreature),
			),
		)
	})

	// Crevasse {2}{R}
	// Enchantment
	// Creatures with mountainwalk can be blocked as though they didn't have mountainwalk.
	Register("Crevasse", func() Card {
		return NewEnchantment("Crevasse", "{2}{R}",
			WithStaticAbility(
				NullifyLandwalkEffect(Mountainwalk),
			),
		)
	})

	// Deadfall {2}{G}
	// Enchantment
	// Creatures with forestwalk can be blocked as though they didn't have forestwalk.
	Register("Deadfall", func() Card {
		return NewEnchantment("Deadfall", "{2}{G}",
			WithStaticAbility(
				NullifyLandwalkEffect(Forestwalk),
			),
		)
	})

	// Demonic Torment {2}{B}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature can't attack.
	// Prevent all combat damage that would be dealt by enchanted creature.
	Register("Demonic Torment", func() Card {
		return NewAura("Demonic Torment", "{2}{B}",
			WithStaticAbility(
				PreventAttachedFromAttacking(AttachAura),
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					g.AddDamagePreventionRule(WithCombatOnly(), WithFrom(NewPermanentFilter("enchanted creature", func(p *Permanent, _ *Game) bool {
						return p.ID() == target.ID()
					})))
					return nil
				}),
			),
		)
	})

	// Divine Intervention {6}{W}{W}
	// Enchantment
	// This enchantment enters with two intervention counters on it.
	// At the beginning of your upkeep, remove an intervention counter from this enchantment.
	// When you remove the last intervention counter from this enchantment, the game is a draw.
	// XXX: needs game draw mechanic
	Register("Divine Intervention", func() Card {
		return NewEnchantment("Divine Intervention", "{6}{W}{W}",
			WithAbility(EntersBattlefieldTrigger(
				AddCounters(Intervention, Fixed(2)).Targeting(ToSource()), false,
			)),
			WithAbility(BeginningOfUpkeepTrigger(
				RemoveCounters(Intervention, 1), false,
				// XXX: needs game draw mechanic — when last counter is removed, game is a draw
			)),
		)
	})

	// Divine Transformation {2}{W}{W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +3/+3.
	Register("Divine Transformation", func() Card {
		return NewBoostAura("Divine Transformation", "{2}{W}{W}", 3, 3)
	})

	// Dream Coat {U}
	// Enchantment — Aura
	// Enchant creature
	// {0}: Enchanted creature becomes the color or colors of your choice. Activate only once each turn.
	Register("Dream Coat", func() Card {
		return NewAura("Dream Coat", "{U}",
			// TODO: convert to pipeline — needs ChooseColor + ColorOverrideAttached primitives
			WithActivatedAbility(
				FuncEffect("change enchanted creature's color",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						target := g.FindPermanent(src.AttachedTo)
						if target == nil {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						color := p.ChooseManaColor("Choose a color")
						g.AddContinuousEffect(ColorOverride(target.ID(), color))
						return nil
					}),
				GenericCost(0),
				WithMaxActivationsPerTurn(1),
			),
		)
	})

	// Equinox {W}
	// Enchantment — Aura
	// Enchant land
	// Enchanted land has "{T}: Counter target spell if it would destroy a land you control."
	// XXX: needs "would destroy a land" spell detection
	Register("Equinox", func() Card {
		return NewAura("Equinox", "{W}",
			WithCastTarget(TargetLand()),
		)
	})

	// Eternal Warrior {R}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature has vigilance.
	Register("Eternal Warrior", func() Card {
		return NewAura("Eternal Warrior", "{R}",
			WithStaticAbility(
				GrantAbilityToAttached(Vigilance, AttachAura),
			),
		)
	})

	// Field of Dreams {U}
	// World Enchantment
	// Players play with the top card of their libraries revealed.
	// XXX: needs revealed library top card engine support
	Register("Field of Dreams", func() Card {
		return NewEnchantment("Field of Dreams", "{U}",
			WithSuperTypes(SuperWorld),
		)
	})

	// Fortified Area {1}{W}{W}
	// Enchantment
	// Wall creatures you control get +1/+0 and have banding. (Any creatures with banding, and up to one without, can attack in a band. Bands are blocked as a group. If any creatures with banding you control are blocking or being blocked by a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
	Register("Fortified Area", func() Card {
		return NewEnchantment("Fortified Area", "{1}{W}{W}",
			WithStaticAbility(
				BoostControlledCreatures(1, 0, HasSubType("Wall")),
				GrantKeywordToControlled(Banding, HasSubType("Wall")),
			),
		)
	})

	// Gaseous Form {2}{U}
	// Enchantment — Aura
	// Enchant creature
	// Prevent all combat damage that would be dealt to and dealt by enchanted creature.
	Register("Gaseous Form", func() Card {
		return NewAura("Gaseous Form", "{2}{U}",
			WithStaticAbility(
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					g.AddDamagePreventionRule(WithCombatOnly(), WithFrom(IsID(target.ID())))
					g.AddDamagePreventionRule(WithCombatOnly(), WithTo(IsID(target.ID())))
					return nil
				}),
			),
		)
	})

	// Giant Strength {R}{R}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +2/+2.
	Register("Giant Strength", func() Card {
		return NewBoostAura("Giant Strength", "{R}{R}", 2, 2)
	})

	// Gravity Sphere {2}{R}
	// World Enchantment
	// All creatures lose flying.
	Register("Gravity Sphere", func() Card {
		return NewEnchantment("Gravity Sphere", "{2}{R}",
			WithSuperTypes(SuperWorld),
			WithStaticAbility(
				RevokeKeywordFromAll(Flying, AnyPermanent),
			),
		)
	})

	// Great Wall {2}{W}
	// Enchantment
	// Creatures with plainswalk can be blocked as though they didn't have plainswalk.
	Register("Great Wall", func() Card {
		return NewEnchantment("Great Wall", "{2}{W}",
			WithStaticAbility(
				NullifyLandwalkEffect(Plainswalk),
			),
		)
	})

	// Greater Realm of Preservation {1}{W}
	// Enchantment
	// {1}{W}: The next time a black or red source of your choice would deal damage to you this turn, prevent that damage.
	Register("Greater Realm of Preservation", func() Card {
		return NewEnchantment("Greater Realm of Preservation", "{1}{W}",
			// TODO: convert to pipeline — needs ChoosePermanent + AddPreventionShield primitives
			WithActivatedAbility(
				FuncEffect("prevent next damage from a black or red source of your choice",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Oracle: "a black or red source of your choice" — player picks a
						// specific black or red permanent; shield prevents next damage from it.
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						var candidates []*Permanent
						candidates = append(candidates, g.FilterBattlefield(Or(HasColorFilter(Black), HasColorFilter(Red)))...)
						if len(candidates) == 0 {
							return nil
						}
						chosen := p.ChoosePermanent(candidates, "Greater Realm of Preservation: choose a black or red source", g)
						if chosen == nil {
							return nil
						}
						g.AddReplacementEffect(&blackOrRedPreventionReplacement{
							playerID:       controller,
							chosenSourceID: chosen.ID(),
						})
						return nil
					}),
				ManaCostOf("{1}{W}"),
			),
		)
	})

	// Greed {3}{B}
	// Enchantment
	// {B}, Pay 2 life: Draw a card.
	Register("Greed", func() Card {
		return NewEnchantment("Greed", "{3}{B}",
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				ManaCostOf("{B}"),
				WithCost(LifePayCost(2)),
			),
		)
	})

	// Horror of Horrors {3}{B}{B}
	// Enchantment
	// Sacrifice a Swamp: Regenerate target black creature. (The next time that creature would be destroyed this turn, instead tap it, remove it from combat, and heal all damage on it.)
	Register("Horror of Horrors", func() Card {
		return NewEnchantment("Horror of Horrors", "{3}{B}{B}",
			WithActivatedAbility(
				RegenerateTarget(),
				SacrificeMatchingCost(And(IsLand, HasSubType("Swamp")), "Sacrifice a Swamp"),
				WithTarget(TargetCreature(HasColorFilter(Black))),
			),
		)
	})

	// Immolation {R}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +2/-2.
	Register("Immolation", func() Card {
		return NewBoostAura("Immolation", "{R}", 2, -2)
	})

	// Imprison {B}
	// Enchantment — Aura
	// Enchant creature
	// Whenever a player activates an ability of enchanted creature with {T} in its activation cost that isn't a mana ability, you may pay {1}. If you do, counter that ability. If you don't, destroy this Aura.
	// Whenever enchanted creature attacks or blocks, you may pay {1}. If you do, tap the creature, remove it from combat, and creatures it was blocking that had become blocked by only that creature this combat become unblocked. If you don't, destroy this Aura.
	// XXX: needs ability counter and tap-ability detection engine support
	Register("Imprison", func() Card {
		return NewAura("Imprison", "{B}")
	})

	// In the Eye of Chaos {2}{U}
	// World Enchantment
	// Whenever a player casts an instant spell, counter it unless that player pays {X}, where X is its mana value.
	Register("In the Eye of Chaos", func() Card {
		return NewEnchantment("In the Eye of Chaos", "{2}{U}",
			WithSuperTypes(SuperWorld),
			WithAbility(
				// TODO: convert to pipeline — needs CounterUnlessPayCMC (variable cost from spell's CMC)
				NewTriggered(EvtSpellCast, false,
					FuncEffect("counter instant unless pay CMC",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							so := g.FindStackObject(targets[0])
							if so == nil {
								return nil
							}
							cmc := so.Card.ManaCost().CMC()
							if cmc > 0 {
								if !g.TryPayCostFromLands(so.Controller, fmt.Sprintf("{%d}", cmc)) {
									g.CounterSpellOnStack(targets[0])
								}
							}
							return nil
						}),
				).
					SetConditionData(SpellCastIsType{Type: TypeInstant}),
			),
		)
	})

	// Infinite Authority {W}{W}{W}
	// Enchantment — Aura
	// Enchant creature
	// Whenever enchanted creature blocks or becomes blocked by a creature with toughness 3 or less, destroy the other creature at end of combat. At the beginning of the next end step, if that creature was destroyed this way, put a +1/+1 counter on the first creature.
	Register("Infinite Authority", func() Card {
		return NewAura("Infinite Authority", "{W}{W}{W}",
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					FuncEffect("destroy creature with toughness 3 or less at end of combat",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							src := g.FindPermanent(sourceID)
							if src == nil || src.AttachedTo == uuid.Nil {
								return nil
							}
							enchantedID := src.AttachedTo
							for _, group := range g.CombatGroups() {
								// Enchanted creature is attacking and blocked
								if group.AttackerID == enchantedID {
									for _, bid := range group.BlockerIDs {
										blocker := g.FindPermanent(bid)
										if blocker != nil && blocker.CurrentToughness(g) <= 3 {
											targetID := bid
											g.RegisterDelayedTrigger(&DelayedTrigger{
												EventType:  EvtEndOfCombat,
												TargetID:   targetID,
												Effects:    []Effect{DestroyTarget()},
												SourceID:   sourceID,
												Controller: controller,
											})
											// Delayed end step: add +1/+1 to enchanted if target died
											g.RegisterDelayedTrigger(&DelayedTrigger{
												EventType:  EvtEndStep,
												TargetID:   enchantedID,
												SourceID:   sourceID,
												Controller: controller,
												Effects: []Effect{FuncEffect(
													"put +1/+1 counter if creature was destroyed",
													EffectProperties{Outcome: OutcomeBenefit},
													func(g *Game, srcID, ctrl uuid.UUID, targets []uuid.UUID) error {
														if len(targets) == 0 {
															return nil
														}
														// Check if the destroyed creature is in graveyard
														destroyed := g.FindPermanent(targetID)
														if destroyed == nil {
															// Not on battlefield = was destroyed, add counter
															perm := g.MutablePermanent(targets[0])
															if perm != nil {
																perm.AddCounter(P1P1, 1)
															}
														}
														return nil
													},
												)},
											})
										}
									}
								}
								// Enchanted creature is blocking
								for _, bid := range group.BlockerIDs {
									if bid == enchantedID {
										attacker := g.FindPermanent(group.AttackerID)
										if attacker != nil && attacker.CurrentToughness(g) <= 3 {
											targetID := group.AttackerID
											g.RegisterDelayedTrigger(&DelayedTrigger{
												EventType:  EvtEndOfCombat,
												TargetID:   targetID,
												Effects:    []Effect{DestroyTarget()},
												SourceID:   sourceID,
												Controller: controller,
											})
											g.RegisterDelayedTrigger(&DelayedTrigger{
												EventType:  EvtEndStep,
												TargetID:   enchantedID,
												SourceID:   sourceID,
												Controller: controller,
												Effects: []Effect{FuncEffect(
													"put +1/+1 counter if creature was destroyed",
													EffectProperties{Outcome: OutcomeBenefit},
													func(g *Game, srcID, ctrl uuid.UUID, targets []uuid.UUID) error {
														if len(targets) == 0 {
															return nil
														}
														destroyed := g.FindPermanent(targetID)
														if destroyed == nil {
															perm := g.MutablePermanent(targets[0])
															if perm != nil {
																perm.AddCounter(P1P1, 1)
															}
														}
														return nil
													},
												)},
											})
										}
									}
								}
							}
							return nil
						}),
				).
					// TODO: convert to data condition
					SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return false
						}
						enchantedID := src.AttachedTo
						for _, group := range g.CombatGroups() {
							if group.AttackerID == enchantedID {
								for _, bid := range group.BlockerIDs {
									blocker := g.FindPermanent(bid)
									if blocker != nil && blocker.CurrentToughness(g) <= 3 {
										return true
									}
								}
							}
							for _, bid := range group.BlockerIDs {
								if bid == enchantedID {
									attacker := g.FindPermanent(group.AttackerID)
									if attacker != nil && attacker.CurrentToughness(g) <= 3 {
										return true
									}
								}
							}
						}
						return false
					}),
			),
		)
	})

	// Invoke Prejudice {U}{U}{U}{U}
	// Enchantment
	// Whenever an opponent casts a creature spell that doesn't share a color with a creature you control, counter that spell unless that player pays {X}, where X is its mana value.
	Register("Invoke Prejudice", func() Card {
		return NewEnchantment("Invoke Prejudice", "{U}{U}{U}{U}",
			WithAbility(
				// TODO: convert to pipeline — needs CounterUnlessPayCMC (variable cost from spell's CMC)
				NewTriggered(EvtSpellCast, false,
					FuncEffect("counter creature unless pay CMC",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							so := g.FindStackObject(targets[0])
							if so == nil {
								return nil
							}
							cmc := so.Card.ManaCost().CMC()
							if cmc > 0 {
								if !g.TryPayCostFromLands(so.Controller, fmt.Sprintf("{%d}", cmc)) {
									g.CounterSpellOnStack(targets[0])
								}
							}
							return nil
						}),
				).
					// TODO: convert to data condition
					SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
						// Only opponent's creature spells
						if evt.PlayerID == controllerID {
							return false
						}
						card := g.FindCardAnywhere(evt.SourceID)
						if card == nil || !card.HasType(TypeCreature) {
							return false
						}
						// Check if spell doesn't share a color with a creature you control
						myCreatures := g.FilterBattlefield(And(IsCreature, ControlledBy(controllerID)))
						cardColors := card.ManaCost().Colors()
						for _, color := range cardColors {
							for _, c := range myCreatures {
								if slices.Contains(c.Card.ManaCost().Colors(), color) {
									return false // shares a color
								}
							}
						}
						return true
					}),
			),
		)
	})

	// Kismet {3}{W}
	// Enchantment
	// Artifacts, creatures, and lands your opponents control enter tapped.
	Register("Kismet", func() Card {
		return NewEnchantment("Kismet", "{3}{W}",
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
				func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					controller := src.Controller
					g.AddEntersTappedRule(func(perm *Permanent) bool {
						if perm.Controller == controller {
							return false
						}
						return perm.HasType(TypeArtifact) || perm.HasType(TypeCreature) || perm.HasType(TypeLand)
					})
					return nil
				})),
		)
	})

	// Land Equilibrium {2}{U}{U}
	// Enchantment
	// If an opponent who controls at least as many lands as you do would put a land onto the battlefield, that player instead puts that land onto the battlefield then sacrifices a land of their choice.
	// XXX: needs land-play replacement effect engine support
	Register("Land Equilibrium", func() Card {
		return NewEnchantment("Land Equilibrium", "{2}{U}{U}")
	})

	// Land Tax {W}
	// Enchantment
	// At the beginning of your upkeep, if an opponent controls more lands than you, you may search your library for up to three basic land cards, reveal them, put them into your hand, then shuffle.
	Register("Land Tax", func() Card {
		return NewEnchantment("Land Tax", "{W}",
			WithAbility(
				// TODO: convert to pipeline — needs SearchLibrary primitive with up-to-N and filter
				BeginningOfUpkeepTrigger(
					FuncEffect("search for up to three basic land cards",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							opp := g.GetOpponent(controller)
							if opp == nil {
								return nil
							}
							myLands := g.CountBattlefield(And(IsLand, ControlledBy(controller)))
							oppLands := g.CountBattlefield(And(IsLand, ControlledBy(opp.PlayerID())))
							if oppLands <= myLands {
								return nil
							}
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							for range 3 {
								lib := p.Library()
								var candidates []Card
								for _, c := range lib {
									if isBasicLand(c) {
										candidates = append(candidates, c)
									}
								}
								if len(candidates) == 0 {
									break
								}
								card := p.ChooseCardFromLibrary(candidates, "search for basic land", g)
								if card == nil {
									break
								}
								newLib := make([]Card, 0, len(lib)-1)
								for _, c := range lib {
									if c.ID() != card.ID() {
										newLib = append(newLib, c)
									}
								}
								p.SetLibrary(newLib)
								p.AddToHand(card)
							}
							p.ShuffleLibrary()
							return nil
						}), true,
				),
			),
		)
	})

	// Land's Edge {1}{R}{R}
	// World Enchantment
	// Discard a card: If the discarded card was a land card, this enchantment deals 2 damage to target player or planeswalker. Any player may activate this ability.
	// XXX: needs discard-check-type conditional damage and any-player activation
	Register("Land's Edge", func() Card {
		return NewEnchantment("Land's Edge", "{1}{R}{R}",
			WithSuperTypes(SuperWorld),
			// TODO: convert to pipeline — needs discarded-card-type check conditional
			WithActivatedAbility(
				FuncEffect("deal 2 if discarded land",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(2)},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						// The discard cost already discarded; we check if it was a land
						// XXX: simplified - always deals 2 damage since we can't check discarded card type
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(targets[0])
						if p != nil {
							g.DealDamageToPlayer(p, 2, sourceID)
						}
						return nil
					}),
				DiscardCost(1),
				WithTarget(TargetPlayer()),
				WithAnyPlayerMay(),
			),
		)
	})

	// Lifeblood {2}{W}{W}
	// Enchantment
	// Whenever a Mountain an opponent controls becomes tapped, you gain 1 life.
	Register("Lifeblood", func() Card {
		return NewEnchantment("Lifeblood", "{2}{W}{W}",
			WithAbility(
				WhenOpponentPermanentBecomesTappedTrigger(
					GainLife(1), false,
					And(IsLand, HasSubType("Mountain")),
				),
			),
		)
	})

	// Living Plane {2}{G}{G}
	// World Enchantment
	// All lands are 1/1 creatures that are still lands.
	Register("Living Plane", func() Card {
		return NewEnchantment("Living Plane", "{2}{G}{G}",
			WithSuperTypes(SuperWorld),
			WithStaticAbility(
				AnimateLands(IsLand, 1, 1),
			),
		)
	})

	// Moat {2}{W}{W}
	// Enchantment
	// Creatures without flying can't attack.
	Register("Moat", func() Card {
		return NewEnchantment("Moat", "{2}{W}{W}",
			WithStaticAbility(
				RevokeKeywordFromAll(AttrCanAttack, NotHasKeywordFilter(Flying)),
			),
		)
	})

	// Nether Void {3}{B}
	// World Enchantment
	// Whenever a player casts a spell, counter it unless that player pays {3}.
	Register("Nether Void", func() Card {
		return NewEnchantment("Nether Void", "{3}{B}",
			WithSuperTypes(SuperWorld),
			WithAbility(
				NewTriggered(EvtSpellCast, false,
					CounterUnlessPay("{3}"),
				),
			),
		)
	})

	// Presence of the Master {3}{W}
	// Enchantment
	// Whenever a player casts an enchantment spell, counter it.
	Register("Presence of the Master", func() Card {
		return NewEnchantment("Presence of the Master", "{3}{W}",
			WithAbility(
				NewTriggered(EvtSpellCast, false,
					CounterSpell(),
				).
					SetConditionData(SpellCastIsType{Type: TypeEnchantment}),
			),
		)
	})

	// Puppet Master {U}{U}{U}
	// Enchantment — Aura
	// Enchant creature
	// When enchanted creature dies, return that card to its owner's hand. If that card is returned to its owner's hand this way, you may pay {U}{U}{U}. If you do, return this card to its owner's hand.
	Register("Puppet Master", func() Card {
		return NewAura("Puppet Master", "{U}{U}{U}",
			WithAbility(
				NewTriggered(EvtZoneChange, false,
					FuncEffect("return enchanted creature to hand",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							// Return the creature card to its owner's hand
							owner := g.GetPlayer(targets[0])
							if owner == nil {
								// targets[0] is the creature ID; find card in graveyard
								for _, pl := range g.AllPlayers() {
									for _, c := range pl.Graveyard() {
										if c.ID() == targets[0] {
											if removed, ok := pl.RemoveFromGraveyard(c.ID()); ok {
												ownerP := g.GetPlayer(c.Owner())
												if ownerP == nil {
													ownerP = pl
												}
												ownerP.AddToHand(removed)
												// Try to pay UUU to return Puppet Master
												if g.TryPayCostFromLands(controller, "{U}{U}{U}") {
													// Return Puppet Master from graveyard
													for _, c2 := range g.GetPlayer(controller).Graveyard() {
														if c2.Name() == "Puppet Master" {
															if rm, ok2 := g.GetPlayer(controller).RemoveFromGraveyard(c2.ID()); ok2 {
																g.GetPlayer(controller).AddToHand(rm)
															}
															break
														}
													}
												}
											}
											return nil
										}
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

	// Quagmire {2}{B}
	// Enchantment
	// Creatures with swampwalk can be blocked as though they didn't have swampwalk.
	Register("Quagmire", func() Card {
		return NewEnchantment("Quagmire", "{2}{B}",
			WithStaticAbility(
				NullifyLandwalkEffect(Swampwalk),
			),
		)
	})

	// Relic Bind {2}{U}
	// Enchantment — Aura
	// Enchant artifact an opponent controls
	// Whenever enchanted artifact becomes tapped, choose one —
	// • This Aura deals 1 damage to target player or planeswalker.
	// • Target player gains 1 life.
	// XXX: needs modal triggered ability and enchant-opponent's-artifact targeting
	Register("Relic Bind", func() Card {
		return NewAura("Relic Bind", "{2}{U}",
			WithCastTarget(TargetPermanentOpponentControls(IsArtifact)),
			WithAbility(
				WhenAttachedBecomesTappedTrigger(
					DealDamageToPlayers(Fixed(1), SelectAttachedController()), false,
				),
			),
		)
	})

	// Revelation {G}
	// World Enchantment
	// Players play with their hands revealed.
	// XXX: needs revealed hand engine support
	Register("Revelation", func() Card {
		return NewEnchantment("Revelation", "{G}",
			WithSuperTypes(SuperWorld),
		)
	})

	// Seeker {2}{W}{W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature can't be blocked except by artifact creatures and/or white creatures.
	Register("Seeker", func() Card {
		return NewAura("Seeker", "{2}{W}{W}",
			WithStaticAbility(
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					for _, p := range g.AllBattlefield() {
						if !p.HasType(TypeCreature) {
							continue
						}
						// Allow artifact creatures and white creatures to block
						if p.HasType(TypeArtifact) {
							continue
						}
						isWhite := slices.Contains(p.Colors(), White)
						if isWhite {
							continue
						}
						// This creature can't block the enchanted creature
						g.PreventBlockPair(p.ID(), target.ID())
					}
					return nil
				}),
			),
		)
	})

	// Spectral Cloak {U}{U}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature has shroud as long as it's untapped. (It can't be the target of spells or abilities.)
	Register("Spectral Cloak", func() Card {
		return NewAura("Spectral Cloak", "{U}{U}",
			WithStaticAbility(
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					if !target.Tapped {
						g.GrantAttr(target.ID(), Shroud)
					}
					return nil
				}),
			),
		)
	})

	// Spirit Link {W}
	// Enchantment — Aura
	// Enchant creature (Target a creature as you cast this. This card enters attached to that creature.)
	// Whenever enchanted creature deals damage, you gain that much life.
	Register("Spirit Link", func() Card {
		return NewAura("Spirit Link", "{W}",
			WithAbility(
				NewTriggered(EvtDamageDealt, false,
					GainLifeAmount(EventAmountValue()),
				).SetConditionData(SourceIsAttachedToEventSource{}),
			),
		)
	})

	// Spirit Shackle {B}{B}
	// Enchantment — Aura
	// Enchant creature
	// Whenever enchanted creature becomes tapped, put a -0/-2 counter on it.
	Register("Spirit Shackle", func() Card {
		return NewAura("Spirit Shackle", "{B}{B}",
			WithAbility(
				NewTriggered(EvtTapped, false,
					AddCounters(M0M2, Fixed(1)).Targeting(ToAttached()),
				).SetConditionData(SourceIsAttachedToEventSource{}),
			),
		)
	})

	// Spiritual Sanctuary {2}{W}{W}
	// Enchantment
	// At the beginning of each player's upkeep, if that player controls a Plains, they gain 1 life.
	Register("Spiritual Sanctuary", func() Card {
		return NewEnchantment("Spiritual Sanctuary", "{2}{W}{W}",
			WithAbility(
				// TODO: convert to pipeline — needs ActivePlayer selector + HasMatchingPermanent condition scoped to active player
				BeginningOfEachUpkeepTrigger(
					FuncEffect("gain 1 life if you control a Plains",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							activePlayer := g.ActivePlayerObj()
							if activePlayer == nil {
								return nil
							}
							// Check if active player controls a Plains
							if g.AnyBattlefield(And(IsLand, HasSubType("Plains"), ControlledBy(activePlayer.PlayerID()))) {
								g.PlayerGainLife(activePlayer, 1)
							}
							return nil
						}), false,
				),
			),
		)
	})

	// Storm World {R}
	// World Enchantment
	// At the beginning of each player's upkeep, this enchantment deals X damage to that player, where X is 4 minus the number of cards in their hand.
	Register("Storm World", func() Card {
		return NewEnchantment("Storm World", "{R}",
			WithSuperTypes(SuperWorld),
			WithAbility(
				BeginningOfEachUpkeepTrigger(HandSizeDamageEffect(4, false), false),
			),
		)
	})

	// Sylvan Library {1}{G}
	// Enchantment
	// At the beginning of your draw step, you may draw two additional cards. If you do, choose two cards in your hand drawn this turn. For each of those cards, pay 4 life or put the card on top of your library.
	// XXX: needs draw-step additional draw and per-card pay-or-put-back engine support
	Register("Sylvan Library", func() Card {
		return NewEnchantment("Sylvan Library", "{1}{G}")
	})

	// Takklemaggot {2}{B}{B}
	// Enchantment — Aura
	// Enchant creature
	// At the beginning of the upkeep of enchanted creature's controller, put a -0/-1 counter on that creature.
	// When enchanted creature dies, that creature's controller chooses a creature that this card could enchant. If the player does, return this card to the battlefield under your control attached to that creature. If they don't, return this card to the battlefield under your control as a non-Aura enchantment. It loses "enchant creature" and gains "At the beginning of that player's upkeep, this enchantment deals 1 damage to that player."
	// XXX: needs aura-to-enchantment mode change engine support
	Register("Takklemaggot", func() Card {
		return NewAura("Takklemaggot", "{2}{B}{B}",
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				AddCounters(M0M1, Fixed(1)).Targeting(ToAttached()), false,
			)),
		)
	})

	// The Abyss {3}{B}
	// World Enchantment
	// At the beginning of each player's upkeep, destroy target nonartifact creature that player controls of their choice. It can't be regenerated.
	Register("The Abyss", func() Card {
		return NewEnchantment("The Abyss", "{3}{B}",
			WithSuperTypes(SuperWorld),
			WithAbility(
				// TODO: convert to pipeline — needs ActivePlayerChoosePermanent + DestroyGatheredNoRegen
				BeginningOfEachUpkeepTrigger(
					FuncEffect("destroy nonartifact creature",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							activePlayer := g.ActivePlayerObj()
							if activePlayer == nil {
								return nil
							}
							creatures := g.FilterBattlefield(And(IsCreature, Not(IsArtifact), ControlledBy(activePlayer.PlayerID())))
							if len(creatures) == 0 {
								return nil
							}
							chosen := activePlayer.ChoosePermanent(creatures, "destroy", g)
							if chosen != nil {
								chosen.GrantBaseAttr(CantRegenerate)
								g.DestroyPermanent(chosen)
							}
							return nil
						}), false,
				),
			),
		)
	})

	// The Brute {1}{R}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +1/+0.
	// {R}{R}{R}: Regenerate enchanted creature.
	Register("The Brute", func() Card {
		return NewAura("The Brute", "{1}{R}",
			WithStaticAbility(
				BoostAttached(1, 0, AttachAura),
				GrantActivatedAbilityToAttached(
					RegenerateSource(),
					ManaCostOf("{R}{R}{R}"),
					AttachAura,
				),
			),
		)
	})

	// Undertow {2}{U}
	// Enchantment
	// Creatures with islandwalk can be blocked as though they didn't have islandwalk.
	Register("Undertow", func() Card {
		return NewEnchantment("Undertow", "{2}{U}",
			WithStaticAbility(
				NullifyLandwalkEffect(Islandwalk),
			),
		)
	})

	// Underworld Dreams {B}{B}{B}
	// Enchantment
	// Whenever an opponent draws a card, Underworld Dreams deals 1 damage to that player.
	Register("Underworld Dreams", func() Card {
		return NewEnchantment("Underworld Dreams", "{B}{B}{B}",
			WithAbility(NewTriggered(EvtCardDrawn, false,
				DealDamageToPlayers(Fixed(1), SelectEventController()),
			).SetConditionData(EventPlayerIsNotController{})),
		)
	})

	// Venarian Gold {X}{U}{U}
	// Enchantment — Aura
	// Enchant creature
	// When this Aura enters, tap enchanted creature and put X sleep counters on it.
	// Enchanted creature doesn't untap during its controller's untap step if it has a sleep counter on it.
	// At the beginning of the upkeep of enchanted creature's controller, remove a sleep counter from that creature.
	Register("Venarian Gold", func() Card {
		return NewAura("Venarian Gold", "{X}{U}{U}",
			// TODO: convert to pipeline — needs TapAttached + AddCountersToAttachedFromX primitives
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("tap creature and add sleep counters",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						target := g.MutablePermanent(src.AttachedTo)
						if target != nil {
							g.TapPermanent(target)
							x := g.XValue()
							if x > 0 {
								target.AddCounter(Sleep, x)
							}
						}
						return nil
					}), false,
			)),
			// Doesn't untap if it has a sleep counter
			WithStaticAbility(AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
				if target.Counters[Sleep] > 0 {
					g.GrantAttr(target.ID(), AttrDoesNotUntap)
				}
				return nil
			})),
			// TODO: convert to pipeline — needs RemoveCounterFromAttached primitive
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				FuncEffect("remove sleep counter",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						target := g.MutablePermanent(src.AttachedTo)
						if target != nil && target.Counters[Sleep] > 0 {
							target.RemoveCounter(Sleep, 1)
						}
						return nil
					}), false,
			)),
		)
	})

}

// blackOrRedPreventionReplacement prevents the next damage from a specific
// chosen black or red source. It implements ReplacementEffect as a one-shot shield.
type blackOrRedPreventionReplacement struct {
	playerID       uuid.UUID
	chosenSourceID uuid.UUID
	consumed       bool
	sourceID       uuid.UUID
}

func (r *blackOrRedPreventionReplacement) SourceID() uuid.UUID   { return r.sourceID }
func (r *blackOrRedPreventionReplacement) GetDuration() Duration { return EndOfTurn }

func (r *blackOrRedPreventionReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	if act.PlayerID() != r.playerID {
		return false
	}
	// Only prevent damage from the specific source the player chose
	return act.ActionSource() == r.chosenSourceID
}

func (r *blackOrRedPreventionReplacement) Replace(a Action, g *Game) Action {
	r.consumed = true
	return nil
}

func (r *blackOrRedPreventionReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *blackOrRedPreventionReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}
