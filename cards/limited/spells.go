package limited

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	Register("Swords to Plowshares", func() Card {
		return NewInstant("Swords to Plowshares", "{W}",
			NewTargetedSpell(TargetCreature(), Pipeline(
				"exile target creature. Its controller gains life equal to its power",
				EffectProperties{Outcome: OutcomeDetriment},
				SnapshotPermanent(SelectTarget, "victim"),
				ExileGathered("victim"),
				GainLifeFromVar("victim.controller", "victim.power"),
			)),
		)
	})

	Register("Disenchant", func() Card {
		return NewInstant("Disenchant", "{1}{W}",
			NewTargetedSpell(TargetArtifactOrEnchantment(), DestroyTargetPermanent()),
		)
	})

	Register("Healing Salve", func() Card {
		c := NewInstant("Healing Salve", "{W}",
			NewTargetedSpell(TargetDamageAnyTarget(), ModalEffect(
				"target player gains 3 life or prevent the next 3 damage that would be dealt to any target this turn",
				GainLifeTarget(Fixed(3)),
				PreventDamageToTarget(Fixed(3)),
			)),
		)
		c.SetModes([]string{
			"Target player gains 3 life",
			"Prevent the next 3 damage that would be dealt to any target this turn",
		})
		return c
	})

	Register("Armageddon", func() Card {
		return NewSorcery("Armageddon", "{3}{W}",
			NewSpellAbility(DestroyAllLands()),
		)
	})

	Register("Balance", func() Card {
		return NewSorcery("Balance", "{1}{W}",
			NewSpellAbility(BalanceEffect()),
		)
	})

	Register("Resurrection", func() Card {
		return NewSorcery("Resurrection", "{2}{W}{W}",
			NewTargetedSpell(TargetCreatureInYourGraveyard(), ReturnFromGraveyardToBattlefield()),
		)
	})

	Register("Reverse Damage", func() Card {
		return NewInstant("Reverse Damage", "{1}{W}{W}",
			NewSpellAbility(Pipeline("prevent the next source of damage to you and gain that much life",
				EffectProperties{Outcome: OutcomeBenefit},
				AddPreventionShieldToControllerStep(1000),
				AddReverseDamageShieldStep(),
			)),
		)
	})

	Register("Death Ward", func() Card {
		return NewInstant("Death Ward", "{W}",
			NewTargetedSpell(TargetCreature(), RegenerateTarget()),
		)
	})

	Register("Guardian Angel", func() Card {
		// XXX: missing repeatable {1} prevention ability after initial X prevention
		return NewInstant("Guardian Angel", "{X}{W}",
			// Prevent the next X damage that would be dealt to any target this turn
			NewTargetedSpell(TargetDamageAnyTarget(), PreventDamageToTarget(XValue())),
		)
	})

	// ===== BLUE SPELLS =====

	Register("Ancestral Recall", func() Card {
		return NewInstant("Ancestral Recall", "{U}",
			NewTargetedSpell(TargetPlayer(), DrawCards(Fixed(3))),
		)
	})

	Register("Braingeyser", func() Card {
		return NewSorcery("Braingeyser", "{X}{U}{U}",
			NewTargetedSpell(TargetPlayer(), DrawCards(XValue())),
		)
	})

	Register("Counterspell", func() Card {
		return NewInstant("Counterspell", "{U}{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpell()),
		)
	})

	Register("Unsummon", func() Card {
		return NewInstant("Unsummon", "{U}",
			NewTargetedSpell(TargetCreature(), ReturnToHandTarget()),
		)
	})

	Register("Spell Blast", func() Card {
		return NewInstant("Spell Blast", "{X}{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpellIfXMeetsCMC()),
		)
	})

	Register("Power Sink", func() Card {
		return NewInstant("Power Sink", "{X}{U}",
			NewTargetedSpell(TargetSpellOnStack(), PowerSinkEffect()),
		)
	})

	// XXX: missing destroy mode — Oracle is modal: "Counter target red spell" / "Destroy target red permanent"
	// Engine doesn't support modal targeting across zones (spell on stack vs permanent on battlefield)
	Register("Blue Elemental Blast", func() Card {
		return NewInstant("Blue Elemental Blast", "{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpellIfColor(Red)),
		)
	})

	Register("Twiddle", func() Card {
		return NewInstant("Twiddle", "{U}",
			NewTargetedSpell(TargetPermanent(Or(IsArtifact, IsCreature, IsLand)), TapOrUntapTarget()),
		)
	})

	Register("Timetwister", func() Card {
		return NewSorcery("Timetwister", "{2}{U}",
			NewSpellAbility(ShuffleHandAndGraveyardIntoLibraryAndDraw(7)),
		)
	})

	Register("Time Walk", func() Card {
		return NewSorcery("Time Walk", "{1}{U}",
			NewSpellAbility(ExtraTurn()),
		)
	})

	Register("Sleight of Mind", func() Card {
		return NewInstant("Sleight of Mind", "{U}",
			NewTargetedSpell(TargetPermanent(), ReplaceKeywordEffect(Swampwalk, Forestwalk)),
		)
	})

	// ===== BLACK SPELLS =====

	Register("Dark Ritual", func() Card {
		return NewInstant("Dark Ritual", "{B}",
			NewSpellAbility(AddMana(Black, 3)),
		)
	})

	Register("Terror", func() Card {
		return NewInstant("Terror", "{1}{B}",
			NewTargetedSpell(TargetCreature(
				Not(HasColorFilter(Black)),
				Not(IsArtifact),
			), DestroyTargetNoRegen()),
		)
	})

	Register("Raise Dead", func() Card {
		return NewSorcery("Raise Dead", "{B}",
			NewTargetedSpell(TargetCreatureInYourGraveyard(), ReturnFromGraveyardToHandTarget()),
		)
	})

	Register("Demonic Tutor", func() Card {
		return NewSorcery("Demonic Tutor", "{1}{B}",
			NewSpellAbility(SearchLibraryToHand()),
		)
	})

	Register("Drain Life", func() Card {
		// XXX: missing "Spend only black mana on X" restriction
		// TODO: convert to pipeline — needs life-capping logic (cap gain at target toughness/life)
		return NewSorcery("Drain Life", "{X}{1}{B}",
			NewTargetedSpell(TargetDamageAnyTarget(), FuncEffect(
				"deal X damage to target and gain life equal to damage dealt",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return fmt.Errorf("no target for drain life")
					}
					amount := g.XValue()
					if amount <= 0 {
						return nil
					}
					target := targets[0]
					lifeGain := amount
					perm := g.FindPermanent(target)
					if perm != nil {
						toughness := perm.CurrentToughness(g)
						if lifeGain > toughness {
							lifeGain = toughness
						}
						g.DealDamageToPermanent(perm, amount, sourceID)
					} else {
						p := g.GetPlayer(target)
						if p != nil {
							life := p.Life()
							if lifeGain > life {
								lifeGain = life
							}
							g.DealDamageToPlayer(p, amount, sourceID)
						}
					}
					caster := g.GetPlayer(controller)
					if caster != nil && lifeGain > 0 {
						caster.GainLife(lifeGain)
						g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: controller, Amount: lifeGain})
					}
					return nil
				})),
		)
	})

	Register("Mind Twist", func() Card {
		return NewSorcery("Mind Twist", "{X}{B}",
			NewTargetedSpell(TargetPlayer(), DiscardCards(XValue())),
		)
	})

	Register("Sinkhole", func() Card {
		return NewLandDestruction("Sinkhole", "{B}{B}")
	})

	// XXX: missing destroy mode — Oracle is modal: "Counter target blue spell" / "Destroy target blue permanent"
	// Engine doesn't support modal targeting across zones (spell on stack vs permanent on battlefield)
	Register("Red Elemental Blast", func() Card {
		return NewInstant("Red Elemental Blast", "{R}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpellIfColor(Blue)),
		)
	})

	// ===== RED SPELLS =====

	Register("Lightning Bolt", func() Card {
		return NewInstant("Lightning Bolt", "{R}",
			NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(3))),
		)
	})

	Register("Shock", func() Card {
		return NewInstant("Shock", "{R}",
			NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(2))),
		)
	})

	Register("Fireball", func() Card {
		return NewSorcery("Fireball", "{X}{R}",
			NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(XValue())),
		)
	})

	// XXX: missing exile-on-death replacement effect — Oracle: "if it would die this turn, exile it instead"
	// TODO: convert to pipeline — needs GrantAttrToTarget primitive for CantRegenerate
	Register("Disintegrate", func() Card {
		return NewSorcery("Disintegrate", "{X}{R}",
			NewTargetedSpell(TargetDamageAnyTarget(), FuncEffect(
				"deal X damage; creature can't be regenerated this turn",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return fmt.Errorf("no target for Disintegrate")
					}
					amount := g.XValue()
					target := targets[0]
					perm := g.FindPermanent(target)
					if perm != nil {
						perm.GrantBaseAttr(CantRegenerate)
						g.DealDamageToPermanent(perm, amount, sourceID)
					} else {
						p := g.GetPlayer(target)
						if p != nil {
							g.DealDamageToPlayer(p, amount, sourceID)
						}
					}
					return nil
				})),
		)
	})

	Register("Earthquake", func() Card {
		return NewSorcery("Earthquake", "{X}{R}",
			NewSpellAbility(CompositeEffects(
				"deal X damage to each creature without flying and each player",
				DealDamageToAllCreatures(XValue(), NotHasKeywordFilter(Flying)),
				DealDamageToPlayers(XValue(), SelectEachPlayer()),
			)),
		)
	})

	Register("Shatter", func() Card {
		return NewInstant("Shatter", "{1}{R}",
			NewTargetedSpell(TargetArtifact(), DestroyTargetArtifact()),
		)
	})

	Register("Stone Rain", func() Card {
		return NewLandDestruction("Stone Rain", "{2}{R}")
	})

	Register("Flashfires", func() Card {
		return NewSorcery("Flashfires", "{3}{R}",
			NewSpellAbility(DestroyAllMatching(
				HasSubType("Plains"),
				"destroy all Plains",
			)),
		)
	})

	Register("Wheel of Fortune", func() Card {
		return NewSorcery("Wheel of Fortune", "{2}{R}",
			NewSpellAbility(DiscardHandAndDraw(7)),
		)
	})

	Register("Fork", func() Card {
		return NewInstant("Fork", "{R}{R}",
			NewTargetedSpell(TargetSpellOnStack(), CopySpellOnStack()),
		)
	})

	Register("Berserk", func() Card {
		return NewInstant("Berserk", "{G}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"Target creature gains trample and gets +X/+0. Destroy at end of turn if it attacked.",
				GrantKeyword(Trample),
				DoubleTargetPower(),
				// TODO: convert to pipeline — needs DelayedTrigger + HasAttackedThisTurn primitives
				FuncEffect("destroy at end of turn if attacked", EffectProperties{}, func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetID := targets[0]
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:  EvtEndStep,
						SourceID:   sourceID,
						Controller: controller,
						Effects: []Effect{FuncEffect(
							"destroy creature if it attacked",
							EffectProperties{},
							func(g2 *Game, srcID, ctrlID uuid.UUID, _ []uuid.UUID) error {
								if g2.HasAttackedThisTurn(targetID) {
									perm := g2.FindPermanent(targetID)
									if perm != nil {
										g2.DestroyPermanent(perm)
									}
								}
								return nil
							}),
						},
					})
					return nil
				}),
			)),
		)
	})

	// ===== GREEN SPELLS =====

	Register("Giant Growth", func() Card {
		return NewInstant("Giant Growth", "{G}",
			NewTargetedSpell(TargetCreature(), Boost(Fixed(3), Fixed(3))),
		)
	})

	Register("Regrowth", func() Card {
		return NewSorcery("Regrowth", "{1}{G}",
			NewTargetedSpell(TargetCardInYourGraveyard(), ReturnFromGraveyardToHandTarget()),
		)
	})

	Register("Hurricane", func() Card {
		return NewSorcery("Hurricane", "{X}{G}",
			NewSpellAbility(CompositeEffects(
				"deal X damage to each creature with flying and each player",
				DealDamageToAllCreatures(XValue(), HasKeywordFilter(Flying)),
				DealDamageToPlayers(XValue(), SelectEachPlayer()),
			)),
		)
	})

	Register("Tranquility", func() Card {
		return NewSorcery("Tranquility", "{2}{G}",
			NewSpellAbility(DestroyAllEnchantments()),
		)
	})

	Register("Tsunami", func() Card {
		return NewSorcery("Tsunami", "{3}{G}",
			NewSpellAbility(DestroyAllMatching(
				HasSubType("Island"),
				"destroy all Islands",
			)),
		)
	})

	Register("Ice Storm", func() Card {
		return NewLandDestruction("Ice Storm", "{2}{G}")
	})

	Register("Stream of Life", func() Card {
		return NewSorcery("Stream of Life", "{X}{G}",
			NewTargetedSpell(TargetPlayer(), GainLifeTarget(XValue())),
		)
	})

	Register("Fog", func() Card {
		return NewInstant("Fog", "{G}",
			NewSpellAbility(PreventAllCombatDamage()),
		)
	})

	Register("Channel", func() Card {
		// TODO: convert to pipeline — needs SetChannelActive primitive
		return NewSorcery("Channel", "{G}{G}",
			NewSpellAbility(FuncEffect(
				"until end of turn, pay 1 life to add {C}",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					g.SetChannelActive(controller)
					return nil
				})),
		)
	})

	Register("Howl from Beyond", func() Card {
		return NewInstant("Howl from Beyond", "{X}{B}",
			NewTargetedSpell(TargetCreature(), Boost(XValue(), Fixed(0))),
		)
	})

	Register("Righteousness", func() Card {
		return NewInstant("Righteousness", "{W}",
			NewTargetedSpell(TargetCreature(IsBlocking), Boost(Fixed(7), Fixed(7))),
		)
	})

	// ===== COLORLESS SPELLS =====

	Register("Wrath of God", func() Card {
		return NewSorcery("Wrath of God", "{2}{W}{W}",
			NewSpellAbility(DestroyAllCreaturesNoRegen()),
		)
	})

	// Psionic Blast {2}{U}
	// Instant
	// Psionic Blast deals 4 damage to any target and 2 damage to you.
	Register("Psionic Blast", func() Card {
		return NewInstant("Psionic Blast", "{2}{U}",
			NewTargetedSpell(TargetDamageAnyTarget(), CompositeEffects(
				"deal 4 damage to any target and 2 damage to you",
				DealDamage(Fixed(4)),
				DealDamageToPlayers(Fixed(2), SelectController()),
			)),
		)
	})

	// Tunnel {R}
	// Instant
	// Destroy target Wall. It can't be regenerated.
	Register("Tunnel", func() Card {
		return NewInstant("Tunnel", "{R}",
			NewTargetedSpell(TargetCreature(HasSubType("Wall")), DestroyTargetNoRegen()),
		)
	})

	// ===== ANTE CARDS =====

	// Contract from Below {B}
	// Sorcery
	// Remove this card from your deck before playing if you're not playing for ante.
	// Discard your hand, ante the top card of your library, then draw seven cards.
	// TODO: convert to pipeline — needs ante manipulation primitives
	Register("Contract from Below", func() Card {
		return NewSorcery("Contract from Below", "{B}",
			NewSpellAbility(FuncEffect(
				"discard hand, ante top card, draw seven",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					for _, c := range p.Hand() {
						p.DiscardCard(c.ID())
					}
					lib := p.Library()
					if len(lib) > 0 {
						top := lib[0]
						p.SetLibrary(lib[1:])
						p.AddToAnte(top)
					}
					for range 7 {
						g.PlayerDrawCard(p)
					}
					return nil
				})),
		)
	})

	// Darkpact {B}{B}{B}
	// Sorcery
	// Remove this card from your deck before playing if you're not playing for ante.
	// You own target card in the ante. Exchange that card with the top card of your library.
	// TODO: convert to pipeline — needs ante manipulation primitives
	Register("Darkpact", func() Card {
		return NewSorcery("Darkpact", "{B}{B}{B}",
			NewSpellAbility(FuncEffect(
				"exchange ante card with top of library",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					ante := p.Ante()
					if len(ante) == 0 {
						return nil
					}
					lib := p.Library()
					if len(lib) == 0 {
						return nil
					}
					anteCard, ok := p.RemoveFromAnte(ante[0].ID())
					if !ok {
						return nil
					}
					topLib := lib[0]
					p.SetLibrary(append([]Card{anteCard}, lib[1:]...))
					p.AddToAnte(topLib)
					return nil
				})),
		)
	})

	// Demonic Attorney {1}{B}{B}
	// Sorcery
	// Remove this card from your deck before playing if you're not playing for ante.
	// Each player antes the top card of their library.
	// TODO: convert to pipeline — needs ante manipulation primitives
	Register("Demonic Attorney", func() Card {
		return NewSorcery("Demonic Attorney", "{1}{B}{B}",
			NewSpellAbility(FuncEffect(
				"each player antes the top card of their library",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					for _, p := range g.AllPlayers() {
						lib := p.Library()
						if len(lib) > 0 {
							top := lib[0]
							p.SetLibrary(lib[1:])
							p.AddToAnte(top)
						}
					}
					return nil
				})),
		)
	})
	// ===== MOVED FROM OTHER FILES =====

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
					n := min(len(lib), 3)
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
			NewTargetedSpell(TargetCreature(), GrantKeyword(CanBlockAny)),
		)
	})

	// TODO implement
	// "Text": "Cast this spell only during the declare blockers step.\nRemove target creature defending player controls from combat. Creatures it was blocking that had become blocked by only that creature this combat become unblocked. You may have it block an attacking creature of your choice.",
	Register("False Orders", func() Card {
		return NewInstant("False Orders", "{R}",
			NewTargetedSpell(TargetCreature(), RemoveFromCombat()),
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

	Register("Jump", func() Card {
		return NewInstant("Jump", "{U}",
			NewTargetedSpell(TargetCreature(), GrantKeyword(Flying)),
		)
	})

}
