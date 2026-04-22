package antiquities

import (
	"git.sr.ht/~cdcarter/mage-go/pkg/catalog"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
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
				AddCounters(Doom, Fixed(1), SelectSource), false,
			)),
			WithAbility(
				NewTriggered(EvtDrawStep, false,
					Pipeline("deal damage equal to doom counters to each player",
						EffectProperties{Outcome: OutcomeDetriment},
						SnapshotSourceCounter(Doom, "doom"),
						DealDamageToPlayersFromVar("doom", SelectEachPlayer()),
					),
				).SetConditionData(EventPlayerIsController{}),
			),
			// {4}: Remove a doom counter. Any player may activate this but only during upkeep.
			WithActivatedAbility(
				RemoveCountersFromSource(Doom, 1),
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
				// TODO: convert to pipeline — needs "add continuous effect while tapped" step
				FuncEffect("target creature gets +2/-2 while ~ remains tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
				// TODO: convert to pipeline — needs "add type to permanent" step
				FuncEffect("put +1/+1 counter and make artifact",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
	// XXX: Oracle says "untap X target lands" (targets declared on activation); engine uses
	// resolution-time choices instead.
	Register("Candelabra of Tawnos", func() Card {
		return NewArtifact("Candelabra of Tawnos", "{1}",
			WithActivatedAbility(
				// TODO: convert to pipeline — needs "repeat X times: choose and untap" step
				FuncEffect("untap X target lands",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						x := g.XValue()
						if x <= 0 {
							return nil
						}
						player := g.GetPlayer(controller)
						if player == nil {
							return nil
						}
						for i := 0; i < x; i++ {
							candidates := g.FilterBattlefield(And(IsLand, IsTapped))
							if len(candidates) == 0 {
								break
							}
							chosen := player.ChoosePermanent(candidates, "untap land", g)
							if chosen == nil {
								break
							}
							chosen.Tapped = false
						}
						return nil
					}),
				XManaCost(),
				WithCost(TapSourceCost()),
			),
		)
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
							g.SetMaxHandSize(opponent.PlayerID(), 4)
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
				DataEffect(ShuffleGraveyardIntoLibrary()),
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
				// TODO: convert to pipeline — needs catalog-based filter as PermanentFilter
				FuncEffect("sacrifice all nontoken Antiquities permanents",
					EffectProperties{Outcome: OutcomeDetriment, Mass: true},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						var toSacrifice []*Permanent
						for _, perm := range g.FilterBattlefield(And(Not(IsToken))) {
							if catalog.Global().CardInSet("ATQ", perm.Name()) {
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
				Pipeline("gain life equal to hand size minus 4",
					EffectProperties{},
					SnapshotPermanent(SelectSource, "src"),
					SetVarFromHandSize(SelectController(), "excess", 4),
					GainLifeFromVar("src.controller", "excess"),
				), false,
			)),
		)
	})

	// Jalum Tome {3}
	// Artifact
	// {2}, {T}: Draw a card, then discard a card.
	Register("Jalum Tome", func() Card {
		return NewArtifact("Jalum Tome", "{3}",
			WithActivatedAbility(
				// TODO: convert to pipeline — needs "draw then choose discard" step
				FuncEffect("draw a card, then discard a card",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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
				// TODO: convert to pipeline — needs "mill target player" step
				FuncEffect("target player mills two cards",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
				Pipeline("prevent 1 damage to target; bounce self at next end step",
					EffectProperties{Outcome: OutcomeBenefit},
					UnwrapEffect(PreventDamageToTarget(Fixed(1))),
					RegisterDelayedTriggerStep(EvtEndStep, "", ReturnToHandTarget()),
				),
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
				Pipeline("deal 1 damage to any target; destroy self at next end step",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(1)},
					UnwrapEffect(DealDamage(Fixed(1))),
					RegisterDelayedTriggerStep(EvtEndStep, "", DestroyTarget()),
				),
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
					Pipeline("you may pay {1}; if you do, gain 1 life",
						EffectProperties{Outcome: OutcomeBenefit},
						IfElse("pay {1} to gain 1 life",
							&TryPayManaCond{Cost: "{1}"},
							UnwrapEffect(GainLife(1)),
							nil),
					),
				).
					// TODO: convert to data condition
					SetCondition(func(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
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
	Register("Tawnos's Coffin", func() Card {
		// Noted state tracked per-card-ID, shared between the activate and return closures.
		type notedState struct {
			counters [NumCounters]uint8
			owner    uuid.UUID
		}
		noted := make(map[uuid.UUID]notedState)

		coffinReturnExiled := func(g *Game, coffinID uuid.UUID) {
			exiled := g.RemoveExiledCardBySource(coffinID)
			var creatureEC *ExiledCard
			var auraECs []ExiledCard
			for i := range exiled {
				if exiled[i].Card.HasType(TypeCreature) {
					creatureEC = &exiled[i]
				} else if exiled[i].Card.HasSubType("Aura") {
					auraECs = append(auraECs, exiled[i])
				}
			}
			var creaturePerm *Permanent
			if creatureEC != nil {
				ns := noted[creatureEC.Card.ID()]
				owner := ns.owner
				if owner == (uuid.UUID{}) {
					owner = creatureEC.Card.Owner()
				}
				creaturePerm = g.PutOnBattlefield(creatureEC.Card, owner)
				if creaturePerm != nil {
					// comes into play tapped, not comes into play then taps
					creaturePerm.Tapped = true
					for ct := CounterType(0); ct < NumCounters; ct++ {
						if count := ns.counters[ct]; count != 0 {
							creaturePerm.AddCounter(ct, int(count))
						}
					}
				}
				delete(noted, creatureEC.Card.ID())
			}
			for _, auraEC := range auraECs {
				ns := noted[auraEC.Card.ID()]
				owner := ns.owner
				if owner == (uuid.UUID{}) {
					owner = auraEC.Card.Owner()
				}
				auraPerm := g.PutOnBattlefield(auraEC.Card, owner)
				if auraPerm != nil && creaturePerm != nil {
					g.Attach(auraPerm.ID(), creaturePerm.ID())
				}
				delete(noted, auraEC.Card.ID())
			}
		}

		return NewArtifact("Tawnos's Coffin", "{4}",
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				// TODO: convert to pipeline — needs exile-with-noted-state tracking
				FuncEffect("exile target creature and all Auras; return when Coffin leaves or untaps",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := g.FindPermanent(targets[0])
						if target == nil {
							return nil
						}
						noted[target.Card.ID()] = notedState{
							counters: target.Counters,
							owner:    target.Controller,
						}
						card := target.Card
						// Collect attached Auras before removing creature
						var auraCards []Card
						for _, attID := range target.Attachments {
							att := g.FindPermanent(attID)
							if att != nil && att.Card.HasSubType("Aura") {
								auraCards = append(auraCards, att.Card)
								noted[att.Card.ID()] = notedState{owner: att.Controller}
							}
						}
						// Remove Auras from battlefield first
						for _, attID := range target.Attachments {
							att := g.FindPermanent(attID)
							if att != nil && att.Card.HasSubType("Aura") {
								g.RemoveFromBattlefield(att)
							}
						}
						g.RemoveFromBattlefield(target)
						g.ExileCard(card, sourceID)
						for _, auraCard := range auraCards {
							g.ExileCard(auraCard, sourceID)
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
					// TODO: convert to pipeline — needs exile-with-noted-state tracking
					FuncEffect("return exiled creature to battlefield",
						EffectProperties{},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							coffinReturnExiled(g, sourceID)
							return nil
						}),
				).SetCondition(IsThisSource),
			),
			// When Tawnos's Coffin becomes untapped, return exiled creature
			WithAbility(
				NewTriggered(EvtBecameUntapped, false,
					// TODO: convert to pipeline — needs exile-with-noted-state tracking
					FuncEffect("return exiled creature when Coffin becomes untapped",
						EffectProperties{},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							coffinReturnExiled(g, sourceID)
							return nil
						}),
				).SetCondition(IsThisSource),
			),
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
				// TODO: convert to pipeline — needs "add continuous effect while tapped" step
				FuncEffect("target creature gets +1/+1 while ~ remains tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
	Register("The Rack", func() Card {
		return NewArtifact("The Rack", "{1}",
			WithAbility(ChooseOpponentOnETB()),
			WithAbility(ChosenPlayerUpkeepTrigger(TheRackEffect(), false)),
		)
	})

	// Urza's Chalice {1}
	// Artifact
	// Whenever a player casts an artifact spell, you may pay {1}. If you do, you gain 1 life.
	Register("Urza's Chalice", func() Card {
		return NewArtifact("Urza's Chalice", "{1}",
			WithAbility(
				NewTriggered(EvtSpellCast, true,
					Pipeline("you may pay {1}; if you do, gain 1 life",
						EffectProperties{Outcome: OutcomeBenefit},
						IfElse("pay {1} to gain 1 life",
							&TryPayManaCond{Cost: "{1}"},
							UnwrapEffect(GainLife(1)),
							nil),
					),
				).
					// TODO: convert to data condition
					SetCondition(func(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
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
					Pipeline("pay {3} to draw a card",
						EffectProperties{Outcome: OutcomeBenefit},
						IfElse("pay {3} to draw",
							&TryPayManaCond{Cost: "{3}"},
							UnwrapEffect(DrawCards(Fixed(1))),
							nil),
					),
				).
					// TODO: convert to data condition
					SetCondition(func(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
						// Only trigger for non-sacrifice (Flag=false) artifact deaths you control
						if evt.Flag {
							return false // was sacrificed
						}
						card := g.FindCardAnywhere(evt.SourceID)
						if card == nil {
							return false
						}
						return card.HasType(TypeArtifact) && evt.PlayerID == controllerID
					}),
			),
		)
	})

	// Bronze Tablet {6}
	// Artifact
	// Remove this card from your deck before playing if you're not playing for ante.
	// Bronze Tablet enters tapped.
	// {4}, {T}: Exile Bronze Tablet and target nontoken permanent an opponent owns. That player
	// may pay 10 life. If they do, put this card into its owner's graveyard. Otherwise, that
	// player owns this card and you own the other exiled card.
	// UNIMPLEMENTABLE: Ante mechanic with permanent ownership swapping between players.
	// The engine does not support changing card ownership during a game.
	Register("Bronze Tablet", func() Card {
		return NewArtifact("Bronze Tablet", "{6}")
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
