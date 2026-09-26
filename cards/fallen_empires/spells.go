package fallen_empires

import (
	. "github.com/benprew/mage-go/pkg/mage/dsl"
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
	Register("Goblin Grenade", withExpansion(func() Card {
		return NewSorcery("Goblin Grenade", "{R}",
			NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(5))),
			WithAdditionalCost(SacrificeMatchingCost(HasSubType("Goblin"), "Sacrifice a Goblin")),
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
		player := TargetPlayer("player")
		return NewSorcery("Hymn to Tourach", "{B}{B}",
			Spell(DiscardAtRandom(player, Fixed(2)), WithTargets(player)),
		)
	}))

	// Icatian Town {5}{W}
	// Sorcery
	// Create four 1/1 white Citizen creature tokens.
	Register("Icatian Town", withExpansion(func() Card {
		return NewSorcery("Icatian Town", "{5}{W}",
			Spell(CreateTokenCopies(You, Fixed(4), DefineToken("Citizen",
				TokenTypes(TypeCreature), TokenSubtypes("Citizen"), TokenColors(White), TokenPT(1, 1)))),
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
	Register("Spore Cloud", withExpansion(func() Card {
		creature := BindPermanent("combat creature")
		blocker := BindPermanent("blocker")
		return NewInstant("Spore Cloud", "{1}{G}{G}",
			Spell(Sequence(
				ForEach(MatchingPermanents(And(IsCreature, IsBlocking)), blocker, Tap(From(blocker))),
				PreventAllCombatDamage(),
				ForEach(MatchingPermanents(And(IsCreature, Or(IsAttacking, IsBlocking))), creature, SkipNextUntap(From(creature))),
			)),
		)
	}))
}
