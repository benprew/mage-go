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
	// XXX: missing damage prevention from artifact sources and targeting restriction from artifact sources
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
						for _, perm := range g.FilterBattlefield(And(IsCreature, IsArtifact)) {
							g.Effects.PreventBlockPair(perm.ID(), attachedID)
						}
						return nil
					}, SourceAttached),
			),
		)
	})

	// Circle of Protection: Artifacts {1}{W}
	// Enchantment
	// {2}: The next time an artifact source of your choice would deal damage to you this turn,
	// prevent that damage.
	// XXX: prevents all artifact damage rather than "next time from a chosen source" per Oracle
	Register("Circle of Protection: Artifacts", func() Card {
		return NewEnchantment("Circle of Protection: Artifacts", "{1}{W}",
			WithActivatedAbility(
				FuncEffect("prevent next artifact damage",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						g.AddTypePrevention(controller, TypeArtifact)
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
	// XXX: auto-pays sequentially from mana pool instead of per-artifact player choice
	Register("Energy Flux", func() Card {
		return NewEnchantment("Energy Flux", "{2}{U}",
			WithAbility(
				BeginningOfEachUpkeepTrigger(
					FuncEffect("sacrifice artifacts unless {2} paid",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							active := g.ActivePlayerObj()
							activeID := active.PlayerID()
							var toSacrifice []*Permanent
							for _, perm := range g.FilterBattlefield(And(IsArtifact, ControlledBy(activeID))) {
								cost := ParseManaCost("{2}")
								if active.ManaPool().CanPay(cost) {
									_ = active.ManaPool().Pay(cost)
								} else {
									toSacrifice = append(toSacrifice, perm)
								}
							}
							for _, perm := range toSacrifice {
								g.Sacrifice(perm)
							}
							return nil
						}),
					false),
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
	// XXX: cost reduction for attached artifact
	Register("Power Artifact", func() Card {
		return NewAura("Power Artifact", "{U}{U}")
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
	// XXX: missing "loses all abilities"; missing "effect continues until end of turn" after leaving
	Register("Titania's Song", func() Card {
		return NewEnchantment("Titania's Song", "{3}{G}",
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield,
					func(g *Game, sourceID uuid.UUID) error {
						for _, perm := range g.FilterBattlefield(And(IsArtifact, Not(IsCreature))) {
							cmc := perm.Card.ManaCost().CMC()
							perm.Card.AddType(TypeCreature)
							perm.Card.SetBasePT(cmc, cmc)
						}
						return nil
					}),
			),
		)
	})
}
