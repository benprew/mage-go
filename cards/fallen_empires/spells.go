package fallen_empires

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerSpells()
}

func registerSpells() {

	// Dwarven Catapult {X}{R}
	// Instant
	// Dwarven Catapult deals X damage divided evenly, rounded down, among all creatures target opponent controls.
	// TODO: implement
	Register("Dwarven Catapult", withExpansion(func() Card {
		return NewInstant("Dwarven Catapult", "{X}{R}",
			NewSpellAbility(),
		)
	}))

	// Goblin Grenade {R}
	// Sorcery
	// As an additional cost to cast this spell, sacrifice a Goblin.
	// Goblin Grenade deals 5 damage to any target.
	// TODO: implement
	Register("Goblin Grenade", withExpansion(func() Card {
		return NewSorcery("Goblin Grenade", "{R}",
			NewSpellAbility(),
		)
	}))

	// High Tide {U}
	// Instant
	// Until end of turn, whenever a player taps an Island for mana, that player adds an additional {U}.
	// TODO: implement
	Register("High Tide", withExpansion(func() Card {
		return NewInstant("High Tide", "{U}",
			NewSpellAbility(),
		)
	}))

	// Hymn to Tourach {B}{B}
	// Sorcery
	// Target player discards two cards at random.
	Register("Hymn to Tourach", withExpansion(func() Card {
		return NewSorcery("Hymn to Tourach", "{B}{B}",
			NewTargetedSpell(TargetPlayer(), DiscardRandom(2)),
		)
	}))

	// Icatian Town {5}{W}
	// Sorcery
	// Create four 1/1 white Citizen creature tokens.
	Register("Icatian Town", withExpansion(func() Card {
		return NewSorcery("Icatian Town", "{5}{W}",
			NewSpellAbility(FuncEffect("create four 1/1 white Citizen creature tokens",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					for i := 0; i < 4; i++ {
						token := NewToken("Citizen", 1, 1, []CardType{TypeCreature}, []string{"Citizen"})
						token.SetOwner(controller)
						g.PutOnBattlefield(token, controller)
					}
					return nil
				})),
		)
	}))

	// Soul Exchange {B}{B}
	// Sorcery
	// As an additional cost to cast this spell, exile a creature you control.
	// Return target creature card from your graveyard to the battlefield. Put a +2/+2 counter on that creature if the exiled creature was a Thrull.
	// TODO: implement
	Register("Soul Exchange", withExpansion(func() Card {
		return NewSorcery("Soul Exchange", "{B}{B}",
			NewSpellAbility(),
		)
	}))

	// Spore Cloud {1}{G}{G}
	// Instant
	// Tap all blocking creatures. Prevent all combat damage that would be dealt this turn. Each attacking creature and each blocking creature doesn't untap during its controller's next untap step.
	// TODO: implement
	Register("Spore Cloud", withExpansion(func() Card {
		return NewInstant("Spore Cloud", "{1}{G}{G}",
			NewSpellAbility(),
		)
	}))
}
