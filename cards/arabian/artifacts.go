package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// Oracle: "{X}, {T}: The next time you would draw a card this turn, instead look at
	// the top X cards of your library, put all but one of them on the bottom of your
	// library in a random order, then draw a card. X can't be 0."
	Register("Aladdin's Lamp", func() Card {
		return NewArtifact("Aladdin's Lamp", "{10}")
	})

	// Oracle: "{8}, {T}: Aladdin's Ring deals 4 damage to any target."
	Register("Aladdin's Ring", func() Card {
		return NewArtifact("Aladdin's Ring", "{8}",
			WithActivatedAbility(
				DealDamage(Fixed(4)),
				TapSourceCost(),
				WithCost(ManaCostOf("{8}")),
				WithTarget(TargetAnyTarget()),
			),
		)
	})

	// Oracle: "{1}, Sacrifice Bottle of Suleiman: Flip a coin. If you win the flip,
	// create a 5/5 colorless Djinn artifact creature token with flying. If you lose
	// the flip, Bottle of Suleiman deals 5 damage to you."
	Register("Bottle of Suleiman", func() Card {
		return NewArtifact("Bottle of Suleiman", "{4}",
			WithActivatedAbility(
				FuncEffect("flip coin: 5/5 Djinn or 5 damage",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if g.FlipCoin(controller) {
							// Win — create 5/5 Djinn artifact creature token with flying
							token := CreateToken("Djinn", 5, 5, []CardType{TypeArtifact, TypeCreature}, []string{"Djinn"}, Flying)
							return token.Apply(g, sourceID, controller, nil)
						}
						// Lose — take 5 damage
						p := g.GetPlayer(controller)
						if p != nil {
							g.DealDamageToPlayer(p, 5, sourceID)
						}
						return nil
					}),
				ManaCostOf("{1}"),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Oracle: "Whenever one or more other nontoken permanents with a name originally
	// printed in the Arabian Nights expansion are on the battlefield, their controllers
	// sacrifice them. Players can't cast spells or play lands with a name originally
	// printed in the Arabian Nights expansion."
	// SKIP: Requires set identity tracking.
	Register("City in a Bottle", func() Card {
		return NewArtifact("City in a Bottle", "{2}")
	})

	// Oracle: "{2}, {T}: Untap target attacking creature you control. Prevent all combat
	// damage that would be dealt to and dealt by that creature this turn."
	Register("Ebony Horse", func() Card {
		return NewArtifact("Ebony Horse", "{3}",
			WithActivatedAbility(
				FuncEffect("untap and remove from combat",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						perm.Tapped = false
						g.RemoveFromCombat(perm.ID())
						return nil
					}),
				TapSourceCost(),
				WithCost(ManaCostOf("{2}")),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	})

	// Oracle: "{2}, {T}: Target creature gains flying until end of turn."
	Register("Flying Carpet", func() Card {
		return NewArtifact("Flying Carpet", "{4}",
			WithActivatedAbility(
				GrantKeywordUntilEndOfTurn(Flying, SelectTarget),
				TapSourceCost(),
				WithCost(ManaCostOf("{2}")),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Oracle: "{2}, {T}, Discard the last card you drew this turn: Draw a card."
	Register("Jandor's Ring", func() Card {
		return NewArtifact("Jandor's Ring", "{6}")
	})

	// Oracle: "{3}, {T}: Untap target creature."
	Register("Jandor's Saddlebags", func() Card {
		return NewArtifact("Jandor's Saddlebags", "{2}",
			WithActivatedAbility(
				UntapTarget(),
				TapSourceCost(),
				WithCost(ManaCostOf("{3}")),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Oracle: "Remove Jeweled Bird from your deck before playing if you're not playing
	// for ante. {T}: Ante Jeweled Bird."
	// SKIP: Ante mechanic.
	Register("Jeweled Bird", func() Card {
		return NewArtifact("Jeweled Bird", "{1}")
	})

	// Oracle: "{2}: Choose one — Destroy target Aura attached to a land. / The next time
	// target land would be destroyed this turn, remove all damage marked on it instead."
	Register("Pyramids", func() Card {
		return NewArtifact("Pyramids", "{6}")
	})

	// Oracle: "{5}, {T}, Exile Ring of Ma'rûf: The next time you would draw a card this
	// turn, instead put a card you own from outside the game into your hand."
	// SKIP: Wish/sideboard mechanic.
	Register("Ring of Ma'rûf", func() Card {
		return NewArtifact("Ring of Ma'rûf", "{5}")
	})

	// Oracle: "{2}, {T}: Target creature gains islandwalk until end of turn. When that
	// creature dies this turn, destroy Sandals of Abdallah."
	Register("Sandals of Abdallah", func() Card {
		return NewArtifact("Sandals of Abdallah", "{4}",
			WithActivatedAbility(
				FuncEffect("grant islandwalk, destroy self if creature dies",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						// Grant islandwalk
						GrantKeywordUntilEndOfTurn(Islandwalk, SelectTarget).Apply(g, sourceID, controller, targets)
						// Register delayed trigger: if that creature dies, destroy Sandals
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:    EvtCreatureDied,
							MatchEventID: targets[0],
							TargetID:     sourceID,
							Effects:      []Effect{DestroyTarget()},
							SourceID:     sourceID,
							Controller:   controller,
						})
						return nil
					}),
				TapSourceCost(),
				WithCost(ManaCostOf("{2}")),
				WithTarget(TargetCreature()),
			),
		)
	})
}
