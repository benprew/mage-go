package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	// Oracle: "At the beginning of your upkeep, put a wind counter on Cyclone, then
	// sacrifice Cyclone unless you pay {G} for each wind counter on it. If you pay,
	// Cyclone deals damage equal to the number of wind counters on it to each creature
	// and each player."
	Register("Cyclone", func() Card {
		return NewEnchantment("Cyclone", "{2}{G}{G}")
	})

	// Oracle: "At the beginning of your upkeep, destroy the creature with the least power.
	// It can't be regenerated. If two or more creatures are tied for least power, you choose
	// one of them. When there are no creatures on the battlefield, sacrifice Drop of Honey."
	Register("Drop of Honey", func() Card {
		return NewEnchantment("Drop of Honey", "{G}",
			WithAbility(
				BeginningOfUpkeepTrigger(
					FuncEffect("destroy least power creature or sacrifice self",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							creatures := g.FilterBattlefield(IsCreature)
							if len(creatures) == 0 {
								src := g.FindPermanent(sourceID)
								if src != nil {
									g.Sacrifice(src)
								}
								return nil
							}
							// Find minimum power
							minPower := creatures[0].CurrentPower(g)
							for _, c := range creatures[1:] {
								pw := c.CurrentPower(g)
								if pw < minPower {
									minPower = pw
								}
							}
							// Collect tied creatures
							var tied []*Permanent
							for _, c := range creatures {
								if c.CurrentPower(g) == minPower {
									tied = append(tied, c)
								}
							}
							var target *Permanent
							if len(tied) == 1 {
								target = tied[0]
							} else {
								p := g.GetPlayer(controller)
								if p != nil {
									target = p.ChoosePermanent(tied, "destroy", g)
								}
							}
							if target != nil {
								// Can't be regenerated
								target.GrantBaseAttr(CantRegenerate)
								g.DestroyPermanent(target)
							}
							return nil
						}), false,
				),
			),
		)
	})

	// Oracle: "As Jihad enters, choose a color and an opponent. White creatures get +2/+1
	// as long as the chosen player controls a nontoken permanent of the chosen color.
	// When the chosen player controls no nontoken permanents of the chosen color,
	// sacrifice Jihad."
	Register("Jihad", func() Card {
		return NewEnchantment("Jihad", "{W}{W}{W}")
	})

	// Oracle: "When Oubliette enters, target creature phases out until Oubliette leaves
	// the battlefield. Tap that creature as it phases in this way."
	Register("Oubliette", func() Card {
		return NewEnchantment("Oubliette", "{1}{B}{B}")
	})

	// ===== AURAS =====

	// Oracle: "Enchant creature. Enchanted creature has islandwalk."
	Register("Fishliver Oil", func() Card {
		return NewAura("Fishliver Oil", "{1}{U}",
			WithAbility(StaticAbility(GrantAbilityToAttached(Islandwalk, AttachAura))),
		)
	})

	// Oracle: "Enchant creature. Enchanted creature gets +3/+3. At the beginning of the
	// upkeep of enchanted creature's controller, put a -1/-1 counter on that creature."
	Register("Unstable Mutation", func() Card {
		return NewAura("Unstable Mutation", "{U}",
			WithAbility(StaticAbility(BoostAttached(3, 3, AttachAura))),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				FuncEffect("put a -1/-1 counter on enchanted creature",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						target := g.FindPermanent(src.AttachedTo)
						if target != nil {
							target.AddCounter(M1M1, 1)
						}
						return nil
					}), false,
			)),
		)
	})
}
