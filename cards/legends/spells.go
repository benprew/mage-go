package legends

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

func init() {
	registerSpells()
}

func registerSpells() {

	// Acid Rain {3}{U}
	// Sorcery
	// Destroy all Forests.
	Register("Acid Rain", func() Card {
		return NewSorcery("Acid Rain", "{3}{U}",
			NewSpellAbility(DestroyAllMatching(And(IsLand, HasSubType("Forest")), "destroy all Forests")),
		)
	})

	// Active Volcano {R}
	// Instant
	// Choose one —
	// • Destroy target blue permanent.
	// • Return target Island to its owner's hand.
	Register("Active Volcano", func() Card {
		c := NewInstant("Active Volcano", "{R}",
			NewTargetedSpell(TargetPermanent(), Pipeline(
				"destroy target blue permanent or return target Island to its owner's hand",
				EffectProperties{Outcome: OutcomeDetriment},
				ModalEffect("choose one",
					DestroyTarget(),
					ReturnToHandTarget(),
				),
			)),
		)
		c.SetModes([]string{
			"Destroy target blue permanent",
			"Return target Island to its owner's hand",
		})
		return c
	})

	// Alabaster Potion {X}{W}{W}
	// Instant
	// Choose one —
	// • Target player gains X life.
	// • Prevent the next X damage that would be dealt to any target this turn.
	// TODO: convert to pipeline — needs ModalEffect with XValue-based GainLifeTarget and PreventDamageToTarget
	Register("Alabaster Potion", func() Card {
		c := NewInstant("Alabaster Potion", "{X}{W}{W}",
			NewTargetedSpell(TargetAnyTarget(), FuncEffect(
				"target player gains X life or prevent the next X damage to any target",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					x := g.XValue()
					if g.ModeValue() == 0 {
						// Mode 1: Target player gains X life
						for _, pl := range g.AllPlayers() {
							if pl.PlayerID() == targets[0] {
								g.PlayerGainLife(pl, x)
								return nil
							}
						}
					} else {
						// Mode 2: Prevent the next X damage to any target
						g.AddPreventionShield(targets[0], x)
					}
					return nil
				},
			)),
		)
		c.SetModes([]string{
			"Target player gains X life",
			"Prevent the next X damage that would be dealt to any target this turn",
		})
		return c
	})

	// All Hallow's Eve {2}{B}{B}
	// Sorcery
	// Exile All Hallow's Eve with two scream counters on it.
	// At the beginning of your upkeep, if this card is exiled with a scream counter on it, remove a scream counter from it. If there are no more scream counters on it, put it into your graveyard and each player returns all creature cards from their graveyard to the battlefield.
	// XXX: requires exile-with-counters and delayed upkeep trigger from exile zone — engine lacks support for tracking counters on exiled cards
	// TODO: implement when exile-counter tracking is available
	Register("All Hallow's Eve", func() Card {
		return NewSorcery("All Hallow's Eve", "{2}{B}{B}",
			NewSpellAbility(),
		)
	})

	// Avoid Fate {G}
	// Instant
	// Counter target instant or Aura spell that targets a permanent you control.
	// XXX: requires targeting spells that target your permanents — engine lacks support for "spell that targets a permanent you control" filter
	// TODO: implement when stack-target-filtering supports conditional target checks
	Register("Avoid Fate", func() Card {
		return NewInstant("Avoid Fate", "{G}",
			NewSpellAbility(),
		)
	})

	// Backdraft {1}{R}
	// Instant
	// Choose a player who cast one or more sorcery spells this turn. Backdraft deals damage to that player equal to half the damage dealt by one of those sorcery spells this turn, rounded down.
	// XXX: requires tracking sorcery spells cast this turn and their damage dealt — engine lacks sorcery-damage history tracking
	// TODO: implement when spell-damage tracking is available
	Register("Backdraft", func() Card {
		return NewInstant("Backdraft", "{1}{R}",
			NewSpellAbility(),
		)
	})

	// Blood Lust {1}{R}
	// Instant
	// If target creature has toughness 5 or greater, it gets +4/-4 until end of turn. Otherwise, it gets +4/-X until end of turn, where X is its toughness minus 1.
	// TODO: convert to pipeline — needs conditional TemporaryBoost based on current toughness
	Register("Blood Lust", func() Card {
		return NewInstant("Blood Lust", "{1}{R}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"target creature gets +4/-4 or +4/-(toughness-1)",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					toughness := perm.CurrentToughness(g)
					var tMod int
					if toughness >= 5 {
						tMod = -4
					} else {
						tMod = -(toughness - 1)
					}
					ce := TemporaryBoost(perm.ID(), 4, tMod)
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					return nil
				},
			)),
		)
	})

	// Boomerang {U}{U}
	// Instant
	// Return target permanent to its owner's hand.
	Register("Boomerang", func() Card {
		return NewInstant("Boomerang", "{U}{U}",
			NewTargetedSpell(TargetPermanent(), ReturnToHandTarget()),
		)
	})

	// Chain Lightning {R}
	// Sorcery
	// Chain Lightning deals 3 damage to any target. Then that player or that permanent's controller may pay {R}{R}. If the player does, they may copy this spell and may choose a new target for that copy.
	// XXX: chain copy mechanic not implemented — just deals 3 damage
	Register("Chain Lightning", func() Card {
		return NewSorcery("Chain Lightning", "{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(3))),
		)
	})

	// Cleanse {2}{W}{W}
	// Sorcery
	// Destroy all black creatures.
	Register("Cleanse", func() Card {
		return NewSorcery("Cleanse", "{2}{W}{W}",
			NewSpellAbility(DestroyAllMatching(And(IsCreature, HasColorFilter(Black)), "destroy all black creatures")),
		)
	})

	// Darkness {B}
	// Instant
	// Prevent all combat damage that would be dealt this turn.
	Register("Darkness", func() Card {
		return NewInstant("Darkness", "{B}",
			NewSpellAbility(PreventAllCombatDamage()),
		)
	})

	// Disharmony {2}{R}
	// Instant
	// Cast this spell only during combat before blockers are declared.
	// Untap target attacking creature and remove it from combat. Gain control of that creature until end of turn.
	// XXX: timing restriction (only before blockers) not enforced
	Register("Disharmony", func() Card {
		return NewInstant("Disharmony", "{2}{R}",
			NewTargetedSpell(TargetCreature(IsAttacking), FuncEffect(
				"untap target attacking creature, remove from combat, gain control until end of turn",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					perm.Tapped = false
					g.RemoveFromCombat(perm.ID())
					targetID := targets[0]
					// Gain control until end of turn — controller resets to owner when EOT effect expires
					ce := FuncContinuousEffect(LayerControl, EndOfTurn, func(g *Game, _ uuid.UUID) error {
						p := g.FindPermanent(targetID)
						if p != nil {
							p.Controller = controller
						}
						return nil
					})
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					return nil
				},
			)),
		)
	})

	// Divine Offering {1}{W}
	// Instant
	// Destroy target artifact. You gain life equal to its mana value.
	Register("Divine Offering", func() Card {
		return NewInstant("Divine Offering", "{1}{W}",
			NewTargetedSpell(TargetArtifact(),
				Pipeline("destroy target artifact; you gain life equal to its mana value",
					EffectProperties{Outcome: OutcomeDetriment},
					SnapshotPermanent(SelectTarget, "victim"),
					DestroyTargetStep(),
					GainLifeControllerFromVar("victim.cmc"),
				)),
		)
	})

	// Dwarven Song {R}
	// Instant
	// One or more target creatures become red until end of turn.
	Register("Dwarven Song", func() Card {
		return NewInstant("Dwarven Song", "{R}",
			NewTargetedSpell(TargetCreature(), ChangeColorEffect(Red)),
		)
	})

	// Enchantment Alteration {U}
	// Instant
	// Attach target Aura attached to a creature or land to another permanent of that type.
	// XXX: requires re-attaching an existing Aura to a new host — engine lacks support for Aura migration
	// TODO: implement when Aura re-attachment is supported
	Register("Enchantment Alteration", func() Card {
		return NewInstant("Enchantment Alteration", "{U}",
			NewSpellAbility(),
		)
	})

	// Energy Tap {U}
	// Sorcery
	// Tap target untapped creature you control. If you do, add an amount of {C} equal to that creature's mana value.
	// TODO: convert to pipeline — needs TapGathered step and AddManaFromVar step
	Register("Energy Tap", func() Card {
		return NewSorcery("Energy Tap", "{U}",
			NewTargetedSpell(TargetControlledCreature(), FuncEffect(
				"tap target untapped creature you control; add {C} equal to its mana value",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					if perm.Tapped {
						return nil // must be untapped
					}
					g.TapPermanent(perm)
					cmc := perm.Card.ManaCost().CMC()
					if cmc > 0 {
						p := g.GetPlayer(controller)
						if p != nil {
							p.ManaPool().Add(Colorless, cmc)
						}
					}
					return nil
				},
			)),
		)
	})

	// Eureka {2}{G}{G}
	// Sorcery
	// Starting with you, each player may put a permanent card from their hand onto the battlefield. Repeat this process until no one puts a card onto the battlefield.
	// XXX: requires iterative player choice of putting permanent cards from hand to battlefield — engine lacks support for repeated interactive permanent-drops
	// TODO: implement when iterative player choice from hand is supported
	Register("Eureka", func() Card {
		return NewSorcery("Eureka", "{2}{G}{G}",
			NewSpellAbility(),
		)
	})

	// Falling Star {2}{R}
	// Sorcery
	// Flip Falling Star onto the playing area from a height of at least one foot. Falling Star deals 3 damage to each creature it lands on. Tap all creatures dealt damage by Falling Star. If Falling Star doesn't turn completely over at least once during the flip, it has no effect.
	// UNIMPLEMENTABLE: Physical dexterity card — requires flipping a physical card onto the play area.
	Register("Falling Star", func() Card {
		return NewSorcery("Falling Star", "{2}{R}",
			NewSpellAbility(),
		)
	})

	// Feint {R}
	// Instant
	// Tap all creatures blocking target attacking creature. Prevent all combat damage that would be dealt this turn by that creature and each creature blocking it.
	Register("Feint", func() Card {
		return NewInstant("Feint", "{R}",
			NewTargetedSpell(TargetCreature(IsAttacking), FuncEffect(
				"tap all blockers of target attacker; prevent all combat damage from it and its blockers this turn",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					attackerID := targets[0]
					// Collect the attacker and all its blockers
					preventIDs := map[uuid.UUID]bool{attackerID: true}
					for _, grp := range g.CombatGroups() {
						if grp.AttackerID == attackerID {
							for _, bid := range grp.BlockerIDs {
								preventIDs[bid] = true
								// Tap each blocker
								blocker := g.FindPermanent(bid)
								if blocker != nil {
									g.TapPermanent(blocker)
								}
							}
							break
						}
					}
					// Prevent all combat damage from the attacker and each blocker
					for permID := range preventIDs {
						pid := permID
						eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, _ uuid.UUID) error {
							g.AddDamagePreventionRule(WithFrom(NewPermanentFilter("feinted creature", func(p *Permanent, _ *Game) bool {
								return p.ID() == pid
							})))
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
					}
					return nil
				},
			)),
		)
	})

	// Flash Counter {1}{U}
	// Instant
	// Counter target instant spell.
	Register("Flash Counter", func() Card {
		return NewInstant("Flash Counter", "{1}{U}",
			NewTargetedSpell(
				TargetSpellOnStack(NewCardFilter("instant", func(c Card) bool {
					return c.HasType(TypeInstant)
				})),
				CounterSpell(),
			),
		)
	})

	// Flash Flood {U}
	// Instant
	// Choose one —
	// • Destroy target red permanent.
	// • Return target Mountain to its owner's hand.
	Register("Flash Flood", func() Card {
		c := NewInstant("Flash Flood", "{U}",
			NewTargetedSpell(TargetPermanent(), Pipeline(
				"destroy target red permanent or return target Mountain to its owner's hand",
				EffectProperties{Outcome: OutcomeDetriment},
				ModalEffect("choose one",
					DestroyTarget(),
					ReturnToHandTarget(),
				),
			)),
		)
		c.SetModes([]string{
			"Destroy target red permanent",
			"Return target Mountain to its owner's hand",
		})
		return c
	})

	// Force Spike {U}
	// Instant
	// Counter target spell unless its controller pays {1}.
	Register("Force Spike", func() Card {
		return NewInstant("Force Spike", "{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterUnlessPay("{1}")),
		)
	})

	// Glyph of Delusion {U}
	// Instant
	// Put X glyph counters on target creature that target Wall blocked this turn, where X is the power of that blocked creature. The creature gains "This creature doesn't untap during your untap step if it has a glyph counter on it" and "At the beginning of your upkeep, remove a glyph counter from this creature."
	Register("Glyph of Delusion", func() Card {
		return NewInstant("Glyph of Delusion", "{U}",
			NewTargetedSpell(TargetCreature(NewPermanentFilter("Wall", func(p *Permanent, _ *Game) bool {
				return p.HasSubType("Wall")
			})), FuncEffect(
				"put glyph counters on creature blocked by target Wall; it doesn't untap while it has glyph counters",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					wallID := targets[0]
					// Find the creature(s) blocked by this Wall in combat groups
					for _, grp := range g.CombatGroups() {
						for _, bid := range grp.BlockerIDs {
							if bid == wallID {
								creature := g.FindPermanent(grp.AttackerID)
								if creature == nil {
									continue
								}
								power := creature.CurrentPower(g)
								if power > 0 {
									creature.AddCounter(Glyph, power)
								}
								creatureID := creature.ID()
								// Grant "doesn't untap while it has glyph counters" (indefinite)
								eff := TargetEffect(LayerAbility, Indefinite, creatureID, func(g *Game, target *Permanent) error {
									if target.Counters[Glyph] > 0 {
										g.GrantAttr(target.ID(), AttrDoesNotUntap)
									}
									return nil
								})
								eff.SetSourceID(sourceID)
								g.AddContinuousEffect(eff)
								// Grant "at beginning of your upkeep, remove a glyph counter"
								trigger := BeginningOfUpkeepTrigger(
									FuncEffect("remove a glyph counter",
										EffectProperties{},
										func(g *Game, srcID, ctrl uuid.UUID, _ []uuid.UUID) error {
											perm := g.FindPermanent(srcID)
											if perm != nil && perm.Counters[Glyph] > 0 {
												perm.RemoveCounter(Glyph, 1)
											}
											return nil
										}), false,
								)
								trigger.SetSource(creature.ID())
								trigger.SetController(creature.Controller)
								creature.RuntimeAbilities = append(creature.RuntimeAbilities, trigger)
							}
						}
					}
					return nil
				},
			)),
		)
	})

	// Glyph of Destruction {R}
	// Instant
	// Target blocking Wall you control gets +10/+0 until end of combat. Prevent all damage that would be dealt to it this turn. Destroy it at the beginning of the next end step.
	Register("Glyph of Destruction", func() Card {
		return NewInstant("Glyph of Destruction", "{R}",
			NewTargetedSpell(TargetCreature(And(IsBlocking, NewPermanentFilter("Wall you control", func(p *Permanent, g *Game) bool {
				return p.HasSubType("Wall")
			}))), FuncEffect(
				"target blocking Wall gets +10/+0 until end of combat; prevent all damage to it this turn; destroy it at end step",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					wallID := targets[0]
					// +10/+0 until end of combat
					eff := TargetEffect(LayerPT, EndOfCombat, wallID, func(g *Game, target *Permanent) error {
						target.BoostPT(10, 0)
						return nil
					})
					eff.SetSourceID(sourceID)
					g.AddContinuousEffect(eff)
					// Prevent all damage to it this turn
					prevEff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, srcID uuid.UUID) error {
						g.AddDamagePreventionRule(WithTo(NewPermanentFilter("target Wall", func(p *Permanent, _ *Game) bool {
							return p.ID() == wallID
						})))
						return nil
					})
					prevEff.SetSourceID(sourceID)
					g.AddContinuousEffect(prevEff)
					// Destroy at beginning of next end step
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:  EvtEndStep,
						TargetID:   wallID,
						Effects:    []Effect{DestroyTarget()},
						SourceID:   sourceID,
						Controller: controller,
					})
					return nil
				},
			)),
		)
	})

	// Glyph of Doom {B}
	// Instant
	// Choose target Wall creature. At this turn's next end of combat, destroy all creatures that were blocked by that creature this turn.
	Register("Glyph of Doom", func() Card {
		return NewInstant("Glyph of Doom", "{B}",
			NewTargetedSpell(TargetCreature(NewPermanentFilter("Wall", func(p *Permanent, _ *Game) bool {
				return p.HasSubType("Wall")
			})), FuncEffect(
				"at end of combat, destroy all creatures blocked by target Wall",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					wallID := targets[0]
					for _, group := range g.CombatGroups() {
						for _, bid := range group.BlockerIDs {
							if bid == wallID {
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
					return nil
				},
			)),
		)
	})

	// Glyph of Life {W}
	// Instant
	// Choose target Wall creature. Whenever that creature is dealt damage by an attacking creature this turn, you gain that much life.
	Register("Glyph of Life", func() Card {
		return NewInstant("Glyph of Life", "{W}",
			NewTargetedSpell(TargetCreature(NewPermanentFilter("Wall", func(p *Permanent, _ *Game) bool {
				return p.HasSubType("Wall")
			})), FuncEffect(
				"whenever target Wall is dealt damage by an attacking creature this turn, gain that much life",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					wallID := targets[0]
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:     EvtDamageDealt,
						SourceID:      sourceID,
						Controller:    controller,
						TargetID:      controller, // pass controller as target for life gain
						MatchTargetID: wallID,     // only fire when the wall receives damage
						Persistent:    true,       // fires each time this turn
						Effects:       []Effect{GainLifeAmount(EventAmountValue())},
					})
					return nil
				},
			)),
		)
	})

	// Glyph of Reincarnation {G}
	// Instant
	// Cast this spell only after combat.
	// Destroy all creatures that were blocked by target Wall this turn. They can't be regenerated. For each creature that died this way, put a creature card from the graveyard of the player who controlled that creature the last time it became blocked by that Wall onto the battlefield under its owner's control.
	// XXX: timing restriction (only after combat) not enforced
	Register("Glyph of Reincarnation", func() Card {
		return NewInstant("Glyph of Reincarnation", "{G}",
			NewTargetedSpell(TargetCreature(NewPermanentFilter("Wall", func(p *Permanent, _ *Game) bool {
				return p.HasSubType("Wall")
			})), FuncEffect(
				"destroy creatures blocked by target Wall; reanimate from their owners' graveyards",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					wallID := targets[0]
					// Find all creatures this Wall blocked this turn
					blockedAttackers := g.GetBlockedThisTurn(wallID)
					for _, attackerID := range blockedAttackers {
						perm := g.FindPermanent(attackerID)
						if perm == nil || !perm.HasType(TypeCreature) {
							continue
						}
						ownerID := perm.Card.Owner()
						// "They can't be regenerated"
						perm.GrantBaseAttr(CantRegenerate)
						// Destroy the creature
						g.DestroyPermanent(perm)
						// Reanimate a creature from the owner's graveyard
						owner := g.GetPlayer(ownerID)
						if owner == nil {
							continue
						}
						for _, card := range owner.Graveyard() {
							if card.HasType(TypeCreature) {
								c, ok := owner.RemoveFromGraveyard(card.ID())
								if ok {
									g.PutOnBattlefield(c, ownerID)
								}
								break
							}
						}
					}
					return nil
				},
			)),
		)
	})

	// Great Defender {W}
	// Instant
	// Target creature gets +0/+X until end of turn, where X is its mana value.
	// TODO: convert to pipeline — needs Boost to support VarInt context binding
	Register("Great Defender", func() Card {
		return NewInstant("Great Defender", "{W}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"target creature gets +0/+X until end of turn, where X is its mana value",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					cmc := perm.Card.ManaCost().CMC()
					if cmc > 0 {
						ce := TemporaryBoost(perm.ID(), 0, cmc)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
					}
					return nil
				},
			)),
		)
	})

	// Heaven's Gate {W}
	// Instant
	// One or more target creatures become white until end of turn.
	Register("Heaven's Gate", func() Card {
		return NewInstant("Heaven's Gate", "{W}",
			NewTargetedSpell(TargetCreature(), ChangeColorEffect(White)),
		)
	})

	// Hell Swarm {B}
	// Instant
	// All creatures get -1/-0 until end of turn.
	Register("Hell Swarm", func() Card {
		return NewInstant("Hell Swarm", "{B}",
			NewSpellAbility(Boost(Fixed(-1), Fixed(0)).Targeting(ToAllMatching(IsCreature))),
		)
	})

	// Hellfire {2}{B}{B}{B}
	// Sorcery
	// Destroy all nonblack creatures. Hellfire deals X plus 3 damage to you, where X is the number of creatures that died this way.
	Register("Hellfire", func() Card {
		return NewSorcery("Hellfire", "{2}{B}{B}{B}",
			NewSpellAbility(FuncEffect(
				"destroy all nonblack creatures; Hellfire deals X plus 3 damage to you where X is the number destroyed",
				EffectProperties{Outcome: OutcomeDetriment, Mass: true},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					toDestroy := g.FilterBattlefield(And(IsCreature, Not(HasColorFilter(Black)), Not(HasKeywordFilter(Indestructible))))
					destroyed := 0
					for _, perm := range toDestroy {
						permID := perm.ID()
						g.DestroyPermanent(perm)
						// Only count creatures that actually died (not saved by regeneration)
						if g.FindPermanent(permID) == nil {
							destroyed++
						}
					}
					damage := destroyed + 3
					p := g.GetPlayer(controller)
					if p != nil {
						g.DealDamageToPlayer(p, damage, sourceID)
					}
					return nil
				},
			)),
		)
	})

	// Holy Day {W}
	// Instant
	// Prevent all combat damage that would be dealt this turn.
	Register("Holy Day", func() Card {
		return NewInstant("Holy Day", "{W}",
			NewSpellAbility(PreventAllCombatDamage()),
		)
	})

	// Indestructible Aura {W}
	// Instant
	// Prevent all damage that would be dealt to target creature this turn.
	Register("Indestructible Aura", func() Card {
		return NewInstant("Indestructible Aura", "{W}",
			NewTargetedSpell(TargetCreature(), PreventDamageToTarget(Fixed(999999))),
		)
	})

	// Jovial Evil {2}{B}
	// Sorcery
	// Jovial Evil deals X damage to target opponent, where X is twice the number of white creatures that player controls.
	Register("Jovial Evil", func() Card {
		return NewSorcery("Jovial Evil", "{2}{B}",
			NewTargetedSpell(TargetOpponent(),
				DealDamageToPlayers(
					Mul(Fixed(2), CountBattlefield(SelectTargetPlayer(), And(IsCreature, HasColorFilter(White)))),
					SelectTargetPlayer(),
				),
			),
		)
	})

	// Juxtapose {3}{U}
	// Sorcery
	// You and target player exchange control of the creature you each control with the greatest mana value. Then exchange control of artifacts the same way. If two or more permanents a player controls are tied for greatest, their controller chooses one of them.
	// XXX: requires mutual control exchange of highest-CMC permanents with tie-breaking choice — complex control exchange not supported
	// TODO: implement when mutual control exchange with player choice is supported
	Register("Juxtapose", func() Card {
		return NewSorcery("Juxtapose", "{3}{U}",
			NewSpellAbility(),
		)
	})

	// Mana Drain {U}{U}
	// Instant
	// Counter target spell. At the beginning of your next main phase, add an amount of {C} equal to that spell's mana value.
	Register("Mana Drain", func() Card {
		return NewInstant("Mana Drain", "{U}{U}",
			NewTargetedSpell(TargetSpellOnStack(), FuncEffect(
				"counter target spell; add {C} equal to its mana value at next main phase",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					// Get the spell's CMC before countering it
					obj := g.FindStackObject(targets[0])
					cmc := 0
					if obj != nil && obj.Card != nil {
						cmc = obj.Card.ManaCost().CMC()
					}
					g.CounterSpellOnStack(targets[0])
					// XXX: Oracle says "at the beginning of your next main phase" but no EvtMainPhase
					// event exists in the engine. Using EvtUpkeep as fallback — mana pools persist
					// between steps so the mana will be available at main phase, but the timing is
					// technically wrong (fires at upkeep instead of precombat main).
					if cmc > 0 {
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:     EvtUpkeep,
							SourceID:      sourceID,
							Controller:    controller,
							MatchPlayerID: controller,
							Effects: []Effect{FuncEffect(
								fmt.Sprintf("add %d colorless mana", cmc),
								EffectProperties{Outcome: OutcomeBenefit},
								func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
									p := g.GetPlayer(controller)
									if p != nil {
										p.ManaPool().Add(Colorless, cmc)
									}
									return nil
								},
							)},
						})
					}
					return nil
				},
			)),
		)
	})

	// Part Water {X}{X}{U}
	// Sorcery
	// X target creatures gain islandwalk until end of turn. (They can't be blocked as long as defending player controls an Island.)
	// XXX: requires X targets — engine lacks variable target count based on X value
	// TODO: implement when X-count targeting is supported
	Register("Part Water", func() Card {
		return NewSorcery("Part Water", "{X}{X}{U}",
			NewSpellAbility(),
		)
	})

	// Psychic Purge {U}
	// Sorcery
	// Psychic Purge deals 1 damage to any target.
	// When a spell or ability an opponent controls causes you to discard this card, that player loses 5 life.
	// XXX: discard trigger not implemented — just deals 1 damage
	Register("Psychic Purge", func() Card {
		return NewSorcery("Psychic Purge", "{U}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(1))),
		)
	})

	// Pyrotechnics {4}{R}
	// Sorcery
	// Pyrotechnics deals 4 damage divided as you choose among any number of targets.
	// XXX: divided damage among multiple targets not supported — deals 4 to single target
	Register("Pyrotechnics", func() Card {
		return NewSorcery("Pyrotechnics", "{4}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(4))),
		)
	})

	// Rapid Fire {3}{W}
	// Instant
	// Cast this spell only before blockers are declared.
	// Target creature gains first strike until end of turn. If it doesn't have rampage, that creature gains rampage 2 until end of turn. (Whenever the creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	// XXX: timing restriction not enforced
	Register("Rapid Fire", func() Card {
		return NewInstant("Rapid Fire", "{3}{W}",
			NewTargetedSpell(TargetCreature(),
				Pipeline("first strike + rampage 2 unless target already has rampage",
					EffectProperties{Outcome: OutcomeBenefit},
					GrantKeyword(FirstStrike),
					GrantAbility(RampageTrigger(2)).Unless(&TargetHasRampageCond{}),
				),
			),
		)
	})

	// Rebirth {3}{G}{G}{G}
	// Sorcery
	// Remove this card from your deck before playing if you're not playing for ante.
	// Each player may ante the top card of their library. If a player does, that player's life total becomes 20.
	// UNIMPLEMENTABLE: Ante mechanic — requires ante zone and permanent ownership changes.
	Register("Rebirth", func() Card {
		return NewSorcery("Rebirth", "{3}{G}{G}{G}",
			NewSpellAbility(),
		)
	})

	// Recall {X}{X}{U}
	// Sorcery
	// Discard X cards, then return a card from your graveyard to your hand for each card discarded this way. Exile Recall.
	Register("Recall", func() Card {
		return NewSorcery("Recall", "{X}{X}{U}",
			NewSpellAbility(FuncEffect(
				"discard X cards, then return X cards from graveyard to hand; exile Recall",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					x := g.XValue()
					if x <= 0 {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					// Discard X cards
					discarded := p.ChooseCardsFromHand(x, "Recall: choose cards to discard", g)
					for _, card := range discarded {
						p.RemoveFromHand(card.ID())
						p.AddToGraveyard(card)
					}
					// Return that many cards from graveyard to hand (player chooses)
					for range discarded {
						gy := p.Graveyard()
						if len(gy) == 0 {
							break
						}
						chosen := p.ChooseCardFromLibrary(gy, "Recall: choose a card to return from graveyard", g)
						if chosen == nil {
							break
						}
						card, ok := p.RemoveFromGraveyard(chosen.ID())
						if ok {
							p.AddToHand(card)
						}
					}
					// Exile Recall
					recallCard := g.FindCardAnywhere(sourceID)
					if recallCard != nil {
						g.ExileCard(recallCard, controller)
					}
					return nil
				},
			)),
		)
	})

	// Reincarnation {1}{G}{G}
	// Instant
	// Choose target creature. When that creature dies this turn, return a creature card from its owner's graveyard to the battlefield under the control of that creature's owner.
	Register("Reincarnation", func() Card {
		return NewInstant("Reincarnation", "{1}{G}{G}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"when target creature dies this turn, reanimate from its owner's graveyard",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetID := targets[0]
					perm := g.FindPermanent(targetID)
					if perm == nil {
						return nil
					}
					ownerID := perm.Card.Owner()
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:     EvtZoneChange,
						SourceID:      sourceID,
						Controller:    controller,
						MatchEventID:  targetID,
						MatchFromZone: ZoneBattlefield,
						MatchToZone:   ZoneGraveyard,
						Effects: []Effect{FuncEffect(
							"return a creature card from owner's graveyard to the battlefield",
							EffectProperties{Outcome: OutcomeBenefit},
							func(g *Game, _ uuid.UUID, _ uuid.UUID, _ []uuid.UUID) error {
								owner := g.GetPlayer(ownerID)
								if owner == nil {
									return nil
								}
								// Find creature cards in owner's graveyard
								for _, card := range owner.Graveyard() {
									if card.HasType(TypeCreature) {
										c, ok := owner.RemoveFromGraveyard(card.ID())
										if ok {
											g.PutOnBattlefield(c, ownerID)
										}
										return nil
									}
								}
								return nil
							},
						)},
					})
					return nil
				},
			)),
		)
	})

	// Remove Enchantments {W}
	// Instant
	// Return to your hand all enchantments you both own and control, all Auras you own attached to permanents you control, and all Auras you own attached to attacking creatures your opponents control. Then destroy all other enchantments you control, all other Auras attached to permanents you control, and all other Auras attached to attacking creatures your opponents control.
	// XXX: complex enchantment/Aura sorting by ownership, control, and attachment — engine lacks fine-grained Aura ownership checks
	// TODO: implement when Aura ownership/attachment queries are supported
	Register("Remove Enchantments", func() Card {
		return NewInstant("Remove Enchantments", "{W}",
			NewSpellAbility(),
		)
	})

	// Remove Soul {1}{U}
	// Instant
	// Counter target creature spell.
	Register("Remove Soul", func() Card {
		return NewInstant("Remove Soul", "{1}{U}",
			NewTargetedSpell(
				TargetSpellOnStack(NewCardFilter("creature spell", func(c Card) bool {
					return c.HasType(TypeCreature)
				})),
				CounterSpell(),
			),
		)
	})

	// Reset {U}{U}
	// Instant
	// Cast this spell only during an opponent's turn after their upkeep step.
	// Untap all lands you control.
	// XXX: timing restriction not enforced
	Register("Reset", func() Card {
		return NewInstant("Reset", "{U}{U}",
			NewSpellAbility(ForEachControlledPermanent(
				SelectController(),
				IsLand,
				UntapTargetStep(),
				"untap all lands you control",
			)),
		)
	})

	// Reverberation {2}{U}{U}
	// Instant
	// All damage that would be dealt this turn by target sorcery spell is dealt to that spell's controller instead.
	// XXX: requires redirecting all damage from a specific sorcery spell to its controller — engine lacks per-spell damage redirection
	// TODO: implement when per-spell damage redirection is supported
	Register("Reverberation", func() Card {
		return NewInstant("Reverberation", "{2}{U}{U}",
			NewSpellAbility(),
		)
	})

	// Rust {G}
	// Instant
	// Counter target activated ability from an artifact source. (Mana abilities can't be targeted.)
	// XXX: requires targeting activated abilities on the stack from artifact sources — engine lacks ability-on-stack targeting
	// TODO: implement when stack-based ability targeting is supported
	Register("Rust", func() Card {
		return NewInstant("Rust", "{G}",
			NewSpellAbility(),
		)
	})

	// Sea Kings' Blessing {U}
	// Instant
	// One or more target creatures become blue until end of turn.
	Register("Sea Kings' Blessing", func() Card {
		return NewInstant("Sea Kings' Blessing", "{U}",
			NewTargetedSpell(TargetCreature(), ChangeColorEffect(Blue)),
		)
	})

	// Shield Wall {1}{W}
	// Instant
	// Creatures you control get +0/+2 until end of turn.
	Register("Shield Wall", func() Card {
		return NewInstant("Shield Wall", "{1}{W}",
			NewSpellAbility(Boost(Fixed(0), Fixed(2)).Targeting(ToMatching(AnyPermanent))),
		)
	})

	// Silhouette {1}{U}
	// Instant
	// Choose target creature. If a spell or ability that targets that creature would cause a source to deal damage to that creature this turn, prevent that damage.
	// XXX: requires conditional damage prevention based on whether the damage source targeted the creature — engine lacks this type of conditional prevention
	// TODO: implement when targeted-source damage prevention is supported
	Register("Silhouette", func() Card {
		return NewInstant("Silhouette", "{1}{U}",
			NewSpellAbility(),
		)
	})

	// Storm Seeker {3}{G}
	// Instant
	// Storm Seeker deals damage to target player equal to the number of cards in that player's hand.
	Register("Storm Seeker", func() Card {
		return NewInstant("Storm Seeker", "{3}{G}",
			NewTargetedSpell(TargetPlayer(), HandSizeDamageEffect(0, true)),
		)
	})

	// Subdue {G}
	// Instant
	// Prevent all combat damage that would be dealt by target creature this turn. That creature gets +0/+X until end of turn, where X is its mana value.
	Register("Subdue", func() Card {
		return NewInstant("Subdue", "{G}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"prevent all combat damage by target creature; it gets +0/+X where X is its mana value",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					targetID := perm.ID()
					cmc := perm.Card.ManaCost().CMC()
					// Prevent all combat damage dealt by this creature
					eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, _ uuid.UUID) error {
						g.AddDamagePreventionRule(WithCombatOnly(), WithFrom(NewPermanentFilter("subdued creature", func(p *Permanent, _ *Game) bool {
							return p.ID() == targetID
						})))
						return nil
					})
					eff.SetSourceID(sourceID)
					g.AddContinuousEffect(eff)
					// +0/+X where X is its mana value
					if cmc > 0 {
						ce := TemporaryBoost(targetID, 0, cmc)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
					}
					return nil
				},
			)),
		)
	})

	// Sylvan Paradise {G}
	// Instant
	// One or more target creatures become green until end of turn.
	Register("Sylvan Paradise", func() Card {
		return NewInstant("Sylvan Paradise", "{G}",
			NewTargetedSpell(TargetCreature(), ChangeColorEffect(Green)),
		)
	})

	// Syphon Soul {2}{B}
	// Sorcery
	// Syphon Soul deals 2 damage to each other player. You gain life equal to the damage dealt this way.
	Register("Syphon Soul", func() Card {
		return NewSorcery("Syphon Soul", "{2}{B}",
			NewSpellAbility(FuncEffect(
				"deal 2 damage to each opponent; gain life equal to damage dealt",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					totalDamage := 0
					for _, pl := range g.AllPlayers() {
						if pl.PlayerID() != controller {
							g.DealDamageToPlayer(pl, 2, sourceID)
							totalDamage += 2
						}
					}
					if totalDamage > 0 {
						p := g.GetPlayer(controller)
						if p != nil {
							g.PlayerGainLife(p, totalDamage)
						}
					}
					return nil
				},
			)),
		)
	})

	// Telekinesis {U}{U}
	// Instant
	// Tap target creature. Prevent all combat damage that would be dealt by that creature this turn. It doesn't untap during its controller's next two untap steps.
	Register("Telekinesis", func() Card {
		return NewInstant("Telekinesis", "{U}{U}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"tap target creature; prevent its combat damage; it doesn't untap during next two untap steps",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					g.TapPermanent(perm)
					// Prevent combat damage from this creature this turn
					g.PreventAllDamageFrom(targets[0])
					// Doesn't untap during controller's next two untap steps
					// In 2-player, that's 4 game turns from now
					targetID := targets[0]
					expiryTurn := g.CurrentTurn() + 4
					ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
						p := g.FindPermanent(targetID)
						if p != nil {
							g.GrantAttr(p.ID(), AttrDoesNotUntap)
						}
						return nil
					}, func(g *Game, _ uuid.UUID) bool {
						return g.CurrentTurn() <= expiryTurn && g.FindPermanent(targetID) != nil
					})
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					return nil
				},
			)),
		)
	})

	// Teleport {U}{U}{U}
	// Instant
	// Cast this spell only during the declare attackers step.
	// Target creature can't be blocked this turn.
	// XXX: timing restriction not enforced
	Register("Teleport", func() Card {
		return NewInstant("Teleport", "{U}{U}{U}",
			NewTargetedSpell(TargetCreature(), MakeUnblockableUntilEndOfTurn()),
		)
	})

	// Touch of Darkness {B}
	// Instant
	// One or more target creatures become black until end of turn.
	Register("Touch of Darkness", func() Card {
		return NewInstant("Touch of Darkness", "{B}",
			NewTargetedSpell(TargetCreature(), ChangeColorEffect(Black)),
		)
	})

	// Transmutation {1}{B}
	// Instant
	// Switch target creature's power and toughness until end of turn.
	Register("Transmutation", func() Card {
		return NewInstant("Transmutation", "{1}{B}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"switch target creature's power and toughness until end of turn",
				EffectProperties{Outcome: OutcomeUnknown},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					power := perm.CurrentPower(g)
					toughness := perm.CurrentToughness(g)
					// Set base P/T to swapped values until end of turn
					ce := SetBasePT(perm.ID(), toughness, power)
					ce.SetSourceID(sourceID)
					g.AddContinuousEffect(ce)
					return nil
				},
			)),
		)
	})

	// Typhoon {2}{G}
	// Sorcery
	// Typhoon deals damage to each opponent equal to the number of Islands that player controls.
	Register("Typhoon", func() Card {
		return NewSorcery("Typhoon", "{2}{G}",
			NewSpellAbility(FuncEffect(
				"deal damage to each opponent equal to the number of Islands they control",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					for _, pl := range g.AllPlayers() {
						if pl.PlayerID() != controller {
							islands := g.FilterBattlefield(And(IsLand, HasSubType("Island"), ControlledBy(pl.PlayerID())))
							if len(islands) > 0 {
								g.DealDamageToPlayer(pl, len(islands), sourceID)
							}
						}
					}
					return nil
				},
			)),
		)
	})

	// Untamed Wilds {2}{G}
	// Sorcery
	// Search your library for a basic land card, put that card onto the battlefield, then shuffle.
	Register("Untamed Wilds", func() Card {
		return NewSorcery("Untamed Wilds", "{2}{G}",
			NewSpellAbility(SearchLibraryToBattlefield(NewCardFilter("basic land card", isBasicLand))),
		)
	})

	// Visions {W}
	// Sorcery
	// Look at the top five cards of target player's library. You may then have that player shuffle that library.
	// XXX: requires looking at top N cards of library and optional shuffle — engine lacks library peek with optional shuffle
	// TODO: implement when library peek is supported
	Register("Visions", func() Card {
		return NewSorcery("Visions", "{W}",
			NewSpellAbility(),
		)
	})

	// Winds of Change {R}
	// Sorcery
	// Each player shuffles the cards from their hand into their library, then draws that many cards.
	Register("Winds of Change", func() Card {
		return NewSorcery("Winds of Change", "{R}",
			NewSpellAbility(FuncEffect(
				"each player shuffles hand into library, then draws that many cards",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					for _, pl := range g.AllPlayers() {
						hand := pl.Hand()
						handSize := len(hand)
						// Shuffle hand into library
						for _, c := range hand {
							pl.RemoveFromHand(c.ID())
							pl.AddToLibrary(c)
						}
						pl.ShuffleLibrary()
						// Draw that many cards
						for range handSize {
							pl.DrawCard()
						}
					}
					return nil
				},
			)),
		)
	})

	// Winter Blast {X}{G}
	// Sorcery
	// Tap X target creatures. Winter Blast deals 2 damage to each of those creatures with flying.
	// XXX: X-count targeting not supported — taps all creatures and deals 2 to those with flying as approximation
	Register("Winter Blast", func() Card {
		return NewSorcery("Winter Blast", "{X}{G}",
			NewSpellAbility(FuncEffect(
				"tap X target creatures; deal 2 damage to each with flying",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					x := g.XValue()
					if x <= 0 {
						return nil
					}
					// Get all creatures and tap up to X of them (opponent's first as heuristic)
					allCreatures := g.FilterBattlefield(IsCreature)
					tapped := 0
					// Prefer opponent's creatures
					for _, perm := range allCreatures {
						if tapped >= x {
							break
						}
						if perm.Controller != controller {
							g.TapPermanent(perm)
							if perm.HasAttr(Flying) {
								g.DealDamageToPermanent(perm, 2, sourceID)
							}
							tapped++
						}
					}
					// If still need more, tap own creatures
					for _, perm := range allCreatures {
						if tapped >= x {
							break
						}
						if perm.Controller == controller {
							g.TapPermanent(perm)
							if perm.HasAttr(Flying) {
								g.DealDamageToPermanent(perm, 2, sourceID)
							}
							tapped++
						}
					}
					return nil
				},
			)),
		)
	})

}

// Suppress unused import warnings.
var _ = fmt.Sprintf
var _ uuid.UUID
