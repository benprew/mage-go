package jumpstart

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {

	// Aether Spellbomb {1}
	// Artifact
	// {U}, Sacrifice this artifact: Return target creature to its owner's hand.
	// {1}, Sacrifice this artifact: Draw a card.
	Register("Aether Spellbomb", func() Card {
		return NewArtifact("Aether Spellbomb", "{1}",
			WithActivatedAbility(
				ReturnToHandTarget(),
				ManaCostOf("{U}"),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCreature()),
			),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				GenericCost(1),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Arcane Encyclopedia {3}
	// Artifact — Book
	// {3}, {T}: Draw a card.
	Register("Arcane Encyclopedia", func() Card {
		return NewArtifact("Arcane Encyclopedia", "{3}",
			WithSubTypes("Book"),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				GenericCost(3),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Bubbling Cauldron {2}
	// Artifact
	// {1}, {T}, Sacrifice a creature: You gain 4 life.
	// {1}, {T}, Sacrifice a creature named Festering Newt: Each opponent loses 4 life. You gain life equal to the life lost this way.
	Register("Bubbling Cauldron", func() Card {
		return NewArtifact("Bubbling Cauldron", "{2}",
			WithActivatedAbility(
				GainLife(4),
				GenericCost(1),
				WithCost(TapSourceCost()),
				WithCost(SacrificeCreatureCost()),
			),
			WithActivatedAbility(
				FuncEffect("each opponent loses 4 life; you gain that much life",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						total := 0
						for _, p := range g.AllPlayers() {
							if p.PlayerID() == controller {
								continue
							}
							before := p.Life()
							p.LoseLife(4)
							lost := before - p.Life()
							if lost > 0 {
								total += lost
							}
						}
						if total > 0 {
							if you := g.GetPlayer(controller); you != nil {
								g.PlayerGainLife(you, total)
							}
						}
						return nil
					}),
				GenericCost(1),
				WithCost(TapSourceCost()),
				WithCost(SacrificeMatchingCost(And(IsCreature, Named("Festering Newt")), "Sacrifice a creature named Festering Newt")),
			),
		)
	})

	// Chromatic Sphere {1}
	// Artifact
	// {1}, {T}, Sacrifice this artifact: Add one mana of any color. Draw a card.
	Register("Chromatic Sphere", func() Card {
		return NewArtifact("Chromatic Sphere", "{1}",
			WithActivatedAbility(
				CompositeEffects("add one mana of any color, then draw a card",
					AddAnyMana(1, Colorless),
					DrawCards(Fixed(1)),
				),
				GenericCost(1),
				WithCost(TapSourceCost()),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Dreamstone Hedron {6}
	// Artifact
	// {T}: Add {C}{C}{C}.
	// {3}, {T}, Sacrifice this artifact: Draw three cards.
	Register("Dreamstone Hedron", func() Card {
		return NewArtifact("Dreamstone Hedron", "{6}",
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 3}),
			WithActivatedAbility(
				DrawCards(Fixed(3)),
				GenericCost(3),
				WithCost(TapSourceCost()),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Guardian Idol {2}
	// Artifact
	// This artifact enters tapped.
	// {T}: Add {C}.
	// {2}: This artifact becomes a 2/2 Golem artifact creature until end of turn.
	Register("Guardian Idol", func() Card {
		return NewArtifact("Guardian Idol", "{2}",
			WithKeyword(EntersTapped),
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 1}),
			WithActivatedAbility(
				FuncEffect("becomes a 2/2 Golem artifact creature until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						eff := TemporaryAnimate(sourceID, 2, 2)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				GenericCost(2),
			),
		)
	})

	// Hedron Archive {4}
	// Artifact
	// {T}: Add {C}{C}.
	// {2}, {T}, Sacrifice this artifact: Draw two cards.
	Register("Hedron Archive", func() Card {
		return NewArtifact("Hedron Archive", "{4}",
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}),
			WithActivatedAbility(
				DrawCards(Fixed(2)),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Herald's Horn {3}
	// Artifact
	// As this artifact enters, choose a creature type.
	// Creature spells you cast of the chosen type cost {1} less to cast.
	// At the beginning of your upkeep, look at the top card of your library. If it's a creature card of the chosen type, you may reveal it and put it into your hand.
	// XXX: requires "as enters choose a creature subtype" + per-subtype spell cost reduction (engine has color/type cost reductions but not subtype). Also requires top-of-library reveal/draw mechanic.
	Register("Herald's Horn", func() Card {
		return NewArtifact("Herald's Horn", "{3}")
	})

	// Mana Geode {3}
	// Artifact
	// When this artifact enters, scry 1.
	// {T}: Add one mana of any color.
	// XXX: scry is not implemented in the engine; the ETB scry is a no-op.
	Register("Mana Geode", func() Card {
		return NewArtifact("Mana Geode", "{3}",
			WithAnyColorMana(),
		)
	})

	// Marauder's Axe {2}
	// Artifact — Equipment
	// Equipped creature gets +2/+0.
	// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Marauder's Axe", func() Card {
		return NewEquipment("Marauder's Axe", "{2}",
			WithStaticAbility(BoostAttached(2, 0, AttachEquipment)),
			WithAbility(NewEquipAbility(ManaCostOf("{2}"))),
		)
	})

	// Pirate's Cutlass {3}
	// Artifact — Equipment
	// When this Equipment enters, attach it to target Pirate you control.
	// Equipped creature gets +2/+1.
	// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Pirate's Cutlass", func() Card {
		return NewEquipment("Pirate's Cutlass", "{3}",
			WithStaticAbility(BoostAttached(2, 1, AttachEquipment)),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("attach to target Pirate you control",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						pirates := g.FilterBattlefield(And(IsCreature, HasSubType("Pirate"), ControlledBy(controller)))
						if len(pirates) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						var chosen *Permanent
						if p != nil {
							chosen = p.ChoosePermanent(pirates, "attach Pirate's Cutlass to a Pirate", g)
						}
						if chosen == nil {
							chosen = pirates[0]
						}
						g.Attach(sourceID, chosen.ID())
						return nil
					}),
				false,
			)),
			WithAbility(NewEquipAbility(ManaCostOf("{2}"))),
		)
	})

	// Prophetic Prism {2}
	// Artifact
	// When this artifact enters, draw a card.
	// {1}, {T}: Add one mana of any color.
	Register("Prophetic Prism", func() Card {
		return NewArtifact("Prophetic Prism", "{2}",
			WithAbility(EntersBattlefieldTrigger(DrawCards(Fixed(1)), false)),
			WithActivatedAbility(
				AddAnyMana(1, Colorless),
				GenericCost(1),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Rogue's Gloves {2}
	// Artifact — Equipment
	// Whenever equipped creature deals combat damage to a player, you may draw a card.
	// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Rogue's Gloves", func() Card {
		return NewEquipment("Rogue's Gloves", "{2}",
			WithAbility(NewTriggered(EvtDamageDealt, true,
				FuncEffect("you may draw a card",
					EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						g.PlayerDrawCard(p)
						return nil
					}),
			).SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
				if !evt.Flag {
					return false
				}
				if g.GetPlayer(evt.TargetID) == nil {
					return false
				}
				src := g.FindPermanent(sourceID)
				if src == nil {
					return false
				}
				return src.IsAttached() && src.AttachedTo == evt.SourceID
			})),
			WithAbility(NewEquipAbility(ManaCostOf("{2}"))),
		)
	})

	// Scroll of Avacyn {1}
	// Artifact
	// {1}, Sacrifice this artifact: Draw a card. If you control an Angel, you gain 5 life.
	Register("Scroll of Avacyn", func() Card {
		return NewArtifact("Scroll of Avacyn", "{1}",
			WithActivatedAbility(
				CompositeEffects("draw a card; if you control an Angel, you gain 5 life",
					DrawCards(Fixed(1)),
					FuncEffect("if you control an Angel, you gain 5 life",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							for range g.FilterBattlefield(And(IsCreature, HasSubType("Angel"), ControlledBy(controller))) {
								g.PlayerGainLife(p, 5)
								return nil
							}
							return nil
						}),
				),
				GenericCost(1),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Terrarion {1}
	// Artifact
	// This artifact enters tapped.
	// {2}, {T}, Sacrifice this artifact: Add two mana in any combination of colors.
	// When this artifact is put into a graveyard from the battlefield, draw a card.
	// XXX: self-graveyard trigger doesn't fire when sacrificed because Game.Sacrifice
	// removes the permanent before firing the event and doesn't capture self-abilities
	// (unlike Game.Destroy which uses checkAbilitiesForEvent). Engine gap.
	Register("Terrarion", func() Card {
		return NewArtifact("Terrarion", "{1}",
			WithKeyword(EntersTapped),
			WithActivatedAbility(
				AddAnyMana(2, Colorless),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithCost(SacrificeSourceCost()),
			),
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(
				DrawCards(Fixed(1)), false,
			)),
		)
	})

	// Unstable Obelisk {3}
	// Artifact
	// {T}: Add {C}.
	// {7}, {T}, Sacrifice this artifact: Destroy target permanent.
	Register("Unstable Obelisk", func() Card {
		return NewArtifact("Unstable Obelisk", "{3}",
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 1}),
			WithActivatedAbility(
				DestroyTargetPermanent(),
				GenericCost(7),
				WithCost(TapSourceCost()),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetPermanent()),
			),
		)
	})

	// Warmonger's Chariot {2}
	// Artifact — Equipment
	// Equipped creature gets +2/+2.
	// As long as equipped creature has defender, it can attack as though it didn't have defender.
	// Equip {3} ({3}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Warmonger's Chariot", func() Card {
		return NewEquipment("Warmonger's Chariot", "{2}",
			WithStaticAbility(
				BoostAttached(2, 2, AttachEquipment),
				GrantAbilityToAttached(AttrCanAttack, AttachEquipment),
			),
			WithAbility(NewEquipAbility(ManaCostOf("{3}"))),
		)
	})

}
