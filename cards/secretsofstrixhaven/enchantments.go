package secretsofstrixhaven

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
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
	Register("Comforting Counsel", func() Card {
		player := dsl.BindPlayer("gainer")
		return NewEnchantment("Comforting Counsel", "{1}{G}",
			WithAbility(dsl.WhenEvent(dsl.ControllerGainsLife(player), dsl.AddCounters(dsl.This, Growth, dsl.Fixed(1)))),
			WithAbility(dsl.Static(dsl.While(dsl.SourceHasCounters(Growth, 5),
				dsl.ModifyEachPT(dsl.ControlledPermanents(IsCreature), dsl.Fixed(3), dsl.Fixed(3)),
			))),
		)
	})

	// Graduation Day {W}
	// Enchantment
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, put a +1/+1 counter on target creature you control.
	Register("Graduation Day", func() Card {
		return NewEnchantment("Graduation Day", "{W}",
			WithAbility(
				WheneverYouCastInstantOrSorceryTargetingCreatureTrigger(
					AddCounters(P1P1, Fixed(1)),
					false,
				).AddTarget(TargetCreatureYouControl()),
			),
		)
	})

	// Living History {1}{R}
	// Enchantment
	// When this enchantment enters, create a 2/2 red and white Spirit creature token.
	// Whenever you attack, if a card left your graveyard this turn, target attacking creature gets +2/+0 until end of turn.
	Register("Living History", func() Card {
		return NewEnchantment("Living History", "{1}{R}",
			WithAbility(
				dsl.WhenEvent(
					dsl.Enters(dsl.This),
					dsl.CreateTokenCopies(
						dsl.You,
						dsl.Fixed(1),
						dsl.DefineToken(
							"Spirit Token",
							dsl.TokenPT(2, 2),
							dsl.TokenColors(Red, White),
							dsl.TokenTypes(TypeCreature),
							dsl.TokenSubtypes("Spirit"),
						),
					),
				),
			),
			WithAbility(WheneverOneOrMoreCreaturesYouControlAttackTrigger(
				Boost(Fixed(2), Fixed(0)).Targeting(ToTarget()).Until(EndOfTurn),
				false,
			).SetCondition(func(evt *GameEvent, g GameReader, sourceID, controller uuid.UUID) bool {
				if evt.PlayerID != controller {
					return false
				}
				if gg, ok := g.(*Game); ok {
					return gg.PlayerHadCardLeaveGraveyardThisTurn(controller)
				}
				return false
			}).AddTarget(TargetCreature(IsAttacking))),
		)
	})

	// Primary Research {4}{W}
	// Enchantment
	// When this enchantment enters, return target nonland permanent card with mana value 3 or less from your graveyard to the battlefield.
	// At the beginning of your end step, if a card left your graveyard this turn, draw a card.
	Register("Primary Research", func() Card {
		nonlandMV3OrLess := NewCardFilter("nonland permanent card with mana value 3 or less", func(c Card) bool {
			if c.HasType(TypeLand) {
				return false
			}
			return c.ManaCost().CMC() <= 3
		})
		target := dsl.TargetGraveyardCard("target", dsl.GraveyardCardFilter(nonlandMV3OrLess))
		return NewEnchantment("Primary Research", "{4}{W}",
			WithAbility(
				dsl.WhenEvent(dsl.Enters(dsl.This), dsl.ReturnCardToBattlefield(target, dsl.You), dsl.WithTargets(target)),
			),
			WithAbility(NewTriggered(EvtEndStep, false,
				FuncEffect("if a card left your graveyard this turn, draw a card",
					EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if !g.PlayerHadCardLeaveGraveyardThisTurn(controller) {
							return nil
						}
						if p := g.GetPlayer(controller); p != nil {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				),
			).SetConditionData(EventPlayerIsController{})),
		)
	})
}
