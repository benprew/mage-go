package legends

import (
	"fmt"

	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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
					counters := int(p.Counters[Charge])
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
	Register("Al-abara's Carpet", func() Card {
		return NewArtifact("Al-abara's Carpet", "{5}",
			WithActivatedAbility(
				FuncEffect(
					"prevent all damage that would be dealt to you this turn by attacking creatures without flying",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, srcID uuid.UUID) error {
							g.AddDamagePreventionRule(
								WithFrom(And(IsAttacking, Not(HasKeywordFilter(Flying)))),
								WithPlayerOnly(),
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
	})


// Alchor's Tomb {4}
// Artifact
// {2}, {T}: Target permanent you control becomes the color of your choice. (This effect lasts indefinitely.)
	Register("Alchor's Tomb", func() Card {
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
	})


// Arena of the Ancients {3}
// Artifact
// Legendary creatures don't untap during their controllers' untap steps.
// When this artifact enters, tap all legendary creatures.
	Register("Arena of the Ancients", func() Card {
		return NewArtifact("Arena of the Ancients", "{3}",
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) && p.Card.HasSuperType(SuperLegendary) {
						g.GrantAttr(p.ID(), AttrDoesNotUntap)
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
						g.TapPermanent(p)
					}
					return nil
				},
			), false)),
		)
	})


// Black Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {B}, then add an additional {B} for each charge counter removed this way.
	Register("Black Mana Battery", func() Card {
		return manaBattery("Black Mana Battery", Black)
	})


// Blue Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {U}, then add an additional {U} for each charge counter removed this way.
	Register("Blue Mana Battery", func() Card {
		return manaBattery("Blue Mana Battery", Blue)
	})


// Forethought Amulet {5}
// Artifact
// At the beginning of your upkeep, sacrifice this artifact unless you pay {3}.
// If an instant or sorcery source would deal 3 or more damage to you, it deals 2 damage to you instead.
// XXX: damage cap from instants/sorceries not yet implemented — needs damage replacement engine feature
	Register("Forethought Amulet", func() Card {
		return NewArtifact("Forethought Amulet", "{5}",
			WithAbility(SacrificeAtUpkeepUnlessPay("{3}")),
		)
	})


// Gauntlets of Chaos {5}
// Artifact
// {5}, Sacrifice this artifact: Exchange control of target artifact, creature, or land you control and target permanent an opponent controls that shares one of those types with it. If those permanents are exchanged this way, destroy all Auras attached to them.
// TODO: implement
	Register("Gauntlets of Chaos", func() Card {
		return NewArtifact("Gauntlets of Chaos", "{5}")
	})


// Green Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {G}, then add an additional {G} for each charge counter removed this way.
	Register("Green Mana Battery", func() Card {
		return manaBattery("Green Mana Battery", Green)
	})


// Horn of Deafening {4}
// Artifact
// {2}, {T}: Prevent all combat damage that would be dealt by target creature this turn.
	Register("Horn of Deafening", func() Card {
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
				ManaCostOf("{2}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})


// Knowledge Vault {4}
// Artifact
// {2}, {T}: Exile the top card of your library face down.
// {0}: Sacrifice this artifact. If you do, discard your hand, then put all cards exiled with this artifact into their owner's hand.
// When this artifact leaves the battlefield, put all cards exiled with it into their owner's graveyard.
// TODO: implement
	Register("Knowledge Vault", func() Card {
		return NewArtifact("Knowledge Vault", "{4}")
	})


// Kry Shield {2}
// Artifact
// {2}, {T}: Prevent all damage that would be dealt this turn by target creature you control. That creature gets +0/+X until end of turn, where X is its mana value.
	Register("Kry Shield", func() Card {
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
							g.AddDamagePreventionRule(WithFrom(NewPermanentFilter("prevented source", func(p *Permanent, _ *Game) bool {
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
				WithTarget(TargetCreatureYouControl()),
			),
		)
	})


// Life Chisel {4}
// Artifact
// Sacrifice a creature: You gain life equal to the sacrificed creature's toughness. Activate only during your upkeep.
	Register("Life Chisel", func() Card {
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
	})


// Life Matrix {4}
// Artifact
// {4}, {T}: Put a matrix counter on target creature and that creature gains "Remove a matrix counter from this creature: Regenerate this creature." Activate only during your upkeep.
	Register("Life Matrix", func() Card {
		return NewArtifact("Life Matrix", "{4}",
			WithActivatedAbility(
				AddCounters(Matrix, Fixed(1), SelectTarget),
				ManaCostOf("{4}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
				WithUpkeepOnly(),
			),
			// Grant "Remove a matrix counter: Regenerate" to all creatures with matrix counters
			WithStaticAbility(GrantActivatedAbilityToAll(
				RegenerateSource(),
				RemoveCountersCost(Matrix, 1),
				NewPermanentFilter("creature with matrix counter", func(p *Permanent, _ *Game) bool {
					return p.Counters[Matrix] > 0
				}),
			)),
		)
	})


// Mana Matrix {6}
// Artifact
// Instant and enchantment spells you cast cost {2} less to cast.
	Register("Mana Matrix", func() Card {
		return NewArtifact("Mana Matrix", "{6}",
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.AddSpellTypeCostReduction(TypeInstant, 2)
				g.AddSpellTypeCostReduction(TypeEnchantment, 2)
				return nil
			})),
		)
	})


// Mirror Universe {6}
// Artifact
// {T}, Sacrifice this artifact: Exchange life totals with target opponent. Activate only during your upkeep.
	Register("Mirror Universe", func() Card {
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
	})


// North Star {4}
// Artifact
// {4}, {T}: For one spell this turn, you may spend mana as though it were mana of any type to pay that spell's mana cost. (Additional costs are still paid normally.)
// TODO: implement
	Register("North Star", func() Card {
		return NewArtifact("North Star", "{4}")
	})


// Nova Pentacle {4}
// Artifact
// {3}, {T}: The next time a source of your choice would deal damage to you this turn, that damage is dealt to target creature of an opponent's choice instead.
// TODO: implement
	Register("Nova Pentacle", func() Card {
		return NewArtifact("Nova Pentacle", "{4}")
	})


// Planar Gate {6}
// Artifact
// Creature spells you cast cost {2} less to cast.
	Register("Planar Gate", func() Card {
		return NewArtifact("Planar Gate", "{6}",
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.AddSpellTypeCostReduction(TypeCreature, 2)
				return nil
			})),
		)
	})


// Red Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {R}, then add an additional {R} for each charge counter removed this way.
	Register("Red Mana Battery", func() Card {
		return manaBattery("Red Mana Battery", Red)
	})


// Relic Barrier {2}
// Artifact
// {T}: Tap target artifact.
	Register("Relic Barrier", func() Card {
		return NewArtifact("Relic Barrier", "{2}",
			WithActivatedAbility(
				TapTarget(),
				TapSourceCost(),
				WithTarget(TargetArtifact()),
			),
		)
	})


// Ring of Immortals {5}
// Artifact
// {3}, {T}: Counter target instant or Aura spell that targets a permanent you control.
// TODO: implement
	Register("Ring of Immortals", func() Card {
		return NewArtifact("Ring of Immortals", "{5}")
	})


// Serpent Generator {6}
// Artifact
// {4}, {T}: Create a 1/1 colorless Snake artifact creature token. It has "Whenever this creature deals damage to a player, that player gets a poison counter." (A player with ten or more poison counters loses the game.)
	Register("Serpent Generator", func() Card {
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
							func(g GameMutator, srcID, ctrl uuid.UUID, targets []uuid.UUID) error {
								// targets[0] = damaged player (passed by trigger system from EvtDamageDealt.TargetID)
								if len(targets) == 0 {
									return nil
								}
								p := g.GetPlayer(targets[0])
								if p == nil {
									return nil
								}
								p.AddPoisonCounters(1)
								return nil
							},
						)).SetCondition(func(evt *GameEvent, g GameReader, srcID, _ uuid.UUID) bool {
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
	})


// Sword of the Ages {6}
// Artifact
// This artifact enters tapped.
// {T}, Sacrifice this artifact and any number of creatures you control: This artifact deals X damage to any target, where X is the total power of the creatures sacrificed this way, then exile this artifact and those creature cards.
	Register("Sword of the Ages", func() Card {
		return NewArtifact("Sword of the Ages", "{6}",
			WithKeyword(EntersTapped),
			WithActivatedAbility(
				FuncEffect("{T}, Sacrifice: deal X damage where X is total power of sacrificed creatures",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Sacrifice source (Sword of the Ages)
						src := g.FindPermanent(sourceID)
						var srcCard Card
						if src != nil {
							srcCard = src.Card
							g.Sacrifice(src)
						}
						// Let player choose any number of creatures to sacrifice
						var totalPower int
						var sacrificedCards []Card
						for {
							creatures := g.FilterBattlefield(And(IsCreature, ControlledBy(controller)))
							if len(creatures) == 0 {
								break
							}
							mode := p.ChooseMode([]string{"Sacrifice a creature", "Done"}, "Sword of the Ages")
							if mode != 0 {
								break // player chose "Done"
							}
							chosen := p.ChoosePermanent(creatures, "Choose creature to sacrifice", g)
							if chosen == nil {
								break
							}
							totalPower += chosen.CurrentPower(g)
							sacrificedCards = append(sacrificedCards, chosen.Card)
							g.Sacrifice(chosen)
						}
						// Deal damage
						if totalPower > 0 {
							targetP := g.GetPlayer(targets[0])
							if targetP != nil {
								g.DealDamageToPlayer(targetP, totalPower, sourceID)
							} else {
								targetPerm := g.FindPermanent(targets[0])
								if targetPerm != nil {
									g.DealDamageToPermanent(targetPerm, totalPower, sourceID)
								}
							}
						}
						// Exile the artifact and creature cards
						if srcCard != nil {
							g.ExileCard(srcCard, controller)
						}
						for _, c := range sacrificedCards {
							g.ExileCard(c, controller)
						}
						return nil
					}),
				TapSourceCost(),
				WithTarget(TargetAnyTarget()),
			),
		)
	})


// Triassic Egg {4}
// Artifact
// {3}, {T}: Put a hatchling counter on this artifact.
// Sacrifice this artifact: Choose one. Activate only if there are two or more hatchling counters on this artifact.
// • You may put a creature card from your hand onto the battlefield.
// • Return target creature card from your graveyard to the battlefield.
	Register("Triassic Egg", func() Card {
		return NewArtifact("Triassic Egg", "{4}",
			// {3}, {T}: Put a hatchling counter
			WithActivatedAbility(
				AddCounters(Hatchling, Fixed(1), SelectSource),
				ManaCostOf("{3}"),
				WithCost(TapSourceCost()),
			),
			// Sacrifice: put creature from hand onto battlefield or reanimate from graveyard
			// Activate only if there are two or more hatchling counters
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
							// Put creature from hand onto battlefield (player chooses)
							var candidates []Card
							for _, card := range p.Hand() {
								if card.HasType(TypeCreature) {
									candidates = append(candidates, card)
								}
							}
							if len(candidates) > 0 {
								chosen := p.ChooseCardFromLibrary(candidates, "Triassic Egg: choose a creature to put onto the battlefield", g)
								if chosen != nil {
									p.RemoveFromHand(chosen.ID())
									g.PutOnBattlefield(chosen, controller)
								}
							}
						} else {
							// Return creature from graveyard to battlefield (player chooses)
							var candidates []Card
							for _, card := range p.Graveyard() {
								if card.HasType(TypeCreature) {
									candidates = append(candidates, card)
								}
							}
							if len(candidates) > 0 {
								chosen := p.ChooseCardFromLibrary(candidates, "Triassic Egg: choose a creature to return from graveyard", g)
								if chosen != nil {
									p.RemoveFromGraveyard(chosen.ID())
									g.PutOnBattlefield(chosen, controller)
								}
							}
						}
						return nil
					},
				),
				SacrificeSourceCost(),
				WithCost(RequireCountersCost(Hatchling, 2)),
			),
		)
	})


// Voodoo Doll {6}
// Artifact
// At the beginning of your upkeep, put a pin counter on this artifact.
// At the beginning of your end step, if this artifact is untapped, destroy this artifact and it deals damage to you equal to the number of pin counters on it.
// {X}{X}, {T}: This artifact deals damage equal to the number of pin counters on it to any target. X is the number of pin counters on this artifact.
	Register("Voodoo Doll", func() Card {
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
							pins := int(src.Counters[Pin])
							g.DestroyPermanent(src)
							if pins > 0 {
								p := g.GetPlayer(controller)
								if p != nil {
									g.DealDamageToPlayer(p, pins, sourceID)
								}
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
					// Only at your end step, if untapped
					if evt.PlayerID != controllerID {
						return false
					}
					src := g.FindPermanent(sourceID)
					return src != nil && !src.Tapped
				}),
			),
			// {X}{X}, {T}: deal damage equal to pin counters to any target
			// XXX: {X}{X} mana cost (where X = pin counters) is paid inside the effect at resolution
			// rather than as an activation cost; the engine lacks dynamic mana costs tied to counter counts
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
						pins := int(src.Counters[Pin])
						if pins <= 0 {
							return nil
						}
						// Pay {X}{X} where X = pin counters (2*pins generic mana)
						manaCost := fmt.Sprintf("{%d}", 2*pins)
						if !g.TryPayCostFromLands(controller, manaCost) {
							return nil // cannot pay
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
	})


// White Mana Battery {4}
// Artifact
// {2}, {T}: Put a charge counter on this artifact.
// {T}, Remove any number of charge counters from this artifact: Add {W}, then add an additional {W} for each charge counter removed this way.
	Register("White Mana Battery", func() Card {
		return manaBattery("White Mana Battery", White)
	})

}
