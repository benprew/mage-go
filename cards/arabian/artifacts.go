package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

// discardLastDrawnCost is an additional cost: discard the last card drawn this turn.
type discardLastDrawnCost struct{}

func (c *discardLastDrawnCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	if p == nil {
		return false
	}
	lastID := p.LastDrawnCardID()
	if lastID == uuid.Nil {
		return false
	}
	for _, card := range p.Hand() {
		if card.ID() == lastID {
			return true
		}
	}
	return false
}

func (c *discardLastDrawnCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	card, ok := p.RemoveFromHand(p.LastDrawnCardID())
	if !ok {
		return nil
	}
	p.AddToGraveyard(card)
	return nil
}

func (c *discardLastDrawnCost) Text() string { return "Discard the last card you drew this turn" }

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// Oracle: "{X}, {T}: The next time you would draw a card this turn, instead look at
	// the top X cards of your library, put all but one of them on the bottom of your
	// library in a random order, then draw a card. X can't be 0."
	Register("Aladdin's Lamp", withExpansion(func() Card {
		return NewArtifact("Aladdin's Lamp", "{10}",
			WithActivatedAbility(
				FuncEffect("set up draw replacement",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						x := g.XValue()
						if x <= 0 {
							return nil // X can't be 0
						}
						g.SetDrawReplacement(controller, x)
						return nil
					}),
				TapSourceCost(),
				WithCost(ManaCostOf("{X}")),
			),
		)
	}))

	// Oracle: "{8}, {T}: Aladdin's Ring deals 4 damage to any target."
	Register("Aladdin's Ring", withExpansion(func() Card {
		return NewArtifact("Aladdin's Ring", "{8}",
			WithActivatedAbility(
				DealDamage(Fixed(4)),
				TapSourceCost(),
				WithCost(ManaCostOf("{8}")),
				WithTarget(TargetAnyTarget()),
			),
		)
	}))

	// Oracle: "{1}, Sacrifice Bottle of Suleiman: Flip a coin. If you win the flip,
	// create a 5/5 colorless Djinn artifact creature token with flying. If you lose
	// the flip, Bottle of Suleiman deals 5 damage to you."
	Register("Bottle of Suleiman", withExpansion(func() Card {
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
	}))

	// Oracle: "Whenever one or more other nontoken permanents with a name originally
	// printed in the Arabian Nights expansion are on the battlefield, their controllers
	// sacrifice them. Players can't cast spells or play lands with a name originally
	// printed in the Arabian Nights expansion."
	Register("City in a Bottle", withExpansion(func() Card {
		return NewArtifact("City in a Bottle", "{2}",
			// ETB: sacrifice all other nontoken Arabian Nights permanents
			WithAbility(
				EntersBattlefieldTrigger(
					FuncEffect("sacrifice all other Arabian Nights nontoken permanents",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							for _, p := range g.FilterBattlefield(HasExpansion(expansionName)) {
								if p.ID() == sourceID {
									continue
								}
								if p.Card.(*BaseCard).IsToken() {
									continue
								}
								g.Sacrifice(p)
							}
							return nil
						}),
					false,
				),
			),
			// Whenever any nontoken Arabian Nights permanent enters (not self), sacrifice it
			WithAbility(
				NewTriggered(EvtEntersBattlefield, false,
					FuncEffect("sacrifice entering Arabian Nights permanent",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							for _, p := range g.FilterBattlefield(HasExpansion(expansionName)) {
								if p.ID() == sourceID {
									continue
								}
								if p.Card.(*BaseCard).IsToken() {
									continue
								}
								g.Sacrifice(p)
							}
							return nil
						}),
				).SetCondition(func(evt *GameEvent, g *Game, sourceID, _ uuid.UUID) bool {
					if evt.SourceID == sourceID {
						return false // don't trigger on self entering
					}
					perm := g.FindPermanent(evt.SourceID)
					if perm == nil {
						return false
					}
					return perm.Card.Expansion() == expansionName && !perm.Card.(*BaseCard).IsToken()
				}),
			),
		)
	}))

	// Oracle: "{2}, {T}: Untap target attacking creature you control. Prevent all combat
	// damage that would be dealt to and dealt by that creature this turn."
	// Combat damage prevention achieved via RemoveFromCombat — once removed, no damage
	// is assigned to or by the creature.
	Register("Ebony Horse", withExpansion(func() Card {
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
	}))

	// Oracle: "{2}, {T}: Target creature gains flying until end of turn."
	Register("Flying Carpet", withExpansion(func() Card {
		return NewArtifact("Flying Carpet", "{4}",
			WithActivatedAbility(
				GrantKeywordUntilEndOfTurn(Flying, SelectTarget),
				TapSourceCost(),
				WithCost(ManaCostOf("{2}")),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Oracle: "{2}, {T}, Discard the last card you drew this turn: Draw a card."
	Register("Jandor's Ring", withExpansion(func() Card {
		return NewArtifact("Jandor's Ring", "{6}",
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				TapSourceCost(),
				WithCost(ManaCostOf("{2}")),
				WithCost(&discardLastDrawnCost{}),
			),
		)
	}))

	// Oracle: "{3}, {T}: Untap target creature."
	Register("Jandor's Saddlebags", withExpansion(func() Card {
		return NewArtifact("Jandor's Saddlebags", "{2}",
			WithActivatedAbility(
				UntapTarget(),
				TapSourceCost(),
				WithCost(ManaCostOf("{3}")),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Oracle: "Remove Jeweled Bird from your deck before playing if you're not playing
	// for ante. {T}: Ante Jeweled Bird. If you do, put all other cards you own from
	// the ante zone into your graveyard, then draw a card."
	Register("Jeweled Bird", withExpansion(func() Card {
		return NewArtifact("Jeweled Bird", "{1}",
			WithActivatedAbility(
				FuncEffect("ante Jeweled Bird, return other ante to graveyard, draw",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						card := perm.Card
						g.RemoveFromBattlefield(perm)
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Put Jeweled Bird into ante
						p.AddToAnte(card)
						// Move all other cards from ante to graveyard
						var toReturn []Card
						for _, c := range p.Ante() {
							if c.ID() != card.ID() {
								toReturn = append(toReturn, c)
							}
						}
						for _, c := range toReturn {
							if removed, ok := p.RemoveFromAnte(c.ID()); ok {
								p.AddToGraveyard(removed)
							}
						}
						// Draw a card
						p.DrawCard()
						return nil
					}),
				TapSourceCost(),
			),
		)
	}))

	// Oracle: "{2}: Choose one — Destroy target Aura attached to a land. / The next time
	// target land would be destroyed this turn, remove all damage marked on it instead."
	Register("Pyramids", withExpansion(func() Card {
		c := NewArtifact("Pyramids", "{6}",
			WithActivatedAbility(
				FuncEffect("destroy aura on land or protect land from destruction",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := g.FindPermanent(targets[0])
						if target == nil {
							return nil
						}
						if g.ModeValue() == 0 {
							// Mode 1: Destroy target Aura attached to a land
							g.DestroyPermanent(target)
						} else {
							// Mode 2: Make target land indestructible until end of turn
							GrantKeywordUntilEndOfTurn(Indestructible, SelectTarget).
								Apply(g, sourceID, controller, targets)
						}
						return nil
					}),
				ManaCostOf("{2}"),
				WithTarget(TargetPermanent(Or(IsAuraOnLand, IsLand))),
			),
		)
		c.SetModes([]string{
			"Destroy target Aura attached to a land",
			"Target land becomes indestructible this turn",
		})
		return c
	}))

	// Oracle: "{5}, {T}, Exile Ring of Ma'rûf: The next time you would draw a card this
	// turn, instead put a card you own from outside the game into your hand."
	// XXX: Ring of Ma'rûf skipped — wish/sideboard mechanic
	Register("Ring of Ma'rûf", withExpansion(func() Card {
		return NewArtifact("Ring of Ma'rûf", "{5}")
	}))

	// Oracle: "{2}, {T}: Target creature gains islandwalk until end of turn. When that
	// creature dies this turn, destroy Sandals of Abdallah."
	Register("Sandals of Abdallah", withExpansion(func() Card {
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
	}))
}
