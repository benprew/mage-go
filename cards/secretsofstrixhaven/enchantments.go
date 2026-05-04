package secretsofstrixhaven

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {

	// Additive Evolution {3}{G}{G}
	// Enchantment
	// When this enchantment enters, create a 0/0 green and blue Fractal creature token. Put three +1/+1 counters on it.
	// At the beginning of combat on your turn, put a +1/+1 counter on target creature you control. It gains vigilance until end of turn.
	Register("Additive Evolution", func() Card {
		return NewEnchantment("Additive Evolution", "{3}{G}{G}",
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("create 0/0 green and blue Fractal token, put three +1/+1 counters on it",
					EffectProperties{Outcome: OutcomeBenefit, TokenPower: 0, TokenToughness: 0},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						token := NewToken("Fractal Token", 0, 0,
							[]CardType{TypeCreature}, []string{"Fractal"})
						token.SetOwner(controller)
						token.SetColorOverride([]Color{Green, Blue})
						perm := g.PutOnBattlefield(token, controller)
						if perm == nil {
							return nil
						}
						g.AddCountersWithReplacement(perm, P1P1, 3, sourceID, false)
						return nil
					},
				),
				false,
			)),
			WithAbility(NewTriggered(EvtBeginCombat, false,
				CompositeEffects("put a +1/+1 counter on target creature you control; it gains vigilance until end of turn",
					AddCounters(P1P1, Fixed(1)),
					GrantKeyword(Vigilance),
				),
			).SetConditionData(EventPlayerIsController{}).AddTarget(TargetCreatureYouControl())),
		)
	})

	// Comforting Counsel {1}{G}
	// Enchantment
	// Whenever you gain life, put a growth counter on this enchantment.
	// As long as there are five or more growth counters on this enchantment, creatures you control get +3/+3.
	// XXX: "growth" is not a defined CounterType in core/counter.go; engine needs a Growth counter type to implement this card.
	Register("Comforting Counsel", func() Card {
		return NewEnchantment("Comforting Counsel", "{1}{G}")
	})

	// Graduation Day {W}
	// Enchantment
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, put a +1/+1 counter on target creature you control.
	// TODO: implement
	Register("Graduation Day", func() Card {
		return NewEnchantment("Graduation Day", "{W}")
	})

	// Living History {1}{R}
	// Enchantment
	// When this enchantment enters, create a 2/2 red and white Spirit creature token.
	// Whenever you attack, if a card left your graveyard this turn, target attacking creature gets +2/+0 until end of turn.
	// XXX: "if a card left your graveyard this turn" — engine per_turn_trackers.go lacks tracking of cards exiting the graveyard; second ability not implemented.
	Register("Living History", func() Card {
		return NewEnchantment("Living History", "{1}{R}",
			WithAbility(EntersBattlefieldTrigger(
				CreateColoredToken("Spirit Token", 2, 2,
					[]Color{Red, White},
					[]CardType{TypeCreature},
					[]string{"Spirit"},
				),
				false,
			)),
		)
	})

	// Primary Research {4}{W}
	// Enchantment
	// When this enchantment enters, return target nonland permanent card with mana value 3 or less from your graveyard to the battlefield.
	// At the beginning of your end step, if a card left your graveyard this turn, draw a card.
	// XXX: "if a card left your graveyard this turn" — engine per_turn_trackers.go lacks tracking of cards exiting the graveyard; second ability not implemented.
	Register("Primary Research", func() Card {
		nonlandMV3OrLess := NewCardFilter("nonland permanent card with mana value 3 or less", func(c Card) bool {
			if c.HasType(TypeLand) {
				return false
			}
			return c.ManaCost().CMC() <= 3
		})
		return NewEnchantment("Primary Research", "{4}{W}",
			WithAbility(
				EntersBattlefieldTrigger(
					ReturnFromGraveyardToBattlefield(),
					false,
				).AddTarget(TargetCardInYourGraveyard(nonlandMV3OrLess)),
			),
		)
	})
}
