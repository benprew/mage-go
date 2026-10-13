package antiquities

import (
	"github.com/google/uuid"

	mage "github.com/benprew/mage-go/pkg/mage"

	"github.com/benprew/mage-go/pkg/catalog"
	"github.com/benprew/mage-go/pkg/mage/core"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// Amulet of Kroog {2}
	// Artifact
	// {2}, {T}: Prevent the next 1 damage that would be dealt to any target this turn.
	Register("Amulet of Kroog", func() Card {
		recipient := TargetDamageRecipient("recipient")
		return NewArtifact("Amulet of Kroog", "{2}",
			WithAbility(Activate(
				Costs(Mana("{2}"), TapCost(This)),
				PreventNextDamage(recipient, Fixed(1)),
				WithTargets(recipient),
			)),
		)
	})

	// Armageddon Clock {6}
	// Artifact
	// At the beginning of your upkeep, put a doom counter on this artifact.
	// At the beginning of your draw step, this artifact deals damage equal to the number of
	// doom counters on it to each player.
	// {4}: Remove a doom counter from this artifact. Any player may activate this ability but
	// only during any upkeep step.
	Register("Armageddon Clock", func() Card {
		return NewArtifact("Armageddon Clock", "{6}",
			WithAbility(At(YourUpkeep,
				AddCounters(This, Doom, Fixed(1)),
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
			WithAbility(Activate(
				Costs(Mana("{4}")),
				RemoveCountersFrom(This, Doom, Fixed(1)),
				OnlyDuring(core.Upkeep),
				AnyPlayerMayActivate(),
			)),
		)
	})

	// Ashnod's Altar {3}
	// Artifact
	// Sacrifice a creature: Add {C}{C}.
	Register("Ashnod's Altar", func() Card {
		return NewArtifact("Ashnod's Altar", "{3}",
			WithAbility(Activate(
				Costs(SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice a creature")),
				AddManaAmount(Colorless, Fixed(2)),
			)),
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
				Boost(Fixed(2), Fixed(-2)).
					Targeting(ToTarget()).
					Until(WhileOnBattlefield).
					WhileSourceTapped(),
				GenericCost(2),
				WithCost(TapCost(This)),
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
				Pipeline("put +1/+1 counter and make artifact",
					EffectProperties{Outcome: OutcomeBenefit},
					mage.AddCounters(P1P1, Fixed(1)),
					GrantType(TypeArtifact),
				),
				TapCost(This),
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
						for range x {
							candidates := g.FilterBattlefield(And(IsLand, IsTapped))
							if len(candidates) == 0 {
								break
							}
							chosen := player.ChoosePermanent(candidates, "untap land", g)
							if chosen == nil {
								break
							}
							mp := g.MutablePermanent(chosen.ID())
							if mp == nil {
								continue
							}
							mp.Tapped = false
						}
						return nil
					}),
				XManaCost(),
				WithCost(TapCost(This)),
			),
		)
	})

	// Coral Helm {3}
	// Artifact
	// {3}, Discard a card at random: Target creature gets +2/+2 until end of turn.
	Register("Coral Helm", func() Card {
		creature := TargetPermanent("creature", IsCreature)
		return NewArtifact("Coral Helm", "{3}",
			WithAbility(Activate(
				Costs(Mana("{3}"), DiscardRandomCost(1)),
				Until(EndOfTurn, ModifyPT(creature, Fixed(2), Fixed(2))),
				WithTargets(creature),
			)),
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
						opponent := g.GetOpponent(src.ControllerID())
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
				ShuffleGraveyardIntoLibrary(),
				TapCost(This),
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
							g.DoSacrifice(perm)
						}
						return nil
					}),
				GenericCost(1),
				WithCost(TapCost(This)),
			),
		)
	})

	// Ivory Tower {1}
	// Artifact
	// At the beginning of your upkeep, you gain X life, where X is the number of cards in your hand minus
	// 4.
	Register("Ivory Tower", func() Card {
		return NewArtifact("Ivory Tower", "{1}",
			WithAbility(At(YourUpkeep,
				GainLife(You, SubtractValues(ValueAtResolution(CountZone(ZoneHand, SelectController(), CardFilter{})), Fixed(4))),
			)),
		)
	})

	// Jalum Tome {3}
	// Artifact
	// {2}, {T}: Draw a card, then discard a card.
	Register("Jalum Tome", func() Card {
		return NewArtifact("Jalum Tome", "{3}",
			WithAbility(Activate(Costs(Mana("{2}"), TapCost(This)),
				Sequence(Draw(You, Fixed(1)), Discard(You, Fixed(1))))),
		)
	})

	// Mightstone {4}
	// Artifact
	// Attacking creatures get +1/+0.
	Register("Mightstone", func() Card {
		return NewArtifact("Mightstone", "{4}",
			WithAbility(Static(While(Controls(AnyPermanent),
				ModifyEachPT(MatchingPermanents(And(IsCreature, IsAttacking)), Fixed(1), Fixed(0))))),
		)
	})

	// Millstone {2}
	// Artifact
	// {2}, {T}: Target player mills two cards.
	Register("Millstone", func() Card {
		player := TargetPlayer("player")
		return NewArtifact("Millstone", "{2}",
			WithAbility(Activate(Costs(Mana("{2}"), TapCost(This)),
				Mill(player, Fixed(2)), WithTargets(player))),
		)
	})

	// Obelisk of Undoing {1}
	// Artifact
	// {6}, {T}: Return target permanent you both own and control to your hand.
	Register("Obelisk of Undoing", func() Card {
		permanent := TargetPermanent("permanent", AnyPermanent, TargetYouOwn(), TargetYouControl())
		return NewArtifact("Obelisk of Undoing", "{1}",
			WithAbility(Activate(
				Costs(Mana("{6}"), TapCost(This)),
				ReturnToHand(permanent),
				WithTargets(permanent),
			)),
		)
	})

	// Rakalite {6}
	// Artifact
	// {2}: Prevent the next 1 damage that would be dealt to any target this turn. Return
	// this artifact to its owner's hand at the beginning of the next end step.
	Register("Rakalite", func() Card {
		recipient := TargetDamageRecipient("recipient")
		return NewArtifact("Rakalite", "{6}",
			WithAbility(Activate(
				Costs(Mana("{2}")),
				Sequence(
					PreventNextDamage(recipient, Fixed(1)),
					AtNext(EachEndStep, ReturnToHand(This)),
				),
				WithTargets(recipient),
			)),
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
					DealDamage(Fixed(1)),
					RegisterDelayedTriggerStep(EvtEndStep, "", DestroyTarget()),
				),
				GenericCost(2),
				WithTarget(TargetDamageAnyTarget()),
				WithControlledSinceTurnStart(),
			),
		)
	})

	// Staff of Zegon {4}
	// Artifact
	// {3}, {T}: Target creature gets -2/-0 until end of turn.
	Register("Staff of Zegon", func() Card {
		target := TargetPermanent("creature", IsCreature)
		return NewArtifact("Staff of Zegon", "{4}",
			WithAbility(Activate(Costs(ManaCostOf("{3}"), TapCost(This)),
				Until(EndOfTurn, ModifyPT(target, Fixed(-2), Fixed(0))), WithTargets(target))),
		)
	})

	// Tablet of Epityr {1}
	// Artifact
	// Whenever an artifact you control is put into a graveyard from the battlefield, you may
	// pay {1}. If you do, you gain 1 life.
	Register("Tablet of Epityr", func() Card {
		return NewArtifact("Tablet of Epityr", "{1}",
			WithAbility(
				NewTriggered(EvtZoneChange, true,
					IfPaid(You, Costs(Mana("{1}")), GainLife(You, Fixed(1))),
				).
					SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
						EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
						EventPlayerIsController{},
						SpellCastIsType{Type: TypeArtifact},
					}}),
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
			counters [NumCounters]int
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
					for ct := range NumCounters {
						if count := ns.counters[ct]; count != 0 {
							creaturePerm.AddCounter(ct, count)
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
							owner:    target.ControllerID(),
						}
						card := target.Card
						// Collect attached Auras before removing creature
						var auraCards []Card
						for _, attID := range target.Attachments {
							att := g.FindPermanent(attID)
							if att != nil && att.Card.HasSubType("Aura") {
								auraCards = append(auraCards, att.Card)
								noted[att.Card.ID()] = notedState{owner: att.ControllerID()}
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
				WithCost(TapCost(This)),
				WithTarget(TargetCreature()),
			),
			// When Tawnos's Coffin leaves the battlefield, return exiled creature
			WithAbility(
				OnLeaveZone(ZoneBattlefield, ZoneAny,
					// TODO: convert to pipeline — needs exile-with-noted-state tracking
					FuncEffect("return exiled creature to battlefield",
						EffectProperties{},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							coffinReturnExiled(g, sourceID)
							return nil
						}),
					false,
				),
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
				).SetConditionData(EventSourceIsSelf{}),
			),
		)
	})

	// Tawnos's Wand {4}
	// Artifact
	// {2}, {T}: Target creature with power 2 or less can't be blocked this turn.
	Register("Tawnos's Wand", func() Card {
		creature := TargetPermanent("creature", And(IsCreature, HasPowerLTE(2)))
		return NewArtifact("Tawnos's Wand", "{4}",
			WithAbility(Activate(Costs(Mana("{2}"), TapCost(This)),
				Until(EndOfTurn, CantBeBlocked(creature)), WithTargets(creature))),
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
				Boost(Fixed(1), Fixed(1)).
					Targeting(ToTarget()).
					Until(WhileOnBattlefield).
					WhileSourceTapped(),
				GenericCost(2),
				WithCost(TapCost(This)),
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
					IfPaid(You, Costs(Mana("{1}")), GainLife(You, Fixed(1))),
				).
					SetConditionData(SpellCastIsType{Type: TypeArtifact}),
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
				NewTriggered(EvtZoneChange, true,
					IfPaid(You, Costs(Mana("{3}")), Draw(You, Fixed(1))),
				).
					SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
						EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
						EventFlagIsFalse{},
						SpellCastIsType{Type: TypeArtifact},
						EventPlayerIsController{},
					}}),
			),
		)
	})

	// Bronze Tablet {6}
	// Artifact
	// Remove this card from your deck before playing if you're not playing for ante.
	// This artifact enters tapped.
	// {4}, {T}: Exile this artifact and target nontoken permanent an opponent owns. That player may pay 10 life. If they do, put this card into its owner's graveyard. Otherwise, that player owns this card and you own the other exiled card.
	Register("Bronze Tablet", func() Card {
		return NewArtifact("Bronze Tablet", "{6}",
			WithKeyword(EntersTapped),
			WithActivatedAbility(
				Pipeline(
					"exile this artifact and target nontoken permanent an opponent owns",
					EffectProperties{},
					SnapshotPermanent(SelectSource, "tablet"),
					SnapshotPermanent(SelectTarget, "target"),
					ExileGathered("tablet"),
					ExileGathered("target"),
					IfPlayerPays(
						VarPlayer("target.owner"),
						LifePayCost(10),
						"pay 10 life for Bronze Tablet",
						MoveExiledGatheredToGraveyard("tablet"),
						Pipeline(
							"exchange ownership",
							EffectProperties{},
							ChangeOwnerGathered("tablet", VarPlayer("target.owner")),
							ChangeOwnerGathered("target", SelectController()),
						),
					),
				),
				GenericCost(4),
				WithCost(TapCost(This)),
				WithTarget(TargetPermanentOpponentOwns(Not(IsToken))),
			),
		)
	})

	// Weakstone {4}
	// Artifact
	// Attacking creatures get -1/-0.
	Register("Weakstone", func() Card {
		return NewArtifact("Weakstone", "{4}",
			WithAbility(Static(While(Controls(AnyPermanent),
				ModifyEachPT(MatchingPermanents(And(IsCreature, IsAttacking)), Fixed(-1), Fixed(0))))),
		)
	})
}
