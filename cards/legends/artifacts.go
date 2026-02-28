package legends

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerArtifacts()
}

// manaBattery creates a Mana Battery artifact with two abilities:
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters: Add one colored mana + one per counter removed.
func manaBattery(name string, color Color) Card {
	return NewArtifact(name, "{4}",
		// {2}, {T}: Put a charge counter on this artifact.
		WithActivatedAbility(
			AddCounters(Charge, Fixed(1), SelectSource),
			ManaCostOf("{2}"),
			WithCost(TapSourceCost()),
		),
		// {T}, Remove any number of charge counters: Add colored mana = counters removed + 1
		WithActivatedAbility(
			FuncEffect(
				"add mana equal to charge counters removed plus one",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.FindPermanent(sourceID)
					if p == nil {
						return nil
					}
					counters := p.Counters[Charge]
					if counters > 0 {
						p.RemoveCounter(Charge, counters)
					}
					// Add 1 base + 1 per counter removed
					total := 1 + counters
					player := g.GetPlayer(controller)
					if player != nil {
						player.ManaPool().Add(color, total)
					}
					return nil
				},
			),
			TapSourceCost(),
		),
	)
}

func registerArtifacts() {

// Al-abara's Carpet {5}
// Artifact
// {5}, {T}: Prevent all damage that would be dealt to you this turn by attacking creatures without flying.
	Register("Al-abara's Carpet", withExpansion(func() Card {
		return NewArtifact("Al-abara's Carpet", "{5}",
			WithActivatedAbility(
				FuncEffect(
					"prevent all damage that would be dealt to you this turn by attacking creatures without flying",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, srcID uuid.UUID) error {
							g.Effects.Damage.AddDamagePreventionRule(
								WithFrom(And(IsAttacking, Not(HasKeywordFilter(Flying)))),
							)
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						return nil
					},
				),
				ManaCostOf("{5}"),
				WithCost(TapSourceCost()),
			),
		)
	}))


// Alchor's Tomb {4}
// Artifact
// {2}, {T}: Target permanent you control becomes the color of your choice. (This effect lasts indefinitely.)
	Register("Alchor's Tomb", withExpansion(func() Card {
		return NewArtifact("Alchor's Tomb", "{4}",
			WithActivatedAbility(
				FuncEffect(
					"target permanent you control becomes the color of your choice",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						color := p.ChooseManaColor("choose a color")
						ce := ColorOverride(targets[0], color)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					},
				),
				ManaCostOf("{2}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetPermanent()),
			),
		)
	}))


// Arena of the Ancients {3}
// Artifact
// Legendary creatures don't untap during their controllers' untap steps.
// When this artifact enters, tap all legendary creatures.
	Register("Arena of the Ancients", withExpansion(func() Card {
		return NewArtifact("Arena of the Ancients", "{3}",
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.Battlefield {
					if p.HasType(TypeCreature) && p.Card.HasSuperType(SuperLegendary) {
						g.Effects.GrantAttr(p.ID(), AttrDoesNotUntap)
					}
				}
				return nil
			})),
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"tap all legendary creatures",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					for _, p := range g.FilterBattlefield(NewPermanentFilter("legendary creature", func(p *Permanent, _ *Game) bool {
						return p.HasType(TypeCreature) && p.Card.HasSuperType(SuperLegendary)
					})) {
						p.Tapped = true
					}
					return nil
				},
			), false)),
		)
	}))


// Black Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {B}, then add an additional {B} for each charge counter removed this way.
	Register("Black Mana Battery", withExpansion(func() Card {
		return manaBattery("Black Mana Battery", Black)
	}))


// Blue Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {U}, then add an additional {U} for each charge counter removed this way.
	Register("Blue Mana Battery", withExpansion(func() Card {
		return manaBattery("Blue Mana Battery", Blue)
	}))


// Forethought Amulet {5}
// Artifact
// At the beginning of your upkeep, sacrifice this artifact unless you pay {3}.
// If an instant or sorcery source would deal 3 or more damage to you, it deals 2 damage to you instead.
// TODO: implement
	Register("Forethought Amulet", withExpansion(func() Card {
		return NewArtifact("Forethought Amulet", "{5}")
	}))


// Gauntlets of Chaos {5}
// Artifact
// {5}, Sacrifice this artifact: Exchange control of target artifact, creature, or land you control and target permanent an opponent controls that shares one of those types with it. If those permanents are exchanged this way, destroy all Auras attached to them.
// TODO: implement
	Register("Gauntlets of Chaos", withExpansion(func() Card {
		return NewArtifact("Gauntlets of Chaos", "{5}")
	}))


// Green Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {G}, then add an additional {G} for each charge counter removed this way.
	Register("Green Mana Battery", withExpansion(func() Card {
		return manaBattery("Green Mana Battery", Green)
	}))


// Horn of Deafening {4}
// Artifact
// {2}, {T}: Prevent all combat damage that would be dealt by target creature this turn.
	Register("Horn of Deafening", withExpansion(func() Card {
		return NewArtifact("Horn of Deafening", "{4}",
			WithActivatedAbility(
				FuncEffect(
					"prevent all combat damage that would be dealt by target creature this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, srcID uuid.UUID) error {
							g.Effects.Damage.AddDamagePreventionRule(WithFrom(NewPermanentFilter("prevented source", func(p *Permanent, _ *Game) bool {
								return p.ID() == targets[0]
							})))
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						return nil
					},
				),
				ManaCostOf("{2}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))


// Knowledge Vault {4}
// Artifact
// {2}, {T}: Exile the top card of your library face down.
// {0}: Sacrifice this artifact. If you do, discard your hand, then put all cards exiled with this artifact into their owner's hand.
// When this artifact leaves the battlefield, put all cards exiled with it into their owner's graveyard.
// TODO: implement
	Register("Knowledge Vault", withExpansion(func() Card {
		return NewArtifact("Knowledge Vault", "{4}")
	}))


// Kry Shield {2}
// Artifact
// {2}, {T}: Prevent all damage that would be dealt this turn by target creature you control. That creature gets +0/+X until end of turn, where X is its mana value.
	Register("Kry Shield", withExpansion(func() Card {
		return NewArtifact("Kry Shield", "{2}",
			WithActivatedAbility(
				FuncEffect(
					"prevent all damage that would be dealt this turn by target creature you control; that creature gets +0/+X until end of turn, where X is its mana value",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := g.FindPermanent(targets[0])
						if target == nil {
							return nil
						}
						cmc := target.Card.ManaCost().CMC()
						// Prevent all damage from target creature this turn
						eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, srcID uuid.UUID) error {
							g.Effects.Damage.AddDamagePreventionRule(WithFrom(NewPermanentFilter("prevented source", func(p *Permanent, _ *Game) bool {
								return p.ID() == targets[0]
							})))
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						// Target creature gets +0/+X where X is mana value
						boost := TemporaryBoost(targets[0], 0, cmc)
						boost.SetSourceID(sourceID)
						g.AddContinuousEffect(boost)
						return nil
					},
				),
				ManaCostOf("{2}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))


// Life Chisel {4}
// Artifact
// Sacrifice a creature: You gain life equal to the sacrificed creature's toughness. Activate only during your upkeep.
	Register("Life Chisel", withExpansion(func() Card {
		return NewArtifact("Life Chisel", "{4}",
			WithActivatedAbility(
				FuncEffect(
					"you gain life equal to the sacrificed creature's toughness",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// The sacrifice cost has already moved a creature to the graveyard.
						// Find the most recently added creature in controller's graveyard.
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						gy := p.Graveyard()
						for i := len(gy) - 1; i >= 0; i-- {
							if gy[i].HasType(TypeCreature) {
								toughness := gy[i].Toughness()
								p.GainLife(toughness)
								return nil
							}
						}
						return nil
					},
				),
				SacrificeCreatureCost(),
				WithUpkeepOnly(),
			),
		)
	}))


// Life Matrix {4}
// Artifact
// {4}, {T}: Put a matrix counter on target creature and that creature gains "Remove a matrix counter from this creature: Regenerate this creature." Activate only during your upkeep.
// TODO: implement
	Register("Life Matrix", withExpansion(func() Card {
		return NewArtifact("Life Matrix", "{4}")
	}))


// Mana Matrix {6}
// Artifact
// Instant and enchantment spells you cast cost {2} less to cast.
	Register("Mana Matrix", withExpansion(func() Card {
		return NewArtifact("Mana Matrix", "{6}",
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.Effects.Rules.SpellTypeCostReductions[TypeInstant] += 2
				g.Effects.Rules.SpellTypeCostReductions[TypeEnchantment] += 2
				return nil
			})),
		)
	}))


// Mirror Universe {6}
// Artifact
// {T}, Sacrifice this artifact: Exchange life totals with target opponent. Activate only during your upkeep.
	Register("Mirror Universe", withExpansion(func() Card {
		return NewArtifact("Mirror Universe", "{6}",
			WithActivatedAbility(
				FuncEffect(
					"exchange life totals with target opponent",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						me := g.GetPlayer(controller)
						opp := g.GetOpponent(controller)
						if me == nil || opp == nil {
							return nil
						}
						myLife := me.Life()
						oppLife := opp.Life()
						me.SetLife(oppLife)
						opp.SetLife(myLife)
						return nil
					},
				),
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
				WithUpkeepOnly(),
			),
		)
	}))


// North Star {4}
// Artifact
// {4}, {T}: For one spell this turn, you may spend mana as though it were mana of any type to pay that spell's mana cost. (Additional costs are still paid normally.)
// TODO: implement
	Register("North Star", withExpansion(func() Card {
		return NewArtifact("North Star", "{4}")
	}))


// Nova Pentacle {4}
// Artifact
// {3}, {T}: The next time a source of your choice would deal damage to you this turn, that damage is dealt to target creature of an opponent's choice instead.
// TODO: implement
	Register("Nova Pentacle", withExpansion(func() Card {
		return NewArtifact("Nova Pentacle", "{4}")
	}))


// Planar Gate {6}
// Artifact
// Creature spells you cast cost {2} less to cast.
	Register("Planar Gate", withExpansion(func() Card {
		return NewArtifact("Planar Gate", "{6}",
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.Effects.Rules.SpellTypeCostReductions[TypeCreature] += 2
				return nil
			})),
		)
	}))


// Red Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {R}, then add an additional {R} for each charge counter removed this way.
	Register("Red Mana Battery", withExpansion(func() Card {
		return manaBattery("Red Mana Battery", Red)
	}))


// Relic Barrier {2}
// Artifact
// {T}: Tap target artifact.
	Register("Relic Barrier", withExpansion(func() Card {
		return NewArtifact("Relic Barrier", "{2}",
			WithActivatedAbility(
				TapTarget(),
				TapSourceCost(),
				WithTarget(TargetArtifact()),
			),
		)
	}))


// Ring of Immortals {5}
// Artifact
// {3}, {T}: Counter target instant or Aura spell that targets a permanent you control.
// TODO: implement
	Register("Ring of Immortals", withExpansion(func() Card {
		return NewArtifact("Ring of Immortals", "{5}")
	}))


// Serpent Generator {6}
// Artifact
// {4}, {T}: Create a 1/1 colorless Snake artifact creature token. It has "Whenever this creature deals damage to a player, that player gets a poison counter." (A player with ten or more poison counters loses the game.)
	Register("Serpent Generator", withExpansion(func() Card {
		return NewArtifact("Serpent Generator", "{6}",
			WithActivatedAbility(
				FuncEffect(
					"create a 1/1 colorless Snake artifact creature token with poison",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						token := NewToken("Snake", 1, 1, []CardType{TypeArtifact, TypeCreature}, []string{"Snake"})
						token.SetOwner(controller)
						// Add poison trigger: whenever this creature deals damage to a player, that player gets a poison counter
						token.AddAbility(NewTriggered(EvtDamageDealt, false, FuncEffect(
							"poison counter",
							EffectProperties{},
							func(g GameMutator, srcID, ctrl uuid.UUID, _ []uuid.UUID) error {
								for _, p := range g.AllPlayers() {
									if p.PlayerID() != ctrl {
										p.AddPoisonCounters(1)
										return nil
									}
								}
								return nil
							},
						)).SetCondition(func(evt *GameEvent, g *Game, srcID, _ uuid.UUID) bool {
							// Only trigger when this creature deals damage to a player (not a permanent)
							return evt.SourceID == srcID && g.GetPlayer(evt.TargetID) != nil
						}))
						g.PutOnBattlefield(token, controller)
						return nil
					},
				),
				ManaCostOf("{4}"),
				WithCost(TapSourceCost()),
			),
		)
	}))


// Sword of the Ages {6}
// Artifact
// This artifact enters tapped.
// {T}, Sacrifice this artifact and any number of creatures you control: This artifact deals X damage to any target, where X is the total power of the creatures sacrificed this way, then exile this artifact and those creature cards.
// TODO: implement
	Register("Sword of the Ages", withExpansion(func() Card {
		return NewArtifact("Sword of the Ages", "{6}")
	}))


// Triassic Egg {4}
// Artifact
// {3}, {T}: Put a hatchling counter on this artifact.
// Sacrifice this artifact: Choose one. Activate only if there are two or more hatchling counters on this artifact.
// • You may put a creature card from your hand onto the battlefield.
// • Return target creature card from your graveyard to the battlefield.
	Register("Triassic Egg", withExpansion(func() Card {
		return NewArtifact("Triassic Egg", "{4}",
			// {3}, {T}: Put a hatchling counter
			WithActivatedAbility(
				AddCounters(Hatchling, Fixed(1), SelectSource),
				ManaCostOf("{3}"),
				WithCost(TapSourceCost()),
			),
			// Sacrifice: put creature from hand onto battlefield or reanimate from graveyard
			WithActivatedAbility(
				FuncEffect("put creature from hand onto battlefield or reanimate",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						mode := p.ChooseMode([]string{
							"Put a creature card from your hand onto the battlefield",
							"Return target creature card from your graveyard to the battlefield",
						}, "Triassic Egg")
						if mode == 0 {
							// Put creature from hand onto battlefield
							for _, card := range p.Hand() {
								if card.HasType(TypeCreature) {
									p.RemoveFromHand(card.ID())
									g.PutOnBattlefield(card, controller)
									return nil
								}
							}
						} else {
							// Return creature from graveyard to battlefield
							for i := len(p.Graveyard()) - 1; i >= 0; i-- {
								card := p.Graveyard()[i]
								if card.HasType(TypeCreature) {
									p.RemoveFromGraveyard(card.ID())
									g.PutOnBattlefield(card, controller)
									return nil
								}
							}
						}
						return nil
					},
				),
				SacrificeSourceCost(),
			),
		)
	}))


// Voodoo Doll {6}
// Artifact
// At the beginning of your upkeep, put a pin counter on this artifact.
// At the beginning of your end step, if this artifact is untapped, destroy this artifact and it deals damage to you equal to the number of pin counters on it.
// {X}{X}, {T}: This artifact deals damage equal to the number of pin counters on it to any target. X is the number of pin counters on this artifact.
	Register("Voodoo Doll", withExpansion(func() Card {
		return NewArtifact("Voodoo Doll", "{6}",
			// Upkeep: put a pin counter
			WithAbility(BeginningOfUpkeepTrigger(
				AddCounters(Pin, Fixed(1), SelectSource), false,
			)),
			// End step: if untapped, destroy and deal damage
			WithAbility(
				NewTriggered(EvtEndStep, false,
					FuncEffect("if untapped, destroy and deal damage equal to pin counters",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							src := g.FindPermanent(sourceID)
							if src == nil {
								return nil
							}
							pins := src.Counters[Pin]
							g.DestroyPermanent(src)
							if pins > 0 {
								p := g.GetPlayer(controller)
								if p != nil {
									g.DealDamageToPlayer(p, pins, sourceID)
								}
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					// Only at your end step, if untapped
					if evt.PlayerID != controllerID {
						return false
					}
					src := g.FindPermanent(sourceID)
					return src != nil && !src.Tapped
				}),
			),
			// {X}{X}, {T}: deal damage equal to pin counters to any target
			WithActivatedAbility(
				FuncEffect("deal damage equal to pin counters to any target",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						pins := src.Counters[Pin]
						if pins <= 0 {
							return nil
						}
						// Deal damage to target
						for _, pl := range g.AllPlayers() {
							if pl.PlayerID() == targets[0] {
								g.DealDamageToPlayer(pl, pins, sourceID)
								return nil
							}
						}
						perm := g.FindPermanent(targets[0])
						if perm != nil {
							g.DealDamageToPermanent(perm, pins, sourceID)
						}
						return nil
					},
				),
				TapSourceCost(),
				WithTarget(TargetAnyTarget()),
			),
		)
	}))


// White Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {W}, then add an additional {W} for each charge counter removed this way.
	Register("White Mana Battery", withExpansion(func() Card {
		return manaBattery("White Mana Battery", White)
	}))

}
