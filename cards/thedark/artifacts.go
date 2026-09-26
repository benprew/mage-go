package thedark

import (
	"fmt"
	"math/rand"

	mage "github.com/benprew/mage-go/pkg/mage"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {

	// Barl's Cage {4}
	// Artifact
	// {3}: Target creature doesn't untap during its controller's next untap step.
	Register("Barl's Cage", func() Card {
		creature := TargetPermanent("creature", IsCreature)
		return NewArtifact("Barl's Cage", "{4}",
			WithAbility(Activate(Costs(Mana("{3}")),
				SkipNextUntap(creature), WithTargets(creature))),
		)
	})

	// Bone Flute {3}
	// Artifact
	// {2}, {T}: All creatures get -1/-0 until end of turn.
	Register("Bone Flute", func() Card {
		return NewArtifact("Bone Flute", "{3}",
			WithAbility(Activate(Costs(Mana("{2}"), TapCost(This)),
				Until(EndOfTurn, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(-1), Fixed(0))))),
		)
	})

	// Book of Rass {6}
	// Artifact
	// {2}, Pay 2 life: Draw a card.
	Register("Book of Rass", func() Card {
		return NewArtifact("Book of Rass", "{6}",
			WithAbility(Activate(
				Costs(Mana("{2}"), LifePayCost(2)),
				Draw(You, Fixed(1)),
			)),
		)
	})

	// Dark Sphere {0}
	// Artifact
	// {T}, Sacrifice this artifact: The next time a source of your choice would deal damage to you this turn, prevent half that damage, rounded down.
	Register("Dark Sphere", func() Card {
		return NewArtifact("Dark Sphere", "{0}",
			WithActivatedAbility(
				FuncEffect(
					"the next time a source of your choice would deal damage to you this turn, prevent half that damage, rounded down",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						g.AddHalfDamageFromSourcePreventionShield(controller, targets[0])
						return nil
					},
				),
				TapCost(This),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetDamageSource()),
			),
		)
	})

	// Fountain of Youth {0}
	// Artifact
	// {2}, {T}: You gain 1 life.
	Register("Fountain of Youth", func() Card {
		return NewArtifact("Fountain of Youth", "{0}",
			WithAbility(Activate(
				Costs(Mana("{2}"), TapCost(This)),
				GainLife(You, Fixed(1)),
			)),
		)
	})

	// Living Armor {4}
	// Artifact
	// {T}, Sacrifice this artifact: Put X +0/+1 counters on target creature, where X is that creature's mana value.
	Register("Living Armor", func() Card {
		creature := TargetPermanent("creature", IsCreature)
		return NewArtifact("Living Armor", "{4}",
			WithAbility(Activate(
				Costs(TapCost(This), SacrificeSourceCost()),
				AddCounters(creature, P0P1, ManaValueOf(creature)),
				WithTargets(creature),
			)),
		)
	})

	// Reflecting Mirror {4}
	// Artifact
	// {X}, {T}: Change the target of target spell with a single target if that target is you. The new target must be a player. X is twice the mana value of that spell.
	Register("Reflecting Mirror", func() Card {
		return NewArtifact("Reflecting Mirror", "{4}",
			WithActivatedAbility(
				FuncEffect(
					"change the target of target spell with a single target if that target is you; new target must be a player; X is twice its mana value",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) < 2 {
							return nil
						}
						spellTargetID := targets[0]
						newPlayerID := targets[1]
						for _, obj := range g.StackObjects() {
							if obj.SourceID == spellTargetID {
								if len(obj.Targets) == 1 && obj.Targets[0] == controller {
									if g.XValue() != 2*obj.Card.ManaCost().CMC() {
										return fmt.Errorf("reflecting mirror X value %d does not equal twice mana value %d", g.XValue(), 2*obj.Card.ManaCost().CMC())
									}
									obj.Targets = []uuid.UUID{newPlayerID}
								}
								break
							}
						}
						return nil
					},
				),
				TapCost(This),
				WithCost(XManaCost()),
				WithTarget(TargetSpellOnStack()),
				WithTarget(mage.TargetPlayer()),
			),
		)
	})

	// Runesword {6}
	// Artifact
	// {3}, {T}: Target attacking creature gets +2/+0 until end of turn. When that creature leaves the battlefield this turn, sacrifice this artifact. If the creature deals damage to a creature this turn, the creature dealt damage can't be regenerated this turn. If a creature dealt damage by the targeted creature would die this turn, exile that creature instead.
	Register("Runesword", func() Card {
		return NewArtifact("Runesword", "{6}",
			WithActivatedAbility(
				FuncEffect(
					"target attacking creature gets +2/+0 until end of turn; when it leaves the battlefield this turn, sacrifice Runesword; damage prevention/exile clauses",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						targetPerm := g.FindPermanent(targetID)
						if targetPerm == nil {
							return nil
						}
						// +2/+0 until end of turn
						boost := TemporaryBoost(targetID, 2, 0)
						boost.SetSourceID(sourceID)
						g.AddContinuousEffect(boost)

						// When that creature leaves the battlefield this turn, sacrifice this artifact
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:     EvtZoneChange,
							SourceID:      sourceID,
							Controller:    controller,
							MatchEventID:  targetID,
							MatchFromZone: ZoneBattlefield,
							Effects:       []Effect{Sacrifice(This)},
						})

						// If the creature deals damage to a creature this turn, the creature dealt damage can't be regenerated this turn.
						// If a creature dealt damage by the targeted creature would die this turn, exile that creature instead.
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:    EvtDamageDealt,
							SourceID:     sourceID,
							Controller:   controller,
							MatchEventID: targetID,
							Persistent:   true,
							Effects: []Effect{
								FuncEffect(
									"creature dealt damage can't regenerate and is exiled if it dies this turn",
									EffectProperties{},
									func(g *Game, srcID, ctrl uuid.UUID, tgts []uuid.UUID) error {
										if len(tgts) == 0 {
											return nil
										}
										victimID := tgts[0]
										victim := g.FindPermanent(victimID)
										if victim != nil && victim.HasType(TypeCreature) {
											victim.GrantBaseAttr(CantRegenerate)
											g.AddExileIfWouldGoToGraveyardThisTurn(victim.Card.ID(), srcID)
										}
										return nil
									},
								),
							},
						})
						return nil
					},
				),
				ManaCostOf("{3}"),
				WithCost(TapCost(This)),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	})

	// Skull of Orm {3}
	// Artifact
	// {5}, {T}: Return target enchantment card from your graveyard to your hand.
	Register("Skull of Orm", func() Card {
		enchantment := TargetGraveyardCard("enchantment", GraveyardCardFilter(IsEnchantmentCard))
		return NewArtifact("Skull of Orm", "{3}",
			WithAbility(Activate(Costs(Mana("{5}"), TapCost(This)),
				ReturnCardToHand(enchantment), WithTargets(enchantment))),
		)
	})

	// Standing Stones {3}
	// Artifact
	// {1}, {T}, Pay 1 life: Add one mana of any color.
	Register("Standing Stones", func() Card {
		return NewArtifact("Standing Stones", "{3}",
			WithActivatedAbility(
				AddAnyMana(1, Colorless),
				ManaCostOf("{1}"),
				WithCost(TapCost(This)),
				WithCost(LifePayCost(1)),
			),
		)
	})

	// Stone Calendar {5}
	// Artifact
	// Spells you cast cost {1} less to cast.
	Register("Stone Calendar", func() Card {
		return NewArtifact("Stone Calendar", "{5}",
			WithStaticAbility(ReduceSpellCostStatic(
				SpellAny(),
				FixedAmount(1),
				nil,
			)),
		)
	})

	// Tormod's Crypt {0}
	// Artifact
	// {T}, Sacrifice this artifact: Exile target player's graveyard.
	Register("Tormod's Crypt", func() Card {
		return NewArtifact("Tormod's Crypt", "{0}",
			WithActivatedAbility(
				FuncEffect(
					"exile target player's graveyard",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetPlayer := g.GetPlayer(targets[0])
						if targetPlayer == nil {
							return nil
						}
						var cardIDs []uuid.UUID
						for _, c := range targetPlayer.Graveyard() {
							cardIDs = append(cardIDs, c.ID())
						}
						removed := g.MoveCardsFromGraveyard(targets[0], cardIDs, ZoneExile)
						for _, c := range removed {
							g.ExileCard(c, sourceID)
						}
						return nil
					},
				),
				TapCost(This),
				WithCost(SacrificeSourceCost()),
				WithTarget(mage.TargetPlayer()),
			),
		)
	})

	// Tower of Coireall {2}
	// Artifact
	// {T}: Target creature can't be blocked by Walls this turn.
	Register("Tower of Coireall", func() Card {
		creature := TargetPermanent("creature", IsCreature)
		return NewArtifact("Tower of Coireall", "{2}",
			WithAbility(Activate(Costs(TapCost(This)),
				Until(EndOfTurn, GrantKeyword(creature, CantBeBlockedByWalls)), WithTargets(creature))),
		)
	})

	// Wand of Ith {4}
	// Artifact
	// {3}, {T}: Target player reveals a card at random from their hand. If it's a land card, that player discards it unless they pay 1 life. If it isn't a land card, the player discards it unless they pay life equal to its mana value. Activate only during your turn.
	Register("Wand of Ith", func() Card {
		return NewArtifact("Wand of Ith", "{4}",
			WithActivatedAbility(
				FuncEffect(
					"target player reveals a card at random from their hand; discard unless pays life",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetPlayer := g.GetPlayer(targets[0])
						if targetPlayer == nil {
							return nil
						}
						hand := targetPlayer.Hand()
						if len(hand) == 0 {
							return nil
						}
						chosenCard := hand[rand.Intn(len(hand))]
						if chosenCard.HasType(TypeLand) {
							// Discard unless pays 1 life
							if targetPlayer.Life() > 1 && targetPlayer.ChooseMayAbility("pay 1 life to avoid discarding "+chosenCard.Name()) {
								targetPlayer.LoseLife(1)
							} else {
								g.PlayerDiscardByEffect(targetPlayer, chosenCard.ID(), sourceID)
							}
						} else {
							cmc := chosenCard.ManaCost().CMC()
							if cmc == 0 {
								// 0 life is paid automatically or discards? "unless they pay life equal to its mana value"
								// If 0 life, can pay 0 life to avoid discard.
								if targetPlayer.ChooseMayAbility("pay 0 life to avoid discarding " + chosenCard.Name()) {
									// paid 0 life
								} else {
									g.PlayerDiscardByEffect(targetPlayer, chosenCard.ID(), sourceID)
								}
							} else if targetPlayer.Life() > cmc && targetPlayer.ChooseMayAbility(fmt.Sprintf("pay %d life to avoid discarding %s", cmc, chosenCard.Name())) {
								targetPlayer.LoseLife(cmc)
							} else {
								g.PlayerDiscardByEffect(targetPlayer, chosenCard.ID(), sourceID)
							}
						}
						return nil
					},
				),
				ManaCostOf("{3}"),
				WithCost(TapCost(This)),
				WithYourTurnOnly(),
				WithTarget(mage.TargetPlayer()),
			),
		)
	})

	// War Barge {4}
	// Artifact
	// {3}: Target creature gains islandwalk until end of turn. When this artifact leaves the battlefield this turn, destroy that creature. A creature destroyed this way can't be regenerated. (A creature with islandwalk can't be blocked as long as defending player controls an Island.)
	Register("War Barge", func() Card {
		return NewArtifact("War Barge", "{4}",
			WithActivatedAbility(
				FuncEffect(
					"target creature gains islandwalk until end of turn; when War Barge leaves the battlefield this turn, destroy that creature without regeneration",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						grant := mage.GrantKeyword(Islandwalk)
						if err := ApplyEffect(g, grant, sourceID, controller, targets); err != nil {
							return err
						}

						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:     EvtZoneChange,
							SourceID:      sourceID,
							Controller:    controller,
							MatchEventID:  sourceID,
							MatchFromZone: ZoneBattlefield,
							Effects: []Effect{
								FuncEffect(
									"destroy target creature without regeneration",
									EffectProperties{Outcome: OutcomeDetriment},
									func(g *Game, srcID, ctrl uuid.UUID, _ []uuid.UUID) error {
										p := g.FindPermanent(targetID)
										if p != nil {
											p.GrantBaseAttr(CantRegenerate)
											g.DestroyPermanent(p)
										}
										return nil
									},
								),
							},
						})
						return nil
					},
				),
				ManaCostOf("{3}"),
				WithTarget(TargetCreature()),
			),
		)
	})

}
