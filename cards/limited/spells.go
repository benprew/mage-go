package limited

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
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
			NewTargetedSpell(TargetAnyTarget(), DataEffect(ModalEffect(
				"target player gains 3 life or prevent the next 3 damage that would be dealt to any target this turn",
				UnwrapEffect(GainLifeTarget(Fixed(3))),
				UnwrapEffect(PreventDamageToTarget(Fixed(3))),
			))),
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
			NewTargetedSpell(TargetAnyTarget(), PreventDamageToTarget(XValue())),
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

	Register("Stasis", func() Card {
		return NewEnchantment("Stasis", "{1}{U}",
			// Players skip their untap steps.
			WithStaticAbility(PreventAllUntaps()),
			// At the beginning of your upkeep, sacrifice Stasis unless you pay {U}.
			WithAbility(SacrificeAtUpkeepUnlessPay("{U}")),
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
			NewTargetedSpell(TargetAnyTarget(), FuncEffect(
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

	Register("Pestilence", func() Card {
		return NewEnchantment("Pestilence", "{2}{B}{B}",
			// At the beginning of the end step, if no creatures are on the battlefield, sacrifice Pestilence.
			WithAbility(BeginningOfEachEndStepTrigger(
				DataEffect(IfElse("sacrifice if no creatures",
					&NotCond{Inner: &HasMatchingPermanentCond{Filter: IsCreature}},
					SacrificeSourceStep(),
					nil,
				)), false,
			)),
			// {B}: Deal 1 damage to each creature and each player
			WithActivatedAbility(
				CompositeEffects("deal 1 damage to each creature and each player",
					DealDamageToAllCreatures(Fixed(1), PermanentFilter{}),
					DealDamageToPlayers(Fixed(1), SelectEachPlayer()),
				),
				ManaCostOf("{B}"),
			),
		)
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
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(3))),
		)
	})

	Register("Fireball", func() Card {
		return NewSorcery("Fireball", "{X}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(XValue())),
		)
	})

	// XXX: missing exile-on-death replacement effect — Oracle: "if it would die this turn, exile it instead"
	// TODO: convert to pipeline — needs GrantAttrToTarget primitive for CantRegenerate
	Register("Disintegrate", func() Card {
		return NewSorcery("Disintegrate", "{X}{R}",
			NewTargetedSpell(TargetAnyTarget(), FuncEffect(
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
				GrantKeywordUntilEndOfTurn(Trample, SelectTarget),
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
			NewTargetedSpell(TargetCreature(), BoostUntilEndOfTurn(Fixed(3), Fixed(3), SelectTarget)),
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
			NewTargetedSpell(TargetCreature(), BoostUntilEndOfTurn(XValue(), Fixed(0), SelectTarget)),
		)
	})

	Register("Righteousness", func() Card {
		return NewInstant("Righteousness", "{W}",
			NewTargetedSpell(TargetCreature(IsBlocking), BoostUntilEndOfTurn(Fixed(7), Fixed(7), SelectTarget)),
		)
	})

	// ===== COLORLESS SPELLS =====

	Register("Chaos Orb", func() Card {
		return NewArtifact("Chaos Orb", "{2}",
			// {1}, {T}: Destroy a random nontoken permanent you don't control, then destroy Chaos Orb.
			WithActivatedAbility(
				ChaosOrbEffect(),
				GenericCost(1),
				WithCost(TapSourceCost()),
			),
		)
	})

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
			NewTargetedSpell(TargetAnyTarget(), CompositeEffects(
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
					for i := 0; i < 7; i++ {
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
}
