package fourthedition

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
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
	Register("Brainwash", func() Card {
		return NewAura("Brainwash", "{W}",
			WithCastTarget(TargetCreature()),
			WithStaticAbility(AttachedCantAttackUnlessPays(GenericCost(3))),
		)
	})

	// Creature Bond {1}{U}
	// Enchantment — Aura
	// Enchant creature
	// When enchanted creature dies, Creature Bond deals damage equal to that creature's toughness to the creature's controller.
	Register("Creature Bond", func() Card {
		return NewAura("Creature Bond", "{1}{U}",
			WithAbility(
				NewTriggered(EvtZoneChange, false,
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
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
					EventSourceWasOfType{Type: TypeCreature},
					SourceIsAttachedToEventSource{},
				}}),
			),
		)
	})

	// Erosion {U}{U}{U}
	// Enchantment — Aura
	// Enchant land
	// At the beginning of the upkeep of enchanted land's controller, destroy that land unless that player pays {1} or 1 life.
	Register("Erosion", func() Card {
		return NewAura("Erosion", "{U}{U}{U}",
			WithCastTarget(TargetLand()),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				FuncEffect("destroy enchanted land unless its controller pays {1} or 1 life",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						source := g.FindPermanent(sourceID)
						if source == nil || !source.IsAttached() {
							return nil
						}
						land := g.FindPermanent(source.AttachedTo)
						if land == nil {
							return nil
						}
						player := g.GetPlayer(land.ControllerID())
						if player == nil {
							return nil
						}
						paid := false
						choices := []string{"don't pay"}
						if g.CanAfford(player.PlayerID(), ParseManaCost("{1}"), nil) {
							choices = append([]string{"pay {1}"}, choices...)
						}
						if player.Life() >= 1 {
							choices = append(choices[:len(choices)-1], "pay 1 life", "don't pay")
						}
						switch choices[player.ChooseMode(choices, "Erosion")] {
						case "pay {1}":
							paid = g.TryPayMana(player.PlayerID(), "{1}")
						case "pay 1 life":
							if player.Life() >= 1 {
								g.PlayerLoseLife(player, 1)
								paid = true
							}
						}
						if !paid {
							g.DestroyPermanent(land)
						}
						return nil
					}), false),
			),
		)
	})

	// Flood {U}
	// Enchantment
	// {U}{U}: Tap target creature without flying.
	Register("Flood", func() Card {
		return NewEnchantment("Flood", "{U}",
			WithActivatedAbility(
				Tap(),
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
	Register("Living Artifact", func() Card {
		return NewAura("Living Artifact", "{G}",
			WithCastTarget(TargetArtifact()),
			WithAbility(NewTriggered(EvtDamageDealt, false,
				AddCounters(Vitality, EventAmountValue()).Targeting(ToSource()),
			).SetCondition(func(evt *GameEvent, _ GameReader, _, controllerID uuid.UUID) bool {
				return evt.TargetID == controllerID && evt.Amount > 0
			})),
			WithAbility(BeginningOfUpkeepTrigger(FuncEffect(
				"remove a vitality counter from Living Artifact; if you do, gain 1 life",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					perm := g.MutablePermanent(sourceID)
					if perm == nil {
						return nil
					}
					if perm.RemoveCounter(Vitality, 1) {
						if p := g.GetPlayer(controller); p != nil {
							g.PlayerGainLife(p, 1)
						}
					}
					return nil
				},
			), true)),
		)
	})

	// Power Leak {1}{U}
	// Enchantment — Aura
	// Enchant enchantment
	// At the beginning of the upkeep of enchanted enchantment's controller, that player may pay any amount of mana. This Aura deals 2 damage to that player. Prevent X of that damage, where X is the amount of mana that player paid this way.
	Register("Power Leak", func() Card {
		return NewAura("Power Leak", "{1}{U}",
			WithCastTarget(TargetPermanent(IsEnchantment)),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				FuncEffect("enchanted enchantment's controller may pay mana to prevent that much of 2 damage",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(2)},
					func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						source := g.FindPermanent(sourceID)
						if source == nil || !source.IsAttached() {
							return nil
						}
						enchantment := g.FindPermanent(source.AttachedTo)
						if enchantment == nil {
							return nil
						}
						player := g.GetPlayer(enchantment.ControllerID())
						if player == nil {
							return nil
						}
						available := g.HypotheticalMana(player.PlayerID())
						paid := player.ChooseNumber(0, available, "Power Leak mana payment")
						if paid > 0 && !g.TryPayMana(player.PlayerID(), fmt.Sprintf("{%d}", paid)) {
							paid = 0
						}
						if paid > 0 {
							g.AddSourcePreventionShield(player.PlayerID(), sourceID, min(paid, 2))
						}
						g.DealDamageToPlayer(player, 2, sourceID)
						return nil
					}), false),
			),
		)
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
