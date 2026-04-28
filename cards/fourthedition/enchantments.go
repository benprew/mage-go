package fourthedition

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {

	// Animate Artifact {3}{U}
	// Enchantment — Aura
	// Enchant artifact
	// As long as enchanted artifact isn't a creature, it's an artifact creature with power and
	// toughness each equal to its mana value.
	Register("Animate Artifact", func() Card {
		return NewAura("Animate Artifact", "{3}{U}",
			WithCastTarget(TargetArtifact()),
			WithStaticAbility(AnimateArtifact(Attached)...),
		)
	})

	// Brainwash {W}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature can't attack unless its controller pays {3}.
	// TODO: implement — needs attack cost mechanism
	Register("Brainwash", func() Card {
		return NewAura("Brainwash", "{W}")
	})

	// Creature Bond {1}{U}
	// Enchantment — Aura
	// Enchant creature
	// When enchanted creature dies, Creature Bond deals damage equal to that creature's toughness to the creature's controller.
	Register("Creature Bond", func() Card {
		return NewAura("Creature Bond", "{1}{U}",
			WithAbility(
				NewTriggered(EvtCreatureDied, false,
					FuncEffect(
						"deal damage equal to creature's toughness to its controller",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							card := g.FindCardAnywhere(targets[0])
							if card == nil {
								return nil
							}
							toughness := card.Toughness()
							owner := card.Owner()
							if p := g.GetPlayer(owner); p != nil && toughness > 0 {
								g.DealDamageToPlayer(p, toughness, sourceID)
							}
							return nil
						},
					),
				).SetConditionData(SourceIsAttachedToEventSource{}),
			),
		)
	})

	// Erosion {U}{U}{U}
	// Enchantment — Aura
	// Enchant land
	// At the beginning of the upkeep of enchanted land's controller, destroy that land unless that player pays {1} or 1 life.
	// TODO: implement — needs choice between mana and life payment
	Register("Erosion", func() Card {
		return NewAura("Erosion", "{U}{U}{U}")
	})

	// Flood {U}
	// Enchantment
	// {U}{U}: Tap target creature without flying.
	Register("Flood", func() Card {
		return NewEnchantment("Flood", "{U}",
			WithActivatedAbility(
				TapTarget(),
				ManaCostOf("{U}{U}"),
				WithTarget(TargetCreature(NotHasKeywordFilter(Flying))),
			),
		)
	})

	// Living Artifact {G}
	// Enchantment — Aura
	// Enchant artifact
	// Whenever you're dealt damage, put that many vitality counters on Living Artifact.
	// At the beginning of your upkeep, you may remove a vitality counter from Living Artifact. If you do, you gain 1 life.
	// TODO: implement — needs vitality counters + damage-to-you trigger
	Register("Living Artifact", func() Card {
		return NewAura("Living Artifact", "{G}")
	})

	// Power Leak {1}{U}
	// Enchantment — Aura
	// Enchant enchantment
	// At the beginning of the upkeep of enchanted enchantment's controller, that player may pay any amount of mana. Power Leak deals 2 damage to that player. Prevent X of that damage, where X is the amount of mana that player paid this way.
	// TODO: implement — needs mana payment choice
	Register("Power Leak", func() Card {
		return NewAura("Power Leak", "{1}{U}")
	})

	// Sunken City {U}{U}
	// Enchantment
	// At the beginning of your upkeep, sacrifice Sunken City unless you pay {U}{U}.
	// Blue creatures get +1/+1.
	Register("Sunken City", func() Card {
		return NewEnchantment("Sunken City", "{U}{U}",
			WithAbility(SacrificeAtUpkeepUnlessPay("{U}{U}")),
			WithStaticAbility(
				BoostAllCreaturesIncludingSelf(1, 1, HasColorFilter(Blue)),
			),
		)
	})

	// Venom {1}{G}{G}
	// Enchantment — Aura
	// Enchant creature
	// Whenever enchanted creature blocks or becomes blocked by a non-Wall creature, destroy the other creature at end of combat.
	Register("Venom", func() Card {
		return NewAura("Venom", "{1}{G}{G}",
			WithAbility(StaticAbility(
				GrantAbilityToAttached(BasiliskTouch, AttachAura),
			)),
		)
	})
}
