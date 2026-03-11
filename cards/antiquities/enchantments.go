package antiquities

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// Artifact Possession {2}{B}
	// Enchantment — Aura
	// Enchant artifact
	// Whenever enchanted artifact becomes tapped or a player activates an ability of enchanted
	// artifact without {T} in its activation cost, Artifact Possession deals 2 damage to that
	// artifact's controller.
	artPossDmgEffect := FuncEffect("deal 2 damage to enchanted artifact's controller",
		EffectProperties{Outcome: OutcomeDetriment},
		func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			src := g.FindPermanent(sourceID)
			if src == nil || !src.IsAttached() {
				return nil
			}
			attached := g.FindPermanent(src.AttachedTo)
			if attached == nil {
				return nil
			}
			p := g.GetPlayer(attached.Controller)
			if p != nil {
				g.DealDamageToPlayer(p, 2, sourceID)
			}
			return nil
		})
	Register("Artifact Possession", func() Card {
		return NewAura("Artifact Possession", "{2}{B}",
			WithCastTarget(TargetArtifact()),
			// Trigger when enchanted artifact becomes tapped
			WithAbility(
				WhenAttachedBecomesTappedTrigger(artPossDmgEffect, false),
			),
			// Trigger when enchanted artifact's ability is activated without {T}
			WithAbility(
				NewTriggered(EvtAbilityActivated, false, artPossDmgEffect,
				).SetCondition(func(evt *GameEvent, g *Game, sourceID, _ uuid.UUID) bool {
					if evt.Flag {
						return false // had a tap cost — already covered by EvtTapped trigger
					}
					src := g.FindPermanent(sourceID)
					if src == nil {
						return false
					}
					return evt.SourceID == src.AttachedTo
				}),
			),
		)
	})

	// Artifact Ward {W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature can't be blocked by artifact creatures.
	// Prevent all damage that would be dealt to enchanted creature by artifact sources.
	// Enchanted creature can't be the target of abilities from artifact sources.
	Register("Artifact Ward", func() Card {
		return NewAura("Artifact Ward", "{W}",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
					func(g *Game, sourceID uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						attachedID := src.AttachedTo
						attached := g.FindPermanent(attachedID)
						if attached == nil {
							return nil
						}
						// Can't be blocked by artifact creatures
						for _, perm := range g.FilterBattlefield(And(IsCreature, IsArtifact)) {
							g.Effects.PreventBlockPair(perm.ID(), attachedID)
						}
						// Can't be targeted by abilities from artifact sources
						g.Effects.GrantAttr(attachedID, AttrCantBeTargetedByArtifacts)
						return nil
					}, SourceAttached),
			),
			// Prevent all damage from artifact sources to enchanted creature
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
					func(g *Game, sourceID uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						attachedID := src.AttachedTo
						if attachedID == (uuid.UUID{}) {
							return nil
						}
						g.Effects.AddCycleReplacement(&artifactDamageToCreaturePreventionReplacement{
							creatureID: attachedID,
						})
						return nil
					}, SourceAttached),
			),
		)
	})

	// Circle of Protection: Artifacts {1}{W}
	// Enchantment
	// {2}: The next time an artifact source of your choice would deal damage to you this turn,
	// prevent that damage.
	Register("Circle of Protection: Artifacts", func() Card {
		return NewEnchantment("Circle of Protection: Artifacts", "{1}{W}",
			WithActivatedAbility(
				FuncEffect("prevent next artifact damage from chosen source",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Choose an artifact source on the battlefield
						candidates := g.FilterBattlefield(IsArtifact)
						if len(candidates) == 0 {
							return nil
						}
						player := g.GetPlayer(controller)
						if player == nil {
							return nil
						}
						chosen := player.ChoosePermanent(candidates, "choose artifact source to prevent damage from", g)
						if chosen == nil {
							return nil
						}
						g.AddSourcePrevention(controller, chosen.ID())
						return nil
					}),
				GenericCost(2),
			),
		)
	})

	// Damping Field {2}{W}
	// Enchantment
	// Players can't untap more than one artifact during their untap steps.
	Register("Damping Field", func() Card {
		return NewEnchantment("Damping Field", "{2}{W}",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
					func(g *Game, _ uuid.UUID) error {
						if g.Effects.Rules.ArtifactUntapMax < 0 || 1 < g.Effects.Rules.ArtifactUntapMax {
							g.Effects.Rules.ArtifactUntapMax = 1
						}
						return nil
					}),
			),
		)
	})

	// Energy Flux {2}{U}
	// Enchantment
	// All artifacts have "At the beginning of your upkeep, sacrifice this artifact unless you
	// pay {2}."
	Register("Energy Flux", func() Card {
		return NewEnchantment("Energy Flux", "{2}{U}",
			WithStaticAbility(
				GrantTriggeredAbilityToAll(
					EvtUpkeep, false,
					func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
						return evt.PlayerID == controllerID
					},
					IsArtifact,
					FuncEffect("sacrifice this artifact unless you pay {2}",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.FindPermanent(sourceID)
							if perm == nil {
								return nil
							}
							if !g.TryPayCostFromLands(controller, "{2}") {
								g.Sacrifice(perm)
							}
							return nil
						}),
				),
			),
		)
	})

	// Gate to Phyrexia {B}{B}
	// Enchantment
	// Sacrifice a creature: Destroy target artifact. Activate only during your upkeep and only
	// once each turn.
	Register("Gate to Phyrexia", func() Card {
		return NewEnchantment("Gate to Phyrexia", "{B}{B}",
			WithActivatedAbility(
				FuncEffect("destroy target artifact",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm != nil {
							g.DestroyPermanent(perm)
						}
						return nil
					}),
				SacrificeCreatureCost(),
				WithTarget(TargetPermanent(IsArtifact)),
				WithUpkeepOnly(),
				WithOncePerTurn(),
			),
		)
	})

	// Haunting Wind {3}{B}
	// Enchantment
	// Whenever an artifact becomes tapped or a player activates an artifact's ability without
	// {T} in its activation cost, Haunting Wind deals 1 damage to that artifact's controller.
	hauntingWindEffect := FuncEffect("deal 1 damage to artifact's controller",
		EffectProperties{Outcome: OutcomeDetriment},
		func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			perm := g.FindPermanent(targets[0])
			if perm == nil {
				return nil
			}
			p := g.GetPlayer(perm.Controller)
			if p != nil {
				g.DealDamageToPlayer(p, 1, sourceID)
			}
			return nil
		})
	isArtifactEvt := func(evt *GameEvent, g *Game, _, _ uuid.UUID) bool {
		perm := g.FindPermanent(evt.SourceID)
		return perm != nil && perm.HasType(TypeArtifact)
	}
	Register("Haunting Wind", func() Card {
		return NewEnchantment("Haunting Wind", "{3}{B}",
			// Trigger when any artifact becomes tapped
			WithAbility(
				NewTriggered(EvtTapped, false, hauntingWindEffect).SetCondition(isArtifactEvt),
			),
			// Trigger when any artifact's ability is activated without {T}
			WithAbility(
				NewTriggered(EvtAbilityActivated, false, hauntingWindEffect,
				).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					if evt.Flag {
						return false // had a tap cost — covered by EvtTapped trigger
					}
					return isArtifactEvt(evt, g, sourceID, controllerID)
				}),
			),
		)
	})

	// Power Artifact {U}{U}
	// Enchantment — Aura
	// Enchant artifact
	// Enchanted artifact's activated abilities cost {2} less to activate. This effect can't
	// reduce the mana in that cost to less than one mana.
	Register("Power Artifact", func() Card {
		return NewAura("Power Artifact", "{U}{U}",
			WithCastTarget(TargetArtifact()),
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
					func(g *Game, sourceID uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						attachedID := src.AttachedTo
						if attachedID == (uuid.UUID{}) {
							return nil
						}
						g.Effects.Rules.ActivationCostReductions[attachedID] = 2
						return nil
					}, SourceAttached),
			),
		)
	})

	// Powerleech {G}{G}
	// Enchantment
	// Whenever an artifact an opponent controls becomes tapped or an opponent activates an
	// artifact's ability without {T} in its activation cost, you gain 1 life.
	powerleechEffect := FuncEffect("gain 1 life",
		EffectProperties{Outcome: OutcomeBenefit},
		func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			p := g.GetPlayer(controller)
			if p != nil {
				g.PlayerGainLife(p, 1)
			}
			return nil
		})
	Register("Powerleech", func() Card {
		return NewEnchantment("Powerleech", "{G}{G}",
			// Trigger when opponent's artifact becomes tapped
			WithAbility(
				WhenOpponentPermanentBecomesTappedTrigger(powerleechEffect, false, IsArtifact),
			),
			// Trigger when opponent activates artifact ability without {T}
			WithAbility(
				NewTriggered(EvtAbilityActivated, false, powerleechEffect,
				).SetCondition(func(evt *GameEvent, g *Game, _, controllerID uuid.UUID) bool {
					if evt.Flag {
						return false // had tap cost
					}
					if evt.PlayerID == controllerID {
						return false // not opponent
					}
					perm := g.FindPermanent(evt.SourceID)
					return perm != nil && perm.HasType(TypeArtifact)
				}),
			),
		)
	})

	// Titania's Song {3}{G}
	// Enchantment
	// Each noncreature artifact loses all abilities and becomes an artifact creature with power
	// and toughness each equal to its mana value. If Titania's Song leaves the battlefield, this
	// effect continues until end of turn.

	// isNoncreatureArtifactByPrint checks if a permanent is an artifact that is NOT inherently
	// a creature by its card's printed types. This is stable across all layers because it checks
	// the card's base types, not granted attrs.
	isNoncreatureArtifactByPrint := func(perm *Permanent) bool {
		if !perm.HasType(TypeArtifact) {
			return false
		}
		// Check the card's base type — HasType on the Card checks the card's registered
		// types, not granted attrs from LayerType
		return !perm.Card.HasType(TypeCreature)
	}

	// Three functions for the three layers:
	titaniasSongType := func(g *Game, _ uuid.UUID) error {
		for _, perm := range g.Battlefield {
			if isNoncreatureArtifactByPrint(perm) {
				g.Effects.GrantAttr(perm.ID(), AttrIsCreature)
				g.Effects.GrantAttr(perm.ID(), AttrCanAttack)
				g.Effects.GrantAttr(perm.ID(), AttrCanBlock)
				g.Effects.GrantAttr(perm.ID(), AttrHasPowerToughness)
			}
		}
		return nil
	}
	titaniasSongAbility := func(g *Game, _ uuid.UUID) error {
		for _, perm := range g.Battlefield {
			if isNoncreatureArtifactByPrint(perm) {
				perm.RuntimeAbilities = nil
			}
		}
		return nil
	}
	titaniasSongPT := func(g *Game, _ uuid.UUID) error {
		for _, perm := range g.Battlefield {
			if isNoncreatureArtifactByPrint(perm) {
				cmc := perm.Card.ManaCost().CMC()
				perm.BasePTOverride = &[2]int{cmc, cmc}
			}
		}
		return nil
	}

	Register("Titania's Song", func() Card {
		return NewEnchantment("Titania's Song", "{3}{G}",
			WithStaticAbility(
				FuncContinuousEffect(LayerType, WhileOnBattlefield, titaniasSongType),
			),
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, titaniasSongAbility),
			),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, titaniasSongPT),
			),
			// When Song leaves the battlefield, continue the effect until end of turn
			WithAbility(
				NewTriggered(EvtLeavesBattlefield, false,
					FuncEffect("continue Titania's Song effect until end of turn",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							effType := FuncContinuousEffect(LayerType, EndOfTurn, titaniasSongType)
							effType.SetSourceID(sourceID)
							g.AddContinuousEffect(effType)
							effAbil := FuncContinuousEffect(LayerAbility, EndOfTurn, titaniasSongAbility)
							effAbil.SetSourceID(sourceID)
							g.AddContinuousEffect(effAbil)
							effPT := FuncContinuousEffect(LayerPT, EndOfTurn, titaniasSongPT)
							effPT.SetSourceID(sourceID)
							g.AddContinuousEffect(effPT)
							return nil
						}),
				).SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == sourceID
				}),
			),
		)
	})
}

// artifactDamageToCreaturePreventionReplacement prevents all damage from
// artifact sources to the specified creature (Artifact Ward).
type artifactDamageToCreaturePreventionReplacement struct {
	creatureID uuid.UUID
}

func (r *artifactDamageToCreaturePreventionReplacement) SourceID() uuid.UUID { return uuid.Nil }

func (r *artifactDamageToCreaturePreventionReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToCreatureAction)
	if !ok {
		return false
	}
	if act.PermanentID() != r.creatureID {
		return false
	}
	source := g.FindCardAnywhere(act.ActionSource())
	if source == nil {
		return false
	}
	return source.HasType(TypeArtifact)
}

func (r *artifactDamageToCreaturePreventionReplacement) Replace(a Action, g GameMutator) Action {
	return nil // prevent the damage
}

func (r *artifactDamageToCreaturePreventionReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.creatureID) != nil
}

func (r *artifactDamageToCreaturePreventionReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}
