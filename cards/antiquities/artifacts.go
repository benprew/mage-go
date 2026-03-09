package antiquities

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// Amulet of Kroog {2}
	// Artifact
	// {2}, {T}: Prevent the next 1 damage that would be dealt to any target this turn.
	Register("Amulet of Kroog", func() Card {
		return NewArtifact("Amulet of Kroog", "{2}",
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(1)),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

	// Armageddon Clock {6}
	// Artifact
	// At the beginning of your upkeep, put a doom counter on Armageddon Clock.
	// At the beginning of your draw step, Armageddon Clock deals damage equal to the number of
	// doom counters on it to each player.
	// {4}: Remove a doom counter from Armageddon Clock. Any player may activate this ability but
	// only during any upkeep step.
	Register("Armageddon Clock", func() Card {
		return NewArtifact("Armageddon Clock", "{6}",
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("put a doom counter on Armageddon Clock",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							perm.AddCounter(Doom, 1)
						}
						return nil
					}), false,
			)),
			WithAbility(
				NewTriggered(EvtDrawStep, false,
					FuncEffect("deal damage equal to doom counters to each player",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.FindPermanent(sourceID)
							if perm == nil {
								return nil
							}
							counters := perm.Counters[Doom]
							if counters <= 0 {
								return nil
							}
							for _, p := range g.AllPlayers() {
								g.DealDamageToPlayer(p, counters, sourceID)
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, _ *Game, _, controllerID uuid.UUID) bool {
					return evt.PlayerID == controllerID
				}),
			),
			// {4}: Remove a doom counter. Any player may activate this but only during upkeep.
			WithActivatedAbility(
				FuncEffect("remove a doom counter from Armageddon Clock",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm != nil {
							perm.RemoveCounter(Doom, 1)
						}
						return nil
					}),
				GenericCost(4),
				WithUpkeepOnly(),
				WithAnyPlayerMay(),
			),
		)
	})

	// Ashnod's Altar {3}
	// Artifact
	// Sacrifice a creature: Add {C}{C}.
	Register("Ashnod's Altar", func() Card {
		return NewArtifact("Ashnod's Altar", "{3}",
			WithActivatedAbility(
				AddMana(Colorless, 2),
				SacrificeCreatureCost(),
			),
		)
	})

	// Ashnod's Battle Gear {2}
	// Artifact
	// You may choose not to untap Ashnod's Battle Gear during your untap step.
	// {2}, {T}: Target creature you control gets +2/-2 for as long as Ashnod's Battle Gear
	// remains tapped.
	Register("Ashnod's Battle Gear", func() Card {
		return NewArtifact("Ashnod's Battle Gear", "{2}",
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				FuncEffect("target creature gets +2/-2 while ~ remains tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						// Create a continuous effect that boosts while source is tapped
						eff := FuncContinuousEffect(LayerPT, WhileOnBattlefield,
							func(g *Game, srcID uuid.UUID) error {
								src := g.FindPermanent(srcID)
								if src == nil {
									return nil
								}
								target := g.FindPermanent(targetID)
								if target == nil {
									return nil
								}
								target.BoostPT(2, -2)
								return nil
							}, SourceTapped)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithTarget(TargetControlledCreature()),
			),
		)
	})

	// Ashnod's Transmogrant {1}
	// Artifact
	// {T}, Sacrifice Ashnod's Transmogrant: Put a +1/+1 counter on target nonartifact creature.
	// That creature becomes an artifact in addition to its other types.
	Register("Ashnod's Transmogrant", func() Card {
		return NewArtifact("Ashnod's Transmogrant", "{1}",
			WithActivatedAbility(
				FuncEffect("put +1/+1 counter and make artifact",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						perm.AddCounter(P1P1, 1)
						perm.Card.AddType(TypeArtifact)
						return nil
					}),
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCreature(Not(IsArtifact))),
			),
		)
	})

	// Candelabra of Tawnos {1}
	// Artifact
	// {X}, {T}: Untap X target lands.
	// XXX: X-targeting for untap lands
	Register("Candelabra of Tawnos", func() Card {
		return NewArtifact("Candelabra of Tawnos", "{1}")
	})

	// Coral Helm {3}
	// Artifact
	// {3}, Discard a card at random: Target creature gets +2/+2 until end of turn.
	Register("Coral Helm", func() Card {
		return NewArtifact("Coral Helm", "{3}",
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(2), Fixed(2), SelectTarget),
				GenericCost(3),
				WithCost(DiscardRandomCost(1)),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Cursed Rack {4}
	// Artifact
	// As Cursed Rack enters the battlefield, choose an opponent.
	// The chosen player's maximum hand size is four.
	// XXX: no ETB opponent choice; auto-picks opponent (correct for 2-player, gap for multiplayer)
	Register("Cursed Rack", func() Card {
		return NewArtifact("Cursed Rack", "{4}",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
					func(g *Game, sourceID uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						opponent := g.GetOpponent(src.Controller)
						if opponent != nil {
							g.Effects.Rules.SetMaxHandSize(opponent.PlayerID(), 4)
						}
						return nil
					}),
			),
		)
	})

	// Feldon's Cane {1}
	// Artifact
	// {T}, Exile Feldon's Cane: Shuffle your graveyard into your library.
	Register("Feldon's Cane", func() Card {
		return NewArtifact("Feldon's Cane", "{1}",
			WithActivatedAbility(
				FuncEffect("shuffle graveyard into library",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						gy := p.Graveyard()
						lib := p.Library()
						for _, card := range gy {
							lib = append(lib, card)
						}
						p.SetLibrary(lib)
						p.ClearGraveyard()
						p.ShuffleLibrary()
						return nil
					}),
				TapSourceCost(),
				WithCost(ExileSourceCost()),
			),
		)
	})

	// Golgothian Sylex {4}
	// Artifact
	// {1}, {T}: Each nontoken permanent with a name originally printed in the Antiquities
	// expansion is sacrificed by its controller.
	Register("Golgothian Sylex", func() Card {
		return NewArtifact("Golgothian Sylex", "{4}",
			WithActivatedAbility(
				FuncEffect("sacrifice all nontoken Antiquities permanents",
					EffectProperties{Outcome: OutcomeDetriment, Mass: true},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						var toSacrifice []*Permanent
						for _, perm := range g.FilterBattlefield(And(Not(IsToken))) {
							if antiquitiesCards[perm.Name()] {
								toSacrifice = append(toSacrifice, perm)
							}
						}
						for _, perm := range toSacrifice {
							g.Sacrifice(perm)
						}
						return nil
					}),
				GenericCost(1),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Ivory Tower {1}
	// Artifact
	// At the beginning of your upkeep, you gain X life, where X is the number of cards in
	// your hand minus 4.
	Register("Ivory Tower", func() Card {
		return NewArtifact("Ivory Tower", "{1}",
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("gain life equal to hand size minus 4",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						excess := len(p.Hand()) - 4
						if excess > 0 {
							g.PlayerGainLife(p, excess)
						}
						return nil
					}), false,
			)),
		)
	})

	// Jalum Tome {3}
	// Artifact
	// {2}, {T}: Draw a card, then discard a card.
	Register("Jalum Tome", func() Card {
		return NewArtifact("Jalum Tome", "{3}",
			WithActivatedAbility(
				FuncEffect("draw a card, then discard a card",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						p.DrawCard()
						chosen := p.ChooseCardsFromHand(1, "discard", g)
						for _, card := range chosen {
							p.RemoveFromHand(card.ID())
							p.AddToGraveyard(card)
						}
						return nil
					}),
				GenericCost(2),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Mightstone {4}
	// Artifact
	// Attacking creatures get +1/+0.
	Register("Mightstone", func() Card {
		return NewArtifact("Mightstone", "{4}",
			WithStaticAbility(
				BoostAllCreaturesIncludingSelf(1, 0, IsAttacking),
			),
		)
	})

	// Millstone {2}
	// Artifact
	// {2}, {T}: Target player mills two cards.
	Register("Millstone", func() Card {
		return NewArtifact("Millstone", "{2}",
			WithActivatedAbility(
				FuncEffect("target player mills two cards",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(targets[0])
						if p == nil {
							return nil
						}
						lib := p.Library()
						for i := 0; i < 2 && len(lib) > 0; i++ {
							card := lib[len(lib)-1]
							lib = lib[:len(lib)-1]
							p.AddToGraveyard(card)
						}
						p.SetLibrary(lib)
						return nil
					}),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithTarget(TargetPlayer()),
			),
		)
	})

	// Obelisk of Undoing {1}
	// Artifact
	// {6}, {T}: Return target permanent you both own and control to your hand.
	// Target checks both "own AND control" per Oracle via ControlledPermanentTarget
	Register("Obelisk of Undoing", func() Card {
		return NewArtifact("Obelisk of Undoing", "{1}",
			WithActivatedAbility(
				ReturnToHandTarget(),
				GenericCost(6),
				WithCost(TapSourceCost()),
				WithTarget(TargetControlledPermanent()),
			),
		)
	})

	// Rakalite {6}
	// Artifact
	// {2}: Prevent the next 1 damage that would be dealt to any target this turn. Return
	// Rakalite to its owner's hand at the beginning of the next end step.
	Register("Rakalite", func() Card {
		return NewArtifact("Rakalite", "{6}",
			WithActivatedAbility(
				FuncEffect("prevent 1 damage to target; bounce self at next end step",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						// Prevent 1 damage to target
						targetPerm := g.FindPermanent(targets[0])
						if targetPerm != nil {
							g.AddPreventionShield(targetPerm.ID(), 1)
						} else {
							// Target is a player
							targetPlayer := g.GetPlayer(targets[0])
							if targetPlayer != nil {
								g.AddPreventionShield(targets[0], 1)
							}
						}
						// Register delayed trigger: bounce self at next end step
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:  EvtEndStep,
							TargetID:   sourceID,
							Effects:    []Effect{ReturnToHandTarget()},
							SourceID:   sourceID,
							Controller: controller,
						})
						return nil
					}),
				GenericCost(2),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

	// Rocket Launcher {4}
	// Artifact
	// {2}: Rocket Launcher deals 1 damage to any target. Destroy Rocket Launcher at the beginning
	// of the next end step. Activate only if you've controlled Rocket Launcher continuously since
	// the beginning of your most recent turn.
	Register("Rocket Launcher", func() Card {
		return NewArtifact("Rocket Launcher", "{4}",
			WithActivatedAbility(
				FuncEffect("deal 1 damage to any target; destroy self at next end step",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(1)},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						// Deal 1 damage
						targetPerm := g.FindPermanent(targets[0])
						if targetPerm != nil {
							g.DealDamageToPermanent(targetPerm, 1, sourceID)
						} else {
							targetPlayer := g.GetPlayer(targets[0])
							if targetPlayer != nil {
								g.DealDamageToPlayer(targetPlayer, 1, sourceID)
							}
						}
						// Register delayed trigger: destroy self at next end step
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:  EvtEndStep,
							TargetID:   sourceID,
							Effects:    []Effect{DestroyTarget()},
							SourceID:   sourceID,
							Controller: controller,
						})
						return nil
					}),
				GenericCost(2),
				WithTarget(TargetAnyTarget()),
				WithControlledSinceTurnStart(),
			),
		)
	})

	// Staff of Zegon {4}
	// Artifact
	// {3}, {T}: Target creature gets -2/-0 until end of turn.
	Register("Staff of Zegon", func() Card {
		return NewArtifact("Staff of Zegon", "{4}",
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(-2), Fixed(0), SelectTarget),
				GenericCost(3),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Tablet of Epityr {1}
	// Artifact
	// Whenever an artifact you control is put into a graveyard from the battlefield, you may
	// pay {1}. If you do, you gain 1 life.
	Register("Tablet of Epityr", func() Card {
		return NewArtifact("Tablet of Epityr", "{1}",
			WithAbility(
				NewTriggered(EvtPutIntoGraveyardFromBattlefield, true,
					FuncEffect("you may pay {1}; if you do, gain 1 life",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							if !g.TryPayCostFromLands(controller, "{1}") {
								return nil
							}
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerGainLife(p, 1)
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					if evt.SourceID == sourceID {
						return false // not itself
					}
					if evt.PlayerID != controllerID {
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

	// Tawnos's Coffin {4}
	// Artifact
	// You may choose not to untap Tawnos's Coffin during your untap step.
	// {3}, {T}: Exile target creature and all Auras attached to it. Note the number and kind of
	// counters that were on that creature. When Tawnos's Coffin leaves the battlefield or becomes
	// untapped, return that exiled card to the battlefield under its owner's control tapped with the
	// noted number and kind of counters on it.
	// XXX: does not exile/return Auras attached to the creature
	Register("Tawnos's Coffin", func() Card {
		// coffinReturnExiled is a helper closure that returns all exiled cards from a Coffin.
		coffinReturnExiled := func(g GameMutator, coffinID uuid.UUID) {
			// Use the concrete *Game to access RemoveExiledCardBySource
			game, ok := g.(*Game)
			if !ok {
				return
			}
			exiled := game.RemoveExiledCardBySource(coffinID)
			for _, ec := range exiled {
				owner := ec.Owner
				if owner == (uuid.UUID{}) {
					owner = ec.Card.Owner()
				}
				perm := g.PutOnBattlefield(ec.Card, owner)
				if perm != nil {
					perm.Tapped = true
					// Restore noted counters
					for ct, count := range ec.Counters {
						perm.AddCounter(ct, count)
					}
				}
			}
		}

		return NewArtifact("Tawnos's Coffin", "{4}",
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				FuncEffect("exile target creature; return when Coffin leaves or untaps",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := g.FindPermanent(targets[0])
						if target == nil {
							return nil
						}
						// Note counters before exile
						counters := make(map[CounterType]int)
						for ct, count := range target.Counters {
							counters[ct] = count
						}
						owner := target.Controller
						card := target.Card
						g.RemoveFromBattlefield(target)
						// Add to exile with counter metadata
						game, ok := g.(*Game)
						if ok {
							game.Exile = append(game.Exile, ExiledCard{
								Card:     card,
								ExiledBy: sourceID,
								Counters: counters,
								Owner:    owner,
							})
						}
						return nil
					}),
				GenericCost(3),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
			// When Tawnos's Coffin leaves the battlefield, return exiled creature
			WithAbility(
				NewTriggered(EvtLeavesBattlefield, false,
					FuncEffect("return exiled creature to battlefield",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							coffinReturnExiled(g, sourceID)
							return nil
						}),
				).SetCondition(IsThisSource),
			),
			// At the beginning of your upkeep, if Coffin is untapped and has exiled cards, return them
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("return exiled creature if Coffin is untapped",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil || perm.Tapped {
							return nil
						}
						coffinReturnExiled(g, sourceID)
						return nil
					}), false,
			)),
		)
	})

	// Tawnos's Wand {4}
	// Artifact
	// {2}, {T}: Target creature with power 2 or less can't be blocked this turn.
	Register("Tawnos's Wand", func() Card {
		return NewArtifact("Tawnos's Wand", "{4}",
			WithActivatedAbility(
				GrantKeywordUntilEndOfTurn(UnblockableKW, SelectTarget),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature(HasPowerLTE(2))),
			),
		)
	})

	// Tawnos's Weaponry {2}
	// Artifact
	// You may choose not to untap Tawnos's Weaponry during your untap step.
	// {2}, {T}: Target creature gets +1/+1 for as long as Tawnos's Weaponry remains tapped.
	Register("Tawnos's Weaponry", func() Card {
		return NewArtifact("Tawnos's Weaponry", "{2}",
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				FuncEffect("target creature gets +1/+1 while ~ remains tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						eff := FuncContinuousEffect(LayerPT, WhileOnBattlefield,
							func(g *Game, srcID uuid.UUID) error {
								src := g.FindPermanent(srcID)
								if src == nil {
									return nil
								}
								target := g.FindPermanent(targetID)
								if target == nil {
									return nil
								}
								target.BoostPT(1, 1)
								return nil
							}, SourceTapped)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// The Rack {1}
	// Artifact
	// As The Rack enters the battlefield, choose an opponent.
	// At the beginning of the chosen player's upkeep, The Rack deals X damage to that player,
	// where X is 3 minus the number of cards in their hand.
	// XXX: no ETB opponent choice; auto-picks opponent (correct for 2-player, gap for multiplayer)
	Register("The Rack", func() Card {
		return NewArtifact("The Rack", "{1}",
			WithAbility(
				NewTriggered(EvtUpkeep, false,
					FuncEffect("deal 3 minus hand size damage",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							opponent := g.GetOpponent(controller)
							if opponent == nil {
								return nil
							}
							damage := 3 - len(opponent.Hand())
							if damage > 0 {
								g.DealDamageToPlayer(opponent, damage, sourceID)
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g *Game, _, controllerID uuid.UUID) bool {
					opponent := g.GetOpponent(controllerID)
					return opponent != nil && evt.PlayerID == opponent.PlayerID()
				}),
			),
		)
	})

	// Urza's Chalice {1}
	// Artifact
	// Whenever a player casts an artifact spell, you may pay {1}. If you do, you gain 1 life.
	Register("Urza's Chalice", func() Card {
		return NewArtifact("Urza's Chalice", "{1}",
			WithAbility(
				NewTriggered(EvtSpellCast, true,
					FuncEffect("you may pay {1}; if you do, gain 1 life",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							if !g.TryPayCostFromLands(controller, "{1}") {
								return nil
							}
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerGainLife(p, 1)
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g *Game, _, _ uuid.UUID) bool {
					card := g.FindCardAnywhere(evt.SourceID)
					if card == nil {
						return false
					}
					return card.HasType(TypeArtifact)
				}),
			),
		)
	})

	// Urza's Miter {3}
	// Artifact
	// Whenever an artifact you control is put into a graveyard from the battlefield, if it wasn't
	// sacrificed, you may pay {3}. If you do, draw a card.
	Register("Urza's Miter", func() Card {
		return NewArtifact("Urza's Miter", "{3}",
			WithAbility(
				NewTriggered(EvtPutIntoGraveyardFromBattlefield, true,
					FuncEffect("pay {3} to draw a card",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							cost := ParseManaCost("{3}")
							if p.ManaPool().CanPay(cost) {
								_ = p.ManaPool().Pay(cost)
								p.DrawCard()
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g *Game, _, controllerID uuid.UUID) bool {
					// Only trigger for non-sacrifice (Flag=false) artifact deaths you control
					if evt.Flag {
						return false // was sacrificed
					}
					card := g.FindCardAnywhere(evt.SourceID)
					if card == nil {
						return false
					}
					return card.HasType(TypeArtifact) && card.Owner() == controllerID
				}),
			),
		)
	})

	// Weakstone {4}
	// Artifact
	// Attacking creatures get -1/-0.
	Register("Weakstone", func() Card {
		return NewArtifact("Weakstone", "{4}",
			WithStaticAbility(
				BoostAllCreaturesIncludingSelf(-1, 0, IsAttacking),
			),
		)
	})
}
