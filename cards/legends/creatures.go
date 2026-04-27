package legends

import (
	"fmt"
	"slices"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

func init() {
	registerCreatures()
}

func registerCreatures() {

	// ===== WHITE CREATURES =====

	// Akron Legionnaire {6}{W}{W}
	// Creature — Giant Soldier
	// 8/4
	// Except for creatures named Akron Legionnaire and artifact creatures, creatures you control can't attack.
	Register("Akron Legionnaire", func() Card {
		return NewCreature("Akron Legionnaire", "{6}{W}{W}", 8, 4,
			WithSubTypes("Giant", "Soldier"),
			WithStaticAbility(
				RevokeAttrFromControlled(AttrCanAttack, And(Not(Named("Akron Legionnaire")), Not(IsArtifact))),
			),
		)
	})

	// Amrou Kithkin {W}{W}
	// Creature — Kithkin
	// 1/1
	// This creature can't be blocked by creatures with power 3 or greater.
	Register("Amrou Kithkin", func() Card {
		return NewCreature("Amrou Kithkin", "{W}{W}", 1, 1,
			WithSubTypes("Kithkin"),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) && p.CurrentPower(g) >= 3 {
						g.PreventBlockPair(p.ID(), sourceID)
					}
				}
				return nil
			})),
		)
	})

	// Clergy of the Holy Nimbus {W}
	// Creature — Human Cleric
	// 1/1
	// If this creature would be destroyed, regenerate it.
	// {1}: This creature can't be regenerated this turn. Only your opponents may activate this ability.
	Register("Clergy of the Holy Nimbus", func() Card {
		return NewCreature("Clergy of the Holy Nimbus", "{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			// Auto-regeneration: always have a regeneration shield available
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.AddRegenerationShield(sourceID)
				return nil
			})),
			// {1}: Can't be regenerated this turn. Only opponents may activate.
			WithActivatedAbility(
				Pipeline("can't be regenerated this turn",
					EffectProperties{Outcome: OutcomeDetriment},
					GrantKeyword(CantRegenerate).Targeting(ToSource()),
				),
				ManaCostOf("{1}"),
				WithOpponentOnlyMay(),
			),
		)
	})

	// D'Avenant Archer {2}{W}
	// Creature — Human Soldier Archer
	// 1/2
	// {T}: This creature deals 1 damage to target attacking or blocking creature.
	Register("D'Avenant Archer", func() Card {
		return NewCreature("D'Avenant Archer", "{2}{W}", 1, 2,
			WithSubTypes("Human", "Soldier", "Archer"),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapSourceCost(),
				WithTarget(TargetCreature(Or(IsAttacking, IsBlocking))),
			),
		)
	})

	// Elder Land Wurm {4}{W}{W}{W}
	// Creature — Dragon Wurm
	// 5/5
	// Defender, trample
	// When this creature blocks, it loses defender.
	Register("Elder Land Wurm", func() Card {
		return NewCreature("Elder Land Wurm", "{4}{W}{W}{W}", 5, 5,
			WithSubTypes("Dragon", "Wurm"),
			WithKeyword(Defender),
			WithKeyword(Trample),
			// TODO: convert to pipeline — needs RevokeBaseAttr(kw, SelectSource) primitive
			WithAbility(BlocksTrigger(FuncEffect("lose defender", EffectProperties{Outcome: OutcomeBenefit}, func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				perm := g.FindPermanent(sourceID)
				if perm != nil {
					perm.RevokeBaseAttr(Defender)
				}
				return nil
			}), false)),
		)
	})

	// Enchanted Being {1}{W}{W}
	// Creature — Human
	// 2/2
	// Prevent all combat damage that would be dealt to this creature by enchanted creatures.
	Register("Enchanted Being", func() Card {
		return NewCreature("Enchanted Being", "{1}{W}{W}", 2, 2,
			WithSubTypes("Human"),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				// Prevent combat damage from enchanted creatures (creatures with auras attached)
				g.AddDamagePreventionRule(
					WithCombatOnly(),
					WithFrom(NewPermanentFilter("enchanted creature", func(p *Permanent, g *Game) bool {
						if !p.HasType(TypeCreature) {
							return false
						}
						for _, attachID := range p.Attachments {
							attached := g.FindPermanent(attachID)
							if attached != nil && attached.HasType(TypeEnchantment) {
								return true
							}
						}
						return false
					})),
					WithTo(NewPermanentFilter("self", func(p *Permanent, _ *Game) bool {
						return p.ID() == sourceID
					})),
				)
				return nil
			})),
		)
	})

	// Ivory Guardians {4}{W}{W}
	// Creature — Giant Cleric
	// 3/3
	// Protection from red
	// Creatures named Ivory Guardians get +1/+1 as long as an opponent controls a nontoken red permanent.
	Register("Ivory Guardians", func() Card {
		return NewCreature("Ivory Guardians", "{4}{W}{W}", 3, 3,
			WithSubTypes("Giant", "Cleric"),
			WithAbility(ProtectionFromColor(Red)),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					// Check if an opponent controls a nontoken red permanent
					opponentHasRed := false
					for _, p := range g.AllBattlefield() {
						if p.Controller == src.Controller {
							continue
						}
						if p.IsToken {
							continue
						}
						if HasColorFilter(Red).Match(p, g) {
							opponentHasRed = true
							break
						}
					}
					if !opponentHasRed {
						return nil
					}
					// Boost all creatures named Ivory Guardians (including self)
					for _, p := range g.AllBattlefield() {
						if p.HasType(TypeCreature) && p.Card.Name() == "Ivory Guardians" {
							p.BoostPT(1, 1)
						}
					}
					return nil
				}),
			),
		)
	})

	// Keepers of the Faith {1}{W}{W}
	// Creature — Human Cleric
	// 2/3
	Register("Keepers of the Faith", func() Card {
		return NewCreature("Keepers of the Faith", "{1}{W}{W}", 2, 3,
			WithSubTypes("Human", "Cleric"),
		)
	})

	// Osai Vultures {1}{W}
	// Creature — Bird
	// 1/1
	// Flying
	// At the beginning of each end step, if a creature died this turn, put a carrion counter on this creature.
	// Remove two carrion counters from this creature: This creature gets +1/+1 until end of turn.
	Register("Osai Vultures", func() Card {
		return NewCreature("Osai Vultures", "{1}{W}", 1, 1,
			WithSubTypes("Bird"),
			WithKeyword(Flying),
			WithAbility(
				NewTriggered(EvtEndStep, false,
					AddCounters(Carrion, Fixed(1)).Targeting(ToSource()),
				).
					SetConditionData(CreatureDeathsOccurred{}),
			),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)).Targeting(ToSource()),
				RemoveCountersCost(Carrion, 2),
			),
		)
	})

	// Petra Sphinx {2}{W}{W}{W}
	// Creature — Sphinx
	// 3/4
	// {T}: Target player chooses a card name, then reveals the top card of their library. If that card has the chosen name, that player puts it into their hand. If it doesn't, the player puts it into their graveyard.
	// TODO: implement — needs engine support for card naming and reveal
	Register("Petra Sphinx", func() Card {
		return NewCreature("Petra Sphinx", "{2}{W}{W}{W}", 3, 4,
			WithSubTypes("Sphinx"),
		)
	})

	// Righteous Avengers {4}{W}
	// Creature — Human Soldier
	// 3/1
	// Plainswalk (This creature can't be blocked as long as defending player controls a Plains.)
	Register("Righteous Avengers", func() Card {
		return NewCreature("Righteous Avengers", "{4}{W}", 3, 1,
			WithSubTypes("Human", "Soldier"),
			WithKeyword(Plainswalk),
		)
	})

	// Thunder Spirit {1}{W}{W}
	// Creature — Elemental Spirit
	// 2/2
	// Flying, first strike
	Register("Thunder Spirit", func() Card {
		return NewCreature("Thunder Spirit", "{1}{W}{W}", 2, 2,
			WithSubTypes("Elemental", "Spirit"),
			WithKeyword(Flying),
			WithKeyword(FirstStrike),
		)
	})

	// Tundra Wolves {W}
	// Creature — Wolf
	// 1/1
	// First strike (This creature deals combat damage before creatures without first strike.)
	Register("Tundra Wolves", func() Card {
		return NewCreature("Tundra Wolves", "{W}", 1, 1,
			WithSubTypes("Wolf"),
			WithKeyword(FirstStrike),
		)
	})

	// Wall of Caltrops {1}{W}
	// Creature — Wall
	// 2/1
	// Defender (This creature can't attack.)
	// Whenever this creature blocks a creature, if at least one other Wall creature is blocking that creature and no non-Wall creatures are blocking that creature, this creature gains banding until end of turn. (If any creatures with banding you control are blocking a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by.)
	Register("Wall of Caltrops", func() Card {
		return NewCreature("Wall of Caltrops", "{1}{W}", 2, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					Pipeline("gain banding until end of turn",
						EffectProperties{Outcome: OutcomeBenefit},
						GrantKeyword(Banding).Targeting(ToSource()),
					),
				).
					// TODO: convert to data condition
					SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
						// Check if this Wall is blocking something
						for _, group := range g.CombatGroups() {
						isBlocking := false
						for _, bid := range group.BlockerIDs {
							if bid == sourceID {
								isBlocking = true
								break
							}
						}
						if !isBlocking {
							continue
						}
						// This Wall is blocking this attacker.
						// Check that at least one OTHER Wall is blocking it,
						// and no non-Wall creatures are blocking it.
						hasOtherWall := false
						hasNonWall := false
						for _, bid := range group.BlockerIDs {
							if bid == sourceID {
								continue
							}
							blocker := g.FindPermanent(bid)
							if blocker == nil {
								continue
							}
							if blocker.HasSubType("Wall") {
								hasOtherWall = true
							} else {
								hasNonWall = true
							}
						}
						if hasOtherWall && !hasNonWall {
							return true
						}
					}
					return false
				}),
			),
		)
	})

// Wall of Light {2}{W}
// Creature — Wall
// 1/5
// Defender (This creature can't attack.)
// Protection from black
	Register("Wall of Light", func() Card {
		return NewCreature("Wall of Light", "{2}{W}", 1, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(ProtectionFromColor(Black)),
		)
	})

	// ===== BLUE CREATURES =====

	// Azure Drake {3}{U}
	// Creature — Drake
	// 2/4
	// Flying
	Register("Azure Drake", func() Card {
		return NewCreature("Azure Drake", "{3}{U}", 2, 4,
			WithSubTypes("Drake"),
			WithKeyword(Flying),
		)
	})

	// Brine Hag {2}{U}{U}
	// Creature — Hag
	// 2/2
	// When this creature dies, change the base power and toughness of all creatures that dealt damage to it this turn to 0/2. (This effect lasts indefinitely.)
	Register("Brine Hag", func() Card {
		return NewCreature("Brine Hag", "{2}{U}{U}", 2, 2,
			WithSubTypes("Hag"),
			WithAbility(
				NewTriggered(EvtCreatureDied, false, FuncEffect(
					"change base P/T of all creatures that dealt damage to this to 0/2",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						sources := g.GetDamageSources(sourceID)
						if sources == nil {
							return nil
						}
						for srcID := range sources {
							perm := g.FindPermanent(srcID)
							if perm != nil && perm.HasType(TypeCreature) {
								permID := perm.ID()
								ce := FuncContinuousEffect(LayerPT, Indefinite, func(g *Game, _ uuid.UUID) error {
									p := g.FindPermanent(permID)
									if p != nil {
										p.BasePTOverride = &[2]int{0, 2}
									}
									return nil
								}, func(g *Game, _ uuid.UUID) bool {
									return g.FindPermanent(permID) != nil
								})
								ce.SetSourceID(sourceID)
								g.AddContinuousEffect(ce)
							}
						}
						return nil
					})).SetConditionData(EventSourceIsSelf{}),
			),
		)
	})

	// Devouring Deep {2}{U}
	// Creature — Fish
	// 1/2
	// Islandwalk (This creature can't be blocked as long as defending player controls an Island.)
	Register("Devouring Deep", func() Card {
		return NewCreature("Devouring Deep", "{2}{U}", 1, 2,
			WithSubTypes("Fish"),
			WithKeyword(Islandwalk),
		)
	})

	// Elder Spawn {4}{U}{U}{U}
	// Creature — Spawn
	// 6/6
	// At the beginning of your upkeep, unless you sacrifice an Island, sacrifice this creature and it deals 6 damage to you.
	// This creature can't be blocked by red creatures.
	Register("Elder Spawn", func() Card {
		return NewCreature("Elder Spawn", "{4}{U}{U}{U}", 6, 6,
			WithSubTypes("Spawn"),
			// Upkeep: sacrifice an Island or sacrifice self + 6 damage
			WithAbility(NewTriggered(EvtUpkeep, false, FuncEffect(
				"sacrifice an Island or sacrifice this creature and it deals 6 damage to you",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					player := g.GetPlayer(controller)
					if player == nil {
						return nil
					}
					candidates := g.FilterBattlefield(NewPermanentFilter("Island", func(p *Permanent, _ *Game) bool {
						return p.Controller == controller && p.HasSubType("Island")
					}))
					if len(candidates) > 0 && player.ChooseMayAbility("sacrifice an Island") {
						chosen := player.ChoosePermanent(candidates, "sacrifice Island", g)
						if chosen != nil {
							g.Sacrifice(chosen)
							return nil
						}
					}
					// No Island or chose not to — sacrifice self and deal 6 damage
					perm := g.FindPermanent(sourceID)
					if perm != nil {
						g.Sacrifice(perm)
						p := g.GetPlayer(controller)
						if p != nil {
							g.DealDamageToPlayer(p, 6, sourceID)
						}
					}
					return nil
				},
			)).SetConditionData(EventPlayerIsController{})),
			// Can't be blocked by red creatures
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) {
						if slices.Contains(p.Colors(), Red) {
							g.PreventBlockPair(p.ID(), sourceID)
						}
					}
				}
				return nil
			})),
		)
	})

	// Psionic Entity {4}{U}
	// Creature — Illusion
	// 2/2
	// {T}: This creature deals 2 damage to any target and 3 damage to itself.
	Register("Psionic Entity", func() Card {
		return NewCreature("Psionic Entity", "{4}{U}", 2, 2,
			WithSubTypes("Illusion"),
			WithActivatedAbility(
				Pipeline("deal 2 damage to any target and 3 damage to self",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(2)},
					DealDamageStep(Fixed(2)),
					DealDamageToSourceStep(Fixed(3)),
				),
				TapSourceCost(),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

	// Segovian Leviathan {4}{U}
	// Creature — Leviathan
	// 3/3
	// Islandwalk (This creature can't be blocked as long as defending player controls an Island.)
	Register("Segovian Leviathan", func() Card {
		return NewCreature("Segovian Leviathan", "{4}{U}", 3, 3,
			WithSubTypes("Leviathan"),
			WithKeyword(Islandwalk),
		)
	})

	// Time Elemental {2}{U}
	// Creature — Elemental
	// 0/2
	// When this creature attacks or blocks, at end of combat, sacrifice it and it deals 5 damage to you.
	// {2}{U}{U}, {T}: Return target permanent that isn't enchanted to its owner's hand.
	Register("Time Elemental", func() Card {
		return NewCreature("Time Elemental", "{2}{U}", 0, 2,
			WithSubTypes("Elemental"),
			// When attacks: register delayed end-of-combat sacrifice + 5 damage
			WithAbility(AttacksTrigger(
				RegisterDelayedTriggerStep(EvtEndOfCombat, "",
					SacrificeSource(),
					DealDamageToPlayers(Fixed(5), SelectController()),
				), false)),
			// When blocks: register delayed end-of-combat sacrifice + 5 damage
			WithAbility(BlocksTrigger(
				RegisterDelayedTriggerStep(EvtEndOfCombat, "",
					SacrificeSource(),
					DealDamageToPlayers(Fixed(5), SelectController()),
				), false)),
			// {2}{U}{U}, {T}: Return target permanent that isn't enchanted to its owner's hand.
			WithActivatedAbility(
				ReturnToHandTarget(),
				ManaCostOf("{2}{U}{U}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetPermanent(NewPermanentFilter("not enchanted", func(p *Permanent, g *Game) bool {
					for _, attachID := range p.Attachments {
						att := g.FindPermanent(attachID)
						if att != nil && att.HasType(TypeEnchantment) {
							return false
						}
					}
					return true
				}))),
			),
		)
	})

	// Wall of Vapor {3}{U}
	// Creature — Wall
	// 0/1
	// Defender (This creature can't attack.)
	// Prevent all damage that would be dealt to this creature by creatures it's blocking.
	Register("Wall of Vapor", func() Card {
		return NewCreature("Wall of Vapor", "{3}{U}", 0, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				// Prevent all damage from creatures this wall is blocking
				g.AddDamagePreventionRule(
					WithFrom(NewPermanentFilter("blocked by this wall", func(p *Permanent, g *Game) bool {
						for _, grp := range g.CombatGroups() {
							if grp.AttackerID == p.ID() {
								for _, bid := range grp.BlockerIDs {
									if bid == sourceID {
										return true
									}
								}
							}
						}
						return false
					})),
					WithTo(NewPermanentFilter("self", func(p *Permanent, _ *Game) bool {
						return p.ID() == sourceID
					})),
				)
				return nil
			})),
		)
	})

	// Wall of Wonder {2}{U}{U}
	// Creature — Wall
	// 1/5
	// Defender (This creature can't attack.)
	// {2}{U}{U}: This creature gets +4/-4 until end of turn and can attack this turn as though it didn't have defender.
	Register("Wall of Wonder", func() Card {
		return NewCreature("Wall of Wonder", "{2}{U}{U}", 1, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithActivatedAbility(
				FuncEffect(
					"this creature gets +4/-4 until end of turn and can attack this turn as though it didn't have defender",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// +4/-4 boost
						boost := TemporaryBoost(sourceID, 4, -4)
						boost.SetSourceID(sourceID)
						g.AddContinuousEffect(boost)
						// Revoke Defender so it can attack this turn
						canAttack := TargetEffect(LayerAbility, EndOfTurn, sourceID, func(g *Game, target *Permanent) error {
							g.RevokeAttr(target.ID(), Defender)
							return nil
						})
						canAttack.SetSourceID(sourceID)
						g.AddContinuousEffect(canAttack)
						return nil
					},
				),
				ManaCostOf("{2}{U}{U}"),
			),
		)
	})

	// Zephyr Falcon {1}{U}
	// Creature — Bird
	// 1/1
	// Flying, vigilance
	Register("Zephyr Falcon", func() Card {
		return NewCreature("Zephyr Falcon", "{1}{U}", 1, 1,
			WithSubTypes("Bird"),
			WithKeyword(Flying),
			WithKeyword(Vigilance),
		)
	})

	// ===== BLACK CREATURES =====

	// Abomination {3}{B}{B}
	// Creature — Horror
	// 2/6
	// Whenever this creature blocks or becomes blocked by a green or white creature, destroy that creature at end of combat.
	Register("Abomination", func() Card {
		return NewCreature("Abomination", "{3}{B}{B}", 2, 6,
			WithSubTypes("Horror"),
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					FuncEffect("destroy green/white creature in combat with Abomination at end of combat",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							isGreenOrWhite := func(p *Permanent) bool {
								for _, c := range p.Colors() {
									if c == Green || c == White {
										return true
									}
								}
								return false
							}
							for _, group := range g.CombatGroups() {
								if group.AttackerID == sourceID {
									for _, bid := range group.BlockerIDs {
										blocker := g.FindPermanent(bid)
										if blocker != nil && isGreenOrWhite(blocker) {
											g.RegisterDelayedTrigger(&DelayedTrigger{
												EventType:  EvtEndOfCombat,
												TargetID:   bid,
												Effects:    []Effect{DestroyTarget()},
												SourceID:   sourceID,
												Controller: controller,
											})
										}
									}
								}
								for _, bid := range group.BlockerIDs {
									if bid == sourceID {
										attacker := g.FindPermanent(group.AttackerID)
										if attacker != nil && isGreenOrWhite(attacker) {
											g.RegisterDelayedTrigger(&DelayedTrigger{
												EventType:  EvtEndOfCombat,
												TargetID:   group.AttackerID,
												Effects:    []Effect{DestroyTarget()},
												SourceID:   sourceID,
												Controller: controller,
											})
										}
									}
								}
							}
							return nil
						}),
				).
					SetConditionData(SourceInCombatWithMatchingCreature{Filter: Or(HasColorFilter(Green), HasColorFilter(White))}),
			),
		)
	})

	// Carrion Ants {2}{B}{B}
	// Creature — Insect
	// 0/1
	// {1}: This creature gets +1/+1 until end of turn.
	Register("Carrion Ants", func() Card {
		return NewCreature("Carrion Ants", "{2}{B}{B}", 0, 1,
			WithSubTypes("Insect"),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)).Targeting(ToSource()),
				GenericCost(1),
			),
		)
	})

	// Cosmic Horror {3}{B}{B}{B}
	// Creature — Horror
	// 7/7
	// First strike
	// At the beginning of your upkeep, destroy this creature unless you pay {3}{B}{B}{B}. If this creature is destroyed this way, it deals 7 damage to you.
	Register("Cosmic Horror", func() Card {
		return NewCreature("Cosmic Horror", "{3}{B}{B}{B}", 7, 7,
			WithSubTypes("Horror"),
			WithKeyword(FirstStrike),
			// At the beginning of your upkeep, destroy unless you pay {3}{B}{B}{B}.
			// If destroyed this way, deals 7 damage to you.
			WithAbility(NewTriggered(EvtUpkeep, false, FuncEffect(
				"destroy Cosmic Horror unless you pay {3}{B}{B}{B}; if destroyed, deal 7 damage",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					if g.TryPayCostFromLands(controller, "{3}{B}{B}{B}") {
						return nil // paid, keep the creature
					}
					perm := g.FindPermanent(sourceID)
					if perm != nil {
						g.DestroyPermanent(perm)
						// If actually destroyed (not indestructible), deal 7 damage
						if g.FindPermanent(sourceID) == nil {
							p := g.GetPlayer(controller)
							if p != nil {
								g.DealDamageToPlayer(p, 7, sourceID)
							}
						}
					}
					return nil
				},
			)).SetConditionData(EventPlayerIsController{})),
		)
	})

	// Cyclopean Mummy {1}{B}
	// Creature — Zombie
	// 2/1
	// When this creature dies, exile it.
	Register("Cyclopean Mummy", func() Card {
		return NewCreature("Cyclopean Mummy", "{1}{B}", 2, 1,
			WithSubTypes("Zombie"),
			WithAbility(NewTriggered(EvtCreatureDied, false,
				ExileSourceFromGraveyard(),
			).SetConditionData(EventSourceIsSelf{})),
		)
	})

	// Evil Eye of Orms-by-Gore {4}{B}
	// Creature — Eye
	// 3/6
	// Non-Eye creatures you control can't attack.
	// This creature can't be blocked except by Walls.
	Register("Evil Eye of Orms-by-Gore", func() Card {
		return NewCreature("Evil Eye of Orms-by-Gore", "{4}{B}", 3, 6,
			WithSubTypes("Eye"),
			WithKeyword(CantBeBlockedExceptByWalls),
			WithStaticAbility(
				RevokeAttrFromControlled(AttrCanAttack, Not(HasSubType("Eye"))),
			),
		)
	})

	// Fallen Angel {3}{B}{B}
	// Creature — Angel
	// 3/3
	// Flying
	// Sacrifice a creature: This creature gets +2/+1 until end of turn.
	Register("Fallen Angel", func() Card {
		return NewCreature("Fallen Angel", "{3}{B}{B}", 3, 3,
			WithSubTypes("Angel"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(2), Fixed(1)).Targeting(ToSource()),
				SacrificeCreatureCost(),
			),
		)
	})

	// Ghosts of the Damned {1}{B}{B}
	// Creature — Spirit
	// 0/2
	// {T}: Target creature gets -1/-0 until end of turn.
	Register("Ghosts of the Damned", func() Card {
		return NewCreature("Ghosts of the Damned", "{1}{B}{B}", 0, 2,
			WithSubTypes("Spirit"),
			WithActivatedAbility(
				Boost(Fixed(-1), Fixed(0)),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Giant Slug {1}{B}
	// Creature — Slug
	// 1/1
	// {5}: At the beginning of your next upkeep, choose a basic land type. This creature gains landwalk of the chosen type until the end of that turn. (It can't be blocked as long as defending player controls a land of that type.)
	Register("Giant Slug", func() Card {
		landTypes := []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}
		return NewCreature("Giant Slug", "{1}{B}", 1, 1,
			WithSubTypes("Slug"),
			WithActivatedAbility(
				FuncEffect("choose a basic land type at next upkeep; gain that landwalk",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:     EvtUpkeep,
							SourceID:      sourceID,
							Controller:    controller,
							MatchPlayerID: controller,
							Effects: []Effect{FuncEffect(
								"choose land type and gain landwalk",
								EffectProperties{Outcome: OutcomeBenefit},
								func(g *Game, srcID, ctrl uuid.UUID, _ []uuid.UUID) error {
									perm := g.FindPermanent(srcID)
									if perm == nil {
										return nil
									}
									p := g.GetPlayer(ctrl)
									if p == nil {
										return nil
									}
									mode := p.ChooseMode([]string{"Plains", "Island", "Swamp", "Mountain", "Forest"}, "choose a basic land type")
									if mode < 0 || mode >= len(landTypes) {
										mode = 0
									}
									chosen := landTypes[mode]
									attr := LandwalkAttr(chosen)
									if attr == 0 {
										return nil
									}
									ce := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, _ uuid.UUID) error {
										p := g.FindPermanent(srcID)
										if p != nil {
											g.GrantAttr(p.ID(), attr)
										}
										return nil
									})
									ce.SetSourceID(srcID)
									g.AddContinuousEffect(ce)
									return nil
								},
							)},
						})
						return nil
					},
				),
				GenericCost(5),
			),
		)
	})

	// Headless Horseman {2}{B}
	// Creature — Zombie Knight
	// 2/2
	Register("Headless Horseman", func() Card {
		return NewCreature("Headless Horseman", "{2}{B}", 2, 2,
			WithSubTypes("Zombie", "Knight"),
		)
	})

	// Hell's Caretaker {3}{B}
	// Creature — Horror
	// 1/1
	// {T}, Sacrifice a creature: Return target creature card from your graveyard to the battlefield. Activate only during your upkeep.
	Register("Hell's Caretaker", func() Card {
		return NewCreature("Hell's Caretaker", "{3}{B}", 1, 1,
			WithSubTypes("Horror"),
			WithActivatedAbility(
				ReturnFromGraveyardToBattlefield(),
				TapSourceCost(),
				WithCost(SacrificeCreatureCost()),
				WithTarget(TargetCreatureInYourGraveyard()),
				WithUpkeepOnly(),
			),
		)
	})

	// Infernal Medusa {3}{B}{B}
	// Creature — Gorgon
	// 2/4
	// Whenever this creature blocks a creature, destroy that creature at end of combat.
	// Whenever this creature becomes blocked by a non-Wall creature, destroy that creature at end of combat.
	Register("Infernal Medusa", func() Card {
		return NewCreature("Infernal Medusa", "{3}{B}{B}", 2, 4,
			WithSubTypes("Gorgon"),
			// When Medusa blocks: destroy the attacker at end of combat
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					ForEachAttackerBlockedBySource(
						RegisterDelayedTriggerStep(EvtEndOfCombat, "_target", DestroyTarget()),
						"destroy creature blocked by Infernal Medusa at end of combat",
					),
				).SetConditionData(SourceIsBlockingInCombat{}),
			),
			// When Medusa is blocked by a non-Wall: destroy that blocker at end of combat
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					ForEachBlockerOfSourceMatching(
						Not(HasSubType("Wall")),
						RegisterDelayedTriggerStep(EvtEndOfCombat, "_target", DestroyTarget()),
						"destroy non-Wall creatures blocking Infernal Medusa at end of combat",
					),
				).SetConditionData(SourceBlockedByCreatureMatching{Filter: Not(HasSubType("Wall"))}),
			),
		)
	})

	// Lesser Werewolf {3}{B}
	// Creature — Werewolf
	// 2/4
	// {B}: If this creature's power is 1 or more, it gets -1/-0 until end of turn and put a -0/-1 counter on target creature blocking or blocked by this creature. Activate only during the declare blockers step.
	Register("Lesser Werewolf", func() Card {
		return NewCreature("Lesser Werewolf", "{3}{B}", 2, 4,
			WithSubTypes("Werewolf"),
			WithActivatedAbility(
				FuncEffect("gets -1/-0; put -0/-1 counter on target creature blocking or blocked by this",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						if perm.CurrentPower(g) < 1 {
							return nil
						}
						// Verify target is blocking or blocked by this creature
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						inCombat := false
						for _, group := range g.CombatGroups() {
							if group.AttackerID == sourceID {
								for _, bid := range group.BlockerIDs {
									if bid == targetID {
										inCombat = true
										break
									}
								}
							} else if group.AttackerID == targetID {
								for _, bid := range group.BlockerIDs {
									if bid == sourceID {
										inCombat = true
										break
									}
								}
							}
							if inCombat {
								break
							}
						}
						if !inCombat {
							return nil
						}
						// -1/-0 until end of turn
						ce := TemporaryBoost(sourceID, -1, 0)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						// Put -0/-1 counter on target
						target := g.FindPermanent(targetID)
						if target != nil {
							target.AddCounter(M0M1, 1)
						}
						return nil
					},
				),
				ManaCostOf("{B}"),
				WithTarget(TargetCreatureBlockingOrBlockedBySource()),
			),
		)
	})

	// Lost Soul {1}{B}{B}
	// Creature — Spirit Minion
	// 2/1
	// Swampwalk (This creature can't be blocked as long as defending player controls a Swamp.)
	Register("Lost Soul", func() Card {
		return NewCreature("Lost Soul", "{1}{B}{B}", 2, 1,
			WithSubTypes("Spirit", "Minion"),
			WithKeyword(Swampwalk),
		)
	})

	// Mold Demon {5}{B}{B}
	// Creature — Fungus Demon
	// 6/6
	// When this creature enters, sacrifice it unless you sacrifice two Swamps.
	Register("Mold Demon", func() Card {
		return NewCreature("Mold Demon", "{5}{B}{B}", 6, 6,
			WithSubTypes("Fungus", "Demon"),
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"sacrifice this creature unless you sacrifice two Swamps",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					player := g.GetPlayer(controller)
					if player == nil {
						return nil
					}
					candidates := g.FilterBattlefield(NewPermanentFilter("Swamp", func(p *Permanent, _ *Game) bool {
						return p.Controller == controller && p.HasSubType("Swamp")
					}))
					if len(candidates) >= 2 && player.ChooseMayAbility("sacrifice two Swamps") {
						first := player.ChoosePermanent(candidates, "sacrifice Swamp (1 of 2)", g)
						if first != nil {
							g.Sacrifice(first)
							// Refresh candidates after first sacrifice
							remaining := g.FilterBattlefield(NewPermanentFilter("Swamp", func(p *Permanent, _ *Game) bool {
								return p.Controller == controller && p.HasSubType("Swamp")
							}))
							if len(remaining) > 0 {
								second := player.ChoosePermanent(remaining, "sacrifice Swamp (2 of 2)", g)
								if second != nil {
									g.Sacrifice(second)
									return nil
								}
							}
						}
					}
					// Can't or chose not to sacrifice two Swamps — sacrifice self
					perm := g.FindPermanent(sourceID)
					if perm != nil {
						g.Sacrifice(perm)
					}
					return nil
				},
			), false)),
		)
	})

	// Pit Scorpion {2}{B}
	// Creature — Scorpion
	// 1/1
	// Whenever this creature deals damage to a player, that player gets a poison counter. (A player with ten or more poison counters loses the game.)
	Register("Pit Scorpion", func() Card {
		return NewCreature("Pit Scorpion", "{2}{B}", 1, 1,
			WithSubTypes("Scorpion"),
			WithAbility(NewTriggered(EvtDamageDealt, false,
				PoisonTargetPlayer(1),
			).SetConditionData(EventSourceIsSelfDamageToPlayer{})),
		)
	})

	// Shimian Night Stalker {3}{B}{B}
	// Creature — Nightstalker
	// 4/4
	// {B}, {T}: All damage that would be dealt to you this turn by target attacking creature is dealt to this creature instead.
	Register("Shimian Night Stalker", func() Card {
		return NewCreature("Shimian Night Stalker", "{3}{B}{B}", 4, 4,
			WithSubTypes("Nightstalker"),
			WithActivatedAbility(
				// TODO: convert to pipeline — needs SetAttackerDamageRedirect data effect
				FuncEffect("redirect damage from target attacker to this creature",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						g.SetAttackerDamageRedirect(targets[0], sourceID)
						return nil
					}),
				ManaCostOf("{B}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	})

	// The Wretched {3}{B}{B}
	// Creature — Demon
	// 2/5
	// At end of combat, gain control of all creatures blocking this creature for as long as you control this creature.
	Register("The Wretched", func() Card {
		return NewCreature("The Wretched", "{3}{B}{B}", 2, 5,
			WithSubTypes("Demon"),
			WithAbility(
				NewTriggered(EvtEndOfCombat, false,
					FuncEffect("gain control of all creatures blocking this creature",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							for _, group := range g.CombatGroups() {
								if group.AttackerID == sourceID {
									for _, bid := range group.BlockerIDs {
										blocker := g.FindPermanent(bid)
										if blocker == nil {
											continue
										}
										blockerID := bid
										ce := FuncContinuousEffect(LayerControl, Indefinite, func(g *Game, srcID uuid.UUID) error {
											src := g.FindPermanent(srcID)
											if src == nil {
												return nil
											}
											target := g.FindPermanent(blockerID)
											if target != nil {
												target.Controller = src.Controller
											}
											return nil
										})
										ce.SetSourceID(sourceID)
										g.AddContinuousEffect(ce)
									}
								}
							}
							return nil
						}),
				).
					SetConditionData(SourceIsBlockedAttacker{}),
			),
		)
	})

	// Vampire Bats {B}
	// Creature — Bat
	// 0/1
	// Flying (This creature can't be blocked except by creatures with flying or reach.)
	// {B}: This creature gets +1/+0 until end of turn. Activate no more than twice each turn.
	Register("Vampire Bats", func() Card {
		return NewCreature("Vampire Bats", "{B}", 0, 1,
			WithSubTypes("Bat"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{B}"),
				WithMaxActivationsPerTurn(2),
			),
		)
	})

	// Walking Dead {1}{B}
	// Creature — Zombie
	// 1/1
	// {B}: Regenerate this creature.
	Register("Walking Dead", func() Card {
		return NewCreature("Walking Dead", "{1}{B}", 1, 1,
			WithSubTypes("Zombie"),
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{B}"),
			),
		)
	})

	// Wall of Putrid Flesh {2}{B}
	// Creature — Wall
	// 2/4
	// Defender (This creature can't attack.)
	// Protection from white
	// Prevent all damage that would be dealt to this creature by enchanted creatures.
	Register("Wall of Putrid Flesh", func() Card {
		return NewCreature("Wall of Putrid Flesh", "{2}{B}", 2, 4,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(ProtectionFromColor(White)),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.AddDamagePreventionRule(
					WithFrom(NewPermanentFilter("enchanted creature", func(p *Permanent, g *Game) bool {
						if !p.HasType(TypeCreature) {
							return false
						}
						for _, attachID := range p.Attachments {
							attached := g.FindPermanent(attachID)
							if attached != nil && attached.HasType(TypeEnchantment) {
								return true
							}
						}
						return false
					})),
					WithTo(NewPermanentFilter("self", func(p *Permanent, _ *Game) bool {
						return p.ID() == sourceID
					})),
				)
				return nil
			})),
		)
	})

	// Wall of Shadows {1}{B}{B}
	// Creature — Wall
	// 0/1
	// Defender (This creature can't attack.)
	// Prevent all damage that would be dealt to this creature by creatures it's blocking.
	// This creature can't be the target of spells that can target only Walls or of abilities that can target only Walls.
	// XXX: "can't be the target of spells that can target only Walls" not yet implemented
	Register("Wall of Shadows", func() Card {
		return NewCreature("Wall of Shadows", "{1}{B}{B}", 0, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.AddDamagePreventionRule(
					WithFrom(NewPermanentFilter("blocked by this wall", func(p *Permanent, g *Game) bool {
						for _, grp := range g.CombatGroups() {
							if grp.AttackerID == p.ID() {
								for _, bid := range grp.BlockerIDs {
									if bid == sourceID {
										return true
									}
								}
							}
						}
						return false
					})),
					WithTo(NewPermanentFilter("self", func(p *Permanent, _ *Game) bool {
						return p.ID() == sourceID
					})),
				)
				return nil
			})),
		)
	})

	// Wall of Tombstones {1}{B}
	// Creature — Wall
	// 0/1
	// Defender (This creature can't attack.)
	// At the beginning of your upkeep, change this creature's base toughness to 1 plus the number of creature cards in your graveyard. (This effect lasts indefinitely.)
	Register("Wall of Tombstones", func() Card {
		return NewCreature("Wall of Tombstones", "{1}{B}", 0, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("set base toughness to 1 plus creature cards in graveyard",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						creatureCount := 0
						for _, card := range p.Graveyard() {
							if card.HasType(TypeCreature) {
								creatureCount++
							}
						}
						newToughness := 1 + creatureCount
						ce := TargetEffect(LayerPT, Indefinite, sourceID, func(g *Game, target *Permanent) error {
							target.BasePTOverride = &[2]int{0, newToughness}
							return nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					}), false,
			)),
		)
	})

	// ===== RED CREATURES =====

	// Aerathi Berserker {2}{R}{R}{R}
	// Creature — Human Berserker
	// 2/4
	// Rampage 3 (Whenever this creature becomes blocked, it gets +3/+3 until end of turn for each creature blocking it beyond the first.)
	Register("Aerathi Berserker", func() Card {
		return NewCreature("Aerathi Berserker", "{2}{R}{R}{R}", 2, 4,
			WithSubTypes("Human", "Berserker"),
			WithAbility(RampageTrigger(3)),
		)
	})

	// Beasts of Bogardan {4}{R}
	// Creature — Beast
	// 3/3
	// Protection from red
	// This creature gets +1/+1 as long as an opponent controls a nontoken white permanent.
	Register("Beasts of Bogardan", func() Card {
		return NewCreature("Beasts of Bogardan", "{4}{R}", 3, 3,
			WithSubTypes("Beast"),
			WithAbility(ProtectionFromColor(Red)),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.AllBattlefield() {
						if p.Controller == src.Controller {
							continue
						}
						if p.IsToken {
							continue
						}
						if HasColorFilter(White).Match(p, g) {
							src.BoostPT(1, 1)
							return nil
						}
					}
					return nil
				}),
			),
		)
	})

	// Blazing Effigy {1}{R}
	// Creature — Elemental
	// 0/3
	// When this creature dies, it deals X damage to target creature, where X is 3 plus the amount of damage dealt to this creature this turn by other sources named Blazing Effigy.
	Register("Blazing Effigy", func() Card {
		return NewCreature("Blazing Effigy", "{1}{R}", 0, 3,
			WithSubTypes("Elemental"),
			WithAbility(
				NewTriggered(EvtCreatureDied, false, FuncEffect(
					"deal 3+ damage to target creature",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						creatures := g.FilterBattlefield(IsCreature)
						if len(creatures) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						target := p.ChoosePermanent(creatures, "choose a creature to deal damage to", g)
						if target == nil {
							return nil
						}
						// XXX: should be 3 plus damage dealt to this creature by other Blazing Effigies this turn;
						// needs per-source damage tracking by card name
						g.DealDamageToPermanent(target, 3, sourceID)
						return nil
					}),
				).SetConditionData(EventSourceIsSelf{}),
			),
		)
	})

	// Crimson Kobolds {0}
	// Creature — Kobold
	// 0/1
	Register("Crimson Kobolds", func() Card {
		return NewCreature("Crimson Kobolds", "{0}", 0, 1,
			WithSubTypes("Kobold"),
		)
	})

	// Crimson Manticore {2}{R}{R}
	// Creature — Manticore
	// 2/2
	// Flying
	// {R}, {T}: This creature deals 1 damage to target attacking or blocking creature.
	Register("Crimson Manticore", func() Card {
		return NewCreature("Crimson Manticore", "{2}{R}{R}", 2, 2,
			WithSubTypes("Manticore"),
			WithKeyword(Flying),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				ManaCostOf("{R}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature(Or(IsAttacking, IsBlocking))),
			),
		)
	})

	// Crookshank Kobolds {0}
	// Creature — Kobold
	// 0/1
	Register("Crookshank Kobolds", func() Card {
		return NewCreature("Crookshank Kobolds", "{0}", 0, 1,
			WithSubTypes("Kobold"),
		)
	})

	// Firestorm Phoenix {4}{R}{R}
	// Creature — Phoenix
	// 3/2
	// Flying
	// If this creature would die, return it to its owner's hand instead. Until that player's next turn, that player plays with that card revealed in their hand and can't play it.
	// XXX: Should be a replacement effect ("would die... instead") not a dies trigger. Requires engine support for "would be put into graveyard from battlefield" replacements that cover both destroy effects and lethal-damage state-based actions. With current implementation, other dies triggers incorrectly see this creature die before it returns to hand.
	// XXX: "plays with that card revealed and can't play it until next turn" restriction not enforced.
	Register("Firestorm Phoenix", func() Card {
		return NewCreature("Firestorm Phoenix", "{4}{R}{R}", 3, 2,
			WithSubTypes("Phoenix"),
			WithKeyword(Flying),
			WithAbility(NewTriggered(EvtCreatureDied, false,
				ReturnSourceToHand(),
			).SetConditionData(EventSourceIsSelf{})),
		)
	})

	// Frost Giant {3}{R}{R}{R}
	// Creature — Giant
	// 4/4
	// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	Register("Frost Giant", func() Card {
		return NewCreature("Frost Giant", "{3}{R}{R}{R}", 4, 4,
			WithSubTypes("Giant"),
			WithAbility(RampageTrigger(2)),
		)
	})

	// Hyperion Blacksmith {1}{R}{R}
	// Creature — Human Artificer
	// 2/2
	// {T}: You may tap or untap target artifact an opponent controls.
	Register("Hyperion Blacksmith", func() Card {
		return NewCreature("Hyperion Blacksmith", "{1}{R}{R}", 2, 2,
			WithSubTypes("Human", "Artificer"),
			WithActivatedAbility(
				TapOrUntapTarget(),
				TapSourceCost(),
				WithTarget(TargetPermanentOpponentControls(IsArtifact)),
			),
		)
	})

	// Kobold Drill Sergeant {1}{R}
	// Creature — Kobold Soldier
	// 1/2
	// Other Kobold creatures you control get +0/+1 and have trample.
	Register("Kobold Drill Sergeant", func() Card {
		return NewCreature("Kobold Drill Sergeant", "{1}{R}", 1, 2,
			WithSubTypes("Kobold", "Soldier"),
			WithStaticAbility(
				BoostOtherControlledCreatures(0, 1, HasSubType("Kobold")),
				GrantKeywordToOtherControlled(Trample, HasSubType("Kobold")),
			),
		)
	})

	// Kobold Overlord {1}{R}
	// Creature — Kobold
	// 1/2
	// First strike
	// Other Kobold creatures you control have first strike.
	Register("Kobold Overlord", func() Card {
		return NewCreature("Kobold Overlord", "{1}{R}", 1, 2,
			WithSubTypes("Kobold"),
			WithKeyword(FirstStrike),
			WithStaticAbility(
				GrantKeywordToOtherControlled(FirstStrike, HasSubType("Kobold")),
			),
		)
	})

	// Kobold Taskmaster {1}{R}
	// Creature — Kobold
	// 1/2
	// Other Kobold creatures you control get +1/+0.
	Register("Kobold Taskmaster", func() Card {
		return NewCreature("Kobold Taskmaster", "{1}{R}", 1, 2,
			WithSubTypes("Kobold"),
			WithStaticAbility(
				BoostOtherControlledCreatures(1, 0, HasSubType("Kobold")),
			),
		)
	})

	// Kobolds of Kher Keep {0}
	// Creature — Kobold
	// 0/1
	Register("Kobolds of Kher Keep", func() Card {
		return NewCreature("Kobolds of Kher Keep", "{0}", 0, 1,
			WithSubTypes("Kobold"),
		)
	})

	// Mountain Yeti {2}{R}{R}
	// Creature — Yeti
	// 3/3
	// Mountainwalk (This creature can't be blocked as long as defending player controls a Mountain.)
	// Protection from white
	Register("Mountain Yeti", func() Card {
		return NewCreature("Mountain Yeti", "{2}{R}{R}", 3, 3,
			WithSubTypes("Yeti"),
			WithKeyword(Mountainwalk),
			WithAbility(ProtectionFromColor(White)),
		)
	})

	// Primordial Ooze {R}
	// Creature — Ooze
	// 1/1
	// This creature attacks each combat if able.
	// At the beginning of your upkeep, put a +1/+1 counter on this creature. Then you may pay {X}, where X is the number of +1/+1 counters on it. If you don't, tap this creature and it deals X damage to you.
	Register("Primordial Ooze", func() Card {
		return NewCreature("Primordial Ooze", "{R}", 1, 1,
			WithSubTypes("Ooze"),
			WithKeyword(AttrMustAttack),
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("add +1/+1 counter, pay or tap and deal damage",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						perm.AddCounter(P1P1, 1)
						counters := int(perm.Counters[P1P1])
						cost := fmt.Sprintf("{%d}", counters)
						player := g.GetPlayer(controller)
						paid := false
						if player != nil && player.ChooseMayAbility(fmt.Sprintf("pay {%d}", counters)) {
							paid = g.TryPayCostFromLands(controller, cost)
						}
						if !paid {
							g.TapPermanent(perm)
							if player != nil {
								g.DealDamageToPlayer(player, counters, sourceID)
							}
						}
						return nil
					}), false,
			)),
		)
	})

	// Quarum Trench Gnomes {3}{R}
	// Creature — Gnome
	// 1/1
	// {T}: If target Plains is tapped for mana, it produces colorless mana instead of white mana. (This effect lasts indefinitely.)
	// TODO: implement — needs engine support for mana production replacement on specific lands
	Register("Quarum Trench Gnomes", func() Card {
		return NewCreature("Quarum Trench Gnomes", "{3}{R}", 1, 1,
			WithSubTypes("Gnome"),
		)
	})

	// Raging Bull {2}{R}
	// Creature — Ox
	// 2/2
	Register("Raging Bull", func() Card {
		return NewCreature("Raging Bull", "{2}{R}", 2, 2,
			WithSubTypes("Ox"),
		)
	})

	// Spinal Villain {2}{R}
	// Creature — Beast
	// 1/2
	// {T}: Destroy target blue creature.
	Register("Spinal Villain", func() Card {
		return NewCreature("Spinal Villain", "{2}{R}", 1, 2,
			WithSubTypes("Beast"),
			WithActivatedAbility(
				DestroyTarget(),
				TapSourceCost(),
				WithTarget(TargetCreature(HasColorFilter(Blue))),
			),
		)
	})

	// Tempest Efreet {1}{R}{R}{R}
	// Creature — Efreet
	// 3/3
	// Remove this card from your deck before playing if you're not playing for ante.
	// {T}, Sacrifice this creature: Target opponent may pay 10 life. If that player doesn't, they reveal a card at random from their hand. Exchange ownership of the revealed card and Tempest Efreet. Put the revealed card into your hand and Tempest Efreet from anywhere into that player's graveyard. This change in ownership is permanent.
	// UNIMPLEMENTABLE: Ante mechanic — requires permanent ownership exchange between players.
	Register("Tempest Efreet", func() Card {
		return NewCreature("Tempest Efreet", "{1}{R}{R}{R}", 3, 3,
			WithSubTypes("Efreet"),
		)
	})

	// Wall of Dust {2}{R}
	// Creature — Wall
	// 1/4
	// Defender (This creature can't attack.)
	// Whenever this creature blocks a creature, that creature can't attack during its controller's next turn.
	Register("Wall of Dust", func() Card {
		return NewCreature("Wall of Dust", "{2}{R}", 1, 4,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(
				NewTriggered(EvtDeclaredBlocker, false,
					FuncEffect("blocked creature can't attack next turn",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) < 2 {
								return nil
							}
							attackerID := targets[1]
							// In a 2-player game, the attacker's controller's next turn is 2 turns later
							expiryTurn := g.CurrentTurn() + 2
							ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
								perm := g.FindPermanent(attackerID)
								if perm != nil {
									g.RevokeAttr(perm.ID(), AttrCanAttack)
								}
								return nil
							}, func(g *Game, _ uuid.UUID) bool {
								return g.CurrentTurn() <= expiryTurn && g.FindPermanent(attackerID) != nil
							})
							ce.SetSourceID(sourceID)
							g.AddContinuousEffect(ce)
							return nil
						})).
					SetCondition(func(evt *GameEvent, _ GameReader, sourceID, _ uuid.UUID) bool {
						return evt.SourceID == sourceID
					}),
			),
		)
	})

	// Wall of Earth {1}{R}
	// Creature — Wall
	// 0/6
	// Defender (This creature can't attack.)
	Register("Wall of Earth", func() Card {
		return NewCreature("Wall of Earth", "{1}{R}", 0, 6,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	})

	// Wall of Heat {2}{R}
	// Creature — Wall
	// 2/6
	// Defender (This creature can't attack.)
	Register("Wall of Heat", func() Card {
		return NewCreature("Wall of Heat", "{2}{R}", 2, 6,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	})

	// Wall of Opposition {3}{R}{R}
	// Creature — Wall
	// 0/6
	// Defender (This creature can't attack.)
	// {1}: This creature gets +1/+0 until end of turn.
	Register("Wall of Opposition", func() Card {
		return NewCreature("Wall of Opposition", "{3}{R}{R}", 0, 6,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				GenericCost(1),
			),
		)
	})

	// ===== GREEN CREATURES =====

	// Aisling Leprechaun {G}
	// Creature — Faerie
	// 1/1
	// Whenever this creature blocks or becomes blocked by a creature, that creature becomes green. (This effect lasts indefinitely.)
	Register("Aisling Leprechaun", func() Card {
		return NewCreature("Aisling Leprechaun", "{G}", 1, 1,
			WithSubTypes("Faerie"),
			WithAbility(
				NewTriggered(EvtBlockersDecl, false,
					FuncEffect("make creature in combat with this green indefinitely",
						EffectProperties{},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							for _, group := range g.CombatGroups() {
								if group.AttackerID == sourceID {
									// Aisling is attacking — make all blockers green
									for _, bid := range group.BlockerIDs {
										ce := ColorOverride(bid, Green)
										ce.SetSourceID(sourceID)
										g.AddContinuousEffect(ce)
									}
								}
								for _, bid := range group.BlockerIDs {
									if bid == sourceID {
										// Aisling is blocking — make attacker green
										ce := ColorOverride(group.AttackerID, Green)
										ce.SetSourceID(sourceID)
										g.AddContinuousEffect(ce)
									}
								}
							}
							return nil
						}),
				).
					SetConditionData(SourceInCombat{}),
			),
		)
	})

	// Barbary Apes {1}{G}
	// Creature — Ape
	// 2/2
	Register("Barbary Apes", func() Card {
		return NewCreature("Barbary Apes", "{1}{G}", 2, 2,
			WithSubTypes("Ape"),
		)
	})

	// Cat Warriors {1}{G}{G}
	// Creature — Cat Warrior
	// 2/2
	// Forestwalk (This creature can't be blocked as long as defending player controls a Forest.)
	Register("Cat Warriors", func() Card {
		return NewCreature("Cat Warriors", "{1}{G}{G}", 2, 2,
			WithSubTypes("Cat", "Warrior"),
			WithKeyword(Forestwalk),
		)
	})

	// Craw Giant {3}{G}{G}{G}{G}
	// Creature — Giant
	// 6/4
	// Trample
	// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	Register("Craw Giant", func() Card {
		return NewCreature("Craw Giant", "{3}{G}{G}{G}{G}", 6, 4,
			WithSubTypes("Giant"),
			WithKeyword(Trample),
			WithAbility(RampageTrigger(2)),
		)
	})

	// Durkwood Boars {4}{G}
	// Creature — Boar
	// 4/4
	Register("Durkwood Boars", func() Card {
		return NewCreature("Durkwood Boars", "{4}{G}", 4, 4,
			WithSubTypes("Boar"),
		)
	})

	// Elven Riders {3}{G}{G}
	// Creature — Elf
	// 3/3
	// This creature can't be blocked except by Walls and/or creatures with flying.
	Register("Elven Riders", func() Card {
		return NewCreature("Elven Riders", "{3}{G}{G}", 3, 3,
			WithSubTypes("Elf"),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) && !p.HasSubType("Wall") && !p.HasKeyword(Flying) {
						g.PreventBlockPair(p.ID(), sourceID)
					}
				}
				return nil
			})),
		)
	})

	// Emerald Dragonfly {1}{G}
	// Creature — Insect
	// 1/1
	// Flying
	// {G}{G}: This creature gains first strike until end of turn.
	Register("Emerald Dragonfly", func() Card {
		return NewCreature("Emerald Dragonfly", "{1}{G}", 1, 1,
			WithSubTypes("Insect"),
			WithKeyword(Flying),
			WithActivatedAbility(
				GrantKeyword(FirstStrike).Targeting(ToSource()),
				ManaCostOf("{G}{G}"),
			),
		)
	})

	// Fire Sprites {1}{G}
	// Creature — Faerie
	// 1/1
	// Flying
	// {G}, {T}: Add {R}.
	Register("Fire Sprites", func() Card {
		return NewCreature("Fire Sprites", "{1}{G}", 1, 1,
			WithSubTypes("Faerie"),
			WithKeyword(Flying),
			WithActivatedAbility(
				AddMana(Red, 1),
				ManaCostOf("{G}"),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Floral Spuzzem {3}{G}
	// Creature — Elemental
	// 2/2
	// Whenever this creature attacks and isn't blocked, you may destroy target artifact defending player controls. If you do, this creature assigns no combat damage this turn.
	Register("Floral Spuzzem", func() Card {
		return NewCreature("Floral Spuzzem", "{3}{G}", 2, 2,
			WithSubTypes("Elemental"),
			WithAbility(
				NewTriggered(EvtBlockersDecl, true, FuncEffect(
					"destroy target artifact defending player controls; assign no combat damage this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						// Find the defending player
						var defenderID uuid.UUID
						for _, group := range g.CombatGroups() {
							if group.AttackerID == sourceID {
								defenderID = group.DefenderID
								break
							}
						}
						if defenderID == uuid.Nil {
							return nil
						}
						// Find an artifact the defending player controls
						artifacts := g.FilterBattlefield(And(IsArtifact, ControlledBy(defenderID)))
						if len(artifacts) == 0 {
							return nil
						}
						// Choose one to destroy
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						target := p.ChoosePermanent(artifacts, "choose an artifact to destroy", g)
						if target == nil {
							return nil
						}
						g.DestroyPermanent(target)
						// Prevent combat damage from Floral Spuzzem this turn
						eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, _ uuid.UUID) error {
							g.AddDamagePreventionRule(WithFrom(NewPermanentFilter("Floral Spuzzem", func(p *Permanent, _ *Game) bool {
								return p.ID() == sourceID
							})))
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						return nil
					})).
					SetConditionData(SourceIsUnblockedAttacker{}),
			),
		)
	})

	// Giant Turtle {1}{G}{G}
	// Creature — Turtle
	// 2/4
	// Giant Turtle can't attack if it attacked during your last turn.
	Register("Giant Turtle", func() Card {
		return NewCreature("Giant Turtle", "{1}{G}{G}", 2, 4,
			WithSubTypes("Turtle"),
			WithAbility(AttacksTrigger(FuncEffect(
				"can't attack next turn",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					// Prevent attacking on controller's next turn (2 turns from now in 2-player)
					expiryTurn := g.CurrentTurn() + 2
					ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							g.RevokeAttr(perm.ID(), AttrCanAttack)
						}
						return nil
					}, func(g *Game, _ uuid.UUID) bool {
						return g.CurrentTurn() <= expiryTurn && g.FindPermanent(sourceID) != nil
					})
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					return nil
				}), false)),
		)
	})

	// Hornet Cobra {1}{G}{G}
	// Creature — Snake
	// 2/1
	// First strike
	Register("Hornet Cobra", func() Card {
		return NewCreature("Hornet Cobra", "{1}{G}{G}", 2, 1,
			WithSubTypes("Snake"),
			WithKeyword(FirstStrike),
		)
	})

	// Ichneumon Druid {1}{G}{G}
	// Creature — Human Druid
	// 1/1
	// Whenever an opponent casts an instant spell other than the first instant spell that player casts each turn, this creature deals 4 damage to that player.
	Register("Ichneumon Druid", func() Card {
		return NewCreature("Ichneumon Druid", "{1}{G}{G}", 1, 1,
			WithSubTypes("Human", "Druid"),
			WithAbility(
				NewTriggered(EvtSpellCast, false,
					FuncEffect("deal 4 damage to opponent who cast second+ instant",
						EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(4)},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) < 2 {
								return nil
							}
							casterID := targets[1]
							p := g.GetPlayer(casterID)
							if p != nil {
								g.DealDamageToPlayer(p, 4, sourceID)
							}
							return nil
						}),
				).
					SetConditionData(OpponentCastNthSpellOfType{Type: TypeInstant, MinCount: 2}),
			),
		)
	})

	// Killer Bees {1}{G}{G}
	// Creature — Insect
	// 0/1
	// Flying
	// {G}: This creature gets +1/+1 until end of turn.
	Register("Killer Bees", func() Card {
		return NewCreature("Killer Bees", "{1}{G}{G}", 0, 1,
			WithSubTypes("Insect"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)).Targeting(ToSource()),
				ManaCostOf("{G}"),
			),
		)
	})

	// Master of the Hunt {2}{G}{G}
	// Creature — Human
	// 2/2
	// {2}{G}{G}: Create a 1/1 green Wolf creature token named Wolves of the Hunt. It has "bands with other creatures named Wolves of the Hunt."
	Register("Master of the Hunt", func() Card {
		return NewCreature("Master of the Hunt", "{2}{G}{G}", 2, 2,
			WithSubTypes("Human"),
			WithActivatedAbility(
				// XXX: "bands with other creatures named Wolves of the Hunt" approximated as Banding
				CreateColoredToken("Wolves of the Hunt", 1, 1, []Color{Green}, []CardType{TypeCreature}, []string{"Wolf"}, Banding),
				ManaCostOf("{2}{G}{G}"),
			),
		)
	})

	// Moss Monster {3}{G}{G}
	// Creature — Elemental
	// 3/6
	Register("Moss Monster", func() Card {
		return NewCreature("Moss Monster", "{3}{G}{G}", 3, 6,
			WithSubTypes("Elemental"),
		)
	})

	// Pixie Queen {2}{G}{G}
	// Creature — Faerie
	// 1/1
	// Flying
	// {G}{G}{G}, {T}: Target creature gains flying until end of turn.
	Register("Pixie Queen", func() Card {
		return NewCreature("Pixie Queen", "{2}{G}{G}", 1, 1,
			WithSubTypes("Faerie"),
			WithKeyword(Flying),
			WithActivatedAbility(
				GrantKeyword(Flying),
				ManaCostOf("{G}{G}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Pradesh Gypsies {2}{G}
	// Creature — Human Nomad
	// 1/1
	// {1}{G}, {T}: Target creature gets -2/-0 until end of turn.
	Register("Pradesh Gypsies", func() Card {
		return NewCreature("Pradesh Gypsies", "{2}{G}", 1, 1,
			WithSubTypes("Human", "Nomad"),
			WithActivatedAbility(
				Boost(Fixed(-2), Fixed(0)),
				ManaCostOf("{1}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Rabid Wombat {2}{G}{G}
	// Creature — Wombat
	// 0/1
	// Vigilance
	// This creature gets +2/+2 for each Aura attached to it.
	Register("Rabid Wombat", func() Card {
		return NewCreature("Rabid Wombat", "{2}{G}{G}", 0, 1,
			WithSubTypes("Wombat"),
			WithKeyword(Vigilance),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					auraCount := 0
					for _, attID := range src.Attachments {
						att := g.FindPermanent(attID)
						if att != nil && att.HasSubType("Aura") {
							auraCount++
						}
					}
					src.BoostPT(2*auraCount, 2*auraCount)
					return nil
				}),
			),
		)
	})

	// Radjan Spirit {3}{G}
	// Creature — Spirit
	// 3/2
	// {T}: Target creature loses flying until end of turn.
	Register("Radjan Spirit", func() Card {
		return NewCreature("Radjan Spirit", "{3}{G}", 3, 2,
			WithSubTypes("Spirit"),
			WithActivatedAbility(
				Pipeline("target creature loses flying until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					RevokeKeywordFromTargetUntilEOT(Flying),
				),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Shelkin Brownie {1}{G}
	// Creature — Ouphe
	// 1/1
	// {T}: Target creature loses all "bands with other" abilities until end of turn.
	// XXX: "bands with other" approximated as Banding; this removes Banding which is the approximation
	Register("Shelkin Brownie", func() Card {
		return NewCreature("Shelkin Brownie", "{1}{G}", 1, 1,
			WithSubTypes("Ouphe"),
			WithActivatedAbility(
				Pipeline("target creature loses all bands with other abilities until end of turn",
					EffectProperties{Outcome: OutcomeDetriment},
					RevokeKeywordFromTargetUntilEOT(Banding),
				),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Whirling Dervish {G}{G}
	// Creature — Human Monk
	// 1/1
	// Protection from black
	// At the beginning of each end step, if this creature dealt damage to an opponent this turn, put a +1/+1 counter on it.
	Register("Whirling Dervish", func() Card {
		return NewCreature("Whirling Dervish", "{G}{G}", 1, 1,
			WithSubTypes("Human", "Monk"),
			WithAbility(ProtectionFromColor(Black)),
			WithAbility(DealsDamageToOpponentTrigger(
				AddCounters(P1P1, Fixed(1)).Targeting(ToSource()), false,
			)),
		)
	})

	// Willow Satyr {2}{G}{G}
	// Creature — Satyr
	// 1/1
	// You may choose not to untap this creature during your untap step.
	// {T}: Gain control of target legendary creature for as long as you control this creature and this creature remains tapped.
	Register("Willow Satyr", func() Card {
		return NewCreature("Willow Satyr", "{2}{G}{G}", 1, 1,
			WithSubTypes("Satyr"),
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				FuncEffect("gain control of target legendary creature while tapped",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						cancelled := false
						ce := FuncContinuousEffect(LayerControl, Indefinite, func(g *Game, srcID uuid.UUID) error {
							if cancelled {
								return nil
							}
							src := g.FindPermanent(srcID)
							if src == nil || !src.Tapped || src.Controller != controller {
								cancelled = true
								return nil
							}
							perm := g.FindPermanent(targetID)
							if perm != nil {
								perm.Controller = src.Controller
							}
							return nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					}),
				TapSourceCost(),
				WithTarget(TargetCreature(NewPermanentFilter("legendary creature", func(p *Permanent, _ *Game) bool {
					return p.Card.HasSuperType(SuperLegendary)
				}))),
			),
		)
	})

	// Wolverine Pack {2}{G}{G}
	// Creature — Wolverine
	// 2/4
	// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	Register("Wolverine Pack", func() Card {
		return NewCreature("Wolverine Pack", "{2}{G}{G}", 2, 4,
			WithSubTypes("Wolverine"),
			WithAbility(RampageTrigger(2)),
		)
	})

	// Wood Elemental {3}{G}
	// Creature — Elemental
	// */*
	// As this creature enters, sacrifice any number of untapped Forests.
	// Wood Elemental's power and toughness are each equal to the number of Forests sacrificed as it entered.
	// TODO: implement — needs engine support for "as enters" replacement effect with variable sacrifice count
	Register("Wood Elemental", func() Card {
		return NewCreature("Wood Elemental", "{3}{G}", 0, 0,
			WithSubTypes("Elemental"),
		)
	})

	// ===== MULTICOLOR CREATURES =====

	// Adun Oakenshield {B}{R}{G}
	// Legendary Creature — Human Knight
	// 1/2
	// {B}{R}{G}, {T}: Return target creature card from your graveyard to your hand.
	Register("Adun Oakenshield", func() Card {
		return NewCreature("Adun Oakenshield", "{B}{R}{G}", 1, 2,
			WithSubTypes("Human", "Knight"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				ReturnFromGraveyardToHandTarget(),
				ManaCostOf("{B}{R}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreatureInYourGraveyard()),
			),
		)
	})

	// Angus Mackenzie {G}{W}{U}
	// Legendary Creature — Human Cleric
	// 2/2
	// {G}{W}{U}, {T}: Prevent all combat damage that would be dealt this turn. Activate only before the combat damage step.
	Register("Angus Mackenzie", func() Card {
		return NewCreature("Angus Mackenzie", "{G}{W}{U}", 2, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				PreventAllCombatDamage(),
				ManaCostOf("{G}{W}{U}"),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Arcades Sabboth {2}{G}{G}{W}{W}{U}{U}
	// Legendary Creature — Elder Dragon
	// 7/7
	// Flying
	// At the beginning of your upkeep, sacrifice Arcades Sabboth unless you pay {G}{W}{U}.
	// Each untapped creature you control gets +0/+2 as long as it's not attacking.
	// {W}: Arcades Sabboth gets +0/+1 until end of turn.
	Register("Arcades Sabboth", func() Card {
		return NewCreature("Arcades Sabboth", "{2}{G}{G}{W}{W}{U}{U}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithAbility(SacrificeAtUpkeepUnlessPay("{G}{W}{U}")),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.AllBattlefield() {
						if p.Controller != src.Controller || !p.HasType(TypeCreature) {
							continue
						}
						if !p.Tapped && !g.IsAttackingInCombat(p.ID()) {
							p.BoostPT(0, 2)
						}
					}
					return nil
				}),
			),
			WithActivatedAbility(
				Boost(Fixed(0), Fixed(1)).Targeting(ToSource()),
				ManaCostOf("{W}"),
			),
		)
	})

	// Axelrod Gunnarson {4}{B}{B}{R}{R}
	// Legendary Creature — Giant
	// 5/5
	// Trample
	// Whenever a creature dealt damage by Axelrod Gunnarson this turn dies, you gain 1 life and Axelrod Gunnarson deals 1 damage to target player or planeswalker.
	Register("Axelrod Gunnarson", func() Card {
		return NewCreature("Axelrod Gunnarson", "{4}{B}{B}{R}{R}", 5, 5,
			WithSubTypes("Giant"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Trample),
			WithAbility(CreatureDealtDamageBySourceDiesTrigger(
				CompositeEffects("gain 1 life and deal 1 damage to target player",
					GainLife(1),
					DealDamage(Fixed(1)),
				),
				false,
			).AddTarget(TargetPlayer())),
		)
	})

	// Ayesha Tanaka {W}{W}{U}{U}
	// Legendary Creature — Human Artificer
	// 2/2
	// Banding (Any creatures with banding, and up to one without, can attack in a band. Bands are blocked as a group. If any creatures with banding you control are blocking or being blocked by a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
	// {T}: Counter target activated ability from an artifact source unless that ability's controller pays {W}. (Mana abilities can't be targeted.)
	// TODO: implement — needs engine support for countering activated abilities from artifact sources
	Register("Ayesha Tanaka", func() Card {
		return NewCreature("Ayesha Tanaka", "{W}{W}{U}{U}", 2, 2,
			WithSubTypes("Human", "Artificer"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Banding),
		)
	})

	// Barktooth Warbeard {4}{B}{R}{R}
	// Legendary Creature — Human Warrior
	// 6/5
	Register("Barktooth Warbeard", func() Card {
		return NewCreature("Barktooth Warbeard", "{4}{B}{R}{R}", 6, 5,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Bartel Runeaxe {3}{B}{R}{G}
	// Legendary Creature — Giant Warrior
	// 6/5
	// Vigilance
	// Bartel Runeaxe can't be the target of Aura spells.
	// XXX: "can't be target of Aura spells" is not enforced (needs engine support for aura targeting restriction)
	Register("Bartel Runeaxe", func() Card {
		return NewCreature("Bartel Runeaxe", "{3}{B}{R}{G}", 6, 5,
			WithSubTypes("Giant", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Vigilance),
		)
	})

	// Boris Devilboon {3}{B}{R}
	// Legendary Creature — Zombie Wizard
	// 2/2
	// {2}{B}{R}, {T}: Create a 1/1 black and red Demon creature token named Minor Demon.
	Register("Boris Devilboon", func() Card {
		return NewCreature("Boris Devilboon", "{3}{B}{R}", 2, 2,
			WithSubTypes("Zombie", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				CreateColoredToken("Minor Demon", 1, 1, []Color{Black, Red}, []CardType{TypeCreature}, []string{"Demon"}),
				ManaCostOf("{2}{B}{R}"),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Chromium {2}{W}{W}{U}{U}{B}{B}
	// Legendary Creature — Elder Dragon
	// 7/7
	// Flying
	// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	// At the beginning of your upkeep, sacrifice Chromium unless you pay {W}{U}{B}.
	Register("Chromium", func() Card {
		return NewCreature("Chromium", "{2}{W}{W}{U}{U}{B}{B}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithAbility(RampageTrigger(2)),
			WithAbility(SacrificeAtUpkeepUnlessPay("{W}{U}{B}")),
		)
	})

	// Dakkon Blackblade {2}{W}{U}{U}{B}
	// Legendary Creature — Human Warrior
	// */*
	// Dakkon Blackblade's power and toughness are each equal to the number of lands you control.
	Register("Dakkon Blackblade", func() Card {
		return NewCreature("Dakkon Blackblade", "{2}{W}{U}{U}{B}", 0, 0,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithStaticAbility(
				PTEqualsControlledCount(IsLand),
			),
		)
	})

	// Gabriel Angelfire {3}{G}{G}{W}{W}
	// Legendary Creature — Angel
	// 4/4
	// At the beginning of your upkeep, choose flying, first strike, trample, or rampage 3. Gabriel Angelfire gains that ability until your next upkeep. (Whenever a creature with rampage 3 becomes blocked, it gets +3/+3 until end of turn for each creature blocking it beyond the first.)
	Register("Gabriel Angelfire", func() Card {
		return NewCreature("Gabriel Angelfire", "{3}{G}{G}{W}{W}", 4, 4,
			WithSubTypes("Angel"),
			WithSuperTypes(SuperLegendary),
			WithAbility(BeginningOfUpkeepTrigger(FuncEffect(
				"choose flying, first strike, trample, or rampage 3",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					modes := []string{"Flying", "First strike", "Trample", "Rampage 3"}
					choice := p.ChooseMode(modes, "Gabriel Angelfire")
					// "Until your next upkeep" = active for ~2 turns in 2-player
					expiryTurn := g.CurrentTurn() + 2
					switch choice {
					case 0: // Flying
						ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
							g.GrantAttr(sourceID, Flying)
							return nil
						}, func(g *Game, _ uuid.UUID) bool {
							return g.CurrentTurn() <= expiryTurn && g.FindPermanent(sourceID) != nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
					case 1: // First strike
						ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
							g.GrantAttr(sourceID, FirstStrike)
							return nil
						}, func(g *Game, _ uuid.UUID) bool {
							return g.CurrentTurn() <= expiryTurn && g.FindPermanent(sourceID) != nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
					case 2: // Trample
						ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
							g.GrantAttr(sourceID, Trample)
							return nil
						}, func(g *Game, _ uuid.UUID) bool {
							return g.CurrentTurn() <= expiryTurn && g.FindPermanent(sourceID) != nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
					case 3: // Rampage 3
						ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
							p := g.FindPermanent(sourceID)
							if p == nil {
								return nil
							}
							rt := RampageTrigger(3)
							rt.SetSource(sourceID)
							rt.SetController(p.Controller)
							p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(rt))
							return nil
						}, func(g *Game, _ uuid.UUID) bool {
							return g.CurrentTurn() <= expiryTurn && g.FindPermanent(sourceID) != nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
					}
					return nil
				}), false)),
		)
	})

	// Gosta Dirk {3}{W}{W}{U}{U}
	// Legendary Creature — Human Warrior
	// 4/4
	// First strike
	// Creatures with islandwalk can be blocked as though they didn't have islandwalk.
	Register("Gosta Dirk", func() Card {
		return NewCreature("Gosta Dirk", "{3}{W}{W}{U}{U}", 4, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(FirstStrike),
			WithStaticAbility(
				NullifyLandwalkEffect(Islandwalk),
			),
		)
	})

	// Gwendlyn Di Corci {U}{B}{B}{R}
	// Legendary Creature — Human Rogue
	// 3/5
	// {T}: Target player discards a card at random. Activate only during your turn.
	Register("Gwendlyn Di Corci", func() Card {
		return NewCreature("Gwendlyn Di Corci", "{U}{B}{B}{R}", 3, 5,
			WithSubTypes("Human", "Rogue"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				DiscardRandom(1),
				TapSourceCost(),
				WithTarget(TargetPlayer()),
				WithYourTurnOnly(),
			),
		)
	})

	// Halfdane {1}{W}{U}{B}
	// Legendary Creature — Shapeshifter
	// 3/3
	// At the beginning of your upkeep, change Halfdane's base power and toughness to the power and toughness of target creature other than Halfdane until the end of your next upkeep.
	Register("Halfdane", func() Card {
		return NewCreature("Halfdane", "{1}{W}{U}{B}", 3, 3,
			WithSubTypes("Shapeshifter"),
			WithSuperTypes(SuperLegendary),
			WithAbility(BeginningOfUpkeepTrigger(FuncEffect(
				"copy target creature's power and toughness until next upkeep",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					// Choose target creature other than Halfdane
					var candidates []*Permanent
					for _, p := range g.FilterBattlefield(IsCreature) {
						if p.ID() != sourceID {
							candidates = append(candidates, p)
						}
					}
					if len(candidates) == 0 {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					target := p.ChoosePermanent(candidates, "Halfdane: choose creature to copy P/T", g)
					if target == nil {
						return nil
					}
					// Snapshot the target's P/T now
					power := target.CurrentPower(g)
					toughness := target.CurrentToughness(g)
					// Apply until next upkeep (~2 turns)
					expiryTurn := g.CurrentTurn() + 2
					ce := FuncContinuousEffect(LayerPT, Indefinite, func(g *Game, _ uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							perm.BasePTOverride = &[2]int{power, toughness}
						}
						return nil
					}, func(g *Game, _ uuid.UUID) bool {
						return g.CurrentTurn() <= expiryTurn && g.FindPermanent(sourceID) != nil
					})
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					return nil
				}), true)),
		)
	})

	// Hazezon Tamar {4}{R}{G}{W}
	// Legendary Creature — Human Warrior
	// 2/4
	// When Hazezon enters, create X 1/1 Sand Warrior creature tokens that are red, green, and white at the beginning of your next upkeep, where X is the number of lands you control at that time.
	// When Hazezon leaves the battlefield, exile all Sand Warriors.
	Register("Hazezon Tamar", func() Card {
		return NewCreature("Hazezon Tamar", "{4}{R}{G}{W}", 2, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			// ETB: register delayed trigger for next upkeep
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"create Sand Warrior tokens at next upkeep",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:     EvtUpkeep,
						SourceID:      sourceID,
						Controller:    controller,
						MatchPlayerID: controller,
						Effects: []Effect{FuncEffect(
							"create Sand Warrior tokens",
							EffectProperties{Outcome: OutcomeBenefit},
							func(g *Game, _ uuid.UUID, controller uuid.UUID, _ []uuid.UUID) error {
								x := g.CountBattlefield(And(ControlledBy(controller), IsLand))
								colors := []Color{Red, Green, White}
								for i := 0; i < x; i++ {
									token := NewToken("Sand Warrior", 1, 1, []CardType{TypeCreature}, []string{"Sand", "Warrior"})
									token.SetOwner(controller)
									perm := g.PutOnBattlefield(token, controller)
									perm.IsToken = true
									perm.ColorOverride = &colors
								}
								return nil
							},
						)},
					})
					return nil
				},
			), false)),
			// When Hazezon leaves, exile all Sand Warriors
			WithAbility(NewTriggered(EvtLeavesBattlefield, false,
				ForEachPermanent(
					And(HasSubType("Sand"), HasSubType("Warrior")),
					ExileTargetStep(),
					"exile all Sand Warriors",
				),
			).SetConditionData(EventSourceIsSelf{})),
		)
	})

	// Hunding Gjornersen {3}{W}{U}{U}
	// Legendary Creature — Human Warrior
	// 5/4
	// Rampage 1 (Whenever this creature becomes blocked, it gets +1/+1 until end of turn for each creature blocking it beyond the first.)
	Register("Hunding Gjornersen", func() Card {
		return NewCreature("Hunding Gjornersen", "{3}{W}{U}{U}", 5, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithAbility(RampageTrigger(1)),
		)
	})

	// Jacques le Vert {1}{R}{G}{W}
	// Legendary Creature — Human Warrior
	// 3/2
	// Green creatures you control get +0/+2.
	Register("Jacques le Vert", func() Card {
		return NewCreature("Jacques le Vert", "{1}{R}{G}{W}", 3, 2,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithStaticAbility(
				BoostControlledCreatures(0, 2, HasColorFilter(Green)),
			),
		)
	})

	// Jasmine Boreal {3}{G}{W}
	// Legendary Creature — Human
	// 4/5
	Register("Jasmine Boreal", func() Card {
		return NewCreature("Jasmine Boreal", "{3}{G}{W}", 4, 5,
			WithSubTypes("Human"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Jedit Ojanen {4}{W}{W}{U}
	// Legendary Creature — Cat Warrior
	// 5/5
	Register("Jedit Ojanen", func() Card {
		return NewCreature("Jedit Ojanen", "{4}{W}{W}{U}", 5, 5,
			WithSubTypes("Cat", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Jerrard of the Closed Fist {3}{R}{G}{G}
	// Legendary Creature — Human Knight
	// 6/5
	Register("Jerrard of the Closed Fist", func() Card {
		return NewCreature("Jerrard of the Closed Fist", "{3}{R}{G}{G}", 6, 5,
			WithSubTypes("Human", "Knight"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Johan {3}{R}{G}{W}
	// Legendary Creature — Human Wizard
	// 5/4
	// At the beginning of combat on your turn, you may have Johan gain "Johan can't attack" until end of combat. If you do, attacking doesn't cause creatures you control to tap this combat if Johan is untapped.
	Register("Johan", func() Card {
		return NewCreature("Johan", "{3}{R}{G}{W}", 5, 4,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithAbility(NewTriggered(EvtBeginCombat, true, FuncEffect(
				"your creatures don't tap to attack",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					// Johan can't attack this combat
					ce := FuncContinuousEffect(LayerAbility, EndOfCombat, func(g *Game, _ uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							g.RevokeAttr(perm.ID(), AttrCanAttack)
						}
						return nil
					})
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					// Grant vigilance to all your creatures while Johan is untapped
					ce2 := FuncContinuousEffect(LayerAbility, EndOfCombat, func(g *Game, _ uuid.UUID) error {
						johan := g.FindPermanent(sourceID)
						if johan == nil || johan.Tapped {
							return nil
						}
						for _, p := range g.AllBattlefield() {
							if p.Controller == johan.Controller && p.HasAttr(AttrIsCreature) {
								g.GrantAttr(p.ID(), Vigilance)
							}
						}
						return nil
					})
					ce2.SetSourceID(sourceID)
					g.AddContinuousEffect(ce2)
					return nil
				},
			)).SetConditionData(EventPlayerIsController{})),
		)
	})

	// Kasimir the Lone Wolf {4}{W}{U}
	// Legendary Creature — Human Warrior
	// 5/3
	Register("Kasimir the Lone Wolf", func() Card {
		return NewCreature("Kasimir the Lone Wolf", "{4}{W}{U}", 5, 3,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Kei Takahashi {2}{G}{W}
	// Legendary Creature — Human Cleric
	// 2/2
	// {T}: Prevent the next 2 damage that would be dealt to target creature this turn.
	Register("Kei Takahashi", func() Card {
		return NewCreature("Kei Takahashi", "{2}{G}{W}", 2, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(2)),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Lady Caleria {3}{G}{G}{W}{W}
	// Legendary Creature — Elf Archer
	// 3/6
	// {T}: Lady Caleria deals 3 damage to target attacking or blocking creature.
	Register("Lady Caleria", func() Card {
		return NewCreature("Lady Caleria", "{3}{G}{G}{W}{W}", 3, 6,
			WithSubTypes("Elf", "Archer"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				DealDamage(Fixed(3)),
				TapSourceCost(),
				WithTarget(TargetCreature(Or(IsAttacking, IsBlocking))),
			),
		)
	})

	// Lady Evangela {W}{U}{B}
	// Legendary Creature — Human Cleric
	// 1/2
	// {W}{B}, {T}: Prevent all combat damage that would be dealt by target creature this turn.
	Register("Lady Evangela", func() Card {
		return NewCreature("Lady Evangela", "{W}{U}{B}", 1, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				FuncEffect(
					"prevent all combat damage that would be dealt by target creature this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						// Add a continuous effect that prevents damage from this creature until EOT
						eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, srcID uuid.UUID) error {
							g.AddDamagePreventionRule(WithCombatOnly(), WithFrom(NewPermanentFilter("prevented source", func(p *Permanent, _ *Game) bool {
								return p.ID() == targets[0]
							})))
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						return nil
					},
				),
				ManaCostOf("{W}{B}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Lady Orca {5}{B}{R}
	// Legendary Creature — Demon
	// 7/4
	Register("Lady Orca", func() Card {
		return NewCreature("Lady Orca", "{5}{B}{R}", 7, 4,
			WithSubTypes("Demon"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Livonya Silone {2}{R}{R}{G}{G}
	// Legendary Creature — Human Warrior
	// 4/4
	// First strike; legendary landwalk (This creature can't be blocked as long as defending player controls a legendary land.)
	Register("Livonya Silone", func() Card {
		return NewCreature("Livonya Silone", "{2}{R}{R}{G}{G}", 4, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(FirstStrike),
			WithKeyword(LegendaryLandwalk),
		)
	})

	// Lord Magnus {3}{G}{W}{W}
	// Legendary Creature — Human Druid
	// 4/3
	// First strike
	// Creatures with plainswalk can be blocked as though they didn't have plainswalk.
	// Creatures with forestwalk can be blocked as though they didn't have forestwalk.
	Register("Lord Magnus", func() Card {
		return NewCreature("Lord Magnus", "{3}{G}{W}{W}", 4, 3,
			WithSubTypes("Human", "Druid"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(FirstStrike),
			WithStaticAbility(NullifyLandwalkEffect(Plainswalk)),
			WithStaticAbility(NullifyLandwalkEffect(Forestwalk)),
		)
	})

	// Marhault Elsdragon {3}{R}{R}{G}
	// Legendary Creature — Elf Warrior
	// 4/6
	// Rampage 1 (Whenever this creature becomes blocked, it gets +1/+1 until end of turn for each creature blocking it beyond the first.)
	Register("Marhault Elsdragon", func() Card {
		return NewCreature("Marhault Elsdragon", "{3}{R}{R}{G}", 4, 6,
			WithSubTypes("Elf", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithAbility(RampageTrigger(1)),
		)
	})

	// Nebuchadnezzar {3}{U}{B}
	// Legendary Creature — Human Wizard
	// 3/3
	// {X}, {T}: Choose a card name. Target opponent reveals X cards at random from their hand. Then that player discards all cards with that name revealed this way. Activate only during your turn.
	// TODO: implement
	Register("Nebuchadnezzar", func() Card {
		return NewCreature("Nebuchadnezzar", "{3}{U}{B}", 3, 3,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Nicol Bolas {2}{U}{U}{B}{B}{R}{R}
	// Legendary Creature — Elder Dragon
	// 7/7
	// Flying
	// At the beginning of your upkeep, sacrifice Nicol Bolas unless you pay {U}{B}{R}.
	// Whenever Nicol Bolas deals damage to an opponent, that player discards their hand.
	Register("Nicol Bolas", func() Card {
		return NewCreature("Nicol Bolas", "{2}{U}{U}{B}{B}{R}{R}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithAbility(SacrificeAtUpkeepUnlessPay("{U}{B}{R}")),
			WithAbility(NewTriggered(EvtDamageDealt, false, FuncEffect(
				"that player discards their hand",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					// targets[0] = damaged player (passed by trigger system from EvtDamageDealt.TargetID)
					if len(targets) == 0 {
						return nil
					}
					p := g.GetPlayer(targets[0])
					if p == nil {
						return nil
					}
					// Copy hand slice to avoid concurrent modification during removal
					hand := make([]Card, len(p.Hand()))
					copy(hand, p.Hand())
					for _, c := range hand {
						p.RemoveFromHand(c.ID())
						p.AddToGraveyard(c)
					}
					return nil
				},
			)).
				SetConditionData(EventSourceIsSelfDamageToOpponent{})),
		)
	})

	// Palladia-Mors {2}{R}{R}{G}{G}{W}{W}
	// Legendary Creature — Elder Dragon
	// 7/7
	// Flying, trample
	// At the beginning of your upkeep, sacrifice Palladia-Mors unless you pay {R}{G}{W}.
	Register("Palladia-Mors", func() Card {
		return NewCreature("Palladia-Mors", "{2}{R}{R}{G}{G}{W}{W}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithKeyword(Trample),
			WithAbility(SacrificeAtUpkeepUnlessPay("{R}{G}{W}")),
		)
	})

	// Pavel Maliki {4}{B}{R}
	// Legendary Creature — Human
	// 5/3
	// {B}{R}: Pavel Maliki gets +1/+0 until end of turn.
	Register("Pavel Maliki", func() Card {
		return NewCreature("Pavel Maliki", "{4}{B}{R}", 5, 3,
			WithSubTypes("Human"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{B}{R}"),
			),
		)
	})

	// Princess Lucrezia {3}{U}{U}{B}
	// Legendary Creature — Human Wizard
	// 5/4
	// {T}: Add {U}.
	Register("Princess Lucrezia", func() Card {
		return NewCreature("Princess Lucrezia", "{3}{U}{U}{B}", 5, 4,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Blue),
		)
	})

	// Ragnar {G}{W}{U}
	// Legendary Creature — Human Cleric
	// 2/2
	// {G}{W}{U}, {T}: Regenerate target creature.
	Register("Ragnar", func() Card {
		return NewCreature("Ragnar", "{G}{W}{U}", 2, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				RegenerateTarget(),
				ManaCostOf("{G}{W}{U}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Ramirez DePietro {3}{U}{B}{B}
	// Legendary Creature — Human Pirate
	// 4/3
	// First strike
	Register("Ramirez DePietro", func() Card {
		return NewCreature("Ramirez DePietro", "{3}{U}{B}{B}", 4, 3,
			WithSubTypes("Human", "Pirate"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(FirstStrike),
		)
	})

	// Ramses Overdark {2}{U}{U}{B}{B}
	// Legendary Creature — Human Assassin
	// 4/3
	// {T}: Destroy target enchanted creature.
	Register("Ramses Overdark", func() Card {
		return NewCreature("Ramses Overdark", "{2}{U}{U}{B}{B}", 4, 3,
			WithSubTypes("Human", "Assassin"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				DestroyTarget(),
				TapSourceCost(),
				WithTarget(TargetCreature(NewPermanentFilter("enchanted", func(p *Permanent, g *Game) bool {
					for _, attachID := range p.Attachments {
						att := g.FindPermanent(attachID)
						if att != nil && att.HasType(TypeEnchantment) {
							return true
						}
					}
					return false
				}))),
			),
		)
	})

	// Rasputin Dreamweaver {4}{W}{U}
	// Legendary Creature — Human Wizard
	// 4/1
	// Rasputin enters with seven dream counters on it.
	// Remove a dream counter from Rasputin: Add {C}.
	// Remove a dream counter from Rasputin: Prevent the next 1 damage that would be dealt to Rasputin this turn.
	// At the beginning of your upkeep, if Rasputin started the turn untapped, put a dream counter on it.
	// Rasputin can't have more than seven dream counters on it.
	Register("Rasputin Dreamweaver", func() Card {
		return NewCreature("Rasputin Dreamweaver", "{4}{W}{U}", 4, 1,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
			// Enters with seven dream counters
			WithAbility(ETBEffect(AddCounters(Dream, Fixed(7)).Targeting(ToSource()))),
			// Remove a dream counter: Add {C}
			WithActivatedAbility(
				AddMana(Colorless, 1),
				RemoveCountersCost(Dream, 1),
			),
			// Remove a dream counter: Prevent the next 1 damage to Rasputin this turn
			// TODO: convert to pipeline — needs PreventDamageToSource(N) primitive
			WithActivatedAbility(
				FuncEffect(
					"prevent the next 1 damage that would be dealt to Rasputin this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						g.AddPreventionShield(sourceID, 1)
						return nil
					},
				),
				RemoveCountersCost(Dream, 1),
			),
			// XXX: "started the turn untapped" checks current tapped state at upkeep, not state at
			// start of turn — needs engine support for start-of-turn state snapshots
			// At the beginning of your upkeep, if Rasputin started the turn untapped, put a dream counter on it (max 7)
			WithAbility(NewTriggered(EvtUpkeep, false, FuncEffect(
				"put a dream counter on Rasputin",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					if perm.Counters[Dream] < 7 {
						perm.AddCounter(Dream, 1)
					}
					return nil
				},
			)).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{EventPlayerIsController{}, SourceIsUntapped{}}})),
		)
	})

	// Riven Turnbull {5}{U}{B}
	// Legendary Creature — Human Advisor
	// 5/7
	// {T}: Add {B}.
	Register("Riven Turnbull", func() Card {
		return NewCreature("Riven Turnbull", "{5}{U}{B}", 5, 7,
			WithSubTypes("Human", "Advisor"),
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Black),
		)
	})

	// Rohgahh of Kher Keep {2}{B}{B}{R}{R}
	// Legendary Creature — Kobold
	// 5/5
	// At the beginning of your upkeep, you may pay {R}{R}{R}. If you don't, tap Rohgahh and all creatures named Kobolds of Kher Keep, then an opponent gains control of them.
	// Creatures you control named Kobolds of Kher Keep get +2/+2.
	Register("Rohgahh of Kher Keep", func() Card {
		return NewCreature("Rohgahh of Kher Keep", "{2}{B}{B}{R}{R}", 5, 5,
			WithSubTypes("Kobold"),
			WithSuperTypes(SuperLegendary),
			// Static: Kobolds of Kher Keep you control get +2/+2
			WithStaticAbility(BoostControlledCreatures(2, 2, Named("Kobolds of Kher Keep"))),
			// Upkeep: pay {R}{R}{R} or tap + lose control
			WithAbility(NewTriggered(EvtUpkeep, false, FuncEffect(
				"pay {R}{R}{R} or tap and lose control",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					player := g.GetPlayer(controller)
					if player != nil && player.ChooseMayAbility("pay {R}{R}{R}") && g.TryPayCostFromLands(controller, "{R}{R}{R}") {
						return nil
					}
					opponent := g.GetOpponent(controller)
					if opponent == nil {
						return nil
					}
					oppID := opponent.PlayerID()
					perm := g.FindPermanent(sourceID)
					if perm != nil {
						g.TapPermanent(perm)
					}
					// Tap Kobolds of Kher Keep
					for _, p := range g.FilterBattlefield(NewPermanentFilter("Kobolds of Kher Keep", func(p *Permanent, _ *Game) bool {
						return p.Name() == "Kobolds of Kher Keep" && p.Controller == controller
					})) {
						g.TapPermanent(p)
					}
					// Use continuous effect at LayerControl to change controller
					// (direct assignment gets reset each Apply cycle)
					eff := FuncContinuousEffect(LayerControl, Indefinite, func(g *Game, srcID uuid.UUID) error {
						// Change controller of Rohgahh
						rohgahh := g.FindPermanent(sourceID)
						if rohgahh != nil {
							rohgahh.Controller = oppID
						}
						// Change controller of all Kobolds of Kher Keep
						for _, p := range g.AllBattlefield() {
							if p.Name() == "Kobolds of Kher Keep" && p.Card.Owner() == controller {
								p.Controller = oppID
							}
						}
						return nil
					})
					eff.SetSourceID(sourceID)
					g.AddContinuousEffect(eff)
					return nil
				},
			)).SetConditionData(EventPlayerIsController{})),
		)
	})

	// Rubinia Soulsinger {2}{G}{W}{U}
	// Legendary Creature — Faerie
	// 2/3
	// You may choose not to untap Rubinia Soulsinger during your untap step.
	// {T}: Gain control of target creature for as long as you control Rubinia Soulsinger and Rubinia Soulsinger remains tapped.
	Register("Rubinia Soulsinger", func() Card {
		return NewCreature("Rubinia Soulsinger", "{2}{G}{W}{U}", 2, 3,
			WithSubTypes("Faerie"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				FuncEffect("gain control of target creature while tapped",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						cancelled := false
						ce := FuncContinuousEffect(LayerControl, Indefinite, func(g *Game, srcID uuid.UUID) error {
							if cancelled {
								return nil
							}
							src := g.FindPermanent(srcID)
							if src == nil || !src.Tapped || src.Controller != controller {
								cancelled = true
								return nil
							}
							perm := g.FindPermanent(targetID)
							if perm != nil {
								perm.Controller = src.Controller
							}
							return nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					}),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Sir Shandlar of Eberyn {4}{G}{W}
	// Legendary Creature — Human Knight
	// 4/7
	Register("Sir Shandlar of Eberyn", func() Card {
		return NewCreature("Sir Shandlar of Eberyn", "{4}{G}{W}", 4, 7,
			WithSubTypes("Human", "Knight"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Sivitri Scarzam {5}{U}{B}
	// Legendary Creature — Human
	// 6/4
	Register("Sivitri Scarzam", func() Card {
		return NewCreature("Sivitri Scarzam", "{5}{U}{B}", 6, 4,
			WithSubTypes("Human"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Sol'kanar the Swamp King {2}{U}{B}{R}
	// Legendary Creature — Demon
	// 5/5
	// Swampwalk (This creature can't be blocked as long as defending player controls a Swamp.)
	// Whenever a player casts a black spell, you gain 1 life.
	Register("Sol'kanar the Swamp King", func() Card {
		return NewCreature("Sol'kanar the Swamp King", "{2}{U}{B}{R}", 5, 5,
			WithSubTypes("Demon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Swampwalk),
			WithAbility(WheneverSpellCastTrigger(GainLife(1), false, HasColorCardFilter(Black))),
		)
	})

	// Stangg {4}{R}{G}
	// Legendary Creature — Human Warrior
	// 3/4
	// When Stangg enters, create Stangg Twin, a legendary 3/4 red and green Human Warrior creature token. Exile that token when Stangg leaves the battlefield. Sacrifice Stangg when that token leaves the battlefield.
	Register("Stangg", func() Card {
		var twinID uuid.UUID
		return NewCreature("Stangg", "{4}{R}{G}", 3, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			// ETB: create Stangg Twin token
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"create Stangg Twin token",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					token := NewToken("Stangg Twin", 3, 4, []CardType{TypeCreature}, []string{"Human", "Warrior"})
					token.SetOwner(controller)
					WithSuperTypes(SuperLegendary)(token)
					perm := g.PutOnBattlefield(token, controller)
					perm.IsToken = true
					colors := []Color{Red, Green}
					perm.ColorOverride = &colors
					twinID = perm.ID()
					return nil
				},
			), false)),
			// When Stangg leaves, exile the token
			WithAbility(NewTriggered(EvtLeavesBattlefield, false, FuncEffect(
				"exile Stangg Twin",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					twin := g.FindPermanent(twinID)
					if twin != nil {
						g.ExilePermanent(twin)
					}
					return nil
				},
			)).SetConditionData(EventSourceIsSelf{})),
			// When the token leaves, sacrifice Stangg
			WithAbility(NewTriggered(EvtLeavesBattlefield, false, FuncEffect(
				"sacrifice Stangg",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm != nil {
						g.Sacrifice(perm)
					}
					return nil
				},
			)).
				// TODO: convert to data condition
				SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == twinID && evt.SourceID != sourceID
				})),
		)
	})

	// Sunastian Falconer {3}{R}{G}
	// Legendary Creature — Human Shaman
	// 4/4
	// {T}: Add {C}{C}.
	Register("Sunastian Falconer", func() Card {
		return NewCreature("Sunastian Falconer", "{3}{R}{G}", 4, 4,
			WithSubTypes("Human", "Shaman"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				AddMana(Colorless, 2),
				TapSourceCost(),
			),
		)
	})

	// Tetsuo Umezawa {U}{B}{R}
	// Legendary Creature — Human Archer
	// 3/3
	// Tetsuo Umezawa can't be the target of Aura spells.
	// {U}{B}{B}{R}, {T}: Destroy target tapped or blocking creature.
	// XXX: "can't be the target of Aura spells" not yet implemented
	Register("Tetsuo Umezawa", func() Card {
		return NewCreature("Tetsuo Umezawa", "{U}{B}{R}", 3, 3,
			WithSubTypes("Human", "Archer"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				DestroyTarget(),
				ManaCostOf("{U}{B}{B}{R}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature(Or(IsTapped, IsBlocking))),
			),
		)
	})

	// The Lady of the Mountain {4}{R}{G}
	// Legendary Creature — Giant
	// 5/5
	Register("The Lady of the Mountain", func() Card {
		return NewCreature("The Lady of the Mountain", "{4}{R}{G}", 5, 5,
			WithSubTypes("Giant"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Tobias Andrion {3}{W}{U}
	// Legendary Creature — Human Advisor
	// 4/4
	Register("Tobias Andrion", func() Card {
		return NewCreature("Tobias Andrion", "{3}{W}{U}", 4, 4,
			WithSubTypes("Human", "Advisor"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Tor Wauki {2}{B}{B}{R}
	// Legendary Creature — Human Archer
	// 3/3
	// {T}: Tor Wauki deals 2 damage to target attacking or blocking creature.
	Register("Tor Wauki", func() Card {
		return NewCreature("Tor Wauki", "{2}{B}{B}{R}", 3, 3,
			WithSubTypes("Human", "Archer"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				DealDamage(Fixed(2)),
				TapSourceCost(),
				WithTarget(TargetCreature(Or(IsAttacking, IsBlocking))),
			),
		)
	})

	// Torsten Von Ursus {3}{G}{G}{W}
	// Legendary Creature — Human Soldier
	// 5/5
	Register("Torsten Von Ursus", func() Card {
		return NewCreature("Torsten Von Ursus", "{3}{G}{G}{W}", 5, 5,
			WithSubTypes("Human", "Soldier"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Tuknir Deathlock {R}{R}{G}{G}
	// Legendary Creature — Human Wizard
	// 2/2
	// Flying
	// {R}{G}, {T}: Target creature gets +2/+2 until end of turn.
	Register("Tuknir Deathlock", func() Card {
		return NewCreature("Tuknir Deathlock", "{R}{R}{G}{G}", 2, 2,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(2), Fixed(2)),
				ManaCostOf("{R}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Ur-Drago {3}{U}{U}{B}{B}
	// Legendary Creature — Elemental
	// 4/4
	// First strike
	// Creatures with swampwalk can be blocked as though they didn't have swampwalk.
	Register("Ur-Drago", func() Card {
		return NewCreature("Ur-Drago", "{3}{U}{U}{B}{B}", 4, 4,
			WithSubTypes("Elemental"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(FirstStrike),
			WithStaticAbility(NullifyLandwalkEffect(Swampwalk)),
		)
	})

	// Vaevictis Asmadi {2}{B}{B}{R}{R}{G}{G}
	// Legendary Creature — Elder Dragon
	// 7/7
	// Flying
	// At the beginning of your upkeep, sacrifice Vaevictis Asmadi unless you pay {B}{R}{G}.
	// {B}: Vaevictis Asmadi gets +1/+0 until end of turn.
	// {R}: Vaevictis Asmadi gets +1/+0 until end of turn.
	// {G}: Vaevictis Asmadi gets +1/+0 until end of turn.
	Register("Vaevictis Asmadi", func() Card {
		return NewCreature("Vaevictis Asmadi", "{2}{B}{B}{R}{R}{G}{G}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithAbility(SacrificeAtUpkeepUnlessPay("{B}{R}{G}")),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{B}"),
			),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{G}"),
			),
		)
	})

	// Xira Arien {B}{R}{G}
	// Legendary Creature — Insect Wizard
	// 1/2
	// Flying
	// {B}{R}{G}, {T}: Target player draws a card.
	Register("Xira Arien", func() Card {
		return NewCreature("Xira Arien", "{B}{R}{G}", 1, 2,
			WithSubTypes("Insect", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				ManaCostOf("{B}{R}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetPlayer()),
			),
		)
	})

	// ===== COLORLESS CREATURES =====

	// Bronze Horse {7}
	// Artifact Creature — Horse
	// 4/4
	// Trample
	// As long as you control another creature, prevent all damage that would be dealt to this creature by spells that target it.
	Register("Bronze Horse", func() Card {
		return NewCreature("Bronze Horse", "{7}", 4, 4,
			WithSubTypes("Horse"),
			WithCardType(TypeArtifact),
			WithKeyword(Trample),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				// Check if controller has another creature
				hasOther := false
				for _, p := range g.AllBattlefield() {
					if p.Controller == src.Controller && p.HasType(TypeCreature) && p.ID() != sourceID {
						hasOther = true
						break
					}
				}
				if !hasOther {
					return nil
				}
				g.AddDamagePreventionRule(
					WithTo(NewPermanentFilter("Bronze Horse targeted by spell", func(p *Permanent, g *Game) bool {
						if p.ID() != sourceID {
							return false
						}
						// Only prevent if damage is from a resolving spell that targets this permanent
						if g.GetResolvingCard() == nil {
							return false
						}
						for _, t := range g.GetResolvingTargets() {
							if t == sourceID {
								return true
							}
						}
						return false
					})),
				)
				return nil
			})),
		)
	})

	// Marble Priest {5}
	// Artifact Creature — Cleric
	// 3/3
	// All Walls able to block this creature do so.
	// XXX: "All Walls able to block this creature do so" forced-block not implemented
	// Prevent all combat damage that would be dealt to this creature by Walls.
	Register("Marble Priest", func() Card {
		return NewCreature("Marble Priest", "{5}", 3, 3,
			WithSubTypes("Cleric"),
			WithCardType(TypeArtifact),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.AddDamagePreventionRule(
					WithCombatOnly(),
					WithFrom(NewPermanentFilter("Wall", func(p *Permanent, _ *Game) bool {
						return p.HasSubType("Wall")
					})),
					WithTo(NewPermanentFilter("self", func(p *Permanent, _ *Game) bool {
						return p.ID() == sourceID
					})),
				)
				return nil
			})),
		)
	})

	// Sentinel {4}
	// Artifact Creature — Shapeshifter
	// 1/1
	// {0}: Change this creature's base toughness to 1 plus the power of target creature blocking or blocked by this creature. (This effect lasts indefinitely.)
	Register("Sentinel", func() Card {
		return NewCreature("Sentinel", "{4}", 1, 1,
			WithSubTypes("Shapeshifter"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				FuncEffect("change base toughness to 1 plus target creature's power",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := g.FindPermanent(targets[0])
						if target == nil {
							return nil
						}
						// Verify target is blocking or blocked by this creature
						inCombat := false
						for _, group := range g.CombatGroups() {
							if group.AttackerID == sourceID {
								for _, bid := range group.BlockerIDs {
									if bid == targets[0] {
										inCombat = true
										break
									}
								}
							} else if group.AttackerID == targets[0] {
								for _, bid := range group.BlockerIDs {
									if bid == sourceID {
										inCombat = true
										break
									}
								}
							}
							if inCombat {
								break
							}
						}
						if !inCombat {
							return nil
						}
						newToughness := 1 + target.CurrentPower(g)
						sentinelID := sourceID
						ce := FuncContinuousEffect(LayerPT, Indefinite, func(g *Game, _ uuid.UUID) error {
							p := g.FindPermanent(sentinelID)
							if p != nil {
								pw := p.Card.Power()
								if p.BasePTOverride != nil {
									pw = p.BasePTOverride[0]
								}
								p.BasePTOverride = &[2]int{pw, newToughness}
							}
							return nil
						}, func(g *Game, _ uuid.UUID) bool {
							return g.FindPermanent(sentinelID) != nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					},
				),
				GenericCost(0),
				WithTarget(TargetCreatureBlockingOrBlockedBySource()),
			),
		)
	})

}
