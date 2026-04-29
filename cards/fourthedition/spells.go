package fourthedition

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

func init() {
	registerSpells()
}

func registerSpells() {

	// Ashes to Ashes {1}{B}{B}
	// Sorcery
	// Exile two target nonartifact creatures. Ashes to Ashes deals 5 damage to you.
	// TODO: implement — needs two-target spell support
	Register("Ashes to Ashes", func() Card {
		return NewSorcery("Ashes to Ashes", "{1}{B}{B}",
			NewSpellAbility(),
		)
	})

	// Fissure {3}{R}{R}
	// Instant
	// Destroy target creature or land. It can't be regenerated.
	Register("Fissure", func() Card {
		return NewInstant("Fissure", "{3}{R}{R}",
			NewTargetedSpell(
				TargetPermanent(Or(IsCreature, IsLand)),
				DestroyTargetNoRegen(),
			),
		)
	})

	// Inferno {5}{R}{R}
	// Instant
	// Inferno deals 6 damage to each creature and each player.
	Register("Inferno", func() Card {
		return NewInstant("Inferno", "{5}{R}{R}",
			NewSpellAbility(CompositeEffects(
				"deal 6 damage to each creature and each player",
				DealDamageToAllCreatures(Fixed(6), PermanentFilter{}),
				DealDamageToPlayers(Fixed(6), SelectEachPlayer()),
			)),
		)
	})

	// Mana Clash {R}
	// Sorcery
	// You and target opponent each flip a coin. Mana Clash deals 1 damage to each player whose coin comes up tails. Repeat this process until both players' coins come up heads on the same flip.
	Register("Mana Clash", func() Card {
		return NewSorcery("Mana Clash", "{R}",
			NewTargetedSpell(
				TargetOpponent(),
				FuncEffect(
					"you and target opponent flip coins; tails takes 1 damage; repeat until both heads",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						you := g.GetPlayer(controller)
						opp := g.GetPlayer(targets[0])
						if you == nil || opp == nil {
							return nil
						}
						for {
							youHeads := g.FlipCoin(controller)
							oppHeads := g.FlipCoin(targets[0])
							if !youHeads {
								g.DealDamageToPlayer(you, 1, sourceID)
							}
							if !oppHeads {
								g.DealDamageToPlayer(opp, 1, sourceID)
							}
							if youHeads && oppHeads {
								return nil
							}
						}
					},
				),
			),
		)
	})

	// Marsh Gas {B}
	// Instant
	// All creatures get -2/-0 until end of turn.
	Register("Marsh Gas", func() Card {
		return NewInstant("Marsh Gas", "{B}",
			NewSpellAbility(
				Boost(Fixed(-2), Fixed(0)).Targeting(ToAllMatching(IsCreature)),
			),
		)
	})

	// Mind Bomb {U}
	// Sorcery
	// Each player may discard up to three cards. Mind Bomb deals damage to each player equal to 3 minus the number of cards they discarded this way.
	// TODO: implement — needs player discard choice
	Register("Mind Bomb", func() Card {
		return NewSorcery("Mind Bomb", "{U}",
			NewSpellAbility(),
		)
	})

	// Morale {1}{W}{W}
	// Instant
	// Attacking creatures get +1/+1 until end of turn.
	Register("Morale", func() Card {
		return NewInstant("Morale", "{1}{W}{W}",
			NewSpellAbility(
				Boost(Fixed(1), Fixed(1)).Targeting(ToAllMatching(IsAttacking)),
			),
		)
	})

	// Volcanic Eruption {X}{U}{U}{U}
	// Sorcery
	// Destroy X target Mountains. Volcanic Eruption deals damage to each creature and each player equal to the number of Mountains put into a graveyard this way.
	// TODO: implement — needs X-target destruction + conditional damage
	Register("Volcanic Eruption", func() Card {
		return NewSorcery("Volcanic Eruption", "{X}{U}{U}{U}",
			NewSpellAbility(),
		)
	})

	// Word of Binding {X}{B}{B}
	// Sorcery
	// Tap X target creatures.
	// TODO: implement — needs X-target support
	Register("Word of Binding", func() Card {
		return NewSorcery("Word of Binding", "{X}{B}{B}",
			NewSpellAbility(),
		)
	})
}
