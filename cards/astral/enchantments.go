package astral

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {

	// Necropolis of Azar {2}{B}{B}
	// Enchantment
	// Whenever a non-black creature is put into any graveyard from play, put a husk counter on Necropolis of Azar.
	// {5}, Remove a husk counter from Necropolis of Azar: Put a Spawn of Azar token into play. Treat this token as a black creature with a random power and toughness, each no less than 1 and no greater than 3, that has swampwalk.
	Register("Necropolis of Azar", func() Card {
		return NewEnchantment("Necropolis of Azar", "{2}{B}{B}",
			WithAbility(
				NewTriggered(EvtZoneChange, false,
					AddCounters(Husk, Fixed(1)).Targeting(ToSource()),
				).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard},
					EventSourceWasOfType{Type: TypeCreature},
					EventSourceWasNotColor{Color: Black},
				}}),
			),
			WithActivatedAbility(
				FuncEffect("create a black Spawn of Azar token with random power and toughness", EffectProperties{},
					func(g *Game, _ uuid.UUID, controllerID uuid.UUID, _ []uuid.UUID) error {
						power := g.RandIntn(3) + 1
						toughness := g.RandIntn(3) + 1
						token := NewToken("Spawn of Azar", power, toughness, []CardType{TypeCreature}, []string{"Spawn"}, Swampwalk)
						token.SetColorOverride([]Color{Black})
						token.SetOwner(controllerID)
						g.PutOnBattlefield(token, controllerID)
						return nil
					},
				),
				GenericCost(5),
				WithCost(RemoveCountersCost(Husk, 1)),
			),
		)
	})

	// Power Struggle {2}{U}{U}{U}
	// Enchantment
	// During each player's upkeep, that player exchanges control of random target artifact, creature or land he or she controls, for control of random target permanent of the same type that a random opponent controls.
	Register("Power Struggle", func() Card {
		return NewEnchantment("Power Struggle", "{2}{U}{U}{U}",
			WithAbility(
				BeginningOfEachUpkeepTrigger(
					ExchangeControlOfTargetsSharingPermanentType(), false,
				).AddTarget(TargetRandomActivePlayerExchangePair()),
			),
		)
	})

}
